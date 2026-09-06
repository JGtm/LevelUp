/**
 * replaySound.ts — CE QUI SONNE, ET QUAND.
 *
 * QUATRE VOISINS, EXTRAITS AU FIL DES LOTS parce que ce fichier atteint son plafond :
 * `replaySoundCursor.ts` tient le curseur de lecture (un événement ne part qu'une fois) et
 * `grenadeSound.ts` tout ce qui est propre à la GRENADE (2026-08-18, lot R2-G) ;
 * `replaySoundVariants.ts` tient le TIRAGE d'une variante et `objectiveSound.ts` les sons
 * d'OBJECTIF, leur camp et leurs silences (2026-08-26). Même règle à chaque fois : une famille
 * qui a sa source, sa clé de jointure et sa doctrine mérite son fichier.
 *
 * LES SOURCES (deux, par nature — fusion du 2026-08-16) : les sons d'ARMES sont EXTRAITS
 * DU JEU (chantier sons-armes : banks Wwise decodees, coups reconstitues selon la
 * semantique prouvee, selection VOTEE par l'utilisateur — recette et provenance :
 * `.ai/V7.5/RECETTE_SONS_ARMES.md`) ; les sons d'EVENEMENTS (lancers, explosions, melee
 * fatale, equipements) viennent du pack fourni par l'utilisateur (2026-08-13/15). Tout vit
 * sous `static/sounds/halo_infinite/`, livre par `_outils/livraison.py` (armes seulement,
 * les evenements ne sont jamais touches). DEUX FAMILLES DE NOMS, parce qu'il y a deux
 * natures : une ARME porte le nom de sa clé canonique (weapon_names.toml, PAS le nom de
 * fichier FR des images — piège Crémateur/Cindershot) ; un ÉVÉNEMENT que le registre
 * d'armes ne nomme pas porte le nom de l'événement (`throw_*`, `explosion_*`,
 * `melee_kill`). Le manifeste — les tables ci-dessous et les deux de `grenadeSound.ts` —
 * est la liste EXACTE des fichiers livrés ; le
 * garde-rail `replaySoundAssets.guard.test.ts` le rejoue contre le dossier : un stem sans
 * fichier ou un fichier sans stem casse le test, jamais l'écoute.
 *
 * LA DURÉE EST UNE PROPRIÉTÉ DE LA CATÉGORIE, et elle est portée par le FICHIER, pas par
 * le lecteur (décision utilisateur du 2026-08-16) : armes, lancers et mêlée à 1,2 s ;
 * explosions de grenade et équipements jusqu'à 4 s, parce qu'à la seconde ils s'entendaient
 * « écourtés ». Le lecteur joue le fichier jusqu'à sa fin (replayAudio.ts) — allonger un
 * son, c'est donc re-couper son asset, jamais toucher au moteur.
 *
 * LE NIVEAU EST ÉGALISÉ, ET C'EST AUSSI UNE PROPRIÉTÉ DU FICHIER (lot R2-S, 2026-08-18).
 * Les 41 assets étaient livrés à leur niveau de source : de -9,20 LUFS (rayon de Sentinelle)
 * à -40,78 LUFS (coup de mêlée), soit 31,58 LU d'écart — c'est la mesure de la remarque
 * utilisateur « surbouclier activation à peine audible » (il était 19 LU sous la carabine à
 * impulsion). Tous ont été renormalisés à -16 LUFS intégré, plafond -1 dBTP, par un GAIN
 * LINÉAIRE strict par fichier (mesure `ffmpeg -af loudnorm` en analyse, gain appliqué en
 * seconde passe) : ni compression, ni limiteur, ni ré-échantillonnage — durée, fréquence
 * d'échantillonnage, canaux et nombre d'échantillons INCHANGÉS, le timbre extrait du jeu
 * n'est pas retouché. Écart résiduel : 7,18 LU.
 *
 * QUINZE FICHIERS N'ATTEIGNENT PAS -16 LUFS, ET C'EST DE LA PHYSIQUE, PAS UN OUBLI : leur
 * facteur de crête (écart entre la crête vraie et le niveau intégré) va de 15 à 24 dB, si
 * bien qu'au plafond -1 dBTP ils plafonnent entre -17,2 et -23,1 LUFS. Les monter à la cible
 * exigerait un limiteur, c'est-à-dire écraser la transitoire d'un coup de feu ou la détente
 * d'une explosion. Ils sont donc laissés AU PLUS PRÈS, plafond atteint exactement : les armes
 * à salves (MA40 AR, MA5K Avenger, Bandit EVO, BR75, MK50 Sidekick, VK78 Commando, Fusil
 * traqueur, Disrupteur, Calcineur) et les impulsions isolées (lancers de fragmentation et de
 * plasma, coup de mêlée, capteur de menaces, explosions de fragmentation et de plasma).
 * Relever leur niveau intégré est une décision d'oreille sur le
 * TIMBRE, pas un réglage de gain — elle n'est pas prise ici.
 *
 * CE QUI DÉCLENCHE UN SON, ET RIEN D'AUTRE :
 *  - les TIRS du film (doc.shots), TOUS — voir la règle de densité ci-dessous ;
 *  - les KILLS du fil (source résolue : vignette OU weapon_key, cf. `killSound`) —
 *    l'horloge est celle du fil, DÉJÀ APPLIQUÉE quand ils arrivent ici, la même qui date le
 *    flash des fiches et l'effet de mort : un son qui partirait sur l'horloge brute
 *    sonnerait à côté de son image ;
 *  - les LANCERS de grenade (doc.grenades, l'auteur est écrit dans le film), par TYPE, et
 *    l'EXPLOSION de CHAQUE grenade à la FIN DE VOL de son projectile (décision utilisateur
 *    du 2026-08-18) — les deux formes du même objet, doctrine et tables : `grenadeSound.ts` ;
 *  - un kill À LA MÊLÉE sonne le coup qui a tué, et un kill À LA grenade l'explosion DE SON
 *    TYPE (c'est elle qui a tué, pas le geste du lancer) — les deux passent par la VIGNETTE
 *    de la source de dégât, pas par le weapon_key (cf. ci-dessous) ;
 *  - les POSES D'ÉQUIPEMENT (doc.equipmentPlacements, schéma 10 — mur, capteur) : le GESTE de
 *    pose sonne, à `t0`, ET SEULEMENT SI LA POSE EST UN DÉPLOIEMENT MESURÉ
 *    (`placementIsDeployedObject`, la MÊME règle que le calque). Rien ne sonne à la fin : la
 *    disparition d'un mur n'est pas un acte, c'est la fin d'une durée. Une famille sans fichier
 *    (les objets non identifiés, et toutes celles que le nommage ajoutera) reste muette ;
 *  - les ÉPISODES D'ÉQUIPEMENT ACTIF (doc.equipmentEpisodes, schéma 7 — camo et
 *    surbouclier, les deux familles MESURÉES) : le début d'épisode sonne l'activation,
 *    la fin sonne la désactivation SEULEMENT quand elle est MESURÉE (`endRead`) — un
 *    épisode fermé par la mort du porteur ne sonne pas de désactivation, rien ne l'a
 *    mesurée et le kill sonne déjà là. CHOIX DOCUMENTÉ pour le surbouclier : sa fin
 *    mesurée est l'ÉPUISEMENT (retour sous 100 %), et y jouer la désactivation est un
 *    choix de mise en scène — le gate d'écoute utilisateur tranchera ;
 *  - les TRACTIONS DE GRAPPIN (doc.grappleLines, schéma 8, lot G du 2026-08-20) : le TIR
 *    sonne à `t0`, un événement par traction. Rien ne sonne à `t1` : l'arrivée sur la
 *    trajectoire ferme la fenêtre du calque (`grappleLayer.ts`), ce n'est pas un geste ;
 *  - les ACTIONS D'OBJECTIF (doc.objectives — lot du 2026-08-26) : chacune sonne à sa frame,
 *    par NOM CANONIQUE DE STATISTIQUE et DANS LE CAMP DE SON AUTEUR. Détail, périmètre et
 *    silences : `objectiveSound.ts`.
 *
 * UNE TROISIÈME SOURCE, DEPUIS LE 2026-08-26 : la RE des banques Wwise
 * (`.ai/V7.5/RE_BANQUES_SONORES_NOMMEES_2026-08-26.md`) — banques NOMMÉES par hachage FNV-1 de
 * leur identifiant, puis leurs événements par la même voie, puis chaque geste reconstruit
 * couche par couche aux gains relevés. Elle apporte les sons d'objectif, et REMPLACE trois
 * fichiers du pack utilisateur À STEM CONSTANT (`grapple_fire`, `repulsor_kill`,
 * `repair_field_activate`) — même nom, source du jeu, et désormais plusieurs variantes tirées
 * à chaque lecture (`replaySoundVariants.ts`).
 *
 * POURQUOI LA VIGNETTE, ET PAS LE weapon_key, POUR LES GRENADES ET LA MÊLÉE. Le registre
 * d'armes ne porte NI la grenade à pointes NI la mêlée générique : la table de pont
 * `killicon/data/rules.tsv` leur donne une ligne SANS weapon_key (GGGL 3 et CLASSE MELEE).
 * Ce qui les distingue est donc la seule quantité que le backend publie pour elles — la
 * vignette du kill feed, qui voyage dans `weaponImageUrl`, et dont l'adapter dit lui-même
 * qu'elle « porte le sens » là où le nom propre n'existe pas (adapter_asset_urls.go). Les
 * quatre grenades ont chacune la leur (GGGL 0..3 -> killfeed-46..49) et la mêlée la
 * sienne (killfeed-65) : quatre explosions distinctes et un son de mêlée, sans jamais
 * DEVINER un type. Le garde-rail rejoue cette table contre `rules.tsv`.
 *
 * CE QUI RESTE MUET ICI, ET POURQUOI (mesuré, pas supposé) :
 *  - le COUP DE MÊLÉE NON FATAL (`Melee - Hit` du pack) : le document de rejeu ne porte
 *    aucun flux de dégâts ni d'impact — ses seuls événements datés sont les tirs, les
 *    lancers, les trajectoires et le fil des MORTS (replay/document.go). Le brancher
 *    demanderait de déduire un impact d'une baisse de bouclier, c'est-à-dire de l'inventer.
 *    Le fichier n'est donc pas livré : un asset que rien ne joue casserait le garde-rail ;
 *  - les 2 étiquettes de grenade AMBIGU sur 17 (`damagetag/data/labels.tsv` : effet
 *    générique traversant plusieurs entrées `gggl`) : non publiables, donc sans vignette,
 *    donc sans son DE KILL. Le type de la source de dégât n'est pas établi : silence, jamais
 *    l'explosion d'une voisine. La grenade concernée garde son explosion de FIN DE VOL —
 *    l'inconnue n'est pas la même : ce qui manque là est le lien mort -> grenade, pas le
 *    type du lancer, que le film publie avec son rang ;
 *  - un LANCER dont le film ne publie pas le rang, ou publie un rang hors des quatre types
 *    connus : ni geste ni explosion. Le type n'est pas établi, donc rien ne sonne — et
 *    `buildGrenadeRestFx` ne replie plus un rang absent sur 0 pour cette raison exacte.
 *
 * LA DENSITÉ N'EST PAS FILTRÉE — DÉCISION UTILISATEUR DU 2026-08-15, mot pour mot : « tu me
 * les mets TOUS autant que possible pour le moment, je verrai si c'est trop ensuite ». Il n'y
 * a donc ICI aucune règle éditoriale : ni cadence minimale entre deux tirs d'un même joueur,
 * ni priorité, ni quota. Le SEUL plafond est TECHNIQUE et il vit à la lecture
 * (`SOUND_MAX_VOICES`, replayAudio.ts) : au-delà de huit voix simultanées, les sources
 * supplémentaires sont refusées plutôt que d'empiler un mur de bruit. Mesures du 2026-08-15
 * (simulation à 1×, une voix tenue 1 s) : film témoin 000d5950 — 483 sons pour 483 tirs,
 * 46 voix refusées ; corpus des 23 artefacts locaux — 17 068 sons pour 17 904 tirs (95,3 %),
 * 4 897 voix refusées (28,7 % des sources). Le plafond MORD, et c'est lui seul qui borne.
 *
 * UN TIR SONNE L'ARME QUI L'A PRODUIT, JAMAIS UNE AUTRE. La jointure passe par la clé
 * canonique publiée avec le libellé (`weaponLabels[id].key`, posée à la requête par le
 * service) : sans clé, ou sans fichier pour cette clé, le tir est MUET. Les armes que le
 * REGISTRE ne nomme pas (Mutilator, tourelles, armes de PNJ) le restent par construction —
 * même règle que les libellés et les effets ; leurs sons extraits existent dans l'archive
 * du chantier, prêts si le registre les nomme un jour.
 *
 * UNE ARME = UN SON, MÊME QUAND SON TAG DÉCLARE DEUX MODES DE TIR — inventaire du
 * 2026-08-18 (lot R2-S, item S3), réponse à la question « il en manque ? ». Le tableau
 * « Weapon Fire Sounds » du tag `weap` déclare les modes d'une arme, et un VRAI mode a son
 * propre son de 1re personne (critère de la recette d'extraction, calibré sur les votes).
 * Passées à ce crible, les 26 armes du registre en déclarent UN SEUL, sauf DEUX :
 *  - Pistolet à plasma (`hinf_plasma_pistol`, weap 000041b7) : modes 1 et 3, le 3 étant le
 *    tir CHARGÉ (surcharge), avec ses événements propres `b4e0f7a3` (1P) et `e370a684` (3P) ;
 *  - Ravageur (`hinf_ravager`, weap 05b2c46c) : modes 1 et 2, le 2 étant le tir CHARGÉ.
 * Aucun des deux modes chargés n'est livré : le fichier joué est le mode 1 dans les deux cas
 * (Ravageur : le tir 3 coups, `bb31841b`). Le tir CONTINU, lui, n'est PAS un second mode de
 * tag : c'est la nature du mode unique du Rayon de Sentinelle (`a220122d`, rôle confirmé par
 * l'utilisateur), et le fichier livré pour cette arme est son tir COURT (`503433748`). Les
 * deux autres candidats au tir continu sont RÉFUTÉS par la mesure de cadence (registre des
 * reports) : Crémateur 1 400 ms d'écart médian entre deux tirs, Calcineur 800 ms.
 *
 * CE QU'ON N'A PAS FAIT, ET POURQUOI : aucun fichier n'a été créé pour ces modes. Le rejeu
 * ne SAIT pas de quel mode vient un tir — le film ne publie que l'arme, et les deux mesures
 * de qualification (jauge de charge, cadence) sont des NO-GO documentés. Livrer un son de
 * tir chargé le ferait donc sonner sur des tirs normaux : ce serait l'invention que ce
 * fichier refuse partout. Les rendus existent dans l'archive du chantier, prêts le jour où
 * une source qualifie le mode d'un tir.
 *
 * UN KILL DONT NI LA VIGNETTE NI LA CLÉ NE RÉPONDENT (véhicules, objets explosifs, dégât
 * global, armes hors registre) = SILENCE
 * PROPRE : jamais le son d'une arme voisine, même règle que les effets de rendu
 * (replay_labels.toml). La mêlée générique, elle, n'est PLUS de ceux-là : elle n'a pas de
 * clé mais elle a une vignette, et c'est par là qu'elle sonne.
 *
 * Pas de React, pas de Web Audio ici : logique pure, testée (replaySound.test.ts).
 * La lecture (AudioContext, enveloppe de gain) vit dans replayAudio.ts.
 */
