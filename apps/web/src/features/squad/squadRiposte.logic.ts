/**
 * squadRiposte.logic — les décisions PURES de la RIPOSTE de l'escouade.
 *
 * Une RIPOSTE est une mort de notre camp dont le tueur est abattu par un coéquipier dans
 * la fenêtre servie par le serveur (5 s). Vocabulaire arrêté le 2026-09-21 (D19) : plus
 * d'« échange », plus de « vengeance ».
 *
 * Aucune dépendance React/DOM : ce module dit CE QUI EST RENDU, les composants disent
 * COMMENT. Les règles produit qui vivent ici sont celles qu'un test doit pouvoir mettre en
 * défaut sans monter un arbre React :
 *
 *   1. l'écart à l'habituel SE TAIT quand le périmètre couvre tout l'historique ;
 *   2. les badges « le plus / le moins couvert » n'apparaissent qu'à ÉCART RÉEL ;
 *   3. un échantillon faible s'affiche AVEC sa réserve et ne classe personne ;
 *   4. la frise rend UNE barre par soirée — une seule soirée est déjà un graphe.
 */
import type { SquadEchange, SquadEchangeBucket, SquadEchangeCell } from '@/lib/api/types'
import { isFullHistoryScope } from '@/lib/baseline'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { ChartPointHistogram } from '@/components/charts/HistogramChart'

/**
 * Plancher d'échantillon : 30 morts d'équipe (plan tactique §1, décision produit
 * 2026-09-05). Miroir EXACT de `coordination.SeuilEchantillonFaible` côté Go — le
 * serveur pose déjà le drapeau `echantillon_faible`, cette constante ne sert qu'aux
 * règles CLIENTES qui doivent nommer le seuil (réserve).
 */
export const PLANCHER_MORTS = 30

/**
 * Écart minimal entre le plus et le moins couvert pour poser les deux badges :
 * 3 ripostes. Une riposte d'écart entre deux coéquipiers ne désigne personne.
 */
export const ECART_BADGE_RIPOSTES = 3

/** Fenêtre de la moyenne glissante de la frise : trois soirées. */
export const FENETRE_TENDANCE = 3

