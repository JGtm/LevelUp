/**
 * vehiclesLayer.ts — LES VÉHICULES sur la carte du rejeu (schéma 39) : la géométrie et les
 * règles, sans une ligne de canvas.
 *
 * SUR LE NUMÉRO DE SCHÉMA. Le chantier véhicules s'est écrit en annonçant « schéma 29 », puis
 * 30, puis 31, au fil de ses lots. AUCUN de ces artefacts n'a existé : le calque a été livré
 * d'un seul tenant et le document est passé de 38 à 39. Ces numéros ont été corrigés partout
 * le 2026-09-05 (revue adversariale de branche) — un lecteur qui chercherait un artefact de
 * schéma 30 pour comprendre une dégradation ne trouverait rien, et conclurait à un bug.
 *
 * CE QUE CE CALQUE AFFIRME. Une entrée de `doc.vehicles` est LA VIE D'UN VÉHICULE : où il naît,
 * sa trajectoire échantillonnée avec son cap, ses épisodes d'occupation (qui est à bord et
 * quand), et jusqu'à quelle frame l'afficher. La fin est une BORNE DE RECENSEMENT — `end` vaut
 * AUJOURD'HUI toujours `unknown` côté document, et dans ce cas le sprite s'efface NETTEMENT à
 * `t1max`, sans aucun effet de destruction.
 *
 * DEPUIS LE 2026-09-03 (schéma 39, EN AVANCE DE PHASE — cf. `ReplayVehicleTrack` dans
 * `lib/api/types.ts`), CE N'EST PLUS LE SEUL RÉGIME : quand `end` publie `"destroyed"` avec
 * `tEnd` (`vehicleDestructionFrame`), la fin devient une PREUVE au lieu d'une borne — le sprite
 * cesse à `tEnd` (qui prend alors l'autorité sur `t1max`, cf. `vehicleVisibleAt`) et
 * `vehiclesPaint.ts` y ancre l'explosion demandée par l'utilisateur (« il faut aussi la
 * destruction et un effet UI »). AUCUN artefact actuel ne porte ce régime — tant que le Go ne
 * mesure pas la destruction, ce texte ne change RIEN à ce qui s'affiche.
 *
 * SIX RESPONSABILITÉS PURES, TESTABLES SANS CANVAS : le REFUS DU DÉCOR (`vehicleIsDecor`,
 * `vehicleCanEmbark`), l'ORIENTATION (`vehicleHeadingAt`, `vehicleScreenAngle`,
 * `vehicleAimAngle`), la TAILLE (`vehicleSpriteScale`, ancrée sur le pion), l'OCCUPATION
 * (`vehicleActiveRides`, `vehicleDriverAt`, `vehicleColorAt`), le PRÉDICAT EMBARQUÉ
 * (`buildEmbarkedPredicate`, consommé par `replayMarkers.drawTracksLayer` pour supprimer le pion
 * d'un occupant) et la DESTRUCTION (`vehicleDestructionFrame`, `vehicleExplosionKindOf`). LE
 * TRACÉ CANVAS EST À CÔTÉ (`vehiclesPaint.ts`, extrait le 2026-09-02 quand le cône du conducteur
 * a fait franchir à ce fichier le seuil de taille du dépôt) : il ne fait qu'assembler ces six
 * réponses image par image.
 *
 * UNE SEPTIÈME EST À CÔTÉ AUSSI (`vehiclesAim.ts`, extrait le 2026-09-03 pour la même raison de
 * taille) : LA VISÉE MESURÉE DE CHAQUE OCCUPANT (schéma 39, `vehicleOccupantAimAt`). Elle
 * s'appuie sur `vehicleAimAngle`/`vehicleHeadingAt` de ce fichier — le cap du châssis n'est plus
 * la direction du cône, il n'en est que le REPLI.
 *
 * CE QUE CE CALQUE REFUSE DE DESSINER — LES FAMILLES NON JOUABLES (verdict utilisateur du
 * 2026-09-02, après visionnage réel). Voir `FAMILLES_NON_JOUABLES` : ce sont des entités de
 * DÉCOR, pas des véhicules de la partie, et elles ne doivent NI se dessiner, NI nommer
 * quiconque, NI faire disparaître un pion.
 *
 * ORIENTATION — LA CONSTANTE D'ÉCART D'ÉCRAN (GATE C6). Les échantillons portent un cap MONDE
 * (`VehicleSample.h`, convention `Point.h` : 0° = +X, 90° = +Y, sens `atan2(y,x)`), mais les
 * sprites sont dessinés NEZ EN HAUT — c'est-à-dire nez vers l'axe -Y LOCAL de l'image, jamais
 * vers +X comme les formes directionnelles du calque de tirs (`muzzleFlash.ts`, qui n'a besoin
 * que d'inverser le signe : `-h`). Un sprite « nez en haut » a donc besoin d'UN QUART DE TOUR DE
 * PLUS que cette inversion : la projection écran inverse l'axe Y du monde, donc le cap-écran
 * pour ORIENTER UNE IMAGE NEZ-HAUT est `(90° - h) `, converti en radians pour `ctx.rotate`.
 * Vérifié par `vehiclesLayer.test.ts` sur les quatre points cardinaux (est/nord/ouest/sud)
 * et RECOUPÉ sur données réelles (artefact `0d76e8f1` : cap publié 272,7° vs 273,2° recalculé
 * sur le déplacement, écart 0,5°). La preuve VISUELLE en jeu reste le gate D1 (utilisateur).
 *
 * TAILLE — DÉCISION DE CADRAGE DU PLAN (2026-09-02, utilisateur) : proportionnelle au monde
 * ENTRE véhicules (longueur de sprite en pixels × mm/px du manifeste), en pixels ÉCRAN FIXES
 * vis-à-vis du zoom (comme les pions), ancrée sur le Mongoose (≈ 1,5-2 pions de long). LE
 * MANIFESTE (`index.json`, servi par le lot A) NE GARANTIT AUJOURD'HUI l'échelle QUE DES
 * FAMILLES WARTHOG/MONGOOSE — les 12 autres portent la même valeur en attendant un re-rendu
 * mesuré (note explicite de la tâche de ce lot) : la règle lit le manifeste tel quel, et
 * DEVIENDRA JUSTE POUR CHAQUE FAMILLE quand SA valeur le sera, sans changement de code ici.
 */
