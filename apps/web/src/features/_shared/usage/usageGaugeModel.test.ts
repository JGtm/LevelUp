/**
 * usageGaugeModel.test.ts — buildGaugeRow, la pile des trois issues et les deux repères de
 * taux (P6, P7). Extrait de `usageLogic.test.ts` le 2026-09-09 (étape E5.1bis, scission de
 * taille — CLAUDE.md n°5) au moment du déménagement du bloc vers `features/_shared/usage/`.
 */
import { describe, expect, it } from 'vitest'

import type { SessionUsageOutcomes } from '@/lib/api/types'

import { buildGaugeRow } from './usageGaugeModel'
import { USAGE_TEXT } from './usageI18n'

const t = USAGE_TEXT.fr

describe('buildGaugeRow — E4.3 : la pile des trois issues et les deux repères de taux (P6, P7)', () => {
  const shares = {
    player_total: 9,
    team_total: 20,
    lobby_total: 43,
    team_share_of_lobby_pct: 45.6,
    player_share_of_team_pct: 20.5,
    player_share_of_lobby_pct: 9.3,
  }

  function rowWithOutcomes(outcomes: SessionUsageOutcomes) {
    return buildGaugeRow({
      key: 'equipment_wall',
      label: 'Mur de protection',
      shares,
      outcomes,
      teamParityPct: 25,
      lobbyParityPct: 12.5,
      teamOfLobbyParityPct: 50,
      t,
      locale: 'fr',
    })
  }

  it('la pile respecte l ordre utilise -> garde -> lache (§3.1, S10), fractions de LA TRANCHE', () => {
    const row = rowWithOutcomes({ used: 6, dropped: 3, kept: 1, taken: 11 })
    const [, joueurEquipe, joueurLobby] = row.gauges
    expect(joueurEquipe.segments?.map((s) => s.key)).toEqual(['used', 'kept', 'dropped'])
    expect(joueurEquipe.segments?.map((s) => s.fraction)).toEqual([0.6, 0.1, 0.3])
    // La jauge « mon equipe dans le lobby » ne porte AUCUN segment : Outcomes est une
    // grandeur du JOUEUR, pas de l equipe agregee.
    expect(row.gauges[0].segments).toBeUndefined()
    // La 3e jauge (joueur/lobby) porte les MEMES segments (meme joueur, meme partition).
    expect(joueurLobby.segments?.map((s) => s.key)).toEqual(['used', 'kept', 'dropped'])
  })

  it('une famille sans troisieme issue (ici : jamais garde) rend DEUX segments', () => {
    const row = rowWithOutcomes({ used: 6, dropped: 4, kept: 0, taken: 10 })
    const [, joueurEquipe] = row.gauges
    expect(joueurEquipe.segments?.map((s) => s.key)).toEqual(['used', 'dropped'])
  })

  it('outcomes absent : segments absents, rendu inchange (E1/E4.1)', () => {
    const row = buildGaugeRow({
      key: 'grapple_pulls',
      label: 'Grappin',
      shares,
      teamParityPct: 25,
      lobbyParityPct: 12.5,
      teamOfLobbyParityPct: 50,
      t,
      locale: 'fr',
    })
    expect(row.gauges[1].segments).toBeUndefined()
    expect(row.gauges[1].teammatesRatePct).toBeNull()
    expect(row.gauges[1].opponentsRatePct).toBeNull()
  })

  it('les deux repere de taux (P7, exclusion du joueur) se projettent tels quels, nil <> 0', () => {
    const row = rowWithOutcomes({
      used: 6,
      dropped: 3,
      kept: 1,
      taken: 11,
      teammates_used_rate_pct: 62,
      opponents_used_rate_pct: 40,
    })
    expect(row.gauges[1].teammatesRatePct).toBe(62)
    expect(row.gauges[1].opponentsRatePct).toBe(40)
  })

  it('reperes absents (scope FFA, camp inconnu) : null, jamais 0 %', () => {
    const row = rowWithOutcomes({ used: 6, dropped: 3, kept: 1, taken: 11 })
    expect(row.gauges[1].teammatesRatePct).toBeNull()
    expect(row.gauges[1].opponentsRatePct).toBeNull()
  })

  it('le compte brut (utilise/garde/lache) reste dans l infobulle (E4.6)', () => {
    const row = rowWithOutcomes({ used: 6, dropped: 3, kept: 1, taken: 11 })
    expect(row.gauges[1].tooltip).toContain('6')
    expect(row.gauges[1].tooltip).toContain('3')
    expect(row.gauges[1].tooltip).toContain('1')
  })
})

describe('buildGaugeRow — jauges de parts et parités', () => {
  const shares = {
    player_total: 9,
    team_total: 20,
    lobby_total: 43,
    team_share_of_lobby_pct: 45.6,
    player_share_of_team_pct: 20.5,
    player_share_of_lobby_pct: 9.3,
  }

  it('rend les trois dénominateurs du §7 avec leurs textes d honnêteté', () => {
    const row = buildGaugeRow({
      key: 'pads',
      label: 'Toutes armes spéciales',
      shares,
      teamParityPct: 25,
      lobbyParityPct: 12.5,
      teamOfLobbyParityPct: 50,
      t,
      locale: 'fr',
    })
    const [camp, joueurEquipe, joueurLobby] = row.gauges
    expect(camp.valueText).toBe('45,6 %')
    expect(camp.honestyText).toBe('20 sur 43')
    expect(joueurEquipe.valueText).toBe('20,5 %')
    expect(joueurEquipe.parityPct).toBe(25)
    expect(joueurLobby.honestyText).toBe('9 sur 43')
    // L ORDRE EST UN CONTRAT DE RENDU : la grille montre la 2e jauge seule quand le
    // repli est fermé (PRIMARY_GAUGE_INDEX, UsageForms). Si cet ordre change,
    // c est « ma part dans le lobby » qui s affiche par défaut, sans que rien ne casse.
    expect(row.gauges.map((g) => g.key)).toEqual([
      'team-of-lobby',
      'player-of-team',
      'player-of-lobby',
    ])
    // Le compte brut vit dans l INFOBULLE, plus dans la cellule (D2).
    expect(joueurEquipe.tooltip).toContain('9 sur 20')
  })

  it('marque un total : la ligne « toutes armes » se sépare de ses familles', () => {
    const row = buildGaugeRow({
      key: 'pads',
      label: 'Toutes armes spéciales',
      shares,
      teamParityPct: 25,
      lobbyParityPct: 12.5,
      teamOfLobbyParityPct: 50,
      isTotal: true,
      t,
      locale: 'fr',
    })
    expect(row.isTotal).toBe(true)
  })

  it('nil ≠ 0 : un scope à camp inconnu rend des jauges vides, jamais 0 %', () => {
    const row = buildGaugeRow({
      key: 'x',
      label: 'X',
      shares: { player_total: 3, lobby_total: 12, player_share_of_lobby_pct: 25 },
      teamParityPct: null,
      lobbyParityPct: 12.5,
      teamOfLobbyParityPct: null,
      t,
      locale: 'fr',
    })
    const [camp, joueurEquipe, joueurLobby] = row.gauges
    expect(camp.valuePct).toBeNull()
    expect(camp.valueText).toBe('—')
    expect(joueurEquipe.valuePct).toBeNull()
    expect(joueurEquipe.honestyText).toBe('3 sur —')
    expect(joueurLobby.valuePct).toBe(25)
  })
})
