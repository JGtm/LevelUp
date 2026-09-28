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
> Statut : **GO utilisateur le 2026-09-28** (« oui go »). Exécution en cours.

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
- [!] V2.5 Portée du radar injectée au collecteur (option de câblage dans `api/wire`), même source
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

- [ ] V2b.1 **Portée du radar par la capture** (V2.5 tranchée, voie (a)) :
  - helper unique dans `games/mappings` (ex. `PorteeDuRadar(table map[string]int, variante string) (float64, bool)` : clé nettoyée par `strings.TrimSpace`, valeur > 0) ;
  - `TacticalService.rayonsParMatch` et `teammates.rayonParMatchDuScope` l'appellent (comportement inchangé, leurs tests verts sans retouche) ;
  - garde-rail (test grep, auto-testé sur les deux anciennes copies) : aucune autre lecture de la table des portées par variante hors du helper ;
  - `CaptureDepuisCatalogue` charge `regulation.toml` du titre par le même chargeur que `api/server_apiv1.go:1233-1238` (best-effort journalisé, comme libellés et objectifs) et l'application de la capture pose `AvecPorteeDuRadar` : les trois lieux de naissance du collecteur l'ont sans code de plus ;
  - tests : la capture sur la configuration réelle du dépôt rend 18 m pour une variante d'Arène et 24 m pour une variante BTB ; un test qui échoue si l'application de la capture cesse de poser la portée ; variante absente → `radar_m` NULL (déjà couvert en V2, vérifier).
