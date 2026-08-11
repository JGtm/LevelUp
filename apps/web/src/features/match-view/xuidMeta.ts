/**
 * xuidMeta.ts — QUI EST QUI DANS UN MATCH : nom d'affichage et camp, par xuid.
 *
 * POURQUOI CE FICHIER EXISTE. La même cascade était écrite deux fois à l'identique
 * (`MatchTugOfWarChart`, `MatchKDCumulChart`) et le rejeu 2D en réclamait une troisième.
 * Règle du dépôt : à la troisième copie on centralise ET on pose le garde-rail — ici
 * `xuidMeta.guard.test.ts`, qui interdit de réécrire la cascade ailleurs.
 *
 * LE CAMP N'EST PAS DANS LA DONNÉE, il se déduit. « Allié » veut dire « du côté du joueur
 * dont on regarde la page » : on lit le `team_side` de sa ligne, et tout le monde qui
 * partage ce camp est allié. Sans ligne pour lui, personne n'est allié — mieux vaut aucun
 * camp qu'un camp inventé, car c'est la couleur des kills qui en dépend.
 */
import { displayPlayerName } from '@/lib/players/displayName'
import type { MatchScoreboardRow } from '@/lib/api/types'

/** Nom d'affichage (bots compris) et appartenance, par xuid. */
export type XuidMeta = ReadonlyMap<string, { gamertag: string; ally: boolean }>

/**
 * resolveXuidMeta indexe le scoreboard par xuid.
 *
 * `meXUID` désigne le joueur de la page ; sa ligne peut aussi se reconnaître à `is_me`,
 * qui reste prioritaire (une ligne marquée « moi » est alliée par définition).
 */
export function resolveXuidMeta(
  scoreboard: MatchScoreboardRow[] | null | undefined,
  meXUID: string | null,
): XuidMeta {
  const sb = scoreboard ?? []
  const meRow = meXUID ? sb.find((r) => r.xuid === meXUID) : undefined
  const allyTeam = meRow?.team_side ?? null
  const meta = new Map<string, { gamertag: string; ally: boolean }>()
  for (const r of sb) {
    const ally = r.is_me || (allyTeam != null && r.team_side === allyTeam)
    meta.set(r.xuid, { gamertag: displayPlayerName(r.gamertag, r.xuid), ally })
  }
  return meta
}

/**
 * meXUIDOf retrouve le joueur de la page dans le scoreboard, quand l'appelant ne le tient
 * pas d'ailleurs. `is_me` est posé par le backend sur la ligne du joueur consulté.
 */
export function meXUIDOf(scoreboard: MatchScoreboardRow[] | null | undefined): string | null {
  return (scoreboard ?? []).find((r) => r.is_me)?.xuid ?? null
}
