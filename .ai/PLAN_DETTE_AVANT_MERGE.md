# PLAN — solder la dette de `feat/replay2d-prod` avant le merge vers `main`

> Écrit le 2026-07-31, à la clôture de `PLAN_RECONCILIATION_BRANCHES.md`.
> Contrat d'exécution : skill `plan-execution`.
>
> **CE PLAN NE CONTIENT PAS DE LISTE FIGÉE, ET C'EST VOULU.** Le travail va continuer sur cette
> branche avant le merge : de nouvelles issues vont apparaître, d'autres vont disparaître. Un
> inventaire écrit aujourd'hui serait faux dans une semaine et donnerait une fausse sécurité.
> Ce qui est figé ici, c'est **la mesure de référence**, **la procédure pour la refaire**, et
> **les décisions déjà prises** — pas le contenu.

---

## 0. RÈGLE DE REPRISE — commencer par re-mesurer, toujours

Toute session sur ce plan démarre par le §1. Ne jamais partir des chiffres écrits plus bas :
ils datent du 2026-07-31 et servent de **point de comparaison**, pas d'inventaire.

---

## 1. LA MESURE — une seule commande, deux branches

```bash
# La branche à livrer.
cd apps/go-api && golangci-lint run --timeout 10m --new-from-merge-base=origin/main

# La lignée killfeed, pour savoir ce qui préexiste (worktree .claude/worktrees/killsource-prod).
cd ../../.claude/worktrees/killsource-prod/apps/go-api \
  && golangci-lint run --timeout 10m --new-from-merge-base=origin/main
```

Puis le partage jetable/livré, qui est la seule question qui compte :

```bash
golangci-lint run --timeout 10m --new-from-merge-base=origin/main 2>&1 \
  | grep -E "^[a-zA-Z_/\\]+\.go:[0-9]+:[0-9]+:" | tr '\\' '/' > /tmp/lint.txt
grep -cE "^cmd/(tmp_|wf_)" /tmp/lint.txt   # outillage jetable — disparaît à l'archivage
grep -c  "^internal/"      /tmp/lint.txt   # code LIVRÉ — reste, doit être traité
cut -d: -f1 /tmp/lint.txt | sort | uniq -c | sort -rn        # par fichier
grep -oE "\(\w+\)$" /tmp/lint.txt | sort | uniq -c | sort -rn # par linter
```

### Référence au 2026-07-31 (à comparer, pas à croire)

| branche | total | dans `cmd/tmp_*` + `cmd/wf_*` | dans `internal/` |
|---|---:|---:|---:|
| `feat/killsource-prod` | 30 | **0** | **30** |
| `feat/replay2d-prod` | 48 | 8 | **40** |

**LE FAIT QUI ORIENTE TOUT LE PLAN : la dette n'est pas dans l'expérimentation, elle est dans
le décodeur.** Archiver l'outillage de recherche fait passer notre compteur de 48 à 40 et celui
de la lignée killfeed de 30 à 30. C'est utile mais marginal — la vraie masse est dans
`internal/analysis/filmdec/` (≈ 29 des 30 préexistantes, ≈ 30 des 40 chez nous).

**Corollaire opérationnel** : `filmdec` étant désormais RÉUNI sur cette branche, traiter la
dette ici la traite **pour les deux lignées à la fois**. Il n'y a pas deux chantiers.

### Nature, au 2026-07-31 (40 issues dans `internal/`)

| linter | n | coût | traitement |
|---|---:|---|---|
| `unused` | 14 | **décision par fonction** | lot C |
| `unparam` | 7 | mécanique | lot B |
| `staticcheck` | 4 | lecture au cas par cas | lot D |
| `revive` (nommage) | 4 | mécanique | lot B |
| `unconvert` | 3 | mécanique | lot B |
| `goconst` | 3 | mécanique | lot B |
| `gocyclo` | 2 | vrai découpage | lot D |
| `errcheck` · `ineffassign` · `prealloc` | 3 | mécanique | lot B |

---

## 2. LES LOTS

### Lot A — archiver l'outillage de recherche  *(retire ~8 issues, et beaucoup de bruit)*

- [x] A1. Trancher ce qui est **jetable** et ce qui a une valeur durable. Critère : un binaire
      qui ne sert qu'à établir une grammaire déjà portée dans `filmdec` est jetable ; un
      binaire qui **produit un artefact versionné** (`cmd/mapobj-build`, `cmd/mapstruct-build`,
      `cmd/mapquant-build`, `cmd/replay-build`) reste.
      *FAIT 2026-08-01 : les 405 répertoires `cmd/tmp_*` + `cmd/tmpdbq` + `cmd/wf_*` sont
      jetables — vérifié qu'aucun n'écrit dans un chemin versionné (`data/titles/*/reference/`,
      `config/`) ; ils LISENT des DB ou des chunks. Les quatre `*-build` restent.*
- [x] A2. Supprimer les jetables — `git` garde l'historique, on ne « range » pas dans un dossier
      `archive/` (règle 7 : pas de musée de code mort).
      *FAIT : 462 fichiers, 405 répertoires. Ils étaient SUIVIS par git malgré la règle
      `.gitignore` `apps/go-api/cmd/tmp_*/` (le `.gitignore` n'agit que sur l'inconnu) —
      l'historique les garde donc réellement, la suppression est réversible.*
