/**
 * Tests — ReplayTimelineTracks (la frise habillée et ses quatre pistes).
 *
 * Ce qu'ils protègent :
 *  1. LES QUATRE PISTES SONT NOMMÉES. Une bande de trois pixels sans étiquette ne se lit pas.
 *  2. LA PISTE MÉDIAS RESTE, MÊME VIDE (demande utilisateur du 2026-08-28) — et son vide est
 *     une phrase, pas une bande grise muette. C'est l'exception assumée à la règle « pas de
 *     commande quand il n'y a rien à commander » : ce n'est pas une commande, c'est un lieu.
 *  3. OUVRIR UN MÉDIA MET LE REJEU EN PAUSE. Le composant ne connaît pas la boucle : il DEMANDE
 *     la pause à l'appelant, et seulement si la lecture tourne.
 *  4. AUCUN HEX. Les encres passent par les tokens du thème (règle color-tokens).
 */
import { describe, expect, it, vi } from 'vitest'
import { createRef } from 'react'
import { fireEvent, render, screen } from '@testing-library/react'

import { ReplayTimelineTracks } from './ReplayTimelineTracks'
import type { PlacedMedia, ReplayMediaItem, TrackMark } from './replayTimelineTracksLogic'
import { TIMELINE_SHORTCUT_ATTR } from './useReplayShortcuts'

function mark(over: Partial<TrackMark> = {}): TrackMark {
  return { key: 'm1', ratio: 0.5, kind: 'kill', clock: '2:30', ...over }
}

function mediaItem(over: Partial<ReplayMediaItem> = {}): ReplayMediaItem {
  return {
    id: 'media-1',
    kind: 'image',
    replayMs: 30_000,
    thumbUrl: '/thumb.png',
    url: '/full.png',
    label: 'Capture Streets',
    ...over,
  }
}

function placed(over: Partial<PlacedMedia> = {}): PlacedMedia {
  return { item: mediaItem(), from: 0.5, to: 0.5, ...over }
}

function renderTracks(over: Partial<Parameters<typeof ReplayTimelineTracks>[0]> = {}) {
  const onRequestPause = vi.fn()
  const onScrub = vi.fn()
  const utils = render(
    <ReplayTimelineTracks
      sliderRef={createRef<HTMLInputElement>()}
      minFrame={0}
      maxFrame={600}
      onScrub={onScrub}
      own={[]}
      allies={[]}
      dominance={[]}
      allyOf={() => null}
      labelOf={(id) => `Équipe ${id}`}
      media={[]}
      showMediaTrack
      playing
      onRequestPause={onRequestPause}
      startClock="0:00"
      midClock="2:30"
      endClock="5:00"
      locale="fr"
      {...over}
    />,
  )
  return { ...utils, onRequestPause, onScrub }
}

describe('ReplayTimelineTracks — les quatre pistes sont nommées', () => {
  it('porte les étiquettes Toi, Alliés, Dominance et Médias', () => {
    renderTracks()
    for (const label of ['Toi', 'Alliés', 'Dominance', 'Médias']) {
      expect(screen.getByText(label)).toBeTruthy()
    }
  })

  it('les trois repères de l’axe datent le match, pas le film', () => {
    renderTracks({ startClock: '0:00', midClock: '4:07', endClock: '8:15' })
    expect(screen.getByText('0:00')).toBeTruthy()
    expect(screen.getByText('4:07')).toBeTruthy()
    expect(screen.getByText('8:15')).toBeTruthy()
  })

  it('le curseur garde ses bornes et son nom accessible', () => {
    renderTracks({ minFrame: 149, maxFrame: 4_929 })
    const frise = screen.getByLabelText('Temps de match') as HTMLInputElement
    expect(frise.tagName).toBe('INPUT')
    expect(frise).toHaveAttribute('min', '149')
    expect(frise).toHaveAttribute('max', '4929')
  })

  /**
   * GARDE-FOU DU LIEN COMPOSANT <-> CLAVIER (décision utilisateur du 2026-08-28).
   *
   * L'exemption de la garde anti-frappe est NOMINATIVE : `useReplayShortcuts` cherche cet
   * attribut, ce composant le pose. Rien dans le typage ne relie les deux — retirer l'attribut
   * du champ compilerait, passerait tous les autres tests, et rendrait muets Espace et les
   * flèches dès le premier clic sur la frise. C'est exactement ce qu'un garde-rail attrape, et
   * la constante est importée du hook pour que le renommer casse ici aussi.
   */
  it('la frise PORTE l’attribut qui lui rend les raccourcis clavier', () => {
    renderTracks()
    expect(screen.getByLabelText('Temps de match')).toHaveAttribute(TIMELINE_SHORTCUT_ATTR)
  })
})

