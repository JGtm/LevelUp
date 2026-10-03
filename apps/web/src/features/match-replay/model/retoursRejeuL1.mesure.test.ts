/// <reference types="node" />
/**
 * retoursRejeuL1.mesure.test.ts — CONTROLE DE PARC DU LOT L1 DES RETOURS DU REJEU (2026-09-23).
 *
 * ## Pourquoi cette mesure existe
 *
 * Exigence utilisateur du 2026-09-23 (§3.0 du plan `PLAN_RETOURS_REJEU_2026-09-23.md`) : un
 * correctif doit valoir pour TOUS les films, présents et futurs, sans régression. Les tests du
 * lot prouvent la règle sur des fixtures ; cette mesure la prouve sur le PARC ENTIER, avant /
 * après, en passant par les VRAIS modules du rejeu (aucune copie de logique) :
 *
 *  - L1.1 : la colonne des fiches (`ReplayTeams`, rendue en jsdom) à des instants échantillonnés
 *    — nombre de libellés d'état de mouvement affichés, et HTML identique au reste près ;
 *  - L1.2 : `scoreTimelineOf` -> `leaderStates` -> `buildScoreDominance` (la composition de
 *    `scoreTrack`, privée du hook ; la garde « sosie de la dominance » demande le fil de la page
 *    et n'est pas rejouée) et `readScoreBanner` sur un tableau de score SYNTHÉTIQUE tiré du
 *    roster (le vrai vient de la base, qu'on n'ouvre pas) ;
 *  - L1.3 : le prédicat de dessin des véhicules (`vehicleIsHidden`, `vehicleIsDecor` à la base) ;
 *  - L1.4 : `positionAt` / `altitudeAt` / `trailAt` sur une grille d'instants, rangés par
 *    rapport aux lacunes (`Point.g`) ;
 *  - L1.5 : `buildShotFx` (famille, teinte, montage) et `shotSoundStem` par tir.
 *
 * ## Régime
 *
 * Le même fichier tourne sur les DEUX arbres (base de campagne et branche du lot) : il détecte
 * les symboles du lot (`vehicleIsHidden`, `campCountOf`) et retombe sur ceux de la base.
 *
 *     RR_L1_MESURE=1 RR_L1_MESURE_DIR=<abs>/data/cache/replays/halo_infinite \
 *       RR_L1_MESURE_OUT=<abs>/apres.json [RR_L1_MESURE_AVANT=<abs>/avant.json] \
 *       npx vitest run src/features/match-replay/model/retoursRejeuL1.mesure.test.ts
 *
 * Avec `RR_L1_MESURE_AVANT`, la sortie courante est comparée à celle de la base et le rapport
 * s'écrit à côté (`<OUT>.rapport.json`). Lecture seule, un document à la fois en mémoire ; sans
 * `RR_L1_MESURE`, la suite est ignorée (même porte que `tourelleVisee.mesure.test.ts`).
 */
