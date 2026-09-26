/**
 * useReplayView — LE CADRAGE du rejeu 2D : ce que la carte montre, et dans quel repère.
 *
 * EXTRAIT DE `ReplayCanvas.tsx` LE 2026-08-26 (neuvième extraction imposée par le seuil de
 * taille, `max-lines` eslint, R5). Le canvas était PILE à son plafond de 742 lignes
 * et le lot y branche la CAPTURE (image PNG, enregistrement vidéo) : le seuil impose alors
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

import { coversPlayedArea } from '../layers/mapBackground'
import { frameBounds, sceneBounds, usefulHeight, visibleBounds } from '../../../lib/replay/replayLogic'
import { type CanvasView } from '../model/replayView'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { lockedZoom, useReplayZoom, type ReplayZoom } from './useReplayZoom'
import { useExportLayout } from '../export/exportLayoutStore'

// LA HAUTEUR DE DESSIN N'EST PLUS FIXE (2026-09-02) : elle est le moindre de ce que l'écran
// laisse et de ce que la carte peut utiliser. La largeur, elle, suit toujours le ratio de la
// scène à cette hauteur (cf. `renderWidth`), sans quoi une carte étirée laisserait des marges
// latérales vides. Le PAD est la marge intérieure du cadrage, en px. Les TOKENS des encres du
// canvas vivent avec elles, dans useReplayInks ; les DURÉES, dans useReplayTiming.
//
// PENDANT UN EXPORT (2026-09-16), l'écran ne décide plus rien : la toile prend le cadre logique
// du format choisi (`export/exportFormats.ts`), et la densité de rendu est celle du format
// (`canvasPixelRatio`). Voir `useReplayView` plus bas.
/**
 * LES BORNES DE LA HAUTEUR DU TERRAIN (2026-09-02, cf. `useReplayViewport`).
 *
 * Le PLANCHER est le seuil sous lequel une carte vue du dessus cesse d'etre lisible. En dessous,
 * on laisse la page defiler : le defilement a alors quelque chose a montrer.
 *
 * Le PLAFOND DUR n'est PAS le plafond utile — il ne borne que la memoire des quatre calques
 * statiques cuits hors ecran, dont la taille croit avec la surface. Le plafond qui compte est
 * calcule PAR CARTE par `usefulHeight` : au-dela, un pixel de hauteur de plus n'agrandit plus la
 * carte, il ajoute une bande vide (cf. `replayLogic.usefulHeight`).
 *
 * Le DEFAUT est la hauteur servie avant toute mesure, le temps d'un rendu.
 */
export const CANVAS_HEIGHT_MIN = 360
export const CANVAS_HEIGHT_CEILING = 720
export const CANVAS_HEIGHT_DEFAULT = 480
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
  /**
   * La hauteur que L'ÉCRAN laisse au terrain, mesurée par `useReplayViewport` — une OFFRE, pas
   * une décision. Ce hook en retient le moindre entre elle et ce que la carte peut vraiment
   * utiliser : lui seul connaît les bornes de la scène, donc lui seul peut dire à partir de
   * quelle hauteur on n'ajoute plus que du vide.
   */
  freeHeight: number
}

/** Le cadrage complet, dans l'ordre où il se décide. */
export interface ReplayView {
  /** Le fond RETENU (`null` = écarté ou absent) : c'est lui qui décide du reste. */
  mapImage: ReplayMapBackgroundLayer | null
  bounds: ReplayBounds
  /** Largeur de dessin en px CSS ; 0 tant que le conteneur n'est pas mesuré. */
  renderWidth: number
  /** Hauteur de dessin RETENUE, en px CSS : `min(offre de l'écran, hauteur utile de la carte)`. */
  renderHeight: number
  /**
   * LA BOÎTE DE LA TOILE À L'ÉCRAN, en px CSS. Hors export, elle vaut la taille de dessin.
   * PENDANT UN EXPORT elle garde la taille d'avant : la toile se dessine au cadre 16:9 du
   * format, et c'est `object-fit: contain` qui l'inscrit dans cette boîte — la page ne saute
   * pas, l'image ne se déforme pas, et une bande du fond de la carte borde le cadre si les
   * proportions de l'écran ne sont pas 16:9.
   */
  screen: { width: number; height: number }
  /** Amplitude verticale de la scène : l'indication d'étage s'y rapporte. */
  zRange: { min: number; max: number }
  /** LA projection, partagée par le dessin et par le survol. */
  canvasView: CanvasView
  /**
   * L'ÉTAT DE NAVIGATION — palier de grossissement et centre. Il vit ICI et non chez l'appelant
   * parce qu'il a besoin des BORNES DE LA SCÈNE pour se borner, et que ces bornes se décident
   * dans ce hook (elles dépendent du fond de carte retenu). Le sortir obligerait l'appelant à
   * les recalculer, donc à tenir une seconde définition de « la scène ».
   */
  zoom: ReplayZoom
}

