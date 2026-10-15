/**
 * mort.logic — la logique PURE de la lecture des coéquipiers à la mort : le bilan de la zone (barre
 * des quatre statuts, rose, phrases), la fiche d'UNE mort (tueur, coéquipiers, distances, échange),
 * et les SCÈNES que le plan peint (nuage de la zone, instantané d'une mort, étape de la séquence).
 *
 * LES RÈGLES DE PORTÉE NE SONT PAS ICI : le serveur publie `statut` et `dans_le_rayon` (borne
 * inclusive, `coordination.APortee`) ; ce fichier les LIT et ne compare jamais une distance à un
 * rayon. Les mesures affichées (distance entre deux points, plus proche) sont des lectures, pas des
 * verdicts. Aucune donnée absente n'est comblée : un point manquant n'est pas peint, jamais (0, 0).
 */
import type {
  TacticalBilanMorts,
  TacticalContribution,
  TacticalCoequipier,
  TacticalSequence,
} from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { distanceEnMetres } from './tueurs.logic'
import { armeDe, decimale } from './zone.logic'

/** Un point du plan, en mètres monde. */
export interface Pt {
  x: number
  y: number
}

// ─── Le bilan de la zone (vue C) ──────────────────────────────────────────────────────────────

/** Les quatre statuts d'une mort, dans l'ordre de la barre (du plus accompagné au plus seul). */
export const ORDRE_DES_STATUTS = ['accompagnee', 'visible_hors_rayon', 'aucun_visible', 'equipe_a_terre'] as const
export type StatutDeMort = (typeof ORDRE_DES_STATUTS)[number]

export interface PartDeBarre {
  statut: StatutDeMort
  n: number
  /** Part des morts examinées, de 0 à 1 ; 0 sans mort examinée. */
  part: number
}

const COMPTE_PAR_STATUT: Record<StatutDeMort, (b: TacticalBilanMorts) => number> = {
  accompagnee: (b) => b.accompagnees,
  visible_hors_rayon: (b) => b.visibles_hors_rayon,
  aucun_visible: (b) => b.aucun_visible,
  equipe_a_terre: (b) => b.equipe_a_terre,
}

/** partsDeBarre — les quatre segments de la barre, comptes du serveur, parts sur les morts examinées. */
export function partsDeBarre(b: TacticalBilanMorts): PartDeBarre[] {
  return ORDRE_DES_STATUTS.map((statut) => {
    const n = COMPTE_PAR_STATUT[statut](b)
    return { statut, n, part: b.examinees > 0 ? n / b.examinees : 0 }
  })
}

/** Un bâton de la rose : le secteur, son compte, sa longueur relative (0..1) et son angle depuis le haut. */
export interface BatonDeRose {
  secteur: number
  n: number
  longueur: number
  angleDeg: number
}

/** batonsDeLaRose — un bâton par secteur non vide, le plus long ramené à 1. Secteur k centré sur 45 k degrés. */
export function batonsDeLaRose(rose: readonly number[]): BatonDeRose[] {
  const max = Math.max(0, ...rose)
  if (max === 0) return []
  const out: BatonDeRose[] = []
  rose.forEach((n, secteur) => {
    if (n > 0) out.push({ secteur, n, longueur: n / max, angleDeg: secteur * 45 })
  })
  return out
}

/**
 * directionDominante — le secteur le plus fourni de la rose et son compte : « 9 sur 14 par la droite ».
 * `null` sans tueur rangé. À égalité, le plus petit index (l'ordre de la rose) : le résultat ne dépend
 * pas de l'ordre de parcours.
 */
export function directionDominante(rose: readonly number[]): { n: number; secteur: number } | null {
  const max = Math.max(0, ...rose)
  if (max === 0) return null
  return { n: max, secteur: rose.indexOf(max) }
}

/** Le total des tueurs rangés dans la rose. */
export function totalDeLaRose(rose: readonly number[]): number {
  return rose.reduce((s, n) => s + n, 0)
}

