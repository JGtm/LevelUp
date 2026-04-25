/**
 * Traductions des libellés de la barre de stats KPI (hero card de la home).
 *
 * Centralise les chaînes hardcodées qui étaient disséminées sous forme de
 * ternaires `locale === 'en' ? 'X' : 'Y'` dans HomePage.tsx.
 */

export type KPILocale = 'fr' | 'en'

interface KPITextDict {
  labels: {
    matches: string
    kda: string
    winRate: string
    totalTime: string
    favoritePlaylist: string
    offDef: string
    accuracy: string
    favoriteWeapon: string
  }
  units: {
    /** Année (suffixe court : "5a" / "5y"). */
    year: string
    /** Mois (suffixe court : "3m" / "3mo"). */
    month: string
    /** Jour (suffixe court : "12j" / "12d"). */
    day: string
    /** Heure (suffixe court : "8h" / "8h"). */
    hour: string
    /** Minute (suffixe court : "45min" / "45min"). */
    minute: string
  }
  /** Sous-titres pluralisés. */
  matches: (count: number) => string
  kills: (count: number) => string
}

const FR: KPITextDict = {
  labels: {
    matches: 'Parties',
    kda: 'FDA',
    winRate: 'Taux de victoire',
    totalTime: 'Durée totale',
    favoritePlaylist: 'Playlist favorite',
    offDef: 'Rendement / Résist.',
    accuracy: 'Précision',
    favoriteWeapon: 'Arme favorite',
  },
  units: {
    year: 'a',
    month: 'm',
    day: 'j',
    hour: 'h',
    minute: 'min',
  },
  matches: () => 'parties',
  kills: () => 'kills',
}

const EN: KPITextDict = {
  labels: {
    matches: 'Matches',
    kda: 'KDA',
    winRate: 'Win rate',
    totalTime: 'Total time',
    favoritePlaylist: 'Favorite playlist',
    offDef: 'Off. / Def.',
    accuracy: 'Accuracy',
    favoriteWeapon: 'Fav. weapon',
  },
  units: {
    year: 'y',
    month: 'mo',
    day: 'd',
    hour: 'h',
    minute: 'min',
  },
  matches: () => 'matches',
  kills: () => 'kills',
}

const DICTS: Record<KPILocale, KPITextDict> = { fr: FR, en: EN }

export function normalizeKPILocale(locale?: string | null): KPILocale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getKPIText(locale?: string | null): KPITextDict {
  return DICTS[normalizeKPILocale(locale)]
}
