/**
 * Tests — ReplayPlayerCard (tuile compacte de la rangée) : ce que la fiche LIT, et comment elle le dit.
 *
 * CE QU'ILS PROTÈGENT : la règle n° 1 des fiches — une valeur non lue s'affiche comme une LACUNE, jamais
 * comme un zéro, une moyenne ou un nom inventé. Sur la tuile compacte, ce qui n'a pas la place d'être écrit
 * passe dans une infobulle, avec sa valeur et l'âge de sa lecture : les cas ci-dessous lisent donc des
 * `title`. Chacun éprouve un étage de la fiche (capacité, grenades, munitions, mort, inventaire vide, compteurs)
 * sur ses deux faces : la mesure s'affiche, l'absence de mesure se dit.
 *
 * Le montage est la rangée des équipes (`ReplayTeamsRow`), qui exige deux camps : Alpha est le joueur de
 * la fiche inspectée, Bravo tient le camp d'en face, loin de toute pose.
 */
import { describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'
import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import { ReplayRowFromBoard } from '../test/replayRowFromBoard'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

/** Les deux types de grenade avec vignette. */
const LABELS_IMG = [
  { en: 'Frag', fr: 'Fragmentation', img: '/static/weapons-assets/halo_infinite/hud/Frag.png', tinted: true },
  { en: 'Plasma', fr: 'Plasma', img: '/static/weapons-assets/halo_infinite/hud/Plasma.png', tinted: true },
]
const LABELS = [
  { en: 'Frag', fr: 'Fragmentation' },
  { en: 'Plasma', fr: 'Plasma' },
]

/** Une vie vivante sur [0,100] pour le slot 512, rattachée au joueur A. */
const TRACK = { slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }] }
const TRACK_B = { slot: 513, team: -1, xuid: 'B', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 90, y: 90 }] }

function docOf(over: Partial<ReplayDocument>) {
  return testReplayDoc({
    roster: [
      { xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', team: 0 },
      { xuid: 'B', filmIndex: 1, seat: 1, name: 'Bravo', team: 1 },
    ],
    tracks: [TRACK, TRACK_B],
    ...over,
  })
}

function renderCard(over: Partial<ReplayDocument>, frame = 10, scoreboard: MatchScoreboardRow[] = [], locale: 'fr' | 'en' = 'fr') {
  return render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={docOf(over)} scoreboard={scoreboard} frame={frame} locale={locale} />)
}

/** La tuile d'Alpha : deux niveaux au-dessus de son nom. */
const tuile = (view: ReturnType<typeof render>) => view.getByText('Alpha').parentElement?.parentElement as HTMLElement
/** Les RANGÉES d'une tuile : ses enfants directs hors couches d'effets (`aria-hidden`). */
const rangees = (t: HTMLElement) => [...t.children].filter((e) => !e.hasAttribute('aria-hidden')) as HTMLElement[]
/** La rangée d'armes : celle qui porte l'âge de la lecture des armes et le texte composé de la main. */
const armes = (view: ReturnType<typeof render>) =>
  within(tuile(view)).getByTitle(/Armes (lues|de la première image-clé)|Weapons (read|from the first keyframe)/)

describe('ReplayPlayerCard — capacité équipée', () => {
  it('nomme la capacité quand la table la connaît et ne sert pas de vignette', () => {
    renderCard({
      abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot' } },
      abilities: [{ t: 0, slot: 512, r: 20, src: 'i48' }],
    })
    expect(screen.getByText('Grappin')).toBeTruthy()
  })
})

describe('ReplayPlayerCard — grenade équipée', () => {
  it('le type équipé se désigne par la SEULE encre warning : aucun fond teinté', () => {
    const vue = renderCard({
      grenadeLabels: LABELS_IMG,
      inventory: [{ t: 0, slot: 512, g: [1, 2], gs: 1 }],
    })
    const cellule = within(tuile(vue)).getByTitle(/^Plasma — .*LU dans le film/)
    expect(cellule.style.color).toContain('warning')
    expect(cellule.style.background).toBe('')
  })
})

