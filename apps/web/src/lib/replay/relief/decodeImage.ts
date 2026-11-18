/**
 * relief/decodeImage — L'IMAGE DU RELIEF : des pixels RGBA aux hauteurs et aux drapeaux.
 *
 * FORMAT (conception §1.1) : `A` vaut 255 quand la hauteur est connue, 0 sinon (jamais une valeur
 * intermédiaire) ; `q = R + 256 × G` ; hauteur = `z0 + q × step` ; `B` porte les drapeaux (bit 0 sur
 * la carte, bit 1 sol du maillage, les autres à zéro). `A = 0` impose `R = G = B = 0`.
 *
 * POURQUOI LE CONTROLE (§6.1) : le canevas 2D range ses pixels prémultipliés et un navigateur peut
 * convertir les couleurs ; une alpha intermédiaire écraserait R, G, B à la relecture. Le format n'a
 * que deux alphas pour cette raison, et la lecture VERIFIE sur tous les pixels que rien n'a bougé :
 * un seul manquement et le relief est tenu pour illisible (erreur typée, l'appelant retombe sur le
 * Plan). C'est le seul filet contre un navigateur qui aurait converti les couleurs.
 *
 * LECTURE : `createImageBitmap` SANS conversion de couleur ni prémultiplication, dessiné à taille
 * 1:1 sur un canevas sans lissage. Si le navigateur refuse les options (vieux WebKit), repli sur un
 * élément image, avec le même contrôle. jsdom n'a pas `createImageBitmap` : la logique testée est
 * `decodeReliefPixels`, qui prend un tableau de pixels ; les chargeurs se testent par injection.
 *
 * Ce fichier touche au DOM : il ne fait PAS partie du graphe du worker.
 */
import { FILE_FLAG_MASK, FLAG_KNOWN } from './constants'
import { ReliefError } from './errors'
import type { ReliefCalibration, ReliefHeightMeta, ReliefHeights } from './types'

/** Des pixels RGBA lus tels quels, 4 octets par pixel, ligne par ligne. */
export interface RawPixels {
  width: number
  height: number
  data: ArrayLike<number>
}

/** Une image décodée dont on peut relire les pixels et libérer la mémoire. */
export interface PixelSource {
  width: number
  height: number
  readPixels(): RawPixels
  release(): void
}

/** Les deux façons de décoder une image ; injectables pour les tests. */
export interface ReliefImageLoaders {
  /** `createImageBitmap` sans conversion de couleur ni prémultiplication. */
  loadBitmap(blob: Blob): Promise<PixelSource>
  /** Repli : un élément image. */
  loadElement(blob: Blob): Promise<PixelSource>
}

/**
 * decodeReliefPixels décode et CONTROLE des pixels RGBA : dimensions du calage, A ∈ {0, 255}, B ne
 * porte que les deux drapeaux du fichier, A = 0 implique R = G = B = 0. Le premier manquement lève
 * une `ReliefError`.
 */
export function decodeReliefPixels(
  image: RawPixels,
  calibration: ReliefCalibration,
  meta: ReliefHeightMeta,
): ReliefHeights {
  const { width, height, data } = image
  if (width !== calibration.widthPx || height !== calibration.heightPx) {
    throw new ReliefError(
      'image-size',
      `image ${width}x${height} au lieu de ${calibration.widthPx}x${calibration.heightPx}`,
    )
  }
  const n = width * height
  if (data.length !== n * 4) {
    throw new ReliefError('image-size', `${data.length} octets pour ${n} pixels`)
  }
  const heights = new Float32Array(n)
  const flags = new Uint8Array(n)
  const { z0, step } = meta
  for (let p = 0, q = 0; p < n; p++, q += 4) {
    const a = data[q + 3]
    const b = data[q + 2]
    if (a === 255) {
      if ((b & ~FILE_FLAG_MASK) !== 0) {
        throw new ReliefError('image-flags', `pixel ${p} : drapeaux ${b} (bits réservés posés)`)
      }
      heights[p] = z0 + (data[q] + 256 * data[q + 1]) * step
      flags[p] = b | FLAG_KNOWN
    } else if (a === 0) {
      if (data[q] !== 0 || data[q + 1] !== 0 || b !== 0) {
        throw new ReliefError('image-flags', `pixel ${p} : hauteur inconnue mais couleur non nulle`)
      }
      heights[p] = NaN
    } else {
      throw new ReliefError('image-alpha', `pixel ${p} : alpha intermédiaire ${a}`)
    }
  }
  return { width, height, heights, flags }
}