describe('ReplayTimelineTracks — les marques et la dominance', () => {
  it('pose chaque marque avec son horloge en infobulle', () => {
    const { container } = renderTracks({
      own: [mark({ key: 'k1', clock: '1:12' }), mark({ key: 'd1', kind: 'death', clock: '3:40' })],
      allies: [mark({ key: 'a1', clock: '2:02' })],
    })
    expect(container.querySelectorAll('[title="1:12"]')).toHaveLength(1)
    expect(container.querySelectorAll('[title="3:40"]')).toHaveLength(1)
    expect(container.querySelectorAll('[title="2:02"]')).toHaveLength(1)
  })

  it('nomme le meneur d’une bande de dominance, avec le libellé du scoreboard', () => {
    renderTracks({
      dominance: [{ key: 's1', from: 0, to: 0.6, teamId: 1 }],
      allyOf: () => true,
      labelOf: () => 'Cobalt',
    })
    expect(screen.getByTitle('Cobalt mène')).toBeTruthy()
  })

  it('N’ÉCRIT AUCUN HEX : toutes les encres passent par les tokens du thème', () => {
    const { container } = renderTracks({
      own: [mark()],
      allies: [mark({ key: 'a1' })],
      dominance: [{ key: 's1', from: 0, to: 1, teamId: 0 }],
    })
    expect(container.innerHTML).not.toMatch(/#[0-9a-fA-F]{6}/)
  })
})

describe('ReplayTimelineTracks — la piste médias', () => {
  it('reste affichée MÊME VIDE, et le dit en toutes lettres', () => {
    renderTracks({ media: [] })
    expect(screen.getByText('Aucun média sur ce match')).toBeTruthy()
  })

  it('un média porte son libellé et devient un bouton', () => {
    renderTracks({ media: [placed()] })
    expect(screen.getByRole('button', { name: 'Capture Streets' })).toBeTruthy()
    expect(screen.queryByText('Aucun média sur ce match')).toBeNull()
  })

  it('OUVRIR MET LE REJEU EN PAUSE quand il tourne, et montre le média en grand', () => {
    const { onRequestPause } = renderTracks({ media: [placed()], playing: true })
    fireEvent.click(screen.getByRole('button', { name: 'Capture Streets' }))
    expect(onRequestPause).toHaveBeenCalledTimes(1)
    expect(screen.getByRole('dialog', { name: 'Capture Streets' })).toBeTruthy()
    expect(screen.getByText('Rejeu en pause')).toBeTruthy()
  })

  it('déjà en pause : rien à demander, la lightbox s’ouvre quand même', () => {
    const { onRequestPause } = renderTracks({ media: [placed()], playing: false })
    fireEvent.click(screen.getByRole('button', { name: 'Capture Streets' }))
    expect(onRequestPause).not.toHaveBeenCalled()
    expect(screen.getByRole('dialog', { name: 'Capture Streets' })).toBeTruthy()
  })

  it('Échap referme la lightbox', () => {
    renderTracks({ media: [placed()] })
    fireEvent.click(screen.getByRole('button', { name: 'Capture Streets' }))
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  /**
   * VIDE N'EST PAS ABSENTE. Les deux états se ressemblent à l'écran et ne disent pas la même
   * chose : « aucun média sur ce match » est un fait du match, tandis qu'un titre sans médias
   * n'a rien à dire du tout — la rangée y mentirait. Le rejeu n'étant gardé que par
   * `matchmaking`, rien d'autre que cette prop ne retire la piste.
   */
  it('LA RANGÉE DISPARAÎT quand le titre ne porte pas les médias (ni piste, ni phrase de vide)', () => {
    renderTracks({ media: [], showMediaTrack: false })
    expect(screen.queryByText('Médias')).toBeNull()
    expect(screen.queryByText('Aucun média sur ce match')).toBeNull()
    // Les trois autres pistes et le curseur restent intacts.
    for (const label of ['Toi', 'Alliés', 'Dominance']) {
      expect(screen.getByText(label)).toBeTruthy()
    }
    expect(screen.getByLabelText('Temps de match')).toBeTruthy()
  })
})
