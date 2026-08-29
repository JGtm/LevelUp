/**
 * replayExportPlan.ts — QUELLES IMAGES L'EXPORT VA PEINDRE, et à quel instant du fichier.
 *
 * # LE CALCUL QUI SÉPARE UN EXPORT D'UN ENREGISTREMENT
 *
 * L'enregistrement temps réel n'a rien à décider : il filme ce qui passe, et le nombre
 * d'images qu'il obtient dépend de la charge de la machine. L'export, lui, ÉCHANTILLONNE : il
 * choisit à l'avance une suite d'instants du match, régulièrement espacés, et va chercher
 * l'image du rejeu à chacun d'eux. C'est ce module qui pose cette suite.
 *
 * DEUX AXES DE TEMPS, ET IL NE FAUT JAMAIS LES CONFONDRE :
 *
 * - l'axe du DOCUMENT, en images du film (`frameIntervalMs` donne leur durée réelle) ;
 * - l'axe du FICHIER, en microsecondes, à cadence fixe (30 im/s, cf. `EXPORT_FPS`).
 *
 * Les deux ne tombent PAS l'un sur l'autre : un film à 24,7 images par seconde exporté à 30
 * demande des images entre les images. Ce n'est pas un problème — `draw()` accepte une image
 * FRACTIONNAIRE (c'est déjà ce que fait la boucle de lecture, `frameRef` y avance par pas de
 * `dt * fps`) et interpole les positions. Arrondir ici à l'image entière la plus proche
 * produirait un mouvement saccadé là où le rejeu est fluide à l'écran.
 *
 * # POURQUOI UN MODULE PUR, ET PAS TROIS LIGNES DANS LA BOUCLE
 *
 * Le nombre d'images conditionne la barre de progression, la durée annoncée et la mémoire
 * consommée. Le calculer dans la boucle le rendrait invérifiable ; ici, il se teste sans
 * navigateur, sans encodeur et sans canvas.
 */
import { frameToMs, msToFrames } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { EXPORT_FPS } from './replayVideoEncoder'
import type { ReplayWindowBounds } from './replayWindow'

/** Les deux bornes de l'export, sur l'axe du DOCUMENT (images, éventuellement fractionnaires). */
export interface ExportBounds {
  startFrame: number
  endFrame: number
}

/**
 * DUREE MINIMALE DE MAINTIEN de la derniere image, en millisecondes de clip, quand la plage va
 * jusqu'au bout du match.
 *
 * POURQUOI CE MAINTIEN EXISTE. L'ecran de fin de match ne se peint QU'A la borne de fin, et
 * l'echantillonnage ne pose qu'UNE image sur cette borne : le clip se terminait donc sur un
 * eclair de 1/30 s de verdict, imperceptible — l'objectif meme que `buildExportPlan` s'assigne
 * n'etait pas atteint. Trois secondes suffisent a lire un verdict et un score.
 */
export const END_HOLD_MS = 3000

/** Le plan complet : ce que la boucle parcourt, et ce que la progression affiche. */
export interface ExportPlan {
  /**
   * Les images du document à peindre, dans l'ordre. Fractionnaires : `draw()` interpole.
   * L'indice dans ce tableau EST le numéro d'image du fichier — c'est lui qui donne
   * l'horodatage, jamais la valeur.
   */
  frames: number[]
  /** Cadence du fichier produit, en images par seconde. */
  fps: number
  /** Durée de la PLAGE de match exportée, en millisecondes — jamais celle du calcul. */
  durationMs: number
  /** Millisecondes de maintien de la derniere image, APRES la plage (0 = aucun). */
  holdMs: number
}

/**
 * defaultExportBounds — la plage proposée par défaut : la fenêtre de GAMEPLAY entière.
 *
 * Sans fenêtre (artefact ancien, en-tête sans durée jouable), on retombe sur le film entier :
 * c'est moins juste — le countdown d'avant-match et la queue y sont — mais c'est tout ce que
 * le document permet d'affirmer, et refuser l'export serait pire.
 */
export function defaultExportBounds(
  doc: ReplayDocumentReady,
  playWindow: ReplayWindowBounds | null,
): ExportBounds {
  if (playWindow) return { startFrame: playWindow.startFrame, endFrame: playWindow.endFrame }
  return { startFrame: 0, endFrame: Math.max(doc.frameCount - 1, 0) }
}

/**
 * clampExportBounds ramène des bornes dans le domaine autorisé et les remet dans l'ordre.
 *
 * LES BORNES VIENNENT DE L'UTILISATEUR (deux champs du dialogue) : il peut les croiser, les
 * pousser hors du match, ou demander une plage vide. Aucun de ces trois cas n'est une erreur à
 * lui remonter — ce sont des gestes, et un geste se borne.
 */
export function clampExportBounds(
  wanted: ExportBounds,
  domain: ExportBounds,
): ExportBounds {
  const lo = Math.min(domain.startFrame, domain.endFrame)
  const hi = Math.max(domain.startFrame, domain.endFrame)
  const a = Math.min(Math.max(wanted.startFrame, lo), hi)
  const b = Math.min(Math.max(wanted.endFrame, lo), hi)
  return { startFrame: Math.min(a, b), endFrame: Math.max(a, b) }
}