/** metresEntiers — un nombre de mètres au mètre. */
export function metresEntiers(v: number, locale: Locale): string {
  return new Intl.NumberFormat(intlLocale(locale), { maximumFractionDigits: 0 }).format(Math.round(v))
}

/** secondesDeTraine — la durée de la traîne en secondes, sans décimale inutile (« 4 »). */export function secondesDeTraine(ms: number, locale: Locale): string {  return new Intl.NumberFormat(intlLocale(locale), { maximumFractionDigits: 1 }).format(ms / 1000)}
/** phraseDeLaRose — « 9 sur 14 par la droite » (le secteur dominant), ou `null` sans tueur rangé. */
export function phraseDeLaRose(t: TacticalText, b: TacticalBilanMorts): string | null {
  const rose = b.rose ?? []
  const d = directionDominante(rose)
  if (!d) return null
  return t.mortRoseFrom(d.n, totalDeLaRose(rose), t.mortDirections[d.secteur])
}

// ─── La fiche d'une mort (vues A et B) ────────────────────────────────────────────────────────

/** Une ligne de la fiche : le tueur ou un coéquipier, sa marque et ses deux distances. */
export interface LigneDeFiche {
  cle: string
  role: 'tueur' | 'coequipier'
  /** Le nom, ou la phrase complète d'un coéquipier en attente ou hors de vue (sans marque ni distance). */
  texte: string
  /** `plein` dans le rayon, `creux` au-delà, `aucune` hors de vue / en attente / tueur. */
  marque: 'plein' | 'creux' | 'aucune'
  distanceMort: string | null
  distanceTueur: string | null
}

const RANG_DE_STATUT: Record<string, number> = { visible: 0, out_of_sight: 1, waiting: 2 }

function ligneDeCoequipier(t: TacticalText, m: TacticalCoequipier, locale: Locale): LigneDeFiche {
  const nom = m.gamertag || t.mortDetailUnnamed
  if (m.statut === 'waiting') return { cle: m.xuid, role: 'coequipier', texte: t.mortDetailWaiting(nom), marque: 'aucune', distanceMort: null, distanceTueur: null }
  if (m.statut !== 'visible') return { cle: m.xuid, role: 'coequipier', texte: t.mortDetailOutOfSight(nom), marque: 'aucune', distanceMort: null, distanceTueur: null }
  return {
    cle: m.xuid,
    role: 'coequipier',
    texte: nom,
    marque: m.dans_le_rayon ? 'plein' : 'creux',
    distanceMort: m.distance_mort_m == null ? null : metresEntiers(m.distance_mort_m, locale),
    distanceTueur: m.distance_tueur_m == null ? null : metresEntiers(m.distance_tueur_m, locale),
  }
}

/** lignesDeLaFiche — le tueur d'abord, puis les coéquipiers : visibles du plus proche au plus loin, hors de vue, en attente. */
export function lignesDeLaFiche(t: TacticalText, c: TacticalContribution, locale: Locale): LigneDeFiche[] {
  const lignes: LigneDeFiche[] = []
  if (c.autre_gamertag) {
    const arme = armeDe(t, c, locale)
    lignes.push({
      cle: 'tueur',
      role: 'tueur',
      texte: arme ? `${c.autre_gamertag} · ${arme}` : c.autre_gamertag,
      marque: 'aucune',
      distanceMort: distanceEnMetres(c) === null ? null : metresEntiers(distanceEnMetres(c) as number, locale),
      distanceTueur: null,
    })
  }
  const coequipiers = [...(c.equipe?.coequipiers ?? [])].sort(
    (a, b) =>
      (RANG_DE_STATUT[a.statut] ?? 3) - (RANG_DE_STATUT[b.statut] ?? 3) ||
      (a.distance_mort_m ?? Infinity) - (b.distance_mort_m ?? Infinity),
  )
  for (const m of coequipiers) lignes.push(ligneDeCoequipier(t, m, locale))
  return lignes
}

