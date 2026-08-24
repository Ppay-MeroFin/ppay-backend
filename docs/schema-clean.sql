-- PostgreSQL database schema (clean version for ppay-backend)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';

SET default_tablespace = '';
SET default_table_access_method = heap;

-- =========================================================
-- idempotency_keys
-- =========================================================

CREATE TABLE public.idempotency_keys (
    id bigint NOT NULL,
    idempotency_key text NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL
);

CREATE SEQUENCE public.idempotency_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.idempotency_keys_id_seq OWNED BY public.idempotency_keys.id;

ALTER TABLE ONLY public.idempotency_keys
    ALTER COLUMN id SET DEFAULT nextval('public.idempotency_keys_id_seq'::regclass);

ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_idempotency_key_key UNIQUE (idempotency_key);

-- =========================================================
-- settlement_ledger
-- =========================================================

CREATE TABLE public.settlement_ledger (
    ppay_ref uuid NOT NULL,
    idempotency_key text NOT NULL,
    state text NOT NULL,
    request_hash text,
    recon_status text NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    from_account text,
    to_account text,
    amount numeric(18,2),
    currency text NOT NULL,
    provider text,
    external_ref text,
    correlation_id text,
    metadata jsonb,
    provider_tx_ref text,
    provider_status text,
    provider_response_payload jsonb
);

ALTER TABLE ONLY public.settlement_ledger
    ADD CONSTRAINT settlement_ledger_pkey PRIMARY KEY (ppay_ref);

ALTER TABLE ONLY public.settlement_ledger
    ADD CONSTRAINT settlement_ledger_idempotency_key_key UNIQUE (idempotency_key);

CREATE INDEX idx_settlement_ledger_created_at
    ON public.settlement_ledger (created_at);

CREATE INDEX idx_settlement_ledger_state
    ON public.settlement_ledger (state);

CREATE INDEX idx_settlement_ledger_provider_tx_ref
    ON public.settlement_ledger (provider_tx_ref);

CREATE INDEX idx_settlement_ledger_correlation_id
    ON public.settlement_ledger (correlation_id);

-- =========================================================
-- outbox_events
-- =========================================================

CREATE TABLE public.outbox_events (
    id bigint NOT NULL,
    aggregate_id uuid,
    event_type text,
    payload jsonb,
    created_at timestamptz DEFAULT now() NOT NULL,
    ppay_ref uuid,
    event_source text,
    correlation_id text,
    event_payload jsonb,
    published_at timestamptz,
    topic text NOT NULL,
    state text NOT NULL,
    attempt_count integer DEFAULT 0 NOT NULL,
    last_error text,
    next_attempt_at timestamptz,
    processed_at timestamptz
);

CREATE SEQUENCE public.outbox_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.outbox_events_id_seq OWNED BY public.outbox_events.id;

ALTER TABLE ONLY public.outbox_events
    ALTER COLUMN id SET DEFAULT nextval('public.outbox_events_id_seq'::regclass);

ALTER TABLE ONLY public.outbox_events
    ADD CONSTRAINT outbox_events_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.outbox_events
    ADD CONSTRAINT outbox_events_ppay_ref_fkey
    FOREIGN KEY (ppay_ref) REFERENCES public.settlement_ledger(ppay_ref);

CREATE INDEX idx_outbox_events_ppay_ref
    ON public.outbox_events (ppay_ref);

CREATE INDEX idx_outbox_events_state_next_attempt_created
    ON public.outbox_events (state, next_attempt_at, created_at);

CREATE INDEX idx_outbox_events_topic
    ON public.outbox_events (topic);

CREATE INDEX idx_outbox_events_correlation_id
    ON public.outbox_events (correlation_id);

-- =========================================================
-- transaction_events
-- =========================================================

CREATE TABLE public.transaction_events (
    id bigint NOT NULL,
    ppay_ref uuid NOT NULL,
    workflow_state text NOT NULL,
    event_source text NOT NULL,
    correlation_id text,
    event_payload jsonb,
    created_at timestamptz DEFAULT now() NOT NULL
);

CREATE SEQUENCE public.transaction_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.transaction_events_id_seq OWNED BY public.transaction_events.id;

ALTER TABLE ONLY public.transaction_events
    ALTER COLUMN id SET DEFAULT nextval('public.transaction_events_id_seq'::regclass);

ALTER TABLE ONLY public.transaction_events
    ADD CONSTRAINT transaction_events_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.transaction_events
    ADD CONSTRAINT transaction_events_ppay_ref_fkey
    FOREIGN KEY (ppay_ref) REFERENCES public.settlement_ledger(ppay_ref);

CREATE INDEX idx_transaction_events_ppay_ref_created_at
    ON public.transaction_events (ppay_ref, created_at);

CREATE INDEX idx_transaction_events_workflow_state
    ON public.transaction_events (workflow_state);

CREATE INDEX idx_transaction_events_correlation_id
    ON public.transaction_events (correlation_id);