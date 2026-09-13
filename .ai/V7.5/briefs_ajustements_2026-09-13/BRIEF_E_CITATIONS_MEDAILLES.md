# LOT E — Citations + Médailles : disposition des blocs

Worktree : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-ajust-citations-medailles` · branche `feat/ajust-citations-medailles` · port Vite **5186**.
Lis d'abord `BRIEF_COMMUN.md`. Lot 100 % web, petit mais le rendu est le juge.

## Repérage déjà fait (vérifie sur pièces)
- Moteur partagé : `lib/layout/blockRowPacking.ts:30-92` — `BLOCK_GRID_COLUMNS = 6` ; `blockSpan()` `:42-45` (≤3 items → 2 ; 4-8 → 3 ; ≥9 → 6) ; `packBlockRows()` `:59-82` glouton sans réordonnancement ; `rowGridTemplate()` `:88-92` ajoute une PISTE FANTÔME `${rowRemainder}fr` sans enfant → un bloc seul de 2 items = `2fr 4fr` = un tiers de largeur ; « On ne dilate PAS les blocs pour combler » (`:23-26`) — c'est exactement ce que l'utilisateur refuse.
- CSS : `styles/globals.css:344-359` `.block-row { display:grid; gap:1.5rem; align-items:start }` + `grid-template-columns: var(--block-row-cols)` au-dessus de 768 px.
- Citations : `features/citations/CitationsView.tsx:55-79`. Médailles : `features/medals/MedalsView.tsx:67-88` (packing PAR super-section `:72`), `MedalCategoryCard :90-118` (pas de `h-full`).
- Tests : `lib/layout/blockRowPacking.test.ts`, `features/medals/MedalsView.test.tsx`.
- Commit d'origine du packing : `9212a1b0e`.

## Cadrage utilisateur (verbatim)
Citations : « des blocs comme “Ennemis” qui est seul sur sa rangée, ce qui est acceptable vu que c'est le seul “petit” bloc mais il ne prend pas toute la largeur ». Médailles : « des blocs côte à côte qui ne sont pas alignés sur la hauteur, des rangées de deux qui prennent pas la largeur, des blocs tous seuls qui prennent un tiers de la rangée. Du travail d'amateur ».

## Items
### E.1 — Une rangée occupe toujours toute la largeur
Supprime la piste fantôme (`rowRemainder` dans `rowGridTemplate`) : les pistes `fr` des blocs présents se dilatent pour remplir la rangée (un bloc seul = 100 %, deux blocs = proportion de leurs spans). Mets à jour le commentaire de doctrine (`:23-26`) — il décrit l'ancien défaut (anti-pattern « doc inversée »). Garde le packing (petits blocs regroupés). Tests `blockRowPacking.test.ts` mis à jour (cas : bloc seul → `2fr` seul ; deux blocs 2+3 → `2fr 3fr`). Regarde aussi si `rowRemainder` est encore lu ailleurs ; si non, supprime-le du type.
### E.2 — Blocs d'une même rangée alignés en hauteur
`.block-row { align-items: stretch }` + les cartes de bloc (`MedalCategoryCard`, la carte de catégorie de `CitationsView`) en `h-full flex flex-col` pour que la bordure/le fond aillent jusqu'en bas ; le contenu (vignettes en flex-wrap) reste en haut de la carte (pas centré verticalement) sauf si le rendu est meilleur centré — regarde la capture et dis ce que tu as choisi.
### E.3 — Vérification
Captures AVANT/APRÈS de Citations et Médailles (page entière, 1440 et 1024). Sur APRÈS, vérifie : aucune rangée avec du vide à droite ; blocs d'une rangée de même hauteur ; un bloc seul prend toute la largeur ; sous 768 px, une colonne. Lance aussi les specs e2e existants qui touchent ces pages (`e2e/slice-2b-citations.spec.ts` et un éventuel spec médailles) sur ton port.
