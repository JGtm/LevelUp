/**
 * _objectiveCaptures.ts — extraction des captures CTF depuis les événements
 * d'objectif, pour l'overlay capture des charts combat (MatchKDCumulChart,
 * MatchTugOfWarChart).
 *
 * Source : `GET /players/{slug}/matches/{matchId}/objective-events` (cf.
 * MatchObjectiveEvent). Pour CTF : objectiveType='flag', eventType='capture',
 * teamId=0|1 (MÊME numérotation que scoreboard.team_side), players[0].xuid =
 * le scorer.
 *
 * La résolution équipe (ally/enemy) suit EXACTEMENT la même logique que les
 * deux charts : allyTeam = team_side du joueur courant (is_me) ; un teamId est
 * "ally" si teamId === allyTeam. Le scorer est résolu en gamertag via le
 * scoreboard (fallback xuid masqué), garantissant zéro xuid brut affiché.
 *
 * Renvoie [] si aucun event flag/capture exploitable (mode non-CTF) → overlay
 * absent, dégradation propre.
 */
import type { MatchObjectiveEvent, MatchScoreboardRow } from '@/lib/api/types'
import { displayPlayerName } from '@/lib/players/displayName'

/** Une capture CTF résolue, prête pour le rendu overlay. */
export interface CtfCapture {
  /** Instant de la capture, en millisecondes depuis le début du match. */
  tMs: number
  /** True si la capture profite à l'équipe du joueur courant. */
  ally: boolean
  /** Nom d'affichage du scorer (gamertag résolu, jamais un xuid brut). */
  scorer: string
}

/**
 * extractCtfCaptures — projette les MatchObjectiveEvent en captures CTF
 * exploitables par l'overlay des charts combat.
 *
 * @param objectiveEvents Événements d'objectif (peut être null/undefined).
 * @param scoreboard      Lignes du scoreboard (résolution équipe + gamertag).
 * @param meXUID          xuid du joueur courant (détermine l'équipe alliée).
 * @returns Liste triée par temps croissant ; [] si pas de capture flag.
 */
export function extractCtfCaptures(
  objectiveEvents: MatchObjectiveEvent[] | null | undefined,
  scoreboard: MatchScoreboardRow[] | null | undefined,
  meXUID: string | null,
): CtfCapture[] {
  if (!objectiveEvents || objectiveEvents.length === 0) return []

  const sb = scoreboard ?? []
  const meRow = meXUID ? sb.find((r) => r.xuid === meXUID) : undefined
  const allyTeam = meRow?.team_side ?? null
  const gamertagByXuid = new Map<string, string>()
  for (const r of sb) gamertagByXuid.set(r.xuid, r.gamertag)

  const captures: CtfCapture[] = []
  for (const ev of objectiveEvents) {
    if (ev.objectiveType !== 'flag' || ev.eventType !== 'capture') continue
    if (ev.timeMs == null || ev.teamId == null) continue

    // allyTeam est un team_side (string|null) côté scoreboard ; teamId est un
    // entier 0|1 dans MÊME numérotation. Comparaison en string pour aligner.
    const ally = allyTeam != null && String(ev.teamId) === allyTeam
    const scorerXuid = ev.players[0]?.xuid
    const scorer = displayPlayerName(
      scorerXuid ? gamertagByXuid.get(scorerXuid) : undefined,
      scorerXuid,
    )
    captures.push({ tMs: ev.timeMs, ally, scorer })
  }

  captures.sort((a, b) => a.tMs - b.tMs)
  return captures
}
