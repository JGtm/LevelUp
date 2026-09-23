# Vérification adverse V-GO-D2

Cadre : lecture seule, dépôt `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`,
branche `feat/v75`, HEAD `736ccf3c3`. Aucune compilation, aucun test lancé, aucun fichier du
jeu ouvert.

## Constat 1 — le runtime écrit un catalogue versionné, chaque déploiement l'écrase : TIENT (gravité → P1)

### Ce que j'ai vérifié

Le chemin d'écriture existe et est câblé, exactement comme décrit :

- `apps/go-api/internal/sync/replayartifacts/mvar_rattrapage.go:205` — `mapcatalog.AddEntry(catPath, mapID, entry)`
- `apps/go-api/internal/sync/replayartifacts/artifacts.go:320` — `rattraperCartesAbsentes(ctx, d, work, d.MvarFetcher)`
- `apps/go-api/internal/sync/convergence.go:622` — `s.client.(replayartifacts.MvarFetcher)`
- `apps/go-api/internal/domain/title/registry.go:797-799` — `MapWeaponPadsPath` → `<repoRoot>/data/titles/<slug>/reference/map_weapon_pads.json`

Fichier suivi par git, et le déploiement le rend bien à sa version commitée :

```
$ git ls-files --error-unmatch data/titles/halo_infinite/reference/map_weapon_pads.json
data/titles/halo_infinite/reference/map_weapon_pads.json
$ sed -n '33p;34p' scripts/deploy.sh
git reset --hard origin/main
git clean -fd --exclude=data/ --exclude=.env.local --exclude=app_settings.json --exclude=db_profiles.json
```

Le commit incriminé est réel, y compris son message :

```
$ git show --numstat 5426e256b -- data/titles/halo_infinite/reference/map_weapon_pads.json
332  0  data/titles/halo_infinite/reference/map_weapon_pads.json
    journal(wip): deux entrees d une session parallele — Match view et Classements
    Commitees TELLES QUELLES, sans relecture ni modification, [...]
```

### Ce que l'auditeur n'a pas vu

**(a) Le code n'est PAS en production, et ne peut pas y être aujourd'hui.**

```
$ git cat-file -e origin/main:apps/go-api/internal/sync/replayartifacts/mvar_rattrapage.go
fatal: path '...' exists on disk, but not in 'origin/main'
$ git log --diff-filter=A --oneline -- apps/go-api/internal/sync/replayartifacts/mvar_rattrapage.go
a0e465ac6 cablage(mvar): le fetch de films comble le catalogue des cartes absentes   (2026-09-01)
```

Le fichier n'existe ni sur `main` ni sur `origin/main`. `scripts/deploy.sh:33` déploie
`origin/main`. Le runtime de production ne contient donc pas une ligne de ce chemin
d'écriture. (Cohérent avec le mode « branche unique `feat/v75`, un seul merge final ».)

**(b) Même après le merge, la production n'atteint pas l'écriture.** La chaîne de gardes,
lue ligne à ligne :

- `internal/replaybuild/placement.go:79-84` — en production, réglage vide ⇒ `worker`.
- `internal/replaybuild/placement.go:72-77` — `worker` **sans `LEVELUP_BUILD_WORKER_TOKEN`
  ⇒ `PlacementOff`**.
- `internal/sync/replayartifacts/artifacts.go:394` — `armee()` rend **faux** sur `Off`.
- `internal/sync/replayartifacts/artifacts.go:295-296` — `Run` **sort là**, c'est-à-dire
  **avant** le `rattraperCartesAbsentes` de la ligne 320.

Et le jeton n'est pas posé : absent de `.env.local.example`, absent de
`docs/RUNBOOK_GO_LIVE*`, et surtout porté comme **report OUVERT** au registre :

- `.ai/V7.5/REGISTRE_REPORTS.md:150` — « La production n'enfile RIEN tant que
  `LEVELUP_BUILD_WORKER_TOKEN` est absent […] **Consequence : ce lot n'active rien en
  prod** […] poser le jeton sur le VPS web au deploiement de l'ouvrier — c'est le SEUL
  geste qui ouvre le fil de l'eau ».
