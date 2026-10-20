# Critique de complétude n° 2 — campagne de grammaire, phase 1 (relue le 2026-10-02)

Rien n'est corrigé. Aucune commande `go` n'est lancée. J'ai vérifié par `git` et par lecture de fichiers.

Ce que j'ai contrôlé moi-même :
- **La branche J12** : `origin/feat/suite-audit-decodeur-j12` = `bc0e2511a`, 29 commits au-dessus de `8b894a677`. `git diff` donne 1 453 fichiers sur le dépôt et 1 130 sous `film/`. Pour chaque fichier, le nombre de lignes changées (`--numstat`) correspond à l'inventaire du PLAN §6.0.
- **Le golden** `frame_closure.golden` : `ks_000d5950 … listes_non_localisees=27`.
- **Le CARTE v2** : sa date de modification est le 2026-10-01 19:02. La critique n° 1 date de 20:25.
- **Le thought_log** : il a des entrées pour les étapes 1, 4 et 5, les mesures bis 1 à 3 et la correction des documents.

Abréviations : RAPPORT, PLAN, MESURES (= MESURES_CIBLEES), BIS_1/2/3, CARTE, J12R (= `J12_RECOUPEMENTS_2026-10-01.md`), ANALYSE.

## 1. Statut des points 1 à 39

