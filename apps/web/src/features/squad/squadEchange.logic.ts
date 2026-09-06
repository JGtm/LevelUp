/**
 * squadEchange.logic — les décisions PURES de l'échange de l'escouade.
 *
 * Aucune dépendance React/DOM : ce module dit CE QUI EST RENDU, les composants
 * disent COMMENT. Les trois règles produit qui vivent ici sont celles qu'un test
 * doit pouvoir mettre en défaut sans monter un arbre React :
 *
 *   1. le CONSTAT DU MOMENT n'est rendu qu'au-dessus de DEUX seuils à la fois ;
 *   2. les badges « le plus / le moins couvert » n'apparaissent qu'à ÉCART RÉEL ;
 *   3. un échantillon faible s'affiche AVEC sa réserve et ne classe personne.
 */
import type { SquadEchange, SquadEchangeBucket, SquadEchangeCell } from '@/lib/api/types'
import { isFullHistoryScope } from '@/lib/baseline'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { ChartPointHistogram } from '@/components/charts/HistogramChart'
import type { KPITrend } from '@/components/layout/KPIStrip'

/**
 * Plancher d'échantillon : 30 morts d'équipe (plan tactique §1, décision produit
 * 2026-09-05). Miroir EXACT de `coordination.SeuilEchantillonFaible` côté Go — le
 * serveur pose déjà le drapeau `echantillon_faible`, cette constante ne sert qu'aux
 * règles CLIENTES qui doivent nommer le seuil (réserve, constat du moment).
 */
export const PLANCHER_MORTS = 30

/**
 * Écart minimal pour que le « constat du moment » ait quelque chose à dire : 5 points
 * (décision de l'utilisateur, 2026-09-06). En dessous, l'écart est du bruit de
 * tirage et la carte n'est PAS rendue — aucun état vide, aucun bruit.
 */
export const ECART_CONSTAT_POINTS = 5

/**
 * Écart minimal entre le plus et le moins couvert pour poser les deux badges :
 * 3 vengeances. Une vengeance d'écart entre deux coéquipiers ne désigne personne.
 */
export const ECART_BADGE_VENGEANCES = 3

/** L'écart d'un périmètre à son habituel — la grandeur commune au KPI et au cap. */
export interface EcartEchange {
  /** Écart brut en unité 0..1 (taux du périmètre moins taux de la référence). */
  ecart: number
  /** Le même, arrondi en POINTS entiers signés — la grandeur affichée. */
  ecartPoints: number
  /**
   * Vrai quand le périmètre couvre tout l'historique : l'écart est alors nul par
   * construction et NE DOIT PAS s'afficher (« ±0 pts vs habituel » ferait croire à
   * une mesure là où il n'y a qu'une tautologie).
   */
  pleinHistorique: boolean
}

/**
 * ecartEchange calcule l'écart au taux habituel ET dit s'il doit se taire.
 *
 * IL VIT ICI ET PAS DANS LE COMPOSANT (correction W3, revue du 2026-09-06) : la
 * soustraction, son arrondi et la condition de masquage étaient inlinés dans
 * `SquadEchangeKpi`, hors de portée de tout test — supprimer le masquage ou inverser
 * le signe passait sans qu'aucune assertion ne bouge. `constatDuMoment` consomme la même
 * fonction : deux surfaces qui affichent « l'écart à l'habituel » ne peuvent pas le
 * calculer chacune de leur côté.
 */
export function ecartEchange(echange: SquadEchange): EcartEchange {
  const ecart = echange.couverture.taux - echange.habituel.taux
  return {
    ecart,
    ecartPoints: Math.round(ecart * 100),
    pleinHistorique: isFullHistoryScope(echange.matchs_total, echange.matchs_habituel),
  }
}

/** Le constat du moment, quand il y a lieu de le rendre. */
export interface ConstatDuMoment {
  /** `consolide` : on échange PLUS que d'habitude. `attention` : moins. */
  ton: 'consolide' | 'attention'
  /** Écart au taux habituel, en POINTS entiers et signés (ex. −7). */
  ecartPoints: number
  /** Écart brut en unité 0..1, pour le formatage signé. */
  ecart: number
  /** Le taux du périmètre et celui de la référence, en unité 0..1. */
  taux: number
  habituel: number
  /** Le dénominateur (morts d'équipe) et le nombre de matchs — jamais un taux seul. */
  morts: number
  matchs: number
}

