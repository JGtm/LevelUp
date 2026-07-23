/**
 * ExplorerBriefingStrip — bandeau de briefing au-dessus du tableau (mode Matchs).
 *
 * Lecture compacte du RÉSULTAT DE RECHERCHE : rangée socle de 4 à 8 tuiles KPI —
 * base (5) : Matchs, Taux de victoire (ruban V-D-N + tooltip), FDA et Perf en
 * triptyques « min · moyenne · max » (DP-1), Durée totale ; puis en cascade par
 * priorité, les 3 tiennent (DP-2) : Séries marquantes, Pic rang, Pic MMR — avec
 * deltas vs baseline personnelle. Les modules « Par… » (dimensions, « Par contexte »,
 * Classement par chaîne) sont rendus par ExplorerBriefingModules sous le socle. En
 * low_sample : seules Matchs / Taux de victoire / FDA / Perf.
 *
 * Dégradation : briefing absent → rien ; low_sample → socle réduit + mention
 * échantillon faible, aucun module. Aucune couleur hex : tokens sémantiques.
 */
import type { ReactNode } from 'react'

import { InfoTooltip } from '@/components/ui/info-tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import { perfScale } from '@/lib/accessibility/scales'
import { kdaNetColor } from '@/lib/colors/outcomePalette'
import { getPerfColor } from '@/lib/perf-color'
import { formatDateRange, formatSignedFixed } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import { useExplorerPrefsStore } from '@/stores/explorerPrefsStore'
import type { ExplorerBriefing } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { BriefingTile } from './BriefingTile'
import { deltaToken, isFullHistoryScope } from './ExplorerBriefing.logic'
import { ExplorerBriefingModules } from './ExplorerBriefingModules'
import {
  DurationTile,
  MinMaxTriptych,
  PeakMmrTile,
  PeakRankTile,
  StreaksTile,
  WinRateTile,
} from './ExplorerBriefingTiles'

type T = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

// Largeurs socle (DP-4/DEC-WIDTHS) : chaque tuile est un flex-item. SOCLE_TILE réplique
// l'ancien minmax(150px,1fr) (largeur cible 150px, grandit pour remplir la ligne ; min-w-0
// évite l'overflow des tabular-nums). « Séries marquantes » ~+10 % (WIDE), « Pic MMR »
// ~−10 % (NARROW) ; flex-wrap + grow absorbent l'absence des conditionnelles sans trou.
// Valeurs basis/grow ajustables en revue visuelle (comme MIN_DECILE_SAMPLE).
const SOCLE_TILE = 'grow basis-[150px] min-w-0'
const SOCLE_TILE_WIDE = 'grow-[1.15] basis-[168px] min-w-0'
const SOCLE_TILE_NARROW = 'grow-[0.9] basis-[136px] min-w-0'

interface Props {
  briefing: ExplorerBriefing | null | undefined
  t: T
}

// Chevron de bascule (SVG inline, currentColor — aucune couleur en dur). Pointe
// vers le bas quand la synthèse est repliée (« dévoiler »), vers le haut sinon.
function BriefingToggleChevron({ collapsed }: { collapsed: boolean }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={collapsed ? 'm6 9 6 6 6-6' : 'm18 15-6-6-6 6'} />
    </svg>
  )
}

function formatPeriod(
  start: string | null | undefined,
  end: string | null | undefined,
  locale: string,
): string | undefined {
  if (!start) return undefined
  // Intervalle daté COMPLET (année incluse) via le helper canonique formatDateRange —
  // factorise mois/année (« 3–12 mars 2025 ») ; date simple si end absent/égal.
  return formatDateRange(start, end, locale === 'en' ? 'en-US' : 'fr-FR')
}

