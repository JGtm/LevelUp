/**
 * replayAntagonists.ts — LES ANTAGONISTES DU REJEU : seule source des surfaces qui les lisent
 * (canevas, frise, fiche du joueur).
 *
 * `lib/antagonists` compte, sur des clés déjà rapprochées et un prédicat d'adversaire fournis
 * par l'appelant. Ce fichier est cet appelant, pour le rejeu : il rapproche les éliminations du
 * fil aux clés du FILM et décide qui est adversaire. Aucune surface ne refait ce rapprochement.
 *
 * LES CLÉS. Un joueur est désigné par sa clé du film (`ReplayPlayer.xuid` : le xuid d'un humain,
 * `bot:<nom>` pour un bot).
 *  - tueur : `kill.xuid` ;
 *  - victime : `kill.victimXuid` quand il est non vide, sinon `botKey(kill.victimGamertag)` quand
 *    le nom est non vide, sinon l'élimination est ignorée (aucune victime n'est devinée).
 * Une clé de la feuille de match (ligne `board.xuid`, `bid(N.0)` pour un bot) est ramenée à la clé
 * du film du joueur quand le roster le connaît ; sinon elle reste telle quelle.
 *
 * L'ADVERSAIRE se lit du point de vue du SUJET : `allegiance.ofXuid(clé) === false`, l'allégeance
 * du film bâtie avec le sujet pour référence (une fiche de joueur n'a pas le point de vue de la
 * page). Autre camp selon le film, tout autre joueur en mêlée générale ; sans équipe connue
 * (`null`), personne n'est compté.
 *
 * LES BOTS COMPTENT COMME TOUT JOUEUR, dans les deux rôles. Le fil de la base nomme la victime bot
 * (par son nom, `botKey`) mais ne porte aucune élimination dont un bot est l'AUTEUR : celles-ci se
 * lisent dans le film (`botKillsOf` : `eliminations[]`, où la frontière a ramené la place du bot à sa
 * clé `bot:<nom>`), et seulement celles-là, pour ne rien compter deux fois. Une élimination de bot
 * sans victime nommée est ignorée : aucune victime n'est devinée.
 *
 * Pur : ni React ni DOM. Le hook `useReplayAntagonists` ne fait que mémoïser la fabrique.
 */
import { buildAntagonists, type AntagonistKill, type Antagonists } from '@/lib/antagonists/antagonists'
import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import { frameToMs } from '@/lib/replay/replayLogic'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import { botKey, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayFeedEntry } from './killFeedLogic'
import { trackedPlayerOf } from './playerPanelLogic'

/** Préfixe des clés de bot du film (`botKey`). */
const BOT_KEY_PREFIX = botKey('')

/** Index sans antagoniste, partagé : sujet inconnu ou absent du roster. */
const NO_ANTAGONISTS: Antagonists = buildAntagonists([], '', () => false)

/** Les antagonistes d'un match, par sujet. Un index par (fabrique, sujet), construit à la demande. */
export interface ReplayAntagonists {
  /**
   * L'index d'un sujet désigné par sa clé du film ; `null` = personne, index vide. Le même objet
   * est rendu à chaque appel pour un même sujet.
   */
  forSubject(subjectKey: string | null): Antagonists
  /**
   * L'index du joueur regardé : `viewpoint` est un xuid de feuille (ou une clé du film), ramené à
   * la clé du film par `antagonistSubjectKey`. Équivaut à `forSubject(antagonistSubjectKey(...))`.
   */
  forViewpoint(viewpoint: string | null): Antagonists
}

/**
 * antagonistSubjectKey — la clé du film du joueur regardé (`trackedPlayerOf`) ; `null` quand
 * personne n'est regardé ou que le joueur n'est pas au roster.
 */
export function antagonistSubjectKey(
  viewpoint: string | null,
  players: readonly ReplayPlayer[],
): string | null {
  return trackedPlayerOf(viewpoint, players)?.xuid ?? null
}

