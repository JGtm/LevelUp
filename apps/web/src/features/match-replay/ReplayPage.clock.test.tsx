/**
 * Tests — L'HORLOGE DE LA BARRE DE LECTURE EN PAUSE, à la bascule de cadre.
 *
 * Le texte « m:ss / m:ss » n'est écrit que par le dessin de la toile (`useReplayClock`), et le Standard et le
 * Cinéma ne montent pas la même barre de lecture. En pause, rien ne redessine : la barre qui vient d'être
 * montée doit pourtant porter l'horloge, sans attendre un geste. La largeur de la toile ne change pas dans
 * ces tests (jsdom ne met rien en page), donc aucun redessin ne vient de la taille.
 */
import { act, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { requestExportLayout } from './export/exportLayoutStore'
import { goToFrame, mountReplayPage, smallDoc, stubReplayPage, unstubReplayPage } from './test/replayPageHarness'

vi.mock('@tanstack/react-router', async (importOriginal) =>
  (await import('./test/replayPageRouter')).routerMock(await importOriginal<typeof import('@tanstack/react-router')>()),
)

const FORME_HORLOGE = /^\d+:\d{2} \/ \d+:\d{2}$/

beforeEach(() => {
  stubReplayPage()
  // Une boîte mesurée et FIXE : la toile se dessine (largeur non nulle) et ne change jamais de taille.
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(
    () => ({ width: 1000, height: 600, top: 0, left: 0, right: 1000, bottom: 600, x: 0, y: 0, toJSON: () => ({}) }),
  )
})

afterEach(() => {
  act(() => requestExportLayout(null))
  unstubReplayPage()
})

/** Attend la première horloge du Standard : la toile est mesurée puis dessinée après le montage. */
async function attendreLHorloge(): Promise<string[]> {
  await waitFor(() => expect(horloges()).toHaveLength(1))
  return horloges()
}

/** Les textes d'horloge « m:ss / m:ss » de la page : une barre de lecture = une horloge. */
function horloges(): string[] {
  return screen.queryAllByText(FORME_HORLOGE).map((el) => el.textContent ?? '')
}

describe('ReplayPage : l’horloge de la barre en pause', () => {
  it('le Standard, ouvert en pause, porte l’horloge', async () => {
    await mountReplayPage(smallDoc())
    expect(await attendreLHorloge()).toHaveLength(1)
  })

  it.each(['cinema', 'plein-cadre'] as const)('en pause, du Standard vers le %s (barre compacte) : l’horloge y est, la même', async (cible) => {
    const { container } = await mountReplayPage(smallDoc())
    const avant = await attendreLHorloge()
    goToFrame(cible)
    const compacte = container.querySelector('[data-replay-transport="compact"]')
    expect(compacte).not.toBeNull()
    expect(compacte?.textContent).toMatch(/\d+:\d{2} \/ \d+:\d{2}/)
    expect(horloges()).toEqual(avant)
  })

  it('en pause, aller-retour entre les cadres : l’horloge est toujours là, sans geste', async () => {
    await mountReplayPage(smallDoc())
    const avant = await attendreLHorloge()
    for (const frame of ['cinema', 'standard', 'plein-cadre', 'cinema', 'standard'] as const) {
      goToFrame(frame)
      expect(horloges(), frame).toEqual(avant)
    }
  })
})
