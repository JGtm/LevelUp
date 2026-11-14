/**
 * ReplayLivesSection.test.tsx — LE GROUPE « VIES » du panneau de la vue : un en-tête qui se replie
 * (replié par défaut, retenu ensuite), trois interrupteurs qui appellent chacun SON callback, des
 * calques sans marque qui ne sont pas proposés, et des affinages (qui, quand) qui n'existent
 * qu'avec un calque allumé — « Joueur suivi » seulement avec un joueur suivi. Les deux langues.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { makeLives, renderPanel } from './replayViewPanelFixtures'

const EN_TITRE = 'Lives'

beforeEach(() => {
  window.localStorage.clear()
})

function deplier(nom = 'Vies') {
  fireEvent.click(screen.getByRole('button', { name: new RegExp(`^${nom}`) }))
}

describe('ReplayLivesSection — l’en-tête qui se replie', () => {
  it('replié par défaut : l’en-tête dit combien de calques sont allumés, aucun interrupteur', () => {
    renderPanel({ lives: makeLives({ deaths: { available: true, show: true, onToggle: vi.fn() } }) })
    const entete = screen.getByRole('button', { name: /^Vies/ })
    expect(entete).toHaveAttribute('aria-expanded', 'false')
    expect(entete.textContent).toContain('1 sur 3')
    expect(screen.queryByRole('switch', { name: 'Apparitions' })).toBeNull()
  })

  it('se déplie et se replie ; l’état est retenu pour la prochaine ouverture', () => {
    const { unmount } = renderPanel()
    deplier()
    expect(screen.getByRole('button', { name: /^Vies/ })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('switch', { name: 'Apparitions' })).toBeTruthy()
    unmount()
    renderPanel()
    expect(screen.getByRole('button', { name: /^Vies/ })).toHaveAttribute('aria-expanded', 'true')
    deplier()
    expect(screen.queryByRole('switch', { name: 'Morts' })).toBeNull()
  })
})

describe('ReplayLivesSection — les trois calques', () => {
  it('chaque interrupteur appelle son callback, jamais celui d’un voisin', () => {
    const spawns = vi.fn()
    const deaths = vi.fn()
    const killers = vi.fn()
    renderPanel({
      lives: makeLives({
        spawns: { available: true, show: false, onToggle: spawns },
        deaths: { available: true, show: false, onToggle: deaths },
        killers: { available: true, show: false, onToggle: killers },
      }),
    })
    deplier()
    fireEvent.click(screen.getByRole('switch', { name: 'Morts' }))
    expect([spawns.mock.calls.length, deaths.mock.calls.length, killers.mock.calls.length]).toEqual([0, 1, 0])
    fireEvent.click(screen.getByRole('switch', { name: 'Tueurs' }))
    fireEvent.click(screen.getByRole('switch', { name: 'Apparitions' }))
    expect([spawns.mock.calls.length, deaths.mock.calls.length, killers.mock.calls.length]).toEqual([1, 1, 1])
  })

  it('un calque sans marque n’est pas proposé, et le groupe disparaît quand aucun ne l’est', () => {
    renderPanel({
      lives: makeLives({ killers: { available: false, show: false, onToggle: vi.fn() } }),
    })
    deplier()
    expect(screen.queryByRole('switch', { name: 'Tueurs' })).toBeNull()
    expect(screen.getByRole('switch', { name: 'Morts' })).toBeTruthy()
  })

  it('le groupe entier est absent d’un film sans aucune marque', () => {
    const off = { available: false, show: false, onToggle: vi.fn() }
    renderPanel({ lives: makeLives({ spawns: off, deaths: off, killers: off }) })
    expect(screen.queryByRole('button', { name: /^Vies/ })).toBeNull()
  })

  it('l’état de chaque interrupteur suit le réglage', () => {
    renderPanel({ lives: makeLives({ deaths: { available: true, show: true, onToggle: vi.fn() } }) })
    deplier()
    expect(screen.getByRole('switch', { name: 'Morts' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('switch', { name: 'Apparitions' })).toHaveAttribute('aria-checked', 'false')
  })
})

describe('ReplayLivesSection — les affinages communs', () => {
  it('sans calque allumé, aucun affinage : filtrer ce qu’on ne voit pas ne dit rien', () => {
    renderPanel()
    deplier()
    expect(screen.queryByRole('radiogroup', { name: 'Qui' })).toBeNull()
    expect(screen.queryByRole('radiogroup', { name: 'Quand' })).toBeNull()
  })

  it('avec un calque allumé : qui et quand, chacun un choix exclusif qui appelle son callback', () => {
    const onSelectWho = vi.fn()
    const onSelectWhen = vi.fn()
    renderPanel({
      lives: makeLives({
        spawns: { available: true, show: true, onToggle: vi.fn() },
        onSelectWho,
        onSelectWhen,
      }),
    })
    deplier()
    const qui = screen.getByRole('radiogroup', { name: 'Qui' })
    expect(qui).toBeTruthy()
    expect(screen.getByRole('radio', { name: 'Tous' })).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(screen.getByRole('radio', { name: 'Mon équipe' }))
    expect(onSelectWho).toHaveBeenCalledWith('team')
    fireEvent.click(screen.getByRole('radio', { name: 'Joueur suivi' }))
    expect(onSelectWho).toHaveBeenCalledWith('followed')
    fireEvent.click(screen.getByRole('radio', { name: 'Jusqu’ici' }))
    expect(onSelectWhen).toHaveBeenCalledWith('sofar')
    expect(screen.getByRole('radio', { name: 'Toute la partie' })).toHaveAttribute('aria-checked', 'true')
  })

  it('« Joueur suivi » n’est proposé que s’il y a un joueur suivi dans ce match', () => {
    renderPanel({
      lives: makeLives({
        spawns: { available: true, show: true, onToggle: vi.fn() },
        followedAvailable: false,
      }),
    })
    deplier()
    expect(screen.queryByRole('radio', { name: 'Joueur suivi' })).toBeNull()
    expect(screen.getByRole('radio', { name: 'Mon équipe' })).toBeTruthy()
  })

  it('le choix courant se lit sur le segmenté', () => {
    renderPanel({
      lives: makeLives({
        deaths: { available: true, show: true, onToggle: vi.fn() },
        who: 'team',
        when: 'sofar',
      }),
    })
    deplier()
    expect(screen.getByRole('radio', { name: 'Mon équipe' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('radio', { name: 'Jusqu’ici' })).toHaveAttribute('aria-checked', 'true')
  })
})

describe('ReplayLivesSection — les deux langues', () => {
  it('l’anglais dit Lives, Spawns, Deaths, Killers, Who, When', () => {
    renderPanel({
      locale: 'en',
      lives: makeLives({ spawns: { available: true, show: true, onToggle: vi.fn() } }),
    })
    fireEvent.click(screen.getByRole('button', { name: new RegExp(`^${EN_TITRE}`) }))
    for (const nom of ['Spawns', 'Deaths', 'Killers']) {
      expect(screen.getByRole('switch', { name: nom })).toBeTruthy()
    }
    expect(screen.getByRole('radiogroup', { name: 'Who' })).toBeTruthy()
    expect(screen.getByRole('radiogroup', { name: 'When' })).toBeTruthy()
    expect(screen.getByRole('radio', { name: 'Followed player' })).toBeTruthy()
    expect(screen.getByRole('radio', { name: 'So far' })).toBeTruthy()
  })
})
