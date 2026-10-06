/**
 * LivesNearTeammateCard.test.tsx — « Mes vies : près d'un coéquipier ou seul » : barre épaisse des vies
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
  render(<LivesNearTeammateCard model={buildLivesModel(b)!} ut={U} />)
}

const width = (id: string) => parseFloat((screen.getByTestId(id) as HTMLElement).style.width)

describe('LivesNearTeammateCard', () => {
  it('titre, libellé de ligne et nombre de vies', () => {
    renderCard()
    expect(screen.getByText('Mes vies : près d’un coéquipier ou seul')).toBeTruthy()
    expect(screen.getByTestId('usages-lives-sub').textContent).toBe(`${n(1859)} vies terminées par une mort`)
  })

  it('barre épaisse : mes vies près / seul, compte et part dans chaque segment', () => {
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

  it('barre fine : mes frags pendant ces vies ; ligne des frags par vie', () => {
    renderCard()
    expect(width('usages-lives-kills-near')).toBeCloseTo((1250 / 1512) * 100)
    expect(screen.getByTestId('usages-lives-kills-line').textContent).toBe(`frags : ${n(1250)} · 82,7 % · 0,8 par vie0,9 par vie · ${n(262)}`)
  })

  it('l’aide ⓘ compte les vies écartées', () => {
    renderCard()
    fireEvent.mouseEnter(screen.getByRole('button', { name: /info/i }))
    expect(screen.getByRole('tooltip').textContent).toContain('écartées (59 ici)')
    expect(screen.getByRole('tooltip').textContent).not.toContain('journal des morts')
  })

  it('l’aide ⓘ compte aussi les vies d’un match au journal des morts non publiable', () => {
    renderCard({ ...lives(), excluded_unpublishable: 12 })
    fireEvent.mouseEnter(screen.getByRole('button', { name: /info/i }))
    const tip = screen.getByRole('tooltip').textContent
    expect(tip).toContain('écartées (59 ici)')
    expect(tip).toContain('Les vies d’un match dont le journal des morts ne se lit pas mort par mort sont écartées aussi (12).')
  })

  it('aucun frag : la barre fine et sa ligne se retirent', () => {
    renderCard({ ...lives(), near: { lives: 10, kills: 0 }, alone: { lives: 5, kills: 0 } })
    expect(screen.queryByTestId('usages-lives-kills-near')).toBeNull()
    expect(screen.queryByTestId('usages-lives-kills-line')).toBeNull()
  })

  it('légende de la maquette', () => {
    renderCard()
    for (const l of ['Près d’un coéquipier', 'Seul', 'Barre fine : mes frags pendant ces vies']) expect(screen.getByText(l)).toBeTruthy()
  })
})
