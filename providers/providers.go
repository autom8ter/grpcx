package providers

// All is a struct that contains all providers
type All struct {
	Logger   Logger
	Database Database
	Cache    Cache
	Stream   Stream
	Metrics  Metrics
}
