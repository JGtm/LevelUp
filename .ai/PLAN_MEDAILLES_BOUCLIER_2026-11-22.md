# Plan : Vengeur rétabli, Cible difficile, Os de verre et barème d'impact, bouclier dans la fiche du rejeu (8.3.0)

Ouvert le 2026-11-22. Contrat d'exécution : skill `plan-execution` + complément `levelup-plan`.
Branche : `feat/medailles-bouclier` (depuis `origin/release/v8.3.0`), worktree
`C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-medailles`. Intégration : `release/v8.3.0`.

## Objectif

Après une dérivation du parc, un joueur voit : la médaille Vengeur de nouveau comptée sur les matchs
récents, la médaille Cible difficile, le rôle Os de verre dans la matrice d'impact de l'Escouade avec
le barème rééquilibré, et dans la fiche détaillée du rejeu trois valeurs de bouclier (brisés,
recharges, différentiel) à l'instant et au total du match.

Critère observable : sur un match récent avec rejeu, la vue du match liste Vengeur et Cible difficile ;
la matrice d'impact porte la colonne Os de verre ; la fiche du rejeu affiche les trois valeurs et elles
bougent avec la tête de lecture.

## Décisions (demande du 2026-11-22, messages de l'utilisateur)

- **D-1 Vengeur** (médaille custom existante `9000000001`) : un frag de K sur V compte quand V est le
  tueur de la dernière mort de K. Une vengeance par mort : la mort est « consommée » par le premier
  frag qui la venge. Suicides, trahisons et bots (xuid vide) exclus. Source : fil des morts du match
  (`match_kill_events_latest`, déjà lu par la dérivation, `LireFragsDuFilDesMorts`).
- **D-2 Bris de bouclier** : dans une vie (une piste du document), un bris = la mesure `Sh` passe d'une
  valeur > 0 à 0 (la première mesure de la vie à 0 compte aussi). Le film ne réplique `Sh` qu'au
  changement.
- **D-3 Recharge complète** : après un bris, `Sh` atteint >= 1 (plein ; surbouclier compris) dans la
  même vie. Au plus une recharge par bris. Un respawn n'est jamais une recharge.
- **D-4 Cible difficile** : médaille custom neuve `9000000002`, une médaille par recharge complète
  (D-3). Nom FR « Cible difficile » (nom officiel FR de Halo 5), EN « Hard Target ». Image :
  `C:/Users/Guillaume/Downloads/HI_Hard_Target.png`, copiée en
  `static/medals/halo_infinite/9000000002.png`. Catégorie : celle du Vengeur.
- **D-5 Différentiel** (fiche du rejeu seulement) = recharges − bris, à l'instant et au total.
- **D-6 Os de verre** : rôle de la matrice d'impact, **−0,5 point**. Attribué par match au membre de
  l'équipe alliée qui a le plus de bris de bouclier (D-2) ; à égalité : moins de recharges, puis xuid
  croissant. Aucun rôle sans mesure de bouclier (match sans rejeu). Noms : FR « Os de verre », EN
  « Glass Jaw ». Icône : style des autres rôles (Fluent Emoji Flat, os), pas l'image de médaille.
- **D-7 Barème** : Première victime −1 → −0,5 ; Touriste −1 → −0,5 ; Kamikaze −1 → −0,5. Le reste
  inchangé. Tout texte qui décrit le barème (infobulles, aide) suit.
- **D-8 Où vit le calcul** : dans LevelUp, pas dans le décodeur (aucune montée du décodeur). Une
  fonction pure par règle sous `internal/analysis/` ; mêmes fonctions pour la dérivation, la matrice et
  la vue du rejeu (une seule définition, rien de recalculé côté web).
- **D-9 Stockage** : trois colonnes NULLABLES sur `match_usage_players` (`shield_breaks`,
  `shield_recharges`, `revenge_kills` ; NULL = non mesuré), écrites par la dérivation du résumé
  d'usage, révision du résumé montée (recette de `shieldless_kills`, révision `us7`).
- **D-10 Lecture des médailles custom** : les médailles dérivées du film ne s'écrivent PAS dans
  `medals_earned` (pas de réécriture possible sans UPDATE). Une vue unique fusionne `medals_earned` et
  les colonnes de `match_usage_players_latest` (Vengeur : la valeur dérivée remplace les lignes
  héritées `9000000001` quand elle existe, les lignes héritées restent sinon). Les lecteurs
  d'AFFICHAGE et les citations lisent la vue ; les lecteurs d'ingestion, de validation, de transfert
  et les notifications « première médaille » gardent la table brute. Liste fermée des lecteurs : sortie
  de l'étape 0. Garde-rail : test qui refuse une lecture brute neuve hors allowlist datée.
- **D-11 Fiche du rejeu** : la vue du rejeu sert, par joueur, les instants des bris et des recharges ;
  le web compte ceux <= tête de lecture (instant) et la totalité (match). Emplacement dans la fiche :
  celui que l'étape 0 désigne, dans la grammaire existante de la fiche.

Décisions prises par le pilote après l'étape 0 (2026-11-22) :

