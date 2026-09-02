/**
 * replayContract.test.ts — LA FRONTIÈRE DE NULLABILITÉ, PROUVÉE COMPLÈTE PAR LE TYPAGE.
 *
 * CE QUE CE FICHIER FERME, ET IL A DÉJÀ COÛTÉ UNE FOIS. Depuis que le document de rejeu vient
 * du contrat généré, ses tableaux sont nullables : un slice Go nil se sérialise en `null`, et le
 * schéma le dit. Le jour où c'est arrivé, 31 erreurs de type sont tombées d'un coup, et la
 * réparation a consisté à poser UN point de passage — `normalizeReplayDocument` — plutôt que de
 * semer des `?? []` dans chaque appelant.
 *
 * MAIS CE POINT DE PASSAGE ÉNUMÈRE SES CHAMPS À LA MAIN. Le jour où le Go publiera un tableau
 * de plus, rien ne le rappellerait : le champ traverserait la frontière tel quel, `null`, et le
 * rendu tomberait à l'exécution — pas à la compilation, puisque le type `*Ready` est construit
 * depuis la même liste manuelle.
 *
 * D'OÙ LA FORME DE CE TEST : la complétude est vérifiée PAR LE COMPILATEUR, contre les types
 * GÉNÉRÉS depuis `api/openapi.yaml`. Les assertions de type font tomber `tsc -b` (donc la CI)
 * avant même qu'un test ne s'exécute :
 *
 *   1. la liste des tableaux nullables de la RACINE est exactement celle qu'on énumère ici ;
 *   2. le document normalisé n'en porte plus aucun à la racine ;
 *   3. la CARTE COMPLÈTE des tableaux nullables — à toute profondeur — est celle qu'on écrit ;
 *   4. de cette carte, la frontière ne laisse passer QUE les chemins de l'allowlist.
 *
 * POURQUOI (3) ET (4) ONT ÉTÉ AJOUTÉES (lot A phase 2, 2026-08-18). La garde de racine était
 * aveugle à ce qui vit DANS les éléments : `weaponPads[].spawns` était déjà passé au travers
 * une fois, et le calque de score du schéma 12 empile QUATRE étages de tableaux nullables
 * (`scoreTimeline.teams[].rounds[].points`). Un quatrième oubli du même genre serait tombé à
 * l'exécution. Le walker `NullableArrayPaths` descend désormais dans les objets ET dans les
 * éléments de tableaux, et l'assertion (4) est celle qui compte vraiment : elle compare la
 * carte du document NORMALISÉ à une allowlist de deux chemins justifiés — tout tableau
 * nullable que la frontière oublierait, à quelque profondeur qu'il vive, y apparaîtrait.
 *
 * Un champ ajouté côté Go arrive donc ici sans que personne n'ait à y penser. Le reste du
 * fichier vérifie le COMPORTEMENT de la frontière — ce qu'un type ne peut pas dire.
 *
 * POURQUOI PAS LIRE `openapi.yaml` DIRECTEMENT. C'était la première forme écrite, et elle
 * marchait ; elle exigeait `js-yaml`, qui n'est présent qu'en `overrides` (dépendance
 * transitive de `openapi-typescript`). S'appuyer dessus, c'est dépendre de ce qu'un autre
 * paquet installe — knip le signale à juste titre. Les types générés viennent du MÊME contrat,
 * et ils sont, eux, une dépendance déclarée du dossier.
 */
import { describe, expect, it } from 'vitest'

import { normalizeReplayDocument } from './replayNormalize'

import type { ReplayDocument } from '@/lib/api/types'
import type { ReplayDocumentReady } from './replayNormalize'

/** Égalité STRICTE de deux types (le double conditionnel différé est ce qui la rend stricte). */
type Equals<A, B> = (<T>() => T extends A ? 1 : 2) extends <T>() => T extends B ? 1 : 2
  ? true
  : false

