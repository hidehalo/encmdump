package main

import (
	"errors"
	"os"
)

func readDir(path string, filter func(fileInfo os.FileInfo) bool) []os.FileInfo {
	stat, err := os.Stat(path)
	handleError(err)
	if stat.IsDir() != true {
		handleError(errors.New("错误的路径" + "\"" + path + "\""))
	}
	file, err := os.Open(path)
	defer file.Close()
	handleError(err)
	subFiles, err := file.Readdir(-1)
	handleError(err)
	ret := make([]os.FileInfo, 0)
	for _, subPath := range subFiles {
		if filter(subPath) {
			ret = append(ret, subPath)
		}
	}

	return ret
}
