/**
 * Couche des infobulles `title` : titres qui CHANGENT pendant la prise en charge (rendu React du rejeu en
 * lecture), prises en charge pointeur et focus indépendantes, défilement d'un focus clavier, et sorties de page.
 * Complète `TitleTooltipLayer.test.tsx`.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import { focusByKeyboard } from '@/test/focus'
import { SHOW_DELAY_MS } from './titleTooltip.controller'
import { TitleTooltipLayer } from './TitleTooltipLayer'

function hover(el: Element) {
  fireEvent.pointerOver(el)
}

function leaveWindow(el: Element) {
  fireEvent.pointerOut(el, { relatedTarget: null })
}

function afterDelay() {
  act(() => {
    vi.advanceTimersByTime(SHOW_DELAY_MS + 1)
  })
}

/** Laisse passer les rappels des MutationObserver (micro-tâches). */
async function flushMutations() {
  await act(async () => {
    await Promise.resolve()
  })
}

function Bouton({ titre }: { titre: string | undefined }) {
  return (
    <button type="button" title={titre}>
      Ok
    </button>
  )
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('titre réécrit ou retiré par l’application pendant la prise en charge', () => {
  it('réécriture : une seule infobulle, avec le texte neuf, et le titre neuf revient à la sortie', async () => {
    const { rerender } = render(
      <>
        <TitleTooltipLayer />
        <Bouton titre="Ancien" />
      </>,
    )
    const bouton = screen.getByRole('button')
    hover(bouton)
    afterDelay()
    expect(screen.getByRole('tooltip')).toHaveTextContent('Ancien')
    rerender(
      <>
        <TitleTooltipLayer />
        <Bouton titre="Neuf" />
      </>,
    )
    await flushMutations()
    // Pas d'infobulle native (title revidé), une seule infobulle, au texte neuf.
    expect(bouton).toHaveAttribute('title', '')
    expect(screen.getAllByRole('tooltip')).toHaveLength(1)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Neuf')
    leaveWindow(bouton)
    expect(bouton).toHaveAttribute('title', 'Neuf')
  })

  it('retrait : l’infobulle disparaît et le titre retiré par l’application ne ressuscite pas', async () => {
    const { rerender } = render(
      <>
        <TitleTooltipLayer />
        <Bouton titre="Il y a 3 s" />
      </>,
    )
    const bouton = screen.getByRole('button')
    hover(bouton)
    afterDelay()
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    rerender(
      <>
        <TitleTooltipLayer />
        <Bouton titre={undefined} />
      </>,
    )
    await flushMutations()
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    leaveWindow(bouton)
    expect(bouton).not.toHaveAttribute('title')
    expect(bouton).not.toHaveAttribute('data-lu-title')
  })

  it('retrait avant l’affichage : rien ne s’affiche après le délai', async () => {
    const { rerender } = render(
      <>
        <TitleTooltipLayer />
        <Bouton titre="Aide" />
      </>,
    )
    hover(screen.getByRole('button'))
    rerender(
      <>
        <TitleTooltipLayer />
        <Bouton titre={undefined} />
      </>,
    )
    await flushMutations()
    afterDelay()
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('le <title> SVG réécrit met l’infobulle à jour', async () => {
    function Arc({ texte }: { texte: string }) {
      return (
        <svg>
          <path data-testid="arc" d="M0 0">
            <title>{texte}</title>
          </path>
        </svg>
      )
    }
    const { rerender } = render(
      <>
        <TitleTooltipLayer />
        <Arc texte="Arc 1" />
      </>,
    )
    const arc = screen.getByTestId('arc')
    hover(arc)
    afterDelay()
    rerender(
      <>
        <TitleTooltipLayer />
        <Arc texte="Arc 2" />
      </>,
    )
    await flushMutations()
    expect(arc.querySelector('title')?.textContent).toBe('')
    expect(screen.getByRole('tooltip')).toHaveTextContent('Arc 2')
    leaveWindow(arc)
    expect(arc.querySelector('title')?.textContent).toBe('Arc 2')
  })
})

describe('pointeur et focus clavier : deux prises en charge indépendantes', () => {
  function deuxBoutons() {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title="Sous le pointeur">
          B
        </button>
        <button type="button" title="Au clavier">
          A
        </button>
      </>,
    )
    return { b: screen.getByRole('button', { name: 'B' }), a: screen.getByRole('button', { name: 'A' }) }
  }

  it('Tab vers A pendant que le pointeur reste sur B ne rend pas le titre de B', () => {
    const { a, b } = deuxBoutons()
    hover(b)
    afterDelay()
    focusByKeyboard(a)
    expect(b).toHaveAttribute('title', '')
    expect(a).toHaveAttribute('title', '')
    expect(screen.getAllByRole('tooltip')).toHaveLength(1)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Au clavier')
    // Le pointeur quitte B : B retrouve son titre, A reste tenu par le focus.
    leaveWindow(b)
    expect(b).toHaveAttribute('title', 'Sous le pointeur')
    expect(a).toHaveAttribute('title', '')
    // Le focus part : A le retrouve.
    act(() => a.blur())
    expect(a).toHaveAttribute('title', 'Au clavier')
  })

  it('le pointeur qui quitte l’élément focalisé ne lui rend pas son titre', () => {
    const { a } = deuxBoutons()
    focusByKeyboard(a)
    hover(a)
    leaveWindow(a)
    expect(a).toHaveAttribute('title', '')
    act(() => a.blur())
    expect(a).toHaveAttribute('title', 'Au clavier')
  })
})

describe('défilement', () => {
  function bouton() {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title="Aide">
          Ok
        </button>
      </>,
    )
    return screen.getByRole('button')
  }

  it('un défilement causé par le focus clavier ne masque pas l’infobulle', () => {
    const b = bouton()
    focusByKeyboard(b)
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    fireEvent.scroll(window)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Aide')
  })

  it('une infobulle de focus se masque quand l’élément sort de la fenêtre', () => {
    const b = bouton()
    focusByKeyboard(b)
    vi.spyOn(b, 'getBoundingClientRect').mockReturnValue({
      top: -80,
      bottom: -40,
      left: 0,
      right: 20,
      width: 20,
      height: 40,
      x: 0,
      y: -80,
      toJSON: () => ({}),
    })
    fireEvent.scroll(window)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('un défilement masque toujours l’infobulle d’un survol', () => {
    const b = bouton()
    hover(b)
    afterDelay()
    fireEvent.scroll(window)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })
})

describe('sorties de la page', () => {
  function bouton() {
    render(
      <>
        <TitleTooltipLayer />
        <button type="button" title="Aide">
          Ok
        </button>
      </>,
    )
    return screen.getByRole('button')
  }

  function setVisibility(state: 'visible' | 'hidden') {
    Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => state })
    fireEvent(document, new Event('visibilitychange'))
  }

  afterEach(() => {
    Reflect.deleteProperty(document, 'visibilityState')
  })

  it('la perte du focus de la fenêtre rend le titre et masque l’infobulle', () => {
    const b = bouton()
    hover(b)
    afterDelay()
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    fireEvent(window, new Event('blur'))
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    expect(b).toHaveAttribute('title', 'Aide')
  })

  it('une page qui devient cachée rend le titre ; une page visible ne change rien', () => {
    const b = bouton()
    hover(b)
    afterDelay()
    setVisibility('visible')
    expect(b).toHaveAttribute('title', '')
    expect(screen.getByRole('tooltip')).toBeInTheDocument()
    setVisibility('hidden')
    expect(b).toHaveAttribute('title', 'Aide')
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })
})
