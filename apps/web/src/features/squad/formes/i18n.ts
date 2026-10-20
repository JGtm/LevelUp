/**
 * i18n.ts — LES LIBELLÉS DES COLONNES D'OBJECTIF du bloc `formes_retenues`, lus par les cartes
 * d'objectif (Escouade › Contributions, Séries temporelles › Usages).
 *
 * FR ET EN À PARITÉ DE TYPAGE (`Record<Locale, …>`) : une clé ajoutée d'un côté casse la
 * compilation de l'autre.
 *
 * LES LIBELLÉS DE COLONNE D'OBJECTIF sont une DEUXIÈME copie assumée de ceux de la vue match
 * (`features/match-view/i18n.ts`, table `objectives.cols`) : l'import croisé entre features est
 * interdit par le ratchet. À la troisième copie, centraliser et poser le garde-rail (règle
 * CLAUDE.md n°6).
 */
import type { Locale } from '@/lib/i18n/locale'

export interface FormesText {
  /** Le libellé d'une colonne publiée par le serveur, par sa clé (une clé inconnue garde sa clé). */
  columns: Record<string, string>
}
const FR_COLUMNS: Record<string, string> = {
  flag_captures: 'Drapeaux capturés',
  // Prises NETTES : le compteur officiel du jeu compte chaque ramassage, donc
  // aussi le jonglage (lancer le drapeau devant soi pour courir plus vite, puis
  // le reprendre). Cette grandeur-ci replie ces allers-retours.
  flag_grabs_net: 'Prises nettes',
  flag_capture_assists: 'Aides à la capture',
  flag_steals: 'Drapeaux volés',
  flag_returners_killed: 'Rapatrieurs abattus',
  flag_returns: 'Retours',
  flag_secures: 'Drapeaux sécurisés',
  flag_carriers_killed: 'Porteurs abattus',
  time_as_flag_carrier_seconds: 'Temps de portage',
  zone_captures: 'Zones capturées',
  zone_secures: 'Zones sécurisées',
  zone_offensive_kills: 'Frags offensifs de zone',
  zone_defensive_kills: 'Frags défensifs de zone',
  time_in_zones_seconds: 'Temps en zone',
  skull_grabs: 'Crânes récupérés',
  skull_carriers_killed: 'Porteurs de crâne abattus',
  time_as_skull_carrier_seconds: 'Temps de portage',
  power_seeds_deposited: 'Graines déposées',
  power_seeds_stolen: 'Graines volées',
  power_seed_carriers_killed: 'Porteurs de graine abattus',
  time_as_power_seed_carrier_seconds: 'Temps de portage',
  successful_extractions: 'Extractions réussies',
  extraction_initiations_completed: 'Amorçages menés à terme',
  extraction_conversions_completed: 'Conversions réussies',
  extraction_conversions_denied: 'Conversions refusées',
  vip_kills: 'VIP adverses abattus',
  vip_assists: 'Aides sur VIP',
  kills_as_vip: 'Frags en étant VIP',
  time_as_vip_seconds: 'Temps en VIP',
}

const EN_COLUMNS: Record<string, string> = {
  flag_captures: 'Flags captured',
  flag_grabs_net: 'Net grabs',
  flag_capture_assists: 'Capture assists',
  flag_steals: 'Flags stolen',
  flag_returners_killed: 'Returners killed',
  flag_returns: 'Returns',
  flag_secures: 'Flags secured',
  flag_carriers_killed: 'Carriers killed',
  time_as_flag_carrier_seconds: 'Carrier time',
  zone_captures: 'Zones captured',
  zone_secures: 'Zones secured',
  zone_offensive_kills: 'Zone offensive kills',
  zone_defensive_kills: 'Zone defensive kills',
  time_in_zones_seconds: 'Zone time',
  skull_grabs: 'Skulls grabbed',
  skull_carriers_killed: 'Skull carriers killed',
  time_as_skull_carrier_seconds: 'Carrier time',
  power_seeds_deposited: 'Seeds deposited',
  power_seeds_stolen: 'Seeds stolen',
  power_seed_carriers_killed: 'Seed carriers killed',
  time_as_power_seed_carrier_seconds: 'Carrier time',
  successful_extractions: 'Successful extractions',
  extraction_initiations_completed: 'Initiations completed',
  extraction_conversions_completed: 'Conversions completed',
  extraction_conversions_denied: 'Conversions denied',
  vip_kills: 'Enemy VIPs killed',
  vip_assists: 'VIP assists',
  kills_as_vip: 'Kills as VIP',
  time_as_vip_seconds: 'VIP time',
}

export const FORMES_TEXT: Record<Locale, FormesText> = {
  fr: { columns: FR_COLUMNS },
  en: { columns: EN_COLUMNS },
}
