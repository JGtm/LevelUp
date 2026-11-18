# PLAN — Entrées du backlog des 17 et 18 novembre, et deux retours du user sur le rejeu (2026-11-18)

Statut : **écrit le 2026-11-18, relu contre la grille `plan-review`. Réponses du user du 2026-11-18 : D-2,
D-3, D-4, D-5 et D-7 retenues telles que recommandées ; D-1 rouverte (la silhouette de l'arme est écartée,
choix d'un tracé sur planche) ; D-6 élargie ; et une correction de périmètre : « ma requête était justement
de les traiter », les entrées que la première version laissait au backlog entrent dans le plan (section 1.3).
Seconde série de réponses le 2026-11-18 : D-1 tranchée (flèche qui monte du sol pour les armes, caisse pour
les équipements) ; D-8 tranchée (plan du décodeur à part, version visée 8.3.0, piloté par cette session avec
ce plan-ci : `.ai/PLAN_DECODEUR_2026-11-18.md`) ; fenêtre du clic après match ramenée à 5 minutes. Toutes
les décisions sont prises. GO D'EXÉCUTION DONNÉ par le user le 2026-11-18 (« ok avec tes recos, go pour les
deux plans »).**
Contrat d'exécution : skill `plan-execution` (ordre strict dans une piste, aucun report, statut de chaque item,
vérification sur pièces, zéro correctif hors périmètre). Statuts : `[x]` fait, `[~]` couvert ailleurs
(référence), `[!]` non traité (motif écrit). Aucune case vide à la clôture.

