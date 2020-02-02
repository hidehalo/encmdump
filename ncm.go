package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bogem/id3v2"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
	"github.com/yoki123/ncmdump"
)

func containPNGHeader(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	return string(data[:8]) == string([]byte{137, 80, 78, 71, 13, 10, 26, 10})
}

func fetchURL(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, err
	}
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {

		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		log.Printf("Failed to download album pic: remote returned %d\n", res.StatusCode)
		return nil, err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func addFLACTag(fileName string, imgData []byte, meta *ncmdump.Meta) {
	f, err := flac.ParseFile(fileName)
	if err != nil {
		consoleOut(err.Error())
		return
	}
	if imgData == nil && meta.Album.CoverUrl != "" {
		imgData, err = fetchURL(meta.Album.CoverUrl)
		handleError(err)
	}
	if imgData != nil {
		picMIME := "image/jpeg"
		if containPNGHeader(imgData) {
			picMIME = "image/png"
		}
		picture, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "Front cover", imgData, picMIME)
		if err == nil {
			picturemeta := picture.Marshal()
			f.Meta = append(f.Meta, &picturemeta)
		}
	} else if meta.Album.CoverUrl != "" {
		picture := &flacpicture.MetadataBlockPicture{
			PictureType: flacpicture.PictureTypeFrontCover,
			MIME:        "-->",
			Description: "Front cover",
			ImageData:   []byte(meta.Album.CoverUrl),
		}
		picturemeta := picture.Marshal()
		f.Meta = append(f.Meta, &picturemeta)
	}
	var cmtmeta *flac.MetaDataBlock
	for _, m := range f.Meta {
		if m.Type == flac.VorbisComment {
			cmtmeta = m
			break
		}
	}
	var cmts *flacvorbis.MetaDataBlockVorbisComment
	if cmtmeta != nil {
		cmts, err = flacvorbis.ParseFromMetaDataBlock(*cmtmeta)
		handleError(err)
	} else {
		cmts = flacvorbis.New()
	}
	titles, err := cmts.Get(flacvorbis.FIELD_TITLE)
	handleError(err)
	if len(titles) == 0 {
		if meta.Name != "" {
			cmts.Add(flacvorbis.FIELD_TITLE, meta.Name)
		}
	}
	albums, err := cmts.Get(flacvorbis.FIELD_ALBUM)
	handleError(err)
	if len(albums) == 0 {
		if meta.Album != nil && meta.Album.Name != "" {
			cmts.Add(flacvorbis.FIELD_ALBUM, meta.Album.Name)
		}
	}
	artists, err := cmts.Get(flacvorbis.FIELD_ARTIST)
	handleError(err)
	if len(artists) == 0 {
		for _, artist := range meta.Artists {
			cmts.Add(flacvorbis.FIELD_ARTIST, artist.Name)
		}
	}
	res := cmts.Marshal()
	if cmtmeta != nil {
		*cmtmeta = res
	} else {
		f.Meta = append(f.Meta, &res)
	}
	f.Save(fileName)
}

func addMP3Tag(fileName string, imgData []byte, meta *ncmdump.Meta) {
	tag, err := id3v2.Open(fileName, id3v2.Options{Parse: true})
	defer tag.Close()
	handleError(err)
	if imgData == nil && meta.Album.CoverUrl != "" {
		imgData, err = fetchURL(meta.Album.CoverUrl)
		handleError(err)
	}
	if imgData != nil {
		picMIME := "image/jpeg"
		if containPNGHeader(imgData) {
			picMIME = "image/png"
		}
		pic := id3v2.PictureFrame{
			Encoding:    id3v2.EncodingISO,
			MimeType:    picMIME,
			PictureType: id3v2.PTFrontCover,
			Description: "Front cover",
			Picture:     imgData,
		}
		tag.AddAttachedPicture(pic)
	} else if meta.Album.CoverUrl != "" {
		pic := id3v2.PictureFrame{
			Encoding:    id3v2.EncodingISO,
			MimeType:    "-->",
			PictureType: id3v2.PTFrontCover,
			Description: "Front cover",
			Picture:     []byte(meta.Album.CoverUrl),
		}
		tag.AddAttachedPicture(pic)
	}
	if tag.GetTextFrame("TIT2").Text == "" {
		if meta.Name != "" {
			tag.AddTextFrame("TIT2", id3v2.EncodingUTF8, meta.Name)
		}
	}
	if tag.GetTextFrame("TALB").Text == "" {
		if meta.Album != nil && meta.Album.Name != "" {
			tag.AddTextFrame("TALB", id3v2.EncodingUTF8, meta.Album.Name)
		}
	}
	if tag.GetTextFrame("TPE1").Text == "" {
		for _, artist := range meta.Artists {
			tag.AddTextFrame("TPE1", id3v2.EncodingUTF8, artist.Name)
		}
	}
	err = tag.Save()
	handleError(err)
}

func isNCM(name string) bool {
	return strings.Contains(name, ".ncm")
}

func dump(path string, overwrite bool) {
	fp, err := os.Open(path)
	defer fp.Close()
	handleError(err)
	meta, err := ncmdump.DumpMeta(fp)
	handleError(err)
	output := strings.Replace(path, ".ncm", "."+meta.Format, -1)
	if _, err = os.Stat(output); !os.IsNotExist(err) && !overwrite {
		// Ignore when NCM file already dumped or need not overwrite
		consoleOut("跳过已处理文件：" + path)
		return
	}
	consoleOut("正在处理文件：" + path + "...")
	data, err := ncmdump.Dump(fp)
	handleError(err)
	err = ioutil.WriteFile(output, data, 0644)
	handleError(err)
	cover, err := ncmdump.DumpCover(fp)
	handleError(err)
	switch meta.Format {
	case "mp3":
		addMP3Tag(output, cover, &meta)
	case "flac":
		addFLACTag(output, cover, &meta)
	}
	consoleOut(path + "处理成功！")
}
