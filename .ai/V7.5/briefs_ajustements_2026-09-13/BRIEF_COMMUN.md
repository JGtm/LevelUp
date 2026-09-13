# BRIEF COMMUN — Ajustements pré-v7.5 (2026-09-13)

Tu es un exécuteur d'un lot UI/produit. Le superviseur intègre, tu n'intègres pas.

## Contexte humain — à lire en premier

L'utilisateur a réexpliqué ces demandes de nombreuses fois. Ses mots : « j'en ai marre de
l'improvisation, hallucination et devinettes ». Conséquences NON NÉGOCIABLES :

1. **Zéro improvisation.** Quand une maquette (artefact HTML) est citée, tu la reproduis
   dans le FOND (données, dénominateurs, libellés, notes) et dans la FORME (disposition,
   formes graphiques, légendes, axes, infobulles). Tu ne « t'inspires » pas, tu portes.
2. **Zéro devinette.** Tu vérifies sur pièces (le fichier, la ligne) avant d'écrire, et tu
   vérifies le RENDU après (capture d'écran lue par toi-même — voir §Vérification visuelle).
   Une hypothèse non vérifiée ne se rapporte pas comme un fait.
3. **Zéro report.** Si une action est faisable dans ta session, tu la fais. Un « à faire
   ensuite » = lot non terminé. Les seuls reports valides : décision qui n'appartient qu'à
   l'utilisateur, donnée réellement absente (prouvée), dépendance externe. Tu les listes
   dans ton rapport avec la preuve.
4. **Zéro hors-périmètre.** Tu notes les découvertes dans ton rapport, tu ne les corriges
   pas. Exception : ce qui bloque ton gate.

## Environnement

- Ton worktree : indiqué dans ton brief. Tu ne touches JAMAIS un autre dossier
  (`LevelUp-go-migration`, autres `LevelUp-wt-*`). Tous tes chemins sont dans ton worktree.
- `node_modules` est une JONCTION vers le principal (ne pas `npm install`, ne pas `npm ci`).
- Go (si ton lot touche l'API) : `GOCACHE` privé — exporte `GOCACHE=<worktree>/.gocache-<lot>`
  avant toute commande go (chemin absolu). CGO : gcc msys64 ucrt64 déjà dans le PATH
  (si `go build` échoue sur le lien, essaie depuis PowerShell natif).
- Un serveur API tourne sur `http://127.0.0.1:8000` (branche feat/v75, données réelles du
  poste, joueur peuplé `JGtm`, coéquipiers `Madina97294`, `Chocoboflor`). Tu ne l'arrêtes
  pas, tu ne lances PAS de second serveur Go sur les mêmes bases (modèle mono-process
  DuckDB : interdit). Si ton lot ajoute des champs/endpoints Go, tu les vérifies par tests Go
  et, côté web, par tests vitest + un stub Playwright de la route ; le superviseur fera la
  passe visuelle sur données réelles après intégration.
- Vite : lance TON instance depuis ton worktree : `cd apps/web && npx vite --port <PORT>
  --strictPort` (le port est dans ton brief), en ARRIÈRE-PLAN via `run_in_background` du
  Bash tool, puis vérifie qu'il répond (`curl -s -o /dev/null -w "%{http_code}"
  http://localhost:<PORT>/`). Le proxy `/api` pointe déjà sur :8000. Arrête-le en fin de lot.
- INTERDIT : les outils MCP `chrome-devtools` (ils lancent un Chrome étranger). Le rendu se
  vérifie avec Playwright headless (ci-dessous).
- INTERDIT : tout run en arrière-plan pour un résultat dont ton rapport dépend (gates en
  avant-plan). Seul Vite tourne en arrière-plan.
- Jamais `git stash`, jamais `git add -A` : tu stages tes fichiers NOMMÉMENT. `git checkout
  -- apps/web/src/routeTree.gen.ts` avant de stager (le plugin le régénère). Tu commits sur
  ta branche (`git branch --show-current` pour vérifier), tu ne pushes pas, tu ne merges pas.
  Tu n'écris pas dans `.ai/` (ni thought_log, ni plan) : ton rapport final remplace.

## Skills à invoquer (Skill tool) AVANT de coder

`plan-execution` (contrat), `frontend-patterns`, `color-tokens` ; `foundations-usage` si tu
crées une page/carte/chart ; `dataviz` si tu crées ou modifies un graphe ; `arch-rules`,
`go-features`, `db-schema` si tu touches au Go. Tu suis leurs règles.

## Règles du dépôt (rappel — CLAUDE.md fait foi)

- Toute string UI en FR **et** EN (`Record<Locale, T>` ; manifestes `lib/i18n/manifests/*.toml`
  si la feature les utilise — regarde comment la feature fait). FR sans anglicisme.
- Aucune couleur hex ni classe Tailwind couleur dans `features/`/`components/` : jetons
  sémantiques via `tokenCssVar` / `resolveToken` / `getSeriesColors`.
- Graphes : wrappers ECharts de `components/charts/` (README) ; `ChartCard`, `SectionCard`,
  `KPIStrip` pour les gabarits. Tableaux interactifs = TanStack Table. Query keys dans
  `lib/query/keys.ts`.
- Fichier ≤ 500 L, fonction ≤ 80 L, ≤ 5 paramètres. Pas d'emoji dans les fichiers.
- 0 code mort : ce que tu débranches, tu le supprimes avec ses tests, ses i18n et ses imports.
- Title-agnostic : jamais `slug === '...'` ; capabilities.
- Chaque texte que tu supprimes : supprime aussi ses clés i18n FR/EN et ses tests.

## Gates (avant-plan, dans ton worktree)

```bash
cd apps/web && npx tsc -b --force            # 0 erreur
cd apps/web && npx eslint <fichiers touchés>  # 0 erreur, pas de nouvel avertissement
cd apps/web && npx vitest run <dossiers touchés>   # vert ; puis, en clôture :
cd apps/web && npx vitest run                # suite complète verte (≈ 2-4 min)
```
Go (si touché) : `cd apps/go-api && go build ./... && go vet ./internal/... && go test
./internal/<paquets touchés>/...`. Baseline lint gelée : tu n'ajoutes aucun avertissement
golangci (`make go-api-lint` si dispo).

Hors sandbox pour vitest si le sandbox l'empêche (mémoire du dépôt : « vitest hors sandbox »).

## Vérification visuelle — OBLIGATOIRE pour tout item de rendu

Un harnais existe : `apps/web/e2e/_helpers/visual.ts` (`prepareVisualPage`, attente de
stabilité canvas) et `apps/web/e2e/visual/app-pages.visual.spec.ts` (exemples de pages).
Écris un spec TEMPORAIRE `apps/web/e2e/visual/_tmp-<lot>.visual.spec.ts` qui :
- ouvre chaque page/onglet cible sur `http://localhost:<PORT>` avec `E2E_VISUAL_PLAYER=JGtm`
  (et `E2E_VISUAL_SQUAD_TEAMMATES=Madina97294,Chocoboflor` pour Escouade), attend la
  stabilité, puis `page.screenshot({ path: '<scratch>/<lot>-<page>.png', fullPage: true })`
  (viewport 1440 px de large ; regarde aussi une largeur 1024 si la disposition est en jeu).
- Commande : `cd apps/web && E2E_BASE_URL=http://localhost:<PORT> E2E_VISUAL_PLAYER=JGtm
  npx playwright test e2e/visual/_tmp-<lot>.visual.spec.ts --project=visual`.
- Puis tu LIS chaque PNG avec l'outil Read (il affiche l'image) et tu vérifies, item par
  item du brief, que le rendu est celui demandé : libellés, disposition, légende, axes,
  couleurs distinguables, absence des textes à retirer. Tu notes dans ton rapport ce que tu
  as VU (pas ce que tu as codé). Si un rendu n'est pas conforme, tu corriges et tu recaptures.
- Capture AVANT (état initial) et APRÈS pour chaque page touchée : les deux jeux de PNG
  restent dans le scratchpad, chemins listés dans ton rapport.
- Supprime le spec temporaire avant ton commit final.
Scratchpad : `C:\Users\GUILLA~1\AppData\Local\Temp\claude\c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration\346b3f3e-40bc-4f2c-9027-587c8b85c85a\scratchpad\<lot>\` (crée le dossier).

## Rapport final (structure imposée)

1. Items du brief, un par un, statut `[x]` / `[~]` (référence) / `[!]` (preuve du blocage).
2. Ce que tu as VU sur les captures APRÈS (par page, phrases courtes, chemins des PNG).
3. Gates : commandes exactes et résultat (chiffres).
4. Commits (hash + message) sur ta branche.
5. Découvertes hors périmètre (non traitées).
6. Décisions que tu as dû prendre faute de précision dans le brief (avec la règle appliquée).
Pas de jargon interne dans le rapport : l'utilisateur pense produit.
