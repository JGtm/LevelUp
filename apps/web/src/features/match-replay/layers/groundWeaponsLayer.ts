/**
 * groundWeaponsLayer.ts — LES ARMES ABANDONNÉES AU SOL sur la carte du rejeu (schéma 27).
 *
 * CE QUE CE CALQUE DESSINE, ET CE QU'IL NE DESSINE PAS. Une entrée de `groundWeapons` est UN
 * OBJET du monde : une arme qui a BOUGÉ — lâchée par un mort, abandonnée en ramassant autre
 * chose, éjectée d'un râtelier — puis s'est arrêtée quelque part. On la pose là où elle gît, et
 * on l'y laisse tant que le film l'y montre. Les armes DE SOCLE ne passent pas par ici : elles
 * appartiennent au calque `weaponPads`, qui les publie par grappes récurrentes ; les dessiner
 * deux fois ferait deux vérités pour un même objet.
 *
 * IL N'Y A NI LOSANGE NI COMPTE À REBOURS, et c'est la distinction qui porte toute la lecture
 * de la carte : le losange de `weaponPadsLayer` dit un LIEU du terrain — un emplacement qui
 * réapprovisionne — et son compteur dit quand il se remplira. Une arme au sol n'est pas un
 * lieu : elle ne revient pas, elle est là ou elle n'y est plus. Elle se dessine donc SEULE,
 * sans socle sous elle, plus petite et plus discrète que la vignette d'un socle.
 *
 * L'ESTOMPAGE EST LA MESURE, PAS UN EFFET : l'opacité vient de `groundWeaponPresenceAt`
 * (groundWeaponTime.ts), qui tient le plein tant qu'une preuve de présence tient et descend
 * pendant l'intervalle où le film ne dit plus rien. Rien n'est peint au-delà de la première
 * preuve d'ABSENCE.
 *
 * SANS VIGNETTE, RIEN N'EST DESSINÉ — jamais l'icône d'une arme voisine, jamais un glyphe de
 * repli. C'est le verdict du 2026-08-26 sur les socles (« j'ai l'impression qu'il y en a
 * deux »), et il vaut a fortiori ici : le socle gardait son losange pour dire le lieu, une arme
 * au sol n'a que son image — un point anonyme de plus sur la carte n'apprendrait rien et se
 * confondrait avec les objets lâchés du calque des poses.
 *
 * DEUXIÈME COPIE DU GESTE « VIGNETTE CERNÉE » (CLAUDE.md n°6). La première est
 * `weaponPadsLayer.drawPadIcon`. Les deux partagent le principe — reposer la SILHOUETTE tout
 * autour du corps, un canvas ne sachant pas cerner une image — et divergent sur les tailles,
 * qui sont le sujet de chaque calque. À la TROISIÈME, centraliser et poser le garde-rail.
 *
 * Pas de React : géométrie pure + un CanvasRenderingContext2D, comme les calques voisins.
 * L'encre arrive de l'appelant, qui la tient des variables du thème.
 */
import type { ReplayGroundWeapon } from '@/lib/api/types'

import { groundWeaponsAt } from '../model/groundWeaponTime'
import { project, type PlacementView } from './placementShapes'
import type { XY } from '../model/replayLogic'

/** Le cadrage est celui des poses et des socles : les trois projettent la même scène. */
export type { PlacementView as GroundWeaponView } from './placementShapes'

/**
 * Hauteur de la vignette, en pixels d'écran.
 *
 * PLUS PETITE QUE LA PLUS PETITE VIGNETTE DE SOCLE (8 px, `PAD_ICON_H_PX.classic`) : une arme
 * abandonnée est un fait secondaire du terrain, et la hiérarchie doit se voir sans qu'on ait à
 * comparer deux objets côte à côte. Assez grande, cependant, pour qu'une silhouette d'arme reste
 * reconnaissable — en deçà de 6 px, toutes les armes se ressemblent.
 */
const GROUND_ICON_H_PX = 6.5

/** Une vignette d'arme est large : au-delà de ce rapport, la largeur est bornée. */
const GROUND_ICON_MAX_ASPECT = 3.2

