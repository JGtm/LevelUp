# Journal de decisions — archive 2025-Q1

> Rotation trimestrielle (regle CLAUDE.md).

## [2025-01-22] feat(media): galerie partagée, filtres fonctionnels, likes sociaux, feed-version

**Statut** : Complété

### Décisions techniques

1. **Upload full-width au-dessus de la grille** — `UploadButton` reçoit une prop `fullWidth` qui élargit la zone de dépôt, et est repositionné hors du `PageHeader` dans `MediaPage.tsx`.
2. **Filtres dynamiques via `BuildQ37MediaQuery`** — remplace la constante `Q37MediaFiles` par une fonction avec WHERE/ORDER BY dynamiques. JOIN vers `shared.match_registry` pour `map_name`/`mode_name`.
3. **`queryKey` corrigée** — `useMediaPage` calcule maintenant un `requestHash = JSON.stringify({p, s, k, sec, map, mod, g, lo})` pour que chaque combinaison de filtres soit indépendante dans le cache TanStack.
4. **`MediaFilters` dans `domain`** — évite la dépendance circulaire `port → platform/duckdb`. Le type est défini en `domain.MediaFilters` et utilisé partout.
5. **Likes sociaux via `media_likes` (shared DB)** — nouvelle table `media_likes(media_path, liker_slug, liker_gamertag, liked_at, PK)` dans `shared_matches_v2.duckdb`. `ToggleSharedLike` écrit/supprime la ligne, `GetMediaLikers` retourne les 3 premiers noms + total avec une window function.
6. **`LikersLine` composant** — affiche "Alice, Bob et 3 autres ♥" dans la carte thumbnail et dans la lightbox footer.
7. **`feedVersion` polling** — `GET /media/feed-version` retourne un compteur atomique incrémenté après chaque upload et like. `useFeedVersion` poll toutes les 10 s et invalide `mediaBase(playerSlug)` si la version a changé.
8. **Erreurs `config`/`halo` pré-existantes** — confirmées présentes avant nos modifications (vérifiées via `git stash`). Hors périmètre.

### Résultats observés

- `go build` sur les packages modifiés : 0 erreur nouvelle
- `tsc --noEmit` : 0 erreur dans les fichiers médias

### Conclusion / prochaine étape

Implémentation complète. Commit à effectuer.



**Statut** : Complété

### Décisions techniques

1. **Utiliser `sticky` plutôt que `fixed` global** — la bannière reste fixée sous la L1 à l'intérieur de la zone scrollable de l'app, sans sortir du layout du shell.
2. **Faire passer le contenu au-dessus au scroll** — la colonne de contenu de `HomePage` est montée en `z-10` sur fond `bg-background`, tandis que la bannière reste en `z-0`, ce qui permet à la page de la recouvrir naturellement pendant le défilement.
3. **Augmenter franchement la hauteur du visuel** — l'image passe de `h-24 / sm:h-32 / lg:h-40` à `h-32 / sm:h-44 / lg:h-52`, soit au moins +30% sur chaque breakpoint.

### Résultats observés

- Le bandeau reste accroché sous la L1 pendant le scroll de la home
- Le contenu de la page passe visuellement par-dessus
- La hauteur du header visuel est sensiblement augmentée sur mobile, tablette et desktop

### Conclusion

La bannière d'accueil fonctionne maintenant comme un fond sticky sous la navigation, avec un impact visuel plus marqué.

## [2025-01-20] Page Synthèse — Navigation L1 + blocs D4/D5/D6/D7

**Statut** : Complété

**Décision technique principale** :
La page Synthèse (route `/players/$playerSlug/synthesis`) était inaccessible depuis la navigation L1 (absente de `L1_SECTIONS`). Correction du bug de navigation + implémentation complète des 4 blocs Sprint 55 manquants dans le frontend React.

**Modifications** :
1. `NavL1.tsx` — Ajout de l'entrée Synthèse entre Escouade et Explorer ; correction du path Citations (`/career?tab=citations`)
2. `NavL1.test.tsx` — 3 nouveaux tests (présence, état actif, ordre DOM)
3. `SynthesisHighlightsSection.tsx` — Nouveau composant D5 (meilleur/pire match)
4. `SynthesisPage.tsx` — Refactoring complet :
   - Remplacement `ScopeOverviewBar` → `ScopeBar` (filters_applied/ignored) + `SynthesisOverviewSection` (7 KPI D4)
   - Ajout `SynthesisBreakdownsSection` inline (tables carte/mode D7)
   - Render mis à jour : D4 → KPIs → bipolaire → détaillée → D5 highlights → heatmap → top semaines → D6 relations → D7 breakdowns
   - Correction `MetricRowProps` (type manquant déclaré)
5. `.ai/CHARTS_AND_TABLES.md` — Section §14 ajoutée (8 sous-sections : scope, overview, bipolaire, heatmap, top semaines, D5/D6/D7)

**Résultats observés** :
- `get_errors` sur SynthesisPage.tsx → 0 erreurs TypeScript
- NavL1.test.tsx : 3 nouveaux tests ajoutés
- SynthesisHighlightsSection.tsx créé (~100 lignes)

**Conclusion** : La page Synthèse est maintenant accessible en L1 et expose tous les blocs Sprint 55 (D4-D9 côté Go, D4-D7 côté React). La branche de travail est `feat/synthesis-nav-and-page-blocks`.
