# TIPER DFMS — system architecture

This document describes how the Depot Fuel Management System (DFMS) is built and how its parts work together. It is for implementers, operators who own the server, and anyone who must reason about a change.

TIPER (Tanzania International Petroleum Reserves Limited, Kigamboni) is a **bonded warehouse**. It does not buy fuel. Oil marketing companies (OMCs) store product after import and lift it by gantry or pump-over. DFMS is the single application for that work: stock, receipts, deliveries, in-tank transfers, billing, EWURA, approvals, and reports.

There is no second order app and no message bus between systems.

Related documents:

| Document | Audience |
|---|---|
| [User guide](user-guide.md) | Operators and approvers |
| [Windows deploy (HTTP)](windows-deploy.md) | Server install on the LAN |
| [Windows deploy (TLS)](windows-deploy-tls.md) | HTTPS with certificates |
| [Maintenance](maintenance.md) | Engineers adding lists, forms, or tables |
| [README](../README.md) | Quick start and module routes |

---

## 1. What the system does

DFMS records the depot’s operational truth and charges OMCs from approved price cards.

| Domain | Outcome |
|---|---|
| Master data | Customers, products, tanks, vessels, trucks, drivers, depots, EWURA licences |
| Reception | Internal and external vessel receipts (provision then final) |
| Gantry | Internal loading request (ILR) → loading order (ILO) → compartmentalization → completion. Request and execution never share a page |
| Terminal | Pump-over request, pump-over report, in-tank transfer (ITT), zerolization, financial hold |
| Inventory | Book stock by customer × product × vessel × status, with provision, hold, and reservations |
| Billing | FCF (fixed storage), VCF (variable / MI-loss), KOJ, TBS — prices on setups; runs generate charges |
| Workflow | Configurable multi-step approval with substitutes, quorum, return, and reject |
| Integrations | ATLAS NEO (ALMA) gantry files, EWURA NPGIS outbox, Sage 200 AR lookup, mail/SMS |
| Reports | PDF / Excel from the same filters operators use on lists |

Historical rows from the retired Django portal are copied **once** with `tools/migrate-fuel-delivery`. After that, operators work only here.

---

## 2. Runtime topology

Production on Windows Server (intranet) is typically:

```
  Browser
     │  http(s)://dfms
     ▼
  nginx  (LAN listener)
     ├── /        →  Next.js  127.0.0.1:3000
     └── /api/    →  dfms.exe 127.0.0.1:8080
                      │
                      ├── Microsoft SQL Server  (DFMS database)
                      ├── Sage 200 MSSQL        (optional second connection)
                      ├── SMTP / SMS
                      ├── ALMA file share       ({alma.filePath}/In and Alma/Files)
                      └── EWURA NPGIS HTTPS     (outbox job)
```

Fiber and Node bind to **loopback**. SQL Server credentials, PASETO keys, and SMTP passwords never sit in the nginx config.

Development: `go run .` (API) and `cd web && npm run dev` (UI). Keys use the `DFMS_` / `DFMS.` prefix.

The API process (`dfms.exe` / `dfms run`) is the composition root. It hosts HTTP, the workflow engine, inventory and billing services, the ALMA file watcher, and the cron job manager in one process.

---

## 3. Logical architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Next.js UI  (web/)                                             │
│  Session cookies · CSRF header · permission-filtered nav        │
└───────────────────────────────┬─────────────────────────────────┘
                                │  HTTPS JSON  /api/v1/…
