/**
 * navL1Sections — définition partagée des sections de navigation L1.
 *
 * Source unique consommée par :
 * - NavL1 (rendu inline desktop, `≥ md`)
 * - NavL1MobileMenu (drawer mobile, `< md`)
 *
 * Garder la structure ici évite toute duplication entre les deux rendus.
 */
import type { ReactNode } from 'react'
import { isCommunityPath, type RouteTo } from './shellNavigation'
import { playerRelativePath } from '@/lib/title-routing'
import type { TitleCapability } from '@/lib/capabilities/capabilities'
import type { CommonManifestKey } from '@/lib/i18n/generated/common'

// Matcher de section : `re` s'applique au SUFFIXE relatif au joueur (playerRelativePath),
// jamais au pathname brut — aucun littéral `/players/` (garde-rail D-10). Retourne false
// hors scope joueur (page agnostique).
function under(pathname: string, re: RegExp): boolean {
  const suffix = playerRelativePath(pathname)
  return suffix !== null && re.test(suffix)
}

// ─── Icône flamme (label Ascension) ──────────────────────────────────────────
// Nœud JSX statique (pas un composant) pour garder ce module « data-only »
// (compatible react-refresh — il n'exporte que des constantes et des types).
const flameIcon: ReactNode = (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="13"
    height="13"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
  >
    <path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z" />
  </svg>
)

// ─── Définition des sections L1 ───────────────────────────────────────────────

export interface L1Tab {
  key: string
  /** Clé i18n du libellé (résolue par le consommateur via `t()`). */
  labelKey: CommonManifestKey
  /** Template de route typé (`$titleSlug`/`$playerSlug` résolus par `params`). */
  path: RouteTo
  /**
   * Capability produit requise pour afficher cet onglet (multi-titre, Phase 5).
   * Absente ⇒ onglet toujours visible. NO-OP pour `halo_infinite`.
   */
  capability?: TitleCapability
}

export interface L1Section {
  key: string
  /** Clé i18n du libellé (résolue par le consommateur via `t()`). */
  labelKey: CommonManifestKey
  /** Icône optionnelle affichée avant le label (ex: flamme pour Ascension). */
  icon?: ReactNode
  /** Route par défaut (template typé) lors du clic sur le label. */
  defaultPath: RouteTo
  /** Retourne true si le pathname courant appartient à cette section. */
  matchPathname: (pathname: string) => boolean
  /** Onglets du dropdown (optionnel — si absent, bouton simple). */
  tabs?: L1Tab[]
  /**
   * Capability produit requise pour afficher CETTE section dans la nav (multi-titre,
   * Phase 5). Absente ⇒ section toujours visible. NO-OP pour `halo_infinite` (déclare
   * toutes les capabilities). Les sections transverses (home, stats, squad, explorer,
   * community) restent non gatées : leur contenu dérive des matchs (suit matchmaking).
   */
  capability?: TitleCapability
}

// Refonte nav L1 (Phase 4 Prestige) :
// - Synthèse devient un onglet de Stats (transverse : sa famille naturelle)
// - Pass saisonnier devient un onglet de Carrière (progression temporelle)
// - Palmarès renommé en "Communauté" + ajout onglet Leaderboard PP
// - Nouvelle entrée L1 "Ascension" (page Prestige : Objectifs + Parcours)
//   Route path /objectifs conservé. Label rétabli "Objectifs" (≠ "Défis" in-game Halo).
export const L1_SECTIONS: L1Section[] = [
  {
    key: 'home',
    labelKey: 'common.nav.section_home',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
    matchPathname: (p) => under(p, /^\/home/),
  },
  {
    key: 'stats',
    labelKey: 'common.nav.section_solo',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/timeseries',
    matchPathname: (p) => under(p, /^\/(stats\/|synthesis)/),
    tabs: [
      { key: 'synthesis', labelKey: 'common.nav.tab_synthesis', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/synthesis' },
      { key: 'timeseries', labelKey: 'common.nav.tab_timeseries', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/timeseries' },
      { key: 'sessions', labelKey: 'common.nav.tab_sessions', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/sessions' },
    ],
  },
  {
    key: 'squad',
    labelKey: 'common.nav.section_squad',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies',
    matchPathname: (p) => under(p, /^\/squad/),
    tabs: [
      { key: 'synergies', labelKey: 'common.nav.tab_synergies', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies' },
      { key: 'contributions', labelKey: 'common.nav.tab_contributions', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/contributions' },
      { key: 'dynamique', labelKey: 'common.nav.tab_dynamique', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/dynamique' },
    ],
  },
  {
    key: 'career',
    labelKey: 'common.nav.section_career',
    capability: 'career',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/career',
    matchPathname: (p) => under(p, /^\/(career|citations|medals|profile)/),
    tabs: [
      { key: 'progression', labelKey: 'common.nav.tab_progression', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career' },
      { key: 'citations', labelKey: 'common.nav.tab_citations', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/citations' },
      { key: 'medals', labelKey: 'common.nav.tab_medals', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/medals' },
      {
        key: 'season-pass',
        labelKey: 'common.nav.tab_season_pass',
        path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/season-pass',
        capability: 'season_pass',
      },
    ],
  },
  {
    key: 'ascension',
    labelKey: 'common.nav.section_ascension',
    icon: flameIcon,
    // Ascension (profil LUSR + leviers + coaching) dérive entièrement du rating
    // LUSR ⇒ gatée sur `lusr`. (Reste aussi conditionnée par le réglage
    // `show_progression` côté NavL1.)
    capability: 'lusr',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/ascension',
    matchPathname: (p) => under(p, /^\/(objectifs|ascension)/),
    tabs: [
      { key: 'profile', labelKey: 'common.nav.tab_profile', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/ascension' },
      { key: 'objectives', labelKey: 'common.nav.tab_objectives', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/objectifs' },
      { key: 'coaching', labelKey: 'common.nav.tab_coaching', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/coaching' },
      { key: 'realisations', labelKey: 'common.nav.tab_realisations', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/realisations' },
    ],
  },
  {
    key: 'community',
    labelKey: 'common.nav.section_community',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/community',
    matchPathname: isCommunityPath,
    // Section transverse non gatée (Relations / Face-à-face dérivent des matchs) ;
    // seul l'onglet « Classements » dépend de `world.leaderboard`.
    tabs: [
      {
        key: 'leaderboard',
        labelKey: 'common.nav.tab_leaderboard',
        path: '/{-$lang}/t/$titleSlug/players/$playerSlug/community',
        capability: 'world.leaderboard',
      },
      { key: 'relations', labelKey: 'common.nav.tab_relations', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/community/relations' },
      { key: 'compare', labelKey: 'common.nav.tab_compare', path: '/{-$lang}/t/$titleSlug/players/$playerSlug/community/compare' },
    ],
  },
  {
    key: 'media',
    labelKey: 'common.nav.section_media',
    capability: 'media',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/media',
    matchPathname: (p) => under(p, /^\/media/),
  },
  {
    key: 'explorer',
    labelKey: 'common.nav.section_explorer',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer',
    matchPathname: (p) => under(p, /^\/explorer/),
  },
]
