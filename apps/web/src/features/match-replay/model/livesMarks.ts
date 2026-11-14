/**
 * livesMarks.ts — LES TRACES DES VIES sur la carte : où chaque vie commence (apparitions), où elle
 * finit (morts) et d'où part le frag qui l'a finie (tueurs). Logique PURE : la construction des
 * marques à partir du document et du fil, le filtre « qui / quand » et le fondu par âge. Le dessin
 * est dans `layers/livesLayer.ts`, le câblage dans `layers/useReplayLives.ts`.
 *
 * TOUT EST DÉJÀ DANS LE DOCUMENT (aucun champ ajouté côté serveur) : les pistes par vie portent le
 * cap par image (`Point.h`), le fil porte les frags et les morts sans tueur, et les positions se
 * relisent par le résolveur de porteur (un joueur tué au volant meurt sur son véhicule).
 *
 * RIEN N'EST DESSINÉ À UNE POSITION INVENTÉE. Une marque dont la position, la couleur ou la fiche
 * manque n'existe pas : elle est COMPTÉE dans `skipped`, jamais posée ailleurs. Seule exception
 * voulue : le cap d'une apparition, qui est optionnel — sans cap lu à l'image d'apparition, la
 * marque reste un anneau, sans pointe.
 *
 * QUI : tous, mon équipe (l'allégeance lue dans le film, vue du point de vue), ou le joueur suivi
 * (celui que la carte marque « moi » : le point de vue). Le sujet d'une marque est celui qui
 * l'a faite : le propriétaire de la vie, la victime, le tueur.
 *
 * QUAND : toute la partie, ou jusqu'ici (ce qui est arrivé avant l'image courante, plus pâle avec
 * l'âge, jusqu'à un plancher). En « jusqu'ici », l'apparition d'une vie EN COURS reste à pleine
 * opacité : c'est la vie qu'on regarde, pas une trace.
 */
