"use client";

import { useEffect } from "react";
import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { Fuel, ShieldCheck, Warehouse } from "lucide-react";
import { Logo } from "@/components/Logo";
import { useAuth } from "@/lib/auth";

const YEAR = new Date().getFullYear();

const PILLARS = [
  {
    icon: Warehouse,
    title: "Bonded stock",
    detail: "Customer × product × vessel parcels — provision, final, hold, and free-to-order.",
  },
  {
    icon: Fuel,
    title: "Receipts and lifts",
    detail: "SBM/KOJ reception, gantry loading, pump-over, ITT, and zerolisation in one ledger.",
  },
  {
    icon: ShieldCheck,
    title: "Billing and approvals",
    detail: "FSF, VSF, TBS, and KOJ with workflow, audit, and Sage posting.",
  },
] as const;

export default function Home() {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (!loading && user) {
      router.replace("/dashboard");
    }
  }, [loading, user, router]);

  return (
    <div className="min-h-screen bg-white text-brand-900">
      <header className="relative z-20 border-b border-slate-200 bg-white">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
          <Logo height={40} />
          <nav className="flex items-center gap-5 text-sm">
            <a
              href="https://tiper.co.tz/"
              target="_blank"
              rel="noopener noreferrer"
              className="hidden min-h-11 items-center font-medium text-slate-600 hover:text-brand-800 sm:inline-flex"
            >
              tiper.co.tz
            </a>
            <Link
              href="/login"
              className="inline-flex min-h-11 items-center rounded-md bg-brand-800 px-4 font-semibold text-white hover:bg-brand-900"
            >
              Sign in
            </Link>
          </nav>
        </div>
      </header>

      <main>
      <section className="relative overflow-hidden bg-brand-900 text-white">
        <Image
          src="/tiper-storage.webp"
          alt="TIPER one-stop gantry and storage tanks at Kigamboni"
          fill
          priority
          fetchPriority="high"
          quality={75}
          sizes="100vw"
          className="object-cover object-[center_42%]"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-[#02243c]/90 via-[#033860]/35 to-[#033860]/10" />
        <div className="relative mx-auto flex min-h-[32rem] max-w-6xl flex-col justify-end px-6 py-16 md:min-h-[38rem] md:py-20">
          <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-white/80">
            Do it safe or not at all
          </p>
          <h1 className="mt-3 max-w-3xl font-display text-4xl font-semibold uppercase tracking-wide drop-shadow-sm md:text-5xl">
            Depot Fuel Management
          </h1>
          <p className="mt-5 max-w-xl text-sm leading-relaxed text-white/95 md:text-base">
            TIPER Kigamboni bonded warehouse — stock, deliveries, billing, and
            approvals for authorised staff.
          </p>
          <div className="mt-8 flex flex-wrap items-center gap-3">
            <Link
              href="/login"
              className="inline-flex min-h-11 items-center rounded-md bg-white px-5 text-sm font-semibold text-brand-900 hover:bg-brand-50"
            >
              Sign in to DFMS
            </Link>
            <a
              href="https://tiper.co.tz/"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex min-h-11 items-center rounded-md border border-white/50 bg-white/10 px-5 text-sm font-semibold text-white backdrop-blur-sm hover:bg-white/20"
            >
              Corporate website
            </a>
          </div>
        </div>
      </section>

      <section className="border-b border-slate-200 bg-brand-50">
        <div className="mx-auto grid max-w-6xl gap-px bg-slate-200 md:grid-cols-3">
          {PILLARS.map(({ icon: Icon, title, detail }) => (
            <div key={title} className="bg-brand-50 px-8 py-10">
              <span className="flex h-10 w-10 items-center justify-center rounded-md bg-brand-800 text-white">
                <Icon className="h-5 w-5" aria-hidden />
              </span>
              <h2 className="mt-4 text-base font-semibold text-brand-900">{title}</h2>
              <p className="mt-2 text-sm leading-relaxed text-slate-600">{detail}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto grid max-w-6xl items-center gap-10 px-6 py-16 md:grid-cols-2">
        <div className="relative aspect-[3/2] overflow-hidden rounded-xl bg-slate-100">
          <Image
            src="/tiper-hero.jpg"
            alt="TIPER storage tanks at Kigamboni"
            fill
            sizes="(min-width: 768px) 50vw, 100vw"
            className="object-cover"
          />
        </div>
        <div>
          <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-brand-700">
            About TIPER
          </p>
          <h2 className="mt-2 text-2xl font-semibold tracking-tight text-brand-900 md:text-3xl">
            Tanzania International Petroleum Reserves Limited
          </h2>
          <p className="mt-4 text-sm leading-relaxed text-slate-600 md:text-base">
            TIPER is a vast storage terminal for petroleum products, licensed as
            a bonded warehouse and jointly owned by the Government of the United
            Republic of Tanzania through the Treasury Registrar and Geneva-based
            Oryx Energies SA.
          </p>
          <p className="mt-3 text-sm leading-relaxed text-slate-600 md:text-base">
            The Kigamboni terminal holds 313,183&nbsp;m³ of operational tankage
            — more than 21% of Dar es Salaam storage — with one-stop vessel
            reception and truck loading for AGO and PMS. This system is for
            authorised TIPER staff only.
          </p>
        </div>
      </section>
      </main>

      <footer className="border-t border-slate-200 bg-brand-900 px-6 py-10 text-sm text-slate-200">
        <div className="mx-auto flex max-w-6xl flex-col gap-6 md:flex-row md:items-end md:justify-between">
          <div>
            <Logo variant="white" height={36} />
            <p className="mt-3 max-w-sm text-sm leading-relaxed text-slate-200">
              Tanzania International Petroleum Reserves Limited · Kigamboni
              <br />
              Confidential corporate system — authorised personnel only.
            </p>
          </div>
          <div className="flex flex-col text-sm">
            <a
              href="mailto:info@tiper.co.tz"
              className="inline-flex min-h-11 items-center text-white underline-offset-2 hover:underline"
            >
              info@tiper.co.tz
            </a>
            <a
              href="tel:+255225511500"
              className="inline-flex min-h-11 items-center text-white underline-offset-2 hover:underline"
            >
              +255 (0) 22 5511 500
            </a>
            <a
              href="https://tiper.co.tz/"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex min-h-11 items-center text-white underline-offset-2 hover:underline"
            >
              tiper.co.tz
            </a>
          </div>
        </div>
        <p className="mx-auto mt-8 max-w-6xl text-sm text-slate-300">
          © {YEAR} TIPER. All rights reserved.
        </p>
      </footer>
    </div>
  );
}
