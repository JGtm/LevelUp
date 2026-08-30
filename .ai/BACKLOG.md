— Tâches et TODO centralisés

---

## 🔄 Aucune tâche en cours

---

## 📋 Backlog

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

**Cible** : sur un match 3+ équipes, écran de fin aux couleurs de l'équipe gagnante et
réplique de SA couleur. La correspondance `team_id` → couleur officielle existe déjà
(`apps/web/src/lib/halo/teamNames.ts`, TEAM_COLORS/TEAM_NAMES 0-8) — il reste à VÉRIFIER
sur pièces le mapping team_id ↔ couleur annoncée (au moins un match multi-équipes réel).
Extraction/normalisation : rejouer la recette du lot C (transcriptions et outillage
conservés sous `Desktop/Halo Infinite - Sons armes/_fin_partie/`). **Effort : S-M**
(gros du travail = l'écran multi-équipes, les sons suivent). Dépendance : aucune —
s'appuie sur l'overlay et le canal son de fin livrés en v7.5.

---

### [replay/sons] Musique d'intro au lancement du rejeu — queue du build-up, sans attente

Noté le 2026-08-27 (chantier rejeu 2D, suite du lot C sons de fin). La piste `402178411`
(16,00 s, pack `SFX/sb_130_mus_multiplayer_global.pck`, extraite et convertie sous
`Desktop/Halo Infinite - Sons armes/_fin_partie/mus_mp_global_wav/`) est très
probablement la musique d'INTRO de match (le build-up du countdown, écrit pour se
résoudre au coup d'envoi). Le cadrage v7.5 démarre la lecture pile au coup d'envoi :
il n'y a plus de place pour la jouer entière, et retarder le départ de 3-4 s a été
REFUSÉ (décision utilisateur 2026-08-27 — pas d'attente imposée, syndrome de l'intro
non skippable).

**Cible si repris** : ne garder que la QUEUE du build-up (les 2-3 dernières secondes,
celles qui se résolvent au coup d'envoi) et la jouer PAR-DESSUS les premières secondes
de lecture, à volume musique (−18 LUFS, comme les fanfares de fin) — zéro attente,
l'anticipation en plus. Coupe d'asset (recette du lot C : fondu, normalisation,
manifeste + garde-rail) + un déclenchement au départ de la lecture, symétrique du
déclenchement de fin du lot C. **Effort : S** (~20 min une fois le lot C en place).
À faire seulement si l'envie revient à l'usage — le statu quo (départ silencieux,
sons diégétiques seuls) est le choix par défaut assumé.

---

### [ops/demo] Hermétisme FICHIERS du mode démo — racine démo autonome

Noté le 2026-08-05 (vague 2, chantier fixture démo). Le mode démo est hermétique côté
RÉSEAU (`internal/platform/netguard`, ratchet de couverture) et côté ENTREPÔTS (bascule
inconditionnelle des 4 DB vers la fixture, `cmd/server/demo_paths.go` + ratchet). Restent
des fuites périphériques sur le `repoRoot` réel :

- cache d'assets écrit sous `{repoRoot}/data/cache/` (au lieu de la racine démo) ;
- fichiers de session HTTP sous `{repoRoot}/data/sessions/` ;
- lecture du `data/auth/` réel (tokens inutilisés depuis netguard, mais lus).

**Cible** : une racine démo totalement autonome — tout chemin d'écriture/lecture dérivé
de la racine fixture quand `LEVELUP_DEMO_MODE=true`. S'appuyer sur l'audit des chemins
qui existera pour la migration AppData Tauri (même besoin d'inventaire exhaustif).
**Risque actuel : faible** (écritures anodines, aucune donnée métier). **Effort : S-M.**

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

**Déblocage** : jouer une partie matchmaking de chacun de ces modes, puis la synchroniser.
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

### [ops/deps] Bump `echarts` 5.6.0 → 6.1.0 (alerte Dependabot moderate, CVE-2026-45249, XSS) — REPORTÉ

Noté le 2026-07-25. Dependabot signale une alerte moderate (CVE-2026-45249, XSS) sur `echarts`
5.6.0 (`apps/web/package.json`), corrigée en 6.1.0. **Non traité en v7.2.1** : bump MAJEUR du
moteur de tous les graphes de l'app (11 wrappers `apps/web/src/components/charts/` + pages
timeseries) — l'utilisateur a explicitement refusé le risque d'instabilité à ce stade.

**Reporté (décision utilisateur 26/07) — à re-planifier** (n'a PAS été pris dans le lot v7.3).
**Critère de go mesurable** : `make test-web` vert (suite charts/timeseries) +
`make check-types` vert + tournée visuelle des pages les plus denses en graphes (Timeseries,
Compare, Synthesis, Match view). **Effort : S-M** (bump + vérif, selon l'ampleur des breaking
changes 6.x).

---

### [POST-V7] Housekeeping post-cutover (optionnel, non bloquant)

> Le cutover Go (la branche Go est devenue `main`) est **terminé** — cf. archive « Récemment complété ».
> Reste 2 micro-tâches optionnelles, non bloquantes :
- [ ] Documenter le default async ON dans le README utilisateur
- [ ] Tuning du janitor (24h → 12h ?) si la latence WAL le justifie en prod

---

### [Migration] Cible desktop Tauri web-first, sans réécriture Rust métier — ⏸️ GARDÉ DE CÔTÉ

> ⏸️ **Gardé de côté** (2026-06-09) : conservé pour distribution desktop néophyte future. Note : le cutover Go étant fait, le « backend Python local packagé » ci-dessous doit se lire **backend Go local** — à re-cadrer si réactivé.

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
3. **Stockage local — migration vers AppData** : tout ce qui est aujourd'hui stocké sur disque à côté de l'app (DuckDB `data/titles/`, `data/auth/`, `data/global/`, `data/sessions/`, config `db_profiles.json`/`app_settings.json`/`.env.local`) ne peut pas rester dans le bundle applicatif Tauri — l'app doit pouvoir être mise à jour/réinstallée sans perdre ces données. Il faut cadrer un chemin utilisateur type `%APPDATA%/LevelUp/` (Windows) et équivalents autres OS, cohérent avec `PathResolver`, sans hardcoder de chemin machine. **À ce moment-là : faire un audit exhaustif de tout ce qui est écrit sur disque** (pas seulement `data/` — logs, caches, fichiers temporaires, médias indexés, tout chemin actuellement dérivé de `REPO_ROOT`) pour ne rien oublier dans la bascule vers AppData.

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
