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

**L1 — Issue de match** : EN COURS (`feat/outcome-cle-canonique`). À la clôture : supprimer
`domain.HomeMatchRow.OutcomeLabel` s'il est bien sans lecteur, migrer `duelOutcomeLabel`
(explorer) sur le même helper si le vocabulaire est le même, sinon TOML dédié.

**L2 — Accueil** (`home_locale.go`) : toutes les paires `labelForLocale(locale, fr, en)` vers
TOML + adapter ; l'accueil est le pilote historique de l'ADR 0011 (labels i18n hors canonical),
il y a donc déjà une frontière à respecter — lire `home_service.go:36-42` et l'ADR avant.

**L3 — Modes / playlists / catégories / portées** (B) → `assets.toml` ; ratchet
`no_bare_resolve_mode_ui_test.go` existe déjà dans `archlint` : l'étendre.

**L4 — Armes** (C) ; **L5 — Rangs** (D, cible existante `mappings/ranks.go`).

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
