import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import { focusByKeyboard, focusByMouse } from '@/test/focus'
import { InfoTooltip } from './info-tooltip'
import { SHOW_DELAY_MS } from './titleTooltip.controller'
import { TitleTooltipLayer } from './TitleTooltipLayer'
import { Tooltip } from './tooltip'

function hover(el: Element) {
  fireEvent.pointerOver(el)
  fireEvent.pointerEnter(el)
}

function afterDelay() {
  act(() => {
    vi.advanceTimersByTime(SHOW_DELAY_MS + 1)
  })
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('TitleTooltipLayer', () => {
  it('survol : retire le title, affiche l’infobulle dans document.body après le délai, puis restitue', () => {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title="Ouvre l’aide">
          Aide
        </button>
      </>,
    )
    const bouton = screen.getByRole('button', { name: 'Aide' })
    hover(bouton)
    // Natif supprimé tout de suite ; rien d'affiché avant le délai.
    expect(bouton).not.toHaveAttribute('title')
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    afterDelay()
    const infobulle = screen.getByRole('tooltip')
    expect(infobulle).toHaveTextContent('Ouvre l’aide')
    expect(infobulle.parentElement).toBe(document.body)
    expect(bouton).toHaveAttribute('aria-describedby', infobulle.id)
    expect(bouton).toHaveAccessibleName('Aide')
    // Sortie du document (pointeur hors fenêtre) : le title revient.
    fireEvent.pointerOut(bouton, { relatedTarget: null })
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(bouton).toHaveAttribute('title', 'Ouvre l’aide')
    expect(bouton).not.toHaveAttribute('aria-describedby')
  })

  it('passer d’un élément à un autre rend le title du premier', () => {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title="Un">
          A
        </button>
        <button type="button" title="Deux">
          B
        </button>
        <span>neutre</span>
      </>,
    )
    const [a, b] = screen.getAllByRole('button')
    hover(a)
    afterDelay()
    hover(b)
    expect(a).toHaveAttribute('title', 'Un')
    expect(b).not.toHaveAttribute('title')
    afterDelay()
    expect(screen.getByRole('tooltip')).toHaveTextContent('Deux')
    hover(screen.getByText('neutre'))
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(b).toHaveAttribute('title', 'Deux')
  })

  it('le clic masque l’infobulle sans rendre le title sous un pointeur immobile ; la sortie le rend', () => {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title="Aide">
          Ok
        </button>
        <span>neutre</span>
      </>,
    )
    const bouton = screen.getByRole('button')
    hover(bouton)
    afterDelay()
    fireEvent.click(bouton)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(bouton).not.toHaveAttribute('title')
    hover(screen.getByText('neutre'))
    expect(bouton).toHaveAttribute('title', 'Aide')
  })

  it('Échap et le défilement masquent l’infobulle', () => {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title="Aide">
          Ok
        </button>
      </>,
    )
    const bouton = screen.getByRole('button')
    hover(bouton)
    afterDelay()
    fireEvent.keyDown(document.body, { key: 'Escape' })
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    fireEvent.pointerOut(bouton, { relatedTarget: null })
    hover(bouton)
    afterDelay()
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    fireEvent.scroll(window)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('un élément retiré du DOM pendant l’affichage fait disparaître l’infobulle', async () => {
    render(<TitleTooltipLayer />)
    const bouton = document.createElement('button')
    bouton.title = 'Aide'
    bouton.textContent = 'Ok'
    document.body.appendChild(bouton)
    hover(bouton)
    afterDelay()
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    await act(async () => {
      bouton.remove()
      // Les rappels d'un MutationObserver sont des micro-tâches.
      await Promise.resolve()
    })
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('focus clavier : affiche tout de suite ; focus souris : rien', () => {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title="Aide">
          Ok
        </button>
      </>,
    )
    const bouton = screen.getByRole('button')
    focusByMouse(bouton)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(bouton).toHaveAttribute('title', 'Aide')
    act(() => bouton.blur())
    focusByKeyboard(bouton)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Aide')
    expect(bouton).not.toHaveAttribute('title')
    act(() => bouton.blur())
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(bouton).toHaveAttribute('title', 'Aide')
  })

  it('un bouton icône garde son nom accessible pendant l’affichage', () => {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title="Fermer">
          <svg />
        </button>
      </>,
    )
    const bouton = screen.getByRole('button')
    focusByKeyboard(bouton)
    expect(bouton).toHaveAccessibleName('Fermer')
    act(() => bouton.blur())
    expect(bouton).not.toHaveAttribute('aria-label')
  })

  it('respecte les retours à la ligne du title', () => {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title={'Ligne 1\nLigne 2'}>
          Ok
        </button>
      </>,
    )
    focusByKeyboard(screen.getByRole('button'))
    const infobulle = screen.getByRole('tooltip')
    expect(infobulle.textContent).toBe('Ligne 1\nLigne 2')
    expect(infobulle.className).toContain('whitespace-pre-line')
  })

  it('un <title> SVG passe par la couche', () => {
    render(
      <>
        <TitleTooltipLayer />
        <svg>
          <path data-testid="arc" d="M0 0">
            <title>Arc 42</title>
          </path>
        </svg>
      </>,
    )
    const arc = screen.getByTestId('arc')
    hover(arc)
    expect(arc.querySelector('title')?.textContent).toBe('')
    afterDelay()
    expect(screen.getByRole('tooltip')).toHaveTextContent('Arc 42')
    fireEvent.pointerOut(arc, { relatedTarget: null })
    expect(arc.querySelector('title')?.textContent).toBe('Arc 42')
  })

  it('pas de double infobulle sous un <Tooltip> ou un <InfoTooltip> : le title est retiré, la couche n’affiche rien', () => {
    render(
      <>
        <TitleTooltipLayer />
        <Tooltip content="Contenu riche">
          <button type="button" title="Natif A">
            A
          </button>
        </Tooltip>
        <InfoTooltip content="Aide ⓘ" trigger={<button type="button" title="Natif B">B</button>} />
      </>,
    )
    const a = screen.getByRole('button', { name: 'A' })
    hover(a)
    fireEvent.mouseEnter(a.parentElement as HTMLElement)
    afterDelay()
    expect(a).not.toHaveAttribute('title')
    const infobulles = screen.getAllByRole('tooltip')
    expect(infobulles).toHaveLength(1)
    expect(infobulles[0]).toHaveTextContent('Contenu riche')

    fireEvent.mouseLeave(a.parentElement as HTMLElement)
    const b = screen.getByRole('button', { name: 'B' })
    hover(b)
    fireEvent.mouseEnter(b.parentElement as HTMLElement)
    afterDelay()
    expect(a).toHaveAttribute('title', 'Natif A')
    expect(b).not.toHaveAttribute('title')
    const apres = screen.getAllByRole('tooltip')
    expect(apres).toHaveLength(1)
    expect(apres[0]).toHaveTextContent('Aide ⓘ')
  })

  it('ignore les pointeurs tactiles : le title reste en place', () => {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title="Aide">
          Ok
        </button>
      </>,
    )
    const bouton = screen.getByRole('button')
    const tactile = new Event('pointerover', { bubbles: true })
    Object.defineProperty(tactile, 'pointerType', { value: 'touch' })
    fireEvent(bouton, tactile)
    afterDelay()
    expect(bouton).toHaveAttribute('title')
  })

  it('le démontage de la couche rend les titres pris en charge', () => {
    const bouton = (
      <button type="button" title="Aide">
        Ok
      </button>
    )
    const { rerender } = render(
      <>
        <TitleTooltipLayer />
        {bouton}
      </>,
    )
    const el = screen.getByRole('button')
    hover(el)
    expect(el).not.toHaveAttribute('title')
    rerender(<>{bouton}</>)
    expect(el).toHaveAttribute('title', 'Aide')
  })
})
