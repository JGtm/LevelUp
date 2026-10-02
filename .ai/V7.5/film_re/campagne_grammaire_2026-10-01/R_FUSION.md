# R_FUSION — essai de fusion de J12 sur la campagne de grammaire (2026-10-02)

Item du plan : « Essai de fusion AVANT tout developpement de phase 2 » (N9). Chantier « fusion »,
prefixe `r_fusion`. Worktree temporaire `LevelUp-wt-cg-fusion`, HEAD detachee sur `fe18bf67c`
(campagne, phase 1 close). Branche fusionnee : `origin/feat/suite-audit-decodeur-j12`, tete
`bc0e2511a` (29 commits depuis la base commune `8b894a677`, 1 453 fichiers touches).

Rien n'est commite. La fusion est laissee NON COMMITEE dans ce worktree (index : fusion + golden
regenere ; arbre : en plus, le patch de la section 3 applique et non indexe).

Convention : **mesure** = sortie d'une commande lancee ici ; **etabli** = tenu par une preuve
citee ; **suppose** = non verifie.

---

## 1. Verdict en une ligne

La fusion est saine a **un** golden pres (conflit attendu, resolu par la porte du depot) et a
**quatre lignes** pres dans **quatre sondes** de la campagne, que J12 casse a la compilation sous
`-tags=research`. Avec le patch `r_fusion_patchs/r_fusion_ratchets_j12.patch` (4 lignes), toutes
les commandes demandees sont vertes, sauf `TestNoExpiredTODO` d'archlint, rouge pour une echeance
calendaire etrangere a la campagne ET a J12 (section 4).

## 2. Deroule (mesure)

| # | Etat | Commande | Resultat |
|---|---|---|---|
| 1 | `fe18bf67c` | `git merge --no-commit --no-ff origin/feat/suite-audit-decodeur-j12` | 1 conflit : `grammar/testdata/grammar_rev.golden`. `cmd_fermeture/main.go` et `gb1_research_test.go` auto-fusionnes (relus : les modernisations de J12 — `strings.SplitSeq`, `errors.Is(err, fs.ErrNotExist)` — et le mode v2 de la campagne coexistent). |
| 2 | conflit | `git checkout --theirs` du golden, puis `LEVELUP_UPDATE_GRAMMAR_REV=1 go test -count=1 ./internal/games/halo_infinite/film/internal/grammar/ -run TestGrammarRevSuitLaGrammaire -update-grammar-rev` | FAIL **voulu** (la porte ne rend jamais `ok`) : « 1 reference(s) reecrite(s) … revision grammar-2026-09-27.3, empreinte `14b3a79d506b27abd3b002dd3577981f01f18f3fc420e7e8aa21719a36dfba44` ». `git add` du golden (marque la resolution, pas de commit). |
| 3 | idem | `go test -count=1 …/grammar/ -run 'TestGrammarRev\|TestChronique'` | `ok` |
| 4 | idem | `gofmt -l ./internal/games/halo_infinite/film/ ./internal/himodule/` | vide |
| 5 | idem | `go vet ./internal/games/halo_infinite/film/...` | exit 0, aucune sortie |
| 6 | idem, **sans patch** | `go vet -tags=research ./internal/games/halo_infinite/film/... ./internal/himodule/` | **exit 1** : 4 erreurs de type dans `grammar.test` (section 3) |
| 7 | idem, **sans patch** | `go test -count=1 ./internal/archlint/` | FAIL : `TestNoExpiredTODO` **seul** (section 4) — 69,96 s |
| 8 | + patch | `go vet -tags=research ./internal/games/halo_infinite/film/... ./internal/himodule/` | exit 0, aucune sortie |
| 9 | + patch | `go test -count=1 -timeout 40m ./internal/games/halo_infinite/film/...` | exit 0 : **17 paquets ok**, 3 sans test, 0 FAIL (grammar 28,8 s ; replay 25,8 s ; 30 s au mur) |
| 10 | + patch | `go test -count=1 ./internal/archlint/` | FAIL : `TestNoExpiredTODO` seul (meme cause) |
| 11 | + patch | `go test -count=1 -skip TestNoExpiredTODO ./internal/archlint/` | `ok` (29,7 s) |
| 12 | + patch | les 11 tests de ratchet poses par J12, en `-v` | 11 PASS : `TestInstrumentsResearchSontTagues`, `TestFichierTagueResearchEstUnInstrument`, `TestGardeNeCachePasDerriereResearch`, `TestCheminsAiCitesDansLeCodeExistent`, `TestAucunCommentaireNeViseLAncienPaquetFilmdec`, `TestLeDecodeurJournaliseSousLeContexteDeLAppelant`, `TestGrammarEtFactsNeJournalisentPas`, `TestAucunPredicatOSIsDansLePerimetreDuFilm`, `TestAucuneVarExporteeModifiableDansLeDecodeur`, `TestTriTotalAucunNouvelAppel`, `TestTriTotalTableNeFaitQueBaisser` |
| 13 | + patch | `go vet -tags=research ./...` (module entier, CGO=1 : le pas CI elargi par J12.7) | exit 0, aucune sortie (50 s) |
| 14 | + patch | `go test -race -count=1 -run TestDeuxFilmsEnParallele …/grammar/` (job `film-race` de J12.6) | `ok` (4,2 s) |
| 15 | + patch | `golangci-lint run --new-from-merge-base=origin/main` sur `film/internal/grammar/...` et `film/research/...` (v2.12.2, tags par defaut) | `0 issues` |
| 16 | + patch | `go vet -tags=research,campagne_overlay -overlay=<r_fusion_overlay_postj12/overlay.json> …/grammar/` | exit 0, aucune sortie |
| 17 | + patch, temoin | idem avec l'ANCIENNE surcouche (`mesures_bis2_overlay`, chemins remappes vers ce worktree) | exit 0 (section 5) |
| 18 | — | `gofmt -l r_fusion_overlay_postj12/*.go` | vide |