/** Les joueurs du roster par clé du film ET par ligne de feuille (la clé du film prime). */
function playersByKey(players: readonly ReplayPlayer[]): ReadonlyMap<string, ReplayPlayer> {
  const byKey = new Map<string, ReplayPlayer>()
  for (const p of players) byKey.set(p.xuid, p)
  for (const p of players) {
    if (p.board?.xuid && !byKey.has(p.board.xuid)) byKey.set(p.board.xuid, p)
  }
  return byKey
}

/**
 * antagonistKillsOf — les éliminations du fil aux clés du film, instants du rejeu
 * (`replayMs`) ; l'ordre est celui du fil, les doublons restent (l'index les écarte).
 */
export function antagonistKillsOf(
  feed: readonly ReplayFeedEntry[],
  players: readonly ReplayPlayer[],
): AntagonistKill[] {
  const byKey = playersByKey(players)
  const filmKey = (key: string): string => byKey.get(key)?.xuid ?? key
  const out: AntagonistKill[] = []
  for (const entry of feed) {
    const kill = entry.kill
    if (!kill || !kill.xuid) continue
    const victim = kill.victimXuid || (kill.victimGamertag ? botKey(kill.victimGamertag) : '')
    if (!victim) continue
    out.push({ killer: filmKey(kill.xuid), victim: filmKey(victim), t: kill.replayMs })
  }
  return out
}

/** Ce que `botKillsOf` lit du document : les éliminations (clés du film) et la cadence. */
export type BotKillsDocument = Pick<ReplayDocumentReady, 'eliminations' | 'frameIntervalMs'>

/**
 * botKillsOf — les éliminations DONT UN BOT EST L'AUTEUR, lues dans le film, aux clés du film et à leur instant sur
 * l'axe du rejeu. Le fil de la base n'en porte aucune ; les éliminations d'un humain, elles, sont déjà dans le fil.
 */
export function botKillsOf(doc: BotKillsDocument, players: readonly ReplayPlayer[]): AntagonistKill[] {
  const byKey = playersByKey(players)
  const out: AntagonistKill[] = []
  for (const e of doc.eliminations) {
    if (!isBotKey(e.xuid, byKey) || !e.victim) continue
    out.push({ killer: e.xuid, victim: byKey.get(e.victim)?.xuid ?? e.victim, t: frameToMs(e.t, doc) })
  }
  return out
}

/** Vrai pour une clé de bot du film, ou un joueur que le roster déclare bot. */
function isBotKey(key: string, byKey: ReadonlyMap<string, ReplayPlayer>): boolean {
  return key.startsWith(BOT_KEY_PREFIX) || byKey.get(key)?.bot === true
}

/**
 * createReplayAntagonists — la fabrique des index d'un match (un fil, un roster, les éliminations de bots du
 * film). À reconstruire quand l'un d'eux change : le cache des index en vit.
 */
export function createReplayAntagonists(
  feed: readonly ReplayFeedEntry[],
  players: readonly ReplayPlayer[],
  botKills: readonly AntagonistKill[] = [],
): ReplayAntagonists {
  const kills = [...antagonistKillsOf(feed, players), ...botKills]
  const cache = new Map<string, Antagonists>()

  const build = (subjectKey: string): Antagonists => {
    const allegiance = buildFilmAllegiance(players, subjectKey)
    return buildAntagonists(kills, subjectKey, (other) => allegiance.ofXuid(other) === false)
  }

  const forSubject = (subjectKey: string | null): Antagonists => {
    if (subjectKey === null) return NO_ANTAGONISTS
    let index = cache.get(subjectKey)
    if (!index) {
      index = build(subjectKey)
      cache.set(subjectKey, index)
    }
    return index
  }

  return {
    forSubject,
    forViewpoint: (viewpoint) => forSubject(antagonistSubjectKey(viewpoint, players)),
  }
}
