import { describe, expect, it, vi, afterEach } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import { mockTrackMetrics } from '@/test/trackMetrics'
import { Carousel, CarouselItem } from './carousel'

let unmock: (() => void) | null = null
afterEach(() => {
  unmock?.()
  unmock = null
})

const items = (n: number) =>
  Array.from({ length: n }, (_, i) => (
    <CarouselItem key={i}>
      <span>tuile-{i}</span>
    </CarouselItem>
  ))

describe('Carousel — chargement de la suite à 5 tuiles de la fin', () => {
  it('ne monte que les enfants reçus', () => {
    render(<Carousel>{items(30)}</Carousel>)
    expect(screen.getAllByText(/^tuile-/)).toHaveLength(30)
  })

  it('appelle onNearEnd quand la fin est à 5 tuiles ou moins', () => {
    unmock = mockTrackMetrics({ scrollWidth: 9000, clientWidth: 1000, scrollLeft: 6600 })
    const onNearEnd = vi.fn()
    render(<Carousel onNearEnd={onNearEnd}>{items(30)}</Carousel>)
    expect(onNearEnd).toHaveBeenCalled()
  })

  it('n\'appelle pas onNearEnd tant que la fin est loin', () => {
    unmock = mockTrackMetrics({ scrollWidth: 9000, clientWidth: 1000, scrollLeft: 0 })
    const onNearEnd = vi.fn()
    render(<Carousel onNearEnd={onNearEnd}>{items(30)}</Carousel>)
    expect(onNearEnd).not.toHaveBeenCalled()
  })

  it('appelle onNearEnd au défilement vers la fin', () => {
    const state = { scrollWidth: 9000, clientWidth: 1000, scrollLeft: 0 }
    unmock = mockTrackMetrics(state)
    const onNearEnd = vi.fn()
    const { container } = render(<Carousel onNearEnd={onNearEnd}>{items(30)}</Carousel>)
    expect(onNearEnd).not.toHaveBeenCalled()
    unmock()
    unmock = mockTrackMetrics({ ...state, scrollLeft: 7000 })
    act(() => {
      fireEvent.scroll(container.querySelector('.overflow-x-auto') as HTMLElement)
    })
    expect(onNearEnd).toHaveBeenCalled()
  })

  it('pendant le chargement : indicateur affiché et onNearEnd silencieux', () => {
    unmock = mockTrackMetrics({ scrollWidth: 9000, clientWidth: 1000, scrollLeft: 8000 })
    const onNearEnd = vi.fn()
    render(
      <Carousel onNearEnd={onNearEnd} loadingMore>
        {items(30)}
      </Carousel>,
    )
    expect(screen.getByRole('status')).toBeInTheDocument()
    expect(onNearEnd).not.toHaveBeenCalled()
  })

  it('sans onNearEnd (plus de suite) : aucun appel, aucun indicateur', () => {
    unmock = mockTrackMetrics({ scrollWidth: 9000, clientWidth: 1000, scrollLeft: 8000 })
    render(<Carousel>{items(30)}</Carousel>)
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })
})
