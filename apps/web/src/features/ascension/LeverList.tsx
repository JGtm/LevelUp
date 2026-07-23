/**
 * LeverList — Phase 3 : leviers calibrés Pattern Engine.
 *
 * 3 leviers patterns prioritaires : métrique courante → cible, barre de
 * progression, horizon en matchs. Distinct du LeveragePanel (ProfileService.C).
 */
import type { PatternLever } from './types'
import type { AscensionText } from './i18n'

interface LeverListProps {
  levers: PatternLever[]
  t: AscensionText
}

export function LeverList({ levers, t }: LeverListProps) {
  const top = levers.filter((l) => l.rank <= 3 && l.impact > 0.3)
  if (top.length === 0) return null

  return (
    <div className="space-y-2">
      {top.map((lev) => (
        <LeverCard key={`${lev.axis}-${lev.rank}`} lever={lev} t={t} />
      ))}
    </div>
  )
}

/**
 * leverContextValue — libellé affichable du contexte visé par un levier (F3).
 * by_map : nom de carte résolu servi par le backend (`context_label`), JAMAIS le
 * GUID de `context_key`. by_squad : « solo »/« escouade » via i18n. by_mode : la
 * clé est déjà un libellé de mode normalisé. Vide pour les leviers comportementaux.
 */
function leverContextValue(lev: PatternLever, t: AscensionText): string {
  if (lev.context_label) return lev.context_label
  const key = lev.context_key
  if (!key) return ''
  if (key === 'with_friends') return t.squadVsSoloSquad
  if (key === 'solo') return t.squadVsSoloSolo
  return key
}

/**
 * leverPhrase — compose la phrase du levier via le gabarit i18n de son axe (F3).
 * Le backend ne sert plus de phrase : `{context}` est interpolé côté front avec
 * le libellé du contexte visé. Repli sur le libellé d'axe si l'axe n'a pas de
 * gabarit (jamais la clé brute).
 */
function leverPhrase(lev: PatternLever, t: AscensionText): string {
  const template = t.leverPhrase?.[lev.axis]
  if (!template) return t.leverAxis?.[lev.axis] ?? lev.axis
  return template.replace('{context}', leverContextValue(lev, t))
}

function LeverCard({ lever: lev, t }: { lever: PatternLever; t: AscensionText }) {
  const axisLabel = t.leverAxis?.[lev.axis] ?? lev.axis
  const phrase = leverPhrase(lev, t)
  // Valeur courante absente (ex leviers comportementaux : tilt, fatigue) → on
  // masque la ligne « Actuel → Cible » plutôt que d'afficher « — → — » (A7).
  const hasCurrent = lev.current_val > 0
  const progress = lev.target_val > 0
    ? Math.min((lev.current_val / lev.target_val) * 100, 100)
    : 0
  const impactPct = Math.round(lev.impact * 100)

  return (
    <div className="rounded-md border border-border bg-card p-3">
      <div className="mb-1 flex items-start justify-between gap-2">
        <div>
          <p className="text-sm font-semibold">{axisLabel}</p>
          <p className="text-xs text-muted-foreground">{phrase}</p>
        </div>
        <span className="shrink-0 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-bold text-primary">
          {impactPct}% {t.leverImpact ?? 'impact'}
        </span>
      </div>

      <div className="my-2 h-1.5 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full bg-primary/60 transition-all"
          style={{ width: `${progress}%` }}
        />
      </div>

      <div className="flex items-center justify-between text-xs text-muted-foreground">
        {hasCurrent && (
          <span>
            {t.leverCurrent ?? 'Actuel'} <strong className="text-foreground">{fmt(lev.current_val)}</strong>
            {' → '}{t.leverTarget ?? 'Cible'} <strong className="text-foreground">{fmt(lev.target_val)}</strong>
          </span>
        )}
        <span className={hasCurrent ? '' : 'ml-auto'}>~{lev.horizon} {t.leverHorizonMatches ?? 'matchs'}</span>
      </div>
    </div>
  )
}

function fmt(v: number): string {
  if (v === 0) return '—'
  // Win rates (0..1) displayed as %
  if (v > 0 && v <= 1 && String(v).includes('.')) {
    return `${Math.round(v * 100)}%`
  }
  return v.toFixed(2)
}
