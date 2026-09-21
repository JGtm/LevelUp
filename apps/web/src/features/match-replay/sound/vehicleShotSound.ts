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
 *  - toute arme de véhicule dont le tag `weap` n'est pas documenté (Shade...) : la clé
 *    n'existe pas, la table ne répond pas, même règle que les montages.
 *
 * # LE WARTHOG SE DÉPARTAGE PAR LA FAMILLE DE SON CHÂSSIS (lot 5.8.4, 2026-09-21)
 *
 * Le seul tag `weap` retrouvé (`c7d50912`) ne départage pas LAAG / Gauss / roquettes
 * (`vehicleWeaponMounts.ts`, témoin V3F §4), et la seule reconstruction validée est celle des
 * ROQUETTES. Jusqu'à ce lot, elle sonnait pour les trois : « assumé et écrit plutôt que muet ».
 *
 * CE QUE LA MESURE DIT DE CE CHOIX, ET IL NE TIENT PAS : sur les 92 artefacts cuits du parc
 * local, les **105 tirs** qui portent ce tag viennent TOUS d'un châssis `warthog` — 100 avec un
 * véhicule publié, 5 sans. **Zéro `rockethog`, zéro `warthog_gauss`.** Le seul son de Warthog
 * jamais joué était donc FAUX dans 100 % des cas mesurés, et le « plutôt que muet » ne
 * préservait aucun cas juste.
 *
 * LE DÉPARTAGE N'A PAS BESOIN D'UN RAPPORT DE RE : le document PUBLIE la famille du châssis du
 * véhicule tireur (`vehicles[].family` : `warthog`, `warthog_gauss`, `rockethog`), et un tir en
 * véhicule porte le slot de son porteur (`Shot.v`). L'entrée se scinde donc, comme l'en-tête le
 * prévoyait — par la FAMILLE et non par le tag.
 *
 * ET LES DEUX AUTRES SE TAISENT, PARCE QUE C'EST LA RÈGLE DU CHANTIER : aucune reconstruction de
 * LAAG ni de Gauss n'existe. Leur prêter le tir de roquette (ou la LMG du Falcon, qui est
 * pourtant une mitrailleuse UNSC) serait « le son d'une voisine » — ce que ce fichier s'interdit
 * en toutes lettres pour le Wasp M2. Le silence propre est dit, pas caché ; le remède est une
 * reconstruction de LAAG, pas un stem emprunté.
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
  [vehicleWeapTag('00015cfa'), 'vehicle_shot_scorpion_1'], // Scorpion — canon principal.
  // LE WARTHOG N'EST PAS ICI : son tag est AMBIGU et se résout par la famille du châssis
  // (`VEHICLE_SHOT_SOUND_BY_CHASSIS`). Une entrée dans les DEUX tables serait la divergence
  // garantie — un tag, une seule règle de résolution.
  [vehicleWeapTag('11725dc4'), 'vehicle_shot_wasp_1'], // Wasp M1 — autocanon de menton.
  // Wasp M2 (d3c407ed, missiles) : pas de reconstruction — silence, cf. en-tête.
  [vehicleWeapTag('0042678e'), 'vehicle_shot_gungoose_1'], // Gungoose — mitrailleuses avant.
  [vehicleWeapTag('00015cd3'), 'vehicle_shot_falcon_lmg_1'], // Falcon — tourelle LMG.
])

/**
 * VEHICLE_SHOT_SOUND_BY_CHASSIS — LES TAGS AMBIGUS : `Shot.w` -> (famille du châssis -> stem).
 *
 * UN SEUL TAG Y FIGURE AUJOURD'HUI, et c'est le Warthog (cf. l'en-tête). La table est néanmoins
 * une TABLE et non un cas particulier : le départage par famille de châssis est une RÈGLE (un
 * même `weap` monté sur plusieurs variantes de châssis), et le prochain tag ambigu s'y ajoutera
 * en une ligne plutôt qu'en une seconde branche.
 *
 * UNE FAMILLE ABSENTE DE LA SOUS-TABLE SE TAIT. C'est le cas du `warthog` (LAAG) et du
 * `warthog_gauss` : aucune reconstruction ne les couvre, et emprunter celle des roquettes est
 * exactement ce que la mesure du lot 5.8.4 a disqualifié.
 */
export const VEHICLE_SHOT_SOUND_BY_CHASSIS: ReadonlyMap<string, ReadonlyMap<string, string>> =
  new Map([
    [
      vehicleWeapTag('c7d50912'),
      new Map([
        // La seule prise validée : le Rockethog. Le LAAG et le Gauss se taisent (cf. l'en-tête).
        ['rockethog', 'vehicle_shot_warthog_rocket_1'],
      ]),
    ],
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
 *
 * `chassisFamily` est la famille du véhicule TIREUR (`vehicles[].family` du porteur désigné par
 * `Shot.v`), ou `undefined` quand le document ne publie pas ce véhicule. Elle ne sert QU'AUX TAGS
 * AMBIGUS — un tag qui désigne une seule arme n'a rien à départager, et lui faire dépendre son
 * son d'un châssis le rendrait muet le jour où une nouvelle variante apparaît.
 *
 * LES DEUX TABLES SONT DISJOINTES ET L'AMBIGUË PASSE D'ABORD : un tag ambigu SANS famille lue se
 * tait, il ne retombe pas sur un stem par défaut. C'est le cas des 5 tirs de Warthog du parc dont
 * le véhicule n'est pas publié — deviner leur variante ferait exactement le bruit que ce lot
 * retire.
 */
export function vehicleShotSoundStem(
  weaponID: string,
  chassisFamily: string | undefined,
): string | undefined {
  const parChassis = VEHICLE_SHOT_SOUND_BY_CHASSIS.get(weaponID)
  if (parChassis) return chassisFamily ? parChassis.get(chassisFamily) : undefined
  return VEHICLE_SHOT_SOUND_STEMS.get(weaponID)
}
