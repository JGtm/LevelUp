/**
 * replayDocumentSchema.ts — LE CONTRAT DU DOCUMENT DE REJEU, À L'EXÉCUTION.
 *
 * POURQUOI CE FICHIER EXISTE (2026-09-13, lot 0.B ; architecture §12, garde-rail 3). Le web
 * type le document depuis `api/openapi.yaml` — et ces types N'EXISTENT PLUS à l'exécution. Un
 * champ renommé côté cuisson, ou dont le type change, ne fait donc rougir NI `tsc` (le contrat
 * généré aurait suivi) NI aucun test : il arrive en production, et le rendu tombe sur
 * `undefined.map`. C'est le trou que ce schéma ferme, à la frontière de transport.
 *
 * # CE QU'IL VALIDE — LA RACINE, ET EXACTEMENT
 *
 * La racine est un objet STRICT : les 57 clés du contrat, ni plus ni moins. Trois manquements
 * en sortent nommés :
 *
 *  - une clé REQUISE absente (`matchId`, `bounds`, `frameCount`, `schemaVersion`, `titleSlug`,
 *    `tracks`) ;
 *  - une clé de NATURE changée (un calque devenu objet, une borne devenue chaîne) ;
 *  - une clé INCONNUE — c'est-à-dire un champ RENOMMÉ (`shots` devenu `shotz`), ou un
 *    producteur en avance sur les types générés du web.
 *
 * LE TROISIÈME CAS EST LA RAISON DU MODE STRICT, et il a coûté une revue (ronde 1 du lot 0.B,
 * constat C1) : `z.object` DÉPOUILLE les clés inconnues et `.optional()` accepte l'absence —
 * si bien qu'un calque renommé passait le contrat en silence, le badge disait « à jour », et le
 * calque se rendait vide. Le mode strict ne bloque jamais le rendu : le manquement voyage
 * jusqu'au badge admin, le document est servi tel quel (cf. `validateReplayDocument`).
 *
 * # CE QU'IL NE VALIDE PAS, ET C'EST ÉCRIT ICI POUR QU'ON NE S'Y TROMPE PAS
 *
 * Il NE DESCEND PAS dans les éléments des calques ni dans les blocs : `z.custom<T>()` porte le
 * TYPE (donc `tsc` voit la forme complète) sans re-valider chaque point de trajectoire. Un
 * champ renommé À L'INTÉRIEUR d'un `Shot`, d'un `Track` ou de `coverage` n'est donc PAS attrapé
 * ici. Le motif est délibéré et son coût est mesuré : un document réel porte des centaines de
 * milliers de points, et les parcourir un à un à chaque chargement coûterait plus cher que le
 * décodage JSON lui-même. La forme PROFONDE a son propre gardien, côté producteur : l'empreinte
 * réfléchie de `ReplayDocument` (`replay/document_shape_test.go`), qui refuse un changement de
 * forme sans montée de `SchemaVersion`.
 *
 * Les TABLES de libellés (`weaponLabels`, `abilityLabels`, `vehicleLabels`, `killEffects`) ne
 * sont pas strictes non plus, et ne peuvent pas l'être : leurs clés SONT la donnée (un
 * identifiant d'arme par entrée). Seule la nature de leur valeur y est tenue.
 *
 * # CE QUI EMPÊCHE CE SCHÉMA DE DÉRIVER DU CONTRAT
 *
 * Les types des éléments sont DÉDUITS de `ReplayDocument` (cf. `Elem` / `Valeur` ci-dessous) :
 * il n'y a aucun nom de type à tenir en phase. Ce qui est écrit à la main, c'est le NOM de
 * chaque clé et sa NATURE — et les deux sont confrontés au contrat généré par des assertions
 * de type dans `replayDocumentSchema.test.ts`, qui font tomber `tsc -b` avant tout test.
 */
import { z } from 'zod'

import type { ReplayDocument } from '@/lib/api/types'

/** Le type d'un ÉLÉMENT du tableau porté par la clé `K` du contrat. */
type Elem<K extends keyof ReplayDocument> =
  NonNullable<ReplayDocument[K]> extends readonly (infer E)[] ? E : never

/** Le type d'une VALEUR de la table portée par la clé `K` du contrat. */
type Valeur<K extends keyof ReplayDocument> =
  NonNullable<ReplayDocument[K]> extends Record<string, infer V> ? V : never

