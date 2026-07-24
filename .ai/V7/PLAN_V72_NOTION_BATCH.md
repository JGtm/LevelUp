# PLAN V7.2 — Chantier « Pour la v7.2 » (backlog Notion)

Date de démarrage : 2026-07-24. Branche : `feat/v7.2-notion-batch`.
Mode : supervision multi-agents (Fable pilote et vérifie ; les agents implémentent).
Source de vérité des échanges : page Notion « Backlog LevelUp », section « Pour la v7.2 »
(annotations bleues = Fable, surlignage jaune = réponse utilisateur, barré = traité).
Contrat : skill `plan-execution`. Gates batchés par lot (UI) ; jamais de commandes go
concurrentes (cache) ; intégration `-p 1` ; vérifs visuelles = utilisateur.

## Inventaire des items

| ID | Résumé | Lot | Statut |
|---|---|---|---|
| V72-01 | Générer openapi.yaml par Huma (exploiter Huma à son potentiel) | 5 | [ ] |
| V72-02 | Dependabot : étudier + merger si nécessaire (PRs #65 postcss, #51 actions/checkout) | S | [ ] |
| V72-03 | Stats modes objectifs CTF/Strongholds/KOTH/Oddball (BDD + backfill + UI 4 pages) | 5 | [ ] |
| V72-04 | Tooltips sur en-têtes de tableaux (explication colonnes) | 4 | [ ] |
| V72-05 | Options de sync page admin (Initial single-player, Delta, forcer planificateur) | 4 | [ ] |
| V72-06 | Unifier le nom des armes (voir fichier backlog) | 5 | [ ] |
| V72-07 | Sélecteur de jeu absent du menu déroulant en local | 2 | [ ] |
| V72-08 | Question : c'est quoi le graphe « Net score cumulé » ? | 1 | [x] répondu (frags−morts cumulé) ; renommage FR « Solde frags − morts cumulé » à faire en lot 2 |
| V72-09 | Page Sessions : « Ouvrir sur Waypoint » et Synergies ne marchent pas | 2 | [ ] |
| V72-10 | Dropdown sélection escouade : « surtout… » + playlists/modes/maps en anglais | 2 | [ ] |
| V72-11 | Migrer tri LeaderboardBlock vers SortableTh (dette v7.1) | 4 | [ ] |
| V72-12 | Garde-rail parité capabilities Go <-> TS (dette v7.1) | 4 | [ ] |
| V72-13 | Graphe « XP de carrière (estimée) » sur page Sessions (avant le tableau) | 4 | [ ] |
| V72-14 | Bannière/emblème H5 partagés entre joueurs (doit être par joueur) + couleurs séparées | 3 | [ ] |
| V72-15 | Traiter les items du backlog (hors Tauri et housekeeping) | 5 | [ ] |
| V72-16 | Épaisseur barres graphe « Outils de destruction » réduite (régression) | 2 | [ ] |
| V72-17 | Question : c'est quoi le « t » dans le slug URL ? | 1 | [x] répondu (namespace route « titre », barré dans Notion) |
| V72-18 | Menu Escouade L1 : ajouter « Dynamique » | 2 | [ ] |
| V72-19 | Description de Meganaute mal traduite | 2 | [ ] |
| V72-20 | Notif quand médaille jamais obtenue est décrochée | 4 | [ ] |
| V72-21 | « Écart de frags cumulé » disparu sur Explorer (recherche joueur) | 2 | [ ] |
| V72-22 | Terme « Lobby » dans le graphe Engagement à traduire | 2 | [ ] |
| V72-23 | Légendes collées en bas des blocs (passe générale) | 2 | [ ] |
| V72-24 | Explorer : recherche joueur lente + rechargements bizarres | 3 | [ ] |
| V72-25 | Explorer : graphe « Écart de frags cumulé » sans axe X | 2 | [ ] |
| V72-26 | Sessions : légende OC / DR en anglais (graphe rendement/résistance) | 2 | [ ] |
| V72-27 | Notif : valeur illisible + nom de rang en anglais | 3 | [ ] |
| V72-28 | Carrière : aligner en hauteur blocs Classements et Évolution LUSR/CSR | 2 | [ ] |
| V72-29 | CRITIQUE : fuite cross-titre — nameplate/emblème H5 affichés avec Infinite actif | 3 | [ ] |
| V72-30 | [FINAL] What's new FR/EN + changelogs FR/EN + notes de version in-app | 6 | [ ] |

## Lots (ordre d'exécution)

- **Lot 0 — Recon & diagnostics** (agents lecture seule, parallèles) : A fuite/apparence
  (29+14), B1 i18n/libellés (10,19,22,26,27), B2 layout/charts/régressions
  (4,7,9,13,16,18,21,23,24,25,28), C questions+backlog (8,17,6,15), D1 Huma (1),
  D2 features (5,11,12,20), E stats objectifs (3, agent Plan).
- **Lot 1 — Réponses & décisions Notion** : réponses aux questions, recos posées en
  bleu + champs jaunes pour l'utilisateur (3 lobby/équipe, 2 feu vert deploy, 5 sémantique).
- **Lot 2 — Fixes UI/i18n** (agents impl, fichiers partitionnés, i18n.ts sérialisé).
- **Lot 3 — Bugs profonds** : 29 (prioritaire), 14, 24, 27.
- **Lot 4 — Features moyennes** : 4, 5, 11, 12, 13, 20.
- **Lot 5 — Gros chantiers** : 1 (Huma), 3 (objectifs — architecture avant backfill),
  6 (armes), 15 (backlog).
- **Gates batchés** après chaque lot d'implémentation : tsc -b (cache purgé), eslint
  fichiers touchés, vitest zones touchées, go build/vet/test séquentiels,
  -tags=integration -p 1 si sync/persist touchés.
- **Lot 6 — [FINAL] V72-30** : changelogs + What's new + notes in-app, en dernier.
- **S — Superviseur** : Dependabot (2), git, commits, Notion, mémoire.

## Décisions

- 2026-07-24 : merge Dependabot = push main = deploy prod → feu vert utilisateur demandé
  dans Notion avant merge.

## Découvertes (hors périmètre — ne pas traiter)

(vide)

## Journal

- 2026-07-24 : démarrage. Branche créée, légende posée dans Notion, vague 0 lancée
  (7 agents recon lecture seule). PRs Dependabot identifiées : #65 (postcss 8.5.16→8.5.23,
  dev-dep), #51 (actions/checkout 6.0.2→6.0.3).
