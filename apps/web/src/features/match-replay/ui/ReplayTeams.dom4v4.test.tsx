/**
 * ReplayTeams.dom4v4.test.tsx — LA FIXATION DU DOM 4v4 : nœud pour nœud, attribut pour
 * attribut, la colonne d'un match qui n'est PAS une Grande équipe ne change pas.
 *
 * POURQUOI CE TEST EXISTE (plan fiches compactes 2026-09-06, item 0.5). Le lot ajoute un
 * gabarit compact pour les matchs `BTB` et promet « aucune modification d'un match qui n'est
 * pas BTB ». Avant lui, RIEN ne protégeait cette promesse : aucun test de la feature n'asserte
 * sur le conteneur des sièges, aucun snapshot n'existe — une extraction qui déplacerait une
 * classe ou un `title` passait tous les tests. La fixture `__fixtures__/replayTeams.4v4.html`
 * a été prise AVANT toute modification de code (premier commit de la branche, seul) : c'est
 * l'état de référence, et toute divergence est une régression 4v4, jamais une fixture à
 * régénérer.
 *
 * LE DOCUMENT EST RICHE À DESSEIN : deux camps de quatre sièges, deux armes avec icônes,
 * inventaire complet (`d` / `am` / `g` / `gs`), capacités (connue et hors table), calque de
 * score sur deux joueurs, une pose d'écran occultant, un épisode de camouflage, un mort à
 * l'image lue, une lecture d'inventaire `empty: 'dead'`, un porteur de drapeau. Chaque étage
 * de la fiche est ainsi présent dans le HTML comparé — une régression sur n'importe lequel se
 * voit.
 *
 * LE GABARIT NORMAL VA JUSQU'À SIX SIÈGES PAR CAMP (la densité se lit sur la catégorie de
 * mode, jamais sur les effectifs — décision D1) : la même fixation à 6 sièges
 * (`__fixtures__/replayTeams.6v6.html`) a été prise à l'étape 2 du plan, AVANT le premier code
 * de la tuile compacte — le seul chemin de rendu était alors le gabarit normal. Un en-tête sans
 * catégorie sur douze sièges rend donc, et doit toujours rendre, la colonne d'aujourd'hui.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { resolveXuidMeta } from '@/features/match-view/xuidMeta'
import type { ReplayDocument } from '@/lib/api/types'

import { ReplayTeams } from './ReplayTeams'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

/** L'image lue : Hotel est mort (vie close à 90, retour à 180), tout le reste est en vie. */
const FRAME = 100

const NOMS = ['Alpha', 'Bravo', 'Charlie', 'Delta', 'Echo', 'Foxtrot', 'Golf', 'Hotel'] as const
const XUIDS = ['A', 'B', 'C', 'D', 'E', 'F', 'G', 'H'] as const

const ICONE_FUSIL = '/static/weapons-assets/halo_infinite/jeu/contour-01.png'
const ICONE_PISTOLET = '/static/weapons-assets/halo_infinite/jeu/contour-02.png'
const ICONE_FRAG = '/static/weapons-assets/halo_infinite/hud/Frag.png'
const ICONE_GRAPPIN = '/static/weapons-assets/halo_infinite/hud/Grapple.png'

type Point = NonNullable<NonNullable<ReplayDocument['tracks']>[number]['points']>[number]

/** Une vie sur [0,300] au slot donné, avec ses points (vitalité comprise ou non). */
function vie(slot: number, xuid: string, points: Point[]) {
  return { slot, team: -1, xuid, startFrame: 0, endFrame: 300, points }
}

/** Les quatre sièges de plus du 6v6 : deux par camp, vivants, une arme, un inventaire lu. */
const NOMS_6V6 = ['India', 'Juliett', 'Kilo', 'Lima'] as const
const XUIDS_6V6 = ['I', 'J', 'K', 'L'] as const