- `.ai/V7.5/REGISTRE_REPORTS.md:149` — le déploiement du 2e VPS est lui-même un report non
  soldé.

La conséquence n°1 du constat (« chaque `git reset --hard` jette les cartes que la synchro
y avait ajoutées ; leurs artefacts retombent définitivement en `map_absent` ») décrit donc
un état qui **n'existe pas** et n'existera qu'après un geste d'activation explicite,
lui-même enregistré comme report. Ce n'est pas une perte silencieuse mesurée : c'est un
risque conditionnel.

**(c) L'écriture à l'exécution est une décision écrite et argumentée, pas une dérive.**
`apps/go-api/internal/mapcatalog/store.go:3-9` : « Jusqu'ici, `map_weapon_pads.json`
n'était écrit que par une CLI […] Le rattrapage au fetch de films l'écrit désormais **A
L'EXECUTION**, pendant que le serveur LIT le même fichier. » Suivent les trois garanties
posées pour ça : écriture atomique à nom de temporaire **unique** (le nom fixe avait été
identifié et corrigé), **ajout seul** structurel (`AddEntry` refuse une clé existante,
`store.go:104-113`), verrou consultatif borné à 2 s (`store.go:73-88`). Le lot est
journalisé (`.ai/thought_log.md`, entrées du 2026-09-01), y compris sa ronde de revue.
L'en-tête de `mvar_rattrapage.go:1-38` motive aussi le placement (le sync rapide reste
intact, la cuisson reste offline-pure).

**(d) En revanche, sur le volet local, le constat est plus fort que ce que l'auditeur a
écrit : les 332 lignes viennent bien du RUNTIME, pas d'un outil lancé à la main.**
`deposerMvar` (`mvar_rattrapage.go:230-236`) écrit `CacheRoot/mvar/{mapID}/{base}` avec
`CacheRootDir() = <repoRoot>/data/cache` (`registry.go:713-715`). Sur disque :

```
data/cache/mvar/d39600e2-3c35-4a3a-bdf5-7b3cbdde98e1/map.mvar   1 218 635 o   3 sept. 14:14
data/cache/mvar/7097bc4f-...  data/cache/mvar/d5c5eb4f-...
```

`cmd/mapopads-build` lit un dépôt PLAT de `.mvar` (`--from`), il ne produit pas cette
arborescence par `map_id`. L'entrée « Detachment » entre au catalogue le 4 sept. à 00:14,
le lendemain du dépôt du `.mvar`, et `a0e465ac6` (câblage, 1er sept.) est bien ancêtre de
`5426e256b`. Le préalable est cohérent lui aussi : Detachment n'entre au catalogue
d'objectifs — sans lequel le rattrapage ne sait pas quel `.mvar` demander — que le 3 sept.
(`bc09e8297`), par `mapobj-build` lancé à la main.

### Conséquence réelle reformulée

Sur un poste de développement, un cycle de sync écrit dans un fichier de référence suivi
par git, et cette donnée est effectivement entrée au dépôt une fois (Detachment, +332 L)
dans un commit dont le message annonce qu'il n'a pas été relu ; en production rien n'est
écrit aujourd'hui (le code n'est pas sur `main`) ni ne le sera avant la pose du jeton
d'ouvrier, report ouvert au registre — donc le motif « P0 donnée versionnée corrompue par
le runtime en prod » tombe et le constat retombe à P1.

---

## Constat 2 — deux tests hors tag `gamefiles` ouvrent l'installation : TIENT (P1 maintenu, chiffrage du coût à corriger)

### Ce que j'ai vérifié

