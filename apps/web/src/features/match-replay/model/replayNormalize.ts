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
 *
 * LES TYPES `*Ready` — la FORME du document normalisé, champ par champ — vivent dans
 * `replayReadyTypes.ts` depuis le 2026-09-05 : un DÉPLACEMENT, sans une ligne changée, pour
 * repasser sous le seuil de 500 lignes. Ce fichier-ci garde LA FONCTION, et les re-publie.
 */
import { normalizeScoreTimeline } from '@/lib/replay/scoreTimeline'
import type { ReplayDocument } from '@/lib/api/types'

import type { ReplayDocumentReady, ReplayStep, ReplayXY } from './replayReadyTypes'

/**
 * LES TYPES `*Ready` SE RE-PUBLIENT ICI, et ce n'est pas un baril de complaisance : ils vivent
 * dans `replayReadyTypes.ts` depuis le découpage du 2026-09-05 (seuil des 500 lignes), et la
 * frontière doit rester UN seul module pour ses ~140 appelants — un déplacement de fichier ne
 * se paie pas en réécriture d'imports partout.
 */
export type {
  ReplayBombCarry,
  ReplayBombStatsReady,
  ReplayDocumentReady,
  ReplayFlagCarryReady,
  ReplayGrenadeReadReady,
  ReplayInventoryReady,
  ReplayObjectiveObjectReady,
  ReplayProjectileReady,
  ReplaySkullCarry,
  ReplaySurfaceReady,
  ReplayTrackReady,
  ReplayVehicleRideReady,
  ReplayVehicleTrackReady,
  ReplayVipPeriod,
  ReplayWeaponPadReady,
  ReplayZoneStateReady,
} from './replayReadyTypes'

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
    // LES FAITS DATÉS DE LA BOMBE (schéma 39) : armements et explosions. Absent = artefact
    // antérieur, ou film hors famille bomb.
    bombEvents: raw.bombEvents ?? [],
    // LES CINQ STATISTIQUES D'ASSAUT (schéma 39). L'OBJET garde le droit d'être absent — un
    // bloc vide se lirait « lu, rien trouvé » —, mais son tableau est comblé.
    bombStats: raw.bombStats == null ? undefined : { ...raw.bombStats, players: raw.bombStats.players ?? [] },
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
    // LA VIE DE CHAQUE VÉHICULE (schéma 29). Absent = artefact antérieur, ou film sans véhicule
    // (`coverage.vehicles` distingue les deux). LES DEUX TABLEAUX IMBRIQUÉS SE COMBLENT AUSSI
    // (`samples`, `rides`), même patron que `tracks` et `weaponPads` : `spawn`, lui, N'EST PAS un
    // tableau et reste tel quel (absent = record de création non lu, jamais un objet inventé).
    vehicles: (raw.vehicles ?? []).map((v) => ({
      ...v,
      samples: v.samples ?? [],
      // LA SÉRIE DE VISÉE D'UN OCCUPANT (schéma 31) SE COMBLE AU TROISIÈME NIVEAU : c'est un
      // tableau nullable dans un tableau imbriqué, et la garde de contrat les exige tous comblés
      // ou justifiés. Vide = artefact antérieur au schéma 31, ou épisode sans lecture — le cône
      // retombe alors sur le cap du châssis (`vehiclesAim.vehicleOccupantAimAt`).
      rides: (v.rides ?? []).map((r) => ({ ...r, aim: r.aim ?? [] })),
    })),
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
