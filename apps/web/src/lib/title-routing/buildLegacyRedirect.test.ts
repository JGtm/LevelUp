/**
 * Tests — buildLegacyRedirect (module title-routing, D-5/D-11).
 *
 * Fonction PURE : mappe une URL legacy /players/… vers /t/{activeSlug}/players/…
 * (préfixe titre) + reproduit VERBATIM les redirections internes legacy
 * existantes (objectifs→ascension/objectifs, palmares/*→community/*,
 * compare→community/compare, synthesis→stats/synthesis,
 * citations→career/citations, commendations→career/commendations).
 *
 * Matrice COMPLÈTE table-driven : chaque famille de routes + préservation
 * ?search + #hash. La vérification navigateur (Phase 3) ne fait que confirmer.
 */
import { describe, it, expect } from 'vitest'
import { buildLegacyRedirect } from './buildLegacyRedirect'

const A = 'halo_infinite' // activeSlug par défaut des cas

interface Case {
  name: string
  pathname: string
  search?: string
  hash?: string
  active?: string
  expected: string | null
}

const CASES: Case[] = [
  // --- Préfixe simple : suffixe conservé tel quel -------------------------
  { name: 'home', pathname: '/players/jgtm/home', expected: '/t/halo_infinite/players/jgtm/home' },
  { name: 'stats index', pathname: '/players/jgtm/stats', expected: '/t/halo_infinite/players/jgtm/stats' },
  { name: 'stats/timeseries', pathname: '/players/jgtm/stats/timeseries', expected: '/t/halo_infinite/players/jgtm/stats/timeseries' },
  { name: 'stats/sessions', pathname: '/players/jgtm/stats/sessions', expected: '/t/halo_infinite/players/jgtm/stats/sessions' },
  { name: 'stats/synthesis', pathname: '/players/jgtm/stats/synthesis', expected: '/t/halo_infinite/players/jgtm/stats/synthesis' },
  { name: 'career hub', pathname: '/players/jgtm/career', expected: '/t/halo_infinite/players/jgtm/career' },
  { name: 'career/citations', pathname: '/players/jgtm/career/citations', expected: '/t/halo_infinite/players/jgtm/career/citations' },
  { name: 'career/commendations', pathname: '/players/jgtm/career/commendations', expected: '/t/halo_infinite/players/jgtm/career/commendations' },
  { name: 'career/season-pass', pathname: '/players/jgtm/career/season-pass', expected: '/t/halo_infinite/players/jgtm/career/season-pass' },
  { name: 'squad index', pathname: '/players/jgtm/squad', expected: '/t/halo_infinite/players/jgtm/squad' },
  { name: 'squad/synergies', pathname: '/players/jgtm/squad/synergies', expected: '/t/halo_infinite/players/jgtm/squad/synergies' },
  { name: 'squad/contributions', pathname: '/players/jgtm/squad/contributions', expected: '/t/halo_infinite/players/jgtm/squad/contributions' },
  { name: 'community index', pathname: '/players/jgtm/community', expected: '/t/halo_infinite/players/jgtm/community' },
  { name: 'community/compare', pathname: '/players/jgtm/community/compare', expected: '/t/halo_infinite/players/jgtm/community/compare' },
  { name: 'community/prestige', pathname: '/players/jgtm/community/prestige', expected: '/t/halo_infinite/players/jgtm/community/prestige' },
  { name: 'community/relations', pathname: '/players/jgtm/community/relations', expected: '/t/halo_infinite/players/jgtm/community/relations' },
  { name: 'media', pathname: '/players/jgtm/media', expected: '/t/halo_infinite/players/jgtm/media' },
  { name: 'explorer', pathname: '/players/jgtm/explorer', expected: '/t/halo_infinite/players/jgtm/explorer' },
  { name: 'matches/$matchId', pathname: '/players/jgtm/matches/abc123', expected: '/t/halo_infinite/players/jgtm/matches/abc123' },
  { name: 'matches/$matchId/replay', pathname: '/players/jgtm/matches/abc123/replay', expected: '/t/halo_infinite/players/jgtm/matches/abc123/replay' },
  { name: 'ascension index', pathname: '/players/jgtm/ascension', expected: '/t/halo_infinite/players/jgtm/ascension' },
  { name: 'ascension/coaching', pathname: '/players/jgtm/ascension/coaching', expected: '/t/halo_infinite/players/jgtm/ascension/coaching' },
  { name: 'ascension/objectifs (route réelle)', pathname: '/players/jgtm/ascension/objectifs', expected: '/t/halo_infinite/players/jgtm/ascension/objectifs' },
  { name: 'ascension/realisations', pathname: '/players/jgtm/ascension/realisations', expected: '/t/halo_infinite/players/jgtm/ascension/realisations' },
  { name: 'notifications', pathname: '/players/jgtm/notifications', expected: '/t/halo_infinite/players/jgtm/notifications' },

  // --- Joueur seul (pas de suffixe) → home --------------------------------
  { name: 'joueur seul', pathname: '/players/jgtm', expected: '/t/halo_infinite/players/jgtm/home' },
  { name: 'joueur seul (trailing slash)', pathname: '/players/jgtm/', expected: '/t/halo_infinite/players/jgtm/home' },

  // --- Redirections internes legacy (remaps reproduits verbatim) ----------
  { name: 'legacy citations → career/citations', pathname: '/players/jgtm/citations', expected: '/t/halo_infinite/players/jgtm/career/citations' },
  { name: 'legacy commendations → career/commendations', pathname: '/players/jgtm/commendations', expected: '/t/halo_infinite/players/jgtm/career/commendations' },
  { name: 'legacy synthesis → stats/synthesis', pathname: '/players/jgtm/synthesis', expected: '/t/halo_infinite/players/jgtm/stats/synthesis' },
  { name: 'legacy compare → community/compare', pathname: '/players/jgtm/compare', expected: '/t/halo_infinite/players/jgtm/community/compare' },
  { name: 'legacy objectifs → ascension/objectifs', pathname: '/players/jgtm/objectifs', expected: '/t/halo_infinite/players/jgtm/ascension/objectifs' },
  { name: 'legacy palmares (index) → community', pathname: '/players/jgtm/palmares', expected: '/t/halo_infinite/players/jgtm/community' },
  { name: 'legacy palmares/prestige → community/prestige', pathname: '/players/jgtm/palmares/prestige', expected: '/t/halo_infinite/players/jgtm/community/prestige' },
  { name: 'legacy palmares/relations → community/relations', pathname: '/players/jgtm/palmares/relations', expected: '/t/halo_infinite/players/jgtm/community/relations' },

  // --- Préservation ?search + #hash (au moins un par forme) ---------------
  { name: 'home + ?f=', pathname: '/players/jgtm/home', search: '?f=abc123', expected: '/t/halo_infinite/players/jgtm/home?f=abc123' },
  { name: 'stats/timeseries + #hash', pathname: '/players/jgtm/stats/timeseries', hash: '#top', expected: '/t/halo_infinite/players/jgtm/stats/timeseries#top' },
  { name: 'explorer + ?f= composite', pathname: '/players/jgtm/explorer', search: '?f=xyz&page=2', expected: '/t/halo_infinite/players/jgtm/explorer?f=xyz&page=2' },
  { name: 'legacy compare + ?target=', pathname: '/players/jgtm/compare', search: '?target=foo', expected: '/t/halo_infinite/players/jgtm/community/compare?target=foo' },
  { name: 'legacy palmares/prestige + #hash', pathname: '/players/jgtm/palmares/prestige', hash: '#lb', expected: '/t/halo_infinite/players/jgtm/community/prestige#lb' },
  { name: 'matches/replay + ?search + #hash', pathname: '/players/jgtm/matches/abc/replay', search: '?ts=5', hash: '#frag', expected: '/t/halo_infinite/players/jgtm/matches/abc/replay?ts=5#frag' },
  { name: 'joueur seul + ?f=', pathname: '/players/jgtm', search: '?f=abc', expected: '/t/halo_infinite/players/jgtm/home?f=abc' },

  // --- Robustesse : search/hash SANS marqueur → marqueur ajouté ----------
  { name: 'search sans ? → ? ajouté', pathname: '/players/jgtm/home', search: 'f=abc', expected: '/t/halo_infinite/players/jgtm/home?f=abc' },
  { name: 'hash sans # → # ajouté', pathname: '/players/jgtm/home', hash: 'top', expected: '/t/halo_infinite/players/jgtm/home#top' },
  { name: 'search/hash vides ignorés', pathname: '/players/jgtm/home', search: '?', hash: '#', expected: '/t/halo_infinite/players/jgtm/home' },

  // --- activeSlug variable ------------------------------------------------
  { name: 'home sur halo_5', pathname: '/players/jgtm/home', active: 'halo_5', expected: '/t/halo_5/players/jgtm/home' },
  { name: 'legacy palmares sur halo_5', pathname: '/players/jgtm/palmares', active: 'halo_5', expected: '/t/halo_5/players/jgtm/community' },

  // --- null : hors périmètre legacy ---------------------------------------
  { name: 'bare /players → null', pathname: '/players', expected: null },
  { name: 'bare /players/ → null', pathname: '/players/', expected: null },
  { name: 'page agnostique /settings → null', pathname: '/settings', expected: null },
  { name: 'déjà title-scoped /t/... → null', pathname: '/t/halo_5/players/jgtm/home', expected: null },
  { name: 'admin → null', pathname: '/admin/titles', expected: null },
]

describe('buildLegacyRedirect (matrice table-driven)', () => {
  for (const c of CASES) {
    it(c.name, () => {
      const out = buildLegacyRedirect(c.pathname, c.search ?? '', c.hash ?? '', c.active ?? A)
      if (c.expected === null) {
        expect(out).toBeNull()
      } else {
        expect(out).toEqual({ href: c.expected })
      }
    })
  }
})
