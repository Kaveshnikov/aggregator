# aggregator
A news aggregator. Collects news from specified RSS feeds and provides API (in development) for searching.

# Parsing Rules

rss - config root, contains array of rules

rule object:

    url: RSS feed URL
    timeout: RSS feed refreshing timeout in seconds
    ItemTags: an array of RSS Item child tags to collect. Tags Title and Link are always collected

Example:

    {
        "rss": [
            {
              "url": "https://lenta.ru/rss/news",
              "timeout": 60,
              "itemTags": ["guid", "description", "published", "category"]
            },
            {
              "url": "https://abcnews.go.com/abcnews/topstories",
              "timeout": 60,
              "itemTags": ["category"]
            }
        ]
    }