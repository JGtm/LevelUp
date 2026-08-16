/**
 * grenadeIcon.ts — LA VIGNETTE D'UN TYPE DE GRENADE sur les fiches joueur.
 *
 * POURQUOI CE MODULE EXISTE. Le document du rejeu porte déjà une vignette par rang
 * (`grenadeLabels[rank].img`, cuite dans l'artefact au build) : ce sont les MASQUES de HUD
 * extraits du jeu, teints à l'encre du thème. Ils marchent — mais la planche du 16/08 a
 * demandé les IMAGES VERSIONNÉES, celles de `static/grenades-assets/{slug}/`, qui existent
 * en deux encres (claire pour le thème sombre, sombre pour le thème clair) et rendent le
 * dessin du jeu plutôt qu'une silhouette.
 *
 * ELLES SE RÉSOLVENT CÔTÉ CLIENT, ET C'EST DÉLIBÉRÉ. Les libellés sont FIGÉS DANS
 * L'ARTEFACT au moment du build : les faire pointer les nouvelles images depuis le Go
 * n'atteindrait que les artefacts reconstruits — c'est-à-dire aucun de ceux qui sont déjà
 * servis, tant que le re-build de masse n'a pas eu lieu. La jointure vit donc ici, sur le
 * NOM ANGLAIS du rang, qui EST la clé de l'index d'assets (`frag`, `plasma`, `dynamo`,
 * `spike`).
 *
 * TITLE-AGNOSTIC PAR CONSTRUCTION : aucun slug n'est écrit ici, il vient de l'appelant, et
 * un titre dont les rangs ne portent aucun de ces noms retombe simplement sur la vignette
 * de son document. Rien n'est deviné : un nom hors catalogue n'emprunte pas l'image d'un
 * voisin (même règle que les libellés, cf. catalogLabel.ts).
 */
import { staticAssetURL } from '@/lib/staticAssets'

import type { CatalogLabel } from './catalogLabel'

/**
 * Les stems LIVRÉS dans `static/grenades-assets/{slug}/` (un par type, deux encres chacun).
 *
 * ÉCRITE PLUTÔT QUE SONDÉE : le client ne demande jamais au serveur si une image existe
 * (pas de 404 exploratoire, même règle que le manifeste sonore). Une image manquante serait
 * un cadre vide inexpliqué ; un stem oublié, une vignette qui retombe en silence sur le
 * masque de HUD. Le garde-rail `grenadeIcon.guard.test.ts` rejoue cette liste contre le
 * dossier d'assets ET contre son `index.json`.
 */
export const GRENADE_ICON_STEMS = ['frag', 'plasma', 'dynamo', 'spike'] as const

/** Encre d'une vignette : le nom de la variante, tel que le dossier d'assets l'écrit. */
export type GrenadeIconInk = 'light' | 'dark'

/**
 * L'ENCRE SUIT LE FOND, ET NON LE NOM DU THÈME : `*_light.png` est le dessin CLAIR, celui
 * qui se lit sur un fond sombre. Un thème clair reçoit donc `*_dark.png`. L'inversion est
 * exactement le piège que ce commentaire existe pour éviter.
 */
export function grenadeIconInk(theme: string): GrenadeIconInk {
  return theme === 'light' ? 'dark' : 'light'
}

/** Ce qu'une fiche a besoin de savoir pour peindre la vignette : l'URL, et son mode. */
export interface GrenadeIconRef {
  url: string
  /** Vrai = masque à teindre (le repli de HUD) ; faux = image finie (l'asset versionné). */
  tinted: boolean
}

/**
 * grenadeIconOf choisit la vignette d'un rang, dans cet ordre :
 *   1. l'image VERSIONNÉE du type, à l'encre du thème — le rendu demandé le 16/08 ;
 *   2. à défaut, la vignette CUITE dans l'artefact (masque de HUD teint) ;
 *   3. à défaut, rien : l'appelant garde le libellé, jamais l'image d'un autre type.
 */
export function grenadeIconOf(
  label: CatalogLabel | undefined,
  titleSlug: string,
  theme: string,
): GrenadeIconRef | null {
  const stem = (label?.en ?? '').trim().toLowerCase()
  if ((GRENADE_ICON_STEMS as readonly string[]).includes(stem)) {
    const url = staticAssetURL('grenade', `${stem}_${grenadeIconInk(theme)}`, '.png', titleSlug)
    if (url) return { url, tinted: false }
  }
  return label?.img ? { url: label.img, tinted: !!label.tinted } : null
}
