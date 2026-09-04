function Card({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <article className="rounded-xl border border-border bg-card p-4 shadow-sm">
      <p className="text-xs uppercase tracking-wider text-muted-foreground">{label}</p>
      <p className="mt-2 text-2xl font-semibold tabular-nums">{value}</p>
      <p className="mt-1 text-sm text-muted-foreground">{hint}</p>
    </article>
  );
}

export function OperationsBoard() {
  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card label="Stock on hand" value="184,220 m³" hint="18 tanks · Kigamboni" />
        <Card label="Today loadings" value="126" hint="4.1M litres released" />
        <Card label="Open receipts" value="3" hint="1 vessel discharging" />
        <Card label="EWURA exceptions" value="2" hint="Action required" />
      </div>
      <section className="rounded-xl border border-border bg-card p-5">
        <h2 className="mb-4 text-lg font-semibold">Tank farm</h2>
        <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
          {[
            ["TK-01", "PMS", 78],
            ["TK-02", "PMS", 41],
            ["TK-04", "AGO", 86],
            ["TK-05", "AGO", 22],
            ["TK-07", "IK", 63],
            ["TK-08", "JPF", 0],
          ].map(([code, product, pct]) => (
            <div key={String(code)} className="rounded-lg border border-border p-3">
              <p className="font-mono text-xs">{code}</p>
              <div className="mt-2 h-24 overflow-hidden rounded-md bg-muted">
                <div className="mt-auto h-full bg-primary/80" style={{ transform: `translateY(${100 - Number(pct)}%)` }} />
              </div>
              <p className="mt-2 text-center text-sm font-medium tabular-nums">{pct}%</p>
              <p className="text-center text-xs text-muted-foreground">{product}</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

export function RegisterPage({
  title,
  columns,
  rows,
}: {
  title: string;
  columns: string[];
  rows: string[][];
}) {
  return (
    <section className="overflow-hidden rounded-xl border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h2 className="font-semibold">{title}</h2>
        <button type="button" className="rounded-md bg-primary px-3 py-1.5 text-sm text-primary-foreground">
          New
        </button>
      </div>
      <table className="w-full text-sm">
        <thead className="bg-muted/60 text-left text-xs uppercase tracking-wider text-muted-foreground">
          <tr>
            {columns.map((c) => (
              <th key={c} className="px-4 py-2 font-medium">
                {c}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i} className="border-t border-border">
              {row.map((cell, j) => (
                <td key={j} className="px-4 py-2">
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
