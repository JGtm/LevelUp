/**
 * relief/types — LES TYPES DE LA BIBLIOTHEQUE DU RELIEF (fond « Relief », plan 2026-11-18, lot D).
 *
 * Deux fichiers par carte alimentent la bibliothèque (`.ai/annexes_fonds_relief/conception.md` §1) :
 *   - `<clé>.relief.webp` : une image dont chaque pixel porte une hauteur et deux drapeaux, décodée
 *     en `ReliefHeights` (`decodeImage.ts`) ;
 *   - `<clé>.relief.json` : le calage, les étages, les sols du maillage et les traits, décodé en
 *     `ReliefData` (`decodeData.ts`).
 *
 * UNITES : les hauteurs et les distances sont en UNITES DU JEU (une unité vaut environ 3 m), jamais
 * en mètres. Le code historique appelle « mètres » ces unités (`metersPerPixel`) ; ce nom est celui
 * du contrat d'API, il est gardé tel quel.
 *
 * REPERE DES PIXELS : un pixel `(i, j)` a son centre en `(i, j)` ; son centre monde est
 * `originX + (i + 0,5) × mpp`, `originY − (j + 0,5) × mpp` (`pixelToWorld`). Les coordonnées de
 * traits portées par `LineSet` sont dans ce repère (le centre du pixel est l'entier).
 *
 * Aucune dépendance au DOM : ces types voyagent entre le fil principal et le worker.
 */

/** Le calage du fond publié auquel le relief a été projeté (copie du sidecar). */
export interface ReliefCalibration {
  /** Unités du jeu par pixel (le contrat d'API dit « mètres »). */
  metersPerPixel: number
  originX: number
  originY: number
  widthPx: number
  heightPx: number
}

/** Le décodage de l'image et les bornes de la rampe de clarté. */
export interface ReliefHeightMeta {
  /** Hauteur de `q = 0`, en unités entières. */
  z0: number
  /** Hauteur d'un pas de `q` (1/256 u). */
  step: number
  /** 2e centile des hauteurs de la zone : bas de la rampe de clarté. */
  low: number
  /** 98e centile des hauteurs de la zone : haut de la rampe de clarté. */
  high: number
}

/** Les degradations déclarées par la cuisson (codes, jamais de phrases : le web les traduit). */
export type ReliefDegradation = 'sans-maillage' | 'hauteurs-ecretees' | 'pile-tronquee'

/**
 * Un ensemble de polylignes à plat. `points` porte `x, y` à la suite ; `offsets` (longueur n + 1)
 * donne, pour la polyligne `k`, l'indice de son premier SOMMET (`points[2 * offsets[k]]`) ; la
 * polyligne `k` couvre les sommets `offsets[k]` à `offsets[k + 1] - 1`.
 */
export interface LineSet {
  points: Float32Array
  offsets: Uint32Array
}

/** Les sols du maillage de navigation : sommets indexés et faces convexes (représentation à plat). */
export interface FloorMesh {
  /** `x, y, z` à la suite, en unités MONDE (le fichier les porte en centièmes d'unité). */
  vertices: Float64Array
  /** Face `f` : indices `faceIndices[faceStart[f] .. faceStart[f + 1] - 1]` ; longueur nombre de faces + 1. */
  faceStart: Uint32Array
  faceIndices: Uint32Array
  /** Boîte englobante monde de chaque face : `minX, minY, maxX, maxY` à la suite. */
  bounds: Float64Array
}

/** Les sols du fichier : étages détectés, seuils de la pile et maillage. */
export interface ReliefFloors {
  /** Hauteurs des étages détectés, croissantes ; vide sans maillage. */
  levels: readonly number[]
  /** Bornes du curseur de hauteur (1er et 99e centiles des sols) ; absentes sans maillage. */
  min: number | null
  max: number | null
  /** Fusion des sols empilés, en unités. */
  merge: number
  /** Ecart toléré entre un dessus et son sol, en unités. */
  tolerance: number
  /** Part de la zone dont la hauteur vient du maillage, 0..1 ; `null` si le fichier ne la porte pas. */
  meshShare: number | null
  mesh: FloorMesh
}

/** Les trois familles de traits, déjà décodées en polylignes de pixels. */
export interface ReliefLines {
  /** Ruptures de hauteur. */
  steps: LineSet
  /** Courbes de niveau. */
  contours: LineSet
  /** Liseré : anneaux FERMES, le dernier sommet se relie au premier. */
  outline: LineSet
}

/** Le contenu du `.relief.json`, validé et décodé. */
export interface ReliefData {
  schemaVersion: 1
  calibration: ReliefCalibration
  height: ReliefHeightMeta
  floors: ReliefFloors
  lines: ReliefLines
  degradations: readonly ReliefDegradation[]
}

/** Les pixels de l'image : une hauteur et des drapeaux par pixel. */
export interface ReliefHeights {
  width: number
  height: number
  /** Hauteur de chaque pixel en unités ; `NaN` quand elle est inconnue (A = 0). */
  heights: Float32Array
  /** Drapeaux de chaque pixel : `FLAG_ON_MAP`, `FLAG_MESH_FLOOR`, `FLAG_KNOWN`. */
  flags: Uint8Array
}

/** Un point du monde ou d'un repère de pixels. */
export interface PointXY {
  x: number
  y: number
}
