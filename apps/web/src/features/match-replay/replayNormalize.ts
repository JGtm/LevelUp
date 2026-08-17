/**
 * replayNormalize.ts — LA FRONTIÈRE du document de rejeu.
 *
 * POURQUOI CE FICHIER EXISTE. Depuis que `ReplayDocument` vient du contrat généré et non
 * plus d'une copie écrite à la main, ses tableaux sont nullables : un slice Go nil se
 * sérialise en `null`, et le schéma le dit. C'est la vérité du transport, pas celle du
 * rendu — pour tout ce qui dessine, « aucune trace » et « le champ vaut null » sont la
 * même chose. Sans point de passage, cette différence se paie en `?.` et `?? []` semés
 * dans chaque appelant, et il en manque toujours un.
 *
 * Le document est donc normalisé UNE FOIS, dans la queryFn (cf. queries.ts), et tout le
 * dossier `match-replay/` ne manipule que les types `*Ready` : aucun tableau n'y est null.
 *
 * CE QUE LA NORMALISATION RÉPARE AUSSI — les longueurs fixes. Le Go écrit
 * `Poly [][2]float32` et `P [][3]float32` : des sommets XY et des pas [dt, x, y]. JSON
 * Schema ne sait pas exprimer un tuple de longueur fixe, le contrat généré rend donc
 * `number[]`. La donnée, elle, EST une paire ou un triplet — c'est le type Go qui le
 * garantit, pas une supposition d'ici. Le rétablir tient en une assertion posée à cet
 * endroit unique, plutôt qu'en un cast par appelant.
 */
import type {
  ReplayDocument,
  ReplayInventory,
  ReplayLoadout,
  ReplayProjectile,
  ReplaySurface,
  ReplayTrack,
} from '@/lib/api/types'

/** Un sommet d'emprise orientée : le `[2]float32` du Go. */
type ReplayXY = [number, number]

/** Un pas de trajectoire de projectile : le `[3]float32` du Go, soit [dt, x, y]. */
type ReplayStep = [number, number, number]

/** Rend NON NULLABLES (et présents) les champs `K` de `T` — les tableaux du contrat. */
type Filled<T, K extends keyof T> = Omit<T, K> & { [P in K]-?: NonNullable<T[P]> }

export type ReplayTrackReady = Filled<ReplayTrack, 'points'>
type ReplayLoadoutReady = Filled<ReplayLoadout, 'w'>
export type ReplayInventoryReady = Filled<ReplayInventory, 'am' | 'g'>
export type ReplaySurfaceReady = Omit<ReplaySurface, 'poly'> & { poly: ReplayXY[] }
export type ReplayProjectileReady = Omit<ReplayProjectile, 'p'> & { p: ReplayStep[] }

/**
 * ReplayDocumentReady — le document tel que le rendu a le droit de le lire : chaque
 * tableau est présent, jamais null, et les coordonnées ont retrouvé leur arité.
 */
export type ReplayDocumentReady = Omit<
  ReplayDocument,
  | 'abilities'
  | 'equipmentEpisodes'
  | 'equipmentPlacements'
  | 'geometry'
  | 'grappleLines'
  | 'grenadeLabels'
  | 'grenades'
  | 'inventory'
  | 'loadouts'
  | 'neutralDeaths'
  | 'objectives'
  | 'projectiles'
  | 'roster'
  | 'shots'
  | 'structure'
  | 'tracks'
> & {
  abilities: NonNullable<ReplayDocument['abilities']>
  equipmentEpisodes: NonNullable<ReplayDocument['equipmentEpisodes']>
  equipmentPlacements: NonNullable<ReplayDocument['equipmentPlacements']>
  geometry: NonNullable<ReplayDocument['geometry']>
  grappleLines: NonNullable<ReplayDocument['grappleLines']>
  grenadeLabels: NonNullable<ReplayDocument['grenadeLabels']>
  grenades: NonNullable<ReplayDocument['grenades']>
  inventory: ReplayInventoryReady[]
  loadouts: ReplayLoadoutReady[]
  neutralDeaths: NonNullable<ReplayDocument['neutralDeaths']>
  objectives: NonNullable<ReplayDocument['objectives']>
  projectiles: ReplayProjectileReady[]
  roster: NonNullable<ReplayDocument['roster']>
  shots: NonNullable<ReplayDocument['shots']>
  structure: ReplaySurfaceReady[]
  tracks: ReplayTrackReady[]
}

