/**
 * ReplayPlayerCard.compact.test.tsx — LA TUILE COMPACTE d'une Grande équipe (gates 2 et 3 du
 * plan fiches compactes 2026-09-06) : un roster 12v12 et un 7v7 sous un en-tête
 * `mode_category: 'BTB'`, et la liste FERMÉE des `it` du plan — (a) à (k) pour le gate 2, puis
 * les reports en infobulle pour le gate 3.
 *
 * LE DOCUMENT EST RICHE À DESSEIN, comme la fixation 4v4 : chaque siège nommé porte UN cas du
 * gate (mort avec retour, mort hors film, écran occultant, champ de réparation, capacité hors
 * table, compteur manquant, porteur de drapeau, vie sans mesure, nom long), et les sièges de
 * remplissage sont vivants et équipés. Le 4v4, lui, est protégé ailleurs (`ReplayTeams.dom4v4`) :
 * ce fichier ne dit rien du gabarit normal, sauf que ses cotes n'apparaissent pas ici.
 */
import { describe, expect, it } from 'vitest'
import { render, within } from '@testing-library/react'

import type { ReplayDocument, ReplayInventory } from '@/lib/api/types'

import { ReplayTeams } from './ReplayTeams'
import { fichierNomme, lire } from '../test/featureFiles'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'
import { formatSeconds, frameToMs } from '../../../lib/replay/replayLogic'

/** L'image lue : Echo et Foxtrot sont morts (vies closes à 5), tout le reste est en vie. */
const FRAME = 100

const CAMP_A = [
  'Alpha', 'Bravo', 'Charlie', 'Delta', 'Echo', 'Foxtrot', 'Golf', 'Hotel', 'India', 'Juliett',
  'Kilo', 'Longgamertagname1234',
] as const
const CAMP_B = [
  'Mike', 'November', 'Oscar', 'Papa', 'Quebec', 'Romeo', 'Sierra', 'Tango', 'Uniform', 'Victor',
  'Whiskey', 'Xray',
] as const

const ICONE_FUSIL = '/static/weapons-assets/halo_infinite/jeu/contour-01.png'
const ICONE_PISTOLET = '/static/weapons-assets/halo_infinite/jeu/contour-02.png'
const ICONE_FRAG = '/static/weapons-assets/halo_infinite/hud/Frag.png'
const ICONE_GRAPPIN = '/static/weapons-assets/halo_infinite/hud/Grapple.png'

type Point = NonNullable<NonNullable<ReplayDocument['tracks']>[number]['points']>[number]

/** Une vie du slot donné, sur [start, end], rattachée au joueur `xuid`. */
function vie(slot: number, xuid: string, points: Point[], start = 0, end = 300) {
  return { slot, team: -1, xuid, startFrame: start, endFrame: end, points }
}

const EN_TETE_BTB = { start_time: '2026-07-24T20:00:00Z', mode_category: 'BTB' }

/** Le loadout d'un siège de remplissage : Juliett tient une ÉPÉE (arme à charge), Quebec est lu À VENIR. */
function loadoutDe(nom: string, slot: number) {
  if (nom === 'Juliett') return { t: 0, slot, w: ['0xCCCC', '0xAAAA'] }
  if (nom === 'Quebec') return { t: 150, slot, w: ['0xAAAA', '0xBBBB'] }
  return { t: 0, slot, w: ['0xAAAA', '0xBBBB'] }
}

/**
 * L'inventaire d'un siège de remplissage — un cas de report par siège (gate 3) :
 *  - Juliett : charge consommée à 25 % → « Charge restante : 75 % » ;
 *  - Hotel : emplacement de la main jamais écrit → « Munitions pleines » ;
 *  - India : armes RANGÉES (D=2) → aucune munition ;
 *  - Charlie : sélecteur NON LU, deux types de grenade → « dégainée ? » et « sél. ? » ;
 *  - Oscar : lecture pleine, puis lecture VIDE corroborée « mort » à l'image 50 ;
 *  - Papa : sélecteur non lu, un seul type → « dégainée ? », sélection DÉDUITE ;
 *  - Quebec : première lecture À VENIR (image 150) → âges négatifs, « dans X s ».
 */
