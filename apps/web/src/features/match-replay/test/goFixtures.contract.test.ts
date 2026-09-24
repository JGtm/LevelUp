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
 * depuis les entrées figées) et traverse toute la frontière du web :
 *
 *  1. le CONTRAT à l'exécution (`validateReplayDocument`) ;
 *  2. la NORMALISATION (`normalizeReplayDocument`) ;
 *  3. les LOGIQUES PURES des calques — celles qui prennent le document entier.
 *
 * # UN BUILD DU JEU N'EST PAS L'AUTRE (lot 0.B.7, 2026-09-13)
 *
 * Le film de référence est un film de HI_1_13_0. Tant qu'il était seul, la frontière n'était
 * prouvée que sur UNE génération de jeu — et le chantier a montré tout du long que le film de
 * référence est justement celui sur lequel les défauts ne se voient pas (fermeture d'image-clé
 * de HI_1_4_1 dix fois moindre, origines de pose de `fb1a1a72`...). Ce fichier itère donc
 * TOUTES les fixtures du manifeste, une par build, et nomme le build dans chaque échec.
 *
 * CHAQUE DOCUMENT EST CHARGÉ PUIS RELÂCHÉ. Les huit pèsent ensemble plus de 40 Mio
 * décompressés : les tenir tous ferait de ce test un consommateur de mémoire, pas une preuve.
 * Le `beforeAll` charge, le `afterAll` lâche, et le pic reste à un seul build.
 *
 * # CE QU'IL N'EST PAS
 *
 * Ce n'est pas un test de VALEURS : les chiffres de chaque film sont figés côté Go
 * (`golden_assembly_test.go`, `golden_builds_test.go`), et les redire ici les dédoublerait sans
 * les prouver deux fois. Ce qu'on vérifie est qu'aucune étape ne LÈVE et qu'aucune ne rend un
 * résultat vide là où le document porte manifestement de la matière — c'est-à-dire exactement
 * ce qu'un champ renommé ou un tableau nul casserait.
 */
