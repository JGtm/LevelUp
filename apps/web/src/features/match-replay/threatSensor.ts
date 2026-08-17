/**
 * threatSensor.ts — LE CAPTEUR DE MENACES : sa portée, son PING, et ce que le ping révèle.
 *
 * LES CHIFFRES SONT OFFICIELS, ET ILS NE SONT PAS MESURÉS ICI. Le film ne porte AUCUNE portée
 * de détection, aucune cadence de balayage, aucune durée de révélation — ni dans le record de
 * création de l'objet, ni dans sa vie. Ce fichier les tient donc de la source publiée par
 * l'éditeur, citée verbatim (ajustements de la saison 4) :
 *
 *   « Ping Frequency: 2.8 -> 1.8 secondes » · « Sensor Radius: 3.1wu -> 4.25wu »
 *   « Sensor Duration: 6.5 -> 15 secondes » · « Reveal Duration: 2.5 -> 0.75 secondes »
 *   -- Halo Waypoint, « Sandbox Overview Season 4 », article `/fr-fr/news/sandbox-overview-season-4`
 *
 * (Le domaine n'est pas écrit ici, et ce n'est pas un oubli : le garde-rail I19
 * `waypointUrl.guard.test.ts` réserve ce littéral à `waypointUrl.ts`. Le chemin de l'article
 * suffit à retrouver la source — ne pas recoller l'URL entière, la suite tomberait.)
 *
 * L'INTENTION QUE LA SOURCE DONNE À CES VALEURS éclaire le rendu : « create more
 * location-based marking, giving more permanence and coverage ». C'est un marquage de LIEU,
 * pas de personne — d'où l'appartenance mesurée À L'INSTANT DU PING (plus bas).
 *
 * L'UNITÉ EST UNE SUPPOSITION, ET ELLE EST ÉCRITE. Les positions du rejeu sont en mètres
 * monde ; « wu » est l'unité monde de Slipspace, SUPPOSÉE ici valoir 1 m. 4,25 m de rayon
 * (8,5 m de diamètre) est plausible pour un capteur d'arène ; l'autre conversion parfois
 * citée (3,048 m/wu) donnerait 26 m de diamètre, invraisemblable sur une carte de 50 m. Le
 * jour où une portée serait MESURÉE dans le film, elle remplacerait la constante — et la
 * remplacerait seule.
 *
 * IL PINGE, IL NE RESPIRE PAS (correction du 2026-08-17). Le premier rendu faisait osciller
 * le rayon du disque de ±6 % autour d'une portée déclarée à 8 m : une respiration lente, qui
 * ne disait rien du jeu et donnait au capteur une zone deux fois trop large. Le capteur du jeu
 * émet une onde toutes les 1,8 s ; c'est cette onde qui est dessinée, à la portée officielle.
 *
 * FONCTION DU TEMPS, PAS ANIMATION À ÉTAT (même règle que `explosionFx`) : la lecture avance,
 * recule, saute au curseur. Le même âge rend toujours la même image, donc un retour en arrière
 * rejoue exactement ce qu'on avait vu. Sous `prefers-reduced-motion`, l'onde ne part pas — le
 * disque et son anneau restent, et la marque de révélation cesse de s'estomper.
 *
 * CE QUE LE PING RÉVÈLE. À chaque ping, les vies ADVERSES du poseur qui se trouvent dans le
 * rayon À L'INSTANT DU PING portent une marque pendant 0,75 s. Trois choix, tous conséquences
 * de la source :
 *  - L'APPARTENANCE SE MESURE AU PING, jamais à l'image courante : la source parle de
 *    « location-based marking », et 0,75 s de course valent ~3 m à la vitesse de base d'un
 *    Spartan, soit les trois quarts du rayon. Tester « maintenant » marquerait les entrants
 *    et démarquerait les sortants — exactement l'inverse de ce que le capteur affirme.
 *  - LA MARQUE SUIT LA VIE : elle est dessinée à la position COURANTE, comme le contour du
 *    jeu suit sa cible. Un halo laissé trois mètres derrière un point qui bouge se lirait
 *    comme un défaut d'affichage.
 *  - ELLE NE SURVIT PAS À SA CIBLE : une vie qui se termine pendant la fenêtre perd sa marque
 *    à l'image même. La mort a déjà son calque, la marque n'a rien à y ajouter.
 *
 * LE CAMP EST CELUI DE LA BASE (`team_side`), PAS « allié / adverse ». Le drapeau d'alliance
 * est relatif au joueur de la page : il rangerait tout le monde en deux camps, ce qui est faux
 * en mêlée générale et en BTB à quatre camps. Ce qu'il faut savoir ici, c'est si DEUX vies
 * s'opposent. Sans camp connu — poseur non mesuré (`owner < 0`), ou vie sans ligne de
 * scoreboard — AUCUNE révélation : le ping se dessine quand même (il est mesuré, lui), et
 * rien n'affirme une inimitié dont on ne sait rien.
 *
 * COÛT PAR IMAGE, MESURÉ SUR LE CORPUS LOCAL (27 artefacts, 2026-08-17). Un seul en porte :
 * `000d5950`, 19 capteurs, tous avec poseur mesuré — au plus DEUX actifs en même temps, et au
 * plus deux dans une fenêtre de révélation (moyenne 0,055 par image, 0,103 actifs). Le
 * balayage coûte donc, au pire de ce corpus : 2 x 99 tests de fenêtre de vie (O(1)) puis
 * 2 x 8 interpolations de position (dichotomie) — négligeable devant les 228 primitives d'UNE
 * explosion. La borne large du corpus (BTB : 256 vies, 28 simultanées) tiendrait sous
 * 8 x 256 tests même avec huit capteurs actifs.
 *
 * Pas de React, pas de couleur écrite : géométrie et arithmétique pures (l'encre vient du
 * calque appelant, qui la tient des tokens du thème).
 */
