export type MediaLocale = 'fr' | 'en'

export interface MediaText {
  title: string
  emptyState: string
  errorPrefix: string
  previousPage: string
  nextPage: string
  pageLabel: (page: number, totalPages: number) => string
  groupSection: {
    sessionOfPrefix: string
    likedSection: string
    notLikedSection: string
    unknownOwner: string
    unknownMap: string
    unknownMode: string
    unknownSession: string
  }
  toolbar: {
    filterLabel: string
    sortLabel: string
    kindAriaLabel: string
    playlistAriaLabel: string
    mapAriaLabel: string
    modeAriaLabel: string
    sortAriaLabel: string
    groupAriaLabel: string
    likedOnlyAriaLabel: string
    authorsAriaLabel: string
    allAuthorsToggle: string
    noAuthors: string
    allTypes: string
    screenshots: string
    clips: string
    allAuthors: string
    mine: string
    allPlaylists: string
    allMaps: string
    allModes: string
    /** Étiquette d'option "Toute la catégorie X" en tête d'un <optgroup>. */
    allInCategory: (categoryLabel: string) => string
    /** Traductions FR/EN des catégories custom (cf. analysis.ModeCategory* en Go). */
    modeCategories: {
      Assassin: string
      Fiesta: string
      'Super Fiesta': string
      'Husky Raid': string
      BTB: string
      Ranked: string
      Firefight: string
      Other: string
    }
    dateDesc: string
    dateAsc: string
    mapAsc: string
    modeAsc: string
    noGrouping: string
    byOwner: string
    byMap: string
    byMode: string
    bySession: string
  }
}

const FR_TEXT: MediaText = {
  title: 'Médias',
  emptyState: 'Aucun média disponible pour ces filtres.',
  errorPrefix: 'Erreur :',
  previousPage: '← Précédent',
  nextPage: 'Suivant →',
  pageLabel: (page, totalPages) => `Page ${page} / ${totalPages}`,
  groupSection: {
    sessionOfPrefix: 'Session du',
    likedSection: 'Aimés',
    notLikedSection: 'Non aimés',
    unknownOwner: 'Auteur inconnu',
    unknownMap: 'Carte inconnue',
    unknownMode: 'Mode inconnu',
    unknownSession: 'Session inconnue',
  },
  toolbar: {
    filterLabel: 'Filtres :',
    sortLabel: 'Tri :',
    kindAriaLabel: 'Type de média',
    playlistAriaLabel: 'Playlist de la galerie',
    mapAriaLabel: 'Carte de la galerie',
    modeAriaLabel: 'Mode de la galerie',
    sortAriaLabel: 'Tri de la galerie',
    groupAriaLabel: 'Groupement de la galerie',
    likedOnlyAriaLabel: 'Afficher seulement les médias aimés',
    authorsAriaLabel: 'Filtrer par auteur',
    allAuthorsToggle: 'Tous',
    noAuthors: 'Aucun auteur disponible',
    allTypes: 'Tous types',
    screenshots: 'Captures',
    clips: 'Clips',
    allAuthors: 'Tous les auteurs',
    mine: 'Mes captures',
    allPlaylists: 'Toutes playlists',
    allMaps: 'Toutes cartes',
    allModes: 'Tous modes',
    allInCategory: () => `Toutes catégories`,
    modeCategories: {
      Assassin: 'Assassin',
      Fiesta: 'Fiesta',
      'Super Fiesta': 'Super Fiesta',
      'Husky Raid': 'Husky Raid',
      BTB: 'Grande bataille en équipe',
      Ranked: 'Classé',
      Firefight: 'Baptême du feu',
      Other: 'Autre',
    },
    dateDesc: 'Date ↓',
    dateAsc: 'Date ↑',
    mapAsc: 'Carte A→Z',
    modeAsc: 'Mode A→Z',
    noGrouping: 'Sans groupement',
    byOwner: 'Par auteur',
    byMap: 'Par carte',
    byMode: 'Par mode',
    bySession: 'Par session',
  },
}

const EN_TEXT: MediaText = {
  title: 'Media',
  emptyState: 'No media available for the current filters.',
  errorPrefix: 'Error:',
  previousPage: '← Previous',
  nextPage: 'Next →',
  pageLabel: (page, totalPages) => `Page ${page} / ${totalPages}`,
  groupSection: {
    sessionOfPrefix: 'Session of',
    likedSection: 'Liked',
    notLikedSection: 'Not liked',
    unknownOwner: 'Unknown author',
    unknownMap: 'Unknown map',
    unknownMode: 'Unknown mode',
    unknownSession: 'Unknown session',
  },
  toolbar: {
    filterLabel: 'Filters:',
    sortLabel: 'Sort:',
    kindAriaLabel: 'Media type',
    playlistAriaLabel: 'Media playlist',
    mapAriaLabel: 'Media map',
    modeAriaLabel: 'Media mode',
    sortAriaLabel: 'Media sorting',
    groupAriaLabel: 'Media grouping',
    likedOnlyAriaLabel: 'Show liked media only',
    authorsAriaLabel: 'Filter by author',
    allAuthorsToggle: 'All',
    noAuthors: 'No authors available',
    allTypes: 'All types',
    screenshots: 'Screenshots',
    clips: 'Clips',
    allAuthors: 'All authors',
    mine: 'My captures',
    allPlaylists: 'All playlists',
    allMaps: 'All maps',
    allModes: 'All modes',
    allInCategory: (cat) => `All ${cat}`,
    modeCategories: {
      Assassin: 'Slayer',
      Fiesta: 'Fiesta',
      'Super Fiesta': 'Super Fiesta',
      'Husky Raid': 'Husky Raid',
      BTB: 'Big Team Battle',
      Ranked: 'Ranked',
      Firefight: 'Firefight',
      Other: 'Other',
    },
    dateDesc: 'Date ↓',
    dateAsc: 'Date ↑',
    mapAsc: 'Map A→Z',
    modeAsc: 'Mode A→Z',
    noGrouping: 'No grouping',
    byOwner: 'By author',
    byMap: 'By map',
    byMode: 'By mode',
    bySession: 'By session',
  },
}

const TEXT: Record<MediaLocale, MediaText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

export function normalizeMediaLocale(locale?: string | null): MediaLocale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getMediaText(locale?: string | null): MediaText {
  return TEXT[normalizeMediaLocale(locale)]
}
