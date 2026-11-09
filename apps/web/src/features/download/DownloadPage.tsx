/**
 * DownloadPage — « Télécharger pour Windows », route /telecharger.
 *
 * Branchée sur la capacité `can_offer_windows_download` de /bootstrap (ADR 0044) : le serveur décide
 * si cette instance propose l'application (l'hébergé oui, l'application de bureau elle-même non) ; la
 * page ne regarde jamais le nom du profil. Accessible sans être connecté (chemin anonyme) : un
 * visiteur doit pouvoir installer avant d'avoir un compte.
 *
 * Elle porte la mention imposée par Microsoft et l'explication de l'avertissement SmartScreen
 * (l'installeur n'est pas signé par un certificat de code, décision D-10 du plan Desktop).
 */
import { MicrosoftNotice } from '@/components/shell/MicrosoftNotice'
import { DetailSection, SectionTitle } from '@/components/ui/detail-section'
import { GITHUB_LATEST_RELEASE_URL } from '@/lib/appLinks'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { DOWNLOAD_TEXT } from './i18n'

const BUTTON_CLASS =
  'inline-flex items-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90'

export function DownloadPage() {
  const locale = useAppShellStore((s) => s.locale)
  const offered = useDeploymentCapability('can_offer_windows_download')
  const t = DOWNLOAD_TEXT[locale]

  if (!offered) {
    return (
      <div className="mx-auto w-full max-w-3xl px-4 py-8">
        <p className="text-sm text-muted-foreground">{t.unavailable}</p>
      </div>
    )
  }

  return (
    <div className="mx-auto w-full max-w-3xl space-y-8 px-4 py-8">
      <header className="space-y-3">
        <h1 className="text-2xl font-semibold text-foreground">{t.title}</h1>
        <p className="text-sm leading-relaxed text-muted-foreground">{t.intro}</p>
      </header>

      <ul className="list-disc space-y-1 pl-5 text-sm leading-relaxed text-muted-foreground">
        {t.highlights.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>

      <section className="space-y-2">
        <a href={GITHUB_LATEST_RELEASE_URL} target="_blank" rel="noopener noreferrer" className={BUTTON_CLASS}>
          {t.action}
        </a>
        <p className="text-xs text-muted-foreground">{t.actionHint}</p>
        <p className="text-xs text-muted-foreground">{t.requirements}</p>
      </section>

      <section aria-labelledby="smartscreen-heading" className="space-y-3 rounded-md border border-border bg-card p-4">
        <SectionTitle id="smartscreen-heading">{t.smartScreenHeading}</SectionTitle>
        <p className="text-sm leading-relaxed text-muted-foreground">{t.smartScreenIntro}</p>
        <ol className="list-decimal space-y-1 pl-5 text-sm leading-relaxed text-foreground">
          {t.smartScreenSteps.map((step) => (
            <li key={step}>{step}</li>
          ))}
        </ol>
        <p className="text-sm leading-relaxed text-muted-foreground">{t.smartScreenOutro}</p>
      </section>

      <DetailSection title={t.dataHeading} className="space-y-2">
        <p className="text-sm leading-relaxed text-muted-foreground">{t.dataBody}</p>
      </DetailSection>

      <div className="border-t border-border pt-4 text-xs text-muted-foreground">
        <MicrosoftNotice />
      </div>
    </div>
  )
}
