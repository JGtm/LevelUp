— Tâches et TODO centralisés

---

## 🔄 Aucune tâche en cours (plan `.ai/PLAN_BACKLOG_2026-10-16.md` clos le 2026-10-18 ; fusion dans `feat/v75` = geste du user)

---

## 📋 Backlog

---

### [rejeu/identité] 15 vies sans nom restantes dans les rejeux cuits

Reste du lot film E8.4 (plan `.ai/PLAN_BACKLOG_2026-10-16.md`) : sur les 36 films en cause (449 vies
sans nom), 15 vies restent sans nom, aucune lecture du film ne les nomme : `72c48587` (7 vies, table
d'identité refusée pour siège vacant intercalé, aucune lecture d'index dans les chunks), `5da6fd30`
(6 vies, film de 38 s, trois bots que BOT_METADATA ne déclare jamais), `7aa132d1` (2 vies, pistes
parasites hors plage des bipèdes, slots 36 et 292). Règle du user : aucune vie anonyme. Reprise : une
mesure par film des enregistrements de création qui nomment ces pistes ; arrêt si aucune lecture n'en
sort. Hors périmètre, déjà décidé : les 5 films de build inconnu (abandonnés par le user le
2026-10-15) et les 2 films Firefight. **Effort : S.**

---

### [rejeu] Découvertes du lot film (E8) non traitées

Relevées par E8.1, E8.3 et E8.4 (détail : section « Découvertes » du plan, hors périmètre de leur
lot) : l'entame (`kill_openings`) ne lit pas le bord, donc un côté sans échantillon à l'instant décalé
reste sans position alors que la tenue le placerait ; `siegesDe` donne un second corps vivant à 83 m
à une victime (`4f77afc1`) et la mort perd son lieu ; des lectures d'occupation attachent un bipède à un
parent qui n'est ni un véhicule échantillonné ni un bipède (`5676a9ba`) ; le châssis de véhicule
`b857fb95` (`e449a696`, `99b8ba5c`) n'est pas dans la table des familles (marqueur neutre). **Effort : S
chacune, à instruire une par une.**

---

### [ui] Indicateur « Analyse en cours » (tuile + vue match) — maquette validée le 2026-10-16

Maquette validée par l'utilisateur le 2026-10-16 (artefact Claude Design, canvas Main / MatchView / Tile),
lot jamais inscrit au backlog jusqu'ici (découverte E0 du plan `.ai/PLAN_BACKLOG_2026-10-16.md`). Retenu :
pastille « Analyse en cours » sur l'image de carte de la tuile, emplacements en attente (frags assistés,
citations, rejeu), bandeau INFO dans la vue match ; état donné par le SERVEUR, rechargement auto ~1 min
tant que l'analyse court, disparition au-delà de 24 h ; libellés « d'ici quelques instants » ; FR + EN,
tokens de couleur. **Effort : M.**

---

### [dette/backlog-2026-10-16] Découvertes non traitées du plan du 2026-10-16

Section « Découvertes » de `.ai/PLAN_BACKLOG_2026-10-16.md` (détail, `fichier:ligne` et origine par
étape). Les principales, à trier :
- **LUSR** : un match exclu reste tracé dans le graphe de carrière avec sa note d'avant (Q8
  `queries_career.go`, `profile/queries.go` sans filtre `is_excluded`) ; `skippedWriteFailed` laisse
  `RunLUSRV2Shadow` rendre `nil` (une exclusion dont une écriture canonique échoue répond « terminé ») ;
  la lecture du précédent par la vue coûte ~5 ms de plus par match noté (index en mémoire possible) ;
  la fixture `psa_index_repro_msr_fixture_test.go` recopie les anciennes requêtes. **S chacune.**
- **Multi-titre** : `auto_sync_engine.go` et `sync_handler.go` passent le provider d'Infinite aux moteurs
  d'autres titres ; passes `--csr`/`--shared-csr` aux chemins fixés sur le titre par défaut (le titre
  de test `synthetic_title_b` déclare la capability) ; ratchet `no_slug_comparison_test.go` sans `cmd/` ;
  `cmd/levelup` n'installe pas les étapes de migration par titre ; `RunForDB` reste dans 3 outils ;
  la route des sessions parcourt les profils `auth_only`. **S à M.**
- **CI / outillage** : `scripts/gate-push.ps1` joue une seule suite en série ; le filtre de chemins ne voit
  pas tous les tests Go qui lisent `apps/web/src` (rattrapé par la CI complète de `feat/v75`) ; le corpus
  `-tags=gamefiles` n'est pas vert sur le poste (dépôt `.ai/re_dump/mapvar` absent, banc Cliffhanger
  64,4 % contre 64,7 %). **S.**
- **Citations** : échec de `LoadCitationMappings` avalé (page vide) ; la barre de filtres de la page est
  sans effet (corps ignoré) ; noms de tuiles tronqués (« Maîtrise de la … » indiscernables) ; le serveur
  ne rejoue jamais le seed des citations sur une metadata existante. **S chacune.**
- **Divers** : `StepDeviceCode.tsx` garde l'effet de récupération sans garde (sondage déjà arrêté) ;
  premières vérifications de participation soumises au budget de lecture ; `cmd_reset_bitmasks.go`
  recopie d'autres constantes ; deux `continue` silencieux dans `weapon_resolver.go` ; démo sans
  `match_bomb_stats` ; coût de `/tactical/maps` (une résolution de fond par carte) à mesurer ;
  instruments `research` de `film/replay` en `filepath.Join(depot, "data", ...)` (~31 fichiers, motif ancien). **S.**

---

### [replay/sons] Fins de partie multi-équipes par couleur — écran + annonceur

Noté le 2026-08-27 (chantier rejeu 2D, plan `.ai/V7.5/PLAN_REPLAY_CADRAGE_VICTOIRE.md`,
branche `wt/replay-cadrage-victoire`). L'écran de victoire et les sons de fin (lots B/C)
couvrent les matchs à 2 équipes + le FFA gagné ; les modes MULTI-ÉQUIPES (3+) n'ont ni
écran (décision D-B1) ni son. Or le jeu porte une famille complète d'annonces dédiées,
identifiée par transcription locale des packs annonceur :

- **8 répliques FR** « Partie terminée, l'équipe X est déclarée vainqueur » — bleue
  `1070034924`, rouge `808622693`, cyan `83592248`, mauve `186794961`, verte `265309140`,
  citron `784566745`, jaune `868146650`, orange `399957729` (ids `.wem` du pack
  `French(France)/sb_001_vo_ai_mp_announcer.pck`) ;
- **8 jumelles EN** « Game over — <color> team wins » — red `1005389916`, blue `101785491`,
  green `1010879786`, purple `100056384`, yellow `927455187`, orange `564119611`,
  lime `92374`, cyan `256805823` (pack `English(US)/…`).

**BLOQUÉ FAUTE DE DONNÉE (plan backlog du 2026-09-26, lot A5, statué `[!]` le 2026-09-27).**
Le plan d'exécution est prêt dans `.ai/PLAN_BACKLOG_2026-09-26.md`, section A5 : pièces, items
A5.0 à A5.7, et décision D-9. D-9 garde l'écran du joueur regardé, ajoute « Victoire de l'équipe
<Nom> » en défaite, et fait jouer la réplique de la couleur du gagnant pour tous. Deux faits établis
en lisant les données le 2026-09-26 bloquent le lot :
- **aucun match multi-équipes jouable** : il n'y a qu'un seul match à 3 équipes ou plus, de 2
  joueurs ou plus chacune, dans toute la base partagée locale (2025-08-13, aucun joueur suivi).
  Aucun des 111 artefacts de rejeu ne porte d'équipe au-delà de 1 ;
- **la table équipe → couleur annoncée n'existe nulle part** : les couleurs de `teamNames.ts`
  ne correspondent pas aux 8 couleurs de l'annonceur (magenta, doublon orange-rouge).

**Condition de reprise** : l'utilisateur joue une partie à 3 équipes ou plus (matchmaking ou
partie personnalisée), la synchronise, et note la couleur annoncée pour l'équipe gagnante. Le lot
peut alors partir, en commençant par la recherche hors ligne de la table dans les fichiers du jeu
installé (A5.0). Les 16 répliques sont déjà extraites et converties sous
`Desktop/Halo Infinite - Sons armes/_fin_partie/annonceur_{fr,en}_wav/` (non normalisées ; les
copies de `livraison/` y sont restées en 4 canaux). **Effort : M.**
**Constat du user le 2026-10-09 : aucun mode à plus de deux équipes (hors FFA) n'est disponible en
jeu actuellement.** Le lot reste en attente d'un tel mode.

---

### [data/objectifs] Blocs `EliminationStats` / `InfectionStats` — BLOQUÉS faute de donnée

Réduit le 2026-07-25 (v7.2.1, V721-02). Sur les 4 blocs de mode non extraits, **3 sont
livrés** (Stockpile, Extraction, et `VipStats` qui n'était même pas déclaré dans
`StatsBundle`) — 18 colonnes. Les 2 restants ne sont **pas** un manque de travail mais un
manque de donnée, établi sur payloads réels (P0, 10 matchs interrogés) :

- `EliminationStats` : les 2 seuls matchs Attrition en base ne portent **aucun** bloc de
  mode (`Stats` = `CoreStats` seul). Aucun mode Elimination parmi les 65 modes distincts
  présents en base.
- `InfectionStats` : « Survive The Undead 3.0 » est un Firefight UGC (catégorie 41), pas
  un Infection. Aucun mode Infection en base.

**Déblocage** : jouer une partie matchmaking de chacun de ces modes, puis la synchroniser. **Constat du user le 2026-10-09 : ni Élimination ni Infection ne sont disponibles en jeu actuellement.**
Le patron existe alors en triple exemplaire — **effort : S par bloc**. Interdiction
explicite d'inventer le schéma par analogie.

