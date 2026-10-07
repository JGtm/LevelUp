/**
 * Tests — MatchEquipmentUsageSection (« Usage d'équipements, par joueur », page match).
 *
 * CE QU'ILS PROTÈGENT, et ce sont les promesses de la carte :
 *   1. LA DOUBLE PORTE. Sans artefact — la quasi-totalité des matchs en production — la
 *      carte ne rend RIEN ; avec un artefact qui ne porte aucune grandeur, non plus. Un cadre
 *      vide répété sur chaque page de match est une promesse non tenue à l'infini.
 *   2. UNE SEULE CARTE, la grille par joueur, et les colonnes que LA DONNÉE justifie — jamais une
 *      liste en dur. La part de chaque équipe se lit dans « Contrôle des ressources, par match ».
 *   3. L'ÉQUIPE D'UNE LIGNE SE LIT DANS LE FILM : l'encre de la ligne est celle de l'équipe du
 *      film du joueur de la page, neutre quand le film tait la sienne.
 *   4. AUCUN TEXTE DE PIED : l'aide du TITRE dit ce que la grille compte, puis, quand elle n'est
 *      pas nulle, la réserve (gestes sans propriétaire, poses d'origine inconnue) en une phrase.
 *   5. CE QUI N'EST PAS MESURÉ SE DIT. Aucune colonne pour le répulseur ni le propulseur, et une
 *      grandeur non mesurée écrit « — » là où un zéro se lirait comme une mesure.
 *
 * LES VALEURS SE LISENT PAR LE NOM ACCESSIBLE DES BARRES (`gridTipFmt` : joueur — grandeur :
 * valeur). C'est ce que porte l'écran, et c'est aussi ce qu'entend un lecteur d'écran : l'éprouver
 * ici éprouve les deux d'un coup.
 *
 * Le calcul est éprouvé chez `equipmentUsageLogic.test.ts` et `valueGridModel.test.ts` ; ici on
 * éprouve le RENDU.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { REPLAY_TEXT } from './i18n/i18n'
import { MatchEquipmentUsageSection } from './MatchEquipmentUsageSection'
import { testReplayDoc } from './test/testDoc'

// La lecture de l'artefact est la SEULE frontière réseau du composant : on la pilote, et tout
// le reste (agrégation, colonnes, libellés) reste le vrai code. Même patron que
// `MatchScoreCurveChart.test.tsx`.
const artefact = vi.hoisted(() => ({ current: undefined as unknown }))
vi.mock('../../lib/replay/queries', () => ({ useMatchReplay: () => ({ data: artefact.current }) }))

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
 * LE TÉMOIN DE RENDU : quatre joueurs dans deux camps DU FILM (dont un que le scoreboard ignore,
 * Delta, que le film range avec Charlie), un geste de chaque canal, et deux vidages de socle de
 * bonus. Frames à 100 ms.
 */
