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

/** Délai d'une mort en SECONDES, ou `null` quand elle n'a jamais été vengée. */
export function delaiSecondes(mort: SquadIsolementMort): number | null {
  if (!mort.vengee || mort.delai_ms == null) return null
  return mort.delai_ms / 1000
}

/** Les deux échelles du nuage, calculées sur les morts publiées. */
export interface EchellesNuage {
  distance: EchelleBande
  delai: EchelleBande
}

export function echellesNuage(morts: SquadIsolementMort[]): EchellesNuage {
  const ratios = morts
    .map((m) => m.distance_ratio)
    .filter((r): r is number => r != null)
  const delais = morts.map(delaiSecondes).filter((d): d is number => d != null)
  return {
    distance: echelleAvecBande(ratios, 0.5, 2),
    delai: echelleAvecBande(delais, 1, 10),
  }
}

/** La position [x, y] d'une mort sur le nuage, bandes comprises. */
export function positionMort(mort: SquadIsolementMort, echelles: EchellesNuage): [number, number] {
  const x = mort.distance_ratio ?? echelles.distance.bandeCentre
  const delai = delaiSecondes(mort)
  return [x, delai ?? echelles.delai.bandeCentre]
}

/** La position [x, y] du GROS point d'un joueur — mêmes bandes que les petits points. */
export function positionRepere(
  repere: SquadIsolementRepere,
  echelles: EchellesNuage,
): [number, number] {
  const x = repere.mediane_distance_ratio ?? echelles.distance.bandeCentre
  const y =
    repere.mediane_delai_ms != null ? repere.mediane_delai_ms / 1000 : echelles.delai.bandeCentre
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

/** Opacité d'un petit point : toutes les morts restent visibles sans masquer les repères
 *  par joueur. */
export const OPACITE_MORT = 0.4

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
 * repereAttenue dit si le repère d'un joueur doit se peindre ATTÉNUÉ : son taux
 * d'isolement reste sous le plancher de confiance servi par le contrat.
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
