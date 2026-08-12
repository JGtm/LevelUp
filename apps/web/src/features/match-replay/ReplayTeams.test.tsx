/**
 * Tests — ReplayTeams (les fiches joueur du rejeu).
 *
 * CE QU'ILS PROTÈGENT : la règle n° 1 des fiches — une valeur non lue s'affiche comme une
 * LACUNE, jamais comme un zéro, une moyenne ou un nom inventé. Chaque bloc ci-dessous
 * éprouve un étage de la fiche (capacité, grenades, santé, armes) sur ses deux faces :
 * la mesure s'affiche, l'absence de mesure se dit.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { ReplayDocument } from '@/lib/api/types'

import { ReplayTeams } from './ReplayTeams'
import { testReplayDoc } from './test/testDoc'

/** Une vie vivante sur [0,100] pour le slot 512, rattachée au joueur A. */
const TRACK = {
  slot: 512,
  team: -1,
  xuid: 'A',
  startFrame: 0,
  endFrame: 100,
  points: [{ t: 0, x: 0, y: 0 }],
}

function renderTeams(over: Partial<ReplayDocument>, frame = 10) {
  const doc = testReplayDoc({
    roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
    tracks: [TRACK],
    ...over,
  })
  return render(<ReplayTeams doc={doc} scoreboard={[]} frame={frame} locale="fr" />)
}

describe('ReplayTeams — capacité équipée', () => {
  it('nomme la capacité quand la table la connaît', () => {
    renderTeams({
      abilityLabels: { '4': { fr: 'Grappin', en: 'Grappleshot' } },
      inventory: [{ t: 0, slot: 512, a: 4 }],
    })
    expect(screen.getByText('Grappin')).toBeTruthy()
  })

  it('garde le NUMÉRO d’un index hors table, marqué non interprétable', () => {
    // La table est partielle (4 index observés pour 11 capacités) : combler par un nom
    // voisin se lirait comme une certitude qu'on n'a pas.
    renderTeams({
      abilityLabels: { '4': { fr: 'Grappin', en: 'Grappleshot' } },
      inventory: [{ t: 0, slot: 512, a: 9 }],
    })
    expect(screen.getByText('capacité inconnue (9)')).toBeTruthy()
    expect(screen.queryByText('Grappin')).toBeNull()
  })

  it('sans lecture de capacité, n’affiche RIEN — l’absence n’est pas une capacité', () => {
    renderTeams({
      abilityLabels: { '4': { fr: 'Grappin', en: 'Grappleshot' } },
      inventory: [{ t: 0, slot: 512, g: [2, 0, 0, 0] }],
    })
    expect(screen.queryByText(/capacité inconnue/)).toBeNull()
    expect(screen.queryByText('Grappin')).toBeNull()
  })
})

describe('ReplayTeams — grenades portées', () => {
  const LABELS = [
    { en: 'Frag', fr: 'Fragmentation' },
    { en: 'Plasma', fr: 'Plasma' },
  ]

  it('rend chaque type porté avec son compte, nommé par rang', () => {
    renderTeams({
      grenadeLabels: LABELS,
      inventory: [{ t: 0, slot: 512, g: [1, 2] }],
    })
    expect(screen.getByText(/Fragmentation ×1/)).toBeTruthy()
    expect(screen.getByText(/Plasma ×2/)).toBeTruthy()
  })

  it('compteurs NON LUS (GrenadesRead=false) : aucune grenade affichée, jamais des zéros', () => {
    // Le décodeur n'écrit `g` que quand le bloc a été lu : absent = lacune. L'inventaire
    // reste affiché pour ce qu'il porte d'autre (ici la capacité).
    renderTeams({
      abilityLabels: { '4': { fr: 'Grappin', en: 'Grappleshot' } },
      grenadeLabels: LABELS,
      inventory: [{ t: 0, slot: 512, a: 4 }],
    })
    expect(screen.getByText('Grappin')).toBeTruthy()
    expect(screen.queryByText(/Fragmentation|Plasma|×/)).toBeNull()
  })
})

