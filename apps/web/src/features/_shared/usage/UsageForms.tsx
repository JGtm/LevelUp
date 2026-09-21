/**
 * UsageForms — LE RENDU DE LA JAUGE « écart à la parité » et de sa grille, forme propre de
 * la grammaire session-usage (handoff §1). Les grilles alignées, elles, passent par la
 * primitive partagée `components/charts/ValueGrid`.
 *
 * Déménagé de `session-detail/SessionUsageForms.tsx` vers ici le 2026-09-09 (étape E5.1,
 * PLAN_EQUIPEMENT_GACHIS_2026-09-09.md) : le bloc devient importable par `features/synthesis`
 * et `features/squad` sans violer `lint-cross-feature-imports`.
 *
 * SCINDÉ LE 2026-09-21 (seuil de taille, CLAUDE.md n°5) : la piste du lobby est partie dans
 * `UsageLobbyTrack.tsx`, la bande de régularité dans `UsageRegularityBand.tsx`, les encres et
 * les textures dans `usageInks.ts`. Trois formes indépendantes qui ne partageaient que ce
 * fichier.
 *
 * DOM ET CSS, PAS ECHARTS — le même choix mesuré que `ValueGrid` : ces formes sont des
 * problèmes de MISE EN PAGE (alignement de rails, segments en pourcentage), sans zoom ni
 * animation. Chaque forme porte son AXE GRADUÉ (doctrine §1) et chaque marque est focusable
 * avec son texte en `aria-label` (même contrat d'accessibilité que ValueGrid).
 *
 * COULEURS — jetons sémantiques uniquement, tous dans `usageInks.ts` : « nous » = `team-ally`,
 * le trait de PARITÉ = `warning` (jeton distinct, jamais une teinte de donnée), « eux » =
 * hachure neutre (la référence n'est jamais colorée — doctrine §1).
 *
 * Aucun calcul ici : tout vient de `usageGaugeModel.ts`.
 */
import { Fragment, useId, useState, type CSSProperties } from 'react'

import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'

import { UsageHatchLegend } from './UsageHatchLegend'
import type { UsageGaugeModel, UsageGaugeOutcomeSegment, UsageGaugeRowModel } from './usageGaugeModel'
import type { UsageText } from './usageI18n'
import {
  ALLY_INK,
  LOBBY_HATCH,
  OPPONENTS_REF_INK,
  PARITY_INK,
  TEAMMATES_REF_INK,
} from './usageInks'

/**
 * Largeur de la colonne des libellés de grandeur (alignée sur ValueGrid).
 * EXPORTÉE depuis le 2026-09-09 (E5.8) : `UsageCountsGrid.tsx` (variante comptes,
 * Synthèse/Escouade) aligne sa propre grille sur ces mêmes constantes plutôt que de
 * les redéfinir (CLAUDE.md n°6).
 */
export const LABEL_WIDTH = 152
/**
 * Gouttière entre colonnes (alignée sur ValueGrid).
 *
 * LA LARGEUR MINI D'UNE COLONNE DE JAUGE A DISPARU le 2026-09-19 : elle forçait la grille à
 * déborder de sa carte, et le débordement se réglait par un scroll horizontal qui cachait la
 * moitié des lignes sans le dire. Les rails partent désormais de zéro (`minmax(0, 1fr)`).
 */
export const COLUMN_GAP = 14

/**
 * Les mêmes largeurs en COLONNE DIVISÉE (drawer de comparaison, `dense`). Depuis le
 * 2026-09-13 le compact ne RETIRE plus rien — il resserre : les trois colonnes de jauges
 * tiennent dans une demi-largeur au lieu d'être remplacées par une seule.
 */
const DENSE_LABEL_WIDTH = 104
const DENSE_COLUMN_GAP = 8

function clampPct(v: number): number {
  return Math.max(0, Math.min(100, v))
}

// ─── Écart à la parité ───────────────────────────────────────────────────────────

/**
 * L'INDEX DE LA JAUGE RAPPORTÉE À MON ÉQUIPE (« ma part dans mon équipe », 2e de
 * `buildGaugeRow`). Les deux autres colonnes rapportent la mesure AU LOBBY : elles portent
 * la hachure (cf. `LOBBY_HATCH`).
 *
 * LES TROIS COLONNES SONT TOUJOURS RENDUES depuis le 2026-09-13 (demande utilisateur) : le
 * repli qui n'en montrait qu'une a été retiré, ici comme dans le drawer de comparaison — une
 * colonne cachée est une colonne qu'on ne lit jamais.
 */
const TEAM_GAUGE_INDEX = 1

