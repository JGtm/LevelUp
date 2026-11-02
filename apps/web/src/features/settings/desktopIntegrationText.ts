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
  saved: 'Saved.',
  saveFailed: 'The setting could not be saved.',
}

const TEXT: Record<Locale, DesktopIntegrationText> = { fr: FR, en: EN }

export function getDesktopIntegrationText(locale: Locale): DesktopIntegrationText {
  return TEXT[locale]
}
