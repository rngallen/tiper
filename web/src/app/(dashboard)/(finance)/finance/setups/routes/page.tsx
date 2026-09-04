"use client";

import { CatalogSetup } from "../_components/CatalogSetup";
import type { RouteRow } from "@/lib/lookups";

export default function RoutesPage() {
  return (
    <CatalogSetup<RouteRow>
      tableId="finance-routes"
      title="Discharge routes"
      subtitle="SBM and KOJ. KOJ fee is ticked on the external receipt when cargo used TIPER's 10-inch pipeline — not a second KOJ code."
      path="/master/routes"
      createLabel="Add route"
      singular="route"
      extra={[]}
    />
  );
}
