/**
 * Prestige API — types et helpers pour le module Prestige (défis, arcs, PP, leaderboard).
 *
 * Aligné sur le contrat Go internal/prestige (Phase 1+2+3+4 backend).
 * Les types reflètent la sérialisation JSON du package Go ; toute évolution
 * du backend doit être répercutée ici.
 *
 * Le câblage des routes côté serveur est conditionnel à PRESTIGE_ENABLED —
 * les appels peuvent retourner 404 si le flag est désactivé. Les UI doivent
 * gérer gracieusement (skeleton + message de feature désactivée).
 */
import { api } from './api/client'

// ──────────────────────── Enums ────────────────────────

export type ChallengeStatus =
  | 'draft'
  | 'active'
  | 'completed'
  | 'expired'
  | 'abandoned'
  | 'archived'

export type Tier = 'normal' | 'heroic' | 'legendary' | 'mythic'

export type Cadence = 'daily' | 'weekly' | 'monthly' | 'free'

export type EvalType = 'threshold' | 'cumulative'

export type WindowType = 'session' | 'rolling_days' | 'deadline' | 'matches_internal'

export type ChallengeMode = 'libre' | 'pilote'

export type DataTier = 'full' | 'estimated' | 'tracking'

export type SquadMode = 'collective' | 'competitive'

// ──────────────────────── Couleurs paliers ────────────────────────

/**
 * Couleurs des paliers selon Annexe B / Annexe F du plan conceptuel.
 *
 * Ces hex sont **autorisés** par exception explicite à la règle CLAUDE.md
 * sur les couleurs : ce sont des couleurs identitaires de gamification
 * (référence Halo Infinite cosmétiques rareté). Documentées dans le doc
 * de plan PLAN_challenges_xp_system.md.
 */
export const TIER_COLORS: Record<Tier, string> = {
  // eslint-disable-next-line no-restricted-syntax -- couleurs identitaires Prestige
  normal: '#9CA3AF',
  // eslint-disable-next-line no-restricted-syntax
  heroic: '#3B82F6',
  // eslint-disable-next-line no-restricted-syntax
  legendary: '#8B5CF6',
  // eslint-disable-next-line no-restricted-syntax
  mythic: '#F59E0B',
}

export const TIER_LABELS_FR: Record<Tier, string> = {
  normal: 'Normal',
  heroic: 'Heroic',
  legendary: 'Legendary',
  mythic: 'Mythic',
}

// ──────────────────────── Domaine ────────────────────────

export interface Challenge {
  id: string
  user_id: string
  title_slug: string
  arc_id?: string
  position?: number
  template_id?: string
  metric: string
  target: number
  target_per_member?: number
  window_type: WindowType
  window_value?: string
  cadence: Cadence
  eval_type: EvalType
  mode: ChallengeMode
  tier?: Tier
  data_tier: DataTier
  label?: string
  status: ChallengeStatus
  created_at: string
  committed_at?: string
  completed_at?: string
  expired_at?: string
  abandoned_at?: string
  last_palier_recompute_at?: string
  is_private: boolean
}

export interface Arc {
  id: string
  user_id: string
  title_slug: string
  title: string
  description?: string
  is_preset: boolean
  preset_id?: string
  created_at: string
  completed_at?: string
}

export interface MomentCard {
  id: string
  challenge_id: string
  blob_path?: string
  created_at: string
}

export interface UserPrestige {
  user_id: string
  title_slug: string
  total_pp: number
  current_level: number
  updated_at: string
}

export interface Template {
  id: string
  title_slug: string
  metric: string
  window_type: WindowType
  window_value?: string
  cadence: Cadence
  eval_type: EvalType
  mode_filter: string
  label_en: string
  label_fr: string
  description_en?: string
  description_fr?: string
  normal_target: number
  heroic_target: number
  legendary_target: number
  mythic_target: number
  schema_version: number
  updated_at: string
}

