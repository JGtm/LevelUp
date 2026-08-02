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
  | 'geometry'
  | 'grenadeLabels'
  | 'grenades'
  | 'inventory'
  | 'loadouts'
  | 'projectiles'
  | 'roster'
  | 'shots'
  | 'structure'
  | 'tracks'
> & {
  geometry: NonNullable<ReplayDocument['geometry']>
  grenadeLabels: NonNullable<ReplayDocument['grenadeLabels']>
  grenades: NonNullable<ReplayDocument['grenades']>
  inventory: ReplayInventoryReady[]
  loadouts: ReplayLoadoutReady[]
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
    geometry: raw.geometry ?? [],
    grenadeLabels: raw.grenadeLabels ?? [],
    grenades: raw.grenades ?? [],
    inventory: (raw.inventory ?? []).map((inv) => ({ ...inv, am: inv.am ?? [], g: inv.g ?? [] })),
    loadouts: (raw.loadouts ?? []).map((lo) => ({ ...lo, w: lo.w ?? [] })),
    // `as` sur l'arité seule : le contenu est celui du contrat, seule la longueur fixe du
    // tuple que JSON Schema ne sait pas dire est réaffirmée (cf. en-tête).
    projectiles: (raw.projectiles ?? []).map((pr) => ({ ...pr, p: (pr.p ?? []) as ReplayStep[] })),
    roster: raw.roster ?? [],
    shots: raw.shots ?? [],
    structure: (raw.structure ?? []).map((s) => ({ ...s, poly: (s.poly ?? []) as ReplayXY[] })),
    tracks: (raw.tracks ?? []).map((t) => ({ ...t, points: t.points ?? [] })),
  }
}
