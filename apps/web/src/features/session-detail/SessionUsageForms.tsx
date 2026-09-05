/**
 * SessionUsageForms — LE RENDU DES FORMES PROPRES de la grammaire session-usage
 * (handoff §1) : « écart à la parité / jauge double avec étendue », « piste du
 * lobby » et « bande de régularité ». Les grilles alignées, elles, passent par la
 * primitive partagée `components/charts/ValueGrid`.
 *
 * DOM ET CSS, PAS ECHARTS — le même choix mesuré que `ValueGrid` : ces formes sont
 * des problèmes de MISE EN PAGE (alignement de rails, segments en pourcentage,
 * cases en ligne), sans zoom ni animation. Chaque forme porte son AXE GRADUÉ avec
 * son intitulé (doctrine §1), défile horizontalement dans SON conteneur, et chaque
 * marque est focusable avec son texte en `aria-label` (même contrat d'accessibilité
 * que ValueGrid).
 *
 * COULEURS — jetons sémantiques uniquement :
 *   - « nous » = `team-ally` ; les joueurs suivis = `squad-player-*` via la source
 *     unique `features/squad/colors.ts` (1 = joueur principal, 2..4 = coéquipiers,
 *     convention de TOUTE l'app — la respecter ici évite qu'un joueur change de
 *     couleur entre la page Escouade et la page Sessions) ;
 *   - le trait de PARITÉ = `warning` — un jeton DISTINCT, jamais une teinte de
 *     donnée (doctrine §1) ;
 *   - « eux » = HACHURE NEUTRE, ni nom ni couleur d'équipe : la référence n'est
 *     jamais affichée, elle n'existe que comme dénominateur (doctrine §1) ;
 *   - la bande de régularité = gamme `divergent-*` (au-dessus / à / sous la parité).
 *
 * Aucun calcul ici : tout vient de `usageLogic.ts`.
 */
import { Fragment, type CSSProperties } from 'react'

import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'

import type { UsageText } from './usageI18n'
import { usagePlayerInk } from './usageGrids'
import type { UsageBandCell, UsageGaugeModel, UsageGaugeRowModel, UsageTrackSegment } from './usageLogic'

/** Largeur de la colonne des libellés de grandeur (alignée sur ValueGrid). */
const LABEL_WIDTH = 152
/** Largeur mini d'une colonne de jauge, et gouttière (alignées sur ValueGrid). */
const GAUGE_MIN = 150
const COLUMN_GAP = 14

/** L'encre du trait de parité — jeton distinct, jamais une teinte de donnée. */
const PARITY_INK = tokenCssVar('warning')
/** L'encre de « nous » (le camp du joueur), surchargeable par l'accessibilité. */
const ALLY_INK = tokenCssVar('team-ally')

/** La hachure anonyme de « eux » : motif neutre du thème, jamais un jeton d'équipe. */
const ENEMY_HATCH: CSSProperties = {
  backgroundImage:
    'repeating-linear-gradient(45deg, transparent 0px, transparent 4px, var(--muted-foreground) 4px, var(--muted-foreground) 6px)',
  opacity: 0.45,
}

function clampPct(v: number): number {
  return Math.max(0, Math.min(100, v))
}

// ─── Écart à la parité / jauge double avec étendue ───────────────────────────────

/**
 * UsageGauge — DEUX cellules de grille (rail, puis texte), jamais un flex local :
 * le rail doit avoir LA MÊME largeur sur toutes les lignes d'une colonne pour que
 * l'axe gradué du pied mesure vraiment les rails qu'il borde (revue adversariale
 * 2026-09-05 : un flex rail+texte donnait des rails raccourcis par leur propre
 * texte, et des graduations 50/100 qui ne tombaient sur rien).
 */
