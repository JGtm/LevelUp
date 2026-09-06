/**
 * replayReadyTypes.ts — LES TYPES `*Ready` : la FORME du document de rejeu une fois passé la
 * frontière, champ par champ, avec la raison de chaque comblement.
 *
 * POURQUOI CE FICHIER EXISTE : DÉPLACEMENT, PAS RÉÉCRITURE. `replayNormalize.ts` franchissait
 * le seuil de 500 lignes (511) après l'arrivée des calques d'Assaut. Les types en sortent tels
 * quels, sans une ligne changée ; `replayNormalize.ts` garde LA FONCTION — c'est-à-dire la
 * frontière elle-même — et re-publie ces types pour que ses appelants n'aient rien à changer.
 *
 * LA FRONTIÈRE EST ICI POUR LE TYPE, LÀ-BAS POUR LA VALEUR : ce fichier dit ce que le rendu a
 * le DROIT de lire (aucun tableau nullable), `replayNormalize.ts` dit comment on l'obtient.
 * Les deux se lisent ensemble ; la doctrine complète est en tête de `replayNormalize.ts`.
 */
import { type ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'
import type {
  ReplayDocument,
  ReplayFlagCarry,
  ReplayGrenadeRead,
  ReplayObjectiveObjectLife,
  ReplayInventory,
  ReplayLoadout,
  ReplayProjectile,
  ReplaySurface,
  ReplayTrack,
  ReplayVehicleRide,
  ReplayVehicleTrack,
  ReplayWeaponPad,
  ReplayZoneState,
} from '@/lib/api/types'

/** Un sommet d'emprise orientée : le `[2]float32` du Go. */
export type ReplayXY = [number, number]

/** Un pas de trajectoire de projectile : le `[3]float32` du Go, soit [dt, x, y]. */
export type ReplayStep = [number, number, number]

/** Rend NON NULLABLES (et présents) les champs `K` de `T` — les tableaux du contrat. */
type Filled<T, K extends keyof T> = Omit<T, K> & { [P in K]-?: NonNullable<T[P]> }

export type ReplayTrackReady = Filled<ReplayTrack, 'points'>
type ReplayLoadoutReady = Filled<ReplayLoadout, 'w'>
export type ReplayInventoryReady = Filled<ReplayInventory, 'am' | 'g'>
export type ReplayGrenadeReadReady = Filled<ReplayGrenadeRead, 'g'>
export type ReplaySurfaceReady = Omit<ReplaySurface, 'poly'> & { poly: ReplayXY[] }
export type ReplayProjectileReady = Omit<ReplayProjectile, 'p'> & { p: ReplayStep[] }
/**
 * ReplayWeaponPadReady — un socle dont les DEUX tableaux imbriqués sont comblés.
 *
 * POURQUOI IL FALLAIT UN TYPE DE PLUS. `_ListeExhaustive` ne voit que les tableaux nullables de
 * la RACINE du document : elle avait donc bien exigé `weaponPads`, et la frontière comblait le
 * tableau de tête — mais `spawns` et `presence`, nullables eux aussi, traversaient tels quels.
 * Un socle sans apparition répliquée arrivait au rendu avec `spawns: null`, et `pad.spawns.map`
 * tombait à l'exécution. Même patron que `tracks` et `structure` (correctif de revue du
 * 2026-08-17).
 *
 * `cycle` N'EST PAS COMBLÉ, et ce n'est pas un oubli : ce n'est pas un tableau mais une mesure
 * qui peut ne pas exister (24 socles sur 57 seulement en portent une). Le combler par un objet
 * vide inventerait un cycle de zéro seconde — l'absence est la donnée.
 */
export type ReplayWeaponPadReady = Filled<ReplayWeaponPad, 'spawns' | 'presence'>
/**
 * ReplayFlagCarryReady — la vie d'un drapeau dont les intervalles sont comblés.
 *
 * MÊME PATRON QUE `weaponPads` : le tableau de tête et le tableau IMBRIQUÉ sont tous deux
 * nullables au contrat, et un drapeau qui arriverait avec `spans: null` ferait tomber le calque
 * à l'exécution, pas à la compilation.
 */
export type ReplayFlagCarryReady = Filled<ReplayFlagCarry, 'spans'>
/**
 * ReplayObjectiveObjectReady — une vie libre d'objet d'objectif, trajectoire comblée.
 *
 * MÊME PATRON QUE `flagCarries` : le tableau de tête et le tableau IMBRIQUÉ (`pts`) sont tous
 * deux nullables au contrat, et une vie qui arriverait avec `pts: null` ferait tomber le calque
 * à l'exécution, pas à la compilation.
 */
export type ReplayObjectiveObjectReady = Filled<ReplayObjectiveObjectLife, 'pts'>
/**
 * ReplayZoneStateReady — l'état d'une zone dont les intervalles ET la jauge en direct sont comblés.
 *
 * MÊME PATRON QUE `flagCarries` et `weaponPads` : le tableau de tête et les tableaux IMBRIQUÉS
 * (`spans`, et `gauge` depuis le schéma 18) sont tous nullables au contrat, et une zone qui
 * arriverait avec `spans: null` ferait tomber le calque à l'exécution, pas à la compilation.
 *
 * `gauge` SE COMBLE À VIDE, et ce n'est pas inventer une valeur : pour le rendu, « artefact de
 * schéma <= 17 » (le champ n'existe pas), « série absente » (aucune rampe sur cette zone) et
 * « série vide » disent la même chose — AUCUN ARC. Le sommet statique n'y supplée pas : il se
 * lisait comme une jauge, et c'est ce que le schéma 18 corrige (décision du plan, lot C-ter).
 */
export type ReplayZoneStateReady = Filled<ReplayZoneState, 'spans' | 'gauge'>
/**
 * ReplayVehicleTrackReady — la vie d'un véhicule dont les DEUX tableaux imbriqués sont comblés.
 *
 * MÊME PATRON QUE `weaponPads` : `samples` (la trajectoire) et `rides` (les épisodes
 * d'occupation) sont nullables au contrat, et un véhicule qui arriverait avec `samples: null`
 * ferait tomber le calque à l'exécution — pas à la compilation. `spawn` N'EST PAS comblé : ce
 * n'est pas un tableau mais un objet optionnel (absent quand le record de création n'a pas été
 * lu), et un objet vide inventerait une naissance que le film ne montre pas.
 */
export type ReplayVehicleTrackReady = Omit<Filled<ReplayVehicleTrack, 'samples'>, 'rides'> & {
  rides: ReplayVehicleRideReady[]
}
/**
 * ReplayVehicleRideReady — un épisode d'occupation dont la SÉRIE DE VISÉE est comblée (schéma 39).
 *
 * TROISIÈME NIVEAU DE COMBLEMENT, et le premier du document : `aim` est un tableau nullable
 * imbriqué DANS un tableau imbriqué. Le combler ici plutôt que de le laisser passer garde la
 * garde de contrat (`replayContract.test.ts`) au même régime pour tout l'artefact — un tableau
 * nullable de plus, à n'importe quelle profondeur, doit être comblé ou justifié, jamais oublié.
 * Le coût est nul en pratique : une poignée d'épisodes par match, comblés une fois au chargement.
 */
export type ReplayVehicleRideReady = Filled<ReplayVehicleRide, 'aim'>

/**
 * ReplayBombStatsReady — le bloc des statistiques d'Assaut, son tableau comblé. L'OBJET reste
 * optionnel au document (cf. `ReplayDocumentReady.bombStats`) ; ce type-ci ne dit que ceci :
 * s'il est là, `players` l'est aussi, fût-il vide. `coverage`, lui, n'est pas un tableau et ne
 * se comble pas — il est toujours écrit par le producteur.
 */
export type ReplayBombStatsReady = Filled<NonNullable<ReplayDocument['bombStats']>, 'players'>

/**
 * ReplayDocumentReady — le document tel que le rendu a le droit de le lire : chaque
 * tableau est présent, jamais null, et les coordonnées ont retrouvé leur arité.
 */
export type ReplayDocumentReady = Omit<
  ReplayDocument,
  | 'abilities'
  | 'abilityCharges'
  | 'abilityImpulses'
  | 'bombArmings'
  | 'bombCarries'
  | 'bombEvents'
  | 'bombStats'
  | 'equipmentChanges'
  | 'equipmentEpisodes'
  | 'equipmentPlacements'
  | 'flagCarries'
  | 'geometry'
  | 'grappleLines'
  | 'groundWeapons'
  | 'grenadeLabels'
  | 'grenadeReads'
  | 'grenades'
  | 'inventory'
  | 'loadouts'
  | 'neutralDeaths'
  | 'objectiveObjects'
  | 'objectives'
  | 'padPickups'
  | 'projectiles'
  | 'roster'
  | 'scoreTimeline'
  | 'shots'
  | 'skullCarries'
  | 'structure'
  | 'tracks'
  | 'translocations'
  | 'vehicles'
  | 'vipCrown'
  | 'pickups'
  | 'weaponChanges'
  | 'weaponPads'
  | 'zoneStates'
> & {
  abilities: NonNullable<ReplayDocument['abilities']>
  /**
   * LES IMPULSIONS DE CAPACITÉ (schéma 38) : une entrée PLATE par geste — (t, slot, family) —
   * l'usage MESURÉ du propulseur, daté par le corps `tag == 1` des composants i57/i59 du film
   * (le même dont le tag 3 porte le grappin) et ATTRIBUÉ par le rang de capacité lu dans la
   * MÊME VIE et antérieurement. Aucun seuil de vitesse, aucune heuristique.
   *
   * CE CALQUE NE COUVRE PAS TOUS LES ÉQUIPEMENTS : seules les familles que le titre déclare
   * MESURÉES y entrent (aujourd'hui le propulseur, et lui seul — le répulseur n'est PAS dans
   * ce canal, négatif mesuré). `coverage.abilityImpulses.otherFamily` compte ce qui est
   * écarté. Vide = artefact antérieur au schéma 38, film sans propulseur, ou palette non
   * classée — `coverage.abilityImpulses` distingue les trois.
   */
  abilityImpulses: NonNullable<ReplayDocument['abilityImpulses']>
  /**
   * LES CHARGES D'ÉQUIPEMENT RESTANTES (schéma 38 enrichi, lot P5) : une entrée PLATE par
   * lecture — (t, slot, family, charges) — le compteur de charges entières transmis AU
   * CHANGEMENT par le composant i56 du film (quartet haut de la valeur 7 bits, rapport R11)
   * et ATTRIBUÉ par le rang de capacité de la MÊME VIE. Ce sont les LECTURES, jamais un
   * compte d'usages dérivé (une baisse peut valoir plusieurs usages), et rien n'est transmis
   * au ramassage : la première lecture est ce qui reste APRÈS le premier usage.
   *
   * CE CALQUE NE COUVRE PAS TOUS LES ÉQUIPEMENTS : seules les familles que le titre déclare
   * MESURÉES y entrent (le grappin et le propulseur — le répulseur n'arme jamais i56,
   * négatif mesuré). `coverage.abilityCharges` porte l'entonnoir complet. Vide = artefact
   * antérieur à ce lot, film sans lecture armée, ou palette non classée —
   * `coverage.abilityCharges` distingue les trois.
   */
  abilityCharges: NonNullable<ReplayDocument['abilityCharges']>
  /**
   * L'ARMEMENT DE LA BOMBE d'Assaut (schéma 29) : le début du hold, l'instant armé et la
   * mèche (fuseMs) — le compte à rebours se dessine sur [t, t + fuseMs] sans autre donnée.
   * Vide = artefact antérieur au schéma 29, mode non couvert (jamais One Bomb), ou film
   * retenu par la confrontation locale — `coverage.bombArmings` distingue les silences.
   */
  bombArmings: NonNullable<ReplayDocument['bombArmings']>
  /**
   * LES PÉRIODES DE PORTAGE DE LA BOMBE d'Assaut (schéma 30) : une entrée plate par période
   * (xuid, t0, t1, closed), le patron de `skullCarries` sur le canal des armes tenues. Vide =
   * artefact antérieur au schéma 30, ou film hors de la famille bomb — `coverage.bombCarries`
   * distingue les deux. La bombe portée est à la position de son porteur ; AU SOL, le rendu la
   * dérive des périodes + pistes (dernier point du lâcheur — aucun canal mesuré côté Go).
   */
  bombCarries: NonNullable<ReplayDocument['bombCarries']>
  /**
   * LES FAITS DATÉS DE LA BOMBE (schéma 39) : armements et explosions sur l'horloge du film,
   * chacun avec la RÈGLE qui a nommé son acteur (`actorSource`) quand la jointure y est
   * parvenue — `carry_drop` (un geste observé) ou `carry_active` (une présence constatée), deux
   * forces de preuve qu'un lecteur ne doit pas confondre. Un fait SANS acteur est publié quand
   * même. Vide = artefact antérieur au schéma 39, ou film hors de la famille bomb.
   */
  bombEvents: NonNullable<ReplayDocument['bombEvents']>
  /**
   * LES CINQ STATISTIQUES D'OBJECTIF DE L'ASSAUT (schéma 39) — le seul bloc du document qui
   * ne soit PAS un calque de rendu : la fiche de match les sert depuis la base, pas depuis
   * l'artefact. Il voyage ici parce que c'est à la CUISSON qu'elles se calculent (leurs quatre
   * sources n'y vivent en pleine fidélité qu'à cet instant), et le crochet de sync les
   * persiste depuis l'artefact rangé.
   * L'OBJET GARDE LE DROIT D'ÊTRE ABSENT (comme `scoreTimeline`) : absent = film hors de la
   * famille bomb, et un bloc vide se lirait « lu, rien trouvé » — ce n'est pas la même chose.
   * Son tableau, lui, est comblé.
   */
  bombStats?: ReplayBombStatsReady
  /**
   * LES RAMASSAGES ET LES CONSOMMATIONS D'ÉQUIPEMENT (schéma 26) : ce qui ARRIVE à un joueur,
   * là où `abilities` dit ce qu'il PORTE. Datés à la milliseconde puis projetés sur l'axe de
   * frames — c'est la source FINE qui affine `abilityAt` entre deux images-clés. Vide =
   * artefact antérieur au schéma 26, ou film qui n'en porte aucun.
   */
  equipmentChanges: NonNullable<ReplayDocument['equipmentChanges']>
  equipmentEpisodes: NonNullable<ReplayDocument['equipmentEpisodes']>
  equipmentPlacements: NonNullable<ReplayDocument['equipmentPlacements']>
  flagCarries: ReplayFlagCarryReady[]
  /**
   * LES ARMES AU SOL individuelles (schéma 27) : une entrée par objet qui a BOUGÉ, avec sa
   * position de repos et ses bornes d'affichage OBSERVÉES. Vide = artefact antérieur au schéma
   * 26, ou film dont aucune arme ne tombe — `coverage.groundWeaponItems` distingue les deux.
   * Les armes de SOCLE restent au calque `weaponPads` : les publier ici en double ferait deux
   * vérités pour un même objet.
   */
  groundWeapons: NonNullable<ReplayDocument['groundWeapons']>
  /**
   * LES PRISES ET LES LÂCHERS D'ARME (schéma 25) : qui, quand, quelle arme. Datés à la
   * milliseconde — c'est la source FINE qui affine `loadoutAt` entre deux images-clés. Vide =
   * artefact antérieur au schéma 25, ou film qui n'en porte aucun.
   */
  weaponChanges: NonNullable<ReplayDocument['weaponChanges']>
  /**
   * LES RAMASSAGES NATIFS (schéma 30) : l'événement `biped_pickup` de la bobine, daté à la
   * milliseconde, ATTRIBUÉ à son ramasseur et portant l'identifiant de catalogue de l'objet.
   * Il ne remplace pas `weaponChanges` — celui-ci qualifie (prise, lâcher, échange) et connaît
   * l'emplacement d'arme ; celui-là voit des prises que l'autre rate et nomme le ramasseur.
   * Vide = artefact antérieur au schéma 30, ou film qui n'en porte aucun
   * (`coverage.pickups` distingue les deux).
   */
  pickups: NonNullable<ReplayDocument['pickups']>
  /**
   * LES OBJETS D'OBJECTIF LIBRES (schéma 21) : où se trouve le crâne d'Oddball quand PERSONNE
   * ne le porte. Vide = artefact antérieur au schéma 21, mode sans objet porté, ou film qui n'en
   * porte pas — `coverage.objectiveObjects` distingue les trois.
   */
  objectiveObjects: ReplayObjectiveObjectReady[]
  geometry: NonNullable<ReplayDocument['geometry']>
  grappleLines: NonNullable<ReplayDocument['grappleLines']>
  grenadeLabels: NonNullable<ReplayDocument['grenadeLabels']>
  /**
   * L'AXE DES GRENADES PORTÉES (schéma 20), alimenté par DEUX canaux : les images-clés
   * (~20 s) et les paquets delta (transmis AU CHANGEMENT). Chaque lecture porte sa `src`.
   * Vide = artefact antérieur au schéma 20, ou film qui n'en transmet pas : la fiche
   * retombe alors sur `inventory`, exactement comme avant.
   */
  grenadeReads: ReplayGrenadeReadReady[]
  grenades: NonNullable<ReplayDocument['grenades']>
  inventory: ReplayInventoryReady[]
  loadouts: ReplayLoadoutReady[]
  neutralDeaths: NonNullable<ReplayDocument['neutralDeaths']>
  objectives: NonNullable<ReplayDocument['objectives']>
  padPickups: NonNullable<ReplayDocument['padPickups']>
  projectiles: ReplayProjectileReady[]
  roster: NonNullable<ReplayDocument['roster']>
  /**
   * LE CALQUE DE SCORE RESTE OPTIONNEL, et c'est la seule façon honnête de l'écrire : un
   * artefact de schéma antérieur à 12 n'en porte AUCUN, et un objet vide se lirait comme
   * « le film a été lu, il n'y avait pas de score ». Absent veut dire « personne n'a
   * regardé » ; les tableaux qu'il contient, eux, sont comblés — « aucun point » est une
   * mesure, et se lit sur la même grille que les pistes.
   */
  scoreTimeline?: ReplayScoreTimelineReady
  shots: NonNullable<ReplayDocument['shots']>
  structure: ReplaySurfaceReady[]
  tracks: ReplayTrackReady[]
  /**
   * LES TÉLÉPORTATIONS DU TRANSLOCATEUR (schéma 38) : une entrée PLATE par saut — (t, slot)
   * et le VA-ET-VIENT (`fx/fy/fz` -> `tx/ty/tz`), tous deux lus dans l'ÉVÉNEMENT type 117 du
   * film — jamais un seuil spatial, jamais le `spent` (jusqu'à 16,5 s de retard mesuré),
   * jamais une discontinuité de piste. Les six coordonnées sont SOLIDAIRES : présentes
   * ensemble, ou absentes en bloc (charge non lue) — `coverage.translocations.positioned`
   * dit combien de sauts les portent. Vide = artefact antérieur au schéma 38, ou film sans
   * translocateur — `coverage.translocations` distingue les deux.
   */
  translocations: NonNullable<ReplayDocument['translocations']>
  /**
   * LA VIE DE CHAQUE VÉHICULE du match (schéma 39) : où il naît, sa trajectoire, ses épisodes
   * d'occupation, jusqu'à quelle frame l'afficher. `end` vaut toujours `unknown` — jamais une
   * destruction (cf. `ReplayVehicleTrack`). Vide = artefact antérieur au schéma 39, ou film sans
   * véhicule — `coverage.vehicles` distingue les deux. `vehicleLabels` (non comblé : c'est une
   * table, pas un tableau) nomme les familles employées et pointe leur sprite.
   */
  vehicles: ReplayVehicleTrackReady[]
  weaponPads: ReplayWeaponPadReady[]
  zoneStates: ReplayZoneStateReady[]
  /**
   * LES PÉRIODES DE PORT DE LA COURONNE VIP (schéma 22) : une entrée par période, nommée par le
   * xuid du VIP. Vide = artefact antérieur au schéma 22, ou film que l'appelant n'a pas reconnu
   * VIP — `coverage.vipCrown` distingue les deux. Aucun tableau imbriqué : la période est plate.
   */
  vipCrown: NonNullable<ReplayDocument['vipCrown']>
  /**
   * LES PÉRIODES DE PORTAGE DU CRÂNE d'Oddball (schéma 23) : une entrée par période, nommée par le
   * xuid du porteur. Vide = artefact antérieur au schéma 23, ou film que l'appelant n'a pas reconnu
   * Oddball — `coverage.skullCarries` distingue les deux. Aucun tableau imbriqué : la période est
   * plate. Le crâne LIBRE (`objectiveObjects`) reste la couche POSITION ; celle-ci est le PORTEUR.
   */
  skullCarries: NonNullable<ReplayDocument['skullCarries']>
}

/** ReplayVipPeriod — UNE période de port de la couronne, telle que le rendu la lit (plate). */
export type ReplayVipPeriod = NonNullable<ReplayDocument['vipCrown']>[number]

/** ReplaySkullCarry — UNE période de portage du crâne, telle que le rendu la lit (plate). */
export type ReplaySkullCarry = NonNullable<ReplayDocument['skullCarries']>[number]

/** ReplayBombCarry — UNE période de portage de la bombe, telle que le rendu la lit (plate). */
export type ReplayBombCarry = NonNullable<ReplayDocument['bombCarries']>[number]