/**
 * LE BRANCHEMENT CLIENT DE L'AXE `grenadeReads` (schéma 20).
 *
 * Une inversion COMPLÈTE de la préférence — la cellule retombant sur l'inventaire au lieu de l'axe, donc
 * l'annulation pure et simple du bénéfice de l'axe — laissait la suite verte : aucun test de rendu n'éprouvait
 * la cellule de grenades sur cet axe. Les quatre cas sont les quatre branches de `grenadeBoxAt` : l'axe gagne
 * quand il est PASSÉ, l'artefact ancien retombe sur l'inventaire, une lecture À VENIR ne prime pas une
 * information passée, et une lecture à venir SEULE s'affiche en le disant.
 */
describe('ReplayPlayerCard — cellule de grenades : quelle lecture, et de quel âge', () => {
  it('artefact 20 : la lecture DELTA plus récente que l’image-clé gagne, avec SON âge et son estompage propre', () => {
    const vue = renderCard(
      {
        grenadeLabels: LABELS_IMG,
        inventory: [{ t: 0, slot: 512, g: [2, 0] }],
        grenadeReads: [
          { t: 0, slot: 512, g: [2, 0], src: 'kf' },
          { t: 60, slot: 512, g: [0, 3], src: 'delta' },
        ],
      },
      90,
    )
    const cellule = within(tuile(vue)).getByTitle(/Grenades lues il y a 0\.5 s/)
    expect(cellule.title, 'les compteurs delta, pas ceux de l’image-clé').toContain('Plasma ×3')
    expect(cellule.title).not.toContain('Fragmentation')
    // L'ESTOMPAGE EST CELUI DE LA CELLULE, PAS CELUI DE LA RANGÉE : sinon les deux se MULTIPLIENT et une lecture
    // delta fraîche s'affiche plus pâle que la lecture qu'elle remplace.
    expect(cellule.style.opacity, 'la cellule porte son estompage propre').not.toBe('')
    expect(Number(cellule.style.opacity)).toBeGreaterThan(0)
    expect(Number(cellule.style.opacity)).toBeLessThanOrEqual(1)
    expect((cellule.parentElement as HTMLElement).style.opacity, 'la rangée n’impose plus son âge à la cellule').toBe('')
  })

  it('artefact ≤ 19 (aucun `grenadeReads`) : la cellule retombe sur l’inventaire, âge inchangé', () => {
    const vue = renderCard({ grenadeLabels: LABELS, inventory: [{ t: 0, slot: 512, g: [1, 2] }] }, 60)
    const cellule = within(tuile(vue)).getByTitle(/Grenades lues il y a 1\.0 s/)
    expect(cellule.title).toContain('Fragmentation ×1')
    expect(cellule.title).toContain('Plasma ×2')
  })

  it('lecture de grenades À VENIR : l’information PASSÉE de l’inventaire prime', () => {
    const vue = renderCard(
      {
        grenadeLabels: LABELS,
        inventory: [{ t: 0, slot: 512, g: [2, 0] }],
        grenadeReads: [{ t: 90, slot: 512, g: [0, 5], src: 'delta' }],
      },
      30,
    )
    const cellule = within(tuile(vue)).getByTitle(/Grenades lues il y a 0\.5 s/)
    expect(cellule.title, 'les compteurs de l’inventaire passé').toContain('Fragmentation ×2')
    expect(cellule.title, 'jamais la lecture à venir').not.toContain('×5')
    expect(cellule.title).not.toContain('Plasma')
  })

  it('lecture À VENIR sans RIEN de passé : elle s’affiche, et l’infobulle dit « dans »', () => {
    const vue = renderCard({ grenadeLabels: LABELS, grenadeReads: [{ t: 90, slot: 512, g: [0, 5], src: 'delta' }] }, 30)
    const cellule = within(tuile(vue)).getByTitle(/Grenades lues dans 1\.0 s/)
    expect(cellule.title).toContain('Plasma ×5')
    expect(within(tuile(vue)).queryByTitle(/Grenades lues il y a/)).toBeNull()
  })
})

