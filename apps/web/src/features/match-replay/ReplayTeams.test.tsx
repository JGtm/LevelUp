/**
 * Tests — ReplayTeams (les fiches joueur du rejeu).
 *
 * CE QU'ILS PROTÈGENT : la règle n° 1 des fiches — une valeur non lue s'affiche comme une
 * LACUNE, jamais comme un zéro, une moyenne ou un nom inventé. Chaque bloc ci-dessous
 * éprouve un étage de la fiche (capacité, grenades, santé, armes) sur ses deux faces :
 * la mesure s'affiche, l'absence de mesure se dit.
 */
import { describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'

import { resolveXuidMeta } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { ReplayTeams } from './ReplayTeams'
import { testReplayDoc } from './test/testDoc'

/** Une ligne de scoreboard minimale : seuls le camp et l'identité comptent pour ces tests. */
function sbRow(xuid: string, gamertag: string, side: string | null): MatchScoreboardRow {
  return {
    xuid,
    gamertag,
    team_side: side,
    is_me: false,
    rank: 1,
    score: 0,
    kills: 1,
    deaths: 1,
    assists: 0,
    shots_fired: null,
    shots_hit: null,
    accuracy: null,
    damage_dealt: null,
    damage_taken: null,
    average_life: null,
    headshot_kills: null,
    max_killing_spree: null,
    perfect_kills: null,
    power_weapon_kills: null,
    melee_kills: null,
    outcome_label: 'Victoire',
  }
}

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
      abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot' } },
      abilities: [{ t: 0, slot: 512, r: 20, src: 'i48' }],
    })
    expect(screen.getByText('Grappin')).toBeTruthy()
  })

  it('rang hors table : un GLYPHE, et le numéro dans son infobulle', () => {
    // La table est partielle ET propre à la palette du film : combler par un nom
    // voisin se lirait comme une certitude qu'on n'a pas. Depuis le 2026-08-17 la fiche
    // pose un glyphe (« pas un caractère », planche du 16/08) et garde le rang en
    // infobulle — la seule chose vraie à cet endroit.
    renderTeams({
      abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot' } },
      abilities: [{ t: 0, slot: 512, r: 9, src: 'i48' }],
    })
    const mark = screen.getByRole('img', { name: 'capacité non identifiée (rang 9)' })
    expect(mark.querySelector('svg')).toBeTruthy()
    expect(screen.queryByText('Grappin')).toBeNull()
  })

  it('sans lecture de capacité, n’affiche RIEN — l’absence n’est pas une capacité', () => {
    renderTeams({
      abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot' } },
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
    expect(screen.getByTitle('Fragmentation').textContent).toContain('×1')
    expect(screen.getByTitle('Plasma').textContent).toContain('×2')
  })

  it('rend la VIGNETTE du HUD quand le document en sert une, le nom restant en infobulle', () => {
    renderTeams({
      grenadeLabels: [
        { en: 'Frag', fr: 'Fragmentation', img: '/static/weapons-assets/halo_infinite/hud/Frag.png', tinted: true },
      ],
      inventory: [{ t: 0, slot: 512, g: [2] }],
    })
    expect(screen.getByRole('img', { name: 'Fragmentation' })).toBeTruthy()
    // Le nom n'est plus écrit en clair : la vignette porte l'identité.
    expect(screen.queryByText('Fragmentation')).toBeNull()
  })

  it('compteurs NON LUS (GrenadesRead=false) : aucune grenade affichée, jamais des zéros', () => {
    // Le décodeur n'écrit `g` que quand le bloc a été lu : absent = lacune. L'inventaire
    // reste affiché pour ce qu'il porte d'autre (ici la capacité).
    renderTeams({
      abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot' } },
      grenadeLabels: LABELS,
      abilities: [{ t: 0, slot: 512, r: 20, src: 'i48' }],
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
    expect(screen.getByTitle(/^Plasma — .*LU dans le film/)).toBeTruthy()
    // L'autre type porté n'est pas marqué.
    expect(screen.getByTitle('Fragmentation')).toBeTruthy()
  })

  it('un seul type porté sans lecture : sélection DÉDUITE, dite déduite', () => {
    renderTeams({
      grenadeLabels: LABELS,
      inventory: [{ t: 0, slot: 512, g: [0, 2] }],
    })
    expect(screen.getByTitle(/^Plasma — .*le seul porté/)).toBeTruthy()
  })

  it('deux types sans lecture : « sél. ? » — l’indétermination se dit, elle ne se devine pas', () => {
    renderTeams({
      grenadeLabels: LABELS,
      inventory: [{ t: 0, slot: 512, g: [1, 2] }],
    })
    expect(screen.getByText('sél. ?')).toBeTruthy()
    // Aucun des deux types n'est marqué sélectionné (title = nom seul).
    expect(screen.getByTitle('Plasma')).toBeTruthy()
    expect(screen.getByTitle('Fragmentation')).toBeTruthy()
  })
})

/**
 * LE BRANCHEMENT CLIENT DE L'AXE `grenadeReads` (schéma 20, lot 4 du suivi delta).
 *
 * CE QUE CES CAS PROTÈGENT, ET POURQUOI ILS EXISTENT. La revue du 2026-08-25 a constaté que
 * l'inversion COMPLÈTE de la préférence — la boîte retombant sur l'inventaire au lieu de l'axe,
 * donc l'annulation pure et simple du bénéfice du lot — laissait 1116 tests verts : aucun test
 * de rendu n'éprouvait la boîte de grenades sur cet axe. Un gain de fraîcheur que rien ne
 * verrouille se perd au premier refactor.
 *
 * LES QUATRE CAS SONT LES QUATRE BRANCHES de `grenadeBoxAt`, et chacune dit une chose vraie
 * différente : l'axe gagne quand il est PASSÉ, l'artefact ancien retombe sur l'inventaire, une
 * lecture À VENIR ne prime pas une information passée, et une lecture à venir SEULE s'affiche
 * en le disant.
 */
describe('ReplayTeams — boîte de grenades : quelle lecture, et de quel âge', () => {
  const LABELS = [
    { en: 'Frag', fr: 'Fragmentation' },
    { en: 'Plasma', fr: 'Plasma' },
  ]

  it('artefact 20 : la lecture DELTA plus récente que l’image-clé gagne, avec SON âge', () => {
    // Le point du lot : entre deux images-clés (~20 s) le paquet delta rajeunit la boîte. Elle
    // doit afficher les compteurs delta ET dater son infobulle de la lecture delta — pas de
    // l'inventaire, dont l'image-clé est trois fois plus vieille ici.
    renderTeams(
      {
        grenadeLabels: LABELS,
        inventory: [{ t: 0, slot: 512, g: [2, 0] }],
        grenadeReads: [
          { t: 0, slot: 512, g: [2, 0], src: 'kf' },
          { t: 60, slot: 512, g: [0, 3], src: 'delta' },
        ],
      },
      90,
    )
    const box = screen.getByTitle(/Grenades lues il y a 0\.5 s/)
    expect(box.textContent, 'les compteurs delta, pas ceux de l’image-clé').toContain('×3')
    expect(box.textContent).not.toContain('Fragmentation')
    // L'ESTOMPAGE EST CELUI DE LA CELLULE, PAS CELUI DE LA RANGÉE (ronde 2 de la revue :
    // les deux mutations — réintroduire l'opacité du conteneur, retirer celle de la boîte —
    // laissaient 1119 tests verts). La boîte porte sa propre opacité, calculée sur SON âge ;
    // le conteneur n'en impose aucune, sinon les deux se MULTIPLIENT et une lecture delta
    // fraîche s'affiche plus pâle qu'avant le lot.
    expect(box.style.opacity, 'la boîte porte son estompage propre').not.toBe('')
    expect(Number(box.style.opacity)).toBeGreaterThan(0)
    expect(Number(box.style.opacity)).toBeLessThanOrEqual(1)
    expect(
      (box.parentElement as HTMLElement).style.opacity,
      'la rangée n’impose plus son âge à la grille',
    ).toBe('')
  })

  it('artefact ≤ 19 (aucun `grenadeReads`) : la boîte retombe sur l’inventaire, âge inchangé', () => {
    // LE REPLI EST LE POINT : un artefact ancien ne se vide pas, il est seulement moins frais.
    renderTeams(
      {
        grenadeLabels: LABELS,
        inventory: [{ t: 0, slot: 512, g: [1, 2] }],
      },
      60,
    )
    const box = screen.getByTitle(/Grenades lues il y a 1\.0 s/)
    expect(within(box).getByRole('img', { name: 'Fragmentation' })).toBeTruthy()
    expect(within(box).getByRole('img', { name: 'Plasma' })).toBeTruthy()
    expect(box.textContent).toContain('×1')
    expect(box.textContent).toContain('×2')
  })

  it('lecture de grenades À VENIR : l’information PASSÉE de l’inventaire prime', () => {
    // Même doctrine que la « lecture vide À VENIR » : une lecture à venir n'affirme rien du
    // présent. Sans cette règle, la fiche affichait au slot 554 du film de référence une plasma
    // mesurée ~60 s plus tard, à pleine opacité, sous une infobulle « lu il y a X ».
    renderTeams(
      {
        grenadeLabels: LABELS,
        inventory: [{ t: 0, slot: 512, g: [2, 0] }],
        grenadeReads: [{ t: 90, slot: 512, g: [0, 5], src: 'delta' }],
      },
      30,
    )
    const box = screen.getByTitle(/Grenades lues il y a 0\.5 s/)
    expect(
      within(box).getByRole('img', { name: 'Fragmentation' }),
      'les compteurs de l’inventaire passé',
    ).toBeTruthy()
    expect(box.textContent).toContain('×2')
    expect(box.textContent, 'jamais la lecture à venir').not.toContain('×5')
    expect(within(box).queryByRole('img', { name: 'Plasma' })).toBeNull()
  })

  it('lecture À VENIR sans RIEN de passé : elle s’affiche, et l’infobulle dit « dans »', () => {
    // Rien de passé à préférer : la lecture à venir informe mieux qu'une boîte vide — à
    // condition d'être ASSUMÉE. Âge négatif, infobulle « dans X s », jamais « il y a ».
    renderTeams(
      {
        grenadeLabels: LABELS,
        grenadeReads: [{ t: 90, slot: 512, g: [0, 5], src: 'delta' }],
      },
      30,
    )
    const box = screen.getByTitle(/Grenades lues dans 1\.0 s/)
    expect(box.textContent).toContain('×5')
    expect(screen.queryByTitle(/Grenades lues il y a/)).toBeNull()
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

  it('SEULE l’arme dégainée garde ses munitions (règle de la fiche unique)', () => {
    const view = renderTeams({
      ...TWO_WEAPONS,
      inventory: [
        { t: 0, slot: 512, d: 1, am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }] },
      ],
    })
    // `d: 1` = l'emplacement 1 est dégainé : ses munitions (5/6) restent, l'autre part.
    const drawn = screen.getByTitle(/DÉGAINÉ/)
    expect(drawn.textContent).toContain('5/6')
    expect(view.container.textContent).not.toContain('10/20')
  })

  it('sélecteur disant « rien de dégainé » (D=2) : AUCUNE munition — aucune arme en main à décrire', () => {
    // Le pictogramme « armes rangées » a été supprimé (demande utilisateur du 2026-08-24 :
    // « je comprends pas ce que c'est ») : l'état se lit à l'absence de soulignement « en
    // main » sur les armes, la cellule de munitions reste vide.
    const view = renderTeams({
      ...TWO_WEAPONS,
      inventory: [{ t: 0, slot: 512, d: 2, am: [{ mag: 10 }, { mag: 5 }] }],
    })
    expect(screen.queryByRole('img', { name: 'Armes rangées' })).toBeNull()
    expect(view.container.textContent).not.toContain('10')
    expect(screen.queryByText('dégainée ?')).toBeNull()
  })

  it('emplacement jamais écrit : pictogramme « Munitions pleines », jamais « aucune »', () => {
    // Décision produit 4 : non écrit = PLEIN (flux différentiel). Le pictogramme le dit
    // sans exposer la mécanique du flux.
    renderTeams({
      ...TWO_WEAPONS,
      inventory: [{ t: 0, slot: 512, d: 0, am: [{}, { mag: 5 }] }],
    })
    expect(screen.getByRole('img', { name: 'Munitions pleines' })).toBeTruthy()
    expect(screen.queryByText('aucune')).toBeNull()
  })

  it('sélecteur NON LU : jeton « dégainée ? » — une lacune dite', () => {
    renderTeams({
      ...TWO_WEAPONS,
      inventory: [{ t: 0, slot: 512, am: [{ mag: 10 }, { mag: 5 }] }],
    })
    expect(screen.getByText('dégainée ?')).toBeTruthy()
  })

  it('arme À CHARGE en main : pourcentage écrit, jamais le « 1/0 » parasite du film', () => {
    // Mesure 2026-08-24 : les cellules mag=1/res=0 des armes à charge (épée, marteau,
    // plasma...) sont des ancrages parasites — la fiche affiche « XXX% » (demande user).
    const view = renderTeams({
      weaponLabels: {
        '0xAAAA': { fr: 'Épée', en: 'Sword', fx: 'melee' },
        '0xBBBB': { fr: 'Pistolet', en: 'Pistol' },
      },
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
      inventory: [{ t: 0, slot: 512, d: 0, am: [{ mag: 1, res: 0 }, { mag: 5 }] }],
    })
    // Rien de consommé (aucune jauge émise, flux différentiel) : la charge est PLEINE.
    expect(screen.getByText('100%')).toBeTruthy()
    expect(view.container.textContent).not.toContain('1/0')
  })

  it('arme à charge avec consommation lue : le POURCENTAGE restant (complément du consommé)', () => {
    renderTeams({
      weaponLabels: { '0xAAAA': { fr: 'Épée', en: 'Sword', fx: 'melee' } },
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA'] }],
      inventory: [{ t: 0, slot: 512, d: 0, am: [{ gauge: 0.25 }] }],
    })
    expect(screen.getByText('75%')).toBeTruthy()
  })
})

describe('ReplayTeams — hauteur constante vivant/mort', () => {
  it('la fiche d’un mort garde ses zones armes et inventaire : la place est réservée', () => {
    // Item 1.1 du plan parité : le même document rend, pour la même fiche, le même NOMBRE
    // de rangées vivant (frame 10) et mort (frame 140, aucune vie suivante). Les zones
    // fantômes réservent la place ; l'égalité au pixel se vérifie au gate visuel user.
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
      tracks: [{ ...TRACK, points: [{ t: 0, x: 0, y: 0, sh: 1, hp: 1 }] }],
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA'] }],
      inventory: [{ t: 0, slot: 512, g: [1, 0] }],
    })
    const cardRows = (view: ReturnType<typeof render>) => {
      const card = view.getByText('Alpha').parentElement?.parentElement as HTMLElement
      return card.childElementCount
    }
    const alive = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" />)
    const aliveRows = cardRows(alive)
    alive.unmount()
    const dead = render(<ReplayTeams doc={doc} scoreboard={[]} frame={140} locale="fr" />)
    expect(dead.getByText('Réapparition ?')).toBeTruthy()
    expect(cardRows(dead)).toBe(aliveRows)
  })
})

