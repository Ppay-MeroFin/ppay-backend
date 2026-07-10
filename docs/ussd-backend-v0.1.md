Here’s a **Ppay USSD backend spec v0.1** you can save in `docs/` (e.g. `ussd-backend-v0.1.md`) alongside your airtime-topup doc.

***

# Ppay USSD Backend Spec v0.1

## Purpose

This document defines how Ppay’s USSD menus map to backend APIs for airtime and data bundles.

Goals:

- Provide a simple USSD main menu similar to MTN MoMo.  
- Map airtime and data-bundle USSD flows to existing `/tx/airtime` and `/tx/data-bundle` endpoints. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)
- Enforce idempotency through `X-Idempotency-Key` headers and request payloads. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)
- Support correlation IDs for traceability. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)

***

## Main Menu

When the user dials the Ppay code (e.g. `*XYZ#`), show:

**Main Menu**

1. Balance  
2. Airtime & Bundles  
3. Send Money (future)  
4. Payments (future)  
5. My Account  
6. Settings  
7. Change PIN  
8. Help

Initial implementation focuses on:

- 1. Balance (to be wired to wallet API later)  
- 2. Airtime & Bundles (fully wired)  
- 7. Change PIN (future)  
- 8. Help

***

## Airtime & Bundles Menu

When user chooses **2. Airtime & Bundles**, show:

**Airtime & Bundles**

1. Buy Airtime (Self)  
2. Buy Airtime (Other Number)  
3. Buy Data Bundle (Self)  
4. Buy Data Bundle (Other Number)  
5. Back

***

## Airtime Flow (Self)

### USSD screens

1. User selects: `2 -> 1` (Airtime & Bundles → Buy Airtime (Self)).  
2. Screen:

   - `Enter amount (SSP):`

3. Screen:

   - `Confirm`  
   - `Amount: SSP <X>`  
   - `From: Ppay Wallet`  
   - `To: <your phone>`  
   - `1. Confirm`  
   - `2. Cancel`

4. If user selects `1. Confirm`, ask for PIN:

   - `Enter your Ppay PIN:`

5. After PIN validation, USSD gateway calls backend API.

### Backend mapping: `/tx/airtime`

Endpoint:

- `POST /tx/airtime` [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)

Headers:

- `Content-Type: application/json`  
- `X-Idempotency-Key: <generated-per-request>`  
- `X-Correlation-ID: <ussd-session-id>`

Body example:

```json
{
  "network": "MTN",
  "currency": "SSP",
  "to_account": "22222222-2222-2222-2222-222222222222",
  "from_account": "11111111-1111-1111-1111-111111111111",
  "amount_minor": 10000,
  "phone_number": "+2119XXXXXXXXX",
  "product_type": "airtime",
  "idempotency_key": "<same as header>",
  "correlation_id": "<same as X-Correlation-ID>"
}
```

Behaviour:

- Duplicate requests with the same `X-Idempotency-Key` and body reuse the same `ppay_ref`. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)
- After settlement by the outbox worker, `settlement_ledger` contains one row per idempotency key. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)

***

## Airtime Flow (Other Number)

### USSD screens

1. User selects: `2 -> 2` (Buy Airtime (Other Number)).  
2. Screen:

   - `Enter recipient phone number:`

3. Screen:

   - `Enter amount (SSP):`

4. Screen:

   - `Confirm`  
   - `Amount: SSP <X>`  
   - `From: Ppay Wallet`  
   - `To: <recipient phone>`  
   - `1. Confirm`  
   - `2. Cancel`

5. If `Confirm`, prompt for PIN, then call `/tx/airtime` as above, with `phone_number` = recipient MSISDN.

***

## Data Bundle Flow (Self)

### USSD screens

1. User selects: `2 -> 3` (Buy Data Bundle (Self)).  
2. Screen:

   - `Select network:`  
     - `1. MTN`  
     - `2. Zain`  
     - `3. Digitel`

3. Screen (based on network):

   - `Select bundle:`  
     - `1. 100MB @ SSP 50`  
     - `2. 500MB @ SSP 150`  
     - `3. 1GB @ SSP 250`  
     - `4. Back`

4. Screen:

   - `Confirm`  
   - `Bundle: 1GB @ SSP 250`  
   - `To: <your phone>`  
   - `1. Confirm`  
   - `2. Cancel`

5. If `Confirm`, prompt for PIN, then call backend.

### Backend mapping: `/tx/data-bundle`

Endpoint:

- `POST /tx/data-bundle` [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)

Headers:

- `Content-Type: application/json`  
- `X-Idempotency-Key: <generated-per-request>`  
- `X-Correlation-ID: <ussd-session-id>`

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
  "idempotency_key": "<same as header>",
  "correlation_id": "<same as X-Correlation-ID>"
}
```

Behaviour:

- Duplicate requests with the same idempotency key reuse the same `ppay_ref`. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)
- `settlement_ledger` has one row for `idempotency_key = 'idem-dup-data-001'` in tests. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)

***

## Balance / My Account / PIN (future scope)

These menu items are reserved:

- **1. Balance**  
  - Show wallet balance.  
  - Will map to a future wallet balance endpoint.

- **5. My Account**  
  - Show basic account info (name, MSISDN, status).

- **7. Change PIN**  
  - Flow: old PIN → new PIN → confirm new PIN.  
  - Will map to a `/auth/pin/change`-style endpoint.

- **8. Help**  
  - Text-only screens describing fees, limits, and support contacts.

***

## Idempotency and Audit Notes

- All USSD-initiated airtime and data-bundle requests must send `X-Idempotency-Key` and `idempotency_key` with the same value. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)
- USSD session ID should be used as `X-Correlation-ID` to allow traceability across logs and database records. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)
- External auditors can verify correctness by:

  - Sending duplicate USSD requests with the same idempotency key.  
  - Confirming responses reuse the same `ppay_ref`. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)
  - Checking `settlement_ledger` has one row per idempotency key. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/62026373/0d90fae2-5420-4583-b9cb-6460589c5cca/paste.txt)

***

You can copy this text into `docs/ussd-backend-v0.1.md` in `ppay-backend`. It gives auditors a clear view of:

- Menus inspired by MTN MoMo.  
- Exact mapping from USSD flows to your tested backend endpoints.  
- Idempotency behaviour you already proved with `generate_airtime_idem.js` and `generate_data_bundle_idem.js`.
