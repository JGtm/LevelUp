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

  normal: '#9CA3AF',

  heroic: '#3B82F6',

  legendary: '#8B5CF6',

  mythic: '#F59E0B',
}

export const TIER_LABELS_FR: Record<Tier, string> = {
  normal: 'Normal',
  heroic: 'Héroïque',
  legendary: 'Légendaire',
  mythic: 'Mythique',
}

export const TIER_LABELS_EN: Record<Tier, string> = {
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
  /** Valeur courante mesurée (calculée par l'évaluateur dans ListActiveChallenges). 0 si non renseignée. */
  current_value?: number
  /** PP crédités à la complétion (PPForCompletion par tier/data_tier). 0/absent si data_tier=tracking. */
  pp_reward?: number
  expires_at?: string
  created_at: string
  committed_at?: string
  completed_at?: string
  expired_at?: string
  abandoned_at?: string
  last_palier_recompute_at?: string
  is_private: boolean
  /** true si ce défi provient d'un SquadChallenge. Absent/null pour les défis perso. */
  is_squad?: boolean
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
  /** Somme des PP des objectifs de l'arc (enrichi en lecture par le backend). */
  objectives_pp?: number
  /** Bonus PP crédité à la complétion de l'arc — distinct des PP des objectifs. */
  completion_bonus_pp?: number
}

// Non exporté : référencé via PresetArc.steps (exposé structurellement), aucun
// consommateur ne l'importe par son nom → évite un export inutilisé (knip).
interface PresetArcStep {
  preset_arc_id: string
  position: number
  template_id: string
  target_tier: Tier
}

export interface PresetArc {
  id: string
  title_slug: string
  title_en: string
  title_fr: string
  description_en?: string
  description_fr?: string
  schema_version: number
  updated_at: string
  /** Étapes hydratées par le backend (aperçu du picker). */
  steps?: PresetArcStep[]
}

export interface MomentCard {
  id: string
  challenge_id: string
  blob_path?: string
  created_at: string
}

export interface PrestigeLevel {
  index: number
  name: string
  threshold_pp: number
  next_threshold_pp: number
  progress_ratio: number
}

export interface UserPrestige {
  user_id: string
  title_slug: string
  total_pp: number
  current_level: number
  updated_at: string
  /** Détails du niveau courant (nom, prochain seuil, ratio). Présent depuis 2026-05-01. */
  level?: PrestigeLevel
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
  /** Fin du cooldown anti-farming sur la métrique pour le joueur courant
   *  (ISO 8601). Absent si aucun cooldown actif. Enrichi par le backend
   *  (SuggestTemplates) ; permet d'afficher un badge + désactiver le choix. */
  cooldown_ends_at?: string
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

/** Escouade (roster) — entité Squad côté backend (clé xuid). */
export interface Squad {
  id: string
  name: string
  created_by: string
  created_at: string
}

/** Membre d'escouade. `user_id` (player_slug) renseigné si le membre est un
 *  utilisateur de l'app (accès lecture/écriture aux objectifs) ; vide sinon.
 *  `gamertag` est un snapshot d'affichage (libellé au moment de l'ajout). */
export interface SquadMember {
  squad_id: string
  xuid: string
  user_id?: string
  gamertag?: string
  joined_at: string
}

/** Escouade + son roster (réponse de listMySquads).
 *  `usual_playlists` / `usual_modes` : indice dérivé des matchs communs réels du
 *  roster (top par fréquence, labels) — auto-adaptatif, jamais stocké. Absent
 *  si non calculé/indisponible. */
export interface SquadWithMembers {
  squad: Squad
  members: SquadMember[]
  usual_playlists?: string[]
  usual_modes?: string[]
}

/** Progression d'un membre sur un défi d'escouade (mode cumulatif). */
export interface SquadParticipantProgress {
  xuid: string
  value: number
  matches: number
  completed: boolean
}

/** Membre initial fourni à la création d'une escouade. */
export interface SquadMemberInput {
  xuid: string
  gamertag?: string
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
 * Préfixe player-scoped des routes Prestige/Escouade.
 *
 * Le module Prestige est monté côté serveur SOUS /players/{player_slug}
 * (server_apiv1.go — groupe gardé par ownershipMW, ADR 0029). L'acteur de la
 * requête (created_by / requested_by / user_id) sert de {player_slug} : le
 * handler lit l'acteur depuis le body/la query, le segment de path ne sert
 * qu'à ownershipMW. Un appel top-level (sans ce préfixe) tombe donc en 404
 * (cause du bug « Enregistrer cette compo » — les routes squad l'oubliaient
 * alors que les lectures défis/arcs/prestige le posaient déjà).
 *
 * Chokepoint unique — ne pas inliner `/players/${...}` pour une route squad
 * (garde-rail : prestige.paths.test.ts).
 */
const scopedToPlayer = (actorSlug: string) => `/players/${encodeURIComponent(actorSlug)}`

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
      `/players/${encodeURIComponent(userId)}/challenges?user_id=${encodeURIComponent(userId)}&title_slug=${encodeURIComponent(titleSlug)}`,
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
      `/players/${encodeURIComponent(userId)}/arcs?user_id=${encodeURIComponent(userId)}&title_slug=${encodeURIComponent(titleSlug)}`,
    ),