function inventaireDe(nom: string, slot: number): ReplayInventory[] {
  switch (nom) {
    case 'Juliett':
      return [{ t: 0, slot, d: 0, am: [{ gauge: 0.25 }, { mag: 7 }], g: [1, 0] }]
    case 'Hotel':
      return [{ t: 0, slot, d: 0, am: [{}, { mag: 3 }], g: [1, 0] }]
    case 'India':
      return [{ t: 0, slot, d: 2, am: [{ mag: 8 }, { mag: 2 }], g: [1, 0] }]
    case 'Charlie':
      return [{ t: 0, slot, am: [{ mag: 4 }, { mag: 1 }], g: [1, 1] }]
    case 'Oscar':
      return [{ t: 0, slot, d: 0, am: [{ mag: 8 }, { mag: 2 }], g: [1, 0] }, { t: 50, slot, empty: 'dead' }]
    case 'Papa':
      return [{ t: 0, slot, am: [{ mag: 8 }, { mag: 2 }], g: [1, 0] }]
    case 'Quebec':
      return [{ t: 150, slot, d: 0, am: [{ mag: 8 }, { mag: 2 }], g: [1, 0] }]
    default:
      return [{ t: 0, slot, d: 0, am: [{ mag: 8 }, { mag: 2 }], g: [1, 0] }]
  }
}

/** Le document de la Grande équipe : `parCamp` sièges par camp (les premiers de chaque liste). */
function documentBTB(parCamp: 12 | 7) {
  const a = CAMP_A.slice(0, parCamp)
  const b = CAMP_B.slice(0, parCamp)
  const noms: string[] = [...a, ...b]
  // Slot d'un siège de remplissage : 600 + son index dans la liste.
  const slotDe = (nom: string) => 600 + noms.indexOf(nom)
  const remplissage = noms.filter(
    (n) => !['Alpha', 'Bravo', 'Echo', 'Foxtrot', 'Golf', 'Kilo'].includes(n),
  )
  // Une trace dont le joueur n'est ni au roster ni au tableau fabriquerait un siège de plus :
  // les sièges nommés ne sont posés que s'ils font partie de l'effectif demandé.
  const present = (nom: string) => noms.includes(nom)
  // Kilo : aucune mesure de vitalité sur SA vie (le document en porte) — plein d'apparition.
  const kilo = present('Kilo') ? [vie(523, 'Kilo', [{ t: 0, x: 0, y: 9 }])] : []
  const equipeKilo = present('Kilo') ? [523] : []
  return testReplayDoc({
    roster: noms.map((nom, i) => ({ xuid: nom, filmIndex: i, name: nom })),
    tracks: [
      // Alpha : bouclier entamé lu à l'image 80.
      vie(512, 'Alpha', [{ t: 0, x: 1, y: 1, sh: 1, hp: 1 }, { t: 80, x: 2, y: 2, sh: 0.6, hp: 1 }]),
      // Bravo : vivant, AUCUNE donnée d'équipement.
      vie(513, 'Bravo', [{ t: 0, x: 4, y: 4, sh: 1, hp: 1 }]),
      // Echo : mort à l'image 5, retour lu à 180 — sa dernière position est sous l'écran.
      vie(516, 'Echo', [{ t: 0, x: 5, y: 0, sh: 1, hp: 1 }], 0, 5),
      vie(517, 'Echo', [{ t: 180, x: 9, y: 1 }], 180, 300),
      // Foxtrot : mort à l'image 5, aucune vie suivante — hors film.
      vie(518, 'Foxtrot', [{ t: 0, x: 9, y: 9, sh: 1, hp: 1 }], 0, 5),
      // Golf : capacité de rang hors table.
      vie(519, 'Golf', [{ t: 0, x: 6, y: 6, sh: 0.5, hp: 0.5 }]),
      ...kilo,
      // Charlie sous l'écran occultant (5,0), Delta dans le champ de réparation (8,8) ; les
      // autres sièges de remplissage loin de toute pose.
      ...remplissage.map((nom) =>
        vie(slotDe(nom), nom, [
          nom === 'Charlie'
            ? { t: 0, x: 5, y: 0, sh: 0.2, hp: 0.5 }
            : nom === 'Delta'
              ? { t: 0, x: 8, y: 8, sh: 1, hp: 1 }
              : { t: 0, x: 2, y: 5, sh: 0.7, hp: 1 },
        ]),
      ),
    ],
    weaponLabels: {
      '0xAAAA': { fr: 'Fusil', en: 'Rifle', img: ICONE_FUSIL, tinted: true },
      '0xBBBB': { fr: 'Pistolet', en: 'Pistol', img: ICONE_PISTOLET, tinted: true },
      '0xCCCC': { fr: 'Épée', en: 'Sword', fx: 'melee' },
    },
    loadouts: [
      { t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] },
      { t: 0, slot: 519, w: ['0xAAAA', '0xBBBB'] },
      ...equipeKilo.map((slot) => ({ t: 0, slot, w: ['0xAAAA', '0xBBBB'] })),
      ...remplissage.map((nom) => loadoutDe(nom, slotDe(nom))),
    ],
    grenadeLabels: [
      { fr: 'Fragmentation', en: 'Frag', img: ICONE_FRAG, tinted: true },
      { fr: 'Plasma', en: 'Plasma' },
    ],
    inventory: [
      // Alpha : main lue, sélection de grenade LUE (la Fragmentation, qui a une vignette).
      { t: 0, slot: 512, d: 0, am: [{ mag: 10, res: 20 }, { mag: 5, res: 6 }], g: [1, 2], gs: 0 },
      { t: 0, slot: 519, d: 0, am: [{ mag: 8 }, { mag: 2 }], g: [1, 0] },
      ...equipeKilo.map((slot) => ({ t: 0, slot, d: 0, am: [{ mag: 8 }, { mag: 2 }], g: [1, 0] })),
      ...remplissage.flatMap((nom) => inventaireDe(nom, slotDe(nom))),
    ],
    abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot', img: ICONE_GRAPPIN, tinted: true } },
    abilities: [
      { t: 0, slot: 512, r: 20, src: 'i48' },
      // Golf : rang hors table — glyphe neutre, rang dans l'infobulle.
      { t: 0, slot: 519, r: 9, src: 'i48' },
      // Delta : un grappin sans lecture de charges depuis le ramassage — « plein ».
      ...(present('Delta') ? [{ t: 0, slot: slotDe('Delta'), r: 20, src: 'i48' }] : []),
    ],
    // Le canal des charges a parlé pour Alpha (2 restantes à l'image 30) ; Delta n'a rien reçu.
    abilityCharges: [{ t: 30, slot: 512, family: 'grapple', charges: 2 }],
    scoreTimeline: {
      teams: null,
      players: [
        {
          xuid: 'Alpha',
          score: { rounds: null, total: [{ t: 5, v: 120 }, { t: 40, v: 350 }] },
          kills: { rounds: null, total: [{ t: 5, v: 1 }, { t: 40, v: 3 }] },
          deaths: { rounds: null, total: [{ t: 20, v: 2 }] },
          assists: { rounds: null, total: [{ t: 40, v: 4 }] },
        },
      ],
    },
    equipmentPlacements: [
      { family: 'shroud_screen', id: '0x5eeb1a13', owner: 999, origin: 'deployed', t0: 10, t1: 200, x: 5, y: 0 },
      { family: 'repair_field', id: '0x32d97758', owner: 998, origin: 'deployed', t0: 0, t1: 150, x: 8, y: 8 },
    ],
    flagCarries: [{ team: 0, spans: [{ state: 'carried', xuid: 'India', t0: 60, t1: 140, x: 3, y: 7 }] }],
  })
}

