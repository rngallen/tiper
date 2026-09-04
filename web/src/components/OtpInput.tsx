"use client";

import { useEffect, useRef } from "react";

const LENGTH = 6;

function digitsOnly(raw: string): string {
  return raw.replace(/\D/g, "").slice(0, LENGTH);
}

/** Six-cell numeric OTP — paste, arrows, and SMS autofill fill the row. */
export function OtpInput({
  value,
  onChange,
  onComplete,
  disabled,
  autoFocus,
  error,
}: {
  value: string;
  onChange: (next: string) => void;
  /** Fires once when the sixth digit is entered or pasted. */
  onComplete?: (code: string) => void;
  disabled?: boolean;
  autoFocus?: boolean;
  error?: boolean;
}) {
  const digits = digitsOnly(value);
  const refs = useRef<Array<HTMLInputElement | null>>([]);

  useEffect(() => {
    if (autoFocus) refs.current[0]?.focus();
  }, [autoFocus]);

  function commit(next: string, focusAt?: number) {
    const clean = digitsOnly(next);
    const wasComplete = digits.length === LENGTH;
    onChange(clean);
    if (clean.length === LENGTH && !wasComplete) {
      onComplete?.(clean);
    }
    if (focusAt == null) return;
    const i = Math.min(Math.max(focusAt, 0), LENGTH - 1);
    requestAnimationFrame(() => refs.current[i]?.focus());
  }

  return (
    <div
      className="flex justify-center gap-2"
      role="group"
      aria-label="6-digit verification code"
    >
      {Array.from({ length: LENGTH }, (_, i) => (
        <input
          key={i}
          ref={(el) => {
            refs.current[i] = el;
          }}
          type="text"
          inputMode="numeric"
          name={i === 0 ? "otp" : undefined}
          autoComplete={i === 0 ? "one-time-code" : "off"}
          autoCorrect="off"
          autoCapitalize="off"
          spellCheck={false}
          pattern="[0-9]*"
          maxLength={i === 0 ? LENGTH : 1}
          disabled={disabled}
          value={digits[i] ?? ""}
          aria-invalid={error || undefined}
          aria-label={`Digit ${i + 1} of ${LENGTH}`}
          className={`otp-cell${error ? " otp-cell-error" : ""}`}
          onChange={(e) => {
            const raw = digitsOnly(e.target.value);
            if (raw.length > 1) {
              commit(digits.slice(0, i) + raw, i + raw.length);
              return;
            }
            commit(digits.slice(0, i) + raw + digits.slice(i + 1), raw ? i + 1 : i);
          }}
          onKeyDown={(e) => {
            if (e.key === "Backspace" && !digits[i] && i > 0) {
              e.preventDefault();
              commit(digits.slice(0, i - 1) + digits.slice(i), i - 1);
            } else if (e.key === "ArrowLeft" && i > 0) {
              e.preventDefault();
              refs.current[i - 1]?.focus();
            } else if (e.key === "ArrowRight" && i < LENGTH - 1) {
              e.preventDefault();
              refs.current[i + 1]?.focus();
            }
          }}
          onPaste={(e) => {
            const text = e.clipboardData.getData("text");
            if (!/\d/.test(text)) return;
            e.preventDefault();
            const pasted = digitsOnly(text);
            commit(pasted, pasted.length);
          }}
          onFocus={(e) => e.target.select()}
        />
      ))}
    </div>
  );
}
