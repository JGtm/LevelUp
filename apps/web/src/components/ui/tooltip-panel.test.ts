import { describe, expect, it } from 'vitest'
import { placeTooltip } from './tooltip-panel'

const VIEWPORT = { width: 1000, height: 800 }
const TIP = { width: 100, height: 40 }

describe('placeTooltip', () => {
  it('centre l’infobulle au-dessus de l’ancre', () => {
    const p = placeTooltip({ left: 400, top: 300, width: 60, bottom: 320 }, TIP, VIEWPORT)
    expect(p).toEqual({ left: 380, top: 252 })
  })

  it('bascule dessous quand le haut de la fenêtre déborde', () => {
    const p = placeTooltip({ left: 400, top: 20, width: 60, bottom: 40 }, TIP, VIEWPORT)
    expect(p.top).toBe(48)
  })

  it('borne l’infobulle à 8 px des bords horizontaux', () => {
    expect(placeTooltip({ left: 0, top: 300, width: 10, bottom: 320 }, TIP, VIEWPORT).left).toBe(8)
    expect(placeTooltip({ left: 995, top: 300, width: 10, bottom: 320 }, TIP, VIEWPORT).left).toBe(892)
  })

  it('reste dans la fenêtre quand le dessous déborde aussi', () => {
    const p = placeTooltip({ left: 400, top: 5, width: 60, bottom: 790 }, TIP, VIEWPORT)
    expect(p.top).toBe(752)
  })
})
