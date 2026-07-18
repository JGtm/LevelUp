# PLAN — Explorer briefing V6 : décile MVP/LVP (team_mmr), triptyques (extrêmes lisibles), tooltips factuels, largeurs socle inégales, Classement pertinent

Statut : PLANIFIE (aucune ligne de code écrite — plan rédigé par l'architecte Opus).
Date : 2026-07-18.
Chantier précédent : `.ai/V7/PLAN_EXPLORER_BRIEFING_V5_2026-07.md` (LIVRÉ sur la branche courante,
commits `4015d6650`, `a0a335a01` ; triptyques FDA/Perf, cascade ≤ 8, Classement en rangée,
MVP/LVP par décile, alignement colonnes, accents, valeurs centrées). Ce chantier = **2e tour de
polish POST-revue visuelle V5**, décisions utilisateur des 2026-07-17/18 (§3, DP-1..DP-5)
TOUTES confirmées — à VÉRIFIER sur pièces et structurer, jamais re-débattre.

Branche cible d'implémentation : **`feat/explorer-briefing-compact`** (déjà la branche courante ;
V6 s'empile sur V5 déjà livré sur cette branche ; NE PAS en changer ; NE committer que ce que ce
plan autorise, par phase ; le superviseur gère la clôture git).

> Contrat d'exécution : ce plan s'exécute sous le skill **`plan-execution`** (ordre strict, une
> étape close avant la suivante, aucun report d'action exécutable maintenant, statut sur chaque
> item, zéro fix hors périmètre). En cas de divergence, le présent plan fait foi ; à défaut, le
> skill est le défaut. Avant de finaliser toute modification du plan : skill **`plan-review`**.
> Avant chaque commit : skill **`delivery-checklist`**. Code Go : **`arch-rules`** +
> **`go-features`** + **`db-schema`** ; code React/TS : **`frontend-patterns`** ; toute couleur :
> **`color-tokens`**. Rappels transverses : tokens sémantiques UNIQUEMENT (aucun hex ni classe
> Tailwind couleur `text-red-*`/`bg-*` dans `features/`/`components/` — `text-foreground` /
> `text-muted-foreground` sont des tokens de thème AUTORISÉS) ; seuils fichier ≤ 500 L / fonction
> ≤ 80 L / ≤ 5 params / complexité ≤ 12 ; FR sans anglicismes (« série » pas « streak », « Taux
> de victoire » pas « WR ») ; parité i18n FR/EN par typage ; branchement Go par **capability**
> jamais par `slug == …` ; **pas de commandes `go` concurrentes** (corruption du cache Windows —
> séquentiel, tuer les `link.exe` orphelins) ; vitest hors sandbox
> (`dangerouslyDisableSandbox=true`) ; purger `apps/web/node_modules/.tmp` avant
> `make check-types`. **La vérification NAVIGATEUR est reprise par l'utilisateur** (§ « À vérifier
> visuellement ») — jamais planifiée comme tâche agent. **Pas d'emojis dans les fichiers
> versionnés** (marqueurs de statut = texte, ex. « CLOSE »/« PASSÉ »).

---

## 1. Objectif et critères de succès (mesurables)

**Objectif.** Cinquième et dernière passe de finition du bandeau de briefing de l'Explorer (mode
Matchs), selon 5 décisions produit tranchées les 2026-07-17/18 (§3) :

- **DP-1** — dans le surlignage MVP/LVP par décile du tableau : **RETIRER** la colonne
  `score_label` et **AJOUTER** `team_mmr` (non inversé, haut = meilleur). Set final :
  `perf_score`, `kda`, `kills`, `deaths` (inversé), `team_mmr`.
- **DP-2** — dans `MinMaxTriptych` (FDA & Perf), rendre les valeurs extrêmes (min/max) **plus
  lisibles** : token `text-foreground` (« blanc » theme-aware) + taille `text-xs` (au lieu de
  `text-2xs`/`text-muted-foreground`). La moyenne reste mise en avant/colorée (inchangée).
- **DP-3** — réécrire **TOUS** les tooltips du bandeau (11 clés `tip_*`) en registre **factuel/
  descriptif**, en SUPPRIMANT tout vouvoiement (« vous / votre / vos », impératif « survolez »),
  FR + EN, parité conservée.
- **DP-4** — **largeurs inégales** de deux tuiles socle : « Séries marquantes » ~10 % plus large,
  « Pic MMR » ~10 % plus étroite en compensation, sans casser le responsive ni les autres tuiles,
  et en gérant l'ABSENCE de l'une ou l'autre (tuiles conditionnelles → aucun trou/déséquilibre).
- **DP-5** — le bloc **Classement** ne liste PLUS toutes les chaînes (LUSR/CSR) : filtrage par
  pertinence MIROIR du pattern dimensions (seuil de matchs nommé + plafond top N par nombre de
  matchs), avec **fallback** garantissant que la chaîne principale (type majoritaire) reste
  représentée. Changement BACKEND (Go).

**Critères de succès (tous vérifiables ; la vérification NAVIGATEUR est reprise par l'utilisateur
— § « À vérifier visuellement »).**

1. **Décile MVP/LVP recomposé (DP-1).** `ExplorerHighlightKey` = `'kills' | 'deaths' | 'kda' |
   'perf_score' | 'team_mmr'` (plus de `'score_label'`). `EXPLORER_INVERTED.team_mmr = false` ;
   `explorerHlExtract.team_mmr = (r) => r.team_mmr ?? null`. La fonction `ownTeamScore` et son
   bloc de test, devenus code mort (unique lecteur = l'extracteur `score_label` retiré), sont
   SUPPRIMÉS. La colonne `score_label` reste affichée dans le tableau (seul son surlignage part).
   Vérifié par `ExplorerMatchesTable.highlight.test.ts` (bande décile sur `team_mmr`, `score_label`
   non surlignée) — aucune modification de `ExplorerMatchesTable.tsx` (le `<td>` est déjà gaté par
   `isExplorerHighlightKey`).
2. **Extrêmes du triptyque lisibles (DP-2).** Les deux `<span>` min/max de `MinMaxTriptych`
   passent de `text-2xs … text-muted-foreground` à `text-xs … text-foreground` (contraste plein,
   theme-aware — blanc en sombre, quasi-noir en clair — via le token `text-foreground`, JAMAIS
   `#fff`). Le `<span>` central (moyenne) est INCHANGÉ (couleur `midColor`, hérite `text-xl`).
   Vérifié par test Strip (classes du triptyque).
3. **Tooltips factuels sans vouvoiement (DP-3).** Les 11 clés `explorer.briefing.tip_*` (liste
   exhaustive §2.3) sont réécrites FR + EN : aucune occurrence de « vous »/« votre »/« vos » ni
   d'impératif de politesse (« survolez ») en FR ; registre descriptif ; le triptyque décrit
   « plus bas · moyen · plus haut » (jamais « votre meilleur/pire »). Parité EN (registre
   impersonnel, sans « your/you » possessif). Manifests régénérés. Garde-rail terminologie vert.
4. **Largeurs socle inégales (DP-4).** « Séries marquantes » rendue ~10 % plus large, « Pic MMR »
   ~10 % plus étroite ; les autres tuiles inchangées ; la rangée reste pleine largeur SANS trou
   quel que soit le nombre de tuiles (4 à 8) et quelle que soit l'absence des conditionnelles.
   Vérifié par test Strip (structure socle) + revue visuelle.
5. **Classement pertinent (DP-5).** Le module Classement n'émet que les chaînes avec
   `Matches ≥ MinRankedChainMatches` (constante nommée, alignée `MinDimensionGroupMatches`),
   plafonnées aux `RankedChainMaxCount` chaînes les plus jouées ; si AUCUNE chaîne n'atteint le
   seuil mais qu'au moins une progression existe, **fallback** : la chaîne principale (première en
   ordre canonique = plus grande chaîne du type majoritaire) est conservée. Ordre d'affichage
   déterministe conservé. Vérifié par tests service (seuil, plafond, fallback, ordre) ; l'algo pur
   `analysis.ComputeRankProgressionByChain` reste INCHANGÉ (ses tests restent verts).
6. **Gates verts** (par phase, §5) : Phase 1 (Go) — `cd apps/go-api && go test ./...` = 0
   (SÉQUENTIEL) + `make go-api-lint` = 0 ; Phases 2-3 (front/i18n) — `make check-types` = 0
   (`.tmp` purgé) + `make test-web` (dangerouslyDisableSandbox) vert + `cd apps/web && npm run
   lint` = 0 erreur + `build_i18n_manifests.mjs` (Phase 3) + greps de clôture. **`generate-types`
   NON requis** (aucun changement OpenAPI/DTO — §4). **`-tags=integration` NON requis** (lecture/
   agrégation mémoire pure — §4).
