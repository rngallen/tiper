import type { ReactNode } from "react";
import Image from "next/image";

/**
 * Sign-in chrome: photo is atmosphere only. The card is locked to the
 * viewport centre on desktop so the gantry subject cannot pull it left.
 */
export function PublicAuthShell({
  children,
  footer,
}: {
  children: ReactNode;
  footer?: ReactNode;
}) {
  return (
    <div className="relative min-h-screen bg-slate-100 lg:bg-[#061018]">
      <div className="relative h-44 overflow-hidden sm:h-52 lg:fixed lg:inset-0 lg:h-auto">
        <Image
          src="/tiper-login-v2.webp"
          alt=""
          fill
          priority
          quality={75}
          sizes="100vw"
          className="object-cover object-center"
        />
        <div className="absolute inset-0 bg-black/40" />
      </div>

      <main className="relative z-10 flex min-h-[calc(100vh-11rem)] items-center justify-center px-4 py-8 sm:px-6 lg:fixed lg:inset-0 lg:min-h-0">
        <div className="w-full max-w-[420px]">
          <div className="login-panel login-card overflow-hidden rounded-xl border border-slate-200/90 bg-white">
            <div className="h-[3px] bg-brand-800" aria-hidden />
            <div className="flex flex-col items-center px-7 py-8 text-center sm:px-8">
              {children}
            </div>
          </div>
          {footer ? (
            <div className="mt-5 px-1 text-center text-sm leading-relaxed text-slate-600 [&_a]:inline-flex [&_a]:min-h-11 [&_a]:items-center lg:text-slate-200 lg:[&_a]:text-white lg:[&_a]:underline">
              {footer}
            </div>
          ) : null}
        </div>
      </main>
    </div>
  );
}