function tableauBTB(parCamp: 12 | 7) {
  const lignes = [
    ...CAMP_A.slice(0, parCamp).map((nom) =>
      // Hotel : une assistance NON LUE dans la base — le triplet dit « ? ».
      scoreboardRow(nom, nom, 't0', nom === 'Hotel' ? { assists: null } : {}),
    ),
    ...CAMP_B.slice(0, parCamp).map((nom) =>
      // Mike : déficitaire (0 / 3 / 0) — le palier `destructive` du fond FDA.
      scoreboardRow(nom, nom, 't1', nom === 'Mike' ? { kills: 0, deaths: 3, assists: 0 } : {}),
    ),
  ]
  return lignes
}

function renderBTB(parCamp: 12 | 7, locale: 'fr' | 'en' = 'fr') {
  const doc = documentBTB(parCamp)
  const vue = render(
    <ReplayTeams doc={doc} scoreboard={tableauBTB(parCamp)} frame={FRAME} locale={locale} header={EN_TETE_BTB} />,
  )
  return { vue, doc }
}

/** La tuile d'un siège : deux niveaux au-dessus du nom (la profondeur que dix tests exploitent). */
function tuile(vue: ReturnType<typeof render>, nom: string): HTMLElement {
  return vue.getByText(nom).parentElement?.parentElement as HTMLElement
}

/** Les RANGÉES d'une tuile : ses enfants directs hors couches d'effets (`aria-hidden`). */
function rangees(t: HTMLElement): HTMLElement[] {
  return [...t.children].filter((e) => !e.hasAttribute('aria-hidden')) as HTMLElement[]
}

