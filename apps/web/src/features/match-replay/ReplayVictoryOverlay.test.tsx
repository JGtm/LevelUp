/**
 * Tests — ReplayVictoryOverlay (l'écran de fin de match).
 *
 * CE QU'ILS PROTÈGENT : les quatre affirmations de l'écran — il n'apparaît QU'À la fin, il
 * porte l'identité DU JOUEUR DE LA PAGE (en défaite comme en victoire, amendement du
 * 2026-08-26), il ne prête aucune identité à une égalité, et il ne se rend pas quand la lecture
 * de fin est absente. Le calcul de l'issue est éprouvé chez `victoryLogic.test.ts` ; ici on
 * éprouve le rendu.
 *
 * Une régression sur la visibilité ne casse rien à l'exécution : elle laisse un voile
 * par-dessus le terrain pendant tout le match, ou n'annonce jamais le résultat.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { normalizeScoreTimeline, type ReplayScoreDocument } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { ReplayVictoryOverlay } from './ReplayVictoryOverlay'
import type { ReplayWindowBounds } from './replayWindow'

/** Une série d'équipe à manche unique. */
function equipe(teamId: number, points: Array<[number, number]>) {
  const pts = points.map(([t, v]) => ({ t, v }))
  return { teamId, rounds: [{ round: 0, points: pts }], total: pts }
}

/** Témoin Slayer : t0 finit à 50, t1 à 30 (le calque publie jusqu'au frame 480). */
const SLAYER_TEAMS = [
  equipe(0, [
    [0, 0],
    [480, 50],
  ]),
  equipe(1, [
    [0, 0],
    [480, 30],
  ]),
]

/** Un document de rejeu réduit à ce que l'écran lui demande (type structurel). */
function docOf(teams: unknown[]): ReplayScoreDocument {
  return { originMs: 0, scoreTimeline: normalizeScoreTimeline({ teams, players: [] } as never) }
}

const SB: MatchScoreboardRow[] = [
  { xuid: 'moi', team_side: 't0', is_me: true },
  { xuid: 'eux', team_side: 't1', is_me: false },
] as MatchScoreboardRow[]

const META = new Map([
  ['moi', { gamertag: 'Moi', ally: true }],
  ['eux', { gamertag: 'Eux', ally: false }],
])

/** La fenêtre de gameplay du témoin : la fin tombe au frame 500. */
const WINDOW: ReplayWindowBounds = {
  startFrame: 10,
  // Le préambule de lecture est à l'image zéro ici (le coup d'envoi tombe à 1 s) : l'écran de
  // fin ne le lit pas — il ne connaît que la borne HAUTE.
  leadInFrame: 0,
  endFrame: 500,
  startMs: 1_000,
  endMs: 50_000,
}

// Eagle (`t0`) vaut `#3B9DFF` au référentiel `lib/halo/teamNames.ts` et Cobra (`t1`)
// `#FE3939` : depuis le 2026-08-28 l'écran ne les porte PLUS (la couleur est celle que
// l'utilisateur a réglée), et ces deux constantes servent justement à le vérifier.
const EAGLE = 'rgb(59, 157, 255)'
const COBRA = 'rgb(254, 57, 57)'

/** Le token de camp allié — la couleur de l'écran de fin depuis le retour du 2026-08-28. */
const ALLY_TOKEN = 'var(--ac-team-ally)'

function renderOverlay(over: Partial<Parameters<typeof ReplayVictoryOverlay>[0]> = {}) {
  return render(
    <ReplayVictoryOverlay
      doc={docOf(SLAYER_TEAMS)}
      scoreboard={SB}
      xuidMeta={META}
      outcomeCode={2}
      outcomeLabel="Victoire"
      playWindow={WINDOW}
      frame={500}
      titleSlug="halo_infinite"
      locale="fr"
      {...over}
    />,
  )
}