/** L'écart d'un périmètre à son habituel — la grandeur commune à la phrase et au chiffre. */
export interface EcartRiposte {
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
 * ecartRiposte calcule l'écart au taux habituel ET dit s'il doit se taire.
 *
 * IL VIT ICI ET PAS DANS LE COMPOSANT (correction W3, revue du 2026-09-06) : la
 * soustraction, son arrondi et la condition de masquage étaient inlinés, hors de portée de
 * tout test — supprimer le masquage ou inverser le signe passait sans qu'aucune assertion
 * ne bouge.
 */
export function ecartRiposte(echange: SquadEchange): EcartRiposte {
  const ecart = echange.couverture.taux - echange.habituel.taux
  return {
    ecart,
    ecartPoints: Math.round(ecart * 100),
    pleinHistorique: isFullHistoryScope(echange.matchs_total, echange.matchs_habituel),
  }
}

/** Le chiffre d'appel de la carte « Riposte » et les grandeurs de sa phrase de lecteur. */
export interface AppelRiposte {
  /** Taux de riposte du camp, unité 0..1. */
  taux: number
  /** Écart à l'habituel et son arrondi en points — voir `ecartRiposte`. */
  ecart: number
  ecartPoints: number
  /** Vrai quand l'écart doit se taire (périmètre = tout l'historique). */
  pleinHistorique: boolean
  /** Taux habituel, unité 0..1 — le terme de comparaison écrit à côté de l'écart. */
  habituel: number
  /** Délai médian des ripostes survenues, en SECONDES. `null` si aucune riposte mesurée. */
  delaiMedianS: number | null
  /** Ripostes survenues, morts restées sans réponse, et leur dénominateur. */
  ripostes: number
  sansReponse: number
  mortsEquipe: number
  /**
   * « Une fois sur N » de la phrase de lecteur — l'inverse ARRONDI du taux. `null` quand
   * le taux est nul : « une fois sur l'infini » ne se dit pas, la phrase change alors.
   */
  surCombien: number | null
  /** Réserve d'échantillon faible du périmètre (jamais un masquage : une mention). */
  echantillonFaible: boolean
}

/**
 * appelRiposte assemble le chiffre d'appel. Aucun quotient nouveau n'est inventé : le taux
 * et son brut viennent de `couverture` (mesurés côté Go), l'écart de `ecartRiposte`, et les
 * morts sans réponse sont la SOUSTRACTION du brut au dénominateur — pas un taux.
 */
export function appelRiposte(echange: SquadEchange): AppelRiposte {
  const { ecart, ecartPoints, pleinHistorique } = ecartRiposte(echange)
  const mortsEquipe = echange.couverture.n
  const ripostes = echange.couverture.brut
  const medianMs = echange.delai_median_ms ?? 0
  return {
    taux: echange.couverture.taux,
    ecart,
    ecartPoints,
    pleinHistorique,
    habituel: echange.habituel.taux,
    delaiMedianS: medianMs > 0 ? medianMs / 1000 : null,
    ripostes,
    sansReponse: Math.max(0, mortsEquipe - ripostes),
    mortsEquipe,
    surCombien: echange.couverture.taux > 0 ? Math.round(1 / echange.couverture.taux) : null,
    echantillonFaible: echange.couverture.echantillon_faible,
  }
}

/** Un joueur du roster et le nombre de fois où son camp a riposté pour lui. */
export interface CouvertureJoueur {
  xuid: string
  gamertag: string
  ripostes: number
}

/** Les deux extrêmes de la couverture, quand ils se distinguent vraiment. */
export interface ExtremesCouverture {
  plusCouvert: CouvertureJoueur
  moinsCouvert: CouvertureJoueur
}

/**
 * couvertureParJoueur compte, par joueur du roster, les ripostes REÇUES — la somme de sa
 * COLONNE dans la matrice (on a riposté pour lui). L'ordre est celui des axes.
 */
export function couvertureParJoueur(echange: SquadEchange): CouvertureJoueur[] {
  const cellules = echange.cellules ?? []
  return (echange.joueurs ?? []).map((j) => ({
    xuid: j.xuid,
    gamertag: j.gamertag,
    ripostes: cellules
      .filter((c) => c.venge_xuid === j.xuid)
      .reduce((total, c) => total + c.nombre, 0),
  }))
}

/**
 * extremesCouverture désigne le plus et le moins couvert — et rend `null` dès que
 * la désignation serait arbitraire.
 *
 * TROIS CONDITIONS, toutes nécessaires : au moins DEUX joueurs, le PLANCHER d'échantillon
 * atteint, et un écart d'au moins `ECART_BADGE_RIPOSTES` entre les deux extrêmes.
 */
export function extremesCouverture(
  echange: SquadEchange | null | undefined,
): ExtremesCouverture | null {
  if (!echange || echange.couverture.n < PLANCHER_MORTS) return null
  const parJoueur = couvertureParJoueur(echange)
  if (parJoueur.length < 2) return null

  const trie = [...parJoueur].sort((a, b) => b.ripostes - a.ripostes)
  const plusCouvert = trie[0]
  const moinsCouvert = trie[trie.length - 1]
  if (plusCouvert.ripostes - moinsCouvert.ripostes < ECART_BADGE_RIPOSTES) return null
  return { plusCouvert, moinsCouvert }
}

/**
 * matriceSeries construit LA série de la matrice.
 *
 * ORIENTATION : y = LIGNE = celui qui riposte, x = COLONNE = celui pour qui on riposte.
 * C'est celle du graphe d'appui (Assistant / Bénéficiaire), son voisin sur la page — deux
 * orientations opposées dans la même section se liraient à l'envers une fois sur deux.
 *
 * TOUTES LES CASES SONT ÉMISES, DIAGONALE COMPRISE, DANS L'ORDRE DU ROSTER. Le wrapper
 * DÉDUIT ses catégories d'axe de l'ordre d'apparition des points : sauter la diagonale
 * décalait l'axe X d'un cran par rapport à l'axe Y. Défaut mesuré et corrigé le 2026-09-06.
 *
 * La diagonale porte une valeur VIDE (`null`), que le wrapper ne peint ni n'étiquette :
 * personne ne riposte pour soi-même, et un « 0 » y suggérerait qu'il manque une mesure.
 */
export function matriceSeries(echange: SquadEchange): ChartSeries<ChartPointHeatmap>[] {
  const joueurs = echange.joueurs ?? []
  const parCouple = new Map<string, SquadEchangeCell>()
  for (const c of echange.cellules ?? []) {
    parCouple.set(`${c.vengeur_xuid}>${c.venge_xuid}`, c)
  }
  const datapoints: ChartPointHeatmap[] = []
  for (const auteur of joueurs) {
    for (const pour of joueurs) {
      if (auteur.xuid === pour.xuid) {
        datapoints.push({ x: pour.gamertag, y: auteur.gamertag, value: null })
        continue
      }
      const c = parCouple.get(`${auteur.xuid}>${pour.xuid}`)
      datapoints.push({
        x: pour.gamertag,
        y: auteur.gamertag,
        value: c?.nombre ?? 0,
        detail: { count: c?.nombre ?? 0, perMatch: c?.par_match ?? 0 },
      })
    }
  }
  return [{ key: 'riposte', datapoints }]
}

/** Vrai quand la matrice n'a aucune riposte à montrer (roster sans riposte interne). */
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

/** Une soirée de la frise : son libellé, son taux, son volume et son verdict. */
export interface SoireeRiposte {
  label: string
  /** Taux de riposte de la soirée, en POURCENTS (l'unité du seul axe Y de la frise). */
  tauxPct: number
  /** Morts mesurées ce soir-là — le DÉNOMINATEUR, rendu sous l'axe et en infobulle. */
  morts: number
  /** Vrai quand la soirée est au-dessus ou à l'habituel : la couleur porte ce verdict. */
  auDessus: boolean
  echantillonFaible: boolean
}

/** Tout ce que la frise peint : les soirées, la tendance glissante, et l'habituel. */
export interface FriseRiposte {
  soirees: SoireeRiposte[]
  /**
   * Moyenne glissante du taux sur `FENETRE_TENDANCE` soirées, en POURCENTS. Même longueur
   * que `soirees` : les premiers points moyennent ce qui existe déjà (aucun trou d'amorce,
   * qui se lirait comme une absence de mesure).
   */
  tendancePct: number[]
  /** Taux habituel, en POURCENTS — la ligne de repère tiretée. */
  habituelPct: number
}

/**
 * friseRiposte projette `taux_par_session` en la matière de la frise.
 *
 * AUCUN PLANCHER DE SOIRÉES (D19, 2026-09-21). L'ancienne carte se repliait en LISTE de
 * définitions sous trois soirées — et filtrer sur UNE soirée est l'usage nominal de la
 * page : le cas le plus fréquent rendait donc le pire objet. Une soirée = un bâton, lu face
 * à la règle d'habituel ; un bâton unique est déjà un graphe.
 */
export function friseRiposte(echange: SquadEchange): FriseRiposte {
  const habituelPct = echange.habituel.taux * 100
  const soirees = (echange.taux_par_session ?? []).map<SoireeRiposte>((p) => ({
    label: libelleCourtSession(p.session_label),
    tauxPct: p.couverture.taux * 100,
    morts: p.couverture.n,
    auDessus: p.couverture.taux >= echange.habituel.taux,
    echantillonFaible: p.couverture.echantillon_faible,
  }))
  return {
    soirees,
    tendancePct: moyenneGlissante(
      soirees.map((s) => s.tauxPct),
      FENETRE_TENDANCE,
    ),
    habituelPct,
  }
}

/**
 * moyenneGlissante rend une moyenne des `fenetre` dernières valeurs, ARRÊTÉE AU DÉBUT de
 * la série : le premier point vaut la première valeur, le deuxième la moyenne des deux.
 *
 * Pas de trou d'amorce (`null` sur les deux premiers points) : un trou au départ d'une
 * courbe se lit comme une mesure absente, alors que la moyenne d'une seule soirée est
 * parfaitement définie — c'est cette soirée.
 */
export function moyenneGlissante(valeurs: number[], fenetre: number): number[] {
  if (fenetre < 1) return [...valeurs]
  return valeurs.map((_, i) => {
    const debut = Math.max(0, i - fenetre + 1)
    const tranche = valeurs.slice(debut, i + 1)
    return tranche.reduce((a, b) => a + b, 0) / tranche.length
  })
}

/**
 * libelleCourtSession réduit un libellé de session à sa DATE.
 *
 * Le libellé complet (« 13/10/2025 22:27–22:46 (3) ») porte la plage horaire et le nombre
 * de matchs : posé sur quarante graduations d'axe, il se chevauche et devient illisible.
 * L'axe dit QUAND — la soirée —, pas le détail, qui reste dans l'infobulle du point.
 *
 * Un libellé sans espace est rendu tel quel : on ne coupe jamais à l'aveugle.
 */
export function libelleCourtSession(label: string): string {
  const espace = label.indexOf(' ')
  return espace > 0 ? label.slice(0, espace) : label
}