/**
 * constatDuMoment applique LA règle de seuil du plan (§1, arrêtée par l'utilisateur le
 * 2026-09-06) : la carte n'existe QUE si le périmètre porte au moins
 * PLANCHER_MORTS morts d'équipe ET que l'écart au taux habituel atteint
 * ECART_CONSTAT_POINTS points, dans un sens ou dans l'autre.
 *
 * `null` = la carte n'est pas rendue. Pas d'état vide, pas de « rien à signaler » :
 * une carte de cap qui s'affiche pour dire qu'elle n'a rien à dire est du bruit, et
 * sous 30 morts l'écart mesuré est du tirage.
 */
export function constatDuMoment(echange: SquadEchange | null | undefined): ConstatDuMoment | null {
  if (!echange) return null
  const { couverture, habituel } = echange
  if (couverture.n < PLANCHER_MORTS) return null
  // Sans référence mesurée, il n'y a pas d'écart : comparer à un taux calculé sur
  // zéro mort donnerait un « écart » qui n'est que l'absence de la référence.
  if (habituel.n <= 0) return null

  const { ecart, ecartPoints } = ecartEchange(echange)
  if (Math.abs(ecartPoints) < ECART_CONSTAT_POINTS) return null

  return {
    ton: ecartPoints > 0 ? 'consolide' : 'attention',
    ecartPoints,
    ecart,
    taux: couverture.taux,
    habituel: habituel.taux,
    morts: couverture.n,
    matchs: echange.matchs_total,
  }
}

/** Un joueur du roster et le nombre de fois où son camp l'a vengé. */
export interface CouvertureJoueur {
  xuid: string
  gamertag: string
  vengeances: number
}

/** Les deux extrêmes de la couverture, quand ils se distinguent vraiment. */
export interface ExtremesCouverture {
  plusCouvert: CouvertureJoueur
  moinsCouvert: CouvertureJoueur
}

/**
 * couvertureParJoueur compte, par joueur du roster, les vengeances REÇUES — la
 * somme de sa COLONNE dans la matrice (il est le vengé). L'ordre est celui des axes.
 */
export function couvertureParJoueur(echange: SquadEchange): CouvertureJoueur[] {
  const cellules = echange.cellules ?? []
  return (echange.joueurs ?? []).map((j) => ({
    xuid: j.xuid,
    gamertag: j.gamertag,
    vengeances: cellules
      .filter((c) => c.venge_xuid === j.xuid)
      .reduce((total, c) => total + c.nombre, 0),
  }))
}

/**
 * extremesCouverture désigne le plus et le moins couvert — et rend `null` dès que
 * la désignation serait arbitraire.
 *
 * TROIS CONDITIONS, toutes nécessaires :
 *
 *   - au moins DEUX joueurs (désigner « le plus couvert » d'un roster d'un seul
 *     joueur ne veut rien dire) ;
 *   - le PLANCHER d'échantillon atteint : sous 30 morts d'équipe, la mesure ne
 *     classe personne (c'est la même règle que la réserve « échantillon faible ») ;
 *   - un écart d'au moins ECART_BADGE_VENGEANCES entre les deux extrêmes.
 */
export function extremesCouverture(
  echange: SquadEchange | null | undefined,
): ExtremesCouverture | null {
  if (!echange || echange.couverture.n < PLANCHER_MORTS) return null
  const parJoueur = couvertureParJoueur(echange)
  if (parJoueur.length < 2) return null

  const trie = [...parJoueur].sort((a, b) => b.vengeances - a.vengeances)
  const plusCouvert = trie[0]
  const moinsCouvert = trie[trie.length - 1]
  if (plusCouvert.vengeances - moinsCouvert.vengeances < ECART_BADGE_VENGEANCES) return null
  return { plusCouvert, moinsCouvert }
}

/**
 * trendEcart traduit un écart signé en flèche de tendance. Le zéro est explicite
 * (`near`, glyphe « = ») : une flèche absente se lirait comme « pas de mesure ».
 */
export function trendEcart(ecartPoints: number): KPITrend {
  if (ecartPoints > 0) return 'above'
  if (ecartPoints < 0) return 'below'
  return 'near'
}

