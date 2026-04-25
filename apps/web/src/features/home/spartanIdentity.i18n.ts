/**
 * Traductions des libellés du bloc identité Spartan (hero card de la home).
 *
 * Centralise les chaînes hardcodées du panneau de gauche (rang carrière, pics
 * CSR/LUSR, panneau vide quand aucun classement n'est disponible).
 */

export type SpartanIdentityLocale = 'fr' | 'en'

interface SpartanIdentityTextDict {
  labels: {
    careerRank: string
    highestCsr: string
    highestLusr: string
    currentProgress: string
    rankPrefix: string
    maxRank: string
    progressTowardsRank: (n: number) => string
  }
  emptyPanel: {
    titleUnavailable: string
    titleNone: string
    descriptionUnavailable: string
    descriptionNone: string
  }
}

const FR: SpartanIdentityTextDict = {
  labels: {
    careerRank: 'Rang carrière',
    highestCsr: 'Meilleur CSR',
    highestLusr: 'Meilleur LUSR',
    currentProgress: 'Progression actuelle',
    rankPrefix: 'Rang',
    maxRank: 'Rang max',
    progressTowardsRank: (name: string) => `Progression vers ${name}`,
  },
  emptyPanel: {
    titleUnavailable: 'Classements indisponibles',
    titleNone: 'Aucun classement disponible',
    descriptionUnavailable: 'Les données compétitives de ce joueur sont partielles ou indisponibles.',
    descriptionNone: 'Ce joueur n’a pas encore de classement CSR ni LUSR.',
  },
}

const EN: SpartanIdentityTextDict = {
  labels: {
    careerRank: 'Career rank',
    highestCsr: 'Highest CSR',
    highestLusr: 'Highest LUSR',
    currentProgress: 'Current progress',
    rankPrefix: 'Rank',
    maxRank: 'Max rank',
    progressTowardsRank: (name: string) => `Progress towards ${name}`,
  },
  emptyPanel: {
    titleUnavailable: 'Rankings unavailable',
    titleNone: 'No rankings available',
    descriptionUnavailable: 'This player’s competitive data is partial or unavailable.',
    descriptionNone: 'This player has no CSR or LUSR rankings yet.',
  },
}

const DICTS: Record<SpartanIdentityLocale, SpartanIdentityTextDict> = { fr: FR, en: EN }

export function normalizeSpartanIdentityLocale(locale?: string | null): SpartanIdentityLocale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getSpartanIdentityText(locale?: string | null): SpartanIdentityTextDict {
  return DICTS[normalizeSpartanIdentityLocale(locale)]
}
