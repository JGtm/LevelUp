/**
 * Tests — LE FIL DES ÉLIMINATIONS POSÉ SUR LE PLAN, quand la rangée des équipes est posée (Cinéma, Plein cadre), et
 * l'option « Fil posé sur le plan » du panneau Vue. Suite de `ReplayPage.row.test.tsx` (même kit : `test/replayRowKit`).
 */
import { fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { REPLAY_TEXT } from './i18n/i18n'
import { MARGE_BASCULE_PX } from './model/rowLayout'
import {
  colonneDesEquipes,
  elimination,
  installerLesDoublesDeLaRangee,
  L4,
  match,
  mount,
  rangee,
  redimensionner,
} from './test/replayRowKit'
import { goToFrame, unstubReplayPage } from './test/replayPageHarness'

vi.mock('@tanstack/react-router', async (importOriginal) =>
  (await import('./test/replayPageRouter')).routerMock(await importOriginal<typeof import('@tanstack/react-router')>()),
)

const T = REPLAY_TEXT.fr

beforeEach(installerLesDoublesDeLaRangee)

afterEach(() => {
  unstubReplayPage()
})

const filDuPlan = (c: HTMLElement) => c.querySelector<HTMLElement>('[data-replay-plan-feed]')
const colonne = (c: HTMLElement) => c.querySelector<HTMLElement>('[data-replay-column]')
/** Trois éliminations au coup d'envoi : le rejeu s'ouvre en pause à l'image 0, elles sont déjà passées. */
const TROIS_ELIMINATIONS = [elimination(0), elimination(0), elimination(0)]

describe('ReplayPage : le fil des éliminations posé sur le plan', () => {
  it.each(['cinema', 'plein-cadre'] as const)('%s, rangée posée : les éliminations passées sont SUR LE PLAN, dans le bloc de la toile', async (frame) => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goToFrame(frame)
    const fil = filDuPlan(container)
    expect(fil).not.toBeNull()
    expect(fil?.querySelectorAll('li')).toHaveLength(3)
    expect(container.querySelector('[data-replay-grid] > section')?.contains(fil)).toBe(true)
    expect(fil?.contains(container.querySelector('canvas'))).toBe(false)
  })

  it('rangée posée : la colonne est cachée (le plan prend sa largeur) mais le fil complet y reste monté, même nœud', async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    const fil = screen.getByTestId('replay-feed-slot')
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
    goToFrame('cinema')
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
    expect(colonne(container)?.classList.contains('hidden')).toBe(true)
    expect(colonne(container)?.contains(screen.getByTestId('replay-feed-slot'))).toBe(true)
    expect(screen.getByTestId('replay-feed-slot')).toBe(fil)
    expect(container.querySelector('[data-replay-grid]')?.className).toContain('xl:grid-cols-[minmax(0,1fr)]')
    expect(container.querySelector('[data-replay-grid]')?.className).not.toContain('30rem')
  })

  it('rangée posée : la colonne reste cachée fiche ouverte (la fiche est une fenêtre), le plan garde sa largeur', async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goToFrame('cinema')
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
    fireEvent.click(within(rangee(container) as HTMLElement).getByRole('button', { name: 'Ouvrir la fiche de Joueur5' }))
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
    expect(container.querySelector('[data-replay-grid]')?.className).not.toContain('30rem')
    expect(screen.getByRole('region', { name: T.playerPanel.labelFmt('Joueur5') })).toBeInTheDocument()
    expect(colonne(container)?.querySelector('[data-replay-fiche-popover]')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: T.playerPanel.close }))
    expect(screen.queryByRole('region', { name: T.playerPanel.labelFmt('Joueur5') })).toBeNull()
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
  })

  it('repli (équipes en colonne) : pas de fil sur le plan, la colonne garde les équipes et le fil complet', async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4 - 1)
    goToFrame('cinema')
    expect(filDuPlan(container)).toBeNull()
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
    expect(colonne(container)?.contains(colonneDesEquipes(container))).toBe(true)
    expect(within(screen.getByTestId('replay-feed-slot')).getAllByRole('listitem').length).toBeGreaterThan(0)
  })

  it('largeur non mesurée : le repli aussi', async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), 0)
    goToFrame('plein-cadre')
    expect(filDuPlan(container)).toBeNull()
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
  })

  it('le Standard : jamais de fil sur le plan, sa colonne porte le fil et reste affichée', async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), 4000)
    expect(filDuPlan(container)).toBeNull()
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
    expect(within(screen.getByTestId('replay-feed-slot')).getAllByRole('listitem').length).toBeGreaterThan(0)
  })

  it("la fenêtre rétrécit puis s'élargit : le fil du plan apparaît et disparaît avec la rangée, sans remonter la toile ni le fil complet", async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goToFrame('cinema')
    const toile = container.querySelector('canvas')
    const fil = screen.getByTestId('replay-feed-slot')
    expect(filDuPlan(container)).not.toBeNull()
    redimensionner(L4 - MARGE_BASCULE_PX - 1)
    expect(filDuPlan(container)).toBeNull()
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
    redimensionner(L4)
    expect(filDuPlan(container)).not.toBeNull()
    expect(container.querySelector('canvas')).toBe(toile)
    expect(screen.getByTestId('replay-feed-slot')).toBe(fil)
  })

  it("rien n'est remonté pendant la lecture : le fil du plan et la cellule du fil complet gardent leur nœud", async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goToFrame('cinema')
    const fil = filDuPlan(container)
    const cellule = screen.getByTestId('replay-feed-slot')
    fireEvent.click(screen.getByRole('button', { name: 'Lecture' }))
    expect(filDuPlan(container)).toBe(fil)
    expect(screen.getByTestId('replay-feed-slot')).toBe(cellule)
  })
})

