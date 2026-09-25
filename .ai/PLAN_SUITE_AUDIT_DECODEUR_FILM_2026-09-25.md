# PLAN — Suite de l'audit du décodeur de film et du générateur d'artefacts de rejeu (2026-09-25)

> **Statut : PROPOSÉ ; décisions DU-1 à DU-9 VALIDÉES le 2026-09-25 — rien n'est lancé.**
> « ok pour le plan » n'est pas un GO : chaque jalon
> démarre sur un GO explicite et daté de l'utilisateur. Contrat d'exécution : skill
> `plan-execution`, précisé au §4 (en cas de divergence, ce plan fait foi).
>
> **Source** : `.ai/AUDIT_DECODEUR_FILM_2026-09-24.md` (arbre `9d335ea43`) — 57 constats retenus
> (1 P0, 13 P1, 43 P2), 11 faiblesses d'architecture, 6 écarts ADR 0034 / arbre, 7 escalades.
>
> **Pièces re-vérifiées le 2026-09-25** sur `feat/v75` (`9d335ea43`) et sur les têtes de la campagne
> des retours rejeu, qui touche 26 des fichiers cités par l'audit : `feat/retours-rejeu`
> (`1a8390e9f`), `feat/rr-vague-d` (`c9ef97ec6`), `feat/rr-m4b` (`f2b52546b`, schéma 71,
> `grammar-2026-09-24`, `killsource-2026-09-24`), `feat/rr-m8` (`5e87b4511`). Les numéros de ligne
> cités sont indicatifs : **les symboles font foi**, et chaque lot rouvre ses pièces avant de coder
> (règle 4 du contrat).

---

## 0. Résumé

Le plan traite **tout ce que l'audit retient**, sous un périmètre fermé : chaque constat, chaque
faiblesse, chaque écart ADR et chaque escalade a **exactement un statut** dans la matrice du §2
(lot du plan, couvert ailleurs `[~]`, ou exclu `[!]` avec motif). Rien n'est ajouté qui ne sorte
pas du registre.

| Jalon | Contenu | Constats | Ce qui bouge | Fusion dans `feat/v75` | Taille |
|---|---|---|---|---|---|
| **J1** | Une seule passe killsource post-sync par processus et par titre | OPS-3 | rien (sync) | dès clôture, **avant v7.5 → main** | S |
| **J2** | Robustesse E/S : cache de films, verrou solo, lecteurs bornés, refus typés | SRC-2/OPS-4, OPS-1, OPS-2, OPS-5, RA1-5, RA1-7, RB1-4, GA1-1/GB-2, CONV-1 | aucune sortie (replay-equiv 0) | dès clôture | M-L |
| **J3** | Modèle de révision et fraîcheur des faits (amendement ADR 0034 D-6/D-7) | SRC-1, RA1-1, RA1-2, RA1-4 | outillage d'empreinte, codec des faits, `coverage.decoder` | vague J11 | M |
| **J4** | Carte de fermeture (J4.0) ; étage de balayage unique (S1) ; lecteurs de bits artisanaux (S2, sous benchmark) | RA1-3 + faiblesses 3 et 5 ; DU-8 | `IsolationDecoderRev` | vague J11 | L |
| **J5** | Identité (slot, génération) typée + **GB-1 (P0)** | GB-1, RA2-1, RA2-2, RA2-3, RA2-6, RB2-3, GA1-3 | `grammar.Rev`, schéma, isolement | vague J11 | L |
| **J6** | Un seul portage par fonction du jeu (DU-1) | GA2-2..GA2-5 (GA2-1 : reprise M4b de la campagne rr) | `grammar.Rev` | vague J11 | M |
| **J7** | killsource | FK-1..FK-7 | `killsource.Rev` (backlog) | vague J11 | M |
| **J8** | Replis D-10 : conversions et câblage des 81 compteurs | GA1-2, RB2-5, RB2-8, FO-1/RA2-4, FO-3, FO-4 | schéma (+ révisions à constante) | vague J11 | L |
| **J9** | Drapeau, bombe, zones | RB1-1, RB1-2, RB1-3, RB1-5..RB1-8 | schéma (+ persist si RB1-3) | vague J11 | M |
| **J10** | Déterminisme et correctifs résiduels | GB-3, RA2-5, RB2-6, RB2-7, GB-4, GA1-4, GA1-5 | `grammar.Rev`, schéma | vague J11 | M |
| **J11** | Vague unique : corpus gate, re-décodage, backfills, fusion | — | parc local (prod au déploiement) | oui | opérationnel |
| **J12** | Modernisation neutre, documentation, CI | faiblesses 7, 8, 10, 11 ; écarts ADR ; DU-5 | rien (replay-equiv 0) | oui | L |

Décisions : §3.1, validées le 2026-09-25 (les recommandations sont retenues).
Coût estimé : §4.7 (~20 agents Opus, effort high, jalon par jalon).

---

## 1. Cadrage

### 1.1 Objectif et critères de succès (mesurables)

1. **P0 GB-1** : sur `084a804d` (BTB 16 min), les corps de génération ≥ 2 ont des positions ; le
   nombre de corps sans aucune position passe de ~122 sur 379 (mesure du dépôt,
   `.ai/V7.5/RAPPORT_PONT_APLATI_2026-09-10.md`) à ceux que le film ne décrit réellement pas
   (chiffre mesuré au J5.0, cible écrite avant de coder) ; `durationMs` n'est plus tronqué ;
   aucune perte sur les autres témoins du corpus.
2. **Chaque constat P0/P1 corrigé** porte un test qui rougit sur le code d'avant (preuve de
   mutation consignée au rapport du lot).
3. **Chaque centralisation** (règle 6) porte son garde-rail dans le même commit.
4. **Aucune régression mesurée** : `replay-equiv` à zéro différence pour les jalons neutres
   (J2, J12, parties structurelles de J3/J4/J5) ; `replay-corpus-gate` sans perte, changements
   = liste déclarée d'avance, pour les jalons de comportement.
5. **Périmètre fermé** : à la clôture, aucune ligne de la matrice du §2 sans statut.
6. **CI verte au niveau job** sur la branche à chaque clôture de jalon, et sur `feat/v75` après
   chaque fusion.

### 1.2 Base, branche, worktree

- Branche : **`feat/suite-audit-decodeur`** (`feat/**` déclenche la CI au push ; jamais `wt/**`).
- Worktree dédié : **`LevelUp-wt-suite-audit`**, à côté du checkout principal, créé depuis
  `feat/v75` : `git worktree add ../LevelUp-wt-suite-audit -b feat/suite-audit-decodeur feat/v75`.
  Le chemin suit le checkout principal : une autre session prévoit (2026-09-25) de basculer le
  dossier de travail vers `C:/Users/Guillaume/Downloads/Scripts/LevelUp` et de faire le ménage
  des worktrees après la fusion de la campagne rr — vérifier `git worktree list` avant de créer.
- Une seule branche pour tout le plan (CLAUDE.md : 1 tâche = 1 branche, N commits), fusionnée dans
  `feat/v75` à quatre points : fin J1, fin J2, fin J11, fin J12. Après chaque fusion, `feat/v75`
  est re-fusionné dans la branche (en cas de conflit, `feat/v75` a raison).
- Un seul exécutant à la fois dans ce worktree (lots séquentiels). Un relecteur travaille dans un
  worktree temporaire détaché, en lecture seule, retiré à la fin de sa revue.

### 1.3 Prérequis

- [x] **P-1** Campagne des retours rejeu fusionnée dans `feat/v75` (intégration D : M4b, M8, M7b)
      et sa republication faite. **Seul J1 peut démarrer avant** (ses fichiers ne sont pas touchés
      par la campagne) ; J2 et la suite exigent P-1, puis `git merge feat/v75` dans la branche.
      FAIT le 2026-09-25 par la session de la campagne : merge `569932b42`, `feat/v75` =
      `3cca6cf47`, poussé ; republication des 111 matchs au schéma 71 faite avant la fusion.
- [x] **P-2** Registre d'audit (`.ai/AUDIT_DECODEUR_FILM_2026-09-24.md`) et ce plan commités sur
      `feat/v75` le 2026-09-25, à la demande de l'utilisateur (« commite juste les plans et autres
      docs en attente »), avec le handoff de la campagne perf ; non poussé.
- [x] **P-3** (FAIT le 2026-09-25 sur `3cca6cf47` : 0 / 7 / 3 / 2 / 0 occurrences, et les deux
      positions de `consumeMobilityActionBody` passent par `consumeSimStateHandleTail` — les cinq
      lignes `[~]` sont confirmées) Re-vérification sur l'arbre fusionné des cinq corrections venues de la campagne
      (statuts `[~]` du §2) :
      `git grep -n "pick := list\[0\]" -- apps/go-api/internal/games/halo_infinite/film/replay/` → 0 ;
      `git grep -n "repli_place_du_remplacant_par_chainage_d_equipe" -- apps/go-api/internal/` ≥ 1 ;
      `git grep -n "DeathsFeed" -- apps/go-api/internal/games/halo_infinite/film/replay/filmfacts*.go` ≥ 1 ;
      `git grep -n "atomicfile" -- apps/go-api/internal/games/halo_infinite/film/filmcache/write.go` ≥ 1 ;
      GA2-1 (reprise M4b) : `git grep -n "consumeE494Position" -- apps/go-api/internal/games/halo_infinite/film/internal/grammar/` → 0
      et les deux positions de `consumeMobilityActionBody` lues par `consumeSimStateHandleTail`.
      Un résultat contraire requalifie la ligne `[~]` en item du plan (journal §9).
- [x] **P-4** Décisions DU-1 à DU-7 tranchées le 2026-09-25 par l'utilisateur : « ok avec toi alors
      tu peux entériner ces décisions avec tes recos » (§3.1) ; DU-8 et DU-9 le même soir.
- [ ] **P-5** Quota hebdomadaire : GO donné en connaissance de l'estimation du §4.7 (quota remis à
      zéro le 2026-09-27 à 6 h).

### 1.4 Hors périmètre (explicite)

| Élément | Motif |
|---|---|
| Faiblesse 6 : décodage multi-passes (~40 traversées, 8 canaux qui refont `walkDeltaBipedRecords`) | Performance, aucune donnée fausse ; l'audit ne recommande pas d'action. Chantier perf séparé si voulu. |
| Découpe complète de `film/replay` (189 fichiers, `Options` ~70 champs en entrée et en sortie) et de la façade | DU-4 (retenu : hors plan). Seules les parties qui corrigent un constat sont faites (J3.4, J4). |
| Représentation intermédiaire du film (la grammaire parcourt le film une fois et range tout ce qu'elle a décodé ; la résolution lit cette structure) | DU-8 (b) : chantier futur, spécifié dans `.ai/SPEC_REPRESENTATION_INTERMEDIAIRE_FILM_2026-09-25.md` ; déclencheur = fermeture des records qui portent une donnée utile, mesurée par la carte de fermeture (J4.0, rejouée en J11.3). |
| 683 `t.Skip`, 3 `t.Parallel` (faiblesse 7, partie) | Aucun constat : les `t.Skip` relevés sont des absences de film local justifiées. |
| Retrait des replis à compte nul | DU-7 : règle 4 de D-10 (« supprimé au jalon suivant ») ; J11 produit la liste mesurée. |
| Constats écartés par l'audit (FO-2, OPS-6, RB2-2, RB1-9, partie de FO-4, occurrence citée de RA2-2) | Réfutés en vérification adverse. |
| Axes non passés (A3 multi-titre, A9 KPI, A10 front) | Non audités. |
| Découvertes du §8 du plan des retours rejeu, sauf 8.4 (= RA1-6), 8.22 (instance de SRC-2, J2.5) et 8.26 (chemins `.ai/` périmés, J12.5) | Hors registre d'audit. 8.20 (inventaire des morceaux absent de l'en-tête des faits) devient sans objet après J2.3 : un chunk tronqué ne se charge plus. |
| Déploiement en production | Geste de l'utilisateur ; J11 écrit la séquence. |

---

## 2. Matrice de traçabilité — le périmètre fermé

Statuts à la clôture : `[x]` fait et vérifié · `[~]` couvert ailleurs (référence) · `[!]` non
traité (justification au journal §9). Aucune case vide.

### 2.1 Constats P0 et P1

