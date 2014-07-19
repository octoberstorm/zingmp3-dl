package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/moovweb/gokogiri"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"
)

const (
	ZING_ROOT_URL  = "http://mp3.zing.vn/"
	SONG_URL_PART  = "bai-hat"
	ALBUM_URL_PART = "album"
	OUTPUT_EXT     = ".mp3"

	SONG_REGEX  = ZING_ROOT_URL + SONG_URL_PART + "/(.*)" + "[/.](.*).html"
	ALBUM_REGEX = ZING_ROOT_URL + ALBUM_URL_PART + "/(.*)/(.*).html"

	DOWNLOAD_URL_PRE = "http://mp3.zing.vn/download/vip/song/"
)

type Song struct {
	URL         string
	DownloadDir string
	Code        string
	Title       string
}

var timeout = time.Duration(5 * time.Second)
var transport = http.Transport{
	Dial: dialTimeout,
}

var client = http.Client{
	Transport: &transport,
}

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

func isDirExist(dir string) bool {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		// TODO other error, i.e. permission..
	}
	return true
}

func NewSong(url, dir string) (*Song, error) {
	song := Song{URL: url, DownloadDir: dir}
	err := song.ParseURL()

	if err != nil {
		return &Song{}, err
	}
	return &song, nil
}

func (s *Song) ParseURL() error {
	m, _ := regexp.MatchString(`^.*\/$`, s.DownloadDir)

	if !m {
		s.DownloadDir = s.DownloadDir + "/"
	}

	if !isDirExist(s.DownloadDir) {
		return errors.New("Directory does not exist")
	}

	r, err := regexp.Compile(SONG_REGEX)
	if err != nil {
		return errors.New("Incorrect Song/Album URL")
	}

	matches := r.FindStringSubmatch(s.URL)
	fmt.Println(matches)
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

	downloadURLResp, err := client.Get(downloadURL)
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

func songListFromAlbum(albumUrl string) ([]string, error) {
	var links []string

	resp, err := client.Get(albumUrl)
	if err != nil {
		return links, err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	content := buf.Bytes()

	doc, err := gokogiri.ParseHtml(content)

	if err != nil {
		return links, err
	}

	hrefs, _ := doc.Root().Search(`//*[@id="_plContainer"]//a`)

	for i := range hrefs {
		v := hrefs[i].Attribute("href")
		if v != nil {
			matched, _ := regexp.MatchString(`^\/bai-hat/.*\.html`, v.String())
			if matched {
				links = append(links, v.String())
			}
		}
	}
	return links, nil
}

func getLinkType(url string) (string, error) {
	songMatched, err := regexp.MatchString(ZING_ROOT_URL+`bai-hat/.*\.html`, url)
	if err != nil {
		return "", err
	}

	if songMatched {
		return "song", nil
	}

	albumMatched, _ := regexp.MatchString(ZING_ROOT_URL+`album/.*\.html`, url)
	if albumMatched {
		return "album", nil
	}

	return "", errors.New("Please provide correct URL")
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

	zingUrl := os.Args[len(os.Args)-1]

	if zingUrl == "" || *downloadDir == "" {
		fmt.Println("Please provide Song URL and Download Dir")
		return
	}

	linkType, err := getLinkType(zingUrl)
	if err != nil {
		fmt.Println(err.Error())
	}

	if linkType == "song" {
		song, err := NewSong(zingUrl, *downloadDir)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		song.Download()
	} else if linkType == "album" {
		fmt.Println("Album detected")

		songUrl, err := songListFromAlbum(zingUrl)

		if err != nil {
			fmt.Printf("Error: %v\n", err.Error())
			return
		}

		for i := range songUrl {
			url := songUrl[i]
			song, err := NewSong("http://mp3.zing.vn"+url, *downloadDir)

			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			song.Download()
		}
	}
	// fmt.Println(song.FileName())
}
