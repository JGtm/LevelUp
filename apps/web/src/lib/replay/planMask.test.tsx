import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { recordingContext } from '@/features/match-replay/test/recordingContext'

import { PlanMaskDef } from './PlanMaskDef'
import { planMaskId, planMaskUrl, rognerAuPlan } from './planMask'

describe('planMask — identifiant et référence', () => {
  it('remet l’identifiant de useId au jeu de caractères sûr, préfixé', () => {
    expect(planMaskId(':r1:')).toBe('plan-mask-_r1_')
    expect(planMaskId('tactique')).toBe('plan-mask-tactique')
  })

  it('la référence de couche est url(#id)', () => {
    expect(planMaskUrl('plan-mask-a')).toBe('url(#plan-mask-a)')
  })
})

describe('PlanMaskDef — le <mask> SVG', () => {
  it('pose une image plan au cadre donné, en masque alpha', () => {
    const { container } = render(
      <svg>
        <defs>
          <PlanMaskDef id="m" href="blob:plan" cadre={{ x: -10, y: -60, width: 60, height: 50 }} />
        </defs>
      </svg>,
    )
    const masque = container.querySelector('mask')
    expect(masque).not.toBeNull()
    expect(masque?.getAttribute('id')).toBe('m')
    expect(masque?.getAttribute('maskUnits')).toBe('userSpaceOnUse')
    expect(masque?.getAttribute('style')).toContain('mask-type: alpha')
    const image = masque?.querySelector('image')
    expect(image?.getAttribute('href')).toBe('blob:plan')
    expect(image?.getAttribute('x')).toBe('-10')
    expect(image?.getAttribute('y')).toBe('-60')
    expect(image?.getAttribute('width')).toBe('60')
    expect(image?.getAttribute('height')).toBe('50')
    expect(image?.getAttribute('preserveAspectRatio')).toBe('none')
  })
})

describe('rognerAuPlan — le canvas', () => {
  it('peint l’image plan en destination-in, puis rend le contexte à son état', () => {
    const { ops, ctx } = recordingContext()
    let pendant: string | undefined
    rognerAuPlan(ctx, () => {
      ops.push({ op: 'peindre le plan', args: [] })
      pendant = ops.filter((o) => o.op === 'set globalCompositeOperation').at(-1)?.args[0] as string
    })
    expect(pendant).toBe('destination-in')
    const noms = ops.map((o) => o.op)
    expect(noms).toEqual(['save', 'set globalCompositeOperation', 'peindre le plan', 'restore'])
  })
})
