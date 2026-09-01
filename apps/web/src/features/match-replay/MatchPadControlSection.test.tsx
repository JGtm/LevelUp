/**
 * Tests — MatchPadControlSection (le contrôle des armes spéciales de la page match).
 *
 * CE QU'ILS PROTÈGENT, et ce sont les quatre promesses de la section :
 *   1. LA DOUBLE PORTE. Sans artefact — la quasi-totalité des matchs en production — la section
 *      ne rend RIEN ; avec un artefact dont aucune occupation n'est attribuée, non plus. Un
 *      cadre vide répété sur chaque page de match est une promesse non tenue à l'infini.
 *   2. LES NOMS D'ARME viennent des tables EXISTANTES du rejeu (catalogue du document, familles
 *      de socle) — jamais une clé brute, jamais une seconde table de noms.
 *   3. LE TOTAL D'ÉQUIPE est la somme des lignes du camp.
 *   4. CE QUI N'EST PAS ATTRIBUÉ SE DIT, avec sa cause : le lecteur doit pouvoir vérifier que
 *      prises affichées + occupations hors tableau = occupations mesurées.
 *
 * Le calcul est éprouvé chez `padControlLogic.test.ts` ; ici on éprouve le RENDU.
 */
import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { REPLAY_TEXT } from './i18n'
import { MatchPadControlSection } from './MatchPadControlSection'
import { testReplayDoc } from './test/testDoc'

// La lecture de l'artefact est la SEULE frontière réseau du composant : on la pilote, et tout
// le reste (agrégation, colonnes, libellés) reste le vrai code. Même patron que
// `MatchEquipmentUsageSection.test.tsx`.
const artefact = vi.hoisted(() => ({ current: undefined as unknown }))
vi.mock('./queries', () => ({ useMatchReplay: () => ({ data: artefact.current }) }))

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

/**
 * La NOTE de bas de tableau qui contient ce fragment. Un matcher de texte brut remonterait
 * jusqu'aux ancêtres (la section entière contient aussi la phrase) : on borne au paragraphe.
 */
function note(vue: ReturnType<typeof render>, fragment: string): HTMLElement {
  const paragraphes = [...vue.container.querySelectorAll('p')]
  const trouve = paragraphes.find((p) => (p.textContent ?? '').includes(fragment))
  if (!trouve) throw new Error(`aucune note ne contient : ${fragment}`)
  return trouve
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

describe('MatchPadControlSection — le tableau', () => {
  it('affiche le titre, la colonne de total et une colonne par socle réellement pris', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByRole('region', { name: t.padControl.title })).toBeTruthy()
    expect(vue.getAllByText(t.padControl.colTotal).length).toBeGreaterThan(0)
    // Le socle de bonus n'a été pris par personne : aucune colonne pour lui.
    expect(vue.queryByText(t.padEquipmentFamily.powerup_overshield)).toBeNull()
  })

  it('nomme le socle par le CATALOGUE du document, jamais par sa clé brute', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByText('S7 Sniper')).toBeTruthy()
    expect(vue.queryByText(SNIPER)).toBeNull()
  })

  it('montre chaque joueur sous son camp, et le total du camp est la somme de ses lignes', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    for (const nom of ['Alpha', 'Bravo', 'Charlie']) expect(vue.getByText(nom)).toBeTruthy()
    const totaux = vue.getAllByText(t.padControl.teamTotal)
    expect(totaux.length).toBe(2)
    // Le camp d'Alpha totalise ses deux prises ; l'autre camp reste à zéro.
    const ligneTotal = totaux[0].closest('tr')
    expect(ligneTotal?.textContent).toContain('2')
  })
})

describe('MatchPadControlSection — ce que l’écran dit de sa mesure', () => {
  it('écrit le dénominateur et ventile les occupations hors tableau par CAUSE', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(note(vue, t.padControl.attributedFmt(2, 4))).toBeTruthy()
    const manques = note(vue, t.padControl.missingFmt(2))
    expect(manques.textContent).toContain(t.padControl.gapFmt.ambiguous(1))
    expect(manques.textContent).toContain(t.padControl.gapFmt.powerup(1))
  })

  it('rend les mêmes nombres en anglais, sans laisser une string française', () => {
    poserArtefact(TEMOIN)
    const vue = afficher('en')
    const en = REPLAY_TEXT.en.padControl
    expect(vue.getByRole('region', { name: en.title })).toBeTruthy()
    expect(note(vue, en.attributedFmt(2, 4))).toBeTruthy()
    expect(vue.queryByText(t.padControl.colTotal)).toBeNull()
  })
})
