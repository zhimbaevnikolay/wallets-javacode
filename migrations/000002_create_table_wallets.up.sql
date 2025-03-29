CREATE TABLE IF NOT EXISTS wallets (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0)
);