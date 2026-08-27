/**
 * Tests — ReplayScoreBanner (le bandeau de score au-dessus du terrain).
 *
 * CE QU'ILS PROTÈGENT : les quatre affirmations que le bandeau fait à l'écran — le score
 * écrit est celui de l'IMAGE LUE, le camp du joueur de la page est à GAUCHE et porte SA
 * couleur, la barre dit le chemin vers la cible de victoire, et le bandeau SE TAIT quand
 * il ne sait pas.
 * Le calcul, lui, est éprouvé chez `scoreBannerLogic.test.ts` ; ici on éprouve le rendu.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { normalizeScoreTimeline, type ReplayScoreDocument } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { ReplayScoreBanner } from './ReplayScoreBanner'

/** Une série d'équipe à manche unique. */
function equipe(teamId: number, points: Array<[number, number]>) {
  const pts = points.map(([t, v]) => ({ t, v }))
  return { teamId, rounds: [{ round: 0, points: pts }], total: pts }
}

/** Le témoin Slayer : t0 mène 43, t1 suit à 30 au frame 500 ; 20/15 au frame 100. */
const SLAYER_TEAMS = [
  equipe(0, [
    [0, 0],
    [100, 20],
    [500, 43],
  ]),
  equipe(1, [
    [0, 0],
    [100, 15],
    [500, 30],
  ]),
]

/** Un document de rejeu réduit à ce que le bandeau lui demande (type structurel). */
function docOf(teams: unknown[]): ReplayScoreDocument {
  return { originMs: 0, scoreTimeline: normalizeScoreTimeline({ teams, players: [] } as never) }
}

const SB: MatchScoreboardRow[] = [
  { xuid: 'moi', team_side: 't0' },
  { xuid: 'eux', team_side: 't1' },
] as MatchScoreboardRow[]

const META = new Map([
  ['moi', { gamertag: 'Moi', ally: true }],
  ['eux', { gamertag: 'Eux', ally: false }],
])

function renderBanner(over: Partial<Parameters<typeof ReplayScoreBanner>[0]> = {}) {
  return render(
    <ReplayScoreBanner
      doc={docOf(SLAYER_TEAMS)}
      scoreboard={SB}
      xuidMeta={META}
      frame={500}
      nowMs={65_000}
      playWindow={null}
      locale="fr"
      {...over}
    />,
  )
}

describe('ReplayScoreBanner — ce qui est écrit', () => {
  it('écrit les deux scores de l\'image lue', () => {
    renderBanner()
    expect(screen.getByText('43')).toBeInTheDocument()
    expect(screen.getByText('30')).toBeInTheDocument()
  })

  it('TIQUE : à une autre image, d\'autres nombres', () => {
    renderBanner({ frame: 100 })
    expect(screen.getByText('20')).toBeInTheDocument()
    expect(screen.getByText('15')).toBeInTheDocument()
    expect(screen.queryByText('43')).not.toBeInTheDocument()
  })

  it('affiche la position de lecture au format du rejeu', () => {
    renderBanner()
    expect(screen.getByText('1:05')).toBeInTheDocument()
  })

  it('CADRÉ : l’horloge est celle du GAMEPLAY, pas celle du film (D-A2)', () => {
    // Coup d'envoi à 14,861 s de film : 65 s de film = 50,139 s de jeu.
    renderBanner({
      playWindow: { startFrame: 149, endFrame: 4_929, startMs: 14_861, endMs: 492_861 },
    })
    expect(screen.getByText('0:50')).toBeInTheDocument()
    expect(screen.queryByText('1:05')).not.toBeInTheDocument()
  })
})

