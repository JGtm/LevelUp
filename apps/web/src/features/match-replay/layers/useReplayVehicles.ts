/**
 * useReplayVehicles — TOUT LE CÂBLAGE DU CALQUE DES VÉHICULES, en un seul point.
 *
 * MÊME PARTI QUE `useReplayWeaponPads` (patron du dépôt, cf. son en-tête) : ce hook réunit les
 * préoccupations du calque — chargement des sprites sources, teinture hors écran par famille ×
 * couleur, lecture du manifeste de mise à l'échelle, et tracé — et ne rend au canvas que trois
 * lignes utiles : `available`, `paint`, et le PRÉDICAT EMBARQUÉ que `ReplayCanvas` transmet au
 * calque des pions (C7) SANS qu'aucun des deux calques n'ait à connaître la logique de l'autre.
 *
 * LE MANIFESTE (`index.json`, servi par le lot A sous `/static/vehicles-assets/{slug}/replay/`)
 * N'EST PAS DANS LE DOCUMENT : `VehicleLabel` (posé à la requête côté service) ne porte que
 * l'URL du sprite et le fait qu'il se teigne — la règle de TAILLE (décision de cadrage) a besoin
 * en plus du `scale_mm_per_px` par famille, qui n'existe que dans ce fichier statique. Il est
 * donc lu ICI, par un `fetch` brut — PAS DE QUERY KEY NOUVELLE (contrainte du plan) : un asset
 * statique n'est pas une donnée de l'API, exactement le même choix que les sons du rejeu
 * (`replayAudio.ts`, qui charge ses WAV hors TanStack Query).
 *
 * TOUT CE QUI EST ASYNCHRONE EST LU DANS DES RÉFÉRENCES, jamais dans un état React (même règle
 * et même conséquence assumée que `useReplayWeaponPads`) : un chargement qui aboutit sous un
 * pointeur immobile n'a pas d'infobulle à réconcilier ici (ce calque n'en porte aucune en V1),
 * il déclenche seulement `redraw()` pour que la prochaine image le montre.
 */
import { useCallback, useEffect, useMemo, useRef } from 'react'

import { staticAssetURL } from '@/lib/staticAssets'
import { useTitleSlug } from '@/lib/title-routing'

import type { FxInk } from './fxInk'
import type { PlacementView } from './placementShapes'
import { tintedIconCanvas } from './replayDraw'
import { frameToMs } from '../model/replayLogic'
import type { ReplayDocumentReady } from '../model/replayNormalize'
import { buildEmbarkedPredicate, vehicleIsDecor } from '../model/vehiclesLayer'
import { drawVehiclesLayer, type VehicleSpriteSize } from './vehiclesPaint'

/** Une entrée de `index.json` (lot A) : seuls les deux champs utiles ici sont lus. */
interface VehicleManifestEntry {
  famille?: string
  scale_mm_per_px?: number
}

export interface VehiclesInput {
  doc: ReplayDocumentReady
  view: PlacementView
  /** Faux quand le calque est éteint : rien n'est dessiné (le prédicat embarqué, lui, reste actif). */
  enabled: boolean
  /** Calque des NOMS (bouton partagé avec les pions) : les noms empilés le suivent. */
  showNames: boolean
  /** Calque de la VISÉE (le MÊME bouton que les pions) : le cône du conducteur le suit. */
  showAim: boolean
  /** Identité PAR SLOT ET PAR IMAGE (cf. `useSlotIdentity`) : même source que les pions. */
  colorOfSlot: (slot: number, frame: number) => string | null
  /**
   * Couleur d'équipe par XUID — SOURCE PRIORITAIRE de la teinte d'un occupant, pour la raison
   * EXACTE de `nameOfXuid` : le document nomme l'occupant lui-même, le pont slot->joueur est muet
   * pendant l'épisode (cf. `VehicleStyle.colorOfXuid`).
   */
  colorOfXuid: (xuid: string) => string | null
  nameOfSlot: (slot: number, frame: number) => string | null
  /**
   * Nom d'un joueur par XUID — SOURCE PRIORITAIRE de l'étiquette d'un occupant, parce que le
   * document nomme l'occupant lui-même (`VehicleRide.xuid`) alors que le pont slot->joueur
   * dépend, lui, d'une trace de bipède jointe à un xuid (cf. `VehicleStyle.nameOfXuid`).
   */
  nameOfXuid: (xuid: string) => string | null
  /** Encre du « aucun occupant connu » (token sémantique, résolu par l'appelant). */
  neutralInk: string
  /** Encre du contour des noms (cf. `useReplayInks`). */
  labelStroke: string
  /**
   * Teintes de nature des effets (fxInk.ts, MÊME source que les tirs/grenades) : l'explosion de
   * destruction d'un véhicule (schéma 39, en avance de phase — cf. `VehicleStyle.explosionInk`)
   * en tire sa couleur PLASMA vs NORMALE.
   */
  explosionInk: FxInk
  /** Sous « mouvement réduit », l'explosion de destruction ne se joue pas. */
  reducedMotion: boolean
  /** Repeindre la scène : les vignettes et le manifeste arrivent après coup (chargement async). */
  redraw: () => void
}

