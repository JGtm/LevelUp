/**
 * matchClock.ts — LES TROIS AXES DE TEMPS D'UN MATCH, ET LES CONVERSIONS ENTRE EUX.
 *
 * POURQUOI CE MODULE EXISTE. La page match et le rejeu 2D posent des faits datés côte à
 * côte alors que ces faits ne comptent PAS depuis le même instant. Le contrat serveur le
 * dit sans détour (`apps/go-api/internal/domain/match_view.go`, champ `T0Ms`) : « les
 * events servis par cette page sont recalés sur le début du GAMEPLAY tandis que le film —
 * donc le rejeu 2D — part du début du MATCH, countdown compris. Sans cet offset, un
 * consommateur qui pose les deux sur la même timeline décale les kills de ~18 à 28 s. »
 * Ce module est ce consommateur, écrit une fois.
 *
 * LES TROIS AXES, ET CE QUI LES ANCRE.
 *
 *   1. L'AXE DU MATCH — millisecondes depuis `header.start_time`, l'instant que l'API date.
 *      C'est l'axe des horodatages absolus (médias, entrées/sorties de partie).
 *   2. L'AXE DU FILM — millisecondes depuis l'IMAGE ZÉRO de l'artefact, c'est-à-dire depuis
 *      son premier paquet de position. `doc.originMs` dit où tombe cette image zéro sur
 *      l'axe du match ; l'écart mesuré va de 3,6 s à 50,8 s selon le match, jamais zéro.
 *      Une image `t` vaut `t × frameIntervalMs` sur cet axe.
 *   3. L'AXE DU GAMEPLAY — millisecondes depuis le COUP D'ENVOI. C'est l'axe de
 *      `event_time_ms` : le serveur a déjà retranché le countdown (`correctMatchViewEventsT0`),
 *      et `header.t0_ms` est précisément la valeur retranchée. D'où l'identité qui fonde
 *      tout ce fichier : sur cet axe, `event_time_ms` se lit TEL QUEL, sans terme correctif.
 *
 * LES DEUX SOUSTRACTIONS, ET RIEN D'AUTRE.
 *
 *     msFilm     = event_time_ms + t0_ms − originMs          (le contrat Go, mot pour mot)
 *     msGameplay = frame × frameIntervalMs + originMs − t0_ms
 *
 * La seconde est l'inverse de la première ; les poser toutes deux ici est ce qui empêche
 * qu'une surface en applique une et sa voisine aucune.
 *
 * DEUX COUPS D'ENVOI COEXISTENT, ET CE MODULE ANCRE SUR CELUI DE L'API. `header.t0_ms` est
 * ESTIMÉ des `first_joined_time` ; `doc.t0FilmMs` est MESURÉ dans le film (premier mouvement
 * réel), et le rejeu 2D le préfère pour SON horloge affichée (`replayWindow.ts`, décision D5
 * du plan T0-film, 2026-09-02). L'axe du gameplay défini ici, lui, est ancré sur
 * `header.t0_ms` et ne peut pas l'être ailleurs : c'est l'ancre que le producteur a
 * retranchée pour fabriquer `event_time_ms`. Y substituer le T0 film déplacerait les frags
 * cumulés — la série qui est déjà juste. Les deux ancres sont donc portées côte à côte, et
 * `t0FilmMs` reste disponible pour qui affiche une horloge de LECTURE.
 *
 * SANS ORIGINE, PAS D'HORLOGE. `doc.originMs` absent (artefact antérieur au schéma 4, ou
 * origine que le producteur a REFUSÉ d'établir — chunk illisible, témoin contradictoire) :
 * l'écart entre les deux axes est un inconnu de 3,6 à 50,8 s, et `matchClock` rend `null`.
 * Zéro n'est pas un repli acceptable, le producteur l'écrit lui-même
 * (`internal/analysis/replay/origin.go` : « ZERO N'EST PAS UNE ORIGINE NEUTRE, C'EST UN
 * REPLI »). Mesure du parc local au 2026-09-05 : 5 artefacts sur 106 sans origine, et les
 * cinq portent déjà `coverage.originResolved: false` — ils sont donc déjà écartés en amont
 * par `filmClockTrusted`, et cette porte-ci ne retire aucune carte de l'écran.
 *
 * IL NE CONNAÎT AUCUNE FEATURE. Ce qu'il attend d'un document est décrit STRUCTURELLEMENT
 * ici (`MatchClockDocument`) : `ReplayDocumentReady` (match-replay) s'y conforme sans que
 * `lib/` ait à le nommer. Même règle que `scoreTimeline.ts`, et pour la même raison — deux
 * features en sont clientes à égalité.
 *
 * Tout ce fichier est PUR : ni React, ni canvas, ni DOM.
 */
import type { MatchViewHeader } from '@/lib/api/types'

/**
 * Le strict minimum qu'un document de rejeu doit porter pour qu'une horloge s'établisse.
 * Structurel, jamais nominal (cf. l'en-tête).
 */