describe('ReplayTeams — grenade SÉLECTIONNÉE', () => {
  const LABELS = [
    { en: 'Frag', fr: 'Fragmentation' },
    { en: 'Plasma', fr: 'Plasma' },
  ]

  it('la sélection LUE (gs) encadre son type, même à deux types portés', () => {
    renderTeams({
      grenadeLabels: LABELS,
      inventory: [{ t: 0, slot: 512, g: [1, 2], gs: 1 }],
    })
    const sel = screen.getByText(/Plasma ×2/)
    expect(sel.title).toMatch(/LU dans le film/)
    // L'autre type porté n'est pas marqué.
    expect(screen.getByText(/Fragmentation ×1/).title).toBe('')
  })

  it('un seul type porté sans lecture : sélection DÉDUITE, dite déduite', () => {
    renderTeams({
      grenadeLabels: LABELS,
      inventory: [{ t: 0, slot: 512, g: [0, 2] }],
    })
    expect(screen.getByText(/Plasma ×2/).title).toMatch(/le seul porté/)
  })

  it('deux types sans lecture : « sél. ? » — l’indétermination se dit, elle ne se devine pas', () => {
    renderTeams({
      grenadeLabels: LABELS,
      inventory: [{ t: 0, slot: 512, g: [1, 2] }],
    })
    expect(screen.getByText('sél. ?')).toBeTruthy()
    expect(screen.getByText(/Plasma ×2/).title).toBe('')
  })
})

describe('ReplayTeams — munitions et sélecteur', () => {
  const TWO_WEAPONS = {
    weaponLabels: {
      '0xAAAA': { fr: 'Fusil', en: 'Rifle' },
      '0xBBBB': { fr: 'Pistolet', en: 'Pistol' },
    },
    loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
  }

  it('les cellules suivent l’ordre des armes : la dégainée d’abord, index d’emplacement gardé', () => {
    renderTeams({
      ...TWO_WEAPONS,
      inventory: [
        { t: 0, slot: 512, d: 1, am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }] },
      ],
    })
    const cells = screen.getAllByTitle(/Munitions de l'emplacement/)
    // L'emplacement 1 (dégainé) est rendu en premier, et son index reste « 1 ».
    expect(cells[0].textContent).toContain('1')
    expect(cells[0].textContent).toContain('5/6')
    expect(cells[0].title).toMatch(/DÉGAINÉ/)
    expect(cells[1].textContent).toContain('10/20')
  })

  it('sélecteur disant « rien de dégainé » : jeton « rangées » — une mesure, pas une lacune', () => {
    renderTeams({
      ...TWO_WEAPONS,
      inventory: [{ t: 0, slot: 512, d: 2, am: [{ mag: 10 }, { mag: 5 }] }],
    })
    expect(screen.getByText('rangées')).toBeTruthy()
  })

  it('sélecteur NON LU : jeton « dégainée ? » — une lacune dite', () => {
    renderTeams({
      ...TWO_WEAPONS,
      inventory: [{ t: 0, slot: 512, am: [{ mag: 10 }, { mag: 5 }] }],
    })
    expect(screen.getByText('dégainée ?')).toBeTruthy()
  })
})

describe('ReplayTeams — mort et réapparition', () => {
  it('mort avec retour lu : compte à rebours ET barre d’avancement depuis la mort', () => {
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
      tracks: [
        TRACK,
        { slot: 514, team: -1, xuid: 'A', startFrame: 180, endFrame: 260, points: [{ t: 180, x: 0, y: 0 }] },
      ],
    })
    render(<ReplayTeams doc={doc} scoreboard={[]} frame={140} locale="fr" />)
    expect(screen.getByText('retour dans')).toBeTruthy()
    expect(screen.getByRole('progressbar')).toBeTruthy()
  })

  it('mort sans vie suivante : « retour ? », jamais un délai deviné, et pas de barre', () => {
    renderTeams({}, 140)
    expect(screen.getByText('retour ?')).toBeTruthy()
    expect(screen.queryByRole('progressbar')).toBeNull()
  })
})

describe('ReplayTeams — armes au SPAWN (avant la première image-clé de la vie)', () => {
  it('montre les armes de la première image-clé à venir, jamais « armes non lues »', () => {
    // Le loadout n'est lu qu'à t=80 ; à frame 10, la fiche montre déjà ces armes (lecture
    // à venir de la MÊME vie, âge négatif dit en infobulle) — doctrine du POC.
    renderTeams({
      weaponLabels: { '0xAAAA': { fr: 'Fusil', en: 'Rifle' } },
      loadouts: [{ t: 80, slot: 512, w: ['0xAAAA'] }],
    })
    expect(screen.getByText('Fusil')).toBeTruthy()
    expect(screen.queryByText('armes non lues sur cette vie')).toBeNull()
  })
})

