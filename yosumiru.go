// jeff@archlinux.org
package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
)

var CacheFile string

func init() {
	xch := os.Getenv("XDG_CACHE_HOME")
	if xch == "" {
		home := os.Getenv("HOME")
		xch = filepath.Join(home, ".cache")
	}
	CacheFile = filepath.Join(xch, "yosumiru", "feeds_seen.json")
}

func main() {
	if LastFeeds() {
		RunUpdate()
	}
}

func RunUpdate() {
	cmd := exec.Command("sudo", "pacman", "-Syu")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
}

type rssFeedXml struct {
	XMLName xml.Name `xml:"rss"`
	Channel *RssFeed
}

type RssFeed struct {
	XMLName xml.Name   `xml:"channel"`
	Items   []*RssItem `xml:"item"`
}

type RssItem struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"title"`
	Description string   `xml:"description"`
	PubDate     string   `xml:"pubDate,omitempty"`
}

type RssEnclosure struct {
	XMLName xml.Name `xml:"enclosure"`
	Url     string   `xml:"url,attr"`
	Length  string   `xml:"length,attr"`
	Type    string   `xml:"type,attr"`
}

const ARCH_NEWS_FEED = "https://www.archlinux.org/feeds/news/"

type FeedsSeen struct {
	SeenMap map[string]struct{} `json:"seen_map"`
}

type Entry struct {
	Title       string
	PubDate     string
	Description string
	Hash        string
}

func (e *Entry) HashIt() {
	var entryb []byte
	entryb = append(entryb, []byte(e.Title)...)
	entryb = append(entryb, []byte(e.PubDate)...)
	entryb = append(entryb, []byte(e.Description)...)

	e.Hash = fmt.Sprintf("%x", sha1.Sum(entryb))
}

func (e *Entry) Print() {
	fmt.Printf(`* %s
[%s]
%s
-----

`, e.Title, e.PubDate, fixDesc(e.Description))
}

func checkUser(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(question + " [Y/n]: ")
	text, _ := reader.ReadString('\n')
	return (text == "\n" ||
		text == "Y\n" ||
		text == "y\n")
}

func GetFeedsSeen(fsc chan *FeedsSeen) {
	var fs FeedsSeen

	cache, err := ioutil.ReadFile(CacheFile)
	if err != nil {
		_, ok := err.(*os.PathError)
		if !ok {
			panic(err)
		}
	} else {
		err = json.Unmarshal(cache, &fs)
		if err != nil {
			panic(err)
		}
	}

	if fs.SeenMap == nil {
		fs.SeenMap = make(map[string]struct{}, 0)
	}

	fsc <- &fs
}

func GetArchFeed(rc chan *rssFeedXml) {
	resp, err := http.Get(ARCH_NEWS_FEED)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	bod, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var res rssFeedXml
	err = xml.Unmarshal(bod, &res)
	if err != nil {
		fmt.Printf("%v\n%#v\n", err, err)
	}

	rc <- &res
}

func SaveCache(f *FeedsSeen) {
	b, err := json.Marshal(*f)
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(filepath.Dir(CacheFile), 0700)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(CacheFile, b, 0600)
	if err != nil {
		panic(err)
	}
}

func LastFeeds() bool {
	feeds := make(chan *rssFeedXml, 0)
	caches := make(chan *FeedsSeen, 0)

	go GetFeedsSeen(caches)
	go GetArchFeed(feeds)

	feed := <-feeds
	cache := <-caches

	entries := make([]*Entry, 0)
	// copy & hash records
	for _, v := range feed.Channel.Items {
		entries = append(entries, &Entry{
			Title:       v.Title,
			PubDate:     v.PubDate,
			Description: v.Description,
		})
	}

	// hash entries
	count := new(int64)
	done := make(chan struct{}, 0)
	for _, v := range entries {
		go func() {
			v.HashIt()
			if atomic.AddInt64(count, 1) == int64(len(entries)) {
				done <- struct{}{}
			}
		}()
	}
	<-done

	// print unseen records
	for _, v := range entries {
		_, ok := cache.SeenMap[v.Hash]
		if ok {
			continue
		}
		v.Print()
		if checkUser("Confirm seen?") {
			cache.SeenMap[v.Hash] = struct{}{}
		}
	}

	c := checkUser("Continue with update?")
	if c {
		SaveCache(cache)
	}
	return c
}

func fixDesc(desc string) string {
	cmd := exec.Command("pandoc", "-f", "html", "-t", "org")
	sip, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	go func() {
		_, err := sip.Write([]byte(desc))
		if err != nil {
			panic(err)
		}
		sip.Close()
	}()

	res, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	return string(res)
}
