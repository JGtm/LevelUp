# Plan — compléments de la 8.2.0 (2026-11-14)

Statut : **en cours** — go du user le 2026-11-14 (branche `feat/complements-820` acceptée).
Maquettes : artefact `6Dk4Su2NEYppk61VL1zftV` (version 2). Base : `origin/release/v8.2.0` à `e83b4f4c2c`.

## 0. Objectif et critère de succès

Ajouter à la 8.2.0, déjà relue et prête à fusionner, les demandes du user du 2026-11-14 et les entrées du backlog
retenues. Succès : chaque item ci-dessous statué `[x]` / `[~]` / `[!]` ; suite Go (`go test ./...` et
`-tags=integration` sur les paquets touchés), `make check-types`, `make test-web`, lint sans régression de baseline ;
un démarrage du serveur sur copie des bases ne produit plus aucun des avertissements classés « problème » au §2 (témoin
avant/après capturé) ; relecture indépendante sans P0/P1 ; notes de version et changelog alignés
(`TestReleaseNotesDocs_*`) ; contrôles à l'écran du user ajoutés à l'artefact des vérifications.

## 1. Contrat d'exécution

- Skill `plan-execution` : ordre strict, une étape à la fois, gate par étape, aucun report hors dépendance.
- Worktree dédié `LevelUp-wt-compl820`, branche `feat/complements-820` depuis `origin/release/v8.2.0` (nom à valider), **sans
  jonction vers `data/`** (incident du 13/11) ; mesures sur COPIES À FROID des bases dans le scratchpad ; caches Go dédiés
  `go-build-compl820` / `golangci-compl820`.
- Coordination avec `levelup-dd` (pilote de clôture 8.2.0, consignes du 2026-11-14) : envois directs sur
  `release/v8.2.0` ; à chaque envoi visible par l'utilisateur, notes de version et changelog FR/EN à jour
  (`TestReleaseNotesDocs_*` verts) et une ligne dans l'artefact des vérifications (collection `verifs`) par contrôle à
  l'écran ; CI verte attendue ; SHA de fin de lot annoncé à `levelup-dd`, qui fait relire par un contexte frais avant la
  fusion dans `main` (geste du user).
- **Aucune lecture ni copie des bases locales tant que `levelup-7c` n'a pas fini ses calculs dérivés** (étapes 0.2, 0.3,
  1.5, 1.9, 7.1, 7.4 en dépendent ; les autres étapes partent sans elles).
- **Voies parallèles** (dérogation à l'ordre strict, décidée au go) : les étapes 1 à 10 sont indépendantes (fichiers
  disjoints) et s'exécutent en voies, un worktree et une branche par voie (`LevelUp-wt-c820-<voie>`, branche
  `feat/c820-<voie>` depuis `origin/release/v8.2.0`), au plus trois voies à la fois (mémoire du poste : environ 7 Go
  libres pendant les passes de levelup-7c). Chaque voie close est rebasée sur `origin/release/v8.2.0` et poussée
  dessus, SHA annoncé à `levelup-dd`. Ordre : voies sans base (2 à 6, 8 à 10, puis 1 hors mesures), puis 7 et les
  mesures de 1 après le signal de levelup-7c.
- Agents d'exécution Sonnet par défaut, Haiku pour les remplacements mécaniques ; une ronde `adversarial-review` Sonnet
  sur les étapes 1 (logs et persistance), 7 (note) et 8 (rejeu).
- Les cinq entrées film (trajets, sièges, schéma, ti=40, `5da6fd30`) sont transmises à `levelup-1f` (chantier film) le
  2026-11-14 : hors de ce plan. Les entrées 3 et 5 demandent une régénération ciblée (la recuisson du parc est faite).

## 2. Décisions

- **D-1 (user, 14/11)** : tous les avertissements qui signalent un problème sont corrigés à la source ; le bruit normal
  est résumé ou abaissé (« pas pro »).
- **D-2 (user)** : Tendances, stats par minute en groupe « Rythme » de la matrice et de l'évolution, agrégat = somme des
  comptes / somme des minutes jouées.
- **D-3 (user, 14/11)** : placement en carte de chaleur A1. Vocabulaire « Groupé ou isolé » (section), quarts « groupé
  et rentable / isolé et rentable / groupé et coûteux / isolé et coûteux » (« accompagné » refusé : trop passif).
- **D-4 (user, 14/11)** : rôles de portée en ruban B1a : trois couleurs, case vide avant la fenêtre, petit échantillon dans l'infobulle seulement.
- **D-5 (user)** : infobulles natives remplacées par une couche globale déléguée (254 `title` sur éléments HTML).
- **D-6 (user)** : rang de session en courbes à triangles de sens ; échelle par bandeau joueur, même amplitude (C4).
  Le graphe « par minute » garde son opposé colorimétrique (réponse du user du 14/11 : non).
- **D-7 (user)** : note de performance, poids de l'objectif selon la part de l'objectif de l'équipe ; objectif mesuré par
  l'indice par mode existant (`analysis/narrative/objective_participation.go`, actions de soutien comprises) ; comparaison
  par famille de mode ; pas de « points requis pour gagner » (double compte du résultat, déjà dans `rank_perf`). Poids
  figés seulement après simulation sur le parc local et accord du user.
- **D-8** : constante `DefaultEffectiveHpToKill` (225 → 229 ou 230) traitée dans la même révision que D-7 : un seul
  recalcul au démarrage (ADR 0047).
