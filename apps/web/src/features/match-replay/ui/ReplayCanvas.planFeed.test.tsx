/**
 * Tests — LE FIL DES ÉLIMINATIONS POSÉ SUR LE PLAN (`planFeed`) : les emplacements du cadre le portent quand la
 * vue de la page est donnée, la toile dit à la fonction de rendu s'il est montré, l'option « Fil des
 * éliminations » de la vue le retire et le rend sans remonter la toile, et la légende de la chaleur lui cède le
 * coin bas-gauche.
 */
import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

import { renderWithProviders } from '../../../test/render-utils'
import { REPLAY_TEXT } from '../i18n/i18n'
import { baseProps, canvasWith, installCanvasDoubles, smallDoc } from '../test/canvasTestKit'
import { canvasSlotsOf } from './frames/replayFrameSlots'
import type { ReplayFrameView } from './frames/replayFrameTypes'
import { ReplayCanvas } from './ReplayCanvas'

const T = REPLAY_TEXT.fr

beforeEach(installCanvasDoubles)

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('ReplayCanvas : le fil des éliminations du plan (`planFeed`)', () => {
  /** La partie de la vue de la page que le fil du plan lit : le fil, l'instant lu, les noms. */
  function planFeedView(doc: ReplayDocumentReady): ReplayFrameView {
    const props = baseProps(doc)
    const kill = {
      tMs: 0, replayMs: 0, xuid: 'a', victimXuid: 'b', victimGamertag: 'Victime', weaponKey: '', weaponLabel: '',
      weaponImageUrl: '', weaponTinted: false, medals: [],
    }
    return {
      model: {
        feed: [{ key: 'k0', replayMs: 0, kill, medal: null, death: null }],
        window: null, identity: new Map([['a', { gamertag: 'Tueur' }]]), allegiance: props.allegiance, marks: new Map(),
      },
      playerPanel: { camera: { player: null, onFollow: () => {} } },
      nowMs: 0, locale: 'fr', feedZoneOf: undefined,
    } as unknown as ReplayFrameView
  }

  it('les emplacements du cadre portent le fil du plan quand la vue est donnée, la barre compacte restant la même', () => {
    const doc = smallDoc()
    const sans = canvasSlotsOf('cinema')
    const avec = canvasSlotsOf('cinema', planFeedView(doc))
    expect(sans.planFeed).toBeUndefined()
    expect(sans.blockOverlays).toBeUndefined()
    expect(avec.planFeed).toBe(true)
    expect(avec.blockOverlays).toBeDefined()
    expect(avec.transport).toBe(sans.transport)
    expect(canvasSlotsOf('plein-cadre', planFeedView(doc)).planFeed).toBe(true)
  })

  it('le Standard ne pose jamais le fil du plan, même quand la vue est donnée', () => {
    const standard = canvasSlotsOf('standard', planFeedView(smallDoc()))
    expect(standard).toBe(canvasSlotsOf('standard'))
    expect(standard.planFeed).toBeUndefined()
  })

  it('la toile dit à la fonction de rendu si le fil du plan est montré, et le fil se pose dans le bloc de la toile', () => {
    const doc = smallDoc()
    const slots = canvasSlotsOf('cinema', planFeedView(doc))
    const { container } = renderWithProviders(
      <ReplayCanvas {...baseProps(doc)} transport={slots.transport} blockOverlays={slots.blockOverlays} planFeed={slots.planFeed} />,
    )
    const fil = container.querySelector('[data-replay-plan-feed]')
    expect(fil).not.toBeNull()
    expect(fil?.parentElement).toBe(container.querySelector('canvas')!.parentElement!.parentElement)
    expect(fil?.textContent).toContain('Tueur')
    expect(fil?.textContent).toContain('Victime')
  })

  it('sans `planFeed` (cadre sans fil sur le plan), la fonction de rendu reçoit « non montré » et rien ne se pose', () => {
    const doc = smallDoc()
    const slots = canvasSlotsOf('cinema', planFeedView(doc))
    const { container } = renderWithProviders(
      <ReplayCanvas {...baseProps(doc)} transport={slots.transport} blockOverlays={slots.blockOverlays} />,
    )
    expect(container.querySelector('[data-replay-plan-feed]')).toBeNull()
  })

  describe("l'option « Fil des éliminations » de la vue", () => {
    const montage = (planFeed: boolean) => {
      const doc = smallDoc()
      const slots = canvasSlotsOf('cinema', planFeedView(doc))
      return renderWithProviders(
        <ReplayCanvas {...baseProps(doc)} transport={slots.transport} blockOverlays={slots.blockOverlays} planFeed={planFeed} />,
      )
    }
    const ouvrirLaVue = () => fireEvent.click(screen.getByRole('button', { name: /^Vue / }))

    it("éteinte, le fil du plan n'est pas rendu et la légende de la chaleur retrouve le bas ; allumée, il l'est", () => {
      localStorage.setItem('replay-show-heatmap', 'true')
      localStorage.setItem('replay-show-plan-feed', 'false')
      const eteint = montage(true)
      expect(eteint.container.querySelector('[data-replay-plan-feed]')).toBeNull()
      expect(eteint.container.querySelector<HTMLElement>(`[title="${T.heatLegendHint}"]`)?.className).toContain('bottom-2')
      eteint.unmount()

      localStorage.setItem('replay-show-plan-feed', 'true')
      const allume = montage(true)
      expect(allume.container.querySelector('[data-replay-plan-feed]')).not.toBeNull()
      expect(allume.container.querySelector<HTMLElement>(`[title="${T.heatLegendHint}"]`)?.className).toContain('top-2')
    })

    it("l'option est dans le panneau quand le cadre pose le fil, et la basculer retire puis rend le fil, sans remonter la toile", () => {
      const { container } = montage(true)
      const toile = container.querySelector('canvas')
      expect(container.querySelector('[data-replay-plan-feed]')).not.toBeNull()
      ouvrirLaVue()
      const option = screen.getByRole('switch', { name: T.frames.planFeed })
      expect(option).toHaveAttribute('aria-checked', 'true')
      fireEvent.click(option)
      expect(container.querySelector('[data-replay-plan-feed]')).toBeNull()
      fireEvent.click(screen.getByRole('switch', { name: T.frames.planFeed }))
      expect(container.querySelector('[data-replay-plan-feed]')).not.toBeNull()
      expect(container.querySelector('canvas')).toBe(toile)
    })

    it("sans `planFeed` (Standard, repli), l'option n'est pas dans le panneau", () => {
      montage(false)
      ouvrirLaVue()
      expect(screen.getByRole('region', { name: T.viewButton })).toBeInTheDocument()
      expect(screen.queryByRole('switch', { name: T.frames.planFeed })).toBeNull()
    })
  })

  describe('la légende de la carte de chaleur', () => {
    const legende = (c: HTMLElement) => c.querySelector<HTMLElement>(`[title="${T.heatLegendHint}"]`)

    it('en bas à gauche du bloc par défaut, en haut à gauche quand le cadre pose le fil sur le plan', () => {
      localStorage.setItem('replay-show-heatmap', 'true')
      const doc = smallDoc()
      const sans = renderWithProviders(canvasWith(doc, canvasSlotsOf('cinema').transport))
      expect(legende(sans.container)?.className).toContain('bottom-2')
      expect(legende(sans.container)?.className).not.toContain('top-2')
      sans.unmount()

      const slots = canvasSlotsOf('cinema', planFeedView(doc))
      const avec = renderWithProviders(
        <ReplayCanvas {...baseProps(doc)} transport={slots.transport} blockOverlays={slots.blockOverlays} planFeed={slots.planFeed} />,
      )
      expect(legende(avec.container)?.className).toContain('top-2')
      expect(legende(avec.container)?.className).not.toContain('bottom-2')
    })
  })
})