  getArc: (id: string) => api.get<Arc>(`/arcs/${id}`),

  /** Supprime un arc. cascade=true supprime aussi les objectifs (abandon, ou
   *  hard delete si l'arc a < 1h) ; cascade=false les détache (gardés, libres). */
  deleteArc: (id: string, userId: string, cascade: boolean) =>
    api.delete<void>(
      `/arcs/${id}?user_id=${encodeURIComponent(userId)}&objectives=${cascade ? 'delete' : 'detach'}`,
    ),

  // Presets d'arc (catalogue + adoption)
  listArcPresets: (userId: string, titleSlug: string) =>
    api.get<{ presets: PresetArc[]; count: number }>(
      `/arcs/presets?user_id=${encodeURIComponent(userId)}&title_slug=${encodeURIComponent(titleSlug)}`,
    ),

  adoptArcPreset: (presetId: string, userId: string, titleSlug: string) =>
    api.post<Arc>(`/arcs/presets/${encodeURIComponent(presetId)}/adopt`, {
      user_id: userId,
      title_slug: titleSlug,
    }),

  // Prestige (PP + niveau)
  getMyPrestige: (userId: string, titleSlug?: string) => {
    const qs = titleSlug
      ? `?user_id=${encodeURIComponent(userId)}&title_slug=${encodeURIComponent(titleSlug)}`
      : `?user_id=${encodeURIComponent(userId)}`
    return api.get<UserPrestige>(`/players/${encodeURIComponent(userId)}/prestige/me${qs}`)
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
  ) => api.post<SquadChallenge>(`${scopedToPlayer(body.created_by)}/squads/${squadId}/challenges`, body),

  listSquadChallenges: (squadId: string, requestedBy: string) =>
    api.get<{ squad_challenges: SquadChallenge[]; count: number }>(
      `${scopedToPlayer(requestedBy)}/squads/${squadId}/challenges`,
    ),

  joinSquadChallenge: (id: string, body: { user_id: string; chosen_tier?: Tier; is_private?: boolean }) =>
    api.post<void>(`${scopedToPlayer(body.user_id)}/squad-challenges/${id}/join`, body),

  // Squad roster (CRUD) — clé xuid, cf. backend Phase C.
  createSquad: (body: { name: string; created_by: string; members?: SquadMemberInput[] }) =>
    api.post<Squad>(`${scopedToPlayer(body.created_by)}/squads`, body),

  listMySquads: (userId: string) =>
    api.get<{ squads: SquadWithMembers[]; count: number }>(
      `${scopedToPlayer(userId)}/squads?user_id=${encodeURIComponent(userId)}`,
    ),

  addSquadMember: (
    squadId: string,
    body: { xuid: string; gamertag?: string; requested_by: string },
  ) => api.post<void>(`${scopedToPlayer(body.requested_by)}/squads/${encodeURIComponent(squadId)}/members`, body),

  removeSquadMember: (squadId: string, xuid: string, requestedBy: string) =>
    api.delete<void>(
      `${scopedToPlayer(requestedBy)}/squads/${encodeURIComponent(squadId)}/members/${encodeURIComponent(xuid)}?requested_by=${encodeURIComponent(requestedBy)}`,
    ),

  // Renommer une escouade (membre-user requis côté backend).
  renameSquad: (squadId: string, body: { name: string; requested_by: string }) =>
    api.patch<void>(`${scopedToPlayer(body.requested_by)}/squads/${encodeURIComponent(squadId)}`, body),

  // Supprimer une escouade (retrait append-only de tous les membres).
  deleteSquad: (squadId: string, requestedBy: string) =>
    api.delete<void>(
      `${scopedToPlayer(requestedBy)}/squads/${encodeURIComponent(squadId)}?requested_by=${encodeURIComponent(requestedBy)}`,
    ),

  // Évaluation de progression d'un défi d'escouade (recalcule + persiste).
  evaluateSquadChallenge: (id: string, requestedBy: string) =>
    api.post<{ progress: SquadParticipantProgress[] }>(
      `${scopedToPlayer(requestedBy)}/squad-challenges/${encodeURIComponent(id)}/evaluate`,
      { requested_by: requestedBy },
    ),

  // Pool de défis suggérés pour l'escouade (biaisé coach). À consommer pour créer.
  refreshSquadPool: (squadId: string, body: { title_slug: string; requested_by: string }) =>
    api.post<{ pool: Template[]; count: number }>(
      `${scopedToPlayer(body.requested_by)}/squads/${encodeURIComponent(squadId)}/challenges/pool/refresh`,
      body,
    ),

  // Orientation coach de l'escouade : axe focal (le plus faible) à renforcer.
  squadOrientation: (squadId: string, requestedBy: string) =>
    api.get<{ axis: string }>(
      `${scopedToPlayer(requestedBy)}/squads/${encodeURIComponent(squadId)}/orientation?requested_by=${encodeURIComponent(requestedBy)}`,
    ),
}
