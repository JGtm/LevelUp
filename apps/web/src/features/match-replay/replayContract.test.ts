/**
 * replayContract.test.ts — LA FRONTIÈRE DE NULLABILITÉ, PROUVÉE COMPLÈTE PAR LE TYPAGE.
 *
 * CE QUE CE FICHIER FERME, ET IL A DÉJÀ COÛTÉ UNE FOIS. Depuis que le document de rejeu vient
 * du contrat généré, ses tableaux sont nullables : un slice Go nil se sérialise en `null`, et le
 * schéma le dit. Le jour où c'est arrivé, 31 erreurs de type sont tombées d'un coup, et la
 * réparation a consisté à poser UN point de passage — `normalizeReplayDocument` — plutôt que de
 * semer des `?? []` dans chaque appelant.
 *
 * MAIS CE POINT DE PASSAGE ÉNUMÈRE SES CHAMPS À LA MAIN. Le jour où le Go publiera un tableau
 * de plus, rien ne le rappellerait : le champ traverserait la frontière tel quel, `null`, et le
 * rendu tomberait à l'exécution — pas à la compilation, puisque le type `*Ready` est construit
 * depuis la même liste manuelle.
 *
 * D'OÙ LA FORME DE CE TEST : la complétude est vérifiée PAR LE COMPILATEUR, contre les types
 * GÉNÉRÉS depuis `api/openapi.yaml`. Deux assertions de type suffisent, et elles font tomber
 * `tsc -b` (donc la CI) avant même qu'un test ne s'exécute :
 *
 *   1. la liste des tableaux nullables du contrat est EXACTEMENT celle qu'on énumère ici ;
 *   2. le document normalisé n'en porte plus AUCUN.
 *
 * Un champ ajouté côté Go arrive donc ici sans que personne n'ait à y penser. Le reste du
 * fichier vérifie le COMPORTEMENT de la frontière — ce qu'un type ne peut pas dire.
 *
 * POURQUOI PAS LIRE `openapi.yaml` DIRECTEMENT. C'était la première forme écrite, et elle
 * marchait ; elle exigeait `js-yaml`, qui n'est présent qu'en `overrides` (dépendance
 * transitive de `openapi-typescript`). S'appuyer dessus, c'est dépendre de ce qu'un autre
 * paquet installe — knip le signale à juste titre. Les types générés viennent du MÊME contrat,
 * et ils sont, eux, une dépendance déclarée du dossier.
 */
import { describe, expect, it } from 'vitest'

import { normalizeReplayDocument } from './replayNormalize'

import type { ReplayDocument } from '@/lib/api/types'
import type { ReplayDocumentReady } from './replayNormalize'

/** Égalité STRICTE de deux types (le double conditionnel différé est ce qui la rend stricte). */
type Equals<A, B> = (<T>() => T extends A ? 1 : 2) extends <T>() => T extends B ? 1 : 2
  ? true
  : false

type Expect<T extends true> = T

/** Les clés de `T` dont la valeur est un TABLEAU que le transport a le droit de laisser `null`. */
type NullableArrayKeys<T> = {
  [K in keyof T]-?: null extends T[K]
    ? NonNullable<T[K]> extends readonly unknown[]
      ? K
      : never
    : never
}[keyof T]

/**
 * NULLABLE_ARRAYS — les tableaux nullables du contrat, énumérés pour les tests d'exécution.
 *
 * CETTE LISTE N'EST PAS UNE DÉCLARATION DE FOI : l'assertion `_ListeExhaustive` ci-dessous la
 * confronte au contrat généré. Si le Go publie un tableau de plus, l'égalité de types échoue et
 * `tsc -b` refuse de compiler — avec, en clair, le nom du champ manquant.
 */