import { afterAll, beforeAll, describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'
import { validateReplayDocument } from '@/lib/replay/replayDocumentSchema'
import {
  normalizeReplayDocument,
  type ReplayDocumentReady,
} from '@/lib/replay/replayNormalize'
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
import { buildSeats, groupSeatsByTeam, seatOccupantAt, seatTileAt } from '../model/seatLogic'
import { goFixtureEntries, goFixtureManifest, loadGoFixture } from './goFixtures'

const entries = goFixtureEntries()

describe('le jeu de fixtures produites par Go', () => {
  it('existe — sans lui, tous les tests ci-dessous ne garderaient rien', () => {
    expect(entries.length).toBeGreaterThan(0)
  })

  it('porte un document par build du jeu, chacun nommé une seule fois', () => {
    // La table Go publie une fixture par build ET par film : deux films d'un même build sont
    // légitimes (HI_1_13_0 en a deux), deux fois le MÊME fichier ne l'est pas.
    expect(new Set(entries.map((e) => e.file)).size).toBe(entries.length)
    expect(new Set(entries.map((e) => e.build)).size).toBeGreaterThan(1)
  })

  it('garde UNE référence intacte et déclare l’amincissement de toutes les autres', () => {
    // L'ARBITRAGE DU PILOTE, TENU PAR UN TEST (2026-09-13) : le film de référence du chantier
    // reste le document plein — c'est lui qui vaut comparaison avec les goldens de valeurs —
    // et les fixtures par build paient le budget en points de piste. Un jeu où plus rien ne
    // serait intact, ou qui amincirait sans le dire, ne se verrait pas autrement.
    const intactes = entries.filter((e) => e.pointsStride === 1)
    expect(intactes.length, 'une seule fixture doit rester intacte').toBe(1)
    for (const e of entries.filter((x) => x.pointsStride !== 1)) {
      expect(e.pointsStride, `${e.file} n’annonce pas son amincissement`).toBeGreaterThan(1)
    }
  })

  it('déclare la même version dans son manifeste que dans chaque entrée', () => {
    const m = goFixtureManifest()
    for (const e of entries) {
      expect(e.schemaVersion, `${e.file} contredit le manifeste`).toBe(m.schemaVersion)
    }
  })

  it('est TOUT ENTIER au-dessus de la version minimale que le web déclare afficher', () => {
    for (const e of entries) {
      expect(
        e.schemaVersion,
        `${e.file} est sous MIN_RENDERABLE_SCHEMA_VERSION : soit le seuil est faux, ` +
          `soit le web ne sait plus lire ce que Go cuit`,
      ).toBeGreaterThanOrEqual(MIN_RENDERABLE_SCHEMA_VERSION)
    }
  })
})

describe('chaque document produit par Go traverse la frontière du web', () => {
  for (const entry of entries) {
    describe(`${entry.build} / ${entry.film}`, () => {
      let charge: { brut: ReplayDocument; pret: ReplayDocumentReady } | null = null
      const brut = (): ReplayDocument => {
        if (!charge) throw new Error(`${entry.file} non chargé`)
        return charge.brut
      }
      const pret = (): ReplayDocumentReady => {
        if (!charge) throw new Error(`${entry.file} non chargé`)
        return charge.pret
      }

      beforeAll(() => {
        const doc = loadGoFixture(entry).doc
        charge = { brut: doc, pret: normalizeReplayDocument(doc) }
      })
      // LÂCHER LE DOCUMENT, et ce n'est pas du zèle : sans cette ligne les huit builds
      // s'accumulent en mémoire jusqu'à la fin du fichier.
      afterAll(() => {
        charge = null
      })

      it('porte dans son corps la version que le manifeste lui prête', () => {
        expect(brut().schemaVersion).toBe(entry.schemaVersion)
      })

      it('respecte le contrat à l’exécution', () => {
        expect(validateReplayDocument(brut())).toBeNull()
      })

      it('et un calque RENOMMÉ sur ce même document sort en manquement NOMMÉ', () => {
        // LA MUTATION DE LA REVUE RONDE 1 (constat C1), rejouée ici pour de bon. Elle
        // traversait : `z.object` dépouillait la clé inconnue, `shots?` acceptait l'absence,
        // `validateReplayDocument` rendait `null`, le badge disait « à jour » et le calque des
        // tirs se rendait VIDE. C'est le renommage silencieux que le contrat existe pour voir.
        const { shots, ...sansShots } = brut()
        const issue = validateReplayDocument({ ...sansShots, shotz: shots })
        expect(issue).not.toBeNull()
        expect(issue).toEqual({ kind: 'unknownKeys', keys: ['shotz'] })
      })

      it('porte de la matière — sans quoi les contrôles suivants seraient vides de sens', () => {
        expect(brut().tracks?.length ?? 0).toBeGreaterThan(0)
        expect(brut().frameCount).toBeGreaterThan(0)
      })

      it('passe la normalisation sans laisser un seul tableau nul à la racine', () => {
        const ready = pret() as unknown as Record<string, unknown>
        const nuls = Object.keys(ready).filter((k) => ready[k] === null)
        expect(nuls, `champ(s) laissés null par la frontière : ${nuls.join(', ')}`).toEqual([])
      })

      it('nourrit les logiques de scène sans lever ni rendre de NaN', () => {
        const doc = pret()
        const scene = sceneBounds(doc)
        expect(Number.isFinite(scene.maxX - scene.minX)).toBe(true)
        expect(scene.maxX).toBeGreaterThan(scene.minX)
        expect(framesPerSecond(doc)).toBeGreaterThan(0)
        expect(Number.isFinite(frameToMs(1, doc))).toBe(true)
        expect(Number.isFinite(msToFrames(1000, doc))).toBe(true)
      })

      it('nourrit les logiques de piste : chaque vie a une fenêtre et une présence', () => {
        const doc = pret()
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
        const doc = pret()
        const players = buildPlayers(doc, [])
        expect(players.length).toBeGreaterThan(0)
        // Aucune feuille de match n'est donnée ici : tous les joueurs tombent dans le groupe
        // sans camp, et c'est ce que la logique DOIT faire plutôt que d'en inventer un.
        expect(groupByTeam(players).length).toBeGreaterThan(0)
        expect(buildSlotOwnership(players)).toBeDefined()
        expect(vitalityPresence(doc)).toBeDefined()
        expect(buildSeats(players, doc).length).toBeGreaterThan(0)
      })

      it('respecte la RÈGLE DES PLACES : jamais plus de places que la taille d’équipe, jamais un parti', () => {
        verifierLaRegleDesPlaces(pret(), entry.film)
      })

      it('nourrit les logiques de calque, chacune sans lever', () => {
        const doc = pret()
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

/**
 * LA TAILLE D'ÉQUIPE DU MODE de chaque film du jeu de fixtures — une DONNÉE DE TEST, lue sur la
 * liste de lecture du match (Grande équipe : 12 contre 12 ; les autres : 4 contre 4). Elle n'est
 * écrite nulle part dans le code de production : la cuisson l'ESTIME (`coverage.seats`,
 * `repli_place_ouverte_sous_la_capacite_estimee`), et ce test vérifie l'estimation sur les huit
 * builds. Un film ajouté au jeu sans sa taille fait rougir le test, il ne passe pas en silence.
 */
const TAILLE_D_EQUIPE: Readonly<Record<string, number>> = {
  '000d5950': 4,
  '60ae07c4': 4,
  bcb6d393: 4,
  fb1a1a72: 4,
  a521164d: 12,
  '11de8353': 12,
  '111fa685': 12,
  e5adf7b2: 12,
}

/** Le pas d'échantillonnage de la propriété : une image par seconde de film (10 à 100 ms). */
const PAS_DE_LA_PROPRIETE = 10

/**
 * verifierLaRegleDesPlaces — LA PROPRIÉTÉ du lot M2.4 sur un document produit par Go (règle des
 * places de l'utilisateur, 2026-09-23), mesurée sur les TUILES QUE LA COLONNE REND (`seatTileAt`,
 * la fonction même de `ReplayTeams`) et contre des sources que la pose des places ne produit pas
 * (revue M2-R3, 2026-09-24) :
 *
 *	la TAILLE D'ÉQUIPE déclarée ci-dessus : à chaque image, une colonne rend au plus autant de
 *	  tuiles (vides comprises) que l'équipe a de places ;
 *	l'UNICITÉ d'un joueur : à une image, un joueur n'est rendu que dans UNE tuile, toutes colonnes
 *	  confondues — un parti ne survit pas sous une autre place ;
 *	un joueur sans entrée de roster ne rend AUCUNE tuile ;
 *	les compteurs de la cuisson, qui mesurent contre la CAPACITÉ (et non contre les places posées) :
 *	  `depassements`, `placesEnTrop` et `identitesHorsRoster` valent 0.
 *
 * Le départ, lui, se prouve contre les ENTITÉS `ti=9` du film, côté Go, sur les huit mêmes builds
 * (`TestRegleDesPlacesContreLesEntitesDesBuilds`) : le document ne les porte pas.
 */
function verifierLaRegleDesPlaces(doc: ReplayDocumentReady, film: string): void {
  const taille = TAILLE_D_EQUIPE[film]
  expect(taille, `${film} : taille d’équipe du mode non déclarée dans TAILLE_D_EQUIPE`).toBeDefined()
  const seats = doc.coverage?.seats
  expect(seats?.depassements, `${film} : dépassements de capacité`).toBe(0)
  expect(seats?.placesEnTrop, `${film} : places au-delà de la capacité`).toBe(0)
  expect(seats?.identitesHorsRoster, `${film} : identités hors roster`).toBe(0)
  const groupes = groupSeatsByTeam(buildSeats(buildPlayers(doc, []), doc))
  for (let f = 0; f < doc.frameCount; f += PAS_DE_LA_PROPRIETE) {
    const rendus = new Set<string>()
    for (const g of groupes) {
      let tuiles = 0
      for (const s of g.seats) {
        const lu = seatTileAt(s, f)
        if (lu === null) continue
        tuiles++
        if (lu.player === null) continue
        expect(rendus.has(lu.player.xuid), `${film} : ${lu.player.xuid} rendu deux fois à l’image ${f}`).toBe(false)
        rendus.add(lu.player.xuid)
        expect(s.horsRoster, `${film} : ${lu.player.xuid} hors roster rendu (${f})`).toBe(false)
      }
      expect(tuiles, `${film} : ${tuiles} tuiles dans un camp à l’image ${f}`).toBeLessThanOrEqual(taille)
    }
  }
  for (const g of groupes) {
    for (const s of g.seats) {
      for (let k = 0; k + 1 < s.occupants.length; k++) {
        const avant = s.occupants[k].presence.at(-1)!
        const apres = s.occupants[k + 1].presence[0]
        if (avant.toMax + 1 < apres.from) {
          expect(seatOccupantAt(s, avant.toMax + 1).kind, `${film} : relais de la place ${s.seat}`).toBe('vide')
        }
      }
    }
  }
}
