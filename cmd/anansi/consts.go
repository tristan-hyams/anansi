package main

import "time"

const (
	defaultWorkers  = 1
	defaultRate     = 1.0
	defaultMaxDepth = 1
	defaultTimeout  = 30 * time.Second
	defaultLogLevel = "info"

	errFmt         = "anansi: %v\n"
	exitCodeError  = 1
	exitCodeSIGINT = 130

	outputResultsFile = "crawl-results.md"
	outputJSONFile    = "crawl-results.json"
	outputErrorsFile  = "crawl-errors.md"
	summaryDurationRound = 100 * time.Millisecond
)
