# TIPER DFMS — user guide

This guide is for people who work in the Depot Fuel Management System every day: jetty and gantry clerks, terminal operators, finance, approvers, and administrators.

You do not need to know how the server is built. For that, see the [system architecture](architecture.md).

---

## 1. What this system is for

TIPER stores other companies’ fuel. DFMS is where you:

- Keep customer, truck, driver, tank, and vessel records
- Record vessel receipts
- Raise and complete gantry loadings and pump-overs
- Move stock between customers (in-tank transfer) or consolidate vessels (zerolization)
- Approve documents that change stock or prices
- Maintain fee cards and generate billing
- Print reports and gate documents

Menus you see depend on your **role**. If a screen is missing, you do not have permission — ask an administrator; do not share someone else’s login.

---

## 2. Sign in

1. Open the DFMS address your IT team gave you (for example `http://dfms`).
2. Enter your **email** and **password**.
3. Enter the **one-time code** sent to your email. If your profile has a phone number, the same code is also sent by SMS.
4. If you are asked to change your password (first login or reset), you are taken to **Profile → Password**. Change it before you continue.

The code expires. Use **Resend** only after the countdown finishes.

If the page says CSRF failed, refresh and sign in again. If you are idle too long, you will be signed out — that is normal.

**Never share your password or OTP.** Approvals and stock movements are recorded against your name.

---

## 3. Finding your way

The left menu is grouped like this:

| Area | Typical work |
|---|---|
| **Dashboard** | Snapshot of operations, stock, receipts, open orders, and waiting approvals. You can add or rearrange widgets. |
| **All reports** | PDF and Excel reports (period, preview, download). |
| **Audit** | Who changed what (if you have access). |
| **Approvals** | Separate inboxes for each document type waiting on **you**. |
| **Stock** | Reception, gantry, terminal, stock reports, and stock setups. |
| **Billing** | Billing runs, change of service, billing reports, and fee setups. |
| **Access** | Users, roles, titles, workflow definitions (administrators). |
| **Settings** | Company, mail, Sage, EWURA, schedules (administrators). |

Your name (top of the screen) opens **Profile**: appearance, table preferences, password, and **substitutes** (who may approve for you while you are away).

Lists usually show the last **90 days**. Widen the dates, search, sort columns, or export Excel from the same filters. Click a document number to open it.

---

## 4. How documents work

Most operational documents follow the same life:

| Status | Meaning | What you can do |
|---|---|---|
| **Draft** | Not yet sent | Edit header and lines, attach files, then **Submit** |
| **Submitted** | In approval | Approvers act; originator usually cannot edit lines |
| **Approved** | Finished the workflow | Stock or prices take effect as designed for that document |
| **Returned** | Sent back for correction | Fix what was asked, then submit again |
| **Rejected** | Stopped for good | Do not reuse it; raise a new document if needed |
| **Posted** | Used where billing/stock posting is a separate step | Treat as closed unless your procedure says otherwise |

**Empty documents cannot be submitted.** Add at least one line (product, rate, or quantity) first.

Request and execution are **never on the same page**. Example: an internal loading request is not the gantry completion screen; a pump-over request is not the pump-over report.

Attach supporting files on the document’s **Files** tab while it is still draft or returned.

---

## 5. Approvals

Open **Approvals** and pick the queue (vessel receipts, internal loading, compartmentalization, and so on). You only see work assigned to your step.

On a task you typically:

- **Approve** (or complete your step — some processes need more than one person)
- **Return** with a comment so the originator can correct it
- **Reject** when the document must not continue

Read the header, lines, and attachments before you act. Comments are part of the record.

If you will be away, set a **substitute** on your profile for the relevant workflow and dates. Your delegate receives those tasks and is told they are covering for you.

---

## 6. Stock setups (do this before operations)

Under **Stock › Setups**, keep masters current. Operations pick from these lists.

| Screen | Use it for |
|---|---|
| Customers | OMCs who store product. Link Sage billing accounts if finance uses them. |
| EWURA licenses | Customer / depot / destination registration with the regulator. |
| Suppliers | Parties on receipts where needed. |
| Products and categories | AGO, PMS, and other grades. |
| Tanks | Shore tanks and capacities. |
| Vessels | Ships that discharge at TIPER. |
| Depots | TIPER and pump-over destinations. |
| Trucks | Horse and trailer, calibration charts (expiry is checked at compartmentalization). |
| Drivers | Licence expiry is checked at compartmentalization. |
| Haulers | Transporters used at the gantry. |
| Destinations and districts | Transit destinations. |
| Stock statuses | Local, transit, mining, and similar. |

