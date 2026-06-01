--
-- PostgreSQL database dump
--


-- Dumped from database version 16.14
-- Dumped by pg_dump version 16.14

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

--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: idempotency_keys; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.idempotency_keys (
    id bigint NOT NULL,
    idempotency_key text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.idempotency_keys OWNER TO postgres;

--
-- Name: idempotency_keys_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.idempotency_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.idempotency_keys_id_seq OWNER TO postgres;

--
-- Name: idempotency_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.idempotency_keys_id_seq OWNED BY public.idempotency_keys.id;


--
-- Name: outbox_events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.outbox_events (
    id bigint NOT NULL,
    aggregate_id uuid,
    event_type text,
    payload jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    ppay_ref uuid,
    event_source text,
    correlation_id text,
    event_payload jsonb,
    published_at timestamp with time zone,
    topic text,
    state text,
    attempt_count integer DEFAULT 0 NOT NULL,
    last_error text,
    next_attempt_at timestamp with time zone
);


ALTER TABLE public.outbox_events OWNER TO postgres;

--
-- Name: outbox_events_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.outbox_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.outbox_events_id_seq OWNER TO postgres;

--
-- Name: outbox_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.outbox_events_id_seq OWNED BY public.outbox_events.id;


--
-- Name: settlement_ledger; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.settlement_ledger (
    ppay_ref uuid NOT NULL,
    idempotency_key text,
    state text NOT NULL,
    request_hash text,
    recon_status text,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    from_account text,
    to_account text,
    amount numeric(18,2),
    currency text,
    provider text,
    external_ref text,
    correlation_id text,
    metadata jsonb
);


ALTER TABLE public.settlement_ledger OWNER TO postgres;

--
-- Name: transaction_events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.transaction_events (
    id bigint NOT NULL,
    ppay_ref uuid NOT NULL,
    workflow_state text NOT NULL,
    event_source text NOT NULL,
    correlation_id text,
    event_payload jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.transaction_events OWNER TO postgres;

--
-- Name: transaction_events_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.transaction_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.transaction_events_id_seq OWNER TO postgres;

--
-- Name: transaction_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.transaction_events_id_seq OWNED BY public.transaction_events.id;


--
-- Name: idempotency_keys id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.idempotency_keys ALTER COLUMN id SET DEFAULT nextval('public.idempotency_keys_id_seq'::regclass);


--
-- Name: outbox_events id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.outbox_events ALTER COLUMN id SET DEFAULT nextval('public.outbox_events_id_seq'::regclass);


--
-- Name: transaction_events id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transaction_events ALTER COLUMN id SET DEFAULT nextval('public.transaction_events_id_seq'::regclass);


--
-- Name: idempotency_keys idempotency_keys_idempotency_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: idempotency_keys idempotency_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_pkey PRIMARY KEY (id);


--
-- Name: outbox_events outbox_events_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.outbox_events
    ADD CONSTRAINT outbox_events_pkey PRIMARY KEY (id);


--
-- Name: settlement_ledger settlement_ledger_idempotency_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.settlement_ledger
    ADD CONSTRAINT settlement_ledger_idempotency_key_key UNIQUE (idempotency_key);


--
-- Name: settlement_ledger settlement_ledger_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.settlement_ledger
    ADD CONSTRAINT settlement_ledger_pkey PRIMARY KEY (ppay_ref);


--
-- Name: transaction_events transaction_events_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.transaction_events
    ADD CONSTRAINT transaction_events_pkey PRIMARY KEY (id);


--
-- PostgreSQL database dump complete
--


