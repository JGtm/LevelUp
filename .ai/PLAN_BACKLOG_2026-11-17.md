# PLAN — Entrées du backlog des 14 au 17 novembre (2026-11-17)

Statut : **écrit le 2026-11-17, relu contre la grille `plan-review` ; décisions D-1 à D-6 tranchées par le user
le 2026-11-17 (« ok avec tes recos ») : les six recommandations sont retenues. Go d'exécution donné par le user
le 2026-11-17 (plan maître `.ai/PLAN_MAITRE_8_3_0_2026-11-17.md`, DM-4), qui fixe aussi le moment de la
régénération : dès la fusion de ce plan (DM-1).**
Ordre entre plans, fichiers partagés et borne de 4 agents d'écriture : plan maître.
Contrat d'exécution : skill `plan-execution` (ordre strict dans une piste, aucun report, statut de chaque item,
vérification sur pièces, zéro correctif hors périmètre). Statuts : `[x]` fait, `[~]` couvert ailleurs
(référence), `[!]` non traité (motif écrit). Aucune case vide à la clôture.

Demande du user (2026-11-17) : analyser les entrées récentes du backlog, les instruire sur preuve, garder ce
qui est pertinent et faisable, planifier en pistes parallèles (Sonnet par défaut, Haiku quand c'est mécanique,
Opus quand il le faut, Fable pour la revue finale), périmètre fermé, découvertes au backlog seulement si elles
valent le coup.

## 0. Objectif et critère de succès

Objectif : vider le backlog de ce qui se règle maintenant, corriger les entrées dont l'instruction montre
qu'elles sont fausses, et rattacher par écrit à leur plan celles qui en dépendent.

Critère de succès : chaque entrée de la section 1 a un sort écrit (traitée, close, reformulée, rattachée, ou
gardée avec sa condition) ; `make gate-push` vert ; le paquet `internal/platform/duckdb` passe sous son budget
de 300 s avec le même nombre de tests réussis ; les fichiers tactiques s'écrivent compressés AVANT la
régénération du parc.

| Piste | Ce que le user verra |
|---|---|
| A | Rien ; `make gate-push` ne coupe plus le paquet DuckDB sous charge |
| B | Rien à l'écran ; environ 1 Go de disque rendu après la régénération du parc |
| C | Escouade : un coéquipier aux frags inconnus ne tire plus le taux par minute vers zéro ; le reste est interne |
| D | Échap et clic dehors ferment 12 écrans de la même façon (une couche par frappe) |
| E | Rien ; plan de refonte du rejeu et backlog remis d'accord avec les faits |
| F | Rien (retrait d'un chemin qui ne s'exécute plus) |
| G | Un bouton « Générer le rejeu » sur un match sans rejeu, application de bureau et poste de développement |

Base : `release/v8.3.0` (`a8738c9d21`), qui contient les suites de l'audit (PR #212 fusionnée le 2026-11-17).
Branche : `chore/830-backlog-1117`, worktree `C:/t/bl1117` (sans jonction vers `data/`). Une seule branche, un
commit par item ou par groupe d'items d'une piste, fusion dans `release/v8.3.0` à la fin, sans PR par étape.
Jamais `main`.
Version : 8.3.0. Aucune montée de brique.

## 1. Instruction : ce que les pièces disent (2026-11-17)

Six instructions en lecture seule sur `chore/830-suites-audit` à `32045396fb` (quatre commits avant sa fusion :
trois correctifs des tests de `replay-diff` et un commit de journal ont suivi), le décodeur v1.4.1 (cache des
modules), libfilmshell, les rejeux du poste (1 455, tous au schéma 124) et une copie de la base partagée.
L'exécutant RE-VÉRIFIE chaque référence avant de coder (étape 0.3).