export interface MatchClockDocument {
  /** Instant de l'image zéro du film sur l'axe du MATCH. Absent = pas d'horloge. */
  originMs?: number
  /** Coup d'envoi MESURÉ dans le film, sur l'axe du match. Absent avant le schéma 36. */
  t0FilmMs?: number
  /** Durée réelle d'une image. Absente = pas d'échelle temporelle, pas d'horloge. */
  frameIntervalMs?: number
  frameCount: number
}

/** Ce que l'horloge lit de l'en-tête du match : l'ancre de `event_time_ms`. */
export type MatchClockHeader = Pick<MatchViewHeader, 't0_ms'>

/** Une horloge ÉTABLIE : les quatre grandeurs, et les conversions qui en découlent. */
export interface MatchClock {
  /** `doc.originMs` — l'image zéro du film sur l'axe du match. */
  originMs: number
  /**
   * `header.t0_ms` — le countdown d'avant-match, en ms, tel que le serveur l'a retranché
   * pour produire `event_time_ms`. Zéro quand il est inconnu : la correction n'a alors pas
   * eu lieu côté events non plus, et ne rien ajouter est exactement juste.
   */
  t0Ms: number
  /**
   * `doc.t0FilmMs` — le coup d'envoi MESURÉ dans le film, sur l'axe du match, ou `null`.
   * L'axe du gameplay de ce module ne s'y ancre PAS (cf. l'en-tête) ; il est porté ici pour
   * les horloges de LECTURE, qui le préfèrent à l'estimation de l'API.
   */
  t0FilmMs: number | null
  frameIntervalMs: number
  frameCount: number
  /**
   * gameplayMsOfFilmMs — un instant de l'axe du FILM, lu sur l'horloge du GAMEPLAY.
   * `filmMs + originMs − t0Ms`. Négatif avant le coup d'envoi : le film déborde le match
   * du countdown, et ce module ne l'écrase pas — c'est à l'affichage de borner s'il veut.
   */
  gameplayMsOfFilmMs(filmMs: number): number
  /** gameplayMsOfFrame — une IMAGE du film, lue sur l'horloge du gameplay. */
  gameplayMsOfFrame(frame: number): number
  /**
   * filmMsOfMatchMs — un instant de l'axe du MATCH, lu sur l'axe du FILM. `matchMs −
   * originMs`. C'EST LA SOUSTRACTION UNIQUE du rejeu 2D : les médias, les entrées/sorties
   * de partie, les relais de siège et le fil des éliminations la font tous, et c'est de
   * l'avoir écrite cinq fois que naissaient trois réponses différentes (P0-5).
   */
  filmMsOfMatchMs(matchMs: number): number
  /**
   * frameOfFilmMs — l'IMAGE d'un instant du film, arrondie à l'image la plus proche et
   * jamais négative : rien ne se joue avant l'image zéro.
   */
  frameOfFilmMs(filmMs: number): number
}

/**
 * matchClock rend l'horloge du match, ou `null` quand elle n'est pas établie : artefact
 * sans origine, sans échelle temporelle, ou sans deux images à mettre bout à bout.
 *
 * UN SEUL VERDICT POUR TOUTE UNE PAGE : les surfaces qui datent quelque chose l'appellent
 * une fois et se dégradent TOUTES de la même façon quand elle rend `null`. C'est ce que la
 * dispersion des politiques de repli avait cassé (registre 2026-09-05, P0-5/P0-7).
 */
export function matchClock(
  doc: MatchClockDocument | null | undefined,
  header: MatchClockHeader | null | undefined,
): MatchClock | null {
  if (!doc) return null
  const originMs = doc.originMs
  const frameIntervalMs = doc.frameIntervalMs
  if (originMs == null || !Number.isFinite(originMs)) return null
  if (!frameIntervalMs || !Number.isFinite(frameIntervalMs)) return null
  if (!(doc.frameCount > 1)) return null

  const t0Ms = Number.isFinite(header?.t0_ms) ? (header?.t0_ms as number) : 0
  const t0FilmMs = Number.isFinite(doc.t0FilmMs) ? (doc.t0FilmMs as number) : null
  // L'ÉCART ENTRE LES DEUX AXES, calculé une fois : c'est lui, et lui seul, que les deux
  // conversions appliquent. Positif quand le film commence AVANT le coup d'envoi (le cas
  // ordinaire, 18 à 28 s de countdown), négatif sur un film qui démarre après (témoin
  // e94163af : origine 39 772 ms contre un T0 de 35 238 ms, soit 4,5 s de retard).
  const filmToGameplayMs = originMs - t0Ms
  const gameplayMsOfFilmMs = (filmMs: number): number => filmMs + filmToGameplayMs
  return {
    originMs,
    t0Ms,
    t0FilmMs,
    frameIntervalMs,
    frameCount: doc.frameCount,
    gameplayMsOfFilmMs,
    gameplayMsOfFrame: (frame: number) => gameplayMsOfFilmMs(frame * frameIntervalMs),
    filmMsOfMatchMs: (matchMs: number) => matchMs - originMs,
    frameOfFilmMs: (filmMs: number) => Math.max(0, Math.round(filmMs / frameIntervalMs)),
  }
}
