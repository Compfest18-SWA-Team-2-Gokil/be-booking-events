/**
 * K6 — War Ticket Simulation
 *
 * Skenario: simulasi "perang tiket" realistis:
 *   1. Setiap VU register akun unik
 *   2. Login → dapat token
 *   3. Semua VU hit /tickets/hold serentak di fase spike
 *
 * Cara run (local, 100 VU — default):
 *   k6 run -e TICKET_TYPE_ID=<uuid> tests/k6/war_ticket.js
 *
 * Cara run (200 VU):
 *   k6 run -e TICKET_TYPE_ID=<uuid> -e MAX_VUS=200 tests/k6/war_ticket.js
 *
 * Catatan lokal vs produksi:
 *   - Server lokal (MacBook, single process) comfortable di ~100-150 VU.
 *   - Server produksi (multi-instance) bisa 500+ VU.
 *   - Gunakan Go goroutine test (TestWarTicket_1000) untuk burst serentak 1000 user.
 */

import http from "k6/http";
import { check, sleep, group } from "k6";
import { Counter, Rate, Trend } from "k6/metrics";
import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";

const BASE_URL    = __ENV.BASE_URL      || "http://localhost:8080";
const TICKET_TYPE_ID = __ENV.TICKET_TYPE_ID;
const MAX_VUS     = parseInt(__ENV.MAX_VUS || "100");

// ── Custom metrics ──────────────────────────────────────────────────────────
const regSuccess   = new Rate("war_register_success");
const loginSuccess = new Rate("war_login_success");
const holdSuccess  = new Counter("war_hold_success");   // HTTP 200
const holdNoStock  = new Counter("war_hold_no_stock");  // HTTP 409
const holdError    = new Counter("war_hold_error");     // lain-lain / timeout
const holdLatency  = new Trend("war_hold_latency_ms", true);
const e2eLatency   = new Trend("war_e2e_latency_ms", true);

// ── Skenario: tiap VU jalankan tepat 1 iterasi (burst sekali, mirip goroutine) ──
// Gunakan MODE=sustained untuk sustained load (VU loop terus).
const MODE = __ENV.MODE || "burst";

export const options = {
  scenarios: {
    war: MODE === "sustained"
      ? {
          // Sustained: VU terus loop, cocok untuk stress test throughput
          executor: "ramping-vus",
          startVUs: 0,
          stages: [
            { duration: "10s", target: MAX_VUS },
            { duration: "20s", target: MAX_VUS },
            { duration: "5s",  target: 0 },
          ],
          gracefulRampDown: "10s",
        }
      : {
          // Burst (default): tiap VU tepat 1 kali register→login→hold
          // Hasilnya setara dengan goroutine test: N user serentak, 1 hold masing-masing
          executor: "per-vu-iterations",
          vus: MAX_VUS,
          iterations: 1,
          maxDuration: "120s",
        },
  },
  thresholds: {
    // Hold p95 < 5 detik untuk lokal (burst mode); sustained wajar lebih tinggi
    war_hold_latency_ms: ["p(95)<5000"],
    // Error rate (timeout/5xx) < 10%
    http_req_failed: ["rate<0.10"],
    // Login harus hampir selalu berhasil
    war_login_success: ["rate>0.95"],
  },
};