7. **Changelog** : entrée `[Unreleased]` v7.0 mise à jour dans `docs/CHANGELOG.md` ET
   `docs/FR/CHANGELOG.md` (parité EN/FR même commit).

---

## 2. Constat sur pièces — état actuel post-V5 (fichier:ligne réels, vérifiés le 2026-07-18)

> Doctrine du projet : RE-VÉRIFIER chaque ancrage sur pièces AVANT de coder ET avant de cocher (le
> code a pu bouger). Numéros ci-dessous = état vérifié le 2026-07-18 (post-V5).

### 2.1 DP-1 — surlignage décile (frontend, aucun backend)

- **`apps/web/src/features/explorer/ExplorerMatchesTable.highlight.ts`** (147 L) :
  - `ExplorerHighlightKey` (`:20`) = `'kills' | 'deaths' | 'kda' | 'perf_score' | 'score_label'`.
  - `EXPLORER_INVERTED` (`:26-32`) : `kills:false, deaths:true, kda:false, perf_score:false,
    score_label:false`. `isExplorerHighlightKey` (`:81-83`) = `key in EXPLORER_INVERTED` (garde de
    type — sa véracité dépend de la présence de la clé dans ce record).
  - `ownTeamScore(label)` (`:41-45`) : 1er entier du libellé « A - B ». **Unique consommateur =
    l'extracteur `score_label`** (`:55`) + son bloc de test (`highlight.test.ts:128-135`) —
    confirmé par grep. Devient CODE MORT après DP-1.
  - `explorerHlExtract` (`:48-56`) : `kills`/`deaths`/`kda` (valeur brute), `perf_score` (gardé sur
    `perf_tier` présent, `:54`), `score_label: (r) => ownTeamScore(r.score_label)` (`:55`).
  - `HL_KEYS = Object.keys(explorerHlExtract)` (`:58`) — dérivé automatiquement (aucun ajustement
    manuel après édition des records).
- **`apps/web/src/features/explorer/ExplorerMatchesTable.tsx`** (854 L, god-file préexistant, hors
  périmètre de split — V5 Découverte-13) :
  - Colonne `team_mmr` : `accessorFn: (r) => r.team_mmr ?? undefined` (`:616`), `id: 'team_mmr'`
    (`:617`), cellule `fmtMmr(getValue())` (`:620-623`). **Colonne CONDITIONNELLE** : rendue
    uniquement si `providesTeamMmr` (`useProvidesTeamMmr()`, `:258` ; H5 → masquée, `:611-613`).
  - Câblage surlignage `<td>` (`:803-804`) :
    `const hlStyle = isExplorerHighlightKey(colId) ? columnHighlightStyle(colId,
    explorerHlExtract[colId](row.original), highlightDeciles) : {}`. **GARDÉ par
    `isExplorerHighlightKey`** → ajouter `team_mmr` aux records auto-câble le surlignage ; retirer
    `score_label` le retire ; si la colonne `team_mmr` est masquée (H5), pas de `<td>` → pas de
    surlignage. **AUCUNE modification de `ExplorerMatchesTable.tsx` requise pour DP-1.**
  - `team_mmr` figure DÉJÀ dans `RIGHT_ALIGNED_COLUMNS` (V5, alignement à droite) — inchangé.
- **`apps/web/src/lib/api/types.ts`** : `ExplorerMatchRow.team_mmr: number | null` (`:611`) — champ
  existant, aucun changement DTO/OpenAPI.
- **Tests** : `ExplorerMatchesTable.highlight.test.ts` (bande décile, `deaths` inversé, garde
  `< MIN_DECILE_SAMPLE`, `p10===p90`, null, `perf` sans tier exclue ; bloc `ownTeamScore` `:128`) ;
  `ExplorerMatchesTable.test.tsx` (bloc MVP/LVP + alignement). Les deux référencent `score_label`
  et `ownTeamScore` → à mettre à jour (DP-1).

### 2.2 DP-2 — extrêmes du triptyque (frontend)

- **`apps/web/src/features/explorer/ExplorerBriefingTiles.tsx`** — `MinMaxTriptych` (`:36-62`) :
  - `<span>` MIN (`:51-53`) : `className="text-2xs font-normal tabular-nums text-muted-foreground"`.
  - `<span>` MOYENNE (`:54-56`) : `style={{ color: midColor }}`, sans classe de taille (hérite
    `text-xl` du conteneur `BriefingTile`). **INCHANGÉ** (DP-2 ne touche que les extrêmes).
  - `<span>` MAX (`:57-59`) : même classe que MIN.
  - DP-2 = remplacer, sur MIN et MAX uniquement, `text-2xs` → `text-xs` et `text-muted-foreground`
    → `text-foreground`. `font-normal tabular-nums` conservés (hiérarchie : moyenne `text-xl`
    colorée >> extrêmes `text-xs` blancs). `text-foreground` est un token de thème (déjà utilisé
    par `BriefingTile.tsx:33`) — AUTORISÉ, JAMAIS `#fff` (color-tokens).
- **Tests** : `ExplorerBriefingStrip.test.tsx` (describe « triptyques … ») — vérifier s'il asserte
  les classes `text-2xs`/`text-muted-foreground` du triptyque ; si oui, mettre à jour.

### 2.3 DP-3 — tooltips (i18n)

- **`apps/web/src/lib/i18n/manifests/explorer.toml`** — 11 clés `tip_*` (source ; le générateur
  produit `apps/web/src/lib/i18n/generated/explorer.ts`). Liste EXHAUSTIVE (grep sur pièces) et
  présence de vouvoiement FR à corriger :

  | Clé (`explorer.briefing.…`) | Ligne toml | Vouvoiement FR actuel |
  |---|---|---|
  | `tip_dimensions` | `:905-907` | « Où **vous** performez… » |
  | `tip_context` | `:909-911` | « **Vos** résultats selon que **vous** jouiez… » |
  | `tip_win_rate` | `:913-915` | « …**survolez**-le pour le détail… » (impératif) |
  | `tip_fda` | `:917-919` | « …**votre** FDA le plus bas, **votre** moyenne… et **votre** meilleur. » |
  | `tip_perf` | `:921-923` | « …**votre** perf la plus basse, **votre** moyenne… et **votre** plus haute. » |
  | `tip_duration` | `:925-927` | (aucun — déjà factuel) |
  | `tip_peak_rank` | `:929-931` | « …pas **votre** palier final. » |
  | `tip_peak_mmr` | `:933-935` | (aucun — déjà factuel) |
  | `tip_ranked` | `:937-939` | « **Votre** palier de classement à la fin… » |
  | `tip_streaks` | `:941-943` | « **Vos** séries extrêmes… » |
  | `tip_highlights` | `:945-947` | « **Vos** matchs marquants… » |

  Consommateurs (grep) : Strip (`tip_win_rate`, `tip_fda`, `tip_perf`, `tip_duration`,
  `tip_streaks`, `tip_peak_rank`, `tip_peak_mmr`), Modules (`tip_dimensions`, `tip_context`,
  `tip_highlights`), RankedBlock (`tip_ranked`). Textes FR/EN par défaut proposés en §3 (DEC-TIP).
- **Garde-rail** : `explorerBriefingTerminology.guard.test.ts` (n'interdit que « Par playlist »/
  « Pronostic »/« Prognosis ») — aucune de ces tournures introduite. Vérifier qu'aucun test
  n'asserte un substrat de tooltip contenant « votre »/« vous » (grep tests avant édition).

### 2.4 DP-4 — largeurs socle (frontend)

