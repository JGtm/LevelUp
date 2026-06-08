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
export {
  challengeKeys,
  useChallenges,
  useCreateChallenge,
  useUpdateChallenge,
  useAbandonChallenge,
} from './hooks/useChallenges'
export { arcKeys, useArcs, useCreateArc, useDeleteArc } from './hooks/useArcs'
export {
  prestigeKeys,
  useMyPrestige,
  useSuggestedTemplates,
  useJoinSquadChallenge,
} from './hooks/usePrestige'
// cross-feature-allow: les hooks profil/campagne vivent désormais dans
// features/ascension/profile (refonte 2026-05-26). Re-broadcast ici pour les
// callers prestige existants.
export {
  profileKeys,
  useActiveCampaign,
  useCampaignMutations,
  usePlayerProfile,
} from '@/features/ascension/profile/queries'
