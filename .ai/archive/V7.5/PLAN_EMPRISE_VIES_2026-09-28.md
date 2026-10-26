# Plan : Emprise — « Placement et rendement de chaque vie » et « Part des vies par placement » (2026-09-28)

> Sources, à lire avant tout lot, et qui FONT FOI pour le rendu :
> - `.ai/HANDOFF_EMPRISE_CLOTURE_2026-09-27.md` §5 (décision utilisateur du 2026-09-27, spécification
>   de rendu reprise au §2 ci-dessous) ;
> - artefact « Écart à l'équipe » https://claude.ai/artifact/TtJstMS6cBuCo4jP7tyRzo, proposition 2
>   (graphes `c2b` et `c2c`, tableau des grains, notes « Ce qu'on lit » / « Mon avis », tableau des
>   pièges) ;
> - `.ai/PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.md` (onglet Emprise : §1 D2 périmètre, §2
>   spécification commune, §4 organisation) ;
> - `.ai/V7.5/PLAN_TACTIQUE_2026-09-06.md:104` (décision utilisateur du 2026-09-07 : « les données
>   d'un match en base sont complètes au sync ; seul le rejeu peut attendre la cuisson »).
>
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun item sans statut, zéro fix hors
> périmètre, découvertes consignées au §7). Statuts : `[x]` fait, `[~]` couvert ailleurs (réf),
> `[!]` non fait (justification écrite). Aucune case vide à la clôture d'un lot.
>
> Statut : **GO utilisateur le 2026-09-28** (« oui go »). **Code livré le 2026-09-30** dans
> `feat/v75` (`511396d60`, CI verte au niveau job). Restent : V5.5 gate visuel par l'utilisateur,
> V5.6 rattrapage prod par l'utilisateur après le déploiement de la v7.5.

## 0. Hors périmètre

- Propositions 1, 3, 4, 5 et 6 de l'artefact (bilan loin / près, frise d'écart, rendement par
  distance, tendance sur la soirée, profil d'écart) : non retenues.
