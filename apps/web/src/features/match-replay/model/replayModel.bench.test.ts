/// <reference types="node" />
// @vitest-environment node
/**
 * MESURE E2.3 — CE QUE COÛTE UNE BASCULE DE POINT DE VUE, sur l'artefact RÉEL du témoin.
 *
 * # POURQUOI CETTE MESURE EXISTE (2026-09-06, lot L2b du plan « frise, point de vue »)
 *
 * Le point de vue entre dans les dépendances de la mémo de `useReplayModel` : en changer refait
 * TOUTE la jointure — `buildPlayers` sur le roster entier, l'alignement statistique du fil, la
 * présence, les médias. Le plan prévoyait deux issues : sous 50 ms on ne découpe rien, au-delà
 * le superviseur tranche. La règle du dépôt est de MESURER avant d'optimiser, pas de découper
 * une mémo « au cas où » — un découpage préventif coûterait deux chemins de dépendances à tenir
 * pour un gain hypothétique.
 *
 * # CE QUE CE FICHIER MESURE, ET SUR QUOI
 *
 * L'artefact réel `4ecdf3e7` (CTF Arena, 2 714 images, 9 joueurs, 38 vies, ~941 Kio), lu du
 * cache du dépôt et normalisé par le chemin de la page (`normalizeReplayDocument`, la queryFn).
 * Le tableau de score et les kills sont fabriqués depuis le roster de l'artefact : la vue match
 * n'est pas dans le cache, et ce qui compte ici est le VOLUME de jointure, pas l'exactitude des
 * noms.
 *
 * # CE QU'IL N'EST PAS
 *
 * Pas un test de non-régression de performance : aucun seuil n'y échoue. Le chiffre est écrit
 * dans la sortie du test, et reporté en commentaire au-dessus de la mémo (`useReplayModel.ts`).
 * Un budget qui casse la CI sur une machine partagée serait un faux positif hebdomadaire.
 *
 * L'ARTEFACT N'EST PAS VERSIONNÉ : la CI ne l'a pas, et le test se saute alors proprement.
 */
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow, MatchViewResponse, ReplayDocument } from '@/lib/api/types'
import { normalizeReplayDocument } from '@/lib/replay/replayNormalize'

import { racineDuDepot } from '../test/featureFiles'
import { buildReplayModel } from './replayModel'

/**
 * Le témoin du plan : `data/cache/replays/halo_infinite/4ecdf3e7.json`, en LECTURE SEULE.
 *
 * TROIS ENDROITS OÙ IL PEUT VIVRE, dans cet ordre. Le cache d'artefacts n'est pas versionné :
 * un WORKTREE dédié ne le porte pas (le sien ne contient que les images de rang), il vit dans
 * le checkout principal — d'où le second chemin, relatif au dossier des checkouts. La variable
 * d'environnement, elle, sert quand les deux échouent. Aucun n'existe en CI : le test se saute.
 */
const CANDIDATS = [
  process.env.LEVELUP_REPLAY_CACHE,
  resolve(racineDuDepot(), 'data', 'cache', 'replays', 'halo_infinite'),
  resolve(racineDuDepot(), '..', 'LevelUp-go-migration', 'data', 'cache', 'replays', 'halo_infinite'),
].filter((d): d is string => !!d)

const ARTEFACT =
  CANDIDATS.map((d) => resolve(d, '4ecdf3e7.json')).find((f) => existsSync(f)) ??
  resolve(CANDIDATS[0], '4ecdf3e7.json')

const ITERATIONS = 20
/** ~90 kills, la densité d'un CTF Arena de 4 minutes et demie. */
const KILLS = 90

/** La médiane, et non la moyenne : une pause du ramasse-miettes ne doit pas la décider. */
function mediane(valeurs: readonly number[]): number {
  const tri = [...valeurs].sort((a, b) => a - b)
  const milieu = Math.floor(tri.length / 2)
  return tri.length % 2 === 0 ? (tri[milieu - 1] + tri[milieu]) / 2 : tri[milieu]
}

