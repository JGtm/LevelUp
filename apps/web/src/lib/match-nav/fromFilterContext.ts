/**
 * fromFilterContext — convertit le `FilterContextInput` du globalFilterStore
 * en `MatchFilterSpec` consommable par /neighbors (Phase 2c).
 *
 * Mapping :
 *   - cascade.playlists → playlist_names (multi, Phase 3 — IN (...) côté backend)
 *   - cascade.modes     → mode_categories (multi, Phase 3 — OR des préfixes)
 *   - period.start_date → date_from (concat avec T00:00:00Z)
 *   - period.end_date   → date_to   (concat avec T23:59:59Z)
 *   - sessions.picked_session_label → session_id (si présent)
 *
 * Cas non gérés (volontaire) :
 *   - cascade.experience_types et cascade.maps : pas mappables vers MatchFilterSpec.
 *   - outcome : pas dans filterContext global, c'est un filtre spécifique
 *     match-history/explorer (à brancher au cas par cas).
 *
 * Multi-titres : ce helper est neutre — il transmet tel quel les valeurs
 * (le backend ignore silencieusement mode_category inconnu côté title).
 */
import type { FilterContextInput } from '@/lib/api/types'
import type { MatchFilterSpec } from './navContext'

/**
 * Construit un MatchFilterSpec à partir du contexte de filtres global.
 * Retourne null si aucun champ n'est mappable.
 */
export function filterContextToMatchFilterSpec(
  ctx: FilterContextInput | null | undefined,
  options?: {
    /** Outcome local supplémentaire (pas dans filterContext). */
    outcome?: 'win' | 'loss' | 'draw' | 'dnf'
  },
): MatchFilterSpec | null {
  if (!ctx) return null

  const spec: MatchFilterSpec = {}

  // Playlists (multi) : toutes les playlists sélectionnées (Phase 3).
  const playlists = (ctx.cascade?.playlists ?? []).filter(Boolean)
  if (playlists.length > 0) {
    spec.playlist_names = playlists
  }

  // Catégories de mode (multi) : idem.
  const modes = (ctx.cascade?.modes ?? []).filter(Boolean)
  if (modes.length > 0) {
    spec.mode_categories = modes
  }

  // Period dates : on suppose YYYY-MM-DD côté store, on rajoute T00:00:00Z / T23:59:59Z
  if (ctx.period?.start_date) {
    spec.date_from = appendTime(ctx.period.start_date, '00:00:00')
  }
  if (ctx.period?.end_date) {
    spec.date_to = appendTime(ctx.period.end_date, '23:59:59')
  }

  // Sessions : on prend le label dominant si pickedSession unique
  const session = pickFirstSessionLabel(ctx)
  if (session) {
    spec.session_id = session
  }

  // Outcome : optionnel, supplémentaire au filterContext (vient du caller)
  if (options?.outcome) {
    spec.outcome = options.outcome
  }

  if (
    !spec.playlist_names?.length &&
    !spec.mode_categories?.length &&
    !spec.date_from &&
    !spec.date_to &&
    !spec.session_id &&
    !spec.outcome
  ) {
    return null
  }
  return spec
}

function appendTime(date: string, time: string): string {
  // Si la string contient déjà un T (ISO complet), on retourne tel quel.
  if (date.includes('T')) return date
  return `${date}T${time}Z`
}

function pickFirstSessionLabel(ctx: FilterContextInput): string | null {
  if (ctx.filter_mode !== 'sessions') return null
  const s = ctx.sessions
  if (!s) return null
  // Priorité : single picked → solo → squad
  if (s.picked_session_label) return s.picked_session_label
  if (s.picked_solo_session_label) return s.picked_solo_session_label
  if (s.picked_squad_session_label) return s.picked_squad_session_label
  if (s.picked_sessions && s.picked_sessions.length === 1 && s.picked_sessions[0]) {
    return s.picked_sessions[0]
  }
  return null
}
