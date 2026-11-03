/**
 * Textes de la carte « Exporter vers l'application de bureau » (lot L22, site hébergé) — FR et EN.
 *
 * Dictionnaire de feature : chaque langue de `config/locales.toml` figure ici (parité par typage :
 * `Record<Locale, ExportForDesktopText>`). Un refus arrive avec un `code` fixe ; le texte est écrit
 * ici, jamais lu dans le message de l'erreur.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface ExportForDesktopText {
  title: string
  intro: string
  generate: string
  generating: string
  tokenLabel: string
  copy: string
  copied: string
  hide: string
  /** Gabarit : {time} (échéance). */
  expiry: string
  instruction: string
  shownOnce: string
  unknownError: string
  /** Refus du serveur, par code. */
  errors: Record<string, string>
}

const FR: ExportForDesktopText = {
  title: "Exporter vers l'application de bureau",
  intro:
    "Générez un jeton pour transférer vos données de ce site vers l'application de bureau LevelUp. Le jeton donne accès à vos données : ne le partagez avec personne.",
  generate: "Générer un jeton d'export",
  generating: 'Génération…',
  tokenLabel: "Jeton d'export",
  copy: 'Copier',
  copied: 'Copié',
  hide: 'Masquer le jeton',
  expiry: "Valable 15 minutes, à usage unique (échéance : {time}).",
  instruction: "Collez-le dans l'application de bureau, Réglages > Application > Importer depuis le site.",
  shownOnce: 'Ce jeton ne sera plus affiché une fois masqué ou la page quittée.',
  unknownError: "Le jeton n'a pas pu être généré.",
  errors: {
    export_rate_limited: 'Un export par heure.',
  },
}

const EN: ExportForDesktopText = {
  title: 'Export to the desktop application',
  intro:
    'Generate a token to move your data from this site to the LevelUp desktop application. The token gives access to your data: share it with nobody.',
  generate: 'Generate an export token',
  generating: 'Generating…',
  tokenLabel: 'Export token',
  copy: 'Copy',
  copied: 'Copied',
  hide: 'Hide the token',
  expiry: 'Valid for 15 minutes, single-use (expires at {time}).',
  instruction: 'Paste it into the desktop application, Settings > Application > Import from the site.',
  shownOnce: 'This token will not be shown again once hidden or after leaving the page.',
  unknownError: 'The token could not be generated.',
  errors: {
    export_rate_limited: 'One export per hour.',
  },
}

const TEXT: Record<Locale, ExportForDesktopText> = { fr: FR, en: EN }

export function getExportForDesktopText(locale: Locale): ExportForDesktopText {
  return TEXT[locale]
}
