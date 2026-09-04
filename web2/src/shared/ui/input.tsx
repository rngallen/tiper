import * as React from "react";
import { cn } from "@/shared/lib/cn";

export type InputProps = React.InputHTMLAttributes<HTMLInputElement> & {
  invalid?: boolean;
  leading?: React.ReactNode;
  trailing?: React.ReactNode;
};

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type = "text", invalid, leading, trailing, ...props }, ref) => {
    const input = (
      <input
        ref={ref}
        type={type}
        data-slot="input"
        aria-invalid={invalid || props["aria-invalid"] || undefined}
        className={cn(
          "flex h-9 w-full min-w-0 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs transition-[color,box-shadow] outline-none",
          "placeholder:text-muted-foreground file:border-0 file:bg-transparent file:text-sm file:font-medium",
          "focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30",
          "aria-invalid:border-destructive aria-invalid:ring-destructive/20",
          "disabled:cursor-not-allowed disabled:opacity-50 read-only:bg-muted/50",
          leading ? "pl-9" : "",
          trailing ? "pr-9" : "",
          className,
        )}
        {...props}
      />
    );
    if (!leading && !trailing) return input;
    return (
      <div className="relative w-full">
        {leading ? (
          <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-muted-foreground [&_svg]:size-4">
            {leading}
          </span>
        ) : null}
        {input}
        {trailing ? (
          <span className="absolute inset-y-0 right-0 flex items-center pr-2 text-muted-foreground [&_svg]:size-4">
            {trailing}
          </span>
        ) : null}
      </div>
    );
  },
);
Input.displayName = "Input";

export const Textarea = React.forwardRef<
  HTMLTextAreaElement,
  React.TextareaHTMLAttributes<HTMLTextAreaElement> & { invalid?: boolean }
>(({ className, invalid, ...props }, ref) => (
  <textarea
    ref={ref}
    data-slot="textarea"
    aria-invalid={invalid || undefined}
    className={cn(
      "flex min-h-20 w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-xs outline-none",
      "placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30",
      "aria-invalid:border-destructive disabled:cursor-not-allowed disabled:opacity-50",
      className,
    )}
    {...props}
  />
));
Textarea.displayName = "Textarea";
