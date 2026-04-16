package main

import "time"

const (
	defaultWorkers  = 1
	defaultRate     = 1.0
	defaultMaxDepth = 1
	defaultTimeout  = 30 * time.Second
	defaultLogLevel  = "info"
	defaultMaxRetries = 2

	errFmt         = "anansi: %v\n"
	exitCodeError  = 1
	exitCodeSIGINT = 130

	// summaryDurationRound is duplicated in reporting/consts.go.
	// Could be consolidated into a shared consts package in the future.
	summaryDurationRound = 100 * time.Millisecond
)
