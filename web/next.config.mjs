/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Dev server 403s `_next/static` when the page is opened as 127.0.0.1
  // while the compiler origin is localhost (Lighthouse does this).
  allowedDevOrigins: ["127.0.0.1", "localhost"],
  images: {
    // Source photos are ≤1620px; 3840 variants just waste bytes on 4K viewports.
    deviceSizes: [640, 750, 828, 1080, 1200, 1620, 1920],
    qualities: [75],
  },
  // Inline Tailwind into <style> so the browser does not wait on a render-blocking
  // CSS request before first paint (FCP/LCP). Production only; see
  // https://nextjs.org/docs/app/api-reference/config/next-config-js/inlineCss
  experimental: {
    inlineCss: true,
  },
  // /api/* is handled by src/app/api/[...path]/route.ts (forwards Set-Cookie).
  async redirects() {
    return [
      { source: "/masterdata", destination: "/stock/setups/customers", permanent: false },
      { source: "/masterdata/:path*", destination: "/stock/setups/:path*", permanent: false },
      { source: "/finance/setups/fcf", destination: "/finance/setups/fsf", permanent: false },
      { source: "/finance/setups/fcf/:path*", destination: "/finance/setups/fsf/:path*", permanent: false },
      { source: "/finance/setups/vcf", destination: "/finance/setups/vsf", permanent: false },
      { source: "/finance/setups/vcf/:path*", destination: "/finance/setups/vsf/:path*", permanent: false },
      { source: "/approvals/fcf-fees", destination: "/approvals/fsf-fees", permanent: false },
      { source: "/approvals/variable-fees", destination: "/approvals/vsf-fees", permanent: false },
    ];
  },
};

export default nextConfig;
