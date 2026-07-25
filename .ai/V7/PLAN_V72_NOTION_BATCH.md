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
| V72-01 | Générer openapi.yaml par Huma (exploiter Huma à son potentiel) | 5 | [x] clos H0-H8 (openapi genere, golden, gardes) |
| V72-02 | Dependabot : étudier + merger si nécessaire (PRs #65 postcss, #51 actions/checkout) | S | [x] #51 fermee, #65 mergee (deploy prod passe) |
| V72-03 | Stats modes objectifs CTF/Strongholds/KOTH/Oddball (BDD + backfill + UI 4 pages) | 5 | [x] P0-P4 + backfill local (5656 lignes) ; backfill PROD post-deploy |
| V72-04 | Tooltips sur en-têtes de tableaux (explication colonnes) | 4 | [x] 2 familles + contenu FR/EN (ADR 0006) |
| V72-05 | Options de sync page admin (Initial single-player, Delta, forcer planificateur) | 4 | [x] initiale un joueur + legende des portees |
| V72-06 | Unifier le nom des armes (voir fichier backlog) | 5 | [x] weapon_names.toml par titre + 2 garde-rails (+ SPNKr a combustible) |
| V72-07 | Sélecteur de jeu absent du menu déroulant en local | 2 | [x] process local perime — rebuild/restart, 2 titres verifies |
| V72-08 | Question : c'est quoi le graphe « Net score cumulé » ? | 1 | [x] repondu + renomme Solde frags - morts cumule |
| V72-09 | Page Sessions : « Ouvrir sur Waypoint » et Synergies ne marchent pas | 2 | [x] synergies ajoutees ; waypoint = non-bug verifie DOM + tests |
| V72-10 | Dropdown sélection escouade : « surtout… » + playlists/modes/maps en anglais | 2 | [x] largeur menu + modes FR serveur + playlists FR par id |
| V72-11 | Migrer tri LeaderboardBlock vers SortableTh (dette v7.1) | 4 | [x] SortableTh partage, exemption retiree |
| V72-12 | Garde-rail parité capabilities Go <-> TS (dette v7.1) | 4 | [x] garde-rail AST bidirectionnel (weapon_kills allowliste) |
| V72-13 | Graphe « XP de carrière (estimée) » sur page Sessions (avant le tableau) | 4 | [x] champ + capability + chart cable avant le tableau |
| V72-14 | Bannière/emblème H5 partagés entre joueurs (doit être par joueur) + couleurs séparées | 3 | [x] store par (titre, joueur) + couleurs separees (migration reset) |
| V72-15 | Traiter les items du backlog (hors Tauri et housekeeping) | 5 | [x] 4 items traites ; 2 laisses backlog (acte user) |
| V72-16 | Épaisseur barres graphe « Outils de destruction » réduite (régression) | 2 | [x] legende hors canvas, epaisseur rendue |
| V72-17 | Question : c'est quoi le « t » dans le slug URL ? | 1 | [x] repondu (namespace titre) |
| V72-18 | Menu Escouade L1 : ajouter « Dynamique » | 2 | [x] entree menu L1 + i18n |
| V72-19 | Description de Meganaute mal traduite | 2 | [x] override migration (loc Waypoint defectueuse) |
| V72-20 | Notif quand médaille jamais obtenue est décrochée | 4 | [x] medal_first_earned + anti-rafale + noms FR/EN |
| V72-21 | « Écart de frags cumulé » disparu sur Explorer (recherche joueur) | 2 | [x] conditionnel matchs communs + etat vide explicite (resolu user) |
| V72-22 | Terme « Lobby » dans le graphe Engagement à traduire | 2 | [x] Partie / Part partie (+ accent Part equipe) |
| V72-23 | Légendes collées en bas des blocs (passe générale) | 2 | [x] ChartCard.legend + migration blocs informatifs (interactifs documentes) |
| V72-24 | Explorer : recherche joueur lente + rechargements bizarres | 3 | [x] replace URL + pool tokens (A1 valide Madina) + typeahead live=0 |
| V72-25 | Explorer : graphe « Écart de frags cumulé » sans axe X | 2 | [x] axe X Explorer (Relations inchange) |
| V72-26 | Sessions : légende OC / DR en anglais (graphe rendement/résistance) | 2 | [x] Rendement / Resistance |
| V72-27 | Notif : valeur illisible + nom de rang en anglais | 3 | [x] rang FR par id + arrondi gap (+ current_mu/next_tier_mu defensif) |
| V72-28 | Carrière : aligner en hauteur blocs Classements et Évolution LUSR/CSR | 2 | [x] h-full + fluid |
| V72-29 | CRITIQUE : fuite cross-titre — nameplate/emblème H5 affichés avec Infinite actif | 3 | [x] fail-closed + cache titre + echo/rejet + sequencement + blindage cles/gardes |
| V72-30 | [FINAL] What's new FR/EN + changelogs FR/EN + notes de version in-app | 6 | [x] What's new + changelogs FR/EN + notes in-app (agent, en finalisation) |
| V72-31 | Alertes Discord : anti-rafale disque + notif rétablissement + releases manquées + notifs sync absentes | 3 | [x] anti-rafale persistant (sans rappel 24h, acte), version bakee, notif cycle ; VPS pre-seme v7.1.0 |
| V72-32 | Notes de performance absentes (BTB placement) : état « En placement » au lieu du vide | 2 | [x] badge En placement (tableaux + tuiles home) ; cause reelle = seuil chaine, pas un bug |
| V72-33 | Catégories de médailles Halo 5 selon le wiki halo.fr | 2 | [x] 11 categories (Bases, Zone de combat) + 3 fantomes masques |
| V72-34 | Question plan Explorer-live : répondu (A1 fait+validé) ; reste A3 badges live_status + éval A2 | 1 | [x] repondu ; A1 fait, A3 badges live faits, A2 saisons = plan Explorer-live ([!] sourcing CMS) |

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

## Cloture (2026-07-25)

Tous les items statues [x] (details ci-dessus). Contre-revue ultracode passee
(5 majeurs corriges, 23 mineurs traites ou justifies). Gates complets verts.
Reste HORS branche : merge main (= deploy prod, decision utilisateur) puis
orchestration des backfills VPS (objectifs + citations, coupure autorisee) et
verification de la notif Discord de release v7.2.0 (test grandeur nature).
