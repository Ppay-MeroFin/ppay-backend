# ppay-backend

Ppay backend service for transaction orchestration, starting with **airtime** and **data bundle** flows.

## Overview

This service is the Go backend for Ppay. It is designed around a clean transaction architecture with:

- request validation at the handler boundary
- domain-level transaction construction in the ledger layer
- atomic writes to settlement and outbox tables
- idempotent transaction creation
- replay and conflict handling for duplicate requests
- reconciliation-ready transaction state management

The current launch scope is focused on:

- Airtime
- Data Bundles

## Tech Stack

- Go
- PostgreSQL
- pgx / pgxpool
- GitHub for source control

## Project Structure

- `cmd/api` - API entrypoint
- `internal/api/handlers` - API handlers
- `internal/handlers` - handler tests and supporting logic
- `internal/ledger` - domain transaction models and state mapping
- `internal/store` - PostgreSQL-backed transaction store
- `internal/providers/simulator` - provider simulation logic
- `docs` - project documents

## Local Setup

### 1. Clone the repository

```bash
git clone https://github.com/Ppay-MeroFin/ppay-backend.git
cd ppay-backend
```

### 2. Create your local environment file

Copy `.env.example` to `.env` and replace the placeholder values with your real local configuration.

### 3. Configure PostgreSQL

This project is currently tested locally against PostgreSQL running on:

- Host: `127.0.0.1`
- Port: `5433`
- Database: `ppay`

Make sure the database exists and that the required tables are created before running integration tests.

## Environment Variables

Current local example:

```env
DBHOST=127.0.0.1
DBPORT=5433
DBUSER=postgres
DBPASSWORD=your-password
DBNAME=ppay
DBSSLMODE=disable
PORT=8080
```

## Run the API

```bash
go run ./cmd/api
```

The API listens on port `8080` in the current local setup.

## Run Tests

Run all tests:

```bash
go test ./... -v
```

Run store tests only:

```bash
go test ./internal/store -v
```

## Current Status

The backend currently has:

- passing Go test suite
- GitHub source control setup
- local environment example file
- local PostgreSQL-backed integration flow for store tests

## Notes

- Keep real secrets in a local `.env` file only.
- Do not commit `.env` or generated binaries.
- The current repository is the backend foundation for the Ppay product as it moves toward production readiness.

## Contact

Project contact: `ceo@merofintech.com`
