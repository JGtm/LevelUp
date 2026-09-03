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
import { fitWidth, sceneBounds, usefulHeight, visibleBounds } from './replayLogic'
import type { CanvasView } from './replayDraw'
import type { ReplayDocumentReady } from './replayNormalize'
import { useReplayZoom, type ReplayZoom } from './useReplayZoom'

// LA HAUTEUR DE DESSIN N'EST PLUS FIXE (2026-09-02) : elle est le moindre de ce que l'écran
// laisse et de ce que la carte peut utiliser. La largeur, elle, suit toujours le ratio de la
// scène à cette hauteur (cf. `renderWidth`), sans quoi une carte étirée laisserait des marges
// latérales vides. Le PAD est la marge intérieure du cadrage, en px. Les TOKENS des encres du
// canvas vivent avec elles, dans useReplayInks ; les DURÉES, dans useReplayTiming.
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

/** Le facteur MAXIMAL applique pendant un export (cf. `exportScaleFor`). */
export const EXPORT_SUPERSAMPLE = 2

/**
 * La hauteur de sortie que l'export VISE, en px CSS avant densite d'ecran. C'est la ligne
 * « double 1004x960 » du tableau ci-dessus : le point mesure ou l'ecart de chrominance tombe a
 * 0,76 pour un fichier de 96 Ko. Le tableau a ete releve a hauteur de toile fixe (480) ; depuis
 * que la toile s'adapte a l'ecran, c'est cette CIBLE qui est la constante, et le facteur qui
 * s'ajuste.
 */
export const EXPORT_TARGET_HEIGHT = 960

/**
 * exportScaleFor — le facteur de surechantillonnage pour une hauteur de toile donnee.
 *
 * IL REND L'EXPORT INVARIANT A LA TAILLE D'ECRAN, ce qui est tout l'interet : une toile de 480
 * est doublee (exactement le comportement d'avant, au pixel pres), une toile de 720 n'est
 * multipliee que par 1,33 — et les deux sortent la MEME video de 960 de haut. Sans lui, une
 * fenetre plus grande produirait mecaniquement un fichier plus lourd a encoder, pour la seule
 * raison que la personne a un grand ecran.
 *
 * Borne a `EXPORT_SUPERSAMPLE` : sur une petite toile, surechantillonner davantage couterait
 * plus d'encodage que ce que la mesure justifie.
 */
export function exportScaleFor(viewHeight: number): number {
  if (!(viewHeight > 0)) return EXPORT_SUPERSAMPLE
  return Math.min(EXPORT_TARGET_HEIGHT / viewHeight, EXPORT_SUPERSAMPLE)
}

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
  const renderHeight = useMemo(
    () =>
      width === 0
        ? freeHeight
        : Math.floor(Math.min(freeHeight, usefulHeight(bounds, width, CANVAS_PAD))),
    [bounds, width, freeHeight],
  )
  // Largeur de dessin = ratio de la scène à la hauteur retenue (évite les marges latérales).
  // Elle SUIT donc la hauteur : un terrain plus court est aussi plus étroit, le cadrage reste
  // celui de la carte.
  const renderWidth = useMemo(
    () => (width === 0 ? 0 : Math.floor(fitWidth(bounds, width, renderHeight, CANVAS_PAD))),
    [bounds, width, renderHeight],
  )
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
  const zoom = useReplayZoom(bounds)
  const viewBounds = useMemo(
    () => visibleBounds(bounds, zoom.level, zoom.center.x, zoom.center.y),
    [bounds, zoom.level, zoom.center],
  )
  const canvasView = useMemo(
    () => ({ bounds: viewBounds, width: renderWidth, height: renderHeight, pad: CANVAS_PAD }),
    [viewBounds, renderWidth, renderHeight],
  )

  return { mapImage, bounds, renderWidth, renderHeight, zRange, canvasView, zoom }
}
