/**
 * Textes de l'import depuis le site (lot L22) — FR et EN.
 *
 * Dictionnaire de feature : chaque langue de `config/locales.toml` figure ici (parité par typage :
 * `Record<Locale, SiteImportText>`). Un échec arrive avec un `code` fixe ; le texte est écrit ici,
 * jamais lu dans le message de l'erreur.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface SiteImportText {
  title: string
  /** Libellé du bloc facultatif de l'assistant de premier lancement. */
  optionalToggle: string
  intro: string
  urlLabel: string
  urlPlaceholder: string
  tokenLabel: string
  tokenHint: string
  gamertagLabel: string
  importButton: string
  /** L'assistant : la synchronisation attend la fin d'un import en cours. */
  blocksSync: string
  starting: string
  downloading: string
  importing: string
  /** Gabarit : {count}. */
  addedOne: string
  addedMany: string
  unknownError: string
  /** Échecs de démarrage et de l'import, par code. */
  errors: Record<string, string>
}

const FR: SiteImportText = {
  title: 'Importer depuis le site',
  optionalToggle: 'Vous avez déjà un compte sur le site ? Importer vos données',
  intro:
    "Récupérez ici les matchs déjà analysés par le site. Sur le site, générez d'abord un jeton d'export dans Réglages > Compte > Exporter vers l'application de bureau, puis collez-le ci-dessous.",
  urlLabel: 'Adresse du site',
  urlPlaceholder: 'https://',
  tokenLabel: "Jeton d'export",
  tokenHint: "Le jeton est à usage unique et valable 15 minutes. Il n'est jamais conservé.",
  gamertagLabel: 'Joueur suivi (gamertag)',
  importButton: 'Importer',
  blocksSync: "Un import est en cours : la synchronisation pourra être lancée une fois qu'il sera terminé.",
  starting: 'Démarrage…',
  downloading: "Téléchargement de l'archive…",
  importing: 'Écriture des données…',
  addedOne: '{count} match ajouté.',
  addedMany: '{count} matchs ajoutés.',
  unknownError: "L'import n'a pas pu démarrer.",
  errors: {
    site_url_invalid: "L'adresse du site n'est pas valide (une adresse https est attendue).",
    site_token_invalid:
      'Le jeton est refusé : il est expiré, déjà utilisé ou incorrect. Générez-en un nouveau sur le site.',
    site_rate_limited: 'Le site limite les exports : réessayez plus tard.',
    site_refused: 'Le site a refusé la demande.',
    site_unreachable: "Le site ne répond pas : vérifiez votre connexion et l'adresse, puis réessayez.",
    site_tls: "La connexion sécurisée au site est refusée (certificat invalide ou expiré) : vérifiez l'adresse.",
    site_not_found: "Cette adresse n'existe pas : vérifiez l'écriture du nom du site.",
    site_archive_too_big: "L'archive est trop volumineuse pour être importée.",
    archive_invalid: "L'archive reçue est corrompue ou incomplète.",
    archive_other_player: 'Cette archive appartient à un autre joueur que celui suivi ici.',
    archive_unknown_table: "L'archive contient des données que cette version ne reconnaît pas.",
    archive_format_unsupported: "Le format de cette archive n'est pas pris en charge par cette version.",
    import_failed: "L'import a échoué ; le détail est dans les journaux.",
    import_already_running: 'Un import est déjà en cours.',
    import_no_tracked_player: "Aucun joueur n'est suivi : ajoutez d'abord votre compte.",
    import_player_without_xuid: "Le joueur suivi n'est pas encore relié à un compte Xbox.",
    import_player_ambiguous: 'Plusieurs joueurs sont suivis : indiquez le gamertag qui reçoit les données.',
  },
}

const EN: SiteImportText = {
  title: 'Import from the site',
  optionalToggle: 'Already have an account on the site? Import your data',
  intro:
    'Bring over the matches the site has already analysed. On the site, first generate an export token in Settings > Account > Export to the desktop application, then paste it below.',
  urlLabel: 'Site address',
  urlPlaceholder: 'https://',
  tokenLabel: 'Export token',
  tokenHint: 'The token is single-use and valid for 15 minutes. It is never kept.',
  gamertagLabel: 'Tracked player (gamertag)',
  importButton: 'Import',
  blocksSync: 'An import is in progress: the synchronisation can be started once it has finished.',
  starting: 'Starting…',
  downloading: 'Downloading the archive…',
  importing: 'Writing the data…',
  addedOne: '{count} match added.',
  addedMany: '{count} matches added.',
  unknownError: 'The import could not start.',
  errors: {
    site_url_invalid: 'The site address is not valid (an https address is expected).',
    site_token_invalid: 'The token was refused: it is expired, already used or incorrect. Generate a new one on the site.',
    site_rate_limited: 'The site limits exports: try again later.',
    site_refused: 'The site refused the request.',
    site_unreachable: 'The site does not answer: check your connection and the address, then try again.',
    site_tls: 'The secure connection to the site was refused (invalid or expired certificate): check the address.',
    site_not_found: 'This address does not exist: check how the site name is spelled.',
    site_archive_too_big: 'The archive is too large to import.',
    archive_invalid: 'The received archive is corrupted or incomplete.',
    archive_other_player: 'This archive belongs to a different player than the one tracked here.',
    archive_unknown_table: 'The archive contains data this version does not recognise.',
    archive_format_unsupported: 'The format of this archive is not supported by this version.',
    import_failed: 'The import failed; the details are in the logs.',
    import_already_running: 'An import is already running.',
    import_no_tracked_player: 'No player is tracked: add your account first.',
    import_player_without_xuid: 'The tracked player is not linked to an Xbox account yet.',
    import_player_ambiguous: 'Several players are tracked: enter the gamertag that receives the data.',
  },
}

const TEXT: Record<Locale, SiteImportText> = { fr: FR, en: EN }

export function getSiteImportText(locale: Locale): SiteImportText {
  return TEXT[locale]
}
