/**
 * Tests — ReplayTimelineCompact (la frise de la barre d'une ligne).
 *
 * Ce qu'ils protègent : la frise compacte lit le MÊME objet `timeline` que la frise à pistes (une ligne ajoutée
 * au fil apparaît sur les deux, et la compacte porte en plus les frags de TOUS les joueurs, à la couleur du camp) ;
 * son champ est celui du geste — `data-replay-timeline`, `onScrub`, `sliderRef` ; la bande de dominance, les repères
 * de médias et leur ouverture ; la géométrie des repères est celle des marques.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { createRef } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { TIMELINE_SHORTCUT_ATTR } from '../hooks/useReplayShortcuts'
import { useReplayTimeline } from '../hooks/useReplayTimeline'
import type { ReplayDeath, ReplayFeedEntry, ReplayKill } from '../model/killFeedLogic'
import type { PlacedMedia } from '../model/replayTimelineTracksLogic'
import type { ReplayWindowBounds } from '../model/replayWindow'
import { testReplayDoc } from '../test/testDoc'
import { makeTimeline } from '../test/transportStubs'
import { ReplayTimelineCompact } from './ReplayTimelineCompact'
import { ReplayTimelineTracks } from './ReplayTimelineTracks'

const ALLY = tokenCssVar('team-ally')
const ENEMY = tokenCssVar('team-enemy')

function fragMarks(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>('[data-replay-frag-mark]'))
}

describe('ReplayTimelineCompact — le champ du geste', () => {
  it('c’est le champ de la frise à pistes : data-replay-timeline, nom « Temps de match », onScrub, sliderRef', () => {
    const sliderRef = createRef<HTMLInputElement>()
    const seen: string[] = []
    const onScrub = vi.fn((e: { currentTarget: HTMLInputElement }) => {
      seen.push(e.currentTarget.value)
    })
    render(<ReplayTimelineCompact timeline={makeTimeline({ sliderRef, onScrub: onScrub as never, minFrame: 10, maxFrame: 400 })} />)
    const field = screen.getByRole('slider', { name: 'Temps de match' })
    expect(field).toHaveAttribute(TIMELINE_SHORTCUT_ATTR)
    expect(field).toHaveAttribute('min', '10')
    expect(field).toHaveAttribute('max', '400')
    expect(sliderRef.current).toBe(field)
    // Aller à un instant : le geste du champ (clic ou glisser) émet un changement, que `onScrub` reçoit.
    fireEvent.change(field, { target: { value: '250' } })
    expect(onScrub).toHaveBeenCalledTimes(1)
    expect(seen).toEqual(['250'])
  })

  it('la racine porte data-replay-cursor-host : c’est là que la boucle de dessin écrit la progression', () => {
    const { container } = render(<ReplayTimelineCompact timeline={makeTimeline()} />)
    expect(container.querySelector('[data-replay-cursor-host]')).not.toBeNull()
    expect(container.querySelector('[data-replay-cursor-host]')?.contains(screen.getByRole('slider'))).toBe(true)
  })

  it('la progression du rail lit la variable de la boucle de dessin, dans la géométrie des marques', () => {
    const { container } = render(<ReplayTimelineCompact timeline={makeTimeline()} />)
    const fill = container.querySelector<HTMLElement>('i.bg-foreground')
    expect(fill?.style.width).toContain('--played-r')
  })
})

describe('ReplayTimelineCompact — repères et dominance', () => {
  it('un repère par frag, à la couleur du camp du tueur : allié, adverse, ou neutre si le camp est inconnu', () => {
    const { container } = render(
      <ReplayTimelineCompact
        timeline={makeTimeline({
          fragMarks: [
            { key: 'a', ratio: 0.1, teamId: 0 },
            { key: 'b', ratio: 0.5, teamId: 1 },
            { key: 'c', ratio: 0.9, teamId: 7 },
          ],
          allyOf: (id) => (id === 0 ? true : id === 1 ? false : null),
        })}
      />,
    )
    const marks = fragMarks(container)
    expect(marks).toHaveLength(3)
    expect(marks[0].getAttribute('style')).toContain(ALLY)
    expect(marks[1].getAttribute('style')).toContain(ENEMY)
    expect(marks[2].getAttribute('style')).toContain('var(--border)')
    // Même géométrie que les marques de la frise à pistes : une position calculée sur la course du curseur.
    expect(marks[1].getAttribute('style')).toContain('calc(')
  })

  it('la bande de dominance : un segment par plage, égalité à l’encre d’égalité', () => {
    const { container } = render(
      <ReplayTimelineCompact
        timeline={makeTimeline({
          dominance: [
            { key: 's0', from: 0, to: 0.2, teamId: null },
            { key: 's1', from: 0.2, to: 1, teamId: 0 },
          ],
          allyOf: () => true,
        })}
      />,
    )
    const band = Array.from(container.querySelectorAll<HTMLElement>('i.top-0.block.h-full'))
    expect(band).toHaveLength(2)
    expect(band[0].getAttribute('style')).toContain(tokenCssVar('outcome-draw'))
    expect(band[1].getAttribute('style')).toContain(ALLY)
  })

  it('sans repère ni dominance, rien n’est dessiné au-dessus ou dessous du rail', () => {
    const { container } = render(<ReplayTimelineCompact timeline={makeTimeline()} />)
    expect(fragMarks(container)).toHaveLength(0)
    expect(container.querySelectorAll('i.top-0.block.h-full')).toHaveLength(0)
  })
})

describe('ReplayTimelineCompact — les repères de médias', () => {
  const media: PlacedMedia = {
    item: { id: 'media-1', kind: 'image', replayMs: 30_000, thumbUrl: '/t.png', url: '/f.png', label: 'Capture Streets' },
    from: 0.5,
    to: 0.5,
  }

  it('un média est un bouton nommé ; l’ouvrir met le rejeu en pause quand il tourne, et montre le média', () => {
    const calls: string[] = []
    render(<ReplayTimelineCompact timeline={makeTimeline({ media: [media], playing: true, onRequestPause: () => calls.push('pause') })} />)
    fireEvent.click(screen.getByRole('button', { name: 'Capture Streets' }))
    expect(calls).toEqual(['pause'])
    expect(screen.getByRole('dialog', { name: 'Capture Streets' })).toBeTruthy()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('déjà en pause : aucune demande de pause', () => {
    const calls: string[] = []
    render(<ReplayTimelineCompact timeline={makeTimeline({ media: [media], playing: false, onRequestPause: () => calls.push('pause') })} />)
    fireEvent.click(screen.getByRole('button', { name: 'Capture Streets' }))
    expect(calls).toEqual([])
  })

  it('un titre sans médias (capability absente) : aucun repère', () => {
    render(<ReplayTimelineCompact timeline={makeTimeline({ media: [media], showMediaTrack: false })} />)
    expect(screen.queryByRole('button', { name: 'Capture Streets' })).toBeNull()
  })
})

/** Document 10 Hz, 200 images : la frise couvre les images 0 à 199 ; un ratio r vaut 199·r. */
const DOC = testReplayDoc({ frameCount: 200, frameIntervalMs: 100, originMs: 0 })
const WINDOW: ReplayWindowBounds = { startFrame: 0, leadInFrame: 0, endFrame: 200, startMs: 0, endMs: 20_000 }
const PLAYERS = [
  { xuid: 'moi', filmName: 'Moi', team: 0, lives: [], board: { xuid: 'moi', team_side: 't0' } },
  { xuid: 'pote', filmName: 'Pote', team: 0, lives: [], board: { xuid: 'pote', team_side: 't0' } },
  { xuid: 'eux', filmName: 'Eux', team: 1, lives: [], board: { xuid: 'eux', team_side: 't1' } },
] as unknown as ReplayPlayer[]
const PLAYBACK = {
  sliderRef: { current: null },
  startFrame: 0,
  endFrame: 200,
  onScrub: () => {},
  playing: false,
  togglePlay: () => {},
  restart: () => {},
  seekBy: () => {},
  stepFrames: () => {},
  seekToFrame: () => {},
}