/** resumeDeLEquipe — « 1 coéquipier dans le rayon de 18 m, 1 au-delà, 1 hors jeu. » ; `null` sans équipe lue. */
export function resumeDeLEquipe(t: TacticalText, c: TacticalContribution, locale: Locale): string | null {
  const eq = c.equipe
  if (!eq) return null
  const liste = eq.coequipiers ?? []
  if (eq.rayon_m == null) return t.mortDetailSummaryNoRadius(liste.length)
  const dedans = liste.filter((m) => m.statut === 'visible' && m.dans_le_rayon).length
  const audela = liste.filter((m) => m.statut === 'visible' && !m.dans_le_rayon).length
  return t.mortDetailSummary(dedans, audela, liste.length - dedans - audela, metresEntiers(eq.rayon_m, locale))
}

// ─── Les scènes du plan ───────────────────────────────────────────────────────────────────────

/** Le rôle d'une marque : fixe sa forme et son jeton de couleur. */
export type RoleDeMarque = 'mort' | 'tueur' | 'coequipier' | 'nuage'

export interface MarqueDeScene {
  role: RoleDeMarque
  pt: Pt
  /** Coéquipier plein (dans le rayon) ou creux (au-delà). Sans objet pour les autres rôles. */
  plein: boolean
}

/** Une traîne : des segments continus (une lacune du film coupe la traîne), du plus ancien au plus récent. */
export interface TraineDeScene {
  role: 'tueur' | 'coequipier' | 'mort'
  segments: Pt[][]
}

/** Ce que le plan peint pour une vue de la lecture des coéquipiers. */
export interface SceneDuPlan {
  /** Le cercle du rayon du radar, en mètres monde. */
  cercle: { centre: Pt; rayonM: number } | null
  traines: TraineDeScene[]
  marques: MarqueDeScene[]
}

export const SCENE_VIDE: SceneDuPlan = { cercle: null, traines: [], marques: [] }

/** sceneDeZone — le nuage : chaque coéquipier VISIBLE à chaque mort de la zone, sans rien d'autre. */
export function sceneDeZone(contributions: readonly TacticalContribution[]): SceneDuPlan {
  const marques: MarqueDeScene[] = []
  for (const c of contributions) {
    if (c.face !== 'mort') continue
    for (const m of c.equipe?.coequipiers ?? []) {
      if (m.statut === 'visible' && m.point) marques.push({ role: 'nuage', pt: m.point, plein: true })
    }
  }
  return { ...SCENE_VIDE, marques }
}

/**
 * sceneDeMort — l'instantané d'UNE mort (vue A) : le cercle du rayon autour du lieu de la mort, le
 * mort et ses coéquipiers visibles (pleins dans le rayon, creux au-delà). Le tueur est peint par la
 * couche des tueurs.
 */
export function sceneDeMort(c: TacticalContribution): SceneDuPlan {
  if (c.face !== 'mort' || !c.point) return SCENE_VIDE
  const marques: MarqueDeScene[] = [{ role: 'mort', pt: c.point, plein: true }]
  for (const m of c.equipe?.coequipiers ?? []) {
    if (m.statut === 'visible' && m.point) marques.push({ role: 'coequipier', pt: m.point, plein: m.dans_le_rayon })
  }
  const rayon = c.equipe?.rayon_m
  return { cercle: rayon == null ? null : { centre: c.point, rayonM: rayon }, traines: [], marques }
}

type Acteur = NonNullable<TacticalSequence['acteurs']>[number]

/** La position d'un acteur de la séquence à un décalage donné. */
function positionDe(a: Acteur, offsetMs: number): Pt | null {
  const p = (a.points ?? []).find((q) => q.offset_ms === offsetMs)
  return p ? { x: p.x, y: p.y } : null
}

