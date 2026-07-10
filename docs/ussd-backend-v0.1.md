***

# Ppay USSD Backend Spec v0.1 – USSD‑First Architecture

## Purpose

Ppay is a **USSD‑first digital payments platform** with a shared transaction engine and multiple access channels.

In the South Sudan context:

- Internet is unreliable and expensive.  
- Data is costly.  
- Feature phones are common.  
- Smartphone penetration and continuous connectivity are limited.

Because of this, **80–90% of Ppay transactions will originate from USSD**, not from the mobile app. The Android/iPhone app is a **secondary convenience channel** for users with smartphones and reliable internet.

This spec defines:

- How USSD menus and sessions map to backend APIs for airtime and data bundles.  
- How USSD, mobile app, and future web/agent channels share the same core platform.  
- The initial production‑leaning design for airtime, data bundles, balance, and PIN via USSD.

***

## High‑Level Architecture

Ppay Core Platform is shared by all access channels:

```
                Ppay Core Platform
                       │
        ┌──────────────┼───────────────┐
        │              │               │
        ▼              ▼               ▼
      USSD         Mobile App      Future Web / Agents
   (Primary)      (Secondary)        (Secondary)
```

All channels call the **same backend transaction APIs** and share:

- Authentication and PIN verification.  
- Wallet and limits logic.  
- Outbox worker and provider connectors.  
- Settlement ledger and reconciliation.

### USSD‑First Components

Within the USSD path, Ppay has these core components:

```
                 USSD Gateway
                       │
                       ▼
               USSD Session Engine
                       │
     ┌─────────────────┼──────────────────┐
     ▼                 ▼                  ▼
Authentication     Menu Engine      Transaction API
     │                 │                  │
     └─────────────────┼──────────────────┘
                       ▼
                  Outbox Worker
                       ▼
             Provider Connectors
       MTN │ Zain │ Digitel │ SSIPSS (future)
```

- **USSD Gateway**: Connects to telco USSD platforms (MTN, Zain, Digitel).  
- **USSD Session Engine**: Manages session state, navigation, timeouts, language, and authentication status.  
- **Authentication**: Verifies PIN and user identity; handles account status (blocked, active).  
- **Menu Engine**: Renders menus from configuration (database), not hard‑coded.  
- **Transaction API**: Handles airtime/data flows, wallet checks, limits, and idempotency.  
- **Outbox Worker**: Asynchronously settles transactions and updates ledgers.  
- **Provider Connectors**: Integrate with MTN, Zain, Digitel, SSIPSS, and future providers.

***

## Phase‑Based Product Priorities

Given the realities of South Sudan, Ppay’s development priorities are:

### Phase 1 – USSD (Now, Primary)

Implemented via USSD, backed by Ppay Core:

- Airtime  
- Data Bundles  
- Balance  
- Mini Statement (basic transaction history)  
- Change PIN  

### Phase 2 – App (Secondary)

Implemented via mobile app, calling the same APIs:

- QR payments  
- Merchant payments  
- Rich transaction history  
- Push notifications and richer UX  

### Phase 3 – Extended Integrations

Using the same transaction engine and outbox:

- SSIPSS (national payment switch)  
- Banks  
- Utilities (electricity, water)  
- Government payments (taxes, fees)  
- NGO disbursements  
- Merchant POS/agent terminals

***

## USSD Main Menu – MVP (USSD‑First)

For the MVP USSD experience, the menu is focused and avoids dead‑end options:

**PPay Main Menu**

1. Balance  
2. Airtime  
3. Data Bundles  
4. My Account  
5. Help

Hidden for now (to avoid confusion until implemented):

- Send Money  
- Payments  
- Settings  
- Advanced services

This keeps the menu simple and aligned with what most USSD users will use daily: balance, airtime, and data.

***

## Menu Engine (Dynamic, Not Hard‑Coded)

Menus are **not hard‑coded in Go or USSD scripts**. Instead, they are stored in configuration (e.g. database tables) and rendered dynamically by the Menu Engine.

Example configuration:

- **Menu ID**: `MAIN`  
- **Entries**:

  - `1 Balance`  
  - `2 Airtime`  
  - `3 Bundles`  
  - `4 My Account`  
  - `5 Help`

Advantages:

- Change menus without deploying code.  
- Add NGO or government menus (e.g. “Pay Electricity”, “School Fees”).  
- Run promotions (e.g. discounted bundles).  
- A/B test menu order.  
- Support multiple languages (e.g. English, local languages).  
- Extend easily to new services without reworking USSD logic.

The USSD Session Engine uses the Menu Engine to render menus per session and language.

***

## Network Determination (MTN, Zain, Digitel)

Ppay is a **neutral telco aggregator**:

- Ppay’s USSD entry code works on MTN, Zain, and Digitel.  
- For **Self** flows, Ppay uses the SIM MSISDN to auto‑detect the network (via configured MSISDN ranges).  
- For **Other Number** flows:
  - Ppay attempts network detection from the entered MSISDN.  
  - If detection is ambiguous, Ppay prompts the user to select:

    ```
    Select network:
    1 MTN
    2 Zain
    3 Digitel
    ```

