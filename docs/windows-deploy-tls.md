# TIPER DFMS — Windows Server deployment (HTTPS)

nginx terminates TLS on **port 443**. Browsers open `https://dfms.tiper.local` (or the DNS name on the certificate). Fiber and Next.js still listen on **loopback HTTP**; they never see the certificate.

| Guide | When to use |
|---|---|
| [windows-deploy.md](windows-deploy.md) | TIPER LAN only, no certificates, port 80. |
| **This file** | Internal CA, purchased cert, or Let's Encrypt. Port 443. Use this if the host is reachable outside the LAN. |

Follow [windows-deploy.md](windows-deploy.md) for the binary, Node, SQL, Windows services, secrets layout, and the `DFMS_` prefix. This document only changes nginx, cookies, CORS, and the firewall.

```
  browser  ──https://dfms.tiper.local:443──►  nginx (certificate)
                                                ├── /     →  Next.js  127.0.0.1:3000
                                                └── /api/ →  dfms.exe 127.0.0.1:8080
  browser  ──http://…:80──►  nginx  301  →  https://…
```

---

## 1. Application settings that differ from HTTP

In `C:\ProgramData\DFMS\.env`:

```env
DFMS.LISTEN_ADDRESS=127.0.0.1:8080
DFMS.SHUTDOWN_TIMEOUT=20s
DFMS.DEBUG=false
DFMS.ALLOW_INSECURE_HTTP=false
DFMS.TRUST_FORWARDED_FOR=true
DFMS.CORS=https://dfms.tiper.local

DFMS.DB.HOST=127.0.0.1
DFMS.DB.PORT=1433
DFMS.DB.USER=dfms
DFMS.DB.NAME=DFMS
DFMS.DB.ENCRYPT=false
```

| Key | HTTP guide | This guide |
|---|---|---|
| `DFMS.ALLOW_INSECURE_HTTP` | `true` (cookies without `Secure`) | **`false`** — cookies are HTTPS-only |
| `DFMS.CORS` | `http://dfms,…` | **`https://dfms.tiper.local`** (every name on the cert) |
| `DFMS.TRUST_FORWARDED_FOR` | `true` | `true` — nginx sets `X-Forwarded-Proto https` |

`DFMS.DEBUG` stays **false**. Secrets stay on the `TIPERDFMS` service `Environment` value ([windows-deploy.md](windows-deploy.md) §3), never next to the binary.

Restart `TIPERDFMS` after the `.env` change. Existing HTTP sessions will not send cookies on HTTPS until operators log in again.

`DFMS.CORS` must list the exact origin operators type, including `https://`. A leftover `http://` origin does not make HTTP login work once `ALLOW_INSECURE_HTTP` is off (the browser will not store `Secure` cookies on `http://`).

---

## 2. Certificates

nginx for Windows reads **PEM** files (`fullchain.pem` + `privkey.pem`). It does not use the Windows certificate store directly.

