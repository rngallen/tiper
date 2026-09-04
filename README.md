# TIPER Depot Fuel Management System (DFMS)

One system for [Tanzania International Petroleum Reserves Limited](https://tiper.co.tz/) at Kigamboni. TIPER is a bonded warehouse — it does not buy fuel. OMCs store product after import and lift it by gantry or pump-over.

Stock, receipts, deliveries, ITT, billing, EWURA, approvals, and reports all live in this repository. There is no second order app and no message bus between systems.

Historical rows from the retired Django portal are copied once with `tools/migrate-fuel-delivery`. After that, operators work only here.

## Stack

- Go 1.27, Fiber v3, GORM
- **Microsoft SQL Server** application database
- Optional Sage 300 (second MSSQL connection — never DFMS tables in the Sage company DB)
- Next.js frontend in `web/`

## Quick start

```bash
cp .env.sample .env
# create empty MSSQL database DFMS
go run . migrate up
go run .

cd web && npm install && npm run dev
```

Default admin: `ngallen4@gmail.com` / `Admin@2026` (change immediately).

Windows Server: [HTTP, no TLS](docs/windows-deploy.md) or [HTTPS with certificates](docs/windows-deploy-tls.md). Keys use the `DFMS_` / `DFMS.` prefix (not `APP_`).

## Documentation

| Guide | Audience |
|---|---|
| [System architecture](docs/architecture.md) | How DFMS is built (API, UI, jobs, integrations) |
| [User guide](docs/user-guide.md) | Operators and approvers |
| [Maintenance](docs/maintenance.md) | Lists, indexes, and content types |

## Modules

| Area | Routes |
|---|---|
| Auth / users / roles / titles | `/api/v1/auth` |
| Workflow inbox | `/api/v1/workflow` |
| Master data | `/api/v1/master` |
| Stock, receipts, deliveries, ITT | `/api/v1/ic` |
| Billing (FCF / VCF / TBS / KOJ) | `/api/v1/billing` |
| EWURA licenses | `/api/v1/ewura` |
| Reports | `/api/v1/reports` |
| Settings | `/api/v1/settings` |

Gantry (ILR, compartmentalization, amendments, completion) and Terminal (pump-over, pump-over reports, ITT) are separate Stock areas. Request and execution report are never on the same page.

Fee prices live under **Billing › Setups** (FCF profiles, VCF / KOJ / TBS batches, FX). Billing runs only generate charges from those approved prices. EWURA licenses sit under **Stock › Setups** (customer / depot / destination registration).

Create/update bodies use `c.Bind().Body()` plus jellyvalidator (`Sanitize` then `Validate`). Prefer that over `Bind().JSON()`.

Operational lists always paginate (`response.ServeList`). Master catalogues dump when `page` is omitted. See [docs/maintenance.md](docs/maintenance.md) for list contracts, indexes, form pickers, and content-type rules.

### Gantry tables (fresh schema)

Django integer PKs are never copied into FKs. Parents store the old pk on `DjangoID`; children look up `WHERE DjangoID = old_id`. Django audit trails are not migrated.

| Django | DFMS |
|---|---|
| `GantryCompartmentalization` | `GantryCompartmentalization` — ILO FK is `IloID` (GantryLoadingLine). `TransactionID` is the shared NPGIS sequence. `PrintedAt` is gate-pass. `AlmaSentAt` / `AlmaFileName` / `NpgisSent` replace sent_at / file_name / sent_to_ewura. `DriverLicense` matches master data spelling. |
| `GantryCompartmentalizationLine` | `GantryCompartmentalizationLine` — `CompartmentalizationID`, capacity from calibration, unique seals. |
| `GantryLoading` ago_*/mogas_* | `GantryLoading` + `GantryLoadingProduct` (one child row per grade). |
| `GantryLoadingSummary` | Rebuilt from loadings after copy — not row-copied. |
| `GantryVesselLoading` | `GantryVesselLoading` — `posted` / Sage sequence numbers dropped. |
| `EwuraPetroleumLicences` | `EwuraPetroleumLicense` — copied in data migration, not on `migrate up`. |

Snapshots (customer code, product code, stock status code) are the DFMS master values (item number is the ALMA/Sage code), not Django integer codes.

Create compartmentalization: `POST /api/v1/orders/compartmentalizations` with `{ "iloId", "badgeId?", "confirmExpiry?" }`. 409 `nearExpiry` means the operator can Continue.

ATLAS NEO (ALMA) is a file share, not a second application:

| Folder | Direction | Format |
|---|---|---|
| `{alma.filePath}/In` | DFMS → gantry | SAP3C order (same layout as tiper-loadings) |
| `{alma.filePath}/Alma/Files` | gantry → DFMS | SAP3R result (same layout as alma-files) |
| `Alma/Archieved` / `Alma/Rejected` | after parse | dated archive |

EWURA NPGIS submissions sit in an outbox (`NpgisSubmission`) and flush on job `ewura.npgis`. There is no RabbitMQ hop.

Copy historical Django rows once. Parent stages stay serial (customers → transporters/drivers/trucks → ILR → ILO → comps → lines). Inside a large table the migrator warms unique keys, then uses batched `CreateInBatches` and a bounded worker pool (`-workers`, `-batch`). Master catalogues (users, products, statuses) stay sequential. Some tables exceed 300k rows.

```
go run ./tools/migrate-fuel-delivery -src "postgres://user:pass@host:5432/stock"
go run ./tools/migrate-fuel-delivery -src "..." -dry-run=false
go run ./tools/migrate-fuel-delivery -src "..." -dry-run=false -workers=8 -batch=200
```

## Commands

```
dfms run | console     start API
dfms migrate up        create schema + seed
dfms migrate status    check schema
dfms migrate reset     drop and recreate (dev only)
```
