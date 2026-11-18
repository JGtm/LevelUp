/**
 * useReliefView — LE FOND RELIEF, SES TRAITS ET LA COUPE PAR ÉTAGE d'une vue 2D sans panneau « Vue » (l'onglet
 * Tactique, « Occupation du terrain »), assemblés en un seul hook.
 *
 * L'appelant lui prête ce que sa vue sait : où lire les deux fichiers du relief (`files`, `null` = la carte
 * n'en a pas), le fond de base déjà chargé (le plan ou le rendu), le fond VOULU et les traits retenus. Le hook
 * rend ce que la vue pose : l'image qui prend la place du fond de base (`surface`), les traits à tracer par-dessus
 * (`linesLayer`), la commande d'étage et les fonds à proposer. Il suit la chaîne du rejeu (`useReplayRelief`) :
 * fichiers -> calculateur (`useReliefRender`) -> images (`ReliefCanvases`).
 *
 * LES TROIS FONDS : le Relief n'est posé que s'il est lisible. Sur une `ReliefError` (fichier absent, version
 * inconnue, image illisible, panne du calculateur), le repli est le PLAN, puis le rendu : la lecture le journalise
 * (`[relief-view]`), l'écran ne dit rien. Pendant la lecture du relief, le fond de base reste posé.
 *
 * LA COUPE vaut pour les trois fonds : sur le Relief par son opacité, sur le Plan et le Rendu par un masque
 * d'opacité du relief. Elle est un état de CETTE vue (jamais persistée : « Tout » à l'ouverture, et pour chaque
 * autre carte, cf. `useScopedCut`). Sans maillage, il n'y a pas de coupe.
 *
 * L'IMAGE DES HAUTEURS, la plus lourde des deux fichiers, n'est demandée que lorsqu'une commande s'en sert
 * (fond Relief voulu, traits allumés, coupe choisie) ; les données se lisent dès que `files` est là.
 */
import { useCallback, useEffect, useMemo, useState } from 'react'

import { reliefUsable, resolveBackgroundVariant, type MapBackgroundVariant } from './mapBackgroundVariant'
import type { QuarterTurns } from './orientation'
import { meshShareOf, quantizeCut, reliefRamp, rgbToCss, type FloorMesh, type ReliefData, type ReliefFrame } from './relief'
import { ReliefCanvases } from './relief/reliefBackground'
import type { ReliefLinesInk, ReliefLinesInput } from './relief/reliefLinesLayer'
import { useReliefFiles, type ReliefFilesSource } from './reliefQueries'
import { viewBackgroundOptions, viewFloorState, reliefViewRequest, type ViewFloorState } from './reliefViewLogic'
import { useReliefRender } from './useReliefRender'

/** Les encres du relief, résolues par jeton par l'appelant (`useReliefInks`). */
export interface ReliefViewInks {
  /** `map-relief-base` : la teinte de base de la rampe. */
  base: string
  /** Le filet clair du cerne des pions (`marker-outline-rim`) : le liseré de la zone jouable. */
  outline: string
}

export interface ReliefViewOptions {
  /** Où lire les fichiers du relief ; `null` = le sidecar du fond n'en annonce pas, rien n'est demandé. */
  files: ReliefFilesSource | null
  /** La coupe retenue (`null` = « Tout »), tenue par l'appelant (`useScopedCut`) : il sait ainsi s'il doit décoder le fond de base. */
  cut: number | null
  onCut: (z: number | null) => void
  /** Le style de fond choisi par l'utilisateur, avant tout repli. */
  style: MapBackgroundVariant
  /** Le fond VOULU (`resolveBackgroundVariant` sans échec de lecture) : il décide des traits et de l'image à charger. */
  wanted: MapBackgroundVariant
  planAvailable: boolean
  /** Les traits du relief sont allumés (retenus par fond, `useReliefLines`). */
  linesOn: boolean
  /** L'orientation EFFECTIVE de la carte : elle ne règle que la direction de la lumière. */
  turns: QuarterTurns
  ink: ReliefViewInks
  /** Le fond de base chargé (plan ou rendu), à moduler par la coupe ; `null` tant qu'il n'est pas là. */
  baseImage: CanvasImageSource | null
  /** Le fond de base est l'image plan (son masque de coupe est celui du plan). */
  baseIsPlan: boolean
}

/** La commande d'étage et la coupe retenue (`null` = « Tout »). */
export interface ReliefViewFloor {
  state: ViewFloorState
  cut: number | null
  selectAll: () => void
  /** Retient un étage par sa hauteur exacte (celle d'une pastille). */
  selectRank: (z: number) => void
  /** Retient une hauteur libre, arrondie au quart d'unité (le curseur). */
  setHeight: (z: number) => void
}

/**
 * L'image que le relief pose à la place du fond de base. `image` est un canvas RÉUTILISÉ d'une frame à l'autre (un curseur
 * qui glisse n'alloue pas 11 Mo par pas) : c'est `frame` qui change quand ses pixels changent, pour qui doit les recopier.
 */
