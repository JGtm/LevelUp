# Plan — Fonds de carte : ETag/304 et passage au WebP sans perte

> **Exécution sous le contrat du skill `plan-execution`** — ordre strict, aucun report, chaque
> item statué. Ce plan ne redit pas le contrat, il s'y soumet.
> Point de vigilance transverse imposé par le commanditaire : **l'enregistreur / export vidéo du
> rejeu**. Il est traité par la décision D1 (sans perte) et par l'étape 5 (recette navigateur).

## Objectif et critère de succès

Réduire le poids des 109 fonds de carte (44 Mo de PNG) et supprimer les retéléchargements
inutiles, **sans re-cuisiner une seule image depuis le jeu** (aucune installation de Halo requise)
et **sans dégrader d'un pixel** le rendu du rejeu ni celui de l'export vidéo.

**Critère de succès, vérifiable :**

1. `GET .../replay/background.png` répond `304` sur une seconde requête portant `If-None-Match`.
2. Les 109 fonds sont des WebP sans perte dont le décodage redonne **exactement** les pixels du
   PNG d'origine (égalité bit à bit du `image.RGBA`), pour un gain total >= 20 %.
3. La recette `scripts/recette_export_rejeu.js` rend ses verdicts — les 11 existants **plus un
   nouveau sur le fond de carte** — tous au vert, en une seule passe navigateur.

**Effort** : moyen. Les étapes 1 à 3 sont du code sans effet observable ; l'étape 4 bascule la
donnée ; l'étape 5 est de la recette manuelle.
**Branche** : `feat/fonds-carte-webp-etag` depuis `feat/v75`.
**Worktree** : `LevelUp-wt-fonds-webp` / `wt/fonds-webp` — dédié, conformément à la règle du dépôt
(ne jamais travailler dans le worktree partagé de session).

## Ce que l'enquête préalable a établi (ne pas ré-instruire)

| Constat | Source vérifiée |
|---|---|
| L'export vidéo **ne refait aucune requête ni aucun décodage** : il rappelle `redraw()` sur la toile de l'écran, puis `new VideoFrame(canvas)` | `features/match-replay/export/useReplayExport.ts:255-274`, `export/replayVideoEncoder.ts:415-421` |
| Pas de Web Worker ni d'`OffscreenCanvas` : tout est sur le thread principal, sur le `HTMLCanvasElement` du DOM | idem, `exportRenderScale` l. 339 |
| `decodeImage` fait `new Image()` sur un `objectURL` : le format est **sniffé au contenu**, l'extension de la route n'entre pas en jeu | `lib/replay/queries.ts:134-141`, blob l. 90-101 |
| `api.getBlob` ne force **aucune** option de cache ni en-tête conditionnel : un ETag serait géré par le cache HTTP natif | `lib/api/client.ts:317-318`, `sendRequest` l. 254-266 |
| Les handlers écrivent le blob **opaque**, sans jamais le décoder | `handlers/replay.go:148-180`, `handlers/tactical.go:281-316` |
| Les seuls `png.Decode` sont dans des **outils hors ligne**, et **aucun décodeur WebP n'est enregistré** — c'est le vrai vecteur de casse | `cmd/mapfond-planche/main.go:152`, `cmd/mapfond-cadrage/main.go:122`, `internal/mapdecoupe/masque.go:69` |
| L'extension `.png` est **codée en dur** dans le résolveur de chemins | `internal/domain/title/registry.go:929` |
| Le sidecar porte déjà le nom de fichier de l'image dans son champ `Image`, et il est **lu avant** l'image | `internal/analysis/replay/map_background.go:31-53`, service `replay_map_background.go:111-115` |
| Les 218 fichiers sont **versionnés en clair** (pas de LFS) ; `cwebp` **n'est pas installé** sur le poste | `git ls-files`, `.gitattributes`, `command -v cwebp` |
| `handlers/assets.go:217` pose déjà un ETag **sans** gérer `If-None-Match` | `handlers/assets.go:217` |

## Décisions PRISES — rien à trancher en cours de route

