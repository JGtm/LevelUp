/**
 * ExplorerRankedBlock — bloc « Classement » du briefing (DP-3 : cellule SIBLING de la
 * grille « Par… », plus empilé en 4e colonne).
 *
 * Rendu comme BriefingSectionCard (cellule de la grille adaptative « Par… », aux côtés
 * des dimensions et de « Par contexte » — ExplorerBriefingModules). UNE ligne PAR CHAÎNE
 * `(rating_type, playlist_group)` de `ranked.kinds` (jamais de flèche inter-chaînes, DEC-1) :
 * « {TYPE}{ · label de chaîne si le type a ≥ 2 chaînes} · {progression début → fin}
 * · {±x pt/match coloré} ». Le label de chaîne est résolu par `lusrChainLabel`
 * (career). Les helpers de progression/placement (D-C/D-D) vivent ici (déplacés de
 * ExplorerBriefingTiles en V4 Phase 4). Tokens sémantiques uniquement.
 */
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import { lusrChainLabel } from '@/features/career/lusr-chains'
import { formatSignedFixed } from '@/lib/formatters'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { ExplorerBriefingRanked, ExplorerBriefingRankedKind } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { BriefingSectionCard } from './BriefingSectionCard'
import { deltaToken } from './ExplorerBriefing.logic'

type T = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

// rankedProgression compose « palier début → palier fin » (D-C), en résolvant les
// paliers de placement via clés i18n (D-D : jamais parser le libellé FR). Null si
// aucun palier n'est résolvable ; paliers égaux → palier seul.
function rankedProgression(k: ExplorerBriefingRankedKind, t: T): string | null {
  const start = k.tier_start_is_placement
    ? t('explorer.briefing.placement')
    : (k.tier_start_label ?? null)
  const end =
    k.tier_end_placement_remaining != null
      ? t('explorer.briefing.placement_remaining', { n: k.tier_end_placement_remaining })
      : (k.tier_end_label ?? null)
  if (start == null && end == null) return null
  if (start != null && end != null) return start === end ? start : `${start} → ${end}`
  return start ?? end
}

/** Libellé « {±x} pt/match » de la chaîne, ou null si aucun delta par match. */
function perMatchLabel(k: ExplorerBriefingRankedKind, t: T): string | null {
  return k.delta_per_match != null
    ? t('explorer.briefing.ranked_per_match', { delta: formatSignedFixed(k.delta_per_match, 1) })
    : null
}

// Une ligne par chaîne. `showChain` : le TYPE possède ≥ 2 chaînes → on affiche le
// label de chaîne (lusrChainLabel) pour distinguer (sinon le type suffit, ex. CSR).
function RankedChainRow({
  k,
  showChain,
  t,
  locale,
}: {
  k: ExplorerBriefingRankedKind
  showChain: boolean
  t: T
  locale: ManifestLocale
}) {
  const prog = rankedProgression(k, t)
  const pm = perMatchLabel(k, t)
  const chainLabel = showChain && k.playlist_group ? lusrChainLabel(k.playlist_group, locale) : null
  return (
    <li className="flex flex-wrap items-baseline gap-x-1.5 text-xs">
      <span className="font-semibold uppercase text-foreground">{k.kind}</span>
      {chainLabel != null && <span className="text-muted-foreground">· {chainLabel}</span>}
      {prog != null && <span className="text-muted-foreground">· {prog}</span>}
      {pm != null && (
        <span className="tabular-nums" style={{ color: tokenCssVar(deltaToken(k.delta_per_match)) }}>
          · {pm}
        </span>
      )}
    </li>
  )
}

// Bloc Classement (DEC-RANK-FE) : une ligne par chaîne de `ranked.kinds`. Le gate
// capability 'ranked' + présence du DTO est assuré par l'appelant (Modules).
export function RankedBlock({
  ranked,
  t,
  locale,
}: {
  ranked: ExplorerBriefingRanked
  t: T
  locale: ManifestLocale
}) {
  const kinds = ranked.kinds ?? []
  if (kinds.length === 0) return null
  // Nombre de chaînes par type → afficher le label de chaîne dès qu'un type en a ≥ 2.
  const countByType = new Map<string, number>()
  for (const k of kinds) countByType.set(k.kind, (countByType.get(k.kind) ?? 0) + 1)
  return (
    <BriefingSectionCard
      className="h-full"
      title={
        <span className="inline-flex items-center gap-1.5">
          {t('explorer.briefing.ranked_title')}
          <InfoTooltip content={t('explorer.briefing.tip_ranked')} iconClass="w-3.5 h-3.5" />
        </span>
      }
    >
      <ul className="space-y-1">
        {kinds.map((k) => (
          <RankedChainRow
            key={`${k.kind}:${k.playlist_group ?? ''}`}
            k={k}
            showChain={(countByType.get(k.kind) ?? 0) >= 2}
            t={t}
            locale={locale}
          />
        ))}
      </ul>
    </BriefingSectionCard>
  )
}
