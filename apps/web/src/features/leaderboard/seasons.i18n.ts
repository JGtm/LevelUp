/**
 * Saisons CSR : noms officiels (saisons/opérations Halo Infinite), TEXTE SEUL (pas de
 * numéro). Depuis C2b, le backend renvoie un display_name AUTORITATIF "Saison N · Nom"
 * localisé (season_catalog, scrape Waypoint) : cette table ne sert plus que de SECOURS
 * — (a) catalogue vide (avant le 1er scrape) ; (b) saison présente au classement mais
 * pas encore nommée dans season_catalog. Noms recoupés PAR DATE de début via wiki.halo.fr
 * (Halo a abandonné les saisons numérotées après la S5 → opérations) :
 *   3-1 Echoes Within (mars 2023) · 4-1 Infection (juin 2023) · 5-1 Reckoning (oct 2023)
 *   6-1 Spirit of Fire (janv 2024) · 7-1 Banished Honor · 8-1 Fleetcom · 9-1 Great Journey
 *   10-1 Frontlines (fév 2025) · 11-1 Last Stand (mai 2025) · 12-1 Shadows · 13-2 Infinite.
 *
 * Isolé dans un fichier `.i18n.ts` (whitelist du linter tools/lint-no-hardcoded-fields.mjs) :
 * certains noms de saison ("Infinite", "Shadows", "Last Stand") coïncident avec des
 * libellés d'assets TOML, mais ce sont ici des noms propres de saison (pas de source
 * catalogue) — la place canonique d'un tel dict est un module i18n local.
 */
export const SEASONS: { id: string; label: string }[] = [
  { id: 'csrseason13-2', label: 'Infinite' },
  { id: 'csrseason13-1', label: 'Infinite' },
  { id: 'csrseason12-1', label: 'Shadows' },
  { id: 'csrseason11-1', label: 'Last Stand' },
  { id: 'csrseason10-1', label: 'Frontlines' },
  { id: 'csrseason9-1', label: 'Great Journey' },
  { id: 'csrseason8-1', label: 'Fleetcom' },
  { id: 'csrseason7-1', label: 'Banished Honor' },
  { id: 'csrseason6-1', label: 'Spirit of Fire' },
  { id: 'csrseason5-1', label: 'Reckoning' },
  { id: 'csrseason4-1', label: 'Infection' },
  { id: 'csrseason3-1', label: 'Echoes Within' },
]
