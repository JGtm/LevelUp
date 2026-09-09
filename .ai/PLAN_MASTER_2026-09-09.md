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
| 0.1 | D3 : `git merge main` dans `feat/v75`, gates Go+web, push | superviseur, accord D3 | `make go-api-test`, `make check-types`, `make test-web` | [ ] |
| 0.2 | D2 : fusion `wt/lot-court` | superviseur, accord D2 | vitest des fichiers touchés, typecheck | [ ] |
| 0.3 | D1 : passe navigateur « retours UI » puis commit des 116 fichiers sur `feat/retours-ui-0909`, fusion | superviseur, accord D1 | `make check-types`, `make test-web`, `make generate-types` sans diff | [ ] |
| 0.4 | E2 : commit du diff `replaydiff` (polarité par-slot → par-xuid), `make replay-corpus-gate --reference=base`, verdict ligne par ligne (une ligne attendue : `flagCarries.homeByObject` sur `3372e7eb`), fusion (D4), `levelup backfill-killsource`, recuisson du parc (`backfill-replay --only-existing`, un film à la fois), puis `swap.sh` sur `8bc6074f d8b13ec2 a4083bd2 bf2a9f05` → `malplaces = 0` | superviseur | gate exit 0 ; `swap.sh` = 0 ; `jq .coverage.vehicles` présent sur les 15 artefacts schéma 38 | [~] en cours 09-09 : gate joué (9 lignes de perte sur 4 témoins, toutes instruites et acceptées au registre — dont `fb1a1a72` : propriétaire du slot 625 corrigé, bornes identiques) ; polarité `replaydiff` commitée `43320cd6f` ; **fusion faite `4dec9bd65`** ; `backfill-killsource` lancé détaché à 16:07 (737 films, 1 967 matchs en crédit ; serveur de dev arrêté pour la passe, journal scratchpad `backfill_killsource.log`). Chaîne de backfills terminée 17:59 (sauvegarde DB, médailles 415 matchs, T0, cache appauvri, **recuisson 64 artefacts schéma 50 en 23 min, pic 539 Mio, `coverage.vehicles` 64/64**, T0 film, résumé d'usage 64, Assaut 0 film au parc) ; **P0 découvert** (voies du registre refusées par `match_lives`, corrigé `bc61b6636`) → passe films de killsource rejouée à 18:00 avec le binaire corrigé ; **`swap.sh` = 0 sur les 4 témoins et 37/38 artefacts** (résidu `58864b3c` au registre). Reste : fin de killsource v2, redémarrage du serveur, revues navigateur |
| 0.5 | Clôtures administratives du registre (§5) + annotation « vague B abandonnée » dans son plan | superviseur | diff docs seul | [ ] |

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
| 2.5 | E5 bloc partagé + Synthèse ; E6 Escouade | Sonnet, effort élevé (Go de E5 : Opus effort bas) | gates E5, E6 | [ ] en cours 09-09 (soir) : deux exécutants en parallèle, fichiers disjoints — Go E5.4-E5.7 + E6.1 (Opus, worktree `LevelUp-wt-equipement-e5-go`, branche `feat/equipement-e5-go` depuis `b369c15c8`) et extraction web E5.1-E5.3 + scissions (Sonnet) **livrée** : `a6b2cf631`, `e1b590bc6` — bloc dans `features/_shared/usage/` (11 fichiers, max 430 L), garde-rail `noLocalUsageCopies.guard.test.ts` mordant prouvé par mutation, ratchet cross-feature inchangé (7 <= 7), `equipmentUsageLogic.ts` 582 -> 488 L + `equipmentKeptLogic.ts` 142 L ; vitest 3 083 verts, tsc 0, lint 0 ; fusion `892ac6d5f`. Puis E5.8-E5.13 et E6.2-E6.5 (web) après fusion des deux |
| 2.6 | **Artefact interactif d'investigation Theater** (consigne utilisateur 2026-09-09) : pour chaque cas à relever dans le jeu — rangs de palette 10/19/22 (objets sans famille, JAMAIS affichés dans l'UI), bots hors table, vies anonymes de `4f77afc1`, `58864b3c` — une page publiée listant le match (date et heure locales, carte, mode), le timestamp à regarder (horloge film ET match), les joueurs impliqués (gamertags, camp), ce qu'on s'attend à voir, et une case de saisie. Un seul artefact pour tous les cas ouverts | registre (dernière ligne) | superviseur (Artifact) | artefact publié, cas vérifiés sur pièces (date/heure depuis `match_registry`, carte depuis le catalogue) | [ ] après E6 |
| 2.R | Revue adversariale UNIQUE (persist touché), `make gate-push`, fusion, CI | — | — | [ ] |

### Vague 3 — gains moyens

| # | Lot | Exécutant | Gate | Statut |
|---|---|---|---|---|
| 3.1 | Escouade B0-B4 (flèche hors cadre, TDD) | Sonnet, effort moyen | gates B0-B4, parité export vérifiée sur pièce | [ ] |
| 3.2 | Tactique point 21 : pas adaptatif (D6), message d'état vide honnête (« densité insuffisante », pas « pas assez de matchs ») ; **défaut canvas 1 070 × 13 375 px** (découverte du 09-09) ; **recuisson des 7 artefacts aux bornes fausses** (`0a44c6cc`, `30a23d15`, `3923bede`, `4f77afc1`, `81c02726`, `879a4dba`, `a4083bd2` — D11, correctif fusionné `4bd2a7969`, serveur arrêté, un film à la fois, journal `build.go` « échantillons écartés ») puis vérification navigateur du fond de carte d'Isolement (Tactique + rejeu) | Opus, effort bas | tests `analysis/tactical`, Illusion affiche des cellules ; `jq .bounds` de `81c02726` : minZ > 0 et `coversPlayedArea` vrai | [ ] |
| 3.3 | Fonds de carte étape 0 (mesure) → D10 → étapes 2-5 (TDD, recette export 12 verdicts). **Consigne utilisateur 2026-09-09 : la page Tactique sert aussi les fonds de carte (`features/tactical/queries.ts`, route `.../tactical/{map_id}/background.png`) — après conversion, vérifier EN NAVIGATEUR que le fond s'affiche sur Tactique, en plus du rejeu et de l'export** | Sonnet, effort moyen | gates du plan + contrôle navigateur Tactique (gate bloquant) | [ ] |
| 3.R | Revue UNIQUE si 3.3 exécutée ; sinon delivery-checklist seule | — | — | [ ] |

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
