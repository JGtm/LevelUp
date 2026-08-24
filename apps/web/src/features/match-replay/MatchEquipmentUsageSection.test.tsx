/**
 * Tests — MatchEquipmentUsageSection (le bilan d'équipement de la page match).
 *
 * CE QU'ILS PROTÈGENT, et ce sont les quatre promesses de la section :
 *   1. LA DOUBLE PORTE. Sans artefact — la quasi-totalité des matchs en production — la
 *      section ne rend RIEN ; avec un artefact qui ne porte aucune grandeur, non plus. Un cadre
 *      vide répété sur chaque page de match est une promesse non tenue à l'infini.
 *   2. LE TOTAL D'ÉQUIPE est la somme des lignes du camp, pas celle du match.
 *   3. L'ANONYME RESTE ANONYME : les socles de bonus vidés sont une ligne au niveau du MATCH,
 *      jamais une colonne rattachée à quelqu'un.
 *   4. CE QUI N'EST PAS MESURÉ SE DIT. Aucune colonne pour le répulseur ni le propulseur, et
 *      une phrase à l'écran explique pourquoi — une colonne de zéros se lirait « zéro usage ».
 *
 * Le calcul est éprouvé chez `equipmentUsageLogic.test.ts` ; ici on éprouve le RENDU.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { REPLAY_TEXT } from './i18n'
import { MatchEquipmentUsageSection } from './MatchEquipmentUsageSection'
import { testReplayDoc } from './test/testDoc'

// La lecture de l'artefact est la SEULE frontière réseau du composant : on la pilote, et tout
// le reste (agrégation, colonnes, libellés) reste le vrai code. Même patron que
// `MatchScoreCurveChart.test.tsx`.
const artefact = vi.hoisted(() => ({ current: undefined as unknown }))
vi.mock('./queries', () => ({ useMatchReplay: () => ({ data: artefact.current }) }))

const t = REPLAY_TEXT.fr

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
 * LE TÉMOIN DE RENDU : quatre joueurs (dont un que le scoreboard ignore), un geste de chaque
 * canal, et deux vidages de socle de bonus. Frames à 100 ms.
 */