| Entrée du backlog | Constat sur pièces | Sort |
|---|---|---|
| Tests du paquet `platform/duckdb` au-delà du budget | Confirmé. 5 cibles de migration ; migration de 0,6 s (partagée en mémoire) à 5 s (metadata) par test ; copie d'un gabarit migré : 19 à 57 ms, y compris vers une base en mémoire (`ATTACH` lecture seule, `COPY FROM DATABASE`, `DETACH`) ; base fermée sans `.wal` ; schéma identique. 31 appels directs dans 27 fichiers, l'essentiel du coût passe par une dizaine de helpers | Piste A |
| Doublon `[tests/local]` (2026-11-15) | Même paquet, même symptôme | Supprimé (E.6) |
| Fichiers tactiques `rasters/*.json` non compressés | Confirmé : 1 455 fichiers, 1,13 Go ; gzip : rapport 0,137. Un écrivain, trois lecteurs, deux purges. Tous les fichiers du poste sont au format 7, illisibles depuis le format 8 : ils seront réécrits de toute façon. La compression coûte 4 à 6 ms de lecture en plus par fichier | Piste B, avant la régénération |
| Probabilité de réapparition : fichiers relus à chaque ouverture | Confirmé (`service/replay_spawn_candidats.go`). Poste : 39 fichiers au plus par carte et famille, médiane 4, environ 0,2 s au pire. Un cache à empreinte existe déjà (`service/catalogue_cache.go`, `cacheAEmpreinte`). Condition de reprise « 1 s mesurée en prod » non vérifiable ici | Piste B (D-1) |
| Coup d'envoi de l'API contre celui du film | Le report du film vers le registre existe (`replayartifacts/t0film.go`, qualité `film_movement`). 186 des 1 455 matchs avec rejeu l'ont (13 %) ; la régénération repasse par ce report | Aucun code ; contrôle après la régénération inscrit au registre (E.5) |
| Générer à la demande un rejeu absent | Aucune route ni écran. La file de régénération du bureau (`internal/replayregen`) ne prend que des rejeux déjà rangés ; il manque le téléchargement du film d'un seul match et la porte d'entrée. Poste : 7 matchs sans rejeu dont le film existe encore | Piste G (D-4) |
| Lignes d'objectif rechargées à chaque passage | Code confirmé, coût non démontré (5 405 lignes pour le joueur le plus actif, requête de l'ordre de 100 ms) | Reste au backlog, condition inchangée |
| Fermeture par Échap ou clic dehors, 17 écrans | 12 vraies couches à migrer ; 5 cas particuliers (`AssetDockedPanel` : Échap limité au volet, tenu par un test ; `TacticalAnalysisView` : deux Échap en pile avec condition ; `TacticalPlanTueurs`, `SquadFocusStrip`, `useSquadPresets` : faux positifs) | Piste D |
| Reliquats du garde-rail des couleurs | 19 marqueurs `color-allow` inertes (pas 23) ; 0 occurrence à corriger pour `shadow`, `from`, `via`, `to` (les 15 trouvées sont dans `palmares/rarity.ts`, fichier autorisé) | Piste D |
| Médailles absentes du fil du rejeu | Cause : 12 399 événements sans nom sur exactement 173 matchs d'avant le 2026-09-02. La commande `levelup backfill-medailles-feed` les répare depuis le cache des films, qui n'est pas sur le VPS : elle ne règle pas la prod. Le plan de refonte ne contient aucun item qui lise `medals` pour le fil : l'entrée était orpheline | Rattachée par écrit au plan de refonte (E.2, D-2) |
| Sprints et sauts déjà dans le document | Confirmé (`stances` : `sprint` lu, `jumpDerived` calculé, `clamber`). Échantillon de 30 rejeux : 15 656 sprints, médiane 0,82 s, aucun au-delà de 30 s. Le plan de refonte (D-23, D-24, T.5) et deux commentaires de code disent le contraire | Textes corrigés (D.4, E.1) ; affichage inscrit au plan de refonte (E.3, D-3) |
| Six petites découvertes | 2 à 6 confirmées (détail en piste C). 1 (test instable sous charge) : non reproduit en 96 passages parallèles, aucune attente bornée dans le code | Piste C ; 1 reste au backlog, reformulée |
| `replay-equiv` ne compare pas les joueurs du tableau | Confirmé ; l'étape manquante vit dans le dépôt du décodeur (une quinzaine de lignes, puis références à refiger). L'écart reste détecté par l'empreinte du document | Chantier décodeur (D-6) |
| Témoin `f75e7053` hors d'atteinte | Confirmé. Remplaçant sur le poste : `0a247154` (même carte, même mode, colline unique, 213 intervalles) | Piste C |
| ADR 0024 périmée | Trois écarts confirmés ; un quatrième : la limite 7 cite `applyMatchToSkillV2`, absent du code | Piste C |
| Carte Forge au maillage illisible | Aucune carte concernée : les 92 maillages du poste se lisent tous ; la branche « illisible » vient d'une lecture de code, pas d'une observation | Close (E.6) |
| Thunderhead et Thunderhead Heavies sans plan | Vérifié le 2026-11-17 (`mapnav-fetch -dry-run`) : la carte ne publie aucun maillage. « Jamais récupéré » est faux ; « sous 1 000 objets » aussi (plus de 5 000 objets) | Close, textes corrigés (C.8) |
| Actions de score : verdict « lecture complète » faux | Cause établie dans le décodeur : Assaut (la bombe paie hors de l'arbre des réponses) et BTB du build `HI_1_11_0` (points de véhicules d'un autre barème). BTB des builds 12 et 13 : exact. Aucun lecteur aujourd'hui | Garde inscrite au plan de refonte (E.4) ; correction : chantier décodeur (D-6) |
| `flagReturnS` jamais écrit | Faux : le champ naît au schéma 125, le parc mesuré était au 124. Le décodeur le lit sur 46 films à drapeau sur 46 | Close, mesure corrigée (E.6) |
| Tirs lus : 12 % des joueurs | Échantillon de 8 fixtures dominé par des builds anciens. Mesure du décodeur sur 193 films : 23 % ; le parc du poste est à environ 90 % au build 13 | Reformulée : à remesurer après la régénération (étape T.0 du plan de refonte, E.4) |
| Touches et précision par l'ancien appariement | Les deux lecteurs existent mais ne s'exécutent plus pour Halo Infinite depuis le 2026-09-01 (capacité `match.weapon.accuracy` non exposée) ; 0 ligne dans `weapon_accuracy` et `match_weapon_hit_distance` sur le poste ; aucun lecteur. Rien de faux n'est persisté | Piste F (D-5) |
| Porteurs du drapeau faux en grande bataille | Confirmé, cause non établie (pont ligne du tableau vers joueur quand un siège change d'occupant) | Chantier décodeur, recherche (D-6) |
| Auteur d'un dégât sur l'index brut | L'auteur est déjà résolu à l'instant ; seule l'arme de la touche se lit encore par index brut | Reformulée ; chantier décodeur (D-6) |
| Écarts du gate du corpus (v1.3.9) | Trois réglés ou expliqués, deux replis restent par construction (films de builds anciens), quatre compteurs nommés faux pour 78 joueurs comparés, cause inconnue | Reformulée ; chantier décodeur, recherche (D-6) |
| Ordre des collines | La suite des collines actives se lit déjà dans le document ; elle diverge entre matchs d'une même carte ; aucun écran ne la lit | Reformulée, condition : un écran qui la lise |
| Noms des marqueurs (liste de cartographer) | Prémisse fausse : 0 des 65 empreintes dans le décodeur, qui ne publie aucun identifiant de marqueur. Vraie piste : la lettre réelle des zones se lit dans le film (libfilmshell), chantier moyen à lourd | Reformulée ; chantier décodeur (D-6) |
| Temps forts de type « mode » | Compté : 23 683 sur 527 matchs. Les destructions de véhicules sont déjà attribuées par une autre voie ; en Slayer, environ la moitié de ces temps forts reste inexpliquée | Entrée mise à jour, priorité basse |
| Commandes et gestes | Sprint, saut et visée déjà publiés ; commandes de déplacement, bouton de saut et rechargements lus mais non publiés | Entrée corrigée ; chantier décodeur (D-6) |
| Répliques, bouclier brisé, jalon de précision | Conditions de reprise non remplies ; jalon : décision du user du 2026-11-15 | Inchangées |

Hors périmètre (entrées antérieures au 14 novembre) : démarrage et pool de jetons, zone du tueur dans le fil,
planches d'icônes, trajet de Warthog, films d'un ancien schéma, pièces montées, pseudos, alerte Dependabot.

## 2. Décisions (tranchées par le user le 2026-11-17 : les six recommandations sont retenues)

- **D-1 — cache des réapparitions sans attendre une mesure en prod.** L'entrée fixait « plus d'une seconde
  mesurée en prod ». La compression des fichiers tactiques double le coût de lecture par fichier. Retenu :
  le cache est livré avec la compression (30 à 60 lignes, patron existant).
- **D-2 — médailles du fil.** Retenu : le plan de refonte reçoit un item qui lit les médailles du document
  pour le fil et pour la fiche (le serveur résout libellé et image), seule voie qui règle aussi la prod. Ce
  plan-ci n'écrit que l'item ; le code part avec la refonte.
- **D-3 — sprints et sauts.** Retenu : affichés dans la section « En passant » dès la refonte, en temps cumulé
  de sprint et en nombre de sauts marqué « estimation », après relecture du témoin `11de8353`.
- **D-4 — rejeu à la demande.** Retenu : dans ce plan, pour l'application de bureau et le poste de
  développement seulement (l'hébergé garde l'action d'administration existante) ; demandeur : propriétaire du
  profil ou administrateur ; génération immédiate dans la file de régénération existante ; jeton du joueur de
  la page, jamais le premier jeton valide.
- **D-5 — précision par film.** Retenu : le chemin qui ne s'exécute plus est supprimé (lecteurs, écrivain,
  tests) ; la table vide reste en base, son retrait est noté au backlog. Une précision par arme tirée de
  `shotStats` serait un chantier neuf, après la régénération.
- **D-6 — décodeur.** Retenu : aucun changement du décodeur dans ce plan. Un chantier décodeur à part
  regroupe verdict du score, étape manquante de `replay-equiv`, arme d'une touche, porteurs en grande bataille,
  compteurs nommés, lettres des zones, commandes et gestes. Aucun lecteur ne souffre aujourd'hui, sauf le
  porteur du drapeau en grande bataille, qui demande une recherche.

Tranché par le pilote (conventions du dépôt, modifiable par le user) :
- Hook `go-vet` : restreint aux trois cibles de la CI et rendu bloquant.
- Témoin du corpus : `f75e7053` remplacé par `0a247154`.
- Couches : les 5 cas particuliers passent sur une liste d'exemptions datée, le plafond chiffré disparaît.
- Fond `sgh_interlock` : rempart dans LevelUp et correction des deux lignes du fichier publié, sans recuisson
  (une recuisson de cette carte échoue sur un module absent du poste) ; la correction dans la brique est notée
  pour sa prochaine version.

## 3. Règles d'exécution

- Une piste = un agent dans son worktree `C:/t/bl-<piste>`, sous-branche `bl1117/<piste>`, SANS jonction vers
  `data/`. Les agents ne poussent pas et ne fusionnent pas. Cache Go dédié `go-build-bl1117-<piste>`, rendu à la
  clôture par `scripts/disk-hygiene.ps1 -Lot bl1117-<piste> -Apply`.
- Ordre : étape 0, puis A, B, C, D, F et G.0 en parallèle ; G.1 et G.2 après G.0 ; E par le pilote pendant ce
  temps ; puis I, puis R. Dans une piste, l'item N est clos (gate passé) avant N+1.
- Fichiers édités par le pilote seul : `apps/go-api/api/openapi.yaml`, `apps/web/src/lib/api/generated.ts`,
  `docs/CHANGELOG.md`, notes de version, `.ai/thought_log.md`, `.ai/BACKLOG.md`, `.ai/REGISTRE_REPORTS.md`,
  `.ai/PLAN_REFONTE_UI_REJEU_2026-11-15.md`, `apps/web/src/features/dismissableLayers.guard.test.ts`.
- Gate commun Go : `cd apps/go-api && GOWORK=off go build ./... && go vet ./... && go test ./<paquets touchés>/...`
  puis `go test ./internal/archlint/...`. Gate commun web : `make check-types && npx vitest run <dossiers
  touchés> && node tools/lint-cross-feature-imports.mjs && node tools/lint-no-hardcoded-colors.mjs`.
- Un helper neuf vient avec son garde-rail et la migration de toutes les copies dans le même commit.
- Aucune génération de rejeu, aucune recuisson de fond, aucun accès à la prod, sauf G.0 et G.4 (un seul match).
- Reprise de session : ce fichier (cases, journal), `git log --oneline <base>..chore/830-backlog-1117`, puis
  l'entrée la plus récente de `.ai/thought_log.md`.

| Piste | Items | Modèle | Effort | Risque |
|---|---|---|---|---|
| 0 témoins | 3 | pilote (Fable) | rapide | aucun |
| A tests DuckDB | 5 | Sonnet | moyen | faible (tests seuls) |
| B fichiers tactiques et cache | 6 | Sonnet | moyen | moyen (format de fichier ; à fusionner avant la régénération) |
| C petits correctifs | 8 | Haiku (C.1, C.2, C.4, C.6, C.7, C.8), Sonnet (C.3, C.5) | rapide | faible |
| D web | 6 | Sonnet (D.1, D.2, D.5), Haiku (D.3, D.4, D.6) | moyen | faible |
| E rattachements écrits | 6 | pilote, Haiku pour le backlog | rapide | aucun |
| F précision par film | 4 | Sonnet | moyen | faible (chemin non exécuté) |
| G rejeu à la demande | 5 | Opus (G.0), Sonnet (G.1, G.2) | lourd | moyen (jetons, file, bureau) |
| I, R intégration et relecture | 6 | pilote, Fable pour la relecture | moyen | aucun |

## Étape 0 — témoins et vérification sur pièces (pilote)

- [x] 0.1 (fait le 2026-11-17, recalée sur `4f6f249458`) Branche `chore/830-backlog-1117` recalée sur la tête de `origin/release/v8.3.0` du jour du go ;
      création des worktrees des pistes.
- [x] 0.2 (fait le 2026-11-17 par un agent Sonnet en lecture, recoupé par le pilote ; durée du paquet DuckDB mesurée SOUS CHARGE, 578,7 s : la mesure au calme se refait en tête de la piste A) Témoins AVANT, dans `.ai/annexes_backlog_1117/temoins_avant.md` : nombre de tests et de réussites du
      paquet `internal/platform/duckdb` (`go test -tags=integration -json`), sa durée seule au calme ; sorties des
      tests de caractérisation par minute de l'Escouade (`per_minute_caracterisation_test.go`,
      `per_minute_temoins_test.go`) ; compte du garde-rail des couches (17) et des marqueurs de couleur (95, dont
      19 inertes) ; nombre de tests vitest des dossiers touchés par D.