const TEMOIN: Partial<ReplayDocument> = {
  frameCount: 200,
  frameIntervalMs: 100,
  roster: [
    { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
    { filmIndex: 1, xuid: 'a2', name: 'Bravo', team: 0 },
    { filmIndex: 2, xuid: 'b1', name: 'Charlie', team: 1 },
    { filmIndex: 3, xuid: 'orphelin', name: 'Delta', team: 1 },
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

function afficher(locale: 'fr' | 'en' = 'fr', scoreboard: MatchScoreboardRow[] = SCOREBOARD) {
  return render(
    <MatchEquipmentUsageSection
      playerSlug="joueur"
      matchId="m1"
      replayAvailable
      scoreboard={scoreboard}
      locale={locale}
    />,
  )
}

/** survolerTitre — ouvre l'infobulle (i) du TITRE de la carte : l'aide, puis la réserve. */
function survolerTitre(vue: ReturnType<typeof afficher>) {
  fireEvent.mouseEnter(vue.getByRole('button', { name: /informations|more info/i }))
}

/** Le texte attendu de l'aide du titre : la mesure, puis la réserve quand elle n'est pas nulle. */
function aide(reserve = 0): string {
  const u = t.equipmentUsage
  return u.infoByPlayer + (reserve > 0 ? u.coverageReserveFmt(reserve) : '')
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

describe('MatchEquipmentUsageSection — la carte', () => {
  it('rend UNE carte, nommée « Usage d’équipements, par joueur »', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getAllByRole('region')).toHaveLength(1)
    expect(vue.getByRole('region', { name: t.equipmentUsage.viewByPlayer })).toBeTruthy()
    expect(vue.queryByText('Part de chaque équipe')).toBeNull()
  })

  it('montre une colonne par canal mesuré : le grappin, puis chaque famille d’équipement', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    for (const colonne of [t.equipmentUsage.groupGrapple, t.placementFamily.field]) {
      expect(vue.getByText(colonne)).toBeTruthy()
    }
  })

  it('nomme les colonnes par les tables EXISTANTES du rejeu, une fois chacune', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Familles de pose : libellés de `placementFamily` (par règle de rendu).
    expect(vue.getByText(t.placementFamily.sensor)).toBeTruthy()
    expect(vue.getAllByText(t.placementFamily.field)).toHaveLength(1)
    // AUCUNE colonne de grenade.
    expect(vue.queryByText('Fragmentation')).toBeNull()
    // Les deux power-ups ont UNE colonne chacun, celle de leur famille d'équipement.
    expect(vue.getAllByText(t.padEquipmentFamily.powerup_camo)).toHaveLength(1)
    expect(vue.getAllByText(t.padEquipmentFamily.powerup_overshield)).toHaveLength(1)
  })

  it('pose la LÉGENDE des issues, et les tractions quand le grappin a sa colonne', () => {
    poserArtefact(TEMOIN)
    const u = t.equipmentUsage
    const vue = afficher()
    for (const libelle of [u.legendUsed, u.legendKept, u.legendDropped, u.legendGrapple]) {
      expect(vue.getByText(libelle)).toBeTruthy()
    }
  })

  it('sans tractions, la légende ne nomme pas le grappin', () => {
    poserArtefact({ ...TEMOIN, grappleLines: [] } as Partial<ReplayDocument>)
    const vue = afficher()
    expect(vue.queryByText(t.equipmentUsage.groupGrapple)).toBeNull()
    expect(vue.queryByText(t.equipmentUsage.legendGrapple)).toBeNull()
    expect(vue.getByText(t.equipmentUsage.legendUsed)).toBeTruthy()
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

  it('l’ÉPISODE d’un power-up alimente le côté « servi » de SA colonne d’équipement', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    const u = t.equipmentUsage
    expect(
      vue.getByLabelText(u.gridTipFmt('Alpha', t.padEquipmentFamily.powerup_camo, '1')),
    ).toBeTruthy()
    // Bravo n'a aucun épisode de camouflage : la mesure a eu lieu et vaut zéro.
    expect(
      vue.getByLabelText(u.gridTipFmt('Bravo', t.padEquipmentFamily.powerup_camo, '0')),
    ).toBeTruthy()
  })

  it('range le joueur HORS SCOREBOARD dans l’équipe que le FILM lui donne, nommée par la feuille', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.getByText('Delta').closest('[title]')?.getAttribute('title')).toBe('Delta — Équipe Cobra')
  })

  it('un joueur dont le film TAIT l’équipe n’a AUCUNE ligne ; ses gestes rejoignent la réserve', () => {
    poserArtefact({
      ...TEMOIN,
      roster: [...(TEMOIN.roster ?? []), { filmIndex: 8, xuid: '', bot: true, name: 'Sandwolf [bot]' }],
      tracks: [...(TEMOIN.tracks ?? []), { ...vie(8, ''), bot: 'Sandwolf [bot]' }],
      grappleLines: [...(TEMOIN.grappleLines ?? []), { slot: 8, t0: 4, t1: 8, ax: 0, ay: 0 }],
    } as unknown as Partial<ReplayDocument>)
    // La feuille joint le bot (côté t0) : elle ne lui donne pas de ligne pour autant.
    const feuille = [...SCOREBOARD, { xuid: 'bid(44.0)', gamertag: 'Sandwolf', team_side: 't0', is_bot: true }] as MatchScoreboardRow[]
    const vue = afficher('fr', feuille)
    expect(vue.queryByText('Sandwolf')).toBeNull()
    expect(vue.queryByText(/Sans équipe|inconnue/)).toBeNull()
    survolerTitre(vue)
    expect(screen.getByRole('tooltip').textContent).toBe(aide(1))
  })
})

