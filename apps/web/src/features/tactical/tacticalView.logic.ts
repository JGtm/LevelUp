/**
 * tacticalView.logic — la logique PURE de la vue d'analyse tactique (Phase 5, items
 * 5.2-5.6). Rien ne dépend de React ni du DOM : titre de page, messages de statut,
 * unité par question, projection du raster serveur en grille de peinture (`heatPaint`)
 * et position (col, row) d'un clic sur le canvas. Les composants ne font que rendre ce
 * que ces fonctions décident (règle du dépôt : pas de logique métier dans un composant).
 */
import type { BornesMonde, CelluleTactique, EchelleTactique } from '@/lib/api/types'

import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import { buildTacticalGrid, type MapFrame, type TacticalGrid } from '@/lib/replay/heatPaint'
import type { TacticalText } from './i18n'

/** Les six lectures offertes par la barre d'outils — même vocabulaire que le contrat
 *  (`TacticalRasterBody.question`). */
export type TacticalQuestion = 'morts' | 'kills' | 'gagne' | 'temps' | 'routes' | 'isole'

/** L'axe « qui » — même vocabulaire que le contrat (`TacticalRasterBody.qui`). */
export type TacticalQui = 'moi' | 'escouade' | 'adv'

/**
 * Plancher de mesure par cellule : 3 matchs distincts. MÊME SEUIL QUE LE SERVEUR
 * (calibration mesurée de `mappos-build`, `.ai/PLAN_TACTIQUE_2026-09-06.md` §6,
 * "Plancher par cellule") — affiché, jamais recalculé : le serveur ne publie que les
 * cellules déjà au-dessus.
 */
export const TACTICAL_CELL_FLOOR = 3

/** Questions qui exigent l'artefact de rejeu (position datée), pas seulement le journal
 *  des morts — même liste que la doc du contrat (`TacticalRasterBody.question`). */
const QUESTIONS_ARTEFACT_REJEU: ReadonlySet<TacticalQuestion> = new Set(['temps', 'routes'])

/** pageTitle — « Plan de <carte> — <question> », le titre H2 de la vue. */
export function pageTitle(t: TacticalText, mapName: string, question: TacticalQuestion): string {
  const libelle = t.analysisQuestions.find((q) => q.id === question)?.label ?? question
  return t.analysisPageTitle(mapName, libelle)
}

/** unitForQuestion — l'unité affichée en légende du plan et sur la cellule sélectionnée. */
export function unitForQuestion(t: TacticalText, question: TacticalQuestion): string {
  return t.units[question]
}

/** sourceForQuestion — la provenance de la mesure, affichée au pied du plan. */
export function sourceForQuestion(t: TacticalText, question: TacticalQuestion): string {
  return QUESTIONS_ARTEFACT_REJEU.has(question) ? t.sourceReplay : t.sourceJournal
}

/**
 * statusMessages — les bandeaux « en attente » / « non disponible » au-dessus du plan.
 * LES DEUX PEUVENT COEXISTER (des matchs en cours de cuisson ET d'autres jamais
 * cuisables) : ce ne sont pas des échecs de la lecture, ce sont des dénominateurs qui
 * varient. Aucun message quand les deux compteurs sont à zéro.
 */
export function statusMessages(
  t: TacticalText,
  matchsEnAttente: number,
  matchsNonCuisables: number,
): string[] {
  const messages: string[] = []
  if (matchsEnAttente > 0) messages.push(t.statusPending(matchsEnAttente))
  if (matchsNonCuisables > 0) messages.push(t.statusUnavailable(matchsNonCuisables))
  return messages
}

