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
 * Événements ponctuels. Le LANCER emprunte un token d'information ; le TIR, lui, ne prend plus
 * aucun token de données : sa couleur dit la NATURE DE LA DÉCHARGE et vient des teintes
 * diégétiques du thème (fxInk.ts, décision utilisateur du 2026-08-15). Le token d'alerte reste
 * employé par les effets de MORT, qui n'ont pas changé.
 */
const SHOT_TOKEN: SemanticToken = 'destructive'
const GRENADE_TOKEN: SemanticToken = 'info'

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
  /** Sol reconstruit ; `edge` sert aussi d'encre NEUTRE à tout ce qui n'a pas de camp. */
  floor: ReplayFloorInk
  /** Teintes des éclairs de bouche, lues une fois par thème (jamais par image). */
  fx: FxInk
  /** Ligne de grappin : l'encre la plus claire du thème (`--foreground`), jamais un hex. */
  grapple: string
  /** Contour des étiquettes : SOMBRE dans les deux thèmes (cf. globals.css). */
  labelStroke: string
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
      fx: readFxInk(),
      grapple: readInk('--foreground'),
      labelStroke: readInk('--replay-label-stroke'),
    }
  }, [paletteVersion])
}
