# PLAN — « Prises nettes » de drapeau (jonglage replié) — 2026-09-13

> Contexte : le portage de « Les formes retenues » (lot D2 des ajustements pré-v7.5) a laissé
> « Drapeaux saisis » hors du rôle « prendre », parce que le compteur officiel `flag_grabs`
> compte chaque ramassage, jonglage compris (lancer-reprise pour avancer plus vite). Décision
> utilisateur (13/09) : « OK pour les prises nettes avec ta reco ».
>
> Contrat : skill `plan-execution`. Branche : `feat/prises-nettes` depuis `feat/v75`, worktree
> dédié. Effort : étape 0 rapide (diagnostic) ; étapes 1-2 moyennes.

## 1. Définition (tranchée)

- **Prise nette** : une prise de drapeau lue dans le film (événement nommé `StatFlagGrabs`,
  `analysis/objectiveevents`) dont le portage PRÉCÉDENT du même drapeau n'était pas porté par
  le MÊME joueur, ou l'était mais s'est terminé il y a plus de `fenetre_jonglage_s` secondes
  (lâcher daté par la renaissance de l'objet aux pieds du porteur, lot 6.7-B1, `flag_carries.go`).
  Un portage jonglé compte UNE fois.
- La fenêtre vit en DONNÉE du titre (`config/titles/halo_infinite/mappings/regulation.toml`,
  clé à créer, valeur posée APRÈS la mesure de l'étape 0), jamais en dur.
- Le compteur brut de l'API (`flag_grabs`) n'entre jamais dans les rôles ; il reste affiché
  là où il l'est déjà, tel quel.
- Un match sans film décodé = « non mesuré » sur cette grandeur (jamais 0), comme le reste des
  formes retenues.

## 2. Étapes

### Étape 0 — Mesure (diagnostic, aucun code de production)
- [x] 0.1 Test de recherche jetable (`*_research_test.go`, précédent lot 6.3) sur les
  artefacts CTF locaux (`data/cache/replays/halo_infinite/*.json`, verdict `flagfilm`) : pour
  chaque joueur et chaque match, prises brutes (événements), prises nettes pour plusieurs
  fenêtres (1, 2, 3, 5 s), part de jonglage ; distribution des délais lâcher → reprise par le
  même joueur (histogramme) pour choisir la fenêtre sur une COUPURE visible, pas un chiffre rond.
- [x] 0.2 Contrôle : prises nettes ≤ prises brutes partout ; sur un match sans lâcher volontaire
  daté, nettes = brutes (sinon la règle fuit).
- [x] 0.3 Rapport `.ai/V7.5/RAPPORT_PRISES_NETTES_<date>.md` : chiffres bruts, fenêtre retenue et
  pourquoi, verdict « l'écart justifie la grandeur nette » ou « le brut suffit ».
- Gate : rapport rendu (`.ai/V7.5/RAPPORT_PRISES_NETTES_2026-09-13.md`). RÉSULTAT : 13 films CTF, 617 prises brutes → 370 nettes à 1,5 s (40 % de jonglage) ; coupure nette entre 1,4 et 1,6 s (99 couples sur [1,0;1,5) contre 27 sur [1,5;2,0)) ; le classement s'inverse sur le témoin 7fce3219. **Fenêtre retenue : 1,5 s.** Verdict : la grandeur nette est justifiée. Contrôle « film sans lâcher daté » non instruisible (aucun sujet) → substitut repliées ≤ closedByObject OK 13/13. Comparaison au compteur API non faite (base tenue par le serveur).

### Étape 1 — Go
- [x] 1.1 `internal/analysis/objectiveevents/` (ou paquet frère pur) : `NetFlagGrabs(evs, carries,
  window)` testé sur fixtures (jonglage, passe de main, reprise après retour, film tronqué).
- [x] 1.2 Persistance : la grandeur par joueur et par match rejoint les stats d'objectif issues
  du film (INSERT-only via `persist`, table append-only + vue `_latest`, recette ADR 0026) ; le
  bloc `formes_retenues` (Escouade) et les objectifs de Sessions la lisent depuis `_latest`.
- [x] 1.3 Table rôle → grandeurs (`analysis/narrative`) : « prises nettes » entre dans « prendre ».
- Gate : go build/vet/test, tests anti-ART (`-tags=integration`), openapi + types.

### Étape 2 — Web
- [x] 2.1 Cartes d'objectif des formes retenues et bloc Sessions : colonne « Prises nettes »
  (FR) / « Net grabs » (EN), infobulle « jonglage replié (fenêtre N s) », « non mesuré » sans film.
- [ ] 2.2 Rattrapage FAIT le 14/09 00:01 (serveur arrêté) : `--dry-run` puis réel — 13 matchs écrits, 1 891 sans artefact, 63 sans calque de drapeau, 0 échec, 106 lignes joueur, 617 brutes → 370 nettes (40,0 %), identique à la mesure de l'étape 0 ; témoin 7fce3219 : brut 85 → net 28. Serveur relancé. Captures (passe 4) en cours — sur un match CTF réel (témoin `7fce3219`) — EN ATTENTE du backfill `levelup backfill-flag-grabs-net` (serveur arrêté, base partagée en écriture), à faire par le superviseur après la revue adversariale et la passe visuelle en cours.
- Gate : tsc, eslint, vitest, capture.

### Étape 3 — Clôture
- [ ] thought_log, référence équipement/objectifs amendée, plan statué, CI verte, fusion.

## 2 bis. Revue adversariale avant fusion (2026-09-13)

Relecteur 1 (L1 écritures + L4 données) : 9 constats recevables, tous renvoyés en correction — P1 : (1) table absente des ratchets `append_only_state_guard_test.go` et `no_raw_rating_reads_test.go` ; (3) web `aggregateRole`/`aggregateColumns`/`roleLobbyParts` transforment une clé absente en 0 et font entrer `flag_grabs_net` dans la somme de « prendre » ; (4) part > 100 % (numérateur tous matchs, dénominateur camp connu) ; (6) dénominateur de couverture Sessions toutes familles + phrase qui accuse le film ; (7) `openings` calculé puis jeté, « compteur officiel » = brut du film. P2 : (2) vue `_latest` par clé, pas par passe (motif `decode_pass` des sœurs) ; (5) présence de clé `teamOf` ; (8) joueur mesuré à 0 prise = « non mesuré » ; (9) référence de fichier de test fausse. Relecteur 2 (L2+L3+L6) : multi-titre, chemins, libellés, seuils, ART, frontière SQL/affichage TIENNENT ; mutations M1-M5 rougissent ; M6 (porte de capability neutralisée) et M7 (lecture produit no-op) restent VERTES → C1 (P1) aucun test sur la porte ni sur les deux chemins de lecture ; C2 (P2) machinerie `Openings`/`evs` morte (recoupe le constat 7) ; C3 (P2) doc inversée du `--dry-run` (rien par match) ; C4 (P2) 4e copie de la porte de capability et 3e copie de `echec*` sans helper ni garde-rail. Les 13 constats CORRIGÉS (`8f8a862b6`, `43b0e2fb0`, `1a2a61a65`, `3d2149a65`), chacun verrouillé par un test ; mutations M6/M7 rejouées et rouges ; décision constat 3 : les agrégats de RÔLE excluent les grandeurs optionnelles (deux couvertures ne se somment pas), la grandeur seule ne porte que les matchs qui la mesurent ; constat 8 : roster complété à zéro (humains), rien écrit si aucun portage nommé ; constat 7 : `openings` persisté et publié. Contrôle superviseur sur pièces (ratchets, `decode_pass`, `optional` dans les agrégats) puis fusion `90b79a96d`, poussée.

## 3. Journal
- 2026-09-13 — Plan écrit ; étape 0 exécutée et fusionnée (`e4ea1e1c6`) ; étapes 1-2 livrées sur `feat/prises-nettes` (`25f8c0985`, `af4fd07c3`, `c83758ad0`, `ba8121daa`) : `NetFlagGrabs` (11 tests, 1,4 s replié / 1,6 s compté), clé `flag_juggle_window_s = 1.5` (regulation.toml schéma 7, Halo 5 sans clé = rien publié), table `match_flag_grabs_net` append-only + `_latest` (3 colonnes : brutes, nettes, fenêtre), persister INSERT-only protégé par `no_art_patterns_test.go`, projection au fil de l'eau sous `film.flag_grabs_net`, CLI de rattrapage, rôle « prendre » (`ObjectiveRoleGrandeurs` ≠ `ObjectiveRoleColumns`), web Escouade + Sessions. Gates : go test complet + intégration persist + golangci 0 + openapi/types à jour + tsc + vitest 704/7 443. Revue adversariale (2 relecteurs L1+L4 / L2+L3+L6) lancée avant fusion.

