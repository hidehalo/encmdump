package filesys

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/hidehalo/encmdump/utils"
)

func ReadDir(path string, filter func(fileInfo os.FileInfo) bool) []os.FileInfo {
	realPath, err := filepath.EvalSymlinks(path)
	stat, err := os.Stat(realPath)
	if stat.IsDir() != true {
		utils.HandleError(errors.New("错误的路径" + "\"" + path + "\""))
	}
	utils.HandleError(err)
	file, err := os.Open(realPath)
	defer file.Close()
	utils.HandleError(err)
	subFiles, err := file.Readdir(-1)
	utils.HandleError(err)
	ret := make([]os.FileInfo, 0)
	for _, subPath := range subFiles {
		if filter(subPath) {
			ret = append(ret, subPath)
		}
	}

	return ret
}