import { heldReading, isAliveAt, msToFrames, positionAt, trackWindow, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import type { ReplayFeedEntry } from './killFeedLogic'

/** Les trois lectures du filtre « qui », dans l'ordre où le panneau les propose. */
export const LIVES_WHO = ['all', 'team', 'followed'] as const
export type LivesWho = (typeof LIVES_WHO)[number]

/** Les deux lectures du filtre « quand », dans l'ordre où le panneau les propose. */
export const LIVES_WHEN = ['all', 'sofar'] as const
export type LivesWhen = (typeof LIVES_WHEN)[number]

/** Défauts : tout le monde, toute la partie. */
export const DEFAULT_LIVES_WHO: LivesWho = 'all'
export const DEFAULT_LIVES_WHEN: LivesWhen = 'all'

/** Opacité d'une marque à l'âge maximal : elle pâlit, elle ne disparaît pas. */
export const LIVES_FADE_FLOOR = 0.15
/** Âge, en ms de match, au-delà duquel une marque est au plancher. */
export const LIVES_FADE_MS = 60_000

/** Celui qui a fait la marque : sa couleur d'équipe et ce qui le situe par rapport au point de vue. */
export interface LivesActor {
  color: string
  /** Du camp du point de vue (allégeance `true`). */
  mine: boolean
  /** Le joueur suivi : celui que la carte marque « moi ». */
  followed: boolean
  /**
   * La vie de celui qui a fait la marque, pour « Seul » et « Masquer » du panneau de joueur : son slot
   * et une image DANS cette vie (`shownAtSlot` ne lit que des vies). `null` = aucune vie retrouvée :
   * la marque ne peut pas être prouvée affichée, elle se tait tant qu'un filtre est actif.
   */
  slot: number | null
  slotFrame: number
}

/** L'apparition d'une vie : sa première image visible, et le regard à cette image. */
export interface SpawnMark extends LivesActor {
  frame: number
  /** Dernière image de la vie : jusque-là, la marque est celle d'une vie EN COURS. */
  endFrame: number
  x: number
  y: number
  /** Cap en degrés lu à l'image d'apparition ; `null` = aucune lecture, anneau seul. */
  headingDeg: number | null
}

/** La mort d'un joueur : le lieu où la victime a été relue. */
export interface DeathMark extends LivesActor {
  frame: number
  x: number
  y: number
}

/** Le tueur à l'image du frag, et sa victime quand elle est positionnée (le trait). */
export interface KillerMark extends LivesActor {
  frame: number
  x: number
  y: number
  victim: XY | null
}

/** Les trois jeux de marques d'un document, et ce qui n'a pas pu être posé. */
export interface LivesMarks {
  spawns: readonly SpawnMark[]
  deaths: readonly DeathMark[]
  killers: readonly KillerMark[]
  /** Compte des marques sans position, couleur ou fiche : jamais dessinées, jamais devinées. */
  skipped: { spawns: number; deaths: number; killers: number }
  /** Un joueur est marqué « moi » : le choix « Joueur suivi » a un sens. */
  followedKnown: boolean
}

/** Les lectures du canvas dont la construction a besoin, toutes INJECTÉES. */
export interface LivesResolvers {
  /** Position monde du porteur (bipède ou véhicule) à une image, `null` si inconnue. */
  posOf: (xuid: string, frame: number) => XY | null
  /** Le joueur occupe-t-il une fiche à cette image ? Sinon il n'a pas de pion. */
  seated: (xuid: string, frame: number) => boolean
  colorOfXuid: (xuid: string) => string | null
  /** Couleur du propriétaire de la vie du slot À L'IMAGE, `null` = aucun pion. */
  colorOfSlot: (slot: number, frame: number) => string | null
  /** Du camp du point de vue ? (allégeance `true`) */
  mineOfXuid: (xuid: string) => boolean
  mineOfSlot: (slot: number, frame: number) => boolean
  /** Le joueur suivi (marque « moi ») ? */
  followedOfXuid: (xuid: string) => boolean
  followedOfSlot: (slot: number, frame: number) => boolean
  /** La vie du joueur à une image (ou la dernière, juste close) : son slot et une image dans la vie. */
  slotAt: (xuid: string, frame: number) => { slot: number; frame: number } | null
}

/**
 * lifeSlotAt — la vie d'un joueur à une image : celle qui la couvre, sinon la dernière close depuis moins
 * de `deathFrames` (la victime vient de mourir, par construction). L'image rendue est ramenée DANS la
 * vie : une mort datée une image après la fin de la vie reste la mort de CETTE vie.
 */
export function lifeSlotAt(
  lives: readonly ReplayTrackReady[] | undefined,
  frame: number,
  deathFrames: number,
): { slot: number; frame: number } | null {
  const alive = lives?.find((l) => isAliveAt(l, frame))
  if (alive) return { slot: alive.slot, frame }
  let best: { slot: number; frame: number } | null = null
  let bestAge = Number.POSITIVE_INFINITY
  for (const l of lives ?? []) {
    const end = trackWindow(l).end
    const age = frame - end
    if (age >= 0 && age <= deathFrames && age < bestAge) {
      bestAge = age
      best = { slot: l.slot, frame: end }
    }
  }
  return best
}

/** Le filtre « Seul / Masquer » : la marque est-elle de ce que la carte montre ? Sans filtre, tout. */
export function passesShown(
  mark: Pick<LivesActor, 'slot' | 'slotFrame'>,
  shownAtSlot: ((slot: number, frame: number) => boolean) | undefined,
): boolean {
  if (!shownAtSlot) return true
  return mark.slot !== null && shownAtSlot(mark.slot, mark.slotFrame)
}

/**
 * Les apparitions : une marque par vie. L'image d'apparition est la première image où la vie est
 * DESSINÉE : le début de la vie, ou son premier point de position s'il est postérieur. Le cap est
 * lu à CETTE image, sans retenue : une lecture plus ancienne appartiendrait à une autre vie.
 */
function buildSpawns(
  doc: ReplayDocumentReady,
  r: LivesResolvers,
): { marks: SpawnMark[]; skipped: number } {
  const marks: SpawnMark[] = []
  let skipped = 0
  for (const life of doc.tracks) {
    const first = life.points[0]
    if (!first) continue
    const w = trackWindow(life)
    const frame = Math.max(w.start, first.t)
    const pos = positionAt(life.points, frame)
    const color = r.colorOfSlot(life.slot, frame)
    if (!pos || !color) {
      skipped++
      continue
    }
    marks.push({
      frame,
      endFrame: w.end,
      x: pos.x,
      y: pos.y,
      headingDeg: heldReading(life.points, frame, (p) => p.h, 0)?.value ?? null,
      color,
      mine: r.mineOfSlot(life.slot, frame),
      followed: r.followedOfSlot(life.slot, frame),
      slot: life.slot,
      slotFrame: frame,
    })
  }
  return { marks, skipped }
}

/** Un acteur posé : sa position et son encre. */
interface PlacedActor {
  pos: XY
  color: string
}

/** Un acteur posé : positionné, assis à une fiche et coloré, sinon rien (règle du pion). */
function placeActor(xuid: string, frame: number, r: LivesResolvers): PlacedActor | null {
  if (!xuid || !r.seated(xuid, frame)) return null
  const pos = r.posOf(xuid, frame)
  const color = r.colorOfXuid(xuid)
  return pos && color ? { pos, color } : null
}

/** Ce qu'une ligne du fil dit d'une mort : qui est mort, et à quelle image. */
interface DeathFact {
  victimXuid: string
  killerXuid: string
  frame: number
}

/**
 * Les morts du fil : les frags (victime nommée) et les morts sans tueur, sur l'image du fil.
 *
 * UNE FIN DE VIE N'EST PAS UNE MORT. Dans un match à manches, les vies des survivants se ferment SANS
 * mort à chaque fin de manche (tous les joueurs repartent au point de départ de leur équipe), et le
 * fil en fait des lignes neutres : une fin de vie qu'aucun frag ne revendique. Seule une mort neutre
 * dont le film ÉTABLIT la cause (`kind` : chute, hors-limites, suicide — lue dans le composant
 * dead-state, `doc.neutralDeaths`) est une vraie mort ; sans cause établie (`kind` vide) la ligne peut
 * être une fin de manche, et on ne pose pas de croix. Elle est comptée dans les morts non posées.
 *
 * LA LIMITE, NOMMÉE : une vraie mort sans type (source de dégât sans pictogramme, refus d'attribution
 * ligne par ligne) n'a pas non plus de croix. Le document ne porte aucune fin de manche lue vie par vie : sa
 * `scoreTimeline` date le dernier palier de chaque manche, les vies se ferment plus tard, et une fenêtre
 * entre les deux serait une heuristique. Distinguer les deux demande un champ publié par le décodeur.
 */
function deathFacts(
  entries: readonly ReplayFeedEntry[],
  doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>,
): { facts: DeathFact[]; unplacedDeaths: number } {
  const facts: DeathFact[] = []
  let unplacedDeaths = 0
  for (const e of entries) {
    if (e.kill) {
      const frame = Math.round(msToFrames(e.kill.replayMs, doc))
      if (e.kill.victimXuid) {
        facts.push({ victimXuid: e.kill.victimXuid, killerXuid: e.kill.xuid, frame })
      } else {
        unplacedDeaths++
        facts.push({ victimXuid: '', killerXuid: e.kill.xuid, frame })
      }
    } else if (e.death) {
      if (e.death.kind === '') {
        unplacedDeaths++
        continue
      }
      const frame = Math.round(msToFrames(e.death.replayMs, doc))
      facts.push({ victimXuid: e.death.xuid, killerXuid: '', frame })
    }
  }
  return { facts, unplacedDeaths }
}

/** La vie d'un acteur à l'image d'une marque, dans la forme des marques. */
function slotOf(xuid: string, frame: number, r: LivesResolvers): Pick<LivesActor, 'slot' | 'slotFrame'> {
  const life = r.slotAt(xuid, frame)
  return { slot: life?.slot ?? null, slotFrame: life?.frame ?? frame }
}

/** La croix d'une mort : le lieu où la victime a été relue, à son encre. */
function deathMarkOf(f: DeathFact, victim: PlacedActor, r: LivesResolvers): DeathMark {
  return {
    frame: f.frame,
    x: victim.pos.x,
    y: victim.pos.y,
    color: victim.color,
    mine: r.mineOfXuid(f.victimXuid),
    followed: r.followedOfXuid(f.victimXuid),
    ...slotOf(f.victimXuid, f.frame, r),
  }
}

/** Le triangle d'un tueur, et le trait vers la victime quand elle est posée. */
function killerMarkOf(
  f: DeathFact,
  killer: PlacedActor,
  victim: PlacedActor | null,
  r: LivesResolvers,
): KillerMark {
  return {
    frame: f.frame,
    x: killer.pos.x,
    y: killer.pos.y,
    victim: victim?.pos ?? null,
    color: killer.color,
    mine: r.mineOfXuid(f.killerXuid),
    followed: r.followedOfXuid(f.killerXuid),
    ...slotOf(f.killerXuid, f.frame, r),
  }
}

/**
 * buildLivesMarks construit les trois jeux de marques d'un document. `entries` est le fil ALIGNÉ de
 * la page (jamais les kills bruts : une seule horloge pour le fil et la carte).
 *
 * UN FRAG SANS VICTIME NOMMÉE donne un tueur (sans trait) mais pas de croix : on ne sait pas où
 * l'autre est tombé. Un tueur qui est sa propre victime n'est pas un tueur (la mort reste).
 */
export function buildLivesMarks(
  doc: ReplayDocumentReady,
  entries: readonly ReplayFeedEntry[],
  r: LivesResolvers,
): LivesMarks {
  const spawn = buildSpawns(doc, r)
  const { facts, unplacedDeaths } = deathFacts(entries, doc)
  const deaths: DeathMark[] = []
  const killers: KillerMark[] = []
  let skippedDeaths = unplacedDeaths
  let skippedKillers = 0
  for (const f of facts) {
    const victim = f.victimXuid ? placeActor(f.victimXuid, f.frame, r) : null
    if (f.victimXuid) {
      if (victim) deaths.push(deathMarkOf(f, victim, r))
      else skippedDeaths++
    }
    if (!f.killerXuid || f.killerXuid === f.victimXuid) continue
    const killer = placeActor(f.killerXuid, f.frame, r)
    if (killer) killers.push(killerMarkOf(f, killer, victim, r))
    else skippedKillers++
  }
  const followedKnown = [...spawn.marks, ...deaths, ...killers].some((m) => m.followed)
  return {
    spawns: spawn.marks,
    deaths,
    killers,
    skipped: { spawns: spawn.skipped, deaths: skippedDeaths, killers: skippedKillers },
    followedKnown,
  }
}

/**
 * effectiveLivesWho — « Joueur suivi » sans joueur suivi ne montrerait rien : la lecture retombe
 * sur « tous », pour CE match seulement (la préférence mémorisée n'est pas effacée).
 */
export function effectiveLivesWho(who: LivesWho, followedKnown: boolean): LivesWho {
  return who === 'followed' && !followedKnown ? 'all' : who
}

/** Le filtre « qui » : la marque est-elle de ceux qu'on veut voir ? */
export function passesWho(mark: Pick<LivesActor, 'mine' | 'followed'>, who: LivesWho): boolean {
  if (who === 'team') return mark.mine
  if (who === 'followed') return mark.followed
  return true
}

/** Le filtre « quand » et ce qu'il faut pour le lire : l'image courante et la durée d'une image. */
export interface LivesTime {
  when: LivesWhen
  frame: number
  /** Durée d'une image sur l'axe du match, en ms. */
  frameMs: number
}

/** L'opacité d'une marque d'âge `ageFrames` : 1 à l'instant, `LIVES_FADE_FLOOR` à `LIVES_FADE_MS`. */
export function fadeByAge(ageFrames: number, frameMs: number): number {
  const t = Math.min(1, Math.max(0, (ageFrames * frameMs) / LIVES_FADE_MS))
  return 1 - (1 - LIVES_FADE_FLOOR) * t
}

/**
 * markAlpha — l'opacité d'une marque à l'image courante : 0 = ne pas la peindre.
 *
 * `liveUntil` : en « jusqu'ici », la marque d'une vie EN COURS (`frame <= liveUntil`) est pleine ;
 * passé la fin de la vie, elle vieillit depuis cette fin. Sans `liveUntil`, elle vieillit depuis
 * son image (morts, tueurs).
 */
export function markAlpha(markFrame: number, time: LivesTime, liveUntil?: number): number {
  if (time.when === 'all') return 1
  if (markFrame > time.frame) return 0
  if (liveUntil === undefined) return fadeByAge(time.frame - markFrame, time.frameMs)
  return time.frame <= liveUntil ? 1 : fadeByAge(time.frame - liveUntil, time.frameMs)
}
