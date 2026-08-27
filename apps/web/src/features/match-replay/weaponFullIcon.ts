/**
 * weaponFullIcon.ts — LA VERSION PLEINE d'une icône d'arme, pour les fiches joueur.
 *
 * POURQUOI CE MODULE EXISTE. Le document du rejeu porte une vignette par arme
 * (`weaponLabels[id].img`, cuite dans l'artefact au build) : c'est l'atlas `contour` —
 * le trait de l'arme, à vide. La maquette retenue (option 2a du handoff 2026-08-27)
 * dessine les fiches avec la version PLEINE, celle du kill feed du jeu : l'atlas
 * `silhouette`, extrait du même binaire, indexé À L'IDENTIQUE — `silhouette-XX` est la
 * même arme que `contour-XX`, mêmes tags `weap` (vérifié 40/40 par le garde-rail
 * `weaponFullIcon.guard.test.ts` sur `jeu/index.json`). L'échange est donc une simple
 * substitution de stem dans l'URL.
 *
 * IL SE RÉSOUT CÔTÉ CLIENT, ET C'EST DÉLIBÉRÉ : l'URL est FIGÉE DANS L'ARTEFACT au moment
 * du build — servir la silhouette depuis le Go n'atteindrait que les artefacts recuits,
 * c'est-à-dire aucun de ceux déjà servis, tant que le re-build de masse n'a pas eu lieu.
 *
 * LE MIROIR. Les atlas `contour` et `silhouette` pointent vers la GAUCHE ; le kill feed
 * du jeu — et la maquette — vers la DROITE. L'icône échangée se rend donc retournée
 * (`scaleX(-1)`), et SEULEMENT elle : une vignette hors atlas (les deux PNG dessinés à la
 * main, les concepts) garde son URL et son sens — retourner un dessin fini n'a pas de
 * fondement dans le jeu.
 *
 * SEULES LES FICHES passent par ici : les râteliers d'armes de la carte
 * (`useReplayWeaponPads`) gardent le contour — ils sont hors du périmètre du handoff.
 */
import type { CSSProperties } from 'react'

/** Les deux stems d'atlas, tels que `static/weapons-assets/{slug}/jeu/` les nomme. */
const CONTOUR_SEG = '/jeu/contour-'
const SILHOUETTE_SEG = '/jeu/silhouette-'

/** Ce que la fiche a besoin de savoir : l'URL à peindre, et si elle se retourne. */
export interface FullWeaponIcon {
  url: string
  /** Vrai = icône d'atlas échangée, à rendre en miroir (le sens du kill feed du jeu). */
  mirrored: boolean
}

/** Le style du miroir — une seule écriture, partagée par les cellules qui le posent. */
export const MIRROR_STYLE: CSSProperties = { transform: 'scaleX(-1)' }

/**
 * weaponFullIcon échange une URL d'atlas `contour` contre sa `silhouette` (version
 * pleine, à retourner). Toute autre URL — dessin fini, concept, autre titre — est rendue
 * telle quelle, dans son sens.
 */
export function weaponFullIcon(url: string): FullWeaponIcon {
  if (url.includes(CONTOUR_SEG)) {
    return { url: url.replace(CONTOUR_SEG, SILHOUETTE_SEG), mirrored: true }
  }
  return { url, mirrored: false }
}
