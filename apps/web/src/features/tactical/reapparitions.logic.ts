/**
 * reapparitions.logic — la logique PURE de la lecture « Réapparitions adverses » : où l'adversaire
 * réapparaît quand l'équipe tient une zone, comparé à la répartition sans condition. Rien ne dépend de
 * React ni du DOM ; les composants rendent ce que ces fonctions décident (règle du dépôt).
 *
 * LA LECTURE COMPTE, ELLE NE PRÉVOIT RIEN : chaque part est celle de réapparitions observées. Les effectifs
 * et les planchers sont ceux du serveur, jamais recalculés : seul l'affichage (arrondi, teinte, écart en
 * points) se décide ici.
 *
 * TROIS ÉTATS, qui ne se disent pas pareil :
 *   - `sans_zone` : aucune zone tenue choisie. La réponse (une zone de sonde, cf. `volumeDeSonde`) ne sert
 *     que par sa référence sans condition, affichée en gris ;
 *   - `servie`   : la répartition pendant la tenue est lisible (planchers atteints) ;
 *   - `vide`     : elle ne l'est pas ; la raison dit pourquoi, les jauges disent combien il manque, et la
 *     référence reste affichée en gris quand elle est lisible.
 */
import type { TacticalPartDeZone, TacticalReapparitionsAdverses, TacticalRepartitionZones } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import { zoneHasShape, zoneRings } from '@/lib/replay/calloutShape'
import { calloutLabel, type CalloutZoneReady } from '@/lib/replay/calloutsPaint'
import type { MapFrame } from '@/lib/replay/heatPaint'

import type { TacticalText } from './i18n'
import {
  projectionDuPlan,
  repereAspect,
  type FenetreDuPlan,
  type RepereTactique,
  type TacticalQuestion,
} from './tacticalView.logic'

/** La lecture « Réapparitions adverses » dans la pilule « Lecture », à côté des sept lectures de placement. */
export const LECTURE_REAPPARITIONS = 'reapparitions_adverses' as const

/** Ce que la pilule « Lecture » peut porter. */
export type LectureDuPlan = TacticalQuestion | typeof LECTURE_REAPPARITIONS

/** Les raisons d'un état vide, en codes stables du serveur (`TacticalRaison*`). */
export type RaisonVide =
  | 'aucun_match_mesure'
  | 'equipe_inconnue'
  | 'zone_sans_forme'
  | 'sous_plancher_matchs'
  | 'sous_plancher_reapparitions'

const RAISONS: ReadonlySet<string> = new Set<RaisonVide>([
  'aucun_match_mesure',
  'equipe_inconnue',
  'zone_sans_forme',
  'sous_plancher_matchs',
  'sous_plancher_reapparitions',
])

/** Une raison inconnue (code ajouté côté serveur) se lit comme un manque de mesures : jamais un état muet. */
function raisonConnue(code: string | undefined): RaisonVide {
  return code !== undefined && RAISONS.has(code) ? (code as RaisonVide) : 'sous_plancher_reapparitions'
}

/** Part minimale (0..1) pour qu'une zone porte son nom et sa part sur le plan. */
export const SEUIL_ETIQUETTE = 0.06

/** Teinte d'une zone, en opacité de remplissage : `base + etendue x (part / maximum)`. */
const TEINTE_TENUE = { base: 0.07, etendue: 0.6 } as const
const TEINTE_GRISE = { base: 0.05, etendue: 0.35 } as const

/** Une zone dans la lecture : sa part pendant la tenue, sa part sans condition et l'écart entre les deux. */
export interface LigneDeZone {
  volumeIndex: number
  nom: string
  /** Part pendant la tenue (0..1) ; `null` quand la répartition « tenue » n'est pas lue. */
  tenue: number | null
  /** Réapparitions pendant la tenue ; 0 quand elle n'est pas lue. */
  tenueReapparitions: number
  /** Part sans condition (0..1) ; 0 quand la zone n'y figure pas. */
  reference: number
  /** Écart en points entre les deux parts ARRONDIES (celles qu'on lit à l'écran) ; `null` sans « tenue ». */
  ecart: number | null
}

