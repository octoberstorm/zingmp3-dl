package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
)

const (
	ZING_ROOT_URL  = "http://mp3.zing.vn/"
	SONG_URL_PART  = "bai-hat"
	ALBUM_URL_PART = "album"
	OUTPUT_EXT     = ".mp3"

	SONG_REGEX = ZING_ROOT_URL + SONG_URL_PART + "/(.*)" + "[/.](.*).html"

	DOWNLOAD_URL_PRE = "http://mp3.zing.vn/download/vip/song/"
)

type Song struct {
	URL         string
	DownloadDir string
	Code        string
	Title       string
}

func NewSong(url, dir string) *Song {
	song := Song{URL: url, DownloadDir: dir}
	song.ParseURL()
	return &song
}

func (s *Song) ParseURL() error {
	r, err := regexp.Compile(SONG_REGEX)
	if err != nil {
		return err
	}

	matches := r.FindStringSubmatch(s.URL)
	s.Title = matches[1]
	s.Code = matches[2]
	return nil
}

func (s *Song) FileName() string {
	return s.Title + OUTPUT_EXT
}

func (s *Song) Path() string {
	return s.DownloadDir + s.FileName()
}

func (s *Song) Download() error {
	downloadURL := DOWNLOAD_URL_PRE + s.Code

	downloadURLResp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}

	realURL := downloadURLResp.Request.URL.String()
	regx, err := regexp.Compile(`(.*)\?filename=.*`)
	if err != nil {
		return err
	}

	finalURLMatch := regx.FindStringSubmatch(realURL)
	finalURL := finalURLMatch[1]

	out, err := os.Create(s.Path())
	defer out.Close()

	fmt.Printf("Downloading %v...\n", realURL)
	resp, err := http.Get(finalURL)
	defer resp.Body.Close()

	if err != nil {
		return err
	}

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Song downloaded to %v. Copied %v bytes\n", s.Path(), n)
	return nil
}

func main() {
	currentDir, _ := os.Getwd()

	downloadDir := flag.String("dir", currentDir+"/", "Download Directory")
	flag.Parse()

	if len(os.Args) == 1 {
		fmt.Println("Usage: zingmp3-dl [Options] <Song_URL>")
		fmt.Println("Options:")
		flag.PrintDefaults()
		return
	}

	songURL := os.Args[len(os.Args)-1]

	if songURL == "" || *downloadDir == "" {
		fmt.Println("Please provide Song URL and Download Dir")
		return
	}

	song := NewSong(songURL, *downloadDir)
	fmt.Println(song.FileName())
	song.Download()
}
