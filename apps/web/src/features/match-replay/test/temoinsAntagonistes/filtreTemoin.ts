/// <reference types="node" />
/**
 * filtreTemoin : CE QUE SEUL ET MASQUER D'UN JOUEUR RÉPONDENT, élément par élément, sur un film de
 * fixture. Trois lectures, celles que les calques font : le prédicat des pions (`shownSlotPredicate`),
 * la règle des éléments à parties (`ownershipShown`) et les prédicats par genre d'élément
 * (`buildFocusFilter`).
 *
 * Une réponse s'écrit en une chaîne de `1` (dessiné) et de `0` (retiré), dans l'ordre du document :
 * deux sorties se comparent caractère par caractère, et un écart désigne l'élément par son rang.
 *
 * `listeDesTests` relève par NOM les tests de la feature qui portent ces règles (fichier, puis
 * intitulés de `describe` et de `it`).
 */
import { isAliveAt, trackWindow } from '@/lib/replay/replayLogic'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { buildFocusFilter } from '../../model/focusFilter'
import { killsOfFeed } from '../../model/killFeedLogic'
import { buildKillFx } from '../../model/killFx'
import { ownershipShown, shownSlotPredicate, type ElementParty } from '../../model/playerPanelLogic'
import { drawablePlacements } from '../../model/spawnedObjects'
import { cheminCourt, lire, testsDeLaFeature } from '../featureFiles'
import { focusDe, type Scene } from './temoinsKit'

/** Une suite de réponses : la chaîne, et le nombre d'éléments dessinés sur le total. */
interface Reponses {
  dessines: number
  total: number
  bits: string
}

function reponses(values: readonly boolean[]): Reponses {
  return { dessines: values.filter(Boolean).length, total: values.length, bits: values.map((v) => (v ? '1' : '0')).join('') }
}

/** Le joueur qui occupe ce slot à cette image, ou `undefined`. */
export function occupantDuSlot(scene: Scene, slot: number, frame: number): ReplayPlayer | undefined {
  return scene.players.find((p) => p.lives.some((l) => l.slot === slot && isAliveAt(l, frame)))
}

/**
 * Les réponses de Seul ou de Masquer de `cible` sur les éléments du film qui appartiennent à un
 * joueur : pions, parties, effets de mort, armes au sol, poses, câbles de grappin, auteurs,
 * poussées, véhicules et occupants. Un genre absent du film rend une suite vide.
 */
export function sortiesDuFiltre(scene: Scene, cible: ReplayPlayer, mode: 'seul' | 'masquer'): Record<string, unknown> {
  const { doc } = scene
  const { hidden } = focusDe(scene, cible, mode)
  const pion = shownSlotPredicate(hidden)
  const partie = ownershipShown(hidden)
  const filtre = buildFocusFilter(doc, hidden)
  if (!pion || !partie || !filtre) throw new Error(`témoins : ${mode} ne filtre rien sur le film ${scene.film}`)

  const autre = scene.players.find((p) => p.xuid !== cible.xuid && p.lives.length > 0)
  if (!autre) throw new Error(`témoins : le film ${scene.film} n'a qu'un joueur`)
  const moi: ElementParty = { xuid: cible.xuid }
  const lui: ElementParty = { xuid: autre.xuid }
  const milieu = (t: { t0: number; t1: number }) => Math.floor((t.t0 + t.t1) / 2)

  return {
    film: scene.film,
    mode,
    cible: scene.players.indexOf(cible),
    // Pions : chaque vie, à sa première image, à son milieu et à sa dernière.
    pions: reponses(
      doc.tracks.flatMap((track) => {
        const w = trackWindow(track)
        return [w.start, Math.floor((w.start + w.end) / 2), w.end].map((f) => pion(track.slot, f))
      }),
    ),
    // La règle des parties, sur ses cas : personne, la cible, un autre, les deux, puis les mêmes une seconde fois (le
    // second jeu était la règle des éléments PARTAGÉS, devenue la règle unique).
    parties: reponses([
      partie([]),
      partie([moi]),
      partie([lui]),
      partie([moi, lui]),
      partie([]),
      partie([moi]),
      partie([lui]),
      partie([moi, lui]),
    ]),
    effetsDeMort: reponses(buildKillFx(doc, killsOfFeed(scene.feed)).map((e) => filtre.killFx(e))),
    armesAuSol: reponses(doc.groundWeapons.map((w) => filtre.groundWeapon(w))),
    // Les poses que le calque dessine : sa source, pas le tableau brut du document.
    poses: reponses(drawablePlacements(doc).map((p) => filtre.placement(p))),
    cablesDeGrappin: reponses(doc.grappleLines.map((g) => filtre.slotAt(g.slot, g.t0))),
    auteurs: reponses(scene.players.map((p) => filtre.author(p.xuid))),
    poussees: reponses(doc.knockbacks.map((k) => filtre.knockback({ slot: k.slot, by: k.by ?? null, frame: k.t }))),
    vehicules: reponses(
      doc.vehicles.flatMap((track) => track.rides.map((ride) => filtre.vehicleAt(track, milieu(ride)))),
    ),
    occupants: reponses(
      doc.vehicles.flatMap((track) => track.rides.map((ride) => filtre.occupant(ride, milieu(ride)))),
    ),
    destructions: reponses(
      doc.vehicles.filter((track) => track.destruction).map((track) => filtre.vehicleDestruction(track, track.t1)),
    ),
  }
}

/** Les symboles par lesquels un test se reconnaît comme portant les règles de Seul et Masquer. */
const REGLES = /ownershipShown|shownSlotPredicate|buildFocusFilter|killFxShown|shownAtSlot|focusFilter|playerFocus/

/** Les intitulés `describe(...)` et `it(...)` d'un fichier de test, dans l'ordre du fichier. */
function intitules(source: string): string[] {
  const out: string[] = []
  const motif = /^\s*(describe|it)(?:\.each\([^\n]*\))?\(\s*(['"`])((?:\\.|(?!\2).)*)\2/gm
  for (let m = motif.exec(source); m !== null; m = motif.exec(source)) out.push(`${m[1]} : ${m[3]}`)
  return out
}

/**
 * La liste, par nom, des tests de la feature qui portent les règles de Seul et Masquer. Les
 * fichiers de ce dossier n'y entrent pas : un témoin ne se cite pas lui-même.
 */
export function listeDesTests(): string {
  const lignes: string[] = []
  for (const fichier of testsDeLaFeature().sort()) {
    const court = cheminCourt(fichier)
    if (court.startsWith('test/temoinsAntagonistes/')) continue
    const source = lire(fichier)
    if (!REGLES.test(source)) continue
    lignes.push(court, ...intitules(source).map((t) => `  ${t}`))
  }
  return `${lignes.join('\n')}\n`
}