/** Les chargeurs du navigateur. */
export const domLoaders: ReliefImageLoaders = {
  loadBitmap: loadWithBitmap,
  loadElement: loadWithElement,
}

/**
 * readReliefImage lit l'image du relief depuis son blob (obtenu par `api.getBlob`, seul client
 * HTTP autorisé) et la décode. Toute défaillance rend une `ReliefError` : l'appelant retombe sur le
 * Plan. Le repli sur l'élément image n'a lieu que si `createImageBitmap` échoue ; le contrôle des
 * pixels est le même dans les deux cas.
 */
export async function readReliefImage(
  blob: Blob,
  calibration: ReliefCalibration,
  meta: ReliefHeightMeta,
  loaders: ReliefImageLoaders = domLoaders,
): Promise<ReliefHeights> {
  const source = await openSource(blob, loaders)
  try {
    if (source.width !== calibration.widthPx || source.height !== calibration.heightPx) {
      throw new ReliefError(
        'image-size',
        `image ${source.width}x${source.height} au lieu de ${calibration.widthPx}x${calibration.heightPx}`,
      )
    }
    return decodeReliefPixels(source.readPixels(), calibration, meta)
  } catch (err) {
    if (err instanceof ReliefError) throw err
    throw new ReliefError('image-decode', `relecture des pixels impossible : ${describe(err)}`)
  } finally {
    source.release()
  }
}

async function openSource(blob: Blob, loaders: ReliefImageLoaders): Promise<PixelSource> {
  try {
    return await loaders.loadBitmap(blob)
  } catch (err) {
    console.warn(`[replay-relief] createImageBitmap refusé, repli sur l'élément image : ${describe(err)}`)
  }
  try {
    return await loaders.loadElement(blob)
  } catch (err) {
    throw new ReliefError('image-decode', `image du relief refusée : ${describe(err)}`)
  }
}

function describe(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

async function loadWithBitmap(blob: Blob): Promise<PixelSource> {
  const bitmap = await createImageBitmap(blob, { colorSpaceConversion: 'none', premultiplyAlpha: 'none' })
  return {
    width: bitmap.width,
    height: bitmap.height,
    readPixels: () => readCanvasPixels(bitmap, bitmap.width, bitmap.height),
    release: () => bitmap.close(),
  }
}

async function loadWithElement(blob: Blob): Promise<PixelSource> {
  const url = URL.createObjectURL(blob)
  try {
    const img = await new Promise<HTMLImageElement>((resolve, reject) => {
      const el = new Image()
      el.onload = () => resolve(el)
      el.onerror = () => reject(new Error('élément image : chargement refusé'))
      el.src = url
    })
    const { naturalWidth: width, naturalHeight: height } = img
    return {
      width,
      height,
      readPixels: () => readCanvasPixels(img, width, height),
      release: () => URL.revokeObjectURL(url),
    }
  } catch (err) {
    URL.revokeObjectURL(url)
    throw err
  }
}

/** Dessine la source à taille 1:1, sans lissage, et relit les pixels (jamais à une autre échelle). */
function readCanvasPixels(source: CanvasImageSource, width: number, height: number): RawPixels {
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d', { willReadFrequently: true, colorSpace: 'srgb' })
  if (!ctx) throw new Error('contexte 2D indisponible')
  ctx.imageSmoothingEnabled = false
  ctx.drawImage(source, 0, 0)
  return { width, height, data: ctx.getImageData(0, 0, width, height).data }
}
