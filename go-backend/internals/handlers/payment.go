package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jagadeesh-2006/Bill-Splitting/go-backend/internals/models"
	"github.com/jagadeesh-2006/Bill-Splitting/go-backend/internals/payment"
)

// InitiatePayment creates a payment order
// POST /api/payments/initiate
func InitiatePayment(c *gin.Context) {
	var req models.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate settlement exists
	var settlementID int
	var fromMemberID, toMemberID int
	var settlementAmount float64
	var groupID int

	err := db.QueryRow(
		context.Background(),
		"SELECT id, from_member, to_member, amount, group_id FROM settlements WHERE id = $1",
		req.SettlementID,
	).Scan(&settlementID, &fromMemberID, &toMemberID, &settlementAmount, &groupID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settlement not found"})
		return
	}

	// Validate amount matches settlement
	if req.Amount != settlementAmount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must match settlement amount"})
		return
	}

	// Initialize payment gateway
	gatewayConfig := map[string]string{
		"key_id":          os.Getenv("RAZORPAY_KEY_ID"),
		"key_secret":      os.Getenv("RAZORPAY_KEY_SECRET"),
		"secret_key":      os.Getenv("STRIPE_SECRET_KEY"),
		"publishable_key": os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		"merchant_id":     os.Getenv("PHONEPAY_MERCHANT_ID"),
		"merchant_key":    os.Getenv("PHONEPAY_MERCHANT_KEY"),
	}

	gateway, err := payment.New(req.Gateway, gatewayConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment gateway"})
		return
	}

	// Create order with the gateway
	orderReq := payment.OrderRequest{
		Amount:        req.Amount,
		Currency:      req.Currency,
		ReceiptID:     fmt.Sprintf("settlement_%d_%d", groupID, req.SettlementID),
		Description:   fmt.Sprintf("Payment for expense settlement between members"),
		CustomerName:  req.Name,
		CustomerEmail: req.Email,
		CustomerPhone: req.Phone,
		Metadata: map[string]interface{}{
			"settlement_id": req.SettlementID,
			"group_id":      groupID,
		},
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	orderResp, err := gateway.CreateOrder(ctx, orderReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create order: %v", err)})
		return
	}

	// Save payment record to database
	var paymentID int
	err = db.QueryRow(
		context.Background(),
		`INSERT INTO payments (settlement_id, amount, currency, gateway, gateway_order_id, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		 RETURNING id`,
		req.SettlementID, req.Amount, req.Currency, req.Gateway, orderResp.OrderID, models.PaymentStatusPending,
	).Scan(&paymentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save payment"})
		return
	}

	// Return order details to client
	c.JSON(http.StatusOK, gin.H{
		"paymentId": paymentID,
		"orderId":   orderResp.OrderID,
		"amount":    orderResp.Amount,
		"currency":  orderResp.Currency,
		"gateway":   req.Gateway,
		"key":       orderResp.Key,
		"clientUrl": orderResp.ClientURL,
	})
}

// VerifyPayment verifies payment after gateway callback
// POST /api/payments/verify
func VerifyPayment(c *gin.Context) {
	var req models.VerifyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var err error

	// Determine which gateway based on provided fields
	var gateway string
	var paymentID, orderID string

	if req.RazorpayOrderID != "" {
		gateway = models.GatewayRazorpay
		orderID = req.RazorpayOrderID
		paymentID = req.RazorpayPaymentID
	} else if req.StripePaymentIntentID != "" {
		gateway = models.GatewayStripe
		paymentID = req.StripePaymentIntentID
	} else if req.PhonePayTransactionID != "" {
		gateway = models.GatewayPhonePay
		paymentID = req.PhonePayTransactionID
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No payment data provided"})
		return
	}

	// Get payment record from database
	var dbPaymentID int
	var settlementID int
	var amount float64
	var status string

	err = db.QueryRow(
		context.Background(),
		`SELECT id, settlement_id, amount, status FROM payments 
		 WHERE gateway = $1 AND gateway_order_id = $2`,
		gateway, orderID,
	).Scan(&dbPaymentID, &settlementID, &amount, &status)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	if status != models.PaymentStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment already processed"})
		return
	}

	// Initialize gateway
	gatewayConfig := map[string]string{
		"key_id":          os.Getenv("RAZORPAY_KEY_ID"),
		"key_secret":      os.Getenv("RAZORPAY_KEY_SECRET"),
		"secret_key":      os.Getenv("STRIPE_SECRET_KEY"),
		"publishable_key": os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		"merchant_id":     os.Getenv("PHONEPAY_MERCHANT_ID"),
		"merchant_key":    os.Getenv("PHONEPAY_MERCHANT_KEY"),
	}

	gw, err := payment.New(gateway, gatewayConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid gateway"})
		return
	}

	// Verify payment
	// Read request body for signature verification
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	verifyReq := payment.VerifyRequest{
		PaymentID: paymentID,
		OrderID:   orderID,
		Signature: req.RazorpaySignature,
		Amount:    amount,
		RawBody:   bodyBytes,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	verifyResp, err := gw.VerifyPayment(ctx, verifyReq)
	if err != nil {
		// Update payment status to failed
		db.Exec(
			context.Background(),
			`UPDATE payments SET status = $1, error_message = $2, updated_at = NOW() 
			 WHERE id = $3`,
			models.PaymentStatusFailed, err.Error(), dbPaymentID,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Payment verification failed: %v", err)})
		return
	}

	// Update payment status to success
	now := time.Now()
	_, err = db.Exec(
		context.Background(),
		`UPDATE payments 
		 SET status = $1, gateway_payment_id = $2, payment_method = $3, 
		     paid_at = $4, updated_at = NOW()
		 WHERE id = $5`,
		models.PaymentStatusSuccess, verifyResp.PaymentID, verifyResp.Method, now, dbPaymentID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment"})
		return
	}

	// Update settlement status to paid (mark settlement as completed)
	_, err = db.Exec(
		context.Background(),
		`UPDATE settlements SET paid_at = NOW() WHERE id = $1`,
		settlementID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settlement"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Payment verified successfully",
		"paymentId": dbPaymentID,
		"status":    models.PaymentStatusSuccess,
	})
}