┌───────────────────────────────▼─────────────────────────────────┐
│  Fiber HTTP  (apps/)                                            │
│  Bind → Sanitize → Validate → permission middleware             │
│  auth · workflow · master · inventory · orders · billing        │
│  ewura · reports · settings · sage · audit · public docs        │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│  Domain services  (internal/)                                   │
│  workflow engine · inventory ledger · orders · billing          │
│  jobs · notify · alma · ewura · sage · integrations             │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│  Shared libraries  (pkg/)                                       │
│  config · db · audit · response · migrate · crypto · docs       │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│  GORM models  (apps/models/)  — schema lives on struct tags     │
│  Microsoft SQL Server                                           │
└─────────────────────────────────────────────────────────────────┘
```

**HTTP routers do not contain business rules.** Routers bind and validate, then call services. Approval outcomes post stock and charges through **workflow hooks**, not through a second “post” screen.

---

## 4. Package layout

| Path | Role |
|---|---|
| `apps/` | Fiber routers and request schemas (auth, billing, inventory, masterdata, orders, reports, settings, workflow, audit, public, sage, ewura) |
| `apps/models/` | GORM models; indexes and check constraints on tags |
| `internal/` | Domain services (not importable from outside the module) |
| `pkg/` | Shared libraries (response, db, audit, middleware, types) |
| `web/` | Next.js App Router operator UI |
| `tools/migrate-fuel-delivery` | One-time Django → DFMS copy |
| `internal/app/run.go` | Composition root: DB, keys, services, hooks, jobs, HTTP |

The application database is **Microsoft SQL Server**. Sage 200 is a **second** MSSQL connection. DFMS tables are never created in the Sage company database.

`dfms migrate up` runs AutoMigrate and seed. Schema migrate/seed is CLI-only; the running API requires a ready schema (`migrate.RequireReady`).

---

## 5. Request and session model

### 5.1 API surface

Authenticated JSON lives under `/api/v1/…`. Operational lists always paginate (`response.ServeList`). Master catalogues dump the active set when `page` is omitted so small pickers still work.

Create/update bodies use `c.Bind().Body()` then jellyvalidator (`Sanitize` then `Validate`). Public IDs on the wire are ULIDs (`id`); integer primary keys stay off the JSON.

Health: `GET /healthz`, `GET /readyz` (DB ping). OpenAPI/Swagger is registered only when `DFMS.DEBUG` is true.

### 5.2 Authentication

1. `GET /api/v1/auth/csrf` — CSRF cookie for the browser.
2. `POST /api/v1/auth/login` — email + password.
3. `POST /api/v1/auth/mfa/verify` — one-time code (email always; SMS when the user has a phone).
4. HttpOnly **PASETO** access and refresh cookies.
5. `POST /api/v1/auth/refresh` and `POST /api/v1/auth/session/touch` keep the session alive.

Permission codes on the role gate every route and every nav leaf. Session version invalidation logs a user out after password or privilege changes.

Rate limiting (200 req/min per IP) applies except health checks and `/api/v1/public/`.

### 5.3 Public documents

Unsigned public URLs are not used. Gate passes and similar PDFs are verified with `pkg/docsig`:

`GET /api/v1/public/documents/:kind/:uid/:sig` and `…/pdf`.

---

## 6. Workflow engine

Package `internal/workflow` is a configurable state machine per **content type** (receipt, ILR, ITT, fee batch, billing run, …).

A **process** has **nodes** and **transitions**. Submitting a document creates a **process instance** and **tasks** for the operators of the current node (role, named users, or initiator pool).

| Outcome | Typical document status | Hook |
|---|---|---|
| Complete (quorum met) | `approved` | `OnComplete` — post stock, send ALMA, freeze prices |
| Soft reject | `returned` | `OnReject(total=false)` — initiator amends and resubmits |
| Total reject | `rejected` | `OnReject(total=true)` |
| Resubmit from returned | `submitted` | `OnResubmit` |

Substitutes cover an operator for a date range. Notifications go through `internal/notify` (outbox → mail/SMS job).

Hooks registered at startup (`internal/app/run.go`):

| Content type | Hook effect (summary) |
|---|---|
| Vessel receipt | Inventory receipt lines |
| Zerolization | Consolidate same-customer vessels |
| ITT | Move book stock between customers |
| Financial hold | Hold / release quantity |
| Gantry loading request | Reserve stock; later ALMA order |
| Pump-over request / report | Reserve then execute pipeline delivery |
| Compartmentalization | Build dispatch file / seals |
| Order amendment | Quantity, product, or cancel |
| Fee batches and billing runs | Mark approved; billing uses those prices |

---

## 7. Inventory ledger

`internal/inventory` is the book-stock service. Quantity is stored as ledger events and rolled into `StockBalance` snapshots: customer × product × vessel × stock status, with provision vs final, financial hold, and reservations from open orders.

**Free to order** is final quantity minus hold minus reserved.

Receipts are provision then final. ITT moves ownership without a physical pump. Zerolization consolidates remaining quantity onto one vessel for the same customer. Financial hold blocks lifting until released.

Balances and movements are read APIs (`GET /api/v1/ic/balances`, movements). They are computed snapshots, not paged document lists.

---

## 8. Orders (gantry and terminal)

`internal/orders` owns ILR, ILO, compartmentalization, amendments, pump-over, and expiry.

Typical gantry path:

```
ILR (request, approval)
  → ILO (loading order, reserved stock)
    → Compartmentalization (truck, seals, ALMA SAP3C)
      → ALMA SAP3R result (watcher)
        → Gantry completion
