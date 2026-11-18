/**
 * Tests — les cotes de la ligne du fil : le gabarit de colonnes en découle, et les classes de rangée
 * (`feedRow.ts`, écrites en valeurs arbitraires) disent la même hauteur et le même écart.
 */
import { describe, expect, it } from 'vitest'

import { COTES_LIGNE_FIL, feedLineColumns } from './cotesFeedLine'
import { FEED_GRID_BUTTON, FEED_GRID_ROW, FEED_ROW } from './feedRow'

describe('feedLineColumns', () => {
  it('avec lieu : instant, lieu, tueur, arme, victime, puis le reste pour les médailles', () => {
    expect(feedLineColumns(true)).toBe('30px 70px 70px 18px 70px minmax(0, 1fr)')
  })

  it('sans lieu : la colonne du lieu disparaît, les autres gardent leurs largeurs', () => {
    expect(feedLineColumns(false)).toBe('30px 70px 18px 70px minmax(0, 1fr)')
  })
})

describe('les classes de rangée disent les cotes', () => {
  const { hauteurPx, ecartPx } = COTES_LIGNE_FIL

  it.each([
    ['FEED_ROW', FEED_ROW, `gap-[${ecartPx}px]`],
    ['FEED_GRID_ROW', FEED_GRID_ROW, `gap-x-[${ecartPx}px]`],
    ['FEED_GRID_BUTTON', FEED_GRID_BUTTON, `gap-x-[${ecartPx}px]`],
  ])('%s : hauteur et écart', (_nom, classes, ecart) => {
    expect(classes.split(' ')).toContain(`h-[${hauteurPx}px]`)
    expect(classes.split(' ')).toContain(ecart)
  })

  it('aucune rangée ne porte de bordure ni de fond au repos', () => {
    for (const classes of [FEED_ROW, FEED_GRID_ROW]) {
      expect(classes).not.toMatch(/\bborder|\bbg-|replay-feed-row/)
    }
  })
})