function killEntry(key: string, replayMs: number, xuid: string, teamID: number): ReplayFeedEntry {
  return { key, replayMs, kill: { xuid, victimXuid: '', medals: [], teamID } as unknown as ReplayKill, medal: null, death: null }
}

/** Les deux frises, alimentées par le MÊME objet que rend `useReplayTimeline` pour le fil donné. */
function BothFrises({ feed }: { feed: readonly ReplayFeedEntry[] }) {
  const timeline = useReplayTimeline({
    doc: DOC,
    playWindow: WINDOW,
    feedEntries: feed,
    marks: new Map(),
    viewpoint: 'moi',
    allegiance: buildFilmAllegiance(PLAYERS, 'moi'),
    players: PLAYERS,
    onSelectViewpoint: () => {},
    lead: { allyOf: (id: number) => id === 0, labelOf: (id: number) => `Équipe ${id}` },
    playback: PLAYBACK,
    toggleSound: () => {},
    toggleFullFrame: () => {},
    toggleCinema: () => {},
    renderWidth: 480,
    scoreTimelineKind: '',
    locale: 'fr',
  })
  return (
    <>
      <div data-testid="tracks">
        <ReplayTimelineTracks {...timeline} clockRef={createRef<HTMLSpanElement>()} />
      </div>
      <div data-testid="compact">
        <ReplayTimelineCompact timeline={timeline} />
      </div>
    </>
  )
}

