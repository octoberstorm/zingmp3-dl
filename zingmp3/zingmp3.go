package zingmp3

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/moovweb/gokogiri"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

// song: mp3.zing.vn/bai-hat/Out-of-My-Heart-Into-Your-Head-BBMak/IWZCIWE6.html
// album: mp3.zing.vn/album/Romantic-Guitar-Various-Artists/ZWZA09UC.html?st=5

// Specs
// config = map[string]interface{}{
//    timeout: 5,
// }
// d := NewDownloader(url, dir, config)
// err := d.Download()
const (
	ZING_URL                     = `^http:\/\/mp3.zing.vn\/(.*)\/(.*)\/(.*)\.html$`
	ALBUM_SINGLE_SONG_LINK_XPATH = `//*[@id="_plContainer"]//a[@class="single-play"]`
	FINAL_LINK_PRE               = "http://mp3.zing.vn/download/vip/song/"
)

type Downloader struct {
	Url         string
	DownloadDir string
	// Song or Album
	UrlType string

	// Config for downloader, e.g. network timeout, create new dir...
	Config map[string]interface{}
}

var wg sync.WaitGroup

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

func NewDownloader(url, dir string, config map[string]interface{}) (*Downloader, error) {
	var d = Downloader{}
	// validate input, check downloader type
	err := d.SetUrl(url)

	if err != nil {
		return &d, err
	}

	err = d.SetDownloadDir(dir)

	if err != nil {
		return &d, err
	}

	return &d, nil
}

func (d *Downloader) SetUrl(url string) error {
	d.Url = url
	return d.ParseUrl()
}

func (d *Downloader) SetDownloadDir(dir string) error {
	m, _ := regexp.MatchString(`^.*\/$`, dir)

	if !m {
		dir = dir + "/"
	}

	if !isDirExist(dir) {
		return errors.New("Directory does not exist")
	}

	d.DownloadDir = dir
	return nil
}

func (d *Downloader) ParseUrl() error {
	r, _ := regexp.Compile(ZING_URL)
	matched := r.MatchString(d.Url)

	if !matched {
		return errors.New("Incorrect Zing URL")
	}

	urlMatches := r.FindStringSubmatch(d.Url)

	urlType := urlMatches[1]

	if urlType == "bai-hat" {
		d.UrlType = "song"
	} else if urlType == "album" {
		d.UrlType = "album"
	} else {
		return errors.New("Incorrect Zing URL")
	}
	return nil
}

// Get list of link to download
func (d *Downloader) DownloadLinks() []string {
	var links []string
	if d.UrlType == "song" {
		links = []string{d.Url}
	}
	if d.UrlType == "album" {
		content, err := getHTMLContent(d.Url)
		if err != nil {
			// TODO may return err?
			return links
		}
		links, _ = songUrlsFromAlbum(content)
	}
	return links
}

func getHTMLContent(url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return []byte{}, err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	content := buf.Bytes()
	return content, nil
}

func songUrlsFromAlbum(albumHTMLContent []byte) ([]string, error) {
	var links []string

	doc, err := gokogiri.ParseHtml(albumHTMLContent)

	if err != nil {
		return links, err
	}

	hrefs, _ := doc.Root().Search(ALBUM_SINGLE_SONG_LINK_XPATH)

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

// Get final redirected link
func getFinalLink(songCode string) (string, error) {
	downloadURL := FINAL_LINK_PRE + songCode

	downloadURLResp, err := client.Get(downloadURL)
	if err != nil {
		return "", err
	}

	realURL := downloadURLResp.Request.URL.String()
	regx, err := regexp.Compile(`(.*)\?filename=.*`)
	if err != nil {
		return "", err
	}

	finalURLMatch := regx.FindStringSubmatch(realURL)
	finalURL := finalURLMatch[1]
	return finalURL, nil
}

func (d *Downloader) Download() {
	links := d.DownloadLinks()
	wg.Add(len(links))

	for i := range links {
		link := links[i]
		// TODO handle error
		go d.RunDownload(link, &wg)
	}
	wg.Wait()
}

func (d *Downloader) RunDownload(link string, wg *sync.WaitGroup) error {
	defer wg.Done()

	codeReg, err := regexp.Compile(`bai-hat/(.*)/([A-Z0-9]+)\.html$`)
	if err != nil {
		return err
	}

	matches := codeReg.FindStringSubmatch(link)
	if len(matches) != 3 {
		return errors.New("Wrong link")
	}

	title := matches[1]
	code := matches[2]

	finalLink, err := getFinalLink(code)
	if err != nil {
		return err
	}

	path := d.DownloadDir + title + ".mp3"
	out, err := os.Create(path)
	defer out.Close()

	fmt.Printf("Downloading %v...\n", link)
	resp, err := client.Get(finalLink)
	defer resp.Body.Close()

	if err != nil {
		return err
	}

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("Song downloaded to %v. Copied %v bytes\n", path, n)
	return nil
}
