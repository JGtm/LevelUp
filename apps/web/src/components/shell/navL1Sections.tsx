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
import { isCommunityPath } from './shellNavigation'
import type { TitleCapability } from '@/lib/capabilities/capabilities'
import type { CommonManifestKey } from '@/lib/i18n/generated/common'

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
  /** Chemin avec $playerSlug en placeholder. */
  path: string
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
  /** Route par défaut lors du clic sur le label (avec $playerSlug en placeholder). */
  defaultPath: string
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
    defaultPath: '/players/$playerSlug/home',
    matchPathname: (p) => /\/players\/[^/]+\/home/.test(p),
  },
  {
    key: 'stats',
    labelKey: 'common.nav.section_solo',
    defaultPath: '/players/$playerSlug/stats/timeseries',
    matchPathname: (p) => /\/players\/[^/]+\/(stats\/|synthesis)/.test(p),
    tabs: [
      { key: 'synthesis', labelKey: 'common.nav.tab_synthesis', path: '/players/$playerSlug/stats/synthesis' },
      { key: 'timeseries', labelKey: 'common.nav.tab_timeseries', path: '/players/$playerSlug/stats/timeseries' },
      { key: 'sessions', labelKey: 'common.nav.tab_sessions', path: '/players/$playerSlug/stats/sessions' },
    ],
  },
  {
    key: 'squad',
    labelKey: 'common.nav.section_squad',
    defaultPath: '/players/$playerSlug/squad/synergies',
    matchPathname: (p) => /\/players\/[^/]+\/squad/.test(p),
    tabs: [
      { key: 'synergies', labelKey: 'common.nav.tab_synergies', path: '/players/$playerSlug/squad/synergies' },
      { key: 'contributions', labelKey: 'common.nav.tab_contributions', path: '/players/$playerSlug/squad/contributions' },
    ],
  },
  {
    key: 'career',
    labelKey: 'common.nav.section_career',
    capability: 'career',
    defaultPath: '/players/$playerSlug/career',
    matchPathname: (p) => /\/players\/[^/]+\/(career|citations|profile)/.test(p),
    tabs: [
      { key: 'progression', labelKey: 'common.nav.tab_progression', path: '/players/$playerSlug/career' },
      { key: 'citations', labelKey: 'common.nav.tab_citations', path: '/players/$playerSlug/career/citations' },
      {
        key: 'season-pass',
        labelKey: 'common.nav.tab_season_pass',
        path: '/players/$playerSlug/career/season-pass',
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
    defaultPath: '/players/$playerSlug/ascension',
    matchPathname: (p) => /\/players\/[^/]+\/(objectifs|ascension)/.test(p),
    tabs: [
      { key: 'profile', labelKey: 'common.nav.tab_profile', path: '/players/$playerSlug/ascension' },
      { key: 'objectives', labelKey: 'common.nav.tab_objectives', path: '/players/$playerSlug/ascension/objectifs' },
      { key: 'coaching', labelKey: 'common.nav.tab_coaching', path: '/players/$playerSlug/ascension/coaching' },
      { key: 'realisations', labelKey: 'common.nav.tab_realisations', path: '/players/$playerSlug/ascension/realisations' },
    ],
  },
  {
    key: 'community',
    labelKey: 'common.nav.section_community',
    defaultPath: '/players/$playerSlug/community',
    matchPathname: isCommunityPath,
    // Section transverse non gatée (Relations / Face-à-face dérivent des matchs) ;
    // seul l'onglet « Classements » dépend de `world.leaderboard`.
    tabs: [
      {
        key: 'leaderboard',
        labelKey: 'common.nav.tab_leaderboard',
        path: '/players/$playerSlug/community',
        capability: 'world.leaderboard',
      },
      { key: 'relations', labelKey: 'common.nav.tab_relations', path: '/players/$playerSlug/community/relations' },
      { key: 'compare', labelKey: 'common.nav.tab_compare', path: '/players/$playerSlug/community/compare' },
    ],
  },
  {
    key: 'media',
    labelKey: 'common.nav.section_media',
    capability: 'media',
    defaultPath: '/players/$playerSlug/media',
    matchPathname: (p) => /\/players\/[^/]+\/media/.test(p),
  },
  {
    key: 'explorer',
    labelKey: 'common.nav.section_explorer',
    defaultPath: '/players/$playerSlug/explorer',
    matchPathname: (p) => /\/players\/[^/]+\/explorer/.test(p),
  },
]
