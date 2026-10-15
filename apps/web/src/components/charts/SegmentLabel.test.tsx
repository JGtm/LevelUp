/**
 * SegmentLabel.test — le CENTRAGE VERTICAL d'une étiquette sur sa barre directe (15/10 : texte
 * décalé de 2 à 3 px, étiquettes de repli posées au-dessus de la barre). jsdom ne mesure aucune
 * géométrie : le test fige la STRUCTURE qui rend le centrage vrai (mesuré au navigateur le
 * 2026-10-15 : écart 0 px sur sept pistes), et la mesure se rejoue avec le harnais du lot.
 *
 *   - la racine est `absolute inset-0 flex` : remplit son segment (la barre directe) et fait de
 *     l'ancre `inline-flex` du Tooltip un enfant de flex — dans un bloc, elle se calait sur la
 *     ligne de base et descendait sous le centre ;
 *   - la rangée du texte est `items-center` sur toute la hauteur, le texte à `leading-none` ;
 *   - une étiquette masquée reste dans le DOM (remesure), jamais déplacée ailleurs.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { SegmentLabel } from './SegmentLabel'

function mount(props: Partial<Parameters<typeof SegmentLabel>[0]> = {}) {
  render(
    <div className="relative h-[22px]" data-testid="segment" data-fit-key="k">
      <SegmentLabel tip="infobulle" {...props}>
        12
      </SegmentLabel>
    </div>,
  )
  const label = screen.getByText('12')
  return { segment: screen.getByTestId('segment'), label, row: label.parentElement!, root: label.parentElement!.parentElement!.parentElement! }
}

describe('SegmentLabel', () => {
  it('remplit son segment en flex : racine absolute inset-0 flex, enfant direct du segment', () => {
    const { segment, root } = mount()
    expect(root.parentElement).toBe(segment)
    expect(root.className).toContain('absolute')
    expect(root.className).toContain('inset-0')
    expect(root.className).toContain('flex')
  })

  it('texte centré sur la hauteur : rangée items-center pleine hauteur, texte leading-none', () => {
    const { row, label } = mount()
    expect(row.className).toContain('items-center')
    expect(row.className).toContain('h-full')
    expect(label.className).toContain('leading-none')
    // Encre blanche à contour sombre : une seule classe, aucune couleur posée par le composant.
    expect(label.className).toContain('segment-ink')
    expect(label.style.color).toBe('')
    expect(label.hasAttribute('data-fit-label')).toBe(true)
  })

  it('aucun autre élément que l’étiquette dans le segment : rien n’est posé au-dessus de la barre', () => {
    const { segment } = mount()
    expect(segment.querySelectorAll('[data-fit-label]')).toHaveLength(1)
    expect(segment.textContent).toBe('12')
  })

  it('alignement : début collé à gauche, fin collé à droite, centre sans marge', () => {
    expect(mount({ align: 'start' }).row.className).toContain('justify-start')
  })

  it('masquée : invisible mais dans le DOM, dans son segment', () => {
    const { label, segment } = mount({ hidden: true })
    expect(label.style.visibility).toBe('hidden')
    expect(segment.contains(label)).toBe(true)
  })
})
