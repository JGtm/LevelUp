/**
 * replayPreferences.ts — LE SEUL ENDROIT qui touche localStorage pour les préférences du
 * rejeu. Patron NÉ dans useReplaySound (son : coupé/volume) : clé simple, lecture sous
 * try — navigation privée ou storage refusé dégradent en silence vers la valeur par
 * défaut, jamais une erreur affichée ni un plantage.
 *
 * CENTRALISÉ ICI (CLAUDE.md règle 6, « à la 3e copie, centraliser ») : le tiroir de
 * réglages (décision utilisateur du 16/08) ajoute au son persisté les calques, la vitesse
 * et le filtre de catégories — quatre à six clés qui partagent EXACTEMENT ce patron.
 * useReplaySound et useReplaySettings l'importent tous les deux plutôt que de le
 * recopier une troisième fois chacun de leur côté.
 */

/** Lit un booléen persisté ; absent ou storage indisponible -> `fallback`. */
export function readStoredFlag(key: string, fallback: boolean): boolean {
  try {
    const raw = localStorage.getItem(key)
    return raw === null ? fallback : raw === 'true'
  } catch {
    return fallback
  }
}

/**
 * Lit un nombre persisté, accepté seulement si `isValid` le confirme (bornes ou
 * appartenance à un ensemble autorisé — ex. les multiplicateurs de vitesse) ; absent,
 * invalide ou storage indisponible -> `fallback`.
 */
export function readStoredNumber(
  key: string,
  fallback: number,
  isValid: (value: number) => boolean,
): number {
  try {
    const raw = localStorage.getItem(key)
    if (raw === null) return fallback
    const value = Number(raw)
    return Number.isFinite(value) && isValid(value) ? value : fallback
  } catch {
    return fallback
  }
}

/** Persiste une préférence (booléen ou nombre déjà tourné en chaîne, JSON pour une liste). */
export function persistPreference(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* stockage refusé (navigation privée) : la préférence ne survit pas, le réglage marche. */
  }
}
