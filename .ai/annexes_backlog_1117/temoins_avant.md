# Temoins AVANT (item 0.2 du plan des entrees du backlog du 2026-11-17)

Mesures du 2026-11-17, lecture et mesure seulement, aucun code applicatif modifie.

- Worktree `C:/t/bl1117`, branche `chore/830-backlog-1117`, tete `3ae75c78f58eb3bfade376ddd4c188ffb4046f2a` (`docs(plan-maitre-830): ...`), arbre propre avant les mesures (`git status --short` vide) ; les mesures n'ont rien ecrit dans le depot hors des trois fichiers de sortie.
- Go `go1.26.1 windows/amd64`, `gcc` de msys64 present dans le PATH (`gcc.exe (Rev13, Built by MSYS2 project) 15.2.0`). `GOWORK=off`, `GOCACHE=%LOCALAPPDATA%/go-build-bl1117-0` (cache dedie, donc COMPILATION A FROID pour la premiere commande).
- Web : `npm ci` dans `apps/web` (code de sortie 0, 0 vulnerabilite signalee).
- Machine : 16 coeurs logiques. D'autres agents tournaient en meme temps (des processus `go.exe` etrangers etaient visibles pendant la mesure du paquet DuckDB).

## 1. Paquet `apps/go-api/internal/platform/duckdb`

Commande : `cd apps/go-api && go test -tags=integration -json -timeout 900s ./internal/platform/duckdb/` (code de sortie 0).