import type { KillEvent } from '@/features/match-view/_momentum'

import { placementIsDeployedObject } from '../equipmentPlacementsLayer'
import { grenadeSoundEvents } from './grenadeSound'
import type { ReplayLocale } from '../i18n/i18n'
import type { ReplayKill } from '../killFeedLogic'
import { bombArmingSoundEvents } from '../bombCountdown'
import { objectiveSoundEvents, type ObjectiveSide } from './objectiveSound'
import { padSpawnSoundEvents } from './padSpawnSound'
import { weaponChangeSoundEvents } from './weaponChangeSound'
import { equipmentChangeSoundEvents } from './equipmentChangeSound'
import { abilityImpulseSoundEvents } from './abilityImpulseSound'
import { roundOverSoundEvents } from './roundOverSound'
import { skullSoundEvents } from './skullSound'
import { zoneSoundEvents } from './zoneSound'
import { frameToMs } from '../replayLogic'
import type { ReplayDocumentReady } from '../replayNormalize'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'
import { vehicleDestructionSound } from './vehicleDestructionSound'
import { vehicleShotSoundStem } from './vehicleShotSound'

export { pickVariantStem, SOUND_VARIANTS, stemsOf, type ReplaySoundEvent } from './replaySoundVariants'
export { OBJECTIVE_SOUND_STEMS, objectiveSoundStem, type ObjectiveSide } from './objectiveSound'
export { ZONE_SOUND_STEMS, zoneSoundEvents } from './zoneSound'
export { SKULL_SOUND_STEMS, skullSoundEvents } from './skullSound'
export { PAD_SPAWN_SOUND_STEM, EQUIPMENT_PAD_SPAWN_SOUND_STEM, padSpawnSoundEvents } from './padSpawnSound'
export { WEAPON_CHANGE_SOUND_STEMS, weaponChangeSoundEvents } from './weaponChangeSound'
export { EQUIPMENT_PICKUP_SOUND_STEM, equipmentChangeSoundEvents } from './equipmentChangeSound'
export { ABILITY_IMPULSE_SOUND_STEMS, abilityImpulseSoundEvents } from './abilityImpulseSound'
export { ROUND_OVER_SOUND_STEMS, roundOverSoundEvents } from './roundOverSound'