- [x] 0.3 (fait le 2026-11-17 : `.ai/annexes_backlog_1117/verification_0_3.md`, écarts et arbitrages au journal) Chaque référence `fichier:ligne` des pistes est rouverte ; un écart avec ce plan est écrit au journal
      avant tout code. Pour F : inventaire fermé des symboles à retirer (F.0) figé au journal.

## Piste A — paquet `internal/platform/duckdb` : base migrée copiée par test (Sonnet)

- [ ] A.1 `migrated_template_test.go` : un gabarit par cible (`shared`, `player`, `metadata`, `shared_social`,
      `shared_pve`), construit une fois (`sync.Once`, erreur mémorisée et rendue à chaque appelant), dans un
      dossier temporaire créé et retiré par le `TestMain` existant (`migration_provider_test.go`) ; base fermée et
      absence de `.wal` vérifiée avant toute copie ; deux accès : copie du fichier dans `t.TempDir()`, ou base en
      mémoire remplie par `ATTACH` en lecture seule, `COPY FROM DATABASE`, `DETACH`.
- [ ] A.2 Test de fidélité : pour chaque cible, tables, vues, index, séquences et contraintes du gabarit copié
      (fichier et mémoire) égaux à ceux d'une migration neuve ; `RunForDB` rejoué sur la copie n'applique rien.
