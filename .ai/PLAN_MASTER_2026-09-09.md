# PLAN MASTER — arbitrage et pilotage des chantiers ouverts (2026-09-09)

> Rédigé par le superviseur (Fable 5.1) après vérification SUR PIÈCES le 2026-09-09, HEAD
> `feat/v75` = `19194fd31` (local, 1 commit d'avance sur `origin`). Ce plan ORDONNE sept
> documents et n'en réécrit aucun : chaque lot renvoie à son plan source, qui reste la liste
> cochable. Contrat d'exécution : skill `plan-execution`. Reprise de session : §7.
>
> Critère d'arbitrage fixé par l'utilisateur : criticité x gain (UX/UI ou correction visible)
> / effort. **Gain mineur = on ne fait pas.** Découvertes : P0 immédiat, tout le reste en
> tout dernier. Revue adversariale : UNE par vague, jamais par lot (sobriété des quotas).

## 0. État réel au 2026-09-09 (vérifié, ne pas ré-instruire)

| Chantier | État constaté | Preuve |
|---|---|---|
| Orchestration vague 3 — P-décodeur E2 (I0-I5 + E2-bis) | **CODE COMPLET**, tous gates verts sauf le **gate corpus complet sur le parc** `[!]` (exige ce poste). 3 commits de polarité `replaydiff` au sommet + 1 diff non commité (tests verts) : c'est la « vérification des gates avant P3 » | `feat/v2-decodeur-e2` = 28 commits d'avance sur v75 ; `LevelUp-wt-e2` ; `RESTES_E2_2026-09-08.md` §E2-bis.6 |
| Vague A (lot court, 7 correctifs) | **8 commits FAITS**, branche `wt/lot-court` (pas de CI sur `wt/`), **non fusionnée** | `git log feat/v75..wt/lot-court` = 8 |
| Vague B (identités interverties) | **Jamais démarrée** — et **SUPERSÉDÉE** : le sondage E2 a trouvé 7 paires ÉCHANGÉES par le pont par morts, le lien direct (record de création `ti=35`) les corrige, D14 impose « 0 vie nommée par le pont » | `SONDAGE_E2_BIPEDE_INDEX_2026-09-08.md` §2.5 ; thought_log E2 |
| Vague C (formes) | Jamais démarrée. C5/C6 absorbés par le plan Équipement. Tâche hors lot (recuisson 15 + 7) absorbée par la recuisson du parc post-E2 | plan vague C, annotations du 09-09 |
| Escouade hors cadre | A0 fait (ADR 0033 + 2 verrous) mais **NON COMMITÉ** (fichiers untracked) ; A1..B4 à faire | `LevelUp-wt-escouade-hors-cadre` `git status` |
| Équipement « servi ou gâché » | Jamais démarré (E0..E6) | pas de branche |
| Fonds de carte WebP/ETag | Jamais démarré | pas de branche ; 109 PNG / 45 Mo mesurés |
| Trois chantiers du 09-09 (lisibilité usage Sessions, retours UI, légendes couleurs) | **TERMINÉS mais NON COMMITÉS** dans le worktree partagé : 93 fichiers modifiés + 23 nouveaux, en attente d'arbitrage de branche par l'utilisateur. « Retours UI » **non vérifié en navigateur** | `git status` de `LevelUp-go-migration` ; thought_log l.128-130, 234, 302 |
| `main` | **13 commits** hors `feat/v75` (hotfix classement 7.3.1 + collections non nulles) | `git log feat/v75..main` |

## 1. Arbitrage

| Chantier / item | Criticité | Gain utilisateur | Effort | Verdict |
|---|---|---|---|---|
| E2 : gate corpus, fusion, backfill-killsource, recuisson du parc | **P0** (identités fausses servies tant que non fusionné) | fort (rejeu juste, K/D/A, kill feed des matchs récents à 0-5 % d'icônes = registre L133) | S (superviseur, temps machine) | **TRAITER EN PREMIER** |
| Vague A : fusion `wt/lot-court` | P1 (7 défauts visibles déjà corrigés) | fort | XS | **TRAITER** (accord fusion) |
| 93 fichiers non commités du 09-09 | P1 (travail fini, invisible, bloque C3/A3/E4) | fort | XS + 1 passe navigateur | **TRAITER** (accord commit) |
| `main` → `feat/v75` | P1 (le hotfix classement se re-perdrait au passage vers `main`) | moyen | XS-S | **TRAITER** (accord fusion) |
| Vague B, phases 1-2 | — | — | L | **ABANDONNER** : remplacée par UNE vérification (`swap.sh` = 0 sur les 4 témoins recuits) dans le gate E2 |
| Escouade A1-A3 (compte unique) | P1 (trois nombres différents à l'écran) | fort | M | **TRAITER** |
| Vague C — C4 médailles en images | P1 (décision utilisateur du 08-09 : l'anneau est un mauvais encodage) | fort | M | **TRAITER** |
| Vague C — C1/C2 grille canonique | P2 | moyen (maquette validée : « aéré et joli », absence visible) | S+S | **TRAITER** (C2 : Synthèse seule, 4 autres heatmaps en allowlist datée) |
| Vague C — C3 nuage d'isolement | P2 | moyen (code mort à rebrancher, maquette validée) | S-M | **TRAITER** après le commit des 93 fichiers |
| Équipement E0-E2 (vue match, sans recuisson) | P1 (maquette validée, la vue match sert tout dès maintenant) | fort | M | **TRAITER** |
| Équipement E3 (backend, migration, recuisson des résumés) | P1 | fort (débloque Sessions/Solo/Escouade) | L, persist | **TRAITER**, vague à part, Opus |
| Équipement E4-E6 | P2 | fort | M+M+M | **TRAITER** après E3 |
| Escouade B0-B4 (pions hors cadre) | P2 | moyen (perte de repère uniquement à zoom > 1) | M | **TRAITER en vague 3** |
| Fonds de carte — étape 1 ETag/304 | P2 (ETag inerte sur `assets.go`, 3e copie du motif) | faible-moyen (retéléchargements après 1 h) | S | **TRAITER** (petit, TDD, helper unique) |
| Fonds de carte — étapes 0, 2-5 WebP sans perte | P3 | faible pour l'utilisateur (poids du dépôt et du 1er chargement) ; risque export vidéo | M | **DERNIER**, conditionné à D10 (>= 20 % mesuré à l'étape 0), TDD, recette navigateur 12 verdicts |
| Tactique point 21 : une carte ouverte n'affiche rien (plancher 3 matchs / cellule 0,5 m) | **P1** (page entière vide) | fort | M (Go `analysis/tactical`) + décision produit | **TRAITER en vague 3** — décision D6 ci-dessous |
| Registre : audit anti-bombe-RAM (L505/L506 au 09-09) | — | nul (sûreté) | — | **RETIRÉ de la séquence de release Notion par l'utilisateur** (vérifié sur la page « Backlog LevelUp » le 09-09 : 15 cases, aucune ne le mentionne) ; le journal du 27/08 le dit « en partie soldé par filmproc + BUILDALL ». Les deux lignes du registre sont PÉRIMÉES : à re-statuer (barrer avec renvoi à `replayartifacts/buildone.go` et `filmproc`) lors du lot 0.5 |
| Registre : clôtures administratives (§5) | — | — | XS | **TRAITER** dans le commit de clôture de vague 0 |
| Registre : reste (voir §5) | P2-P3 | faible | — | **REPORTER**, condition inchangée |

## 2. Décisions PRISES par le superviseur (dans son mandat)

| # | Décision |
|---|---|
| S1 | Un lot = un worktree dédié `LevelUp-wt-<slug>` + branche `feat/<slug>` depuis `feat/v75` (CI sur `feat/`). Jamais le worktree partagé. Les agents commitent sur leur branche ; toute fusion dans `feat/v75` attend l'accord de l'utilisateur |
| S2 | Modèle : Sonnet pour le mécanique / web / tests ; Opus pour E0 (recherche), E3 (persist), point 21 (algo + décision). Effort ajusté par lot dans le brief |
| S3 | Revue adversariale : une par vague, sur le diff cumulé, avant fusion. Vague 2 (persist) : obligatoire. Vague 1 : oui (contrat Go + 4 lots web). Vague 3 : seulement si WebP est exécuté |
| S4 | Vague B abandonnée ; la preuve de non-régression du P0 « identités » = `swap.sh` sur les 4 témoins après recuisson E2 (§3, lot 0.4). Si `malplaces > 0` subsiste : rouvrir la phase 1 de la vague B, pas avant |
| S5 | ETag : le helper unique réutilise la logique d'`If-None-Match` déjà présente dans `handlers/helpers.go:writeJSONCached` (le plan croyait qu'aucun 304 n'existait ; il y en a 5). Le plan D9 est amendé : `writeJSONCached` MIGRE vers le helper, garde-rail grep sur `Header().Set("ETag"` |
| S6 | C2 : les 4 autres `type: 'heatmap'` (`ascension/ActivityCalendarChart`, `explorer/ExplorerActivityHeatmapChart`, `palmares/RelationsMomentsHeatmap`, `squad/charts/squadMapHeatmapChart`) entrent dans l'allowlist DATÉE du garde-rail, pas dans le lot (périmètre fermé du plan) |
| S7 | La recuisson du parc post-E2 remplace la « tâche hors lot » de la vague C (15 artefacts schéma 38) et les lignes L567/569/584/602 du registre. **Corrigé le 09-09 (soir)** : les 7 artefacts aux bornes fausses n'étaient PAS couverts — le correctif vivait sur `wt/bornes-aberrantes` (`490dc595e`), jamais fusionné, et la recuisson de 17:37 a été jouée sans lui (`81c02726` porte toujours z = −325 m, vérifié sur pièces). Fusionné en `4bd2a7969` (D11) ; la recuisson des 7 artefacts est rattachée au lot 3.2 |
| S8 | `backfill-killsource` post-E2 (révision d'isolement bumpée) est la MÊME opération que le report L133 (arme du kill 0-5 % avril-juillet) : une passe, deux clôtures |
| S9 | **`deployed_*` reste au contrat de session** (09-09 soir). E4 a remplacé ces grandeurs à l'écran par `equipment_<famille>` ; les retirer du contrat exigerait une révision de résumé (us5) et une recuisson pour un gain mineur (clés non affichées, encore utilisées par la classification et le repli). Reporté à P5 (nettoyage du registre) — noté au §6 |
| S10 | **Ordre canonique des trois issues = utilisé → gardé → lâché** (09-09 soir, revue 2.R). Le plan équipement se contredisait : la table §3.1 (gamme ordinale bon / neutre / mauvais, qui porte la justification) donne cet ordre, l'item E4.6 en donnait un autre (utilisé → lâché → gardé), et les pages le suivaient chacune à sa façon (vue match d'un côté, Sessions/Synthèse/Escouade de l'autre). Le modèle partagé `usageGaugeModel.ts` et son test sont réalignés sur §3.1 ; E4.6 est amendé dans le plan équipement |

## 3. Décisions qui appartiennent à l'utilisateur (bloquantes, avec recommandation)

| # | Question | Bloque | Reco |
|---|---|---|---|
| D1 | Commiter les 93 + 23 fichiers du worktree partagé sur une branche `feat/retours-ui-0909` (3 chantiers du 09-09), après UNE passe navigateur sur « retours UI » | C3, A3, E4, E5 | **Oui.** Sans commit, tout lot web qui touche `squad/`, `session-detail/`, `components/charts/` fusionnera en conflit |
| D2 | Fusionner `wt/lot-court` (vague A, 8 correctifs) dans `feat/v75` | rien | **Oui, tout de suite** : 7 défauts visibles corrigés, jamais passés en CI |
| D3 | Fusionner `main` dans `feat/v75` (13 commits hotfix classement) | release | **Oui, avant tout autre merge** — plus tard = plus de conflits |
| D4 | Fusionner `feat/v2-decodeur-e2` dans `feat/v75` si le gate corpus est propre (lot 0.4) puis lancer `backfill-killsource` + recuisson du parc (heures machine) | vague 1 rejeu, P3 | **Oui** dès le verdict du gate |
| D5 | Vague B abandonnée au profit de la vérification S4 | — | **Oui** |
| D6 | Tactique point 21 : pas de grille **adaptatif** (0,5 → 1 → 2 m jusqu'à ce que >= N cellules atteignent le plancher) ou plancher abaissé à 2 ? | vague 3 | **Adaptatif** : le plancher de 3 matchs distincts est ce qui rend une cellule fiable ; abaisser ment |
| D7 | WebP : exécuter les étapes 2-5 si l'étape 0 mesure >= 20 % (D10 du plan) — sinon ETag seul | vague 3 | **Oui, en dernier** ; la recette export vidéo (12 verdicts) se joue par l'utilisateur ou avec navigateur |
| D8 | ~~Audit anti-bombe-RAM à la séquence de release~~ — **SANS OBJET** : l'utilisateur l'a retiré de la séquence Notion (vérifié le 09-09) ; le registre L505/L506 est en retard sur Notion et sera barré au lot 0.5 | — | — |
| D8bis | Séquence Notion « à dérouler à la release » : la case « Re-cuisson du parc au schéma 48 » doit passer à **50** (E2 fusionné le 09-09) — modification de la page Notion, à faire par l'utilisateur ou sur son accord | release | À mettre à jour |
| D9 | Seuil d'activation du rejeu public (registre L120, 88 %) : re-statuer maintenant ? | — | Une ligne de décision, après recuisson du parc |
| D10 | **Tag de version / release : JAMAIS par le superviseur** (consigne utilisateur 2026-09-09). Aucun tag, aucun push sur `main` ; la séquence de release (R9) reste à la main de l'utilisateur | — | consigne ferme |
| D11 | **Deux branches oubliées du plan** (question de l'utilisateur, 09-09 soir) : `wt/bornes-aberrantes` (correctif de cuisson, 1 commit) et `wt/orchestration-0907` (journal du superviseur du 09-08, 10 commits de docs) n'étaient reprises nulle part. Décision : **fusionner le code maintenant, recuire les 7 artefacts en vague 3 (lot 3.2)**. Fusions `4bd2a7969` (code) et `2cc39f89b` (docs, union) le 09-09 |

Tranché le 2026-09-09 par l'utilisateur : D1, D2, D3, D4 (« champ libre pour commit », seule
session active) — exécutées en vague 0 ; D10 ferme. Questionnaire du 09-09 : lot 1.6 → **P5**
(pas en vague 1) ; D6 → **pas adaptatif** ; D7 → **mesurer d'abord, décider au chiffre** ;
D8bis → **case Notion mise à jour par le superviseur** (schéma 48 → 50, faite le 09-09).

## 4. Séquence d'exécution

Une vague = un diff cumulé = une revue = une CI. Un seul lot en vol par domaine
(Go sync-persist / web / rejeu). Statuts : `[x]` fait · `[~]` couvert ailleurs · `[!]` non
traité (justifié).

### Vague 0 — consolidation (superviseur, Sonnet pour aucun)

| # | Lot | Exécutant | Gate | Statut |
|---|---|---|---|---|
| 0.1 | D3 : `git merge main` dans `feat/v75`, gates Go+web, push | superviseur, accord D3 | `make go-api-test`, `make check-types`, `make test-web` | [x] 09-09 : `2751a484f` (main dans feat/v75, hotfix 7.3.1), main non fusionné = 0, gates verts, CI de vague verte |
| 0.2 | D2 : fusion `wt/lot-court` | superviseur, accord D2 | vitest des fichiers touchés, typecheck | [x] 09-09 : `e7926ded3` (wt/lot-court, 7 correctifs courts), reste = 0 |
| 0.3 | D1 : passe navigateur « retours UI » puis commit des 116 fichiers sur `feat/retours-ui-0909`, fusion | superviseur, accord D1 | `make check-types`, `make test-web`, `make generate-types` sans diff | [x] 09-09 : `9212a1b0e` sur `feat/retours-ui-0909`, fusionnée (reste = 0) ; passe navigateur faite à la revue de vague 1 |
| 0.4 | E2 : commit du diff `replaydiff` (polarité par-slot → par-xuid), `make replay-corpus-gate --reference=base`, verdict ligne par ligne (une ligne attendue : `flagCarries.homeByObject` sur `3372e7eb`), fusion (D4), `levelup backfill-killsource`, recuisson du parc (`backfill-replay --only-existing`, un film à la fois), puis `swap.sh` sur `8bc6074f d8b13ec2 a4083bd2 bf2a9f05` → `malplaces = 0` | superviseur | gate exit 0 ; `swap.sh` = 0 ; `jq .coverage.vehicles` présent sur les 15 artefacts schéma 38 | [x] 09-09 (boîte statuée le soir) : gate joué (9 lignes de perte sur 4 témoins, toutes instruites et acceptées au registre — dont `fb1a1a72` : propriétaire du slot 625 corrigé, bornes identiques) ; polarité `replaydiff` commitée `43320cd6f` ; **fusion faite `4dec9bd65`** ; `backfill-killsource` lancé détaché à 16:07 (737 films, 1 967 matchs en crédit ; serveur de dev arrêté pour la passe, journal scratchpad `backfill_killsource.log`). Chaîne de backfills terminée 17:59 (sauvegarde DB, médailles 415 matchs, T0, cache appauvri, **recuisson 64 artefacts schéma 50 en 23 min, pic 539 Mio, `coverage.vehicles` 64/64**, T0 film, résumé d'usage 64, Assaut 0 film au parc) ; **P0 découvert** (voies du registre refusées par `match_lives`, corrigé `bc61b6636`) → passe films de killsource rejouée à 18:00 avec le binaire corrigé ; **`swap.sh` = 0 sur les 4 témoins et 37/38 artefacts** (résidu `58864b3c` au registre). Reste : fin de killsource v2, redémarrage du serveur, revues navigateur |
| 0.5 | Clôtures administratives du registre (§5) + annotation « vague B abandonnée » dans son plan | superviseur | diff docs seul | [x] 09-09 (soir) : 22 lignes closes et 2 reclassées dans `REGISTRE_REPORTS.md` — les numéros du §5 datent du tri du matin et sont décalés de +19 (unions du jour) : L164→183, L117→136, L589→608, L397→416, L524→543, L18→37, L593/601→612/620, L434→453, L482/484→501/503, L480→499, L254/213/368→273/232/387, L588→607, L14→33, L567/569/584/602→586/588/603/621, L133→152 ; L505/506 (numéros courants) reclassés hors séquence de release. Annotation « vague B abandonnée » déjà posée le matin |

### Vague 1 — P1 visibles, effort modéré (parallèle : 4 lots, domaines disjoints)

| # | Lot | Source | Exécutant | Worktree / branche | Gate | Statut |
|---|---|---|---|---|---|---|
| 1.1 | Escouade A1 + A2 (contrat Go de l'écart + source unique web), TDD ; **A3 attend D1** (touche `squad/i18n.ts` modifié dans le worktree partagé) | plan escouade §3 | Sonnet, effort élevé | `LevelUp-wt-escouade-hors-cadre` / `wt/escouade-hors-cadre` (existant, A0 à commiter d'abord) | gates A1, A2 du plan | [x] 09-09 : `39bbb2416` (A0), `676bda47a` (A1), `97024d24d` (A2) sur `wt/escouade-hors-cadre` ; TDD observé ; Go tests/vet/lint verts, `generate-types` idempotent ; web tsc 0, vitest 7 032 verts, ratchet `singleCountSource.guard` à allowlist vide ; vérifié sur pièces (0 comparaison de slug). A3 en attente D1 (désormais levée) → lot 1.5 |
| 1.2 | C4 : médailles de la frise en images, piste élargie, infobulle titre+description | plan vague C, C4 | Sonnet, effort moyen | `LevelUp-wt-frise-medailles` / `feat/frise-medailles-images` | gate C4 + de visu Origin `8bc6074f` 3:49 | [x] 09-09 : `2a4969a72` sur `feat/frise-medailles-images` ; piste 18 → 24 px, `TrackMedal` porte l'identité complète (`MedalEvent`), infobulle héritée de `MedalBadges` ; vitest match-replay 2 572 tests verts, tsc 0 ; vérifié sur pièces par le superviseur (0 `ring-1`, 0 couleur en dur). De visu `[~]` à la revue de vague. Découverte : `rosterHeight.guard` garde le plafond des fiches, pas la hauteur de piste |
| 1.3 | Fonds de carte étape 1 : ETag/304 centralisé (S5), TDD, garde-rail grep | plan WebP étape 1 | Sonnet, effort bas | `LevelUp-wt-fonds-etag` / `feat/fonds-carte-etag` | `go test ./internal/api/handlers/`, `go build ./...` | [x] 09-09 : `8356118c8` sur `feat/fonds-carte-etag` ; helper `servirBlobAvecETag`, 4 sites migrés (replay, tactical, assets, `writeJSONCached`), garde-rail grep prouvé par mutation ; handlers/build/vet/lint verts ; vérifié sur pièces (un seul `Set("ETag"` = `cache_http.go`). Découverte : 3 handlers Huma (`capabilities`, `feature_matrix`, `field_mappings`) calculent leur propre SHA-256, non migrables (contrat déclaratif) |
| 1.4 | C1 : `Heatmap2DChart` padding + absence hachurée + plafond ; C2 : garde-rail grep (allowlist S6) — **migration de `SynthesisHeatmapChart` attend D1** (9 fichiers de `synthesis/` modifiés dans le worktree partagé) | plan vague C, C1-C2 | Sonnet, effort moyen | `LevelUp-wt-formes` / `feat/formes-maquettes` | gates C1, C2 | [x] 09-09 : `ff85316d8` (C1) + `4dcde150a` (C2 garde-rail, allowlist datée des 5 sites, mordant prouvé) sur `feat/formes-maquettes` ; charts 283 tests + consommateurs 806 verts, tsc 0 ; vérifié sur pièces (0 couleur en dur, `borderWidth` + `decal`). Migration Synthèse → lot 1.5. Découverte : commentaire de `squadEchange.logic.ts:200-203` partiellement périmé |
| 1.5 | C3 nuage d'isolement + A3 (lisibilité de l'écart) + C2 migration Synthèse | plans C et escouade | Sonnet | mêmes worktrees, rebasés sur v75 après D1 | gates C3, A3 | [x] 09-09 : `e897b7d3f` (A3), `e06ffa297` (C3), `7568a7bd2` (C2 Synthèse) sur `feat/vague1-integration` (worktree `LevelUp-wt-vague1`, intégration des 4 lots + ces 3 items) ; gates d'intégration avant le lot : Go verts, tsc 0, vitest 7 124 ; gates du lot : squad+shell 648, squad 487, charts+synthesis 409, grep 4 sites restants. A3.5 revue navigateur `[~]` revue de vague |
| 1.6 | **Impact résiduel du registre d'identité (vérification utilisateur du 09-09)** : six consommateurs lisent encore le pont APLATI `PontParSlot()` (premier occupant d'un slot recyclé) — `bomb_carries.go:142`, `bomb_stats_document.go:77`, `build.go:152` (frags sous équipement) et `:219` (ramassages), `inventory_dead_readings.go:42`, et surtout `killcollector/positions.go:301` (positions de kill persistées AU SYNC, servies à Tactique). Sur un film long à slots recyclés (`084a804d` : 123 slots sur 256), ces lectures peuvent créditer le mauvais joueur. Le registre sait répondre à l'instant (`XUIDAt`). Lot : migrer les six sites vers `XUIDAt`/`PontEpure`, vider l'allowlist de `PontParSlot`, gate corpus, `backfill-killsource` à rejouer si `positions.go` change (bump `IsolationDecoderRev`) | E2-bis découverte 2 (prévue P5, avancée) | Opus, effort moyen | gate corpus 7/7, garde-rail à allowlist vide | [!] **tranché par l'utilisateur le 09-09 (questionnaire) : reste en P5 comme prévu par E2-bis.** Pas de lot en vague 1 ; l'exemption écrite sur `PontParSlot` tient lieu de garde |
| 1.7 | **`kill_positions` par passe + protection append-only** (pré-requis Notion du backfill distance, constaté non fait le 09-09 : la table a bien id + `written_at` + vue `_latest` par CLÉ, mais ni `decode_pass` ni inscription à la liste des tables protégées, contrairement à `kill_openings` née avec les deux). Conséquence : une position rétractée par un re-décodage reste servie ; rien n'interdit un DELETE. Lot : migration de reconstruction (`append_only_rebuild.go`, colonne `decode_pass`, vue « dernière passe entière par match » sur le modèle de `kill_openings_latest`), persister `PersistPass`, inscription à la liste protégée, `-p 1` intégration. Le backfill killsource du 09-09 a écrit par clé : la migration le relit tel quel | Notion « séquence release », item `kill_positions` | Opus, effort moyen (persist, ADR 0026/0030) | `go test -tags=integration -p 1 ./internal/persist/ ./internal/migration/`, `no_art_patterns_test` | [x] 09-09 : `e6f4b16ad`, `43548bae3`, `5a1a9d195` (Opus) ; migration `shared_kill_positions_decode_pass_v1` (rebuild rejouable, `legacy-<match_id>`, vue à deux étages), `PersistPass`, garde-rails déjà enrôlés (justifications corrigées) ; gates unit + intégration `-p 1` + NoART + lint verts ; **revue double** (L1, L6) 0 constat, 5 mutations rougies (`.ai/V7.5/REVUE_LOT17_KILL_POSITIONS_2026-09-09.md`) ; fusion `8102b8639`, poussée. Migration appliquée au prochain boot local. Découvertes : `seed_demo_corpus.go:362` `UPDATE kill_positions` hors garde-rails ; ADR 0026 ne documente pas `decode_pass` |
| 1.R | Revue adversariale UNIQUE sur le diff cumulé 1.1-1.5, une ronde de corrections, fusion, CI | Sonnet contexte frais | — | P0 = 0, P1 = 0 | [x] 09-09 : `.ai/V7.5/REVUE_VAGUE1_2026-09-09.md` — 2 constats recevables (coupure client avalée dans le helper ETag, requalifié P1 ; contrat « jamais vide » faux sur un match sans équipe alliée connue), 0 jeté, corrigés + testés `82d27bd3b` ; ronde 2 non jouée (corrections de 15 lignes, sobriété) ; gates finaux : Go verts, tsc 0, vitest 7 149, `generated.ts` stable ; fusion dans `feat/v75` et push décidés par questionnaire. **CI de vague VERTE** sur `af3ef58b6` (run 34377941308, tous jobs verts, E2E skip) après un premier rouge (`8102b8639`, oracle figé avant E2, refigé) |

### Vague 2 — Équipement « servi ou gâché » (séquentiel, le lot lourd)

| # | Lot | Exécutant | Gate | Statut |
|---|---|---|---|---|
| 2.1 | E0 mesure `equipmentChanges` sur >= 20 artefacts (après recuisson 0.4) ; décision de sortie (rangs non nommés > 15 % ou écart médian > 10 % = deux issues) | Opus, effort moyen | `go test ./internal/analysis/replay/ -run Research -v` + 5 chiffres dans la référence canaux | [x] 09-09 : `393b9a116` + `4d435bbb2` sur `feat/equipement-gachis` (Opus). Mesures sur 64 artefacts : 1 880 changements (75,6 % pris / 24,4 % consommés), émissions manquées 3,6 %, **rangs non nommés 25,2 %** (rangs 10, 19, 22 + 8 artefacts sans `abilityLabels`), slots non rattachés 1,1 %, identité pris = utilisé + lâché + gardé : **écart médian 0 %**, pire cas 20 %. Répulseur : 375 pris, 0 utilisé (P4 confirmée). Seuil « rangs non nommés > 15 % » franchi → **décision utilisateur (questionnaire) : garder les trois issues, afficher les objets sans famille en réserve visible, lot manifeste à part pour nommer 10/19/22** (19 et 22 attendent un relevé Theater de l'utilisateur). Découverte : toutes les familles sauf le mur sont à 1:6-1:12 déployé/lâché — l'anomalie est le 1:1 du mur |
| 2.2 | E1 + E2 (cellule empilée `ValueGrid`, vue match) | Sonnet, effort moyen | gates E1, E2 + greps couleur | [x] 09-09 : `2a89a855d` (E1), `0c2d55e19` (E2) + amendement superviseur (objets sans famille non rendus, `unnamedTaken` conservé) sur `feat/equipement-gachis` ; « gardé » dérivé côté web depuis `equipmentChanges` (pris − utilisé − lâché, famille par racine de libellé bilingue) ; vitest ciblé 999, complet 7 166, match-replay 2 596 après amendement, tsc 0, greps couleur = baseline ; eslint : 9 avertissements PRÉEXISTANTS hors diff (dette, §6). Non fusionné : revue de vague après E3-E6 |
| 2.3 | E3 backend (`UsageSummaryRev` us3 → us4, migration, agrégats, taux de référence, capability) + recuisson des résumés | Opus, effort élevé | `go test -tags=integration -p 1 ./...` obligatoire | [x] 09-09 : `58297a995`, `73be18490`, `3859b256c`, `0e32aa161`, `9583f2426` sur `feat/equipement-gachis` ; us3 → us4, migration `shared_match_usage_players_outcomes_v1` (4 colonnes + vue `_latest` recréée), INSERT-only, 0 entrée d'allowlist anti-ART ; contrat régénéré ; E3.7 `[~]` (déjà gaté par `film.usage_summary`) ; **fusion `da01eba2e`** ; gates superviseur après fusion : Go unitaires verts (persist, migration, replay, sessionusage, service, duckdb, replayartifacts, archlint), tsc forcé 0, vitest match-replay+charts 2 903 verts, intégration `-p 1` (voir journal) ; **E3.11 fait** : serveur arrêté, `backfill-usage-summary --dry-run` puis passe réelle sans `--force` = **64 écrits, 0 déjà à jour, 0 échec** (2 s), serveur redémarré (health 200). **P1 à traiter en E5** : `equipmentUsageLogic.ts` 430 → 582 L sans exemption (règle 5) — à scinder dans le lot qui refond ce modèle |
| 2.4 | E4 Sessions (après D1) | Sonnet, effort bas | gate E4 | [x] 09-09 : `4140d5119`, `c498fb183`, `00664e3ba` sur `feat/equipement-gachis` ; TDD observé (10 + 4 rouges avant code) ; vitest session-detail 146 verts, tsc forcé 0, eslint 0 ; vérifié sur pièces par le superviseur : 0 couleur en dur, `metricDropped` encore consommé par la classification (pas de clé morte) ; **fusion `b369c15c8`**. **P1 consigné** : `usageLogic.ts` 507 → 680 L (dette gelée accrue) — scission confiée à l'extraction E5.1 |
| 2.5 | E5 bloc partagé + Synthèse ; E6 Escouade | Sonnet, effort élevé (Go de E5 : Opus effort bas) | gates E5, E6 | [x] 09-09 (soir) : deux exécutants en parallèle, fichiers disjoints — Go E5.4-E5.7 + E6.1 (Opus, worktree `LevelUp-wt-equipement-e5-go`, branche `feat/equipement-e5-go`) **livré** : `38e67e08b`, `24ec33a61`, `ea39fba80`, `576c1fba5`, `1b19aff59` — `domain.EquipmentUsageBlock` attaché aux réponses Synthèse et Escouade (aucun endpoint dédié), AUCUNE requête DuckDB neuve (les trois lecteurs `SessionUsageRepo` réutilisés, `platform/duckdb` intact), `ResolveScopeFriends` (union, sans ami configuré = aucun ami) à côté de `ResolveTrackedSquad` (intersection), parts en comptes sans pourcentage, FFA intégral = donuts absents ; gates exécutant : go test vert, intégration `-p 1` duckdb/service/api verts, lint 0, contrat régénéré 0 diff, tsc 0 ; vérifié sur pièces par le superviseur (0 `slug ==`, 0 SELECT neuf, 0 fichier > 500 L) ; fusion `4e1f9b5f0`. Découverte à noter : le worktree avait été créé sous `C:Users...` (conversion de chemin MSYS désactivée pendant le `git worktree add`), relocalisé par l'exécutant. et extraction web E5.1-E5.3 + scissions (Sonnet) **livrée** : `a6b2cf631`, `e1b590bc6` — bloc dans `features/_shared/usage/` (11 fichiers, max 430 L), garde-rail `noLocalUsageCopies.guard.test.ts` mordant prouvé par mutation, ratchet cross-feature inchangé (7 <= 7), `equipmentUsageLogic.ts` 582 -> 488 L + `equipmentKeptLogic.ts` 142 L ; vitest 3 083 verts, tsc 0, lint 0 ; fusion `892ac6d5f`. Puis E5.8-E5.13 et E6.2-E6.5 (web, Sonnet) **livrés** : `ea724f3e7`, `f5c957c2d`, `d61d0be1a`, `455fe4cbf`, `7e10fa95f` — variante comptes dans `_shared/usage/` (`usageCountsModel`, `usageEquipmentPartiesModel`, `UsageCountsGrid`, `UsageEquipmentDonutCard`, `EquipmentUsageSection` solo/squad), 29 tests neufs en TDD, suite web complète 7 214 verts, tsc 0, eslint 28/28 inchangé, ratchet cross-feature 7 <= 7, 0 couleur en dur ; fusion `cf7972f3b`. **Découverte bloquante (exécutant, vérifiée sur pièces)** : E6.1 a publié le bloc sur `SquadPageV2Response` (`GET /pages/squad/v2`, consommé par aucune page) alors que l'Escouade lit `POST /pages/teammates` — le web a déclaré le champ à la main sur `TeammatesPageResponse` et la section s'auto-masque. **Lot correctif E6.1bis lancé** (Sonnet, `feat/equipement-e5-go`) : publier sur Teammates, retirer de SquadV2 (0 code mort), régénérer le contrat, aligner `types.ts`  **E6.1bis livré et fusionné `52b49ae9e`** (Sonnet, `03aaca8ab`..`0db15ebb8`) : `equipment_usage` publié sur `TeammatesPageResponse` via `TeammatesService.WithEquipmentUsage` (scope = `filteredMatches` de `GetPage`, amis = coéquipiers sélectionnés), retiré de squad v2 (0 code mort), `buildEquipmentUsageBlock` déplacé dans la feuille `squadagg` (cycle rompu), contrat régénéré, `types.ts` aligné ; gates exécutant verts (go test, intégration `-p 1` service+api, lint 0, tsc 0, vitest squad+_shared 594). Vérifié sur pièces : 0 `slug ==`, 0 `EquipmentUsage` sur squad v2 ; +10/+11 L sur `domain/teammates.go` et `teammates_service.go` déjà > 500 L (glu minimale, consignée) |
| 2.6 | **Artefact interactif d'investigation Theater** (consigne utilisateur 2026-09-09) : pour chaque cas à relever dans le jeu — rangs de palette 10/19/22 (objets sans famille, JAMAIS affichés dans l'UI), bots hors table, vies anonymes de `4f77afc1`, `58864b3c` — une page publiée listant le match (date et heure locales, carte, mode), le timestamp à regarder (horloge film ET match), les joueurs impliqués (gamertags, camp), ce qu'on s'attend à voir, et une case de saisie. Un seul artefact pour tous les cas ouverts | registre (dernière ligne) | superviseur (Artifact) | artefact publié, cas vérifiés sur pièces (date/heure depuis `match_registry`, carte depuis le catalogue) | [x] 09-09 (soir, après E6 web) : artefact publié — https://claude.ai/code/artifact/cf36cd85-9517-497d-adf3-e5441ed4c970 — douze cas : rang 10 (85 prises / 16 matchs : `cde26226`, `f0220a96`, `4f77afc1`), rang 19 (52 / 15 : `0797ce72`, `9ffce8ef`, `4f77afc1`), rang 22 (48 / 14 : `9ffce8ef` où JGtm ramasse lui-même, `bf2a9f05`, `a396aa7f`, `d1dfbc02`), 18 corps hors table de `4f77afc1` (hypothèse : bots 343 entrés en cours), camps de `58864b3c`. Chaque cas : date/heure locales, carte, mode, camps, chrono du match ET position film (formules de `matchClock.ts`), joueurs, quoi regarder, réponse enregistrée dans la base de la page (`releves/<cas>`, relue par `read_db`). Rendu non vérifié de visu (connexion claude.ai requise dans le volet) |
| 2.R | Revue adversariale UNIQUE (persist touché), `make gate-push`, fusion, CI | — | — | [x] 09-09 (soir) : deux relecteurs Sonnet en contexte frais, aveugles l'un à l'autre, sur `git diff af3ef58b6 feat/v75` (100 fichiers, +7 227 / −1 287) — A : L1 anti-ART + L2 multi-titre + L4 données (Go) ; B : L5 front + L6 couverture des tests ; `make gate-push` lancé en parallèle . **Résultat** : relecteur A (Go) 0 constat recevable, 13 conditions tenues ; relecteur B (front/tests) 0 P0, 0 P1, 2 P2 (ordre des segments incohérent entre vue match et bloc partagé → décision S10, réaligné ; regex du garde-rail anti-copie contournable par renommage → consigné §6). `make gate-push` EXIT 0 (0 erreur lint, 30 avertissements préexistants, 0 test rouge). Ronde de corrections : `OUTCOME_ORDER` = utilisé → gardé → lâché + 4 tests réalignés, commentaire du persister (colonne nullable), commentaire obsolète de `SquadSynergiesPage` (page + test) ; gates rejoués : vitest 3 396 verts (5 dossiers), tsc 0, eslint 0 sur les fichiers touchés ; `.ai/V7.5/REVUE_VAGUE2_2026-09-09.md`. Push `feat/v75` = CI de vague |

### Vague 3 — gains moyens

| # | Lot | Exécutant | Gate | Statut |
|---|---|---|---|---|
| 3.1 | Escouade B0-B4 (flèche hors cadre, TDD) | Sonnet, effort moyen | gates B0-B4, parité export vérifiée sur pièce | [x] 10-09 : `0ca4dbfab`..`0abb367da` sur `wt/escouade-hors-cadre` (B0-B4, TDD observé, `make test-web` 7 242 verts, tsc 0, eslint 0 sur le lot, 0 couleur en dur) ; B3.3 couronne VIP `[!]` hors D3 ; **fusion `b859cc65e`** ; B4.2 revue navigateur **faite le 10-09** : Ravin Parasite (`4f77afc1`) à 0:59, zoom x4, chevrons d'équipe à la marge avec « nom · distance » (MiniScotsMin 8 m, Shiloh0209 31 m…) lisibles ; parité export vérifiée sur pièces par l'exécutant, clip non exporté (le code d'export n'a pas changé) |
| 3.2 | Tactique point 21 : pas adaptatif (D6), message d'état vide honnête (« densité insuffisante », pas « pas assez de matchs ») ; **défaut canvas 1 070 × 13 375 px** (découverte du 09-09) ; **recuisson des 7 artefacts aux bornes fausses** (`0a44c6cc`, `30a23d15`, `3923bede`, `4f77afc1`, `81c02726`, `879a4dba`, `a4083bd2` — D11, correctif fusionné `4bd2a7969`, serveur arrêté, un film à la fois, journal `build.go` « échantillons écartés ») puis vérification navigateur du fond de carte d'Isolement (Tactique + rejeu) | Opus, effort bas | tests `analysis/tactical`, Illusion affiche des cellules ; `jq .bounds` de `81c02726` : minZ > 0 et `coversPlayedArea` vrai | [x] 10-09 : `c194c121d`..`eb1cfe41d` sur `feat/tactique-grille` — `ChoisirPas` 0,5 → 1 → 2 m, N = 22 (le plus petit N où deux cellules dépassent le p95 de l'échelle), plancher inchangé, sidecars 0,5 m regroupés (aucune recuisson), `cellule.pas_m` au contrat, état vide « densité insuffisante » FR/EN, cadre du plan borné (rapport exact, largeur max 720 px) ; **deux défauts antérieurs corrigés** : adresse de clic relative à `min_x` et heatmap peinte à l'origine (0,0) — cause probable du plan vide ; gates verts (Go, lint 0, vitest tactical 142, tsc 0) ; **fusion `376894f91`**. **Recuisson des 7 bornes faite** (parc entier, 64/64 en 22 min, pic 553 Mio ; `81c02726` minZ −325,4 → 114,51). Revue navigateur **faite le 10-09** : Illusion affiche des cellules, pied « Grille : 2 m par cellule », canvas 635 × 720 (plus de 13 375 px), état vide absent (des cellules existent) ; rejeu d'Isolement (`81c02726`) : fond de carte servi sous les pions (écarté la veille). Gates post-fusion : Go verts, tsc 0, vitest tactical+match-replay 2 764 verts |
| 3.3 | Fonds de carte étape 0 (mesure) → D10 → étapes 2-5 (TDD, recette export 12 verdicts). **Consigne utilisateur 2026-09-09 : la page Tactique sert aussi les fonds de carte (`features/tactical/queries.ts`, route `.../tactical/{map_id}/background.png`) — après conversion, vérifier EN NAVIGATEUR que le fond s'affiche sur Tactique, en plus du rejeu et de l'export** | Sonnet, effort moyen | gates du plan + contrôle navigateur Tactique (gate bloquant) | [x] 10-09 : étapes 0-5 livrées — `1485835a9` (banc), `a0715c6f2`..`f3a69387d` + `4cd489371` (conversion, commit du superviseur : les fonds sont des assets versionnés) + `1734af2e0` (deux P1 du relecteur : test synthétique du MIME `.webp`/`.png` avec mutation prouvée, doc du schéma), **fusion `fa28c5de8`** (351 fichiers). 109/109 identiques à l'octet, 45,7 → 28,7 Mo (37,2 %), 81 s ; sauvegarde PNG `data/backups/map_backgrounds-png-2026-09-10/` (109 + 109). Gates worktree : Go verts, tsc 0, vitest 3 025 verts. **Vérifié en navigateur le 10-09** : Tactique reçoit `image/webp` + ETag (79 fonds chargés), rejeu d'Isolement servi, **recette export 13/13 OK** (dont `fondDeCarte` 71 %, clip 4,95 Mo en 2,2 s). L'exécutant avait lancé un relecteur de son propre chef (hors consigne) : ses deux P1 ont été retenus et corrigés |
| 3.4 | **Nommer les rangs de palette relevés au Theater** (artefact 2.6) : rang 10 = écran occultant (`shroud_screen`), rang 19 = mur de protection à une utilisation (`wall`, variante), rang 22 = détecteur de menaces à 2-4 utilisations (`sensor`, variante Super Fiesta ; relevé le 10-09, 12/12 relevés reçus) → entrées `[ability_palettes.ranks]` de `config/titles/halo_infinite/mappings/replay_labels.toml` (icônes, FR/EN), garde-rail si le manifeste en a un, puis **recuisson du parc** (les libellés sont figés à la cuisson, `abilityLabelsUsed`) et `backfill-usage-summary --force` (le résumé joint sur la racine du libellé). Relevés complets le 10-09 : débloqué | Sonnet, effort bas ; recuisson superviseur | tests `analysis/replay` ; après recuisson : `prisesSansFamille` du backfill usage 269 → proche de 0 ; vue match : plus d'objets sans famille comptés | [x] 10-09 : manifeste `6a712e85a` (fusion `678f895c5`) ; **recuisson du parc faite** (64/64, 19 min 23 s, pic 527 Mio) ; `backfill-usage-summary --force` : 64 résumés, `prisesSansFamille` **269 → 130** (le reste = artefacts sans table d'équipement, mesure E0 : 8/64, et rang 32 isolé) ; **vérifié en navigateur** : match `9ffce8ef` (Bazaar), onglet Chronologie, « Usages d'équipement » porte une colonne « Capteur de menaces » à 19 objets, aucune ligne « sans famille » |
| 3.R | Revue UNIQUE si 3.3 exécutée ; sinon delivery-checklist seule | — | — | [x] 10-09 : relecteur unique (3.1 + 3.2) : 0 P0, 1 P1 (clic temps/routes au pas grossier sans test → `5af99b55b`, mutation prouvée), 8 conditions tenues ; relecteur du lot WebP (lancé hors consigne par l'exécutant, constats retenus) : 2 P1 corrigés `1734af2e0` ; `make gate-push` EXIT 0 (0 erreur, 30 avertissements préexistants, 0 test rouge) ; registre `.ai/V7.5/REVUE_VAGUE3_2026-09-10.md` ; fusions `849604321`, `678f895c5` ; push = CI de vague — **CI verte le 10-09 sur `737e4fe50`** (CI, Secrets gitleaks, Deploy Pre-Check : 3/3 success) |

### Vague 4 — Découvertes retenues (décision utilisateur du 2026-09-10 : « vague 4 courte », plus deux ajouts arbitrés le même jour)

| # | Lot | Exécutant | Gate | Statut |
|---|---|---|---|---|
| 4.1 | `browserslist` 4.28.2 → 4.28.9 (alerte Dependabot 18, high, outillage de build) | superviseur | `npm ls browserslist` | [x] 10-09 : `80d31ee98` |
| 4.2 | Console de la page de rejeu : neuf clés React dupliquées (identifiants de carte) + un 404 à identifier après redémarrage du serveur | Sonnet, effort bas | vitest + tsc + eslint sur le périmètre, cause corrigée à la source | [x] 10-09 : `0e8d3c725`, fusion `4423a6b93` — cause : le tiroir d'assets (`AssetGrid`, monté en permanence par `AppShell`) reçoit du catalogue de cartes deux entrées de même `map_asset_id` (`ListMapsByTitle` dédoublonne par `name_canonical`, `metadata_repo_assets_list.go:27`) ; dédoublonnage pur et testé côté web (`assetDrawerLogic.ts`, premier gagnant). Gates : vitest 6 + 171, tsc 0, eslint 0. 404 : **non reproduit** après le correctif et le redémarrage (page de rejeu rechargée : 250 ressources, 0 × 404, 0 × 5xx) ; la requête Go garde son `DISTINCT ON (name_canonical)` — deux libellés pour un même asset est un choix de données à trancher, consigné §6 |
| 4.3 | Identité : nommer les corps hors table par le tableau de l'API (bots 343, arrivées en cours — confirmé par le relevé Theater) avec une voix nouvelle déclarée au persist ; **+ publier la table rang → famille dans le document** (bump schéma 51 / us5, racines de libellé supprimées) ; recuisson du parc et re-résumé par le superviseur | Opus, effort moyen | tests replay + goldens, intégration `-p 1` persist/sync/archlint, lint 0 ; après recuisson : `index_hors_table` de `4f77afc1` 18 → 0 attendu, `prisesSansFamille` ≈ 0 | [x] 10-09 : `2fb53db4e`, `62a8f73f3`, `0161e67d3`, **fusion `0db1b21c6`** (36 fichiers, +977). Étape 2bis du registre (`identity_registry_scoreboard.go`) : bot déclaré au tableau → `bid` posé sur la vie ; siège partagé (bot puis humain arrivé en cours) → fenêtre de participation tranche PAR VIE, à cheval on compte ; index que personne ne déclare (index 24 de `4f77afc1`, 8 vies) → `[!]` rien posé, attendu **18 → 8** et non 0 (le tableau n'indexe pas). Voix `tableau_api` déclarée dans le même commit côté `persist` + DDL. Deux défauts corrigés au passage : `BotID` perdu dans `replaybuild.botIdentities` (`roster[].bid` vide sur tout le parc), garde du `bid` contre l'élimination roster. Table rang → famille : `Label.Family`, `equipmentOutcomeStems` supprimée, rangs 8/9 dotés de `powerup_*`, schéma 51 / us5, golden +2 lignes, garde-rail familles étendu, `REFERENCE_CANAUX_EQUIPEMENT` corrigée. Gates : go test, intégration `-p 1`, lint 0, openapi + types web, tsc. **Recuit 64/64 en 22 min (pic 552 Mio, 0 erreur), us5 64 écrits** ; vérifié par l'API : `4f77afc1` `index_hors_table` **18 → 8**, `externe` 10, bots servis `bid(6.0)` / `bid(42.0)`. Découverte (préexistante, consignée) : 8 artefacts sur 64 sans table de capacités (palette « non classée », ≤ 7 lectures), dont ce Grand combat malgré 187 épisodes — le résumé us5 y reste à 130 prises sans famille, mesure de 3.4 inchangée. Découverte consignée : `equipmentKeptLogic.ts` garde `EQUIPMENT_CHANGE_FAMILY_STEMS` (copie web des racines, peut lire `abilityLabels[].family`) |
| 4.4 | Rejeu : flèche hors cadre pour les joueurs EMBARQUÉS (le pion en véhicule n'était pas dessiné hors cadre ; cas le plus fréquent en Grand combat) | Sonnet, effort bas | vitest match-replay, tsc, eslint, parité export | [x] 10-09 : `1d72237df` (8 cas rouges, 7/8 rouges avant code) + `6c2c90e1c`, **fusion `e12770669`**. Une seule flèche par véhicule (jamais une par occupant) : étiquette du conducteur (siège 0) si nommé, sinon « N joueurs · d m » (clé `offscreenGroupMarkerFmt` FR/EN) ; la flèche remplace les noms empilés, ne s'y ajoute pas ; véhicule vide hors cadre = glyphe plaqué seul. Gates worktree : vitest 2 631 verts, tsc 0, eslint 1 avertissement préexistant. Découverte notée : cônes de visée des occupants dessinés à la position bornée (angle inchangé). **Contrôle navigateur fait le 10-09** sur `4f77afc1` (Warthog de BLADERUNNER3141 plaqué au bord avec étiquette, sprite + nom en rentrant dans le cadre ; 0 erreur console) |
| 4.5 | **Artefact de vérification Theater n° 2** (demande utilisateur du 10-09) : lever la réserve `SOUS_RESERVE` du tag répulseur `07104b31` et vérifier la classe de la chute témoin (`1eedd3c8`, Némésis, Theater 02:04, JGtm → EIcRriizz) | superviseur | lectures en base pendant l'arrêt serveur ; artefact si témoin | [x] 10-09 : **pas d'artefact — rien à relever**. Répulseur : **0 ligne** dans `match_kill_events` toutes passes (5 révisions, 979 matchs en killsource v2) ; le « seul kill » du 29/08 n'a pas survécu à la passe v2, la réserve reste posée dans `labels.tsv` (`[!]` sans témoin). Chute : **en base** (124 140 ms, tag `00403594` `DEGAT_GLOBAL` VALIDE, lecture « source-victime », diverges vrai), registre → `hinf_environment` OK, `weapons` de metadata OK ; la perte était dans la SEULE vue match (provenance film non recopiée, `fragdist` n'ouvre équipement/environnement que mesurés au film — verrou Halo 5). **Correctif `85960f48d`, fusion `85ab36cdb`** (`FromDamageSource` sur `BulkWeaponKillRaw`, posé par le lecteur film, recopié par l'assemblage) : test rouge → vert, mutation prouvée, Halo 5 inchangé. **Vérifié par l'API après redémarrage : `1eedd3c8` = sidearm 9 / mêlée 4 / environnement 1, total 14, plus de « non attribué »** |
| 4.R | Revue (persist touché par 4.3 : deux relecteurs, courts), `make gate-push`, CI | — | P0 = 0, P1 = 0 | [x] 10-09 : revues faites — relecteur A (L1 + L4, Opus) 0 P0 / 0 P1 / 3 P2 consignés, 20 conditions tenues ; relecteur B (L6 + L3, Sonnet) 0 constat recevable, 9 conditions tenues, 2 mutations jouées ; registre `.ai/V7.5/REVUE_VAGUE4_2026-09-10.md`. **`make gate-push` EXIT 0** (0 erreur, 30 avertissements eslint préexistants, 15 782 tests, baseline 9 778 couverte) ; **push `737e4fe50..0879f1787`** (24 commits) le 10-09 13:52 = CI de vague 4 (CI, Secrets, Deploy Pre-Check, gate ADR 0021), **CI verte 4/4 le 10-09 sur `0879f1787`** (CI, Secrets, Deploy Pre-Check, gate ADR 0021) |

Consigné sans lot (arbitrage du 10-09 avec impacts écrits au journal) : révision us5 des armes spéciales (P5), hygiène de code (garde-rail, alias, eslint, flake, journal), couronne VIP, socles d'équipement, mesures E0 (investigation bornée si quota), part de la Synthèse, lot hygiène du registre (en tout dernier si quota).

### Après ce plan (non planifié ici, pour mémoire)

P3 → P4 → P5 du paradigme (Opus, orchestration §4) ; audit anti-bombe-RAM (D8) ; registre :
R08 score en direct (demande utilisateur du 20/08, décodeur `ti=6` prêt sans appelant), R09
cadrage étiré sur cartes Forge, R99 gate visuel des fonds (XS côté utilisateur, débloque R06).

### Revues navigateur de la vague 1 (superviseur, 2026-09-09, serveur local relancé après les backfills)

| Écran | Verdict |
|---|---|
| Rejeu Origin `8bc6074f`, point de vue JGtm | **OK** : badge image « Revirement — Tuez un ennemi qui vous a attaqué en premier » rendu à 3:49 sur la piste (titre + description) ; fond de carte Origin affiché sous le rejeu (route `replay/background.png` via le helper ETag) |
| Escouade, session du 27 août, composition stricte cochée (3 coéquipiers sélectionnés) | **OK** : un seul nombre partout — rail « 7 matchs », résultats 7 (6 V / 1 D). Le cas « 4 sur 7 » n'apparaît pas avec cette sélection (rien n'est écarté) ; couvert par les tests A3, filtres de l'utilisateur non modifiés |
| Tactique, liste des cartes | **OK** : fonds de carte servis sur toutes les vignettes (route `tactical/{map}/background.png`) |
| Tactique, plan d'Illusion | Couverture **26,6 % → 67,9 %** après le backfill des positions, 38 matchs retenus ; plan toujours vide (cause A, lot 3.2). **Découverte** : canvas 1 070 × 13 375 px non peint (registre) — à traiter dans le lot 3.2 |
| « Retours UI » du 09-09 (tuiles, catalogue) | Non revu séparément : les trois pages ci-dessus rendent sans erreur console visible ; à confirmer par l'utilisateur à l'usage |

## 5. Registre des reports — sort décidé (tri Opus du 09-09, vérifié sur pièces par le superviseur)

**Clôtures administratives (le texte du registre ou le dépôt prouvent la résolution) :**
L164 (`originMs` des objectifs : corrigé le 2026-08-18, `objectives.go:59-66`) ; L117 (le
rejeu lit `map_backgrounds/` : handlers `replay.go`/`tactical.go`) ; L589 (question `isole`
présente dans `features/tactical/i18n.ts:65`) ; L397 (`TestNoPlayerIndexInFilmScope` VERT au
09-09) ; L524 clause 1 (`2b45b0dad` est sur `feat/v75`) ; L18 (doublon de L10 clos) ; L593,
L601 (fermées vague 1) ; L434 (posé le 25/08, réfute L61) ; L482/L484 (supersédées par L492) ;
L480 (conditions atteintes) ; L254, L213, L368 (absorbées par L602) ; L588 (CI verte 09-09).
**Après fusion E2** : L14 (pont par manche : P2 « élimination sur le roster » ramène l'écart
K/D/A de `51ebbc0f` et `d9781168` à 0 ; résidu `64e8adfa`/`c75f33b8` = lien statborg, lot
décodeur suivant, à écrire en résidu) ; L567/569/584/602 (recuisson) ; L133 (backfill).

**Reportés, condition inchangée (gain faible ou dépendance réelle)** : RE image-clé et grammaire
type-2 (borne R7-e), portage Oddball (5 négatifs), Total Control/BTB (ratifié hors v7.5),
précision par arme (NO-GO documenté), Forge industrialisation, 2e VPS et ouvrier distant
(post-tag), sons en attente d'assets/écoute, dette de taille de fichiers (gelée par baseline),
`killpos` (échéance 2026-11-08), migration boot ADR 0023 (échéance **2026-10-01**, à
surveiller), TS 7.

**Petits items regroupables en un lot hygiène (XS chacun, gain nul, à traiter en tout dernier
si quota)** : L29 (`replay-diff` spans/n), L302 (`World.HeldWeapon` mort), L514
(`buildFormTab` mort), L62 (littéral `film_chunks`), L66 (`match_player_positions` à droper),
L63 (`loadGameVariant` erreur avalée), L595 (3e copie clé de roster bot), L575/L437
(`useReplaySound`), L603 (seuil temporel `LoaderRunsInParallel`), L590 (`time.Sleep`), L577
(bench hors dépôt), L473 (`EXPECTED_REPLAY_SCHEMA_VERSION`), L11 (doc inversée
`vehicle_rides.go`), L542 (`docs/WEAPONS.md`).

## 6. Découvertes de cette planification (consignées, non traitées sauf mention)

- 2026-09-10 (4.2) ; `apps/go-api/internal/platform/duckdb/metadata_repo_assets_list.go:27` ; `ListMapsByTitle` dédoublonne par `name_canonical`, donc deux libellés partageant un `map_asset_id` sortent en deux assets de même identifiant (les clés React dupliquées du tiroir). Le web dédoublonne désormais par identifiant ; côté données, choisir quel libellé porte l'asset (ou dédoublonner par `map_asset_id` dans la requête) reste à trancher. P2.
- 2026-09-10 (revue navigateur) ; page rejeu, console : neuf avertissements React « two children with the same key » portant des identifiants de carte (liste des cartes du sélecteur, doublons d'`map_id`) et un `404` sur une ressource non identifiée. Préexistants (aucun lot de la vague ne touche ce sélecteur). P2, non traités.
- 2026-09-10 (revue navigateur) ; Tactique Illusion, plan à 2 m : les cellules lisibles se regroupent en bas du plan (deux zones rouge/bleu). Plausible (réapparitions), mais à confirmer d'un œil utilisateur face au repère corrigé par 3.2 (`min_x`/`min_y`).
- 2026-09-09 (revue 2.R, relecteur B) ; `apps/web/src/features/_shared/usage/noLocalUsageCopies.guard.test.ts:41-49` ; les regex à frontière de mot (`function buildGaugeRow`) laissent passer une copie renommée d'un suffixe (`buildGaugeRowLocal`). Le cas historique (copie à l'identique) reste couvert. P2, non traité.
- 2026-09-09 (revue 2.R, relecteur A, non recevable) ; `internal/persist/usage_summary_persister.go:170` ; le commentaire dit « colonne NOT NULL » alors que la migration pose `DEFAULT '{}'` sans NOT NULL (DuckDB refuse les contraintes sur ADD COLUMN). Doc inexacte, corrigée dans la ronde de corrections de la vague.
- 2026-09-09 (E6.1bis) ; `apps/web/src/features/squad/SquadSynergiesPage.tsx` (commentaire et test) ; décrit l'ancien écart de contrat, obsolète depuis que Teammates publie le bloc. Corrigé dans la ronde de corrections de la vague (doc inversée, anti-pattern 9).
- 2026-09-09 (E3, rapport de l'exécutant) ; `internal/analysis/replay/usage_summary_outcomes.go` ; le document cuit ne publie pas la table rang → famille du manifeste, d'où deux reconstructions par racine de libellé (plafond de la règle n°6). La publier fermerait le sujet mais c'est une recuisson du parc, pas une re-projection : à décider en vague 3 avec la recuisson des 7 bornes (lot 3.2) si le quota le permet, sinon reporté.
- 2026-09-09 (E3) ; contrat de session : `deployed_*` et `equipment_*` coexistent. Le lot qui retire `deployed_*` de l'écran (E4 ou E5) doit le retirer du contrat dans le même lot ; E4 reçoit la consigne de LISTER les consommateurs restants, E5 tranche.
- 2026-09-09 (soir) ; `.ai/PLAN_MASTER_2026-09-09.md` §0 ; l'inventaire initial des pièces a manqué deux branches sans worktree `LevelUp-wt-*` nommé dans les plans lus : `wt/bornes-aberrantes` (référencée seulement par le plan de la vague C, tâche hors lot) et `wt/orchestration-0907`. Leçon : l'inventaire de reprise part de `git branch --no-merged feat/v75` + `git worktree list`, pas des plans. Traité par D11.
- 2026-09-09 ; `handlers/helpers.go:113` ; cinq handlers gèrent déjà `If-None-Match` — le plan
  WebP l'ignorait. Traité par S5 (amendement de D9), pas reporté.
- 2026-09-09 ; `apps/web/src/features/{ascension,explorer,palmares,squad/charts}` ; quatre
  heatmaps ECharts construites à la main hors `Heatmap2DChart` en plus de la Synthèse. S6.
- 2026-09-09 ; worktree partagé ; 116 fichiers de trois chantiers terminés non commités, dont
  `SquadIsolementNuageCard.tsx`, `squad/i18n.ts`, `SessionUsageForms.tsx` — collision directe
  avec C3, A3, E4. D1.
- 2026-09-09 ; `main` ; 13 commits absents de `feat/v75`. D3.
- 2026-09-09 ; **P0 TRAITÉ IMMÉDIATEMENT** ; `persist/lives_persister.go` refusait les quatre voies
  de nommage du registre d'identité (E2) : la passe `backfill-killsource` locale a échoué sur
  les 736 films (faits d'isolement non réécrits ; en prod, chaque nouveau match aurait été
  refusé au sync). Correctif `fix/match-lives-named-by-e2` fusionné (`bc61b6636`) : constantes,
  validateur, garde-rail `no_life_cause_divergence_test` étendu à tout le paquet replay et à la
  forme `const X = "..."`. Passe films rejouée après la chaîne (`suite_killsource.sh`). Point
  prod ajouté au registre (dernière ligne) : `backfill-killsource --films-only` au déploiement.
- 2026-09-09 ; CI de `8102b8639` rouge sur UN test : `wire/TestOuvrierReel_ConstruitEtLivre`
  (intégration CGO) — oracle figé avant E2 (« 1 vie anonyme », « 0 mort à l'API ⇔ absent du
  roster ») ; le lien direct nomme désormais `2535458702376288` (0 mort). Gain, pas régression :
  oracle refigé (`fix/wire-ouvrier-oracle-e2`), assertion réduite à « mort au moins une fois ⇒
  nommé ». Leçon : le lot E2 n'avait pas joué `-tags=integration ./...` complet (wire) ; le
  gate CI par vague l'a rattrapé.
- 2026-09-09 ; GitHub signale 1 vulnérabilité « high » (dependabot n°18) sur la branche par
  défaut — hors périmètre, à regarder par l'utilisateur.
- 2026-09-09 ; bots réels anonymes après E2 (index hors table) ; consigne utilisateur : croiser
  d'abord les bornes d'arrivée/départ des bots publiées par l'API (`bid(N.0)`,
  `present_at_beginning` / `left_in_progress` / `last_leave_time`) avec les corps anonymes du
  film ; sinon le nom est dans le film, pas encore localisé. Entrée au registre des reports
  (dernière ligne), lot dédié à ouvrir après la vague 1.
- 2026-09-09 ; `.ai/thought_log.md` ; 102 417 lignes / 8,1 Mo, conforme à la rotation
  (Q2+Q3) mais lourd pour tout `git diff` : rotation Q2 → archive à faire au 2026-10-01.


### Découvertes de la vague 4 (revue 4.R, 2026-09-10 — consignées, non traitées)

- P2 `cmd/levelup/cmd_backfill_usage_summary.go:285-292` : la reprise `(rev, schema)` ne refuse
  pas un artefact au schéma < 51 après le bump us5 ; dans l'ordre inverse (re-résumé avant
  recuisson) le bilan d'équipement sortirait vide et marqué à jour. Ordre supervisé respecté ici.
- P2 `internal/analysis/replay/identity_registry_creation.go:374-378` : compteur de l'alarme
  « index lu mais absent de la table » soustrait une population de lecture directe d'un résidu
  post-tableau → alarme muette ou sous-évaluée sur les films à sièges de bots nommés.
- P2 (préexistant) `internal/sync/killcollector/positions.go:484-489` : le collecteur de sync ne
  passe ni `Bots` ni `Participants` au registre ; `match_lives` et l'artefact divergent sur un
  siège partagé bot → humain, et la voix `tableau_api` n'a aucun producteur vers `match_lives`.
- P2 (préexistant) : 8 artefacts sur 64 sans table de capacités (palette « non classée »,
  ≤ 7 lectures) dont `4f77afc1` malgré 187 épisodes de capacité — source des 130 « prises sans
  famille » du résumé us5 ; à instruire avec le seuil de classement de la palette.
- P3 `apps/web/src/features/match-replay/model/equipmentKeptLogic.ts` : copie web des racines
  de libellé (`EQUIPMENT_CHANGE_FAMILY_STEMS`) désormais remplaçable par `abilityLabels[].family`.
- P3 rejeu : cônes de visée des occupants d'un véhicule hors cadre dessinés à la position
  bornée (angle inchangé) — pas de défaut constaté.
## 7. Reprise de session

Lire ce plan, puis `.ai/thought_log.md` (entrées `[2026-09-09]`), puis le plan source du lot en
cours ; `git worktree list` ; reprendre à la première case non statuée de la vague en cours.
Les agents commitent sur leur branche `feat/<slug>` ; le superviseur vérifie SUR PIÈCES avant
de cocher ; fusion dans `feat/v75` seulement avec l'accord de l'utilisateur.

## Journal

- **2026-09-09** — Plan créé. Sept documents lus, état vérifié sur pièces (branches,
  worktrees, statuts git, tests). Registre trié par un agent Opus (162 groupes ouverts), huit
  lignes prioritaires contre-vérifiées. Gate corpus E2 lancé ; lots 1.1, 1.2, 1.3, 1.4 lancés
  en parallèle (Sonnet, worktrees dédiés). D1-D9 soumises à l'utilisateur.
- **2026-09-09 (soir)** — Question de l'utilisateur : « les branches `wt/bornes-aberrantes` et
  `wt/orchestration-0907` sont-elles reprises ? » Non, ni l'une ni l'autre. Vérifié sur pièces :
  le correctif des bornes n'était dans aucun autre chemin, et l'artefact `81c02726` recuit à 17:37
  au schéma 50 porte toujours `minZ = -325.4` — la décision S7 était fausse sur les 7 bornes.
  D11 : fusion du code (`4bd2a7969`, conflit `thought_log` par union, `build.go` auto-fusionné
  avec E2) et du journal d'orchestration (`2cc39f89b`, 3 docs par union, 0 marqueur résiduel,
  lignes E2/E2-bis et D12-D14 toutes présentes) ; recuisson des 7 artefacts rattachée au lot 3.2.
  Gates : `go build ./...` + tests `analysis/replay` et `replaydiff` (résultat au journal
  thought_log). Pas de push : la CI reste une par vague.
- **2026-09-09 (soir, suite)** — E3 rendu (Opus, 70 min, 5 commits) et vérifié sur pièces :
  0 comparaison de slug, 0 allowlist agrandie, `fmt.Printf` limité à la sortie CLI de
  `cmd/levelup`. Fusion `da01eba2e` (conflit `thought_log` par union). L'échec d'intégration
  signalé par l'exécutant (`TestOuvrierReel_ConstruitEtLivre`) est l'oracle refigé en
  `af3ef58b6` sur `feat/v75`, absent de son point de branche — vérifié par le gate `-p 1` après
  fusion. E3.11 joué (64 résumés us4). E4 lancé (Sonnet) dans le même worktree, avancé au niveau
  de `feat/v75` en avance rapide.
- **2026-09-09 (soir, E4)** — E4 rendu (Sonnet, 26 min) et vérifié sur pièces, fusion
  `b369c15c8`. Décision S9 (`deployed_*` reste au contrat). E5 scindé en deux exécutants
  parallèles sur fichiers disjoints (Go dans un worktree neuf, extraction web dans le worktree
  du chantier), les deux P1 de taille de fichier confiés à l'extraction.
- **2026-09-09 (soir, E5)** — Extraction web fusionnée (`892ac6d5f`) puis Go E5/E6.1 fusionné
  (`4e1f9b5f0`), les deux vérifiés sur pièces. Lancement du dernier exécutant de la vague : web
  Synthèse (E5.8-E5.13) + Escouade (E6.2-E6.5), Sonnet, sur `feat/equipement-gachis` avancée en
  avance rapide. Reste ensuite : 2.6 artefact Theater, 2.R revue double, `make gate-push`, CI.
- **2026-09-09 (soir, E5/E6 web)** — Dernier lot web de la vague fusionné (`cf7972f3b`). Le
  bloc Escouade est monté mais aveugle : le Go l'a publié sur la mauvaise réponse (squad v2,
  legacy) — vérifié sur pièces (`squad/queries.ts:30`). Lot correctif E6.1bis lancé au lieu de
  laisser une donnée morte et une page vide. Reste ensuite : 2.6 artefact Theater, 2.R.
- **2026-09-09 (soir, 2.6)** — Artefact d'investigation Theater publié (https://claude.ai/code/artifact/cf36cd85-9517-497d-adf3-e5441ed4c970), douze cas
  alimentés par les artefacts locaux (rangs, corps hors table, camps) et l'API authentifiée
  (dates, cartes, modes, camps via le volet navigateur). Les réponses arrivent dans la base de
  la page : à relire au début de la vague 3 avant de nommer les familles. En vol : E6.1bis.
- **2026-09-10 (00:05)** — **Vague 2 close.** CI de `feat/v75` VERTE sur `1cfbf9cea` (CI,
  Deploy Pre-Check, ADR 0021 Gate, Secrets). Bilan : E0-E6 livrés et fusionnés, E6.1bis
  (contrat Escouade) attrapé avant la CI, revue double 0 P0 / 0 P1 / 2 P2, `make gate-push`
  vert, 64 résumés d'usage recuits au us4, artefact d'investigation Theater publié. Vague 3
  ouverte : 3.1 et 3.2 indépendants (escouade / tactique), 3.3 après mesure.
- **2026-09-10 (matin)** — Sept relevés Theater reçus sur douze (`read_db`, collection
  `releves`) : rang 10 = écran occultant, rang 19 = mur de protection à une utilisation, corps
  hors table = bots 343 et arrivées en cours, camps de `58864b3c` bons (résidu clos). Lot 3.4
  créé (nommage au manifeste + recuisson + re-résumé), en attente des cinq relevés restants.
- **2026-09-10 (matin, Notion)** — Décision utilisateur : **la release ne rejoue aucun backfill
  en prod ; les bases locales et les replays sont copiés sur le VPS (sauvegarde puis
  écrasement)**. En conséquence, dix cases de la séquence Notion « à dérouler à la release »
  sont cochées avec la mention « fait en local » : cache appauvri, médailles du kill feed,
  killsource, T0 (deux passes), `kill_positions` (migration 1.7), distance des kills et
  `kill_openings` (passe films du 09/09), recuisson au schéma 50 (rejouée le 10/09 avec les
  bornes), résumé d'usage us4, Assaut (0 film local). Restent à l'utilisateur : jeton, curl de
  pré-vol, `systemctl`, `LEVELUP_REPLAY_PUBLIC`, vérifications, copie des bases, nettoyage des
  worktrees, déplacement du décodeur (post-merge). Le surveillant de la chaîne du 09/09 (tail
  persistant) a été arrêté ce matin.
- **2026-09-10 (fin de matinée)** — 3.3 clos : conversion WebP commitée et fusionnée, recette
  export 13/13 en navigateur, Tactique et rejeu vérifiés. Revue unique 3.R lancée (3.1 + 3.2)
  et `make gate-push` en parallèle ; ensuite la CI de vague, puis 3.4 quand les cinq relevés
  arrivent.
- **2026-09-10 (midi)** — Vague 3 : revue 3.R close (3 P1 corrigés avec tests et mutations
  prouvées), gate-push vert, douze relevés Theater reçus et les trois rangs nommés au manifeste.
  Push de `feat/v75` = CI de vague 3 ; recuisson du parc et re-résumé d'usage à suivre (3.4).
- **2026-09-10 (après-midi)** — Arbitrage des découvertes validé par l'utilisateur : us5 armes
  spéciales en P5, table rang → famille dans 4.3, hygiène de code sans lot, VIP et socles non
  traités (la couronne reste dessinée sur le pion ; seul le VIP hors cadre n'a pas sa couronne à
  la marge), investigation E0 bornée si quota, part de la Synthèse non, lot hygiène du registre
  en tout dernier si quota. Kills hors arme : lot déjà livré (`2b45b0dad`, bobines par énergie,
  chute et environnement en une classe, répulseur sous réserve) — lot 4.5 créé pour lever la
  réserve du répulseur par relevé Theater.
- **2026-09-10 (après-midi, suite)** — CI de la vague 3 verte sur `737e4fe50` (trois ateliers).
  Vague 3 close de bout en bout. Vague 4 : 4.1 et 4.2 fusionnés, 4.3 et 4.4 en vol, 4.5 en
  attente de l'arrêt serveur.
- **2026-09-10 (après-midi, 4.4)** — Contrôle navigateur du véhicule hors cadre sur `4f77afc1`
  (Capture du drapeau, Ravin Parasite, 24 juil. 2026, 97 pistes véhicules), zoom 3x à 0:39-1:59 :
  le Warthog occupé par BLADERUNNER3141 (siège 0, film 29 s → 81 s) est plaqué au bord haut
  avec son étiquette quand il sort du cadre ; trois déplacements de vue vers le haut le font
  rentrer : sprite rouge + nom empilé, plus de flèche. Les autres flèches de bord (joueurs à
  pied) gardent leur « nom · distance m ». Console : 0 erreur nouvelle (le seul 404 est
  `/replay/callouts`, carte sans annonces, déjà connu). Vitest du lot rejoué dans le worktree
  partagé après fusion : 75/75 verts sur `vehiclesPaint` + `replayMarkers`.
- **2026-09-10 (après-midi, 4.3 à 4.5)** — 4.3 fusionné (`0db1b21c6`), recuisson 51 du parc
  64/64 en 22 min, us5 re-résumé ; `4f77afc1` hors table 18 → 8 (le tableau de l'API nomme 10
  vies, l'index que personne ne déclare reste non résolu par décision). 4.5 sans artefact :
  répulseur sans aucune ligne en base toutes passes (réserve maintenue), chute témoin en base et
  perdue dans la seule vue match — provenance film recopiée (`85ab36cdb`), vérifié par l'API.
  Revue 4.R : relecteur tests 0 constat recevable, 9 conditions tenues ; relecteur données en cours.
- **2026-09-10 (fin d'après-midi)** — Vague 4 : revue 4.R close (0 P0, 0 P1, 3 P2 consignés),
  `make gate-push` vert, push de `feat/v75` (`0879f1787`) = CI de vague 4, quatre ateliers en
  cours. Reste au plan après la CI : les points « consignés seulement » (us5 armes spéciales en
  P5, hygiène, VIP, socles, E0 bornée si quota, hygiène du registre en dernier si quota).
- **2026-09-10 (soir)** — CI de la vague 4 verte sur `0879f1787` (quatre ateliers). Vague 4 close.
  Réparation du plan maître (`f2a394c8c`) après duplication par une insertion sed sans adresse dans
  deux commits poussés. Restent au plan les seuls points « consignés » et « si quota » : us5 armes
  spéciales (P5), hygiène de code, investigation E0 bornée, hygiène du registre en dernier.
