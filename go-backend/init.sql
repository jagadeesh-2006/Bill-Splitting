-- Registered users (only creators need an account)
CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    username   TEXT NOT NULL,
    email      TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,
    phone      TEXT NOT NULL UNIQUE
);

-- Groups created by a registered user
CREATE TABLE IF NOT EXISTS groups (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    created_by INT  NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Members of a group — no user account needed
-- Phone must be unique within a group
CREATE TABLE IF NOT EXISTS members (
    id         SERIAL PRIMARY KEY,
    group_id   INT  NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    phone      TEXT NOT NULL,
    UNIQUE(group_id, phone)
);

-- Expenses within a group
CREATE TABLE IF NOT EXISTS expenses (
    id          SERIAL PRIMARY KEY,
    group_id    INT     NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    description TEXT    NOT NULL,
    amount      NUMERIC(10,2) NOT NULL,
    paid_by_id  INT     NOT NULL REFERENCES members(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- How each expense is split between members
CREATE TABLE IF NOT EXISTS expense_splits (
    id          SERIAL PRIMARY KEY,
    expense_id  INT     NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    member_id   INT     NOT NULL REFERENCES members(id),
    amount_owed NUMERIC(10,2) NOT NULL
);

-- Settlements: member paying off debt to another member
CREATE TABLE IF NOT EXISTS settlements (
    id          SERIAL PRIMARY KEY,
    group_id    INT     NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    from_member INT     NOT NULL REFERENCES members(id), -- who paid
    to_member   INT     NOT NULL REFERENCES members(id), -- who received
    amount      NUMERIC(10,2) NOT NULL,
    note        TEXT    NOT NULL DEFAULT '',
    paid_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Payments: Track online payments for settlements
CREATE TABLE IF NOT EXISTS payments (
    id                  SERIAL PRIMARY KEY,
    settlement_id       INT     NOT NULL REFERENCES settlements(id) ON DELETE CASCADE,
    amount              NUMERIC(10,2) NOT NULL,
    currency            TEXT    NOT NULL DEFAULT 'INR',
    gateway             TEXT    NOT NULL, -- razorpay, stripe, phonepay, googlepay
    gateway_order_id    TEXT    NOT NULL,
    gateway_payment_id  TEXT,
    status              TEXT    NOT NULL DEFAULT 'pending', -- pending, success, failed, cancelled, refunded
    payment_method      TEXT,  -- card, upi, wallet, etc
    receipt             TEXT,
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at             TIMESTAMPTZ,
    UNIQUE(gateway, gateway_order_id)
);