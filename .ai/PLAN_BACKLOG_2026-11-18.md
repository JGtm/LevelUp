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
d'une ligne de ramassage du fil du rejeu lui déplaît ; sur le match `38900a5e`, à 5:00, le joueur A est donné mort
alors qu'il quitte la partie.

Abréviation : `MR/` = `apps/web/src/features/match-replay/`.

## 0. Objectif et critère de succès

Objectif : régler ce qui se règle maintenant dans les entrées des 17 et 18 novembre, corriger les deux retours
du user, et donner un sort écrit à chaque entrée instruite.

Critère de succès : chaque entrée de la section 1 a un sort écrit (traitée, close, reformulée, gardée avec sa
condition) ; sur le match `38900a5e`, le joueur A n'a plus de ligne « mort » à 5:00 et sa fiche dit qu'il a quitté
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
| Le joueur A « mort » à 5:00 (match `38900a5e`) | Le décodeur lit le départ : mort à 4:41 (image 9539, compte à rebours lu), réapparition à l'image 9783, corps SUPPRIMÉ à l'image 9785 (`endReason: "deleted"`, aucun lâcher, aucun compte à rebours), présence au tableau fermée à l'image 9937, place reprise par le bot 343 Mickey à 9938. Le site n'en tire rien : toute fin de vie close sans frag devient une ligne « mort » (`MR/model/killFeedLogic.ts:300-302` et `:469-471`), `MR/model/lifeEndProof.ts:9` range `deleted` sous « fin de manche », et la fiche lit « mort » dès qu'aucune vie ne couvre l'image (`apps/web/src/lib/replay/rosterLogic.ts:538-554`). Seule la croix sur la carte est évitée. Parc : 1 386 départs en cours de match dans 729 rejeux ; 875 finissent par un corps supprimé (le schéma du joueur A), 503 par une vraie mort suivie du départ, 8 sans raison publiée ; 5 rejeux ne publient pas de présence | Piste A (A.1), côté site seulement |
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

- [x] 0.1 (fait le 2026-11-18, depuis `1a7329e625`) Branche `chore/830-backlog-1118` créée depuis la tête de `origin/release/v8.3.0` du jour du go ;
      worktrees des pistes de la vague 1.
- [x] 0.2 (fait le 2026-11-18 : rejeu 390 fichiers de test, 5 517 réussis ; composants 1 208 ; Escouade 999 ; 12 tests Go par minute ; témoin du match pris SANS les frags de l'API, dit en tête de l'annexe) Témoins AVANT, dans `.ai/annexes_backlog_1118/temoins_avant.md` : nombre de tests vitest de `MR/`,
      de `apps/web/src/components/` et de `apps/web/src/features/squad/` ; sorties des tests par minute de
      l'Escouade (`per_minute_caracterisation_test.go`, `per_minute_temoins_test.go`) ; compte du garde-rail
      des couches ; pour le match `38900a5e`, les lignes du fil entre 4:30 et 5:30 et l'état de la fiche
      du joueur A aux images 9700, 9786, 9900 et 9940, tirés d'un test jetable hors dépôt.
- [x] 0.3 (fait le 2026-11-18 : (a) 0 compte à rebours sur 875 ; (b) 6 539 joueurs sur 12 126 en écart sur au moins une image, 166 au total final différent ; (c) 0 lâcher repris à + 1 ou + 2 sur 28 540 ; (d) 2,2 à 4,3 ms) Mesures qui décident, sur les rejeux du poste, un fichier à la fois, lecture seule, résultats dans
      `.ai/annexes_backlog_1118/mesures.md` :
      (a) départs : faite le 2026-11-18 (`mesure_departs.md`) ; à compléter par la répartition de l'écart
      entre la fin de la dernière vie et la fermeture de la présence, et par le compte des corps supprimés
      qui ne sont PAS une dernière vie de partant, classés (fin de manche, fin de match, autre) ;
      (b) score à l'instant : nombre de joueurs et d'images où le score du calque diffère de la somme des
      points par action ;
      (c) lâchers d'équipement datés à fin de vie + 2 dont la place est reprise à + 1 ou + 2 ;
      (d) durée de `readInstantDocSeries` sur trois rejeux réels (4 contre 4, Grande bataille, match long).
      Seuils : D.5 s'exécute si (c) > 0 ; D.6 s'exécute si (d) dépasse 50 ms sur un des trois.
- [x] 0.4 (exécuté par chaque agent de piste en tête de son travail, écarts rapportés au pilote : amendement du journal) Chaque référence `fichier:ligne` des pistes est rouverte ; un écart avec ce plan est écrit au
      journal avant tout code.
- **Gate 0** : les deux annexes existent ; (a) confirme qu'aucun corps supprimé en dernière vie de partant
  n'a de compte à rebours ; seuils de D.5 et D.6 statués au journal.

## Piste A — rejeu : départs, ramassages, médailles du fil (Sonnet)