import type { ReplayVehicleRide } from '@/lib/api/types'

import { CORE_RADIUS } from './replayMarkers'
import { lastIndexAt, positionAt, type XY } from './replayLogic'
import type { ReplayVehicleTrackReady } from './replayNormalize'
import { covers } from './replaySpans'

// --- FAMILLES REFUSÉES ------------------------------------------------------------------------

/**
 * FAMILLES_NON_JOUABLES — les familles que ce calque IGNORE ENTIÈREMENT.
 *
 * SOURCE : verdict utilisateur du 2026-09-02 après visionnage réel du rejeu (« sur Behemoth,
 * des falcon apparaissent alors que la partie n'en avait aucun »), recoupé sur l'artefact
 * `0d76e8f1` — le châssis `0x0000254b` est bien le modèle du Falcon, mais il est porté par des
 * entités de DÉCOR : vitesse moyenne 0,3-0,8 m/s sur toute leur vie (l'une n'a parcouru
 * strictement aucune distance), vivantes du début à la fin du film. Le Falcon, le Pelican, le
 * Phantom et le Skiff ne sont PAS pilotables en multijoueur Halo Infinite : une entité qui porte
 * leur modèle est un élément de mise en scène, jamais un véhicule de la partie.
 *
 * CE N'EST PAS UNE CORRECTION DU DOCUMENT. Le serveur a raison de publier ces vies : il recense
 * ce que le film contient (archétype ti=40), et le châssis EST celui d'un Falcon. C'est
 * l'AFFICHAGE qui doit se taire — d'où un ensemble tenu ici, côté calque, et pas un filtre Go.
 *
 * TROIS CONSÉQUENCES, TOUTES NÉCESSAIRES :
 *  1. AUCUN DESSIN — ni sprite, ni marqueur de repli : un décor ne se montre pas comme un
 *     véhicule (c'est la plainte initiale).
 *  2. AUCUN NOM — les noms d'occupants ne sont écrits que sur des véhicules réels.
 *  3. AUCUNE PARTICIPATION AU PRÉDICAT EMBARQUÉ — et c'était le dégât le plus grave : le liant
 *     « trou de position » a prêté à un de ces props TROIS épisodes d'occupation (slot 771 de
 *     l'artefact cité), ce qui ESCAMOTAIT le pion des joueurs passés à côté. Un faux embarquement
 *     efface un joueur bien réel de la carte ; le refus doit donc porter d'abord ici.
 */
