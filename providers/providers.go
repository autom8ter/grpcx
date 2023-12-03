package providers

// All is a struct that contains all providers
type All struct {
	// Logger is the registered logger provider
	Logger Logger
	// Database is the registered database provider
	Database Database
	// Cache is the registered cache provider
	Cache Cache
	// Stream is the registered stream provider
	Stream Stream
	// Metrics is the registered metrics provider
	Metrics Metrics
	// Tagger is the registered tagger provider
	Email Emailer
	// PaymentProcessor is the registered payment processor provider
	PaymentProcessor PaymentProcessor
	// Storage is the registered storage provider
	Storage Storage
}