/**
 * normalizeReplayDocument comble les tableaux absents et rétablit l'arité des coordonnées.
 *
 * Aucune valeur n'est inventée : un tableau null ou absent devient vide, ce qui est
 * exactement ce que le producteur voulait dire. Les objets ne sont recopiés qu'en surface
 * — les points de trajectoire, qui font le poids du document, ne sont jamais dupliqués.
 */
export function normalizeReplayDocument(raw: ReplayDocument): ReplayDocumentReady {
  return {
    ...raw,
    // Le calque des lectures de CAPACITÉ (schéma 6). Il remplace `Inventory.a`, retiré le
    // même jour : celui-ci portait `rang − 16` (le canal d'image-clé ne voit que 16..23),
    // ce calque porte le RANG complet, et chaque lecture dit par quel canal elle est venue.
    // Absent = aucune lecture, la fiche montre l'inventaire sans capacité nommée.
    abilities: raw.abilities ?? [],
    // Les épisodes d'ÉTAT ACTIF d'équipement (schéma 7) : camouflage et surbouclier,
    // datés par vie — les deux seules familles dont l'état est MESURÉ. Absent = aucune
    // vie publiée n'en porte : les fiches restent sobres, jamais un effet deviné.
    equipmentEpisodes: raw.equipmentEpisodes ?? [],
    // Les POSES d'équipement (schéma 9) : mur, capteur, et les objets du monde qui
    // partagent l'archétype — ces derniers publiés en famille `other`, avec leur
    // identifiant de tag et sans nom. Absent = le film n'en porte aucune, OU sa largeur
    // de bloc de réplication n'a pas été tranchée : `coverage.placements.calibrated`
    // distingue les deux, et c'est pour cela qu'il est publié.
    equipmentPlacements: raw.equipmentPlacements ?? [],
    geometry: raw.geometry ?? [],
    // Les TRACTIONS de grappin (schéma 8) : fenêtre mesurée [t0, t1] par vie + point
    // d'accroche en coordonnées monde. Absent = aucune traction lue sur ce film : rien
    // ne se trace, jamais une ligne devinée.
    grappleLines: raw.grappleLines ?? [],
    grenadeLabels: raw.grenadeLabels ?? [],
    grenades: raw.grenades ?? [],
    inventory: (raw.inventory ?? []).map((inv) => ({ ...inv, am: inv.am ?? [], g: inv.g ?? [] })),
    loadouts: (raw.loadouts ?? []).map((lo) => ({ ...lo, w: lo.w ?? [] })),
    // Le TYPE des morts que personne ne revendique (chute, hors-limites, sa propre arme) :
    // le fil déduit ces lignes de ses pistes, cette table dit seulement DE QUOI le joueur
    // est mort. Absente = aucune n'est établie, le fil garde son repère neutre.
    neutralDeaths: raw.neutralDeaths ?? [],
    // Le calque d'actions d'objectif traverse la frontière comme les autres tableaux ;
    // il nourrit les PULSES du canvas (objectivesLayer.buildObjectivePulses, lot 4.4).
    objectives: raw.objectives ?? [],
    // `mapObjectives` (objectifs STATIQUES du mode, servis à la requête) passe par
    // `...raw` : c'est un objet optionnel, pas un tableau — sa normalisation vit à
    // l'entrée du calque (normalizeMapObjectives), comme celle des callouts.
    // `as` sur l'arité seule : le contenu est celui du contrat, seule la longueur fixe du
    // tuple que JSON Schema ne sait pas dire est réaffirmée (cf. en-tête).
    projectiles: (raw.projectiles ?? []).map((pr) => ({ ...pr, p: (pr.p ?? []) as ReplayStep[] })),
    roster: raw.roster ?? [],
    shots: raw.shots ?? [],
    structure: (raw.structure ?? []).map((s) => ({ ...s, poly: (s.poly ?? []) as ReplayXY[] })),
    tracks: (raw.tracks ?? []).map((t) => ({ ...t, points: t.points ?? [] })),
  }
}