function UsageGauge({ gauge }: { gauge: UsageGaugeModel }) {
  return (
    <>
      <Tooltip content={gauge.tooltip} className="w-full">
        <div
          className="relative h-[11px] w-full min-w-[60px] bg-muted"
          tabIndex={0}
          role="img"
          aria-label={gauge.tooltip}
        >
          {gauge.valuePct != null && (
            <div
              className="absolute left-0 top-0 h-full"
              style={{ width: `${clampPct(gauge.valuePct)}%`, backgroundColor: ALLY_INK }}
            />
          )}
          {gauge.rangeMinPct != null && gauge.rangeMaxPct != null && (
            <div
              className="absolute top-[-4px] h-[2px]"
              style={{
                left: `${clampPct(gauge.rangeMinPct)}%`,
                width: `${clampPct(gauge.rangeMaxPct) - clampPct(gauge.rangeMinPct)}%`,
                backgroundColor: ALLY_INK,
                opacity: 0.55,
              }}
            />
          )}
          {gauge.parityPct != null && (
            <div
              className="absolute top-[-2px] h-[15px] w-[2px]"
              style={{ left: `calc(${clampPct(gauge.parityPct)}% - 1px)`, backgroundColor: PARITY_INK }}
            />
          )}
        </div>
      </Tooltip>
      <span className="whitespace-nowrap text-right text-3xs tabular-nums">
        <span className="text-foreground">{gauge.valueText}</span>{' '}
        <span className="text-muted-foreground">({gauge.honestyText})</span>
      </span>
    </>
  )
}

/** L'axe gradué 0 · 50 · 100 % d'une colonne de jauges, avec son intitulé. */
function GaugeAxis({ title }: { title: string }) {
  return (
    <div className="relative mt-1 h-[15px] border-t border-border text-3xs text-muted-foreground tabular-nums">
      <span className="absolute left-0 top-0.5">0</span>
      <span className="absolute left-1/2 top-0.5 -translate-x-1/2">50</span>
      <span className="absolute right-0 top-0.5">{`100 — ${title}`}</span>
    </div>
  )
}

/**
 * UsageGaugeGrid — les jauges de parts alignées en trois colonnes (les trois
 * dénominateurs du §7), une ligne par grandeur, un axe gradué par colonne.
 */
export function UsageGaugeGrid({ rows, t }: { rows: UsageGaugeRowModel[]; t: UsageText }) {
  if (rows.length === 0) return null
  const gaugeCount = rows[0].gauges.length
  // Chaque colonne de jauge = DEUX sous-colonnes : le rail (élastique, borné) puis le
  // texte (à la largeur du plus long de la colonne). Ainsi tous les rails d'une colonne
  // sont de même largeur, et l'axe gradué du pied (posé dans la sous-colonne rail
  // seulement) mesure exactement ce qu'il borde.
  const gridStyle: CSSProperties = {
    gridTemplateColumns: `${LABEL_WIDTH}px repeat(${gaugeCount}, minmax(${GAUGE_MIN}px, 1fr) max-content)`,
    minWidth: LABEL_WIDTH + gaugeCount * (GAUGE_MIN + COLUMN_GAP),
    columnGap: COLUMN_GAP,
  }
  const headers = [t.gaugeTeamOfLobby, t.gaugePlayerOfTeam, t.gaugePlayerOfLobby]
  return (
    <div className="overflow-x-auto">
      <div className="grid items-center gap-y-[6px]" style={gridStyle}>
        <div aria-hidden="true" />
        {headers.slice(0, gaugeCount).map((h) => (
          <div
            key={h}
            className="mb-1 whitespace-nowrap border-b border-border pb-1.5 text-3xs font-semibold uppercase tracking-wider"
            style={{ gridColumn: 'span 2' }}
          >
            {h}
          </div>
        ))}
        {rows.map((row) => (
          <Fragment key={row.key}>
            <div className="overflow-hidden whitespace-nowrap text-xs" title={row.label}>
              <span className="truncate">{row.label}</span>
            </div>
            {row.gauges.map((g) => (
              <UsageGauge key={g.key} gauge={g} />
            ))}
          </Fragment>
        ))}
        <div aria-hidden="true" />
        {headers.slice(0, gaugeCount).map((h) => (
          <Fragment key={h}>
            <GaugeAxis title={t.axisSharePct} />
            <div aria-hidden="true" />
          </Fragment>
        ))}
      </div>
      <p className="pt-1.5 text-[11px] text-muted-foreground">{t.parityLegend}</p>
    </div>
  )
}

// ─── Piste du lobby ──────────────────────────────────────────────────────────────

/** L'encre d'un segment de piste, selon sa nature (voir l'en-tête du fichier). */
function trackSegmentStyle(seg: UsageTrackSegment): CSSProperties {
  switch (seg.kind) {
    case 'me':
      return { backgroundColor: usagePlayerInk('me') }
    case 'squad':
      return { backgroundColor: usagePlayerInk('squad', seg.squadIndex ?? 0) }
    case 'team-rest':
      return { backgroundColor: ALLY_INK, opacity: 0.5 }
    case 'enemy':
      return ENEMY_HATCH
  }
}

