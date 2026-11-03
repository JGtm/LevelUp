/**
 * AboutDialog — fenêtre « À propos » de l'application de bureau.
 *
 * Prend la place du pied de page du site (absent du chrome de bureau) : version, recherche de mise
 * à jour, liens du projet, soutien, dossier des données, mentions légales dont la mention imposée
 * par Microsoft. Modale accessible : rôle `dialog`, focus renvoyé au déclencheur à la fermeture,
 * Tab bouclé dans la fenêtre (et ramené dedans quand le focus en sort), Échap et clic sur le fond ferment.
 *
 * Sous la capacité `can_report_diagnostic`, le rapport de diagnostic (DiagnosticReportSection) crée l'archive
 * à joindre à un signalement.
 *
 * Les liens externes viennent de `lib/appLinks` (garde-rail `appLinks.guard.test.ts`) et se posent
 * SANS `target="_blank"` : la coquille ouvre tout lien web dans le navigateur par défaut et laisse
 * la fenêtre où elle est (navigation.rs).
 */
import { Link } from '@tanstack/react-router'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { DiagnosticReportSection } from '@/components/shell/DiagnosticReportSection'
import { MicrosoftNotice } from '@/components/shell/MicrosoftNotice'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer/feedbackDrawer.store'
import { APP_NAME } from '@/lib/appBrand'
import { requestUpdateCheck } from '@/lib/desktop/shellBridge'
import { useCopyToClipboard } from '@/lib/clipboard/useCopyToClipboard'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import {
  GITHUB_LICENSE_URL,
  GITHUB_URL,
  PAYPAL_URL,
  SPONSORS_URL,
  thirdPartyNoticesUrl,
} from '@/lib/appLinks'
import { useAppShellStore } from '@/stores/appShellStore'

const LINK_CLASS = 'block py-0.5 text-foreground underline-offset-2 hover:underline'
const BUTTON_CLASS =
  'rounded-md border border-border bg-card px-3 py-1.5 text-sm text-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:opacity-60'
const FOCUSABLE = 'a[href], button:not([disabled]), [tabindex]:not([tabindex="-1"])'

export interface AboutDialogProps {
  open: boolean
  onClose: () => void
}

type UpdateStatus = 'idle' | 'requested' | 'unreachable'

function Column({ heading, children }: { heading: string; children: ReactNode }) {
  return (
    <div>
      <h3 className="mb-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground">{heading}</h3>
      {children}
    </div>
  )
}

/** La fenêtre n'existe que fermée ou ouverte : son état (statut de la recherche, copie) repart de zéro à chaque ouverture. */
export function AboutDialog({ open, onClose }: AboutDialogProps) {
  return open ? <AboutDialogBody onClose={onClose} /> : null
}

