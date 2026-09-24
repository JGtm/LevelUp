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
 * DES VARIANTES D'ARMES À TIR CONTINU (Ghost, Banshee M1, Chopper, LMG du Falcon, LMG du Wasp)
 * restent ici alors qu'aucun film ne les publie encore : ce sont des ASSETS livrés, pas des clés de
 * tag — le registre les nommera le jour où le décodeur lira le tir continu (lot M4b).
 *
 * LIVRAISON DU 2026-09-24 (retours du rejeu, lot M6.1 ; sons désignés À L'OREILLE par
 * l'utilisateur sur la page d'écoute `rr_2026-09-23`, « les premiers candidats sont tous bons »).
 * Sources : rendus V3E hors dépôt (`Halo Infinite - Sons v75/rr_2026-09-23/`, manifeste
 * `manifeste_rr_2026-09-23.json` : gains de chemin complets, une normalisation commune par rendu).
 * Même recette que la livraison du 2026-09-04 (même en-tête RIFF `Lavf`, mesurée sur les stems
 * livrés) : coupe à 1,2 s avec un fondu de sortie de 50 ms pour un coup plus long, source plus
 * courte livrée entière ; gain LINÉAIRE = min(-16 LUFS - intégré, -1 dBTP - crête vraie), crête
 * seule sous 0,4 s (intégré non mesurable) ; 48 kHz / 16 bits / stéréo.
 *  - MISSILES DU WASP (`vehicle_shot_wasp_*`) : REMPLACÉS par le rendu rééquilibré
 *    (`wasp_missiles_controle/coup_3p*`, événement e22a0d32, couches à +22 / +13 / +8 dB au lieu
 *    des +7,8 / +2,8 / +0,8 dB de l'ancien rendu rev9 — mêmes médias, preuve au manifeste).
 *  - LMG DU WASP (`vehicle_shot_wasp_lmg_*`, événement 5baca8ee, cadence du tag 600/min) : la
 *    BOUCLE 3P (8 s, `boucle_8s_3p`) tenue pendant le tir et le COUP 3P isolé (0,2 s, deux prises)
 *    en queue à l'arrêt. La boucle est égalisée par la recette ; la queue reçoit le MÊME gain
 *    rapporté au rendu (écart de normalisation du manifeste : 1,64 contre 3,97 dB), pour que le
 *    dernier coup ne sonne pas plus fort que la rafale qu'il termine. Câblage : lot M4b.
 *  - LANCE-GRENADES DU FALCON : AUCUN fichier — le registre lui donne les roquettes du
 *    Rockethog, le jeu joue le MÊME événement (18 médias identiques, sonde SONS du 2026-09-24).
 *  - Variantes lointaines A-E : NON retenues (décision du 2026-09-24), non livrées.
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
  // LMG du Wasp : les deux prises du COUP isolé, la queue d'une rafale (cf. en-tête).
  vehicle_shot_wasp_lmg_1: ['vehicle_shot_wasp_lmg_1', 'vehicle_shot_wasp_lmg_2'],
}

/**
 * VEHICLE_SHOT_HELD_LOOPS — LE CORPS DE BOUCLE d'une arme à TIR CONTINU, keyé par le stem de la
 * première variante de son COUP (celui que le registre du titre nommera, comme pour une arme au
 * coup). Décision de l'utilisateur du 2026-09-24 : la boucle se tient du DÉBUT à la FIN de la
 * rafale, jamais relancée à chaque coup (Q5), et le coup isolé sonne en queue à l'arrêt.
 *
 * INERTE AUJOURD'HUI, ET DÉCLARÉ QUAND MÊME : aucun film ne publie encore d'intervalle de tir
 * continu — c'est le lot M4b (vue de contrôle) qui le lira et qui câblera cette table. Elle existe
 * dès la livraison parce que le garde-rail d'assets exige que tout fichier livré soit DÉCLARÉ (un
 * fichier que rien ne nomme est un asset mort), et parce que ses propriétés (format, durée de
 * boucle, variante de coup existante) se vérifient sur le fichier, pas sur le câblage.
 * Ce ne sont pas des sons d'ARME au sens de la règle de durée (1,2 s) : une boucle de rafale
 * retronquée à la coupe des armes deviendrait un hoquet.
 */
export const VEHICLE_SHOT_HELD_LOOPS: Readonly<Record<string, string>> = {
  vehicle_shot_wasp_lmg_1: 'vehicle_shot_wasp_lmg_loop',
}
