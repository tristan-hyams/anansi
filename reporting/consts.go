package reporting

import "time"

const (
	summaryWidth         = 40
	summaryDurationRound = 100 * time.Millisecond
	pathSeparator        = "/"

	pct50  = 50
	pct95  = 95
	pct99  = 99
	pct100 = 100

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
