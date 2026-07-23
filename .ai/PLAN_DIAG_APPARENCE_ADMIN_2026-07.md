# PLAN — Page admin : traitement des retours utilisateur + diagnostic apparence Spartan ID — 2026-07

> Plan d'exécution pour agent. Contrat : skill `plan-execution` (ordre strict, une étape
> à la fois, gate par lot, statuts [x]/[~]/[!], zéro fix hors périmètre).
> Rédigé le 2026-07-08 (volet diagnostic apparence, après l'épisode « bannière JGtm
> figée »). RÉVISÉ le 2026-07-16 : intégration des retours utilisateur (Notion, Backlog
> LevelUp § « Rapports page Admin ») et re-vérification sur pièces de toutes les
> références. Toutes les décisions produit ci-dessous sont TRANCHÉES, ne pas les rouvrir.
> Le fichier garde son nom d'origine pour la traçabilité (référencé dans Notion,
> thought_log et la mémoire agent).

## Statut au 2026-07-16 (répond à la première question du backlog)

- Le volet diagnostic apparence n'est PAS implémenté (vérifié sur pièces : aucun
  `AppearanceDiagnosis` / `DiagnoseNameplate` / `appearance_diag*` côté Go, aucun panneau
  front). Le plan reste donc ACTIF dans `.ai/` — l'archivage vers `.ai/v7.1/` se fait à
  la clôture (item I5).
- L'ex-« dépendance dure » est LEVÉE : le resolver `(cfg, definitive)` de la branche
  `refactor/audits-2026-07` est en prod depuis le 2026-07-10 (vérifié :
  `spartan_nameplate_resolver.go:210` appelle bien `resolvePositiveEmblemCfg` →
  `(cfg, definitive)`). Le plan est exécutable immédiatement.
- La page admin a été livrée/refondue entre-temps (onglets État / Détections / Données /
  Sync / Système / Gestion — chantiers monitoring + audits). Les retours utilisateur du
  2026-07-16 portent sur cet existant → volet 1 ci-dessous, à exécuter AVANT le volet 2
  (il fixe le canon UI que le nouveau panneau réutilise).

## Contexte et objectif

**Volet 1 — retours page admin (2026-07-16)** : revue utilisateur complète de la page
admin (Notion § « Rapports page Admin », 24 retours, traçabilité en annexe). Constats :
défauts d'UI structurelle (titres de section DANS les cartes, cartes imbriquées, doublons
de titres), libellés obscurs (« Tas Go »), états trompeurs (Restic « introuvable » alors
que les sauvegardes ne sont simplement pas configurées), données absentes ou amnésiques
(post-sync vide après chaque reboot, tailles DB/WAL invisibles, « Vu pour la dernière
fois » toujours « - »), flux austères (note de détection via `window.prompt`), et résidus
de l'ère migration (panneau parité pointant `parity_check.py`, script qui N'EXISTE PLUS
dans le repo).

**Volet 2 — diagnostic apparence Spartan ID** (objectif d'origine, inchangé) : épisode
2026-07-08 — la bannière de JGtm ne « se mettait pas à jour ». Root cause : emblème
équipé `3806589-SpartanEmblem` (nouvelle génération) sans AUCUNE image de nameplate
publiée par Microsoft (absent de `mapping.json`, aucune cfg positive au CMS, 404 CDN).
L'app servait correctement la dernière bannière connue (champs d'apparence indépendants,
jamais vides) — mais rien ne rendait ce verdict visible : le diagnostic a nécessité
3 CLI (`diag_appearance`, `diag_emblem_mapping`, `diag_emblem_colors`) et les logs prod.
Objectif : depuis la page admin, un diagnostic à la demande, par joueur suivi, des
composants du Spartan ID (bannière, emblème, backdrop, service tag) avec un VERDICT
ACTIONNABLE par composant.

**Critère de succès global** :
- Volet 1 : chacun des 24 retours Notion est traité et statué ([x]/[~]/[!] — table de
  traçabilité en annexe) ; plus aucun titre de section dans une carte sur la page admin ;
  plus aucune référence Python ; suites Go + web vertes.
- Volet 2 : un clic « Diagnostiquer » sur un joueur suivi affiche en < 10 s le verdict
  des 4 composants avec explication FR/EN ; le cas « emblème sans nameplate upstream »
  affiche « Rien à faire — dernière bannière connue servie par design ».

## Décisions produit actées (2026-07-08 et 2026-07-16 — ne pas rediscuter)

Volet 2 (2026-07-08, inchangées) :
- Surface = page **admin**, introspection à la demande (bouton par joueur). PAS le
  monitoring ; aucune alerte/notification émise ; calcul à la demande, pas de polling ni
  de persistance nouvelle (l'historique passif reste `career_progression.last_fetch_status`).
- Périmètre v1 = **joueurs suivis** (`db_profiles.json`) uniquement — pas de tiers.
- Verdicts (enum fermé, 5 valeurs) :
  - `ok` — le live résout, la valeur servie est à jour ;
  - `upstream_missing` — définitif côté Microsoft (nameplate absente de mapping.json +
    aucune cfg positive) → « Rien à faire, dernière valeur connue servie par design ;
    se réparera seul si Microsoft publie l'image » ;
  - `transient` — échec réseau/HTTP/parse indéterminé → « Se répare seul au prochain
    refresh » ;
  - `auth_required` — tokens absents/morts (`auth_missing`, 401/403 owner) → « Action
    requise : réauthentification » (renvoyer vers le flux SSO existant) ;
  - `not_supported` — le titre ne fournit pas ce composant via ce pipeline (capability
    absente) → dégradation propre, jamais un faux « cassé ».
- La sémantique de LECTURE de l'apparence ne bouge PAS : champs 100 % indépendants,
  chacun sert sa dernière valeur non vide, jamais vide (verrouillé par tests le
  2026-07-08 — toute condition croisée bannière↔emblème est interdite).
- Pas de synthèse locale de bannière dans ce plan (piste future distincte).

Volet 1 (2026-07-16) :
- **Règles UI structurelles admin** (transverses) : un titre de section ne vit JAMAIS
  dans une carte (`SectionHeader` au-dessus du contenu) ; jamais de carte dans une
  carte ; états vides via `EmptyState*` avec sémantique visible d'un coup d'œil (succès
  = vert perceptible, pas un placeholder gris).
- Onglet « Gestion » réintégré dans le flux normal des onglets (suppression du
  `ml-auto`/séparateur qui l'excentre à droite — l'utilisateur ne le voyait pas).
- Panneau « Diagnostics d'instance » : le rapport de parité est SUPPRIMÉ (il référence
  `parity_check.py`, script inexistant — migration Python terminée ; doublon interne
  signalé). Guards médailles : règle de décision fermée au D2.
