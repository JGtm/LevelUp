/**
 * cockpit.logic — la logique PURE de l'écran unique de l'onglet Tactique : les mesures de la
 * grille à trois colonnes, la colonne « Cartes jouées » (recherche, ouvrables, repli des cartes
 * sous le plancher) et la carte lue d'office.
 *
 * LE PLANCHER N'EST PAS RECALCULÉ ICI : le serveur publie `sous_plancher` par carte, le client
 * lit ce verdict (`estOuvrable`) et ne juge pas.
 *
 * L'URL N'EST JAMAIS RÉÉCRITE PAR LA SÉLECTION D'OFFICE (D11) : la carte lue sans `?carte=` est un
 * état DÉRIVÉ de la liste des cartes, recalculé à chaque rendu. L'écrire dans l'URL créerait une
 * entrée d'historique que l'utilisateur n'a pas demandée.
 */
import type { CSSProperties } from 'react'

import type { TacticalMapCard } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { estOuvrable, nomCarte, trierCartes } from './tacticalLogic'

// ─── Mesures de la grille (D13, maquette l. 195-228) ───────────────────────────────────────
/** Largeur de la colonne « Cartes jouées », en px. */
const COLONNE_CARTES_LARGEUR_PX = 208
/** Hauteur FIXE de la colonne « Cartes jouées » à trois colonnes, en px : la liste défile dedans. */
const COLONNE_CARTES_HAUTEUR_PX = 551
/** Largeur de la colonne « Zone sélectionnée », en px. */
const COLONNE_ZONE_LARGEUR_PX = 360
/** Hauteur maximale de la boîte du plan (fond + calque), en px. */
const PLAN_BOITE_HAUTEUR_MAX_PX = 800
/** Hauteur de la rampe verticale de la légende, en px. */
const RAMPE_HAUTEUR_PX = 220
/** Marge droite réservée à la légende à côté du fond, en px. */
const LEGENDE_MARGE_PX = 90
/** Largeur de la vignette compacte d'une carte (16:9), en px. */
export const VIGNETTE_LARGEUR_PX = 100

/**
 * Les mesures ci-dessus en variables CSS, posées sur la racine du cockpit : les classes de mise
 * en page (`min-[1400px]:…`, trois colonnes dès 1 400 px de large) les lisent par `var(…)`, et
 * les constantes restent la seule source des valeurs.
 */
export const VARIABLES_COCKPIT = {
  '--tac-cartes-l': `${COLONNE_CARTES_LARGEUR_PX}px`,
  '--tac-cartes-h': `${COLONNE_CARTES_HAUTEUR_PX}px`,
  '--tac-zone-l': `${COLONNE_ZONE_LARGEUR_PX}px`,
  '--tac-plan-hmax': `${PLAN_BOITE_HAUTEUR_MAX_PX}px`,
  '--tac-rampe-h': `${RAMPE_HAUTEUR_PX}px`,
  '--tac-legende-m': `${LEGENDE_MARGE_PX}px`,
} as CSSProperties

/** D'où vient la carte lue (D11). */
type OrigineCarte = 'url' | 'defaut' | 'hors_filtre' | 'aucune'

export interface CarteEffective {
  /** La carte affichée ; '' = aucune. */
  mapId: string
  /**
   * `url` : la carte de l'URL, ouvrable, est lue ; `defaut` : sans carte dans l'URL, la plus jouée
   * des ouvrables est lue ; `hors_filtre` : la carte de l'URL est absente du filtre ou sous le
   * plancher — nommée, jamais lue ; `aucune` : aucune carte ouvrable, rien à lire.
   */
  origine: OrigineCarte
}

/** carteEffective — la carte que le plan affiche (D11). */
export function carteEffective(carteUrl: string, cartes: readonly TacticalMapCard[]): CarteEffective {
  if (carteUrl !== '') {
    const demandee = cartes.find((c) => c.map_id === carteUrl)
    return { mapId: carteUrl, origine: demandee && estOuvrable(demandee) ? 'url' : 'hors_filtre' }
  }
  const premiere = trierCartes(cartes).find(estOuvrable)
  return premiere ? { mapId: premiere.map_id, origine: 'defaut' } : { mapId: '', origine: 'aucune' }
}

/** carteLue — la carte effective est-elle LUE (requête de lecture) ? */
export function carteLue(c: CarteEffective): boolean {
  return c.origine === 'url' || c.origine === 'defaut'
}

/** normaliserRecherche — minuscules, sans diacritiques, sans blancs autour. */
export function normaliserRecherche(texte: string): string {
  return texte
    .toLowerCase()
    .normalize('NFD')
    .replace(/\p{Diacritic}/gu, '')
    .trim()
}

/**
 * filtrerCartes — les cartes dont le nom AFFICHÉ ou le nom CANONIQUE contient la recherche, sans
 * casse ni accents. Recherche vide : toutes.
 */
export function filtrerCartes(
  cartes: readonly TacticalMapCard[],
  recherche: string,
  locale: Locale,
): TacticalMapCard[] {
  const q = normaliserRecherche(recherche)
  if (q === '') return [...cartes]
  return cartes.filter(
    (c) => normaliserRecherche(nomCarte(c, locale)).includes(q) || normaliserRecherche(c.map_name).includes(q),
  )
}

/** Ce que rend la colonne « Cartes jouées ». */
export interface ColonneDesCartes {
  /** Les cartes ouvrables qui correspondent à la recherche, de la plus jouée à la moins jouée. */
  ouvrables: TacticalMapCard[]
  /** Les cartes sous le plancher qui correspondent à la recherche, même ordre. */
  sousPlancher: TacticalMapCard[]
  /** Toutes les cartes sous le plancher du filtre, recherche ignorée (« x sur N »). */
  totalSousPlancher: number
  /** Une recherche est saisie : le repli se résume « x sur N », sinon « N ». */
  rechercheActive: boolean
  /** Le repli s'ouvre quand seules des cartes sous le plancher correspondent. */
  repliOuvert: boolean
  /**
   * Ce que la liste dit quand elle n'a aucune vignette : aucune ouvrable ne correspond à la
   * recherche, ou aucune carte du filtre n'est ouvrable ; `null` quand elle en a, ou quand le
   * filtre n'a aucune carte (l'état vide de la page le dit).
   */
  listeVide: 'aucune_correspondance' | 'aucune_ouvrable' | null
}

/** colonneDesCartes — la partition ouvrables / sous le plancher après recherche, et le repli. */
export function colonneDesCartes(
  cartes: readonly TacticalMapCard[],
  recherche: string,
  locale: Locale,
): ColonneDesCartes {
  const triees = trierCartes(cartes)
  const totalSousPlancher = triees.filter((c) => !estOuvrable(c)).length
  const visibles = filtrerCartes(triees, recherche, locale)
  const ouvrables = visibles.filter(estOuvrable)
  const sousPlancher = visibles.filter((c) => !estOuvrable(c))
  const rechercheActive = normaliserRecherche(recherche) !== ''
  let listeVide: ColonneDesCartes['listeVide'] = null
  if (ouvrables.length === 0 && triees.length > 0) {
    listeVide = rechercheActive ? 'aucune_correspondance' : 'aucune_ouvrable'
  }
  return {
    ouvrables,
    sousPlancher,
    totalSousPlancher,
    rechercheActive,
    repliOuvert: ouvrables.length === 0 && sousPlancher.length > 0,
    listeVide,
  }
}
