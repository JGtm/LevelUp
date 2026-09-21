/**
 * EquipmentCards.tsx — LES SEPT CARTES DU BLOC « USAGES D'ÉQUIPEMENTS »
 * (artefact 2ec1b8eb) : trois en contexte Solo, quatre en contexte Escouade.
 *
 * LES CINQ GESTES SONT CEUX QUE LA PÉRIODE MESURE — camouflage, mur de
 * protection, surbouclier, grappin, objets lâchés au sol. Les grenades n'en sont
 * pas (ce ne sont pas des équipements) et le répulseur n'a pas de ligne (aucun
 * canal ne mesure son usage).
 *
 * AUCUNE CADENCE PAR MINUTE : les comptes se lisent PAR MATCH (décision
 * utilisateur du 2026-09-13).
 */
import { FormesCard } from '../FormesCard'
import { MINUS_INK, PLUS_INK, SPREAD_INK, TEAM_REST_INK, axisInk } from '../colors'
import { BandeForm, type BandeRow } from '../forms/BandeForm'
import { BatonMinMaxForm } from '../forms/BatonMinMaxForm'
import { GrilleForm, type GrilleColumn, type GrilleRow } from '../forms/GrilleForm'
import { JaugeDoubleForm, type JaugeRow } from '../forms/JaugeDoubleForm'
import { Piste100Form, type PisteRow } from '../forms/Piste100Form'
import { EQUIPMENT_AXES, playerAxisValue, type EquipmentAxis } from '../model/access'
import { measuredWindow } from '../model/display'
import {
  aggregateAxis,
  lobbyParts,
  playerSpread,
  playerTotal,
  teamShareOfMatch,
} from '../model/aggregates'
import type { FormesViewModel } from '../viewModel'
import { lobbyTrackRow, matchColumns, squadSegments } from './shared'

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
 * LES AXES DE LA CADENCE PAR MATCH — SANS « Objets lâchés au sol » (décision D9 du
 * 2026-09-21). La colonne existait, mais le bloc `formes_retenues` ne sert qu'un COMPTE
 * global de lâchers (`SquadFormesLobbyPlayer.Dropped`, un entier) : la ventilation par
 * famille existe en amont dans le film (`replay.PlayerUsage.DroppedByFamily`) et s'arrête à
 * `analysis/squadformes/formes.go`. Une colonne qui mélange un mur, un grappin et un capteur
 * lâchés à la mort ne se compare à rien — tant que la famille n'est pas servie, la colonne
 * ne se tient pas. Les autres formes du bloc gardent l'axe : elles le lisent comme un
 * volume, pas comme une colonne de comparaison.
 */
const BY_MATCH_AXES = EQUIPMENT_AXES.filter((axis) => axis !== 'dropped')

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
  const columns: GrilleColumn[] = BY_MATCH_AXES.map((axis) => ({
    key: axis,
    label: t.axes[axis],
    total: t.common.totalFmt(vm.fmtCount(playerTotal(vm.block, vm.mainXuid, axis))),
  }))
  const matchById = new Map(shown.rows.map((m) => [m.match_id, m]))
  return (
    <FormesCard
      title={ct.cards.equipmentByMatch.title}
      note={ct.cards.equipmentByMatch.note}
      help={[t.common.foldMeasuredFmt(shown.rows.length, shown.hidden, shown.unmeasured)]}
      legend={BY_MATCH_AXES.map((axis) => ({ label: t.axes[axis], ink: axisInk(axis) }))}
    >
      <GrilleForm
        rows={rows}
        columns={columns}
        value={(row, col) => {
          const match = matchById.get(row.key)
          return match ? playerAxisValue(match, vm.mainXuid, col.key as EquipmentAxis) : null
        }}
        ink={(_row, col) => axisInk(col.key as EquipmentAxis)}
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

/** Carte 4 — « Régularité match par match » (bande, part de mon camp vs 50 %). */
export function EquipmentRegularityCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const shown = measuredWindow(vm.block)
  const rows: BandeRow[] = EQUIPMENT_AXES.map((axis) => ({
    key: axis,
    label: t.axes[axis],
    cells: shown.rows.map((m) => {
      const pct = teamShareOfMatch(m, vm.mainXuid, axis)
      const label = `${vm.matchLabel(m)} · ${vm.matchMap(m)}`
      return {
        key: m.match_id,
        pct,
        tooltip:
          pct == null
            ? `${label} — ${m.measured ? t.common.noMeasureOnAxis : t.common.noFilm}`
            : t.common.matchTipFmt(
                label,
                t.axes[axis],
                vm.fmtPct(pct),
                vm.fmtSigned(pct - 50),
              ),
      }
    }),
  }))
  return (
    <FormesCard
      title={ct.cards.equipmentRegularity.title}
      note={ct.cards.equipmentRegularity.note}
      help={[t.common.foldMeasuredFmt(shown.rows.length, shown.hidden, shown.unmeasured)]}
      legend={[
        { label: t.common.moreThanOpponent, ink: PLUS_INK },
        { label: t.common.less, ink: MINUS_INK },
        { label: t.common.noMeasureOnAxis, unmeasured: true },
      ]}
    >
      <BandeForm
        rows={rows}
        columns={matchColumns(vm, shown.rows)}
        parity={50}
        axisTitle={t.common.matchesAxis}
      />
    </FormesCard>
  )
}