/**
 * LA FICHE MORTE, DÉPOUILLÉE (demande utilisateur du 2026-08-25 : « virer l'accentuation sur la
 * bordure gauche de la fiche quand le joueur est mort ? Et centre le compteur de réapparition et
 * virer la jauge »). Trois retraits, trois verrous — et ce qui RESTE est vérifié aussi : la mort
 * doit continuer de se lire, sans quoi le dépouillement l'aurait effacée.
 */
describe('ReplayTeams — mort et réapparition', () => {
  /** La fiche d'Alpha : deux niveaux au-dessus de son nom (marque retirée, cf. A1). */
  const carte = (view: ReturnType<typeof render>) =>
    view.getByText('Alpha').parentElement?.parentElement as HTMLElement

  const docAvecRetour = () =>
    testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
      tracks: [
        TRACK,
        { slot: 514, team: -1, xuid: 'A', startFrame: 180, endFrame: 260, points: [{ t: 180, x: 0, y: 0 }] },
      ],
    })

  it('mort avec retour lu : le compte à rebours, et AUCUNE jauge', () => {
    render(<ReplayTeams doc={docAvecRetour()} scoreboard={[]} frame={140} locale="fr" />)
    expect(screen.getByText('Réapparition dans')).toBeTruthy()
    expect(screen.queryByRole('progressbar')).toBeNull()
  })

  // LA RANGÉE ELLE-MÊME, PAS SON PARENT : la cellule qui l'accueille porte DÉJÀ
  // `justify-center` — pour son centrage VERTICAL (elle est en `flex-col`). Viser le parent
  // ferait passer ce cas quoi qu'on écrive dans `RespawnRow` ; un premier essai l'a montré,
  // la mutation survivait. `getByText` rend bien la rangée : elle porte le libellé en enfant
  // texte DIRECT, le compte étant dans un `<b>`.
  it('le compte à rebours est CENTRÉ dans sa rangée', () => {
    const view = render(<ReplayTeams doc={docAvecRetour()} scoreboard={[]} frame={140} locale="fr" />)
    expect(view.getByText('Réapparition dans').className).toContain('justify-center')
  })

  it('la LACUNE est centrée elle aussi — la cellule ne se décale pas selon ce qu’elle porte', () => {
    const view = renderTeams({}, 140)
    expect(view.getByText('Réapparition ?').className).toContain('text-center')
  })

  it('mort sans vie suivante : « Réapparition ? », jamais un délai deviné', () => {
    renderTeams({}, 140)
    expect(screen.getByText('Réapparition ?')).toBeTruthy()
  })

  // (a) DU VERBATIM : plus de liseré gauche sur une fiche morte. Le `boxShadow` était le seul
  // porteur de cet accent — sa disparition du style en ligne est donc la mesure exacte.
  it('aucun liseré gauche sur une fiche morte', () => {
    const view = render(<ReplayTeams doc={docAvecRetour()} scoreboard={[]} frame={140} locale="fr" />)
    expect(carte(view).style.boxShadow).toBe('')
  })

  // CE QUI RESTE, et qui doit rester : la mort se lit encore au fond teinté et au nom. Retirer
  // le liseré ne devait pas rendre une fiche morte indistinguable d'une fiche vivante.
  it('la mort reste DITE : fond teinté et nom à l’encre d’alerte', () => {
    const mort = render(<ReplayTeams doc={docAvecRetour()} scoreboard={[]} frame={140} locale="fr" />)
    expect(carte(mort).style.background).toContain('color-mix')
    expect(mort.getByText('Alpha').style.color).not.toBe('')
    mort.unmount()
    const vivant = render(<ReplayTeams doc={docAvecRetour()} scoreboard={[]} frame={10} locale="fr" />)
    expect(carte(vivant).style.background).toBe('')
    expect(vivant.getByText('Alpha').style.color).toBe('')
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
    // La lacune vit en INFOBULLE depuis la grille à cellules fixes (2026-08-24) : les
    // cellules restent, vides en pointillés, et la phrase les explique au survol.
    expect(screen.getByTitle('armes non lues sur cette vie')).toBeTruthy()
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

describe('ReplayTeams — état actif équipement (camo/surbouclier), cahier des charges 21.1', () => {
  const cardOf = (view: ReturnType<typeof render>) =>
    view.getByText('Alpha').parentElement?.parentElement as HTMLElement

  it('camouflage actif : effet de VERRE (flou + fond translucide), JAMAIS une opacité réduite', () => {
    // Règle n°1.1 du fichier : un état se dit par un effet dédié, jamais en faisant
    // fondre la fiche entière (l'ancien défaut que ce lot corrige).
    const view = renderTeams({
      equipmentEpisodes: [{ slot: 512, fam: 'camo', t0: 0, t1: 50, endRead: true }],
    })
    const card = cardOf(view)
    expect(card.style.backdropFilter).toContain('blur')
    expect(card.style.background).toContain('var(--card)')
    expect(card.style.opacity).toBe('')
  })

  it('surbouclier actif : ENCADRÉ au token `legendary`, jamais `info` (réservé à la jauge de bouclier)', () => {
    const view = renderTeams({
      equipmentEpisodes: [{ slot: 512, fam: 'overshield', t0: 0, t1: 50, endRead: true }],
    })
    const card = cardOf(view)
    expect(card.style.boxShadow).toContain('var(--ac-legendary)')
    expect(card.style.boxShadow).not.toContain('var(--ac-info)')
    // Le cadre n'a pas de fond propre : cf. la composition avec le verre ci-dessous.
    expect(card.style.background).toBe('')
  })

  it('les deux effets se COMPOSENT : les ombres s’accumulent, le fond reste celui du verre', () => {
    const view = renderTeams({
      equipmentEpisodes: [
        { slot: 512, fam: 'camo', t0: 0, t1: 50, endRead: true },
        { slot: 512, fam: 'overshield', t0: 0, t1: 50, endRead: true },
      ],
    })
    const card = cardOf(view)
    expect(card.style.backdropFilter).toContain('blur')
    expect(card.style.background).toContain('var(--card)')
    expect(card.style.boxShadow).toContain('var(--ac-legendary)')
    expect(card.style.boxShadow).toContain('var(--border)')
  })

  it('hors de la fenêtre [t0,t1] mesurée : aucun des deux effets — la fiche reste normale', () => {
    const view = renderTeams({
      equipmentEpisodes: [{ slot: 512, fam: 'camo', t0: 60, t1: 90, endRead: true }],
    }, 10)
    const card = cardOf(view)
    expect(card.style.backdropFilter).toBe('')
    expect(card.style.boxShadow).toBe('')
  })

  it('une fiche MORTE ne porte jamais un effet d’équipement (un épisode se ferme à la mort au plus tard)', () => {
    const view = renderTeams({
      equipmentEpisodes: [{ slot: 512, fam: 'camo', t0: 0, t1: 99, endRead: false }],
    }, 140)
    const card = cardOf(view)
    expect(card.style.backdropFilter).toBe('')
  })
})

/**
 * L'EN-TÊTE DE COLONNE (décision D8, 2026-08-16) : il affichait `t0` / `t1`, c'est-à-dire
 * l'identifiant de transport du backend. Il porte maintenant le libellé résolu — la cascade
 * du scoreboard, pas une troisième copie — et la couleur d'équipe du reste de la page.
 */
describe('ReplayTeams — nom d’équipe des colonnes (D8)', () => {
  const twoTeams = () =>
    testReplayDoc({
      roster: [
        { xuid: 'A', filmIndex: 0, name: 'Alpha' },
        { xuid: 'B', filmIndex: 1, name: 'Bravo' },
      ],
      tracks: [TRACK, { ...TRACK, slot: 513, xuid: 'B' }],
    })

  it('résout `t0` en « Équipe Eagle » (FR) et `t1` en « Team Cobra » (EN)', () => {
    const doc = twoTeams()
    const board = [sbRow('A', 'Alpha', 't0'), sbRow('B', 'Bravo', 't1')]
    const fr = render(<ReplayTeams doc={doc} scoreboard={board} frame={10} locale="fr" />)
    expect(fr.getByText('Équipe Eagle')).toBeTruthy()
    expect(fr.queryByText('t0')).toBeNull()
    fr.unmount()
    render(<ReplayTeams doc={doc} scoreboard={board} frame={10} locale="en" />)
    expect(screen.getByText('Team Cobra')).toBeTruthy()
  })

  it('numérote une équipe hors référentiel plutôt que d’inventer un nom', () => {
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
      tracks: [TRACK],
    })
    render(<ReplayTeams doc={doc} scoreboard={[sbRow('A', 'Alpha', 't12')]} frame={10} locale="fr" />)
    expect(screen.getByText('Équipe 12')).toBeTruthy()
  })

  it('sans camp connu : « Sans équipe », encre neutre — aucune couleur d’équipe empruntée', () => {
    const view = renderTeams({})
    const header = view.getByText('Sans équipe').parentElement as HTMLElement
    expect(header.style.borderLeft).toBe('')
    expect(header.className).toContain('text-muted-foreground')
  })

  it('teinte la colonne du camp du joueur de la page (allié) et l’autre en adverse', () => {
    const doc = twoTeams()
    const board = [sbRow('A', 'Alpha', 't0'), sbRow('B', 'Bravo', 't1')]
    const meta = resolveXuidMeta(board, 'A')
    const view = render(
      <ReplayTeams doc={doc} scoreboard={board} frame={10} locale="fr" xuidMeta={meta} />,
    )
    const headerOf = (label: string) => view.getByText(label).parentElement as HTMLElement
    expect(headerOf('Équipe Eagle').style.borderLeft).toContain('var(--ac-team-ally)')
    expect(headerOf('Équipe Cobra').style.borderLeft).toContain('var(--ac-team-enemy)')
  })
})