```
$ head -1 apps/go-api/cmd/mapstruct-build/equivalence_gamefiles_test.go
// cmd/mapstruct-build — equivalence_gamefiles_test.go : le test qui manquait.
$ grep -n "go:build" apps/go-api/cmd/mapstruct-build/equivalence_gamefiles_test.go
(aucun résultat)

$ grep -rln "DeployRoot(\|LevelsDir(\|ChercheModuleInstalle(" --include='*_test.go' \
    apps/go-api/internal apps/go-api/cmd | grep -v '^apps/go-api/internal/himap/'
apps/go-api/cmd/mapfond-build/reglages_test.go
apps/go-api/cmd/mapstruct-build/equivalence_gamefiles_test.go
```

- `equivalence_gamefiles_test.go:40` appelle `himap.LevelsDir(deployVariant)`, `:64`
  `extractStructure(...)`. `deployVariant` est une **constante** (`main.go:43`, `= "pc"`) :
  **aucune garde par variable d'environnement**, `grep Getenv|Setenv` sur les deux fichiers
  ne rend rien.
- `reglages_test.go:145-173` (`TestModuleGeometrieExisteDansLInstallation`) appelle
  `himap.DeployRoot()` puis, par carte, `os.Stat` **et** `himap.ChercheModuleInstalle(cle)`
  — donc pas seulement des `Stat`, mais une porte d'entrée que le ratchet du dépôt liste
  nommément.
- Le ratchet est bien aveugle hors de son paquet : `internal/himap/corpus_tag_test.go:66`
  `filepath.Glob("*_gamefiles_test.go")` et `:98` `Glob("*_test.go")`, relatifs au paquet ;
  son allowlist `horsCorpusAutorises` (`:56-59`) ne compte que deux fichiers, tous deux
  dans `internal/himap`.
- La règle enfreinte est bien celle du dépôt : `appelsInstallation = {"DeployRoot(",
  "LevelsDir(", "ChercheModuleInstalle("}` (`corpus_tag_test.go:44`).

**Le gate local les exécute réellement.** `make go-api-test` ne les atteint pas (il ne
cible que `./internal/domain/... ./internal/analysis/... ./contracttest/...`, Makefile:193-196)
et le job unitaire de la CI non plus (`ci.yml:193`). Mais `make gate-push` (Makefile:310)
appelle `scripts/check_test_baseline.sh tests`, dont `run_current_suite()` lance
`CGO_ENABLED=1 go test -tags=integration -count=1 -timeout=300s -p 1 -json ./...`
(`scripts/check_test_baseline.sh:275-276`) — **tout le module, `cmd/` compris**.

### Ce qui confirme, et ce qui est surévalué

Confirme : les deux fichiers échappent au tag, le ratchet ne peut structurellement pas les
voir, et l'asymétrie qui rend l'oubli invisible est exactement celle que
`corpus_tag_test.go` décrit — les deux **skippent proprement** sans le jeu
(`t.Skipf("installation du jeu introuvable : %v", err)` `:41` ; `t.Skip(err)`
`reglages_test.go:147`), donc la CI reste verte et personne ne voit le coût.

Surévalué : la conséquence chiffrée. `equivalence_gamefiles_test.go` ne régénère que les
structures **figées présentes au dépôt**, et il y en a deux
(`data/titles/halo_infinite/reference/map_structure/{ridgeline,sgh_streets}.json`) — pas les
26 cartes du balayage à 203 s. Et `reglages_test.go` se limite à un `DeployRoot` + des
`Stat` + un `ReadDir` du répertoire des niveaux. L'ordre de grandeur est donc de deux
modules, pas du sinistre de 20 minutes qui a motivé le tag. L'auditeur le concède
d'ailleurs à demi dans son traitement (« allowlister `reglages_test.go` […] même cas que
`deploy_root_test.go` »), mais sa ligne « Conséquence » ne le dit pas.

### Conséquence réelle reformulée

`make gate-push` rouvre bien l'installation du jeu depuis `cmd/` sur un poste où Halo est
installé, hors de tout tag et hors de portée du ratchet — le trou de règle est réel et le
garde-rail est structurellement incapable de le voir ; seul le coût annoncé est à ramener
de « dizaines de secondes par carte » à deux modules plus quelques `Stat`.

---

## Constat 3 — `internal/himap/heightfield.go` : 175 L de code mort : TIENT (et se renforce)

