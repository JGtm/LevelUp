/**
 * Tests — LES DEUX EMPLACEMENTS DE LA TOILE (`transport`, `blockOverlays`) : la toile reçoit sa barre de
 * lecture et les superpositions de son bloc comme des FONCTIONS DE RENDU, et leur remet un seul objet de
 * commandes (`replayTransportCommands`). Ce que ces tests prouvent :
 *  - la barre du Standard, rendue par l'emplacement, pilote la toile (lecture, pause, saut) ;
 *  - une AUTRE barre, minimale, branchée sur le même emplacement, pilote la même toile : la toile ne
 *    connaît pas sa barre ;
 *  - `blockOverlays` rend son contenu dans le bloc de la toile, après la légende, hors de la barre ;
 *  - ni la lecture, ni le changement de fonction de rendu ne remontent la barre : c'est le même nœud DOM.
 */
import { act, cleanup, fireEvent, screen } from '@testing-library/react'
import type { ComponentProps, ReactElement } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { renderWithProviders } from '../../../test/render-utils'
import { buildPlayers } from '../../../lib/replay/rosterLogic'
import { REPLAY_TEXT } from '../i18n/i18n'
import { FOLLOW_MIN_LEVEL } from '../model/followLogic'
import { createPlaybackStore } from '../model/playbackStore'
import { usesCompactBar } from '../model/replayFrame'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { makeFrameControl } from '../test/frameControlStub'
import { recordingContext } from '../test/recordingContext'
import { testReplayDoc } from '../test/testDoc'
import { canvasSlotsOf } from './frames/replayFrameSlots'
import { ReplayCanvas } from './ReplayCanvas'
import type { ReplayFrameView } from './frames/replayFrameTypes'
import type { ReplayCommandsRender, ReplaySlotCommands } from './replayTransportCommands'

const T = REPLAY_TEXT.fr

function smallDoc(): ReplayDocumentReady {
  const smallest = [...goFixtureEntries()].sort((a, b) => a.bytes - b.bytes)[0]
  return testReplayDoc(loadGoFixture(smallest).doc)
}

/** Monte la toile avec les emplacements donnés ; le reste des propriétés est le minimum requis. */
function canvasWith(doc: ReplayDocumentReady, transport: ReplayCommandsRender, blockOverlays?: ReplayCommandsRender): ReactElement {
  return <ReplayCanvas {...baseProps(doc)} transport={transport} blockOverlays={blockOverlays} />
}

/** Les propriétés requises, mémorisées par document : deux montages du même document reçoivent les mêmes valeurs. */
const BASE_PROPS = new WeakMap<ReplayDocumentReady, BaseProps>()
type BaseProps = Omit<ComponentProps<typeof ReplayCanvas>, 'transport' | 'blockOverlays'>

function baseProps(doc: ReplayDocumentReady): BaseProps {
  const known = BASE_PROPS.get(doc)
  if (known) return known
  const players = buildPlayers(doc, [])
  const viewpoint = players.find((p) => p.team !== undefined && p.team >= 0 && !p.xuid.startsWith('bot:'))?.xuid ?? null
  const props: BaseProps = {
    doc,
    locale: 'fr',
    playWindow: null,
    playbackStore: createPlaybackStore(),
    allegiance: buildFilmAllegiance(players, viewpoint),
    players,
    viewpoint,
    fullscreen: { active: false, disabled: false, toggle: () => {} },
    frameControl: makeFrameControl(),
  }
  BASE_PROPS.set(doc, props)
  return props
}

beforeEach(() => {
  try {
    localStorage.clear()
  } catch {
    // Pas de stockage : les réglages gardent leurs défauts, ce que ces tests supposent déjà.
  }
  // Aucune boucle d'animation : la lecture change d'état, l'image ne bouge que sur un saut.
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  // Une largeur mesurée : sans elle la toile ne peint rien et l'horloge reste vide.
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    x: 0, y: 0, top: 0, left: 0, bottom: 540, right: 960, width: 960, height: 540, toJSON: () => ({}),
  })
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => recordingContext().ctx as never)
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