- `window.prompt` interdit dans l'app : la note des détections passe par un dialog
  in-app (`components/ui/alert-dialog.tsx` existe déjà).
- Sauvegarde : distinguer « Sauvegardes non configurées » (état neutre, backups OFF) de
  « Restic introuvable » (erreur, seulement si backups activés ET binaire absent).
- Amnésie post-boot (post-sync, dernières exécutions d'actions) : persistance LÉGÈRE en
  JSON hors DuckDB (aucune nouvelle écriture DuckDB — invariants anti-ART intouchés) ;
  chemin exposé par la config, jamais de `filepath.Join(..., "data", ...)` à la main.
- Qualité de données : libellés locale-agnostiques (« sans traduction » paramétré par la
  locale cible, défaut `fr` — on ne construit PAS d'autre locale aujourd'hui) + exemples
  concrets par ligne (jusqu'à 3 IDs de matchs cliquables, ouverture dans un nouvel onglet).
- `migrate-media-paths` : reste un CLI one-shot, PAS de bouton admin ; documenter (D4).
- Route du diagnostic apparence : `GET /admin/diag/appearance/{player_slug}` — convention
  actuelle (sous-routeur `/admin` + Huma). L'ancien préfixe `_admin/*` du plan initial
  est legacy (la route modèle `POST /_admin/progression/backfill` a été neutralisée par
  le lot sécurité S2) : NE PAS l'utiliser.
- Panneau apparence monté dans l'onglet **Données** de la page admin.

## Références de code (RE-VÉRIFIÉES le 2026-07-16 — re-vérifier sur pièces à chaque lot)

Volet 1 :

| Élément | Emplacement |
|---|---|
| Layout onglets admin (+ excentrage Gestion) | `apps/web/src/features/admin/AdminLayout.tsx:31-38` (TABS), `:93` (classe `ml-auto border-l pl-6`) |
| Composants canoniques admin (le canon) | `features/admin/components/SectionHeader.tsx`, `AdminTable.tsx`, `AdminKpi.tsx` ; `components/cards/KpiCard` ; `components/ui/empty-state.tsx` ; `components/ui/alert-dialog.tsx` |
| Anti-modèles à corriger (h2 DANS une Card) | `features/admin/sections/InvariantsSection.tsx:44`, `sections/DBContentionSection.tsx:20`, `sections/UsersSection.tsx:60`, `titles/AdminTitlesPage.tsx:42` |
| Onglet Données (montage) | `features/admin/data/AdminDataPage.tsx:16-35` ; qualité : `data-quality/AdminDataQualityPage.tsx`, `IssueSections.tsx`, `IssueTable.tsx:53-54` (colonnes génériques Matchs / Vu pour la dernière fois) |
| Convergence post-sync (in-memory, perdu au boot) | front `convergence/PostSyncTimeline.tsx`, `PostSyncMatrix.tsx` (montés `AdminConvergencePage.tsx:73/79`) ; Go `internal/scheduler/auto_sync.go:311` (`Snapshot()`), servi par `GET /admin/monitoring/scheduler` (`handlers/admin_monitoring.go`, « zéro I/O DuckDB ») |
| Détections | `features/admin/monitoring/DetectionsPanel.tsx:35` (boutons `:111-128` — `flex flex-wrap gap-1` ; `window.prompt` `:49`) ; `PATCH /admin/monitoring/detections/{fingerprint}` |
| Sauvegarde / Restic | front `system/AdminBackupSection.tsx` + `features/settings/i18n.ts:542` (`backupStatusResticMissing`) ; Go `pkg/duckdbbackup/restic.go:25` (`IsAvailable` = `exec.LookPath`), `scheduler.go:228` (`Status()`) |
| Ressources (« Tas Go », tailles DBs/WAL) | `system/ResourcesSection.tsx:70` (heap), `:90-123` (`DatabasesTable`) ; `GET /admin/monitoring/resources` |
| Logs bruts (pas de pagination : LIMITS 50/200/500 + troncature 8 Mio) | `features/admin/logs/AdminLogsPage.tsx:25,150` |
| Actions globales / rapides (sans horodatage persisté) | `data-quality/AdminDataQualityPage.tsx:79-84`, `overview/AdminQuickActions.tsx` ; `POST /admin/actions/*` (`handlers/admin_data_quality.go`, `admin_actions*.go`) |
| Couverture résolution d'arme | `overview/WeaponCoveragePanel.tsx` (onglet État) ; `GET /admin/monitoring/weapon-coverage` → `wire/registry_weapon_coverage.go:37-38` (vue `v_weapon_kills`, sentinelles 0/1/2 exclues, jointures `weapon_ids`/`weapons`/`weapon_labels`) |
| Panneau lab à purger | front `features/admin/lab/DiagnosticsPanel.tsx` + `lab/queries.ts` ; Go `internal/platform/lab/provider.go` (fixture `apps/go-api/tests/fixtures/parity_report.json`, swallow `return nil, nil` L71-74) ; route `GET /lab/diagnostics` protégée par `internal/api/lab_routes_mounted_test.go` (+ `server_apiv1.go:225`) |
| Gestion des titres | `titles/AdminTitlesPage.tsx` (`RegisterHelpNotice` `:91-97`), `titles/TitleDetailCards.tsx:49/109/143` (capabilities = comptes seuls) ; `GET /admin/titles*` |
| Modèle handler admin Huma | `handlers/admin_invariants.go:44-47` (`Mount` : `humacore.NewAPI(r)` + `huma.Get`, middleware RequireAuth/RequireAdmin + NoStore hérités du sous-routeur `/admin`) |

Volet 2 :

| Élément | Emplacement |
|---|---|
| Fetch appearance + résolution images | `internal/sync/haloclient/halo_client_career.go:70` (`GetSpartanCustomization`) + `resolveCustomizationImageURL` (halo_client.go) |
| Resolver nameplate (définitif/transitoire déjà distingué) | `internal/sync/haloclient/spartan_nameplate_resolver.go:177` (`ResolveNameplateURL`), `:263` (`resolvePositiveEmblemCfg` → `(cfg, definitive)`) |
| Statuts fetch persistés | `internal/service/career_live_partial.go` (`FetchStatus` : ok/api_empty/forbidden_403/auth_missing/failed) |
| Dernières valeurs servies | `internal/platform/duckdb/career_live_repo.go` (`LoadLastCareerRank`) |
| Tokens serveur par joueur | `*auth.MultiUserTokenStore` + `auth.RefreshHaloTokensViaStoreFirst` (ADR 0023 ; SISU seul provider depuis 2026-07-15) ; modèle d'usage : `cmd/diag_appearance/main.go` |
| PIÈGE xuid | `HaloXUID(ctx)` = compte CONNECTÉ, pas le joueur de la page (régression apparence corrigée le 2026-07-16, PR #63) — utiliser le xuid du PROFIL `db_profiles.json` |
| Rate budget par compte | `internal/platform/ratebudget` (`ForXUID`) — cf. `CareerFetcherFactoryFromTokens` |
| Capability H5 | `spartan_customizer` déclaré dans `config/titles/halo_5/title.toml:64` (bannière = rendu Spartan, pipeline `games/halo_5/livesync/appearance_persist.go`) |
| Garde-rail hosts | `internal/archlint/no_halowaypoint_literal_test.go` (aucun host en dur — passer par le resolver d'endpoints) |
| i18n admin | `apps/web/src/lib/i18n/manifests/admin.toml` (+ `common.toml` pour `common.admin.*`) → regen `node apps/web/scripts/build_i18n_manifests.mjs` ; settings : `features/settings/i18n.ts` |
| Query keys | `apps/web/src/lib/query/keys.ts` |
| OpenAPI (manuel) | `apps/go-api/api/openapi.yaml` → `make generate-types` |
| Cas de test canonique | emblème `Inventory/Spartan/Emblems/3806589-SpartanEmblem.json`, cfg `-1766636888` → `upstream_missing` (payloads réels dans le thought_log 2026-07-08) |

## Branche et livraison

- Branche : **`feat/admin-retours-diag`**, créée depuis `main` À JOUR.
  - DÉVIATION CONSIGNÉE (2026-07-23, demande utilisateur) : branche créée depuis
    `feat/title-slug-in-url` @ `7a79c7961` (chantier D7 livré, non mergé) dans le
    worktree dédié `LevelUp-wt-admin-retours` — le travail admin s'empile sur les
    routes D7 et suivra son intégration.
- 1 tâche = 1 branche, N commits — au moins 1 commit par lot. Prévenir l'utilisateur
  avant tout merge dans `main` (= deploy prod auto).
- Ordre : lots A → I, strictement séquentiels.
- **Gate front** (réutilisé par les lots A/B/C/D/G) :
  `cd apps/web && Remove-Item -Recurse -Force node_modules\.tmp ; npm run typecheck &&
  npm run lint && npm run test` — exit 0 (purge tsbuildinfo OBLIGATOIRE avant le
  typecheck de clôture).

## Volet 1 — Traitement des retours (lots A → D)

### Lot A — Structure UI transverse (front seul)

- [x] A1. Sortir les titres des cartes et supprimer les imbrications — périmètre FERMÉ,
      4 composants : `InvariantsSection.tsx` (« Intégrité des données » sous la section
      « Invariants » → un seul niveau, titre via `SectionHeader` hors carte),
      `DBContentionSection.tsx` (« Contention DB (sync) »), `UsersSection.tsx` +
      `management/AdminManagementPage.tsx` (doublon « Utilisateurs » : UN seul titre de
      section, la carte perd son h2), `titles/AdminTitlesPage.tsx` (« Titres » section
      vs « Gestion des titres » carte : garder un seul niveau de titre).
- [x] A2. `AdminLayout.tsx` : réintégrer « Gestion » dans le flux des onglets (retirer
      `ml-auto border-l pl-6` et le flag `management` s'il ne sert plus qu'à ça).
      Ordre final : État, Détections, Données, Sync, Système, Gestion.
- [x] A3. Sauvegarde (`AdminBackupSection.tsx`) : fusionner les deux blocs quasi vides
      en UNE section (`SectionHeader` « Sauvegarde ») + une carte statut/action compacte.
      (Réalisé dans `settings/BackupTab.tsx`, siège réel des deux cartes ; consommateur
      unique = `AdminBackupSection`, vérifié par grep.)
- [x] A4. « Sélections hors catalogue » : état « catalogue complet » rendu comme un
      succès visible (EmptyState avec icône/badge token success), plus un texte
      placeholder gris. Balayer les autres états vides de `IssueSections.tsx` pour la
      même sémantique (périmètre : ce fichier uniquement).
- [x] A5. Ajuster les tests vitest impactés par A1-A4 (pas de nouveau test de layout).
      (Constat : aucun test n'assertait les titres retirés — zéro adaptation nécessaire,
      suite complète verte sans modification.)

**Gate A** : gate front + vérification visuelle locale (`make dev`) sur liste fermée :
Données (Invariants), Système (Contention, Sauvegarde), Gestion (Utilisateurs, Titres) —
titres hors cartes, zéro doublon de titre, onglet Gestion visible sans scroll horizontal.

### Lot B — Libellés, flows et clarté (front + i18n, retouches DTO possibles)

- [x] B1. Détections (`DetectionsPanel.tsx`) : (a) aligner les boutons d'action (largeur
      de colonne suffisante / `whitespace-nowrap` — plus de chevauchement au wrap) ;
      (b) micro-copy expliquant chaque action (Reconnaître = pris en compte, reste
      visible ; Sourdine = masqué jusqu'à réapparition ; Résoudre = clos ; Rouvrir) via
      info-tooltip ou légende ; (c) remplacer `window.prompt` (`:49`) par un dialog
      in-app (alert-dialog + champ texte) qui explique l'usage de la note ; (d) FR/EN.
- [x] B2. « Tas Go » → libellé compréhensible : « Mémoire serveur (tas Go) » + courte
      description (« mémoire allouée par le processus du serveur ») ; FR/EN.
- [x] B3. Restic/Sauvegarde : vérifier ce que `Status()` expose (enabled vs available) ;
      (Constat : le DTO exposait déjà `enabled` + `available` → front seul, zéro Go.)
      étendre le DTO `/settings/backup/status` si nécessaire ; le front n'affiche
      « Restic introuvable » (erreur) que si sauvegardes ACTIVÉES et binaire absent ;
      sinon « Sauvegardes non configurées » (neutre) + CTA d'activation. FR/EN.
- [x] B4. « Enregistrer un 2e titre » → « Enregistrer un nouveau titre » ; contenu
      étoffé : pré-requis concrets (arborescence `config/titles/{slug}/`, `title.toml`,
      mappings TOML, commande CLI), lien doc. FR/EN, pas d'anglicisme.
- [x] B5. XUIDs orphelins : reformuler le hint « …lancer une convergence joueur si
      pressé » en formulation professionnelle (ex. « Résolution automatique au prochain
      cycle ; “Converger” force la résolution immédiate »). FR/EN.
- [x] B6. Gestion des titres : afficher la LISTE des capabilities déclarées par titre
      (Infinite ET H5), pas seulement les comptes (`TitleDetailCards.tsx:143` ; étendre
      le DTO `GET /admin/titles/{slug}` si les clés n'y figurent pas) ; carte
      « Diagnostic » : une ligne par vérification avec verdict explicite (OK/KO +
      explication courte) — narration lisible.

**Gate B** : gate front + regen i18n (`node apps/web/scripts/build_i18n_manifests.mjs`)
sans diff inattendu + `grep -rn "window.prompt" apps/web/src` → 0 occurrence + si DTO Go
touché : `cd apps/go-api && go test ./internal/api/... -count=1` et `make generate-types`
sans dérive.

### Lot C — Données manquantes et amnésie (Go + front)

- [x] C1. Post-sync : persister en fin de cycle un snapshot léger JSON (timeline +
      matrice par joueur + horodatage du cycle) HORS DuckDB, chemin via la config ;
      réhydrater au boot ; UI : afficher « Dernier cycle : <ts> » et distinguer « aucun
      cycle depuis le boot » de « aucune donnée connue ». AUCUNE écriture DuckDB.
- [x] C2. Actions globales : journal JSON des exécutions (par action : `last_run_at`,
      outcome, déclencheur) écrit par les handlers d'action ; affichage « Dernière
      exécution : … » sous chaque bouton (les horodatages in-memory existants — audit
      data, cycle sync — basculent sur ce journal pour survivre au reboot).
      (C2 statué [x] avec C1 — sous-lot C-I, voir journal.)
- [x] C3. XUIDs orphelins : (a) alimenter « Vu pour la dernière fois » côté Go (dernier
      match du xuid dans le registry) — aujourd'hui toujours « - » ; (b) pagination
      serveur (limit/offset sur l'endpoint issues) + pagination TanStack côté front.
- [x] C4. Logs bruts : « Charger plus » (curseur arrière sur le tail) en plus du
      sélecteur 50/200/500 ; conserver la troncature 8 Mio.
- [x] C5. Tailles DBs/WAL : diagnostiquer pourquoi `databases[]` de
      `/admin/monitoring/resources` n'affiche rien chez l'utilisateur (réponse vide ?
      chemins ? rendu conditionnel ?) et corriger — la table front existe déjà.
      (VERDICT : liste jamais vide par construction ; cause probable = RepoRoot mal
      résolu en prod → os.Stat ENOENT avalé → tailles nulles. Durci : sonde de la
      racine data → statut `db_inventory_status=unavailable` + log ERROR + état UI
      explicite ; erreurs stat non-ENOENT logguées. À CONFIRMER en prod au moment du
      déploiement : vérifier `LEVELUP_REPO_ROOT` du process serveur.)
- [ ] C6. Couverture résolution d'arme (H5) : extraire les `effective_weapon_id` classés
      inconnus sur les données réelles ; trancher : compléter le registre metadata H5 /
      corriger la jointure / documenter la cause légitime (ex. kills sans source de
      dégât dans le carnage) ; si résidu légitime, l'UI doit l'expliquer.
- [ ] C7. Modes sans traduction / Assets UUID bruts : (a) libellés locale-agnostiques
      (« Modes sans traduction (fr) » — locale paramétrée `?locale=`, défaut `fr`) ;
      (b) jusqu'à 3 IDs de matchs d'exemple cliquables par ligne (nouvel onglet vers la
      match view) pour décider traduire/résoudre en connaissance — extension du DTO
      issues + `IssueTable`.

**Gate C** : `cd apps/go-api && go test ./... -count=1` (a minima packages touchés :
api, service, scheduler, platform) + gate front + `make generate-types` sans dérive +
vérification visuelle locale (liste fermée : convergence, actions, XUIDs, logs,
ressources, couverture arme, exemples DQ).

### Lot D — Purge des résidus de migration (Go + front)

- [ ] D1. Rapport de parité : SUPPRIMER le bloc front (`DiagnosticsPanel.tsx` partie
      parité) ET le chargement Go (`provider.go` : parityPath/parityReport), les types
      domain associés et les clés i18n — `parity_check.py` n'existe plus dans le repo,
      la migration est terminée, et le panneau fait doublon avec son propre bloc
      fichier (signalé par l'utilisateur).
- [ ] D2. Guards médailles (« Entrées analysées 0 ») : re-vérifier sur pièces (requête
      locale sur les données prod via duckdb CLI). RÈGLE FERMÉE : si la validation
      produit des données réelles utiles → la re-héberger comme invariant dans
      `internal/sync/invariants` (elle apparaît alors dans « Intégrité des données ») ;
      sinon SUPPRIMER avec le reste. Défaut = suppression. Le swallow `return nil, nil`
      (`provider.go:71-74`) disparaît dans les deux cas.
- [ ] D3. Si D1+D2 = suppression totale : retirer la route `GET /lab/diagnostics`, le
      garde-fou `lab_routes_mounted_test.go` (retrait CONSCIENT, justifié dans le
      message de commit), `lab/queries.ts`, le port `LabProvider`, le package
      `internal/platform/lab` — 0 code mort, git garde l'historique.
- [ ] D4. `migrate-media-paths` : vérifier son statut (déjà exécuté en prod ? encore
      utile ?) ; s'il reste pertinent le documenter dans `docs/COMMANDS.md`, sinon
      consigner sa suppression candidate en Découvertes. Pas de surface admin.

**Gate D** : `cd apps/go-api && go test ./internal/api/... ./internal/platform/... -count=1`
+ `grep -rni "parity" apps/web/src apps/go-api/internal` → 0 hit applicatif (les hits
docs/historique se justifient un par un) + `grep -rn "parity_check" .` → 0 + gate front.

## Volet 2 — Diagnostic apparence Spartan ID (lots E → H)

### Lot E — haloclient : résolution diagnostique structurée (Go)

Objectif : exposer le POURQUOI de la résolution sans dupliquer la logique existante
(règle ≤ 2 copies : refactorer, ne pas copier).

- [ ] E1. Créer dans `internal/sync/haloclient/` un type exporté
      `AppearanceDiagnosis` : par composant {`ServedFrom` (live/carry), `ResolvedURL`,
      `Verdict` (enum 5 valeurs), `Detail` (clé technique : mapping_miss,
      no_positive_cfg, cms_http_error, no_banner_field, …)}.
- [ ] E2. Extraire de `ResolveNameplateURL` (`spartan_nameplate_resolver.go:177`) une
      fonction interne unique qui retourne (url, verdict, detail) ;
      `ResolveNameplateURL` devient un wrapper qui n'en garde que l'url (comportement
      byte-identique, logs inchangés). Ajouter une fonction exportée
      `DiagnoseNameplate(ctx, emblemPath, cfg, tokens…)` sur cette même interne.
      INTERDIT : dupliquer le fetch mapping/CMS.
- [ ] E3. Pendant emblème/backdrop : diagnostiquer via `resolveCustomizationImageURL`
      (succès/échec HTTP → ok/transient). Service tag : présent/absent du payload.
- [ ] E4. Tests unitaires haloclient : verdicts sur cache mapping seedé
      (`seedEmblemMappingCacheForTest`), cas `upstream_missing` (JSON CMS réel du cas
      3806589 en fixture), cas transient (HTTP KO), cas cfg positive directe.

**Gate E** : `cd apps/go-api && go test ./internal/sync/haloclient/... -count=1 &&
go vet ./internal/sync/...` — exit 0, zéro FAIL. Les `TestResolveNameplateURL_*`
existants passent SANS modification de leurs attentes.

### Lot F — service + endpoint admin (Go)

- [ ] F1. Service `internal/service/appearance_diag_service.go` : pour un `player_slug`
      → profil `db_profiles.json` ; **xuid = celui du PROFIL** (PIÈGE : jamais
      `HaloXUID(ctx)` = compte connecté — régression PR #63) ; tokens via
      `RefreshHaloTokensViaStoreFirst` (échec → verdict global `auth_required`, pas
      d'erreur 500) ; fetch `GetSpartanCustomization` + diagnostics du Lot E ; lecture
      des valeurs SERVIES via `LoadLastCareerRank` + dernier `last_fetch_status` ;
      assemblage DTO. Limiteur `ratebudget.ForXUID`. Aucune écriture DB.
- [ ] F2. Multi-titre : brancher sur capabilities (JAMAIS `slug ==`). Titre avec
      `spartan_customizer` (H5, `title.toml:64`) : bannière/emblème = pipeline
      appearance dédié → verdict `not_supported` pour la résolution nameplate +
      affichage des valeurs servies uniquement. Dégradation
      `ErrCapabilityNotSupported` → réponse partielle propre, pas de 500.
- [ ] F3. Handler Huma `GET /admin/diag/appearance/{player_slug}` : nouveau fichier
      `internal/api/handlers/` (zéro logique métier), montage façon
      `admin_invariants.go` (`Mount(r chi.Router)` + `humacore.NewAPI` + `huma.Get`)
      sur le sous-routeur `/admin` (RequireAuth/RequireAdmin + NoStore hérités) ;
      wiring `internal/api/wire/` ; OpenAPI manuel + DTO.
- [ ] F4. Logging : `slog.InfoContext` début/fin (player, titleSlug, duration),
      `ErrorContext` avec `"err"` sur échec non-trivial. Pas de token en clair.
- [ ] F5. Tests service avec mock fetcher (verdicts par scénario, y compris tokens
      absents et capability absente) + test handler `httptest` (200 nominal, 404 slug
      inconnu).

**Gate F** : `cd apps/go-api && go test ./internal/service/... ./internal/api/... -count=1
&& go vet ./...` exit 0 ; `grep -rn "slug ==" apps/go-api/internal/service/appearance_diag*`
→ vide ; route visible dans `api/openapi.yaml` ; `make generate-types` sans diff front
inattendu.

### Lot G — panneau admin (front)

- [ ] G1. Section « Diagnostic apparence Spartan » dans l'onglet **Données** :
      `SectionHeader` (titre HORS carte — règle du Lot A) + sélecteur joueur suivi +
      bouton « Diagnostiquer » (mutation à la demande, pas de refetch auto).
- [ ] G2. Rendu par composant (bannière/emblème/backdrop/service tag) : vignette de la
      valeur servie, badge verdict, explication dépliable (le POURQUOI + « quoi faire » :
      rien / attendre / réauthentifier). Composants canoniques (`AdminTable`, `KpiCard`,
      `EmptyState`) ; aucune couleur hex/Tailwind : tokens sémantiques uniquement
      (skill `color-tokens`).
- [ ] G3. i18n FR **et** EN dans `admin.toml` (labels verdicts + explications + CTA),
      regen manifests. Pas d'anglicisme côté FR.
- [ ] G4. Query key dans `lib/query/keys.ts` ; types depuis `generated.ts` (pas de
      types manuels dupliqués).
- [ ] G5. Test vitest du composant (au minimum : rendu des 5 verdicts depuis des
      fixtures, mock API).

**Gate G** : gate front (purge tsbuildinfo obligatoire).

### Lot H — vérification fonctionnelle du volet 2

- [ ] H1. Vérification manuelle dev local : diagnostiquer JGtm → bannière =
      `upstream_missing` avec l'explication « rien à faire » ; un joueur sain → `ok`.
      NOTE : si Microsoft a publié l'image entre-temps, le cas JGtm peut être devenu
      `ok` (le verdict vit) — la fixture 3806589 du test E4 reste la preuve du
      `upstream_missing` ; vérifier alors en manuel un `ok` + un `auth_required`
      (profil sans tokens).

**Gate H** : constat écrit dans ce plan (résultats des 2-3 diagnostics manuels).

## Lot I — Clôture globale

- [ ] I1. Skill `delivery-checklist` complet (aucun TODO introduit, fichiers ≤ 500 L,
      fonctions ≤ 80 L, parité i18n FR/EN).
- [ ] I2. Entrée `thought_log.md` + statuts finaux dans ce plan (aucun item sans statut,
      table de traçabilité annexe statuée).
- [ ] I3. CI de branche verte (`gh run list --branch feat/admin-retours-diag`).
- [ ] I4. Prévenir l'utilisateur pour le merge (deploy prod auto) + lui remettre la
      liste des retours Notion traités (le backlog Notion est SON carnet : ne pas y
      écrire).
- [ ] I5. Après merge : déplacer ce plan vers `.ai/v7.1/` (créer le dossier si besoin —
      demande utilisateur du 2026-07-16) + MAJ `.ai/project_map.md`.

## Annexe — Traçabilité retours Notion (2026-07-16) → items du plan

À statuer à la clôture, aucun retour sans statut.

| # | Retour (résumé) | Item(s) |
|---|---|---|
| 1 | Plan implémenté ? / archivage `.ai/v7.1` | En-tête « Statut » + I5 |
| 2 | Invariants / Intégrité : cases dans des cases, titres dans blocs | A1 |
| 3 | Contention DB (sync) : même défaut | A1 |
| 4 | Sauvegarde : deux grands blocs quasi vides | A3 |
| 5 | Gestion : deux blocs « Utilisateurs » | A1 |
| 6 | « Gestion des titres » mal organisé (titres/blocs/UI) | A1 + B6 |
| 7 | « Enregistrer un 2e titre » succinct et daté | B4 |
| 8 | Détections : boutons superposés, flow obscur, popup navigateur | B1 |
| 9 | « Tas Go » incompréhensible | B2 |
| 10 | Restic « introuvable » alors que non configuré | B3 |
| 11 | Paginer les logs bruts | C4 |
| 12 | Tailles des DBs et WAL invisibles | C5 |
| 13 | Timeline post-sync + post-sync dernier cycle vides | C1 |
| 14 | XUIDs orphelins : pagination, « Vu pour la dernière fois » = « - », wording | C3 + B5 |
| 15 | « Diagnostics d'instance » : UI horrible | D1-D3 (suppression) |
| 16 | Rapport de parité : Python inexistant + doublon | D1 |
| 17 | Guards médailles : « Entrées analysées 0 » | D2 |
| 18 | « Migrate-media-paths » absent | D4 |
| 19 | Modes sans traduction FR / Assets UUID : locale + exemples concrets | C7 |
| 20 | Actions globales : dernière exécution inconnue | C2 |
| 21 | Couverture arme H5 : inconnues inexpliquées | C6 |
| 22 | Onglet Gestion excentré, invisible | A2 |
| 23 | Capabilities d'Infinite absentes + « Diagnostic » peu narratif | B6 |
| 24 | Sélections hors catalogue : succès pas visible (placeholder) | A4 |

## Découvertes (à consigner ici, NE PAS traiter hors périmètre)

- [Lot A] 3 clés i18n orphelines après retrait des h2 : `common.admin.invariants_section`,
  `common.admin.users_section` (common.toml), `admin.titles.heading` (admin.toml).
  À purger au Lot B (regen manifests incluse dans son gate).
- [Lot A] `UsersSection.tsx` (~L30) utilise `confirm()` natif pour la suppression
  d'utilisateur — même famille que le `window.prompt` visé par B1 ; le plan ne couvre
  que les détections. À trancher au Lot B (extension B1 ou découverte assumée).
- [Lot A] C5 à re-qualifier : la table tailles DBs/WAL de `/admin/system` AFFICHE des
  données en dev local (12 bases, tailles + WAL, vérifié navigateur 2026-07-23) — le
  symptôme « invisible » est donc environnemental (prod ?) et non un bug de rendu.
- [Lot A] `TitleDetailCards` affiche DÉJÀ la liste détaillée des capabilities pour H5
  (vu navigateur) — B6 devra vérifier sur pièces ce qui manque réellement (retour #23
  visait surtout Infinite).
- [Lot B] Diagnostic B6 : les capabilities d'Infinite ne sont PAS absentes par code —
  les deux titres passent par `caps.GetCapabilities(slug)` (capabilities.toml chargés au
  boot, Infinite 17 clés / H5 16). Le symptôme utilisateur est environnemental au
  runtime (RepoRoot du process serveur / process obsolète) — à confirmer en prod via le
  log de boot `mappings_loaded kind=capabilities title_slug=halo_infinite`. Par ailleurs
  la carte de détail est mono-titre SUR SÉLECTION de ligne (défaut = 1re ligne = H5) :
  l'utilisateur n'a probablement jamais vu la carte Infinite faute de savoir la ligne
  cliquable. Le correctif front (capabilities du registre toujours rendues, issues du
  TitleDescriptor) couvre la visibilité dans tous les cas.
- [Lot B] `IntegrityBadge` (BackupTab) utilise les caractères `✓`/`⚠` préexistants —
  toléré (Unicode, pas emoji), non touché.
- [Lot B — REQUALIFIÉE] weapon-coverage : fausse piste « en-tête ignoré » — le canal
  de titre de cet endpoint est `?title=` (query param) et le panel front le passe
  correctement. État réel H5 constaté en live (2026-07-23) : 66 armes distinctes,
  89,4 % couvertes, 7 non résolues SEULEMENT, dont UNE dominante : `2457457776`
  (2353 frags — les 6 autres ≤ 11 frags). Le C6 = identifier/trancher ces 7 IDs
  (surtout la dominante), pas un bug de plomberie.

## Journal d'exécution

### [2026-07-23] Sous-lot C-II (C3+C4+C5) — CLOS (agent Opus + revue orchestrateur)

- C3 : last-seen des xuids orphelins par jointure `match_registry` +
  `MAX(SQLStartTimeCanonical("r"))` (fragment partagé, règle n°8) ; pagination serveur
  `limit/offset/total` rétrocompatible (`windowIssues`, jamais nil) ; front pagination
  TanStack `manualPagination` (pied de page piloté par le total, kinds sans pagination
  inchangés).
- C4 : curseur arrière = offset octet absolu (`BeforeOffset` exclusif, `NextOffset` =
  début de la ligne la plus ancienne, `HasMore`) ; franchit le budget 8 Mio page par
  page ; front `useInfiniteQuery` + garde anti-boucle (`nextLogCursor` pur, testé).
- C5 : VERDICT documenté au statut de l'item (RepoRoot / stat avalé) + durcissement
  livré (`db_inventory_status`, sonde racine data, logs structurés, EmptyState
  explicite).
- Déviation acceptée : +1 warning eslint (React Compiler skip sur `useReactTable`,
  inhérent au pattern TanStack prescrit par la règle 13, identique aux 7 tables
  existantes).
- Gate re-exécuté par l'orchestrateur : go test ops/api/domain exit 0, go vet 0,
  tsc 0, eslint 0 erreur, vitest 299 fichiers / 2632 passés ; matrice navigateur 9/9
  (pagination 1/6→2/6 avec offset émis, « Charger plus » concatène + before= émis,
  inventaire indisponible explicite — nouvelles surfaces stubées, chemin Go couvert
  par tests).

### [2026-07-23] Sous-lot C-I (C1+C2) — CLOS (agent Opus + revue orchestrateur)

- Architecture : package leaf `internal/platform/adminstate` (`FileStore` atomique
  temp+sync+rename, mutex, lecture tolérante ; `ActionJournal` typé thread-safe).
  Chemins via `PathResolver` : `data/global/admin_state/{post_sync_snapshot,action_journal}.json`.
  AUCUNE écriture DuckDB.
- C1 : persistance en fin de `storeCycleResult` (point unique tick+V2, hors verrou,
  best-effort loggé) ; `RehydrateFromDisk` au boot ; champ `since_boot` sur
  `SchedulerSnapshot` ; banner 3 états (« Dernier cycle » / « Cycle précédent (avant
  redémarrage) » daté / « Aucun cycle enregistré ») sur la page Données.
- C2 : journal écrit par la COUCHE SERVICE (defer nommé APRÈS les gardes busy/dry-run —
  seules les exécutions réelles, outcome ok/error, trigger tick/manual) ; 6 actions ;
  endpoint `GET /admin/actions/journal` (Huma, NoStore, openapi manuel) ;
  `<ActionLastRun>` sous chaque bouton d'action globale (État + Données), invalidation
  après action ; logique pure `actionJournalDisplay.ts` testée.
- Gate re-exécuté par l'orchestrateur : go test ./... complet exit 0, go vet 0 ;
  tsc 0 ; eslint 0 erreur ; vitest 298 fichiers / 2629 passés (+9) ; matrice navigateur
  7/7 (3 états C1 stubés + journal rempli/vide C2, 0 pageerror — les surfaces nouvelles
  stubées car l'API :8000 est l'ancien code ; les tests Go couvrent le chemin réel).
- Vérification e2e réelle de la réhydratation (reboot serveur avec données) : reportée
  au Lot H (environnement dev complet) — les tests unitaires scheduler/persistence la
  couvrent en attendant.

### [2026-07-23] Lot B — CLOS (agent Opus + revue orchestrateur)

- B1-B6 tous [x] + purge des 3 clés i18n orphelines du Lot A (B7 de fait). Front seul —
  aucun DTO Go touché (B3 : `enabled`/`available` déjà exposés ; B6 : `declared_capabilities`
  déjà exposé). `AlertDialog` étendu (slot `children` + `autoFocusConfirm`) plutôt qu'un
  2e shell modal (règle ≤ 2 copies) — rétrocompatible, tests existants inchangés.
  Sémantique W4 préservée (annuler = zéro mutation, vérifié navigateur PATCH intercepté).
  Clés retirées : `note_prompt`, `backupStatusDisabled` (renommée `backupStatusNotConfigured`).
  Contenus B4 vérifiés sur pièces (cmd_title.go, RUNBOOK_ADD_TITLE.md, vars BACKUP_ENABLED/
  RESTIC_REPOSITORY réelles).
- Gate re-exécuté par l'orchestrateur : regen i18n idempotente (18 manifests, 2589 clés) ;
  tsc -b 0 ; eslint 0 erreur (67 warnings = baseline) ; vitest 297 fichiers / 2620 passés ;
  grep window.prompt = 0.
- Matrice navigateur : légende 4 actions + dialog in-app (focus textarea, annulation sans
  PATCH), « Mémoire serveur (tas Go) » + tooltip, badge backup 3 états (réel « Restic
  introuvable » enabled&&!available ; stub !enabled → « Non configurées » + CTA concret),
  carte Infinite complète (15 chips registre + déclarées + feature-matrix), « Enregistrer
  un nouveau titre », hint XUIDs reformulé. 0 erreur console.

### [2026-07-23] Lot A — CLOS (agent Opus + revue orchestrateur)

- A1-A5 tous [x]. Fichiers : `empty-state.tsx` (prop `tone` success, token sémantique),
  `AdminLayout.tsx` (Gestion dans le flux + doc corrigée), `IssueSections.tsx`
  (tone=success sur les 4 états vides), `DBContentionSection.tsx` + `InvariantsSection.tsx`
  (cartes supprimées, SectionHeader hors carte), `UsersSection.tsx` + `AdminTitlesPage.tsx`
  (h2 doublons retirés), `BackupTab.tsx` (fusion en une carte, logique Restic intacte,
  en-tête de doc corrigé par l'orchestrateur), `settings/i18n.ts` (clé `backupTitle` retirée FR+EN).
- Gate re-exécuté par l'orchestrateur : tsc -b 0 (purge .tmp) ; eslint 0 erreur
  (67 warnings préexistants) ; vitest 297 fichiers / 2620 passés / 0 échec.
- Matrice navigateur Playwright (vite worktree :5174 + API réelle :8000, bootstrap
  intercepté is_admin) : 6 onglets dans le flux (Gestion dernier, 0 ml-auto), 0 h2 en
  carte sur data/system/management, un seul titre Utilisateurs, « Gestion des titres »
  disparu, 4 états vides succès (stub zéro) + 1 en données réelles, 0 erreur console.
  Captures revues visuellement (cohérence tokens/graisse OK).

## Protocole de reprise de session

Lire : ce plan (statuts des cases) → `thought_log.md` entrées `feat/admin-retours-diag`
→ `git log --oneline -10` sur la branche. Reprendre au premier item non statué du
premier lot non clos. Un lot est « clos » quand tous ses items sont statués ET son gate
est passé (code de sortie vérifié, pas seulement la sortie filtrée).

## Estimation

Lourd-moyen : ~2,5 à 3 journées agent. Volet 1 ≈ 1,5 j (beaucoup de petites surfaces,
C1/C6 sont les plus épais) ; volet 2 ≈ 1 j (E/F = gros du travail, G cadré par le canon
UI posé au Lot A).
