/**
 * teamNames — résolution team_id → nom officiel d'équipe Halo Infinite.
 *
 * Port 1:1 de `src/config.py::TEAM_MAP` (branche main Python). Halo Infinite
 * définit 9 équipes nommées (Eagle / Cobra / ...) attribuées aux team_id 0..8
 * de manière fixe par 343 Industries — ce mapping ne dépend pas du match.
 *
 * Le backend expose `team_side` au format "t{N}" pour MatchScoreboard
 * (cf. buildTeamTabFull qui fait `fmt.Sprintf("t%d", *s.TeamID)`). D'autres
 * surfaces (picker média) exposent l'entier directement — utiliser alors
 * `resolveTeamNameFromID`.
 *
 * Module placé dans `lib/halo/` (et non `features/match-view/`) pour pouvoir
 * être consommé par plusieurs features sans couplage cross-feature.
 */

const TEAM_NAMES_HALO_INFINITE: Record<number, string> = {
  0: 'Eagle',
  1: 'Cobra',
  2: 'Hades',
  3: 'Valkyrie',
  4: 'Rampart',
  5: 'Cutlass',
  6: 'Valor',
  7: 'Hazard',
  8: 'Observer',
}

/**
 * Couleur d'identité (hex #RRGGBB) par team_id Halo Infinite — thème officiel
 * Arrowhead / SpartanRecord, fixe par 343 Industries (indépendant du match).
 *
 * Ces littéraux hex vivent ici (lib/halo/), PAS dans features/ : la règle
 * color-tokens interdit les hex dans features/components, mais un référentiel de
 * couleurs d'identité de jeu (au même titre que rarity.ts) est un cas légitime —
 * ce sont des données de domaine Halo, pas des choix de design UI. Le backend fournit
 * la couleur H5 par la donnée (team_color) ; Infinite n'a pas de référentiel serveur,
 * d'où cette map cliente.
 */
const TEAM_COLORS_HALO_INFINITE: Record<number, string> = {
  0: '#3B9DFF',
  1: '#FE3939',
  2: '#C43AAC',
  3: '#8D3AC4',
  4: '#49B8FE',
  5: '#DA3A04',
  6: '#FFEA00',
  7: '#DC5839',
  8: '#8AFFBE',
}

/**
 * Parse "t{N}" → N. Retourne null si le format n'est pas reconnu (équipe vide
 * ou ancienne convention). Idempotent face à null/undefined.
 */
export function parseTeamSideID(teamSide: string | null | undefined): number | null {
  if (!teamSide) return null
  const m = /^t(\d+)$/.exec(teamSide)
  if (!m) return null
  return Number(m[1])
}

/**
 * Résout team_side "t{N}" → nom officiel Halo Infinite. Retourne null si
 * team_id absent de la map (équipe non standard ou nouvelle non encore
 * référencée) ou si le format est invalide.
 */
export function resolveTeamName(teamSide: string | null | undefined): string | null {
  const id = parseTeamSideID(teamSide)
  if (id == null) return null
  return TEAM_NAMES_HALO_INFINITE[id] ?? null
}

/**
 * Variante directe : team_id (entier) → nom officiel, sans passer par le
 * format "t{N}". Utile pour les surfaces qui exposent déjà l'entier.
 */
export function resolveTeamNameFromID(teamID: number | null | undefined): string | null {
  if (teamID == null) return null
  return TEAM_NAMES_HALO_INFINITE[teamID] ?? null
}

/**
 * Résout la couleur d'identité (hex #RRGGBB) d'une équipe Halo Infinite par team_id.
 * Retourne null si team_id absent de la map (équipe non standard) ou nul → le caller
 * dégrade sur son accent d'équipe existant. La couleur H5, elle, arrive par la DONNÉE
 * (row.team_color) et n'a pas besoin de cette map.
 */
export function resolveTeamColorFromID(teamID: number | null | undefined): string | null {
  if (teamID == null) return null
  return TEAM_COLORS_HALO_INFINITE[teamID] ?? null
}

/**
 * Chemin de l'asset logo d'une équipe : `/titles/{slug}/teams/{teamId}.png`.
 * Résolution d'asset uniforme title-agnostic (le slug paramètre le chemin, ce n'est
 * pas une branche de comportement). Retourne null si l'un des deux est absent →
 * le caller n'affiche pas de logo. Certains team_id peuvent ne pas avoir d'asset :
 * l'onError du <img> gère le 404 côté rendu.
 */
export function teamLogoPath(
  slug: string | null | undefined,
  teamID: number | null | undefined,
): string | null {
  if (!slug || teamID == null) return null
  return `/titles/${slug}/teams/${teamID}.png`
}

/**
 * Détecte si un libellé d'équipe contient DÉJÀ le mot « équipe »/« team »
 * (accents et casse ignorés). Sert à éviter le double préfixe : le backend Halo 5
 * fournit un libellé déjà complet et localisé depuis team_colors (« Équipe Cobra »),
 * alors que les noms officiels résolus côté front (Eagle / Cobra) sont NUS et
 * attendent le préfixe « Équipe »/« Team » ajouté par teamLabelFmt.
 *
 * Title-agnostic : on teste uniquement la DONNÉE (le libellé), jamais le slug.
 * Aucun nom d'équipe officiel Halo (Eagle/Cobra/Hades/…) ne contient ces mots,
 * donc le test ne produit pas de faux positif sur les noms nus.
 */
export function labelHasTeamWord(name: string): boolean {
  const normalized = name
    .normalize('NFD')
    .replace(/\p{Diacritic}/gu, '')
    .toLowerCase()
  return /\b(equipe|team)\b/.test(normalized)
}
