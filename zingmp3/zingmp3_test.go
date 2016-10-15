package zingmp3

import (
	"fmt"
	"os"
	"testing"
)

func TestValidSongUrlInfo(t *testing.T) {
	songUrl := "http://mp3.zing.vn/bai-hat/Maroc-7-The-Shadows/ZW6OU68E.html"
	albumUrl := "http://mp3.zing.vn/album/Romantic-Guitar-Various-Artists/ZWZA09UC.html"
	incorrectUrl := "http://incorrect.com/song/test.html"

	dir := "/tmp"
	config := map[string]interface{}{
		"timeout": 5,
	}
	d, _ := NewDownloader(songUrl, dir, config)

	if d.UrlType != "song" {
		t.Error("Invalid UrlType")
	}

	// correct dir
	if d.DownloadDir != "/tmp/" {
		t.Errorf("Incorrect Download dir, expected %v, was %v", "/tmp/", d.DownloadDir)
	}

	err := d.SetDownloadDir("/wrongdir")
	if err == nil {
		t.Error("Wrong download dir should raise error")
	}

	var links []string
	links = append(links, songUrl)

	if len(d.DownloadLinks()) != 1 || d.DownloadLinks()[0] != songUrl {
		t.Errorf("Invalid download links, expected %v, was %v", links, d.DownloadLinks())
	}

	d.SetUrl(albumUrl)
	if d.UrlType != "album" {
		t.Errorf("Invalid UrlType, expected %v, was %v", "album", d.UrlType)
	}

	if len(d.DownloadLinks()) != 11 {
		t.Errorf("Total links is not correct, expected 11, was %v", len(d.DownloadLinks()))
	}

	secondLink := "http://mp3.zing.vn/bai-hat/Johnny-Guitar-Various-Artists/ZW60ZEAW.html"
	if d.DownloadLinks()[1] != secondLink {
		t.Errorf("Wrong link, expected %v, was %v", secondLink, d.DownloadLinks()[1])
	}

	// test incorrect url
	err = d.SetUrl(incorrectUrl)
	if err.Error() != "Incorrect Zing URL" {
		t.Errorf("Incorrect URL should raise error")
	}

	d, err = NewDownloader(incorrectUrl, dir, config)
	if err == nil {
		t.Error("Incorrect URL should raise error")
	}
}

func TestDownload(t *testing.T) {
	songUrl := "http://mp3.zing.vn/bai-hat/Maroc-7-The-Shadows/ZW6OU68E.html"
	// albumUrl := "http://mp3.zing.vn/album/Romantic-Guitar-Various-Artists/ZWZA09UC.html"

	dir := "/tmp/"
	config := map[string]interface{}{
		"timeout": 5,
	}
	d, _ := NewDownloader(songUrl, dir, config)

	d.Download()

	filename := "Maroc-7-The-Shadows.mp3"
	filepath := d.DownloadDir + filename
	fmt.Println("Downloading to ", filepath)

	file, err := os.Open(filepath)
	if err != nil {
		t.Errorf("Download failed: %v\n", err.Error())
	}

	fileInfo, _ := file.Stat()
	if fileInfo.Size() == 0 {
		t.Error("Downloaded file is zero length.")
	}
}
