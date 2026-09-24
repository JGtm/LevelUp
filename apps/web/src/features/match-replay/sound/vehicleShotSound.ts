/**
 * vehicleShotSound.ts — LES VARIANTES DU SON D'UN TIR EN VÉHICULE (lot du 2026-09-04).
 *
 * DEPUIS LE SCHÉMA 69 (retours du rejeu du 2026-09-23, lot M4a), QUELLE ARME SONNE QUEL STEM
 * N'EST PLUS DÉCIDÉ ICI. La table `Shot.w -> stem` de ce fichier était clée par des tags `weap`
 * du module du jeu, dont la plupart n'apparaissaient dans aucun film (7 sur 11 au parc du
 * 2026-09-23). Le REGISTRE DU TITRE la remplace (`config/titles/{slug}/mappings/vehicle_weapons.toml`,
 * clé = tag OBSERVÉ, `sound` = le stem de la première variante, ou `silence` et sa raison) ; le
 * document le publie et `model/vehicleWeaponRegistry.ts` le lit. Ce fichier ne garde que ce que
 * le registre ne dit pas : les DEUX PRISES de chaque son, pour le tirage par coup.
 *
 * LES SOURCES SONT LES RECONSTRUCTIONS Wwise VALIDÉES PAR L'UTILISATEUR (manifeste_v3.json,
 * rev12, `.ai/V7.5/film_re/sons_v3_reconstruits/`), livrées en DEUX variantes (deux prises de la
 * même perspective), coupées à la règle des ARMES (1,2 s max), 48 kHz / 16 bits / stéréo,
 * égalisées à la convention du lot R2-S (-16 LUFS visé, plafond -1 dBTP, gain LINÉAIRE seul).
 * Perspective : la prise « 3e personne » (vue spectateur) par défaut, sauf désignation à l'écoute
 * (Wraith et Banshee M2 : « vue pilote prise 1 », 2026-09-04 ; Banshee M1 : la reconstruction
 * ORIGINALE de 0,125 s, 2026-09-05 — la piste « cadence du tag » a été abandonnée à la réécoute).
 *
 * DES VARIANTES D'ARMES À TIR CONTINU (Ghost, Banshee M1, Chopper, LMG du Falcon) restent ici
 * alors qu'aucun film ne les publie encore : ce sont des ASSETS livrés, pas des clés de tag — le
 * registre les nommera le jour où le décodeur lira le tir continu (lot M4b).
 *
 * LES TIRS DE VÉHICULE SONT DES BRUITAGES COMME LES AUTRES ARMES : catégorie `weapon` du
 * tiroir, plafond de voix du lecteur, famille `sfx` à l'export — rien à voir avec le bus moteur.
 */

/**
 * VEHICLE_SHOT_SOUND_VARIANTS — les deux prises de chaque arme, fusionnées dans
 * `SOUND_VARIANTS` (replaySoundVariants.ts) pour que `soundEvent` les attache et que
 * `pickVariantStem` tire à chaque coup — la MÊME mécanique que le grappin. Table nommée ici
 * (et pas inline là-bas) parce que le garde-rail d'assets a besoin de la reconnaître : ces
 * variantes suivent la règle de durée des ARMES (1,2 s), pas celle des équipements.
 */
export const VEHICLE_SHOT_SOUND_VARIANTS: Readonly<Record<string, readonly string[]>> = {
  vehicle_shot_ghost_1: ['vehicle_shot_ghost_1', 'vehicle_shot_ghost_2'],
  vehicle_shot_banshee_m2_1: ['vehicle_shot_banshee_m2_1', 'vehicle_shot_banshee_m2_2'],
  // Banshee M1 : _1 = prise 3e personne (vue spectateur), _2 = vue pilote — cf. en-tête.
  vehicle_shot_banshee_m1_1: ['vehicle_shot_banshee_m1_1', 'vehicle_shot_banshee_m1_2'],
  vehicle_shot_wraith_1: ['vehicle_shot_wraith_1', 'vehicle_shot_wraith_2'],
  vehicle_shot_chopper_1: ['vehicle_shot_chopper_1', 'vehicle_shot_chopper_2'],
  vehicle_shot_scorpion_1: ['vehicle_shot_scorpion_1', 'vehicle_shot_scorpion_2'],
  vehicle_shot_warthog_rocket_1: [
    'vehicle_shot_warthog_rocket_1',
    'vehicle_shot_warthog_rocket_2',
  ],
  vehicle_shot_wasp_1: ['vehicle_shot_wasp_1', 'vehicle_shot_wasp_2'],
  vehicle_shot_gungoose_1: ['vehicle_shot_gungoose_1', 'vehicle_shot_gungoose_2'],
  vehicle_shot_falcon_lmg_1: ['vehicle_shot_falcon_lmg_1', 'vehicle_shot_falcon_lmg_2'],
}
