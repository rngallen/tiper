import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_app/ewura")({
  component: () => (
    <div className="grid gap-4 md:grid-cols-2">
      <section className="rounded-xl border border-border bg-card p-5">
        <h2 className="font-semibold">Depot licence</h2>
        <p className="mt-2 text-sm text-muted-foreground">Nightly sync from NPGIS. Valid to 31 Dec 2026.</p>
      </section>
      <section className="rounded-xl border border-border bg-card p-5">
        <h2 className="font-semibold">Exceptions</h2>
        <ul className="mt-3 list-disc space-y-2 pl-4 text-sm">
          <li>Driver licence T 118 BTX expired — loading on hold.</li>
          <li>NPGIS shipment 4419 missing density certificate.</li>
        </ul>
      </section>
    </div>
  ),
});