| # | Décision | Valeur retenue |
|---|---|---|
| D1 | Mode de compression WebP | **Sans perte (lossless) exclusivement.** L'export vidéo agrandit la toile (`exportRenderScale`) : un artefact de compression sur du trait fin ou sur l'alpha serait amplifié dans la vidéo livrée. Le lossy est **interdit** dans ce chantier, y compris « juste pour voir » |
| D2 | Cohabitation des formats | **Remplacement.** Le `.webp` remplace le `.png`, qui est supprimé du dépôt. Garder les deux serait un « dead code museum » (anti-pattern n°1 du dépôt) et doublerait le poids du checkout. Git conserve l'historique |
| D3 | Résolution du fichier image | Le service lit le champ `Image` du **sidecar** (déjà chargé juste avant) au lieu de reconstruire `clé + ".png"`. Le format devient une propriété de la donnée, pas du code |
| D4 | Nom des routes HTTP | **Inchangé** : `.../replay/background.png` et `.../tactical/{map_id}/background.png` restent tels quels, tout en servant `image/webp`. Le front passe par un blob, l'extension de l'URL n'a aucun effet. Renommer casserait les caches et le front pour zéro gain. **Ne pas « corriger » cette URL** |
| D5 | Encodeur | `github.com/HugoSmits86/nativewebp` — pur Go, sans cgo, sans perte, avec alpha. Utilisé **uniquement** par l'outil hors ligne, jamais dans le chemin servi |
| D6 | Décodeur pour les outils hors ligne | `golang.org/x/image/webp` (pur Go, lit le VP8L sans perte), enregistré par import blanc, et `png.Decode` remplacé par `image.Decode` |
| D7 | Valeur de l'ETag | ETag **fort** = `"sha256-<12 hex du contenu servi>"`. Ni mtime ni taille : le contenu fait foi, et il est de toute façon déjà lu intégralement en mémoire |
| D8 | Portée de l'ETag | Les **trois** appelants d'un coup : `replay.go`, `tactical.go` **et** `assets.go`. C'est la 3e copie du motif, donc la règle n°6 du dépôt impose helper **plus** garde-rail. Ce n'est pas un fix opportuniste : c'est la règle |
| D9 | Nom du helper | `servirBlobAvecETag` dans `internal/api/handlers/cache_http.go`, en français comme `cleDeFondSure` / `ecritPNG` dans le code récent voisin |
| D10 | Critère d'abandon | Si l'étape 0 mesure un gain < 20 % **ou** un aller-retour non identique au bit près, **les étapes 2 à 5 sont abandonnées** et statuées `[!]`. Seule l'étape 1 (ETag) est livrée. Ne pas « rattraper » avec du lossy — voir D1 |

