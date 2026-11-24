/**
 * useVehicleSprites — LES SPRITES DU CALQUE DES VÉHICULES (`useReplayVehicles`) : chargement des images
 * sources par famille, lecture du manifeste de mise à l'échelle, bordures à la demande, vignettes
 * teintées cuites hors écran, et taille d'une famille.
 *
 * Extrait de `useReplayVehicles` pour tenir la règle des 80 lignes par fonction. TOUT CE QUI EST
 * ASYNCHRONE EST LU DANS DES RÉFÉRENCES, jamais dans un état React (cf. l'en-tête de
 * `useReplayVehicles`) : un chargement qui aboutit déclenche seulement `redraw()`.
 */
import { useCallback, useEffect, useRef } from 'react'

import { staticAssetURL } from '@/lib/staticAssets'

import { withLoadedImage } from './loadedImage'
import { outlinedSpriteCanvas, tintedIconCanvas } from './replayDraw'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import {
  parseVehicleManifest,
  vehicleBodyPx,
  vehicleManifestAssetId,
  type VehicleManifestEntry,
} from '../model/vehicleSpriteManifest'
import { vehicleIsDecor } from '../model/vehiclesLayer'
import type { VehicleSpriteSize } from './vehiclesPaint'

interface VehicleSpritesInput {
  /** Faux quand le calque est éteint : rien n'est chargé. */
  enabled: boolean
  labels: ReplayDocumentReady['vehicleLabels']
  titleSlug: string
  /** Repeindre la scène : les vignettes et le manifeste arrivent après coup. */
  redraw: () => void
}

export interface VehicleSprites {
  /** La vignette teintée d'une famille (couleur résolue), ou `null` tant que l'image ou le manifeste manque. */
  spriteOf: (family: string, color: string) => CanvasImageSource | null
  /**
   * Dimensions natives + échelle manifeste d'UNE famille, ou `null` (chargement pas encore abouti) —
   * la MÊME source que celle du sprite dessiné (règle ≤ 2 copies).
   */
  sizeOf: (family: string) => VehicleSpriteSize | null
}

export function useVehicleSprites({ enabled, labels, titleSlug, redraw }: VehicleSpritesInput): VehicleSprites {
  const rawImagesRef = useVehicleRawImages({ enabled, labels, redraw })
  const manifestRef = useVehicleManifest({ enabled, titleSlug, redraw })

  // LES BORDURES (sprites redessinés du 2026-09-16) : `{famille}_outline.png`, nommée par le
  // manifeste, chargée À LA DEMANDE au premier tracé de la famille (une famille absente du match
  // ne coûte aucune requête). Valeur `null` = chargement lancé ; échec = entrée retirée de la
  // table d'attente et marquée dans `outlineFailedRef` : le sprite se dessine alors SANS bordure
  // plutôt que jamais (la bordure est un habillage, pas une donnée).
  const outlineImagesRef = useRef<Map<string, HTMLImageElement | null>>(new Map())
  const outlineFailedRef = useRef<Set<string>>(new Set())
  const outlineOf = useCallback(
    (family: string, entry: VehicleManifestEntry): HTMLImageElement | null | 'none' => {
      if (!entry.outline || outlineFailedRef.current.has(family)) return 'none'
      const images = outlineImagesRef.current
      if (images.has(family)) return images.get(family) ?? null
      const url = staticAssetURL('vehicle', vehicleManifestAssetId(entry.outline), '.png', titleSlug)
      images.set(family, null)
      // CHARGEMENT PAR `withLoadedImage` (garde-rail loadedImage.guard.test.ts). Il rend une image
      // DEJA chargee SYNCHRONEMENT, et `outlineOf` est appele PENDANT le trace : redessiner a ce
      // moment-la relancerait le trace dans le trace. On ne redessine donc que sur un chargement
      // asynchrone ; le cas synchrone rend l'image directement.
      let pending = true
      withLoadedImage(
        url,
        (im) => {
          images.set(family, im)
          if (pending) return
          redraw()
        },
        () => {
          console.warn('[replay-vehicles] bordure indisponible, sprite sans bordure :', url)
          images.delete(family)
          outlineFailedRef.current.add(family)
          redraw()
        },
      )
      pending = false
      return images.get(family) ?? null
    },
    [titleSlug, redraw],
  )

  // LES VIGNETTES TEINTÉES, cuites HORS ÉCRAN à la demande, par famille × COULEUR RÉSOLUE — la
  // couleur d'équipe d'un véhicule change rarement en cours de lecture (changement de
  // conducteur), le cache grossit donc lentement. `multiply`, jamais `source-in` : les sprites
  // véhicules sont des silhouettes à arêtes sombres (décision de cadrage, cf. `tintedIconCanvas`).
  // LA BORDURE EST CUITE DANS LA MÊME VIGNETTE, SOUS le sprite et APRÈS sa teinte : c'est ce qui
  // la garde blanche (la teinte `multiply` ne voit que le sprite) sans seconde passe au tracé.
  const tintedRef = useRef<Map<string, HTMLCanvasElement>>(new Map())
  const spriteOf = useCallback(
    (family: string, color: string): CanvasImageSource | null => {
      const key = `${family}|${color}`
      const cached = tintedRef.current.get(key)
      if (cached) return cached
      const raw = rawImagesRef.current.get(family)
      const entry = manifestRef.current?.get(family)
      if (!raw || !entry) return null
      const outline = outlineOf(family, entry)
      if (outline === null) return null
      const tinted = tintedIconCanvas(raw, color, { composite: 'multiply' })
      const sprite = outline === 'none' ? tinted : outlinedSpriteCanvas(outline, tinted)
      tintedRef.current.set(key, sprite)
      return sprite
    },
    [outlineOf, rawImagesRef, manifestRef],
  )

  // LA TAILLE EST CELLE DE LA BOÎTE DU VÉHICULE, marge de bordure retirée
  // (`vehicleSpriteManifest.vehicleBodyPx`) : longueur à l'écran, ancres d'armes et rayon
  // d'explosion restent ceux du sprite d'avant la bordure. L'échelle qui en découle est un
  // facteur par pixel source : appliquée à l'image entière, elle pose la bordure AUTOUR du
  // véhicule sans le grossir.
  const sizeOf = useCallback((family: string): VehicleSpriteSize | null => {
    const raw = rawImagesRef.current.get(family)
    const entry = manifestRef.current?.get(family)
    if (!raw || !entry) return null
    return {
      naturalWidthPx: vehicleBodyPx(raw.naturalWidth, entry.pad),
      naturalHeightPx: vehicleBodyPx(raw.naturalHeight, entry.pad),
      mmPerPx: entry.mmPerPx,
    }
  }, [rawImagesRef, manifestRef])

  return { spriteOf, sizeOf }
}

