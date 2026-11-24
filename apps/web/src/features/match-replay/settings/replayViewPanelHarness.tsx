/**
 * replayViewPanelHarness.tsx — LE BOUTON « VUE » ET SON PANNEAU, montés sur les VRAIS réglages (`useReplaySettings`,
 * `useReplayViewMenu`, la mémoire des vues) avec un film, un fond, un relief et un son que chaque test choisit. Il
 * FOURNIT, il ne juge pas : aucune assertion. Partagé par les tests du panneau, de l'accès rapide et du bouton.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { vi } from 'vitest'

import { DEFAULT_LOCALE } from '@/lib/i18n/locale'
import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import type { ReplayRelief } from '../hooks/useReplayRelief'
import type { ReplayLocale } from '../i18n/i18n'
import type { ReplaySound } from '../sound/useReplaySound'
import { makeReplayRelief } from '../test/reliefFixtures'
import { FULL_FILM } from '../test/settingsContextFixtures'
import { NO_SOUND } from '../test/soundFixtures'
import { testReplayDoc } from '../test/testDoc'
import type { FilmLayerAvailability } from './filmLayerAvailability'
import { ReplayViewMenu } from './ReplayViewMenu'
import type { ReplayOrientation } from './useReplayOrientation'
import { useReplaySettings } from './useReplaySettings'
import { useReplayViewMenu } from './useReplayViewMenu'

export interface HarnessOptions {
  film?: FilmLayerAvailability
  drawn?: MapBackgroundVariant
  planFeedAvailable?: boolean
  thinBarAvailable?: boolean
  relief?: ReplayRelief
  sound?: ReplaySound
  orientation?: ReplayOrientation
  heatKillsAvailable?: boolean
  locale?: ReplayLocale
  matchKey?: string
}

const DOC = testReplayDoc()
const ALLEGIANCE = buildFilmAllegiance([], null)
const NO_ORIENTATION: ReplayOrientation = { retained: null, setTurns: vi.fn(), available: false }

// eslint-disable-next-line react-refresh/only-export-components -- fabrique de test : le composant monté n est pas exporté, seules les fonctions de rendu le sont.
function Harness(o: HarnessOptions) {
  const settings = useReplaySettings()
  const view = useReplayViewMenu({
    settings,
    orientation: o.orientation ?? NO_ORIENTATION,
    sound: o.sound ?? NO_SOUND,
    available: o.film ?? FULL_FILM,
    heat: { mode: 'presence', killsAvailable: o.heatKillsAvailable ?? true },
    relief: o.relief ?? makeReplayRelief(),
    drawn: o.drawn ?? 'rendu',
    planFeedAvailable: o.planFeedAvailable ?? true,
    thinBarAvailable: o.thinBarAvailable ?? false,
    locale: o.locale ?? DEFAULT_LOCALE,
    feed: { doc: DOC, entries: [], allegiance: ALLEGIANCE },
    matchKey: o.matchKey ?? 'jg/m1',
  })
  return (
    <div className="relative">
      <ReplayViewMenu {...view} />
    </div>
  )
}

/** Monte le bouton « Vue » (panneau fermé). */
export function renderViewMenu(options: HarnessOptions = {}) {
  return render(<Harness {...options} />)
}

/** Monte le bouton « Vue » et ouvre le panneau. */
export function renderOpenPanel(options: HarnessOptions = {}) {
  const view = renderViewMenu(options)
  fireEvent.click(screen.getByRole('button', { name: /^(Vue|View)\b/ }))
  return view
}

