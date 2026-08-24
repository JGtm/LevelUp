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

- [x] 0.1 `mcp__ghidra__list_instances` puis connexion (`connect_instance`). Si aucune
      instance ou binaire non chargé dans le projet : STOP immédiat, CR court (c'est une
      action utilisateur, pas la tienne). Vérifier quel programme est ouvert.
      **FAIT** — `HaloInfinite.exe` chargé (311 103 fonctions). Écart : `connect_instance`
      refuse (nom de projet `unknown` en découverte UDS) ; le greffon répond sur
      `127.0.0.1:8089` (même PID 32880) et le lot l'a piloté par son API HTTP, en lecture
      seule. Consigné au rapport (§ encadré de tête) et en découverte n°4.
- [x] 0.2 Inventaire des points d'entrée, chacun consigné avec adresses :
      chaînes de localisation/désignation des zones (motifs candidats : « zone », «
      stronghold », « objective », clés de type hud/nav/waypoint, formats « %s ») ;
      références au navpoint/marqueur (symboles ou RTTI contenant navpoint/waypoint) ;
      la table/structure des objets de mode (managed-objective, pas 0x3810 connu).
      **FAIT** — table d'adresses au rapport §2 (`chud_variant_objective_designator`
      `143bf8d48` ; `objective_type_stronghold` `143bfdbc0` ; les 28 composants
      `managed-navpoint-*` de ti=12 ; 200+ liaisons de script `Navpoint_*` dont
      `Navpoint_SetDisplayText` `143c5c318` ; `ManagedGameEngine_ResolveStringId`
      `142c90f2c`).
- [x] 0.3 Choisir les 2-3 pistes les plus prometteuses et le dire au CR intermédiaire
      (une ligne dans le rapport, pas d'attente d'accord).
      **FAIT** — (1) `managed-navpoint-formatted-text-component`, (2) la fonction
      `string_id` `FUN_140748a74`, (3) `chud_variant_objective_designator` (fermée : 0
      lecteur sur 13,6 M d'instructions balayées).

Gate 0 : connexion établie + liste d'adresses concrètes (pas « à chercher »).
**TENU** (2026-08-24).

## Phase 1 — Remonter la source de l'index de lettre

- [x] 1.1 Depuis les points d'entrée, remonter les xrefs jusqu'à l'endroit où un index
      0/1/2 (ou 'A'+n) est choisi pour une zone : décompiler (`analyze_function`),
      identifier la donnée source (champ de l'objet de mode ? position dans une liste ?
      valeur répliquée ?).
      **FAIT — l'index N'EST PAS dans le binaire** (rapport §3). Aucune fonction
      `Stronghold*`/`Hill*`/`CaptureZone*` ; le moteur n'expose qu'une API au SCRIPT
      (`Objective_*`, `Navpoint_*`, `Managed*`, erreurs d'interpréteur,
      `temp_objective_fragments.lua`). La lettre sort du script par
      `Navpoint_SetDisplayText` / `Navpoint_SetDisplayFormattedText`. Acquis annexe :
      la fonction `string_id` du jeu est établie sur pièces (`FUN_140748a74` =
      normalisation + murmur3 x86_32 seed 0), recoupée sur 5 labels du dépôt.
- [x] 1.2 Dire si cette source est OBSERVABLE dans le film : si c'est un rang de liste,
      lequel (ordre de création ti=11 ? ordre du bloc de variante ?) ; si c'est une
      propriété répliquée, laquelle (un slot ti=13 ? un composant ti=12 non porté ?).
      **FAIT — canal désigné : `ti=12 i9` `managed-navpoint-formatted-text-component`**,
      déserialiseur `FUN_1410E7B90`, grammaire complète (rapport §4.2 : `R(8)` n puis
      n x { `R(32)` textStringId ; `R(1)` ; si 1 : `R(32)` + `R(3)` + args }).
      **NON observable aujourd'hui** : 1,7x le plancher de bruit en delta (206 annonces
      contre 119,5 sur `7344d24f`, mesure existante du lot C) — le texte vit dans
      l'image-clé, dont la lecture exige de porter d'abord `ti=12 i1..i8` (§4.3, §5).
