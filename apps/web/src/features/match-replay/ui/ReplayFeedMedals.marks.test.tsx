/**
 * ReplayFeedMedals.marks.test.tsx — LES PICTOGRAMMES DE TIR À LA TÊTE ET DE FRAG PARFAIT dans la dernière colonne d'une
 * ligne d'élimination : une icône par marque vraie, nommée dans les deux langues, en tête avant les médailles ; aucune
 * icône sans marque ; l'icône du frag parfait s'efface devant la médaille « Parfait ».
 */
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { MedalEvent } from '../model/killFeedLogic'
import { FeedMedals } from './ReplayFeedMedals'
import { HEADSHOT_ICON_STEM, PERFECT_ICON_STEM } from './ReplayFragMarks'

const medal = (name: string): MedalEvent => ({ tMs: 0, xuid: 'a', gamertag: 'Alpha', name, label: name, description: '', imageUrl: '' })

const marksOf = (container: HTMLElement) => [...container.querySelectorAll('[data-frag-mark]')].map((e) => e.getAttribute('data-frag-mark'))

describe('FeedMedals — les marques du tableau des scores', () => {
  it('une icône par marque vraie, le tir à la tête puis le frag parfait, avant les médailles', () => {
    const { container } = render(
      <FeedMedals medals={[medal('Double Kill')]} marks={{ headshot: true, perfect: true }} locale="fr" />,
    )
    expect(marksOf(container)).toEqual([HEADSHOT_ICON_STEM, PERFECT_ICON_STEM])
    const colonne = container.querySelector('[data-feed-col="medailles"]') as HTMLElement
    expect((colonne.firstElementChild as HTMLElement).getAttribute('data-frag-mark')).toBe(HEADSHOT_ICON_STEM)
  })

  it('les infobulles sont dites dans la langue de la page', () => {
    const fr = render(<FeedMedals medals={[]} marks={{ headshot: true, perfect: true }} locale="fr" />)
    expect([...fr.container.querySelectorAll('[data-frag-mark]')].map((e) => e.getAttribute('title'))).toEqual([
      'Tir à la tête',
      'Frag parfait',
    ])
    fr.unmount()
    const en = render(<FeedMedals medals={[]} marks={{ headshot: true, perfect: true }} locale="en" />)
    expect([...en.container.querySelectorAll('[data-frag-mark]')].map((e) => e.getAttribute('title'))).toEqual([
      'Headshot',
      'Perfect kill',
    ])
  })

  it('sans marque vraie, aucune icône : ni zéro ni texte de remplacement', () => {
    const { container } = render(<FeedMedals medals={[]} marks={undefined} locale="fr" />)
    expect(marksOf(container)).toEqual([])
    expect(container.textContent).toBe('')
    const vide = render(<FeedMedals medals={[]} marks={{}} locale="fr" />)
    expect(marksOf(vide.container)).toEqual([])
  })

  it('une seule marque : une seule icône', () => {
    const { container } = render(<FeedMedals medals={[]} marks={{ headshot: true }} locale="fr" />)
    expect(marksOf(container)).toEqual([HEADSHOT_ICON_STEM])
  })

  it('la médaille « Parfait » de la ligne dit déjà le frag parfait : pas de seconde icône', () => {
    const { container } = render(
      <FeedMedals medals={[medal('Perfect')]} marks={{ headshot: true, perfect: true }} locale="fr" />,
    )
    expect(marksOf(container)).toEqual([HEADSHOT_ICON_STEM])
  })
})