import { createHash } from 'node:crypto'
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { createElement } from 'react'
import { render, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { ReplayDocument, ReplayPoint } from '@/lib/api/types'

import * as Pos from '../../../lib/replay/replayLogic'
import * as Score from '../../../lib/replay/scoreTimeline'
import { rosterEntryKey } from '../../../lib/replay/rosterLogic'
import type { ReplayDocumentReady, ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { useReplayTiming } from '../hooks/useReplayTiming'
import { shotSoundStem } from '../sound/replaySound'
import { racineDuDepot } from '../test/featureFiles'
import { testReplayDoc } from '../test/testDoc'
import { ReplayTeams } from '../ui/ReplayTeams'
import { buildScoreDominance, trackScale } from './replayTimelineTracksLogic'
import { readScoreBanner } from './scoreBannerLogic'
import { buildShotFx } from './shotFx'
import { vehicleCycleLives } from './vehicleCycleTime'
import * as Veh from './vehiclesLayer'

const ACTIF = process.env.RR_L1_MESURE === '1'
/** Les libellés d'état que la fiche affichait du 21/09 au 23/09 (FR, rendus en locale fr). */
const LIBELLES_ETAT = ['Accroupi', 'Glissade', 'Escalade', 'Saut (dérivé)', 'Sprint']
/** Le mot d'état tel que la fiche le posait : un span à cette classe exacte, sur la ligne du nom. */
const SPAN_ETAT = /<span class="shrink-0 pl-1 text-\[9px\] uppercase tracking-\[\.06em\] text-muted-foreground">[^<]*<\/span>/g
const INSTANTS_UNIFORMES = 12
const INSTANTS_CIBLES = 6
const PAS_GRILLE = 30

const dossier = (): string =>
  process.env.RR_L1_MESURE_DIR ?? join(racineDuDepot(), 'data', 'cache', 'replays', 'halo_infinite')

const sha = (): ReturnType<typeof createHash> => createHash('sha1')

function charger(fichier: string): ReplayDocumentReady {
  const brut = JSON.parse(readFileSync(join(dossier(), fichier), 'utf8')) as Partial<ReplayDocument>
  return testReplayDoc(brut)
}

// --- L1.4 : lacunes -----------------------------------------------------------------------------

const estLacune = (p: { g?: number }): boolean => (p.g ?? 0) > 0

/** Classement d'un instant : DANS une lacune (strictement entre `a` et `b`, `b.g > 0`) ? */
function dansLacune(points: readonly ReplayPoint[], t: number): boolean {
  if (points.length < 2 || t <= points[0].t || t >= points[points.length - 1].t) return false
  let i = 1
  while (i < points.length && points[i].t <= t) i++
  return t > points[i - 1].t && estLacune(points[i])
}

function grilleDe(points: readonly ReplayPoint[]): number[] {
  const out = new Set<number>()
  const a = points[0].t
  const b = points[points.length - 1].t
  for (let t = a; t <= b; t += PAS_GRILLE) out.add(t)
  out.add(b)
  for (let i = 1; i < points.length; i++) {
    if (!estLacune(points[i])) continue
    const p = points[i - 1].t
    const q = points[i].t
    for (const t of [p, p + 1, Math.floor((p + q) / 2), q - 1, q, q + 1]) if (t >= a && t <= b) out.add(t)
  }
  return [...out].sort((x, y) => x - y)
}

interface MesurePositions {
  lacunes: number
  lacunesPlus10m: number
  instantsHors: number
  hashHors: string
  instantsTraineeTraverse: number
  hashTraverseTete: string
  hashTraverseTrainee: string
  dansLacunes: [number, number, number, number, number][]
  desaccordsInGapAt: number
  echantillonsVehiculeAvecG: number
  hashVehicules: string
}

function mesurerPositions(doc: ReplayDocumentReady, fenetreTrainee: number): MesurePositions {
  const hors = sha()
  const tete = sha()
  const trainee = sha()
  const m: MesurePositions = {
    lacunes: 0, lacunesPlus10m: 0, instantsHors: 0, hashHors: '', instantsTraineeTraverse: 0,
    hashTraverseTete: '', hashTraverseTrainee: '', dansLacunes: [], desaccordsInGapAt: 0,
    echantillonsVehiculeAvecG: 0, hashVehicules: '',
  }
  const inGapAt = (Pos as unknown as Record<string, unknown>).inGapAt as
    | ((p: readonly ReplayPoint[], t: number) => boolean)
    | undefined
  doc.tracks.forEach((track, ti) => {
    const pts = track.points
    if (pts.length === 0) return
    for (let i = 1; i < pts.length; i++) {
      if (!estLacune(pts[i])) continue
      m.lacunes++
      if (Math.hypot(pts[i].x - pts[i - 1].x, pts[i].y - pts[i - 1].y) > 10) m.lacunesPlus10m++
    }
    for (const t of grilleDe(pts)) {
      const pos = Pos.positionAt(pts, t)
      const z = Pos.altitudeAt(pts, t)
      const lacune = dansLacune(pts, t)
      if (inGapAt && inGapAt(pts, t) !== lacune) m.desaccordsInGapAt++
      if (lacune) {
        m.dansLacunes.push([ti, t, pos?.x ?? NaN, pos?.y ?? NaN, z ?? NaN])
        continue
      }
      const tr = Pos.trailAt(pts, t, fenetreTrainee)
      const traverse = pts.some((p) => p.t >= t - fenetreTrainee && p.t <= t && estLacune(p))
      if (traverse) {
        m.instantsTraineeTraverse++
        tete.update(JSON.stringify([ti, t, pos, z]))
        trainee.update(JSON.stringify([ti, t, tr]))
      } else {
        m.instantsHors++
        hors.update(JSON.stringify([ti, t, pos, z, tr]))
      }
    }
  })
  const veh = sha()
  for (const v of doc.vehicles) {
    m.echantillonsVehiculeAvecG += v.samples.filter((s) => estLacune(s as { g?: number })).length
    for (let t = v.t0; t <= v.t1max; t += PAS_GRILLE) veh.update(JSON.stringify(Veh.vehiclePositionAt(v, t)))
  }
  m.hashHors = hors.digest('hex')
  m.hashTraverseTete = tete.digest('hex')
  m.hashTraverseTrainee = trainee.digest('hex')
  m.hashVehicules = veh.digest('hex')
  return m
}

// --- L1.1 : fiches ------------------------------------------------------------------------------

interface InstantFiche {
  frame: number
  etats: number
  motsEtat: number
  hash: string
  lacune: boolean
}

function instantsFiches(doc: ReplayDocumentReady): number[] {
  const out = new Set<number>()
  const n = Math.max(1, doc.frameCount)
  for (let i = 0; i < INSTANTS_UNIFORMES; i++) out.add(Math.floor(((i + 0.5) * n) / INSTANTS_UNIFORMES))
  const pas = Math.max(1, Math.floor(doc.stances.length / INSTANTS_CIBLES))
  for (let i = 0; i < doc.stances.length && out.size < INSTANTS_UNIFORMES + INSTANTS_CIBLES; i += pas) {
    const s = doc.stances[i]
    out.add(Math.floor((s.t0 + s.t1) / 2))
  }
  const gaps: number[] = []
  for (const tr of doc.tracks) {
    for (let i = 1; i < tr.points.length && gaps.length < INSTANTS_CIBLES; i++) {
      if (estLacune(tr.points[i])) gaps.push(Math.floor((tr.points[i - 1].t + tr.points[i].t) / 2))
    }
  }
  for (const g of gaps) out.add(g)
  return [...out].sort((a, b) => a - b)
}

function mesurerFiches(doc: ReplayDocumentReady): InstantFiche[] {
  return instantsFiches(doc).map((frame) => {
    const { container, unmount } = render(createElement(ReplayTeams, { doc, scoreboard: [], frame, locale: 'fr' }))
    const html = container.innerHTML
    const texte = container.textContent ?? ''
    unmount()
    const etats = (html.match(SPAN_ETAT) ?? []).length
    const motsEtat = LIBELLES_ETAT.reduce((n, l) => n + texte.split(l).length - 1, 0)
    const lacune = doc.tracks.some((t) => dansLacune(t.points, frame))
    return { frame, etats, motsEtat, hash: sha().update(html.replace(SPAN_ETAT, '')).digest('hex'), lacune }
  })
}

// --- L1.2 : score -------------------------------------------------------------------------------

function mesurerScore(doc: ReplayDocumentReady) {
  const tl = Score.scoreTimelineOf(doc)
  const campCount = typeof Score.campCountOf === 'function' ? Score.campCountOf(doc.roster) : 0
  const camps = [...new Set(doc.roster.map((r) => r.team).filter((t): t is number => t != null && t >= 0))].sort((x, y) => x - y)
  const scoreboard = doc.roster
    .filter((r) => r.team != null && r.team >= 0)
    .map((r) => ({ xuid: rosterEntryKey(r), team_side: `t${r.team}` }))
  const allies = new Map(scoreboard.map((r) => [r.xuid, { ally: r.team_side === `t${camps[0]}` }]))
  const frames = [0.1, 0.5, 0.9, 1].map((f) => Math.floor(f * Math.max(0, doc.frameCount - 1)))
  const bandeau = frames.map((f) => readScoreBanner(tl, scoreboard, allies, f))
  const etats = Score.leaderStates(tl, campCount)
  return {
    teamIdentity: doc.coverage?.score?.teamIdentity ?? null,
    series: tl?.teams.length ?? null,
    seriesSansCamp: tl ? tl.teams.filter((t) => t.teamId == null).length : null,
    campsRoster: camps.length,
    etats: JSON.stringify(etats),
    segments: JSON.stringify(buildScoreDominance(etats, trackScale(null, doc.frameCount))),
    bandeau: JSON.stringify(bandeau),
    bandeauAffiche: bandeau.some((b) => b !== null),
  }
}

// --- L1.3 : véhicules ---------------------------------------------------------------------------

function cache(v: ReplayVehicleTrackReady): boolean {
  const veh = Veh as unknown as Record<string, unknown>
  return typeof veh.vehicleIsHidden === 'function'
    ? (veh.vehicleIsHidden as (t: ReplayVehicleTrackReady) => boolean)(v)
    : Veh.vehicleIsDecor(v.family)
}

function mesurerVehicules(doc: ReplayDocumentReady) {
  const surEmplacement = new Set<ReplayVehicleTrackReady>()
  for (const c of doc.vehicleCycles) for (const v of vehicleCycleLives(c, doc.vehicles)) surEmplacement.add(v)
  return doc.vehicles.map((v) => ({
    slot: v.slot, gen: v.gen, family: v.family ?? '', t0: v.t0, end: v.end, samples: v.samples.length,
    s0AtT0: v.samples.length > 0 && v.samples[0].t === v.t0, rides: v.rides.length,
    cache: cache(v), emplacement: surEmplacement.has(v),
  }))
}

// --- L1.5 : tirs --------------------------------------------------------------------------------

function mesurerTirs(doc: ReplayDocumentReady, aimHold: number) {
  const fx = buildShotFx(doc, aimHold)
  const perso = sha()
  let nPerso = 0
  const vehicule: [string, boolean, string, string, string, string][] = []
  let j = 0
  let premiereVieFausse = 0
  for (const s of doc.shots) {
    const e = fx[j]
    const rendu = e && e.frame === s.t && e.seed === s.t + s.slot ? fx[j++] : null
    const style = rendu ? `${rendu.fam}/${rendu.tint}` : 'melee'
    const stem = shotSoundStem(doc, s) ?? '-'
    const montage = JSON.stringify(rendu?.vehicleShot?.mount ?? null)
    if (s.w && doc.weaponLabels?.[s.w]) {
      nPerso++
      perso.update(JSON.stringify([s.t, s.slot, style, stem, rendu?.vehicleShot ?? null]))
      continue
    }
    vehicule.push([s.w ?? '', s.v !== undefined, style, stem, montage, JSON.stringify(rendu?.vehicleShot?.family ?? null)])
    if (s.v !== undefined) {
      const premiere = doc.vehicles.find((v) => v.slot === s.v)
      const couvre = doc.vehicles.find((v) => v.slot === s.v && v.t0 <= s.t && s.t <= v.t1max)
      if (premiere !== couvre) premiereVieFausse++
    }
  }
  return { nPerso, hashPerso: perso.digest('hex'), vehicule, premiereVieFausse, rendus: fx.length }
}

// --- Assemblage, comparaison --------------------------------------------------------------------

function mesurerDocument(fichier: string) {
  const doc = charger(fichier)
  const { result } = renderHook(() => useReplayTiming(doc))
  const timing = result.current.timing
  return {
    schema: doc.schemaVersion,
    positions: mesurerPositions(doc, timing.trail),
    fiches: mesurerFiches(doc),
    score: mesurerScore(doc),
    vehicules: mesurerVehicules(doc),
    tirs: mesurerTirs(doc, timing.aimHold),
  }
}

type Mesure = ReturnType<typeof mesurerDocument>

/** Les écarts qui NE DOIVENT PAS exister, par document ; et ceux attendus, décrits. */
function comparer(avant: Record<string, Mesure>, apres: Record<string, Mesure>) {
  const r = {
    documents: 0, regressions: [] as string[],
    l11: { etatsAvant: 0, etatsApres: 0, motsApres: 0, instants: 0, htmlDiffHorsLacune: 0, htmlDiffEnLacune: 0 },
    l12: { bandeauTu: [] as string[], bandeauAutre: [] as string[], pisteApparue: [] as string[], pisteChangee: [] as string[], pisteDisparue: [] as string[] },
    l13: { masques: [] as string[], masquesHorsForme: [] as string[], demasques: [] as string[], surEmplacement: [] as string[] },
    l14: { lacunes: 0, lacunesPlus10m: 0, instantsHors: 0, instantsTraverse: 0, instantsDans: 0, deplacesEvites: 0, deplacementMax: 0, traineesChangees: 0, desaccordsInGapAt: 0, echantillonsVehiculeAvecG: 0 },
    l15: { parTag: {} as Record<string, Record<string, number>>, persoChanges: [] as string[], nPerso: 0, premiereVieFausse: 0 },
  }
  for (const [id, b] of Object.entries(avant)) {
    const a = apres[id]
    if (!a) { r.regressions.push(`${id}: absent apres`); continue }
    r.documents++
    comparerFiches(id, b, a, r)
    comparerScore(id, b, a, r)
    comparerVehicules(id, b, a, r)
    comparerPositions(id, b, a, r)
    comparerTirs(id, b, a, r)
  }
  return r
}
type Rapport = ReturnType<typeof comparer>

function comparerFiches(id: string, b: Mesure, a: Mesure, r: Rapport): void {
  b.fiches.forEach((fb, i) => {
    const fa = a.fiches[i]
    r.l11.instants++
    r.l11.etatsAvant += fb.etats
    r.l11.etatsApres += fa.etats
    r.l11.motsApres += fa.motsEtat
    if (fa.hash === fb.hash) return
    if (fb.lacune) r.l11.htmlDiffEnLacune++
    else { r.l11.htmlDiffHorsLacune++; r.regressions.push(`${id}: fiche differente hors lacune a ${fb.frame}`) }
  })
}

function comparerScore(id: string, b: Mesure, a: Mesure, r: Rapport): void {
  if (b.score.bandeau !== a.score.bandeau) {
    const tu = b.score.bandeauAffiche && !a.score.bandeauAffiche && (a.score.seriesSansCamp ?? 0) > 0
    ;(tu ? r.l12.bandeauTu : r.l12.bandeauAutre).push(`${id} (${b.score.teamIdentity}, series ${b.score.series}, sans camp ${b.score.seriesSansCamp})`)
  }
  if (b.score.segments === a.score.segments) return
  const vide = (s: string): boolean => s === '[]'
  const ligne = `${id} (${b.score.teamIdentity}, series ${b.score.series}, camps ${b.score.campsRoster})`
  if (vide(b.score.segments)) r.l12.pisteApparue.push(ligne)
  else if (vide(a.score.segments)) r.l12.pisteDisparue.push(ligne)
  else r.l12.pisteChangee.push(ligne)
}

function comparerVehicules(id: string, b: Mesure, a: Mesure, r: Rapport): void {
  b.vehicules.forEach((vb, i) => {
    const va = a.vehicules[i]
    const nom = `${id} slot ${vb.slot}/${vb.gen} ${vb.family || '?'} samples=${vb.samples} rides=${vb.rides} end=${vb.end}`
    if (!vb.cache && va.cache) {
      r.l13.masques.push(nom)
      if (vb.samples !== 1 || !vb.s0AtT0 || vb.rides !== 0 || vb.end !== 'film_end') r.l13.masquesHorsForme.push(nom)
      if (vb.emplacement) r.l13.surEmplacement.push(nom)
    } else if (vb.cache && !va.cache) r.l13.demasques.push(nom)
  })
}

function comparerPositions(id: string, b: Mesure, a: Mesure, r: Rapport): void {
  const pb = b.positions
  const pa = a.positions
  r.l14.lacunes += pa.lacunes
  r.l14.lacunesPlus10m += pa.lacunesPlus10m
  r.l14.instantsHors += pa.instantsHors
  r.l14.instantsTraverse += pa.instantsTraineeTraverse
  r.l14.instantsDans += pa.dansLacunes.length
  r.l14.desaccordsInGapAt += pa.desaccordsInGapAt
  r.l14.echantillonsVehiculeAvecG += pa.echantillonsVehiculeAvecG
  if (pb.hashHors !== pa.hashHors) r.regressions.push(`${id}: position/trainee hors lacune differente`)
  if (pb.hashTraverseTete !== pa.hashTraverseTete) r.regressions.push(`${id}: tete differente hors lacune (trainee traversante)`)
  if (pb.hashTraverseTrainee !== pa.hashTraverseTrainee) r.l14.traineesChangees++
  if (pb.hashVehicules !== pa.hashVehicules) r.regressions.push(`${id}: position de vehicule differente`)
  pb.dansLacunes.forEach((d, i) => {
    const e = pa.dansLacunes[i]
    const dist = Math.hypot(d[2] - e[2], d[3] - e[3])
    if (dist > 0) r.l14.deplacesEvites++
    r.l14.deplacementMax = Math.max(r.l14.deplacementMax, dist)
  })
}

function comparerTirs(id: string, b: Mesure, a: Mesure, r: Rapport): void {
  r.l15.nPerso += a.tirs.nPerso
  r.l15.premiereVieFausse += a.tirs.premiereVieFausse
  if (b.tirs.hashPerso !== a.tirs.hashPerso) {
    r.l15.persoChanges.push(id)
    r.regressions.push(`${id}: tir d arme personnelle change`)
  }
  b.tirs.vehicule.forEach((vb, i) => {
    const va = a.tirs.vehicule[i]
    const cle = `${vb[0] || '(sans tag)'} ${vb[1] ? 'v' : 'sans-v'}`
    const t = (r.l15.parTag[cle] ??= {})
    const ligne = `style ${vb[2]} -> ${va[2]} | son ${vb[3]} -> ${va[3]} | montage ${vb[4] === 'null' ? 'non' : 'oui'} -> ${va[4] === 'null' ? 'non' : 'oui'}`
    t[ligne] = (t[ligne] ?? 0) + 1
  })
}

/**
 * LES FILMS FUTURS : un document au schéma courant auquel manque ce que le lot lit. Chaque cas
 * doit retomber sur le comportement d'avant le lot, ou sur le neutre — jamais sur une lecture
 * d'un voisin.
 */
function filmsFuturs() {
  const vehicule = { slot: 800, gen: 1, t0: 0, t1: 100, t1max: 100, end: 'film_end', family: 'warthog' }
  const doc = testReplayDoc({
    roster: [{ xuid: 'A', filmIndex: 0, name: 'A', team: 0 }, { xuid: 'B', filmIndex: 1, name: 'B', team: 1 }],
    tracks: [{ slot: 1, team: 0, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 10, y: 0 }] }],
    vehicles: [vehicule],
    shots: [{ slot: 1, t: 50, x: 0, y: 0, w: '0xDEADBEEF00000000', v: 800 }],
    scoreTimeline: { teams: [{ teamId: 0, rounds: null, total: [] }] },
  })
  const [fx] = buildShotFx(doc, 50)
  const tl = doc.scoreTimeline
  return {
    sansG: Pos.positionAt(doc.tracks[0].points, 50),
    sansGTrainee: Pos.trailAt(doc.tracks[0].points, 50, 100).length,
    vehiculeSansSamplesCache: cache(doc.vehicles[0]),
    tagInconnu: { fam: fx.fam, tint: fx.tint, mount: fx.vehicleShot?.mount ?? null, son: shotSoundStem(doc, doc.shots[0]) ?? null },
    serieVide: { etats: Score.leaderStates(tl, 2), bandeau: readScoreBanner(tl, [], undefined, 50) },
    seriesAbsentes: Score.leaderStates(testReplayDoc({ scoreTimeline: { teams: [] } }).scoreTimeline, 2),
  }
}

