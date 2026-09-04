import { type NextRequest, NextResponse } from "next/server";

const TARGET = (process.env.API_PROXY_TARGET || "http://127.0.0.1:8080").replace(
  /\/$/,
  "",
);

/**
 * Same-origin BFF proxy so HttpOnly Set-Cookie from Fiber sticks on the UI
 * host. Explicitly forwards Cookie / Authorization — undici can drop those if
 * they are only copied via Headers in some runtimes.
 */
async function proxy(req: NextRequest, pathParts: string[]) {
  const incoming = new URL(req.url);
  const targetUrl = `${TARGET}/api/${pathParts.join("/")}${incoming.search}`;

  const headers = new Headers();
  req.headers.forEach((value, key) => {
    const k = key.toLowerCase();
    if (
      k === "host" ||
      k === "connection" ||
      k === "content-length" ||
      k === "cookie" ||
      k === "authorization"
    ) {
      return;
    }
    headers.set(key, value);
  });

  const cookie = req.headers.get("cookie");
  if (cookie) headers.set("cookie", cookie);
  const authorization = req.headers.get("authorization");
  if (authorization) headers.set("authorization", authorization);

  const init: RequestInit = {
    method: req.method,
    headers,
    redirect: "manual",
  };
  if (req.method !== "GET" && req.method !== "HEAD") {
    init.body = await req.arrayBuffer();
  }

  const upstream = await fetch(targetUrl, init);
  const outHeaders = new Headers();
  upstream.headers.forEach((value, key) => {
    if (key.toLowerCase() === "set-cookie") return;
    outHeaders.append(key, value);
  });

  const setCookies =
    typeof upstream.headers.getSetCookie === "function"
      ? upstream.headers.getSetCookie()
      : [];
  for (const c of setCookies) {
    outHeaders.append("Set-Cookie", c);
  }

  return new NextResponse(upstream.body, {
    status: upstream.status,
    statusText: upstream.statusText,
    headers: outHeaders,
  });
}

type Ctx = { params: Promise<{ path: string[] }> };

export const dynamic = "force-dynamic";

async function proxyWithParams(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}

export async function GET(req: NextRequest, ctx: Ctx) {
  return proxyWithParams(req, ctx);
}
export async function POST(req: NextRequest, ctx: Ctx) {
  return proxyWithParams(req, ctx);
}
export async function PUT(req: NextRequest, ctx: Ctx) {
  return proxyWithParams(req, ctx);
}
export async function PATCH(req: NextRequest, ctx: Ctx) {
  return proxyWithParams(req, ctx);
}
export async function DELETE(req: NextRequest, ctx: Ctx) {
  return proxyWithParams(req, ctx);
}
export async function OPTIONS(req: NextRequest, ctx: Ctx) {
  return proxyWithParams(req, ctx);
}
export async function HEAD(req: NextRequest, ctx: Ctx) {
  return proxyWithParams(req, ctx);
}
