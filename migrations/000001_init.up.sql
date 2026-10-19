-- Les bons comptes : schéma initial.
-- Les montants sont en centimes (BIGINT) : jamais de flottant pour de l'argent.

CREATE TABLE accounts (
                          id            BIGSERIAL PRIMARY KEY,
                          email         TEXT        NOT NULL UNIQUE,
                          password_hash TEXT        NOT NULL,
                          display_name  TEXT        NOT NULL,
                          created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE wallets (
                         id         BIGSERIAL PRIMARY KEY,
                         name       TEXT        NOT NULL,
                         owner_id   BIGINT      NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
                         status     TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
                         created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                         closed_at  TIMESTAMPTZ,
                         CONSTRAINT closed_at_coherent CHECK (
                             (status = 'closed' AND closed_at IS NOT NULL)
                                 OR (status = 'open' AND closed_at IS NULL)
                             )
);

CREATE INDEX idx_wallets_owner ON wallets (owner_id);

-- Appartenance et invitations dans une seule table : le statut porte l'invitation.
CREATE TABLE wallet_members (
                                wallet_id    BIGINT      NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
                                account_id   BIGINT      NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
                                status       TEXT        NOT NULL DEFAULT 'pending'
                                    CHECK (status IN ('pending', 'accepted', 'declined')),
                                invited_by   BIGINT      REFERENCES accounts (id) ON DELETE SET NULL,
                                invited_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
                                responded_at TIMESTAMPTZ,
                                PRIMARY KEY (wallet_id, account_id)
);

-- Sert la page « mes porte-monnaies » filtrée par statut.
CREATE INDEX idx_wallet_members_account_status ON wallet_members (account_id, status);

CREATE TABLE expenses (
                          id           BIGSERIAL PRIMARY KEY,
                          wallet_id    BIGINT      NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
                          label        TEXT        NOT NULL,
                          amount_cents BIGINT      NOT NULL CHECK (amount_cents > 0),
    -- Le compte qui a réellement payé. NULL si non attribué.
                          payer_id     BIGINT      REFERENCES accounts (id) ON DELETE SET NULL,
    -- Qui a saisi la ligne. Sert la règle de suppression de l'énoncé.
                          created_by   BIGINT      NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
                          spent_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
                          created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_expenses_wallet ON expenses (wallet_id, spent_at DESC);
CREATE INDEX idx_expenses_payer ON expenses (payer_id);

-- Virements calculés à la clôture. settled_at non nul = remboursement effectué.
CREATE TABLE settlements (
                             id              BIGSERIAL PRIMARY KEY,
                             wallet_id       BIGINT      NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
                             from_account_id BIGINT      NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
                             to_account_id   BIGINT      NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
                             amount_cents    BIGINT      NOT NULL CHECK (amount_cents > 0),
                             settled_at      TIMESTAMPTZ,
                             created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
                             CONSTRAINT no_self_transfer CHECK (from_account_id <> to_account_id)
);

CREATE INDEX idx_settlements_wallet ON settlements (wallet_id);

-- Sessions côté serveur : le cookie ne porte qu'un identifiant opaque.
CREATE TABLE sessions (
                          token      TEXT PRIMARY KEY,
                          account_id BIGINT      NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
                          expires_at TIMESTAMPTZ NOT NULL,
                          created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_expires ON sessions (expires_at);