export const FAMILLES_NON_JOUABLES: ReadonlySet<string> = new Set([
  'falcon',
  'pelican',
  'phantom',
  'skiff',
])

/**
 * vehicleIsDecor — vrai quand la famille est refusée par `FAMILLES_NON_JOUABLES`. Une famille
 * VIDE (châssis non résolu) n'est PAS du décor : on ne sait pas ce que c'est, et le losange
 * neutre continue de le dire.
 */
export function vehicleIsDecor(family: string | undefined): boolean {
  return family !== undefined && FAMILLES_NON_JOUABLES.has(family)
}

// --- DESTRUCTION (schéma 39 — EN AVANCE DE PHASE, cf. ReplayVehicleTrack dans types.ts) --------

/**
 * VEHICLE_END_DESTROYED — LA SEULE valeur de `track.end` que ce calque traite comme une preuve
 * de destruction. Nommée plutôt que semée en littéral (même raison que `FAMILLES_NON_JOUABLES`) :
 * un seul endroit à changer si le Go choisit un autre mot. Toute AUTRE valeur — `"unknown"`
 * aujourd'hui, systématiquement — est lue comme « fin non affirmée » : rien ne se dessine.
 */
export const VEHICLE_END_DESTROYED = 'destroyed'

/**
 * vehicleDestructionFrame — l'index de frame de la destruction, ou `null` quand elle n'est PAS
 * établie. DEUX CONDITIONS, LES DEUX NÉCESSAIRES (demande explicite du lot) : `end` doit valoir
 * `VEHICLE_END_DESTROYED` ET `tEnd` doit être publié. Un document qui affirmerait l'un sans
 * l'autre (mesure Go partielle) ne déclenche rien plutôt que de deviner une frame — même
 * prudence que `vehiclePositionAt` refusant d'inventer une position.
 *
 * TANT QUE LE GO NE PUBLIE QUE `end: "unknown"` (l'état actuel de CHAQUE artefact existant),
 * cette fonction rend TOUJOURS `null` : zéro changement visible, et c'est le comportement
 * attendu de ce lot — il rend le calque PRÊT, il n'invente pas la mesure.
 */
export function vehicleDestructionFrame(track: ReplayVehicleTrackReady): number | null {
  if (track.end !== VEHICLE_END_DESTROYED) return null
  return track.tEnd ?? null
}

/**
 * VehicleExplosionKind — LES DEUX MISES EN SCÈNE DEMANDÉES PAR L'UTILISATEUR, mot pour mot :
 * « explosion plasma et explosion "normale" pour les véhicules humains ».
 */
export type VehicleExplosionKind = 'plasma' | 'normal'

/**
 * VEHICLE_PLASMA_FAMILIES — TABLE FACTION -> EFFET (nommée et commentée, comme demandé) : les
 * châssis Covenant/Bannis, à énergie, reçoivent l'explosion PLASMA. La Shade (tourelle montée
 * Covenant, immobile — cf. `VEHICLE_DEFAULT_HEADING_DEG`) reste une énergie Covenant même sans
 * jamais se déplacer, elle entre donc ici et pas dans la table humaine ci-dessous.
 *
 * TOUTE AUTRE FAMILLE — y compris VIDE (châssis non résolu) ou inconnue de ce calque — REÇOIT LE
 * REPLI NORMAL (`vehicleExplosionKindOf`) : c'est un choix NEUTRE et documenté, jamais une
 * affirmation Covenant par défaut. La liste humaine (`VEHICLE_HUMAN_FAMILIES`) n'est donc pas
 * consultée par le code — elle n'existe QUE pour documenter la couverture de la demande
 * utilisateur et sert de fixture aux tests (vehiclesLayer.test.ts / vehiclesPaint.test.ts).
 */
