/**
 * useCopyToClipboard.test.tsx — mécanique du hook canonique « copier + coche ».
 *
 * Ce que chaque test prouve :
 *  - copie réussie : `copied` passe à vrai puis retombe à faux après la durée
 *    unique (`COPIED_FEEDBACK_MS`), pas avant ;
 *  - un 2e clic réarme la fenêtre depuis zéro (la coche de la 2e copie ne
 *    disparaît pas au minuteur de la 1re — le bug des quatre copies) ;
 *  - échec du presse-papier : `copied` reste FAUX (jamais de faux « copié ») et
 *    l'échec est journalisé au lieu d'être avalé ;
 *  - démontage pendant le délai : aucun setState après démontage (aucun
 *    avertissement React).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act, render } from '@testing-library/react'

import { COPIED_FEEDBACK_MS, useCopyToClipboard } from './useCopyToClipboard'
import { log } from './_logger'

/** Sonde : expose l'état du hook et déclenche la copie à la demande. */
function Probe({ text, onReady }: { text: string; onReady: (copy: () => Promise<void>) => void }) {
  const { copy, copied } = useCopyToClipboard()
  onReady(() => copy(text))
  return <span data-testid="state">{copied ? 'copied' : 'idle'}</span>
}

function writeTextMock(): ReturnType<typeof vi.fn> {
  const fn = vi.fn().mockResolvedValue(undefined)
  Object.defineProperty(globalThis.navigator, 'clipboard', {
    value: { writeText: fn },
    configurable: true,
  })
  return fn
}

describe('useCopyToClipboard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    log._resetForTests()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('copie réussie : copied vrai, puis faux après COPIED_FEEDBACK_MS', async () => {
    const writeText = writeTextMock()
    let run: () => Promise<void> = async () => {}
    const { getByTestId } = render(<Probe text="ABC-123" onReady={(c) => (run = c)} />)

    await act(async () => {
      await run()
    })
    expect(writeText).toHaveBeenCalledWith('ABC-123')
    expect(getByTestId('state').textContent).toBe('copied')

    // Juste avant l'échéance : la coche tient encore.
    act(() => {
      vi.advanceTimersByTime(COPIED_FEEDBACK_MS - 1)
    })
    expect(getByTestId('state').textContent).toBe('copied')

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(getByTestId('state').textContent).toBe('idle')
  })

  it('2e copie : le minuteur est réarmé (la coche ne tombe pas au minuteur de la 1re)', async () => {
    writeTextMock()
    let run: () => Promise<void> = async () => {}
    const { getByTestId } = render(<Probe text="ABC-123" onReady={(c) => (run = c)} />)

    await act(async () => {
      await run()
    })
    act(() => {
      vi.advanceTimersByTime(COPIED_FEEDBACK_MS - 500)
    })
    await act(async () => {
      await run()
    })

    // Instant où le 1er minuteur aurait expiré : la coche de la 2e copie tient.
    act(() => {
      vi.advanceTimersByTime(500)
    })
    expect(getByTestId('state').textContent).toBe('copied')

    act(() => {
      vi.advanceTimersByTime(COPIED_FEEDBACK_MS)
    })
    expect(getByTestId('state').textContent).toBe('idle')
  })

  it('échec du presse-papier : copied reste faux et l’échec est journalisé', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('NotAllowedError'))
    Object.defineProperty(globalThis.navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    })
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    let run: () => Promise<void> = async () => {}
    const { getByTestId } = render(<Probe text="ABC-123" onReady={(c) => (run = c)} />)

    await act(async () => {
      await run()
    })

    expect(getByTestId('state').textContent).toBe('idle')
    expect(errorSpy).toHaveBeenCalled()
  })

  it('démontage pendant le délai : aucun avertissement React (pas de setState après unmount)', async () => {
    writeTextMock()
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    let run: () => Promise<void> = async () => {}
    const { unmount } = render(<Probe text="ABC-123" onReady={(c) => (run = c)} />)

    await act(async () => {
      await run()
    })
    unmount()
    act(() => {
      vi.advanceTimersByTime(COPIED_FEEDBACK_MS * 2)
    })

    expect(warnSpy).not.toHaveBeenCalled()
    expect(errorSpy).not.toHaveBeenCalled()
  })
})