/**
 * Catégorie d'un son, pour le filtre du tiroir de réglages (phase 2, décision utilisateur
 * du 16/08) : ARMES (tirs et kills d'arme, même table — cf. WEAPON_SOUND_STEMS), GRENADES
 * (lancers ET explosions : c'est le même objet, sous deux formes), MÊLÉE (le seul coup
 * fatal sonné, `melee_kill`), ÉQUIPEMENTS (camouflage, surbouclier).
 */
export type SoundCategory = 'weapon' | 'grenade' | 'melee' | 'equipment' | 'objective'

/**
 * Son d'ARME par weapon_key -> stem de fichier sous static/sounds/{slug}/.
 *
 * UNE SEULE TABLE POUR LES TIRS ET LES KILLS D'ARME : c'est la même arme, donc le même
 * son. La séparer en deux ferait deux vérités à tenir à jour, et le jour où elles
 * divergeraient un tir et le kill qu'il produit ne sonneraient plus pareil.
 *
 * LES GRENADES N'Y SONT PAS, et ce n'est pas un oubli : le film ne publie aucun TIR de
 * grenade (mesure du 2026-08-15 sur les 23 artefacts locaux : 0 des 17 904 tirs porte une
 * clé de grenade). Elles ont leurs deux tables à elles, une par forme du même objet — le
 * geste (THROW_SOUND_STEMS) et la détonation (EXPLOSION_SOUND_STEMS) — et chacune sa propre
 * horloge.
 *
 * Une clé absente de cette table = silence propre.
 */
export const WEAPON_SOUND_STEMS: Readonly<Record<string, string>> = {
  hinf_ma40_ar: 'hinf_ma40_ar',
  hinf_br75: 'hinf_br75',
  hinf_cqs48_bulldog: 'hinf_cqs48_bulldog',
  hinf_cindershot: 'hinf_cindershot',
  hinf_vk78_commando: 'hinf_vk78_commando',
  hinf_disruptor: 'hinf_disruptor',
  hinf_energy_sword: 'hinf_energy_sword',
  hinf_gravity_hammer: 'hinf_gravity_hammer',
  hinf_heatwave: 'hinf_heatwave',
  hinf_hydra: 'hinf_hydra',
  hinf_mangler: 'hinf_mangler',
  hinf_needler: 'hinf_needler',
  hinf_plasma_pistol: 'hinf_plasma_pistol',
  hinf_pulse_carbine: 'hinf_pulse_carbine',
  hinf_ravager: 'hinf_ravager',
  hinf_s7_sniper: 'hinf_s7_sniper',
  hinf_sentinel_beam: 'hinf_sentinel_beam',
  hinf_shock_rifle: 'hinf_shock_rifle',
  hinf_sidekick: 'hinf_sidekick',
  hinf_skewer: 'hinf_skewer',
  hinf_m41_spnkr: 'hinf_m41_spnkr',
  hinf_stalker_rifle: 'hinf_stalker_rifle',
  // Les quatre que le pack initial n'avait pas — l'extraction du jeu les fournit.
  hinf_bandit: 'hinf_bandit',
  hinf_ma5k_avenger: 'hinf_ma5k_avenger',
  hinf_fuel_rod_spnkr: 'hinf_fuel_rod_spnkr',
  hinf_vestige_carbine: 'hinf_vestige_carbine',
}

