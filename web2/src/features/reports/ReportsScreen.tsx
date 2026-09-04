import { useQuery } from "@tanstack/react-query";
import { http } from "@/shared/api/http";
import { toPage } from "@/shared/api/list";
import { errorMessage } from "@/shared/api/errors";
import { EmptyState } from "@/shared/ui/empty-state";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import type { ResourceDef } from "@/features/catalog/registry";

interface ReportItem { code?: string; title?: string; name?: string; group?: string; description?: string; href?: string }

export function ReportsScreen({ resource }: { resource: ResourceDef }) {
  const list = useQuery({
    queryKey: ["reports", resource.api, resource.extraQuery],
    queryFn: async () => toPage<ReportItem>(await http.get(resource.api, { page: 1, pageSize: 200, ...resource.extraQuery })),
  });
  const items = list.data?.items ?? [];
  if (list.isError) return <EmptyState title="Reports unavailable" description={errorMessage(list.error)} />;
  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">{resource.title}</h2>
        <p className="text-sm text-muted-foreground">Run a catalog report. Excel and PDF use the same engine as web/.</p>
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {items.map((r) => {
          const code = r.code || r.href?.split("/").pop();
          return (
            <Card key={code ?? r.title}>
              <CardHeader>
                <p className="text-[10px] uppercase tracking-wider text-muted-foreground">{r.group || "Report"}</p>
                <CardTitle>{r.title || r.name}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <p className="text-sm text-muted-foreground">{r.description || "Operational extract."}</p>
                {code ? (
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" onClick={() => void http.download(`/reports/view/${code}`, { export: true })}>Excel</Button>
                    <Button size="sm" variant="outline" onClick={() => void http.download(`/reports/${code}.pdf`)}>PDF</Button>
                  </div>
                ) : null}
              </CardContent>
            </Card>
          );
        })}
      </div>
      {items.length === 0 && !list.isLoading ? <EmptyState title="No reports in this group" description="The registry endpoint returned an empty page." /> : null}
    </div>
  );
}