- « Isolement, soirée après soirée » (colonne droite du bloc « Par rapport à d'habitude ») : non
  reprise (décision du 2026-09-27). La colonne reste vide, comme aujourd'hui.
- La carte Riposte de Synergies (« Morts ripostées », « Temps de riposte », frise « Riposte ») et
  l'en-tête « Coordination » : CONSERVÉS (voir V7).
- Halo 5 : pas de film, donc pas de vies (V10).
- Véhicules comme ressource de l'Emprise : plan séparé `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`.
- Rattrapage de la prod : fait par l'utilisateur après le déploiement de la v7.5 (V5.6).

## 1. Décisions tranchées (ne pas rediscuter pendant l'exécution)

- **V1 — Lieu du calcul : AU SYNC, par le collecteur de kills.** Troisième projection du matériau
  que la passe de positions a déjà lu (`sync/killcollector/isolation_facts.go`,
  `materiauDIsolement` : registre d'identité + positions des bipèdes), après `match_lives` et
  `match_death_context`. Fondement : décision utilisateur du 2026-09-07, « les données d'un match
  en base sont complètes au sync ; seul le rejeu peut attendre la cuisson », qui a placé les faits
  d'isolement au sync ; le placement d'une vie EST un fait d'isolement. Aucune lecture d'artefact
  de rejeu, aucune révision de décodage montée (`decfilm.Rev`, `facts.Rev` : interdits).
- **V2 — Grain et stockage.** Une vie = une vie nommée du registre (la même que la ligne de
  `match_lives`, clé `(match_id, xuid, start_ms)`), tout le lobby. Table append-only NEUVE
  `match_life_placement` + vue `match_life_placement_latest` (dernière passe entière par match,
  modèle `migration/steps_shared_match_lives.go`), écrite par un persister INSERT-only sous son
  propre lease court, APRÈS les vies et sans jamais les bloquer (même doctrine que
  `projeterFaitsDIsolement` : échec journalisé et compté, jamais propagé). Révision propre
  `PlacementRev` dans `decoder_rev` ; `IsolationDecoderRev` NE BOUGE PAS (le contenu de
  `match_lives` ne change pas).
- **V3 — Mesure d'une vie.** Grille de 100 ms sur `[start_ms, end_ms]`, horloge du match.
  - Coéquipier = même `team_id` (base, `ids.Equipes`), autre xuid. Vivant = dans une de ses vies.
    Situé = position de moins d'une seconde (`visibleA` de `film/replay/death_context.go:361`,
    RÉUTILISÉE, jamais recopiée — `vivantA` idem, `:328`). Distance = 2D horizontale en mètres,
    même calcul que le contexte de mort.
  - Chaque instant reçoit UNE cause, dans cet ordre : `carrier` (le joueur porte l'objectif) >
    `team_down` (aucun coéquipier vivant) > `unplaced` (le joueur n'est pas situé : en véhicule ou
    position non lue, cf. V0.1) > `teammate_unplaced` (un coéquipier vivant n'est pas situé) >
    MESURÉ. Seuls les instants mesurés entrent dans la médiane et la part hors radar ; chaque
    cause est cumulée en ms sur la ligne de la vie et publiée.
  - `median_m` = médiane, sur les instants mesurés, de la distance au coéquipier vivant le plus
    proche ; NULL si moins de 2 000 ms mesurées (« vie non mesurée »).
  - `beyond_ms` = ms mesurées où cette distance dépasse la portée du radar (V5).
  - `kills` = frags du joueur rattachés à la vie : frag publiable (`publishable`), tueur et victime
    de camps différents (trahisons exclues), instant dans `[début de la vie, début de sa vie
    suivante)` — un frag posthume (grenade, échange) appartient à la vie qui vient de finir
    (1,5 % des frags du 22/09). Frags antérieurs à la première vie : comptés (compteur + journal),
    non rattachés.
  - `duration_ms` = `end_ms − start_ms`.
- **V4 — Quarts.** Isolé = `median_m / radar_m ≥ 1,0` ; rentable = `kills ≥ 1`. Quatre quarts :
  à portée et rentable, isolé et rentable, à portée et coûteux, isolé et coûteux. Une vie non
  mesurée n'est ni tracée ni classée ; elle est comptée et publiée. Classement calculé en Go
  (`analysis/`), jamais dans un composant React.
- **V5 — Portée du radar.** Table `[radar_range_m]` de `config/titles/halo_infinite/mappings/regulation.toml`
  (clé = nom de variante), résolue À L'ÉCRITURE par la même source que la lecture
  (`wire/registry.go` `radarRangeFor`, `teammates_squad_isolement.go` `rayonParMatchDuScope`),
  injectée dans le collecteur (option de câblage, aucune comparaison de slug), stockée par ligne
  (`radar_m`). Variante absente : `radar_m` et `beyond_ms` NULL ; le lecteur sort le match de
  l'univers et le compte (`matchs_sans_rayon`, motif existant). À la lecture, une ligne dont
  `radar_m` diffère de la portée courante du match est écartée, comptée et journalisée en `Warn`
  (la table a changé : un rattrapage est dû).
- **V6 — Porteur d'objectif.** Exclu du dénominateur (V3). Sa lecture au sync est la seule pièce
  neuve du décodage : décidée sur mesure en V0.2, puis par l'utilisateur le 2026-09-28 : **voie (b)**,
  les lectures de la cuisson (enregistrements d'entité, bursts de capture, pont par manche, objets du
  monde pour le drapeau) appelées depuis le film que le collecteur a déjà ouvert, calques produits
  par l'assembleur de PRODUCTION du rejeu (aucune copie de sa logique), sous la même garde de mode
  que la cuisson. Surcoût accepté (×2 environ sur le Drapeau). Familles couvertes : drapeau, crâne,
  bombe, VIP (celles que le rejeu publie). Stockpile et autres : non couverts, ce que dit l'infobulle
  de couverture. Limite connue, non traitée ici (§7) : le calque du drapeau ne publie aucun portage
  en Big Team Battle CTF (`flagFilm = false`), la cause `carrier` y reste donc muette.
- **V7 — Synergies : le nuage « Frags non ripostés » disparaît, la carte Riposte reste.** Vérifié
  sur pièces le 2026-09-28 : le titre « Frags non ripostés » est `squad.isolement.card_title`
  (`lib/i18n/manifests/squad.toml:977-979`), celui de `SquadIsolementNuageCard`
  (`SquadSynergiesPage.tsx:171`), pas celui de `SquadRiposteCard`. L'artefact, source de la
  décision, dit « Le nuage « Frags non ripostés » actuel disparaît, celui-ci prend sa place » ; le
  handoff §5 a nommé `SquadRiposteCard` par confusion des deux cartes. Rangée 1 de Synergies :
  « Appui » seul, règle actuelle de la grille inchangée (la carte présente garde sa colonne) —
  point soumis au gate visuel.
- **V8 — Emplacement.** Onglet Emprise, nouveau bloc « Groupés ou isolés » (EN « Grouped or
  isolated »), entre « Prendre, et s'en servir » et « Par rapport à d'habitude » (position du bloc
  dans la maquette de l'onglet, `MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html:435`) : le nuage
  pleine largeur, la barre des quarts dessous. Périmètre = D2 de l'Emprise (composition exacte ∩
  matchs filtrés). Joueurs tracés = ceux de la composition (JGtm et coéquipiers sélectionnés), dans
  l'ordre des fiches de l'Emprise ; « reste du camp » non tracé.
- **V9 — Rendu** : §2, à la lettre.
- **V10 — Capacité.** Même porte que les vies : `games.CapFilmKillPositions` (`film.kill_positions`,
  déclarée pour Halo Infinite seul). Absente : pas de projection au sync, pas de lecture, bloc
  absent du contrat, `ErrCapabilityNotSupported` journalisé en Debug comme les positions. Le
  prédicat de contenu de l'onglet (`empriseHasContent`) compte le bloc.
- **V11 — Lecture bornée (ADR 0036).** Un seul chargement par requête, borné par la liste des
  matchs du périmètre ET les xuids de la composition (paramètre liste, `platform/duckdb/perimetre_liste.go`),
  sur la vue `_latest` uniquement ; aucune lecture de `v_gamertag_lookup`.
- **V12 — Rattrapage.** `matchsAJour` (`cmd/levelup/cmd_backfill_killsource_selection.go:106`) exige
  en plus une passe de `match_life_placement_latest` à `PlacementRev` pour les matchs qui ont des
  vies. Le backlog automatique du post-sync (`killcollector/postsync.go`, `conditionBacklog`) NE
  CHANGE PAS : il ne regarde que `decfilm.Rev`, si bien que le déploiement ne relance aucun
  redécodage de lui-même.

## 2. Spécification de rendu (NON NÉGOCIABLE — handoff §5, artefact proposition 2)

Règles communes de l'Emprise (plan de l'onglet §2) : légendes en bas et centrées ; graphe centré
verticalement dans son bloc ; titres factuels ; FR et EN ; jetons de couleur seulement (skill
`color-tokens`), aucune valeur hex ni classe Tailwind couleur.

### 2.1 « Placement et rendement de chaque vie » (EN « Placement and yield of each life ») — nuage, hauteur 420

- Un point par vie, couleur du joueur (couleurs d'escouade, `getSquadPlayerColors`), liseré 1 px
  couleur de carte. Taille = durée : `6 + min(durée_s, 90) / 9`.
- X : « distance médiane au coéquipier le plus proche pendant la vie, en portées de radar » (EN
  « median distance to the nearest teammate during the life, in radar ranges »), titre d'axe centré
  sous l'axe, bornes 0 à 2, pas de 0,25, libellés à deux décimales avec virgule (point en EN). Une
  vie au-delà de 2 est posée à 2 (l'infobulle garde sa vraie valeur).
- Y : « frags dans la vie » (EN « kills in the life »), −0,5 à 5,5, pas de 1, libellés négatifs
  masqués ; décalage vertical ±0,25 pour décoller les points de même compte, DÉTERMINISTE (dérivé
  de `(match_id, xuid, start_ms)`, pas de `Math.random`), affichage seulement ; une vie à plus de 5
  frags est posée à 5 (l'infobulle garde sa vraie valeur).
- Repère du radar : trait vertical pointillé à 1,0, couleur d'accent, étiquette « portée du radar »
  (EN « radar range ») en haut à l'intérieur (11 px) ; PAS de zone « isolé » teintée sur ce graphe.
- Frontière horizontale pointillée (gris discret) à 0,5 : « au moins un frag dans la vie ».
- Quatre quarts nommés, texte gris discret 12 px, titre en capitales + sous-titre :
  - haut gauche « À PORTÉE ET RENTABLE » / « sûr » ;
  - haut droite « ISOLÉ ET RENTABLE » / « flanqueur, surveiller la régularité » ;
  - bas gauche « À PORTÉE ET COÛTEUX » / « duel à travailler, pas le placement » ;
  - bas droite « ISOLÉ ET COÛTEUX » / « vie donnée pour rien, seul » — le SEUL quart teinté (couleur
    `perf-tier-5` à 10 %, son texte dans cette couleur). Les quarts ne se classent pas du bon au
    mauvais.
  - EN : « IN RANGE AND PRODUCTIVE » / « safe » ; « ISOLATED AND PRODUCTIVE » / « flanker, watch the
    consistency » ; « IN RANGE AND COSTLY » / « a duel to work on, not the placement » ; « ISOLATED
    AND COSTLY » / « a life given away, alone ».
- Gros point par joueur : médiane X × médiane des frags de ses vies mesurées, taille
  `18 + min(nombre de vies, 200) / 10`, couleur du joueur, cerclé 2 px couleur de texte, au-dessus
  du semis.
- Infobulle d'une vie : « **Joueur** · une vie de m:ss » / « distance médiane X radar · P % de la
  vie hors radar · k frag(s) ». Infobulle du gros point : « **Joueur** · N vies » / « médiane X
  radar · k frag(s) par vie » / « P % des vies isolées et sans frag ». EN parallèles (« a life of
  m:ss », « median distance X radar · P% of the life out of radar · k kill(s) », « N lives »,
  « median X radar · k kill(s) per life », « P% of lives isolated and without a kill »).
- Légende ECharts en bas, centrée, un item par joueur ; un clic isole le semis ET le gros point du
  joueur (même nom de série). Choix délibéré de la légende native (l'isolement par nom de série
  en fait partie), et non de la légende DOM des autres cartes.
- Infobulle de la carte (titre, 3 phrases au plus) : ce qu'est une vie et la mesure ; les instants
  hors mesure (porteur d'objectif, équipe à terre, joueur ou coéquipier non situé) ; N vies
  mesurées sur N vies, matchs sans portée de radar connue.

### 2.2 « Part des vies par placement » (EN « Share of lives by placement ») — barres, hauteur 230

- Une barre horizontale empilée à 100 % par joueur (premier joueur en haut), épaisseur 22,
  séparateur 1 px couleur de carte entre segments. Axe X de 0 à 100 %, noms des joueurs en gras.
- Quatre segments, dans cet ordre et ces couleurs : « à portée et rentable » `perf-tier-1`,
  « isolé et rentable » `perf-tier-2`, « à portée et coûteux » `perf-tier-4`, « isolé et coûteux »
  `perf-tier-5` (`perf-tier-3` inutilisé ; mêmes couleurs pour les quarts du nuage ; jamais une même
  teinte à deux opacités).
- Valeur « v % » écrite DANS le segment (11 px, gras, encre sombre), seulement à partir de 8 %.
- Infobulle : « **Joueur** · N vies » puis une ligne par quart « nom : v % ».
- Légende en bas, centrée.

## 3. Organisation

- Worktree : `C:\Users\Guillaume\Projects\LevelUp-wt-emprise`, branche `wt/emprise` (chantier
  Emprise), rebasé en avance rapide sur `feat/v75` avant V0. Un exécuteur Opus par lot, lots
  SÉQUENTIELS. Le plan de référence est la copie du WORKTREE.
- Superviseur : vérifie chaque clôture sur pièces (gates rejoués, diff relu, découvertes
  requalifiées), fusionne `wt/emprise` dans `feat/v75` à la fin (verdict = CI de `feat/v75`, les
  branches `wt/**` n'ont pas de CI). Pas de push par les agents ; commits `feat(emprise-vies/<lot>)`.
- Environnement (tous lots) : une commande `go` à la fois ; CGO avec le gcc winlibs ; `GOCACHE`
  isolé au worktree ; données réelles = `C:\Users\Guillaume\Projects\LevelUp\data` en LECTURE SEULE
  (films du cache, copies de base dans le scratchpad si le serveur tient la base) ; aucun serveur,
  navigateur, backfill ni ouverture RW d'une base réelle par un agent ; vitest hors sandbox ; gates
  en avant-plan.
- Gate commun de clôture de lot (en plus du gate propre) :
  `cd apps/go-api && go test ./...` ; `make go-api-lint` ; `make check-types` ; `make test-web` ;
  `cd apps/web && npm run lint` ; `node tools/knip-ratchet.mjs` ;
  `node tools/lint-cross-feature-imports.mjs` ; `node tools/lint-no-hardcoded-colors.mjs` ;
  si le contrat change : `make openapi-gen && make generate-types && make openapi-check` ;
  lots web et lot de clôture : `lefthook run pre-push` (leçon L6.3 : deux garde-rails n'échouaient
  qu'au push).
- Clôture de lot = gate vert + items statués + section du lot mise à jour ici + entrée
  `.ai/thought_log.md` du worktree + commit du lot + rapport (fait / non fait / découvertes).
- Ordre : un lot ne commence qu'une fois le précédent CLOS au sens ci-dessus et vérifié par le
  superviseur. Un STOP de V0 arrête le plan entier jusqu'à la décision de l'utilisateur.

## 4. Lots

### V0 — Mesures préalables (Go, recherche) · moyen

Aucun code livré hors tests de recherche (`*_research_test.go`, sautés sans données, même motif que
`filmdec/e192_i0_catalogue_mesure_research_test.go`). Seuils écrits ICI, avant la mesure ; un seuil
manqué = STOP et rapport au superviseur, qui remonte à l'utilisateur (aucun repli choisi par
l'exécuteur).

- [x] V0.1 **Le joueur en véhicule n'est pas situé.** Sur au moins deux films Big Team Battle du
  cache dont l'occupation se lit (épisodes `src = film` du calque véhicules, schéma ≥ 67, construits
  en mémoire par le constructeur du rejeu), mesurer la part du temps à bord (épisodes lus) où le
  joueur n'a aucune position de moins d'une seconde dans `ScanBipedPositions`.
  **Seuil : ≥ 95 %.** Atteint : la cause `unplaced` couvre le véhicule, libellée « en véhicule ou
  position non lue ». Manqué : STOP.
  → **Atteint** : 96,27 % agrégé (96,87 / 95,48 / 95,26 % par film), journal V0 ci-dessous.
- [x] V0.2 **Porteurs d'objectif au sync.** Deux voies à mesurer, dans cet ordre : (a) le canal des
  armes tenues déjà balayé (le portage de la bombe n'y lit « aucune donnée de plus »,
  `replaybuild/matchfacts.go:180-190`) ; (b) les lectures de la cuisson (`replaybuild/matchfacts.go`
  `flagInput` / `skullInput` / `bombInput` / `vipInput`, enregistrements d'entité, bursts de capture,
  pont par manche), appelées depuis le film que le collecteur a déjà ouvert. Films : trois de CTF,
  deux d'Oddball, un d'Assaut, un de VIP s'il en existe, et les 12 films du 22/09 (seuls ceux d'un
  mode à porteur paient la lecture). Pour chaque voie : intervalles de port comparés à ceux de
  l'artefact du même match, surcoût de temps par match, pic de mémoire (les lectures d'entité ont
  déjà monté à 19-22 Go sur un film d'une autre grammaire).
  **Seuils : intervalles identiques à ±100 ms sur ≥ 98 % du temps porté ; surcoût moyen ≤ 25 % du
  temps actuel de la passe du collecteur sur les matchs à porteur ; pic de mémoire ≤ 1,5 × celui de
  la passe actuelle ; aucune révision de décodage à monter.** La voie (a) est retenue si elle passe,
  sinon (b). Aucune ne passe : STOP.
  → **Manqué par les deux voies — STOP V0.2** (décision à l'utilisateur, aucun repli choisi) :
  (a) fidélité 61,95 % < 98 % et surcoût moyen 31,4 % > 25 % ; (b) fidélité 100,00 %, mémoire
  1,44 × ≤ 1,5 ×, mais surcoût moyen 59,9 % > 25 %. Chiffres au journal V0 ci-dessous.
  → **Levé par décision utilisateur du 2026-09-28** (question posée par le superviseur avec les
  chiffres : ×2 environ sur le Drapeau, +5 à +11 s par match, +34 % sur la Bombe, +6 à 8 % sur
  Oddball et VIP, rien ailleurs ; réponse « Accepter le coût ») : **voie (b) retenue**, surcoût
  accepté. Le seuil de 25 % n'est pas réécrit : il reste la trace de ce qui a été mesuré et décidé.
- [x] V0.3 Rapport de mesure collé dans ce plan (tableaux, films, commandes), voie retenue en V0.2.
  → Rapport collé ci-dessous ; voie retenue par l'exécuteur : aucune (STOP V0.2) ; voie retenue
  après la décision utilisateur : **(b)**.
- Gate : les tests de recherche passent sur le poste, sautés sans données (`go test ./...` vert sans
  cache de films) ; gate commun côté Go.

#### Journal V0 (exécuteur, 2026-09-28)

**Instruments** (tests de recherche, sautés sans leurs variables d'environnement ; aucun code de
production modifié ; aucune écriture dans `data/`) :

| Fichier | Rôle |
|---|---|
| `apps/go-api/internal/sync/killcollector/emprise_v0_passe_research_test.go` | `TestEmpriseV0Identites` (noms de carte par `ReplayMapRepo`, faits par `ReplayFactsRepo`, sur la copie de base en lecture seule) ; `TestEmpriseV0Passe` : la passe ACTUELLE (`CollectMatch`, câblage de production : trois capabilities, cache disque, roster par défaut sur la copie, `CaptureDepuisCatalogue`, écritures dans une base temporaire migrée), 3 tours, médiane et pic ; puis une passe instrumentée (`decfilm.Decode`, `IdentitiesForMatch`, `BuildKillSourceBatch`, `buildPositionRows`) qui garde registre et positions |
| `apps/go-api/internal/sync/killcollector/emprise_v0_vehicules_research_test.go` | V0.1 : sondes à bord passées par `replay.ContextesDesMorts` (la seule porte exportée vers `visibleA`, sans recopie) |
| `apps/go-api/internal/replaybuild/emprise_v0_reference_research_test.go` | `TestEmpriseV0Reference` : le document de rejeu construit EN MÉMOIRE par le code actuel (les étapes de `BuildBytes`, faits de film ni relus ni rangés), calques véhicules et porteurs, horloge de l'axe |
| `apps/go-api/internal/games/halo_infinite/film/replay/emprise_v0_porteurs_research_test.go` + `emprise_v0_rapport_research_test.go` | `TestEmpriseV0Porteurs` : la base de la passe rejouée (registre contrôlé contre celui de la vraie passe), puis les deux voies, chronométrées (3 tours, médiane) et comparées à la référence |
| `apps/go-api/internal/games/halo_infinite/film/replay/emprise_v0_synthese_research_test.go` | `TestEmpriseV0Rapport` : tableaux agrégés ci-dessous |

**Commandes** (bash, depuis `apps/go-api`, une commande `go` à la fois ; `SP` = scratchpad de la
session ; lots de films découpés pour tenir sous 10 min par commande) :

```bash
export CGO_ENABLED=1 GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-emprise/.gocache
export EMPRISE_V0_DIR=$SP/emprise_v0 EMPRISE_V0_DB=$SP/shared_copy.duckdb \
       EMPRISE_V0_CACHE=C:/Users/Guillaume/Projects/LevelUp/data/cache
EMPRISE_V0_FILMS=<22 match_id> go test ./internal/sync/killcollector/ -run '^TestEmpriseV0Identites$' -v -count=1
EMPRISE_V0_FILMS=<lot> go test ./internal/replaybuild/ -run '^TestEmpriseV0Reference$' -v -count=1 -timeout 9m30s
EMPRISE_V0_FILMS=<lot> go test ./internal/sync/killcollector/ -run '^TestEmpriseV0Passe$' -v -count=1 -timeout 9m45s
EMPRISE_V0_FILMS=<lot> go test ./internal/games/halo_infinite/film/replay/ -run '^TestEmpriseV0Porteurs$' -v -count=1 -timeout 9m45s
go test ./internal/games/halo_infinite/film/replay/ -run '^TestEmpriseV0Rapport$' -v -count=1
```

**Films** (22, tous au schéma 71 en mémoire) : V0.1 — `4f77afc1` (BTB:CTF, Flood Gulch), `879a4dba`
(BTB:CTF, Fortitude), `5676a9ba` (BTB:Total Control, Insolence). V0.2 — CTF `6fe2acb7` (Ranked:CTF,
Aquarius), `0ffebf8b` (Ranked:CTF 3 Captures, Origin), `81c0fc99` (CTF:Arena, Catalyst) ; Oddball
`f9e99ca4` (Live Fire), `b4f9064c` (Streets) ; Assaut `69b16f5d` (Neutral Bomb, Origin) ; VIP
`00761d27` (Arena:VIP, Bazaar) ; les 12 du 22/09 de JGtm : `50256dd2`, `d3249f8a`, `43e96765`,
`859da825`, `fc3dcb49`, `39910eb1` (Super Fiesta), `ab526724`, `d6918972` (CTF:Arena), `8e376cb1`,
`316eff0c`, `5eb5d3b3`, `2b50122a` (Team Slayer). Registre rejoué = registre de la vraie passe
(calage et vies nommées identiques) sur les 22.

**Horloges (V0.1).** Frame `f` d'un épisode → film µs = `origineUs + f × 100 000`, où `origineUs`
est le premier paquet de position de la cuisson (minimum des positions qu'elle a balayées ;
contrôle : premier paquet du collecteur, écart 0 µs sur les trois films) ; puis horloge du MATCH du
collecteur = film µs / 1000 − `DeathOffsetMS()` de son registre — l'horloge de `visibleA`. Une
sonde par frame `[T0, T1]` (bornes incluses) d'un épisode `src = film`.

**V0.1 — part du temps à bord (épisodes lus) sans position de moins d'une seconde** :

| Film | Épisodes lus (écartés : proximité / sans xuid) | Sondes (100 ms) | Non situées | Part | 1re seconde non située |
|---|---|---|---|---|---|
| `4f77afc1` | 67 (44 / 2) | 14 004 | 13 566 | 96,87 % | 237 / 650 |
| `879a4dba` | 32 (14 / 0) | 5 326 | 5 085 | 95,48 % | 90 / 317 |
| `5676a9ba` | 22 (41 / 0) | 4 047 | 3 855 | 95,26 % | 40 / 220 |
| **Agrégé** | **121** | **23 377** | **22 506** | **96,27 %** | 367 / 1 187 |

Verdict : seuil ≥ 95 % **atteint** (et par film). Les instants situés à bord sont surtout la
première seconde après la montée (la fenêtre de visibilité d'une seconde garde la dernière
position au sol).

**V0.2 — méthode.** Référence : calques `FlagCarries` (état `carried`), `SkullCarries`,
`BombCarries`, `VipCrown` du document construit en mémoire, convertis en ms du match avec l'origine
et le calage (`coverage.bridge.deathOffsetMs`) de ce document. Fidélité par joueur : `identique` =
|R ∩ dilatation(C, 100 ms)|, `en trop` = |C| − |C ∩ dilatation(R, 100 ms)|, taux = identique /
(référence + en trop), agrégé en sommes. Voie (a) : chaîne de `balayerPortage` (images-clés d'armes,
dotations de naissance, `ScanHeldWeaponChanges`), portages par famille (drapeau `0x2a392328`, crâne
`0x0017592c`, bombe `0x3fee4fcf`) par `BuildHeldObjectCarry` ponté par le registre du collecteur ; la
couronne VIP n'est pas un objet tenu (aucune source). Voie (b) : `StatRecordsCtx`,
`CaptureBurstTimes`, pont par manche (morts, triplet, élimination, résidu), plus ce que chaque calque
exige (drapeau : équipes du film et objets du monde — poses puis socles, d'où les vies libres ;
bombe : le canal de (a)), calques produits par l'assembleur de production (`BuildFromPositions`)
nourri des seules entrées du sync (positions et registre du collecteur, garde de mode par la
variante). Le profil calibré par le kill-feed est posé sur le contexte avant les lectures
supplémentaires (ordre de la cuisson). Coût : médiane de 3 tours de chaque lecture supplémentaire,
rapportée à la médiane de 3 tours de `CollectMatch` (même film, cache disque chaud après le premier
tour, écritures dans une base temporaire). Mémoire : pic de `filmproc.Footprint` échantillonné à
10 ms, après `runtime.GC` + `FreeOSMemory` en début de phase ; rapport = max(base, lectures) / base,
la base (passe rejouée : décodage, positions, fil des morts, index, créations, registre) restant
vivante pendant les lectures.

**V0.2 — par match à porteur** (passe = médiane `CollectMatch`) :

| Film | Mode | Passe | (a) coût | (a) % | (b) coût | (b) % | Pic passe / base / (a) / (b) Gio | (a) fidélité | (b) fidélité |
|---|---|---|---|---|---|---|---|---|---|
| `6fe2acb7` | drapeau | 11 607 ms | 3 537 ms | 30,5 | 11 708 ms | 100,9 | 0,29 / 0,28 / 0,28 / 0,38 | 70,76 % | 100,00 % |
| `0ffebf8b` | drapeau | 5 706 ms | 1 814 ms | 31,8 | 5 636 ms | 98,8 | 0,15 / 0,14 / 0,14 / 0,19 | 89,57 % | 100,00 % |
| `81c0fc99` | drapeau | 7 613 ms | 2 105 ms | 27,6 | 6 756 ms | 88,7 | 0,18 / 0,16 / 0,16 / 0,21 | 84,35 % | 100,00 % |
| `ab526724` | drapeau | 13 167 ms | 5 106 ms | 38,8 | 13 425 ms | 102,0 | 0,36 / 0,30 / 0,30 / 0,39 | 73,14 % | 100,00 % |
| `d6918972` | drapeau | 12 771 ms | 3 905 ms | 30,6 | 12 237 ms | 95,8 | 0,31 / 0,33 / 0,33 / 0,39 | 94,02 % | 100,00 % |
| `f9e99ca4` | crâne | 28 294 ms | 8 166 ms | 28,9 | 1 769 ms | 6,3 | 0,46 / 0,43 / 0,43 / 0,62 | 80,88 % | 100,00 % |
| `b4f9064c` | crâne | 19 224 ms | 4 916 ms | 25,6 | 1 075 ms | 5,6 | 0,37 / 0,36 / 0,36 / 0,47 | 85,61 % | 100,00 % |
| `69b16f5d` | bombe | 5 619 ms | 1 808 ms | 32,2 | 1 899 ms | 33,8 | 0,14 / 0,12 / 0,12 / 0,15 | 100,00 % | 100,00 % |
| `00761d27` | VIP | 8 117 ms | 2 979 ms | 36,7 | 620 ms | 7,6 | 0,22 / 0,22 / 0,22 / 0,27 | 0,00 % | 100,00 % |

Les 10 autres films du 22/09 (Super Fiesta, Team Slayer) ne sont pas d'un mode à porteur : la
garde de mode ne lit rien, aucune voie ne paie (passes de 7 266 à 9 534 ms, pics 0,19 à 0,28 Gio).

Détail du coût (b) sur le drapeau (statborg / pont / équipes / objets du monde / assemblage, ms) :
`6fe2acb7` 596 / 1 / 1 154 / 9 654 / 303 ; `0ffebf8b` 282 / 1 / 473 / 4 744 / 138 ; `81c0fc99`
337 / 0 / 572 / 5 697 / 150 ; `ab526724` 610 / 0 / 1 323 / 11 204 / 287 ; `d6918972` 622 / 1 /
1 088 / 10 253 / 273. Les objets du monde (poses puis socles) font 82 à 85 % du coût (b) du drapeau.
Sensibilité mesurée, PAS une voie : le drapeau sans objets du monde coûterait 13,9 à 17,7 % de la
passe, mais sa fidélité tombe à 55,41 % (portages non fermés au lâcher : 411 100 ms en trop).

**V0.2 — agrégats et verdict par seuil :**

| Seuil | Voie (a) | Voie (b) |
|---|---|---|
| Fidélité ±100 ms ≥ 98 % du temps porté | **61,95 %** — manqué (drapeau 81,87 %, crâne 82,53 %, bombe 100 %, VIP 0 % : 517 600 ms sans source) | **100,00 %** — atteint (drapeau 510 900 ms, crâne 698 500 ms, bombe 125 300 ms, VIP 517 600 ms, rien en trop) |
| Surcoût moyen ≤ 25 % (9 matchs à porteur) | **31,4 %** — manqué | **59,9 %** — manqué (drapeau 88,7 à 102,0 %, bombe 33,8 %, crâne 5,6 et 6,3 %, VIP 7,6 %) |
| Pic mémoire ≤ 1,5 × la passe | 1,00 × — atteint | 1,44 × (pire : `f9e99ca4`) — atteint |
| Aucune révision de décodage | atteint : aucune sortie décodée existante ne change (`decfilm.Rev`, `facts.Rev` intacts) ; mais le canal des armes tenues n'est pas exposé par la façade `decfilm` (lecture de `film/internal/grammar`) : un symbole de façade à ajouter (ratchet 166, justification datée) | atteint : `StatRecordsCtx`, `CaptureBurstTimes`, `ResolveRoundIdentity` sont déjà à la façade ; les calques sont privés à `film/replay` (une entrée exportée à créer) |

**Conclusion : aucune voie ne passe tous les seuils → STOP V0.2.** Le plan s'arrête jusqu'à la
décision de l'utilisateur (§3). Aucun repli choisi, aucun seuil ajusté.

### V1 — Calcul pur (Go) · moyen

Périmètre : `games/halo_infinite/film/replay/placement_des_vies.go` (+ tests), à côté de
`ContextesDesMorts` ; lecture des porteurs retenue en V0.2 (fonction pure, entrée du calcul).

- [x] V1.1 `PlacementDesVies(entree) []PlacementVie` : entrée = registre, positions, vies nommées,
  camps, morts du journal (tueur, victime, instant, publiable), intervalles de port, portée du
  radar ; sortie = une ligne par vie (V3). Réutilise `visibleA` / `vivantA` (aucune copie).
  → `film/replay/placement_des_vies.go` ; signature `PlacementDesVies(EntreePlacement)
  ([]PlacementVie, BilanPlacement)` (le second retour porte les frags écartés, dont « avant la
  première vie » que la décision V3 veut comptés). Journal V1 ci-dessous.
- [x] V1.2 Tests unitaires synthétiques, un par règle : ordre des causes ; équipe à terre ;
  porteur ; joueur non situé ; coéquipier non situé ; médiane ; `beyond_ms` ; vie de moins de
  2 000 ms mesurées ; frag posthume rattaché à la vie finie ; frag avant la première vie ; trahison
  exclue ; frag non publiable exclu ; variante sans portée. Chaque test vu ROUGE sous une mutation
  nommée dans le journal du lot.
  → 14 tests (les 13 règles + le refus du pont non publiable), 14 mutations vues rouges (M1 à M14).
- [x] V1.3 Test témoin sur deux films du 22/09 (sauté sans cache) : nombre de vies = celui de
  `match_lives_latest` pour ces matchs ; ≥ 97 % des frags publiables rattachés ; pour les vies
  finies par une mort, distance du dernier instant mesuré à moins de 1 m de celle du contexte de
  mort (`match_death_context`) sur ≥ 90 % d'entre elles.
  → `sync/killcollector/emprise_v1_temoin_research_test.go`, `8e376cb1` et `2b50122a` : 102/102 et
  88/88 vies, 100 % des frags, 100 % des fins (écart max 0,26 m).
- [x] V1.4 Intervalles de port depuis les entrées du sync (voie (b), V6) : une entrée EXPORTÉE de
  `film/replay` qui rend, pour un film et le registre du collecteur, les intervalles de port par
  xuid (drapeau, crâne, bombe, VIP) en produisant les calques par l'assembleur de production — la
  même chaîne que l'instrument `emprise_v0_porteurs_research_test.go` a mesurée, sans recopier la
  logique des calques ni celle du pont par manche (si le pont vit dans `replaybuild`, il est déplacé
  ou partagé, jamais dupliqué). Garde de mode identique à la cuisson. Ratchets de surface
  (`archlint/film_facade_surface_test.go`, compteurs `replay.X`) mis à jour avec justification
  datée. Tests : intervalles identiques à ceux du document de rejeu sur les 9 films à porteur de V0
  (sauté sans cache), et une garde de mode testée (aucune lecture hors mode à porteur).
  → `replay.PortagesAuSync` (`film/replay/porteurs_au_sync.go`) ; le pont par manche, les gardes
  de mode et les entrées des quatre calques DÉPLACÉS de `replaybuild` vers `film/replay`
  (`pont_par_manche.go`, `porteurs_entrees.go`), `replaybuild` les appelle ; 9 films sur 9
  identiques à l'union près, ratchet compagnon 278 → 291 daté.
- Gate : tests cités + gate commun (Go).

#### Journal V1 (exécuteur, 2026-09-28)

**Fichiers.** Neufs dans `apps/go-api/internal/games/halo_infinite/film/replay/` :
`placement_des_vies.go` (+ `_test.go`), `porteurs_au_sync.go` (+ `_test.go`),
`porteurs_entrees.go`, `pont_par_manche.go`, `emprise_v1_porteurs_research_test.go`, et quatre
tests DÉPLACÉS de `replaybuild` avec le code qu'ils testent (`porteurs_drapeau_identite_test.go` ←
`flagidentity_test.go` sauf `TestScoreboardTeamsEstUnControle`, resté ; `porteurs_crane_identite_test.go`
← `skullidentity_test.go` ; `pont_par_manche_residu_test.go` ← `pontresidu_test.go` ;
`porteurs_gardes_test.go` ← `bombvariant_test.go`, dont le ratchet « one bomb » balaie désormais
`replay`, où vit la garde). Modifiés : `death_context.go` (`indexerParXUID(positions, registre)` et
`distanceHorizontale`, partagés par les deux lecteurs — aucune copie de `visibleA`/`vivantA`, appelés
tels quels), `replaybuild/matchfacts.go` et `zones.go` (copies supprimées : `pontParManche`,
`withFlagIdentity`, `flagInput`, `skullInput`, `vipInput`, `bombInput`, `isVipVariant`,
`isSkullVariant`, `isBombVariant`), trois tests de `replaybuild` (`replay.NouveauPontParManche`),
six commentaires de `replay` qui nommaient les anciens emplacements, et
`archlint/film_facade_surface_test.go` (plafond compagnon 278 → 291, 13 symboles nommés, 0
retrait ; façade `decfilm` intacte à 166). Témoin : `sync/killcollector/emprise_v1_temoin_research_test.go`.

**Décisions prises dans le cadre de V3 (à relire par le superviseur).**
- Grille FERMÉE `[start_ms, end_ms]` (texte de V3) : une vie compte `⌊durée/100⌋ + 1` instants de
  100 ms, si bien que la somme des cinq cumuls dépasse `duration_ms` d'au plus un pas (témoin :
  3 176 900 ms cumulés pour 3 171 602 ms de durée sur `8e376cb1`). `beyond_ms / measured_ms` n'en
  est pas affecté.
- Médiane d'un nombre pair d'instants = moyenne des deux centraux ; arrondie à deux décimales comme
  toute distance publiée (`arrondiMetres`). « Dépasse la portée » = strictement supérieur.
- Un joueur SANS camp en base n'a aucun coéquipier : ses instants non portés sont « équipe à terre ».
- Frag dont un camp est inconnu (victime bot ou non résolue) : ni frag ni trahison prouvés, écarté et
  compté (`FragsCampInconnu`) ; tueur non résolu : écarté et compté. Témoin : 0 et 0.
- Registre au pont non publiable (`PontPubliable` faux) : AUCUNE ligne, `BilanPlacement.PontNonPubliable`
  — même refus et même raison que `ContextesDesMorts` (règle ajoutée, non écrite en V3 : §7).
  **Décision amendée le 2026-09-29 (superviseur, lot V2b.2)** : une ligne PAR VIE nommée, chaque
  instant « non situé » (`unplaced`), `measured_ms = 0`, médiane NULL, portée recopiée (hors radar
  0), frags rattachés comme ailleurs, `PontNonPubliable` toujours dit au bilan — sans ligne, un
  match à vies restait candidat au rattrapage à chaque passe.
- `PlacementVie.DerniereMesure` (dernier instant mesuré et sa distance) est la valeur que le critère 3
  de V1.3 confronte au contexte de mort ; V2.1 ne la persiste pas.
- Porteurs : seul l'état `carried` du drapeau compte ; `carried_open` (borne haute, aucun lâcher daté)
  est écarté et compté (`BilanPortages.DrapeauOuverts`) — 0 sur les 9 films. Garde du SYNC pour le
  drapeau : la famille CTF de la variante (`GardesDesPorteurs.Drapeau`), puis les trois signaux du
  film comme à la cuisson (§7).

**Mutations V1.2** (appliquées une à une sur `placement_des_vies.go`, test ciblé rouge, fichier
restauré par copie depuis le scratchpad, restauration vérifiée par `cmp`) :

| # | Test | Mutation | Rouge |
|---|---|---|---|
| M1 | `OrdreDesCauses` | « équipe à terre » examinée AVANT « porteur » | porteur 0 ms / à terre 6 000 |
| M2 | `EquipeATerre` | règle « équipe à terre » retirée | mesure 10 100 ms (distance infinie mesurée) |
| M3 | `Porteur` | borne de fin exclue dans `porteA` | porteur 900 ms |
| M4 | `JoueurNonSitue` | joueur non situé classé « coéquipier non situé » | non situé 0 |
| M5 | `CoequipierNonSitue` | coéquipier non situé sauté (`continue`) | mesure 10 100 ms |
| M6 | `Mediane` | valeur centrale haute sans moyenne | 10 m au lieu de 6,5 |
| M7 | `HorsRadar` | `>=` au lieu de `>` | 10 100 ms au lieu de 5 100 |
| M8 | `VieCourteNonMesuree` | seuil `<=` au lieu de `<` | médiane nil à 2 000 ms |
| M9 | `FragPosthume` | vie bornée par sa fin (`[début, fin]`) | 0 frag rattaché |
| M10 | `FragAvantLaPremiereVie` | frag rattaché à la première vie | 1 frag sur la vie |
| M11 | `TrahisonExclue` | test de camp retiré | 1 frag, 0 trahison |
| M12 | `FragNonPubliableExclu` | test de publiabilité retiré | 1 frag |
| M13 | `VarianteSansPortee` | `HorsRadarMS` posé sans portée | hors radar 0 au lieu de nil |
| M14 | `PontNonPubliable` | refus du pont retiré | 3 vies rendues |

Mutations de l'entrée des porteurs (`porteurs_au_sync.go`) : G1 retour anticipé de la garde retiré
(rouge : un assemblage a lieu, `SansCalage`), G2 `carried_open` compté (rouge), G3 pas de frame en
ms au lieu de µs (rouge), G4 statborg lu hors drapeau/crâne/VIP (rouge sur la bombe), G5 lectures du
drapeau hors CTF (rouge sur VIP et Oddball), G6 armes tenues hors bombe (rouge). Premier essai de
G1 NON rouge : chaque lecture était déjà gardée par famille, le retour anticipé n'évitait que
l'assemblage — le test exige désormais un bilan vierge.

**Témoin V1.3** (`EMPRISE_V1_FILMS=8e376cb1…,2b50122a…` Team Slayer du 22/09, copie de base,
cache de films ; porteurs non nourris : la garde de V1.4 ne lit rien en Team Slayer ; portée non
nourrie : V2.5, aucun critère ne la lit) :

| Film | Vies calculées / base | Avec médiane | Frags rattachés / éligibles | Fins comparées à < 1 m | Écart max |
|---|---|---|---|---|---|
| `8e376cb1` | 102 / 102 | 102 | 98 / 98 (100 %) | 98 / 98 (100 %) | 0,20 m |
| `2b50122a` | 88 / 88 | 86 | 83 / 83 (100 %) | 80 / 80 (100 %), 1 vie sans contexte à distance | 0,26 m |

Ventilation (ms) `8e376cb1` : durée 3 171 602, mesuré 2 997 200, porteur 0, équipe à terre 36 800,
non situé 102 400, coéquipier non situé 40 500 ; `2b50122a` : 3 127 986 / 3 023 000 / 0 / 24 700 /
26 800 / 58 100. Écartés par règle : 0 partout (non publiables, tueur inconnu, camp inconnu,
trahisons).

**Fidélité V1.4** (`TestEmpriseV1Porteurs`, protocole et références du V0, registre rejoué contrôlé
conforme à la passe) : 9 films sur 9, joueurs à l'union des portages IDENTIQUE, 100,00 % à ±100 ms,
rien en trop.

| Film | Famille | Lectures | Intervalles | Joueurs identiques | Référence | Durée de l'entrée |
|---|---|---|---|---|---|---|
| `6fe2acb7` | drapeau | statborg, équipes, monde | 144 | 8/8 | 83 400 ms | 12 150 ms |
| `0ffebf8b` | drapeau | statborg, équipes, monde | 48 | 8/8 | 48 000 ms | 6 058 ms |
| `81c0fc99` | drapeau | statborg, équipes, monde | 44 | 7/7 | 127 600 ms | 7 704 ms |
| `ab526724` | drapeau | statborg, équipes, monde | 31 | 7/7 | 124 300 ms | 17 181 ms |
| `d6918972` | drapeau | statborg, équipes, monde | 25 | 6/6 | 127 600 ms | 14 746 ms |
| `f9e99ca4` | crâne | statborg | 45 | 7/7 | 444 100 ms | 1 730 ms |
| `b4f9064c` | crâne | statborg | 33 | 6/6 | 254 400 ms | 1 169 ms |
| `69b16f5d` | bombe | armes tenues | 24 | 4/4 | 125 300 ms | 2 319 ms |
| `00761d27` | VIP | statborg | 13 | 7/7 | 517 600 ms | 699 ms |

(La durée est celle d'un tour unique, pont par manche et assemblage compris, cache chaud ; les
coûts de V0.2 restent la mesure de référence.)

**Gate** (depuis `apps/go-api`, `CGO_ENABLED=1`, `GOCACHE` du worktree) : `go test` par trois lots
(games+sync+replaybuild+archlint : 48 ok ; reste d'`internal` : 108 ok ; `cmd` : 34 ok) — un premier
passage du lot 1 a vu `sync/skill` `TestLUSRV2Shadow_RafalesBornees_300Candidats` dépasser son seuil
de 2 s sous charge (2,02 s), vert seul puis vert au second passage entier (§7) ; `go vet` et
`golangci-lint run` sur `replay`, `replaybuild`, `killcollector`, `archlint` : 0 problème ;
`go test ./internal/archlint/...` vert ; `gofmt -l` vide. Commandes des tests à données :

```bash
export CGO_ENABLED=1 GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-emprise/.gocache
EMPRISE_V1_FILMS=8e376cb1-8885-4ed5-a942-fca601e86620,2b50122a-b4e0-43a4-a46b-16f28279bb40 \
EMPRISE_V1_DB=$SP/shared_copy.duckdb EMPRISE_V1_CACHE=C:/Users/Guillaume/Projects/LevelUp/data/cache \
  go test ./internal/sync/killcollector/ -run '^TestEmpriseV1Temoin$' -v -count=1
EMPRISE_V0_FILMS=<9 films a porteur, deux lots> EMPRISE_V0_DIR=$SP/emprise_v0 \
EMPRISE_V0_CACHE=C:/Users/Guillaume/Projects/LevelUp/data/cache \
  go test ./internal/games/halo_infinite/film/replay/ -run '^TestEmpriseV1Porteurs$' -v -count=1
```

**Ce que V2 doit câbler.**
- `replay.PlacementDesVies(EntreePlacement{Positions: mat.positions, Registre: mat.registre,
  Equipes: equipesNumeriques(ids.Equipes), Journal, Portages, RadarM})` APRÈS
  `projeterFaitsDIsolement`, même matériau. `Journal` = la liste `fusionnees` ET la publiabilité de
  la passe fusionnée : `write` ne rend aujourd'hui que `batch.Deaths` — il doit rendre aussi
  `Publishable`. Compteurs : `BilanPlacement` (frags avant la première vie, sans vie, écartés par
  règle, pont non publiable) et les vies à `MedianeM` nil (non mesurées).
- `replay.PortagesAuSync(ctx, EntreePorteursAuSync{…})` : le film, le `FilmContext` ouvert par
  `buildPositionRows` (aujourd'hui local — `materiauDIsolement` doit le porter), l'entrée de carte,
  `game_variant_name` (le collecteur ne le lit pas encore), `&res.ProfilCalibre`, l'`IdentityInput`
  de `entreeDuRegistre` (à porter aussi), la feuille du match (frags, morts, assistances par xuid),
  les socles de drapeau du catalogue d'objectifs par `map_id` (la projection de
  `replaybuild.flagSpawns`/`flagSpawnTeam` est à PARTAGER, pas à recopier) et le catalogue de
  libellés (`replaylabels.Load`). La garde de mode est DANS l'entrée (`GardesDeLaVariante`) : hors
  mode à porteur elle rend avant toute lecture ; `BilanPortages.Lectures` dit ce qui a été payé.
- `RadarM` : portée de la variante injectée (V2.5), nil si absente.

### V2 — Écriture au sync (Go, persistance — lot sensible) · lourd

Périmètre : migration de la table, persister, câblage du collecteur, révision, rattrapage.

- [x] V2.1 Migration `match_life_placement` (séquence `id`, `match_id`, `decode_pass`, `decoder_rev`,
  `written_at`, `xuid`, `start_ms`, `end_ms`, `duration_ms`, `measured_ms`, `median_m`, `beyond_ms`,
  `radar_m`, `carrier_ms`, `team_down_ms`, `unplaced_ms`, `teammate_unplaced_ms`, `kills`), un seul
  index, vue `match_life_placement_latest` (dernière passe entière par match). Inscrite à l'ordre
  des migrations (`migration/order.go`).
  → `migration/steps_shared_match_lives_placement.go`, step `shared_match_life_placement_v1`
  (global, comme `shared_match_lives_v1`), inscrit juste après lui dans `canonicalOrder` (le nom du
  fichier trie juste après `steps_shared_match_lives.go`). Colonnes du plan, noms tels quels ; index
  unique `idx_match_life_placement_lookup (match_id, xuid, start_ms, written_at)`.
- [x] V2.2 `persist.LifePlacementPersister` INSERT-only, une transaction par passe, validation des
  lignes (modèle `persist/lives_persister.go`) ; tests d'intégration (`-tags=integration`).
  → `persist/life_placement_persister.go` (+ `_integration_test.go`, 4 tests).
- [x] V2.3 Garde-rails : `sync/no_art_patterns_test.go`, `sync/append_only_state_guard_test.go`,
  `migration/compaction_registry.go` (+ `games/halo_infinite/migrations/compaction_e2e_test.go`).
  Aucune entrée d'allowlist sans justification datée.
  → les quatre inscrits, AUCUNE entrée d'allowlist ; en plus, la table est enrôlée au garde de
  lecture brute (`platform/duckdb/no_raw_rating_reads_test.go`, ADR 0030 D-4) avant tout lecteur.
- [x] V2.4 Collecteur : `projeterPlacementDesVies` appelée APRÈS `projeterFaitsDIsolement`, jamais
  bloquante, sous son propre lease ; compteurs ADR 0009 (matchs couverts, vies écrites, vies non
  mesurées, frags hors vie, échecs d'écriture, matchs sans portée) ; `slog` structuré (Info au
  succès, Error sur échec d'écriture, Debug sans capability). Lecture des porteurs (voie V0.2) sous
  la même garde de mode que la cuisson.
  → `sync/killcollector/placement_des_vies.go`. Appelée seulement si les vies sont ÉCRITES
  (`projeterFaitsDIsolement` rend désormais ce booléen), sous `CapFilmKillPositions` (même porte
  que les positions, message Debug élargi). Journal détaillé ci-dessous.
- [~] V2.5 Portée du radar injectée au collecteur (option de câblage dans `api/wire`), même source
  que la lecture ; test de câblage qui lit l'arbre syntaxique (modèle
  `api/wire/registry_pages_home_teammates_wiring_test.go`).
  → **INAPPLICABLE TELLE QU'ÉCRITE — ARRÊT et rapport (consigne du brief)** : `api/wire` ne
  construit NI le collecteur NI le moteur de sync qui le porte (vérifié : aucune référence à
  `killcollector` sous `internal/api`). Le collecteur naît en trois lieux : `killcollector.RunPostSync`
  (le hook est créé par `sync.NewSyncEngineForTitle` lui-même, « pas au wiring », pour qu'aucun
  site ne l'oublie ; les moteurs serveur sont construits par `scheduler.BuildEngine` et
  `cmd/server/sync_v2_wiring.go`), `cmd/levelup` `backfill-killsource` et `--online` — dont la passe
  de rattrapage V5.2, qui ne passe par aucun `api/wire`. Fait côté collecteur : l'option de
  RÉCEPTION `AvecPorteeDuRadar(PorteeDuRadar)` (testée, `RadarM` nil sans elle, compteur
  `killsource_placement_matchs_sans_portee`). NON FAIT : l'appel de production (aucun appelant :
  en l'état, toute ligne s'écrit `radar_m`/`beyond_ms` NULL) et le test de câblage par AST.
  Décision à prendre (superviseur/utilisateur), voir le journal V2.
  → **Tranché par le superviseur le 2026-09-29 (décision technique, lot V2b ci-dessous)** : voie (a).
  La portée voyage par `CaptureDepuisCatalogue`, le chemin commun aux trois lieux de naissance du
  collecteur (il y charge déjà libellés et objectifs), qui charge `regulation.toml` du titre par le
  MÊME chargeur que `api/server_apiv1.go:1233-1238` ; la résolution variante → mètres est
  CENTRALISÉE (règle 6 : troisième site) dans un seul helper de `games/mappings`, les deux copies
  existantes (`TacticalService.rayonsParMatch`, `rayonParMatchDuScope`) y migrent, et un garde-rail
  interdit toute autre résolution. Le test de câblage porte sur l'application de la capture au
  collecteur, pas sur `api/wire`.
  → **`[~]` couvert par V2b.1** (2026-09-29, journal V2b) : la portée voyage par la capture, les
  trois lieux de naissance l'ont ; le test de câblage est `TestAvecCapture_PoseLaPorteeDuRadar`.
- [x] V2.6 `PlacementRev` + `matchsAJour` étendu (V12) ; `conditionBacklog` du post-sync inchangé,
  avec un test qui le vérifie.
  → `killcollector.PlacementRev = "placement-2026-09-29-v1"` ; `IsolationDecoderRev`, `decfilm.Rev`,
  `facts.Rev` intactes (diff vide sur leurs constantes) ; `postsync.go` non modifié ;
  `TestBacklogAJour_IgnoreLePlacementDesVies` et `TestMatchsAJour_ExigeLePlacementDesVies`.
- [x] V2.7 Test d'intégration du collecteur sur un film témoin (sauté sans cache) : les lignes
  écrites égalent le calcul pur de V1.3.
  → `TestEmpriseV2Temoin` : `8e376cb1` (Team Slayer) 102/102 et `ab526724` (CTF, 31 portages lus)
  150/150 lignes égales champ à champ.
- [x] V2.8 Skill `db-schema` : la table et sa vue ajoutées.
  → section `match_life_placement` + mention des tables à `decode_pass` dans la règle append-only.
- Gate : gate commun + `cd apps/go-api && go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/migration/... ./internal/games/halo_infinite/migrations/...`.
  → vert (commandes et sorties au journal V2). Gate web non joué : lot Go seul, aucun fichier web.

#### Journal V2 (exécuteur, 2026-09-29)

**Fichiers.** Neufs : `migration/steps_shared_match_lives_placement.go` ;
`persist/life_placement_persister.go` (+ `_integration_test.go`) ;
`sync/killcollector/placement_des_vies.go` (+ `_test.go`, `_integration_test.go`) ;
`games/halo_infinite/film/replay/socles_de_drapeau.go` ;
`cmd/levelup/cmd_backfill_killsource_selection_placement_integration_test.go`. DÉPLACÉ :
`replaybuild/flagspawns_test.go` → `replay/socles_de_drapeau_test.go` (avec la projection qu'il
teste ; son second test appelait une copie de la projection, il appelle désormais la projection).
Modifiés : `migration/order.go`, `migration/compaction_registry.go` (+ commentaires « 14 tables » de
`compaction_test.go` et `compaction_e2e_test.go`), `sync/no_art_patterns_test.go`,
`sync/append_only_state_guard_test.go`, `platform/duckdb/no_raw_rating_reads_test.go`,
`games/halo_infinite/migrations/compaction_e2e_test.go`, `replaybuild/flagspawns.go` (appelle
`MapObjectivesEntry.SoclesDeDrapeau`, `flagSpawnTeam` retirée), collecteur (`identities.go`,
`roster.go`, `isolation_facts.go`, `positions.go`, `collector.go`, `collector_run.go`,
`capture.go`), tests du collecteur ajustés aux signatures (`positions_test.go`,
`positions_openings_integration_test.go`, `backfill_cout_integration_test.go`),
`emprise_v1_temoin_research_test.go` (sa copie `v1Journal` remplacée par `journalDuPlacement` de
production), `collector_ouvriers_integration_test.go` (la vue du placement comparée entre 1 et
3 ouvriers : 200 lignes identiques), `cmd/levelup/cmd_backfill_killsource_selection.go`,
`archlint/film_facade_surface_test.go` (compagnon 291 → 293 daté : +`EntreePorteursAuSync`,
`BilanPortages`, `IntervalleDePort`, −`PointObjective`), `.claude/skills/db-schema/SKILL.md`.

**Câblage.** `collect` → `write` rend désormais la PASSE FUSIONNÉE (`persist.KillSourceBatch` :
morts ET publiabilité) → `collectPositions(…, batch.Deaths, fusionne)` → `ecrireLesDeuxPasses` :
positions, puis `projeterFaitsDIsolement` (rend « vies écrites »), puis SI les vies sont écrites
`projeterPlacementDesVies` → `replay.PortagesAuSync` (garde de mode dans l'entrée) →
`replay.PlacementDesVies` → `persist.LifePlacementPersister` sous son propre lease. Le matériau
(`materiauDIsolement`) porte en plus le film, le contexte du balayage, l'entrée de carte, le profil
calibré et l'`IdentityInput` du registre. Variante, `map_id` et feuille (frags, morts, assistances)
viennent de la MÊME lecture du roster que les équipes (`participantsForMatch`, porte de la base et
segment de lecture inchangés). Libellés (`replaylabels.Load`) et catalogue d'objectifs chargés par
`CaptureDepuisCatalogue`, le chemin commun aux trois appelants (post-sync, backfill, `--online`),
best-effort journalisé. Portée : option `AvecPorteeDuRadar(PorteeDuRadar)` SANS appelant de
production (V2.5 [!]).

**V2.5 — pourquoi l'arrêt, et les deux voies à trancher.** `api/wire` ne construit ni le
collecteur ni le moteur de sync : le hook de l'étape 1.57 est créé par `sync.NewSyncEngineForTitle`
(volontairement « pas au wiring »), les moteurs du serveur par `scheduler.BuildEngine` et
`cmd/server/sync_v2_wiring.go`, et les deux passes CLI (dont le rattrapage V5.2) construisent leur
collecteur dans `cmd/levelup`. La table vient bien de `regulation.toml` (`[radar_range_m]`,
`mappings.RegulationSet.RadarRangeMap`), chargée pour la lecture par `api/server_apiv1.go` puis
`ServiceRegistry.WithRadarRange`. Deux voies, aucune choisie :
(a) `CaptureDepuisCatalogue` (chemin commun aux trois collecteurs, où libellés et objectifs sont
déjà chargés) charge aussi `regulation.toml` par le même chargeur et pose la portée ; test AST sur
les sites de construction du collecteur (`killcollector`, `cmd/levelup`) plutôt que sur `api/wire` ;
(b) faire descendre la table depuis `server_apiv1` jusqu'aux deux fabriques de moteur puis au hook,
et la charger à part pour la CLI. Dans les deux cas, la résolution « variante nettoyée, portée > 0 »
existe déjà en DEUX copies (`TacticalService.rayonsParMatch`, `teammates.rayonParMatchDuScope`) :
un troisième site impose de la centraliser avec un garde-rail (règle 6 du dépôt).

**Décisions prises dans le cadre du lot (à relire).**
- Le placement ne s'écrit que si les vies viennent d'être écrites : jamais une ligne contre une
  vie absente de `match_lives` (échec d'écriture des vies, match sans équipes).
- Pont non publiable : refus AVANT la lecture des porteurs (aucun coût payé), compteur
  `killsource_placement_pont_non_publiable` en plus des six du plan.
- Compteurs (ADR 0009) : `killsource_placement_matchs_couverts`, `_vies_ecrites`,
  `_vies_non_mesurees`, `_frags_hors_vie` (avant la première vie + tueur sans vie),
  `_erreurs_ecriture`, `_matchs_sans_portee`, `_pont_non_publiable`, `_porteurs_lus`,
  `_porteurs_sans_calage`. Info au succès (bilan complet des frags et des porteurs), Error sur
  échec d'écriture, Debug sans capability (message de la porte `CapFilmKillPositions` élargi).
- Validation du persister : bornes et durée cohérentes, cumuls non négatifs couvrant la vie
  (grille fermée), médiane seulement avec du temps mesuré, `radar_m` et `beyond_ms` nuls ENSEMBLE,
  hors radar ≤ mesuré, frags ≥ 0.
- Tueur d'un frag = `FeedKillerXUID` (le fil), comme le témoin V1.

**Témoin V2.7** (copie de base recopiée et migrée dans le dossier du test, cache de films,
câblage de production `CollectMatch`, portée lue dans `regulation.toml` ; calcul pur nourri
INDÉPENDAMMENT : faits de match par `ReplayFactsRepo`, catalogues relus) :

| Film | Variante | Vies écrites / `match_lives_latest` | Non mesurées | Porteur (ms) | Portages lus | Égalité |
|---|---|---|---|---|---|---|
| `8e376cb1` | Team Slayer:Arena | 102 / 102 | 0 | 0 | aucune lecture | 102 lignes, champ à champ |
| `ab526724` | CTF:Arena | 150 / 150 | 5 | 124 100 | statborg, équipes, objets du monde : 31 | 150 lignes, champ à champ |

```bash
export CGO_ENABLED=1 GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-emprise/.gocache
EMPRISE_V2_FILMS=8e376cb1-8885-4ed5-a942-fca601e86620,ab526724-3684-4335-b759-a18edcccc137 \
EMPRISE_V2_DB=$SP/shared_copy.duckdb EMPRISE_V2_CACHE=C:/Users/Guillaume/Projects/LevelUp/data/cache \
  go test -tags=integration ./internal/sync/killcollector/ -run '^TestEmpriseV2Temoin$' -v -count=1   # PASS 99,2 s
```

**Mutations** (appliquées une à une par copie dans le scratchpad, test ciblé, restauration par
copie vérifiée par `cmp`) :

| # | Test | Mutation | Rouge |
|---|---|---|---|
| P1 | `LifePlacementPersister_AllerRetour` | un pointeur nil s'écrit 0 | « les nil doivent se relire NULL » |
| P2 | `…_LaVueRendLaDernierePasseEntiere` | vue partitionnée par (match, xuid, début) | m1 = 3 lignes au lieu de 1 |
| P3 | `…_PasseVide_RienNEstEcrit` | passe vide refusée | erreur « passe vide » |
| P4 | `…_Refus` | contrôle de couverture des cumuls retiré | « cumuls courts » acceptés, 1 ligne écrite |
| G1 | `TestNoRawDeleteOnAppendOnlyTables`, `TestNoMutationOnAppendOnlyStateTables` | `DELETE FROM match_life_placement` dans le persister | 1 violation chacun |
| G1b | `TestNoARTPatternsOnProtectedTables` | `ON CONFLICT (id) DO UPDATE` sur l'INSERT | 1 violation |
| G2 | `TestCompaction_SchemaReel_BoutABout` | entrée retirée du registre de compaction | rouge |
| G3 | `TestNoRawAppendOnlyReads` | lecture brute `FROM match_life_placement` en `platform/duckdb` | rouge |
| C1 | `ProjeterPlacementDesVies_PontNonPubliable_…` | garde du pont retirée | panique du writer |
| C2 | `…_EchecDEcriture_NEstPasBloquant` | compteur d'échec retiré | 0 au lieu de 1 |
| C3 | `PortagesDuMatch_HorsModeAPorteur_RienNEstLu` | retour anticipé de la garde de `PortagesAuSync` retiré | bilan « sans calage » |
| C4 | `ToPlacementRows_…` | non situé / coéquipier non situé croisés | ligne 111 fausse |
| C5 / C5b | `JournalDuPlacement_…` | publiabilité forcée / tueur = assistant | frag publiable d'une passe non publiable / tueur 999 |
| C6 | `PorteeDe` | `connue` ignoré | portée d'une variante inconnue |
| C7 | `SoclesDe_…` | liste vide au lieu de nil hors catalogue | `[]` |
| C8 | `ProjeterPlacementDesVies_EcritUneLigneParVie` | `RadarM: nil` | portée NULL |
| C9a / C9b | `TestEmpriseV2Temoin` (`ab526724`) | appel du placement retiré / variante vide transmise | 0 ligne pour 150 vies / porteur 0 contre 1 800 ms (ligne 5) |
| C10 | `BacklogAJour_IgnoreLePlacementDesVies` | le backlog lit le placement | 2 matchs au backlog |
| C11 | `MatchsAJour_ExigeLePlacementDesVies` | clause du placement retirée | 2 matchs dits à jour |
| C12 / C12b | `EntreeDesPorteurs_PorteCeQueLaPasseALu` | feuille / variante non transmises | rouge |
| C13 / C13b | `SharedRoster_LitVarianteCarteEtFeuille` | `map_id` non nettoyé / feuille non remplie | rouge |
| S1 | `EquipeDuSocle_…`, `SoclesDeDrapeau…` | label neutre ignoré | équipe 0 au lieu de −1 |

Premier essai de C10 NON rouge : la mutation écrite excluait les matchs au lieu de les ajouter ;
réécrite dans le bon sens. Premier essai d'une mutation de la feuille (`Lignes: nil`) sur le témoin
NON rouge, ni sur `ab526724` ni sur `b4f9064c` (Oddball) : le pont par manche rend les mêmes
portages sans le triplet sur ces films. D'où la couture `entreeDesPorteurs` et son test (C12),
motif d'`entreeDuRegistre`.

**Gate** (depuis `apps/go-api`, `CGO_ENABLED=1`, `GOCACHE` du worktree, une commande `go` à la
fois) :
- `go test` en trois lots (état final) : `./internal/sync/... ./internal/games/...
  ./internal/replaybuild/... ./internal/archlint/...` 50 ok ; reste d'`internal` 102 ok ; hors
  `internal` 38 ok ; aucun FAIL. Au premier passage (avant la couture), `api/handlers`
  `TestSettingsHandler_PostMediaReset_OK` a dépassé son délai de 200 ms sous charge, vert seul et
  5 fois de suite, puis vert au passage final (§7).
- `go test -tags=integration -p 1 ./internal/sync/...` : 11 ok (9 min 57 s, premier passage ;
  seul `killcollector` a changé depuis, rejoué seul : ok) ; `./internal/persist/...
  ./internal/migration/... ./internal/games/halo_infinite/migrations/... ./internal/api/wire/...
  ./cmd/levelup/...` : 5 ok. Au passage final, `killcollector`
  `TestRosterDesFilms_AnnuaireContreJointure` a rendu un facteur 9,86 pour un seuil de 10 ; rejoué
  3 fois vert (10,1 à 12,0), et 3 fois sur le roster D'ORIGINE (10,4 à 12,9) : banc à la limite
  de son seuil avant le lot (§7) ; paquet rejoué entier : ok.
- `make go-api-lint` (golangci-lint présent, `--new-from-merge-base=origin/main`) : 0 issues.
- `go test ./internal/archlint/...` : ok. `gofmt -l internal cmd` : vide.

### V2b — Corrections du lot V2 (Go, superviseur → exécuteur) · moyen

Décisions du superviseur du 2026-09-29, sur le rapport V2 vérifié sur pièces :

- [x] V2b.1 **Portée du radar par la capture** (V2.5 tranchée, voie (a)) :
  - helper unique dans `games/mappings` (ex. `PorteeDuRadar(table map[string]int, variante string) (float64, bool)` : clé nettoyée par `strings.TrimSpace`, valeur > 0) ;
  - `TacticalService.rayonsParMatch` et `teammates.rayonParMatchDuScope` l'appellent (comportement inchangé, leurs tests verts sans retouche) ;
  - garde-rail (test grep, auto-testé sur les deux anciennes copies) : aucune autre lecture de la table des portées par variante hors du helper ;
  - `CaptureDepuisCatalogue` charge `regulation.toml` du titre par le même chargeur que `api/server_apiv1.go:1233-1238` (best-effort journalisé, comme libellés et objectifs) et l'application de la capture pose `AvecPorteeDuRadar` : les trois lieux de naissance du collecteur l'ont sans code de plus ;
  - tests : la capture sur la configuration réelle du dépôt rend 18 m pour une variante d'Arène et 24 m pour une variante BTB ; un test qui échoue si l'application de la capture cesse de poser la portée ; variante absente → `radar_m` NULL (déjà couvert en V2, vérifier).
  → helper `mappings.PorteeDuRadar` (`games/mappings/portee_du_radar.go`), les deux copies migrées
  (tests des services verts sans retouche), garde-rail `archlint/no_local_radar_range_lookup_test.go`
  (auto-testé sur les trois anciennes formes, copie de test du témoin comprise), chargeur
  `mappings.LoadRegulationForTitle` (celui du registre, que `LoadFromConfigDir` appelle désormais),
  `DepsCapture.Portee` posée par `AvecCapture` via `AvecPorteeDuRadar` (gardée : c'est le moyen
  d'application). Variante absente → NULL : vérifié (`TestProjeterPlacementDesVies_EcritUneLigneParVie`,
  seconde passe). Journal V2b.
- [x] V2b.2 **Rattrapage convergent** : un match au pont non publiable écrit quand même ses lignes
  (V1 amendée) — chaque instant de la vie compte en `unplaced` (les positions ne s'attribuent à
  personne : le joueur n'est pas situé), `measured_ms = 0`, médiane NULL, frags rattachés comme
  ailleurs ; le compteur `killsource_placement_pont_non_publiable` reste. `matchsAJour` converge
  alors sans règle de plus. Test de la règle (rouge sous mutation) et test de convergence de
  `matchsAJour` sur ce cas.
  → règle dans `replay.PlacementDesVies` (`sansPont` : tout instant « non situé », porteurs non
  lus par le collecteur) ; tests pur, collecteur sur base, convergence de `matchsAJour`. **Réserve
  (§7)** : les 3 matchs de la copie cités par le journal V2 ne sont PAS ce cas et restent candidats
  (cause antérieure au lot V2, journal V2b).
- [x] V2b.3 Gate du lot V2 rejoué en entier (commandes du journal V2), dont le témoin V2.7 : les
  lignes du 22/09 portent désormais `radar_m` (18 m en Arène).
  → vert (sorties au journal V2b) ; témoin : 18 m sur `8e376cb1` et `ab526724`, `beyond_ms` non
  NULL, part hors radar 6,46 % et 10,75 %.
- Gate : celui de V2.

#### Journal V2b (exécuteur, 2026-09-29)

**Fichiers.** Neufs : `games/mappings/portee_du_radar.go` (+ `_test.go`),
`archlint/no_local_radar_range_lookup_test.go`, `sync/killcollector/capture_portee_test.go`.
Modifiés : `games/mappings/registry.go` (`RegulationPath`, `LoadRegulationForTitle` ; le registre
lit son `regulation.toml` par `RegulationPath`, comportement identique),
`service/tactical_service_isolement.go` (`rayonsParMatch`), `service/teammates/teammates_squad_isolement.go`
(`rayonParMatchDuScope`), `sync/killcollector/capture.go` (`DepsCapture.Portee`, `porteeDuTitre`,
`AvecCapture`), `collector.go` et `placement_des_vies.go` du collecteur (commentaires, porteurs non
lus sur pont refusé, compteur du pont déplacé à la publication, champ `pont_non_publiable` du
journal), `film/replay/placement_des_vies.go` (+ `_test.go`), tests du collecteur (unitaire,
intégration, témoin), `cmd/levelup/cmd_backfill_killsource_selection.go` (godoc de `matchsAJour`)
et son test d'intégration.

**V2b.1 — comment la portée atteint les trois lieux de naissance.** `CaptureDepuisCatalogue`
(appelée par l'étape post-sync `postsync.go:173`, `backfill-killsource` et `--online` via
`cmd_backfill_killsource_positions.go:55`) charge `regulation.toml` par
`mappings.LoadRegulationForTitle` — la fonction que le registre des mappings de `server_apiv1`
emploie (`LoadFromConfigDir` → `loadRegulationIfExists(RegulationPath(…))`), sans charger les
autres manifestes — et pose `DepsCapture.Portee` (fermeture sur `RadarRangeMap()` résolue par
`mappings.PorteeDuRadar`). `AvecCapture` l'applique par `AvecPorteeDuRadar` ; les trois sites
appliquent la capture (garde-rail `no_collecteur_sans_capture_test.go`), aucun code de plus.
Fichier illisible ou absent : `Warn` et aucune portée (lignes NULL, compteur « sans portée »).
Aucune comparaison de slug. Le témoin n'injecte plus la portée : elle vient de la capture, et
`v2CritereDeLaPortee` la confronte à une lecture indépendante (`LoadRegulationFromFile` + helper).

**V2b.2 — la règle.** `PlacementDesVies` ne refuse plus : sans pont publiable, `mesureDesVies`
porte `sansPont` et `classer` rend « non situé » AVANT toute autre cause (le portage et la vitalité
s'attribuent par le même pont) ; mêmes grille fermée, portée recopiée (hors radar 0), frags par
`rattacherLesFrags`, `PontNonPubliable` au bilan. Le collecteur ne lit plus les porteurs sur un
pont refusé (`portagesDuMatch`), écrit, et compte `killsource_placement_pont_non_publiable` à la
publication. Le persister accepte ces lignes sans changement (cumul non situé = grille de la vie).

**Réserve sur la population réelle (vérifiée sur la copie).** Les 3 matchs du journal V2
(`03af54c3`, `50247b26`, `13b00e35`, BTB:Slayer) ne sont PAS des ponts non publiables à vies
écrites : leurs vies datent d'une passe du 2026-09-12 à `isolement-2026-09-10-pont-a-l-instant`,
et la passe actuelle n'établit aucun pont (`13b00e35` rejoué : « positions — passe ignorée : pont
slot->xuid vide », aucune vie écrite, donc aucun placement). La clause d'isolement de `matchsAJour`
(sans le placement) les re-sélectionnait DÉJÀ avant V2 (requête rejouée sur la copie : aucun des
trois à jour). La règle V2b.2 couvre le cas `PontEtabli` mais `IndexDisagreements > 0` (vies
écrites, contexte des morts refusé) ; sur la copie, 0 match à vies de la révision courante n'est
sans contexte de mort. §7.

**Mutations** (script de copie dans le scratchpad, test ciblé, restauration vérifiée par `cmp`) :

| # | Test | Mutation | Rouge |
|---|---|---|---|
| R1 | `TestPorteeDuRadar` | clé non nettoyée | `"  Slayer:Arena\t"` : (0, false) |
| R2 | `TestPorteeDuRadar` | portée ≤ 0 acceptée | `Casse` (0, true), `Negative` (−3, true) |
| R3 | `TestAvecCapture_PoseLaPorteeDuRadar` | `AvecCapture` sans `AvecPorteeDuRadar` | portée nil |
| R3b | `TestEmpriseV2Temoin` (`8e376cb1`) | même mutation | « ligne 0 : portée attendue 18 m, radar nil » |
| R4 | `TestCaptureDepuisCatalogue_PorteeDuDepot` | la capture ne pose pas `Portee` | « capture sans portée » |
| R5 | `TestPorteeDuTitre_BestEffort` | dégradation retirée | une portée sur fichier absent |
| R6 | `TestLoadRegulationForTitle_LeChargeurDuRegistre` | chemin sans `mappings/` | (nil, nil) |
| R7 | `TestNoLocalRadarRangeLookup` | copie réintroduite dans `rayonsParMatch` | 1 violation |
| R8 | `TestNoLocalRadarRangeLookup_ReconnaitLesCopies` | empreinte 2 retirée | copie du témoin non reconnue |
| N1 | `TestPlacementDesVies_PontNonPubliable`, `…_PontNonPubliable_EcritLesVies` (base), `TestMatchsAJour_PontNonPubliable_Converge` | règle V1 d'origine (aucune ligne) | 0 vie sur 3 / 0 ligne / 0 vie |
| N2 | `TestPlacementDesVies_PontNonPubliable` | garde `sansPont` retirée de `classer` | 10 100 ms « équipe à terre » |
| N3 | `…_PontNonPubliable_EcritSansLireLesPorteurs` | porteurs lus sur pont refusé | bilan : drapeau gardé, statborg, équipes, monde lus |
| N4 | `…_PontNonPubliable_EcritLesVies` | compteur du pont retiré | 0 au lieu de 1 |
| N5 | `TestPlacementDesVies_PontNonPubliable` | refus non dit au bilan | `PontNonPubliable` faux |
| N6 | `TestMatchsAJour_PontNonPubliable_Converge` | `matchsAJour` exige `measured_ms > 0` | reste candidat |
| N7 | `…_PontNonPubliable_EcritSansLireLesPorteurs` | refus d'écrire rétabli au collecteur | 0 échec d'écriture (non tentée) |

Premier essai de N3 NON rouge : avec un film nil les lectures ne paniquent pas (elles rendent
vide) ; le test pince désormais le bilan de `portagesDuMatch` (vierge sur pont refusé).

**Gate** (depuis `apps/go-api`, `CGO_ENABLED=1`, `GOCACHE` du worktree, une commande `go` à la
fois) :
- `go test` en trois lots : `./internal/sync/... ./internal/games/... ./internal/replaybuild/...
  ./internal/archlint/...` : tout ok sauf `sync/skill` `TestLUSRV2Shadow_RafalesBornees_300Candidats`
  (2,01 à 2,06 s pour 2 s, §7 V1) — rejoué seul : ok ; reste d'`internal` : 102 ok, aucun FAIL ;
  hors `internal` : 38 ok, aucun FAIL.
- `go test -tags=integration -p 1` : `./internal/sync/killcollector/... ./internal/persist/...
  ./internal/migration/... ./internal/games/halo_infinite/migrations/... ./internal/api/wire/...
  ./cmd/levelup/...` : 5 ok, `killcollector` `TestRosterDesFilms_AnnuaireContreJointure` facteur
  9,97 pour 10 (§7 V2) — paquet rejoué seul : ok ; les 9 autres paquets de `./internal/sync/...` :
  ok ; `./internal/sync/` : ok (275,8 s) ; `./internal/service/...` : 5 ok,
  `service` `TestCareerLive_NilAPIResponse_NotCached` échoue une fois (délai de 2 s, §7) — vert
  seul puis deux fois sur le paquet entier.
- Témoin : `EMPRISE_V2_FILMS=8e376cb1…,ab526724…` (commande du journal V2) : PASS 74,9 s ;
  `8e376cb1` Team Slayer:Arena 102/102, 18 m, hors radar 193 700 / 2 997 200 ms (6,46 %) ;
  `ab526724` CTF:Arena 150/150, 18 m, hors radar 346 600 / 3 224 300 ms (10,75 %), 31 portages ;
  lignes égales au calcul pur champ à champ.
- `make go-api-lint` : 0 issues. `go test ./internal/archlint/...` : ok. `gofmt -l internal cmd` :
  vide. Sous `--build-tags=integration` (hors cible) sur les paquets touchés : seul le signalement
  `goimports` antérieur de `postsync_backlog_integration_test.go` (§7 V2).

### V3 — Lecture et contrat (Go) · moyen

- [x] V3.1 Dépôt `platform/duckdb/squad_life_placement_repo.go` : un chargement borné (V11) ; test
  DuckDB `:memory:` (lecture `_latest` seulement, bornes respectées, ligne à portée périmée écartée).
  Le service le consomme par une interface déclarée comme celle du dépôt de l'Emprise (même paquet,
  même motif d'injection `With…`), testée avec un dépôt simulé.
  → `LoadLifePlacement(ctx, matchIDs, xuids)` : une requête sur `match_life_placement_latest`
  (ADR 0026), `match_id` en constante `VARCHAR[]` sous la fenêtre (`clauseListeMatchs`), xuids en
  semi-jointure (`clauseListeParJointure`), `match_registry` pour la variante, aucune lecture de
  `v_gamertag_lookup`, `ORDER BY` total ; table absente → `games.ErrCapabilityNotSupported`. Port
  `port.SquadLifePlacementRepository`, injection `WithLifePlacement`. Tests : 4 sur `:memory:`
  migrée (dernière passe seule, fenêtre bornée par `exigerFenetresBornees`, NULL relus nil, listes
  vides sans requête, table absente, ligne périmée écartée) ; 6 du service sur dépôt simulé.
- [x] V3.2 Calcul pur `analysis/squademprise/placement.go` (+ tests) : par joueur de la composition,
  les vies mesurées (X = `median_m / radar_m`, part hors radar = `beyond_ms / measured_ms`, frags,
  durée, identifiants), médianes, comptes des quatre quarts ; couverture (matchs mesurés, matchs
  sans portée, lignes à portée périmée, vies non mesurées, ms par cause d'exclusion).
  → `Placement(PlacementInput) (*domain.SquadEmprisePlacement, PlacementBilan)` ; 11 tests
  (dont `TestPlacement_ContratJSON`, ajouté par ce lot). Univers : matchs à portée courante connue
  (sinon `matches_without_range`), lignes à portée écrite ≠ courante (ou NULL) écartées et comptées
  (`stale_lives`, matchs nommés au bilan).
- [x] V3.3 Bloc `placement` dans `squad_emprise` (domaine, service `teammates_service_emprise.go`,
  câblage sous `CapFilmKillPositions`) ; `ErrCapabilityNotSupported` → bloc absent, testé.
  → `domain.SquadEmpriseBlock.Placement` (`omitempty`), `TeammatesService.attacherPlacement`
  (`teammates_service_emprise_placement.go`, section de durée `emprise_placement`, appelé par
  `loadUsageBlocks` sur le MÊME `scope` que l'Emprise), câblage `wire/registry_pages_home.go` sous
  `games.CapFilmKillPositions` ; capability absente ou table absente : Debug + bloc absent ; lecture
  en échec : Error + bloc absent ; portée périmée : Warn ; test de câblage par AST
  `TestTeammatesCtx_CableLePlacementDesVies`.
- [x] V3.4 Suppression Go du nuage de Synergies (V7) : champ `SquadEchange.nuage_isolement`,
  `domain/squad_isolement.go`, producteur `service/teammates/teammates_squad_isolement.go` (la
  résolution de portée `rayonParMatchDuScope` et le câblage radar de `TeammatesService` sont
  GARDÉS et déplacés vers leur nouveau lecteur), appel `teammates_squad_echange.go:155`, garde-rail
  `TestSquadNuageIsolement_Contrat` et tests dédiés. `analysis/coordination/` et
  `match_death_context` intouchés (lus par l'onglet Tactique et la vue match).
  → supprimés : le champ, `squad_isolement.go`, `teammates_squad_isolement.go` (+ test), l'appel,
  `TestSquadNuageIsolement_Contrat` ; `rayonParMatchDuScope` et `WithRadarRange` déplacés dans
  `teammates_service_emprise_placement.go` (test `TestRayonParMatchDuScope`) ; `analysis/coordination/`
  et `match_death_context` non touchés ; `TestBuildSquadEchange_JournalRestreintALaComposition`
  exige désormais AUCUNE lecture du contexte des morts. Aucun résidu (grep du gate, §journal V3).
- [x] V3.5 Contrat régénéré (`openapi.yaml`, `generated.ts`), instantané de surface et ratchet de
  contrat à jour.
  → `openapi-gen -check` à jour, `check-generated-types-fresh` OK, `contract-surface.snapshot.json`
  régénéré (schémas `SquadEmprisePlacement*`, deux enums de quart ; retrait de
  `SquadNuageIsolement`, `SquadIsolementMort`, `SquadIsolementRepere` ; les schémas `SquadEmprise*` et
  `SquadObjective*` des lots précédents y manquaient aussi), `lint-contract-ratchet` vert au
  `lefthook run pre-push`.
- [x] V3.6 (déplacé de V4.4 par le superviseur le 2026-09-29 : retirer `nuage_isolement` du contrat
  casse le typage de son seul lecteur, la suppression web part donc avec) — item V4.4 exécuté ici,
  à la lettre.
  → supprimés : `SquadIsolementNuageCard.tsx` (+ test), `squadIsolementNuageOption.ts`,
  `squadIsolement.logic.ts` (+ test), `squadIsolementStrings.ts`, 26 clés `squad.isolement.*` du
  manifeste (`generated/squad.ts` régénéré), types `SquadNuageIsolement` / `SquadIsolementMort` /
  `SquadIsolementRepere` de `types.ts`, montage retiré de `SquadSynergiesPage.tsx` (rangée 1 =
  « Appui » seul, commentaire de section mis à jour) ; test `SquadSynergiesPage` « rangée Appui ».
  Quatre commentaires qui nommaient l'ancienne carte reformulés.
- Gate : gate commun (web compris, et `lefthook run pre-push`) + `go test -tags=integration -p 1 ./internal/service/teammates/... ./internal/platform/duckdb/...`.
  → vert, sorties au journal V3.

#### Journal V3 (exécuteur, 2026-09-30)

**Reprise.** Un premier exécutant a été coupé (quota) avant de statuer le lot : son travail
(non commité, ni plan ni journal) a été AUDITÉ item par item sur pièces, complété et rejoué en
entier. Compilation constatée : `go build ./...`, `go vet`, `tsc -b --force` verts d'emblée.

**Audit (fait par le premier exécutant / complété ici).**

| Exigence | État |
|---|---|
| V3.1 lecture par `match_life_placement_latest` seule ; bornée par matchs (constante `VARCHAR[]`) ET xuids (semi-jointure) ; pas de `v_gamertag_lookup` ; interface + `With…` + dépôt simulé | fait (premier exécutant) |
| V3.2 X = `median_m / radar_m`, quarts (isolé X ≥ 1,0, rentable frags ≥ 1), médianes par joueur, couverture (matchs, sans portée, périmées, non mesurées, ms par cause) | fait (premier exécutant) ; test de contrat JSON ajouté ici |
| Portée périmée écartée ET journalisée en `Warn` ; résolution par `mappings.PorteeDuRadar` seul (garde-rail `no_local_radar_range_lookup_test.go`, commentaire mis à jour) | fait (premier exécutant) |
| V3.3 bloc sous `CapFilmKillPositions`, `ErrCapabilityNotSupported` en Debug, test de câblage AST | fait (premier exécutant) ; test « sans bloc Emprise » ajouté ici |
| V3.4 / V3.6 suppressions Go et web | faites (premier exécutant) ; TROIS commentaires résiduels (`lowSampleNote.ts`, son garde-rail, `portee_du_radar.go`) reformulés ici : le grep de résidu est à 0 |
| V3.5 contrat, instantané, ratchet | fait (premier exécutant), revérifié ici (`-check`, fraîcheur, ratchet) |
| Plan, journal, mutations, gate, ADR 0036 (garde-rail de lecture ajouté à I2) | manquaient : faits ici |

**Fichiers.** Neufs : `analysis/squademprise/placement.go` (+ test), `domain/squad_emprise_placement.go`,
`platform/duckdb/squad_life_placement_repo.go` (+ test), `service/teammates/teammates_service_emprise_placement.go`
(+ test). Modifiés : `analysis/squademprise/input.go` (`PlacementRow`, `PlacementRead`),
`domain/squad_emprise.go` (champ `Placement`), `domain/squad_echange.go` (champ retiré), `port/squad_emprise.go`,
`service/teammates/{teammates_service.go,teammates_service_usage.go,teammates_squad_echange.go}` + tests,
`api/wire/registry_pages_home.go` (+ test), `archlint/no_local_radar_range_lookup_test.go` (commentaire),
`games/mappings/portee_du_radar.go` (commentaire), contrat (`openapi.yaml`, `generated.ts`,
`contract-surface.snapshot.json`, `types.ts`), web (suppressions V3.6, `SquadSynergiesPage.tsx` et son
test, i18n `squad.toml` / `generated/squad.ts`), `docs/adr/0036-page-reads-are-scoped.md`. Supprimés : voir V3.4 et V3.6.

**Forme JSON du bloc** (`squad_emprise.placement`, absent quand la capability manque, que la lecture
échoue ou qu'aucune vie n'est écrite pour la composition sur le périmètre ; parts en unité 0..1,
X en portées de radar) :

```json
"placement": {
  "isolated_from_ratio": 1,
  "productive_from_kills": 1,
  "players": [
    {
      "xuid": "2535...", "gamertag": "JGtm",
      "lives_total": 146, "lives_measured": 141,
      "median_radar_ratio": 0.62, "median_kills": 1,
      "quadrants": [
        {"quadrant": "in_range_productive",  "lives": 70, "share": 0.4965},
        {"quadrant": "isolated_productive",  "lives": 9,  "share": 0.0638},
        {"quadrant": "in_range_costly",      "lives": 52, "share": 0.3688},
        {"quadrant": "isolated_costly",      "lives": 10, "share": 0.0709}
      ],
      "lives": [
        {"match_id": "8e376cb1-...", "start_ms": 1000, "duration_ms": 41200, "radar_ratio": 0.75,
         "out_of_radar_share": 0.0, "kills": 2, "quadrant": "in_range_productive"}
      ]
    }
  ],
  "coverage": {
    "matches_total": 12, "matches_with_placement": 12, "matches_without_range": 0, "stale_lives": 0,
    "lives_total": 299, "lives_measured": 290, "lives_unmeasured": 9,
    "measured_ms": 8900000, "carrier_ms": 124100, "team_down_ms": 96000,
    "unplaced_ms": 310000, "teammate_unplaced_ms": 140000
  }
}
```

(Valeurs d'illustration ; la forme est celle des types `domain.SquadEmprisePlacement*`.) Sans vie
mesurée, un joueur garde ses quatre quarts (`lives` = 0, sans `share`), `lives: []`, et ni
`median_radar_ratio` ni `median_kills`. `radar_ratio` est la VRAIE valeur (le client pose à 2 pour
tracer, l'infobulle garde la vraie) ; `lives` dans l'ordre chronologique du périmètre.

**Décisions prises dans le cadre du lot (à relire).**
- `lives_total` et les cumuls en ms ne comptent que les vies RETENUES (match à portée courante connue,
  ligne à portée non périmée) ; les vies d'un match sans portée ne sont comptées nulle part (le match
  l'est : `matches_without_range`), celles à portée périmée le sont dans `stale_lives`.
- `matches_with_placement` inclut les matchs sans portée et ceux à vies périmées (« matchs qui ont une
  ligne écrite pour la composition »).
- Le bloc est publié dès qu'une ligne d'un match du périmètre existe, même si toutes sont écartées
  (couverture non vide, joueurs à zéro vie) : l'infobulle dit alors pourquoi.

**Mutations** (script de copie dans le scratchpad, test ciblé, restauration par copie vérifiée par
`cmp` à chaque fois ; TOUS les tests NEUFS du lot vus rouges) :

| # | Test | Mutation | Rouge |
|---|---|---|---|
| Q1 | `Placement_XEtPartHorsRadar` | X = médiane sans division par la portée | X = 27 |
| Q2 | idem | part hors radar rapportée à la durée, non au mesuré | 0,125 au lieu de 0,25 |
| Q3 / Q3b | `Placement_QuartsEtBornes` | isolé `>` au lieu de `≥` / rentable `>` au lieu de `≥` | quart de la borne faux |
| Q4 | `Placement_VieNonMesuree` | vie non mesurée non comptée | rouge |
| Q5 | `Placement_MatchSansPortee` | match sans portée non compté | 0 au lieu de 1 |
| Q6 | `Placement_PorteePerimee` | comparaison portée écrite / courante retirée | 1 périmée au lieu de 2 |
| Q7 | `Placement_HorsCompositionEtHorsPerimetre` | filtre du périmètre retiré | 1 ignorée au lieu de 2 |
| Q8 | `Placement_OrdreDesFichesEtChronologie` | tri chronologique retiré | rouge |
| Q9 | `Placement_Medianes` | médiane = première valeur | 0,5 / 0 |
| Q10 | `Placement_VieNonMesuree` | part rapportée aux vies retenues, non aux mesurées | rouge |
| Q11 | `Placement_CouvertureDesCauses` | cumul « porteur » lit la cause « non situé » | cumuls faux |
| Q12 | `Placement_AucuneVie` | bloc publié sans vie | rouge |
| Q13 | `…OrdreDesFiches…`, `…ContratJSON` | `lives` nil au lieu de `[]` | les deux rouges |
| Q14 / Q15 | `Placement_ContratJSON` | `omitempty` retiré de la médiane / de la part | rouge |
| D1 | `SquadLifePlacementRepo_BorneEtDernierePasse` | lecture de la table brute | passes anciennes relues |
| D2 | idem | matchs liés en semi-jointure (fenêtre non bornée) | `exigerFenetresBornees` rouge |
| D3 | idem | joueurs non bornés à la composition | vies de C relues |
| D4 | idem | NULL de la médiane relu comme valeur | rouge |
| D5 | `SquadLifePlacementRepo_ListesVides` | requête émise sur une seule liste vide | requête envoyée |
| D6 | `SquadLifePlacementRepo_TableAbsente` | erreur de table absente non reconnue | « Catalog Error » remontée |
| D7 | `…BorneEtDernierePasse` | variantes non rendues | `map[]` |
| S1 / S1b | `GetPage_Placement_UneLectureBornee…` | xuids tronqués / matchs vides | bornes fausses |
| S2 | `GetPage_Placement_DepotNonSupporte` | `ErrCapabilityNotSupported` non reconnue | bloc publié / journal absent |
| S3 | `GetPage_Placement_LectureEnEchec` | Error rétrogradé en Warn | niveau faux |
| S4 | `GetPage_Placement_UneLectureBornee…` | bloc non attaché | placement nil |
| S5 | `GetPage_Placement_PorteePerimeeJournalisee` | Warn rétrogradé en Info | niveau faux |
| S6 | `GetPage_Placement_CapabilityAbsente` | Debug rétrogradé en Info | niveau faux |
| S7 | idem + `RayonParMatchDuScope` | résolution avec une table vide | rouge (panique d'index) |
| S8 | `GetPage_Placement_LectureEnEchec` | bloc vide publié sur échec | rouge |
| S9 | `AttacherPlacement_SansBlocEmprise_NeLitRien` | garde `bloc == nil` retirée | panique (nil) |
| W1 | `TeammatesCtx_CableLePlacementDesVies` | porte de capability retirée | rouge |
| W2 | idem | autre dépôt câblé | rouge |
| W3 | idem | portée du radar non câblée | rouge |
| L1 | `BuildSquadEchange_JournalRestreint…` | la page relit le contexte des morts | contexte 1, attendu 0 |
| Y1 | `SquadSynergiesPage` « rangée Appui » | seconde cellule dans la grille | 2 cartes au lieu de 1 |

Premiers essais Q7, Q9 et S7 NON rouges (erreur de compilation de la mutation : variable ou import
devenu inutilisé) : réécrits en mutations qui compilent.

**Gate** (depuis `apps/go-api`, `CGO_ENABLED=1`, `GOCACHE` du worktree, une commande à la fois) :
- `go build ./...`, `go vet ./internal/... ./cmd/...` : sans sortie. `go test` en trois lots :
  `sync games replaybuild archlint` tout ok (sync 115 s, grammar 72 s, archlint 61 s) ; reste d'`internal` :
  102 ok, aucun FAIL ; hors `internal` : 38 ok, aucun FAIL.
- `go test -tags=integration -p 1 ./internal/service/teammates/... ./internal/platform/duckdb/... ./internal/api/wire/...` :
  ok (duckdb 260 s, wire 61 s, prestige 35 s, teammates 1 s, sharedprovider 9 s, halo5 0,3 s).
- `make go-api-lint` : 0 issues. `go test ./internal/archlint/...` : ok. `gofmt -l internal cmd` : vide.
- `openapi-gen -check` : à jour ; `tools/check-generated-types-fresh.mjs` : OK.
- Web : `npx tsc -b --force` : 0 erreur ; `npx vitest run --pool=forks` : 825 fichiers ok / 5 sautés,
  8 753 tests ok / 23 sautés (333 s), aucun rejeu nécessaire ; `npm run lint` : 0 erreur (26 avertissements
  existants) ; `knip-ratchet` 0/0/0 ; `lint-cross-feature-imports` 7 ≤ plafond 7 ; `lint-no-hardcoded-colors`
  0 violation ; `lefthook run pre-push` : 9 étapes vertes (go-vet-cgo, govulncheck, knip, contrat, imports,
  couleurs, champs, shared-social, rappel).

### V4 — Cartes (web) · moyen

- [x] V4.1 `features/squad/emprise/placementCharts.ts` : deux constructeurs d'option (§2.1, §2.2),
  en portant les motifs de `squadIsolementNuageOption.ts` (repère du radar, gros point) avant sa
  suppression ; tests des options (bornes, repères, tailles, couleurs par jeton, seuil de 8 %,
  décalage déterministe, plafonds à 2 et à 5).
  → `placementCharts.ts` (`buildPlacementLifeOption`, `buildPlacementQuartsOption`,
  `resolvePlacementColors`) + `placementCharts.test.ts` (34 tests) ; journal V4 ci-dessous.
- [x] V4.2 Cartes `PlacementVieCard.tsx` et `PlacementQuartsCard.tsx`, bloc « Groupés ou isolés »
  monté dans `SquadEmprisePage.tsx` (V8), `empriseSections` / `empriseHasContent` étendus.
  → bloc monté dans `UsageSections` entre « Prendre, et s'en servir » et « Par rapport à
  d'habitude » ; `placementHasLives` (`empriseContent.ts`) : au moins un joueur à une vie mesurée.
- [x] V4.3 Textes FR / EN dans un fichier neuf `emprise/placementStrings.ts`
  (`Record<Locale, …>`) : `empriseStrings.ts` est à 490 lignes, seuil 500.
  → `placementStrings.ts` (+ `placementStrings.test.ts` : trois phrases au plus par aide, FR et
  EN) ; `empriseStrings.ts` non touché.
- [~] V4.4 (fait en V3.6) Suppression web du nuage de Synergies : `SquadIsolementNuageCard.tsx`,
  `squadIsolementNuageOption.ts`, `squadIsolementStrings.ts`, clés `squad.isolement.*` du manifeste
  `squad.toml`, leurs tests, montage `SquadSynergiesPage.tsx:167-176` ; commentaire de section de
  Synergies mis à jour (rangée 1 = « Appui »).
- [x] V4.5 Tests de page : bloc présent sur la soirée témoin (fixture), absent sans le bloc, onglet
  toujours masqué sans aucun contenu.
  → `SquadEmprisePage.test.tsx`, describe « Groupés ou isolés » (5 tests) ; fixture
  `placement.fixtures.ts` (3 joueurs, quatre quarts, une vie au-delà des plafonds).
- Gate : gate commun web + `lefthook run pre-push`.
  → vert, sorties au journal V4.

#### Journal V4 (exécuteur, 2026-09-30)

**Fichiers.** Neufs dans `apps/web/src/features/squad/emprise/` : `placementCharts.ts` (+ `.test.ts`),
`placementStrings.ts` (+ `.test.ts`), `PlacementVieCard.tsx`, `PlacementQuartsCard.tsx`,
`placement.fixtures.ts`. Modifiés : `SquadEmprisePage.tsx` (bloc dans `UsageSections`, en-tête de
page) et son test, `emprise/empriseContent.ts` (`placementHasLives`, `empriseSections` et
`empriseHasContent` étendus), `emprise/useEmpriseModels.ts` (rend `placement`), `lib/api/types.ts`
(quatre alias `SquadEmprisePlacement*` du contrat). Le fichier de test de la page est à 500 lignes
(seuil) : les gabarits de bloc de la suite y sont repris d'un helper (`sansMesure`).

**Décisions du lot (à relire).**
- **Repère du radar : jeton `extreme`.** La spécification dit « couleur d'accent » ; l'accent de la
  maquette est un violet (`--accent`), et l'app n'a pas de jeton `accent` de graphe (`--accent`
  du thème est un gris de survol). `extreme` (fuchsia) est la teinte violette du thème ; `warning`
  (ambre) est celle du joueur 2 et du trait à 50 % des autres cartes, `info` (bleu) celle du joueur 1.
  Un seul point de changement : `resolvePlacementColors`.
- **Encre sombre des valeurs des segments : `--warning-foreground`** (sombre dans les deux thèmes,
  lue au rendu ; repli sur la couleur du texte hors navigateur). Aucun jeton « encre » n'existe.
- **« N vies » des infobulles et taille du gros point = `lives_measured`** (les vies dans les
  médianes et dans les parts) ; « N vies mesurées sur M » de l'aide de carte = `coverage`.
- **Seuils de tracé lus au bloc** : repère du radar à `isolated_from_ratio`, frontière à
  `productive_from_kills − 0,5`, quatre quarts aux mêmes coins ; les bornes d'axe (0-2, −0,5 à 5,5)
  et les plafonds (2, 5) sont ceux de la spécification, en constantes nommées.
- **Valeurs des segments** : la donnée empilée est la part exacte (×100), la valeur écrite est
  arrondie et masquée sous 8 % après arrondi (comme la maquette). Les noms des joueurs sont
  laissés à `containLabel` (`getGridBase`) plutôt qu'à une marge gauche fixe de 80 px.
- **Hachage du décalage** : FNV-1a puis brassage final (`fmix32`) ; FNV-1a seul dispersait mal des
  débuts de vie consécutifs (34 valeurs distinctes sur 50 vues rouge du test de dispersion).
- **Bloc absent** si aucun joueur n'a de vie mesurée (Halo 5, portée du radar inconnue) : même
  prédicat pour la page et pour la barre d'onglets.

**Mutations** (script de copie dans le scratchpad, test ciblé, restauration par copie vérifiée par
`cmp` à chaque fois ; TOUS les tests neufs vus rouges) :

| # | Test | Mutation | Rouge |
|---|---|---|---|
| M1 | axes | `X_MAX` 2 → 3 | bornes et plafond |
| M2 | plafonds | `KILLS_CAP` 5 → 6 | vie à 7 frags |
| M3 / M4 | tailles | diviseur 9 → 10 ; plafond du gros point 200 → 100 | rouge |
| M5 / M5b / M5c | décalage | décalage nul ; `Math.random` ; clé sans `start_ms` | dispersion, hasard, clé |
| M6 / M7 | repères | radar en constante 1 ; frontière sans −0,5 | seuils du bloc |
| M8 / M8b / M14 | quarts | 2e quart teinté ; opacité 30 % ; zone teintée sur le radar | rouge |
| M9 / M10 | légende | légende en haut ; gros point sous un autre nom de série | rouge |
| M11 | seuil de 8 % | 8 → 5 | rouge |
| M12 / M12b / M18 | jetons | perf-tier-3 ; perf-tier-2 pour le 1er ; repère en `warning` | rouge (le test applique la palette : sans elle `resolveToken` rend `''` partout, M18 restait vert au premier essai) |
| M13 | ordre | premier joueur en bas | rouge |
| M15 / M15b / M23 | barres | non empilées ; épaisseur 20 ; séparateur retiré | rouge |
| M16 | infobulle | gamertag non échappé | rouge |
| M17 | axe X | libellés sans deux décimales | rouge |
| M19 / M20 / M21 | plafonds | gros point sans plafond (2e essai : le premier restait vert, la fixture n'avait pas de médiane au-delà) ; frags sans plafond ; radar plafonné dans l'infobulle | rouge |
| M22 / M30 | couleurs | joueur au repli du thème ; principal = 2e joueur | rouge |
| M24 / M25 / M26 / M29 | styles | liseré du semis ; cerclage 2 → 1 ; encre = texte ; variable d'encre claire | rouge |
| M27 / M28 | infobulle | part lue dans le mauvais quart ; durée m:ss fausse | rouge |
| P1 / P2 / P3 / P5 | page | prédicat toujours faux ; ignore les vies ; section sans prédicat ; `empriseHasContent` sans placement | rouge |
| P4 / P4c / P6 / P7 / P8 | page | bloc après habitude ; bloc avant prendre ; cartes côte à côte ; quarts au-dessus ; intertitre non traduit | rouge (P4b, mutation mal écrite, ne déplaçait rien : réécrite en P4c) |
| S1 / S2 / S3 / S4 | textes | 4e phrase ; matchs sans portée oubliés ; seuil de frags oublié (2e essai : la valeur 2 apparaissait aussi dans « 1,25 ») ; titre de quart en minuscules | rouge |

**Gate** (depuis `apps/web` puis la racine du worktree) : `npx tsc -b --force` : 0 erreur ;
`npx vitest run --pool=forks` : 827 fichiers ok / 5 sautés, 8 798 tests ok / 23 sautés (316 s),
aucun rejeu nécessaire ; `npm run lint` : 0 erreur (26 avertissements existants) ;
`knip-ratchet` 0/0/0 ; `lint-cross-feature-imports` 7 ≤ plafond 7 ; `lint-no-hardcoded-colors`
0 violation ; `lefthook run pre-push` : 9 étapes vertes.

**Non fait.** Aucun rendu réel (ni serveur ni navigateur, brief) : le gate visuel est celui de V5.5.

### V5 — Clôture (superviseur) · moyen

- [x] V5.1 Revue adversariale du diff cumulé (skill `adversarial-review` : lot sync / persistance),
  correctifs par lot rouvert, un test de non-régression rouge sous mutation par correction.
  Ronde 1 (2026-09-30, trois relecteurs Sonnet aveugles, contrat de 6 lignes ; diff
  `549d7f7f0..1781ae3c3`) : écritures et sync (L1, L2) — 12 conditions tiennent, 1 constat ;
  calcul et tests (L4, L6) — 14 conditions tiennent, aucun défaut de calcul, 3 trous de test ;
  web (L5, L3, §2) — 30 conditions tiennent, 2 constats. Triage du superviseur :
  - [x] R1 (P1) `placementCharts.ts:342-345` : l'axe des frags montre le libellé « 5.5 » (ECharts
    ajoute une graduation aux bornes de l'étendue ; le formateur ne masque que les négatifs) —
    le §2.1 veut des graduations entières de pas 1. Correction : n'écrire que les entiers ≥ 0.
    Fait 2026-09-30 : `placementCharts.ts` (formateur : `v >= 0 && Number.isInteger(v)`), test `placementCharts.test.ts` (« Y : … les seuls entiers ≥ 0 », `formatter(5.5)` = vide) ; rouge vu sur le code d'avant.
  - [x] R2 (P1, requalifié : invariant V12 de ce lot) `cmd_backfill_killsource_selection.go`
    `matchsAJour` : vies réécrites (révision d'isolement montée) puis écriture du placement en
    échec → l'ancienne passe de placement, déjà à `PlacementRev`, fait croire le match à jour, et
    l'Emprise lit un placement calculé sur d'anciennes vies. Correction : un match n'est à jour que
    si sa passe de placement à `PlacementRev` est écrite APRÈS (ou avec) sa passe de vies courante.
    Fait 2026-09-30 : `cmd_backfill_killsource_selection.go` (`pl.written_at >= MAX(l.written_at)` des vues `_latest` ; critère `written_at`, `decode_pass` étant aléatoire), test d'intégration `TestMatchsAJour_PlacementPosterieurAuxVies` ; rouge vu sous l'ancienne requête.
  - [x] R3 (P1, requalifié : V1.2 promettait un test par règle) tests manquants de
    `placement_des_vies.go` : frag à l'instant exact du début de la vie suivante (borne
    `[début, début suivant)`) ; paires de causes simultanées porteur/non situé, équipe à
    terre/non situé, non situé/coéquipier non situé ; branches de `fragRecevable` tueur inconnu,
    camp inconnu, tueur sans vie. Chaque test rouge sous la mutation nommée par le relecteur.
    Fait 2026-09-30 : `placement_des_vies_test.go` (`FragALInstantDuDebutDeLaVieSuivante`, `CausesSimultanees`, `FragsEcartesParLeurRefus`) ; 7 mutations rouges (`>` en `>=` ; 3 inversions de blocs de `classer` ; refus tueur inconnu ; `||` en `&&` ; refus tueur sans vie).
  - [x] R4 (P1, même motif) tests manquants de `porteurs_au_sync.go` `portagesDuDocument` : bombe,
    couronne VIP, tri par début.
    Fait 2026-09-30 : `porteurs_au_sync_test.go` (`TestPortagesDuDocument_BombeEtCouronne`) ; 3 mutations rouges (boucle bombe, boucle couronne VIP, tri).
  - R5 (P3, jeté) seuil de 8 % appliqué à la part arrondie : c'est exactement la maquette validée
    (`p.value >= 8` sur la valeur arrondie).
  Ronde 2 (2026-09-30, relecteur Sonnet frais, diff des seules corrections `875992902..cc4f3f483`) :
  **aucun défaut recevable** ; 6 conditions tiennent (types et fuseau de `written_at`, vues
  `_latest` à passe unique, match sans vies ou sans placement, ordre vies puis placement, test R2
  rouge sous mutation, non-régressions et formateur de l'axe verts). P0 + P1 : 4 → 0. Revue close.
- [ ] V5.2 Rattrapage local (serveur arrêté, binaire du worktree, `LEVELUP_REPO_ROOT` = dossier
  principal) : `levelup backfill-killsource --films-only --dry-run` d'abord, la liste doit contenir
  les 12 matchs filmés du 22/09 ; puis la passe réelle, bornée par `--limit` si la liste dépasse la
  soirée témoin et le mois qui la précède.
  Amendé par le superviseur le 2026-09-30 : `--limit` prend les films les MOINS CHERS d'abord, pas
  ceux d'une soirée, et la passe entière du parc local (~1 300 films, ~1 min par film et par
  ouvrier) tiendrait le serveur arrêté des heures. Sous-item :
  - [x] V5.2a Option `--match` de `backfill-killsource` (liste d'identifiants séparés par des
    virgules, préfixes courts de 8 caractères acceptés s'ils sont univoques) : restreint la
    sélection hors ligne à ces matchs, `matchsAJour` toujours appliqué sauf `--force`, refusée
    avec `--online` ; tests (filtre, préfixe ambigu refusé, combinaison avec `--force` et
    `--dry-run`) ; documentée dans `docs/COMMANDS.md` et `docs/FR/COMMANDS.md`.
    Fait 2026-09-30 : `cmd_backfill_killsource_match.go` (nouveau : `validerMatch`, `resoudreMatchs`,
    `registreDeLaPasse`), branchements dans `cmd_backfill_killsource{,_selection}.go` ;
    `afficherPlan(candidats, tout)` liste tout sous `--match` ; refusée aussi avec `--credit-only`
    (elle serait ignorée). Tests `cmd_backfill_killsource_match_test.go` et
    `..._match_integration_test.go` ; 11 mutations rouges (filtre ignoré, ambigu accepté, inconnu
    ignoré, `--online` accepté, préfixe court accepté, `--credit-only` accepté, trim retiré,
    `--force` ignoré, `matchsAJour` ignoré, plan non complet, liste vide acceptée). Docs EN et FR.
    Découverte (non traitée) : `afficherPlan` n'affiche jamais son « ... » sur une longue liste
    (le `continue` de `i == 5` le précède).
  - [x] V5.2b Passe réelle bornée aux 12 matchs filmés du 22/09 par `--match`.
    Fait 2026-09-30 (superviseur ; serveur local déjà arrêté — aucun `air`/`server.exe`, port 8000
    libre —, binaire du worktree `2822130f0`, `LEVELUP_REPO_ROOT` = dossier principal) : la
    migration `shared_match_life_placement_v1` s'applique à l'ouverture (schéma 220) ; dry-run sur
    les 13 préfixes de la soirée : 13 au registre, 1 déjà à jour (`f3061ab7`, Detachment : aucune
    position), 12 films ; passe réelle `--films-only --match …` : 12 films écrits, 1 144 morts,
    0 erreur, 0 absent, 1 min 14 s (3 ouvriers) ; `placement des vies ecrit` sur les 12 matchs,
    portée connue partout, aucun pont non publiable, 0 frag hors vie. Serveur laissé arrêté (il
    l'était avant la passe).
- [x] V5.3 Vérification sur données réelles, soirée du 22/09 : ~146 vies pour JGtm, 80 pour
  Madina97294, 73 pour Chocoboflor (relevé du 2026-09-28 sur `match_lives_latest`) ; part des vies
  mesurées et répartition des causes d'exclusion relevées et collées ici.
  Fait 2026-09-30 (`match_life_placement_latest`, lecture seule, radar 18 m partout) :

  | Joueur | Matchs | Vies | Mesurées | Médiane X | Médiane frags | À portée rentable | À portée coûteux | Isolé rentable | Isolé coûteux | Hors radar | Mesuré | Porteur | Équipe à terre | Non situé | Coéquipier non situé |
  |---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
  | JGtm | 12 | 146 | 145 | 0,44 | 0 | 67 | 71 | 4 | 3 | 9,1 % | 93,3 % | 1,4 % | 1,1 % | 2,3 % | 2,0 % |
  | Madina97294 | 6 | 80 | 79 | 0,43 | 1 | 50 | 26 | 1 | 2 | 9,8 % | 92,2 % | 1,8 % | 1,0 % | 3,9 % | 1,4 % |
  | Chocoboflor | 6 | 73 | 72 | 0,42 | 1 | 36 | 32 | 1 | 3 | 7,5 % | 93,4 % | 0,8 % | 0,9 % | 4,0 % | 1,0 % |

  Nombres de vies identiques au relevé du 2026-09-28. Lecture : en Arène, l'escouade reste groupée
  (médiane à 0,43 portée, 4 à 7 vies isolées par joueur sur la soirée) ; le nuage se tassera à
  gauche du trait du radar, ce que le gate visuel dira.
- Ordre des items suivants (superviseur, 2026-09-30) : V5.7 (notes de version) passe AVANT la
  fusion V5.4, pour que la fusion porte ses notes ; aucun autre changement d'ordre.
- [x] V5.4 Fusion `wt/emprise` → `feat/v75` (sur accord), CI verte au niveau job.
  → CI verte au niveau job sur `511396d60` : CI push 36747409603 et pull_request 36747424695
  (tous les jobs `success`, E2E ignoré comme d'habitude), Secrets, ADR 0021 Gate et Deploy
  Pre-Check verts.
  2026-09-30 (superviseur) : avant fusion, `go vet ./...` et la suite d'intégration complète
  `-tags=integration -p 1` rejouée en trois morceaux sur `c20d3a1b6` (191 paquets `ok`, 0 échec).
  Fusion en avance rapide, poussée (`549d7f7f0..c20d3a1b6`, pre-push vert). CI du push (run
  36742090869) : tous les jobs Go verts sauf « Go Coverage + Baseline », rouge sur la baseline de
  présence — `go test` y sort 0, mais deux tests de la baseline sont absents :
  `replaybuild::TestIsBombVariant` et `replaybuild::TestAucuneGardeParNomDeVariante`, DÉPLACÉS par
  le lot V1 vers `film/replay` (`TestGardesDeLaVariante_Bombe`, `TestAucuneGardeParNomDeVariante`)
  sans mise à jour de la baseline (le pre-push ne joue pas ce contrôle). Remède prescrit par
  `scripts/check_test_baseline.sh` : les deux entrées repointées, rien d'autre ne bouge (diff = 2
  lignes), contrôle rejoué sur le JSONL de la CI : vert. Job web annulé à sa limite de 20 min
  pendant `playwright install --with-deps` (miroir apt Ubuntu à quelques dizaines de Ko/s ;
  typage, lint, build et 8 798 tests vitest verts dans ce même job) : infrastructure, relancé.
- [ ] V5.5 Gate visuel par l'utilisateur APRÈS la fusion (onglet Emprise, soirée du 22/09 ; rangée
  « Appui » seule sur Synergies) ; il nomme les témoins.
- [!] V5.6 Rattrapage prod : fait par l'utilisateur après le déploiement de la v7.5. Coût à lui
  annoncer : `backfill-killsource` redécode chaque film éligible (mesure consignée : 2 films en
  2 min 13 s, serveur arrêté).
- [x] V5.7 `CHANGELOG` et `RELEASE_NOTES` (EN et FR) de la 7.5 ; journal ; mémoire.
  Fait 2026-09-30 : `docs/CHANGELOG.md` et `docs/FR/CHANGELOG.md` (`[7.5.0]` : Ajouté x3 — onglet Emprise, bloc « Groupés ou isolés », option `--match` ; Ops (4b) placement des vies et phrase Emprise ajoutée à (3) ; deux phrases corrigées), `docs/RELEASE_NOTES.md` et `docs/FR/RELEASE_NOTES.md` (deux entrées, une phrase corrigée) ; entrée `.ai/thought_log.md`. La mémoire (index `MEMORY.md`) est hors du worktree : au superviseur.

## 5. Reprise de session

Lire ce fichier (statuts), puis `git -C ../LevelUp-wt-emprise log --oneline -15`, puis l'entrée la
plus récente du journal. Reprendre au premier item non statué du premier lot non clos.

## 6. Journal (superviseur)

- 2026-09-28 : plan écrit sur deux relevés en lecture seule (vies et trajectoires ; véhicules).
  Faits établis : la table des vies existe (`match_lives`, 1 519 matchs en local, ~107 vies par
  match, horloge du match) et n'a aucun lecteur de page ; aucune mesure par vie n'existe ; 97,6 %
  des frags du 22/09 tombent dans une vie du tueur, 1,5 % dans les 250 ms qui suivent sa fin ; la
  portée du radar est câblée côté lecture ; le porteur d'objectif n'existe qu'à la cuisson.
- 2026-09-28 : GO utilisateur (« oui go », puis « fais tout ça en autonomie »). V0 clos
  (`0ab34fe83`) : V0.1 atteint (96,27 %) ; STOP V0.2 (voie (a) infidèle, voie (b) fidèle mais
  +59,9 % en moyenne sur les matchs à porteur) ; le superviseur a vérifié le rapport sur pièces
  (tableaux du journal V0, commit, tests de recherche sautés sans données) et posé la question avec
  les chiffres ; l'utilisateur accepte le coût : voie (b), V6 réécrite, item V1.4 ajouté pour
  l'entrée exportée des porteurs.
- 2026-09-29 : V1 clos (`5d061da28`). Vérifié sur pièces : diff relu (règles V3 dans `classer` et
  `rattacherLesFrags`), tests de `film/replay`, `replaybuild` et `archlint` rejoués verts par le
  superviseur. Décisions de l'exécuteur ACCEPTÉES (dans le cadre de V3) : grille fermée
  `[start, end]` (le plan l'écrit ainsi) ; pont non publiable → aucune ligne (comme
  `ContextesDesMorts`) ; frag à camp inconnu écarté et compté ; joueur sans camp = équipe à terre ;
  drapeau : seul l'état `carried`. Témoin : 100 % des frags rattachés, 100 % des fins de vie à
  moins de 1 m du contexte de mort ; porteurs identiques au rejeu sur les 9 films.

## 7. Découvertes (à consigner ici, pas à traiter)

- `CLAUDE.md` cite `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` ; le fichier vit sous `.ai/V7.5/`.
- Les ressources de l'Emprise (bonus, armes spéciales) viennent de dérivations de l'artefact de
  rejeu (`sync/replayartifacts/derivations.go`) : leur population est celle des matchs cuits, pas
  celle du sync. Écart à la décision du 2026-09-07, à statuer hors de ce plan.
- (V0, 2026-09-28) `f3061ab7` (22/09, Team Slayer, Detachment) A UN FILM au cache local : 32 morceaux
  et un manifeste finalisé (morceau des temps forts présent). Le brief du lot disait le contraire ;
  le lot a mesuré les 12 autres films comme demandé. Film sans porteur : sans effet sur V0.2.
- (V0) Sur les deux BTB:CTF mesurés (`4f77afc1`, `879a4dba`), le calque du drapeau du rejeu ne
  publie AUCUN portage : `coverage.flagCarries.flagFilm = false`, `bursts = 0` (captures 1 et 3,
  vols 2 et 4, socles 6). Le signal de mode par les bursts ne reconnaît pas ces films ; toute voie
  qui reprend ce calque laisserait la cause `carrier` muette sur le CTF en BTB.
- (V0) Le collecteur balaie positions et créations de bipède SANS le profil calibré par le
  kill-feed (`buildPositionRows` : `NewFilmContextForMap(film, &entry, nil)`, aucun
  `PoserProfilDeBalayage`), alors que la cuisson le pose avant TOUS ses balayages
  (`poserProfilPuisCarte`). Deux producteurs du même fait sous deux profils ; effet sur les
  positions non mesuré (les registres rejoués du lot sont conformes à la passe, tous deux sans profil).
- (V0) `BuildHeldObjectCarry` suppose un seul exemplaire de l'objet (« une prise par un autre slot
  borne la période ouverte ») : juste pour la bombe et le crâne, faux pour le CTF à deux drapeaux —
  une des causes de la fidélité de 81,87 % de la voie (a) sur le drapeau.
- (V0) Le godoc de `replaybuild.SansFaitsPersistes` cite un garde-rail
  `TestReglageDuDecodageForceNAQuUnAppelant` qui n'existe pas ; le garde réel est
  `archlint/un_seul_forcage_du_decodage_test.go` (`TestUnSeulForcageDuDecodage`), qui exclut les
  tests — trois tests de recherche appellent le réglage, le godoc dit « un appelant ».
- (V1, 2026-09-28) `PlacementDesVies` refuse un registre au pont non publiable (`PontPubliable`
  faux) : aucune ligne, refus au bilan. Règle NON écrite en V3, ajoutée par cohérence avec
  `ContextesDesMorts` (mêmes positions attribuées par les mêmes vies). Conséquence : sur un tel film,
  `match_lives` aura des lignes et `match_life_placement` aucune. À confirmer par le superviseur.
  → Amendée le 2026-09-29 (V2b.2) : une ligne par vie, entière « non située ».
- (V1) Garde du drapeau au sync : la cuisson pose l'entrée du drapeau sur TOUT film et laisse les
  trois signaux du film trancher ; le sync ne paie les lectures du drapeau que si la variante est de
  la famille CTF (`GardesDesPorteurs.Drapeau`), puis les mêmes signaux tranchent. Un film que ses
  signaux disent CTF sous une variante non reconnue CTF aurait des portages au rejeu et aucun au
  sync. Non observé sur les 9 films ; non mesuré sur le parc.
- (V1) `replaybuild.deathInstantsOf` et `replay.deathInstantsOf` sont deux copies identiques
  (antérieures au lot). Le pont déplacé prend des `DeathInstant` (les tests de `replaybuild` y
  mettent des xuids non numériques) : les deux copies restent. Deux copies, sous le seuil de la
  règle 6 ; non traité.
- (V1) `document_chronicle.go:808` (chronique append-only du schéma) cite
  `internal/replaybuild/matchfacts.go (pontParManche)`, déplacé le 2026-09-28. La chronique ne se
  réécrit pas ; le pointeur est désormais historique.
- (V1) `sync/skill` `TestLUSRV2Shadow_RafalesBornees_300Candidats` a dépassé son seuil de 2 s
  (2,02 s) au premier passage du gate par lot (paquets en parallèle), vert seul et au passage
  suivant : test chronométré sensible à la charge.
- (V0) L'en-tête de `film/internal/grammar/e192_i0_catalogue_mesure_research_test.go` donne une
  commande sur `./internal/games/halo_infinite/film/filmdec/`, paquet qui n'existe plus ; le plan
  (§4 V0) cite le fichier sous `filmdec/`.
- (V2, 2026-09-29) **V2.5 inapplicable telle qu'écrite** : `api/wire` ne construit ni le collecteur
  ni le moteur de sync (détail et deux voies au journal V2). Tant que V2.5 n'est pas tranchée,
  toute ligne de `match_life_placement` s'écrit `radar_m` / `beyond_ms` NULL.
  → Réglé par V2b.1 (portée par la capture).
- (V2) La résolution « variante nettoyée → portée > 0 » vit en DEUX copies
  (`service/tactical_service*.go` `rayonsParMatch`, `service/teammates/teammates_squad_isolement.go`
  `rayonParMatchDuScope`, qui le dit dans son godoc) ; le câblage V2.5 en ferait une troisième. Le
  témoin V2.7 en porte une copie de test (`v2Portee`).
  → Réglé par V2b.1 (`mappings.PorteeDuRadar` + garde-rail, la copie de test comprise).
- (V2) Rattrapage : un match dont le pont slot->xuid n'est pas publiable a des vies et aucun
  placement (même refus que le contexte des morts), donc `matchsAJour` le garde candidat à chaque
  passe manuelle. Mesuré sur la copie du 2026-09-28 : 3 matchs sur 1 519 à vies n'ont aucun
  contexte de mort (le signe de ce refus). Le backlog automatique n'est pas touché.
  → Règle réglée par V2b.2 ; MAIS le signe était faux pour ces 3 matchs (entrée V2b ci-dessous).
- (V2) La feuille du match (triplet frags / morts / assistances) n'a changé AUCUN portage sur les
  films témoins `ab526724` (CTF) et `b4f9064c` (Oddball) : le pont par manche s'en passe sur eux.
  Seul le test de couture `TestEntreeDesPorteurs_PorteCeQueLaPasseALu` en garde la transmission.
- (V2) `CaptureDepuisCatalogue` charge désormais aussi `replay_labels.toml` / `weapon_names.toml`
  et `map_objectives.json` (2,8 Mo) à CHAQUE cycle post-sync qui a du travail (comme le catalogue de
  bornes l'était déjà) ; coût non mesuré, aucune mémorisation dans le hook.
- (V2) `killcollector` `TestRosterDesFilms_AnnuaireContreJointure` (seuil de facteur 10) est à la
  limite : facteur 9,86 au passage final du gate, 10,4 à 12,9 sur le roster d'origine. Banc
  chronométré sensible à la charge, antérieur au lot.
- (V2) `api/handlers` `TestSettingsHandler_PostMediaReset_OK` (délai de 200 ms) a échoué une fois
  sous la charge du gate par lots ; vert seul, 5 fois, et au passage final.
- (V2) Sous `golangci-lint --build-tags=integration` (hors cible `make`), deux signalements
  antérieurs au lot : `goimports` sur le bloc d'import de
  `sync/killcollector/postsync_backlog_integration_test.go` (bloc d'import intact ; le lot n'a
  ajouté qu'un test en fin de fichier), `goconst` `name_fr` dans
  `migration/steps_metadata_purge_weapon_families_labels.go`.
- (V2b, 2026-09-29) **Les 3 matchs « à vies sans contexte de mort » de la copie ne convergent
  toujours pas, et ce n'est pas le pont non publiable.** `03af54c3`, `50247b26`, `13b00e35`
  (BTB:Slayer) : vies d'une passe du 2026-09-12 à `isolement-2026-09-10-pont-a-l-instant`
  (révision périmée), positions d'une passe ancienne ; la passe ACTUELLE n'établit aucun pont
  (`13b00e35` rejoué par `CollectMatch` : « positions — passe ignorée : pont slot->xuid vide
  (vies=0 nommees=0 lectures_index=0) »), donc n'écrit ni positions, ni vies, ni placement. La
  clause d'isolement de `matchsAJour` (positions présentes, vies à une révision périmée, équipes
  en base) les re-sélectionnait DÉJÀ avant V2 : non-convergence antérieure au lot, que V2b.2 ne
  touche pas. 8 matchs de la copie ont encore des vies à cette révision périmée. Non traité.
- (V2b) `service` `TestCareerLive_NilAPIResponse_NotCached` (attente de 2 s) a échoué une fois sous
  `-tags=integration -p 1` ; vert seul, puis deux fois sur le paquet entier. Test chronométré
  sensible à la charge, hors du lot.
- (V2b) Le chemin de `regulation.toml` s'écrit à la main en plusieurs endroits antérieurs au lot
  (`sync/replayartifacts/flaggrabsnet.go:73`, `padtiers.go:268` via `TitleMappingsDir`, le témoin
  V2.7) ; le lot ajoute `mappings.RegulationPath` (celui du registre) sans migrer ces sites.
- (V3, 2026-09-30) L'instantané de surface du contrat (`contract-surface.snapshot.json`) ne portait
  ni les schémas `SquadEmprise*` ni `SquadObjective*` des lots antérieurs : sa régénération de V3 les
  a ajoutés en même temps que `SquadEmprisePlacement*`. Aucun garde-rail ne le signalait.
- (V3) L'ordre chronologique de `lives` est celui du périmètre passé au calcul (`perimetreEscouade`,
  « dans l'ordre de filteredMatches ») ; le calcul trie par rang de match dans ce périmètre, il ne
  relit aucun horodatage. Non revérifié que `filteredMatches` est chronologique ; le rendu V4 ne s'en
  sert que pour l'ordre des points (le décalage vertical dérive de la clé de la vie).
- (V3) Rien ne garde l'OBLIGATION de déclarer une section de durée (ADR 0036 I6, « not yet guarded »),
  y compris la section `emprise_placement` du bloc.
- (V3) Un premier passage de mutations (Q7, Q9, S7) a produit des mutations qui ne compilaient pas :
  un « rouge » d'échec de compilation ne prouve rien ; le script `mut.sh` ne le distingue pas d'un
  échec de test (relu à la main).
- (V5.2a) `afficherPlan` (`cmd_backfill_killsource_selection.go`) n'affiche jamais son « ... » sur une
  longue liste : le `continue` de la ligne `i >= 5 && i < len-3` précède le `if i == 5`. Sous `--match`
  la liste est complète, donc sans effet ; défaut cosmétique préexistant, non traité.
- (V5.2a) Un `sed` de mutation `if false {` sur `dejaFaits[id]` ne compile pas (paquet de test en
  échec de build) : remplacé par `&& len(id) < 0`, rouge par échec de test et non de compilation.
- (V5.7) `CHANGELOG` et `RELEASE_NOTES` (EN et FR) décrivent encore l'Escouade pour « Les formes retenues »
  (`features/squad/formes`, 19 cartes) et « Équipement utilisé, gardé, gaspillé » (« Synthèse, Escouade,
  Sessions ») ; les cartes squad de `formes/` ont été supprimées (D12) et `formes_retenues` n'est monté que
  par Contributions et les Séries temporelles. Phrases non réécrites (hors des contradictions frontales), à
  reprendre par le superviseur si l'on veut être exact.