/**
 * Un tableau FACULTATIF et NULLABLE : `E[] | null | undefined`. C'est la forme de tout slice Go
 * sous `omitempty` — la vingtaine de calques du document.
 */
function calque<E>() {
  return z.array(z.custom<E>()).nullish()
}

/** Un objet FACULTATIF : présent, il doit être un objet ; absent, c'est un silence légitime. */
function bloc<T>() {
  return z.custom<T>((v) => typeof v === 'object' && v !== null).optional()
}

/** Une table FACULTATIVE clé -> valeur (les catalogues de libellés ; cf. en-tête). */
function table<V>() {
  return z.record(z.string(), z.custom<V>()).optional()
}

/**
 * Les bornes du monde. Validées POUR DE VRAI (et pas par `z.custom`) parce que tout le rendu
 * en dépend : une borne manquante ou non numérique fait une scène de taille NaN, c'est-à-dire
 * une page blanche sans message. STRICTES pour la même raison que la racine : une borne
 * renommée est une borne absente, et le silence est le pire des deux.
 */
const bornes = z.strictObject({
  maxX: z.number(),
  maxY: z.number(),
  maxZ: z.number().optional(),
  minX: z.number(),
  minY: z.number(),
  minZ: z.number().optional(),
})

/**
 * replayDocumentSchema — les 57 clés de la racine, dans l'ordre du contrat généré.
 *
 * L'ORDRE EST CELUI DU CONTRAT, et ce n'est pas de la coquetterie : c'est ce qui rend la
 * confrontation lisible en revue quand le contrat gagne une clé.
 *
 * STRICT : toute clé hors de cette liste est un manquement nommé (cf. en-tête).
 */
export const replayDocumentSchema = z.strictObject({
  abilities: calque<Elem<'abilities'>>(),
  abilityCharges: calque<Elem<'abilityCharges'>>(),
  abilityImpulses: calque<Elem<'abilityImpulses'>>(),
  abilityLabels: table<Valeur<'abilityLabels'>>(),
  bombArmings: calque<Elem<'bombArmings'>>(),
  bombCarries: calque<Elem<'bombCarries'>>(),
  bombEvents: calque<Elem<'bombEvents'>>(),
  bombStats: bloc<NonNullable<ReplayDocument['bombStats']>>(),
  bounds: bornes,
  coverage: bloc<NonNullable<ReplayDocument['coverage']>>(),
  durationMs: z.number().optional(),
  equipmentChanges: calque<Elem<'equipmentChanges'>>(),
  equipmentEpisodes: calque<Elem<'equipmentEpisodes'>>(),
  equipmentPlacements: calque<Elem<'equipmentPlacements'>>(),
  flagCarries: calque<Elem<'flagCarries'>>(),
  flagReturnZone: bloc<NonNullable<ReplayDocument['flagReturnZone']>>(),
  frameCount: z.number(),
  frameIntervalMs: z.number().optional(),
  geometry: calque<Elem<'geometry'>>(),
  geometryBounds: bornes.optional(),
  grappleLines: calque<Elem<'grappleLines'>>(),
  grenadeLabels: calque<Elem<'grenadeLabels'>>(),
  grenadeReads: calque<Elem<'grenadeReads'>>(),
  grenades: calque<Elem<'grenades'>>(),
  groundWeapons: calque<Elem<'groundWeapons'>>(),
  identity: bloc<NonNullable<ReplayDocument['identity']>>(),
  inventory: calque<Elem<'inventory'>>(),
  killEffects: z.record(z.string(), z.string()).optional(),
  loadouts: calque<Elem<'loadouts'>>(),
  mapObjectives: bloc<NonNullable<ReplayDocument['mapObjectives']>>(),
  mapWeaponPads: bloc<NonNullable<ReplayDocument['mapWeaponPads']>>(),
  matchId: z.string(),
  neutralDeaths: calque<Elem<'neutralDeaths'>>(),
  objectiveObjects: calque<Elem<'objectiveObjects'>>(),
  objectives: calque<Elem<'objectives'>>(),
  originMs: z.number().optional(),
  padPickups: calque<Elem<'padPickups'>>(),
  pickups: calque<Elem<'pickups'>>(),
  projectiles: calque<Elem<'projectiles'>>(),
  roster: calque<Elem<'roster'>>(),
  schemaVersion: z.number(),
  scoreTimeline: bloc<NonNullable<ReplayDocument['scoreTimeline']>>(),
  shots: calque<Elem<'shots'>>(),
  skullCarries: calque<Elem<'skullCarries'>>(),
  structure: calque<Elem<'structure'>>(),
  structureBounds: bornes.optional(),
  t0FilmMs: z.number().optional(),
  titleSlug: z.string(),
  // `tracks` est le SEUL tableau que le contrat déclare REQUIS (nullable, mais présent) : sans
  // trajectoire il n'y a pas de rejeu, et la cuisson refuse déjà un document qui n'en porte
  // aucune (`validateArtifact`, côté Go).
  tracks: z.array(z.custom<Elem<'tracks'>>()).nullable(),
  translocations: calque<Elem<'translocations'>>(),
  vehicleLabels: table<Valeur<'vehicleLabels'>>(),
  vehicles: calque<Elem<'vehicles'>>(),
  vipCrown: calque<Elem<'vipCrown'>>(),
  weaponChanges: calque<Elem<'weaponChanges'>>(),
  weaponLabels: table<Valeur<'weaponLabels'>>(),
  weaponPads: calque<Elem<'weaponPads'>>(),
  zoneStates: calque<Elem<'zoneStates'>>(),
})

