/**
 * ExplorerBriefingTiles — tuiles KPI « Classement » et « Séries » du socle.
 *
 * V3 (compaction) : anciennes cartes pleine largeur RankedCard/StreaksCard
 * converties en tuiles compactes du socle (DP-2, DP-3), réutilisant BriefingTile.
 * `rankedProgression` (composition « début → fin », D-C/D-D) déplacé ici depuis
 * ExplorerBriefingModules. Tokens sémantiques uniquement (aucune couleur hex).
 */
import type { ReactNode } from 'react'

import { InfoTooltip } from '@/components/ui/info-tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import type {
  ExplorerBriefingRanked,
  ExplorerBriefingRankedKind,
  ExplorerBriefingStreaks,
} from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { BriefingTile } from './BriefingTile'
import { deltaToken, formatSignedFixed } from './ExplorerBriefing.logic'

type T = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

// rankedProgression compose « palier début → palier fin » (D-C), en résolvant les
// paliers de placement via clés i18n (D-D : jamais parser le libellé FR). Null si
// aucun palier n'est résolvable (segment omis). Paliers égaux → palier seul.
// Local (non exporté) : consommé uniquement par la 2e ligne du sous-texte
// (`react-refresh/only-export-components` — le fichier n'exporte que des composants).
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

/** Libellé « {±x} pt/match » du type, ou null si aucun delta par match. */
function perMatchLabel(k: ExplorerBriefingRankedKind, t: T): string | null {
  return k.delta_per_match != null
    ? t('explorer.briefing.ranked_per_match', { delta: formatSignedFixed(k.delta_per_match, 1) })
    : null
}

/** Valeur de la tuile = palier de FIN du type (placement restant / label / pt/match / —). */
function rankedValue(k: ExplorerBriefingRankedKind, t: T): ReactNode {
  if (k.tier_end_placement_remaining != null)
    return t('explorer.briefing.placement_remaining', { n: k.tier_end_placement_remaining })
  if (k.tier_end_label != null) return k.tier_end_label
  return perMatchLabel(k, t) ?? '—'
}

/** « depuis {palier de début} » du type MAJORITAIRE, ou null si début non résolvable. */
function sinceLabel(k: ExplorerBriefingRankedKind, t: T): string | null {
  const start = k.tier_start_is_placement
    ? t('explorer.briefing.placement')
    : (k.tier_start_label ?? null)
  return start != null ? t('explorer.briefing.ranked_since', { tier: start }) : null
}

// Ligne de sous-texte d'un type : « TYPE · {middle} · {±x} pt/match » (segments
// non résolvables omis). Le pt/match est coloré via le token de delta canonique.
function rankedSubLine(k: ExplorerBriefingRankedKind, middle: string | null, t: T): ReactNode {
  const pm = perMatchLabel(k, t)
  return (
    <>
      <span className="font-semibold uppercase text-foreground">{k.kind}</span>
      {middle != null && <> · {middle}</>}
      {pm != null && (
        <>
          {' · '}
          <span className="tabular-nums" style={{ color: tokenCssVar(deltaToken(k.delta_per_match)) }}>
            {pm}
          </span>
        </>
      )}
    </>
  )
}

// Tuile Classement (DP-2) : valeur = palier de fin du type MAJORITAIRE (kinds[0],
// déjà ordonné Count desc). Sous-texte : ligne 1 du type majoritaire (« depuis
// {début} »), ligne 2 (si un 2e type) en progression compacte. JAMAIS croiser les
// paliers de deux types. Gate useCapability('ranked') assuré par l'appelant (Strip).
export function RankedTile({ ranked, t }: { ranked: ExplorerBriefingRanked; t: T }) {
  const kinds = ranked.kinds ?? []
  if (kinds.length === 0) return null
  const primary = kinds[0]
  const secondary = kinds[1]
  return (
    <BriefingTile
      label={t('explorer.briefing.ranked_title')}
      info={<InfoTooltip content={t('explorer.briefing.tip_ranked')} iconClass="w-3.5 h-3.5" />}
      value={rankedValue(primary, t)}
      sub={
        <>
          <div className="truncate">{rankedSubLine(primary, sinceLabel(primary, t), t)}</div>
          {secondary != null && (
            <div className="truncate">{rankedSubLine(secondary, rankedProgression(secondary, t), t)}</div>
          )}
        </>
      }
    />
  )
}

// Tuile Séries (DP-3) : valeur bicolore « {best} V / {worst} D » (tokens
// outcome-win/outcome-loss). Segment à zéro omis ; la tuile elle-même est omise
// par l'appelant (Strip) quand les deux segments sont à zéro.
export function StreaksTile({ streaks, t }: { streaks: ExplorerBriefingStreaks; t: T }) {
  const best = streaks.best_win_streak ?? 0
  const worst = streaks.worst_loss_streak ?? 0
  return (
    <BriefingTile
      label={t('explorer.briefing.streaks_title')}
      info={<InfoTooltip content={t('explorer.briefing.tip_streaks')} iconClass="w-3.5 h-3.5" />}
      value={
        <span className="inline-flex items-baseline gap-1">
          {best > 0 && (
            <span style={{ color: tokenCssVar('outcome-win') }}>
              {t('explorer.briefing.streak_wins', { n: best })}
            </span>
          )}
          {best > 0 && worst > 0 && <span className="text-muted-foreground">/</span>}
          {worst > 0 && (
            <span style={{ color: tokenCssVar('outcome-loss') }}>
              {t('explorer.briefing.streak_losses', { n: worst })}
            </span>
          )}
        </span>
      }
    />
  )
}
