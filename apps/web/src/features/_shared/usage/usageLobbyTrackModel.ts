/**
 * usageLobbyTrackModel.ts — LA FORME « piste du lobby » du bloc « usages d'équipement, armes
 * spéciales et objectifs » : un segment coloré = nous (découpé par joueur), hachuré = eux.
 *
 * Extrait de `session-detail/usageLogic.ts` le 2026-09-09 (étape E5.1bis, scission de taille —
 * CLAUDE.md n°5) au moment du déménagement du bloc vers `features/_shared/usage/` (étape E5.1).
 *
 * La référence — l'équipe d'en face — n'apparaît dans aucun modèle : elle n'existe que comme
 * complément du dénominateur et comme segment HACHURÉ anonyme.
 *
 * Pur : aucun React, aucune couleur en dur.
 */
import type { SessionUsageSquadPlayer, SessionUsageSquadShare } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { formatUsageCount, formatUsagePct } from './usageFormat'
import type { UsageText } from './usageI18n'
import type { UsageSharesLike } from './usageGaugeModel'

/** Un segment de la piste : coloré = nous (découpé par joueur), hachuré = eux. */
export interface UsageTrackSegment {
  key: string
  label: string
  kind: 'me' | 'squad' | 'team-rest' | 'enemy'
  /** Index 0..2 dans SquadPlayers — décide le jeton squad-player-1..3. */
  squadIndex?: number
  count: number
  pctText: string
  tooltip: string
}

export interface UsageTrackInput {
  meLabel: string
  shares: UsageSharesLike
  squadPlayers: SessionUsageSquadPlayer[]
  squadShares: SessionUsageSquadShare[] | null | undefined
  t: UsageText
  locale: Locale
}

/**
 * buildLobbyTrack — la piste 100 % du lobby : moi, chaque coéquipier suivi, le reste
 * de mon équipe (colorés), et EUX en un seul segment hachuré, anonyme — ni nom, ni
 * couleur d'équipe (doctrine §1 : la référence n'est jamais affichée).
 *
 * `team_total` absent (aucun match du scope à camp connu) → PAS de piste : sans la
 * frontière nous/eux, le découpage mentirait. Les résidus (reste d'équipe, camp
 * adverse) sont bornés à zéro : les scopes joueur (tout le scope) et équipe (matchs
 * à camp connu) peuvent différer d'une poignée d'unités sur une session mixte.
 */
export function buildLobbyTrack(input: UsageTrackInput): UsageTrackSegment[] | null {
  const { shares, t, locale } = input
  if (shares.team_total == null || shares.lobby_total <= 0) return null

  const bySquadXuid = new Map((input.squadShares ?? []).map((s) => [s.xuid, s.total]))
  const squadCounts = input.squadPlayers.map((p) => bySquadXuid.get(p.xuid) ?? 0)
  const squadSum = squadCounts.reduce((a, b) => a + b, 0)
  const teamRest = Math.max(0, shares.team_total - shares.player_total - squadSum)
  const enemy = Math.max(0, shares.lobby_total - shares.team_total)

  const raw: Array<Omit<UsageTrackSegment, 'pctText' | 'tooltip'>> = [
    { key: 'me', label: input.meLabel, kind: 'me', count: shares.player_total },
    ...input.squadPlayers.map((p, i) => ({
      key: `squad-${p.xuid}`,
      label: p.gamertag,
      kind: 'squad' as const,
      squadIndex: i,
      count: squadCounts[i],
    })),
    { key: 'team-rest', label: t.segTeamRest, kind: 'team-rest', count: teamRest },
    { key: 'enemy', label: t.segEnemy, kind: 'enemy', count: enemy },
  ]
  const total = raw.reduce((a, s) => a + s.count, 0)
  if (total <= 0) return null

  return raw
    .filter((s) => s.count > 0)
    .map((s) => {
      const pctText = formatUsagePct((s.count / total) * 100, locale)
      return {
        ...s,
        pctText,
        tooltip: t.trackTipFmt(s.label, formatUsageCount(s.count, locale), pctText),
      }
    })
}
