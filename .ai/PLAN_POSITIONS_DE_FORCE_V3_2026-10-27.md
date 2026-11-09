# PLAN — Positions de force v3 (reprise de la recherche)

> **Statut : PROJET, DÉCISIONS TRANCHÉES LE 2026-10-27 (D10-D13), AUCUNE EXÉCUTION.** Écrit le 2026-10-27 à la demande de
> l'utilisateur (« on reste au niveau du plan et de l'audit, on peut compléter un max mais pas
> d'implem à ce stade »). Rien de ce plan ne s'exécute avant un « go » daté de l'utilisateur ;
> un « ok plan » n'est pas un go.
>
> Socle : `.ai/AUDIT_POSITIONS_DE_FORCE_2026-10-27.md` (état des données, échelle tranchée,
> décisions du 27/10). Précédent : `PLAN_POSITIONS_DE_FORCE_2026-09-20.md` sur la branche
> `wt/power-positions` (quatre verdicts NO-GO) et son document de définition
> (`.ai/V7.5/positions_de_force/DEFINITION_POSITION_DE_FORCE_2026-09-20.md`, même branche).
>
> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la fois, aucun report
> d'une étape exécutable, chaque item statué `[x]` / `[~]` réf / `[!]` justifié, zéro fix hors
> périmètre, vérification sur pièces avant de cocher).

## Objectif et critère de succès

Ce plan est un plan de RECHERCHE. Il produit un verdict, pas une fonctionnalité.

**Objectif** : dire, carte par carte et par famille de mode, s'il existe des positions de force
mesurables dans les données, avec une méthode qui corrige les trois défauts identifiés en
septembre (échelle fausse, confusion par le niveau des joueurs, corpus de bons joueurs absent).

**Critère de succès (écrit avant toute mesure, inchangeable après l'étape 5)** :

1. Sur au moins **4 cartes de validation** (cartes non utilisées pour régler la méthode), le
   classement retrouve les positions fortes de l'oracle pro v2 avec **rappel ≥ 0,70 et
   précision ≥ 0,60**, aux règles d'appariement du verdict v2 (`VERDICT_V2_2026-09-20.md`,
   « Règles appliquées »), et **zéro piège PUR** coloré sur ces cartes.
2. Le **témoin géographique** (chaque position attendue remplacée par la zone la plus
   éloignée) fait chuter rappel et précision d'au moins 0,25 en moyenne : sinon l'appariement
   ne mesure que la densité des zones.
3. La carte des positions est **reproductible** : deux moitiés de matchs tirées au sort
   (graine écrite) donnent des classements corrélés (Spearman ≥ 0,6 sur les candidats retenus).
4. Le résultat **tient sans Nuzzles** : rappel sur les cartes de validation avec ses matchs
   retirés ≥ rappel complet − 0,15.

Un échec n'est pas un réglage à reprendre : c'est un NO-GO écrit, avec diagnostic par zone.

**Hors périmètre** : la production (catalogue de référence, API, calques rejeu / vue de match /
Tactique). Elle fera l'objet d'un plan distinct, sur GO, qui reprendra les étapes 3 à 7 du plan
du 20/09 avec les unités corrigées.

## Ce qui est acquis (vérifié sur pièces le 2026-10-27)

