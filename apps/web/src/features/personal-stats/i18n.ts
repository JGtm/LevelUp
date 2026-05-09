/**
 * i18n.ts — Dictionnaire FR/EN des strings UI de la feature Personal Stats.
 *
 * Frontière stricte avec les mappings TOML multi-titres :
 *  - Les libellés métier (FieldKey, assets, outcomes) restent dans les TOML
 *    et passent par useFieldLabel / useAssetLabel / useOutcomeLabel.
 *  - Ce fichier ne contient que des strings UI non-titre-bound : navigation,
 *    empty states, errors, boutons.
 *
 * Pattern aligné avec features/squad/i18n.ts.
 */

export type PersonalStatsLocale = 'fr' | 'en'

export interface PersonalStatsText {
  intlLocale: string
  nav: {
    summary: string
    mapsModes: string
    distributions: string
    progression: string
    advanced: string
  }
  filter: {
    analyse: string
    reset: string
  }
  empty: {
    noDataTitle: string
    noDataDescription: string
    placeholderTitle: string
    placeholderDescription: string
  }
  errors: {
    loadError: (message: string) => string
  }
}

const FR_TEXT: PersonalStatsText = {
  intlLocale: 'fr-FR',
  nav: {
    summary: 'Résumé',
    mapsModes: 'Cartes & Modes',
    distributions: 'Distributions',
    progression: 'Progression',
    advanced: 'Avancé',
  },
  filter: {
    analyse: 'Analyser',
    reset: '↺ Réinitialiser',
  },
  empty: {
    noDataTitle: 'Données indisponibles',
    noDataDescription:
      'Aucune réponse exploitable n\'a été renvoyée pour cette page. Vérifie les filtres ou la disponibilité des matchs.',
    placeholderTitle: 'Section en construction',
    placeholderDescription: 'Le contenu de cet onglet sera ajouté prochainement.',
  },
  errors: {
    loadError: (message) => `Erreur : ${message}`,
  },
}

const EN_TEXT: PersonalStatsText = {
  intlLocale: 'en-US',
  nav: {
    summary: 'Summary',
    mapsModes: 'Maps & Modes',
    distributions: 'Distributions',
    progression: 'Progression',
    advanced: 'Advanced',
  },
  filter: {
    analyse: 'Analyse',
    reset: '↺ Reset',
  },
  empty: {
    noDataTitle: 'Data unavailable',
    noDataDescription:
      'No usable response was returned for this page. Check filters or match availability.',
    placeholderTitle: 'Section under construction',
    placeholderDescription: 'Content for this tab will be added soon.',
  },
  errors: {
    loadError: (message) => `Error: ${message}`,
  },
}

const DICTS: Record<PersonalStatsLocale, PersonalStatsText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

/** Retourne le dictionnaire pour la locale demandée (fallback fr). */
export function getPersonalStatsText(
  locale: PersonalStatsLocale | string | undefined,
): PersonalStatsText {
  if (locale === 'en') return DICTS.en
  return DICTS.fr
}

// Exports nommés pour permettre des tests de parité FR/EN.
export { FR_TEXT, EN_TEXT }
