/**
 * useReplayFlagCarries — TOUT LE CÂBLAGE DU CALQUE DES DRAPEAUX, en un seul point.
 *
 * POURQUOI UN HOOK ET NON QUATRE MORCEAUX DANS `ReplayCanvas`. Le canvas du rejeu porte une
 * dette de taille GELÉE par un cliquet (`placementFamily.guard.test.ts`) : toute addition s'y
 * fait par EXTRACTION, jamais par empilement. Ce hook réunit donc les préoccupations du calque —
 * la relecture de position du porteur, le tracé par image, le survol et l'infobulle — et n'en
 * rend au canvas que quatre lignes utiles. Même parti que `useReplayWeaponPads`, `useZoneStates`
 * et `useSlotIdentity` avant lui.
 *
 * LE PORTEUR SE RELIT DANS SES TRAJECTOIRES, image par image, et c'est ce qui « colle » le
 * drapeau à son marqueur : le span publie UNE position pour tout son intervalle, alors que le
 * porteur court. `posOfPlayerAt` est le même utilitaire que les effets de mort et les pulses
 * d'objectif — une position relue, jamais devinée (cf. `flagPointAt` pour le repli).
 *
 * L'ONDE DE CAPTURE ENTRE ICI, ET PAS DANS LE CANVAS (2026-08-27). Elle a besoin des mêmes trois
 * choses que le glyphe — le document, la relecture de position, le camp vu de la page — et le
 * canvas est sous cliquet de taille : la brancher ailleurs aurait recopié cette jointure. Elle se
 * peint dans le MÊME `paint`, après le calque, et partage sa garde d'affichage.
 *
 * LE CAMP ALLIÉ EST UN NUMÉRO ICI, PAS UN XUID, exactement comme pour les zones : l'équipe du
 * drapeau vient du film, le point de vue de la page se lit sur la ligne « moi » du tableau de
 * bord (`team_side` écrit `t{N}`). Sans cette ligne, AUCUN camp n'est allié — le drapeau garde
 * l'encre neutre plutôt qu'une couleur devinée.
 *
 * L'IMAGE COURANTE EST LUE DANS UNE RÉFÉRENCE, jamais dans un état React (même règle et même
 * conséquence assumée que `useReplayWeaponPads`) : si l'état d'un drapeau change SOUS un pointeur
 * immobile, son infobulle attend le prochain mouvement. À l'arrêt — le cas où l'on inspecte — la
 * lecture est exacte.
 */
import { useCallback, useMemo, useState, type PointerEvent, type RefObject } from 'react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { parseTeamSideID } from '@/lib/halo/teamNames'

import { KILLPOS_WINDOW_MS, posOfPlayerAt } from './killFx'
import {
  drawFlagCarries,
  flagAt,
  type FlagCarriesInput,
  type FlagNow,
} from './flagCarriesLayer'
import {
  buildFlagCaptureFx,
  drawFlagCaptureFx,
  FLAG_CAPTURE_HOLD_FRAMES,
  type FlagCaptureStyle,
} from './flagCaptureFx'
import type { CanvasView } from './objectivesLayer'
import { buildFlagReturnDrops, drawFlagReturnZones } from './flagReturnZone'
import { canvasScale, frameToMs, msToFrames, worldToCanvas, type XY } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

/** Le camp d'un drapeau VU DE LA PAGE — `unknown` quand la ligne « moi » manque. */
export type FlagSide = 'ally' | 'enemy' | 'unknown'

/** Ce qui est survolé : le drapeau, son état LU À CET INSTANT, et de quoi l'écrire. */
export interface FlagHover {
  now: FlagNow
  at: XY
  side: FlagSide
  /** Le porteur dans la langue du lecteur, ou `null` quand rien ne le nomme. */
  carrier: string | null
  /** Depuis combien de temps cet état dure, en millisecondes du match. */
  sinceMs: number
}

export interface FlagCarriesHookInput {
  doc: ReplayDocumentReady
  view: CanvasView
  /** L'image courante, telle que la boucle de lecture la tient. */
  frameRef: RefObject<number>
  /** Faux quand le calque est éteint : rien n'est dessiné, rien ne se survole. */
  enabled: boolean
  scoreboard: MatchScoreboardRow[] | null | undefined
  /** Encre d'un camp vu de la page (tokens déjà résolus par l'appelant). */
  teamColorOf: (ally: boolean) => string
  /** Encre servie quand le camp est inconnu : ni équipe inventée, ni glyphe invisible. */
  neutral: string
  /**
   * Encre du FOND (tokens déjà résolus par l'appelant) : celle du liseré posé sous le glyphe.
   * C'est LA MÊME que le canvas sert déjà aux vignettes de socle — une seule résolution pour
   * les deux calques, sinon les deux contours divergeraient au premier réglage de thème.
   */
  outline: string
  reducedMotion: boolean
}

export interface ReplayFlagCarries {
  /** Le film porte-t-il des drapeaux ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /** Trace le calque à l'image demandée ; ne fait rien quand il est éteint. */
  paint: (ctx: CanvasRenderingContext2D, frame: number) => void
  hover: FlagHover | null
  onPointerMove: (event: PointerEvent<HTMLCanvasElement>) => void
  onPointerLeave: () => void
}