/** La traîne d'un acteur à `offsetMs` : ses échantillons des `traineMs` précédentes, coupés aux lacunes. */
function traineDe(a: Acteur, offsetMs: number, traineMs: number): Pt[][] {
  const segments: Pt[][] = []
  let courant: Pt[] = []
  for (const p of a.points ?? []) {
    if (p.offset_ms > offsetMs || p.offset_ms < offsetMs - traineMs) continue
    if (p.coupure && courant.length > 0) {
      segments.push(courant)
      courant = []
    }
    courant.push({ x: p.x, y: p.y })
  }
  if (courant.length > 0) segments.push(courant)
  return segments.filter((s) => s.length > 1)
}

/**
 * sceneDeSequence — l'étape `etape` (index de `seq.etapes`) de la séquence : le cercle à son centre
 * (la victime avant la mort, le lieu de la mort ensuite), les traînes de 4 s, et chaque acteur à sa
 * position de l'instant. Le mort reste au lieu de sa mort à partir de l'instant 0 ; un acteur sans
 * position à cet instant n'est pas peint.
 */
export function sceneDeSequence(seq: TacticalSequence, etape: number): SceneDuPlan {
  const e = seq.etapes?.[etape]
  if (!e) return SCENE_VIDE
  const marques: MarqueDeScene[] = []
  const traines: TraineDeScene[] = []
  for (const a of seq.acteurs ?? []) {
    const role = a.role === 'victime' ? 'mort' : a.role === 'tueur' ? 'tueur' : 'coequipier'
    const pos = a.role === 'victime' && e.offset_ms >= 0 ? seq.mort : positionDe(a, e.offset_ms)
    if (pos) marques.push({ role, pt: pos, plein: a.role !== 'coequipier' || (e.dans_le_rayon_xuids ?? []).includes(a.xuid ?? '') })
    const segments = traineDe(a, e.offset_ms, seq.traine_ms)
    if (segments.length > 0) traines.push({ role, segments })
  }
  return { cercle: { centre: e.centre, rayonM: seq.rayon_m }, traines, marques }
}

/** Une ligne de la fiche d'étape : un acteur et sa distance au centre du cercle (« — » sans position). */
export interface DistanceDEtape {
  cle: string
  role: 'tueur' | 'coequipier'
  nom: string
  distance: string | null
  /** Coéquipier : plein dans le rayon à l'étape, creux au-delà (règle du serveur) ; `aucune` sans position. */
  marque: 'plein' | 'creux' | 'aucune'
}

/** distancesDEtape — la distance de chaque acteur (hors victime) au centre du cercle à l'étape. */
export function distancesDEtape(t: TacticalText, seq: TacticalSequence, etape: number, locale: Locale): DistanceDEtape[] {
  const e = seq.etapes?.[etape]
  if (!e) return []
  const out: DistanceDEtape[] = []
  for (const a of seq.acteurs ?? []) {
    if (a.role === 'victime') continue
    const p = positionDe(a, e.offset_ms)
    out.push({
      cle: `${a.role}-${a.xuid ?? a.gamertag ?? ''}`,
      role: a.role === 'tueur' ? 'tueur' : 'coequipier',
      nom: a.gamertag || t.mortDetailUnnamed,
      distance: p ? metresEntiers(Math.hypot(p.x - e.centre.x, p.y - e.centre.y), locale) : null,
      marque: a.role !== 'coequipier' || !p ? 'aucune' : (e.dans_le_rayon_xuids ?? []).includes(a.xuid ?? '') ? 'plein' : 'creux',
    })
  }
  return out
}

/** libelleDeDecalage — « −4,0 s », « 0,0 s », « +3,0 s » (signe moins typographique). */
export function libelleDeDecalage(offsetMs: number, locale: Locale): string {
  const n = decimale(Math.abs(offsetMs) / 1000, locale)
  if (offsetMs < 0) return `−${n} s`
  return offsetMs > 0 ? `+${n} s` : `${n} s`
}