If a truck or driver is missing or expired, gantry dispatch will stop you. Fix the master, then continue.

---

## 7. Reception (vessel receipts)

**Stock › Reception**

| Screen | When to use |
|---|---|
| Internal receipts | Product received into TIPER tanks. |
| External receipts | Receipts that belong on the external list (other terminals / share reporting). |

Typical path:

1. Create the header (vessel, customer, route, dates).
2. Add parcel lines (product, quantities, status).
3. Attach survey or supporting files.
4. Submit for approval.

Receipts often go **provision** first, then **final** when figures are confirmed. Approved receipts update book stock. Until then, do not expect balances to move.

---

## 8. Gantry (road loading)

**Stock › Gantry**

Keep the steps in order. Do not skip to completion.

### 8.1 Internal loading (ILR)

Raise the **request**: customer, product, quantity, destination, truck/driver as required by the form.

Submit. After approval the system holds (reserves) stock so it cannot be promised twice.

Watch the **expiry date**. Overnight, expired loading orders are closed and reservations are released. Extend the order if the truck will load later.

### 8.2 Compartmentalization

When the truck is at the gantry, open **Compartmentalization** from the approved loading order.

The system checks:

- The loading order is not expired (and warns if it expires within three days — you may continue after confirming)
- Driver licence is valid
- Truck calibration is valid

Enter compartments and seals. After approval, DFMS sends the order to the gantry automation (ALMA). Do not invent a second loading file outside DFMS.

### 8.3 Amendments

Use **Amendments** to change quantity or product, or to cancel, on an order that is already in progress. Amendments have their own approval. Do not silently edit an approved request.

### 8.4 Gantry completion

**Gantry completion** is the execution side: recorded loaded quantities after the gantry result is in. It is not the same screen as the original request.

Print **gate-in**, **gate-out**, and **delivery note** from the document or from **Reports** when your procedure requires a signed paper. Visitors can confirm a signed PDF via the QR / public link on the document — they do not log in.

---

## 9. Terminal

**Stock › Terminal**

| Screen | Meaning |
|---|---|
| Pump-over | Pipeline delivery **request** (DR) to another depot/terminal. |
| Pump-over reports | What was **actually** pumped. Create this after the request is approved and the move happened. |
| In-tank transfers | Ownership change between two customers with no physical pump. |
| Zerolization | Same customer: remaining quantity on one vessel moved onto another (clean-up). |
| Financial hold | Quantity that must not be lifted until finance/operations release it. |

ITT and hold change book stock only after approval. Pump-over stock moves when the **report** (execution) is approved, not when the request is raised.

---

## 10. Stock balances and movements

**Stock › Reports**

- **Stock balances** — book stock by customer, product, vessel, and status, including provision, hold, reserved, and free to order.
- **Movements** — the ledger behind those balances.
- **Report catalog** — the same stock PDFs as under All reports, filtered to stock.

**Free to order** is what you may still load: final stock minus financial hold minus open reservations.

If a balance looks wrong, check: is the receipt approved? Is there a hold? Is an ILR still open (reserved)? Use movements and the document number rather than adjusting by memory.

---

## 11. Billing setups (prices)

**Billing › Setups** — cards that **must be approved** before a billing run can charge them.

Work in this spirit:

1. Create the **header** (date, effective from, description).
2. Add **lines**.
3. Submit for approval.

### 11.1 Exchange rates

Keep an approved USD/TZS (and other) quote. Fee cards that convert currencies use the rate on the card (you may override only when the form allows a manual rate).

### 11.2 MI-loss

One product appears **once** on a batch (one contract). Enter the loss as a **percent** on the form; the system stores it as a fraction (0.5% → 0.005). It must be greater than 0% and less than 100%.

MI-loss can stay in force for a long time (for example a year) while **prices change monthly**.

### 11.3 Variable fees (VCF)

Monthly EWURA price and density per product.

You must pick an **approved MI-loss batch**. Products on the VCF can only come from that batch; contract and MI-loss % are copied from it.

Matching uses **effective date**, not the day the batch was typed in. Batches are always created **before** they become effective.

- Allowed: MI-loss **Effective from** is the same day as, or earlier than, the variable fee **Effective from** (January MI-loss on a September price card is normal).
- Not allowed: MI-loss that only becomes effective **after** the variable fee effective date.

If you change the MI-loss source on a draft VCF, products on the card are cleared — add them again from the new batch.

