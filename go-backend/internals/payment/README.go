package payment

/*

Payment Gateway Abstraction Layer
==================================

This package provides an abstraction layer for integrating multiple payment gateways
into the Bill Splitting application. The design allows easy switching between different
payment providers without changing business logic.

ARCHITECTURE
============

The payment system uses the Gateway interface pattern:

    ┌─────────────────┐
    │   Handler       │
    │ (handlers/)     │
    └────────┬────────┘
             │
    ┌────────▼─────────────────────┐
    │   Gateway Interface          │
    │  (payment/gateway.go)         │
    └────────┬─────────────────────┘
             │
      ┌──────┴──────┬──────────┬──────────┐
      │             │          │          │
   ┌──▼──┐      ┌──▼──┐   ┌──▼──┐   ┌──▼──┐
   │  RZ  │      │ STR  │   │ PHP │   │ GPA │
   │      │      │      │   │     │   │     │
   └──────┘      └──────┘   └─────┘   └─────┘

   RZ  = Razorpay (✓ Implemented)
   STR = Stripe (⚠ Stub ready)
   PHP = PhonePay (⚠ Stub ready)
   GPA = Google Pay (⚠ Stub ready)

FLOW
====

1. USER INITIATES PAYMENT
   └─ User clicks "Pay Now" on a settlement
   └─ Payment page loads with settlement details

2. PAYMENT INITIALIZATION
   └─ Frontend collects: email, name, phone, gateway choice
   └─ POST /api/payments/initiate
   └─ Backend creates order with chosen gateway
   └─ Returns order ID and gateway-specific data

3. GATEWAY CHECKOUT
   └─ Frontend passes control to gateway (Razorpay modal, etc.)
   └─ User completes payment with gateway
   └─ Gateway returns payment confirmation

4. PAYMENT VERIFICATION
   └─ Frontend receives payment confirmation
   └─ POST /api/payments/verify with gateway response
   └─ Backend verifies signature and amount
   └─ Updates settlement as "paid"

5. CONFIRMATION
   └─ User sees success message
   └─ Settlement marked as paid in dashboard


GATEWAY INTERFACE
=================

All gateways must implement:

    type Gateway interface {
        CreateOrder(ctx context.Context, req OrderRequest) (*OrderResponse, error)
        VerifyPayment(ctx context.Context, verifyReq VerifyRequest) (*VerifyResponse, error)
        RefundPayment(ctx context.Context, paymentID string, amount float64) error
        GetGatewayName() string
    }

CreateOrder: Initiates a payment transaction
  - Input: Amount, currency, customer details, metadata
  - Output: Order ID, key for frontend, any additional data
  - Example: Razorpay creates an order and returns order_id

VerifyPayment: Verifies a payment was successful
  - Input: Payment ID, order ID, signature from gateway
  - Output: Verification status, payment details
  - Example: Razorpay verifies HMAC-SHA256 signature

RefundPayment: Refunds a completed payment
  - Input: Payment ID, amount to refund
  - Output: Success/error
  - Example: Razorpay processes partial or full refund

GetGatewayName: Returns gateway identifier
  - Used for logging and identifying gateway type in DB


IMPLEMENTATION EXAMPLE: RAZORPAY
================================

1. CreateOrder:
   - Create HTTPS request to https://api.razorpay.com/v1/orders
   - Set amount in paise (1 INR = 100 paise)
   - Send customer details, receipt ID
   - Return order_id in OrderResponse

2. VerifyPayment:
   - Calculate HMAC-SHA256(order_id|payment_id, secret_key)
   - Compare with provided signature
   - Fetch payment details from Razorpay API
   - Verify amount matches and status is "captured"
   - Return VerifyResponse with payment details

3. RefundPayment:
   - Create HTTPS request to refund endpoint
   - Specify amount in paise
   - Return success if status 200

See: razorpay.go for complete example


DATABASE SCHEMA
===============

    CREATE TABLE payments (
        id                  SERIAL PRIMARY KEY,
        settlement_id       INT NOT NULL,          -- Foreign key to settlements
        amount              NUMERIC(10,2) NOT NULL,-- Payment amount
        currency            TEXT NOT NULL,         -- INR, USD, etc
        gateway             TEXT NOT NULL,         -- razorpay, stripe, etc
        gateway_order_id    TEXT NOT NULL,         -- Order ID from gateway
        gateway_payment_id  TEXT,                  -- Payment ID from gateway
        status              TEXT NOT NULL,         -- pending, success, failed, cancelled, refunded
        payment_method      TEXT,                  -- card, upi, wallet, etc
        error_message       TEXT,                  -- If payment failed
        created_at          TIMESTAMPTZ,
        updated_at          TIMESTAMPTZ,
        paid_at             TIMESTAMPTZ,           -- When payment completed
        UNIQUE(gateway, gateway_order_id)          -- Prevent duplicate orders
    );


PAYMENT LIFECYCLE
=================

    Settlement Created
         │
         ▼
    User Clicks "Pay Now"
         │
         ▼
    POST /api/payments/initiate
         │
         ├─ Validate settlement
         ├─ Create gateway instance
         ├─ Call gateway.CreateOrder()
         ├─ Save payment to DB with status "pending"
         └─ Return order details to frontend
         │
         ▼
    Frontend Opens Gateway Checkout
         │
         ├─ Razorpay: Modal popup
         ├─ Stripe: Redirect or embedded form
         ├─ PhonePay: Redirect to payment page
         └─ Google Pay: Native payment sheet
         │
         ▼
    User Completes Payment
         │
         ▼
    Gateway Returns Payment Confirmation
         │
         ▼
    POST /api/payments/verify
         │
         ├─ Validate signature/token
         ├─ Call gateway.VerifyPayment()
         ├─ Verify amount matches
         ├─ Update payment status to "success"
         └─ Mark settlement as paid
         │
         ▼
    Settlement Marked as Paid
         │
         ▼
    User Sees Success Confirmation


ERROR HANDLING
==============

Possible errors and recovery:

1. Gateway Unreachable
   └─ Save payment as "pending" and retry later
   └─ User can retry manually

2. Invalid Signature
   └─ Mark payment as "failed"
   └─ Display error: "Payment could not be verified"
   └─ Settlement remains unpaid

3. Amount Mismatch
   └─ Mark payment as "failed"
   └─ Alert: "Amount mismatch detected"
   └─ Enable refund if payment was processed

4. Duplicate Order
   └─ Check UNIQUE constraint
   └─ Return existing payment record
   └─ Prevent double charging


TESTING
=======

Each gateway provides test/sandbox environment:

Razorpay:
  - Test cards provided in Razorpay docs
  - Switch between test/live in dashboard
  - No real money charged in test mode

Stripe:
  - Use published test cards (4242 4242...)
  - Cannot be confused with live cards
  - Separate test and live keys

PhonePay:
  - Sandbox environment available
  - Test merchant ID provided by PhonePay

Google Pay:
  - Test cards through Google Play Console
  - Sandbox mode for testing


ADDING NEW GATEWAYS
===================

Step 1: Create Gateway Implementation
   └─ Create file: {name}_gateway.go
   └─ Implement Gateway interface

Step 2: Register Gateway
   └─ Add case in payment.New()
   └─ Add config keys to .env.example

Step 3: Update Frontend
   └─ Add gateway option to payment.html
   └─ Add handler function for new gateway
   └─ Update GATEWAY_CONFIG in payment.js

Step 4: Test
   └─ Test with test credentials
   └─ Verify signature verification
   └─ Test refund functionality

Step 5: Documentation
   └─ Add gateway to PAYMENT_SETUP_GUIDE.md
   └─ Document setup steps
   └─ Provide troubleshooting tips


SECURITY CONSIDERATIONS
=======================

1. Signature Verification: Always verify gateway signatures
   - Prevents tampering
   - Validates payment authenticity

2. Amount Validation: Always verify amount matches
   - Prevents accidental overcharging
   - Catches payment manipulation

3. Idempotency: Payment verification must be idempotent
   - Safe to retry without duplicate charges
   - Use unique order ID + gateway combination

4. Credentials: Use environment variables
   - Never hardcode credentials
   - Separate test and live keys

5. Encryption: Use HTTPS always
   - Payment data must be encrypted in transit
   - PCI DSS requirement

6. Logging: Log all payment transactions
   - Include amount, gateway, status
   - Useful for reconciliation and debugging

*/

// No code here, this is documentation for the payment package