export type EtatDeLaLecture = 'sans_zone' | 'servie' | 'vide'

/** Les effectifs d'une répartition. */
export interface EffectifsDeRepartition {
  matchs: number
  reapparitions: number
  suffisante: boolean
}

/** La lecture prête à rendre : le modèle de vue que décide `construireLaLecture`. */
export interface LectureReapparitions {
  etat: EtatDeLaLecture
  /** La raison de l'état vide ; `null` hors `vide`. */
  raison: RaisonVide | null
  /** La zone tenue choisie (indice de volume et nom) ; `null` sans zone. */
  zone: { volumeIndex: number; nom: string } | null
  /** Les zones à comparer : tenue triée d'abord quand elle est lue, sinon référence triée. */
  lignes: readonly LigneDeZone[]
  tenue: EffectifsDeRepartition
  reference: EffectifsDeRepartition
  /** Les planchers du serveur (matchs distincts, réapparitions) et le nombre de joueurs qui font tenir une zone. */
  plancher: { matchs: number; reapparitions: number; joueursPourTenir: number }
  /** La plus grande part « tenue » (0 si non lue) et la plus grande part de référence : bornes des teintes. */
  maxTenue: number
  maxReference: number
}

/** Une répartition vue comme ses effectifs. */
function effectifs(r: TacticalRepartitionZones): EffectifsDeRepartition {
  return { matchs: r.matchs, reapparitions: r.reapparitions, suffisante: r.suffisante }
}

/** Les parts d'une répartition par indice de volume (vide quand elle n'est pas servie). */
function partsParVolume(r: TacticalRepartitionZones): Map<number, TacticalPartDeZone> {
  return new Map((r.zones ?? []).map((z) => [z.volume_index, z]))
}

/** Écart en points entre deux parts : celles qui s'affichent (arrondies au point), jamais les brutes. */
export function ecartEnPoints(tenue: number, reference: number): number {
  return Math.round(tenue * 100) - Math.round(reference * 100)
}

/** Les lignes de zones : l'union des deux répartitions, triées sur la mesure qui s'affiche. */
function lignesDeZones(
  reference: TacticalRepartitionZones,
  tenue: TacticalRepartitionZones,
  tenueLue: boolean,
): LigneDeZone[] {
  const parRef = partsParVolume(reference)
  const parTenue = tenueLue ? partsParVolume(tenue) : new Map<number, TacticalPartDeZone>()
  const indices = new Set([...parRef.keys(), ...parTenue.keys()])
  const lignes = [...indices].map((volumeIndex): LigneDeZone => {
    const r = parRef.get(volumeIndex)
    const t = parTenue.get(volumeIndex)
    const partRef = r?.part ?? 0
    const partTenue = tenueLue ? (t?.part ?? 0) : null
    return {
      volumeIndex,
      nom: (t ?? r)?.nom ?? '',
      tenue: partTenue,
      tenueReapparitions: t?.reapparitions ?? 0,
      reference: partRef,
      ecart: partTenue === null ? null : ecartEnPoints(partTenue, partRef),
    }
  })
  const mesure = (l: LigneDeZone) => (tenueLue ? (l.tenue ?? 0) : l.reference)
  return lignes.sort((a, b) => mesure(b) - mesure(a) || a.volumeIndex - b.volumeIndex)
}

/** Le plus grand nombre d'une liste de parts (0 si vide). */
function maximum(parts: readonly number[]): number {
  return parts.reduce((max, p) => (p > max ? p : max), 0)
}

/** L'état de la lecture : `sans_zone` sans choix, `servie` quand le serveur la sert, `vide` sinon. */
function etatDe(rep: TacticalReapparitionsAdverses, choisie: number | null): EtatDeLaLecture {
  if (choisie === null) return 'sans_zone'
  return rep.etat === 'servie' ? 'servie' : 'vide'
}

