/// <reference types="node" />
/**
 * ReplayTeams.perf.test.tsx — LA MESURE JS DE LA COLONNE DES FICHES, rejouable, jamais en CI.
 *
 * CE QUE CE FICHIER MESURE, ET CE QU'IL NE MESURE PAS (plan fiches compactes 2026-09-06, I4).
 * La colonne rend TOUTES les fiches à chaque publication d'image (150 ms, `FRAME_PUBLISH_MS`
 * de `hooks/useReplayClock.ts`) — 25 sur le témoin BTB, 8 sur le témoin 4v4. Ce qu'on
 * chiffre ici est le coût JS de cette publication : le modèle par fiche (lectures pures) et
 * la réconciliation React de la colonne. jsdom ne PEINT pas : la peinture et les animations
 * CSS — le risque réel du lot — sont le volet navigateur, remis à l'utilisateur (DevTools).
 *
 * PROTOCOLE (fermé par le plan) : document construit par `testReplayDoc(JSON.parse(...))` ;
 * tableau bâti depuis `doc.roster` ; `<Profiler>` autour de `<ReplayTeams>` ; 20 rendus
 * d'échauffement puis 100 `rerender` aux images `floor(base + k × 1,5)` (la cadence de
 * publication réelle, 150 ms pour des images de 100 ms) ; métrique `actualDuration` en
 * p50 / p95 / total ; 5 répétitions, médiane des médianes. Puis la même boucle sur le modèle
 * seul (sans React). `base` vaut 5 000 quand le film le permet ; le témoin 4v4 n'a que
 * 4 985 images, il part donc de 45 % de sa durée — dit dans le résultat.
 *
 * GARDES : `REPLAY_PERF` absent = tout est sauté (jamais en CI) ; témoin absent = cas sauté.
 * Le témoin vit dans `data/cache/replays/halo_infinite/` du dépôt, ou dans le dossier que
 * `REPLAY_PERF_DIR` désigne (un worktree sans `data/` lit celui du principal, en lecture).
 *
 * Commande : `REPLAY_PERF=1 npx vitest run src/features/match-replay/ui/ReplayTeams.perf.test.tsx`
 */
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { Profiler, type ProfilerOnRenderCallback } from 'react'
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'
import { scoreTimelineOf } from '@/lib/replay/scoreTimeline'
import { stripBotSuffix } from '@/lib/players/displayName'

import { ReplayTeams } from './ReplayTeams'
import { REPLAY_TEXT } from '../i18n/i18n'
import { teleportMoments } from '../model/placementTeleport'
import { playerCardReadings, type CardFxScene } from '../model/playerCardReadings'
import type { PresenceHeader } from '../model/presenceFeed'
import { buildSeats, groupSeatsByTeam, seatOccupantAt } from '../model/seatLogic'
import { racineDuDepot } from '../test/featureFiles'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'
import { frameToMs, msToFrames } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import {
  buildPlayers,
  buildSlotOwnership,
  sideResolver,
  vitalityPresence,
} from '../../../lib/replay/rosterLogic'

const WARMUP = 20
const SAMPLES = 100
const REPEATS = 5
/** Cadence de publication réelle : 150 ms pour des images de 100 ms. */
const STEP = 1.5
const BASE_FRAME = 5_000

/**
 * LES TROIS MESURES. Les deux premières sont celles de l'étape 0 (AVANT), sans en-tête : la
 * colonne y rend le gabarit NORMAL sur les deux témoins — c'est la base, et elle reste
 * mesurée telle quelle. La troisième (ajoutée au 5.2, 2026-09-07) rejoue le témoin BTB sous
 * `mode_category: 'BTB'` : c'est la TUILE COMPACTE, donc le vrai APRÈS du lot — sans elle, la
 * lecture d'inventaire supplémentaire de `handCellHint` (jamais faite en normal) ne serait
 * jamais dans la mesure, et le seuil serait tenu à vide.
 */