export interface Vehicles {
  /** Le film porte-t-il des véhicules ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /** Trace le calque à l'image demandée ; ne fait rien quand il est éteint. */
  paint: (ctx: CanvasRenderingContext2D, frame: number, k: number) => void
  /**
   * PRÉDICAT « EMBARQUÉ À T » (C7) : à consommer par `replayMarkers.drawTracksLayer`
   * (`MarkerStyle.embarkedAtSlot`) pour supprimer le pion et le nom d'un occupant SANS dupliquer
   * la logique d'occupation.
   *
   * IL SUIT LE TOGGLE DU CALQUE (revue adversariale 2026-09-02, point 7) : calque ÉTEINT, il rend
   * `false` partout et les pions restent dessinés. Supprimer un occupant sans dessiner son
   * véhicule ferait disparaître des joueurs sans aucun réglage pour les récupérer — c'est ce
   * qu'un prédicat « toujours actif » produirait, et l'implémentation ne l'a jamais fait.
   */
  isEmbarkedAt: (slot: number, frame: number) => boolean
  /**
   * Dimensions natives + échelle manifeste d'UNE famille, ou `null` (chargement pas encore
   * abouti). EXPOSÉ depuis le 2026-09-03 pour `drawShotsLayer` (origine des tirs en véhicule,
   * `vehicleWeaponMounts.vehicleShotPlacement`) : LA MÊME source déjà chargée ici pour dessiner
   * le sprite, jamais un second chargement du manifeste ou des images (règle ≤ 2 copies).
   */
  sizeOf: (family: string) => VehicleSpriteSize | null
}