export const VEHICLE_PLASMA_FAMILIES: ReadonlySet<string> = new Set([
  'ghost',
  'banshee',
  'wraith',
  'chopper',
  'shade',
])

/**
 * VEHICLE_HUMAN_FAMILIES — LES CHÂSSIS UNSC/HUMAINS explicitement cités par la demande
 * utilisateur : ils reçoivent l'explosion NORMALE, mais par le REPLI (« toute famille qui n'est
 * pas dans `VEHICLE_PLASMA_FAMILIES` »), pas par une seconde table consultée au rendu — deux
 * tables actives auraient pu diverger (une famille absente des deux, par exemple un futur
 * châssis) sans qu'aucun test ne le voie.
 */
export const VEHICLE_HUMAN_FAMILIES: ReadonlySet<string> = new Set([
  'warthog',
  'rockethog',
  'warthog_gauss',
  'razorback',
  'mongoose',
  'gungoose',
  'scorpion',
  'wasp',
  'tourelle_montee',
])

/**
 * vehicleExplosionKindOf — LA FACTION DE LA DÉFLAGRATION. `undefined`/`''` (châssis non résolu)
 * et toute famille absente de `VEHICLE_PLASMA_FAMILIES` (y compris une famille future, inconnue
 * de ce calque) reçoivent `'normal'` — le repli neutre documenté ci-dessus.
 */
export function vehicleExplosionKindOf(family: string | undefined): VehicleExplosionKind {
  return family !== undefined && VEHICLE_PLASMA_FAMILIES.has(family) ? 'plasma' : 'normal'
}

/**
 * vehicleCanEmbark — une vie de véhicule peut-elle faire DISPARAÎTRE le pion d'un joueur ?
 *
 * DEUX REFUS, POUR LA MÊME RAISON : un pion supprimé est un joueur qu'on n'affiche plus, et cela
 * ne se justifie que si le véhicule qui le porte est, lui, affiché et certain. Le décor est
 * refusé (cf. `FAMILLES_NON_JOUABLES`), et le CHÂSSIS NON RÉSOLU aussi : un épisode d'occupation
 * posé sur une famille inconnue peut porter sur n'importe quoi — dont un autre prop — et le prix
 * de l'erreur (un joueur effacé) est plus lourd que celui de l'abstention (un pion de trop
 * pendant qu'il conduit).
 */
export function vehicleCanEmbark(track: ReplayVehicleTrackReady): boolean {
  return track.family !== undefined && track.family !== '' && !vehicleIsDecor(track.family)
}

// --- ORIENTATION ----------------------------------------------------------------------------

/**
 * VEHICLE_DEFAULT_HEADING_DEG — le cap MONDE d'un véhicule dont AUCUN échantillon ne porte de
 * cap : ni mort (la tourelle Shade et la tourelle montée ne translatent jamais — décision de
 * cadrage « TOURELLES »), ni pas-encore-mobile. Choisi à 90° (monde +Y) précisément parce que
 * `vehicleScreenAngle(90) === 0` : AUCUNE rotation supplémentaire n'est appliquée, et le sprite
 * reste NEZ VERS LE HAUT DE L'ÉCRAN tel qu'il est dessiné — exactement la règle demandée.
 */
export const VEHICLE_DEFAULT_HEADING_DEG = 90

