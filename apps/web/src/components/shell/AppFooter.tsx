/**
 * AppFooter — pied de page du projet.
 *
 * Deux variantes :
 *   - `full`    : rendu en bas du contenu scrollable de l'AppShell (utilisateur
 *                 connecté). Colonnes « Le projet » et « Soutenir ».
 *   - `minimal` : rendu sur les écrans de connexion, seuls écrans qu'un visiteur
 *                 anonyme de la démo publique voit. Pas de lien « Signaler un
 *                 problème » : le FeedbackDrawer n'y est pas monté.
 *
 * Les URL externes viennent de `lib/appLinks` (source unique, garde-rail
 * `appLinks.guard.test.ts`).
 */
import { Link } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer/feedbackDrawer.store'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import {
  GITHUB_URL,
  GITHUB_LICENSE_URL,
  GITHUB_PROFILE_URL,
  SPONSORS_URL,
  PAYPAL_URL,
  csinsightUrl,
} from '@/lib/appLinks'

const LINK_CLASS =
  'text-muted-foreground underline-offset-2 transition-colors hover:text-foreground hover:underline'

interface ExternalLinkProps {
  href: string
  children: string
}

function ExternalLink({ href, children }: ExternalLinkProps) {
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" className={LINK_CLASS}>
      {children}
    </a>
  )
}

export interface AppFooterProps {
  variant?: 'full' | 'minimal'
}

export function AppFooter({ variant = 'full' }: AppFooterProps) {
  const locale = useAppShellStore((s) => s.locale)
  const openFeedback = useFeedbackDrawerStore((s) => s.open)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  if (variant === 'minimal') {
    return (
      <footer
        aria-label={t('common.footer.aria')}
        className="flex flex-col items-center gap-1.5 text-center text-xs text-muted-foreground"
      >
        <p>{t('common.footer.disclaimer')}</p>
        <p className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1">
          <Link to="/privacy" className={LINK_CLASS}>
            {t('common.footer.privacy')}
          </Link>
          <ExternalLink href={GITHUB_URL}>{t('common.footer.source_code')}</ExternalLink>
          <ExternalLink href={csinsightUrl(locale)}>{t('common.footer.csinsight')}</ExternalLink>
        </p>
        <p className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1">
          <ExternalLink href={SPONSORS_URL}>{t('common.footer.sponsors')}</ExternalLink>
          <ExternalLink href={PAYPAL_URL}>{t('common.footer.paypal')}</ExternalLink>
        </p>
      </footer>
    )
  }

  return (
    <footer
      aria-label={t('common.footer.aria')}
      className="mt-8 border-t border-border px-4 py-6 text-xs text-muted-foreground"
    >
      <div className="flex flex-wrap gap-x-12 gap-y-6">
        <nav className="flex flex-col gap-1.5">
          <h2 className="font-medium text-foreground">{t('common.footer.project_heading')}</h2>
          <ExternalLink href={GITHUB_URL}>{t('common.footer.source_code')}</ExternalLink>
          <button type="button" onClick={openFeedback} className={`${LINK_CLASS} text-left`}>
            {t('common.footer.report_issue')}
          </button>
          <Link to="/changelog" className={LINK_CLASS}>
            {t('common.footer.changelog')}
          </Link>
          <ExternalLink href={GITHUB_LICENSE_URL}>{t('common.footer.license')}</ExternalLink>
          <Link to="/privacy" className={LINK_CLASS}>
            {t('common.footer.privacy')}
          </Link>
        </nav>

        <nav className="flex max-w-xs flex-col gap-1.5">
          <h2 className="font-medium text-foreground">{t('common.footer.support_heading')}</h2>
          <p>{t('common.footer.support_note')}</p>
          <ExternalLink href={SPONSORS_URL}>{t('common.footer.sponsors')}</ExternalLink>
          <ExternalLink href={PAYPAL_URL}>{t('common.footer.paypal')}</ExternalLink>
        </nav>

        <nav className="flex flex-col gap-1.5">
          <h2 className="font-medium text-foreground">{t('common.footer.elsewhere_heading')}</h2>
          <ExternalLink href={GITHUB_PROFILE_URL}>{t('common.footer.developer')}</ExternalLink>
          <ExternalLink href={csinsightUrl(locale)}>{t('common.footer.csinsight')}</ExternalLink>
        </nav>
      </div>

      <p className="mt-6 flex flex-wrap gap-x-2 border-t border-border pt-4">
        <span>{t('common.footer.copyright')}</span>
        <span>{t('common.footer.disclaimer')}</span>
      </p>
    </footer>
  )
}
