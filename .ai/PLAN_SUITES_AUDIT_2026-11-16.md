# Plan — suites de l'audit des versions 8.2.0 à 8.3.0 (2026-11-16)

État : écrit, relu contre la grille `plan-review`. Réponses du user du 2026-11-16 portées ci-dessous
(« Réponses du user »). Restent à trancher : D-2 i, D-2 j, D-6, D-7, D-8 (réexpliquées), puis le go.
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

- [ ] 0.1 Rouvrir chaque `fichier:ligne` du registre ; noter ici tout constat déplacé ou déjà réglé.
- [ ] 0.2 Témoins AVANT, sur trois matchs à rejeu et une carte, rangés hors dépôt : réponse du rejeu
      (`encoding=columns`), des deux routes de réapparitions, de la page Séries temporelles, de la vue du match ;
      empreintes des 114 sidecars et des images de `map_backgrounds/` ; prévisions de réapparition de trois
      matchs (scores par candidat).
- [ ] 0.3 Compteurs de départ : `grep` de chaque motif visé par un garde-rail neuf (copies de la Mainmise,
      `polygon.length >= 3`, `ui-sans-serif`, `arcTo(`, `buildPlayers(`, fonctions à plus de 5 paramètres et de
      plus de 80 lignes de code sous `apps/web/src`, paquets du décodeur importés par `internal/port`).
- [ ] 0.4 D-7 : nombre de zones du catalogue sans contour mais avec une partie de 3 sommets ou plus.
- [ ] 0.5 Comparaison des sources (réponse du user à D-2 a à d), sur 20 matchs du parc lus un par un, et pour
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
- [ ] 0.6 D-3 : nombre de rejeux `.json` non compressés dans le parc local et en prod (lecture seule).
- Gate : témoins écrits, compteurs notés dans ce fichier, tableau 0.5 remis au user.

## Piste A — réapparitions, côté Go (8.3.0 non publiée : aucun contrat à tenir)

- [ ] A.1 Ligne de vue retirée : `analysis/tactical/ligne_de_vue.go`, `VarVisibles`, `PorteeVueM`, `Visibilite`,
      paramètre `vue` de `Variables`, `AvecVisibilite`, `ReglagesEchantillons.Vue` ; option `--ligne-de-vue`,
      `fonctionVue`, `obstaclesDuDocument`, section m2 ; clé `ennemis_a_vue_40m` de `spawn_model.json`
      (révision du fichier montée) ; `TestObstacles_Voit` ; `docs/COMMANDS.md` et sa version FR.
      Gate : `git grep -nwE "NouveauxObstacles|AvecVisibilite|VarVisibles|ligne-de-vue" -- apps docs data` vide ;
      prévisions des trois matchs témoins identiques au centième.
- [ ] A.2 La CLI appelle les helpers d'`analysis/tactical` au lieu de les recoder : `fini` -> `Point3.Fini`,
      `evenementDe` -> une fonction exportée unique, `fonctionZone` -> `IndiceDeZone`, littéral `2` ->
      `CandidatsMinPrevision`. Gate : `git grep -nE "func (fini|evenementDe|fonctionZone)\b" -- apps/go-api` vide.
- [ ] A.3 D-5 appliquée : section m1 et ses constantes de témoin retirées avec leurs tests.
- [ ] A.4 D-2 e et g appliquées : champs retirés de `domain/replaydoc/spawn_forecast.go`,
      `domain/tactical_reapparitions.go`, des services et de leurs tests ; le journal `slog` les garde.
      D-2 f et h : les champs de motif restent (lecteur à l'étape E.7).
- [ ] A.5 Repli sur l'écart de 4 à 15 s : commentaire au format de la règle 11 (date, cible de retrait, critère :
      part du repli lue dans les comptes du sidecar, D-2 k) ; les comptes `film` / `repli_ecart` sont lus et
      rendus par `spawn-model fit` dans son bilan.
- [ ] A.6 D-2 j appliquée : `source` et `debut_de_manche` retirés du sidecar, `TacticalRasterSchemaVersion` 7 -> 8,
      note au registre des reports : « part avec la régénération au schéma 125 ».
- [ ] A.7 `SpawnFamilySet.TitleSlug()`, `SchemaVersion()` et leurs champs retirés avec leur test.
- [ ] A.8 Règles de jeu pures sorties de `service/tactical_service_lectures.go` vers `analysis/tactical`
      (`spawnsDuJoueur`, `sidecarsAuxVies`, `routesDesVies`, `reapparitionsAdversesDuMatch`) : le service traduit
      `domain` vers les entrées d'analysis, la règle se teste sur les types d'analysis ; garde
      `archlint/no_local_spawn_depart_test.go` mis à jour.
- Gate de piste : `go test ./internal/analysis/tactical/... ./internal/service/... ./cmd/levelup/...`.

