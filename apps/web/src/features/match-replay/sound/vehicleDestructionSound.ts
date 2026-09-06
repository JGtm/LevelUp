/**
 * vehicleDestructionSound.ts — LE SON DE LA DESTRUCTION D'UN VÉHICULE (lot du 2026-09-05).
 *
 * LE SIGNAL EST EXACTEMENT CELUI DE L'EFFET VISUEL (schéma 39, `vehiclesLayer.ts`) : un
 * véhicule dont `end === VEHICLE_END_DESTROYED` ET dont `tEnd` est publié — les deux
 * conditions, portées par `vehicleDestructionFrame`, la MÊME fonction que l'explosion du
 * calque (`vehiclesPaint.ts`). Tant que le Go ne publie que `end: "unknown"` (l'état actuel
 * de CHAQUE artefact existant), cette chaîne rend ZÉRO événement : le son est PRÊT, il
 * n'invente pas la mesure — exactement comme l'effet visuel qu'il accompagne.
 *
 * C'EST UN BRUITAGE COMME LES AUTRES : un one-shot posé dans `buildSoundTimeline` (catégorie
 * `weapon` du tiroir — même choix que les bobines : une explosion du monde, et le tiroir n'a
 * pas de case « décor »), famille `sfx` à l'export, plafond de voix normal du lecteur. RIEN
 * à voir avec le bus moteur (0,85) des boucles continues (`vehicleEngineSound.ts`) : la
 * destruction est un événement, pas un état.
 *
 * LES BANQUES SONT DÉDUPLIQUÉES, ET C'EST UNE MESURE (md5 sur les sources, 2026-09-05) :
 * dans `sons_v3_reconstruits/<Vehicule>/destruction/`, les fichiers du Wasp sont IDENTIQUES
 * octet à octet à ceux du Warthog_roquettes, et ceux du Wraith à ceux de la Banshee. Livrer
 * huit dossiers aurait dupliqué quatre banques : les fichiers sont donc nommés par JEU DE
 * SONS (`vehicle_boom_<set>_<n>`), et les FAMILLES du document (les mêmes clés que
 * `VEHICLE_ENGINE_STEMS`) se mappent dessus. Six jeux distincts :
 *
 *   warthog         <- banque Warthog_roquettes (familles warthog, wasp)
 *   covenant_lourd  <- banque Banshee           (familles banshee, wraith)
 *   chopper         <- banque Chopper           (famille chopper)
 *   ghost           <- banque Ghost             (famille ghost)
 *   scorpion        <- banque Scorpion          (famille scorpion)
 *   mongoose        <- banque Gungoose          (famille mongoose — le Gungoose EST un
 *                      Mongoose armé, même châssis : parenté écrite, pas un emprunt arbitraire,
 *                      même règle que le traqueur qui prend le son du capteur)
 *
 * LES TROIS PRISES PROCHES de chaque banque (`explosion.wav`, `_v2`, `_v3`) sont les trois
 * variantes tirées à chaque destruction (`SOUND_VARIANTS`, même mécanique que le grappin) ;
 * les prises LOINTAINES (`explosion_lointaine_*`) sont IGNORÉES — le rejeu n'a pas de notion
 * de distance de caméra. Durées livrées : 2,58 à 5,36 s, toutes ENTIÈRES (aucune n'atteint le
 * plafond de sûreté du lecteur, 12 s — la troncature au fondu n'a pas eu à servir).
 * Égalisation : -16 LUFS visé, plafond -1 dBTP, gain LINÉAIRE seul (convention du lot R2-S).
 *
 * CE QUI NE SONNE PAS : le DÉCOR (falcon, pelican, phantom, skiff — `vehicleIsDecor`, le même
 * refus que le calque et les moteurs : rien ne les détruit dans une partie), une famille NON
 * RÉSOLUE (chaîne vide) ou SANS JEU DE SONS — silence propre, jamais la banque d'une voisine.
 */
import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { vehicleDestructionFrame, vehicleIsDecor } from '../model/vehiclesLayer'

/**
 * VEHICLE_BOOM_SETS — jeu de sons -> ses trois variantes, la première portant le stem que la
 * jointure famille -> set désigne. C'est un MANIFESTE : le garde-rail
 * `replaySoundAssets.guard.test.ts` le rejoue contre le dossier d'assets.
 */
export const VEHICLE_BOOM_SETS: Readonly<Record<string, readonly string[]>> = {
  warthog: ['vehicle_boom_warthog_1', 'vehicle_boom_warthog_2', 'vehicle_boom_warthog_3'],
  covenant_lourd: [
    'vehicle_boom_covenant_lourd_1',
    'vehicle_boom_covenant_lourd_2',
    'vehicle_boom_covenant_lourd_3',
  ],
  chopper: ['vehicle_boom_chopper_1', 'vehicle_boom_chopper_2', 'vehicle_boom_chopper_3'],
  ghost: ['vehicle_boom_ghost_1', 'vehicle_boom_ghost_2', 'vehicle_boom_ghost_3'],
  scorpion: ['vehicle_boom_scorpion_1', 'vehicle_boom_scorpion_2', 'vehicle_boom_scorpion_3'],
  mongoose: ['vehicle_boom_mongoose_1', 'vehicle_boom_mongoose_2', 'vehicle_boom_mongoose_3'],
}

/**
 * VEHICLE_BOOM_FAMILY_SET — famille du document (`VehicleTrack.family`, les mêmes clés que
 * `VEHICLE_ENGINE_STEMS`) -> jeu de sons. Les doublons de banque mesurés se lisent ici :
 * wasp partage la banque du warthog, wraith celle de la banshee. Une famille absente = silence.
 */
export const VEHICLE_BOOM_FAMILY_SET: Readonly<Record<string, string>> = {
  warthog: 'warthog',
  wasp: 'warthog',
  banshee: 'covenant_lourd',
  wraith: 'covenant_lourd',
  chopper: 'chopper',
  ghost: 'ghost',
  scorpion: 'scorpion',
  mongoose: 'mongoose',
}

/**
 * VEHICLE_BOOM_SOUND_VARIANTS — premier stem de chaque jeu -> ses trois variantes, fusionné
 * dans `SOUND_VARIANTS` (replaySoundVariants.ts) pour que `soundEvent` les attache et que
 * `pickVariantStem` tire à chaque destruction — la même mécanique que les tirs de véhicule,
 * et le même sens de dépendance (ce module n'importe PAS replaySoundVariants : pas de cycle).
 */
export const VEHICLE_BOOM_SOUND_VARIANTS: Readonly<Record<string, readonly string[]>> =
  Object.fromEntries(Object.values(VEHICLE_BOOM_SETS).map((stems) => [stems[0], stems]))

/** Tous les stems de destruction — préchargement et garde-rail d'assets. */
export function allVehicleBoomStems(): string[] {
  return Object.values(VEHICLE_BOOM_SETS).flat()
}

/**
 * vehicleDestructionSound — la destruction d'UN véhicule : la frame de la preuve et le stem
 * de la première variante, ou `null` (destruction non établie, décor, famille sans jeu).
 * `buildSoundTimeline` fait le reste (datation `frameToMs`, attache des variantes).
 */
export function vehicleDestructionSound(
  track: ReplayVehicleTrackReady,
): { frame: number; stem: string } | null {
  const frame = vehicleDestructionFrame(track)
  if (frame === null || !track.family || vehicleIsDecor(track.family)) return null
  const set = VEHICLE_BOOM_FAMILY_SET[track.family]
  const stems = set ? VEHICLE_BOOM_SETS[set] : undefined
  const stem = stems?.[0]
  return stem ? { frame, stem } : null
}