describe('MatchEquipmentUsageSection — l’encre des lignes suit le film', () => {
  /** L'encre de la ligne d'un joueur (le filet coloré devant son nom). */
  const encreDe = (vue: ReturnType<typeof afficher>, nom: string) =>
    vue.getByText(nom).closest('[title]')!.querySelector('[aria-hidden="true"]')!.getAttribute('style')

  it('L’ÉQUIPE DU JOUEUR DE LA PAGE SE LIT DANS LE FILM : une feuille qui la dit en face ne change pas l’encre', () => {
    poserArtefact(TEMOIN)
    // La feuille range Alpha (`is_me`) du côté t1 ; le film l'écrit au camp 0.
    const contradictoire = SCOREBOARD.map((r) => (r.xuid === 'a1' ? { ...r, team_side: 't1' } : r))
    const vue = afficher('fr', contradictoire)
    expect(encreDe(vue, 'Alpha')).toContain('team-ally')
    expect(encreDe(vue, 'Charlie')).toContain('team-enemy')
  })

  it('joueur de la page dont le film TAIT l’équipe : aucune ligne n’a d’encre d’équipe', () => {
    poserArtefact({
      ...TEMOIN,
      roster: TEMOIN.roster!.map((e) => (e.xuid === 'a1' ? { ...e, team: undefined } : e)),
    } as Partial<ReplayDocument>)
    const vue = afficher()
    for (const nom of ['Bravo', 'Charlie']) {
      expect(encreDe(vue, nom)).toContain('muted-foreground')
      expect(encreDe(vue, nom)).not.toContain('team-')
    }
  })
})

