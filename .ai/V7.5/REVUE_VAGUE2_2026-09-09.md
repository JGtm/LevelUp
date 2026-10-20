# Revue adversariale de la vague 2 (équipement « servi ou gâché ») — 2026-09-09

Diff relu : `git diff af3ef58b6 feat/v75` (100 fichiers, +7 227 / −1 287), deux relecteurs
Sonnet en contexte frais, aveugles l'un à l'autre, une ronde. Contrat, lentilles et règles de
recevabilité du skill `adversarial-review`.

## Relecteur A — Go, lentilles L1 (anti-ART) + L2 (multi-titre) + L4 (données)

Aucun constat recevable. Conditions vérifiées qui tiennent : 13 — vue `_latest` recréée après
l'ALTER et exposant les quatre colonnes ; ordre persister ↔ lecteur (21 placeholders) ;
vocabulaire de famille unifié entre canaux ; exclusion du sujet dans les deux taux ; FFA
intégral → taux `nil`, aucun donut à zéro ; parts exhaustives et exclusives = lobby ;
dénominateurs nuls → `nil` ; gating par capability ; INSERT-only sans allowlist élargie ; clé de
reprise du backfill ; planchers et seuil du rejet des bornes testés ; câblage E6.1 retiré sans
résidu ; compilation et tests ciblés verts. Non recevable mais corrigé : commentaire
`usage_summary_persister.go:170` (« NOT NULL » alors que la colonne est nullable).

## Relecteur B — web + tests, lentilles L5 (front) + L6 (couverture)

- P2 — `match-replay/model/equipmentUsageColumns.ts:314-320` : la vue match empile
  utilisé → gardé → lâché, le bloc partagé utilisé → lâché → gardé ; le plan se contredisait
  (§3.1 vs E4.6). **Décision S10** : ordre ordinal de §3.1 partout ; `usageGaugeModel.ts`,
  trois tests et l'item E4.6 réalignés dans la ronde de corrections.
- P2 — `_shared/usage/noLocalUsageCopies.guard.test.ts:41-49` : les regex à frontière de mot
  laissent passer une copie renommée d'un suffixe. Consigné (§6 du master plan), non traité.

Conditions vérifiées qui tiennent : 13 (états vides honnêtes, alias dérivés du type généré,
parts = lobby, variante comptes sans pourcentage ni parité, pile Sessions, objets sans famille
jamais rendus, i18n FR/EN sans clé morte, 0 couleur en dur, 0 requête neuve, tests vitest et
Go ciblés verts, oracles indépendants, cas limites Go couverts au niveau du calcul).

## Filet local

`make gate-push` EXIT 0 : 0 erreur lint, 30 avertissements préexistants, 0 test rouge.
Après corrections : vitest 281 fichiers / 3 396 tests verts sur `_shared`, `session-detail`,
`squad`, `match-replay`, `synthesis` ; `tsc -b --force` 0 ; eslint 0 sur les fichiers touchés.

## Bilan

P0 = 0, P1 = 0, P2 = 2 (1 corrigé par décision, 1 consigné). Une ronde, pas de seconde.
