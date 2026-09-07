# PLAN OUVERT — SORTIR LES DERNIERS LIBELLÉS FR/EN CODÉS EN DUR DU CODE GO

> Rédigé le 2026-09-07 par le superviseur du chantier « frise / point de vue », POUR UN AUTRE
> AGENT. Il est OUVERT : l'inventaire est daté, les options sont posées, l'ordre est proposé,
> les décisions qui appartiennent à l'utilisateur sont listées en §7 et ne sont PAS prises ici.
> À l'exécution : contrat `plan-execution` (ordre strict, statut par item, vérification sur
> pièces avant de coder — **doctrine RE-VÉRIFIER : tous les `fichier:ligne` ci-dessous sont
> une carte datée, pas une vérité ; re-grep avant chaque lot**), skills `arch-rules`,
> `canonical-types`, `frontend-patterns`, `adversarial-review` avant merge.

## 1. LE PROBLÈME

La règle du dépôt (CLAUDE.md, « Multi-titre ») : *libellés/assets/outcomes via
`TitleSemanticAdapter` + TOML `config/titles/{slug}/mappings/` — jamais de label FR/EN en dur
côté Go*. La réalité au 2026-09-07 : plusieurs familles de libellés produit vivent encore dans
des maps ou des littéraux Go, tantôt en français seul (bug visible sous UI anglaise), tantôt en
FR **et** EN (localisé, mais au mauvais endroit : un deuxième titre ne peut pas déclarer les
siens). Pour un même mot — « Victoire » — l'application a aujourd'hui **trois sources** : une
map FR (`match_history_service.go`), deux maps FR/EN (`home_locale.go`), et les TOML du titre
lus par le web (`useOutcomeMapping`).

S'y ajoute un défaut d'encodage transverse : **45 fichiers Go (hors tests) portent des
séquences mojibake « Ã »** (UTF-8 doublement encodé), la plupart dans des commentaires, mais
**13 occurrences dans des LITTÉRAUX de chaîne**, dont deux libellés d'issue de la page
d'accueil (`home_locale.go:56-57` : `"DÃ©faite"`, `"Ã‰galitÃ©"`, vérifié à l'octet :
`D 303 203 302 251 f a i t e`). Ce n'est pas un problème de libellé, c'est un problème de
fichier — et il se règle en premier, parce qu'il pollue toute relecture.

## 2. INVENTAIRE PAR FAMILLE — daté du 2026-09-07, à re-vérifier

Méthode : `grep -rn --include=*.go -E '"[^"]*[éèêàùçÉÈ][^"]*"' internal` (hors `_test.go`, hors
`migrations/`), puis classement à la main. Les LOGS `slog`, les sorties CLI (`cmd/*`), les
migrations et les données semées (`ops/seed_*`) sont HORS PÉRIMÈTRE (§2.H) : la règle vise ce
que l'utilisateur lit à l'écran.

### A. Libellés d'ISSUE de match

