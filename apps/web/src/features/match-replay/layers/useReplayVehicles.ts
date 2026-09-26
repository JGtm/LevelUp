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
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type PointerEvent,
  type RefObject,
} from 'react'

import { staticAssetURL } from '@/lib/staticAssets'
import { useTitleSlug } from '@/lib/title-routing'

import type { FxInk } from './fxInk'
import { withLoadedImage } from './loadedImage'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { PlacementView } from './placementShapes'
import { outlinedSpriteCanvas, tintedIconCanvas } from './replayDraw'
import { frameToMs, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { Respawn } from '../model/respawnCountdown'
import {
  vehicleCycleLives,
  vehicleCycleReadingAt,
  type VehicleCycle,
} from '../model/vehicleCycleTime'
import {
  parseVehicleManifest,
  vehicleBodyPx,
  vehicleManifestAssetId,
  type VehicleManifestEntry,
} from '../model/vehicleSpriteManifest'
import { buildEmbarkedPredicate, vehicleIsDecor, vehicleIsHidden } from '../model/vehiclesLayer'
import { drawVehicleCyclesLayer, vehicleCycleIndexAt } from './vehicleCyclesLayer'
import { drawVehiclesLayer, type VehicleSpriteSize } from './vehiclesPaint'

export interface VehiclesInput {
  doc: ReplayDocumentReady
  view: PlacementView
  /**
   * L'image courante, telle que la boucle de lecture la tient — LUE DANS UNE RÉFÉRENCE, jamais
   * dans un état React (même règle et même conséquence assumée que `useReplayWeaponPads`) : si
   * un emplacement change d'état SOUS un pointeur immobile, son infobulle attend le prochain
   * mouvement. À l'arrêt — le cas où l'on inspecte — la lecture est exacte.
   */
  frameRef: RefObject<number>
  /** Faux quand le calque est éteint : rien n'est dessiné (le prédicat embarqué, lui, reste actif). */
  enabled: boolean
  /**
   * LA LANGUE DU LECTEUR — elle ne sert QU'au libellé d'une famille qui n'est pas un véhicule
   * (lot 1.9.9, cf. `VehicleStyle.labelOfFamily`). Le calque lui-même ne connaît aucune langue :
   * le texte vient du DOCUMENT, ce hook ne fait que choisir la colonne.
   */
  locale: ReplayLocale
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
  /**
   * BORNAGE HORS CADRE (plan escouade hors cadre, lot 4.4, 2026-09-10) : texte de la flèche
   * d'un SEUL occupant hors fenêtre — nom ET distance (cf. `VehicleStyle.offscreenLabelOf`).
   */
  offscreenLabelOf: (name: string, meters: number) => string
  /** MÊME bornage, plusieurs occupants sans conducteur nommé — « N joueurs · distance ». */
  offscreenGroupLabelOf: (count: number, meters: number) => string
  /** Encre du « aucun occupant connu » (token sémantique, résolu par l'appelant). */
  neutralInk: string
  /** Encre du contour des noms (cf. `useReplayInks`). */
  labelStroke: string
  /**
   * Les DEUX encres du MARQUAGE — ce qui est rempli, ce qui est cerné —, celles que les socles
   * d'arme emploient déjà pour leur compte à rebours (`markInk` du canvas). Le marqueur de
   * réapparition d'un emplacement de véhicule écrit le même compte, il doit donc porter la même
   * encre : un chiffre de carte qui changerait d'encre selon le calque se lirait comme deux
   * natures d'information.
   */
  markInk: { fill: string; outline: string }
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

/**
 * Ce qui est survolé sur un EMPLACEMENT DE NAISSANCE (schéma 63) : l'emplacement, son nom, son
 * occupation LUE À CET INSTANT, son compte à rebours, et où poser l'infobulle.
 *
 * `occupied` ET `respawn` NE SONT PAS REDONDANTS : un emplacement libre peut n'avoir aucun compte
 * (aucune fin datée d'où partir, aucune naissance suivante dans le film), et l'infobulle doit
 * alors se taire plutôt que d'annoncer une présence. Les deux silences ne disent pas la même
 * chose.
 */
export interface VehicleCycleHover {
  cycle: VehicleCycle
  at: XY
  /** Famille dominante nommée par le document, ou le titre générique de l'emplacement. */
  name: string
  occupied: boolean
  respawn: Respawn | null
}

export interface Vehicles {
  /** Le film porte-t-il des véhicules ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /**
   * LE NOM DU CALQUE, porté par le calque et non par la table de liaison du canvas
   * (2026-09-06, revue R1 constat C2) : c'est ce qui rend impossible de peindre ce geste
   * sous l'identité d'un autre calque.
   */
  id: 'vehicules'
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
  /** L'emplacement de naissance survolé, ou `null` (cf. `ReplayVehicleCycleTip`). */
  cycleHover: VehicleCycleHover | null
  onPointerMove: (event: PointerEvent<HTMLCanvasElement>) => void
  onPointerLeave: () => void
}

export function useReplayVehicles({
  doc,
  view,
  frameRef,
  enabled,
  locale,
  showNames,
  showAim,
  colorOfSlot,
  colorOfXuid,
  nameOfSlot,
  nameOfXuid,
  offscreenLabelOf,
  offscreenGroupLabelOf,
  neutralInk,
  labelStroke,
  markInk,
  explosionInk,
  reducedMotion,
  redraw,
}: VehiclesInput): Vehicles {
  const titleSlug = useTitleSlug()
  const tracks = doc.vehicles
  const labels = doc.vehicleLabels
  const cycles = doc.vehicleCycles
  // LES LIBELLÉS DE LA LANGUE DU LECTEUR, pour le SEUL marqueur de ce calque qui écrive du texte
  // de l'application : le compte à rebours d'un emplacement de naissance et le titre générique de
  // son infobulle. Le calque des véhicules, lui, ne connaît toujours aucune langue — les noms
  // qu'il affiche viennent du DOCUMENT (`labelOfFamily`).
  const t = REPLAY_TEXT[locale]
  // Durée RÉELLE d'une frame : l'explosion de destruction a une timeline en TEMPS, pas en
  // frames (même besoin que `RestWindow.frameMs` des grenades, `frameToMs` porte déjà le repli
  // des artefacts sans échelle temporelle). Ne dépend que du document, jamais de l'image.
  const frameMs = useMemo(() => frameToMs(1, doc), [doc])

  // LA NATURE D'UNE FAMILLE, telle que le DOCUMENT la publie (lot 1.9.9) : `undefined` pour un
  // véhicule, le cas général. Résolue ici UNE fois pour les deux consommateurs — le prédicat
  // embarqué (un élément de carte ne porte personne) et le tracé (il a son pictogramme dédié).
  // Le calque ne décide jamais seul qu'une famille n'est pas un véhicule : c'est le serveur qui
  // le dit, depuis le manifeste du titre.
  const kindOf = useCallback(
    (family: string): string | undefined => labels?.[family]?.kind,
    [labels],
  )

  // LE LIBELLÉ D'UNE FAMILLE dans la langue du lecteur, tel que le DOCUMENT le publie (lot
  // 1.9.9). Vide pour dix-huit familles sur dix-neuf : un nom de véhicule est un nom propre du
  // jeu et la clé EST le nom. Aucune chaîne n'est écrite ici — elles viennent du manifeste du
  // titre, comme tous les libellés du rejeu.
  const labelOfFamily = useCallback(
    (family: string): string | null => {
      const lbl = labels?.[family]
      return (locale === 'fr' ? lbl?.fr : lbl?.en) ?? null
    },
    [labels, locale],
  )

  // LE PRÉDICAT EMBARQUÉ SUIT LE TOGGLE DU CALQUE (revue adversariale 2026-09-02, point 7) :
  // calque ÉTEINT, on rend les pions — supprimer un occupant sans dessiner son véhicule ferait
  // disparaître des joueurs sans aucun réglage pour les récupérer.
  const isEmbarkedAt = useMemo(() => {
    if (!enabled) return () => false
    return buildEmbarkedPredicate(tracks, kindOf)
  }, [enabled, tracks, kindOf])

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
        console.warn('[replay-vehicles] manifeste de taille indisponible, silence :', url, err)
        manifestRef.current = new Map()
        redraw()
      })
    return () => {
      cancelled = true
    }
  }, [enabled, titleSlug, redraw])

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
    [outlineOf],
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
  }, [])

  // L'APPARIEMENT EMPLACEMENT -> VIES, UNE SEULE FOIS PAR DOCUMENT. C'est un balayage de toutes
  // les vies pour chaque emplacement (la maille de l'amas, cf. `vehicleCycleTime`) : le refaire
  // soixante fois par seconde serait un doublon de travail, et le refaire au survol un doublon de
  // source. Le tableau est PARALLÈLE à `cycles` — c'est pourquoi le test de survol rend un RANG.
  const cycleLives = useMemo(
    () => cycles.map((cycle) => vehicleCycleLives(cycle, tracks)),
    [cycles, tracks],
  )
  const livesOf = useCallback((index: number) => cycleLives[index] ?? [], [cycleLives])

  // LE NOM D'UN EMPLACEMENT : la famille DOMINANTE telle que le document la nomme, sa clé à
  // défaut (un nom de véhicule est un nom propre du jeu), et le titre générique quand
  // l'emplacement ne porte aucune famille — jamais le nom d'un voisin.
  const nameOfCycle = useCallback(
    (cycle: VehicleCycle): string =>
      cycle.family ? (labelOfFamily(cycle.family) ?? cycle.family) : t.vehicleCycleTitle,
    [labelOfFamily, t.vehicleCycleTitle],
  )

  const [cycleHover, setCycleHover] = useState<VehicleCycleHover | null>(null)

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number, k: number) => {
      if (!enabled || tracks.length === 0) return
      drawVehiclesLayer(
        ctx,
        tracks,
        view,
        { frame, k, frameMs },
        {
          neutralInk, labelStroke, showNames, showAim, spriteOf, sizeOf, kindOf, labelOfFamily,
          colorOfSlot, colorOfXuid, nameOfSlot, nameOfXuid, offscreenLabelOf, offscreenGroupLabelOf,
          explosionInk, reducedMotion,
        },
      )
      // LES EMPLACEMENTS DE NAISSANCE APRÈS LES VÉHICULES, et dans le même geste : ils suivent la
      // MÊME bascule (un marqueur de réapparition de véhicule sans son calque de véhicules serait
      // un calque sans interrupteur) et se posent au-dessus du fond, sous rien d'autre — un
      // emplacement occupé ne dessine rien, il n'y a donc jamais de marque sous un sprite.
      drawVehicleCyclesLayer(
        ctx,
        cycles,
        livesOf,
        view,
        { frame, frameMs, k },
        {
          ink: neutralInk,
          fill: markInk.fill,
          outline: markInk.outline,
          countdownLabel: t.padCountdownFmt,
        },
      )
    },
    [
      enabled, tracks, view, neutralInk, labelStroke, showNames, showAim, spriteOf, sizeOf, kindOf,
      labelOfFamily, colorOfSlot, colorOfXuid, nameOfSlot, nameOfXuid, offscreenLabelOf, offscreenGroupLabelOf,
      frameMs, explosionInk, reducedMotion,
      cycles, livesOf, markInk.fill, markInk.outline, t.padCountdownFmt,
    ],
  )

  /**
   * LE SURVOL D'UN EMPLACEMENT DE NAISSANCE — le seul survol de ce calque (les véhicules
   * eux-mêmes n'en portent aucun).
   *
   * MÊME AMORCE QUE CELLE DES SOCLES ET DES POSES : le rapport pixels CSS -> pixels du contexte
   * se CALCULE plutôt qu'il ne se suppose (la mise en page peut remettre le canevas à l'échelle).
   */
  const onPointerMove = useCallback(
    (event: PointerEvent<HTMLCanvasElement>) => {
      if (!enabled || cycles.length === 0 || view.width === 0) {
        setCycleHover((prev) => (prev === null ? prev : null))
        return
      }
      const rect = event.currentTarget.getBoundingClientRect()
      const kx = rect.width > 0 ? view.width / rect.width : 1
      const ky = rect.height > 0 ? view.height / rect.height : 1
      const at = { x: (event.clientX - rect.left) * kx, y: (event.clientY - rect.top) * ky }
      const i = vehicleCycleIndexAt(cycles, view, window.devicePixelRatio || 1, at)
      setCycleHover((prev) => {
        if (i < 0) return prev === null ? prev : null
        const cycle = cycles[i]
        const lu = vehicleCycleReadingAt(cycle, cycleLives[i] ?? [], frameRef.current, frameMs)
        const next: VehicleCycleHover = {
          cycle,
          at,
          name: nameOfCycle(cycle),
          occupied: lu.occupant !== null,
          respawn: lu.respawn,
        }
        // LE COMPTE SE COMPARE CHAMP À CHAMP, jamais par identité : la lecture construit un objet
        // neuf à chaque appel, et l'infobulle se recréerait à chaque mouvement de pointeur.
        if (
          prev &&
          prev.cycle === next.cycle &&
          prev.occupied === next.occupied &&
          prev.respawn?.seconds === next.respawn?.seconds &&
          prev.respawn?.measured === next.respawn?.measured &&
          prev.at.x === at.x &&
          prev.at.y === at.y
        ) {
          return prev
        }
        return next
      })
    },
    [enabled, cycles, view, cycleLives, frameRef, frameMs, nameOfCycle],
  )

  const onPointerLeave = useCallback(() => {
    setCycleHover((prev) => (prev === null ? prev : null))
  }, [])

  // « DISPONIBLE » = AU MOINS UN VÉHICULE QUE LE CALQUE DESSINERAIT. Un film qui ne porte que du
  // décor (Pelican & consorts, cf. `FAMILLES_NON_JOUABLES`) n'a pas de calque à commander : la
  // bascule ne s'affiche pas, plutôt que d'allumer un calque resté vide. Le DÉCOR DE CARTE
  // (posé par la carte hors de sa zone jouable, déclaré par le serveur — `vehicleIsScenery`,
  // lots L1.3 et M7) compte de même.
  const available = useMemo(() => tracks.some((t) => !vehicleIsHidden(t)), [tracks])

  // UNE SEULE LIGNE, ET C'EST UN RATCHET : `sceneBinding.guard.test.ts` exige que l'id du calque
  // soit rendu ICI, en tete de l'objet — c'est ce qui rend impossible de peindre ce geste sous
  // l'identite d'un autre calque.
  return { id: 'vehicules', available, paint, isEmbarkedAt, sizeOf, cycleHover, onPointerMove, onPointerLeave }
}
