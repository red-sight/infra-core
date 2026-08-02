"use client";

import { BrandLogo } from "@/components/brand-logo";
import { Logo } from "@/components/logo";
import { useResponsiveLayout } from "@/lib/theme-hooks";
import { BrandingSettings } from "@zitadel/proto/zitadel/settings/v2/branding_settings_pb";
import React, { Children, ReactNode } from "react";
import { Card } from "./card";
import { ThemeWrapper } from "./theme-wrapper";

/**
 * DynamicTheme component handles layout switching between traditional top-to-bottom
 * and modern side-by-side layouts based on NEXT_PUBLIC_THEME_LAYOUT.
 *
 * For side-by-side layout:
 * - First child: Goes to left side (title, description, etc.)
 * - Second child: Goes to right side (forms, buttons, etc.)
 * - Single child: Falls back to right side for backward compatibility
 *
 * For top-to-bottom layout:
 * - All children rendered in traditional centered layout
 */
export function DynamicTheme({
  branding,
  children,
}: {
  children: ReactNode | ((isSideBySide: boolean) => ReactNode);
  branding?: BrandingSettings;
}) {
  const { isSideBySide } = useResponsiveLayout();
  const hasLogo = Boolean(branding?.lightTheme?.logoUrl || branding?.darkTheme?.logoUrl);

  // Resolve children immediately to avoid passing functions through React
  const actualChildren: ReactNode = React.useMemo(() => {
    if (typeof children === "function") {
      return (children as (isSideBySide: boolean) => ReactNode)(isSideBySide);
    }
    return children;
  }, [children, isSideBySide]);

  return (
    <ThemeWrapper branding={branding}>
      {isSideBySide
        ? // Side-by-side layout: first child goes left, second child goes right
          (() => {
            const childArray = Children.toArray(actualChildren);
            const leftContent = childArray[0] || null;
            const rightContent = childArray[1] || null;

            // If there's only one child, it's likely the old format - keep it on the right side
            const hasLeftRightStructure = childArray.length === 2;

            return (
              <div className="relative mx-auto w-full max-w-[1100px] px-8 py-4">
                <Card>
                  <div className="flex min-h-[400px]">
                    {/* Left side: First child + branding */}
                    <div className="from-primary-50 to-primary-100 dark:from-primary-900/20 dark:to-primary-800/20 flex w-1/2 flex-col justify-center bg-gradient-to-br p-4 lg:p-8">
                      <div className="mx-auto max-w-[440px] space-y-8">
                        {/* Logo and branding */}
                        {branding && (
                          <Logo
                            lightSrc={branding.lightTheme?.logoUrl}
                            darkSrc={branding.darkTheme?.logoUrl}
                            height={150}
                            width={150}
                          />
                        )}

                        {/* First child content (title, description) - only if we have left/right structure */}
                        {hasLeftRightStructure && (
                          <div className="flex flex-col items-start space-y-4 text-left">
                            {/* Apply larger styling to the content */}
                            <div className="space-y-6 [&_h1]:text-left [&_h1]:text-4xl [&_h1]:leading-tight [&_h1]:text-gray-900 [&_h1]:lg:text-4xl [&_h1]:dark:text-white [&_p]:text-left [&_p]:leading-relaxed [&_p]:text-gray-700 [&_p]:dark:text-gray-300">
                              {leftContent}
                            </div>
                          </div>
                        )}
                      </div>
                    </div>

                    {/* Right side: Second child (form) or single child if old format */}
                    <div className="flex w-1/2 items-center justify-center p-4 lg:p-8">
                      <div className="w-full max-w-[440px]">
                        <div className="space-y-6">{hasLeftRightStructure ? rightContent : leftContent}</div>
                      </div>
                    </div>
                  </div>
                </Card>
              </div>
            );
          })()
        : // Traditional top-to-bottom layout - center title/description, left-align forms
          (() => {
            const childArray = Children.toArray(actualChildren);
            const titleContent = childArray[0] || null;
            const formContent = childArray[1] || null;
            const hasMultipleChildren = childArray.length > 1;

            return (
              // FORK: on phones the card IS the screen — it grows to fill the
              // viewport and drops its chrome (see .login-card in globals.scss),
              // so the form sits centered on a full-height surface instead of in
              // a band floating in dead space. From sm up it is a floating card.
              <div className="relative mx-auto flex w-full max-w-[440px] flex-1 flex-col px-0 py-0 sm:block sm:flex-none sm:px-4 sm:py-4">
                <Card className="login-card flex flex-1 flex-col justify-center sm:block sm:flex-none" padding="p-5 sm:p-6">
                  {/* FORK: 24px rhythm (admin --d-gap) instead of 32px. */}
                  <div className="mx-auto flex flex-col items-center space-y-6">
                    {/* FORK: the org's uploaded logo when the label policy has one,
                        otherwise the platform brand lockup. Upstream rendered an
                        always-present empty wrapper, which cost 32px of dead space
                        above the title whenever no logoUrl was set. */}
                    <div className="relative flex flex-row items-center justify-center">
                      {hasLogo ? (
                        <Logo
                          lightSrc={branding?.lightTheme?.logoUrl}
                          darkSrc={branding?.darkTheme?.logoUrl}
                          height={150}
                          width={150}
                        />
                      ) : (
                        <BrandLogo />
                      )}
                    </div>

                    {hasMultipleChildren ? (
                      <>
                        {/* Title and description - center aligned */}
                        <div className="flex w-full flex-col items-center gap-1 text-center">{titleContent}</div>

                        {/* Form content - left aligned */}
                        <div className="w-full">{formContent}</div>
                      </>
                    ) : (
                      // Single child - use original behavior
                      <div className="w-full">{actualChildren}</div>
                    )}
                    {/* FORK: dropped an empty trailing <div> that space-y turned
                        into 32px of dead space at the bottom of every card. */}
                  </div>
                </Card>
              </div>
            );
          })()}
    </ThemeWrapper>
  );
}