| Site | État | Lecteurs |
|---|---|---|
| `internal/service/outcome_label.go` (nouveau, lot `feat/outcome-cle-canonique`) : `resolveOutcomeLabel(ctx, outcomes, code)` = `Canonical(rawCode)` → `Get(key).Label(locale)` ; l'ancienne map FR `outcomeLabels` y est le REPLI daté (bascule 2026-09-07, retrait cible 2026-12-01, critère « 0 log `outcome_label: repli sur la map FR` sur 30 j ») | **LIVRÉ pour deux sites** : en-tête Match View (`applyMatchHeaderOutcomeLabel`, `match_view_data_loaders.go`) et lignes d'historique / Explorer / export CSV (`rowFormatters(ctx, …)` — sa closure était figée sur `"fr"`) | **CINQ appelants restent sur le repli `outcomeLabel(code)`** (correction du 07/09 : ils n'ont PAS « leur propre map », ils appellent le même helper) : `career_service_encounters.go:341`, `explorer_service_convert.go:85`, `match_history_explorer_options.go:94` (options du filtre « Résultat »), `match_view_builders_summary.go:88` (`PersonalResult.OutcomeLabel`), `match_view_builders_team.go:95` (lignes de scoreboard). Vérifié : côté web, seul `match-view/MatchHeader.card.tsx:362` lit `header.outcome_label` ; les `outcome_label` du scoreboard et du résumé ne sont PAS rendus → candidats « champ mort » (§7.4). **Reste de L1** : migrer ces cinq appels sur `resolveOutcomeLabel` (ils ont tous un `ctx` et un adapter à portée), puis retirer le repli à sa date. |
| `internal/analysis/home_locale.go:54` `homeOutcomeLabels` FR + `homeOutcomeLabelsEN` + `outcomeLabelForLocale` | FR+EN en Go, **2 valeurs FR en mojibake** | `domain.HomeMatchRow.OutcomeLabel` (`domain/home.go:245`) — **aucun lecteur trouvé dans `apps/web/src/features/home`** (grep du 2026-09-07) : champ probablement MORT → à vérifier puis supprimer (règle n° 7) plutôt que migrer |
| `internal/service/explorer_service_convert.go:60` `duelOutcomeLabel` | issue de DUEL (gagné/perdu un face-à-face), vocabulaire distinct | Explorer (`domain/explorer.go:37,434`) |

### B. Modes, playlists, catégories, portées

| Site | Contenu |
|---|---|
| `internal/games/halo_infinite/mode_category.go` (37 séquences mojibake dans les commentaires) | catégorie de mode FR (« Assassin »…), traduction `mode_name_tr` |
| `internal/games/halo_infinite/rankedplaylists/rankedplaylists.go:38` | `NameEN` / `NameFR` par playlist classée, en Go |
| `internal/service/match_history_service.go:51-53` `expTypePVPRanked = "PVP classé"`… | libellés d'expérience/portée servis aux filtres Explorer |
| `internal/games/halo_infinite/migrations/mode_playlist_fr.go` | données de migration (traductions FR) — DONNÉES, pas libellés vivants : hors périmètre sauf si la source de vérité doit migrer vers TOML |

Note : le skill `halo-modes` décrit la normalisation des modes ; `config/titles/halo_infinite/mappings/assets.toml` porte déjà des libellés d'assets (`useAssetLabel('game_variant', id)` côté web). La cible naturelle de B est ce fichier.

### C. Armes

| Site | Contenu |
|---|---|
| `internal/games/weapons/registry.go:317+` | table `{clé, "Sniper Rifle", "Fusil de précision"}` — EN/FR en Go |
| `internal/games/weapons/labels.go` | 12 littéraux FR |

Cible : `mappings/fields.toml` / `assets.toml` (décider : arme = asset), ou un `weapons.toml` par
titre si la table porte plus que des libellés (familles, icônes — cf. `frag-*` tokens web).

### D. Rangs / CSR

| Site | Contenu |
|---|---|
| `internal/service/compare_service.go:88` `csrUnrankedLabel = "Non classé"` | + commentaires « Général Platine VI » |
| `internal/games/mappings/ranks.go` | **existe déjà** avec `.Label(lang)` : la cible est là |

### E. Narratif, citations, prestige, synthèse

| Site | Contenu |
|---|---|
| `internal/analysis/narrative/encounter.go` (et voisins) | badges de rencontre « Allié+ », « Dur à cuire »… — le web a des tokens `narrative-encounter-*` (couleurs) : vérifier si les LIBELLÉS y sont déjà en FR/EN (`features/*/i18n.ts`) ; si oui le Go ne devrait servir que la clé |
| `internal/analysis/prestigetuning/render.go` (16) | rendu FR |
| `internal/games/halo_infinite/citations_custom.go` (1 littéral mojibake) | citations custom |
| `internal/service/synthesis_service.go` (1) / `synthesis_service_legacy.go` (5 littéraux mojibake) | synthèse coach — ADR 0028 « template synthesis » : les textes sont du CONTENU, pas des libellés ; décider s'ils relèvent de ce plan (§7) |
| `internal/service/teammates/teammates_service.go`, `stats_service.go`, `session_page_service.go:146` | 1 littéral mojibake chacun (messages d'erreur internes) |

### F. Messages d'erreur et descriptions d'API (handlers)

`handlers/settings.go` (25), `user_auth.go` (27), `prestige.go` (24), `groups.go` (23),
`auth_xbox_oauth.go` (20), `admin_*.go`, `media.go`, `setup.go`, `auth.go`… Deux natures
mêlées :
- **descriptions OpenAPI** (`humacore.Op("patchSettings", "Mettre à jour la configuration", …)`) —
  de la DOC, en FR, qui finit dans `openapi.yaml` ;
- **messages d'erreur** renvoyés au client en FR. Or `ApiError` porte déjà un `code` machine
  (`apps/web/src/lib/api/client.ts:13`, ex. `match_not_participant`) : le web PEUT localiser par
  code. Décision §7.

### G. Notifications Discord

`internal/notify/discord.go` (35 littéraux FR). `notify/labels.go` a déjà un chemin sémantique
(`semanticLabels.Outcome(key, lang)` sur les TOML) avec repli `haloLabels` FR/EN en Go. La
langue d'une notification est celle du destinataire — vérifier comment elle est choisie.

### H. HORS PÉRIMÈTRE (à ne pas « corriger »)

Logs `slog` en français (internes), sorties des CLI `cmd/*`, migrations, `ops/seed_*` (contenu
semé), commentaires — SAUF le mojibake, qui se corrige partout où il est (§4 L0).

## 3. OPTIONS D'ARCHITECTURE — ouvertes, une par famille

1. **Le backend sert la CLÉ canonique, le web localise** (`useFieldLabel` / `useOutcomeLabel` /
   `useAssetLabel` depuis `/api/v1/titles/{slug}/field-mappings`, ADR 0003/0011). Le plus propre,
   le plus multi-titre ; coût : contrat d'API par DTO (`*_label` → `*_key`), `openapi.yaml`,
   `make generate-types`, migration des lecteurs web (une vingtaine de fichiers pour
   `outcome_label` seul).
2. **Le backend garde `*_label` mais le localise depuis les TOML** (`TitleSemanticAdapter`,
   locale de la requête `ctxkeys.Locale(ctx)`), map Go = repli daté. Transitionnel, sans
   changement de contrat ; c'est ce que fait le lot issue en cours. Coût faible ; laisse un
   champ « présentation » dans l'API.
3. **Les deux** : 2 tout de suite pour éteindre le bug visible, 1 comme cible pour tout NOUVEAU
   DTO et pour les familles où les lecteurs web sont peu nombreux.

Recommandation du rédacteur (non tranchée) : option 3, famille par famille ; l'option 1 est
obligatoire pour F (les erreurs), où un libellé serveur n'a aucun sens multi-locale.

## 4. ORDRE PROPOSÉ — lots indépendants, chacun mergeable seul

**L0 — Balayage mojibake + garde-rail** (petit, transverse, à faire EN PREMIER).
- Réencoder les 45 fichiers (les littéraux d'abord : 13 occurrences, §1) ; vérifier à l'octet.
- Garde-rail `internal/archlint/no_mojibake_test.go` : aucun fichier `.go` du module ne
  contient la séquence `0xC3 0x83` (ni `Ã©`, `Ã¨`, `Ã `…) — modèle : les `no_*_test.go`
  du même dossier (allowlist vide, c'est le point).
- Chercher la CAUSE (un outil qui relit en ANSI et réécrit — mémoire du dépôt : les
  roundtrips PowerShell 5.1 `Get-Content | Set-Content` produisent exactement ça) et
  l'écrire dans le test.
- Gate : `go build ./...`, `go vet`, tests des packages touchés, garde vert.

**L1 — Issue de match** : **CLOS le 2026-09-07 (lot Q4, branche `feat/issue-cle-canonique`),
forme retenue = OPTION 1 (décision D5)**. Chaque DTO qui portait `outcome_label` porte
`outcome` = clé canonique (`win|loss|tie|dnf`, `omitempty`) ; `resolveOutcomeLabel`,
`outcomeLabel`, `outcomeLabels`, ses 4 constantes et le kill-switch ont disparu
(`internal/service/outcome_label.go` réécrit : `outcomeKey`/`outcomeText`/
`outcomeKeyFromHaloCode`) ; `home_locale.go` (`homeOutcomeLabels*`, `outcomeLabelForLocale`,
`outcomeLabel`) supprimés, remplacés par un résolveur injecté
(`RecentMatchesOptions.OutcomeText`) qui lit l'adapter du titre ; `domain.RecentMatchItem.
OutcomeLabel` supprimé (D4, 0 lecteur confirmé). `duelOutcomeLabel` (Explorer,
`explorer_service_convert.go:60`) vérifié DÉJÀ conforme (clé `win|loss|other` sur un champ
`Outcome`) — `[~]`, non touché. Seule exception texte : l'export CSV
(`handlers/match_history.go`, serveur, sans JS) reste servi en clair via
`MatchHistoryService.OutcomeText`, résolu depuis l'adapter du titre — jamais une map Go.
Web : lecteurs migrés sur `useOutcomeLabel`/`useOutcomeMapping`
(`MatchHeader.card.tsx:362`, seul lecteur réel de `header.outcome_label` — les autres
occurrences trouvées au grep du 07/09 étaient soit l'options du filtre Explorer (i18n
locale, non le backend), soit du code déjà migré en amont). Garde-rail final posé :
`internal/archlint/no_french_label_literal_test.go` (ratchet par fichier, 132 fichiers /
538 littéraux au 2026-09-07, périmètre `internal/{service,analysis,api/handlers,notify,
games}` hors migrations/tests/slog/fmt.Errorf).

**L2 — Accueil** (`home_locale.go`) : toutes les paires `labelForLocale(locale, fr, en)` vers
TOML + adapter ; l'accueil est le pilote historique de l'ADR 0011 (labels i18n hors canonical),
il y a donc déjà une frontière à respecter — lire `home_service.go:36-42` et l'ADR avant.

**L2 — CLOS le 2026-09-07 (lot M5 première moitié, branche `feat/libelles-accueil-rangs`)** —
re-vérifié sur pièces (doctrine RE-VÉRIFIER : la carte datait déjà) : `home_locale.go` ne
portait déjà PLUS aucune paire `labelForLocale(locale, fr, en)` avec des littéraux FR/EN en
dur — les maps `homeOutcomeLabels*`/`outcomeLabelForLocale` avaient disparu avec Q4 (L1). Les
appels restants de `labelForLocale`/`labelFR` (dans `home_canonical*.go`) résolvent des noms
d'ASSET dynamiques (map/mode/playlist) depuis `canonical.AssetReference.Labels`, peuplés à la
sync depuis `metadata.asset_translations` / `mode_name_tr` (DB) — ce n'est PAS un littéral Go,
c'est de la donnée par-match localisée par le TITRE (famille L3, hors périmètre de ce lot).
`buildHomeNarrativeBadges` sert déjà des CLÉS (`"dominant"`, `"humiliation"`…), pas du texte.
Seul point réellement en dur trouvé : `RecentMatchItem.Title`, composite Go
`"<mot d'issue> · <carte>"` assemblé dans `home_canonical_recent.go` via
`RecentMatchesOptions.OutcomeText` (résolveur injecté depuis l'adapter du titre, D5 — donc pas
un mot FR en dur, mais un TEXTE PRÉ-ASSEMBLÉ côté Go, contraire à l'option 1). Vérifié qu'il
n'a qu'UN seul lecteur web (`match-card.tsx::buildMatchHeading`), et seulement en DERNIER
repli quand `map_ui` ET `mode_ui` manquent tous les deux (`MapUI`/`ModeUI` déjà servis à part
et couvrant le cas nominal) — pas une refonte de contrat nécessaire. **Supprimé** :
`domain.RecentMatchItem.Title`, `analysis.RecentMatchesOptions.OutcomeText`, la construction
`label`/`title` dans `home_canonical_recent.go`, `outcomes := outcomesOf(s.semantic)` +
`OutcomeText: func(...)` dans `home_service.go` (règle 7, 0 code mort — git garde
l'historique). Le web compose désormais ce repli depuis la même clé i18n que le placeholder
d'image (`common.match_card.map_unknown`, "Map inconnue"/"Unknown map") ; `OutcomeTone`
(déjà la clé canonique `win|loss|tie|dnf`, cf. `outcomes.toml`) reste disponible pour un futur
lecteur via `useOutcomeLabel`. `openapi.yaml` ne modélise pas `HomePageResponse` (TODO Sprint
32 préexistant) : 0 impact contrat, `generate-types` sans diff. Tests Go et web mis à jour
(gates verts). Pages à vérifier à l'écran : Accueil (tuiles de matchs récents, FR et EN).

**L3 — Modes / playlists / catégories / portées** (B) → `assets.toml` ; ratchet
`no_bare_resolve_mode_ui_test.go` existe déjà dans `archlint` : l'étendre. **HORS PÉRIMÈTRE du
lot M5 première moitié (2026-09-07)** — périmètre fermé à L2+L5 par consigne d'exécution ;
reste `[ ]` pour la seconde moitié de M5.

**L4 — Armes** (C). **HORS PÉRIMÈTRE du lot M5 première moitié (2026-09-07)** — reste `[ ]`
pour la seconde moitié de M5.

**L5 — Rangs** (D, cible existante `mappings/ranks.go`).

**L5 — CLOS le 2026-09-07 (lot M5 première moitié, branche `feat/libelles-accueil-rangs`)** —
**découverte majeure, doctrine RE-VÉRIFIER confirmée** : `internal/games/mappings/ranks.go`
(`RankCatalog`, `.Label`/`.FullLabel`) est le catalogue du rang de CARRIÈRE (XP, « Général
Platine VI »), **PAS** le tier CSR (Bronze..Onyx + sous-palier). La carte du plan associait à
tort ce fichier à `csrUnrankedLabel` — ce ne sont pas le même système de rang. Traité au
périmètre exact demandé : `compare_service.go::csrUnrankedLabel = "Non classé"` → clé
canonique `"unranked"` (D5) ; consommé par `csrSummary.currentLabel`/`allTimeLabel` →
`domain.NormalizedPlayerStats.HighestCSRLabel`/`HighestCSRAllTimeLabel` →
`CompareMetricRow.DisplayA`/`DisplayB` (métriques `csr`/`csr_alltime` de la page Comparaison).
Le web (`ComparePage.tsx::formatMetricValue`) passait déjà `display` à
`lib/skillTiers.ts::localizeTierLabel` — mécanisme CLIENT-SIDE existant (pas de TOML) qui
localise déjà tous les noms de palier CSR (Bronze/Or/Platine…) : la clé `"unranked"` y est
ajoutée (`TIER_NAME_BY_KEY['unranked'] = {fr:'Non classé', en:'Unranked'}`), réutilisant LE
MÊME canal plutôt que d'en ouvrir un nouveau (cohérent avec la consigne « jamais un nouveau
canal », et avec le fait que les noms de palier CSR eux-mêmes ne passent pas par
`/field-mappings` aujourd'hui). Le ratchet `no_french_label_literal_test.go` baisse de 3 → 2
pour `compare_service.go`. Commentaires « Général Platine VI » : laissés (ce sont des
commentaires, consigne explicite). **Non traité, consigné en découverte (§9)** : `csrRankLabel`
(même fichier) formate encore le TIER en clair (« Platine IV ») via `skillTierLabel` — doublon
avec `home_canonical_skill.go::csrTierENtoFR` et `sync/csr_writes.go::tierENtoFR` (3 copies,
règle 6 dépassée) ; `sync/csr_writes.go` va plus loin en PERSISTANT le libellé FR dans
`match_skill_rank.tier_label` (table append-only ADR 0026) — migration hors format de ce lot.
`match_history_explorer_options.go::skillTierLabel`/`perfTierLabel` (déjà repérés §10, Q4) :
`.Label` y est mort (le web n'utilise que `.value`/`.count`, `ExplorerPage.filterOptions.ts`)
mais touchent aussi une famille distincte (paliers de perf, pas CSR) — laissés pour une
décision de périmètre séparée. Pages à vérifier à l'écran : Comparaison de joueurs (ligne CSR
d'un joueur non classé, FR et EN).

**L6 — Narratif / prestige / synthèse** (E) : APRÈS décision §7 (contenu ou libellé).

**L7 — Erreurs API** (F) : APRÈS décision §7 ; si option 1 : `code` seul côté Go + table
`code → texte` FR/EN côté web (`lib/api/errors.i18n.ts`), garde-rail « un message d'erreur
sans code est refusé ».

**L8 — Discord** (G) : langue du destinataire + `semanticLabels` partout, repli `haloLabels`
retiré avec date.

**Garde-rail FINAL, transverse** : `internal/archlint/no_french_label_literal_test.go` —
ratchet sur le nombre de littéraux accentués dans `internal/{service,analysis,api/handlers,
notify,games}` hors tests, allowlist NOMMÉE et DATÉE par fichier restant, compteur qui ne peut
que décroître (patron : les ratchets lint du dépôt). Poser le ratchet dès L0 avec le compte du
jour, le faire baisser lot par lot.

## 5. GATES PAR LOT

- Go : `gofmt`, `go vet`, `go test` des packages touchés (CGO : `$env:Path =
  "C:\msys64\ucrt64\bin;$env:Path"; $env:CGO_ENABLED="1"; $env:CC=…gcc.exe`), jamais
  `go test ./...` global (tests `himap` = jeu installé, dizaines de minutes) ; `make go-api-lint`
  (ratchet) avant push.
- Contrat : si `openapi.yaml` change → `make generate-types` et le fichier généré dans le même
  commit ; parité FR/EN des TOML vérifiée par le loader (`loader_outcomes.go` valide déjà).
- Web : `npm run typecheck` (purger `node_modules/.tmp`), vitest complet (`--pool=forks`, hors
  sandbox), lint = baseline.
- Multi-titre : aucune comparaison de slug (`no_slug_comparison_test.go`), dégradation
  `ErrCapabilityNotSupported` si un titre ne déclare pas la famille.
- Visuel : parité FR/EN à l'écran par l'utilisateur, sur les pages touchées (Match View,
  accueil, Explorer, Carrière…), avant/après.
- Revue adversariale (skill) sur chaque lot qui touche un contrat d'API.

## 6. CE QUI EXISTE DÉJÀ — à réutiliser, pas à réinventer

`internal/games/mappings/{outcomes,ranks,assets,types}.go` (`.Label(lang)`),
`internal/games/adapter.go` (`Outcomes()`, `TitleSemanticAdapter`), `notify/labels.go`
(`semanticLabels` = l'exemple d'usage), `handlers/field_mappings.go` (l'endpoint que le web
lit), `apps/web/src/lib/i18n/fieldMappings.ts` (`useFieldLabel`, `useOutcomeLabel`,
`useOutcomeMapping`, `useAssetLabel`), `ctxkeys.Locale(ctx)`, ADR 0003 (i18n manifests +
lint), ADR 0011 (frontière canonical / semantic / asset), ADR 0028 (templates de synthèse),
`.ai/PLAN_MULTITITRE_INDEX.md` et `PLAN_MULTITITRE_PERIPHERY.md` (registre MT-xx — vérifier
si une entrée couvre déjà une famille ci-dessus avant d'en créer une).

## 7. DÉCISIONS QUI APPARTIENNENT À L'UTILISATEUR — à prendre AVANT d'exécuter le lot concerné

1. **Erreurs API (F)** : localiser côté web par `code` (option 1) ou servir un message localisé
   par la locale de requête (option 2) ? Et les DESCRIPTIONS OpenAPI en français sont-elles
   acceptables (c'est de la doc) ?
2. **Narratif / synthèse / citations (E)** : sont-ce des LIBELLÉS (→ ce plan) ou du CONTENU
   (→ ADR 0028, templates par titre, hors plan) ?
3. **Discord (G)** : langue du destinataire, du compte, ou de l'instance ?
4. **Champs `*_label` sans lecteur web** (ex. `HomeMatchRow.OutcomeLabel`) : supprimer (règle
   n° 7) ou conserver pour des clients tiers ?
5. **Cible finale par famille** : clé canonique (option 1) ou libellé localisé serveur
   (option 2) — et le calendrier de retrait des replis FR.
6. **Périmètre du mojibake** : sources Go seulement, ou aussi `docs/`, `.ai/`, TOML ?

## 8. REPRISE ET CONTEXTE

- Lot issue en cours : branche `feat/outcome-cle-canonique` (worktree `LevelUp-wt-outcome-cle`),
  empilée sur `feat/v75-frise-pov` — son rapport liste les autres producteurs d'`outcome_label`.
- Registre : `.ai/V7.5/REGISTRE_REPORTS.md`, entrées du 2026-09-07 (« `header.outcome_label`
  en FR codé en dur » + décision utilisateur « clé canonique côté Go »).
- Journal : `.ai/thought_log.md`, entrée du 2026-09-06/07 « Frise du rejeu ».
- Mémoire agent : `project_frise_point_de_vue_chantier`, `project_multititre_gap_register`.

## 9. DÉCOUVERTES DE L'EXÉCUTION (Q3, 2026-09-07)

> Le brief d'exécution de Q3 renvoyait à un « §3 Découvertes » qui n'existe pas dans ce
> document (§3 est occupé par « OPTIONS D'ARCHITECTURE » — doctrine RE-VÉRIFIER). Section
> ajoutée ici en fin de fichier pour ne pas perturber la numérotation existante. Consignées
> SANS être traitées, conformément au contrat plan-execution (règle 7).

- 2026-09-07 ; `internal/service/home_service.go:258`, `session_page_service.go:146`,
  `stats_service.go:79`, `synthesis_service.go:192`, `teammates/teammates_service.go:237` ;
  cinq `fmt.Errorf("...: PlayerMatchesRepo non câblé (P4.3 finale exige le wiring DI)")` —
  message FR en dur, non catalogué dans l'inventaire §2.F (« Messages d'erreur et
  descriptions d'API »). Reprise : L7, une fois D6 tranché — vérifier d'abord si ce message
  atteint effectivement un client HTTP (sinon il reste une erreur de câblage interne,
  jamais affichée, et n'a pas besoin d'un code machine).
- 2026-09-07 ; `home_highlights.go:4,87,118`, `stats_canonical.go:46`,
  `synthesis_service.go:189`, `synthesis_service_legacy.go:20,46,84,161-162`,
  `synthesis_service_builders.go:63` ; une corruption SECONDAIRE (espace insécable
  aplati en espace normal, apostrophe/guillemet courbe aplati en ASCII droit),
  antérieure ou postérieure au roundtrip PowerShell 5.1 (cause non identifiée), avait
  détruit l'octet nécessaire à la réinterprétation CP1252 automatique sur ces 6 sites —
  corrigés à la main caractère près (le mojibake retiré, l'espacement/la ponctuation
  environnante laissés tels quels, règle « ne rien changer d'autre »). Reprise : si un
  lot `docs/`/`.ai/` traite un jour le mojibake hors Go (D9), s'attendre au même résidu
  et à la même méthode manuelle — un simple decode-CP1252 automatique ne suffira pas
  partout.

## 10. DÉCOUVERTES DE L'EXÉCUTION (Q4, 2026-09-07)

> Même remarque qu'en §9 : consignées SANS être traitées (règle 7 plan-execution).

- 2026-09-07 ; `internal/archlint/no_french_label_literal_test.go` (mesure AST complète,
  périmètre `internal/{service,analysis,api/handlers,notify,games}` hors migrations) ; le
  garde-rail final révèle **132 fichiers / 538 littéraux accentués**, très au-delà des ~15
  fichiers cités dans l'inventaire §2 (échantillon, pas exhaustif — §2.F listait 9 fichiers
  avec « … »). L'essentiel : des messages d'erreur `api/handlers/*` non catalogués
  individuellement. Reprise : L7, une fois D6 tranché — la liste complète est dans
  `frenchLabelAllowlist`, prête à servir de check-list de migration.
- 2026-09-07 ; `internal/domain/match_view.go` (`MatchPersonalResult`, `MatchScoreboardRow`)
  ; `personal_result` (l'objet entier) et le `outcome` par ligne de scoreboard n'ont AUCUN
  lecteur web trouvé (grep `personal_result` et `outcome_label`/`outcome` scoreboard dans
  `apps/web/src` : 0 hit hors types générés). Pas traité ici (D4 ne visait que
  `outcome_label` nommément ; supprimer tout `MatchPersonalResult`/ligne de scoreboard
  serait une extension de périmètre non autorisée par Q4). Reprise : à qualifier — code
  mort plus large que le seul champ d'issue, ou lecteur futur prévu ?
- 2026-09-07 ; `internal/service/match_history_explorer_options.go` (`computeAvailableOutcomes`,
  `perfTierLabel`, `skillTierLabel`) ; ce fichier porte AUSSI des libellés FR en dur pour
  les paliers de perf et les tiers CSR (`"Excellent"`, `"Bon"`, `"Correct"`…, `"Argent"`,
  `"Or"`…), hors périmètre Q4 (familles D/rangs, L5) mais dans le même fichier que le site
  touché (`outcomeLabel(o)` → `outcomeKeyFromHaloCode(o)`). Reprise : L5.
- 2026-09-07 ; `apps/web/src/features/explorer/ExplorerPage.filterOptions.ts`
  (`withCounts`) ; le filtre « Résultat » de l'Explorer ignore déjà le champ `label` servi
  par le backend pour `available_outcomes` (ne lit que `.value`/`.count`) — ses libellés
  viennent d'un i18n local (`explorer.toml`, `t('explorer.filters.outcome_win')` etc.).
  `computeAvailableOutcomes` sert donc désormais une clé canonique jamais lue par le web
  actuel (ni avant, ni après Q4) : conforme à D5 en l'état, mais le champ `Label` de
  `domain.LabelValue` pour cette dimension est un candidat « champ mort » au même titre
  que `MatchPersonalResult` ci-dessus. Reprise : à qualifier avec L3 (options de filtre
  title-agnostic).

## 11. DÉCOUVERTES DE L'EXÉCUTION (M5 première moitié — L2+L5, 2026-09-07)

> Consignées SANS être traitées (règle 7 plan-execution). Périmètre fermé au lot : L2 (accueil)
> et L5 (rangs) seulement — PAS L3 (modes), PAS L4 (armes).

- 2026-09-07 ; `internal/games/mappings/ranks.go` ; la carte du plan (§2.D, §4 L5) associait ce
  fichier au tier CSR de `compare_service.go::csrUnrankedLabel`. Vérifié sur pièces :
  `RankCatalog`/`RankEntry` couvrent le rang de CARRIÈRE (XP, « Général Platine VI »), un
  système DIFFÉRENT du tier CSR (Bronze..Onyx + sous-palier, ranked matchmaking). Aucune
  cible commune n'existe aujourd'hui pour les DEUX (le CSR se localise 100% côté web via
  `lib/skillTiers.ts`, pas via `mappings/ranks.go` ni via `/field-mappings`). Reprise : si un
  lot futur veut unifier ces deux catalogues de rang sous `/field-mappings`, corriger d'abord
  la doctrine (deux familles distinctes, pas une).
- 2026-09-07 ; `internal/service/compare_service.go::csrRankLabel` (+ `skillTierLabel` du même
  fichier) ; formate encore le TIER CSR en clair (« Platine IV », FR uniquement — bug visible
  sous UI EN, non corrigé par ce lot car hors du périmètre exact demandé : seul
  `csrUnrankedLabel` était nommé). Ce calcul est dupliqué EN TROIS ENDROITS avec des variantes
  légères : `compare_service.go::csrRankLabel`/`skillTierLabel`,
  `analysis/home_canonical_skill.go::BuildCSRTierLabelFromEN`/`csrTierENtoFR`, et
  `sync/csr_writes.go::formatCSRTierLabel`/`tierENtoFR` — au-delà du seuil de 2 copies (règle
  6 du dépôt). `sync/csr_writes.go` va plus loin : il PERSISTE le libellé FR dans
  `match_skill_rank.tier_label` (table append-only, ADR 0026), donc corriger cette famille à la
  racine implique une migration de données (`backfill-killsource`-like), pas un simple
  remplacement de littéral — hors format d'un lot L5 « accueil/rangs ». Reprise : lot dédié
  « CSR tier label — clé canonique + migration append-only », après une décision explicite de
  l'utilisateur sur le coût (migration DB) vs bénéfice (bug EN visible, aujourd'hui contourné
  côté web par `localizeTierLabel` pour les 2 lecteurs qui appellent cette fonction).
- 2026-09-07 ; `internal/service/match_history_explorer_options.go::skillTierLabel` +
  `perfTierLabel` ; toujours en dur (déjà repérés §10 lors de Q4, « Reprise : L5 »). Vérifié à
  nouveau : `domain.LabelValue.Label` pour ces deux dimensions (`available_skill_tiers`,
  `available_perf_tiers`) n'a AUCUN lecteur web (`ExplorerPage.filterOptions.ts::withCounts` ne
  lit que `.value`/`.count`) — même statut « champ mort » que `computeAvailableOutcomes` avant
  Q4. `skillTierLabel` recoupe le même tier CSR que la découverte précédente (4e copie du même
  calcul si on compte celle-ci). `perfTierLabel` (paliers de performance, pas CSR) est une
  famille distincte, non couverte par l'inventaire §2 du plan. Non traité ici : le fichier est
  partagé entre plusieurs dimensions Explorer, et une correction partielle (un seul des deux
  littéraux) aurait laissé le fichier dans un état incohérent sans plan de test dédié. Reprise :
  décision de périmètre (L5 strict = CSR seulement, ou nouveau L9 = paliers de perf ?) avant
  d'y toucher.
