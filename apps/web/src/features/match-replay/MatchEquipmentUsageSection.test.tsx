/**
 * Tests — MatchEquipmentUsageSection (le bilan d'équipement de la page match).
 *
 * CE QU'ILS PROTÈGENT, et ce sont les cinq promesses de la section :
 *   1. LA DOUBLE PORTE. Sans artefact — la quasi-totalité des matchs en production — la
 *      section ne rend RIEN ; avec un artefact qui ne porte aucune grandeur, non plus. Un cadre
 *      vide répété sur chaque page de match est une promesse non tenue à l'infini.
 *   2. LES DEUX VUES, et les colonnes que LA DONNÉE justifie — jamais une liste en dur.
 *   3. LA PART D'UNE ÉQUIPE est la somme des gestes de ses joueurs, pas celle du match, et le
 *      COMPTE BRUT est écrit à côté du pourcentage.
 *   4. L'ANONYME RESTE ANONYME : les socles de bonus vidés sont une ligne au niveau du MATCH,
 *      jamais une colonne rattachée à quelqu'un.
 *   5. CE QUI N'EST PAS MESURÉ SE DIT. Aucune colonne pour le répulseur ni le propulseur, et
 *      une phrase à l'écran explique pourquoi — mais pas la même raison pour les deux depuis le
 *      2026-09-03 : le répulseur n'a AUCUN canal (une colonne de zéros se lirait « zéro usage »),
 *      le propulseur en a un (schéma 38) mais son geste se lit sur la CARTE, pas ici. Et une
 *      grandeur non mesurée écrit « — » là où un zéro se lirait comme une mesure.
 *
 * LES VALEURS SE LISENT PAR LE NOM ACCESSIBLE DES BARRES (`gridTipFmt` : joueur — grandeur :
 * valeur). C'est ce que porte l'écran depuis que la section est un graphe, et c'est aussi ce
 * qu'entend un lecteur d'écran : l'éprouver ici éprouve les deux d'un coup.
 *
 * Le calcul est éprouvé chez `equipmentUsageLogic.test.ts` et `valueGridModel.test.ts` ; ici on
 * éprouve le RENDU.
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

describe('MatchEquipmentUsageSection — les deux vues', () => {
  it('rend la carte et ses deux vues, chacune nommée', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByRole('region', { name: t.equipmentUsage.title })).toBeTruthy()
    expect(vue.getByRole('region', { name: t.equipmentUsage.viewByPlayer })).toBeTruthy()
    expect(vue.getByRole('region', { name: t.equipmentUsage.viewTeamShare })).toBeTruthy()
  })

  it('montre une FAMILLE par canal mesuré, en légende comme en part d’équipe', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    for (const famille of [
      t.equipmentUsage.groupGrapple,
      t.equipmentUsage.groupActive,
      t.equipmentUsage.groupDeployed,
      t.equipmentUsage.groupDropped,
      t.equipmentUsage.groupGrenades,
    ]) {
      expect(vue.getAllByText(famille).length).toBeGreaterThan(0)
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
    // Frags sous effet actif (LOT F.2) : en-tête complet, une colonne par famille mesurée.
    expect(vue.getByText(t.equipmentUsage.activeKillsFamily.camo)).toBeTruthy()
    expect(vue.getByText(t.equipmentUsage.activeKillsFamily.overshield)).toBeTruthy()
  })

  it('rend une ligne par joueur, y compris celui que le scoreboard ignore', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    for (const nom of ['Alpha', 'Bravo', 'Charlie', 'Delta']) {
      expect(vue.getByText(nom)).toBeTruthy()
    }
  })

  it('écrit la valeur de chaque barre dans son nom accessible : joueur, grandeur, valeur', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const tip = t.equipmentUsage.gridTipFmt
    expect(vue.getByLabelText(tip('Alpha', t.equipmentUsage.groupGrapple, '1'))).toBeTruthy()
    expect(vue.getByLabelText(tip('Delta', t.equipmentUsage.groupGrapple, '0'))).toBeTruthy()
  })

  it('affiche la durée cumulée en m:ss, et « 0:00 » quand la mesure vaut zéro', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const duree = `Camouflage (${t.equipmentUsage.activeDuration})`
    const tip = t.equipmentUsage.gridTipFmt
    expect(vue.getByLabelText(tip('Alpha', duree, '0:05'))).toBeTruthy()
    // Bravo n'a aucun épisode de camouflage : la mesure a eu lieu et vaut zéro.
    expect(vue.getByLabelText(tip('Bravo', duree, '0:00'))).toBeTruthy()
  })

  it('un épisode MESURÉ de durée nulle s’écrit « 0:00 », pas « — »', () => {
    // t1 === t0 : l'épisode a bien été observé, sa durée vaut zéro. Le nombre d'épisodes
    // dit « 1 » ; la durée doit dire « 0:00 » et non un repli d'absence.
    poserArtefact({
      ...TEMOIN,
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 40, t1: 40 },
        { slot: 2, fam: 'overshield', t0: 0, t1: 30 },
      ],
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    const tip = t.equipmentUsage.gridTipFmt
    expect(
      vue.getByLabelText(tip('Alpha', `Camouflage (${t.equipmentUsage.activeCount})`, '1')),
    ).toBeTruthy()
    expect(
      vue.getByLabelText(tip('Alpha', `Camouflage (${t.equipmentUsage.activeDuration})`, '0:00')),
    ).toBeTruthy()
  })

  it('range le joueur HORS SCOREBOARD sous « équipe inconnue », jamais dans un camp nommé', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Son nom porte son camp en infobulle de ligne — et ce camp est l'inconnu.
    expect(vue.getByText('Delta').closest('[title]')?.getAttribute('title')).toContain(
      t.teamUnknown,
    )
  })
})

describe('MatchEquipmentUsageSection — la part de chaque équipe', () => {
  it('somme les gestes du CAMP, pas ceux du match, et écrit le compte brut avec la part', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Grappin : Alpha + Bravo = 2 tractions pour le camp t0, sur les 3 du match.
    const segment = vue.getByLabelText(
      t.equipmentUsage.shareTipFmt('Équipe Eagle', t.equipmentUsage.groupGrapple, 2, 3),
    )
    expect(segment.textContent).toBe('2 · 67 %')
    expect(
      vue.getByLabelText(
        t.equipmentUsage.shareTipFmt('Équipe Cobra', t.equipmentUsage.groupGrapple, 1, 3),
      ).textContent,
    ).toBe('1 · 33 %')
  })

  it('ne rend aucune ligne pour une famille qu’aucun camp n’a employée', () => {
    poserArtefact({ ...TEMOIN, grappleLines: [] } as Partial<ReplayDocument>)
    const vue = afficher()
    // Plus de tractions : ni colonne, ni ligne de part, ni entrée de légende.
    expect(vue.queryByText(t.equipmentUsage.groupGrapple)).toBeNull()
  })

  it('porte la RÉSERVE d’une famille en infobulle de son NOM, dans la vue des parts', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Le nom de la famille apparaît en légende de la vue 1 PUIS en tête de sa ligne de part :
    // c'est cette dernière occurrence qui porte la réserve de mesure.
    const noms = vue.getAllByText(t.equipmentUsage.groupActive)
    fireEvent.mouseEnter(noms[noms.length - 1].parentElement as Element)
    expect(screen.getByRole('tooltip').textContent).toBe(t.equipmentUsage.groupActiveHint)
  })
})

describe('MatchEquipmentUsageSection — ce que l’écran DIT de sa mesure', () => {
  it('pose les socles de bonus vidés HORS des vues, avec leur dénominateur', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const ligne = vue.getByText(t.equipmentUsage.powerupPads).closest('p')
    expect(ligne?.textContent).toContain('2')
    expect(ligne?.textContent).toContain(t.padEquipmentFamily.powerup_overshield)
    expect(ligne?.textContent).toContain(t.equipmentUsage.powerupPadsDenomFmt(1))
    // ET IL RESTE HORS DES GRAPHES : aucune barre ne le porte.
    expect(ligne?.querySelector('[role="img"]')).toBeNull()
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

  it('n’ouvre AUCUNE colonne pour le répulseur ni le propulseur, et dit pourquoi', () => {
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

  it('compte à part les gestes mesurés sans propriétaire, hors des vues', () => {
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
  it('EN : titre, vues et familles passent en anglais', () => {
    poserArtefact(TEMOIN)
    const vue = afficher('en')
    const en = REPLAY_TEXT.en
    expect(vue.getByRole('region', { name: en.equipmentUsage.title })).toBeTruthy()
    expect(vue.getByRole('region', { name: en.equipmentUsage.viewByPlayer })).toBeTruthy()
    expect(vue.getAllByText(en.equipmentUsage.groupDeployed).length).toBeGreaterThan(0)
    // Le catalogue du document est bilingue : le type de grenade suit la langue.
    expect(vue.getByText('Frag')).toBeTruthy()
    expect(vue.getByText(en.placementFamily.sensor)).toBeTruthy()
    // Frags sous effet actif (LOT F.2) : en-tête EN, pas la clé FR.
    expect(vue.getByText(en.equipmentUsage.activeKillsFamily.camo)).toBeTruthy()
  })
})

describe('MatchEquipmentUsageSection — frags sous effet actif (LOT F.2)', () => {
  it('écrit « — », jamais 0, quand la jointure n’a pas pu être tentée (killsRead faux)', () => {
    // TEMOIN ne pose pas `coverage.equipment.killsRead` : la jointure est réputée non tentée.
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(
      vue.getByLabelText(
        t.equipmentUsage.gridTipFmt('Alpha', t.equipmentUsage.activeKillsFamily.camo, '—'),
      ),
    ).toBeTruthy()
  })

  it('écrit le compte réel, kills sommés sur les épisodes de la famille, quand killsRead est vrai', () => {
    poserArtefact({
      ...TEMOIN,
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 60, k: 2 },
        { slot: 2, fam: 'overshield', t0: 0, t1: 30, k: 0 },
      ],
      coverage: {
        equipment: {
          tracksTotal: 40,
          camoLives: 1,
          camoEpisodes: 1,
          overshieldLives: 1,
          overshieldEpisodes: 1,
          killsRead: true,
        },
      },
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    expect(
      vue.getByLabelText(
        t.equipmentUsage.gridTipFmt('Alpha', t.equipmentUsage.activeKillsFamily.camo, '2'),
      ),
    ).toBeTruthy()
  })
})
