// jeff@archlinux.org
package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	//	"net/http"
	"os/exec"
	//	"strings"
	"bufio"
	"os"
)

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

// private wrapper around the RssFeed which gives us the <rss>..</rss> xml
type rssFeedXml struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel *RssFeed
}

type RssImage struct {
	XMLName xml.Name `xml:"image"`
	Url     string   `xml:"url"`
	Title   string   `xml:"title"`
	Link    string   `xml:"link"`
	Width   int      `xml:"width,omitempty"`
	Height  int      `xml:"height,omitempty"`
}

type RssTextInput struct {
	XMLName     xml.Name `xml:"textInput"`
	Title       string   `xml:"title"`
	Description string   `xml:"description"`
	Name        string   `xml:"name"`
	Link        string   `xml:"link"`
}

type RssFeed struct {
	XMLName        xml.Name `xml:"channel"`
	Title          string   `xml:"title"`       // required
	Link           string   `xml:"link"`        // required
	Description    string   `xml:"description"` // required
	Language       string   `xml:"language,omitempty"`
	Copyright      string   `xml:"copyright,omitempty"`
	ManagingEditor string   `xml:"managingEditor,omitempty"` // Author used
	WebMaster      string   `xml:"webMaster,omitempty"`
	PubDate        string   `xml:"pubDate,omitempty"`       // created or updated
	LastBuildDate  string   `xml:"lastBuildDate,omitempty"` // updated used
	Category       string   `xml:"category,omitempty"`
	Generator      string   `xml:"generator,omitempty"`
	Docs           string   `xml:"docs,omitempty"`
	Cloud          string   `xml:"cloud,omitempty"`
	Ttl            int      `xml:"ttl,omitempty"`
	Rating         string   `xml:"rating,omitempty"`
	SkipHours      string   `xml:"skipHours,omitempty"`
	SkipDays       string   `xml:"skipDays,omitempty"`
	Image          *RssImage
	TextInput      *RssTextInput
	Items          []*RssItem `xml:"item"`
}

type RssItem struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"title"`       // required
	Link        string   `xml:"link"`        // required
	Description string   `xml:"description"` // required
	Author      string   `xml:"author,omitempty"`
	Category    string   `xml:"category,omitempty"`
	Comments    string   `xml:"comments,omitempty"`
	Enclosure   *RssEnclosure
	Guid        string `xml:"guid,omitempty"`    // Id used
	PubDate     string `xml:"pubDate,omitempty"` // created or updated
	Source      string `xml:"source,omitempty"`
}

type RssEnclosure struct {
	XMLName xml.Name `xml:"enclosure"`
	Url     string   `xml:"url,attr"`
	Length  string   `xml:"length,attr"`
	Type    string   `xml:"type,attr"`
}

const ARCH_NEWS_FEED = "https://www.archlinux.org/feeds/news/"

func LastFeeds() bool {
	// resp, err := http.Get(ARCH_NEWS_FEED)
	// if err != nil {
	// 	panic(err)
	// }
	// defer resp.Body.Close()

	bod, err := ioutil.ReadFile("newsfeed.txt")
	if err != nil {
		panic(err)
	}

	var res rssFeedXml
	err = xml.Unmarshal(bod, &res)
	if err != nil {
		fmt.Printf("%v\n%#v\n", err, err)
	}

	// print most recent 2 items
	for i := 2; i > 0; i-- {
		fmt.Printf(`* %s
[%s]
%s
-----

`, res.Channel.Items[i].Title, res.Channel.Items[i].PubDate, fixDesc(res.Channel.Items[i].Description))
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Given the post, continue? [Y/n]: ")
		text, _ := reader.ReadString('\n')
		if text == "\n" ||
			text == "Y\n" ||
			text == "y\n" {
			continue
		}

	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Continue with update? [Y/n]: ")
	text, _ := reader.ReadString('\n')
	return (text == "\n" ||
		text == "Y\n" ||
		text == "y\n")
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