- [!] 1.3 Si un canal film est désigné : le confronter sur le corpus local (instrument
      gaté env var, lecture seule, plafonné en mémoire — bombe RAM consignée) aux 2
      matchs Bastion : la permutation prédite reproduit-elle l'ordre ti=13 du fallback ?
      **NON JOUÉE** — le canal est désigné mais inatteignable dans le budget du lot :
      atteindre i9 dans l'image-clé impose de porter cinq listes de filtres
      (`ti=12 i2..i6`, déserialiseurs `140DBDE1C` / `140DBDF80` / `140DBDFAC` /
      `140DBE194` / `140DBDF34`) en plus de trois largeurs triviales. Coût chiffré et
      marche à suivre en 5 étapes au rapport §6. Aucun périmètre adapté en douce, aucun
      bit deviné.

Gate 1 : verdict (a)/(b)/(c) écrit, chaque affirmation adossée à une adresse décompilée
ou une mesure rejouable. En (a) : dire explicitement si le fallback ti=13 est CONFIRMÉ,
INFIRMÉ (donner la permutation correcte), ou orthogonal.
**TENU (2026-08-24) — verdict (c), négatif LOCALISÉ** : la règle n'est pas « introuvable
à coût raisonnable », elle est absente du binaire (donnée de script de mode). Acquis de
type (b) : le canal qui porte la lettre est désigné et décodé, mais non observable
aujourd'hui. **Repli ti=13 : ORTHOGONAL** — ni confirmé ni infirmé, aucune permutation
corrigée à livrer ; rapport §7 (un slot ti=13 est une propriété réseau nommée, sans lien
de construction avec le navpoint qui porte la lettre ; l'ordre des slots n'est déjà pas
l'ordre spatial — 1532/1537/1542 -> rangs 1/2/0 en phase 2a — mais il est stable).

## Garde-rails d'exécution

- Ghidra : lecture seule du binaire ; ne lance AUCUNE analyse destructive de projet ;
  si l'instance tombe en route, consigner l'état et STOP (l'utilisateur la relancera).
- Si des commandes `go` sont nécessaires (instrument 1.3) : UNE à la fois, GOCACHE privé
  (`<worktree>/.gocache`), CGO_ENABLED=0, données du principal en lecture seule.
- Rapport : `.ai/V7.5/replay2d/registre_film/LOT_RE_LETTRE_HUD.md` (dans TON worktree).
- Commits `re-lettre(pN): ...`, jamais `git add -A`, aucun push, pas de `.ai/thought_log.md`
  ni `REGISTRE_REPORTS.md` du principal.

## Découvertes

(consigner ici — rien corriger). Détail au rapport `registre_film/LOT_RE_LETTRE_HUD.md` §8.

1. **`mapvar.LabelHash` est la bonne fonction, amputée de sa normalisation.** Le jeu
   normalise avant de hacher (minuscules, `'-'`/`' '` -> `'_'`, `'\n'` -> `'#'`).
   Conséquence : la force brute du lot C-ter volet 2 a exploré des « variantes de casse »
   qui retombent toutes sur le MÊME hash — l'espace fouillé pour les trois labels de KOTH
   était plus petit qu'annoncé. Ré-ouvrir la chasse (avec espaces et tirets) est bon marché.
2. **Le « trio de tag 5 » de ti=13 a un candidat nommé** : le moteur expose exactement
   trois propriétés réseau de raison de fin de manche
   (`ManagedGameVariant_Set{Winning,Losing,Tie}RoundReasonNetworkedPropertyName`,
   `142c94bcc` / `142c9372c` / `142c93e34`). Colle à l'observation du volet 1 (trois slots
   consécutifs, une émission, 15-21 ms après la capture terminale ; deux slots seulement
   sur `0a247154`). Hypothèse NON mesurée.
3. **Les valeurs `ti=13 i0` publiées en phase 2a ne sont pas des string-ids propres**
   (six valeurs à fort préfixe nul sur 26 : incompatible avec un murmur3) — cohérent avec
   le chaînage de 1-3 % déjà mesuré. Les noms de propriété ne se lisent qu'en image-clé.
4. **Le pont MCP Ghidra ne sait pas nommer l'instance** (`project: unknown`,
   `connect_instance` refuse) alors que le greffon répond sur `127.0.0.1:8089`.

## CR attendu

Statut par item, adresses et fonctions clés (nom ou adresse + une ligne), verdict
(a)/(b)/(c), confrontation 1.3 si jouée, liste des commits.
