import type { Metadata } from "next";
import { Source_Sans_3 } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "@/lib/auth";
import { AppearanceProvider } from "@/lib/appearance";
import { CurrencyCatalogProvider } from "@/lib/currencies";
import { DecimalPrecisionProvider } from "@/lib/precision";
import { NotifyHost } from "@/components/NotifyHost";

// Source Sans 3 stays legible at form sizes (unlike Quicksand, which looks faint).
// Apply both variable (Tailwind) and className so the preloaded face is used
// immediately and browsers do not warn about an unused preload.
const sourceSans = Source_Sans_3({
  subsets: ["latin"],
  variable: "--font-sans",
  display: "swap",
  weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: "TIPER DFMS",
  description:
    "Tanzania International Petroleum Reserves Limited — bonded warehouse at Kigamboni. Depot fuel stock, billing, and approvals.",
  icons: {
    icon: [{ url: "/favicon.png", type: "image/png", sizes: "150x150" }],
    apple: [{ url: "/apple-touch-icon.png", sizes: "300x300" }],
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en-GB"
      className={`${sourceSans.variable} ${sourceSans.className}`}
      suppressHydrationWarning
    >
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){try{var s=localStorage.getItem("dfms.colorScheme");if(s==="dark"){document.documentElement.classList.add("dark");document.documentElement.style.colorScheme="dark";}}catch(e){}})();`,
          }}
        />
      </head>
      <body className="font-sans font-normal text-slate-900">
        <AuthProvider>
          <AppearanceProvider>
            <CurrencyCatalogProvider>
              <DecimalPrecisionProvider>
                {children}
                <NotifyHost />
              </DecimalPrecisionProvider>
            </CurrencyCatalogProvider>
          </AppearanceProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