/**
 * Les trois causes d'un plan vide. `null` = le plan n'est pas vide.
 *
 * ELLES NE SE DISENT PAS PAREIL, et c'est tout l'objet de cette distinction (point 21 des
 * retours utilisateur, lot 3.2) :
 *
 *	aucun-match     le FILTRE ne retient aucun match sur cette carte. Rien n'a été mesuré
 *	                parce que rien n'a été joué dans ce périmètre.
 *	aucune-mesure   des matchs, mais aucun mesurable — film jamais décodé, journal des
 *	                morts illisible. C'est le message historique, et il reste vrai ici.
 *	densite         des matchs MESURÉS, mais trop dispersés : aucune zone n'atteint le
 *	                plancher de 3 matchs distincts, même à la grille la plus grossière.
 *
 * LE DÉFAUT CORRIGÉ : le troisième cas affichait le message du deuxième. Sur Illusion,
 * 38 matchs étaient retenus ET mesurés, et la page répondait « pas assez de matchs
 * mesurés » — un message qui envoie élargir un filtre déjà large, pour un problème qui
 * n'est pas là.
 */
export type TacticalPlanEmptyReason = 'aucun-match' | 'aucune-mesure' | 'densite'

/**
 * planEmptyReason — pourquoi le plan est vide, ou `null` s'il ne l'est pas.
 *
 * La cause se lit sur les DEUX dénominateurs déjà publiés par le contrat
 * (`matchs_filtres`, `matchs_retenus`) et sur le nombre de cellules peintes : aucun champ
 * supplémentaire n'est nécessaire, et aucune règle n'est recalculée côté client — le
 * serveur ne publie que les cellules déjà au-dessus du plancher.
 */
export function planEmptyReason(
  cellulesPeintes: number,
  matchsRetenus: number,
  matchsFiltres: number,
): TacticalPlanEmptyReason | null {
  if (cellulesPeintes > 0) return null
  if (!(matchsFiltres > 0)) return 'aucun-match'
  if (!(matchsRetenus > 0)) return 'aucune-mesure'
  return 'densite'
}

/** planEmptyText — le titre et la description à afficher pour une cause donnée. */
export function planEmptyText(
  t: TacticalText,
  raison: TacticalPlanEmptyReason,
  matchsRetenus: number,
  pasM: number,
): { title: string; description: string } {
  switch (raison) {
    case 'aucun-match':
      return { title: t.planEmptyNoMatchTitle, description: t.planEmptyNoMatchDescription }
    case 'aucune-mesure':
      return { title: t.planEmptyTitle, description: t.planEmptyDescription }
    default:
      return {
        title: t.planEmptyDensityTitle,
        // LE PAS CITÉ EST CELUI QUE LA LECTURE A RETENU : quand aucune densité ne suffit,
        // c'est le plus grossier essayé, et le dire évite qu'on croie le plan calculé
        // à 0,5 m.
        description: t.planEmptyDensityDescription(matchsRetenus, TACTICAL_CELL_FLOOR, pasM),
      }
  }
}

/** ratioSafe — une proportion 0..1, jamais une division par zéro. */
export function ratioSafe(numerateur: number, denominateur: number): number {
  if (!(denominateur > 0)) return 0
  return numerateur / denominateur
}

/**
 * LÉGENDE DU PLAN — les deux bornes affichées de part et d'autre de la rampe, et le mode
 * de rampe à employer (maquette 034b1915).
 *
 * Une lecture SIGNÉE se lit de −borne à +borne autour du zéro ; les autres de 0 au p95,
 * saturées au-delà. L'unité n'est accolée qu'à la borne haute — la répéter des deux côtés
 * n'ajoute rien et coupe la rampe en deux.
 */
export function planLegend(
  echelle: EchelleTactique,
  unite: string,
  format: (n: number) => string,
): { lo: string; hi: string; mode: 'intensity' | 'divergent' } {
  if (echelle.symetrique) {
    const borne = Math.abs(echelle.borne)
    return {
      lo: `− ${format(borne)}`,
      hi: `+ ${format(borne)} ${unite}`.trim(),
      mode: 'divergent',
    }
  }
  return { lo: format(0), hi: `${format(echelle.p95)} ${unite}`.trim(), mode: 'intensity' }
}