describe.skipIf(!ACTIF)('mesure de parc — lot L1 des retours du rejeu (2026-09-23)', () => {
  it('films futurs : un document auquel manque ce que le lot lit retombe sur le neutre', () => {
    const f = filmsFuturs()
    console.log(JSON.stringify(f))
    expect(f.sansG).toEqual({ x: 5, y: 0 })
    expect(f.sansGTrainee).toBe(2)
    expect(f.vehiculeSansSamplesCache).toBe(false)
    expect(f.tagInconnu).toEqual({ fam: 'plain', tint: 'neutral', mount: null, son: null })
    expect(f.serieVide.etats).toEqual([])
    expect(f.seriesAbsentes).toEqual([])
  })

  it('mesure chaque document du parc, puis compare a la base quand elle est donnee', () => {
    const filtre = process.env.RR_L1_MESURE_FILTRE?.split(',')
    const fichiers = readdirSync(dossier())
      .filter((f) => f.endsWith('.json') && !f.includes('derived'))
      .filter((f) => !filtre || filtre.some((p) => f.startsWith(p)))
      .sort()
    const out: Record<string, Mesure> = {}
    for (const f of fichiers) out[f.slice(0, 8)] = mesurerDocument(f)
    const cible = process.env.RR_L1_MESURE_OUT
    if (cible) writeFileSync(cible, JSON.stringify(out))
    console.log(`documents mesures : ${fichiers.length}`)
    const base = process.env.RR_L1_MESURE_AVANT
    if (!base || !existsSync(base)) return
    const rapport = comparer(JSON.parse(readFileSync(base, 'utf8')) as Record<string, Mesure>, out)
    if (cible) writeFileSync(`${cible}.rapport.json`, JSON.stringify(rapport, null, 2))
    console.log(JSON.stringify(rapport, null, 2))
    expect(rapport.regressions).toEqual([])
    expect(rapport.l11.etatsApres + rapport.l11.motsApres).toBe(0)
    expect(rapport.l13.masquesHorsForme).toEqual([])
    expect(rapport.l14.desaccordsInGapAt).toBe(0)
  }, 3_600_000)
})
