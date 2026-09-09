/**
 * SessionUsageForms — LE RENDU DES FORMES PROPRES de la grammaire session-usage
 * (handoff §1) : « écart à la parité », « piste du lobby » et « bande de régularité ». Les grilles alignées, elles, passent par la
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
import { Fragment, useState, type CSSProperties } from 'react'

import { CollapsedItemsToggle } from '@/components/ui/collapsed-items-toggle'
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'

import type { UsageText } from './usageI18n'
import { usagePlayerInk } from './usageGrids'
import type {
  UsageBandCell,
  UsageGaugeModel,
  UsageGaugeOutcomeSegment,
  UsageGaugeRowModel,
  UsageTrackSegment,
} from './usageLogic'

/** Largeur de la colonne des libellés de grandeur (alignée sur ValueGrid). */
const LABEL_WIDTH = 152
/** Largeur mini d'une colonne de jauge, et gouttière (alignées sur ValueGrid). */
const GAUGE_MIN = 150
const COLUMN_GAP = 14

/** L'encre du trait de parité — jeton distinct, jamais une teinte de donnée. */
const PARITY_INK = tokenCssVar('warning')
/** L'encre de « nous » (le camp du joueur), surchargeable par l'accessibilité. */
const ALLY_INK = tokenCssVar('team-ally')
/**
 * LES DEUX REPÈRES DE TAUX DANS LA TRANCHE (P7, §3.2, étape E4) : « reste de mon
 * équipe » reprend le jeton `team-ally` (la référence EST mon camp) ; « eux » n'a
 * PAS de jeton de donnée — un trait pointillé neutre, comme la hachure de la piste
 * du lobby (règle du bloc usage : l'adversaire n'est jamais coloré).
 */
const TEAMMATES_REF_INK = tokenCssVar('team-ally')
const OPPONENTS_REF_INK = 'var(--muted-foreground)'

/** La hachure anonyme de « eux » : motif neutre du thème, jamais un jeton d'équipe. */
const ENEMY_HATCH: CSSProperties = {
  backgroundImage:
    'repeating-linear-gradient(45deg, transparent 0px, transparent 4px, var(--muted-foreground) 4px, var(--muted-foreground) 6px)',
  opacity: 0.45,
}

function clampPct(v: number): number {
  return Math.max(0, Math.min(100, v))
}

// ─── Écart à la parité ───────────────────────────────────────────────────────────

/**
 * L'INDEX DE LA JAUGE MONTRÉE SEULE quand le repli est fermé : « ma part dans mon
 * équipe » (2e de `buildGaugeRow`). C'est le seul des trois dénominateurs qui réponde à
 * la question que le lecteur se pose — est-ce que je porte mon équipe ou est-ce que je la
 * suis. Les deux autres rapportent la même mesure au lobby ; ils sont VRAIS mais
 * secondaires, et trois rails par ligne rendaient la grille illisible (D4).
 */
const PRIMARY_GAUGE_INDEX = 1

/**
 * UsageGauge — DEUX cellules de grille (rail, puis texte), jamais un flex local :
 * le rail doit avoir LA MÊME largeur sur toutes les lignes d'une colonne pour que
 * l'axe gradué du pied mesure vraiment les rails qu'il borde (revue adversariale
 * 2026-09-05 : un flex rail+texte donnait des rails raccourcis par leur propre
 * texte, et des graduations 50/100 qui ne tombaient sur rien).
 *
 * LE COMPTE BRUT N'EST PLUS ÉCRIT DANS LA CELLULE (D2) — il est dans l'infobulle du
 * rail, où il était déjà. « 49,3 % (105 sur 213) » sur trois lignes de trois colonnes
 * faisait neuf fractions à lire pour neuf pourcentages qui suffisaient.
 */