import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import { isAliveAt, positionAt } from './replayLogic'
import type { ReplayTrackReady } from './replayNormalize'

/**
 * Rayon du capteur, en mètres monde. VALEUR OFFICIELLE (« Sensor Radius: 4.25wu »), sous la
 * supposition d'unité écrite en tête : 1 wu = 1 m.
 */
export const SENSOR_RADIUS_M = 4.25

/** Période du ping, en millisecondes. VALEUR OFFICIELLE (« Ping Frequency: 1.8 secondes »). */
export const SENSOR_PING_MS = 1_800

/**
 * Durée de la marque « révélé », en millisecondes. VALEUR OFFICIELLE (« Reveal Duration:
 * 0.75 secondes ») : elle est BRÈVE, et c'est le sens de l'ajustement de la saison 4 — la
 * révélation ne suit plus longtemps sa cible, c'est le capteur qui dure et couvre un lieu.
 */
export const SENSOR_REVEAL_MS = 750

/**
 * Durée de vie officielle du capteur, en millisecondes (« Sensor Duration: 15 secondes »).
 *
 * ELLE BORNE CE QUI S'AFFICHE, et ce n'est plus le choix qui figurait ici. Le commentaire
 * précédent lisait la fenêtre `[t0, t1]` du document comme la RÉPLICATION de l'objet et en
 * concluait qu'un capteur « que le film ne réplique plus » ne devait pas être prolongé. La
 * mesure du 2026-08-18 (`filmdec/equipment_life_end_test.go`) l'a réfuté : `t1` date l'instant
 * où l'objet cesse de BOUGER — le décodage ne suit que les records porteurs d'une position —
 * et le recensement des keyframes montre l'entité bien vivante après (101 poses sur 295 du
 * film `000d5950`). Les 2,1 s de vie médiane d'un capteur sont donc la durée de sa CHUTE, pas
 * de son service.
 *
 * Le film ne datant aucune disparition, la seule durée disponible est celle-ci : officielle,
 * citée verbatim en tête de fichier, du même genre que la portée et la cadence qui gouvernent
 * déjà tout ce tracé. Elle sert aussi de CONTRÔLE de coût (cf. SENSOR_PINGS_DECLARED).
 */
export const SENSOR_DURATION_MS = 15_000

/**
 * Nombre de pings d'une vie officielle : aux âges 0, 1,8 s, ... 14,4 s — soit NEUF. C'est la
 * borne de coût d'un capteur, au-delà il n'existe plus.
 */
export const SENSOR_PINGS_DECLARED = Math.floor(SENSOR_DURATION_MS / SENSOR_PING_MS) + 1