export default function () {
  if (!TICKET_TYPE_ID) {
    console.error("Set TICKET_TYPE_ID via -e flag, contoh: k6 run -e TICKET_TYPE_ID=<uuid> ...");
    return;
  }

  const e2eStart = Date.now();
  const email    = `war_${__VU}_${__ITER}_${uuidv4().slice(0, 8)}@k6.test`;
  const password = "password123";
  let token      = "";

  // ── Phase 1: Register ────────────────────────────────────────────────────
  group("register", () => {
    const res = http.post(
      `${BASE_URL}/api/v1/auth/register`,
      JSON.stringify({ email, name: `WarUser${__VU}`, password }),
      { headers: { "Content-Type": "application/json" }, timeout: "15s" }
    );
    regSuccess.add(res.status === 201 || res.status === 409);
  });

  // ── Phase 2: Login ──────────────────────────────────────────────────────
  group("login", () => {
    const res = http.post(
      `${BASE_URL}/api/v1/auth/login`,
      JSON.stringify({ email, password }),
      { headers: { "Content-Type": "application/json" }, timeout: "15s" }
    );
    const ok = check(res, { "login 200": (r) => r.status === 200 });
    loginSuccess.add(ok);
    if (ok) {
      try { token = JSON.parse(res.body).token; } catch {}
    }
  });

  if (!token) return;

  // Sedikit jeda agar semua VU sudah login sebelum spike hold dimulai.
  sleep(0.2);

  // ── Phase 3: Hold tiket — inti "perang" ─────────────────────────────────
  group("hold_ticket", () => {
    const start = Date.now();
    const res = http.post(
      `${BASE_URL}/api/v1/tickets/hold`,
      JSON.stringify({ items: [{ ticket_type_id: TICKET_TYPE_ID, quantity: 1 }] }),
      {
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        timeout: "30s",  // lebih panjang agar tidak false-fail saat beban tinggi
      }
    );
    holdLatency.add(Date.now() - start);

    if (res.status === 200) {
      holdSuccess.add(1);
      check(res, {
        "hold has unit_ids": (r) => {
          try { return JSON.parse(r.body).unit_ids?.length > 0; } catch { return false; }
        },
      });
    } else if (res.status === 409) {
      holdNoStock.add(1);
    } else {
      holdError.add(1);
      if (res.status !== 0) {  // 0 = timeout, sudah tercatat di http_req_failed
        console.error(`VU${__VU}: hold status=${res.status} body=${res.body?.slice(0, 120)}`);
      }
    }
  });

  e2eLatency.add(Date.now() - e2eStart);
}

export function handleSummary(data) {
  const success = data.metrics.war_hold_success?.values?.count  ?? 0;
  const noStock = data.metrics.war_hold_no_stock?.values?.count ?? 0;
  const errors  = data.metrics.war_hold_error?.values?.count    ?? 0;
  const p95     = data.metrics.war_hold_latency_ms?.values?.["p(95)"]?.toFixed(0) ?? "?";
  const p99     = data.metrics.war_hold_latency_ms?.values?.["p(99)"]?.toFixed(0) ?? "?";
  const rps     = data.metrics.http_reqs?.values?.rate?.toFixed(1) ?? "?";
  const loginOk = ((data.metrics.war_login_success?.values?.rate ?? 0) * 100).toFixed(1);

  console.log("\n╔══════════════════════════════════════════╗");
  console.log("║          WAR TICKET REPORT (K6)          ║");
  console.log("╠══════════════════════════════════════════╣");
  console.log(`║  Max VUs            : ${String(MAX_VUS).padStart(6)}               ║`);
  console.log(`║  Login success      : ${String(loginOk+"%").padStart(6)}               ║`);
  console.log("╠══════════════════════════════════════════╣");
  console.log(`║  Dapat tiket (200)  : ${String(success).padStart(6)}               ║`);
  console.log(`║  Kehabisan (409)    : ${String(noStock).padStart(6)}               ║`);
  console.log(`║  Error/Timeout      : ${String(errors).padStart(6)}               ║`);
  console.log("╠══════════════════════════════════════════╣");
  console.log(`║  Hold p95 latency   : ${String(p95+"ms").padStart(8)}             ║`);
  console.log(`║  Hold p99 latency   : ${String(p99+"ms").padStart(8)}             ║`);
  console.log(`║  Throughput         : ${String(rps+" req/s").padStart(12)}         ║`);
  console.log("╚══════════════════════════════════════════╝\n");

  return { stdout: JSON.stringify(data, null, 2) };
}