/**
 * construireLaLecture — le modèle de vue de la lecture : l'état, les lignes de zones, les effectifs et
 * les bornes de teinte. `rep` à `undefined` (réponse pas encore là) = pas de lecture (`null`).
 *
 * `choisie` est la zone tenue CHOISIE par l'utilisateur, jamais la zone de sonde : la réponse d'une sonde
 * ne sert que par sa référence (l'état vaut alors `sans_zone`).
 */
export function construireLaLecture(
  rep: TacticalReapparitionsAdverses | undefined,
  choisie: number | null,
): LectureReapparitions | null {
  if (!rep) return null
  const etat = etatDe(rep, choisie)
  const tenueLue = etat === 'servie'
  const lignes = lignesDeZones(rep.reference, rep.tenue, tenueLue)
  return {
    etat,
    raison: etat === 'vide' ? raisonConnue(rep.raison) : null,
    zone: choisie === null ? null : { volumeIndex: choisie, nom: rep.zone.nom },
    lignes,
    tenue: effectifs(rep.tenue),
    reference: effectifs(rep.reference),
    plancher: {
      matchs: rep.plancher.matchs,
      reapparitions: rep.plancher.reapparitions,
      joueursPourTenir: rep.plancher.joueurs_pour_tenir,
    },
    maxTenue: tenueLue ? maximum(lignes.map((l) => l.tenue ?? 0)) : 0,
    maxReference: maximum(lignes.map((l) => l.reference)),
  }
}

/**
 * opaciteDeLaTeinte — l'opacité de remplissage d'une zone : proportionnelle à sa part rapportée à la plus
 * grande, avec un plancher qui laisse lire une zone presque vide. Teinte d'adversaire pendant la tenue,
 * teinte grise (plus discrète) pour la référence sans condition.
 */
export function opaciteDeLaTeinte(part: number, maximumDesParts: number, grise: boolean): number {
  const { base, etendue } = grise ? TEINTE_GRISE : TEINTE_TENUE
  const rapport = maximumDesParts > 0 ? Math.min(1, Math.max(0, part / maximumDesParts)) : 0
  return Math.round((base + etendue * rapport) * 1000) / 1000
}

/** Les zones qui portent leur nom et leur part sur le plan : celles dont la part « tenue » atteint le seuil. */
export function lignesEtiquetees(lecture: LectureReapparitions): readonly LigneDeZone[] {
  if (lecture.etat !== 'servie') return []
  return lecture.lignes.filter(
    (l) => (l.tenue ?? 0) >= SEUIL_ETIQUETTE && l.volumeIndex !== lecture.zone?.volumeIndex,
  )
}

/**
 * formaterLaPart — une part (0..1) en pourcentage entier dans la langue : « 12 % », et « <1 % » pour une
 * part non nulle sous le demi-point (jamais « 0 % » pour une zone qui compte des réapparitions).
 */
export function formaterLaPart(part: number, locale: Locale): string {
  const format = new Intl.NumberFormat(intlLocale(locale), { style: 'percent', maximumFractionDigits: 0 })
  if (part > 0 && part < 0.005) return `<${format.format(0.01)}`
  return format.format(Math.round(part * 100) / 100)
}

/** Le texte et le sens de la puce d'écart : ▲ n, ▼ n ou =. */
export function puceDEcart(ecart: number): { sens: 'haut' | 'bas' | 'egal'; texte: string } {
  if (ecart > 0) return { sens: 'haut', texte: `▲ ${ecart}` }
  if (ecart < 0) return { sens: 'bas', texte: `▼ ${Math.abs(ecart)}` }
  return { sens: 'egal', texte: '=' }
}

/** Une jauge d'effectif contre son plancher : la valeur, le plancher, la fraction remplie et si elle est atteinte. */
export function jaugeContrePlancher(
  valeur: number,
  plancher: number,
): { valeur: number; plancher: number; fraction: number; atteint: boolean } {
  const fraction = plancher > 0 ? Math.min(1, Math.max(0, valeur / plancher)) : 1
  return { valeur, plancher, fraction, atteint: valeur >= plancher }
}

