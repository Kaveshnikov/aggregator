package aggregator

import (
	"encoding/json"
	"github.com/mmcdole/gofeed"
	"io/ioutil"
	"os"
)

type rawParsingRule struct {
	Timeout  int      `json:"timeout"`
	URL      string   `json:"url"`
	ItemTags []string `json:"itemTags"` // Tags are case sensitive because of XML
	mapping  map[string]struct{}
}

func (rawRule rawParsingRule) SetMapping() {
	rawRule.mapping = make(map[string]struct{}, len(rawRule.ItemTags))

	for _, tag := range rawRule.ItemTags {
		rawRule.mapping[tag] = struct{}{}
	}
}

func (rawRule rawParsingRule) Contains(tag string) (contains bool) {
	if rawRule.mapping == nil {
		rawRule.SetMapping()
	}

	_, contains = rawRule.mapping[tag]
	return
}

// rawConfig is necessary for deserialization
type rawConfig struct {
	RawRules []rawParsingRule `json:"rss"`
}

// ParsingRule contains a rule for parsing items from one RSS feed
type ParsingRule struct {
	// Required fields
	Timeout int    // Timeout in seconds to refresh RSS feed
	URL     string // Feed URL

	// Flags to collect optional fields (feed item fields)
	Categories  bool
	Description bool
	GUID        bool
	Link        bool
	PubDate     bool
	Title       bool
}

// Declare not exported constructor because it uses internal structures and is made for internal use only
func newParsingRule(raw rawParsingRule) (parsingRule ParsingRule) {
	// Reflection can be used here, but direct field call is more efficient
	if raw.Contains("categories") {
		parsingRule.Categories = true
	}

	if raw.Contains("description") {
		parsingRule.Description = true
	}

	if raw.Contains("guid") {
		parsingRule.GUID = true
	}

	if raw.Contains("pubDate") {
		parsingRule.PubDate = true
	}

	parsingRule.URL = raw.URL
	parsingRule.Timeout = raw.Timeout
	return
}

// Checks if News item satisfies the rule
func (rule ParsingRule) Apply(item *gofeed.Item) (news News) {
	// Reflection can be used here, but direct field call is more efficient
	if rule.Categories {
		news.Categories = item.Categories
	}

	if rule.Description {
		news.Description = item.Description
	}

	if rule.GUID {
		news.GUID = item.GUID
	}

	if rule.PubDate {
		news.Published = item.Published
	}

	news.Link = item.Link
	news.Title = item.Title

	return
}

// Config contains parsing rules for feeds and Aggregator settings
type Config struct {
	ParsingRules []ParsingRule
}

func ParseConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	rawConfigData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var _config rawConfig
	if err := json.Unmarshal(rawConfigData, &_config); err != nil {
		return nil, err
	}

	parsingRules := make([]ParsingRule, 0, len(_config.RawRules))

	for _, rawRule := range _config.RawRules {
		parsingRules = append(parsingRules, newParsingRule(rawRule))
	}

	config := Config{parsingRules}

	return &config, nil
}