/**
 * QUESTIONS SANS CELLULE. « Mes routes de spawn » empile des trajets : il n'y a pas de
 * grandeur par cellule à détailler, donc pas de match à ouvrir depuis le plan. La carte
 * « Cellule sélectionnée » le DIT, au lieu d'inviter à un clic qui ne rendrait rien.
 */
export function questionSansCellule(question: TacticalQuestion): boolean {
  return question === 'routes'
}

/**
 * PLAN_ASPECT_DEFAUT — le rapport largeur/hauteur du cadre du plan quand ni le fond ni les
 * bornes ne disent rien d'exploitable : 16/9, celui des vignettes de carte (`TacticalMapTile`,
 * `aspect-video`), qui affichent LE MÊME fond. Un plan vide a donc exactement la taille
 * d'une carte normale, avec son état vide par-dessus.
 */
export const PLAN_ASPECT_DEFAUT = 16 / 9

/**
 * PLAN_HAUTEUR_MAX_PX — plafond de hauteur du cadre, en pixels.
 *
 * LE DÉFAUT QU'IL FERME (constaté le 2026-09-09 sur le plan d'Illusion) : le cadre était
 * mis au seul `aspect-ratio` des bornes, sur une largeur de conteneur libre — un rapport
 * très allongé rendait alors un canvas de 1 070 x 13 375 px, une hauteur qui n'est plus
 * une page.
 *
 * LE RAPPORT N'EST JAMAIS DÉFORMÉ POUR TENIR : le peintre projette le monde avec UNE
 * SEULE échelle (px par mètre) et le clic s'inverse par la même règle de trois — un cadre
 * dont le rapport ne serait plus celui des bornes désalignerait les deux. Le plafond passe
 * donc par une LARGEUR maximale (`rapport x plafond`), qui laisse `aspect-ratio` intact.
 */
export const PLAN_HAUTEUR_MAX_PX = 720

/** trouveCellule — la cellule serveur à (col, row), ou `null` si jamais atteinte. */
export function trouveCellule(
  cellules: readonly CelluleTactique[],
  col: number,
  row: number,
): CelluleTactique | null {
  return cellules.find((c) => c.col === col && c.lig === row) ?? null
}

// TACTICAL_REPLAY_FRAME_INTERVAL_MS / instantToFrame ont vécu ici (lot M1, « voir dans le
// rejeu ») : une conversion instant -> frame MÉCANIQUE, sans correction du décalage
// d'horloge match/film pour quatre questions sur six. Retirées le 2026-09-08 (lot M1b,
// décision utilisateur ferme « corriger le décalage ») — mortes : `TacticalCellCard` ne
// pré-calcule plus de frame, il construit `?t=&clock=` et laisse la ROUTE du rejeu
// convertir une fois le document (et son calage) chargé
// (`lib/replay/replayLogic.resolveTacticalReplayInstant` + `msToFrames`).

// ─── Section « Coordination d'équipe » (maquette 034b1915) ────────────────────

/**
 * libelleRayons — « 18 m », ou « 18 m ou 24 m » quand le filtre mélange deux formats.
 * JAMAIS une moyenne : la moyenne de deux règles du jeu n'est la règle d'aucun match.
 */
export function libelleRayons(
  t: TacticalText,
  rayons: readonly number[],
  locale: Locale,
): string {
  return rayons
    .map((r) => t.radiusValue(formatDistanceM(r, locale)))
    .join(t.radiusJoin)
}

/**
 * DISTANCE_DECIMALES — une decimale pour toute distance en metres affichee par l'onglet.
 *
 * LE DEFAUT QU'ELLE FERME : la mediane sortait du serveur en flottant brut et ICU la rendait
 * telle quelle — « Distance mediane a l'equipier : 9,905 m » (constate le 2026-09-13). Un
 * millimetre n'a aucun sens sur une distance mesuree entre deux joueurs, et le reste de la
 * carte est deja au dixieme.
 */
export const DISTANCE_DECIMALES = 1