Detail ligne a ligne : `r_fusion_tsv/r_fusion_commandes.tsv`.

Environnement : go1.26.5 windows/amd64, `GOCACHE`/`GOTMPDIR` dans le scratchpad
(`gocache-r_fusion`, `gotmp-r_fusion`), `PATH` prefixe de `C:/msys64/ucrt64/bin`, `CGO_ENABLED=1`,
une commande go a la fois. Aucun film lu, aucune base ouverte, aucun Ghidra.

## 3. Fichiers de la campagne qui violent un ratchet de J12 (etabli)

**Quatre fichiers, une ligne chacun, une seule cause.**

| Fichier (sous `apps/go-api/internal/games/halo_infinite/film/internal/grammar/`) | Ligne | Correction exacte |
|---|---|---|
| `campagne_bis1_research_test.go` | 275 | `wr := profile.QuantRangeCEBiped` → `wr := profile.QuantRangeCEBiped()` |
| `campagne_bis2_chassis_research_test.go` | 73 | idem |
| `campagne_bis3_research_test.go` | 90 | idem |
| `campagne_mesures_research_test.go` | 88 | idem |

- **Ratchet** : `archlint/film_exported_vars_test.go` (`TestAucuneVarExporteeModifiableDansLeDecodeur`,
  lot J12.4, allowlist videe en J12.4 residu `7a64fd34a`). Il a converti la variable exportee
  `profile.QuantRangeCEBiped` en accesseur `func QuantRangeCEBiped() Vec3Range`
  (`profile/plages_quant.go:50`).
- **Symptome** : `cannot use &wr (value of type *func() profile.Vec3Range) as *profile.Vec3Range
  value in argument to ScanVehicleCreations` — la sonde prend l'adresse de la FONCTION.
