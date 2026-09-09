# Revue adversariale — lot 1.7 `kill_positions` par passe (2026-09-09)

Diff relu : `feat/v75...feat/kill-positions-par-passe` (migration `shared_kill_positions_decode_pass_v1`,
persister par passe, garde-rails, tests). Classe de risque persist/migration : DEUX relecteurs Sonnet
en contexte frais, aveugles l'un a l'autre, lentilles L1 (ecritures DuckDB / anti-ART) et L6 (ce que les
tests ne couvrent pas), contrat en six lignes, filtre de recevabilite du skill.

## Constats recevables : 0 / 0

- L1 : 8 conditions verifiees qui tiennent (alias exportes du rebuild, `IDConditional` sans
  renumerotation, `MarkerColumn` = idempotence rejouable apres G.2, tie-break `id DESC` sur
  `written_at` identique, second etage de la vue cloisonne par `decode_pass`, chemin builder no-op
  apres premiere ecriture, aucune allowlist agrandie, ordre canonique). Un test jetable `:memory:`
  a prouve le cas « deux passes au meme instant ».
- L6 : 5 mutations jouees, toutes rougies (retrait du second etage de la vue, inversion de l'ordre
  de passe, retrait du NOT NULL, `decode_pass` par ligne au lieu de par passe, `DELETE FROM
  kill_positions` dans un fichier de prod). Les tests `platform/duckdb` et `persist` construisent la
  table par la VRAIE chaine de migration.

## Lecon de methode

Les deux relecteurs partageaient le meme worktree ; le relecteur L1 a annule a la volee une mutation
du relecteur L6, qui a du la rejouer isolement. Regle pour les prochaines revues doubles : UN
WORKTREE PAR RELECTEUR des qu'un relecteur joue des mutations.

Dette confirmee, non traitee (registre) : `internal/ops/seed_demo_corpus.go:362` fait un `UPDATE
kill_positions` hors des garde-rails anti-ART (anonymisation demo).