### 11.4 FCF, KOJ, TBS

Fixed storage, Kurasini Oil Jetty, and TBS fee cards. Fill every required field on a line. Empty batches cannot be submitted.

FCF uses **parcel** quantity (not a whole-ship switch).

### 11.5 Catalogues

Tenders, discharge routes, delivery methods, procurement, contract types, pricing natures, and billing cycles are lookup lists used on receipts and fee lines. Keep codes stable; operators pick them by name.

---

## 12. Billing transactions

**Billing › Transactions**

| Screen | Meaning |
|---|---|
| Billing runs | Charges generated from **approved** price cards (FCF, VCF, KOJ, TBS as applicable). Review, then submit the run for approval. |
| Change of service | Switch a parcel’s **delivery method**. This is not switching which fee card applies. |

Scheduled jobs also create due nth-day FCF, daily TBS, and monthly VCF runs. You still review and approve those runs in the inbox.

If a run is empty or missing a product, check that the price card is **approved** and **effective** for that day, and that the parcel has quantity.

---

## 13. Reports

**All reports** (or Stock / Billing report catalogues):

1. Choose the report.
2. Set the period or the document number.
3. Preview the PDF.
4. Download PDF or export Excel where offered.

Useful groups:

- **Stock** — position, EOM, statements, aging, ullage, EWURA/PBPA weekly
- **Reception** — SBM/KOJ receipts and market share
- **Gantry** — daily/monthly loading, seals, open orders, ILR status
- **Billing** — run summary and lists by product or class of trade
- **Documents** — print one ILR, delivery note, gate pass, pump-over DR

Printed operational documents can include a check link so a gate officer confirms the paper without logging in.

---

## 14. Dashboard

**Dashboard** shows KPIs, book stock, receipts by route, daily throughput, approvals waiting, and open orders.

Use a **preset** (Operations, Reception, …) or add/remove widgets. Layout is saved on your user, not for everyone.

---

## 15. Access (administrators)

**Access**

| Screen | Use |
|---|---|
| Users | Create accounts, assign roles, deactivate people who left. |
| Roles | Permission sets. Menus and API follow these codes. |
| Titles | Job titles on the user profile. |
| Workflows | Steps, who acts, and who is in the initiator pool. Change with care — in-flight documents keep their instance. |

After a password or role change, the user may need to sign in again.

---

## 16. Settings (administrators)

**Settings**

- **Company** — name, letterhead attachments, currencies.
- **Notifications** — mail and SMS. Login codes and approval mail use these.
- **Sage** — connection to Sage 200 for customer/currency lookup. DFMS data stays in the DFMS database.
- **Session** — idle timeout.
- **EWURA** — NPGIS and licence sync.
- **Precision** — decimal display for quantities and money.
- **Schedules** — when nightly billing, expiry, EWURA, and mail jobs run.

If OTP never arrives, check mail settings (or the application log on a new install). Do not turn off MFA.

---

## 17. Everyday problems

| What you see | What to try |
|---|---|
| Menu item missing | You lack permission. Ask an administrator — do not borrow a login. |
| Cannot submit | Add lines; required dates and masters must be filled. Read the red message. |
| Cannot add a VCF product | Select an MI-loss batch first. Only products on that batch appear. |
| MI-loss not in the VCF list | Its **effective from** is after the variable fee effective date. Use an older MI-loss or change the VCF effective date. |
| Cannot compartmentalize | ILO expired, driver licence expired, or truck calibration expired. Extend or update masters. |
| Stock not updating | Document is not approved yet, or you are looking at the wrong customer/vessel/status. |
| “CSRF validation failed” | Refresh the page and retry. |
| Signed out | Idle timeout or password/role change. Sign in again. |
| OTP not received | Check spam; wait for resend; ask IT to confirm mail/SMS. |

Error messages appear in a dialog. Success appears briefly as a toast. Read the dialog before retrying — submitting twice can create duplicates on some screens.

---

## 18. Good practice

- Work in **your** queue and **your** documents.
- Submit only complete, checked figures.
- Put comments on returns so the next person knows what to fix.
- Keep truck calibration, driver licences, and EWURA licences current **before** the truck arrives.
- Do not treat document **date** (when you typed the card) as the same as **effective from** (when it applies).
- For prices: approve MI-loss, then monthly VCF, then expect billing runs to use those cards.

If something still looks wrong after these checks, give IT the **document number**, the **time**, and a screenshot of the message — not your password.