export function useReplayVehicles({
  doc,
  view,
  enabled,
  showNames,
  showAim,
  colorOfSlot,
  colorOfXuid,
  nameOfSlot,
  nameOfXuid,
  neutralInk,
  labelStroke,
  explosionInk,
  reducedMotion,
  redraw,
}: VehiclesInput): Vehicles {
  const titleSlug = useTitleSlug()
  const tracks = doc.vehicles
  const labels = doc.vehicleLabels
  // Durée RÉELLE d'une frame : l'explosion de destruction a une timeline en TEMPS, pas en
  // frames (même besoin que `RestWindow.frameMs` des grenades, `frameToMs` porte déjà le repli
  // des artefacts sans échelle temporelle). Ne dépend que du document, jamais de l'image.
  const frameMs = useMemo(() => frameToMs(1, doc), [doc])

  // LE PRÉDICAT EMBARQUÉ SUIT LE TOGGLE DU CALQUE (revue adversariale 2026-09-02, point 7) :
  // calque ÉTEINT, on rend les pions — supprimer un occupant sans dessiner son véhicule ferait
  // disparaître des joueurs sans aucun réglage pour les récupérer.
  const isEmbarkedAt = useMemo(() => {
    if (!enabled) return () => false
    return buildEmbarkedPredicate(tracks)
  }, [enabled, tracks])

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
      const url = labels?.[family]?.img
      if (!url) continue
      const im = new Image()
      im.onload = () => {
        map.set(family, im)
        redraw()
      }
      im.src = url
    }
  }, [enabled, labels, redraw])

  // LE MANIFESTE DE MISE À L'ÉCHELLE : une seule requête par montage (il est petit, et sert
  // toutes les familles quel que soit le match). Une absence (404, réseau) dégrade en « aucune
  // famille dimensionnée » — les véhicules restent NON DESSINÉS (jamais une taille inventée),
  // exactement le même contrat que `replayAudio.ts` sur un son manquant.
  const manifestRef = useRef<Map<string, number> | null>(null)
  useEffect(() => {
    if (!enabled || manifestRef.current) return
    let cancelled = false
    const url = staticAssetURL('vehicle', 'replay/index', '.json', titleSlug)
    if (!url) return
    fetch(url)
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error(`HTTP ${res.status}`))))
      .then((raw: unknown) => {
        if (cancelled) return
        const map = new Map<string, number>()
        if (Array.isArray(raw)) {
          for (const entry of raw as VehicleManifestEntry[]) {
            if (entry.famille && typeof entry.scale_mm_per_px === 'number') {
              map.set(entry.famille, entry.scale_mm_per_px)
            }
          }
        }
        manifestRef.current = map
        redraw()
      })
      .catch((err: unknown) => {
        console.warn('[replay-vehicles] manifeste de taille indisponible, silence :', url, err)
        manifestRef.current = new Map()
        redraw()
      })
    return () => {
      cancelled = true
    }
  }, [enabled, titleSlug, redraw])

  // LES VIGNETTES TEINTÉES, cuites HORS ÉCRAN à la demande, par famille × COULEUR RÉSOLUE — la
  // couleur d'équipe d'un véhicule change rarement en cours de lecture (changement de
  // conducteur), le cache grossit donc lentement. `multiply`, jamais `source-in` : les sprites
  // véhicules sont des silhouettes à traits noirs (décision de cadrage, cf. `tintedIconCanvas`).
  const tintedRef = useRef<Map<string, HTMLCanvasElement>>(new Map())
  const spriteOf = useCallback((family: string, color: string): CanvasImageSource | null => {
    const key = `${family}|${color}`
    const cached = tintedRef.current.get(key)
    if (cached) return cached
    const raw = rawImagesRef.current.get(family)
    if (!raw) return null
    const tinted = tintedIconCanvas(raw, color, { composite: 'multiply' })
    tintedRef.current.set(key, tinted)
    return tinted
  }, [])

  const sizeOf = useCallback((family: string): VehicleSpriteSize | null => {
    const raw = rawImagesRef.current.get(family)
    const mmPerPx = manifestRef.current?.get(family)
    if (!raw || mmPerPx === undefined) return null
    return { naturalWidthPx: raw.naturalWidth, naturalHeightPx: raw.naturalHeight, mmPerPx }
  }, [])

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number, k: number) => {
      if (!enabled || tracks.length === 0) return
      drawVehiclesLayer(
        ctx,
        tracks,
        view,
        { frame, k, frameMs },
        {
          neutralInk, labelStroke, showNames, showAim, spriteOf, sizeOf, colorOfSlot, colorOfXuid,
          nameOfSlot, nameOfXuid, explosionInk, reducedMotion,
        },
      )
    },
    [
      enabled, tracks, view, neutralInk, labelStroke, showNames, showAim, spriteOf, sizeOf,
      colorOfSlot, colorOfXuid, nameOfSlot, nameOfXuid, frameMs, explosionInk, reducedMotion,
    ],
  )

  // « DISPONIBLE » = AU MOINS UN VÉHICULE QUE LE CALQUE DESSINERAIT. Un film qui ne porte que du
  // décor (Falcon & consorts, cf. `FAMILLES_NON_JOUABLES`) n'a pas de calque à commander : la
  // bascule ne s'affiche pas, plutôt que d'allumer un calque resté vide.
  const available = useMemo(() => tracks.some((t) => !vehicleIsDecor(t.family)), [tracks])

  return { available, paint, isEmbarkedAt, sizeOf }
}
