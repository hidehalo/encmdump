package main

import (
	"flag"
	"os"
	"runtime"
	"sync"

	"github.com/hidehalo/encmdump/filesys"
	"github.com/hidehalo/encmdump/ncm"
)

// Overwrite flag
var _OW bool

func findDumpJobs(path string, jobsCh chan string, exit *sync.WaitGroup) {
	filter := func(f os.FileInfo) bool {
		if f.Name() == ".DS_Store" {
			return false
		}
		if f.Name() == ".git" {
			return false
		}
		if !f.IsDir() && !ncm.IsNCM(f.Name()) {
			return false
		}

		return true
	}
	files := filesys.ReadDir(path, filter)
	nextDirs := make([]string, 0)
	for _, file := range files {
		if file.IsDir() {
			nextDirs = append(nextDirs, path+"/"+file.Name())
		} else if ncm.IsNCM(file.Name()) {
			exit.Add(1)
			jobsCh <- (path + "/" + file.Name())
			ncm.ConsoleOut(path + "/" + file.Name() + "添加到处理队列")
		}
	}
	for _, dirPath := range nextDirs {
		findDumpJobs(dirPath, jobsCh, exit)
	}
}

func processJobs(jobsCh chan string, exit *sync.WaitGroup) {
	for path := range jobsCh {
		ncm.Dump(path, _OW)
		exit.Done()
	}
}

// TODO: improve error handle
func main() {
	var rootPath string
	flag.BoolVar(&ncm.MUTE, "m", true, "是否静音执行")
	flag.BoolVar(&_OW, "o", false, "是否覆盖已经存在的结果文件")
	flag.StringVar(&rootPath, "p", "", "NCM文件所在目录")
	flag.Parse()
	var exit sync.WaitGroup
	concurrency := runtime.NumCPU()
	jobsCh := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		go processJobs(jobsCh, &exit)
	}
	findDumpJobs(rootPath, jobsCh, &exit)

	close(jobsCh)
	exit.Wait()
}