/**
 * formatDistanceM — une distance en metres, AU PLUS au dixieme, dans la langue courante.
 *
 * « AU PLUS », et pas « exactement » : les rayons de la table de regulation sont des entiers
 * (18 m, 24 m) et `formatNumber` les rembourrerait en « 18,0 m ». Une portee de radar ne se
 * mesure pas au decimetre ; la mediane, elle, en a besoin d'un.
 */
export function formatDistanceM(metres: number, locale: Locale): string {
  return new Intl.NumberFormat(intlLocale(locale), {
    maximumFractionDigits: DISTANCE_DECIMALES,
  }).format(metres)
}

/**
 * positionCategorie — où tombe une distance sur l'axe des CATÉGORIES de l'histogramme des
 * distances, en indice fractionnaire (18 m sur des intervalles de 10 m = 1,3).
 *
 * L'indice `i` désigne le CENTRE de la barre `i`, pas son bord gauche : le bord gauche est
 * donc à `i − 0,5`, et on y ajoute la position dans l'intervalle. Arrondir à une frontière
 * de barre déplacerait la règle du jeu à l'écran.
 *
 * `null` quand la distance sort des intervalles servis : mieux vaut aucun seuil qu'un seuil
 * collé au bord du graphe, qui se lirait comme une valeur mesurée.
 */
export function positionCategorie(
  distance: number,
  bins: readonly { min_m: number; max_m?: number | null }[],
): number | null {
  for (let i = 0; i < bins.length; i += 1) {
    const min = bins[i].min_m
    const max = bins[i].max_m
    if (max == null) return distance >= min ? i : null
    if (distance >= min && distance < max) {
      return i - 0.5 + (distance - min) / (max - min)
    }
  }
  return null
}

// ─── LA PROJECTION DU PLAN — refaite le 2026-09-13, sur constat utilisateur ───
//
// CE QUI ÉTAIT FAUX, ET QUI L'ÉTAIT DE TROIS FAÇONS À LA FOIS (mesuré sur Illusion, 54
// matchs, « Où je meurs », réponse `/raster` du serveur en main) :
//
//  1. LE PEINTRE JETAIT LES CELLULES D'INDEX NÉGATIF. Le serveur adresse ses cellules sur
//     l'ORIGINE DU MONDE (`col = floor(x / pas)`), donc en nombres signés ; `drawHeatmap`
//     n'énumère que `0..nx` et `0..ny`. Sur Illusion : 11 cellules dessinées sur les 54
//     servies. C'est tout le « calque quasi vide ».
//  2. LE CALQUE ET LE FOND N'ÉTAIENT PAS DANS LE MÊME REPÈRE. Le cadre prenait la boîte
//     englobante des cellules MESURÉES (30 x 36 m sur Illusion) et l'image du fond, qui
//     couvre 53 x 69 m, y était étirée : aucune correspondance monde vers image. D'où des
//     zones chaudes « à côté du bâtiment ».
//  3. L'AXE Y N'ÉTAIT PAS INVERSÉ, alors que le calage publié avec chaque fond l'impose
//     (`yMonde = originY - (py + 0.5) * metersPerPixel`).
//
// LE PLANCHER N'Y EST POUR RIEN, et c'est mesuré : 417 morts localisées sur 433 (96 %),
// 54 cellules retenues à 2 m. La carte avait de quoi être peinte.
//
// CE QU'ON FAIT MAINTENANT — exactement ce que fait « Occupation du terrain » (`_positionsHeat.ts`),
// qui pose 100 % de ses positions sur le plan : le repère est LE CADRE DU FOND, les cellules
// serveur y sont réindexées en 0-based, et Y est inversé.

/**
 * RepereTactique — le rectangle MONDE dans lequel le canvas du plan est tracé, et le pas de
 * la grille servie.
 *
 * Il vient du CADRE DU FOND quand la carte en a un (le cas normal : c'est lui qui aligne le
 * calque sur l'image), et de la boîte englobante des cellules sinon — une carte sans fond
 * figé n'affiche aucune image, le calque s'y lit seul et son cadrage propre reste le bon.
 */
