/**
 * ActionLastRun — ligne « Dernière exécution : <relatif> · <déclencheur> » sous
 * un bouton d'action globale (ou « Jamais exécutée »). Issue en échec → suffixe
 * en token destructive. Lecture via le journal partagé (actionJournal.ts).
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { adminRelativeTime, adminAbsoluteTime } from './format'
import { useAdminT, useAdminLocale } from './useAdminText'
import { describeActionRun } from './actionJournalDisplay'
import { useActionJournal } from './actionJournal'

export function ActionLastRun({ action }: { action: string }) {
  const { data } = useActionJournal()
  const tA = useAdminT()
  const locale = useAdminLocale()
  const display = describeActionRun(data?.actions?.find((a) => a.action === action))

  if (!display) {
    return <p className="text-xs text-muted-foreground/70">{tA('admin.actions.never_run')}</p>
  }
  return (
    <p className="text-xs text-muted-foreground" title={adminAbsoluteTime(display.at, locale)}>
      {tA('admin.actions.last_run')} : {adminRelativeTime(display.at, locale)} · {tA(display.triggerKey)}
      {display.failed && (
        <span className="ml-1 font-medium" style={{ color: tokenCssVar('destructive') }}>
          ({tA('admin.actions.outcome_error')})
        </span>
      )}
    </p>
  )
}