## Piste B — rejeu servi, côté Go

- [ ] B.1 Fraîcheur du document décidée par le service : `port.ReplayService.GetReplay` rend l'état de cadence ;
      `handlers/replay.go` n'importe plus `replaybuild`. Test de service qui échoue si la règle s'inverse.
      Gate : `grep -n "replaybuild" apps/go-api/internal/api/handlers/replay.go` vide.
- [ ] B.2 D-3 appliquée, retrait en 8.3.0 : `replayartifact` ne connaît plus que `.json.gz` (`Locate`, `Read`,
      détection par octets magiques, `RemoveLegacy`, `HasLegacySibling`, compteur, `ReplayArtifactCandidates`) ;
      `replaybuild/artifact_compress.go`, sa commande et leurs tests retirés ; un `.json` restant vaut « rejeu
      absent » et se régénère (la 8.3.0 régénère de toute façon tout le parc au schéma 125). Migration de
      démarrage unique qui supprime les `.json` de rejeu restés dans le dossier des rejeux et journalise leur
      nombre (aucun autre fichier touché ; test sur un dossier temporaire). Notes de la 8.3.0.
      Gate : `git grep -nE "RemoveLegacy|HasLegacySibling|PARC MIXTE" -- apps/go-api` vide.
- [ ] B.3 Liste datée des champs du document sans lecteur web : un test web lit les clés de premier niveau de
      `replayDocumentSchema.ts` et exige, pour chacune, un lecteur hors schéma et normalisation, ou une entrée
      `{ clé: "AAAA-MM-JJ : lecteur prévu" }`. Entrées de départ : les champs que le user garde après le
      tableau de l'étape 0.5. La liste ne fait que rétrécir ; une montée du décodeur qui ajoute un champ sans
      lecteur doit l'y inscrire.
- [ ] B.4 Champs que le user ne garde pas après l'étape 0.5 : retirés du document servi (`replaydoc`,
      `replayview`, contrat, types web, schéma et normalisation web), sans toucher au décodeur.
- [ ] B.5 Types du décodeur dans `internal/port` : garde-rail qui gèle les paquets importés à l'état de
      l'étape 0.3 (plafond qui ne fait que descendre).
      `[!]` prévu pour le retrait lui-même : trois types traversent `port/` et 9 fichiers les consomment ; c'est
      un chantier à part, à ranger avec les évolutions du décodeur (ligne au registre des reports).
- Gate de piste : `go test ./internal/service/... ./internal/api/... ./internal/replayartifact/...` ; témoin du
  rejeu identique octet pour octet, hors B.4.

## Piste C — indicateurs et couches, côté Go

- [ ] C.1 Mainmise : un helper de lecture (charge, journalise sauf capacité absente, pose les deux pointeurs)
      appelé par les quatre services ; garde-rail qui interdit le bloc recopié.
      Gate : `grep -rn "share, changes := v.Share, v.Changes" apps/go-api/internal --include=*.go | wc -l` = 1.
- [ ] C.2 D-2 i appliquée : champs retirés de `TimeseriesMatchRow`, du service et de son test.
- [ ] C.3 D-2 l appliquée : `objectiveblend.Result` n'exporte que `Score` ; commentaire remis au vrai.
- [ ] C.4 Harnais `sync/performance_objective_simulation_research_test.go` et `..._report_research_test.go`
      retirés (578 lignes).
- [ ] C.5 Agrégat « par minute » : une seule fonction Go dans `analysis/` appelée par `trends` et par
      `teammates` (`paceSums` disparaît) ; jeu de cas partagé lu par le test Go et par le test de
      `lib/charts/perMinute.ts` (modèle : `testdata/meneur_cas.json`).
- [ ] C.6 Famille d'objectif : `objectiveFamilyFromRaw` et `objectiveColumnValues` passent dans
      `analysis/narrative` ; test de parité avec le SQL `objectiveFamilyCaseSQL` sur une table de cas (DuckDB en
      mémoire), un cas par famille et un cas sans objectif.
- [ ] C.7 Révisions de notation : l'ordre et l'indépendance des deux étapes passent de
      `api/wire/registry_scoring_revision.go` à `service/scoringrevision` ; le câblage ne fait plus qu'injecter
      les deux étapes. Tests déplacés avec la règle.
- [ ] C.8 `PlayerProfileService` : interface déplacée dans `port/`, `handlers/player_profile.go` n'importe plus
      `progression/profile` pour son service.
- [ ] C.9 `port/media_indexer.go` : paramètres de `ResetAndReindex` et `ScanAllMedia` regroupés en une
      structure ; `*jobs.Store` remplacé par l'interface étroite que l'indexeur utilise.
