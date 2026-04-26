/**
 * fallback.i18n.ts — labels de repli quand /field-mappings n'est pas chargé.
 *
 * Ces dictionnaires servent uniquement quand `MULTI_TITLE_API_ENABLED=false`
 * côté serveur. Quand le flag est on, `useAssetLabel('prestige_level', idx)`
 * et `useAssetLabel('cadence', key)` retournent les libellés du TOML.
 *
 * Les valeurs ici doivent rester alignées sur
 * config/titles/halo_infinite/mappings/assets.toml.
 */

export const PRESTIGE_LEVEL_NAMES_FALLBACK = [
  'Recruit',
  'Soldier',
  'Veteran',
  'Specialist',
  'Elite',
  'Legend',
] as const

export const CADENCE_FREE_FALLBACK_FR = 'Libre'