- **Qui l'attrape** : pas le ratchet lui-meme (il lit le paquet `profile`, pas les lecteurs), mais
  le pas CI `go vet -tags=research ./...` que J12.7 a etendu a tout le module : sans le patch, la
  CI de la vraie fusion est **rouge**.
- **Pourquoi le build par defaut ne voit rien** : les quatre fichiers portent `//go:build research`
  (etape 5 verte, etape 6 rouge).
- **Semantique de mesure** : inchangee — l'accesseur rend la meme valeur que l'ancienne variable
  (etabli : meme litteral `{{-41.10318, 72.10963}, {-56.60697, 57.212566}, {-84.37078, 53.18034}}` dans `plages_quant.go` avant (`8b894a677`, l.44, variable) et apres (`bc0e2511a`, l.50-51, accesseur)).

**Patch** : `r_fusion_patchs/r_fusion_ratchets_j12.patch` (4 hunks, 4 lignes ; sha256
`6b571e5c2dcb70cf82722c88e762bd2bc5b1fee0ed5bb5763eb4d71117d2b20c`). Verifie :
`git apply --cached --check` sur l'index de la fusion brute → APPLICABLE ; `git apply --check -R`
sur l'arbre patche → REVERSIBLE ; `git diff --numstat fe18bf67c` des quatre fichiers = `1 1`
chacun (le patch est leur seul ecart a la campagne). A appliquer depuis la racine du depot :
`git apply .ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_fusion_patchs/r_fusion_ratchets_j12.patch`.

**Aucun autre fichier de la campagne ne viole un ratchet de J12** (mesure, etapes 12 et 15). Les
points ou il aurait pu y en avoir, et pourquoi ils passent :

- **Tris non totaux** (`film_tri_total_test.go`, cliquet vide depuis J12.1) : la campagne a 14 appels
  `sort.Strings/Slice/Ints`, tous dans des `_test.go` ; le perimetre du cliquet exclut les tests
  et les fichiers `//go:build research` (en-tete du ratchet). `frame_closure_detail*.go`
  (production non taguee) : 0 appel `sort.`.
- **Journal dans grammar / slog sans contexte / `os.Is*`** : 0 occurrence dans les 35 fichiers Go (hors `.ai/`) de
  la campagne (grep), les trois ratchets PASS.
- **Variables exportees** : `frame_closure_detail.go` n'exporte qu'un type, une fonction et des
  constantes (`NombreDeSortiesDeVueB` est dans le bloc `const`).
- **Tag research** : toutes les sondes `campagne_*` et les `v2*.go` de `cmd_fermeture` portent
  `//go:build research` en ligne 1 ; aucune n'a un suffixe de garde.
- **Chemins `.ai/` cites** (`doc_chemins_ai_test.go`) : PASS — voir la decouverte D-F2.
- **Lint ratchet** (`--new-from-merge-base=origin/main`) : 0 issue sur grammar et research.

## 4. Le rouge restant : `TestNoExpiredTODO` (etabli, hors perimetre)

`internal/api/handlers/json_huma_coverage_test.go:34` porte `TODO(expiry:2026-10-01)` (migration Huma
de `groups.go`). Echu le 2026-10-01, le test est rouge depuis le 2026-10-02 sur **toute** branche :
le fichier est le meme dans `fe18bf67c`, `bc0e2511a` et `origin/feat/v75` (1 occurrence de
`expiry:2026-10-01` dans chacun, mesure par `git show | grep -c`). Ni la campagne ni J12 n'en sont
la cause ; le corriger est un fix hors perimetre (regle 5 du plan-execution), donc NON traite. Le
reste d'archlint est vert (`-skip TestNoExpiredTODO` : ok).

## 5. Le golden de revision (etabli)

- Avant : campagne `…27.3 3cbd64ac…`, J12 `…27.3 558380db…`, base `…27.3 4bc8e05a…`. Chacune des
  deux branches avait regenere la meme ligne `.3` (la campagne pour `frame_closure_detail*.go`,
  J12 pour ses refactors de grammar) sans monter la revision.