/** Le document de transport du 4v4 — la forme que `testReplayDoc` complète. */
function richeOver(): Partial<ReplayDocument> {
  return {
    roster: XUIDS.map((xuid, i) => ({ xuid, filmIndex: i, name: NOMS[i] })),
    tracks: [
      // Alpha : bouclier entamé lu à l'image 80 (âge 20 à l'image lue).
      vie(512, 'A', [{ t: 0, x: 1, y: 1, sh: 1, hp: 1 }, { t: 80, x: 2, y: 2, sh: 0.6, hp: 1 }]),
      // Bravo : aucune mesure sur SA vie — plein d'apparition (le document porte la vitalité).
      vie(513, 'B', [{ t: 0, x: 4, y: 4 }]),
      // Charlie : sous l'écran occultant posé en (5,0).
      vie(514, 'C', [{ t: 0, x: 5, y: 0, sh: 0.2, hp: 0.5 }]),
      // Delta : camouflage actif.
      vie(515, 'D', [{ t: 0, x: 8, y: 8, sh: 1, hp: 1 }]),
      vie(516, 'E', [{ t: 0, x: 0, y: 9, sh: 0.9, hp: 0.9 }]),
      // Foxtrot : porteur de drapeau.
      vie(517, 'F', [{ t: 0, x: 3, y: 7, sh: 1, hp: 0.8 }]),
      vie(518, 'G', [{ t: 0, x: 6, y: 6, sh: 0.5, hp: 0.5 }]),
      // Hotel : mort à l'image lue, retour lu à 180.
      { slot: 519, team: -1, xuid: 'H', startFrame: 0, endFrame: 90, points: [{ t: 0, x: 9, y: 1, sh: 1, hp: 1 }] },
      { slot: 520, team: -1, xuid: 'H', startFrame: 180, endFrame: 300, points: [{ t: 180, x: 9, y: 1 }] },
    ],
    weaponLabels: {
      '0xAAAA': { fr: 'Fusil', en: 'Rifle', img: ICONE_FUSIL, tinted: true },
      '0xBBBB': { fr: 'Pistolet', en: 'Pistol', img: ICONE_PISTOLET, tinted: true },
      '0xCCCC': { fr: 'Épée', en: 'Sword', fx: 'melee' },
    },
    loadouts: [
      { t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] },
      { t: 0, slot: 513, w: ['0xAAAA', '0xBBBB'] },
      { t: 0, slot: 514, w: ['0xAAAA', '0xBBBB'] },
      { t: 0, slot: 515, w: ['0xAAAA', '0xBBBB'] },
      { t: 0, slot: 516, w: ['0xCCCC', '0xAAAA'] },
      { t: 0, slot: 517, w: ['0xCCCC', '0xAAAA'] },
      { t: 0, slot: 518, w: ['0xCCCC', '0xAAAA'] },
      { t: 0, slot: 519, w: ['0xCCCC', '0xAAAA'] },
    ],
    grenadeLabels: [
      { fr: 'Fragmentation', en: 'Frag', img: ICONE_FRAG, tinted: true },
      { fr: 'Plasma', en: 'Plasma' },
    ],
    inventory: [
      // Alpha : main lue (emplacement 0), sélection de grenade LUE.
      { t: 0, slot: 512, d: 0, am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }], g: [1, 2], gs: 1 },
      // Bravo : main sur l'emplacement 0, jamais écrit (pictogramme « plein »), un seul type porté.
      { t: 0, slot: 513, d: 0, am: [{}, { mag: 3 }], g: [0, 2] },
      // Charlie : sélecteur NON LU, deux types portés sans sélection.
      { t: 0, slot: 514, am: [{ mag: 4 }, { mag: 1 }], g: [1, 1] },
      // Delta : armes rangées (D=2).
      { t: 0, slot: 515, d: 2, am: [{ mag: 8 }, { mag: 2 }], g: [2, 0] },
      // Echo : lecture pleine, puis lecture VIDE corroborée « mort » à t=50.
      { t: 0, slot: 516, d: 0, am: [{ gauge: 0.1 }, { mag: 6 }], g: [2, 1] },
      { t: 50, slot: 516, empty: 'dead' },
      // Foxtrot : arme à charge en main, consommation lue.
      { t: 0, slot: 517, d: 0, am: [{ gauge: 0.25 }, { mag: 7 }], g: [1, 0], gs: 0 },
      // Golf : ancrage parasite « 1/0 » sur l'épée rangée, main sur le fusil.
      { t: 0, slot: 518, d: 1, am: [{ mag: 1, res: 0 }, { mag: 8, res: 16 }], g: [1, 2], gs: 0 },
      { t: 0, slot: 519, d: 0, am: [{ gauge: 0 }, { mag: 9 }], g: [1, 1] },
    ],
    // Alpha reçoit une lecture DELTA de grenades plus récente que l'image-clé.
    grenadeReads: [
      { t: 0, slot: 512, g: [1, 2], src: 'kf' },
      { t: 70, slot: 512, g: [0, 2], src: 'delta' },
    ],
    abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot', img: ICONE_GRAPPIN, tinted: true } },
    abilities: [
      { t: 0, slot: 512, r: 20, src: 'i48' },
      // Golf : rang hors table — glyphe neutre, rang dans l'infobulle.
      { t: 0, slot: 518, r: 9, src: 'i48' },
    ],
    scoreTimeline: {
      teams: null,
      players: [
        {
          xuid: 'A',
          score: { rounds: null, total: [{ t: 5, v: 120 }, { t: 40, v: 350 }] },
          kills: { rounds: null, total: [{ t: 5, v: 1 }, { t: 40, v: 3 }] },
          deaths: { rounds: null, total: [{ t: 20, v: 2 }] },
          assists: { rounds: null, total: [{ t: 40, v: 4 }] },
        },
        {
          xuid: 'F',
          score: { rounds: null, total: [{ t: 30, v: 200 }] },
          kills: { rounds: null, total: [{ t: 30, v: 2 }] },
          deaths: { rounds: null, total: [{ t: 60, v: 3 }] },
          assists: { rounds: null, total: [] },
        },
      ],
    },
    equipmentPlacements: [
      {
        family: 'shroud_screen',
        id: '0x5eeb1a13',
        owner: 519,
        origin: 'deployed',
        t0: 10,
        t1: 200,
        x: 5,
        y: 0,
      },
    ],
    equipmentEpisodes: [{ slot: 515, fam: 'camo', t0: 0, t1: 150, endRead: true }],
    flagCarries: [{ team: 1, spans: [{ state: 'carried', xuid: 'F', t0: 60, t1: 140, x: 3, y: 7 }] }],
  }
}