- **`apps/web/src/features/explorer/ExplorerBriefingStrip.tsx`** (191 L) :
  - Grille socle (`:100`) : `<div className="grid gap-2 grid-cols-2
    sm:[grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">` — **tuiles ÉGALES** (auto-fit /
    tracks `1fr` uniformes ; aucune largeur par tuile possible en l'état).
  - Tuiles de base (5 hors low_sample) rendues comme enfants DIRECTS de la grille : Matchs
    (`:102-108`, `<BriefingTile>`), WinRate (`:110`), FDA (`:113-141`, `<BriefingTile>` inline),
    Perf (`:144-172`, `<BriefingTile>` inline), Durée (`:175`).
  - Tuiles CONDITIONNELLES collectées dans `conditionalTiles: ReactNode[]` (`:85-96`) par priorité :
    **StreaksTile** (« Séries marquantes », `:88`), PeakRankTile (`:91`), **PeakMmrTile** (« Pic
    MMR », `:94`) ; rendues via `{conditionalTiles}` (`:178`), toutes présentes (V5 : cap 8, les 3
    tiennent). **Streaks et Pic MMR sont conditionnelles** (peuvent être absentes).
- **`apps/web/src/features/explorer/BriefingTile.tsx`** (40 L) : `TileProps = { label, value, info?,
  sub?, accent? }` — **PAS de prop `className`**. Rend `<KpiCard accent={accent}
  className="h-full">` (`:27`).
- **`apps/web/src/components/cards/KpiCard.tsx`** (47 L) : prop `className?` (`:28`) fusionnée par
  **concaténation** de chaîne sur la racine `<div>` (`:37` : `overflow-hidden rounded-lg border
  border-border bg-card ${className}`). **Aucun helper `cn`/`clsx`/`twMerge` dans le repo** (grep =
  0) → un override de classe Tailwind (`basis-[…]`) par concaténation serait fragile (deux
  `basis-*` co-présents, ordre CSS imprévisible). Conséquence : DP-4 ne doit PAS reposer sur un
  override de className via BriefingTile ; l'approche retenue (DEC-WIDTHS) place TOUTE la logique de
  largeur dans le Strip (wrappers flex), sans toucher aux composants partagés.
- **Tests** : `ExplorerBriefingStrip.test.tsx` (structure/décompte du socle : « 8 cellules »,
  « 8 barres 3px ») — à re-vérifier après passage grid→flex (le décompte des `KpiCard`/barres reste
  8 ; un décompte d'enfants DIRECTS de la grille reste 8 si l'on wrappe chaque tuile).

### 2.5 DP-5 — Classement pertinent (backend Go)

- **`apps/go-api/internal/service/match_history_service_briefing_ranked.go`** (83 L) :
  - `buildBriefingRanked(ctx, scope)` (`:27-57`) : `samples := rankChainSamples(scope)` (`:28`) →
    `progs := analysis.ComputeRankProgressionByChain(samples)` (`:32`) → mappe **TOUTES** les progs
    vers `Kinds` (`:36-55`). **Aucun filtrage de pertinence aujourd'hui** — toutes les chaînes
    émises. C'est ici (SERVICE) que DP-5 s'applique.
  - `rankChainSamples` (`:63-82`) : projection raw rows → samples. INCHANGÉ.
- **`apps/go-api/internal/service/match_history_service_briefing.go`** — pattern de pertinence des
  DIMENSIONS à MIRRORER :
  - Bloc const (`:32-43`) : `MinDimensionGroupMatches = 10` (`:40`, « seuil sous lequel un groupe
    n'apparaît pas en top/flop »), `dimensionTopFlopCount = 3` (`:42`).
  - `buildDimension` (`:318-372`) : `qualified` = groupes avec `d.Session.Played >=
    MinDimensionGroupMatches` (`:333-338`), puis `selectTopFlop(qualified, dimensionTopFlopCount)`
    (`:351`). C'est le pattern SEUIL + PLAFOND à répliquer pour les chaînes.
  - `selectTopFlop[T](items, k)` (`:376-384`) : générique, retourne les k premiers + k derniers
    d'une liste triée ; `len ≤ 2k` → tel quel. (Le plafond ranked n'a pas de « flop » — un helper
    dédié « top N » est plus adapté, cf. DEC-RANK.)
- **`apps/go-api/internal/analysis/rank_progression.go`** — `ComputeRankProgressionByChain`
  (`:74-97`) : algo PUR title-agnostic (arch-rules — 0 DB/HTTP/effet de bord). Ordre déterministe :
  type majoritaire d'abord (nb total de matchs du type desc, tie CSR d'abord), puis chaînes du type
  par `Matches` desc (tie clé de chaîne asc). `RankChainProgression.Matches` (`:45`) = nb de matchs
  de la chaîne. **À NE PAS MODIFIER** (le filtrage de pertinence = politique produit = couche
  SERVICE, symétrie stricte avec `buildDimension`). Ses tests (`rank_progression_test.go`) restent
  verts.
- **Tests service existants qui encodent le comportement ACTUEL (à réécrire, jamais skipper)** —
  `apps/go-api/internal/service/match_history_service_briefing_test.go` :
  - `TestBuildExplorerBriefing_RankedMonoChainProgression` (`:165`) : 15 matchs CSR chaîne unique →
    **survit** (15 ≥ seuil). Inchangé.
  - `TestBuildExplorerBriefing_RankedMultiChainNeverCrossed` (`:210`) : CSR 13 (≥10), LUSR
    arena_slayer 6 (< 10), LUSR btb 5 (< 10). Asserte 3 chaînes → **CASSE** avec DP-5 (seuil 10 →
    1 chaîne). L'intention réelle du test (isolation des paliers par chaîne, « never crossed ») est
    ORTHOGONALE au seuil → réécrire en portant arena_slayer et btb ≥ 10 matchs tout en gardant CSR
    type majoritaire (ex. CSR 25 / arena 12 / btb 10 : LUSR total 22 < 25 ; ordre csr→arena→btb
    inchangé ; cap 3 conserve les 3).
  - `TestBuildExplorerBriefing_RankedSmallSecondaryChainStillEmitted` (`:285`) : CSR 15 + LUSR btb
    3 ; asserte 2 chaînes (« aucune omise »). Son NOM et sa prémisse CONTREDISENT DP-5 → réécrire +
    RENOMMER `…SmallSecondaryChainOmitted` (btb 3 < 10 → omise → 1 chaîne CSR) + corriger le
    commentaire (`:286-287`, « PLUS de seuil de type secondaire » → il y a désormais un seuil).
  - `TestBuildExplorerBriefing_RankedStartInPlacement` (`:339`) / `…EndInPlacement` (`:368`) :
    RE-VÉRIFIER le nombre de matchs construits (boucle) ; s'il est < seuil, le porter ≥ seuil (le
    test cible le placement, pas la pertinence).
  - `TestBuildExplorerBriefing_RankedNoTierLabels` (`:310`, 15 matchs), `…NilWhenNoRatedRows`
    (`:396`, 0 rated → nil, fallback non déclenché) : **survivent**. Inchangés.
  - NOUVEAUX tests DP-5 à ajouter (§5 Phase 1) : fallback (toutes < seuil → chaîne principale
    conservée), plafond (> `RankedChainMaxCount` chaînes qualifiées → top N par matchs, ordre
    canonique).

**Conclusion du constat.** V6 = **backend léger d'abord** (DP-5 : filtrage de pertinence dans le
service, sans toucher l'algo pur, sans SQL, sans DTO), puis **frontend** (DP-1 highlight : 1 fichier
+ tests ; DP-2 triptyque : 2 classes ; DP-4 largeurs : flex dans le Strip), puis **i18n** (DP-3 :
11 clés FR+EN), puis **clôture**. Aucun changement OpenAPI/DTO → `generate-types` non requis (§4).
Aucune écriture DB → `-tags=integration` non requis (§4).

---

## 3. Décisions — pré-tranchées (fermes, ne pas re-débattre en exécution)

### Décisions produit (utilisateur, 2026-07-17/18 — reprises telles quelles)

- **DP-1.** Surlignage décile : RETIRER `score_label`, AJOUTER `team_mmr` (non inversé). Set final :
  `perf_score`, `kda`, `kills`, `deaths` (inversé), `team_mmr`.
- **DP-2.** Extrêmes min/max du triptyque « en blanc » (contraste plein theme-aware) et un peu plus
  gros ; moyenne inchangée (mise en avant, colorée).
- **DP-3.** TOUS les tooltips du bandeau en registre factuel/descriptif, ZÉRO vouvoiement ; pour le
  triptyque, « plus haut / plus bas » (jamais « votre meilleur / votre pire »). FR + EN, parité.
- **DP-4.** « Séries marquantes » ~10 % plus large, « Pic MMR » ~10 % plus étroite en compensation ;
  responsive et autres tuiles préservés ; gérer l'absence des conditionnelles (pas de trou).
- **DP-5.** Classement NON exhaustif : nombre pertinent de chaînes, comme « Par carte »/« Par mode »
  (seuil de matchs + top N), fallback « au moins la chaîne principale » (type majoritaire
  représenté), ordre déterministe conservé.

### Décisions techniques (architecte)

