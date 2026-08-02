"use client";

import { getComponentRoundness } from "@/lib/theme";
import { CheckCircleIcon } from "@heroicons/react/24/solid";
import { clsx } from "clsx";
import { ChangeEvent, DetailedHTMLProps, forwardRef, InputHTMLAttributes, ReactNode } from "react";

export type TextInputProps = DetailedHTMLProps<InputHTMLAttributes<HTMLInputElement>, HTMLInputElement> & {
  label: string;
  suffix?: string;
  placeholder?: string;
  defaultValue?: string;
  error?: string | ReactNode;
  success?: string | ReactNode;
  disabled?: boolean;
  onChange?: (value: ChangeEvent<HTMLInputElement>) => void;
  onBlur?: (value: ChangeEvent<HTMLInputElement>) => void;
  roundness?: string; // Allow override via props
};

// FORK: admin's .input (admin/src/assets/components.css) — 36px control height,
// transparent-on-card background, hairline --ui-input border, and admin's focus
// treatment (ring color + 3px soft ring) instead of upstream's 40px filled box
// with an italic placeholder.
const styles = (error: boolean, disabled: boolean, roundnessClasses: string = "rounded-md") =>
  clsx(
    {
      "h-[36px] mb-[2px] px-3 bg-transparent transition-[border-color,box-shadow] duration-150 grow": true,
      "border border-[var(--ui-input)] hover:border-[var(--ui-ring)]": true,
      "focus:border-[var(--ui-ring)] focus:shadow-[var(--ui-ring-shadow)]": true,
      "focus:outline-none focus:ring-0 text-[13.5px] text-black dark:text-white placeholder:text-[var(--ui-muted-foreground)]": true,
      "border-warn-light-500 dark:border-warn-dark-500 hover:border-warn-light-500 hover:dark:border-warn-dark-500 focus:border-warn-light-500 focus:dark:border-warn-dark-500 focus:shadow-[0_0_0_3px_rgba(220,38,38,.12)]":
        error,
      "pointer-events-none cursor-default text-gray-500 dark:text-gray-600 border-[var(--ui-border)] hover:border-[var(--ui-border)] opacity-60":
        disabled,
    },
    roundnessClasses, // Apply the full roundness classes directly
  );

// Helper function to get default input roundness from theme
function getDefaultInputRoundness(): string {
  return getComponentRoundness("input");
}

export const TextInput = forwardRef<HTMLInputElement, TextInputProps>(
  (
    {
      label,
      placeholder,
      defaultValue,
      suffix,
      required = false,
      error,
      disabled,
      success,
      onChange,
      onBlur,
      roundness,
      ...props
    },
    ref,
  ) => {
    // Use theme-based roundness if not explicitly provided
    const actualRoundness = roundness || getDefaultInputRoundness();

    return (
      // FORK: admin's .field / .field__label — 13px medium label in the
      // foreground color, 6px gap, instead of a 12px muted micro-label.
      <label className="text-text-light-500 dark:text-text-dark-500 relative flex flex-col text-[13px] font-medium">
        <span className={`mb-1.5 leading-4 ${error ? "text-warn-light-500 dark:text-warn-dark-500" : ""}`}>
          {label} {required && "*"}
        </span>
        <input
          suppressHydrationWarning
          ref={ref}
          className={styles(!!error, !!disabled, actualRoundness)}
          defaultValue={defaultValue}
          required={required}
          disabled={disabled}
          placeholder={placeholder}
          autoComplete={props.autoComplete ?? "off"}
          onChange={(e) => onChange && onChange(e)}
          onBlur={(e) => onBlur && onBlur(e)}
          {...props}
        />

        {suffix && (
          <span
            className={clsx(
              "bg-background-light-500 dark:bg-background-dark-500 absolute right-[3px] bottom-[22px] z-30 translate-y-1/2 transform p-2",
              // Extract just the roundness part for the suffix (no padding)
              actualRoundness.split(" ")[0], // Take only the first part (rounded-full, rounded-md, etc.)
            )}
          >
            @{suffix}
          </span>
        )}

        <div className="leading-14.5px h-14.5px text-12px text-warn-light-500 dark:text-warn-dark-500 flex flex-row items-center">
          <span>{error ? error : " "}</span>
        </div>

        {success && (
          <div className="text-md mt-1 flex flex-row items-center text-green-500">
            <CheckCircleIcon className="h-4 w-4" />
            <span className="ml-1">{success}</span>
          </div>
        )}
      </label>
    );
  },
);
