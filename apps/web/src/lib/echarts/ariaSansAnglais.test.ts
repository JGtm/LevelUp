import { describe, expect, it } from 'vitest'

import { sansTexteAriaAnglais } from '@/lib/echarts/ariaSansAnglais'

describe('sansTexteAriaAnglais', () => {
  it('coupe le libellé automatique (anglais) d un graphe dont aria est actif sans description', () => {
    const out = sansTexteAriaAnglais({ aria: { decal: { show: true } }, series: [] }) as unknown as {
      aria: { label: { enabled: boolean }; decal: { show: boolean } }
    }
    expect(out.aria.label.enabled).toBe(false)
    expect(out.aria.decal.show).toBe(true)
  })

  it('garde la description localisée fournie par le graphe', () => {
    const option = { aria: { enabled: true, label: { description: 'Résumé localisé' } } }
    expect(sansTexteAriaAnglais(option)).toBe(option)
  })

  it('ne touche pas une option sans aria', () => {
    const option = { series: [] }
    expect(sansTexteAriaAnglais(option)).toBe(option)
  })
})