- **DEC-HL (DP-1 — set de highlight décile).** Dans `ExplorerMatchesTable.highlight.ts` :
  1. `ExplorerHighlightKey` (`:20`) → `'kills' | 'deaths' | 'kda' | 'perf_score' | 'team_mmr'`.
  2. `EXPLORER_INVERTED` (`:26-32`) : retirer `score_label`, ajouter `team_mmr: false` (haut =
     meilleur, comme Frags/FDA/Perf). Commentaire : `team_mmr` = MMR d'équipe, plus haut = meilleur.
  3. `explorerHlExtract` (`:48-56`) : retirer `score_label`, ajouter
     `team_mmr: (r) => r.team_mmr ?? null` (valeur brute, cohérent avec `kills`/`kda` ; `fmtMmr`
     n'est qu'un formateur monotone → l'arrondi d'affichage est cosmétiquement négligeable pour une
     bande de décile, cf. réserve pré-notée Découverte-2).
  4. Supprimer `ownTeamScore` (`:41-45`) — CODE MORT après retrait de `score_label` (CLAUDE.md §7).
     RE-VÉRIFIER le grep (0 lecteur hors test) avant suppression.
  5. `isExplorerHighlightKey`/`HL_KEYS`/`computeColumnDeciles`/`decileCellState`/`columnHighlightStyle`
     INCHANGÉS (dérivent des records). **Aucune modification de `ExplorerMatchesTable.tsx`** (le
     `<td>` `:803-804` est gaté par `isExplorerHighlightKey`, la colonne `team_mmr` existe déjà,
     conditionnelle H5).
  - Tests (`highlight.test.ts`) : bloc `ownTeamScore` (`:128-135`) SUPPRIMÉ ; dataset décile étendu
    à `team_mmr` (≥ 10 valeurs hétérogènes → haut surligné best, bas worst, non inversé) ;
    assertions `score_label` retirées. `ExplorerMatchesTable.test.tsx` : bloc MVP/LVP mis à jour
    (colonne MMR équipe surlignée, plus de `score_label`).
- **DEC-TRIPTYCH (DP-2 — extrêmes lisibles).** Dans `ExplorerBriefingTiles.tsx`, sur les DEUX
  `<span>` min/max de `MinMaxTriptych` (`:52`, `:58`) : `text-2xs` → `text-xs`,
  `text-muted-foreground` → `text-foreground`. Conserver `font-normal tabular-nums`. Le `<span>`
  central (moyenne, `:54-56`) INCHANGÉ. Aucune couleur hex. Mettre à jour le commentaire du
  composant (`:31-35`, « bornes min/max petites et discrètes (muted, poids normal) » → « bornes
  min/max lisibles (contraste plein, `text-foreground`), moyenne mise en avant colorée »).
