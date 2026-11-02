/**
 * Textes de la page « Télécharger pour Windows » — FR et EN côte à côte.
 *
 * Dictionnaire de feature (`Record<Locale, T>`) : de la prose, pas des libellés d'interface. La mention
 * imposée par Microsoft n'est PAS ici : son texte reste en anglais dans toutes les langues
 * (`lib/appLinks`, rendue par `MicrosoftNotice`).
 */
import type { Locale } from '@/lib/i18n/locale'

export interface DownloadText {
  title: string
  intro: string
  highlights: readonly string[]
  action: string
  actionHint: string
  requirements: string
  smartScreenHeading: string
  smartScreenIntro: string
  smartScreenSteps: readonly string[]
  smartScreenOutro: string
  dataHeading: string
  dataBody: string
  unavailable: string
}

export const DOWNLOAD_TEXT: Record<Locale, DownloadText> = {
  fr: {
    title: 'Télécharger LevelUp pour Windows',
    intro:
      'Une application Windows gratuite et sans publicité qui fait tourner LevelUp sur votre PC : vos statistiques, le rejeu 2D et les films de vos parties, sans serveur à gérer.',
    highlights: [
      'Tout reste sur votre ordinateur : vos données, vos jetons de connexion, vos films.',
      "Les mises à jour vous sont proposées dans l'application, jamais imposées.",
      'Le code source est public, sous licence AGPL-3.0.',
    ],
    action: 'Télécharger la dernière version',
    actionHint: 'Sur la page de la version, choisissez le fichier « LevelUp_…_x64-setup.exe » (environ 200 Mo).',
    requirements: "Windows 10 ou 11, 64 bits. Installation pour votre seul compte, sans droits d'administrateur.",
    smartScreenHeading: 'Avertissement de Windows SmartScreen',
    smartScreenIntro:
      "L'installeur n'est pas signé par un certificat de code (ils sont payants, et ce projet est gratuit). À chaque installation, Windows peut afficher « Windows a protégé votre ordinateur ». Ce n'est pas la détection d'un virus : Windows ne connaît simplement pas encore cet éditeur. Pour continuer :",
    smartScreenSteps: ['Cliquez sur « Informations complémentaires ».', 'Cliquez sur « Exécuter quand même ».'],
    smartScreenOutro:
      "Les mises à jour de l'application, elles, sont vérifiées par la clé de signature du projet, indépendamment de Windows.",
    dataHeading: 'Vos données',
    dataBody:
      'Elles vivent dans le dossier %LOCALAPPDATA%\\LevelUp et y restent après une désinstallation : réinstaller LevelUp les retrouve.',
    unavailable: "Cette page n'est proposée que par l'instance hébergée de LevelUp.",
  },
  en: {
    title: 'Download LevelUp for Windows',
    intro:
      'A free, ad-free Windows application that runs LevelUp on your own PC: your statistics, the 2D replay and the films of your matches, with no server to manage.',
    highlights: [
      'Everything stays on your computer: your data, your sign-in tokens, your films.',
      'Updates are offered inside the application, never forced.',
      'The source code is public, under the AGPL-3.0 licence.',
    ],
    action: 'Download the latest version',
    actionHint: 'On the release page, pick the file “LevelUp_…_x64-setup.exe” (about 200 MB).',
    requirements: 'Windows 10 or 11, 64-bit. Installs for your account only, no administrator rights needed.',
    smartScreenHeading: 'Windows SmartScreen warning',
    smartScreenIntro:
      'The installer is not signed with a code-signing certificate (they cost money, and this project is free). At each installation Windows may show “Windows protected your PC”. This is not a virus detection: Windows simply does not know this publisher yet. To continue:',
    smartScreenSteps: ['Click “More info”.', 'Click “Run anyway”.'],
    smartScreenOutro:
      "The application's updates, on the other hand, are checked against the project's own signing key, independently of Windows.",
    dataHeading: 'Your data',
    dataBody:
      'It lives in the %LOCALAPPDATA%\\LevelUp folder and stays there after an uninstall: reinstalling LevelUp finds it again.',
    unavailable: 'This page is only offered by the hosted LevelUp instance.',
  },
}
