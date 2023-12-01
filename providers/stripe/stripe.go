package stripe

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/viper"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/client"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/paymentintent"

	"github.com/autom8ter/grpcx/internal/utils"
	"github.com/autom8ter/grpcx/providers"
)

type StripePayments struct {
	client *client.API
}

// CreateCustomer creates a customer
func (s *StripePayments) CreateCustomer(ctx context.Context, cust *providers.Customer) (*providers.Customer, error) {
	params := &stripe.CustomerParams{
		Params:      stripe.Params{},
		Description: stripe.String(cust.Description),
		Email:       stripe.String(cust.Email),
		Name:        stripe.String(cust.Name),
		Phone:       stripe.String(cust.Phone),
	}
	if cust.Metadata != nil {
		for k, v := range cust.Metadata {
			params.AddMetadata(k, v)
		}
	}
	c, err := s.client.Customers.New(params)
	if err != nil {
		return nil, utils.WrapError(err, "failed to create customer")
	}
	cust.ID = c.ID
	cust.CreatedAt = time.Unix(c.Created, 0)
	return cust, nil
}

// GetCustomer gets a customer
func (s *StripePayments) GetCustomer(ctx context.Context, id string) (*providers.Customer, error) {
	c, err := s.client.Customers.Get(id, nil)
	if err != nil {
		return nil, utils.WrapError(err, "failed to get customer")
	}
	cust := &providers.Customer{
		ID:          c.ID,
		Name:        c.Name,
		Phone:       c.Phone,
		Email:       c.Email,
		Description: c.Description,
		Metadata:    c.Metadata,
		CreatedAt:   time.Unix(c.Created, 0),
	}
	if c.InvoiceSettings != nil && c.InvoiceSettings.DefaultPaymentMethod != nil {
		cust.DefaultCardID = c.InvoiceSettings.DefaultPaymentMethod.ID
	}
	return cust, nil
}

// UpdateCustomer updates a customer
func (s *StripePayments) UpdateCustomer(ctx context.Context, cust *providers.Customer) (*providers.Customer, error) {
	params := &stripe.CustomerParams{
		Params:      stripe.Params{},
		Description: stripe.String(cust.Description),
		Email:       stripe.String(cust.Email),
		Name:        stripe.String(cust.Name),
	}
	if cust.Phone != "" {
		params.Phone = stripe.String(cust.Phone)
	}
	if cust.DefaultCardID != "" {
		params.InvoiceSettings.DefaultPaymentMethod = stripe.String(cust.DefaultCardID)
	}
	if cust.Metadata != nil {
		for k, v := range cust.Metadata {
			params.AddMetadata(k, v)
		}
	}

	c, err := s.client.Customers.Update(cust.ID, params)
	if err != nil {
		return nil, utils.WrapError(err, "failed to update customer")
	}
	cust.ID = c.ID
	return cust, nil
}

// DeleteCustomer deletes a customer
func (s *StripePayments) DeleteCustomer(ctx context.Context, id string) error {
	_, err := customer.Del(id, nil)
	return utils.WrapError(err, "failed to delete customer")
}

// ListCustomers lists customers
func (s *StripePayments) ListCustomers(ctx context.Context, after string, limit int64) ([]*providers.Customer, error) {
	if limit == 0 {
		limit = 100
	}
	params := &stripe.CustomerListParams{
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(limit),
		},
	}
	if after != "" {
		params.StartingAfter = stripe.String(after)
	}
	iter := s.client.Customers.List(params)
	var customers []*providers.Customer
	for iter.Next() {
		c := iter.Customer()
		cust := &providers.Customer{
			ID:            c.ID,
			Name:          c.Name,
			Phone:         c.Phone,
			Email:         c.Email,
			Description:   c.Description,
			DefaultCardID: "",
			Metadata:      c.Metadata,
			CreatedAt:     time.Unix(c.Created, 0),
		}
		if c.InvoiceSettings != nil && c.InvoiceSettings.DefaultPaymentMethod != nil {
			cust.DefaultCardID = c.InvoiceSettings.DefaultPaymentMethod.ID
		}
		customers = append(customers, cust)
	}
	return customers, nil
}

// CreateCard creates a card for a customer
func (s *StripePayments) CreateCard(ctx context.Context, customerID string, token string) (*providers.Card, error) {
	attachParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	}
	pm, err := s.client.PaymentMethods.Attach(token, attachParams)
	if err != nil {
		return nil, utils.WrapError(err, "failed to attach card to customer")
	}
	cust, err := s.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, utils.WrapError(err, "failed to get customer")
	}
	return &providers.Card{
		ID:        pm.ID,
		Brand:     string(pm.Card.Brand),
		Last4:     pm.Card.Last4,
		ExpM:      pm.Card.ExpMonth,
		ExpY:      pm.Card.ExpYear,
		IsDefault: cust.DefaultCardID == pm.ID,
		CreatedAt: time.Unix(pm.Created, 0),
	}, nil
}

