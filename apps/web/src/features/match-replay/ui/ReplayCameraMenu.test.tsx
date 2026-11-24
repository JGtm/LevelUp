/**
 * Tests — ReplayCameraMenu (le menu de la caméra de la barre de lecture compacte).
 *
 * Ce qu'ils protègent :
 *  - DEUX LISTES, « Suivre » (avec « Aucun ») et « Point de vue », rangées par camp, bots compris ; la liste du
 *    point de vue est celle de la frise, ses options inertes le restent ;
 *  - le joueur suivi est coché et nommé dans le nom du bouton ; sans suivi, c'est « Aucun » qui est coché ;
 *  - choisir un joueur de « Suivre » est le geste de la fiche (clé de ROSTER, jamais la clé du point de vue d'un
 *    bot) ; choisir un point de vue pose le point de vue ; les deux gestes ne se mêlent pas ;
 *  - le panneau se ferme au choix, par Échap (avant de sortir d'un Plein cadre) et par clic dehors ; le focus
 *    revient au bouton après un choix ou Échap, et Espace lance toujours la lecture ensuite ;
 *  - sans aucun joueur, ni bouton ni liste : jamais une commande qui n'ouvre rien.
 */
import { act, fireEvent, render, renderHook, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { useReplayFrame } from '../hooks/useReplayFrame'
import { REPLAY_ESCAPE_LAYER_ATTR, useReplayFullscreen } from '../hooks/useReplayFullscreen'
import { useReplayShortcuts } from '../hooks/useReplayShortcuts'
import type { ViewpointOptionGroup } from '../model/viewpointOptions'
import { ReplayCameraMenu } from './ReplayCameraMenu'

const GROUPS: readonly ViewpointOptionGroup[] = [
  {
    key: 'camp:0',
    label: 'Cobalt',
    options: [
      { value: 'x1', rosterKey: 'x1', label: 'JGtm', disabled: false, title: 'JGtm' },
      { value: 'bid(3.0)', rosterKey: 'bot:Cortana', label: 'Cortana', disabled: false, title: 'Cortana' },
    ],
  },
  {
    key: 'camp:1',
    label: 'Ambre',
    options: [
      { value: 'x2', rosterKey: 'x2', label: 'Rival', disabled: false, title: 'Rival' },
      { value: 'bot:Fantome', rosterKey: 'bot:Fantome', label: 'Fantome', disabled: true, title: 'Aucune donnée de match pour ce joueur' },
    ],
  },
]

const player = (xuid: string): ReplayPlayer => ({ xuid, lives: [] })

afterEach(() => {
  document.body.innerHTML = ''
})

function setup({ followed = null, viewpoint = 'x1', groups = GROUPS }: {
  followed?: string | null
  viewpoint?: string | null
  groups?: readonly ViewpointOptionGroup[]
} = {}) {
  const onFollow = vi.fn()
  const onSelectViewpoint = vi.fn()
  const view = render(
    <ReplayCameraMenu
      timeline={{ viewpoint, viewpointGroups: groups, onSelectViewpoint }}
      cameraFollow={{ player: followed === null ? null : player(followed), onFollow }}
      locale="fr"
    />,
  )
  return { ...view, onFollow, onSelectViewpoint }
}

const trigger = () => screen.getByRole('button', { name: /^Caméra/ })
const open = () => fireEvent.click(trigger())
const followList = () => screen.getByRole('group', { name: 'Caméra : suivre' })
const viewpointList = () => screen.getByRole('group', { name: 'Point de vue' })
const namesOf = (list: HTMLElement) => within(list).getAllByRole('button').map((b) => b.textContent)

describe('ReplayCameraMenu — le bouton', () => {
  it('icône seule, nommée « Caméra » sans joueur suivi, panneau fermé', () => {
    setup()
    expect(trigger()).toHaveAttribute('aria-label', 'Caméra')
    expect(trigger()).toHaveAttribute('title', 'Caméra')
    expect(trigger()).toHaveAttribute('aria-expanded', 'false')
    expect(trigger().textContent).toBe('')
    expect(screen.queryByRole('group', { name: 'Caméra' })).toBeNull()
  })

  it('un joueur suivi : « Caméra : suivre <nom> » en nom accessible et en infobulle, bot compris (nom sans suffixe)', () => {
    setup({ followed: 'x2' })
    expect(trigger()).toHaveAttribute('aria-label', 'Caméra : suivre Rival')
    expect(trigger()).toHaveAttribute('title', 'Caméra : suivre Rival')
  })

  it('un bot suivi se retrouve par sa clé de roster', () => {
    setup({ followed: 'bot:Cortana' })
    expect(trigger()).toHaveAttribute('aria-label', 'Caméra : suivre Cortana')
  })

  it('sans aucun joueur : ni bouton ni liste', () => {
    const { container } = setup({ groups: [] })
    expect(container.firstChild).toBeNull()
  })
})

describe('ReplayCameraMenu — les deux listes', () => {
  it('« Suivre » : Aucun, puis les joueurs de chaque camp, bots compris ; « Point de vue » : les mêmes joueurs', () => {
    setup()
    open()
    expect(trigger()).toHaveAttribute('aria-expanded', 'true')
    expect(namesOf(followList())).toEqual(['Aucun', 'JGtm', 'Cortana', 'Rival', 'Fantome'])
    expect(namesOf(viewpointList())).toEqual(['JGtm', 'Cortana', 'Rival', 'Fantome'])
    expect(within(followList()).getByText('Cobalt')).toBeInTheDocument()
    expect(within(viewpointList()).getByText('Ambre')).toBeInTheDocument()
  })

  it('sans joueur suivi, « Aucun » est coché ; le joueur suivi est coché, et lui seul', () => {
    setup()
    open()
    expect(within(followList()).getByRole('button', { name: 'Aucun' })).toHaveAttribute('aria-pressed', 'true')
    expect(within(followList()).getAllByRole('button', { pressed: true })).toHaveLength(1)
  })

  it('le joueur suivi est coché dans « Suivre », « Aucun » ne l’est plus', () => {
    setup({ followed: 'bot:Cortana' })
    open()
    expect(within(followList()).getByRole('button', { name: 'Cortana' })).toHaveAttribute('aria-pressed', 'true')
    expect(within(followList()).getByRole('button', { name: 'Aucun' })).toHaveAttribute('aria-pressed', 'false')
    expect(within(followList()).getAllByRole('button', { pressed: true })).toHaveLength(1)
  })

  it('le point de vue courant est coché dans « Point de vue », sans toucher à « Suivre »', () => {
    setup({ viewpoint: 'bid(3.0)', followed: 'x2' })
    open()
    const checked = within(viewpointList()).getAllByRole('button', { pressed: true })
    expect(checked.map((b) => b.textContent)).toEqual(['Cortana'])
    expect(within(followList()).getByRole('button', { name: 'Rival' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('l’option inerte du point de vue le reste (infobulle = la raison), mais le joueur se suit', () => {
    const { onFollow, onSelectViewpoint } = setup()
    open()
    const inerte = within(viewpointList()).getByRole('button', { name: 'Fantome' })
    expect(inerte).toBeDisabled()
    expect(inerte).toHaveAttribute('title', 'Aucune donnée de match pour ce joueur')
    fireEvent.click(inerte)
    expect(onSelectViewpoint).not.toHaveBeenCalled()
    fireEvent.click(within(followList()).getByRole('button', { name: 'Fantome' }))
    expect(onFollow).toHaveBeenCalledWith('bot:Fantome')
  })
})

describe('ReplayCameraMenu — choisir', () => {
  it('un joueur de « Suivre » appelle le geste de la fiche avec sa clé de ROSTER, et rien d’autre', () => {
    const { onFollow, onSelectViewpoint } = setup()
    open()
    fireEvent.click(within(followList()).getByRole('button', { name: 'Cortana' }))
    expect(onFollow).toHaveBeenCalledTimes(1)
    expect(onFollow).toHaveBeenCalledWith('bot:Cortana')
    expect(onSelectViewpoint).not.toHaveBeenCalled()
  })

  it('« Aucun » ne suit personne', () => {
    const { onFollow } = setup({ followed: 'x2' })
    open()
    fireEvent.click(within(followList()).getByRole('button', { name: 'Aucun' }))
    expect(onFollow).toHaveBeenCalledWith(null)
  })

  it('un point de vue appelle `select` avec la valeur de l’option (la ligne de feuille d’un bot), sans suivre personne', () => {
    const { onFollow, onSelectViewpoint } = setup()
    open()
    fireEvent.click(within(viewpointList()).getByRole('button', { name: 'Cortana' }))
    expect(onSelectViewpoint).toHaveBeenCalledTimes(1)
    expect(onSelectViewpoint).toHaveBeenCalledWith('bid(3.0)')
    expect(onFollow).not.toHaveBeenCalled()
  })

  it('un choix ferme le panneau et rend le focus au bouton', () => {
    setup()
    open()
    fireEvent.click(within(followList()).getByRole('button', { name: 'Rival' }))
    expect(screen.queryByRole('group', { name: 'Caméra : suivre' })).toBeNull()
    expect(trigger()).toHaveAttribute('aria-expanded', 'false')
    expect(document.activeElement).toBe(trigger())
  })
})

describe('ReplayCameraMenu — fermer', () => {
  it('Échap ferme sans rien changer et rend le focus au bouton', () => {
    const { onFollow, onSelectViewpoint } = setup()
    open()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('group', { name: 'Caméra : suivre' })).toBeNull()
    expect(document.activeElement).toBe(trigger())
    expect(onFollow).not.toHaveBeenCalled()
    expect(onSelectViewpoint).not.toHaveBeenCalled()
  })

  it('un clic dehors ferme (au geste, pas au relâché), sans rien changer', () => {
    const { onFollow } = setup()
    open()
    fireEvent.pointerDown(document.body)
    expect(screen.queryByRole('group', { name: 'Caméra : suivre' })).toBeNull()
    expect(onFollow).not.toHaveBeenCalled()
  })

  it('un clic DANS le panneau ne le ferme pas', () => {
    setup()
    open()
    fireEvent.pointerDown(within(followList()).getByText('Cobalt'))
    expect(screen.getByRole('group', { name: 'Caméra : suivre' })).toBeInTheDocument()
  })

  it('le panneau se signale au mode plein écran : Échap le ferme AVANT de sortir du Plein cadre', () => {
    const { result } = renderHook(() => {
      const frame = useReplayFrame('match-camera')
      return { frame, fullscreen: useReplayFullscreen(frame) }
    })
    setup()
    act(() => result.current.frame.toggleFullFrame())
    open()
    expect(screen.getByRole('group', { name: 'Caméra' })).toHaveAttribute(REPLAY_ESCAPE_LAYER_ATTR)

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    })
    expect(screen.queryByRole('group', { name: 'Caméra' })).toBeNull()
    expect(result.current.frame.frame).toBe('plein-cadre')

    act(() => {
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    })
    expect(result.current.frame.frame).toBe('standard')
  })
})

describe('ReplayCameraMenu — le clavier du lecteur reste actif', () => {
  /** Le clavier du lecteur, câblé comme la toile le câble, à côté du menu. */
  function Keyboard({ togglePlay, seekBy }: { togglePlay: () => void; seekBy: (s: number) => void }) {
    useReplayShortcuts({
      togglePlay, seekBy, stepFrames: vi.fn(), restart: vi.fn(), toggleSound: vi.fn(),
      toggleFullFrame: vi.fn(), toggleCinema: vi.fn(), skipSeconds: 10, enabled: true,
    })
    return null
  }

  function withKeyboard() {
    const togglePlay = vi.fn()
    const seekBy = vi.fn()
    const onFollow = vi.fn()
    render(
      <>
        <Keyboard togglePlay={togglePlay} seekBy={seekBy} />
        <ReplayCameraMenu
          timeline={{ viewpoint: 'x1', viewpointGroups: GROUPS, onSelectViewpoint: vi.fn() }}
          cameraFollow={{ player: null, onFollow }}
          locale="fr"
        />
      </>,
    )
    return { togglePlay, seekBy }
  }

  it('Espace lance ou arrête la lecture après un choix : le focus est sur le bouton, pas sur une entrée disparue', () => {
    const { togglePlay } = withKeyboard()
    open()
    fireEvent.click(within(followList()).getByRole('button', { name: 'Rival' }))
    expect(document.activeElement).toBe(trigger())
    fireEvent.keyDown(document.activeElement as Element, { key: ' ' })
    expect(togglePlay).toHaveBeenCalledTimes(1)
  })

  it('Espace agit aussi après Échap', () => {
    const { togglePlay } = withKeyboard()
    open()
    fireEvent.keyDown(window, { key: 'Escape' })
    fireEvent.keyDown(document.activeElement as Element, { key: ' ' })
    expect(togglePlay).toHaveBeenCalledTimes(1)
  })

  it('panneau ouvert, une touche utilisée dans ses listes ne traverse pas vers le clavier du lecteur', () => {
    const { togglePlay, seekBy } = withKeyboard()
    open()
    const entry = within(followList()).getByRole('button', { name: 'Rival' })
    fireEvent.keyDown(entry, { key: 'ArrowRight' })
    fireEvent.keyDown(entry, { key: ' ' })
    expect(seekBy).not.toHaveBeenCalled()
    expect(togglePlay).not.toHaveBeenCalled()
  })
})
