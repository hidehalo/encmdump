package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
)

// Overwrite flag
var _OW bool

// Mute flag
var _MUTE bool

func handleError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func consoleOut(message string) {
	if !_MUTE {
		fmt.Println(message)
	}
}

func findDumpJobs(path string, jobsCh chan string, exit *sync.WaitGroup) {
	filter := func(f os.FileInfo) bool {
		if f.Name() == ".DS_Store" {
			return false
		}
		if f.Name() == ".git" {
			return false
		}

		return true
	}
	files := readDir(path, filter)
	nextDirs := make([]string, 0)
	for _, file := range files {
		if isNCM(file.Name()) {
			exit.Add(1)
			jobsCh <- (path + "/" + file.Name())
		} else if file.IsDir() {
			nextDirs = append(nextDirs, path+"/"+file.Name())
		}
	}
	for _, dirPath := range nextDirs {
		findDumpJobs(dirPath, jobsCh, exit)
	}
}

func processJobs(jobsCh chan string, exit *sync.WaitGroup) {
	for path := range jobsCh {
		dump(path, _OW)
		exit.Done()
	}
}

// TODO: improve error handle
func main() {
	var rootPath string
	flag.BoolVar(&_MUTE, "m", true, "是否静音执行")
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