function documentRiche() {
  return testReplayDoc(richeOver())
}

/** Le même document, plus deux sièges par camp : la limite haute du gabarit normal. */
function documentSixParCamp() {
  const base = richeOver()
  return testReplayDoc({
    ...base,
    roster: [
      ...(base.roster ?? []),
      ...XUIDS_6V6.map((xuid, i) => ({ xuid, filmIndex: 8 + i, name: NOMS_6V6[i] })),
    ],
    tracks: [
      ...(base.tracks ?? []),
      ...XUIDS_6V6.map((xuid, i) => vie(521 + i, xuid, [{ t: 0, x: i, y: 5, sh: 0.7, hp: 1 }])),
    ],
    loadouts: [
      ...(base.loadouts ?? []),
      ...XUIDS_6V6.map((_, i) => ({ t: 0, slot: 521 + i, w: ['0xBBBB', '0xAAAA'] })),
    ],
    inventory: [
      ...(base.inventory ?? []),
      ...XUIDS_6V6.map((_, i) => ({ t: 0, slot: 521 + i, d: 0, am: [{ mag: 6, res: 12 }, { mag: 20 }], g: [1, 0] })),
    ],
  })
}

/** Le tableau : `parCamp` sièges dans `t0`, le reste dans `t1` ; Alpha est le joueur de la page. */
function tableau(parCamp: 4 | 6) {
  const noms = parCamp === 6 ? [...NOMS, ...NOMS_6V6] : [...NOMS]
  const xuids = parCamp === 6 ? [...XUIDS, ...XUIDS_6V6] : [...XUIDS]
  // Les sièges de plus du 6v6 (I, J → t0 ; K, L → t1) s'ajoutent à chaque camp du 4v4.
  const camp = (i: number) => (i < 4 || i === 8 || i === 9 ? 't0' : 't1')
  return xuids.map((xuid, i) => scoreboardRow(xuid, noms[i], camp(i), i === 0 ? { is_me: true } : {}))
}

describe('ReplayTeams — fixation du DOM 4v4 (aucune régression hors BTB)', () => {
  it('le HTML de la colonne est celui de la fixture prise avant le lot', async () => {
    const board = tableau(4)
    const vue = render(
      <ReplayTeams
        doc={documentRiche()}
        scoreboard={board}
        frame={FRAME}
        locale="fr"
        xuidMeta={resolveXuidMeta(board, 'A')}
        header={{ start_time: '2026-07-24T20:00:00Z' }}
      />,
    )
    // Les huit fiches sont là, dans leurs deux camps — sans quoi la fixture comparerait un vide.
    for (const nom of NOMS) expect(vue.getByText(nom)).toBeTruthy()
    expect(vue.getByText('Équipe Eagle')).toBeTruthy()
    expect(vue.getByText('Équipe Cobra')).toBeTruthy()
    await expect(vue.container.innerHTML).toMatchFileSnapshot('./__fixtures__/replayTeams.4v4.html')
  })

  it('même fixation à 6 sièges par camp (limite haute du gabarit normal) : sans catégorie, la colonne d’aujourd’hui', async () => {
    const board = tableau(6)
    const vue = render(
      <ReplayTeams
        doc={documentSixParCamp()}
        scoreboard={board}
        frame={FRAME}
        locale="fr"
        xuidMeta={resolveXuidMeta(board, 'A')}
        header={{ start_time: '2026-07-24T20:00:00Z' }}
      />,
    )
    for (const nom of [...NOMS, ...NOMS_6V6]) expect(vue.getByText(nom)).toBeTruthy()
    // Douze sièges, et la colonne SIMPLE du gabarit normal — jamais la grille : aucun repli sur
    // les effectifs (D1), la fixture 6v6 en est la preuve nœud pour nœud.
    expect(vue.container.innerHTML).not.toContain('auto-fill')
    expect(vue.container.querySelectorAll('.h-\\[35px\\]')).toHaveLength(12)
    await expect(vue.container.innerHTML).toMatchFileSnapshot('./__fixtures__/replayTeams.6v6.html')
  })
})