- [x] A3. Retirer du même coup les entrées d'allowlist devenues sans objet :
      `halowaypointAllowlist["cmd/tmp_filmmanifest/main.go"]`, les préfixes `cmd/tmp_` et
      `cmd/wf_` de `longestRunAllowedPrefixes`. **Une allowlist dont la cible a disparu est un
      trou latent** — le self-check `TestHalowaypointAllowlistEntriesPointToExistingFiles` le
      dit déjà pour son propre ratchet.
      *FAIT : les trois entrées retirées, commentaires de retrait datés du 2026-08-01. Les deux
      préfixes `internal/analysis/filmdec/*` restent : leur condition de retrait (une primitive
      de plage de bits propre à `filmdec`) n'est pas remplie.*
- [x] A4. Re-mesurer (§1). Gate : `go build ./...` + `go test ./...` verts.
      *FAIT : `go build ./...` OK, `go test ./...` vert (0 échec), `internal/archlint` vert.
      **Mesure : 70 → 62** — exactement les 8 `unconvert` de l'outillage jetable.*

### Lot B — la passe mécanique  *(~19 issues, zéro risque pour les mesures)*

`unconvert`, `goconst`, `revive` (nommage), `unparam`, `errcheck`, `ineffassign`, `prealloc`.

- [x] B1. Traiter linter par linter, pas fichier par fichier — c'est plus rapide et le diff se
      relit.
      *FAIT 2026-08-01, dans cet ordre : `unconvert` (3), `goconst` (3), `revive` var-naming (3),
      `unparam` (7 + 1 en cascade), `errcheck` (1), `ineffassign` (1), `prealloc` (2).*
      **`revive` `argument-limit` (3) NON TRAITÉ `[!]`** — ce n'est pas de la passe mécanique :
      rentrer sous le plafond demande de regrouper des paramètres dans une structure, donc de
      toucher des signatures de décodage pour une raison de forme. Renvoyé à la session 2.
      Cibles : `kfBetterCand` (keyframe_world.go, 8), `fire_scanner_v3.go:120` (9),
      `killsource/bijection.go:259` (9).
- [x] B2. **Gate non négociable après chaque linter traité** : reconstruire les trois artefacts
      de rejeu et vérifier qu'ils sont **identiques** (tableau du §5 de
      `PLAN_RECONCILIATION_BRANCHES.md`), plus `go test ./internal/games/halo_infinite/film/killsource/...`.
      Un `unconvert` mal appliqué sur une largeur de bits change silencieusement un décodage.
      *FAIT : gate joué 5 fois (une par palier de linter, plus le gate à blanc d'entrée et le
      gate final). Les 7 grandeurs des 3 films sont restées identiques à l'unité près à CHAQUE
      passage. `go test ./...` vert en entrée et en sortie.*
      **Note de méthode** : les états d'inventaire ne sont pas journalisés par `replay-build` ;
      les relever sur l'artefact (`jq '.inventory|length'`) — sans quoi une des 7 grandeurs du
      §5 n'est pas vérifiée du tout.
- [x] B3. Re-mesurer (§1). **62 → 43.**

### Lot C — les `unused` de `filmdec`  *(14 au 2026-07-31 — le seul lot qui demande du jugement)*

Ce sont des **grammaires de composants portées mais pas branchées au dispatch**
(`consumeMusicState`, `consumeTacmapPoiicon`, `consumeCrewOrder`, `consumeSelectableZoneData`,
`consumeEffectStateData`, `consumeStateBroker`, `consumeCrewMarkedObjects`, `consumeOpt6`,
`consumeObjectDeadStateBiped`, `consume1432026f4`…). C'est la forme normale du
reverse-engineering : on porte la grammaire avant d'en avoir l'usage. La règle du dépôt dit
malgré tout « 0 code mort ».

- [x] C1. Pour chacune, répondre à **une** question : *le composant est-il atteignable depuis
      `traverse.go` pour un archétype qu'on décode déjà ?*
      - **oui** → la brancher (c'est un bug de dispatch, pas du code mort) ;
      - **non, et le composant n'a pas d'usage produit identifié** → la supprimer ;
      - **non, mais elle documente une grammaire coûteuse à re-établir** → `//nolint:unused`
        avec **date + raison + condition de retrait**, comme tout kill-switch du dépôt.
      *FAIT 2026-08-01 sur les **34** (et non 14 : legs de J3-1). **0 cas (a)**, 32 cas (b),
      2 cas (c). Méthode et table complète au journal §5.*
- [x] C2. Écrire le verdict des 14 dans ce fichier (section Journal), pas seulement dans le
      code : la prochaine reprise ne doit pas re-instruire le dossier.
      *FAIT — les 34 verdicts, avec archétype, indice, couverture de dispatch et raison.*
- [x] C3. Gate : artefacts de rejeu identiques + golden killfeed verts.
      *FAIT — 7 grandeurs des 3 films identiques à l'unité près (gate à blanc d'entrée puis
      gate de sortie, tous deux mesurés dans la session) ; `replay`, `filmdec`, `killsource`,
      `objectiveevents` verts.*

### Lot D — `gocyclo` et `staticcheck`  *(le seul vrai refactor)*

- [x] D1. `internal/himap/sbsp.go` → `boundsFromTagInfo` (complexité 22, plafond 15).
      *FAIT 2026-08-01 — découpé en ses trois étapes réelles : `rootBlockIndex`,
      `rootTagBlockOffsets`, `worldBoundsFieldOffset`. Couvert par les tests `himap`.*
