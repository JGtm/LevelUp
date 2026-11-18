/**
 * reliefBackground — LES IMAGES DU FOND que le relief produit : le relief teinté, et un fond Plan ou
 * Rendu dont la coupe par étage module l'opacité.
 *
 * Le calculateur (ce dossier) rend des tableaux d'octets au calage du fond ; ce module les
 * pose dans des canvas, qui se dessinent comme n'importe quelle image de fond (`drawMapBackground` :
 * même rectangle, même rotation par quarts de tour). Rien de géométrique ici.
 *
 *   - `paintReliefImage` : le relief TEINTÉ (RGBA, l'opacité de la coupe y est déjà) ;
 *   - `paintMaskedImage` : l'image de fond (Plan ou Rendu) dont l'opacité est multipliée par le masque de
 *     la coupe (`destination-in` d'un canvas dont l'alpha est le masque : la composition reste sur le GPU).
 *
 * LES CANVAS SE RÉUTILISENT d'une coupe à l'autre (`target`) : un curseur qui glisse ne doit pas allouer
 * 11 Mo à chaque pas. Il rend `null` quand le navigateur refuse le contexte 2D (le fond de base reste posé).
 */
import type { ReliefFrame } from './engine'

/** Un canvas de la taille demandée : celui qu'on réutilise s'il a déjà cette taille, un neuf sinon. */
function canvasOfSize(target: HTMLCanvasElement | null, width: number, height: number): HTMLCanvasElement {
  const canvas = target ?? document.createElement('canvas')
  if (canvas.width !== width || canvas.height !== height) {
    canvas.width = width
    canvas.height = height
  }
  return canvas
}

/** paintReliefImage pose le relief teinté de `frame` dans `target` (ou un canvas neuf). `null` sans image teintée ou sans contexte. */
export function paintReliefImage(target: HTMLCanvasElement | null, frame: ReliefFrame): HTMLCanvasElement | null {
  if (!frame.image) return null
  const canvas = canvasOfSize(target, frame.width, frame.height)
  const ctx = canvas.getContext('2d')
  if (!ctx) return null
  ctx.putImageData(new ImageData(frame.image as Uint8ClampedArray<ArrayBuffer>, frame.width, frame.height), 0, 0)
  return canvas
}

/** Un canvas dont l'alpha est `mask` (un octet par pixel) et dont la couleur ne compte pas. */
function maskCanvas(mask: Uint8Array, width: number, height: number): HTMLCanvasElement | null {
  const canvas = canvasOfSize(null, width, height)
  const ctx = canvas.getContext('2d')
  if (!ctx) return null
  const rgba = new Uint8ClampedArray(width * height * 4)
  for (let p = 0, q = 3; p < mask.length; p++, q += 4) rgba[q] = mask[p]
  ctx.putImageData(new ImageData(rgba, width, height), 0, 0)
  return canvas
}

/**
 * paintMaskedImage copie `image` (le fond Plan ou Rendu, à la taille du calage) dans `target` en
 * multipliant son opacité par `mask`. `null` sans contexte : l'appelant garde l'image d'origine.
 */
export function paintMaskedImage(
  target: HTMLCanvasElement | null,
  image: CanvasImageSource,
  mask: Uint8Array,
  width: number,
  height: number,
): HTMLCanvasElement | null {
  const canvas = canvasOfSize(target, width, height)
  const ctx = canvas.getContext('2d')
  const masker = maskCanvas(mask, width, height)
  if (!ctx || !masker) return null
  ctx.save()
  ctx.globalCompositeOperation = 'source-over'
  ctx.clearRect(0, 0, width, height)
  ctx.drawImage(image, 0, 0, width, height)
  ctx.globalCompositeOperation = 'destination-in'
  ctx.drawImage(masker, 0, 0)
  ctx.restore()
  return canvas
}

/**
 * Les deux canvas d'un fond Relief, tenus ensemble : celui du relief teinté et celui du fond Plan ou
 * Rendu modulé. Chacun se réutilise d'une frame à l'autre (cf. l'en-tête) ; un objet à part, parce que le
 * hook qui les produit les range dans un état et ne manipule aucune référence pendant le rendu.
 */
export class ReliefCanvases {
  private relief: HTMLCanvasElement | null = null
  private masked: HTMLCanvasElement | null = null

  /** Le relief teinté de `frame`, ou `null`. */
  reliefImage(frame: ReliefFrame): HTMLCanvasElement | null {
    this.relief = paintReliefImage(this.relief, frame)
    return this.relief
  }

  /** `image` modulée par le masque de `frame` pour le fond posé (`plan` : masque du plan ; sinon celui du rendu), ou `null` sans masque. */
  maskedImage(frame: ReliefFrame, image: CanvasImageSource, plan: boolean): HTMLCanvasElement | null {
    const mask = plan ? frame.planAlpha : frame.alpha
    if (!mask) return null
    this.masked = paintMaskedImage(this.masked, image, mask, frame.width, frame.height)
    return this.masked
  }
}
