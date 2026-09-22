/**
 * squadRangeRoles.logic — les décisions PURES de la carte « Rôles de portée »
 * (`TeammatesPageResponse.range_profiles`, lot N2 du 2026-09-21).
 *
 * LA PORTÉE EST RELATIVE AU MATCH, JAMAIS LA MÉDIANE BRUTE. L'ordonnée est l'écart de la
 * médiane d'un joueur à la médiane de TOUS les joueurs du match (`lobby_delta_m`, servi
 * signé en mètres) : c'est ce qui neutralise la carte et le mode. Sur un BTB la médiane du
 * lobby vaut ~24 m, sur une arène ~11 m — en médiane brute, un joueur ne dessine que la
 * playlist de sa soirée.
 *
 * LES SEUILS DE RÔLE SONT DES QUANTILES, PAS DES MÈTRES EN DUR : tiers bas / tiers haut de
 * TOUS les points pleins de la période. Une escouade entièrement « longue » garde donc trois
 * rôles internes — le seul moyen d'en lire un équilibre.
 *
 * UN POINT SOUS `PLANCHER_MESURE` FRAGS MESURÉS N'EST PAS UNE MÉDIANE : il se dessine creux,
 * il sort de la tendance et il ne porte aucun rôle. Le rôle, lui, se lit sur une FENÊTRE
 * GLISSANTE de `FENETRE_ROLE` matchs — un match atypique se voit, il ne fait pas basculer
 * l'étiquette ; les `FENETRE_ROLE - 1` premiers matchs d'un joueur n'ont donc pas de rôle.
 *
 * Aucune dépendance React/ECharts : `SquadRangeRolesCard` ne fait que peindre ce que ce
 * module décide.
 */
import { truncateMap } from '@/lib/charts/matchLabels'
import type { MatchRangePlayer, MatchRangeProfile } from '@/lib/api/types'

/** Frags mesurés minimaux pour qu'une médiane de match compte (point plein). */
export const PLANCHER_MESURE = 5

/** Largeur de la fenêtre glissante qui décide d'un rôle (et trace la tendance). */
export const FENETRE_ROLE = 5

/**
 * Les trois rôles, ORDINAUX du plus bas au plus haut sur l'axe : `front` = tiers bas,
 * `polyvalent` = tiers médian, `sniper` = tiers haut.
 *
 * CE SONT DES RANGS, PAS DES LIBELLÉS. La grandeur portée les nomme « Ligne de front /
 * Polyvalent / Tireur d'élite » ; la grandeur hauteur les nomme « Contrebas / À niveau /
 * Hauteurs ». Les noms viennent du manifest i18n de la carte (`squad.<grandeur>.band_*`),
 * jamais de ce module : c'est ce qui permet au même nuage de servir les deux lectures.
 */
export type RoleDePortee = 'front' | 'polyvalent' | 'sniper'

/**
 * La GRANDEUR projetée en ordonnée. `portee` = distance médiane des frags (lot R) ;
 * `hauteur` = dénivelé médian signé des frags (`killer_z - victim_z`, lot U / E1). Même
 * échelle relative dans les deux cas : l'écart à la médiane du LOBBY du match.
 */
export type GrandeurProfil = 'portee' | 'hauteur'

/**
 * mesureDeJoueur lit la (médiane, écart au lobby) d'un joueur pour une grandeur.
 *
 * `null` quand la grandeur n'est pas mesurée sur ce (match, joueur) : les champs de
 * dénivelé sont OPTIONNELS côté API — absents = rien à dire, alors qu'un 0 m est une
 * mesure (« à plat »). Un point sans mesure ne se dessine pas du tout — il ne devient PAS
 * un point creux, qui lui dit « mesuré, mais sur trop peu de frags ».
 */
export function mesureDeJoueur(
  joueur: MatchRangePlayer,
  grandeur: GrandeurProfil,
): { medianeM: number; ecartM: number } | null {
  if (grandeur === 'portee') {
    return { medianeM: joueur.median_m, ecartM: joueur.lobby_delta_m }
  }
  const medianeM = joueur.elevation_median_m
  const ecartM = joueur.elevation_lobby_delta_m
  if (medianeM == null || ecartM == null) return null
  return { medianeM, ecartM }
}

/** Les trois rôles dans l'ordre ordinal — l'ordre de la légende et des teintes. */
export const ROLES_ORDONNES: RoleDePortee[] = ['front', 'polyvalent', 'sniper']