describe('ReplayTeams — compteurs de fiche : publiés, ou ceux de la base', () => {
  const board = () => [sbRow('A', 'Alpha', 't0'), sbRow('B', 'Bravo', 't0')]
  const twoLives = {
    roster: [
      { xuid: 'A', filmIndex: 0, name: 'Alpha' },
      { xuid: 'B', filmIndex: 1, name: 'Bravo' },
    ],
    tracks: [TRACK, { ...TRACK, slot: 513, xuid: 'B' }],
  }
  const timeline = {
    teams: null,
    players: [
      {
        xuid: 'A',
        score: { rounds: null, total: [{ t: 5, v: 120 }, { t: 40, v: 350 }] },
        kills: { rounds: null, total: [{ t: 5, v: 1 }, { t: 40, v: 3 }] },
        deaths: { rounds: null, total: [{ t: 20, v: 2 }] },
        assists: { rounds: null, total: [{ t: 40, v: 4 }] },
      },
    ],
  }

  it('le joueur PUBLIÉ montre son score personnel et ses compteurs à l’instant lu', () => {
    const doc = testReplayDoc({ ...twoLives, scoreTimeline: timeline })
    const view = render(<ReplayTeams doc={doc} scoreboard={board()} frame={100} locale="fr" />)
    expect(view.getByTitle("Score personnel à l'instant lu").textContent).toBe('350')
    const live = view.getByTitle("Frags / morts / assistances à l'instant lu")
    expect(live.textContent).toBe('3/2/4')
  })

  it('et ils TIQUENT : à l’image 10, seuls les paliers déjà passés comptent', () => {
    const doc = testReplayDoc({ ...twoLives, scoreTimeline: timeline })
    const view = render(<ReplayTeams doc={doc} scoreboard={board()} frame={10} locale="fr" />)
    expect(view.getByTitle("Score personnel à l'instant lu").textContent).toBe('120')
    // Aucune mort ni assistance transmise avant l'image 20 : zéro est la valeur, pas une lacune.
    expect(view.getByTitle("Frags / morts / assistances à l'instant lu").textContent).toBe('1/0/0')
  })

  it('le joueur NON publié garde les totaux de la BASE — jamais un zéro inventé', () => {
    // Bravo n'a pas de série : sa fiche ne montre pas de score personnel, et ses trois
    // nombres restent ceux du match (sbRow : 1 frag, 1 mort, 0 assistance).
    const doc = testReplayDoc({ ...twoLives, scoreTimeline: timeline })
    const view = render(<ReplayTeams doc={doc} scoreboard={board()} frame={100} locale="fr" />)
    expect(view.getAllByTitle("Score personnel à l'instant lu")).toHaveLength(1)
    const base = view.getAllByTitle('Frags / morts / assistances du match')
    expect(base).toHaveLength(1)
    expect(base[0].textContent).toBe('1/1/0')
  })

  it('sans calque publié, toutes les fiches gardent les totaux du match', () => {
    const doc = testReplayDoc(twoLives)
    const view = render(<ReplayTeams doc={doc} scoreboard={board()} frame={100} locale="fr" />)
    expect(view.queryAllByTitle("Score personnel à l'instant lu")).toHaveLength(0)
    expect(view.getAllByTitle('Frags / morts / assistances du match')).toHaveLength(2)
  })

  it('EN : les mêmes surfaces portent les libellés anglais', () => {
    const doc = testReplayDoc({ ...twoLives, scoreTimeline: timeline })
    const view = render(<ReplayTeams doc={doc} scoreboard={board()} frame={100} locale="en" />)
    expect(view.getByTitle('Personal score at the moment being played').textContent).toBe('350')
    expect(view.getByTitle('Kills / deaths / assists at the moment being played')).toBeTruthy()
    expect(view.getByTitle('Kills / deaths / assists for the whole match')).toBeTruthy()
  })
})

