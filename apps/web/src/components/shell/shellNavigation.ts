import type { FileRouteTypes } from '@/routeTree.gen'
import { playerRelativePath } from '@/lib/title-routing'

/**
 * Cible de route valide du routeur généré (union des `to`). Typer les champs `to` des
 * items de nav ainsi (et non `string`) fait ENTRER la nav L1/L2 dans le typecheck : un
 * chemin inexistant devient une erreur tsc au lieu d'un lien mort silencieux. Source
 * unique réutilisée par NavL1/NavL2/navL1Sections.
 */
export type RouteTo = FileRouteTypes['to']

// NOTE (I18, 2026-07-24) : `ShellNavItem`/`ShellUtilityLink` et les tables
// `PLAYER_PRIMARY_NAV_ITEMS` / `PLAYER_SECONDARY_NAV_ITEMS` / `GLOBAL_SHELL_LINKS`
// (labels FR figés, eyebrow/description) ont été supprimées ici : leur SEUL
// consommateur restant était `lib/pageTitle.ts` (dérivation du titre d'onglet), qui
// vient de basculer sur sa propre table locale-aware (Record<Locale,string>). La nav
// réelle (L1/L2, tabs) vit depuis longtemps dans `navL1Sections.tsx` + `NavL2.tsx`,
// locale-aware via `commonManifest` — ces exports n'y étaient plus branchés (0 code
// mort, règle CLAUDE.md n°7).

/**
 * Section Communauté : pages /community + les legacy /palmares (hors season-pass,
 * passé sous Carrière) et /compare (Face-à-face). Raisonne sur le SUFFIXE relatif
 * au joueur (playerRelativePath) — aucun littéral `/players/` (garde-rail D-10).
 *
 * Source unique partagée par NavL1 (surlignage du bouton) et NavL2 (sous-onglets).
 */
export function isCommunityPath(pathname: string): boolean {
  const suffix = playerRelativePath(pathname)
  if (suffix === null) return false
  const community = /^\/community(?:\/|$)/.test(suffix)
  const legacy =
    (/^\/palmares(?:\/|$)/.test(suffix) && !/^\/palmares\/season-pass/.test(suffix)) ||
    /^\/compare(?:\/|$)/.test(suffix)
  return community || legacy
}

/**
 * Verdict de routage de la route index ('/'). Fonction pure (logique hors
 * composant, règle 7) : IndexPage se contente de projeter le résultat.
 *
 * Gardes, du plus prioritaire au moins prioritaire :
 *  1. `wait`   — bootstrap pas encore hydraté. Avant hydratation `authMode` vaut son
 *               défaut 'none', donc la garde `login` ne serait pas fiable : on attend.
 *  2. `login`  — auth requise (password|xbox) mais session anonyme. Typiquement juste
 *               après « Se déconnecter » (qui recharge sur '/'). En anonyme xbox
 *               `available_players` est vide (filtrage ownership, ADR 0029) : sans
 *               cette garde on tombait sur `setup`, rendu via le <Outlet/> nu de
 *               __root — donc SANS NavL1 — et la redirection impérative de __root
 *               (navigate dans un useEffect) pouvait se perdre dans la course avec le
 *               settle initial du routeur au rechargement plein, bloquant l'utilisateur
 *               sur une page sans barre de nav ni lien de reconnexion.
 *  3. `player` — un joueur est actif (courant, sinon 1er disponible) : sa home.
 *  4. `setup`  — aucun joueur configuré : inviter à configurer l'application.
 */
export type IndexRedirect =
  | { kind: 'wait' }
  | { kind: 'login' }
  | { kind: 'player'; slug: string }
  | { kind: 'setup' }

export interface IndexRedirectInput {
  isBootstrapped: boolean
  authMode: 'none' | 'password' | 'xbox'
  currentUsername: string | null
  currentPlayerSlug?: string | null
  firstAvailablePlayerSlug?: string | null
}

export function resolveIndexRedirect(input: IndexRedirectInput): IndexRedirect {
  if (!input.isBootstrapped) {
    return { kind: 'wait' }
  }
  const authRequired = input.authMode === 'password' || input.authMode === 'xbox'
  if (authRequired && !input.currentUsername) {
    return { kind: 'login' }
  }
  const slug = input.currentPlayerSlug ?? input.firstAvailablePlayerSlug
  if (slug) {
    return { kind: 'player', slug }
  }
  return { kind: 'setup' }
}

/**
 * Décision de navigation lors d'un changement de JOUEUR actif (même titre). Pure
 * (logique hors composant, règle 7) : NavL1 projette le résultat.
 *  - `same-route` : on est sous une sous-page joueur → rester sur la même route et ne
 *    changer que le param `playerSlug` (préserve la section, le titre ET la langue).
 *  - `home` : on n'est pas sous une sous-page joueur (page agnostique, ou racine
 *    joueur nue) → aller à l'accueil title-scoped du nouveau joueur.
 */
export type PlayerSwitchNav = { kind: 'same-route' } | { kind: 'home' }

export function resolvePlayerSwitch(pathname: string): PlayerSwitchNav {
  // suffix null (page agnostique) OU '' (racine joueur nue) → home ; sinon on préserve
  // la sous-page en ne changeant que le playerSlug.
  return playerRelativePath(pathname) ? { kind: 'same-route' } : { kind: 'home' }
}

/**
 * Filet joueur (D-8, trou n°1 de la revue v2) — décision pure projetée par le layout
 * joueur au fresh-load (une fois `isBootstrapped`). Le beforeLoad du layout couvre les
 * navigations SPA (store chaud) mais NE re-tourne PAS sur un simple re-render : ce
 * résolveur ferme le cas fresh-load où le store s'hydrate après le matching.
 *  - `index`    : aucun joueur disponible → retour à l'index (onboarding).
 *  - `ok`       : le slug d'URL existe → rendre la page.
 *  - `redirect` : slug d'URL inconnu → premier joueur disponible (même titre).
 */
export type PlayerFallback = { kind: 'ok' } | { kind: 'redirect'; slug: string } | { kind: 'index' }

export function resolvePlayerFallback(
  playerSlug: string,
  availablePlayers: readonly { player_slug: string }[],
): PlayerFallback {
  if (availablePlayers.length === 0) return { kind: 'index' }
  if (availablePlayers.some((p) => p.player_slug === playerSlug)) return { kind: 'ok' }
  return { kind: 'redirect', slug: availablePlayers[0].player_slug }
}

/**
 * Chemins consultables SANS compte, en dehors des écrans d'authentification
 * eux-mêmes (/login, /register, traités séparément dans `routes/__root.tsx`).
 *
 * Aujourd'hui : la politique de confidentialité. Un visiteur de la démo publique
 * doit pouvoir lire ce qui est fait de ses données AVANT de décider de connecter
 * son compte Microsoft — l'éjecter vers /login rendrait le lien du pied de page
 * inutilisable et la page inatteignable.
 *
 * Le préfixe est comparé avec sa frontière de segment : `/privacy-autre` n'est
 * PAS anonyme.
 */
const ANONYMOUS_PATHS: readonly string[] = ['/privacy']

export function isAnonymousPath(pathname: string): boolean {
  return ANONYMOUS_PATHS.some((p) => pathname === p || pathname.startsWith(`${p}/`))
}
