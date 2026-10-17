/**
 * sequenceLecture.logic — la LECTURE ANIMÉE d'une séquence de mort : l'horloge de lecture (bornes,
 * avance), la position d'un acteur à un instant quelconque (interpolation entre les échantillons
 * servis) et la scène du plan à cet instant. Pur : ni React, ni DOM, ni horloge système.
 *
 * L'AXE EST CELUI DU SERVEUR : un décalage en millisecondes depuis la mort (`offset_ms`), de la
 * première à la dernière étape servie (−4 s à +3 s). Les traces sont échantillonnées au pas de
 * `pas_ms` (200 ms) ; entre deux échantillons CONSÉCUTIFS la position est interpolée linéairement,
 * sauf quand le second porte une `coupure` (acteur absent, lacune du film, autre vie) : la position
 * y est inconnue, l'acteur n'est pas peint — jamais une ligne droite à travers une lacune.
 *
 * LES RÈGLES DE PORTÉE RESTENT AU SERVEUR : le caractère plein ou creux d'un coéquipier se lit sur
 * l'étape en vigueur (la dernière dont le décalage est atteint), publiée par le serveur ; la lecture
 * ne compare jamais une distance à un rayon.
 *
 * L'HORLOGE ÉCRITE est celle du rejeu : l'instant affiché de l'étape de la mort (`instant_affiche_ms`,
 * déjà converti par le serveur sur l'horloge du coup d'envoi, plancher 0) plus le décalage. Aucune
 * conversion d'axe n'est refaite ici.
 */
import type { TacticalSequence } from '@/lib/api/types'

import { SCENE_VIDE, type MarqueDeScene, type Pt, type SceneDuPlan, type TraineDeScene } from './mort.logic'

/** Le plus grand pas de temps d'une image de lecture : un onglet resté en arrière-plan ne fait pas sauter la lecture. */
export const PAS_MAX_MS = 100

type Acteur = NonNullable<TacticalSequence['acteurs']>[number]
type Echantillon = NonNullable<Acteur['points']>[number]

/** Les bornes de la lecture : la première et la dernière étape servies. */
export interface BornesDeLecture {
  debutMs: number
  finMs: number
}

/** bornesDeLecture — de la première à la dernière étape ; `null` sans deux étapes (rien à lire). */
export function bornesDeLecture(seq: TacticalSequence): BornesDeLecture | null {
  const etapes = seq.etapes ?? []
  if (etapes.length < 2) return null
  return { debutMs: etapes[0].offset_ms, finMs: etapes[etapes.length - 1].offset_ms }
}

/**
 * avancer — la tête de lecture après `dtMs` de temps réel, à vitesse 1. Le pas est borné à
 * `PAS_MAX_MS` et la lecture s'arrête à la borne de fin (`fini`) sans la dépasser.
 */
export function avancer(
  offsetMs: number,
  dtMs: number,
  bornes: BornesDeLecture,
): { offsetMs: number; fini: boolean } {
  const pas = Math.min(Math.max(dtMs, 0), PAS_MAX_MS)
  const suivant = Math.min(Math.max(offsetMs, bornes.debutMs) + pas, bornes.finMs)
  return { offsetMs: suivant, fini: suivant >= bornes.finMs }
}

/**
 * etapeALInstant — l'index de l'étape en vigueur à `offsetMs` : la dernière dont le décalage est
 * atteint, la première avant elle. `null` sans étape.
 */
export function etapeALInstant(seq: TacticalSequence, offsetMs: number): number | null {
  const etapes = seq.etapes ?? []
  if (etapes.length === 0) return null
  let index = 0
  etapes.forEach((e, i) => {
    if (e.offset_ms <= offsetMs) index = i
  })
  return index
}

/** etapeDeLaMort — l'index de l'étape à 0 ms (l'instant de la mort) ; `null` si la séquence n'en sert pas. */
export function etapeDeLaMort(seq: TacticalSequence): number | null {
  const i = (seq.etapes ?? []).findIndex((e) => e.offset_ms === 0)
  return i < 0 ? null : i
}

/**
 * horlogeAffichee — l'instant que l'horloge du rejeu affiche à `offsetMs` (ms depuis le coup d'envoi,
 * plancher 0), ou `null` quand le serveur n'a pas servi l'instant de la mort (coup d'envoi inconnu) :
 * l'appelant écrit alors le décalage.
 */
export function horlogeAffichee(seq: TacticalSequence, offsetMs: number): number | null {
  const mort = (seq.etapes ?? []).find((e) => e.offset_ms === 0)
  if (mort?.instant_affiche_ms == null) return null
  return Math.max(0, mort.instant_affiche_ms + offsetMs)
}

/**
 * positionALInstant — la position d'un acteur à `offsetMs` : l'échantillon exact, ou l'interpolation
 * linéaire entre deux échantillons consécutifs. `null` avant le premier, après le dernier, ou dans une
 * lacune (le second échantillon porte une coupure).
 */