| Mesure | Valeur |
|---|---|
| Tests lances (evenements `run` d'un test) | 1 395 (dont 1 256 de premier niveau, 139 sous-tests) |
| Reussis (evenements `pass` d'un test, sous-tests compris) | 1 393 |
| Echoues | 0 |
| Sautes | 2 |
| Reussis, tests de premier niveau seuls | 1 254 (sur 1 256 lances) |
| Sautes, premier niveau | 2 (les deux, donc aucun saut dans un sous-test) |
| Duree du paquet rapportee par `go test` (evenement du paquet) | 578,657 s |
| Duree murale de la commande (compilation a froid + essais) | 658 s (environ 79 s avant le premier evenement de test) |

**Cette duree est mesuree SOUS CHARGE (d'autres agents occupaient la machine, cache de compilation a froid pour la premiere commande) : elle ne vaut PAS la mesure « au calme » que prescrit l'item 0.2. La mesure au calme, paquet seul, reste a prendre (A.5 compare a elle). Le nombre de tests, lui, ne depend pas de la charge : c'est la valeur de reference pour « meme nombre de reussites » (gate A).**

Les deux tests sautes (avec le motif imprime par le test) :
- `TestMediaFilters_ModeFilter_ExactNotSubstring` : `media_repo_filters_test.go:802: semantique ModeFilter ambigue : sous-mode seul vs categorie/sous_mode`.
- `TestTemoinBasculeArme` : `temoin_bascule_arme_probe_test.go:225: sonde non armee : TEMOIN_ARME_SHARED et TEMOIN_ARME_META requis`.

Les plus longs tests de premier niveau (secondes, sous charge) : `TestAllMediaFilesReadsFilterDeleted` 25,1 ; `TestPlayerProgressionStore_VideEtIsole` 8,4 ; `TestLecturesDeLaVueDesNoms_Ratchet` 7,6 ; `TestNoATTACHOnSocialDB` 7,3 ; `TestLoadMediaFiles_RealTopology_NoCrossDBSQL` 7,1 ; `TestUpsertItemDefinition_InsertsStructuredData` 6,1 ; `TestNoPlainPlayerHandleReads` 5,1. Somme des durees de premier niveau : 591,2 s (les tests ne tournent pas en parallele entre eux : la somme approche la duree du paquet).

Le sous-paquet `profilerepo`, `worldrepo` et `prestige` (dossiers sous `internal/platform/duckdb/`) sont des paquets distincts : ils ne sont pas dans ce compte.

## 2. Tests de caracterisation du taux par minute de l'Escouade

Commande : `go test -count=1 -v -run 'TestTemoinCadence_StatsParMinuteDeLEscouade|TestTemoinMinute_SynergyScorePerMinute|TestStatsParMinuteEscouade_MatchSansDureeHorsDuTauxEtDuCompte|TestBuildSquadPerMinuteStats' ./internal/service/teammates/` : `ok`, 9 tests `PASS` (les trois temoins ci-dessous plus les six `TestBuildSquadPerMinuteStats_*` de `teammates_squad_per_minute_test.go`).

Fichiers : `apps/go-api/internal/service/teammates/per_minute_caracterisation_test.go` et `per_minute_temoins_test.go` (plus `per_minute_sans_duree_test.go`, voisin, qui fixe aussi des valeurs).

### `TestTemoinCadence_StatsParMinuteDeLEscouade` (per_minute_caracterisation_test.go)

Valeurs fixees dans le test (comparees en chaine `'g', -1`). Jeu : 4 matchs du joueur principal (m1 600 s, m2 937 s, m3 sans seconde jouee, m4 hors session), 3 du coequipier `friend` (m1, m2, m4), un doublon de ligne de squad pour m1.

| Joueur | MatchCount | frags/min | morts/min | assistances/min | degats infliges/min | degats subis/min |
|---|---|---|---|---|---|---|
| `main` | 3 | 0.66 | 0.31 | 0.12 | 300 | 136.63 |
| `friend` | 2 | 0.23 | 0.35 | 0.35 | 48.8 | 19.21 |

Le test echoue si `DamageDealtPerMinute` ou `DamageTakenPerMinute` est nil pour l'un des deux. Ces valeurs sont celles que C.3 ne doit pas faire bouger (le coequipier de ce jeu a tous ses frags, morts et assistances connus).

### `TestStatsParMinuteEscouade_MatchSansDureeHorsDuTauxEtDuCompte` (per_minute_sans_duree_test.go)

| Joueur | frags/min | morts/min | assistances/min | MatchCount |
|---|---|---|---|---|
| `main` (m2 sans duree : `TimePlayed` nil) | 0.6 | 0.3 | 0.3 | 2 |
| `friend` (m2 avec `TimePlayed` = 0) | 1.2 | 0.6 | 0.9 | 2 |

### `TestTemoinMinute_SynergyScorePerMinute` (per_minute_temoins_test.go)

Sortie `-v` (lignes `t.Logf`) :

```
synergy.937 = 79.01814300960513      (score 1234, 937 s)
synergy.59 = 78.30508474576271       (score 77, 59 s)
synergy.zero = 0                     (500, 0 s)
synergy.negatif = 0                  (500, -10 s)
```

Note de lecture : `paceMatch` (`teammates_squad_per_minute.go:19`) est `struct{ kills, deaths, assists, secs int }` ; aucun des jeux ci-dessus ne contient un coequipier aux frags inconnus (`Self.Kills == nil`), donc aucun temoin existant ne couvre le cas que C.3 corrige.

## 3. Garde-rail des couches : `apps/web/src/features/dismissableLayers.guard.test.ts`

Plafond chiffre actuel : `PLAFOND_HORS_REJEU = 17` (ligne 28), compte en FICHIERS `.tsx` de `features/` hors `features/match-replay/` ; le test echoue aussi quand le compte baisse sans que la constante suive (cliquet). Le test passe (inclus dans la serie vitest du paragraphe 5).

Liste exacte des 17 fichiers que le garde-rail compte, obtenue en rejouant sa logique (detecteur AST de `dismissPatterns` : `addEventListener('pointerdown' | 'mousedown')` ou comparaison a `'Escape'`, sur les `.tsx` de `features/` hors tests) ; ligne et motif :

| # | Fichier (sous `apps/web/src/`) | Lignes et motifs |
|---|---|---|
| 1 | `features/_shared/ExperienceDropdown.tsx` | 37 `addEventListener('mousedown')` |
| 2 | `features/_shared/ViewDropdown.tsx` | 27 `addEventListener('mousedown')` |
| 3 | `features/ascension/campaign/StartCampaignModal.tsx` | 58 `=== 'Escape'` |
| 4 | `features/asset-drawer/AssetDockedPanel.tsx` | 61 `=== 'Escape'` |
| 5 | `features/asset-drawer/AssetDrawer.tsx` | 22 `=== 'Escape'` |
| 6 | `features/explorer/GamertagSearchInput.tsx` | 64 `addEventListener('mousedown')` ; 113 `=== 'Escape'` |
| 7 | `features/feedback-drawer/FeedbackDrawer.tsx` | 80 `=== 'Escape'` |
| 8 | `features/media/CoverFlowModal.tsx` | 554 `=== 'Escape'` |
| 9 | `features/media/MediaAudioConfigButton.tsx` | 82 `=== 'Escape'` |
| 10 | `features/media/MediaToolbar.tsx` | 63 `=== 'Escape'` ; 65 `addEventListener('mousedown')` |
| 11 | `features/notifications/NotificationsBell.tsx` | 117 `addEventListener('mousedown')` ; 125 `=== 'Escape'` |
| 12 | `features/palmares/BattlePassRewardLightbox.tsx` | 127 `=== 'Escape'` |
| 13 | `features/spartan-customizer/SpartanCustomizerModal.tsx` | 112 `=== 'Escape'` |
| 14 | `features/squad/SquadFocusStrip.tsx` | 213 `=== 'Escape'` |
| 15 | `features/squad/useSquadPresets.tsx` | 202 `=== 'Escape'` |
| 16 | `features/tactical/TacticalAnalysisView.tsx` | 297 `!== 'Escape'` ; 329 `=== 'Escape'` |
| 17 | `features/tactical/TacticalPlanTueurs.tsx` | 151 `addEventListener('pointerdown')` |

Total : 17 fichiers, attendu 17. Confirme. Douze (lignes 1 a 3 et 5 a 13) sont les ecrans de D.1 ; les cinq autres (4, 14, 15, 16, 17) sont les cas particuliers de D.2. Fichiers deja sur `useDismissable` hors rejeu : `components/shell/_filter_pills/{FiltresPill,PeriodePill,SaisonPill,SessionPill}.tsx`.

## 4. Marqueurs `color-allow`

Critere d'inertie (etabli en lisant `tools/lint-no-hardcoded-colors.lib.mjs`, fonction `scanLine`) : un marqueur est inerte quand, sans lui, `scanLine(chemin, ligne)` rend une liste vide pour sa ligne. Detail et liste fermee : `.ai/annexes_backlog_1117/marqueurs_inertes.md`. La mesure neutralise le mot `color-allow` de la ligne puis rappelle `scanLine`.

| Mesure | Valeur | Attendu |
|---|---|---|
| Lignes portant `color-allow` dans le perimetre balaye par le lint (`features/`, `components/`, `lib/replay/`, hors fichiers autorises) | 95 (47 fichiers) | 95 : confirme |
| dont marqueurs inertes | 19 (15 fichiers ; 5 sur une ligne de commentaire, 14 sur une ligne de code) | 19 : confirme |
| marqueurs actifs | 76 (29 `tailwind`, 22 `hex`, 20 `fonction-couleur`, 5 `hex`+`fonction-couleur`) | |
| `node tools/lint-no-hardcoded-colors.mjs` | `clean (0 violation)` | |
| `shadow-`, `from-`, `via-`, `to-` suivis d'une couleur nommee et d'une teinte, dans le perimetre | 15 occurrences, toutes dans `features/palmares/rarity.ts` (fichier autorise) ; 0 ailleurs | 15 / 0 ailleurs : confirme |

Hors perimetre du lint, non comptes : 2 lignes de `apps/web/src/lib/accessibility/tokenTone.ts` (30, 31) et 13 lignes dans 2 fichiers de test ; total brut `grep` dans `apps/web/src` : 110.

## 5. Tests vitest des dossiers touches par D

Commande, depuis `apps/web` : `npx vitest run src/features/media src/features/asset-drawer src/features/feedback-drawer src/features/spartan-customizer src/features/palmares src/features/ascension src/features/_shared src/features/explorer src/features/notifications src/lib/hooks src/lib/accessibility src/features/dismissableLayers.guard.test.ts` (code de sortie 0).

| Mesure | Valeur |
|---|---|
| Fichiers de test | 138 reussis sur 138 |
| Tests | 1 281 reussis sur 1 281 |
| Echoues / sautes | 0 / 0 (aucun compte de saut affiche par vitest) |
| Duree | 57,15 s (sous charge) |

Sortie de bruit connue et sans effet : messages jsdom « Not implemented: HTMLMediaElement's play() method » et « HTMLCanvasElement's getContext() method » (tests de `media`).

## Ce qui n'a pas ete mesure ici

- La duree du paquet DuckDB « au calme » (voir 1) : a reprendre sans autre agent sur la machine.
- Les durees de migration par cible (0,6 s a 5 s) et de copie de gabarit (19 a 57 ms) citees par le plan : non reproduites ici (hors de l'item 0.2).