/**
 * LE GLYPHE D'IDENTITÉ A QUITTÉ LES FICHES (demande utilisateur du 2026-08-25). Ces cas sont le
 * VERROU du retrait, et pas une simple absence constatée : ils montent la situation qui faisait
 * apparaître la marque — le joueur de la page ET un ami dans la même colonne — et vérifient que
 * la colonne n'écrit plus rien. Le glyphe, lui, n'est pas supprimé : il reste au FIL des
 * éliminations (`ReplayFeedName`), où un nom défile au milieu d'autres, et il y est testé.
 */
describe('ReplayTeams — plus aucune marque d’identité sur les fiches', () => {
  it('ni « Moi » ni « Ami », même avec les deux sur le tableau de bord', () => {
    const doc = testReplayDoc({
      roster: [
        { xuid: 'A', filmIndex: 0, name: 'Alpha' },
        { xuid: 'B', filmIndex: 1, name: 'Bravo' },
        { xuid: 'C', filmIndex: 2, name: 'Charlie' },
      ],
      tracks: [TRACK, { ...TRACK, slot: 513, xuid: 'B' }, { ...TRACK, slot: 514, xuid: 'C' }],
    })
    const board = [
      { ...sbRow('A', 'Alpha', 't0'), is_me: true },
      sbRow('B', 'Bravo', 't0'),
      sbRow('C', 'Charlie', 't1'),
    ]
    render(<ReplayTeams doc={doc} scoreboard={board} frame={10} locale="fr" />)
    expect(screen.queryByRole('img', { name: 'Moi' })).toBeNull()
    expect(screen.queryByRole('img', { name: 'Ami' })).toBeNull()
    // Aucun glyphe du tout : ces fiches ne portent ni capacité ni arme, la colonne est muette.
    expect(screen.queryAllByRole('img')).toHaveLength(0)
  })

  it('les noms des trois joueurs restent écrits — c’est la MARQUE qui part, pas l’identité', () => {
    renderTeams({})
    expect(screen.getByText('Alpha')).toBeTruthy()
  })
})