/**
 * vehicleHeadingAt — le cap MONDE du véhicule à `frame`, degrés, MÊME convention que `Point.h`.
 *
 * TROIS RÉGIMES, DANS L'ORDRE DE LA DÉCISION DE CADRAGE :
 *  1. EN MOUVEMENT / À L'ARRÊT — le dernier échantillon connu AU PLUS TARD à `frame` qui porte
 *     un cap (`h !== undefined`) : le serveur y écrit déjà la direction de la vélocité en
 *     mouvement, et REPORTE ce même cap sur les échantillons immobiles qui suivent (cf.
 *     `VehicleSample.H`) — chercher EN ARRIÈRE depuis `frame` couvre donc les deux à la fois,
 *     sans jamais interpoler un angle (l'interpolation d'un cap ferait tourner le véhicule par
 *     le chemin le plus court à travers 0°/360°, un artefact que le film ne montre pas).
 *  2. AVANT LE PREMIER MOUVEMENT — aucun échantillon EN ARRIÈRE ne porte de cap (le véhicule est
 *     encore à quai) : le cap du PREMIER échantillon mobile À VENIR, pour que le véhicule
 *     s'oriente déjà dans le sens qu'il va prendre plutôt que de pivoter brutalement au premier
 *     mouvement.
 *  3. JAMAIS MOBILE — aucun échantillon, nulle part, ne porte de cap (véhicule jamais conduit,
 *     ou tourelle qui ne translate pas) : `VEHICLE_DEFAULT_HEADING_DEG`, nez vers le haut.
 */
export function vehicleHeadingAt(track: ReplayVehicleTrackReady, frame: number): number {
  const samples = track.samples
  if (samples.length === 0) return VEHICLE_DEFAULT_HEADING_DEG
  const idx = lastIndexAt(samples, frame)
  for (let i = idx; i >= 0; i--) {
    const h = samples[i].h
    if (h !== undefined) return h
  }
  for (let i = Math.max(idx + 1, 0); i < samples.length; i++) {
    const h = samples[i].h
    if (h !== undefined) return h
  }
  return VEHICLE_DEFAULT_HEADING_DEG
}

/**
 * vehicleScreenAngle — LA CONSTANTE D'ÉCART D'ÉCRAN (gate C6) : convertit un cap MONDE en
 * radians CANEVAS pour un sprite dessiné NEZ EN HAUT. Voir l'en-tête du fichier pour la dérivation
 * complète ; le résultat tient en une ligne : `(90° − cap) → radians`.
 */
export function vehicleScreenAngle(headingDeg: number): number {
  return ((90 - headingDeg) * Math.PI) / 180
}

/**
 * vehicleAimAngle — LE MÊME CAP, mais pour le CÔNE DE VISÉE, et ce n'est PAS le même angle.
 *
 * `vehicleScreenAngle` oriente une IMAGE dessinée nez-en-haut : elle porte le quart de tour qui
 * rattrape la convention du sprite. Le cône, lui, est une forme tracée à l'angle nu — comme le
 * cône des pions (`replayAimCone` : « monde -> canevas, l'axe Y est inversé, donc l'angle l'est
 * aussi »), il n'a besoin QUE de l'inversion de signe. Confondre les deux ferait pointer le cône
 * à 90° du nez du véhicule : d'où deux fonctions nommées, et pas une constante partagée.
 */
export function vehicleAimAngle(headingDeg: number): number {
  return (-headingDeg * Math.PI) / 180
}

// --- POSITION ET FENÊTRE D'AFFICHAGE ---------------------------------------------------------

/**
 * vehicleVisibleAt — `t1max` est une PREUVE D'ABSENCE (cf. `document_vehicles.go`) : la franchir
 * affirmerait une présence que le film réfute. Décision de cadrage : « disparition NETTE » —
 * aucun estompage entre `t1`/`t1max` (à la différence des armes au sol) : le véhicule est plein
 * jusqu'à `t1max` inclus, rien après.
 *
 * `tEnd` FAIT AUTORITÉ SUR `t1max` QUAND LA DESTRUCTION EST ÉTABLIE (schéma 39, cf.
 * `vehicleDestructionFrame`) : la demande utilisateur du lot (« il faut aussi la destruction et
 * un effet UI ») veut que le SPRITE cesse au moment exact de la destruction, pas à la dernière
 * preuve de présence du recensement — l'explosion (`vehiclesPaint.ts`) prend ensuite le relais à
 * la MÊME frame, sur SA propre fenêtre, indépendamment de cette visibilité. Tant que le document
 * ne publie que `end: "unknown"` (aujourd'hui, toujours), `vehicleDestructionFrame` rend `null`
 * et ce repli sur `t1max` est donc EXACTEMENT le comportement actuel — zéro changement visible.
 */
export function vehicleVisibleAt(track: ReplayVehicleTrackReady, frame: number): boolean {
  const lastFrame = vehicleDestructionFrame(track) ?? track.t1max
  return frame >= track.t0 && frame <= lastFrame
}