export interface RepereTactique {
  minX: number
  maxX: number
  /** Y monde du bord HAUT du cadre (le Y décroît vers le bas de l'image). */
  maxY: number
  minY: number
  pasM: number
}

/**
 * repereDuPlan — le repère de projection : le cadre du fond s'il est connu, la boîte
 * englobante des cellules sinon. `null` quand ni l'un ni l'autre n'est exploitable.
 */
export function repereDuPlan(
  fond: MapFrame | null,
  bornes: BornesMonde,
  pasM: number,
): RepereTactique | null {
  if (!(pasM > 0)) return null
  if (fond && fond.widthM > 0 && fond.heightM > 0) {
    return {
      minX: fond.originX,
      maxX: fond.originX + fond.widthM,
      maxY: fond.originY,
      minY: fond.originY - fond.heightM,
      pasM,
    }
  }
  const largeur = bornes.max_x - bornes.min_x
  const hauteur = bornes.max_y - bornes.min_y
  if (!bornes.valide || !(largeur > 0) || !(hauteur > 0)) return null
  return { minX: bornes.min_x, maxX: bornes.max_x, maxY: bornes.max_y, minY: bornes.min_y, pasM }
}

/** Le rapport largeur/hauteur du repère — celui que prend le cadre de la carte. */
export function repereAspect(repere: RepereTactique): number {
  return (repere.maxX - repere.minX) / (repere.maxY - repere.minY)
}

/** Nombre de colonnes et de lignes du repère, au pas servi. */
export function repereDimensions(repere: RepereTactique): { nx: number; ny: number } {
  return {
    nx: Math.max(1, Math.ceil((repere.maxX - repere.minX) / repere.pasM)),
    ny: Math.max(1, Math.ceil((repere.maxY - repere.minY) / repere.pasM)),
  }
}

/** L'adresse 0-based, dans le repère, d'une cellule SERVEUR (ancrée sur l'origine du monde). */
function adresseDansLeRepere(
  col: number,
  lig: number,
  repere: RepereTactique,
): { col: number; row: number } {
  const centreX = (col + 0.5) * repere.pasM
  const centreY = (lig + 0.5) * repere.pasM
  return {
    col: Math.floor((centreX - repere.minX) / repere.pasM),
    row: Math.floor((repere.maxY - centreY) / repere.pasM),
  }
}

/**
 * grilleDuPlan — la grille de peinture, RÉINDEXÉE sur le repère.
 *
 * Chaque cellule serveur est replacée par son CENTRE monde, puis adressée en 0-based dans le
 * cadre (Y inversé). Une cellule hors du cadre est IGNORÉE, jamais rabattue sur un bord : un
 * point hors carte n'a rien à dire d'un bord (même règle que « Occupation du terrain »).
 */
export function grilleDuPlan(
  cellules: readonly CelluleTactique[],
  repere: RepereTactique,
  echelle: EchelleTactique,
): TacticalGrid | null {
  const { nx, ny } = repereDimensions(repere)
  const cells: { col: number; row: number; value: number }[] = []
  for (const c of cellules) {
    const adresse = adresseDansLeRepere(c.col, c.lig, repere)
    if (adresse.col < 0 || adresse.col >= nx || adresse.row < 0 || adresse.row >= ny) continue
    cells.push({ col: adresse.col, row: adresse.row, value: c.valeur })
  }
  if (cells.length === 0) return null
  return buildTacticalGrid(
    cells,
    { cell: repere.pasM, nx, ny, minX: repere.minX, minY: repere.minY },
    { lo: echelle.p50, hi: echelle.p95, signee: echelle.symetrique, borne: echelle.borne },
    cells.length,
  )
}