/**
 * LA FICHE UNIQUE (ex-« compacte », devenue LA fiche le 2026-08-24) : armes, munitions de
 * la seule arme en main, grenades et capacité sur UNE rangée en grille à cellules fixes.
 * Ces tests tiennent les règles qui restent : munitions de la main seule, rangée qui ne se
 * replie jamais, et parité de gabarit morte/vivante.
 */
describe('ReplayTeams — la fiche unique', () => {
  const DEUX_ARMES = {
    weaponLabels: {
      '0xAAAA': { fr: 'Fusil', en: 'Rifle' },
      '0xBBBB': { fr: 'Pistolet', en: 'Pistol' },
    },
    loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
  }
  const AMMO = {
    ...DEUX_ARMES,
    inventory: [{ t: 0, slot: 512, d: 1, am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }] }],
  }

  function renderCard(over: Partial<ReplayDocument>, frame = 10) {
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
      tracks: [TRACK],
      ...over,
    })
    return render(<ReplayTeams doc={doc} scoreboard={[]} frame={frame} locale="fr" />)
  }

  /** Le nombre de RANGÉES d'une fiche : les enfants directs de la carte d'Alpha. */
  const rangees = (view: ReturnType<typeof render>) =>
    (view.getByText('Alpha').parentElement?.parentElement as HTMLElement).childElementCount

  it('sans sélecteur lu, AUCUNE munition n’est montrée — jamais une arme désignée au hasard', () => {
    const sansSelecteur = {
      ...DEUX_ARMES,
      inventory: [{ t: 0, slot: 512, am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }] }],
    }
    const vue = renderCard(sansSelecteur)
    expect(vue.container.textContent).not.toContain('10/20')
    expect(vue.container.textContent).not.toContain('5/6')
  })

  it('la fiche porte nom, armes, grenades et capacité sur sa rangée unique', () => {
    const doc = {
      ...AMMO,
      inventory: [
        { t: 0, slot: 512, d: 1, g: [2, 0, 0, 0], am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }] },
      ],
      abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot' } },
      abilities: [{ t: 0, slot: 512, r: 20, src: 'i48' }],
    }
    const vue = renderCard(doc)
    const texte = vue.container.textContent ?? ''
    expect(vue.getByText('Alpha')).toBeTruthy()
    // Les DEUX armes restent (c'est leur MUNITION qui est réduite, pas la rangée d'armes).
    expect(texte).toContain('Fusil')
    expect(texte).toContain('Pistolet')
    expect(texte).toContain('Grappin')
    // Les grenades portées gardent leur compteur.
    expect(texte).toContain('2')
  })

  it('la rangée unique refuse de se replier — sinon la fiche changerait de hauteur', () => {
    const vue = renderCard(AMMO)
    const carte = vue.getByText('Alpha').parentElement?.parentElement as HTMLElement
    const rangee = [...carte.children].find((e) => e.className.includes('flex-nowrap'))
    expect(rangee, 'la rangée armes + inventaire').toBeTruthy()
    expect(rangee!.className).toContain('overflow-hidden')
  })

  it('la fiche MORTE reste lisible : le retour s’affiche', () => {
    const vue = renderCard({}, 140)
    expect(vue.getByText('Réapparition ?')).toBeTruthy()
  })

  /**
   * LA PARITÉ DE GABARIT MORTE/VIVANTE — la règle 1.1 de la fiche. Le nombre de rangées ne
   * dépend JAMAIS de l'état vital : une fiche qui perd une rangée à la mort fait sauter
   * toute la colonne à chaque élimination (retour utilisateur du 2026-08-24 : la fiche
   * morte paraissait plus haute). La mort remplace le CONTENU d'une rangée, jamais la
   * rangée.
   */
  it('la fiche MORTE a EXACTEMENT le gabarit de la vivante', () => {
    const vivante = renderCard({}, 10)
    const attendu = rangees(vivante)
    vivante.unmount()
    const morte = renderCard({}, 140)
    expect(morte.getByText('Réapparition ?')).toBeTruthy()
    expect(rangees(morte)).toBe(attendu)
  })

  /**
   * AUCUNE CLASSE DE HAUTEUR NE SE CONDITIONNE À L'ÉTAT VITAL — le corollaire de la règle
   * ci-dessus, et celui qu'un futur « state.alive ? 'h-6' : 'h-4' » casserait sans toucher
   * au nombre de rangées. Les rangées ont une hauteur FIXE (h-3.5 pour la vitalité,
   * h-[18px] pour les armes) : ce sont les mêmes classes, mortes ou vivantes. Seuls le
   * CONTENU et la couleur changent, et la couleur passe par `style`, pas par une classe.
   */
  it('les classes des rangées sont identiques morte et vivante', () => {
    const classes = (view: ReturnType<typeof render>) =>
      [...((view.getByText('Alpha').parentElement?.parentElement as HTMLElement).children)]
        .map((e) => e.className)
    const vivante = renderCard({}, 10)
    const attendues = classes(vivante)
    vivante.unmount()
    const morte = renderCard({}, 140)
    expect(classes(morte)).toEqual(attendues)
  })
})

