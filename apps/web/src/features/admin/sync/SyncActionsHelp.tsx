/**
 * SyncActionsHelp — légende explicative de la portée des différentes actions de
 * synchronisation exposées dans l'app (page admin + réglages). Répond à la
 * confusion « forcer le planificateur vs delta » : chaque action a une portée
 * distincte, rappelée ici en une ligne.
 */
import type { TAdmin } from '../useAdminText'

export function SyncActionsHelp({ tA }: { tA: TAdmin }) {
  const rows: Array<{ term: string; desc: string }> = [
    { term: tA('admin.sync.help_forced_cycle_term'), desc: tA('admin.sync.help_forced_cycle') },
    { term: tA('admin.sync.help_manual_all_term'), desc: tA('admin.sync.help_manual_all') },
    { term: tA('admin.sync.help_convergence_term'), desc: tA('admin.sync.help_convergence') },
    { term: tA('admin.sync.help_initial_term'), desc: tA('admin.sync.help_initial') },
  ]
  return (
    <div className="rounded-md border border-border/50 bg-muted/30 p-3">
      <p className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {tA('admin.sync.help_title')}
      </p>
      <dl className="space-y-1 text-sm">
        {rows.map((r) => (
          <div key={r.term} className="flex flex-col gap-0.5 sm:flex-row sm:gap-2">
            <dt className="shrink-0 font-medium text-foreground">{r.term}</dt>
            <dd className="text-muted-foreground">{r.desc}</dd>
          </div>
        ))}
      </dl>
    </div>
  )
}