/**
 * vehiclePositionAt — la position MONDE à `frame`, ou `null` (rien à dessiner).
 *
 * `positionAt` (replayLogic.ts) tient déjà l'interpolation entre échantillons et le maintien de
 * la dernière position connue une fois le dernier échantillon dépassé — LA MÊME logique que pour
 * une trajectoire de joueur, réutilisée telle quelle (`VehicleSample` a le même sous-ensemble de
 * champs que `Point`). Elle rend `null` AVANT le premier échantillon : c'est alors le SPAWN qui
 * répond (la naissance, seule chose connue tant que personne n'a conduit le véhicule). Ni l'un
 * ni l'autre : le record de création n'a pas été lu ET aucun échantillon n'existe — rien ne se
 * dessine, plutôt que d'inventer une position.
 */
export function vehiclePositionAt(track: ReplayVehicleTrackReady, frame: number): XY | null {
  const fromSamples = track.samples.length > 0 ? positionAt(track.samples, frame) : null
  if (fromSamples) return fromSamples
  return track.spawn ? { x: track.spawn.x, y: track.spawn.y } : null
}

// --- TAILLE (décision de cadrage : ancrée sur le pion) ---------------------------------------

/**
 * PION_REFERENCE_PX — le noyau du marqueur joueur (`CORE_RADIUS`, replayMarkers.ts), SEULE
 * partie TOUJOURS visible d'un pion quel que soit son étage. C'est l'ancre de la règle de
 * taille : « pion = pixels fixes, CORE 3,4 + RING 6,5 px » (décision de cadrage).
 */
const PION_REFERENCE_PX = CORE_RADIUS * 2

/** Milieu de la fourchette demandée (1,5-2 pions de long pour le Mongoose). */
const MONGOOSE_TO_PION_RATIO = 1.75

/**
 * MONGOOSE_REFERENCE_LENGTH_MM — la longueur RÉELLE (nez-en-haut) du sprite Mongoose VALIDÉ du
 * lot A (`V4_RAPPORT_SPRITES_2026-08-31.md`, statut "valide") : 128 px de sprite × 10 mm/px.
 * C'EST, AVEC LE WARTHOG, LA SEULE FAMILLE DONT L'ÉCHELLE EST GARANTIE (note de la tâche de ce
 * lot) : l'ancre de la conversion mm → pixel-écran se calibre donc sur elle, jamais sur un
 * sprite dont l'échelle n'est pas mesurée — ce chiffre n'a PAS besoin de charger le sprite
 * Mongoose au runtime, il est dérivé une fois, ici, de sa mesure connue.
 */
const MONGOOSE_REFERENCE_LENGTH_MM = 1280

/**
 * VEHICLE_PX_PER_MM — LA CONSTANTE NOMMÉE UNIQUE de la règle de taille (décision de cadrage) :
 * un millimètre-monde vaut CE NOMBRE de pixels-écran, POUR TOUTE FAMILLE. Parce qu'elle est
 * UNIQUE et appliquée à `naturalHeightPx × mmPerPx` (une propriété du COUPLE sprite×manifeste,
 * jamais couplée à une famille précise dans la formule), les tailles RELATIVES entre véhicules
 * suivent le manifeste : le jour où les 12 familles non garanties reçoivent leur propre mm/px
 * mesuré, leur taille à l'écran devient juste SANS toucher à cette constante.
 */
const VEHICLE_PX_PER_MM = (PION_REFERENCE_PX * MONGOOSE_TO_PION_RATIO) / MONGOOSE_REFERENCE_LENGTH_MM

/** Plancher de lisibilité : aucun véhicule ne descend sous le noyau d'un pion. */
export const VEHICLE_FLOOR_PX = PION_REFERENCE_PX

/**
 * Plafond DOUX : au-delà, la croissance ralentit (racine carrée) au lieu de s'arrêter net — un
 * véhicule très long reste visiblement plus grand qu'un plus petit, mais cesse de dominer
 * l'écran. Choisi à 4× la cible du Mongoose : rien du corpus mesuré aujourd'hui (Warthog,
 * Mongoose, mm/px identique) ne l'atteint — il protège la lecture pour le jour où une famille
 * hors gabarit (le Pelican, dropship, nommé par la décision de cadrage) recevra sa propre
 * mesure et non plus la valeur provisoire des 12 familles non garanties.
 */