/**
 * matriceSeries construit LA série de la matrice.
 *
 * ORIENTATION : y = LIGNE = le VENGEUR, x = COLONNE = le VENGÉ. C'est celle de
 * `SquadAssistPairsTable` (Assistant / Bénéficiaire), son voisin immédiat sur la
 * page — deux orientations opposées dans la même colonne se liraient à l'envers une
 * fois sur deux.
 *
 * TOUTES LES CASES SONT ÉMISES, DIAGONALE COMPRISE, DANS L'ORDRE DU ROSTER. Le
 * wrapper DÉDUIT ses catégories d'axe de l'ordre d'apparition des points : sauter la
 * diagonale décalait l'axe X d'un cran par rapport à l'axe Y (roster [A,B,C,D] →
 * lignes A,B,C,D mais colonnes B,C,D,A), et sur un duo les deux axes sortaient
 * inversés. La matrice se lisait alors de travers, sans que rien ne le signale.
 * Défaut mesuré et corrigé le 2026-09-06 (revue ronde 1, W1).
 *
 * La diagonale porte donc une valeur VIDE (`null`), que le wrapper ne peint ni
 * n'étiquette : personne ne se venge soi-même, et un « 0 » y suggérerait qu'il manque
 * une mesure. Les cases hors diagonale sans échange, elles, valent bien 0 — c'est un
 * fait mesuré.
 */
export function matriceSeries(echange: SquadEchange): ChartSeries<ChartPointHeatmap>[] {
  const joueurs = echange.joueurs ?? []
  const parCouple = new Map<string, SquadEchangeCell>()
  for (const c of echange.cellules ?? []) {
    parCouple.set(`${c.vengeur_xuid}>${c.venge_xuid}`, c)
  }
  const datapoints: ChartPointHeatmap[] = []
  for (const vengeur of joueurs) {
    for (const venge of joueurs) {
      if (vengeur.xuid === venge.xuid) {
        datapoints.push({ x: venge.gamertag, y: vengeur.gamertag, value: null })
        continue
      }
      const c = parCouple.get(`${vengeur.xuid}>${venge.xuid}`)
      datapoints.push({
        x: venge.gamertag,
        y: vengeur.gamertag,
        value: c?.nombre ?? 0,
        detail: { count: c?.nombre ?? 0, perMatch: c?.par_match ?? 0 },
      })
    }
  }
  return [{ key: 'echange', datapoints }]
}

/** Vrai quand la matrice n'a aucune vengeance à montrer (roster sans échange interne). */
export function matriceVide(echange: SquadEchange): boolean {
  return (echange.cellules ?? []).length === 0
}

/** Un intervalle de délai, prêt à peindre : le compte, sa borne, et son statut. */
export interface IntervalleDelai {
  bucket: SquadEchangeBucket
  point: ChartPointHistogram
}

/**
 * delaisSeries transpose les intervalles PRÉ-BINNÉS par le serveur (ADR 0010) en
 * datapoints d'histogramme. Aucun re-binning ici : le client peint des barres, il
 * ne décide pas des bornes.
 *
 * Les bornes sont en SECONDES à l'affichage — un axe en millisecondes ferait lire
 * « 4000 » là où le joueur pense « 4 s ».
 */
export function delaisSeries(echange: SquadEchange): ChartSeries<ChartPointHistogram>[] {
  const datapoints = (echange.delais ?? []).map<ChartPointHistogram>((b) => ({
    binStart: b.debut_ms / 1000,
    binEnd: b.ouvert ? b.debut_ms / 1000 : b.fin_ms / 1000,
    count: b.nombre,
  }))
  return [{ key: 'delais', datapoints }]
}

/** Comptes de la ligne narrative des délais : dans la fenêtre, hors fenêtre, total. */
export interface ResumeDelais {
  dansLaFenetre: number
  horsFenetre: number
  total: number
}

/**
 * resumeDelais additionne les deux populations. `horsFenetre` est MONTRÉ et
 * n'entre dans aucun taux : c'est ce que la ligne narrative doit dire, sans quoi le
 * lecteur additionnerait les deux et se tromperait de dénominateur.
 */
export function resumeDelais(echange: SquadEchange): ResumeDelais {
  let dansLaFenetre = 0
  let horsFenetre = 0
  for (const b of echange.delais ?? []) {
    if (b.hors_fenetre) horsFenetre += b.nombre
    else dansLaFenetre += b.nombre
  }
  return { dansLaFenetre, horsFenetre, total: dansLaFenetre + horsFenetre }
}
