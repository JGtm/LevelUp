/**
 * squadIsolement.logic — les décisions PURES du nuage « Pourquoi la vengeance ne vient
 * pas » (`SquadEchange.nuage_isolement`, contrat refondu le 2026-09-19).
 *
 * Aucune dépendance React/ECharts : les deux échelles d'axe (distance rapportée à la
 * portée du radar, délai avant vengeance) avec leur BANDE de valeurs absentes, et la
 * taille du gros point par joueur. `SquadIsolementNuageCard` ne fait que peindre ce que
 * ce module décide.
 */
import type { SquadIsolementMort, SquadIsolementRepere } from '@/lib/api/types'

/**
 * Une échelle d'axe qui porte une BANDE pour les valeurs absentes.
 *
 * Une mort sans coéquipier visible n'a pas de distance, et une mort jamais vengée n'a pas
 * de délai : les poser à zéro (ou à l'extrême de l'axe) mentirait dans les deux cas. Elles
 * se posent donc dans une bande NOMMÉE, séparée du nuage mesuré par un intervalle vide.
 */
export interface EchelleBande {
  /** Haut de la zone MESURÉE — au-delà, plus aucune valeur réelle. */
  mesure: number
  /** Début de la bande des valeurs absentes. */
  bandeDebut: number
  /** Position où se posent les points de la bande (son centre). */
  bandeCentre: number
  /** Haut de l'axe. */
  max: number
}

/**
 * echelleAvecBande projette les valeurs observées sur un axe, puis réserve une bande à son
 * extrémité pour les valeurs ABSENTES.
 *
 * `pas` arrondit le haut de la zone mesurée (0,5 pour un ratio de distance, 1 s pour un
 * délai) ; `minimum` garde une échelle lisible quand le nuage est très tassé.
 */
export function echelleAvecBande(valeurs: number[], pas: number, minimum: number): EchelleBande {
  const observe = valeurs.length > 0 ? Math.max(...valeurs) : 0
  const mesure = Math.max(minimum, Math.ceil(observe / pas) * pas)
  const bandeDebut = mesure + pas
  const max = bandeDebut + 2 * pas
  return { mesure, bandeDebut, bandeCentre: (bandeDebut + max) / 2, max }
}

/**
 * Une échelle LOGARITHMIQUE qui porte, elle aussi, une bande pour les valeurs absentes.
 *
 * POURQUOI LE LOG (décision utilisateur du 2026-09-22) : l'axe des délais court de la
 * fraction de seconde à la minute. En linéaire, tout le nuage utile — les ripostes sous la
 * fenêtre de 5 s — s'écrase sur le douzième bas de la hauteur et devient illisible, pendant
 * que les quelques chutes tardives occupent tout le reste.
 *
 * En log, une bande ne peut plus être « `max + 2 × pas` » : elle se réserve par un FACTEUR,
 * et son centre est la moyenne GÉOMÉTRIQUE de ses bornes — c'est le milieu visuel une fois
 * l'axe projeté.
 */
export interface EchelleLog {
  /** Bas de l'axe — le PLANCHER d'affichage : un log n'atteint jamais zéro. */
  min: number
  /** Haut de la zone MESURÉE — le plafond du contrat (`plafond_ms`). */
  mesure: number
  /** Début de la bande des valeurs absentes. */
  bandeDebut: number
  /** Position où se posent les points de la bande (son centre GÉOMÉTRIQUE). */
  bandeCentre: number
  /** Haut de l'axe. */
  max: number
}

/**
 * PLANCHER D'AFFICHAGE de l'axe des délais, en secondes.
 *
 * Un axe logarithmique n'a pas de zéro, et une riposte instantanée (0 ms sur l'horloge du
 * match, ou 90 ms) n'a nulle part où se poser. Elle est donc POSÉE ici — POUR LE DESSIN
 * SEULEMENT : `delaiSecondes` et l'infobulle gardent la vraie valeur.
 */
export const PLANCHER_DELAI_S = 0.2

/**
 * FACTEUR de réservation de la bande haute sur un axe log — l'équivalent multiplicatif du
 * `+ pas` de `echelleAvecBande`. 1,4 laisse la bande lisible (environ un dixième de la
 * hauteur) sans voler de place au nuage mesuré.
 */
const FACTEUR_BANDE_LOG = 1.4

/**
 * echelleLogAvecBande réserve la bande des valeurs absentes AU-DESSUS du plafond mesuré.
 *
 * `mesure` est le plafond DU CONTRAT (`plafond_ms`), jamais le maximum observé : au-delà,
 * le serveur ne publie plus d'état de riposte, donc l'axe n'a aucune raison de monter plus
 * haut — et la bande « jamais ripostée » reste dans l'espace réservé, au-dessus.
 */
export function echelleLogAvecBande(min: number, mesure: number): EchelleLog {
  const bandeDebut = mesure * FACTEUR_BANDE_LOG
  const max = bandeDebut * FACTEUR_BANDE_LOG * FACTEUR_BANDE_LOG
  return { min, mesure, bandeDebut, bandeCentre: Math.sqrt(bandeDebut * max), max }
}

