package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Link struct {
	Title      string
	URL        string
	TimeAdded  string
	Tags       string
	IsURLTitle bool
}

func usage() {
	fmt.Println("Usage: take-out-pocket-csv <Pocket export HTML> <Output CSV file name>")
}

func findLink(node *html.Node) *Link {
	var title, href, tags string
	var timeAdded time.Time

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			title = c.Data
		}
	}

	for _, v := range node.Attr {
		if v.Key == "href" {
			href = v.Val
		}

		if v.Key == "tags" {
			tags = v.Val
		}

		if v.Key == "time_added" {
			n, _ := strconv.Atoi(v.Val)
			timeAdded = time.Unix(int64(n), 0)
		}
	}

	return &Link{Title: title, URL: href, Tags: tags, TimeAdded: timeAdded.Format("2006/01/02"), IsURLTitle: href == title}
}

func findLinks(doc *html.Node, links *[]*Link) {
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			if c.DataAtom == atom.A {
				*links = append(*links, findLink(c))
			}

			findLinks(c, links)
		}
	}
}

func findTitle(node *html.Node) string {
	title := ""

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			title = c.Data
		}
	}

	return title
}

func findTitleText(node *html.Node, title *string) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			if c.DataAtom == atom.Title {
				fetchedTitle := findTitle(c)

				// If the host is not found, etc., the title will be an empty string.
				if fetchedTitle != "" {
					*title = fetchedTitle
				}
			}
		}

		if *title == "" {
			findTitleText(c, title)
		}
	}
}

func FetchTitleText(link *Link, Index int) {
	fmt.Println(strconv.Itoa(Index)+" - Request URL: ", link.URL)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	res, err := client.Get(link.URL)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return
	}

	defer res.Body.Close()

	doc, err := html.Parse(res.Body)
	if err != nil {
		fmt.Println("ERROR: ", err)
		return
	}

	var title string
	findTitleText(doc, &title)

	link.Title = title
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Error: You must specify pocket export html file and output file name.")
		fmt.Println("============================ USAGE ================================")
		usage()
		os.Exit(0)
	}

	htmlPath := os.Args[1]
	outputPath := os.Args[2]

	_, err := os.Stat(htmlPath)
	if err != nil {
		fmt.Println("Error: Invalid specify filepath")
		os.Exit(1)
	}

	f, err := os.Open(htmlPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	doc, err := html.Parse(f)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	var links []*Link
	findLinks(doc, &links)

	var wg sync.WaitGroup

	for i, a := range links {
		if a.IsURLTitle {
			wg.Add(1)

			go func() {
				FetchTitleText(a, i)
				defer wg.Done()
			}()

			time.Sleep(300 * time.Millisecond)
		}
	}

	wg.Wait()

	// output csv
	file, err := os.Create(outputPath)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	header := []string{
		"title", "url", "tags", "addedAt",
	}

	w := csv.NewWriter(file)
	defer w.Flush()

	w.Write(header)

	for _, a := range links {
		col := []string{a.Title, a.URL, a.Tags, a.TimeAdded}
		w.Write(col)
	}
}