describe('ReplayTeams — armes en icônes', () => {
  it('une arme avec visuel rend son icône (accessible par son nom), pas son libellé en texte', () => {
    renderTeams({
      weaponLabels: {
        '0xAAAA': { fr: 'Fusil', en: 'Rifle', img: '/static/weapons-assets/halo_infinite/jeu/contour-01.png', tinted: true },
        '0xBBBB': { fr: 'Pistolet', en: 'Pistol' },
      },
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
    })
    // L'icône porte le nom en accessibilité ; le texte n'est plus rendu pour elle.
    expect(screen.getByRole('img', { name: 'Fusil' })).toBeTruthy()
    // L'arme SANS visuel garde son libellé : jamais l'icône d'une arme voisine.
    expect(screen.getByText('Pistolet')).toBeTruthy()
  })
})

describe('ReplayTeams — dégradation par ABSENCE DE DONNÉE (multi-titre)', () => {
  // Un titre sans décodage film (ou un match sans film) publie des champs simplement
  // ABSENTS : `hp` sur les points, `d`/`a` dans l'inventaire. La fiche doit rendre ce
  // qu'elle sait, dire ses lacunes, et ne JAMAIS jeter — aucune comparaison de slug.
  it('document sans AUCUNE vitalité (Point sans sh/hp) : PAS de barres — on n’invente pas une jauge', () => {
    renderTeams({
      titleSlug: 'autre_titre',
      weaponLabels: { '0xAAAA': { fr: 'Fusil', en: 'Rifle' }, '0xBBBB': { fr: 'Pistolet', en: 'Pistol' } },
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
      inventory: [{ t: 0, slot: 512, g: [1, 0] }],
      grenadeLabels: [{ fr: 'Fragmentation', en: 'Frag' }],
    })
    // La fiche existe et nomme le joueur.
    expect(screen.getByText('Alpha')).toBeTruthy()
    // Aucun point du document ne porte sh/hp : AUCUNE barre — un titre qui ne transmet pas
    // la vitalité n'affiche pas des jauges pleines inventées.
    expect(screen.queryByLabelText('Bouclier')).toBeNull()
    expect(screen.queryByLabelText('Santé')).toBeNull()
    // Les armes s'affichent par emplacement, sans main désignée.
    expect(screen.getByText('Fusil')).toBeTruthy()
    expect(screen.getByText('Pistolet')).toBeTruthy()
    expect(screen.queryByText('en main')).toBeNull()
    // Aucune capacité lue : la ligne n'existe pas, et rien n'est inventé.
    expect(screen.queryByText(/capacité inconnue/)).toBeNull()
  })

  it('document réduit aux traces (ni inventaire, ni loadout, ni vitalité) : la fiche dit ses lacunes sans erreur', () => {
    renderTeams({})
    expect(screen.getByText('Alpha')).toBeTruthy()
    expect(screen.queryByLabelText('Bouclier')).toBeNull()
    expect(screen.queryByLabelText('Santé')).toBeNull()
    expect(screen.getByText('armes non lues sur cette vie')).toBeTruthy()
  })
})

describe('ReplayTeams — vitalité : plein d’apparition', () => {
  it('le document porte la vitalité mais la vie n’a pas encore de mesure : barres PLEINES', () => {
    // On apparaît vie et bouclier pleins (règle du jeu) et le flux différentiel ne
    // retransmet que ce qui change : « rien d'arrivé » = « plein », pas « inconnu »
    // (décision utilisateur 2026-08-12, doctrine du POC).
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
      tracks: [
        TRACK, // la vie affichée : aucun point ne porte sh/hp
        {
          slot: 513,
          team: -1,
          xuid: 'B',
          startFrame: 0,
          endFrame: 100,
          // C'est un AUTRE joueur qui prouve que le document transmet la vitalité.
          points: [{ t: 0, x: 0, y: 0, sh: 0.4, hp: 0.7 }],
        },
      ],
    })
    render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" />)
    const shield = screen.getAllByLabelText('Bouclier')
    const health = screen.getAllByLabelText('Santé')
    // La fiche d'Alpha (sans mesure) montre des barres PLEINES, pas une lacune.
    const alphaShield = shield[0].firstElementChild as HTMLElement
    const alphaHealth = health[0].firstElementChild as HTMLElement
    expect(alphaShield.style.width).toBe('100%')
    expect(alphaHealth.style.width).toBe('100%')
  })
})
