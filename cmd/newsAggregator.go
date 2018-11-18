package main

import (
	"context"
	"flag"
	"github.com/julienschmidt/httprouter"
	"github.com/kaveshnikov/agregator"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var (
		dbPath     string
		configPath string
		debug      bool
	)

	flag.BoolVar(&debug, "debug", false, "Debug mode")
	flag.StringVar(&dbPath, "Database path", "db.sqlite", "Aggregator db path")
	flag.StringVar(&configPath, "Config parse path", "config.json",
		"Path to config with rules for parsing")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	agr, err := aggregator.InitAggregator(dbPath, configPath)

	if err != nil {
		log.Fatal("Error during Aggregator initializing.")
	}

	go agr.StartWork(ctx)

	router := httprouter.New()
	router.HandlerFunc("GET", "/", agr.IndexHandler)
	router.HandlerFunc("POST", "/search", agr.HandleSearch)
	go log.Print(http.ListenAndServe(":8080", router))

	select {
	case <-ctx.Done():
		cancel()
	case <-sigs:
		cancel()
	}
}
