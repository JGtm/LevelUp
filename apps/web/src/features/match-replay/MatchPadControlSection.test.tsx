/**
 * Tests — MatchPadControlSection (le contrôle des armes spéciales de la page match).
 *
 * CE QU'ILS PROTÈGENT, et ce sont les cinq promesses de la section :
 *   1. LA DOUBLE PORTE. Sans artefact — la quasi-totalité des matchs en production — la section
 *      ne rend RIEN ; avec un artefact dont aucune occupation n'est attribuée, non plus. Un
 *      cadre vide répété sur chaque page de match est une promesse non tenue à l'infini.
 *   2. LES NOMS D'ARME viennent des tables EXISTANTES du rejeu (catalogue du document, familles
 *      de socle) — jamais une clé brute, jamais une seconde table de noms.
 *   3. UNE ARME, UNE BARRE (2026-09-13) : un rail par socle valant 100 % de ses occupations
 *      nommées, le camp du joueur de la page en tête, le total de la ligne à côté du nom.
 *   4. CE QUI N'A PAS DE RAMASSEUR NOMMÉ N'EST VERSÉ À PERSONNE : il est annoté à droite de sa
 *      ligne, hors de la barre.
 *   5. TOUTES LES ARMES SONT VISIBLES AU CHARGEMENT : le vote « game changers » ordonne les
 *      lignes, il n'en cache plus aucune (décision D3 révoquée par l'utilisateur le 13/09).
 *
 * Le calcul est éprouvé chez `padControlLogic.test.ts` ; ici on éprouve le RENDU.
 */
import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { REPLAY_TEXT } from './i18n/i18n'
import { MatchPadControlSection } from './MatchPadControlSection'
import { testReplayDoc } from './test/testDoc'

// La lecture de l'artefact est la SEULE frontière réseau du composant : on la pilote, et tout
// le reste (agrégation, colonnes, libellés) reste le vrai code. Même patron que
// `MatchEquipmentUsageSection.test.tsx`.
const artefact = vi.hoisted(() => ({ current: undefined as unknown }))
vi.mock('../../lib/replay/queries', () => ({ useMatchReplay: () => ({ data: artefact.current }) }))

const t = REPLAY_TEXT.fr
const SNIPER = '0xAAAA1111'

const SCOREBOARD = [
  { xuid: 'a1', gamertag: 'Alpha', team_side: 't0', is_me: true },
  { xuid: 'a2', gamertag: 'Bravo', team_side: 't0' },
  { xuid: 'b1', gamertag: 'Charlie', team_side: 't1' },
] as unknown as MatchScoreboardRow[]

/** Une vie du film : le slot, son propriétaire, et de quoi donner une fenêtre. */
function vie(slot: number, xuid: string) {
  return {
    slot,
    xuid,
    team: -1,
    startFrame: 0,
    endFrame: 100,
    points: [
      { t: 0, x: 0, y: 0 },
      { t: 100, x: 1, y: 1 },
    ],
  }
}

/**
 * LE TÉMOIN DE RENDU : trois joueurs, un socle d'arme nommé au catalogue, un socle de bonus, et
 * quatre occupations dont deux seulement portent un ramasseur.
 */
const TEMOIN: Partial<ReplayDocument> = {
  frameCount: 200,
  frameIntervalMs: 100,
  roster: [
    { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
    { filmIndex: 1, xuid: 'a2', name: 'Bravo' },
    { filmIndex: 2, xuid: 'b1', name: 'Charlie' },
  ],
  tracks: [vie(1, 'a1'), vie(2, 'a2'), vie(3, 'b1')],
  weaponLabels: { [SNIPER]: { fr: 'S7 Sniper', en: 'S7 Sniper', key: 'hinf_s7_sniper' } },
  weaponPads: [
    { weapon: SNIPER, x: 0, y: 0, spawns: [], presence: [] },
    { weapon: 'powerup_overshield', x: 1, y: 1, spawns: [], presence: [] },
  ],
  padPickups: [
    { pad: 0, t: 10, tLow: 5, tHigh: 15, xuid: 'a1' },
    { pad: 0, t: 40, tLow: 35, tHigh: 45, xuid: 'a1' },
    { pad: 0, tLow: 60, tHigh: 70, xuid: null },
    { pad: 1, tLow: 80, tHigh: 90, xuid: null },
  ],
  coverage: {
    padDating: {
      occupations: 4,
      dated: 2,
      named: 2,
      ambiguous: 1,
      uncovered: 0,
      powerupOccupations: 1,
    },
  },
} as unknown as Partial<ReplayDocument>

function poserArtefact(over: Partial<ReplayDocument> | null) {
  artefact.current = over ? testReplayDoc(over) : undefined
}

function afficher(locale: 'fr' | 'en' = 'fr') {
  return render(
    <MatchPadControlSection
      playerSlug="joueur"
      matchId="m1"
      replayAvailable
      scoreboard={SCOREBOARD}
      locale={locale}
    />,
  )
}

describe('MatchPadControlSection — la double porte', () => {
  it('ne rend RIEN sans artefact : 404 = pas de film, pas de cadre vide', () => {
    poserArtefact(null)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand le film ne porte AUCUNE occupation de socle', () => {
    poserArtefact({ tracks: [vie(1, 'a1')] } as Partial<ReplayDocument>)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand des socles se sont vidés mais qu’AUCUNE prise n’est attribuée', () => {
    poserArtefact({
      ...TEMOIN,
      padPickups: [
        { pad: 0, tLow: 5, tHigh: 15, xuid: null },
        { pad: 1, tLow: 80, tHigh: 90, xuid: null },
      ],
    } as unknown as Partial<ReplayDocument>)
    expect(afficher().container.firstChild).toBeNull()
  })
})

describe('MatchPadControlSection — le graphe', () => {
  it('affiche le titre et une ligne par socle réellement pris', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByRole('region', { name: t.padControl.title })).toBeTruthy()
    // Le socle de bonus n'a été pris par personne : aucune ligne pour lui.
    expect(vue.queryByText(t.padEquipmentFamily.powerup_overshield)).toBeNull()
  })

  it('nomme le socle par le CATALOGUE du document, jamais par sa clé brute', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByText('S7 Sniper')).toBeTruthy()
    expect(vue.queryByText(SNIPER)).toBeNull()
  })

  it('nomme chaque segment par son joueur, son camp, son socle et ses prises', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const segment = vue.getByLabelText(
      t.padControl.barTipFmt('Alpha', 'Équipe Eagle', 'S7 Sniper', 2),
    )
    // Deux prises nommées sur deux : le segment remplit le rail et porte son nombre.
    expect(segment.textContent).toBe('2')
  })

  it('pose le camp du joueur de la page EN HAUT, l’adverse en dessous', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const legende = vue.getByTestId('chart-legend')
    expect([...legende.querySelectorAll('li')].map((li) => li.textContent)).toEqual([
      'Équipe Eagle',
      'Équipe Cobra',
    ])
  })

  it('écrit le TOTAL nommé du socle à côté de son nom : le dénominateur du rail', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Deux prises nommées sur le socle S7 — la troisième occupation n'a pas de ramasseur.
    const ligne = vue.getByText('S7 Sniper').parentElement
    expect(ligne?.textContent).toContain('2')
  })

  it('annote à droite les occupations SANS ramasseur nommé, hors des bâtons', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Trois occupations du socle S7, deux nommées : la troisième s'affiche à part.
    const annotation = vue.getByText(t.padControl.unnamedFmt(1))
    expect(annotation.querySelector('[role="img"]')).toBeNull()
  })
})