describe('ReplayVictoryOverlay — le verdict vient du backend', () => {
  it('écrit `outcome_label` tel quel, sans le reformuler', () => {
    renderOverlay()
    expect(screen.getByText('Victoire')).toBeInTheDocument()
  })

  it('écrit « Défaite » quand c’est le verdict servi', () => {
    renderOverlay({ outcomeCode: 3, outcomeLabel: 'Défaite' })
    expect(screen.getByText('Défaite')).toBeInTheDocument()
  })

  it('sans libellé servi : rien — un panneau qui n’annonce rien serait pire', () => {
    const { container } = renderOverlay({ outcomeLabel: undefined })
    expect(container).toBeEmptyDOMElement()
  })
})

describe('ReplayVictoryOverlay — l’habillage est celui du joueur de la page', () => {
  it('en VICTOIRE : nom, logo et couleur de mon équipe', () => {
    const { container } = renderOverlay()
    expect(screen.getByText('Équipe Eagle')).toBeInTheDocument()
    expect(container.querySelector('img')).toHaveAttribute(
      'src',
      '/titles/halo_infinite/teams/0.png',
    )
    expect(container.innerHTML).toContain(ALLY_TOKEN)
  })

  it('en DÉFAITE : TOUJOURS mon équipe — jamais l’emblème du vainqueur', () => {
    const { container } = renderOverlay({ outcomeCode: 3, outcomeLabel: 'Défaite' })
    expect(screen.getByText('Équipe Eagle')).toBeInTheDocument()
    expect(screen.queryByText('Équipe Cobra')).not.toBeInTheDocument()
    expect(container.querySelector('img')).toHaveAttribute(
      'src',
      '/titles/halo_infinite/teams/0.png',
    )
    expect(container.innerHTML).toContain(ALLY_TOKEN)
  })

  it('suit le joueur de la page quand il est dans l’autre camp', () => {
    const { container } = renderOverlay({
      scoreboard: [
        { xuid: 'moi', team_side: 't0', is_me: false },
        { xuid: 'eux', team_side: 't1', is_me: true },
      ] as MatchScoreboardRow[],
    })
    expect(screen.getByText('Équipe Cobra')).toBeInTheDocument()
    // Le camp change, la couleur NON : c'est toujours celle du joueur de la page.
    expect(container.innerHTML).toContain(ALLY_TOKEN)
  })

  it('porte le token de camp RÉGLABLE, jamais la couleur officielle du jeu (D1)', () => {
    const { container } = renderOverlay()
    expect(container.innerHTML).toContain(ALLY_TOKEN)
    expect(container.innerHTML).not.toContain(EAGLE)
    expect(container.innerHTML).not.toContain(COBRA)
  })

  it('applique la recette de teinte partagée (fond à 22 %, trait à 55 %)', () => {
    const { container } = renderOverlay()
    expect(container.innerHTML).toContain(`${ALLY_TOKEN} 22%`)
    expect(container.innerHTML).toContain(`${ALLY_TOKEN} 55%`)
  })

  it('le BLOC ne porte QUE le statut — nom et score sont dehors (retour du 2026-08-28)', () => {
    renderOverlay()
    const bloc = screen.getByText('Victoire')
    expect(bloc.textContent).toBe('Victoire')
    expect(bloc).not.toContainElement(screen.getByText('Équipe Eagle'))
    expect(bloc).not.toContainElement(screen.getByText('50'))
  })

  it('le logo est décoratif — le nom est écrit juste dessous', () => {
    const { container } = renderOverlay()
    expect(container.querySelector('img')).toHaveAttribute('aria-hidden')
  })
})

describe('ReplayVictoryOverlay — le score et l’accessibilité', () => {
  it('écrit le score FINAL, celui de la borne de fin', () => {
    renderOverlay()
    expect(screen.getByText('50')).toBeInTheDocument()
    expect(screen.getByText('30')).toBeInTheDocument()
  })

  it('montre les DEUX camps même en défaite', () => {
    renderOverlay({ outcomeCode: 3, outcomeLabel: 'Défaite' })
    expect(screen.getByText('50')).toBeInTheDocument()
    expect(screen.getByText('30')).toBeInTheDocument()
  })

  it('laisse passer les clics — la frise est dessous', () => {
    renderOverlay()
    expect(screen.getByRole('status').className).toContain('pointer-events-none')
  })

  it('s’annonce poliment aux lecteurs d’écran', () => {
    renderOverlay()
    const panel = screen.getByRole('status')
    expect(panel).toHaveAttribute('aria-live', 'polite')
    expect(panel).toHaveAttribute('aria-label', 'Fin du match')
  })

  it('nomme la région en anglais sous la locale EN', () => {
    renderOverlay({ locale: 'en' })
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'End of match')
  })
})