/** Le document tel que le schéma le décrit — confronté au contrat généré dans son test. */
export type ReplayDocumentFromSchema = z.infer<typeof replayDocumentSchema>

/**
 * ReplayContractIssue — LE MANQUEMENT, EN DONNÉES ET NON EN PHRASE.
 *
 * POURQUOI CE TYPE EXISTE (2026-09-13, ronde 2 de la revue, constat R2-2). Cette fonction
 * rendait une PHRASE, et cette phrase était française : en locale anglaise, le badge affichait
 * `contract violated (cle(s) inconnue(s) : shotz)` — un fragment FR écrit hors d'`i18n.ts`,
 * inséré au milieu d'une phrase EN. La règle n° 1 du dépôt veut toute chaîne d'interface en FR
 * ET en EN, tenue par le typage de parité ; une phrase fabriquée ici ne peut pas l'être.
 *
 * Ce module rend donc ce qu'il SAIT (quel genre de manquement, quelle clé, quel chemin) et
 * laisse l'interface le DIRE (`REPLAY_TEXT[locale].contract*`, cf. `ReplaySchemaBadge`).
 *
 * `detail` reste le texte de zod, en anglais et non traduit : c'est un diagnostic technique
 * destiné à un administrateur (« expected number, received string »), pas une phrase de
 * produit. Le traduire exigerait de réécrire la table d'erreurs de la bibliothèque.
 */
export type ReplayContractIssue =
  | { kind: 'unknownKeys'; keys: string[] }
  | { kind: 'invalidField'; path: string; detail: string }
  | { kind: 'malformed' }

/**
 * validateReplayDocument rend `null` si le document respecte le contrat, sinon le PREMIER
 * manquement, en données.
 *
 * IL NE LÈVE JAMAIS, ET NE JETTE JAMAIS LE DOCUMENT. Un contrat violé est un signal pour
 * l'exploitant (le badge admin le montre), pas une raison de refuser d'afficher : le rendu
 * dégrade déjà champ par champ, et une page blanche apprendrait moins qu'un rejeu incomplet.
 *
 * LA CLÉ INCONNUE EST PORTÉE À PART : zod la range dans `issue.keys` et non dans le chemin, si
 * bien qu'un manquement construit sur le seul chemin dirait « racine » sans jamais dire QUOI —
 * c'est-à-dire sans donner le nom du champ renommé, la seule information utile de ce cas.
 */
export function validateReplayDocument(raw: unknown): ReplayContractIssue | null {
  const r = replayDocumentSchema.safeParse(raw)
  if (r.success) return null
  const premier = r.error.issues[0]
  if (!premier) return { kind: 'malformed' }
  if (premier.code === 'unrecognized_keys') {
    return { kind: 'unknownKeys', keys: [...premier.keys] }
  }
  const chemin = premier.path.join('.')
  return chemin
    ? { kind: 'invalidField', path: chemin, detail: premier.message }
    : { kind: 'malformed' }
}
