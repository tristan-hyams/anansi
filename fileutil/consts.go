package fileutil

import "time"

const (
	summaryWidth         = 40
	summaryDurationRound = 100 * time.Millisecond
	pathSeparator        = "/"

	outputResultsFile = "crawl-results.md"
	outputJSONFile    = "crawl-results.json"
	outputErrorsFile  = "crawl-errors.md"

	//nolint:revive // ASCII art banner for the crawl summary.
	banner = `
    /\  .-"""-.  /\
   //\\/  ,,,  \//\\
   |/\| ,;;;;;, |/\|
   //\\\;-"""-;///\\
  //  \/   .   \/  \\
 (| ,-_| \ | / |_-, |)
   //` + "`" + `__\.-.-./__` + "`" + `\\
  // /.-( \;/ )-.\ \\
 (\ ,-_/  |_|  \_-, /)
  \/    '-._._.-'    \/
   \  /  |   |  \  /
    \/   |   |   \/
`
)
