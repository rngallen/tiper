import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/shared/lib/cn";

export const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-medium whitespace-nowrap transition-colors [&>svg]:size-3",
  {
    variants: {
      variant: {
        default: "border-transparent bg-primary text-primary-foreground",
        secondary: "border-transparent bg-secondary text-secondary-foreground",
        outline: "text-foreground",
        destructive: "border-transparent bg-destructive/12 text-destructive dark:bg-destructive/20",
        success: "border-transparent bg-success/15 text-success-foreground dark:text-success [&]:text-[oklch(0.4_0.12_150)] dark:[&]:text-[oklch(0.8_0.14_150)]",
        warning: "border-transparent bg-warning/20 text-[oklch(0.45_0.1_75)] dark:text-[oklch(0.85_0.13_80)]",
        info: "border-transparent bg-info/15 text-[oklch(0.42_0.12_240)] dark:text-[oklch(0.8_0.12_240)]",
        muted: "border-transparent bg-muted text-muted-foreground",
      },
    },
    defaultVariants: { variant: "default" },
  },
);

export type BadgeProps = React.HTMLAttributes<HTMLSpanElement> & VariantProps<typeof badgeVariants>;

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span data-slot="badge" className={cn(badgeVariants({ variant }), className)} {...props} />;
}
