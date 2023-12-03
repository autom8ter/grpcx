package providers

import (
	"context"
	"time"
)

// Charge is a struct for charging a customer
type Charge struct {
	ID          string    `json:"id"`
	CustomerID  string    `json:"customer_id"`
	CardID      string    `json:"card_id"`
	Amount      int64     `json:"amount"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Customer is a struct for creating a customer
type Customer struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Phone         string            `json:"phone"`
	Email         string            `json:"email"`
	Description   string            `json:"description"`
	DefaultCardID string            `json:"default_card_id"`
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Card is a struct for a customer's card
type Card struct {
	ID        string    `json:"id"`
	Brand     string    `json:"brand"`
	Last4     string    `json:"last4"`
	ExpM      int64     `json:"exp_month"`
	ExpY      int64     `json:"exp_year"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

// PaymentProcessor is a function that returns a PaymentProcessor implementation
type PaymentProcessor interface {
	// CreateCustomer creates a customer
	CreateCustomer(ctx context.Context, charge *Customer) (*Customer, error)
	// UpdateCustomer updates a customer
	UpdateCustomer(ctx context.Context, charge *Customer) (*Customer, error)
	// GetCustomer gets a customer
	GetCustomer(ctx context.Context, id string) (*Customer, error)
	// DeleteCustomer deletes a customer
	DeleteCustomer(ctx context.Context, id string) error
	// ListCustomers lists customers
	ListCustomers(ctx context.Context, after string, limit int64) ([]*Customer, error)
	// CreateCard creates a card for a customer
	CreateCard(ctx context.Context, customerID string, token string) (*Card, error)
	// GetCard gets a card for a customer
	GetCard(ctx context.Context, customerID string, cardID string) (*Card, error)
	// DeleteCard deletes a card for a customer
	DeleteCard(ctx context.Context, customerID string, cardID string) error
	// ListCards lists cards for a customer
	ListCards(ctx context.Context, customerID string) ([]*Card, error)
	// CreateCharge creates a charge
	CreateCharge(ctx context.Context, charge *Charge) (*Charge, error)
	// GetCharge gets a charge
	GetCharge(ctx context.Context, id string) (*Charge, error)
	// ListCharges lists charges
	ListCharges(ctx context.Context, customerID, after string, limit int64) ([]*Charge, error)
}
