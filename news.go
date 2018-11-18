package aggregator

import (
	"github.com/mmcdole/gofeed"
	"log"
)

// News contains an item from RSS
type News struct {
	GUID        string
	Title       string
	Link        string
	Description string
	Published   string
	Categories  []string
}

// Parses RSS XML file and sends news to the specified channel
// Recommended to run this method in the goroutine because it requests XML from URL
func (agr *Aggregator) ParseRSS(rule ParsingRule, newsChan chan<- News) error {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(rule.URL)

	if err != nil {
		log.Printf("Error in xml decode process %s", err)
	}

	// Declare news here to allocate memory only once
	news := News{}

	for _, item := range feed.Items {
		news.GUID = item.GUID
		news.Title = item.Title
		news.Link = item.Link
		news.Description = item.Description
		news.Published = item.Published
		news.Categories = item.Categories

		newsChan <- news
	}

	return nil
}
