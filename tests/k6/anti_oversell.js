/**
 * K6 Load Test — Anti-Oversell Engine
 *
 * Skenario: N virtual user berebut tiket yang stoknya terbatas.
 * Hanya sejumlah stok yang tersedia yang boleh berhasil di-hold.
 *
 * Setup sebelum run:
 * 1. Pastikan server berjalan di BASE_URL
 * 2. Set env vars: TICKET_TYPE_ID, BUYER_TOKEN
 *    TICKET_TYPE_ID = UUID ticket type yang sudah di-provision
 *    BUYER_TOKEN    = JWT token BUYER yang valid
 *
 * Cara run:
 *   k6 run -e TICKET_TYPE_ID=<uuid> -e BUYER_TOKEN=<token> tests/k6/anti_oversell.js
 *
 * Atau dengan lebih banyak VU:
 *   k6 run --vus 100 --duration 10s -e TICKET_TYPE_ID=<uuid> -e BUYER_TOKEN=<token> tests/k6/anti_oversell.js
 */

import http from "k6/http";
import { check, sleep } from "k6";
import { Counter, Rate } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TICKET_TYPE_ID = __ENV.TICKET_TYPE_ID;
const BUYER_TOKEN = __ENV.BUYER_TOKEN;

// Metrik custom
const holdSuccess = new Counter("hold_success");
const holdFailed = new Counter("hold_failed_no_stock");
const holdError = new Counter("hold_error");
const successRate = new Rate("hold_success_rate");

export const options = {
  scenarios: {
    spike: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "5s", target: 50 },   // ramp up ke 50 VU dalam 5 detik
        { duration: "10s", target: 50 },  // tahan 50 VU selama 10 detik
        { duration: "3s", target: 0 },    // ramp down
      ],
    },
  },
  thresholds: {
    // 95% request harus selesai < 800ms sesuai target SLA PRD-00
    http_req_duration: ["p(95)<800"],
    // Error rate harus < 5% (selain 409 yang memang expected saat tiket habis)
    http_req_failed: ["rate<0.05"],
  },
};

export default function () {
  if (!TICKET_TYPE_ID || !BUYER_TOKEN) {
    console.error("Set TICKET_TYPE_ID dan BUYER_TOKEN via -e flag");
    return;
  }

  const payload = JSON.stringify({
    items: [{ ticket_type_id: TICKET_TYPE_ID, quantity: 1 }],
  });

  const res = http.post(`${BASE_URL}/api/v1/tickets/hold`, payload, {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${BUYER_TOKEN}`,
    },
    timeout: "5s",
  });

  if (res.status === 200) {
    holdSuccess.add(1);
    successRate.add(true);
    check(res, {
      "hold response has unit_ids": (r) => {
        const body = JSON.parse(r.body);
        return Array.isArray(body.unit_ids) && body.unit_ids.length > 0;
      },
      "hold response has held_until": (r) => {
        const body = JSON.parse(r.body);
        return typeof body.held_until === "string";
      },
    });
  } else if (res.status === 409) {
    // Stok habis — ini expected behavior
    holdFailed.add(1);
    successRate.add(false);
    check(res, {
      "out-of-stock returns 409": (r) => r.status === 409,
    });
  } else {
    holdError.add(1);
    successRate.add(false);
    console.error(`Unexpected status ${res.status}: ${res.body}`);
  }

  sleep(0.1);
}

export function handleSummary(data) {
  const succeeded = data.metrics.hold_success
    ? data.metrics.hold_success.values.count
    : 0;
  const failed = data.metrics.hold_failed_no_stock
    ? data.metrics.hold_failed_no_stock.values.count
    : 0;
  const errors = data.metrics.hold_error
    ? data.metrics.hold_error.values.count
    : 0;

  console.log("\n========== ANTI-OVERSELL SUMMARY ==========");
  console.log(`Hold berhasil (200):         ${succeeded}`);
  console.log(`Hold gagal - stok habis (409): ${failed}`);
  console.log(`Error lain:                  ${errors}`);
  console.log(`Total request:               ${succeeded + failed + errors}`);
  console.log("============================================\n");

  return {
    stdout: JSON.stringify(data, null, 2),
  };
}