/**
 * Son d'un KILL dont la SOURCE DE DÉGÂT n'a pas de weapon_key CÔTÉ REGISTRE D'ARMES —
 * grenades, mêlée, et désormais le répulseur : la vignette du kill feed -> son + catégorie.
 *
 * LA CLÉ EST LE STEM DE LA VIGNETTE (`killfeed-NN`), parce que c'est la seule quantité que
 * le backend publie pour ces sources : `killicon/data/rules.tsv` leur donne une ligne sans
 * weapon_key (GGGL 3 = grenade à pointes, CLASSE MELEE = geste partagé par tout l'arsenal,
 * NOM Repulsor = équipement hors registre d'armes — cf. ci-dessous). L'ordre des grenades
 * est celui du dépôt, établi par deux chaînes indépendantes : 0 Fragmentation, 1 Plasma,
 * 2 Dynamo, 3 Spike (RECETTE_LOADOUT §8, replay_labels.toml), et `rules.tsv` le reporte tel
 * quel sur les vignettes 46 à 49.
 *
 * CETTE TABLE PASSE AVANT WEAPON_SOUND_STEMS sur un kill : les trois grenades qui ONT une
 * clé (frag, plasma, dynamo) sonneraient sinon deux vérités concurrentes. Une vignette
 * absente de cette table (les armes du registre, les véhicules, les objets) retombe sur la
 * clé canonique, et une source sans vignette du tout reste muette.
 *
 * DEPUIS LE 2026-08-18, ELLE N'EST PLUS LA SEULE SOURCE D'EXPLOSION : la fin de vol de
 * chaque grenade en programme une (EXPLOSION_SOUND_STEMS). Ce que cette table garde en
 * propre, c'est la MÊLÉE — et, pour les grenades, la priorité au dédoublonnage.
 *
 * LA VALEUR PORTE LA CATÉGORIE AVEC LE STEM (type `KillSound`, défini plus bas), et ce
 * n'est plus un simple stem depuis le répulseur (lot R6, 2026-08-25) : avant lui, cette
 * table ne contenait QUE des grenades et LA mêlée, deux catégories qu'un simple test
 * `stem === 'melee_kill'` suffisait à distinguer dans `killSound`. Le répulseur casse cette
 * hypothèse binaire — sa catégorie est ARME (weapon), pas grenade — d'où le passage à une
 * table qui nomme sa catégorie EXPLICITEMENT plutôt que de la re-déduire d'un stem au point
 * de lecture (même principe que EQUIPMENT_SOUND_STEMS : une seule vérité, portée par la
 * table, jamais reconstruite ailleurs).
 *
 * LE RÉPULSEUR (`killfeed-56` -> `repulsor_kill`, catégorie `weapon`) REJOINT CETTE TABLE
 * ET NON `WEAPON_SOUND_STEMS`, alors qu'il a désormais un nom propre côté Go
 * (`damagetag/data/labels.tsv` : ARME SOUS_RESERVE "Repulsor") : ce nom n'a PAS de
 * `weapon_key` dans le registre d'armes (`killicon/data/rules.tsv`, règle NOM Repulsor,
 * colonne weapon_key VIDE — même situation que Mutilator et Sandwich, des armes HORS
 * registre). Sans weapon_key, `WEAPON_SOUND_STEMS` ne peut jamais répondre pour lui ; seule
 * la vignette le peut, exactement comme la mêlée et les grenades avant lui. Identification
 * du `jpt!` (07104b31) par RE hors ligne via `himap` (lot R6, 2026-08-25) : chaîne
 * `eqip 7ca85adc` (répulseur, `replay_labels.toml`) -> `sofa 6845f2b3` -> `eqip` frère
 * `1e79ebda` -> `jpt! 07104b31`, propulseur écarté (eqip distincts 0x430dda48/0xeef5d48d).
 * Détail : `damagetag/data/labels.tsv` et `killicon/data/rules.tsv`.
 *
 * Le garde-rail `replaySoundAssets.guard.test.ts` rejoue ces six clés contre `rules.tsv` :
 * un index d'atlas qui bougerait, ou une sixième source sans weapon_key, casse le test —
 * jamais l'écoute.
 */
export const KILL_SPRITE_SOUND_STEMS: Readonly<Record<string, KillSound>> = {
  'killfeed-46': { stem: 'explosion_frag', category: 'grenade' },
  'killfeed-47': { stem: 'explosion_plasma', category: 'grenade' },
  'killfeed-48': { stem: 'explosion_dynamo', category: 'grenade' },
  'killfeed-49': { stem: 'explosion_spike', category: 'grenade' },
  'killfeed-65': { stem: 'melee_kill', category: 'melee' },
  'killfeed-56': { stem: 'repulsor_kill', category: 'weapon' },
  // LES QUATRE BOBINES (2026-08-27). Un kill de bobine n'avait AUCUNE icône jusqu'ici, et la
  // cause était technique, pas doctrinale : la coupe à trois segments qui extrait la racine de
  // banque rendait `exp_single_small_{shock,plasma,kineticunsc,hardlight}` toutes identiques.
  // La racine LONGUE les distingue, et l'atlas porte leurs quatre vignettes depuis la passe de
  // nommage du 2026-08-09. Le son est l'événement à CINQ COUCHES SIMULTANÉES de chaque banque
  // — le seul des trois qu'elle porte qui soit l'explosion complète, les deux autres étant des
  // perspectives (RE du 2026-08-26, §5).
  //
  // CATÉGORIE ARME, comme le répulseur : une bobine tue par explosion, mais c'est une source de
  // dégât du décor, pas une grenade lancée — et le tiroir de réglages n'a pas de case « décor ».
  'killfeed-42': { stem: 'coil_shock', category: 'weapon' },
  'killfeed-43': { stem: 'coil_blast', category: 'weapon' },
  'killfeed-44': { stem: 'coil_fusion', category: 'weapon' },
  'killfeed-45': { stem: 'coil_plasma', category: 'weapon' },
}

/**
 * Sons d'ÉQUIPEMENT par famille d'épisode (`fam` du document) -> stems d'activation et de
 * désactivation. La clé est l'identifiant STABLE publié par l'artefact — une famille hors
 * table (un futur équipement non mesuré) reste muette, jamais le son d'une voisine.
 * Le garde-rail `replaySoundAssets.guard.test.ts` rejoue ces stems contre le dossier.
 */
export const EQUIPMENT_SOUND_STEMS: Readonly<
  Record<string, { activate: string; deactivate: string }>
> = {
  camo: { activate: 'camo_activate', deactivate: 'camo_deactivate' },
  overshield: { activate: 'overshield_activate', deactivate: 'overshield_deactivate' },
}