describe('ReplayPlayerCard — munitions de l’arme en main', () => {
  const TWO_WEAPONS = {
    weaponLabels: { '0xAAAA': { fr: 'Fusil', en: 'Rifle' }, '0xBBBB': { fr: 'Pistolet', en: 'Pistol' } },
    loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
  }

  it('SEULE l’arme dégainée garde ses munitions : `d: 1` désigne l’emplacement 1, celles de l’autre partent', () => {
    const vue = renderCard({
      ...TWO_WEAPONS,
      inventory: [{ t: 0, slot: 512, d: 1, am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }] }],
    })
    expect(armes(vue).title).toContain('Munitions : 5 / 6')
    expect(armes(vue).title).not.toContain('10 / 20')
    expect(vue.container.textContent).not.toContain('5/6')
  })

  it('sans sélecteur lu, AUCUNE munition n’est dite — jamais une arme désignée au hasard', () => {
    const vue = renderCard({
      ...TWO_WEAPONS,
      inventory: [{ t: 0, slot: 512, am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }] }],
    })
    expect(armes(vue).title).not.toContain('Munitions')
    expect(vue.container.textContent).not.toContain('10/20')
    expect(vue.container.textContent).not.toContain('5/6')
  })

  it('arme À CHARGE en main, rien de consommé : charge PLEINE à 100 %, jamais le « 1/0 » parasite du film', () => {
    const vue = renderCard({
      weaponLabels: { '0xAAAA': { fr: 'Épée', en: 'Sword', fx: 'melee' }, '0xBBBB': { fr: 'Pistolet', en: 'Pistol' } },
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
      inventory: [{ t: 0, slot: 512, d: 0, am: [{ mag: 1, res: 0 }, { mag: 5 }] }],
    })
    expect(armes(vue).title).toContain('Charge restante : 100 %')
    expect(armes(vue).title).not.toMatch(/1 ?\/ ?0/)
  })

  it('l’arme en main SANS visuel garde son libellé : jamais l’icône d’une arme voisine', () => {
    const vue = renderCard({
      weaponLabels: {
        '0xAAAA': { fr: 'Pistolet', en: 'Pistol' },
        '0xBBBB': { fr: 'Fusil', en: 'Rifle', img: '/static/weapons-assets/halo_infinite/jeu/contour-01.png', tinted: true },
      },
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
      inventory: [{ t: 0, slot: 512, d: 0, am: [{ mag: 8 }, { mag: 2 }] }],
    })
    expect(within(tuile(vue)).getByText('Pistolet')).toBeTruthy()
    expect(within(tuile(vue)).queryByRole('img', { name: 'Fusil' })).toBeNull()
  })
})

describe('ReplayPlayerCard — mort et réapparition', () => {
  const docAvecRetour = () =>
    docOf({ tracks: [TRACK, TRACK_B, { slot: 514, team: -1, xuid: 'A', startFrame: 180, endFrame: 260, points: [{ t: 180, x: 0, y: 0 }] }] })
  const montee = (frame: number) =>
    render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={docAvecRetour()} scoreboard={[]} frame={frame} locale="fr" />)

  // PLUS DE LISERÉ GAUCHE, ni de PEINTURE de mort sur la couche : c'est la TUILE qui dit la mort, par une
  // bordure symétrique teintée destructive. À l'image 140 l'éclat du coup fatal est encore dans sa fenêtre
  // (mort à 100) : la couche existe, mais ne porte QUE l'éclat — ni cadre, ni fond, donc aucun accent
  // directionnel possible.
  it('aucun liseré gauche : bordure symétrique de la tuile, couche réduite à l’éclat', () => {
    const vue = montee(140)
    expect(tuile(vue).style.borderLeft).toBe('')
    expect(tuile(vue).style.borderColor).toContain('var(--ac-destructive)')
    const couche = tuile(vue).querySelector('.replay-card-fx') as HTMLElement
    expect(couche.className).toContain('replay-flash-death')
    expect(couche.style.boxShadow).toBe('')
    expect(couche.style.backgroundColor).toBe('')
  })

  it('la mort reste DITE : la tuile teintée destructive et le nom à l’encre éteinte ; une fiche vivante sans effet n’a aucune couche', () => {
    const mort = montee(140)
    expect(tuile(mort).style.background).toContain('var(--ac-destructive)')
    expect(mort.getByText('Alpha').className).toContain('text-muted-foreground')
    mort.unmount()
    const vivant = montee(10)
    expect(tuile(vivant).style.background).not.toContain('var(--ac-destructive)')
    expect(tuile(vivant).querySelector('.replay-card-fx')).toBeNull()
    expect(vivant.getByText('Alpha').className).toContain('text-foreground')
  })
})

