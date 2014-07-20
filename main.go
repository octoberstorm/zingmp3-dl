package main

import (
	"flag"
	"fmt"
	"github.com/octoberstorm/zingmp3-dl/zingmp3"
	"os"
)

func main() {
	currentDir, _ := os.Getwd()

	downloadDir := flag.String("dir", currentDir+"/", "Download Directory")
	flag.Parse()

	if len(os.Args) == 1 {
		fmt.Println("Usage: zingmp3-dl [Options] <Song or Album URL>")
		fmt.Println("Options:")
		flag.PrintDefaults()
		return
	}

	zingUrl := os.Args[len(os.Args)-1]

	if zingUrl == "" || *downloadDir == "" {
		fmt.Println("Please provide Song URL and Download Dir")
		return
	}

	config := map[string]interface{}{
		"timeout": 5,
	}
	// fmt.Println(zingUrl)
	downloader, _ := zingmp3.NewDownloader(zingUrl, *downloadDir, config)

	downloader.Download()
	// fmt.Println(err)
	// if err != nil {
	// 	downloader.Download()
	// } else {
	// 	fmt.Println("Error")
	// }
}
