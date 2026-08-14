/**
 * K6 Load Test — Full Buyer Flow
 *
 * Skenario: simulasi buyer lengkap:
 *   login → join queue → hold tiket → create order → get order
 *
 * Setup:
 *   - Server berjalan
 *   - Ada event + ticket type yang sudah di-provision cukup banyak
 *
 * Cara run:
 *   k6 run \
 *     -e TICKET_TYPE_ID=<uuid> \
 *     -e EVENT_ID=<uuid> \
 *     tests/k6/full_flow.js
 */

import http from "k6/http";
import { check, sleep, group } from "k6";
import { Rate, Trend } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TICKET_TYPE_ID = __ENV.TICKET_TYPE_ID;
const EVENT_ID = __ENV.EVENT_ID;

const loginSuccess = new Rate("login_success");
const holdSuccess = new Rate("hold_success");
const orderSuccess = new Rate("order_success");
const e2eLatency = new Trend("e2e_latency_ms");

export const options = {
  scenarios: {
    buyers: {
      executor: "constant-vus",
      vus: 20,
      duration: "30s",
    },
  },
  thresholds: {
    login_success: ["rate>0.99"],
    hold_success: ["rate>0.8"],
    http_req_duration: ["p(95)<3000"],
  },
};

export default function () {
  const startTime = Date.now();
  const email = `buyer_${__VU}_${__ITER}@test.com`;
  const password = "password123";
  let token = "";

  // 1. Register + Login
  group("auth", () => {
    http.post(
      `${BASE_URL}/api/v1/auth/register`,
      JSON.stringify({ email, name: `Buyer ${__VU}`, password }),
      { headers: { "Content-Type": "application/json" } }
    );

    const loginRes = http.post(
      `${BASE_URL}/api/v1/auth/login`,
      JSON.stringify({ email, password }),
      { headers: { "Content-Type": "application/json" } }
    );

    const ok = check(loginRes, {
      "login status 200": (r) => r.status === 200,
      "login has token": (r) => {
        try {
          return JSON.parse(r.body).token !== undefined;
        } catch {
          return false;
        }
      },
    });
    loginSuccess.add(ok);
    if (ok) token = JSON.parse(loginRes.body).token;
  });

  if (!token) return;

  sleep(0.2);

  // 2. Hold tiket
  let unitIDs = [];
  group("hold_ticket", () => {
    const holdRes = http.post(
      `${BASE_URL}/api/v1/tickets/hold`,
      JSON.stringify({
        items: [{ ticket_type_id: TICKET_TYPE_ID, quantity: 1 }],
      }),
      {
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
      }
    );

    const ok = check(holdRes, {
      "hold 200 or 409": (r) => r.status === 200 || r.status === 409,
    });
    holdSuccess.add(holdRes.status === 200);

    if (holdRes.status === 200) {
      try {
        unitIDs = JSON.parse(holdRes.body).unit_ids || [];
      } catch {}
    }
  });

  if (unitIDs.length === 0) return;

  sleep(0.2);

  // 3. Create order
  group("create_order", () => {
    const orderRes = http.post(
      `${BASE_URL}/api/v1/orders`,
      JSON.stringify({ event_id: EVENT_ID, unit_ids: unitIDs }),
      {
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
          "Idempotency-Key": `order-${__VU}-${__ITER}`,
        },
      }
    );

    const ok = check(orderRes, {
      "order created 201": (r) => r.status === 201,
    });
    orderSuccess.add(ok);
  });

  e2eLatency.add(Date.now() - startTime);
  sleep(1);
}
