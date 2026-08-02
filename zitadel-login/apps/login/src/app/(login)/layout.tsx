import "@/styles/globals.scss";

import { BackgroundWrapper } from "@/components/background-wrapper";
import { LanguageProvider } from "@/components/language-provider";
import { LanguageSwitcher } from "@/components/language-switcher";
import { Skeleton } from "@/components/skeleton";
import { ThemeProvider } from "@/components/theme-provider";
import ThemeSwitch from "@/components/theme-switch";
import { LANGS, getLanguage } from "@/lib/i18n";
import { getServiceConfig } from "@/lib/service-url";
import { getAllowedLanguages } from "@/lib/zitadel";
import * as Tooltip from "@radix-ui/react-tooltip";
import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { Geist } from "next/font/google";
import { headers } from "next/headers";
import React, { Suspense } from "react";

// FORK: Geist instead of upstream's Lato, matching admin's --font-sans
// (admin/src/assets/theme.css). Self-hosted by next/font at build time.
const geist = Geist({
  subsets: ["latin"],
});

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("common");
  return { title: t("title") };
}

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const _headers = await headers();
  const { serviceConfig } = getServiceConfig(_headers);

  let languages = LANGS;
  try {
    const settings = await getAllowedLanguages({ serviceConfig });
    if (settings.allowedLanguages?.length) {
      languages = settings.allowedLanguages
        .filter((code) => LANGS.find((l) => l.code === code))
        .map((code) => getLanguage(code));
    }
  } catch (e) {
    console.error("Failed to load supported languages", e);
  }

  return (
    <html className={`${geist.className}`} suppressHydrationWarning>
      <head />
      <body>
        <ThemeProvider>
          <Tooltip.Provider>
            <Suspense
              fallback={
                <BackgroundWrapper
                  className={`bg-background-light-400 dark:bg-background-dark-500 sm:bg-background-light-600 sm:dark:bg-background-dark-600 relative flex min-h-dvh flex-col sm:justify-center`}
                >
                  <div className="relative mx-auto flex w-full max-w-[440px] flex-1 flex-col justify-center py-0 sm:block sm:flex-none sm:py-8">
                    <Skeleton>
                      <div className="h-40"></div>
                    </Skeleton>
                    <div className="flex flex-row items-center justify-end space-x-4 py-4">
                      <ThemeSwitch />
                    </div>
                  </div>
                </BackgroundWrapper>
              }
            >
              <LanguageProvider>
                <BackgroundWrapper
                  className={`bg-background-light-400 dark:bg-background-dark-500 sm:bg-background-light-600 sm:dark:bg-background-dark-600 relative flex min-h-dvh flex-col sm:justify-center`}
                >
                  <div className="relative mx-auto flex w-full max-w-[1100px] flex-1 flex-col py-0 sm:flex-none sm:py-8">
                    <div className="flex flex-1 flex-col sm:block sm:flex-none">{children}</div>
                    {/* FORK: upstream widened this row to the full 1100px container
                        on md+, so the language/theme controls drifted ~300px right
                        of the 440px card. Keep them aligned to the card. */}
                    <div className="mx-auto flex max-w-[440px] flex-row flex-wrap items-center justify-end gap-3 px-4 py-4">
                      <LanguageSwitcher languages={languages} />
                      <ThemeSwitch />
                    </div>
                  </div>
                </BackgroundWrapper>
              </LanguageProvider>
            </Suspense>
          </Tooltip.Provider>
        </ThemeProvider>
      </body>
    </html>
  );
}