export function ExplorerBriefingStrip({ briefing, t }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const collapsed = useExplorerPrefsStore((s) => s.briefingCollapsed)
  const toggleCollapsed = useExplorerPrefsStore((s) => s.toggleBriefingCollapsed)
  if (!briefing) return null

  const scope = briefing.scope
  const baseline = briefing.baseline
  const period = formatPeriod(briefing.period_start, briefing.period_end, locale)
  const matchesCount = scope?.matches ?? 0

  const kda = scope?.kda ?? null
  const perf = scope?.avg_perf ?? null
  const vs = t('explorer.briefing.vs_baseline')
  // Plein historique (scope == baseline) → deltas « vs habituel » nuls par
  // construction : on masque le fragment (valeur + libellé) sur le socle ET les
  // lignes de dimension (P-1). Sous filtre, comportement V1 inchangé.
  const fullHistory = isFullHistoryScope(scope?.matches, baseline?.matches)
  // Meilleure série : conditionnelle omise si les deux segments sont à zéro (DP-3).
  const streaks = briefing.streaks ?? null
  const showStreaks =
    streaks != null && ((streaks.best_win_streak ?? 0) > 0 || (streaks.worst_loss_streak ?? 0) > 0)

  const lowSample = !!briefing.low_sample
  const peakRanks = scope?.peak_ranks ?? []
  const peakMmr = scope?.peak_team_mmr ?? null

  // Cascade des tuiles conditionnelles (DP-2) : collectées par PRIORITÉ décroissante
  // (Séries marquantes > Pic rang > Pic MMR) et TOUTES rendues si présentes → socle
  // plafonné à 8 (5 base hors low_sample + 3 ; le retrait du Pic FDA libère le slot,
  // Pic MMR redevient visible). Omises entièrement en low_sample.
  const conditionalTiles: ReactNode[] = []
  if (scope && !lowSample) {
    if (showStreaks && streaks != null) {
      conditionalTiles.push(
        <div key="streaks" className={SOCLE_TILE_WIDE}>
          <StreaksTile streaks={streaks} t={t} />
        </div>,
      )
    }
    if (peakRanks.length > 0) {
      conditionalTiles.push(
        <div key="peak-rank" className={SOCLE_TILE}>
          <PeakRankTile ranks={peakRanks} t={t} />
        </div>,
      )
    }
    if (peakMmr != null) {
      conditionalTiles.push(
        <div key="peak-mmr" className={SOCLE_TILE_NARROW}>
          <PeakMmrTile value={peakMmr} t={t} />
        </div>,
      )
    }
  }

  return (
    <div className="space-y-2">
      {/* En-tête discret : bascule réduire/afficher de la synthèse (choix mémorisé
          par explorerPrefsStore → localStorage). Aligné à droite pour ne pas
          concurrencer le socle de tuiles ; jamais en flex-wrap (le socle reste le
          premier flex-wrap du bandeau). */}
      <div className="flex justify-end">
        <button
          type="button"
          onClick={toggleCollapsed}
          aria-expanded={!collapsed}
          aria-label={
            collapsed
              ? t('explorer.briefing.expand_aria')
              : t('explorer.briefing.collapse_aria')
          }
          className="inline-flex items-center gap-1 text-2xs font-medium text-muted-foreground hover:text-foreground transition-colors"
        >
          <span>
            {collapsed ? t('explorer.briefing.expand') : t('explorer.briefing.collapse')}
          </span>
          <BriefingToggleChevron collapsed={collapsed} />
        </button>
      </div>

      {!collapsed && (
        <>
          <div className="flex flex-wrap gap-2">
            {/* Matchs + période */}
        <div className={SOCLE_TILE}>
          <BriefingTile
            label={t('explorer.briefing.matches_label')}
            value={String(matchesCount)}
            sub={period}
            accent="outcome-draw"
          />
        </div>

        {/* Taux de victoire (hero : ruban V-D-N + tooltip des 4 issues, DEC-TILES) */}
        {scope && (
          <div className={SOCLE_TILE}>
            <WinRateTile scope={scope} baseline={baseline} fullHistory={fullHistory} t={t} />
          </div>
        )}

        {/* FDA en triptyque min · moyenne · max (DP-1) + delta */}
        {scope && (
          <div className={SOCLE_TILE}>
            <BriefingTile
              label={t('explorer.briefing.fda_label')}
              info={<InfoTooltip content={t('explorer.briefing.tip_fda')} iconClass="w-3.5 h-3.5" />}
              value={
                <MinMaxTriptych
                  min={scope.min_kda}
                  mid={kda}
                  max={scope.peak_kda}
                  midColor={kda != null ? kdaNetColor(kda) : undefined}
                  format={(v) => v.toFixed(2)}
                />
              }
              sub={
                baseline && !fullHistory ? (
                  <>
                    <span
                      className="font-semibold"
                      style={{ color: tokenCssVar(deltaToken(baseline.delta_kda)) }}
                    >
                      {formatSignedFixed(baseline.delta_kda, 2)}
                    </span>{' '}
                    {vs}
                  </>
                ) : undefined
              }
              accent={deltaToken(kda)}
            />
          </div>
        )}

        {/* Perf en triptyque min · moyenne · max (DP-1) — moyenne colorée + accent perf (DP-6) + delta */}
        {scope && (
          <div className={SOCLE_TILE}>
            <BriefingTile
              label={t('explorer.briefing.perf_label')}
              info={<InfoTooltip content={t('explorer.briefing.tip_perf')} iconClass="w-3.5 h-3.5" />}
              accent={perf != null ? perfScale(perf) : 'outcome-draw'}
              value={
                <MinMaxTriptych
                  min={scope.min_perf}
                  mid={perf}
                  max={scope.max_perf}
                  midColor={perf != null ? getPerfColor(perf) : undefined}
                  format={(v) => v.toFixed(0)}
                />
              }
              sub={
                baseline && !fullHistory && baseline.delta_perf != null ? (
                  <>
                    <span
                      className="font-semibold"
                      style={{ color: tokenCssVar(deltaToken(baseline.delta_perf)) }}
                    >
                      {formatSignedFixed(baseline.delta_perf, 0)}
                    </span>{' '}
                    {vs}
                  </>
                ) : undefined
              }
            />
          </div>
        )}

        {/* Durée totale : tuile de base hors low_sample (Pic FDA fusionné dans le triptyque FDA) */}
        {scope && !lowSample && (
          <div className={SOCLE_TILE}>
            <DurationTile seconds={scope.total_duration_seconds} t={t} />
          </div>
        )}

            {/* Conditionnelles en cascade (Séries marquantes > Pic rang > Pic MMR ; les 3 tiennent) */}
            {conditionalTiles}
          </div>

          {/* Échantillon faible : socle seul + mention ; sinon modules conditionnels. */}
          {briefing.low_sample ? (
            <p className="text-2xs text-muted-foreground">
              {t('explorer.briefing.low_sample', { n: matchesCount })}
            </p>
          ) : (
            <ExplorerBriefingModules briefing={briefing} t={t} hideDelta={fullHistory} />
          )}
        </>
      )}
    </div>
  )
}