| Pt | Statut | Ce qui le prouve (ou ce qui manque) |
|---|---|---|
| 1 | traité, avec un reste | PLAN §6.0 « Inventaire des recoupements », fichier par fichier ; D-60 ; RAPPORT §0.8. Les chiffres correspondent à mon `git diff`. Restes : `killsource/world.go` (+15/−5), `diagnostics.go` (nouveau) et `equivalence_lecteur_test.go` (+4/−4), tous touchés par J12, ne sont pas listés. Aucun essai de fusion ne prouve que « tous les conflits sont mécaniques » (voir N6). |
| 2 | traité | PLAN §6.0, « Option prouver maintenant… » : pour, contre, recommandation. |
| 3 | traité, avec un reste | PLAN §6.0 « Chaîne qui bloque J12 » et « Où tournent les gates ». Les heures sont des chiffres transmis, non vérifiés. Le plan SUITE dit seulement « la fusion de J12 reste après celle de J11 », et J11.6 est fait. ANALYSE §6 dessine la fusion de J12 AVANT J11.4. Les sources ne sont pas réconciliées. |
| 4 | traité | PLAN §6.0 « Montée de `grammar.Rev` contre la recuisson J11.4 » ; D7. |
| 5 | partiellement | RAPPORT §7 l.389 et J12R « Vérifié par diff » : le conflit J12.7 est résolu. Je l'ai vérifié : le sens 2 du ratchet interdit le tag sur la production. En revanche, rien ne traite la règle « 0 code mort ». `frame_closure_detail*.go`, `FrameClosure` et `LireBlocDeDatums` n'ont que des appelants dans `research/cmd_fermeture` (grep). « Comme `FrameClosure` » est un précédent, pas une justification. |
| 6 | traité | PLAN §6.3 D1 : « DÉJÀ TRANCHÉE », reformulée. |
| 7 | traité (décision en attente) | PLAN §6.3 D-RI. Mais sa recommandation crée un nouveau conflit (N3). |
| 8 | traité | PLAN §6.3 D-VEH. |
| 9 | traité | MESURES corr. 1 ; RAPPORT §2, ligne T1-3 (« perte brute 7 751, net +11 354 »). Plus aucune « perte nette −7 751 » dans le PLAN (grep). |
| 10 | traité | RAPPORT, convention « borne » et §3 (dénominateur fixe maximum et paquets sains) ; MESURES corr. 7 ; BIS_1 §7. |
| 11 | traité | MESURES corr. 6 ; BIS_1 §5. |
| 12 | traité | RAPPORT §1, notes « Région (iii) » ; MESURES corr. 5 ; BIS_1 §6. |
| 13 | traité | RAPPORT §1 : ligne (iii') ouverte (89 / 13), total 5 170 / 239. |
| 14 | traité | RAPPORT §0.1 (92,7 contre 92,5 %), §7 l.395 ; PLAN L3 (4 208 + 123) ; D-47. |
| 15 | traité, avec un reste | `VERIFICATIONS_ADVERSES.md` : 43 sections, deux lentilles, bilan 41/1/1. Reste : le thought_log n'a toujours aucune entrée pour les étapes 2 et 3. |
| 16 | traité | RAPPORT §0.1 (« Probable seulement », trois réserves), §2 T1, §5 « Ouvert » (D-35). |
| 17 | traité, avec un reste | RAPPORT §2 T1-3 (« non établi pour le slot ») ; PLAN L1 « Réserves » ; MESURES corr. 10. Reste : « établi pour la tête sur HI_1_12_0 » (voir N10). |
| 18 | traité | RAPPORT §4.2 ; MESURES corr. 11. |
| 19 | traité | RAPPORT §1 « Liaisons fausses » et §4.9 ; PLAN L1 « Réserves ». |
| 20 | traité | RAPPORT §0.2, première limite. |
| 21 | traité | BIS_1 §3 ; MESURES corr. 9 ; RAPPORT §0.6. |
| 22 | traité (borne basse seulement) | BIS_1 §1 ; RAPPORT §3 « Moitié entrées ». Le dénominateur exact reste à faire (`[!]` proposé). |
| 23 | traité | BIS_1 §2 (`tete-bloc+inv`) ; D-46. |
| 24 | traité | BIS_2 §2 : double essai mesuré et témoins décalés d'un bit. Ancien L5 sorti (PLAN §6.1). |
| 25 | traité, avec un reste | BIS_2 §3.1 et §3.3 ; L6a et L6b. Reste : le juge des invariants n'a pas été joué sur les 17 A/B par site (BIS_2 §6). |
| 26 | traité | BIS_2 §5 ; PLAN L2, gate (`81c02726` témoin du lot). |
| 27 | traité | BIS_3 §0 à §5 ; lots L9 et L10, recherches R-P3 et R-P6 ; P2 couvert par L1 et L0. |
| 28 | traité | BIS_3 §6 ; lot L8 ; (iii) sortie de la borne de L1 (D-45). |
| 29 | traité | BIS_1 §4 ; PLAN §6.1, ligne L1 ; §6.2, L1 « Gain mesuré ». |
| 30 | partiellement | R-L3 est créée (PLAN §6.1, table des recherches), mais sans porteur, sans méthode et sans gate propre. Comparer avec R-L4 dans L4. La cause reste non élucidée. |
| 31 | traité pour les lots de marche | PLAN §6.0 gate 3 ; `walk.go` dans les fichiers de L1 ; D-17 relié. Ce gate manque pour les lots de composants (N1). |
| 32 | partiellement | PLAN §6.0 gate 4 : la mesure est définie, mais « Plafond à faire valider par l'utilisateur ». Aucun seuil, et cette décision ne figure pas au §6.3 (N8). |
| 33 | traité | PLAN §6.0 gate 2 (« AUCUN FILM »). Pas repris au RAPPORT §7. |
| 34 | traité | PLAN §6.0 gate 5 ; D-61 (vérifié : `listes_non_localisees=27`) ; L0.3 ; recopie de la sonde citée. |
| 35 | traité | PLAN L4, « Prérequis, avec porteur et méthode » ; D5. |
| 36 | traité | PLAN §6.1 : L5 sorti, L6a et L6b mesurés, L3 et L4 marqués « estimé ». |
| 37 | partiellement | Le journal §4 est complété et le §6.4 propose des statuts. Mais au PLAN §2, les items 1.3, 4.1, 5.1 et 5.2 sont toujours `[ ]`, et l'en-tête dit encore « EN COURS — phase 1 lancée ». |
| 38 | partiellement | `[~]` et `[!]` sont proposés au PLAN §6.4, pas appliqués : l'item 1.3 est toujours `[ ]`. |
| 39 | traité | PLAN §4, journal (vet, tests `-tags=research`, `-run 'Closure|Fermeture|GrammarRev'`) ; BIS_1 §10 ; BIS_2 §8 ; BIS_3 §10. Ce gate n'inclut pas `archlint` (N2). |

Le tableau §7 du RAPPORT (« par point ») ne cite pas les points 30 à 37 ni 39. On ne peut donc pas retrouver ces points depuis le RAPPORT.

## 2. Nouveaux manques

### Gates absents

**N1 — Les lots de composants modifient aussi killsource, sans gate killsource.**
- Le gate 3 du PLAN §6.0 ne vise que « tout lot qui touche une marche ».
- Or `facts/killsource/walk.go:69` appelle `grammar.DecodeFrameRecords`, donc le dispatch commun. D-RI l'écrit d'ailleurs : « lecteurs appelés par le dispatch commun à toutes les marches ».
- Le gate de L8 se limite aux « points 1, 2, 6 et 7 ». L2, L3, L4 et L6 peuvent changer la sortie killsource sans équivalence, sans montée de `killsource.Rev` et sans backfill déclaré.

**N2 — `go test ./internal/archlint/` est probablement rouge dès maintenant.** C'est estimé par lecture du code, pas exécuté.
- `archlint/film_file_size_test.go` balaie tous les `.go` sous `film/`, tests compris, avec un seuil de 500 lignes hors table.
- `campagne_bis1_research_test.go` fait 616 lignes et `campagne_bis2_vehicules_research_test.go` 571 (mesuré, `wc -l`).
- Aucun gate de la phase 1 ni des bis ne lance `archlint`. Or le gate 1 de la phase 2 l'exige.

**N3 — La surcouche n'est ni compilée en CI ni protégée contre J12.**
- Les fichiers taggés `research && campagne_overlay` (bis2 positions et véhicules, bis3 `ti3`) échappent au `go vet -tags=research ./...` de la CI.
- `overlay.json` remplace des fichiers ENTIERS par des chemins absolus vers ce worktree.
- J12 modifie `lecteur_position.go` (+1/−1) et `lecteur_position_exceptions.go` (+4/−4). Après la fusion, les copies annuleraient ces changements dans la mesure.
- Le PLAN ne prévoit aucune resynchronisation des copies pour L2, L6 et L8. D10 ne pose pas cette question.

### Décisions cachées

**N4 — La recommandation de D-RI crée un cycle avec une décision déjà prise.**
- D-RI dit « L1 en entier (avec killsource) avec ou après le lot 2.7, comme ANALYSE l'ordonnait ».
- Or ANALYSE §7 (2), retenue par l'utilisateur, dit « le lot 2.7 attend la campagne ». ANALYSE §5.1 dit « avant le lot 2.7 ».
- Le plus gros levier de la campagne attendrait donc 2.7, qui attend la campagne. L'attribution « comme ANALYSE l'ordonnait » est fausse.

**N5 — Fusionner les composants pendant l'étape 1 casse les preuves de cette étape.**
- D-RI fait fusionner les lots de composants « avant ou pendant l'étape 1 » de la représentation intermédiaire.
- Or RI 1.2 et 1.3 ont pour gate `replay-equiv 0` et `frame_closure.golden` / `keyframe_closure.golden` identiques (ANALYSE §3.3).
- Un lot de composant fusionné pendant l'étape 1 oblige à refiger ces références. Ce couplage n'est pas écrit.

**N6 — Le PLAN se contredit sur la montée de `grammar.Rev`.**
- Le gate 1 du §6.0 exige une montée par lot (« `grammar.Rev` monte avec son entrée de `rev_chronique.go` »).
- Le même §6.0 et D7 recommandent « une montée par vague ».
- La règle qui l'emporte n'est pas écrite.

**N7 — Le dénominateur « fixe maximum » n'a pas de définition utilisable en phase 2.**
- Il vaut aujourd'hui le maximum sur les 14 marches de bis 1.
- Or D-42 et RAPPORT §3 montrent qu'il monte avec chaque lot (2 574 513 → 2 759 700 sous `ti=3`).
- D1 (a) et la règle « Pourcentages » du §6.0 ne disent pas sur quel ensemble de marches le recalculer.

**N8 — Le plafond de performance et de mémoire (gate 4) relève de l'utilisateur, mais n'apparaît pas dans D1 à D10.**

### Affirmations sans preuve, ou plus fortes que la preuve

**N9 — « Tous les conflits sont mécaniques » (RAPPORT §0.8) n'est pas prouvé.**
- Aucun essai de fusion (`git merge-tree` ou équivalent).
- J12.3 retire `slog` de `film_context.go` (+28/−14), un fichier de L6a. J12.4 réécrit `registry.go` (+11/−16), un fichier de L8. Ce sont des contraintes de structure, pas des tris.

**N10 — « La phase 2 seule ne fera pas atteindre le déclencheur » (RAPPORT §3) est une extrapolation.**
- Les leviers sont mesurés séparément (L1, L8, L2, L9). Leur combinaison sur HI_1_13_0 n'est ni mesurée ni estimée.
- La phrase n'est pas marquée « estimé ».

**N11 — La grammaire `ti=3` est classée « établi » (RAPPORT §0.4 et §5) alors qu'une partie est déduite.**
- BIS_3 §8 et PLAN L8 disent que l'appartenance de la table `0x143d07af0` (`i1`, 26 bits) à `ti=3` est « déduite, pas lue ».
- Selon la propre convention du RAPPORT, la partie `i1` est donc « probable ».

**N12 — « Tête établie sur HI_1_12_0 » (RAPPORT §2, T1-3) repose sur trop peu de cas.**
- Le 97,7 % ne concerne que HI_1_13_0.
- Pour HI_1_12_0, il n'y a que le rang 0 : pool 1 à 13/15, pool 4 à 10/18 (MESURES §T1-3).

**N13 — Le CARTE v2, instrument du gate de chaque lot, n'est pas révisé.**
- Il publie encore l'estimateur des entrées tautologique (§5) et les classes « naissance non lue » et « réalloué » qui ont les deux erreurs D-43 et D-44.
- Il n'a aucun bandeau de correction. Le RAPPORT §1 s'appuie sur son 84,4 %.

**N14 — L'ampleur de l'erreur D-44 sur le corpus n'est pas mesurée.**
- Le 3 560 cité au RAPPORT §1 vient de `81c02726`, qui est HORS des 20 films.
- La part des NEW lus mais désynchronisés dans les 213 033 « naissances attestées » du corpus est inconnue. La région (iii) en montre au moins 13 978 paquets sur HI_1_13_0.

### Chiffres incohérents

**N15 — La colonne « Estimateur CARTE §5 » de BIS_1 §1 ne correspond pas au CARTE §5.**

| | BIS_1 §1 | CARTE §5 |
|---|---|---|
| HI_1_10_0 | 22,2 % | 21,9 % |
| corpus | 45,3 % | 43,6 % |
| HI_1_4_1 | ≤ 1,6 % | « - » |

Par ailleurs, la phrase « les deux dernières colonnes sont égales build par build » est fausse sur la ligne des vieux builds de BIS_1 lui-même (≤ 1,6 % contre ≤ 6,3 %).

**N16 — « 279 paquets fermés sur 284 425 » (RAPPORT §2 T5, MESURES l.459) a le mauvais dénominateur.** On a 284 704 − 279 = 284 425 : le dénominateur est le nombre de fermés NON contredits. Il faut 279 / 284 704.

**N17 — `DEL ti=0` vaut 658 au BIS_3 §0, mais 622 aux BIS_3 §3.2 et §3.4, au D-56 et au RAPPORT §4.6.**

**N18 — Sur `81c02726`, BIS_2 donne 4 149 paquets « hors cadre » au §5.1 et 4 150 au §5.3.**

**N19 — Trois écarts de libellé mineurs :**
- D-5 dit « 13 644 paquets hors cadre » ; BIS_3 §1.2 dit 13 644 paquets, dont 13 643 hors cadre.
- RAPPORT §3 dit « Sous l'oracle, HI_1_13_0 est ≥ 48,5 % » : c'est la valeur de l'oracle-NEW. Sous (i)+(ii), qui est l'oracle de référence partout ailleurs, BIS_1 §7 donne 48,8 %.
- BIS_1 §9 compte 598 eid, contre 623 au §6 et au RAPPORT, sans expliquer la différence de 25.

**N20 — J12R se contredit.** La section « Faits transmis » dit la branche J12 « NON poussée, pas vérifiés par diff ». La section « Vérifié par diff » s'appuie sur `origin/…`, qui est donc poussée. Aucune mention ne dit que la première section est périmée.

**N21 — Sous la grammaire T7, le « hors cadre » de HI_1_12_0 MONTE (3 249 → 4 029, BIS_2 §5.4) alors que ses paquets fermés triplent.** Ce point n'est commenté nulle part.

Fichiers relus, sous `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-campagne-grammaire/` :
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/CRITIQUE_COMPLETUDE_1.md`
- `.ai/V7.5/film_re/RAPPORT_CAMPAGNE_GRAMMAIRE_PHASE1_2026-10-01.md`
- `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (§2, §4, §5, §6)
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/MESURES_CIBLEES.md`
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/MESURES_BIS_1.md`
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/MESURES_BIS_2.md`
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/MESURES_BIS_3_POPULATIONS.md`
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/J12_RECOUPEMENTS_2026-10-01.md`
- `.ai/V7.5/film_re/CARTE_FERMETURE_V2_2026-10-01.md`
- `apps/go-api/internal/archlint/film_file_size_test.go`
- `apps/go-api/internal/games/halo_infinite/film/internal/facts/killsource/walk.go`