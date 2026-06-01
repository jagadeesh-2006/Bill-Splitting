package models

import "time"

// Payment statuses
const (
	PaymentStatusPending   = "pending"
	PaymentStatusSuccess   = "success"
	PaymentStatusFailed    = "failed"
	PaymentStatusCancelled = "cancelled"
)

// Payment gateway types
const (
	GatewayRazorpay  = "razorpay"
	GatewayStripe    = "stripe"
	GatewayPhonePay  = "phonepay"
	GatewayGooglePay = "googlepay"
)

// Payment represents a payment transaction
type Payment struct {
	ID               int        `json:"id"`
	SettlementID     int        `json:"settlementId"` // FK → settlements.id
	Amount           float64    `json:"amount"`
	Currency         string     `json:"currency"`         // USD, INR, etc
	Gateway          string     `json:"gateway"`          // razorpay, stripe, etc
	GatewayOrderID   string     `json:"gatewayOrderId"`   // Order ID from gateway (Razorpay: order_id)
	GatewayPaymentID string     `json:"gatewayPaymentId"` // Payment ID from gateway (Razorpay: payment_id)
	Status           string     `json:"status"`           // pending, success, failed, cancelled
	PaymentMethod    string     `json:"paymentMethod"`    // card, upi, wallet, etc
	Receipt          string     `json:"receipt"`          // Receipt ID from gateway
	ErrorMessage     string     `json:"errorMessage"`     // If payment failed
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	PaidAt           *time.Time `json:"paidAt"` // When payment was successful
}

// CreatePaymentRequest for initiating a payment
type CreatePaymentRequest struct {
	SettlementID int     `json:"settlementId" binding:"required"`
	Amount       float64 `json:"amount" binding:"required,gt=0"`
	Currency     string  `json:"currency" binding:"required"` // INR, USD, etc
	Gateway      string  `json:"gateway" binding:"required"`  // razorpay, stripe, etc
	Email        string  `json:"email" binding:"required,email"`
	Phone        string  `json:"phone" binding:"required"`
	Name         string  `json:"name" binding:"required"`
}

// VerifyPaymentRequest for verifying payment after gateway callback
type VerifyPaymentRequest struct {
	RazorpayOrderID   string `json:"razorpay_order_id"`
	RazorpayPaymentID string `json:"razorpay_payment_id"`
	RazorpaySignature string `json:"razorpay_signature"`

	// Stripe
	StripePaymentIntentID string `json:"stripe_payment_intent_id"`

	// PhonePay
	PhonePayTransactionID string `json:"phonepay_transaction_id"`
}

// PaymentResponse for API responses
type PaymentResponse struct {
	ID            int        `json:"id"`
	SettlementID  int        `json:"settlementId"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	Gateway       string     `json:"gateway"`
	Status        string     `json:"status"`
	PaymentMethod string     `json:"paymentMethod"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	PaidAt        *time.Time `json:"paidAt"`
}

// GatewayOrderResponse is returned when creating a payment order
type GatewayOrderResponse struct {
	OrderID   string  `json:"orderId"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Gateway   string  `json:"gateway"`
	Key       string  `json:"key,omitempty"`       // For Razorpay key
	Signature string  `json:"signature,omitempty"` // For signatures
	ClientURL string  `json:"clientUrl,omitempty"` // For redirect-based payments
}