- **D-12 Révision propre à LevelUp** (amende D-9) : la révision du décodeur (`replay.UsageSummaryRev`)
  n'est pas touchée. Une constante de révision LevelUp, déclarée à côté de la dérivation, est persistée
  avec la ligne de résumé (colonne neuve nullable, ou suffixe de `summary_rev` si l'étape A.2 montre
  qu'aucun lecteur ne compare `summary_rev` à l'égalité stricte ; le choix et sa raison vont au
  journal). Le rattrapage (`cmd_backfill_usage_summary.go:357`) re-dérive un match quand l'une OU
  l'autre révision diffère. Les trois valeurs passent au persister à côté de
  `replay.UsagePlayerSummary`, sans type du décodeur modifié.
- **D-13 Une seule définition des bris côté fiche** (amende D-11) : la ligne « Boucliers perdus »
  (`model/playerInstantStats.ts`, `shieldDrops`) est remplacée par les instants servis ; le calcul web
  est supprimé avec ses tests. Emplacement : section « Depuis le début du match », trois lignes, valeur
  à l'instant puis total du match en `Note` (proposition 0.5).
- **D-14 Vengeur hérité** (précise D-10) : les 894 matchs porteurs de lignes héritées ont tous un fil
  des morts ; la règle D-1 les remplace partout. La vue de fusion ignore donc les lignes héritées
  `9000000001` dès qu'une valeur dérivée existe pour le match (en pratique : toutes après Z.3). Effet
  visible annoncé à l'utilisateur : les comptes anciens baissent (l'ancien calcul comptait une mort
  plusieurs fois).
- **D-15 `medal_exploit_loader`** : brut (valeur persistée au sync, hors affichage de médailles).
- **D-16 Citations après dérivation** : le recalcul des citations d'un match après la dérivation de son
  film existe (entrée du journal du 2026-10-16, « E4 ») ; A.3 vérifie qu'il couvre la citation
  « Vengeur » lue par la vue, et le prouve par un test.

Reviennent à l'utilisateur : la validation à l'écran, toute opération en prod, la fusion vers `main`.

## Étapes

Statuts : `[ ]` à faire, `[x]` fait, `[~]` couvert ailleurs (référence), `[!]` non traité (justifié).
Aucune case vide à la clôture. Ordre strict dans une piste ; une étape est close quand son gate passe
et que son commit est fait.

### Étape 0 — mesures et cartographie (Fable, lecture seule + test `research`) — moyen

- [x] 0.1 Mesure sur les rejeux locaux (`data/cache/replays/halo_infinite/*.json.gz`, lecture seule) :
  par vie, bris (D-2) et recharges (D-3) ; issue de chaque bris (recharge, mort, fin de vie sans mort,
  ni l'un ni l'autre avec la vie qui continue plus de 10 s) ; distribution par joueur et par match ;
  part des matchs sans aucune mesure `Sh` (modes sans bouclier).
- [x] 0.2 Vengeur (D-1) sur une COPIE de `shared_matches_v2.duckdb` (scratchpad) : distribution par
  joueur et par match ; comparaison aux lignes héritées `9000000001` sur les matchs qui ont les deux.
- [x] 0.3 Barème : moyenne des points d'impact par match et par joueur de l'escouade de JGtm, ancien
  barème contre D-6 + D-7, sur les matchs de la copie (témoin avant/après du lot B).
- [x] 0.4 Inventaire fermé des lecteurs de `medals_earned` (D-10) : chaque fichier classé « vue » ou
  « brut », avec la raison en une ligne.
- [x] 0.5 Câblage : où la dérivation pose les colonnes ; où la matrice lit ses entrées ; quel endpoint
  et quel type servent la fiche du rejeu ; emplacement des trois valeurs dans la fiche.
- Gate : section « Résultats de l'étape 0 » remplie ci-dessous. **Arrêt et retour à l'utilisateur** si
  plus de 10 % des bris restent sans issue alors que la vie continue (mesure de recharge non fiable),
  ou si la médiane de Cible difficile dépasse 6 par joueur et par match.

### Piste A — données (Sonnet) — chemin critique — lourd

- [x] A.1 Fonctions pures : `analysis` bris et recharges par joueur (instants compris, entrée = pistes
  du document) et Vengeur par joueur (entrée = frags). Tests unitaires purs, cas limites de D-1 à D-3.
- [x] A.2 Migration des trois colonnes + vue `_latest` recréée ; dérivation qui les pose ; persister ;
  révision du résumé d'usage montée. Tests d'intégration du persister et de la migration.
- [x] A.3 Médaille `9000000002` : définition et traductions fr/en (forme actuelle des seeds, aucune
  colonne `_fr`/`_en` neuve), image, catégorie ; vue de fusion D-10 ; bascule des lecteurs « vue » ;
  garde-rail de lecture brute.
- Gate : `go test ./...` et `go test -tags=integration ./internal/persist/... ./internal/migration/...
  ./internal/sync/... ./internal/platform/duckdb/...` verts ; `make go-api-lint` sans constat neuf.

### Piste B — matrice d'impact (Sonnet) — après A.2 (colonnes) — moyen