- **DEC-TIP (DP-3 — tooltips factuels).** Réécrire les 11 clés dans `explorer.toml` (FR + EN) puis
  `node apps/web/scripts/build_i18n_manifests.mjs`. Registre : impersonnel, descriptif, aucun
  « vous/votre/vos » ni impératif de politesse FR ; EN sans « your/you » possessif. Textes FR/EN
  PAR DÉFAUT (l'exécutant peut affiner la FORMULATION, pas le REGISTRE) :

  | Clé | FR par défaut | EN par défaut |
  |---|---|---|
  | `tip_dimensions` | Cartes, modes ou sélections où les performances sont les meilleures et les moins bonnes. | Maps, modes or selections with the best and worst performances. |
  | `tip_context` | Résultats en solo et en escouade : nombre de matchs, taux de victoire et FDA par contexte. | Results solo vs in a squad: number of matches, win rate and KDA per context. |
  | `tip_win_rate` | Part de matchs gagnés dans la sélection. Le ruban sous la valeur répartit victoires, nuls, abandons et défaites ; survol pour le détail chiffré. | Share of matches won in the selection. The ribbon below the value splits wins, draws, DNFs and losses; hover for the detailed counts. |
  | `tip_fda` | Frags, Décès, Assistances : indicateur d'impact par match qui valorise frags et assistances et pénalise les décès (pas une simple division frags/décès). La tuile montre, dans la sélection, le plus bas · la moyenne (au centre, en couleur) · le plus haut. | Kills, Deaths, Assists: a per-match impact indicator that rewards kills and assists and penalises deaths (not a plain kills/deaths division). Across the selection, the tile shows lowest · average (centre, coloured) · highest. |
  | `tip_perf` | Score de performance de 0 à 100, relatif à l'historique du joueur (50 = niveau habituel). La tuile montre, dans la sélection, le plus bas · la moyenne (au centre, en couleur) · le plus haut. | Performance score from 0 to 100, relative to the player's own history (50 = usual level). Across the selection, the tile shows lowest · average (centre, coloured) · highest. |
  | `tip_duration` | Temps de jeu total cumulé sur la sélection. | Total play time across the selection. |
  | `tip_peak_rank` | Meilleur palier de classement atteint sur la sélection — pas le palier final. | Highest ranking tier reached over the selection — not the final tier. |
  | `tip_peak_mmr` | Meilleur niveau de compétence estimé (MMR) atteint sur la sélection. | Highest estimated skill level (MMR) reached over the selection. |
  | `tip_ranked` | Palier de classement en fin de sélection, palier de départ, et moyenne de points de classement gagnés ou perdus par match. | Ranking tier at the end of the selection, the starting tier, and the average ranking points gained or lost per match. |
  | `tip_streaks` | Séries extrêmes dans la sélection : la plus longue suite de victoires et la plus longue suite de défaites. | Extreme streaks in the selection: the longest run of wins and the longest run of losses. |
  | `tip_highlights` | Matchs marquants : dominations (large victoire), humiliations (large défaite), remontadas (victoire après avoir été mené), débandades (défaite après avoir mené), contre-remontadas (remontée adverse stoppée après avoir mené). | Standout matches: dominations (large win), humiliations (large loss), comebacks (win after trailing), collapses (loss after leading), counter-comebacks (opponent's comeback stopped after leading). |

  Note : `tip_duration` / `tip_peak_mmr` étaient déjà factuels ; harmonisés « matchs affichés » →
  « la sélection » pour la cohérence (aucun vouvoiement à retirer). Le mot « FDA » (français
  canonique du projet) est conservé, « KDA » côté EN (convention i18n existante).
- **DEC-WIDTHS (DP-4 — largeurs inégales, approche retenue).** Comparaison des options (mission) :
  - `grid-column: span N` ciblé : granularité 2× (une tuile prend 2 tracks), incompatible avec un
    écart fin ~10 %. REJETÉ.
  - largeur `min`/`max` par tuile dans la grille auto-fit : les tracks `1fr` restent uniformes
    (largeur définie sur le conteneur, pas par item) → n'obtient pas un +10 % fiable. REJETÉ.
  - **flex avec `flex-basis`/`flex-grow` différenciés : RETENU** — seul moyen d'un écart fin tout en
    gardant le remplissage pleine largeur (grow) et le wrap responsive, et en absorbant l'absence
    des conditionnelles (reflow, aucun trou).
  - **Implémentation (TOUT dans `ExplorerBriefingStrip.tsx`, zéro modif de composant partagé — car
    `KpiCard` concatène `className` sans `twMerge`, cf. §2.4)** :
    1. Conteneur socle (`:100`) : `grid gap-2 grid-cols-2 sm:[…auto-fit,minmax(150px,1fr)…]` →
       `flex flex-wrap gap-2`.
    2. Constante locale `SOCLE_TILE = 'grow basis-[150px] min-w-0'` (réplique `minmax(150px,1fr)` :
       largeur cible 150px, grandit pour remplir la ligne, sans overflow des `tabular-nums`).
    3. Envelopper CHAQUE tuile socle dans `<div className={SOCLE_TILE}>…</div>` (les `KpiCard`
       remplissent leur wrapper : `<div>` bloc, largeur 100% ; `h-full` déjà porté).
    4. Différenciation : la tuile « Séries marquantes » dans `<div className="grow-[1.15]
       basis-[168px] min-w-0">` (~+10 %) ; « Pic MMR » dans `<div className="grow-[0.9]
       basis-[136px] min-w-0">` (~−10 %). Valeurs de `basis`/`grow` AJUSTABLES en revue visuelle
       (comme `MIN_DECILE_SAMPLE`). Les conditionnelles étant construites dans `conditionalTiles`
       (`:85-96`), y appliquer le wrapper approprié par tuile (Streaks/PeakRank/PeakMmr).
    5. Absence gérée nativement : si « Pic MMR » (ou « Séries ») absente, `flex-wrap`+`grow`
       redistribue l'espace, la rangée reste pleine SANS trou. Sur mobile, `basis-[150px]` + wrap ≈
       l'ancien `grid-cols-2` (≈ 2 tuiles/ligne < 316px → 1). À valider en revue visuelle.
  - *Alternative documentée* (si l'exécutant préfère ne pas ajouter de wrappers DOM) : threader un
    prop `className?` via `BriefingTile`→`KpiCard` ET introduire un helper de fusion de classes
    (`twMerge`) pour un override fiable de `basis-*`. REJETÉE par défaut (touche des composants
    partagés + nouvelle dépendance/util ; churn > wrappers locaux). DÉFAUT = wrappers dans le Strip.
- **DEC-RANK (DP-5 — pertinence des chaînes, dans le SERVICE).** Placement = `buildBriefingRanked`
  (`match_history_service_briefing_ranked.go`), PAS l'algo pur (miroir de `buildDimension` où le
  seuil vit dans le service). `analysis.ComputeRankProgressionByChain` reste inchangé.
  1. Constantes (bloc const de `match_history_service_briefing.go:32-43`, cohésion avec les seuils
     dimensions) : `MinRankedChainMatches = MinDimensionGroupMatches` (= 10 par référence, PAS un
     littéral dupliqué ; commentaire : « seuil de pertinence d'une chaîne de Classement, aligné sur
     le pattern dimensions ; découpler en littéral si la revue veut un seuil ranked distinct ») ;
     `RankedChainMaxCount = 3` (« nombre max de chaînes affichées, parité `dimensionTopFlopCount` ;
     ajustable revue »).
  2. Règle EXACTE, dans `buildBriefingRanked`, après `progs :=
     analysis.ComputeRankProgressionByChain(samples)` (et avant le mapping vers `Kinds`) :
     - `qualified` = progs avec `Matches >= MinRankedChainMatches`.
     - **Fallback** : si `len(qualified) == 0 && len(progs) > 0` → `qualified = progs[:1]` (la chaîne
       principale = première en ordre canonique = plus grande chaîne du type majoritaire → garantit
       la représentation du type majoritaire ; jamais tout masquer quand une progression existe).
     - **Plafond** : si `len(qualified) > RankedChainMaxCount`, garder les `RankedChainMaxCount`
       chaînes aux `Matches` les plus élevés (« le plus joué ») via un helper
       `selectTopByMatches(qualified, RankedChainMaxCount)` : sélection stable par `Matches` desc
       (tie-break = index canonique croissant), puis **restitution dans l'ordre canonique** pour
       l'affichage (« ordre déterministe conservé »). Réaliste : jusqu'à 5 chaînes (CSR ranked + 4
       chaînes LUSR) → le plafond peut mordre.
     - Mapper ensuite `qualified` (au lieu de `progs`) vers `Kinds`. Le log Debug « no tier label »
       (`:39-44`) ne concerne plus que les chaînes retenues.
     - Bord : `len(progs) == 0` → nil (inchangé) ; le fallback ne se déclenche QUE s'il existe au
       moins une progression.
  3. Taille : `match_history_service_briefing_ranked.go` 83 L + helper `selectTopByMatches`
     (~15 L) + filtrage (~8 L) ≈ 106 L (≤ 500) ; `buildBriefingRanked` ≤ 80 L (vérifier ; extraire
     si dépassement). `selectTopByMatches` peut être générique `[T]` avec un accès `Matches` via
     closure, ou spécifique `[]analysis.RankChainProgression` (choix d'exécution, consigné §6).
  4. **Aucun changement DTO** : `ExplorerBriefingRanked.Kinds` émet simplement MOINS d'entrées
     (même forme) → pas d'OpenAPI, pas de `generate-types`, pas de changement frontend (`RankedBlock`
     rend les kinds reçus).
  - *Alternative documentée* : plafond par simple `qualified[:RankedChainMaxCount]` en ordre
    canonique (plus simple, déterministe) — mais peut écarter la plus grande chaîne SINGLE d'un type
    minoritaire ; moins fidèle à « top N par nombre de matchs (le plus joué) ». DÉFAUT =
    `selectTopByMatches`. L'exécutant peut retenir l'alternative en la consignant §6 (les deux sont
    déterministes ; le test de plafond fixe le comportement retenu).

---

## 4. Périmètre

**Dans le périmètre :**
- Backend `apps/go-api` (DP-5 / DEC-RANK) : `service/match_history_service_briefing_ranked.go`
  (filtrage `qualified` + fallback + `selectTopByMatches`), `service/match_history_service_briefing.go`
  (2 constantes `MinRankedChainMatches`/`RankedChainMaxCount`), tests
  `service/match_history_service_briefing_test.go` (réécriture des 2 tests contradictoires +
  nouveaux tests fallback/plafond + re-vérif placement).
- Frontend `apps/web` : `ExplorerMatchesTable.highlight.ts` (DP-1 : records + suppression
  `ownTeamScore`), `ExplorerBriefingTiles.tsx` (DP-2 : 2 classes du triptyque),
  `ExplorerBriefingStrip.tsx` (DP-4 : flex + wrappers), manifest `explorer.toml` (DP-3 : 11 clés) +
  régénération, tests (`highlight.test.ts`, `ExplorerMatchesTable.test.tsx`,
  `ExplorerBriefingStrip.test.tsx`).
- Vérification (reprise UTILISATEUR), journal du plan, `.ai/thought_log.md`, changelog
  (`docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`).

**Hors périmètre (noter en §6 Découvertes si rencontré, NE PAS traiter) :**
- L'algo pur `analysis.ComputeRankProgressionByChain` et ses tests (`rank_progression_test.go`) —
  INCHANGÉS (la pertinence est une politique produit de la couche service).
- Tout autre calcul du briefing (dimensions, context, streaks, dominance, baseline, scope Min/Max
  V5) — inchangé.
- Le masquage MMR global (chantier séparé) — DP-1 n'ajoute le surlignage `team_mmr` que si la
  colonne est présente (déjà masquée H5 par `providesTeamMmr`).
- Le split du god-file `ExplorerMatchesTable.tsx` (854 L — V5 Découverte-13) ; DP-1 ne le touche PAS
  (highlight.ts uniquement).
- La 2ᵉ copie inline du style best/worst (`SquadImpactScoreboard.tsx`, dette V4 Découverte-17).
- Dette lint pré-existante (baseline gelée) ; tout Python (interdit) ; SQLite (interdit).

**`generate-types` NON requis (justification).** Aucun changement d'`api/openapi.yaml` ni de DTO :
DP-1/2/4 sont purement frontend (le champ `team_mmr` existe déjà sur `ExplorerMatchRow`) ; DP-3 est
i18n (aucune clé neuve, aucun schéma) ; DP-5 émet MOINS d'entrées dans le DTO existant
`ExplorerBriefingRanked` (forme inchangée). Aucune ré-émission Huma → `TestOpenAPISchemaDrift` non
concerné.

**`-tags=integration` NON requis (justification).** DP-5 est une agrégation/filtrage EN MÉMOIRE sur
des `RankChainProgression` déjà calculées à partir de raw rows déjà scannées : aucune requête SQL
modifiée, **aucune écriture, aucun writer/lease, aucune table créée, aucune migration**. Les tests
anti-ART (`-tags=integration`) couvrent les écritures per-match — sans objet ici. La suite standard
`cd apps/go-api && go test ./...` (SÉQUENTIELLE, inclut le package `service` + vet) +
`make go-api-lint` suffit (règle CLAUDE.md : `-tags=integration` OBLIGATOIRE seulement avant
livraison sync/persist). Si l'exécution constate qu'un test service touché est gardé par le tag
integration, le noter §6 et l'exécuter — mais aucune écriture n'est planifiée.

---

## 5. Phases (ordre strict — une étape CLOSE avant la suivante)

> Clôture d'étape = gate passé (commandes exactes, sorties propres — jamais de test skippé/
> désactivé) + tous les items statués `[x]` fait / `[~]` couvert ailleurs (réf) / `[!]` non traité
> (justif écrite) + plan mis à jour (cases + journal) + entrée `.ai/thought_log.md` + point d'étape
> utilisateur. Aucune case vide à la clôture. Zéro fix hors périmètre (→ §6).
>
> Notes d'exécution : commandes `go` SÉQUENTIELLES (jamais concurrentes — cache Windows ; tuer les
> `link.exe` orphelins). Après toute édition de `explorer.toml` : `node
> apps/web/scripts/build_i18n_manifests.mjs` AVANT `make check-types`. vitest →
> `dangerouslyDisableSandbox=true`. Purger `apps/web/node_modules/.tmp` avant `make check-types`.
> Commits par phase (skill `plan-execution`) ; le superviseur exécute la clôture git (push `main` =
> deploy auto → seulement après revue visuelle utilisateur).

### Phase 0 — Cadrage & re-vérification sur pièces (rapide)

- [x] Confirmer `git branch --show-current` = `feat/explorer-briefing-compact` (sinon la retrouver
      via `git log --oneline -10` ; NE PAS reprendre sur `main` ni une branche de train). CGO OK
      (CGO_ENABLED=1, gcc msys64) pour le gate Go de la Phase 1.
      → FAIT : branche = `feat/explorer-briefing-compact` ; `git status` propre (seul le plan V6
      non traqué) ; CGO validé au gate Phase 1 (go test compile le driver DuckDB).
