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

import { resolveXuidMeta } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { normalizeCallouts } from './calloutsLayer'
import { buildPlayerMarks } from './playerMarks'
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

describe('ReplayTeams — munitions et sélecteur', () => {
  const TWO_WEAPONS = {
    weaponLabels: {
      '0xAAAA': { fr: 'Fusil', en: 'Rifle' },
      '0xBBBB': { fr: 'Pistolet', en: 'Pistol' },
    },
    loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
  }

  it('les cellules suivent l’ordre des armes : la dégainée d’abord, index d’emplacement gardé', () => {
    const view = renderTeams({
      ...TWO_WEAPONS,
      inventory: [
        { t: 0, slot: 512, d: 1, am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }] },
      ],
    })
    // Seule la cellule DÉGAINÉE porte une infobulle (item 1.2 : le rattachement méthodique
    // emplacement↔arme est sorti de l'écran) : c'est elle qui s'identifie, et son index
    // d'emplacement reste « 1 ».
    const drawn = screen.getByTitle(/DÉGAINÉ/)
    expect(drawn.textContent).toContain('1')
    expect(drawn.textContent).toContain('5/6')
    // L'ordre du DOM suit l'ordre des armes : la dégainée (5/6) précède l'autre (10/20).
    const txt = view.container.textContent ?? ''
    expect(txt.indexOf('5/6')).toBeGreaterThanOrEqual(0)
    expect(txt.indexOf('5/6')).toBeLessThan(txt.indexOf('10/20'))
  })

  it('sélecteur disant « rien de dégainé » : pictogramme « Armes rangées » — une mesure, muette', () => {
    // Décision produit 4 : l'état mesuré D=2 se montre par un dessin discret et UNE
    // infobulle simple, sans jargon de flux ni jeton de texte.
    renderTeams({
      ...TWO_WEAPONS,
      inventory: [{ t: 0, slot: 512, d: 2, am: [{ mag: 10 }, { mag: 5 }] }],
    })
    expect(screen.getByRole('img', { name: 'Armes rangées' })).toBeTruthy()
    expect(screen.queryByText('rangées')).toBeNull()
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

describe('ReplayTeams — zone courante (callouts)', () => {
  const ZONES = normalizeCallouts({
    module: 'ridgeline',
    provenance: 'brut',
    zones: [
      {
        volume_index: 1, name: 'yard', en: 'Yard', fr: 'Cour',
        x: 0, y: 0, z: 0, z_bottom: -1, z_top: 3,
        polygon: [[-2, -2], [2, -2], [2, 2]],
      },
      {
        volume_index: 2, name: 'far', en: 'Far', fr: 'Loin',
        x: 50, y: 50, z: 0, z_bottom: -1, z_top: 3,
        polygon: [[48, 48], [52, 48], [52, 52]],
      },
    ],
  })

  it('affiche la zone du joueur vivant, affectée à sa position', () => {
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
      tracks: [TRACK],
    })
    render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" callouts={ZONES} />)
    expect(screen.getByTitle('Zone de la carte').textContent).toBe('Cour')
  })

  it('mort : la ligne de zone RÉSERVE sa place, vide — la fiche ne se compacte pas', () => {
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
      tracks: [TRACK],
    })
    const cardRows = (view: ReturnType<typeof render>) => {
      const card = view.getByText('Alpha').parentElement?.parentElement as HTMLElement
      return card.childElementCount
    }
    const alive = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" callouts={ZONES} />)
    const aliveRows = cardRows(alive)
    alive.unmount()
    const dead = render(<ReplayTeams doc={doc} scoreboard={[]} frame={140} locale="fr" callouts={ZONES} />)
    expect(dead.queryByTitle('Zone de la carte')).toBeNull()
    expect(cardRows(dead)).toBe(aliveRows)
  })

  it('sans callouts : aucune ligne de zone — pas de place réservée pour rien', () => {
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
      tracks: [TRACK],
    })
    render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" />)
    expect(screen.queryByTitle('Zone de la carte')).toBeNull()
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
    expect(screen.getByText('Réapparition dans')).toBeTruthy()
    expect(screen.getByRole('progressbar')).toBeTruthy()
  })

  it('mort sans vie suivante : « Réapparition ? », jamais un délai deviné, et pas de barre', () => {
    renderTeams({}, 140)
    expect(screen.getByText('Réapparition ?')).toBeTruthy()
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

describe('ReplayTeams — marques d’identité sur les fiches (D5)', () => {
  it('marque « Moi » le joueur de la page et « Ami » un ami, rien pour les autres', () => {
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
    const marks = buildPlayerMarks(board, ['  bRaVo '])
    render(<ReplayTeams doc={doc} scoreboard={board} frame={10} locale="fr" marks={marks} />)
    expect(screen.getByRole('img', { name: 'Moi' })).toBeTruthy()
    expect(screen.getByRole('img', { name: 'Ami' })).toBeTruthy()
    // Charlie n'est ni l'un ni l'autre : deux glyphes en tout sur la page, pas trois.
    expect(screen.getAllByRole('img').length).toBe(2)
  })

  it('sans marques : aucune fiche n’en porte', () => {
    renderTeams({})
    expect(screen.queryByRole('img', { name: 'Moi' })).toBeNull()
    expect(screen.queryByRole('img', { name: 'Ami' })).toBeNull()
  })
})
