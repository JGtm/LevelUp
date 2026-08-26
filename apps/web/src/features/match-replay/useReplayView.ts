/**
 * useReplayView — LE CADRAGE du rejeu 2D : ce que la carte montre, et dans quel repère.
 *
 * EXTRAIT DE `ReplayCanvas.tsx` LE 2026-08-26 (neuvième extraction imposée par le cliquet de
 * taille, `placementFamily.guard.test.ts`). Le canvas était PILE à son plafond de 742 lignes
 * et le lot y branche la CAPTURE (image PNG, enregistrement vidéo) : le cliquet impose alors
 * d'extraire AVANT d'ajouter, jamais de relever le nombre — c'est sa raison d'être, et il
 * descend d'autant.
 *
 * POURQUOI CES SIX VALEURS ENSEMBLE, ET PAS D'AUTRES. Elles forment UNE chaîne de décision où
 * chaque maillon dépend du précédent : le FOND retenu (ou écarté) décide des BORNES de la
 * scène, les bornes décident de la LARGEUR de dessin à hauteur fixée, et les deux ensemble
 * donnent la PROJECTION que le dessin et le survol doivent partager — un pointeur qui viserait
 * un autre cadre que celui peint ne toucherait rien. La trame d'altitudes et l'amplitude
 * verticale appartiennent au même repère : elles ne se cuisent que pour lui.
 *
 * CE QUI N'EST PAS ICI dit la frontière aussi bien que ce qui y est : ni le TEMPS de la lecture
 * (`useReplayPlayback`, `useReplayTiming`), ni les ENCRES (`useReplayInks`), ni un seul trait de
 * dessin. Ce hook ne peint rien ; il dit seulement où les choses tombent.
 *
 * LES NOMS SORTENT INCHANGÉS, et c'est délibéré : l'appelant les déstructure tels quels, donc
 * l'extraction n'a pas touché une seule ligne du dessin qui les lit.
 */
import { useMemo } from 'react'

import type { ReplayBounds, ReplayMapBackgroundCalibration } from '@/lib/api/types'

import { coversPlayedArea } from './mapBackground'
import { buildFloorGrid, type FloorGrid } from './mapFloor'
import { fitWidth, sceneBounds } from './replayLogic'
import type { CanvasView } from './replayDraw'
import type { ReplayDocumentReady } from './replayNormalize'

// La HAUTEUR de dessin est fixe : c'est la largeur qui suit le ratio de la scène (cf.
// `renderWidth`), sans quoi une carte étirée laisserait des marges latérales vides. Le PAD est
// la marge intérieure du cadrage, en px. Les TOKENS des encres du canvas vivent avec elles,
// dans useReplayInks ; les DURÉES et leur conversion en images, dans useReplayTiming.
export const CANVAS_HEIGHT = 480
export const CANVAS_PAD = 24

/**
 * Le FOND DE CARTE : l'image cuite de la carte, et le calage qui la pose dans le repère
 * monde du rejeu. Les deux voyagent ensemble — une image sans calage ne se superpose à rien,
 * et l'appelant ne doit jamais pouvoir en fournir une seule.
 */
export interface ReplayMapBackgroundLayer {
  calibration: ReplayMapBackgroundCalibration
  image: HTMLImageElement
}

export interface ReplayViewOptions {
  doc: ReplayDocumentReady
  /** Fond de carte figé, tel que la page l'a chargé. Absent = la carte n'en a pas. */
  background?: ReplayMapBackgroundLayer | null
  /** Largeur mesurée du conteneur, en px CSS. 0 = le canvas n'est pas encore mesuré. */
  width: number
}

/** Le cadrage complet, dans l'ordre où il se décide. */
export interface ReplayView {
  /** Le fond RETENU (`null` = écarté ou absent) : c'est lui qui décide du reste. */
  mapImage: ReplayMapBackgroundLayer | null
  bounds: ReplayBounds
  /** Largeur de dessin en px CSS ; 0 tant que le conteneur n'est pas mesuré. */
  renderWidth: number
  /** Amplitude verticale de la scène : l'indication d'étage s'y rapporte. */
  zRange: { min: number; max: number }
  /** LA projection, partagée par le dessin et par le survol. */
  canvasView: CanvasView
  /** Trame d'altitudes du sol reconstruit — `null` quand un fond figé la remplace. */
  floorGrid: FloorGrid | null
}

export function useReplayView({ doc, background, width }: ReplayViewOptions): ReplayView {
  // LE FOND DE CARTE PREND LA PLACE DU SOL RECONSTRUIT, il ne s'y ajoute pas : l'image
  // porte la carte telle que le jeu la dessine, la trame d'altitudes n'en est que
  // l'approximation. Les superposer ne ferait que voiler la meilleure des deux.
  //
  // Il est ÉCARTÉ quand il ne recouvre pas la zone jouée : un fond qui ne contient pas le
  // terrain n'est pas un défaut d'affichage, c'est le signe que les deux repères ne sont
  // pas le même — mieux vaut alors le sol reconstruit qu'une carte posée à côté des joueurs.
  const mapImage = useMemo(() => {
    if (!background) return null
    return coversPlayedArea(background.calibration, doc.bounds) ? background : null
  }, [background, doc.bounds])
  // Le cadrage se décide APRÈS le fond : une image posée écarte les props du cadre (sceneBounds).
  const bounds = useMemo(() => sceneBounds(doc, mapImage !== null), [doc, mapImage])

  // La trame d'altitudes ne dépend QUE du document : construite une fois, pas à chaque resize.
  const floorGrid = useMemo(
    () => (!mapImage && doc.structure?.length ? buildFloorGrid(doc.structure, doc.bounds) : null),
    [doc.structure, doc.bounds, mapImage],
  )
  // Largeur de dessin = ratio de la scène à hauteur fixée (évite les marges latérales).
  const renderWidth = useMemo(
    () => (width === 0 ? 0 : Math.floor(fitWidth(bounds, width, CANVAS_HEIGHT, CANVAS_PAD))),
    [bounds, width],
  )
  const zRange = useMemo(
    () => ({ min: doc.bounds.minZ ?? 0, max: doc.bounds.maxZ ?? 0 }),
    [doc.bounds.minZ, doc.bounds.maxZ],
  )
  // LE CADRAGE, une fois : le dessin ET le survol doivent lire la MÊME projection — un
  // pointeur qui viserait un autre cadre que celui peint ne toucherait rien.
  const canvasView = useMemo(
    () => ({ bounds, width: renderWidth, height: CANVAS_HEIGHT, pad: CANVAS_PAD }),
    [bounds, renderWidth],
  )

  return { mapImage, bounds, renderWidth, zRange, canvasView, floorGrid }
}