/**
 * Son d'une POSE d'équipement par FAMILLE (`family` du document, schéma 10) -> stem du fichier
 * joué à `t0`, l'instant du geste.
 *
 * UNE TABLE DISTINCTE DE `EQUIPMENT_SOUND_STEMS`, et ce n'est pas une duplication : les deux
 * parlent d'équipement, mais pas du même OBJET. L'une porte des ÉPISODES D'ÉTAT sur le
 * porteur (camouflage, surbouclier : un début ET une fin mesurés) ; celle-ci porte des OBJETS
 * POSÉS sur le terrain, dont seul le geste de pose est un événement — la disparition d'un mur
 * n'est pas un acte, c'est la fin d'une durée, et rien ne la sonne. Les fondre obligerait
 * chaque famille à déclarer une désactivation qu'elle n'a pas.
 *
 * Une famille absente de cette table est MUETTE (les objets non identifiés, `other`, et toutes
 * les familles que le lot de nommage ajoutera sans son) — jamais le son d'une voisine, même
 * règle que partout ailleurs ici. Le garde-rail `replaySoundAssets.guard.test.ts` rejoue ces
 * stems contre le dossier d'assets.
 *
 * LE TRAQUEUR PREND LE SON DU CAPTEUR, ET CE N'EST PAS « LE SON D'UNE VOISINE » (décision
 * utilisateur du 2026-08-18). La règle générale de ce fichier — une famille sans source reste
 * muette — vaut contre les EMPRUNTS ARBITRAIRES : donner au champ de réparation le son du mur
 * affirmerait un objet pour un autre. Ici, le traqueur de menaces et le capteur de menaces
 * sont le MÊME appareil dans le jeu, à un mode près : le capteur balaie sa zone en boucle, le
 * traqueur émet une impulsion unique. Le geste de POSE, lui, est identique — c'est le même
 * boîtier qu'on lâche. L'emprunt est donc adossé à une parenté, écrite ici et testée, et non
 * au fait qu'il « reste un son disponible ».
 *
 * LE CHAMP DE RÉPARATION SONNE DEPUIS LE 2026-08-18, ET SON SON VIENT DU JEU. Le relevé du
 * lot R3 concluait « la chaîne d'extraction ne connaît PAS le groupe `eqip` ». Elle le connaît
 * désormais (`PLAN_EQUIPEMENTS_MANQUANTS_SONS`, phase 3) : `cmd/weapon-sounds`, modes
 * `eqip-sons` puis `eqip-banks`, chaîne
 * `eqip -> effe -> snd! -> événement Wwise -> sbnk -> .wem`.
 *
 * POURQUOI LE MAILLON `effe` EST INDISPENSABLE, et pourquoi personne ne l'avait vu : la table
 * de dépendances d'un `eqip` ne porte que deux `snd!`, et ce sont les MÊMES pour 21 objets
 * d'équipement — du mur au surbouclier. C'est l'EFFET (`effe`) qui est propre à un geste.
 *
 * CE QUI DÉSIGNE CE FICHIER-LÀ, et non un autre des 35 `.wem` de la banque : l'`eqip`
 * `5c8e2316` (l'appareil du champ de réparation, second `eqip` du `sofa` `0e1febf8` =
 * `repair_field`) atteint la banque `5724312f`, que AUCUN autre `eqip` du jeu n'atteint, et
 * dont 4 `.wem` tombent dans le pack `sb_007_abl_repairfield.pck` — le jeu la nomme lui-même.
 * Son unique `snd!` (`22c2323a`) désigne deux événements qui rendent LES MÊMES trois `.wem`,
 * variantes uniformes d'un seul `RandomSequence` : il n'y a pas de choix à faire, seulement un
 * tirage.
 *
 * CE QUI A CHANGÉ LE 2026-08-27, ET C'EST UNE ERREUR CORRIGÉE, PAS UN RÉGLAGE. Le fichier
 * livré depuis le 2026-08-18 était le geste de POSE (`play_007_abl_repairfield_deploy_player`,
 * événement `8ed46d21`, 0,38 s) — un claquement d'objet lâché. L'ACTIVATION du champ est un
 * autre événement, `play_007_abl_repairfield_activate` (`c48cf171`, 3,26 s), et c'est celui
 * que l'utilisateur veut entendre quand un joueur pose le champ (décision du 2026-08-27,
 * après écoute de la planche). Les deux événements ne se distinguaient pas avant le nommage
 * des banques par hachage : la chaîne de tags menait aux deux sans dire lequel était lequel.
 * Les trois variantes livrées sont désormais les trois `.wem` de `c48cf171`
 * (143632032 / 222530989 / 640887009), gain de chemin +1 dB appliqué.
 *
 * ÉGALISATION : crête vraie plafonnée à -1,0 dBTP par gain LINÉAIRE (-0,8 / -1,0 / -2,0 dB),
 * plafond du lot R2-S, même convention que `repair_field_end`.
 *
 * LA BALISE DU TRANSLOCATEUR RESTE MUETTE, ET SON SILENCE A CHANGÉ DE RAISON. Sa banque est
 * TROUVÉE (`dcfaa487`, 70 `.wem`, atteinte par les deux seuls `eqip` `quantum_translocator` du
 * jeu) — mais elle porte ONZE gestes distincts (8 `snd!`, 11 événements, de 0,83 à 4,53 s), et
 * RIEN dans les tags ne dit lequel est la pose de la balise : les noms de champs d'un `eqip`
 * ne sont pas lisibles hors ligne. Choisir serait deviner. La recette des armes tranche ce cas
 * par le VOTE (« les votes priment sur tout critère », `RECETTE_SONS_ARMES` §5) : il faut une
 * écoute, pas une mesure de plus.
 *
 * RESTENT MUETS AUSSI, par décision et non par manque : le répulseur (que `PLACEMENT_RENDER`
 * ne DESSINE pas non plus — c'est une capacité qui agit sur son porteur, pas un objet posé ;
 * son activation n'est dans aucun canal, négatif mesuré sur neuf le 2026-09-03, et son seul
 * son reste celui de son kill), les deux bonus, et l'objet non identifié `other` (il a pourtant
 * SA banque, `92c830f5`, 38 `.wem` — mais un objet qu'on ne sait pas nommer n'a pas à
 * s'annoncer, et son dessin dépend d'une bascule que le son ne partage pas). LE GRAPPIN
 * (lot G, 2026-08-20) ET LE PROPULSEUR (2026-09-03) N'EN SONT PLUS : ils sonnent par LEURS
 * PROPRES tables — `GRAPPLE_SOUND_STEM` ci-dessous (`doc.grappleLines`, schéma 8) et
 * `ABILITY_IMPULSE_SOUND_STEMS` (`abilityImpulseSound.ts`, `doc.abilityImpulses`, schéma 38).
 * Ce ne sont pas des objets posés au sens de CETTE table-ci.
 */
export const EQUIPMENT_PLACEMENT_SOUND_STEMS_END: Readonly<Record<string, string>> = {
  // Le champ de réparation SONNE SA FIN depuis le 2026-08-26, et c'est une décision produit
  // de l'utilisateur (« y compris le champ de réparation qui disparaît »), pas une déduction.
  //
  // ELLE AMENDE UNE RÈGLE DE CE FICHIER, et il faut le dire plutôt que de la contourner en
  // silence : « rien ne sonne à la fin — la disparition d'un mur n'est pas un acte, c'est la
  // fin d'une durée ». Elle reste vraie POUR LE MUR ; le champ de réparation, lui, est un
  // émetteur qui s'ÉTEINT, et le jeu lui donne un événement propre pour ça
  // (`play_007_abl_repairfield_deactivate`, trois variantes).
  //
  // LA RÉSERVE D'HORLOGE EST ÉCRITE, parce qu'elle ne s'entend pas : `t1` d'une pose est sa
  // MISE AU REPOS mesurée, c'est-à-dire une BORNE INFÉRIEURE de sa disparition (correctif de
  // la revue des poses, 2026-08-17). Le son part donc au plus tôt à l'instant où le film
  // cesse de voir l'objet actif, jamais après. Aucune autre famille n'entre ici sans la même
  // vérification : une famille dont `t1` n'est pas mesuré sonnerait à un instant inventé.
  repair_field: 'repair_field_end',
}