function AboutDialogBody({ onClose }: { onClose: () => void }) {
  const locale = useAppShellStore((s) => s.locale)
  const appInfo = useAppShellStore((s) => s.appInfo)
  const openFeedback = useFeedbackDrawerStore((s) => s.open)
  const t = (key: CommonManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(commonManifest, key, locale, vars)
  const titleId = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  const titleRef = useRef<HTMLHeadingElement>(null)
  const { copy, copied } = useCopyToClipboard()
  const [update, setUpdate] = useState<UpdateStatus>('idle')
  const [checking, setChecking] = useState(false)

  // Focus : sur « Fermer » à l'ouverture, rendu au déclencheur à la fermeture.
  useEffect(() => {
    const opener = document.activeElement as HTMLElement | null
    closeRef.current?.focus()
    return () => opener?.focus()
  }, [])

  // Clavier posé sur la fenêtre du navigateur, pas sur le conteneur : quand le focus tombe sur
  // `body` (clic sur du texte, bouton qui se désactive), Échap et Tab doivent encore être tenus.
  useEffect(() => {
    function onKeyDown(e: globalThis.KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
        return
      }
      if (e.key !== 'Tab') return
      const items = Array.from(dialogRef.current?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? [])
      if (items.length === 0) return
      const first = items[0]
      const last = items[items.length - 1]
      const active = document.activeElement
      const inside = active !== null && dialogRef.current?.contains(active) === true
      if (!inside) {
        e.preventDefault()
        ;(e.shiftKey ? last : first).focus()
      } else if (e.shiftKey && active === first) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && active === last) {
        e.preventDefault()
        first.focus()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  async function checkForUpdate() {
    // Le bouton se désactive pendant la recherche : le focus passe au titre de la fenêtre au lieu de tomber sur `body`.
    titleRef.current?.focus()
    setChecking(true)
    const sent = await requestUpdateCheck()
    setChecking(false)
    setUpdate(sent ? 'requested' : 'unreachable')
  }

  function reportIssue() {
    onClose()
    openFeedback()
  }

  const version = appInfo?.version ?? ''
  const updateHint =
    update === 'requested'
      ? t('common.about.update_requested')
      : update === 'unreachable'
        ? t('common.about.update_unreachable')
        : t('common.about.update_hint')

  return (
    <div
      className="fixed inset-0 z-[70] grid place-items-center bg-black/55 p-4"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="flex max-h-full w-full max-w-xl flex-col gap-4 overflow-y-auto rounded-xl border border-border bg-background p-5 text-foreground shadow-xl"
      >
        <header className="flex items-center gap-3">
          <img src="/logo.png" alt="" className="h-10 w-10 shrink-0 object-contain" />
          <div className="min-w-0">
            <h2 ref={titleRef} id={titleId} tabIndex={-1} className="text-lg font-bold outline-none">
              {APP_NAME}
            </h2>
            {version && (
              <p className="font-mono text-xs text-muted-foreground" data-testid="about-version">
                {t('common.about.version', { version })}
              </p>
            )}
          </div>
          <button ref={closeRef} type="button" onClick={onClose} className={`${BUTTON_CLASS} ml-auto`}>
            {t('common.about.close')}
          </button>
        </header>

        <div className="flex flex-wrap items-center gap-3 rounded-lg border border-border bg-card px-3 py-2.5">
          <button
            type="button"
            onClick={() => void checkForUpdate()}
            disabled={checking}
            className="rounded-md bg-sidebar-primary px-3 py-1.5 text-sm text-sidebar-primary-foreground transition-colors hover:bg-sidebar-primary/90 disabled:opacity-60"
          >
            {t('common.about.check_update')}
          </button>
          <p role="status" className="min-w-0 flex-1 text-xs text-muted-foreground">
            {updateHint}
          </p>
        </div>

        <DiagnosticReportSection />

        <div className="grid gap-4 text-sm sm:grid-cols-3">
          <Column heading={t('common.footer.project_heading')}>
            <Link to="/changelog" onClick={onClose} className={LINK_CLASS}>
              {t('common.footer.changelog')}
            </Link>
            <a href={GITHUB_URL} rel="noopener noreferrer" className={LINK_CLASS}>
              {t('common.footer.source_code')}
            </a>
            <button type="button" onClick={reportIssue} className={`${LINK_CLASS} text-left`}>
              {t('common.footer.report_issue')}
            </button>
            <a href={GITHUB_LICENSE_URL} rel="noopener noreferrer" className={LINK_CLASS}>
              {t('common.footer.license')}
            </a>
            <a href={thirdPartyNoticesUrl(version)} rel="noopener noreferrer" className={LINK_CLASS}>
              {t('common.about.third_party')}
            </a>
          </Column>
          <Column heading={t('common.footer.support_heading')}>
            <a href={SPONSORS_URL} rel="noopener noreferrer" className={LINK_CLASS}>
              {t('common.footer.sponsors')}
            </a>
            <a href={PAYPAL_URL} rel="noopener noreferrer" className={LINK_CLASS}>
              {t('common.footer.paypal')}
            </a>
          </Column>
          <Column heading={t('common.about.data_heading')}>
            {appInfo?.data_root && (
              <div className="mb-1">
                <p className="text-xs text-muted-foreground">{t('common.about.data_folder')}</p>
                <p
                  className="break-all font-mono text-xs"
                  data-testid="about-data-root"
                  title={appInfo.data_root}
                >
                  {appInfo.data_root}
                </p>
                <button
                  type="button"
                  onClick={() => void copy(appInfo.data_root)}
                  className={`${BUTTON_CLASS} mt-1 px-2 py-0.5 text-xs`}
                >
                  {copied ? t('common.about.copied') : t('common.about.copy')}
                </button>
              </div>
            )}
            <Link to="/privacy" onClick={onClose} className={LINK_CLASS}>
              {t('common.footer.privacy')}
            </Link>
            <Link to="/terms" onClick={onClose} className={LINK_CLASS}>
              {t('common.footer.terms')}
            </Link>
          </Column>
        </div>

        <div className="flex flex-col gap-1 border-t border-border pt-3 text-xs text-muted-foreground">
          <MicrosoftNotice />
          <p>{t('common.about.ffmpeg_notice')}</p>
          <p>{t('common.about.sounds_notice')}</p>
          <p>{t('common.footer.copyright')}</p>
        </div>
      </div>
    </div>
  )
}
