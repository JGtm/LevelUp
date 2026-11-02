/**
 * Largeur du volet des référentiels ancré à droite du contenu (application de bureau).
 *
 * Bornes 280 à 640 px ; la borne haute recule quand la fenêtre est étroite pour que le contenu garde
 * `MIN_CONTENT_WIDTH` à côté de la barre latérale. La largeur choisie survit au redémarrage
 * (`localStorage`, clé `PANEL_WIDTH_STORAGE_KEY`) ; un stockage indisponible (fenêtre privée, accès
 * refusé) ramène la largeur par défaut sans casser l'écran.
 */

export const PANEL_WIDTH_STORAGE_KEY = 'levelup-asset-panel-width'
export const PANEL_MIN_WIDTH = 280
export const PANEL_MAX_WIDTH = 640
export const PANEL_DEFAULT_WIDTH = 360
/** Pas d'une pression sur une flèche de la poignée ; avec Maj, le pas large. */
export const PANEL_KEY_STEP = 16
export const PANEL_KEY_STEP_LARGE = 64

/** Largeur de la barre latérale du bureau (`w-60`). */
const SIDEBAR_WIDTH = 240
/** Largeur minimale laissée au contenu de la page. */
const MIN_CONTENT_WIDTH = 480

/** Borne haute effective : 640 px au plus, moins si la fenêtre ne laisse pas sa place au contenu. */
export function maxPanelWidth(viewportWidth: number): number {
  const room = viewportWidth - SIDEBAR_WIDTH - MIN_CONTENT_WIDTH
  return Math.max(PANEL_MIN_WIDTH, Math.min(PANEL_MAX_WIDTH, Math.floor(room)))
}

/** Ramène une largeur dans les bornes ; une valeur non finie donne la largeur par défaut. */
export function clampPanelWidth(width: number, viewportWidth: number): number {
  if (!Number.isFinite(width)) return Math.min(PANEL_DEFAULT_WIDTH, maxPanelWidth(viewportWidth))
  return Math.min(maxPanelWidth(viewportWidth), Math.max(PANEL_MIN_WIDTH, Math.round(width)))
}

let storageWarned = false

/** Une seule alerte par session : le stockage refusé n'empêche rien, mais doit se voir. */
function warnStorage(action: string, err: unknown): void {
  if (storageWarned) return
  storageWarned = true
  console.warn(`[asset-drawer] largeur du volet : ${action} impossible`, err)
}

/** Largeur mémorisée, ou la largeur par défaut (rien de mémorisé, valeur illisible, stockage refusé). */
export function loadPanelWidth(): number {
  try {
    const raw = window.localStorage.getItem(PANEL_WIDTH_STORAGE_KEY)
    if (raw === null) return PANEL_DEFAULT_WIDTH
    const parsed = Number(raw)
    return Number.isFinite(parsed) ? parsed : PANEL_DEFAULT_WIDTH
  } catch (err) {
    warnStorage('lecture', err)
    return PANEL_DEFAULT_WIDTH
  }
}

/** Mémorise la largeur ; l'échec n'est pas une erreur de l'écran (la largeur courante reste en mémoire). */
export function savePanelWidth(width: number): void {
  try {
    window.localStorage.setItem(PANEL_WIDTH_STORAGE_KEY, String(width))
  } catch (err) {
    // La largeur ne survivra pas au redémarrage, rien d'autre ne change.
    warnStorage('écriture', err)
  }
}