Demande du user (2026-11-18) : analyser les entrées du backlog ajoutées la veille et le jour même, les
instruire sur preuve, garder ce qui est pertinent et faisable, planifier en pistes parallèles (Sonnet par
défaut, Haiku quand c'est mécanique, Opus quand il le faut, Fable pour la revue finale), périmètre fermé,
découvertes au backlog seulement si elles valent le coup. Deux retours ajoutés par le user : le pictogramme
d'une ligne de ramassage du fil du rejeu lui déplaît ; sur le match `38900a5e`, à 5:00, ELOsk11 est donné mort
alors qu'il quitte la partie.

Abréviation : `MR/` = `apps/web/src/features/match-replay/`.

## 0. Objectif et critère de succès

Objectif : régler ce qui se règle maintenant dans les entrées des 17 et 18 novembre, corriger les deux retours
du user, et donner un sort écrit à chaque entrée instruite.

Critère de succès : chaque entrée de la section 1 a un sort écrit (traitée, close, reformulée, gardée avec sa
condition) ; sur le match `38900a5e`, ELOsk11 n'a plus de ligne « mort » à 5:00 et sa fiche dit qu'il a quitté
la partie ; `make gate-push` vert ; suite complète du web verte ; CI verte sur `release/v8.3.0`.

| Piste | Ce que le user verra |
|---|---|
| A | Rejeu : un joueur qui quitte la partie n'est plus donné mort (fil et fiche) ; une ligne de ramassage porte le tracé choisi sur la planche (D-1) |
| M | Rejeu : la barre de lecture d'une ligne reste lisible sur une fenêtre étroite ; au Plein cadre, barre en surimpression (les deux sur planche validée) |
| Q | Rejeu : les écarts non voulus avec les maquettes sont corrigés |
| T | Un seul coup d'envoi par match, le même sur toutes les pages : temps de jeu, taux par minute, Dominance et rejeu s'accordent (après le rattrapage, geste du user) |
| B | Rejeu : « Seul » et « Suivre » ne se contredisent plus ; le point de vue repart de zéro à chaque match ; le plein écran du navigateur se quitte avec le Plein cadre |
| C | Rejeu : la rangée de fiches ne clignote plus à certaines largeurs ; l'orientation Auto ne dépend plus du match précédent ni d'un redimensionnement pendant un export ; l'horloge de la barre ne se vide plus en pause ; Tactique et « Occupation du terrain » suivent l'orientation Auto (selon D-3) |
| D | Fiche du joueur : réapparitions justes en début de match, une ligne par arme dans « Tirs et dégâts », nom de grenade de repli traduit |
| E | Rien à l'écran ; le téléchargement des films et du catalogue n'emploie plus le jeton d'un joueur qui n'a pas accepté la mise en commun |
| F | Bouton « Générer le rejeu » : n'est plus proposé quand il échouera ; bureau : les vieux matchs gardent leur place dans la Tactique après la mise à jour (selon D-5) |
| G | Escouade : un coéquipier sans compte connu a un trou, pas un zéro (selon D-4) ; Dominance découpée sur la durée lue dans le film ; le reste est interne |
| H | Échap ferme de la même façon les menus de la barre de navigation et les boîtes de dialogue ; menus annoncés aux lecteurs d'écran |
| K, S | Rien (garde-rails, outillage, schéma des bases neuves, tests du paquet DuckDB) |

Base : `release/v8.3.0` (`e95769096a`). Branche : `chore/830-backlog-1118`, worktree du pilote `C:/t/bl1118`
(sans jonction vers `data/`). Une seule branche, un commit par item ou par groupe d'items d'une piste, fusion
dans `release/v8.3.0` à la fin, sans PR par étape. Jamais `main`. Version : 8.3.0. Aucune montée de brique,
aucune régénération de rejeu.

## 1. Instruction : ce que les pièces disent (2026-11-18)

Trois instructions en lecture seule (agents Sonnet) sur `release/v8.3.0` à `e95769096a`, recoupées par le
pilote sur une quinzaine de références ; le rejeu du match `38900a5e` lu champ par champ ; une mesure sur les
1 455 rejeux du poste (`.ai/annexes_backlog_1118/mesure_departs.md`). L'exécutant RE-VÉRIFIE chaque référence
avant de coder (étape 0.4).

### 1.1 Les deux retours du user

| Retour | Constat sur pièces | Sort |
|---|---|---|
| ELOsk11 « mort » à 5:00 (match `38900a5e`) | Le décodeur lit le départ : mort à 4:41 (image 9539, compte à rebours lu), réapparition à l'image 9783, corps SUPPRIMÉ à l'image 9785 (`endReason: "deleted"`, aucun lâcher, aucun compte à rebours), présence au tableau fermée à l'image 9937, place reprise par le bot 343 Mickey à 9938. Le site n'en tire rien : toute fin de vie close sans frag devient une ligne « mort » (`MR/model/killFeedLogic.ts:300-302` et `:469-471`), `MR/model/lifeEndProof.ts:9` range `deleted` sous « fin de manche », et la fiche lit « mort » dès qu'aucune vie ne couvre l'image (`apps/web/src/lib/replay/rosterLogic.ts:538-554`). Seule la croix sur la carte est évitée. Parc : 1 386 départs en cours de match dans 729 rejeux ; 875 finissent par un corps supprimé (le schéma d'ELOsk11), 503 par une vraie mort suivie du départ, 8 sans raison publiée ; 5 rejeux ne publient pas de présence | Piste A (A.1), côté site seulement |
| Pictogramme d'une ligne de ramassage | Tracé au trait générique (`MR/model/feedGlyphs.ts`, clés `pickup`, `pickupPower`, `pickupEquipment`), choisi par la nature de la prise (`MR/ui/ReplayEventLine.tsx:61`). La ligne connaît l'arme, et le document porte son dessin (`weaponLabels[clé].img`, `tinted`), déjà employé par une ligne d'élimination (`FeedWeaponIcon`) | Piste A (A.2), selon D-1 |

### 1.2 Entrées retenues

| Entrée du backlog | Constat sur pièces | Sort |
|---|---|---|
| « Suivre » un joueur et « Seul » sur un autre | Confirmé dans les deux sens (`MR/model/playerPanelLogic.ts:104` et `:79-84`) : la carte ne dessine que Y et se cadre sur X | B.1 (D-2) |
| Point de vue gardé d'un match à l'autre | Confirmé : état sans portée (`MR/hooks/useReplayViewpoint.ts:38`), alors que le cadre et la fiche se remettent à zéro au changement de match | B.2 |
| Plein écran du navigateur laissé actif | Confirmé, deux scénarios (`MR/hooks/useReplayFullscreen.ts:156-158` ; refus du natif : `:95-101`) | B.3 |
| Rangée de fiches qui oscille | Confirmé par le calcul : comparaison sans mémoire (`MR/model/rowLayout.ts:65`), bande large comme la barre de défilement (1286 à environ 1302 px en 4 contre 4) ; même défaut pour le score en-tête / toile (`:132`, seuil 820 px) | C.1 |
| Soupçon : orientation Auto qui dépend du match précédent | Démontré (`MR/hooks/useResolvedTurns.ts:41-49`), quand le document et le contexte du match suivant sont déjà en cache | C.2 |
| Soupçon : Auto pendant un export | Démontré : un redimensionnement de fenêtre pendant l'export peut retourner le plan (`MR/hooks/useReplayView.ts:179`) | C.2 |
| Soupçon : horloge de la barre vide en pause | Mécanisme établi (texte écrit par le seul dessin, `MR/hooks/useReplayClock.ts:80-85`) ; tranché par un test | C.3 |
| Tactique et « Occupation du terrain » sans l'orientation Auto | Confirmé et voulu par le code actuel (`settings/useMapViewPreferences.ts:22-44`). La Tactique mesure déjà sa zone ; « Occupation du terrain » non | C.4 (D-3) |
| Soupçon : réapparitions comptant une vie d'avant le match | Démontré (`MR/model/playerInstantStats.ts:320` et `:379`) | D.1 |
| Clé de liste par nom d'arme | Confirmé possible : plusieurs familles d'arme du film partagent un nom (Bandit, épée à énergie, fusil à choc, bobine : `apps/go-api/internal/games/weapons/registry.go:653-706`) ; le joueur verrait deux lignes au même nom | D.2 |
| Nom de grenade « rang N » en français | Confirmé (`MR/model/inventoryReading.ts:326`), deux appelants | D.3 |
| Cinq cotes de l'ancien gabarit sans lecteur | Confirmé et plus large : onze champs de `GABARIT_NORMAL` sans lecteur, seul `bodyPx` vit ; deux branches mortes dans `MR/ui/ReplayCountersBadge.tsx` | D.4 |
| Soupçon : lâcher daté à fin de vie + 2 | Démontré par lecture (`MR/model/equipmentGestures.ts:115`) ; fréquence inconnue | D.5, sous condition de la mesure 0.3 |
| La fiche relit tout le lobby à chaque changement de joueur | Remontage confirmé (`MR/ui/frames/ReplayCinemaFrame.tsx:91`) ; coût non mesuré | D.6, sous condition de la mesure 0.3 |
| Jetons : `haloTokensForDrain` | Confirmé et en production : `apps/go-api/internal/api/wire/registry_catalog_drain.go:151-170` prend le premier profil dont le jeton se rafraîchit, sans passer par la mise en commun ; appelé par le catalogue et par la mise en file des rejeux après chaque synchronisation (`registry_build_queue.go:159`). Le jeton d'un joueur sert donc au film d'un autre, sans son accord (ADR 0038). La source conforme existe (`registry_pool_source.go:61-73`). Le conseil de recapture de `refreshTokensFromDB` vise l'absence de jeton (premier enrôlement) : libellé à reformuler seulement | E.1 |
| Rejeu à la demande : restes de la relecture | Cinq vrais défauts mineurs (demande reprise non revalidée, nouvel essai qui saute les contrôles, film laissé sur le disque après échec, bouton offert alors que l'action échouera, jeton refusé gardé en cache), deux théoriques à fermer à bas coût (garde en liste noire, capacité « développement = non hébergé ») | F.1 à F.7 |
| Régénération du bureau : fichier tactique au film expiré | Confirmé (`internal/replayregen/scan.go:77`). Le rattrapage existant ne couvre que les 64 matchs les plus récents ; au-delà, le match devient « non mesuré » dans la Tactique. Une reprojection sans film existe déjà | F.8 (D-5) |
| Événements de médaille : trois champs servis sans lecteur | Confirmé (`internal/service/match_view_killfeed_medals.go:32-74`, contrat `MatchHighlightEvent`) ; `medal_name_id` de l'événement n'a pas de lecteur non plus ; `MedalImageURL` des adaptateurs de titre devient sans appelant | G.1 |
| Escouade : taux par minute à 0 au lieu d'inconnu | Confirmé (`internal/service/teammates/teammates_squad_per_minute.go:65-71`) ; le graphe sait déjà dessiner un trou | G.2 (D-4) |
| Soupçon : difficulté d'une médaille prise à la première langue | Confirmé comme défaut de code (parcours d'une table sans ordre, `internal/service/replay_medal_labels.go:149-163`) | G.3 |
| Compteur des rejeux illisibles | Confirmé (`internal/replayartifact/gate.go:97-126`) | G.4 |
| Coup d'envoi de l'API contre celui du film | Le coup d'envoi se lit partout dans le registre, donc dans le film depuis le 2026-11-18. Restent sur l'estimation de l'API : la durée du découpage de la Dominance (`internal/service/match_view_data_loaders.go:243-256`), le temps de jeu des participants et la durée jouable du registre (écrits à la synchronisation) | G.5 pour la Dominance ; le reste reformulé (D-6) |
| Garde-rail des couches : périmètre | Confirmé : 13 composants de `components/` (14 sites) écoutent à la main, 4 cas légitimes ; `lib/` n'en a aucun | H.1, H.2 |
| Accessibilité des menus | Confirmé pour les trois composants | H.3 |
| Clé de requête du contexte du rejeu sans la langue | Confirmé : la réponse porte encore des libellés traduits (carte, mode, playlist, rang, équipe) | H.4 |
| Contrôle des couleurs : entrées mortes | Confirmé, trois et non deux (`lib/accessibility/_palettes.ts` aussi) | K.1 |
| Rempart contre les chemins du poste | Fond publié propre ; cas de bord réel dans le motif (`apps/go-api/cmd/mapfond-build/degradations.go:14-36`) | K.2 |
| `scripts/disk-hygiene.ps1` | Confirmé (lignes 280 et 288) | K.3 |
| Test instable sous charge | Aucune course trouvée ; la sortie d'erreur du processus fils est perdue | K.4 (diagnostic seul) |
| `bootstrap_service.go` au-delà du seuil | Confirmé, 690 lignes ; trois blocs sortent sans renommage ; trois garde-rails indexés sur le chemin | K.5 |
| Schéma d'une base joueur selon l'ordre de création | Confirmé à la définition (`performance_score` en `FLOAT` côté synchronisation, `DOUBLE` côté migrations) ; les 37 lignes ne sont archivées nulle part ; l'invariant existant ne compare que des noms ; les deux ordres existent en production pour une base neuve | Piste S |

### 1.3 Entrées que la première version du plan laissait au backlog, reprises à la demande du user

La première version écartait neuf entrées (gain faible, défaut latent, maquette ou mesure manquante). Le user
a corrigé le 2026-11-18 : sa demande est de les traiter. Chacune a maintenant son item ; ce qui dépend d'un
geste du user (maquette à valider, écriture en base, version du décodeur) porte ce geste comme porte.

| Entrée | Constat | Sort |
|---|---|---|
| Paquet DuckDB : migrations neuves indirectes | Onze sites ; cinq partent d'un fichier joueur absent et passent par la base migrée copiée ; gain estimé de 5 à 15 s ; le garde-rail se contourne par une valeur de fonction | S.4 |
| Médailles du fil pour un titre à planches d'images | Confirmé dans le code ; un composant sait déjà dessiner une médaille depuis une planche (`components/ui/MedalIcon.tsx`) ; aucun titre à planches n'a de rejeu aujourd'hui | A.3 |
| Soupçon : deux scores « à l'instant » | Deux sources distinctes (calque de score pour l'infobulle de la tuile, points par action gardés pour la fiche) ; divergence à mesurer | Mesure 0.3 b, puis D.7 |
| Soupçon : barre d'une ligne sur écran étroit | Probable par le calcul : en disposition colonne, la frise tombe vers 14 px de large à 1 440 px de fenêtre | Piste M (planche, puis code) |
| Barre de lecture en surimpression | Idée du user, qui demande une maquette avant tout code | Piste M (même planche, puis code) |
| Écarts avec les maquettes (onze points) | Conformité non relue ligne à ligne ; les maquettes et la grille sont au dépôt | Piste Q (instruction contre les maquettes, puis corrections) |
| Rejeu à la demande : clic juste après la fin d'un match | Le risque ne se mesure pas sans un match réel ; il se ferme sans mesure : pas de verdict « film absent » définitif pour un match qui vient de finir | F.9 |
| Coup d'envoi : temps de jeu des participants et durée jouable du registre | Écrits à la synchronisation sur l'estimation de l'API, jamais recalculés quand le coup d'envoi du film arrive ; une commande de rattrapage existe (`cmd/backfill_time_played`) ; la note LUSR lit le temps de jeu (`internal/sync/skill`) | Piste T (D-6) |
| `medalLabels` sans jumeau dans le décodeur, et les entrées `[décodeur]` (verdict du score en Assaut et en Grande bataille, tirs et touches, commandes et gestes, répliques, temps forts de type « mode », lettres des zones, ordre des collines, porteurs du drapeau, arme d'une touche, cas de bord des lâchers, retrait de l'ancienne précision par film) | Toutes demandent une version du décodeur, donc une seconde montée et une seconde régénération du parc dans la 8.3.0, ou une version suivante | D-8 |

## 2. Décisions

### Tranché par le user le 2026-11-18

| Décision | Réponse du user | Effet |
|---|---|---|
| D-2 — « Seul » et « Suivre » | « ça a du sens, ok » | « Seul » sur un joueur arrête le suivi d'un autre, et suivre un autre joueur lève « Seul » (B.1) |
| D-3 — orientation Auto hors du rejeu | « ok » | La Tactique et « Occupation du terrain » appliquent la règle du rejeu, chacune sur sa propre zone d'affichage (C.4) |
| D-4 — Escouade, taux par minute | « ok » | Un coéquipier sans aucun compte connu a un trou, pas un zéro ; trois champs du contrat deviennent facultatifs (G.2) |
| D-5 — bureau, vieux matchs dans la Tactique | « ok » | La file de régénération refait, sans film, le fichier tactique des rejeux au film expiré (F.8) |
| D-6 — durées dérivées du coup d'envoi | « voir ma question plus haut » : ne rien laisser au backlog | La Dominance (G.5) ET le temps de jeu des participants et la durée jouable du registre (piste T) |
| D-7 — médailles des événements du match | « ok » | Retrait des trois champs sans lecteur ; `medal_name_id` de l'événement selon l'inventaire de G.1 |
| Périmètre | « Pourquoi tu as laissé les trucs au backlog ? ma requête était justement de les traiter » | Section 1.3 : chaque entrée écartée a un item |

| D-1 — pictogramme d'une ligne de ramassage | Silhouette de l'arme écartée (« ça risque d'être confusant ») ; puis, sur la planche (https://claude.ai/artifact/FhyKN9Q88cQaebNFFkScPA) : « j'aime bien l'option B pour arme de râtelier et arme spéciale. Option C des caisses ça me paraît mieux pour l'équipement et l'équipement + à la rigueur (sachant que normalement on sait quel équipement est game changer ou non) » | Quatre tracés : flèche qui monte du sol (arme de râtelier), la même et l'étoile (arme spéciale), caisse (équipement), caisse et étoile (équipement qui change la partie : surbouclier, camouflage) (A.2) |
| D-8 — entrées du décodeur | « on vise cette version mais ce qu'on peut faire c'est un plan à part oui, tu le traiteras avec plus de rigueur et de prudence. du coup tu piloteras le plan backlog et le plan décodeur » | Plan `.ai/PLAN_DECODEUR_2026-11-18.md`, version visée 8.3.0 (donc une seconde montée du décodeur et une seconde régénération du parc dans la 8.3.0, par exception à la règle d'une montée par version, décidée par le user) |
| Clic juste après la fin d'un match | « c'est quand même rapide donc je baisserai à 5 minutes si jamais on a pas moyen de détecter le manque autrement » | F.9 : chercher d'abord un signal du film pas encore publié ; à défaut, fenêtre de 5 minutes |
| Temps de jeu | « Normalement le temps de jeu on l'a avec T0 jusqu'à la fin du match transmise par l'API non ? » | Confirmé sur pièces (`analysis/timeline/compute_time_played.go`) : présence du joueur bornée par le coup d'envoi et la fin du match de l'API. Seul le coup d'envoi change ; la valeur rangée a été calculée avec l'ancien et n'est jamais recalculée (piste T) |

| Coup d'envoi | « Ok pour le coup d'envoi mais faut remettre au propre, qu'on ait pas plusieurs paramètres et de fallback, je veux un système cohérent » | Piste T réécrite : une définition, une résolution, une écriture, une commande ; modèle validé par le user avant le code (T.1) |

### À trancher par le user

Aucune décision ouverte avant le go. Une validation viendra en cours de route : le modèle du coup d'envoi
(T.1), présenté en termes de ce que les pages affichent.

### Tranché par le pilote (conventions du dépôt, modifiable par le user)

- Départ : la règle est celle de A.1 ; une vraie mort suivie d'un départ garde sa ligne « mort ».
- « Tirs et dégâts » : deux variantes d'une même arme font UNE ligne (tirs et dégâts additionnés).
- Jetons : sans aucun compte ayant accepté la mise en commun, rien n'est téléchargé et l'erreur est
  journalisée (ADR 0038 : aucune décision = refus).
- Couches : les 13 composants migrent vers le mécanisme commun dans le même lot que l'élargissement du
  garde-rail ; les 4 cas légitimes passent sur la liste d'exemptions datée.
- Schéma d'une base joueur : la forme de référence est celle que portent les bases existantes (mesurée en
  S.1) ; l'autre définition s'aligne ; aucune base existante n'est migrée.
- Marge de la rangée de fiches : 24 px entre les deux sens.
- Score à l'instant : s'il diverge, l'infobulle de la tuile lit la même valeur gardée que la fiche ; quand la
  garde refuse, l'infobulle ne montre pas de score.
- Clic juste après la fin d'un match : voir la réponse du user ci-dessus (signal du film d'abord, fenêtre de
  5 minutes à défaut).
- Écarts avec les maquettes : hors des écarts déjà décidés, la maquette fait foi (règle du plan maître) ; un
  point sans dessin va au user.

## 3. Règles d'exécution

- Une piste = un agent dans son worktree `C:/t/bl18-<piste>`, sous-branche `bl1118/<piste>`, SANS jonction
  vers `data/` ni vers `node_modules`. Les agents ne commitent pas, ne poussent pas, ne fusionnent pas. Cache
  Go dédié `go-build-bl1118-<piste>`, rendu à la clôture (`scripts/disk-hygiene.ps1 -Lot bl1118-<piste> -Apply`).
- Bornes : au plus 4 agents Sonnet d'écriture à la fois (les Haiku s'y ajoutent) ; un seul `make gate-push` ;
  la piste S seule quand elle passe ses tests d'intégration.