/**
 * UsageOutcomeStack — LE REMPLISSAGE DE LA TRANCHE (P1, P6, étape E4) : la pile
 * utilisé → lâché → gardé (ordre ratifié par le test E4.6), puis les DEUX REPÈRES
 * DE TAUX qui excluent le joueur (P7, §3.2) — des marques SANS CHIFFRE (E4.2),
 * positionnées EN POURCENTAGE DE LA TRANCHE elle-même (même dénominateur que les
 * segments), jamais du rail entier : le conteneur appelant est déjà large de
 * `valuePct`, donc `left: X%` ici tombe au bon endroit sans calcul composé.
 *
 * Décoratif (`aria-hidden`) : le texte qui compte vit dans le `tooltip` combiné du
 * rail entier (même contrat que le trait de parité, qui n'a jamais eu sa propre
 * aria-label).
 */
function UsageOutcomeStack({
  segments,
  teammatesRatePct,
  opponentsRatePct,
}: {
  segments: UsageGaugeOutcomeSegment[]
  teammatesRatePct: number | null
  opponentsRatePct: number | null
}) {
  return (
    <div className="relative flex h-full w-full" aria-hidden="true">
      {segments.map((seg) => (
        <div
          key={seg.key}
          data-outcome-key={seg.key}
          className="h-full"
          style={{ width: `${seg.fraction * 100}%`, backgroundColor: tokenCssVar(seg.token) }}
        />
      ))}
      {teammatesRatePct != null && (
        <div
          data-outcome-ref="teammates"
          className="absolute top-0 h-full w-[2px]"
          style={{ left: `${clampPct(teammatesRatePct)}%`, backgroundColor: TEAMMATES_REF_INK }}
        />
      )}
      {opponentsRatePct != null && (
        <div
          data-outcome-ref="opponents"
          className="absolute top-0 h-full border-l border-dashed"
          style={{ left: `${clampPct(opponentsRatePct)}%`, borderColor: OPPONENTS_REF_INK }}
        />
      )}
    </div>
  )
}

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
            <div className="absolute left-0 top-0 h-full" style={{ width: `${clampPct(gauge.valuePct)}%` }}>
              {gauge.segments != null ? (
                <UsageOutcomeStack
                  segments={gauge.segments}
                  teammatesRatePct={gauge.teammatesRatePct}
                  opponentsRatePct={gauge.opponentsRatePct}
                />
              ) : (
                <div data-outcome-fill="" className="h-full w-full" style={{ backgroundColor: ALLY_INK }} />
              )}
            </div>
          )}
          {gauge.parityPct != null && (
            <div
              className="absolute top-[-2px] h-[15px] w-[2px]"
              style={{ left: `calc(${clampPct(gauge.parityPct)}% - 1px)`, backgroundColor: PARITY_INK }}
            />
          )}
        </div>
      </Tooltip>
      <span className="whitespace-nowrap text-right text-3xs tabular-nums text-foreground">
        {gauge.valueText}
      </span>
    </>
  )
}

/** L'axe gradué 0 · 50 · 100 % d'une colonne de jauges. */
function GaugeAxis() {
  return (
    <div className="relative mt-1 h-[15px] border-t border-border text-3xs text-muted-foreground tabular-nums">
      <span className="absolute left-0 top-0.5">0</span>
      <span className="absolute left-1/2 top-0.5 -translate-x-1/2">50</span>
      <span className="absolute right-0 top-0.5">100 %</span>
    </div>
  )
}

/**
 * UsageGaugeGrid — les jauges de parts, UNE colonne par défaut (« ma part dans mon
 * équipe ») et les deux autres dénominateurs derrière un repli, une ligne par grandeur,
 * un axe gradué par colonne rendue.
 *
 * L'ÉTAT DU REPLI N'EST PAS PERSISTÉ et vit ici, pas chez l'appelant : les trois cartes
 * du bloc ont chacune leur grille, et une préférence partagée ferait s'ouvrir la carte
 * des objectifs parce qu'on a déplié celle de l'équipement (même décision que le repli
 * « game changers » de la vue match, D3 de son plan).
 */
