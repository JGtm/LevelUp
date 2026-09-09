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
| Registre : audit anti-bombe-RAM (L486/L487) | P0 prod, **bloquant release déclaré** | nul (sûreté) | L | **Hors master plan** : case de la séquence de release, à lancer au moment R9 (Opus). Non oublié |
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
| S7 | La recuisson du parc post-E2 remplace la « tâche hors lot » de la vague C (15 artefacts schéma 38 + 7 bornes fausses) et les lignes L567/569/584/602 du registre |
| S8 | `backfill-killsource` post-E2 (révision d'isolement bumpée) est la MÊME opération que le report L133 (arme du kill 0-5 % avril-juillet) : une passe, deux clôtures |

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
| D8 | Audit anti-bombe-RAM (L486/487) : le lancer à la séquence de release, hors ce plan | release | Oui, Opus, à R9 |
| D9 | Seuil d'activation du rejeu public (registre L120, 88 %) : re-statuer maintenant ? | — | Une ligne de décision, après recuisson du parc |

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
| 0.4 | E2 : commit du diff `replaydiff` (polarité par-slot → par-xuid), `make replay-corpus-gate --reference=base`, verdict ligne par ligne (une ligne attendue : `flagCarries.homeByObject` sur `3372e7eb`), fusion (D4), `levelup backfill-killsource`, recuisson du parc (`backfill-replay --only-existing`, un film à la fois), puis `swap.sh` sur `8bc6074f d8b13ec2 a4083bd2 bf2a9f05` → `malplaces = 0` | superviseur | gate exit 0 ; `swap.sh` = 0 ; `jq .coverage.vehicles` présent sur les 15 artefacts schéma 38 | [ ] gate lancé le 09-09 |
| 0.5 | Clôtures administratives du registre (§5) + annotation « vague B abandonnée » dans son plan | superviseur | diff docs seul | [ ] |

### Vague 1 — P1 visibles, effort modéré (parallèle : 4 lots, domaines disjoints)

| # | Lot | Source | Exécutant | Worktree / branche | Gate | Statut |
|---|---|---|---|---|---|---|
| 1.1 | Escouade A1 + A2 (contrat Go de l'écart + source unique web), TDD ; **A3 attend D1** (touche `squad/i18n.ts` modifié dans le worktree partagé) | plan escouade §3 | Sonnet, effort élevé | `LevelUp-wt-escouade-hors-cadre` / `wt/escouade-hors-cadre` (existant, A0 à commiter d'abord) | gates A1, A2 du plan | [ ] lancé 09-09 |
| 1.2 | C4 : médailles de la frise en images, piste élargie, infobulle titre+description | plan vague C, C4 | Sonnet, effort moyen | `LevelUp-wt-frise-medailles` / `feat/frise-medailles-images` | gate C4 + de visu Origin `8bc6074f` 3:49 | [ ] lancé 09-09 |
| 1.3 | Fonds de carte étape 1 : ETag/304 centralisé (S5), TDD, garde-rail grep | plan WebP étape 1 | Sonnet, effort bas | `LevelUp-wt-fonds-etag` / `feat/fonds-carte-etag` | `go test ./internal/api/handlers/`, `go build ./...` | [ ] lancé 09-09 |
| 1.4 | C1 : `Heatmap2DChart` padding + absence hachurée + plafond ; C2 : garde-rail grep (allowlist S6) — **migration de `SynthesisHeatmapChart` attend D1** (9 fichiers de `synthesis/` modifiés dans le worktree partagé) | plan vague C, C1-C2 | Sonnet, effort moyen | `LevelUp-wt-formes` / `feat/formes-maquettes` | gates C1, C2 | [ ] lancé 09-09 |
| 1.5 | C3 nuage d'isolement + A3 (lisibilité de l'écart) + C2 migration Synthèse | plans C et escouade | Sonnet | mêmes worktrees, rebasés sur v75 après D1 | gates C3, A3 | [ ] après D1 |
| 1.R | Revue adversariale UNIQUE sur le diff cumulé 1.1-1.5, une ronde de corrections, fusion, CI | Sonnet contexte frais | — | P0 = 0, P1 = 0 | [ ] |

### Vague 2 — Équipement « servi ou gâché » (séquentiel, le lot lourd)

| # | Lot | Exécutant | Gate | Statut |
|---|---|---|---|---|
| 2.1 | E0 mesure `equipmentChanges` sur >= 20 artefacts (après recuisson 0.4) ; décision de sortie (rangs non nommés > 15 % ou écart médian > 10 % = deux issues) | Opus, effort moyen | `go test ./internal/analysis/replay/ -run Research -v` + 5 chiffres dans la référence canaux | [ ] |
| 2.2 | E1 + E2 (cellule empilée `ValueGrid`, vue match) | Sonnet, effort moyen | gates E1, E2 + greps couleur | [ ] |
| 2.3 | E3 backend (`UsageSummaryRev` us3 → us4, migration, agrégats, taux de référence, capability) + recuisson des résumés | Opus, effort élevé | `go test -tags=integration -p 1 ./...` obligatoire | [ ] |
| 2.4 | E4 Sessions (après D1) | Sonnet, effort bas | gate E4 | [ ] |
| 2.5 | E5 bloc partagé + Synthèse ; E6 Escouade | Sonnet, effort élevé (Go de E5 : Opus effort bas) | gates E5, E6 | [ ] |
| 2.R | Revue adversariale UNIQUE (persist touché), `make gate-push`, fusion, CI | — | — | [ ] |

### Vague 3 — gains moyens

| # | Lot | Exécutant | Gate | Statut |
|---|---|---|---|---|
| 3.1 | Escouade B0-B4 (flèche hors cadre, TDD) | Sonnet, effort moyen | gates B0-B4, parité export vérifiée sur pièce | [ ] |
| 3.2 | Tactique point 21 : pas adaptatif (D6), message d'état vide honnête (« densité insuffisante », pas « pas assez de matchs ») | Opus, effort bas | tests `analysis/tactical`, Illusion affiche des cellules | [ ] |
| 3.3 | Fonds de carte étape 0 (mesure) → D10 → étapes 2-5 (TDD, recette export 12 verdicts) | Sonnet, effort moyen | gates du plan | [ ] |
| 3.R | Revue UNIQUE si 3.3 exécutée ; sinon delivery-checklist seule | — | — | [ ] |

### Après ce plan (non planifié ici, pour mémoire)

P3 → P4 → P5 du paradigme (Opus, orchestration §4) ; audit anti-bombe-RAM (D8) ; registre :
R08 score en direct (demande utilisateur du 20/08, décodeur `ti=6` prêt sans appelant), R09
cadrage étiré sur cartes Forge, R99 gate visuel des fonds (XS côté utilisateur, débloque R06).

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

- 2026-09-09 ; `handlers/helpers.go:113` ; cinq handlers gèrent déjà `If-None-Match` — le plan
  WebP l'ignorait. Traité par S5 (amendement de D9), pas reporté.
- 2026-09-09 ; `apps/web/src/features/{ascension,explorer,palmares,squad/charts}` ; quatre
  heatmaps ECharts construites à la main hors `Heatmap2DChart` en plus de la Synthèse. S6.
- 2026-09-09 ; worktree partagé ; 116 fichiers de trois chantiers terminés non commités, dont
  `SquadIsolementNuageCard.tsx`, `squad/i18n.ts`, `SessionUsageForms.tsx` — collision directe
  avec C3, A3, E4. D1.
- 2026-09-09 ; `main` ; 13 commits absents de `feat/v75`. D3.
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
