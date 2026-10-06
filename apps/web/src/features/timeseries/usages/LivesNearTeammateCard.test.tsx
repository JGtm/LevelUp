/**
 * LivesNearTeammateCard.test.tsx — « Vies à portée d'un coéquipier, vies isolées » : barre épaisse des vies
 * (près / seul, comptes et parts dans les segments), barre fine des frags, ligne « frags … par vie »,
 * vies écartées dites dans l'aide ⓘ (chiffres de la maquette : 1 558 / 301 vies, 1 250 / 262 frags).
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type * as SegmentLabelFit from '@/components/charts/segmentLabelFit'

import { lives } from './usages.fixtures'
import { buildLivesModel } from './usages.logic'
import { LivesNearTeammateCard } from './LivesNearTeammateCard'
import { USAGES_TEXT } from './usagesText'

/**
 * La mesure au pixel (jsdom : aucune largeur, tout part au repli) est remplaçable par test : `fit.hidden`
 * fixe les segments dont l'étiquette ne tient pas.
 */
const fit = vi.hoisted(() => ({ hidden: null as ReadonlySet<string> | null }))
vi.mock('@/components/charts/segmentLabelFit', async (importOriginal) => {
  const mod = await importOriginal<typeof SegmentLabelFit>()
  return {
    ...mod,
    useSegmentLabelFit: (...args: Parameters<typeof mod.useSegmentLabelFit>) => {
      const measured = mod.useSegmentLabelFit(...args)
      return fit.hidden ?? measured
    },
  }
})
afterEach(() => {
  fit.hidden = null
})

const U = USAGES_TEXT.fr.cards
const n = U.intFmt

function renderCard(b = lives()) {
  render(<LivesNearTeammateCard model={buildLivesModel(b)!} player="JGtm" ut={U} />)
}

const width = (id: string) => parseFloat((screen.getByTestId(id) as HTMLElement).style.width)

describe('LivesNearTeammateCard', () => {
  it('titre, ligne au gamertag du joueur et nombre de vies', () => {
    renderCard()
    expect(screen.getByText('Isolement')).toBeTruthy()
    expect(screen.getByText('JGtm')).toBeTruthy()
    expect(screen.getByTestId('usages-lives-sub').textContent).toBe(`${n(1859)} vies terminées par une mort`)
  })

  it('barre épaisse : vies à portée / isolées, compte et part dans chaque segment', () => {
    renderCard()
    expect(width('usages-lives-near')).toBeCloseTo((1558 / 1859) * 100)
    expect(screen.getByTestId('usages-lives-near').textContent).toBe(`${n(1558)} · 83,8 %`)
    expect(screen.getByTestId('usages-lives-alone').textContent).toBe(`16,2 % · ${n(301)}`)
  })

  it('repli : seule la valeur qui ne tient pas dans son segment monte au-dessus (jamais affichée deux fois)', () => {
    fit.hidden = new Set(['alone'])
    renderCard()
    const repli = screen.getByTestId('usages-lives-repli')
    expect(repli.textContent).toBe(`16,2 % · ${n(301)}`)
    expect(screen.getByTestId('usages-lives-near').querySelector('[data-fit-label]')!.getAttribute('style')).toContain('visible')
  })

  it('repli : rien au-dessus quand les deux valeurs tiennent', () => {
    fit.hidden = new Set()
    renderCard()
    expect(screen.queryByTestId('usages-lives-repli')).toBeNull()
  })

  it('barre fine : frags du joueur pendant ces vies ; ligne des frags par vie', () => {
    renderCard()
    expect(width('usages-lives-kills-near')).toBeCloseTo((1250 / 1512) * 100)
    expect(screen.getByTestId('usages-lives-kills-line').textContent).toBe(`frags : ${n(1250)} · 82,7 % · 0,8 par vie0,9 par vie · ${n(262)}`)
  })

  it('l’aide ⓘ compte les vies écartées', () => {
    renderCard()
    fireEvent.mouseEnter(screen.getByRole('button', { name: /info/i }))
    expect(screen.getByRole('tooltip').textContent).toContain('Écartées : vies sans coéquipier situé (59).')
    expect(screen.getByRole('tooltip').textContent).not.toContain('journal des morts')
  })

  it('l’aide ⓘ compte aussi les vies d’un match au journal des morts non publiable', () => {
    renderCard({ ...lives(), excluded_unpublishable: 12 })
    fireEvent.mouseEnter(screen.getByRole('button', { name: /info/i }))
    const tip = screen.getByRole('tooltip').textContent
    expect(tip).toContain('Écartées : vies sans coéquipier situé (59)')
    expect(tip).toContain(', vies d’un match au journal des morts non publiable (12).')
  })

  it('aucun frag : la barre fine et sa ligne se retirent', () => {
    renderCard({ ...lives(), near: { lives: 10, kills: 0 }, alone: { lives: 5, kills: 0 } })
    expect(screen.queryByTestId('usages-lives-kills-near')).toBeNull()
    expect(screen.queryByTestId('usages-lives-kills-line')).toBeNull()
  })

  it('légende', () => {
    renderCard()
    for (const l of ['À portée d’un coéquipier', 'Isolée', 'Barre fine : frags du joueur pendant ces vies']) expect(screen.getByText(l)).toBeTruthy()
  })
})