describe('ReplayScoreBanner — les deux camps', () => {
  it('met le camp du joueur de la page à GAUCHE', () => {
    renderBanner()
    const bars = screen.getAllByRole('progressbar')
    expect(bars).toHaveLength(2)
    expect(bars[0]).toHaveAttribute('aria-label', 'Équipe alliée')
    expect(bars[1]).toHaveAttribute('aria-label', 'Équipe adverse')
  })

  it('inverse les côtés quand le joueur de la page est dans l\'autre camp', () => {
    renderBanner({
      xuidMeta: new Map([
        ['moi', { gamertag: 'Moi', ally: false }],
        ['eux', { gamertag: 'Eux', ally: true }],
      ]),
    })
    const bars = screen.getAllByRole('progressbar')
    // La barre alliée reste à gauche : c'est son SCORE qui change de camp.
    expect(bars[0]).toHaveAttribute('aria-label', 'Équipe alliée')
    expect(bars[0]).toHaveAttribute('aria-valuetext', '30')
    expect(bars[1]).toHaveAttribute('aria-valuetext', '43')
  })

  it('prend les tokens d\'équipe de la page, jamais une couleur en dur', () => {
    const { container } = renderBanner()
    const html = container.innerHTML
    expect(html).toContain('var(--ac-team-ally)')
    expect(html).toContain('var(--ac-team-enemy)')
    expect(html).not.toMatch(/#[0-9a-fA-F]{6}/)
  })
})

describe('ReplayScoreBanner — le remplissage', () => {
  it('remplit à la cible atteinte, et l\'autre à proportion de la cible', () => {
    renderBanner()
    const bars = screen.getAllByRole('progressbar')
    expect(bars[0]).toHaveAttribute('aria-valuenow', '100')
    expect(bars[1]).toHaveAttribute('aria-valuenow', String(Math.round((30 / 43) * 100)))
  })

  it('en cours de match, AUCUNE barre n\'est pleine tant que la cible n\'est pas atteinte', () => {
    renderBanner({ frame: 100 })
    const bars = screen.getAllByRole('progressbar')
    expect(bars[0]).toHaveAttribute('aria-valuenow', String(Math.round((20 / 43) * 100)))
    expect(bars[1]).toHaveAttribute('aria-valuenow', String(Math.round((15 / 43) * 100)))
  })

  it('peint l\'aplat à la couleur d\'équipe PLEINE, sans mélange', () => {
    const { container } = renderBanner()
    expect(container.innerHTML).not.toContain('color-mix')
  })

  it('dit la MESURE au lecteur d\'écran, pas le pourcentage relatif', () => {
    renderBanner()
    const bars = screen.getAllByRole('progressbar')
    expect(bars[0]).toHaveAttribute('aria-valuetext', '43')
    expect(bars[1]).toHaveAttribute('aria-valuetext', '30')
  })

  it('laisse les deux barres vides à 0-0', () => {
    renderBanner({ frame: 0 })
    for (const bar of screen.getAllByRole('progressbar')) {
      expect(bar).toHaveAttribute('aria-valuenow', '0')
    }
  })
})

describe('ReplayScoreBanner — la manche', () => {
  const ODDBALL = [
    {
      teamId: 0,
      rounds: [
        { round: 0, points: [{ t: 0, v: 0 }, { t: 50, v: 100 }] },
        { round: 1, points: [{ t: 100, v: 0 }, { t: 150, v: 100 }] },
      ],
      total: [{ t: 0, v: 0 }, { t: 50, v: 100 }, { t: 150, v: 200 }],
    },
    equipe(1, [
      [0, 0],
      [50, 78],
      [150, 121],
    ]),
  ]

  it('annonce la manche en cours quand le mode en a plusieurs', () => {
    renderBanner({ doc: docOf(ODDBALL), frame: 150 })
    expect(screen.getByText('Manche 2')).toBeInTheDocument()
    expect(screen.getByTitle('Manche 2 sur 2')).toBeInTheDocument()
  })

  it('affiche le TOTAL du match à côté, jamais la valeur de la manche', () => {
    renderBanner({ doc: docOf(ODDBALL), frame: 150 })
    expect(screen.getByText('200')).toBeInTheDocument()
  })

  it('se tait sur un mode à manche unique', () => {
    renderBanner()
    expect(screen.queryByText(/^Manche /)).not.toBeInTheDocument()
  })
})

describe('ReplayScoreBanner — quand il se tait', () => {
  it('FFA (aucun camp) : rien du tout, pas même un cadre vide', () => {
    const { container } = renderBanner({
      scoreboard: [
        { xuid: 'moi', team_side: null },
        { xuid: 'eux', team_side: null },
      ] as MatchScoreboardRow[],
    })
    expect(container).toBeEmptyDOMElement()
  })

  it('mode sans compteur (calque vide) : rien', () => {
    const { container } = renderBanner({ doc: docOf([]) })
    expect(container).toBeEmptyDOMElement()
  })

  it('côté allié inconnu : rien — aucune des deux couleurs n\'est un défaut', () => {
    const { container } = renderBanner({ xuidMeta: undefined })
    expect(container).toBeEmptyDOMElement()
  })

  it('horloge du film non recalée : rien (la garde de `scoreTimelineOf` est bien passée)', () => {
    const { container } = renderBanner({
      doc: {
        coverage: { originResolved: false },
        scoreTimeline: normalizeScoreTimeline({ teams: SLAYER_TEAMS, players: [] } as never),
      },
    })
    expect(container).toBeEmptyDOMElement()
  })
})

describe('ReplayScoreBanner — les deux langues', () => {
  it('nomme les camps en anglais sous la locale EN', () => {
    renderBanner({ locale: 'en' })
    const bars = screen.getAllByRole('progressbar')
    expect(bars[0]).toHaveAttribute('aria-label', 'Allied team')
    expect(bars[1]).toHaveAttribute('aria-label', 'Enemy team')
  })

  it('traduit aussi la manche', () => {
    renderBanner({
      locale: 'en',
      doc: docOf([
        {
          teamId: 0,
          rounds: [
            { round: 0, points: [{ t: 0, v: 1 }] },
            { round: 1, points: [{ t: 100, v: 1 }] },
          ],
          total: [{ t: 0, v: 1 }, { t: 100, v: 2 }],
        },
        equipe(1, [[0, 0]]),
      ]),
      frame: 100,
    })
    expect(screen.getByText('Round 2')).toBeInTheDocument()
    expect(screen.getByTitle('Round 2 of 2')).toBeInTheDocument()
  })
})