- Gate de piste : `go test ./internal/service/... ./internal/analysis/... ./internal/sync/...
  ./internal/api/...` ; témoins de la vue du match et des Séries temporelles identiques, hors C.2.

## Piste D — fonds de carte

- [ ] D.1 D-1 appliquée : ADR 0048 amendé (D-3, D-7, « État courant », amendement daté) ; ligne 0048 de
      `CLAUDE.md` corrigée par le pilote.
- [ ] D.2 D-6 appliquée, options : `-rogne-publie` retiré avec `rogne_publie.go` et son test ; rapport Markdown
      retiré ; `docs/COMMANDS.md` EN et FR.
- [ ] D.3 D-6 appliquée, leviers : champs, accesseurs et câblage des leviers sans donnée retirés ; les
      accesseurs restants passent par UN accesseur générique ; `reglages.go` sous 500 lignes ; garde-rail :
      tout levier déclaré a au moins une entrée dans `map_fond_reglages.json` ou une exemption datée.
- [ ] D.4 Ancres : `ancresDuCatalogue` appelée par les trois sites (`cuisson.go`, `cuisson_forge.go`,
      `plan_seul.go`). Gate : `grep -n "\[3\]float64{o.Pos.X" apps/go-api/cmd/mapfond-build/*.go | wc -l` = 1.
- [ ] D.5 Carte Forge sans maillage : `cuisson_forge.go` pose la dégradation nommée ; test qui fixe l'ensemble
      des fonds sans plan (trois, nommés). Les deux sidecars Thunderhead ne sont réécrits que si la commande
      rend une image identique à l'empreinte de l'étape 0 ; sinon `[!]` et ligne au registre (prochaine cuisson).
- [ ] D.6 Rotation du fond : une fonction d'application pour le canvas et une pour le CSS dans
      `lib/replay/orientation.ts`, appelées par le rejeu, la Tactique et « Occupation du terrain » ; garde-rail
      étendu aux trois vues (aucun `ctx.rotate(` ni `rotate(${turns` hors de ce fichier).
- [ ] D.7 Commentaires de `main.go` et `reglages.go` qui citent `libs/` et `licence_test.go` remis au vrai.
- Gate de piste : `go test ./cmd/mapfond-build/... ./internal/mapfondsidecar/...` ; empreintes des 114 sidecars
  et des images identiques à l'étape 0, hors D.5.

## Piste E — front du rejeu et de la Tactique

- [ ] E.1 Fermeture d'une couche (Échap, clic dehors) : `useDismissable` sort de
      `components/shell/_filter_pills/_hooks.ts` vers un hook partagé, avec le cas « couche intérieure ouverte » ;
      appelé par `ReplayFeedFilterMenu`, `ReplaySpeedMenu`, `ReplayViewPanel`, `ReplayPlayerPanel`,
      `ReplayMediaLightbox` et par le shell. Garde-rail : aucun `addEventListener('pointerdown'` ni
      `key === 'Escape'` dans un `.tsx` de `features/`.
- [ ] E.2 `ReplayFeedFilterMenu` : navigation clavier du menu dans un hook testé seul ; composant sous 80 lignes
      de code.
- [ ] E.3 Rectangle arrondi : une fonction dans `lib/replay` (repli sans `ctx.roundRect` compris), appelée par
      `respawnForecastLegend`, `respawnForecastPaint`, `export/overlayPaint` ; garde-rail sur `arcTo(`.
- [ ] E.4 Les 4 signatures à plus de 5 paramètres ajoutées dans la fenêtre passent à un objet d'options
      (`drawProjectilesLayer`, `drawTintBar`, et les deux rectangles, réglés par E.3).
- [ ] E.5 D-7 appliquée : un prédicat et une énumération des contours d'une zone dans `lib/replay`, appelés par
      `calloutsPaint`, `killerZone`, `respawnForecastPaint`, `reapparitions.logic`, `TacticalReapparitionsZones` ;
      garde-rail sur `polygon.length >= 3` hors de ce fichier.
- [ ] E.6 Règle de rognage au plan sortie de `ReplayPage.tsx` vers un fichier de logique, avec son test (plan
      choisi et image présente, plan choisi sans image, rendu).
- [ ] E.7 D-2 f et h appliquées : le calque de probabilité et la lecture « Réapparitions adverses » affichent le
      motif du refus dans un état vide (bloc affiché, pointillé, aucune action impossible), en fr et en,
      couleurs par tokens sémantiques (skills `frontend-patterns`, `color-tokens`).
