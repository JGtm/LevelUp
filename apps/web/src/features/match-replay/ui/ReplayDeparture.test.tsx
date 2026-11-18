/**
 * ReplayDeparture.test.tsx — LA FICHE D'UN JOUEUR PARTI.
 *
 * Un joueur dont le corps est supprimé sans lâcher ni compte à rebours, puis dont la présence se ferme, a
 * QUITTÉ la partie : sa fiche, sa tuile compacte et sa ligne le disent (« a quitté la partie »), sans
 * « Éliminé », sans « Hors film », sans compte à rebours. Un joueur mort (compte à rebours lu) garde son état.
 * Alpha vit sur [0, 100] ; sa présence se ferme à l'image 200, dans un document de 300 images de 100 ms.
 */
import { useMemo } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'
import { buildPlayers, vitalityPresence } from '@/lib/replay/rosterLogic'

import { ReplayCompactCard, type CompactForm } from './ReplayCompactCard'
import { ReplayPlayerNow } from './ReplayPlayerNow'
import { EliminatedBox } from './ReplayVitality'
import { useCardScene } from '../hooks/useCardScene'
import { REPLAY_TEXT } from '../i18n/i18n'
import { cardChrome, SEAT_WAITING_CHROME } from '../model/playerCardFx'
import { playerNowReadings } from '../model/playerNowReadings'
import { respawnText } from '../model/rosterRowReadings'
import { testReplayDoc } from '../test/testDoc'

const T = REPLAY_TEXT.fr
const FORMES: CompactForm[] = ['stacked', 'line']

/** Alpha : une vie [0, 100] finie par `fin` ; présence jusqu'à 200 ; un bot présent jusqu'au bout. */
function document1(fin: 'deleted' | 'released', countdown = false, retour = false) {
  return testReplayDoc({
    frameCount: 300,
    frameIntervalMs: 100,
    originMs: 0,
    roster: [
      { xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', team: 0, presence: [{ from: 0, to: 200 }] },
      { xuid: '', bot: true, filmIndex: 1, seat: 1, name: 'Marc', team: 1, presence: [{ from: 0, to: 299 }] },
    ],
    tracks: [
      { slot: 512, team: 0, xuid: 'A', startFrame: 0, endFrame: 100, endReason: fin, releaseFrame: fin === 'released' ? 100 : undefined, points: [{ t: 0, x: 0, y: 0, sh: 1, hp: 1 }] },
      ...(retour ? [{ slot: 513, team: 0, xuid: 'A', startFrame: 180, endFrame: 290, points: [{ t: 180, x: 0, y: 0, sh: 1, hp: 1 }] }] : []),
      { slot: 600, team: 1, xuid: '', bot: 'Marc', startFrame: 0, endFrame: 290, points: [{ t: 0, x: 1, y: 1, sh: 1, hp: 1 }] },
    ] as never,
    respawnCountdowns: countdown ? [{ filmIndex: 0, from: 100, to: 180, seconds: 8, startRead: true, endRead: true }] : [],
  })
}

function Fiche({ doc, frame, form }: { doc: ReturnType<typeof document1>; frame: number; form: CompactForm }) {
  const players = useMemo(() => buildPlayers(doc, []), [doc])
  const scene = useCardScene(doc, players)
  const player = players.find((p) => p.xuid === 'A')!
  return <ReplayCompactCard doc={doc} player={player} scene={scene} frame={frame} locale="fr" form={form} ally={NO_ALLEGIANCE.ofTeam(0)} onCard={vi.fn()} />
}

function lecture(doc: ReturnType<typeof document1>, frame: number) {
  const player = buildPlayers(doc, []).find((p) => p.xuid === 'A')!
  return playerNowReadings({ player, doc, frame, presence: vitalityPresence(doc), locale: 'fr' })
}

describe.each(FORMES)('la tuile compacte d’un joueur parti — forme %s', (form) => {
  it('dit « a quitté la partie » à la place du compte à rebours, dans une habit de place sans joueur', () => {
    render(<Fiche doc={document1('deleted')} frame={150} form={form} />)
    const f = screen.getByTitle('Alpha').closest('[data-replay-player-card]') as HTMLElement
    const mot = within(f).getByText(T.presenceLeft)
    expect(mot.getAttribute('title')).toBe(T.presenceLeft)
    expect(within(f).queryByText(T.goneLabel)).toBeNull()
    expect(within(f).queryByText(/^Réapp\./)).toBeNull()
    expect(f.style.borderStyle).toBe('dashed')
  })

  it('avant le départ de sa vie, et après la fermeture de sa présence : aucune mention de départ', () => {
    render(<Fiche doc={document1('deleted')} frame={50} form={form} />)
    expect(screen.queryByText(T.presenceLeft)).toBeNull()
  })

  it('un joueur mort (lâcher de commande, compte à rebours lu) garde « Réapp. » ou « Hors film » : jamais « a quitté »', () => {
    render(<Fiche doc={document1('released', true)} frame={150} form={form} />)
    expect(screen.queryByText(T.presenceLeft)).toBeNull()
    expect(screen.getByText(T.goneLabel)).toBeTruthy()
  })
})

describe('l’encadré et le bloc « À l’instant » d’un joueur parti', () => {
  const doc = document1('deleted')

  it('le bloc « À l’instant » écrit « a quitté la partie » sans « Éliminé » ni « ne revient plus »', () => {
    const player = buildPlayers(doc, []).find((p) => p.xuid === 'A')!
    const ages = { presence: vitalityPresence(doc), vitalityFade: 60, readingFull: 200 }
    render(<ReplayPlayerNow player={player} doc={doc} frame={150} clock="0:15" ages={ages} locale="fr" />)
    expect(screen.getByText(T.presenceLeft)).toBeTruthy()
    expect(screen.queryByText(T.eliminatedLabel)).toBeNull()
    expect(screen.queryByText(T.goneValue)).toBeNull()
  })

  it('les deux hauteurs d’encadré (35 et 31) disent le départ ; le mort garde « Éliminé » avec son compte', () => {
    const parti = lecture(doc, 150).state
    for (const bodyPx of [35, 31] as const) {
      const vue = render(<EliminatedBox state={parti} doc={doc} frame={150} locale="fr" bodyPx={bodyPx} />)
      expect(screen.getByText(T.presenceLeft)).toBeTruthy()
      expect(screen.queryByText(T.eliminatedLabel)).toBeNull()
      expect(screen.queryByText(T.goneLabel)).toBeNull()
      vue.unmount()
    }
    const mortDoc = document1('released', true, true)
    const mort = lecture(mortDoc, 150).state
    render(<EliminatedBox state={mort} doc={mortDoc} frame={150} locale="fr" bodyPx={35} />)
    expect(screen.getByText(T.eliminatedLabel)).toBeTruthy()
    expect(screen.getByTitle(T.respawnIn)).toBeTruthy()
    expect(screen.queryByText(T.presenceLeft)).toBeNull()
  })

  it('la ligne de la colonne dit « a quitté la partie » ; l’habit de la tuile n’est plus celui de la mort', () => {
    const parti = lecture(doc, 150).state
    expect(parti.departed).toBe(true)
    expect(respawnText(parti, doc, 150, T)).toBe(T.presenceLeft)
    expect(cardChrome(parti.alive, parti.departed)).toEqual(SEAT_WAITING_CHROME)
    expect(cardChrome(false, false)).not.toEqual(SEAT_WAITING_CHROME)
  })
})
