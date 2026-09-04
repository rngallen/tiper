import type { Metadata } from "next";
import { NotFoundView } from "@/components/NotFoundView";

export const metadata: Metadata = {
  title: "Page not found — TIPER DFMS",
  robots: { index: false, follow: false },
};

export default function NotFound() {
  return <NotFoundView />;
}
