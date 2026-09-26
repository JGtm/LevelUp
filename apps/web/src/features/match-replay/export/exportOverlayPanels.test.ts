/**
 * exportOverlayPanels.test.ts — LE PANNEAU DE FIN REPEINT DANS LA VIDÉO dit-il la même
 * chose que celui affiché à l'écran ?
 *
 * Les deux surfaces annoncent le MÊME match : si elles tranchaient séparément, la vidéo
 * exportée afficherait les points de la dernière manche (« 100 - 43 ») pendant que la page
 * dit « 2 - 1 ». Le jumeau DOM est couvert par ReplayVictoryOverlay.test.tsx ; ce fichier
 * couvre le choix côté export, qui ne l'était pas (constat de revue adversariale du
 * 2026-08-29).
 */
import { describe, expect, it } from 'vitest'

import { buildOverlayPanelSource, exportFinalScore, type OverlayPanelDeps } from './exportOverlayPanels'
import { normalizeScoreTimeline, type ReplayScoreDocument } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import type { ReplayWindowBounds } from '../model/replayWindow'

const film = { ally: { score: 100 }, enemy: { score: 43 } }

describe('exportFinalScore', () => {
  it("préfère le score servi par l'API quand il existe", () => {
    expect(exportFinalScore({ ally: 2, enemy: 1 }, film)).toEqual({ ally: 2, enemy: 1 })
  })

  it('retombe sur la lecture du calque du film quand l’API ne dit rien', () => {
    expect(exportFinalScore(null, film)).toEqual({ ally: 100, enemy: 43 })
    expect(exportFinalScore(undefined, film)).toEqual({ ally: 100, enemy: 43 })
  })

  it("n'invente rien quand aucune des deux sources ne parle", () => {
    expect(exportFinalScore(null, null)).toBeNull()
  })

  it('accepte un zéro : 2 manches à 0 est une mesure, pas une absence', () => {
    expect(exportFinalScore({ ally: 2, enemy: 0 }, film)).toEqual({ ally: 2, enemy: 0 })
  })
})

/**
 * AJOUT DU 2026-09-07 — LE MOT DU PANNEAU EXPORTÉ SUIT LE POINT DE VUE, ET N'A QU'UNE SOURCE.
 *
 * Le clip peignait son verdict avec `outcome.label`, c'est-à-dire `header.outcome_label` : le
 * mot du JOUEUR DE LA PAGE, fabriqué par une map Go en dur en français. Vu depuis un adversaire
 * d'un match gagné, `readVictory` permutait bien l'issue et le camp du panneau — mais le titre
 * disait encore « Victoire », au-dessus de l'équipe perdante. Le libellé de l'issue LUE arrive
 * désormais dans `viewedLabel`, résolu par `useReplayCapture` (un hook ; ce module est pur)
 * depuis les mappings du titre, et c'est le SEUL mot que ce module accepte : sans lui, silence.
 */
describe('buildOverlayPanelSource — le verdict peint dans la vidéo', () => {
  const DOC = {
    originMs: 0,
    frameIntervalMs: 100,
    scoreTimeline: normalizeScoreTimeline({ teams: [], players: [] } as never),
  } as unknown as ReplayDocumentReady & ReplayScoreDocument

  const SB = [
    { xuid: 'moi', team_side: 't0', is_me: true },
    { xuid: 'eux', team_side: 't1', is_me: false },
  ] as MatchScoreboardRow[]

  const WINDOW: ReplayWindowBounds = {
    startFrame: 0,
    leadInFrame: 0,
    endFrame: 500,
    startMs: 0,
    endMs: 50_000,
  }

  const INK = {
    background: '#000',
    foreground: '#fff',
    muted: '#888',
    border: '#444',
    card: '#222',
  }

  function panel(over: Partial<OverlayPanelDeps> = {}) {
    return buildOverlayPanelSource({
      doc: DOC,
      scoreboard: SB,
      playWindow: WINDOW,
      outcome: { code: 2, viewedLabel: 'Victoire' },
      viewpoint: null,
      locale: 'fr',
      ink: INK,
      teamStyle: { background: '#111', border: '#222' },
      logo: null,
      ...over,
    }).panelAt(WINDOW.endFrame)
  }

  it('point de vue par défaut : le libellé de l’issue, tel quel', () => {
    expect(panel()?.status).toBe('Victoire')
  })

  it('point de vue du joueur de la page : le même mot, rien n’est permuté', () => {
    expect(panel({ viewpoint: 'moi' })?.status).toBe('Victoire')
  })

  it('vu depuis un adversaire : le libellé de l’issue PERMUTÉE, fourni par la couture', () => {
    const p = panel({
      viewpoint: 'eux',
      outcome: { code: 2, viewedLabel: 'Défaite' },
    })
    expect(p?.status).toBe('Défaite')
  })

  it('`viewedLabel` à `null` : pas de panneau — jamais une clé brute dans le clip', () => {
    // Le cas dégradé : les mappings du titre ne donnent aucun libellé pour l'issue lue. Même
    // silence que le DOM, et sans repli sur un second vocabulaire.
    expect(panel({ viewpoint: 'eux', outcome: { code: 2, viewedLabel: null } })).toBeNull()
  })

  it('`viewedLabel` ABSENT : pas de panneau non plus — il n’y a plus d’autre source', () => {
    expect(panel({ outcome: { code: 2 } })).toBeNull()
  })
})