type Expect<T extends true> = T

/** Les clés de `T` dont la valeur est un TABLEAU que le transport a le droit de laisser `null`. */
type NullableArrayKeys<T> = {
  [K in keyof T]-?: null extends T[K]
    ? NonNullable<T[K]> extends readonly unknown[]
      ? K
      : never
    : never
}[keyof T]

/**
 * NULLABLE_ARRAYS — les tableaux nullables du contrat, énumérés pour les tests d'exécution.
 *
 * CETTE LISTE N'EST PAS UNE DÉCLARATION DE FOI : l'assertion `_ListeExhaustive` ci-dessous la
 * confronte au contrat généré. Si le Go publie un tableau de plus, l'égalité de types échoue et
 * `tsc -b` refuse de compiler — avec, en clair, le nom du champ manquant.
 */
const NULLABLE_ARRAYS = [
  // `abilities` : le calque des lectures de capacité (schéma 6, 2026-08-14). Il remplace le
  // champ `Inventory.a` RETIRÉ le même jour — celui-ci portait `rang − 16` (canal d'image-clé
  // borgne), le calque porte le RANG. Deux grandeurs différentes : le champ a été retiré
  // plutôt que réinterprété, et ce garde-fou a fait son travail en refusant le nouveau
  // tableau tant qu'il n'était pas déclaré ici.
  'abilities',
  // `equipmentEpisodes` : l'état ACTIF du camouflage et du surbouclier (schéma 7,
  // 2026-08-16), en épisodes datés par vie. Deux familles seulement — les deux mesurées.
  'equipmentEpisodes',
  // `equipmentPlacements` : les POSES d'équipement sur la carte (schéma 9, 2026-08-18) —
  // mur et capteur nommés par le manifeste du titre, tout le reste publié en famille
  // `other` avec son identifiant de tag. Chaque pose porte son poseur MESURÉ (-1 quand
  // aucun bipède contemporain n'est assez proche) et le cap de visée de ce poseur.
  'equipmentPlacements',
  // `flagCarries` : LA VIE DES DRAPEAUX de CTF (schéma 14, 2026-08-18) — une entrée par objet,
  // une suite d'intervalles d'état (`carried`, `carried_open`, `dropped`, `home`). Le quatrième
  // état n'en est pas un de plus par confort : un portage que RIEN ne ferme court jusqu'à la fin
  // de l'axe, et le publier sous le nom d'un portage établi affirmerait une fin que le film ne
  // date pas.
  'flagCarries',
  'geometry',
  // `grappleLines` : les tractions de grappin (schéma 8, 2026-08-16) — fenêtre mesurée
  // [t0, t1] par vie + point d'accroche en coordonnées monde.
  'grappleLines',
  'grenadeLabels',
  // `grenadeReads` : L'AXE DES GRENADES PORTEES (schema 20, 2026-08-25) — DEUX canaux sur une
  // seule grandeur, chaque lecture disant d'ou elle vient (`src` : 'kf' / 'delta'). Il ne
  // remplace PAS `inventory` : celui-ci reste la source des munitions, de l'emplacement degaine
  // et du marqueur de lecture vide. Les melanger ferait masquer une lecture pleine par une
  // lecture partielle, et la cellule de munitions se viderait — le defaut ferme en v19.
  'grenadeReads',
  'grenades',
  'inventory',
  'loadouts',
  'neutralDeaths',
  // `objectiveObjects` : les vies LIBRES du crane d'Oddball (schema 21) — ou l'objet se trouve
  // quand PERSONNE ne le porte. Nullable au contrat comme les autres tableaux de tete ; son
  // tableau IMBRIQUE (`pts`) l'est aussi, d'ou la seconde entree dans NULLABLE_ARRAY_PATHS.
  'objectiveObjects',
  'objectives',
  // `padPickups` : les occupations de socle ACHEVÉES (schéma 11, 2026-08-17) — le socle s'est
  // vidé quelque part dans [tLow, tHigh]. Un INTERVALLE, pas un instant : le film ne porte aucun
  // événement de ramassage, et la seule preuve de disparition est le recensement des
  // images-clés, espacé de ~20 s. `xuid` vaut TOUJOURS `null` (oracle à 79,7 %, seuil 90 %).
  'padPickups',
  'projectiles',
  'roster',
  'shots',
  'structure',
  'tracks',
  // `weaponPads` : les SOCLES D'ARME du match (schéma 11, 2026-08-17) — position, famille,
  // apparitions, intervalles de présence bornés par les images-clés, et cycle de réapparition
  // SEULEMENT s'il est établi. Par MATCH et non par carte : sur deux films de la même carte les
  // socles tombent aux mêmes coordonnées au centimètre, mais l'arme qui y apparaît change.
  'weaponPads',
  // `zoneStates` : L'ÉTAT DE CHAQUE ZONE du mode (schéma 16, 2026-08-18) — une entrée par zone
  // appariée aux captures du match, une suite d'intervalles de propriété. `owner` vaut `null`
  // quand PERSONNE ne tient la zone, et c'est une MESURE (la valeur neutre du canal), pas une
  // absence de donnée. `zoneRef` indexe `mapObjectives.zones`, le calque servi à la requête.
  'zoneStates',
  // `vipCrown` : LES PÉRIODES DE PORT DE LA COURONNE VIP (schéma 22, 2026-08-27) — une entrée
  // PLATE par période (xuid, t0, t1, closed), sans tableau imbriqué. `closed` faux = rien ne
  // ferme le port (borne haute à la fin de l'axe). La garde de mode est côté serveur : `comp
  // 22 A` vaut `flag_grabs` en CTF, donc le calque n'est rempli que sur un film reconnu VIP.
  'vipCrown',
  // `skullCarries` : LES PÉRIODES DE PORTAGE DU CRÂNE d'Oddball (schéma 23, 2026-08-28) — une
  // entrée PLATE par période (xuid, t0, t1, closed), sans tableau imbriqué. Le porteur est le
  // joueur dont les tics de score de mode montent (`comp 0 A`), nommé par le pont d'instants de
  // mort PAR MANCHE. La garde de mode est côté serveur : `comp 0 A` est le score de mode de tout
  // mode, donc le calque n'est rempli que sur un film reconnu Oddball.
  'skullCarries',
  // `weaponChanges` : LES PRISES ET LES LÂCHERS D'ARME (schéma 25, 2026-08-30) — le composant
  // d'état d'arme n'entre au masque du flux delta qu'au CHANGEMENT : chaque émission est donc
  // une prise, un lâcher ou un échange, daté à la milliseconde et non dans un intervalle de
  // vingt secondes. Il ne remplace pas `loadouts` : celui-ci reste l'ÉTAT lu à l'image-clé,
  // ceci en date les transitions (cf. changeRefine.ts).
  'weaponChanges',
  // `equipmentChanges` : LES RAMASSAGES ET LES CONSOMMATIONS d'équipement (schéma 26,
  // 2026-08-30) — même matière qu'`abilities` (i48), autre question : ce qui ARRIVE au joueur,
  // et non ce qu'il PORTE. Les annonces de réapparition en sont écartées côté serveur.
  'equipmentChanges',
  // `groundWeapons` : LES ARMES AU SOL individuelles (schéma 27, 2026-08-30) — un objet par
  // arme qui a BOUGÉ, sa position de repos, et des bornes d'affichage OBSERVÉES : ramassage
  // daté (`pickup`), dernière preuve de présence (`seen`, avec `t1max` pour première preuve
  // d'absence), ou aucune preuve de disparition (`open`). Les armes de SOCLE restent à
  // `weaponPads` : deux vérités pour un même objet seraient une de trop.
  'groundWeapons',
  // `vehicles` : LA VIE DE CHAQUE VÉHICULE du match (schéma 29, 2026-09-02) — naissance,
  // trajectoire échantillonnée (cap compris), épisodes d'occupation, borne d'affichage. `end`
  // vaut TOUJOURS `inconnue` : la datation de la destruction a été mesurée et RÉFUTÉE (le
  // conducteur sort vivant, le véhicule réplique encore 13-36 s après avoir été quitté).
  'vehicles',
] as const

