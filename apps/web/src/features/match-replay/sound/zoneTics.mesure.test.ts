/// <reference types="node" />
/**
 * zoneTics.mesure.test.ts — LOT 5.8.6 : LE TIC DE SCORE DES BASES, MESURÉ SUR LE PARC CUIT.
 *
 * ## Ce que cette mesure répond, et pourquoi elle est nécessaire
 *
 * Le tic de score était ancré sur le DÉBUT de la domination, battait la seconde, et s'arrêtait au
 * 180e tic. Deux questions se posaient, et aucune ne se tranche sans les documents :
 *
 *  1. LE PLAFOND MORDAIT-IL VRAIMENT ? Un plafond de 180 tics ne se voit que si une domination
 *     dépasse trois minutes. La mesure compte les intervalles de domination qui les dépassent, et
 *     combien de secondes de domination restaient SANS AUCUN TIC sous l'ancienne règle.
 *  2. L'HORLOGE DE SCORE EST-ELLE LÀ ? Le tic vient désormais des MARCHES de
 *     `scoreTimeline.teams[].total`. Si le calque de score ne couvrait pas les documents à zones,
 *     le repli synthétique répondrait partout et le lot n'aurait rien changé.
 *
 * ## Pas de seconde implémentation
 *
 * Les intervalles de domination viennent de `zoneDominationIntervals`, LA FONCTION DE PRODUCTION
 * (exportée pour cet instrument) : recopier son balayage ici en aurait fait une version qui dérive
 * sans que rien ne le voie (leçon « DDL de test recopiées »). Les tics, eux, viennent de
 * `zoneSoundEvents` — la chaîne réelle. L'AVANT/APRÈS s'obtient en RETIRANT `scoreTimeline` du
 * document, ce qui exerce le repli synthétique par le même code.
 *
 * ## Régime
 *
 *     ZONE_MESURE=1 npx vitest run src/features/match-replay/sound/zoneTics.mesure.test.ts
 *     ZONE_MESURE_DIR=<abs>/data/cache/replays/halo_infinite   (défaut : celui du dépôt)
 *
 * Lecture seule sur des documents DÉJÀ CUITS : aucun décodage de film, aucune base ouverte.
 * Sans `ZONE_MESURE`, la suite est ignorée — même porte que `tourelleVisee.mesure.test.ts`.
 */
import { existsSync, readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { racineDuDepot } from '../test/featureFiles'
import { testReplayDoc } from '../test/testDoc'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { ZONE_SOUND_STEMS, ZONE_TICK_BUDGET_PAR_INTERVALLE, zoneDominationIntervals, zoneSoundEvents } from './zoneSound'

const PORTE = process.env.ZONE_MESURE === '1'

function dossier(): string {
  return (
    process.env.ZONE_MESURE_DIR ?? join(racineDuDepot(), 'data', 'cache', 'replays', 'halo_infinite')
  )
}

/** Les documents à ZONES SIMULTANÉES du parc : des zones, une jauge, aucun marqueur de colline. */
function documentsAZones(): { court: string; doc: ReplayDocumentReady }[] {
  const dir = dossier()
  if (!existsSync(dir)) return []
  const out: { court: string; doc: ReplayDocumentReady }[] = []
  for (const f of readdirSync(dir)) {
    if (!f.endsWith('.json') || f.includes('.derived.')) continue
    const brut = JSON.parse(readFileSync(join(dir, f), 'utf8')) as Partial<ReplayDocument>
    const doc = testReplayDoc(brut)
    const zones = doc.zoneStates ?? []
    if (zones.length < 2) continue
    if (zones.some((z) => z.spans.some((s) => s.active))) continue
    out.push({ court: f.replace('.json', ''), doc })
  }
  return out.sort((a, b) => a.court.localeCompare(b.court))
}

/** Les tics de score d'un document, tous camps confondus (le compte ne dépend pas du camp). */
function tics(doc: ReplayDocumentReady): number[] {
  const stems: readonly string[] = [ZONE_SOUND_STEMS.tick.ally, ZONE_SOUND_STEMS.tick.enemy]
  return zoneSoundEvents(doc, 0)
    .filter((e) => stems.includes(e.stem))
    .map((e) => e.ms)
}

/** Le MÊME document, privé de son horloge de score : c'est l'AVANT, par le même code. */
function sansHorlogeDeScore(doc: ReplayDocumentReady): ReplayDocumentReady {
  return { ...doc, scoreTimeline: undefined }
}

describe.skipIf(!PORTE)('mesure — le tic de score des bases (lot 5.8.6)', () => {
  it('rend le tableau de la portee du plafond et de la couverture de l horloge', () => {
    const docs = documentsAZones()
    expect(docs.length, 'aucun document a zones simultanees dans le parc').toBeGreaterThan(0)
    const budgetMs = ZONE_TICK_BUDGET_PAR_INTERVALLE * 1000
    for (const { court, doc } of docs) {
      const ivs = zoneDominationIntervals(doc.zoneStates ?? [])
      const durees = ivs.map((iv) => frameToMs(iv.t1, doc) - frameToMs(iv.t0, doc))
      const longs = durees.filter((d) => d > budgetMs)
      // LES SECONDES QUE L'ANCIEN PLAFOND LAISSAIT SANS AUCUN TIC : au-dela du 180e tic, la
      // boucle s'arretait net jusqu'a la fin de l'intervalle.
      const perduesS = longs.reduce((a, d) => a + (d - budgetMs), 0) / 1000
      const avant = tics(sansHorlogeDeScore(doc))
      const apres = tics(doc)
      const equipes = (doc.scoreTimeline?.teams ?? []).map(
        (t) => `${t.teamId}:${t.total.length}`,
      )
      // UN SEUL FOYER D'ÉCRITURE, et c'est `process.stdout` et non `console.log` : vitest
      // capture le second et la mesure resterait invisible (même choix que l'instrument du
      // lot 5.5).
      process.stdout.write(
        `=== ${court} : ${ivs.length} intervalles de domination, duree totale ` +
          `${Math.round(durees.reduce((a, b) => a + b, 0) / 1000)} s, la plus longue ` +
          `${Math.round(Math.max(0, ...durees) / 1000)} s\n` +
          `    plafond de ${ZONE_TICK_BUDGET_PAR_INTERVALLE} tics : ${longs.length} intervalle(s) ` +
          `le depassaient, soit ${Math.round(perduesS)} s de domination SANS AUCUN TIC\n` +
          `    tics emis : REPLI synthetique ${avant.length} -> HORLOGE DE SCORE ${apres.length} ` +
          `(escaliers publies ${equipes.join(', ') || 'aucun'})
`,
      )
    }
  })
})
