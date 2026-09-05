/**
 * usageGrids.ts — LES GRILLES ALIGNÉES du bloc « usages d'équipement, socles et
 * objectifs » (chantier session-usage S3) : trois projections de la primitive
 * partagée `components/charts/valueGridModel` (une échelle et un axe PAR COLONNE,
 * jamais une seconde grille — handoff §5/S3).
 *
 * 1. `buildCadenceGrid` — cadences par dix minutes : lignes = moi + coéquipiers
 *    suivis, puis les agrégats (mon équipe, lobby) derrière un filet ; colonnes =
 *    grandeurs d'équipement.
 * 2. `buildObjectiveFamilyGrid` — ma part d'équipe par famille de mode × rôle.
 * 3. `buildSquadRoleGrid` — contexte ESCOUADE : la part d'équipe de chaque joueur
 *    suivi, rôle par rôle.
 *
 * Les COMPTES BRUTS ne sortent qu'en infobulle (dénominateur d'honnêteté, doctrine
 * §1) ; `undefined` du contrat → `null` de la grille (« non mesuré », jamais 0).
 * Pur : aucun React, aucune couleur en dur — les encres arrivent par `UsageGridInks`.
 */
import type { ValueGridModel } from '@/components/charts/valueGridModel'
import { buildValueGrid } from '@/components/charts/valueGridModel'
import {
  SQUAD_MAIN_PLAYER_TOKEN,
  SQUAD_TEAMMATE_COLOR_TOKENS,
} from '@/features/squad/colors'
import { tokenCssVar } from '@/lib/accessibility'
import type {
  SessionObjectiveFamilyBlock,
  SessionObjectiveRoleMetric,
  SessionUsageMetric,
  SessionUsageSquadPlayer,
  SessionUsageSquadShare,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { familyLabel, roleLabel, type UsageText } from './usageI18n'
import {
  ROLE_ORDER,
  formatUsageCount,
  formatUsagePct,
  formatUsageRate,
  metricLabel,
  sortRoles,
} from './usageLogic'

/**
 * L'encre d'identité d'un joueur suivi : moi = squad-player-1, coéquipiers = 2..4 —
 * la SOURCE UNIQUE d'affectation `features/squad/colors.ts` (import allowlisté
 * session-detail=>squad), pour qu'un joueur garde sa couleur d'une page à l'autre.
 */
export function usagePlayerInk(kind: 'me' | 'squad', squadIndex = 0): string {
  if (kind === 'me') return tokenCssVar(SQUAD_MAIN_PLAYER_TOKEN)
  const token = SQUAD_TEAMMATE_COLOR_TOKENS[squadIndex % SQUAD_TEAMMATE_COLOR_TOKENS.length]
  return tokenCssVar(token)
}

/** Ce que l'appelant fournit pour encrer les grilles (jetons résolus en CSS). */
export interface UsageGridInks {
  /** Encre d'une colonne de grandeur (famille de geste ou rôle d'objectif). */
  columnColor: (columnKey: string) => string
  /** Trait d'identité d'une ligne (joueur, coéquipier, agrégat). */
  rowAccent: (kind: 'me' | 'squad' | 'aggregate', squadIndex?: number) => string | undefined
}

export interface UsageCadenceGridInput extends UsageGridInks {
  metrics: SessionUsageMetric[]
  squadPlayers: SessionUsageSquadPlayer[]
  meLabel: string
  t: UsageText
  locale: Locale
}

/** Une ligne interne des grilles à lignes-joueurs. */
interface PlayerRowSpec<T> {
  key: string
  label: string
  group: string
  kind: 'me' | 'squad' | 'aggregate'
  squadIndex?: number
  emphasis?: boolean
  rate: (col: T) => number | null
  raw: (col: T) => number | null
}

/**
 * buildCadenceGrid — la grille des cadences par dix minutes de jeu mesuré. Un champ
 * de cadence absent (match sans échelle de temps, scope camp connu vide) rend une
 * cellule NON MESURÉE — le contrat refuse les cadences inventées, la grille aussi.
 */
export function buildCadenceGrid(input: UsageCadenceGridInput): ValueGridModel | null {
  const { metrics, t, locale } = input
  if (metrics.length === 0) return null

  const squadRow = (m: SessionUsageMetric, xuid: string): SessionUsageSquadShare | null =>
    (m.squad ?? []).find((s) => s.xuid === xuid) ?? null
  const rows: PlayerRowSpec<SessionUsageMetric>[] = [
    {
      key: 'me',
      label: input.meLabel,
      group: 'players',
      kind: 'me',
      emphasis: true,
      rate: (m) => m.player_per_10min ?? null,
      raw: (m) => m.player_total,
    },
    ...input.squadPlayers.map(
      (p, i): PlayerRowSpec<SessionUsageMetric> => ({
        key: `squad-${p.xuid}`,
        label: p.gamertag,
        group: 'players',
        kind: 'squad',
        squadIndex: i,
        rate: (m) => squadRow(m, p.xuid)?.per_10min ?? null,
        raw: (m) => squadRow(m, p.xuid)?.total ?? null,
      }),
    ),
    {
      key: 'team',
      label: t.rowMyTeam,
      group: 'aggregates',
      kind: 'aggregate',
      rate: (m) => m.team_per_10min ?? null,
      raw: (m) => m.team_total ?? null,
    },
    {
      key: 'lobby',
      label: t.rowLobby,
      group: 'aggregates',
      kind: 'aggregate',
      rate: (m) => m.lobby_per_10min ?? null,
      raw: (m) => m.lobby_total,
    },
  ]

  return buildValueGrid({
    rows: rows.map((r) => ({
      key: r.key,
      label: r.label,
      group: r.group,
      accent: input.rowAccent(r.kind, r.squadIndex),
      emphasis: r.emphasis,
    })),
    columns: metrics.map((m) => ({
      key: m.key,
      label: metricLabel(m.key, t),
      showTotal: false, // la somme de cadences n'a pas de sens
    })),
    value: (r, c) => rows[r].rate(metrics[c]),
    format: (v) => formatUsageRate(v, locale),
    color: (_r, c) => input.columnColor(metrics[c].key),
    tooltip: (r, c, text) =>
      t.cadenceTipFmt(
        rows[r].label,
        metricLabel(metrics[c].key, t),
        text,
        formatUsageCount(rows[r].raw(metrics[c]), locale),
      ),
  })
}

export interface UsageObjectiveFamilyGridInput extends UsageGridInks {
  families: SessionObjectiveFamilyBlock[]
  t: UsageText
  locale: Locale
}

/**
 * buildObjectiveFamilyGrid — « ma part d'équipe, par famille de mode » : lignes =
 * familles jouées, colonnes = rôles présents (une famille sans « tenir » a une
 * cellule non mesurée), valeur = part d'équipe du joueur en %.
 */
export function buildObjectiveFamilyGrid(
  input: UsageObjectiveFamilyGridInput,
): ValueGridModel | null {
  const { families, t, locale } = input
  if (families.length === 0) return null
  const roleKeys = ROLE_ORDER.filter((rk) =>
    families.some((f) => (f.roles ?? []).some((r) => r.role === rk)),
  )
  if (roleKeys.length === 0) return null

  const roleOf = (f: SessionObjectiveFamilyBlock, rk: string) =>
    (f.roles ?? []).find((r) => r.role === rk) ?? null

  return buildValueGrid({
    rows: families.map((f) => ({
      key: f.family,
      label: familyLabel(f.family, t),
      group: 'families',
      hint: `${familyLabel(f.family, t)} — ${f.matches}`,
    })),
    columns: roleKeys.map((rk) => ({ key: rk, label: roleLabel(rk, t), showTotal: false })),
    value: (r, c) => roleOf(families[r], roleKeys[c])?.player_share_of_team_pct ?? null,
    format: (v) => formatUsagePct(v, locale),
    color: (_r, c) => input.columnColor(roleKeys[c]),
    tooltip: (r, c, text) => {
      const role = roleOf(families[r], roleKeys[c])
      const honesty =
        role == null
          ? text
          : t.honestyFmt(
              formatUsageCount(role.player_total, locale, role.is_duration === true),
              formatUsageCount(role.team_total, locale, role.is_duration === true),
            )
      return t.gaugeTipFmt(
        `${familyLabel(families[r].family, t)} — ${roleLabel(roleKeys[c], t)}`,
        t.gaugePlayerOfTeam,
        text,
        honesty,
      )
    },
  })
}

export interface UsageSquadRoleGridInput extends UsageGridInks {
  roles: SessionObjectiveRoleMetric[]
  squadPlayers: SessionUsageSquadPlayer[]
  meLabel: string
  t: UsageText
  locale: Locale
}

/**
 * buildSquadRoleGrid — contexte ESCOUADE : la part d'équipe de chaque joueur suivi,
 * rôle par rôle (lignes = moi + coéquipiers, colonnes = rôles). `null` en solo :
 * la forme n'existe que quand la piste du lobby a des joueurs à découper.
 */
export function buildSquadRoleGrid(input: UsageSquadRoleGridInput): ValueGridModel | null {
  const { t, locale } = input
  if (input.squadPlayers.length === 0 || input.roles.length === 0) return null
  const roles = sortRoles(input.roles)

  const rows: PlayerRowSpec<SessionObjectiveRoleMetric>[] = [
    {
      key: 'me',
      label: input.meLabel,
      group: 'players',
      kind: 'me',
      emphasis: true,
      rate: (r) => r.player_share_of_team_pct ?? null,
      raw: (r) => r.player_total,
    },
    ...input.squadPlayers.map(
      (p, i): PlayerRowSpec<SessionObjectiveRoleMetric> => ({
        key: `squad-${p.xuid}`,
        label: p.gamertag,
        group: 'players',
        kind: 'squad',
        squadIndex: i,
        rate: (r) => (r.squad ?? []).find((s) => s.xuid === p.xuid)?.share_of_team_pct ?? null,
        raw: (r) => (r.squad ?? []).find((s) => s.xuid === p.xuid)?.total ?? null,
      }),
    ),
  ]

  return buildValueGrid({
    rows: rows.map((r) => ({
      key: r.key,
      label: r.label,
      group: r.group,
      accent: input.rowAccent(r.kind, r.squadIndex),
      emphasis: r.emphasis,
    })),
    columns: roles.map((r) => ({ key: r.role, label: roleLabel(r.role, t), showTotal: false })),
    value: (r, c) => rows[r].rate(roles[c]),
    format: (v) => formatUsagePct(v, locale),
    color: (_r, c) => input.columnColor(roles[c].role),
    tooltip: (r, c, text) =>
      t.gaugeTipFmt(
        `${rows[r].label} — ${roleLabel(roles[c].role, t)}`,
        t.gaugePlayerOfTeam,
        text,
        formatUsageCount(rows[r].raw(roles[c]), locale, roles[c].is_duration === true),
      ),
  })
}
