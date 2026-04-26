/**
 * fallback.i18n.ts (home) — labels de repli quand /field-mappings n'est pas chargé.
 *
 * Quand `MULTI_TITLE_API_ENABLED=false` côté serveur, les hooks `useAssetLabel`
 * et `useOutcomeLabel` retournent la clé brute. Ces dictionnaires fournissent
 * des libellés FR lisibles dans ce cas dégradé.
 *
 * Les valeurs ici doivent rester alignées sur
 * config/titles/halo_infinite/mappings/assets.toml [assets.cadence.*].
 */

export const CADENCE_DAILY_FALLBACK_FR = 'Quotidien'
export const CADENCE_WEEKLY_FALLBACK_FR = 'Hebdo'