// ─── Les zones nommées du catalogue : choix de la zone tenue, sonde, géométrie ─────────────────────────

/** Une zone qu'on peut tenir : une zone qui a une forme (la règle du jeu, `zoneHasShape`). */
function choisissable(z: CalloutZoneReady): boolean {
  return zoneHasShape(z)
}

/** Une zone proposée dans la pilule « Zone tenue » : son indice de volume et son nom. */
export interface ZoneProposee {
  volumeIndex: number
  nom: string
}

/**
 * zonesProposees — les zones qu'on peut choisir comme zone tenue : celles qui ont un contour et un nom
 * dans la langue de la page (repli du lexique compris), triées par nom puis par indice de volume.
 */
export function zonesProposees(zones: readonly CalloutZoneReady[], locale: Locale): ZoneProposee[] {
  const collateur = new Intl.Collator(intlLocale(locale), { sensitivity: 'base' })
  return zones
    .filter(choisissable)
    .map((z) => ({ volumeIndex: z.volumeIndex, nom: calloutLabel(z, locale) }))
    .filter((z) => z.nom !== '')
    .sort((a, b) => collateur.compare(a.nom, b.nom) || a.volumeIndex - b.volumeIndex)
}

/**
 * volumeDeSonde — l'indice de volume dont on se sert pour lire la référence sans condition tant qu'AUCUNE
 * zone n'est choisie : la référence ne dépend pas de la zone tenue. Le plus petit indice choisissable, ou
 * `null` quand la carte n'en a aucun (rien à lire).
 */
export function volumeDeSonde(zones: readonly CalloutZoneReady[]): number | null {
  const indices = zones.filter(choisissable).map((z) => z.volumeIndex)
  return indices.length > 0 ? Math.min(...indices) : null
}

/** Le chemin SVG d'une zone dans l'espace de la couche (mètres monde, Y inversé) : contour, parties, trous. */
export function cheminDeLaZone(zone: CalloutZoneReady): string {
  const nombre = (v: number) => Math.round(v * 100) / 100
  const anneau = (pts: readonly { x: number; y: number }[]) =>
    `M${pts.map((p) => `${nombre(p.x)} ${nombre(-p.y)}`).join('L')}Z`
  if (!choisissable(zone)) return ''
  return zoneRings(zone).map(anneau).join('')
}

/** Les bornes de la lecture de placement, telles que le serveur les publie. */
export interface BornesDeLaLecture {
  min_x: number
  max_x: number
  min_y: number
  max_y: number
  valide: boolean
}

/**
 * cadreDeLaCouche — le rectangle de la couche SVG, dans le même espace que `cheminDeLaZone` : celui du
 * cadre du fond (le calage de l'image) quand il existe, sinon celui des bornes de la lecture. `null` quand
 * aucun n'est exploitable.
 */
export function cadreDeLaCouche(
  fond: MapFrame | null,
  bornes: BornesDeLaLecture | undefined,
): { x: number; y: number; width: number; height: number } | null {
  if (fond && fond.widthM > 0 && fond.heightM > 0) {
    return { x: fond.originX, y: -fond.originY, width: fond.widthM, height: fond.heightM }
  }
  if (!bornes?.valide) return null
  const width = bornes.max_x - bornes.min_x
  const height = bornes.max_y - bornes.min_y
  if (!(width > 0) || !(height > 0)) return null
  return { x: bornes.min_x, y: -bornes.max_y, width, height }
}

/**
 * positionDeLEtiquette — où poser un nom de zone sur le cadre du plan, en pourcentages du cadre, pour un point
 * MONDE ; `null` quand il tombe hors de la fenêtre. Même projection que la chaleur (`projectionDuPlan`), la
 * rotation de la carte comprise.
 */
