import { APPEARANCE_STYLES, getComponentRoundness, getThemeConfig } from "@/lib/theme";
import { ThemeableProps } from "@/lib/themeUtils";
import { clsx } from "clsx";
import { ButtonHTMLAttributes, DetailedHTMLProps, forwardRef } from "react";

export enum ButtonSizes {
  Small = "Small",
  Large = "Large",
}

export enum ButtonVariants {
  Primary = "Primary",
  Secondary = "Secondary",
  Destructive = "Destructive",
}

export enum ButtonColors {
  Neutral = "Neutral",
  Primary = "Primary",
  Warn = "Warn",
}

export type ButtonProps = DetailedHTMLProps<ButtonHTMLAttributes<HTMLButtonElement>, HTMLButtonElement> & {
  size?: ButtonSizes;
  variant?: ButtonVariants;
  color?: ButtonColors;
} & ThemeableProps;

export const getButtonClasses = (
  size: ButtonSizes,
  variant: ButtonVariants,
  color: ButtonColors,
  roundnessClasses: string = "rounded-md", // Default fallback
  appearance: string = "", // Theme appearance (shadows, borders, etc.)
) =>
  clsx(
    {
      // FORK: admin's .btn metrics — 13.5px/500, centered content, 150ms
      // transitions and a focus-visible ring instead of a 300ms/14px/normal
      // button. Sizes below are retuned to admin's 36px control height.
      "box-border inline-flex items-center justify-center gap-[7px] whitespace-nowrap text-[13.5px] font-medium leading-none focus:outline-none focus-visible:shadow-[var(--ui-ring-shadow)] transition-[background-color,color,border-color,box-shadow,opacity] duration-150": true,
      "disabled:border-none disabled:bg-gray-300 disabled:text-gray-600 disabled:shadow-none disabled:cursor-not-allowed disabled:dark:bg-gray-700 disabled:dark:text-gray-900":
        variant === ButtonVariants.Primary,
      "bg-primary-light-500 dark:bg-primary-dark-500 hover:bg-primary-light-400 hover:dark:bg-primary-dark-400 text-primary-light-contrast-500 dark:text-primary-dark-contrast-500":
        variant === ButtonVariants.Primary && color !== ButtonColors.Warn,
      "bg-warn-light-500 dark:bg-warn-dark-500 hover:bg-warn-light-400 hover:dark:bg-warn-dark-400 text-white dark:text-white":
        variant === ButtonVariants.Primary && color === ButtonColors.Warn,
      // FORK: admin's .btn--outline — hairline border, --ui-accent on hover.
      "border border-[var(--ui-border)] text-gray-950 dark:text-white hover:bg-[var(--ui-accent)] focus:bg-[var(--ui-accent)] disabled:text-gray-600 disabled:hover:bg-transparent disabled:dark:hover:bg-transparent disabled:cursor-not-allowed disabled:dark:text-gray-900":
        variant === ButtonVariants.Secondary,
      "border border-button-light-border dark:border-button-dark-border text-warn-light-500 dark:text-warn-dark-500 hover:bg-warn-light-500 hover:bg-opacity-10 dark:hover:bg-warn-light-500 dark:hover:bg-opacity-10 focus:bg-warn-light-500 focus:bg-opacity-20 dark:focus:bg-warn-light-500 dark:focus:bg-opacity-20":
        color === ButtonColors.Warn && variant !== ButtonVariants.Primary,
      // admin .btn--lg / .btn
      "px-5 h-[44px] text-[14px]": size === ButtonSizes.Large,
      "px-[14px] h-[36px]": size === ButtonSizes.Small,
    },
    roundnessClasses, // Apply the full roundness classes directly
    appearance, // Apply appearance-specific styling (shadows, borders, etc.)
  );

// Helper function to get default button roundness from theme
function getDefaultButtonRoundness(): string {
  return getComponentRoundness("button");
}

// Helper function to get default button appearance from centralized theme system
function getDefaultButtonAppearance(): string {
  const themeConfig = getThemeConfig();
  const appearance = APPEARANCE_STYLES[themeConfig.appearance];
  return appearance?.button || "border border-button-light-border dark:border-button-dark-border"; // Fallback to flat design
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      children,
      className = "",
      variant = ButtonVariants.Primary,
      size = ButtonSizes.Small,
      color = ButtonColors.Primary,
      roundness, // Will use theme default if not provided
      ...props
    },
    ref,
  ) => {
    // Use theme-based values if not explicitly provided
    const actualRoundness = roundness || getDefaultButtonRoundness();
    const actualAppearance = getDefaultButtonAppearance();

    return (
      <button
        type="button"
        ref={ref}
        className={`${getButtonClasses(size, variant, color, actualRoundness, actualAppearance)} ${className}`}
        {...props}
      >
        {children}
      </button>
    );
  },
);
