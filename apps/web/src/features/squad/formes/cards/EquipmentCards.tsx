/**
 * EquipmentCards.tsx — LES TROIS CARTES DU BLOC « USAGES D'ÉQUIPEMENTS »
 * (artefact 2ec1b8eb), contexte Solo (Séries temporelles). Les quatre cartes du contexte
 * Escouade ont été retirées avec l'ancien onglet Usages (lot L5.4 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : l'onglet Emprise les remplace.
 *
 * LES CINQ GESTES SONT CEUX QUE LA PÉRIODE MESURE — camouflage, mur de
 * protection, surbouclier, grappin, objets lâchés au sol. Les grenades n'en sont
 * pas (ce ne sont pas des équipements) et le répulseur n'a pas de ligne (aucun
 * canal ne mesure son usage).
 *
 * AUCUNE CADENCE PAR MINUTE : les comptes se lisent PAR MATCH (décision
 * utilisateur du 2026-09-13).
 */
import { equipmentFamilyLabel, USAGE_TEXT } from '@/features/_shared/usage/usageI18n'

import { FormesCard } from '../FormesCard'
import { SPREAD_INK, axisInk } from '../colors'
import { BatonMinMaxForm } from '../forms/BatonMinMaxForm'
import { GrilleForm, type GrilleColumn, type GrilleRow } from '../forms/GrilleForm'
import { JaugeDoubleForm, type JaugeRow } from '../forms/JaugeDoubleForm'
import {
  EQUIPMENT_AXES,
  droppedFamiliesOf,
  playerAxisValue,
  playerDroppedFamily,
  type EquipmentAxis,
} from '../model/access'
import { measuredWindow } from '../model/display'
import { aggregateAxis, playerSpread, playerTotal } from '../model/aggregates'
import type { FormesViewModel } from '../viewModel'

/**
 * La colonne des noms d'une grille PAR MATCH : « 22:51 · Assassin en équipe » et
 * sa carte ne tiennent pas dans la largeur par défaut (mesuré sur les vrais
 * libellés de mode du titre).
 */
const MATCH_NAME_WIDTH = 240

/** Carte 1 — « Ma part, dans mon équipe et dans le lobby » (jauge double). */
export function EquipmentSharesCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const rows: JaugeRow[] = EQUIPMENT_AXES.map((axis) => {
    const agg = aggregateAxis(vm.block, axis)
    return {
      key: axis,
      label: t.axes[axis],
      tracks: [
        {
          side: t.common.inMyTeam,
          pct: agg.team.pct,
          parity: agg.team.parity,
          min: agg.team.min,
          max: agg.team.max,
          above: agg.team.above,
          measured: agg.team.measured,
          detail: t.common.outOfFmt(vm.fmtCount(agg.team.value), vm.fmtCount(agg.team.total)),
        },
        {
          side: t.common.inLobby,
          pct: agg.lobby.pct,
          parity: agg.lobby.parity,
          min: agg.lobby.min,
          max: agg.lobby.max,
          above: agg.lobby.above,
          measured: agg.lobby.measured,
          detail: t.common.outOfFmt(vm.fmtCount(agg.lobby.value), vm.fmtCount(agg.lobby.total)),
        },
      ],
    }
  })
  return (
    <FormesCard
      title={ct.cards.equipmentShares.title}
      note={ct.cards.equipmentShares.note}
      legend={[
        { label: t.common.myShare, ink: vm.squad[0]?.ink },
        { label: t.common.spreadByMatch, ink: SPREAD_INK },
        { label: t.common.lineParity, line: true },
      ]}
    >
      <JaugeDoubleForm
        rows={rows}
        formatPct={vm.fmtPct}
        notMeasuredLabel={t.common.notMeasured}
        aboveFmt={t.common.aboveFmt}
        parityTipFmt={t.common.parityFmt}
        shareTipFmt={t.common.shareTipFmt}
        spreadTipFmt={t.common.spreadTipFmt}
        axisTitle={t.common.shareAxis}
      />
    </FormesCard>
  )
}

/**
 * LES AXES DE LA CADENCE PAR MATCH — SANS l'axe GÉNÉRIQUE « Objets lâchés au sol »
 * (décision D9 du 2026-09-21). Une colonne qui mélange un mur, un grappin et un capteur
 * lâchés à la mort ne se compare à rien.
 *
 * ELLE EST REMPLACÉE, LE MÊME JOUR (lot G), PAR UNE COLONNE PAR FAMILLE LÂCHÉE : le bloc
 * `formes_retenues` sert désormais la ventilation (`dropped_by_family`, ajoutée à
 * `domain.SquadFormesLobbyPlayer` — elle existait en amont depuis `usage_summary.go` et
 * s'arrêtait à `analysis/squadformes/formes.go`). Les colonnes sont celles que la PÉRIODE
 * porte réellement, jamais une liste écrite d'avance. Les autres formes du bloc gardent
 * l'axe global : elles le lisent comme un volume, pas comme une colonne de comparaison.
 */