export function UsageGaugeGrid({ rows, t }: { rows: UsageGaugeRowModel[]; t: UsageText }) {
  const [expanded, setExpanded] = useState(false)
  if (rows.length === 0) return null

  const allHeaders = [t.gaugeTeamOfLobby, t.gaugePlayerOfTeam, t.gaugePlayerOfLobby]
  // Les index de jauge rendus. Une ligne qui n'aurait pas la jauge primaire (contrat
  // partiel) retombe sur la première : mieux vaut la mauvaise colonne que rien.
  const total = rows[0].gauges.length
  const primary = PRIMARY_GAUGE_INDEX < total ? PRIMARY_GAUGE_INDEX : 0
  const shown = expanded ? rows[0].gauges.map((_, i) => i) : [primary]
  const hidden = total - shown.length

  // Chaque colonne de jauge = DEUX sous-colonnes : le rail (élastique, borné) puis le
  // texte (à la largeur du plus long de la colonne). Ainsi tous les rails d'une colonne
  // sont de même largeur, et l'axe gradué du pied (posé dans la sous-colonne rail
  // seulement) mesure exactement ce qu'il borde.
  const gridStyle: CSSProperties = {
    gridTemplateColumns: `${LABEL_WIDTH}px repeat(${shown.length}, minmax(${GAUGE_MIN}px, 1fr) max-content)`,
    minWidth: LABEL_WIDTH + shown.length * (GAUGE_MIN + COLUMN_GAP),
    columnGap: COLUMN_GAP,
  }
  return (
    <div className="overflow-x-auto">
      <div className="grid items-center gap-y-[6px]" style={gridStyle}>
        <div aria-hidden="true" />
        {shown.map((gaugeIndex, i) => (
          <div
            key={allHeaders[gaugeIndex] ?? gaugeIndex}
            className="mb-1 flex items-center justify-between gap-2 whitespace-nowrap border-b border-border pb-1.5 text-3xs font-semibold uppercase tracking-wider"
            style={{ gridColumn: 'span 2' }}
          >
            <span>{allHeaders[gaugeIndex]}</span>
            {/* Le bouton vit dans le DERNIER en-tête rendu : à droite de la seule
                colonne quand c'est replié, à droite de la dernière quand c'est ouvert. */}
            {i === shown.length - 1 && (
              <CollapsedItemsToggle
                count={expanded ? total - 1 : hidden}
                expanded={expanded}
                onToggle={() => setExpanded((v) => !v)}
                showLabelFmt={t.sharesShowMoreFmt}
                hideLabel={t.sharesHide}
                hint={t.sharesHint}
              />
            )}
          </div>
        ))}
        {rows.map((row, rowIndex) => (
          <Fragment key={row.key}>
            {/* Un TOTAL se sépare de ses propres composantes par un filet pleine largeur
                (`1 / -1`), jamais par une bordure sur la seule cellule de libellé : le
                filet doit traverser les rails, sinon la ligne se lit comme une grandeur
                de plus. Filet omis en tête de grille — il n'y aurait rien au-dessus. */}
            {row.isTotal && rowIndex > 0 && (
              <div className="mt-0.5 border-t border-border pt-0.5" style={{ gridColumn: '1 / -1' }} />
            )}
            <div
              className={`overflow-hidden whitespace-nowrap text-xs${row.isTotal ? ' text-muted-foreground' : ''}`}
              title={row.label}
            >
              <span className="truncate">{row.label}</span>
            </div>
            {shown.map((gaugeIndex) => (
              <UsageGauge key={row.gauges[gaugeIndex].key} gauge={row.gauges[gaugeIndex]} />
            ))}
          </Fragment>
        ))}
        <div aria-hidden="true" />
        {shown.map((gaugeIndex) => (
          <Fragment key={`axis-${gaugeIndex}`}>
            <GaugeAxis />
            <div aria-hidden="true" />
          </Fragment>
        ))}
      </div>
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