```

Request screens and execution/report screens are separate. Pump-over follows the same idea: request (DR) then report.

A midnight job (`orders.expire`) expires ILOs whose expiration date has been reached and releases reservations. Compartmentalization refuses an expired ILO and warns when expiry is within three days. Driver licence and truck calibration expiry are gated the same way.

---

## 9. Billing

Prices live under **Billing › Setups**. Billing **runs** only generate charges from **approved** cards.

| Card | Role |
|---|---|
| FCF | Fixed storage (first / nth days, flat or tier, parcel quantity) |
| MI-loss | Product × contract loss fraction (0–1 exclusive). Can remain in force for a year |
| Variable fee (VCF) | Monthly EWURA price and density; MI-loss must already be **effective** on the VCF effective date (same day or earlier). Document date is not the cutoff — batches are created before they take effect |
| KOJ / TBS | Jetty and TBS fee cards |
| FX | USD/TZS (and other) quotes used to home-price cards |

Jobs:

| Job | Default (cron with seconds) | Purpose |
|---|---|---|
| `billing.nth` | 02:30 daily | Nth-day FCF for parcels still on hand |
| `billing.tbs` | 00:15 daily | Previous day’s TBS |
| `billing.vcf` | 03:30 daily | Monthly variable-fee run when due |

Change of service switches a parcel’s delivery method; it is not a fee-card switch.

---

## 10. Integrations

Configuration is stored sealed in the application database (`internal/integrations`) and edited under **Settings**. Secrets are not kept in the Next.js bundle.

### ATLAS NEO (ALMA)

A file share, not a second application. DFMS writes and reads the files itself (no RabbitMQ).

| Folder | Direction | Format |
|---|---|---|
| `{alma.filePath}/In` | DFMS → gantry | SAP3C order (same layout as tiper-loadings) |
| `{alma.filePath}/Alma/Files` | gantry → DFMS | SAP3R result |
| `Alma/Archieved` / `Alma/Rejected` | after parse | dated archive |

`internal/alma` watches inbound files; `orderSvc.SetAlmaRoot` writes outbound orders when compartmentalization is approved.

### EWURA

- Licence sync job (`ewura.licenses`) refreshes petroleum licences used at customer/depot/destination setup.
- NPGIS submissions sit in `NpgisSubmission` and flush on job `ewura.npgis`.

### Sage 200

Optional second MSSQL connection. Used to look up AR clients and currencies (`internal/sage`, `GET /api/v1/sage/…`). DFMS never writes its schema into the Sage company database.

### Mail and SMS

OTP, workflow tasks, and informational notices go through `internal/notify` into an outbox. Job `notify.outbox` delivers. If mail/SMS is disabled, OTP may be logged so first bootstrap can complete.

---

## 11. Background jobs

`internal/jobs` binds named functions to cron specs from Settings → Operations → Schedules.

| Job | Default | Purpose |
|---|---|---|
| `logs.rotation` | midnight | Rotate application / access / DB logs |
| `ewura.licenses` | 02:00 | Sync EWURA licences |
| `ewura.npgis` | every minute | Flush NPGIS outbox |
| `billing.nth` | 02:30 | Nth-day FCF |
| `billing.tbs` | 00:15 | Daily TBS |
| `billing.vcf` | 03:30 | Monthly VCF |
| `orders.expire` | 00:05 | Expire ILOs |
| `notify.outbox` | every minute | Send queued mail/SMS |

---

## 12. Frontend

Next.js App Router in `web/`. Navigation is permission-filtered (`web/src/lib/navTree.ts`) from two workspace modules:

- **Stock** — Reception, Gantry, Terminal, Reports, Setups
- **Billing** — Transactions, Reports, Setups

Approvals are one inbox per seeded process (not a mixed queue). Lists use a shared enterprise table: date window (default 90 days), status, search, sort, Excel export.

The dashboard is a user-configurable widget layout (KPIs, stock, receipts, throughput, workflow aging, open orders).

---

## 13. Security notes

- HttpOnly PASETO cookies; CSRF on mutating requests; SameSite Lax.
- Permission middleware on every privileged route; nav hides what the role cannot open.
- Secrets in config / sealed integration rows, not in the SPA.
- Audit recorder (`pkg/audit`) writes who changed what; UI under **Audit**.
- Trust forwarded-for only when `TrustForwardedFor` is on (behind nginx).
- Public PDF links require a document signature, not a guessable UID.

---

## 14. Data and migration

GORM tags on `apps/models` **are** the schema. After a breaking schema change, recreate the database and re-run `tools/migrate-fuel-delivery` if historical Django rows are still required.

Django integer PKs are never copied into foreign keys. Parents store `DjangoID`; children resolve `WHERE DjangoID = old_id`. Django audit trails are not migrated.

Content-type numbers in `pkg/types/contentType.go` are stable. Do not renumber.

---

## 15. Commands

```
dfms run | console     start API (HTTP + jobs)
dfms migrate up        create schema + seed
dfms migrate status    check schema
dfms migrate reset     drop and recreate (dev only)
```

One-time historical copy:

```
go run ./tools/migrate-fuel-delivery -src "postgres://user:pass@host:5432/stock"
go run ./tools/migrate-fuel-delivery -src "..." -dry-run=false
```