### Ce que j'ai vérifié

```
$ grep -rn "HeightField\|MinNormalZWalkable\|NewHeightField\|rasteriseTriangle" apps/go-api --include=*.go \
    | sed 's/:[0-9]*:.*//' | sort | uniq -c
     15 apps/go-api/internal/himap/heightfield.go
      4 apps/go-api/internal/himap/heightfield_test.go
$ wc -l apps/go-api/internal/himap/heightfield.go apps/go-api/internal/himap/heightfield_test.go
 175 heightfield.go
  97 heightfield_test.go
```

Aucun appelant indirect : les 30 `\.AddMesh\(` du dépôt portent tous sur `Volume` ou
`Rendu` (`volume_test.go`, `rendu*.go`, `cuisson_forge.go:565`, `objet_isole.go:259`),
jamais sur un `*HeightField` — et le type n'implémente aucune interface (aucun `interface`
ne déclare `AddMesh`/`Couverture`/`Cellule`). Les lignes citées par l'auditeur sont
exactes : `:16` `MinNormalZWalkable`, `:20` `HeightField`, `:36` `NewHeightField`, `:50`
`AddMesh`, `:83` `rasteriseTriangle`, `:133` `At`, `:147` `Cellule`, `:156` `Couverture`.
Les seules références externes sont `heightfield_test.go:49,83,90`.

### Ce qui confirme

**Aucune réservation au registre.** `grep -in "heightfield" .ai/V7.5/REGISTRE_REPORTS.md` →
0 résultat. La seule ligne du registre sur une primitive non branchée
(`REGISTRE_REPORTS.md:55`) vise `ComposanteAccessible`, un autre symbole, et elle est
explicite (« conservee comme instrument, non branchee ») — ce que `heightfield.go` n'a pas.

**Deux documents du chantier le condamnent déjà.**
`.ai/V7.5/cartes/HANDOFF_PORT_TRIANGLES_2026-08-08.md:231` : « `internal/himap/heightfield.go`
champ d'altitude — **APPROCHE ABANDONNEE**, cf. §3 » ; et `:260` : « **Dette restante** :
`heightfield.go` est une approche abandonnee toujours au depot (son en-tete le dit) — **la
supprimer si rien ne la rappelle** ». L'item de plan qui le mentionne
(`PLAN_PORT_TRIANGLES_GO.md:136`, T6) est marqué `[~]` et son propre texte raconte que le
sujet a été résolu **autrement** (correction du rendu, `volume.go`), pas par le champ
d'altitude.

**Aggravation que l'auditeur n'a pas relevée** : le handoff affirme « son en-tete le dit ».
C'est faux. `grep -in "abandon|obsolete|deprecated|instrument" heightfield.go` → aucun
résultat ; l'en-tête `:1-9` se lit comme une justification de conception vivante (« POURQUOI
CE FICHIER EXISTE […] Mesure du 2026-08-08 sur Cliffhanger »). Le seul garde-fou documenté
contre le piège de lecture n'existe donc pas — c'est l'anti-pattern n°9 (« doc inversée »)
superposé à l'anti-pattern n°1.

### Conséquence réelle reformulée

175 L + 97 L de test compilés et lintés à chaque passage sur `himap`, sans appelant ni
réservation, déjà désignés « approche abandonnée / dette restante, à supprimer » par le
handoff du 2026-08-08 — et dont l'en-tête, contrairement à ce que ce handoff promet, ne
prévient pas le lecteur.

---

## Constat 4 — cinq fichiers > 500 L, aucune exemption : TIENT

### Ce que j'ai vérifié

Recompte sur tout le périmètre déclaré (`himap`, `himodule`, `ooz`, `hinavmesh`,
`mapdecoupe`, `mapcatalog`, les 17 outils `cmd/map*|vehicle-sprite|vs-measure|
weapon-icons-build|zone-attribution|oddball-terrain`) :