/** Un point du nuage : un (match, joueur). */
export interface PointPortee {
  /** Position du match dans la période, du plus ancien (0) au plus récent. */
  ordre: number
  matchId: string
  mapName?: string
  xuid: string
  gamertag: string
  /** Médiane des frags du joueur sur ce match, en mètres. */
  medianeM: number
  /** Écart à la médiane du lobby du match, en mètres (signé). */
  ecartM: number
  /** Frags mesurés du joueur sur ce match. */
  mesures: number
  /** `false` = point creux : sous le plancher, hors tendance et sans rôle. */
  plein: boolean
}

/** La suite des points d'UN joueur, dans l'ordre des matchs. */
export interface SeriePortee {
  xuid: string
  gamertag: string
  points: PointPortee[]
}

/** Les deux seuils de rôle : les quantiles 1/3 et 2/3 des points pleins. */
export interface SeuilsRoles {
  bas: number
  haut: number
}

/**
 * ordonnerProfils range les matchs du PLUS ANCIEN AU PLUS RÉCENT — l'axe des x de la carte.
 * Tri stable, départage par `match_id` pour que deux matchs au même horodatage gardent un
 * ordre déterministe (sinon l'axe change d'un rendu à l'autre).
 */
export function ordonnerProfils(profils: MatchRangeProfile[]): MatchRangeProfile[] {
  return [...profils].sort((a, b) => {
    const ta = Date.parse(a.played_at)
    const tb = Date.parse(b.played_at)
    if (Number.isFinite(ta) && Number.isFinite(tb) && ta !== tb) return ta - tb
    return a.match_id.localeCompare(b.match_id)
  })
}

/**
 * categoriesMatchs rend les étiquettes de l'axe X : « #3 · Streets », ou « #3 » sans carte.
 * MÊME GABARIT que les autres graphes par match de la page (`squadEfficiencyChart`) —
 * même séparateur, même troncature (`truncateMap`, 9 caractères).
 */
export function categoriesMatchs(profilsOrdonnes: MatchRangeProfile[]): string[] {
  return profilsOrdonnes.map((p, i) =>
    p.map_name ? `#${i + 1} · ${truncateMap(p.map_name)}` : `#${i + 1}`,
  )
}

/**
 * seriesPortee projette les profils ordonnés en une suite de points PAR JOUEUR.
 *
 * `ordreRoster` (gamertags, joueur principal en tête) fixe l'ordre des séries — donc
 * l'ordre de la légende et l'attribution des encres. La comparaison est insensible à la
 * casse : le serveur rend le gamertag tel que le titre l'écrit, l'URL le rend souvent en
 * minuscules. Un joueur servi hors roster est rendu en queue, jamais perdu.
 *
 * `grandeur` choisit ce qui est projeté en ordonnée (portée ou hauteur) : un (match,
 * joueur) sans mesure pour cette grandeur est ABSENT de la série, et un joueur qui n'en a
 * aucune n'a pas de série du tout — l'état vide de la carte en découle.
 */
export function seriesPortee(
  profilsOrdonnes: MatchRangeProfile[],
  ordreRoster: string[] = [],
  grandeur: GrandeurProfil = 'portee',
): SeriePortee[] {
  const parXuid = new Map<string, SeriePortee>()
  profilsOrdonnes.forEach((profil, ordre) => {
    for (const joueur of profil.players ?? []) {
      const mesure = mesureDeJoueur(joueur, grandeur)
      if (!mesure) continue
      const gamertag = joueur.gamertag ?? joueur.xuid
      let serie = parXuid.get(joueur.xuid)
      if (!serie) {
        serie = { xuid: joueur.xuid, gamertag, points: [] }
        parXuid.set(joueur.xuid, serie)
      }
      serie.points.push({
        ordre,
        matchId: profil.match_id,
        mapName: profil.map_name,
        xuid: joueur.xuid,
        gamertag,
        medianeM: mesure.medianeM,
        ecartM: mesure.ecartM,
        mesures: joueur.measured,
        plein: joueur.measured >= PLANCHER_MESURE,
      })
    }
  })
  const rang = new Map(ordreRoster.map((gt, i) => [gt.toLowerCase(), i]))
  return [...parXuid.values()].sort((a, b) => {
    const ra = rang.get(a.gamertag.toLowerCase()) ?? Number.MAX_SAFE_INTEGER
    const rb = rang.get(b.gamertag.toLowerCase()) ?? Number.MAX_SAFE_INTEGER
    if (ra !== rb) return ra - rb
    return a.gamertag.localeCompare(b.gamertag)
  })
}