### [ops/h5] `weapon_labels` / `cmd/h5-metadata-fetch` — VERDICT : à CONSERVER (seuls porteurs des icônes)

Audit fait le 2026-07-26. Il répond à la question posée par la version précédente de cette
entrée (« la couverture du registre est-elle totale ? ») et en déplace la conclusion.

**Ce qui est établi** :
- **Couverture du registre : 100 %.** Les 66 `weapon_id` H5 connus ont désormais un
  `weapon_key` (`weaponRegistryH5Stock` étendu dans
  `internal/games/halo_infinite/migrations/weapon_registry.go`, noms via
  `config/titles/halo_5/mappings/weapon_names.toml`). Le critère de retrait posé le
  2026-07-25 est donc **atteint pour les NOMS**.
- **Mais la table `weapon_labels` et le CLI `cmd/h5-metadata-fetch` restent VIVANTS**, pour
  une raison qui n'était pas dans le périmètre initial : ils sont les **seuls porteurs des
  `icon_url` d'armes H5** (consommées par `internal/games/halo_5/adapter_asset_urls.go`, qui
  indexe l'URL d'icône par `name_en` officiel) **et du catalogue admin d'assets**
  (`internal/platform/duckdb/metadata_repo_assets_list.go`). Retirer le CLI, c'est perdre les
  icônes d'armes H5, pas seulement un repli de nom.

**Ce qui resterait supprimable, et sous quelle preuve** : uniquement le **3e maillon du
COALESCE de NOM** dans `internal/platform/duckdb/weapon_resolver.go`
(`… NULLIF(wl.name_fr,''), NULLIF(wl.name_en,'') …`), probablement mort depuis que le
registre couvre 100 % des ids. **Preuve exigée avant retrait** : un relevé LIVE de
`GET /admin/monitoring/weapon-coverage?title=halo_5` (session admin requise) montrant
**0 identifiant non résolu par `weapon_name_labels`**. Tant que ce relevé n'est pas produit,
ne rien toucher : le maillon coûte une clause SQL, son retrait à l'aveugle coûterait des noms
d'armes vides en production.

**Hors périmètre du retrait, dans tous les cas** : la colonne `weapon_labels.name_en` sert
AUSSI de clé de résolution d'image (`COALESCE(wl.name_en,'') AS name_en` dans le même
resolver) — elle ne suit pas le sort du maillon de nom. **Effort : S** (le relevé live).

---

### [Migration] Cible desktop Tauri web-first, sans réécriture Rust métier — V9 POTENTIELLE

> ⏸️ **Gardé de côté** (2026-06-09) : conservé pour distribution desktop néophyte future. Note : le cutover Go étant fait, le « backend Python local packagé » ci-dessous doit se lire **backend Go local** — à re-cadrer si réactivé.
> **Décision du user (2026-10-09)** : gardé comme chantier potentiel de la V9 ; rien à faire avant.

**Noté le** : 2026-04-12 | **Priorité** : Moyenne (distribution simplifiée, non bloquante pour les slices MVP)

**Référence plan** : `.ai/MIGRATION_MASTER.md`, `.ai/migration/DECISIONS.md`

**Problème** : La migration React/FastAPI améliore l'UX et le déploiement web, mais ne résout pas à elle seule le cas utilisateur néophyte qui ne doit ni installer Python, ni lancer `pip`, ni manipuler un terminal. Il faut documenter une cible desktop installable qui n'abîme pas la stratégie web/VPS.

**Décision cible** : Conserver une architecture **web-first** (`apps/web` + `apps/api`) comme source de vérité produit, puis ajouter **Tauri comme coque desktop** optionnelle. Rust est explicitement **hors périmètre métier** : aucune logique de sync, auth Halo, DuckDB, filtres, agrégats, visualisations ou contrats API ne doit être réécrite en Rust.

**Solution** : Préparer un spike de packaging Tauri autour du frontend React existant et d'un backend FastAPI/Python local packagé, avec un contrat d'intégration minimal et réversible.

**Changements ciblés** :
1. Architecture : figer la règle `React navigateur d'abord`, `FastAPI canonique`, `Tauri simple shell desktop`
2. Packaging : définir comment lancer/arrêter proprement le backend Python local depuis l'app desktop, avec gestion des logs, ports, répertoires de données et erreurs de démarrage
3. Frontend : isoler les appels natifs desktop derrière une couche d'adaptation pour que l'app reste exécutable telle quelle sur navigateur et sur VPS
4. Données locales : cadrer les chemins Windows pour DuckDB, médias, cache et configuration utilisateur sans hardcoder de chemins machine
5. Distribution : évaluer installateur Windows, taille du bundle, temps de démarrage et absence de prérequis Python côté utilisateur final
6. Exploitation : préserver explicitement la cible VPS en interdisant toute dépendance produit au runtime Tauri/Rust
7. Go/no-go : définir les critères du spike (installation propre, backend embarqué stable, auth utilisable, fichiers locaux OK, perf de lancement acceptable)

**Point de vigilance** : Tauri implique mécaniquement une fine couche Rust côté shell. Ce point est acceptable uniquement comme détail d'enveloppe technique. Toute dérive vers des commandes Rust métier, un stockage canonique côté Tauri ou une divergence desktop-only dans les flux React/FastAPI doit être refusée.

