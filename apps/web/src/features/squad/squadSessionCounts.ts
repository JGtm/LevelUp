/**
 * squadSessionCounts — SOURCE UNIQUE du compte de matchs d'une session
 * escouade (ADR 0033 backend `docs/adr/0033-squad-session-population-single-count.md`).
 *
 * Une même session affichait jusqu'à 3 nombres différents selon la surface :
 * la pastille du rail L2 venait de `/filters/resolve` (population du JOUEUR
 * PRINCIPAL, aveugle à la composition et à `filter_exact_composition`), le
 * SessionMultiSelect venait de `composition_sessions` (population RÉELLE de la
 * page). Règle : `composition_sessions[].match_count` (population « commencés
 * ensemble », post-filtre composition exacte) est la SEULE source d'un compte
 * de session en contexte escouade — `/filters/resolve` ne sert qu'au repli
 * tant que la réponse teammates n'est pas arrivée.
 *
 * Absorbe l'ancien `mergeSessionCounts` de `squadPending.ts` (une seule règle
 * de compte côté web, CLAUDE.md règle 6) — enrichi de `match_count_roster`
 * (compte AVANT le filtre composition exacte, `total`) pour publier l'écart
 * (D1 : « 4 sur 7 »).
 */

/** Repli /filters/resolve : population du joueur principal, cascade seule. */
export interface SquadSessionFallbackEntry {
  label: string
  match_count_filtered: number
}

/** Sous-ensemble structurel de FilterContextResolved consommé par
 *  resolveSquadSessionFallback — seul point du module (et de features/squad/,
 *  cf. singleCountSource.guard.test.ts) à connaître la forme exacte du champ
 *  volatil de /filters/resolve qui porte le repli de chargement. */
interface ResolvedWithSessionOptions {
  session_options?: { all_sessions?: SquadSessionFallbackEntry[] | null } | null
}

/**
 * Extrait la liste de repli (population du joueur principal) depuis les deux
 * sources possibles de la page : le preview live (mis à jour à la volée) en
 * priorité, sinon le contexte résolu commité (figé jusqu'au clic Analyser).
 * Centralisé ICI (et nulle part ailleurs dans features/squad/) pour que le
 * champ volatil de /filters/resolve n'ait qu'un seul lecteur.
 */
export function resolveSquadSessionFallback(
  previewResolve: ResolvedWithSessionOptions | null | undefined,
  resolvedContext: ResolvedWithSessionOptions | null | undefined,
): SquadSessionFallbackEntry[] {
  return previewResolve?.session_options?.all_sessions ?? resolvedContext?.session_options?.all_sessions ?? []
}

/** Entrée composition_sessions consommée ici — sous-ensemble structurel de
 *  domain.CompositionSessionEntry (généré dans lib/api/generated.ts). */
export interface SquadSessionCompositionEntry {
  label: string
  match_count?: number
  match_count_roster?: number
}

/** Compte affiché pour une session : `shown` = population réelle (source
 *  unique, ce que les tableaux/graphes de la page consomment) ; `total` =
 *  compte AVANT le filtre composition exacte (même valeur que `shown` hors
 *  écart — rien à publier). */
export interface SquadSessionCount {
  shown: number
  total: number
}

/**
 * Résout le compte d'UNE session : `composition_sessions` fait foi dès qu'elle
 * porte une entrée exploitable (match_count > 0) pour ce label — y compris
 * quand son nombre est plus bas que le repli (c'est exactement le cas de la
 * composition exacte, ADR 0033). Le repli `/filters/resolve` ne s'applique que
 * pour un label ABSENT de `composition_sessions` (réponse teammates pas encore
 * arrivée, ou session que la composition n'a jamais jouée).
 */
export function squadSessionCount(
  label: string,
  compositionSessions: SquadSessionCompositionEntry[],
  fallback: SquadSessionFallbackEntry[],
): SquadSessionCount | undefined {
  const entry = compositionSessions.find((s) => s.label === label)
  if (entry && typeof entry.match_count === 'number' && entry.match_count > 0) {
    return { shown: entry.match_count, total: entry.match_count_roster ?? entry.match_count }
  }
  const fb = fallback.find((s) => s.label === label)
  if (fb && fb.match_count_filtered > 0) {
    return { shown: fb.match_count_filtered, total: fb.match_count_filtered }
  }
  return undefined
}

/**
 * Variante "un seul nombre" pour les consommateurs qui n'affichent QUE la
 * population réelle sans publier l'écart (ex. SessionMultiSelect : masquage
 * des sessions vides + nombre par ligne — même module, même règle, jamais une
 * seconde lecture de `composition_sessions`).
 */
export function squadSessionShownCount(
  label: string,
  compositionSessions: SquadSessionCompositionEntry[],
  fallback: SquadSessionFallbackEntry[],
): number | undefined {
  return squadSessionCount(label, compositionSessions, fallback)?.shown
}