const TEMOINS: readonly { nom: string; fichier: string; header?: PresenceHeader }[] = [
  { nom: 'BTB 12v12 (4f77afc1)', fichier: '4f77afc1.json' },
  { nom: '4v4 (000d5950)', fichier: '000d5950.json' },
  { nom: 'BTB 12v12 (4f77afc1) — tuile compacte (mode_category BTB)', fichier: '4f77afc1.json', header: { mode_category: 'BTB' } },
]

function dossierTemoins(): string {
  return process.env.REPLAY_PERF_DIR ?? join(racineDuDepot(), 'data', 'cache', 'replays', 'halo_infinite')
}

function chargerTemoin(fichier: string): ReplayDocumentReady {
  const brut = JSON.parse(readFileSync(join(dossierTemoins(), fichier), 'utf8')) as Partial<ReplayDocument>
  return testReplayDoc(brut)
}

/** Le tableau depuis le roster : deux camps en alternance (le film ne porte pas le camp). */
function tableauDepuisRoster(doc: ReplayDocumentReady) {
  return doc.roster.map((r, i) => {
    const side = i % 2 === 0 ? 't0' : 't1'
    if (r.bot) {
      return scoreboardRow(`bid(${i}.0)`, stripBotSuffix(r.name ?? ''), side, { is_bot: true })
    }
    return scoreboardRow(r.xuid, r.name ?? r.xuid, side)
  })
}

function imageDeBase(doc: ReplayDocumentReady): number {
  return doc.frameCount > BASE_FRAME + SAMPLES * STEP ? BASE_FRAME : Math.floor(doc.frameCount * 0.45)
}

function quantile(valeurs: readonly number[], q: number): number {
  const tri = [...valeurs].sort((a, b) => a - b)
  if (tri.length === 0) return Number.NaN
  const idx = Math.min(tri.length - 1, Math.max(0, Math.ceil(q * tri.length) - 1))
  return tri[idx]
}

interface Stats {
  p50: number
  p95: number
  total: number
}

function stats(valeurs: readonly number[]): Stats {
  return {
    p50: quantile(valeurs, 0.5),
    p95: quantile(valeurs, 0.95),
    total: valeurs.reduce((a, b) => a + b, 0),
  }
}

/** Médiane, par métrique, des statistiques de chaque répétition. */
function medianeDesRepetitions(reps: readonly Stats[]): Stats {
  return {
    p50: quantile(reps.map((r) => r.p50), 0.5),
    p95: quantile(reps.map((r) => r.p95), 0.5),
    total: quantile(reps.map((r) => r.total), 0.5),
  }
}

function arrondi(s: Stats): Record<keyof Stats, number> {
  return { p50: +s.p50.toFixed(3), p95: +s.p95.toFixed(3), total: +s.total.toFixed(1) }
}

/** Une répétition de la colonne : échauffement, puis SAMPLES rerender profilés. */
function mesurerColonne(doc: ReplayDocumentReady, base: number, header?: PresenceHeader): Stats {
  const scoreboard = tableauDepuisRoster(doc)
  const durees: number[] = []
  let mesure = false
  const onRender: ProfilerOnRenderCallback = (_id, _phase, actualDuration) => {
    if (mesure) durees.push(actualDuration)
  }
  const arbre = (frame: number) => (
    <Profiler id="ReplayTeams" onRender={onRender}>
      <ReplayTeams doc={doc} scoreboard={scoreboard} frame={frame} locale="fr" header={header} />
    </Profiler>
  )
  const vue = render(arbre(base))
  for (let k = 1; k < WARMUP; k++) vue.rerender(arbre(Math.floor(base + k * STEP)))
  mesure = true
  for (let k = 0; k < SAMPLES; k++) vue.rerender(arbre(Math.floor(base + k * STEP)))
  vue.unmount()
  expect(durees).toHaveLength(SAMPLES)
  return stats(durees)
}

