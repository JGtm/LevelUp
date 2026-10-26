/**
 * ExplorerBriefingStrip — bandeau de briefing au-dessus du tableau (mode Matchs).
 *
 * Lecture compacte du RÉSULTAT DE RECHERCHE : socle de 5 à 8 tuiles KPI —
 * base (5) : Matchs, Taux de victoire (ruban V-D-N + tooltip), FDA et Perf en
 * triptyques « min · moyenne · max » (DP-1), Durée totale ; puis en cascade par
 * priorité, les 3 tiennent (DP-2) : Séries, Pic rang, Pic MMR — avec
 * deltas vs baseline personnelle.
 *
 * Disposition du socle : grille à colonnes ÉGALES. Toutes les tuiles sur une ligne quand
 * elles tiennent (150 px minimum chacune), sinon des lignes équilibrées aux colonnes
 * alignées (8 -> 4+4, 7 -> 4+3, 6 -> 3+3, 5 -> 3+2), plancher 2 colonnes. Le choix se fait
 * en CSS pur, par container queries sur la largeur du bandeau (`socleGridClass`) : aucune
 * lecture du DOM. Les modules « Par… » (dimensions, « Par contexte »,
 * Classement par chaîne), l'arme favorite et les moments forts sont rendus par
 * ExplorerBriefingModules sous le socle.
 *
 * Dégradation : briefing absent → rien. L'échantillon faible (low_sample) ne retire rien :
 * socle, écarts « vs habituel » (dès qu'une baseline est servie) et modules sont rendus
 * comme au-delà du seuil. Il est signalé seulement par une icône d'avertissement, avec
 * infobulle, sur la tuile Matchs. Aucune couleur hex : tokens sémantiques.
 */
import type { ReactNode } from 'react'

import { InfoTooltip } from '@/components/ui/info-tooltip'
import { NoticeIcon } from '@/components/ui/notice'
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
import { isFullHistoryScope } from '@/lib/baseline'
import { deltaToken, socleGridClass } from './ExplorerBriefing.logic'
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

// Wrapper de tuile du socle : la largeur vient de la grille (colonnes égales, voir
// socleGridClass), jamais de la tuile ; min-w-0 évite que les tabular-nums élargissent
// une colonne `minmax(0,1fr)`.
const SOCLE_TILE = 'min-w-0'

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

// Avertissement d'échantillon faible, posé à la place de l'icône (i) sur la tuile Matchs :
// même gabarit (InfoTooltip, 14 px), triangle teinté par le jeton `warning` de Notice. Le
// texte sert à la fois d'infobulle et de libellé accessible de l'icône.
function LowSampleWarning({ text }: { text: string }) {
  return (
    <InfoTooltip
      content={text}
      triggerFocusable
      trigger={
        <span role="img" aria-label={text} data-testid="briefing-low-sample" className="inline-flex">
          <NoticeIcon tone="warning" className="h-3.5 w-3.5" />
        </span>
      }
    />
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
  const lowSampleText = t('explorer.briefing.low_sample', { n: matchesCount })
  const peakRanks = scope?.peak_ranks ?? []
  const peakMmr = scope?.peak_team_mmr ?? null

  // Cascade des tuiles conditionnelles (DP-2) : collectées par PRIORITÉ décroissante
  // (Séries > Pic rang > Pic MMR) et TOUTES rendues si présentes → socle plafonné à 8
  // (5 base + 3).
  const conditionalTiles: ReactNode[] = []
  if (scope) {
    if (showStreaks && streaks != null) {
      conditionalTiles.push(
        <div key="streaks" className={SOCLE_TILE}>
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
        <div key="peak-mmr" className={SOCLE_TILE}>
          <PeakMmrTile value={peakMmr} t={t} />
        </div>,
      )
    }
  }

  // Tuiles effectivement rendues : Matchs, puis quatre tuiles de base quand le scope existe,
  // puis les conditionnelles. Le nombre de colonnes de la grille en découle.
  const socleTileCount = 1 + (scope ? 4 : 0) + conditionalTiles.length

  return (
    <div className="space-y-2">
      {/* En-tête discret : bascule réduire/afficher de la synthèse (choix mémorisé
          par explorerPrefsStore → localStorage). Aligné à droite pour ne pas
          concurrencer le socle de tuiles. whitespace-nowrap + flex-shrink-0 sur le
          bouton : le libellé "Afficher le résumé"/"Show briefing" (expand) est
          nettement plus long que "Réduire"/"Collapse" (collapse) — sans ces
          classes le texte peut passer sur 2 lignes sous contrainte de largeur, ce
          qui décentre verticalement le chevron par rapport au texte replié. */}
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
          className="inline-flex flex-shrink-0 items-center gap-1 whitespace-nowrap text-2xs font-medium text-muted-foreground hover:text-foreground transition-colors"
        >
          <span>
            {collapsed ? t('explorer.briefing.expand') : t('explorer.briefing.collapse')}
          </span>
          <BriefingToggleChevron collapsed={collapsed} />
        </button>
      </div>

      {!collapsed && (
        <>
          {/* @container : les paliers de la grille se mesurent sur la largeur du bandeau,
              pas sur celle de la fenêtre. */}
          <div className="@container">
          <div
            data-testid="briefing-socle"
            className={`grid gap-2 ${socleGridClass(socleTileCount)}`}
          >
            {/* Matchs + période */}
        <div className={SOCLE_TILE}>
          <BriefingTile
            label={t('explorer.briefing.matches_label')}
            info={lowSample ? <LowSampleWarning text={lowSampleText} /> : undefined}
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

        {/* Durée totale : tuile de base, aussi en échantillon faible (Pic FDA fusionné dans le triptyque FDA) */}
        {scope && (
          <div className={SOCLE_TILE}>
            <DurationTile seconds={scope.total_duration_seconds} t={t} />
          </div>
        )}

            {/* Conditionnelles en cascade (Séries > Pic rang > Pic MMR ; les 3 tiennent) */}
            {conditionalTiles}
          </div>
          </div>

          <ExplorerBriefingModules briefing={briefing} t={t} hideDelta={fullHistory} />
        </>
      )}
    </div>
  )
}