/**
 * UsageOutcomeStack — LE REMPLISSAGE DE LA TRANCHE (P1, P6, étape E4) : la pile utilisé →
 * lâché → gardé (ordre ratifié par le test E4.6), puis les DEUX REPÈRES DE TAUX qui excluent
 * le joueur (P7, §3.2) — des marques SANS CHIFFRE (E4.2), positionnées EN POURCENTAGE DE LA
 * TRANCHE elle-même (même dénominateur que les segments), jamais du rail entier : le
 * conteneur appelant est déjà large de `valuePct`, donc `left: X%` ici tombe au bon endroit
 * sans calcul composé.
 *
 * Décoratif (`aria-hidden`) : le texte qui compte vit dans le `tooltip` combiné du rail
 * entier (même contrat que le trait de parité, qui n'a jamais eu sa propre aria-label).
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

/**
 * UsageGauge — DEUX cellules de grille (rail, puis texte), jamais un flex local : le rail
 * doit avoir LA MÊME largeur sur toutes les lignes d'une colonne pour que l'axe gradué du
 * pied mesure vraiment les rails qu'il borde (revue adversariale 2026-09-05 : un flex
 * rail+texte donnait des rails raccourcis par leur propre texte, et des graduations 50/100
 * qui ne tombaient sur rien).
 *
 * LE COMPTE BRUT N'EST PLUS ÉCRIT DANS LA CELLULE (D2) — il est dans l'infobulle du rail.
 *
 * EXPORTÉE depuis le 2026-09-09 (E5.8) : `UsageCountsGrid.tsx` (variante comptes,
 * Synthèse/Escouade, P9) réutilise cette MÊME cellule — seul `gauge.valuePct` change de sens
 * et `gauge.parityPct` y reste toujours `null`. Aucune seconde copie (CLAUDE.md n°6).
 */
