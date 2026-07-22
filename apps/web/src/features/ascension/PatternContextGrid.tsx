/**
 * PatternContextGrid — grille des patterns contextuels (mode / carte).
 *
 * Badge Signal (strength/weakness/neutral) + win rate + OC/DR delta par contexte.
 * La comparaison Solo/Escouade (by_squad) n'apparaît PAS dans la grille : elle
 * est rendue une seule fois par SquadVsSoloCard (dédoublonnage B7).
 */
import type { CascadeInput } from '@/lib/api/types'
import { buildSoloFilterLink } from '@/features/filters/filterLink'
import type { ContextualPattern, ContextType } from './types'
import type { AscensionText } from './i18n'
import { CombatAbbr } from './CombatAbbr'

interface PatternContextGridProps {
  patterns: ContextualPattern[]
  t: AscensionText
  /** Seuil de matchs servi par le backend (DEC-8) — jamais codé en dur ici. */
  minMatchesForSignal: number
  /** Contexte pour construire les liens « voir les matchs » (C5). */
  playerSlug: string
  titleSlug: string
}

// by_squad exclu : la comparaison Solo/Escouade vit dans SquadVsSoloCard (B7).
const CONTEXT_ORDER: ContextType[] = ['by_mode', 'by_map']

export function PatternContextGrid({
  patterns,
  t,
  minMatchesForSignal,
  playerSlug,
  titleSlug,
}: PatternContextGridProps) {
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
              <ContextCard
                key={`${p.type}-${p.key}`}
                pattern={p}
                t={t}
                lowSample={p.match_count < minMatchesForSignal}
                href={soloLinkForPattern(p, playerSlug, titleSlug)}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

/**
 * soloLinkForPattern — deep-link Solo filtré pour une carte pattern (F7).
 *
 * Le lien se construit TOUJOURS sur `filter_key` — la clé de filtrage stable
 * servie par le backend, exactement ce que le pipeline de filtres matche
 * (indépendante de la locale). by_map : nom FR-first (jamais le GUID `key` ni le
 * `label` localisé, qui ne matcherait pas en EN) ; by_mode : mode normalisé.
 * Sans `filter_key` résolu → pas de lien (plutôt qu'un filtre qui ne matche rien).
 * by_squad n'atteint pas la grille.
 */
function soloLinkForPattern(p: ContextualPattern, playerSlug: string, titleSlug: string): string | undefined {
  const key = p.filter_key
  if (!key) return undefined
  let cascade: Partial<CascadeInput> | null = null
  if (p.type === 'by_mode') {
    cascade = { modes: [key] }
  } else if (p.type === 'by_map') {
    cascade = { maps: [key] }
  }
  if (!cascade) return undefined
  return buildSoloFilterLink({ playerSlug, titleSlug, cascade })
}

function ContextCard({
  pattern: p,
  t,
  lowSample,
  href,
}: {
  pattern: ContextualPattern
  t: AscensionText
  lowSample: boolean
  href?: string
}) {
  // DEC-8 : sous le seuil de matchs, on neutralise le signal (bordure + delta)
  // et on affiche un badge « Échantillon faible » à la place de Force/Faiblesse.
  const signalClass = lowSample
    ? 'border-border bg-card'
    : {
        strength: 'border-green-500/40 bg-green-500/5', // color-allow: signal state — CLAUDE.md §20
        weakness: 'border-red-500/40 bg-red-500/5', // color-allow: signal state — CLAUDE.md §20
        neutral: 'border-border bg-card',
      }[p.signal]

  const deltaSign = p.delta >= 0 ? '+' : ''
  const deltaClass = lowSample
    ? 'text-muted-foreground'
    : p.delta > 0
      ? 'text-green-600 dark:text-green-400' // color-allow: delta trend — CLAUDE.md §20
      : p.delta < 0
        ? 'text-red-500 dark:text-red-400' // color-allow: delta trend — CLAUDE.md §20
        : 'text-muted-foreground'

  const body = (
    <>
      <div className="mb-1 flex items-center justify-between">
        <span className="text-sm font-semibold">{contextLabel(p)}</span>
        <SignalBadge signal={p.signal} lowSample={lowSample} t={t} />
      </div>
      <div className="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
        <span>
          {t.patternWinRate} <strong className="text-foreground">{pct(p.win_rate)}</strong>
          {' '}<span className={deltaClass}>({deltaSign}{pct(p.delta)})</span>
        </span>
        <span><CombatAbbr metric="oc" t={t} /> <strong className="text-foreground">{Math.round(p.avg_oc * 100)}%</strong></span>
        <span><CombatAbbr metric="dr" t={t} /> <strong className="text-foreground">{Math.round((p.avg_dr - 1) * 100)}%</strong></span>
        <span className="text-muted-foreground/70">{p.match_count} {t.patternMatches}</span>
      </div>
    </>
  )

  const cardClass = `block rounded-md border p-3 ${signalClass}`
  // Navigation PLEINE PAGE (`<a href>`, pas `<Link>`) : le `?f=` n'est décodé
  // qu'au rehydrate du store solo (cf. filterLink). Sans href → carte statique.
  if (href) {
    return (
      <a href={href} className={`${cardClass} transition-colors hover:border-primary/60`} title={t.patternSeeMatches}>
        {body}
      </a>
    )
  }
  return <div className={cardClass}>{body}</div>
}

function SignalBadge({ signal, lowSample, t }: { signal: ContextualPattern['signal']; lowSample: boolean; t: AscensionText }) {
  // DEC-8 : échantillon insuffisant → badge neutre, jamais Force/Faiblesse.
  if (lowSample) {
    return (
      <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] font-semibold text-muted-foreground">
        {t.signalLowSample}
      </span>
    )
  }
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
 *  - by_map  : nom de carte résolu côté backend (`label`), jamais le GUID de `key`.
 *  - by_mode : la clé est déjà un libellé de mode normalisé.
 * (by_squad n'atteint jamais la grille — cf. CONTEXT_ORDER.)
 */
function contextLabel(p: ContextualPattern): string {
  if (p.type === 'by_map') return p.label ?? p.key
  return p.key
}

function pct(v: number): string {
  return `${Math.round(v * 100)}%`
}
