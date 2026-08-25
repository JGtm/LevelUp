/**
 * useReplayInks — LES ENCRES DU REJEU, résolues une fois par palette.
 *
 * POURQUOI CE FICHIER EXISTE. Huit valeurs de couleur du canvas partageaient EXACTEMENT le même
 * corps — `void paletteVersion` puis un `resolveToken` ou un `readInk`, mémoïsé sur la version
 * de palette — recopié huit fois dans `ReplayCanvas.tsx`. C'est la 3e copie de la règle
 * CLAUDE.md n°6 largement dépassée, et c'est aussi ce qui a fait grossir le canvas jusqu'à son
 * cliquet de taille (placementFamily.guard.test.ts) : l'extraction est la façon prescrite d'y
 * ajouter un calque, pas le relèvement du plafond.
 *
 * UN SEUL MÉMO POUR LES HUIT, ET C'EST MIEUX QUE HUIT : elles dépendaient toutes de la même
 * chose, elles changent donc toutes en même temps. Les consommateurs y gagnent des références
 * stables entre deux rendus — ce dont les tableaux de dépendances du canvas ont besoin.
 *
 * DEUX SOURCES DE COULEUR, ET LEUR DIFFÉRENCE EST DU SENS. `resolveToken` sert les couleurs qui
 * DISENT quelque chose (une équipe, un lancer, une mort) : ce sont des tokens sémantiques, et
 * ils suivent la palette d'accessibilité que l'utilisateur a réglée. `readInk` sert la mise en
 * page (arête d'une marche, contour d'une étiquette) : ce sont les variables du système de
 * design, jamais un littéral (cf. canvasInk.ts). Aucune valeur écrite en dur ici.
 */
import { useMemo } from 'react'

import { resolveToken } from '@/lib/accessibility/resolveToken'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

import { readInk } from './canvasInk'
import { readFxInk, type FxInk } from './fxInk'

/** Fond de carte : token neutre, sans connotation directionnelle (le sujet = les joueurs). */
const GEOMETRY_TOKEN: SemanticToken = 'divergent-neutral'
/**
 * L'ENCRE DU « AUCUN CAMP », et c'est un TOKEN parce que ça DIT quelque chose.
 *
 * Le rejeu servait jusqu'ici `floor.edge` — c'est-à-dire `--muted-foreground` via `readInk` —
 * partout où il n'y avait pas de camp. C'était l'entorse que `canvasInk.ts` s'interdit
 * lui-même en toutes lettres : « toute couleur qui DIT quelque chose (une équipe, un tir, une
 * mort) passe par un token ». « Personne ne tient cette zone » dit quelque chose — c'est même
 * une MESURE du film (valeur neutre du canal de propriété), pas de la mise en page.
 *
 * `divergent-neutral` EST DÉJÀ le neutre sémantique de la feature : le fil des éliminations
 * l'emploie pour une mort que personne ne revendique. Les objectifs sans camp parlent donc
 * maintenant la même langue que le fil, et suivent la palette d'accessibilité de
 * l'utilisateur — ce que `readInk` ne fait pas.
 *
 * MÊME TOKEN QUE `GEOMETRY_TOKEN`, ET DEUX CONSTANTES QUAND MÊME : le fond de carte est neutre
 * parce qu'il n'est pas le sujet, une zone est neutre parce que personne ne la tient. Deux
 * raisons distinctes, qui doivent pouvoir diverger sans qu'on ait à démêler laquelle on change.
 */
const NEUTRAL_TOKEN: SemanticToken = 'divergent-neutral'
/**
 * Événements ponctuels. Le LANCER emprunte un token d'information ; le TIR, lui, ne prend plus
 * aucun token de données : sa couleur dit la NATURE DE LA DÉCHARGE et vient des teintes
 * diégétiques du thème (fxInk.ts, décision utilisateur du 2026-08-15). Le token d'alerte reste
 * employé par les effets de MORT, qui n'ont pas changé.
 */
const SHOT_TOKEN: SemanticToken = 'destructive'
const GRENADE_TOKEN: SemanticToken = 'info'

/**
 * LE MARQUEUR DU JOUEUR DE LA PAGE (lot R2-V, retour utilisateur du 2026-08-18 : « avoir
 * l'icône du joueur actif qui se démarque de tous les autres »).
 *
 * `success` EST LE VERT DU DÉPÔT, et c'est le vert qui a été demandé — « j'aurais bien aimé du
 * vert, mais pour l'accessibilité je sais pas si ça peut le faire ». La réponse est : oui, à
 * condition que la couleur ne soit JAMAIS SEULE. Elle ne teint ici que le DOUBLE CONTOUR et le
 * halo, deux signes de FORME qui se lisent sans elle ; le noyau garde la couleur d'ÉQUIPE, qui
 * dit le camp. Un lecteur qui ne distingue pas ce vert voit toujours le seul marqueur de la
 * carte qui porte deux anneaux.
 */
