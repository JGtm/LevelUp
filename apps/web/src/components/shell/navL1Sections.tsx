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
  label: string
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
  label: string
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
    label: 'Accueil',
    defaultPath: '/players/$playerSlug/home',
    matchPathname: (p) => /\/players\/[^/]+\/home/.test(p),
  },
  {
    key: 'stats',
    label: 'Solo',
    defaultPath: '/players/$playerSlug/stats/timeseries',
    matchPathname: (p) => /\/players\/[^/]+\/(stats\/|synthesis)/.test(p),
    tabs: [
      { key: 'synthesis', label: 'Synthèse', path: '/players/$playerSlug/synthesis' },
      { key: 'timeseries', label: 'Séries temporelles', path: '/players/$playerSlug/stats/timeseries' },
      { key: 'sessions', label: 'Sessions', path: '/players/$playerSlug/stats/sessions' },
    ],
  },
  {
    key: 'squad',
    label: 'Escouade',
    defaultPath: '/players/$playerSlug/squad/synergies',
    matchPathname: (p) => /\/players\/[^/]+\/squad/.test(p),
    tabs: [
      { key: 'synergies', label: 'Synergies', path: '/players/$playerSlug/squad/synergies' },
      { key: 'contributions', label: 'Contributions', path: '/players/$playerSlug/squad/contributions' },
    ],
  },
  {
    key: 'career',
    label: 'Carrière',
    capability: 'career',
    defaultPath: '/players/$playerSlug/career',
    matchPathname: (p) => /\/players\/[^/]+\/(career|citations|profile)/.test(p),
    tabs: [
      { key: 'progression', label: 'Progression', path: '/players/$playerSlug/career' },
      { key: 'citations', label: 'Citations', path: '/players/$playerSlug/citations' },
      {
        key: 'season-pass',
        label: 'Pass saisonnier',
        path: '/players/$playerSlug/career/season-pass',
        capability: 'season_pass',
      },
    ],
  },
  {
    key: 'ascension',
    label: 'Ascension',
    icon: flameIcon,
    // Ascension (profil LUSR + leviers + coaching) dérive entièrement du rating
    // LUSR ⇒ gatée sur `lusr`. (Reste aussi conditionnée par le réglage
    // `show_progression` côté NavL1.)
    capability: 'lusr',
    defaultPath: '/players/$playerSlug/ascension',
    matchPathname: (p) => /\/players\/[^/]+\/(objectifs|ascension)/.test(p),
    tabs: [
      { key: 'profile', label: 'Profil & objectifs', path: '/players/$playerSlug/ascension' },
      { key: 'coaching', label: 'Entraînement', path: '/players/$playerSlug/ascension/coaching' },
      { key: 'realisations', label: 'Réalisations', path: '/players/$playerSlug/ascension/realisations' },
    ],
  },
  {
    key: 'community',
    label: 'Communauté',
    defaultPath: '/players/$playerSlug/community',
    matchPathname: isCommunityPath,
    // Section transverse non gatée (Relations / Face-à-face dérivent des matchs) ;
    // seul l'onglet « Classements » dépend de `world.leaderboard`.
    tabs: [
      {
        key: 'leaderboard',
        label: 'Classements',
        path: '/players/$playerSlug/community',
        capability: 'world.leaderboard',
      },
      { key: 'relations', label: 'Relations', path: '/players/$playerSlug/community/relations' },
      { key: 'compare', label: 'Face-à-face', path: '/players/$playerSlug/community/compare' },
    ],
  },
  {
    key: 'media',
    label: 'Médias',
    capability: 'media',
    defaultPath: '/players/$playerSlug/media',
    matchPathname: (p) => /\/players\/[^/]+\/media/.test(p),
  },
  {
    key: 'explorer',
    label: 'Explorer',
    defaultPath: '/players/$playerSlug/explorer',
    matchPathname: (p) => /\/players\/[^/]+\/explorer/.test(p),
  },
]