const TEMOIN: Partial<ReplayDocument> = {
  frameCount: 200,
  frameIntervalMs: 100,
  roster: [
    { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
    { filmIndex: 1, xuid: 'a2', name: 'Bravo' },
    { filmIndex: 2, xuid: 'b1', name: 'Charlie' },
    { filmIndex: 3, xuid: 'orphelin', name: 'Delta' },
  ],
  tracks: [vie(1, 'a1'), vie(2, 'a2'), vie(3, 'b1'), vie(4, 'orphelin')],
  grappleLines: [
    { slot: 1, t0: 1, t1: 5, ax: 0, ay: 0 },
    { slot: 2, t0: 2, t1: 6, ax: 0, ay: 0 },
    { slot: 3, t0: 3, t1: 7, ax: 0, ay: 0 },
  ],
  equipmentEpisodes: [
    { slot: 1, fam: 'camo', t0: 10, t1: 60 },
    { slot: 2, fam: 'overshield', t0: 0, t1: 30 },
  ],
  equipmentPlacements: [
    { family: 'sensor', origin: 'deployed', owner: 1, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
    { family: 'sensor', origin: 'deployed', owner: 2, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
    { family: 'repair_field', origin: 'dropped', owner: 3, id: '0x2', t0: 5, t1: 9, x: 0, y: 0 },
  ],
  grenades: [{ slot: 1, rank: 0, t: 5, i: 0, s: 'x', x: 0, y: 0 }],
  grenadeLabels: [{ fr: 'Fragmentation', en: 'Frag' }],
  weaponPads: [{ weapon: 'powerup_overshield', x: 0, y: 0, spawns: [], presence: [] }],
  padPickups: [
    { pad: 0, tLow: 10, tHigh: 20, xuid: null },
    { pad: 0, tLow: 60, tHigh: 70, xuid: null },
  ],
  coverage: {
    equipment: { tracksTotal: 40, camoLives: 1, camoEpisodes: 1, overshieldLives: 1, overshieldEpisodes: 1 },
    grapple: { pulls: 3, pullLives: 3, lightReads: 4, heavyReads: 3, unpairedFires: 1, brokenBodies: 0 },
    groundWeapons: { powerupPads: 1 },
  },
} as unknown as Partial<ReplayDocument>

function poserArtefact(over: Partial<ReplayDocument> | null) {
  artefact.current = over ? testReplayDoc(over) : undefined
}

function afficher(locale: 'fr' | 'en' = 'fr') {
  return render(
    <MatchEquipmentUsageSection
      playerSlug="joueur"
      matchId="m1"
      replayAvailable
      scoreboard={SCOREBOARD}
      locale={locale}
    />,
  )
}

describe('MatchEquipmentUsageSection — la double porte', () => {
  it('ne rend RIEN sans artefact : 404 = pas de film, pas de cadre vide', () => {
    poserArtefact(null)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand l’artefact existe mais ne porte AUCUNE grandeur mesurée', () => {
    poserArtefact({ tracks: [vie(1, 'a1')] } as Partial<ReplayDocument>)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand le film ne porte que des gestes SANS propriétaire', () => {
    poserArtefact({
      tracks: [{ slot: 9, team: -1, startFrame: 0, endFrame: 10, points: [{ t: 0, x: 0, y: 0 }] }],
      grappleLines: [{ slot: 9, t0: 1, t1: 5, ax: 0, ay: 0 }],
    } as unknown as Partial<ReplayDocument>)
    expect(afficher().container.firstChild).toBeNull()
  })
})

describe('MatchEquipmentUsageSection — le tableau', () => {
  it('affiche le titre et un groupe de colonnes par canal mesuré', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByRole('region', { name: t.equipmentUsage.title })).toBeTruthy()
    for (const groupe of [
      t.equipmentUsage.groupGrapple,
      t.equipmentUsage.groupActive,
      t.equipmentUsage.groupDeployed,
      t.equipmentUsage.groupDropped,
      t.equipmentUsage.groupGrenades,
    ]) {
      expect(vue.getAllByText(groupe).length).toBeGreaterThan(0)
    }
  })

  it('nomme les colonnes par les tables EXISTANTES du rejeu, jamais un nom en dur', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Familles de pose : libellés de `placementFamily` (par règle de rendu).
    expect(vue.getByText(t.placementFamily.sensor)).toBeTruthy()
    expect(vue.getByText(t.placementFamily.field)).toBeTruthy()
    // Type de grenade : le catalogue bilingue du DOCUMENT.
    expect(vue.getByText('Fragmentation')).toBeTruthy()
    // États actifs : nombre ET durée cumulée.
    expect(vue.getByText(`Camouflage (${t.equipmentUsage.activeCount})`)).toBeTruthy()
    expect(vue.getByText(`Camouflage (${t.equipmentUsage.activeDuration})`)).toBeTruthy()
  })

  it('rend une ligne par joueur, chaque camp avec son TOTAL', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    for (const nom of ['Alpha', 'Bravo', 'Charlie', 'Delta']) {
      expect(vue.getByText(nom)).toBeTruthy()
    }
    // Trois camps rendus : t0, t1, et les joueurs sans ligne de scoreboard.
    expect(vue.getAllByText(t.equipmentUsage.teamTotal)).toHaveLength(3)
  })

  it('le TOTAL d’un camp est la somme de SES lignes, pas celle du match', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const ligneTotal = vue.getAllByText(t.equipmentUsage.teamTotal)[0].closest('tr')
    // Camp t0 = Alpha + Bravo : 2 tractions de grappin (le match en compte 3).
    expect(ligneTotal?.querySelectorAll('td')[1]?.textContent).toBe('2')
  })

  it('affiche la durée cumulée en m:ss, et « — » quand la famille n’a aucun épisode', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const ligneAlpha = vue.getByText('Alpha').closest('tr')
    const cellules = [...(ligneAlpha?.querySelectorAll('td') ?? [])].map((c) => c.textContent)
    // nom | grappin 1 | camo 1 épisode | camo 0:05 | surbouclier 0 | surbouclier — | ...
    expect(cellules.slice(0, 6)).toEqual(['Alpha', '1', '1', '0:05', '0', '—'])
  })

  it('range le joueur HORS SCOREBOARD sous « sans équipe », jamais dans un camp nommé', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByText(t.teamUnknown)).toBeTruthy()
    const corps = vue.getByText('Delta').closest('tbody')
    expect(corps?.textContent).toContain(t.teamUnknown)
    expect(corps?.textContent).not.toContain('Alpha')
  })
})

