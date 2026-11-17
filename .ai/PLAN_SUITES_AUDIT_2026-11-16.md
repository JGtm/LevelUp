# Plan — suites de l'audit des versions 8.2.0 à 8.3.0 (2026-11-16)

État : écrit, relu contre la grille `plan-review`. Toutes les décisions sont tranchées (trois séries de
réponses du user, 2026-11-16, ci-dessous) ; quand une réponse contredit une recommandation de D-1 à D-8, la
réponse fait foi. Attend le go d'exécution.
Source : `.ai/AUDIT_VERSIONS_8_2_0_A_8_3_0_2026-11-16.md` (8 P1, 4 familles de P2). Demande du user
(2026-11-16) : traiter tous les points de la revue sans attendre, amender l'ADR 0048, valider une à une les
données servies sans lecteur.

Objectif : que le code de la 8.3.0 ne porte plus ni mesure close, ni donnée sans lecteur non décidée, ni
troisième copie sans helper, ni exception sans date, et que chaque point soit tenu par un garde-rail.
Critère de succès : chaque constat du registre a un statut `[x]`, `[~]` ou `[!]` ; `make gate-push` vert ;
aucune sortie servie ne change hors des retraits décidés en D-2 (témoins de l'étape 0).

Branche : `chore/830-suites-audit`, depuis `release/v8.3.0` (`71cf7db427`). Une seule branche, un commit par
item ou par groupe d'items d'une piste ; fusion dans `release/v8.3.0` à la fin, avant la refonte du rejeu et
le rejeu 3D, qui s'appuient sur ce code. Aucune PR par étape. Version : 8.3.0. Aucune montée de brique.
Contrat d'exécution : skill `plan-execution`.

Hors périmètre : justesse des formules, couverture des tests, invariants d'écriture en base, code des dépôts
des briques (axes non passés par l'audit) ; les trois plans en attente (refonte du rejeu, rejeu 3D, temps
forts) et les évolutions du décodeur, sauf la ligne ajoutée à l'étape E.9.

## Décisions à trancher avant le go

Chaque décision porte une recommandation ; les étapes ci-dessous sont écrites sur la recommandation.

- **D-1 — ADR 0048, plan des murs.** Recommandé : D-3 devient « le plan existe sous deux formes tirées du même
  maillage : une image, qui est le fond "Plan", et un vecteur dans le sidecar, dessiné par-dessus le fond
  "Rendu" ». D-7 dit que `-plan-seul` refait le vecteur ET l'image plan. « État courant » remis à jour.
- **D-2 — données servies sans lecteur.** Une ligne par donnée ; « garder » veut dire inscrite sur la liste
  datée de l'étape B.3, avec son lecteur prévu.

  | # | Donnée | Depuis | Lecteur prévu | Recommandé |
  |---|---|---|---|---|
  | a | Rejeu : touches, dégâts, bilan des tirs (`hits`, `damage`, `shotStats`) | 8.2.1 / 8.3.0 | fiche à l'instant de la refonte (lot T, D-22) | garder |
  | b | Rejeu : médailles LUES DANS LE FILM (`medals`). Les médailles affichées aujourd'hui viennent des événements du match, pas de ce champ | 8.2.1 | fiche à l'instant, temps forts (D-1) | garder |
  | c | Rejeu : actions de score (`scoreActions`) | 8.2.1 | fiche à l'instant (score) | garder |
  | d | Rejeu : horloge de manche et règles (`roundClock`, `rules`) | 8.2.1 | aucun plan ne les cite | à trancher : garder avec un lecteur nommé, ou retirer du document servi |
  | e | Réapparitions du rejeu : 9 champs de diagnostic (`family`, `model`, `model_date`, `matches_on_map`, `candidates`, `discarded_deaths`, `sample_step_ms`, `z`, `outside_zone`) | 8.3.0, non publiée | aucun (déjà journalisés) | retirer |
  | f | Réapparitions du rejeu : motif du refus (`reason`, `message`, 5 messages en 2 langues) | 8.3.0 | aucun : le calque n'affiche rien quand la prévision est refusée | AFFICHER le motif dans l'état vide du calque |
  | g | Réapparitions adverses : `map_id`, `matchs_filtres`, `matchs_retenus`, `matchs_sans_equipe`, `zone.volume_index`, `sans_zone` | 8.3.0 | aucun (déjà journalisés) | retirer |
  | h | Réapparitions adverses : `raison` par répartition | 8.3.0 | aucun | AFFICHER dans l'état vide, comme f |
  | i | Séries temporelles : Mainmise par match (`match_rows[].lead_share`, `lead_changes`) | 8.2.0 | aucun (le graphe lit l'agrégat par mois) | retirer |
  | j | Sidecar tactique : `source` et `debut_de_manche` par réapparition | 8.2.0 | aucun | retirer ; le format du sidecar monte, la régénération au schéma 125 déjà due le couvre (D-8) |
  | k | Sidecar tactique : comptes `film`, `repli_ecart`, `debuts_de_manche` | 8.2.0 | aucun | garder et leur donner un lecteur : c'est le compteur du repli (étape A.5) |
  | l | Note de performance : 7 champs de détail de `objectiveblend.Result` | 8.2.0 | tests seuls | rendre non exportés |

- **D-3 — rejeux `.json` non compressés.** TRANCHÉ (user, 2026-11-16) : la lecture est retirée en 8.3.0, pas
  en 8.4.0 (étape B.2). La recommandation d'origine (critère écrit, retrait en 8.4.0) est abandonnée.
- **D-4 — interrupteur `LEVELUP_SYNC_RESOLVE_ASSETS`** (échu le 2026-09-01, actif par défaut). Recommandé : retiré.
- **D-5 — outil de mesure des réapparitions (`spawn-model evaluate`).** Recommandé : retirer la section ligne de
  vue (m2) et le témoin à constantes figées (m1) ; garder `fit`, `--variantes`, « par mode » (m3), « mort
  contre réapparition » (m4) et l'accord, parce que les évolutions du décodeur redemanderont la mesure.
- **D-6 — outil des fonds (`mapfond-build`).** Recommandé : retirer `-rogne-publie` (geste unique, déjà joué,
  contraire à D-7) ; garder `-avis` (il porte la règle de silhouette par carte) ; garder UN rapport, le TSV avec
  `-avant-dir`, retirer le rapport Markdown `-rapport`. Retirer les 14 leviers par carte sans donnée ni test et
  `sourceNavmesh` (contraire à D-2 de l'ADR) ; garder `sansNavmesh` et `navmeshMode`, cités par l'ADR.
- **D-7 — prédicat « la zone a une forme ».** Recommandé : une zone a une forme si elle a un contour OU une
  partie d'au moins 3 sommets (la règle la plus large, celle du calque des tueurs), partout. L'étape 0 compte les
  zones du catalogue concernées ; à 0, rien ne change à l'écran.
- **D-8 — régénération.** Recommandé : ce plan ne lance aucune régénération. Le retrait D-2 j part avec la
  régénération au schéma 125 déjà inscrite au registre, lancée par le user.

## Réponses du user (2026-11-16)

| Décision | Réponse | Effet sur le plan |
|---|---|---|
| D-1 | accord | D.1 inchangée |
| D-2 a, b, c | ne pas trancher « garder » avant d'avoir comparé chaque statistique du film à ce qui la donne déjà ailleurs ; quand deux sources existent, garder la plus fiable | étape 0.5 neuve ; B.3 attend son résultat |
| D-2 d | horloge : garder seulement si elle est plus fiable que l'horloge actuelle du rejeu ; règles : usage à établir (question du user : devaient-elles servir au classement des modes ?) | étape 0.5 ; B.4 attend son résultat |
| D-2 e, g | accord : retirer les champs de diagnostic des deux routes | A.4 inchangée |
| D-2 f, h | accord : afficher le motif du refus | E.7 inchangée |
| D-2 k | accord : garder les comptes du repli et leur donner un lecteur | A.5 inchangée |
| D-2 i, j | à réexpliquer | C.2 et A.6 en attente |
| D-2 l | (non commentée ; technique, sans effet visible) | C.3 inchangée |
| D-3 | pas de 8.4.0 : le retrait se fait en 8.3.0 | B.2 réécrite |
| D-4, D-5 | accord | F.1, A.3 inchangées |
| D-6, D-7, D-8 | à réexpliquer | D.2, D.3, E.5, A.6 en attente |
| Trois règles de doctrine (G.2) | accord | G.2 : à écrire telles quelles |
| Commit du registre, du plan et du journal | accord | fait sur `chore/830-suites-audit` |

Secondes réponses du user (2026-11-16, soir) :

| Décision | Réponse | Effet sur le plan |
|---|---|---|
| D-2 d, règles | à garder : elles peuvent confirmer des lectures (réapparitions) | `rules` inscrit sur la liste datée de B.3, lecteur prévu « confirmation des lectures de réapparition » ; l'étape 0.5 mesure quand même la part des matchs où chaque règle est écrite |
| D-2 j | accord | A.6 inchangée |
| D-6 | accord, et demande d'aller plus loin : rationaliser et revoir l'outil | étape D.8 neuve |
| D-7 | les zones nommées sont définies par le jeu, pas par LevelUp | E.5 : la règle unique suit le jeu, toute zone que le jeu définit (contour ou morceaux) compte ; aucune règle maison |
| D-8 | la régénération des rejeux se fait à la toute fin | H.4 neuve : dernier geste, après la fusion |
| D-2 i | non tranchée (le user ne situait plus la Mainmise ; réexpliquée) | C.2 en attente |
| Replis | remarque du user : « je n'aime pas fonctionner avec des replis alors qu'on ne fait que de la lecture » | PROPOSITION à confirmer (une remarque n'est pas une validation) : A.5 devient le RETRAIT du repli sur l'écart de 4 à 15 s (0,08 % des réapparitions du parc régénéré, 1 artefact resté à un ancien schéma) ; un match sans lecture du film n'a alors pas de réapparitions, dit comme tel ; les comptes de D-2 k partent avec lui |

Troisièmes réponses du user (2026-11-16, soir) : toutes les décisions sont tranchées.

| Décision | Réponse | Effet sur le plan |
|---|---|---|
| D-2 i | accord : la Mainmise par match n'est plus envoyée aux Séries temporelles ; le user veut juger le format de la Mainmise en conditions réelles | C.2 inchangée ; élément `suites-audit-mainmise-format` ajouté à l'artefact des vérifications ; H.2 le reprend |
| Repli des réapparitions | accord : retiré | A.5 réécrite ; D-2 k devient un retrait (les comptes du repli partent avec lui, format 8 du fichier tactique, avec A.6) |

Inventaire fait le 2026-11-16 pour l'étape 0.5 (sources en double, sans mesure de fiabilité) :

| Statistique du film (champ du rejeu) | Source déjà en place |
|---|---|
| Médailles datées (`medals`) | événements du match (`highlight_events`), lus par `collectMedalEvents` : c'est ce que le rejeu affiche |
| Actions de score (`scoreActions`) | score personnel du match (total, sans date) ; courbe du score du rejeu (`scoreTimeline`) |
| Touches, dégâts, bilan des tirs (`hits`, `damage`, `shotStats`) | totaux du match donnés par l'API (tirs, touches, dégâts infligés et subis), sans date ni détail par adversaire |
| Horloge de manche (`roundClock`) | horloge du rejeu calculée côté web (`model/replayClock.ts`, `useReplayClock`) |
| Règles de la variante (`rules` : délai de réapparition, pénalités, retour du drapeau, tirs alliés) | aucune ; le classement des modes lit le nom de la paire mode / carte (skill `halo-modes`), pas ces règles ; la jauge de retour du drapeau lit les périodes du film, pas `flagReturnS` |

## Règles d'exécution

- Skill `plan-execution`. Une piste = un agent Sonnet dans son worktree, SANS jonction vers `data/` ; étape 0
  et relecture finale par Fable ; une seule ronde de relecture, en fin de plan. Les agents ne poussent pas.
- Ordre : étape 0, puis les pistes A à F en parallèle (elles ne partagent aucun fichier, sauf la liste ci-dessous),
  puis G, puis H. Dans une piste, l'item N est clos (gate passé) avant N+1.
- Fichiers partagés, édités par le pilote seul à l'intégration : `apps/go-api/api/openapi.yaml`,
  `apps/web/src/lib/api/generated.ts`, `docs/CHANGELOG.md`, `.ai/thought_log.md`, `CLAUDE.md`.
- Statuts : `[x]` fait, `[~]` couvert ailleurs (référence), `[!]` non traité (justification). Aucune case vide à
  la clôture. Toute trouvaille hors périmètre va dans « Découvertes », sans correction.
- Gate commun d'un item Go : `cd apps/go-api && GOWORK=off go build ./... && go vet ./... && go test ./<paquets
  touchés>/...`. Gate commun d'un item web : `make check-types && npx vitest run <dossiers touchés>` puis
  `node tools/lint-cross-feature-imports.mjs`.
- Un helper neuf vient avec son garde-rail (test qui interdit l'ancien littéral) et la migration de TOUTES les
  copies dans le même commit.
- Reprise de session : lire ce fichier (cases), `git log --oneline release/v8.3.0..chore/830-suites-audit`,
  puis l'entrée la plus récente de `.ai/thought_log.md`.

## Effort par piste

| Piste | Items | Effort | Risque |
|---|---|---|---|
| 0 témoins | 4 | rapide | aucun |
| A réapparitions Go | 8 | moyen | faible (rien de publié) |
| B rejeu servi Go | 5 | rapide | faible |
| C indicateurs et couches Go | 9 | moyen | moyen (C.6 et C.7 déplacent une règle : témoins obligatoires) |
| D fonds | 7 | moyen | faible (aucune recuisson) |
| E front | 12 | lourd | moyen (E.1 et E.9 touchent beaucoup de fichiers) |
| F exceptions | 7 | rapide | moyen sur F.4 seul (lecture d'une note) |
| G, H intégration et relecture | 7 | moyen | aucun |

## Étape 0 — témoins et vérification sur pièces (pilote, Fable)

Go du user le 2026-11-16 (soir) : exécution en mode orchestrateur (le pilote n'écrit pas le code), Sonnet par
défaut, Haiku quand c'est possible, Opus si nécessaire, relecture finale par un contexte Fable frais, périmètre
fermé, tout report est soumis au user qui juge s'il va au backlog.

- [x] 0.1 Constats P1 rouverts par le pilote avant le plan (registre, « Vérification ») ; les P2 sont rouverts
      par l'exécuteur de chaque piste avant de coder (règle 4 du contrat), qui signale tout constat déplacé.
- [x] 0.2 Témoins AVANT. Empreintes des 339 fichiers de `map_backgrounds/` (114 sidecars, 114 fonds, 111 images
      plan), de `spawn_model.json` et de `map_fond_reglages.json`, rangées hors dépôt (dossier `temoins` de la
      session). ADAPTATION dite : pas de témoin pris sur un serveur lancé. Lancer le serveur de cette branche sur
      les données locales déclencherait ses migrations de démarrage, ses recalculs et son suivi en direct ; les
      témoins des sorties servies sont donc des tests de caractérisation écrits par l'exécuteur AVANT sa
      modification (sortie figée sur les fixtures du dépôt), qui doivent rester verts après, hors retraits
      décidés.
- [x] 0.3 Compteurs de départ (2026-11-16, hors tests) : copies de la Mainmise 4 ; `polygon.length >= 3` 6 ;
      `ui-sans-serif` 7 ; `arcTo(` 8 ; `buildPlayers(` 9 ; `.tsx` de `features/` qui écoutent Échap ou le clic
      dehors 22 ; `ctx.rotate(` et `rotate(${` 7 ; paquets du décodeur importés par `internal/port` 3
      (`objectiveevent`, `playerposition`, `replay`) ; accesseurs de `reglages.go` 42, 906 lignes. Fonctions à
      plus de 5 paramètres et de plus de 80 lignes de code sous `apps/web/src` : comptées par l'exécuteur de la
      piste E avant E.11 (script à l'API TypeScript).
- [x] 0.4 D-7 : 3 927 zones au catalogue (`map_callouts.json`) ; 0 zone sans contour ; 10 zones ont un contour et
      des morceaux. La règle unique ne change donc rien à l'écran.

Décisions du pilote à l'étape 0, consignées :
- E.1 : 22 `.tsx` de `features/` écoutent Échap ou le clic dehors, pas 5. Le périmètre de E.1 reste celui du
  constat (les 5 fichiers du rejeu et le hook du shell). Le garde-rail interdit le motif sous
  `features/match-replay` et gèle le compte du reste de `features/` (ne fait que descendre). Les 17 autres
  fichiers : report soumis au user.
- B.2 : parc local 1 455 rejeux compressés, 0 brut ; prod 0 brut (journal). Le dossier des rejeux contient aussi
  `rasters/*.json` (fichiers tactiques), qu'une suppression par motif emporterait. La migration de suppression
  des bruts est donc ABANDONNÉE : risque réel, bénéfice nul sur les parcs connus. Un brut resté sur une
  installation ancienne n'est plus lu ; les notes de version le disent.
- [x] 0.5 Comparaison des sources (réponse du user à D-2 a à d), sur 20 matchs du parc lus un par un, et pour
      les champs du schéma 125 sur 3 matchs générés hors du parc (`--garder-documents`), aucune régénération :
      - médailles du film contre médailles des événements du match : compte par joueur, médailles propres à
        chaque source, écart de date ;
      - somme des actions de score par joueur contre le score personnel du match (là où le document dit la
        lecture complète) ;
      - bilan des tirs du film (tirs, touches, dégâts infligés et subis) contre les totaux de l'API, pour les
        joueurs à lecture jugée suffisante ;
      - horloge de manche du film contre l'horloge calculée du rejeu : écart en secondes aux débuts et fins de
        manche et en prolongation ;
      - règles de la variante : part des matchs où chaque champ est écrit ; usages possibles relevés (délai de
        réapparition pour le repli de A.5, tirs alliés, pénalités), sans rien brancher.
      Sortie : un tableau « source, accord, cas où l'une a raison contre l'autre, source recommandée » dans
      `.ai/annexes_suites_audit/sources.md`, soumis au user. B.3 et B.4 n'avancent qu'après sa réponse ; les
      autres pistes n'attendent pas.
- [x] 0.6 D-3 : 0 rejeu non compressé dans le parc local (1 455 `.json.gz`) ; 0 en prod d'après le journal du
      2026-11-14 (non relu sur le serveur : aucune opération sur le VPS dans ce plan).
- Gate : témoins écrits, compteurs notés dans ce fichier, tableau 0.5 remis au user.

## Piste A — réapparitions, côté Go (8.3.0 non publiée : aucun contrat à tenir)

- [x] A.1 Ligne de vue retirée : `analysis/tactical/ligne_de_vue.go`, `VarVisibles`, `PorteeVueM`, `Visibilite`,
      paramètre `vue` de `Variables`, `AvecVisibilite`, `ReglagesEchantillons.Vue` ; option `--ligne-de-vue`,
      `fonctionVue`, `obstaclesDuDocument`, section m2 ; clé `ennemis_a_vue_40m` de `spawn_model.json`
      (révision du fichier montée) ; `TestObstacles_Voit` ; `docs/COMMANDS.md` et sa version FR.
      Gate : `git grep -nwE "NouveauxObstacles|AvecVisibilite|VarVisibles|ligne-de-vue" -- apps docs data` vide ;
      prévisions des trois matchs témoins identiques au centième.
- [x] A.2 La CLI appelle les helpers d'`analysis/tactical` au lieu de les recoder : `fini` -> `Point3.Fini`,
      `evenementDe` -> une fonction exportée unique, `fonctionZone` -> `IndiceDeZone`, littéral `2` ->
      `CandidatsMinPrevision`. Gate : `git grep -nE "func (fini|evenementDe|fonctionZone)\b" -- apps/go-api` vide.
- [x] A.3 D-5 appliquée : section m1 et ses constantes de témoin retirées avec leurs tests.
- [x] A.4 D-2 e et g appliquées : champs retirés de `domain/replaydoc/spawn_forecast.go`,
      `domain/tactical_reapparitions.go`, des services et de leurs tests ; le journal `slog` les garde.
      D-2 f et h : les champs de motif restent (lecteur à l'étape E.7).
- [x] A.5 Repli sur l'écart de 4 à 15 s RETIRÉ (accord du user) : `EcartReapparitionMinMs` / `MaxMs`,
      `SourceRepliEcart`, la branche de `analysis/tactical/reapparitions.go`, `sansLecturesDuFilm` de la CLI et
      leurs tests ; les comptes `film`, `repli_ecart`, `debuts_de_manche` du fichier tactique partent avec lui
      (même montée de format que A.6). Un match sans lecture du film n'a pas de réapparitions : dégradation
      nommée et journalisée, la Tactique et le calque du rejeu affichent leur état vide. La définition écrite
      dans `spawn_model.json` est remise au vrai au prochain `fit`.
      Gate : `git grep -nE "SourceRepliEcart|EcartReapparitionM(in|ax)Ms|repli_ecart" -- apps/go-api` vide ;
      sur les trois matchs témoins (tous à lecture du film), réapparitions identiques à l'étape 0.
- [x] A.6 D-2 j appliquée : `source` et `debut_de_manche` retirés du sidecar, `TacticalRasterSchemaVersion` 7 -> 8,
      note au registre des reports : « part avec la régénération au schéma 125 ».
- [x] A.7 `SpawnFamilySet.TitleSlug()`, `SchemaVersion()` et leurs champs retirés avec leur test.
- [x] A.8 Règles de jeu pures sorties de `service/tactical_service_lectures.go` vers `analysis/tactical`
      (`spawnsDuJoueur`, `sidecarsAuxVies`, `routesDesVies`, `reapparitionsAdversesDuMatch`) : le service traduit
      `domain` vers les entrées d'analysis, la règle se teste sur les types d'analysis ; garde
      `archlint/no_local_spawn_depart_test.go` mis à jour.
- Gate de piste : `go test ./internal/analysis/tactical/... ./internal/service/... ./cmd/levelup/...`.

## Piste B — rejeu servi, côté Go

- [x] B.1 Fraîcheur du document décidée par le service : `port.ReplayService.GetReplay` rend l'état de cadence ;
      `handlers/replay.go` n'importe plus `replaybuild`. Test de service qui échoue si la règle s'inverse.
      Gate : `grep -n "replaybuild" apps/go-api/internal/api/handlers/replay.go` vide.
- [x] B.2 D-3 appliquée, retrait en 8.3.0 : `replayartifact` ne connaît plus que `.json.gz` (`Locate`, `Read`,
      détection par octets magiques, `RemoveLegacy`, `HasLegacySibling`, compteur, `ReplayArtifactCandidates`) ;
      `replaybuild/artifact_compress.go`, sa commande et leurs tests retirés ; un `.json` restant vaut « rejeu
      absent » et se régénère (la 8.3.0 régénère de toute façon tout le parc au schéma 125). Aucune
      suppression de fichier (décision de l'étape 0). Notes de la 8.3.0.
      Gate : `git grep -nE "RemoveLegacy|HasLegacySibling|PARC MIXTE" -- apps/go-api` vide.
- [x] B.3 Liste datée des champs du document sans lecteur web : un test web lit les clés de premier niveau de
      `replayDocumentSchema.ts` et exige, pour chacune, un lecteur hors schéma et normalisation, ou une entrée
      `{ clé: "AAAA-MM-JJ : lecteur prévu" }`. Entrées de départ : les champs que le user garde après le
      tableau de l'étape 0.5. La liste ne fait que rétrécir ; une montée du décodeur qui ajoute un champ sans
      lecteur doit l'y inscrire.
- [~] B.4 Champs que le user ne garde pas après l'étape 0.5 : retirés du document servi (`replaydoc`,
      `replayview`, contrat, types web, schéma et normalisation web), sans toucher au décodeur.
- [x] B.5 Types du décodeur dans `internal/port` : garde-rail qui gèle les paquets importés à l'état de
      l'étape 0.3 (plafond qui ne fait que descendre).
      `[!]` prévu pour le retrait lui-même : trois types traversent `port/` et 9 fichiers les consomment ; c'est
      un chantier à part, à ranger avec les évolutions du décodeur (ligne au registre des reports).
- Gate de piste : `go test ./internal/service/... ./internal/api/... ./internal/replayartifact/...` ; témoin du
  rejeu identique octet pour octet, hors B.4.

## Piste C — indicateurs et couches, côté Go

- [x] C.1 Mainmise : un helper de lecture (charge, journalise sauf capacité absente, pose les deux pointeurs)
      appelé par les quatre services ; garde-rail qui interdit le bloc recopié.
      Gate : `grep -rn "share, changes := v.Share, v.Changes" apps/go-api/internal --include=*.go | wc -l` = 1.
- [x] C.2 D-2 i appliquée : champs retirés de `TimeseriesMatchRow`, du service et de son test.
- [x] C.3 D-2 l appliquée : `objectiveblend.Result` n'exporte que `Score` ; commentaire remis au vrai.
- [x] C.4 Harnais `sync/performance_objective_simulation_research_test.go` et `..._report_research_test.go`
      retirés (578 lignes).
- [x] C.5 Agrégat « par minute » : une seule fonction Go dans `analysis/` appelée par `trends` et par
      `teammates` (`paceSums` disparaît) ; jeu de cas partagé lu par le test Go et par le test de
      `lib/charts/perMinute.ts` (modèle : `testdata/meneur_cas.json`).
- [x] C.6 Famille d'objectif : `objectiveFamilyFromRaw` et `objectiveColumnValues` passent dans
      `analysis/narrative` ; test de parité avec le SQL `objectiveFamilyCaseSQL` sur une table de cas (DuckDB en
      mémoire), un cas par famille et un cas sans objectif.
- [x] C.7 Révisions de notation : l'ordre et l'indépendance des deux étapes passent de
      `api/wire/registry_scoring_revision.go` à `service/scoringrevision` ; le câblage ne fait plus qu'injecter
      les deux étapes. Tests déplacés avec la règle.
- [x] C.8 `PlayerProfileService` : interface déplacée dans `port/`, `handlers/player_profile.go` n'importe plus
      `progression/profile` pour son service.
- [x] C.9 `port/media_indexer.go` : paramètres de `ResetAndReindex` et `ScanAllMedia` regroupés en une
      structure ; `*jobs.Store` remplacé par l'interface étroite que l'indexeur utilise.
- Gate de piste : `go test ./internal/service/... ./internal/analysis/... ./internal/sync/...
  ./internal/api/...` ; témoins de la vue du match et des Séries temporelles identiques, hors C.2.

## Piste D — fonds de carte

- [x] D.1 D-1 appliquée : ADR 0048 amendé (D-3, D-7, « État courant », amendement daté) ; ligne 0048 de
      `CLAUDE.md` corrigée par le pilote.
- [x] D.2 D-6 appliquée, options : `-rogne-publie` retiré avec `rogne_publie.go` et son test ; rapport Markdown
      retiré ; `docs/COMMANDS.md` EN et FR.
- [x] D.3 D-6 appliquée, leviers : champs, accesseurs et câblage des leviers sans donnée retirés ; les
      accesseurs restants passent par UN accesseur générique ; `reglages.go` sous 500 lignes ; garde-rail :
      tout levier déclaré a au moins une entrée dans `map_fond_reglages.json` ou une exemption datée.
- [x] D.4 Ancres : `ancresDuCatalogue` appelée par les trois sites (`cuisson.go`, `cuisson_forge.go`,
      `plan_seul.go`). Gate : `grep -n "\[3\]float64{o.Pos.X" apps/go-api/cmd/mapfond-build/*.go | wc -l` = 1.
- [x] D.5 Carte Forge sans maillage : `cuisson_forge.go` pose la dégradation nommée ; test qui fixe l'ensemble
      des fonds sans plan (trois, nommés). Les deux sidecars Thunderhead ne sont réécrits que si la commande
      rend une image identique à l'empreinte de l'étape 0 ; sinon `[!]` et ligne au registre (prochaine cuisson).
- [x] D.6 Rotation du fond : une fonction d'application pour le canvas et une pour le CSS dans
      `lib/replay/orientation.ts`, appelées par le rejeu, la Tactique et « Occupation du terrain » ; garde-rail
      étendu aux trois vues (aucun `ctx.rotate(` ni `rotate(${turns` hors de ce fichier).
- [x] D.7 Commentaires de `main.go` et `reglages.go` qui citent `libs/` et `licence_test.go` remis au vrai.
- [x] D.8 Outil revu dans son ensemble (demande du user), après D.2 à D.4, sans changer une sortie :
      - trois gestes nommés et documentés, et eux seuls : cuire des cartes, refaire le plan sans recuire,
        produire le tableau avant / après ; toute option restante se rattache à l'un des trois ;
      - `main.go` : les options groupées par geste, validation des combinaisons en un point ;
      - réglages par carte : la table des leviers, l'accesseur générique et la validation dans trois fichiers
        distincts, chacun sous 500 lignes ;
      - `docs/COMMANDS.md` EN et FR : une section par geste, avec la commande exacte.
      Gate : empreintes des 114 sidecars et des images identiques à l'étape 0 après une cuisson de 3 cartes
      témoins (une native, une Forge, une sans maillage) et un `-plan-seul` ; `wc -l` de chaque fichier < 500.
- Gate de piste : `go test ./cmd/mapfond-build/... ./internal/mapfondsidecar/...` ; empreintes des 114 sidecars
  et des images identiques à l'étape 0, hors D.5.

## Piste E — front du rejeu et de la Tactique

- [x] E.1 Fermeture d'une couche (Échap, clic dehors) : `useDismissable` sort de
      `components/shell/_filter_pills/_hooks.ts` vers un hook partagé, avec le cas « couche intérieure ouverte » ;
      appelé par `ReplayFeedFilterMenu`, `ReplaySpeedMenu`, `ReplayViewPanel`, `ReplayPlayerPanel`,
      `ReplayMediaLightbox` et par le shell. Garde-rail : aucun `addEventListener('pointerdown'`,
      `addEventListener('mousedown'` ni `key === 'Escape'` dans un `.tsx` de `features/match-replay` ; compte
      gelé pour le reste de `features/` (décision de l'étape 0).
- [x] E.2 `ReplayFeedFilterMenu` : navigation clavier du menu dans un hook testé seul ; composant sous 80 lignes
      de code.
- [x] E.3 Rectangle arrondi : une fonction dans `lib/replay` (repli sans `ctx.roundRect` compris), appelée par
      `respawnForecastLegend`, `respawnForecastPaint`, `export/overlayPaint` ; garde-rail sur `arcTo(`.
- [x] E.4 Les 4 signatures à plus de 5 paramètres ajoutées dans la fenêtre passent à un objet d'options
      (`drawProjectilesLayer`, `drawTintBar`, et les deux rectangles, réglés par E.3).
- [x] E.5 D-7 appliquée : un prédicat et une énumération des contours d'une zone dans `lib/replay`, appelés par
      `calloutsPaint`, `killerZone`, `respawnForecastPaint`, `reapparitions.logic`, `TacticalReapparitionsZones` ;
      garde-rail sur `polygon.length >= 3` hors de ce fichier.
- [x] E.6 Règle de rognage au plan sortie de `ReplayPage.tsx` vers un fichier de logique, avec son test (plan
      choisi et image présente, plan choisi sans image, rendu).
- [x] E.7 D-2 f et h appliquées : le calque de probabilité et la lecture « Réapparitions adverses » affichent le
      motif du refus dans un état vide (bloc affiché, pointillé, aucune action impossible), en fr et en,
      couleurs par tokens sémantiques (skills `frontend-patterns`, `color-tokens`).
- [x] E.8 Police du canvas : une constante lue par les 7 sites ; garde-rail sur `ui-sans-serif`.
- [x] E.9 `buildPlayers` : appelé une seule fois (`model/replayModel.ts`) ; `ReplayTeams`, `useSlotIdentity`,
      `equipmentUsageLogic`, `killerZone` reçoivent `model.players` ; garde-rail qui rend vrai le commentaire
      « nulle part ailleurs ».
      `[~]` prévu pour le découpage de `ReplayCanvas`, `ReplayPlayerPanel` et `LayersSection` : la refonte du
      rejeu les réécrit. Une ligne est ajoutée à `PLAN_REFONTE_UI_REJEU_2026-11-15.md` : ces trois fonctions
      sortent de la refonte sous 80 lignes de code.
- [x] E.10 `useReplayPlayback` (110), `useReplayFlagCarries` (102), `useReplayVehicles` (186) : ramenées sous
      80 lignes de code par extraction, ou exemption commentée si le corps est une table.
- [x] E.11 Deux compteurs gelés sous `apps/web/src` (test) : fonctions à plus de 5 paramètres, fonctions de plus
      de 80 lignes de code ; valeurs de l'étape 0.3 moins ce que cette piste retire ; ne font que descendre.
- [x] E.12 Les 8 `eslint-disable no-explicit-any` des tests de graphes remplacés par des types.
- Gate de piste : `make check-types`, `make test-web` COMPLET, `node tools/lint-cross-feature-imports.mjs`.

## Piste F — exceptions

- [x] F.1 D-4 appliquée : `LEVELUP_SYNC_RESOLVE_ASSETS` retiré avec sa branche ; entrée retirée de
      `lecturesEnvAutorisees`.
- [x] F.2 Les trois lectures « à porter par AppConfig » passent par `internal/config`
      (`registry_replay_notify.go`, `engine_highlight_events.go`, `snapshot_readiness_eval.go`) ; entrées
      retirées. Le contrôle de date de l'allowlist exige le format `AAAA-MM-JJ`.
- [x] F.3 `effective_hp_copies_test.go` : entrée datée, le test exige la date.
- [x] F.4 `Q24LUSRHistory` (`queries_career_encounters.go`) : lecture brute de `match_skill_rank` revérifiée ;
      filtre `rating_type` et tie-break `written_at` ajoutés, test sur un match à double ligne ; l'entrée
      d'allowlist dit ce qui est vrai. Skill `db-schema` avant d'écrire.
- [~] F.5 `[~]` prévu : les deux entrées de `pool_no_shared_handle_test.go` sont datées, le risque est écrit,
      le code est déplacé à l'identique ; aucun changement.
- [~] F.6 `[~]` prévu : plafond de surface du décodeur (316 -> 335), chaque marche datée et justifiée ;
      `CLAUDE.md` cite le garde-rail au lieu des nombres (pilote).
- [~] F.7 `[~]` prévu : liste des imports entre fonctionnalités (79 -> 99), seuil du cliquet à 0 ; aucun changement.
- Gate de piste : `go test ./internal/archlint/... ./internal/config/... ./internal/platform/duckdb/...`.

## Étape G — intégration (pilote)

- [x] G.1 Fusion des pistes dans `chore/830-suites-audit` ; `openapi.yaml` et `make generate-types` une fois.
- [x] G.2 Doctrine, trois ajouts soumis au user avant écriture : (1) « une donnée servie ou persistée a un
      lecteur, ou une ligne datée qui nomme son lecteur prévu » (`CLAUDE.md`) ; (2) « à la clôture d'un plan,
      l'outillage d'une mesure close est retiré » (skill `plan-execution`) ; (3) « une version de LevelUp monte
      le décodeur une fois, la régénération du parc se lance à la publication » (`CLAUDE.md`).
- [ ] G.3 Notes de la 8.3.0, `docs/CHANGELOG.md`, `.ai/BACKLOG.md` (P2 traités retirés), registre des reports
      (A.6, B.2, B.5, D.5 le cas échéant), registre de l'audit (colonne « Décision » remplie), `thought_log`.
- [x] G.4 Comparaison des témoins APRÈS aux témoins de l'étape 0 : écarts listés, chacun rattaché à une
      décision D-2 ou déclaré défaut.
- Gate : `make gate-push` ; `cd apps/go-api && go test -tags=integration ./internal/sync/...` ; `make go-api-lint`.

## Étape H — relecture et clôture

- [x] H.1 Une ronde de relecture adversariale du diff complet (skill `adversarial-review`, Fable) ; P0 et P1
      corrigés, P2 au backlog. Fait le 2026-11-17 : 0 P0, 2 P1, 5 P2 ; voir le journal.
- [ ] H.2 Contrôles à l'écran du user, dans l'artefact des vérifications : menus du rejeu (fermeture par Échap et
      clic dehors), calque de probabilité et son état vide, Tactique « Réapparitions adverses », fonds Plan et
      Rendu tournés sur les trois vues, format de la Mainmise (élément déjà posé le 2026-11-16).
- [ ] H.3 Tous les items statués ; fusion dans `release/v8.3.0` (CI verte).
- [ ] H.4 Régénération du parc des rejeux (schéma 125, format 8 du fichier tactique) : dernier geste, lancé par
      le user après la fusion ; ce plan ne la lance pas.

## Journal d'exécution

### 2026-11-17 — pistes A, C, D, E, F closes (branches `sa/a`, `sa/c`, `sa/d`, `sa/e`, `sa/f`, non fusionnées)

Chaque compte rendu a été rouvert sur pièces par le pilote (commits, `git status` vide, motifs visés comptés).

- 0.5 : tableau remis au user (`.ai/annexes_suites_audit/sources.md`). Aucun vrai doublon ; réponse attendue pour
  B.3 et B.4.
- Piste A (8 commits). Écarts : A.3, l'accord est retiré aussi (il comparait le film au repli, retiré par A.5) ;
  A.6, `source` retiré et format 7 -> 8, mais `debut_de_manche` GARDÉ : il est lu
  (`analysis/tactical/spawns_joueurs.go`, définition « toutes les vies ») ; le constat D-2 j de l'audit était
  faux sur ce champ. `spawn_model.json` passe à la révision 2.
- Piste C (12 commits). C.5 : la fonction unique est appliquée aux dégâts ; frags, morts et assistances par
  minute de l'Escouade gardent leur comportement (un match sans durée compte au numérateur), question posée au
  user. C.8 : 15 types de réponse du profil déplacés dans `domain/` pour que l'interface entre dans `port/`.
- Piste D (8 commits). 339 fichiers de fonds identiques à l'étape 0 ; cuissons témoins natives identiques.
  D.5 : les deux sidecars Thunderhead ne sont pas réécrits (report : prochaine cuisson). La cuisson Forge n'a
  pas tourné dans le worktree (dépôt hors ligne des variantes absent) : témoin AVANT pris par le pilote sur 3
  cartes Forge, comparaison à faire en G.4.
- Piste E (13 commits). E.9 adapté : un seul appel sur la page du rejeu ; quatre surfaces de la page Match sur
  une liste datée en cliquet (plafond « 1 » impossible, elles n'ont pas de modèle de rejeu). E.7 : la `raison`
  par répartition n'est pas lue (elle répète la raison de la réponse) : à retirer côté serveur en G. Compteurs
  gelés : 46 fonctions à plus de 5 paramètres, 252 de plus de 80 lignes de code ; 17 `.tsx` hors rejeu écoutent
  encore Échap ou le clic dehors (au backlog, accord du user).
- Piste F (4 commits). F.4 : la requête rendait 9 lignes pour 5 matchs (autre échelle, lignes périmées) ;
  corrigée ; elle n'a aucun appelant de production (question posée au user : la supprimer).

### 2026-11-17 — piste B close, intégration faite (étape G)

- Piste B (4 commits). B.2 a entraîné plus que prévu : `write_exclusive.go`, l'étape de compression de la file
  de régénération et ses compteurs, le cache du banc `replay-corpus-gate` (désormais en `.json.gz`), les fixtures
  de dix paquets. B.4 : `[~]`, aucun champ retiré du document (recommandation du pilote après la mesure 0.5 :
  garder les sept ; confirmation du user attendue).
- Fusion des six pistes dans `chore/830-suites-audit` sans conflit ; `go build ./...` et `go vet` verts sur
  l'arbre fusionné.
- Intégration (branche `sa/g`, 11 commits) : types du front régénérés et fixtures corrigées ; `raison` par
  répartition retirée (celle de la réponse est lue) ; test périmé du fond par défaut corrigé ; test de parité
  « par minute » Go / TS (8 cas, aucune divergence) ; quatre champs de compression retirés de l'état de la file
  de régénération, du contrat et de l'écran ; `--garder-documents` dépose en `.json.gz` par la porte ; liste
  datée des champs du document sans lecteur (`lib/replay/replayDocumentReaders.guard.test.ts`) ; ADR 0048 (marge
  réelle 0,5 m), 0034 et 0045 amendés ; notes de version et changelog de la 8.3.0.
- B.3 : la mesure trouve six clés sans lecteur en plus des sept attendues : `bombEvents`, `bombStats`,
  `identity`, `stances`, `structureBounds`, `vehicleScenery`, inscrites « à statuer par le user ».
- G.2 : les trois règles approuvées par le user sont écrites (`CLAUDE.md` règles 18 et 19, skill
  `plan-execution` règle 6) ; ligne de l'ADR 0048 et chiffres de la façade du décodeur remis au vrai dans
  `CLAUDE.md`.
- G.4 : cuisson Forge de 3 cartes avec le code fusionné : images et tableau identiques à l'octet au témoin
  d'avant ; seules différences, l'horodatage et la dégradation nommée des deux Thunderhead (effet voulu de D.5).
- Gate de l'intégration (exécuteur) : `go test` 192 paquets verts, tests `integration` 20 paquets verts,
  typecheck web 0 erreur, vitest 13 191 tests verts, 0 échec. Diff depuis `release/v8.3.0`, hors `.ai/` :
  332 fichiers, +7 911 / -8 792.
- Reste : G.3 (backlog et registres, après les réponses du user), `make gate-push`, étape H.

### 2026-11-17 — relecture finale (H.1), corrections, gate

- Relecture par un contexte Fable frais du diff `origin/release/v8.3.0...chore/830-suites-audit` : 44 conditions
  vérifiées tiennent ; 0 P0, 2 P1, 5 P2. Non relu par elle : le découpage de `useReplayVehicles`, l'intérieur de
  la cuisson des fonds (couvert par les cuissons témoins), la plupart des fichiers de test.
- Corrections (branche `sa/g`, 5 commits, chaque constat rouvert avant correction, tous confirmés) :
  - R.1 (P1) : un fichier tactique au format 7 d'un rejeu présent n'était jamais reprojeté (la marque de
    dérivation n'avait pas bougé). `DerivationsRev` monte ; le rattrapage des dérivations le reprojette sans le
    film, sur les 64 matchs éligibles les plus récents, cinq par cycle de synchronisation, jamais au démarrage.
    Au-delà : régénération du rejeu, ou `levelup tactical-rasters --backfill` (non livrée avec le bureau).
  - R.2 (P2) : Échap dans le filtre du fil fermait aussi le panneau ouvert derrière. Le hook partagé ferme la
    couche du dessus, seule ; deux couches ouvertes se ferment une par frappe (seul changement voulu).
  - R.3 (P2) : les gardes des couches voient les quatre comparaisons à `'Escape'` ; compte hors rejeu inchangé (17).
  - R.4 (P2) : le nettoyage de la démo ne supprime plus que les `.json.gz` hors manifeste.
  - R.5 (P1) : les documents disaient à tort qu'un rejeu brut « se régénère ». Faits établis : le rattrapage
    après synchronisation ne couvre que les 64 matchs éligibles les plus récents ; la file de régénération ne
    liste que les `.json.gz` ; aucun écran ne propose de générer un rejeu absent ; la commande
    `levelup backfill-replay` n'est pas livrée avec le bureau. Documents corrigés. L'affirmation du pilote au
    user (« vu comme absent et régénéré ») était fausse au-delà de ces 64 matchs : décision soumise au user.
  - P2 non corrigés : six clés du document « à statuer par le user » ; sept champs non exportés de
    `objectiveblend.Result` lus par les tests seuls (gardés : ils fixent les termes de la formule dans les tests).
- Gate de l'exécuteur après corrections : `go build`, `go vet`, tests Go ciblés et `integration` verts, archlint
  vert, `golangci-lint --new-from-merge-base=origin/main` 0 problème, vitest 13 199 tests verts.
- `make gate-push` (pilote) : étape 1, lint Go : 2 `unparam` trouvés puis corrigés, 0 problème ensuite.
  Étape 2, harnais d'équivalence du rejeu : `[!]` IMPOSSIBLE sur ce poste, le fichier local
  `data/film_identites/xuids_originaux.txt` n'existe plus depuis l'incident du 2026-11-16 ; soumis au user.
  Étape 3, typecheck et lint web : verts. Étape 4, baseline des tests Go : aucun test en échec ; le paquet
  `internal/platform/duckdb` a dépassé son budget de 300 s pendant que d'autres compilations tournaient
  (199 tests non joués) ; rejoué seul par le pilote avec les tests `integration` : 1 395 tests verts, 0 échec, en 307 s, soit au-dessus du budget de 300 s même seul (découverte : ce paquet est à la limite de son budget sur ce poste).
- Vérifications à l'écran posées dans l'artefact (`suites-audit-mainmise-format`, `-menus-rejeu`,
  `-motif-probabilite`, `-fonds-tournes`).

## Découvertes (consignées, non traitées sauf mention)

1. Test `MatchPositionsHeatmap.orientation.test.tsx` rouge sur `release/v8.3.0` (attend « rendu », le défaut est
   « plan » depuis `47b316e39b`). Bloque le gate : corrigé en G.
2. ADR 0048 D-2 annonce une marge de rognage de 1,5 m, le code applique 0,5 m : corrigé en G avec l'amendement.
3. ADR 0048 D-5 : une carte Forge au maillage illisible avertit et publie sans plan, l'ADR dit « erreur de la
   cuisson ». Question posée au user.
4. Fichiers tactiques `rasters/*.json` non compressés : 1,2 Go pour 1 455 matchs. Question posée au user.
5. Médailles absentes du rejeu dans 11,9 % des matchs (événements sans nom). Question posée au user.
6. Décodeur : verdict « lecture complète » du score faux en BTB et en Assaut ; `flagReturnS` jamais écrit ;
   lecture des tirs suffisante pour 12 % des joueurs d'un échantillon ; coup d'envoi de l'API à 5,3 s du film.
7. `StatsRepository.LoadLUSRHistory` et `Q24LUSRHistory` sans appelant de production.
8. Escouade : cadence de frags gonflée par les matchs sans durée (voir C.5).
9. `TestBackfillRegistryNames_BaseTenueParUnAutreProcessus` (`cmd/levelup`) échoue sous charge, passe seul.
10. Le hook `go-vet` affiche des erreurs de contraintes de build sur des commandes sans CGO et passe quand même.
11. `sgh_interlock.json` porte des chemins Windows absolus dans `degradations`.
12. `zonesPuresDuCatalogue` (CLI `spawn-model`) double la projection de zones du service.
13. Caches de compilation `go-build-sa-*` à rendre à la clôture (`disk-hygiene.ps1 -Lot`).
