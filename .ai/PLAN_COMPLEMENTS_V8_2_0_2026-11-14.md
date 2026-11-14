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
- [ ] 1.1 `platform/duckdb/sharedprovider/provider_writer.go:217` et `reader_inflight.go:101` : une ligne au franchissement du seuil, une ligne à la libération (durée, lectures retenues), plus de répétition périodique ; test de comptage des lignes.
- [ ] 1.2 `sync/skill/skill_v2_solver_fallback.go:61` : compteur par rejeu, une ligne de synthèse en fin de rejeu (matchs, replis). Instruire la non-convergence de l'EP (part des matchs, profil des matchs en cause) ; correction seulement si la cause est établie, et alors dans la révision de l'étape 7.
- [ ] 1.3 `scheduler/data_health_passes.go:102` + `migration/compaction.go` : profil bureau, compaction au démarrage avant ouverture si au-dessus du seuil (D-12), durée mesurée sur copie, événement de démarrage visible par la coquille ; message sans « de nuit » hors VPS. ADR 0045 amendé.
- [ ] 1.4 Table héritée `streak` : vérifier qu'aucun lecteur ne la lit (`streaks_repo.go:117` lit-il `streak` ou `streak_latest` ?), retirer sa création (`steps_player_base.go:228`) et la supprimer par migration après la recopie de `steps_player.go:201` ; plus de refus de réparation au démarrage.
- [ ] 1.5 Précision en 0..100 (`records`, valeur 61,73) : trouver le match et la source d'écriture, corriger l'échelle à l'ingestion, réparer les lignes touchées par la voie append-only ; le record se pose.
- [ ] 1.6 Bail joueur annulé pendant le recalcul (`dblease/writer.go:198`, `career_live_service.go:460`, `db_recovery.go:180`) : cause établie ; l'écriture partielle de carrière n'est plus perdue (attente ou reprise) ; plus de connexion fermée sous un lecteur.
- [ ] 1.7 `world_leaderboard_cron.go:290,415`, `playlist_expansion_service.go:78` : une erreur 5xx du service Halo = une ligne de synthèse par passe (playlists en échec, statut), niveau WARN.
- [ ] 1.8 `pool/pool.go:642`, `watcher/match_poller.go:104` : 429 absorbé par le retrait = INFO compté ; WARN seulement si le retrait échoue ou dure.
- [ ] 1.4b et 1.5b Skill `db-schema` appliqué ; migrations au registre (démarrage), tests sur base joueur legacy et neuve ; séquence prod ajoutée à `.ai/annexes_v8_1_2/gestes_prod_8_2_0.md` (D-15).
- [ ] 1.9 Gate : témoin 0.3 rejoué, comparaison ligne à ligne ; restent seulement les replis nommés du décodeur et les cartes non résolues (normaux, D-10 ADR 0034).

### Étape 2 — Tendances, groupe « Rythme »
- [x] 2.1 `analysis/trends/indicators.go` : frags, morts, assistances, dégâts infligés et subis par minute (`Match.Seconds`), rapport des sommes, `inMatrix`, sens (morts : bas = mieux) ; groupe et clés de domaine ; vue escouade (`[~]` : la vue escouade ne porte aucun indicateur de combat par membre, seulement FDA et part des frags ; un allié n'a ni secondes jouées ni dégâts dans `MemberStat` ; le groupe est absent de cette vue, test à l'appui). Code : `indicators_pace.go`, `TrendsGroupPace`, `TrendsUnitPerMin`, cinq clés `*_per_minute`.
- [x] 2.2 Libellés FR/EN (manifeste `tendances`), unités ; tests Go et web ; `make generate-types` si le contrat change (inchangé : `group`, `unit` et `key` sont des chaînes libres dans `openapi.yaml`). Deux graphiques « Rythme » ajoutés à la grille « Niveau, combat et style » de l'Évolution.

### Étape 3 — Placement A1
- [ ] 3.1 `features/squad/emprise/placementCharts.ts` : carte de chaleur (cases distance × frags, quarts, repère radar, médianes par joueur), infobulle de case (part, compte).
- [ ] 3.2 Vocabulaire D-3 dans `placementStrings.ts` (FR/EN) et garde-rails de textes ; « Part des vies par placement » inchangée.

### Étape 4 — Rôles de portée B1
- [ ] 4.1 `SquadRangeRolesCard` + `squadRangeRolesChart.ts` : ruban joueur × match, rôle de la fenêtre glissante, case vide avant `FENETRE_ROLE`, échantillon selon Q-2 ; logique pure inchangée (`squadRangeRoles.logic.ts`). Résumé solo inchangé.

### Étape 5 — Infobulles
- [x] 5.1 Couche globale (`components/ui/`) montée dans la coquille : survol et focus clavier d'un `[title]`, titre déplacé en attribut de données pendant l'affichage, `aria-describedby`, style de `tooltip.tsx`, position bornée à la fenêtre ; SVG `<title>` (2) traités.
- [x] 5.2 `info-tooltip.tsx` : focus filtré au clavier (entrée du backlog) ; garde-rail de couleurs étendu (`ring-offset`, `outline`, `decoration`, `accent`, `caret`, portée de `color-allow`).
- [ ] 5.3 Contrôle : partage de fenêtre Discord par le user (artefact des vérifications).

### Étape 6 — Rang de session C4
- [ ] 6.1 `squadPerformanceLineCharts.ts` (`buildTeamMMROption`) : un bandeau par joueur, même amplitude (plus grand écart de la session + marge), triangles de sens, rang final et écart à gauche ; MMR d'équipe en bandeau séparé ; `hexComplement` retiré de ce graphe.
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
- [ ] 9.1 `SquadAppuiCard.tsx` en flux assistant → finisseur (D-10), même forme sur la page Match (`MatchAssistChart`) ; textes FR/EN à la voix active ; garde-rail `vocabulaireCoordination.guard.test.ts` ajusté ; infobulles de ruban et de joueur.

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

### Journal

- 2026-11-14 : plan relu par la grille `plan-review` (gates, reprise, multi-titre ajoutés) ; voies `escouade`,
  `infobulles`, `rejeu` ouvertes.
- 2026-11-14 (voie infobulles, branche `feat/c820-infobulles`) : 5.1 et 5.2 faits. Couche `TitleTooltipLayer` montée dans `AppProviders` (couvre connexion et pages hors coquille) ; `Tooltip` et `InfoTooltip` marqués `data-lu-tooltip` (pas de double infobulle) ; `InfoTooltip` ouvert au focus clavier seulement ; lint des couleurs étendu (cinq préfixes) et `color-allow(<genre>)` borné au genre nommé (76 lignes migrées, aucune allowlist élargie). 5.3 reste au user.
- 2026-11-14 : voie rejeu (étape 8) faite sur `feat/c820-rejeu` : trois calques « Vies » (apparitions, morts, tueurs), seize calques, affinages qui / quand, préréglages D-9, textes FR/EN, aucun changement du document de rejeu (D-13). Contrôle à l'écran remis au user.
- 2026-11-14 (voie `tendances`, branche `feat/c820-tendances`) : étape 2 faite. Groupe `pace` (« Rythme ») de la matrice et deux graphiques de l'Évolution ; agrégat somme des comptes / somme des minutes ; match sans seconde jouée exclu des deux termes ; dégâts absents (nil) = ligne absente, pas zéro. Vue escouade : non concernée (voir 2.1).