- [x] Re-vérifier §2 sur pièces (rouvrir chaque fichier:ligne — le code a pu bouger depuis le
      2026-07-18) ; consigner tout décalage en §6. Points sensibles : garde `isExplorerHighlightKey`
      au `<td>` (§2.1) ; nombre de matchs des tests placement ranked (§2.5) ; présence du helper de
      fusion de classes (§2.4, attendu = absent).
      → FAIT : AUCUN décalage (Découverte-8). `<td>` gaté par `isExplorerHighlightKey`
      (`ExplorerMatchesTable.tsx:803-804`) ; tests placement ranked = 15 matchs chacun (≥ seuil) ;
      aucun `cn`/`clsx`/`twMerge` (grep = 0) ; `ownTeamScore` = 2 lecteurs source (déf + extracteur
      `score_label`) hors test.
- [x] Confirmer les défauts techniques : DEC-WIDTHS (flex + wrappers, `basis` 150/168/136,
      `grow` 1/1.15/0.9) ; DEC-RANK (`MinRankedChainMatches = MinDimensionGroupMatches` = 10,
      `RankedChainMaxCount = 3`, `selectTopByMatches`) ; DEC-TIP (textes FR/EN §3). Consigner au
      journal.
      → FAIT : `MinDimensionGroupMatches = 10` (`:40`), `dimensionTopFlopCount = 3` (`:42`),
      `selectTopFlop[T]` générique (`:376`) confirmés ; défauts appliqués tels quels.

Gate Phase 0 : branche correcte ; constat re-vérifié ; défauts confirmés. Pas de gate de build
(aucun code applicatif modifié). Pas de commit (ou commit doc du plan si le superviseur le demande).
→ PASSÉ (2026-07-18).

### Phase 1 — Backend : Classement pertinent (moyen) — DP-5 / DEC-RANK

- [x] **1a (constantes).** `match_history_service_briefing.go` (bloc const `:32-43`) : ajouter
      `MinRankedChainMatches = MinDimensionGroupMatches` et `RankedChainMaxCount = 3`, chacune
      commentée (pertinence chaîne alignée dimensions ; plafond parité `dimensionTopFlopCount`).
      → FAIT (`:43-50`). `MinRankedChainMatches = MinDimensionGroupMatches` (par référence, pas de
      littéral 10 dupliqué) ; `RankedChainMaxCount = 3`.
- [x] **1b (filtrage service).** `match_history_service_briefing_ranked.go` : dans
      `buildBriefingRanked`, après `ComputeRankProgressionByChain`, calculer `qualified`
      (`Matches >= MinRankedChainMatches`), appliquer le **fallback** (`len(qualified)==0 &&
      len(progs)>0` → `progs[:1]`) puis le **plafond** `selectTopByMatches(qualified,
      RankedChainMaxCount)` (top N par `Matches`, restitution en ordre canonique) ; mapper
      `qualified` vers `Kinds`. Ajouter `selectTopByMatches`. Fonctions ≤ 80 L, fichier ≤ 500 L.
      → FAIT. Logique de pertinence EXTRAITE dans `rankChainsByRelevance(progs)` (SRP, garde
      `buildBriefingRanked` ~30 L) ; `buildBriefingRanked` mappe `qualified`. `selectTopByMatches`
      ajouté (tri stable d'INDICES par `Matches` desc, tie-break index canonique croissant, puis
      restitution en ordre canonique). Import `sort` ajouté. Fichier ~130 L (≤ 500), fonctions
      ≤ 30 L (≤ 80). Choix d'exécution consigné §6 (Découverte-9).
- [x] **1c (tests service).** `match_history_service_briefing_test.go` :
      - RÉÉCRIRE `TestBuildExplorerBriefing_RankedMultiChainNeverCrossed` (porter arena_slayer/btb
        ≥ 10, CSR type majoritaire — ex. 25/12/10 ; l'isolation des paliers reste assertée).
        → FAIT : CSR 25 / arena_slayer 12 / btb 10 ; paliers isolés (Or I→Or III, Argent II→Argent V,
        Or III→Or I) + co-signage delta conservés ; 3 chaînes, plafond non mordant.
      - RÉÉCRIRE + RENOMMER `…SmallSecondaryChainStillEmitted` → `…SmallSecondaryChainOmitted`
        (CSR 15 + btb 3 → 1 chaîne CSR ; corriger le commentaire du seuil). → FAIT (btb 3 < seuil
        → omise ; commentaire corrigé « PLUS de seuil » → « seuil de pertinence DP-5 »).
      - AJOUTER `…RankedFallbackKeepsPrincipalChain` (toutes chaînes < seuil, ≥ 1 progression → la
        chaîne principale/type majoritaire est conservée, 1 kind). → FAIT (CSR 8 + LUSR btb 5, tous
        < 10 → fallback conserve csr/ranked, 1 kind).
      - AJOUTER `…RankedCapsToMostPlayed` (> `RankedChainMaxCount` chaînes qualifiées → seules les
        top N par matchs, en ordre canonique). → FAIT (CSR 40 + arena_slayer 15 + arena_objectif 12
        + btb 10 → 4 qualifiées, top 3 = csr/arena_slayer/arena_objectif en ordre canonique, btb
        écartée).
      - RE-VÉRIFIER `…StartInPlacement`/`…EndInPlacement` : porter le compte ≥ seuil si nécessaire.
        → RE-VÉRIFIÉ : 15 matchs chacun (≥ seuil 10) → INCHANGÉS (aucun portage requis, Découverte-8).
      Aucun test skippé. `analysis/rank_progression_test.go` NON modifié (reste vert).
      → FAIT : aucun skip ; `rank_progression_test.go` intact.

Gate Phase 1 : `cd apps/go-api && go test ./...` = 0 (SÉQUENTIEL ; tests 1c inclus) ;
`make go-api-lint` = 0 ; grep : `MinRankedChainMatches`/`RankedChainMaxCount`/`selectTopByMatches`
présents ; `ComputeRankProgressionByChain` NON modifié (diff = 0 sur `analysis/rank_progression.go`).
`generate-types`/`-tags=integration` NON requis (§4).
→ PASSÉ (2026-07-18) : `go test ./...` EXIT 0, 0 FAIL (service 12.7s) ; `make go-api-lint` EXIT 0 ;
`go vet ./internal/service/...` 0 ; golangci-lint 0 issue sur les 3 fichiers briefing ; symboles
présents ; `git diff --stat rank_progression.go` = vide.

### Phase 2 — Frontend : highlight, triptyque, largeurs (moyen) — DP-1 / DP-2 / DP-4

- [x] **2a (highlight — DP-1/DEC-HL).** `ExplorerMatchesTable.highlight.ts` : `ExplorerHighlightKey`
      → retirer `score_label`, ajouter `team_mmr` ; `EXPLORER_INVERTED.team_mmr = false` ;
      `explorerHlExtract.team_mmr = (r) => r.team_mmr ?? null` ; supprimer l'extracteur `score_label`
      ET la fonction `ownTeamScore` (RE-VÉRIFIER grep : 0 lecteur hors test). Aucune modif de
      `ExplorerMatchesTable.tsx` (vérifier la garde `<td>`).
      → FAIT. `ExplorerHighlightKey` (`:20`), `EXPLORER_INVERTED.team_mmr` (`:32`),
      `explorerHlExtract.team_mmr` (`:45`) ; `ownTeamScore` supprimé (grep = 0). `ExplorerMatchesTable.tsx`
      NON modifié (garde `<td>` `isExplorerHighlightKey` `:803-805` inchangée).
- [x] **2b (triptyque — DP-2/DEC-TRIPTYCH).** `ExplorerBriefingTiles.tsx` : sur les 2 `<span>`
      min/max de `MinMaxTriptych`, `text-2xs`→`text-xs` et `text-muted-foreground`→`text-foreground`.
      `<span>` central inchangé. Commentaire du composant mis à jour.
      → FAIT (`:53`, `:59` = `text-xs … text-foreground`) ; span central (moyenne) inchangé ;
      commentaire réécrit (« lisibles, contraste plein »).
- [x] **2c (largeurs — DP-4/DEC-WIDTHS).** `ExplorerBriefingStrip.tsx` : conteneur socle grid→`flex
      flex-wrap gap-2` ; constante `SOCLE_TILE` ; envelopper chaque tuile socle ; Streaks (wrapper
      large ~+10 %) et Pic MMR (wrapper étroit ~−10 %) différenciés ; conditionnelles wrappées à la
      construction de `conditionalTiles`. Vérifier l'absence de trou (Streaks/Pic MMR absentes).
      → FAIT. Socle = `flex flex-wrap gap-2` (`:121`) ; `SOCLE_TILE`/`SOCLE_TILE_WIDE`
      (basis-168/grow-1.15)/`SOCLE_TILE_NARROW` (basis-136/grow-0.9) ; 5 tuiles socle + 3
      conditionnelles wrappées ; `flex-wrap`+`grow` gèrent l'absence sans trou (validation visuelle
      utilisateur). Aucun composant partagé modifié.
