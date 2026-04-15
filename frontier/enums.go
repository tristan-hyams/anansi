package frontier

// Status represents the processing state of a FrontierURL.
type Status int

// Processing states for FrontierURL.
const (
	StatusPending Status = iota
	StatusVisited
	StatusError
)