- [x] D2. `internal/himodule/module.go` → `(*Module).loadHd1` (16).
      *FAIT — `hd1Probes` (l'échantillon de témoins) et `hd1BaseScore` (le score d'une base)
      sortent de la fonction. **`internal/himodule` n'a AUCUN test** : l'équivalence est par
      lecture (les deux extraits sont verbatim), pas par exécution. Consigné en Découvertes.*
- [x] D3. Les 4 `staticcheck` — à lire, ils signalent souvent un vrai défaut.
      *LUS D'ABORD, et **aucun ne signale un défaut de logique**. Les deux `SA9003` (branche
      vide) sont des portes dont le corps est vide PAR CONSTRUCTION — la branche prise appelle
      `FUN_14076e494`, un résolveur d'id qui ne lit aucun bit ; le bit de porte, lui, se
      consomme. Le `if` disparaît, le `ReadBit` reste, et le commentaire dit désormais pourquoi
      il n'y a rien à lire dedans. Les deux `QF1007` deviennent une expression booléenne
      directe, avec la borne enfin écrite en clair (`ID5` sur 0..3, `B1D` sur 0..1 — hors
      bornes = record lu de travers). Correctifs triviaux ET prouvés par le gate.*
- [x] D4. Gate : artefacts de rejeu identiques + `go test ./...`.
      *FAIT — 7 grandeurs identiques ; `go test ./...` vert en sortie de session.*

### Les 3 `revive argument-limit` — hérités de J3-1

- [x] Regrouper les paramètres de décodage en structure. *FAIT 2026-08-01. Dans les trois cas
      les valeurs groupées sont de MÊME TYPE et indexées par la même chose — le cas exact où
      une inversion passe le compilateur sans bruit. `kfCand` porte les quatre clés de tri
      d'une ancre keyframe (`betterThan` devient une méthode) ; `fireSite` les quatre positions
      de bits d'un fire-event et `fireWeapon` ses deux formes d'identifiant ; `hungarianState`
      le jeu de travail de l'algorithme hongrois (`step` devient une méthode).*

### Lot E — le reste du go/no-go, qui n'est PAS du lint

Ces points dérivent tout seuls pendant qu'on travaille. À revérifier **juste avant** le merge,
jamais « une bonne fois » :

- [ ] E1. `openapi.yaml` régénéré (`go run ./cmd/openapi-gen`) **et** `generated.ts`
      (`npm run generate-types`). Deux tests le vérifient : `TestOpenAPIYAMLIsUpToDate`,
      `TestContractRoutesDocumented`.
- [ ] E2. `routeTree.gen.ts` régénéré par le générateur du projet (jamais édité à la main)
      dès qu'un fichier de `src/routes/` bouge.
- [ ] E3. `npx knip` vert — tout export non consommé le fait rougir, y compris un helper
      ajouté « pour dans deux jours ».
- [ ] E4. `npx vitest run` + `npx tsc --noEmit` + `npm run lint` (0 **erreur** ; les 19 warnings
      react-compiler/TanStack Table sont préexistants et hors périmètre).
- [ ] E5. `go test ./...` **et** `go test -tags=integration ./...` (les tests persist anti-ART
      sont obligatoires avant toute livraison touchant sync/persist).
- [ ] E6. `govulncheck ./...` — au 2026-07-31 il remonte 11 CVE de la bibliothèque standard
      **Go 1.26.1, corrigées en 1.26.2**. Ce n'est pas de la dette de code : c'est la version de
      la chaîne d'outils. **Monter Go avant le merge** et re-mesurer ; ce qui resterait alors
      serait, lui, à traiter.
- [ ] E7. Les quatre ratchets `archlint` / sentinel ADR 0023 (cf. journal de
      `PLAN_RECONCILIATION_BRANCHES.md`) : re-vérifier qu'aucune allowlist n'a grossi sans
      justification datée depuis.
- [ ] E8. Entrée `.ai/thought_log.md` + rotation trimestrielle à re-appliquer (elle a été
      volontairement suspendue à la réconciliation pour ne rien perdre) + rangement de `.ai/`.

### Lot F — la clôture

- [ ] F1. Compteur `golangci-lint --new-from-merge-base=origin/main` = **0** sur
      `feat/replay2d-prod`. C'est la définition de « prêt ».
- [ ] F2. Rebaser ou merger `origin/main` à jour dans la branche, **puis re-mesurer** : la base
      du ratchet bouge avec `main`, un compteur à 0 hier peut ne plus l'être.
- [ ] F3. **Prévenir avant le push sur `main`** — push sur `main` = déploiement prod
      automatique (`docs/RUNBOOK_GO_LIVE*`).

---

## 3. CE QUI VA S'AJOUTER — et comment ne pas se faire déborder

Le travail continue sur cette branche. Trois règles pour que la dette ne re-croisse pas plus
vite qu'on la solde :

1. **Mesurer avant de commencer une séance, pas seulement avant de merger.** Le delta d'une
   séance se corrige en minutes ; le cumul d'un mois se corrige en jours.
2. **Un lot commencé est un lot fini** (contrat `plan-execution`). Ne pas picorer les issues
   faciles de plusieurs lots.