/**
 * GRADUATIONS LISIBLES de l'axe des délais, en secondes. Un axe log laissé à lui-même ne
 * gradue qu'aux puissances de dix (0,2 · 1 · 10 · 100) : trois repères pour toute la
 * hauteur, dont un hors plafond. Cette suite nomme les durées qu'un lecteur reconnaît.
 */
const GRADUATIONS_DELAI_S = [0.2, 0.5, 1, 2, 5, 10, 30, 60]

/** Les graduations qui tombent dans la zone mesurée de l'échelle — la bande n'en porte
 *  aucune, elle est nommée par son étiquette. */
export function graduationsDelai(echelle: EchelleLog): number[] {
  return GRADUATIONS_DELAI_S.filter((g) => g >= echelle.min && g <= echelle.mesure)
}

/**
 * L'ÉTAT d'une mort du nuage — trois valeurs exclusives, décidées par le SERVEUR (règle des
 * 5 s partout et PLAFOND de 60 s, décisions du 2026-09-22) et jamais recalculées ici :
 *
 *   `ripostee`     le tueur est tombé DANS la fenêtre (`fenetre_ms`) — la seule vraie
 *                  riposte, la même que la carte « Riposte » et le taux d'échange ;
 *   `horsFenetre`  il est tombé APRÈS la fenêtre mais avant le plafond (`plafond_ms`) : le
 *                  délai existe, le point reste visible, mais ce n'est pas une riposte ;
 *   `jamais`       aucune riposte connue — ou une chute au-delà du plafond, que le serveur
 *                  ne publie plus : pas de délai, bande haute.
 */
export type EtatMort = 'ripostee' | 'horsFenetre' | 'jamais'

export function etatMort(mort: SquadIsolementMort): EtatMort {
  if (mort.vengee) return 'ripostee'
  if (mort.hors_fenetre && mort.delai_ms != null) return 'horsFenetre'
  return 'jamais'
}

/**
 * Délai d'une mort en SECONDES, ou `null` quand aucune riposte n'est connue.
 *
 * LES MORTS HORS FENÊTRE EN ONT UN : c'est toute la demande du 2026-09-22 — un tueur tombé
 * à 60 s doit se VOIR à 60 s sur l'axe, et non se tasser dans la bande « jamais ripostée »
 * où il ne dirait plus à quelle distance de la fenêtre la riposte est passée. C'est
 * `etatMort` qui dit si ce délai compte comme une riposte, jamais sa seule présence.
 */
export function delaiSecondes(mort: SquadIsolementMort): number | null {
  if (mort.delai_ms == null) return null
  if (!mort.vengee && !mort.hors_fenetre) return null
  return mort.delai_ms / 1000
}

/** La fenêtre de riposte du contrat, en SECONDES — le repère horizontal du graphe. */
export function fenetreSecondes(nuage: { fenetre_ms: number }): number {
  return nuage.fenetre_ms / 1000
}

/** Le plafond du contrat, en SECONDES — le haut de la zone mesurée de l'axe des délais.
 *  Jamais un 60 codé en dur : la règle du jeu vient du serveur (`plafond_ms`). */
export function plafondSecondes(nuage: { plafond_ms: number }): number {
  return nuage.plafond_ms / 1000
}

/**
 * Les deux échelles du nuage.
 *
 * L'AXE DES DÉLAIS EST LOGARITHMIQUE ET BORNÉ PAR LE CONTRAT (décision utilisateur du
 * 2026-09-22). Il court du plancher d'affichage (`PLANCHER_DELAI_S`) au plafond
 * (`plafond_ms`), et pas plus haut : au-delà, le serveur ne publie plus de délai du tout.
 * Le log rend enfin lisible la zone qui porte le sens — les ripostes sous la fenêtre de
 * 5 s — au lieu de l'écraser au ras de l'axe.
 *
 * L'axe des DISTANCES, lui, reste linéaire et libre : un ratio de portée de radar se lit en
 * multiples, pas en décades.
 */
export interface EchellesNuage {
  distance: EchelleBande
  delai: EchelleLog
}

export function echellesNuage(morts: SquadIsolementMort[], plafondS: number): EchellesNuage {
  const ratios = morts
    .map((m) => m.distance_ratio)
    .filter((r): r is number => r != null)
  return {
    distance: echelleAvecBande(ratios, 0.5, 2),
    delai: echelleLogAvecBande(PLANCHER_DELAI_S, plafondS),
  }
}

/**
 * ORDONNÉE DE DESSIN d'un délai : le délai lui-même, retenu entre le plancher et le
 * plafond de l'échelle.
 *
 * LE PLANCHER EST UN GESTE DE DESSIN, PAS UNE MESURE : une riposte à 90 ms n'a pas de place
 * sur un axe log et se pose à 0,2 s, mais `delaiSecondes` et l'infobulle gardent la vraie
 * valeur. Le plafond ne devrait jamais mordre (le serveur ne publie pas de délai au-delà) —
 * c'est une ceinture, pour qu'aucun point ne puisse se poser DANS la bande réservée.
 */