export interface SquadChallenge {
  id: string
  squad_id: string
  template_id?: string
  title_slug: string
  mode: SquadMode
  eval_type: EvalType
  window_type: WindowType
  window_value?: string
  target_per_member?: number
  expires_at?: string
  created_by: string
  created_at: string
}

// ──────────────────────── DTOs requête ────────────────────────

export interface CreateChallengeBody {
  user_id: string
  title_slug: string
  arc_id?: string
  template_id?: string
  metric: string
  target: number
  window_type: WindowType
  window_value?: string
  cadence: Cadence
  eval_type: EvalType
  mode: ChallengeMode
  label?: string
  is_private?: boolean
  target_per_member?: number
  position?: number
}

export interface UpdateChallengeBody {
  target?: number
  label?: string
}

export interface CreateArcBody {
  user_id: string
  title_slug: string
  title: string
  description?: string
}

// ──────────────────────── API client ────────────────────────

/**
 * Section api.prestige — toutes les requêtes vers les endpoints Prestige.
 *
 * Si le backend retourne 404 (PRESTIGE_ENABLED désactivé), les fonctions
 * propagent l'erreur ; les hooks React Query l'interpréteront comme
 * "feature désactivée".
 */
export const prestigeApi = {
  // Défis
  createChallenge: (body: CreateChallengeBody) =>
    api.post<Challenge>('/challenges', body),

  getChallenge: (id: string) => api.get<Challenge>(`/challenges/${id}`),

  listActiveChallenges: (userId: string, titleSlug: string) =>
    api.get<{ challenges: Challenge[]; count: number }>(
      `/challenges?user_id=${encodeURIComponent(userId)}&title_slug=${encodeURIComponent(titleSlug)}`,
    ),

  updateChallenge: (id: string, body: UpdateChallengeBody) =>
    api.patch<Challenge>(`/challenges/${id}`, body),

  abandonChallenge: (id: string) => api.delete<void>(`/challenges/${id}`),

  suggestNext: (id: string) =>
    api.post<{ suggestions: Template[] }>(`/challenges/${id}/suggest-next`),

  // Arcs
  createArc: (body: CreateArcBody) => api.post<Arc>('/arcs', body),

  listArcs: (userId: string, titleSlug: string) =>
    api.get<{ arcs: Arc[]; count: number }>(
      `/arcs?user_id=${encodeURIComponent(userId)}&title_slug=${encodeURIComponent(titleSlug)}`,
    ),

  getArc: (id: string) => api.get<Arc>(`/arcs/${id}`),

  // Prestige (PP + niveau)
  getMyPrestige: (userId: string, titleSlug?: string) => {
    const qs = titleSlug
      ? `?user_id=${encodeURIComponent(userId)}&title_slug=${encodeURIComponent(titleSlug)}`
      : `?user_id=${encodeURIComponent(userId)}`
    return api.get<UserPrestige>(`/prestige/me${qs}`)
  },

  // Templates
  suggestTemplates: (userId: string, titleSlug: string, count = 3) =>
    api.get<{ templates: Template[] }>(
      `/templates/suggest?user_id=${encodeURIComponent(userId)}&title_slug=${encodeURIComponent(titleSlug)}&count=${count}`,
    ),

  // Squad
  createSquadChallenge: (
    squadId: string,
    body: {
      template_id?: string
      title_slug: string
      mode: SquadMode
      eval_type: EvalType
      window_type: WindowType
      window_value?: string
      target_per_member?: number
      created_by: string
    },
  ) => api.post<SquadChallenge>(`/squads/${squadId}/challenges`, body),

  listSquadChallenges: (squadId: string) =>
    api.get<{ squad_challenges: SquadChallenge[]; count: number }>(
      `/squads/${squadId}/challenges`,
    ),

  joinSquadChallenge: (id: string, body: { user_id: string; chosen_tier?: Tier; is_private?: boolean }) =>
    api.post<void>(`/squad-challenges/${id}/join`, body),
}
