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
 * (`replayAudio.ts`, qui charge ses sons hors TanStack Query).
 *
 * TOUT CE QUI EST ASYNCHRONE EST LU DANS DES RÉFÉRENCES, jamais dans un état React (même règle
 * et même conséquence assumée que `useReplayWeaponPads`) : un chargement qui aboutit sous un
 * pointeur immobile n'a pas d'infobulle à réconcilier ici (ce calque n'en porte aucune en V1),
 * il déclenche seulement `redraw()` pour que la prochaine image le montre.
 */
import { useCallback, useMemo, type PointerEvent, type RefObject } from 'react'

import { pickText } from '@/lib/i18n/pickText'
import { useTitleSlug } from '@/lib/title-routing'

import type { FxInk } from './fxInk'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { PlacementView } from './placementShapes'
import { durationFrameMs, frameToMs, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { Respawn } from '../model/respawnCountdown'
import type { VehicleCycle } from '../model/vehicleCycleTime'
import { buildEmbarkedPredicate, vehicleIsHidden } from '../model/vehiclesLayer'
import { drawVehicleCyclesLayer } from './vehicleCyclesLayer'
import { useVehicleCycles } from './useVehicleCycles'
import { useVehicleSprites } from './useVehicleSprites'
import { drawVehiclesLayer, type VehicleSpriteSize, type VehicleStyle } from './vehiclesPaint'

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
  /** Encre d'un élément de carte non jouable (gris `zone-neutral`, cf. `VehicleStyle.mapElementInk`). */
  mapElementInk: string
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
  /**
   * SEUL ET MASQUER (panneau de joueur) : retire du TRACÉ les véhicules de ceux qu'on masque (ou qui ne sont
   * pas celui qu'on isole). Le prédicat embarqué et les emplacements de naissance n'en dépendent pas.
   */
  shown?: VehicleStyle['shown']
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
  mapElementInk,
  labelStroke,
  markInk,
  explosionInk,
  reducedMotion,
  shown,
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

  const { kindOf, labelOfFamily, isEmbarkedAt } = useVehicleFamilies({ labels, locale, enabled, tracks })

  // LES SPRITES (chargement, manifeste, bordures, teinture) : un hook à part, `useVehicleSprites`.
  const { spriteOf, sizeOf } = useVehicleSprites({ enabled, labels, titleSlug, redraw })

  const { livesOf, cycleHover, onPointerMove, onPointerLeave } = useVehicleCycles({
    cycles,
    tracks,
    view,
    frameRef,
    frameMs,
    enabled,
    labelOfFamily,
    cycleTitle: t.vehicleCycleTitle,
  })

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number, k: number) => {
      if (!enabled || tracks.length === 0) return
      drawVehiclesLayer(
        ctx,
        tracks,
        view,
        { frame, k, frameMs, durationFrameMs: durationFrameMs(doc) },
        {
          neutralInk, mapElementInk, labelStroke, showNames, showAim, spriteOf, sizeOf, kindOf,
          labelOfFamily, colorOfSlot, colorOfXuid, nameOfSlot, nameOfXuid, offscreenLabelOf, offscreenGroupLabelOf,
          explosionInk, reducedMotion, shown,
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
      enabled, tracks, view, neutralInk, mapElementInk, labelStroke, showNames, showAim, spriteOf, sizeOf, kindOf,
      labelOfFamily, colorOfSlot, colorOfXuid, nameOfSlot, nameOfXuid, offscreenLabelOf, offscreenGroupLabelOf,
      frameMs, explosionInk, reducedMotion, shown,
      cycles, livesOf, markInk.fill, markInk.outline, t.padCountdownFmt,
    ],
  )

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

interface VehicleFamiliesInput {
  labels: ReplayDocumentReady['vehicleLabels']
  locale: ReplayLocale
  enabled: boolean
  tracks: ReplayDocumentReady['vehicles']
}

/** Ce que le document dit d'une FAMILLE de véhicule : sa nature, son libellé, et le prédicat « embarqué à T ». */
function useVehicleFamilies({ labels, locale, enabled, tracks }: VehicleFamiliesInput) {
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
      return pickText(lbl, locale) || null
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

  return { kindOf, labelOfFamily, isEmbarkedAt }
}
