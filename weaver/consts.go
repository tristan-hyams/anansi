package weaver

import (
	"errors"
	"time"
)

// errMaxDepth indicates a URL was skipped because it exceeded the configured max depth.
var errMaxDepth = errors.New("max depth exceeded")

const (
	defaultUserAgent  = "Anansi"
	defaultBufferSize = 0 // use frontier's default

	logKeyURL   = "url"
	logKeyDepth = "depth"
	logKeyLinks = "links"

	summaryWidth         = 40
	summaryDurationRound = 100 * time.Millisecond
	pathSeparator        = "/"

	pct50  = 50
	pct95  = 95
	pct99  = 99
	pct100 = 100

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