describe('ReplayPlayerCard — la tuile compacte (mode_category BTB) : gate 2', () => {
  it('(a) hauteur ET classes des rangées identiques vivant/mort — 24 tuiles 12v12, 14 tuiles 7v7', () => {
    const { vue } = renderBTB(12)
    // Un seul gabarit sur toute la colonne : 24 corps de 31, aucun corps de 35.
    expect(vue.container.querySelectorAll('.h-\\[31px\\]')).toHaveLength(24)
    expect(vue.container.querySelectorAll('.h-\\[35px\\]')).toHaveLength(0)
    const vivante = rangees(tuile(vue, 'Alpha'))
    const morte = rangees(tuile(vue, 'Echo'))
    expect(vivante).toHaveLength(2)
    expect(vivante.map((e) => e.className)).toEqual(morte.map((e) => e.className))
    // Ligne 1 bridée à 14 px ; corps fixe 31 px ; tuile à marge 6 et rayon 6.
    expect(vivante[0].className).toContain('leading-[14px]')
    expect(vivante[1].className).toContain('h-[31px]')
    expect(vivante[1].className).toContain('overflow-hidden')
    expect(tuile(vue, 'Alpha').className).toContain('rounded-md border px-1.5 py-1.5')
    expect(tuile(vue, 'Echo').className).toBe(tuile(vue, 'Alpha').className)
    vue.unmount()
    const sept = renderBTB(7).vue
    expect(sept.container.querySelectorAll('.h-\\[31px\\]')).toHaveLength(14)
    expect(sept.container.innerHTML).toContain('auto-fill')
  })

  it('(b) le nom occupe seul la ligne 1, `title` = nom complet, et l’infobulle de la TUILE reste atteignable sur cette ligne (D6)', () => {
    const { vue } = renderBTB(12)
    const nom = vue.getByText('Longgamertagname1234')
    const ligne = nom.parentElement as HTMLElement
    expect(ligne.children).toHaveLength(1)
    expect(nom.title).toBe('Longgamertagname1234')
    expect(nom.className).toContain('truncate')
    expect(nom.className).not.toContain('flex-1')
    // Charlie est sous l'écran occultant : la tuile porte la phrase, et la ligne du nom ne la
    // masque pas — le plus proche ancêtre titré de la ligne est la tuile elle-même.
    const charlie = tuile(vue, 'Charlie')
    expect(charlie.title).toContain('écran occultant')
    const ligneCharlie = vue.getByText('Charlie').parentElement as HTMLElement
    expect(ligneCharlie.hasAttribute('title')).toBe(false)
    expect(ligneCharlie.closest('[title]')).toBe(charlie)
  })

  it('(c) triplet présent en vie ET en mort (dans le corps), « ? » sans fond quand un compteur manque, fond FDA à trois paliers', () => {
    const { vue } = renderBTB(12)
    // Vivant, publié : le triplet est sur la ligne des jauges (corps), pas sur la ligne du nom.
    const alpha = tuile(vue, 'Alpha')
    const tripletAlpha = within(alpha).getByTitle(/FDA 2,33/)
    expect(rangees(alpha)[1].contains(tripletAlpha)).toBe(true)
    expect(rangees(alpha)[0].contains(tripletAlpha)).toBe(false)
    expect(tripletAlpha.style.background).toContain('var(--ac-success)')
    expect(tripletAlpha.textContent).toBe('3/2/4')
    // Mort : le triplet des totaux de la base, dans l'encadré « Éliminé » du corps.
    const echo = tuile(vue, 'Echo')
    const tripletEcho = within(echo).getByTitle(/FDA 0,00/)
    expect(rangees(echo)[1].contains(tripletEcho)).toBe(true)
    expect(tripletEcho.style.background).toContain('var(--ac-info)')
    expect(within(echo).getByText('Éliminé')).toBeTruthy()
    // Un compteur non lu : « ? », aucun fond — une couleur est une affirmation.
    const hotel = within(tuile(vue, 'Hotel')).getByTitle('Frags / morts / assistances du match')
    expect(hotel.textContent).toBe('1/1/?')
    expect(hotel.style.background).toBe('')
    // Déficitaire : le palier destructive.
    const mike = within(tuile(vue, 'Mike')).getByTitle(/FDA/)
    expect(mike.style.background).toContain('var(--ac-destructive)')
  })

  it('(d) cellules rendues VIDES quand la donnée manque — un seul `role=img`, la lacune de capacité', () => {
    const { vue } = renderBTB(12)
    const bravo = tuile(vue, 'Bravo')
    // FUSION feat/v75 (correctif P0-2 du lot « vies anonymes », 2026-09-06) : le SEUL `role=img`
    // de la tuile est la LACUNE de capacité. La fixation avait été prise avant ce correctif,
    // quand la cellule de capacité ne rendait rien du tout sur une vie sans lecture. Elle rend
    // désormais le même glyphe neutre que les armes non lues (`ReplayAbilityCell`, gardé par
    // `doc.abilities.length > 0`) : une lacune DÉCLARÉE, jamais une donnée d'une vie précédente.
    const glyphes = [...bravo.querySelectorAll('[role="img"]')] as HTMLElement[]
    expect(glyphes.map((g) => g.getAttribute('aria-label'))).toEqual([
      'capacité non lue sur cette vie',
    ])
    // La cellule d'arme, vide, à 48 px : la grille des tuiles ne bouge pas.
    const cellulesVides = [...bravo.querySelectorAll('span[aria-hidden]')] as HTMLElement[]
    expect(cellulesVides.map((c) => c.style.width)).toContain('48px')
    expect(within(bravo).getByTitle('armes non lues sur cette vie')).toBeTruthy()
  })

  it('(e) grenades non lues → rien : ni compte, ni vignette, ni infobulle de stock', () => {
    const { vue } = renderBTB(12)
    const bravo = tuile(vue, 'Bravo')
    expect(within(bravo).queryByText(/×/)).toBeNull()
    expect(within(bravo).queryByTitle(/Grenades lues/)).toBeNull()
    expect(within(bravo).queryByText('sél. ?')).toBeNull()
  })

  it('(f) capacité hors table → glyphe neutre de 16 px, rang dans le `title`', () => {
    const { vue } = renderBTB(12)
    const golf = tuile(vue, 'Golf')
    const glyphe = within(golf).getByRole('img', { name: 'capacité non identifiée (rang 9)' })
    expect(glyphe.title).toContain('rang 9')
    expect(glyphe.querySelector('svg')?.getAttribute('width')).toBe('16')
    // Et la capacité connue d'Alpha : la vignette du HUD à 16 px.
    const grappin = within(tuile(vue, 'Alpha')).getByRole('img', { name: 'Grappin' }) as HTMLElement
    expect(grappin.style.width).toBe('16px')
  })

  it('(g) aucune barre sans `sh`/`hp` dans le document ; 100 % sans mesure sur la vie ; jauges 4 / 2 px en `flex-1`', () => {
    // Un titre sans décodage film : le document ne porte ni bouclier ni santé — aucune jauge.
    const sansVitalite = testReplayDoc({
      roster: [{ xuid: 'Alpha', filmIndex: 0, name: 'Alpha' }],
      tracks: [vie(512, 'Alpha', [{ t: 0, x: 0, y: 0 }])],
    })
    const nue = render(
      <ReplayTeams doc={sansVitalite} scoreboard={[scoreboardRow('Alpha', 'Alpha', 't0')]} frame={FRAME} locale="fr" header={EN_TETE_BTB} />,
    )
    expect(nue.queryByLabelText('Bouclier')).toBeNull()
    expect(nue.queryByLabelText('Santé')).toBeNull()
    expect(nue.container.querySelectorAll('.h-\\[31px\\]')).toHaveLength(1)
    nue.unmount()
    // Le document porte la vitalité mais la vie de Kilo n'a aucune mesure : barres PLEINES.
    const { vue } = renderBTB(12)
    const kilo = tuile(vue, 'Kilo')
    const bouclier = within(kilo).getByLabelText('Bouclier')
    const sante = within(kilo).getByLabelText('Santé')
    expect((bouclier.firstElementChild as HTMLElement).style.width).toBe('100%')
    expect((sante.firstElementChild as HTMLElement).style.width).toBe('100%')
    expect(bouclier.style.height).toBe('4px')
    expect(sante.style.height).toBe('2px')
    expect((bouclier.parentElement as HTMLElement).className).toContain('flex-1')
    // Et une mesure reste une mesure : le bouclier d'Alpha à 60 %.
    const alpha = within(tuile(vue, 'Alpha')).getByLabelText('Bouclier')
    expect((alpha.firstElementChild as HTMLElement).style.width).toBe('60%')
  })

  it('(h) « hors film » : le repère seul avec le triplet, ligne 2 vide, « ne revient plus » en infobulle de l’encadré ; le retour lu garde son décompte et son `title`', () => {
    const { vue, doc } = renderBTB(12)
    const foxtrot = tuile(vue, 'Foxtrot')
    const repere = within(foxtrot).getByText('Hors film')
    expect(within(foxtrot).queryByText('ne revient plus')).toBeNull()
    const encadre = repere.parentElement?.parentElement as HTMLElement
    expect(encadre.title).toBe('ne revient plus')
    expect(within(foxtrot).queryByTitle('Réapparition dans')).toBeNull()
    expect(within(foxtrot).getByTitle(/FDA/)).toBeTruthy()
    // Deux lignes dans le corps : la seconde est vide.
    expect(encadre.children).toHaveLength(2)
    expect(encadre.children[1].textContent).toBe('')
    // Echo revient à 180 : le décompte lu, sous « Réapparition dans », en bas à droite.
    const echo = tuile(vue, 'Echo')
    const decompte = within(echo).getByTitle('Réapparition dans')
    expect(decompte.textContent).toBe(formatSeconds(frameToMs(180 - FRAME, doc)))
    expect((decompte.parentElement as HTMLElement).className).toContain('justify-end')
    expect(vue.queryByRole('progressbar')).toBeNull()
  })

  it('(i) DEUX éclairs et TROIS croix en compacte (les tests existants restent à 3 / 3 sur le gabarit normal)', () => {
    const { vue } = renderBTB(12)
    const eclairs = [...tuile(vue, 'Charlie').querySelectorAll('.replay-zone-bolt')] as HTMLElement[]
    expect(eclairs).toHaveLength(2)
    for (const e of eclairs) expect(e.style.animationDelay.startsWith('-')).toBe(true)
    expect(tuile(vue, 'Delta').querySelectorAll('.replay-zone-cross')).toHaveLength(3)
  })

  it('(j) la fiche morte ne porte ni cadre, ni zone, ni filigrane — même morte sous l’écran', () => {
    const { vue } = renderBTB(12)
    const echo = tuile(vue, 'Echo')
    expect(echo.querySelector('.replay-card-fx')).toBeNull()
    expect(echo.querySelector('.replay-zone-cloud')).toBeNull()
    expect(echo.querySelector('.replay-zone-bolt')).toBeNull()
    expect(echo.querySelector('.replay-zone-cross')).toBeNull()
    expect(echo.querySelector('.replay-zone-sensor')).toBeNull()
    expect(echo.querySelector('svg')).toBeNull()
    expect(echo.style.boxShadow).toBe('')
    expect(echo.hasAttribute('title')).toBe(false)
  })

  it('(k) le filigrane de porteur est présent, à 34 px', () => {
    const { vue } = renderBTB(12)
    const india = tuile(vue, 'India')
    expect(india.querySelector('svg[width="34"]')).not.toBeNull()
    expect(vue.container.querySelector('svg[width="46"]')).toBeNull()
  })

  it('(l) 4.6 — les couches d’effets épousent le rayon de la tuile compacte (`rounded-md`), aucun `rounded-lg` dans la colonne', () => {
    const { vue } = renderBTB(12)
    // La tuile est en `rounded-md` ; une couche `inset-0` en `rounded-lg` (8 px) déborderait ses coins (6 px).
    expect(vue.container.innerHTML).not.toContain('rounded-lg')
    // La couche SOUS le contenu (Charlie : le voile de l'écran occultant).
    const charlie = tuile(vue, 'Charlie')
    const sous = charlie.querySelector('.replay-card-fx') as HTMLElement
    expect(sous.className).toContain('absolute inset-0 rounded-md')
    // L'incrustation AU-DESSUS (le nuage et les éclairs de Charlie, les croix de Delta).
    const incrustation = charlie.querySelector('.replay-zone-cloud')?.parentElement as HTMLElement
    expect(incrustation.className).toBe('pointer-events-none absolute inset-0 overflow-hidden rounded-md')
    const croix = tuile(vue, 'Delta').querySelector('.replay-zone-cross')?.parentElement as HTMLElement
    expect(croix.className).toBe('pointer-events-none absolute inset-0 overflow-hidden rounded-md')
    // Le filigrane de porteur (India).
    const filigrane = tuile(vue, 'India').querySelector('svg[width="34"]')?.parentElement as HTMLElement
    expect(filigrane.className).toContain('overflow-hidden rounded-md')
    // Les deux couches que ce document ne pose pas (anneau du capteur, fourreau de
    // translocation) reçoivent le même rayon : au source, aucune classe de couche n'écrit plus
    // `rounded-lg` en dur — seule la table `TILE_LAYOUT` le porte, pour le corps de 35.
    const src = lire(fichierNomme('ReplayPlayerCard.tsx'))
    expect(src).not.toMatch(/className="[^"]*rounded-lg/)
    expect(src).toMatch(/replay-zone-sensor absolute inset-0 \$\{radiusClass\}/)
    expect(src).toMatch(/replay-flash-translocation absolute inset-0 \$\{radiusClass\}/)
    expect(src).toMatch(/replay-card-fx pointer-events-none absolute inset-0 \$\{L\.layerRadius\}/)
    expect(src.match(/layerRadius: 'rounded-lg'/g)).toHaveLength(1)
    expect(src.match(/layerRadius: 'rounded-md'/g)).toHaveLength(1)
    expect(lire(fichierNomme('ReplayObjectiveMark.tsx'))).not.toMatch(/className="[^"]*rounded-lg/)
  })
})

