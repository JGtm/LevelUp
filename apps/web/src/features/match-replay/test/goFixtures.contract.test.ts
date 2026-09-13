/// <reference types="node" />
// @vitest-environment node
/**
 * goFixtures.contract.test.ts — LA PREUVE QUI TRAVERSE LA FRONTIÈRE GO / WEB.
 *
 * # CE QUE CE FICHIER FERME (architecture §12, garde-rails 1 et 2 ; lot 0.B, 2026-09-13)
 *
 * Les preuves du rejeu étaient ASYMÉTRIQUES : côté Go des goldens sur un film réel, côté web
 * une fixture écrite à la main déclarant `schemaVersion: 1` — un document que le serveur
 * n'envoie jamais. Rien ne vérifiait que ce que la cuisson ÉCRIT est ce que le rendu SAIT
 * LIRE. Ici, le document vient du producteur (`replay/contract_fixtures_test.go` le cuit
 * depuis le film de référence) et traverse toute la frontière du web :
 *
 *  1. le CONTRAT à l'exécution (`validateReplayDocument`) ;
 *  2. la NORMALISATION (`normalizeReplayDocument`) ;
 *  3. les LOGIQUES PURES des calques — celles qui prennent le document entier.
 *
 * # CE QU'IL N'EST PAS
 *
 * Ce n'est pas un test de VALEURS : les chiffres du film de référence sont figés côté Go
 * (`golden_assembly_test.go`), et les redire ici les dédoublerait sans les prouver deux fois.
 * Ce qu'on vérifie est qu'aucune étape ne LÈVE et qu'aucune ne rend un résultat vide là où le
 * document porte manifestement de la matière — c'est-à-dire exactement ce qu'un champ renommé
 * ou un tableau nul casserait.
 */
import { describe, expect, it } from 'vitest'

import { validateReplayDocument } from '@/lib/replay/replayDocumentSchema'
import { normalizeReplayDocument } from '@/lib/replay/replayNormalize'
import {
  frameToMs,
  framesPerSecond,
  isAliveAt,
  msToFrames,
  positionAt,
  sceneBounds,
  trackWindow,
} from '@/lib/replay/replayLogic'
import {
  buildPlayers,
  buildSlotOwnership,
  groupByTeam,
  loadoutAt,
  vitalityPresence,
} from '@/lib/replay/rosterLogic'

import { hasAbilityChargeLayer } from '../model/abilityChargeLogic'
import { buildEquipmentUsage } from '../model/equipmentUsageLogic'
import { equippedWeapons } from '../model/equippedLogic'
import { readHillHold } from '../model/hillHoldLogic'
import { buildPadControl } from '../model/padControlLogic'
import { MIN_RENDERABLE_SCHEMA_VERSION } from '../model/replaySchemaStatusLogic'
import { roundCount } from '../model/roundsLogic'
import { buildSeats } from '../model/seatLogic'
import { goFixtureManifest, loadGoFixtures } from './goFixtures'

const fixtures = loadGoFixtures()

describe('le jeu de fixtures produites par Go', () => {
  it('existe — sans lui, tous les tests ci-dessous ne garderaient rien', () => {
    expect(fixtures.length).toBeGreaterThan(0)
  })

  it('déclare la même version dans son manifeste que dans chaque document', () => {
    const m = goFixtureManifest()
    for (const f of fixtures) {
      expect(f.doc.schemaVersion, `${f.file} contredit le manifeste`).toBe(m.schemaVersion)
    }
  })

  it('est TOUT ENTIER au-dessus de la version minimale que le web déclare afficher', () => {
    for (const f of fixtures) {
      expect(
        f.doc.schemaVersion,
        `${f.file} est sous MIN_RENDERABLE_SCHEMA_VERSION : soit le seuil est faux, ` +
          `soit le web ne sait plus lire ce que Go cuit`,
      ).toBeGreaterThanOrEqual(MIN_RENDERABLE_SCHEMA_VERSION)
    }
  })
})

