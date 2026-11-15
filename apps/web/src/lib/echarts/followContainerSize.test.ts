import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { beginLayoutTransition, endLayoutTransition } from '@/lib/layout/layoutSettling'
import { followContainerSize, type ResizableChart } from './followContainerSize'

let observed: (() => void) | null = null
const disconnect = vi.fn()

class FakeResizeObserver {
  constructor(cb: () => void) {
    observed = cb
  }
  observe() {}
  disconnect() {
    disconnect()
  }
}

function fakeChart(domWidth: number, chartWidth: number) {
  const dom = { clientWidth: domWidth, clientHeight: 260 } as HTMLElement
  const resize = vi.fn()
  const chart: ResizableChart = {
    getDom: () => dom,
    getWidth: () => chartWidth,
    getHeight: () => 260,
    isDisposed: () => false,
    resize,
  }
  return { chart, resize }
}

describe('followContainerSize', () => {
  beforeEach(() => {
    vi.stubGlobal('ResizeObserver', FakeResizeObserver)
    vi.stubGlobal('requestAnimationFrame', (cb: () => void) => {
      cb()
      return 1
    })
    vi.stubGlobal('cancelAnimationFrame', () => {})
  })
  afterEach(() => {
    endLayoutTransition()
    vi.unstubAllGlobals()
    observed = null
    disconnect.mockReset()
  })

  it("redimensionne un graphe resté à sa largeur de montage quand le conteneur a rétréci", () => {
    const { chart, resize } = fakeChart(610, 1246)
    followContainerSize(chart)
    observed?.()
    expect(resize).toHaveBeenCalledWith({ width: 'auto', height: 'auto' })
  })

  it("ne touche pas un graphe déjà à la taille du conteneur (animation d'entrée préservée)", () => {
    const { chart, resize } = fakeChart(610, 610)
    followContainerSize(chart)
    observed?.()
    expect(resize).not.toHaveBeenCalled()
  })

  it('débranche l observateur', () => {
    const { chart } = fakeChart(610, 610)
    followContainerSize(chart)()
    expect(disconnect).toHaveBeenCalledOnce()
  })

  it("ignore les mesures pendant une transition de mise en page, puis se synchronise une fois", () => {
    const { chart, resize } = fakeChart(610, 1246)
    followContainerSize(chart)
    beginLayoutTransition()
    observed?.()
    observed?.()
    observed?.()
    expect(resize).not.toHaveBeenCalled()
    endLayoutTransition()
    expect(resize).toHaveBeenCalledTimes(1)
    expect(resize).toHaveBeenCalledWith({ width: 'auto', height: 'auto' })
  })

  it("une transition qui se termine sur la taille du graphe ne le redimensionne pas", () => {
    const { chart, resize } = fakeChart(610, 610)
    followContainerSize(chart)
    beginLayoutTransition()
    observed?.()
    endLayoutTransition()
    expect(resize).not.toHaveBeenCalled()
  })

  it("débranché, il ne se synchronise plus à la fin d une transition", () => {
    const { chart, resize } = fakeChart(610, 1246)
    followContainerSize(chart)()
    beginLayoutTransition()
    endLayoutTransition()
    expect(resize).not.toHaveBeenCalled()
  })
})