- [ ] V2b.2 **Rattrapage convergent** : un match au pont non publiable écrit quand même ses lignes
  (V1 amendée) — chaque instant de la vie compte en `unplaced` (les positions ne s'attribuent à
  personne : le joueur n'est pas situé), `measured_ms = 0`, médiane NULL, frags rattachés comme
  ailleurs ; le compteur `killsource_placement_pont_non_publiable` reste. `matchsAJour` converge
  alors sans règle de plus. Test de la règle (rouge sous mutation) et test de convergence de
  `matchsAJour` sur ce cas.
- [ ] V2b.3 Gate du lot V2 rejoué en entier (commandes du journal V2), dont le témoin V2.7 : les
  lignes du 22/09 portent désormais `radar_m` (18 m en Arène).
- Gate : celui de V2.

### V3 — Lecture et contrat (Go) · moyen

- [ ] V3.1 Dépôt `platform/duckdb/squad_life_placement_repo.go` : un chargement borné (V11) ; test
  DuckDB `:memory:` (lecture `_latest` seulement, bornes respectées, ligne à portée périmée écartée).
  Le service le consomme par une interface déclarée comme celle du dépôt de l'Emprise (même paquet,
  même motif d'injection `With…`), testée avec un dépôt simulé.
- [ ] V3.2 Calcul pur `analysis/squademprise/placement.go` (+ tests) : par joueur de la composition,
  les vies mesurées (X = `median_m / radar_m`, part hors radar = `beyond_ms / measured_ms`, frags,
  durée, identifiants), médianes, comptes des quatre quarts ; couverture (matchs mesurés, matchs
  sans portée, lignes à portée périmée, vies non mesurées, ms par cause d'exclusion).
- [ ] V3.3 Bloc `placement` dans `squad_emprise` (domaine, service `teammates_service_emprise.go`,
  câblage sous `CapFilmKillPositions`) ; `ErrCapabilityNotSupported` → bloc absent, testé.
- [ ] V3.4 Suppression Go du nuage de Synergies (V7) : champ `SquadEchange.nuage_isolement`,
  `domain/squad_isolement.go`, producteur `service/teammates/teammates_squad_isolement.go` (la
  résolution de portée `rayonParMatchDuScope` et le câblage radar de `TeammatesService` sont
  GARDÉS et déplacés vers leur nouveau lecteur), appel `teammates_squad_echange.go:155`, garde-rail
  `TestSquadNuageIsolement_Contrat` et tests dédiés. `analysis/coordination/` et
  `match_death_context` intouchés (lus par l'onglet Tactique et la vue match).
- [ ] V3.5 Contrat régénéré (`openapi.yaml`, `generated.ts`), instantané de surface et ratchet de
  contrat à jour.
- Gate : gate commun + `go test -tags=integration -p 1 ./internal/service/teammates/... ./internal/platform/duckdb/...`.

### V4 — Cartes (web) · moyen

- [ ] V4.1 `features/squad/emprise/placementCharts.ts` : deux constructeurs d'option (§2.1, §2.2),
  en portant les motifs de `squadIsolementNuageOption.ts` (repère du radar, gros point) avant sa
  suppression ; tests des options (bornes, repères, tailles, couleurs par jeton, seuil de 8 %,
  décalage déterministe, plafonds à 2 et à 5).
- [ ] V4.2 Cartes `PlacementVieCard.tsx` et `PlacementQuartsCard.tsx`, bloc « Groupés ou isolés »
  monté dans `SquadEmprisePage.tsx` (V8), `empriseSections` / `empriseHasContent` étendus.
- [ ] V4.3 Textes FR / EN dans un fichier neuf `emprise/placementStrings.ts`
  (`Record<Locale, …>`) : `empriseStrings.ts` est à 490 lignes, seuil 500.
- [ ] V4.4 Suppression web du nuage de Synergies : `SquadIsolementNuageCard.tsx`,
  `squadIsolementNuageOption.ts`, `squadIsolementStrings.ts`, clés `squad.isolement.*` du manifeste
  `squad.toml`, leurs tests, montage `SquadSynergiesPage.tsx:167-176` ; commentaire de section de
  Synergies mis à jour (rangée 1 = « Appui »).
- [ ] V4.5 Tests de page : bloc présent sur la soirée témoin (fixture), absent sans le bloc, onglet
  toujours masqué sans aucun contenu.
- Gate : gate commun web + `lefthook run pre-push`.

### V5 — Clôture (superviseur) · moyen

- [ ] V5.1 Revue adversariale du diff cumulé (skill `adversarial-review` : lot sync / persistance),
  correctifs par lot rouvert, un test de non-régression rouge sous mutation par correction.
- [ ] V5.2 Rattrapage local (serveur arrêté, binaire du worktree, `LEVELUP_REPO_ROOT` = dossier
  principal) : `levelup backfill-killsource --films-only --dry-run` d'abord, la liste doit contenir
  les 12 matchs filmés du 22/09 ; puis la passe réelle, bornée par `--limit` si la liste dépasse la
  soirée témoin et le mois qui la précède.
- [ ] V5.3 Vérification sur données réelles, soirée du 22/09 : ~146 vies pour JGtm, 80 pour
  Madina97294, 73 pour Chocoboflor (relevé du 2026-09-28 sur `match_lives_latest`) ; part des vies
  mesurées et répartition des causes d'exclusion relevées et collées ici.
- [ ] V5.4 Fusion `wt/emprise` → `feat/v75` (sur accord), CI verte au niveau job.
- [ ] V5.5 Gate visuel par l'utilisateur APRÈS la fusion (onglet Emprise, soirée du 22/09 ; rangée
  « Appui » seule sur Synergies) ; il nomme les témoins.
- [!] V5.6 Rattrapage prod : fait par l'utilisateur après le déploiement de la v7.5. Coût à lui
  annoncer : `backfill-killsource` redécode chaque film éligible (mesure consignée : 2 films en
  2 min 13 s, serveur arrêté).
- [ ] V5.7 `CHANGELOG` et `RELEASE_NOTES` (EN et FR) de la 7.5 ; journal ; mémoire.

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
- (V2) La résolution « variante nettoyée → portée > 0 » vit en DEUX copies
  (`service/tactical_service*.go` `rayonsParMatch`, `service/teammates/teammates_squad_isolement.go`
  `rayonParMatchDuScope`, qui le dit dans son godoc) ; le câblage V2.5 en ferait une troisième. Le
  témoin V2.7 en porte une copie de test (`v2Portee`).
- (V2) Rattrapage : un match dont le pont slot->xuid n'est pas publiable a des vies et aucun
  placement (même refus que le contexte des morts), donc `matchsAJour` le garde candidat à chaque
  passe manuelle. Mesuré sur la copie du 2026-09-28 : 3 matchs sur 1 519 à vies n'ont aucun
  contexte de mort (le signe de ce refus). Le backlog automatique n'est pas touché.
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
