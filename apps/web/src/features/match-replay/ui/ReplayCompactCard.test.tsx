/**
 * Tests — la fiche compacte de la Grande équipe (`ReplayCompactCard`), dans ses deux formes.
 *
 * Ce qu'ils protègent : nom, pastille de camp (jamais de bordure colorée), deux barres sans F / M / A ni
 * arme ni grenades ni équipement ; l'éliminé qui écrit « Réapp. » à la place des jauges ; le joueur suivi
 * (bordure de suivi, nom en gras) ; le porteur d'objectif et le bot ; la place en attente ; le clic.
 */
import { useMemo } from 'react'
import type { ComponentProps } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { ReplayCompactCard, ReplayCompactWaitingCard, type CompactForm } from './ReplayCompactCard'
import { useCardScene } from '../hooks/useCardScene'
import { GABARIT_FICHE_COMPACTE as G } from '../model/cardGabarit'
import { REPLAY_TEXT } from '../i18n/i18n'
import { testReplayDoc } from '../test/testDoc'

const T = REPLAY_TEXT.fr

interface Vie { startFrame: number; endFrame: number; slot?: number; sh?: number; hp?: number }

/** Un document à deux joueurs : Alpha (la vie donnée) et Marc, un bot. */
function document2(alpha: Vie[], extra: Parameters<typeof testReplayDoc>[0] = {}) {
  return testReplayDoc({
    frameCount: 300,
    frameIntervalMs: 100,
    originMs: 0,
    roster: [
      { xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', team: 0 },
      { xuid: '', bot: true, filmIndex: 1, seat: 1, name: 'Marc', team: 1 },
    ],
    tracks: [
      ...alpha.map((v, k) => ({
        slot: v.slot ?? 512 + k, team: 0, xuid: 'A', startFrame: v.startFrame, endFrame: v.endFrame,
        points: [{ t: v.startFrame, x: 0, y: 0, sh: v.sh, hp: v.hp }],
      })),
      { slot: 600, team: 1, xuid: '', bot: 'Marc', startFrame: 0, endFrame: 290, points: [{ t: 0, x: 1, y: 1, sh: 1, hp: 1 }] },
    ] as never,
    ...extra,
  })
}

type Props = Omit<ComponentProps<typeof ReplayCompactCard>, 'player' | 'scene' | 'ally' | 'doc'> & {
  doc: ReturnType<typeof testReplayDoc>
  who?: string
}

/** La fiche d'un joueur du document, montée comme la colonne : jointure, puis scène de lecture une fois. */
function Fiche({ doc, who = 'A', ...rest }: Props) {
  const players = useMemo(() => buildPlayers(doc, []), [doc])
  const scene = useCardScene(doc, players)
  const player = players.find((p) => p.xuid === who)
  if (!player) throw new Error(`joueur absent : ${who}`)
  return <ReplayCompactCard {...rest} doc={doc} player={player} scene={scene} ally={NO_ALLEGIANCE.ofTeam(0)} />
}

const FORMES: CompactForm[] = ['stacked', 'line']
const PLEIN: Vie[] = [{ startFrame: 0, endFrame: 290, sh: 1, hp: 1 }]

function fiche(name: string) {
  return screen.getByTitle(name).closest('[data-replay-player-card]') as HTMLElement
}

describe.each(FORMES)('ReplayCompactCard — forme %s', (form) => {
  const hauteur = form === 'stacked' ? G.stackedPx : G.linePx
  const rendre = (doc: ReturnType<typeof testReplayDoc>, frame = 10, extra: Partial<Props> = {}) =>
    render(<Fiche doc={doc} frame={frame} locale="fr" form={form} onCard={vi.fn()} {...extra} />)

  it('plein : le nom, la pastille de camp, deux barres pleines, à la hauteur de la forme', () => {
    rendre(document2(PLEIN))
    const f = fiche('Alpha')
    expect(f.style.height).toBe(`${hauteur}px`)
    expect(within(f).getByText('Alpha')).toBeTruthy()
    expect(f.querySelectorAll('[aria-hidden="true"].rounded-sm').length).toBeGreaterThan(0)
    expect((within(f).getByLabelText('Bouclier').firstElementChild as HTMLElement).style.width).toBe('100%')
    expect((within(f).getByLabelText('Santé').firstElementChild as HTMLElement).style.width).toBe('100%')
  })

  it('jamais de bordure colorée à gauche, ni de frags / morts / assistances, ni d’arme, de grenades ou d’équipement', () => {
    const doc = document2(PLEIN, {
      weaponLabels: { '0xAAAA': { fr: 'Fusil', en: 'Rifle' } },
      loadouts: [{ t: 0, slot: 512, w: ['0xAAAA'] }],
      grenadeLabels: [{ en: 'Frag', fr: 'Fragmentation' }],
      inventory: [{ t: 0, slot: 512, d: 0, am: [{ mag: 1, res: 2 }], g: [2] }],
    })
    rendre(doc)
    const f = fiche('Alpha')
    expect(f.style.borderLeft).toBe('')
    expect(f.style.borderLeftColor).toBe('')
    expect(f.textContent).toBe('Alpha')
    expect(within(f).queryByText('Fusil')).toBeNull()
  })

  it('bouclier entamé : la barre bleue raccourcie ; santé sans bouclier : barre de bouclier vide, santé raccourcie', () => {
    rendre(document2([{ startFrame: 0, endFrame: 290, sh: 0.46, hp: 1 }]))
    expect((screen.getByLabelText('Bouclier').firstElementChild as HTMLElement).style.width).toBe('46%')
    expect((screen.getByLabelText('Santé').firstElementChild as HTMLElement).style.width).toBe('100%')
  })

  it('santé sans bouclier', () => {
    rendre(document2([{ startFrame: 0, endFrame: 290, sh: 0, hp: 0.38 }]))
    expect((screen.getByLabelText('Bouclier').firstElementChild as HTMLElement).style.width).toBe('0%')
    expect((screen.getByLabelText('Santé').firstElementChild as HTMLElement).style.width).toBe('38%')
  })

  it('éliminé : « Réapp. » à la place des jauges, le texte long en infobulle, le nom éteint', () => {
    rendre(document2([{ startFrame: 0, endFrame: 100, sh: 1, hp: 1 }, { startFrame: 180, endFrame: 290, sh: 1, hp: 1, slot: 700 }]), 140)
    const f = fiche('Alpha')
    const court = within(f).getByText(/^Réapp\. \d/)
    expect(court.getAttribute('title')).toMatch(/^Réapparition dans \d/)
    expect(within(f).queryByLabelText('Bouclier')).toBeNull()
    expect((within(f).getByText('Alpha') as HTMLElement).style.opacity).toBe(String(G.deadOpacity))
  })

  it('suivi : bordure à la couleur de suivi, nom en gras ; un autre joueur : nom ordinaire', () => {
    rendre(document2(PLEIN), 10, { tracked: true })
    const f = fiche('Alpha')
    expect(f.style.borderColor).toBe('var(--ring)')
    expect(within(f).getByText('Alpha').className).toContain('font-bold')
  })

  it('porteur d’objectif : le glyphe à droite des jauges, sa phrase en infobulle', () => {
    const doc = document2(PLEIN, { flagCarries: [{ team: 1, spans: [{ state: 'carried', xuid: 'A', t0: 0, t1: 100, x: 0, y: 0 }] }] })
    rendre(doc)
    const marque = within(fiche('Alpha')).getByTitle(T.objectiveCarry.flag)
    expect(marque.querySelector('svg')).toBeTruthy()
  })

  it('nom long : tronqué par des points de suspension, nom entier en infobulle', () => {
    const doc = document2(PLEIN, {
      roster: [
        { xuid: 'A', filmIndex: 0, seat: 0, name: 'Sentinelle de garde Alpha', team: 0 },
        { xuid: '', bot: true, filmIndex: 1, seat: 1, name: 'Marc', team: 1 },
      ],
    })
    rendre(doc)
    const nom = within(fiche('Sentinelle de garde Alpha')).getByText('Sentinelle de garde Alpha')
    expect(nom.className).toContain('truncate')
  })

  it('un clic rend la page avec la clé du joueur ; sans onCard, la fiche est inerte', () => {
    const onCard = vi.fn()
    const vue = rendre(document2(PLEIN), 10, { onCard })
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' }))
    expect(onCard).toHaveBeenCalledWith('A')
    expect(screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' }).hasAttribute('data-replay-player-card')).toBe(true)
    vue.unmount()
    rendre(document2(PLEIN), 10, { onCard: undefined })
    expect(screen.queryAllByRole('button')).toEqual([])
  })

  it('place libre et occupant pas encore apparu : une fiche tiretée de même hauteur', () => {
    const libre = render(<ReplayCompactWaitingCard occupant={null} locale="fr" form={form} />)
    const boite = screen.getByTitle(T.seatVacantHint) as HTMLElement
    expect(boite.style.height).toBe(`${hauteur}px`)
    expect(boite.style.borderStyle).toBe('dashed')
    expect(screen.getByText(T.seatVacant)).toBeTruthy()
    libre.unmount()
    const doc = document2(PLEIN)
    const occupant = buildPlayers(doc, [])[0]
    render(<ReplayCompactWaitingCard occupant={occupant} locale="fr" form={form} />)
    expect(screen.getByText('Alpha')).toBeTruthy()
    expect(screen.getByText(T.seatNotSpawned)).toBeTruthy()
  })
})

describe('ReplayCompactCard — bot', () => {
  it('la pastille « bot » (prédicat du roster) et son infobulle, dans les deux formes', () => {
    for (const form of FORMES) {
      const vue = render(<Fiche doc={document2(PLEIN)} who="bot:Marc" frame={10} locale="fr" form={form} onCard={vi.fn()} />)
      const pastille = screen.getByText(T.roster.bot)
      expect(pastille.getAttribute('title')).toBe(T.roster.botHint)
      expect(within(screen.getByRole('button', { name: 'Ouvrir la fiche de Marc' })).getByText('Marc').className).toContain('text-muted-foreground')
      vue.unmount()
    }
  })

  it('un humain ne porte pas la pastille', () => {
    render(<Fiche doc={document2(PLEIN)} frame={10} locale="fr" form="line" onCard={vi.fn()} />)
    expect(screen.queryByText(T.roster.bot)).toBeNull()
  })
})

describe('ReplayCompactCard — anglais', () => {
  it('la forme abrégée et la pastille parlent anglais', () => {
    const doc = document2([{ startFrame: 0, endFrame: 100, sh: 1, hp: 1 }, { startFrame: 180, endFrame: 290, sh: 1, hp: 1, slot: 700 }])
    render(<Fiche doc={doc} frame={140} locale="en" form="line" onCard={vi.fn()} />)
    expect(screen.getByText(/^Resp\. \d/)).toBeTruthy()
  })
})