export function useReplayView({
  doc,
  background,
  width,
  freeHeight,
}: ReplayViewOptions): ReplayView {
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

  // LA HAUTEUR RETENUE — le moindre de ce que l'écran offre et de ce que la carte peut utiliser.
  // Les deux bornes sont indispensables et ne disent pas la même chose : l'offre empêche la page
  // de déborder, la hauteur utile empêche d'ajouter des bandes vides au-dessus et au-dessous
  // d'une carte que la largeur limite déjà. Sur une carte allongée dans une colonne étroite,
  // c'est la seconde qui mord ; sur une carte carrée dans un grand écran, c'est la première.
  const screenHeight = useMemo(
    () =>
      width === 0
        ? freeHeight
        : Math.floor(Math.min(freeHeight, usefulHeight(bounds, width, CANVAS_PAD))),
    [bounds, width, freeHeight],
  )
  // LA TOILE PREND TOUTE LA LARGEUR DU BLOC (2026-09-03, « quand je zoome c'est croppé alors que
  // le bloc du replay est assez grand »). Elle épousait le ratio de la SCÈNE : dès que la hauteur
  // était bornée par l'écran — le cas courant — la toile devenait plus étroite que le bloc, et
  // l'image zoomée se coupait à ses bords pendant que la place d'à côté ne servait à rien.
  //
  // Ce n'est plus la toile qui prend la forme de la scène, c'est la FENÊTRE qui prend la forme de
  // la toile (cf. `frameBounds` juste en dessous).
  const screenWidth = width
  // L'EXPORT REMPLACE L'OFFRE DE L'ÉCRAN PAR LE CADRE DU FORMAT (2026-09-16) : ni la fenêtre ni
  // la hauteur utile de la carte n'entrent dans le fichier. Le cadre de la scène (`frameBounds`)
  // s'élargit alors au 16:9 comme il s'élargit d'ordinaire à la forme de la toile — la carte y
  // est ajustée, entourée du fond si ses proportions diffèrent, jamais rognée ni étirée.
  const exportLayout = useExportLayout()
  const renderWidth = exportLayout ? exportLayout.width : screenWidth
  const renderHeight = exportLayout ? exportLayout.height : screenHeight
  const zRange = useMemo(
    () => ({ min: doc.bounds.minZ ?? 0, max: doc.bounds.maxZ ?? 0 }),
    [doc.bounds.minZ, doc.bounds.maxZ],
  )
  // LE CADRAGE, une fois : le dessin ET le survol doivent lire la MÊME projection — un
  // pointeur qui viserait un autre cadre que celui peint ne toucherait rien.
  //
  // LA FENÊTRE VISIBLE REMPLACE LA SCÈNE DANS LA PROJECTION (2026-09-02, zoom). C'est TOUT ce
  // que le zoom change dans ce fichier, et c'est voulu : la projection est entièrement définie
  // par ses bornes, donc rétrécir les bornes suffit à grossir. `worldToCanvas`, `canvasScale`,
  // le survol, le fond de carte et les quatre calques statiques suivent sans une ligne.
  //
  // LA TAILLE DE DESSIN, ELLE, RESTE CALCULÉE SUR LA SCÈNE (`renderWidth`/`renderHeight`
  // ci-dessus) : `visibleBounds` préserve l'aspect, donc le résultat est le même — mais le dire
  // sur la scène garde ces deux valeurs STABLES au zoom. Le terrain ne doit pas changer de
  // taille quand on grossit ; c'est ce qu'on y montre qui change.
  // LE CADRE — la scène élargie à la forme de la toile. C'est LUI, et non la scène, que le zoom
  // rétrécit et que le déplacement borne : sinon la fenêtre resterait plus étroite que la toile,
  // et le recadrage reviendrait au premier cran de grossissement.
  const frame = useMemo(
    () => frameBounds(bounds, renderWidth, renderHeight, CANVAS_PAD),
    [bounds, renderWidth, renderHeight],
  )
  const userZoom = useReplayZoom(frame)
  // PENDANT UN EXPORT (décisions D6/D7, 2026-09-16) : les gestes de cadrage sont éteints, et le
  // cadrage « carte entière » rend la fenêtre à 1x SANS toucher à l'état du zoom — l'utilisateur
  // le retrouve tel quel à la fin. « Cadrage actuel » garde palier et centre.
  const exporting = exportLayout !== null
  const zoom = exporting ? lockedZoom(userZoom) : userZoom
  const shownLevel = exportLayout?.framing === 'whole' ? 1 : zoom.level
  const viewBounds = useMemo(
    () => visibleBounds(frame, shownLevel, zoom.center.x, zoom.center.y),
    [frame, shownLevel, zoom.center],
  )
  // `exportLayout` EST UNE DÉPENDANCE MÊME QUAND LES TAILLES NE CHANGENT PAS : la densité de
  // rendu, elle, change — un nouveau cadrage force la recuisson des calques statiques à la
  // densité du format (`cookLayer` lit `canvasPixelRatio`), puis à celle de l'écran au retour.
  const canvasView = useMemo(
    () => {
      void exportLayout
      return { bounds: viewBounds, width: renderWidth, height: renderHeight, pad: CANVAS_PAD }
    },
    [viewBounds, renderWidth, renderHeight, exportLayout],
  )
  const screen = useMemo(() => ({ width: screenWidth, height: screenHeight }), [screenWidth, screenHeight])

  return { mapImage, bounds, renderWidth, renderHeight, screen, zRange, canvasView, zoom }
}