- [x] B.1 Barème D-7 et rôle Os de verre D-6 (`squadimpact` : clé, poids, ordre des colonnes et
  d'empilement) ; lecture des bris par match derrière un port (`platform/duckdb`).
- [ ] B.2 Libellés serveur (`internal/i18n/messages/impact.toml`), libellés web fr/en, icône Fluent
  « os » sous `apps/web/src/assets/badges/fluent-flat/` (validée par l'utilisateur), textes du barème ;
  infobulle d'Os de verre au format des autres rôles (une phrase courte, poids compris), demande de
  l'utilisateur du 2026-11-22.
- [ ] B.3 Témoin : la mesure 0.3 rejouée sur le code livré donne les valeurs « après » de l'étape 0.
- Gate : tests `squadimpact`, `teammates`, `analysis` ; `npx vitest run src/features/squad
  src/components/feedback --maxWorkers=2` ; `make check-types`.

### Piste C — fiche du rejeu (Sonnet) — après A.1 — moyen

- [x] C.1 La vue du rejeu sert par joueur les instants des bris et des recharges (A.1).
- [x] C.2 Fiche : trois valeurs à l'instant et au total (D-5, D-11), libellés fr/en, tokens de
  couleur, état vide quand le match n'a pas de mesure.
- Gate : tests `replayview` ; `npx vitest run src/features/match-replay --maxWorkers=2` ;
  `make check-types` ; `make generate-types` sans diff résiduel.

### Clôture — pilote

- [ ] Z.1 Intégration des pistes sur `feat/medailles-bouclier`, `npx vitest run` complet une fois,
  `go test ./...`, intégration.
- [ ] Z.2 Revue adversariale du diff cumulé (Fable, une ronde ; skills `adversarial-review` +
  `levelup-revue`), P0/P1 corrigés.
- [ ] Z.3 Dérivation locale du parc (`backfill-usage-summary`, lecture des documents, aucune cuisson)
  annoncée à l'utilisateur avant le lancement ; contrôle sur trois matchs.
- [ ] Z.4 `thought_log`, fusion dans `release/v8.3.0`, CI lue après la poussée.
- [ ] Z.5 Validation à l'écran par l'utilisateur.

## Résultats de l'étape 0

Mesures du 2026-11-22. Instruments (tag `research`, lecture seule, copie de la base) :
`apps/go-api/internal/sync/replayartifacts/medailles_bouclier_research_test.go` (0.1 et 0.2) et
`apps/go-api/internal/platform/duckdb/impact_bareme_research_test.go` (0.3, témoin de B.3). Les
commandes sont en tête de chaque fichier.

**Verdict du gate : passé.** Bris sans issue alors que la vie continue plus de 10 s : 1,7 % (seuil
10 %). Médiane de Cible difficile par joueur et par match : 2 (seuil 6).

### 0.1 Bris et recharges (1 455 rejeux, 154 242 vies nommées)

- Rejeux sans aucune mesure de bouclier : 27 (1,9 %). Vies avec au moins une mesure : 85,0 %.
- 127 493 bris. Issue de chaque bris (l'horizon d'un bris est le bris suivant ou la fin de la vie) :

| Issue | Bris | Part |
|---|---|---|
| Recharge complète (`Sh` >= 1) | 29 889 | 23,4 % |
| Mort dans les 10 s | 91 851 | 72,0 % |
| Nouveau bris dans les 10 s, après une remontée partielle | 3 149 | 2,5 % |
| Fin de vie sans mort dans les 10 s | 456 | 0,4 % |
| Nouveau bris après plus de 10 s, sans recharge complète | 1 381 | 1,1 % |
| Mort après plus de 10 s, sans recharge complète | 689 | 0,5 % |
| Fin de vie sans mort après plus de 10 s, sans recharge complète | 78 | 0,1 % |
| **Sans issue, vie continuée plus de 10 s (les trois dernières lignes)** | **2 148** | **1,7 %** |

- Une vie finit par une mort quand sa piste porte `endReason = released` ou qu'une mort du fil des
  morts tombe à moins de 1,5 s de sa fin.
- Avec un seuil de recharge à 0,95 au lieu de 1 : 30 682 recharges (+2,7 %), sans issue 1,2 %. Sur
  les 2 148 bris sans issue, le bouclier maximal revu ensuite a pour médiane 0,89 (p90 0,98) : la
  plupart remontent presque à plein sans que le film écrive la mesure pleine.
- Délai du bris à la recharge complète : p10 7,0 s, médiane 7,0 s, p90 8,9 s, p99 23 s. Ce délai
  très régulier montre que le film écrit bien la mesure pleine dans le cas général.
- Par joueur et par match (14 007 lignes, 1 428 rejeux mesurés) :

| Grandeur | Moyenne | p25 | Médiane | p75 | p90 | p99 | Max | À zéro |
|---|---|---|---|---|---|---|---|---|
| Bris | 9,10 | 6 | 9 | 12 | 15 | 22 | 34 | 2,2 % |
| Recharges (Cible difficile) | 2,13 | 1 | 2 | 3 | 4 | 7 | 14 | 19,2 % |

- Par match : bris médiane 81 (moyenne 89, p90 130, max 511) ; recharges médiane 19 (moyenne 21,
  p90 32, max 104).
- Première mesure de la vie déjà à 0 (comptée comme bris par D-2) : 10 670 bris (8,4 %), dont 30
  seulement dans la seconde qui suit la naissance de la vie. Aucun rejeu mesuré n'a ses bris faits à
  80 % de premières mesures à 0 : les modes sans bouclier sont les 27 rejeux sans aucune mesure, la
  règle D-2 n'y fabrique pas de bris.

### 0.2 Vengeur (fil des morts de la copie : 1 462 matchs, 146 445 lignes)

- Écartés par D-1 : 2 362 lignes avec un bot ou un nom non résolu, 0 suicide, 2 trahisons. L'équipe
  vient de `match_participants.team_id` : le fil des morts ne la porte pas, la fonction pure de A.1
  doit la recevoir en entrée.
- Par joueur et par match (15 637 lignes) : moyenne 1,83, médiane 2, p75 3, p90 4, p99 6, max 10 ;
  27,4 % à zéro. Par match : médiane 20, p90 26, max 49.
- Lignes héritées `9000000001` : 7 147 lignes sur 894 matchs, du 2023-04-15 au 2026-03-18. Les 568
  autres matchs du fil des morts (du 2025-08-03 au 2026-11-10) n'en ont aucune : c'est le trou que
  la dérivation comble.
- Sur les 894 matchs qui ont les deux (10 292 joueurs-matchs) : égal 54,9 % ; dérivé inférieur à
  l'hérité 45,1 % ; dérivé supérieur : jamais. Total hérité 42 680, total dérivé 17 476 (2,4 fois
  moins). Hors égalité, les couples les plus fréquents sont (hérité 4, dérivé 1), (8, 2), (3, 1),
  (12, 3), (2, 1), (16, 4) : l'hérité vaut souvent 4 fois le dérivé. La valeur héritée est gonflée ;
  la règle D-1 n'en trouve jamais plus.

### 0.3 Barème (escouade de JGtm, 643 matchs)

Coéquipiers retenus (le plus de matchs dans l'équipe de JGtm) : Madina97294 (618), Chocoboflor (496),
XxDaemonGamerxX (39). Population : les 643 matchs où JGtm partage son équipe avec au moins un des
trois ; 618 ont une mesure de bouclier (rejeu local), donc D-6 est calculé. Points par match = total
des points du joueur divisé par ses matchs dans la population.

| Joueur | Matchs | Ancien barème | D-7 seul | D-6 + D-7 | Os de verre reçus |
|---|---|---|---|---|---|
| JGtm | 643 | −0,736 | −0,378 | −0,519 | 182 |
| Madina97294 | 618 | +0,330 | +0,700 | +0,535 | 204 |
| Chocoboflor | 496 | −0,501 | −0,150 | −0,215 | 64 |
| XxDaemonGamerxX | 39 | −1,410 | −0,897 | −0,936 | 3 |

Sur les 38 matchs où les quatre jouent ensemble (ancien / D-7 seul / D-6 + D-7) : JGtm −0,882 /
−0,500 / −0,724 ; Madina97294 +0,105 / +0,434 / +0,289 ; Chocoboflor −0,461 / −0,211 / −0,276 ;
XxDaemonGamerxX −1,368 / −0,868 / −0,908.

Os de verre tombe sur un membre de l'escouade dans 453 des 618 matchs mesurés, sur un allié hors
escouade dans 164 (le rôle se décide sur l'équipe alliée complète, comme les autres rôles), et n'est
pas attribué dans 1 (aucun bris allié).

Écarts de l'instrument avec le service (`teammates.rolesDesMatchs`) : mêmes requêtes (Q32 avec son
repli, Q32b, Q32e) et même `squadimpact.RolesOfMatch`, mais la population n'a pas les filtres de la
page (dates, modes, sessions) et se lit en un seul groupe ; le T0 du match n'est pas retranché (les
rôles n'en dépendent pas) ; les coéquipiers sont choisis par nombre de matchs communs. Le barème
candidat est une table dans l'instrument : au lot B.3 on la vide, le poids d'Os de verre vient de
`squadimpact.Weight`, et les colonnes « ancien » et « candidat » doivent alors donner les valeurs
« D-7 seul » ci-dessus.

### 0.4 Lecteurs de `medals_earned` (Go hors tests)

| Fichier | Classe | Raison |
|---|---|---|
| `platform/duckdb/queries_match.go:163` (Q14, `match_view_repo_medals.go` GetMatchMedals) | vue | médailles du joueur dans la vue du match (critère observable du plan) |
| `platform/duckdb/queries_match_detail.go:227` (Q27, GetMatchBulkMedals) | vue | médailles de tous les joueurs du match |
| `platform/duckdb/queries_home_citations.go:149` (Q26h, `home_repo_medals_citations.go`) | vue | médailles des tuiles de match de l'accueil et des sessions |
| `platform/duckdb/queries_citations.go:41` (Q36a, lu par `citations_repo.go` et `medals_repo.go`) | vue | totaux par médaille : page Citations et catalogue des médailles |
| `platform/duckdb/medals_by_xuid_repo.go:123` | vue | médailles par joueur et par match (résumé des médailles de l'Escouade) |
| `platform/duckdb/explorer_repo.go:211` (GetMedalCountsForMatches) | vue | total et nombre de médailles distinctes affichés dans l'Explorateur |
| `platform/duckdb/explorer_repo_medals.go:34` (GetTopMedalsForMatches) | vue | médailles les plus gagnées affichées dans l'Explorateur |
| `sync/citations.go:216` (loadMedalsForMatch) | vue | citations calculées par match, dont « Vengeur » (`avenger` -> 9000000001) |
| `sync/medal_exploit_loader.go:65` | à trancher (A.3) | score d'exploit calculé au sync, pondéré par la difficulté de `medal_definitions` : le Vengeur hérité y entre déjà (difficulté « Normal ») ; valeur persistée, donc brut par défaut |
| `platform/duckdb/player_matches_repo.go:334` | brut | somme filtrée sur les médailles « élimination parfaite » (identifiants natifs) |
| `platform/duckdb/queries_squad.go:96` (Q30) | brut | idem, éliminations parfaites |
| `platform/duckdb/queries_match.go:60, 245, 298, 367` (Q12, Q29, Q29 groupé, Q19c) | brut | idem, éliminations parfaites |
| `platform/duckdb/queries_home_citations.go:42` (Q26) | brut | idem, éliminations parfaites |
| `platform/duckdb/compare_repo.go:119` | brut | idem, éliminations parfaites |
| `platform/duckdb/vehicle_commendation_stats_repo.go:341` | brut | somme filtrée sur les médailles de détournement (identifiants natifs) |
| `sync/comeback.go:397` | brut | filtre sur une médaille native (Steaktacular) |
| `platform/duckdb/player_snapshot_store.go:264` (appelé par `api/wire/post_sync_medal_first_earned.go`) | brut | notification « première médaille » (D-10) |
| `sync/backfill.go:205` | brut | ingestion : matchs dont les médailles restent à charger |
| `sync/invariants/invariants.go:299, 417` | brut | validation : lignes orphelines, citations manquantes |
| `validation/gate.go:260` | brut | validation : liste des tables contrôlées |
| `transfer/format.go:78` | brut | transfert : table copiée avec sa clé |
| `ops/snapshot_export.go:24`, `ops/seed_demo.go:189` | brut | export et jeu de démonstration : copie de table |
| `cmd/h5-backfill/main.go:150` | brut | diagnostic : compte de lignes |

Soit 8 lectures « vue » dans 8 fichiers, 19 lectures « brut » dans 15 fichiers (dont deux qui portent aussi une lecture « vue » : `queries_match.go` et `queries_home_citations.go`) et 1 à trancher. Les autres
fichiers qui citent `medals_earned` sont des écritures (`persist/shared_persister.go`,
`ops/seed_demo_synthetic_shared.go`, ingestion Halo 5, import OpenSpartan), des migrations, des
schémas ou des commentaires.

### 0.5 Câblage

Résumé d'usage (chemins sous `apps/go-api/`) :
- Projection : `internal/sync/replayartifacts/usage.go:229` (`projeterResumesUsage` :
  `replay.BuildUsageSummary` puis `replay.AttachShieldlessKills`, ligne 233). Le document et les
  frags sont sur `artefactLu` (`derivations.go:105-113`, champ `frags`), lus par `usage_frags.go:25`
  (`LireFragsDuFilDesMorts`) et `:63` (`preparerFragsDuResume`).
- Rattrapage : `cmd/levelup/cmd_backfill_usage_summary.go:350-366` (même projection ; saute un match
  quand `rec.rev == replay.UsageSummaryRev` et que le schéma colle, ligne 357).
- Persister : `internal/persist/usage_summary_persister.go:147` (`insertUsagePlayerRows`), valeurs
  lignes 176-183, requête `insertMatchUsagePlayerSQL` ligne 251.
- Migration : `internal/migration/steps_shared_usage_summary.go:62-67` (enregistrement de
  `shieldless_kills`), `:249` (`applyMatchUsageShieldless` : `addColumnIfMissing` puis recréation de
  la vue `:229`, qui fige son `SELECT *`) ; ordre dans `internal/migration/order.go:339`.
- Révision : `replay.UsageSummaryRev = "us7"` est une constante DU DÉCODEUR
  (`halo-infinite-film-decoder@v1.4.1/replay/usage_summary.go:71`), comme le champ
  `UsagePlayerSummary.ShieldlessKills` (`:125`) et `AttachShieldlessKills`. Voir « Décisions que les
  mesures contredisent ».
- Lecteur existant d'une colonne du résumé pour une citation : `internal/sync/citationsfilm/stats.go:33`.

Matrice d'impact :
- Entrée : `internal/service/teammates/teammates_squad_impact.go:146` (`rolesDesMatchs`), appelée
  par `buildSquadImpact` (`:103`, appelé en `teammates_service_sections.go:144`). Événements :
  `teammates_squad_charts_medal_digest.go:50` -> port `LoadImpactEventsParGroupes`
  (`internal/port/repository_data.go:48`, `internal/platform/duckdb/squad_repo.go:372`, Q32
  `queries_squad.go:178`). Équipe alliée : `LoadMainTeamParticipants` (`repository_data.go:76`,
  `squad_repo_synthesis.go:14`, Q32b `queries_squad.go:309`). Voleur :
  `teammates_squad_impact_thief.go:16` -> `LoadSquadKillLog` (`squad_repo_kill_log.go:56`).
- Rôles, ordre et barème : `internal/analysis/squadimpact/roles.go` (clés `:17-28`, `roleOrder`
  `:31`, `stackOrder` `:39`, `weights` `:46`, `RolesOfMatch` `:91`). Le port des bris par match de
  B.1 se pose à côté de `LoadSquadKillLog`, se lit dans `rolesDesMatchs` et entre dans `MatchInput`.
- Libellés : `internal/i18n/messages/impact.toml` ; web `features/squad/i18n/{fr,en}.ts`,
  `lib/i18n/manifests/squad.toml`, `features/squad/SquadImpactScoreboard.tsx`,
  `features/squad/impact/impactHistoryStrings.ts`, icônes `components/feedback/BadgeIcon.tsx` et
  `assets/badges/fluent-flat/`.

Fiche du rejeu :
- Endpoint : `GET /matches/{match_id}/replay` (`internal/api/handlers/replay.go:51`,
  `handleGetReplay` `:116`, corps `replaydoc.ReplayDocument`). Service :
  `internal/service/replay_service.go:208` (`GetReplay`) ; conversion `replayview.FromArtifact`
  (`internal/service/replayview/replayview.go:29`, appelée en `replay_service.go:254`), puis les
  champs posés à la requête (`resolveMedalLabels`, `:257`). Type servi :
  `internal/domain/replaydoc/document.go:14` ; port `internal/port/services.go:176`
  (`ServedReplay`) et `:185`.
- Un champ servi sans source dans le document stocké se déclare dans
  `internal/testutil/replay_servis_seuls.go:29` (`ReplayChampsServisSeuls`, précédent :
  `MedalLabels`), que lit `replayview/parity_test.go`. C'est la voie de C.1.
- Fiche : `apps/web/src/features/match-replay/ui/ReplayPlayerPanelBody.tsx:118-131` ordonne les
  sections ; tout se lit par `hooks/usePlayerInstant` -> `model/playerInstantStats.ts` et
  `model/playerInstantView.ts`.
- Emplacement des trois valeurs : la section « Depuis le début du match »
  (`ui/fiche/FicheDepuisDebut.tsx`) porte déjà la ligne « Boucliers perdus » (`:34`), comptée
  jusqu'à la tête de lecture. Aucune section n'écrit un couple instant / total sur une même ligne :
  l'instant vit dans « Depuis le début du match », le total officiel dans « Totaux du match »
  (`ui/fiche/FicheGroupes.tsx:16`, alimenté par la feuille de match, `model/playerMatchTotals.ts`).
  Proposition : les trois lignes (brisés, recharges, différentiel) à la place de « Boucliers perdus »
  dans « Depuis le début du match », la valeur à l'instant suivie du total du match en texte discret
  (`Note`, `ui/fiche/ficheParts.tsx:76`), comme le rang suit le score. La forme exacte reste à
  valider à l'écran par l'utilisateur (Z.5).

### Décisions que les mesures contredisent

1. **D-9 contre D-8 (révision du résumé).** La « recette de `shieldless_kills` » passe par le
   décodeur : la révision `us7`, le champ de la ligne et la fonction d'attache y vivent. Monter cette
   révision exige une version du décodeur, que D-8 exclut. Sans montée, il faut une révision propre à
   LevelUp (par exemple une révision LevelUp accolée à celle du décodeur dans `summary_rev`, comparée
   dans le rattrapage `cmd_backfill_usage_summary.go:357`) et un persister qui reçoit les trois
   valeurs à côté de `replay.UsagePlayerSummary`. À trancher avant A.2.
2. **D-8 et D-11 contre la fiche actuelle.** La ligne « Boucliers perdus » est déjà calculée par le
   web (`model/playerInstantStats.ts:244`, `shieldDrops`). Sa règle est presque celle de D-2 (elle
   n'écarte que la première mesure à 0 posée à l'image de naissance, 30 cas au plus sur le parc),
   mais c'est une seconde définition. C.2 doit la remplacer par les instants servis, sinon la fiche
   porte deux calculs de bris.
3. **D-10, Vengeur sur les matchs anciens.** La valeur dérivée remplace une valeur héritée 2,4 fois
   plus haute : 45 % des joueurs-matchs anciens verront leur compte de Vengeur baisser, aucun ne le
   verra monter, et le total de la citation « Vengeur » baisse d'autant. Ce n'est pas une
   contradiction, mais c'est visible : à dire à l'utilisateur avant Z.3.

Les autres décisions tiennent : D-2 et D-3 sont mesurables (1,7 % de bris sans issue), D-4 donne une
médaille d'ordre 2 par joueur et par match, D-6 attribue un rôle dans chaque match mesuré.

## Découvertes

Hors périmètre : notées ici, jamais corrigées dans ce plan.

- Les lignes héritées du Vengeur (`medals_earned`, `9000000001`) sont gonflées : 2,4 fois la règle D-1 en total, souvent 4 fois par joueur et par match ; aucun code Go ne les écrit plus (dernière ligne le 2026-03-18).
- Les citations par match (`sync/citations.go`, table `match_citations` du joueur) sont calculées au sync, avant la dérivation du rejeu, et écrites une seule fois : la citation « Vengeur » d'un match récent restera à 0 si rien ne la recalcule après la dérivation. À regarder au lot A.3 (bascule de `loadMedalsForMatch` sur la vue).
- Le garde-rail d'exclusion de la campagne (`internal/archlint/campaign_exclusion_guard_test.go:82`) reconnaît `medals_earned` par mot entier : une vue de fusion au nom suffixé lui échappe. À couvrir avec le garde-rail de A.3.
- Les rôles d'impact sont aussi nommés hors de l'Escouade (`analysis/narrative/impact_roles.go`, `features/match-view/MatchImpactBadgesBar.tsx`, `lib/i18n/manifests/profile.toml`) : y chercher un texte de barème au lot B.2.
- À vérifier au lot A.3 : la vue de fusion est posée par une migration `TargetShared` ; si la base partagée de Halo 5 la reçoit, elle doit y rendre `medals_earned` seul quand `match_usage_players` est vide ou absente.

## Reprise de session

Lire ce fichier (statuts, résultats de l'étape 0, journal ci-dessous), puis `git log --oneline -10` dans
le worktree.

## Journal

- 2026-11-22 : plan ouvert, décisions D-1 à D-11 prises avec l'utilisateur.
- 2026-11-22 : étape 0 close. Gate passé (bris sans issue 1,7 %, médiane de Cible difficile 2). Deux
  points à trancher avant la piste A : la révision du résumé d'usage vit dans le décodeur (D-9 contre
  D-8), et la fiche calcule déjà « Boucliers perdus » côté web (C.2 le remplace). Instruments
  `research` gardés comme témoins du lot B.3.
- 2026-11-22 : A.1 close. Paquets `internal/analysis/shieldcycle` (`Cycles(lives []Life) map[string][]Event`,
  `Totals`, types `Life`, `Measure`, `Event`, `Kind` = `Break` | `Recharge` ; instant = image du film, clé
  présente = joueur mesuré) et `internal/analysis/revenge` (`Count(frags []Frag, teamOf map[string]int)
  map[string]int`). `analysis/` ne peut pas importer le décodeur : l'adaptateur document -> vies est
  `internal/replayshield` (`Lives(*replay.ReplayDocument)`, `FrameMS(frame, intervalMS)`), appelé par A.2
  (dérivation) et C.1 (vue du rejeu). L'entrée n'a pas d'image de début ni de fin de vie : les règles D-2
  et D-3 n'en ont pas besoin.
- 2026-11-22 : A.2 close. Migration `shared_match_usage_levelup_v1` : `shield_breaks`, `shield_recharges`,
  `revenge_kills` (INTEGER nullables) sur `match_usage_players`, `levelup_rev` (VARCHAR nullable) sur
  `match_usage_films`, les deux vues `_latest` recréées. Forme de la révision D-12 : COLONNE NEUVE
  `match_usage_films.levelup_rev`, constante `replayartifacts.UsageLevelUpRev = "lu1"` (à changer dès qu'une
  règle de `shieldcycle`/`revenge` ou leur jointure change). Raison : le rattrapage est le seul lecteur de
  `summary_rev` et il le compare à l'égalité stricte (`cmd_backfill_usage_summary.go`), donc un suffixe
  aurait mêlé deux révisions indépendantes dans une chaîne ; une colonne se compare séparément et se lit
  NULL sur les passes d'avant (re-dérivées). Une seule fonction partagée,
  `replayartifacts.AttacherMesuresLevelUp` (fil de l'eau `usage.go` ET rattrapage), rend
  `persist.UsageLevelUp` (révision + mesures par xuid) ; `PersistPass` prend ce 4e argument (valeur zéro =
  tout NULL). Lectures neuves avant tout writer : `LireEquipesDesMatchs` (`match_participants.team_id`,
  même segment de lecture que le fil des morts). NULL quand non mesuré : bouclier = match dont aucune piste
  ne porte `Sh`, ou joueur sans piste ; vengeances = fil des morts non lu OU équipes non lues (les
  trahisons ne s'écarteraient pas). Dans un match mesuré, un joueur avec piste et sans mesure `Sh` vaut 0
  (son bouclier n'a jamais bougé). Un joueur sans ligne de résumé reçoit une ligne neuve seulement s'il a
  une valeur non nulle (précédent `AttachShieldlessKills`) ; un joueur à zéro partout n'en reçoit pas, et
  le lecteur B lira « match mesuré » par les autres lignes. `canonicalOrderFloor` relevé à 241.
- 2026-11-22 : A.3 close. Médaille `9000000002` : step `seed_custom_hard_target_medal` (metadata, titre Infinite
  seul, `games/halo_infinite/migrations/steps_medal_hard_target.go`), placé APRÈS
  `metadata_definition_texts_to_translations_v1` dans `canonicalOrder` : la forme du seed du Vengeur
  (colonnes `name_fr`/`name_en` de `medal_definitions`, migrées ensuite) est impossible sur une base de prod,
  dont ces colonnes ont été retirées. Définition sans colonne de texte (custom, difficulté Normal, catégorie
  skill comme le Vengeur) ; textes fr « Cible difficile » / en « Hard Target » et descriptions dans
  `medal_translations` par le helper neuf `migration.UpsertMedalTexts` (même forme que `UpsertAssetTexts`).
  Image `static/medals/halo_infinite/9000000002.png` (256x256, comme `9000000001.png` ; le garde d'icônes
  ne couvre que la taxonomie, la médaille custom est hors table). Vue de fusion `v_medals_earned`
  (migration `shared_medals_earned_view_v1`, TargetShared, juste après `shared_match_usage_levelup_v1`) :
  colonnes match_id, xuid, medal_name_id, count ; Vengeur dérivé remplace les lignes héritées dès qu'une
  ligne du match porte `revenge_kills` non NULL (une valeur dérivée à zéro n'est pas une ligne) ; Cible
  difficile = `shield_recharges` > 0 ; sans colonnes dérivées ou sans résumé d'usage (base partagée de
  Halo 5 : elle reçoit les steps globaux de la cible shared) la vue rend `medals_earned` seul ; sans table
  de médailles, rien n'est posé. Les 8 lecteurs « vue » de 0.4 lisent la vue (queries_match Q14,
  queries_match_detail Q27, queries_home_citations Q26h, queries_citations Q36a, medals_by_xuid_repo,
  explorer_repo, explorer_repo_medals, sync/citations) ; `medal_exploit_loader` reste brut (D-15).
  Garde-rails : `archlint/medals_earned_lecture_brute_test.go` (liste fermée et datée des 19 lectures brutes
  + exploit, compte exact par fichier, entrée périmée refusée) et regex du garde d'exclusion de campagne
  élargie à `v_medals_earned` (le mot entier `medals_earned` ne la reconnaissait pas). D-16 : le recalcul des
  citations après la dérivation (clé `match_usage_films_latest.written_at` plus récent que les citations)
  couvrait déjà la citation lue par la vue ; test `citations_vengeur_vue_integration_test.go` (Vengeur hérité
  7 avant film, 2 après, lu par la vue). `canonicalOrderFloor` relevé à 243. Corrigé au passage (A.1,
  bloquait le lint) : De Morgan dans `revenge.vengeable`.

- 2026-11-22 : piste C fusionnée (`da64d02f26` C.1, `e9b59a7b0d` C.2). Champ servi `shieldCycles` (xuid -> `{breaks, recharges}` en ms sur l'axe du document, origine non retranchée) ; bots servis sous la clé `bot:<nom>` du roster (règle « les bots comptent partout »). Gate de la piste : `go test` service/api/replayshield/archlint/contracttest verts, vitest `match-replay` 5802 verts, typecheck, lint, couleurs, imports croisés et knip verts ; test du service relancé par le pilote après gofmt : vert.
- 2026-11-22 : B.1 close. `squadimpact` : clé `glass_jaw` (−0,5), Première victime / Touriste / Kamikaze à −0,5 (D-7). `stackOrder` suit la règle écrite (pertes de la plus forte à la plus faible, maquette à barème égal) : Boulet, Faux-frère, Voleur (−1, qui passe devant les trois à −0,5), Première victime, Touriste, Kamikaze, Os de verre. `roleOrder` (colonnes de la matrice) n'a pas de règle écrite au-delà de « canonique » : Os de verre est placé après Voleur, avant Bourreau (choix du lot, à redire si la maquette en veut un autre). Attribution D-6 dans `squadimpact.GlassJaw` (pur) : parmi l'équipe alliée complète, le plus de bris, puis moins de recharges, puis xuid croissant ; aucun rôle sans mesure (`MatchInput.Shields` vide) ni sans bris allié ; `RolesOfMatch` l'émet avant le Voleur. Lecture : port `SquadRepository.LoadMatchShields` (à côté de `LoadSquadKillLog`), Q32f sur `match_usage_players_latest` seule, lignes dont les deux colonnes sont NULL écartées (un match est mesuré dès qu'une ligne porte une valeur), dégradation journalisée en (nil, nil) ; lue UNE fois pour tous les matchs dans `rolesDesMatchs` (`teammates_squad_impact_shields.go`), donc par le même chemin pour la matrice et les points par soirée (même map de rôles, `BuildHistory`). Tests : purs (`roles_test.go`), service avec port simulé (parité matrice / soirée, match non mesuré, lecture en échec), repo sur DuckDB mémoire au schéma livré (`squad_repo_shields_test.go`).
