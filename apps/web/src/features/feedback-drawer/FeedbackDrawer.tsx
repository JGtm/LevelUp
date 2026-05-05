/**
 * FeedbackDrawer — second panneau latéral droit, sous AssetDrawer.
 *
 * Pattern UI : reprend strictement AssetDrawer (mini-tab + panneau, transform
 * translateX, tokens sémantiques). Décalé verticalement avec un gap visible
 * sous le AssetDrawer.
 *
 * Submit → URL GitHub Issues préremplie via `window.open`. Une GitHub Action
 * post-création déclenche Claude Haiku pour affiner la classification.
 *
 * Toutes les couleurs passent par des tokens sémantiques (`bg-popover`,
 * `text-info`, etc.) — règle 20 CLAUDE.md.
 */
import { useEffect, useMemo, useState } from 'react'
import { useDeferredValue } from 'react'
import { toast } from 'sonner'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { formatMessage } from '@/lib/i18n/format'
import { feedbackDrawerManifest } from '@/lib/i18n/generated/feedback_drawer'
import {
  getRecentConsoleEntries,
  getRecentFailedRequests,
} from '@/lib/global-capture/buffers'
import { useFeedbackDrawerStore } from './feedbackDrawer.store'
import {
  classifyFeedback,
  type UserPickedType,
} from './classifyFeedback'
import { collectContext, describeFocusedElement } from './collectContext'
import { buildIssueUrl } from './buildIssueUrl'
import { useSimilarIssues } from './queries'
import {
  MAX_SUBMITS_PER_HOUR,
  getRemainingSubmits,
  recordSubmit,
} from './rateLimit'
import { log } from './_logger'

const TITLE_MAX_LENGTH = 80

