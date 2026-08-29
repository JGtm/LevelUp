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
/**
 * SURECHANTILLONNAGE DE L'EXPORT : le facteur par lequel la toile est rendue PENDANT un export,
 * et 1 le reste du temps.
 *
 * # POURQUOI IL EXISTE, ET C'EST UNE MESURE, PAS UNE INTUITION
 *
 * H.264 echantillonne la CHROMINANCE a 4:2:0 : un demi-pixel de couleur sur chaque axe. Sur un
 * rejeu — des traits fins, des petits chiffres, des aplats colores — c'est la que part la
 * nettete, et AUCUN debit ne la rachete. Mesure faite dans le navigateur le 2026-08-28 sur une
 * toile de 502x480, en comparant l'image decodee a sa source :
 *
 * | Rendu | Ecart moyen | Pixels franchement alteres | Poids |
 * |---|---|---|---|
 * | natif 502x480 a 2 Mb/s  | 5,08 | 7,9 % | 49 Ko |
 * | natif 502x480 a 20 Mb/s | 4,96 | 7,9 % | 184 Ko |
 * | double 1004x960         | 0,76 | 0 %   | 96 Ko |
 *
 * Multiplier le debit par DIX ne gagne rien ; doubler la resolution divise l'ecart par sept, et
 * ne coute que le double du poids d'un fichier qui est de toute facon petit.
 *
 * # POURQUOI UN OBJET MUTABLE DE MODULE, ET PAS UNE PROP
 *
 * Le trace lit cette valeur au moment ou il dimensionne le backing store. La faire descendre en
 * prop ou en etat traverserait `ReplayCanvas`, qui est A SON PLAFOND DE TAILLE
 * (`placementFamily.guard.test.ts`) : le cablage couterait plus de lignes que la fonctionnalite.
 * Elle vit donc ici, ou le canvas puise deja sa geometrie, et l'export la repose a 1 dans un
 * `finally` — c'est le seul ecrivain, et il ne la laisse jamais levee.
 */
export const exportRenderScale = { current: 1 }

/** Le facteur applique pendant un export (cf. `exportRenderScale`). */
export const EXPORT_SUPERSAMPLE = 2

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