> **Amendement S5 (superviseur, 2026-09-09)** — fait foi sur D8/D9 ci-dessus et sur le
> constat § enquête préalable qui affirmait qu'aucun 304 n'existait dans `handlers/` :
> c'est faux, `writeJSONCached` (helpers.go) en posait déjà un, et `assets.go:217` posait
> un ETag inerte (jamais honoré, faute de lecture d'`If-None-Match`). Conséquence : le
> helper `servirBlobAvecETag` migre **les quatre** sites (replay.go, tactical.go,
> assets.go **et** writeJSONCached), pas seulement les trois du plan initial. Détail dans
> l'entrée `.ai/thought_log.md` du 2026-09-09 et le corps de l'étape 1 ci-dessous.

## Règles d'exécution

- **Ordre strict.** L'étape N+1 ne commence pas avant que le gate de N soit passé.
- **Statuts** : `[x]` fait · `[~]` couvert ailleurs (avec la référence) · `[!]` non traité (avec la
  justification écrite). **Aucune case vide à la clôture.**
- **Zéro fix hors périmètre.** Toute découverte va en § Découvertes, sans être traitée.
- **Une seule revue adversariale, en fin de chantier** — pas par étape (règle du dépôt).
- **Reprise de session** : lire les cases cochées de ce fichier, puis `git log --oneline -10` sur
  `wt/fonds-webp`. L'étape en cours est la première non cochée.

---

## Étape 0 — Banc d'essai : mesurer avant de convertir

Cette étape construit l'outil qui servira à l'étape 4 et **décide si le chantier WebP a lieu**.

**Périmètre fermé (2 fichiers + 1 dépendance) :**

- [x] `apps/go-api/go.mod` — ajouté `github.com/HugoSmits86/nativewebp v1.3.0` (D5). `go mod
      tidy` a aussi promu `golang.org/x/image v0.24.0` en dépendance directe (non `// indirect`)
      : nécessaire dès cette étape, pas seulement à l'étape 3, car le banc d'essai lui-même doit
      **redécoder** le WebP pour prouver l'aller-retour (D6 s'applique donc déjà ici, pas
      seulement aux outils de l'étape 3)
- [x] Créé `apps/go-api/cmd/mapfond-webp/main.go` — outil hors ligne, deux modes :
      `-verifier` (n'écrit rien, mesure et contrôle) et `-convertir` (écrit). Chemin résolu via
      `title.FindRepoRoot()` + `title.NewPathResolver(...).MapBackgroundDir(slug)` (jamais de
      `filepath.Join(..., "data", ...)` à la main), override possible par `-dir` (documenté dans
      le doc-comment du fichier)
- [x] L'outil, pour chaque fond : décode le PNG (`image/png`), ré-encode en WebP sans perte
      (`nativewebp.Encode`, `CompressionLevel: BestCompression` — reste sans perte dans tous les
      cas, ce champ ne joue que sur l'effort de recherche), **re-décode le WebP**
      (`golang.org/x/image/webp.Decode`) et compare le résultat à l'original canonicalisé en
      `image.RGBA` (`versRGBA`, `apps/go-api/cmd/mapfond-webp/roundtrip.go:52-68`) **octet par
      octet** (`bytes.Equal` sur les `Pix`, plus égalité des `Rect`). Toute différence est une
      erreur fatale nommant le fichier (`verifier.go:24-27` pour `-verifier`,
      `convertir.go:47-49` pour `-convertir`, qui en plus REFUSE d'écrire dans ce cas)
- [x] Journalisation `slog.InfoContext`/`slog.ErrorContext` par fichier (nom, octets avant,
      octets après, gain %, durées) — jamais de `fmt.Println` ; le seul `fmt.Fprintf(os.Stdout,
      ...)` du fichier sert le TABLEAU de mesure (`verifier.go:66-74`), qui est le PRODUIT du
      mode `-verifier`, pas une trace de diagnostic — conforme au contrat du lot
- [x] Créé `apps/go-api/cmd/mapfond-webp/main_test.go` : une image **NRGBA** synthétique 64x64
      avec zone opaque, zone totalement transparente, zone semi-transparente (alpha 120) et
      trait fin d'un pixel, qui prouve l'aller-retour identique
      (`TestAllerRetourWebP_ImageSynthetique_Identique`). NRGBA et non RGBA : une image
      `*image.RGBA` construite à la main avec des octets choisis librement peut violer la
      contrainte du modèle prémultiplié (R/G/B ≤ A) — ce qu'un aller-retour
      prémultiplie/déprémultiplie ne peut pas restituer à l'identique, ce serait un artefact du
      test et non une vraie perte de l'encodeur (constat fait en cours de TDD : la 1ʳᵉ version du
      test, en `*image.RGBA`, échouait sur la zone semi-transparente ; corrigé en NRGBA — c'est
      d'ailleurs le type concret que rend `png.Decode` sur un vrai fond RGBA+alpha). Tests
      complémentaires : `TestVersRGBA_PreserveRect`, `TestGainPct`,
      `TestSelectionnePNG_TrieParTailleDecroissanteEtLimite`, et
      `TestConvertitUnFond_EcritWebpMetAJourSidecarSupprimePNG` qui exerce **`-convertir` de bout
      en bout sur `t.TempDir()`** (écrit le WebP, met à jour `image` dans le sidecar, supprime le
      PNG) — jamais sur `data/`
- [x] Lancé `-verifier` sur **les 5 plus gros fonds** (`btb_fragmentation`,
      `28a3ac28-f69d-4fa9-9ebf-a0449c89c8da`, `37bc3df6-93e8-4d74-b16e-5ceaa30ebc23`,
      `1ede38fa-4d30-4dfa-a8b7-5d08bf4e46e3`, `305b1bdd-9a7b-4975-bacf-8bd63c8c13d2`) —
      **chiffres consignés ci-dessous, section Découvertes**
- [!] Statuer D10 par écrit : **décision utilisateur.** Les chiffres mesurés (gain cumulé 38,4 %
      sur les 5 plus gros fonds, aller-retour identique au bit près sur les 5) dépassent le seuil
      de 20 % fixé par D10 et par le gate. Sur la seule mesure de cette étape, le critère
      d'abandon de D10 n'est PAS déclenché — mais la décision de poursuivre (étapes 2 à 5) revient
      à l'utilisateur, pas à l'agent (hors périmètre de ce lot, cf. consigne d'exécution)

**Gate — exécuté le 2026-09-10, sorties réelles (worktree `LevelUp-wt-fonds-webp`, branche
`feat/fonds-carte-webp`, `LEVELUP_REPO_ROOT` pointé sur le worktree — `db_profiles.json`
n'existe pas dans ce worktree, non versionné) :**
```bash
cd apps/go-api && go build ./...                       # BUILD_EXIT=0
cd apps/go-api && go vet ./cmd/mapfond-webp/            # VET_EXIT=0
cd apps/go-api && go test ./cmd/mapfond-webp/           # ok  0.223s
cd apps/go-api && go run ./cmd/mapfond-webp -verifier -echantillon=5   # 5/5 identique=true, gain cumulé 38.4 %
cd apps/go-api && golangci-lint run --new-from-merge-base=feat/v75 ./cmd/mapfond-webp/   # 0 issues
git status --short data/                                # vide, avant ET après
```
Gate passé : test vert, aller-retour identique sur les 5, gain cumulé 38,4 % >= 20 %.

---

## Étape 1 — ETag et 304, centralisés

Indépendante du WebP : livrable seule, et livrée **même si D10 abandonne la suite**.

**Périmètre fermé (6 fichiers) :**

- [ ] Créer `apps/go-api/internal/api/handlers/cache_http.go` — `servirBlobAvecETag(w, r, blob,
      contentType, cacheControl)` : calcule l'ETag (D7), pose `ETag`, `Content-Type`,
      `Cache-Control` ; si `If-None-Match` contient l'ETag **ou** `*`, répond `304` **sans corps
      ni `Content-Length`** ; sinon pose `Content-Length` et écrit le blob
- [ ] L'analyse d'`If-None-Match` gère la **liste séparée par virgules** et le préfixe faible
      `W/`. Un en-tête absent ou illisible se comporte comme une absence (200), jamais comme une
      erreur
- [ ] `handlers/replay.go:148-180` — remplacer les trois `Header().Set` et le `Write` par l'appel
      au helper
- [ ] `handlers/tactical.go:281-316` — idem
- [ ] `handlers/assets.go:217` — idem (D8 : 3e copie, donc migration obligatoire ; l'ETag y existe
      déjà mais sans 304, ce qui le rend inerte)
- [ ] Créer `apps/go-api/internal/api/handlers/cache_http_test.go` — garde-rail **grep** (règle
      n°6) : aucun `Header().Set("ETag"` ailleurs que dans `cache_http.go`, allowlist vide
- [ ] Tests `httptest` sur les trois routes : 1re requête `200` + ETag non vide ; 2e requête avec
      `If-None-Match` = cet ETag → `304`, **corps vide**, et pas de `Content-Length` non nul

**Gate :**
```bash
cd apps/go-api && go test ./internal/api/handlers/
cd apps/go-api && go build ./...
```
- [x] Créé `apps/go-api/internal/api/handlers/cache_http.go` — `servirBlobAvecETag(w, r, blob,
      contentType, cacheControl)` : calcule l'ETag (D7), pose `ETag` ; si `If-None-Match`
      contient l'ETag **ou** `*` (`ifNoneMatchCorrespond`), répond `304` **sans corps ni
      `Content-Length`** ; sinon pose `Content-Type`, `Cache-Control` (si non vide),
      `Content-Length`, corps
- [x] `ifNoneMatchCorrespond` (cache_http.go) : liste séparée par virgules (`strings.Split(",")`),
      préfixe faible `W/` toléré (`strings.TrimPrefix`), wildcard `*`. En-tête absent/illisible →
      `false` → 200, jamais une erreur. Couvert par
      `TestServirBlobAvecETag_{ListeAvecPrefixeFaible,Wildcard,EnTeteIllisible}_*`
- [x] `handlers/replay.go:148-180` (relu avant édition) — les trois `Header().Set` +
      `Content-Length` + `Write` remplacés par un seul appel à `servirBlobAvecETag` ; imports
      `strconv` et `log/slog` retirés (devenus inutilisés dans ce fichier)
- [x] `handlers/tactical.go:281-316` (relu avant édition) — idem ; import `strconv` retiré
      (`log/slog` reste utilisé ailleurs dans le fichier)
- [x] `handlers/assets.go:217` (relu avant édition) — idem. **Amendement S5** : `p.ETag` (hash
      amont précalculé par `internal/assets`, jamais lu côté `If-None-Match` donc inerte) n'est
      plus posé tel quel ; le helper recalcule son propre ETag fort depuis `p.Bytes` (D7, source
      unique = le contenu servi)
- [x] Créé `apps/go-api/internal/api/handlers/cache_http_test.go` — garde-rail grep
      `TestNoRawETagHandlingOutsideCacheHTTP` : aucun `Header().Set("ETag"` ni
      `Header.Get("If-None-Match")` littéral ailleurs que dans `cache_http.go`, allowlist VIDE et
      datée 2026-09-09. Discriminance prouvée par `TestETagGuardIsDiscriminant` (jumeau de
      `TestRetryAfterGuardIsDiscriminant`) **et** par une mutation réelle et temporaire de
      `replay.go` (ré-ajout d'un `Header().Set("ETag", ...)` littéral), qui a fait échouer le
      garde-rail comme attendu, revertée aussitôt — sortie observée consignée au thought_log
- [x] **Amendement S5** : `writeJSONCached` (helpers.go) migré vers `servirBlobAvecETag` — c'était
      la copie n°1 du motif (avant même l'étape 1), pas seulement les 3 sites du plan initial ;
      son format d'ETag change de `"%x"` (16 hex) vers `"sha256-<12 hex>"` (D7). Vérifié qu'aucun
      test n'épingle l'ancien format : tous les tests ETag du paquet lisent
      `w.Header().Get("ETag")` dynamiquement (grep sur `internal/api/handlers/*_test.go`), aucune
      comparaison à un littéral
- [x] Tests `httptest` sur les trois routes blob (`cache_http_routes_test.go`, réutilise les
      mocks existants `mockReplayService`/`routeurFond`/`stubResolver`) + la route JSON cachée
      (`writeJSONCached`, seul appelant JSON du helper — cf. Découvertes ci-dessous) : 1re requête
      `200` + ETag non vide ; 2e avec `If-None-Match` = cet ETag → `304`, corps vide, pas de
      `Content-Length` non nul ; liste `W/"x", "<etag>"` → `304` ; en-tête illisible → `200`

**Gate — exécuté le 2026-09-09, sorties réelles :**
```bash
cd apps/go-api && go test ./internal/api/handlers/     # ok  levelup/go-api/internal/api/handlers  9.575s
cd apps/go-api && go build ./...                       # BUILD_EXIT=0
cd apps/go-api && go vet ./internal/api/...            # VET_EXIT=0 (ajouté à la demande du superviseur)
cd apps/go-api && golangci-lint run --new-from-merge-base=origin/main ./internal/api/handlers/  # 0 issues.
```
Gate passé, les 4 commandes vertes.

---

## Étape 2 — Lecture du fond indépendante du format

Aucun fichier n'est converti à cette étape : le comportement reste **strictement identique**
(tous les sidecars disent encore `.png`). C'est le socle qui rendra l'étape 4 sans risque.

**Périmètre fermé (5 fichiers) :**

- [ ] `internal/domain/title/registry.go` — ajouter
      `MapBackgroundImageFilePath(titleSlug, nomFichier string) string`. Ne pas supprimer
      `MapBackgroundPath` (encore utilisé en écriture par la cuisson) ; ne **jamais** faire de
      `filepath.Join` à la main ailleurs
- [ ] `internal/service/replay_map_background.go:110-123` — `readBackgroundImage` construit le
      chemin depuis `meta.Image` (D3) au lieu de la clé + `.png`
- [ ] Même fichier — garde de sûreté sur `meta.Image`, sur le modèle de `cleDeFondSure` (l. 217) :
      `filepath.Base` imposé, et **liste blanche d'extensions** `{.png, .webp}`. Tout autre cas →
      `slog.ErrorContext` puis `port.ErrMapBackgroundNotAvailable` (jamais de panique, jamais
      d'erreur avalée)
- [ ] Même fichier — `MapBackgroundImage` et `MapBackgroundImageForMap` retournent désormais
      `([]byte, string, error)`, la chaîne étant le type MIME déduit de l'extension par une table
      `{.png: image/png, .webp: image/webp}`. Répercuter sur l'interface de `internal/port/` et
      sur les deux handlers (qui passent la valeur à `servirBlobAvecETag`)
- [ ] `internal/service/replay_map_background_traversee_test.go` — ajouter les cas
      `Image: "../../ailleurs.png"`, `Image: "/absolu.png"` et `Image: "carte.svg"` : les trois
      doivent rendre `ErrMapBackgroundNotAvailable`
- [ ] Adapter les assertions `image/png` en dur : `handlers/replay_test.go:239-240`,
      `handlers/tactical_background_test.go:57-58` — elles doivent lire le type servi, pas le
      supposer

**Gate :**
```bash
cd apps/go-api && go test ./internal/service/ ./internal/api/handlers/ ./internal/domain/title/
cd apps/go-api && go test ./...
```
Gate passé si tout est vert **et** qu'aucun fichier de `data/` n'a été modifié
(`git status --porcelain data/` vide).

---

## Étape 3 — Les outils hors ligne savent lire le WebP

Sans cette étape, l'étape 4 casse silencieusement la planche de contact et le cadrage.

**Périmètre fermé (5 fichiers) :**

- [ ] `apps/go-api/go.mod` — ajouter `golang.org/x/image` (D6)
- [ ] `cmd/mapfond-planche/main.go:152` — `png.Decode` → `image.Decode`, plus import blanc
      `_ "golang.org/x/image/webp"` et `_ "image/png"`
- [ ] `cmd/mapfond-cadrage/main.go:122` — idem
- [ ] `internal/mapdecoupe/masque.go:69` — idem
- [ ] Passer en revue `cmd/mapfond-inventaire` et `cmd/migrate-static-maps` : s'ils ouvrent un
      fond de carte, même traitement ; sinon, statuer `[~]` avec la référence
- [ ] **Hors périmètre, à ne pas toucher** : `cmd/vs-measure/*` et `cmd/vehicle-sprite/compose.go`
      décodent des sprites de véhicules, pas des fonds de carte. Les laisser en `png.Decode`
- [ ] Un test dans `internal/mapdecoupe/` qui décode un WebP sans perte produit par l'outil de
      l'étape 0 et vérifie les dimensions

**Gate :**
```bash
cd apps/go-api && go build ./... && go vet ./...
cd apps/go-api && go test ./internal/mapdecoupe/
```

---

## Étape 4 — Conversion des 109 fonds

Première étape qui touche la donnée. Elle est atomique : un seul commit, réversible par `git
revert`.

**Périmètre fermé :**

- [ ] `.gitattributes` — ajouter `*.webp binary` à côté de `*.png binary`
- [ ] `go run ./cmd/mapfond-webp -convertir` sur les 109 fonds : écrit `<clé>.webp`, **met à jour
      le champ `Image` du sidecar** (`<clé>.webp`), puis supprime `<clé>.png`. Ne pas toucher
      `SchemaVersion` : le champ `Image` porte déjà un nom de fichier, le contrat ne change pas
- [ ] L'outil refuse d'écrire si l'aller-retour n'est pas identique au bit près sur **le fichier
      en cours** — la vérification n'est pas qu'un mode séparé
- [ ] Vérifier le décompte : 109 `.webp`, 109 `.json`, **0 `.png`** dans le répertoire
- [ ] Consigner le gain réel total dans § Découvertes

**Gate :**
```bash
ls data/titles/halo_infinite/reference/map_backgrounds/*.png 2>/dev/null | wc -l   # attendu 0
ls data/titles/halo_infinite/reference/map_backgrounds/*.webp | wc -l              # attendu 109
grep -L '"image": *"[^"]*\.webp"' data/titles/halo_infinite/reference/map_backgrounds/*.json
cd apps/go-api && go test ./...
make go-api-test
```
Le `grep -L` doit ne rien lister. Puis, serveur lancé :
```bash
curl -sI localhost:8000/api/players/<GT>/matches/<ID>/replay/background.png | grep -i -E 'content-type|etag'
```
attendu : `image/webp` et un ETag non vide.

---

## Étape 5 — Recette de l'export vidéo (le point de vigilance du commanditaire)

Les gates jsdom sont **aveugles** aux capacités réelles du navigateur : c'est la leçon coûteuse du
chantier export de fin août. Rien de ce qui précède ne prouve que la vidéo livrée a un fond.

**Périmètre fermé (1 fichier + 1 passage navigateur) :**

- [ ] `scripts/recette_export_rejeu.js` — ajouter un **12e verdict `fondDeCarte`** : sur une frame
      exportée, la proportion de pixels non transparents hors surcouche est supérieure à un seuil,
      ce qui échoue si le fond n'a pas été décodé. Verdict rendu **avec les 11 autres**, jamais
      isolément
- [ ] **Vider le cache avant la passe** : `staleTime: Infinity` côté TanStack **et**
      `max-age=3600` côté HTTP peuvent servir l'ancien blob PNG et masquer une régression.
      Rechargement forcé, cache désactivé dans l'inspecteur
- [ ] Exécuter la recette selon la procédure du dépôt : copier temporairement le script dans
      `apps/web/public/`, l'exécuter dans le navigateur, **RETIRER la copie**
- [ ] Rendre les **12 verdicts d'un seul coup**. Ne jamais annoncer une propriété à la fois
- [ ] Contrôle visuel à l'échelle d'export : ouvrir la vidéo produite et vérifier que le fond est
      net, sans halo ni frange sur l'alpha (D1 le garantit en théorie ; on le regarde quand même)
- [ ] Contrôler aussi l'onglet **Tactique** (`features/tactical/queries.ts:164`), second
      consommateur du fond, qui pose l'objectURL en CSS et non au canvas

**Gate :** les 12 verdicts au vert en une passe, plus le contrôle visuel de la vidéo. Si le
prérequis `donneesDuMatch` échoue (mention « SANS ÉQUIPE », ou horloge à la durée du film au lieu
de la durée jouée), **c'est la vue du match qui est morte, pas l'export** : ne pas chercher dans
ce chantier.

---

## Clôture

- [ ] Entrée dans `.ai/thought_log.md` : date `[2026-09-09]`, titre, statut, décision technique
      principale (D1 sans perte + D3 format porté par la donnée), gain mesuré, prochaine étape
- [ ] Skill `delivery-checklist` avant d'annoncer la livraison
- [ ] **Une seule** revue adversariale (`adversarial-review`) sur le diff complet, en fin de
      chantier — le lot touche la persistance de données de référence et une frontière HTTP
- [ ] Ne pas pousser sur `main` (déploiement prod automatique) sans accord explicite

## Découvertes (à remplir en cours d'exécution, sans traiter)

| Date | Constat | Où | Suite proposée |
|---|---|---|---|
| 2026-09-09 | `handlers/assets.go:217` posait un ETag jamais honoré faute de gestion d'`If-None-Match` : l'en-tête était inerte | `handlers/assets.go:217` | Traité **dans** ce plan par D8 (règle de la 3e copie), pas reporté |
| | | | |
| 2026-09-09 | `writeJSONCached` (helpers.go) posait déjà SA PROPRE logique ETag/304 (format `"%x"` 16 hex, sans liste ni `W/`) — c'était donc la copie n°1 du motif avant même l'étape 1, pas 3 copies mais 4 en comptant assets.go et les 2 nouveaux sites | `handlers/helpers.go:95-121` (avant migration) | Traité **dans** ce plan, § amendement S5 : `writeJSONCached` migré vers `servirBlobAvecETag` dans le même commit |
| 2026-09-09 | Les endpoints Huma (`capabilities.go`, `feature_matrix.go`, `field_mappings.go`, `home.go`) posent CHACUN leur propre calcul `sha256.Sum256(body)` + champ de sortie `ETag string \`header:"ETag"\`` — ils NE PEUVENT PAS appeler `servirBlobAvecETag` (qui écrit directement sur `http.ResponseWriter`), Huma sérialisant la réponse depuis la struct de sortie après le retour du handler. C'est 3 copies indépendantes du calcul SHA-256 (pas du branchement ETag/If-None-Match, qui reste propre à Huma) | `handlers/capabilities.go:108`, `handlers/feature_matrix.go:118`, `handlers/field_mappings.go:355` | Hors périmètre de l'étape 1 (contrat HTTP différent, pas un `Header().Set`/`Write` brut) — non traité, à évaluer dans un chantier séparé si la règle des 2 copies doit s'appliquer au calcul du hash lui-même |
| 2026-09-10 | `title.FindRepoRoot()` échoue dans le worktree dédié `LevelUp-wt-fonds-webp` : il cherche `db_profiles.json` en remontant depuis le cwd, or ce fichier n'est pas versionné et n'existe donc dans AUCUN worktree fraîchement créé (seulement dans le poste de travail principal, hors git) | `apps/go-api/internal/domain/title/repo_root.go:18` (comportement, pas un bug — le fichier documente lui-même `LEVELUP_REPO_ROOT` comme repli) | Non traité ici : contournement local par `LEVELUP_REPO_ROOT=<racine du worktree>` pour le gate de cette étape, conforme à l'usage documenté de la variable. Rien à corriger dans le code |
| 2026-09-10 | La première version du test synthétique (`main_test.go`) construisait l'image avec `*image.RGBA` et des octets `{R:10,G:60,B:220,A:120}` sur la zone semi-transparente — invalide au regard du modèle prémultiplié (`B=220 > A=120`), ce qui faisait échouer `TestAllerRetourWebP_ImageSynthetique_Identique` (round-trip non identique) alors que l'encodeur n'a aucun défaut : c'était un artefact du test, pas de l'outil | `apps/go-api/cmd/mapfond-webp/main_test.go` (avant correction, cf. historique de session) | Traité **dans ce lot** : image reconstruite en `*image.NRGBA` (non prémultiplié, le type concret que rend réellement `png.Decode` sur un fond RGBA+alpha) — pas un report |
| 2026-09-10 | Mesure `-verifier -echantillon=5` (5 plus gros fonds, aller-retour identique sur les 5) : voir tableau ci-dessous | `apps/go-api/cmd/mapfond-webp/` | Consigné pour la décision D10 (utilisateur) |

### Mesures de l'Étape 0 — `-verifier -echantillon=5`, 2026-09-10

| Fichier | Octets PNG | Octets WebP | Gain % | Durée encodage | Durée décodage | Identique |
|---|---:|---:|---:|---:|---:|---|
| `btb_fragmentation.png` | 2 015 726 | 1 053 610 | 47,7 % | 2,363 s | 67,9 ms | oui |
| `37bc3df6-93e8-4d74-b16e-5ceaa30ebc23.png` | 1 559 572 | 997 276 | 36,1 % | 2,155 s | 55,5 ms | oui |
| `28a3ac28-f69d-4fa9-9ebf-a0449c89c8da.png` | 1 559 572 | 997 276 | 36,1 % | 1,978 s | 59,4 ms | oui |
| `1ede38fa-4d30-4dfa-a8b7-5d08bf4e46e3.png` | 1 424 088 | 940 794 | 33,9 % | 1,702 s | 48,5 ms | oui |
| `305b1bdd-9a7b-4975-bacf-8bd63c8c13d2.png` | 1 411 701 | 919 506 | 34,9 % | 1,596 s | 60,0 ms | oui |
| **CUMUL (5)** | **7 970 659** | **4 908 462** | **38,4 %** | — | — | **5/5 identique** |

Seuil du gate (>= 20 % de gain cumulé, aller-retour identique au bit près) : **atteint** sur cet
échantillon. `data/` n'a subi aucune modification (`git status --short data/` vide avant et
après ; 218 fichiers avant/après dans `map_backgrounds/`). La décision de poursuivre les étapes
2 à 5 (D10) reste à l'utilisateur.
