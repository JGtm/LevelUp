/**
 * PatternContextGrid — Phase 1 : grille des patterns contextuels (mode/map/squad).
 *
 * Badge Signal (strength/weakness/neutral) + win rate + OC/DR delta
 * pour chaque contexte (mode, map, squad).
 */
import type { ContextualPattern, ContextType } from './types'
import type { AscensionText } from './i18n'

interface PatternContextGridProps {
  patterns: ContextualPattern[]
  t: AscensionText
}

const CONTEXT_ORDER: ContextType[] = ['by_mode', 'by_map', 'by_squad']

export function PatternContextGrid({ patterns, t }: PatternContextGridProps) {
  if (patterns.length === 0) return null

  const byType = CONTEXT_ORDER.map((ct) => ({
    type: ct,
    items: patterns
      .filter((p) => p.type === ct)
      .sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta)),
  })).filter((g) => g.items.length > 0)

  return (
    <div className="space-y-4">
      {byType.map(({ type, items }) => (
        <div key={type}>
          <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {t.contextType?.[type] ?? type}
          </p>
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {items.map((p) => (
              <ContextCard key={`${p.type}-${p.key}`} pattern={p} t={t} />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function ContextCard({ pattern: p, t }: { pattern: ContextualPattern; t: AscensionText }) {
  const signalClass = {
    strength: 'border-green-500/40 bg-green-500/5', // color-allow: signal state — CLAUDE.md §20
    weakness: 'border-red-500/40 bg-red-500/5', // color-allow: signal state — CLAUDE.md §20
    neutral: 'border-border bg-card',
  }[p.signal]

  const deltaSign = p.delta >= 0 ? '+' : ''
  const deltaClass =
    p.delta > 0
      ? 'text-green-600 dark:text-green-400' // color-allow: delta trend — CLAUDE.md §20
      : p.delta < 0
        ? 'text-red-500 dark:text-red-400' // color-allow: delta trend — CLAUDE.md §20
        : 'text-muted-foreground'

  return (
    <div className={`rounded-md border p-3 ${signalClass}`}>
      <div className="mb-1 flex items-center justify-between">
        <span className="text-sm font-semibold">{contextLabel(p, t)}</span>
        <SignalBadge signal={p.signal} t={t} />
      </div>
      <div className="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
        <span>
          {t.patternWinRate} <strong className="text-foreground">{pct(p.win_rate)}</strong>
          {' '}<span className={deltaClass}>({deltaSign}{pct(p.delta)})</span>
        </span>
        <span>OC <strong className="text-foreground">{Math.round(p.avg_oc * 100)}%</strong></span>
        <span>DR <strong className="text-foreground">{Math.round((p.avg_dr - 1) * 100)}%</strong></span>
        <span className="text-muted-foreground/70">{p.match_count} {t.patternMatches}</span>
      </div>
    </div>
  )
}

function SignalBadge({ signal, t }: { signal: ContextualPattern['signal']; t: AscensionText }) {
  const map = {
    strength: {
      label: t.signalStrength ?? 'Force',
      cls: 'bg-green-500/20 text-green-700 dark:text-green-300', // color-allow: signal state — CLAUDE.md §20
    },
    weakness: {
      label: t.signalWeakness ?? 'Faiblesse',
      cls: 'bg-red-500/20 text-red-700 dark:text-red-300', // color-allow: signal state — CLAUDE.md §20
    },
    neutral: {
      label: t.signalNeutral ?? 'Neutre',
      cls: 'bg-muted text-muted-foreground',
    },
  }[signal]

  return (
    <span className={`rounded px-1.5 py-0.5 text-[10px] font-semibold ${map.cls}`}>
      {map.label}
    </span>
  )
}

/**
 * contextLabel — libellé lisible d'un pattern contextuel selon son type :
 *  - by_map   : nom de carte résolu côté backend (`label`), jamais le GUID de `key`.
 *  - by_squad : libellé i18n (Solo / Escouade) mappé depuis la clé technique.
 *  - by_mode  : la clé est déjà un libellé de mode normalisé.
 */
function contextLabel(p: ContextualPattern, t: AscensionText): string {
  if (p.type === 'by_map') return p.label ?? p.key
  if (p.type === 'by_squad') return p.key === 'with_friends' ? t.squadVsSoloSquad : t.squadVsSoloSolo
  return p.key
}

function pct(v: number): string {
  return `${Math.round(v * 100)}%`
}