- **D-9 (user, 14/11)** : rejeu, TROIS calques séparés regroupés sous « Vies » dans le panneau : Apparitions (goutte
  creuse orientée, S1), Morts (croix), Tueurs (triangle creux + trait vers la victime) ; affinages communs au groupe :
  qui (tous, équipe, joueur suivi) et quand (partie, jusqu'ici) ; règle « vivant plein, trace creuse ». Préréglages :
  Combat = Morts + Tueurs, Déplacements = Apparitions, Complet et Objectif = éteints. 16 calques au lieu de 13.
- **D-10 (user, 14/11)** : Appui en flux F1, vocabulaire « assistant → finisseur », flèche. Phrases toujours à la voix
  active, sujet = l'assistant (« Madina a assisté JGtm sur 14 frags »), jamais « X a été assisté par » ; en-têtes de
  colonnes en mini-phrase (« Assistant : prépare le frag », « Finisseur : conclut le frag ») ; infobulle d'un joueur :
  assistances données et frags conclus sur assistance.
- **D-11 (user)** : backlog retenu : cause de l'échec du code d'appareil, état réel du sync sur l'accueil, version de
  l'application dans les signalements, ADR 0020 et 0006, budget de `gate-push`, binaire vide de `desktop-test`, badge du
  rejeu testé de bout en bout, `info-tooltip` au focus et garde-rail de couleurs. Reportés : pseudos réels (deux
  entrées), alerte glib (écartée : Windows seul).
- **D-13 (vérifié sur pièces, 14/11) : le calque « Vies » ne change PAS le document de rejeu, aucune régénération.**
  Tout est déjà servi et lu côté web : pistes par siège avec cap par image (`replaydoc/document.go`, `Point.H`), image de
  réapparition (`lib/replay/rosterLogic.ts`, `respawnFrame`), lieu de mort (`killFx[].deathX/deathY`,
  `useReplayHeatmap.ts`), position du tueur résolue à l'image du frag (`model/killFocusLogic.ts`, `resolveActor`). Si
  l'exécution découvre qu'un champ manque : arrêt et décision du user, jamais de champ cuit ni de régénération dans ce
  plan.
- **D-14 (vérifié sur pièces) : un seul recalcul des notes en prod.** `MatchRecomputer.ApplyScoringRevision`
  (`sync/match_recomputer.go:227`) compare une seule chaîne composée (`skill.ScoringRevision()` = perf + LUSR) à celle
  rangée par joueur, et un seul `recompute` rejoue note de performance ET LUSR. Changer `PerformanceScoreRev` et
  `LUSRRev` dans la même release = un recalcul au premier démarrage, pas deux (test à ajouter : deux montées, un appel à
  `recompute`). La phrase des notes de version sur le LUSR (« usually by less than five points », `docs/RELEASE_NOTES.md:33`
  et son jumeau FR) est réécrite avec les écarts mesurés par la simulation 7.4 (médiane, 90e centile, part des matchs qui
  changent de palier), pour la note et pour le LUSR.
- **D-15 : gestes de prod.** La suppression de `streak` (bases joueur) et toute réparation de la précision sont des
  migrations qui s'appliquent seules au démarrage (registre de migrations, skill `db-schema`), précédées de la sauvegarde
  habituelle du VPS ; séquence inscrite dans `.ai/annexes_v8_1_2/gestes_prod_8_2_0.md` (sauvegarde, démarrage, contrôle
  du log : plus de refus de réparation, recalcul des notes terminé pour chaque joueur). **Aucune copie de base locale
  vers le VPS** (règle du user : les joueurs ne sont pas les mêmes) : les copies à froid restent dans le scratchpad, pour
  mesurer seulement ; en prod, tout changement de données passe par les migrations au démarrage ou par des passes
  `backfill-*` lancées sur le serveur.
- **D-12** : compaction des passes sur le bureau (§4, 1.3) : au démarrage, avant l'ouverture des bases, si la jauge
  dépasse le seuil — à confirmer par le user (la doctrine actuelle dit « jamais au boot »).

## 3. Questions ouvertes au user

- Q-5 Témoins de la note : le dernier match d'escouade en Drapeau sur Houseki (proposé par le user, court et peu de
  frags) ; chercher dans les bases, après le signal de levelup-7c, de meilleurs témoins (critères en 7.4a).
- Q-6 réglée : branche `feat/complements-820`, go donné le 2026-11-14.

## 4. Étapes

### Étape 0 — Préparation
- [ ] 0.1 Worktree, branche, caches dédiés ; aucune jonction.
- [ ] 0.2 Copies à froid de `shared_matches_v2`, `metadata` et des bases joueur (serveur local arrêté ou fichiers non tenus) dans le scratchpad.
- [ ] 0.3 **Témoin avertissements** : démarrage du binaire de la branche sur les copies, log capturé et classé (même grille que `warn2.txt`).
- [ ] 0.4 Commit du plan sur la branche ; annonce à `levelup-dd`.

