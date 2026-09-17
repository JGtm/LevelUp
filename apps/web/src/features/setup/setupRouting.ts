/**
 * setupRouting — décide si l'utilisateur CONNECTÉ doit passer par le wizard de
 * mise en place, et quelle étape lui montrer (ADR 0035 D3, item 2.8).
 *
 * POURQUOI. `setup_required` et `setup_state` décrivent l'INSTANCE (« existe-t-il
 * au moins un profil ? »), pas l'utilisateur. Sur une instance déjà peuplée, un
 * compte qui vient de se connecter par SSO sans profil recevait donc
 * `setup_state: 'ready'` : il atterrissait sur la page « on synchronise tes
 * derniers matchs » — alors que depuis l'ADR 0035 plus rien ne tourne pour lui
 * (ni poller, ni sync) tant qu'un profil n'existe pas. Il faut le conduire là où
 * il peut en déclarer un.
 *
 * `available_players` est, lui, filtré par propriété côté serveur (ADR 0029) :
 * une liste vide pour un utilisateur connecté et non administrateur signifie
 * exactement « ce compte n'a aucun profil accessible ».
 *
 * Fonctions PURES : la décision ne vit pas dans les composants.
 */

/** Ce que la décision a besoin de savoir de l'utilisateur courant. */
export interface SetupAudience {
  authMode: 'none' | 'password' | 'xbox'
  currentUsername: string | null
  isAdmin: boolean
  availablePlayerCount: number
}

export const SETUP_PATH = '/setup'

/**
 * true si l'utilisateur connecté n'a AUCUN profil accessible et doit en déclarer
 * un avant que l'application ait quoi que ce soit à lui montrer.
 *
 * Faux quand :
 * - l'authentification n'est pas activée (mono-utilisateur, démo) : il n'y a pas
 *   de propriété, la liste n'est pas filtrée ;
 * - personne n'est connecté : la garde de login s'applique d'abord ;
 * - l'utilisateur est administrateur : il voit tous les profils de l'instance, et
 *   le cas « aucun profil du tout » est déjà couvert par `setup_required`.
 */
export function needsOwnProfile(audience: SetupAudience): boolean {
  if (audience.authMode === 'none') return false
  if (!audience.currentUsername) return false
  if (audience.isAdmin) return false
  return audience.availablePlayerCount === 0
}

/**
 * Chemin vers lequel rediriger l'utilisateur, ou null s'il peut rester où il est.
 * Ne redirige jamais depuis une page consultable sans compte, depuis les pages
 * d'authentification, ni depuis le wizard lui-même (anti-boucle).
 */
export function setupRedirectPath(
  audience: SetupAudience,
  currentPath: string,
  isAnonymousPath: (path: string) => boolean,
): string | null {
  if (!needsOwnProfile(audience)) return null
  if (currentPath === SETUP_PATH) return null
  if (currentPath === '/login' || currentPath === '/register') return null
  if (isAnonymousPath(currentPath)) return null
  return SETUP_PATH
}

/** Étapes du wizard, dans l'ordre du parcours. */
export type SetupStep = 'device_code' | 'player' | 'initial_sync' | 'done'

/**
 * Étape à afficher. `needsOwn` (l'utilisateur n'a pas de profil à lui) prime sur
 * l'état d'instance : sans identité Halo liée il faut d'abord la lier, sinon il
 * faut déclarer le profil — quel que soit le nombre de profils des AUTRES.
 */
export function resolveSetupStep(
  setupState: string | null | undefined,
  needsOwn: boolean,
  hasLinkedHaloIdentity: boolean,
): SetupStep {
  if (needsOwn) {
    return hasLinkedHaloIdentity ? 'player' : 'device_code'
  }
  switch (setupState) {
    case 'no_halo_link':
      return 'device_code'
    case 'halo_linked_no_profile':
      return 'player'
    case 'profile_ready_no_sync':
      return 'initial_sync'
    default:
      return 'done'
  }
}

/**
 * true si le wizard doit rendre la main à l'application. Un utilisateur sans
 * profil à lui n'en sort JAMAIS tout seul : sans cette exception il rebondirait
 * vers l'accueil (l'instance, elle, est « prête »), puis la garde de route le
 * renverrait ici — une boucle.
 */
export function shouldLeaveSetup(
  setupState: string | null | undefined,
  setupRequired: boolean,
  needsOwn: boolean,
): boolean {
  if (needsOwn) return false
  return setupState === 'ready' || !setupRequired
}