export function FeedbackDrawer() {
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const { isOpen, close, toggle } = useFeedbackDrawerStore()

  const [pickedType, setPickedType] = useState<UserPickedType>('bug')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [showPreview, setShowPreview] = useState(false)
  // Tick incrémenté à chaque submit pour re-déclencher la lecture localStorage.
  const [submitTick, setSubmitTick] = useState(0)

  const t = (key: keyof typeof feedbackDrawerManifest, vars?: Record<string, string>) =>
    formatMessage(feedbackDrawerManifest, key, locale, vars)

  // Lecture directe localStorage à chaque render (coût microseconde) — évite
  // un useState + setState-in-effect (anti-pattern react-hooks).
  void submitTick // dépendance implicite : re-render après recordSubmit
  const remaining = isOpen ? getRemainingSubmits() : MAX_SUBMITS_PER_HOUR

  // Escape ferme
  useEffect(() => {
    if (!isOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') close()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [isOpen, close])

  // Focus auto sur le titre à l'ouverture
  useEffect(() => {
    if (!isOpen) return
    const id = window.setTimeout(() => {
      document.querySelector<HTMLInputElement>('#feedback-drawer-title-input')?.focus()
    }, 50)
    return () => window.clearTimeout(id)
  }, [isOpen])

  const deferredDescription = useDeferredValue(description)
  const deferredTitle = useDeferredValue(title)

  const similarQuery = useSimilarIssues(deferredTitle, isOpen)

  const builtUrl = useMemo(() => {
    if (!isOpen) return null
    const browser = {
      url: window.location.pathname + window.location.search,
      pathname: window.location.pathname,
      userAgent: navigator.userAgent,
      viewportWidth: window.innerWidth,
      viewportHeight: window.innerHeight,
      locale,
      theme,
      timestampIso: new Date().toISOString(),
      focusedElement: describeFocusedElement(),
    }
    const ctx = collectContext({
      browser,
      shell: {
        titleSlug,
        playerSlug: currentPlayer?.gamertag ?? null,
        appVersion: null,
      },
      filters: filterContext,
      console: getRecentConsoleEntries(),
      failedRequests: getRecentFailedRequests(),
    })
    const classification = classifyFeedback(
      { pickedType, description: deferredDescription },
      { pathname: browser.pathname, recentConsole: ctx.console },
    )
    return buildIssueUrl({
      title: title.trim() || '_(sans titre)_',
      description: deferredDescription,
      context: ctx,
      classification,
    })
  }, [
    isOpen,
    locale,
    theme,
    titleSlug,
    currentPlayer,
    filterContext,
    pickedType,
    deferredDescription,
    title,
  ])

  function handleSubmit() {
    if (!builtUrl) return
    if (remaining === 0) return
    const allowed = recordSubmit()
    setSubmitTick((n) => n + 1)
    if (!allowed) return
    if (builtUrl.wasTruncated) log.warn('url:truncated', 'feedback URL body truncated')
    // Observabilité dev : labels (type/severity/area) + taille URL.
    // Pas de PII : pas de titre, pas de description, pas de gamertag.
    const labels = new URL(builtUrl.url).searchParams.get('labels') ?? ''
    log.info('feedback submitted', { labels, urlLength: builtUrl.url.length })
    const opened = window.open(
      builtUrl.url,
      '_blank',
      'noopener,noreferrer',
    )
    if (!opened) {
      log.error('clipboard:open_failed', 'window.open returned null (popup blocked)')
      copyToClipboardWithToast(builtUrl.url, t('feedback_drawer.popup_blocked'))
      return
    }
    close()
    setTitle('')
    setDescription('')
  }

  function copyToClipboardWithToast(url: string, message: string) {
    if (typeof navigator === 'undefined' || !navigator.clipboard?.writeText) {
      // Contexte non-secure (http://) ou clipboard indisponible : fail visible.
      toast.error(message)
      return
    }
    navigator.clipboard
      .writeText(url)
      .then(() => toast.info(message))
      .catch(() => toast.error(message))
  }

  const submitDisabled = !title.trim() || remaining === 0
  const similarItems = similarQuery.data ?? []

  return (
    <>
      <button
        type="button"
        onClick={toggle}
        aria-label={t(isOpen ? 'feedback_drawer.mini_tab.aria_close' : 'feedback_drawer.mini_tab.aria_open')}
        aria-expanded={isOpen}
        className={`fixed right-0 top-[calc(50%+125px)] z-40 -translate-y-1/2 ${isOpen ? 'hidden' : 'hidden sm:flex'} h-9 w-10 cursor-pointer select-none items-center justify-center rounded-l border border-r-0 border-border bg-popover text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground`}
      >
        <ChatBubbleIcon />
      </button>

      {isOpen && (
        <div
          className="fixed inset-0 z-[49]"
          onClick={close}
          aria-hidden="true"
        />
      )}

      <div
        role="complementary"
        aria-label={t('feedback_drawer.title')}
        aria-hidden={!isOpen}
        className="fixed right-0 top-1/2 z-50 hidden h-[min(540px,70vh)] w-[340px] flex-col rounded-l-lg border border-r-0 border-border bg-popover shadow-xl ring-1 ring-border transition-transform duration-200 ease-out sm:flex"
        style={{
          transform: isOpen
            ? 'translateX(0) translateY(-50%)'
            : 'translateX(100%) translateY(-50%)',
        }}
      >
        <div className="flex shrink-0 items-center justify-between border-b border-border px-3 py-2">
          <h2 className="text-sm font-semibold text-popover-foreground">
            {t('feedback_drawer.title')}
          </h2>
          <button
            type="button"
            onClick={close}
            aria-label={t('feedback_drawer.mini_tab.aria_close')}
            className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
          >
            <CloseIcon />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-3 py-3">
          <div className="mb-3 flex gap-1" role="tablist" aria-label={t('feedback_drawer.type.aria')}>
            <TypeButton
              active={pickedType === 'bug'}
              onClick={() => setPickedType('bug')}
              label={t('feedback_drawer.type.bug')}
            />
            <TypeButton
              active={pickedType === 'enhancement'}
              onClick={() => setPickedType('enhancement')}
              label={t('feedback_drawer.type.idea')}
            />
            <TypeButton
              active={pickedType === 'question'}
              onClick={() => setPickedType('question')}
              label={t('feedback_drawer.type.question')}
            />
          </div>

          <label className="mb-1 block text-xs font-medium text-muted-foreground">
            {t('feedback_drawer.field.title')}
          </label>
          <input
            id="feedback-drawer-title-input"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value.slice(0, TITLE_MAX_LENGTH))}
            placeholder={t('feedback_drawer.field.title_placeholder')}
            className="mb-2 w-full rounded border border-border bg-background px-2 py-1 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            maxLength={TITLE_MAX_LENGTH}
          />

          {similarItems.length > 0 && (
            <div className="mb-3 rounded border border-border bg-muted/30 p-2 text-xs">
              <p className="mb-1 text-muted-foreground">
                {t('feedback_drawer.similar.label')}
              </p>
              <ul className="space-y-1">
                {similarItems.map((it) => (
                  <li key={it.number}>
                    <a
                      href={it.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-info hover:underline"
                    >
                      #{it.number} — {it.title}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <label className="mb-1 block text-xs font-medium text-muted-foreground">
            {t('feedback_drawer.field.description')}
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t('feedback_drawer.field.description_placeholder')}
            rows={5}
            className="mb-3 w-full resize-none rounded border border-border bg-background px-2 py-1 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />

          <label className="mb-2 flex items-center gap-2 text-xs text-popover-foreground">
            <input
              type="checkbox"
              checked={showPreview}
              onChange={(e) => setShowPreview(e.target.checked)}
              className="rounded border-border"
            />
            {t('feedback_drawer.attach.label')}
          </label>

          {showPreview && builtUrl && (
            <details open className="mb-3 rounded border border-border bg-muted/30">
              <summary className="cursor-pointer px-2 py-1 text-xs font-medium text-foreground">
                {t('feedback_drawer.attach.preview_summary')}
              </summary>
              <pre className="max-h-48 overflow-auto px-2 py-1 text-[10px] leading-tight text-muted-foreground">
                {builtUrl.body}
              </pre>
            </details>
          )}
        </div>

        <div className="shrink-0 border-t border-border px-3 py-2">
          <button
            type="button"
            onClick={handleSubmit}
            disabled={submitDisabled}
            className="w-full rounded bg-accent px-3 py-1.5 text-sm font-medium text-accent-foreground hover:bg-accent/80 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {t('feedback_drawer.submit')}
          </button>
          <p className="mt-1 text-[10px] text-muted-foreground">
            {remaining === 0
              ? t('feedback_drawer.rate_limit')
              : t('feedback_drawer.submit_note')}
          </p>
        </div>
      </div>
    </>
  )
}

function TypeButton({
  active,
  label,
  onClick,
}: {
  active: boolean
  label: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={`flex-1 rounded px-2 py-1 text-xs font-medium transition-colors ${
        active
          ? 'bg-accent text-accent-foreground'
          : 'text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground'
      }`}
    >
      {label}
    </button>
  )
}

function ChatBubbleIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      className="h-4 w-4"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
    >
      <path
        fillRule="evenodd"
        d="M18 10c0 3.866-3.582 7-8 7a8.84 8.84 0 01-3.6-.745L3 17l1.395-3.72C3.512 12.4 3 11.247 3 10c0-3.866 3.582-7 7-7s8 3.134 8 7zM7 9a1 1 0 100 2 1 1 0 000-2zm3 0a1 1 0 100 2 1 1 0 000-2zm3 0a1 1 0 100 2 1 1 0 000-2z"
        clipRule="evenodd"
      />
    </svg>
  )
}

function CloseIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      className="h-4 w-4"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
    >
      <path
        fillRule="evenodd"
        d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
        clipRule="evenodd"
      />
    </svg>
  )
}