describe('MatchPadControlSection — le repli « game changers » (plan 2026-09-05)', () => {
  const BR = '0xDDDD4444'

  /**
   * LE TÉMOIN MIXTE : le Sniper (élu) et un BR dont le label n'a PAS de clé canonique —
   * replié par la dégradation D6. Trois prises attribuées sur cinq occupations mesurées.
   */
  const TEMOIN_MIXTE: Partial<ReplayDocument> = {
    ...TEMOIN,
    weaponLabels: {
      [SNIPER]: { fr: 'S7 Sniper', en: 'S7 Sniper', key: 'hinf_s7_sniper' },
      [BR]: { fr: 'BR75', en: 'BR75' },
    },
    weaponPads: [
      { weapon: SNIPER, x: 0, y: 0, spawns: [], presence: [] },
      { weapon: 'powerup_overshield', x: 1, y: 1, spawns: [], presence: [] },
      { weapon: BR, x: 2, y: 2, spawns: [], presence: [] },
    ],
    padPickups: [
      { pad: 0, t: 10, tLow: 5, tHigh: 15, xuid: 'a1' },
      { pad: 0, t: 40, tLow: 35, tHigh: 45, xuid: 'a1' },
      { pad: 2, t: 50, tLow: 45, tHigh: 55, xuid: 'a2' },
      { pad: 0, tLow: 60, tHigh: 70, xuid: null },
      { pad: 1, tLow: 80, tHigh: 90, xuid: null },
    ],
    coverage: {
      padDating: {
        occupations: 5,
        dated: 3,
        named: 3,
        ambiguous: 1,
        uncovered: 0,
        powerupOccupations: 1,
      },
    },
  } as unknown as Partial<ReplayDocument>

  it('AFFICHE TOUTES LES ARMES au chargement : le vote ordonne, il ne cache plus rien', () => {
    poserArtefact(TEMOIN_MIXTE)
    const vue = afficher()
    expect(vue.getByText('S7 Sniper')).toBeTruthy()
    expect(vue.getByText('BR75')).toBeTruthy()
    expect(vue.queryByRole('button', { name: /Voir plus/ })).toBeNull()
    expect(vue.queryByRole('button', { name: t.collapsedColumnsHide })).toBeNull()
  })

  it('le TOTAL ne ment pas : la prise du socle non élu est bien à sa ligne', () => {
    poserArtefact(TEMOIN_MIXTE)
    const vue = afficher()
    expect(
      vue.getByLabelText(t.padControl.barTipFmt('Bravo', 'Équipe Eagle', 'BR75', 1)),
    ).toBeTruthy()
  })
})

describe('MatchPadControlSection — ce que l’écran dit de sa mesure', () => {
  // LE PIED DE CARTE (« N prises attribuées sur N occupations… ») A ÉTÉ RETIRÉ le 2026-09-13
  // sur demande de l'utilisateur : l'annotation de ligne est le seul aveu qui reste.
  it('n’écrit AUCUN pied de carte : rien sous le graphe et sa légende', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.container.textContent).not.toContain('attribuée')
    expect(vue.container.textContent).not.toContain('occupation')
  })

  it('rend les mêmes libellés en anglais, sans laisser une string française', () => {
    poserArtefact(TEMOIN)
    const vue = afficher('en')
    const en = REPLAY_TEXT.en.padControl
    expect(vue.getByRole('region', { name: en.title })).toBeTruthy()
    expect(vue.getByText(en.unnamedFmt(1))).toBeTruthy()
    expect(vue.queryByText(t.padControl.unnamedFmt(1))).toBeNull()
  })
})
