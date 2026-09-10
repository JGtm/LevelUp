# Revue adversariale de la vague 3 — 2026-09-10

Deux relectures en contexte frais, une ronde chacune, règles du skill `adversarial-review`.

## Fonds de carte WebP (lot 3.3, `feat/v75..feat/fonds-carte-webp`)

Relecteur lancé par l'exécutant lui-même (hors consigne : une revue par vague, décidée par le
superviseur) — ses constats ont été retenus parce que recevables.

- P1 — `internal/service/replay_map_background.go` (`readBackgroundImage`) : aucun test
  synthétique n'exerçait le chemin `.webp` du calcul du type MIME (les tests handlers passent
  par un mock, le test « données réelles » dépendait du parc). **Corrigé `1734af2e0`** :
  `replay_map_background_webp_test.go` (sidecar synthétique `.webp`, `.png`, extension en
  majuscules), mutation prouvée (forcer `image/png` fait rougir le cas WebP).
- P1 — `internal/analysis/replay/map_background.go:30,37` : commentaire de schéma disant
  encore « PNG » (doc inversée). **Corrigé `1734af2e0`**.
- Non recevables (garde de traversée de chemin, table MIME, erreurs journalisées, tailles,
  `PathResolver`) : vérifiés et tenus. Dette préexistante confirmée : `decoupe_masque.go`
  code `.png` en dur (déjà consignée).

## Escouade hors cadre + grille tactique (lots 3.1 et 3.2, `1cfbf9cea..feat/v75`)

Lentilles L4 (données), L5 (front), L6 (couverture des tests).

- P1 — `internal/service/tactical_service_cellule.go:294-298` (`celluleVisee`) : le clic sur
  les plans « temps passé » et « routes » à un pas de 1 ou 2 m n'avait aucun test (tous les
  tests existants restaient au pas par défaut, où le réadressage est l'identité). **Corrigé
  `5af99b55b`** (`tactical_service_cellule_pas_test.go` : temps à 2 m, bornes négatives,
  route traversant deux cellules fines = une contribution), mutation prouvée (supprimer le
  réadressage fait rougir les trois tests).
- Conditions vérifiées qui tiennent : 8 — plancher en matchs distincts après regroupement ;
  règle du premier pas suffisant / tentative la plus fournie ; carte lisible à 0,5 m
  inchangée bit à bit ; convention `floor` identique client/serveur aux bornes négatives ;
  `planFrameStyle` sur bornes dégénérées ; `edgeMarkFor` (bord, coin, axe Y, échelle `k`) ;
  un seul chemin de dessin partagé écran/export ; i18n FR/EN complète.

## Filet local

`make gate-push` (résultat au master plan, 3.R).

## Bilan

P0 = 0, P1 = 3 (tous corrigés dans la vague, avec test de non-régression et mutation
prouvée), P2 = 0 nouveau. Pas de seconde ronde.
