/**
 * contract.ts — Le TYPE des textes de l'onglet « Emprise » : une seule déclaration, que `fr.ts`,
 * `en.ts` et toute langue ajoutée doivent satisfaire (`satisfies BaseEmpriseText`).
 */
import type { VehicleText } from '../vehicleStrings'

/**
 * Les noms d'une ressource, selon l'endroit où elle s'écrit : `label` (« Bonus » : légendes,
 * fiches), titres de ligne qui se lisent seuls (« Prises de bonus », « Frags avec arme
 * spéciale » — jamais le nom suivi d'un sous-libellé qui le complète), mot du pied de fiche
 * (« armes spéciales »).
 */
export interface ResourceText {
  label: string
  /** Piste du bilan : titre (« Prises de bonus ») et précision facultative (« camouflage, surbouclier »). */
  pisteTitle: string
  pisteSub?: string
  /** Synthèse de la grille match par match (« Prises de bonus »). */
  gridTitle: string
  footer: string
  /** Cases vides : synthèse (« Aucun bonus sur cette carte. »), objet (« : pas sur cette carte. », après son nom). */
  absent: string
  itemAbsent: string
  /** Ligne de « Frags par ressource » (« Frags avec arme spéciale »). */
  productionTitle: string
  /** Sous-libellé du « Rendement… » (« frags par prise »). */
  yieldSub: string
}

/** Une exposition (barre fine) : son nom (« temps d’effet ») et le format de sa valeur (« 2 min 39 »). */
export interface ExposureText { name: string; fmt: (v: number) => string }

/** Les textes avant l'ajout des véhicules (`vehicleStrings.ts`). */
export type BaseEmpriseText = Omit<EmpriseText, 'vehicles'>

export interface EmpriseText {
  vehicles: VehicleText // les mots propres aux véhicules ; leur entrée de ressource est posée par withVehicles
  sections: { bilan: string; roles: string; carte: string; prendre: string; habitude: string }
  resources: Record<string, ResourceText>
  ourSide: string
  opponent: string
  parity: string
  /** « 60 % », « 44,2 % » (une décimale au plus) ; `pctIntFmt` : « 17 % » (entier). */
  pctFmt: (v: number) => string
  pctIntFmt: (v: number) => string
  control: {
    title: string
    info: string
    ariaLabel: string
    segmentTip: (side: string, resource: string, sub: string | undefined, value: number, total: number, pct: string) => string
  }
  fil: {
    title: string
    info: string
    winLoss: string
    dominance: string
    pointTip: (v: { match: string; outcome: string | null; resource: string; us: number; them: number; pct: string; cumUs: number; cumTotal: number; cumPct: string }) => string
    endTip: (resource: string, cumUs: number, cumTotal: number, pct: string) => string
    bandTip: (match: string, result: string | null, dominance: string | null) => string
  }
  sheets: {
    title: string
    info: string
    dominant: string
    legendTaken: string
    legendLost: string
    lineTip: (player: string, item: string, n: number, camp: number, lost: string | null) => string
    /** « 1 perdue : gardé sans l’activer ». */
    lostFmt: (kept: number, dropped: number) => string | null
  }
  production: {
    title: string
    info: string
    ariaLabel: string
    thinLegend: string
    exposure: Record<string, ExposureText>
    segmentTip: (side: string, title: string, value: number, total: number, pct: string) => string
    thinTip: (side: string, name: string, value: string, pct: string) => string
    exposureLine: (name: string, value: string, pct: string) => string
  }
  yield: {
    title: string
    info: string
    ariaLabel: string
    more: string
    less: string
    /** Graduations de l'axe (−50 % à +50 %), écart signé (« +14 % »), rendements bruts (« 3,0 contre 2,7 »). */
    axis: [string, string, string, string, string]
    gapFmt: (gap: number) => string
    rawFmt: (us: number, them: number) => string
    tip: (resource: string, sub: string, us: number, them: number, gap: string) => string
  }
  habit: {
    title: string
    info: string
    tonight: string
    pointTip: (resource: string, evening: string, value: string, median: string | null) => string
    eveningOf: (date: string) => string
    medianTip: (resource: string, value: string) => string
    /** Infobulle d'une soirée hors comparaison (aucun mode de ce soir filmé) : sa raison, hors médiane. */
    notComparableTip: (families: string) => string
    /** Le bloc placeholder : aucune soirée, ce soir compris, n'a de part. */
    empty: { title: string; description: string }
  }
  grid: {
    title: string
    info: string
    more: string
    less: string
    nothing: string
    killsName: string
    killsAbsent: string
    /**
     * Le nombre de TYPES d'armes de râtelier listés sous la ligne repliable (« 16 types d'armes »),
     * jamais un nombre de prises. Source unique : la Vue match le reprend.
     */
    rackTypes: (n: number) => string
    matchHead: (time: string, map: string, mode: string, result: string | null) => string
    cellTip: (name: string, us: number, them: number, pct: string) => string
    whoFmt: (list: string) => string
    restLower: string
    padsFmt: (emptied: number, attributed: number) => string
    dominanceTip: (label: string) => string
  }
}