describe('chaque document produit par Go traverse la frontière du web', () => {
  for (const f of fixtures) {
    describe(f.file, () => {
      it('respecte le contrat à l’exécution', () => {
        expect(validateReplayDocument(f.doc)).toBeNull()
      })

      it('et un calque RENOMMÉ sur ce même document sort en manquement NOMMÉ', () => {
        // LA MUTATION DE LA REVUE RONDE 1 (constat C1), rejouée ici pour de bon. Elle
        // traversait : `z.object` dépouillait la clé inconnue, `shots?` acceptait l'absence,
        // `validateReplayDocument` rendait `null`, le badge disait « à jour » et le calque des
        // tirs se rendait VIDE. C'est le renommage silencieux que le contrat existe pour voir.
        const { shots, ...sansShots } = f.doc
        const issue = validateReplayDocument({ ...sansShots, shotz: shots })
        expect(issue).not.toBeNull()
        expect(issue).toContain('shotz')
      })

      it('porte de la matière — sans quoi les contrôles suivants seraient vides de sens', () => {
        expect(f.doc.tracks?.length ?? 0).toBeGreaterThan(0)
        expect(f.doc.frameCount).toBeGreaterThan(0)
      })

      it('passe la normalisation sans laisser un seul tableau nul à la racine', () => {
        const ready = normalizeReplayDocument(f.doc) as unknown as Record<string, unknown>
        const nuls = Object.keys(ready).filter((k) => ready[k] === null)
        expect(nuls, `champ(s) laissés null par la frontière : ${nuls.join(', ')}`).toEqual([])
      })

      it('nourrit les logiques de scène sans lever ni rendre de NaN', () => {
        const doc = normalizeReplayDocument(f.doc)
        const scene = sceneBounds(doc)
        expect(Number.isFinite(scene.maxX - scene.minX)).toBe(true)
        expect(scene.maxX).toBeGreaterThan(scene.minX)
        expect(framesPerSecond(doc)).toBeGreaterThan(0)
        expect(Number.isFinite(frameToMs(1, doc))).toBe(true)
        expect(Number.isFinite(msToFrames(1000, doc))).toBe(true)
      })

      it('nourrit les logiques de piste : chaque vie a une fenêtre et une présence', () => {
        const doc = normalizeReplayDocument(f.doc)
        for (const track of doc.tracks) {
          const w = trackWindow(track)
          expect(Number.isFinite(w.start) && Number.isFinite(w.end)).toBe(true)
          expect(w.end).toBeGreaterThanOrEqual(w.start)
          // Vivant à sa propre première image, et positionné : c'est le minimum qu'un calque
          // attend pour dessiner quoi que ce soit.
          expect(isAliveAt(track, w.start)).toBe(true)
          expect(positionAt(track.points, w.start)).not.toBeNull()
        }
      })

      it('nourrit le roster : des joueurs, des camps, une table de propriété par slot', () => {
        const doc = normalizeReplayDocument(f.doc)
        const players = buildPlayers(doc, [])
        expect(players.length).toBeGreaterThan(0)
        // Le film de référence n'a pas de feuille de match ici : tous les joueurs tombent dans
        // le groupe sans camp, et c'est ce que la logique DOIT faire plutôt que d'en inventer un.
        expect(groupByTeam(players).length).toBeGreaterThan(0)
        expect(buildSlotOwnership(players)).toBeDefined()
        expect(vitalityPresence(doc)).toBeDefined()
        expect(buildSeats(players, null, doc).length).toBeGreaterThan(0)
      })

      it('nourrit les logiques de calque, chacune sans lever', () => {
        const doc = normalizeReplayDocument(f.doc)
        expect(typeof hasAbilityChargeLayer(doc)).toBe('boolean')
        expect(buildEquipmentUsage(doc, undefined)).toBeDefined()
        expect(buildPadControl(doc, undefined)).toBeDefined()
        expect(roundCount(doc.scoreTimeline)).toBeGreaterThanOrEqual(0)
        // Ces deux-là lisent un SLOT et une IMAGE : on prend ceux d'une vie réelle du
        // document, pas des valeurs inventées — un slot absent rendrait `null` sans rien
        // exercer.
        const vie = doc.tracks[0]
        const frame = trackWindow(vie).start
        expect(() => loadoutAt(doc, vie.slot, frame)).not.toThrow()
        expect(() => equippedWeapons(doc, vie.slot, frame)).not.toThrow()
        expect(() => readHillHold(doc, 0, 1, frame)).not.toThrow()
      })
    })
  }
})