/** Le tableau de score fabriqué depuis le roster du film : deux camps, une ligne « moi ». */
function scoreboardDuRoster(roster: readonly { xuid: string; name?: string }[]): MatchScoreboardRow[] {
  return roster.map((p, i) => ({
    xuid: p.xuid,
    gamertag: p.name ?? p.xuid,
    team_side: i % 2 === 0 ? 't0' : 't1',
    is_me: i === 0,
  })) as unknown as MatchScoreboardRow[]
}

/** Des kills répartis régulièrement sur la durée jouée, acteurs et victimes pris au roster. */
function killsDuRoster(roster: readonly { xuid: string }[], dureeMs: number) {
  return Array.from({ length: KILLS }, (_, i) => ({
    event_type: 'kill',
    actor_xuid: roster[i % roster.length].xuid,
    victim_xuid: roster[(i + 1) % roster.length].xuid,
    event_time_ms: Math.round(((i + 1) / (KILLS + 1)) * dureeMs),
  }))
}

describe('E2.3 — coût d’une bascule de point de vue sur le match témoin', () => {
  it.skipIf(!existsSync(ARTEFACT))(
    'reconstruit le modèle entier en une poignée de millisecondes',
    () => {
      const brut = JSON.parse(readFileSync(ARTEFACT, 'utf8')) as ReplayDocument
      const doc = normalizeReplayDocument(brut)
      const scoreboard = scoreboardDuRoster(doc.roster)
      // La durée jouée du film ; `durationMs` est optionnel au contrat, on retombe sur le
      // produit des images par leur intervalle — la même arithmétique que l'horloge du rejeu.
      const dureeMs = doc.durationMs ?? doc.frameCount * (doc.frameIntervalMs ?? 100)
      const matchView = {
        header: {
          t0_ms: 20_000,
          playable_duration_seconds: Math.round(dureeMs / 1000),
          start_time: '2026-09-01T12:00:00Z',
        },
        team_tab: { scoreboard },
        combat_tab: { highlight_events: killsDuRoster(doc.roster, dureeMs) },
        media_tab: { media_items: [] },
      } as unknown as MatchViewResponse

      // DEUX POINTS DE VUE, ALTERNÉS : c'est bien une BASCULE qu'on chronomètre, pas un cache
      // qui se réchaufferait sur le même sujet.
      const sujets = [doc.roster[0].xuid, doc.roster[1].xuid]
      // Un tour à blanc : la première exécution paie la compilation JIT, pas la jointure.
      const temoin = buildReplayModel(doc, matchView, null, sujets[0])
      // ON MESURE BIEN LE TRAVAIL COMPLET, et pas une jointure qui se serait arrêtée à la
      // première porte fermée : sans ces assertions, un chiffre de 0,1 ms ne prouverait rien.
      expect(temoin.players).toHaveLength(doc.roster.length)
      expect(temoin.window).not.toBeNull()
      expect(temoin.feed.filter((e) => e.kill).length).toBe(KILLS)

      const mesures: number[] = []
      for (let i = 0; i < ITERATIONS; i += 1) {
        const t = performance.now()
        const modele = buildReplayModel(doc, matchView, null, sujets[i % 2])
        mesures.push(performance.now() - t)
        expect(modele.viewpoint).toBe(sujets[i % 2])
      }

      const ms = mediane(mesures)
      // LA MESURE EST LE PRODUIT DE CE TEST (E2.3) : elle s'écrit dans sa sortie, pas dans une
      // assertion. C'est le seul `console` légitime du dossier — un test qui mesure doit dire
      // ce qu'il a mesuré, sinon il ne mesure rien.
      console.info(
        `[E2.3] buildReplayModel sur 4ecdf3e7 (${doc.frameCount} images, ${doc.roster.length} joueurs, ` +
          `${doc.tracks.length} vies, ${KILLS} kills) : médiane ${ms.toFixed(2)} ms sur ${ITERATIONS} bascules ` +
          `(min ${Math.min(...mesures).toFixed(2)} / max ${Math.max(...mesures).toFixed(2)})`,
      )
      expect(mesures).toHaveLength(ITERATIONS)
    },
  )
})
