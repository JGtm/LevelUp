/**
 * replaySoundVariants.ts — UN GESTE, PLUSIEURS FICHIERS : le tirage de variante.
 *
 * EXTRAIT DE `replaySound.ts` LE 2026-08-26, pour la même raison que `replaySoundCursor.ts`
 * et `grenadeSound.ts` avant lui : le fichier d'origine atteignait son plafond de lisibilité.
 * Ce qui vit ici est tout ce qui concerne le CHOIX du fichier — le type d'événement, le
 * manifeste des variantes, le constructeur qui les attache, et le tirage lui-même.
 *
 * POURQUOI DES VARIANTES. Un événement Wwise dont la couche est un `RandomSequence` ne joue
 * pas un fichier : il en TIRE un, uniformément (mesure du chantier armes : 6 976 tables de
 * poids égales sur 6 976). Livrer une seule variante publiait donc un geste appauvri — trois
 * poses de champ de réparation dans un même match sonnaient exactement pareil, là où le jeu
 * en joue trois différentes.
 *
 * LE TIRAGE SE FAIT À LA LECTURE, JAMAIS À LA CONSTRUCTION DE LA PISTE. Une piste est bâtie
 * une fois pour tout le match (`buildSoundTimeline`, mémoïsée) : y tirer figerait le choix, et
 * deux occurrences du même geste rejoueraient le même fichier — exactement ce que la variante
 * est censée éviter.
 */

/** Un événement sonore posé sur l'horloge du rejeu. */
export interface ReplaySoundEvent {
  /** Instant en ms sur l'horloge du rejeu (celle du fil et des fiches). */
  ms: number
  /**
   * Stem du fichier sous static/sounds/{titleSlug}/ — la PREMIÈRE variante quand il y en a
   * plusieurs, et le fichier joué quand `variants` est absent.
   */
  stem: string
  /** Toutes les variantes jouables de ce geste, `stem` compris. Absent = un seul fichier. */
  variants?: readonly string[]
}

/**
 * SOUND_VARIANTS — les gestes que le jeu joue en TIRANT une variante, et la liste EXACTE de
 * leurs fichiers. C'est un MANIFESTE au même titre que les tables de `replaySound.ts` : le
 * garde-rail `replaySoundAssets.guard.test.ts` le rejoue contre le dossier d'assets, un stem
 * sans fichier ou un fichier sans stem casse le test.
 *
 * LA PREMIÈRE VARIANTE GARDE LE STEM NU, et ce n'est pas de la coquetterie : les tables et les
 * garde-rails existants nomment déjà `grapple_fire`, `repulsor_kill`, `repair_field_activate`.
 * Suffixer les trois aurait fait un renommage en cascade sans rien apporter à l'oreille.
 *
 * PÉRIMÈTRE : seuls les gestes dont le rendu du 2026-08-26 a produit plusieurs variantes y
 * entrent. Un stem absent de cette table se joue tel quel — c'est le cas de tous les autres,
 * y compris les sept sons d'objectif (leurs événements n'ont qu'une variante par couche).
 */
export const SOUND_VARIANTS: Readonly<Record<string, readonly string[]>> = {
  grapple_fire: ['grapple_fire', 'grapple_fire_v2', 'grapple_fire_v3'],
  repulsor_kill: ['repulsor_kill', 'repulsor_kill_v2', 'repulsor_kill_v3'],
  repair_field_activate: [
    'repair_field_activate',
    'repair_field_activate_v2',
    'repair_field_activate_v3',
  ],
  repair_field_end: ['repair_field_end', 'repair_field_end_v2', 'repair_field_end_v3'],
  // LE CRANE d'Oddball (banque `sb_004_mod_mp_oddball`, rendu du 2026-08-29) : quatre de ses
  // gestes sont des `RandomSequence` a plusieurs `.wem`, exactement comme le grappin. Le jeu
  // TIRE une variante a chaque lecture ; le rejeu aussi. `objective_skull_spawn` n'y est pas :
  // son evenement est « 1 couche, 1 son », un seul fichier.
  objective_skull_despawn: ['objective_skull_despawn', 'objective_skull_despawn_v2'],
  objective_skull_taken: ['objective_skull_taken', 'objective_skull_taken_v2'],
  objective_skull_pickup: ['objective_skull_pickup', 'objective_skull_pickup_v2'],
  objective_skull_dropped: [
    'objective_skull_dropped',
    'objective_skull_dropped_v2',
    'objective_skull_dropped_v3',
  ],
}

/**
 * soundEvent — le SEUL constructeur d'événement sonore de la chaîne : il attache les variantes
 * du stem quand il en a. Passer par lui partout est ce qui garantit qu'un geste à variantes ne
 * peut pas être poussé « nu » par un appelant qui l'ignorerait.
 */
export function soundEvent(ms: number, stem: string): ReplaySoundEvent {
  const variants = SOUND_VARIANTS[stem]
  return variants ? { ms, stem, variants } : { ms, stem }
}

/**
 * pickVariantStem — le TIRAGE d'une variante, à la lecture. Uniforme, comme le
 * `RandomSequence` du jeu. `rnd` est injectable pour que le test soit déterministe.
 */
export function pickVariantStem(
  event: Pick<ReplaySoundEvent, 'stem' | 'variants'>,
  rnd: () => number = Math.random,
): string {
  const list = event.variants
  if (!list || list.length === 0) return event.stem
  const i = Math.min(list.length - 1, Math.max(0, Math.floor(rnd() * list.length)))
  return list[i] ?? event.stem
}

/** Tous les stems jouables d'un événement — ce que le préchargement doit couvrir. */
export function stemsOf(event: Pick<ReplaySoundEvent, 'stem' | 'variants'>): readonly string[] {
  return event.variants && event.variants.length > 0 ? event.variants : [event.stem]
}
