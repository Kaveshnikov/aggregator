//TODO: refactor with SQL builder or efficient ORM

package aggregator

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// Unfortunately guid is not unique and an optional field, so it cannot be used as the identifier
// http://www.xn--8ws00zhy3a.com/blog/2006/08/rss-dup-detection
// https://cyber.harvard.edu/rss/rss.html#ltguidgtSubelementOfLtitemgt
// To eliminate duplications the link field will be used as an unique one, despite it is optional too.
const (
	recordSchema = `
		CREATE TABLE IF NOT EXISTS "record" (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
    	"guid" VARCHAR(64),
    	"title" TEXT,
    	"link" VARCHAR(64) UNIQUE,
    	"description" TEXT,
    	"published" DATE
	);`

	// So as a feed item can have several categories we need to come to 1NF
	categorySchema = `
		CREATE TABLE IF NOT EXISTS "category" (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"name" VARCHAR(64) NOT NULL UNIQUE
	);`

	categoryToRecordSchema = `
		CREATE TABLE IF NOT EXISTS "categoryToRecord" (
		"recordId" INTEGER NOT NULL,
		"categoryId" INTEGER NOT NULL,
		
		FOREIGN KEY(recordId) REFERENCES record(id) 
		ON DELETE CASCADE ON UPDATE NO ACTION,
		
		FOREIGN KEY(categoryId) REFERENCES category(id)
		ON DELETE CASCADE ON UPDATE NO ACTION
	);`

	// Index is necessary to improve user search requests
	indexSchema = `
		CREATE INDEX IF NOT EXISTS "titleIndex"
		ON record(title)
	;`

	recordInsertQ = `
		INSERT OR IGNORE INTO record(guid, title, link, description, published) 
		VALUES(?,?,?,?,?)
	`

	// It is a template, not a finished query, because the values quantity differs
	categoryToRecordInsert = `
		INSERT OR IGNORE INTO categoryToRecord(recordId, categoryId)
		VALUES
	`

	// Cannot insert all values at once, because some of them can already be in DB
	categoryInsertQ = `
		INSERT OR IGNORE INTO category(name) VALUES(?)
	`

	// Template for select query from category table
	categorySelect = `
		SELECT id FROM category WHERE 
	`

	recordSelectQ = `
		SELECT id FROM record WHERE link=?
	`

	recordFindQ = `
		SELECT 
			record.id,
			record.guid,
			record.title,
			record.link,
			record.description,
			record.published,
			category.name as category
		FROM record
			INNER JOIN categoryToRecord as c2r ON c2r.recordId=record.id
			INNER JOIN category ON c2r.categoryId=category.id 
		WHERE record.title LIKE '%' || ? || '%'
		ORDER BY record.id ASC
	`
)

func initDataBase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	schemas := []string{recordSchema, categorySchema, categoryToRecordSchema, indexSchema}

	for _, schema := range schemas {
		stmt, err := db.Prepare(schema)
		if err != nil {
			return nil, err
		}

		_, err = stmt.Exec()
		if err != nil {
			return nil, err
		}

		stmt.Close()
	}

	return db, nil
}

func execQueryInTransaction(transaction *sql.Tx, query string, values []interface{}) error {
	stmt, err := transaction.Prepare(query)
	if err != nil {
		err = fmt.Errorf("prepare sql record insert query error: %s", err)

		transactionErr := transaction.Rollback()
		if transactionErr != nil {
			panic(fmt.Errorf(
				"could not rollback the transaction: %s. Rollback reason %s", transactionErr, err))
		}

		return err
	}

	_, err = stmt.Exec(values...)
	if err != nil {
		err = fmt.Errorf("error when saving news in database: %s", err)

		transactionErr := transaction.Rollback()
		if transactionErr != nil {
			panic(fmt.Errorf(
				"could not rollback the transaction: %s. Rollback reason %s", transactionErr, err))
		}

		return err
	}

	return nil
}

func getRowsT(transaction *sql.Tx, query string, values []interface{}) (rows *sql.Rows, err error) {
	rows, err = transaction.Query(query, values...)

	if err != nil {
		log.Printf("select query error: %s", err)

		transactionErr := transaction.Rollback()
		if transactionErr != nil {
			panic(fmt.Errorf(
				"could not rollback the transaction: %s. Rollback reason %s", transactionErr, err))
		}

		return
	}

	return
}

func buildAbstractQ(quantity int, template, valuesTemplate, separator string) string {
	tmp := make([]string, 0, quantity)

	for i := 0; i < quantity; i++ {
		tmp = append(tmp, valuesTemplate)
	}

	return template + strings.Join(tmp, separator)
}

func buildCategorySelectQ(categoriesQuantity int) string {

	return buildAbstractQ(
		categoriesQuantity,
		categorySelect,
		"name=?",
		", or ")
}

func buildCategoryToRecordInsertQ(rowsQuantity int) string {

	return buildAbstractQ(
		rowsQuantity,
		categoryToRecordInsert,
		"(?, ?)",
		",")
}

func insertRecord(transaction *sql.Tx, news *News) (err error) {
	err = execQueryInTransaction(
		transaction,
		recordInsertQ,
		[]interface{}{news.GUID, news.Title, news.Link, news.Description, news.Published})

	return
}

func insertCategories(transaction *sql.Tx, news *News) (err error) {
	value := make([]interface{}, 1, 1)
	for _, category := range news.Categories {
		value[0] = category
		err = execQueryInTransaction(transaction, categoryInsertQ, value)
		if err != nil {
			return
		}
	}

	return
}

func insertCategoryToRecord(transaction *sql.Tx, news *News) (err error) {
	value := []interface{}{news.Link}
	rows, err := getRowsT(transaction, recordSelectQ, value)
	if err != nil {
		return
	}

	var recordId int
	// Need only one categoryId
	rows.Next()
	err = rows.Scan(&recordId)
	if err != nil {
		err = fmt.Errorf("error when scan select result: %s", err)

		transactionErr := transaction.Rollback()
		if transactionErr != nil {
			panic(fmt.Errorf(
				"could not rollback the transaction: %s. Rollback reason %s", transactionErr, err))
		}

		return
	}

	categories := make([]interface{}, 0, len(news.Categories))
	for _, category := range news.Categories {
		categories = append(categories, category)
	}

	rows, err = getRowsT(
		transaction,
		buildCategorySelectQ(len(news.Categories)),
		categories)
	if err != nil {
		return
	}

	var categoryId int

	values := make([]interface{}, 0, len(news.Categories))
	for rows.Next() {
		err = rows.Scan(&categoryId)
		if err != nil {
			err = fmt.Errorf("error when scan select result: %s", err)

			transactionErr := transaction.Rollback()
			if transactionErr != nil {
				panic(fmt.Errorf(
					"could not rollback the transaction: %s. Rollback reason %s", transactionErr, err))
			}

			return
		}

		values = append(values, recordId)
		values = append(values, categoryId)
	}

	err = execQueryInTransaction(
		transaction,
		buildCategoryToRecordInsertQ(len(news.Categories)),
		values)

	return
}
