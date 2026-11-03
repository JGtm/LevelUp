/**
 * SaveStatusBadge — le badge flottant d'enregistrement (état : `saveFeedbackStore`), monté une fois par
 * `SettingsPage`, dans le conteneur `relative` de la page.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { getSettingsText } from '@/features/settings/i18n'
import { useSaveFeedbackStore } from './saveFeedbackStore'

export function SaveStatusBadge() {
  const status = useSaveFeedbackStore((s) => s.status)
  const locale = useAppShellStore((s) => s.locale)
  if (!status) return null
  const t = getSettingsText(locale)
  return (
    <div className="pointer-events-none absolute right-6 top-4 z-10">
      {status === 'saved' ? (
        <span className="rounded-md bg-success/10 px-2 py-1 text-sm text-success shadow-sm" role="status" aria-live="polite">
          {t.savedStatus}
        </span>
      ) : (
        <span className="rounded-md bg-destructive/10 px-2 py-1 text-sm text-destructive shadow-sm" role="alert">
          {t.errorStatus}
        </span>
      )}
    </div>
  )
}