3. **Toute nouvelle allowlist / tout nouveau `//nolint` porte une date, une raison et une
   condition de retrait.** Sans les trois, c'est un trou permanent — c'est exactement le
   mécanisme par lequel ces 48 issues sont apparues sans que personne ne le décide.

---

## 4. CE QUI N'EST PAS DANS CE PLAN

- Les **19 warnings eslint** (`react-hooks/incompatible-library` sur `useReactTable`) :
  préexistants sur `main`, non introduits par la réconciliation, hors périmètre.
- La suite fonctionnelle du rejeu : `PLAN_FINALISATION_REJEU_2D.md` (le fil des éliminations,
  débloqué par la réconciliation, est le lot 5).
- Le retrait du garde local du rejeu : il porte déjà son critère mesurable
  (`handlers/replay_local_gate.go`), il se retirera de lui-même quand le critère sera atteint.

---

## 5. JOURNAL D'EXÉCUTION

### 2026-08-01 — session J3-1 : lots A et B (mécanique) + hygiène J3.1/J3.2/J3.6

**Mesure d'entrée (§1, cache golangci purgé d'abord — J3.1 `[x]`)** : **70 issues**,
8 dans `cmd/tmp_*`/`cmd/wf_*`, 62 dans `internal/`. Conforme à la référence d'après J1.3.
Répartition : `unused` 33 · `unconvert` 11 · `unparam` 7 · `revive` 6 · `staticcheck` 4 ·
`goconst` 3 · `prealloc` 2 · `gocyclo` 2 · `ineffassign` 1 · `errcheck` 1.

**Gate à blanc AVANT toute modification** (pour que le filet ait une valeur de référence
mesurée dans cette session, pas recopiée) : les 3 artefacts reconstruits, tous les chiffres du
§5 de `PLAN_RECONCILIATION_BRANCHES.md` conformes à l'unité près —
`000d5950` 99/29 221 · 475/519 · 90/105 · 70/70 · 439 projectiles · 184 états d'inventaire sur
24 images-clés · 10 223 emprises ; `01e1f945` 1 862/2 154 ; `64e8adfa` 2 312/2 879.
Les états d'inventaire ne sont PAS journalisés par `replay-build` : ils se relèvent sur
l'artefact (`jq '.inventory|length'` et `[.inventory[].t]|unique|length`).

**Lot A** : `[x]` A1-A4. 462 fichiers / 405 répertoires supprimés, 3 entrées d'allowlist
retirées. **70 → 62.**

**J3.2** `[x]` — jeu d'images de grenades orphelin supprimé : 8 PNG en casse majuscule
(`Dynamo`/`Frag`/`Plasma`/`Spike` × `-light`/`-dark`), lignée `killsource`. Vérifié sur pièces
que `static/grenades-assets/halo_infinite/index.json` ne désigne QUE le jeu minuscule
(`dynamo_light.png`…) et qu'aucun `.ts`/`.tsx`/`.go` ne les référence.

**J3.6** `[!]` **NON TRAITÉ — la prémisse du plan est fausse, action refusée.**
`js-yaml` en `overrides` d'`apps/web` n'est pas une dépendance fantôme : c'est un **correctif
de sécurité** posé le 2026-06-22 (commit `54a6eb3df`) pour **CVE-2026-53550 /
GHSA-h67p-54hq-rp68** (DoS par complexité quadratique, js-yaml ≤ 4.1.1). `@redocly/openapi-core`
(dép. transitive dev via `openapi-typescript`) l'épingle en **exact `4.1.1`** ; l'`overrides`
est le seul levier qui hisse le lockfile à `4.3.0`. **Le retirer réintroduirait la
vulnérabilité** et rouvrirait l'alerte Dependabot #3. La découverte J2-D5 disait seulement que
le test `replayContract.test.ts` ne devait pas *s'appuyer* sur un paquet installé par un
tiers — ce qui est vrai, et déjà appliqué. À trancher par le superviseur ; ne rien faire est
la bonne action par défaut.

**Lot B** : `[x]` B1-B3, **sauf `revive` `argument-limit` (3) en `[!]`** (renvoyé session 2,
justification dans B1). Les 7 linters mécaniques du périmètre sont à **zéro** :
`unconvert` 11→0 · `goconst` 3→0 · `revive` var-naming 3→0 · `unparam` 7→0 · `errcheck` 1→0 ·
`ineffassign` 1→0 · `prealloc` 2→0. **62 → 43.**

### MESURE DE SORTIE DE SESSION — 43 issues (contre 70 à l'entrée)

| linter | avant | après | lot |
|---|---:|---:|---|
| `unused` | 33 | **34** | C — session 2 |
| `unconvert` | 11 | 0 | B ✔ |
| `unparam` | 7 | 0 | B ✔ |
| `revive` | 6 | **3** | var-naming fait ; `argument-limit` → session 2 |
| `staticcheck` | 4 | 4 | D — session 2 |
| `goconst` | 3 | 0 | B ✔ |
| `prealloc` | 2 | 0 | B ✔ |
| `gocyclo` | 2 | 2 | D — session 2 |
| `ineffassign` | 1 | 0 | B ✔ |
| `errcheck` | 1 | 0 | B ✔ |
| **total** | **70** | **43** | |

