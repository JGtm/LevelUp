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
import {
  normalizeScoreTimeline,
  type ReplayScoreTimelineReady,
} from '@/lib/replay/scoreTimeline'
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
  ReplayWeaponPad,
  ReplayZoneState,
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
    // L'ARMEMENT DE LA BOMBE d'Assaut (schéma 29) : hold, instant armé, mèche. Absent =
    // artefact antérieur, mode non couvert, ou calque retenu par la confrontation locale —
    // `coverage.bombArmings` distingue les silences. Aucun tableau imbriqué : l'entrée est plate.
    bombArmings: raw.bombArmings ?? [],
    // LES PÉRIODES DE PORTAGE DE LA BOMBE d'Assaut (schéma 30) : une entrée plate par période
    // (xuid, t0, t1, closed), le patron de `skullCarries` sur le canal des armes tenues.
    // Absent = artefact antérieur, ou film hors famille bomb — `coverage.bombCarries`
    // distingue les deux.
    bombCarries: raw.bombCarries ?? [],
    // LES RAMASSAGES ET LES CONSOMMATIONS d'équipement (schéma 26) : la source FINE de datation
    // de ce que porte un joueur. `abilities` reste la LECTURE (ce qu'il porte, échantillonné) ;
    // ceci est l'ÉVÉNEMENT (ce qui lui arrive, daté). Absent = artefact antérieur, ou film qui
    // n'en porte aucun — `coverage.equipmentChanges` distingue les deux.
    equipmentChanges: raw.equipmentChanges ?? [],
    // LES PRISES ET LES LÂCHERS D'ARME (schéma 25) : même rapport à `loadouts` que ci-dessus —
    // la lecture d'image-clé dit l'ÉTAT, ces événements datent le CHANGEMENT. Absent = artefact
    // antérieur, ou film qui n'en porte aucun (`coverage.weaponChanges` distingue les deux).
    weaponChanges: raw.weaponChanges ?? [],
    // LES RAMASSAGES NATIFS (schéma 30) : l'événement que la bobine écrit elle-même, là où
    // `weaponChanges` déduit d'un changement de composant. Absent = artefact antérieur, ou film
    // qui n'en porte aucun (`coverage.pickups` distingue les deux).
    pickups: raw.pickups ?? [],
    // LES ARMES AU SOL individuelles (schéma 27) : une entrée par objet qui a bougé, bornée par
    // l'OBSERVATION. Absent = artefact antérieur, ou film dont aucune arme ne tombe —
    // `coverage.groundWeaponItems` distingue les deux. Aucun tableau imbriqué : l'objet est plat.
    groundWeapons: raw.groundWeapons ?? [],
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
    // LA VIE DES DRAPEAUX de CTF (schéma 14) : une entrée par objet, une suite d'intervalles
    // d'état. Absent = le film n'est pas reconnu comme du CTF, ou personne ne l'a lu pour ce
    // calque — `coverage.flagCarries` distingue les deux, et c'est pour cela qu'il est publié.
    //
    // LE TABLEAU IMBRIQUÉ SE COMBLE AUSSI (`spans`), comme pour `weaponPads` et `tracks` : le
    // contrat le déclare nullable, et un drapeau qui arriverait avec `spans: null` ferait
    // tomber le calque à l'exécution — pas à la compilation.
    flagCarries: (raw.flagCarries ?? []).map((f) => ({ ...f, spans: f.spans ?? [] })),
    // LES PÉRIODES DE PORT DE LA COURONNE VIP (schéma 22) : une entrée plate par période
    // (xuid, t0, t1, closed), aucun tableau imbriqué. Absent = artefact antérieur, ou film
    // non reconnu VIP — `coverage.vipCrown` distingue les deux, et c'est pour cela qu'il existe.
    vipCrown: raw.vipCrown ?? [],
    // LES PÉRIODES DE PORTAGE DU CRÂNE d'Oddball (schéma 23) : une entrée plate par période
    // (xuid, t0, t1, closed), aucun tableau imbriqué. Absent = artefact antérieur, ou film non
    // reconnu Oddball — `coverage.skullCarries` distingue les deux.
    skullCarries: raw.skullCarries ?? [],
    // LES TÉLÉPORTATIONS DU TRANSLOCATEUR (schéma 38) : une entrée plate par saut — (t, slot)
    // et le va-et-vient, datés et situés par l'ÉVÉNEMENT du film. Absent = artefact antérieur
    // au schéma 38, ou film sans translocateur — `coverage.translocations` distingue les deux.
    translocations: raw.translocations ?? [],
    // LES IMPULSIONS DE CAPACITÉ (schéma 38) : une entrée plate par geste (t, slot, family),
    // l'usage MESURÉ du propulseur. Absent = artefact antérieur au schéma 38, film sans
    // propulseur, ou palette non classée — `coverage.abilityImpulses` distingue les trois.
    abilityImpulses: raw.abilityImpulses ?? [],
    // LES CHARGES D'ÉQUIPEMENT RESTANTES (schéma 38 enrichi, lot P5) : une entrée plate par
    // lecture (t, slot, family, charges) — jamais un compte d'usages dérivé. Absent =
    // artefact antérieur, film sans lecture armée, ou palette non classée —
    // `coverage.abilityCharges` distingue les trois.
    abilityCharges: raw.abilityCharges ?? [],
    // LES OBJETS D'OBJECTIF LIBRES (schéma 21) : une entrée par VIE de l'objet hors portage.
    // Absent = artefact antérieur, mode sans objet porté, ou film qui n'en porte pas —
    // `coverage.objectiveObjects` distingue les trois, et c'est pour cela qu'il est publié.
    // Le tableau IMBRIQUÉ (`pts`) se comble aussi, même raison que `spans` ci-dessus.
    objectiveObjects: (raw.objectiveObjects ?? []).map((o) => ({ ...o, pts: o.pts ?? [] })),
    geometry: raw.geometry ?? [],
    // Les TRACTIONS de grappin (schéma 8) : fenêtre mesurée [t0, t1] par vie + point
    // d'accroche en coordonnées monde. Absent = aucune traction lue sur ce film : rien
    // ne se trace, jamais une ligne devinée.
    grappleLines: raw.grappleLines ?? [],
    grenadeLabels: raw.grenadeLabels ?? [],
    // `g` est comblé comme partout ailleurs : le contrat le déclare nullable, et une lecture
    // qui arriverait avec `g: null` ferait tomber la boîte de grenades à l'exécution.
    grenadeReads: (raw.grenadeReads ?? []).map((gr) => ({ ...gr, g: gr.g ?? [] })),
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
    // Les occupations de SOCLE achevées (schéma 11) : le socle s'est vidé quelque part dans
    // [tLow, tHigh]. Absent = aucun socle ne s'est vidé sur ce film — ou le film n'en porte
    // aucun (`coverage.groundWeapons` distingue les deux, et c'est pour cela qu'il est publié).
    padPickups: raw.padPickups ?? [],
    // `mapObjectives` (objectifs STATIQUES du mode, servis à la requête) passe par
    // `...raw` : c'est un objet optionnel, pas un tableau — sa normalisation vit à
    // l'entrée du calque (normalizeMapObjectives), comme celle des callouts.
    // `as` sur l'arité seule : le contenu est celui du contrat, seule la longueur fixe du
    // tuple que JSON Schema ne sait pas dire est réaffirmée (cf. en-tête).
    projectiles: (raw.projectiles ?? []).map((pr) => ({ ...pr, p: (pr.p ?? []) as ReplayStep[] })),
    roster: raw.roster ?? [],
    // Le SCORE DANS LE TEMPS (schéma 12) : quatre étages de tableaux nullables comblés d'un
    // coup (cf. normalizeScoreTimeline). L'OBJET, lui, garde le droit d'être absent.
    scoreTimeline: normalizeScoreTimeline(raw.scoreTimeline),
    shots: raw.shots ?? [],
    structure: (raw.structure ?? []).map((s) => ({ ...s, poly: (s.poly ?? []) as ReplayXY[] })),
    tracks: (raw.tracks ?? []).map((t) => ({ ...t, points: t.points ?? [] })),
    // Les SOCLES D'ARME du match (schéma 11). Absent = le film n'en porte aucun : rien ne se
    // dessine, jamais un socle deviné. Une donnée de MATCH, pas de carte — l'arme qui apparaît
    // sur un socle change d'un match à l'autre, la position non.
    //
    // LES DEUX TABLEAUX IMBRIQUÉS SE COMBLENT AUSSI (`spawns`, `presence`), comme pour `tracks`
    // et `structure` : le contrat les déclare nullables, et un socle qui arriverait avec
    // `spawns: null` ferait tomber le calque à l'exécution — pas à la compilation.
    weaponPads: (raw.weaponPads ?? []).map((pad) => ({
      ...pad,
      spawns: pad.spawns ?? [],
      presence: pad.presence ?? [],
    })),
    // L'ÉTAT DES ZONES (schéma 16) : une entrée par zone appariée, une suite d'intervalles de
    // propriété. Absent = le mode n'a pas de zone, ou l'appariement n'a rien rattaché —
    // `coverage.zones` distingue les deux, et c'est pour cela qu'il est publié.
    //
    // LES TABLEAUX IMBRIQUÉS SE COMBLENT AUSSI (`spans`, et `gauge` — la jauge en direct du
    // schéma 18), comme pour `flagCarries` et `weaponPads`. Une jauge absente (schéma <= 17, ou
    // zone sans rampe) devient VIDE : aucun arc, jamais le sommet statique à sa place.
    zoneStates: (raw.zoneStates ?? []).map((z) => ({ ...z, spans: z.spans ?? [], gauge: z.gauge ?? [] })),
  }
}