const BY_MATCH_AXES = EQUIPMENT_AXES.filter((axis) => axis !== 'dropped')

/** Préfixe des clés de colonne « famille lâchée » — jamais collision avec un axe. */
const DROPPED_COL = 'dropped:'

/** Carte 2 — « Cadence de gestes, match par match » (grille, une ligne par match). */
export function EquipmentByMatchCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  // UNE LIGNE PAR MATCH MESURÉ, les plus récents : sur une portée de mille
  // matchs, une ligne par match donnait quinze écrans de hachure.
  const shown = measuredWindow(vm.block)
  const rows: GrilleRow[] = shown.rows.map((m) => ({
    key: m.match_id,
    label: vm.matchLabel(m),
    sublabel: vm.matchMap(m),
    accent: vm.squad[0]?.ink,
  }))
  // LES FAMILLES LÂCHÉES VIENNENT DE LA PÉRIODE AFFICHÉE, pas d'une liste : une famille
  // que personne n'a lâchée n'a pas de colonne à zéro. Elles se posent APRÈS les usages.
  const droppedFamilies = droppedFamiliesOf(shown.rows, vm.mainXuid)
  const ut = USAGE_TEXT[vm.locale]
  const columns: GrilleColumn[] = [
    ...BY_MATCH_AXES.map((axis) => ({
      key: axis as string,
      label: t.axes[axis],
      total: t.common.totalFmt(vm.fmtCount(playerTotal(vm.block, vm.mainXuid, axis))),
    })),
    ...droppedFamilies.map((family) => ({
      key: `${DROPPED_COL}${family}`,
      label: t.common.droppedFamilyFmt(equipmentFamilyLabel(family, ut)),
      total: t.common.totalFmt(
        vm.fmtCount(
          shown.rows.reduce((n, m) => n + playerDroppedFamily(m, vm.mainXuid, family), 0),
        ),
      ),
    })),
  ]
  const matchById = new Map(shown.rows.map((m) => [m.match_id, m]))
  return (
    <FormesCard
      title={ct.cards.equipmentByMatch.title}
      note={ct.cards.equipmentByMatch.note}
      help={[t.common.foldMeasuredFmt(shown.rows.length, shown.hidden, shown.unmeasured)]}
      legend={[
        ...BY_MATCH_AXES.map((axis) => ({ label: t.axes[axis], ink: axisInk(axis) })),
        ...(droppedFamilies.length > 0
          ? [{ label: t.axes.dropped, ink: axisInk('dropped') }]
          : []),
      ]}
    >
      <GrilleForm
        rows={rows}
        columns={columns}
        value={(row, col) => {
          const match = matchById.get(row.key)
          if (!match) return null
          return col.key.startsWith(DROPPED_COL)
            ? playerDroppedFamily(match, vm.mainXuid, col.key.slice(DROPPED_COL.length))
            : playerAxisValue(match, vm.mainXuid, col.key as EquipmentAxis)
        }}
        // Toutes les colonnes de lâcher partagent l'encre de l'axe « lâchés » : elles
        // disent la même grandeur, découpée — une encre par famille ferait croire à cinq
        // mesures différentes.
        ink={(_row, col) =>
          col.key.startsWith(DROPPED_COL)
            ? axisInk('dropped')
            : axisInk(col.key as EquipmentAxis)
        }
        format={(v) => vm.fmtCount(v)}
        tooltip={(row, col, text) => t.common.valueTipFmt(row.label, col.label, text)}
        notMeasuredLabel={t.common.noFilm}
        axisTitle={t.common.gesturesPerMatchAxis}
        nameWidth={MATCH_NAME_WIDTH}
      />
    </FormesCard>
  )
}

/** Carte 3 — « Étendue et moyenne de la période » (bâton min-max). */
export function EquipmentSpreadCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const series = EQUIPMENT_AXES.map((axis) => {
    const spread = playerSpread(vm.block, vm.mainXuid, axis)
    return {
      key: axis,
      label: t.axes[axis],
      ink: axisInk(axis),
      min: spread?.min ?? 0,
      max: spread?.max ?? 0,
      mean: spread?.mean ?? 0,
    }
  })
  return (
    <FormesCard
      title={ct.cards.equipmentSpread.title}
      note={ct.cards.equipmentSpread.note}
      legend={[
        { label: t.common.lowestToHighest, ink: axisInk('camo') },
        { label: t.common.meanPerMatch, line: true },
      ]}
    >
      <BatonMinMaxForm
        series={series}
        formatValue={(v) => vm.fmtCount(v)}
        meanPrefix={t.common.meanPrefix}
        rangeTipFmt={t.common.rangeTipFmt}
        meanTipFmt={t.common.meanTipFmt}
        axisTitle={t.common.spreadAxis}
      />
    </FormesCard>
  )
}