const NULLABLE_ARRAYS = [
  // `abilities` : le calque des lectures de capacité (schéma 6, 2026-08-14). Il remplace le
  // champ `Inventory.a` RETIRÉ le même jour — celui-ci portait `rang − 16` (canal d'image-clé
  // borgne), le calque porte le RANG. Deux grandeurs différentes : le champ a été retiré
  // plutôt que réinterprété, et ce garde-fou a fait son travail en refusant le nouveau
  // tableau tant qu'il n'était pas déclaré ici.
  'abilities',
  // `equipmentEpisodes` : l'état ACTIF du camouflage et du surbouclier (schéma 7,
  // 2026-08-16), en épisodes datés par vie. Deux familles seulement — les deux mesurées.
  'equipmentEpisodes',
  // `equipmentPlacements` : les POSES d'équipement sur la carte (schéma 9, 2026-08-18) —
  // mur et capteur nommés par le manifeste du titre, tout le reste publié en famille
  // `other` avec son identifiant de tag. Chaque pose porte son poseur MESURÉ (-1 quand
  // aucun bipède contemporain n'est assez proche) et le cap de visée de ce poseur.
  'equipmentPlacements',
  'geometry',
  // `grappleLines` : les tractions de grappin (schéma 8, 2026-08-16) — fenêtre mesurée
  // [t0, t1] par vie + point d'accroche en coordonnées monde.
  'grappleLines',
  'grenadeLabels',
  'grenades',
  'inventory',
  'loadouts',
  'neutralDeaths',
  'objectives',
  // `padPickups` : les occupations de socle ACHEVÉES (schéma 11, 2026-08-17) — le socle s'est
  // vidé quelque part dans [tLow, tHigh]. Un INTERVALLE, pas un instant : le film ne porte aucun
  // événement de ramassage, et la seule preuve de disparition est le recensement des
  // images-clés, espacé de ~20 s. `xuid` vaut TOUJOURS `null` (oracle à 79,7 %, seuil 90 %).
  'padPickups',
  'projectiles',
  'roster',
  'shots',
  'structure',
  'tracks',
  // `weaponPads` : les SOCLES D'ARME du match (schéma 11, 2026-08-17) — position, famille,
  // apparitions, intervalles de présence bornés par les images-clés, et cycle de réapparition
  // SEULEMENT s'il est établi. Par MATCH et non par carte : sur deux films de la même carte les
  // socles tombent aux mêmes coordonnées au centimètre, mais l'arme qui y apparaît change.
  'weaponPads',
] as const

/** (1) La liste couvre EXACTEMENT les tableaux nullables du contrat — ni plus, ni moins. */
type _ListeExhaustive = Expect<
  Equals<(typeof NULLABLE_ARRAYS)[number], NullableArrayKeys<ReplayDocument>>
>

/** (2) Le document normalisé n'en porte plus aucun : la frontière les a TOUS comblés. */
type _FrontiereComplete = Expect<Equals<NullableArrayKeys<ReplayDocumentReady>, never>>