- [ ] A.3 Helpers rebranchés, liste fermée : `newKillSourceTestPlayerDB`, `newTacticalTestPlayerDB`,
      `newBaseNotee`, `newObjectiveStatsRepoOnMem`, `newNotifTestDB`, `openBattlePassTestDB`, `setupPrestigeDB`,
      `migratedFile`, `openTempMetadataDBWithThresholds`, `setupCatalogRepoDB`, `setupPipelineEnv`, plus les
      appels directs restants recensés en 0.3 (liste écrite au journal avant de coder).
- [ ] A.4 Garde-rail `no_fresh_migration_in_tests_test.go` : `migration.RunForDB(` et `RunForTitleDB(` interdits
      dans les `*_test.go` du paquet hors une liste dont chaque raison commence par une date, avec son test de
      date. Liste de départ : le fichier du gabarit, `career_progression_fresh_migrations_test.go`,
      `pool_shared_social_recovery_concurrent_test.go`, `csr_pipeline_e2e_integration_test.go`,
      `media_loadfiles_real_topology_test.go`.
- [ ] A.5 Mesure APRÈS, au calme, paquet seul : durée et nombre de réussites, comparés à 0.2.
- **Gate A** : `cd apps/go-api && go test -tags=integration -timeout 300s ./internal/platform/duckdb/` vert, même
  nombre de réussites qu'en 0.2, aucun test sauté en plus ; `go test ./internal/platform/duckdb/...` (sans le
  tag) vert ; durée sous 220 s. Au-delà de 220 s : écrit au journal avec la mesure, sans relever le budget.

## Piste B — fichiers tactiques compressés et cache des réapparitions (Sonnet)

- [ ] B.1 Cache par fichier dans `service/replay_spawn_candidats.go`, sur `cacheAEmpreinte` : clé = chemin,
      valeur = la structure maigre des réapparitions, invalidation par date et taille. Tests : le second appel ne
      relit aucun fichier (compteur), un fichier réécrit est relu, un fichier disparu compte « absent ».
- [ ] B.2 `title.ReplaySidecarExt = ".json.gz"` (`internal/domain/title/film_id.go`) ; l'écrivain
      (`replayartifacts.EcrireSidecarRaster`) passe par `replayartifact.Write` ; les trois lecteurs
      (`service/tactical_service_rasters.go`, `service/replay_spawn_candidats.go`,
      `cmd/levelup/cmd_tactical_rasters.go`) passent par `replayartifact.Read` ou `ReadBytes`. Aucune lecture des
      `.json` bruts (même doctrine que les rejeux).
- [ ] B.3 Purges : `scheduler/replay_purge_cron.go` ne reconstruit plus le chemin à la main (fonction du
      résolveur) ; elle et `replaybuild/artifact_purge.go` retirent les deux extensions. À l'écriture d'un
      `.json.gz`, le `.json` voisin est supprimé, avec une ligne de journal ; une suppression en échec est
      journalisée, jamais muette.
