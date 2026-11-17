/**
 * replayPageHarness.tsx — LE MONTAGE DE LA VRAIE PAGE DU REJEU dans les tests (`ReplayPage.*.test.tsx`) : la vraie page,
 * la vraie toile (contexte 2D enregistreur), le réseau simulé ; seuls le routeur (`replayPageRouter`) et la boucle
 * d'animation sont remplacés.
 *
 * Un test pose l'environnement dans son `beforeEach` (`stubReplayPage`), le défait dans son `afterEach`
 * (`unstubReplayPage`), sert un document et un contexte de match (`mountReplayPage`) puis change de cadre
 * (`goToFrame`). Les cas propres à un fichier (largeur du corps, rectangles, export) restent dans ce fichier.
 */
import { act, cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { expect, vi } from 'vitest'

import { API_BASE_URL } from '@/lib/api/client'
import { testCapabilities } from '@/test/capabilities'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'

import { REPLAY_TEXT } from '../i18n/i18n'
import type { ReplayFrame } from '../model/replayFrame'
import { ReplayPage } from '../ReplayPage'
import { goFixtureEntries, loadGoFixture } from './goFixtures'
import { recordingContext } from './recordingContext'
import { resetRoute } from './replayPageRouter'

/** L'adresse du document de rejeu du match affiché par défaut. */
export const REPLAY_API = `${API_BASE_URL}/players/jgtm/matches/match-1/replay`

/** Le contexte du match de départ : de quoi nommer le match dans les en-têtes, sans tableau de score. */
export const MATCH_CONTEXT = {
  header: {
    map_ui: 'Detachment',
    mode_ui: 'Assassin en équipe',
    playlist_label: 'Partie rapide',
    start_time_label: '10 nov. 2026, 21:40',
  },
  scoreboard: [],
  highlight_events: [],
  media: null,
  source_unavailable: false,
}

/** Le plus petit document de rejeu du manifeste de Go : il a des pistes, donc la page est « prête ». */
export function smallDoc() {
  const smallest = [...goFixtureEntries()].sort((a, b) => a.bytes - b.bytes)[0]
  return loadGoFixture(smallest).doc
}

/** Sert le document de rejeu du match affiché par défaut. */
export function serveDoc(doc: object): void {
  server.use(http.get(REPLAY_API, () => HttpResponse.json(doc)))
}

/**
 * Pose l'environnement de la page : route de départ, aucune boucle d'animation, contexte 2D enregistreur, langue
 * française, application sans génération de rejeu.
 */
export function stubReplayPage(): void {
  resetRoute()
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => recordingContext().ctx as never)
  act(() => useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities({ can_generate_replays: false }) }))
}

/** Défait l'environnement : l'arbre est démonté, les globales et les espions restitués. */
export function unstubReplayPage(): void {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  act(() => useAppShellStore.setState({ capabilities: null }))
}

/** Sert le document et le contexte du match, monte la page et attend sa toile. */
export async function mountReplayPage(doc: object = smallDoc(), context: object = MATCH_CONTEXT) {
  serveDoc(doc)
  server.use(http.get(`${REPLAY_API}/context`, () => HttpResponse.json(context)))
  const view = renderWithProviders(<ReplayPage />)
  await waitFor(() => expect(view.container.querySelector('canvas')).not.toBeNull())
  return view
}

/** Change de cadre par l'icône de la barre de lecture. */
export function goToFrame(frame: ReplayFrame): void {
  const icons = within(screen.getByRole('group', { name: REPLAY_TEXT.fr.frames.groupLabel }))
  fireEvent.click(icons.getByRole('button', { name: REPLAY_TEXT.fr.frames.frameTip(frame) }))
}
