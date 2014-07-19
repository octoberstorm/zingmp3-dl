package main

import (
	"os"
	"sync"
	"testing"
)

func TestValidSongInfo(t *testing.T) {
	url := "http://mp3.zing.vn/bai-hat/Maroc-7-The-Shadows/ZW6OU68E.html"
	filename := "Maroc-7-The-Shadows.mp3"
	downloadDir := "/tmp/"

	song := Song{URL: url, DownloadDir: downloadDir}
	song.ParseURL()

	if song.URL != url || song.DownloadDir != downloadDir {
		t.Error("Invalid song URL or Download dir")
	}

	if song.Code != "ZW6OU68E" {
		t.Error("Invalid song code.")
	}

	if song.FileName() != filename {
		t.Errorf("Expected Filename to be %v, was %v", filename, song.FileName())
	}

	if song.Path() != (downloadDir + filename) {
		t.Error("Invalid song path")
	}

}

func TestDownloadSong(t *testing.T) {
	url := "http://mp3.zing.vn/bai-hat/Maroc-7-The-Shadows/ZW6OU68E.html"
	filename := "Maroc-7-The-Shadows.mp3"
	downloadDir := "/tmp/"

	song, _ := NewSong(url, downloadDir)
	var wg sync.WaitGroup
	wg.Add(1)
	go song.Download(&wg)
	wg.Wait()

	file, err := os.Open(downloadDir + filename)
	if err != nil {
		t.Errorf("Download failed: %v\n", err.Error())
	}

	fileInfo, _ := file.Stat()
	if fileInfo.Size() == 0 {
		t.Error("Downloaded file is zero length.")
	}
}

func TestAlbumSongList(t *testing.T) {
	// albumUrl := "http://mp3.zing.vn/album/Romantic-Guitar-Various-Artists/ZWZA09UC.html"
	// downloadDir := "/tmp"
	// album := NewAlbum(albumUrl, downloadDir)

}