export function positionALInstant(points: readonly Echantillon[], offsetMs: number): Pt | null {
  const i = points.findIndex((p) => p.offset_ms >= offsetMs)
  if (i < 0) return null
  const apres = points[i]
  if (apres.offset_ms === offsetMs) return { x: apres.x, y: apres.y }
  if (i === 0 || apres.coupure) return null
  const avant = points[i - 1]
  const f = (offsetMs - avant.offset_ms) / (apres.offset_ms - avant.offset_ms)
  return { x: avant.x + f * (apres.x - avant.x), y: avant.y + f * (apres.y - avant.y) }
}

/**
 * traineALInstant — la traîne d'un acteur à `offsetMs` : ses échantillons des `traineMs` précédentes,
 * prolongés de la position interpolée à la tête (l'instant) et à la queue (`offsetMs − traineMs`) quand
 * elles tombent entre deux échantillons, et coupés aux lacunes. Un segment d'un seul point n'est pas un trait.
 */
function traineALInstant(a: Acteur, offsetMs: number, traineMs: number): Pt[][] {
  const points = a.points ?? []
  const debut = offsetMs - traineMs
  const fenetre: { offset_ms: number; x: number; y: number; coupure: boolean }[] = []
  const queue = positionALInstant(points, debut)
  if (queue && !points.some((p) => p.offset_ms === debut)) fenetre.push({ offset_ms: debut, ...queue, coupure: false })
  for (const p of points) {
    if (p.offset_ms >= debut && p.offset_ms <= offsetMs) {
      fenetre.push({ offset_ms: p.offset_ms, x: p.x, y: p.y, coupure: p.coupure === true })
    }
  }
  const tete = positionALInstant(points, offsetMs)
  if (tete && !points.some((p) => p.offset_ms === offsetMs)) fenetre.push({ offset_ms: offsetMs, ...tete, coupure: false })

  const segments: Pt[][] = []
  let courant: Pt[] = []
  for (const p of fenetre) {
    if (p.coupure && courant.length > 0) {
      segments.push(courant)
      courant = []
    }
    courant.push({ x: p.x, y: p.y })
  }
  if (courant.length > 0) segments.push(courant)
  return segments.filter((s) => s.length > 1)
}

function roleDeMarque(a: Acteur): Exclude<MarqueDeScene['role'], 'nuage'> {
  return a.role === 'victime' ? 'mort' : a.role === 'tueur' ? 'tueur' : 'coequipier'
}

/**
 * centreDuCercle — le centre du cercle de portée à `offsetMs` : celui de l'étape à un décalage d'étape,
 * sinon le lieu de la mort (à partir de 0) ou la position interpolée de la victime (avant la mort).
 */
function centreDuCercle(seq: TacticalSequence, etape: NonNullable<TacticalSequence['etapes']>[number], offsetMs: number, victime: Pt | null): Pt {
  if (etape.offset_ms === offsetMs) return etape.centre
  if (offsetMs >= 0) return seq.mort
  return victime ?? etape.centre
}

/**
 * sceneALInstant — la scène du plan à `offsetMs` : le cercle à son centre (la victime avant la mort, le
 * lieu de la mort ensuite), les traînes de `traine_ms`, et chaque acteur à sa position de l'instant. Le mort
 * reste au lieu de sa mort à partir de l'instant 0 ; un acteur sans position à cet instant n'est pas peint.
 * Plein ou creux : l'étape en vigueur. À un décalage d'étape, la scène est exactement celle de l'étape.
 */
export function sceneALInstant(seq: TacticalSequence, offsetMs: number): SceneDuPlan {
  const index = etapeALInstant(seq, offsetMs)
  const etape = index === null ? undefined : seq.etapes?.[index]
  if (!etape) return SCENE_VIDE
  const marques: MarqueDeScene[] = []
  const traines: TraineDeScene[] = []
  let victime: Pt | null = null
  for (const a of seq.acteurs ?? []) {
    const role = roleDeMarque(a)
    const pos = a.role === 'victime' && offsetMs >= 0 ? seq.mort : positionALInstant(a.points ?? [], offsetMs)
    if (pos) {
      marques.push({ role, pt: pos, plein: a.role !== 'coequipier' || (etape.dans_le_rayon_xuids ?? []).includes(a.xuid ?? '') })
      if (a.role === 'victime') victime = pos
    }
    const segments = traineALInstant(a, offsetMs, seq.traine_ms)
    if (segments.length > 0) traines.push({ role, segments })
  }
  return { cercle: { centre: centreDuCercle(seq, etape, offsetMs, victime), rayonM: seq.rayon_m }, traines, marques }
}

/** sceneDeSequence — l'étape `etape` (index de `seq.etapes`) : la scène à son décalage. */
export function sceneDeSequence(seq: TacticalSequence, etape: number): SceneDuPlan {
  const e = seq.etapes?.[etape]
  return e ? sceneALInstant(seq, e.offset_ms) : SCENE_VIDE
}
