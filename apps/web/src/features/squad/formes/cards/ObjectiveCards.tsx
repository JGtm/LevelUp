/**
 * ObjectiveCards.tsx — LES CINQ CARTES DU BLOC « OBJECTIFS » (artefact
 * 2ec1b8eb) : trois en contexte Solo, deux en contexte Escouade.
 *
 * DEUX LECTURES QUI NE DISENT PAS LA MÊME CHOSE. Par RÔLE (prendre / défendre /
 * tenir), la part réconcilie des modes qui n'ont pas les mêmes colonnes. Par
 * COLONNE RÉELLE, on retrouve le geste que le rôle avait fondu. Les deux sont
 * dans le bloc, et la note de chaque carte dit laquelle répond à quoi.
 *
 * UNE DURÉE NE SE COMPARE QU'À SA PROPRE PARITÉ : les colonnes « tenir » sont en
 * secondes et s'écrivent en m:ss.
 */
import { FormesCaption, FormesCard, FormesSubtitle } from '../FormesCard'
import { MINUS_INK, PLUS_INK, SPREAD_INK, TEAM_REST_INK, squadPlayerInk } from '../colors'
import { EcartForm, type EcartRow } from '../forms/EcartForm'
import { GrilleForm, type GrilleColumn, type GrilleRow } from '../forms/GrilleForm'
import { JaugeDoubleForm, type JaugeRow } from '../forms/JaugeDoubleForm'
import { Piste100Form, type PisteRow } from '../forms/Piste100Form'
import {
  OBJECTIVE_ROLES,
  aggregateColumns,
  aggregateRole,
  columnsOfFamily,
  matchesOfFamily,
  objectiveFamilies,
  objectiveValue,
  roleLobbyParts,
} from '../model/objectives'
import type { FormesViewModel } from '../viewModel'
import { listWindow } from '../model/display'
import { lobbyTrackRow } from './shared'

/** La colonne des noms d'une grille par match — cf. EquipmentCards. */
const MATCH_NAME_WIDTH = 240

/** Le libellé d'une famille de mode (clé stable -> nom du titre). */
function familyLabel(vm: FormesViewModel, family: string): string {
  return vm.ct.families[family] ?? family
}

/** Carte 15 — « Écart à la parité, par rôle » (deux lignes par rôle). */
export function ObjectivesGapRoleCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const rows: EcartRow[] = []
  for (const role of OBJECTIVE_ROLES) {
    const agg = aggregateRole(vm.block, role)
    if (agg.lobby === 0 && agg.team === 0) continue
    rows.push({
      key: `${role}-team`,
      label: t.roles[role],
      sublabel: t.common.inMyTeam,
      pct: agg.myShareOfTeamPct,
      parity: agg.teamParity ?? 0,
      detail: t.roleHints[role],
    })
    rows.push({
      key: `${role}-lobby`,
      label: '',
      sublabel: t.common.inLobby,
      pct: agg.myShareOfLobbyPct,
      parity: agg.lobbyParity ?? 0,
      detail: t.roleHints[role],
    })
  }
  return (
    <FormesCard
      title={ct.cards.objectivesGapRole.title}
      note={ct.cards.objectivesGapRole.note}
      legend={[
        { label: t.common.morePlus, ink: PLUS_INK },
        { label: t.common.less, ink: MINUS_INK },
        { label: t.common.lineParity, line: true },
      ]}
    >
      <EcartForm
        rows={rows}
        formatPct={vm.fmtPct}
        formatSigned={vm.fmtSigned}
        notMeasuredLabel={t.common.notMeasured}
        parityTipFmt={t.common.parityFmt}
        rowTipFmt={t.common.gapTipFmt}
        pointsFmt={t.common.pointsFmt}
        axisTitle={t.common.gapAxis}
        parityAxisLabel={t.common.parity}
      />
    </FormesCard>
  )
}

/** Carte 16 — « Ma part par famille de mode » (jauge double, colonnes réelles). */
export function ObjectivesSharesByFamilyCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  return (
    <FormesCard
      title={ct.cards.objectivesSharesByFamily.title}
      note={ct.cards.objectivesSharesByFamily.note}
      legend={[
        { label: t.common.myShare, ink: squadPlayerInk(0) },
        { label: t.common.spreadByMatch, ink: SPREAD_INK },
        { label: t.common.lineParity, line: true },
      ]}
    >
      {objectiveFamilies(vm.block).map((family) => {
        const matches = matchesOfFamily(vm.block, family)
        const rows: JaugeRow[] = columnsOfFamily(vm.block, family).map((col) => {
          const agg = aggregateColumns(matches, vm.mainXuid, [col.key])
          return {
            key: `${family}-${col.key}`,
            label: t.columns[col.key] ?? col.key,
            tracks: [
              {
                side: t.common.inMyTeam,
                pct: agg.myShareOfTeamPct,
                parity: agg.teamParity,
                min: agg.teamSpread.min,
                max: agg.teamSpread.max,
                above: agg.teamSpread.above,
                measured: agg.teamSpread.measured,
                detail: t.common.outOfFmt(
                  vm.fmtCount(agg.me, col.duration),
                  vm.fmtCount(agg.team, col.duration),
                ),
              },
              {
                side: t.common.inLobby,
                pct: agg.myShareOfLobbyPct,
                parity: agg.lobbyParity,
                min: agg.lobbySpread.min,
                max: agg.lobbySpread.max,
                above: agg.lobbySpread.above,
                measured: agg.lobbySpread.measured,
                detail: t.common.outOfFmt(
                  vm.fmtCount(agg.me, col.duration),
                  vm.fmtCount(agg.lobby, col.duration),
                ),
              },
            ],
          }
        })
        return (
          <div key={family}>
            <FormesSubtitle>{ct.familyMatchesFmt(familyLabel(vm, family), matches.length)}</FormesSubtitle>
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
          </div>
        )
      })}
    </FormesCard>
  )
}

