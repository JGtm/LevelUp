/**
 * useReplayInks — TOUTES LES ENCRES DU REJEU, résolues une fois par palette.
 *
 * POURQUOI CE FICHIER EXISTE. Neuf `useMemo` partageaient exactement la même amorce dans
 * `ReplayCanvas.tsx` — `void paletteVersion` puis un `resolveToken` ou un `readInk` — et le
 * canvas porte une dette de taille GELÉE par un cliquet (`placementFamily.guard.test.ts`,
 * « prochaine addition : extraire d'abord »). Les encres sont le bloc qui part le plus
 * proprement : elles ne connaissent ni le document, ni l'image courante, ni un seul réglage.
 *
 * POURQUOI UN SEUL OBJET MÉMOÏSÉ, et pas neuf valeurs indépendantes : `draw` est un
 * `useCallback` dont la liste de dépendances cite ces encres une par une. Un objet reconstruit
 * à chaque rendu recuirait le calque soixante fois par seconde ; l'objet est donc mémoïsé, et
 * les valeurs qu'on en destructure sont stables tant que la palette ne bouge pas.
 *
 * `paletteVersion` EST RENDU AVEC ELLES : l'appelant en a encore besoin pour les couleurs de
 * ZONES (`getSeriesColors`), qui dépendent du nombre de grandes zones — une donnée que ce hook
 * n'a pas à connaître.
 */
import { useMemo } from 'react'

import { resolveToken } from '@/lib/accessibility/resolveToken'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'

import { readInk } from './canvasInk'
import { readFxInk, type FxInk } from './fxInk'
import type { FloorStyle } from './replayDraw'

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

/**
 * LE MARQUEUR DU JOUEUR DE LA PAGE (V1, retour utilisateur du 2026-08-18 : « avoir l'icône du
 * joueur actif qui se démarque de tous les autres »).
 *
 * `success` EST LE VERT DU DÉPÔT, et c'est le vert qui a été demandé — « j'aurais bien aimé du
 * vert, mais pour l'accessibilité je sais pas si ça peut le faire ». La réponse est : oui, à
 * condition que la couleur ne soit JAMAIS SEULE. Elle ne teint ici que le DOUBLE CONTOUR et le
 * halo, deux signes de FORME qui se lisent sans elle ; le noyau garde la couleur d'ÉQUIPE, qui
 * dit le camp. Un lecteur qui ne distingue pas ce vert voit toujours le seul marqueur de la
 * carte qui porte deux anneaux.
 */
const SELF_TOKEN: SemanticToken = 'success'

/** Les encres du rejeu, toutes résolues — plus la version de palette qui les a produites. */
export interface ReplayInks {
  paletteVersion: number
  /** Les deux couleurs d'ÉQUIPE réglées par l'utilisateur (D1) : allié / adversaire. */
  teamColorOf: (isAlly: boolean) => string
  geometryColor: string
  shotColor: string
  grenadeColor: string
  /** Encres de mise en page du sol : elles suivent le thème, pas la palette d'accessibilité. */
  floorStyle: FloorStyle
  /** Teintes des ÉCLAIRS DE BOUCHE, lues UNE fois par thème (jamais par image). */
  fxInk: FxInk
  /** La LIGNE DE GRAPPIN : l'encre la plus claire du thème, jamais un hex. */
  grappleInk: string
  /** Contour des noms : sombre dans les DEUX thèmes (cf. globals.css). */
  labelStroke: string
  /** Double contour et halo du joueur de la page (cf. SELF_TOKEN). */
  selfInk: string
}

export function useReplayInks(): ReplayInks {
  const paletteVersion = useColorPaletteVersion()
  return useMemo(() => {
    // `void paletteVersion` : `resolveToken` et `readInk` lisent le DOM, la version de palette
    // est ce qui force la re-résolution au changement de thème ou de réglage d'accessibilité.
    void paletteVersion
    const ally = resolveToken('team-ally')
    const enemy = resolveToken('team-enemy')
    const geometryColor = resolveToken(GEOMETRY_TOKEN)
    return {
      paletteVersion,
      teamColorOf: (isAlly: boolean) => (isAlly ? ally : enemy),
      geometryColor,
      shotColor: resolveToken(SHOT_TOKEN),
      grenadeColor: resolveToken(GRENADE_TOKEN),
      floorStyle: { fill: geometryColor, edge: readInk('--muted-foreground') },
      fxInk: readFxInk(),
      grappleInk: readInk('--foreground'),
      labelStroke: readInk('--replay-label-stroke'),
      selfInk: resolveToken(SELF_TOKEN),
    }
  }, [paletteVersion])
}