// ListCards lists a customer's cards
func (s *StripePayments) ListCards(ctx context.Context, customerID string) ([]*providers.Card, error) {
	cust, err := s.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	params := &stripe.PaymentMethodListParams{
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(100),
		},
	}
	iter := s.client.PaymentMethods.List(params)
	var cards []*providers.Card
	for iter.Next() {
		pm := iter.PaymentMethod()
		cards = append(cards, &providers.Card{
			ID:        pm.ID,
			Brand:     string(pm.Card.Brand),
			Last4:     pm.Card.Last4,
			ExpM:      pm.Card.ExpMonth,
			ExpY:      pm.Card.ExpYear,
			IsDefault: cust.DefaultCardID == pm.ID,
			CreatedAt: time.Unix(pm.Created, 0),
		})
	}
	return cards, nil
}

func (s *StripePayments) DeleteCard(ctx context.Context, customerID string, cardID string) error {
	detachParams := &stripe.PaymentMethodDetachParams{
		Params: stripe.Params{},
		Expand: nil,
	}
	_, err := s.client.PaymentMethods.Detach(cardID, detachParams)
	return err
}

// GetCard gets a card
func (s *StripePayments) GetCard(ctx context.Context, customerID string, cardID string) (*providers.Card, error) {
	pm, err := s.client.PaymentMethods.Get(cardID, nil)
	if err != nil {
		return nil, utils.WrapError(err, "failed to get card")
	}
	cust, err := s.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, utils.WrapError(err, "failed to get customer")
	}
	return &providers.Card{
		ID:        pm.ID,
		Brand:     string(pm.Card.Brand),
		Last4:     pm.Card.Last4,
		ExpM:      pm.Card.ExpMonth,
		ExpY:      pm.Card.ExpYear,
		IsDefault: cust.DefaultCardID == pm.ID,
		CreatedAt: time.Unix(pm.Created, 0),
	}, nil
}

// CreateCharge charges a customer
func (s *StripePayments) CreateCharge(ctx context.Context, charge *providers.Charge) (*providers.Charge, error) {
	params := &stripe.PaymentIntentParams{
		Params:             stripe.Params{},
		Amount:             stripe.Int64(charge.Amount),
		Confirm:            stripe.Bool(true),
		ConfirmationMethod: nil,
		Currency:           stripe.String(string(stripe.CurrencyUSD)),
		Customer:           stripe.String(charge.CustomerID),
		Description:        stripe.String(charge.Description),
		Metadata:           nil,
		PaymentMethod:      stripe.String(charge.CardID),
		CaptureMethod:      stripe.String(string(stripe.PaymentIntentCaptureMethodAutomatic)),
		Expand:             []*string{stripe.String("latest_charge")},
	}
	intent, err := paymentintent.New(params)
	if err != nil {
		return nil, utils.WrapError(err, "failed to charge customer")
	}
	if intent.LatestCharge == nil {
		return nil, fmt.Errorf("latest charge is nil")
	}
	charge.ID = intent.LatestCharge.ID
	charge.Status = string(intent.LatestCharge.Status)
	charge.CreatedAt = time.Unix(intent.LatestCharge.Created, 0)
	return charge, nil
}

// GetCharge gets a charge
func (s *StripePayments) GetCharge(ctx context.Context, id string) (*providers.Charge, error) {
	ch, err := s.client.Charges.Get(id, nil)
	if err != nil {
		return nil, utils.WrapError(err, "failed to get charge")
	}
	return &providers.Charge{
		ID:          ch.ID,
		CustomerID:  ch.Customer.ID,
		CardID:      ch.PaymentMethod,
		Amount:      ch.Amount,
		Description: ch.Description,
		Status:      string(ch.Status),
		CreatedAt:   time.Unix(ch.Created, 0),
	}, nil
}

// ListCharges lists a customer's charges
func (s *StripePayments) ListCharges(ctx context.Context, customerID string, starting_after string, limit int64) ([]*providers.Charge, error) {
	if limit == 0 {
		limit = 100
	}
	params := &stripe.ChargeListParams{
		Customer: stripe.String(customerID),
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(limit),
		},
	}
	if starting_after != "" {
		params.StartingAfter = stripe.String(starting_after)
	}
	iter := s.client.Charges.List(params)
	var charges []*providers.Charge
	for iter.Next() {
		ch := iter.Charge()
		charges = append(charges, &providers.Charge{
			ID:          ch.ID,
			CustomerID:  ch.Customer.ID,
			CardID:      ch.PaymentMethod,
			Amount:      ch.Amount,
			Description: ch.Description,
			Status:      string(ch.Status),
		})
	}
	return charges, nil
}

// Provider is a function that returns a PaymentProcessor implementation
// requires payment_processing.stripe.secret_key
func Provider(ctx context.Context, config *viper.Viper) (providers.PaymentProcessor, error) {
	sc := &client.API{}
	key := config.GetString("payment_processing.stripe.secret_key")
	if key == "" {
		return nil, fmt.Errorf("no stripe secret key found (payment_processing.stripe.secret_key)")
	}
	sc.Init(key, nil)
	return &StripePayments{client: sc}, nil
}
