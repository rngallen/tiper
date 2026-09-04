# DFMS maintenance

How the system is meant to stay maintainable. Read this before adding a list, a form picker, or a table.

## Layout

| Path | Role |
|---|---|
| `apps/` | HTTP routers and request validation (Fiber) |
| `internal/` | Domain services (orders, inventory, billing, workflow, jobs) |
| `pkg/` | Shared libraries (response, db, audit, export, middleware) |
| `apps/models/` | GORM models — indexes and constraints live on the tags |
| `web/` | Next.js operator UI |
| `tools/migrate-fuel-delivery` | One-time Django → DFMS copy |

The application database is **MSSQL**. Sage 300 is a **second** MSSQL connection. Never create DFMS tables in the Sage company database.

## Requests

Bind JSON with `c.Bind().Body()` then jellyvalidator (`Sanitize` then `Validate`). Do not use `Bind().JSON()`.

UIDs in JSON are the public `id`. Integer primary keys stay off the wire.

## List APIs

Operational lists (ILR, ILO, ITT, receipts, billing runs, fee batches, …) always paginate. Never `Limit(200)` and dump.

Shared helper: `response.ServeList` in `pkg/response/ops.go`.

| Query | Behaviour |
|---|---|
| `page`, `pageSize` | Required for operational lists. Default lookback is **90 days**. |
| `fromDate`, `toDate` | Inclusive window on `DateColumn`. |
| `allDates=true` | Skip the date window (form pickers). |
| `search` | `LIKE` on a whitelist of columns (`response.ApplyLike`). |
| `orderBy`, `sortDirection` | Whitelist only (`ListOpts.Sort`). |
| `status` | Optional document status. `returned` maps to `rejected` on order lists. |
| `export=xlsx` | Same filter, Excel via `export.Query`. |

Response shape is always:

```json
{ "items": [], "page": 1, "pageSize": 10, "totalPages": 1, "itemsCount": 0 }
```

Master catalogues (`/api/v1/master/…`, badges, destinations) set `DumpIfNoPage: true`. Omitting `page` returns the full active set so small pickers still work. Prefer sending `page=1&pageSize=40&search=` from the UI.

Joined list queries must:

1. `Select("[Table].*")` so GORM does not collide columns.
2. Tie-break with `response.StableOrderTie` on `[Table].ID`.

Do not preload children that the list/inquiry does not show. Receipt details and billing run lines stay off the list query.

`GET /ic/balances` is a computed snapshot, not a paged document list. Workflow process definitions are small and may dump.

To add a new operational list:

1. Put a date + status composite index on the model (`idx_*List`).
2. Call `parseOps` + `ServeList` (leave `DumpIfNoPage` false).
3. Point `EnterpriseTable` at the path with `remote={{ path, dates: true }}`.

## Indexes and constraints

GORM tags on `apps/models` are the schema. `dfms migrate up` runs AutoMigrate on an empty database.

Every operational table needs:

- Unique `UID` and unique `DocumentNumber` where the document has a number.
- Index on the list date column and a composite `(Status, Date)` for filtered lists.
- FK columns indexed.
- Check constraints on status enums and non-empty codes.

Django integer PKs are **never** copied into FKs. Parents keep `DjangoID`; children resolve `WHERE DjangoID = old_id`. Django audit rows are not migrated. After a schema change, recreate the database and re-run `tools/migrate-fuel-delivery`.

## Content types

`pkg/types/contentType.go` — values are stable. **Do not renumber.**

- 1–36 reserved (auth, workflow, platform).
- Domain from 40.
- Orders / gantry from 80.

## Stock navigation

Operators work by location, not one mixed Transactions list:

| Area | Documents |
|---|---|
| Reception | Vessel receipts |
| Gantry | Internal loading (ILR), compartmentalization, amendments, gantry completion |
| Terminal | Pump-over request, pump-over report, ITT |

Pump-over **request** and pump-over **report** are separate documents with their own list, status pills, and approval inbox. Do not put both on one page.

## Frontend lists and forms

Every operational document uses the same shape: **list** (status pills, New, Submit, Open) → **`/new`** create form → **`/[id]`** document. Gantry completion is the exception (workbench on open ILO lines).

`EnterpriseTable` with `remote` fetches the server list, date range (default last 90 days), status pills, and Excel export. Do not pass `rowSearch` / `dateOf` on remote tables — the server already filters.

Form pickers:

- `AsyncSearchableSelect` + `searchMaster` for catalogues and document lookups.
- `QtyInput` for quantities (thousand grouping in the field, raw number to the API).
- `ErrorBanner` for API errors.
- Native `<Select>` only for small closed enums (route, tender, amendment kind).

`PICKER_QUERY` in `web/src/lib/list.tsx` is `{ page: 1, pageSize: 100, allDates: true }` for one-shot picker fetches.

ILR create/edit is the reference UX: customer name, AGO/PMS pairing, local vs transit destinations, truck combo plates, attachments with preview.

## Workflow and integrations

Every submit-for-approval document has a process (receipts, ILR, compartmentalization, amendments, pump-over, pump-over report, ITT, zerolization, billing runs, fee batches). Seeded steps have **no operator role**. On submit, empty steps skip until a step has users or the document reaches Approved. Later, assign a role to a step under Access → Workflows and put people in that role under Access → Users — no code change.

Gantry completion posts loaded quantity; it is not an approval document. FCF profiles, FX, and EWURA licenses are catalogues, not workflow documents.

Approvals run in `internal/workflow`. ATLAS NEO (ALMA) is a file share (`In` / `Alma/Files`), not a second app. EWURA NPGIS uses `NpgisSubmission` and job `ewura.npgis`. There is no RabbitMQ hop.

## Comments

Comment the *why*: content-type ranges, list-index purpose, bind/validate, Sage vs DFMS. Do not narrate every assignment.

## Checks

```bash
go test ./pkg/response/ ./apps/orders/ ./apps/inventory/ ./apps/billing/ ./apps/masterdata/
cd web && npx tsc --noEmit
```
