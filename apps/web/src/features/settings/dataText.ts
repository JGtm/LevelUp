/**
 * Textes de la carte « Données » de l'application de bureau (lot L22) — FR et EN.
 *
 * Dictionnaire de feature : chaque langue de `config/locales.toml` figure ici (parité par typage :
 * `Record<Locale, DataText>`). Un refus du serveur arrive avec un `code` fixe ; le texte est écrit
 * ici, dans la langue de l'interface, et jamais lu dans le message de l'erreur.
 */
import type { Locale } from '@/lib/i18n/locale'
import type { BackupInterval } from './dataLogic'

export interface DataText {
  title: string
  intro: string
  backupHeading: string
  backupDirLabel: string
  backupDirPlaceholder: string
  backupDirHint: string
  save: string
  sameVolumeWarning: string
  intervalLabel: string
  intervals: Record<BackupInterval, string>
  /** Gabarit d'une fréquence que la liste n'offre pas : {hours} (nombre d'heures). */
  intervalEvery: string
  keepLabel: string
  keepHint: string
  runNow: string
  running: string
  runStarted: string
  lastLabel: string
  lastNone: string
  /** L'état n'est pas arrivé (lecture en cours). */
  lastLoading: string
  /** La lecture de l'état a échoué : on ne sait pas s'il existe une sauvegarde. */
  lastUnavailable: string
  /** Gabarit : {date}, {size}, {count} (nombre de bases), {duration}. */
  lastSummary: string
  databasesOne: string
  databasesMany: string
  lastPartial: string
  /** Gabarit : {date}, {reason}. */
  lastAttemptFailed: string
  attemptErrors: Record<string, string>
  memoryHeading: string
  /** Gabarit : {value} en Mo. */
  memoryCurrent: string
  memoryUnavailable: string
  memoryLimitLabel: string
  memoryLimitHint: string
  /** Gabarit : {limit} (valeur du serveur, par exemple « 4096MiB »). */
  memoryEffective: string
  saved: string
  saveFailed: string
  loadFailed: string
  /** Refus du serveur, par code. */
  errors: Record<string, string>
}

const FR: DataText = {
  title: 'Données et sauvegarde',
  intro:
    "Ces réglages ne concernent que cette application de bureau. La sauvegarde copie vos bases dans un dossier de votre choix, au format Parquet, sans interrompre LevelUp.",
  backupHeading: 'Sauvegarde automatique',
  backupDirLabel: 'Dossier de sauvegarde',
  backupDirPlaceholder: 'Chemin absolu, par exemple D:\\Sauvegardes\\LevelUp',
  backupDirHint:
    "Laissez vide pour désactiver la sauvegarde. Le dossier doit exister, être accessible en écriture et se trouver hors du dossier des données de LevelUp.",
  save: 'Enregistrer',
  sameVolumeWarning:
    'Ce dossier est sur le même disque que vos données : une panne du disque emporterait les deux.',
  intervalLabel: 'Fréquence de la sauvegarde',
  intervals: {
    '1h': 'Toutes les heures',
    '6h': 'Toutes les 6 heures',
    '12h': 'Toutes les 12 heures',
    '24h': 'Tous les jours',
    '72h': 'Tous les 3 jours',
    '168h': 'Toutes les semaines',
  },
  intervalEvery: 'Toutes les {hours} h',
  keepLabel: 'Jeux de sauvegarde conservés',
  keepHint: 'De 1 à 60. Les plus anciens sont supprimés au fur et à mesure.',
  runNow: 'Sauvegarder maintenant',
  running: 'Sauvegarde en cours…',
  runStarted: 'Sauvegarde lancée.',
  lastLabel: 'Dernière sauvegarde',
  lastNone: 'Aucune sauvegarde pour le moment.',
  lastLoading: "Lecture de l'état…",
  lastUnavailable: "L'état des sauvegardes est indisponible : on ne sait pas si une sauvegarde existe.",
  lastSummary: '{date} · {size} · {count} · {duration}',
  databasesOne: 'base',
  databasesMany: 'bases',
  lastPartial: 'Jeu incomplet : une base est restée occupée ou a échoué. La prochaine sauvegarde réessaiera.',
  lastAttemptFailed: 'Le dernier passage a échoué ({date}) : {reason}',
  attemptErrors: {
    nothing_exported: "aucune base n'a pu être copiée.",
    backup_failed: "la sauvegarde n'a pas pu être écrite ; le détail est dans les journaux.",
  },
  memoryHeading: 'Mémoire',
  memoryCurrent: 'Mémoire actuelle : {value} Mo',
  memoryUnavailable: 'Mémoire actuelle : mesure indisponible.',
  memoryLimitLabel: 'Plafond mémoire par base ouverte (Mo)',
  memoryLimitHint:
    "0 = valeur par défaut du serveur ; sinon de 1 024 à 32 768 Mo. Le plafond vaut pour CHAQUE base ouverte : plusieurs bases ouvertes en même temps peuvent donc consommer davantage au total. Un plafond plus bas laisse plus de mémoire au jeu, au prix de requêtes plus lentes.",
  memoryEffective: 'Plafond effectif par base : {limit}',
  saved: 'Enregistré.',
  saveFailed: "Le réglage n'a pas pu être enregistré.",
  loadFailed: "L'état des données est indisponible pour le moment.",
  errors: {
    backup_dir_not_absolute: 'Le chemin doit être absolu (par exemple D:\\Sauvegardes).',
    backup_dir_missing: "Ce dossier n'existe pas.",
    backup_dir_not_writable: 'LevelUp ne peut pas écrire dans ce dossier.',
    backup_dir_inside_data:
      'Le dossier de sauvegarde ne peut pas se trouver dans le dossier des données de LevelUp.',
    backup_interval_invalid: 'Cette fréquence de sauvegarde est inconnue.',
    backup_keep_invalid: 'Le nombre de jeux conservés doit être compris entre 1 et 60.',
    memory_limit_invalid: 'Le plafond doit valoir 0 (défaut) ou être compris entre 1 024 et 32 768 Mo.',
    data_setting_not_offered: "Ce réglage n'est offert que dans l'application de bureau.",
    backup_disabled: "Choisissez d'abord un dossier de sauvegarde.",
    backup_already_running: 'Une sauvegarde est déjà en cours.',
  },
}