describe("ReplayPage : l'option « Fil posé sur le plan » du panneau Vue", () => {
  const option = () => screen.queryByRole('switch', { name: T.frames.planFeed })
  const ouvrirLaVue = () => {
    fireEvent.click(screen.getByRole('button', { name: /^Vue / }))
    fireEvent.click(screen.getByRole('tab', { name: /^Fil/ }))
  }

  it("Cinéma, rangée posée : l'option est proposée, allumée ; l'éteindre retire le fil du plan, le rallumer le rend", async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goToFrame('cinema')
    expect(filDuPlan(container)).not.toBeNull()
    ouvrirLaVue()
    expect(option()).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(option() as HTMLElement)
    expect(filDuPlan(container)).toBeNull()
    fireEvent.click(option() as HTMLElement)
    expect(filDuPlan(container)).not.toBeNull()
  })

  it('éteinte, elle le reste au remontage de la page', async () => {
    const premier = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goToFrame('plein-cadre')
    ouvrirLaVue()
    fireEvent.click(option() as HTMLElement)
    expect(filDuPlan(premier.container)).toBeNull()
    premier.unmount()
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goToFrame('plein-cadre')
    expect(filDuPlan(container)).toBeNull()
    ouvrirLaVue()
    expect(option()).toHaveAttribute('aria-checked', 'false')
  })

  it("Standard : l'option n'est pas montrée (aucun effet à l'écran)", async () => {
    await mount(match(4, 2, TROIS_ELIMINATIONS), 4000)
    ouvrirLaVue()
    expect(screen.getByRole('dialog', { name: T.viewButton })).toBeInTheDocument()
    expect(option()).toBeNull()
  })

  it("repli (équipes en colonne) au Cinéma : l'option n'est pas montrée", async () => {
    await mount(match(4, 2, TROIS_ELIMINATIONS), L4 - 1)
    goToFrame('cinema')
    ouvrirLaVue()
    expect(screen.getByRole('dialog', { name: T.viewButton })).toBeInTheDocument()
    expect(option()).toBeNull()
  })

  it("option éteinte au Cinéma, rangée posée : la colonne reste cachée (elle ne sert qu'à une fiche ouverte)", async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goToFrame('cinema')
    ouvrirLaVue()
    fireEvent.click(option() as HTMLElement)
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
  })
})