- [x] **2d (tests).** `highlight.test.ts` (supprimer bloc `ownTeamScore` ; dataset décile
      `team_mmr` ; retirer `score_label`). `ExplorerMatchesTable.test.tsx` (bloc MVP/LVP : MMR
      équipe surlignée, plus de `score_label`). `ExplorerBriefingStrip.test.tsx` (classes triptyque
      `text-xs`/`text-foreground` ; structure socle flex — décomptes `KpiCard`/barres inchangés).
      Aucun test skippé.
      → FAIT. `highlight.test.ts` : describe `ownTeamScore` supprimé + import retiré + it `team_mmr`
      ajouté (p10=1000/p90=1800). `ExplorerMatchesTable.test.tsx` : it décile MMR équipe + Score
      jamais surligné (valeurs < 1000 pour fmtMmr sans séparateur). `ExplorerBriefingStrip.test.tsx` :
      3 tests grid→flex (socle ciblé par `[class*="flex-wrap"]`, 8 flex-items/8 barres inchangés) +
      assertions DP-2 (bornes `text-foreground`/`text-xs`, plus `text-muted-foreground`). Aucun skip.

Gate Phase 2 : `make check-types` = 0 (`.tmp` purgé) ; `make test-web` (dangerouslyDisableSandbox)
vert ; `cd apps/web && npm run lint` = 0 erreur ; greps : 0 `score_label` dans
`ExplorerMatchesTable.highlight.ts` ; 0 `ownTeamScore` (code + suppression du test) ; `team_mmr`
présent dans `EXPLORER_INVERTED` + `explorerHlExtract` ; 0 `text-2xs`/`text-muted-foreground` sur les
extrêmes du triptyque (2 `text-foreground`) ; conteneur socle = `flex flex-wrap` (0 `auto-fit` sur
le socle) ; 0 hex ni classe Tailwind couleur nouvelle sous `features/explorer`.
→ PASSÉ (2026-07-18) : check-types EXIT 0 ; test-web EXIT 0 (264 fichiers, 2329 passed, 14 skipped
préexistants) ; `npm run lint` 0 erreur (68 warnings baseline gelée, 0 sur les 6 fichiers touchés —
eslint ciblé EXIT 0) ; les 6 greps de clôture conformes (voir ci-dessus).

### Phase 3 — i18n : tooltips factuels (rapide-moyen) — DP-3 / DEC-TIP

- [x] **3a (réécriture).** `explorer.toml` : réécrire les 11 clés `tip_*` FR + EN (textes §3
      DEC-TIP), zéro vouvoiement FR, parité EN. `node apps/web/scripts/build_i18n_manifests.mjs`.
      → FAIT : 11 clés réécrites (textes DEC-TIP par défaut, registre factuel/descriptif) ;
      manifests régénérés (diff = 11 clés sur `generated/explorer.ts`, aucune clé neuve — 224 clés).
      « matchs affichés » harmonisé en « la sélection » (Découverte-7).
- [x] **3b (tests).** RE-VÉRIFIER qu'aucun test n'asserte un substrat de tooltip supprimé (grep
      « votre »/« vous »/« survolez » dans les tests explorer) ; ajuster si besoin. Garde-rail
      `explorerBriefingTerminology.guard.test.ts` vert.
      → FAIT : grep tests explorer = 0 occurrence de « votre/vos/vous/survolez » (les tests Strip
      utilisent le stub `t=(key)=>key`, jamais les valeurs). Aucun ajustement requis. Garde-rail
      terminologie vert (interdit uniquement « Par playlist »/« Pronostic »/« Prognosis », absents).

