/**
 * ReplayTeams.density.test.tsx — LA COLONNE sous ses deux gabarits : les cas limites de
 * l'étape 4 du plan fiches compactes (2026-09-06), un `it` par item.
 *
 * CE QUE CE FICHIER PROTÈGE, et que le test de la tuile (`ReplayPlayerCard.compact`) ne dit
 * pas : la densité est celle du MATCH (décision D1), donc UN SEUL gabarit sur toute la colonne
 * quels que soient les effectifs (12v9), les relais (le compte de sièges ne bouge pas, la fiche
 * suit l'occupant) et le contenu du film (un titre sans décodage rend des cellules vides, jamais
 * des zéros) ; et, en miroir, un roster de 24 SANS catégorie `BTB` rend la colonne d'aujourd'hui
 * — aucun repli sur le nombre de sièges. La chaîne de la classe de grille est assertée à
 * l'octet, à la manière de `rosterHeight.guard.test.ts` : une valeur arbitraire Tailwind qui
 * contient un espace ou un `calc(` ne produit aucune règle, en silence.
 *
 * LE DOCUMENT EST NU À DESSEIN (une vie par siège, avec ou sans vitalité) : ce qui est éprouvé
 * ici est la colonne, pas la fiche.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { ReplayTeams } from './ReplayTeams'
import { fichierNomme, lire } from '../test/featureFiles'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

/** Les deux chaînes de `ReplayTeams.tsx`, recopiées À L'OCTET : le test les compare au source ET au DOM. */
const SEATS_GRID_CLASS =
  'grid min-h-0 flex-1 auto-rows-max grid-cols-[repeat(auto-fill,minmax(115px,1fr))] gap-1 overflow-y-auto'
const SEATS_COLUMN_CLASS = 'flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto'

const START_TIME = '2026-07-24T20:00:00Z'
const EN_TETE_BTB = { start_time: START_TIME, mode_category: 'BTB' }
const FRAME = 50

/** Douze noms par camp, sans collision avec les bandeaux « Équipe Eagle » / « Équipe Cobra ». */
function noms(prefixe: string, n: number): string[] {
  return Array.from({ length: n }, (_, i) => `${prefixe}${String(i + 1).padStart(2, '0')}`)
}
const NORD = noms('Nord', 12)
const SUD = noms('Sud', 12)

type Track = NonNullable<ReplayDocument['tracks']>[number]

/** Une vie du siège `i` : avec ou sans mesure de vitalité, sur [start, 300]. */
function vie(nom: string, i: number, vitalite: boolean, start = 0): Track {
  const point = vitalite ? { t: start, x: 1, y: 1, sh: 0.7, hp: 1 } : { t: start, x: 1, y: 1 }
  return { slot: 512 + i, team: -1, xuid: nom, startFrame: start, endFrame: 300, points: [point] }
}

/**
 * Le document : un siège par nom, l'horloge ÉTABLIE (`originMs` publié, 100 ms l'image —
 * sans elle, aucun relais de siège ne se date, cf. `seatLogic`). Aucun label d'arme, aucun
 * loadout, aucun inventaire, aucune capacité : un titre sans décodage film quand `vitalite`
 * est faux.
 */
function documentDe(liste: readonly string[], vitalite = true, extra: Track[] = []) {
  return testReplayDoc({
    frameCount: 300,
    frameIntervalMs: 100,
    originMs: 0,
    roster: [...liste, ...extra.map((t) => t.xuid ?? '')].map((nom, i) => ({ xuid: nom, filmIndex: i, name: nom })),
    tracks: [...liste.map((nom, i) => vie(nom, i, vitalite)), ...extra],
  })
}

/** Le tableau : compteurs NON NULS partout, pour qu'un « 0 » dans une tuile soit un zéro inventé. */
function tableau(nord: readonly string[], sud: readonly string[], over: Record<string, Partial<MatchScoreboardRow>> = {}) {
  const ligne = (nom: string, camp: 't0' | 't1') =>
    scoreboardRow(nom, nom, camp, { kills: 2, deaths: 1, assists: 3, ...(over[nom] ?? {}) })
  return [...nord.map((n) => ligne(n, 't0')), ...sud.map((n) => ligne(n, 't1'))]
}

/** Le conteneur des sièges d'un camp : nom → ligne du nom → tuile → conteneur. */
function conteneur(vue: ReturnType<typeof render>, nom: string): HTMLElement {
  return vue.getByText(nom).parentElement?.parentElement?.parentElement as HTMLElement
}

function corps(vue: ReturnType<typeof render>, px: 35 | 31): number {
  return vue.container.querySelectorAll(`.h-\\[${px}px\\]`).length
}