function ordonneeDelai(secondes: number, echelle: EchelleLog): number {
  return Math.min(Math.max(secondes, echelle.min), echelle.mesure)
}

/** La position [x, y] d'une mort sur le nuage, bandes comprises. */
export function positionMort(mort: SquadIsolementMort, echelles: EchellesNuage): [number, number] {
  const x = mort.distance_ratio ?? echelles.distance.bandeCentre
  const delai = delaiSecondes(mort)
  return [x, delai != null ? ordonneeDelai(delai, echelles.delai) : echelles.delai.bandeCentre]
}

/** La position [x, y] du GROS point d'un joueur — mêmes bandes que les petits points. */
export function positionRepere(
  repere: SquadIsolementRepere,
  echelles: EchellesNuage,
): [number, number] {
  const x = repere.mediane_distance_ratio ?? echelles.distance.bandeCentre
  const y =
    repere.mediane_delai_ms != null
      ? ordonneeDelai(repere.mediane_delai_ms / 1000, echelles.delai)
      : echelles.delai.bandeCentre
  return [x, y]
}

/**
 * ABSCISSE DE RÉFÉRENCE : 1,0 = la portée du radar du match. C'est la SEULE graduation qui
 * porte du sens sur cet axe — au-delà, un coéquipier ne peut pas savoir que vous êtes en
 * difficulté.
 */
export const REPERE_PORTEE_RADAR = 1

/** Diamètre (px) d'un petit point (une mort), le même pour tous : les morts disent la
 *  DISPERSION, jamais le volume — un seul encodage de taille par graphe. */
export const TAILLE_MORT = 5

// AUCUNE CONSTANTE D'OPACITÉ ICI (décision utilisateur du 2026-09-22) : les points, les
// repères médians et les bandes se peignent à ENCRE PLEINE. `OPACITE_MORT` (0,4),
// `OPACITE_MORT_HORS_FENETRE` (0,75) et `CONTOUR_MORT_HORS_FENETRE` (1,2 px) ont été
// retirées avec le point creux qu'elles servaient — une mort hors fenêtre se distingue
// désormais par sa FORME (losange, `SYMBOLE_MORT_HORS_FENETRE`), et la hiérarchie petit
// point / gros repère par la TAILLE et l'ordre de superposition.

const TAILLE_REPERE_MIN = 18
const TAILLE_REPERE_MAX = 46

/**
 * tailleRepere rend le diamètre (px) du gros point d'un joueur en projetant son NOMBRE DE
 * MORTS sur la plage réelle du roster (`min`..`max`), et non sur une constante arbitraire :
 * projetée sur les extrêmes observés, la différence entre joueurs reste lisible quel que
 * soit le volume de la sélection.
 *
 * `min === max` (un seul joueur, ou volumes égaux) → la taille médiane de la plage : il n'y
 * a pas d'écart à montrer.
 */
export function tailleRepere(nbMorts: number, min: number, max: number): number {
  if (max <= min) return (TAILLE_REPERE_MIN + TAILLE_REPERE_MAX) / 2
  const t = Math.min(1, Math.max(0, (nbMorts - min) / (max - min)))
  return TAILLE_REPERE_MIN + t * (TAILLE_REPERE_MAX - TAILLE_REPERE_MIN)
}

/**
 * repereAttenue dit si le repère d'un joueur porte une RÉSERVE : son taux d'isolement reste
 * sous le plancher de confiance servi par le contrat.
 *
 * Le nom dit « atténué » pour des raisons d'histoire ; depuis le 2026-09-22 la réserve ne
 * se peint plus par une opacité mais par une BORDURE TIRETÉE sur un point plein (et par une
 * note dans l'infobulle).
 */
export function repereAttenue(repere: SquadIsolementRepere): boolean {
  return repere.part_isolee.echantillon_faible
}

/**
 * ordreDessinReperes trie les repères médians par TAILLE DÉCROISSANTE — c'est-à-dire par
 * nombre de morts décroissant, la grandeur qui décide du diamètre (`tailleRepere`).
 *
 * POURQUOI CET ORDRE, et pas celui du roster (retour utilisateur du 2026-09-21) : ECharts
 * dessine les séries dans l'ordre du tableau `series`, la dernière AU-DESSUS. Les repères
 * étant émis dans l'ordre du roster, un gros point pouvait recouvrir entièrement le petit
 * point d'un autre joueur — le joueur au plus petit échantillon DISPARAISSAIT de la figure,
 * sans que rien ne le signale. En posant le plus gros EN PREMIER et le plus petit EN
 * DERNIER, le petit reste au premier plan et se lit toujours.
 *
 * Tri STABLE (`sort` l'est depuis ES2019) : à volume égal, l'ordre du roster est conservé.
 */
export function ordreDessinReperes(reperes: SquadIsolementRepere[]): SquadIsolementRepere[] {
  return [...reperes].sort((a, b) => b.nb_morts - a.nb_morts)
}
