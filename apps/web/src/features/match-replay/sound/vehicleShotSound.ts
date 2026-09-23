/**
 * vehicleShotSound.ts — LE SON D'UN TIR EN VÉHICULE (lot du 2026-09-04).
 *
 * LA JOINTURE, VÉRIFIÉE SUR PIÈCES AVANT CE FICHIER : `Shot.w` d'un tir marqué `v` (tir en
 * véhicule) porte pour une arme DE VÉHICULE l'identifiant `0x<weap 8 hex>00000000` — le
 * gabarit établi et vérifié en direct par `vehicleWeaponMounts.ts` (artefact 0d76e8f1,
 * Warthog c7d50912 et Wasp 11725dc4). Ces identifiants sont ABSENTS de `weaponLabels`
 * (le registre ne nomme que les armes de joueur) : `shotSoundStem` tombait donc sur
 * `undefined` et TOUT tir d'arme de véhicule était muet. Cette table est la jointure
 * manquante — même clé que `VEHICLE_WEAPON_MOUNTS`, même fonction `vehicleWeapTag`, jamais
 * une copie du gabarit.
 *
 * LES SOURCES SONT LES RECONSTRUCTIONS Wwise VALIDÉES PAR L'UTILISATEUR (manifeste_v3.json,
 * rev12, `.ai/V7.5/film_re/sons_v3_reconstruits/`) : Ghost, Banshee mode 2 (bombe à
 * combustible), Wraith, Chopper validés le 2026-09-04 ; Scorpion, Warthog à roquettes, Wasp,
 * Gungoose et Falcon (tourelle LMG) validés antérieurement. Chaque arme est livrée en DEUX
 * variantes (deux prises de la même perspective), coupées à la règle des ARMES (1,2 s max,
 * une source plus courte se livre entière), 48 kHz / 16 bits / stéréo, égalisées à la
 * convention du lot R2-S (-16 LUFS visé, plafond -1 dBTP, gain LINÉAIRE seul).
 *
 * LE CHOIX DE PERSPECTIVE : le rejeu est une vue SPECTATEUR, donc la prise « 3e personne »
 * (vue extérieure) par défaut — SAUF quand l'utilisateur a désigné une prise à l'écoute :
 * Wraith et Banshee M2 sonnent la « vue pilote prise 1 » validée le 2026-09-04 (prise 2 de la
 * même perspective en seconde variante). La prise extérieure 1 du Ghost dure 0,128 s — trop
 * courte pour une mesure LUFS intégrée : elle est livrée plafonnée à -1 dBTP, au plus près.
 *
 * LA BANSHEE MODE 1 (canons à plasma, weap 0000aa68) SONNE DEPUIS LE 2026-09-05, et c'est
 * l'UTILISATEUR qui a tranché : la reconstruction ORIGINALE (0,125 s par tir) est la bonne —
 * la piste « cadence du tag » a été essayée (v2 à 240 coups/min) et ABANDONNÉE à la réécoute.
 * Les votes priment sur tout critère (RECETTE_SONS_ARMES §5) : ses deux prises livrées sont
 * les fichiers `tir_M1_*` d'origine, la 3e personne d'abord (vue spectateur), la vue pilote
 * en seconde variante. Trop courtes pour une mesure LUFS intégrée, elles sont livrées au
 * plafond -1 dBTP, comme la prise extérieure 1 du Ghost.
 *
 * CE QUI NE SONNE PAS, ET POURQUOI (mesuré ou décidé, jamais supposé) :
 *  - les MISSILES du Wasp (weap d3c407ed) : aucune reconstruction n'existe — silence ;
 *  - le LANCE-GRENADES du Falcon (`0bb6976b`, 52 tirs observés, décision Q10 du 2026-09-23) :
 *    aucune reconstruction (ni au manifeste V3, ni sur les planches, ni sous `static/sounds`) —
 *    silence DÉCIDÉ, question posée à l'utilisateur ;
 *  - toute arme de véhicule dont le tag `weap` n'est pas documenté (Shade...) : la clé
 *    n'existe pas, la table ne répond pas, même règle que les montages.
 *
 * # LE ROCKETHOG SONNE PAR SON TAG (retours du 2026-09-23, lot L1.5, décision Q9)
 *
 * Le lot 5.8.4 (2026-09-21) avait déclaré `c7d50912` AMBIGU (« LAAG / Gauss / roquettes non
 * départagés ») et ne le faisait sonner que pour la famille de châssis `rockethog`. C'était faux :
 * `c7d50912` est le LANCE-ROQUETTES du Rockethog (`WARTHOG_FINAL_2026-09-02.md` §1 :
 * `vehi bcfb852f -> weap c7d50912`, banque `veh_un_rockethog`) — la LAAG est `0c6fd911`, le
 * canon Gauss `8647925a`. Et AUCUN châssis ne porte la famille `rockethog` : les 127 tirs en
 * véhicule du parc sont portés par la tourelle ENFANT `bcfb852f` (125, famille vide) ou par le
 * châssis `warthog` (2). Le son validé le 31/08 était donc MUET depuis le 21/09. Le tag suffit :
 * la table des tags ambigus et le départage par famille ont disparu avec lui (0 code mort).
 *
 * LES TIRS DE VÉHICULE SONT DES BRUITAGES COMME LES AUTRES ARMES : catégorie `weapon` du
 * tiroir (ils entrent dans la piste par la boucle des tirs de `buildSoundTimeline`), plafond
 * de voix du lecteur, famille `sfx` à l'export — RIEN à voir avec le bus moteur (0,85) des
 * boucles continues (`vehicleEngineSound.ts`).
 *
 * Un tir d'arme DE JOUEUR depuis un siège passager n'arrive jamais ici : son `Shot.w` est
 * dans `weaponLabels`, et `shotSoundStem` répond par `WEAPON_SOUND_STEMS` AVANT ce repli.
 */
