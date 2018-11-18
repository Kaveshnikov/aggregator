package aggregator

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Aggregator is a main struct of the app.
// It collects rss items in DB according to the config
type Aggregator struct {
	db          *sql.DB
	rssSettings *Config
}

// SearchRecord allows search of a record in the database and returns list of news
func (agr Aggregator) SearchRecord(query string) ([]*News, error) {
	stmt, err := agr.db.Prepare(recordFindQ)
	if err != nil {
		return nil, fmt.Errorf("error when prepare sql: %s", err)
	}

	rows, err := stmt.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error when make sql query: %s", err)
	}

	var foundNews []*News
	var previousRecord *News
	var previousId int

	for rows.Next() {
		var id int
		var category string
		var record News

		if err := rows.Scan(&id, &record.GUID, &record.Title, &record.Link,
			&record.Description, &record.Published, &category); err != nil {
			return nil, fmt.Errorf("error when scan select result: %s", err)
		}

		if previousId == id {
			previousRecord.Categories = append(previousRecord.Categories, category)
		} else {
			foundNews = append(foundNews, &record)
			previousId = id
			previousRecord = &record
		}
	}

	return foundNews, nil
}

// Saves the record to DB
// The unique constraint disturbance is an ordinary case, but not the error
// News can contain large strings, so it is more efficient to send a pointer
// Use pointer is safe until only one goroutine executes CollectRecordsAndSave()
// TODO: make bulk insert
func (agr Aggregator) saveRecord(news *News) error {
	transaction, err := agr.db.Begin()
	if err != nil {
		return fmt.Errorf("error occured during transaction creating %s", err)
	}

	err = insertRecord(transaction, news)
	if err != nil {
		return err
	}

	err = insertCategories(transaction, news)
	if err != nil {
		return err
	}

	err = insertCategoryToRecord(transaction, news)
	if err != nil {
		return err
	}

	err = transaction.Commit()
	if err != nil {
		panic(fmt.Errorf("could not rollback the transaction: %s", err))
	}

	return nil
}

// Retrieves News items from the channel and save them to DB
// Must be run in goroutine
func (agr Aggregator) CollectRecordsAndSave(result <-chan News, ctx context.Context) {
	for {
		select {
		case news := <-result:
			err := agr.saveRecord(&news)

			if err != nil {
				log.Print(err)
				// It's better to miss some records then lose all so do not crush
			}
		case <-ctx.Done():
			return
		}
	}
}

// Starts endless loop for RSS feed refreshing and parsing
// Must be run in goroutine
func (agr Aggregator) GetRss(rule ParsingRule, RecordChan chan<- News, ctx context.Context) {
	ticker := time.NewTicker(time.Duration(rule.Timeout) * time.Second)
	defer ticker.Stop()

	agr.ParseRSS(rule, RecordChan)

	for {
		select {
		case <-ticker.C:
			agr.ParseRSS(rule, RecordChan)
		case <-ctx.Done():
			return
		}
	}
}

func (agr Aggregator) StartWork(ctx context.Context) {
	RecordChan := make(chan News, 200)

	go agr.CollectRecordsAndSave(RecordChan, ctx)

	for _, rssSetting := range agr.rssSettings.ParsingRules {
		go agr.GetRss(rssSetting, RecordChan, ctx)
	}
}

func InitAggregator(dbPath, configPath string) (*Aggregator, error) {
	db, err := initDataBase(dbPath)

	if err != nil {
		log.Printf("Error during database initialization: %s", err)
		return nil, err
	}

	rssSettings, err := ParseConfig(configPath)

	if err != nil {
		log.Printf("Error during config file parsing: %s", err)
		return nil, err
	}

	agr := &Aggregator{
		db:          db,
		rssSettings: rssSettings,
	}

	return agr, nil
}