/**
 * Course de l'onde d'un ping, en millisecondes : le temps qu'elle met à atteindre le rayon.
 *
 * CE N'EST PAS UNE MESURE et la source n'en dit rien : c'est un choix de RYTHME. 400 ms sur
 * une période de 1 800 laissent l'onde se lire comme un coup (elle occupe moins du quart du
 * cycle) puis rendent le disque au repos — une onde qui remplirait la période donnerait un
 * capteur qui gonfle en continu, soit la respiration qu'on vient de retirer.
 */
export const SENSOR_SWEEP_MS = 400

/** Remplissage du disque : très bas — c'est une zone, elle ne doit rien masquer. */
export const SENSOR_FILL_ALPHA = 0.1
/** Anneau fixe au rayon : la bordure de la zone, présente entre deux pings. */
export const SENSOR_RING_ALPHA = 0.55
export const SENSOR_RING_WIDTH = 1.25
/** Opacité de départ de l'onde du ping : elle claque, puis s'efface en s'ouvrant. */
export const SENSOR_PING_ALPHA = 0.9
/** Épaisseur de l'onde : plus franche que l'anneau fixe, c'est elle l'événement. */
export const SENSOR_PING_WIDTH = 2

/**
 * Rayon de la marque « révélé » autour de la vie, en pixels d'ÉCRAN.
 *
 * 12 px LA POSE HORS DE PORTÉE DES AUTRES ANNEAUX du marqueur de joueur : son noyau monte à
 * 4,8 px, son liseré à 5,8, l'anneau d'identité à ~7,4 et les anneaux d'étage à 6,5 et 9,3
 * (cf. replayMarkers.ts). Elle s'en distingue aussi par la COULEUR — les anneaux d'étage sont
 * à l'encre du thème, la marque porte la teinte de l'équipe du poseur.
 */
export const REVEAL_RADIUS_PX = 12
/** Remplissage du halo : bas, il éclaire le joueur sans effacer la carte sous lui. */
export const REVEAL_FILL_ALPHA = 0.18
/** Opacité du liseré au ping ; elle décroît jusqu'à l'extinction de la marque. */
export const REVEAL_ALPHA = 0.8
export const REVEAL_WIDTH = 1.5

/** Ce que le capteur a besoin de savoir de l'instant courant. */
export interface SensorTime {
  /** Image courante (fractionnaire pendant la lecture). */
  frame: number
  /** Durée RÉELLE d'une frame, en ms (`frameToMs(1, doc)`), jamais un compteur d'images. */
  frameMs: number
}

/** Les vies du film, et le camp de chacune — tout ce que la révélation a besoin de lire. */
export interface SensorScene {
  /** Toutes les vies : le balayage filtre lui-même sur la fenêtre de vie. */
  lives: readonly ReplayTrackReady[]
  /** Camp d'une vie (`team_side` de la base) ; null = camp inconnu, donc jamais un ennemi. */
  sideOfSlot: (slot: number) => string | null
}

/** L'onde du ping courant : où elle en est de sa course, et ce qu'il lui reste d'opacité. */
export interface SensorPingWave {
  /** Rayon atteint, en FRACTION du rayon du capteur — dans [0, 1[. */
  reach: number
  /** Opacité du trait, décroissante : l'onde s'efface en s'ouvrant. */
  alpha: number
}

/** Une vie RÉVÉLÉE par un ping, à l'image courante. */
export interface SensorReveal {
  /** Slot de la vie révélée. */
  slot: number
  /** Sa position MONDE à l'image COURANTE : la marque suit le joueur (cf. en-tête). */
  x: number
  y: number
  /** Slot du POSEUR : la marque porte SA couleur d'équipe — c'est son camp qui voit. */
  owner: number
  /** Temps écoulé depuis le ping qui l'a révélée, en ms — dans [0, SENSOR_REVEAL_MS]. */
  sinceMs: number
}

/**
 * sensorPingAgeMs — temps écoulé depuis le DERNIER ping, à l'âge donné.
 *
 * Le premier ping part à l'âge zéro : le capteur balaie dès qu'il touche le sol. Un âge
 * négatif (la pose n'existe pas encore) rend zéro plutôt qu'une phase inversée — mais le
 * calque ne l'appelle jamais hors de la fenêtre [t0, t1].
 */
export function sensorPingAgeMs(ageMs: number): number {
  if (!(ageMs > 0)) return 0
  return ageMs % SENSOR_PING_MS
}