- [ ] B.4 Garde-rail : test qui interdit `os.ReadFile` et `os.WriteFile` sur un chemin de fichier tactique hors
      `internal/replayartifact`. Tests adaptés, liste fermée : `cmd_tactical_rasters_test.go`, `raster_test.go`,
      `replay_purge_cron_test.go`, `replay_spawn_forecast_test.go`, `tactical_service_rasters_test.go`, test du
      `PathResolver`.
- [ ] B.5 Mesure écrite au journal : lecture de la Tactique et des réapparitions sur 39 fichiers compressés,
      avec et sans cache (banc dans le paquet, retiré à la clôture s'il n'a pas d'usage courant).
- [ ] B.6 `docs/COMMANDS.md` FR et EN si l'extension des fichiers tactiques y est citée.
- **Gate B** : gate commun Go sur `./internal/domain/title/... ./internal/sync/replayartifacts/...
  ./internal/service/... ./internal/scheduler/... ./internal/replaybuild/... ./internal/replayartifact/...
  ./cmd/levelup/...` ; `grep -rn '"\.json"' apps/go-api/internal/sync/replayartifacts apps/go-api/internal/scheduler/replay_purge_cron.go`
  ne rend aucun chemin de fichier tactique.
- Contrainte : B est fusionnée dans `release/v8.3.0` AVANT la régénération du parc (étape H.4 du plan des
  suites de l'audit), sinon les fichiers sont réécrits deux fois.

## Piste C — petits correctifs Go

- [ ] C.1 (Haiku) `tactical.ZonesDepuisLePort([]domain.ZoneNommee)` dans `internal/analysis/tactical`, avec son
      test ; remplace `zonesPures` (`service/tactical_service_lectures.go`, 6 appels) et
      `zonesPuresDuCatalogue` (`cmd/levelup/cmd_spawn_model_zones.go`). Garde-rail : test qui refuse une
      seconde projection de zones hors de ce paquet.
- [ ] C.2 (Haiku) `internal/migration/psa_index_repro_msr_fixture_test.go:173` et
      `psa_index_repro_msr_planprobe_test.go:34` : la mention de `Q24LUSRHistory` devient « instantané du
      2026-09-27 ; requête supprimée le 2026-11-17 ».
- [ ] C.3 (Sonnet) Escouade : `paceMatch` (`service/teammates/teammates_squad_per_minute.go`) porte frags, morts
      et assistances en pointeurs ; un compte inconnu sort le match du taux (numérateur et dénominateur), comme
      les dégâts. Test rouge puis vert : coéquipier aux frags inconnus. Les témoins de 0.2 ne bougent pas ; s'ils
      bougent, arrêt et retour au user.
- [ ] C.4 (Haiku) `scripts/git-hooks/lefthook/go-vet.sh` : cibles `./internal/domain/... ./internal/analysis/...
      ./internal/games/weapons/...`, sans `|| true` ; commentaires de `lefthook.yml` mis d'accord.
- [ ] C.5 (Sonnet) `cmd/mapfond-build/cuisson.go` (`ecritSidecar`) : une dégradation ne porte plus de chemin
      absolu (nom du module seul) ; test. Les deux lignes de
      `data/titles/halo_infinite/reference/map_backgrounds/sgh_interlock.json` sont corrigées à la valeur que
      produit le rempart. Garde-rail : aucun sidecar publié ne contient de chemin absolu.
- [ ] C.6 (Haiku) `config/replay_corpus.toml` : le témoin `f75e7053` est remplacé par `0a247154`, raison
      réécrite. La ligne fausse de `.ai/REGISTRE_REPORTS.md` sur ce témoin est corrigée par le pilote (E.5).
- [ ] C.7 (Haiku) `docs/adr/0024-lusr-v2-trueskill2-with-counts.md` (en anglais seulement) : 500 itérations
      (lignes 81 et 139) ; limites 4, 5 et 6 marquées livrées avec leurs fichiers ; limite 7 vérifiée puis
      corrigée ou retirée ; cinq liens vers `.ai/archive/V7/LUSR%20v2/`. Amendement daté. Aucune formule touchée.
- [ ] C.8 (Haiku) Thunderhead : le commentaire de `cmd/mapfond-build/sans_plan_test.go:63`,
      `map_fond_reglages.json` et `REGISTRE_CARTES.md` disent « la carte ne publie aucun maillage (vérifié le
      2026-11-17) » et ne citent plus le seuil de 1 000 objets pour ces deux cartes.
- **Gate C** : gate commun Go sur `./internal/analysis/tactical/... ./internal/service/... ./internal/migration/...
  ./cmd/levelup/... ./cmd/mapfond-build/...` ; `bash scripts/git-hooks/lefthook/go-vet.sh` sort en 0 et sort en
  erreur sur une faute de `vet` introduite puis retirée ; `go run ./cmd/replay-corpus-gate` n'est PAS lancé (I.3).

## Piste D — web

- [ ] D.1 (Sonnet, trois lots sans fichier commun) Migration vers `useDismissable`
      (`apps/web/src/lib/hooks/useDismissable.ts`), un test de fermeture par écran :
      lot 1 `media/MediaAudioConfigButton.tsx`, `media/CoverFlowModal.tsx` (le fichier ne grossit pas),
      `media/MediaToolbar.tsx` ;
      lot 2 `asset-drawer/AssetDrawer.tsx`, `feedback-drawer/FeedbackDrawer.tsx`,
      `spartan-customizer/SpartanCustomizerModal.tsx`, `palmares/BattlePassRewardLightbox.tsx`,
      `ascension/campaign/StartCampaignModal.tsx` (la condition « envoi en cours » passe dans `onClose`) ;
      lot 3 `_shared/ExperienceDropdown.tsx`, `_shared/ViewDropdown.tsx`, `explorer/GamertagSearchInput.tsx`
      (`open` = liste visible), `notifications/NotificationsBell.tsx`. Les flèches gardent leur écouteur.
- [ ] D.2 (pilote, après D.1) `dismissableLayers.guard.test.ts` : le plafond chiffré devient une liste
      d'exemptions datée de 5 fichiers (`AssetDockedPanel.tsx`, `TacticalAnalysisView.tsx`,
      `TacticalPlanTueurs.tsx`, `SquadFocusStrip.tsx`, `useSquadPresets.tsx`), chacune avec sa raison ; test de
      date.
- [ ] D.3 (Haiku) Les 19 marqueurs `color-allow` inertes sont retirés (la raison reste en commentaire, sans le
      mot du marqueur) ; liste fermée dans `.ai/annexes_backlog_1117/marqueurs_inertes.md`, écrite en 0.3.
- [ ] D.4 (Haiku) Commentaires faux sur le sprint : `apps/go-api/internal/domain/replaydoc/inventory.go:96-103`
      et `apps/web/src/features/match-replay/model/replayContract.test.ts:131-133`.
- [ ] D.5 (Sonnet, après D.3) `tools/lint-no-hardcoded-colors.lib.mjs` : `shadow`, `from`, `via`, `to` entrent
      dans les préfixes balayés ; genre neuf « marqueur inerte » au plafond 0, câblé dans le script ; tests dans
      `lintNoHardcodedColors.guard.test.ts`, dont la preuve qu'un marqueur inerte est refusé.
- [ ] D.6 (Haiku) Vérification : `palmares/rarity.ts` reste le seul fichier à dégradés nommés, par sa ligne de la
      liste des fichiers autorisés.
- **Gate D** : gate commun web ; `npx vitest run src/features/dismissableLayers.guard.test.ts
  src/lib/accessibility` vert ; la mesure de l'entrée du backlog rend 5 fichiers.

## Piste E — rattachements écrits (pilote ; Haiku pour le backlog)

- [x] E.1 Plan de refonte du rejeu : D-23, D-24, « En passant » et T.5 ne disent plus que le document ne porte
      ni sprints ni sauts ; la visée y est citée comme déjà publiée. Fait le 2026-11-17.
- [x] E.2 (fait le 2026-11-17 : item T.6 du plan de refonte) Plan de refonte, item neuf du lot T (selon D-2) : une seule lecture « médailles du match » depuis
      `medals` du document, pour le fil et la fiche ; libellé et image résolus par le serveur (patron de
      `resolveVehicleLabels`, champ servi déclaré dans `internal/domain/replaydoc`, textes par langue) ;
      substitution, sans fusion avec les événements du match. La phrase « aucune requête nouvelle, aucun
      changement du document » du lot T est amendée.
- [x] E.3 (fait le 2026-11-17 : item T.7 du plan de refonte) Plan de refonte, item neuf du lot T (selon D-3) : temps cumulé de sprint et nombre de sauts
      (« estimation ») dans « En passant », modèle pur ; précédé de la relecture du témoin `11de8353`.
- [x] E.4 (fait le 2026-11-17 : T.0 et T.3 amendés) Plan de refonte, T.0 et T.3 : les mesures se font par build du jeu, après la régénération ; la somme
      des points par action ne s'affiche que si elle égale le score personnel de l'API, en plus du verdict du
      document.
- [ ] E.5 `.ai/REGISTRE_REPORTS.md` : contrôle après la régénération (part des matchs avec rejeu dont le coup
      d'envoi vient du film, attendue proche de 100 %, sinon diagnostic du report avant tout code) ; ligne
      Thunderhead corrigée ; ligne du témoin `f75e7053` corrigée ; les fichiers tactiques s'écrivent compressés.
- [ ] E.6 `.ai/BACKLOG.md` (Haiku, texte dicté par le pilote) : entrées closes (maillage Forge, Thunderhead,
      `flagReturnS`, postures, doublon des tests, et celles des pistes A à D, F, G une fois fusionnées) ;
      entrées reformulées selon la section 1 (tirs lus, arme d'une touche, écarts du gate, ordre des collines,
      marqueurs et lettres des zones, temps forts « mode », commandes et gestes, test instable, médailles) ;
      `.ai/annexes_suites_audit/sources.md` : ligne des règles corrigée, note sur l'échantillon des tirs.
- **Gate E** : aucune entrée de la section 1 sans sort écrit ; relecture croisée du backlog contre ce tableau.

## Piste F — précision par film : retrait du chemin non exécuté (Sonnet, selon D-5)

- [x] F.0 (fait le 2026-11-17 dans `verification_0_3.md`, section F) Inventaire fermé écrit au journal : symboles, fichiers, tests, allowlists et commentaires de
      `sync/killcollector/hits.go`, `games/halo_infinite/ingest/weapon_accuracy_film.go`,
      `persist` (`WeaponHitDistancePersister`) et de leurs appelants ; preuve qu'aucun lecteur de
      `match_weapon_hit_distance` n'existe.
- [!] F.1 (non traité, décision du user du 2026-11-17 : « Reporter au chantier décodeur ». Retirer `hits.go` casse `internal/filmreplis` tant que le décodeur v1.4.1 déclare deux replis tenus par l'hôte ; le plan interdit toute montée de brique. Reporté au registre et au backlog en E.5 et E.6) Suppression des lecteurs, de l'écrivain, de leurs tests et de leurs entrées d'allowlist.
- [!] F.2 (même motif que F.1) Ce qui reste est mis d'accord : commentaire de `match.weapon.accuracy` dans `capabilities.toml`
      (Halo 5 garde ses compteurs natifs), plafond de `archlint/film_facade_surface_test.go` descendu.
- [~] F.3 (couvert par l'entrée de report de F.1 : la table part avec le chemin, au chantier décodeur) La table `match_weapon_hit_distance` n'est pas supprimée (aucune écriture destructive sur la base
      partagée dans ce plan) : une ligne au backlog.
- **Gate F** : gate commun Go sur `./internal/sync/... ./internal/persist/... ./internal/games/...` ;
  `go test -tags=integration ./internal/sync/...` ; `grep -rn "PairWeaponHits\|WeaponHitDistancePersister" apps/go-api`
  ne rend rien ; un test de Halo 5 sur `weapon_accuracy` reste vert.

## Piste G — générer le rejeu d'un match qui n'en a pas (selon D-4)

Périmètre : placement local (application de bureau, poste de développement). Couches : handler, port, service,
file existante ; aucun SQL dans `api/` ni `service/` ; rien ne se décide sur le nom du profil ni sur le slug.

- [x] G.0 (fait le 2026-11-17 : conception relue par le pilote, vérification sur un match passée) (Opus) Conception écrite dans `.ai/annexes_backlog_1117/rejeu_a_la_demande.md` : port
      `port.ReplayGeneration` (demande, état par match), extension de `replayregen.Service` par une file de
      demandes explicites (un film à la fois, verrou `film_decode.lock`, mode jeu, pause), téléchargement et
      archivage du film d'un seul match, jeton du joueur de la page par `auth.RefreshHaloTokensViaStoreFirst`,
      pose du bit « film absent » sur un film introuvable par le persister existant, états servis (générable, en
      file, en cours, échec, film indisponible, non pris en charge). Vérification sur UN match du poste, avec
      le go du user : le manifeste du film se résout avec ce jeton. Relue par le pilote avant G.1.
- [ ] G.1 (Sonnet) Go : port, service sous `internal/service/` (contrôles : capacité `film.replay_artifact`,
      mode géré, film absent, plafond de demandes en vol), handler `POST
      /players/{player_slug}/matches/{match_id}/replay/generate` (202, sous `RequirePlayerOwnership`), capacité
      `can_generate_replays` du bootstrap dérivée du placement et de la capacité du titre (ADR 0044). Un titre
      sans la capacité répond 503 propre (`ErrCapabilityNotSupported`). Journal `slog` à chaque refus et à chaque
      échec. Tests : service sur faux ports (chaque refus, doublon idempotent), handler `httptest` (droits,
      capacité absente), file (ordre, reprise, film introuvable).
- [ ] G.2 (Sonnet) Web : bouton « Générer le rejeu » dans l'état vide de `ReplayPage` et dans l'en-tête de la
      page du match quand le rejeu manque et que la capacité est vraie ; hook dans la feature, clés dans
      `lib/query/keys.ts` ; états repris de l'indicateur « analyse en cours » ; aucun bouton quand l'action est
      impossible ; textes fr et en ; tokens de couleur uniquement ; commentaire périmé de `MatchHeader.replayLink.tsx` corrigé. Tests du hook
      et du rendu par état.
- [ ] G.3 (pilote) `openapi.yaml`, `make generate-types`, instantané de la surface du contrat ; notes de version
      de la 8.3.0 et `docs/CHANGELOG.md` FR et EN ; `docs/COMMANDS.md` si une commande change.
- [ ] G.4 (pilote, avec le go du user) Un match du poste généré de bout en bout par la route, serveur de
      développement ; une ligne dans l'artefact des vérifications pour le contrôle à l'écran du user.
- **Gate G** : gate commun Go sur les paquets touchés, `go test ./internal/archlint/...` ; gate commun web ; une
  relecture adversariale dédiée (skill `adversarial-review`, lot à risque : jetons et file), P0 et P1 corrigés.

## Étape I — intégration (pilote)

- [ ] I.1 Fusion des sous-branches dans `chore/830-backlog-1117` ; `openapi.yaml` et `make generate-types` une
      seule fois.
- [ ] I.2 Témoins APRÈS comparés à 0.2 : écarts listés, chacun rattaché à un item ou déclaré défaut.
- [ ] I.3 `go run ./cmd/replay-corpus-gate` sur le seul témoin `0a247154`, avec le go du user (une cuisson).
- [ ] I.4 `.ai/thought_log.md`, registre des reports, backlog (E.6) ; `docs/CHANGELOG.md` et notes de version
      de la 8.3.0, FR et EN, pour C.3 (taux par minute de l'Escouade) ; caches dédiés rendus.
- **Gate I** : `make gate-push` ; `cd apps/go-api && go test -tags=integration ./internal/sync/... ./internal/platform/duckdb/...` ;
  `make go-api-lint` ; `cd apps/web && npx vitest run` complet.

## Étape R — relecture et clôture

- [ ] R.1 Une ronde de relecture adversariale du diff complet (skill `adversarial-review`, Fable) ; P0 et P1
      corrigés, P2 au backlog seulement s'ils valent le coup.
- [ ] R.2 Tous les items statués ; outillage de mesure retiré ou documenté ; fusion dans `release/v8.3.0`, CI
      verte. La régénération du parc reste le geste du user, après cette fusion.

## Journal d'exécution

- 2026-11-17 — go du user (plan maître, DM-4). E.1 à E.4 exécutés par le pilote dans le plan de refonte, avant
  son audit de conception : D-23 et D-24 corrigés, T.0 et T.3 amendés, items T.6 (médailles lues dans le
  document) et T.7 (sprints et sauts) ajoutés, phrases « aucun code Go » et « aucune requête nouvelle »
  amendées. Vérifié sur pièces avant d'écrire : `Medals` existe dans `domain/replaydoc/document.go`, le patron
  de résolution des libellés est `service/replay_vehicle_labels.go`, et le commentaire de
  `domain/replaydoc/inventory.go:99` dit encore le sprint « réfuté » (D.4 le corrige).

- 2026-11-17 — étape 0 close. Écarts relevés par la vérification sur pièces et arbitrages du pilote :
  - **A** : 30 appels directs dans 26 fichiers (pas 31 dans 27) ; A.3 s'étend aux 13 appels hors des listes (liste dans `verification_0_3.md`) ; `setupPipelineEnv` est rebranché et son fichier ne reste exempté que si une migration neuve y est justifiée, raison datée ; le gabarit `shared_pve` n'est pas construit (aucun test ne l'utilise) ; les migrations neuves indirectes (`GetOrOpen`, `ForTarget` avec `ApplySchema`, `ExecScript`) restent hors du garde-rail et sont notées en découverte.
  - **B** : la liste de tests de B.4 s'étend aux 6 fichiers relevés ; le grep du gate B ne discrimine pas, il est remplacé par `archlint/no_replay_json_literal_test.go` et le garde-rail neuf ; les commentaires de `film_id.go` et de ce test sont mis d'accord ; la commande de contrôle de `docs/COMMANDS.md` (FR et EN) est mise à jour en B.6 ; le gate ajoute `go test ./...` sur tout le module (leçon du 2026-11-17 : un changement de format de fichier se valide sur le module entier).
  - **C** : `zonesPures` a 7 sites d'appel dans le paquet `service` ; C.3 ne vise que la branche des coéquipiers (le joueur principal est en entiers) ; C.5 se corrige côté LevelUp seulement (le chemin absolu naît dans la brique des cartes) ; `REGISTRE_CARTES.md` est sous `.ai/archive/V7.5/cartes/`.
  - **D** : `shadow`, `from`, `via`, `to` entrent dans `TW_PREFIXES` seulement ; `ExperienceDropdown` et `ViewDropdown` gagnent la fermeture par Échap (changement voulu) ; deux entrées mortes de `ALLOWED_FILES` : découverte.
  - **F** : bloquée, voir F.1 (décision du user : reportée au chantier décodeur).
  - **G** : le profil `dev` n'a pas la capacité de régénération ; `can_generate_replays` se dérive du placement (bureau ou développement) et de la capacité du titre, pas de `ReplayRegen` : à écrire en G.0.

- 2026-11-17 — G.0 : conception rendue (`.ai/annexes_backlog_1117/rejeu_a_la_demande.md`, branche `bl1117/g`, `66617729cc`), relue en entier par le pilote. Arbitrages du pilote sur ses questions ouvertes :
  - jeton : le chemin serveur passe par `tokencache.ResolveFreshPlayerTokens(ctx, xuid)`, point d'entrée unique du chemin par joueur dans le processus (échange dédupliqué, aucun marquage « reconnexion requise » sur un échec concurrent), et non par `auth.RefreshHaloTokensViaStoreFirst`, helper des commandes. L'intention de D-4 est tenue : le jeton du joueur de la page, jamais celui d'un autre ;
  - profil suivi sans compte relié : aucun prêt du pool, état « non pris en charge » ; plafond de 10 demandes et 50 issues en constantes nommées ; film retiré après la génération quand l'archive est désactivée ; un échec n'est pas gardé après un redémarrage ; bouton de l'en-tête avec libellé ;
  - contrat : les deux routes ne sont montées que sous la capacité, donc absentes du contrat généré ; G.3 n'ajoute que `can_generate_replays`.
  Reste, avant G.1 : la vérification sur UN match (paragraphe 12 de la conception), avec l'accord du user, serveur et application arrêtés.

- 2026-11-17 — G.0, vérification sur un match, avec l'accord du user (« go, tout est arrêté, cherche le match toi-même ») : aucun processus LevelUp ni port d'écoute relevé avant l'essai ; match sans rejeu du 2026-10-30 (court `33ef54a9`) ; la liste des morceaux du film se résout avec le jeton du joueur : 25 morceaux (1 de type 1, 23 de type 2, 1 de type 3), jeton renouvelé et rangé par le chemin normal ; programme jetable supprimé, arbre propre. À noter pour G.1 : dans `cfg.LoadPlayers`, le slug du profil est le pseudo avec sa casse (`JGtm`), pas sa forme en minuscules.

## Découvertes (consignées, non traitées)

Relevées par l'instruction du 2026-11-17, à porter au backlog en E.6 :

1. `queryKeys.matchReplayContext` ne porte pas la langue alors que la réponse contient des libellés de médailles
   traduits : après un changement de langue, le contexte peut rester dans l'ancienne. Non vérifié à l'écran.
2. Le garde-rail des couches ne balaie que `features/` ; quinze fichiers de `components/` et `lib/` écoutent
   Échap ou le clic dehors à la main.
3. `haloTokensForDrain` (action d'administration) prend le premier jeton valide sans passer par le pool, et son
   message d'erreur conseille de recapturer un jeton, contre l'ADR 0023. À corriger avant toute route ouverte
   aux joueurs sur l'hébergé.
4. La régénération du bureau ne reprend que les rejeux dont le film est encore dans l'archive : un fichier
   tactique d'un rejeu au film expiré resterait à l'ancien format. Vide sur le poste, non vérifié en prod.
5. (2026-11-17) Huit ouvertures `GetOrOpen`, deux `ForTarget` avec `ApplySchema` et un `ExecScript` migrent à neuf dans les tests du paquet DuckDB sans être vus par le garde-rail de A.4.
6. (2026-11-17) `ALLOWED_FILES` de `tools/lint-no-hardcoded-colors.lib.mjs` porte deux entrées mortes (`features/home/rarity.ts`, `features/battlepass/rarity.ts`).
7. (2026-11-17) `ConfigureFilmAccuracy` n'a aucun appelant de production : la passe de précision par film est morte deux fois.
