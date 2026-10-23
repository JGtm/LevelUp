import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

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
})