describe('ReplayTimelineCompact — le même fil que la frise à pistes', () => {
  /** Les marques de la piste du joueur regardé : les spans portant l'instant en infobulle. */
  function tracksMarks(): HTMLElement[] {
    return Array.from(screen.getByTestId('tracks').querySelectorAll<HTMLElement>('span[title]')).filter((el) => /^\d+:\d{2}$/.test(el.title))
  }

  it('un repère ajouté au fil apparaît sur les deux frises', () => {
    const feed = [killEntry('k1', 5_000, 'moi', 0)]
    const { rerender } = render(<BothFrises feed={feed} />)
    expect(tracksMarks()).toHaveLength(1)
    expect(fragMarks(screen.getByTestId('compact'))).toHaveLength(1)

    rerender(<BothFrises feed={[...feed, killEntry('k2', 12_000, 'moi', 0)]} />)
    expect(tracksMarks()).toHaveLength(2)
    expect(fragMarks(screen.getByTestId('compact'))).toHaveLength(2)
  })

  it('la frise compacte porte aussi les frags de TOUS les joueurs, que la frise à pistes ne montre pas', () => {
    // Le frag d'un adversaire n'est ni sur la piste du joueur regardé ni sur celle de ses coéquipiers.
    const feed = [killEntry('k1', 5_000, 'moi', 0), killEntry('k2', 9_000, 'eux', 1), killEntry('k3', 14_000, 'pote', 0)]
    render(<BothFrises feed={feed} />)
    expect(tracksMarks()).toHaveLength(2)
    const marks = fragMarks(screen.getByTestId('compact'))
    expect(marks).toHaveLength(3)
    expect(marks.map((m) => (m.getAttribute('style') ?? '').includes(ENEMY))).toEqual([false, true, false])
  })

  it('une mort neutre n’est pas un frag : aucun repère', () => {
    const death: ReplayFeedEntry = { key: 'd1', replayMs: 4_000, kill: null, medal: null, death: { xuid: 'moi' } as ReplayDeath }
    render(<BothFrises feed={[death]} />)
    expect(fragMarks(screen.getByTestId('compact'))).toHaveLength(0)
  })

  it('un frag hors de la fenêtre de gameplay n’est pas posé', () => {
    render(<BothFrises feed={[killEntry('k1', 60_000, 'moi', 0)]} />)
    expect(fragMarks(screen.getByTestId('compact'))).toHaveLength(0)
  })
})
