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

- [ ] B1. Traiter linter par linter, pas fichier par fichier — c'est plus rapide et le diff se
      relit.
- [ ] B2. **Gate non négociable après chaque linter traité** : reconstruire les trois artefacts
      de rejeu et vérifier qu'ils sont **identiques** (tableau du §5 de
      `PLAN_RECONCILIATION_BRANCHES.md`), plus `go test ./internal/games/halo_infinite/film/killsource/...`.
      Un `unconvert` mal appliqué sur une largeur de bits change silencieusement un décodage.
- [ ] B3. Re-mesurer (§1).

### Lot C — les `unused` de `filmdec`  *(14 au 2026-07-31 — le seul lot qui demande du jugement)*

Ce sont des **grammaires de composants portées mais pas branchées au dispatch**
(`consumeMusicState`, `consumeTacmapPoiicon`, `consumeCrewOrder`, `consumeSelectableZoneData`,
`consumeEffectStateData`, `consumeStateBroker`, `consumeCrewMarkedObjects`, `consumeOpt6`,
`consumeObjectDeadStateBiped`, `consume1432026f4`…). C'est la forme normale du
reverse-engineering : on porte la grammaire avant d'en avoir l'usage. La règle du dépôt dit
malgré tout « 0 code mort ».

- [ ] C1. Pour chacune, répondre à **une** question : *le composant est-il atteignable depuis
      `traverse.go` pour un archétype qu'on décode déjà ?*
      - **oui** → la brancher (c'est un bug de dispatch, pas du code mort) ;
      - **non, et le composant n'a pas d'usage produit identifié** → la supprimer ;
      - **non, mais elle documente une grammaire coûteuse à re-établir** → `//nolint:unused`
        avec **date + raison + condition de retrait**, comme tout kill-switch du dépôt.
- [ ] C2. Écrire le verdict des 14 dans ce fichier (section Journal), pas seulement dans le
      code : la prochaine reprise ne doit pas re-instruire le dossier.
- [ ] C3. Gate : artefacts de rejeu identiques + golden killfeed verts.

### Lot D — `gocyclo` et `staticcheck`  *(le seul vrai refactor)*

- [ ] D1. `internal/himap/sbsp.go` → `boundsFromTagInfo` (complexité 22, plafond 15).
- [ ] D2. `internal/himodule/module.go` → `(*Module).loadHd1` (16).
- [ ] D3. Les 4 `staticcheck` — à lire, ils signalent souvent un vrai défaut.
- [ ] D4. Gate : artefacts de rejeu identiques + `go test ./...`.

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
