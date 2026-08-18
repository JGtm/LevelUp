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

/**
 * Lit un CHOIX persisté parmi un ensemble fermé (ex. la lecture de la carte de chaleur :
 * présence ou éliminations) ; absent, hors liste ou storage indisponible -> `fallback`.
 *
 * MÊME EXIGENCE QUE `readStoredNumber` : une valeur écrite par une AUTRE version de l'app —
 * ou par n'importe qui, `localStorage` est ouvert — ne doit jamais entrer dans l'état. Ce
 * qui n'est pas dans la liste n'existe pas.
 */
export function readStoredChoice<T extends string>(
  key: string,
  fallback: T,
  allowed: readonly T[],
): T {
  try {
    const raw = localStorage.getItem(key)
    return raw !== null && (allowed as readonly string[]).includes(raw) ? (raw as T) : fallback
  } catch {
    return fallback
  }
}

/**
 * LES ABONNÉS D'UNE CLÉ, et pourquoi ils sont apparus (2026-08-18, item R3.7).
 *
 * Jusqu'ici UN SEUL composant lisait chaque préférence : l'état React local du hook suffisait.
 * Les FICHES COMPACTES cassent cette hypothèse — la bascule vit dans le tiroir (sous le
 * canvas), les fiches vivent dans une AUTRE colonne de la page. Deux `useState` initialisés
 * du même `localStorage` ne se parlent pas : la bascule aurait bougé sans que les fiches
 * changent, jusqu'au prochain rechargement.
 *
 * L'ÉVÉNEMENT `storage` DU NAVIGATEUR NE RÉPOND PAS À CE BESOIN : il ne se déclenche QUE
 * pour les AUTRES onglets, jamais pour celui qui écrit. C'est le piège classique, et c'est
 * pour cela que ce registre existe plutôt qu'un abonnement à `window`.
 */
const abonnes = new Map<string, Set<(value: string) => void>>()

/** subscribePreference — écoute une clé ; rend la fonction qui se désabonne. */
export function subscribePreference(key: string, fn: (value: string) => void): () => void {
  const set = abonnes.get(key) ?? new Set()
  abonnes.set(key, set)
  set.add(fn)
  return () => set.delete(fn)
}

/** Persiste une préférence (booléen ou nombre déjà tourné en chaîne, JSON pour une liste). */
export function persistPreference(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* stockage refusé (navigation privée) : la préférence ne survit pas, le réglage marche. */
  }
  // Les abonnés sont prévenus MÊME si le stockage a refusé : le réglage doit marcher dans
  // la session, c'est sa survie au rechargement qui est perdue, pas son effet.
  for (const fn of abonnes.get(key) ?? []) fn(value)
}