| Acquis | Où | Chiffre |
|---|---|---|
| Corpus de positions de kills | `LevelUp-recherche-positions/data/.../shared_matches_v2.duckdb` (copie de la sauvegarde d'avant purge) | 375 096 kills, 3 399 matchs, 2023-04 → 2026-10-25 |
| Niveau de tout le lobby | `match_csrs` (dernière écriture par match et joueur) | 1 798 matchs à positions avec CSR ; 782 lobbys de moyenne ≥ 1500 |
| Attente de performance par joueur | `match_participants.kills_expected / deaths_expected` | ~72 % des participants |
| Réservoir classé non décodé | `match_registry` classés sans position de kill | 2 991 matchs classés (dont 1 474 de 2021-2023 et 583 lobbys Onyx), Live Fire 871 classés dont 288 décodés ; dans le périmètre D11 (≥ 2023) : 2 376, dont 859 de 2023 |
| Échelle | audit §2 | coordonnées en unités monde, 1 unité = 3,048 m |
| Décalage Live Fire - Ranked | audit §1.5 | résolu, variantes classées utilisables |
| Disponibilité des films anciens | VPS (503 films téléchargés les 26-27/10, matchs 2024-10 → 2026-10) ; sonde `cmd/diag_film_avail` (43/43 servis, tirages 2026 seulement) | au-delà de 2024 : inconnu |
| Couverture du décodeur par build | `film/internal/profile/profile_table.go` | `HI_1_4_1` mesuré sur UN film ; majeure 31 sur un film ; `HI_1_8_0` → `HI_1_13_0` mieux couverts |
| Oracle pro v2 | branche `wt/power-positions`, `ORACLE_PRO_V2_2026-09-20.md` §6 | 28 fortes sur 8 cartes |
| Outillage de septembre | branche `wt/power-positions` : `cmd/mappower-build`, `cmd/mapgeo-build`, `internal/analysis/powerpos{,/geo,/fusion}`, harnais de verdict (tag `research`) | base `feat/v75` au 2026-09-20, non compilé contre la base actuelle |

Joueurs à élargir (décision user du 27/10) :

| Joueur | Matchs dans la sauvegarde | CSR médian / max | Ce qui manque |
|---|---|---|---|
| calloussss | 325 (15 en 2025, 0 avant) | 1 686 / 1 805 | tout l'historique avant 2025 ; tous ses films (purgés le 26/10) |
| Brooks SVP | 425 (10 en 2025, 0 avant) | 1 464 / 1 548 | idem |
| Nuzzles | 7 274 | 1 502 / 1 756 | ses films (0 en local, 11 au VPS) ; ~5 500 de ses matchs n'ont jamais eu de positions |

## Décisions tranchées

**D1 — Données isolées.** Toute lecture et toute écriture de la recherche se font dans
`C:\Users\Guillaume\Downloads\Scripts\LevelUp-recherche-positions\` (arborescence `data/`
miroir). Le `data/` de la copie principale, le serveur local et le VPS ne sont jamais écrits ni
ouverts en écriture par ce chantier. Le VPS n'est pas une source : aucune copie de sa base.

**D2 — Unités monde partout.** Toute constante de distance de la recherche est exprimée en
unités monde, nommée avec le suffixe `U` (et non `M`), et sa valeur en mètres figure en
commentaire avec le facteur 3,048 d'une seule constante nommée. Les réglages de septembre
(`ReglageV1`, `ReglageV2`, `ReglageGeoV1`, `ReglageFusionV1`) restent des preuves datées,
jamais retouchées ; la v3 a ses propres réglages.

**D3 — Le niveau des joueurs est une covariable, pas un filtre.** Le signal de duel se modélise
par engagement (une mort = un duel gagné à la position du tueur, perdu à celle de la victime),
avec l'écart de niveau des deux joueurs (CSR du match ; à défaut `kills_expected` /
`deaths_expected`) comme terme connu. L'avantage propre à un emplacement est ce qui reste.
Le filtre « lobbys Onyx » sert de contrôle, pas de méthode.

**D4 — La largeur de zone se choisit par reproductibilité.** Noyau gaussien de rayon `h`
(en unités) ; `h` est la plus petite valeur pour laquelle deux moitiés de matchs tirées au
sort donnent la même carte (critère 3). Aucune largeur choisie à l'œil.

**D5 — Trois signaux, un classement relatif.** (a) avantage de duel corrigé du niveau (D3) ;
(b) densité d'occupation (temps passé) des joueurs de haut niveau, depuis les trajectoires ;
(c) géométrie corrigée des unités (hauteur, vision, abri, accès, échappatoire). Le classement
est RELATIF à la carte et au mode (précision user du 20/09) : on retient les lieux qui se
détachent par un écart, sans plancher absolu ni plafond du nombre de positions.

**D6 — Familles de mode séparées.** Assassin, objectif fixe (Bastions, Roi de la colline),
objectif mobile (Drapeau, Oddball). Une carte n'est jugée que dans une famille où son corpus
suffit (seuil écrit à l'étape 4, avant la mesure).

**D7 — Calibrage / validation.** Comme en v2 : réglages choisis sur Recharge, Aquarius et
Streets ; jugés sur les autres cartes à oracle. L'oracle reste le juge, jamais une entrée du
score.

**D8 — Cuisson et téléchargement par paquets, sur accord.** Aucun téléchargement de masse ni
aucune cuisson de rejeux en lot sans accord daté de l'utilisateur pour CE paquet (règle
mémoire : pic mémoire). Chaque paquet annonce avant de partir : nombre de films, disque prévu,
pic mémoire mesuré sur le paquet précédent.

**D9 — Code de recherche, pas de production.** Paquets purs sous `internal/analysis/powerpos`,
outils sous `cmd/`, harnais de verdict sous tag `research`. Aucun handler, aucun service,
aucune route, aucun écran. Chemins par `PathResolver` construit sur la racine du dossier de
recherche. Les outils se branchent sur la capability `film.kill_positions`, jamais sur le slug.
Journalisation `slog` structurée : chaque lecture et chaque passe rapportent lignes lues,
écartées (avec motif) et écrites ; aucune erreur avalée. Règles de taille du dépôt (fichier
≤ 500 L, fonction ≤ 80 L) tenues comme pour le code de production.

## Décisions de l'utilisateur du 2026-10-27 (ex-questions ouvertes O1-O4)

- **D10 (O1) — Instance de recherche isolée (option A de § Acquisition)**, avec les jetons du
  dépôt principal : « on peut utiliser les jetons du repo principal pas de soucis. la copie
  isolée c'est bien ». Mise en œuvre : jonction vers `data/auth/watcher_tokens/` du dépôt
  principal, jamais de copie du fichier (rotation) ; acquisition serveur local arrêté.
- **D11 (O2) — Matchs antérieurs à 2023 ignorés** : « On ignore les matchs d'avant 2023 ».
  Périmètre temporel = matchs du 2023-01-01 inclus à aujourd'hui. Les films de 2023 restent
  soumis à la sonde de décodabilité par build (2.2).
- **D12 (O3) — Plafond disque 100 Go** pour films et rejeux de la recherche cumulés ; à
  l'atteinte, arrêt et nouvelle demande : « Ok pour 100go et me redemander ».
- **D13 (O4) — Branche `research/positions-de-force-v3`** depuis `release/v8.1.0`, worktree
  `LevelUp-wt-positions-de-force` : « ok ».

Ces réponses tranchent les questions du plan ; elles ne valent pas « go » d'exécution.

## Acquisition

Inventaire de l'outillage existant fait le 2026-10-27 (lecture de code, rien lancé). Références
relatives à `apps/go-api/`.

### Ce que l'outillage permet déjà

| Besoin | Outil existant | Contrainte vérifiée |
|---|---|---|
| Rediriger toutes les données | variable `LEVELUP_REPO_ROOT` (`internal/config/config.go:217`, `internal/domain/title/repo_root.go:42`) ; aucun `--data-root` global | le dossier visé doit contenir `config/` (capabilities), `data/titles/halo_infinite/reference/` (`map_quant_bounds.json`), `warehouse/metadata.duckdb`, `.env.local`, `db_profiles.json`, `data/auth/watcher_tokens/` |
| Historique complet d'un joueur | `levelup sync-full` (pagination `GetMatchHistory` par 25, `internal/sync/haloclient/halo_client.go:233`) | exige le joueur dans `db_profiles.json` (`cmd/levelup/cmd_sync.go:285-296`) ; crée sa base joueur sous le dossier visé |
| Registre, participants avec attentes, CSR de tout le lobby | même sync : `GetMatchStats` + `GetMatchSkill` + `ExtractAllSharedCSRRows` (`internal/sync/engine_fetch.go:82-152`) | `map_name` vient de `metadata.duckdb` |
| Télécharger les films | `levelup archive-films` (`cmd_archive_films.go`) ; candidats = `match_registry` | le match doit être au registre ; manifeste par jeton Halo (mutualisé), blobs CDN sans jeton |
| Positions de kills | `levelup backfill-killsource` (hors ligne) | passe légère : pic du pire film 422 Mio, 3 ouvriers, plafond 4 Gio ; exige carte résolue + bornes, roster, films complets ; base partagée tenue en écriture |
| Trajectoires par joueur | `levelup backfill-replay` (`--match`, `--since`, `--limit`, `--mem-limit-gib`) → `data/cache/replays/halo_infinite/{short8}.json` | lourd : ~800 fois la taille du film en mémoire, plafond 3 Gio par défaut, un film à la fois (verrou `film_decode.lock`) ; `match_player_positions` ne sert pas (grain 20 s, sans joueur) |
| Disponibilité d'un film | `cmd/diag_film_avail` (manifeste seul) | 43/43 sur 2026 ; ~29 % des matchs n'ont plus de film côté serveur (commentaire de `halo_client_film.go:370`, mesure non datée) ; un film expiré est perdu |
| Jetons | magasin `data/auth/watcher_tokens/{xuid}.json`, pool `PolicyAnyPublic` | la découverte ne parcourt que les joueurs de `db_profiles.json` ; un jeton sans consentement de mise en commun est exclu (ADR 0038) |

Aucune CLI ne liste puis n'insère l'historique d'un joueur non suivi sans profil.

### O1 — Options

- **A. Instance de recherche (RECOMMANDÉE).** Le dossier de recherche devient une instance
  LevelUp complète et isolée : `LEVELUP_REPO_ROOT` pointé dessus, son propre `db_profiles.json`
  avec Nuzzles, calloussss, Brooks SVP (et JGtm, porteur du jeton). On n'utilise QUE des CLI
  existantes et testées (`sync-full`, `archive-films`, `backfill-killsource`, `backfill-replay`),
  aucun code d'acquisition neuf. La règle « aucun sync sans profil suivi » (ADR 0035) porte sur
  l'instance de l'appli : ici, les profils sont déclarés, dans une instance qui n'est servie à
  personne. Les bases joueur créées restent dans le dossier de recherche.
- **B. Outil d'acquisition dédié.** Un `cmd/` de recherche qui appelle `GetMatchHistory`,
  `GetMatchStats`, `GetMatchSkill` et n'écrit que la base partagée de recherche. Évite les
  bases joueur, mais ré-câble un chemin d'écriture et d'API non testé (contraire à la règle
  « CLI testées d'abord »).

### Points durs de l'option A (à régler à l'étape 0, avant tout appel API)

1. **Jetons : jamais de copie du fichier.** Un jeton de rafraîchissement tourne à chaque
   usage ; une copie qui rafraîchit fait perdre la rotation à l'autre copie (CLAUDE.md, règle
   auth). Le dossier `data/auth/watcher_tokens/` de l'instance de recherche est une JONCTION
   vers le magasin principal. Le magasin ne protège la concurrence qu'au sein d'un processus
   (`internal/platform/auth/multi_user_token_store.go:113`, mutex + renommage atomique) : toute
   acquisition se fait **serveur local arrêté**.
2. **Consentement de mise en commun** : le jeton utilisé doit porter l'accord (ADR 0038), sinon
   le pool rend `ErrNoHealthySlot`. Vérifier, ne pas créer d'accord sans décision.
3. **Un seul décodage de film sur la machine** : le verrou `film_decode.lock` vit sous le
   `data/cache` de chaque instance ; deux instances ne se voient pas. Aucune cuisson de
   recherche pendant une cuisson de la copie principale.
4. **Jonctions et nettoyage** : jamais `git worktree remove --force` ni suppression récursive
   sur un arbre qui contient une jonction (incident du 2026-09-16) ; le dossier de recherche
   n'est pas un worktree.
5. **Référentiels** : `config/` et `reference/` copiés depuis la branche de travail (pas en
   jonction : la recherche ne doit pas changer sous l'effet d'un commit), `metadata.duckdb`
   copié depuis une sauvegarde au repos (pas depuis le fichier tenu par le serveur).

## Étapes

Ordre strict. Une étape est CLOSE quand tous ses items sont statués et son gate passé, gate
relu sur pièces. Effort : rapide (< 1 session), moyen (1-2), lourd (> 2).

### Étape 0 — Socle de travail (rapide)

- [ ] 0.1 Branche `research/positions-de-force-v3` depuis `release/v8.1.0` (D13), worktree
      dédié `LevelUp-wt-positions-de-force`.
- [ ] 0.2 Reprendre de `wt/power-positions`, par `git checkout <ref> -- <chemins>` : les paquets
      `internal/analysis/powerpos`, `/geo`, `/fusion`, les outils `cmd/mappower-build`,
      `cmd/mapgeo-build`, les documents `.ai/V7.5/positions_de_force/` (déplacés sous
      `.ai/positions_de_force/` pour le chantier vivant) ; ne rien reprendre d'autre.
- [ ] 0.3 Faire compiler contre la base actuelle sans changer aucun comportement ; rejouer les
      tests de septembre.
- [ ] 0.4 Outils de recherche (`mappower-build`, `mapgeo-build`) : racine des données par
      `--data-root`, refusée si elle désigne la copie principale. CLI de l'appli : par
      `LEVELUP_REPO_ROOT` (existant, aucun flag neuf).
- [ ] 0.5 Monter l'instance de recherche (option A, si retenue) : `config/`, `reference/`,
      `metadata.duckdb` au repos, `.env.local`, `db_profiles.json` de recherche, jonction des
      jetons ; points durs 1 à 5 de § Acquisition vérifiés un par un.
- [ ] 0.6 Contrôle à vide : `levelup` avec `LEVELUP_REPO_ROOT` sur le dossier de recherche
      lit la base copiée (compte du registre = 10 026) et n'ouvre AUCUN fichier de la copie
      principale (vérifié par les chemins journalisés au démarrage).

Gate 0 : `go build ./internal/analysis/powerpos/... ./cmd/mappower-build/ ./cmd/mapgeo-build/` ;
`go test ./internal/analysis/powerpos/...` vert ; `go vet -tags research ./cmd/mappower-build/` ;
`golangci-lint run` sur les paquets touchés 0 issue ; `go test ./internal/archlint/` vert.

### Étape 1 — Échelle et grandeurs physiques en unités (rapide)

- [ ] 1.1 Constante unique du facteur unités → mètres (3,048), avec sa preuve (audit §2).
- [ ] 1.2 Mesurer dans les rejeux existants, en unités : diamètre de collision (distance
      minimale entre Spartans, déjà relevé 0,26-0,48 au 0,1e centile), hauteur du point de tir
      (origine des tirs moins altitude des pieds) = hauteur des yeux, hauteur de saut
      (0,85 au lot 5.7.5). Ces trois mesures fixent `HauteurYeuxU`, `HauteurLibreU`,
      `SautMaxU`, `MarcheMaxU`.
- [ ] 1.3 Renommer et re-valoriser les grandeurs physiques de `powerpos/geo` (`Parametres`)
      en unités ; `ParametresParDefautV3` neuf, `ParametresParDefaut` de septembre gardé tel
      quel comme preuve.
- [ ] 1.4 Contrôle : sur Recharge, la part des positions jouées (trajectoires) qui tombent
      sur un sol dérivé, avant / après correction. Attendu : hausse ; consigné quel qu'il soit.

Gate 1 : tests unitaires des nouvelles constantes ; rapport `ECHELLE_2026-xx-xx.md` avec les
trois mesures et le contrôle 1.4.

### Étape 2 — Sondes d'acquisition, sans téléchargement de masse (rapide)

- [ ] 2.1 Disponibilité des films par année (2023 → 2026, D11) : manifeste seul, 10 tirages
      par mois sur les matchs classés du réservoir (`diag_film_avail`).
- [ ] 2.2 Décodabilité par build : sur 3 films téléchargés par build rencontré, part de kills
      positionnés obtenue ; un build sous 90 % est exclu de la mesure (seuil écrit ici).
- [ ] 2.3 Historique complet de calloussss et Brooks SVP : nombre de matchs par année et par
      playlist classée, sans rien écrire (`probe-world-stats --depth-pages` lit l'historique
      d'un compte sans profil et n'écrit pas de match).
- [ ] 2.4 Chiffrage : films à télécharger par priorité (P1 classés cartes à oracle lobby
      ≥ 1500 ; P2 classés cartes à oracle ; P3 classés autres cartes), matchs ≥ 2023 seulement
      (D11), disque et temps prévus contre le plafond de 100 Go (D12).

Gate 2 : document `ACQUISITION_2026-xx-xx.md` avec les quatre tableaux ; **arrêt pour accord
utilisateur** sur les paquets de l'étape 3 (D8). C'est un blocage nécessitant l'utilisateur, pas
un report.

### Étape 3 — Acquisition par paquets (lourd, sur accord par paquet)

- [ ] 3.1 Historiques : `levelup sync-full` pour calloussss, Brooks SVP (historique depuis
      2023, D11) et Nuzzles (rattrapage depuis le 25/10) dans l'instance de recherche ;
      registre, participants avec attentes et CSR de tout le lobby viennent avec.
- [ ] 3.1b Compteur disque de la recherche (films + rejeux) relevé à chaque paquet ; arrêt et
      demande à l'utilisateur dès 100 Go (D12).
- [ ] 3.2 Paquet pilote de films : 50 films P1 par `levelup archive-films` ; mesure du disque,
      du temps, des films absents côté serveur.
- [ ] 3.3 Paquets P1 suivants, taille fixée sur la mesure 3.2, chacun annoncé (D8).
- [ ] 3.4 Paquets P2 (puis P3 si accord).

Gate 3 : comptes par carte et par famille de mode dans la base de recherche (requête écrite au
document), zéro erreur de décodage non comptée, journal de chaque paquet.

### Étape 4 — Positions et trajectoires (moyen à lourd)

- [ ] 4.1 Positions de kills des matchs ajoutés : `levelup backfill-killsource` hors ligne
      (passe légère, pic mesuré à 422 Mio sur le pire film).
- [ ] 4.2 Trajectoires (occupation) des lobbys de niveau ≥ 1300 : `levelup backfill-replay`
      par paquets `--limit`, `--mem-limit-gib` fixé sur le paquet pilote, un film à la fois,
      sur accord par paquet (D8). Aucune passe « trajectoires seules » n'existe ; en créer une
      est hors périmètre (découverte à consigner si le coût bloque).
- [ ] 4.3 Seuils de corpus par carte et famille de mode, ÉCRITS avant toute mesure (D6).

Gate 4 : tableau carte × famille de mode : matchs, kills, heures de trajectoire, lobbys ≥ 1500.

### Étape 5 — Mesures et réglage v3 (lourd)

- [ ] 5.1 Avantage de duel corrigé du niveau (D3), paquet pur testé (cas synthétiques : un
      lieu sans avantage + joueurs forts doit rendre 0 ; un lieu avantagé + joueurs égaux doit
      rendre l'avantage).
- [ ] 5.2 Densité d'occupation pondérée par le niveau, par famille de mode.
- [ ] 5.3 Géométrie en unités (étape 1), validée par les lignes de vue prouvées des kills :
      part des kills (hors grenades, véhicules, mêlée) dont la ligne tueur → victime est
      visible pour la géométrie ≥ 95 %, sinon correction de l'occlusion avant de continuer.
- [ ] 5.4 Choix de `h` par reproductibilité (D4), pour chaque signal.
- [ ] 5.5 `ReglageV3` (poids, sélection relative) figé sur les cartes de calibrage, avec
      horodatage, AVANT le verdict.

Gate 5 : tests des paquets purs verts ; `ReglageV3` sérialisé et daté dans le document de
mesure ; planches par carte.

### Étape 6 — Verdict (moyen)

- [ ] 6.1 Verdict v3 contre l'oracle v2, critères 1 à 4, cartes de validation seules.
- [ ] 6.2 Contrôles : témoin géographique, moitiés de matchs, « sans Nuzzles », lobbys Onyx
      seuls.
- [ ] 6.3 Diagnostic de chaque forte manquée et de chaque faux positif, nommés.
- [ ] 6.4 Verdict v1/v2 rejoués à l'identique (non-régression des preuves de septembre).

Gate 6 : `go test -tags research -run VerdictV3 ./cmd/mappower-build/` PASS ; document
`VERDICT_V3_2026-xx-xx.md`, verdict en une ligne en tête.

### Étape 7 — Clôture de la recherche (rapide)

- [ ] 7.1 GO / NO-GO consigné, journal, REGISTRE_REPORTS.
- [ ] 7.2 Si GO : proposition du plan de production (non écrit ici).
- [ ] 7.3 Nettoyage des caches dédiés (`scripts/disk-hygiene.ps1 -Lot positions-de-force-v3`).
- [ ] 7.4 Sort de la branche et du dossier de recherche (fusion des outils, archivage, ou
      suppression) : décision utilisateur, rien de supprimé sans elle.

## Risques connus

| Risque | Effet | Parade |
|---|---|---|
| Films anciens non servis ou sur builds mal couverts | corpus 2021-2023 inaccessible | sondes de l'étape 2, exclusion par build chiffrée |
| Versions de carte successives (`map_version_id`) | positions décalées entre versions | contrôle par version comme pour Live Fire - Ranked ; une version décalée est séparée |
| Un joueur domine le corpus | carte de ses habitudes | critère 4 « sans Nuzzles » ; élargissement calloussss / Brooks |
| Positions de dissuasion (peu disputées) | invisibles au duel | signal d'occupation (D5 b) |
| Oracle mince (28 fortes, cartes à 2 fortes) | verdict fragile | règle « indéterminé » de la v2 conservée ; pas de densification de l'oracle dans ce plan |
| Pic mémoire de décodage / cuisson | poste saturé | D8, paquets mesurés ; jamais en même temps qu'une cuisson de la copie principale |
| Rotation de jeton perdue | authentification cassée sur le poste | jonction vers le magasin principal, jamais de copie ; serveur local arrêté pendant l'acquisition |
| ~29 % des films absents côté serveur | corpus plus mince que prévu | chiffré à l'étape 2 avant tout paquet |

## Découvertes hors périmètre (à consigner, ne pas traiter)

1. (audit §2) `SpartanJumpHeightM` et d'autres constantes du décodeur et du rejeu portent
   l'unité « mètre » sur des valeurs en unités monde ; l'affirmation « positions du `.mvar` en
   mètres » de `ETAT_DE_L_ART_FORGE_PALETTE_ZONES.md` est contredite.

## Protocole de reprise

Lire ce fichier (statuts), puis l'audit, puis le dernier document produit sous
`.ai/positions_de_force/`, puis `git log --oneline -10` sur la branche. Reprendre au premier
item non statué de la première étape non close. Une retouche de réglage après 5.5 invalide le
verdict et se traite comme une nouvelle étape 5. Chaque clôture d'étape ajoute une ligne au
journal ci-dessous et une entrée au `thought_log` ; aucun commit sans accord de l'utilisateur.

## Journal d'avancement

| Date | Étape | Statut | Note |
|---|---|---|---|
| 2026-10-27 | Plan | projet | écrit sur l'audit du jour ; inventaire de l'outillage d'acquisition fait (lecture de code) ; O1-O3 ouvertes, option A recommandée pour O1 ; relu à la grille `plan-review` ; aucune exécution |
| 2026-10-27 | Plan | décisions | D10-D13 tranchées par l'utilisateur (instance isolée avec jetons du dépôt principal, ≥ 2023, 100 Go, branche) ; reste le go d'exécution |