describe('la frontière du document de rejeu', () => {
  it('énumère EXACTEMENT les tableaux nullables du contrat — vérifié à la compilation', () => {
    const exhaustive: _ListeExhaustive = true
    expect(exhaustive).toBe(true)
    expect(NULLABLE_ARRAYS.length).toBeGreaterThan(0)
  })

  it('ne laisse AUCUN tableau nullable au rendu — vérifié à la compilation', () => {
    const complete: _FrontiereComplete = true
    expect(complete).toBe(true)
  })

  it('comble un document HOSTILE, où tout ce qui peut être null l’est', () => {
    const hostile = Object.fromEntries(
      NULLABLE_ARRAYS.map((k) => [k, null]),
    ) as unknown as ReplayDocument
    const ready = normalizeReplayDocument(hostile) as unknown as Record<string, unknown>
    const oublis = NULLABLE_ARRAYS.filter((k) => !Array.isArray(ready[k]))
    expect(
      oublis,
      `champ(s) que la frontière ne comble pas : ${oublis.join(', ')}. ` +
        `À ajouter dans normalizeReplayDocument, jamais par un « ?? [] » chez l’appelant.`,
    ).toEqual([])
  })

  it('traite ABSENT comme NULL : pour le rendu, les deux disent « aucune donnée »', () => {
    const ready = normalizeReplayDocument({} as ReplayDocument) as unknown as Record<string, unknown>
    for (const k of NULLABLE_ARRAYS) {
      expect(Array.isArray(ready[k]), `champ ${k}`).toBe(true)
    }
  })

  it('n’invente aucune valeur : un tableau absent devient VIDE, jamais peuplé', () => {
    const ready = normalizeReplayDocument({} as ReplayDocument) as unknown as Record<
      string,
      unknown[]
    >
    for (const k of NULLABLE_ARRAYS) {
      expect(ready[k], `champ ${k}`).toHaveLength(0)
    }
  })

  it('comble aussi les tableaux IMBRIQUÉS, que la liste de tête ne voit pas', () => {
    // CE QUE `_ListeExhaustive` NE PEUT PAS DIRE. L'assertion de type n'énumère que les tableaux
    // nullables de la RACINE : elle exige bien `weaponPads`, `tracks`, `inventory`… mais elle est
    // aveugle à ceux qui vivent DANS leurs éléments. Un socle sans apparition répliquée arrivait
    // ainsi au rendu avec `spawns: null`, et `pad.spawns.map` tombait à l'exécution — c'est le
    // correctif de revue du 2026-08-17, et ce test est ce qui le retient.
    const raw = {
      weaponPads: [{ x: 1, y: 2, weapon: '0x0000ffff' }],
      tracks: [{ slot: 1, team: -1 }],
      inventory: [{ t: 0, slot: 1 }],
      loadouts: [{ t: 0, slot: 1 }],
    } as unknown as ReplayDocument
    const ready = normalizeReplayDocument(raw)
    expect(ready.weaponPads[0].spawns, 'weaponPads[].spawns').toEqual([])
    expect(ready.weaponPads[0].presence, 'weaponPads[].presence').toEqual([])
    expect(ready.tracks[0].points, 'tracks[].points').toEqual([])
    expect(ready.inventory[0].am, 'inventory[].am').toEqual([])
    expect(ready.inventory[0].g, 'inventory[].g').toEqual([])
    expect(ready.loadouts[0].w, 'loadouts[].w').toEqual([])
    // `cycle` N'EST PAS un tableau : une mesure absente reste absente, jamais un objet vide qui
    // se lirait comme un cycle de zéro seconde.
    expect(ready.weaponPads[0].cycle ?? null).toBeNull()
  })

  it('conserve ce que le socle porte quand il le porte', () => {
    const raw = {
      weaponPads: [
        {
          x: 1,
          y: 2,
          weapon: '0x0000ffff',
          spawns: [10, 200],
          presence: [{ t0: 10, tLow: 200, tHigh: 400 }],
          cycle: { medianS: 30.5, p10S: 30.2, p90S: 30.8, gaps: 2, missing: 1 },
        },
      ],
    } as unknown as ReplayDocument
    const ready = normalizeReplayDocument(raw)
    expect(ready.weaponPads[0].spawns).toEqual([10, 200])
    expect(ready.weaponPads[0].presence).toHaveLength(1)
    // `missing` dit combien d'écarts le socle offrait sans qu'on ait pu les mesurer : sans lui,
    // « 2 écarts » se lit comme « 2 sur 2 » alors que la mesure en a perdu un.
    expect(ready.weaponPads[0].cycle?.gaps).toBe(2)
    expect(ready.weaponPads[0].cycle?.missing).toBe(1)
  })

  it('recopie EN SURFACE : les points de trajectoire ne sont jamais dupliqués', () => {
    // Ce sont eux qui font le poids du document (29 221 points sur le film de référence) ;
    // les recopier à chaque normalisation coûterait à chaque chargement, pour rien.
    const points = [{ t: 0, x: 1, y: 2 }]
    const raw = { tracks: [{ slot: 1, team: -1, points }] } as unknown as ReplayDocument
    expect(normalizeReplayDocument(raw).tracks[0].points).toBe(points)
  })

  it('rétablit l’arité que JSON Schema ne sait pas rendre en TypeScript', () => {
    // `Surface.poly` est un [2]float32 côté Go, `Projectile.p` un [3]float32, et le contrat le
    // DIT (minItems = maxItems — vérifié côté Go par contracttest/replay_contract_test.go).
    // C'est le générateur de types qui retombe sur `number[]` ; la frontière rétablit l'arité
    // une seule fois, ici, plutôt qu'un cast par appelant.
    const raw = {
      structure: [{ x0: 0, y0: 0, x1: 1, y1: 1, z: 0, zb: 0, poly: [[1, 2]] }],
      projectiles: [{ t0: 0, p: [[0, 1, 2]] }],
    } as unknown as ReplayDocument
    const ready = normalizeReplayDocument(raw)
    expect(ready.structure[0].poly[0]).toHaveLength(2)
    expect(ready.projectiles[0].p[0]).toHaveLength(3)
    // Et l'absence reste une absence : jamais un tuple fabriqué.
    const vide = normalizeReplayDocument({
      structure: [{ x0: 0, y0: 0, x1: 1, y1: 1, z: 0, zb: 0 }],
    } as unknown as ReplayDocument)
    expect(vide.structure[0].poly).toEqual([])
  })
})