| ID | Grav. | Constat (résumé) | Traitement | Statut |
|---|---|---|---|---|
| GB-1 | P0 | Positions et canaux delta bipède limités à la génération 1 du handle | J5.0, J5.2, J5.5 | [ ] |
| OPS-3 | P1 | Étape 1.57 lancée une fois par joueur, en parallèle, sur le même arriéré | J1 | [ ] |
| SRC-2/OPS-4 | P1 | Chunks écrits sans atomicité, « présent » tenu pour « complet » | J2.1-J2.5 (manifeste atomique : `[~]` rr L3, contrôle P-3) | [ ] |
| FK-1 | P1 | Un remplaçant humain désépingle le bot de relais, sans signal | J7.2 | [ ] |
| FK-2 | P1 | Nom de remplissage `?N` publié comme assistant nommé | J7.1 | [ ] |
| GA2-1 | P1 | Portage divergent de `FUN_14076e524` (corps d'i54) | Reprise M4b de la campagne rr, fusionnée le 2026-09-25 (`3cca6cf47`) : les deux positions d'i54 lues au niveau 0x10 par `consumeSimStateHandleTail`, `consumeE494Position` supprimé — confirmé par P-3 ; les autres sites du même lecteur restent en J6 | [~] |
| GA2-2 | P1 | `flock-position` lu au niveau 0 au lieu de l'immédiat 0x10 | J6 | [ ] |
| GA1-2 | P1 | Repli « liaison par anticipation » hors registre, non compté | J8.1 | [ ] |
| RA1-1 | P1 | Faits persistés dépendants des gardes de l'appelant, non comparées | J3.4 | [ ] |
| RB2-1 | P1 | Sièges : un arrivant précoce arrête l'appariement du camp | rr vague D (M2 : chaînage nommé `repli_place_du_remplacant_par_chainage_d_equipe`) — confirmé par P-3 sur `3cca6cf47` | [~] |
| RB2-4 | P1 | `spawnSetFrom` compare à une image-clé future | rr vague D (M3.2 : « jamais un relevé à venir », borne de vie) — confirmé par P-3 sur `3cca6cf47` | [~] |
| RB2-5 | P1 | Un ramassage natif date plusieurs occupations de socle | J8.2 | [ ] |
| RB1-1 | P1 | `flag_carriers_killed` peut fermer le portage d'un porteur vivant | J9.1 | [ ] |
| FO-1/RA2-4 | P1 | Provenance `residu_de_manche` publiée « déduit » sans voie | J8.4 | [ ] |

### 2.2 Constats P2

| ID | Traitement | Statut | ID | Traitement | Statut |
|---|---|---|---|---|---|
| SRC-1 | J3.2 | [ ] | FK-6 | J7.6 | [ ] |
| RA1-3 | J4 | [ ] | FK-7 | J7.7 | [ ] |
| RA1-4 | J3.5 | [ ] | RB2-3 | J5.4 | [ ] |
| RA1-6 | rr M8 (verdict du fil des morts dans les faits) — confirmé par P-3 | [~] | RB2-6 | J10.3 | [ ] |
| RA1-5 | J2.7 | [ ] | RB2-7 | J10.4 | [ ] |
| RA1-2 | J3.6 | [ ] | RB2-8 | J8.3 | [ ] |
| RA1-7 | J2.10 | [ ] | RB1-2 | J9.4 | [ ] |
| OPS-1 | J2.6 | [ ] | RB1-3 | J9.6 | [ ] |
| OPS-2 | J2.13 | [ ] | RB1-4 | J2.8 | [ ] |
| OPS-5 | J2.12 | [ ] | RB1-5 | J9.2 | [ ] |
| FO-3 | J8.5 | [ ] | RB1-6 | J9.3 | [ ] |
| FO-4 | J8.6 | [ ] | RB1-7 | J9.5 | [ ] |
| GA2-3, GA2-4, GA2-5 | J6 | [ ] | RB1-8 | J9.7 | [ ] |
| GA1-1/GB-2 | J2.9 | [ ] | CONV-1 | J2.11 | [ ] |
| GA1-3 | J5.4 | [ ] | RA2-1 | J5.4 | [ ] |
| GA1-4 | J10.5 | [ ] | RA2-2 | J5.4 | [ ] |
| GA1-5 | J10.6 | [ ] | RA2-3 | J5.4 | [ ] |
| GB-3 | J10.1 | [ ] | RA2-5 | J10.1 | [ ] |
| GB-4 | J10.2 | [ ] | RA2-6 | J5.4 | [ ] |
| FK-3, FK-4, FK-5 | J7.3, J7.4, J7.5 | [ ] | | | |

### 2.3 Faiblesses d'architecture (§« Architecture et conventions » du registre)

| # | Faiblesse | Traitement | Statut |
|---|---|---|---|
| 1 | Fraîcheur et révisions mal calées sur les sorties | J3 (+ J4 pour RA1-3) | [ ] |
| 2 | Identité (slot, génération) non typée | J5 | [ ] |
| 3 | Portages jumeaux ; 7 lectures de bits artisanales hors `source` | J6 (portages) ; J4.6 (lectures, S2) | [ ] |
| 4 | D-10 tenue par un ratchet de vocabulaire ; 81/99 compteurs non câblés | J8 | [ ] |
| 5 | `film/replay` paquet-dieu ; seconde séquence de balayage dans `killcollector` | J4 (séquence unique) ; découpe complète : DU-4 | [ ] |
| 6 | Décodage multi-passes | `[!]` hors périmètre (§1.4) | [ ] |
| 7 | CI : `-race` absent du film ; instruments `research` non tagués | J12.6 ; J12.7 (DU-5) | [ ] |
| 8 | Contexte et journalisation (`context.Background()`, `slog` sans contexte, D-4 à moitié, variables exportées modifiables) | J12.3, J12.4 | [ ] |
| 9 | Déterminisme par chance (43 `sort.Slice`) | J10.1 | [ ] |
| 10 | Bibliothèque standard moderne quasi absente | J12.1, J12.2 | [ ] |
| 11 | Documentation périmée | J12.5 | [ ] |

### 2.4 Écarts ADR 0034 / arbre

| Affirmation fausse aujourd'hui | Rendue vraie par | Statut |
|---|---|---|
| « replay ne décode rien » | J4.2 | [ ] |
| « seule porte aux octets » (lectures artisanales, façade qui ré-expose `LecteurSur`/`Paquets`/`Inflate`) | J4.5 (façade), J4.6 (lecteurs, S2 ; sinon ADR D-2 corrigé) | [ ] |
| « D-5 prouvé sous -race » | J12.6 | [ ] |
| « un seul étage de balayage » | J4.3 | [ ] |
| « verrou pris par les SIX points d'entrée » (un septième ne le prend pas) | J2.13 (identifier le septième au J2.13, avant de coder) | [ ] |
| « D-10 Reached as written » | J8 puis J12.5 | [ ] |

### 2.5 Escalades (§Suite du registre)

| # | Escalade | Traitement | Statut |
|---|---|---|---|
| 1 | OPS-3 avant la fusion v7.5 → main | J1 | [ ] |
| 2 | GB-1 : mesurer le parc puis lot de comportement | J5 | [ ] |
| 3 | Modèle de révision / fraîcheur | J3 (DU-2) | [ ] |
| 4 | Identité (slot, génération) de première classe | J5 | [ ] |
| 5 | GA2-1 / GA2-2 (résidu de film dense clos le 23/09) | DU-1 = oui → J6 (GA2-1 : `[~]` reprise M4b) | [ ] |
| 6 | Découpe de `film/replay` et façade (décision V25) | DU-4 = hors plan (`[!]` à la clôture, motif : décision du 2026-09-25) | [ ] |
| 7 | Politique de commentaires, tag `research` | DU-5 → J12.7, J12.8 | [ ] |

---

## 3. Décisions

### 3.1 Décisions utilisateur — VALIDÉES le 2026-09-25

Message de l'utilisateur (2026-09-25) : « ok avec toi alors tu peux entériner ces décisions avec
tes recos ». Chaque décision retient la recommandation ; elles sont FERMES (règle 10 du contrat :
ne pas les re-décider en cours d'exécution). DU-3 retient l'option S découpée en S1 / S2, telle que
recommandée le même jour.

**DU-1 — Portages jumeaux (GA2-1 à GA2-5) : ouvrir J6 ?**
GA2-1 et GA2-2 sont des candidats (marginaux) au résidu de film dense, clos par la décision du
2026-09-23. J6 ne rouvre PAS la recherche d'un troisième suspect : il corrige des défauts prouvés
par l'audit (sous-lecture de 22 à 48 bits par record `ti=21`, i54 lu à 344 bits) en ramenant
chaque fonction du jeu à UN portage (principe de non-redondance), vérifié dans Ghidra.
**Retenu : oui.** GA2-1 est traité entre-temps par la reprise M4b de la campagne rr (§2.1) ; J6
garde GA2-2 à GA2-5 et les autres sites du même lecteur.

**DU-2 — Modèle de révision (amendement ADR 0034 D-6/D-7) : adopter les cinq changements de J3 ?**
(a) empreinte insensible aux commentaires ; (b) périmètre d'une couche = fermeture de ses imports,
figée par un golden de paquets (corrige SRC-1 par construction) ; (c) une révision par consommateur
de faits (`killsource.Rev`, `objectives.Rev`) au lieu d'un `facts.Rev` unique (plus de backlog
killsource rouvert par un changement d'objectifs) ; (d) gardes de l'appelant inscrites dans les
faits et comparées par `Utilisable`, aucun fait écrit quand l'artefact est refusé (RA1-1) ;
(e) clé de cuisson = empreinte de toute l'entrée de catalogue (RA1-4).
**Retenu : les cinq.**

**DU-3 — Étage de balayage unique (J4) : option S ou M ?**
S (structurelle) : le balayage quitte `film/replay` pour `grammar`, la cuisson et le collecteur
appellent un seul étage de lectures ; les écarts ADR « replay ne décode rien », « un seul étage »
deviennent vrais. M (minimale) : un seul constructeur d'options et une seule fonction « balayer le
pont d'identité » dans `replay`, partagés.
**Retenu : S, en deux temps.** M est écartée parce qu'elle ne sait pas corriger RA1-3 proprement :
les révisions couvrent des PAQUETS ; hacher le code de balayage resté dans `replay` exigerait soit
une liste de fichiers tenue à la main (le défaut que l'audit reproche aux périmètres actuels), soit
de hacher tout `replay` (toute modification de publication re-décoderait le parc, contre D-7).
- **S1 (obligatoire)** : cinq fichiers de lecture pure descendent dans `grammar`, leurs types de
  résultat dans `film/types`, un seul étage de lectures du pont d'identité sert la cuisson et le
  collecteur (J4.1 à J4.5). Le registre d'identité reste dans `replay` (déjà partagé, décision D11) :
  c'est la borne qui sépare S1 de la découpe complète (DU-4).
- **S2 (sous condition fixée d'avance)** : les 7 lecteurs de bits artisanaux passent au lecteur
  canonique de `source` si `replay-equiv` reste à zéro ET si le benchmark du balayage ne régresse pas
  de plus de 5 % ; sinon arrêt, rapport, exceptions datées dans un ratchet structurel et ADR D-2
  corrigé (J4.6).
Le ratchet de surface de la façade (décision V25) est réajusté avec justification datée.

**DU-4 — Découpe complète de `film/replay` et d'`Options` : dans ce plan ?**
**Retenu : non** (chantier et ADR dédiés ; risque de conflit massif ; la décision V25 du
2026-09-18 tient). Ce plan n'en fait que ce qui corrige un constat (J3.4, J4).

**DU-5 — Instruments `research` et politique de commentaires (J12.7, J12.8) ?**
(a) Taguer les 218 `*_research_test.go` non tagués (115 le sont déjà), ratchet
`archlint/research_tag_test.go` sur le modèle de `gamefiles_tag_test.go`, étendre le
`go vet -tags=research` de la CI (aujourd'hui limité à `film/internal/grammar/`) à tout le module ;
(b) règle écrite dans CLAUDE.md et `arch-rules` : « le code porte le contrat ; l'histoire va dans
l'ADR, la chronique ou le journal », appliquée au code neuf et aux affirmations fausses — pas de
réécriture de masse des ~55 000 lignes de commentaires.
**Retenu : (a) et (b).**

**DU-6 — Une seule vague de re-décodage (J11) pour J3-J10 ?**
**Retenu : oui.** Aucun jalon J3-J10 n'est fusionné seul dans `feat/v75` (sinon le parc se
re-décoderait par morceaux au fil des cycles de dev). La vague elle-même reste un GO opérationnel
au moment venu (serveur arrêté, machine libre).

**DU-7 — Retrait des replis à compte nul après le câblage (J8) ?**
**Retenu : hors plan.** J11 publie la liste mesurée ; le retrait suit la règle 4 de D-10
au jalon suivant, sur décision.

**DU-8 — Carte de fermeture et représentation intermédiaire (ajout validé le 2026-09-25).**
Message de l'utilisateur : « Ok je suis d'accord avec toi. Donc tu mets à jour le plan actuel pour
la carte de fermeture et tu prépares une spec pour la structure intermédiaire ». Distinction qui
fonde la décision : TRAVERSER un composant (connaître sa largeur pour atteindre le suivant) n'est
pas l'INTERPRÉTER (savoir ce que la valeur veut dire). Sur la tête de la campagne rr, la table ECS
compte 1 067 composants, dont 44 servent au produit ; 544 sont portés, dont 502 sans aucun usage
produit — portés pour être SAUTÉS. On n'interprétera jamais tout, et il n'en est pas besoin ; la
traversée, elle, se mesure et progresse.
**Retenu :** (a) la **carte de fermeture** entre au plan comme lot J4.0 (instrument de mesure,
aucune sortie de production modifiée) : elle donne la référence de fermeture avant les jalons de
grammaire et l'ordre de portage le plus rentable ; (b) la **représentation intermédiaire du film**
fait l'objet d'une spécification de chantier FUTUR,
`.ai/SPEC_REPRESENTATION_INTERMEDIAIRE_FILM_2026-09-25.md`, hors de ce plan (§1.4) ; son
déclencheur est mesuré par la carte.

**DU-9 — Largeurs présumées par mesure (décidée le 2026-09-25).**
Message de l'utilisateur : « Oui ok avec toi ». Pour un composant de taille FIXE qu'on ne fait que
sauter, une largeur trouvée par essais (la seule avec laquelle les paquets se ferment au bit près),
vérifiée sur tout le corpus de chaque build, est admise comme valeur PRÉSUMÉE : provenance écrite
(mesure, films, date) dans `ecs_table.tsv` et au code, liste gelée testée sur le modèle de
`empreintesPresumeesGelees`. Ghidra reste obligatoire pour les composants dont la valeur est
utilisée, ceux de taille variable, et toute largeur présumée qui casse sur un build nouveau.
Amendement de l'ADR 0034 D-3 écrit en J3.7 ; règle détaillée dans la spec
(`.ai/SPEC_REPRESENTATION_INTERMEDIAIRE_FILM_2026-09-25.md`, §11).

### 3.2 Décisions techniques du plan (fermes, sauf objection au GO)

| ID | Décision | Pourquoi |
|---|---|---|
| DT-1 | OPS-3 : exclusivité par (processus, titre), `TryLock` ; le perdant saute, compte et journalise en DEBUG. Pas de `singleflight` par match, pas de déplacement de l'étape au niveau du cycle (ADR 0027 inchangé). | `singleflight` ferait itérer N passes en file indienne et compter N fois `ecrits` ; `DefaultPostSyncPerCycle` est une borne PAR CYCLE, qu'une passe unique rétablit. |
| DT-2 | OPS-1 : verrou OS (`LockFileEx` / `flock`) via `golang.org/x/sys`, déjà dépendance directe, dans un nouveau paquet `platform/filelock`. Battement de cœur et reprise de verrou périmé supprimés. | Le noyau rend le verrou à la mort du détenteur ; `gofrs/flock` écarté (une dépendance pour ~40 lignes au-dessus de `x/sys`). |
| DT-3 | Chunks écrits par `atomicfile.WriteFileStrict` (atomique ou erreur) ; `WriteFile` réécrit au-dessus (strict, puis repli in-place). | L'en-tête d'`atomicfile` interdit le repli in-place pour une perte irréversible ; une seule implémentation de l'écriture atomique. |
| DT-4 | Taille : annoncée par l'API comparée au téléchargement ; inscrite au manifeste (`size_bytes`, facultatif) et comparée à la lecture. Manifeste historique sans taille : lisible, non rétro-rempli. | « Présent » cesse d'être « complet » aux deux bouts. |
| DT-5 | Raison d'un refus d'enfant de cuisson : ligne de protocole `filmproc` (même mécanisme que `EmitPeak`/`parsePeak`), jeton opaque ; le vocabulaire vit dans `replaychild`. Aucun classement par texte. | Réutilise le canal existant ; `filmproc` reste générique. |
| DT-6 | Nom de remplissage : un prédicat unique dans `killsource/roster.go` (où il est fabriqué) + ratchet. | Contradiction interne au paquet (FK-2). |
| DT-7 | `types.EquipmentLifeKey` renommé `types.LifeKey{Slot, Gen}` (renommage mécanique, aucun alias) ; lecture du handle centralisée dans un helper de `grammar` + ratchet sur le littéral `p+14, 2)`. | La clé (slot, génération) existe déjà pour les objets du monde : on la généralise, on n'en crée pas une seconde. |
| DT-8 | Le filtre `RequireTag1` devient un prédicat de génération vivante (records de création + images-clés). Génération inconnue pour un slot : repli nommé, compté, qui garde `tag == 1`. | Garde le pouvoir discriminant du filtre (2 bits) sans perdre les corps de génération ≥ 2. |
| DT-9 | Tris : `slices.SortStableFunc`/`SortFunc` à comparateur TOTAL (chaîne `cmp.Or` finissant sur une clé unique) ; ratchet interdisant `sort.Slice*`/`sort.Sort` dans le périmètre. | Faiblesse 9 ; pdqsort n'est stable par accident que sous 13 éléments. |
| DT-10 | Modernisation : `go fix` Go 1.26 + stdlib, SAUF les quatre pièges de l'audit (tri non total, `slices.Clone` nil/vide en JSON, `omitzero`, `math/rand` → `math/rand/v2`). Un test fige la permutation à graine fixe de `killsource/options.go`. | La permutation fait partie de la sortie de la bijection. |
| DT-11 | Mesures sur films : instruments en test tagué `research` (ou outil existant), superviseur seul, un film à la fois (§4.3). | Quatre sinistres RAM passés. |
| DT-12 | Carte de fermeture (J4.0) : `grammar.FrameClosure` prolonge le contrat de `KeyframeClosure` (fermés / total / bloquant le PLUS FRÉQUENT, départage par nom) aux trames delta, par vue et par archétype, en réutilisant les marcheurs existants (`DecodeFrameViewsCurseur`, `LectureVueC` et `ArretVueC`, vue des messages) ; golden `testdata/frame_closure.golden` sur les huit mini-bobines et ratchet « aucune baisse », même porte de régénération que `keyframe_closure_ratchet_test.go` ; outil de corpus `film/research/cmd_fermeture` sur le modèle de `film/research/cmd_grenadeids` (tag `research`, sentinelle `filmproc.Arm`, films un à un, rapport écrit hors de `data/`). | Aucune roue réinventée : même contrat, mêmes marcheurs, même modèle d'instrument ; l'outil vit sous `film/`, donc la façade `decfilm` ne grandit pas. |

---

## 4. Contrat d'exécution

### 4.1 Règles

1. Les 10 règles du skill `plan-execution` s'appliquent.
2. **Ordre** : J1 → J12, strictement. Exceptions écrites : J1 peut démarrer avant P-1 ; J12
   démarre après la fusion de J11.
3. **Dans un jalon** : lots dans l'ordre, items dans l'ordre, protocole TDD (§4.2).
4. **Clôture d'un jalon** = (a) gate du jalon passé, sorties propres, code de sortie vérifié ;
   (b) items statués ; (c) ce fichier mis à jour (cases + journal §9) et commité ; (d) entrée
   `.ai/thought_log.md` (hunks stagés par patch — surface partagée) ; (e) CI de la branche verte au
   niveau job (`gh run list --branch feat/suite-audit-decodeur`) ; (f) point d'étape à
   l'utilisateur (fait / sauté + pourquoi / découvert).
5. **Revue adversariale** (skill `adversarial-review`) en fin de J1, J3, J5 et J7 (sync, fraîcheur,
   identité, données persistées) : un relecteur, un worktree temporaire en lecture seule ; ses
   constats sont corrigés dans le jalon ou consignés au §8. Pas de chaîne impl → revue →
   correction systématique.
6. **Découvertes** : §8, non traitées, sauf si elles bloquent le gate courant.
7. **Explosion de périmètre** (> 2x l'estimation) ou hypothèse invalidée (ex. : J5.0 ne reproduit
   pas GB-1) : arrêt propre et message immédiat à l'utilisateur.

### 4.2 Protocole TDD et preuve de mutation

1. **Rouge d'abord** : pour chaque correction, écrire le test qui encode la règle enfreinte (la
   colonne « Règle » du registre), l'exécuter, OBSERVER l'échec, coller la ligne `--- FAIL` au
   rapport du lot. Le test et le correctif partent dans le MÊME commit (la CI reste verte commit
   par commit) ; le rouge est prouvé par le rapport, pas par un commit rouge.
2. **Vert** : implémenter au plus court dans la bonne couche.
3. **Mutation** : pour chaque P0/P1 et chaque ratchet, réintroduire le défaut (une ligne) →
   rouge → restaurer ; consigné au rapport (mutation, test qui rougit).
4. **Structurel** (déplacement, renommage, `go fix`) : pas de nouveau test de comportement ; preuve
   = `replay-equiv` à zéro différence + tests existants verts.
5. **Fixtures** : synthétiques d'abord (fonctions pures), puis mini-bobines du dépôt
   (`film/replay/testdata/minifilm_*`, ≤ 1 Mio par build), puis films réels UNIQUEMENT par les
   gates locaux (jamais en CI).
6. **Fuzz** : nouvelles cibles sur le modèle de `grammar/fuzz_records_test.go` — graines sous
   `testdata/fuzz/<Cible>/`, rejouées par chaque `go test` ; campagne longue à la main, jamais en
   CI ; commande documentée dans l'en-tête du fichier.
7. **Garde-rails** : toute centralisation (règle 6) livre son ratchet dans le même commit, avec sa
   mutation. Ratchets prévus : nom de chunk (J2.3), comptes du codec des faits (J2.7), classement
   par texte (J2.12), décodage dans `internal/api` (J2.13), périmètres de révision (J3.2),
   fermeture des trames (J4.0),
   `replay.Scan*` hors de `film/` (J4.3), lecteurs de bits hors `source` (J4.6), handle
   (J5.1), nom de remplissage (J7.1), replis comptés (J8.7), tri total (J10.1), chemins `.ai/`,
   `film/filmdec/`, `slog` sans contexte, `slog` dans `grammar`/`facts`, variables exportées
   (J12), tag `research` (J12.7).
8. **Fichiers ≤ 500 lignes, fonctions ≤ 80 lignes** (ratchets `film_file_size_test.go`,
   `film_function_length_test.go`) ; paramètres ≤ 5.

### 4.3 Règles opérationnelles

- **Git** : jamais le checkout principal (partagé) ; jamais `git add -A` (staging nommé) ; jamais
  `git stash` ; `git log -1` avant chaque commit et chaque push (une autre session peut écrire).
  Le GO d'un jalon vaut accord pour les commits et les pushes de CE jalon sur la branche du plan ;
  toute fusion dans `feat/v75` se demande à l'utilisateur, jalon par jalon.
- **`go`** : une commande à la fois par worktree, jamais deux `go build`/`go test` en même temps
  sur la machine ; `GOCACHE=<worktree>/.gocache-audit` ;
  `GOLANGCI_LINT_CACHE=<worktree>/.golangci-cache` ; CGO quand DuckDB ou `himap` est tiré :
  `PATH=C:\msys64\ucrt64\bin;%PATH%` + `CGO_ENABLED=1`.
- **Décodage de film** (`replay-equiv`, `replay-corpus-gate`, sondes, re-décodage) : **superviseur
  seul**, premier plan, un à la fois, sous `filmproc.AcquireSolo`, jamais en boucle de shell ni en
  arrière-plan, jamais pendant un `go build` ; annoncé à l'utilisateur avant ; un BTB en sonde
  seulement au J5.0, après accord. Signal d'alerte (`fork: Resource temporarily unavailable`) :
  tuer immédiatement.
- **Jonctions** `data/cache/film_chunks` et `data/cache/film_manifests` du worktree vers le
  principal : `mklink /J` (jamais `ln -s` sous MSYS). Retrait AVANT `git worktree remove`, par
  PowerShell `(Get-Item <jonction>).Delete()`, puis vérification ; `git worktree remove` SANS
  `--force`.
- **Faits persistés** : aucun gate n'écrit dans `data/cache/film_facts` du principal (racine
  temporaire). Contrôle après chaque gate : poser un marqueur avant, puis
  `find <principal>/data/cache/film_facts -newer <marqueur> | wc -l` = 0.
- **`replay-equiv`** : `-repo-root <worktree>` (jamais `LEVELUP_REPO_ROOT`, qui ferait écrire les
  références du principal). **`replay-corpus-gate`** : `--source-root <worktree>
  --parc-root <principal>`, serveur dev arrêté (lecture de la base partagée, modèle mono-process).
- **Agents** : Opus explicite, effort high, un exécutant par lot ; ni push, ni serveur, ni
  décodage, ni ouverture des bases vivantes ; rapport = fait / non fait + justification /
  découvertes / mutations jouées / rouge observé. Le superviseur vérifie le rapport SUR PIÈCES
  avant de le relayer.

### 4.4 Recettes

- **Montée de révision de couche** : constante `Rev` (`<couche>-AAAA-MM-JJ[.n]`), entrée de
  chronique dans le MÊME commit, golden par la porte (`-update-<couche>-rev` +
  `LEVELUP_UPDATE_<COUCHE>_REV=1`), second oracle `film/revision/equivalence_test.go` vert.
- **Régénération à révision constante** (changement prouvé neutre) : même porte ; la dernière
  ligne du golden est réécrite ; la preuve (`replay-equiv` 0, équivalence killsource 0) est citée
  dans le message de commit.
- **Montée de schéma** : `SchemaVersion`++ (`film/replay/document.go`), entrée de chronique dans le
  même commit, table `couchesDesCalques` à jour, fixtures de contrat régénérées
  (`document_shape_test.go`, porte + `LEVELUP_CONTRACT_FIXTURES=1`), `openapi.yaml` +
  `make generate-types` si la forme API change, normaliseur web (`replayNormalize.ts`) si un champ
  apparaît, puis `make check-types` et `make test-web`.
- **Codec des faits** : `VersionCodecFaits` (conteneur, en-tête) ou `SchemaDesFaits` (charge) ; un
  test de refus d'un fichier de l'ancienne version accompagne toute montée.
- **Test renommé ou supprimé** : retiré de la baseline JSONL dans le même commit.

### 4.5 Gates standard (depuis `apps/go-api`)

| Nom | Commande | Qui |
|---|---|---|
| G-unit | `go test <paquets du lot> -count=1` | exécutant |
| G-arch | `go test ./internal/archlint/ -count=1` | exécutant |
| G-vet | `go vet ./...` (CGO) | exécutant |
| G-lint | `make go-api-lint` | exécutant |
| G-integ | `go test -tags=integration -p 1 <paquets sync/persist/migration touchés> -count=1`, code de sortie vérifié, échecs par `grep '^--- FAIL:'` ancré | exécutant |
| G-film | `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1` | superviseur, clôture |
| G-equiv | `go run ./cmd/replay-equiv -repo-root <worktree>` → zéro divergence | superviseur |
| G-corpus | `go run ./cmd/replay-corpus-gate --reference=base --source-root <worktree> --parc-root <principal> --json <rapport>` → 0 perte, changements = liste déclarée | superviseur |
| G-web | `make check-types` (après purge de `node_modules/.tmp`), `make test-web` (hors sandbox) | superviseur, si schéma |
| G-CI | `gh run list --branch feat/suite-audit-decodeur --limit 3` → jobs verts | superviseur |
| G-push | `make gate-push` avant toute fusion dans `feat/v75` | superviseur |

### 4.6 Commits, fusions, journal

- Préfixe de commit : `audit(Jx.y): …` ; au moins un commit par lot ; fin des messages : ligne
  `Co-Authored-By` en vigueur.
- Fusion dans `feat/v75` : `git merge --no-ff` depuis un checkout propre, `make gate-push`, push,
  CI verte au niveau job ; prévenir l'utilisateur (push `main` = déploiement : jamais par ce plan).
- `.ai/thought_log.md` : une entrée par clôture de jalon (date, titre, statut, décision, résultats,
  suite).

### 4.7 Mode d'exécution et coût (à valider, P-5)

- Superviseur : la session principale (pilotage, vérification sur pièces, gates de film, Ghidra en
  lecture seule par HTTP direct `127.0.0.1:8089`).
- Exécutants : J1 : 1 · J2 : 2 (J2-a E/S, J2-b lecteurs et enfant) · J3 : 1 · J4 : 2 (J4.0,
  puis J4.1-J4.6) · J5 : 2 (J5.1-J5.3, puis J5.4) · J6 : 1 · J7 : 1 · J8 : 2 (J8.1-J8.6, puis J8.7)
  · J9 : 1 · J10 : 1 · J12 : 2 → **16 exécutants**, plus **4 relecteurs** (J1, J3, J5, J7) =
  **~20 agents Opus** (la carte de fermeture ajoute un exécutant de taille moyenne).
- Ordre de grandeur : la campagne des retours rejeu (67 agents, effort high/max, chaînes
  impl → revue → correction, gates rejoués par chaque rôle) a consommé un quota hebdomadaire en
  ~30 h. Ce plan, à effort high, un exécutant borné par lot et une revue par jalon à risque :
  **~25 à 35 % d'un quota hebdomadaire**, étalé sur deux semaines. J11 coûte surtout du temps
  machine (re-décodage local, backfill killsource ~1 h).
- Recommandation : un GO par jalon ; J1 (et J2 si P-1 est acquis) dès la remise à zéro du quota,
  avant la fusion v7.5 → main.

---

## 5. Jalons et lots

### J1 — Une seule passe killsource post-sync par processus et par titre (OPS-3)

**But.** L'étape 1.57 décode chaque film au plus une fois par cycle, quel que soit le nombre de
joueurs synchronisés en parallèle. La doctrine « le post-sync du serveur garde la boucle en série »
(`sync/killcollector/collector.go`, en-tête) redevient vraie par construction. Absent de `main`
(`cf333a388`) : le défaut arrive avec la fusion v7.5 — **ce jalon la précède**.

**Pièces vérifiées** (`feat/v75`, fichiers non touchés par la campagne rr) :
- `sync/killcollector/postsync.go` `RunPostSync` : arriéré GLOBAL (`backlogAJour`, aucun filtre
  joueur, `LIMIT PostSyncBacklogHorizon`), `ordonnancer`, `CollectMatches` ;
- `sync/v2/post_sync.go` `RunPostSync` : une goroutine par joueur (`PostSyncParallelism = 0`) ;
- `sync/v2/cycle.go` (phase 6) : un match d'escouade est dans `insertedByPlayer` de chaque
  participant ;
- `sync/engine_options.go` : un `PostSyncHook` par moteur, posé au constructeur (voulu : ne pas
  dépendre du câblage) ;
- `sync/convergence.go` `runKillSource`.

**Conception.**
- L'exclusivité vit dans `killcollector` (orchestrateur de l'étape) — ni dans l'orchestrateur de
  cycle v2 (qui ignore les étapes), ni dans le moteur.
- Registre de paquet NON exporté `passesEnCours` (titre → `*sync.Mutex`), de la forme de
  `platform/dblease.leaseMutex`. `TryLock` après les gardes (dépendances, capability
  `film.kill_source`) et AVANT le segment de lecture de l'arriéré ; rendu par `defer`.
- Perdant : rend 0, `observability.AddInt("killsource_postsync_passe_deja_en_cours", 1)`,
  `slog.DebugContext` (ce n'est pas un défaut : la passe en cours traite le même arriéré global,
  trié du plus récent au plus vieux, où les matchs fraîchement insérés sont en tête).
- Aucune constante ne change (`DefaultPostSyncPerCycle`, `PostSyncBudget`, `PostSyncMatchTimeout`).

**Items.**
- [ ] J1.1 Test rouge `TestRunPostSync_UnePasseParTitreALaFois` (`postsync_test.go`, sans DuckDB) :
      deux appels concurrents, capability réelle (`racineDepot`), le `WithRead` du premier bloque
      sur un canal ; le second rend 0 SANS appeler `WithRead`, compteur +1 ; après libération, un
      troisième appel ouvre son segment.
- [ ] J1.2 Tests `TestRunPostSync_VerrouRenduEntreDeuxPasses` (deux appels successifs passent) et
      `TestPassesEnCours_TitresIndependants` (deux clés ne se bloquent pas).
- [ ] J1.3 Implémentation (registre, `TryLock`, compteur, journal).
- [ ] J1.4 Mutation : retirer le `TryLock` → J1.1 rouge.
- [ ] J1.5 Documentation : en-têtes de `collector.go` et `postsync.go` (le mécanisme qui tient la
      série), commentaire de `DefaultPostSyncPerCycle`, liste des compteurs expvar si elle existe
      (ADR 0009).
- [ ] J1.6 Revue adversariale.

**Gate.** G-unit (`./internal/sync/killcollector/ ./internal/sync/v2/...`) ;
`go test -race -gcflags=all=-d=checkptr=0 -run "TestRunPostSync_UnePasse|TestPassesEnCours" ./internal/sync/killcollector/ -count=1` (CGO) ;
G-integ (`./internal/sync/killcollector/ ./internal/sync/`) ; G-vet ; G-arch ; G-lint ; G-CI ;
G-push → fusion dans `feat/v75` sur accord ; message à l'utilisateur : OPS-3 levé pour la fusion
v7.5 → main.

**Révisions** : aucune. **Taille** : S.

---

### J2 — Robustesse des entrées/sorties et des lecteurs (aucune sortie de décodage ne bouge)

**Constats.** SRC-2/OPS-4 (P1), OPS-1, OPS-2, OPS-5, RA1-5, RA1-7, RB1-4, GA1-1/GB-2, CONV-1.
**Prérequis.** P-1 (`filmcache/write.go`, `replay/filmfacts_flux.go`, `replay/geometry.go`,
`grammar/event_list.go` ont bougé dans la campagne rr).
**Révisions.** Aucune montée ; l'empreinte de `grammar` est régénérée à révision constante pour
J2.9 (preuve G-equiv 0).
**Découpage.** Exécutant J2-a : J2.1 à J2.6 ; exécutant J2-b : J2.7 à J2.13.

#### J2.1 Écriture atomique stricte (`platform/atomicfile`) — DT-3
- [ ] Tests rouges : `TestWriteFileStrict_Atomique`, `TestWriteFileStrict_RenameRefuseRendUneErreur`
      (injection `renameFile` → EBUSY : erreur, cible intacte, aucun temporaire résiduel),
      `TestWriteFileStrict_TemporaireImpossibleRendUneErreur` (injection `createTemp`),
      `TestWriteFileStrict_CibleExistanteRemplaceeEntiere`.
- [ ] `WriteFileStrict` (temporaire du même dossier, `finalizeTemp`, rename ; toute panne = erreur) ;
      `WriteFile` = `WriteFileStrict` puis repli in-place sur `isBusy` / temporaire impossible —
      aucune copie de `finalizeTemp`. L'en-tête du paquet renvoie vers `WriteFileStrict` pour les
      pertes irréversibles.
- [ ] Tests existants de `WriteFile` verts, inchangés.

#### J2.2 Écrivain du cache de films (`film/filmcache/write.go`) — SRC-2/OPS-4
- [ ] Tests rouges (`write_test.go`) : `TestWrite_ChunkTronqueSurDisqueEstRemplace`,
      `TestWrite_ChunkIdentiqueNestPasReecrit`, `TestWrite_EchecDEcritureNeLaissePasDeChunkPartiel`,
      `TestWrite_ManifestePorteLesTailles`, `TestWrite_ManifesteIllisibleEstRepareParUneListeFinalisee`,
      `TestWrite_ManifesteIllisibleEtListeNonFinaliseeRefuse`.
- [ ] Chunks par `atomicfile.WriteFileStrict` ; un chunk présent n'est adopté que si sa taille vaut
      `len(c.Data)`, sinon il est remplacé (WARN + compteur `film_cache_chunk_remplace`) ; champ
      `size_bytes` (omitempty) au manifeste écrit ; manifeste illisible + liste finalisée → réécrit
      atomiquement (WARN + `film_cache_manifeste_repare`). Les règles L3 (finalisation,
      sur-ensemble exact, `ErrManifesteDivergent`) sont inchangées.

#### J2.3 Lecteurs du cache : un seul lieu pour la disposition et la validation
- [ ] Tests rouges : `TestSourceChunk_TailleDuManifesteDiffereRendErrChunkTronque` (`filmcache`),
      `TestLocalFilmCacheLoadChunk_MemeValidationQueFilmcache` (`sync/haloclient`),
      `TestRemoteFilms_ChunkTronqueSurDisqueRetelecharge` (`sync/killcollector`).
- [ ] `filmcache.ErrChunkTronque` (index, attendu, lu) ; validation dans `Source.Chunk` /
      `LoadFilm` quand le manifeste porte la taille ; `haloclient.LocalFilmCache.LoadChunk` délègue
      à `filmcache` (suppression de sa copie de `chunk_%02d.bin`) ; `RemoteFilms` traite
      `ErrChunkTronque` comme « absent du disque » (réseau, puis réécriture par J2.2).
- [ ] Ratchet : `archlint/no_hardcoded_film_cache_dirs_test.go` étendu au nom de fichier de chunk ;
      migrations : `cmd/fetch_film_chunks` écrit par `filmcache.Write` (un seul écrivain du cache),
      `cmd/rdata_weapon_scan` lit par un helper exporté de `filmcache` ; mutation jouée.

#### J2.4 Téléchargement (`sync/haloclient/halo_client_film.go`)
- [ ] Test rouge `TestFetchFilmChunks_TailleAnnonceeDiffereRendErrChunkIncomplet` (`httptest` qui
      sert moins d'octets que `ChunkSize`).
- [ ] Comparaison `len(data)` / `ChunkSize` (si > 0) → `ErrChunkIncomplet` typée ; rien n'est
      écrit ; compteur `film_fetch_chunk_incomplet` ; reprise au cycle suivant, aucun marqueur
      terminal.

#### J2.5 `levelup archive-films`
- [ ] Tests : `TestFilmDejaEnCache_ManifestePartielNestPasComplet`,
      `TestFilmDejaEnCache_ChunkTronqueNestPasComplet`.
- [ ] Contexte par `contexteDArret()` (helper existant de `cmd/levelup`) au lieu de
      `context.Background()` ; `filmDejaEnCache` juge par `filmcache.Finalise` et les tailles, plus
      par l'existence du manifeste (rr §8.22, même défaut « présent = complet »).

#### J2.6 Verrou solo par verrou OS (OPS-1) — DT-2
- [ ] Tests rouges (`platform/filelock`) : `TestTryLock_SecondDetenteurRefuse`,
      `TestTryLock_LibereALaMortDuProcessus` (processus enfant de test, modèle
      `filmproc/runner_child_test.go`), `TestUnlock_Idempotent`.
- [ ] Tests (`filmproc/solo_test.go`) : `TestAcquireSolo_ReleaseNeLibereQueSonVerrou` (rouge
      aujourd'hui : `Release` supprime le fichier d'un autre détenteur),
      `TestAcquireSolo_RefusNommeLeDernierDetenteur`, `TestAcquireSoloWait_AttendPuisPrend`.
- [ ] `platform/filelock` : `TryLock(path) (*Lock, error)`, `ErrLocked`, `(*Lock).Unlock` ;
      `filelock_windows.go` (`windows.LockFileEx`, `LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY`),
      `filelock_unix.go` (`unix.Flock`, `LOCK_EX|LOCK_NB`).
- [ ] `filmproc.AcquireSolo` tient le verrou OS ; le détenteur est décrit dans
      `film_decode.lock.json` (écrit par `WriteFileStrict`, `StartedAt` posé une seule fois) ;
      SUPPRESSION de `beat`, `soloStealIfDead`, `soloHeartbeat`, `soloStale` et de leurs tests
      (règle 7) ; en-tête du fichier réécrit. Chemin sous `PathResolver.CacheRootDir()` inchangé.
- [ ] `archlint` : `TestPointsDEntreeDeDecodageArmentUneSentinelle`, `TestNoUnboundedFilmLoop`
      verts.

#### J2.7 Fichier de faits : décodeur borné (RA1-5)
- [ ] Test rouge `TestDecodeFilmFactsFile_CompteImpossibleRendUneErreur` (compte 2^40 : erreur,
      pas de panique) ; cible `FuzzDecodeFilmFactsFile` (graines : fichiers de faits des
      mini-bobines, sous `testdata/fuzz/FuzzDecodeFilmFactsFile/`).
- [ ] Les lectures `int(r.u())` qui dimensionnent une allocation (34 sur `feat/rr-m8`, 11 passent
      déjà par `compte`) passent toutes par `greader.compte(coutMinimal)`.
- [ ] Ratchet : aucune `make(` dimensionnée par `int(r.u())` dans `replay/filmfacts*.go` ; mutation.
- [ ] Équivalence S8 (tests existants des faits) verte.

#### J2.8 Lecteur Bond `.mvar` borné (RB1-4)
- [ ] Test rouge `TestParse_CompteDeConteneurImpossibleRendUneErreur` ; cible `FuzzMapvarParse`
      (graines : `.mvar` versionnés du dépôt).
- [ ] Comptes de conteneur, de tableau et de chaîne bornés par les octets restants
      (`replay/mapvar/cb2.go`, `readContainerHeader` et ses appelants) ; erreur typée.

#### J2.9 Lecteurs de références d'événement bornés (GA1-1/GB-2)
- [ ] Graines ajoutées au harnais `FuzzFilmRecordReaders` pour `grammar/event_list.go` et
      `grammar/equipment_spawn_events.go` ; test rouge sur payload tronqué.
- [ ] Bornes explicites ; empreinte `grammar` régénérée à révision constante (G-equiv 0 cité).

#### J2.10 `LoadGeometry` (RA1-7)
- [ ] Test rouge `TestLoadGeometry_LigneCSVInvalideRendUneErreurNommee`.
- [ ] Erreur typée (ligne, colonne) remontée ; l'appelant journalise (WARN) AVANT de dégrader.

#### J2.11 `replaybuild/zones.go` (CONV-1)
- [ ] Test rouge : table absente enveloppée par `%w` → branche « table absente » (DEBUG), pas WARN.
- [ ] `errors.Is(err, fs.ErrNotExist)`.

#### J2.12 Raison d'un refus de l'enfant de cuisson (OPS-5) — DT-5
- [ ] Tests rouges : `TestEnfant_RaisonParRefusType` (table : `replaybuild.ErrMapNotInCatalog`,
      `replaybuild.ErrUnknownFilmKey`, `filmcache.ErrFilmNonFinalise`, enveloppées → jeton),
      `TestSpawn_JetonVersErreurTypee`, `TestReplayArtifacts_FilmNonFinaliseReporteSansEchec`.
- [ ] `filmproc.EmitRaison` / `parseRaison` (même protocole que `EmitPeak`), `Result.Raison` ;
      l'enfant classe par `errors.Is` ; `replaychild` mappe le jeton vers l'erreur typée
      enveloppée ; `sync/replayartifacts` classe par `errors.Is`.
- [ ] Suppression de tout classement par `strings.Contains(err.Error(), …)` dans `replaychild` et
      `sync/replayartifacts` + ratchet (grep) ; mutation.

#### J2.13 Action admin « construire le rejeu » (OPS-2)
- [ ] Identifier sur pièces le « septième point d'entrée » de l'écart ADR (§2.4) ; s'il n'est pas
      cette action, l'ajouter à ce lot et le signaler.
- [ ] Test rouge `TestRunReplayBuild_PasseParLaStrategieEnfant` (stratégie injectée : l'action
      n'appelle jamais `replaybuild.NewBuilder` dans le processus serveur).
- [ ] L'action réutilise le chemin de l'étape 1.58 (`replayartifacts` : `SpawnBuildOne` puis
      `replaybuild.StoreArtifact`, exposés par UNE fonction exportée du paquet — pas de copie) ;
      `replayBuildMu` conservé ; les commentaires qui invoquent un verrou supprimé sont corrigés.
- [ ] Ratchet : aucun appel à `replaybuild.NewBuilder` / `BuildMatch` / `BuildBytes` sous
      `internal/api/` ; mutation.

**Gate J2.** G-unit (`platform/atomicfile`, `platform/filelock`, `filmproc`, `film/filmcache`,
`sync/haloclient`, `sync/killcollector`, `replaychild`, `sync/replayartifacts`, `replaybuild`,
`film/replay/...`, `film/internal/grammar`, `api/wire`, `cmd/levelup`) ; G-arch ; G-vet ; G-lint ;
G-integ (`sync/killcollector`, `sync/replayartifacts`, `sync/haloclient`) ; G-film ; **G-equiv
0 divergence** ; G-CI ; G-push → fusion dans `feat/v75` sur accord.
**Taille** : M-L.

---

### J3 — Modèle de révision et fraîcheur des faits (DU-2)

**Constats.** SRC-1, RA1-1 (P1), RA1-2, RA1-4 ; faiblesse 1 (RA1-3 → J4 ; RA1-6 → `[~]` rr M8).
**Pièces vérifiées.** `film/revision/{empreinte.go, porte.go, equivalence_test.go}` (quatre
couches, octets bruts commentaires compris, racines déclarées à la main) ; `film/internal/*/rev.go`
(`facts.Rev` = `"killsource-…"` couvre tout `facts/`, dont `objectives/` et `fallback/`) ;
`replay/filmfacts_fichier.go` `Utilisable` (codec, schéma, quatre révisions,
`verifierCleDeCuisson(module, AxisW, …)`) ; `replay/film_scan.go` `balayerCalquesGardes` (gardes
`opt.Flag`, `opt.Zones`, `opt.Bomb`) et `balayerPont` (`opt.RosterXUIDs`) ;
`replaybuild/filmfacts_cuisson.go` (faits écrits même quand l'artefact est refusé) ;
`replay/filmfacts_decode.go` (inventaire nil → tranche vide).

#### J3.1 Empreinte insensible aux commentaires (outillage)
- [ ] Tests rouges (`revision/empreinte_test.go`) : `TestCalculer_CommentaireSansEffet` (deux
      arbres de fixture qui ne diffèrent que par des commentaires),
      `TestCalculer_UnJetonChangeLEmpreinte`, `TestCalculer_ChaineContenantDeuxBarres`,
      `TestCalculer_DirectiveGoCompte` (`//go:build`, `//go:embed` changent l'empreinte).
- [ ] `revision.Calculer` hache le flux de jetons `go/scanner` hors commentaires, directives
      `//go:` conservées ; cadre inchangé (chemin relatif à la racine + longueur).
- [ ] Les quatre goldens régénérés à révision CONSTANTE dans le même commit ; le commit ne touche
      aucun fichier des quatre couches (`git diff --stat` limité à `film/revision/` et aux
      goldens) : c'est ce qui prouve qu'il s'agit d'outillage.

#### J3.2 Périmètre = fermeture des imports (SRC-1)
- [ ] Tests rouges : `TestPerimetre_FermetureDesImports` (arbre de fixture : A → B → C ; une couche
      révisée entre par sa VALEUR), `TestPerimetreDeChaqueCoucheEgaleSonGolden`
      (`testdata/<couche>_perimetre.golden` : la liste des paquets hachés) ; mutation : un import
      ajouté dans une couche rougit le golden.
- [ ] Fermeture calculée par `go/parser` (`ImportsOnly`), bornée au module, arrêtée aux couches
      révisées ; `film/types`, `film/damagetag`, `games/weapons/filmshell` entrent dans les
      périmètres qui les importent ; le second oracle (`equivalence_test.go`) redéclare la même
      règle ; goldens d'empreinte régénérés à révision constante (outillage, aucune sortie ne
      change).

#### J3.3 Une révision par consommateur de faits
- [ ] Mesure écrite au rapport : fermetures de `facts/killsource` et de `facts/objectives`.
- [ ] `killsource.Rev` (garde la valeur actuelle de `facts.Rev`, donc AUCUNE réouverture du
      backlog) et `objectives.Rev` remplacent `facts.Rev` ; le backlog de `killcollector`
      (`decoder_rev`) lit `killsource.Rev` ; `couchesDesCalques` attribue chaque calque à la
      révision précise ; `coverage.decoder` publie les révisions par consommateur ; `Utilisable`
      compare toutes les révisions de couche.
- [ ] Tests : golden de périmètre de `killsource` sans paquet d'objectifs ;
      `TestCalquesNePortentQueLesRevisionsConnues` mis à jour.

#### J3.4 Gardes de l'appelant tracées (RA1-1)
- [ ] Tests rouges : `TestUtilisable_RefuseDesFaitsCuitsSansUneGardeDemandee` (zones faux dans les
      faits, vrai demandé → `ErrFilmFactsGardes`), `TestUtilisable_AccepteUnSurEnsembleDeGardes`,
      `TestUtilisable_RefuseUnRosterDifferent`, `TestCuisson_AucunFaitEcritQuandLArtefactEstRefuse`
      (`replaybuild`).
- [ ] `GardesDeCuisson{Drapeau, Zones, Bombe bool; Roster [32]byte}` dérivées d'`Options` en UN
      point ; inscrites à l'en-tête des faits (montée `VersionCodecFaits`) ;
      `Utilisable(entry, gardes)` ; la cuisson n'écrit pas de faits quand l'artefact est refusé.

#### J3.5 Clé de cuisson complète (RA1-4)
- [ ] Test rouge en table : chaque champ de `MapQuantEntry` qui gouverne le décodage (module,
      `AxisW`, `Region`, `RegionIndexBits`, bornes, porte) changé → `ErrFilmFactsCarte`.
- [ ] `EmpreinteDeCle(entry)` (hachage canonique des champs gouvernants) inscrite à l'en-tête et
      comparée par `verifierCleDeCuisson`.

#### J3.6 Inventaire nil ou vide (RA1-2)
- [ ] Test rouge : aller-retour nil → nil et `[]` → `[]` ; couverture et calque identiques entre
      la branche film et la branche faits sur une fixture sans inventaire.
- [ ] Drapeau de présence au codec (même montée que J3.4).

#### J3.7 ADR 0034 amendé (EN)
- [ ] D-6 et D-7 : empreinte sans commentaires, périmètre par fermeture figé par golden, révisions
      par consommateur, gardes de l'appelant, clé complète ; constats cités.
- [ ] D-3, règle 2 : la règle des largeurs présumées (DU-9) — une largeur trouvée par la fermeture
      est admise pour un composant de taille fixe qu'on ne fait que sauter, marquée « présumée »
      et listée par un test gelé ; l'écrivain du jeu reste la source pour tout le reste.

**Gate J3.** G-unit (`film/revision`, `film/internal/...`, `film/replay/...`, `replaybuild`,
`sync/killcollector`) ; G-arch ; G-integ (`sync/killcollector`) ; G-film ; **G-equiv** :
divergences limitées à `coverage.decoder` et `layers` (déclarées) ; test de refus des faits de
l'ancien codec ; G-web (montée de schéma) ; G-CI. Revue adversariale.
**Révisions.** Toutes à constante (outillage) ; `SchemaVersion` +1 ; codec des faits +1 → faits du
parc à re-décoder en J11. **Taille** : M.

---

### J4 — Étage de balayage unique (DU-3 = S : S1 obligatoire, S2 sous condition)

**Constats.** RA1-3 ; faiblesses 3 (lectures artisanales) et 5 (seconde séquence) ; écarts ADR.
**Pièces mesurées (`feat/rr-m4b`, 2026-09-25).**
- Cinq fichiers de lecture pure dans `film/replay` : `deaths_source.go` (146 L), `player_index.go`
  (165 L), `origin.go` (147 L), `film_player_table.go` (210 L), `inventory_decode.go` (393 L, porte
  le lecteur privé `invBitAt`/`invBits`) — ~1 060 lignes ; tests associés ~740 lignes. Dans
  `replay`, seul `film_scan.go` les appelle.
- Types de résultat : `Death`, `PlayerIndexTable`, `FilmPlayerTable`, `KeyframeInventory` —
  ~146 références (dont 23 hors de `replay`).
- Appels hors de `film/` : `sync/killcollector/positions.go` (`ScanClockOrigin`, `ScanDeaths`,
  `ScanPlayerIndices`), `replaybuild/filmload.go` (`ScanDeaths`), quatre outils `cmd/`
  (`diag_deaths`, `oddball-terrain`, `statnames-sweep`, `zone-attribution`) + des tests.
- `sync/killcollector/positions.go` `buildPositionRows` : seconde séquence (positions SANS les
  exemptions de translocation que la cuisson applique, puis horloge, morts, index, créations).
- Lecteurs de bits artisanaux hors `source` : `readBitsAt`, `PeekBits`, `kfReadBits`,
  `kfReadBitsLoop`, `kfBitAt` (dans `grammar`), `invBitAt`, `invBits` (dans `replay`). Le ratchet
  `no_raw_film_bytes_outside_source_test.go` les repère par NOM (`bitAt`, `bitsN`…) : aucun de ces
  sept noms n'y figure.

#### J4.0 — Carte de fermeture (DU-8 a, DT-12) : un instrument, aucune sortie de production ne bouge

**But.** Mesurer, build par build, ce que la grammaire sait TRAVERSER dans les trames delta — pas
ce qu'elle interprète — et nommer ce qui bloque. Elle donne la référence avant les jalons de
grammaire (J4.6, J5, J6, J10), l'ordre de portage le plus rentable, et le déclencheur du chantier
de représentation intermédiaire (§1.4).
**Pièces (`feat/rr-m4b`).** `grammar/keyframe_closure.go` (`KeyframeClosure`,
`KeyframeClosureStat{Closed, Total, Blocking}`, bloquant = composant non porté le PLUS FRÉQUENT) avec
`keyframe_closure_ratchet_test.go` et `testdata/keyframe_closure.golden` (historique des
régénérations en tête) ; trame delta = préambule R(1) puis trois vues : messages
(`frame_vue_messages.go`), entités, contrôle (`frame_vue_controle.go` : `LectureVueC`,
`vueCFermee`, causes `ArretVueC`) ; `DecodeFrameViewsCurseur` et `EntityTrace.DesyncAt` (premier
composant présent sans lecteur) ; `testdata/ecs_table.tsv` (statuts `porte`, `non_porte`,
`partiel`, `deser_non_cable`, colonne d'usage produit) ; huit mini-bobines, une par build
(`film/replay/testdata/minifilm_*`) ; modèle d'outil `film/research/cmd_grenadeids`.

- [ ] J4.0.1 Tests rouges synthétiques (`grammar/frame_closure_test.go`) :
      `TestFrameClosure_PaquetFermeAuBitPres` (trois vues lues jusqu'à leur terminateur, curseur sur
      la fin du paquet), `TestFrameClosure_ComposantSansLecteurNommeLeBloquant` (un delta dont le
      masque annonce un composant non porté : vue des entités non fermée, bloquant
      `ti=<a> i<idx> <nom>`), `TestFrameClosure_ArretDeLaVueDeControleCompteParCause` (chaque
      `ArretVueC`), `TestFrameClosure_BloquantLePlusFrequentDepartageParNom` (règle de
      `KeyframeClosure`).
- [ ] J4.0.2 `grammar.FrameClosure(fc) (FrameClosureReport, error)` : par vue — paquets atteints,
      fermés au bit près, causes d'arrêt ; par archétype — records NEW et delta, fermés, bloquant
      le plus fréquent ; et **la fermeture des records UTILES** — ceux qui portent un composant à
      usage produit (colonne `product_use` de `ecs_table.tsv`, passée en entrée : `grammar` ne lit
      pas de fichier) et les entrées de la vue de contrôle que le produit lit : c'est la mesure du
      déclencheur de la spec (§9). AUCUNE nouvelle lecture de bits : les marcheurs existants sont
      appelés tels quels ; pure, sans I/O, sans état de paquet (D-5). Ni la cuisson ni le
      collecteur ne l'appellent. Test rouge ajouté à J4.0.1 :
      `TestFrameClosure_RecordUtileFermeCompteAParte`.
- [ ] J4.0.3 Golden `testdata/frame_closure.golden` sur les huit mini-bobines (une ligne par
      (film, vue) et par (film, archétype)) et ratchet `TestFrameClosureRatchet`, rouge sur une
      BAISSE de fermés ; même porte de régénération que le ratchet d'image-clé ; en-tête
      « historique des régénérations » ; mutation : décaler d'un bit un lecteur de composant →
      rouge.
- [ ] J4.0.4 Outil `film/research/cmd_fermeture` (tag `research`) : films lus un à un, en place,
      dans l'ordre donné, sentinelle `filmproc.Arm`, option `-limite` ; sortie = TSV brut +
      résumé Markdown : par build, part des paquets fermés par vue et **part des records utiles
      fermés** (le déclencheur de la spec) ; classement des composants bloquants (archétype, index,
      nom, statut et usage produit lus dans `ecs_table.tsv`, paquets bloqués, gain potentiel =
      records utiles qui se fermeraient si ce seul composant était porté — BORNE SUPÉRIEURE, un
      autre composant peut bloquer derrière) : c'est la liste courte de ce qui mérite Ghidra (ou
      une largeur mesurée présumée pour un composant de taille fixe qu'on ne fait que sauter,
      DU-9). Déclaré aux ratchets de points
      d'entrée s'ils couvrent `film/research`.
- [ ] J4.0.5 Mesure de référence (superviseur, après accord ; témoins de
      `config/replay_corpus.toml`, au moins un film par build, BTB seulement sur accord) →
      `.ai/V7.5/film_re/CARTE_FERMETURE_<date>.md`. Ces chiffres sont la base des deltas de J4.6,
      J5, J6, J10 et J11.3.
- Révisions : aucune montée (aucune sortie ne change) ; empreinte de `grammar` régénérée à
  révision constante (preuve G-equiv 0).

**Gate J4.0.** G-unit (`grammar`), G-arch, `go vet -tags=research ./internal/games/halo_infinite/film/research/cmd_fermeture/`,
G-equiv 0, G-CI.

#### S1 — le déplacement qui corrige les défauts (obligatoire)
- [ ] J4.1 Inventaire écrit (rapport) : chaque balayage et chaque lecture d'octets de film hors
      `source` (fichier, fonction, couche cible selon D-1, consommateurs) ; les chiffres ci-dessus
      re-mesurés sur l'arbre fusionné.
- [ ] J4.2 Déplacement pur des cinq fichiers et de leurs tests vers `grammar` ; les quatre types
      vers `film/types` (renommage mécanique, aucun alias) ; appelants migrés ;
      **G-equiv 0 divergence** ; empreintes régénérées à révision constante. `replay` ne lit plus
      d'octet de film (écart ADR « replay ne décode rien » fermé).
- [ ] J4.3 Test rouge `TestCollecteurEtCuissonAppellentLeMemeEtage` (ratchet :
      `sync/killcollector` n'appelle aucun `replay.Scan*`) et
      `TestPontDIdentite_ExemptionsDeTranslocationAppliquees` ; un étage unique de LECTURES du pont
      d'identité (positions avec exemptions, créations, morts, index, origine d'horloge) dans
      `grammar`, exposé par `decfilm`, appelé par `replay` ET par `killcollector.buildPositionRows`
      (seconde séquence supprimée). Le registre d'identité (`BuildIdentityRegistry`) reste dans
      `replay`.
- [ ] J4.4 `IsolationDecoderRev` monté (le collecteur gagne les exemptions : changement déclaré
      des positions de mort et des faits d'isolement).
- [ ] J4.5 Façade : consommateurs de `LecteurSur` / `Paquets` / `Inflate` hors `film/` listés ;
      ceux de recherche passent sous `film/research` ; ceux de production restent et l'ADR est
      corrigé en conséquence ; ratchet de surface V25 réajusté avec justification datée (entrées
      ajoutées à `decfilm`, identifiants `replay.X` retirés hors de `film/`).

#### S2 — les sept lecteurs de bits artisanaux (condition fixée d'avance)
- [ ] J4.6 Benchmark du balayage bit à bit (`b.Loop`, mini-bobines du dépôt) mesuré AVANT ; les
      sept lecteurs passent au lecteur canonique de `source`, conventions hors bornes rendues
      explicites (ex. : `invBitAt` rend 0 hors du tampon). Deux conditions : `replay-equiv` à zéro
      divergence ET régression du benchmark ≤ 5 %. Condition tenue : ratchet structurel (toute
      fonction qui extrait des bits d'un `[]byte` hors de `source` est rouge, plus seulement par
      nom) avec une allowlist vide. Condition non tenue : arrêt, rapport, les sept inscrits comme
      exceptions datées de ce ratchet structurel (critère de retrait écrit) et ADR D-2 corrigé.

**Gate J4.** G-unit, G-arch, G-integ (`sync/killcollector`), G-film, G-equiv 0 (cuisson),
équivalence killsource sur fixtures (changement déclaré : positions), benchmark J4.6, G-CI.
**Taille** : M-L.

---

### J5 — Identité (slot, génération) typée et GB-1 (P0)

**Constats.** GB-1 (P0), RA2-1, RA2-2, RA2-3, RA2-6, RB2-3, GA1-3 ; faiblesse 2. Couplage
imposé par l'audit : corriger GB-1 seul réveillerait RA2-1/2/3 — tout part dans ce jalon.
**Pièces vérifiées (`feat/rr-m4b`).** `grammar/offline_biped.go` (`ScanFilmOptions.RequireTag1`,
`DefaultScanFilmOptions` à `true`, `matchBipedHeaderRaw` : `readBitsAt(pay, p+14, 2) != 1`) ;
`grammar/delta_biped_walk.go:108` (`true` en dur, marcheur de 8 canaux) ;
`grammar/equipment_recovery.go:234` et `grammar/offline_aim_only.go:161` (copies du filtre ; la
seconde est née dans la campagne rr) ; `replay/build_vehicles.go` (seul à désarmer) ;
`grammar/frame_records.go` `readRecordID` (« the tag is generation ») ;
`types/grammar_equipement.go` `EquipmentLifeKey{Slot, Gen}` (69 références, 18 fichiers).

#### J5.0 Mesure préalable (superviseur, avant tout code)
- [ ] Instrument : test `research` (ou outil du rapport `RAPPORT_PONT_APLATI_2026-09-10.md`,
      réutilisé s'il existe encore) : par film, records de création par génération, vies (slot,
      génération) avec et sans positions, `durationMs` publié contre la durée du match, en-têtes
      valides dont (slot, tag) n'est aucune vie connue (faux positifs potentiels). S'il est écrit,
      il s'ajoute comme mode de `film/research/cmd_fermeture` (J4.0.4) : même boucle de films, même
      sentinelle — pas un second harnais.
- [ ] Films : témoins de `config/replay_corpus.toml` + `084a804d`, `1c4c63c2`, `a349fea8` (BTB),
      un à la fois, après accord de l'utilisateur.
- [ ] Critère de poursuite : `084a804d` reproduit l'ordre de grandeur de l'audit (~122 corps sans
      position sur 379). Sinon : ARRÊT et rapport.
- [ ] Rapport `.ai/V7.5/film_re/MESURE_GB1_<date>.md` ; cibles de J5.5 écrites avant de coder.

#### J5.1 Clé de vie typée (DT-7) — zéro différence
- [ ] Renommage `types.EquipmentLifeKey` → `types.LifeKey` (doc : objet du monde, véhicule, corps
      de bipède) ; aucun alias.
- [ ] Helper unique de lecture du handle (slot + génération) dans `grammar` ; les quatre copies
      l'appellent ; ratchet sur le littéral `p+14, 2)` ; mutation.
- [ ] G-equiv 0 ; empreintes régénérées à révision constante.

#### J5.2 Filtre de génération vivante (GB-1) — DT-8
- [ ] Tests rouges synthétiques (`grammar`) : `TestEnTeteBipede_TagDeLaGenerationVivanteAccepte`,
      `TestEnTeteBipede_TagDUneGenerationMorteRefuse`,
      `TestEnTeteBipede_SlotSansCreationRetombeSurLeRepliNomme` (compté),
      `TestMarcheurDelta_CorpsDeGenerationDeuxVuParLesHuitCanaux`.
- [ ] `ScanFilmOptions.RequireTag1` SUPPRIMÉ ; `GenerationsVivantes` construit depuis les records
      de création et les images-clés, passé au marcheur ; `balayerCreations` avant les positions ;
      véhicules : « toutes générations » explicite ; repli `repli_generation_vivante_inconnue_tag1`
      inscrit au registre, compté ; appelants migrés : `replay/build_from_film.go`, l'étage de
      J4 (collecteur), `grammar/weapon_hit_distance_resolver.go`.

#### J5.3 Canaux chaînés par vie
- [ ] Tests rouges : `TestEquipmentChanges_NouvelleVieNHeritePasDeLaPrecedente`,
      `TestHeldWeaponChanges_ChaineParVie`.
- [ ] `grammar/equipment_changes.go` et `grammar/held_weapon_changes.go` chaînent par `LifeKey`.

#### J5.4 Registre d'identité par vie
- [ ] RA2-1 : test rouge `TestRegistre_RecordDeCreationNOuvreQueSaVie` (slot recyclé) → registre
      par `LifeKey` (`replay/identity_registry_creation.go`).
- [ ] RA2-2 : test rouge `TestViesSansNom_NeFranchissentPasUneFrontiereDeCorps` → nommage par
      occupation borné à la vie (`replay/unnamed_lives.go`, `identity.go`) ; ce qui reste = repli
      nommé, inscrit, compté.
- [ ] RA2-3 : test rouge `TestTirsEtLancers_AttribuesALaVieALInstant` → attribution par (slot,
      génération à l'instant) par le registre, même abstention que les équipes
      (`TeamCoverage.TracksSlotAmbiguous`, `IdentityRegistry.PontDeSlot`) ; le pont aplati
      slot → premier occupant disparaît de `shots.go`, `grenades.go`, `coverage_decoder.go`.
      **Re-mesurer à l'entrée du lot** : la campagne rr a changé le rattachement des tirs
      (`tirParLUnite`, référence 0 de l'unité tireuse ; `tirs_par_place.go` ; repli
      `repli_index_de_tireur_hors_place` de la reprise M4b). L'item porte sur ce qui reste du pont
      aplati (lancers de `grenades.go`, `coverage_decoder.go`, chemin par `owner` de `slotFor`) ;
      le périmètre re-mesuré est écrit au rapport AVANT de coder.
- [ ] RA2-6 : test rouge `TestNameTracksByLives_VieDUnSeulEchantillonNommeeParSaVie` (règle
      produit : aucune vie anonyme) → recouvrement à bornes incluses.
- [ ] RB2-3 : test rouge `TestVieDUneCle_BorneeParLaVieSuivanteDuSlot`
      (`replay/ground_weapon_objects.go`, `equipment_placement_ends.go`).
- [ ] GA1-3 : test rouge `TestEtatsDeMouvement_RecordNEWPublieSousSaPropreVie`
      (`grammar/frame_infer.go`) ; la note D13 « NON TRAITÉ » est levée.

#### J5.5 Montées et gate de comportement
- [ ] `grammar.Rev`, `SchemaVersion`, `IsolationDecoderRev` (+ chroniques).
- [ ] G-corpus, changements déclarés d'avance : BTB → vies de génération ≥ 2 positionnées,
      `durationMs` corrigé, positions de mort et distances gagnées ; témoins sans slot recyclé :
      zéro changement ; zéro perte.

**Gate J5.** G-unit, G-arch, G-integ (`sync/killcollector`), G-film, G-corpus, G-web, G-CI.
Revue adversariale. **Taille** : L.

---

### J6 — Un seul portage par fonction du jeu (DU-1 = oui)

**Constats.** GA2-2 (P1), GA2-3, GA2-4, GA2-5 ; faiblesse 3 (portages). GA2-1 : `[~]` reprise M4b
de la campagne rr (contrôle P-3).
**Pièces.** `grammar/components_flock.go:56-62` (appel `dispatch_biped.go:82-84`, niveau 0 au lieu
de 0x10) et asset-transform (`dispatch_biped.go:236-251`, niveau 30) ; `DAT_144632be0` câblé à 1
(`unit_control.go`, `default_state.go`, `components_position_i0.go`) ; `default_state_arch.go`
(`FUN_140ce59bc`) ; `dispatch_object.go:158-163` (largeurs de carte avec `idx = -1`) ;
`traverse.go:180-192`, `components_position_i0.go`, `transloc_events.go:207-230` (validé 18/18) ;
`profile/loi_largeurs.go` (« le NIVEAU est un IMMÉDIAT du site d'appel »).
**Apport de la reprise M4b (en cours le 2026-09-25, non commitée, à relire sur l'arbre fusionné).**
Les deux positions du corps d'i54 (`consumeMobilityActionBody`) sont lues par
`consumeSimStateHandleTail` au niveau 0x10, largeurs absolues de la carte ; `consumeE494Position`
est supprimé. La reprise laisse volontairement le site `ti=38 i18` (transformations d'un corps
rigide, même lecteur au niveau 0x10, lu dans l'état complet d'une image-clé en R(96)) et le
consigne à son rapport : ce site entre dans J6.

- [ ] J6.1 Relevés Ghidra (superviseur, lecture seule) : immédiats de niveau de chaque site d'appel
      de `FUN_14076e524` (dont `ti=38 i18` et les deux sites d'i54 déjà repris), valeur et
      références de `DAT_144632be0`, `FUN_140ce59bc` et son jumeau ; chaque valeur citée avec
      fonction, adresse et date (D-3 règle 1).
- [ ] J6.2 Tests rouges par site d'appel : flux de bits synthétique construit selon l'écrivain du
      jeu ; `transloc_events` (18/18) reste l'oracle inchangé.
- [ ] J6.3 Un seul portage paramétré par l'immédiat de niveau, qui PART de
      `consumeSimStateHandleTail` (déjà employé par la reprise M4b pour i54) si J6.1 le confirme —
      pas de second portage ; tous les sites l'appellent ; `DAT_144632be0` lu du profil ou constante
      prouvée ; GA2-4 tranché par J6.1 ; GA2-5 aligné sur `absAxisWFor` ; `ti=38 i18` traité avec la
      fermeture d'image-clé `ti=38` du golden comme garde (aucune baisse) ; test en table des sites
      et de leurs immédiats (ratchet).
- [ ] J6.4 `grammar.Rev` monté ; G-corpus : rejets en baisse (`bfecd02b`), `ti=21` lu en entier
      sur les cinq bobines, fermeture d'image-clé sans baisse, zéro perte ; carte de fermeture
      (J4.0) rejouée : delta par vue et par archétype contre la référence, golden
      `frame_closure.golden` régénéré en hausse seulement.

**Gate J6.** G-unit (`grammar`), G-arch, G-film, G-corpus, G-CI. **Taille** : M.

---

### J7 — killsource

**Constats.** FK-1, FK-2 (P1), FK-3 à FK-7.
**Pièces (`feat/rr-m4b`).** `killsource/roster.go` (`buildRoster` : `nHumans = len(kf.names)` ;
`pinBots` : `b.Slot < r.nHumans` → désépinglé ; remplissage `"?%d"` ; ligne ~401 :
`== "?" || HasPrefix(…, "?")`) ; `killsource/assist.go:406` (ne rejette que `"?"`) ;
`replayidentity/bot_identities.go` (retrait silencieux) ; `killsource/hybrid.go` `runBots` (écrit
`p.byTime[…]` sans vérifier l'instant, contrairement à `runSelfSource`) ; `killsource/decode.go`
`ProfilDeDepartPourCarte` (rend « largeurs changées », faux sur Cliffhanger dont l'entrée EST
l'invariant).

- [ ] J7.1 FK-2 : prédicat unique `estNomDeRemplissage` (DT-6) employé par `assist.go`, le site de
      la ligne ~401 et tout nom publié (tueur, victime) ; test rouge
      `TestAssist_NomDeRemplissageNestJamaisPublie` ; test d'intégration
      `TestCollecteur_NEcritJamaisUnNomDeRemplissage` ; ratchet (aucune comparaison à `"?"` hors du
      prédicat dans `killsource`) ; mutation.
- [ ] J7.2 FK-1 : la borne de l'espace des humains vient du film (sièges de la table, capacité des
      équipes — règle produit : places finies, un partant libère sa place), jamais du nombre de
      noms du kill-feed ; bots non épinglés et retraits de `bot_identities.go` journalisés et
      comptés (`killsource_bots_non_epingles`), publiés en couverture ; tests rouges
      `TestRoster_RemplacantHumainNeDesepinglePasLeBotDeRelais`,
      `TestRoster_BotNonEpingleEstCompte` ; le témoin `index_motif_test.go` gagne le cas « bot dans
      l'espace des humains ».
- [ ] J7.3 FK-3 : le motif du xuid cherche aussi les joueurs qui tuent sans mourir
      (`index_motif.go`, `feed.go`) ; test rouge.
- [ ] J7.4 FK-4 : le temps 4 ne réécrit jamais un instant publié (collision comptée) ; invariant
      `Covered ≤ RealPairs` tenu et testé (`match.go`) ; tests rouges.
- [ ] J7.5 FK-5 : numérateur de santé sans double comptage (`hybrid.go` contre le contrat de
      `killhealth.go`) ; test rouge.
- [ ] J7.6 FK-6 : dédoublonnage des records par le champ `bit` prévu pour cela
      (`feed_couples.go`, `assist.go`) ; test rouge (records dupliqués → aucun couple fabriqué).
- [ ] J7.7 FK-7 : `ProfilDeDepartPourCarte` rend « carte appliquée » quand l'entrée fournit ses
      largeurs ; test rouge `TestProfilDeDepartPourCarte_CarteEgaleALInvariantEstAppliquee`.
- [ ] J7.8 `killsource.Rev` monté (+ chronique) → backlog rouvert, traité en J11.

**Gate J7.** G-unit (`killsource`, `replayidentity`), G-arch, G-integ (`sync/killcollector`,
`persist`), tests `KILLSOURCE_FIXTURES` en local (superviseur), G-corpus (changements déclarés :
`?N` disparus, bots épinglés), G-CI. Revue adversariale. **Taille** : M.

---

### J8 — Replis D-10 : conversions et câblage des 81 compteurs

**Constats.** GA1-2 (P1), RB2-5 (P1), FO-1/RA2-4 (P1), RB2-8, FO-3, FO-4 ; faiblesse 4.
**Pièces.** `facts/fallback/` (registre à `9d335ea43` : 99 entrées, 18 `CompteurBranche: true`,
81 `false`, dont 46 ciblent `comptageFamille19` ; `noms.go` : constantes des seuls replis câblés).
La campagne rr y ajoute des entrées câblées, dont deux par la reprise M4b en cours
(`repli_physique_de_type_de_vehicule_supposee`, `repli_index_de_tireur_hors_place`) : les
comptes se RE-MESURENT à l'entrée du jalon ;
`grammar/world.go` (`LierParAnticipation`, `AnticipationsParArchetype` sans appelant) ;
`replay/identity_registry_section.go` `methodeStatborg` (`residu_de_manche` → `MethodNone`) ;
`games/canonical/film_identity.go` (`LinkMethod`).

- [ ] J8.1 GA1-2 : entrée `repli_liaison_par_anticipation` (constante, fait, mécanisme, condition,
      ordre, site, date, cible, critère) ; identifiants renommés pour porter « Repli » (le ratchet
      de vocabulaire les voit) ; compte remonté par `AnticipationsParArchetype` jusqu'au
      `fallback.Compteur` de la cuisson (`DeclencheN`) ; test rouge sur un `World` synthétique.
      La reprise M4b donne à la table anticipée un second usage (`SlotDeLArchetype` : bande d'un
      archétype pour le début de liste de `debut_de_liste.go`, PROUVÉ par la chaîne de records qui
      finit au bit près) : c'est une lecture, pas un repli ; seul `LierParAnticipation` s'inscrit,
      et le rapport le justifie.
- [ ] J8.2 RB2-5 : un ramassage natif date au plus une occupation (ensemble consommé) ; le reste
      s'abstient et se compte ; plus de double crédit dans `usage_summary.go` ; tests rouges ;
      golden `assembly_a521164d` : changement déclaré.
- [ ] J8.3 RB2-8 : fenêtre 200 ms / 1,5 m de l'origine `dropped` inscrite comme repli nommé et
      comptée (critère de retrait : conversion le jour où une lecture est trouvée) ; test.
- [ ] J8.4 FO-1/RA2-4 : `canonical.MethodRoundResidue` (`"residu_de_manche"`) et son mapping ;
      test d'exhaustivité (chaque `objectives.Origin*` a une voie canonique non vide) ; entrée de
      repli de la voie, comptée ; contrat Go/web vérifié (union TS des méthodes si elle existe).
- [ ] J8.5 FO-3 : le pont par instants de mort lit le compteur avec les MÊMES filtres que la série
      publiée (une fonction de filtre partagée) ; test rouge.
- [ ] J8.6 FO-4 : site manquant de `repli_emission_du_compteur_de_morts_jetee` ajouté ; branche
      `len(kept) == 0` supprimée ou prouvée atteignable par un test.
- [ ] J8.7 Câblage (sous-lots par famille ; comptes de l'audit, à re-mesurer : `grammar` 11,
      `killsource` 29, `objectives` 20, `replay` 21) : constante + `Declenche` au site + `CompteurBranche: true` + `CibleComptage`
      vidée ; entrée sans code vivant retirée ; hors cuisson (collecteur), le compte voyage dans
      le résultat et se publie en expvar par nom + journal par film ; `CibleRetrait` des entrées
      qui nomment un jalon clos réécrite selon la règle 4 (« retrait au jalon suivant si compte
      nul au corpus gate de J11 », datée) ; ratchet `TestChaqueRepliEstCompte` (0 entrée à
      `false`) ; `no_stale_fallback_target_test.go` étendu aux cibles symboliques.
- [ ] J8.8 Montées : `SchemaVersion` (contenu de `coverage.fallbacks`) ; empreintes des couches
      régénérées à révision constante quand les sorties de décodage sont inchangées (preuve
      G-equiv hors `coverage.fallbacks` + équivalence killsource sur les lignes).

**Gate J8.** G-unit, G-arch, G-integ (`sync/killcollector`), G-film, G-corpus (changements
déclarés : `coverage.fallbacks`, `usage`, provenance), G-web, G-CI. **Taille** : L.

---

### J9 — Drapeau, bombe, zones

**Constats.** RB1-1 (P1), RB1-2, RB1-3, RB1-5, RB1-6, RB1-7, RB1-8.
**Pièce clé (`feat/rr-m4b`).** `replay/flag_carries.go` `closeByCarrierKills` : seuls sont
exclus les portages dont le xuid est celui du tueur ; un tueur non identifié n'exclut personne.

- [ ] J9.1 RB1-1 : candidat = porteur d'une équipe ADVERSE à celle du tueur (équipe du balayage
      passée au contexte) ; tueur non identifié → aucune fermeture, compteur ; tests rouges en
      table.
- [ ] J9.2 RB1-5 : hors catalogue, le nombre de drapeaux ne se suppose pas (`flag_carries_handoff.go`) :
      lu du film, sinon repli nommé sans fermeture par passage ; test rouge.
- [ ] J9.3 RB1-6 : fermoirs indexés par (manche, slot) ; test rouge (deux manches, même slot).
- [ ] J9.4 RB1-2 : ordre total (instant, lâcher avant reprise) dans `flag_carries_lives.go` ; test.
- [ ] J9.5 RB1-7 : portage de bombe fermé au premier de (mort, lâcher) (`held_object_carry.go`) ;
      test.
- [ ] J9.6 RB1-3 : origine d'horloge illisible → `bomb_arms` ABSENT (NULL), jamais zéro mesuré
      (« absent n'est pas zéro ») ; si la colonne interdit NULL : migration (`internal/migration/`,
      skill `db-schema`) et écriture par le persister INSERT-only ; tests (dont G-integ `persist`).
- [ ] J9.7 RB1-8 : descente de jauge publiée dans la seconde (`zone_states_gauge.go`) ; test.
- [ ] J9.8 `SchemaVersion` monté.

**Gate J9.** G-unit, G-arch, G-integ (si J9.6 touche `persist`/`migration`), G-film, G-corpus
(changements déclarés : portages, bombe), G-web, G-CI. **Taille** : M.

---

### J10 — Déterminisme et correctifs résiduels

- [ ] J10.1 Tri total (GB-3, RA2-5, faiblesse 9 — DT-9) : les 43 `sort.Slice` du périmètre relus un
      par un (clé unique prouvée, ou départage total ajouté) ; `equipmentChanges[]` et `buildRoster`
      reçoivent un ordre total ; tests rouges `TestEquipmentChanges_OrdreStableSurExAequo` et
      `TestBuildRoster_DeuxBotsDuMemeIndex` (50 exécutions, sortie identique) ; ratchet
      `archlint/film_tri_total_test.go` ; G-equiv exécuté DEUX fois (sorties identiques entre les
      deux passes).
- [ ] J10.2 GB-4 : `_, _ = DecodeFrameRecords(…)` et le calcul mort de `weapon_hits.go` supprimés
      avec les tests qui ne tenaient qu'eux (règle 7).
- [ ] J10.3 RB2-6 : tolérance de siège exprimée en `time.Duration` (plus de division d'une durée en
      ms par un pas en µs) ; test rouge.
- [ ] J10.4 RB2-7 : le lien prise → arme au sol respecte `LowUS` ; test rouge.
- [ ] J10.5 GA1-4 : règle de coïncidence de la LIS des datums alignée sur la grammaire, `ambigus`
      mesure ce que son commentaire dit ; test rouge.
- [ ] J10.6 GA1-5 : recul sur les vacants de tête non neutralisé par le bourrage ; test rouge.
- [ ] J10.7 `grammar.Rev` et `SchemaVersion` montés.

**Gate J10.** G-unit, G-arch, G-film, G-equiv (x2), G-corpus (changements déclarés : ordres sur
ex æquo, sièges, armes au sol), carte de fermeture rejouée pour J10.5 et J10.6 (aucune baisse),
G-web, G-CI. **Taille** : M.

---

### J11 — Vague unique : gates de corpus, re-décodage, backfills, fusion (GO opérationnel)

- [ ] J11.0 J3 à J10 clos (J6 statué) ; `git merge feat/v75` dans la branche ; G-film, G-arch,
      G-integ complets verts ; G-CI.
- [ ] J11.1 G-corpus complet (`--reference=base`, base = tête de `feat/v75`) : zéro perte non
      expliquée ; changements = union des déclarations J3-J10 (table au §9) ; rapport JSON archivé.
- [ ] J11.2 `replay-equiv` : références re-figées à la tête (`-update`), digests commités.
- [ ] J11.3 Mesures de clôture : GB-1 (vies sans positions par film, avant/après), compte par repli
      sur le corpus (→ liste de retrait soumise à DU-7), assistants `?N` = 0, bots épinglés ;
      carte de fermeture rejouée contre la référence de J4.0.5 (→ état du déclencheur de la
      représentation intermédiaire, rapporté à l'utilisateur).
- [ ] J11.4 **GO utilisateur**, puis vague locale, serveur arrêté, PRÉVENIR :
      `levelup backfill-replay` (un film à la fois, verrou solo) → `levelup backfill-usage-summary`
      → `levelup backfill-pad-tiers --force` → `levelup backfill-killsource` (3 ouvriers ;
      positions si `IsolationDecoderRev` a monté) → `levelup backfill-bomb-stats` (si J9.6) ;
      `levelup healthcheck` ; redémarrage du serveur.
- [ ] J11.5 Vérification visuelle par l'utilisateur sur les témoins QU'IL nomme.
- [ ] J11.6 Fusion `feat/suite-audit-decodeur` → `feat/v75` (G-push, push, CI verte au niveau
      job) ; séquence de production écrite (identique, au déploiement — geste de l'utilisateur).

---

### J12 — Modernisation neutre, documentation, CI (zéro différence)

- [ ] J12.1 `go fix` (Go 1.26) + stdlib sur le périmètre, SAUF les quatre pièges (DT-10) ; test
      `TestBijectionPermutationGraineFixe` posé AVANT ; G-equiv 0, équivalence killsource 0 ;
      empreintes régénérées à révision constante.
- [ ] J12.2 `errors.Is` partout où `os.IsNotExist` / `IsExist` / `IsPermission` restent ; ratchet.
- [ ] J12.3 Contexte : les points d'entrée de `replay`/`replaybuild` reçoivent le `ctx` de
      l'appelant (`BuildBytes` ne crée plus de `context.Background()`) ; les 219 appels `slog` du
      périmètre passent aux variantes `…Context` ; D-4 : plus de `slog` dans `grammar` et `facts`
      (diagnostics typés remontés à l'orchestrateur) ; deux ratchets (allowlists vides).
- [ ] J12.4 Les 22 variables de paquet exportées modifiables deviennent non exportées, `const` ou
      champs ; ratchet étendu (famille `filmdec_package_vars_test.go`) à `replay`, `replaybuild`,
      `killcollector`.
- [ ] J12.5 Documentation : 13 affirmations d'état fausses (liste rétablie sur pièces, dont la doc
      de paquet de `killsource`) ; chemins `.ai/` morts (12 relevés + ~60 de rr §8.26) et ratchet
      `TestCheminsAiCitesDansLeCodeExistent` ; 170 commandes `go test` visant `film/filmdec/` et
      ratchet sur ce littéral ; 10 doc comments recollés à leur déclaration ; CLAUDE.md :
      `.ai/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` ; ADR 0034 : décisions + état courant
      court, l'historique M2-M4 déplacé en annexe (`docs/adr/0034-annex-history.md`, EN) ; les six
      affirmations du §2.4 alignées sur l'arbre. Avec J3.1, les commentaires ne bougent plus les
      empreintes.
- [ ] J12.6 CI : job `film-race` —
      `go test -race -run TestDeuxFilmsEnParallele ./internal/games/halo_infinite/film/internal/grammar/`
      (mini-bobines du dépôt, sans DuckDB) ; D-5 devient vrai.
- [ ] J12.7 (DU-5 a) Tag `research` sur les 218 fichiers ; ratchet `archlint/research_tag_test.go` ;
      `go vet -tags=research` de la CI étendu au module ; baseline JSONL nettoyée dans le même
      commit.
- [ ] J12.8 (DU-5 b) Règle des commentaires écrite dans CLAUDE.md et `arch-rules`.

**Gate J12.** G-unit, G-arch, G-vet, G-lint, G-film, G-equiv 0, équivalence killsource 0,
`go test ./... -count=1` complet, G-CI, G-push → fusion. **Taille** : L.

---

## 6. Protocole de reprise

Relire le skill `plan-execution`, puis ce fichier : §3 (décisions validées, fermes), §9 (journal),
première case non statuée du jalon en cours ; `git -C <worktree> log --oneline -15`. Ne jamais
relancer un agent sans avoir vérifié qu'il est mort. Ne pas re-décider ce qui est tranché.

---

## 7. Conformité à la grille `plan-review`

| Axe | Réponse |
|---|---|
| Structure | Objectif et critères mesurables (§1.1) ; jalons ordonnés par dépendance (fraîcheur avant comportement, neutre en dernier) ; bloquants (P-1 à P-5) ; tailles ; branche nommée. |
| Couches Go | Le décodeur est propre au titre (`games/halo_infinite/film`, ADR 0012) : `analysis/` n'est pas concerné. Types : `film/types` (`LifeKey`), `games/canonical` (`LinkMethod`). Orchestration : `sync/killcollector`, `replaybuild`, `replaychild`. Infrastructure : `platform/atomicfile`, `platform/filelock`. Aucune logique ajoutée dans un handler ; J2.13 retire un décodage du processus serveur. Écriture DuckDB : persisters INSERT-only (J9.6). |
| Multi-titre | Étape killsource gardée par la capability `film.kill_source` (inchangée) ; tous les chemins par `PathResolver` ; aucune comparaison de slug ; aucun nouveau champ de stats. |
| Adapters | Non concerné (décodeur de film, pas de chargement canonique ajouté). |
| Tests | Par couche : unitaires purs (`grammar`, `facts`, `replay`), intégration `-p 1` (`killcollector`, `persist`), `httptest` (`haloclient`), processus enfant (`filelock`), fuzz, ratchets `archlint`, gates de corpus. |
| Journalisation | `slog.*Context` à clés standard ; erreur journalisée AVANT toute dégradation (règle 3) ; compteurs expvar nommés. |
| Front | Seulement aux montées de schéma (types, normaliseur) ; aucune chaîne d'UI nouvelle. |
| Livraison | Clôture en six actions par jalon (§4.1) ; thought_log par jalon ; dépendances externes écrites (P-1, GO). |
| Exécutabilité | Items fermés et cochables ; gates en commandes exactes ; statuts ; ordre ; découvertes ; décisions listées AVANT le GO ; reprise ; renvoi au skill `plan-execution`. |

---

## 8. Découvertes (consignées, non traitées)

1. (2026-09-25, reprise M4b de la campagne rr, non commitée) Le blob des faits
   `REPLAYINPUTS27` gagne deux compteurs en queue de `MovementStateStats`
   (`EventPacketsNewRecordStart`, `VehicleTypePhysicsAssumed`) SANS monter sa version ; la
   version 27 est née dans le commit `0c9021c7b` du même lot. Sans effet tant qu'aucun fichier de
   faits v27 d'avant la reprise n'est conservé (aucun dans le cache vivant, qui porte des faits
   antérieurs) ; à signaler au superviseur de la campagne. C'est la famille RA1 : J3 pose la règle
   « toute modification du codec monte sa version, avec un test de refus de l'ancienne ».

---

## 9. Journal (superviseur)

- 2026-09-25 : plan écrit sur la base du registre du 2026-09-24 ; pièces re-vérifiées sur
  `feat/v75` et sur les têtes de la campagne rr (voir l'en-tête) ; constats sur l'arbre de la
  campagne : RB2-1 et RB2-4 corrigés (`[~]`), RA1-6 corrigé par M8, manifeste atomique posé par L3,
  une quatrième copie du filtre de génération née dans `offline_aim_only.go`, 34 comptes non bornés
  dans le codec des faits. Rien de lancé ; en attente de P-1 à P-5.
- 2026-09-25 (soir) : **DU-1 à DU-7 validées** par l'utilisateur (« ok avec toi alors tu peux
  entériner ces décisions avec tes recos ») ; DU-3 = S en deux temps (S1 obligatoire, S2 sous
  benchmark), J4 réécrit avec les chiffres mesurés sur `feat/rr-m4b`. P-4 coché.
- 2026-09-25 (soir) : recoupement avec l'agent en cours (session `levelup-go-migration-29`, reprise
  M4b de la campagne rr, worktree `LevelUp-wt-rr-m4b`, 61 fichiers modifiés non commités, gate
  `go test` en cours depuis 16 h 37 ; aucun commit sur les branches depuis 14 h 09) :
  **GA2-1 pris en charge** (i54 au niveau 0x10 par `consumeSimStateHandleTail`) → `[~]` + contrôle
  P-3 ; site `ti=38 i18` laissé par la reprise → ajouté à J6 ; J6 part de
  `consumeSimStateHandleTail` ; rattachement des tirs changé par la campagne → RA2-3 (J5.4) à
  re-mesurer à l'entrée ; deux replis câblés ajoutés au registre → comptes de J8 à re-mesurer ;
  la table anticipée gagne un usage de lecture (`SlotDeLArchetype`) → précisé en J8.1. Aucun autre
  item du plan touché ; les autres worktrees sont inactifs depuis le 2026-09-20. Découverte
  consignée au §8.1.
- 2026-09-25 (soir) : P-2 fait (commit `baa0e4e14`, docs seules, non poussé).
- 2026-09-25 (soir) : **DU-8 validée** (« Ok je suis d'accord avec toi. Donc tu mets à jour le plan
  actuel pour la carte de fermeture et tu prépares une spec pour la structure intermédiaire ») :
  lot J4.0 (carte de fermeture) ajouté, DT-12, renvois en J5.0, J6.4, J10, J11.3 ; J4 passe à deux
  exécutants (~20 agents au total) ; la représentation intermédiaire est spécifiée à part,
  `.ai/SPEC_REPRESENTATION_INTERMEDIAIRE_FILM_2026-09-25.md`, hors plan (§1.4).
- 2026-09-25 (soir) : déclencheur de la spec corrigé avec l'utilisateur (« Oui ok ») — la fermeture
  des records qui PORTENT une donnée utile (colonne `product_use`), pas celle de tous les paquets ;
  J4.0.2 et J4.0.4 mesurent cette fermeture et classent les bloquants par records utiles
  débloqués ; la question des largeurs mesurées sans Ghidra est ouverte dans la spec (§11, 5).
  Plan, spec et thought_log commités sur `feat/v75`, sans push (`873f70663`).
- 2026-09-25 (soir) : **DU-9 décidée** (« Oui ok avec toi ») : largeurs présumées par mesure
  admises pour les composants de taille fixe qu'on ne fait que sauter ; Ghidra obligatoire pour
  les valeurs utilisées, les tailles variables et toute largeur présumée qui casse ; amendement
  D-3 ajouté à J3.7 ; règle détaillée dans la spec (§11).
- 2026-09-25 (soir) : à la demande de la session de la campagne rr, modifications DU-9 mises de
  côté hors du dépôt (copies vérifiées), arbre rendu propre, puis réappliquées après sa fusion
  (`569932b42`, `feat/v75` = `3cca6cf47`, poussé) — plan et spec inchangés par la fusion, entrée
  du thought_log rajoutée en fin. **P-1 et P-3 cochés** : les cinq lignes `[~]` sont confirmées
  sur l'arbre fusionné. Plus rien ne bloque J2 et la suite, hormis le GO par jalon. §8.1 reste
  valable (le blob des faits est toujours `REPLAYINPUTS27`, avec les deux compteurs ajoutés).
