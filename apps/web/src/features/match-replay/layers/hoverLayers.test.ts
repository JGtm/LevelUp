/**
 * hoverLayers.test.ts — le survol SE TAIT pendant un export (decision D7, 2026-09-16).
 *
 * Pendant l'export la toile est dessinee au cadre du format et affichee en `contain` : un survol
 * projete dans ce cadre viserait a cote. Il efface donc ce qu'il montrait et ne distribue plus
 * rien, puis reprend quand l'export rend la main.
 */
import type { PointerEvent } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { exportFormatOf, exportLayoutFor } from '../export/exportFormats'
import { requestExportLayout } from '../export/exportLayoutStore'
import { hoverHandlers } from './hoverLayers'

afterEach(() => {
  requestExportLayout(null)
})

describe('hoverHandlers — pendant un export', () => {
  it('efface les infobulles et ne distribue ni survol ni glisser', () => {
    const layer = { onPointerMove: vi.fn(), onPointerLeave: vi.fn() }
    const pan = { onPointerDown: vi.fn(), onPointerMove: vi.fn(), onPointerUp: vi.fn() }
    const h = hoverHandlers([layer], pan)
    const e = {} as PointerEvent<HTMLCanvasElement>

    requestExportLayout(exportLayoutFor(exportFormatOf('1080p')))
    h.onPointerMove(e)
    expect(layer.onPointerMove).not.toHaveBeenCalled()
    expect(pan.onPointerMove).not.toHaveBeenCalled()
    expect(layer.onPointerLeave).toHaveBeenCalledTimes(1)

    requestExportLayout(null)
    h.onPointerMove(e)
    expect(layer.onPointerMove).toHaveBeenCalledTimes(1)
    expect(pan.onPointerMove).toHaveBeenCalledTimes(1)
  })
})
