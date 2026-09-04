import { roundQtyNumber } from "@/lib/format";

export function fromLitres(litres: string, density: string): { cubicMeter: string; metricTonne: string } {
  const l = Number(litres);
  const d = Number(density);
  if (!Number.isFinite(l) || l === 0) {
    return { cubicMeter: "", metricTonne: "" };
  }
  const cm = l / 1000;
  const mt = Number.isFinite(d) ? cm * d : 0;
  return {
    cubicMeter: String(roundQtyNumber(cm, "m3")),
    metricTonne: String(roundQtyNumber(mt, "mt")),
  };
}

export function addQty(a?: string, b?: string) {
  return String(roundQtyNumber(Number(a || 0) + Number(b || 0), "l"));
}
