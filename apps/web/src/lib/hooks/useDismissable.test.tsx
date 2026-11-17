/**
 * useDismissable — Échap et clic dehors ferment la couche ouverte ; rien n'est écouté couche fermée ;
 * le geste dehors se choisit (`mousedown`, `pointerdown`, aucun) ; un élément `ignore` n'est pas
 * « dehors » ; une autre couche d'Échap ouverte retient Échap ; deux couches ouvertes, Échap ne
 * ferme que celle du dessus.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { useRef } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { useDismissable, type DismissableOptions } from './useDismissable'

function Sonde({
  open = true,
  onClose,
  options,
}: {
  open?: boolean
  onClose: (reason: string) => void
  options?: DismissableOptions<HTMLDivElement>
}) {
  const ref = useDismissable(open, onClose, options)
  return (
    <div>
      <button data-testid="dehors">dehors</button>
      <div ref={ref} data-testid="couche">
        <button data-testid="dedans">dedans</button>
      </div>
    </div>
  )
}

describe('useDismissable', () => {
  it('Échap ferme, avec la raison « escape » ; une autre touche non', () => {
    const onClose = vi.fn()
    render(<Sonde onClose={onClose} />)
    fireEvent.keyDown(window, { key: 'Enter' })
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledExactlyOnceWith('escape')
  })

  it('un mousedown dehors ferme (défaut), pas un mousedown dedans', () => {
    const onClose = vi.fn()
    render(<Sonde onClose={onClose} />)
    fireEvent.mouseDown(screen.getByTestId('dedans'))
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.mouseDown(screen.getByTestId('dehors'))
    expect(onClose).toHaveBeenCalledExactlyOnceWith('outside')
  })

  it('avec « pointerdown », le pointerdown dehors ferme et le mousedown ne ferme pas', () => {
    const onClose = vi.fn()
    render(<Sonde onClose={onClose} options={{ outside: 'pointerdown' }} />)
    fireEvent.mouseDown(screen.getByTestId('dehors'))
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.pointerDown(screen.getByTestId('dehors'))
    expect(onClose).toHaveBeenCalledExactlyOnceWith('outside')
  })

  it('avec « null », aucun clic dehors ne ferme, Échap ferme toujours', () => {
    const onClose = vi.fn()
    render(<Sonde onClose={onClose} options={{ outside: null }} />)
    fireEvent.mouseDown(screen.getByTestId('dehors'))
    fireEvent.pointerDown(screen.getByTestId('dehors'))
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledWith('escape')
  })

  it('couche fermée : ni Échap ni clic dehors ne sont écoutés', () => {
    const onClose = vi.fn()
    render(<Sonde open={false} onClose={onClose} />)
    fireEvent.keyDown(window, { key: 'Escape' })
    fireEvent.mouseDown(screen.getByTestId('dehors'))
    expect(onClose).not.toHaveBeenCalled()
  })

  it('le déclencheur `ignore` n’est pas « dehors »', () => {
    const onClose = vi.fn()
    function AvecDeclencheur() {
      const trigger = useRef<HTMLButtonElement>(null)
      const ref = useDismissable(true, onClose, { ignore: trigger })
      return (
        <div>
          <button ref={trigger} data-testid="declencheur">
            ouvrir
          </button>
          <div ref={ref} />
          <button data-testid="ailleurs">ailleurs</button>
        </div>
      )
    }
    render(<AvecDeclencheur />)
    fireEvent.mouseDown(screen.getByTestId('declencheur'))
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.mouseDown(screen.getByTestId('ailleurs'))
    expect(onClose).toHaveBeenCalledExactlyOnceWith('outside')
  })

  it('une autre couche d’Échap ouverte retient Échap ; la couche elle-même ne se compte pas', () => {
    const onClose = vi.fn()
    const SELECTOR = '[data-test-layer]'
    function Couche({ avecAutre }: { avecAutre: boolean }) {
      const ref = useDismissable(true, onClose, { innerLayerSelector: SELECTOR })
      return (
        <div>
          <div ref={ref} data-test-layer="" />
          {avecAutre && <div data-test-layer="" data-testid="autre" />}
        </div>
      )
    }
    const { rerender } = render(<Couche avecAutre={false} />)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)

    rerender(<Couche avecAutre />)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)

    rerender(<Couche avecAutre={false} />)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(2)
  })

  it('un onClose neuf est lu à l’événement suivant, sans rebrancher les écouteurs', () => {
    const premier = vi.fn()
    const second = vi.fn()
    const spy = vi.spyOn(window, 'addEventListener')
    const { rerender } = render(<Sonde onClose={premier} />)
    const branchements = spy.mock.calls.length
    rerender(<Sonde onClose={second} />)
    expect(spy.mock.calls.length).toBe(branchements)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(premier).not.toHaveBeenCalled()
    expect(second).toHaveBeenCalledTimes(1)
    spy.mockRestore()
  })

  it('les écouteurs sont retirés au démontage', () => {
    const onClose = vi.fn()
    const { unmount } = render(<Sonde onClose={onClose} />)
    unmount()
    fireEvent.keyDown(window, { key: 'Escape' })
    fireEvent.mouseDown(document.body)
    expect(onClose).not.toHaveBeenCalled()
  })
})

/** Une couche nommée : `onClose` dit laquelle s'est fermée. */
function Couche({
  nom,
  open = true,
  onClose,
  options,
}: {
  nom: string
  open?: boolean
  onClose: (nom: string) => void
  options?: DismissableOptions<HTMLDivElement>
}) {
  const ref = useDismissable(open, () => onClose(nom), options)
  return <div ref={ref} data-testid={nom} />
}

