# PLAN — Source unique de l'arme d'un kill, et rejeu 2D au fil de l'eau

> Date : 2026-09-01. Auteur : session « armes/rejeu ». Contrat d'exécution : skill
> `plan-execution` (ordre strict, aucun report d'une étape exécutable, statut sur chaque item,
> vérification sur pièces, zéro fix hors périmètre).
>
> Deux volets INDÉPENDANTS. Ils peuvent être exécutés en parallèle dans deux worktrees
> distincts. Aucune étape du volet B ne dépend du volet A et réciproquement.

---

## 0. Décisions tranchées AVANT exécution

Ces décisions sont prises. Un exécuteur ne les rouvre pas ; s'il pense qu'une est fausse, il
l'inscrit en section 6 (Découvertes) et continue.

| # | Décision |
|---|---|
| **D1** | Le producteur `weapon_kills` de **Halo Infinite** (corrélation tirs de l'attaquant ↔ instant du kill, `internal/sync/backfill_weapons.go`) est **SUPPRIMÉ**, pas désactivé, pas mis derrière un drapeau. |
| **D2** | La table `weapon_kills` et la vue `v_weapon_kills` sont **SUPPRIMÉES du fichier `halo_infinite`**, par une migration *title-owned*. **AUCUNE ligne n'est conservée, aucune arme de kill de l'ancien parser ne survit — zéro** (décision utilisateur du 2026-09-01). On ne vide pas la table : on la fait disparaître, ainsi que la vue, pour qu'aucun lecteur ne puisse la retrouver. Elles **RESTENT pour Halo 5** : 550 926 lignes, `confidence = "native"`, issues de la timeline de l'API H5 — donnée autoritaire, sans rapport avec la corrélation défaillante. |
| **D3** | Sur Halo Infinite, l'identité canonique d'une arme de kill devient la **`weapon_key` du registre**, résolue depuis `match_kill_events_latest.source_tag` par `film/killicon`, **à la lecture et jamais stockée**. Les identifiants 64 bits `filmshell` ne servent plus qu'à Halo 5. |
| **D4** | **Le TOTAL des classes mêlée / grenade / capacités spartanes reste servi par les compteurs API** (autoritaires, présents même sans film) — la source de dégât ne le sert PAS, sinon double comptage. **MAIS la VENTILATION de niveau 2 vient du film**, réconciliée à ce total : c'est déjà la mécanique appliquée aux grenades (`grenadeRoles`), on ne change que sa source. Mesure du 2026-09-01 qui fonde cette correction (une première rédaction excluait le film de ces classes, à tort) : le tag distingue **4 types de grenade** par son entrée dans la liste `gggl` du jeu — fragmentation 3 048 morts, dynamo/choc 1 538, plasma 1 482, à pointes (Banished) 411, plus 1 tag AMBIGU à 24 morts non publiable, soit **6 479 / 6 504 = 99,6 % typées** là où l'API ne rend qu'un total. |
| **D4bis** | **LA MÊLÉE NE SE VENTILE PAS PAR ARME, ET C'EST MESURÉ — ne pas le retenter.** Les 6 tags de mêlée observés portent tous la mention « effet partagé par N weap », N valant 12 à 47 : l'effet de dégât de mêlée est GÉNÉRIQUE dans le moteur. Le film sait qu'il y a eu mêlée, jamais avec quoi. Ce n'est pas une lacune du décodage. En revanche `source_category` porte une distinction que l'API n'a pas : **assassinat 1 774 contre mêlée ordinaire 10 541**, et sur les armes à feu **30 321 tirs à la tête** et 3 405 dégâts par collage. **Hors périmètre de ce lot**, inscrit en section 6. |
| **D5** | Divergence crédit/source (1 921 morts, 1,75 %) : on compte au **CRÉDIT du kill-feed** (`feed_killer_xuid`) avec l'**arme de la SOURCE**. C'est déjà la convention de `KillSourceClassRepo` et celle du reste de l'app. |
| **D6** | Les passes de décodage **non publiables ligne à ligne** (31 335 morts, 28,6 %) sont **COMPTÉES en agrégat**. Décision déjà en vigueur (`port/kill_source_class.go`), on ne la rouvre pas. |
| **D7** | Une source qui ne résout **aucune** clé de registre reste dans « Non attribué ». On ne devine pas, on ne proratise pas. |
| **D8** | **Aucune cuisson d'artefact de rejeu dans ce plan.** Le volet B pose la mécanique et les garde-fous ; il ne génère rien. Toute génération, même d'un seul artefact de vérification, exige l'accord explicite de l'utilisateur (sinistre RAM du 2026-08-31). |
| **D9** | `internal/analysis/weaponv3` n'est **PAS** supprimé en bloc : `replay/loadouts.go` et `replay/ground_weapon_rules.go` en utilisent la **table de noms** (`WeaponName`, `KnownWeaponHigh32`). Seule la partie **corrélation** part. |
| **D10** | Le niveau 2 du sunburst pour les classes d'arme à feu reste **PAR RÔLE** (automatique / précision / sniper / …), comme aujourd'hui. Pas de bascule vers un niveau 2 par arme dans ce plan. |
| **D11** | **UN SEUL chemin de chargement.** Aujourd'hui les six surfaces chargent la répartition des frags par DEUX chemins : `WeaponKillsRepo` (armes à feu, depuis `weapon_kills`) et `KillSourceClassRepo` + `killsourceload` (répulseur / bobines / chute, depuis la source de dégât), avec une mécanique anti-double-comptage pour qu'ils ne se recouvrent pas. Les deux **FUSIONNENT** dans le nouveau lecteur : le filtre « clé sans identifiant numérique », la règle anti-double-comptage et le paramètre `sources` de `fragdist.Build` disparaissent avec eux. |
| **D13** | **LE GRAPHE ET LE KILL FEED DOIVENT DIRE LA MÊME CHOSE DU MÊME KILL.** Ils partagent déjà LE MÊME dictionnaire (`killicon` + `damagetag`, tables embarquées, `killsource_registry.go` le dit explicitement) — mais ils lui posent des questions différentes : le feed demande une IMAGE, le graphe une CLÉ DE REGISTRE. **19 règles sur 55 rendent une image SANS clé** (mesure du 2026-09-01) : les 3 entrées NOM `Infected Energy Sword` / `Mythic Sandwich` / `Mutilator`, l'entrée GGGL n°3, le repli CLASSE `MELEE`, et surtout **les 14 entrées BANQUE — tous les véhicules et toutes les tourelles** (Ghost, Banshee, Wraith, Phantom, Chopper, Wasp, Scorpion, Rockethog, Pelican, Falcon x2, mitrailleuse, canon plasma, tourelle Shade). Sur ces sources, le feed montre une icône et le graphe dirait « Non attribué ». Volume mesuré : **1 441 morts de classe VEHICULE**. La divergence doit être **MESURÉE, PUBLIÉE et TENUE PAR UN GARDE-RAIL** (items A0.4 et A1.9). Second axe, sur le NOM : le feed affiche le libellé `damagetag`, le graphe celui de `metadata.weapons` — écart déjà documenté dans le dépôt (« Mk51 Sidekick » en base contre « Mk50 Sidekick » au registre). |
| **D12** | **OÙ SE FAIT LA TRADUCTION `source_tag` -> arme : en Go, jamais en SQL.** C'est la seule question que cette décision tranche, et il faut la lire telle quelle — une première rédaction opposait « vue SQL » et « code Go » comme s'il s'agissait de deux architectures concurrentes. C'est faux : une vue ne stocke rien (c'est une requête nommée), et l'app combine déjà les deux aujourd'hui. La seule différence réelle est celle-ci : **faire la traduction en SQL oblige à COPIER dans la base la table de correspondance (468 lignes) qui vit aujourd'hui dans le binaire**, donc à la maintenir synchronisée à chaque évolution du catalogue — lequel évolue à chaque saison (206 des 468 entrées sortent encore « Autres »). La faire en Go ne copie rien. Corollaire : les deux autres arguments avancés initialement — la disparition du nom `v_weapon_kills` et la fusion des chemins de chargement — sont INDÉPENDANTS de ce choix ; ils sont obtenus par D2 et D11 quelle que soit la réponse ici. |

---

## 1. État mesuré (base de production locale, 2026-09-01)

Mesures faites au lecteur SQL `cmd/diag_q`, en lecture seule.

### 1.1 Les deux chaînes en présence

| | ancienne chaîne | nouvelle chaîne |
|---|---|---|
| étape post-sync | 1.55 `runWeaponKills` | 1.57 `runKillSource` |
| méthode | corrèle les **tirs** de l'attaquant avec l'instant du kill | lit la **source du dégât** dans l'état de mort de la victime (`jpt!`) |
| table | `weapon_kills` → vue `v_weapon_kills` | `match_kill_events` → vue `match_kill_events_latest` |
| lecteurs | les 6 surfaces de « Répartition des frags », vue match, explorer, citations, home | le kill feed du rejeu 2D, et 3 classes hors arsenal du sunburst |

### 1.2 Couverture comparée, sur les **mêmes 1 361 matchs**

| | `weapon_kills` | source de dégât |
|---|---|---|
| lignes | 112 139 kills | 109 738 morts avec source mesurée |
| confiance « haute » | 69,9 % | — |
| confiance basse / nulle | 9,0 % / **16,4 %** | — |
| **non résolu à une arme** | **21 010 (18,7 %)** | **752 INCONNU + 12 hors table (0,7 %)** |
| classé et publiable | — | **99,3 %** |

Classe ARME seule (celle qui remplace les classes d'arme à feu du sunburst) :
**87 347 morts, dont 85 945 obtiennent une `weapon_key` — 98,4 %, sur 27 clés distinctes.**

### 1.3 L'écart arme par arme

| arme | via `weapon_kills` | via la source de dégât |
|---|---|---|
| *(non résolu)* | 21 010 | 0 |
| MA40 AR | 19 831 | 12 534 |
| BR75 | 16 517 | 13 873 |
| Sidekick | 14 793 | 12 100 |
| Marteau à gravité | **131** | **6 727** |
| Faisceau Sentinelle | **62** | **3 159** |
| Épée à énergie | **191** | **3 014** |

Lecture : l'ancienne chaîne lit des **tirs**. Une épée, un marteau, un faisceau n'en émettent
pas — leurs kills sont perdus ou recollés sur l'arme à feu tenue, ce qui gonfle l'AR, le BR et
le Sidekick.

### 1.4 Ce que la bascule NE fait PAS perdre

Matchs portant des lignes `weapon_kills` mais **aucune** source mesurée : **1** (sur 1 361).
Le film est le facteur limitant des DEUX chaînes ; il n'y a pas de population servie par
l'ancienne et pas par la nouvelle.

### 1.5 Halo 5 — le garde-fou de la décision D2

`data/titles/halo_5/warehouse/shared_matches_v2.duckdb` : **550 926 lignes** de `weapon_kills`
sur **2 754 matchs** (registre : 3 032). Producteur : `games/halo_5/ingest/kills.go`, arme
**native** de la timeline API. Halo 5 n'a **pas** de décodeur de film, donc pas de
`source_tag`. Supprimer la table pour tous les titres aveuglerait Halo 5.

---

## 2. Périmètre

### Dans le périmètre

- Halo Infinite : bascule de TOUS les lecteurs d'arme-par-kill vers la source de dégât.
- Suppression du producteur `weapon_kills` Halo Infinite et de la chaîne de corrélation qui ne
  sert plus qu'à lui.
- Suppression de la table et de la vue **dans le fichier `halo_infinite`**.
- Garde-rails interdisant la réapparition d'une lecture Halo Infinite.
- Volet B : garde-rail de câblage, journalisation et rattrapage de l'étape 1.58.

### Hors périmètre (à ne pas toucher)

- Halo 5 : producteur, table, vue, lecteurs, `kill_kind`, `steps_shared_h5_weapon_kill_kind`.
- `killer_victim_pairs` (264 843 lignes) — autre chantier.
- `match_weapon_shots` / `match_weapon_shots_latest` — les TIRS du rejeu, chaîne distincte.
- `weapon_accuracy` (0 ligne côté Infinite) et son port — noter en Découvertes, ne pas traiter.
- `kill_positions` — alimentée ailleurs.
- Toute re-cuisson d'artefacts de rejeu (D8).
- Le niveau 2 du sunburst (D10).

---

## 3. VOLET A — l'arme du kill

Branche : `wt/arme-source-unique`, worktree DÉDIÉ créé depuis `feat/v75` :

```bash
git worktree add ../LevelUp-wt-arme-source -b wt/arme-source-unique feat/v75
```

### Étape A0 — Le témoin de bascule (à faire AVANT toute modification)

Objectif : pouvoir dire, chiffres en main, ce que la bascule change — et non l'affirmer.

- [ ] A0.1 Écrire un test d'instrumentation (`//go:build` normal, mais **skippé sans variable
      d'environnement**, motif des sondes du dépôt) qui, pour un lot de matchs donné, produit
      côté à côté : total de kills API, ventilation par classe issue de `weapon_kills`,
      ventilation par classe issue de la source de dégât, et le résidu « Non attribué » des deux.
- [ ] A0.2 Le lancer sur **au moins 200 matchs** et consigner la sortie dans
      `.ai/V7.5/MESURE_BASCULE_ARME_2026-09-XX.md`.
- [ ] A0.3 Écrire dans ce même fichier, **avant** de lire les résultats, le seuil d'acceptation :
      *la bascule est acceptée si le résidu « Non attribué » agrégé DIMINUE et si aucune classe
      d'arme à feu ne perd plus de 2 % de ses kills sans explication nommée.*
- [ ] A0.4 **Mesure de concordance graphe / kill feed (D13).** Sur les tags de source réellement
      observés en base, produire trois nombres : (a) tags qui obtiennent une IMAGE via
      `killicon.Lookup`, (b) tags qui obtiennent une CLÉ via
      `KillSourceRegistry{}.KillSourceRegistryKey`, (c) tags qui obtiennent l'une sans l'autre —
      avec, pour chacun de ces derniers, son nombre de morts et sa classe `damagetag`. Consigner
      la table complète dans le fichier de mesure. C'est le chiffre qui dit combien de kills
      seraient nommés dans le rejeu et anonymes dans le graphe.
- [ ] A0.5 Même mesure sur le NOM (second axe de D13) : pour les tags qui obtiennent une clé,
      comparer le libellé `damagetag.Lookup(...).Name` (celui du kill feed) au libellé
      `metadata.weapons.name` de la clé (celui du graphe). Lister **toutes** les paires qui
      diffèrent, avec le nombre de morts concernées.

**Gate A0** : le fichier de mesure existe, il porte le seuil écrit AVANT les résultats, et la
commande de reproduction y figure telle quelle.

### Étape A1 — Le lecteur unique

Objectif : `port.WeaponKillRepository` sert Halo Infinite depuis `match_kill_events_latest`.

- [ ] A1.1 Nouveau repo `internal/platform/duckdb/killsource_weapon_kills_repo.go` implémentant
      `port.WeaponKillRepository` : lit `match_kill_events_latest`, filtre
      `source_tag IS NOT NULL AND feed_killer_xuid IN (…) AND match_id IN (…)`, traduit le tag
      par `port.KillSourceClassifier`, résout `Role`/`Class`/`Family`/`WeaponKey` par la même
      passe registre que `weapon_resolver.go`.
- [ ] A1.2 Exclusion structurelle des classes servies par les compteurs API (D4) : les clés dont
      la classe registre vaut `melee` ou `grenade` ne sont pas remontées. **Écrire le garde-rail**
      qui vérifie que l'ensemble exclu est exactement celui-là (modèle :
      `TestHorsArsenalHINFSansIdNumerique`).
- [ ] A1.3 **Fusion des deux chemins (D11)** : le nouveau repo rend AUSSI les classes hors
      arsenal (répulseur, bobines, environnement). Par conséquent, supprimer
      `internal/service/killsourceload/`, `internal/platform/duckdb/killsource_class_repo.go`,
      `internal/port/kill_source_class.go`, le paramètre `sources` de `fragdist.Build`, la
      fonction `buildKillSourceFragClasses` et les 7 sites d'appel de `killsourceload.Load`.
      Le filtre « clé sans identifiant numérique » et la règle anti-double-comptage disparaissent :
      il n'y a plus qu'une source, donc plus rien à départager.
- [ ] A1.4 Conséquence sur le niveau 2 : `equipment` et `environmental` doivent entrer dans
      l'ensemble des classes ventilées PAR OBJET (`domain.IsPerWeaponFragClass`) pour conserver
      le rendu actuel. Les classes d'arme à feu restent PAR RÔLE (D10).
- [ ] A1.5 **PIÈGE HALO 5, à traiter avant de coder A1.4** : `isRegistryFragClass` est du code
      PARTAGÉ. Élargir l'ensemble des classes servies par le registre ferait aussi remonter les
      lignes `h5_environmental` de Halo 5, qui portent un identifiant numérique et vivent dans
      `weapon_kills` — le sunburst de Halo 5 changerait. **Le sunburst de Halo 5 ne doit pas
      bouger dans ce lot.** Poser un test doré (golden) sur une base Halo 5 forgée, et le faire
      passer AVANT et APRÈS A1.4 avec une sortie identique octet pour octet.
- [ ] A1.6 Câblage par capability dans `internal/api/wire/registry_pages.go` : titre portant
      `film.kill_source` → nouveau repo ; sinon → `WeaponKillsRepo` existant. Jamais `slug ==`.
- [ ] A1.7 Dégradation : capability absente ou `match_kill_events_latest` absente →
      `games.ErrCapabilityNotSupported`, réponse partielle propre, jamais de panique.
- [ ] A1.8 Tests : repo sur DuckDB `:memory:` (lignes forgées), service avec mock du port,
      cas de dégradation.
- [ ] A1.9 **Garde-rail de concordance (D13)** : test qui parcourt `killicon.ResolvedTags()` et
      exige que l'ensemble des tags rendant une IMAGE et l'ensemble des tags rendant une CLÉ
      soient identiques, **à l'exception d'une allowlist explicite, datée et justifiée entrée par
      entrée**. Toute règle ajoutée plus tard sans clé fera rougir ce test — c'est le seul moyen
      d'empêcher la divergence de revenir en silence. Modèle : `no_art_patterns_test.go`.
      L'allowlist initiale contient les 19 entrées mesurées en A0.4 ; chacune porte la raison
      pour laquelle le graphe n'a pas de clé à leur donner.
- [ ] A1.10 Journaliser, au niveau du nouveau repo, le nombre de morts écartées faute de clé de
      registre, par classe `damagetag` (`slog.InfoContext`, compteur entier). Un kill qui tombe
      dans « Non attribué » alors que le rejeu sait le nommer ne doit pas disparaître en silence.

**Gate A1** :
```bash
cd apps/go-api && go test ./internal/platform/duckdb/... ./internal/service/... ./internal/api/...
go vet ./...
```
plus : le témoin A0 relancé montre la nouvelle ventilation sur les 6 surfaces.

### Étape A2 — Les autres lecteurs Halo Infinite

Liste **FERMÉE**. Aucun autre site n'est concerné ; si l'exécuteur en trouve un, il l'ajoute
ici avant de le traiter.

- [ ] A2.1 [queries_match.go:59](apps/go-api/internal/platform/duckdb/queries_match.go#L59) — armes du match
- [ ] A2.2 [queries_match.go:194](apps/go-api/internal/platform/duckdb/queries_match.go#L194) — armes par joueur
- [ ] A2.3 [queries_match_detail.go:229](apps/go-api/internal/platform/duckdb/queries_match_detail.go#L229) — détail par joueur
- [ ] A2.4 [explorer_repo.go:559](apps/go-api/internal/platform/duckdb/explorer_repo.go#L559) — top armes de la cible
- [ ] A2.5 [queries_home_citations.go:479](apps/go-api/internal/platform/duckdb/queries_home_citations.go#L479) — arme favorite (accueil)
- [ ] A2.6 [sync/citations.go:535](apps/go-api/internal/sync/citations.go#L535) — statistique `weapon_stat` du moteur de citations
- [ ] A2.7 [registry_weapon_coverage.go:38](apps/go-api/internal/api/wire/registry_weapon_coverage.go#L38) — page admin « couverture de résolution d'arme ». **Décision à prendre à l'étape** : soit rebasculer sur les tags `jpt!` non résolus, soit supprimer la page avec son type `domain/admin_weapon_coverage.go` et sa route. Trancher par écrit, ne pas laisser en l'état.
- [ ] A2.8 [scheduler/data_health_check.go:298](apps/go-api/internal/scheduler/data_health_check.go#L298) — « matchs sans `weapon_kills` » → « matchs sans source de dégât »
- [ ] A2.9 [api/handlers/health_home.go:146](apps/go-api/internal/api/handlers/health_home.go#L146) — message de diagnostic
- [ ] A2.10 [validation/gate.go:151](apps/go-api/internal/validation/gate.go#L151) et `:276` — la vue `v_weapon_kills` figure dans les vues V6 attendues. La rendre conditionnelle au titre, sinon le gate deviendra rouge à l'étape A4.

**Gate A2** :
```bash
cd apps/go-api && grep -rn "v_weapon_kills\|FROM weapon_kills\|JOIN weapon_kills" --include=*.go internal/ \
  | grep -v "_test.go" | grep -v "halo_5\|halo5\|/migration/"
```
doit ne rendre **que** les sites Halo 5 et les migrations. Aucune case A2 sans statut.

### Étape A3 — Mort du producteur

- [ ] A3.1 Supprimer `internal/sync/backfill_weapons.go` et ses tests
      (`backfill_weapons_test.go`, `_pipeline_test.go`, `_regression_test.go`).
- [ ] A3.2 Retirer l'appel `films.runWeaponKills(ctx, insertedIDs)` de
      [engine_postsync.go:278](apps/go-api/internal/sync/engine_postsync.go#L278) et
      `runWeaponKills` de `convergence.go`, plus la mesure `clock.lap` associée.
- [ ] A3.3 Retirer les compteurs `WeaponKillsProcessed` / `WeaponKillsNoFilm` de
      `domain/sync.go` et leurs lecteurs.
- [ ] A3.4 Bits de masque `MBitWeaponKills` (1<<21) et `MBitWeaponKillsNoFilm` (1<<22) :
      les retirer de `sync/matchflags/flags.go` et de `sync/backfill_flags.go`. **Ne pas
      réattribuer les numéros de bit** — un bit libéré reste libéré (les masques persistés
      portent l'ancienne valeur).
- [ ] A3.5 Supprimer la chaîne de corrélation devenue sans appelant :
      `internal/analysis/weapon_correlation.go`, `weapon_parser.go`, `weapon_scanner.go`,
      `weapon_reconciliation.go` et leurs tests — **après avoir vérifié sur pièces** qu'aucun
      autre paquet ne les importe.
- [ ] A3.6 `internal/analysis/weaponv3` : supprimer `correlate.go` et `fire_scanner_v3.go`,
      **CONSERVER** la table de noms (`WeaponName`, `KnownWeaponHigh32`) dont dépendent
      `replay/loadouts.go` et `replay/ground_weapon_rules.go` (D9). Si la table reste seule dans
      le paquet, la déplacer sous un nom qui dit ce qu'elle est.
- [ ] A3.7 Supprimer la table morte `weapon_kills_v3` (0 ligne) : `domain/weapon_kills_v3.go`,
      `platform/duckdb/weapon_kills_v3_repo.go`, `migration/steps_shared_weapon_kills_v3.go`,
      `cmd/diag_weapons_v3/`.
- [ ] A3.8 Sous-commandes CLI devenues sans objet : `cmd/merge_weapon_kills`,
      `cmd/fix_weapon_bitmask`, `cmd/verify_weapons`, `cmd/diag_weapons`, `cmd/diag_weapons_match`,
      `cmd/diag_weapons_sweep`, `cmd/diag_squad_weapons`, `cmd/diag_weapon_citations`,
      `cmd/frontb_coverage`, `cmd/probe_pi_reconcile`, et l'option `--weapons` de
      `cmd/levelup/cmd_backfill.go`. **Vérifier chacune** : celles qui servent encore Halo 5
      restent.
- [ ] A3.9 Réglage `spnkr_refresh_backfill_weapons` : le retirer de `settings/store.go`, de
      l'UI d'administration et de `app_settings.example.json`.

**Gate A3** :
```bash
cd apps/go-api && go build ./... && go vet ./... && go test ./...
```
plus `grep -rn "runWeaponKills\|MBitWeaponKills\|CorrelateKillsGlobal" --include=*.go .` → aucun
résultat hors historique Git.

### Étape A4 — Mort de la table côté Halo Infinite

- [ ] A4.1 Nouvelle étape de migration **title-owned**, dans
      `internal/games/halo_infinite/migrations/`, nom neuf et jamais réutilisé
      (ex. `shared_drop_weapon_kills_v1`) : `DROP VIEW IF EXISTS v_weapon_kills` puis
      `DROP TABLE IF EXISTS weapon_kills`, `DROP TABLE IF EXISTS weapon_kills_v3`,
      `DROP SEQUENCE IF EXISTS weapon_kills_generation_seq`, suivi d'un `CHECKPOINT`.
      **AUCUNE archive, aucune table de sauvegarde, aucune copie « au cas où » (D2).** Les
      112 139 lignes Halo Infinite partent en totalité ; l'utilisateur a ses sauvegardes de base
      et a explicitement refusé qu'on en garde une trace lisible dans le schéma.
- [ ] A4.1bis Vérification d'exécution : après migration sur la base locale,
      `SELECT COUNT(*) FROM duckdb_tables() WHERE table_name IN ('weapon_kills','weapon_kills_v3')`
      et `SELECT COUNT(*) FROM duckdb_views() WHERE view_name = 'v_weapon_kills'` rendent **0**
      sur `halo_infinite`, et rendent les valeurs d'origine sur `halo_5`.
- [ ] A4.2 L'inscrire dans `order.go` du paquet title-owned, **jamais** dans l'ordre partagé —
      la migration ne doit pas s'exécuter sur le fichier `halo_5`.
- [ ] A4.3 Test de migration qui vérifie, sur une base forgée : la table part côté
      `halo_infinite`, elle **survit** côté `halo_5`.
- [ ] A4.4 Retirer de l'ordre partagé les étapes qui ne servaient qu'à cette table côté
      Infinite (`steps_shared_append_only_weapon_kills`) **si et seulement si** Halo 5 ne s'en
      sert pas — vérifier sur pièces, sinon les laisser et l'écrire.
- [ ] A4.5 Garde-rail de non-retour : test `archlint` interdisant, dans tout fichier hors
      `games/halo_5/`, `platform/duckdb/halo5/` et `internal/migration/`, le littéral
      `weapon_kills` (avec allowlist explicite et datée, comme `no_art_patterns_test.go`).
- [ ] A4.6 Adapter `internal/sync/testutil/fixture.go` (il crée la vue) et les golden tests.

**Gate A4** :
```bash
cd apps/go-api && go test ./internal/migration/... ./internal/games/halo_infinite/migrations/... ./internal/archlint/...
go test -tags=integration -p 1 ./...
```
Le second est **obligatoire** : ce lot touche `migration/` et `persist/`.

### Étape A5 — Livraison

- [ ] A5.1 Sauvegarde de `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb` **avant**
      première exécution de la migration en local.
- [ ] A5.2 Lancer la migration en local, relancer le témoin A0, consigner le résultat.
- [ ] A5.3 `make gate-push`, puis CI verte au niveau JOB sur `feat/v75`.
- [ ] A5.4 Entrée dans `.ai/thought_log.md`.
- [ ] A5.5 Vérification visuelle par l'utilisateur : les 6 surfaces de « Répartition des frags »
      + vue match + explorer. **C'est lui qui prononce le gate visuel**, pas l'exécuteur.
- [ ] A5.6 Note de déploiement : la migration s'exécutera au démarrage du serveur de production.
      Prévenir l'utilisateur **avant** le push sur `main` (push = déploiement automatique).

---

## 4. VOLET B — le rejeu 2D au fil de l'eau

Branche : `wt/rejeu-fil-de-l-eau`, worktree dédié depuis `feat/v75`.

Constat qui fonde ce volet : sur les 222 matchs des 90 derniers jours, **1 seul** a un artefact ;
les 50 artefacts locaux sont tous d'anciens matchs cuits à la main. Le 2026-08-27 à 23h02 un
cycle a inséré 7 matchs, l'étape 1.55 les a traités la minute même avec le film disponible, et
**aucune ligne « rejeu 2D » n'existe dans le journal**.

### Étape B1 — Rendre l'étape observable

- [ ] B1.1 [convergence.go:674](apps/go-api/internal/sync/convergence.go#L674) : l'assertion
      `fetcher, _ := s.client.(replayartifacts.ChunksFetcher)` **jette son résultat**. La
      journaliser en WARN avec un compteur, exactement comme le fait `runKillSource` juste
      au-dessus.
- [ ] B1.2 Garde-rail de câblage à la COMPILATION, calqué sur
      `internal/sync/kill_source_wiring_test.go` :
      `var _ replayartifacts.ChunksFetcher = (*PooledHaloClient)(nil)` et les deux autres
      clients réels. C'est le test qui manque, et son absence a déjà coûté un lot.
- [ ] B1.3 Les trois sorties silencieuses de `replayartifacts.Run` (placement éteint, client
      sans chunks, rien construit) émettent chacune une ligne au niveau adéquat, **une fois par
      cycle**, jamais une par match.
- [ ] B1.4 Compteurs expvar : sélectionnés, construits, sautés-déjà-à-jour, films persistés,
      **retard restant**.

**Gate B1** : `go test ./internal/sync/...` ; puis un cycle de sync local produit au moins une
ligne « rejeu 2D » quel que soit le résultat.

### Étape B2 — Le rattrapage

L'étape 1.58 ne voit que `insertedIDs` : **une seule tentative, à l'instant de l'insertion**,
alors que le film Theater se publie plus tard. L'étape 1.57, elle, a un rattrapage.

- [ ] B2.1 Ajouter à `replayartifacts` une sélection de retard calquée sur
      `killcollector.backlogAJour` + `ordonnancer` : matchs de la fenêtre de rétention, sans
      artefact à jour, du plus récent au plus ancien.
- [ ] B2.2 Bornes **non négociables** : `maxPerCycle` inchangé (5), plus un **budget de durée**
      par cycle sur le modèle de `killcollector.PostSyncBudget`. Le solde repart au cycle suivant.
- [ ] B2.3 Publier le retard restant en expvar (jauge, jamais un ratio).
- [ ] B2.4 Tests d'intégration : nominal, retard non vide, budget épuisé — modèle
      `postsync_backlog_integration_test.go`.

**Gate B2** : `go test -tags=integration -p 1 ./internal/sync/...`, et la jauge de retard
décroît sur deux cycles consécutifs en local.

### Étape B3 — Le réglage, et rien de plus

- [ ] B3.1 Documenter dans `docs/COMMANDS.md` (FR + EN) les trois valeurs de
      `replay_build_location` et ce que chacune implique.
- [ ] B3.2 Vérifier le réglage effectif de la production : `local` y est refusé, le défaut est
      `worker`, qui exige `LEVELUP_BUILD_WORKER_TOKEN` **et** un `cmd/replay-worker` qui vide la
      file. Rendre compte à l'utilisateur ; **ne pas** modifier la production sans son accord.
- [ ] B3.3 Entrée `.ai/thought_log.md`.

**Aucune construction d'artefact dans ce volet (D8).**

---

### Étape A6 — Combler le trou véhicules et tourelles (D14)

Cette étape vient APRÈS A4. Elle ne touche à aucun décodage : c'est une addition au registre
d'armes, qui fait passer 1 441 morts de « Non attribué » à la classe qui leur revient et met
le graphe d'accord avec le kill feed.

**D14 — LES 14 ENTRÉES, LIBELLÉS ARRÊTÉS PAR L'UTILISATEUR LE 2026-09-01.** Règle qu'il a
donnée : les noms de véhicules gardent l'anglais, à trois exceptions près. Les tourelles sont
des descriptions et non des noms propres : elles sont traduites (règle « UI FR sans
anglicismes »).

| clé | classe | banque sonore (règle BANQUE) | `name_en` | `name_fr` |
|---|---|---|---|---|
| `hinf_ghost` | vehicle | `veh_cv_ghost` | Ghost | Ghost |
| `hinf_banshee` | vehicle | `veh_cv_banshee` | Banshee | Banshee |
| `hinf_wraith` | vehicle | `veh_cv_wraith` | Wraith | **Apparition** |
| `hinf_phantom` | vehicle | `veh_cv_phantom` | Phantom | Phantom |
| `hinf_chopper` | vehicle | `veh_bt_chopper` | Chopper | Chopper |
| `hinf_wasp` | vehicle | `veh_un_wasp` | Wasp | Wasp |
| `hinf_scorpion` | vehicle | `veh_un_scorpion` | Scorpion | Scorpion |
| `hinf_rockethog` | vehicle | `veh_un_rockethog` | Rockethog | **Warthog lance-roquettes** |
| `hinf_pelican` | vehicle | `veh_un_pelican` | Pelican | **Pélican** |
| `hinf_falcon_lmg` | turret | `veh_un_falconlmgturret` | Falcon LMG turret | Tourelle LMG du Falcon |
| `hinf_falcon_gl` | turret | `veh_un_falcongrenadelauncher` | Falcon grenade launcher | Lance-grenades du Falcon |
| `hinf_turret_machinegun` | turret | `tur_un_machinegun` | Machine gun turret | Tourelle mitrailleuse |
| `hinf_turret_plasma` | turret | `tur_cv_plasmacannon` | Plasma cannon | Canon à plasma |
| `hinf_turret_shade` | turret | `tur_cv_shadeturret` | Shade turret | Tourelle Shade |

- [ ] A6.1 Seed `metadata.weapons` : les 14 lignes, `title_slug = 'halo_infinite'`,
      `class` de la table, `role` = la classe, `family_key` = la classe. **Recette à copier
      telle quelle sur les 6 entrées hors arsenal existantes** (`hinf_repulsor`,
      `hinf_coil_*`, `hinf_environment`), qui ont été faites exactement ainsi.
- [ ] A6.2 Seed `metadata.weapon_name_labels` : les 14 paires `name_en` / `name_fr` de la
      table ci-dessus, verbatim. Les trois traductions sont **arrêtées par l'utilisateur** —
      ne pas les réinterpréter.
- [ ] A6.3 Renseigner la colonne `weapon_key` des 14 règles de genre BANQUE dans
      `internal/games/halo_infinite/film/killicon/data/rules.tsv`.
- [ ] A6.4 **AUCUN identifiant numérique** pour ces 14 clés (pas de ligne `weapon_ids`) : elles
      n'existent pas dans `weapon_kills` et n'ont pas à y exister. Vérifier que le garde-rail du
      paquet `weapons` (`TestHorsArsenalHINFSansIdNumerique`) est mis à jour en conséquence,
      avec sa justification datée.
- [ ] A6.5 Rétrécir l'allowlist du garde-rail de concordance (A1.9) des 14 entrées désormais
      résolues. Il doit rester exactement 5 exceptions : `Infected Energy Sword`,
      `Mythic Sandwich`, `Mutilator`, l'entrée GGGL n°3, le repli CLASSE `MELEE`.
- [ ] A6.6 Re-lancer le témoin A0 : les 1 441 morts de classe VEHICULE doivent quitter
      « Non attribué » pour les classes véhicule et tourelle. Consigner l'avant / après.
- [ ] A6.7 Vérifier que le sunburst de **Halo 5 ne bouge pas** (même test doré qu'en A1.5).
- [ ] A6.8 **RECLASSER L'ÉPÉE ET LE MARTEAU — décision utilisateur du 2026-09-01.** Ces deux
      armes sont aujourd'hui `class = 'melee'` au registre, si bien que le nouveau lecteur les
      écarte (D4 : le total de la classe mêlée vient du compteur API). Or **le compteur API ne
      les compte PAS** : mesure sur 200 matchs, `melee_kills` = 1 717 quand l'épée et le marteau
      pèsent 2 514 à eux deux — corpus entier : **marteau 6 727, épée 3 014, soit 9 741 frags**
      qui tombent en « Non attribué » sans que personne ne les serve. Mot de l'utilisateur :
      « bien sûr que les épées et marteaux ça ne compte pas dans les stats de mêlée ». Passer
      `hinf_energy_sword` et `hinf_gravity_hammer` à `class = 'heavy'`, `role = 'power'` (la
      classe existe déjà avec 5 armes) — ce sont des armes lourdes, pas une mécanique.
      **Aucun double comptage possible** : le compteur API de mêlée les ignore, c'est mesuré.
- [ ] A6.9 Après A6.8, re-lancer le témoin : les ~9 700 frags d'épée et de marteau doivent
      quitter « Non attribué » pour la classe Arme lourde. Consigner l'avant / après.

**Gate A6** :
```bash
cd apps/go-api && go test ./internal/games/... ./internal/service/fragdist/... ./internal/platform/duckdb/...
```
plus le témoin A0 relancé, avec le nombre de morts déplacées.

---

## 4bis. VOLET C — les qualificatifs du kill (assassinat, collage)

**À LANCER APRÈS LE VOLET A**, jamais en parallèle : il touche les mêmes lecteurs de vue
match, et il doit se construire sur le nouveau lecteur, pas sur l'ancien.
Branche : `wt/qualificatifs-kill`, worktree dédié depuis la branche du volet A une fois
fusionnée.

### D15 — Ce que `source_category` est, et ce qu'elle n'est PAS

`match_kill_events_latest.source_category` est un **QUALIFICATIF DU KILL**, au même titre
que le tir à la tête : une précision sur *comment* la mort est arrivée. Ce n'est **PAS** un
axe de classement. Elle **n'entre pas** dans le sunburst : celui-ci a deux niveaux (classe,
puis rôle ou objet), un troisième n'est pas géré et ce lot ne l'introduit pas
(cadrage utilisateur du 2026-09-01). Les qualificatifs se publient comme des **compteurs**,
sur la même étagère que `headshot_kills`.

### D16 — Le décodage de cette colonne est VALIDÉ par un oracle indépendant

Contrôle déclaré avant mesure : les tirs à la tête de l'API Halo (`match_participants.
headshot_kills`) servent d'oracle ; le film doit les reproduire s'il décode correctement.
Population isolée pour séparer l'erreur de décodage du trou de couverture : les couples
(match, joueur) où le film a vu **tous** les kills du joueur.

| | |
|---|---|
| couples comparés | 8 626 |
| accord **exact** | **8 621 (99,94 %)** |
| total API | 23 289 |
| total film | 23 292 |

Trois kills d'écart sur 23 000. Les valeurs `SilentMelee` et `AttachedDamage`, portées par
la MÊME colonne, héritent de cette crédibilité — et l'API ne les fournit pas.

**Découverte du contrôle, à instruire :** `HeadshotMultiplier` (2 250 kills) n'est PAS un
tir à la tête au sens du jeu. L'inclure fait chuter l'accord de 99,94 % à 83,5 %. C'est une
mécanique distincte, non identifiée. Ne pas l'agréger aux tirs à la tête.

### Étapes

- [ ] C1 Compteurs par joueur et par périmètre, lus depuis `match_kill_events_latest` :
      **assassinats** (`source_category = 'SilentMelee'`, 1 774 mesurés) et **kills par
      collage** (`AttachedDamage`, 3 405 mesurés — grenade collante ET supercombinaison du
      Needler, le sens est LARGE). Réutiliser le lecteur unique du volet A, ne pas ouvrir un
      second chemin.
- [ ] C2 Publier ces compteurs à côté du tir à la tête existant — mêmes surfaces, même
      forme. Aucune nouvelle page.
- [ ] C3 Libellés FR **et** EN dans `i18n.ts`, parité typée. FR sans anglicismes.
- [ ] C4 Le contrôle de D16 devient un test : sur une base forgée, le compteur de tirs à la
      tête issu du film doit égaler celui de l'API. C'est le garde-rail qui protège les deux
      autres compteurs, qu'aucun oracle ne couvre.
- [ ] C5 Dégradation : match sans film → compteurs absents, jamais zéro. Un zéro affiché
      dirait « mesuré à zéro » là où l'on n'a rien mesuré.

**Hors périmètre du volet C** : le troisième niveau du sunburst, l'identification de
`HeadshotMultiplier`, et toute ventilation de la mêlée par arme (impossible, cf. D4bis).

## 5. Pièges — à relire avant de coder

1. **`power_weapon_kills` n'est PAS `weapon_kills`.** C'est une colonne de
   `match_participants`, native de l'API, lue par l'explorer, la synthèse et les séries
   temporelles. Elle ne bouge pas. Un `grep weapon_kills` naïf remonte 190 fichiers dont la
   quasi-totalité concerne cette colonne.
2. **Halo 5 partage le schéma, pas le fichier.** Chaque titre a son
   `shared_matches_v2.duckdb`. Une étape de migration partagée s'exécute sur les DEUX ; une
   étape *title-owned* sur un seul. D2 dépend entièrement de ce point.
3. **`DuckDB` fige les `SELECT *` des vues à leur création.** `v_weapon_kills` est un
   `SELECT *, COALESCE(...)`. Toute vue qui la lirait devrait être recréée — vérifier
   `duckdb_views()` avant le DROP.
4. **`match_kill_events` a DEUX producteurs** et la vue `_latest` retient **la dernière passe
   par match, entièrement**. Un producteur crédit-seul qui repasserait après le décodeur de film
   effacerait la source de la lecture. Contrainte d'ordonnancement, pas de schéma.
5. **Le film est le facteur limitant des deux chaînes**, pas la bascule : 1 361 matchs contre
   1 362. Ne pas présenter comme une perte ce qui l'était déjà.
6. **`weaponv3` n'est pas mort** (D9) — le rejeu 2D en dépend pour les noms d'arme.
7. Écritures per-match : `persist.BatchBuilder` uniquement, jamais d'UPSERT concurrent.
   Lectures append-only : vues `_latest` uniquement.
8. `npm run typecheck` (`tsc -b`) fait foi côté web ; purger `node_modules/.tmp` avant.
   `go test -p 1` obligatoire pour les tests d'intégration DuckDB.

---

## 5bis. LE PLANCHER DU RÉSIDU EST LE BTB — mesuré le 2026-09-01

À lire avant d'interpréter le « Non attribué » du témoin A0 : il ne descendra pas à zéro, et
la raison est nommée, mesurée, et **hors du périmètre de ce plan**.

**Le film voit 99,6 % des frags** (137 159 morts vues pour 137 650 frags de l'API sur les
matchs porteurs d'une passe). Ce qui manque n'est PAS la mort, c'est la LECTURE DE LA SOURCE.

**Et ce manque a une cause unique : le nombre de joueurs du match.**

| profil du match | matchs | joueurs (moyenne) | morts (moyenne) | source lue |
|---|---:|---:|---:|---:|
| toutes les morts sourcées | 822 | **8,6** | 88,7 | **100 %** |
| partiel | 540 | **14,1** | 118,7 | 69,1 % |
| aucune source | 3 | **20,3** | 144,7 | 0 % |

Corrélation monotone et parfaite. **En 4v4 le décodeur est à 100 %** ; il se dégrade à mesure
que le nombre d'entités à suivre monte, et s'effondre en BTB. Contrôles écartés : ce n'est pas
une révision de décodeur périmée (les deux groupes portent `killsource-2026-07-31`), ce ne sont
pas les suicides ni les chutes (**zéro** mort sans tueur crédité dans la table, 251 tueurs bots
soit 0,2 %).

Sur les joueurs suivis, la couverture réelle est donc : JGtm **96,8 %**, Chocoboflor **96,4 %**,
Madina97294 **95,0 %**, XxDaemonGamerxX 85,3 %.

CONSÉQUENCE POUR LE TÉMOIN : après A6 et A6.8, un résidu de l'ordre de **4 à 6 %** sur des
matchs 4v4 est le PLANCHER attendu, pas un défaut du lot. Un résidu plus élevé sur un
échantillon signifie que l'échantillon contient du BTB — le vérifier avant d'incriminer la
bascule. Le BTB n'est pas supporté aujourd'hui (décision utilisateur du 2026-09-01) : faire
tenir le décodeur sur les gros matchs est un chantier à part, chiffré à ~9 % de frags.

## 6. Découvertes hors périmètre

À remplir par l'exécuteur. **Consigner, ne pas traiter.**

- `weapon_accuracy` : 0 ligne côté Halo Infinite, port et repo vivants — statut à instruire.
- `kill_positions` : 0 ligne côté Halo Infinite (alimentée par le chantier rejeu).
- `killer_victim_pairs` : 264 843 lignes, doublon fonctionnel de `match_kill_events` selon
  l'en-tête de `steps_shared_kill_events.go` — retrait à instruire séparément.

---

## 7. Reprise de session

- Avancement : les cases de ce fichier. `[x]` fait, `[~]` couvert ailleurs (avec référence),
  `[!]` non traité (avec justification écrite). **Aucune case vide à la clôture d'une étape.**
- Ordre : strict. L'étape N+1 ne commence pas tant que le gate de N n'est pas passé.
- Branche : `git log --oneline -10` sur le worktree du volet concerné.
- Contrat : skill `plan-execution`. Avant livraison : skill `delivery-checklist`.
- Reports : tout report entre au `.ai/V7.5/REGISTRE_REPORTS.md` avec sa condition de reprise.
