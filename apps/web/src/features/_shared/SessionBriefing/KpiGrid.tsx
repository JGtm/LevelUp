/**
 * KpiGrid — bande KPI du SessionBriefing.
 *
 * Cards permanentes : Frags / Morts / Assists / Précision / Vie (5). "Matchs
 * joués" et "Durée totale" sont rendues dans SquadVerdict (toujours monté) ;
 * le seul caller passe donc omitSummaryCards=true → baseCols=5. La branche
 * omitSummaryCards=false (baseCols=7, Matchs+Durée inclus) reste supportée
 * mais n'est plus empruntée.
 *
 * Cards conditionnelles :
 *  - Rendement/Résistance (avg_offensive_conversion / avg_defensive_resistance) :
 *    fusionnées en UNE card composite (barre + valeurs + dégâts/frag·mort) sur 2
 *    colonnes, via le composant partagé CombatYieldDisplay (même format que la
 *    Synthesis et la home). Si une seule des deux métriques est présente, fallback
 *    sur une KpiCell classique (span-1).
 *  - Delta rang : couleur ABSOLUE par SIGNE (divergent-pos/neg/neutral). Kind
 *    détermine le label (Delta CSR / Delta LUSR) et la précision (CSR=int,
 *    LUSR=2 décimales). Pas de glyphe trend (la couleur + le signe explicite
 *    sur la valeur suffisent).
 *
 * "Matchs joués" affiche le count + la durée moyenne par match en inline-sub
 * (ex: "12  8min07/match"). Cette présentation est répliquée dans
 * SquadVerdict en mode squad pour préserver l'info.
 *
 * Trends ▲/▼ vs teamAvgKpis (mode squad uniquement) sur les 5 cards
 * évaluatives historiques (frags, morts, assists, précision, vie). Morts =
 * lower_is_better. Si teamAvgKpis est null (mode solo) → trends 'none'.
 */
import type { CSSProperties } from 'react'

import type { KPIStats, RankDelta } from '@/lib/api/types'

import { formatDurationDhm, formatMinSec, formatMmss } from './format'
import type { BriefingTexts } from './i18n'
import { computeTrend, trendSymbol, type TrendState } from './trends'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { accuracyScale, assistsScale, kdScale, lifespanScale } from '@/lib/accessibility/scales'
import { CombatYieldDisplay } from '@/components/ui/combat-yield-display'
import { combatYieldToken } from '@/lib/formatters/combatYield'
import { formatDurationMShort, formatSignedFixed } from '@/lib/formatters'
import { KpiCard } from '@/components/cards/KpiCard'
import { useProvidesDamageTaken } from '@/lib/damage/effectiveHp'

interface KpiGridProps {
  kpis: KPIStats
  /** Référence pour calcul trends ▲/▼. Null = mode solo, pas de trends. */
  teamAvgKpis: KPIStats | null
  texts: BriefingTexts
  /** Titre de la grille — gestionnaire externe (drilled vs self) */
  title: string
  /** Hint affiché à droite du titre — affiché seulement si teamAvgKpis présent */
  hint?: string
  /** Si true, retire les cards "Matchs joués" et "Durée totale" de la grille
   *  car elles sont rendues ailleurs (typiquement SquadVerdict en mode squad). */
  omitSummaryCards?: boolean
}

interface CellProps {
  label: string
  value: string
  /** Sub-info affichée en dessous (ex: "1.20/min" sous frags par match). */
  sub?: string
  /** Sub-info affichée à droite de la valeur sur la même ligne, plus petite
   *  et muted. Utile pour combiner deux infos liées dans une seule colonne
   *  (ex: count matchs + durée moyenne par match). */
  inlineSub?: string
  trend?: TrendState
  /** Couleur absolue de la valeur. Si fourni, override la couleur trend. */
  valueColorToken?: SemanticToken
  /** Accent de QUALITÉ ABSOLUE (échelle métier, ex. accuracyScale) appliqué en
   *  solo quand il n'y a pas de tendance vs équipe — évalue la valeur dans
   *  l'absolu (30% précision → rouge, ≥55% → vert). */
  absoluteAccent?: SemanticToken
  /** Tooltip natif (attribut title) affiché au survol de la cellule. Utilisé pour
   *  lever l'ambiguïté d'une valeur MM:SS (ex. "1:05" → tooltip "1m05s"). */
  valueTitle?: string
}

