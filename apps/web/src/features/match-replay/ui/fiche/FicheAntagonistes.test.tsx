/**
 * Tests — FicheAntagonistes : la section « Antagonistes » de la fiche d'un joueur.
 *
 * Elle écrit, À L'INSTANT LU et POUR LE JOUEUR DE LA FICHE (quel qu'il soit, pas le point de vue de la page) :
 *  - « Némésis » : l'adversaire qui l'a le plus éliminé, « X, k éliminations » (« 1 élimination ») ;
 *  - « Souffre-douleur » : l'adversaire qu'il a le plus éliminé, « X, éliminé k fois » ;
 *  - le suffixe « , à égalité avec Y » (« Y et Z », « Y, Z et W ») ;
 *  - l'infobulle de l'icône, qui définit le titre ;
 *  - un cadre pointillé « Aucun antagoniste à cet instant. » quand le joueur n'a aucun titre.
 * Les noms portent l'encre du camp (jetons). Français et anglais.
 */
import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'

import type { Antagonists, AntagonistsAt } from '@/lib/antagonists/antagonists'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayAntagonists } from '../../model/replayAntagonists'
import { FicheAntagonistes } from './FicheAntagonistes'

const ME: ReplayPlayer = { xuid: 'me-1', filmName: 'JGtm', lives: [] }
const PLAYERS: ReplayPlayer[] = [
  ME,
  { xuid: 'foe-1', filmName: 'Rival', lives: [] },
  { xuid: 'foe-2', filmName: 'Fantome', lives: [] },
  { xuid: 'foe-3', filmName: 'Brume', lives: [] },
  { xuid: 'ally-1', filmName: 'Ami', lives: [] },
]

const NONE: AntagonistsAt = { nemesis: null, victim: null }

/** Une fabrique dont l'index du sujet répond `at(t)` ; l'espion note le sujet demandé. */
function fabrique(at: (t: number) => AntagonistsAt) {
  const forSubject = vi.fn<(key: string | null) => Antagonists>(() => ({ changes: [], at }))
  const antagonists: ReplayAntagonists = { forSubject, forViewpoint: () => ({ changes: [], at }) }
  return { antagonists, forSubject }
}

function monter(at: (t: number) => AntagonistsAt, over: Partial<Parameters<typeof FicheAntagonistes>[0]> = {}) {
  const { antagonists, forSubject } = fabrique(at)
  const props = {
    player: ME,
    antagonists,
    players: PLAYERS,
    allegiance: { ofXuid: (key: string | null | undefined) => key === 'ally-1' || key === 'me-1' },
    nowMs: 5_000,
    locale: 'fr' as const,
    ...over,
  }
  const utils = render(<FicheAntagonistes {...props} />)
  return { ...utils, forSubject, props }
}

const text = (c: HTMLElement, role: 'nemesis' | 'victim') =>
  c.querySelector(`[data-fiche-antagoniste="${role}"]`)?.textContent

describe('FicheAntagonistes — les titres, en français', () => {
  const DEUX: AntagonistsAt = {
    nemesis: { holder: 'foe-1', count: 3, tiedWith: [] },
    victim: { holder: 'foe-2', count: 2, tiedWith: [] },
  }

  it('titre de section, libellés et valeurs', () => {
    const { container, getByText } = monter(() => DEUX)
    expect(container.querySelector('section')?.getAttribute('aria-label')).toBe('Antagonistes')
    expect(getByText('Némésis')).toBeTruthy()
    expect(getByText('Souffre-douleur')).toBeTruthy()
    expect(text(container, 'nemesis')).toBe('Rival, 3 éliminations')
    expect(text(container, 'victim')).toBe('Fantome, éliminé 2 fois')
  })

  it('une élimination se lit au singulier', () => {
    const { container } = monter(() => ({ nemesis: { holder: 'foe-1', count: 1, tiedWith: [] }, victim: null }))
    expect(text(container, 'nemesis')).toBe('Rival, 1 élimination')
  })

  it('un titre sans tenant n est pas écrit, l autre l est', () => {
    const { container, queryByText } = monter(() => ({ nemesis: null, victim: DEUX.victim }))
    expect(queryByText('Némésis')).toBeNull()
    expect(text(container, 'victim')).toBe('Fantome, éliminé 2 fois')
  })

  it('l icône de chaque titre porte sa définition en infobulle (N = le joueur de la fiche)', () => {
    const { container } = monter(() => DEUX)
    const nemesis = container.querySelector('[data-antagonist-icon="nemesis"]')?.parentElement
    const victim = container.querySelector('[data-antagonist-icon="victim"]')?.parentElement
    expect(nemesis?.getAttribute('title')).toBe(
      "L'adversaire qui a éliminé JGtm le plus de fois jusqu'à cet instant. En cas d'égalité, le titre reste à celui qui l'avait.",
    )
    expect(victim?.getAttribute('title')).toBe(
      "L'adversaire que JGtm a éliminé le plus de fois jusqu'à cet instant. En cas d'égalité, le titre reste à celui qui l'avait.",
    )
  })

  it('les noms portent l encre de leur camp : adversaire à l encre adverse, ami à l encre alliée', () => {
    const { container } = monter(() => ({
      nemesis: { holder: 'foe-1', count: 2, tiedWith: [] },
      victim: { holder: 'ally-1', count: 1, tiedWith: [] },
    }))
    const nom = (role: string) => container.querySelector<HTMLElement>(`[data-fiche-antagoniste="${role}"] [data-fiche-nom]`)
    expect(nom('nemesis')?.style.color).toContain('--ac-team-enemy')
    expect(nom('victim')?.style.color).toContain('--ac-team-ally')
  })
})