Gate Phase 3 : `build_i18n_manifests.mjs` (diff = 11 clés attendues, aucune clé neuve) ;
`make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 erreur ; greps : 0 « votre »/
« vous »/« Vos »/« survolez » dans les 11 `tip_*` FR de `explorer.toml` ET de
`generated/explorer.ts` ; garde-rail terminologie vert.
→ PASSÉ (2026-07-18) : `build_i18n_manifests.mjs` EXIT 0 (11 clés, 0 neuve) ; `make check-types`
EXIT 0 (post-régén) ; tests explorer + garde-rail EXIT 0 (19 fichiers, 143 tests) ; greps
vouvoiement TOML + generated + tests = 0. `make test-web` complet + `npm run lint` complet
re-confirmés au gate final Phase 4 (changement Phase 3 isolé au TOML/generated).

### Phase 4 — Clôture (rapide) — changelog + delivery-checklist

- [x] **4a (changelog).** `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md`, entrée `[Unreleased]` v7.0 :
      bullet React « Explorer — briefing V6 » (décile MVP/LVP sur MMR équipe au lieu du score,
      extrêmes de triptyque lisibles, tooltips factuels sans vouvoiement, tuiles « Séries
      marquantes »/« Pic MMR » à largeurs différenciées) + bullet Go (« Classement : chaînes
      limitées aux plus jouées au-dessus d'un seuil, fallback chaîne principale »). Parité EN/FR
      même commit.
      → FAIT : bullet React « Explorer — briefing V6 » + bullet Go « relevant ranking chains (V6) »
      ajoutés dans les DEUX changelogs (Added React après V5, Added Go API après DTO V5), à parité
      EN/FR.
- [x] **4b (clôture).** Dérouler `delivery-checklist`. Passe finale des gates §1.6 verte en une
      fois. Entrée `.ai/thought_log.md` finale. Point d'étape utilisateur. NE PAS committer la
      livraison finale sans autorisation (merge `main` = deploy prod auto → après revue visuelle
      utilisateur).
      → FAIT : `delivery-checklist` déroulé (complétude, tests Go/front, logging, archi ≤500L/≤80L,
      couleurs/i18n) ; passe finale des gates verte (ci-dessous) ; entrée `.ai/thought_log.md`
      [2026-07-18] ajoutée (statut Complété) ; point d'étape = rapport final. NON committé (clôture
      git + vérif CI branche = superviseur ; merge main = deploy prod auto APRÈS revue visuelle).

Gate Phase 4 : `cd apps/go-api && go test ./...` = 0 ; `make go-api-lint` = 0 ; `make check-types`
= 0 ; `make test-web` vert ; `npm run lint` = 0 erreur ; greps de clôture (Phases 1-3) verts ;
changelog EN + FR à jour ; chaque item statué ; entrée thought_log présente.
→ PASSÉ (2026-07-18, passe finale une fois) : `go test ./...` EXIT 0 (0 FAIL) ; `make go-api-lint`
+ `go vet ./...` EXIT 0 ; `make check-types` (.tmp purgé) EXIT 0 ; `npx vitest run` EXIT 0 (264
fichiers, 2329 passed, 14 skipped préexistants) ; `npm run lint` 0 erreur (68 warnings baseline) ;
greps clôture Phases 1-3 tous verts ; changelogs EN+FR à jour ; toutes les cases statuées ; entrée
thought_log présente.

---

## 6. Découvertes (à remplir en exécution — ne pas traiter hors périmètre)

- **Découverte-1 (pré-notée) — DP-1 sans modif table.tsx.** Le `<td>`
  (`ExplorerMatchesTable.tsx:803-804`) est gaté par `isExplorerHighlightKey(colId)` ; ajouter
  `team_mmr` aux records de `highlight.ts` suffit. La colonne `team_mmr` existe déjà (conditionnelle
  `providesTeamMmr`, masquée H5) → le surlignage ne s'applique que si la colonne est présente.
- **Découverte-2 (pré-notée) — extracteur `team_mmr` = valeur brute.** `explorerHlExtract.team_mmr`
  = `r.team_mmr ?? null` (brut), alors que la cellule affiche `fmtMmr(...)`. `fmtMmr` étant un
  formateur monotone (arrondi), un écart de bande de décile au bord est cosmétiquement négligeable
  (même réserve que `kda`/`perf` en V4/V5). Ne pas « corriger » en formatant l'extracteur.
- **Découverte-3 (pré-notée) — `ownTeamScore` code mort.** Unique lecteur = extracteur `score_label`
  (retiré DP-1) + son bloc de test. Supprimer la fonction ET le describe `ownTeamScore`
  (`highlight.test.ts:128-135`) — CLAUDE.md §7 (zéro code mort). RE-VÉRIFIER le grep avant.
- **Découverte-4 (pré-notée) — 2 tests ranked contradictoires.** `…RankedMultiChainNeverCrossed`
  (arena 6/btb 5 < 10) et `…RankedSmallSecondaryChainStillEmitted` (btb 3 < 10) encodent l'absence
  de seuil ; DP-5 les invalide → réécriture/renommage (Phase 1c). Le second a un NOM et un
  commentaire à retourner (« still emitted » → « omitted » ; « PLUS de seuil » → « seuil de
  pertinence »).
- **Découverte-5 (pré-notée) — pas de helper de fusion de classes.** Aucun `cn`/`clsx`/`twMerge`
  dans le repo (grep = 0) ; `KpiCard`/`BriefingTile` concatènent `className`. DEC-WIDTHS place donc
  la largeur dans des wrappers du Strip (chaque wrapper porte UNE seule `basis-*`/`grow-*`, aucun
  override conflictuel) plutôt que de threader un override via BriefingTile.
- **Découverte-6 (pré-notée) — `MinRankedChainMatches` par référence.** `= MinDimensionGroupMatches`
  (évite un littéral 10 dupliqué, CLAUDE.md §6). Si la revue veut un seuil ranked distinct, le
  découpler en littéral nommé (le commentaire le prévoit).
- **Découverte-7 (pré-notée) — `tip_duration`/`tip_peak_mmr` déjà factuels.** Aucun vouvoiement à
  retirer ; harmonisés « matchs affichés » → « la sélection » pour la cohérence de registre (pas un
  changement de sens). Aucune régression de parité.

- **Découverte-8 (exécution, Phase 0) — re-vérification sur pièces : AUCUN décalage vs §2.** Tous
  les ancrages fichier:ligne du plan (2026-07-18) correspondent à l'état du code au moment de
  l'exécution. Points sensibles confirmés : (a) le `<td>` `ExplorerMatchesTable.tsx:803-805` est
  gaté `isExplorerHighlightKey(colId)` → DP-1 ne touche PAS ce fichier ; (b) `team_mmr` déjà dans
  `RIGHT_ALIGNED_COLUMNS` (`:146`) et colonne conditionnelle `providesTeamMmr` (`:613`) ; (c) tests
  `RankedStartInPlacement`/`RankedEndInPlacement` = 15 matchs CSR chacun (≥ seuil 10) → survivent
  SANS retouche du compte (seule une re-vérif, pas de portage) ; (d) `ownTeamScore` = 2 lecteurs
  source uniquement (déf `:41` + extracteur `score_label` `:55`), reste = test → code mort après
  DP-1 ; (e) `MinDimensionGroupMatches = 10` (`:40`), `dimensionTopFlopCount = 3` (`:42`),
  `selectTopFlop[T]` générique (`:376`) confirmés ; (f) aucun helper `cn`/`clsx`/`twMerge` (grep = 0)
  → DEC-WIDTHS par wrappers Strip validé.

- **Découverte-9 (exécution, Phase 1) — choix d'exécution DEC-RANK.** (a) `selectTopByMatches`
  implémenté SPÉCIFIQUE `[]analysis.RankChainProgression` (pas générique `[T]`) : plus lisible, un
  seul appelant, aucune duplication à centraliser. (b) La logique de pertinence (seuil + fallback +
  plafond) est EXTRAITE dans `rankChainsByRelevance(progs)` plutôt qu'inline dans
  `buildBriefingRanked` — SRP (arch-rules), garde l'appelant court et le rend testable isolément si
  besoin. (c) Plafond retenu = `selectTopByMatches` (top N par matchs, DÉFAUT du plan), PAS
  l'alternative `qualified[:N]` : fidèle à « le plus joué » même quand une grande chaîne d'un type
  minoritaire précède une petite chaîne du type majoritaire en ordre canonique. Restitution en
  ordre canonique préservée (tri d'indices + filtre keep).

Consigner ici tout décalage fichier:ligne vs §2, tout lecteur i18n/test inattendu, tout compte de
matchs des tests placement < seuil, tout choix d'exécution (helper `selectTopByMatches` générique
vs spécifique ; alternative plafond `qualified[:N]`), toute dette repérée hors périmètre. Ne pas
corriger hors items scopés.

---

## 7. Protocole de reprise de session

1. `git branch --show-current` doit être `feat/explorer-briefing-compact` (sinon la retrouver via
   `git log --oneline -10`). Ne jamais reprendre sur `main` ni une branche de train.
2. Lire ce fichier : la dernière phase dont le **Gate** est passé est close ; reprendre à la
   première non close. Les cases `[ ]` d'une phase non close = travail restant.
3. Lire l'entrée `.ai/thought_log.md` la plus récente de ce chantier (avancement + décisions, dont
   DEC-WIDTHS valeurs `basis`/`grow`, DEC-RANK seuil/plafond, choix d'exécution consignés §6).
4. Re-vérifier sur pièces les fichier:ligne de la phase courante AVANT d'éditer ou de cocher (le
   code a pu bouger).
5. Ne jamais commencer une phase N+1 tant que le Gate de N n'est pas vert.

---

## 8. Effort estimé & dépendances

| Bloc | Phase | Effort | Couche |
|---|---|---|---|
| Cadrage + re-vérif | 0 | Rapide | git + plan |
| Classement pertinent (seuil + plafond + fallback) | 1 | Moyen | service Go + tests |
| Highlight (team_mmr) + triptyque (extrêmes) + largeurs socle | 2 | Moyen | front + tests |
| Tooltips factuels (11 clés) | 3 | Rapide-Moyen | i18n + tests |
| Changelog + clôture | 4 | Rapide | docs + gates |

**Dépendances inter-phases.** Les 5 DP sont largement INDÉPENDANTES : DP-5 (Phase 1, backend) ne
touche aucun champ consommé par le frontend (émet moins d'entrées d'un DTO inchangé) ; DP-1/2/4
(Phase 2) et DP-3 (Phase 3) sont frontend/i18n, sans dépendance sur la Phase 1. L'ordre backend →
frontend → i18n → clôture suit la consigne (Go d'abord car gate `go test` distinct) et sépare les
gates. Phase 4 en dernier. **Points à confirmer par l'utilisateur** : aucun blocage — les 5 DP sont
TRANCHÉES ; les micro-défauts techniques (largeurs `basis` 168/136 & `grow` 1.15/0.9 ; seuil ranked
10 ; plafond 3) sont appliqués tels quels et AJUSTABLES en revue visuelle. **Aucun déploiement
prod** dans ce chantier (le merge `main` = deploy auto reste la décision de l'utilisateur, après
revue visuelle).

---

## À vérifier visuellement par l'utilisateur (repris par l'utilisateur, PAS une tâche agent)

Sur l'Explorer mode Matchs (dev local `:8000`/vite), profils réels halo_infinite (LUSR + CSR) +
un titre H5 (dégradation MMR/ranked) + un état low_sample + spot-check EN :

1. **Décile MVP/LVP (DP-1)** : la colonne **MMR équipe** est surlignée en bande de décile (haut =
   vert/best, bas = rouge/worst, teinte douce) ; la colonne **Score** n'est PLUS surlignée ; sur H5
   (colonne MMR masquée) aucun surlignage MMR, aucune erreur console.
2. **Triptyques (DP-2)** : sur FDA et Perf, les valeurs **min et max** sont lisibles (blanc en thème
   sombre, quasi-noir en clair) et un peu plus grandes ; la **moyenne** centrale reste dominante et
   colorée (aucune concurrence visuelle) ; bornes absentes → moyenne seule sans « — » parasite.
3. **Tooltips (DP-3)** : au survol de l'icône (i) de CHAQUE tuile/section (FDA, Perf, Taux de
   victoire, Durée, Séries marquantes, Pic rang, Pic MMR, Par carte/mode, Par contexte, Classement,
   Moments forts) : registre factuel/descriptif, AUCUN « vous/votre/vos » ni « survolez » ; le
   triptyque dit « plus bas · moyen · plus haut ». EN en parité (pas de « your/you » possessif).
4. **Largeurs socle (DP-4)** : « Séries marquantes » visiblement un peu plus large, « Pic MMR » un
   peu plus étroite ; les autres tuiles inchangées ; rangée pleine largeur SANS trou ; tester les
   cas où « Pic MMR » et/ou « Séries marquantes » sont absentes (pas de déséquilibre) ; responsive
   mobile ≈ 2 tuiles/ligne.
5. **Classement (DP-5)** : le bloc ne liste plus TOUTES les chaînes — seulement les plus jouées
   au-dessus du seuil (nombre « pertinent », comme « Par carte »/« Par mode ») ; un joueur dont
   toutes les chaînes sont sous le seuil voit AU MOINS sa chaîne principale (type majoritaire) ;
   ordre stable. Console 0 erreur sur les 4 états. Puis décision de merge (`main` = deploy prod
   auto).
