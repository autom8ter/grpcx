//go:generate mockgen -destination=./mocks/email.go -package=mocks . Email

package providers

import (
	"context"
)

// Email is a struct for sending emails
type Email struct {
	From        string   `json:"from"`
	To          []string `json:"to"`
	Subject     string   `json:"subject"`
	Body        string   `json:"body"`
	ContentType string   `json:"content_type"`
}

// Emailer is an interface that represents an email client
type Emailer interface {
	// SendEmail sends an email
	SendEmail(ctx context.Context, email *Email) error
}