const EN: DataText = {
  title: 'Data and backup',
  intro:
    'These settings only apply to this desktop application. The backup copies your databases to a folder of your choice, in Parquet format, without interrupting LevelUp.',
  backupHeading: 'Automatic backup',
  backupDirLabel: 'Backup folder',
  backupDirPlaceholder: 'Absolute path, for example D:\\Backups\\LevelUp',
  backupDirHint:
    'Leave empty to turn the backup off. The folder must exist, be writable and sit outside the LevelUp data folder.',
  save: 'Save',
  sameVolumeWarning: 'This folder is on the same disk as your data: a disk failure would take both.',
  intervalLabel: 'Backup frequency',
  intervals: {
    '1h': 'Every hour',
    '6h': 'Every 6 hours',
    '12h': 'Every 12 hours',
    '24h': 'Every day',
    '72h': 'Every 3 days',
    '168h': 'Every week',
  },
  intervalEvery: 'Every {hours} h',
  keepLabel: 'Backup sets kept',
  keepHint: 'From 1 to 60. The oldest ones are deleted as new ones are written.',
  runNow: 'Back up now',
  running: 'Backup in progress…',
  runStarted: 'Backup started.',
  lastLabel: 'Last backup',
  lastNone: 'No backup yet.',
  lastLoading: 'Reading the status…',
  lastUnavailable: 'The backup status is unavailable: it is not known whether a backup exists.',
  lastSummary: '{date} · {size} · {count} · {duration}',
  databasesOne: 'database',
  databasesMany: 'databases',
  lastPartial: 'Incomplete set: a database stayed busy or failed. The next backup will try again.',
  lastAttemptFailed: 'The last attempt failed ({date}): {reason}',
  attemptErrors: {
    nothing_exported: 'no database could be copied.',
    backup_failed: 'the backup could not be written; the details are in the logs.',
  },
  memoryHeading: 'Memory',
  memoryCurrent: 'Current memory: {value} MB',
  memoryUnavailable: 'Current memory: measurement unavailable.',
  memoryLimitLabel: 'Memory limit per open database (MB)',
  memoryLimitHint:
    '0 = the server default; otherwise 1,024 to 32,768 MB. The limit applies to EACH open database: several databases open at once can therefore use more in total. A lower limit leaves more memory to the game, at the cost of slower queries.',
  memoryEffective: 'Effective limit per database: {limit}',
  saved: 'Saved.',
  saveFailed: 'The setting could not be saved.',
  loadFailed: 'The data status is unavailable right now.',
  errors: {
    backup_dir_not_absolute: 'The path must be absolute (for example D:\\Backups).',
    backup_dir_missing: 'This folder does not exist.',
    backup_dir_not_writable: 'LevelUp cannot write to this folder.',
    backup_dir_inside_data: 'The backup folder cannot be inside the LevelUp data folder.',
    backup_interval_invalid: 'This backup frequency is unknown.',
    backup_keep_invalid: 'The number of sets kept must be between 1 and 60.',
    memory_limit_invalid: 'The limit must be 0 (default) or between 1,024 and 32,768 MB.',
    data_setting_not_offered: 'This setting is only offered in the desktop application.',
    backup_disabled: 'Choose a backup folder first.',
    backup_already_running: 'A backup is already running.',
  },
}

const TEXT: Record<Locale, DataText> = { fr: FR, en: EN }

export function getDataText(locale: Locale): DataText {
  return TEXT[locale]
}