export const VEHICLE_SOFT_CEIL_PX = 4 * MONGOOSE_TO_PION_RATIO * PION_REFERENCE_PX

/**
 * vehicleScreenLengthPx — la longueur d'écran (pixels CSS fixes, avant densité `k`) d'un
 * véhicule, plancher et plafond doux appliqués. `naturalHeightPx` est la hauteur NATIVE du
 * sprite chargé (nez-en-haut : la hauteur EST l'axe de longueur du véhicule) ; `mmPerPx` vient
 * du manifeste (`index.json`) pour SA famille.
 */
export function vehicleScreenLengthPx(naturalHeightPx: number, mmPerPx: number): number {
  if (naturalHeightPx <= 0 || mmPerPx <= 0) return 0
  const raw = naturalHeightPx * mmPerPx * VEHICLE_PX_PER_MM
  const floored = Math.max(raw, VEHICLE_FLOOR_PX)
  return floored <= VEHICLE_SOFT_CEIL_PX
    ? floored
    : VEHICLE_SOFT_CEIL_PX + Math.sqrt(floored - VEHICLE_SOFT_CEIL_PX)
}

/**
 * vehicleSpriteScale — le facteur multiplicatif à donner à `drawRotatedSprite` (avant densité
 * `k`, que l'appelant applique en plus) pour que le sprite atteigne `vehicleScreenLengthPx` sur
 * son axe de hauteur, aspect ratio préservé.
 */
export function vehicleSpriteScale(naturalHeightPx: number, mmPerPx: number): number {
  if (naturalHeightPx <= 0) return 0
  return vehicleScreenLengthPx(naturalHeightPx, mmPerPx) / naturalHeightPx
}

/** Demi-diagonale du petit losange neutre d'un châssis non résolu — le noyau d'un pion. */
export const VEHICLE_UNKNOWN_HALF_PX = CORE_RADIUS

// --- OCCUPATION -------------------------------------------------------------------------------

/**
 * vehicleActiveRides — les épisodes d'occupation qui couvrent `frame`, TRIÉS conducteur
 * d'abord (siège 0) puis sièges croissants (décision de cadrage C7). Un siège NON LU (`seat`
 * absent) passe en dernier, à égalité entre eux dans l'ordre du document — jamais avant un
 * siège connu, dont la place est, elle, affirmée.
 */
export function vehicleActiveRides(
  track: ReplayVehicleTrackReady,
  frame: number,
): ReplayVehicleRide[] {
  return track.rides
    .filter((r) => covers(r, frame))
    .sort((a, b) => (a.seat ?? Number.POSITIVE_INFINITY) - (b.seat ?? Number.POSITIVE_INFINITY))
}

/**
 * vehicleDriverAt — L'ÉPISODE DU CONDUCTEUR à `frame` (siège 0), ou `null`.
 *
 * SEUL LE SIÈGE 0 COMPTE, et un siège NON LU ne fait pas l'affaire : c'est le conducteur, et lui
 * seul, dont le CAP DU CHÂSSIS approche la direction du regard (décision utilisateur du
 * chantier : « à l'arrêt on assume qu'il regarde devant lui ; en mouvement, la direction du
 * déplacement »).
 *
 * CE N'EST PLUS LA PORTE DU CÔNE DEPUIS LE SCHÉMA 31 (`vehicleOccupantAimAt`) : chaque occupant
 * porte désormais SA visée mesurée, y compris l'artilleur et le passager. Cette fonction reste
 * l'accès au conducteur pour tout ce qui le concerne LUI (teinte du véhicule, ordre des noms).
 */
export function vehicleDriverAt(
  track: ReplayVehicleTrackReady,
  frame: number,
): ReplayVehicleRide | null {
  return vehicleActiveRides(track, frame).find((r) => r.seat === 0) ?? null
}