function KpiCell({ label, value, sub, inlineSub, trend = 'none', valueColorToken, absoluteAccent, valueTitle }: CellProps) {
  const trendToken: SemanticToken | null =
    trend === 'above'
      ? 'divergent-pos'
      : trend === 'below'
        ? 'divergent-neg'
        : trend === 'near'
          ? 'divergent-neutral'
          : null

  // Accent dynamique (type 4 du catalogue) : couleur absolue du delta de rang si
  // fournie, sinon la tendance vs moyenne d'équipe (mode squad), sinon l'accent
  // de QUALITÉ ABSOLUE de la métrique si défini (ex. accuracyScale : 30% → rouge,
  // ≥55% → vert). Pas de barre uniquement si aucune de ces références n'existe.
  const accent: SemanticToken | undefined = valueColorToken ?? trendToken ?? absoluteAccent

  const valueStyle: CSSProperties | undefined = valueColorToken
    ? { color: tokenCssVar(valueColorToken) }
    : undefined

  return (
    <KpiCard accent={accent} className="h-full">
      <div className="px-3 py-2" title={valueTitle}>
        <p className="text-3xs uppercase tracking-wide text-muted-foreground">{label}</p>
        <div className="mt-0.5 flex items-baseline">
          <span className="text-lg font-bold" style={valueStyle}>{value}</span>
          {inlineSub && (
            <span className="ml-1.5 text-3xs text-muted-foreground">{inlineSub}</span>
          )}
          {!valueColorToken && trendToken && (
            <span
              className="ml-1 text-xs font-bold"
              style={{ color: tokenCssVar(trendToken) }}
            >
              {trendSymbol(trend)}
            </span>
          )}
        </div>
        {sub && <p className="mt-0.5 text-2xs text-muted-foreground">{sub}</p>}
      </div>
    </KpiCard>
  )
}

/** Formate la valeur signée d'un RankDelta. CSR = entier, LUSR = 2 décimales.
 *  Préfixe explicite (+/−/±) pour que le signe soit lisible sans dépendre
 *  uniquement de la couleur. */
function formatRankDeltaValue(delta: RankDelta): string {
  return formatSignedFixed(delta.value, delta.kind === 'csr' ? 0 : 2)
}

/** Token couleur absolue selon le signe du delta : pos/neg/neutral. */
function rankDeltaColorToken(value: number): SemanticToken {
  if (value > 0) return 'divergent-pos'
  if (value < 0) return 'divergent-neg'
  return 'divergent-neutral'
}

