package payment

import (
	"context"
	"fmt"
)

// Gateway is the interface all payment gateways must implement
type Gateway interface {
	// CreateOrder creates a payment order/session with the gateway
	CreateOrder(ctx context.Context, req OrderRequest) (*OrderResponse, error)

	// VerifyPayment verifies that payment was successful
	VerifyPayment(ctx context.Context, verifyReq VerifyRequest) (*VerifyResponse, error)

	// RefundPayment refunds a payment
	RefundPayment(ctx context.Context, paymentID string, amount float64) error

	// GetGatewayName returns the name of the gateway
	GetGatewayName() string
}

// OrderRequest is the request to create an order
type OrderRequest struct {
	Amount        float64
	Currency      string
	ReceiptID     string
	Description   string
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
	Metadata      map[string]interface{}
}

// OrderResponse is returned when order is created
type OrderResponse struct {
	OrderID      string
	Amount       float64
	Currency     string
	Key          string // API key for client-side (Razorpay)
	ClientURL    string // URL for redirect-based payments (PhonePay)
	SignatureReq string // Additional signature if needed
}

// VerifyRequest is used to verify payment
type VerifyRequest struct {
	PaymentID string
	OrderID   string
	Signature string
	Amount    float64
	RawBody   []byte

	// Gateway-specific fields
	StripeIntentID string
	PhonePayTxnID  string
}

// VerifyResponse is returned after verification
type VerifyResponse struct {
	IsValid   bool
	PaymentID string
	OrderID   string
	Amount    float64
	Status    string
	Method    string
}

// New creates a new gateway instance
func New(gatewayType string, config map[string]string) (Gateway, error) {
	switch gatewayType {
	case "razorpay":
		return NewRazorpayGateway(config["key_id"], config["key_secret"])
	case "stripe":
		return NewStripeGateway(config["secret_key"], config["publishable_key"])
	case "phonepay":
		return NewPhonePayGateway(config["merchant_id"], config["merchant_key"])
	case "googlepay":
		return NewGooglePayGateway(config["merchant_id"])
	default:
		return nil, fmt.Errorf("unknown payment gateway: %s", gatewayType)
	}
}