import { vehicleWeapTag } from '../model/vehicleWeaponMounts'

/**
 * VEHICLE_SHOT_SOUND_STEMS — `Shot.w` (gabarit weap) -> stem de la PREMIÈRE variante.
 * Les variantes s'attachent par `SOUND_VARIANTS` (replaySoundVariants.ts), même mécanique
 * de tirage par coup que le grappin. Un tag absent = silence propre.
 */
export const VEHICLE_SHOT_SOUND_STEMS: ReadonlyMap<string, string> = new Map([
  [vehicleWeapTag('00015435'), 'vehicle_shot_ghost_1'], // Ghost — canons à plasma jumeaux.
  [vehicleWeapTag('0000aa69'), 'vehicle_shot_banshee_m2_1'], // Banshee M2 — bombe à combustible.
  [vehicleWeapTag('0000aa68'), 'vehicle_shot_banshee_m1_1'], // Banshee M1 — canons à plasma (validée 2026-09-05).
  [vehicleWeapTag('121b4009'), 'vehicle_shot_wraith_1'], // Wraith — mortier à plasma.
  [vehicleWeapTag('b40e9618'), 'vehicle_shot_chopper_1'], // Chopper — canons jumeaux avant.
  // Scorpion — canon principal, sous le tag que le film PUBLIE (`00015cfa`, celui du module, n'a
  // jamais été vu dans un document).
  [vehicleWeapTag('49e40d17'), 'vehicle_shot_scorpion_1'],
  [vehicleWeapTag('c7d50912'), 'vehicle_shot_warthog_rocket_1'], // Rockethog — lance-roquettes (Q9).
  // Falcon lance-grenades (0bb6976b) : pas de reconstruction — silence DÉCIDÉ, cf. en-tête.
  [vehicleWeapTag('11725dc4'), 'vehicle_shot_wasp_1'], // Wasp M1 — autocanon de menton.
  // Wasp M2 (d3c407ed, missiles) : pas de reconstruction — silence, cf. en-tête.
  [vehicleWeapTag('0042678e'), 'vehicle_shot_gungoose_1'], // Gungoose — mitrailleuses avant.
  [vehicleWeapTag('00015cd3'), 'vehicle_shot_falcon_lmg_1'], // Falcon — tourelle LMG.
])

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

/**
 * vehicleShotSoundStem — le stem d'un tir dont `weaponLabels` ne répond pas, ou undefined.
 *
 * Appelé par `shotSoundStem` (replaySound.ts) en SECOND : une arme de joueur garde sa table.
 * LE TAG SEUL DÉCIDE : chaque tag `weap` observé désigne UNE arme (le Rockethog compris, cf.
 * l'en-tête) — aucun châssis n'a à être lu, et un tag absent se tait.
 */
export function vehicleShotSoundStem(weaponID: string): string | undefined {
  return VEHICLE_SHOT_SOUND_STEMS.get(weaponID)
}
