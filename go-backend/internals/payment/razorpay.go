package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// RazorpayGateway implements the Gateway interface for Razorpay
type RazorpayGateway struct {
	keyID     string
	keySecret string
}

// NewRazorpayGateway creates a new Razorpay gateway
func NewRazorpayGateway(keyID, keySecret string) (*RazorpayGateway, error) {
	if keyID == "" || keySecret == "" {
		return nil, fmt.Errorf("razorpay credentials not provided")
	}
	return &RazorpayGateway{
		keyID:     keyID,
		keySecret: keySecret,
	}, nil
}

// GetGatewayName returns the gateway name
func (r *RazorpayGateway) GetGatewayName() string {
	return "razorpay"
}

// CreateOrder creates a Razorpay order
func (r *RazorpayGateway) CreateOrder(ctx context.Context, req OrderRequest) (*OrderResponse, error) {
	// Convert amount to paise (smallest unit in India)
	amountInPaise := int64(req.Amount * 100)

	payload := url.Values{}
	payload.Set("amount", strconv.FormatInt(amountInPaise, 10))
	payload.Set("currency", req.Currency)
	payload.Set("receipt", req.ReceiptID)
	payload.Set("description", req.Description)

	// Customer details
	payload.Set("customer_notify", "1")
	if req.CustomerEmail != "" {
		payload.Set("customer[email]", req.CustomerEmail)
	}
	if req.CustomerPhone != "" {
		payload.Set("customer[phone]", req.CustomerPhone)
	}
	if req.CustomerName != "" {
		payload.Set("customer[name]", req.CustomerName)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.razorpay.com/v1/orders",
		strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add auth header
	httpReq.SetBasicAuth(r.keyID, r.keySecret)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Razorpay API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.Unmarshal(body, &errResp)
		return nil, fmt.Errorf("razorpay error: %v", errResp)
	}

	var orderResp map[string]interface{}
	if err := json.Unmarshal(body, &orderResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &OrderResponse{
		OrderID:  orderResp["id"].(string),
		Amount:   req.Amount,
		Currency: req.Currency,
		Key:      r.keyID,
	}, nil
}

// VerifyPayment verifies a Razorpay payment signature
func (r *RazorpayGateway) VerifyPayment(ctx context.Context, verifyReq VerifyRequest) (*VerifyResponse, error) {
	// Construct the signature input
	signatureInput := verifyReq.OrderID + "|" + verifyReq.PaymentID

	// Verify signature
	h := hmac.New(sha256.New, []byte(r.keySecret))
	h.Write([]byte(signatureInput))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(expectedSignature), []byte(verifyReq.Signature)) {
		return nil, fmt.Errorf("invalid payment signature")
	}

	// Fetch payment details from Razorpay to verify amount
	httpReq, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://api.razorpay.com/v1/payments/%s", verifyReq.PaymentID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(r.keyID, r.keySecret)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var paymentResp map[string]interface{}
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Verify amount (Razorpay stores in paise)
	amountInPaise := int64(verifyReq.Amount * 100)
	if int64(paymentResp["amount"].(float64)) != amountInPaise {
		return nil, fmt.Errorf("amount mismatch: expected %d, got %d", amountInPaise, int64(paymentResp["amount"].(float64)))
	}

	return &VerifyResponse{
		IsValid:   true,
		PaymentID: verifyReq.PaymentID,
		OrderID:   verifyReq.OrderID,
		Amount:    verifyReq.Amount,
		Status:    paymentResp["status"].(string),
		Method:    paymentResp["method"].(string),
	}, nil
}

// RefundPayment refunds a Razorpay payment
func (r *RazorpayGateway) RefundPayment(ctx context.Context, paymentID string, amount float64) error {
	amountInPaise := int64(amount * 100)

	payload := url.Values{}
	payload.Set("amount", strconv.FormatInt(amountInPaise, 10))

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("https://api.razorpay.com/v1/payments/%s/refund", paymentID),
		strings.NewReader(payload.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(r.keyID, r.keySecret)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call Razorpay API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("razorpay refund failed: %s", string(body))
	}

	return nil
}