/** Une barre minimale, qui ne connaît de la toile que l'objet de commandes. */
const minimalBar: ReplayCommandsRender = (c) => (
  <div data-testid="minimal-bar">
    <button type="button" onClick={c.onTogglePlay}>
      {c.playing ? 'minimal-pause' : 'minimal-lecture'}
    </button>
    <button type="button" onClick={() => c.onSeekBy(10)}>
      minimal-avance
    </button>
    <button type="button" onClick={() => c.onSetSpeed(2)}>
      minimal-vitesse
    </button>
    <span data-testid="minimal-speed">{c.speed}</span>
    <span data-testid="minimal-clock" ref={c.clockRef} />
  </div>
)

describe('ReplayCanvas : la barre du Standard passe par l’emplacement `transport`', () => {
  it('elle pilote la toile : lecture, pause, saut', () => {
    const doc = smallDoc()
    renderWithProviders(canvasWith(doc, canvasSlotsOf('standard').transport))
    expect(screen.getByRole('button', { name: 'Lecture' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Lecture' }))
    expect(screen.getByRole('button', { name: 'Pause' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Pause' }))
    expect(screen.getByRole('button', { name: 'Lecture' })).toBeTruthy()

    const clock = () => screen.getByText(/^\d+:\d{2} \/ \d+:\d{2}$/).textContent
    const before = clock()
    fireEvent.click(screen.getByRole('button', { name: /Avancer de 10 s/ }))
    expect(clock()).not.toBe(before)
  })

  it('le Cinéma et le Plein cadre posent la barre d’une ligne, sans zoom flottant ; le Standard garde la barre complète et son zoom', () => {
    const complete = canvasSlotsOf('standard')
    const compacte = canvasSlotsOf('cinema')
    expect(complete.blockOverlays).toBeDefined()
    expect(compacte.transport).not.toBe(complete.transport)
    expect(compacte.blockOverlays).toBeUndefined()
    expect(canvasSlotsOf('plein-cadre')).toBe(compacte)
    // Le modèle décide, pas un nom de cadre : chaque cadre reçoit le jeu que `usesCompactBar` désigne.
    for (const frame of ['standard', 'cinema', 'plein-cadre'] as const) {
      expect(canvasSlotsOf(frame)).toBe(usesCompactBar(frame) ? compacte : complete)
    }
  })
})

describe('ReplayCanvas : une autre barre pilote la même toile', () => {
  it('lecture, saut et vitesse passent par l’objet de commandes, sans que la toile connaisse la barre', () => {
    renderWithProviders(canvasWith(smallDoc(), minimalBar))
    expect(screen.queryByRole('button', { name: 'Lecture' })).toBeNull()
    expect(screen.getByRole('button', { name: 'minimal-lecture' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'minimal-lecture' }))
    expect(screen.getByRole('button', { name: 'minimal-pause' })).toBeTruthy()

    const clock = () => screen.getByTestId('minimal-clock').textContent ?? ''
    const before = clock()
    fireEvent.click(screen.getByRole('button', { name: 'minimal-avance' }))
    expect(clock()).not.toBe(before)

    expect(screen.getByTestId('minimal-speed').textContent).toBe('1')
    fireEvent.click(screen.getByRole('button', { name: 'minimal-vitesse' }))
    expect(screen.getByTestId('minimal-speed').textContent).toBe('2')
  })

  it('l’objet de commandes porte ce que la barre actuelle reçoit, zoom compris', () => {
    const seen: ReplaySlotCommands[] = []
    renderWithProviders(
      canvasWith(smallDoc(), (c) => {
        seen.push(c)
        return null
      }),
    )
    const commands = seen[seen.length - 1]
    expect(Object.keys(commands).sort()).toEqual(
      [
        'autoPlay', 'cameraFollow', 'capture', 'clockRef', 'frameControl', 'fullscreen', 'locale', 'onRestart', 'onSeekBy',
        'onSetSpeed', 'onTogglePlay', 'onToggleAutoPlay', 'planFeedShown', 'playing', 'sound', 'speed', 'timeline', 'zoom', 'autoRotation', 'thinBar',
      ].sort(),
    )
    expect(typeof commands.zoom.zoomIn).toBe('function')
    // La caméra : sans page autour de la toile, personne n'est suivi et le geste est inerte.
    expect(commands.cameraFollow.player).toBeNull()
    expect(commands.planFeedShown).toBe(false)
    expect(commands.autoRotation).toBeNull()
  })
})

describe('ReplayCanvas : la barre se pose sur le plan au Plein cadre, préférence allumée', () => {
  const dockOf = () => screen.getByTestId('minimal-bar').parentElement as HTMLElement

  it('Plein cadre, préférence par défaut (allumée) : la barre est posée en bas du bloc, hors du flux', () => {
    renderWithProviders(
      <ReplayCanvas {...baseProps(smallDoc())} frameControl={makeFrameControl('plein-cadre')} transport={minimalBar} />,
    )
    expect(dockOf().className).toContain('absolute')
    expect(dockOf().className).toContain('bottom-0')
  })

  it('Plein cadre, préférence éteinte : la barre reste dans le flux, sous le plan', () => {
    localStorage.setItem('replay-thin-bar-full-frame', 'false')
    renderWithProviders(
      <ReplayCanvas {...baseProps(smallDoc())} frameControl={makeFrameControl('plein-cadre')} transport={minimalBar} />,
    )
    expect(dockOf().className).not.toContain('absolute')
  })

  it.each(['standard', 'cinema'] as const)('%s : la barre reste dans le flux, quelle que soit la préférence', (frame) => {
    renderWithProviders(<ReplayCanvas {...baseProps(smallDoc())} frameControl={makeFrameControl(frame)} transport={minimalBar} />)
    expect(dockOf().className).not.toContain('absolute')
  })
})

describe('ReplayCanvas : l’emplacement `blockOverlays`', () => {
  it('rend son contenu dans le bloc de la toile, hors de la barre', () => {
    const { container } = renderWithProviders(
      canvasWith(smallDoc(), minimalBar, (c) => <div data-testid="overlay">{c.locale}</div>),
    )
    const overlay = screen.getByTestId('overlay')
    expect(overlay.textContent).toBe('fr')
    const canvas = container.querySelector('canvas')!
    const block = canvas.parentElement!.parentElement!
    expect(overlay.parentElement).toBe(block)
    expect(screen.getByTestId('minimal-bar').contains(overlay)).toBe(false)
    expect(block.contains(screen.getByTestId('minimal-bar'))).toBe(false)
  })

  it('absent, il ne rend rien', () => {
    renderWithProviders(canvasWith(smallDoc(), minimalBar))
    expect(screen.queryByTestId('overlay')).toBeNull()
  })

  it('la commande de zoom flottante est celle que les cadres y posent, et elle pilote le cadrage', () => {
    const { container } = renderWithProviders(
      canvasWith(smallDoc(), minimalBar, canvasSlotsOf('standard').blockOverlays),
    )
    const group = screen.getByRole('group', { name: T.zoomGroup })
    const block = container.querySelector('canvas')!.parentElement!.parentElement!
    expect(group.parentElement).toBe(block)
    expect(screen.getByText(T.zoomLevelFmt(1))).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: T.zoomIn }))
    expect(screen.queryByText(T.zoomLevelFmt(1))).toBeNull()
  })
})

describe('ReplayCanvas : la barre n’est jamais remontée', () => {
  it('la barre du Standard garde ses nœuds pendant la lecture et à chaque nouvelle fonction de rendu', () => {
    const doc = smallDoc()
    const bar = canvasSlotsOf('standard').transport
    const view = renderWithProviders(canvasWith(doc, bar))
    const play = screen.getByRole('button', { name: 'Lecture' })
    const seek = screen.getByRole('button', { name: /Avancer de 10 s/ })

    fireEvent.click(play)
    expect(screen.getByRole('button', { name: 'Pause' })).toBe(play)
    fireEvent.click(seek)
    expect(screen.getByRole('button', { name: /Avancer de 10 s/ })).toBe(seek)

    // Une fonction de rendu d'identité nouvelle à chaque rendu : le même élément, donc le même nœud.
    act(() => {
      view.rerender(canvasWith(doc, (c) => bar(c)))
    })
    expect(screen.getByRole('button', { name: 'Pause' })).toBe(play)
    expect(screen.getByRole('button', { name: /Avancer de 10 s/ })).toBe(seek)
  })

  it('une autre barre garde aussi son nœud pendant la lecture', () => {
    renderWithProviders(canvasWith(smallDoc(), minimalBar))
    const bar = screen.getByTestId('minimal-bar')
    fireEvent.click(screen.getByRole('button', { name: 'minimal-lecture' }))
    fireEvent.click(screen.getByRole('button', { name: 'minimal-vitesse' }))
    expect(screen.getByTestId('minimal-bar')).toBe(bar)
  })
})

describe('ReplayCanvas : la barre d’une ligne (Cinéma, Plein cadre) pilote la toile', () => {
  const compact = canvasSlotsOf('cinema').transport
  const clock = () => screen.getByText(/^\d+:\d{2} \/ \d+:\d{2}$/).textContent

  it('lecture, pause et saut passent par l’objet de commandes', () => {
    renderWithProviders(canvasWith(smallDoc(), compact))
    fireEvent.click(screen.getByRole('button', { name: 'Lecture' }))
    expect(screen.getByRole('button', { name: 'Pause' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Pause' }))
    const before = clock()
    fireEvent.click(screen.getByRole('button', { name: /Avancer de 10 s/ }))
    expect(clock()).not.toBe(before)
  })

  it('aller à un instant : le champ de la frise compacte déplace l’horloge et pose la progression sur sa racine', () => {
    const { container } = renderWithProviders(canvasWith(smallDoc(), compact))
    const field = container.querySelector<HTMLInputElement>('input[data-replay-timeline]')!
    expect(field).not.toBeNull()
    const before = clock()
    fireEvent.change(field, { target: { value: field.max } })
    expect(clock()).not.toBe(before)
    const host = field.closest<HTMLElement>('[data-replay-cursor-host]')!
    expect(host.style.getPropertyValue('--played-r')).toBe('1')
  })

  it('le zoom de la barre pilote le cadrage de la carte', () => {
    renderWithProviders(canvasWith(smallDoc(), compact))
    expect(screen.getByRole('button', { name: T.zoomOut })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: T.zoomIn }))
    expect(screen.getByRole('button', { name: T.zoomOut })).toBeEnabled()
  })

  it('rien n’est remonté pendant la lecture : mêmes nœuds de barre, de frise et de champ', () => {
    const { container } = renderWithProviders(canvasWith(smallDoc(), compact))
    const play = screen.getByRole('button', { name: 'Lecture' })
    const field = container.querySelector('input[data-replay-timeline]')
    const bar = container.querySelector('[data-replay-transport="compact"]')
    fireEvent.click(play)
    fireEvent.click(screen.getByRole('button', { name: /Avancer de 10 s/ }))
    fireEvent.click(screen.getByRole('button', { name: T.zoomIn }))
    expect(screen.getByRole('button', { name: 'Pause' })).toBe(play)
    expect(container.querySelector('input[data-replay-timeline]')).toBe(field)
    expect(container.querySelector('[data-replay-transport="compact"]')).toBe(bar)
  })

  it('la lecture automatique est un interrupteur de la barre', () => {
    renderWithProviders(canvasWith(smallDoc(), compact))
    const toggle = screen.getByRole('switch', { name: T.autoPlay })
    const was = toggle.getAttribute('aria-checked')
    fireEvent.click(toggle)
    expect(screen.getByRole('switch', { name: T.autoPlay }).getAttribute('aria-checked')).not.toBe(was)
  })
})

describe('ReplayCanvas : la caméra suit le joueur de `cameraFollow`, fiche ouverte ou fermée', () => {
  /** Un joueur dont une vie couvre toute la durée du film : la carte a toujours une position à suivre. */
  function followable(doc: ReplayDocumentReady): ReplayPlayer {
    const end = doc.frameCount
    const points = [{ t: 0, x: 10, y: 20 }, { t: end, x: 30, y: 40 }]
    return { xuid: 'suivi', lives: [{ slot: 1, team: 0, xuid: 'suivi', startFrame: 0, endFrame: end, points }] as never }
  }

  function mount(doc: ReplayDocumentReady, extra: Partial<ComponentProps<typeof ReplayCanvas>>) {
    const seen: ReplaySlotCommands[] = []
    renderWithProviders(
      <ReplayCanvas
        {...baseProps(doc)}
        transport={(c) => {
          seen.push(c)
          return null
        }}
        {...extra}
      />,
    )
    return () => seen[seen.length - 1]
  }

  it('personne n’est suivi : la carte reste entière', () => {
    const doc = smallDoc()
    const last = mount(doc, {})
    expect(last().zoom.level).toBe(1)
  })

  it('un joueur suivi, AUCUNE fiche ouverte (playerFocus absent) : la carte se cadre sur lui', () => {
    const doc = smallDoc()
    const last = mount(doc, { cameraFollow: { player: followable(doc), onFollow: vi.fn() } })
    expect(last().zoom.level).toBeGreaterThanOrEqual(FOLLOW_MIN_LEVEL)
  })

  it('la fiche ouverte d’un AUTRE joueur, avec Seul, ne retire pas la caméra du joueur suivi', () => {
    const doc = smallDoc()
    const players = baseProps(doc).players ?? []
    const suivi = followable(doc)
    const autre = players.find((p) => p !== suivi) ?? suivi
    const last = mount(doc, {
      cameraFollow: { player: suivi, onFollow: vi.fn() },
      playerFocus: { player: autre, follow: false, solo: true, hide: false },
    })
    expect(last().zoom.level).toBeGreaterThanOrEqual(FOLLOW_MIN_LEVEL)
  })

  it('le drapeau « suivre » d’une fiche ouverte ne cadre rien sans joueur suivi côté caméra', () => {
    const doc = smallDoc()
    const last = mount(doc, { playerFocus: { player: followable(doc), follow: true, solo: false, hide: false } })
    expect(last().zoom.level).toBe(1)
  })

  it('un geste manuel de zoom arrête le suivi : la page est prévenue (onFollowStop) ; le cadrage du suivi, lui, ne l’est pas', () => {
    const doc = smallDoc()
    const onFollowStop = vi.fn()
    const last = mount(doc, { cameraFollow: { player: followable(doc), onFollow: vi.fn() }, onFollowStop })
    expect(onFollowStop).not.toHaveBeenCalled()
    act(() => last().zoom.zoomIn())
    expect(onFollowStop).toHaveBeenCalled()
  })
})

describe('ReplayCanvas : le fil des éliminations du plan (`planFeed`)', () => {
  /** La partie de la vue de la page que le fil du plan lit : le fil, l'instant lu, les noms. */
  function planFeedView(doc: ReplayDocumentReady): ReplayFrameView {
    const props = baseProps(doc)
    const kill = {
      tMs: 0, replayMs: 0, xuid: 'a', victimXuid: 'b', victimGamertag: 'Victime', weaponKey: '', weaponLabel: '',
      weaponImageUrl: '', weaponTinted: false, medals: [],
    }
    return {
      model: {
        feed: [{ key: 'k0', replayMs: 0, kill, medal: null, death: null }],
        window: null, identity: new Map([['a', { gamertag: 'Tueur' }]]), allegiance: props.allegiance, marks: new Map(),
      },
      nowMs: 0, locale: 'fr', feedZoneOf: undefined,
    } as unknown as ReplayFrameView
  }

  it('les emplacements du cadre portent le fil du plan quand la vue est donnée, la barre compacte restant la même', () => {
    const doc = smallDoc()
    const sans = canvasSlotsOf('cinema')
    const avec = canvasSlotsOf('cinema', planFeedView(doc))
    expect(sans.planFeed).toBeUndefined()
    expect(sans.blockOverlays).toBeUndefined()
    expect(avec.planFeed).toBe(true)
    expect(avec.blockOverlays).toBeDefined()
    expect(avec.transport).toBe(sans.transport)
    expect(canvasSlotsOf('plein-cadre', planFeedView(doc)).planFeed).toBe(true)
  })

  it('le Standard ne pose jamais le fil du plan, même quand la vue est donnée', () => {
    const standard = canvasSlotsOf('standard', planFeedView(smallDoc()))
    expect(standard).toBe(canvasSlotsOf('standard'))
    expect(standard.planFeed).toBeUndefined()
  })

  it('la toile dit à la fonction de rendu si le fil du plan est montré, et le fil se pose dans le bloc de la toile', () => {
    const doc = smallDoc()
    const slots = canvasSlotsOf('cinema', planFeedView(doc))
    const { container } = renderWithProviders(
      <ReplayCanvas {...baseProps(doc)} transport={slots.transport} blockOverlays={slots.blockOverlays} planFeed={slots.planFeed} />,
    )
    const fil = container.querySelector('[data-replay-plan-feed]')
    expect(fil).not.toBeNull()
    expect(fil?.parentElement).toBe(container.querySelector('canvas')!.parentElement!.parentElement)
    expect(fil?.textContent).toContain('Tueur')
    expect(fil?.textContent).toContain('Victime')
  })

  it('sans `planFeed` (cadre sans fil sur le plan), la fonction de rendu reçoit « non montré » et rien ne se pose', () => {
    const doc = smallDoc()
    const slots = canvasSlotsOf('cinema', planFeedView(doc))
    const { container } = renderWithProviders(
      <ReplayCanvas {...baseProps(doc)} transport={slots.transport} blockOverlays={slots.blockOverlays} />,
    )
    expect(container.querySelector('[data-replay-plan-feed]')).toBeNull()
  })

  describe("l'option « Fil des éliminations » de la vue", () => {
    const montage = (planFeed: boolean) => {
      const doc = smallDoc()
      const slots = canvasSlotsOf('cinema', planFeedView(doc))
      return renderWithProviders(
        <ReplayCanvas {...baseProps(doc)} transport={slots.transport} blockOverlays={slots.blockOverlays} planFeed={planFeed} />,
      )
    }
    const ouvrirLaVue = () => fireEvent.click(screen.getByRole('button', { name: /^Vue / }))

    it("éteinte, le fil du plan n'est pas rendu et la légende de la chaleur retrouve le bas ; allumée, il l'est", () => {
      localStorage.setItem('replay-show-heatmap', 'true')
      localStorage.setItem('replay-show-plan-feed', 'false')
      const eteint = montage(true)
      expect(eteint.container.querySelector('[data-replay-plan-feed]')).toBeNull()
      expect(eteint.container.querySelector<HTMLElement>(`[title="${T.heatLegendHint}"]`)?.className).toContain('bottom-2')
      eteint.unmount()

      localStorage.setItem('replay-show-plan-feed', 'true')
      const allume = montage(true)
      expect(allume.container.querySelector('[data-replay-plan-feed]')).not.toBeNull()
      expect(allume.container.querySelector<HTMLElement>(`[title="${T.heatLegendHint}"]`)?.className).toContain('top-2')
    })

    it("l'option est dans le panneau quand le cadre pose le fil, et la basculer retire puis rend le fil, sans remonter la toile", () => {
      const { container } = montage(true)
      const toile = container.querySelector('canvas')
      expect(container.querySelector('[data-replay-plan-feed]')).not.toBeNull()
      ouvrirLaVue()
      const option = screen.getByRole('switch', { name: T.frames.planFeed })
      expect(option).toHaveAttribute('aria-checked', 'true')
      fireEvent.click(option)
      expect(container.querySelector('[data-replay-plan-feed]')).toBeNull()
      fireEvent.click(screen.getByRole('switch', { name: T.frames.planFeed }))
      expect(container.querySelector('[data-replay-plan-feed]')).not.toBeNull()
      expect(container.querySelector('canvas')).toBe(toile)
    })

    it("sans `planFeed` (Standard, repli), l'option n'est pas dans le panneau", () => {
      montage(false)
      ouvrirLaVue()
      expect(screen.getByRole('region', { name: T.viewButton })).toBeInTheDocument()
      expect(screen.queryByRole('switch', { name: T.frames.planFeed })).toBeNull()
    })
  })

  describe('la légende de la carte de chaleur', () => {
    const legende = (c: HTMLElement) => c.querySelector<HTMLElement>(`[title="${T.heatLegendHint}"]`)

    it('en bas à gauche du bloc par défaut, en haut à gauche quand le cadre pose le fil sur le plan', () => {
      localStorage.setItem('replay-show-heatmap', 'true')
      const doc = smallDoc()
      const sans = renderWithProviders(canvasWith(doc, canvasSlotsOf('cinema').transport))
      expect(legende(sans.container)?.className).toContain('bottom-2')
      expect(legende(sans.container)?.className).not.toContain('top-2')
      sans.unmount()

      const slots = canvasSlotsOf('cinema', planFeedView(doc))
      const avec = renderWithProviders(
        <ReplayCanvas {...baseProps(doc)} transport={slots.transport} blockOverlays={slots.blockOverlays} planFeed={slots.planFeed} />,
      )
      expect(legende(avec.container)?.className).toContain('top-2')
      expect(legende(avec.container)?.className).not.toContain('bottom-2')
    })
  })
})