/** Les sprites sources, une image par famille, chargés UNE FOIS : la table se lit dans une référence. */
function useVehicleRawImages({ enabled, labels, redraw }: Pick<VehicleSpritesInput, 'enabled' | 'labels' | 'redraw'>) {
  // LES SPRITES SOURCES, une par FAMILLE employée par `doc.vehicleLabels` — chargés UNE FOIS,
  // jamais reteints ici (la teinture par équipe se fait à la demande, cf. `spriteOf`).
  // LES FAMILLES DE DÉCOR SONT SAUTÉES : le calque ne les dessine pas (cf.
  // `FAMILLES_NON_JOUABLES`), leur image n'a donc aucune raison de traverser le réseau.
  const rawImagesRef = useRef<Map<string, HTMLImageElement>>(new Map())
  useEffect(() => {
    if (!enabled) return
    const map = rawImagesRef.current
    for (const family of Object.keys(labels ?? {})) {
      if (map.has(family) || vehicleIsDecor(family)) continue
      // PAS D'URL = AUCUNE REQUÊTE (lot 1.9.9). Le serveur ne compose plus d'`img` pour une
      // famille qu'il sait sans asset (un élément de carte, aujourd'hui) : la sauter ici est ce
      // qui évite un 404 par match — et le calque a déjà son pictogramme pour elle.
      const url = labels?.[family]?.img
      if (!url) continue
      withLoadedImage(url, (im) => {
        map.set(family, im)
        redraw()
      })
    }
  }, [enabled, labels, redraw])

  return rawImagesRef
}

/** Le manifeste de mise à l'échelle des sprites : une référence, `null` tant qu'il n'est pas lu. */
function useVehicleManifest({ enabled, titleSlug, redraw }: Pick<VehicleSpritesInput, 'enabled' | 'titleSlug' | 'redraw'>) {
  // LE MANIFESTE DE MISE À L'ÉCHELLE : une seule requête par montage (il est petit, et sert
  // toutes les familles quel que soit le match). Une absence (404, réseau) dégrade en « aucune
  // famille dimensionnée » — les véhicules restent NON DESSINÉS (jamais une taille inventée),
  // exactement le même contrat que `replayAudio.ts` sur un son manquant.
  const manifestRef = useRef<Map<string, VehicleManifestEntry> | null>(null)
  useEffect(() => {
    if (!enabled || manifestRef.current) return
    let cancelled = false
    const url = staticAssetURL('vehicle', 'replay/index', '.json', titleSlug)
    if (!url) return
    fetch(url)
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error(`HTTP ${res.status}`))))
      .then((raw: unknown) => {
        if (cancelled) return
        manifestRef.current = parseVehicleManifest(raw)
        redraw()
      })
      .catch((err: unknown) => {
        // Composant démonté avant la réponse : ni journal ni redessin d'une scène qui n'existe plus.
        if (cancelled) return
        console.warn('[replay-vehicles] manifeste de taille indisponible, silence :', url, err)
        manifestRef.current = new Map()
        redraw()
      })
    return () => {
      cancelled = true
    }
  }, [enabled, titleSlug, redraw])

  return manifestRef
}