export interface ReliefSurface {
  image: HTMLCanvasElement
  frame: ReliefFrame
}

/** Ce que le hook rend. */
export interface ReliefView {
  /** Le fond RETENU après les replis (relief illisible -> plan -> rendu) : ce que la pilule montre. */
  selected: MapBackgroundVariant
  /** Les fonds que la pilule propose ; vide quand il n'y a pas de choix. */
  options: readonly MapBackgroundVariant[]
  /** La part du relief tirée du maillage (D-16), en pourcentage entier ; `null` quand elle est inconnue. */
  meshPercent: number | null
  /** Les données du relief, lues et utilisables ; `null` sinon. */
  data: ReliefData | null
  /** Le maillage des sols, pour juger des positions ; `null` sans données ou sans maillage. */
  mesh: FloorMesh | null
  /** L'interrupteur « Traits du relief » est proposé (les données sont lues). */
  linesAvailable: boolean
  floor: ReliefViewFloor
  /** L'image qui prend la place du fond de base : le relief teinté, ou le fond de base modulé par la coupe. `null` = le fond de base. */
  surface: ReliefSurface | null
  /** `surface` est le relief teinté : le fond posé est le Relief (le plan des murs s'y trace). */
  reliefShown: boolean
  /** Les traits à tracer par-dessus ; `null` = pas de traits. */
  linesLayer: ReliefLinesInput | null
}

const NO_FLOOR: ViewFloorState = { state: 'none' }

/** Les encres des traits, tirées de la base de teinte ; `null` quand la base n'est pas une couleur lisible. */
function linesInkOf(ink: ReliefViewInks): ReliefLinesInk | null {
  const ramp = reliefRamp(ink.base)
  if (!ramp) return null
  return { edge: rgbToCss(ramp.edge), contour: rgbToCss(ramp.contour), wall: rgbToCss(ramp.wall), outline: ink.outline }
}

export function useReliefView(o: ReliefViewOptions): ReliefView {
  const { files: source, cut, onCut: setCut, style, wanted, planAvailable, linesOn, turns, ink, baseImage, baseIsPlan } = o
  const offered = source !== null
  const needImage = offered && (wanted === 'relief' || linesOn || cut !== null)
  const files = useReliefFiles(source, needImage)
  const request = useMemo(
    () => reliefViewRequest({ needed: needImage, turns, cut, tintBase: wanted === 'relief' ? ink.base : null, lines: linesOn }),
    [needImage, turns, cut, wanted, ink.base, linesOn],
  )
  const render = useReliefRender(files.data, files.blob, request)

  // LE REPLI : une lecture qui échoue rend le plan. L'erreur des FICHIERS se journalise ici, celle du calculateur
  // par `useReliefRender`.
  const failed = files.error !== null || render.status === 'failed'
  useEffect(() => {
    if (files.error !== null) console.warn(`[relief-view] lecture des fichiers du relief impossible, repli sur le plan : ${String(files.error)}`)
  }, [files.error])
  const usable = reliefUsable({ available: offered, failed })
  const selected = resolveBackgroundVariant(style, planAvailable, { available: offered, failed })

  const [canvases] = useState(() => new ReliefCanvases())
  const frame = render.frame
  const surface = useMemo<ReliefSurface | null>(() => {
    if (!frame || !usable) return null
    let image: HTMLCanvasElement | null = null
    if (selected === 'relief') image = canvases.reliefImage(frame)
    else if (cut !== null && baseImage) image = canvases.maskedImage(frame, baseImage, baseIsPlan)
    return image ? { image, frame } : null
  }, [frame, usable, selected, cut, baseImage, baseIsPlan, canvases])

  const data = usable ? files.data : null
  const linesInk = useMemo(() => linesInkOf(ink), [ink])
  const linesLayer = useMemo<ReliefLinesInput | null>(() => {
    if (!linesOn || !data || !frame?.lines || !linesInk) return null
    return {
      calibration: data.calibration,
      steps: frame.lines.steps,
      contours: frame.lines.contours,
      walls: frame.lines.walls,
      outline: data.lines.outline,
      ink: linesInk,
    }
  }, [linesOn, data, frame, linesInk])

  const state = useMemo(() => (data ? viewFloorState(data) : NO_FLOOR), [data])
  const selectAll = useCallback(() => setCut(null), [setCut])
  const selectRank = useCallback((z: number) => setCut(z), [setCut])
  const setHeight = useCallback((z: number) => setCut(quantizeCut(z)), [setCut])
  return {
    selected,
    options: viewBackgroundOptions(planAvailable, usable),
    meshPercent: data ? (meshShareOf(data)?.percent ?? null) : null,
    data,
    mesh: data && state.state === 'ready' ? data.floors.mesh : null,
    linesAvailable: data !== null,
    floor: { state, cut: state.state === 'ready' ? cut : null, selectAll, selectRank, setHeight },
    surface,
    reliefShown: surface !== null && selected === 'relief',
    linesLayer,
  }
}