```
1048  apps/go-api/internal/himap/cuisson_forge.go
 879  apps/go-api/cmd/mapfond-build/reglages.go
 830  apps/go-api/internal/himap/sonde_locs_gamefiles_test.go          (test)
 731  apps/go-api/internal/himap/physique_sonde_gamefiles_test.go      (test)
 681  apps/go-api/internal/himap/cartes_forge.go
 615  apps/go-api/internal/himap/cuisson.go
 537  apps/go-api/cmd/mapfond-build/cuisson.go
 502  apps/go-api/internal/himap/filtres_reclaimer_gamefiles_test.go   (test)
```

Les cinq chiffres du constat sont exacts au nombre près. Le constat **sous-estime** même :
trois fichiers de test dépassent aussi 500 L et ne sont pas comptés.

Aucune exemption : `grep -in "exemption|nolint|500 L|derogation|CLAUDE.md"` sur les cinq
fichiers ne rend que des champs d'options sans rapport (`SeuilArete`, `SeuilCouverture`,
`SeuilSubstitution`…).

Tous créés pendant la campagne, et rien n'est gelé par la baseline :

```
cuisson_forge.go / cuisson.go / cmd/mapfond-build/cuisson.go  2026-08-10
cartes_forge.go                                               2026-08-13
cmd/mapfond-build/reglages.go                                 2026-08-26
$ git ls-tree v7.3.0 apps/go-api/internal/ | grep -E "himap|himodule|ooz|hinavmesh|mapdecoupe|mapcatalog"  -> vide
$ git ls-tree v7.3.0 apps/go-api/cmd/     | grep -E "mapfond|mapstruct|mapopads|mapobj|mapcallouts|mapquant" -> vide
```

Aucun garde-rail de taille de fichier : `.golangci.yml:36-46` n'active que
`revive/gocyclo/funlen/lll/goconst/unconvert/unparam/bodyclose/noctx/prealloc` — aucun ne
mesure la longueur d'un fichier — et `:152-167` exempte `^cmd/` de `funlen`, `lll`,
`gocyclo`.

### La seule nuance à porter au crédit du code

`cartes_forge.go:1-5` porte bien une raison écrite sur sa **nature** : « Ce fichier est de
la **DONNEE** : une declaration par carte Forge dont le fond est produit. La chaine de
cuisson vit dans `cuisson_forge.go` ; les garde-rails de la declaration dans
`cle_forge_test.go` ». Ce n'est pas une exemption R5 (le seuil n'est jamais invoqué), et
cela ne dit nulle part **pourquoi** cette donnée reste compilée plutôt que publiée sous
`reference/` comme les neuf autres catalogues — donc la conséquence du constat (« ajouter
une carte Forge impose une recompilation ») tient entière. Mais la formule « table de
données compilée dans le binaire » doit se lire avec ce commentaire : le fichier se déclare
donnée et est gardé comme telle, il n'a simplement pas d'exemption de taille.

### Conséquence réelle reformulée

Cinq fichiers de production nés pendant la campagne dépassent le seuil de 500 L sans
exemption écrite, hors de toute dette gelée, et la règle R5 n'a aucun linter qui la mesure
— sur `cmd/` elle n'a même pas `funlen` pour la suppléer.

---

## Bilan : 3 tiennent, 0 réfutés, 1 requalifié

| Constat | Verdict |
|---|---|
| 1 — runtime écrit un catalogue versionné | **TIENT, gravité P0 → P1** (la production ne l'exécute pas : code absent de `main`, et `worker` sans jeton dégrade en `off` avant l'appel ; reste le volet local, confirmé et même renforcé — l'écriture vient bien du runtime) |
| 2 — deux tests hors tag `gamefiles` | **TIENT** (P1 maintenu ; le chiffrage du coût est à ramener à 2 modules) |
| 3 — `heightfield.go` code mort | **TIENT** (renforcé : aucune réservation au registre, et l'en-tête ne porte pas l'avertissement que le handoff lui prête) |
| 4 — cinq fichiers > 500 L | **TIENT** (chiffres exacts, périmètre même sous-estimé ; nuance écrite sur `cartes_forge.go`) |