describe('FicheAntagonistes — l égalité', () => {
  const avec = (tiedWith: string[]): AntagonistsAt => ({ nemesis: { holder: 'foe-1', count: 2, tiedWith }, victim: null })

  it('un seul : « à égalité avec Y »', () => {
    const { container } = monter(() => avec(['foe-2']))
    expect(text(container, 'nemesis')).toBe('Rival, 2 éliminations, à égalité avec Fantome')
  })

  it('deux : « Y et Z »', () => {
    const { container } = monter(() => avec(['foe-2', 'foe-3']))
    expect(text(container, 'nemesis')).toBe('Rival, 2 éliminations, à égalité avec Fantome et Brume')
  })

  it('trois : « Y, Z et W »', () => {
    const { container } = monter(() => avec(['foe-2', 'foe-3', 'ally-1']))
    expect(text(container, 'nemesis')).toBe('Rival, 2 éliminations, à égalité avec Fantome, Brume et Ami')
  })

  it('le souffre-douleur à égalité s écrit de même', () => {
    const { container } = monter(() => ({ nemesis: null, victim: { holder: 'foe-1', count: 2, tiedWith: ['foe-2'] } }))
    expect(text(container, 'victim')).toBe('Rival, éliminé 2 fois, à égalité avec Fantome')
  })
})

describe('FicheAntagonistes — l état vide', () => {
  it('un joueur sans adversaire : un cadre pointillé, et aucun titre', () => {
    const { container, getByText, queryByText } = monter(() => NONE)
    const vide = getByText('Aucun antagoniste à cet instant.')
    expect(vide.className).toContain('border-dashed')
    expect(queryByText('Némésis')).toBeNull()
    expect(queryByText('Souffre-douleur')).toBeNull()
    // La section existe : son titre reste, comme celui d'une fiche qui n'a rien à dire de plus.
    expect(container.querySelector('section')?.getAttribute('aria-label')).toBe('Antagonistes')
  })
})

describe('FicheAntagonistes — le joueur de la fiche, l instant lu', () => {
  it('lit l index du joueur DE LA FICHE, par sa clé du film', () => {
    const { forSubject } = monter(() => NONE)
    expect(forSubject).toHaveBeenCalledWith('me-1')
  })

  it('suit l instant : la même fiche relue à un autre instant écrit un autre titre', () => {
    const at = (t: number): AntagonistsAt =>
      t < 10_000
        ? { nemesis: { holder: 'foe-1', count: 1, tiedWith: [] }, victim: null }
        : { nemesis: { holder: 'foe-2', count: 2, tiedWith: [] }, victim: null }
    const { container, rerender, props } = monter(at, { nowMs: 5_000 })
    expect(text(container, 'nemesis')).toBe('Rival, 1 élimination')
    rerender(<FicheAntagonistes {...props} nowMs={20_000} />)
    expect(text(container, 'nemesis')).toBe('Fantome, 2 éliminations')
  })

  it('passe l instant lu à l index', () => {
    const at = vi.fn<(t: number) => AntagonistsAt>(() => NONE)
    monter(at, { nowMs: 7_300 })
    expect(at).toHaveBeenCalledWith(7_300)
  })
})

describe('FicheAntagonistes — en anglais', () => {
  const DEUX: AntagonistsAt = {
    nemesis: { holder: 'foe-1', count: 3, tiedWith: ['foe-2', 'foe-3'] },
    victim: { holder: 'foe-2', count: 1, tiedWith: [] },
  }

  it('titre de section, libellés et valeurs', () => {
    const { container, getByText } = monter(() => DEUX, { locale: 'en' })
    expect(container.querySelector('section')?.getAttribute('aria-label')).toBe('Antagonists')
    expect(getByText('Nemesis')).toBeTruthy()
    expect(getByText('Bully target')).toBeTruthy()
    expect(text(container, 'nemesis')).toBe('Rival, 3 eliminations, tied with Fantome and Brume')
    expect(text(container, 'victim')).toBe('Fantome, eliminated 1 time')
  })

  it('une élimination se lit au singulier', () => {
    const { container } = monter(() => ({ nemesis: { holder: 'foe-1', count: 1, tiedWith: [] }, victim: null }), { locale: 'en' })
    expect(text(container, 'nemesis')).toBe('Rival, 1 elimination')
  })

  it('les définitions des titres', () => {
    const { container } = monter(() => DEUX, { locale: 'en' })
    expect(container.querySelector('[data-antagonist-icon="nemesis"]')?.parentElement?.getAttribute('title')).toBe(
      'The opponent who has eliminated JGtm the most times so far. If there is a tie, the title stays with whoever held it.',
    )
    expect(container.querySelector('[data-antagonist-icon="victim"]')?.parentElement?.getAttribute('title')).toBe(
      'The opponent JGtm has eliminated the most times so far. If there is a tie, the title stays with whoever held it.',
    )
  })

  it('l état vide', () => {
    const { getByText } = monter(() => NONE, { locale: 'en' })
    expect(getByText('No antagonist at this moment.')).toBeTruthy()
  })
})