export function UsageGauge({
  gauge,
  denominator = 'team',
}: {
  gauge: UsageGaugeModel
  /**
   * Ce que la jauge rapporte : mon équipe (aplat) ou LE LOBBY (aplat + hachure neutre, cf.
   * `LOBBY_HATCH`). Défaut « team » — la variante comptes (`UsageCountsGrid`) n'a qu'un
   * dénominateur et rend donc exactement comme avant.
   */
  denominator?: 'team' | 'lobby'
}) {
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
              {denominator === 'lobby' && (
                <div
                  data-gauge-denominator="lobby"
                  className="absolute inset-0"
                  style={LOBBY_HATCH}
                  aria-hidden="true"
                />
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
 * LE BOUTON D'UN DÉPLIABLE DE LIGNES, posé EN PLEINE LARGEUR de la grille (`1 / -1`).
 *
 * EXPORTÉ : `UsageCountsGrid` replie exactement les mêmes lignes (les armes de base, D2)
 * dans sa propre grille — un second bouton recopié aurait divergé au premier ajustement.
 */
export function UsageCollapseToggle({
  label,
  open,
  onToggle,
  controls,
}: {
  label: string
  open: boolean
  onToggle: () => void
  controls: string
}) {
  return (
    <div style={{ gridColumn: '1 / -1' }} className="pt-0.5">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        aria-controls={controls}
        className="inline-flex items-center gap-1 text-3xs font-medium text-muted-foreground hover:text-foreground"
      >
        <span aria-hidden="true">{open ? '▾' : '▸'}</span>
        {label}
      </button>
    </div>
  )
}

/** Les lignes d'une grille de jauges — corps partagé entre lignes visibles et repliées. */
function GaugeRows({
  rows,
  shown,
  startsGrid,
}: {
  rows: UsageGaugeRowModel[]
  shown: number[]
  /** Vrai quand ces lignes ouvrent la grille : un total n'y porte pas de filet de séparation. */
  startsGrid: boolean
}) {
  return (
    <>
      {rows.map((row, rowIndex) => (
        <Fragment key={row.key}>
          {/* Un TOTAL se sépare de ses propres composantes par un filet pleine largeur
              (`1 / -1`), jamais par une bordure sur la seule cellule de libellé : le filet
              doit traverser les rails, sinon la ligne se lit comme une grandeur de plus.
              Filet omis en tête de grille — il n'y aurait rien au-dessus. */}
          {row.isTotal && !(startsGrid && rowIndex === 0) && (
            <div className="mt-0.5 border-t border-border pt-0.5" style={{ gridColumn: '1 / -1' }} />
          )}
          <div
            className={`overflow-hidden whitespace-nowrap text-xs${row.isTotal ? ' text-muted-foreground' : ''}`}
            title={row.hint ?? row.label}
          >
            <span className="truncate">{row.label}</span>
          </div>
          {shown.map((gaugeIndex) => (
            <UsageGauge
              key={row.gauges[gaugeIndex].key}
              gauge={row.gauges[gaugeIndex]}
              denominator={gaugeIndex === TEAM_GAUGE_INDEX ? 'team' : 'lobby'}
            />
          ))}
        </Fragment>
      ))}
    </>
  )
}

/**
 * UsageGaugeGrid — les jauges de parts : LES TROIS DÉNOMINATEURS, toujours rendus, une ligne
 * par grandeur, un axe gradué par colonne, et SA LÉGENDE DE TEXTURE (D7, 2026-09-21).
 *
 * PLUS DE REPLI DE COLONNES (demande utilisateur du 2026-09-13) : les trois sont à l'écran,
 * en pleine largeur comme en colonne divisée — `dense` ne change que les largeurs. Le seul
 * repli qui subsiste est celui de LIGNES nommées par l'appelant (`collapsedRows`, D2 : les
 * armes de base), FERMÉ par défaut.
 */
export function UsageGaugeGrid({
  rows,
  t,
  dense = false,
  collapsedRows,
  collapsedLabel,
}: {
  rows: UsageGaugeRowModel[]
  t: UsageText
  /** Colonne divisée : mêmes colonnes, rails et libellés plus étroits. */
  dense?: boolean
  /** Lignes repliées derrière un bouton, fermé par défaut (D2). */
  collapsedRows?: UsageGaugeRowModel[]
  collapsedLabel?: string
}) {
  const [open, setOpen] = useState(false)
  const groupId = useId()
  const collapsed = collapsedRows ?? []
  if (rows.length === 0 && collapsed.length === 0) return null

  const allHeaders = [t.gaugeTeamOfLobby, t.gaugePlayerOfTeam, t.gaugePlayerOfLobby]
  const gaugeCount = (rows[0] ?? collapsed[0]).gauges.length
  const shown = Array.from({ length: gaugeCount }, (_, i) => i)
  const labelWidth = dense ? DENSE_LABEL_WIDTH : LABEL_WIDTH
  const columnGap = dense ? DENSE_COLUMN_GAP : COLUMN_GAP
  const hasCollapsed = collapsed.length > 0 && collapsedLabel != null

  // Chaque colonne de jauge = DEUX sous-colonnes : le rail (élastique, borné) puis le texte
  // (à la largeur du plus long de la colonne). Ainsi tous les rails d'une colonne sont de
  // même largeur, et l'axe gradué du pied (posé dans la sous-colonne rail seulement) mesure
  // exactement ce qu'il borde.
  const gridStyle: CSSProperties = {
    gridTemplateColumns: `${labelWidth}px repeat(${shown.length}, minmax(0, 1fr) max-content)`,
    columnGap,
  }
  return (
    <div className="min-w-0">
      <div className="grid items-center gap-y-[6px]" style={gridStyle} id={groupId}>
        <div aria-hidden="true" />
        {shown.map((gaugeIndex) => (
          <div
            key={allHeaders[gaugeIndex] ?? gaugeIndex}
            // `overflow-hidden` + retour à la ligne en dense : à demi-largeur, un en-tête
            // `nowrap` débordait sur celui de la colonne suivante et les trois titres se
            // lisaient collés (capture du 2026-09-13).
            className={`mb-1 flex items-center justify-between gap-2 overflow-hidden border-b border-border pb-1.5 text-3xs font-semibold uppercase tracking-wider ${
              dense ? 'whitespace-normal leading-tight' : 'whitespace-nowrap'
            }`}
            style={{ gridColumn: 'span 2' }}
          >
            <span title={allHeaders[gaugeIndex]}>{allHeaders[gaugeIndex]}</span>
          </div>
        ))}
        <GaugeRows rows={rows} shown={shown} startsGrid />
        {hasCollapsed && (
          <>
            <UsageCollapseToggle
              label={collapsedLabel}
              open={open}
              onToggle={() => setOpen((v) => !v)}
              controls={groupId}
            />
            {open && <GaugeRows rows={collapsed} shown={shown} startsGrid={false} />}
          </>
        )}
        <div aria-hidden="true" />
        {shown.map((gaugeIndex) => (
          <Fragment key={`axis-${gaugeIndex}`}>
            <GaugeAxis />
            <div aria-hidden="true" />
          </Fragment>
        ))}
      </div>
      <UsageHatchLegend t={t} variant="gauge" />
    </div>
  )
}
