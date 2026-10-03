/**
 * PadCards.tsx — LES TROIS CARTES DU BLOC « CONTRÔLE DES ARMES SPÉCIALES »
 * (artefact 2ec1b8eb), contexte Solo (Séries temporelles). Les quatre cartes du contexte
 * Escouade ont été retirées avec l'ancien onglet Usages (lot L5.4 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : l'onglet Emprise les remplace.
 *
 * LE DÉNOMINATEUR EST LE NOMBRE DE PRISES NOMMÉES, jamais le nombre de socles :
 * les occupations sans ramasseur nommé n'entrent dans aucun camp.
 */
import { FormesCard, FormesSubtitle } from '../FormesCard'
import { MINUS_INK, PLUS_INK, SPREAD_INK, squadPlayerInk } from '../colors'
import { BandeForm } from '../forms/BandeForm'
import { EcartForm, type EcartRow } from '../forms/EcartForm'
import { GrilleForm, type GrilleColumn, type GrilleRow } from '../forms/GrilleForm'
import { JaugeDoubleForm } from '../forms/JaugeDoubleForm'
import { PAD_AXIS, sharePct } from '../model/access'
import { measuredWindow } from '../model/display'
import { aggregateAxis, myShareOfMatch } from '../model/aggregates'
import {
  WEAPON_CLASSES,
  aggregateWeaponClass,
  matchesWithWeapon,
  weaponOccupations,
  weaponPickupsByPlayer,
  weaponsByVolume,
} from '../model/pads'
import type { FormesViewModel } from '../viewModel'
import { matchColumns } from './shared'

/** Le nombre d'armes détaillées par la grille solo ; les autres sont repliées. */
const SOLO_WEAPON_ROWS = 12
/** La colonne des noms d'arme : un nom et son dénominateur tiennent en 210 px. */
const WEAPON_NAME_WIDTH = 210

/** Carte 8 — « Écart à la parité, par famille d'arme » (contexte Solo). */
export function PadsGapSoloCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const all = aggregateAxis(vm.block, PAD_AXIS)
  const rows: EcartRow[] = [
    {
      key: 'all',
      label: t.allWeapons,
      sublabel: t.allWeaponsHint,
      pct: all.lobby.pct,
      parity: all.lobby.parity ?? 0,
      detail: t.common.outOfFmt(vm.fmtCount(all.lobby.value), vm.fmtCount(all.lobby.total)),
    },
    ...WEAPON_CLASSES.map((cls) => {
      const agg = aggregateWeaponClass(vm.block, vm.weapons, cls)
      return {
        key: cls,
        label: t.weaponClasses[cls],
        pct: agg.myShareOfLobbyPct,
        parity: agg.lobbyParity ?? 0,
        detail: t.common.outOfFmt(vm.fmtCount(agg.me), vm.fmtCount(agg.lobby)),
      }
    }),
  ]
  return (
    <FormesCard
      title={ct.cards.padsGapSolo.title}
      note={ct.cards.padsGapSolo.note}
      legend={[
        { label: t.common.morePlus, ink: PLUS_INK },
        { label: t.common.lessThanLobby, ink: MINUS_INK },
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

/** Carte 9 — « Ma part des prises de socle, et sa dispersion » (jauge + bande). */
export function PadsShareSoloCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const agg = aggregateAxis(vm.block, PAD_AXIS)
  const parity = agg.lobby.parity ?? 0
  const shown = measuredWindow(vm.block)
  return (
    <FormesCard
      title={ct.cards.padsShareSolo.title}
      note={ct.cards.padsShareSolo.note}
      help={[t.common.foldMeasuredFmt(shown.rows.length, shown.hidden, shown.unmeasured)]}
      legend={[
        { label: t.common.myShare, ink: squadPlayerInk(0) },
        { label: t.common.spread, ink: SPREAD_INK },
        { label: t.common.parity, line: true },
        { label: t.common.aboveBelowByMatch, ink: PLUS_INK },
      ]}
    >
      <JaugeDoubleForm
        rows={[
          {
            key: 'pads',
            label: t.common.myShare,
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
                detail: t.common.outOfFmt(
                  vm.fmtCount(agg.lobby.value),
                  vm.fmtCount(agg.lobby.total),
                ),
              },
            ],
          },
        ]}
        formatPct={vm.fmtPct}
        notMeasuredLabel={t.common.notMeasured}
        aboveFmt={t.common.aboveFmt}
        parityTipFmt={t.common.parityFmt}
        shareTipFmt={t.common.shareTipFmt}
        spreadTipFmt={t.common.spreadTipFmt}
        axisTitle={t.common.shareAxis}
      />
      <FormesSubtitle>{ct.subtitles.myShareByMatch}</FormesSubtitle>
      <BandeForm
        rows={[
          {
            key: 'my-share',
            label: t.common.myShare,
            cells: shown.rows.map((m) => {
              const pct = myShareOfMatch(m, vm.mainXuid, PAD_AXIS)
              const label = `${vm.matchLabel(m)} · ${vm.matchMap(m)}`
              return {
                key: m.match_id,
                pct,
                tooltip:
                  pct == null
                    ? `${label} — ${m.measured ? t.common.noMeasureOnAxis : t.common.noFilm}`
                    : t.common.matchTipFmt(
                        label,
                        t.common.myShare,
                        vm.fmtPct(pct),
                        vm.fmtSigned(pct - parity),
                      ),
              }
            }),
          },
        ]}
        columns={matchColumns(vm, shown.rows)}
        parity={parity}
        axisTitle={t.common.matchesAxis}
      />
    </FormesCard>
  )
}

/** Carte 10 — « Taux de rafle par arme » (grille, 3 colonnes). */
export function PadsWeaponGridCard({ vm }: { vm: FormesViewModel }) {
  const { t, ct } = vm
  const weapons = weaponsByVolume(vm.block).slice(0, SOLO_WEAPON_ROWS)
  const rows: GrilleRow[] = weapons.map((w) => ({
    key: w.key,
    label: vm.weaponLabel(w.key),
    sublabel: t.common.matchesShortFmt(matchesWithWeapon(vm.block, w.key)),
    accent: squadPlayerInk(0),
  }))
  const columns: GrilleColumn[] = [
    { key: 'rate', label: t.padsColumns.rate, percent: true },
    { key: 'mine', label: t.padsColumns.mine },
    { key: 'occ', label: t.padsColumns.occupations },
  ]
  return (
    <FormesCard
      title={ct.cards.padsWeaponGrid.title}
      note={ct.cards.padsWeaponGrid.note}
      legend={[{ label: t.padsColumns.rateLegend, ink: squadPlayerInk(0) }]}
    >
      <GrilleForm
        rows={rows}
        columns={columns}
        value={(row, col) => {
          const occ = weaponOccupations(vm.block, row.key)
          const mine = weaponPickupsByPlayer(vm.block, row.key, vm.mainXuid)
          if (col.key === 'rate') return sharePct(mine, occ) ?? 0
          return col.key === 'mine' ? mine : occ
        }}
        ink={() => squadPlayerInk(0)}
        format={(v, col) => (col.key === 'rate' ? vm.fmtPct(v) : vm.fmtCount(v))}
        tooltip={(row, col, text) => t.common.valueTipFmt(row.label, col.label, text)}
        notMeasuredLabel={t.common.notMeasured}
        axisTitle={t.padsColumns.rateAxis}
        nameWidth={WEAPON_NAME_WIDTH}
      />
    </FormesCard>
  )
}
