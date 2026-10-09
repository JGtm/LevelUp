/**
 * cardsI18n.ts — LES NOMS DES FAMILLES DE MODE À OBJECTIF du bloc `formes_retenues`, lus par les
 * cartes d'objectif (Escouade › Contributions, Séries temporelles › Usages) : une clé de famille
 * du serveur, son libellé. Une famille inconnue garde sa clé à l'écran (repli de l'appelant).
 * Parité FR / EN par le typage `Record<Locale, …>`.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface FormesCardsText {
  families: Record<string, string>
}

export const FORMES_CARDS_TEXT: Record<Locale, FormesCardsText> = {
  fr: {
    families: {
      ctf: 'Drapeau',
      zones_koth: 'Roi de la colline',
      zones_strongholds: 'Bases',
      oddball: 'Crâne',
      stockpile: 'Réserve',
      extraction: 'Extraction',
      vip: 'VIP',
    },
  },
  en: {
    families: {
      ctf: 'Capture the Flag',
      zones_koth: 'King of the Hill',
      zones_strongholds: 'Strongholds',
      oddball: 'Oddball',
      stockpile: 'Stockpile',
      extraction: 'Extraction',
      vip: 'VIP',
    },
  },
}