export const EQUIPMENT_PLACEMENT_SOUND_STEMS: Readonly<Record<string, string>> = {
  wall: 'wall_activate',
  sensor: 'sensor_activate',
  // Même appareil que le capteur, un mode près : même geste de pose, donc même son.
  threat_seeker: 'sensor_activate',
  // Extrait du jeu le 2026-08-18 par la chaîne `eqip -> effe -> snd! -> sbnk` (cf. ci-dessus).
  repair_field: 'repair_field_activate',
  // LE TRANSLOCATEUR SONNE DEPUIS LE 2026-08-27, ET SA SOURCE N'EST NI LA CHAÎNE DE TAGS NI LE
  // HACHAGE : c'est l'UTILISATEUR qui a extrait le son du jeu et l'a désigné. Après deux jours
  // de recherche dans sa banque (`sb_007_abl_quantum`, 23 événements, 4 noms cassés) et l'échec
  // mesuré d'une comparaison spectrale contre une capture vidéo, c'est la voie qui a abouti —
  // et c'est la règle du chantier depuis le début : « les votes priment sur tout critère »
  // (`RECETTE_SONS_ARMES` §5). Le fichier est son extraction, ré-échantillonnée à 48 kHz et
  // plafonnée à -1 dBTP comme tous les autres, rien de plus.
  //
  // CE QUE LE SON EST, dans ses mots : « la première activation, la dépose de la faille
  // spatio-temporelle ». Ce n'est donc PAS une balise lancée — l'équipement se porte au poignet
  // et ouvre une faille sur la position exacte du joueur. Le nom de famille du calque
  // (`translocator_beacon`) et son libellé (« balise ») portent encore l'ancienne lecture ;
  // les corriger touche le manifeste ET le dessin, c'est un lot à part.
  //
  // L'USAGE (la téléportation elle-même) est livré sous `translocator_teleport` mais N'EST PAS
  // branché : le film ne publie aucun événement pour le retour, seulement la POSE. Le livrer
  // sans déclencheur en ferait un asset mort, que le garde-rail refuse à juste titre.
  translocator_beacon: 'translocator_deploy',
  // L ECRAN OCCULTANT, nomme le 2026-08-27. Il etait la DERNIERE famille dessinee sans son, et
  // ce qui le tenait muet n etait pas le son — `play_007_abl_shroud_deploy_player` est casse
  // depuis le 2026-08-26 — mais la FAMILLE : l objet s appelait `other` au manifeste, faute
  // d une voie de nommage. Sa banque sonore l a donnee (`sb_007_abl_shroud`).
  shroud_screen: 'shroud_deploy',
}

/**
 * Son du TIR de grappin (doc.grappleLines, schéma 8, lot G du 2026-08-20) — UN SEUL stem,
 * aucune famille : chaque traction sonne le même tir, quel que soit le slot. Catégorie
 * ÉQUIPEMENT (le grappin est un objet d'équipement du joueur, au même titre que camouflage
 * et surbouclier) — cf. `buildSoundTimeline`.
 *
 * SOURCE DISTINCTE DES DEUX DÉCRITES EN TÊTE DE FICHIER : une archive utilisateur de sons
 * extraits du jeu (nov. 2021), variante « Activate (Hit Short) » — décision superviseur, lot
 * G. 1,687 s, au-dessus de la coupe des armes (1,2 s) : gardée entière, comme les
 * équipements. ÉGALISATION : gain LINÉAIRE de +15,07 dB (mesuré -31,07 LUFS / -16,95 dBTP en
 * entrée), résultat -16,01 LUFS / -1,88 dBTP — cible et plafond du lot R2-S atteints.
 */
export const GRAPPLE_SOUND_STEM = 'grapple_fire'

/**
 * LES CATÉGORIES QUE L'UTILISATEUR PEUT COUPER. Elles étaient cinq (phase 2, décision du
 * 16/08) ; « Objectifs » en est sorti le 2026-09-02 sur retour utilisateur — « c'était pas
 * clair du tout » : dans une liste de bascules, « Objectifs » se lit comme un CALQUE, pas comme
 * une famille de sons. Les sons d'objectif jouent désormais toujours.
 *
 * CE N’EST PAS LA LISTE DES CATÉGORIES : le type SoundCategory en compte toujours cinq, car
 * classer un son reste utile. C'est la liste de ce qui se RÈGLE. Une préférence stockée qui
 * nommerait encore « objective » ne se relit donc plus, et la catégorie se rallume d'elle-même.
 */
/**
 * Le sous-type des categories REGLABLES. Il existe pour que le compilateur tienne la paire :
 * une categorie retiree d ici doit perdre son libelle i18n, et reciproquement. Sans lui, le
 * retrait de « Objectifs » compilait avec un libelle manquant a l execution.
 */
export type TogglableSoundCategory = Exclude<SoundCategory, 'objective'>

export const SOUND_CATEGORIES: readonly TogglableSoundCategory[] = [
  'weapon',
  'grenade',
  'melee',
  'equipment',
]

/** Filtre par catégorie : une entrée par catégorie, `true` = catégorie audible. */
export type SoundCategoryFilter = Readonly<Record<SoundCategory, boolean>>

/** Le comportement D'AUJOURD'HUI, inchangé par défaut : tout sonne. */
export const SOUND_CATEGORIES_DEFAULT: SoundCategoryFilter = {
  weapon: true,
  grenade: true,
  melee: true,
  equipment: true,
  objective: true,
}

/**
 * shotSoundStem — le fichier d'un TIR, ou undefined pour le silence.
 *
 * Le film date le tir et nomme son arme par un identifiant ; la clé canonique arrive avec
 * le libellé (`weaponLabels[id].key`). Aucune direction n'est requise — c'est tout l'intérêt
 * du son : il n'a besoin QUE de l'instant (demande utilisateur du 2026-08-15).
 *
 * DEUX JOINTURES DEPUIS LE LOT VÉHICULES (2026-09-04), DANS CET ORDRE : le registre d'armes
 * d'abord (les armes de joueur, tirées à pied OU depuis un siège), puis les armes DE VÉHICULE
 * — leurs identifiants (`0x<weap>00000000`) sont ABSENTS de `weaponLabels`, c'est la table de
 * `vehicleShotSound.ts` qui les nomme. Aucune des deux ne répond = silence propre, inchangé.
 */
export function shotSoundStem(doc: ReplayDocumentReady, weaponID: string | undefined): string | undefined {
  if (!weaponID) return undefined
  const key = doc.weaponLabels?.[weaponID]?.key
  if (key) return WEAPON_SOUND_STEMS[key]
  return vehicleShotSoundStem(weaponID)
}

/**
 * killSourceSpriteStem — le stem de la vignette d'une source de dégât, lu dans son URL.
 *
 * Le backend compose `/static/weapons-assets/{slug}/jeu/killfeed-NN.png` (adapter_asset_urls.go,
 * `static.URL` : ni empreinte ni paramètre de version). Le stem est donc le nom de fichier
 * sans son extension. Rien à extraire = chaîne vide, jamais une supposition.
 */