**`unused` monte de 33 à 34, et ce n'est PAS une régression** : golangci ne rend qu'une issue
par position. `consumeForgePlayerDataEditedObjectsIDs` portait à la fois un `revive`
var-naming et un `unused` à `components_batch8.go:94:6` ; le var-naming corrigé, l'`unused`
apparaît. Il était là avant. **Conséquence pour la session 2 : le lot C compte 34 verdicts,
pas 33.**

### DÉCOUVERTES DE CETTE SESSION — consignées, NON traitées

1. **`consumeQuantVec3` et `consumeQuantVec3WithGate` sont des quasi-doublons**
   (`components_movement.go`) : même grammaire FUN_14076e524, seule la porte `precHigh`
   diffère. Hors périmètre de la passe mécanique — à instruire quand le lot C ouvrira
   `filmdec`.
2. **~30 fichiers de `internal/` citent un `cmd/tmp_*` supprimé** en commentaire de
   PROVENANCE (« mesure : cmd/tmp_hdrtruth », « sweepé à l'oracle, cmd/tmp_reccheck »). Ces
   citations restent VRAIES — elles nomment l'outil qui a établi le fait, et git le garde.
   Décision prise : ne pas les réécrire (le diff serait énorme et perdrait la traçabilité de
   la mesure). Deux exceptions à surveiller si elles gênent un jour : un « go run
   ./cmd/tmp_tagname » en instruction d'exécution dans `damagetag.go`, et l'en-tête de
   `keyframe_world.go` qui décrit `WalkKeyframeWorld` comme le portage de `tmp_kfworldpos`.
3. **Les règles `.gitignore` `apps/go-api/cmd/tmp_*/` et `cmd/tmpdbq/` sont conservées** :
   elles n'ont jamais empêché le suivi (les fichiers étaient tracés), et elles restent un
   filet utile si une session de recherche recrée un binaire jetable. Ce n'est pas une
   allowlist — pas de trou latent.

---

### 2026-08-01 — session J3-2 : lots C et D + les 3 `argument-limit` + J3.3/J3.4/J3.5

**Mesure d'entrée (§1, cache golangci purgé)** : **43 issues**, conformes à la sortie de J3-1
à l'unité près — `unused` 34 · `staticcheck` 4 · `revive argument-limit` 3 · `gocyclo` 2.

> **CORRECTIF DE LA PROCÉDURE §1** : la commande de comptage écrite plus haut,
> `grep -E "^[a-zA-Z_/\\]+\.go:..."`, **sous-compte** — sa classe de caractères n'admet pas
> les CHIFFRES, donc tout fichier au nom numéroté (`components_batch3.go`,
> `components_batch8.go`…) tombe. Elle rendait **21** au lieu de 43 sur cette même sortie.
> Compter avec `grep -cE "^[^ ].*\.go:[0-9]+:[0-9]+:"`.

**Gate à blanc AVANT toute modification** (référence mesurée dans CETTE session) : les 3
artefacts reconstruits, les 7 grandeurs du §5 de `PLAN_RECONCILIATION_BRANCHES.md` conformes —
`000d5950` 99/29 221 · 475/519 · 90/105 · 70/70 · 439 projectiles · 184 états d'inventaire sur
24 images-clés · 10 223 emprises ; `01e1f945` 1 862/2 154 ; `64e8adfa` 2 312/2 879.

#### Lot C — les 34 `unused` : la méthode, puis les 34 verdicts

**Ce qui tranche (a), et pourquoi aucun des 34 n'en relève.** Le traverseur s'arrête au
**premier composant présent non porté** (`traverse.go`, `t.DesyncAt = i` puis sortie de
boucle ; `unportedStubWidth` est vide hors banc de calibration). « Atteignable » se décide
donc par un INDICE, pas par une intuition : un lecteur du composant i<sub>k</sub> ne change
rien si un trou existe à un indice **antérieur**. Croisement fait sur pièces entre le registre
ECS (`cmd/rdata_weapon_scan`, 118 archétypes / **1 067** composants — le compte de J0.4) et les
188 noms du `switch consumeByName`. Couverture de dispatch relevée :

| archétype | couverture | archétype | couverture |
|---|---|---|---|
| ti=6 · ti=18 · ti=21 · ti=29 | **complets** | ti=0 / ti=2 (game-engine) | 12/27 · 12/18, **1er trou à i11** |
| ti=35 biped | 59/64 | ti=12 navpoints | 1/28, **1er trou à i1** |
| ti=37 · ti=41 · ti=42 | 30/31 · 21/22 · 20/21 | ti=19 · ti=23 · ti=26 · ti=31 · ti=46 · ti=48 | **0/N** |
| ti=40 véhicules · ti=43 dispositifs | 31/48 · 18/41 | ti=11 objectifs | **0/34** |

Et le gate du lot exige des **artefacts identiques** : un branchement qui améliorerait le
décodage le ferait ÉCHOUER. Améliorer le décodage n'est pas de la dette — c'est J6-A.

**Le garde-fou du superviseur appliqué** : un lecteur d'un composant d'archétype PLANIFIÉ
(objectifs ti=11, zones ti=23, véhicules ti=40, dispositifs ti=43) est un (c), jamais un (b).
Deux des 34 sont dans ce cas. **Aucun des 34 ne concerne ti=40 ni ti=43** — vérifié : pas un
seul lecteur `vehicle-*` ni `device-*` dans la liste.

**Critère (b) vs (c) hors garde-fou, écrit une fois** : un `//nolint` n'est honnête que si sa
**condition de retrait est évaluable** par une session future, c'est-à-dire adossée à un
travail NOMMÉ dans un plan. Sans cela, c'est un trou permanent (anti-pattern n°2 du dépôt) et
le code part — `git` le garde, avec l'adresse `FUN_` qui a servi à l'établir.

**(c) — 2 lecteurs gardés sous `//nolint:unused` daté**

| symbole | fichier | composant | ti/i | condition de retrait |
|---|---|---|---|---|
| `consumeObjectiveFormattedText` | `components_batch3.go` | `managed-objective-formatted-text-component` + son jumeau `-secondary-` | **11**/i2, i9 | branchée ou retirée quand ti=11 sera décodé (J6-A §4 « Objectifs étape 3 ») |
| `consumeSelectableZoneData` | `components_world.go` | `selectable-zone-data-component` | **23**/i0..i31 | branchée ou retirée quand ti=23 sera décodé (J6-A §4, zones par images-clés + footer type-3) |

**(b) — 32 supprimés, en trois familles**

*Famille 1 — SUPPLANTÉS (17). Le nom du composant est déjà dispatché : `traverse.go` porte la
grammaire, INLINE et depuis la table ECS live de J0.4. L'orphelin est un port antérieur. Les
rebrancher serait un retour en arrière — et pour cinq d'entre eux, un retour à une approche
explicitement écartée (largeurs fixes `TraversalPrecision` au lieu de la quantification
6+level ; devinette de bits au lieu de la désynchronisation propre).*

| symbole | fichier | composant (ti/i) | port vivant |
|---|---|---|---|
| `consumeEquipmentTrackedStack` | batch3 | equipment-tracked-object-handles-stack (37/i28) | `consumeEquipmentTrackedStack2` |
| `consumeGameEngineAlliance` | batch4 | game-engine-alliance (0·1·2/i10) | inline |
| `consumeTrackFrame` | batch5 | track-frame (16/i5) | `consumeTrackFrameComponent` |
| `consumeCrewOrdersOffFlags` | batch8 | crew-orders-off-flags (14/i2) | inline R(8) |
| `consumeMusicVariables` | batch8 | music-variables (17/i0) | inline |
| `consumeEquipmentHasInfiniteUses` | batch8 | equipment-has-infinite-uses (37/i30) | inline R(1) |
| `consumeBipedEmpTimer` | batch8 | biped-emp-timer (35/i51) | inline R(8) |
| `consumeGameEngineCurrentState` | batch8 | game-engine-current-state (0·1·2/i2) | inline R(3) |
| `consumeGameEngineGameFinished` | batch8 | game-engine-game-finished (0·1·2/i3) | inline R(1) — **et la table live corrige le `ti=0` écrit dans batch8 : c'est ti=2 i3** |
| `consumeStatborgEntryIndexAndType` | batch8 | statborg-entry-index-and-type (6/i57) | inline R(32)+R(8) |
| `consumeEffectStateData` | world | effect-state-data (18/i0..31) | inline |
| `consumeMusicState` | world | music-state (17/i1) | inline |
| `consumeTacmapPoiicon` | world | tacmap-poiicon (30/i0) | inline (vec3 quant 6+level) |
| `consumeCrewMarkedObjects` | world | crew-marked-objects (14/i1) | inline |
| `consumeCrewOrder` | world | crew-order (14/i0) | inline (`consumeQuantVec3`, 6+level) |
| `consumeObjectPositionDynamicPrecision` | object | object-position-dynamic-precision (35·40/i0) | `consumeObjectPositionDynamicPrecisionD` — le port bit-exact ; l'orphelin portait le **6/6/6 fautif** et disait lui-même « should be empirically re-validated » |
| `consumeObjectDeadStateBiped` | object | object-dead-state (35/i11) | `consumeObjectDeadStateBipedTI(br, 0x23)` — l'orphelin n'était que la façade qui fixait le typeIndex |

*Famille 2 — ARCHÉTYPE NI DÉCODÉ NI PLANIFIÉ (11). Aucune condition de retrait évaluable.*

| symbole | fichier | composant (ti/i) | fait qui tranche |
|---|---|---|---|
| `consumeNavpointFormattedText` | batch3 | managed-navpoint-formatted-text (**12**/i9) | trou antérieur à i1 |
| `consumeSupplyLinesBusy` | batch4 | supply-lines-busy-state (**26**/i32) | trou à i0 ; **aucune adresse `FUN_`** au commentaire |
| `consumeLowFrequency` | batch4 | `low-frequency` (**3**/i0) | **aucune adresse `FUN_`** ; les trois vrais low-frequency (object/unit/biped) ont chacun leur port vivant et validé |
| `consumeGameEngineDisabledKillVolume` | batch5 | game-engine-disabled-kill-volume-flags (**0·2**/i13) | trou antérieur à i11 (`game-engine-soft-ceilings`) |
| `consumeTacmapCategory` | batch5 | tacmap-category (**31**/i0) | 1/11 — les 10 autres tacmap-* manquent |
| `consumeSoundPlacementStateData` | batch8 | sound-placement-state-data (**19**/i0..31) | 0/32, et rien ne planifie ti=19 — voir Découverte n°1 |
| `consumeStateBrokerStateChangedData` | batch8 | state-broker-state-changed-data (**46**/i0..63) | 0/64, hors World, hors piste — voir Découverte n°1 |
| `consumeStateBroker` | world | **le même composant** | **doublon** du précédent, et le moins documenté des deux |
| `consumeForgePlayerDataEditedObjectsIDs` | batch8 | forge-player-data-edited-objects-ids (**48**/i0) | 1/2 : `forge-player-data-editing-graphs` manque |
| `forgeEditedEntry` | batch8 | sous-lecteur du précédent | suit son appelant |
| `consume1432026f4` | biped_ability | corps annoncé des tags 9/10 d'i63 | **PRÉMISSE RÉFUTÉE** : la vérité EXE du 2026-06-13, inscrite dans le `default` de `consumeBipedActionTag`, est que le dispatch ne gère QUE 0..5 et que tout tag ≥ 6 consomme **zéro bit**. Le garder exposait à re-câbler une lecture connue fausse |

*Famille 3 — PRIMITIVES SANS COÛT DE RÉTABLISSEMENT (4). Le fait survit au retrait.*

| symbole | fichier | fait conservé |
|---|---|---|
| `deadStateMortOffset` (const) | object | l'offset `0x70` est déjà écrit à son unique point d'usage (`ds.Mort = br.ReadBit() // comp+0x70`) |
| `consumeOpt6` | object | « `FUN_1406d1024` -> R(1); if 0 -> R(6) » est dans la table de primitives en tête du fichier |
| `readBitsBE` | objectiveevents | lecture MSB-first générique, 12 lignes |
| `extractType2` | objectiveevents | `walkFrames`, dans le même fichier, parcourt le même conteneur |

**Le quasi-doublon `consumeQuantVec3` / `consumeQuantVec3WithGate` (découverte J3-1) est
STATUÉ : ce n'en est pas un.** Malgré son suffixe, `WithGate` a **une porte de moins** — c'est
`consumeQuantVec3` qui ouvre par la porte `precHigh` à sortie immédiate (0 bit). Les deux
comptes de bits diffèrent, les deux ont des appelants vivants (flock d'un côté, cinq sites du
dispatch de l'autre), et les fusionner changerait un décodage. Aucune action de code : le
piège de nommage est documenté au site (`components_movement.go`). **Le renommer serait la
suite logique — non fait, hors périmètre J3-2 (aucun gain de dette).**

#### J3.3 / J3.4 / J3.5 — ce qui a été trouvé en les traitant

**J3.3 `[x]`** — le hook pre-push `knip-ratchet` est glob-gaté sur `apps/web/src/**`. **Élargir
le glob n'aurait rien réglé** : le mode d'échec réellement constaté en J1-b n'est pas « le push
ne touchait pas de source web », c'est « la branche accumulait des exports morts et n'a JAMAIS
été poussée » — aucun glob ne rattrape une branche non poussée. Le ratchet devient donc un step
du job `frontend` de `ci.yml`, qui n'a pas de filtre de chemin. Vérifié depuis `apps/web` (le
cwd du job) : le chemin relatif résout, le ratchet sort en 0. Le hook local reste en filet
rapide, et son commentaire dit maintenant qu'il n'est PAS l'autorité.

**J3.4 `[x]`** — garde de longueur posée sur `decodeFireEvent` (qui rend désormais `ok bool`),
`PeekBits` aligné sur son contrat (la tolérance valait à la fin, pas au début), deux tests de
régression dédiés, et le harnais de fuzz appelle maintenant les lecteurs avec EXACTEMENT le
contrat de la production — ses deux contournements sont retirés. L'entrée de crash conservée en
régression est la graine `seed_04` (troncature à trois octets d'un payload de tir), produite par
`collectFuzzSeeds` donc régénérable. **Audit des frères : ils sont sains** — `scanGrenadeThrows`
et `scanProjectileRecords` bornent leur balayage ET lisent par `PeekBits` ; `offline_aim.go`
teste `at+n > total` avant chaque composant ; `offline_biped.go` et `i0_layout.go` bornent leur
boucle sur `total`. `decodeFireEvent` était le seul à lire à offsets FIXES derrière une garde de
taille qui ne les couvrait pas.

**J3.5 `[x]` — et la cause annoncée n'était que la moitié de l'histoire.** Diff champ par champ
de deux constructions du même film : exactement DEUX champs bougent. `projectiles` a le même
multi-ensemble à l'ordre près (instabilité de tri, comme annoncé). Mais **`grenades` change de
VALEUR** — le lancer `t=1580` de `01e1f945` sort à (11,41 ; 17,99) ou à (12,72 ; −187,11) selon
l'exécution. La chaîne est unique : l'itération de la map de `ScanFilmWorldObjects` fixe l'ordre
d'arrivée, un tri sur le seul instant laisse cet aléa départager les naissances de projectile
EX ÆQUO, et `birthNear` prend la naissance d'un INDICE donné — l'aléa choisissait donc la
position publiée. Les quatre tris deviennent totaux (`filmdec` : échantillons et vies ;
`replay` : naissances et lancers, plus les projectiles publiés).

**Preuve, les deux volets** : deux constructions successives de chacun des trois films rendent
des octets identiques (`54a4fdf5…`, `55e169d0…`, `c17f7a91…`) ; et le fixture d'entrées des
goldens, régénéré deux fois, rend la même empreinte — alors qu'il DIFFÉRAIT de la version
committée, laquelle n'était donc qu'un tirage parmi d'autres. Le fixture stabilisé est committé ;
**la sortie figée d'assemblage, elle, est INCHANGÉE**.

#### Découvertes de cette session — consignées, NON traitées

1. **UN SEUL LECTEUR PORTERAIT UN ARCHÉTYPE ENTIER — deux cas, à instruire en J6-A.** Certains
   archétypes ne répètent qu'un composant : `ti=19` = 32 × `sound-placement-state-data`,
   `ti=46` = 64 × `state-broker-state-changed-data`, `ti=23` = 32 × `selectable-zone-data`.
   Y câbler LE lecteur ferait passer l'archétype de 0/N à N/N d'un seul coup. Pour ti=23
   (zones) le lecteur est gardé sous `//nolint` (archétype planifié) ; pour **ti=19 (présent
   dans le World de `000d5950`, slot 118) et ti=46**, les lecteurs ont été supprimés faute de
   piste nommée — le FAIT est ici, et `git` garde le code (commit du lot C). Si J6-A veut
   remonter le taux de marche propre des FRAMES, ce sont deux leviers à un câble chacun.
2. **Le compteur de la procédure §1 sous-comptait** (classe de caractères sans chiffres) —
   corrigé en tête de ce journal. Toute mesure antérieure lue par cette commande est suspecte.
3. **UN GARDE DE CI NE SE DÉCLENCHE JAMAIS — le plus sérieux de la liste.** Dans `ci.yml`, job
   `frontend`, step « Guard — feedback-drawer ne doit pas importer le wrapper api » : le job
   déclare `defaults.run.working-directory: apps/web`, et le `grep` du step vise
   `apps/web/src/features/feedback-drawer/queries.ts` — soit, depuis ce cwd,
   `apps/web/apps/web/src/…`, qui n'existe pas. `grep` sort en 2, la condition est fausse, **le
   step passe toujours**. Vérifié sur pièces (`apps/web/apps` n'existe pas). Ce garde protège
   d'une fuite de `X-LevelUp-Title` et des cookies de session vers GitHub — il ne protège rien
   aujourd'hui. Le correctif est le retrait du préfixe `apps/web/` ; **non fait, hors périmètre
   J3-2** (aucun rapport avec la dette de lint). À traiter en priorité.
4. **Les plafonds du ratchet knip sont périmés** : `files=29 · exports=90 · types=86`, alors que
   le compte réel est **0 / 0 / 0** (knip 6.29.0, vérifié qu'il analyse bien — il rend des
   *hints* sur des fichiers réels, ce n'est pas un scan vide). Les abaisser verrouillerait le
   gain, mais durcirait le gate pour tout travail en cours : décision de superviseur, pas un
   effet de bord de J3.3.
5. **`internal/himodule` n'a aucun fichier de test**, et le paquet lit un format binaire du jeu
   (`.module` / `.module_hd1`) avec calibration de base par score. Le découpage de `loadHd1`
   (D2) est donc justifié par LECTURE, pas par exécution — les artefacts de rejeu ne passent pas
   par ce paquet (la structure de carte vient d'un JSON versionné). Chantier piste J6-B.
6. **`internal/analysis/weaponv3` n'est consommé que par `cmd/diag_weapons_v3`**, et le même
   `cmd` est le seul consommateur d'`internal/analysis/objectiveevents` — c'est la lignée « v3
   shadow », jamais promue. Son retrait est un chantier à part entière (il emporterait deux
   paquets et un binaire) : **non instruit ici**, mais c'est le plus gros gisement de code mort
   restant, et il échappe à `unused` parce qu'un `cmd` le référence.
7. **Le départage des naissances de projectile ex æquo est ARBITRAIRE, faute de mieux.** J3.5 le
   rend stable (par coordonnée), pas *juste* : deux grenades lancées au même instant de
   réplication par deux joueurs différents peuvent voir leurs naissances permutées. Le bon
   départage passerait par le propriétaire du projectile, qui n'est pas décodé. Piste J6-A.

### MESURE DE SORTIE DE SESSION — **0 issue** (contre 43 à l'entrée)

| linter | entrée J3-2 | sortie J3-2 | traitement |
|---|---:|---:|---|
| `unused` | 34 | **0** | lot C — 32 retirés, 2 sous `//nolint` daté avec condition de retrait |
| `staticcheck` | 4 | **0** | lot D — lus d'abord, aucun défaut de logique |
| `revive` (`argument-limit`) | 3 | **0** | paramètres regroupés en structure |
| `gocyclo` | 2 | **0** | lot D — découpage réel |
| **total** | **43** | **0** | |

`golangci-lint run --new-from-merge-base=origin/main`, cache purgé avant la mesure : **`0 issues.`**
C'est la cible **F1** du lot F. Les deux seuls survivants du dépôt sont les deux `//nolint:unused`
du lot C, chacun daté, motivé et porteur d'une condition de retrait évaluable.

**Ce qui RESTE au lot F et ne peut pas être fait ici** : F2 — rebaser/merger `origin/main` à jour
puis **re-mesurer** (la base du ratchet bouge avec `main`, un 0 d'aujourd'hui n'est pas un 0 de
demain) ; F3 — prévenir avant le push sur `main`. Et tout le lot E, qui dérive tout seul.

> **J3 EST CLOS. 70 → 43 → 0.** J3-1 a soldé le mécanique, J3-2 le jugement et les gardes.