/** La rangée d'armes d'une tuile : celle qui porte l'âge de la lecture des armes. */
function rangeeArmes(t: HTMLElement): HTMLElement {
  return within(t).getByTitle(/Armes (lues il y a|de la première image-clé)|Weapons (read|from the first keyframe)/)
}

describe('ReplayPlayerCard — la tuile compacte : gate 3 (ce qui quitte la tuile est dans un `title`, avec sa valeur)', () => {
  it('une seule cellule d’arme, à 48 px ; l’arme rangée est NOMMÉE dans le `title` (weaponStowedFmt) ; loadout non lu → cellule vide + `title`', () => {
    const { vue } = renderBTB(12)
    const alpha = tuile(vue, 'Alpha')
    const rangee = rangeeArmes(alpha)
    expect(rangee.title).toMatch(/^Arme rangée : Pistolet · /)
    expect(rangee.title).toContain('Armes lues il y a')
    // Une cellule, et une seule : la main. Le Pistolet n'a plus de vignette, il est nommé.
    expect(rangee.children).toHaveLength(1)
    const cellule = rangee.children[0].firstElementChild as HTMLElement
    expect(cellule.style.width).toBe('48px')
    expect(cellule.hasAttribute('title')).toBe(false)
    expect(within(alpha).getByRole('img', { name: 'Fusil' })).toBeTruthy()
    expect(within(alpha).queryByRole('img', { name: 'Pistolet' })).toBeNull()
    // La cellule vide du loadout non lu : une seule, à 48 px, sous « armes non lues ».
    const bravo = tuile(vue, 'Bravo')
    const nonLu = within(bravo).getByTitle('armes non lues sur cette vie')
    expect(nonLu.querySelectorAll('span[aria-hidden]')).toHaveLength(1)
    expect((nonLu.querySelector('span[aria-hidden]') as HTMLElement).style.width).toBe('48px')
    // Ligne 3 : arme 48 · grenade 14 · capacité 16, et rien d'autre à largeur fixe dans le
    // CORPS (l'incrustation d'effets, `aria-hidden`, a ses propres largeurs d'éclairs).
    const largeurs = [...rangees(bravo)[1].querySelectorAll('[style*="width"]')]
      .map((e) => (e as HTMLElement).style.width)
      .filter((w) => w !== '' && w !== '100%')
    expect(largeurs).toEqual(['48px', '14px', '16px'])
  })

  it('munitions de la main dans le `title` de l’arme : compte, charge restante, pleines — et RIEN en D=2', () => {
    const { vue } = renderBTB(12)
    expect(rangeeArmes(tuile(vue, 'Alpha')).title).toContain('Munitions : 10 / 20')
    expect(rangeeArmes(tuile(vue, 'Juliett')).title).toContain('Charge restante : 75 %')
    expect(rangeeArmes(tuile(vue, 'Hotel')).title).toContain('Munitions pleines')
    // Armes rangées (D=2) : aucune arme en main, aucune munition à décrire — ni lacune.
    const india = rangeeArmes(tuile(vue, 'India')).title
    expect(india).not.toContain('Munitions')
    expect(india).not.toContain('Charge')
    expect(india).not.toContain('dégainée')
    // Et aucune munition en TEXTE sur aucune tuile.
    expect(vue.queryByText('10')).toBeNull()
    expect(vue.queryByText('75%')).toBeNull()
    expect(vue.queryByLabelText('Munitions pleines')).toBeNull()
  })

  it('stock des grenades dans le `title` de la cellule de grenade (grenadeBoxHint), sélection LUE / DÉDUITE / « sél. ? » ; cellule de 14 px', () => {
    const { vue } = renderBTB(12)
    const alpha = tuile(vue, 'Alpha')
    const cellule = within(alpha).getByTitle(/Fragmentation ×1 · Plasma ×2 · Grenades lues il y a/)
    expect(cellule.title).toContain('Fragmentation — Type équipé, LU dans le film')
    expect(cellule.style.width).toBe('14px')
    expect(within(cellule).getByRole('img', { name: 'Fragmentation' })).toBeTruthy()
    expect(within(alpha).queryByText(/×/)).toBeNull()
    // Un seul type porté, sans sélecteur : DÉDUITE, et dite déduite.
    const golf = within(tuile(vue, 'Golf')).getByTitle(/Grenades lues il y a/)
    expect(golf.title).toContain('Type équipé : le seul porté')
    // Deux types sans sélecteur : la cellule reste VIDE, « sél. ? » dans son infobulle.
    const charlie = within(tuile(vue, 'Charlie')).getByTitle(/Grenades lues il y a/)
    expect(charlie.title).toContain('sél. ?')
    expect(charlie.textContent).toBe('')
    expect(charlie.querySelector('[role="img"]')).toBeNull()
    expect(vue.queryByText('sél. ?')).toBeNull()
  })

  it('score personnel dans le `title` du triplet (playerScoreLiveFmt), aucune cellule de score', () => {
    const { vue } = renderBTB(12)
    const alpha = within(tuile(vue, 'Alpha')).getByTitle(/FDA 2,33/)
    expect(alpha.title).toBe("Frags / morts / assistances à l'instant lu — FDA 2,33 · Score personnel à l'instant lu : 350")
    expect(vue.queryByText('350')).toBeNull()
    expect(vue.queryByTitle("Score personnel à l'instant lu")).toBeNull()
    expect([...vue.container.querySelectorAll('[style*="min-width: 30px"]')]).toHaveLength(0)
    // Non publié : le triplet des totaux, sans score — la base ne le porte pas à cet endroit.
    expect(within(tuile(vue, 'Hotel')).getByTitle('Frags / morts / assistances du match').title).not.toContain('Score')
  })

  it('marques de D5 dans les `title`, avec leurs âges : « Mort » (deux âges), « dégainée ? », charges « ×N » et « plein » — jamais en texte', () => {
    const { vue } = renderBTB(12)
    // Lecture vide corroborée : le libellé, l'âge de la lecture VIDE, celui de l'équipement.
    const oscar = rangeeArmes(tuile(vue, 'Oscar')).title
    expect(oscar).toMatch(
      /Mort — Lecture vide, et le fil des éliminations donne le joueur pour mort — lue il y a 0\.8 s · l.équipement affiché est la dernière lecture pleine, lue il y a 1\.7 s/,
    )
    expect(vue.queryByText('Mort')).toBeNull()
    // Sélecteur non lu sur une lecture pleine : la lacune est dite.
    expect(rangeeArmes(tuile(vue, 'Papa')).title).toContain('dégainée ?')
    expect(vue.queryByText('dégainée ?')).toBeNull()
    // Charges lues (Alpha) et « plein » (Delta) : dans l'infobulle de la capacité.
    const grappinAlpha = within(tuile(vue, 'Alpha')).getByRole('img', { name: 'Grappin' }).parentElement as HTMLElement
    expect(grappinAlpha.title).toContain("Capacité d'armure équipée — Grappin")
    expect(grappinAlpha.title).toContain('Charges restantes : 2 · Charges lues il y a 1.2 s')
    expect(within(tuile(vue, 'Alpha')).queryByText('×2')).toBeNull()
    const grappinDelta = within(tuile(vue, 'Delta')).getByRole('img', { name: 'Grappin' }).parentElement as HTMLElement
    expect(grappinDelta.title).toContain('Charges pleines')
    expect(within(tuile(vue, 'Delta')).queryByText('plein')).toBeNull()
  })

  it('`fx.title` reste atteignable sur la ligne du nom, même quand chaque cellule porte sa propre infobulle', () => {
    const { vue } = renderBTB(12)
    const charlie = tuile(vue, 'Charlie')
    expect(charlie.title).toContain('écran occultant')
    // Les cellules masquent la tuile là où elles sont…
    expect(rangeeArmes(charlie).closest('[title]')).toBe(rangeeArmes(charlie))
    expect(within(charlie).getByTitle(/Grenades lues/).closest('[title]')).not.toBe(charlie)
    // … et la ligne du nom, elle, remonte à la tuile.
    expect((vue.getByText('Charlie').parentElement as HTMLElement).closest('[title]')).toBe(charlie)
  })

  it('estompage par CELLULE (opacité sur la cellule, jamais sur la rangée) ; âge négatif dit « dans X s »', () => {
    const { vue } = renderBTB(12)
    const quebec = tuile(vue, 'Quebec')
    // Première lecture à l'image 150, lue à l'image 100 : 50 images à venir.
    expect(rangeeArmes(quebec).title).toContain('Armes de la première image-clé de cette vie, lue dans 0.8 s')
    const grenade = within(quebec).getByTitle(/Grenades lues dans 0\.8 s/)
    expect(Number(grenade.style.opacity)).toBeLessThan(1)
    const rangeeInventaire = grenade.parentElement as HTMLElement
    expect(rangeeInventaire.style.opacity).toBe('')
    expect(rangeeInventaire.title).toContain('Inventaire de la première image-clé de cette vie, lue dans 0.8 s')
    // La rangée d'armes s'estompe sur SA lecture ; la cellule de capacité sur la sienne.
    expect(Number(rangeeArmes(quebec).style.opacity)).toBeLessThan(1)
    const grappin = within(tuile(vue, 'Alpha')).getByRole('img', { name: 'Grappin' }).parentElement as HTMLElement
    expect(grappin.style.opacity).not.toBe('')
  })

  it('EN : les mêmes reports portent les libellés anglais', () => {
    const { vue } = renderBTB(12, 'en')
    const alpha = tuile(vue, 'Alpha')
    expect(rangeeArmes(alpha).title).toContain('Stowed weapon: Pistol')
    expect(rangeeArmes(alpha).title).toContain('Ammo: 10 / 20')
    expect(within(alpha).getByTitle(/KDA 2\.33/).title).toContain('Personal score at the moment being played: 350')
    expect(rangeeArmes(tuile(vue, 'Hotel')).title).toContain('Ammo full')
    expect(rangeeArmes(tuile(vue, 'Juliett')).title).toContain('Charge left: 75 %')
  })
})
