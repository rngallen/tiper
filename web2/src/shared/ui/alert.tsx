import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { AlertCircle, AlertTriangle, CheckCircle2, Info } from "lucide-react";
import { cn } from "@/shared/lib/cn";

const alertVariants = cva("relative flex w-full gap-3 rounded-md border px-4 py-3 text-sm [&>svg]:mt-0.5 [&>svg]:size-4 [&>svg]:shrink-0", {
  variants: {
    variant: {
      default: "bg-card text-card-foreground",
      info: "border-info/30 bg-info/8 text-foreground [&>svg]:text-info",
      success: "border-success/30 bg-success/8 text-foreground [&>svg]:text-success",
      warning: "border-warning/40 bg-warning/10 text-foreground [&>svg]:text-[oklch(0.55_0.13_75)]",
      destructive: "border-destructive/30 bg-destructive/8 text-foreground [&>svg]:text-destructive",
    },
  },
  defaultVariants: { variant: "default" },
});

const icons = { default: Info, info: Info, success: CheckCircle2, warning: AlertTriangle, destructive: AlertCircle };

export function Alert({
  className,
  variant = "default",
  title,
  children,
  icon,
  actions,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof alertVariants> & { title?: React.ReactNode; icon?: React.ReactNode; actions?: React.ReactNode }) {
  const Icon = icons[variant ?? "default"];
  return (
    <div role={variant === "destructive" ? "alert" : "status"} className={cn(alertVariants({ variant }), className)} {...props}>
      {icon === undefined ? <Icon /> : icon}
      <div className="min-w-0 flex-1 space-y-0.5">
        {title ? <div className="font-medium leading-tight">{title}</div> : null}
        {children ? <div className="text-muted-foreground [&_p]:leading-relaxed">{children}</div> : null}
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </div>
  );
}