- Apres la porte : `grammar-2026-09-27.3	14b3a79d506b27abd3b002dd3577981f01f18f3fc420e7e8aa21719a36dfba44`.
  `grammar.Rev` reste `grammar-2026-09-27.3` (rev.go:110, intact). Le reste du fichier est celui de
  J12, identique a celui de la campagne (`git diff --cached fe18bf67c` du golden = 1 ligne).
- L'empreinte hache les `.go` hors `_test.go` de la couche : le patch de la section 3 (des
  `_test.go`) ne la change pas. Lors de la vraie fusion, la meme porte doit redonner `14b3a79d…`
  si les sources non-test de grammar sont celles de cet essai (suppose, pas reverifiable ici).
- Les autres goldens touches par l'une ou l'autre branche (`frame_closure.golden`, `ecs_table.tsv`
  de la campagne ; `grammar_perimetre`, `killsource_rev`, `objectives_rev`, `profile_rev`,
  `source_rev` de J12) passent sans regeneration (etape 9). En particulier la carte de fermeture
  des mini-bobines du depot (`frame_closure.golden`) ne bouge PAS avec J12.

## 6. La surcouche re-synchronisee (etabli)

Dossier neuf : `r_fusion_overlay_postj12/`.

| Fichier | Change par J12 | Nature | Delta de mesure campagne (+/-) | Delta re-applique (+/-) | Conflits |
|---|---|---|---|---|---|
| `capture.go` | 0 ligne | — | 22/0 | 22/0 | 0 |
| `lecteur_position.go` | 1 ligne | `go fix` : `for axe := range 3` | 52/0 | 52/0 | 0 |
| `lecteur_position_exceptions.go` | 4 lignes | `go fix` : `range 3`, `range uint(8)` | 152/3 | 152/3 | 0 |

Methode : `git merge-file` a trois voies (mien = version J12, base = version `fe18bf67c`, autre =
`mesures_bis2_overlay/`). Les trois fichiers de la campagne etaient identiques a la base commune
(`fe18bf67c` = `8b894a677` pour ces trois chemins), donc le delta re-applique est EXACTEMENT celui
de la campagne (memes comptes de lignes), et le seul ecart entre l'ancienne surcouche et la neuve
est les 5 boucles modernisees par J12 (`r_fusion_tsv/r_fusion_overlay.tsv`).

- `overlay.json` : chemins de CE worktree (gate de cette mission, etape 16).
- `overlay_campagne.json` : memes remplacements, chemins du worktree de la campagne
  (`LevelUp-wt-campagne-grammaire`), a utiliser apres integration.
- `r_fusion_delta_mesure_NE_PAS_APPLIQUER.diff` : le delta de mesure au format diff, contre les
  fichiers post-J12 — documentation de ce que la surcouche ajoute, JAMAIS a appliquer aux fichiers
  suivis (D10 : outillage de mesure seulement). `git apply --check` le dit applicable sur l'arbre
  post-J12, raison de plus pour le nommer ainsi.
- Temoin (etape 17) : l'ANCIENNE surcouche compile aussi sur l'arbre fusionne. Elle n'est donc pas
  cassee, mais elle remplace les fichiers post-J12 par des versions pre-`go fix` ; les boucles
  `for i := 0; i < 3; i++` et `for i := range 3` sont equivalentes quand le corps ne modifie pas
  l'indice (c'est le cas des 5 boucles, relues) — l'ecart est de forme, pas de mesure.

## 7. Decouvertes (non traitees — regle 5)

