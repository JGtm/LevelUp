/**
 * Textes de l'intégration au poste de l'application de bureau — FR et EN.
 *
 * Dictionnaire de feature : chaque langue de `config/locales.toml` figure ici (parité par typage :
 * `Record<Locale, DesktopIntegrationText>`). Les refus du serveur arrivent rédigés dans la langue de la
 * requête : la carte les affiche tels quels.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface DesktopIntegrationText {
  title: string
  intro: string
  gameWatchLabel: string
  gameWatchHint: string
  gameRunning: string
  gameNotRunning: string
  nativeNotificationsLabel: string
  nativeNotificationsHint: string
  closeToTrayLabel: string
  closeToTrayHint: string
  startWithWindowsLabel: string
  startWithWindowsHint: string
  saved: string
  saveFailed: string
}

const FR: DesktopIntegrationText = {
  title: 'Intégration au poste',
  intro:
    "Ces réglages ne concernent que cette application de bureau. Rien n'est imposé : tout ce qui touche à Windows est désactivé tant que tu ne l'actives pas.",
  gameWatchLabel: 'Détecter le jeu en cours',
  gameWatchHint:
    "LevelUp regarde la liste des processus de Windows (le nom du programme, rien d'autre : jamais la mémoire du jeu). Pendant une partie, la construction des rejeux et les tâches lourdes attendent la fin ; à la fermeture du jeu, une synchronisation démarre.",
  gameRunning: 'Le jeu tourne : les tâches lourdes attendent la fin de la partie.',
  gameNotRunning: "Le jeu n'est pas lancé.",
  nativeNotificationsLabel: 'Notifications Windows',
  nativeNotificationsHint:
    "Les notifications de la cloche (synchronisation, défis, séries, records, rang) s'affichent aussi comme notifications Windows quand la fenêtre est réduite, masquée ou derrière une autre application. Jamais quand LevelUp est au premier plan.",
  closeToTrayLabel: 'Réduire dans la zone de notification à la fermeture',
  closeToTrayHint:
    "La croix de la fenêtre réduit LevelUp dans la zone de notification au lieu de le quitter : le serveur continue de tourner. Un clic sur l'icône rouvre la fenêtre ; son menu propose Ouvrir, Synchroniser maintenant et Quitter. Désactivé, la croix quitte LevelUp.",
  startWithWindowsLabel: 'Démarrer avec Windows',
  startWithWindowsHint:
    "LevelUp se lance, réduit dans la zone de notification, à l'ouverture de ta session Windows. L'entrée de démarrage est la tienne seulement (jamais pour toute la machine) ; la désactiver la supprime.",
  saved: 'Enregistré.',
  saveFailed: "Le réglage n'a pas pu être enregistré.",
}

const EN: DesktopIntegrationText = {
  title: 'Desktop integration',
  intro:
    'These settings only apply to this desktop application. Nothing is forced: everything that touches Windows stays off until you turn it on.',
  gameWatchLabel: 'Detect the running game',
  gameWatchHint:
    "LevelUp looks at Windows' list of processes (the program name, nothing else: never the game's memory). During a match, replay building and heavy tasks wait for it to end; when the game closes, a sync starts.",
  gameRunning: 'The game is running: heavy tasks wait for the match to end.',
  gameNotRunning: 'The game is not running.',
  nativeNotificationsLabel: 'Windows notifications',
  nativeNotificationsHint:
    'Bell notifications (sync, challenges, streaks, records, rank) also show as Windows notifications when the window is minimized, hidden or behind another application. Never while LevelUp is in the foreground.',
  closeToTrayLabel: 'Minimize to the notification area when closing',
  closeToTrayHint:
    'The window close button minimizes LevelUp to the notification area instead of quitting: the server keeps running. Click the icon to reopen the window; its menu offers Open, Sync now and Quit. When off, the close button quits LevelUp.',
  startWithWindowsLabel: 'Start with Windows',
  startWithWindowsHint:
    'LevelUp starts, minimized in the notification area, when your Windows session opens. The startup entry is yours only (never machine-wide); turning this off deletes it.',
  saved: 'Saved.',
  saveFailed: 'The setting could not be saved.',
}

const TEXT: Record<Locale, DesktopIntegrationText> = { fr: FR, en: EN }

export function getDesktopIntegrationText(locale: Locale): DesktopIntegrationText {
  return TEXT[locale]
}
