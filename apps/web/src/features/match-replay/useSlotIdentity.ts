/**
 * useSlotIdentity — CE QUE LE CALQUE DES JOUEURS SAIT D'UNE VIE, par slot ET PAR IMAGE : sa
 * couleur d'équipe, sa marque d'identité (« moi », « ami ») et le nom à écrire dessous.
 *
 * POURQUOI UN HOOK À PART. Le calcul est UNE jointure (film <-> scoreboard), un index de
 * PROPRIÉTÉ par image (`buildSlotOwnership`), et quatre résolveurs frame-aware, tous purs et
 * déjà écrits dans `rosterLogic.ts` ; ce fichier n'ajoute que la mémoïsation React. Le poser
 * ici garde `ReplayCanvas.tsx` — qui porte déjà une dette de taille gelée — à sa longueur, au
 * lieu d'y coller vingt-cinq lignes de plus (CLAUDE.md n°5).
 *
 * TOUT EST RÉSOLU PAR SLOT ET PAR IMAGE, jamais par rang de trace ni par une valeur figée pour
 * tout le match : un slot de biped est une VIE, et son propriétaire est la vie qui l'occupe À
 * L'IMAGE lue. Une valeur figée par slot montrait un seul joueur pour tout le match dès qu'un
 * slot était réattribué entre manches (« deux DinoR00 et pas de SHROOM » — le bug que ce
 * chantier corrige).
 *
 * UNE VIE SANS PROPRIÉTAIRE — OU UN SLOT LIBRE À CETTE IMAGE — NE SE DESSINE PAS. `buildPlayers`
 * écarte les traces sans xuid (`if (!track.xuid) continue`) : leur slot n'entre dans aucune vie,
 * et `colorOfSlot` rend `null` — la convention du calque pour « ne rien dessiner » (cf.
 * `MarkerStyle`). Ce sont les caméras et les spectateurs de fin de partie ; les replier sur
 * l'encre neutre semait des pions gris qui ne désignaient personne (retour utilisateur du
 * 2026-08-20). La marque et le nom, eux, tombent déjà sur `undefined` / `null`.
 */
import { useMemo } from 'react'

import type { XuidMeta } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { NO_MARKS, type PlayerMarkKind } from './playerMarks'
import {
  buildPlayers,
  buildSlotOwnership,
  colorResolver,
  markResolver,
  nameResolver,
  sideResolver,
  type ReplayPlayer,
  type SlotOwnership,
} from './rosterLogic'
import type { ReplayDocumentReady } from './replayNormalize'

export interface SlotIdentity {
  /**
   * Couleur d'équipe de la vie qui occupe le slot À L'IMAGE — pour les marqueurs et les
   * traînées. `null` = aucun propriétaire à cette image (slot libre, ou vie sans propriétaire) :
   * RIEN à dessiner (convention `MarkerStyle.colorOfSlot`).
   */
  colorOfSlot: (slot: number, frame: number) => string | null
  markOfSlot: (slot: number, frame: number) => PlayerMarkKind | undefined
  nameOfSlot: (slot: number, frame: number) => string | null
  /**
   * CAMP de la vie qui occupe le slot à l'image (`team_side`), null quand il est inconnu ou le
   * slot libre. Distinct de la couleur : celle-ci ne connaît que « allié / adverse » vu du
   * joueur de la page, alors qu'opposer deux vies demande leur camp réel (cf. rosterLogic).
   * Le capteur de menaces en a besoin.
   */
  sideOfSlot: (slot: number, frame: number) => string | null
}

export interface SlotIdentityInput {
  doc: ReplayDocumentReady
  scoreboard: MatchScoreboardRow[] | undefined
  xuidMeta: XuidMeta | undefined
  marks: ReadonlyMap<string, PlayerMarkKind> | undefined
  /** Couleur d'un camp, tokens déjà résolus par l'appelant (ils suivent la palette). */
  teamColorOf: (ally: boolean) => string
  /**
   * Encre servie à une entrée de roster SANS xuid : elle a un slot, donc elle se dessine,
   * mais aucune équipe ne peut lui être attribuée (cf. `rosterLogic.colorResolver`). À ne pas
   * confondre avec un slot LIBRE à cette image — celui-là ne se dessine pas du tout.
   */
  neutral: string
  /**
   * COULEURS DISTINCTES PAR JOUEUR (option du tiroir, 2026-08-24) : une palette résolue,
   * cyclée sur l'ordre STABLE des joueurs de la jointure — la couleur d'un joueur ne bouge
   * pas d'une image à l'autre ni d'une vie à l'autre. `null`/absente = couleurs d'équipe
   * (le défaut, doctrine D1).
   */
  distinctColors?: readonly string[] | null
}

/**
 * distinctColorResolver — le résolveur du mode « distinctes par joueur » : chaque joueur prend
 * la couleur de son RANG dans la jointure (cyclée au-delà de la palette), et toutes ses vies la
 * portent — mais résolu PAR IMAGE, comme les couleurs d'équipe, pour qu'un slot réattribué entre
 * manches suive son propriétaire courant. Pur, testé à part du hook.
 */
export function distinctColorResolver(
  ownership: SlotOwnership,
  players: readonly ReplayPlayer[],
  colors: readonly string[],
): (slot: number, frame: number) => string | null {
  const rank = new Map<ReplayPlayer, number>(players.map((p, i) => [p, i]))
  return (slot, frame) => {
    const p = ownership.ownerAtFrame(slot, frame)
    if (!p || colors.length === 0) return null
    return colors[(rank.get(p) ?? 0) % colors.length]
  }
}

export function useSlotIdentity({
  doc, scoreboard, xuidMeta, marks, teamColorOf, neutral, distinctColors,
}: SlotIdentityInput): SlotIdentity {
  // LA JOINTURE FILM <-> BASE, faite une fois : elle donne à chaque vie son propriétaire.
  const players = useMemo(() => buildPlayers(doc, scoreboard ?? []), [doc, scoreboard])
  // L'INDEX DE PROPRIÉTÉ PAR IMAGE : slot -> vies triées, résolu à `ownerAtFrame`. Construit une
  // fois par jointure ; les résolveurs ci-dessous ne font que le lire.
  const ownership = useMemo(() => buildSlotOwnership(players), [players])
  const colorOfSlot = useMemo(
    () =>
      distinctColors && distinctColors.length > 0
        ? distinctColorResolver(ownership, players, distinctColors)
        : colorResolver(ownership, teamColorOf, (xuid) => xuidMeta?.get(xuid)?.ally ?? false, neutral),
    [ownership, players, distinctColors, teamColorOf, xuidMeta, neutral],
  )
  const markOfSlot = useMemo(() => markResolver(ownership, marks ?? NO_MARKS), [ownership, marks])
  const nameOfSlot = useMemo(() => nameResolver(ownership), [ownership])
  const sideOfSlot = useMemo(() => sideResolver(ownership), [ownership])

  return { colorOfSlot, markOfSlot, nameOfSlot, sideOfSlot }
}