describe('MatchEquipmentUsageSection — ce que l’écran DIT de sa mesure', () => {
  it('pose les socles de bonus vidés HORS du tableau, avec leur dénominateur', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const ligne = vue.getByText(t.equipmentUsage.powerupPads).closest('p')
    expect(ligne?.textContent).toContain('2')
    expect(ligne?.textContent).toContain(t.padEquipmentFamily.powerup_overshield)
    expect(ligne?.textContent).toContain(t.equipmentUsage.powerupPadsDenomFmt(1))
    // ET IL RESTE HORS DU TABLEAU : aucune cellule ne le porte.
    expect(ligne?.closest('table')).toBeNull()
  })

  it('n’affiche aucune ligne de socles quand le film n’en a vidé aucun', () => {
    poserArtefact({ ...TEMOIN, padPickups: [] } as Partial<ReplayDocument>)
    expect(afficher().queryByText(t.equipmentUsage.powerupPads)).toBeNull()
  })

  it('reprend les dénominateurs de couverture du document', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByText(t.equipmentUsage.coverageActiveFmt(40))).toBeTruthy()
    expect(vue.getByText(t.equipmentUsage.coverageGrappleFmt(3, 3))).toBeTruthy()
  })

  it('dit que le répulseur et le propulseur ne sont pas mesurés, SANS colonne vide', () => {
    poserArtefact({
      ...TEMOIN,
      equipmentPlacements: [
        { family: 'repulsor', origin: 'deployed', owner: 1, id: '0x3', t0: 5, t1: 9, x: 0, y: 0 },
        { family: 'thruster', origin: 'dropped', owner: 1, id: '0x4', t0: 5, t1: 9, x: 0, y: 0 },
      ],
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    expect(vue.getByText(t.equipmentUsage.notMeasured)).toBeTruthy()
    expect(vue.queryByText(t.equipmentUsage.groupDeployed)).toBeNull()
    expect(vue.queryByText(t.equipmentUsage.groupDropped)).toBeNull()
    expect(vue.queryByText('repulsor')).toBeNull()
    expect(vue.queryByText('thruster')).toBeNull()
  })

  it('porte la RÉSERVE des états actifs en infobulle d’en-tête (source non distinguée)', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    fireEvent.mouseEnter(vue.getByText(t.equipmentUsage.groupActive).parentElement as Element)
    expect(screen.getByRole('tooltip').textContent).toBe(t.equipmentUsage.groupActiveHint)
  })

  it('compte à part les gestes mesurés sans propriétaire, hors du tableau', () => {
    poserArtefact({
      ...TEMOIN,
      equipmentPlacements: [
        { family: 'sensor', origin: 'deployed', owner: 1, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
        { family: 'sensor', origin: 'deployed', owner: -1, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
      ],
    } as unknown as Partial<ReplayDocument>)
    expect(afficher().getByText(REPLAY_TEXT.fr.equipmentUsage.unattributedFmt(1))).toBeTruthy()
  })
})

describe('MatchEquipmentUsageSection — parité FR/EN', () => {
  it('EN : titre, groupes et libellé de total passent en anglais', () => {
    poserArtefact(TEMOIN)
    const vue = afficher('en')
    const en = REPLAY_TEXT.en
    expect(vue.getByRole('region', { name: en.equipmentUsage.title })).toBeTruthy()
    expect(vue.getAllByText(en.equipmentUsage.groupDeployed).length).toBeGreaterThan(0)
    expect(vue.getAllByText(en.equipmentUsage.teamTotal)).toHaveLength(3)
    // Le catalogue du document est bilingue : le type de grenade suit la langue.
    expect(vue.getByText('Frag')).toBeTruthy()
    expect(vue.getByText(en.placementFamily.sensor)).toBeTruthy()
  })
})
