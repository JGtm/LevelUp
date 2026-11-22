/**
 * playerInstantScore.test.ts — UN SEUL SCORE À L'INSTANT : la tuile et la fiche lisent la même valeur.
 *
 * Ce qu'ils protègent : à chaque image, la lecture rangée pour la rangée des tuiles (`playerScoreAt`) et celle de
 * la fiche (`playerInstantEstimatesAt`) rendent le même nombre, garde tenue ou refusée ; une garde refusée rend
 * `null` (jamais zéro) ; le calque de score du film, décalé d'une image ou durablement écarté, ne décide de rien ;
 * un joueur étranger au lobby n'a pas de score.
 */
import { describe, expect, it } from 'vitest'

import { INSTANT_LAST_FRAME, instantDoc, instantPlayer, instantPlayers, instantSheet } from '../test/instantFixture'
import { playerInstantEstimatesAt, readInstantEstimatesSeries } from './playerInstantEstimates'
import { playerScoreAt, readInstantScoreSeries } from './playerInstantScore'

/** Le score de la fiche et celui de la tuile de chaque joueur, à chaque image du match. */
function lectures(doc = instantDoc(), sheet = instantSheet()) {
  const players = instantPlayers(doc, sheet)
  const tuiles = readInstantScoreSeries(doc, players)
  const fiche = readInstantEstimatesSeries({ doc, players })
  return { players, tuiles, fiche }
}

describe('playerInstantScore — la tuile et la fiche lisent la même valeur', () => {
  it('à chaque image et pour chaque joueur du lobby, garde tenue', () => {
    const { players, tuiles, fiche } = lectures()
    for (const p of players) {
      for (let f = 0; f <= INSTANT_LAST_FRAME; f++) {
        expect(playerScoreAt(tuiles, p, f), `${p.xuid} @${f}`).toBe(playerInstantEstimatesAt(fiche, p, f).score)
      }
    }
    expect(playerScoreAt(tuiles, instantPlayer(players, 'A'), 70)).toBe(200)
  })

  it('à chaque image, garde refusée : la feuille d’Alpha ne dit pas la somme des points', () => {
    const sheet = instantSheet().map((r) => (r.xuid === 'A' ? { ...r, score: 999 } : r))
    const { players, tuiles, fiche } = lectures(instantDoc(), sheet)
    const alpha = instantPlayer(players, 'A')
    for (let f = 0; f <= INSTANT_LAST_FRAME; f += 7) {
      expect(playerScoreAt(tuiles, alpha, f)).toBeNull()
      expect(playerInstantEstimatesAt(fiche, alpha, f).score).toBeNull()
    }
    // Les autres joueurs, eux, gardent leur score.
    expect(playerScoreAt(tuiles, instantPlayer(players, 'E'), INSTANT_LAST_FRAME)).toBe(400)
  })

  it('garde refusée : sans ligne de feuille, document incomplet, horloge non recalée — aucun score, jamais zéro', () => {
    const sansFeuille = lectures(instantDoc(), instantSheet().filter((r) => r.xuid !== 'A'))
    expect(playerScoreAt(sansFeuille.tuiles, instantPlayer(sansFeuille.players, 'A'), 100)).toBeNull()

    const base = instantDoc()
    const incomplet = lectures(instantDoc({ coverage: { ...base.coverage, scoreActions: { complete: false } } as never }))
    const horloge = lectures(instantDoc({ coverage: { ...base.coverage, originResolved: false } as never, originMs: undefined }))
    for (const l of [incomplet, horloge]) {
      for (const p of l.players) expect(playerScoreAt(l.tuiles, p, 100)).toBeNull()
    }
  })

  it('un verdict faux du décodeur : la cause est « score incomplet », pour la tuile comme pour la fiche', () => {
    const base = instantDoc()
    const doc = instantDoc({ coverage: { ...base.coverage, scoreActions: { complete: false } } as never })
    const players = instantPlayers(doc, instantSheet())
    const series = readInstantScoreSeries(doc, players)
    for (const p of players) expect(series.get(p.xuid)).toEqual({ ticks: null, absence: 'score-incomplet' })
  })

  it('le calque de score du film ne décide pas : décalé d’une image, il n’est pas lu', () => {
    const base = instantDoc()
    const calque = {
      teams: null,
      players: [
        {
          xuid: 'A',
          // Les mêmes paliers que les points par action, une image plus tard.
          score: { rounds: null, total: [{ t: 21, v: 100 }, { t: 61, v: 200 }, { t: 101, v: 300 }] },
          kills: { rounds: null, total: [] },
          deaths: { rounds: null, total: [] },
          assists: { rounds: null, total: [] },
        },
      ],
    } as never
    const { players, tuiles, fiche } = lectures(instantDoc({ scoreTimeline: calque, coverage: base.coverage }))
    const alpha = instantPlayer(players, 'A')
    expect(playerScoreAt(tuiles, alpha, 20)).toBe(100)
    expect(playerInstantEstimatesAt(fiche, alpha, 20).score).toBe(100)
  })

  it('un joueur étranger au lobby n’a pas de score', () => {
    const { players, tuiles } = lectures()
    const etranger = { ...instantPlayer(players, 'A'), xuid: 'inconnu' }
    expect(playerScoreAt(tuiles, etranger, 100)).toBeNull()
  })
})
