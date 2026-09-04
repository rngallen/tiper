# TIPER DFMS — Windows Server deployment (intranet, no TLS)

Operators open `http://dfms` (or the server name / LAN IP). There is **no public Internet**, **no TLS certificates**, and **nginx** is the only process that listens on the LAN.

| Guide | When to use |
|---|---|
| **This file** | TIPER LAN only. No certificates. Port 80. |
| [windows-deploy-tls.md](windows-deploy-tls.md) | Internal CA or purchased certificate. Port 443. Required if the host is reachable outside the LAN. |

```
  browser  ──http://dfms:80──►  nginx
                                  ├── /           →  Next.js  127.0.0.1:3000
                                  └── /api/       →  dfms.exe 127.0.0.1:8080
                                                    (SQL Server on this host)
```

Fiber and Node bind to **loopback only**. SQL Server, PASETO keys, and SMTP passwords never sit in the nginx config or next to the web files.

Config keys are **`DFMS.`** in files and **`DFMS_`** in the process environment (`DFMS_SYMMETRIC_KEY`, not `APP_SYMMETRIC_KEY`). Another product on the same server can use its own prefix (`ABC_…`) without colliding.

---

## 1. What you need

| Piece | Role |
|---|---|
| Windows Server 2019+ (x64) | Host |
| SQL Server (same machine is fine) | Application database `DFMS` |
| [nginx for Windows](https://nginx.org/en/docs/windows.html) | Reverse proxy on port 80 |
| Node.js 20.9+ LTS | `next start` for the UI |
| `dfms.exe` | API + background jobs (Windows service `TIPERDFMS`) |
| NSSM or WinSW | Run nginx and Node as services |

Build the API on a machine with Go 1.27+:

```bat
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-X dfms/pkg/winsvc.Version=1.0.0" -o dfms.exe .
```

Build the UI (from `web/`):

```bat
cd web
npm ci
npm run build
```

Copy to the server, for example:

```
C:\dfms\dfms.exe
C:\dfms\web\          (package.json, .next, node_modules, public)
C:\nginx\
C:\ProgramData\DFMS\  (config + secrets — see §3)
```

Create an empty SQL database named `DFMS` (or whatever you set in config). Then, from an elevated command prompt **in `C:\dfms`**:

```bat
dfms.exe migrate up
```

---

## 2. Why not HTTPS, and what that implies

`DFMS.ALLOW_INSECURE_HTTP=true` is required so login cookies (`dfms_access`, `dfms_refresh`, `dfms_csrf`) are sent over plain HTTP. `DFMS.DEBUG` must stay **false** in production.

This is acceptable **only** because:

- The host is on the TIPER LAN, not reachable from the Internet.
- nginx is the only inbound port (80). 3000, 8080, and 1433 stay on localhost / host firewall deny.
- Operators are on a trusted network.

If this host is ever published outside the LAN, use [windows-deploy-tls.md](windows-deploy-tls.md). Do not leave `ALLOW_INSECURE_HTTP` on.

CORS / CSRF origins must match **exactly** what operators type in the address bar (scheme + host + port). List every variant they will use:

```
DFMS.CORS=http://dfms,http://dfms.tiper.local,http://10.20.30.40
```

No trailing slash. If someone opens `http://10.20.30.40` and that IP is missing from `DFMS.CORS`, login returns 403 CSRF.

---

## 3. Secrets — Windows service environment (recommended)

Do **not** put `DFMS.SYMMETRIC_KEY`, `DFMS.REFRESH_KEY`, `DFMS.MFA_KEY`, or `DFMS.DB.PASSWORD` in a `.env` next to `dfms.exe`. That folder is copied, backed up, and the first place anyone looks.

Those four values encrypt PASETO tokens (changing a key **logs everyone out**) and the SMTP / SMS / Sage passwords in Settings (`enc:v1:…` in SQL). Losing `DFMS.SYMMETRIC_KEY` makes those Settings secrets unreadable.

**Recommended: the `TIPERDFMS` service `Environment` value.** SCM injects it only into this service — nginx, Node, and another product on the same box never see `DFMS_*`. The key is already ACL’d to SYSTEM and Administrators. There is no secrets file to copy off the disk.

```
HKLM\SYSTEM\CurrentControlSet\Services\TIPERDFMS
  Environment  REG_MULTI_SZ
    DFMS_SYMMETRIC_KEY=…
    DFMS_REFRESH_KEY=…
    DFMS_MFA_KEY=…
    DFMS_DB_PASSWORD=…
```

Generate keys on a trusted machine, **not** the sample zeros in `.env.sample`. Windows PowerShell 5.1:

```powershell
-join ([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32) | ForEach-Object { '{0:x2}' -f $_ })
```

Run that **three times** (symmetric, refresh, MFA). Never reuse a key across environments. Keep a KeePass / paper copy **off the server**.

Install the service first (so the registry key exists), **then** set Environment, **then** start:

```bat
dfms.exe install
```

```powershell
$svc = "HKLM:\SYSTEM\CurrentControlSet\Services\TIPERDFMS"
$keys = @(
  "DFMS_SYMMETRIC_KEY=<hex>",
  "DFMS_REFRESH_KEY=<hex>",
  "DFMS_MFA_KEY=<hex>",
  "DFMS_DB_PASSWORD=<sql password>"
)
New-ItemProperty -Path $svc -Name Environment -PropertyType MultiString -Value $keys -Force
```

```bat
dfms.exe start
```

`New-ItemProperty -Force` **replaces** the whole multi-string. To add a value later, read the current list, append, and write it back.

```powershell
(Get-ItemProperty $svc).Environment
```

A restart is required after any change (`dfms.exe restart`). `dfms.exe uninstall` / `install` can drop `Environment` — set the keys again before the first start.

Do **not** pass keys on `dfms.exe install` (they land in command history). After pasting secrets into an elevated PowerShell session, clear history:

```powershell
Clear-History
Remove-Item (Get-PSReadlineOption).HistorySavePath -ErrorAction SilentlyContinue
```

Do **not** put `DFMS_*` in the machine-wide Environment Variables Control Panel. Every process on the box would inherit them.

### Fallback: `secrets.env` (no SCM, or console runs)

If you must run `dfms.exe console` outside the service, or you cannot write the service key, use `C:\ProgramData\DFMS\secrets.env` (ACL SYSTEM + Administrators only):

```env
DFMS.SYMMETRIC_KEY=
DFMS.REFRESH_KEY=
DFMS.MFA_KEY=
DFMS.DB.PASSWORD=
```

```powershell
New-Item -ItemType Directory -Force -Path C:\ProgramData\DFMS | Out-Null
icacls C:\ProgramData\DFMS /inheritance:r
icacls C:\ProgramData\DFMS /grant:r "NT AUTHORITY\SYSTEM:(OI)(CI)F" "BUILTIN\Administrators:(OI)(CI)F"
icacls C:\ProgramData\DFMS\secrets.env /inheritance:r
icacls C:\ProgramData\DFMS\secrets.env /grant:r "NT AUTHORITY\SYSTEM:F" "BUILTIN\Administrators:F"
```

Service `Environment` still wins over this file. Do not keep both in sync as two sources of truth.

### What stays in a non-secret `.env`

Put this in `C:\ProgramData\DFMS\.env` (or `C:\dfms\config\.env`). No passwords:

```env
DFMS.LISTEN_ADDRESS=127.0.0.1:8080
DFMS.SHUTDOWN_TIMEOUT=20s
DFMS.DEBUG=false
DFMS.ALLOW_INSECURE_HTTP=true
DFMS.TRUST_FORWARDED_FOR=true
DFMS.CORS=http://dfms,http://dfms.tiper.local

DFMS.DB.HOST=127.0.0.1
DFMS.DB.PORT=1433
DFMS.DB.INSTANCE=
DFMS.DB.USER=dfms
DFMS.DB.NAME=DFMS
DFMS.DB.ENCRYPT=false
```

Load order (later wins): `.env` (first match) → `secrets.env` (fallback) → `DFMS_*` on the process, including the service `Environment` value.

Search path for `.env`: `DFMS_CONFIG_DIR` (if set), `C:\ProgramData\DFMS`, current directory, `.\config`.

Search path for `secrets.env`: `DFMS_CONFIG_DIR` and `C:\ProgramData\DFMS` only — **not** the install folder.

### Settings after first login

Mail, SMS, Sage, attachment directory, and job schedules are **not** in `.env`. They live in SQL, encrypted with `DFMS.SYMMETRIC_KEY`. Set them under **Settings** in the UI. Point attachments at something like `D:\dfms\uploads`, not under `C:\nginx\html`.

---

## 4. nginx (port 80, no SSL)

Install the official Windows nginx zip to `C:\nginx`. Replace `conf\nginx.conf` with:

```nginx
worker_processes  1;
error_log  logs/error.log;
pid        logs/nginx.pid;

events {
    worker_connections  1024;
}

http {
    include       mime.types;
    default_type  application/octet-stream;
    sendfile      on;
    keepalive_timeout  65;
    # Fiber body cap is 64 MiB (attachments).
    client_max_body_size 64m;

    map $http_upgrade $connection_upgrade {
        default upgrade;
        ''      close;
    }

    upstream dfms_api {
        server 127.0.0.1:8080;
        keepalive 16;
    }
    upstream dfms_web {
        server 127.0.0.1:3000;
    }

    server {
        listen       80;
        server_name  dfms dfms.tiper.local 10.20.30.40 _;

        # Overwrite forwarding headers (do not pass the client's X-Forwarded-For).
        proxy_set_header Host              $host;
        proxy_set_header X-Forwarded-For   $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host  $host;
        proxy_http_version 1.1;
        proxy_set_header Connection "";

        location /api/ {
            proxy_pass         http://dfms_api;
            proxy_read_timeout 120s;
            # Keep Path=/api cookies on this host (same-origin).
        }

        location = /healthz {
            proxy_pass http://dfms_api/healthz;
        }

        location /_next/static/ {
            proxy_pass http://dfms_web;
            add_header Cache-Control "public, max-age=31536000, immutable";
        }

        location / {
            proxy_pass http://dfms_web;
            proxy_read_timeout 60s;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection $connection_upgrade;
        }
    }
}
```

Change `server_name` and `DFMS.CORS` to the names/IPs operators will use.

Test and start:

```bat
cd C:\nginx
nginx -t
nginx
```

Reload after edits: `nginx -s reload`.

Because nginx proxies `/api/` to Fiber, the Next.js BFF (`web/src/app/api`) is unused for browsers. Keep `NEXT_PUBLIC_API_BASE=/api/v1` so the UI calls same-origin `/api/v1`.

---

## 5. Windows services

### API (`dfms.exe`)

Elevated, from `C:\dfms`:

```bat
dfms.exe install
```

Set `Environment` on `TIPERDFMS` (§3) **before** the first start, then:

```bat
dfms.exe start
dfms.exe status
```

The service name is `TIPERDFMS`. It starts automatically on boot and restarts on failure. On start the process changes directory to the folder that contains `dfms.exe` (Windows SCM otherwise starts in `C:\Windows\System32`).

### Next.js

The UI must listen on loopback only. Create `C:\dfms\web\.env.production.local`:

```
NEXT_PUBLIC_API_BASE=/api/v1
```

With NSSM:

```bat
nssm install DFMSWeb "C:\Program Files\nodejs\node.exe"
nssm set DFMSWeb AppDirectory C:\dfms\web
nssm set DFMSWeb AppParameters "C:\dfms\web\node_modules\next\dist\bin\next start -p 3000 -H 127.0.0.1"
nssm set DFMSWeb Start SERVICE_AUTO_START
nssm start DFMSWeb
```

### nginx

```bat
nssm install nginx C:\nginx\nginx.exe
nssm set nginx AppDirectory C:\nginx
nssm set nginx Start SERVICE_AUTO_START
nssm start nginx
```

Start order on boot: SQL Server → `TIPERDFMS` → `DFMSWeb` → `nginx`. If SQL is local, `dfms.exe` already retries the connection for a few seconds at boot.

### Firewall

Allow inbound **TCP 80** from the LAN. Do **not** allow 3000, 8080, or 1433 from other hosts.

```powershell
New-NetFirewallRule -DisplayName "DFMS nginx HTTP" -Direction Inbound -Protocol TCP -LocalPort 80 -Action Allow
```

---

## 6. First login and checks

1. Browse to `http://dfms` (or the IP you put in `server_name` / `DFMS.CORS`).
2. Log in with the seeded superuser and **change the password immediately**.
3. Confirm cookies are set for host `dfms` (not `127.0.0.1`) and `Secure` is off.
4. `http://dfms/healthz` returns JSON from Fiber.
5. Upload a test attachment; it should land under the directory set in Settings → Attachments.

Logs: `C:\dfms\logs\app.log`, `access.log`, `db.log`. nginx: `C:\nginx\logs\`.

---

## 7. What not to do

- Do not copy `.env.sample` into `C:\dfms` and call it production.
- Do not put keys in `web/.env*` — anything `NEXT_PUBLIC_*` is visible in the browser bundle.
- Do not listen Fiber on `:8080` (all interfaces) “for convenience”.
- Do not use the old `APP_` / `APP.` names — the process only reads `DFMS_` / `DFMS.`.
- To add TLS later, follow [windows-deploy-tls.md](windows-deploy-tls.md) (`DFMS.ALLOW_INSECURE_HTTP=false` and `https://…` in `DFMS.CORS`).
- Do not rotate `DFMS.SYMMETRIC_KEY` without re-entering SMTP / Sage / SMS passwords in Settings.

---

## 8. Optional: SQL TLS on the same box

`DFMS.DB.ENCRYPT=false` is normal when SQL Server has no certificate. If you enable Force Encryption on SQL, set `DFMS.DB.ENCRYPT=true`; the driver already sets `trust server certificate=true` for a local cert.