describe('MatchEquipmentUsageSection — ce que l’écran DIT de sa mesure', () => {
  it('ne pose AUCUN texte de pied sous le bloc (décision utilisateur 2026-09-14)', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Le pied de carte a porté successivement le paragraphe répulseur/propulseur, la ligne des
    // socles de bonus vidés, les dénominateurs de couverture et les deux réserves. Il ne porte
    // plus rien : aucune de ces phrases ne doit revenir sous la grille.
    expect(vue.queryByText(/Socles de bonus|États actifs mesurés|traction.* de grappin lue/)).toBeNull()
    expect(vue.queryByText(/hors de la grille|origine inconnue/)).toBeNull()
  })

  it('dit la RÉSERVE au survol du TITRE, en une phrase', () => {
    poserArtefact({
      ...TEMOIN,
      coverage: {
        ...TEMOIN.coverage,
        placements: { byFamilyOrigin: { 'sensor/deployed': 2, 'wall/unknown': 3 } },
      },
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    survolerTitre(vue)
    expect(screen.getByRole('tooltip').textContent).toBe(aide(3))
  })

  it('sans réserve, l’aide du titre dit seulement ce que la grille compte', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    survolerTitre(vue)
    expect(screen.getByRole('tooltip').textContent).toBe(aide())
  })

  it('ne rend JAMAIS les objets pris sans famille connue (décision utilisateur 2026-09-09)', () => {
    poserArtefact({
      ...TEMOIN,
      // Aucune table `abilityLabels` : le rang 5 n'a aucun nom dans ce film. Il est compté par la
      // logique (`unnamedTaken`, outillage d'investigation) mais aucune ligne ni note ne le montre.
      equipmentChanges: [{ t: 5, slot: 1, kind: 'taken', r: 5, from: -1 }],
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    expect(vue.queryByText(/sans famille connue|without a known family/i)).toBeNull()
  })

  it('n’ouvre AUCUNE colonne pour le répulseur ni le propulseur', () => {
    poserArtefact({
      ...TEMOIN,
      equipmentPlacements: [
        { family: 'repulsor', origin: 'deployed', owner: 1, id: '0x3', t0: 5, t1: 9, x: 0, y: 0 },
        { family: 'thruster', origin: 'dropped', owner: 1, id: '0x4', t0: 5, t1: 9, x: 0, y: 0 },
      ],
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    // Le témoin garde les épisodes camo/surbouclier de TEMOIN (non écrasés) : la colonne
    // « équipement » existe donc toujours pour EUX (E2, décision D9 amendée), mais ni le
    // répulseur ni le propulseur n'y ouvrent de colonne — c'est ce que ce test protège.
    expect(vue.queryByText('repulsor')).toBeNull()
    expect(vue.queryByText('thruster')).toBeNull()
  })

  it('compte à part les gestes mesurés sans propriétaire, hors de la grille', () => {
    poserArtefact({
      ...TEMOIN,
      equipmentPlacements: [
        { family: 'sensor', origin: 'deployed', owner: 1, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
        { family: 'sensor', origin: 'deployed', owner: -1, id: '0x1', t0: 5, t1: 9, x: 0, y: 0 },
      ],
    } as unknown as Partial<ReplayDocument>)
    const vue = afficher()
    survolerTitre(vue)
    expect(screen.getByRole('tooltip').textContent).toBe(aide(1))
  })
})

describe('MatchEquipmentUsageSection — tout est affiché (retrait du repli, 2026-09-19)', () => {
  it('montre d’emblée TOUTES les colonnes que la donnée justifie, sans bouton de repli', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    // Le grappin et le champ de réparation lâché étaient repliés par défaut jusqu'au
    // 2026-09-19 (vote « game changers ») : ils sont là au chargement.
    expect(vue.getAllByText(t.equipmentUsage.groupGrapple).length).toBeGreaterThan(0)
    expect(vue.getAllByText(t.placementFamily.field).length).toBeGreaterThan(0)
    expect(vue.getByText(t.placementFamily.sensor)).toBeTruthy()
    expect(vue.queryByRole('button', { name: /Voir plus|Replier/ })).toBeNull()
  })

  it('les GRENADES n’ont plus aucune colonne (retrait utilisateur du 2026-09-13)', () => {
    poserArtefact(TEMOIN)
    const vue = afficher()
    expect(vue.queryByText('Fragmentation')).toBeNull()
  })
})

describe('MatchEquipmentUsageSection — parité FR/EN', () => {
  it('EN : le titre, l’aide, la légende et les familles passent en anglais', () => {
    poserArtefact(TEMOIN)
    const vue = afficher('en')
    const en = REPLAY_TEXT.en.equipmentUsage
    expect(vue.getByRole('region', { name: en.viewByPlayer })).toBeTruthy()
    expect(vue.getByText(en.groupGrapple)).toBeTruthy()
    expect(vue.getByText(REPLAY_TEXT.en.placementFamily.sensor)).toBeTruthy()
    expect(vue.getAllByText(REPLAY_TEXT.en.padEquipmentFamily.powerup_camo)).toHaveLength(1)
    expect(vue.getByText(en.legendKept)).toBeTruthy()
    survolerTitre(vue)
    expect(screen.getByRole('tooltip').textContent).toBe(en.infoByPlayer)
  })
})
