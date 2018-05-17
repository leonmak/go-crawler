package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"net/http"
	"strings"
	"os"
	log "github.com/llimllib/loglevel"
)

type Link struct {
	url string
	text string  // tag where href was found
	depth int
}

func (self Link) String() string {
	spacer := strings.Repeat("\t", self.depth)
	return fmt.Sprintf("%s%s (%d) - %s", spacer, self.text, self.depth, self.url)
}

func (self Link) Valid() bool {
	if len(self.text) == 0 {
		return false
	}
	if len(self.url) == 0 ||
		strings.Contains(strings.ToLower(self.url), "javascript") {
		return false
	}
	return true
}

// API-specific Errors
type HttpGetError struct {
	original string
}

func (self HttpGetError) Error() string {
	return self.original
}

func ExtractLinks(resp *http.Response, depth int) (links []Link) {
	page := html.NewTokenizer(resp.Body) // tokenizer parse html into tokens

	var start *html.Token
	var text string

	for {
		_ = page.Next() 		// move tokenizer forward
		token := page.Token()  	// get token

		if token.Type == html.ErrorToken {
			return
		}

		// Set text for previous token if have start
		if start != nil && token.Type == html.TextToken {
			text = fmt.Sprintf("%s%s", text, token.Data)
		}

		// Set start if anchor token
		if token.DataAtom == atom.A {
			switch token.Type {
			case html.StartTagToken:
				if len(token.Attr) > 0 {
					start = &token
				}
			case html.EndTagToken:
				if start == nil {
					log.Warnf("Link End found, no Start: %s", text)
					return
				}
				link := NewLink(*start, text, depth)
				if link.Valid() {
					links = append(links, link)
					log.Debugf("Link Found %v", link)
				}
				start = nil
				text = ""
			}
		}
	}

	log.Debug(links)
	return links
}

// Create link
func NewLink(tag html.Token, text string, depth int) Link {
	link := Link {text: strings.TrimSpace(text), depth: depth}
	for _, attr := range tag.Attr {
		if attr.Key == atom.Href.String() {
			link.url = strings.TrimSpace(attr.Val)
		}
	}
	return link
}

// Iterative BFS crawler with channels
func crawler(urls []string, maxDepth int) (res []Link) {
	frontier := make(chan []Link)
	visited := make(map[string]bool)  			// map string url to bool isVisited

	requestTokens := make(chan struct{}, 10)  	// set limit of 10 concurrent requests
	n := len(urls) 								// number of pending sends
	go func() {
		initialLinks := []Link{}
		for _, url := range urls {
			initialLink := Link{text: url, url: strings.TrimSpace(url), depth: 0}
			initialLinks = append(initialLinks, initialLink)
		}
		frontier <-initialLinks
	}()

	// 1. Dequeue frontier, get its links, append to frontier.
	// 2. Increment depth. If max depth, stop.

	for ; n > 0; n-- {
		// receive set of neighbours from channel and decrease n
		links := <-frontier

		for _, link := range links {
			if visited[link.url] {
				continue
			}

			visited[link.url] = true
			res = append(res, link)
			log.Infof("Appended: %s at Depth: %d", link.url, link.depth)
			log.Debugf("n sends to send: %d", n)

			// don't add children sets to frontier if depth is maxed
			if link.depth == maxDepth {
				continue
			}

			n++
			go func(link Link) {

				// send children to channel
				resp, err := getUrl(link.url)
				if err != nil {
					// Last url always bug out
					// `write tcp 127.0.0.1:49345->127.0.0.1:8000: write: broken pipe`
					frontier<- []Link{}
					return
				}

				requestTokens <- struct{}{}
				newLinks := ExtractLinks(resp, link.depth + 1)
				<-requestTokens

				frontier<- newLinks
			}(link)
		}
		//close(frontier)
	}
	return
}

func getUrl(url string) (resp *http.Response, err error) {
	log.Debugf("Downloading %s", url)
	resp, err = http.Get(url)
	if err != nil {
		log.Debugf("Error: %s", err)
		return
	}
	if resp.StatusCode > 299 {
		errStr := fmt.Sprintf("Error (%d): %s", resp.StatusCode, url)
		log.Debug(HttpGetError{original: errStr})
		return
	}
	return
}

func writeToFile(path string, text string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte(text)); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func writeLinksToCsv(outputPath string, links []Link) {
	err := os.RemoveAll(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	os.Create(outputPath)
	writeToFile(outputPath, "text, url, depth\n")
	for _, link := range links {
		text := strings.Replace(link.text, "\n", " ", -1)
		row := fmt.Sprintf("%s, %s, %d\n", text, link.url, link.depth)
		writeToFile(outputPath, row)
	}
}

func initVars(depth *int) {
	flag.IntVar(depth,
		"depth",
		1,
		"Max depth to crawl, root is at depth 0, default: 1")
	flag.Parse()

}

func main() {
	var maxDepth int  // TEST: with maxDepth >/</== tree depth
	initVars(&maxDepth)

	log.SetPriorityString("info")
	//log.SetPriorityString("debug")
	log.SetPrefix("Crawler ")

	log.Debugf("Args: %v", os.Args[1:])
	if len(os.Args) < 2 {
		log.Fatalln("Missing Url arg")
	}

	outputDir := "output"
	os.RemoveAll(outputDir)
	os.MkdirAll(outputDir, os.ModePerm)
	csvPath := "output.csv"

	urls := os.Args[1:]
	if os.Args[1] == "-depth" {
		urls = os.Args[3:]
	}

	if len(urls) > 1 {
		for _, url := range urls {
			log.Infof("====================================")
			log.Infof("CRAWLING: %s", url)
			log.Infof("====================================")
			links := crawler([]string{url}, maxDepth)
			r := strings.NewReplacer(":", "-", "/", "-", ".", "-")
			urlStrip := r.Replace(url)
			path := outputDir + "/" + urlStrip + ".csv"
			log.Infof("Results in: %s", path)
			writeLinksToCsv(path, links)
		}
	} else {
		links := crawler(urls, maxDepth)
		writeLinksToCsv(outputDir + "/" + csvPath, links)
	}

}

