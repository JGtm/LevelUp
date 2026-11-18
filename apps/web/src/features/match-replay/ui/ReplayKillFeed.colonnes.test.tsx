/**
 * Tests — la LIGNE D'ÉLIMINATION du fil, en grille de 28 px à colonnes fixes.
 *
 * Ce qu'ils protègent :
 *  1. L'ORDRE DES COLONNES : instant, lieu de la victime, tueur, arme, victime, médailles. Le gabarit
 *     (`feedLineColumns`) vient des cotes ; carte sans zone nommée, la colonne du lieu disparaît et les
 *     autres gardent leur alignement ; un lieu inconnu laisse la colonne, vide.
 *  2. LES NOMS ET LE LIEU SONT TRONQUÉS, jamais renvoyés à la ligne.
 *  3. LES MÉDAILLES : deux pictogrammes au plus, puis « +N » ; les noms des médailles en trop sont en
 *     infobulle ; une médaille sans visuel garde son texte.
 *  4. L'ASSISTANCE ET LE PORTEUR DU DRAPEAU n'ont pas de colonne : ils sont dans l'infobulle de la ligne.
 *  5. LES AUTRES FORMES (mort neutre, médaille seule, entrée ou sortie, objectif) tiennent sur une ligne
 *     de la même hauteur, ouverte par la même gouttière d'instant.
 *  6. LA PLUS RÉCENTE EST EN TÊTE.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { COTES_LIGNE_FIL } from '../model/cotesFeedLine'
import type { MedalEvent, ReplayFeedEntry, ReplayKill } from '../model/killFeedLogic'
import { testReplayDoc } from '../test/testDoc'
import { ReplayKillFeed } from './ReplayKillFeed'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const DOC = testReplayDoc({
  frameIntervalMs: 100,
  roster: [
    { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
    { xuid: 'foe', filmIndex: 1, name: 'Cobra01', team: 1 },
  ],
  objectives: [{ stat: 'flag_captures', t: 60, xuid: 'me', timeMs: 6_000 }] as never,
})
const ALLEGIANCE = filmAllegianceOf(DOC, [], 'me')

const META = new Map([
  ['me', { gamertag: 'JGtm' }],
  ['foe', { gamertag: 'Cobra01' }],
  ['long', { gamertag: 'UnNomVraimentTresLongQuiDebordeLaColonne' }],
])

function medal(i: number, over: Partial<MedalEvent> = {}): MedalEvent {
  return {
    tMs: 1_000, xuid: 'me', gamertag: 'JGtm', name: `Medal${i}`, label: `Médaille ${i}`,
    description: '', imageUrl: `/m${i}.png`, ...over,
  }
}

function killOf(over: Partial<ReplayKill> = {}, medals: MedalEvent[] = []): ReplayKill {
  return {
    tMs: 1_000, xuid: 'me', teamID: 0, weaponKey: 'br75', weaponLabel: 'BR75',
    weaponImageUrl: '/br75.png', weaponTinted: true, assistState: '', assistGamertag: '',
    assistTeamID: null, killerDamagePct: null, assistDamagePct: null, victimXuid: 'foe',
    victimGamertag: 'Cobra01', replayMs: 1_000, medals, ...over,
  } as ReplayKill
}

function ligne(key: string, over: Partial<ReplayFeedEntry> = {}): ReplayFeedEntry {
  return { key, replayMs: 1_000, kill: null, medal: null, death: null, ...over }
}

function afficher(entries: ReplayFeedEntry[], zoneOf?: (key: string) => string, doc = false) {
  return render(
    <ReplayKillFeed
      entries={entries}
      nowMs={600_000}
      playWindow={null}
      scoreboard={[]}
      xuidMeta={META}
      allegiance={ALLEGIANCE}
      locale="fr"
      zoneOf={zoneOf}
      doc={doc ? DOC : undefined}
    />,
  )
}

/** Les colonnes d'une ligne, dans l'ordre : le nom de la colonne, ou « nom » pour un nom de joueur. */
const colonnes = (li: Element) =>
  Array.from(li.children).map((c) => c.getAttribute('data-feed-col') ?? 'nom')

