-- Tummesan PPOB: complete PostgreSQL 15+ schema for a fresh database.
-- Apply this file once to an empty application database with psql.
-- Core migration 002 is intentionally consolidated here.
-- No external payment gateway, points ledger, invoice_number, or fulfillment_status.

-- Provides gen_random_uuid() on PostgreSQL versions where it is not built in.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

BEGIN;

-- ---------------------------------------------------------------------------
-- Laravel database-backed session, cache, and queue support (Zero Redis).
-- ---------------------------------------------------------------------------
CREATE TABLE sessions (
    id              VARCHAR(255) PRIMARY KEY,
    user_id         BIGINT NULL,
    ip_address      VARCHAR(45) NULL,
    user_agent      TEXT NULL,
    payload         TEXT NOT NULL,
    last_activity   INTEGER NOT NULL
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_last_activity_idx ON sessions (last_activity);

CREATE TABLE cache (
    key         VARCHAR(255) PRIMARY KEY,
    value       TEXT NOT NULL,
    expiration  INTEGER NOT NULL
);
CREATE TABLE cache_locks (
    key         VARCHAR(255) PRIMARY KEY,
    owner       VARCHAR(255) NOT NULL,
    expiration  INTEGER NOT NULL
);

CREATE TABLE jobs (
    id              BIGSERIAL PRIMARY KEY,
    queue           VARCHAR(255) NOT NULL,
    payload         TEXT NOT NULL,
    attempts        SMALLINT NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 255),
    reserved_at     INTEGER NULL,
    available_at    INTEGER NOT NULL,
    created_at      INTEGER NOT NULL
);
CREATE INDEX jobs_queue_idx ON jobs (queue);
CREATE TABLE job_batches (
    id              VARCHAR(255) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    total_jobs      INTEGER NOT NULL,
    pending_jobs    INTEGER NOT NULL,
    failed_jobs     INTEGER NOT NULL,
    failed_job_ids  TEXT NOT NULL,
    options         TEXT NULL,
    cancelled_at    INTEGER NULL,
    created_at      INTEGER NOT NULL,
    finished_at     INTEGER NULL
);
CREATE TABLE failed_jobs (
    id          BIGSERIAL PRIMARY KEY,
    uuid        VARCHAR(255) NOT NULL UNIQUE,
    connection  TEXT NOT NULL,
    queue       TEXT NOT NULL,
    payload     TEXT NOT NULL,
    exception   TEXT NOT NULL,
    failed_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Laravel migration repository. The records below mark the bundled Laravel
-- infrastructure migrations as represented by this consolidated schema.
CREATE TABLE migrations (
    id          SERIAL PRIMARY KEY,
    migration   VARCHAR(255) NOT NULL,
    batch       INTEGER NOT NULL
);

-- ---------------------------------------------------------------------------
-- KYC identities, price groups, users, and OTP verification.
-- ---------------------------------------------------------------------------
CREATE TABLE user_identities (
    id                      BIGSERIAL PRIMARY KEY,
    id_card_type            VARCHAR(20) NOT NULL DEFAULT 'KTP',
    id_card_number          VARCHAR(50) NOT NULL UNIQUE,
    full_legal_name         VARCHAR(150) NOT NULL,
    birth_place             VARCHAR(100) NULL,
    birth_date              DATE NULL,
    gender                  VARCHAR(10) NULL,
    full_address            TEXT NULL,
    city                    VARCHAR(100) NULL,
    province                VARCHAR(100) NULL,
    postal_code             VARCHAR(10) NULL,
    id_card_image_url       TEXT NULL,
    selfie_image_url        TEXT NULL,
    verification_status     VARCHAR(20) NOT NULL DEFAULT 'UNVERIFIED',
    rejection_reason        TEXT NULL,
    verified_at             TIMESTAMPTZ NULL,
    verified_by_user_id     BIGINT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_identities_status_check
        CHECK (verification_status IN ('UNVERIFIED', 'PENDING', 'VERIFIED', 'REJECTED'))
);

CREATE TABLE price_groups (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(50) NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    description TEXT NULL,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX price_groups_one_default_idx ON price_groups (is_default) WHERE is_default = TRUE;

CREATE TABLE users (
    id                  BIGSERIAL PRIMARY KEY,
    identity_id         BIGINT NULL REFERENCES user_identities(id) ON DELETE SET NULL,
    price_group_id      INTEGER NULL REFERENCES price_groups(id) ON DELETE SET NULL,
    name                VARCHAR(100) NOT NULL,
    phone_number        VARCHAR(20) NOT NULL UNIQUE,
    email               VARCHAR(255) NULL UNIQUE,
    email_verified_at   TIMESTAMPTZ NULL,
    password_hash       VARCHAR(255) NULL,
    -- Laravel compatibility; merchant authentication uses password_hash via Core.
    password            VARCHAR(255) NULL,
    remember_token      VARCHAR(100) NULL,
    role                VARCHAR(20) NOT NULL DEFAULT 'MERCHANT',
    store_name          VARCHAR(100) NULL,
    store_address       TEXT NULL,
    current_device_id   VARCHAR(255) NULL,
    is_phone_verified   BOOLEAN NOT NULL DEFAULT FALSE,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    -- Kept at zero for compatibility with the shared Go user scanner; no points feature is enabled.
    points_balance      NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (points_balance >= 0),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_identity_unique UNIQUE (identity_id)
);
CREATE INDEX users_role_active_idx ON users (role, is_active);

ALTER TABLE user_identities
    ADD CONSTRAINT user_identities_verified_by_fk
    FOREIGN KEY (verified_by_user_id) REFERENCES users(id) ON DELETE SET NULL;

CREATE TABLE password_reset_tokens (
    email       VARCHAR(255) PRIMARY KEY,
    token       VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ NULL
);

CREATE TABLE otp_verifications (
    id              BIGSERIAL PRIMARY KEY,
    phone_number    VARCHAR(20) NOT NULL,
    otp_code_hash   VARCHAR(255) NOT NULL,
    action_type     VARCHAR(50) NOT NULL,
    channel         VARCHAR(20) NOT NULL DEFAULT 'WHATSAPP',
    attempts_count  SMALLINT NOT NULL DEFAULT 0 CHECK (attempts_count BETWEEN 0 AND 3),
    is_consumed     BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT otp_expiry_after_creation CHECK (expires_at > created_at),
    CONSTRAINT otp_ttl_max_five_minutes CHECK (expires_at <= created_at + INTERVAL '5 minutes')
);
CREATE INDEX otp_phone_action_created_idx
    ON otp_verifications (phone_number, action_type, created_at DESC, id DESC);
CREATE INDEX otp_active_expiry_idx
    ON otp_verifications (phone_number, action_type, expires_at)
    WHERE is_consumed = FALSE;

-- ---------------------------------------------------------------------------
-- Catalog, one B2B provider, prefixes, and per-segment pricing.
-- ---------------------------------------------------------------------------
CREATE TABLE provider_config (
    id                  SERIAL PRIMARY KEY,
    provider_name       VARCHAR(50) NOT NULL DEFAULT 'MAIN_B2B_PROVIDER',
    base_url            VARCHAR(255) NOT NULL,
    api_username        VARCHAR(100) NOT NULL,
    api_key_encrypted   TEXT NOT NULL,
    deposit_balance     NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (deposit_balance >= 0),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX provider_config_one_active_idx ON provider_config (is_active) WHERE is_active = TRUE;

CREATE TABLE product_categories (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(50) NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    icon_url    TEXT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_brands (
    id          SERIAL PRIMARY KEY,
    category_id INTEGER NULL REFERENCES product_categories(id) ON DELETE RESTRICT,
    code        VARCHAR(50) NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    logo_url    TEXT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX product_brands_category_active_idx ON product_brands (category_id, is_active, sort_order);

CREATE TABLE brand_prefixes (
    id              SERIAL PRIMARY KEY,
    brand_id        INTEGER NOT NULL REFERENCES product_brands(id) ON DELETE CASCADE,
    prefix_number   VARCHAR(10) NOT NULL UNIQUE,
    description     VARCHAR(100) NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX brand_prefixes_prefix_number_idx ON brand_prefixes (prefix_number);
CREATE INDEX brand_prefixes_brand_active_idx ON brand_prefixes (brand_id, is_active);

CREATE TABLE digital_products (
    id                      SERIAL PRIMARY KEY,
    category_id             INTEGER NOT NULL REFERENCES product_categories(id) ON DELETE RESTRICT,
    brand_id                INTEGER NOT NULL REFERENCES product_brands(id) ON DELETE RESTRICT,
    sku_internal            VARCHAR(50) NOT NULL UNIQUE,
    provider_sku            VARCHAR(100) NOT NULL,
    name                    VARCHAR(150) NOT NULL,
    nominal_value           NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (nominal_value >= 0),
    base_cost               NUMERIC(12,2) NOT NULL CHECK (base_cost >= 0),
    provider_admin_fee      NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (provider_admin_fee >= 0),
    provider_merchant_fee   NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (provider_merchant_fee >= 0),
    product_type            VARCHAR(20) NOT NULL DEFAULT 'PREPAID'
        CHECK (product_type IN ('PREPAID', 'POSTPAID')),
    is_fast_product         BOOLEAN NOT NULL DEFAULT FALSE,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX digital_products_catalog_idx ON digital_products (category_id, brand_id, is_active, nominal_value);
CREATE INDEX digital_products_fast_idx ON digital_products (is_fast_product, category_id, is_active)
    WHERE is_fast_product = TRUE;
CREATE INDEX digital_products_provider_sku_idx ON digital_products (provider_sku);

CREATE TABLE pricing_rules (
    id                  SERIAL PRIMARY KEY,
    digital_product_id  INTEGER NOT NULL REFERENCES digital_products(id) ON DELETE CASCADE,
    price_group_id      INTEGER NOT NULL REFERENCES price_groups(id) ON DELETE CASCADE,
    spread              NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (spread >= 0),
    admin_fee           NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (admin_fee >= 0),
    merchant_fee        NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (merchant_fee >= 0),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pricing_rules_product_group_unique UNIQUE (digital_product_id, price_group_id)
);
CREATE INDEX pricing_rules_group_product_idx ON pricing_rules (price_group_id, digital_product_id);

-- Empty compatibility table for the Go transaction scanner. No external gateway
-- payment methods are seeded or used by the direct-to-provider API.
CREATE TABLE payment_methods (
    id                      SERIAL PRIMARY KEY,
    code                    VARCHAR(50) NOT NULL UNIQUE,
    gateway_channel_code    VARCHAR(50) NOT NULL DEFAULT '',
    name                    VARCHAR(100) NOT NULL,
    category                VARCHAR(50) NOT NULL DEFAULT 'INTERNAL',
    icon_url                TEXT NULL,
    fee_type                VARCHAR(15) NOT NULL DEFAULT 'FIXED',
    fixed_fee               NUMERIC(12,2) NOT NULL DEFAULT 0,
    percentage_fee          NUMERIC(5,2) NOT NULL DEFAULT 0,
    min_amount              NUMERIC(12,2) NOT NULL DEFAULT 0,
    max_amount              NUMERIC(12,2) NOT NULL DEFAULT 0,
    is_active               BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order              INTEGER NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ---------------------------------------------------------------------------
-- Transaction record: reference_no is the sole public identifier.
-- Internal provider accounting/status fields are never public API fields.
-- ---------------------------------------------------------------------------
CREATE TABLE transactions (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_no                VARCHAR(50) NOT NULL UNIQUE,
    user_id                     BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    price_group_id              INTEGER NULL REFERENCES price_groups(id) ON DELETE SET NULL,
    payment_method_id           INTEGER NULL REFERENCES payment_methods(id) ON DELETE SET NULL,
    customer_phone              VARCHAR(20) NOT NULL,
    digital_product_id          INTEGER NULL REFERENCES digital_products(id) ON DELETE SET NULL,
    customer_id                 VARCHAR(50) NOT NULL,
    customer_name               VARCHAR(100) NULL,
    base_hpp                    NUMERIC(12,2) NOT NULL CHECK (base_hpp >= 0),
    provider_admin_fee          NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (provider_admin_fee >= 0),
    provider_merchant_fee       NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (provider_merchant_fee >= 0),
    product_price               NUMERIC(12,2) NOT NULL CHECK (product_price >= 0),
    admin_fee                   NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (admin_fee >= 0),
    total_amount                NUMERIC(12,2) NOT NULL CHECK (total_amount >= 0),
    merchant_markup             NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (merchant_markup >= 0),
    grand_total                 NUMERIC(12,2) NOT NULL CHECK (grand_total >= 0),
    -- Legacy scanner compatibility; direct confirmation does not use gateway/points.
    marketing_fee_earned        NUMERIC(12,2) NOT NULL DEFAULT 0,
    cashback_points             NUMERIC(12,2) NOT NULL DEFAULT 0,
    payment_status_code         VARCHAR(50) NOT NULL DEFAULT 'NOT_APPLICABLE',
    payment_status_desc         TEXT NOT NULL DEFAULT 'Pembayaran langsung ke provider',
    payment_gateway_ref         VARCHAR(100) NULL,
    -- payment_payload             TEXT NULL,
    status_code                 VARCHAR(50) NOT NULL DEFAULT '06',
    status_description          TEXT NOT NULL DEFAULT 'Inquiry berhasil, siap untuk pembayaran',
    provider_status_code        VARCHAR(100) NULL,
    provider_status_description TEXT NULL,
    provider_ref_id             VARCHAR(100) NULL,
    -- provider_raw_response       JSONB NULL,
    serial_number               VARCHAR(255) NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX transactions_customer_phone_created_idx ON transactions (customer_phone, created_at DESC);
CREATE INDEX transactions_user_created_idx ON transactions (user_id, created_at DESC);
CREATE INDEX transactions_status_updated_idx ON transactions (status_code, updated_at ASC);
CREATE INDEX transactions_provider_ref_idx ON transactions (provider_ref_id) WHERE provider_ref_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- Seed data: one default guest/new-user group and minimal catalog examples.
-- Replace demo product/provider SKU and cost values before production use.
-- ---------------------------------------------------------------------------
INSERT INTO price_groups (code, name, description, is_default, is_active) VALUES
    ('DEFAULT_RETAIL', 'Retail & Guest', 'Default group for guests and newly registered users', TRUE, TRUE),
    ('MERCHANT_SILVER', 'Merchant Silver', 'Standard merchant pricing', FALSE, TRUE),
    ('MERCHANT_GOLD', 'Merchant Gold', 'High-volume merchant pricing', FALSE, TRUE)
ON CONFLICT (code) DO NOTHING;

INSERT INTO product_categories (code, name, icon_url, sort_order) VALUES
    ('PULSA', 'Pulsa', '/icons/pulsa.svg', 1),
    ('PLN_TOKEN', 'Token PLN', '/icons/pln.svg', 2),
    ('PLN_TAGIHAN', 'Tagihan PLN', '/icons/pln.svg', 3)
ON CONFLICT (code) DO NOTHING;

INSERT INTO product_brands (code, name, logo_url, sort_order) VALUES
    ('TELKOMSEL', 'Telkomsel', '/icons/brands/telkomsel.svg', 1),
    ('INDOSAT', 'Indosat', '/icons/brands/indosat.svg', 2),
    ('XL', 'XL', '/icons/brands/xl.svg', 3),
    ('TRI', 'Tri', '/icons/brands/tri.svg', 4),
    ('SMARTFREN', 'Smartfren', '/icons/brands/smartfren.svg', 5),
    ('PLN', 'PLN', '/icons/brands/pln.svg', 6)
ON CONFLICT (code) DO NOTHING;

INSERT INTO brand_prefixes (brand_id, prefix_number, description)
SELECT b.id, v.prefix_number, v.description
FROM (VALUES
    ('TELKOMSEL', '0811', 'Telkomsel'), ('TELKOMSEL', '0812', 'Telkomsel'),
    ('TELKOMSEL', '0813', 'Telkomsel'), ('TELKOMSEL', '0821', 'Telkomsel'),
    ('TELKOMSEL', '0822', 'Telkomsel'), ('TELKOMSEL', '0823', 'Telkomsel'),
    ('TELKOMSEL', '0851', 'Telkomsel'), ('TELKOMSEL', '0852', 'Telkomsel'),
    ('TELKOMSEL', '0853', 'Telkomsel'), ('INDOSAT', '0814', 'Indosat'),
    ('INDOSAT', '0815', 'Indosat'), ('INDOSAT', '0816', 'Indosat'),
    ('INDOSAT', '0855', 'Indosat'), ('INDOSAT', '0856', 'Indosat'),
    ('INDOSAT', '0857', 'Indosat'), ('INDOSAT', '0858', 'Indosat'),
    ('XL', '0817', 'XL'), ('XL', '0818', 'XL'), ('XL', '0819', 'XL'),
    ('XL', '0859', 'XL'), ('XL', '0877', 'XL'), ('XL', '0878', 'XL'),
    ('TRI', '0895', 'Tri'), ('TRI', '0896', 'Tri'), ('TRI', '0897', 'Tri'),
    ('TRI', '0898', 'Tri'), ('TRI', '0899', 'Tri'),
    ('SMARTFREN', '0881', 'Smartfren'), ('SMARTFREN', '0882', 'Smartfren'),
    ('SMARTFREN', '0883', 'Smartfren'), ('SMARTFREN', '0887', 'Smartfren'),
    ('SMARTFREN', '0888', 'Smartfren')
) AS v(brand_code, prefix_number, description)
JOIN product_brands b ON b.code = v.brand_code
ON CONFLICT (prefix_number) DO NOTHING;

INSERT INTO digital_products
    (category_id, brand_id, sku_internal, provider_sku, name, nominal_value, base_cost,
     provider_admin_fee, provider_merchant_fee, product_type, is_fast_product, is_active)
SELECT c.id, b.id, v.sku, v.provider_sku, v.name, v.nominal, v.base_cost,
       v.provider_admin_fee, v.provider_merchant_fee, v.product_type, v.is_fast, TRUE
FROM (VALUES
    ('PULSA', 'TELKOMSEL', 'TSEL_10K', 'htelkomsel10000', 'Telkomsel 10.000', 10000, 10150, 0, 0, 'PREPAID', TRUE),
    ('PULSA', 'TELKOMSEL', 'TSEL_20K', 'htelkomsel20000', 'Telkomsel 20.000', 20000, 19950, 0, 0, 'PREPAID', FALSE),
    ('PULSA', 'INDOSAT', 'ISAT_10K', 'hindosat10000', 'Indosat 10.000', 10000, 10200, 0, 0, 'PREPAID', FALSE),
    ('PLN_TOKEN', 'PLN', 'PLN_TOKEN_20K', 'pln20000', 'Token PLN 20.000', 20000, 20100, 0, 0, 'PREPAID', TRUE),
    ('PLN_TOKEN', 'PLN', 'PLN_TOKEN_50K', 'pln50000', 'Token PLN 50.000', 50000, 50100, 0, 0, 'PREPAID', FALSE),
    ('PLN_TAGIHAN', 'PLN', 'PLN_POSTPAID', 'plnpostpaid', 'Tagihan PLN Pascabayar', 0, 0, 0, 0, 'POSTPAID', FALSE)
) AS v(category_code, brand_code, sku, provider_sku, name, nominal, base_cost,
       provider_admin_fee, provider_merchant_fee, product_type, is_fast)
JOIN product_categories c ON c.code = v.category_code
JOIN product_brands b ON b.code = v.brand_code
ON CONFLICT (sku_internal) DO NOTHING;

-- Every seeded product is priced for every seeded segment. Adjust these
-- demo spreads and fees to match the commercial pricing plan.
INSERT INTO pricing_rules (digital_product_id, price_group_id, spread, admin_fee, merchant_fee)
SELECT p.id, g.id,
       CASE g.code WHEN 'DEFAULT_RETAIL' THEN 1000 WHEN 'MERCHANT_SILVER' THEN 600 ELSE 400 END,
       CASE WHEN p.product_type = 'POSTPAID' THEN 1500 ELSE 0 END,
       CASE g.code WHEN 'DEFAULT_RETAIL' THEN 0 WHEN 'MERCHANT_SILVER' THEN 100 ELSE 150 END
FROM digital_products p
CROSS JOIN price_groups g
WHERE p.sku_internal IN ('TSEL_10K', 'TSEL_20K', 'ISAT_10K', 'PLN_TOKEN_20K', 'PLN_TOKEN_50K', 'PLN_POSTPAID')
  AND g.code IN ('DEFAULT_RETAIL', 'MERCHANT_SILVER', 'MERCHANT_GOLD')
ON CONFLICT (digital_product_id, price_group_id) DO NOTHING;

-- Keep serial sequences ahead of the demo IDs if rows are reloaded manually.
SELECT setval(pg_get_serial_sequence('price_groups', 'id'), COALESCE((SELECT MAX(id) FROM price_groups), 1));
SELECT setval(pg_get_serial_sequence('product_categories', 'id'), COALESCE((SELECT MAX(id) FROM product_categories), 1));
SELECT setval(pg_get_serial_sequence('product_brands', 'id'), COALESCE((SELECT MAX(id) FROM product_brands), 1));
SELECT setval(pg_get_serial_sequence('digital_products', 'id'), COALESCE((SELECT MAX(id) FROM digital_products), 1));

INSERT INTO migrations (migration, batch) VALUES
    ('0001_01_01_000000_create_users_table', 1),
    ('0001_01_01_000001_create_cache_table', 1),
    ('0001_01_01_000002_create_jobs_table', 1),
    ('2026_10_12_000000_ensure_sessions_table_exists', 1);

COMMIT;