/** (1) La liste couvre EXACTEMENT les tableaux nullables du contrat — ni plus, ni moins. */
type _ListeExhaustive = Expect<
  Equals<(typeof NULLABLE_ARRAYS)[number], NullableArrayKeys<ReplayDocument>>
>

/** (2) Le document normalisé n'en porte plus aucun : la frontière les a TOUS comblés. */
type _FrontiereComplete = Expect<Equals<NullableArrayKeys<ReplayDocumentReady>, never>>

/**
 * Profondeur restante du walker. Le contrat n'a pas de type récursif — c'est au compilateur
 * qu'il faut le prouver, et une borne explicite le fait sans rien coûter. 6 laisse deux étages
 * de marge au plus profond des chemins connus (score.rounds[].points d'un joueur, à 4).
 */
type Prev = [never, 0, 1, 2, 3, 4, 5, 6]

/**
 * NullableArrayPaths — TOUS les chemins de `T` qui mènent à un tableau que le transport a le
 * droit de laisser `null`, à quelque profondeur qu'il vive.
 *
 * `a.b` se lit « le champ b de l'objet a » ; `a[].b` « le champ b des ÉLÉMENTS du tableau a ».
 * Les objets à signature d'index (`weaponLabels`, `killEffects`, `coverage.verdict` : des
 * dictionnaires, pas des structures) sont exclus — ils n'ont pas de champ à énumérer.
 */