describe('ligne d’élimination — l’ordre des colonnes', () => {
  it('instant, lieu, tueur, arme, victime, médailles, dans un gabarit tiré des cotes', () => {
    afficher([ligne('k', { kill: killOf() })], () => 'Fer à cheval')
    const li = screen.getByRole('listitem')
    expect(colonnes(li)).toEqual(['instant', 'lieu', 'nom', 'arme', 'nom', 'medailles'])
    const cellules = Array.from(li.children)
    expect(cellules[0].textContent).toBe('0:01')
    expect(cellules[1].textContent).toBe('Fer à cheval')
    expect(cellules[2].textContent).toBe('JGtm')
    expect(cellules[3].querySelector('[role="img"]')?.getAttribute('aria-label')).toBe('BR75')
    expect(cellules[4].textContent).toBe('Cobra01')
    const c = COTES_LIGNE_FIL.colonnes
    expect((li as HTMLElement).style.gridTemplateColumns).toBe(
      `${c.instantPx}px ${c.lieuPx}px ${c.tueurPx}px ${c.armePx}px ${c.victimePx}px minmax(0, 1fr)`,
    )
  })

  it('carte SANS zone nommée : la colonne du lieu disparaît, les autres gardent leur alignement', () => {
    afficher([ligne('k', { kill: killOf() })])
    const li = screen.getByRole('listitem')
    expect(colonnes(li)).toEqual(['instant', 'nom', 'arme', 'nom', 'medailles'])
    const c = COTES_LIGNE_FIL.colonnes
    expect((li as HTMLElement).style.gridTemplateColumns).toBe(
      `${c.instantPx}px ${c.tueurPx}px ${c.armePx}px ${c.victimePx}px minmax(0, 1fr)`,
    )
    // Mêmes largeurs de colonnes que la grille avec lieu, moins celle du lieu : rien d'autre ne bouge.
  })

  it('lieu vide pour UNE ligne : la colonne reste, vide, et les colonnes suivantes ne bougent pas', () => {
    afficher(
      [ligne('a', { kill: killOf() }), ligne('b', { replayMs: 2_000, kill: killOf({ xuid: 'foe', victimXuid: 'me', victimGamertag: 'JGtm' }) })],
      (key) => (key === 'a' ? 'Pont Est' : ''),
    )
    const [recente, ancienne] = screen.getAllByRole('listitem')
    expect(colonnes(recente)).toEqual(colonnes(ancienne))
    expect(colonnes(recente)).toContain('lieu')
    expect(recente.children[1].textContent).toBe('')
    expect(ancienne.children[1].textContent).toBe('Pont Est')
    expect((recente as HTMLElement).style.gridTemplateColumns).toBe((ancienne as HTMLElement).style.gridTemplateColumns)
  })

  it('une élimination sans victime nommée garde ses colonnes', () => {
    afficher([ligne('k', { kill: killOf({ victimGamertag: '' }) })], () => '')
    expect(colonnes(screen.getByRole('listitem'))).toEqual(['instant', 'lieu', 'nom', 'arme', 'nom', 'medailles'])
  })

  it('le tueur et la victime gardent l’encre de leur camp', () => {
    afficher([ligne('k', { kill: killOf() })])
    expect((screen.getByText('JGtm') as HTMLElement).style.color).toBe('var(--ac-team-ally)')
    expect((screen.getByText('Cobra01') as HTMLElement).style.color).toBe('var(--ac-team-enemy)')
  })
})

describe('ligne d’élimination — la troncature', () => {
  it('un nom long et un lieu long sont tronqués, jamais renvoyés à la ligne', () => {
    afficher(
      [ligne('k', { kill: killOf({ xuid: 'long' }) })],
      () => 'Une zone au nom beaucoup trop long pour sa colonne',
    )
    const li = screen.getByRole('listitem')
    expect(screen.getByText('UnNomVraimentTresLongQuiDebordeLaColonne').className).toContain('truncate')
    expect(screen.getByText('Une zone au nom beaucoup trop long pour sa colonne').className).toContain('truncate')
    expect(li.className).toContain('whitespace-nowrap')
    expect(li.className).toContain('overflow-hidden')
    // La colonne du nom est à largeur fixe : un nom plus long ne l'élargit pas.
    expect((li as HTMLElement).style.gridTemplateColumns).toContain(`${COTES_LIGNE_FIL.colonnes.tueurPx}px`)
  })
})