const SELF_TOKEN: SemanticToken = 'success'

/**
 * LE MUR DE PROTECTION (W1/R2-5, verdict du 2026-08-18 : « je préférerais un orange doré »).
 *
 * DEUX TOKENS ÉTAIENT CANDIDATS et la planche les a montrés côte à côte : `legendary` (l'or,
 * déjà porté par l'encadré du surbouclier) et `warning` (l'orange d'alerte). `warning` est
 * retenu. Ce qu'on PERD est écrit : la couleur d'ÉQUIPE du poseur — le mur ne dit plus QUI
 * l'a posé. Ce qu'on gagne : il dit QUOI, et un objet du terrain qui garde sa teinte quel que
 * soit le camp se reconnaît d'un coup d'œil au milieu de huit trajectoires colorées.
 */
const WALL_TOKEN: SemanticToken = 'warning'

/** Les encres de mise en page du sol reconstruit : son aplat et l'arête de ses marches. */
export interface ReplayFloorInk {
  fill: string
  edge: string
}

export interface ReplayInks {
  /** Les DEUX couleurs d'équipe telles que l'utilisateur les a réglées (décision D1). */
  teamColorOf: (isAlly: boolean) => string
  /** Props Forge du fond de carte, quand aucun sol reconstruit n'est disponible. */
  geometry: string
  /** Effets de MORT (le tir, lui, prend sa teinte de la décharge : cf. `fx`). */
  shot: string
  /** Lancers de grenade. */
  grenade: string
  /** Sol reconstruit ; `edge` sert aussi d'encre de mise en page à ce qui n'a pas de rôle. */
  floor: ReplayFloorInk
  /** Encre SÉMANTIQUE du « aucun camp » : objectifs neutres, zone que personne ne tient. */
  neutral: string
  /** Teintes des éclairs de bouche, lues une fois par thème (jamais par image). */
  fx: FxInk
  /** Ligne de grappin : l'encre la plus claire du thème (`--foreground`), jamais un hex. */
  grapple: string
  /** Contour des étiquettes : SOMBRE dans les deux thèmes (cf. globals.css). */
  labelStroke: string
  /** Double contour et halo du joueur de la page (cf. SELF_TOKEN). */
  self: string
  /** Arc du mur de protection : un token FIXE, plus la couleur d'équipe (cf. WALL_TOKEN). */
  wall: string
  /**
   * LE MARQUAGE DES SOCLES : ce qui est REMPLI, et ce qui le CERNE (verdict du 2026-08-18 —
   * « icône blanche remplie, contour noir »). Ce sont les deux encres du THÈME, pas des
   * tokens de données : en sombre le remplissage est clair et le liseré sombre, en clair
   * l'inverse. La demande se lit donc telle quelle sur les deux fonds de carte, ce qu'un
   * blanc et un noir écrits en dur n'auraient pas fait.
   */
  mark: { fill: string; outline: string }
}

/**
 * useReplayInks résout toutes les encres du rejeu pour la palette courante.
 *
 * `paletteVersion` vient de `useColorPaletteVersion()` chez l'appelant — il l'observe déjà pour
 * ses propres couleurs de série. Le passer plutôt que de le relire ici garde UN observateur de
 * style pour la page, et rend la dépendance visible à la lecture.
 */
export function useReplayInks(paletteVersion: number): ReplayInks {
  return useMemo(() => {
    void paletteVersion
    const ally = resolveToken('team-ally')
    const enemy = resolveToken('team-enemy')
    return {
      teamColorOf: (isAlly: boolean) => (isAlly ? ally : enemy),
      geometry: resolveToken(GEOMETRY_TOKEN),
      shot: resolveToken(SHOT_TOKEN),
      grenade: resolveToken(GRENADE_TOKEN),
      floor: { fill: resolveToken(GEOMETRY_TOKEN), edge: readInk('--muted-foreground') },
      neutral: resolveToken(NEUTRAL_TOKEN),
      fx: readFxInk(),
      grapple: readInk('--foreground'),
      labelStroke: readInk('--replay-label-stroke'),
      self: resolveToken(SELF_TOKEN),
      wall: resolveToken(WALL_TOKEN),
      mark: { fill: readInk('--foreground'), outline: readInk('--background') },
    }
  }, [paletteVersion])
}