- Vagues : étape 0 (avec M.1, Q.1 et T.0, qui ne font que lire ou dessiner) ; vague 1 = A, C, E, G et
  l'item K.2 ; vague 2 = B, F, H, S ; vague 3 = D, le reste de K, Q.2, T ; vague 4 = M.2 et M.3, après la
  planche validée ; puis I,
  puis R. Dans une piste, l'item N est clos (gate passé) avant N+1.
- Portes du user : planche de la barre de lecture avant M.2 et M.3 ; liste
  des écarts à corriger avant Q.2 ; modèle du coup d'envoi validé avant T.2 ; copie de la base et serveur
  arrêté avant T.6. Une porte qui n'est pas
  passée ne bloque que ses items : le reste du plan s'intègre sans eux, et l'item garde sa case ouverte avec
  la date de la demande.
- Un agent ne lance que des tests ciblés, à deux processus (`--maxWorkers=2`). La suite complète du web est
  passée UNE fois par le pilote, sur la branche intégrée.
- Fichiers édités par le pilote seul : `apps/go-api/api/openapi.yaml`, `apps/web/src/lib/api/generated.ts`,
  `docs/CHANGELOG.md`, notes de version, `.ai/thought_log.md`, `.ai/BACKLOG.md`, `.ai/REGISTRE_REPORTS.md`.