- **D-F1 — Regle des commentaires de J12.8 (CLAUDE.md regle 17, skill arch-rules)** : « l'histoire
  (mesures datees, lots, comptes du jour) va dans l'ADR, la chronique ou le journal », pour le code
  neuf. `frame_closure_detail.go` (production non taguee de grammar, neuf de la campagne) porte en
  tete « campagne de recherche sur la grammaire, phase 1, etape 1, 2026-10-01 » et cite des lots
  (`lot 5.16.4`, `lot 5.23`). Ce n'est **pas** un ratchet (aucun test ne le tient), donc hors du
  patch ; a arbitrer avant la livraison de ce fichier hors campagne. Les `v2*.go` de
  `cmd_fermeture` font de meme mais sont des instruments `research`.
- **D-F2 — Chemin cite par deux sondes** : `campagne_bis2_positions_research_test.go:13` et
  `campagne_bis3_ti3_research_test.go:9` citent `mesures_bis2_overlay/`. Le ratchet
  `TestCheminsAiCitesDansLeCodeExistent` (J12.5) passe tant que ce dossier existe ; le supprimer
  ou le remplacer par `r_fusion_overlay_postj12/` sans mettre a jour ces deux commentaires rendrait
  archlint rouge. Recommandation : garder les deux dossiers, ou changer la citation dans le meme
  commit.
- **D-F3 — Equivalence des MESURES apres J12 non mesuree sur film** : J12.1 a converti au moins 155 appels de tri (122 dans replay + 33 dans grammar, messages des commits `499a46743` et `ef7f4db1a`)
  (grammar, replay...) en comparateurs totaux, et J12.3 a retire la journalisation de grammar. Les
  mini-bobines du depot ne bougent pas (`frame_closure.golden` vert), mais aucun chiffre de la
  campagne (fermetures, juge `cmJuge`, T1..T8) n'a ete rejoue sur les 20 temoins apres fusion. Ce
  rejeu n'etait pas dans la mission ; c'est le contrôle a faire si la phase 2 part de l'arbre
  fusionne et compare ses chiffres a ceux de la phase 1.
- **D-F4 — `TestNoExpiredTODO`** rouge partout depuis le 2026-10-02 (section 4) : la CI de toute
  branche le sera aussi.

## 8. Fichiers produits (a integrer a la campagne)

- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/R_FUSION.md` (cette note)
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_fusion_tsv/r_fusion_commandes.tsv`
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_fusion_tsv/r_fusion_violations.tsv`
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_fusion_tsv/r_fusion_overlay.tsv`
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_fusion_patchs/r_fusion_ratchets_j12.patch`
- `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_fusion_overlay_postj12/` (`capture.go`,
  `lecteur_position.go`, `lecteur_position_exceptions.go`, `overlay.json`, `overlay_campagne.json`,
  `r_fusion_delta_mesure_NE_PAS_APPLIQUER.diff`)

Aucune sonde neuve : la mission n'en demandait pas. Fichiers suivis modifies dans CE worktree
temporaire seulement : ceux de la fusion (index), le golden regenere (index), et les quatre sondes
du patch (arbre, non indexees) — tout cela disparait avec le worktree.

## 9. Recette de la vraie fusion (deduite de l'essai)

1. Fusionner `feat/suite-audit-decodeur-j12` ; un seul conflit attendu (`grammar_rev.golden`).
2. Prendre la version J12 du golden, puis
   `LEVELUP_UPDATE_GRAMMAR_REV=1 go test -count=1 ./internal/games/halo_infinite/film/internal/grammar/ -run TestGrammarRevSuitLaGrammaire -update-grammar-rev`
   (FAIL voulu), relancer sans le drapeau → `ok` ; valeur attendue `14b3a79d…` (section 5).
3. `git apply …/r_fusion_patchs/r_fusion_ratchets_j12.patch`.
4. Gates : etapes 4, 5, 8, 9, 11, 13, 14, 16 de la section 2. `TestNoExpiredTODO` restera rouge
   tant que l'echeance n'est pas traitee par ailleurs.
5. Pointer les mesures overlay sur `r_fusion_overlay_postj12/overlay_campagne.json`.