export function useReplayFlagCarries({
  doc,
  view,
  frameRef,
  enabled,
  scoreboard,
  teamColorOf,
  neutral,
  outline,
  reducedMotion,
}: FlagCarriesHookInput): ReplayFlagCarries {
  const carries = doc.flagCarries
  const [hover, setHover] = useState<FlagHover | null>(null)

  // LES VIES PAR JOUEUR, indexées une fois : la relecture de position tourne à chaque image et
  // pour chaque drapeau — la refaire dans la boucle balaierait toutes les traces du film.
  const livesByXuid = useMemo(() => {
    const map = new Map<string, ReplayTrackReady[]>()
    for (const t of doc.tracks) {
      if (!t.xuid) continue
      const list = map.get(t.xuid)
      if (list) list.push(t)
      else map.set(t.xuid, [t])
    }
    return map
  }, [doc.tracks])

  const deathFrames = useMemo(
    () => Math.max(1, Math.round(msToFrames(KILLPOS_WINDOW_MS, doc))),
    [doc],
  )
  const posOf = useCallback(
    (xuid: string, frame: number) => posOfPlayerAt(livesByXuid.get(xuid), frame, deathFrames),
    [livesByXuid, deathFrames],
  )

  const allyTeamID = useMemo(
    () => parseTeamSideID(scoreboard?.find((r) => r.is_me)?.team_side ?? null),
    [scoreboard],
  )
  // UN DRAPEAU SANS ÉQUIPE N'EST PAS « ADVERSE ». Deux cas le produisent : la carte est hors du
  // catalogue d'objectifs (aucun socle, donc aucun camp), et — depuis le schéma 29 — la variante
  // À DRAPEAU NEUTRE, où l'unique drapeau n'appartient à personne. Sans cette garde, le premier
  // camp qui n'est pas le mien emporte la comparaison et le glyphe prend l'encre ennemie : une
  // couleur affirmée là où il n'y a rien à affirmer.
  const sideOf = useCallback(
    (team: number): FlagSide => {
      if (allyTeamID === null || team < 0) return 'unknown'
      return team === allyTeamID ? 'ally' : 'enemy'
    },
    [allyTeamID],
  )
  const nameOfXuid = useMemo(() => {
    const map = new Map<string, string>()
    for (const r of scoreboard ?? []) if (r.gamertag) map.set(r.xuid, r.gamertag)
    return map
  }, [scoreboard])

  // L'ENCRE SUIT LE CAMP VU DE LA PAGE, pas l'index d'équipe du film : c'est la couleur que
  // l'utilisateur a choisie pour « allié » et « adverse » (règle d'accessibilité du rejeu). Sans
  // ligne « moi », le neutre du thème — jamais une équipe supposée. DEUX calques la demandent
  // désormais (le glyphe et l'onde de capture) : une seule règle, jamais deux copies.
  const inkOfTeam = useCallback(
    (team: number) => {
      const side = sideOf(team)
      return side === 'unknown' ? neutral : teamColorOf(side === 'ally')
    },
    [sideOf, neutral, teamColorOf],
  )

  const layer = useMemo<FlagCarriesInput>(
    () => ({ style: { colorOfTeam: inkOfTeam, outline, reducedMotion }, posOf }),
    [inkOfTeam, outline, reducedMotion, posOf],
  )

  // LES CAPTURES (retour du 2026-08-27) : construites UNE fois par document, comme les autres
  // effets ponctuels. Elles ne rallument PAS les pulses substituts retirés (`flagPulsesRetired`
  // reste tel quel) — cet effet n'écoute qu'une stat, `flag_captures`, et la pose au lieu RELU de
  // son auteur, là où le substitut posait toute la famille `flag_` sur le socle le plus proche.
  const captures = useMemo(() => buildFlagCaptureFx(doc, posOf), [doc, posOf])

  // LE CAMP DE L'AUTEUR SE LIT AU TABLEAU DE BORD, jamais dans le film : l'action ne porte que le
  // xuid. Un auteur absent du tableau (ou dont le `team_side` ne se parse pas) n'a PAS de camp —
  // l'onde prend le neutre du thème plutôt qu'une équipe devinée, même règle que le glyphe.
  const teamOfXuid = useMemo(() => {
    const map = new Map<string, number>()
    for (const r of scoreboard ?? []) {
      const team = parseTeamSideID(r.team_side ?? null)
      if (team !== null) map.set(r.xuid, team)
    }
    return map
  }, [scoreboard])

  // LA ZONE DE RETOUR (schéma 29) : le cercle autour d'un drapeau tombé, et la jauge qui s'y
  // vide. Elle se construit UNE fois par document, comme les ondes de capture — l'occupation se
  // compte image par image, et la recompter à chaque peinture coûterait le match entier.
  //
  // C'EST ICI, ET NON SUR LE SERVEUR, QUE LES DÉFENSEURS SE COMPTENT : l'équipe d'un joueur n'est
  // pas dans le film, elle vient du tableau de bord — que cette page a déjà joint pour colorer
  // les camps (`teamOfXuid`, juste au-dessus). Le serveur publie la RÈGLE, pas l'occupation.
  const defendersOf = useCallback(
    (team: number) => {
      const out: string[] = []
      if (team < 0) return out
      for (const [xuid, t] of teamOfXuid) if (t === team) out.push(xuid)
      return out
    },
    [teamOfXuid],
  )

  // LES CONTESTATAIRES SONT LES AUTRES, et le drapeau NEUTRE n'en a aucun : son équipe vaut -1,
  // personne n'en est « l'ennemi », et la jauge se réduit alors à la minuterie — ce que le jeu
  // fait aussi (un drapeau neutre ne se renvoie pas, il revient tout seul).
  const enemiesOf = useCallback(
    (team: number) => {
      const out: string[] = []
      if (team < 0) return out
      for (const [xuid, t] of teamOfXuid) if (t !== team) out.push(xuid)
      return out
    },
    [teamOfXuid],
  )

  const returnDrops = useMemo(
    () =>
      buildFlagReturnDrops(carries, {
        rule: doc.flagReturnZone ?? null,
        frameIntervalMs: doc.frameIntervalMs ?? 0,
        posOf,
        defendersOf,
        enemiesOf,
      }),
    [carries, doc.flagReturnZone, doc.frameIntervalMs, posOf, defendersOf, enemiesOf],
  )

  const captureStyle = useMemo<FlagCaptureStyle>(
    () => ({
      inkOf: (xuid: string) => {
        const team = teamOfXuid.get(xuid)
        return team === undefined ? neutral : inkOfTeam(team)
      },
      reducedMotion,
    }),
    [teamOfXuid, neutral, inkOfTeam, reducedMotion],
  )

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      // LA GARDE « AUCUN DRAPEAU PUBLIÉ » VAUT AUSSI POUR LES ONDES, et ce n'est pas un détail :
      // les pulses substituts d'`objectivesLayer` ne se taisent QUE quand le document publie des
      // drapeaux. Sur un film qui n'en publie pas, ils marquent encore les captures — dessiner
      // l'onde en plus les doublerait, à deux endroits différents.
      if (!enabled || carries.length === 0) return
      // LA ZONE PASSE SOUS LE GLYPHE : c'est un décor de lieu, le drapeau est le sujet.
      drawFlagReturnZones(
        ctx,
        returnDrops,
        (p) => worldToCanvas(p, view.bounds, view.width, view.height, view.pad),
        canvasScale(view.bounds, view.width, view.height, view.pad),
        frame,
        { colorOfTeam: inkOfTeam },
      )
      drawFlagCarries(ctx, layer, carries, view, frame)
      // Les ondes APRÈS le calque : l'anneau s'ouvre autour du glyphe, il ne passe pas dessous.
      drawFlagCaptureFx(ctx, captures, view, { frame, hold: FLAG_CAPTURE_HOLD_FRAMES }, captureStyle)
    },
    [enabled, carries, layer, view, captures, captureStyle, returnDrops, inkOfTeam],
  )

  const onPointerMove = useCallback(
    (event: PointerEvent<HTMLCanvasElement>) => {
      if (!enabled || carries.length === 0 || view.width === 0) {
        setHover((prev) => (prev === null ? prev : null))
        return
      }
      const rect = event.currentTarget.getBoundingClientRect()
      // Le contexte dessine en pixels CSS ; le rapport ne vaut 1 que si la mise en page ne remet
      // pas le canevas à l'échelle — on le calcule plutôt que de le supposer (même amorce que le
      // survol des socles et celui des poses).
      const kx = rect.width > 0 ? view.width / rect.width : 1
      const ky = rect.height > 0 ? view.height / rect.height : 1
      const at = { x: (event.clientX - rect.left) * kx, y: (event.clientY - rect.top) * ky }
      const frame = frameRef.current
      const found = flagAt(carries, layer, view, frame, at)
      setHover((prev) => {
        if (!found) return prev === null ? prev : null
        const next: FlagHover = {
          now: found.now,
          at: found.at,
          side: sideOf(found.now.team),
          carrier: found.now.xuid ? (nameOfXuid.get(found.now.xuid) ?? null) : null,
          sinceMs: Math.max(frameToMs(frame - found.now.t0, doc), 0),
        }
        if (
          prev &&
          prev.now.team === next.now.team &&
          prev.now.state === next.now.state &&
          prev.now.t0 === next.now.t0 &&
          prev.at.x === next.at.x &&
          prev.at.y === next.at.y &&
          prev.sinceMs === next.sinceMs
        ) {
          return prev
        }
        return next
      })
    },
    [enabled, carries, layer, view, frameRef, sideOf, nameOfXuid, doc],
  )

  const onPointerLeave = useCallback(() => {
    setHover((prev) => (prev === null ? prev : null))
  }, [])

  return { available: carries.length > 0, paint, hover, onPointerMove, onPointerLeave }
}