export function killSourceSpriteStem(imageURL: string | undefined): string {
  const path = (imageURL ?? '').split('?')[0]
  const file = path.slice(path.lastIndexOf('/') + 1)
  return file.endsWith('.png') ? file.slice(0, -'.png'.length) : ''
}

/** Un son de kill résolu, avec la catégorie qui a répondu — sert le filtre par catégorie. */
interface KillSound {
  stem: string
  category: Extract<SoundCategory, 'weapon' | 'grenade' | 'melee'>
}

/**
 * killSound — DEUX JOINTURES, DANS CET ORDRE : la vignette de la source d'abord (c'est elle
 * qui sait distinguer les quatre grenades, la mêlée et le répulseur, que le registre d'armes
 * ne nomme pas), la clé canonique ensuite (les armes DU registre). Aucune des deux ne répond
 * = silence propre. Elle rend le stem ET la catégorie, parce que `buildSoundTimeline` a
 * besoin des deux : le fichier à jouer, et la catégorie que le filtre du tiroir peut couper.
 *
 * PLUS DE TERNAIRE MELEE/GRENADE ICI (lot R6, 2026-08-25) : tant que
 * `KILL_SPRITE_SOUND_STEMS` ne portait que des grenades et LA mêlée, un test
 * `stem === 'melee_kill'` suffisait à trancher la catégorie. Le répulseur (catégorie ARME)
 * a cassé cette hypothèse binaire — la table nomme désormais sa catégorie EXPLICITEMENT
 * (type `KillSound`), et cette fonction la restitue telle quelle : une seule vérité, portée
 * par la table, jamais re-devinée ici à partir du stem.
 *
 * EXPORTÉE POUR ÊTRE TESTÉE À L'UNITÉ, et c'est son seul appelant de production
 * (`buildSoundTimeline`) qui lui donne son sens. La façade `killSoundStem`, qui n'en
 * exposait que le stem, a été SUPPRIMÉE le 2026-08-16 : le filtre par catégorie lui a pris
 * son dernier appelant de production, et un export qui ne survit que par ses tests est le
 * code mort que la règle 7 de CLAUDE.md interdit. Ses cas sont testés dans
 * `replaySound.test.ts`, augmentés de la catégorie — l'information neuve du filtre.
 */
export function killSound(kill: Pick<KillEvent, 'weaponKey' | 'weaponImageUrl'>): KillSound | undefined {
  const sprite = killSourceSpriteStem(kill.weaponImageUrl)
  const bySprite = sprite ? KILL_SPRITE_SOUND_STEMS[sprite] : undefined
  if (bySprite) return bySprite
  const byKey = kill.weaponKey ? WEAPON_SOUND_STEMS[kill.weaponKey] : undefined
  return byKey ? { stem: byKey, category: 'weapon' } : undefined
}

/**
 * buildSoundTimeline précalcule la piste sonore du document : TIRS datés par leur frame de
 * film + kills DÉJÀ RECALÉS par le fil (`killsOfFeed` — une seule horloge, et un seul
 * recalage : ni ce module ni l'effet de mort ne le rejouent, 2026-09-05 J2) + lancers de
 * grenade et explosions de fin de vol. Triée chronologiquement, construite une fois.
 *
 * LES TIRS ET LES KILLS COEXISTENT SANS DÉDUPLICATION, et c'est voulu : le tir qui tue est
 * un événement du film, la mort qu'il cause en est un autre, daté par le fil. Les fondre
 * ferait disparaître l'un des deux sur une horloge où ils ne tombent pas au même instant.
 *
 * `categories` (tiroir de réglages, phase 2) RETIRE À LA CONSTRUCTION les catégories
 * coupées par l'utilisateur — jamais en aval, dans le lecteur : une catégorie coupée ne
 * précharge même pas ses fichiers. Par défaut, tout sonne (SOUND_CATEGORIES_DEFAULT) :
 * un appelant qui ne passe pas ce 4e paramètre garde le comportement d'aujourd'hui, à
 * l'identique — c'est le cas de tous les appels existants avant cette table.
 */