The `network` field in backend requests is always consistent with the actual provider.

***

## Airtime Flow (Self) – USSD

### USSD Screens

1. User selects: `2 Airtime` → `1 Self`.  
2. Screen:

   - `Enter amount (SSP):`  
   - `Min: SSP 10, Max: SSP 50,000`

3. Screen:

   - `Confirm`  
   - `Amount: SSP <X>`  
   - `From: Ppay Wallet`  
   - `To: <your phone>`  
   - `Network: <auto-detected MTN/Zain/Digitel>`  
   - `1. Confirm`  
   - `2. Cancel`

4. If user selects `1. Confirm`, ask for PIN:

   - `Enter your Ppay PIN:`

5. USSD Session Engine marks the request as “pending authentication” and calls the Auth service.

### PIN & Authentication

- PIN is **never logged**, never stored in plaintext, and never sent to providers.
- PIN is stored hashed (e.g. Argon2id or bcrypt) in an **Auth service**.

Auth flow:

1. USSD gateway/Session Engine calls Auth:

   - `POST /auth/pin/verify` (or similar), over HTTPS.

2. Auth verifies:

   - PIN correctness.  
   - Account status (active, not blocked).  
   - Optional additional checks (e.g. PIN retry limits).

3. If authentication succeeds:

   - The session is marked authenticated.  
   - The Session Engine calls the Transaction API (`/tx/airtime`) with an authenticated context.

4. If authentication fails:

   - USSD shows an error and **does not** call `/tx/airtime`.

### Pre‑Transaction Validation (Wallet & Limits)

Before creating an outbox event, `/tx/airtime` validates:

- Wallet exists and belongs to the user.  
- Wallet is active (not blocked/frozen).  
- Balance is sufficient for the amount and applicable fees.  
- Amount is within configured min/max (e.g. SSP 10–50,000).  
- Daily and per‑transaction limits (e.g. max daily airtime volume, count of purchases per day).  
- Basic AML/velocity checks (e.g. no more than N airtime purchases per hour for the same MSISDN).

If any of these checks fail, the API returns a suitable error and does **not** create an outbox event.

### Backend Mapping: `/tx/airtime` (Self)

Endpoint:

- `POST /tx/airtime`

Headers:

- `Content-Type: application/json`  
- `X-Idempotency-Key: <generated per request>`  
- `X-Correlation-ID: <USSD session ID>`

Body example (Self):

```json
{
  "network": "MTN",
  "currency": "SSP",
  "to_account": "22222222-2222-2222-2222-222222222222",
  "from_account": "11111111-1111-1111-1111-111111111111",
  "amount_minor": 10000,
  "phone_number": "+2119XXXXXXXXX",
  "product_type": "airtime",
  "idempotency_key": "<same as X-Idempotency-Key>",
  "correlation_id": "<same as X-Correlation-ID>"
}
```

Idempotency:

- Duplicate requests with the same `X-Idempotency-Key` and identical payload reuse the same `ppay_ref`.  
- `settlement_ledger` contains exactly one row per idempotency key.  
- Idempotency keys have a configurable expiry window (e.g. 24–48 hours); expired keys are rejected.

***

## Airtime Flow (Other Number) – USSD

### USSD Screens

1. User selects: `2 Airtime` → `2 Other Number`.  
2. Screen:

   - `Enter recipient phone number:`  

3. Screen:

   - `Network: <auto-detected>`, or if ambiguous:

     ```
     Select network:
     1 MTN
     2 Zain
     3 Digitel
     ```

4. Screen:

   - `Enter amount (SSP):`  
   - `Min: SSP 10, Max: SSP 50,000`

5. Screen:

   - `Confirm`  
   - `Amount: SSP <X>`  
   - `From: Ppay Wallet`  
   - `To: <recipient phone>`  
   - `Network: <MTN/Zain/Digitel>`  
   - `1. Confirm`  
   - `2. Cancel`

6. If `Confirm`, prompt for PIN and follow the Auth and validation flow. Backend calls `/tx/airtime` with `phone_number` = recipient MSISDN.

***

## Data Bundle Flow (Self) – USSD

### USSD Screens

1. User selects: `3 Data Bundles` → `Self`.  
2. Screen:

   - `Select network:`  
     - `1 MTN`  
     - `2 Zain`  
     - `3 Digitel`

3. Screen (bundle catalogue):

   - `Select bundle:`  
     - `1 100MB @ SSP 50`  
     - `2 500MB @ SSP 150`  
     - `3 1GB @ SSP 250`  
     - `4 More bundles` (optional)  
     - `5 Back`

Bundle options and prices are sourced from a **provider or Ppay catalogue**, not hard‑coded, so configuration changes don’t require code deployments.

4. Screen:

   - `Confirm`  
   - `Bundle: <selected bundle>`  
   - `Network: <selected network>`  
   - `To: <your phone>`  
   - `1. Confirm`  
   - `2. Cancel`

5. If `Confirm`, prompt for PIN and follow Auth and validation flow, then call `/tx/data-bundle`.

### Backend Mapping: `/tx/data-bundle`