export function positionDeLEtiquette(
  point: { x: number; y: number },
  repere: RepereTactique,
  fenetre: FenetreDuPlan = repere,
): { left: string; top: string } | null {
  const projection = projectionDuPlan(repere, 1, fenetre)
  if (!projection) return null
  const p = projection(point)
  const haut = p.y * repereAspect(fenetre)
  if (p.x < 0 || p.x > 1 || haut < 0 || haut > 1) return null
  return { left: `${(p.x * 100).toFixed(2)}%`, top: `${(haut * 100).toFixed(2)}%` }
}

// ─── L'état vide : ce qu'il dit et ce qu'il mesure ────────────────────────────────────────────────────

/** Une jauge d'effectif de l'état vide : « 2 sur 3 matchs ». */
export interface JaugeDuBloc {
  cle: 'matchs' | 'reapparitions'
  valeur: number
  plancher: number
  fraction: number
  atteint: boolean
}

/** Le bloc pointillé de la colonne : un titre, une phrase, et les jauges quand un effectif manque. */
export interface BlocVide {
  titre: string
  corps: string
  jauges: readonly JaugeDuBloc[]
}

/** Les deux jauges (matchs, réapparitions) d'une répartition contre les planchers du serveur. */
function jaugesDe(effectif: EffectifsDeRepartition, plancher: LectureReapparitions['plancher']): JaugeDuBloc[] {
  return [
    { cle: 'matchs' as const, ...jaugeContrePlancher(effectif.matchs, plancher.matchs) },
    { cle: 'reapparitions' as const, ...jaugeContrePlancher(effectif.reapparitions, plancher.reapparitions) },
  ]
}

/** Le bloc d'un manque de mesures : le compte, les planchers et les deux jauges. */
function blocDeMesures(
  t: TacticalText,
  titre: string,
  effectif: EffectifsDeRepartition,
  plancher: LectureReapparitions['plancher'],
): BlocVide {
  const planchers = { floorMatchs: plancher.matchs, floorN: plancher.reapparitions }
  return {
    titre,
    corps: t.reappEmptyFloorBody(effectif.matchs, effectif.reapparitions, planchers),
    jauges: jaugesDe(effectif, plancher),
  }
}

/** Le bloc d'une raison de l'état vide qui n'est pas un manque de mesures : un titre et une phrase, sans jauge. */
function blocDeLaRaison(t: TacticalText, raison: RaisonVide): BlocVide | null {
  switch (raison) {
    case 'aucun_match_mesure':
      return { titre: t.reappEmptyNoMatchTitle, corps: t.reappEmptyNoMatchBody, jauges: [] }
    case 'equipe_inconnue':
      return { titre: t.reappEmptyTeamTitle, corps: t.reappEmptyTeamBody, jauges: [] }
    case 'zone_sans_forme':
      return { titre: t.reappEmptyShapeTitle, corps: t.reappEmptyShapeBody, jauges: [] }
    default:
      return null
  }
}

/**
 * blocVideDeLaLecture — le bloc pointillé de la colonne, ou `null` quand la lecture est servie :
 *   - état vide : la raison du serveur (manque de mesures avec jauges, ou cause nommée) ;
 *   - aucune zone choisie : la phrase qui dit comment la choisir, ou, si la référence elle-même est sous les
 *     planchers, le manque de mesures sur la carte.
 */
export function blocVideDeLaLecture(t: TacticalText, lecture: LectureReapparitions): BlocVide | null {
  if (lecture.etat === 'servie') return null
  if (lecture.etat === 'vide' && lecture.raison) {
    const nom = lecture.zone?.nom ?? ''
    return blocDeLaRaison(t, lecture.raison) ?? blocDeMesures(t, t.reappEmptyFloorTitle(nom), lecture.tenue, lecture.plancher)
  }
  if (!lecture.reference.suffisante) {
    return blocDeMesures(t, t.reappEmptyReferenceTitle, lecture.reference, lecture.plancher)
  }
  return { titre: t.reappEmptyNoneTitle, corps: t.reappEmptyNoneBody, jauges: [] }
}
