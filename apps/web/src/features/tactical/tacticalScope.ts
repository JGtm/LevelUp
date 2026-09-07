/**
 * tacticalScope — contrat d'état persistable de l'onglet Tactique.
 *
 * Deux représentations manipulées par `usePageScope` (modèle `explorerScope`) :
 *   - `TacticalScope`        : état riche consommé par la page et la barre L2
 *   - `EncodedTacticalScope` : forme plate sérialisée en query params
 *
 * Param URL ↔ champ applicatif :
 *   de → debut · a → fin · exp → experience · pl → playlists · md → modes
 *   ses → sessions · vue → vue · eq → coequipiers · carte → carte
 *
 * PAS DE PARAM « SAISON », et ce n'est pas un oubli : la pill Saison n'est qu'un
 * raccourci qui POSE une période (`seasonToPeriod`), et `useActiveSeason` relit la
 * saison depuis cette période. Un second paramètre porterait la même information
 * une deuxième fois, avec le risque qu'ils se contredisent dans une URL partagée.
 *
 * LA COMPOSITION EST STOCKÉE EN GAMERTAGS, pas en xuids : c'est le vocabulaire du
 * sélecteur (`GamertagCombobox`), celui de la page Escouade
 * (`selected_gamertags`), et le seul qui reste lisible dans une URL partagée. La
 * traduction vers les xuids qu'attend le serveur se fait au moment de la requête,
 * à partir de la liste de coéquipiers — et un nom qu'on ne sait pas traduire
 * ARRÊTE la requête plutôt que de l'élargir en silence (cf. `resoudreComposition`).
 *
 * Module pur : aucune dépendance React/Router, testable sans DOM.
 */
import { csvToSet, setToCsv } from '@/lib/page-scope/serialize'
import type { Experience } from '@/features/_shared/ExperienceDropdown'
import type { MatchView } from '@/features/_shared/useLocalFilterBar'

/** Nombre maximum de coéquipiers dans la composition — même plafond que la page
 *  Escouade (`MAX_SELECTION`), parce que c'est la même notion d'escouade. */
export const MAX_COEQUIPIERS = 3

/** État riche consommé par l'onglet. */
export interface TacticalScope {
  /** Bornes de période, format `YYYY-MM-DD` ; vide = aucune borne. */
  debut: string
  fin: string
  experience: Experience
  playlists: string[]
  modes: string[]
  /** Labels de session épinglés (vocabulaire de `SessionMultiSelect`). */
  sessions: string[]
  vue: MatchView
  /** Gamertags de la composition choisie (0 à MAX_COEQUIPIERS). */
  coequipiers: string[]
  /** Carte sélectionnée dans la grille ; '' = aucune. */
  carte: string
}

/** Forme plate sérialisée en query params (toutes les clés optionnelles). */
export interface EncodedTacticalScope {
  de?: string
  a?: string
  exp?: string
  pl?: string
  md?: string
  ses?: string
  vue?: string
  eq?: string
  carte?: string
}

/** Clés de scope dans l'URL (détection cold-start + reset). */
export const TACTICAL_URL_KEYS: readonly (keyof EncodedTacticalScope)[] = [
  'de',
  'a',
  'exp',
  'pl',
  'md',
  'ses',
  'vue',
  'eq',
  'carte',
]

/** Le scope d'arrivée : aucun filtre, aucune composition, aucune carte. */
export const TACTICAL_SCOPE_DEFAUT: TacticalScope = {
  debut: '',
  fin: '',
  experience: 'all',
  playlists: [],
  modes: [],
  sessions: [],
  vue: 'all',
  coequipiers: [],
  carte: '',
}

/** App → URL. `undefined` pour les valeurs vides : le param est alors omis. */
export function encodeTacticalScope(s: TacticalScope): EncodedTacticalScope {
  return {
    de: s.debut || undefined,
    a: s.fin || undefined,
    exp: s.experience !== 'all' ? s.experience : undefined,
    pl: setToCsv(new Set(s.playlists)),
    md: setToCsv(new Set(s.modes)),
    ses: setToCsv(new Set(s.sessions)),
    vue: s.vue !== 'all' ? s.vue : undefined,
    eq: setToCsv(new Set(s.coequipiers)),
    carte: s.carte || undefined,
  }
}

/** URL (partielle) → App. Remplit les défauts pour les params absents. */
export function decodeTacticalScope(raw: Partial<EncodedTacticalScope>): TacticalScope {
  return {
    debut: typeof raw.de === 'string' ? raw.de : '',
    fin: typeof raw.a === 'string' ? raw.a : '',
    experience: estExperience(raw.exp) ? raw.exp : 'all',
    playlists: [...csvToSet(raw.pl)],
    modes: [...csvToSet(raw.md)],
    sessions: [...csvToSet(raw.ses)],
    vue: estVue(raw.vue) ? raw.vue : 'all',
    // Le plafond est appliqué À LA LECTURE : une URL bricolée à la main ne doit
    // pas faire entrer six coéquipiers dans une composition qui en accepte trois.
    coequipiers: [...csvToSet(raw.eq)].slice(0, MAX_COEQUIPIERS),
    carte: typeof raw.carte === 'string' ? raw.carte : '',
  }
}

function estExperience(v: unknown): v is Experience {
  return v === 'ranked' || v === 'unranked' || v === 'all'
}

function estVue(v: unknown): v is MatchView {
  return v === 'solo' || v === 'squad' || v === 'all'
}