/**
 * VehicleInk — LES DEUX SOURCES DE LA COULEUR D'UN OCCUPANT, dans leur ordre de priorité.
 *
 * `colorOfXuid` LIT CE QUE LE DOCUMENT AFFIRME : l'épisode d'occupation porte le xuid de son
 * occupant, et c'est ce que son contrat serveur promet (« c'est lui qui donne sa couleur au
 * véhicule », `document_vehicles.go`). `colorOfSlot` est le PONT slot->joueur, qui répond « qui
 * occupe ce slot à cette image » — muet pendant l'épisode quand la trace du bipède n'a pas été
 * jointe à un joueur. Le pont reste donc le REPLI, exactement comme pour le nom.
 */
export interface VehicleInk {
  colorOfXuid: (xuid: string) => string | null
  colorOfSlot: (slot: number, frame: number) => string | null
}

/**
 * vehicleRideColor — la couleur d'UN occupant : le xuid du document d'abord, le pont ensuite.
 */
export function vehicleRideColor(
  ride: ReplayVehicleRide,
  frame: number,
  ink: VehicleInk,
): string | null {
  const byXuid = ride.xuid ? ink.colorOfXuid(ride.xuid) : null
  return byXuid ?? ink.colorOfSlot(ride.slot, frame)
}

/**
 * vehicleColorAt — la teinte du véhicule à `frame` (décision de cadrage C7) : la couleur du
 * CONDUCTEUR (siège 0) quand il est actif et son identité résolue ; À DÉFAUT, celle de
 * N'IMPORTE QUEL occupant actif dont l'identité est connue ; sinon `null` (neutre — l'appelant
 * y pose son encre de thème, ce fichier ne connaît aucun token, cf. règle color-tokens).
 */
export function vehicleColorAt(
  track: ReplayVehicleTrackReady,
  frame: number,
  ink: VehicleInk,
): string | null {
  const active = vehicleActiveRides(track, frame)
  const driver = active.find((r) => r.seat === 0)
  if (driver) {
    const c = vehicleRideColor(driver, frame, ink)
    if (c) return c
  }
  for (const r of active) {
    const c = vehicleRideColor(r, frame, ink)
    if (c) return c
  }
  return null
}

/**
 * buildEmbarkedPredicate — LE PRÉDICAT « EMBARQUÉ À T » (C7, rappel utilisateur explicite :
 * MULTI-PASSAGERS). Regroupe TOUS les épisodes d'occupation de TOUS les véhicules du document
 * par slot de bipède UNE SEULE FOIS, puis répond en O(occupations de ce slot) — quelques
 * unités par joueur sur un match entier, jamais un balayage de tous les véhicules par image.
 *
 * SORTIES INDÉPENDANTES PAR CONSTRUCTION : chaque `VehicleRide` porte SON PROPRE `[t0,t1]`, donc
 * deux occupants du MÊME véhicule (conducteur + passagers, Warthog 3 places, Razorback 4...)
 * reprennent leur pion chacun à SA frame de sortie, sans dépendre l'un de l'autre — c'est le sens
 * même de « plusieurs épisodes SIMULTANÉS », pas une propriété qu'il faut recoder ici.
 *
 * SEULES LES VIES QUI PASSENT `vehicleCanEmbark` Y ENTRENT (2026-09-02) : ni décor, ni châssis
 * non résolu. Un épisode posé sur l'un ou l'autre effacerait un joueur bien réel de la carte,
 * sans rien montrer à sa place — c'est exactement ce qu'ont produit les trois faux épisodes du
 * prop Falcon de l'artefact `0d76e8f1`.
 */
export function buildEmbarkedPredicate(
  tracks: readonly ReplayVehicleTrackReady[],
): (slot: number, frame: number) => boolean {
  const bySlot = new Map<number, ReplayVehicleRide[]>()
  for (const track of tracks) {
    if (!vehicleCanEmbark(track)) continue
    for (const ride of track.rides) {
      const list = bySlot.get(ride.slot)
      if (list) list.push(ride)
      else bySlot.set(ride.slot, [ride])
    }
  }
  return (slot, frame) => {
    const list = bySlot.get(slot)
    if (!list) return false
    return list.some((r) => covers(r, frame))
  }
}

