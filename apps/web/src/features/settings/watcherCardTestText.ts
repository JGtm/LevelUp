/**
 * watcherCardTestText.ts — LA fixture i18n des tests de la carte watcher.
 *
 * Extraite de `WatcherCard.test.tsx` le 2026-09-17 : le fichier de test franchissait le
 * seuil des 500 lignes (CLAUDE.md n° 5) et ces lignes étaient une table de données sans
 * aucune décision. Elle vit donc à côté, et le test ne porte plus que des tests.
 *
 * ELLE NE PORTE QUE LES CLÉS `watcher*`, c'est-à-dire EXACTEMENT celles que
 * `WatcherCard.tsx` et `watcherPresence.ts` lisent (vérifié au grep) — la version inline
 * en traînait une cinquantaine d'autres (sync, Discord, médias, backfill) qu'aucun rendu
 * de cette carte ne touche. `as unknown as SettingsText` : c'est donc un SOUS-ENSEMBLE
 * assumé ; ajouter ici la clé de tout nouveau libellé lu par la carte.
 */
import type { SettingsText } from './i18n'

export const WATCHER_TEST_TEXT = {
  watcherTitle: 'Détection de présence',
  watcherPresenceEnabled: 'Détection automatique',
  watcherPresenceDescription: 'Description',
  watcherAuthButton: 'Connecter via Xbox',
  watcherAuthReconnect: 'Rafraîchir Xbox',
  watcherAuthInstructions: 'Rendez-vous sur {url}',
  watcherAuthCopyCode: 'Copier le code',
  watcherAuthCodeCopied: 'Code copié',
  watcherAuthOpenLink: 'Ouvrir le lien',
  watcherAuthPending: 'En attente…',
  watcherAuthSuccess: 'Connexion réussie !',
  watcherAuthFailed: 'Échec de la connexion.',
  watcherTokenValid: 'Jeton valide jusqu\'au {date} ({gamertag})',
  watcherTokenExpired: 'Jeton expiré',
  watcherTokenMissing: 'Aucun jeton Xbox',
  watcherPlayersLabel: 'Joueurs surveillés',
  watcherPlayersAll: 'Tous les joueurs',
  watcherSubscriptionsUpdated: 'Mis à jour',
  watcherRtaConnected: 'RTA connecté',
  watcherRtaDisconnected: 'RTA déconnecté',
  watcherSubscribeError: 'Échec surveillance',
  watcherStateIdle: 'Absent',
  watcherStateWatching: 'En surveillance',
  watcherStateSyncing: 'Synchronisation',
  watcherStateCooling: 'Cooldown',
  watcherInGame: 'En jeu',
  watcherPresenceOnline: 'En ligne',
  watcherPresenceAway: 'Absent',
  watcherPresenceOffline: 'Hors-ligne',
  watcherPresenceUnknown: '—',
  watcherTitleXboxDashboard: "l'accueil Xbox",
  watcherLastSeenRelative: 'Vu il y a {duration} sur {title}',
  watcherLastSeenAbsolute: 'Vu le {date} sur {title}',
  watcherNeverSeen: 'Jamais vu en jeu',
} as unknown as SettingsText