describe('ReplayPlayerCard — vitalité et lacunes d’un titre sans décodage film', () => {
  it('un document sans AUCUNE vitalité (Point sans sh/hp) : PAS de barres, on n’invente pas une jauge', () => {
    const vue = renderCard({
      titleSlug: 'autre_titre',
      weaponLabels: { '0xAAAA': { fr: 'Fusil', en: 'Rifle' }, '0xBBBB': { fr: 'Pistolet', en: 'Pistol' } },
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
      inventory: [{ t: 0, slot: 512, g: [1, 0] }],
      grenadeLabels: [{ fr: 'Fragmentation', en: 'Frag' }],
    })
    expect(vue.getByText('Alpha')).toBeTruthy()
    expect(vue.queryByLabelText('Bouclier')).toBeNull()
    expect(vue.queryByLabelText('Santé')).toBeNull()
    // Aucune main désignée, aucune capacité lue : rien n'est inventé.
    expect(vue.queryByText('en main')).toBeNull()
    expect(vue.queryByText(/capacité inconnue/)).toBeNull()
  })

  it('24 sièges sans décodage film : cellules vides, aucune barre, aucun zéro, aucun `role=img`', () => {
    const noms = (prefixe: string) => Array.from({ length: 12 }, (_, i) => `${prefixe}${String(i + 1).padStart(2, '0')}`)
    const nord = noms('Nord')
    const sud = noms('Sud')
    const tous = [...nord, ...sud]
    const doc = testReplayDoc({
      frameCount: 300,
      frameIntervalMs: 100,
      originMs: 0,
      roster: tous.map((nom, i) => ({ xuid: nom, filmIndex: i, name: nom, team: nom.startsWith('Sud') ? 1 : 0 })),
      tracks: tous.map((nom, i) => ({ slot: 512 + i, team: -1, xuid: nom, startFrame: 0, endFrame: 300, points: [{ t: 0, x: 1, y: 1 }] })),
    })
    const board = tous.map((nom) =>
      scoreboardRow(nom, nom, nom.startsWith('Sud') ? 't1' : 't0', { kills: 2, deaths: 1, assists: 3 }),
    )
    const vue = render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={board} frame={50} locale="fr" />)
    expect(vue.container.querySelectorAll('.h-\\[31px\\]')).toHaveLength(24)
    expect(vue.queryByLabelText('Bouclier')).toBeNull()
    expect(vue.queryByLabelText('Santé')).toBeNull()
    expect(vue.container.querySelectorAll('[role="img"]')).toHaveLength(0)
    expect(vue.container.querySelectorAll('svg')).toHaveLength(0)
    // Aucun zéro inventé : le tableau n'en porte aucun, une tuile n'en fabrique pas.
    expect(vue.queryByText('0')).toBeNull()
    expect(vue.queryByText(/×|%|\d+ \/ \d+/)).toBeNull()
    expect(vue.queryByTitle(/Grenades|Inventaire|Munitions|Charge/)).toBeNull()
    // Les TROIS cellules fixes de la ligne 3 (arme 48, grenade 14, capacité 16) sont rendues VIDES sur chacun
    // des 24 sièges — donnée absente = cellule vide, jamais un décalage — et aucune autre largeur fixe n'apparaît.
    expect(vue.queryByTitle(/non lue/)).toBeNull()
    const cellules = [...vue.container.querySelectorAll('[style*="width"]')]
      .map((e) => e as HTMLElement)
      .filter((e) => e.style.width !== '' && e.style.width !== '100%')
    expect(cellules.map((e) => e.style.width)).toEqual(Array.from({ length: 24 }, () => ['48px', '14px', '16px']).flat())
    for (const c of cellules) expect(c.textContent).toBe('')
    // Et le triplet, lui, dit le tableau : 2/1/3 sur chaque tuile.
    const triplets = vue.getAllByTitle(/^Frags \/ morts \/ assistances du match/)
    expect(triplets).toHaveLength(24)
    for (const t of triplets) expect(t.textContent).toBe('2/1/3')
    expect(vue.container.textContent).not.toContain('?')
  })
})