- [ ] E.8 Police du canvas : une constante lue par les 7 sites ; garde-rail sur `ui-sans-serif`.
- [ ] E.9 `buildPlayers` : appelé une seule fois (`model/replayModel.ts`) ; `ReplayTeams`, `useSlotIdentity`,
      `equipmentUsageLogic`, `killerZone` reçoivent `model.players` ; garde-rail qui rend vrai le commentaire
      « nulle part ailleurs ».
      `[~]` prévu pour le découpage de `ReplayCanvas`, `ReplayPlayerPanel` et `LayersSection` : la refonte du
      rejeu les réécrit. Une ligne est ajoutée à `PLAN_REFONTE_UI_REJEU_2026-11-15.md` : ces trois fonctions
      sortent de la refonte sous 80 lignes de code.
- [ ] E.10 `useReplayPlayback` (110), `useReplayFlagCarries` (102), `useReplayVehicles` (186) : ramenées sous
      80 lignes de code par extraction, ou exemption commentée si le corps est une table.
- [ ] E.11 Deux compteurs gelés sous `apps/web/src` (test) : fonctions à plus de 5 paramètres, fonctions de plus
      de 80 lignes de code ; valeurs de l'étape 0.3 moins ce que cette piste retire ; ne font que descendre.
- [ ] E.12 Les 8 `eslint-disable no-explicit-any` des tests de graphes remplacés par des types.
- Gate de piste : `make check-types`, `make test-web` COMPLET, `node tools/lint-cross-feature-imports.mjs`.

## Piste F — exceptions

- [ ] F.1 D-4 appliquée : `LEVELUP_SYNC_RESOLVE_ASSETS` retiré avec sa branche ; entrée retirée de
      `lecturesEnvAutorisees`.
- [ ] F.2 Les trois lectures « à porter par AppConfig » passent par `internal/config`
      (`registry_replay_notify.go`, `engine_highlight_events.go`, `snapshot_readiness_eval.go`) ; entrées
      retirées. Le contrôle de date de l'allowlist exige le format `AAAA-MM-JJ`.
- [ ] F.3 `effective_hp_copies_test.go` : entrée datée, le test exige la date.
- [ ] F.4 `Q24LUSRHistory` (`queries_career_encounters.go`) : lecture brute de `match_skill_rank` revérifiée ;
      filtre `rating_type` et tie-break `written_at` ajoutés, test sur un match à double ligne ; l'entrée
      d'allowlist dit ce qui est vrai. Skill `db-schema` avant d'écrire.
- [ ] F.5 `[~]` prévu : les deux entrées de `pool_no_shared_handle_test.go` sont datées, le risque est écrit,
      le code est déplacé à l'identique ; aucun changement.
- [ ] F.6 `[~]` prévu : plafond de surface du décodeur (316 -> 335), chaque marche datée et justifiée ;
      `CLAUDE.md` cite le garde-rail au lieu des nombres (pilote).
- [ ] F.7 `[~]` prévu : liste des imports entre fonctionnalités (79 -> 99), seuil du cliquet à 0 ; aucun changement.
- Gate de piste : `go test ./internal/archlint/... ./internal/config/... ./internal/platform/duckdb/...`.

## Étape G — intégration (pilote)

- [ ] G.1 Fusion des pistes dans `chore/830-suites-audit` ; `openapi.yaml` et `make generate-types` une fois.
- [ ] G.2 Doctrine, trois ajouts soumis au user avant écriture : (1) « une donnée servie ou persistée a un
      lecteur, ou une ligne datée qui nomme son lecteur prévu » (`CLAUDE.md`) ; (2) « à la clôture d'un plan,
      l'outillage d'une mesure close est retiré » (skill `plan-execution`) ; (3) « une version de LevelUp monte
      le décodeur une fois, la régénération du parc se lance à la publication » (`CLAUDE.md`).
- [ ] G.3 Notes de la 8.3.0, `docs/CHANGELOG.md`, `.ai/BACKLOG.md` (P2 traités retirés), registre des reports
      (A.6, B.2, B.5, D.5 le cas échéant), registre de l'audit (colonne « Décision » remplie), `thought_log`.
- [ ] G.4 Comparaison des témoins APRÈS aux témoins de l'étape 0 : écarts listés, chacun rattaché à une
      décision D-2 ou déclaré défaut.
- Gate : `make gate-push` ; `cd apps/go-api && go test -tags=integration ./internal/sync/...` ; `make go-api-lint`.

## Étape H — relecture et clôture

- [ ] H.1 Une ronde de relecture adversariale du diff complet (skill `adversarial-review`, Fable) ; P0 et P1
      corrigés, P2 au backlog.
- [ ] H.2 Contrôles à l'écran du user, dans l'artefact des vérifications : menus du rejeu (fermeture par Échap et
      clic dehors), calque de probabilité et son état vide, Tactique « Réapparitions adverses », fonds Plan et
      Rendu tournés sur les trois vues.
- [ ] H.3 Tous les items statués ; fusion dans `release/v8.3.0` (CI verte).

## Découvertes

(vide)
