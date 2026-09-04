/**
 * Tests — MatchObjectivesSection (le bloc « Objectifs » de la page match).
 *
 * CE QU'ILS PROTÈGENT :
 *   1. LA DOUBLE PORTE. Slayer ne porte aucun bloc d'objectif -> `detectObjectiveMode` rend
 *      null -> RIEN à l'écran. Assaut tombe dans la même porte : l'API ne fournit aucune
 *      statistique de bombe, et la base n'en porte aucune colonne.
 *   2. LES DEUX VUES, et le fait qu'elles suivent LE MODE : les colonnes viennent de
 *      `objectiveColsFor`, pas d'une liste écrite dans le composant. Éprouvé sur trois
 *      grandeurs (Bastion) ET sur cinq (VIP) — la grille doit encaisser les deux.
 *   3. LE FACE-À-FACE N'EXISTE QU'À DEUX CAMPS, et son total d'équipe respecte l'agrégat de
 *      la colonne (cumul, ou maximum d'un « meilleur temps »).
 *
 * La projection est éprouvée chez `objectivesChart.test.ts` ; ici on éprouve le RENDU.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import type { MatchScoreboardRow } from '@/lib/api/types'

import { MATCH_VIEW_TEXT } from './i18n'
import { MatchObjectivesSection } from './MatchObjectivesSection'

const t = MATCH_VIEW_TEXT.fr

/** Une ligne de scoreboard réduite à ce que la section lit. */
function ligne(
  gamertag: string,
  side: string,
  objective: Record<string, number | null> | null,
  isMe = false,
): MatchScoreboardRow {
  return {
    xuid: gamertag,
    gamertag,
    team_side: side,
    is_me: isMe,
    objective,
  } as unknown as MatchScoreboardRow
}

/** Bastion (zones) : trois grandeurs, dont une durée. Deux camps de deux joueurs. */
const BASTION = [
  ligne('Alpha', 't0', { zone_captures: 10, zone_secures: 6, time_in_zones_seconds: 128 }, true),
  ligne('Bravo', 't0', { zone_captures: 9, zone_secures: 1, time_in_zones_seconds: 112 }),
  ligne('Charlie', 't1', { zone_captures: 6, zone_secures: 0, time_in_zones_seconds: 83 }),
  ligne('Delta', 't1', { zone_captures: 4, zone_secures: 0, time_in_zones_seconds: 51 }),
]

function afficher(rows: MatchScoreboardRow[], teams = ['t0', 't1'], myTeamSide: string | null = 't0') {
  return render(
    <MatchObjectivesSection rows={rows} teams={teams} myTeamSide={myTeamSide} t={t} />,
  )
}

describe('MatchObjectivesSection — la double porte', () => {
  it('ne rend RIEN en Slayer : aucune ligne ne porte de bloc d’objectif', () => {
    const slayer = [ligne('Alpha', 't0', null, true), ligne('Charlie', 't1', null)]
    expect(afficher(slayer).container.firstChild).toBeNull()
  })

  it('ne rend rien quand le mode est détecté mais qu’aucune ligne ne le porte', () => {
    expect(afficher([]).container.firstChild).toBeNull()
  })
})