/** Carte 17 — « Objectif par mode, en valeurs brutes » (grille par famille). */
export function ObjectivesRawGridCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  return (
    <FormesCard
      title={ct.cards.objectivesRawGrid.title}
      note={ct.cards.objectivesRawGrid.note}
      legend={[{ label: vm.squad[0]?.label ?? '', ink: squadPlayerInk(0) }]}
    >
      {objectiveFamilies(vm.block).map((family) => {
        // LA FEUILLE DE MATCH N'A PAS BESOIN DE FILM : aucun match n'est écarté
        // ici, la forme se borne seulement en nombre (les plus récents).
        const shown = listWindow(matchesOfFamily(vm.block, family))
        const matches = shown.rows
        const cols = columnsOfFamily(vm.block, family)
        const rows: GrilleRow[] = matches.map((m) => ({
          key: m.match_id,
          label: vm.matchLabel(m),
          sublabel: vm.matchMap(m),
          accent: squadPlayerInk(0),
        }))
        const columns: GrilleColumn[] = cols.map((c) => ({
          key: c.key,
          label: t.columns[c.key] ?? c.key,
          duration: c.duration,
        }))
        const byId = new Map(matches.map((m) => [m.match_id, m]))
        return (
          <div key={family}>
            <FormesSubtitle>{ct.familyMatchesFmt(familyLabel(vm, family), matches.length)}</FormesSubtitle>
            <GrilleForm
              rows={rows}
              columns={columns}
              value={(row, col) => {
                const match = byId.get(row.key)
                return match ? objectiveValue(match, vm.mainXuid, col.key) : null
              }}
              ink={() => squadPlayerInk(0)}
              format={(v, col) => vm.fmtCount(v, col.duration)}
              tooltip={(row, col, text) => t.common.valueTipFmt(row.label, col.label, text)}
              notMeasuredLabel={t.common.notMeasured}
              axisTitle={t.common.gesturesAxis}
              nameWidth={MATCH_NAME_WIDTH}
            />
            <FormesCaption>{t.common.foldListFmt(matches.length, shown.hidden)}</FormesCaption>
          </div>
        )
      })}
    </FormesCard>
  )
}

/** Carte 18 — « Rapport de force par famille de mode » (écart, parité 50 %). */
export function ObjectivesGapSquadCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  return (
    <FormesCard
      title={ct.cards.objectivesGapSquad.title}
      note={ct.cards.objectivesGapSquad.note}
      legend={[
        { label: t.common.moreThanOpponent, ink: PLUS_INK },
        { label: t.common.less, ink: MINUS_INK },
        { label: t.common.parity, line: true },
      ]}
    >
      {objectiveFamilies(vm.block).map((family) => {
        const matches = matchesOfFamily(vm.block, family)
        const rows: EcartRow[] = columnsOfFamily(vm.block, family).map((col) => {
          const agg = aggregateColumns(matches, vm.mainXuid, [col.key])
          return {
            key: `${family}-${col.key}`,
            label: t.columns[col.key] ?? col.key,
            pct: agg.teamShareOfLobbyPct,
            parity: 50,
            detail: t.common.outOfFmt(
              vm.fmtCount(agg.team, col.duration),
              vm.fmtCount(agg.lobby - agg.team, col.duration),
            ),
          }
        })
        return (
          <div key={family}>
            <FormesSubtitle>{ct.familyMatchesFmt(familyLabel(vm, family), matches.length)}</FormesSubtitle>
            <EcartForm
              rows={rows}
              formatPct={vm.fmtPct}
              formatSigned={vm.fmtSigned}
              notMeasuredLabel={t.common.notMeasured}
              parityTipFmt={t.common.parityFmt}
              rowTipFmt={t.common.gapTipFmt}
              pointsFmt={t.common.pointsFmt}
              axisTitle={t.common.gapAxis}
              parityAxisLabel={t.common.parity}
            />
          </div>
        )
      })}
    </FormesCard>
  )
}

/** Carte 19 — « Ce que mon camp prend de l'objectif » (piste 100, par rôle). */
export function ObjectivesLobbyTrackCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const squadXuids = vm.squad.map((s) => s.xuid)
  const rows: PisteRow[] = OBJECTIVE_ROLES.map((role) =>
    lobbyTrackRow(
      vm,
      role,
      t.roles[role],
      roleLobbyParts(vm.block, role, squadXuids),
      t.roleHints[role],
    ),
  )
  return (
    <FormesCard
      title={ct.cards.objectivesLobbyTrack.title}
      note={ct.cards.objectivesLobbyTrack.note}
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