describe('ReplayPlayerCard — compteurs : publiés, ou ceux de la base', () => {
  const board = () => [scoreboardRow('A', 'Alpha', 't0'), scoreboardRow('B', 'Bravo', 't1')]
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

  it('ils TIQUENT : à l’image 10, seuls les paliers déjà passés comptent — zéro est une valeur, pas une lacune', () => {
    // Le score de l'infobulle est celui de la fiche : les points par action, gardés par la feuille (350 au total).
    const actions = {
      coverage: { scoreActions: { complete: true } } as never,
      scoreActions: [
        { t: 5, xuid: 'A', name: 'killPersonalScore', points: 120 },
        { t: 40, xuid: 'A', name: 'killPersonalScore', points: 230 },
      ],
    }
    const feuille = [scoreboardRow('A', 'Alpha', 't0', { score: 350 }), scoreboardRow('B', 'Bravo', 't1')]
    const vue = renderCard({ scoreTimeline: timeline, ...actions }, 10, feuille)
    const live = within(tuile(vue)).getByTitle(/^Frags \/ morts \/ assistances à l'instant lu — FDA /)
    expect(live.textContent).toBe('1/0/0')
    expect(live.title).toContain("Score personnel à l'instant lu : 120")
  })

  it('sans calque publié, toutes les fiches gardent les totaux du match, sans score personnel', () => {
    const vue = renderCard({}, 100, board())
    expect(vue.queryAllByTitle(/Score personnel/)).toHaveLength(0)
    expect(vue.getAllByTitle(/^Frags \/ morts \/ assistances du match — FDA /)).toHaveLength(2)
  })

  /**
   * LE FOND DU TRIPLET EST UNE LECTURE DU FDA, pas un ornement : il suit la MÊME horloge que les nombres qu'il
   * colore. Sur un joueur publié il change AVEC la lecture — c'est le cas qui distingue un fond calculé d'un fond
   * figé au score final.
   */
  it("le fond du triplet suit le FDA à l'instant lu, et CHANGE de palier avec la lecture", () => {
    // Image 20 : 1 frag, 2 morts, 0 assistance -> FDA = −1 -> palier déficitaire.
    const tot = renderCard({ scoreTimeline: timeline }, 20, board())
    const deficit = within(tuile(tot)).getByTitle(/^Frags \/ morts \/ assistances à l'instant lu — FDA /)
    // Le signe négatif vient de l'Intl de la plateforme (tiret ou moins typographique) : ce qui est vérifié ici,
    // c'est la VALEUR et son signe, pas le glyphe.
    expect(deficit.title).toMatch(/FDA .1,00/)
    expect(deficit.style.background).toContain('var(--ac-destructive)')
    tot.unmount()
    // Image 100 : 3 frags, 2 morts, 4 assistances -> FDA = 3 + 4/3 − 2 = 2,33 -> bénéficiaire.
    const gain = within(tuile(renderCard({ scoreTimeline: timeline }, 100, board()))).getByTitle(/^Frags \/ morts \/ assistances à l'instant lu — FDA /)
    expect(gain.title).toContain('2,33')
    expect(gain.style.background).toContain('var(--ac-success)')
  })
})

describe('ReplayPlayerCard — plus aucune marque d’identité', () => {
  it('ni « Moi » ni « Ami », même avec les deux sur le tableau de bord : seule la MARQUE part, pas le nom', () => {
    const doc = testReplayDoc({
      roster: [
        { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
        { xuid: 'B', filmIndex: 1, name: 'Bravo', team: 0 },
        { xuid: 'C', filmIndex: 2, name: 'Charlie', team: 1 },
      ],
      tracks: [TRACK, { ...TRACK, slot: 513, xuid: 'B' }, { ...TRACK, slot: 514, xuid: 'C' }],
    })
    const board = [
      { ...scoreboardRow('A', 'Alpha', 't0'), is_me: true },
      scoreboardRow('B', 'Bravo', 't0'),
      scoreboardRow('C', 'Charlie', 't1'),
    ]
    render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={board} frame={10} locale="fr" />)
    expect(screen.queryByRole('img', { name: 'Moi' })).toBeNull()
    expect(screen.queryByRole('img', { name: 'Ami' })).toBeNull()
    // Ces fiches ne portent ni capacité ni arme : aucun glyphe du tout.
    expect(screen.queryAllByRole('img')).toHaveLength(0)
    expect(screen.getByText('Alpha')).toBeTruthy()
  })
})

describe('ReplayPlayerCard — la rangée de la fiche ne se replie pas et se laisse rétrécir', () => {
  it('la rangée armes + inventaire est `flex-nowrap overflow-hidden` : sinon la fiche changerait de hauteur', () => {
    const vue = renderCard({
      weaponLabels: { '0xAAAA': { fr: 'Fusil', en: 'Rifle' }, '0xBBBB': { fr: 'Pistolet', en: 'Pistol' } },
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
      inventory: [{ t: 0, slot: 512, d: 1, am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }] }],
    })
    const rangee = tuile(vue).querySelector('.flex-nowrap')
    expect(rangee, 'la rangée armes + inventaire').toBeTruthy()
    expect(rangee!.className).toContain('overflow-hidden')
  })

  // JSDOM NE MET RIEN EN PAGE : la mesure qui a établi le défaut (un contenu qui déborde sa rangée, coupé en
  // silence par le `overflow-hidden` du parent) est irreproductible ici. Ce test fixe donc la CAUSE : la rangée
  // d'inventaire est un élément flex à `min-width: auto` sans `min-w-0`, et prend alors sa largeur hypothétique.
  it('la rangée d’inventaire porte `min-w-0` : sans lui, un texte long déborde la fiche', () => {
    const vue = renderCard({ grenadeLabels: LABELS, inventory: [{ t: 0, slot: 512, g: [1, 2] }] })
    const cellule = within(tuile(vue)).getByTitle(/Grenades lues/)
    expect((cellule.parentElement as HTMLElement).className).toContain('min-w-0')
    expect(rangees(tuile(vue))).toHaveLength(2)
  })
})

/**
 * LECTURE D'INVENTAIRE VIDE (schéma 19). Le défaut corrigé : une lecture vide, étant la plus récente <= T,
 * gagnait contre la lecture PLEINE qui la précédait, et la ligne entière disparaissait pendant ~20 s (17,4 %
 * des lectures publiées sont dans ce cas). Sur la tuile compacte le constat se dit dans l'infobulle de l'arme.
 */
describe('ReplayPlayerCard — lecture d’inventaire VIDE (schéma 19)', () => {
  const ARMES = {
    weaponLabels: { '0xAAAA': { fr: 'Fusil', en: 'Rifle' }, '0xBBBB': { fr: 'Pistolet', en: 'Pistol' } },
    loadouts: [{ t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
    grenadeLabels: LABELS,
  }

  it('garde l’équipement de la dernière lecture PLEINE, et annonce la mort', () => {
    const vue = renderCard(
      { ...ARMES, inventory: [{ t: 0, slot: 512, g: [2, 1] }, { t: 50, slot: 512, empty: 'dead' }] },
      60,
    )
    expect(within(tuile(vue)).getByTitle(/Fragmentation ×2/)).toBeTruthy()
    expect(armes(vue).title).toContain('Mort')
  })

  it('lecture vide que le fil des éliminations n’explique pas : « indisponible », jamais « Mort »', () => {
    const vue = renderCard(
      { ...ARMES, inventory: [{ t: 0, slot: 512, g: [2, 1] }, { t: 50, slot: 512, empty: 'unknown' }] },
      60,
    )
    expect(armes(vue).title).toContain('Inventaire indisponible')
    expect(armes(vue).title).not.toContain('Mort')
  })

  it('sans AUCUNE lecture pleine antérieure, la fiche dit l’état au lieu de se taire', () => {
    const vue = renderCard({ ...ARMES, inventory: [{ t: 0, slot: 512, empty: 'dead' }] }, 10)
    expect(armes(vue).title).toContain('Mort')
  })

  it('aucun report : l’infobulle dit qu’il n’y a aucune lecture avant, et ne promet rien', () => {
    const vue = renderCard({ ...ARMES, inventory: [{ t: 0, slot: 512, empty: 'unknown' }] }, 10)
    const title = armes(vue).title
    expect(title).toContain('aucune lecture d’inventaire avant cet instant')
    expect(title).not.toContain('dernière lecture pleine')
    // Un seul âge de lecture vide, et AUCUN âge d'équipement à dater : il n'y a pas d'équipement affiché. (Le titre
    // porte en plus l'âge des armes, « Armes lues il y a », qui est une autre lecture.)
    expect(title.match(/\blue il y a/g) ?? []).toHaveLength(1)
  })

  it('une lecture vide À VENIR n’affiche pas « Mort » — la mort n’est pas encore survenue', () => {
    const vue = renderCard({ ...ARMES, inventory: [{ t: 50, slot: 512, empty: 'dead' }] }, 10)
    expect(within(tuile(vue)).queryByTitle(/Mort/)).toBeNull()
    expect(within(tuile(vue)).queryByTitle(/Inventaire indisponible/)).toBeNull()
  })
})
