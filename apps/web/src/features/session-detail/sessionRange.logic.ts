/**
 * sessionRange.logic — les décisions PURES de la carte « Portée des engagements » de la
 * colonne de session (lot O, D22-4).
 *
 * LA PORTÉE EST RELATIVE AU LOBBY DU MATCH, jamais la médiane brute : la hauteur d'un bâton
 * est `lobby_delta_m`, l'écart signé entre ma médiane de frag et celle de tous les joueurs
 * du match. C'est lui qui neutralise la carte et le mode — en médiane brute, une session ne
 * dessine que la playlist de la soirée. Même normalisation que la carte « Rôles de portée »
 * de l'Escouade (lot R), à un autre scope.
 *
 * LES DÉCISIONS COMMUNES SONT IMPORTÉES, PAS RECOPIÉES (CLAUDE.md n°6) : le plancher de
 * mesure, l'ordre des matchs, les étiquettes « #N · carte », les quantiles et le classement
 * en rôles viennent de `features/squad/squadRangeRoles.logic` (import cross-feature
 * `session-detail=>squad`, déjà allowlisté). Ce qui est PROPRE à la session : un seul
 * joueur, un bâton par match, et des bandes de rôle assises sur les bâtons PLEINS DE LA
 * SESSION, pas sur une période.
 *
 * Pur : aucun React, aucune couleur en dur.
 */
import {
  PLANCHER_MESURE,
  categoriesMatchs,
  ordonnerProfils,
  quantileLineaire,
  roleDeEcart,
  type RoleDePortee,
  type SeuilsRoles,
} from '@/features/squad/squadRangeRoles.logic'
import type { MatchRangeBlock, MatchRangeProfile } from '@/lib/api/types'

export { PLANCHER_MESURE }
export type { RoleDePortee, SeuilsRoles }

/**
 * NOMBRE MINIMAL DE BÂTONS PLEINS pour dessiner les trois bandes de rôle. Sous ce nombre,
 * les tiers ne séparent rien (à deux bâtons, chaque bande tient un point) : pas de bandes
 * plutôt que des bandes fabriquées.
 */
export const MIN_BATONS_POUR_BANDES = 3

/** Un bâton : un match de la session, vu du joueur consulté. */
export interface BatonPortee {
  /** Position du match dans la session, du plus ancien (0) au plus récent. */
  ordre: number
  matchId: string
  mapName?: string
  /** Ma médiane de frag sur ce match, en mètres. */
  medianeM: number
  /** La médiane du lobby du match, en mètres. */
  lobbyM: number
  /** Mon écart à la médiane du lobby, en mètres (signé) — LA HAUTEUR DU BÂTON. */
  ecartM: number
  /** Mes frags mesurés sur ce match. */
  mesures: number
  /** `false` = bâton CREUX : sous le plancher, hors médiane de session et hors bandes. */
  plein: boolean
}

/**
 * Le joueur consulté dans un profil de match. Le scope Sessions ne sert QU'UN joueur par
 * profil (lot N2) ; `meLabel` ne départage donc que le cas dégradé d'un serveur qui en
 * servirait plusieurs, par gamertag insensible à la casse (le slug de route est en
 * minuscules, le titre écrit le gamertag comme il l'entend).
 */
function joueurConsulte(profil: MatchRangeProfile, meLabel: string) {
  const joueurs = profil.players ?? []
  if (joueurs.length <= 1) return joueurs[0]
  const cible = meLabel.toLowerCase()
  return joueurs.find((j) => (j.gamertag ?? '').toLowerCase() === cible) ?? joueurs[0]
}

/** batonsPortee projette le bloc en un bâton par match MESURÉ, dans l'ordre chronologique. */
export function batonsPortee(
  block: MatchRangeBlock | null | undefined,
  meLabel: string,
): { batons: BatonPortee[]; categories: string[] } {
  const profils = ordonnerProfils(block?.profiles ?? [])
  const categories = categoriesMatchs(profils)
  const batons: BatonPortee[] = []
  const gardees: string[] = []
  profils.forEach((profil, ordre) => {
    const joueur = joueurConsulte(profil, meLabel)
    // Aucun frag mesuré = aucune médiane : le match n'a pas de bâton du tout (un zéro
    // dessiné se lirait « joué pile à la distance du lobby »).
    if (joueur == null || joueur.measured <= 0) return
    batons.push({
      ordre,
      matchId: profil.match_id,
      mapName: profil.map_name,
      medianeM: joueur.median_m,
      lobbyM: profil.lobby_median_m,
      ecartM: joueur.lobby_delta_m,
      mesures: joueur.measured,
      plein: joueur.measured >= PLANCHER_MESURE,
    })
    gardees.push(categories[ordre] ?? `#${ordre + 1}`)
  })
  return { batons, categories: gardees }
}

/**
 * seuilsSession rend les deux bornes des bandes de rôle : les TIERS des bâtons pleins de la
 * session. `null` sous `MIN_BATONS_POUR_BANDES` bâtons pleins — pas de bandes.
 */
export function seuilsSession(batons: readonly BatonPortee[]): SeuilsRoles | null {
  const ecarts = batons
    .filter((b) => b.plein)
    .map((b) => b.ecartM)
    .sort((a, b) => a - b)
  if (ecarts.length < MIN_BATONS_POUR_BANDES) return null
  return { bas: quantileLineaire(ecarts, 1 / 3), haut: quantileLineaire(ecarts, 2 / 3) }
}

/**
 * medianeSession rend la médiane des écarts des bâtons PLEINS — la ligne de repère du
 * graphe. `null` quand aucun bâton n'est plein : une médiane sur des mesures écartées
 * n'est pas une médiane.
 */
export function medianeSession(batons: readonly BatonPortee[]): number | null {
  const ecarts = batons
    .filter((b) => b.plein)
    .map((b) => b.ecartM)
    .sort((a, b) => a - b)
  if (ecarts.length === 0) return null
  return quantileLineaire(ecarts, 0.5)
}

/** Le rôle d'un bâton sur les bandes de la session — `null` s'il est creux ou sans bandes. */
export function roleDuBaton(baton: BatonPortee, seuils: SeuilsRoles | null): RoleDePortee | null {
  if (!baton.plein || seuils == null) return null
  return roleDeEcart(baton.ecartM, seuils)
}