describe('MatchObjectivesSection — les deux vues', () => {
  it('rend la carte et ses deux vues, chacune nommée', () => {
    const vue = afficher(BASTION)
    expect(vue.getByRole('region', { name: t.objectives.title })).toBeTruthy()
    expect(vue.getByRole('region', { name: t.objectives.viewByPlayer })).toBeTruthy()
    expect(vue.getByRole('region', { name: t.objectives.viewTeamTotals })).toBeTruthy()
  })

  it('prend ses colonnes du MODE, avec les libellés de l’i18n', () => {
    const vue = afficher(BASTION)
    for (const key of ['zone_captures', 'zone_secures', 'time_in_zones_seconds']) {
      expect(vue.getAllByText(t.objectives.cols[key].label).length).toBeGreaterThan(0)
    }
    // Aucune grandeur d'un autre mode ne s'invite (« Captures » ne discrimine pas : les zones
    // et le drapeau partagent le mot — « Retours » n'appartient qu'au drapeau).
    expect(vue.queryByText(t.objectives.cols.flag_returns.label)).toBeNull()
  })

  it('rend une ligne par joueur et écrit sa valeur dans le nom accessible de sa barre', () => {
    const vue = afficher(BASTION)
    const tip = t.objectives.gridTipFmt
    const captures = t.objectives.cols.zone_captures.label
    for (const nom of ['Alpha', 'Bravo', 'Charlie', 'Delta']) expect(vue.getByText(nom)).toBeTruthy()
    expect(vue.getByLabelText(tip('Alpha', captures, '10'))).toBeTruthy()
    expect(vue.getByLabelText(tip('Delta', captures, '4'))).toBeTruthy()
  })

  it('formate les durées en m:ss, jamais en secondes brutes', () => {
    const vue = afficher(BASTION)
    const temps = t.objectives.cols.time_in_zones_seconds.label
    expect(vue.getByLabelText(t.objectives.gridTipFmt('Alpha', temps, '2:08'))).toBeTruthy()
  })

  it('oppose les deux camps grandeur par grandeur, avec l’agrégat de la colonne', () => {
    const vue = afficher(BASTION)
    const duel = t.objectives.duelTipFmt
    const captures = t.objectives.cols.zone_captures.label
    expect(vue.getByLabelText(duel('Équipe Eagle', captures, '19'))).toBeTruthy()
    expect(vue.getByLabelText(duel('Équipe Cobra', captures, '10'))).toBeTruthy()
  })

  it('encaisse CINQ grandeurs (VIP) sans changer de forme', () => {
    const vip = [
      ligne(
        'Alpha',
        't0',
        {
          vip_kills: 3,
          times_selected_as_vip: 2,
          kills_as_vip: 4,
          time_as_vip_seconds: 90,
          longest_time_as_vip_seconds: 60,
        },
        true,
      ),
      ligne('Charlie', 't1', {
        vip_kills: 1,
        times_selected_as_vip: 2,
        kills_as_vip: 1,
        time_as_vip_seconds: 45,
        longest_time_as_vip_seconds: 30,
      }),
    ]
    const vue = afficher(vip)
    for (const key of [
      'vip_kills',
      'times_selected_as_vip',
      'kills_as_vip',
      'time_as_vip_seconds',
      'longest_time_as_vip_seconds',
    ]) {
      expect(vue.getAllByText(t.objectives.cols[key].label).length).toBeGreaterThan(0)
    }
    // Le « meilleur temps » est un MAXIMUM de camp, pas une somme.
    expect(
      vue.getByLabelText(
        t.objectives.duelTipFmt(
          'Équipe Eagle',
          t.objectives.cols.longest_time_as_vip_seconds.label,
          '1:00',
        ),
      ),
    ).toBeTruthy()
  })

  it('garde la vue par joueur SEULE quand le lobby ne présente pas deux camps', () => {
    const vue = afficher([BASTION[0], BASTION[1]], ['t0', 't1'], 't0')
    expect(vue.getByRole('region', { name: t.objectives.viewByPlayer })).toBeTruthy()
    expect(vue.queryByRole('region', { name: t.objectives.viewTeamTotals })).toBeNull()
  })
})

describe('MatchObjectivesSection — parité FR/EN', () => {
  it('EN : titre, vues et libellés de colonne passent en anglais', () => {
    const en = MATCH_VIEW_TEXT.en
    const vue = render(
      <MatchObjectivesSection rows={BASTION} teams={['t0', 't1']} myTeamSide="t0" t={en} />,
    )
    expect(vue.getByRole('region', { name: en.objectives.title })).toBeTruthy()
    expect(vue.getByRole('region', { name: en.objectives.viewTeamTotals })).toBeTruthy()
    expect(vue.getAllByText(en.objectives.cols.zone_secures.label).length).toBeGreaterThan(0)
    expect(vue.queryByText(t.objectives.viewByPlayer)).toBeNull()
  })
})