describe('useDismissable — deux couches ouvertes', () => {
  const echap = () => fireEvent.keyDown(window, { key: 'Escape' })

  it('Échap ne ferme que la couche ouverte en dernier ; la frappe suivante ferme celle du dessous', () => {
    const onClose = vi.fn()
    const { rerender } = render(
      <>
        <Couche nom="dessous" onClose={onClose} />
        <Couche nom="dessus" onClose={onClose} />
      </>,
    )
    echap()
    expect(onClose).toHaveBeenCalledExactlyOnceWith('dessus')

    rerender(
      <>
        <Couche nom="dessous" onClose={onClose} />
        <Couche nom="dessus" open={false} onClose={onClose} />
      </>,
    )
    echap()
    expect(onClose).toHaveBeenCalledTimes(2)
    expect(onClose).toHaveBeenLastCalledWith('dessous')
  })

  it('le dessus est celui qui s’est ouvert en dernier, pas celui qui est rendu en dernier', () => {
    const onClose = vi.fn()
    const { rerender } = render(
      <>
        <Couche nom="premiere" open={false} onClose={onClose} />
        <Couche nom="seconde" onClose={onClose} />
      </>,
    )
    rerender(
      <>
        <Couche nom="premiere" onClose={onClose} />
        <Couche nom="seconde" onClose={onClose} />
      </>,
    )
    echap()
    expect(onClose).toHaveBeenCalledExactlyOnceWith('premiere')
  })

  it('une couche retenue par une couche intérieure hors registre laisse celle du dessous se fermer', () => {
    const onClose = vi.fn()
    const SELECTOR = '[data-test-layer]'
    const interieure = document.createElement('div')
    interieure.setAttribute('data-test-layer', '')
    document.body.appendChild(interieure)
    try {
      render(
        <>
          <Couche nom="dessous" onClose={onClose} />
          <Couche nom="retenue" onClose={onClose} options={{ innerLayerSelector: SELECTOR }} />
        </>,
      )
      echap()
      expect(onClose).toHaveBeenCalledExactlyOnceWith('dessous')
    } finally {
      interieure.remove()
    }
  })

  it('la frappe n’est prise que par une couche : aucune ne la reçoit une fois toutes fermées', () => {
    const onClose = vi.fn()
    const { unmount } = render(
      <>
        <Couche nom="a" onClose={onClose} />
        <Couche nom="b" onClose={onClose} />
      </>,
    )
    unmount()
    echap()
    expect(onClose).not.toHaveBeenCalled()
  })
})
