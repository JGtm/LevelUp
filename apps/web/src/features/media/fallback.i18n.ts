/**
 * fallback.i18n.ts (media) — labels de repli quand /field-mappings n'est pas chargé.
 *
 * Quand `MULTI_TITLE_API_ENABLED=false` côté serveur, les hooks `useOutcomeLabel`
 * et `useAssetLabel` retournent la clé brute. Ce dictionnaire fournit les
 * libellés FR pour les outcomes Halo dans ce cas dégradé.
 *
 * Les valeurs ici doivent rester alignées sur
 * config/titles/halo_infinite/mappings/outcomes.toml.
 */

/** Indexé par le code outcome Halo (2 win, 3 loss, 1 tie, 4 dnf). */
export const OUTCOME_LABELS_FALLBACK_FR: Record<number, string> = {
  2: 'Victoire',
  3: 'Défaite',
  1: 'Égalité',
  4: 'DNF',
}