/**
 * vueDuPlan — la projection monde vers canvas d'un repère : ce que `drawTacticalHeatmap`
 * attend, une fois les cellules réindexées en 0-based (l'origine du calque est donc le coin
 * du canvas, et l'échelle celle du cadre).
 */
export function vueDuPlan(
  repere: RepereTactique,
  canvasWidth: number,
): { topLeftWorld: { x: number; y: number }; scale: number } | null {
  const largeur = repere.maxX - repere.minX
  if (!(largeur > 0) || !(canvasWidth > 0)) return null
  return { topLeftWorld: { x: 0, y: 0 }, scale: canvasWidth / largeur }
}

/**
 * celluleDuClic — l'adresse SERVEUR (ancrée sur l'origine du monde) de la cellule sous un
 * clic. C'est cette adresse-là que `/tactical/{map}/cellule` attend, et celle que portent
 * les `CelluleTactique.col/lig` de la réponse agrégée.
 *
 * `null` si le clic tombe hors du canvas.
 */
export function celluleDuClic(
  clickX: number,
  clickY: number,
  canvas: { width: number; height: number },
  repere: RepereTactique,
): { col: number; row: number } | null {
  const { width, height } = canvas
  if (!(width > 0) || !(height > 0)) return null
  if (clickX < 0 || clickY < 0 || clickX > width || clickY > height) return null
  const worldX = repere.minX + (clickX / width) * (repere.maxX - repere.minX)
  const worldY = repere.maxY - (clickY / height) * (repere.maxY - repere.minY)
  return { col: Math.floor(worldX / repere.pasM), row: Math.floor(worldY / repere.pasM) }
}

/**
 * rectSelection — le cadre, en pixels canvas, de la cellule SERVEUR choisie. Même projection
 * que la peinture, donc le cadre se pose exactement sur la cellule peinte. `null` quand rien
 * n'est choisi ou que la cellule tombe hors du cadre du fond.
 */
export function rectSelection(
  selected: { col: number; row: number } | null,
  repere: RepereTactique,
  canvasWidth: number,
): { x: number; y: number; size: number } | null {
  if (!selected) return null
  const vue = vueDuPlan(repere, canvasWidth)
  if (!vue) return null
  const { nx, ny } = repereDimensions(repere)
  const adresse = adresseDansLeRepere(selected.col, selected.row, repere)
  if (adresse.col < 0 || adresse.col >= nx || adresse.row < 0 || adresse.row >= ny) return null
  const taille = repere.pasM * vue.scale
  return { x: adresse.col * taille, y: adresse.row * taille, size: taille }
}

/**
 * vueContain — la projection d'un repère dans un cadre à rapport IMPOSÉ : le monde est mis à
 * l'échelle pour TENIR EN ENTIER, centré, sans déformation.
 *
 * Sert aux VIGNETTES de la grille, dont le cadre est à hauteur fixe : au rapport du monde,
 * des cartes de rapports différents donneraient des vignettes de hauteurs différentes et la
 * grille perdrait ses lignes. Étirer le calque pour remplir aurait déformé la carte — une
 * zone chaude ronde y deviendrait ovale, et deux vignettes ne se compareraient plus.
 */
export function vueContain(
  repere: RepereTactique,
  canvasWidth: number,
  canvasHeight: number,
): { topLeftWorld: { x: number; y: number }; scale: number } | null {
  const largeur = repere.maxX - repere.minX
  const hauteur = repere.maxY - repere.minY
  if (!(largeur > 0) || !(hauteur > 0)) return null
  if (!(canvasWidth > 0) || !(canvasHeight > 0)) return null
  const scale = Math.min(canvasWidth / largeur, canvasHeight / hauteur)
  // Les cellules sont déjà réindexées en 0-based dans le repère : l'origine du calque est
  // donc le coin haut-gauche du cadre, décalé des marges de centrage.
  return {
    topLeftWorld: {
      x: (canvasWidth - largeur * scale) / 2,
      y: (canvasHeight - hauteur * scale) / 2,
    },
    scale,
  }
}
