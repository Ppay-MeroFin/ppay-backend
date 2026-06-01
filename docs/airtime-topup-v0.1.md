# Airtime Top-up Backend Slice v0.1

## Purpose

This is the first working backend slice for Ppay airtime top-up.

Goals:
- expose a server-side API for creating airtime top-ups
- enforce idempotency through request headers
- support correlation IDs for traceability
- validate requests through a simulator provider
- prepare for later Postgres, ledger, and wallet integration

## Endpoint

POST /v1/airtime/topups

## Request headers

- Content-Type: application/json
- Idempotency-Key: string
- X-Correlation-ID: string

## Request body example

{
  "user_id": "user-123",
  "phone_number": "211912345678",
  "network": "mtn",
  "country_code": "SS",
  "amount_minor": 500,
  "currency": "SSP",
  "provider": "simulator",
  "client_ref": "demo-topup-001"
}

## Success response example

{
  "id": "20260507143127.078837295",
  "correlation_id": "corr-001",
  "idempotency_key": "topup-001",
  "status": "PROCESSING",
  "provider": "simulator",
  "phone_number": "211912345678",
  "network": "mtn",
  "amount_minor": 500,
  "currency": "SSP"
}

## Current behavior

- the service accepts valid supported networks
- the simulator currently supports: mtn, zain, digitel
- repeated requests with the same idempotency key return the same topup
- unsupported networks return an error
- successful submissions move from PENDING to PROCESSING

## Internal design

### Model

AirtimeTopup includes:
- ID
- CorrelationID
- IdempotencyKey
- UserID
- PhoneNumber
- Network
- CountryCode
- AmountMinor
- Currency
- Status
- Provider
- ProviderRef
- ClientRef
- FailureReason
- CreatedAt
- UpdatedAt

### State machine

Allowed transitions:
- PENDING -> PROCESSING
- PENDING -> FAILED
- PROCESSING -> SUCCESS
- PROCESSING -> FAILED

### Service flow

CreateAndSubmitTopup:
- checks idempotency key
- creates topup in PENDING
- sends request to provider simulator
- if simulator rejects, marks FAILED
- if simulator accepts, moves to PROCESSING

### Repository

Current repository is in-memory:
- topups stored by ID
- idempotency lookup stored by key
- sync.RWMutex protects concurrent access

### Provider

Current provider is a simulator:
- validates phone number
- validates network
- validates amount > 0
- accepts only supported networks

## Files implemented

- cmd/api/main.go
- internal/airtime/model.go
- internal/airtime/states.go
- internal/airtime/service.go
- internal/airtime/repository.go
- internal/airtime/events.go
- internal/api/handlers/airtime.go
- internal/providers/simulator/service.go

## Run locally

From backend root:

go run ./cmd/api

Server starts on:
:8080

## Curl test used

curl -X POST http://localhost:8080/v1/airtime/topups \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: topup-001" \
  -H "X-Correlation-ID: corr-001" \
  -d '{
    "user_id": "user-123",
    "phone_number": "211912345678",
    "network": "mtn",
    "country_code": "SS",
    "amount_minor": 500,
    "currency": "SSP",
    "provider": "simulator",
    "client_ref": "demo-topup-001"
  }'

## Verified outcomes

- idempotency works: retry with same key returns same topup record
- unsupported network is rejected
- backend starts successfully on port 8080
- first airtime backend slice is operational

## Next steps

- add GET /v1/airtime/topups/{id}
- improve request validation
- add structured error mapping
- record airtime events
- replace in-memory repository with Postgres
- prepare callback-style provider flow