**Activités supplémentaires à prévoir si/quand Tauri est réactivé** (notées le 2026-07-19, non planifiées) :
1. **Distribution de release** : pipeline de build/signature des installateurs desktop (au moins Windows ; macOS/Linux à trancher), versionnage aligné avec les releases web/API, publication des artefacts sur **GitHub Releases** du repo (pas d'hébergement/download depuis le site LevelUp — juste un lien vers la release GitHub la plus récente si besoin d'un CTA produit).
2. **SISU/SSO Xbox — améliorer le flux desktop** : il existe bien deux mécaniques d'auth Xbox distinctes.
   - **Flux web actuel (en place aujourd'hui)** : OAuth « live » classique (`login.live.com/oauth20_desktop.srf` pour un client public/native, ou device-code flow pour du headless/CLI) → user token → XSTS.
   - **Flux SISU natif** : POST vers `sisu.xboxlive.com/authorize` (AccessToken + AppId + DeviceToken + SessionId) — utilisé par les apps Xbox/mobiles natives, potentiellement plus rapide/fluide (moins d'allers-retours navigateur), mais suppose un contexte natif (device token, app registrée côté Xbox) plus contraignant à mettre en place que l'OAuth desktop classique.
   - Point technique à noter : **WAM (Web Account Manager)**, le broker d'auth Windows utilisé par MSAL pour du SSO silencieux avec le compte Windows courant, n'est **pas disponible pour Xbox** — donc pas de raccourci via WAM, il faudra creuser SISU directement ou rester sur l'OAuth desktop existant amélioré (moins d'allers-retours navigateur, SSO local dans la coque Tauri).
   - À trancher au moment venu : est-ce que le gain (fluidité) justifie l'effort d'implémentation SISU vs. optimiser l'OAuth desktop actuel dans la coque Tauri.
3. **Stockage local — migration vers AppData** : tout ce qui est aujourd'hui stocké sur disque à côté de l'app (DuckDB `data/titles/`, `data/auth/`, `data/global/`, `data/sessions/`, config `db_profiles.json`/`app_settings.json`/`.env.local`) ne peut pas rester dans le bundle applicatif Tauri — l'app doit pouvoir être mise à jour/réinstallée sans perdre ces données. Il faut cadrer un chemin utilisateur type `%APPDATA%/LevelUp/` (Windows) et équivalents autres OS, cohérent avec `PathResolver`, sans hardcoder de chemin machine. **À ce moment-là : faire un audit exhaustif de tout ce qui est écrit sur disque** (pas seulement `data/` — logs, caches, fichiers temporaires, médias indexés, tout chemin actuellement dérivé de `REPO_ROOT`) pour ne rien oublier dans la bascule vers AppData. **Précision (2026-09-27, plan backlog du 2026-09-26)** : l'inventaire des chemins d'état existe désormais (journal du lot B5 du plan, et `config.LoadServer` / `RuntimePaths`). `PathResolver` n'a qu'une racine, qui sert à la fois la configuration versionnée et les données d'exécution. Une racine de données distincte (environ 70 fichiers, 121 appels de `NewPathResolver`) est le préalable technique de cette bascule.

---

## 🎮 Backlog — Coach proactif × Prestige (post-V2)

Référence : ADR 0020 — Coach proactif : pont vers Prestige. ADR 0021 — Synthèse dynamique de Template et Arc ad-hoc.

> **MàJ 2026-07-17** : les 3 extensions autrefois parkées sont LIVRÉES (train backlog) —
> V2.1 télémétrie `source` + endpoint diag, analyseur de tuning de la grammaire
> (recommandations, validation manuelle), et canal externe Discord webhook (opt-in, OFF
> par défaut). Voir « Récemment complété ».

**Chantiers suivis dans des plans dédiés** (retirés du backlog actif — les plans font foi) :
- **Arcs multi-titres** — indépendance stricte par titre, Option A (retrait `arc_titles`)
  confirmée le 2026-07-18 : [.ai/PLAN_CROSS_TITLE_ARCS_2026-07.md](.ai/PLAN_CROSS_TITLE_ARCS_2026-07.md).

Enrichissements possibles (non planifiés) : funnel `coach_proposal.status` dans l'analyseur de
tuning, overlay Discord par titre (`LoadNotifyConfigForTitle`), exposition des catégories
forwardées via settings.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-10-18 | **[données] Indicateur « données de compétence » des 157 matchs synchronisés du 25 au 30 mai** (plan `.ai/PLAN_BACKLOG_2026-10-16.md`, après E10, sur décision du user) : la donnée était en base (MMR d'équipe présent ; joueurs suivis concernés : JGtm 137 matchs, Chocoboflor 87, Madina97294 75, XxDaemonGamerxX 10, joués de nov. 2021 à mai 2026), seul le bit skill (4) manquait. `levelup reset-bitmasks` serveur arrêté, base sauvegardée avant (`data/backups/avant_reset_bitmasks_2026-10-18/`) : 157 lignes `match_registry` et 1 302 lignes `match_participants`, dry-run rejoué = 0, `levelup healthcheck` tout OK. Les bases locales étant copiées vers le VPS à la release, rien à faire en prod. |
| 2026-10-18 | **[déploiement] Recalcul des dérivés Halo 5, LUSR v2 de tous les joueurs, séquence d'après déploiement** (plan `.ai/PLAN_BACKLOG_2026-10-16.md`, E9, branche `feat/backlog-2026-10-16`) : (1) `levelup backfill --title <slug>` est honoré par `--engagement-scores`, `--citations`, `--citations-recompute-all`, `--composite-only`, `--lusr` (v2), `--perf`, `--assists-model` et `--compare-formulas` (un pilote et une table de passes remplacent six paires de boucles recopiées ; `--all` prend les profils du titre, `--csr`/`--shared-csr` refusent un autre titre, migrations de la base DU titre) ; sur Halo 5 les passes de citations répondent « non supporte » par la capability `citations.engine`, sans rien ouvrir ; `POST /backfill/start` prend un champ `title` (404 si le joueur n'est pas suivi sur ce titre), construit le moteur et le provider shared DU titre (`config.SharedProviderForTitle`, aussi utilisé par le replay LUSR de l'admin, qui passait le provider d'Infinite à un joueur Halo 5). (2) `POST /settings/sessions/recalculate` recalculait CHAQUE profil sur les bases du titre par défaut (titre vide) : un profil Halo 5 n'était jamais recalculé ; corrigé par profil, test rouge avant. (3) « Dates de paliers » : source `cmd/backfill_milestone_dates` (hors image de prod, décision au runbook). (4) Runbook : section « Post-deploy recomputes (v7.5) » (R1 à R10 : 157 matchs, dérivés Halo 5, LUSR v2 des deux titres, séquence film, sessions, accords du pool, re-seed de la démo), commandes et routes exactes. Rien n'est exécuté : geste du user. |
| 2026-10-18 | **[rejeu] Lot film : positions du coup fatal, score KOTH Lattice, Harvest, vies sans nom** (plan `.ai/PLAN_BACKLOG_2026-10-16.md`, E8, branche `feat/backlog-2026-10-16`) : (1) positions : la passe lit le journal fusionné par xuid (joueurs renommés), tient la dernière position dans la même vie, lit le siège du véhicule et place le tueur déjà mort à sa mort ; le contexte de mort lit la victime par la même lecture que `kill_positions` (source unique, tolérance de 0,5 m retirée) : tueurs sans position 1 476 -> 78 et victimes 1 028 -> 85 sur 233 films (22 535 frags), 2 contextes de mort sur 23 326 non placés contre 148 gagnés. (2) vies sans nom : 449 -> 15 sur les 36 films en cause (index des chunks hors table, chaîne de contradictions, corps de bot arrivé avant sa première déclaration), aucune vie nommée perdue sur les 2 660 films comparés. (3) score KOTH Lattice : une émission hors domaine (canal B non nul d'un slot d'équipe) remplaçait le point réel par 0 et le seuil de garde retenait un intervalle fusionné : `7de0b91d` rend 40 sans contradiction, `26602661` sans descente, 6 films changent de série, les autres KOTH inchangés. (4) Harvest : carte Forge, canevas `fo11_blank` prouvé par le level_id de la variante (426470249, unicité 1/1), catalogues à 85 cartes (bornes) et 131 (objectifs), `e449a696` et `99b8ba5c` cuisent (158 et 144 pistes, 0 vie sans nom). Révisions montées une fois : `objectives-2026-10-17`, `isolement-2026-10-17-positions`, schéma 99, `placement-2026-10-17`, `grammar-2026-10-18`, `killsource-2026-10-18`. Gate : `replay-corpus-gate` 19/19, 0 perte. Reste après la fusion (serveur arrêté, puis prod, le user prévenu) : `backfill-replay --only-existing`, `backfill-killsource`, usage-summary, `backfill-pad-tiers --force` ; contrôle visuel du user en lecture sur `7de0b91d` et `e449a696`. |
| 2026-10-17 | **[perf/tactique] Chargement de la page Tactique : de 96 requêtes à 5 hors images** (plan `.ai/PLAN_BACKLOG_2026-10-16.md`, E7, branche `feat/backlog-2026-10-16`) : le chargement coûtait 2 requêtes par carte ouvrable (le calage `GET /tactical/{map}/background`, puis l'image `background.png`) en plus de 4 requêtes fixes, soit 96 pour 46 cartes. `/tactical/maps` porte désormais le calage du fond de chaque carte ouvrable (`calage_fond`, absent pour une carte sans fond figé ; même source que la route par carte ; panne journalisée sans arrêter la grille) ; la vignette lit le cadre du fond dans la carte (plus aucune requête de calage par vignette) et ne demande son image qu'une fois entrée dans le champ de vision, et seulement si la carte a un calage (hook `useVisibleUneFois`, même clé de cache que le plan principal). Repli de D-10 : pas de `<img loading="lazy">`, car un `<img>` n'emporte pas `X-LevelUp-Title` et l'API résoudrait le titre de la session. Restent 5 requêtes de données (4 fixes + le calage de la carte du plan) puis les images des vignettes visibles. Mesure établie par le code et un test à 46 cartes ; la mesure en navigateur (onglet Réseau, temps d'affichage) reste au contrôle visuel. |
| 2026-10-17 | **[dette/citations] Premiers frags du profil, capability des citations, maîtrise Carabine à impulsion** (plan `.ai/PLAN_BACKLOG_2026-10-16.md`, E6, branche `feat/backlog-2026-10-16`) : (1) `computeFKFD` lisait des colonnes absentes de `highlight_events` et avalait l'erreur : premiers frags et premières morts à 0 pour tous les titres ; il passe par le chargeur unifié des événements (repli sur le journal des frags pour Halo 5), `analysis.FirstBlood` et le nouveau `analysis.FirstDeath`, sur les matchs du joueur, Campagne exclue, erreur journalisée ; tests Infinite et Halo 5 rouges avant, verts après. (2) `citations.engine` passe à `supported` pour Infinite (TOML, repli de l'adapter, parité verte ; seul lecteur : la matrice des fonctionnalités admin). (3) Diagnostic de la maîtrise Carabine à impulsion absente de la page Citations : aucun défaut de code (ligne présente et active au seed et dans les copies à froid de la metadata, chaîne service/page rejouée sur copie, image suivie par git, les 32 maîtrises `weapon_kills:` du seed se résolvent) ; reste à lire l'état de la metadata de l'instance regardée (requête au journal du plan) et, au besoin, `levelup seed citation-mappings`. Storm : aucune citation ajoutée (l'équivalent Infinite est la Carabine à impulsion). |
| 2026-10-17 | **[dette] Dettes Go du plan du 2026-10-15** (plan `.ai/PLAN_BACKLOG_2026-10-16.md`, E5, branche `feat/backlog-2026-10-16`) : code mort supprimé avec ses tests et lignes de baseline (`FindMatchesMissingParticipantBits`, `loadAwardsForScoreboard`, `WithAwardsRepo`, `CachedHighlightEventsRepo`, `BatchBuilder.AddLUSRComponents`, paramètre `titleSlug` de `SquadUsualContexts` remplacé par `prestige.Deps.TitleSlug`, 14 réexportations de constantes) ; erreurs avalées journalisées avant la dégradation (libellés d'armes, image de carte, modes connus, nom de paire, assistances attendues) et `ApplyResolutionViews` remonte l'échec de `v_match_full` ; DDL de `v_match_full` en source unique `migration.VMatchFullViewSQL` + garde-rail archlint ; une requête abandonnée ne bascule plus en lecture live ni ne compte ni n'écrit d'ERROR sur la vue match ; garde de la Campagne étendu au registre lu sur fenêtre de temps (2 dispenses d'outils opérateur) ; `reset-bitmasks` sur `matchflags.BackfillParticipants`, mapper OpenSpartan honnête sur les médailles (`EventsLoaded` supprimé), `--title` sur 3 CLI et passage du slug à `backfill-killsource` ; `match_bomb_stats` exporté dans le snapshot (un cut ancien reste lisible) ; un ami suivi sans player DB est une absence et non plus une panne des extras. Reste : le bit skill de 157 matchs (geste de déploiement, runbook). |
| 2026-10-17 | **[data/LUSR] Le recalcul après exclusion d'un match utilisait encore LUSR v1** (plan `.ai/PLAN_BACKLOG_2026-10-16.md`, E4, branche `feat/backlog-2026-10-16`) : la v1 du LUSR est supprimée (batch, étape de post-sync, aperçu `levelup backfill --lusr --dry-run`, interrupteurs `LEVELUP_LUSR_V2_ENABLED` et `LEVELUP_LUSR_CANONICAL`, `RecomputeAfterARTRebuild`, tests v1). L'exclusion d'un match rejoue la chaîne v2 du joueur (sentinelle de reset puis rejeu ; l'écrivain partagé est pris par rafales courtes, jamais en continu) et la v2 saute les matchs `is_excluded`. Les 8 composantes (`lusr_component_history`, lues par Prestige, le profil et l'escouade), écrites jusque-là par la seule v1 donc figées depuis la bascule canonique, sont écrites par la v2 : même calcul, parité vérifiée valeur à valeur (275 valeurs identiques à 1e-12). Constat corrigé : la vue `match_skill_rank_latest` ne laissait pas la v1 masquer la v2 (`start_time` avant `written_at`), mais le graphe d'évolution (`match_skill_rank_latest_by_type`) servait bien les lignes v1. Mesure sur une copie à froid de JGtm (sauvegarde du 16/10) : composantes 7 137 lignes pour 1 028 matchs, dernière ligne du 18/07, 125 matchs notés sans composantes, puis après rejeu 8 011 lignes pour 1 153 matchs et 0 note sans composantes (rejeu complet 47 s en 24 rafales). Reste : le simulateur de formules `levelup backfill --compare-formulas` (TrueSkill v1, n'écrit rien) est conservé. |
| 2026-10-17 | **[ops/arrêt] Séquence d'arrêt du serveur bornée sous le délai de Docker** (plan `.ai/PLAN_BACKLOG_2026-10-16.md`, E3, branche `feat/backlog-2026-10-16`) : `stop_grace_period: 45s` sur `levelup` et `levelup-demo` ; budget global de 40 s depuis la demande d'arrêt (5 s gardés pour la fermeture des bases), chaque phase d'attente prend la plus petite de sa borne et du budget restant, y compris `watcherDaemon.Stop`, `BatchQueue.Close`, `workerWG.Wait` et le gate (jusque-là non bornés ou hors chronomètre) ; `slog.WarnContext` nommant la phase qui dépasse ; `duckdb.CloseAll` toujours atteint. Séquence extraite de `main()` dans `cmd/server/shutdown.go` + `shutdown_budget.go`. Tests : budget sous la grâce lue dans `docker-compose.yml` (rouge prouvé sans la clé), phases bloquées, ordre HTTP, recalculs, écrivains, fermeture. Commentaire périmé (`stop_timeout=20s`) supprimé. |
| 2026-10-17 | **[sécurité/suites] Code Xbox redemandé après une connexion réussie, et vérification de participation en 503** (plan `.ai/PLAN_BACKLOG_2026-10-16.md`, E2, branche `feat/backlog-2026-10-16`) : (1) `XboxLoginPage` ne relance plus `start` sur `attempt_not_found` une fois la tentative autorisée (la rotation de session au succès rend l'ancienne tentative introuvable) et le sondage `useDeviceFlowStatus` s'arrête dès `authorized`/`provisioned` (prédicat `isDeviceFlowSucceeded`, `setup/queries.ts`) ; test Vitest rouge avant le correctif (5 sondages au lieu de 1). (2) `RequireMatchParticipation` : cache borné (4 096 entrées, le plus ancien sort) des participations CONFIRMÉES seulement, clé titre + xuid + match, ni `false` ni erreur mémorisés, 503 fermé conservé sur échec non caché (`api/wire/participation_cache.go`, câblé dans `MatchParticipation`) ; tests : 2 appels = 1 requête, clé, borne, bout en bout avec le middleware. La première vérification d'un couple reste soumise au budget de lecture. |
| 2026-10-17 | **[ci/durée] CI raccourcie de 49 à ~16 min, mêmes tests** (plan `.ai/PLAN_BACKLOG_2026-10-16.md`, E1, branche `feat/backlog-2026-10-16`) : suite Go de couverture en 6 lots sur 6 machines, chacun en `-p 1` (définition unique `.github/ci/go-shards.txt`, dernier lot = complément calculé, `grammar` coupé en deux par une liste nommée de tests lourds `-run`/`-skip`), job de fusion (contrôle 356 paquets sur 356, gate baseline, profils additionnés, ratchet de couverture : 81,0 % identique) ; second passage `internal/sync` supprimé ; Vitest en 3 lots fusionnés ; `concurrency` (runs périmés annulés sur les branches de travail, groupe propre au run sur `main`/`feat/v75`) ; filtre de chemins sur les branches de travail (base `feat/v75`) ; `-timeout` revenu à 600 s ; `make gate-push` à deux voies (film sans DuckDB en parallèle). Garde-rails archlint : lots et partition, contournement du filtre, concurrency, voie film sans DuckDB. Run de preuve 38979844734 : 15 min 39 s. |
| 2026-10-16 | **[sécurité] Durcissement de la surface exposée + restes de la démo** — audit adversarial (9 axes, 12 P0 / 17 P1 / 36 P2, `.ai/AUDIT_SECURITE_SURFACE_EXPOSEE_2026-10-15.md`) puis plan `.ai/PLAN_SECURITE_2026-10-15.md` (lots L0-L11, deux rondes de revue adversariale, recette, E2E) ; fusionné dans `feat/v75` (c6a72cf02). nginx de prod durci le 2026-10-15/16. Reste au déploiement de la v7.5 : V-3 (`data/demo_films/`), V-5 (unité de l'ouvrier), V-6 (limites mémoire après mesure). |
| 2026-10-16 | **[lot backlog du 2026-10-15] Campagne dans les lecteurs, vue match « lecture en échec », intégrité du registre** (branche `feat/backlog-2026-10-15`, plan `.ai/PLAN_BACKLOG_2026-10-15.md`, E1 à E6) : lecteurs de la Campagne de `sync`/`ops`/`cmd` statués et garde étendu (E2), dettes Go (temps canonique, `GET /squads`) (E3), raisons `<source>_unavailable` distinctes de `<source>_empty` (E4), colonnes d'intégrité de `match_registry` (E5), deux rondes de revue adversariale corrigées (E6.2a, E6.2b). Reste : fusion dans `feat/v75` (geste du user) et les entrées ci-dessus. |
| 2026-09-27 | **[lot backlog du 2026-09-26, fusion 3] Dérogations mortes du lint inter-features, musique d'intro du rejeu, niveau des fanfares de fin** (branche `feat/backlog-2026-09-26`, plan `.ai/PLAN_BACKLOG_2026-09-26.md`, lots A3 et A4). **Lint (5)** : 16 dérogations mortes retirées ; le script échoue désormais sur toute dérogation qui ne sert plus ; README des graphes corrigé (12 wrappers, `FirstBloodLanes` dans `components/charts/`) ; lint ajouté au job web de la CI (`1ea900034`). **Musique d'intro (7)** : les 3 dernières secondes de la montée de la piste `402178411`, résolution R = 3,19 s calée sur le coup d'envoi, −18 LUFS, jouée au clic sur « Lecture » ou « Recommencer » depuis le préambule, jamais à la reprise, au saut ni dans l'export. Au premier « Lecture » après un rechargement, l'intro attend son décodage et part avec un décalage borné par la durée du préambule (`6bac387b8`, `eb57ec3c2`). **Niveau des fanfares (suite de l'item 11)** : les trois fanfares de fin mesuraient −25 LUFS au lieu des −18 prévus (réduction stéréo du 28/08 faite après la normalisation). Remises à −18 LUFS par un gain linéaire, avec un garde-rail de niveau sur les échantillons PCM (`e93c19ed9`). Fins multi-équipes (6) : `[!]`, faute de donnée (entrée ci-dessus). Écoutes de l'utilisateur après la fusion. |
| 2026-09-27 | **[lot backlog du 2026-09-26] Musique de fin et son du rejeu, classement mondial, découpeur SQL, index ART de `match_skill_rank`, garde-rail d'exclusion de la campagne, mode démo hermétique** (branche `feat/backlog-2026-09-26`, plan `.ai/PLAN_BACKLOG_2026-09-26.md`, un exécutant Opus à la fois, deux relectures adversariales puis une ronde 2 sur les corrections : 8 constats en ronde 1, dont 2 P1, et 0 P1 en ronde 2). **Musique de fin (11)** : la conclusion échappe au plafond de 8 voix dans la page, comme dans l'export (`soundOccupiesVoice`, `a373539d9`). **Son au rechargement (12)** : le lecteur s'ouvre au premier clic ou à la première touche, ou dès l'affichage si le document a déjà reçu un geste (`useAudioUnlock`, `89f311191`) ; la cause était la lecture automatique. **Classement mondial (1)** : `ErrDrainTimeout`, 3 essais espacés de 30 s sans nouveau scrape, premier tir à +2 min (`cd47936fc`). **Découpeur SQL (3)** : cœur unique `migration.ExecScriptContext`, 3 copies supprimées, garde-rail AST (`edd0054d5`). **Index de `match_skill_rank` (2)** : les trois `idx_msr_*` sont retirés par une migration et par un soin convergent rejoué à chaque ouverture (MSR et PSA) ; ratchet étendu à `sync/schema.go` ; sonde, `repair_msr_index` et `indexcheck` supprimés ; critère D-4 amendé (relatif, `EXPLAIN ANALYZE`) ; copies réelles vérifiées (`7e9ef7c15`, `0b6c11f73`). **Exclusion de la campagne (4)** : garde AST multi-racines (83 lecteurs, 21 dispenses datées), 12 lecteurs corrigés ; les chiffres Halo 5 qui comptaient la campagne changent (`a9e2192ac`, `c61c60761`). **Mode démo (8)** : tâches de fond coupées et chemins d'état redirigés par le SEUL serveur démo (`config.LoadServer`), 4 mutations en 403 `demo_mode_forbidden`. La preuve de bout en bout sur le vrai checkout donne 0 écriture (`3de419efe`, corrections B-C1 à B-C10). Écoutes de l'utilisateur (11, 12) faites après la fusion. Découvertes versées en 6 entrées ci-dessus. |
| 2026-09-19 | **[hygiène] Lot compare / armes / frontières** (branche `feat/hygiene-compare-armes`, 4 commits `e4238dea6`→`aa1a8dc2c`, exécuté par Opus sous pilotage) — **A** champ `filters` de `CompareRequest` retiré (Go + `types.ts` ; le fragment OpenAPI manuel ne le déclarait déjà pas, `FilterContextInput` conservé : 7 autres consommateurs). **B** repli « échantillon croisé » SUPPRIMÉ — mais la prémisse du backlog était fausse : la branche n'était pas morte, elle était FAUTIVE. `GetCrossMatchSample` n'excluait pas la campagne alors que `GetLocalStats` le fait ; mesuré sur copie du shared Halo 5 : pour un B présent uniquement en coop campagne, `GetLocalStats` rend 0 ligne et l'échantillon croisé rend 1 match (stats de campagne servies sous un service record matchmade). Exclusion alignée ⇒ branche morte par construction ⇒ retrait complet (service, repo, port + noop, `domain.CrossMatchSample`, `IsLocalSample`/`is_local_sample` régénéré par Huma, 2 tests, 8 lignes de baseline). **C** `buildTopWeapons` des séries temporelles délègue à `topWeaponKillRows` (départage sur le libellé) ; garde-rail étendu au motif `WeaponID <` sans propriétaire ; changement assumé : une arme sans libellé résolu n'est plus publiée (barre anonyme avant). **D** `SynthesisCards` → `components/ui/section-primitives.tsx`, `SynthesisWeaponAccuracyChart` → `components/charts/WeaponAccuracyChart.tsx` ; DEUX dérogations retirées (`timeseries=>synthesis` et `session-detail=>synthesis`) ; le ratchet du script compte les violations non déclarées (7/7, inchangé), pas les dérogations. Gates : Go build/vet/test 22 paquets, openapi-gen -check, types frais, tsc -b (cache purgé), eslint 0 erreur, lint inter-features, vitest 7984 tests — tous verts, rejoués par le pilote. Découvertes → 2 items backlog ci-dessus. |
| 2026-08-03 | **[ops/deps] Bump `echarts` 5.6.0 → 6.1.0** (CVE-2026-45249, XSS) — livré par `545b870de` (lot B4 echarts6, diff visuel joint). L'entrée « REPORTÉ » du backlog était restée après la livraison ; retirée le 2026-09-19. |
| 2026-07-26 | **[ops/prod] Écritures `app_settings.json` dans le conteneur (bind-mount fichier → rename EBUSY)** (v7.3, `branche feat/v7.3-notion-batch, lot backlog du 26/07`) — point d'écriture unique `internal/platform/atomicfile.WriteFile` : atomique (temp + rename) d'abord, repli **in-place** (truncate + un seul Write + fsync) quand le rename répond EBUSY ou que le répertoire parent refuse le temporaire. Toute AUTRE erreur de rename reste remontée (le repli couvre une contrainte d'environnement connue, pas un diagnostic manquant). Audit des écritures runtime de settings : **3 call sites**, tous migrés — `settings.Store.Save` (chemin de TOUS les toggles admin `PATCH /settings`, qui faisait un `os.WriteFile` nu, donc jamais atomique), `settings.Store.SaveTitleOverlay`, `notify.writeLastNotifiedVersion` (le bug d'origine : notif Discord « nouvelle version » rejouée à chaque redémarrage). Limite ASSUMÉE et documentée : le repli n'est pas atomique — risque borné (contenu déjà sérialisé en mémoire, un seul Write, fsync, fichiers reconstructibles). Garde-rail `archlint/no_bare_settings_write_test.go` (interdit `os.WriteFile`/`os.Rename` nus dans les packages writers de settings). Tests : rename EBUSY → repli, rename ENOSPC → erreur non masquée, temporaire impossible → repli, troncature, création. |
| 2026-07-26 | **[ops/notifs] Bruit WARN `app_release: emit` pour les comptes auth_only** (v7.3, `branche feat/v7.3-notion-batch, lot backlog du 26/07`) — filtre pur `appReleaseTargets` dans `internal/api/wire/notifications_boot.go` : les profils `auth_only` de `db_profiles.json` (5 en prod, `db_path` vide — ils n'existent que pour le pool de tokens) sont écartés AVANT toute résolution, avec une trace `DebugContext` groupée au lieu de 5 WARN par redémarrage. Découverte au passage, corrigée dans le même filtre : `LoadPlayers()` sans filtre de titre renvoie une entrée par (titre, joueur) alors que la notification est per-JOUEUR → un joueur déclaré sur 2 titres était traité deux fois ; déduplication par slug + ordre d'émission stable. 5 tests unitaires. |
| 2026-07-26 | **[archi/contrat] Reliquats V72-01 — clos** (v7.3, `branche feat/v7.3-notion-batch, lot backlog du 26/07`) — (a) **`securitySchemes`** : le contrat ne déclarait AUCUN mécanisme d'auth ; `sessionCookie` (apiKey/cookie) est désormais posé côté Go (`internal/api/openapi_security.go`) en lisant le nom du cookie à sa source unique `session.CookieName` — aucune exigence `security` par opération n'est ajoutée (une part de la surface est publique par conception ; l'inventaire route→garde reste le ratchet `bare_routes`, qui est exécutable). (b) **UI `/docs`** : `internal/api/openapi_docs.go`, montée sur le routeur RACINE (le `DocsPath` de Huma aurait enregistré la route une fois par sous-routeur, sous son préfixe) et gatée sur `IsProduction() **OU** DemoMode` — la démo est publique et ne pose pas `LEVELUP_ENV`, la gater sur la seule production l'aurait exposée. Sert le document VIVANT (`/docs/openapi.{json,yaml}`), CSP dédiée. (c) **`ApiError.details`** : `huma.SchemaTransformer` sur `humacore.apiError` restaure `oneOf: [object(additionalProperties true), array]`, perdu par le type Go `any` — corps runtime inchangé. **Statués `[!]`, non traités et pourquoi** : validation automatique des inputs + résolveurs cross-champs = CLOS par décision produit (contrat `RawBody`/400 CONSERVÉ, pas de bascule 422) ; les **7 descriptions de schéma racine** restent au fragment manuel — les rapatrier exigerait un `SchemaTransformer` sur 7 types de `internal/domain`, donc d'y importer `huma` alors que ce package n'a **aucune** dépendance externe aujourd'hui : coût architectural disproportionné pour 7 chaînes déjà présentes au contrat publié. |
| 2026-07-25 | **[data/objectifs] Stockpile + Extraction + VIP extraits** (v7.2.1, V721-02) — 18 colonnes ajoutées à `match_objective_stats` par une migration séparée incluant la recréation de `match_objective_stats_latest` (une vue `SELECT *` fige ses colonnes à la création : sans ça, les 18 colonnes restaient invisibles à TOUS les lecteurs). `VipStats` n'était même pas déclaré dans `StatsBundle`. Payloads réels capturés (P0, 10 matchs), fixtures committées, INSERT pur. Vérifié sur copie des vraies bases : 27 → 45 colonnes, 5757 lignes préservées. Restent `EliminationStats`/`InfectionStats`, bloqués faute de donnée (cf. backlog actif). |
| 2026-07-25 | **[feat/citations] 10 citations d'objectifs livrées** (v7.2.1, V721-03) — 10 retenues sur 19 (9 écartées par décision utilisateur). Paliers **calibrés sur données réelles**, ce qui a invalidé les paliers génériques initiaux : 4 de ces citations seraient restées au premier palier à vie (3, 1, 4 et 3 occurrences au total chez le meilleur joueur). Parité EN garantie par un garde-rail neuf (le nom EN n'en avait aucun, seule la description était couverte) + garde-rail d'existence des visuels sur disque. Seed vérifié : 88 → 98, 0 sans nom ni description EN. Visuels de 2 citations : bouche-trous provisoires, l'utilisateur les produit (blocage explicite avant déploiement). |
| 2026-07-25 | **[data/armes] Colonne résiduelle `weapons.name_fr` purgée** (v7.2.1, V721-05) — migration de reconstruction CTAS-swap idempotente (no-op sur une base déjà propre), clé primaire composite recréée, garde anti-perte sur le nombre de lignes. Vérifié sur copie de la vraie metadata : colonne absente, 84 lignes et PK intactes. |
| 2026-07-25 | **[archi/contrat] `DefaultStatus` + routes absentes du contrat** (v7.2.1, V721-04) — portée réelle **23 routes** et non 11 : 7 mentaient en silence (elles publiaient `200` pour un 201/202 sans que le fragment ne les corrige), 1 découverte en plus. Champ `Status` retiré partout où le statut est fixe (une seule source de vérité) + ratchet anti-retour. Et **35 routes** absentes du contrat (et non 38 : 3 étaient un artefact de comptage, deux implémentations déclarant le même contrat) montées via un harnais unique — un second harnais aurait rouvert le canal de dérive contrat/tests fermé en v7.2. Fragment manuel : 3291 → 3237 lignes. |
| 2026-07-25 | **[archi/data] Unifier la SOURCE des noms d'armes** — traductions centralisées dans un fichier TOML par titre keyé `weapon_key` (`config/titles/{halo_infinite,halo_5}/mappings/weapon_names.toml`), résolution via `weapon_key` dans `weapon_resolver.go` (:76-94), colonne `name_fr` retirée du CREATE du registre (`weapon_registry.go` :177-189), garde-rail `weapon_names_completeness_test.go`. Commits `cebe2fed9` (V72-06) + `95dd7b5e3`. |
| 2026-07-25 | **[feat/frags] Sunburst : niveau 2 pour la classe Grenade** — ventilation par type de grenade (`grenadeRoles`, `fragdist.go` :226-258) avec résidu « Autre grenade » et invariant Σ niveau 2 == kills classe respecté, i18n `frags.toml` :124-146. Commit `0eb523bb2` (V72-15.2). |
| 2026-07-25 | **[data/frags] H5 : Σ des classes du sunburst > total (double-comptage mêlée/assassinat)** — `buildGunFragClasses` retranche désormais `MechanicKills` des kills d'arme (`fragdist.go` :93-107), invariant Σ classes == total rétabli. Commit `0eb523bb2` (V72-15.3). |
| 2026-07-25 | **[archi/match-view] Retirer le fallback LIVE du Match view** — `GetMatchMeta` miss renvoie désormais `domain.ErrNotFound` (404 `match_not_found`) au lieu d'un fetch live API ; chaîne `tryCanonicalMatchView`/`WithDataAdapter`/`WithViewerGamertag`/`LoadMatchDetail` H5 entièrement supprimée (`match_view_service.go` :229-244). Commit `468154424` (V72-15.4). |
| 2026-07-18 | **[bug/h5] Fuite d'affichage campagne (287 matchs) — balayage complet** — cause : le mécanisme centralisé d'exclusion (`analysis.campaignExcludedVariantIDs` + fragments) n'était pas appliqué sur `Q5SharedHistory` (LISTE historique + compteur) ni `Q4/Q4MV` (filtres). Fix appliqué à **la source** : liste/filtres + relations/career hub (Q26/Q28/Q28Scoped/QRelationsCoreForm). Un **garde-rail structurel** (scan AST de tous les lecteurs `match_participants`/`mv_player_matches` + `xuid=?`) a révélé la fuite systémique → balayage complet des lecteurs restants (Q10 rencontres, Q19 communs, Q23/Q23b détail-match, Q29Heatmap/Q30Rival moments, Q29TopTeammates/Q30SquadShared/Q31/Q42 escouade). Exempts justifiés (allowlist) : mono-match (Q17/Q17b/Q26), filtré au call site (Q25Template, QRelationsPlayerWinRate), sur-lecture inoffensive (Q25MatchParticipants), code mort (Q30SquadMatches). Nouveau résolveur `resolveCampaignExclusionByMatchID` (sous-requête sans placeholder, sûr avant Sprintf). Tests : structural coverage + comportemental (alias/by-match-id/token). PAS de purge BDD (règle ART). Même session : code mort `Q30SquadMatches` supprimé (règle 7). |
| 2026-07-18 | **[sécurité/autZ] BOLA prestige objet-level clos** — découverte : `WithPlayerSlug` jamais câblé → les routes `{id}` pures (`GetChallenge`/`Update`/`Abandon`/`SuggestNext`/`GetArc`) + `ListMySquads`-avec-escouades échouaient en `ErrPlayerNotResolved`. Fix : middleware `prestigePlayerSlugCtx` stampe le slug du chemin (ownership-gardé) dans le contexte → répare la résolution ET clôt le BOLA par isolation player DB (défis/arcs perso en `stats.duckdb` du joueur du chemin → `{id}` étranger = 404). Défis d'escouade (shared_social, non isolés) : garde `assertMemberUser` ajoutée à `ListSquadChallenges` (`requestedBy` = slug du chemin). Tests service+handler+middleware. Même session : `JoinSquadChallenge` (rejoindre un défi d'escouade par `{id}`) reçoit la même garde d'appartenance. |
| 2026-07-17 | **[data/h5] Classification hors-arsenal + capture mécanique de kill** — 26 IDs classés (véhicule/tourelle/environnement/non-attribué/autres) au donut, exclus de l'insight coach (`16d2a09eb`) ; colonne `kill_kind` persistée à l'ingestion H5 pour cesser de jeter la mécanique (`c13e7f6bc`). Phase 2 (backfill + découpage « Capacités Spartan »/Corps-à-corps/Non-attribué) → backlog actif. Investigations : « Spartan » = bucket d'attribution sans arme (API officielle `weapon_type=Unknown`), concentré dans les modes mêlée. |
| 2026-07-17 | **[ux/relations] Lot G LIVRÉ** (revirement produit) — CSR/tier de la bête noire affiché en dégradation gracieuse (rien si absent). Justification d'abandon initiale corrigée : la donnée EST collectée (`ExtractAllSharedCSRRows` → `match_csrs_latest`) ; couverture nulle en dev car base sociale (1,8 % classé), non représentative des joueurs compétitifs cibles. Backend best-effort + chip front conditionnel + tests. Commit `8570af76a`. |
| 2026-07-17 | **[vérif finale + dettes] passe QA post-train** — audit logging (5 flux best-effort re-routés hors `general.log` vers modules dédiés + erreur avalée `decodeParams` comblée, `e6671ff89`) ; renforcement tests (branches best-effort/gardes nil, `058ba9486`/`d49a1734f`) ; dettes réglées : erreurs avalées coach/prestige/handlers, doc inversée `notifications/types.go`, schéma OpenAPI orphelin `BattlePassResponse` retiré, `SUBTIER_ROMAN` centralisé + garde-rail (3e copie trouvée → seuil franchi). Commits `d48da5912`/`556991069`/`6bb186e9b`/`3c975bde3`. Gates Go unit+intégration `-p 1` (114 ok/0 FAIL) + web (2258 tests) verts. |
| 2026-07-17 | **[data/h5] Inventaire complet des frags hors-arsenal** — `.ai/V7/H5_WEAPON_LONGTAIL_UNMAPPED.md` : 26 IDs non couverts (16 649 frags = 6,2 %), dont bucket « Spartan » 8 812 frags. Base pour une classification produit (catégories véhicule/tourelle/corps-à-corps/environnement à trancher). Commit `fbbfd809a`. |
| 2026-07-17 | **[train backlog] Purges code mort** — `squad/v2` purgé (knip croisé grep, 4 types vivants conservés) ; route `GET /players/{slug}/battlepass` supprimée après preuve de mort (service `GetBattlePass` conservé, consommé par season-pass). Commits `42b317cd8`, `c9bfa0e7d`. |
| 2026-07-17 | **[train backlog] Registre armes H5** — long-tail `v_weapon_kills` : 5 armes tenues mappées (3 grenades + golf club + oddball, ~18,4 k frags) ; véhicules/tourelles/UGC documentés hors-arsenal (item résiduel ouvert). Commit `9e0a8217d`. |
| 2026-07-17 | **[train backlog] i18n tiers + cache défis** — mapping tiers CSR centralisé dans `lib/skillTiers.ts` (+ garde-rail grep) ; défis locale-aware via locale dans les query keys `home`/`seasonPass`. Commits `45200bc4c`, `7fc82f575`. |
| 2026-07-17 | **[train backlog] Release CI (GHCR)** — livré (`7bb6a4257`) puis **RETIRÉ le 2026-07-18** sur décision user (pas d'intérêt) : retour au build local sur le VPS (comportement d'origine). |
| 2026-07-17 | **[train backlog] Prod** — `/health` expose `media_tooling` (sonde ffmpeg au boot, figée) ; `Cache-Control: public, max-age=31536000, immutable` sur les assets Vite hashés (côté Go). Commits `731c3934f`, `5f486b0b1`. |
| 2026-07-17 | **[chantier Relations UX] Plan 2026-07 CLOS** — lots A (tri colonnes), B (duels cliquables → match view via `matchIndexAtX`), C (toggle « jamais affrontés », défaut masqué, migrate store v2), F (CoreCards supprimées + lien Escouade), D (volet « Quoi de neuf » : `is_revived` SQL+DTO+strip front), E (notification `rival_encounter` post-sync par watermark, garde-fous 3/sync et 7 j), H régularisé `[~]`, G `[!]` (0 % couverture CSR rivaux). Plan archivé : `.ai/V7/PLAN_RELATIONS_UX_2026-07.md`. Commits `91649c01d`→`e38592ca9`. |
| 2026-07-17 | **[sécurité] BOLA acteur prestige clos** — 14 routes non-squad réconcilient l'acteur body/query avec la session (`authorizeActor`, 403 `player_forbidden`), garde-rail AST structurel anti-régression, tests 403+verts par endpoint. Commit `b8e97cb43`. Résiduel objet-level → item backlog. |
| 2026-07-17 | **[coach/prestige] V2.1 + V3 livrés** — colonne `source` (challenge + prestige_telemetry, migration `prestige_add_source_columns_v1`), source renseignée aux 4 origines (coach/user/pilot_mode/preset), endpoint diag `GET /_diag/prestige/telemetry/{slug}` ; analyseur `cmd/prestige-tuning-analyze` (recommandations grammaire, seuils 30 %/50, validation manuelle) ; canal Discord webhook pour notifications coach (opt-in `discord_notify_coach`, OFF par défaut, best-effort). Commits `709bac9ad`, `7fa643699`, `02797d4cb`, `2f30e180d`. |
| 2026-06 | **[POST-V7] Go-live / cutover Python → Go** — la branche Go est devenue `main` (code applicatif Python `src/` retiré). Cutover terminé ; reste 2 micro-tâches optionnelles (doc async, tuning janitor) listées en backlog actif. |
| 2026-06 | **[V8/Compare] CSR + CSR ATH (re-implémentation)** — cron CSR mondial autonome (`internal/scheduler/world_leaderboard_cron.go`, snapshots append-only), champs `HighestCSR*` restaurés dans `domain/compare.go` + `compare_service.go::applyCSRSummary`, exposés au front (`highest_csr`/`csr_alltime`). Commits `1693e6e1`, `aeaaffcd` (Phase 2 livrée). |
| 2026-06 | **[auth/unification] Consolidation ADR 0023 — read-path + watcher daemon → MultiUserTokenStore** — phases 3a/3b/3c livrées (`ab0ebefa`, `9eb9b738`, `5c7d87a8`) : tous les chemins lisent le multi-user store en priorité (fallback legacy), tracker auto-découvert au boot, migration boot-time `MigrateLegacyTokens`. Couvre les items backlog « PR 2.5b watcher migration », « read-path switch » et « auth/cleanup migration ». PRs A–D du lockdown auth livrées (`2e2357db`, `00cb920c`, `58bf9ec7`, `d9fcb178`). |
| 2026-06 | **[persist/safety] Gap [D] circuit breaker** — `internal/persist/queue.go:69-79` (`consecutiveFailures` + seuil 5, `ErrDrainCircuitBreaker` fail-fast ~1s) + 5 tests verts `queue_circuit_breaker_test.go`. Gaps restants (G/E + A/B optionnels) → [.ai/PLAN_PERSIST_ROBUSTNESS.md](.ai/PLAN_PERSIST_ROBUSTNESS.md). |
| 2026-06 | **[db-concurrency] leased-writer-enforcement intégré** — `internal/platform/dblease/` (`LeasedWriter`, `AcquireWriterCtx`) mergé et utilisé dans 10+ sites sync (commits `351798cc`, `65ca246d`, `f50f5753`). Conditions de déblocage backlog (build cgo, baseline, coordination) levées par le go-live. |
| 2026-05/06 | **[frontend/nav] Nettoyage `PlayerScopeNav`** — composant mort supprimé (cleanup knip `4b73b584`). Les constantes `PLAYER_*_NAV_ITEMS` sont conservées (toujours consommées par `pageTitle.ts`, donc non mortes). |
| 2026-05/06 | **[feedback-drawer] Drawer feedback + sync labels** — feature mergée sur `main` (`apps/web/src/features/feedback-drawer/`), `.github/workflows/sync-labels.yml` + `.github/labels.yml` présents. Labels GitHub confirmés présents (2026-06-09) → item entièrement clos. |
| 2026-05 | **[Go/PR 7] Sync Engine Migration to dblease** — 17 sites `AcquireLeaseCtx` → `AcquireWriterCtx` migrés (engine.go ×10, backfill_weapons.go ×1, citations_backfill.go ×2, friends_recompute.go ×2, session_recalc.go ×2). Deprecation comment sur legacy facade. |
| 2026-05 | **[Go/PR 4-6] Leased-Writer-Enforcement Foundation** — type `LeasedWriter` + interfaces `DBExecutor`/`DBWriter`, expvar metrics `dblease_acquire_total{kind,status}`, 26 tests intégration (burst, coordination, atomicity), corrections fixtures (global schema), CI workflow updates (go-lease-enforcement, go-baseline-tests jobs), preservation 1662 tests baseline. |
| 2026-05-24 | **[auth/unification] E.v2 — Pool.AddOrUpdateSource + periodic re-scan** (commit `4508df92`) : hot-add ou refresh d'un slot, goroutine main.go 15min tick, 5 tests TDD GREEN. |
| 2026-05-24 | **[auth/unification] PR 2.5b phase 1 — RefreshLoop.WithMultiUserMirror** (commit `157d80a8`) : mirror write legacy → multi-user, 3 tests TDD GREEN. |
| 2026-04-28 | **[Multi-titre] Migration `static/` vers arborescence title-scopée** — Plan finition multi-titres Phase 6 livré (branche `feat/multi-title-static-fs-rescope`, 6 commits). Couche 2 `internal/assets/static/` (35 tests) + couche 3 `TitleAssetURLAdapter` HI + ST_B stub + bascule des 5 callers Go (A1–A5, C1, F) + frontend `apps/web/src/lib/staticAssets.ts` (D1–D2) + big bang atomique (328 fichiers `git mv` + 180 rows UPDATE DB + flag flip + fixtures D3+D4) + cleanup Phase 6.6 (suppression flag + script jetable + dead branches). H5G/HI renames vers slugs canoniques longs. |
| 2026-04-10 | **Score de forme individuel + escouade** : `compute_form_score_history()` (Polars rolling avg_14 - avg_90), `load_full_performance_history()` (DB query), `plot_form_score_history()` (Plotly multi-lignes + fill). Intégré en tête de l'onglet Résumé (Timeseries) et avant "Taux de victoires vs historique" (Teammates). st.metric + graphe historique avec points session surlignés. |
| 2026-04-06 | **Discord i18n — assets résolus par ID dans l'embed** : `fetch_last_match_info()` remonte `map_id`/`playlist_id`/`pair_id`/`game_variant_id` + libellés EN bruts ; `src/utils/_discord_embed.py` résout désormais les traductions via `asset_translations` selon `discord_lang`, avec fallback unique vers l'anglais en BDD. Les colonnes `*_fr` de `v_match_full` ne sont plus utilisées dans ce flux. Tests ciblés : 138 passés (`test_discord_notifier.py`, `test_translations.py`, `test_delta_sync.py`). |
| 2026-03-30 | **i18n — Table `asset_translations` peuplée dans `metadata.duckdb`** : 9 674 traductions (698 assets × 14 langues BCP-47). Script `populate_asset_translations.py` réécrit avec `_build_version_id_cache()` (version_id SPNKr requis, `""` → 404), parallélisme `asyncio.gather` sur les 14 langues, reprise possible. |
| 2026-03-30 | **Fix critique — `v_match_full` sans traductions en prod** : `_try_attach_meta_for_views()` cherchait `meta.maps` (table absente en v6) → toujours `None` → vue créée sans JOINs i18n. Fix : vérifier `meta.asset_translations`. `_create_v_match_full()` : suppression des 4 JOINs legacy (`meta.maps/playlists/playlist_map_mode_pairs/game_variants`), 8 JOINs `asset_translations` (en-US + fr-FR × 4 types). Vue recréée en prod : "Starboard"→"Tribord", "The Pit"→"La fosse", etc. |
| 2026-03-30 | **Docs — Renommage ARCHITECTURE_V5 → V6** : `git mv` + mise à jour contenu (titre, version 6.3.0, `shared_matches_v2.duckdb`). §6 asset_translations ajouté dans la version FR. Toutes les références mises à jour : `CLAUDE.md`, `README.md`, `README_FR.md`, `FR/README.md`, `FR/COMMANDS.md`, `.ai/project_map.md`, `.ai/START_HERE.md`. |
| 2026-03-30 | **Docs — CHANGELOG 6.3.0** : entrées EN + FR documentant `asset_translations`, refonte `v_match_full` v6, fix `_try_attach_meta_for_views`. |
| 2026-03-30 | **Normalisation des labels de modes de jeu (v6.2.1)** : `resolve_display_mode()` dans `src/analysis/mode_display.py`, colonne `canonical_category` dans `mode_prefix_names`, 29 overrides dans `mode_pair_overrides`, `translate_pair_name` délégue au resolver, fichier plat de contrôle généré et validé. |
| 2026-03-30 | **Audit KDA locaux → `efficiency` (v6.2.1)** : sémantiques séparées — `p.kda` API conservé per-match, agrégats session/carte/cumul renommés `efficiency`/`session_efficiency` ; clés i18n `efficiency`/`efficacité` ajoutées ; 6 modules `src/analysis/` mis à jour (`cumulative.py`, `stats.py`, `_performance_relative.py`, `_performance_relative_helpers.py`, `_performance_session.py`, `stats.py` domain model). |
| 2026-03-27 | **Bug — `index_media.py --force` levait `ConstraintError: Duplicate key`** : quand `force_rescan=True`, `existing` était laissé vide `{}` → toutes les entrées considérées "nouvelles" → INSERT sur des clés déjà présentes. Fix : `existing` est toujours chargé depuis la DB ; `force_rescan` contourne uniquement le filtre delta `mtime`. Ré-indexation JGtm (73 médias) exécutée avec succès après fix. |
| 2026-03-26 | **Bug critique — `mv_player_matches` recalcule le KDA au lieu de lire la valeur API** : vue recréait `(kills + assists/3)/deaths` au lieu de `COALESCE(p.kda, fallback)`. Fix : détection dynamique `has_kda_col` (même pattern `has_enemy_mmr`) + génération SQL conditionnelle. |
| 2026-03-26 | **UX — Score d'équipe supérieur aux scores individuels (En-tête Page Coéquipiers)** : carte équipe n'affichait pas les bonus collectifs. Fix : `_render_compact_team_card` calcule `bonus = score - base_avg` et affiche `"moy. X (+Y collectif)"` quand > 0. |
| 2026-03-26 | **Bug — Colonne "Dernière rencontre" incohérente (Page Match · Encounters)** : SQL `MAX(start_time)` incluait le match courant et les matchs futurs. Fix : `filter_past` CTE + `_fetch_match_start_time` helper + guard `days = max(0, delta.days)` + colonne renommée "Précédente rencontre" + "1ère rencontre" pour les nouvelles têtes. |
| 2026-03-26 | **Bug annexe — `datetime.utcnow()` déprécié dans `career_lusr.py`** : remplacé par `datetime.now(timezone.utc).replace(tzinfo=None)`. |
| 2026-03-26 | **Bug — Médias mal rattachés aux matchs (décalage fuseau horaire)** : `epoch(capture_end_utc)` → `epoch(timezone('UTC', capture_end_utc))` dans `associate_with_matches()` + EXIF naïf ignoré (heure locale caméra, pas UTC). Ré-indexation requise (faite pour JGtm le 2026-03-27). |
| 2026-03-26 | **Bug RÉCURRENT CRITIQUE — Session escouade absente du graphe "Évolution de la performance"** : root cause A (fanout ouvrait shared en R/W → conflit handle Streamlit) fixée via Phase J (`shared_read_only=True` dans `_engine_fanout.py`). Fix défensif LEFT JOIN dans `_performance_squad._join_perf_frames()`. Les deux chemins de fix documentés dans l'audit sont implémentés. |
| 2026-03-26 | **Bug — Stats coéquipiers absentes (Page Teammates)** : résolu par le fix fanout R/O (Phase J). La root cause était identique au bug session escouade — fanout silencieux → PME coéquipier non créées. À revalider sur la prochaine session de jeu. |
| 2026-03-26 | **Bug annexe — `get_sync_metadata` lit mauvaise DB** : `SELECT last_sync_at FROM meta.sync_meta WHERE xuid=?` → `SELECT value FROM sync_meta WHERE key='last_sync_at'` dans la player DB. Fix commité dans `_diagnostic_repo.py` (Phase F). |
| 2026-03-26 | **Piste — Crashes silencieux (Page Coéquipiers · Top medals)** : source principale (connexions zombies fanout R/W) supprimée par Phase J. Si non récurrent → archivé. |
| 2026-03-21 | **Bug — Frags vs. détail armes (double-comptage melee)** : melee kills filmés attribués à l'arme tenue + `melee_kills` API → double-comptage. Fix : remainder `api_total - film_kills` dans 3 fichiers + `load_total_kills_for_player()` + 2 nouveaux tests. |
| 2026-03-21 | **UI — Graphe stats/min escouade : morts sous l'axe** — `plot_per_minute_timeseries` : deaths tracées en négatif (`dpm_neg`), `customdata[5]` = valeur absolue, `hover_dpm_neg` i18n, ticks Y absolus via `build_symmetric_abs_ticks` (extrait dans `src/visualization/_permin_helpers.py`). `timeseries.py` à exactement 500L. |
| 2026-03-21 | **Maintenance — Nettoyage dossier `scripts/`** — 10 scripts investigation → `scripts/investigation/` + README ; `cleanup_legacy_tables.py` + `cleanup_player_dbs_v5.py` → `scripts/_archive/` ; `.tmp.*` supprimés. |
| 2026-03-21 | **CI — Scripts exclus par `.gitignore`** — `check_code_size.py` → `enforce_size_limits.py` ; `check_imports.py` → `validate_imports.py` ; stubs `test_page_router_smoke.py` + `test_page_router_regressions.py` créés. Références mises à jour dans `ci.yml`, `.pre-commit-config.yaml`, `test_code_quality.py`. |
| 2026-03-21 | **UI — Notation de session escouade (Page Coéquipiers)** — `compute_squad_performance_score()` dans `src/analysis/_performance_squad.py` ; `SQUAD_GRADE_THRESHOLDS` + `resolve_squad_grade()` dans `performance_config.py` ; `render_squad_session_header()` + `_render_squad_score_block()` dans `src/ui/components/performance.py` ; 7 clés i18n `squad_grade_*` dans `src/ui/i18n/pages/teammates.py` ; bloc tendance K/D remplacé dans `teammates.py` ; 18 tests unitaires. |
| 2026-03-21 | **Perf — `_MAX_CONCURRENT_CHUNKS`** : déjà à 50 en production (`weapon_extraction_service.py`). Tâche obsolète — objectif déjà atteint. |
| 2026-03-19 | **Medal definitions en BDD** — table `medal_definitions` dans `metadata.duckdb` (167 médailles, DB-first + JSON-fallback). Migration, script population, CLI `--medal-metadata`, `MedalsMixin.load_medal_definitions()` / `get_medal_label()`, UI DB-first dans `medals.py`, 16 tests unitaires + 4 intégration. Orphan `citations_{fr,en}.json` supprimés. |
| 2026-03-19 | **Phase 8 — Couche centralisée médailles** (`medal_definitions.py`) — `src/data/medal_definitions.py` source canonique unique ; `_medal_data.py` thin re-export ; `medals.py` wrapper `@st.cache_data` délégant ; `_medals_repo.py` délègue. 3 chemins DB indépendants → 1. Fallbacks JSON applicatifs supprimés de `medals.py`. JSON `static/medals/*.json` conservés (source pour `populate_medal_metadata.py`). 51 tests passent. Commit `88d5cf0`. |
| 2026-03-19 | **Migration `b5>>4`** — `scan_fire_events_b5` implémenté, `fire_seq%n_players` supprimé, `map_b2_to_player`/`group_events_by_pi`/`POV_PLAYER_INDEX` retirés, 25 nouveaux tests — 4968 tests passent. Relancer `--force-weapons --all` pour re-extraire. |
| 2026-03-19 | **Backfill enrichissement** JGtm + Madina97294 — 8 matchs du 18 mars rattrapés (performance_score, sessions, citations) |
| 2026-03-19 | **Fix 11 — Fan-out multi-joueurs** : `FanoutEnrichmentMixin` (`_engine_fanout.py`) + branchement dans `engine.py` après `_detach_shared_from_player_conn()`. Résout le manquement d'enrichissement local pour les joueurs qui ne sync pas eux-mêmes. |
| 2026-03-19 | **Fix 10 — Performance vs historique** : `performance_score` ajouté à `COLUMNS_COMMON` + JOIN `player_match_enrichment` dans `load_matches_as_polars` + `df_history` propagé dans `WinLossService` |
| 2026-03-19 | **Fix 9 — Radar escouade** : `radar_squad_ids` sauvegardé avant filtre UI ; DFs historiques séparés (`radar_me_df/f1/f2/f3`) passés à `render_trio_synergy_radar` |
| 2026-03-19 | **Fix 8 — Heatmap monochrome** : `compute_map_breakdown` lit `performance_score` depuis la colonne quand présente (fallback percentile supprimé pour les joueurs enrichis) |
| 2026-03-19 | **Fix 7 — Performance vue 1 coéquipier** : `enrich_with_performance_score` appelé pour `me_df` et `friend_df` dans `render_single_teammate_view` |
| 2026-03-19 | **Fix 6 — MediaFileStorageError icônes rang** : images rang converties en data URI base64 dans `career.py` (IDs Streamlit éphémères éliminés) |
| 2026-03-19 | **Fix 5 — Joueurs fantômes** : `_is_ghost_player` requiert la présence des clés stat + filtre appliqué uniquement dans `filter_encounter_xuids` (scoreboard non filtré — joueurs légitimes à 0 stats conservés) |
| 2026-03-19 | **Fix 4 — ratio=kda** : `ratio = pl.col("kda").alias("ratio")` dans `_finalize_polars_df` + `p.kda AS ratio` dans `_query_teammate_shared_stats` — source unique API, plus de recalcul |
| 2026-03-19 | **Fix 3 — Matrice d'impact** : `.unique(maintain_order=True)` dans `friends_impact_heatmap.py` |
| 2026-03-19 | **Fix 2 — Bots bid(33.0)** : `get_bot_name()` appelé dans `_build_encounter_rows` avant le fallback `xuid[:8]` |
| 2026-03-19 | **Fix 1 — ColumnNotFoundError map_name** : `mr.map_name` ajouté au SELECT de `load_friend_match_details` + `_FRIEND_DF_EMPTY_SCHEMA` mis à jour |
| 2026-03-19 | **Bonus — `resolve_weapon_display` fusion avant DB** : la fusion map est appliquée (étape 0) avant le lookup `weapon_labels`, évitant que M392 Bandit / Fuel Rod SPNKr contournent leur regroupement canonique |
| 2026-03-16 | Audit post-V6 : `weapon_kills` bit sync + logging, `v_gamertag_lookup` systématique, `shared_matches_v2.duckdb` production, LEGACY SyncScope supprimés, 17 nouveaux tests — 4799 tests passent |
| 2026-03-16 | Sprint refactor : splits fonctions/modules >80/500L, `_teammates_trio_helpers`, `_match_relations`, `_roster_loader` helpers, `render_trio_charts` DRY |
| 2026-03-15 | Phase 3 v6 : migration complète `duckdb_read_only` UI → repo — 7 fichiers migrés, 17 tests + 9 tests antagonistes, 4764 tests passent |
| 2026-03-15 | Phase 2 v6 : `career`, `career_lusr`, `explorer` migrés + `CareerMixin` créé |
| 2026-03-15 | Migration last_match : requêtes directes → DuckDBRepository (`load_player_match_enrichment`, `is_abandoned_match`) — 12 tests |
| 2026-03-15 | Fixes Phase 1 v6 : `player_provisioning.py` bare connect, `cache_filters.py` `_get_connection()` privé, `multiplayer.py` dead code — 6 tests |
| 2026-03-15 | Couche résolution gamertag→XUID : `lookup_xuid_for_gamertag()` dans `src/utils/xuid.py` + `GamertagResolverMixin` — 9 fichiers migrés, 11 tests |
| 2026-03-15 | **v5.8 Wave 5** : nettoyage i18n playlists/modes obsolètes → `metadata.duckdb` |
| 2026-03-15 | **v5.8 Wave 4** : suppression `highlight_events.gamertag` + helper `resolve_medal_name` |
| 2026-03-15 | **v5.8 Wave 3** : nettoyage wrappers XUID + dead code outcomes → `Outcome` enum |
| 2026-03-15 | **v5.8 Wave 2** : migration consommateurs directs (gamertags, KV pairs, assets) |
| 2026-03-15 | **v5.8 Wave 1** : vues SQL `v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full` + `GamertagResolverMixin` |
| 2026-03-15 | **Fix weapon-parser** : corrélation globale — taux `fire_event` 15% → 95% |
| 2026-03-15 | **Navigation last_match** : boutons ◀/▶ entre matchs filtrés |
| 2026-03-13 | Couverture tests `migrations.py` (lacunes v5.5–v5.7) |
| 2026-03-13 | Conflit `shared_matches.duckdb` — sync depuis UI Streamlit |
| 2026-03-13 | **[UI] Heatmap performance par joueur × carte** — Page Teammates |
| 2026-03-13 | **[UI] Performance par carte vs historique** — vues escouade et joueur |
| 2026-03-08 | **Bug #0 : match invisible post-sync** — suppression `_filters_loaded_*` dans `_clear_app_caches()` |
| 2026-03-08 | **Perf UI** — vues matérialisées lazy, pagination SQL, projections fines, `@fragment_if_available` |
| 2026-03-28 | [v6.2] Badges Remontada / Débandade / Contre-Remontada — `DominanceFlag` 3-5, `comeback_analysis.py`, `comeback_backfill.py`, `--comeback-badges` CLI |
| 2026-03-28 | [v6.2] Unification vue coéquipier unique → vue escouade — `f2_xuid` optionnel, suppression `render_single_teammate_view` |
| 2026-03-28 | [v6.2] Graphe combiné Frags↑/Morts↓ — `plot_trio_kills_deaths()`, axe Y symétrique, `safe_chart_render()` |