Put them in `C:\ProgramData\DFMS\certs\` and ACL to SYSTEM + Administrators only. The private key must not live under `C:\nginx\html` or the app tree that operators copy.

```powershell
New-Item -ItemType Directory -Force -Path C:\ProgramData\DFMS\certs | Out-Null
icacls C:\ProgramData\DFMS\certs /inheritance:r
icacls C:\ProgramData\DFMS\certs /grant:r "NT AUTHORITY\SYSTEM:(OI)(CI)F" "BUILTIN\Administrators:(OI)(CI)F"
```

The certificate **Subject Alternative Name** must include every hostname operators will type (`dfms.tiper.local`, `dfms.tiper.co.tz`, …). Browsers reject a name that is not on the cert. Prefer a DNS name over a raw IP; an IP requires a SAN of type IP.

### Internal CA (typical TIPER LAN)

Issue a web-server certificate from Active Directory Certificate Services (or the corporate PKI) for `dfms.tiper.local`. Export as PFX, then convert to PEM.

If OpenSSL is installed:

```bat
openssl pkcs12 -in dfms.pfx -clcerts -nokeys -out C:\ProgramData\DFMS\certs\fullchain.pem
openssl pkcs12 -in dfms.pfx -nocerts -nodes -out C:\ProgramData\DFMS\certs\privkey.pem
```

Include the issuing CA (and intermediates) in `fullchain.pem` so workstations that already trust the TIPER CA do not show a warning. Domain-joined PCs that trust the CA need no extra click-through.

### Purchased / public CA

Same PEM layout. Use the hostname that is in public or internal DNS. Keep the private key off shared drives.

### Let's Encrypt (public hostname only)

If the server has a public DNS name and inbound 80/443, [win-acme](https://www.win-acme.com/) can issue and renew. Point its store to PEM files (or a scheduled export) at `C:\ProgramData\DFMS\certs\`, then `nginx -s reload` after renewal. Do not use Let's Encrypt for a name that exists only on the LAN (HTTP-01 / TLS-ALPN will fail).

### Self-signed

Workstations will warn on every browser. Use only for a lab. Do not ship a self-signed cert to operators.

---

## 3. nginx (443 + HTTP redirect)

Replace `C:\nginx\conf\nginx.conf`. Adjust `server_name` to the names on the certificate.

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

    # HTTP → HTTPS. Keep /healthz on 80 if a LAN monitor cannot speak TLS.
    server {
        listen       80;
        server_name  dfms.tiper.local;

        location = /healthz {
            proxy_pass http://127.0.0.1:8080/healthz;
        }

        location / {
            return 301 https://$host$request_uri;
        }
    }

    server {
        listen       443 ssl;
        server_name  dfms.tiper.local;

        ssl_certificate      C:/ProgramData/DFMS/certs/fullchain.pem;
        ssl_certificate_key  C:/ProgramData/DFMS/certs/privkey.pem;
        ssl_protocols        TLSv1.2 TLSv1.3;
        ssl_session_cache    shared:SSL:10m;
        ssl_session_timeout  1d;

        # Overwrite forwarding headers. Proto must be https so Fiber cookies stay Secure.
        proxy_set_header Host              $host;
        proxy_set_header X-Forwarded-For   $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host  $host;
        proxy_http_version 1.1;
        proxy_set_header Connection "";

        location /api/ {
            proxy_pass         http://dfms_api;
            proxy_read_timeout 120s;
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

```bat
cd C:\nginx
nginx -t
nginx -s reload
```

If nginx was not yet a service, install it as in the HTTP guide.

After a certificate renewal, copy the new PEM files over the old ones and `nginx -s reload`. Do not restart Node or `dfms.exe` for a cert rotation.

---

## 4. Firewall

Allow **TCP 443** (and **80** if you keep the redirect). Still do not allow 3000, 8080, or 1433 from other hosts.

```powershell
New-NetFirewallRule -DisplayName "DFMS nginx HTTPS" -Direction Inbound -Protocol TCP -LocalPort 443 -Action Allow
New-NetFirewallRule -DisplayName "DFMS nginx HTTP redirect" -Direction Inbound -Protocol TCP -LocalPort 80 -Action Allow
```

---

## 5. Checks

1. `https://dfms.tiper.local` — padlock, name matches the cert, no mixed-content warnings.
2. `http://dfms.tiper.local` — 301 to HTTPS (except `/healthz` if you left it on port 80).
3. Login cookies have `Secure`. DevTools → Application → Cookies.
4. POST from the UI succeeds (CSRF origin is `https://dfms.tiper.local`).
5. With `ALLOW_INSECURE_HTTP=false` the API also sends `Strict-Transport-Security` and CSP `upgrade-insecure-requests`.

---

## 6. Moving from the HTTP guide to this one

1. Issue or buy the certificate; write PEM files.
2. Switch nginx to the config above; open 443.
3. Set `DFMS.ALLOW_INSECURE_HTTP=false` and `DFMS.CORS=https://…`.
4. Restart `TIPERDFMS`.
5. Tell operators the new URL. Old `http://` bookmarks should land on HTTPS via the 301.

Do not run HTTP (insecure cookies) and HTTPS (Secure cookies) as two production URLs at once — browsers will mix two cookie jars for the same host.