### Étape 1 — Avertissements
- [x] 1.1 `platform/duckdb/sharedprovider/provider_writer.go:217` et `reader_inflight.go:101` : une ligne au franchissement du seuil, une ligne à la libération (durée, lectures retenues), plus de répétition périodique ; test de comptage des lignes. (fait ; voir Journal, voie logs)
- [x] 1.2 `sync/skill/skill_v2_solver_fallback.go:61` : compteur par rejeu, une ligne de synthèse en fin de rejeu (matchs, replis). Instruire la non-convergence de l'EP (part des matchs, profil des matchs en cause) ; correction seulement si la cause est établie, et alors dans la révision de l'étape 7. (fait ; non-convergence instruite, correction renvoyée à l'étape 7)
- [x] 1.3 `scheduler/data_health_passes.go:102` + `migration/compaction.go` : profil bureau, compaction au démarrage avant ouverture si au-dessus du seuil (D-12), durée mesurée sur copie, événement de démarrage visible par la coquille ; message sans « de nuit » hors VPS. ADR 0045 amendé. (fait ; durée mesurée sur copie : 11 s)
- [x] 1.4 Table héritée `streak` : vérifier qu'aucun lecteur ne la lit (`streaks_repo.go:117` lit-il `streak` ou `streak_latest` ?), retirer sa création (`steps_player_base.go:228`) et la supprimer par migration après la recopie de `steps_player.go:201` ; plus de refus de réparation au démarrage. (fait)
- [x] 1.5 Précision en 0..100 (`records`, valeur 61,73) : trouver le match et la source d'écriture, corriger l'échelle à l'ingestion, réparer les lignes touchées par la voie append-only ; le record se pose. (fait : lecture en ratio et migration des records rangés en pourcentage)
- [!] 1.6 Bail joueur annulé pendant le recalcul (`dblease/writer.go:198`, `career_live_service.go:460`, `db_recovery.go:180`) : cause établie ; l'écriture partielle de carrière n'est plus perdue (attente ou reprise) ; plus de connexion fermée sous un lecteur. (bail : fait ; connexion fermée sous un lecteur : cause par le code, correction NON faite, voir Journal)
- [x] 1.7 `world_leaderboard_cron.go:290,415`, `playlist_expansion_service.go:78` : une erreur 5xx du service Halo = une ligne de synthèse par passe (playlists en échec, statut), niveau WARN. (fait)
- [x] 1.8 `pool/pool.go:642`, `watcher/match_poller.go:104` : 429 absorbé par le retrait = INFO compté ; WARN seulement si le retrait échoue ou dure. (fait)
- [x] 1.4b et 1.5b Skill `db-schema` appliqué ; migrations au registre (démarrage), tests sur base joueur legacy et neuve ; séquence prod ajoutée à `.ai/annexes_v8_1_2/gestes_prod_8_2_0.md` (D-15). (fait)
- [x] 1.9 Gate : témoin 0.3 rejoué, comparaison ligne à ligne ; restent seulement les replis nommés du décodeur et les cartes non résolues (normaux, D-10 ADR 0034). (fait : 230 WARN et ERROR avant, 38 après)

### Étape 2 — Tendances, groupe « Rythme »
- [x] 2.1 `analysis/trends/indicators.go` : frags, morts, assistances, dégâts infligés et subis par minute (`Match.Seconds`), rapport des sommes, `inMatrix`, sens (morts : bas = mieux) ; groupe et clés de domaine ; vue escouade (`[~]` : la vue escouade ne porte aucun indicateur de combat par membre, seulement FDA et part des frags ; un allié n'a ni secondes jouées ni dégâts dans `MemberStat` ; le groupe est absent de cette vue, test à l'appui). Code : `indicators_pace.go`, `TrendsGroupPace`, `TrendsUnitPerMin`, cinq clés `*_per_minute`.
- [x] 2.2 Libellés FR/EN (manifeste `tendances`), unités ; tests Go et web ; `make generate-types` si le contrat change (inchangé : `group`, `unit` et `key` sont des chaînes libres dans `openapi.yaml`). Deux graphiques « Rythme » ajoutés à la grille « Niveau, combat et style » de l'Évolution.
- [ ] 2.3 Règle du user (09/11 : un indicateur multi-matchs vit sur Sessions, Séries temporelles ET Escouade, et Sessions a sa vue compacte normalisée). Frags, morts et assistances par minute y existent déjà (`session-detail/_chartSections.tsx` « FDA par minute », `timeseries/TimeseriesPage.progression.tsx`, `squad/charts/squadPerMinuteChart.ts`) ; ajouter les dégâts infligés et subis par minute sur les trois surfaces, vue compacte normalisée de Sessions comprise, avec les mêmes règles (rapport des sommes, match sans temps joué exclu, absent si aucun dégât connu). Escouade : seulement si les dégâts par membre existent dans les données, sinon `[!]` avec le constat. Côté Go, toute cadence passe par `analysis.PerMinute` (`internal/analysis/indicators.go:146`), jamais une nouvelle division.

### Étape 3 — Placement A1
- [x] 3.1 `features/squad/emprise/placementCharts.ts` : carte de chaleur (cases distance × frags, quarts, repère radar, médianes par joueur), infobulle de case (part, compte). Fait (voie escouade, `9aa2b26b56`) : grille pure `placementHeatmap.logic.ts`, série `custom` de rectangles `placementHeatmapChart.ts` (le type `heatmap` d'ECharts impose des axes catégoriels, qui ne portent ni le seuil d'isolement du contrat ni les gros points à position continue), gros point par joueur = distance médiane x frags MOYENS par vie, hauteur de la carte 420 -> 520.
- [x] 3.2 Vocabulaire D-3 dans `placementStrings.ts` (FR/EN) et garde-rails de textes ; « Part des vies par placement » inchangée. Fait (`9aa2b26b56`) : section « Groupé ou isolé », quatre quarts « GROUPÉ ET RENTABLE… », noms courts de la barre alignés, titre « Distance au coéquipier le plus proche et frags, par vie ».

### Étape 4 — Rôles de portée B1
- [x] 4.1 `SquadRangeRolesCard` + `squadRangeRolesChart.ts` : ruban joueur × match, rôle de la fenêtre glissante, case vide avant `FENETRE_ROLE`, échantillon selon Q-2 ; logique pure inchangée (`squadRangeRoles.logic.ts`). Résumé solo inchangé. Fait (`529ff466a0`) : `charts/squadRangeRolesRibbon.ts` (nouveau ; `squadRangeRolesChart.ts` reste pour le Résumé solo et les Sessions), trois jetons `range-role-front/mid/far` (aucun trio existant ne convenait : `heatmap-cold/hot` portent un jugement bon/mauvais), test de contraste et de séparation daltonisme par palette, bande repliable `SquadRangeRolesTape` supprimée (le ruban la remplace).

### Étape 5 — Infobulles
- [x] 5.1 Couche globale (`components/ui/`) montée dans la coquille : survol et focus clavier d'un `[title]`, titre déplacé en attribut de données pendant l'affichage, `aria-describedby`, style de `tooltip.tsx`, position bornée à la fenêtre ; SVG `<title>` (2) traités.
- [x] 5.2 `info-tooltip.tsx` : focus filtré au clavier (entrée du backlog) ; garde-rail de couleurs étendu (`ring-offset`, `outline`, `decoration`, `accent`, `caret`, portée de `color-allow`).
- [ ] 5.3 Contrôle : partage de fenêtre Discord par le user (artefact des vérifications).

### Étape 6 — Rang de session C4
- [x] 6.1 `squadPerformanceLineCharts.ts` (`buildTeamMMROption`) : un bandeau par joueur, même amplitude (plus grand écart de la session + marge), triangles de sens, rang final et écart à gauche ; MMR d'équipe en bandeau séparé ; `hexComplement` retiré de ce graphe. Fait (`939f636650`) : `buildTeamMMROption` supprimé de `squadPerformanceLineCharts.ts` (568 -> 449 lignes) au profit de `charts/squadRankStripsChart.ts` + `squadRankStrips.logic.ts` ; `squadPerMinuteChart.ts` et `hexComplement` intacts.
- [!] 6.2 `squadPerMinuteChart.ts` inchangé : réponse du user du 14/11 (garder l'opposé sur ce graphe).

### Étape 7 — Note de performance et vie à 229/230
- [ ] 7.1 Mesure 229 / 230 sur davantage de films (compteur de dégâts subis sur vies sans recharge), décision chiffrée.
- [ ] 7.2 Part d'objectif de l'équipe par match : depuis `match_objective_stats_latest` (toutes les lignes de l'équipe), pondération `ObjectiveFamilyActionWeights`, par famille.
- [ ] 7.3 `sync/performance.go` : métrique objectif = indice par famille du match (au lieu des points `objective` des awards), poids α = 0,12 + 0,38 × part ; percentile par famille de mode (repli sur la chaîne sous 10 matchs).
- [ ] 7.4a Témoins (après le signal de levelup-7c) : matchs à objectif en escouade, durée normale (au moins 8 min), un joueur à forte part d'objectif et peu de frags, un coéquipier d'escorte (passes de capture, retours, porteurs abattus), un tueur sans objectif dans le même match ; plusieurs modes (Drapeau, Bases, Colline) ; Houseki gardé en témoin court. Liste remise au user.
- [ ] 7.4 Simulation sur copie du parc : distribution des notes avant/après par chaîne, notes inchangées à part nulle, matchs témoins (Q-5) ; tableau remis au user ; poids figés avec lui.
- [ ] 7.5 `PerformanceScoreRev` et `LUSRRev` changés une fois (D-14 : test d'un seul recalcul pour deux montées ; phrase des notes de version réécrite avec les chiffres de 7.4) ; ADR 0047 et 0006 amendés ; textes d'aide de la note et de « 1 vie ≈ 225 points » ; `/bootstrap` (`effective_hp_to_kill`).

### Étape 8 — Rejeu, calque « Vies »
- [x] 8.1 Données : apparition = première image de chaque vie et cap de visée de cette image ; mort = position de la victime ; tueur = position du tueur à l'image du frag ; rien sans position (compté). `model/livesMarks.ts` (`buildLivesMarks`, compteur `skipped`), aucun champ ajouté au document (D-13 tenu).
- [x] 8.2 Trois calques `spawns`, `deaths`, `killers` dans `viewPresetLogic.ts` (16 calques), groupe « Vies » dans le panneau (`ReplayViewPanel.tsx`, `ReplayLivesSection.tsx`), affinages communs qui / quand ; préréglages selon D-9 ; garde-rails de calques mis à jour.
- [x] 8.3 Peintre canvas : goutte creuse orientée, croix, triangle creux, trait tueur → victime ; fondu par âge en « jusqu'ici » ; encres d'équipe (`canvasInk`) ; textes FR/EN. `layers/livesLayer.ts` (couleur d'équipe des pions + liseré `labelStroke`), `layers/useReplayLives.ts`, `i18n/lives.i18n.ts`.

### Étape 9 — Appui
- [x] 9.1 `SquadAppuiCard.tsx` en flux assistant → finisseur (D-10), même forme sur la page Match (`MatchAssistChart`) ; textes FR/EN à la voix active ; garde-rail `vocabulaireCoordination.guard.test.ts` ajusté ; infobulles de ruban et de joueur. Fait (voie escouade, dernier commit de la voie) : `features/_shared/assists/` (logique, option `sankey`, composant, textes FR/EN UNIQUES des deux pages), `matchAssistFlow.logic.ts`, garde-rail étendu à `match-view` et aux textes partagés (larbin, patron, minion, lackey, boss, « a été assisté »), props devenues sans appelant retirées de `components/charts/BarStackedChart.tsx` avec leurs tests.

### Étape 10 — Backlog retenu
- [ ] 10.1 Code d'appareil : distinguer services Halo injoignables d'un refus de compte (`handlers/auth.go`, `pollDeviceFlow`).
- [ ] 10.2 Accueil : état réel du sync au lieu de `total_matches == 0` (`features/home/HomePage.tsx`).
- [ ] 10.3 Signalements : version de l'application depuis `/bootstrap`.
- [ ] 10.4 ADR 0020 (`PatternReport`) et 0006 (quotient) corrigés.
- [ ] 10.5 `gate-push` : budget des trois paquets lourds.
- [ ] 10.6 `desktop-test` : binaire vide nettoyé ou ignoré par `TestLesBinairesDeAppsSontCouverts`.
- [ ] 10.7 Badge du rejeu testé au niveau de la page.

### Étape 11 — Clôture
- [ ] 11.1 Relecture indépendante (contexte frais, `adversarial-review`) du diff complet ; annonce du SHA à `levelup-dd`.
- [ ] 11.2 Notes de version et changelog FR/EN (dont le recalcul des notes au premier démarrage), `TestReleaseNotesDocs_*` verts.
- [ ] 11.3 Contrôles à l'écran ajoutés à l'artefact des vérifications.
- [ ] 11.4 `.ai/BACKLOG.md` (entrées traitées retirées, dont 229/230 versée puis close), `REGISTRE_REPORTS.md`, `thought_log`, nettoyage des caches du lot.

## 5. Risques
- La 8.2.0 était close : toute modification rouvre la relecture et les notes (coordination D-1 avec `levelup-dd`).
- Étape 7 change toutes les notes au premier démarrage : poids figés seulement après simulation et accord.
- Étape 1.3 touche le démarrage du bureau (durée de compaction) : mesure avant d'activer.
- Étape 8 : calques et préréglages ont des garde-rails de contrat ; respecter « Complet = état d'ouverture ».

## 6. Hors périmètre
Entrées film (`levelup-1f`), pseudos réels, alerte glib, toute recuisson.

## 7. Gates par voie (commandes exactes)

- Web (étapes 3, 4, 5, 6, 8, 9, et la part web de 2 et 10) : depuis `apps/web`, `npx tsc --noEmit -p .`,
  `npx vitest run <dossiers touchés>` puis `npx vitest run` complet avant le rebase, `node tools/lint-no-hardcoded-colors.mjs`,
  `node tools/knip-ratchet.mjs` ; manifests i18n régénérés (`node scripts/build_i18n_manifests.mjs`) et leur test de
  fraîcheur vert.
- Go (étapes 1, 2, 7, 10) : `go test ./internal/<paquets touchés>/...`, puis `go test -tags=integration` sur `persist`,
  `sync`, `migration`, `platform/duckdb` quand ils sont touchés (budget `-timeout 40m`, entrée gate-push) ;
  `golangci-lint run` sans nouvelle violation ; `make generate-types` puis typecheck web si le contrat change.
- Avant chaque push sur `release/v8.2.0` : hooks `pre-push` verts (sans `--no-verify`), `TestReleaseNotesDocs_*` verts si
  les notes bougent, puis CI verte attendue ; SHA annoncé à `levelup-dd`.
- Multi-titre : les nouveaux indicateurs (2) et la note (7) se branchent sur les capacités du titre (Halo 5 sans
  `match_objective_stats` : métrique absente et poids renormalisé, jamais de panique) ; test du cas Halo 5 dans chaque voie
  Go.

## 8. Reprise, découvertes, journal

- Reprise : ce fichier fait foi ; relire le skill `plan-execution`, puis le journal ci-dessous, reprendre à la première case
  non statuée de chaque voie ouverte.
- Découvertes : consignées ici, non traitées sauf si elles bloquent un gate.

### Découvertes

- 2026-11-14 (voie infobulles) : 23 marqueurs `color-allow` nus restants n'excusent plus rien (marqueur placé sur une autre ligne que la couleur, ou ligne sans couleur) : commentaires à retirer dans une passe de nettoyage, non traités ici.
- 2026-11-14 (voie infobulles) : le lint des couleurs ne balaie pas les préfixes `shadow`, `from`, `via`, `to` avec une couleur nommée (seulement le noir et le blanc absolus) ; hors de l'entrée du backlog, non traité.
- 2026-11-14 (voie infobulles) : un bouton `disabled` ne reçoit pas `pointerover` dans tous les navigateurs ; la couche ne peut pas y prendre en charge le `title` (l'infobulle native y reste). À contrôler à l'écran dans WebView2 (point 5.3) ; non relevé site par site.
- Voie rejeu (étape 8) : la goutte des apparitions n'a une pointe que si le premier point de la vie porte un cap (`Point.h` non nul : le champ est `omitempty`, un cap exactement nul est absent). Sinon anneau seul, par construction (règle « sans cap lu à cette image »). À constater à l'écran sur un rejeu réel ; non traité ici.
- 2026-11-14 (voie `tendances`) : `archlint.TestNoNewFrenchLabelLiteral` ROUGE sur la base `release/v8.2.0` : `api/handlers/background_variante.go` porte 1 littéral FR hors allowlist (fichier introduit par `1f4623c326`, étranger à la voie ; non traité ici). À corriger par la voie qui l'a posé (allowlist datée ou clé canonique).
- **Garde « aucun inconnu à l'écran » (`lib/i18n/noUnknownMentions.guard.test.ts`) contre la consigne du 14/11** : elle interdit « frags mesurés » dans toute chaîne d'interface. L'infobulle d'une case de petit échantillon du ruban de portée dit donc « 3 frags, match peu sûr » (et non « 3 frags mesurés, match peu sûr »), et l'aide de la carte « moins de N frags ». Si le user veut le mot « mesurés », c'est la règle du 09/10 à amender, pas ce lot.
- **Ruban de portée : la règle « un match creux ne porte pas de rôle » est inchangée** (`rolesFenetre`, son test) : un match sous le plancher est une case vide, survolable, dont l'infobulle dit « match peu sûr ». Si le user attendait qu'un match creux garde le rôle de sa fenêtre (la moyenne glissante ignore déjà ses frags), c'est une décision de règle à prendre.
- **Placement : le gros point d'un joueur porte la moyenne des frags de ses vies** (consigne), calculée depuis `lives[]` ; le contrat sert `median_kills`, gardé en repli quand la liste des vies est vide. Une moyenne n'est pas un champ du contrat.
- **`BarStackedChart`** : ses quatre props propres à l'ancien « Appui » (`tooltipComponentNote`, `categoryAxisName`, `valueAxisName`, `tooltipRoles`) et l'estimation de la bande d'étiquettes sont retirées (aucun appelant restant).
- **Gardes de balayage de l'arbre sous charge** (`colorDistance.guard`, `locale.ratchet`, `perf-tier.guard`, `squadCompositionPicker.guard`, `fragClass.colorSource.guard`, `manifestGenerator`) : dépassent 5 s quand deux suites tournent ensemble ; verts seuls avec `--testTimeout=60000`. Sans rapport avec ce lot.
- 2026-11-14 (voie logs) : le jalon `accuracy_threshold_days` compare la colonne `match_participants.accuracy` (pourcentage 0 à 100, valeur native de l'API) au seuil 0,50 (`platform/duckdb/post_sync_progression_store.go` Aggregates, `ops/milestone_dates.go`) : tout match au-dessus de 0,5 % compte, le jalon est atteint trop facilement. Même échelle fausse que celle des records, corrigée seulement pour les records (1.5). Le corriger change des jalons visibles : décision du user.
- 2026-11-14 (voie logs, 1.2) : proposition pour l'étape 7 (révision ADR 0047) : le solveur EP (`analysis/skill_v2/ep`) est un passage en boucle sans amortissement (`Runner.Run`, 500 passes, tolérance 1e-4) sur un graphe où chaque joueur est observé deux fois (frags et morts, `SumFactor` avec la moyenne de TOUS les adversaires) : les cycles et les écarts de taille d'équipe (4 contre 6 dans la fixture) font osciller les messages. Deux pistes à mesurer sur le parc avant toute décision : un amortissement des messages sur les seuls facteurs de comptes, et une variance d'observation plus large quand les comptes s'éloignent de l'espérance. Ni l'une ni l'autre n'a été essayée ici (toute formule change `LUSRRev`).
- 2026-11-14 (voie logs, 1.6) : `OpenReadForQuery` rend un instantané sans possession (doctrine de `platform/duckdb/read_recovery.go`). Sur un chemin SANS fournisseur (base joueur, metadata, pve), un emprunt qui incrémente le compte de références supprimerait la classe « le propriétaire ferme sous le lecteur » sans casser le B-swap, qui ne concerne que les chemins gérés par `sharedprovider`. Proposition : un registre des chemins gérés par un fournisseur (posé par `sharedprovider.New`) et un emprunt possédant partout ailleurs ; à mesurer (fuites de références : tout appelant doit libérer) avant de l'engager.
- 2026-11-14 (voie logs, 1.3) : [mesuré : 11 s sur la copie, marge de sept fois] la coquille attend `ready` 120 s au plus et le contrat d'événements n'a pas de phase de maintenance : la compaction du démarrage est bornée à 75 s (`compactionBootBudget`). Si la mesure sur copie dépasse ce budget, il faudra une étape de porte `maintenance` (`boot_gate.go`, libellés web) et un événement `maintenance` dans l'ADR 0045 et la coquille Rust. Non ajouté ici : le contrat ne le prévoit pas et la durée n'est pas mesurée.
- 2026-11-14 (voie logs, 1.4) : `canonicalOrderFloor` passe à 239 (`migration/order_floor_test.go`) avec les steps `drop_streak_legacy_v1` et `records_accuracy_to_ratio_v1` ; toute autre voie qui ajoute une étape de migration doit relever le plancher à la valeur suivante au rebase.

### Journal

- 2026-11-14 : plan relu par la grille `plan-review` (gates, reprise, multi-titre ajoutés) ; voies `escouade`,
  `infobulles`, `rejeu` ouvertes.
- 2026-11-14 (voie infobulles, branche `feat/c820-infobulles`) : 5.1 et 5.2 faits. Couche `TitleTooltipLayer` montée dans `AppProviders` (couvre connexion et pages hors coquille) ; `Tooltip` et `InfoTooltip` marqués `data-lu-tooltip` (pas de double infobulle) ; `InfoTooltip` ouvert au focus clavier seulement ; lint des couleurs étendu (cinq préfixes) et `color-allow(<genre>)` borné au genre nommé (76 lignes migrées, aucune allowlist élargie). 5.3 reste au user.
- 2026-11-14 : voie rejeu (étape 8) faite sur `feat/c820-rejeu` : trois calques « Vies » (apparitions, morts, tueurs), seize calques, affinages qui / quand, préréglages D-9, textes FR/EN, aucun changement du document de rejeu (D-13). Contrôle à l'écran remis au user.
- 2026-11-14 (voie `tendances`, branche `feat/c820-tendances`) : étape 2 faite. Groupe `pace` (« Rythme ») de la matrice et deux graphiques de l'Évolution ; agrégat somme des comptes / somme des minutes ; match sans seconde jouée exclu des deux termes ; dégâts absents (nil) = ligne absente, pas zéro. Vue escouade : non concernée (voir 2.1).
- 2026-11-14 (voie `tendances`, gates) : `go test ./...` 300 paquets : tout vert sauf `archlint` (1 test rouge, base, voir Découvertes) ; `archlint` et `sync` ont dépassé 40 min sous charge dans la passe complète (machine partagée), relancés seuls : `sync` vert (2038 s), `archlint` : seul `TestNoNewFrenchLabelLiteral` rouge. Web : `vitest` complet vert avec `--testTimeout=60000` (deux gardes de balayage dépassaient 5 s sous charge dans la passe ciblée).
- 2026-11-14 : voie `escouade` (étapes 3, 4, 6.1, 9) exécutée dans `LevelUp-wt-c820-escouade`, branche `feat/c820-escouade` : quatre commits (un par étape), gates web verts, contrôles à l'écran à faire par le user (carte de chaleur du placement, ruban des rôles, bandeaux du rang, flux d'appui sur Synergies et sur la page Match).
- 2026-11-14 (voie infobulles, ronde 2) : relecture indépendante, cinq constats corrigés sur `feat/c820-infobulles`. P1-1 : le `title` est VIDÉ (pas retiré) pendant la prise en charge et un observateur ciblé suit les réécritures (texte neuf, infobulle mise à jour) et retraits (prise oubliée, rien restitué) par l'application ; `<title>` SVG idem. P1-4 : changelogs et notes de version sans « tout / toutes ». P2-2 : prises en charge pointeur et focus indépendantes. P2-3 : une infobulle ouverte au focus suit son contrôle au défilement (masquée seulement hors fenêtre). P2-5 : tests des sorties de page (fenêtre cachée, perte du focus), vérifiés par mutation.
- 2026-11-14 : ronde 2 de la voie rejeu (relecture indépendante, lot poussé à abf8dd2650 / 97d992c9c4) : P1 les fins de manche ne donnent plus de croix (seule une mort neutre à cause établie par le film en donne une) ; P2-a un préréglage Combat ou Déplacements retenu avant le lot reste reconnu (défaut des trois calques tiré du préréglage retenu) ; P2-b descriptions des préréglages corrigées FR et EN, tenues par un test contre la table ; P2-c Seul et Masquer s'appliquent aux traces des vies ; entrée de backlog pour les effets de mort (même filtre, règle écrite comme voulue dans `playerPanelLogic`).
- 2026-11-14 : ronde 2 bis de la voie rejeu : (1) le document ne porte pas de fin de manche lue vie par vie (`scoreTimeline` ne date que le dernier palier de chaque manche, les vies se ferment 7,6 à 9,1 s plus tard) : critère « cause établie » conservé, rien de cuit (D-13), limite écrite dans `deathFacts`, entrée de backlog (morts sans type, `repli_repere_neutre_generique_conserve`, exemple 58d09c44, effort M) et phrase ajoutée aux notes de version FR et EN ; (2) le défaut des vies tiré du préréglage retenu ne s'applique que si les treize calques d'avant le lot sont encore à l'état de ce préréglage ; (3) commentaire de `shownSlotPredicate` corrigé (les traces des vies s'y plient, les effets de mort non).
- 2026-11-14 : ronde 2 de la voie `escouade` (relecture indépendante, constats P2-2 à P2-6 ; le P1 sur les chaînes LUSR de `rankStrips` attend la décision du user) : état vide du ruban quand aucune case n'a de rôle, marges du flux d'appui selon la largeur mesurée (libellés sur les rubans sous 520 px), notes de version aux libellés exacts de l'écran et au comportement réel, README de `BarStackedChart` sans `tooltipComponentNote`.
- 2026-11-14 : ronde 2 de la voie `escouade`, constat P1 (décision du user) : le rang de session passe par un sélecteur de chaîne (CSR classé = une chaîne, LUSR par `skill_playlist_group`), la plus jouée par défaut, choix gardé dans l'état de la page ; écart de session, étendue et amplitude commune calculés sur la chaîne choisie seule, libellés de la Carrière (`career.lusr.chain.*`) avec le nombre de matchs, pas de sélecteur avec une seule chaîne.
- 2026-11-14 (voie logs, branche `feat/c820-logs`, base `origin/release/v8.2.0`) : étape 1 hors mesures sur les bases, sans lire ni copier aucune base locale.
  - 1.1 [x] `sharedprovider` : un dépassement de seuil se dit UNE fois par épisode (`hold_journal.go` : série de dépassements de la même clé séparés de moins de 5 min), avec le nombre de dépassements tus de l'épisode précédent ; la ligne de libération (`writer RW libéré après dépassement du seuil`) porte la durée totale et les lectures retenues. Les compteurs expvar voient chaque dépassement. Même principe pour « lecture tenue longtemps » (clé = appelant). Cause des ~200 lignes : le rejeu des notes prend l'écrivain partagé des centaines de fois, chaque prise au-delà de 2 s, et le watchdog écrivait une ligne par prise. Tests : `hold_journal_test.go` (200 dépassements, une ligne), `writer_watchdog_lignes_integration_test.go` (12 prises, 1 ligne de dépassement et 1 de libération avec `lectures_retenues=1`, compteur +12 ; 5 lectures longues, 1 ligne), `-race` vert.
  - 1.2 [x] `skill_v2_solver_fallback.go` : un `repliTally` posé sur le ctx du rejeu (`runLUSRV2Shadow`) compte les matchs en repli ; le détail par match passe en DEBUG, une ligne de synthèse WARN en fin de rejeu (`matchs_rejoues`, `matchs_en_repli`, trois `match_id` d'exemple). Sans tally (appel direct) le comportement WARN par match est conservé. Non-convergence instruite par le code, sans changer le solveur : `ep.Runner.Run` est un passage en boucle sans amortissement (500 passes, tolérance 1e-4 sur le plus grand écart de message) ; chaque joueur est observé deux fois (frags puis morts, `SumFactor` avec la moyenne de TOUS les adversaires, variance d'observation 25) ; les cycles ainsi créés, les équipes de tailles inégales (la fixture est un 4 contre 6 avec deux poids de temps joué à 0,5 et un abandon) et les comptes très éloignés de l'espérance (0 à 20 frags) font osciller les messages. Aucun réglage essayé (plus d'itérations, amortissement) ne rattrape plus de la moitié de ces matchs (commentaire du fichier). Proposition consignée en Découvertes ; part des matchs et profil des matchs en cause : à mesurer sur le parc, attend le signal. Tests : `skill_v2_solver_fallback_tally_test.go` (36 replis, 0 WARN avant la synthèse, 1 après, 36 DEBUG, compteur +36).
  - 1.3 [x] hors mesure. Capacité `CompactPassesAtStartup` (vraie sur desktop seul, ADR 0044), `cmd/server/boot_compaction.go` : après les migrations de tous les titres, avant l'ouverture des bases partagées ; mesure de la jauge en lecture seule, rien sous `migration.SeuilPassesParMatch` ; sinon `migration.BackupBeforeUpgrade` (copie du fichier fermé, refus = pas de compaction) puis `migration.CompactSupersededPasses` (mêmes primitives que `levelup compact-passes`, aucune copie du code de compaction ; l'orchestration diffère : slog, budget de 75 s sous le `ready` de 120 s de la coquille). Échec = ERROR et démarrage normal. Durée et tables loguées. Message de `data_health` : « compaction attendue au prochain démarrage » quand la capacité est vraie, « de nuit » sinon. ADR 0026 et ADR 0045 amendés, commentaires « jamais au boot » corrigés (`compaction.go`, `cmd_compact_passes.go`). Pas d'événement stdout ajouté : le contrat de l'ADR 0045 ne prévoit pas de phase de maintenance (Découvertes). Tests : `boot_compaction_test.go` (base réelle à 6 lignes brutes pour 3 gardées : copie puis compaction, 6 vers 3 ; sous le seuil rien ; copie refusée, pas de compaction ; capacité absente ou démo : jamais appelée ; échec simulé : tous les titres traités), `data_health_passes_message_test.go`, `deployment_test.go` (capacité par profil). Durée MESURÉE sur une copie de la copie à froid (serveur de la branche en `-mode desktop`, 4 joueurs suivis) : halo_infinite 19 tables au-dessus du seuil sur 20 (4,20 millions de lignes brutes vers 1,88 million), copie préalable de 485 Mio comprise 8,8 s ; halo_5 1 table (`match_kill_events`, 536 674 vers 268 337 lignes) 2,3 s ; au total environ 11 s, contre 75 s de budget ; événements `listening` puis `ready` reçus normalement. `levelup compact-passes --dry-run` sur la même copie : 2,7 s. Le budget de 75 s laisse donc une marge de sept fois sur ces bases.
  - 1.4 [x] aucun lecteur ne lit `streak` : `streaks_repo.go:38` lit `streak_latest`, l'écriture est un INSERT dans `streak_history` (`:78`), `progression_diag_repo.go:42` compte `streak_latest` ; la seule autre mention SQL de `streak` est la recopie de `create_streak_history_append_only`. Cause du refus répété : `create_streak_history_append_only` (ordre 96 de `canonicalOrder`) passe AVANT `create_progression_player_schema` (ordre 127), qui recréait une `streak` vide sur toute base neuve, et la vue `streak_latest` la faisait prendre pour append-only par le réparateur. Corrigé : la création de `streak` (et ses deux index) est retirée de `create_progression_player_schema` ; step `drop_streak_legacy_v1` (ordre 97, `steps_player_streak_drop.go`) supprime la table des bases existantes seulement si chaque `id` est déjà dans `streak_history` (sinon la laisse et journalise `streak héritée conservée` avec le nombre de lignes non recopiées) ; `canonicalOrderFloor` 237 vers 238. Tests (`player_streak_drop_test.go`, `-tags=integration`) : base neuve sans `streak` ni index ; base existante recopiée, table supprimée, `streak_history` intacte (2 lignes), aucune ligne `réparation refusée` ; ligne non recopiée, table conservée et WARN ; tests existants adaptés (`order_audit_test`, `steps_player_test` halo_5, `declared_keys_repair_test`).
  - 1.5 [x] code et données (mesuré sur la copie à froid). Cause par le code : `match_participants.accuracy` est stockée en POURCENTAGE (0 à 100, valeur native de l'API, `sync/transforms.go:343-351` ; `compare_repo.go:109` divise par 100 ; `player_matches_projection.go:292` le dit), alors que `PostSyncProgressionStore.MatchRows` la passait telle quelle aux records, dont les bornes sont en ratio 0 à 1 (ADR 0006) : TOUTE précision était refusée, 61,73 étant simplement la meilleure précision de la fenêtre (trois lignes : all_time, 90d, 30d). Sur la copie : 29 091 participants, la précision vaut 100 x tirs touchés / tirs lancés (match 346d54ba-93ea-4642-b21b-b1a612181821, xuid 2535436454772398 : 213 / 345 = 61,73 ; match 846d608c-f829-4a09-8238-49ab8ad9e75b, Madina97294 : 345 / 619 = 55,73) ; minimum 0, maximum 120, deux valeurs au-dessus de 100 (120 pour 6 tirs touchés sur 5 lancés, 104 pour 26 sur 25 : anomalies de l'API, non touchées) ; les neuf valeurs au plus égales à 1 sont de vraies précisions de 0,4 à 0,9 %. Les autres sources (import OpenSpartan, même charge utile de l'API ; Halo 5, colonne absente) suivent la même convention : la colonne n'est pas corrompue, seule la lecture des records l'était. MAIS les records de précision déjà rangés l'étaient en pourcentage (shared_social : 15 lignes de `player_records_history` dont les 11 dernières valeurs, de 37,12 à 73,33 ; `record_history` des joueurs : même échelle) : après la correction de la lecture, une précision de 0,73 ne battrait jamais 73,3 et aucun record ne se poserait. Migration `records_accuracy_to_ratio_v1` (INSERT seul, ADR 0026) : une ligne neuve en ratio par dernier record de précision compris entre 1 et 100, les autres valeurs et métriques intactes ; test `TestAccuracyRecordsToRatio` (`-tags=integration`) ; `canonicalOrderFloor` 239. `record_history` (journal par joueur) n'est lue par aucun lecteur de page : non touchée. Corrigé à la lecture : `COALESCE(mp.accuracy, 0) / 100.0` (`post_sync_progression_store.go`). Test : `TestPostSyncProgressionStore_PrecisionEnRatio` (61,73 donne 0,6173 et passe `records.IsPlausibleValue`) ; fixture passée en pourcentage. Requêtes de contrôle après la migration, sur une copie : `SELECT metric, period, value FROM player_records_latest WHERE metric = 'accuracy'` dans `shared_social.duckdb` (attendu : toutes les valeurs entre 0 et 1) et `SELECT MIN(accuracy), MAX(accuracy) FROM match_participants` dans `shared_matches_v2.duckdb` (inchangé : 0 et 120). Voisin non corrigé, en Découvertes : le jalon `accuracy_threshold_days`.
  - 1.6 [!] partiel. Cause du bail annulé, par le code : `MatchRecomputer.recompute` (`sync/match_recomputer.go`) tient le bail `KindPlayer` pendant TOUT le rejeu du joueur (notes de performance puis LUSR sur tout l'historique : minutes) ; l'instantané de carrière (`kickoffBackgroundRefresh`) attendait ce bail sous le contexte de `careerLiveBgTimeout` (30 s), déjà consommé par les appels à l'API : `AcquireWriterCtx` rend `context deadline exceeded` et l'instantané est perdu. Corrigé : l'écriture a son propre contexte, détaché de celui des appels (`context.WithoutCancel`), borné à `careerLivePersistWait` (5 min) ; test `TestCareerLive_PersistAttendLEcrivainAuDelaDuBudgetDeFetch`. Non fait : « connexion fermée sous un lecteur » (`db_recovery.go:180`). Le mécanisme par le code est connu et déjà décrit par `read_recovery.go` (un emprunt via `OpenReadForQuery` n'a pas de référence, le propriétaire ferme le `*sql.DB` dessous : fin d'un `OpenPlayerDB` propriétaire unique comme celui de `recompute`, `Reopen()`, B-swap) ; la doctrine écrite refuse l'emprunt possédant à cause du B-swap, qui ne concerne que les chemins d'un fournisseur. Je n'ai pas pu établir par le code LAQUELLE de ces fermetures a produit la ligne du journal (sans journal détaillé ni exécution) et je n'ai pas changé la sémantique d'`OpenReadForQuery` sans mesure ; proposition en Découvertes.
  - 1.7 [x] `world_leaderboard_cron.go` : une ligne de synthèse par passe (`scrape de playlists échoué` : nombre, liste des playlists, statuts HTTP, première cause), WARN quand tous les échecs sont des 5xx, ERROR dès qu'un échec n'en est pas un ; plus d'ERROR par playlist. `catalog_expand: terminé avec échecs` porte maintenant `statuts_http` et `cause` (le détail par playlist reste en DEBUG). Lecture du statut : nouveau petit paquet `internal/util/httpstatus` (les clients Halo citent « HTTP 500 » dans leur message ; les couches ne les importent pas). Tests : `world_leaderboard_failures_test.go`, `TestPlaylistExpansion_SyntheseDitLaCauseEtLeStatut`, `httpstatus_test.go`.
  - 1.8 [x] `pool.On429ForToken` : INFO + compteur expvar `levelup.auth_pool.per_token_429_absorbed_total` ; WARN seulement au-delà de `perToken429WarnCooldown` (30 s). `match_poller` : un 429 est un INFO compté (`match_poller_429_total`, retenté au poll suivant), les autres erreurs restent des WARN. Tests : `pool_429_log_test.go`, `match_poller_429_test.go`.
  - 1.4b et 1.5b [x] `gestes_prod_8_2_0.md` : G4b (sauvegarde habituelle du VPS, démarrage, contrôle du journal : plus de refus `streak`, recalcul des notes terminé pour chaque joueur) et G4c (précision) ajoutés ; aucune copie de base locale vers le VPS.
  - 1.9 [x] témoin avant/après. Deux copies à froid distinctes de la copie du pilote (4 joueurs suivis, `scoring_revision` remise à une valeur ancienne pour forcer le recalcul des notes de chacun au démarrage), serveur lancé en profil dev, ressources et données pointées sur les copies (`LEVELUP_RESOURCES_DIR`, `LEVELUP_DATA_DIR`, `LEVELUP_LOGS_DIR`), ports 18771 et 18772, 7 min chacun : AVANT = binaire de `origin/release/v8.2.0` (709b4438d0), APRÈS = binaire de la branche. WARN et ERROR du journal de démarrage, comptés par message : `sharedprovider: writer RW tenu au-delà du seuil` 163 vers 0 (remplacé par 1 ligne de dépassement et 1 de libération) ; `LUSR v2: solveur non convergé` 33 vers 0 (remplacé par 3 lignes de synthèse, une par joueur concerné) ; `migration: clé déclarée absente, réparation refusée ... streak` 1 vers 0 ; total WARN et ERROR 230 vers 38. Recalcul des notes terminé pour les 4 joueurs dans les deux cas (Chocoboflor 566 notes de performance et 596 LUSR, JGtm 1 134 et 1 171, Madina97294 1 192 et 1 207, XxDaemonGamerxX 16 et 38). Restent, identiques avant et après et normaux sur une copie sans jeton : `data_health: passes supersédées ... compaction de nuit attendue` (19 lignes, une par table, profil dev : le profil desktop compacte au démarrage, mesuré ci-dessus), `spartan_cron`, `world-enrich`, `auto_sync` (aucun jeton), `catalog_refresh_cron` (aucun jeton valide : ERROR dit la cause), `csr_season_id absent`, `disk_watch`. Ni `dblease ctx cancelled`, ni `database is closed`, ni `records: valeur hors bornes`, ni `HTTP 429` n'apparaissent dans les deux journaux (copie sans jeton, pas de passage `career_live`, pas de post-sync) : ces trois lignes ne se rejouent pas sur une copie ; 1.6, 1.5 (côté post-sync) et 1.8 sont couverts par leurs tests.
