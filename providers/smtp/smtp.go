package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"sync"

	"github.com/autom8ter/grpcx/providers"
)

// SmtpProvider is a struct for sending emails
type SmtpProvider struct {
	password string
	smtpHost string
	smtpPort int
	clients  map[string]*smtpClient
	mu       sync.RWMutex
}

// New returns a new smtp email provider
func New(password string, host string, port int) (providers.Emailer, error) {
	s := &SmtpProvider{
		password: password,
		smtpHost: host,
		smtpPort: port,
	}
	if s.password == "" {
		return nil, fmt.Errorf("no smtp password found")
	}
	if s.smtpHost == "" {
		return nil, fmt.Errorf("no smtp host found")
	}
	if s.smtpPort == 0 {
		return nil, fmt.Errorf("no smtp port found")
	}
	s.clients = make(map[string]*smtpClient)
	return s, nil
}

type smtpClient struct {
	from   string
	client *smtp.Client
	conn   *tls.Conn
	mu     sync.Mutex
}

func (s *smtpClient) SendEmail(ctx context.Context, email *providers.Email) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Set the sender and recipient first
	if err := s.client.Mail(email.From); err != nil {
		return fmt.Errorf("Failed to set sender: %v", err)
	}
	for _, recipient := range email.To {
		if err := s.client.Rcpt(recipient); err != nil {
			return fmt.Errorf("Failed to set recipient: %v", err)
		}
	}

	// Send the email body.
	wc, err := s.client.Data()
	if err != nil {
		return fmt.Errorf("Failed to send email body: %v", err)
	}
	defer wc.Close()
	// The recipient addresses must be separated by commas in the "To:" header
	toHeader := strings.Join(email.To, ", ")
	builder := strings.Builder{}
	builder.WriteString("To: " + toHeader + "\r\n")
	builder.WriteString("Subject: " + email.Subject + "\r\n")
	if email.ContentType == "" {
		email.ContentType = "text/plain"
	}
	builder.WriteString("Content-Type: " + email.ContentType + "; charset=UTF-8\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(email.Body + "\r\n")
	_, err = wc.Write([]byte(builder.String()))
	if err != nil {
		return fmt.Errorf("Failed to write message: %v", err)
	}
	return nil
}

// SendEmail sends an email to multiple recipients
func (s *SmtpProvider) SendEmail(ctx context.Context, email *providers.Email) error {
	s.mu.RLock()
	client, ok := s.clients[email.From]
	s.mu.RUnlock()
	if !ok {
		var err error
		client, err = s.createSMTPClient(email.From)
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.clients[email.From] = client
		s.mu.Unlock()
	}
	// Set the sender and recipient first
	if err := client.SendEmail(ctx, email); err != nil {
		s.mu.Lock()
		delete(s.clients, email.From)
		s.mu.Unlock()
		return err
	}
	return nil
}

func (s *SmtpProvider) createSMTPClient(from string) (*smtpClient, error) {
	// TLS config
	tlsConfig := &tls.Config{
		ServerName: s.smtpHost,
	}

	// Connect to the SMTP Server
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort), tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("TLS dial failed: %v", err)
	}

	client, err := smtp.NewClient(conn, s.smtpHost)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("Failed to create SMTP client: %v", err)
	}

	// Authenticate
	auth := smtp.PlainAuth("", from, s.password, s.smtpHost)
	if err = client.Auth(auth); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to authenticate: %v", err)
	}

	return &smtpClient{
		from:   from,
		client: client,
		conn:   conn,
		mu:     sync.Mutex{},
	}, nil
}