- [x] A.1 (fait le 2026-11-18, `d99966fcaa` sur `bl1118/a` ; 884 départs reconnus sur le parc avec la marge de fin de 3 s du fil, soit les 875 mesurés à 10 s et 9 présences fermées entre 3 et 10 s avant la fin ; « a quitté la partie » dure jusqu'à la fin du document et vaut aussi pour un joueur mort puis parti, à partir de la fermeture de sa présence : complément demandé par le pilote ; la tuile d'un parti prend l'allure d'une place vide, choix de présentation à faire valider à l'écran) Un départ n'est pas une mort. Règle, dans un module pur neuf `MR/model/departureLogic.ts` : une fin
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
- [x] A.2 (fait le 2026-11-18, même commit ; étoile à 1,9 unité de grille du tracé principal, vérifiée par un test) Pictogramme d'une ligne de ramassage, selon D-1, quatre tracés redessinés proprement sur la grille
      de 24 dans `MR/model/feedGlyphs.ts` : `pickup` = flèche qui monte du sol ; `pickupPower` = la même et
      l'étoile ; `pickupEquipment` = caisse (le bouclier est retiré) ; un tracé neuf, caisse et étoile, pour
      un équipement qui change la partie. La nature d'une prise du fil gagne ce quatrième cas : un équipement
      dont la clé figure parmi les socles de puissance (`POWER_PAD_KEYS` de `weaponPadFamilies.ts` :
      surbouclier, camouflage) ; `PadFamily`, qui teinte les socles de la carte, ne change pas. L'entrée
      « Armes et bonus » du menu de filtre prend la flèche. Aucune silhouette d'arme. Tests : les quatre cas
      rendent quatre tracés ; un équipement hors catalogue reste « caisse ».
- [x] A.3 (fait le 2026-11-18, même commit ; une médaille à image seule garde exactement son rendu) Médailles du fil depuis une planche d'images : `MedalEvent` porte la découpe servie par le document
      (`matchMedals.ts:40`) ; `ReplayFeedMedals.tsx` et `MedalBadges.tsx` dessinent par
      `components/ui/MedalIcon.tsx`, comme la fiche. Test : une médaille à découpe rend son image dans les deux
      composants ; une médaille à image seule est inchangée.
- **Gate A** : gate commun web sur `MR/` ; témoin 0.2 rejoué : plus de ligne « mort » pour le joueur A, fiche
  « a quitté la partie » aux images 9786 et 9900 ; mesure (a) rejouée avec la règle codée : 875 départs
  reconnus, 511 fins de vie gardées (503 morts et 8 dernières vies sans raison publiée ; chiffre corrigé le
  2026-11-18 sur la mesure 0.3). Amendé le 2026-11-18 au retour de la piste : 884 départs avec la marge de 3 s du fil
  (875 + 9), acceptés par le pilote ; la suite complète du web est passée par le pilote à l'intégration.

## Piste B — rejeu : états de la page (Sonnet)

- [x] B.1 (fait le 2026-11-18, `0aa6debb61` sur `bl1118/b` ; trois tests existants qui attendaient « Seul » gardé en suivant un autre joueur passent sur « Masquer ») Selon D-2 : dans `MR/model/playerPanelLogic.ts`, « Seul » éteint le suivi d'un autre joueur et
      `followPlayer` sur un autre joueur lève « Seul ». Quatre tests (les deux sens, le même joueur, fiche
      fermée).
- [x] B.2 (fait le 2026-11-18, même commit ; la portée passée est `playerSlug/matchId`, la clé du cadre valant `null` pendant le chargement) `useReplayViewpoint(scopeKey, scoreboard)` : état rangé avec sa portée et remis à zéro au rendu
      (patron de `usePlayerPanel.ts:63-71`), clé du cadre (`ReplayPage.tsx:179`). Test : nouvelle portée et
      même joueur présent, point de vue par défaut.
- [x] B.3 (fait le 2026-11-18, même commit ; quatre des tests neufs échouent sur l'ancien code ; un test existant affirmait le défaut et a été retourné) `useReplayFullscreen.ts` : la sortie d'un Plein cadre né du geste passe par `release` quel que soit
      le chemin (touche F, icône « Standard », « Quitter ») ; un plein écran refusé ou absent remet
      l'intention à faux. Tests : les deux scénarios de la section 1.
- **Gate B** : gate commun web sur `MR/`.

## Piste C — rejeu : cadres et orientation (Sonnet)

- [x] C.1 (fait le 2026-11-18, `c5acd8f47e` sur `bl1118/c` ; quatre tests de page existants adaptés à la marge) Marge de 24 px entre « la rangée tient » et « elle ne tient plus » : fonction pure dans
      `MR/model/rowLayout.ts`, mémoire de la disposition précédente dans `useCardRowLayout.ts` ; même marge
      pour l'emplacement du score. Tests aux bornes, et séquence 1290, 1275, 1290.
- [x] C.2 (fait le 2026-11-18, même commit ; la règle Auto n'a plus qu'un appelant, `lib/replay/useAutoTurned.ts`, gardé sur tout le site ; second match du test à 23 % de gain et non 22, entre les deux seuils comme voulu) `useResolvedTurns` : l'état « tourné avant » est rangé avec sa portée (un match) ; il est gelé tant
      qu'un export est en cours. Tests : gain de 30 % puis nouveau match à 22 % (attendu : droit) ; largeur
      changée pendant un export (attendu : inchangé).
- [x] C.3 (fait le 2026-11-18, même commit : le test passe, `useReplayClock` inchangé, test gardé ; l'horloge reste écrite parce que la toile est redessinée à chaque rendu, voir Découvertes) Horloge de la barre en pause : test d'abord (pause, bascule Standard vers Cinéma à taille de toile
      égale, le texte de l'horloge est présent) ; s'il échoue, `useReplayClock` réécrit le dernier texte quand
      l'élément change ; s'il passe, l'item est clos sur ce test, gardé.
- [x] C.4 (fait le 2026-11-18, même commit ; `TacticalAnalysisView.test.tsx` pose l'orientation retenue de sa carte parce qu'il force la largeur de tous les éléments) Selon D-3 : la Tactique (`TacticalPlanCard.tsx`, `TacticalReapparitionsPlan.tsx`) et « Occupation
      du terrain » (`MatchPositionsHeatmap.tsx`) résolvent l'Auto sur leur propre zone par la règle du rejeu
      (`autoTurns`) ; « Occupation du terrain » mesure sa zone ; un choix retenu pour la carte prime
      toujours ; `useMapViewPreferences.ts` et son commentaire sont mis d'accord. Tests par vue : carte en
      Auto, zone large puis haute.
- **Gate C** : gate commun web sur `MR/`, `features/tactical/`, `features/match-view/`.

## Piste D — fiche du joueur (Sonnet ; Haiku pour D.1 et D.3)

- [x] D.1 (fait le 2026-11-18, `14e69f8124` sur `bl1118/d` ; le test est rouge sans le filtre) `playerInstantStats.ts` : les débuts de vie ne retiennent que les vies dont la fin dépasse le début
      du match, comme `closedLives`. Test : vies [10, 40] et [41, 200], début à 50, image 100, 0 réapparition.
- [x] D.2 (fait le 2026-11-18, même commit) « Tirs et dégâts » : une ligne par nom d'arme affiché, familles de même nom additionnées
      (`playerInstantView.ts`) ; la clé de liste est celle de la première famille du groupe
      (`FicheTirs.tsx:35`). Test : deux familles au même nom, une ligne, sommes justes.
- [x] D.3 (fait le 2026-11-18, même commit ; `fr.ts` 475 lignes, `en.ts` 470) Nom de grenade de repli : `grenadeRank` au contrat des textes, en français et en anglais ;
      `grenadesCarriedFrom` reçoit le texte ; les deux appelants suivent. Test en anglais.
- [x] D.4 (fait le 2026-11-18, même commit ; le garde-rail garde ses trois interdits et gagne un cliquet contre le retour de `GABARIT_NORMAL` ; le texte `playerScoreLive`, sans lecteur, est retiré) `GABARIT_NORMAL` retiré : une cote nommée pour le corps du bloc « À l'instant » ; `CardGabarit` ne
      garde que la tuile compacte ; branches mortes de `ReplayCountersBadge.tsx` et leurs tests retirés ;
      `cardGabarit.guard.test.ts` suit.
- [!] D.5 (non exécuté : condition non remplie, mesure 0.3 c = 0 lâcher sur 28 540 ; l'écart minimal entre la fin d'une vie et la reprise de sa place est de 101 images) (si 0.3 c > 0) Lâcher daté à fin de vie + 2 : une résolution « la vie qui s'est terminée à l'image
      t − 2 ou t − 1 » prime sur l'occupant courant pour un lâcher (`rosterLogic.ts`, `equipmentGestures.ts`).
      Témoin : nombre de lâchers attribués sur les huit documents réels, avant et après (338 sur 338 avant).
- [!] D.6 (non exécuté : condition non remplie, mesure 0.3 d = 2,2 à 4,3 ms en médiane, 13,4 ms au pire) (si 0.3 d > 50 ms) Séries de la fiche gardées par document et par liste de joueurs, hors de
      l'instance du composant (`playerInstantSources.ts`). Test : deux lectures, même objet.
- [x] D.7 (fait le 2026-11-18, même commit : `model/playerInstantScore.ts`, même fonction pour la tuile et la fiche, garde-rail `scoreSingleSource.guard.test.ts` ; l'infobulle n'écrit un score que pour un joueur publié par le calque de compteurs, choix de l'agent gardé par le pilote : sans calque la tuile montre des totaux de fin de match) (si 0.3 b > 0) Un seul score à l'instant : l'infobulle de la tuile (`ReplayCountersBadge.tsx:146`)
      lit la valeur gardée de la fiche ; quand la garde refuse, elle ne montre pas de score. Si 0.3 b = 0 :
      item clos sur la mesure, l'égalité des deux sources devient un test.
- **Gate D** : gate commun web sur `MR/` ; `fr.ts` et `en.ts` sous 500 lignes.

## Piste E — jetons des téléchargements de fond (Sonnet)

- [x] E.1 (fait le 2026-11-18, `89f9764fd1` sur `bl1118/e` ; erreur typée `wire.ErrNoPublicReadToken` ; 9 comptes
      consentants sur 9 au magasin du poste, et le bureau pose l'accord d'office : aucun poste ne perd ses
      téléchargements ; relu sur pièces) `haloTokensForDrain` prend son jeton à la source de lecture publique de la mise en commun
      (`publicReadTokenSrc.ResolveAny`), jamais dans la liste des profils ; sans source ou sans compte
      consentant : erreur typée, journalisée, sans conseil de recapture. Les trois appelants (catalogue,
      expansion des playlists, mise en file des rejeux) suivent. Libellé de `registry_auth.go:289-291`
      reformulé (premier enrôlement, pas recapture). Tests : compte retenu jamais servi ; aucune source ;
      `registry_catalog_lectures_cgo_test.go` adapté.
- [x] E.2 (fait le 2026-11-18, même commit ; rouge prouvé sur l'ancien code) Garde-rail : `RefreshHaloTokensViaStoreFirst` combiné à un parcours des profils est interdit dans
      `internal/api/wire`, liste d'exemptions datée vide.
- **Gate E** : gate commun Go ; `go test ./internal/api/wire/... ./internal/platform/auth/...` ;
  `go test -tags=integration -p 1 ./internal/sync/...`.

## Piste F — rejeu à la demande et régénération du bureau (Sonnet)

- [x] F.1 (fait le 2026-11-18, `7ae8245722` sur `bl1118/f`) Garde des jetons en liste blanche : dans `registry_replay_generation*.go`, seul
      `tokencache.ResolveFreshPlayerTokens` est admis.
- [x] F.2 (fait le 2026-11-18, `7ae8245722` sur `bl1118/f`) Demande reprise après redémarrage : revalidée par la sonde du match avant tout téléchargement
      (`replayregen/demand_run.go`).
- [x] F.3 (fait le 2026-11-18, `7ae8245722` sur `bl1118/f`) Nouvel essai après échec : les contrôles d'état passent d'abord
      (`service/replay_generation_service.go:117-119`).
- [x] F.4 (fait le 2026-11-18, `7ae8245722` sur `bl1118/f`) Film retiré du disque après un échec de génération non réessayable quand l'archive est désactivée.
- [x] F.5 (fait le 2026-11-18, `7ae8245722` sur `bl1118/f`) Capacité de génération : bureau, ou développement non exposé (`config/deployment.go`) ; test dans
      `deployment_test.go`.
- [x] F.6 (fait le 2026-11-18, `7ae8245722` sur `bl1118/f`) Bouton : pas proposé quand le jeton du joueur demande une reconnexion ; « Réessayer » retiré pour
      un jeton indisponible (web, `replayGeneration/logic.ts`).
- [x] F.7 (fait le 2026-11-18, `7ae8245722` sur `bl1118/f`) Jeton refusé pendant la résolution du film : un nouvel essai avec un jeton frais
      (`tokencache.RetryWithFreshTokens`).
- [x] F.8 (fait le 2026-11-18, même commit ; le test de péremption lisait et décodait chaque fichier : 27 s pour 1 455 fichiers à chaud, 85 s à froid ; il lit maintenant le seul en-tête : 0,25 s, mesuré sur les fichiers réels du poste) Selon D-5 : la file de régénération reprojette, sans film, le fichier tactique périmé d'un rejeu
      dont le film a expiré (`replayregen/scan.go`, `ProjeterRasterTactique`). Tests : film absent et fichier
      au format précédent, fichier refait ; film absent et fichier à jour, rien.
- [x] F.9 (fait le 2026-11-18, même commit ; inventaire : aucun signal ne distingue un film pas encore publié d'un film absent pour un manifeste introuvable ou vide, seul « sans morceau de temps forts » le fait et il est déjà réessayable ; fenêtre de 5 minutes posée ; phrase dédiée « film pas encore disponible » dans les deux langues) Clic juste après la fin d'un match. D'abord l'inventaire, écrit au journal, de ce que la réponse du
      service des films distingue déjà : film introuvable, manifeste sans morceaux, manifeste sans morceau de
      temps forts (`finalise.ErrFilmNonFinalise`, déjà réessayable). Si un de ces signaux reconnaît à lui seul
      un film pas encore publié, il suffit et aucune fenêtre n'est ajoutée. À défaut : la sonde du match porte
      l'heure de fin, et pendant les 5 minutes qui suivent (constante nommée) un film introuvable ne pose pas
      le verdict définitif « film absent » : la demande échoue en « film pas encore prêt », réessayable. La
      synchronisation n'est pas modifiée. Tests : match fini depuis 2 minutes, depuis 2 heures.
- **Gate F** : gate commun Go ; `go test ./internal/replayregen/... ./internal/service/... ./internal/config/...` ;
  gate commun web sur le dossier de F.6.

## Piste G — serveur : contrat et lectures (Sonnet ; Haiku pour G.3 et G.4)

- [x] G.1 (fait le 2026-11-18, `919774dc49` sur `bl1118/g` ; inventaire : les trois champs et `medal_name_id` de l'événement sans aucun lecteur, retirés ; `MedalImageURL` retiré des trois adaptateurs ; `medal_name` de l'événement gardé, hors périmètre, voir Découvertes) Selon D-7 : inventaire fermé au journal (lecteurs de chaque champ de `MatchHighlightEvent` propre
      aux médailles), puis retrait des trois champs, de `match_view_killfeed_medals.go`, de son appel, de
      `MedalImageURL` des adaptateurs de titre s'il n'a plus d'appelant, des tests et fixtures qui les
      portent. `medal_name_id` de l'événement : retiré si l'inventaire ne lui trouve aucun lecteur, gardé et
      écrit au journal sinon.
- [x] G.2 (fait le 2026-11-18, même commit ; aucun chiffre des tests de caractérisation n'a changé ; une escouade aux comptes mesurés à zéro rend maintenant des taux à 0, connus) Selon D-4 : les trois taux par minute de l'Escouade deviennent facultatifs ; « inconnu » quand
      aucun match n'est retenu ; tests de caractérisation mis à jour ; type du graphe côté web.
- [x] G.3 (fait le 2026-11-18, même commit ; le test rougit sur l'ancien parcours) `medalLabelOf` : parcours des langues dans l'ordre de `i18n.Locales()`. Test déterministe.
- [x] G.4 (fait le 2026-11-18, même commit) Compteur et message distincts pour un fichier tactique illisible (`replayartifact/gate.go`).
- [~] G.5 Durée du découpage de la Dominance : couvert par la piste T (T.3), qui fait passer tous les
      lecteurs par une résolution unique au lieu d'ajouter une préférence de plus ici.
- **Gate G** : gate commun Go sur `internal/service/...`, `internal/games/...`, `internal/replayartifact/...` ;
  témoins 0.2 de l'Escouade comparés ; le pilote régénère `openapi.yaml` et `generated.ts` à l'intégration.

## Piste H — couches, accessibilité, clé de requête (Sonnet ; Haiku pour H.4)

- [x] H.1 (fait le 2026-11-18, `ff610e89e8` sur `bl1118/h` ; sept exemptions datées, dont `GamertagCombobox.tsx` et `useAudioUnlock.ts` ajoutées sur pièces ; `sidebarCollapse.ts` et `assetPanel.ts` n'en ont pas besoin : ni Échap ni nom non littéral) Garde-rail des couches élargi à `components/`, `lib/` et aux fichiers `.ts` de `features/` ; il
      reconnaît un nom d'événement non littéral et `case 'Escape'` ; exemptions datées pour
      `titleTooltip.controller.ts`, `tooltipPosition.ts`, `sidebarCollapse.ts`, `useDismissable.ts`,
      `useReplayFullscreen.ts`, `assetPanel.ts`.
- [x] H.2 (fait le 2026-11-18, même commit ; option `escape` ajoutée au mécanisme pour un clic dehors seul ; tests existants verts sans retouche) Les 13 composants migrent vers `useDismissable` : `analysis-pending-badge`, `alert-dialog`,
      `GamertagCombobox` (clic dehors seul), `info-tooltip`, `SessionMultiSelect`, `AboutDialog` (son piège
      de tabulation reste), `HelpSplitButton`, `NavL1` (deux sites), `MultiSelectFilter`,
      `NavL1MobileActions`, `NavL1MobileMenu`, `PlayerSwitcher`, `RailPopover`. Changements voulus, écrits aux
      notes de version : Échap ferme aussi les cinq qui ne l'écoutaient pas ; des couches empilées se ferment
      une par une. Les tests existants de ces composants restent verts sans retouche de leurs assertions.
- [x] H.3 (fait le 2026-11-18, même commit) `MediaPickerDialog` écoute Échap par le mécanisme commun ; `ExperienceDropdown` et `ViewDropdown`
      annoncent leur état ouvert. Tests calqués sur `CoverFlowModal.dismiss.test.tsx`.
- [x] H.4 (fait le 2026-11-18, même commit) `queryKeys.matchReplayContext` porte la langue ; `useMatchReplayContext` la lit ;
      `keys.locale.test.ts` range la clé parmi les localisées.
- **Gate H** : gate commun web sur `components/`, `features/_shared/`, `features/media/`, `lib/query/`,
  `lib/replay/`.

## Piste K — garde-rails et outillage (Haiku ; Sonnet pour K.2 et K.5)

- [x] K.1 (fait le 2026-11-18, `c79566f3e8` sur `bl1118/k2`) Trois entrées mortes retirées de `ALLOWED_FILES` ; test « chaque chemin de la liste existe ».
- [x] K.2 (fait le 2026-11-18, `8cb85efc25` sur `bl1118/k` ; gate : build, vet, `go test ./cmd/mapfond-build/...`
      et `./internal/archlint/...` verts, 133 catalogues et 114 fonds publiés sans faux positif ; relu sur
      pièces par le pilote) Rempart des fonds de carte : retrait de la racine d'installation connue en plus du motif ; dossiers
      à point et à espaces ; garde-rail étendu (partage réseau, `/opt/`, `/mnt/`, `/tmp/`, `/var/`,
      `/Volumes/`). Aucune recuisson.
- [x] K.3 (fait le 2026-11-18, `c79566f3e8` sur `bl1118/k2`) `scripts/disk-hygiene.ps1` : base résolue (amont de la branche courante, sinon branche par défaut
      du dépôt), paramètre `-BaseRef`, « base introuvable » journalisé sans arrêt. Contrôle : simulation sans
      erreur.
- [x] K.4 (fait le 2026-11-18, `c79566f3e8` sur `bl1118/k2`) Test du processus qui tient la base : sortie d'erreur du fils branchée, lecture du « prêt » sous
      délai explicite, sortie du fils jointe à l'échec.
- [x] K.5 (fait le 2026-11-18, `c79566f3e8` sur `bl1118/k2`) `bootstrap_service.go` : trois blocs déplacés sans renommage (`bootstrap_setup_state.go`,
      `bootstrap_ownership.go`, `bootstrap_helpers.go`) ; `buildCapabilities` et `buildFeatureFlags` ne
      bougent pas ; compteurs des trois garde-rails indexés sur le chemin ajustés, somme inchangée.
- **Gate K** : gate commun Go et web selon l'item ; `wc -l` de `bootstrap_service.go` sous 500.

## Piste M — barre de lecture : écran étroit et surimpression (Sonnet)

- [x] M.1 (fait le 2026-11-18 : https://claude.ai/artifact/Q2duSkmGtSf4QLWASqZY83, source gardée sous `.ai/annexes_backlog_1118/maquettes/` ; cotes vérifiées sur le code : commandes fixes 894 px, 1 012 avec « Plan tourné » ; la frise ne tombe pas à 14 px, elle garde 120 px et RECOUVRE les commandes de droite ; le pire cas est 1 280 px, où la colonne est encore à côté du plan ; variantes : A menu « plus » à deux paliers, B deux lignes, D ligne de progression repliée, C barre de 32 px translucide ; rendu non vu dans un navigateur) Planche publiée (page d'artefact, dessinée DANS les cadres validés de la refonte), à trois largeurs
      de fenêtre (1 280, 1 440, 1 536 px) : (a) la barre d'une ligne quand la place manque, deux variantes de
      repli (groupes zoom, capture et vitesse rangés dans un menu ; ou barre sur deux lignes) ; (b) la barre en
      surimpression au Plein cadre, selon la proposition notée au backlog (fine ligne de progression au bas du
      plan pendant la lecture, dépliée au survol du bas, à toute touche et en pause ; fil des éliminations qui
      remonte quand elle est dépliée) et une variante. Aucun code de l'application.
- [x] M.2 (fait le 2026-11-18, `2816593c6d` : deux lignes quand la frise n'a plus sa largeur minimale, marge de bascule réemployée) (porte : variante de repli choisie par le user) Repli de la barre d'une ligne sous le seuil de
      largeur retenu sur la planche (`ReplayTransportCompact.tsx`, `cotesBarreCompacte.ts`) ; la frise ne
      descend jamais sous sa largeur minimale. Tests de logique pure sur le seuil.
- [x] M.3 (fait le 2026-11-18, `2816593c6d` : barre fine et translucide posée sur le plan au Plein cadre, pleine à la pause, au survol, au focus et après un geste ; préférence « Barre de lecture fine ») (porte : surimpression validée par le user sur la planche) Barre en surimpression au Plein cadre,
      telle que validée ; Standard et Cinéma gardent leur barre fixe ; préférence lue une fois, textes dans les
      deux langues. Tests : pause, survol, touche, lecture.
- **Gate M** : gate commun web sur `MR/` ; contrôle à l'écran du user inscrit à l'artefact des vérifications.

## Piste Q — écarts avec les maquettes (Sonnet)

- [x] Q.1 (fait le 2026-11-18 : `.ai/annexes_backlog_1118/ecarts_maquettes.md` ; 8 écarts à corriger, 3 points conformes à une décision datée, 3 points sans dessin posés au user ; liste fermée ci-dessous, soumise au user le 2026-11-18) Instruction en lecture des onze points de l'entrée du backlog, contre les fichiers de maquette
      (`.ai/annexes_refonte_ui_rejeu/maquettes/`) et la grille (`conformite_maquettes.md`) : chaque point classé
      « conforme à une décision datée », « écart à corriger » (avec la cote ou le dessin attendu) ou « sans
      dessin, au user ». Résultat dans `.ai/annexes_backlog_1118/ecarts_maquettes.md` ; les écarts à corriger
      sont inscrits ici en liste fermée avant tout code, et montrés au user.
- [x] Q.2 (fait le 2026-11-22, `9277c6fa73`, un seul commit pour toute la liste ; portes passées : décisions du user du 2026-11-18 sur les écarts, puis sur le prototype des quatre choix) Corrections livrées : a) commandes de la barre d'une ligne en carrés de 30 px au rayon de 8 px, sans bordure ni fond au repos, lecture inchangée (rond plein de 36 px) ; b) capture, export et enregistrement en boutons ordinaires dans la barre d'une ligne, sans cartouche ; c) fenêtre de fiche à 8 px sous le haut du bloc du plan ; e) un seul bloc arrondi pour la toile et la colonne du Standard ; g) « Tirs à la tête » lu sur le compteur du jeu, valeur seule, ligne absente quand le film ne le donne pas ; h) marges du Plein cadre à 16 px sur les côtés et 6 px en bas ; i) lignes du fil qui concernent le joueur suivi sur fond légèrement teinté ; j) barre de lecture sous le bloc arrondi au Cinéma, et au Plein cadre quand la barre fine est éteinte ; k) Suivre / Seul / Masquer en un segment de trois colonnes égales. Écartés par le user : d) défilement entier de la fenêtre de fiche (l'actuel reste) et f) gain en pour-cent dans l'infobulle de « Plan tourné » (la maquette n'est pas suivie).
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

- [x] T.0 (fait le 2026-11-18 : inventaire fermé de 11 sources, 7 qualités, 22 replis, 4 durées ; verdict
      sur les 22 rejeux : l'horloge a raison, prouvé par les positions des joueurs ; annexe sur la branche
      d'intégration) Audit (lecture seule ; base lue sur une COPIE), dans `.ai/annexes_backlog_1118/coup_d_envoi.md` :
      inventaire FERMÉ de chaque source, seuil, qualité, repli, écrivain, lecteur (Go, SQL, web) et commande ;
      mesures sur le parc : écart entre l'horloge de manche du film, le premier mouvement et l'estimation de
      l'API ; matchs sans film, par source restante ; écart de temps de jeu par joueur ; dépendance de la note
      de performance et du LUSR (ADR 0047).
- [x] T.1 (modèle écrit le 2026-11-18 : `.ai/annexes_backlog_1118/modele_coup_d_envoi.md` ; VALIDÉ par le
      user le 2026-11-18, « Oui je valide », avec ses quatre points ; il fait foi sur T.2 à T.6 là où il les précise : rattrapage au démarrage
      par révision ET par commande, montée des deux révisions de notation, rattrapage du poste UNE fois après
      la régénération du plan du décodeur, colonnes retirées seulement si la recette de migration ne
      reconstruit pas la table) Modèle cible, écrit par le pilote et VALIDÉ PAR LE USER avant tout code. Principes proposés :
      (1) une définition : le coup d'envoi est l'instant où l'horloge du jeu démarre ; (2) une source par
      match, dans un ordre fixe écrit une fois : le film quand il existe, l'estimation de l'API sinon,
      « inconnu » sinon, et la source est rangée avec la valeur ; (3) un seul point de résolution, qui rend la
      chronologie du match (coup d'envoi, fin, durée de jeu) : aucun lecteur ne choisit entre deux durées ni
      ne remplace un vide par zéro ; (4) une seule écriture, par un persisteur : coup d'envoi et valeurs
      dérivées (temps de jeu de chaque joueur) dans la même transaction, à la synchronisation comme à
      l'arrivée du film ; (5) une seule commande de rattrapage ; (6) la durée jouable de l'API n'a plus de
      lecteur. Le modèle dit aussi ce que devient chaque ligne de l'inventaire de T.0 (gardée, fusionnée,
      retirée), et s'écrit en ADR (anglais) avec ses garde-rails.
- [x] T.2 (fait le 2026-11-18, `87910a4602` puis `28137cda38` sur `bl1118/t` : `timeline.ResolveKickoff`, ordre fixe film, estimation, inconnu ; trois états du rejeu, un coup d'envoi du film ne redescend jamais faute de rejeu lisible ; valeur du rejeu hors de la fenêtre du match refusée et comptée ; ADR 0050) Résolution unique : type et fonction de la chronologie du match ; les qualités de l'estimation
      deviennent un diagnostic journalisé et compté, plus une valeur que les lecteurs interprètent.
- [x] T.3 (fait le 2026-11-22, `6b2bf79a85` : `timeline.ForMatch` seule règle de lecture, deux fragments SQL, garde-rail `archlint/kickoff_read_guard_test.go`, `playable_duration_seconds` ni lue ni écrite, en-tête servi en `gameplay_duration_seconds` ; témoins avant et après sur 33 matchs : durée de l'en-tête −1 s sur 22 matchs par troncature, Dominance, fiche, accueil et LUSR identiques, 17 prolongations avant et après) Lecteurs : chacun de la liste fermée de T.0 passe par la résolution unique (Dominance comprise :
      l'item G.5 est couvert ici) ; garde-rail : aucune lecture de la durée jouable de l'API ni du coup
      d'envoi brut hors de la résolution.
- [x] T.4 (fait le 2026-11-18 : `persist.KickoffPersister`, une transaction par match ; la synchronisation pose les valeurs sur ses lignes avant l'INSERT ; colonne `real_start_source` amorcée par la migration ; garde-rail `archlint/kickoff_write_guard_test.go` ; relecture adverse faite, 1 bloquant et 5 constats corrigés) Écriture unique : le persisteur écrit coup d'envoi, source et temps de jeu ensemble (règles
      anti-corruption du dépôt, jamais d'écriture directe) ; le report du film et la synchronisation
      l'appellent tous deux.
- [x] T.5 (fait le 2026-11-18 : `levelup backfill-kickoff`, simulation comprise ; rattrapage au démarrage par révision `KickoffRev`, liée aux révisions des notes ; `cmd/backfill_t0`, `backfill_t0_film`, `backfill_time_played` et `analysis/spawn_detection.go` supprimés avec leurs tests et leurs 38 paires de la baseline ; mesure sur copie : 2 045 matchs, 1 447 film, 587 estimation, 11 inconnus, second passage sans écriture, 50 rejeux retirés : 50 coups d'envoi gardés) Une commande `levelup` de rattrapage avec simulation ; les trois `cmd/backfill_*` sont supprimés
      avec leurs tests (aucun code mort).
- [!] T.6 (reporté par décision du user du 2026-11-22 : la régénération du parc se fait une seule fois, après la version suivante du décodeur ; le rattrapage du coup d'envoi et le rejeu des notes partent au premier démarrage de la version ; voir le registre des reports) (porte : accord du user, serveur arrêté, copie de la base faite) Rattrapage du poste, puis rejeu de
      la note LUSR si T.0 l'a jugé nécessaire. La prod reçoit le même geste à la publication.
- Lien avec le plan du décodeur : si T.0 montre que l'horloge de manche date mieux le coup d'envoi que le
  premier mouvement, le décodeur publie le coup d'envoi lu sur l'horloge et retire le détecteur de mouvement
  (item du plan du décodeur) ; LevelUp lit le même champ avant comme après.
- **Gate T** : modèle validé par le user ; `cd apps/go-api && go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/platform/duckdb/...`,
  code de sortie 0 ; `go test ./internal/archlint/...` ; inventaire de T.0 entièrement statué ; relecture
  ciblée des écritures (R.2).

## Piste S — schéma d'une base joueur et tests du paquet DuckDB (Sonnet)

- [x] S.1 Test d'égalité dans les deux sens (« synchronisation puis migrations » contre « migrations
      seules ») sur types, défauts et `NOT NULL` ; écart listé dans `.ai/annexes_backlog_1118/schema_joueur.md` ;
      forme des bases existantes relevée sur une COPIE d'une base joueur du poste.
- [x] S.2 La définition qui s'écarte de la forme de référence est alignée, pour les bases neuves seulement ;
      aucune migration d'une base existante ; si l'alignement demande de modifier une étape de migration déjà
      jouée en production, arrêt et retour au user.
- [x] S.3 L'exemption de `csr_pipeline_e2e_integration_test.go` est retirée si le test d'égalité la rend
      inutile ; sinon elle garde sa date et son motif.
- [x] S.4 Migrations neuves indirectes : les cinq sites qui partent d'un fichier joueur absent
      (`pool_attach_swap_integration_test.go:116`, `pool_shared_reader_integration_test.go:71`, `:125`, `:186`,
      `pool_swap_multititre_integration_test.go:103`) reçoivent la base migrée copiée avant `GetOrOpen` ;
      `pool_migration_test.go` (dont l'objet est la migration) passe sur la liste d'exemptions datée ; le
      garde-rail reconnaît `GetOrOpen` et un nom de fonction de migration pris en valeur ; les deux
      `ForTarget` et l'`ExecScript` (quelques étapes, gain nul) sont nommés hors périmètre dans son en-tête.
      Témoin : durée et nombre de tests du paquet, avant et après.
- **Gate S** : `cd apps/go-api && go test -tags=integration -p 1 ./internal/sync/... ./internal/platform/duckdb/...`,
  code de sortie 0, échecs cherchés par `^--- FAIL:`.

## Étape I — intégration (pilote)

- [x] I.1 (fait : trois livraisons fusionnées dans `release/v8.3.0` au lieu d'une seule, amendement écrit au journal du 2026-11-18 ; la troisième le 2026-11-22) Recalage sur `origin/release/v8.3.0` ; fusion des sous-branches dans `chore/830-backlog-1118` ;
      `openapi.yaml` et `make generate-types` une seule fois.
- [ ] I.2 Témoins APRÈS comparés à 0.2 : écarts listés, chacun rattaché à un item ou déclaré défaut.
- [x] I.3 (fait le 2026-11-23 pour le changelog et les notes de version, en français et en anglais ; le dossier `C:/t` des caches dédiés a été vidé le 2026-11-22, voir journal) `docs/CHANGELOG.md` et notes de version de la 8.3.0, en français et en anglais (départs, ramassages,
      « Seul » et « Suivre », Escouade, Échap) ; `.ai/thought_log.md` ; caches dédiés rendus.
- **Gate I** : `make gate-push` ; `cd apps/go-api && go test -tags=integration -p 1 ./internal/sync/... ./internal/platform/duckdb/...` ;
  `make go-api-lint` ; `cd apps/web && npx vitest run` complet, cache de types purgé avant `make check-types`.

## Étape R — relecture et clôture

- [x] R.1 (fait le 2026-11-23 : entrées traitées closes ou reformulées, découvertes du plan ajoutées, reports inscrits au registre) Backlog mis à jour par un agent Haiku : entrées traitées déplacées au tableau des entrées closes ;
      une entrée dont la porte du user n'est pas passée garde sa ligne, avec la date de la demande ; entrées
      du décodeur selon D-8 ; registre des reports.
- [x] R.2 (les relectures ciblées des jetons, de la file et du coup d'envoi sont couvertes par les axes 1 et 2 de cette relecture, qui les a examinés en priorité et jugés sains) (fait le 2026-11-23 : relecture adversariale du code livré par Fable, aucun P0 ni P1 de code ; quatre P2 et six P3 consignés au backlog. Un P1 de confidentialité corrigé le jour même : des annexes versionnées nommaient des joueurs par leur pseudo et portaient un chemin du poste ; libellés neutres, commit `9bb45958da`.)
      Une ronde de relecture adversariale du diff complet (skill `adversarial-review`, Fable) et deux
      relectures ciblées (Sonnet) : jetons et file (pistes E et F), écritures en base (piste T) ; P0 et P1
      corrigés en une ronde, P2 au backlog seulement s'ils valent le coup.
- [x] R.3 (fait le 2026-11-23 : 16 lignes inscrites dans l'artefact de vérifications du user (collection `verifs`), 14 faisables dès maintenant et 2 à venir ; la ligne sur les pictogrammes de ramassage est réécrite. Les 5 lignes « à venir » de la montée du décodeur relèvent de son plan, pas de celui-ci.)
      Artefact des vérifications du user : départ du joueur A à 5:00 sur le match `38900a5e` ; ligne de
      ramassage ; « Seul » puis « Suivre » ; sortie du Plein cadre ; orientation Auto dans la Tactique ; barre
      d'une ligne au Cinéma, fenêtre d'environ 1 440 px, disposition en colonne.
- [ ] R.4 Tous les items statués ; outillage de mesure retiré ou documenté ; fusion dans `release/v8.3.0` ;
      CI lue après la poussée.

## Journal d'exécution

- 2026-11-23 — relecture finale et vérifications à l'écran. La relecture adversariale du code livré ne relève aucun P0 ni P1 de code ; un P1 de confidentialité (pseudos de joueurs et chemin du poste dans des annexes versionnées) est corrigé (commit `9bb45958da`). Les vérifications à l'écran sont inscrites dans l'artefact de vérifications du user : 16 lignes pour ce plan, 14 faisables dès maintenant, 2 à venir. R.2 et R.3 passent en `[x]`. Restent ouverts : I.2 (témoins après), R.4 (clôture, fusion dans `main`, CI) ; la validation à l'écran reste due au user.

- 2026-11-23 — CLÔTURE DES DOCUMENTS. Code livré en trois fois dans `release/v8.3.0` (première : K.2, E, H, C, G, B, A, F ; seconde : D, K, S ; troisième : piste T, M.2, M.3 et Q.2) ; Q.2 statué (`[x]`), I.1 et I.3 faits (changelog et notes de version des deux langues), R.1 fait (backlog et registre).
  T.6 passe en `[!]` : le user a décidé le 2026-11-22 de ne régénérer le parc qu'une fois, après la version suivante du décodeur ; le rattrapage du coup d'envoi et le rejeu des notes partent au premier démarrage de la version.
  Restent ouverts : I.2 (témoins après), R.2 (relecture), R.3 (vérifications à l'écran du user), R.4 (clôture, fusion dans `main`, CI).
  Grille de conformité aux maquettes mise à jour (`.ai/annexes_refonte_ui_rejeu/conformite_maquettes.md`).

- 2026-11-18 — PREMIÈRE LIVRAISON dans `release/v8.3.0` (`f5c0b92cec`, poussée) : K.2, E, H, C, G, B, A, F. AMENDEMENT
  écrit : le plan prévoyait une seule fusion à la fin ; cette livraison partielle sert le plan du fond Relief, qui
  attendait C.4 et K.2, et met le code sous les yeux du user pour sa validation à l'écran. Vérifications du pilote
  sur l'intégration, deux passes : types, lint, lints d'imports et de couleurs, contrat de l'API, compilation et
  `vet` verts ; `go test ./...` 228 paquets verts (archlint avec un délai de 40 minutes) ; tests d'intégration en
  `-p 1` verts (synchronisation, DuckDB, persist, replayregen, replayartifacts) ; suite complète du web : 14 379
  réussis, un vrai rouge corrigé (type du fil homonyme d'un schéma du contrat), huit dépassements de délai sous
  charge rejoués seuls et verts. M.1 refait en prototype interactif après le retour du user (« aucune animation
  ni rien, je vois pas du tout ce qu'on vise »), même lien. Restent à cueillir sur l'intégration : D, K (K.1, K.3,
  K.4, K.5) ; en cours : T (écriture), S.

- 2026-11-18 — M.1 rendu, lu par le pilote (feuille de style, balises et attributs énumérés, texte entier) et
  publié ; choix du user attendu (porte de M.2 et M.3). Seconde passe de vérifications : suite complète du web
  14 379 réussis, 9 échecs : 1 vrai (type `MedalSprite` du fil homonyme d'un schéma du contrat, renommé en
  `MedalSpriteCut`, `c24b59aa99`) et 8 dépassements de délai de garde-rails qui parcourent les fichiers, sous
  charge, à rejouer seuls.

- 2026-11-18 — piste K rendue (K.1, K.3, K.4, K.5), relue, commitée (`c79566f3e8` sur `bl1118/k2`) : toute la piste
  K est faite. Q.1 rendu : liste fermée des écarts de maquette soumise au user avec trois questions (icônes de
  cadre, curseur de volume, bouton caméra pendant un suivi). M.1 (planche de la barre de lecture) lancé.

- 2026-11-18 — piste D rendue, relue, commitée (`14e69f8124`, au-dessus de A et B). Pistes S (`C:/t/bl18-s`) et
  Q.1 (lecture seule) lancées. Seconde passe des vérifications lourdes en cours sur l'intégration (K.2, E, H, C,
  G, B, A, F ; plafond des fonctions longues à 245).

- 2026-11-18 — piste F rendue après quatre compléments du pilote (échec de jeton mémorisé périmé quand le jeton
  revient ; phrase du film pas encore prêt ; balise du champ `stale` ; coût du décompte borné), relue, commitée
  (`7ae8245722`). À faire par le pilote à l'intégration : régénérer le contrat de l'API (la balise de `stale` a
  changé). `TestCompactPassesNuit` avait dépassé son délai sous charge : rejoué seul, il passe en 19 s. Le user a
  doublé la borne commune (8 agents d'écriture, dit à la session du fond Relief et inscrit au plan maître) :
  cette session en tient 3 au plus pour l'instant. Pistes D et K lancées (`C:/t/bl18-d`, au-dessus de A et B ;
  `C:/t/bl18-k2`). Écart au tableau des modèles : K.1, K.3, K.4 (prévus Haiku) faits par l'agent Sonnet de K.5.

- 2026-11-18 — piste A rendue après trois compléments du pilote (durée de « a quitté la partie », taille de
  `killFeedLogic.ts` ramenée de 591 à 567 lignes, test de la marge), relue, commitée (`d99966fcaa`). Sur le vrai
  rejeu du match signalé par le user : fiche du joueur A « mort » à l'image 9700, « a quitté la partie » aux
  images 9786, 9900, 9940 et à la dernière ; plus de ligne « mort » à 5:00. Sa place d'écriture va à la session
  du fond Relief (deux écrivains chacun). Vérifications lourdes du pilote sur l'intégration (K.2, E, H, C, G) :
  types, lint, compilation et `vet` verts ; suite complète du web 14 315 réussis, 2 échecs sans défaut de code
  (plafond des fonctions longues à abaisser de 247 à 245 ; un garde-rail hors délai sous charge, à rejouer seul).

- 2026-11-18 — piste B rendue, relue, commitée (`0aa6debb61`) ; elle sera cueillie sur l'intégration à la fin des
  vérifications lourdes du pilote, qui tournent dans ce dossier (on ne modifie pas un arbre pendant son gate).
  La place d'écriture libérée va à la session du fond Relief, comme convenu (deux places).

- 2026-11-18 — piste G rendue, relue, commitée (`919774dc49`), cueillie sur l'intégration, où la branche de
  version a été ramenée (documents seuls). Écart au plan relevé : le paquet des contrats est `./contracttest/...`,
  pas `./internal/contracttest/...`. Piste T, côté écriture (T.2, T.4, T.5), lancée dans `C:/t/bl18-t`.
  AMENDEMENT DE MODÈLE, motif écrit : cette moitié de la piste T part sur OPUS et non Sonnet. Motif : elle
  écrit dans la base partagée des matchs (table critique, défaut ART), pose un rattrapage au démarrage et
  monte les deux révisions de notation ; une erreur y coûte une base. Les lecteurs (T.3) resteront sur Sonnet,
  après elle. Le tableau « aucun item ne demande Opus » est donc faux pour T.

- 2026-11-18 — piste C rendue, relue sur pièces (garde-rail de l'orientation réécrit sans être affaibli : plus aucun
  appel à la règle Auto dans la page du rejeu), commitée (`c5acd8f47e`) et cueillie sur l'intégration
  (`6253b1d5cb`). Un test de garde (`match-view/xuidMeta.guard.test.ts`) a dépassé son délai sous charge et passe
  seul : à revoir dans la suite complète du pilote. Piste B lancée (`C:/t/bl18-b`).

- 2026-11-18 — ÉTAPE 0 CLOSE (Gate 0 passée : annexes `temoins_avant.md` et `mesures.md` sur la branche
  d'intégration ; (a) confirmée, 0 sur 875 ; seuils statués : D.5 et D.6 non exécutés, D.7 exécuté). Pistes K.2,
  E et H cueillies sur `chore/830-backlog-1118` (`910fb2e805`, `3fded4ffb9`, `405ca31220`). Piste A lancée
  (`C:/t/bl18-a`). Restent au pilote pour ces pistes : suite complète du web, archlint entier et tests
  d'intégration de la synchronisation, à passer quand la marche à blanc du décodeur aura rendu la machine.

- 2026-11-18 — piste E rendue, relue et commitée (`89f9764fd1`). Reste à passer par le pilote à
  l'intégration : `go test -tags=integration -p 1 ./internal/sync/...` et la suite entière d'archlint (non
  menée à terme par l'agent, machine chargée). Piste F lancée (`C:/t/bl18-f`). Découverte : sur une instance
  sans aucun compte consentant, le refus est journalisé en avertissement à chaque mise en file d'un rejeu.
- 2026-11-18 — T.0 clos ; modèle du coup d'envoi (T.1) validé par le user. T.2 à T.5 partent dès qu'un
  écrivain se libère ET que la piste G est intégrée (mêmes fichiers : `service/match_view_shared_build.go`,
  `domain/match_view.go`). Lien inscrit au plan du décodeur : sa piste A publie le coup d'envoi lu sur l'horloge.
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
- 2026-11-18 — K.2 rendu, relu sur pièces et commité (`8cb85efc25`, branche `bl1118/k`). La session du fond
  Relief n'a plus d'écrivain : pistes E, G et H lancées (worktrees `C:/t/bl18-e`, `-g`, `-h`), soit 4
  écrivains Sonnet avec C. Écart au tableau des modèles, voulu : G.3, G.4 et H.4 (prévus Haiku) sont faits
  par l'agent Sonnet de leur piste, pour ne pas ouvrir un dossier de travail de plus. L'agent de G régénère
  le contrat de l'API dans son dossier pour ses contrôles de types ; le pilote le régénère une seule fois à
  l'intégration. Les gates d'intégration lourds (`-tags=integration`, suites complètes) restent au pilote.
- 2026-11-22 — REPRISE. `C:/t` vidé le 22/11 à 03:03 par la tâche de ménage du disque (72 h d'inactivité) : dossiers
  de travail et travail non commité perdus, branches intactes. T.3 et Q.2 rejoués depuis les transcripts des agents
  coupés puis terminés par deux agents neufs, dans `Scripts/LevelUp-wt-t` et `LevelUp-wt-m`. Q.2 fait (`9277c6fa73` :
  items 0, a, b, c, e, h, i, j, k, g ; choix de l'agent : au Plein cadre, barre fine éteinte, la barre va sous le bloc
  comme au Cinéma). T.3 fait (`6b2bf79a85`). TROISIÈME LIVRAISON fusionnée dans `release/v8.3.0` après gate lourd
  vert sur l'intégration : typage, lint, vitest complet (14 568 tests), imports entre features, contrat de l'API,
  `go build`, `go vet`, `go test -tags=integration -p 1 ./...` (228 paquets, code 0). Découvertes : les sections
  rencontres et némésis de l'onglet d'équipe diffèrent entre deux exécutions du même code (ordre non déterministe,
  préexistant) ; `ReplayPage.row.test.tsx` à 508 lignes ; `cmd/investigate_matches` supprimé avec la colonne qu'il
  lisait. Restent : vérifications à l'écran du user (barre de lecture et corrections du rejeu), T.6 après la
  régénération du décodeur, étapes I et R.
- 2026-11-18 — Q.2-g DÉCIDÉ par le user (« ok ») : la ligne « Tirs à la tête » de la fiche lit le compteur daté du
  tableau du jeu (valeur à l'instant = dernière valeur de la courbe cumulée), sans mention ni bulle ; ligne
  absente quand le film ne donne pas le compteur pour le joueur ; l'ancienne estimation et sa note sont
  supprimées. Ajouté à l'agent de Q.2 en cours, après ses huit corrections. La barre de lecture (M.2, M.3) est
  rendue, relue et commitée (`2816593c6d` sur `bl1118/m`) : écart assumé avec le prototype, la barre fine
  redevient pleine à la pause, au survol, au focus et après un geste ; forme fine obtenue par une réduction à
  80 % (lisibilité des petits chiffres à juger par le user à l'écran).
- 2026-11-18 — DÉCISION DU USER sur le prototype des quatre choix : « tout proposé sauf le 3. » Donc, à écrire en
  Q.2 : (i) lignes du fil du joueur suivi sur fond légèrement teinté (token, sans bordure à gauche), distinct de
  la ligne choisie ; (ii) au Cinéma, la barre de lecture SOUS le bloc arrondi du plan ; (iii) Suivre / Seul /
  Masquer en un segment de trois colonnes égales, l'actif plein. Le volume (3) reste comme aujourd'hui : curseur
  toujours visible. LISTE FERMÉE DE Q.2, dans l'ordre d'écriture, un seul écrivain, après M.2 et M.3 : Q.2-a
  boutons carrés de la barre d'une ligne ; Q.2-b capture et export sans cartouche ; Q.2-c fenêtre de fiche à
  8 px sous le haut du plan ; Q.2-e bloc arrondi unique du Standard ; Q.2-h marge du Plein cadre à 16 px ;
  Q.2-i fond teinté du joueur suivi ; Q.2-j barre du Cinéma sous le bloc ; Q.2-k segment de la fiche. Q.2-g
  (tirs à la tête) attend la réponse du user sur la lecture du compteur du jeu. Q.2-d et Q.2-f : écartés.
- 2026-11-18 — prototype des quatre choix publié, lu en entier par le pilote avant publication :
  https://claude.ai/artifact/PpcJfwonVvFY3WhNj93etb (source : `.ai/annexes_backlog_1118/maquettes/rejeu-quatre-choix.html`).
  Quatre bascules « Aujourd'hui / Proposé » : fond teinté des lignes du joueur suivi, barre du Cinéma sous le
  bloc, volume en icône seule, segment Suivre / Seul / Masquer. Jamais ouvert dans un navigateur par l'agent :
  le rendu est à juger par le user. Limite dite au user : au choix de la barre, le gain de hauteur du plan est
  faible dans le prototype.
- 2026-11-18 — DÉCISIONS DU USER sur les écarts de maquette (Q.2), message du 2026-11-18 : « pour les corrections,
  ok avec tes recos sauf pour 6, j'en veux pas du tout, 7 on a bien un (i) pour expliquer que c'est estimé ? Pour
  12 et 14 j'en sais rien je vois pas de quoi ça a l'air. Pour le 17 je vois pas non plus [...]. Pour les trucs
  plus tard je te donne mon go d'avance ». Lecture du pilote, redite au user : RETENUS : boutons carrés de la
  barre d'une ligne (Q.2-a), capture et export sans cartouche (Q.2-b), fenêtre de fiche à 8 px sous le haut du
  plan (Q.2-c), bloc arrondi unique du Standard (Q.2-e), marge du Plein cadre à 16 px (Q.2-h). ÉCARTÉS : le
  défilement de toute la fenêtre de fiche (Q.2-d : l'actuel reste), le gain en pour-cent dans l'infobulle de
  « Plan tourné » (Q.2-f : refusé, la maquette n'est pas suivie). EN QUESTION : tirs à la tête (Q.2-g) : il n'y
  a aucune bulle d'information aujourd'hui ; proposition du pilote à valider : lire le compteur du jeu lui-même
  (`scoreTimeline.players[].stats`, série dans le temps, accord de 99,85 % avec l'API) au lieu des éliminations
  dont le coup fatal est mesuré, ce qui retire l'estimation. SUR PROTOTYPE (le user ne peut pas juger sans
  rendu) : fond teinté des lignes du joueur suivi, barre du Cinéma sous le bloc, volume en icône seule, segment
  Suivre / Seul / Masquer : prototype interactif commandé (`C:/t/maquettes/rejeu-quatre-choix.html`). Sans
  changement : pictogrammes des cadres, bouton caméra relevé pendant un suivi, cinq écarts de forme des bases
  joueur (nommés au test), colonnes mortes du registre (backlog). Q.2 s'écrit après M.2 et M.3 (mêmes fichiers,
  un seul écrivain). GO D'AVANCE du user pour « les trucs plus tard » : le rattrapage du poste (T.6) et la
  régénération du parc du plan du décodeur ; le pilote prévient quand même avant de lancer (serveur arrêté,
  2 h 30, mémoire) ; la fusion dans `main` reste annoncée avant, et la version attend la 3D et les temps forts.
- 2026-11-18 — barre de lecture : le user CONFIRME dans cette session (« je t'ai déjà validé C+B non ? ») le
  choix reçu par relais : B (deux lignes quand la place manque) et C (barre fine et translucide en
  surimpression au Plein cadre). Portes de M.2 et M.3 ouvertes ; agent Sonnet lancé dans `C:/t/bl18-d` (branche
  `bl1118/m`, partie de l'intégration), avec la consigne de placer ses ajouts de préférence et de textes hors
  des lignes que `feat/fonds-relief` modifie, pour une fusion sans conflit. Piste T : ronde de corrections
  commitée (`28137cda38`), T.3 (lecteurs) lancé avec témoins avant et après. CI de la seconde livraison verte.
- 2026-11-18 — SECONDE LIVRAISON fusionnée et poussée dans `release/v8.3.0` (`709786d758` : pistes D, K, S).
  Gate lourd sur l'intégration : typage, lint, suite complète du site (14 406 tests), imports entre features,
  `go build`, `go vet`, `go test -tags=integration -p 1 ./...` (229 paquets, code 0), contrat de l'API à jour.
  La CI de la première livraison avait ROUGI (job de la baseline de tests) : 4 tests supprimés avec leur code
  par la piste G restaient dans la baseline ; retirés dans cette livraison (`4b76bacbc2`). Le gate local ne
  joue pas ce contrôle : la piste T, qui supprime des tests, doit retirer ses paires (consigne envoyée).
  Relecture adverse de la piste T rendue : 1 bloquant, 5 à corriger, ronde de corrections en cours ; l'ADR
  du coup d'envoi devient 0050 (0049 pris par le plan du fond Relief).
- 2026-11-18 — barre de lecture (M.2, M.3) : la session du fond Relief RELAIE une phrase du user reçue chez
  elle, « pour la maquette du lecteur en superposition je valide l'option C+B » (C : barre fine et translucide en
  surimpression au Plein cadre ; B : deux lignes quand la place manque). Relais, pas un message du user dans
  cette session : la porte de M.2 et M.3 ne s'ouvre qu'à sa confirmation ici. L'écriture attend de toute façon
  l'intégration du D.3 de l'autre plan (`settings/replayPreferences.ts`, nœud N25).
- 2026-11-18 — piste T, écriture (T.2, T.4, T.5) rendue et commitée sur sa branche (`87910a4602`) ; relecture
  adverse des écritures lancée (agent neuf, lecture seule) AVANT la migration des lecteurs (T.3). Mesure sur la
  copie : 2 045 matchs, sources film 1 447, estimation 587, inconnue 11 ; 13 coups d'envoi changent ; 9 873
  temps de jeu changent (médiane −8 s) ; second passage : 0 écriture. Écarts au modèle : les trois colonnes
  mortes restent en base (DuckDB refuse le retrait sans reconstruire la table) ; `playable_duration_seconds`
  reste écrite tant que la Dominance la lit.
- 2026-11-18 — pistes D, K (K.1, K.3 à K.5) et S rendues, relues sur pièces, commitées sur leurs branches et
  cueillies sur l'intégration (`e7f8ff4ae4`, `bddd4751a6`, `b976137ffc`). S : 16 lignes d'écart de forme avant,
  11 colonnes alignées pour les bases neuves, 5 écarts résiduels nommés (voir Découvertes), exemption de
  `csr_pipeline_e2e` retirée, garde-rail des migrations neuves élargi. Piste T : écriture en cours (Opus).
  Plan du décodeur : pistes J et A lancées dans deux dossiers de travail du clone.
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

- 2026-11-18 (T) — les colonnes `t0_quality`, `film_match_start_ms` et, à terme, `playable_duration_seconds` du
  registre n'ont plus d'écrivain ou n'en auront plus : DuckDB refuse `DROP COLUMN` sur une table à clé primaire
  et index, le retrait demande de reconstruire la table. À porter au backlog à la clôture (R.1).
- 2026-11-18 (T) — deux rejeux (`ed3b6497`, `dd29957c`, matchs de 21 et 24 s) publient un coup d'envoi après la
  fin du match : refusés par la résolution et comptés. À recouper avec le plan du décodeur (coup d'envoi par
  l'horloge).
- 2026-11-18 (T) — une publication de rejeu qui change un temps de jeu de quelques secondes ne rejoue pas les
  notes (limite écrite dans l'ADR 0050).
- 2026-11-18 (T) — les tests unitaires de `sync/replayartifacts` écrivent sous `<racine>/data/cache/replays/halo_5/`
  un fichier dont le nom porte un octet nul ; le renommage échoue sous Windows. Préexistant.

- 2026-11-18 (S) — cinq écarts de forme restent entre une base joueur née de la synchronisation et une base
  née des migrations (`performance_score` FLOAT contre DOUBLE, table et vue ; défaut de `sync_meta.updated_at` ;
  défaut FALSE de `is_with_friends` et `is_excluded`). Les fermer demande de modifier une étape de migration
  déjà jouée en production, ce que S.2 interdit : ils sont nommés et datés dans le test de forme. Trois options
  écrites dans `.ai/annexes_backlog_1118/schema_joueur.md` ; le pilote retient « laisser », soumis au user.

- 2026-11-18 (intégration) — `internal/mapcatalog`, `TestAddOverlayEntryConcurrentNePerdPasDEntree` : instable, 1
  échec sur 10 passages sur le code d'AVANT le plan. Quand le verrou d'écriture ne se pose pas à temps, le code
  force le passage et une écriture concurrente du catalogue est perdue : c'est un vrai défaut, pas seulement un
  test fragile. À porter au backlog à la clôture (R.1), priorité moyenne.
- 2026-11-18 (intégration) — sous charge, huit garde-rails du site qui parcourent tous les fichiers dépassent le
  délai de 5 s d'un test dans une suite complète (ils passent seuls en moins de 30 s au total).

- 2026-11-18 (Q.1) — une décision datée du user n'est pas réalisée dans le code : le fond teinté des lignes du
  joueur suivi dans le fil (décision D-5 de la refonte, 2026-11-15) ; seul le gras de la fiche ouverte existe.
  Soumis au user avec la liste de Q.2.
- 2026-11-18 (Q.1) — hors des onze points : au Cinéma et au Plein cadre la barre de lecture est dans le bloc du
  plan alors que les maquettes la dessinent dessous ; « Suivre / Seul / Masquer » sont trois boutons séparés
  là où les maquettes dessinent un segment de trois colonnes. Soumis au user.
- 2026-11-18 (K) — `docs/COMMANDS.md` (vers la ligne 1678) cite encore `origin/feat/v75` comme base par défaut
  d'un outil de comparaison de rejeux. À porter au backlog à la clôture (R.1).
- 2026-11-18 (K) — sur ce poste, `go list` a échoué une fois avec « Ressources système insuffisantes » sur un
  fichier lisible par ailleurs : état de Windows, pas du code.

- 2026-11-18 (F) — après un échec de jeton, la page redevient « générable » dès que le jeton est présent, même
  s'il ne se rafraîchit pas : un second clic échouera de même. Sans gravité ; noté.
- 2026-11-18 (F) — les lecteurs du fichier tactique ont quitté `sync/replayartifacts` pour `replayartifact`
  (un garde-rail interdit tout décodage JSON dans le premier) ; à recouper avec la piste T, qui écrit dans
  `sync/replayartifacts`.

- 2026-11-18 (F.9) — la synchronisation pose le verdict « film absent » sans fenêtre après la fin du match. MESURÉ
  sur la copie de la base : défaut ÉCARTÉ. 0 match sur 251 vus moins de 5 minutes après leur fin porte le
  verdict ; 2 sur 503 parmi ceux vus plus tard dans les 180 jours ; 614 sur 699 au-delà (films expirés). Aucune
  fenêtre n'est ajoutée à la synchronisation. Un match (`58e6f72a`, décembre 2023) porte le verdict alors que
  ses 33 morceaux de film sont à l'archive : à porter au backlog à la clôture (R.1).

- 2026-11-18 (G.1) — `medal_name` des événements de médaille du contexte du match n'a plus aucun lecteur de
  production, et le site ignore ces événements (il lit le document de rejeu). À porter au backlog à la clôture
  (R.1) : retirer le champ, voire les événements de médaille du contexte. L'ADR 0011 cite encore
  `MedalImageURL`.

- 2026-11-18 (C.3) — la scène de la page du rejeu est reconstruite et la toile redessinée à chaque rendu : les
  hooks de calques rendent des objets non mémoïsés. C'est ce qui garde l'horloge écrite après une bascule de
  cadre en pause, par effet de bord, et c'est un coût de rendu probable. À porter au backlog à la clôture (R.1).
- 2026-11-18 (C.4) — quatre mesures de largeur coexistent (`useBodyWidth`, l'effet de taille de « Occupation du
  terrain », `useReplayViewport`, `useContentWidth`) : la règle des trois copies appellerait à les réunir. À
  porter au backlog à la clôture (R.1).

- 2026-11-18 (0.3) — 128 corps supprimés sans mort appartiennent à un joueur qui a une vie ensuite (ni fin de
  manche, ni départ, ni bot) : ils donnent aujourd'hui une ligne « mort » non prouvée dans le fil. Cause non
  instruite (retour d'un joueur parti, réapparition forcée ?). À porter au backlog à la clôture (R.1).
- 2026-11-18 (0.3) — score à l'instant : dans 3 845 cas l'écart entre l'infobulle et la fiche n'est qu'un
  décalage d'une image ; 781 joueurs ont un écart de plus d'une seconde, en général de 100 points (même
  famille que la réponse de 100 points hors arbre du plan du décodeur, piste B).
- 2026-11-18 (H) — le lint du web sort 4 avertissements dans trois fichiers non touchés
  (`tooltip-panel.tsx`, `useGamertagSuggestions.ts`, `MediaAudioConfigButton.tsx`).

- 2026-11-18 (T.0) — hors du système du coup d'envoi, validé comme tel par le user : ce que font les notes
  quand le temps de jeu d'un joueur manque (600 s supposées pour la note de performance, poids et ordre des
  joueurs partis pour le LUSR ; replis R11 à R14 de l'audit). À porter au backlog à la clôture (R.1).
- 2026-11-18 (T.0) — 437 matchs ont une durée de l'API qui diffère de « fin moins début » de plus d'une
  seconde ; non instruit. À porter au backlog à la clôture (R.1).
- 2026-11-18 (K.2) — le rempart des fonds de carte prend un nom de fichier suivi d'un texte et d'un chemin
  relatif pour un dossier : « x.module vers any/globals » devient « globals », le nom du module est perdu.
  Défaut d'avant côté Windows, étendu aux chemins Unix par l'assouplissement. Rien ne fuit ; le message perd
  de l'information. Non porté au backlog (aucun fond publié concerné).