export function buildSoundTimeline(
  doc: ReplayDocumentReady,
  kills: readonly ReplayKill[],
  categories: SoundCategoryFilter = SOUND_CATEGORIES_DEFAULT,
  sideOfXuid?: (xuid: string) => ObjectiveSide,
  // Le camp ALLIÉ, en numéro d'équipe — la clé des sons d'ÉTAT DE ZONE, qui joignent sur le
  // propriétaire d'une zone et non sur le xuid d'un joueur (`zoneSound.ts`). `null` = camp non
  // résolu : capture en cours et tics de score se taisent, la nouvelle colline sonne quand même.
  allyTeam: number | null = null,
  // La LANGUE DE L'INTERFACE — la seule entrée locale-aware de la piste : le son « manche
  // terminée » est une VOIX d'annonceur (`roundOverSound.ts`). Absente (anciens appels, mesure de
  // disponibilité), ce son se tait : le rejeu ne devine pas une langue.
  locale?: ReplayLocale,
): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  if (categories.weapon) {
    for (const s of doc.shots) {
      const stem = shotSoundStem(doc, s.w)
      if (stem) out.push(soundEvent(frameToMs(s.t, doc), stem))
    }
    // Les RAMASSAGES ET LÂCHERS D'ARME (schéma 25) : datés par `weaponChanges`, plus rien à
    // déduire — c'est ce canal qui a remplacé la règle « au premier tir » retirée le même
    // jour. Doctrine, sons et choix du `swapped` : `weaponChangeSound.ts`.
    out.push(...weaponChangeSoundEvents(doc))
    // La DESTRUCTION d'un véhicule (schéma 30, lot du 2026-09-05) : le MÊME signal que
    // l'effet visuel — `end === VEHICLE_END_DESTROYED` ET `tEnd` publié, rien d'autre.
    // INERTE AUJOURD'HUI : tout artefact existant publie `end: "unknown"`, donc zéro
    // événement — le son est prêt, il n'invente pas la mesure. Un bruitage one-shot,
    // catégorie ARMES du tiroir (comme les bobines), jamais le bus moteur. Doctrine,
    // jeux de sons dédupliqués et jointure famille -> set : `vehicleDestructionSound.ts`.
    for (const v of doc.vehicles) {
      const boom = vehicleDestructionSound(v)
      if (boom) out.push(soundEvent(frameToMs(boom.frame, doc), boom.stem))
    }
  }
  // Les explosions programmées PAR UN KILL, retenues au passage : ce sont elles qui
  // dédoublonnent les fins de vol, et elles doivent donc être connues avant.
  const killExplosions: ReplaySoundEvent[] = []
  if (kills.length > 0 && doc.tracks.length > 0) {
    for (const k of kills) {
      const snd = killSound(k)
      if (!snd || !categories[snd.category]) continue
      const ev = soundEvent(k.replayMs, snd.stem)
      out.push(ev)
      if (snd.category === 'grenade') killExplosions.push(ev)
    }
  }
  // Le filtre « grenades » couvre les DEUX formes de l'objet, comme depuis le 16/08 : couper
  // la catégorie coupe les lancers ET les explosions, y compris celles des kills ci-dessus.
  if (categories.grenade) out.push(...grenadeSoundEvents(doc, killExplosions))
  // Les épisodes d'équipement actif : l'activation au début, la désactivation SEULEMENT
  // sur une fin MESURÉE (`endRead`) — un épisode fermé par la mort ne sonne rien de plus,
  // le kill sonne déjà là et rien n'a mesuré une désactivation.
  if (categories.equipment) {
    for (const e of doc.equipmentEpisodes) {
      const stems = EQUIPMENT_SOUND_STEMS[e.fam]
      if (!stems) continue
      out.push(soundEvent(frameToMs(e.t0, doc), stems.activate))
      if (e.endRead) out.push(soundEvent(frameToMs(e.t1, doc), stems.deactivate))
    }
    // Les POSES d'équipement (schéma 10) : le GESTE sonne, à `t0`, et lui seul. La fin de vie
    // d'un mur n'est pas un acte — c'est la fin d'une durée mesurée, et rien ne la sonne.
    //
    // LE FILTRE EST CELUI DU CALQUE, ET IL EST PARTAGÉ EXPRÈS. Sur les 11 films calibrés,
    // 88,6 % des poses sont des objets LÂCHÉS À LA MORT du porteur : les sonner ferait partir
    // un « mur déployé » à chaque mort tenant un mur — 91 fois sur 222 poses de mur, 106 fois
    // sur 155 poses de capteur. Et un mur réellement déployé produit DEUX poses (l'appareil et
    // ses panneaux) : sans la règle des panneaux, il sonnerait deux fois.
    for (const p of doc.equipmentPlacements) {
      if (!placementIsDeployedObject(p)) continue
      const stem = EQUIPMENT_PLACEMENT_SOUND_STEMS[p.family]
      if (stem) out.push(soundEvent(frameToMs(p.t0, doc), stem))
      // La FIN de la pose, pour les seules familles qui ont un son d'extinction propre dans
      // le jeu (`EQUIPMENT_PLACEMENT_SOUND_STEMS_END`) : elle suit la même porte que le début
      // — un objet lâché à la mort ne s'est pas éteint, il est tombé.
      const fin = EQUIPMENT_PLACEMENT_SOUND_STEMS_END[p.family]
      if (fin) out.push(soundEvent(frameToMs(p.t1, doc), fin))
    }
    // Les TRACTIONS de grappin (schéma 8) : le TIR sonne à `t0`, un événement par traction —
    // aucune fin sonnée, `t1` ne fait que fermer la fenêtre du calque (`grappleLayer.ts`).
    for (const g of doc.grappleLines) {
      out.push(soundEvent(frameToMs(g.t0, doc), GRAPPLE_SOUND_STEM))
    }
    // LE RAMASSAGE SUR SOCLE A ÉTÉ RETIRÉ LE 2026-08-30, et il faut dire pourquoi plutôt que de
    // laisser un trou muet. Il sonnait au PREMIER TIR d'une famille d'arme de socle, faute de
    // canal qui date le ramassage (`padPickups` : médiane de 20,00 s entre `tLow` et `tHigh` ;
    // loadouts sur la même grille d'images-clés, 0 sur 597 datés à moins de 5 s). Les mesures
    // étaient justes, la conclusion ne l'était pas : la doctrine de cette chaîne est que le rejeu
    // SE TAIT plutôt que de deviner, et on avait déplacé le son sur un AUTRE geste — il partait
    // en même temps qu'un tir, il partait pour une arme prise au sol ou sur un mort, et il ne
    // partait jamais pour une arme ramassée sans être tirée. Décision utilisateur du 2026-08-30 :
    // se taire jusqu'à ce qu'un canal DATE le ramassage. Le son, lui, reste identifié
    // (`play_007_abl_shared_pickup`, événement `c73036e4`) et audible sur la planche
    // `3c84fab7-5e36-4777-a2d9-bd1c90b08f65` ; il n'est plus livré tant que rien ne le déclenche.
    //
    // L'APPARITION SUR SOCLE — arme OU équipement, chacun son son : le calque DATE l'instant
    // (`weaponPads[].spawns`) et publie la famille qui les sépare. Détail : `padSpawnSound.ts`.
    out.push(...padSpawnSoundEvents(doc))
    // LE RAMASSAGE D'ÉQUIPEMENT (schéma 26) : daté par `equipmentChanges`. Le son attendait cet
    // instant depuis le 2026-08-27 ; la consommation reste muette (elle sonne déjà par sa
    // famille). Doctrine : `equipmentChangeSound.ts`.
    out.push(...equipmentChangeSoundEvents(doc))
    // L'USAGE D'UNE CAPACITÉ QUI POUSSE SON PORTEUR (schéma 38) : le propulseur, daté par le
    // film et validé 5/5 contre un relevé Theater. Un son par impulsion publiée, à sa frame ;
    // aucune fin sonnée — le geste EST une impulsion. Table par famille : `abilityImpulseSound.ts`.
    out.push(...abilityImpulseSoundEvents(doc))
  }
  // Les ACTIONS D'OBJECTIF : chacune sonne à sa frame, dans le camp de son auteur. Sans
  // résolveur de camp (appelant qui n'a pas le tableau de score), les seules actions qui
  // sonnent sont celles sans variante d'équipe — le rejeu ne devine jamais un camp.
  if (categories.objective) {
    out.push(...objectiveSoundEvents(doc, sideOfXuid))
    // Les sons d'ÉTAT DE ZONE : capture en cours, domination, déplacement de la colline. Ils ne
    // viennent d'aucun joueur — leur source est `doc.zoneStates`, pas `doc.objectives` — mais
    // ils appartiennent à la même catégorie du tiroir, et se coupent donc avec elle.
    out.push(...zoneSoundEvents(doc, allyTeam))
    // LE SON DE LA BOMBE ARMÉE (Assaut, schéma 29) : un événement au début de chaque mèche.
    // Sa source est `doc.bombArmings`, pas `doc.objectives` — le canal de l'anneau ti=12 ne
    // nomme aucun joueur. Le stem est celui de la NOUVELLE COLLINE, emprunté par référence
    // (décision utilisateur du portage) : doctrine et provenance dans `bombCountdown.ts`.
    out.push(...bombArmingSoundEvents(doc))
    // LES SONS DU CRANE d'Oddball : prise et chute. Leur source est `doc.skullCarries`
    // (schema 23), pas `doc.objectives` — le nommage statborg ne couvre pas Oddball. Ils
    // n'ont pas de camp : la banque du mode ne porte pas de variante `_team`/`_enemy` sur
    // ces gestes. Doctrine et inventaire de ce qui reste dehors : `skullSound.ts`.
    out.push(...skullSoundEvents(doc, allyTeam))
    // LE SON « MANCHE TERMINÉE », sur la même mesure que le message inter-manche : la bascule
    // d'une manche à la suivante (`roundTransitions`, roundsLogic), datée par le calque de score.
    // CÂBLÉ depuis le 2026-08-28 (l'asset annonceur FR/EN a été fourni) — exactement comme le
    // stub qui vivait ici l'annonçait. Sans `locale`, il se tait : une voix a une langue, et le
    // rejeu n'en devine pas une. Table de stem et doctrine : `roundOverSound.ts`.
    if (locale) out.push(...roundOverSoundEvents(doc, locale))
  }
  return out.sort((a, b) => a.ms - b.ms)
}
