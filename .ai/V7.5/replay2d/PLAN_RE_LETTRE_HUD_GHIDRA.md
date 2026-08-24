# PLAN — RE : la règle d'attribution de la lettre A/B/C d'une base (Ghidra, statique)

Date : 2026-08-24. Origine : backlog Notion REPLAY 2D item 2 — la lettre n'existe dans
aucune donnée décodée du film ; l'utilisateur a relancé Ghidra (instance MCP disponible,
PID vérifié) et arbitré la route « les deux en parallèle » : un lot fallback publie des
lettres par l'ordre des slots ti=13 pendant que CE lot cherche la règle VRAIE dans le
binaire. Si ce lot contredit le fallback, le superviseur corrige le fallback avant fusion.

Branche : `wt/re-lettre-hud` (worktree dédié, base feat/v75 `b16ba17e5`). Exécution sous
le contrat du skill `plan-execution`. Ce plan est commité PAR le lot (premier commit).

## Objectif et critère de succès

Établir COMMENT le jeu choisit la lettre affichée sur une base (Strongholds / Total
Control) : d'où vient l'index (ordre d'une liste d'objets de mode ? propriété réseau ?
rang défini dans la variante ?) et à quel ordre OBSERVABLE du film il correspond (slots
ti=13, ordre de création ti=11, ordre du fichier de variante). Trois issues, toutes des
succès : (a) règle trouvée + correspondance film établie ; (b) règle trouvée mais sans
correspondant observable dans le film (le fallback reste « nos lettres ») ; (c) négatif
écrit (introuvable à coût raisonnable, avec ce qui a été fouillé).

## Faits établis (ne pas re-mesurer)

- Le film ne porte la lettre nulle part en clair : ti=47 i2 réfuté le 24/08 (valeur par
  joueur à 20 Hz) ; tacmap ti=30/34 = campagne (négatif mesuré) ; le composant texte du
  marqueur d'objectif `managed-navpoint` (ti=12) n'est PAS porté (seul i0 lu, 27
  composants non portés) — c'est un canal candidat si la règle passe par le navpoint.
- ti=13 : 3 slots / 3 ids de nommage identiques sur les 2 matchs Bastion (ordre stable).
- Les recherches EXE précédentes ont montré que les chaînes se trouvent (« vip » : 30
  occurrences) et que le pas de structure 0x3810 se retrouve (18 sites) — le binaire est
  fouillable. `ApplyVIPPlayerFX` n'existe pas : les mécaniques de mode vivent en partie
  en HavokScript, ce qui peut aussi être le cas de la lettre (issue (c) possible).

## Hors périmètre (fermé)

- Toute modification de code de production (ce lot produit un RAPPORT + au plus des
  instruments de recherche gatés par env var, patron TI47_FILM).
- Cheat Engine / dynamique (le jeu n'est pas supposé tourner) : STATIQUE Ghidra seulement.
  Si la réponse exige du dynamique, c'est une issue (c) avec la marche à suivre proposée.
- Le rendu des lettres (lot fallback séparé).

## Phase 0 — Connexion et points d'entrée

- [ ] 0.1 `mcp__ghidra__list_instances` puis connexion (`connect_instance`). Si aucune
      instance ou binaire non chargé dans le projet : STOP immédiat, CR court (c'est une
      action utilisateur, pas la tienne). Vérifier quel programme est ouvert.
- [ ] 0.2 Inventaire des points d'entrée, chacun consigné avec adresses :
      chaînes de localisation/désignation des zones (motifs candidats : « zone », «
      stronghold », « objective », clés de type hud/nav/waypoint, formats « %s ») ;
      références au navpoint/marqueur (symboles ou RTTI contenant navpoint/waypoint) ;
      la table/structure des objets de mode (managed-objective, pas 0x3810 connu).
- [ ] 0.3 Choisir les 2-3 pistes les plus prometteuses et le dire au CR intermédiaire
      (une ligne dans le rapport, pas d'attente d'accord).

Gate 0 : connexion établie + liste d'adresses concrètes (pas « à chercher »).

## Phase 1 — Remonter la source de l'index de lettre

- [ ] 1.1 Depuis les points d'entrée, remonter les xrefs jusqu'à l'endroit où un index
      0/1/2 (ou 'A'+n) est choisi pour une zone : décompiler (`analyze_function`),
      identifier la donnée source (champ de l'objet de mode ? position dans une liste ?
      valeur répliquée ?).
- [ ] 1.2 Dire si cette source est OBSERVABLE dans le film : si c'est un rang de liste,
      lequel (ordre de création ti=11 ? ordre du bloc de variante ?) ; si c'est une
      propriété répliquée, laquelle (un slot ti=13 ? un composant ti=12 non porté ?).
- [ ] 1.3 Si un canal film est désigné : le confronter sur le corpus local (instrument
      gaté env var, lecture seule, plafonné en mémoire — bombe RAM consignée) aux 2
      matchs Bastion : la permutation prédite reproduit-elle l'ordre ti=13 du fallback ?

Gate 1 : verdict (a)/(b)/(c) écrit, chaque affirmation adossée à une adresse décompilée
ou une mesure rejouable. En (a) : dire explicitement si le fallback ti=13 est CONFIRMÉ,
INFIRMÉ (donner la permutation correcte), ou orthogonal.

## Garde-rails d'exécution

- Ghidra : lecture seule du binaire ; ne lance AUCUNE analyse destructive de projet ;
  si l'instance tombe en route, consigner l'état et STOP (l'utilisateur la relancera).
- Si des commandes `go` sont nécessaires (instrument 1.3) : UNE à la fois, GOCACHE privé
  (`<worktree>/.gocache`), CGO_ENABLED=0, données du principal en lecture seule.
- Rapport : `.ai/V7.5/replay2d/registre_film/LOT_RE_LETTRE_HUD.md` (dans TON worktree).
- Commits `re-lettre(pN): ...`, jamais `git add -A`, aucun push, pas de `.ai/thought_log.md`
  ni `REGISTRE_REPORTS.md` du principal.

## Découvertes

(consigner ici — rien corriger)

## CR attendu

Statut par item, adresses et fonctions clés (nom ou adresse + une ligne), verdict
(a)/(b)/(c), confrontation 1.3 si jouée, liste des commits.