/**
 * quantileLineaire — interpolation linéaire entre les deux rangs encadrants, la même règle
 * que la médiane servie côté Go (`analysis/weapon_range.go`, `percentileLinear`).
 */
export function quantileLineaire(valeursTriees: number[], q: number): number {
  if (valeursTriees.length === 0) return 0
  if (valeursTriees.length === 1) return valeursTriees[0]
  const pos = q * (valeursTriees.length - 1)
  const bas = Math.floor(pos)
  const haut = Math.ceil(pos)
  if (bas === haut) return valeursTriees[bas]
  return valeursTriees[bas] + (valeursTriees[haut] - valeursTriees[bas]) * (pos - bas)
}

/**
 * seuilsRoles rend les deux bornes des bandes de rôle : les tiers de TOUS les points PLEINS
 * de la période, tous joueurs confondus. `null` quand rien n'est mesuré — pas de bandes
 * plutôt que des bandes fabriquées.
 */
export function seuilsRoles(series: SeriePortee[]): SeuilsRoles | null {
  const ecarts = series
    .flatMap((s) => s.points)
    .filter((p) => p.plein)
    .map((p) => p.ecartM)
    .sort((a, b) => a - b)
  if (ecarts.length === 0) return null
  return { bas: quantileLineaire(ecarts, 1 / 3), haut: quantileLineaire(ecarts, 2 / 3) }
}

/** roleDeEcart classe un écart (en mètres) sur les bandes de la période. */
export function roleDeEcart(ecartM: number, seuils: SeuilsRoles): RoleDePortee {
  if (ecartM < seuils.bas) return 'front'
  if (ecartM >= seuils.haut) return 'sniper'
  return 'polyvalent'
}

/**
 * moyenneGlissante rend, pour chaque point d'un joueur, la moyenne des écarts des points
 * PLEINS de la fenêtre qui s'y termine — la ligne de tendance du nuage.
 *
 * `null` sur les `fenetre - 1` premiers points (la fenêtre n'est pas pleine) et quand la
 * fenêtre ne contient aucun point mesuré. Les points creux ne comptent ni au numérateur ni
 * au dénominateur : une médiane sur 3 frags n'est pas une médiane.
 */
export function moyenneGlissante(
  points: PointPortee[],
  fenetre: number = FENETRE_ROLE,
): (number | null)[] {
  return points.map((_, i) => {
    if (i < fenetre - 1) return null
    const pleins = points.slice(i - fenetre + 1, i + 1).filter((p) => p.plein)
    if (pleins.length === 0) return null
    return pleins.reduce((s, p) => s + p.ecartM, 0) / pleins.length
  })
}

/** Diamètres (px) des points du nuage — les deux bouts de l'échelle de `measured`. */
export const TAILLE_POINT_MIN = 7
export const TAILLE_POINT_MAX = 18

/**
 * taillePoint projette les frags mesurés d'un point sur la plage RÉELLE de la période
 * (`min`..`max`), comme le gros point du nuage d'isolement : une échelle absolue perdrait
 * tout écart dès que la sélection est petite. `min === max` → le milieu de la plage.
 */
export function taillePoint(mesures: number, min: number, max: number): number {
  if (max <= min) return (TAILLE_POINT_MIN + TAILLE_POINT_MAX) / 2
  const t = Math.min(1, Math.max(0, (mesures - min) / (max - min)))
  return TAILLE_POINT_MIN + t * (TAILLE_POINT_MAX - TAILLE_POINT_MIN)
}

/**
 * rolesFenetre rend le rôle LU SUR LA FENÊTRE GLISSANTE, point par point.
 *
 * `null` = pas encore de rôle. Trois causes, et une seule lecture à l'écran (jeton gris) :
 * la fenêtre n'est pas pleine (les `fenetre - 1` premiers matchs), le match lui-même est
 * creux, ou la fenêtre ne porte aucun point mesuré.
 */
export function rolesFenetre(
  points: PointPortee[],
  seuils: SeuilsRoles | null,
  fenetre: number = FENETRE_ROLE,
): (RoleDePortee | null)[] {
  const moyennes = moyenneGlissante(points, fenetre)
  return points.map((p, i) => {
    if (!seuils || !p.plein) return null
    const moyenne = moyennes[i]
    return moyenne == null ? null : roleDeEcart(moyenne, seuils)
  })
}
