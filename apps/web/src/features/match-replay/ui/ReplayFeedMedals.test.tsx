/**
 * ReplayFeedMedals.test.tsx — les médailles du fil depuis une planche d'images : une médaille à découpe de
 * planche (`sprite`) se dessine par `MedalIcon` dans les deux composants du fil (`FeedMedals`, la colonne de
 * la ligne ; `MedalBadges`, les badges de la frise), comme sur la fiche ; une médaille à image seule garde
 * exactement son `<img>` ; une médaille sans visuel garde son texte.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { readMatchMedals } from '../model/matchMedals'
import type { MedalEvent } from '../model/killFeedLogic'
import { testReplayDoc } from '../test/testDoc'
import { MedalBadges } from './MedalBadges'
import { FeedMedals } from './ReplayFeedMedals'

const SHEET = '/static/medals/sheet.png'

function medal(over: Partial<MedalEvent> = {}): MedalEvent {
  return { tMs: 1_000, xuid: 'A', gamertag: '', name: 'Double Kill', label: 'Double', description: 'Deux frags', imageUrl: '', ...over }
}

const SPRITE_MEDAL = medal({ sprite: { sheet: SHEET, left: 74, top: 148, width: 74, height: 74 } })
const IMAGE_MEDAL = medal({ imageUrl: '/static/medals/double.png' })
const TEXT_MEDAL = medal()

const COMPOSANTS = [
  ['FeedMedals', (m: MedalEvent) => <FeedMedals medals={[m]} locale="fr" />],
  ['MedalBadges', (m: MedalEvent) => <MedalBadges medals={[m]} />],
] as const

describe.each(COMPOSANTS)('%s', (_nom, rendre) => {
  it('une médaille à découpe de planche rend son image par MedalIcon : la feuille, la position, le nom, l’infobulle', () => {
    const { container, getByRole } = render(rendre(SPRITE_MEDAL))
    const image = getByRole('img')
    expect(image.getAttribute('aria-label')).toBe('Double')
    const cellule = image.firstElementChild as HTMLElement
    expect(cellule.style.backgroundImage).toContain(SHEET)
    expect(cellule.style.backgroundPosition).toBe('-74px -148px')
    expect(container.querySelector('[title]')?.getAttribute('title')).toBe('Double — Deux frags')
    expect(container.querySelector('img')).toBeNull()
  })

  it('une médaille à image seule garde son <img>, sans MedalIcon ni texte', () => {
    const { container } = render(rendre(IMAGE_MEDAL))
    const img = container.querySelector('img')!
    expect(img.getAttribute('src')).toBe('/static/medals/double.png')
    expect(img.getAttribute('alt')).toBe('Double')
    expect(img.getAttribute('title')).toBe('Double — Deux frags')
    expect(img.getAttribute('width')).toBe(img.getAttribute('height'))
    expect(img.getAttribute('loading')).toBe('lazy')
    expect(img.className).toContain('object-contain')
    expect(container.querySelector('[role="img"]')).toBeNull()
    expect(container.textContent).toBe('')
  })

  it('une médaille avec une image ET une découpe garde l’image : la découpe ne sert qu’à défaut', () => {
    const { container } = render(rendre({ ...SPRITE_MEDAL, imageUrl: '/static/medals/double.png' }))
    expect(container.querySelector('img')?.getAttribute('src')).toBe('/static/medals/double.png')
    expect(container.querySelector('[role="img"]')).toBeNull()
  })

  it('une médaille sans visuel garde son texte : jamais le visuel d’une autre', () => {
    const { container } = render(rendre(TEXT_MEDAL))
    expect(container.textContent).toBe('Double')
    expect(container.querySelector('img, [role="img"]')).toBeNull()
  })
})

describe('readMatchMedals — la découpe suit jusqu’au fil', () => {
  it('la table du document porte la découpe : la médaille du fil la reçoit ; sans découpe, null', () => {
    const doc = testReplayDoc({
      medals: [{ ms: 2_000, xuid: 'A', name: 'Double Kill' }, { ms: 3_000, xuid: 'A', name: 'Killing Spree' }],
      medalLabels: {
        'Double Kill': { fr: 'Double', en: 'Double', sprite: { sheet: SHEET, left: 74, top: 0, width: 74, height: 74 } },
        'Killing Spree': { fr: 'Série', en: 'Spree', img: '/static/medals/spree.png' },
      },
    } as never)
    const [double, serie] = readMatchMedals(doc, 'fr')
    expect(double.sprite).toEqual({ sheet: SHEET, left: 74, top: 0, width: 74, height: 74 })
    expect(double.imageUrl).toBe('')
    expect(serie.sprite).toBeNull()
    expect(serie.imageUrl).toBe('/static/medals/spree.png')
  })
})