export function KpiGrid({ kpis, teamAvgKpis, texts, title, hint, omitSummaryCards = false }: KpiGridProps) {
  // false (Halo 5 : API sans damage_taken) → la Résistance n'est pas calculable.
  // On retire proprement la métrique DR de la grille (pas de cellule à 0, pas de
  // composite fondé sur un DR faux) ; le Rendement (OC) reste affiché seul.
  // Défaut true → Infinite : grille inchangée.
  const providesDamageTaken = useProvidesDamageTaken()
  const trendKills = teamAvgKpis
    ? computeTrend(kpis.kills_per_game, teamAvgKpis.kills_per_game)
    : 'none'
  const trendDeaths = teamAvgKpis
    ? computeTrend(kpis.deaths_per_game, teamAvgKpis.deaths_per_game, { lowerIsBetter: true })
    : 'none'
  const trendAssists = teamAvgKpis
    ? computeTrend(kpis.assists_per_game, teamAvgKpis.assists_per_game)
    : 'none'
  const trendAcc = teamAvgKpis
    ? computeTrend(kpis.avg_accuracy, teamAvgKpis.avg_accuracy)
    : 'none'
  const trendLife = teamAvgKpis
    ? computeTrend(kpis.avg_life_seconds, teamAvgKpis.avg_life_seconds)
    : 'none'

  // K/D dérivé → accent de qualité absolue des cartes Frags / Morts en solo
  // (≥1 bon, <1 moyen). Les rates par match sont mode-dépendants ; le ratio l'est
  // beaucoup moins, d'où le choix de kdScale plutôt qu'un seuil sur le compte brut.
  const kd = kpis.deaths_per_game > 0 ? kpis.kills_per_game / kpis.deaths_per_game : kpis.kills_per_game

  // Cards conditionnelles : rank_delta et rendement/résistance si données dispo.
  const hasDelta = kpis.rank_delta != null
  const hasOC = kpis.avg_offensive_conversion != null
  const hasDR = providesDamageTaken && kpis.avg_defensive_resistance != null
  // OC + DR fusionnés en UNE card composite (barre + 2 valeurs) sur 2 colonnes,
  // alignée sur le format de la home. Cas dégénéré (une seule des 2 métriques) :
  // fallback sur une KpiCell classique span-1, pour éviter d'afficher un côté faux.
  const hasComposite = hasOC && hasDR
  const hasSingleYield = hasOC !== hasDR
  const baseCols = omitSummaryCards ? 5 : 7
  const colCount =
    baseCols + (hasDelta ? 1 : 0) + (hasComposite ? 2 : 0) + (hasSingleYield ? 1 : 0)
  // Tailwind ne purge pas les `grid-cols-N` dynamiques → utiliser inline
  // grid-template-columns pour un colCount calculé.
  const gridStyle: CSSProperties = {
    gridTemplateColumns: `repeat(${colCount}, minmax(0, 1fr))`,
  }

  // Header row : affichée uniquement quand un titre existe (mode self).
  // En mode drilled (title vide), la reset bar "Vue active : X" indique déjà
  // le scope ; on évite l'empty line + on supprime le hint trend redondant
  // (les ▲/▼ colorés à côté de chaque value sont auto-explicites).
  return (
    <div>
      {title && (
        <div className="mb-1.5 flex items-center justify-between px-1">
          <span className="text-3xs font-semibold uppercase tracking-wider text-muted-foreground">
            {title}
          </span>
          {teamAvgKpis && hint && (
            <span className="text-2xs text-muted-foreground">{hint}</span>
          )}
        </div>
      )}
      <div className="grid gap-2" style={gridStyle}>
        {!omitSummaryCards && (
          <>
            <KpiCell
              label={texts.grid.matchesPlayed}
              value={String(kpis.matches_count)}
              inlineSub={`${formatMinSec(kpis.avg_match_seconds)}${texts.grid.perMatch}`}
            />
            <KpiCell
              label={texts.grid.totalDuration}
              value={formatDurationDhm(kpis.total_play_seconds)}
            />
          </>
        )}
        <KpiCell
          label={texts.grid.fragsPerMatch}
          value={kpis.kills_per_game.toFixed(2)}
          sub={`${kpis.kills_per_minute.toFixed(2)}${texts.grid.perMin}`}
          trend={trendKills}
          absoluteAccent={kdScale(kd)}
        />
        <KpiCell
          label={texts.grid.deathsPerMatch}
          value={kpis.deaths_per_game.toFixed(2)}
          sub={`${kpis.deaths_per_minute.toFixed(2)}${texts.grid.perMin}`}
          trend={trendDeaths}
          absoluteAccent={kdScale(kd)}
        />
        <KpiCell
          label={texts.grid.assistsPerMatch}
          value={kpis.assists_per_game.toFixed(2)}
          sub={`${kpis.assists_per_minute.toFixed(2)}${texts.grid.perMin}`}
          trend={trendAssists}
          absoluteAccent={assistsScale(kpis.assists_per_game)}
        />
        <KpiCell
          label={texts.grid.accuracy}
          value={`${kpis.avg_accuracy.toFixed(2)}%`}
          trend={trendAcc}
          absoluteAccent={accuracyScale(kpis.avg_accuracy)}
        />
        <KpiCell
          label={texts.grid.lifespan}
          value={formatMmss(kpis.avg_life_seconds)}
          valueTitle={formatDurationMShort(kpis.avg_life_seconds)}
          trend={trendLife}
          absoluteAccent={lifespanScale(kpis.avg_life_seconds)}
        />
        {hasComposite && (
          <KpiCard
            accent={combatYieldToken(kpis.avg_offensive_conversion, kpis.avg_defensive_resistance)}
            className="col-span-2 h-full"
          >
            <div className="px-3 py-2">
              <p className="text-3xs uppercase tracking-wide text-muted-foreground">
                {texts.grid.offDef}
              </p>
              <div className="mt-1">
                <CombatYieldDisplay
                  className="w-full"
                  offensiveConversion={kpis.avg_offensive_conversion}
                  defensiveResistance={kpis.avg_defensive_resistance}
                  dmgPerKill={kpis.combat_profile?.dmg_per_kill}
                  dmgPerDeath={kpis.combat_profile?.dmg_per_death}
                  align="start"
                />
              </div>
            </div>
          </KpiCard>
        )}
        {hasSingleYield && hasOC && (
          <KpiCell
            label={texts.grid.rendement}
            value={`${((kpis.avg_offensive_conversion ?? 0) * 100).toFixed(0)}%`}
            sub={texts.grid.refBaseline}
          />
        )}
        {hasSingleYield && hasDR && (
          <KpiCell
            label={texts.grid.resistance}
            value={`${(((kpis.avg_defensive_resistance ?? 1) - 1) * 100).toFixed(0)}%`}
            sub={texts.grid.refBaseline}
          />
        )}
        {hasDelta && (
          <KpiCell
            label={
              kpis.rank_delta!.kind === 'csr'
                ? texts.grid.rankDeltaCSR
                : texts.grid.rankDeltaLUSR
            }
            value={formatRankDeltaValue(kpis.rank_delta!)}
            valueColorToken={rankDeltaColorToken(kpis.rank_delta!.value)}
          />
        )}
      </div>
    </div>
  )
}