describe('ReplayTeams — la densité de la colonne : cas limites (étape 4)', () => {
  it('4.1 BTB à effectifs inégaux (12v9) : un seul gabarit sur toute la colonne, les deux camps en grille compacte', () => {
    const sud = SUD.slice(0, 9)
    const vue = render(
      <ReplayTeams doc={documentDe([...NORD, ...sud])} scoreboard={tableau(NORD, sud)} frame={FRAME} locale="fr" header={EN_TETE_BTB} />,
    )
    expect(corps(vue, 31)).toBe(21)
    expect(corps(vue, 35)).toBe(0)
    const nord = conteneur(vue, 'Nord01')
    const sudC = conteneur(vue, 'Sud01')
    expect(nord).not.toBe(sudC)
    expect(nord.children).toHaveLength(12)
    expect(sudC.children).toHaveLength(9)
    // Le camp de neuf n'est pas moins compact que celui de douze : la densité est celle du match.
    expect(nord.className).toBe(SEATS_GRID_CLASS)
    expect(sudC.className).toBe(SEATS_GRID_CLASS)
    expect(vue.container.querySelectorAll('.rounded-md.border')).toHaveLength(21)
  })

  it('4.2 sièges relayés en BTB : le compte de sièges ne bouge pas (24), la fiche suit l’occupant', () => {
    // Nord12 quitte à +10 s (image 100) ; Zulu le remplace à la même seconde, même camp t0.
    // Sa vie reste ouverte jusqu'à 300 : sans appariement, la colonne compterait 25 sièges.
    const doc = documentDe([...NORD, ...SUD], true, [vie('Zulu', 24, true, 100)])
    const board = [
      ...tableau(NORD, SUD, {
        Nord12: { left_in_progress: true, last_leave_time: '2026-07-24T20:00:10Z' },
      }),
      scoreboardRow('Zulu', 'Zulu', 't0', { joined_in_progress: true, first_joined_time: '2026-07-24T20:00:10Z' }),
    ]
    const arbre = (frame: number) => (
      <ReplayTeams doc={doc} scoreboard={board} frame={frame} locale="fr" header={EN_TETE_BTB} />
    )
    const vue = render(arbre(FRAME))
    // Avant le relais : le partant tient la fiche, le remplaçant n'est nulle part.
    expect(corps(vue, 31)).toBe(24)
    expect(vue.getByText('Nord12')).toBeTruthy()
    expect(vue.queryByText('Zulu')).toBeNull()
    expect(conteneur(vue, 'Nord12').children).toHaveLength(12)
    expect(conteneur(vue, 'Nord12').className).toBe(SEATS_GRID_CLASS)
    // Après : le même siège, le même compte, l'autre occupant — toujours en compacte.
    vue.rerender(arbre(150))
    expect(corps(vue, 31)).toBe(24)
    expect(corps(vue, 35)).toBe(0)
    expect(vue.getByText('Zulu')).toBeTruthy()
    expect(vue.queryByText('Nord12')).toBeNull()
    expect(conteneur(vue, 'Zulu').children).toHaveLength(12)
    expect(conteneur(vue, 'Zulu').className).toBe(SEATS_GRID_CLASS)
  })

  it('4.3 la chaîne EXACTE de la classe de grille (source et DOM), et la classe de colonne normale inchangée', () => {
    // Au source, à l'octet — la technique de rosterHeight.guard : une classe Tailwind avec un
    // espace ou un calc() dans sa valeur arbitraire passerait un test de rendu sans produire de CSS.
    // Les commentaires sont ôtés : l'en-tête de la constante cite lui-même le piège `calc(`.
    const src = lire(fichierNomme('ReplayTeams.tsx'))
      .replace(/\/\*[\s\S]*?\*\//g, '')
      .replace(/^\s*\/\/.*$/gm, '')
    expect(src).toContain(`'${SEATS_GRID_CLASS}'`)
    expect(src).toContain(`'${SEATS_COLUMN_CLASS}'`)
    expect(src).not.toContain('calc(')
    expect(src).not.toMatch(/grid-cols-\[[^\]]*\s[^\]]*\]/)
    expect(src.match(/grid-cols-\[/g)).toHaveLength(1)
    // Au DOM : la grille en BTB, la colonne simple sinon — et rien entre les deux.
    const btb = render(
      <ReplayTeams doc={documentDe([...NORD, ...SUD])} scoreboard={tableau(NORD, SUD)} frame={FRAME} locale="fr" header={EN_TETE_BTB} />,
    )
    expect(conteneur(btb, 'Nord01').className).toBe(SEATS_GRID_CLASS)
    expect(conteneur(btb, 'Sud01').className).toBe(SEATS_GRID_CLASS)
    btb.unmount()
    const quatre = NORD.slice(0, 4)
    const arena = render(
      <ReplayTeams doc={documentDe([...quatre, ...SUD.slice(0, 4)])} scoreboard={tableau(quatre, SUD.slice(0, 4))} frame={FRAME} locale="fr" header={{ start_time: START_TIME }} />,
    )
    expect(conteneur(arena, 'Nord01').className).toBe(SEATS_COLUMN_CLASS)
    expect(conteneur(arena, 'Sud01').className).toBe(SEATS_COLUMN_CLASS)
  })

  it('4.4 titre sans décodage film en compacte : cellules vides, aucune barre, aucun zéro, aucun `role=img`', () => {
    const vue = render(
      <ReplayTeams doc={documentDe([...NORD, ...SUD], false)} scoreboard={tableau(NORD, SUD)} frame={FRAME} locale="fr" header={EN_TETE_BTB} />,
    )
    expect(corps(vue, 31)).toBe(24)
    // Aucune barre : le document ne porte ni `sh` ni `hp`.
    expect(vue.queryByLabelText('Bouclier')).toBeNull()
    expect(vue.queryByLabelText('Santé')).toBeNull()
    // Aucune vignette, aucun glyphe, aucun filigrane.
    expect(vue.container.querySelectorAll('[role="img"]')).toHaveLength(0)
    expect(vue.container.querySelectorAll('svg')).toHaveLength(0)
    // Aucun zéro inventé : le tableau n'en porte aucun, une tuile n'en fabrique pas.
    expect(vue.queryByText('0')).toBeNull()
    expect(vue.queryByText(/×|%|\d+ \/ \d+/)).toBeNull()
    expect(vue.queryByTitle(/Grenades|Inventaire|Munitions|Charge/)).toBeNull()
    // Les TROIS cellules fixes de la ligne 3 (arme 48, grenade 14, capacité 16) sont rendues
    // VIDES sur chacun des 24 sièges — donnée absente = cellule vide, jamais un décalage — et
    // aucune autre largeur fixe n'apparaît.
    expect(vue.getAllByTitle('armes non lues sur cette vie')).toHaveLength(24)
    const cellules = [...vue.container.querySelectorAll('[style*="width"]')]
      .map((e) => e as HTMLElement)
      .filter((e) => e.style.width !== '' && e.style.width !== '100%')
    expect(cellules.map((e) => e.style.width)).toEqual(
      Array.from({ length: 24 }, () => ['48px', '14px', '16px']).flat(),
    )
    for (const c of cellules) expect(c.textContent).toBe('')
    // Et le triplet, lui, dit le tableau : 2/1/3 sur chaque tuile.
    const triplets = vue.getAllByTitle(/^Frags \/ morts \/ assistances du match/)
    expect(triplets).toHaveLength(24)
    for (const t of triplets) expect(t.textContent).toBe('2/1/3')
    expect(vue.container.textContent).not.toContain('?')
  })

  it('4.5 en-tête sans catégorie (ou `Arena`) sur un roster de 24 : densité normale, le DOM de la colonne d’aujourd’hui (D1, aucun repli sur les effectifs)', () => {
    const doc = documentDe([...NORD, ...SUD])
    const board = tableau(NORD, SUD)
    const rendu = (header: { start_time?: string; mode_category?: string } | null | undefined) => {
      const vue = render(<ReplayTeams doc={doc} scoreboard={board} frame={FRAME} locale="fr" header={header} />)
      const html = vue.container.innerHTML
      vue.unmount()
      return html
    }
    const sansCategorie = rendu({ start_time: START_TIME })
    // Cinq en-têtes qui ne disent pas `BTB` : cinq fois le même HTML, à l'octet.
    for (const header of [undefined, null, { start_time: START_TIME, mode_category: 'Arena' }, { start_time: START_TIME, mode_category: '' }]) {
      expect(rendu(header)).toBe(sansCategorie)
    }
    // Et c'est bien la colonne d'aujourd'hui : 24 corps de 35, colonne simple, tuile `rounded-lg`.
    const vue = render(<ReplayTeams doc={doc} scoreboard={board} frame={FRAME} locale="fr" header={{ start_time: START_TIME }} />)
    expect(corps(vue, 35)).toBe(24)
    expect(corps(vue, 31)).toBe(0)
    expect(conteneur(vue, 'Nord01').className).toBe(SEATS_COLUMN_CLASS)
    expect(conteneur(vue, 'Sud01').className).toBe(SEATS_COLUMN_CLASS)
    expect(vue.container.querySelectorAll('.rounded-lg.border.px-2\\.5.py-2')).toHaveLength(24)
    expect(vue.container.innerHTML).not.toContain('auto-fill')
    expect(vue.container.innerHTML).not.toContain('rounded-md')
    vue.unmount()
    // Contre-épreuve : le MÊME document et le MÊME tableau, sous `BTB`, sont compacts — seule
    // la catégorie décide, jamais l'effectif.
    expect(rendu(EN_TETE_BTB)).not.toBe(sansCategorie)
    expect(rendu(EN_TETE_BTB)).toContain('auto-fill')
  })
})