Endpoint:

- `POST /tx/data-bundle`

Headers:

- `Content-Type: application/json`  
- `X-Idempotency-Key: <generated per request>`  
- `X-Correlation-ID: <USSD session ID>`

Body example:

```json
{
  "network": "MTN",
  "currency": "SSP",
  "to_account": "22222222-2222-2222-2222-222222222222",
  "from_account": "11111111-1111-1111-1111-111111111111",
  "amount_minor": 25000,
  "phone_number": "+2119XXXXXXXXX",
  "product_type": "data-bundle",
  "bundle_code": "MTN-1GB",
  "idempotency_key": "<same as X-Idempotency-Key>",
  "correlation_id": "<same as X-Correlation-ID>"
}
```

Behaviour:

- Duplicate requests with the same idempotency key reuse the same `ppay_ref`.  
- `settlement_ledger` has one row per idempotency key (validated via tests like `'idem-dup-data-001'`).

***

## Session Timeout Behaviour

USSD sessions expire after inactivity (e.g. 90–180 seconds). The USSD Session Engine ensures:

- If a session times out **before** PIN entry or confirmation, **no API call** is made and **no outbox event** is created.  
- Only confirmed and authenticated flows hit `/tx/airtime` or `/tx/data-bundle`.

This simplifies reasoning about duplicate sessions and reduces “ghost” transactions.

***

## USSD Success and Failure UX

### Success Screens

Once the backend has accepted and (eventually) settled the transaction, users see:

**Airtime Success**

```
Airtime purchase successful.

Amount: SSP 100
Network: MTN
Phone: 0923 XXX XXX
Ref: PPY24071012345

Balance: SSP 1,500
```

**Data Bundle Success**

```
Data bundle purchase successful.

Bundle: 1GB @ SSP 250
Network: MTN
Phone: 0923 XXX XXX
Ref: PPY24071067890
```

These references (`PPY...`) help users and support teams track transactions.

### Failure Screens

For failed transactions:

```
Transaction failed.

Reason:
Insufficient funds.

1 Retry
2 Main Menu
```

Other reasons might include:

- Limit exceeded.  
- Wrong PIN.  
- Provider timeout.  
- Account blocked.

The USSD experience must always show a clear reason and options.

***

## Balance, Mini Statement, My Account, PIN, Help

These menu items are reserved and will be implemented using the same core platform:

- **Balance**  
  - Shows wallet balance; later, may show total daily spent.  
  - Uses `/wallet/balance`.

- **Mini Statement** (via My Account)  
  - Shows a small list of recent transactions (e.g. last 5).  
  - Uses `/wallet/transactions` or similar.

- **My Account**  
  - Shows basic account info (name, MSISDN, status).

- **Change PIN**  
  - Flow: old PIN → new PIN → confirm new PIN.  
  - Uses `/auth/pin/change` on Auth service.

- **Help**  
  - Static or semi‑dynamic messages describing fees, limits, and telco/Ppay support contacts.

***

## Idempotency, Identifiers, and Audit

Idempotency:

- All USSD‑initiated airtime and data‑bundle requests must send `X-Idempotency-Key` and `idempotency_key` with the same value.  
- Idempotency keys expire after a configured period; expired keys are rejected.

Identifiers:

- **USSD Session ID** – used as `X-Correlation-ID`.  
- **Ppay Reference** (`ppay_ref`) – internal transaction reference.  
- **Provider Reference** (`provider_tx_ref`) – external telco/provider reference.

Every transaction is tracked using all three IDs for end‑to‑end traceability.

Audit logging:

Each transaction writes an audit record with:

- Timestamp  
- User ID / MSISDN  
- Session ID  
- Correlation ID  
- `ppay_ref`  
- `provider_tx_ref`  
- Network  
- Amount  
- Channel (USSD, app, web)  
- Outcome (success, failed, pending)  
- Error reason (if any)

***

## Reconciliation and Operations

Reconciliation:

- Nightly reconciliation between:
  - Ppay settlement ledger.  
  - Provider reports (MTN, Zain, Digitel).  
  - Ppay internal ledger (wallet balances).

Discrepancies raise alerts and trigger investigation.

Monitoring and alerting:

- Queue depth, worker health, error rates, and processing latency are monitored.  
- Alerts for:
  - High failure rates.  
  - Long processing times.  
  - Unusual transaction volumes (potential fraud).

Retries and resilience:

- Outbox worker uses backoff and retry policies (e.g. 1s → 5s → 15s, capped).  
- Failed transactions eventually move to terminal states (FAILED/UNKNOWN), with appropriate audit entries.

***

## USSD‑First Design Principle

Ppay is formally defined as:

> **A USSD‑first digital payments platform with a shared transaction engine and multiple access channels.**

The React Native app is a **premium interface** for smartphone users, not the primary product.

All channels—USSD, mobile app, future WhatsApp integrations, agent terminals, and web portals—call the same transaction APIs. Improvements to the core platform (wallet, limits, idempotency, reconciliation) benefit every channel.

This architecture scales as Ppay grows from airtime/data purchases into broader national payment infrastructure.
