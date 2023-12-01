package providers

import (
	"context"

	"github.com/spf13/viper"
)

// Email is a struct for sending emails
type Email struct {
	From        string   `json:"from"`
	To          []string `json:"to"`
	Subject     string   `json:"subject"`
	Body        string   `json:"body"`
	ContentType string   `json:"content_type"`
}

type Emailer interface {
	SendEmail(ctx context.Context, email *Email) error
}

// EmailProvider is a function that returns an Emailer implementation
type EmailProvider func(ctx context.Context, config *viper.Viper) (Emailer, error)