/**
 * UsageLobbyTrack — la piste 100 % du lobby : coloré = nous, découpé par joueur ;
 * hachuré = eux, anonyme. Le compte brut et la part sont écrits dans les segments
 * assez larges, et toujours dans l'infobulle.
 */
export function UsageLobbyTrack({ segments, label }: { segments: UsageTrackSegment[]; label: string }) {
  const total = segments.reduce((a, s) => a + s.count, 0)
  return (
    <div className="overflow-x-auto">
      <div className="min-w-[420px]">
        <div className="flex h-[22px]" role="img" aria-label={label}>
          {/* La largeur est portée par l'ITEM du flex, en `calc(%)` — jamais un flexGrow
              sur le contenu d'un Tooltip : le wrapper du Tooltip garde flex-grow 0 et les
              segments se dimensionneraient à leur texte, pas à leurs comptes (revue
              adversariale 2026-09-05 ; pattern correct : MatchPadControlSection). */}
          {segments.map((seg) => (
            <div
              key={seg.key}
              className="mr-[2px] h-full last:mr-0"
              style={{ width: total > 0 ? `calc(${(seg.count / total) * 100}% - 2px)` : '0%' }}
            >
              <Tooltip content={seg.tooltip} className="h-full w-full">
                {/* `text-white` : le libellé est posé SUR l'aplat du segment, quelle que
                    soit la palette réglée — le contraste d'un texte dans un aplat, pas
                    une couleur sémantique (même usage que UsageTeamShares, vue match). */}
                <div
                  className="flex h-full w-full items-center justify-center overflow-hidden whitespace-nowrap bg-muted px-1 text-3xs font-semibold text-white"
                  style={trackSegmentStyle(seg)}
                  tabIndex={0}
                  role="img"
                  aria-label={seg.tooltip}
                >
                  {/* Le libellé n'est écrit que si le segment pèse assez pour le porter. */}
                  {total > 0 && seg.count / total >= 0.12 ? `${seg.count} · ${seg.pctText}` : ''}
                </div>
              </Tooltip>
            </div>
          ))}
        </div>
        <div className="relative mt-1 h-[15px] border-t border-border text-3xs text-muted-foreground tabular-nums">
          <span className="absolute left-0 top-0.5">0</span>
          <span className="absolute left-1/2 top-0.5 -translate-x-1/2">50</span>
          <span className="absolute right-0 top-0.5">100 %</span>
        </div>
      </div>
    </div>
  )
}

// ─── Bande de régularité ─────────────────────────────────────────────────────────

/** L'encre d'une case : au-dessus / à / sous la parité, ou non mesurée. */
function bandCellStyle(cell: UsageBandCell): CSSProperties | undefined {
  switch (cell.tone) {
    case 'above':
      return { backgroundColor: tokenCssVar('divergent-pos') }
    case 'near':
      return { backgroundColor: tokenCssVar('divergent-neutral') }
    case 'below':
      return { backgroundColor: tokenCssVar('divergent-neg') }
    case 'unmeasured':
      return undefined // reste sur le fond `bg-muted` : non mesuré, pas une donnée
  }
}

/**
 * UsageRegularityBand — une case par match mesuré, dans l'ordre de la session,
 * teintée par l'écart à la parité. La légende de comptage (« matchs au-dessus de la
 * parité ») vient de l'appelant : elle est calculée côté Go contre la parité de
 * chaque match.
 */
export function UsageRegularityBand({
  label,
  cells,
  caption,
}: {
  label: string
  cells: UsageBandCell[]
  caption: string | null
}) {
  if (cells.length === 0) return null
  return (
    <div className="grid grid-cols-[152px_1fr] items-center gap-x-3.5 gap-y-0.5">
      <div className="overflow-hidden whitespace-nowrap text-xs" title={label}>
        <span className="truncate">{label}</span>
      </div>
      <div className="flex flex-wrap items-center gap-[3px]">
        {cells.map((cell) => (
          <Tooltip key={cell.matchId} content={cell.tooltip}>
            <span
              className="h-3.5 w-3.5 flex-none bg-muted"
              style={bandCellStyle(cell)}
              tabIndex={0}
              role="img"
              aria-label={cell.tooltip}
            />
          </Tooltip>
        ))}
        {caption != null && (
          <span className="pl-2 text-[11px] text-muted-foreground">{caption}</span>
        )}
      </div>
    </div>
  )
}