describe('ligne d’élimination — les médailles', () => {
  const medalsCell = (li: Element) => li.querySelector('[data-feed-col="medailles"]') as HTMLElement

  it('aucune médaille : la colonne est vide', () => {
    afficher([ligne('k', { kill: killOf() })])
    const cell = medalsCell(screen.getByRole('listitem'))
    expect(cell.textContent).toBe('')
    expect(cell.querySelectorAll('img')).toHaveLength(0)
  })

  it('une médaille : un pictogramme, son nom en infobulle, pas de « +N »', () => {
    afficher([ligne('k', { kill: killOf({}, [medal(1, { description: 'Une description.' })]) })])
    const cell = medalsCell(screen.getByRole('listitem'))
    const imgs = cell.querySelectorAll('img')
    expect(imgs).toHaveLength(1)
    expect(imgs[0].getAttribute('alt')).toBe('Médaille 1')
    expect(imgs[0].getAttribute('title')).toBe('Médaille 1 — Une description.')
    expect(imgs[0].getAttribute('width')).toBe(String(COTES_LIGNE_FIL.medailles.pictoPx))
    expect(cell.textContent).toBe('')
  })

  it('deux médailles : deux pictogrammes, toujours pas de « +N »', () => {
    afficher([ligne('k', { kill: killOf({}, [medal(1), medal(2)]) })])
    const cell = medalsCell(screen.getByRole('listitem'))
    expect(cell.querySelectorAll('img')).toHaveLength(2)
    expect(cell.textContent).toBe('')
  })

  it('quatre médailles : deux pictogrammes puis « +2 », dont l’infobulle nomme les deux autres', () => {
    afficher([ligne('k', { kill: killOf({}, [medal(1), medal(2), medal(3), medal(4)]) })])
    const cell = medalsCell(screen.getByRole('listitem'))
    expect(Array.from(cell.querySelectorAll('img')).map((i) => i.getAttribute('alt'))).toEqual(['Médaille 1', 'Médaille 2'])
    const reste = screen.getByText('+2')
    expect(reste.getAttribute('title')).toBe('Autres médailles : Médaille 3, Médaille 4')
  })

  it('une médaille sans visuel garde son TEXTE, jamais le pictogramme d’une autre', () => {
    afficher([ligne('k', { kill: killOf({}, [medal(1, { imageUrl: '', label: '' })]) })])
    const cell = medalsCell(screen.getByRole('listitem'))
    expect(cell.querySelectorAll('img')).toHaveLength(0)
    expect(screen.getByText('Medal1')).toBeTruthy()
  })
})

describe('ligne d’élimination — l’infobulle porte l’assistance et le porteur', () => {
  it('assistance nommée : le nom et sa part sont dans l’infobulle, pas dans la rangée', () => {
    const k = killOf({ assistState: 'named', assistGamertag: 'Aidant77', assistDamagePct: 37 })
    const { container } = afficher([ligne('k', { kill: k })])
    const li = container.querySelector('li') as HTMLElement
    expect(li.getAttribute('title')).toContain('Assistance : Aidant77 (37 %)')
    expect(li.textContent).not.toContain('Aidant77')
    expect(li.textContent).not.toContain('37')
  })

  it('victime porteuse du drapeau : l’infobulle le dit, aucun badge dans la rangée', () => {
    const { container } = afficher([ligne('k', { kill: killOf(), flagCarrierKill: true })])
    const li = container.querySelector('li') as HTMLElement
    expect(li.getAttribute('title')).toBe('La victime portait le drapeau')
    expect(li.querySelectorAll('svg')).toHaveLength(0)
    expect(li.textContent).toBe('0:01JGtmCobra01')
  })

  it('assistée ET porteuse : les deux mentions, séparées', () => {
    const k = killOf({ assistState: 'named', assistGamertag: 'Aidant77' })
    const { container } = afficher([ligne('k', { kill: k, flagCarrierKill: true })])
    const titre = container.querySelector('li')?.getAttribute('title') ?? ''
    expect(titre).toContain('Assistance : Aidant77')
    expect(titre).toContain('La victime portait le drapeau')
  })
})

describe('les autres formes tiennent sur une ligne de la même hauteur, alignées sur l’instant', () => {
  it('mort neutre, médaille seule, entrée de partie et objectif : 28 px, une rangée, instant en tête', () => {
    afficher(
      [
        ligne('d', { replayMs: 1_000, death: { replayMs: 1_000, xuid: 'foe', kind: '', img: '', tinted: false, proven: false } }),
        ligne('m', { replayMs: 2_000, medal: medal(1) }),
        ligne('p', { replayMs: 3_000, presence: { kind: 'joined', xuid: 'foe', name: 'Cobra01', bot: false, source: 'api' } }),
      ],
      undefined,
      true,
    )
    const lignes = screen.getAllByRole('listitem')
    // Objectif (document) + les trois lignes du fil.
    expect(lignes).toHaveLength(4)
    for (const li of lignes) {
      expect(li.className).toContain('h-[28px]')
      expect(li.className).toContain('overflow-hidden')
      expect(li.className).toMatch(/flex-nowrap|whitespace-nowrap/)
      const instant = li.firstElementChild as HTMLElement
      expect(instant.getAttribute('data-feed-col')).toBe('instant')
      expect(instant.style.width).toBe(`${COTES_LIGNE_FIL.colonnes.instantPx}px`)
      expect(li.querySelector('div')).toBeNull()
    }
  })
})

describe('la plus récente est en tête', () => {
  it('les lignes se lisent de la plus récente à la plus ancienne', () => {
    afficher([
      ligne('a', { replayMs: 1_000, kill: killOf({ replayMs: 1_000 }) }),
      ligne('b', { replayMs: 5_000, kill: killOf({ replayMs: 5_000 }) }),
      ligne('c', { replayMs: 9_000, kill: killOf({ replayMs: 9_000 }) }),
    ])
    const instants = screen.getAllByRole('listitem').map((li) => li.firstElementChild?.textContent)
    expect(instants).toEqual(['0:09', '0:05', '0:01'])
  })
})