/**
 * Épaisseur du LISERÉ, en pixels d'écran, et le nombre de directions où on repose la
 * silhouette pour l'obtenir. Huit : à quatre, les diagonales laissent passer le fond.
 *
 * PLUS FIN QUE CELUI DES SOCLES (1,2 px) parce que la vignette est plus petite : à taille
 * réduite, un liseré épais mange la forme qu'il est censé détacher.
 */
const GROUND_OUTLINE_PX = 0.9
const GROUND_OUTLINE_STEPS = 8

/**
 * Une vignette prête à poser : son CORPS et son LISERÉ, déjà teints hors écran.
 *
 * DEUX IMAGES ET NON UNE, pour la même raison que les socles : un canvas ne sait pas cerner une
 * image, le liseré s'obtient en reposant la même forme, teinte de l'encre du fond, tout autour
 * du corps.
 */
export interface GroundWeaponIcon {
  fill: CanvasImageSource
  outline: CanvasImageSource
}

/** Ce que le calque emprunte au thème et au catalogue du document. */
export interface GroundWeaponStyle {
  /**
   * Vignette TEINTE de la famille, ou null — le calque ne dessine alors rien pour cet objet
   * (cf. l'en-tête). La clé est `groundWeapons[].w`, le MÊME espace d'identifiants que
   * `loadouts[].w` et `weaponPads[].weapon`.
   */
  iconOf: (weapon: string) => GroundWeaponIcon | null
}

/** Ce que le calque a besoin de savoir de l'instant courant. */
export interface GroundWeaponTime {
  frame: number
  /** Densité de pixels : les épaisseurs d'écran la suivent (même règle que les marqueurs). */
  k: number
}

/**
 * drawGroundIcon — la vignette posée À PLAT sur la position de repos, remplie et cernée.
 *
 * Hauteur imposée, largeur déduite du rapport de l'image (bornée : une vignette d'arme est très
 * large). Le liseré est la même silhouette reposée tout autour : c'est ce qui la détache d'un
 * fond de carte clair comme d'un fond sombre — sans lui, une arme au sol disparaît dans le gris
 * des cartes reconstruites.
 */
function drawGroundIcon(
  ctx: CanvasRenderingContext2D,
  centre: XY,
  h: number,
  icon: GroundWeaponIcon,
  k: number,
): void {
  const source = icon.fill
  const natW = 'width' in source && typeof source.width === 'number' ? source.width : 0
  const natH = 'height' in source && typeof source.height === 'number' ? source.height : 0
  const aspect = natW > 0 && natH > 0 ? Math.min(natW / natH, GROUND_ICON_MAX_ASPECT) : 1
  const w = h * aspect
  const x = centre.x - w / 2
  const y = centre.y - h / 2
  const d = GROUND_OUTLINE_PX * k
  for (let i = 0; i < GROUND_OUTLINE_STEPS; i++) {
    const a = (i / GROUND_OUTLINE_STEPS) * Math.PI * 2
    ctx.drawImage(icon.outline, x + Math.cos(a) * d, y + Math.sin(a) * d, w, h)
  }
  ctx.drawImage(source, x, y, w, h)
}

/**
 * drawGroundWeaponsLayer trace les armes au sol VISIBLES à l'image courante.
 *
 * L'ORDRE DE LA LISTE EST CELUI DU DOCUMENT (trié par apparition côté serveur) : deux armes
 * tombées au même endroit se recouvrent dans l'ordre où elles y sont arrivées, la plus récente
 * au-dessus. Aucun arbitrage n'est fait ici — il n'y a rien à arbitrer, les deux sont vraies.
 */
export function drawGroundWeaponsLayer(
  ctx: CanvasRenderingContext2D,
  items: readonly ReplayGroundWeapon[],
  view: PlacementView,
  time: GroundWeaponTime,
  style: GroundWeaponStyle,
): void {
  if (items.length === 0 || view.width === 0) return
  const visible = groundWeaponsAt(items, time.frame)
  if (visible.length === 0) return
  ctx.save()
  const h = GROUND_ICON_H_PX * time.k
  for (const { item, presence } of visible) {
    const icon = style.iconOf(item.w)
    if (!icon) continue
    ctx.globalAlpha = presence.alpha
    drawGroundIcon(ctx, project({ x: item.x, y: item.y }, view), h, icon, time.k)
  }
  ctx.globalAlpha = 1
  ctx.restore()
}
