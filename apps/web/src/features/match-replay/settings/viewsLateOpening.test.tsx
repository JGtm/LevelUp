/**
 * viewsLateOpening.test.tsx — LA VUE D'OUVERTURE ET CE QUE LE MATCH PROPOSE TARD : la probabilité de réapparition n'est
 * proposée qu'à la réponse du serveur, les répliques qu'une fois la table téléchargée. La vue d'ouverture (ou la vue
 * rattachée à l'enjeu) vaut pour tout réglage qui devient proposé APRÈS l'ouverture, tant que le lecteur n'a rien changé
 * lui-même à ce rejeu ; au premier geste, plus aucune application tardive.
 */
import { act, renderHook } from '@testing-library/react'
import { useState } from 'react'
import { beforeEach, describe, expect, it } from 'vitest'

import { FULL_FILM, makeSettingsContext } from '../test/settingsContextFixtures'
import type { SettingsContext } from './settingsAvailability'
import type { SettingsBinding } from './settingsBinding'
import { factoryValues, type SettingValues } from './settingsTable'
import { useViewsController } from './useViewsController'

const film = (over: Partial<typeof FULL_FILM>) => ({ ...FULL_FILM, ...over })
/** Un match dont ni la probabilité ni les répliques ne sont encore proposées. */
const EARLY = makeSettingsContext({ film: film({ respawnForecast: false, chatter: { lines: false, voice: false } }) })
const WITH_FORECAST = makeSettingsContext({ film: film({ respawnForecast: true, chatter: { lines: false, voice: false } }) })
const WITH_CHATTER = makeSettingsContext({ film: film({ respawnForecast: true, chatter: { lines: true, voice: true } }) })

function mount(matchKey: string, ctx: SettingsContext) {
  return renderHook(
    (props: { ctx: SettingsContext; matchKey: string }) => {
      const [values, setValues] = useState<SettingValues>(factoryValues())
      const binding: SettingsBinding = { values, set: (id, value) => setValues((v) => ({ ...v, [id]: value })) }
      const controller = useViewsController({ ctx: props.ctx, binding, mode: 'flag', matchKey: props.matchKey })
      return { controller, values, binding }
    },
    { initialProps: { ctx, matchKey } },
  )
}

/** Pose « Combat » comme vue d'ouverture, puis ferme ce rejeu : le suivant s'ouvre dessus. */
function chooseOpening(id: string | null) {
  const setup = mount('reglage', EARLY)
  act(() => setup.result.current.controller.setDefault(id))
  setup.unmount()
}

beforeEach(() => window.localStorage.clear())

describe('la vue d’ouverture et les réglages proposés après l’ouverture', () => {
  it('la probabilité, puis les répliques, rejoignent la vue d’ouverture ; la vue n’est jamais « modifiée »', () => {
    chooseOpening('combat')
    const { result, rerender } = mount('m2', EARLY)
    expect(result.current.values.forecast).toBe(true) // pas encore proposée : l'usine reste
    rerender({ ctx: WITH_FORECAST, matchKey: 'm2' })
    expect(result.current.values.forecast).toBe(false) // Combat éteint la probabilité
    expect(result.current.controller.changed).toEqual([])
    rerender({ ctx: WITH_CHATTER, matchKey: 'm2' })
    expect(result.current.values).toMatchObject({ chatterBubbles: false, chatterFeed: false, chatterVoice: false, famEnemy: true, famPraise: false })
    expect(result.current.controller.changed).toEqual([])
    expect(result.current.controller.state.activeId).toBe('combat')
  })

  it('un geste du lecteur entre les deux : son choix reste, plus aucune application tardive', () => {
    chooseOpening('combat')
    const { result, rerender } = mount('m2', EARLY)
    act(() => result.current.binding.set('aim', false)) // le lecteur change quelque chose
    rerender({ ctx: WITH_FORECAST, matchKey: 'm2' })
    expect(result.current.values.forecast).toBe(true)
    rerender({ ctx: WITH_CHATTER, matchKey: 'm2' })
    expect(result.current.values.chatterBubbles).toBe(true)
  })

  it('appliquer une autre vue à la main met fin à l’application tardive', () => {
    chooseOpening('combat')
    const { result, rerender } = mount('m2', EARLY)
    act(() => result.current.controller.apply('complet'))
    rerender({ ctx: WITH_FORECAST, matchKey: 'm2' })
    expect(result.current.values.forecast).toBe(true)
    expect(result.current.controller.state.activeId).toBe('complet')
  })

  it('sans vue d’ouverture, rien ne change quand le match propose plus', () => {
    chooseOpening(null)
    const { result, rerender } = mount('m2', EARLY)
    const before = result.current.values
    rerender({ ctx: WITH_CHATTER, matchKey: 'm2' })
    expect(result.current.values).toEqual(before)
  })
})