type NullableArrayPaths<T, D extends number = 6> = [D] extends [never]
  ? never
  : string extends keyof T
    ? never
    : {
        [K in keyof T & string]-?:
          | (null extends T[K] ? (NonNullable<T[K]> extends readonly unknown[] ? K : never) : never)
          | (NonNullable<T[K]> extends readonly (infer E)[]
              ? `${K}[].${NullableArrayPaths<NonNullable<E>, Prev[D]>}`
              : NonNullable<T[K]> extends object
                ? `${K}.${NullableArrayPaths<NonNullable<T[K]>, Prev[D]>}`
                : never)
      }[keyof T & string]

/**
 * NULLABLE_ARRAY_PATHS — la CARTE du contrat : 54 chemins, racine et profondeurs confondues.
 *
 * Elle n'est pas décorative : l'assertion (3) la confronte au contrat généré. Le Go publie un
 * tableau de plus, où que ce soit, et `tsc -b` refuse de compiler en nommant le chemin.
 */
const NULLABLE_ARRAY_PATHS = [
  // Racine — les mêmes que NULLABLE_ARRAYS ci-dessus, retrouvées par le walker.
  'abilities',
  'equipmentEpisodes',
  'equipmentPlacements',
  'flagCarries',
  'geometry',
  'grappleLines',
  'grenadeLabels',
  'grenadeReads',
  'grenades',
  'inventory',
  'loadouts',
  'neutralDeaths',
  'objectiveObjects',
  'objectives',
  'padPickups',
  'projectiles',
  'roster',
  'shots',
  'structure',
  'tracks',
  'weaponPads',
  'zoneStates',
  // `vipCrown` (schéma 22) : période PLATE, aucun tableau imbriqué — un seul chemin, la racine.
  'vipCrown',
  // `skullCarries` (schéma 23) : période PLATE, aucun tableau imbriqué — un seul chemin, la racine.
  'skullCarries',
  // Schémas 25-27 : trois calques PLATS, aucun tableau imbriqué — un seul chemin chacun. Les
  // objets qu'ils portent ne contiennent que des nombres et des chaînes.
  'weaponChanges',
  'equipmentChanges',
  'groundWeapons',
  'vehicles',
  // Dans les ÉLÉMENTS d'un tableau de tête — ce que la garde de racine ne voyait pas.
  'flagCarries[].spans',
  // La vie d'un véhicule (schéma 29) porte DEUX tableaux imbriqués nullables : sa trajectoire
  // (`samples`) et ses épisodes d'occupation (`rides`). `spawn`, lui, n'est PAS un tableau — un
  // objet optionnel absent quand le record de création n'a pas été lu — et ne figure donc pas ici.
  'vehicles[].samples',
  'vehicles[].rides',
  // La trajectoire d'une vie libre d'objet d'objectif (schema 21) : comblee par la
  // frontiere, comme `flagCarries[].spans` — une vie qui arriverait avec `pts: null` ferait
  // tomber le calque a l'execution, pas a la compilation.
  'objectiveObjects[].pts',
  'zoneStates[].spans',
  // `zoneStates[].gauge` : LA JAUGE DE CAPTURE EN DIRECT (schéma 18, 2026-08-18 — le 17 est
  // parti aux socles de power-up, fusionnés avant nous) — la série datée `[{t, v}]` de la
  // jauge pendant ses rampes. Absente sur un artefact de schéma <= 17 : la frontière la comble
  // à VIDE, et le rendu ne dessine alors aucun arc.
  'zoneStates[].gauge',
  // `grenadeReads[].g` : le quadruplet de compteurs de l'axe des grenades (schema 20). Comble
  // a VIDE par la frontiere, comme `inventory[].g` : une lecture qui arriverait avec `g: null`
  // ferait tomber la boite de grenades a l'execution — pas a la compilation.
  'grenadeReads[].g',
  'inventory[].am',
  'inventory[].g',
  'loadouts[].w',
  'projectiles[].p',
  'structure[].poly',
  'tracks[].points',
  'weaponPads[].presence',
  'weaponPads[].spawns',
  // Le CALQUE DE SCORE (schéma 12) : quatre étages, dix-sept chemins. Les paliers d'une
  // manche (`rounds[].points`) et le cumul du match (`total`) sont deux tableaux distincts,
  // pour les équipes comme pour chacune des quatre séries d'un joueur.
  'scoreTimeline.players',
  'scoreTimeline.teams',
  'scoreTimeline.players[].assists.rounds',
  'scoreTimeline.players[].assists.rounds[].points',
  'scoreTimeline.players[].assists.total',
  'scoreTimeline.players[].deaths.rounds',
  'scoreTimeline.players[].deaths.rounds[].points',
  'scoreTimeline.players[].deaths.total',
  'scoreTimeline.players[].kills.rounds',
  'scoreTimeline.players[].kills.rounds[].points',
  'scoreTimeline.players[].kills.total',
  'scoreTimeline.players[].score.rounds',
  'scoreTimeline.players[].score.rounds[].points',
  'scoreTimeline.players[].score.total',
  'scoreTimeline.teams[].rounds',
  'scoreTimeline.teams[].rounds[].points',
  'scoreTimeline.teams[].total',
  // La GARDE de la colline (KOTH) : la serie de tics par camp, lue et non reconstruite.
  'scoreTimeline.holdTicks',
  'scoreTimeline.holdTicks[].ticks',
  // Les objectifs STATIQUES du mode : servis à la requête, normalisés à l'entrée de leur
  // calque (`normalizeMapObjectives`) et non par la frontière du document — d'où leur
  // présence dans l'allowlist ci-dessous.
  'mapObjectives.markers',
  'mapObjectives.zones',
  // Les socles de carte croises (allumes seulement) : meme regime que mapObjectives —
  // servis a la requete, consommes par le calque des socles avec son propre repli.
  'mapWeaponPads.pads',
] as const

