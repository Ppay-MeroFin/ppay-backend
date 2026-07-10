// generate_airtime_idem.js

const http = require("http");

const API_HOST = "localhost";
const API_PORT = 8080;

// Matches mux.HandleFunc("/tx/airtime", h.AirtimeHandler) in cmd/api/main.go
const AIRTIME_PATH = "/tx/airtime";

function makeRequest(body, idemKey) {
  return new Promise((resolve, reject) => {
    const data = JSON.stringify(body);

    const options = {
      hostname: API_HOST,
      port: API_PORT,
      path: AIRTIME_PATH,
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Content-Length": Buffer.byteLength(data),
        // API expects idempotency key here:
        "X-Idempotency-Key": idemKey,
      },
    };

    const req = http.request(options, (res) => {
      let buf = "";
      res.on("data", (chunk) => {
        buf += chunk;
      });
      res.on("end", () => {
        resolve({ status: res.statusCode, body: buf });
      });
    });

    req.on("error", (err) => {
      console.error("HTTP request error:", err.message);
      reject(err);
    });

    req.write(data);
    req.end();
  });
}

async function main() {
  // Idempotency key we will reuse for two requests
  const idem = "idem-dup-test-001";

  const payload = {
    network: "MTN",
    currency: "SSP",
    to_account: "22222222-2222-2222-2222-222222222222",
    from_account: "11111111-1111-1111-1111-111111111111",
    amount_minor: 100,
    phone_number: "+211912345678",
    product_type: "airtime",
    idempotency_key: idem,
    correlation_id: "corr-dup-test-001",
  };

  console.log("Sending first airtime request...");
  try {
    const res1 = await makeRequest(payload, idem);
    console.log(
      "First response:",
      res1.status,
      (res1.body || "").slice(0, 120),
      "..."
    );
  } catch (err) {
    console.error("First request error:", err.message);
  }

  console.log("Sending second airtime request (same idempotency_key)...");
  try {
    const res2 = await makeRequest(payload, idem);
    console.log(
      "Second response:",
      res2.status,
      (res2.body || "").slice(0, 120),
      "..."
    );
  } catch (err) {
    console.error("Second request error:", err.message);
  }

  console.log("Done sending duplicate requests with idempotency_key =", idem);
}

main().catch((e) => console.error("Script error:", e));