- Fichiers à un seul écrivain, dans l'ordre : `apps/web/src/lib/replay/rosterLogic.ts` (A.1, puis D.5) ;
  `MR/i18n/i18nContract.ts`, `fr.ts`, `en.ts` (A.1 s'il lui faut un texte, puis D.3) ;
  `MR/hooks/useResolvedTurns.ts` et `apps/web/src/lib/replay/orientation.ts` (C.2, puis C.4) ;
  `internal/archlint/deployment_single_resolver_test.go` (K.5 seul) ;
  `internal/archlint/replay_generation_token_source_test.go` (F.1 seul) ; `internal/replayregen/` (piste F
  seule) ; `apps/web/src/lib/query/keys.ts` (H.4 seul) ; `apps/web/src/features/media/MediaPickerDialog.tsx`
  (H.3 seul, après H.2).
- Nœud avec la validation à l'écran de la refonte : la session qui la préparait est archivée (dit par le
  user le 2026-11-18), ses écritures sont dans `release/v8.3.0` ; le user n'a pas encore fait ses
  vérifications. Les corrections qui en sortiront arrivent à cette session : elles s'ajoutent à la piste Q
  en liste fermée, datée, avant leur code. Avant l'intégration, le pilote recale la branche sur la tête de
  `origin/release/v8.3.0`.
- Nœuds avec le plan du fond « Relief » (`.ai/PLAN_FONDS_RELIEF_2026-11-18.md`, session levelup-98, plan
  maître N22 à N26, reçus le 2026-11-18) : C.4 passe AVANT son item D.4 (mêmes fichiers :
  `useMapViewPreferences.ts`, `TacticalPlanCard.tsx`, `TacticalReapparitionsPlan.tsx`,
  `MatchPositionsHeatmap.tsx`) ; K.2 passe AVANT son item B.1 (`cmd/mapfond-build/`) ; ses items web du
  rejeu (`ReplayViewMenu.tsx`, `replayPreferences.ts`, textes du rejeu) passent APRÈS ceux de ce plan. Pour
  ne pas la faire attendre, la piste C et l'item K.2 partent dans la première vague. Le pilote prévient
  levelup-98 dès que C.4 et K.2 sont fusionnés. La borne de 4 agents d'écriture vaut pour tous les plans de
  la 8.3.0 ensemble : levelup-98 en tient 2 au 2026-11-18, ce plan et celui du décodeur se partagent les 2
  autres tant que c'est le cas, et le pilote lui annonce chaque lancement.
- Nœud avec le plan du décodeur (`.ai/PLAN_DECODEUR_2026-11-18.md`) : ce plan-ci ne change ni le décodeur ni
  le document de rejeu et s'intègre le premier ; la montée du décodeur et la régénération du parc
  appartiennent à l'autre plan. `MR/model/equipmentGestures.ts` (D.5) et la règle des départs (A.1) lisent
  le document tel qu'il est au schéma 125 : si le plan du décodeur change un champ qu'ils lisent, c'est lui
  qui les adapte.
- Gate commun Go : `cd apps/go-api && GOWORK=off go build ./... && go vet ./... && go test ./<paquets touchés>/...`
  puis `go test ./internal/archlint/...`. Gate commun web : `make check-types && npx vitest run <dossiers
  touchés> --maxWorkers=2 && node tools/lint-cross-feature-imports.mjs && node tools/lint-no-hardcoded-colors.mjs`.
- Un helper neuf vient avec son garde-rail et la migration de toutes les copies dans le même commit.
- Aucune génération de rejeu, aucune recuisson de fond, aucun accès à la prod, aucun navigateur.
- Reprise de session : ce fichier (cases, journal), `git log --oneline e95769096a..chore/830-backlog-1118`,
  `git worktree list`, `ListAgents`, puis l'entrée la plus récente de `.ai/thought_log.md`.

