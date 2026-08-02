import { Layers } from "lucide-react";

/**
 * FORK: the platform's default brand lockup, mirroring the admin console's shell
 * logo (admin/src/components/shell/Logo.vue) — same lucide "layers" glyph in a
 * rounded square filled with the primary color, plus the wordmark.
 *
 * This is a *fallback*: it renders only when the organization's Zitadel label
 * policy carries no logoUrl, so an org that uploads its own logo still wins.
 * Unlike an uploaded image it follows the branded primary color and both themes
 * automatically, which is why the platform default is a component and not an asset.
 *
 * The name is build-time (NEXT_PUBLIC_BRAND_NAME) — changing it needs a rebuild.
 */
export const BRAND_NAME = process.env.NEXT_PUBLIC_BRAND_NAME || "Helm";

export function BrandLogo({ name = BRAND_NAME }: { name?: string }) {
  return (
    <div className="flex items-center gap-2.5">
      <span className="bg-primary-light-500 text-primary-light-contrast-500 dark:bg-primary-dark-500 dark:text-primary-dark-contrast-500 flex h-9 w-9 items-center justify-center rounded-md">
        <Layers size={20} strokeWidth={2} />
      </span>
      <span className="text-[17px] font-bold tracking-[-0.02em]">{name}</span>
    </div>
  );
}