// GetPaymentsByGroup gets all payments for a group
// GET /api/groups/:groupId/payments
func GetPaymentsByGroup(c *gin.Context) {
	groupIDStr := c.Param("groupId")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	rows, err := db.Query(
		context.Background(),
		`SELECT p.id, p.settlement_id, p.amount, p.currency, p.gateway, 
		        p.status, p.payment_method, p.created_at, p.updated_at, p.paid_at
		 FROM payments p
		 JOIN settlements s ON p.settlement_id = s.id
		 WHERE s.group_id = $1
		 ORDER BY p.created_at DESC`,
		groupID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
		return
	}
	defer rows.Close()

	var payments []models.PaymentResponse
	for rows.Next() {
		var p models.PaymentResponse
		err := rows.Scan(&p.ID, &p.SettlementID, &p.Amount, &p.Currency, &p.Gateway,
			&p.Status, &p.PaymentMethod, &p.CreatedAt, &p.UpdatedAt, &p.PaidAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse payments"})
			return
		}
		payments = append(payments, p)
	}

	c.JSON(http.StatusOK, gin.H{"payments": payments})
}

// GetPaymentByID gets a single payment by ID
// GET /api/payments/:paymentId
func GetPaymentByID(c *gin.Context) {
	paymentIDStr := c.Param("paymentId")
	paymentID, err := strconv.Atoi(paymentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment ID"})
		return
	}

	var p models.PaymentResponse
	err = db.QueryRow(
		context.Background(),
		`SELECT id, settlement_id, amount, currency, gateway, status, payment_method, 
		        created_at, updated_at, paid_at
		 FROM payments WHERE id = $1`,
		paymentID,
	).Scan(&p.ID, &p.SettlementID, &p.Amount, &p.Currency, &p.Gateway,
		&p.Status, &p.PaymentMethod, &p.CreatedAt, &p.UpdatedAt, &p.PaidAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	c.JSON(http.StatusOK, p)
}

// RefundPayment refunds a successful payment
// POST /api/payments/:paymentId/refund
func RefundPayment(c *gin.Context) {
	paymentIDStr := c.Param("paymentId")
	paymentID, err := strconv.Atoi(paymentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment ID"})
		return
	}

	// Get payment details
	var gateway, gatewayPaymentID, status string
	var amount float64

	err = db.QueryRow(
		context.Background(),
		`SELECT gateway, gateway_payment_id, status, amount FROM payments WHERE id = $1`,
		paymentID,
	).Scan(&gateway, &gatewayPaymentID, &status, &amount)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	if status != models.PaymentStatusSuccess {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only successful payments can be refunded"})
		return
	}

	// Initialize gateway
	gatewayConfig := map[string]string{
		"key_id":          os.Getenv("RAZORPAY_KEY_ID"),
		"key_secret":      os.Getenv("RAZORPAY_KEY_SECRET"),
		"secret_key":      os.Getenv("STRIPE_SECRET_KEY"),
		"publishable_key": os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		"merchant_id":     os.Getenv("PHONEPAY_MERCHANT_ID"),
		"merchant_key":    os.Getenv("PHONEPAY_MERCHANT_KEY"),
	}

	gw, err := payment.New(gateway, gatewayConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid gateway"})
		return
	}

	// Process refund
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	err = gw.RefundPayment(ctx, gatewayPaymentID, amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Refund failed: %v", err)})
		return
	}

	// Update payment status
	_, err = db.Exec(
		context.Background(),
		`UPDATE payments SET status = $1, updated_at = NOW() WHERE id = $2`,
		"refunded", paymentID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Payment refunded successfully",
		"paymentId": paymentID,
	})
}