| Piste | Items | Modèle | Effort | Risque |
|---|---|---|---|---|
| 0 témoins et mesures | 4 | pilote (Fable) ; un agent Sonnet en lecture pour 0.3 | rapide | aucun |
| A départs, ramassages, médailles du fil | 3 | Sonnet | moyen | moyen (règle validée sur le parc) |
| B états de la page | 3 | Sonnet | rapide | faible |
| C cadres et orientation | 4 | Sonnet | moyen | faible ; C.4 moyen |
| D fiche du joueur | 7 | Sonnet (D.2, D.4, D.5, D.6, D.7), Haiku (D.1, D.3) | moyen | faible |
| E jetons | 2 | Sonnet | moyen | moyen (jetons, mise en commun) |
| F rejeu à la demande et régénération | 9 | Sonnet | moyen | moyen (file, jetons du joueur) |
| G serveur : contrat et lectures | 5 | Sonnet (G.1, G.2, G.5), Haiku (G.3, G.4) | moyen | faible (contrat de l'API) |
| H couches, accessibilité, clé de requête | 4 | Sonnet (H.1 à H.3), Haiku (H.4) | moyen | moyen (composants de toute l'application) |
| K garde-rails et outillage | 5 | Haiku (K.1, K.3, K.4), Sonnet (K.2, K.5) | rapide | faible |
| M barre de lecture : écran étroit et surimpression | 3 | Sonnet | lourd | moyen (interface, sur planche validée) |
| Q écarts avec les maquettes | 2 | Sonnet | moyen | moyen (même page que la validation à l'écran en cours) |
| S schéma d'une base joueur et tests du paquet DuckDB | 4 | Sonnet | moyen | moyen (définition des bases neuves) |
| T coup d'envoi : un seul système | 7 | pilote (Fable) pour T.0 et T.1 ; Sonnet | lourd | élevé (écritures sur une table critique, note LUSR, tous les lecteurs de durée) |
| I, R intégration et relecture | 8 | pilote ; Fable pour la relecture générale, Sonnet pour les relectures ciblées (jetons ; écritures de la piste T) ; Haiku pour le backlog | moyen | aucun |

Aucun item ne demande Opus. Estimation : environ 18 agents Sonnet, 8 Haiku, 1 Fable, plus une ronde de
corrections (2 à 3 Sonnet). Le chantier décodeur (D-8) n'est pas compté.

## Étape 0 — témoins, mesures et vérification sur pièces

- [ ] 0.1 Branche `chore/830-backlog-1118` créée depuis la tête de `origin/release/v8.3.0` du jour du go ;
      worktrees des pistes de la vague 1.
- [ ] 0.2 Témoins AVANT, dans `.ai/annexes_backlog_1118/temoins_avant.md` : nombre de tests vitest de `MR/`,
      de `apps/web/src/components/` et de `apps/web/src/features/squad/` ; sorties des tests par minute de
      l'Escouade (`per_minute_caracterisation_test.go`, `per_minute_temoins_test.go`) ; compte du garde-rail
      des couches ; pour le match `38900a5e`, les lignes du fil entre 4:30 et 5:30 et l'état de la fiche
      d'ELOsk11 aux images 9700, 9786, 9900 et 9940, tirés d'un test jetable hors dépôt.
- [ ] 0.3 Mesures qui décident, sur les rejeux du poste, un fichier à la fois, lecture seule, résultats dans
      `.ai/annexes_backlog_1118/mesures.md` :
      (a) départs : faite le 2026-11-18 (`mesure_departs.md`) ; à compléter par la répartition de l'écart
      entre la fin de la dernière vie et la fermeture de la présence, et par le compte des corps supprimés
      qui ne sont PAS une dernière vie de partant, classés (fin de manche, fin de match, autre) ;
      (b) score à l'instant : nombre de joueurs et d'images où le score du calque diffère de la somme des
      points par action ;
      (c) lâchers d'équipement datés à fin de vie + 2 dont la place est reprise à + 1 ou + 2 ;
      (d) durée de `readInstantDocSeries` sur trois rejeux réels (4 contre 4, Grande bataille, match long).
      Seuils : D.5 s'exécute si (c) > 0 ; D.6 s'exécute si (d) dépasse 50 ms sur un des trois.
- [ ] 0.4 Chaque référence `fichier:ligne` des pistes est rouverte ; un écart avec ce plan est écrit au
      journal avant tout code.
- **Gate 0** : les deux annexes existent ; (a) confirme qu'aucun corps supprimé en dernière vie de partant
  n'a de compte à rebours ; seuils de D.5 et D.6 statués au journal.

## Piste A — rejeu : départs, ramassages, médailles du fil (Sonnet)

- [ ] A.1 Un départ n'est pas une mort. Règle, dans un module pur neuf `MR/model/departureLogic.ts` : une fin
      de vie est un DÉPART quand (1) c'est la dernière vie du joueur dans le document, (2) son `endReason`
      vaut `deleted`, (3) aucun compte à rebours de réapparition du joueur ne commence dans la tolérance de
      `lifeEndProof`, (4) la présence du joueur au tableau (`roster[].presence`) se ferme avant la fin du
      document, hors marge de fin de partie. Un document sans présence publiée garde le comportement actuel.
      Effets : aucune ligne « mort » pour cette fin de vie (`killFeedLogic.ts`, les deux chemins
      d'alignement), qui ne compte plus parmi les morts non posées ; de la fin de cette vie à la fermeture
      de la présence, la fiche et la tuile du joueur disent « a quitté la partie » (texte existant
      `presenceLeft`) au lieu de l'état « mort » ; aucune marque de mort sur sa piste de la frise. La ligne
      « a quitté la partie » du fil et l'en-tête de `lifeEndProof.ts` (un corps supprimé n'est pas toujours
      une fin de manche) sont mis d'accord avec la règle. Tests : document du match `38900a5e` aminci en
      fixture ; un partant mort avant de partir (ligne « mort » gardée) ; une fin de manche (inchangée) ; un
      document sans présence.
- [ ] A.2 Pictogramme d'une ligne de ramassage, selon D-1, quatre tracés redessinés proprement sur la grille
      de 24 dans `MR/model/feedGlyphs.ts` : `pickup` = flèche qui monte du sol ; `pickupPower` = la même et
      l'étoile ; `pickupEquipment` = caisse (le bouclier est retiré) ; un tracé neuf, caisse et étoile, pour
      un équipement qui change la partie. La nature d'une prise du fil gagne ce quatrième cas : un équipement
      dont la clé figure parmi les socles de puissance (`POWER_PAD_KEYS` de `weaponPadFamilies.ts` :
      surbouclier, camouflage) ; `PadFamily`, qui teinte les socles de la carte, ne change pas. L'entrée
      « Armes et bonus » du menu de filtre prend la flèche. Aucune silhouette d'arme. Tests : les quatre cas
      rendent quatre tracés ; un équipement hors catalogue reste « caisse ».
- [ ] A.3 Médailles du fil depuis une planche d'images : `MedalEvent` porte la découpe servie par le document
      (`matchMedals.ts:40`) ; `ReplayFeedMedals.tsx` et `MedalBadges.tsx` dessinent par
      `components/ui/MedalIcon.tsx`, comme la fiche. Test : une médaille à découpe rend son image dans les deux
      composants ; une médaille à image seule est inchangée.
- **Gate A** : gate commun web sur `MR/` ; témoin 0.2 rejoué : plus de ligne « mort » pour ELOsk11, fiche
  « a quitté la partie » aux images 9786 et 9900 ; le pilote rejoue la mesure (a) avec la règle codée : 875
  départs reconnus, 503 morts gardées.

## Piste B — rejeu : états de la page (Sonnet)

- [ ] B.1 Selon D-2 : dans `MR/model/playerPanelLogic.ts`, « Seul » éteint le suivi d'un autre joueur et
      `followPlayer` sur un autre joueur lève « Seul ». Quatre tests (les deux sens, le même joueur, fiche
      fermée).
- [ ] B.2 `useReplayViewpoint(scopeKey, scoreboard)` : état rangé avec sa portée et remis à zéro au rendu
      (patron de `usePlayerPanel.ts:63-71`), clé du cadre (`ReplayPage.tsx:179`). Test : nouvelle portée et
      même joueur présent, point de vue par défaut.
- [ ] B.3 `useReplayFullscreen.ts` : la sortie d'un Plein cadre né du geste passe par `release` quel que soit
      le chemin (touche F, icône « Standard », « Quitter ») ; un plein écran refusé ou absent remet
      l'intention à faux. Tests : les deux scénarios de la section 1.
- **Gate B** : gate commun web sur `MR/`.

## Piste C — rejeu : cadres et orientation (Sonnet)

- [ ] C.1 Marge de 24 px entre « la rangée tient » et « elle ne tient plus » : fonction pure dans
      `MR/model/rowLayout.ts`, mémoire de la disposition précédente dans `useCardRowLayout.ts` ; même marge
      pour l'emplacement du score. Tests aux bornes, et séquence 1290, 1275, 1290.
- [ ] C.2 `useResolvedTurns` : l'état « tourné avant » est rangé avec sa portée (un match) ; il est gelé tant
      qu'un export est en cours. Tests : gain de 30 % puis nouveau match à 22 % (attendu : droit) ; largeur
      changée pendant un export (attendu : inchangé).
- [ ] C.3 Horloge de la barre en pause : test d'abord (pause, bascule Standard vers Cinéma à taille de toile
      égale, le texte de l'horloge est présent) ; s'il échoue, `useReplayClock` réécrit le dernier texte quand
      l'élément change ; s'il passe, l'item est clos sur ce test, gardé.
- [ ] C.4 Selon D-3 : la Tactique (`TacticalPlanCard.tsx`, `TacticalReapparitionsPlan.tsx`) et « Occupation
      du terrain » (`MatchPositionsHeatmap.tsx`) résolvent l'Auto sur leur propre zone par la règle du rejeu
      (`autoTurns`) ; « Occupation du terrain » mesure sa zone ; un choix retenu pour la carte prime
      toujours ; `useMapViewPreferences.ts` et son commentaire sont mis d'accord. Tests par vue : carte en
      Auto, zone large puis haute.
- **Gate C** : gate commun web sur `MR/`, `features/tactical/`, `features/match-view/`.

## Piste D — fiche du joueur (Sonnet ; Haiku pour D.1 et D.3)

- [ ] D.1 `playerInstantStats.ts` : les débuts de vie ne retiennent que les vies dont la fin dépasse le début
      du match, comme `closedLives`. Test : vies [10, 40] et [41, 200], début à 50, image 100, 0 réapparition.
- [ ] D.2 « Tirs et dégâts » : une ligne par nom d'arme affiché, familles de même nom additionnées
      (`playerInstantView.ts`) ; la clé de liste est celle de la première famille du groupe
      (`FicheTirs.tsx:35`). Test : deux familles au même nom, une ligne, sommes justes.
- [ ] D.3 Nom de grenade de repli : `grenadeRank` au contrat des textes, en français et en anglais ;
      `grenadesCarriedFrom` reçoit le texte ; les deux appelants suivent. Test en anglais.
- [ ] D.4 `GABARIT_NORMAL` retiré : une cote nommée pour le corps du bloc « À l'instant » ; `CardGabarit` ne
      garde que la tuile compacte ; branches mortes de `ReplayCountersBadge.tsx` et leurs tests retirés ;
      `cardGabarit.guard.test.ts` suit.
- [ ] D.5 (si 0.3 c > 0) Lâcher daté à fin de vie + 2 : une résolution « la vie qui s'est terminée à l'image
      t − 2 ou t − 1 » prime sur l'occupant courant pour un lâcher (`rosterLogic.ts`, `equipmentGestures.ts`).
      Témoin : nombre de lâchers attribués sur les huit documents réels, avant et après (338 sur 338 avant).
- [ ] D.6 (si 0.3 d > 50 ms) Séries de la fiche gardées par document et par liste de joueurs, hors de
      l'instance du composant (`playerInstantSources.ts`). Test : deux lectures, même objet.
- [ ] D.7 (si 0.3 b > 0) Un seul score à l'instant : l'infobulle de la tuile (`ReplayCountersBadge.tsx:146`)
      lit la valeur gardée de la fiche ; quand la garde refuse, elle ne montre pas de score. Si 0.3 b = 0 :
      item clos sur la mesure, l'égalité des deux sources devient un test.
- **Gate D** : gate commun web sur `MR/` ; `fr.ts` et `en.ts` sous 500 lignes.

## Piste E — jetons des téléchargements de fond (Sonnet)

- [ ] E.1 `haloTokensForDrain` prend son jeton à la source de lecture publique de la mise en commun
      (`publicReadTokenSrc.ResolveAny`), jamais dans la liste des profils ; sans source ou sans compte
      consentant : erreur typée, journalisée, sans conseil de recapture. Les trois appelants (catalogue,
      expansion des playlists, mise en file des rejeux) suivent. Libellé de `registry_auth.go:289-291`
      reformulé (premier enrôlement, pas recapture). Tests : compte retenu jamais servi ; aucune source ;
      `registry_catalog_lectures_cgo_test.go` adapté.
- [ ] E.2 Garde-rail : `RefreshHaloTokensViaStoreFirst` combiné à un parcours des profils est interdit dans
      `internal/api/wire`, liste d'exemptions datée vide.
- **Gate E** : gate commun Go ; `go test ./internal/api/wire/... ./internal/platform/auth/...` ;
  `go test -tags=integration -p 1 ./internal/sync/...`.

## Piste F — rejeu à la demande et régénération du bureau (Sonnet)

- [ ] F.1 Garde des jetons en liste blanche : dans `registry_replay_generation*.go`, seul
      `tokencache.ResolveFreshPlayerTokens` est admis.
- [ ] F.2 Demande reprise après redémarrage : revalidée par la sonde du match avant tout téléchargement
      (`replayregen/demand_run.go`).
- [ ] F.3 Nouvel essai après échec : les contrôles d'état passent d'abord
      (`service/replay_generation_service.go:117-119`).
- [ ] F.4 Film retiré du disque après un échec de génération non réessayable quand l'archive est désactivée.
- [ ] F.5 Capacité de génération : bureau, ou développement non exposé (`config/deployment.go`) ; test dans
      `deployment_test.go`.
- [ ] F.6 Bouton : pas proposé quand le jeton du joueur demande une reconnexion ; « Réessayer » retiré pour
      un jeton indisponible (web, `replayGeneration/logic.ts`).
- [ ] F.7 Jeton refusé pendant la résolution du film : un nouvel essai avec un jeton frais
      (`tokencache.RetryWithFreshTokens`).
- [ ] F.8 Selon D-5 : la file de régénération reprojette, sans film, le fichier tactique périmé d'un rejeu
      dont le film a expiré (`replayregen/scan.go`, `ProjeterRasterTactique`). Tests : film absent et fichier
      au format précédent, fichier refait ; film absent et fichier à jour, rien.
- [ ] F.9 Clic juste après la fin d'un match. D'abord l'inventaire, écrit au journal, de ce que la réponse du
      service des films distingue déjà : film introuvable, manifeste sans morceaux, manifeste sans morceau de
      temps forts (`finalise.ErrFilmNonFinalise`, déjà réessayable). Si un de ces signaux reconnaît à lui seul
      un film pas encore publié, il suffit et aucune fenêtre n'est ajoutée. À défaut : la sonde du match porte
      l'heure de fin, et pendant les 5 minutes qui suivent (constante nommée) un film introuvable ne pose pas
      le verdict définitif « film absent » : la demande échoue en « film pas encore prêt », réessayable. La
      synchronisation n'est pas modifiée. Tests : match fini depuis 2 minutes, depuis 2 heures.
- **Gate F** : gate commun Go ; `go test ./internal/replayregen/... ./internal/service/... ./internal/config/...` ;
  gate commun web sur le dossier de F.6.

## Piste G — serveur : contrat et lectures (Sonnet ; Haiku pour G.3 et G.4)

- [ ] G.1 Selon D-7 : inventaire fermé au journal (lecteurs de chaque champ de `MatchHighlightEvent` propre
      aux médailles), puis retrait des trois champs, de `match_view_killfeed_medals.go`, de son appel, de
      `MedalImageURL` des adaptateurs de titre s'il n'a plus d'appelant, des tests et fixtures qui les
      portent. `medal_name_id` de l'événement : retiré si l'inventaire ne lui trouve aucun lecteur, gardé et
      écrit au journal sinon.
- [ ] G.2 Selon D-4 : les trois taux par minute de l'Escouade deviennent facultatifs ; « inconnu » quand
      aucun match n'est retenu ; tests de caractérisation mis à jour ; type du graphe côté web.
- [ ] G.3 `medalLabelOf` : parcours des langues dans l'ordre de `i18n.Locales()`. Test déterministe.
- [ ] G.4 Compteur et message distincts pour un fichier tactique illisible (`replayartifact/gate.go`).
- [~] G.5 Durée du découpage de la Dominance : couvert par la piste T (T.3), qui fait passer tous les
      lecteurs par une résolution unique au lieu d'ajouter une préférence de plus ici.
- **Gate G** : gate commun Go sur `internal/service/...`, `internal/games/...`, `internal/replayartifact/...` ;
  témoins 0.2 de l'Escouade comparés ; le pilote régénère `openapi.yaml` et `generated.ts` à l'intégration.

## Piste H — couches, accessibilité, clé de requête (Sonnet ; Haiku pour H.4)

- [ ] H.1 Garde-rail des couches élargi à `components/`, `lib/` et aux fichiers `.ts` de `features/` ; il
      reconnaît un nom d'événement non littéral et `case 'Escape'` ; exemptions datées pour
      `titleTooltip.controller.ts`, `tooltipPosition.ts`, `sidebarCollapse.ts`, `useDismissable.ts`,
      `useReplayFullscreen.ts`, `assetPanel.ts`.
- [ ] H.2 Les 13 composants migrent vers `useDismissable` : `analysis-pending-badge`, `alert-dialog`,
      `GamertagCombobox` (clic dehors seul), `info-tooltip`, `SessionMultiSelect`, `AboutDialog` (son piège
      de tabulation reste), `HelpSplitButton`, `NavL1` (deux sites), `MultiSelectFilter`,
      `NavL1MobileActions`, `NavL1MobileMenu`, `PlayerSwitcher`, `RailPopover`. Changements voulus, écrits aux
      notes de version : Échap ferme aussi les cinq qui ne l'écoutaient pas ; des couches empilées se ferment
      une par une. Les tests existants de ces composants restent verts sans retouche de leurs assertions.
- [ ] H.3 `MediaPickerDialog` écoute Échap par le mécanisme commun ; `ExperienceDropdown` et `ViewDropdown`
      annoncent leur état ouvert. Tests calqués sur `CoverFlowModal.dismiss.test.tsx`.
- [ ] H.4 `queryKeys.matchReplayContext` porte la langue ; `useMatchReplayContext` la lit ;
      `keys.locale.test.ts` range la clé parmi les localisées.
- **Gate H** : gate commun web sur `components/`, `features/_shared/`, `features/media/`, `lib/query/`,
  `lib/replay/`.

## Piste K — garde-rails et outillage (Haiku ; Sonnet pour K.2 et K.5)

- [ ] K.1 Trois entrées mortes retirées de `ALLOWED_FILES` ; test « chaque chemin de la liste existe ».
- [ ] K.2 Rempart des fonds de carte : retrait de la racine d'installation connue en plus du motif ; dossiers
      à point et à espaces ; garde-rail étendu (partage réseau, `/opt/`, `/mnt/`, `/tmp/`, `/var/`,
      `/Volumes/`). Aucune recuisson.
- [ ] K.3 `scripts/disk-hygiene.ps1` : base résolue (amont de la branche courante, sinon branche par défaut
      du dépôt), paramètre `-BaseRef`, « base introuvable » journalisé sans arrêt. Contrôle : simulation sans
      erreur.
- [ ] K.4 Test du processus qui tient la base : sortie d'erreur du fils branchée, lecture du « prêt » sous
      délai explicite, sortie du fils jointe à l'échec.
- [ ] K.5 `bootstrap_service.go` : trois blocs déplacés sans renommage (`bootstrap_setup_state.go`,
      `bootstrap_ownership.go`, `bootstrap_helpers.go`) ; `buildCapabilities` et `buildFeatureFlags` ne
      bougent pas ; compteurs des trois garde-rails indexés sur le chemin ajustés, somme inchangée.
- **Gate K** : gate commun Go et web selon l'item ; `wc -l` de `bootstrap_service.go` sous 500.

## Piste M — barre de lecture : écran étroit et surimpression (Sonnet)

- [ ] M.1 Planche publiée (page d'artefact, dessinée DANS les cadres validés de la refonte), à trois largeurs
      de fenêtre (1 280, 1 440, 1 536 px) : (a) la barre d'une ligne quand la place manque, deux variantes de
      repli (groupes zoom, capture et vitesse rangés dans un menu ; ou barre sur deux lignes) ; (b) la barre en
      surimpression au Plein cadre, selon la proposition notée au backlog (fine ligne de progression au bas du
      plan pendant la lecture, dépliée au survol du bas, à toute touche et en pause ; fil des éliminations qui
      remonte quand elle est dépliée) et une variante. Aucun code de l'application.
- [ ] M.2 (porte : variante de repli choisie par le user) Repli de la barre d'une ligne sous le seuil de
      largeur retenu sur la planche (`ReplayTransportCompact.tsx`, `cotesBarreCompacte.ts`) ; la frise ne
      descend jamais sous sa largeur minimale. Tests de logique pure sur le seuil.
- [ ] M.3 (porte : surimpression validée par le user sur la planche) Barre en surimpression au Plein cadre,
      telle que validée ; Standard et Cinéma gardent leur barre fixe ; préférence lue une fois, textes dans les
      deux langues. Tests : pause, survol, touche, lecture.
- **Gate M** : gate commun web sur `MR/` ; contrôle à l'écran du user inscrit à l'artefact des vérifications.

## Piste Q — écarts avec les maquettes (Sonnet)

- [ ] Q.1 Instruction en lecture des onze points de l'entrée du backlog, contre les fichiers de maquette
      (`.ai/annexes_refonte_ui_rejeu/maquettes/`) et la grille (`conformite_maquettes.md`) : chaque point classé
      « conforme à une décision datée », « écart à corriger » (avec la cote ou le dessin attendu) ou « sans
      dessin, au user ». Résultat dans `.ai/annexes_backlog_1118/ecarts_maquettes.md` ; les écarts à corriger
      sont inscrits ici en liste fermée avant tout code, et montrés au user.
- [ ] Q.2 (porte : liste lue par le user) Corrections de la liste fermée de Q.1 et des retours de la
      validation à l'écran du user, un commit par point.
- **Gate Q** : gate commun web sur `MR/` ; grille de conformité mise à jour ligne par ligne.

## Piste T — coup d'envoi du match : un seul système (pilote pour T.0 et T.1 ; Sonnet)

Demande du user du 2026-11-18 : « faut remettre au propre, qu'on ait pas plusieurs paramètres et de
fallback, je veux un système cohérent ». La piste ne se borne donc plus à recalculer le temps de jeu : elle
remplace l'empilement actuel par une seule définition, une seule résolution, une seule écriture.

État relevé sur pièces le 2026-11-18 (à compléter en T.0) :
- deux colonnes au registre (`real_start_time`, `t0_quality`), sept valeurs de qualité
  (`analysis/timeline/compute_t0.go`), dont quatre disent « valeur » et trois « rejet » ;
- une estimation par l'API à la synchronisation (présence des joueurs, seuils de 2 s et de 120 s), remplacée
  plus tard par le film, lu comme « premier mouvement des joueurs » avec des cas refusés (8 rejeux du poste) ;
- un repli « coup d'envoi = 0 » dans les lecteurs quand la colonne est vide ;
- trois durées concurrentes : durée du match, durée jouable de l'API (fausse en classé Arène), durée moins
  coup d'envoi ; la Dominance préfère la deuxième, l'en-tête la troisième ;
- le temps de jeu d'un joueur : valeur de l'API, remplacée par un calcul si celui-ci aboutit, jamais refait
  quand le coup d'envoi change ; une page lit encore « temps de jeu, sinon durée jouable » ;
- trois commandes de rattrapage à écriture directe (`cmd/backfill_t0`, `cmd/backfill_t0_film`,
  `cmd/backfill_time_played`) ;
- dans le rejeu : l'origine, le coup d'envoi « premier mouvement », et l'horloge de manche du jeu, lue chaque
  seconde, qui date elle aussi le coup d'envoi.

- [ ] T.0 Audit (lecture seule ; base lue sur une COPIE), dans `.ai/annexes_backlog_1118/coup_d_envoi.md` :
      inventaire FERMÉ de chaque source, seuil, qualité, repli, écrivain, lecteur (Go, SQL, web) et commande ;
      mesures sur le parc : écart entre l'horloge de manche du film, le premier mouvement et l'estimation de
      l'API ; matchs sans film, par source restante ; écart de temps de jeu par joueur ; dépendance de la note
      de performance et du LUSR (ADR 0047).
- [ ] T.1 Modèle cible, écrit par le pilote et VALIDÉ PAR LE USER avant tout code. Principes proposés :
      (1) une définition : le coup d'envoi est l'instant où l'horloge du jeu démarre ; (2) une source par
      match, dans un ordre fixe écrit une fois : le film quand il existe, l'estimation de l'API sinon,
      « inconnu » sinon, et la source est rangée avec la valeur ; (3) un seul point de résolution, qui rend la
      chronologie du match (coup d'envoi, fin, durée de jeu) : aucun lecteur ne choisit entre deux durées ni
      ne remplace un vide par zéro ; (4) une seule écriture, par un persisteur : coup d'envoi et valeurs
      dérivées (temps de jeu de chaque joueur) dans la même transaction, à la synchronisation comme à
      l'arrivée du film ; (5) une seule commande de rattrapage ; (6) la durée jouable de l'API n'a plus de
      lecteur. Le modèle dit aussi ce que devient chaque ligne de l'inventaire de T.0 (gardée, fusionnée,
      retirée), et s'écrit en ADR (anglais) avec ses garde-rails.
- [ ] T.2 Résolution unique : type et fonction de la chronologie du match ; les qualités de l'estimation
      deviennent un diagnostic journalisé et compté, plus une valeur que les lecteurs interprètent.
- [ ] T.3 Lecteurs : chacun de la liste fermée de T.0 passe par la résolution unique (Dominance comprise :
      l'item G.5 est couvert ici) ; garde-rail : aucune lecture de la durée jouable de l'API ni du coup
      d'envoi brut hors de la résolution.
- [ ] T.4 Écriture unique : le persisteur écrit coup d'envoi, source et temps de jeu ensemble (règles
      anti-corruption du dépôt, jamais d'écriture directe) ; le report du film et la synchronisation
      l'appellent tous deux.
- [ ] T.5 Une commande `levelup` de rattrapage avec simulation ; les trois `cmd/backfill_*` sont supprimés
      avec leurs tests (aucun code mort).
- [ ] T.6 (porte : accord du user, serveur arrêté, copie de la base faite) Rattrapage du poste, puis rejeu de
      la note LUSR si T.0 l'a jugé nécessaire. La prod reçoit le même geste à la publication.
- Lien avec le plan du décodeur : si T.0 montre que l'horloge de manche date mieux le coup d'envoi que le
  premier mouvement, le décodeur publie le coup d'envoi lu sur l'horloge et retire le détecteur de mouvement
  (item du plan du décodeur) ; LevelUp lit le même champ avant comme après.
- **Gate T** : modèle validé par le user ; `cd apps/go-api && go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/platform/duckdb/...`,
  code de sortie 0 ; `go test ./internal/archlint/...` ; inventaire de T.0 entièrement statué ; relecture
  ciblée des écritures (R.2).

## Piste S — schéma d'une base joueur et tests du paquet DuckDB (Sonnet)

- [ ] S.1 Test d'égalité dans les deux sens (« synchronisation puis migrations » contre « migrations
      seules ») sur types, défauts et `NOT NULL` ; écart listé dans `.ai/annexes_backlog_1118/schema_joueur.md` ;
      forme des bases existantes relevée sur une COPIE d'une base joueur du poste.
- [ ] S.2 La définition qui s'écarte de la forme de référence est alignée, pour les bases neuves seulement ;
      aucune migration d'une base existante ; si l'alignement demande de modifier une étape de migration déjà
      jouée en production, arrêt et retour au user.
- [ ] S.3 L'exemption de `csr_pipeline_e2e_integration_test.go` est retirée si le test d'égalité la rend
      inutile ; sinon elle garde sa date et son motif.
- [ ] S.4 Migrations neuves indirectes : les cinq sites qui partent d'un fichier joueur absent
      (`pool_attach_swap_integration_test.go:116`, `pool_shared_reader_integration_test.go:71`, `:125`, `:186`,
      `pool_swap_multititre_integration_test.go:103`) reçoivent la base migrée copiée avant `GetOrOpen` ;
      `pool_migration_test.go` (dont l'objet est la migration) passe sur la liste d'exemptions datée ; le
      garde-rail reconnaît `GetOrOpen` et un nom de fonction de migration pris en valeur ; les deux
      `ForTarget` et l'`ExecScript` (quelques étapes, gain nul) sont nommés hors périmètre dans son en-tête.
      Témoin : durée et nombre de tests du paquet, avant et après.
- **Gate S** : `cd apps/go-api && go test -tags=integration -p 1 ./internal/sync/... ./internal/platform/duckdb/...`,
  code de sortie 0, échecs cherchés par `^--- FAIL:`.

## Étape I — intégration (pilote)

- [ ] I.1 Recalage sur `origin/release/v8.3.0` ; fusion des sous-branches dans `chore/830-backlog-1118` ;
      `openapi.yaml` et `make generate-types` une seule fois.
- [ ] I.2 Témoins APRÈS comparés à 0.2 : écarts listés, chacun rattaché à un item ou déclaré défaut.
- [ ] I.3 `docs/CHANGELOG.md` et notes de version de la 8.3.0, en français et en anglais (départs, ramassages,
      « Seul » et « Suivre », Escouade, Échap) ; `.ai/thought_log.md` ; caches dédiés rendus.
- **Gate I** : `make gate-push` ; `cd apps/go-api && go test -tags=integration -p 1 ./internal/sync/... ./internal/platform/duckdb/...` ;
  `make go-api-lint` ; `cd apps/web && npx vitest run` complet, cache de types purgé avant `make check-types`.

## Étape R — relecture et clôture

- [ ] R.1 Backlog mis à jour par un agent Haiku : entrées traitées déplacées au tableau des entrées closes ;
      une entrée dont la porte du user n'est pas passée garde sa ligne, avec la date de la demande ; entrées
      du décodeur selon D-8 ; registre des reports.
- [ ] R.2 Une ronde de relecture adversariale du diff complet (skill `adversarial-review`, Fable) et deux
      relectures ciblées (Sonnet) : jetons et file (pistes E et F), écritures en base (piste T) ; P0 et P1
      corrigés en une ronde, P2 au backlog seulement s'ils valent le coup.
- [ ] R.3 Artefact des vérifications du user : départ d'ELOsk11 à 5:00 sur le match `38900a5e` ; ligne de
      ramassage ; « Seul » puis « Suivre » ; sortie du Plein cadre ; orientation Auto dans la Tactique ; barre
      d'une ligne au Cinéma, fenêtre d'environ 1 440 px, disposition en colonne.
- [ ] R.4 Tous les items statués ; outillage de mesure retiré ou documenté ; fusion dans `release/v8.3.0` ;
      CI lue après la poussée.

## Journal d'exécution

- 2026-11-18 — GO du user. Plans commités sur `release/v8.3.0` (`1a7329e625`). 0.1 fait : branche
  `chore/830-backlog-1118` et worktrees `C:/t/bl1118` (intégration, étape 0), `C:/t/bl18-c` (piste C),
  `C:/t/bl18-k` (K.2), tous depuis `1a7329e625`, sans jonction.
  Amendement du pilote, écrit avant tout lancement : les pistes qui ne lisent aucune mesure de l'étape 0
  (C, E, G, K.2) démarrent EN PARALLÈLE de l'étape 0, parce que les témoins « avant » se prennent sur le
  commit de base dans un autre dossier et que le plan du fond Relief attend C.4 et K.2 ; AUCUNE piste n'est
  intégrée avant la Gate 0. L'item 0.4 (références rouvertes) est exécuté par chaque agent de piste en tête
  de son travail, écarts rapportés avant tout code. Le suivi (cases, journal) s'écrit dans le dossier
  principal sur `release/v8.3.0`, commits de documentation seuls.
  Lancés (5 agents Sonnet) : étape 0 (0.2 et 0.3, mesures) ; piste C ; K.2 ; audit du coup d'envoi (T.0) avec
  l'export de vérité de l'API sur une copie de la base (`C:/t/verite/`, partagé avec le plan du décodeur) ;
  et, pour le plan du décodeur, sa marche à blanc. Deux agents d'écriture seulement (C et K.2) : la session
  du fond Relief en tient deux.
- 2026-11-18 — plan écrit ; trois instructions en lecture seule et une mesure sur le parc ; aucun code.
- 2026-11-18 — réponses du user : D-2, D-3, D-4, D-5, D-7 retenues ; silhouette de l'arme écartée pour le
  ramassage, planche de quatre tracés publiée (D-1 rouverte) ; correction de périmètre : les neuf entrées
  écartées entrent dans le plan (A.3, D.7, F.9, S.4, pistes M, Q et T) ; D-8 posée pour les entrées du
  décodeur. Erreur du pilote : avoir lu « ce qui est pertinent et possible » comme un tri, alors que la
  demande était de traiter les entrées.
- 2026-11-18 — seconde série de réponses du user : quatre tracés de ramassage (flèche pour les armes, caisse
  pour les équipements, étoile pour ce qui change la partie) ; fenêtre du clic après match à 5 minutes, après
  recherche d'un signal du film ; décodeur dans un plan à part, version visée 8.3.0, les deux plans pilotés
  par cette session ; la session de validation à l'écran est archivée. Trois instructions en lecture seule
  lancées pour le plan du décodeur.

## Découvertes (consignées, non traitées)

(vide)