/**
 * sensorPing — l'onde du ping courant, ou null quand elle a fini sa course : entre deux
 * pings, le capteur ne montre plus que son disque et son anneau.
 *
 * L'ouverture suit un easeOutCubic — l'onde part vite du centre puis se calme au bord, le
 * patron de l'onde de choc de l'explosion (`explosionFx.drawWave`), pour la même raison :
 * c'est ainsi qu'une impulsion se lit. Sous mouvement réduit, aucune onde.
 */
export function sensorPing(ageMs: number, reducedMotion: boolean): SensorPingWave | null {
  if (reducedMotion) return null
  const since = sensorPingAgeMs(ageMs)
  if (since >= SENSOR_SWEEP_MS) return null
  const pr = since / SENSOR_SWEEP_MS
  return { reach: 1 - Math.pow(1 - pr, 3), alpha: (1 - pr) * SENSOR_PING_ALPHA }
}

/**
 * revealAlpha — l'opacité de la marque à son âge : franche au ping, éteinte à 0,75 s.
 *
 * Sous mouvement réduit elle NE S'ESTOMPE PAS : la marque reste une information (« cet
 * adversaire a été détecté »), et une opacité fixe la garde sans la faire palpiter.
 */
export function revealAlpha(sinceMs: number, reducedMotion: boolean): number {
  if (reducedMotion) return REVEAL_ALPHA
  const pr = Math.min(1, Math.max(0, sinceMs / SENSOR_REVEAL_MS))
  return REVEAL_ALPHA * (1 - pr)
}

/**
 * sensorReveals — les vies révélées par les capteurs ACTIFS à cette image.
 *
 * `sensors` est la liste des poses de famille `sensor` déjà retenues par la table de rendu et
 * actives à l'image (c'est le calque qui la constitue, il boucle déjà dessus) : cette fonction
 * ne décide ni du rendu d'une famille ni de la fenêtre d'une pose, elle décide de la
 * révélation. Tout le reste — camp, portée, instant de mesure — est écrit en tête du fichier.
 */
export function sensorReveals(
  sensors: readonly ReplayEquipmentPlacement[],
  scene: SensorScene,
  time: SensorTime,
): SensorReveal[] {
  const r2 = SENSOR_RADIUS_M * SENSOR_RADIUS_M
  const bySlot = new Map<number, SensorReveal>()
  for (const s of sensors) {
    // SANS POSEUR MESURÉ, AUCUNE RÉVÉLATION : on ignore le camp du capteur, donc personne
    // n'est « ennemi » de lui. Le ping, lui, continue de se dessiner.
    if (s.owner < 0) continue
    const side = scene.sideOfSlot(s.owner)
    if (side === null) continue
    const sinceMs = sensorPingAgeMs((time.frame - s.t0) * time.frameMs)
    if (sinceMs > SENSOR_REVEAL_MS) continue
    // L'image du PING, d'où l'appartenance est mesurée. Fractionnaire : `positionAt`
    // interpole, et un ping ne tombe pas sur une frame ronde.
    const pingFrame = time.frameMs > 0 ? time.frame - sinceMs / time.frameMs : time.frame
    for (const life of scene.lives) {
      // Les deux tests de fenêtre d'abord : O(1) chacun, et ils écartent 99 vies sur 8
      // avant toute dichotomie de position (cf. le coût en tête).
      if (!isAliveAt(life, pingFrame)) continue
      if (!isAliveAt(life, time.frame)) continue
      const other = scene.sideOfSlot(life.slot)
      if (other === null || other === side) continue
      const atPing = positionAt(life.points, pingFrame)
      if (!atPing) continue
      const dx = atPing.x - s.x
      const dy = atPing.y - s.y
      if (dx * dx + dy * dy > r2) continue
      const now = positionAt(life.points, time.frame)
      if (!now) continue
      // DEUX CAPTEURS PEUVENT VOIR LA MÊME VIE : une seule marque, la PLUS FRAÎCHE. Deux
      // halos superposés doubleraient l'opacité sans rien dire de plus.
      const held = bySlot.get(life.slot)
      if (held && held.sinceMs <= sinceMs) continue
      bySlot.set(life.slot, { slot: life.slot, x: now.x, y: now.y, owner: s.owner, sinceMs })
    }
  }
  return [...bySlot.values()]
}
