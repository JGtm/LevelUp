/**
 * Types DTO du coach_advisor (ADR 0020 Phase 9 HTTP).
 *
 * Aligné sur internal/api/handlers/coach_proposals.go proposalDTO.
 */

export type ProposalKind = 'challenge' | 'arc'

export type ProposalStatus =
  | 'pending'
  | 'accepted'
  | 'dismissed'
  | 'superseded'
  | 'obsoleted'
  | 'stale'

export type ProposalOrigin = 'catalog' | 'synthesized'

export interface CoachProposal {
  id: string
  kind: ProposalKind
  template_id?: string
  suggested_tier?: string
  source_signal: string
  source_metric?: string
  radar_axis?: string
  strength: number
  origin: ProposalOrigin
  /** Clé d'i18n du motif, dans la langue de la requête. */
  reason_key?: string
  reason_params?: string // JSON-encoded
  status: ProposalStatus
  created_at: string
  resolved_at?: string | null
  resolved_ref?: string
}

export interface ProposalsListResponse {
  items: CoachProposal[]
}

export interface AcceptResponse {
  status: 'accepted'
  challenge_id?: string
  arc_id?: string
  challenge_ids?: string[]
}

export interface DismissResponse {
  status: 'dismissed'
}