/**
 * buildExportPlan pose la suite des images à peindre.
 *
 * LA DERNIÈRE IMAGE EST TOUJOURS LA BORNE DE FIN, exactement — et c'est une décision, pas un
 * effet de bord de la boucle. L'échantillonnage régulier tombe presque toujours AVANT la
 * borne (la plage n'est pas un multiple entier de 1/30 s) ; s'arrêter là couperait le clip
 * juste avant l'écran de fin de match, qui ne se peint QU'À cette borne. Un clip de fin de
 * match sans son écran de fin serait le seul défaut que tout le monde remarquerait.
 *
 * Une plage vide ou dégénérée (bornes égales, document sans échelle de temps) rend UNE image :
 * un clip d'une image est un arrêt sur image, ce qui reste un résultat ; zéro image serait un
 * fichier vide que le navigateur téléchargerait quand même.
 */
export function buildExportPlan(
  bounds: ExportBounds,
  doc: ReplayDocumentReady,
  fps: number = EXPORT_FPS,
  /**
   * Maintien de la DERNIERE image apres la plage. L'appelant le calcule : il vaut au moins
   * `END_HOLD_MS` quand la plage atteint la fin du match, et au moins la queue du son (une
   * fanfare de fin dure jusqu'a 11 s) — sans quoi la piste sonore jouerait sur un lecteur qui
   * n'a plus d'image a montrer.
   */
  holdMs = 0,
): ExportPlan {
  const startMs = frameToMs(bounds.startFrame, doc)
  const endMs = frameToMs(bounds.endFrame, doc)
  const durationMs = Math.max(endMs - startMs, 0)
  const stepMs = 1000 / fps
  // `floor` et non `round` : une image de plus que la plage ferait déborder le clip au-delà de
  // la borne demandée, et la dernière image est de toute façon posée sur la borne exacte.
  //
  // L'EPSILON N'EST PAS DE LA SUPERSTITION. `1000 / 30` ne tombe pas juste en binaire :
  // `2000 / (1000 / 30)` vaut 59,999999999999996 et non 60, donc `floor` rendait 59 — une
  // image de moins dès que la plage est un multiple exact de la cadence, c'est-à-dire dans le
  // cas le plus courant. La tolérance rend le compte prévisible sans jamais déborder : elle
  // est mille fois plus petite qu'une image.
  const steps = Math.max(Math.floor(durationMs / stepMs + 1e-9), 0)
  const frames: number[] = []
  for (let i = 0; i < steps; i++) {
    frames.push(bounds.startFrame + msToFrames(i * stepMs, doc))
  }
  frames.push(bounds.endFrame)
  // LE MAINTIEN REPETE LA DERNIERE IMAGE : elle est deja peinte, le cout est celui de son
  // encodage seul (une image identique se compresse a presque rien).
  const held = Math.max(Math.floor(holdMs / stepMs + 1e-9), 0)
  for (let i = 0; i < held; i++) frames.push(bounds.endFrame)
  return { frames, fps, durationMs, holdMs: Math.max(holdMs, 0) }
}

/**
 * exportProgressPct — la part parcourue, en pourcentage borné.
 *
 * Ici et pas dans le composant : c'est la même règle que `writeCursor` pour la frise — une
 * part qui se calcule à deux endroits finit par diverger, et une barre de progression qui
 * dépasse 100 % est le genre de détail qui fait douter du reste.
 */
/**
 * Sous ce reste, l'estimation ne sert plus a rien : elle dit « on y est presque », ce que la
 * barre montre deja. Vu en recette le 2026-08-28 — sur un export rapide, elle affichait
 * « environ 0:00 restantes » pendant toute sa duree, ce qui se lit « c'est fini » alors que ca
 * tourne encore.
 */
const ETA_SHOW_MIN_MS = 3000
/** Au-dela d'une minute, `m:ss` ; en deca, des SECONDES — « 0:07 » se lit moins bien que « 7 s ». */
const ETA_MINUTE_MS = 60_000

/**
 * etaLabel met en forme le temps restant, ou rend `null` quand il ne faut rien afficher.
 *
 * `null` couvre DEUX cas qu'il ne faut pas confondre : l'estimation pas encore fiable (l'appelant
 * n'a rien a nous donner) et l'estimation devenue inutile (moins de trois secondes).
 */
export function etaLabel(ms: number | undefined): string | null {
  if (ms === undefined || !Number.isFinite(ms) || ms < ETA_SHOW_MIN_MS) return null
  if (ms < ETA_MINUTE_MS) return `${Math.round(ms / 1000)} s`
  const total = Math.round(ms / 1000)
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`
}

export function exportProgressPct(done: number, total: number): number {
  if (total <= 0) return 0
  return Math.min(100, Math.max(0, (done / total) * 100))
}