/**
 * PATHS_HORS_FRONTIERE — les seuls chemins que `normalizeReplayDocument` laisse passer, et la
 * raison tient en une ligne : `mapObjectives` n'appartient pas à l'artefact. Il est REMPLI À LA
 * REQUÊTE depuis le catalogue de cartes, et son calque a sa propre entrée
 * (`normalizeMapObjectives`) — le combler ici en ferait une seconde vérité.
 */
const PATHS_HORS_FRONTIERE = [
  'mapObjectives.markers',
  'mapObjectives.zones',
  'mapWeaponPads.pads',
] as const

/** (3) La carte couvre EXACTEMENT les tableaux nullables du contrat, à toute profondeur. */
type _CarteExhaustive = Expect<
  Equals<(typeof NULLABLE_ARRAY_PATHS)[number], NullableArrayPaths<ReplayDocument>>
>

/** (4) De cette carte, le document normalisé ne laisse passer QUE l'allowlist justifiée. */
type _FrontiereProfonde = Expect<
  Equals<(typeof PATHS_HORS_FRONTIERE)[number], NullableArrayPaths<ReplayDocumentReady>>
>

describe('la frontière du document de rejeu', () => {
  it('énumère EXACTEMENT les tableaux nullables du contrat — vérifié à la compilation', () => {
    const exhaustive: _ListeExhaustive = true
    expect(exhaustive).toBe(true)
    expect(NULLABLE_ARRAYS.length).toBeGreaterThan(0)
  })

  it('ne laisse AUCUN tableau nullable au rendu — vérifié à la compilation', () => {
    const complete: _FrontiereComplete = true
    expect(complete).toBe(true)
  })

  it('comble un document HOSTILE, où tout ce qui peut être null l’est', () => {
    const hostile = Object.fromEntries(
      NULLABLE_ARRAYS.map((k) => [k, null]),
    ) as unknown as ReplayDocument
    const ready = normalizeReplayDocument(hostile) as unknown as Record<string, unknown>
    const oublis = NULLABLE_ARRAYS.filter((k) => !Array.isArray(ready[k]))
    expect(
      oublis,
      `champ(s) que la frontière ne comble pas : ${oublis.join(', ')}. ` +
        `À ajouter dans normalizeReplayDocument, jamais par un « ?? [] » chez l’appelant.`,
    ).toEqual([])
  })

  it('traite ABSENT comme NULL : pour le rendu, les deux disent « aucune donnée »', () => {
    const ready = normalizeReplayDocument({} as ReplayDocument) as unknown as Record<string, unknown>
    for (const k of NULLABLE_ARRAYS) {
      expect(Array.isArray(ready[k]), `champ ${k}`).toBe(true)
    }
  })

  it('n’invente aucune valeur : un tableau absent devient VIDE, jamais peuplé', () => {
    const ready = normalizeReplayDocument({} as ReplayDocument) as unknown as Record<
      string,
      unknown[]
    >
    for (const k of NULLABLE_ARRAYS) {
      expect(ready[k], `champ ${k}`).toHaveLength(0)
    }
  })

  it('comble aussi les tableaux IMBRIQUÉS, que la liste de tête ne voit pas', () => {
    // CE QUE `_ListeExhaustive` NE PEUT PAS DIRE. L'assertion de type n'énumère que les tableaux
    // nullables de la RACINE : elle exige bien `weaponPads`, `tracks`, `inventory`… mais elle est
    // aveugle à ceux qui vivent DANS leurs éléments. Un socle sans apparition répliquée arrivait
    // ainsi au rendu avec `spawns: null`, et `pad.spawns.map` tombait à l'exécution — c'est le
    // correctif de revue du 2026-08-17, et ce test est ce qui le retient.
    const raw = {
      weaponPads: [{ x: 1, y: 2, weapon: '0x0000ffff' }],
      tracks: [{ slot: 1, team: -1 }],
      inventory: [{ t: 0, slot: 1 }],
      loadouts: [{ t: 0, slot: 1 }],
      flagCarries: [{ team: 0 }],
      zoneStates: [{ zoneRef: 0 }],
      vehicles: [{ slot: 5, gen: 1, t0: 0, t1: 100, t1max: 100, end: 'inconnue' }],
    } as unknown as ReplayDocument
    const ready = normalizeReplayDocument(raw)
    expect(ready.flagCarries[0].spans, 'flagCarries[].spans').toEqual([])
    expect(ready.vehicles[0].samples, 'vehicles[].samples').toEqual([])
    expect(ready.vehicles[0].rides, 'vehicles[].rides').toEqual([])
    // `spawn` N'EST PAS un tableau : absent reste absent, jamais un objet inventé qui se lirait
    // comme une naissance que le film ne montre pas.
    expect(ready.vehicles[0].spawn ?? null).toBeNull()
    expect(ready.zoneStates[0].spans, 'zoneStates[].spans').toEqual([])
    // La jauge en direct (schéma 18) : un artefact plus ancien ne la porte pas, et elle se
    // comble à VIDE — « aucun arc », jamais le sommet statique à sa place.
    expect(ready.zoneStates[0].gauge, 'zoneStates[].gauge').toEqual([])
    expect(ready.weaponPads[0].spawns, 'weaponPads[].spawns').toEqual([])
    expect(ready.weaponPads[0].presence, 'weaponPads[].presence').toEqual([])
    expect(ready.tracks[0].points, 'tracks[].points').toEqual([])
    expect(ready.inventory[0].am, 'inventory[].am').toEqual([])
    expect(ready.inventory[0].g, 'inventory[].g').toEqual([])
    expect(ready.loadouts[0].w, 'loadouts[].w').toEqual([])
    // `cycle` N'EST PAS un tableau : une mesure absente reste absente, jamais un objet vide qui
    // se lirait comme un cycle de zéro seconde.
    expect(ready.weaponPads[0].cycle ?? null).toBeNull()
  })

  it('conserve ce que le socle porte quand il le porte', () => {
    const raw = {
      weaponPads: [
        {
          x: 1,
          y: 2,
          weapon: '0x0000ffff',
          spawns: [10, 200],
          presence: [{ t0: 10, tLow: 200, tHigh: 400 }],
          cycle: { medianS: 30.5, p10S: 30.2, p90S: 30.8, gaps: 2, missing: 1 },
        },
      ],
    } as unknown as ReplayDocument
    const ready = normalizeReplayDocument(raw)
    expect(ready.weaponPads[0].spawns).toEqual([10, 200])
    expect(ready.weaponPads[0].presence).toHaveLength(1)
    // `missing` dit combien d'écarts le socle offrait sans qu'on ait pu les mesurer : sans lui,
    // « 2 écarts » se lit comme « 2 sur 2 » alors que la mesure en a perdu un.
    expect(ready.weaponPads[0].cycle?.gaps).toBe(2)
    expect(ready.weaponPads[0].cycle?.missing).toBe(1)
  })

  it('recopie EN SURFACE : les points de trajectoire ne sont jamais dupliqués', () => {
    // Ce sont eux qui font le poids du document (29 221 points sur le film de référence) ;
    // les recopier à chaque normalisation coûterait à chaque chargement, pour rien.
    const points = [{ t: 0, x: 1, y: 2 }]
    const raw = { tracks: [{ slot: 1, team: -1, points }] } as unknown as ReplayDocument
    expect(normalizeReplayDocument(raw).tracks[0].points).toBe(points)
  })

  it('énumère EXACTEMENT les tableaux nullables du contrat À TOUTE PROFONDEUR', () => {
    const exhaustive: _CarteExhaustive = true
    const profonde: _FrontiereProfonde = true
    expect(exhaustive).toBe(true)
    expect(profonde).toBe(true)
    // La carte couvre la racine : tout ce que la liste de tête énumère s'y retrouve tel quel.
    const manquants = NULLABLE_ARRAYS.filter((k) => !NULLABLE_ARRAY_PATHS.includes(k))
    expect(
      manquants,
      `chemin(s) de racine absent(s) de la carte : ${manquants.join(', ')}`,
    ).toEqual([])
    expect(PATHS_HORS_FRONTIERE.length).toBeLessThan(NULLABLE_ARRAY_PATHS.length)
  })

  it('comble les QUATRE étages du calque de score, que la garde de racine ne voit pas', () => {
    // Un calque de score entièrement hostile : chaque tableau des quatre niveaux vaut null.
    const raw = {
      scoreTimeline: {
        teams: [{ teamId: 0, rounds: [{ round: 0, points: null }], total: null }],
        players: [
          {
            xuid: '2533274815845110',
            score: { rounds: [{ round: 0, points: null }], total: null },
            kills: { rounds: null, total: null },
            deaths: { rounds: null, total: null },
            assists: { rounds: null, total: null },
          },
        ],
      },
    } as unknown as ReplayDocument
    const st = normalizeReplayDocument(raw).scoreTimeline
    expect(st, 'scoreTimeline').toBeDefined()
    expect(st?.teams[0].total, 'teams[].total').toEqual([])
    expect(st?.teams[0].rounds[0].points, 'teams[].rounds[].points').toEqual([])
    expect(st?.players[0].score.rounds[0].points, 'players[].score.rounds[].points').toEqual([])
    for (const k of ['score', 'kills', 'deaths', 'assists'] as const) {
      expect(st?.players[0][k].total, `players[].${k}.total`).toEqual([])
      expect(st?.players[0][k].rounds, `players[].${k}.rounds`).toBeInstanceOf(Array)
    }
  })

  it('comble teams et players quand le calque existe mais qu’ils valent null', () => {
    const raw = { scoreTimeline: { teams: null, players: null } } as unknown as ReplayDocument
    const st = normalizeReplayDocument(raw).scoreTimeline
    expect(st?.teams).toEqual([])
    expect(st?.players).toEqual([])
  })

  it('n’invente PAS un calque de score : absent reste absent', () => {
    // Un artefact de schéma antérieur à 12 n'en porte aucun. Un objet vide se lirait
    // « le film a été lu, il n'y avait pas de score » — ce n'est pas la même chose.
    expect(normalizeReplayDocument({} as ReplayDocument).scoreTimeline).toBeUndefined()
  })

  it('rétablit l’arité que JSON Schema ne sait pas rendre en TypeScript', () => {
    // `Surface.poly` est un [2]float32 côté Go, `Projectile.p` un [3]float32, et le contrat le
    // DIT (minItems = maxItems — vérifié côté Go par contracttest/replay_contract_test.go).
    // C'est le générateur de types qui retombe sur `number[]` ; la frontière rétablit l'arité
    // une seule fois, ici, plutôt qu'un cast par appelant.
    const raw = {
      structure: [{ x0: 0, y0: 0, x1: 1, y1: 1, z: 0, zb: 0, poly: [[1, 2]] }],
      projectiles: [{ t0: 0, p: [[0, 1, 2]] }],
    } as unknown as ReplayDocument
    const ready = normalizeReplayDocument(raw)
    expect(ready.structure[0].poly[0]).toHaveLength(2)
    expect(ready.projectiles[0].p[0]).toHaveLength(3)
    // Et l'absence reste une absence : jamais un tuple fabriqué.
    const vide = normalizeReplayDocument({
      structure: [{ x0: 0, y0: 0, x1: 1, y1: 1, z: 0, zb: 0 }],
    } as unknown as ReplayDocument)
    expect(vide.structure[0].poly).toEqual([])
  })
})