/**
 * Le MODÈLE SEUL : les lectures pures qu'une fiche fait à chaque image (compteurs, état vital,
 * armes, équipement actif, position, zones, objectif, translocation, composition des effets),
 * pour tous les sièges de la colonne, sans React. La scène par document est construite une
 * fois, comme les `useMemo` de la colonne.
 *
 * La mesure AVANT (étape 0 du plan, journal) a été prise sur une RECOPIE de ces lectures
 * telles qu'elles étaient dans `PlayerCard` ; depuis l'étape 1.3 elles sont
 * `model/playerCardReadings.ts`, et c'est l'extraction qu'on mesure — le même code, déplacé.
 */
function mesurerModele(doc: ReplayDocumentReady, base: number): Stats {
  const text = REPLAY_TEXT.fr
  const scoreboard = tableauDepuisRoster(doc)
  const players = buildPlayers(doc, scoreboard)
  const seats = buildSeats(players, null, doc)
  const groups = groupSeatsByTeam(seats)
  const presence = vitalityPresence(doc)
  const flashFrames = Math.max(1, msToFrames(1_400, doc))
  const sideOfSlot = sideResolver(buildSlotOwnership(players))
  const fxScene: CardFxScene = {
    zones: { placements: doc.equipmentPlacements, sideOfSlot },
    time: { frameMs: frameToMs(1, doc), frames: doc.frameCount },
    teleports: teleportMoments(doc),
  }
  const scoreTimeline = scoreTimelineOf(doc)

  const lireColonne = (frame: number) => {
    let n = 0
    for (const g of groups) {
      for (const seat of g.seats) {
        const r = playerCardReadings({
          player: seatOccupantAt(seat, frame), doc, frame, presence, flashFrames, scoreTimeline, fxScene, text,
        })
        if (r.fx.title !== undefined) n++
      }
    }
    return n
  }

  for (let k = 0; k < WARMUP; k++) lireColonne(Math.floor(base + k * STEP))
  const durees: number[] = []
  for (let k = 0; k < SAMPLES; k++) {
    const frame = Math.floor(base + k * STEP)
    const t0 = performance.now()
    lireColonne(frame)
    durees.push(performance.now() - t0)
  }
  return stats(durees)
}

describe.skipIf(!process.env.REPLAY_PERF)('ReplayTeams — mesure JS de la colonne (REPLAY_PERF)', () => {
  for (const temoin of TEMOINS) {
    const chemin = join(dossierTemoins(), temoin.fichier)
    it.skipIf(!existsSync(chemin))(
      `${temoin.nom} : colonne (Profiler) et modèle seul, ${REPEATS} répétitions`,
      () => {
        const doc = chargerTemoin(temoin.fichier)
        const base = imageDeBase(doc)
        const colonne: Stats[] = []
        const modele: Stats[] = []
        for (let r = 0; r < REPEATS; r++) {
          colonne.push(mesurerColonne(doc, base, temoin.header))
          modele.push(mesurerModele(doc, base))
        }
        const resultat = {
          temoin: temoin.fichier,
          gabarit: temoin.header?.mode_category ?? 'normal',
          fiches: doc.roster.length,
          images: doc.frameCount,
          base,
          poses: doc.equipmentPlacements.length,
          colonneMs: arrondi(medianeDesRepetitions(colonne)),
          modeleMs: arrondi(medianeDesRepetitions(modele)),
          repetitions: { colonneP50: colonne.map((s) => +s.p50.toFixed(3)), modeleP50: modele.map((s) => +s.p50.toFixed(3)) },
        }
        process.stdout.write(`REPLAY_PERF ${JSON.stringify(resultat)}\n`)
        expect(resultat.colonneMs.p50).toBeGreaterThan(0)
        expect(resultat.modeleMs.p50).toBeGreaterThanOrEqual(0)
      },
      600_000,
    )
  }
})
