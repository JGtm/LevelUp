import { afterEach, describe, expect, it } from 'vitest'

import { barLabel } from '@/components/charts/barLabel'

import { segmentInk } from './segmentInk'

afterEach(() => {
  document.documentElement.style.removeProperty('--segment-ink')
  document.documentElement.style.removeProperty('--segment-ink-outline')
})

describe('segmentInk', () => {
  it('blanc cerné d’un contour sombre par défaut, quel que soit le thème', () => {
    expect(segmentInk()).toEqual({ ink: '#ffffff', outline: '#0b1220' })
  })

  it('suit les jetons CSS quand ils sont posés', () => {
    document.documentElement.style.setProperty('--segment-ink', '#fafafa')
    document.documentElement.style.setProperty('--segment-ink-outline', '#101010')
    expect(segmentInk()).toEqual({ ink: '#fafafa', outline: '#101010' })
  })
})

describe('barLabel', () => {
  it('étiquette interne centrée, blanche à contour sombre fin ; le reste se surcharge', () => {
    const l = barLabel({ fontSize: 10 })
    expect(l).toMatchObject({
      show: true,
      position: 'inside',
      verticalAlign: 'middle',
      color: '#ffffff',
      textBorderColor: '#0b1220',
      textBorderWidth: 2,
      fontSize: 10,
    })
  })
})