/** Carte 5 — « Ce que mon camp prend du lobby » (piste 100 avec parité). */
export function EquipmentLobbyTrackCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const rows: PisteRow[] = EQUIPMENT_AXES.map((axis) => {
    const parts = lobbyParts(vm.measured, vm.squad.map((s) => s.xuid), axis)
    return lobbyTrackRow(vm, axis, t.axes[axis], parts)
  })
  return (
    <FormesCard
      title={ct.cards.equipmentLobbyTrack.title}
      note={ct.cards.equipmentLobbyTrack.note}
      legend={[
        ...vm.squad.map((s) => ({ label: s.label, ink: s.ink })),
        { label: t.common.teamRest, ink: TEAM_REST_INK },
        { label: t.common.enemyTeamCounted, hatch: true },
        { label: t.common.parity, line: true },
      ]}
    >
      <Piste100Form
        rows={rows}
        showParity
        axisTitle={t.common.lobbyShareAxis}
        parityLabel={t.common.parityFmt('50 %')}
        emptyLabel={t.common.noMeasure}
        formatCount={(v) => vm.fmtCount(v)}
        segmentTipFmt={t.common.segmentTipFmt}
      />
    </FormesCard>
  )
}

/** Carte 6 — « Cadence de chacun sur la période » (grille, une ligne par joueur). */
export function EquipmentSquadGridCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const measured = vm.measured.length
  const rows: GrilleRow[] = vm.squad.map((s) => ({ key: s.xuid, label: s.label, accent: s.ink }))
  const columns: GrilleColumn[] = EQUIPMENT_AXES.map((axis) => ({
    key: axis,
    label: t.axes[axis],
    total: t.common.totalFmt(
      vm.fmtCount(vm.squad.reduce((a, s) => a + playerTotal(vm.block, s.xuid, axis), 0)),
    ),
  }))
  const inkByXuid = new Map(vm.squad.map((s) => [s.xuid, s.ink]))
  return (
    <FormesCard
      title={ct.cards.equipmentSquadGrid.title}
      note={ct.cards.equipmentSquadGrid.note}
      legend={vm.squad.map((s) => ({ label: s.label, ink: s.ink }))}
    >
      <GrilleForm
        rows={rows}
        columns={columns}
        value={(row, col) =>
          measured === 0
            ? null
            : playerTotal(vm.block, row.key, col.key as EquipmentAxis) / measured
        }
        ink={(row) => inkByXuid.get(row.key) ?? ''}
        format={(v) => vm.fmtCount(v)}
        tooltip={(row, col, text) => t.common.valueTipFmt(row.label, col.label, text)}
        notMeasuredLabel={t.common.notMeasured}
        axisTitle={t.common.gesturesPerMatchAxis}
      />
    </FormesCard>
  )
}

/** Carte 7 — « Qui porte quel geste dans l'escouade » (piste 100 SANS parité). */
export function EquipmentSquadTrackCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const rows: PisteRow[] = EQUIPMENT_AXES.map((axis) => {
    const segments = squadSegments(vm, (xuid) => playerTotal(vm.block, xuid, axis))
    const total = segments.reduce((a, s) => a + s.value, 0)
    return {
      key: axis,
      label: t.axes[axis],
      sublabel: t.common.inSquadFmt(total),
      segments,
    }
  })
  return (
    <FormesCard
      title={ct.cards.equipmentSquadTrack.title}
      note={ct.cards.equipmentSquadTrack.note}
      legend={vm.squad.map((s) => ({ label: s.label, ink: s.ink }))}
    >
      <Piste100Form
        rows={rows}
        showParity={false}
        axisTitle={t.common.squadShareAxis}
        parityLabel=""
        emptyLabel={t.common.noMeasure}
        formatCount={(v) => vm.fmtCount(v)}
        segmentTipFmt={t.common.segmentTipFmt}
      />
    </FormesCard>
  )
}
