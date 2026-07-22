/**
 * Re-export des hooks Prestige séparés.
 *
 * Les hooks sont organisés par domaine dans `hooks/` :
 *  - useChallenges.ts : queries + mutations sur défis
 *  - useArcs.ts       : queries + mutations sur arcs
 *  - usePrestige.ts   : prestige/me, templates, squad
 *
 * Ce fichier rebroadcaste tout pour les imports existants.
 */
// NB : les clés de requête (ex-challengeKeys/arcKeys/prestigeKeys/profileKeys) sont
// centralisées dans `@/lib/query/keys` (queryKeys.challenge/arc/prestige/playerProfile) —
// L5, CLAUDE.md n°13. Ce barrel ne rebroadcaste plus que les hooks.
export {
  useChallenges,
  useChallengeHistory,
  useCreateChallenge,
  useUpdateChallenge,
  useAbandonChallenge,
  usePilotMode,
} from './hooks/useChallenges'
export {
  useArcs,
  useCreateArc,
  useDeleteArc,
  useArcPresets,
  useAdoptArcPreset,
} from './hooks/useArcs'
export {
  useMyPrestige,
  useSuggestedTemplates,
  useJoinSquadChallenge,
} from './hooks/usePrestige'
// cross-feature-allow: les hooks profil/campagne vivent désormais dans
// features/ascension/profile (refonte 2026-05-26). Re-broadcast ici pour les
// callers prestige existants.
export {
  useActiveCampaign,
  useCampaignMutations,
  usePlayerProfile,
} from '@/features/ascension/profile/queries'