describe('ReplayVictoryOverlay — l’égalité', () => {
  it('reste neutre : ni logo, ni couleur, ni nom d’équipe', () => {
    const { container } = renderOverlay({ outcomeCode: 1, outcomeLabel: 'Égalité' })
    expect(screen.getByText('Égalité')).toBeInTheDocument()
    expect(screen.queryByText(/Équipe /)).not.toBeInTheDocument()
    expect(container.querySelector('img')).toBeNull()
    expect(container.innerHTML).not.toMatch(/rgb\(/)
  })

  it('garde le score final : une égalité a des chiffres', () => {
    renderOverlay({ outcomeCode: 1, outcomeLabel: 'Égalité' })
    expect(screen.getByText('50')).toBeInTheDocument()
  })
})

describe('ReplayVictoryOverlay — quand il ne se rend pas', () => {
  it('avant la borne de fin : rien du tout', () => {
    const { container } = renderOverlay({ frame: 499 })
    expect(container).toBeEmptyDOMElement()
  })

  it('au-delà de la borne (frise tirée au bout) : il reste affiché', () => {
    renderOverlay({ frame: 4_000 })
    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('sans cadrage (D-A3) : rien, même à la dernière image du film', () => {
    const { container } = renderOverlay({ playWindow: null, frame: 10_000 })
    expect(container).toBeEmptyDOMElement()
  })

  it('lecture nulle (FFA) : rien', () => {
    const { container } = renderOverlay({
      scoreboard: [
        { xuid: 'moi', team_side: null, is_me: true },
        { xuid: 'eux', team_side: null, is_me: false },
      ] as MatchScoreboardRow[],
    })
    expect(container).toBeEmptyDOMElement()
  })

  it('code d’issue absent : rien', () => {
    const { container } = renderOverlay({ outcomeCode: undefined })
    expect(container).toBeEmptyDOMElement()
  })

  it('abandon (code 4) : rien', () => {
    const { container } = renderOverlay({ outcomeCode: 4, outcomeLabel: 'Abandon' })
    expect(container).toBeEmptyDOMElement()
  })

  it('mode sans compteur : le panneau reste, la ligne de score disparaît (D-B4)', () => {
    renderOverlay({ doc: docOf([]) })
    expect(screen.getByText('Victoire')).toBeInTheDocument()
    expect(screen.queryByText('50')).not.toBeInTheDocument()
  })
})

// ─── Le score final vient de l'API sur un mode à manches ───────────────────
//
// Le calque du film rendrait ici les points de la DERNIÈRE MANCHE, présentés comme le score
// du match. Sur un mode à manches c'est faux, et c'est le défaut le plus visible que ce
// chantier corrige : l'écran de fin doit dire ce que dit la vue match.
describe('ReplayVictoryOverlay — score final servi par l’API', () => {
  it('écrit le compte de manches quand la page le fournit', () => {
    renderOverlay({ finalScore: { ally: 2, enemy: 1 } })
    expect(screen.getByLabelText('Équipe alliée')).toHaveTextContent('2')
    expect(screen.getByLabelText('Équipe adverse')).toHaveTextContent('1')
  })

  it('garde la lecture du calque quand la page ne fournit rien (mode en points)', () => {
    renderOverlay({ finalScore: null })
    // Les valeurs du témoin Slayer du fichier, inchangées.
    expect(screen.getByLabelText('Équipe alliée')).toBeInTheDocument()
    expect(screen.getByLabelText('Équipe adverse')).toBeInTheDocument()
  })

  it('accepte un zéro : 2 manches à 0 est une mesure', () => {
    renderOverlay({ finalScore: { ally: 2, enemy: 0 } })
    expect(screen.getByLabelText('Équipe adverse')).toHaveTextContent('0')
  })
})