describe('ReplayTeams — lecture d’inventaire VIDE (schéma 19)', () => {
  // LE DÉFAUT CORRIGÉ : une lecture vide, étant la plus récente <= T, gagnait contre la
  // lecture PLEINE qui la précédait, et `ReplayInventoryRow` rendait `null` — la ligne
  // entière disparaissait pendant ~20 s. 17,4 % des lectures publiées sont dans ce cas.
  // DEUX types portes : sans selecteur lu, aucun n'est marque « equipe » et l'infobulle de la
  // vignette reste le seul nom (meme convention que les blocs de grenades ci-dessus).
  const LABELS = [
    { en: 'Frag', fr: 'Fragmentation' },
    { en: 'Plasma', fr: 'Plasma' },
  ]

  it('garde l’équipement de la dernière lecture PLEINE, et annonce la mort', () => {
    renderTeams(
      {
        grenadeLabels: LABELS,
        inventory: [
          { t: 0, slot: 512, g: [2, 1] },
          { t: 50, slot: 512, empty: 'dead' },
        ],
      },
      60,
    )
    expect(screen.getByTitle('Fragmentation').textContent).toContain('×2')
    expect(screen.getByText('Mort')).toBeTruthy()
  })

  it('lecture vide que le fil des éliminations n’explique pas : « indisponible », jamais « Mort »', () => {
    renderTeams(
      {
        grenadeLabels: LABELS,
        inventory: [
          { t: 0, slot: 512, g: [2, 1] },
          { t: 50, slot: 512, empty: 'unknown' },
        ],
      },
      60,
    )
    expect(screen.getByText('Inventaire indisponible')).toBeTruthy()
    expect(screen.queryByText('Mort')).toBeNull()
  })

  it('sans AUCUNE lecture pleine antérieure, la ligne dit l’état au lieu de disparaître', () => {
    renderTeams({ grenadeLabels: LABELS, inventory: [{ t: 0, slot: 512, empty: 'dead' }] }, 10)
    expect(screen.getByText('Mort')).toBeTruthy()
  })

  // L'INFOBULLE NE PROMET UN REPORT QUE QUAND IL A LIEU (revue adversariale, constat 4). Elle
  // affirmait « l'équipement affiché est la dernière lecture pleine, lue il y a X » dans TOUS les
  // cas, y compris quand aucune lecture pleine n'existait — et X était alors l'âge de la lecture
  // VIDE, donné pour celui d'un équipement que la fiche ne montrait même pas.
  it('report réel : l’infobulle annonce la lecture pleine, et les DEUX âges — le vide, l’équipement', () => {
    renderTeams(
      {
        grenadeLabels: LABELS,
        inventory: [
          { t: 0, slot: 512, g: [2, 1] },
          { t: 50, slot: 512, empty: 'dead' },
        ],
      },
      60,
    )
    const title = screen.getByText('Mort').getAttribute('title') ?? ''
    expect(title).toContain('l’équipement affiché est la dernière lecture pleine, lue il y a')
    expect(title).not.toContain('aucune lecture d’inventaire avant cet instant')
    // La lecture vide date de 10 frames, l'équipement de 60 : deux durées DIFFÉRENTES, et c'est
    // tout l'objet de la distinction (`InventoryEmptyState.age` contre `InventoryReading.age`).
    const durees = title.match(/\d+[.,]\d s/g) ?? []
    expect(durees).toHaveLength(2)
    expect(durees[0]).not.toBe(durees[1])
  })

  it('aucun report : l’infobulle dit qu’il n’y a aucune lecture avant, et ne promet rien', () => {
    renderTeams({ grenadeLabels: LABELS, inventory: [{ t: 0, slot: 512, empty: 'unknown' }] }, 10)
    const title = screen.getByText('Inventaire indisponible').getAttribute('title') ?? ''
    expect(title).toContain('aucune lecture d’inventaire avant cet instant')
    expect(title).not.toContain('dernière lecture pleine')
    // Un seul âge : celui de la lecture vide. Le second n'existe pas — il n'y a pas
    // d'équipement affiché à dater.
    expect(title.match(/\d+[.,]\d s/g) ?? []).toHaveLength(1)
  })

  it('une lecture vide À VENIR n’affiche pas « Mort » — la mort n’est pas encore survenue', () => {
    // Ronde 2 de la revue adversariale (2026-08-25) : la première lecture du slot est vide et
    // FUTURE (t=50, image 10). Le badge s'affichait avec « lue il y a 0,7 s » pendant que la
    // ligne disait « dans 0,7 s » — une mort affirmée avant qu'elle survienne, jusqu'à ~11 s
    // d'avance sur le film de référence. La ligne se tait sur l'état, comme avant le lot.
    renderTeams({ grenadeLabels: LABELS, inventory: [{ t: 50, slot: 512, empty: 'dead' }] }, 10)
    expect(screen.queryByText('Mort')).toBeNull()
    expect(screen.queryByText('Inventaire indisponible')).toBeNull()
  })
})
