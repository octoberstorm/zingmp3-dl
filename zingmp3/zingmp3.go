package zingmp3

// TODO clean not in-use code

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cheggaaa/pb"
	"github.com/moovweb/gokogiri"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
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
	ALBUM_SINGLE_SONG_LINK_XPATH = `//div[@id="playlistItems"]//a[@class="fn-name"]`
	FINAL_LINK_PRE               = "http://v3.mp3.zing.vn/download/vip/song/"
)

type Downloader struct {
	Url         string
	DownloadDir string
	// Song or Album
	UrlType string

	// Config for downloader, e.g. network timeout, create new dir...
	Config map[string]interface{}
}

type SongData struct {
	Id          string
	Name        string
	Artist      string
	Link        string
	Cover       string
	Qualities   []string
	Source_list []string
	Source_base string
	Lyric       string
}

type XmlData struct {
	Msg  string
	Data []SongData
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
			// http://mp3.zing.vn/bai-hat/Fly-Me-To-The-Moon-Westlife/ZWZ9BCIB.html
			matched, _ := regexp.MatchString(`\/bai-hat\/.*/.*\.html`, v.String())
			if matched {
				links = append(links, v.String())
			}
		}
	}

	return links, nil
}

// Get final redirected link
func getFinalLink(songUrl string) (string, error) {
	downloadURLResp, err := client.Get(songUrl)
	defer downloadURLResp.Body.Close()
	if err != nil {
		return "", err
	}

	songContentBuf := new(bytes.Buffer)
	songContentBuf.ReadFrom(downloadURLResp.Body)
	songContent := songContentBuf.String()

	xmlUrlR := regexp.MustCompile(`data-xml="(.*)" class`)

	if !xmlUrlR.MatchString(songContent) {
		return "", errors.New("XML url not found")
	}

	xmlUrl := xmlUrlR.FindStringSubmatch(songContent)[1]
	// fmt.Println(xmlUrl)

	xmlContentResp, err := client.Get(xmlUrl)
	if err != nil {
		return "", err
	}

	xmlContentBuf := new(bytes.Buffer)
	xmlContentBuf.ReadFrom(xmlContentResp.Body)
	jsonData := xmlContentBuf.String()

	var xmlData XmlData

	err = json.Unmarshal([]byte(jsonData), &xmlData)
	if err != nil {
		return "", err
	}

	url := xmlData.Data[0].Source_list[0]
	if url == "" {
		url = xmlData.Data[0].Source_list[1]
	}

	if url == "" {
		return "", errors.New("Invalid Song URL")
	}

	finalUrl := "http://" + url

	return finalUrl, nil
}

func (d *Downloader) Download() {
	links := d.DownloadLinks()
	// channels := make([]chan int, len(links))
	wg.Add(len(links))

	for i := range links {
		link := links[i]
		// channels[i] = make(chan int)
		// TODO handle error
		go d.RunDownload(link, &wg)
	}
	wg.Wait()

	fmt.Println("")

}

func (d *Downloader) RunDownload(link string, wg *sync.WaitGroup) error {
	var source io.Reader
	var sourceSize int64

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
	// code := matches[2]

	finalLink, err := getFinalLink(link)
	if err != nil {
		return err
	}

	filename := title + ".mp3"

	path := d.DownloadDir + filename
	out, err := os.Create(path)
	defer out.Close()

	resp, err := client.Get(finalLink)
	defer resp.Body.Close()

	if err != nil {
		return err
	}

	i, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	sourceSize = int64(i)
	source = resp.Body
	// create bar
	bar := pb.StartNew(int(sourceSize)).
		SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10).
		Prefix("[" + filename + "] ")
	bar.ShowSpeed = true

	// fmt.Printf("Downloading %v...\n", link)
	// create multi writer
	writer := io.MultiWriter(out, bar)

	// and copy
	_, err = io.Copy(writer, source)
	bar.Increment()

	bar.Finish()
	print("")
	if err != nil {
		return err
	}
	// fmt.Printf("Song downloaded to %v. Copied %v bytes\n", path, n)
	return nil
}
