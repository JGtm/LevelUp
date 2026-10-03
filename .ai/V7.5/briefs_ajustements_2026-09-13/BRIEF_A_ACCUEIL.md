# LOT A — Accueil + Médias

Worktree : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-ajust-accueil` · branche `feat/ajust-accueil` · port Vite **5181**.
Lis d'abord `BRIEF_COMMUN.md` (même dossier).

## Repérage déjà fait (vérifie sur pièces, le code a pu bouger)

- Accueil : `features/home/HomePage.tsx:566` monte `RecentMediaRail` (`features/home/RecentMediaRail.tsx:106-125`, Carousel de `MediaThumbnailCard`). Médias : `features/media/MediaPage.tsx:395-416`, même `MediaThumbnailCard` en grille.
- Vignette : `features/media/MediaViewer.tsx:164-353`. Cascade image : `:220-255` (clip + thumbnail_path → `GifHoverThumbnail` ; sinon img thumbnail_path ; sinon img file_path pour screenshot ; sinon `<video preload=metadata>`).
- `GifHoverThumbnail` : `components/ui/gif-hover-thumbnail.tsx:20-113` — canvas `opacity-0` tant que `fetch(src)` + `createImageBitmap` n'ont pas abouti ; repli `<img>` seulement si `posterFailed`. Suspect n°1 pour des tuiles vides et silencieuses.
- Compteur « j'aime » : `MediaLikeButton` `MediaViewer.tsx:100-136` (overlay compact `:287-293`) ; ligne des noms `LikersLine` `:56-70`, montée `:349` en pied de carte ; format en dur « Alice, Bob et 3 autres ♥ » (non i18n). Lightbox : `features/media/CoverFlowModal.tsx:645`.
- Tooltip disponible : `components/ui/tooltip.tsx:20-70` (`<Tooltip content wide?>`, portal fixed → s'échappe d'un `overflow-hidden`).

## Items

### A.1 — Les miniatures des médias ne s'affichent plus sur l'accueil
1. REPRODUIS d'abord : capture AVANT de l'accueil de JGtm (`/…/players/JGtm` — trouve le chemin exact dans `e2e/slice-5-home.spec.ts`) et de la page Médias. Regarde les tuiles : cadres vides ? `EmptyStateNotice` ? Note-le.
2. Diagnostique sur pièces ET en exécution : avec Playwright, intercepte les réponses réseau des tuiles (`page.on('response')`) pour voir si `thumbnail_path` est servi (200 ? 404 ? 401 ?) et regarde la console. Compare Accueil vs Médias (mêmes items ? même URL ?). Le rapport doit nommer LA cause prouvée (pas une liste de suspects).
3. Corrige la cause. Si c'est `GifHoverThumbnail`, la règle est : une vignette montre TOUJOURS quelque chose immédiatement (l'image statique `thumbnail_path`), l'animation au survol est un plus. Si c'est une donnée absente côté API (thumbnail_path null), dis-le avec la requête qui le prouve et applique le repli visible (image du `file_path` / première frame) — pas de tuile vide.
4. Capture APRÈS : les miniatures visibles sur Accueil ET Médias.

### A.2 — Mentions « j'aime » en infobulle
1. Le compteur cœur + nombre reste tel quel. La ligne `LikersLine` en pied de carte disparaît (Accueil ET Médias, puisque même composant ; vérifie aussi la lightbox `CoverFlowModal.tsx:645` et applique la même règle : infobulle au survol du cœur).
2. Au survol (et au focus clavier) du cœur-compteur : `<Tooltip>` de `components/ui/tooltip.tsx` avec le contenu « Aimé par Alice, Bob et 3 autres » (FR) / « Liked by … » (EN) — i18n dans `features/media/i18n.ts` (ou le dictionnaire de la feature, regarde comment elle fait), pluriel FR/EN corrects, 0 like → pas d'infobulle. `aria-label` du bouton i18n aussi (aujourd'hui FR en dur `MediaViewer.tsx:125`).
3. Le bouton reste cliquable (toggle like) sans que l'infobulle gêne ; l'infobulle ne doit pas être coupée par l'`overflow-hidden` de la tuile (le Tooltip est en portal, vérifie sur la capture en survolant : Playwright `page.hover()` puis capture).
4. Supprime le code mort (`LikersLine`, ses tests) ; adapte les tests existants (`MediaViewer*.test.tsx`, `queries.likeToast.test.ts` s'ils asservissent le texte).

Captures APRÈS : accueil (rail médias), page Médias (grille), une tuile survolée avec l'infobulle visible.
