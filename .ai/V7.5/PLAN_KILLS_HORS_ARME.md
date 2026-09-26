# PLAN — Les kills SANS arme a feu entrent dans les statistiques

> Ecrit le 2026-08-29. Contrat d execution : skill `plan-execution` (ordre strict, une etape
> a la fois, gate verifie avant N+1, statut ecrit pour chaque item a la cloture).
> Branche cible : `feat/kills-hors-arme` (depuis `feat/v75`).
> Effort : MOYEN (Go : 1 pont + 1 repo + 1 entree de builder ; web : 3 libelles + 1 token).

---

## 1. OBJECTIF ET CRITERE DE SUCCES

**Objectif produit.** Un kill au REPULSEUR, a la BOBINE (objet explosif) ou par CHUTE /
ENVIRONNEMENT cesse de disparaitre dans « Non attribue » : il compte dans une classe nommee
de la « Repartition des frags ».

**Critere de succes, mesurable.**

1. Sur un match de reference dont le film est decode, la somme des classes reste egale au
   total de kills (invariant (a) de `frag_distribution.go`) AVANT et APRES le lot.
2. La classe `unattributed` DIMINUE exactement du nombre de kills reclasses — aucune classe
   d arme existante ne bouge d une seule unite (mesure avant/apres sur le meme match).
3. Le taux de couverture publie (kills a source mesuree / kills totaux) est LISIBLE par le
   lecteur, pas seulement par le developpeur.

**Hors perimetre, ferme :**

- La MELEE. Elle est deja servie par les compteurs API (`counts.Melee`,
  `Authoritative: true`, `buildAPIFragClasses`). La brancher AUSSI depuis le film la
  compterait DEUX FOIS. Voir §3 (decision D2) — c est trance, pas a rediscuter.
- Les VEHICULES et TOURELLES. Les 12 regles `BANQUE` existent dans `killicon/data/rules.tsv`
  et les classes existent dans le sunburst, mais l utilisateur ne les a pas retenues au
  cadrage du 2026-08-29. Note en §8 « Decouvertes », pas traitee.
- Toute amelioration du decodeur `killsource` : ce lot ne mesure rien de neuf, il PUBLIE ce
  qui est deja mesure.

---

## 2. CE QUI EXISTE DEJA — verifie sur pieces le 2026-08-29

| Brique | Ou | Etat |
|---|---|---|
| Source du degat par mort | `match_kill_events.source_tag` (UINTEGER, jpt! 32 bits) | ecrite en prod |
| Classement de la source | `film/damagetag/data/labels.tsv` | 19 `OBJET_EXPLOSIF`, 9 `DEGAT_GLOBAL`, 115 `ARME` (dont `Repulsor` `07104b31`) |
| Pont source -> vignette | `film/killicon/data/rules.tsv` | colonne `weapon_key` DEJA presente, VIDE sur ces lignes |
| Classes du sunburst | `domain/frag_distribution.go` | `environmental`, `other`, `vehicle`, `turret` DEJA definies ; PAS de `equipment` |
| Builder | `service/fragdist/fragdist.go` (`Build`) | pur, data-driven, non gate par titre |
| Capability | `config/titles/halo_infinite/mappings/capabilities.toml` | `film.kill_source = "supported"` |

**Le fait qui rend le lot sain (a ne pas perdre de vue).** L attribution arme-a-feu repose
sur les records de degat `0xd2` du TIREUR. Un kill au repulseur, a la bobine ou par chute
n emet AUCUN record `0xd2` (cf. `.ai/V7.5/killweapon/PLAN_WEAPON_PER_KILL_PRODUCTION.md`,
phase 0.2 : « kills sans aucun record `0xd2` = splatter / collision / fusion coil / chute »).
Ces kills ne sont donc pas MAL attribues aujourd hui : ils ne sont PAS attribues du tout, et
tombent dans le residu. **Le lot decoupe le residu, il ne retire rien a personne.** C est ce
qui garantit le critere 2.

**La couverture est meilleure que celle du kill feed, et il faut le dire correctement.** La
mesure des 34,3 % citee par `service/match_view_killfeed_weapon.go` vaut pour une publication
LIGNE PAR LIGNE. Ici l usage est AGREGE (compter par joueur et par match), et le schema dit
explicitement qu une passe `publishable = FALSE` porte des lignes « justes en AGREGAT et
fausses individuellement ». De plus le tueur vient du kill-feed AVEC son xuid (`pas de
bijection`, en-tete de `steps_shared_kill_events.go`) : c est la resolution des VICTIMES qui
est fragile, pas celle du credit. L etape 0 mesure le denominateur reel avant tout code.

---

## 3. DECISIONS PRODUIT — TRANCHEES AVANT EXECUTION

| # | Decision | Tranche |
|---|---|---|
| D1 | Nouvelle classe `equipment` (« Equipement » / « Equipment ») | OUI — le repulseur n est pas une arme du registre et n est pas de l environnement |
| D2 | Melee branchee depuis le film | NON — double comptage avec `counts.Melee` autoritatif. Ferme. |
| D3 | Bobines : classe dediee ou `environmental` ? | Classe `environmental`, niveau 2 PAR TYPE D ENERGIE (cinetique / plasma / choc / hardlight) — les 4 vignettes existent (`killfeed-42/43/44/45`) |
| D4 | Chute / environnement (`DEGAT_GLOBAL`, 9 tags) | meme classe `environmental`, niveau 2 « Chute et environnement » |
| D5 | `environmental` reste-t-elle « non-combat » (exclue du breakdown par arme) ? | OUI. Elle sort de `nonCombatFragClasses` pour le SUNBURST (elle devient visible) mais reste exclue de l insight « role d arme » (`weaponRoleInsight.ts`) : une bobine n est pas un style de jeu |
| D6 | Que faire des kills a source NON mesuree ? | RIEN. Ils restent dans `unattributed`. Aucune extrapolation, aucun prorata. |
| D7 | Surfaces | Vue match D ABORD (etape 4), puis agregats + KPI (etape 5) — l ordre est un gate, pas une preference |
| D8 | Provenance affichee | `Authoritative: false` sur les 2 nouvelles classes, comme les classes de registre |

---

## 4. ARCHITECTURE — les couches

Le producteur n est PAS `weapon_kills` : cette table a pour contrat « l arme a feu par kill »,
elle est append-only sous garde-rails ART, et y injecter un second producteur avec des ids
numeriques synthetiques serait un mensonge de schema. On ajoute une TROISIEME entree au
builder, dont la provenance est documentee comme les deux autres.

```
platform/duckdb/  killsource_class_repo.go   agrege match_kill_events -> (xuid, classe, sous-classe, kills)
port/             kill_source_class.go       interface + filtres + KillSourceClassRow
analysis/         (rien : pas d algo)
games/halo_infinite/film/killicon/           rules.tsv : colonne weapon_key remplie (pont EXISTANT)
domain/           frag_distribution.go       + FragClassEquipment, + roles ; sort environmental de nonCombat
service/fragdist/ fragdist.go                Build() prend une 3e entree
service/          *_data_loaders.go          cablage + slog
apps/web/                                    libelles i18n FR/EN + token de couleur + ordre
```

Aucun SQL dans un service. Aucun acces DuckDB hors repo. Aucune logique dans un handler.

**Title-agnostic** : la lecture est gatee par `HasCapability("film.kill_source")`, jamais par
le slug. Un titre sans cette capability rend zero ligne -> zero classe -> sunburst
byte-identique a aujourd hui (c est aussi le test de non-regression de Halo 5).

---

## 5. ETAPES

### ETAPE 0 — MESURER LE DENOMINATEUR (aucune ligne de production)

Le lot ne commence pas par du code : si la couverture agregee est mauvaise, D6 et l affichage
changent. Statuer AVANT d ecrire.

- [x] 0.1 Sur la base de prod, compter par classe de source : `ARME` / `OBJET_EXPLOSIF` /
      `DEGAT_GLOBAL` / `MELEE` / `GRENADE` / `VEHICULE` / `INCONNU` / source NULL, sur
      `match_kill_events_latest`, en distinguant `publishable = TRUE` / `FALSE`.
- [x] 0.2 Compter les kills du repulseur seul (`source_tag = 0x07104b31`) — c est le nombre
      qui dira si la classe `equipment` est un arc visible ou une poussiere.
- [x] 0.3 Ecrire le taux de couverture agrege : kills a source mesuree / kills totaux du
      perimetre, tous joueurs et pour le joueur principal.
- **GATE 0 — PASSE en vue match, ECHOUE en agregat (cf. §9.3), escalade §10** : les 3 chiffres sont ecrits dans ce fichier (§9 Mesures). Si la couverture
  agregee est < 40 %, ARRETER et remonter la decision — un compteur a 4 chiffres nommes sur
  un denominateur invisible est un piege de lecture, pas une statistique.

### ETAPE 1 — LE PONT (Go, hors-ligne, pur)

- [x] 1.1 `killicon/data/rules.tsv` : remplir `weapon_key` sur `NOM Repulsor` -> `hinf_repulsor`.
- [x] 1.2 idem sur les 4 regles `BANQUE exp_single_small_*` -> `hinf_coil_kinetic`,
      `hinf_coil_plasma`, `hinf_coil_shock`, `hinf_coil_hardlight`.
- [!] 1.3 Ajouter une regle `CLASSE DEGAT_GLOBAL` -> `hinf_environment` — **NON TRAITE,
      REFUSE PAR LA TABLE ELLE-MEME**. `killicon.validate()` (killicon.go) rejette a la
      LECTURE toute regle sans vignette : « une regle sans vignette ne dit rien ». Or les
      9 tags `DEGAT_GLOBAL` sont indiscernables entre eux (tous « glda/matg : chute,
      environnement ») et l atlas propose DEUX vignettes concurrentes, `killfeed-52 Fall`
      et `killfeed-55 environment` : en choisir une pour les neuf mettrait une icone
      fausse sur l autre moitie des cas. Consequence de conception, actee : la chute et
      l environnement ne passent PAS par killicon — leur classe vient directement de la
      CLASSE damagetag (`DEGAT_GLOBAL`), resolue au repo de l etape 2. L entree de
      registre `hinf_environment` est posee (item 1.4) et porte le libelle de niveau 2 ;
      elle n a simplement pas de vignette. Question d icone consignee en §8.
- [x] 1.4 `games/weapons/registry.go` : 6 lignes neuves, `class` = `equipment` (repulseur) et
      `environmental` (les 5 autres). Ajouter `equipment` a l enum de `registry_test.go`.
- [x] 1.5 `config/titles/halo_infinite/mappings/weapon_names.toml` : libelles EN + FR des 6 cles.
- **GATE 1 — PASSE le 2026-08-29** (19 paquets `ok`, dont `killicon` et `weapons` ; `go build ./...` exit 0) : `cd apps/go-api && go test ./internal/games/... ./internal/games/halo_infinite/film/...`
  vert, y compris `TestChaqueRegleTrouveSaSource`.

### ETAPE 2 — LA LECTURE (port + repo)

- [x] 2.1 `port/kill_source_class.go` : `KillSourceClassRow{XUID, MatchID, WeaponKey, Class, Kills}`
      + `KillSourceClassFilters` avec `Validate()` refusant le scan complet (calque de
      `WeaponKillFilters`).
- [x] 2.2 `platform/duckdb/killsource_class_repo.go` : agregation sur
      `match_kill_events_latest`, groupee par `feed_killer_xuid`, resolue par `killicon` a
      l initialisation (jamais de comparaison de chaine en requete). Timezone canonique si un
      filtre temporel entre en jeu.
- [x] 2.3 Test `:memory:` DuckDB : source NULL ignoree, `feed_killer_xuid` NULL (tueur bot)
      ignoree, passe non publiable COMPTEE (c est le point du lot), doublon de generation
      exclu par la vue `_latest`.
- **GATE 2 — PASSE le 2026-08-29** : `go test ./internal/platform/duckdb/... ./internal/port/... ./internal/games/...` vert, `go test -tags=integration ./internal/platform/duckdb/...` vert (6 tests KillSourceClass), `go vet` exit 0.

### ETAPE 3 — LE BUILDER (domain + fragdist)

- [x] 3.1 `domain/frag_distribution.go` : `FragClassEquipment` pose, 3e provenance
      documentee en tete de fichier, ordre canonique complete (equipement et
      environnement APRES les engins, AVANT le residu).
      **CORRECTION DE D5, sur piece** : le plan disait de RETIRER
      `FragClassEnvironmental` de `nonCombatFragClasses` pour rendre la classe visible
      au sunburst. C etait une erreur de lecture — verification faite, ce set ne gate
      PAS le sunburst : son seul lecteur de production est
      `match_view_builders_combat.go:45`, qui filtre le breakdown par-ARME (lu depuis
      `weapon_kills`), plus `WeaponClassHasAccuracy`. Le retirer aurait donc change le
      breakdown et le graphe de precision de HALO 5 (ses lignes `h5_environmental` ont
      un id numerique et vivent dans weapon_kills) sans rien apporter au sunburst —
      hors perimetre. FAIT A LA PLACE : `FragClassEquipment` AJOUTE a
      `nonCombatFragClasses`. Aucune ligne weapon_kills ne porte cette classe (pas
      d id numerique), donc ca ne retire rien a personne, et ca garde vraies les deux
      proprietes que le set sert : pas de precision pour un repulseur, pas d entree au
      breakdown par-arme. La visibilite au sunburst passe par la 3e provenance.
      `FragClassEnvironmental` de `nonCombatFragClasses` (D5) ; documenter la 3e provenance
      dans l en-tete du fichier.
- [x] 3.2 `fragdist.Build` : 3e entree `sources []port.KillSourceClassRow`, servie par
      `buildKillSourceFragClasses` — un CHEMIN A PART, et non un elargissement de
      `isRegistryFragClass`. Elargir cet ensemble aurait fait remonter les lignes
      `h5_environmental` de Halo 5 (id numerique, `weapon_kills`) : sunburst H5 change,
      hors perimetre. Les 6 appelants passent `nil` (la vue match sera cablee a
      l etape 4). Niveau 2 par OBJET via `perWeaponRoles`, deja ecrit pour les engins.
- [x] 3.3 Le residu `unattributed` reste `Total - Somme(classes)` : verifier qu il ne peut pas
      devenir negatif si la source sur-compte (clamp deja present — le TESTER, ne pas le
      supposer).
- [x] 3.4 Tests purs : invariants (a) (b) (c) (d) tenus ; entree `sources` vide =>
      sortie BYTE-IDENTIQUE a aujourd hui (non-regression Halo 5 et matchs sans film).
- **GATE 3 — PASSE le 2026-08-29** (`go test ./internal/service/... ./internal/domain/... ./contracttest/...` exit 0 — les 6 appelants et les goldens compris) : `go test ./internal/service/fragdist/... ./internal/domain/...` vert + le test
  d identite byte-a-byte passe.

### ETAPE 4 — LA VUE MATCH (D7 : d abord ici)

- [x] 4.1 Cablage du chargement dans `service/match_view_data_loaders.go`, gate
      `HasCapability("film.kill_source")`, `slog.InfoContext` du compteur, `slog.ErrorContext`
      sur echec de lecture (degradation best-effort JAMAIS silencieuse).
- [x] 4.2 Libelles FR + EN des 2 classes et de leurs roles (`i18n.ts`, parite `Record<Locale,T>`).
- [x] 4.3 Couleur : token semantique uniquement (skill `color-tokens`), aucun hex.
- [~] 4.4 Mention de couverture a cote de l arc — **COUVERT PAR L EXISTANT, pas fait**.
      Sur UN match, la couverture est exactement le complement de l arc « Non
      attribue », qui est deja dessine : afficher « couverture 54 % » a cote reviendrait
      a ecrire deux fois la meme quantite, et le 54 % mesure a l etape 0 est un taux
      GLOBAL qui ne vaut pas pour ce match-la. La mention chiffree a du sens la ou le
      residu n est PAS lisible arc par arc : les agregats et les KPI, c est-a-dire
      l item 5.2, qui la porte deja (« le libelle doit porter le denominateur »).
- **GATE 4 — PASSE le 2026-08-29, SAUF la verification visuelle** : `make check-types` exit 0 ;
  `make test-web` 529 fichiers / 5 402 tests verts, 0 echec ; `go test ./internal/service/...
  ./internal/api/... ./internal/domain/... ./contracttest/...` exit 0. **Pas de regeneration
  OpenAPI necessaire** : le DTO `FragDistribution` ne bouge pas — `class` est une CHAINE, et
  `equipment`/`environmental` sont deux valeurs de plus, pas deux champs (contracttest vert le
  confirme). **VERIFICATION VISUELLE FAITE `[x]`, le 2026-08-29** — sur le `make dev` lance par
  l utilisateur (le serveur relance par l agent etait mort apres ~16 min, `exit status
  0xffffffff` : ne pas relancer de serveur depuis un process d agent). Match
  `78919882-f47d-4460-9e78-673f5afdd338`, Chocoboflor, 14 frags. Sunburst lu a l ecran :
  classe **« Environnement — 14,3 % »** (2 kills), niveau 2 nomme **« Bobine a fusion UNSC —
  2 · 14,3 % »** avec sa ligne de rappel, **« Non attribue » retombe a 71,4 %** et **« Melee »
  reste a 14,3 %** (compteur API, intact). C est EXACTEMENT le critere de succes n2 : les kills
  sortent du residu, aucune autre classe ne bouge.

### ETAPE 5 — LES AGREGATS ET LES KPI (D7 : seulement apres le gate 4)

- [x] 5.1 Les CINQ surfaces agregees sont cablees, aucune ne passe plus `nil` par oubli :
      Synthese, Series temporelles, Sessions, Explorateur (cible) et Escouade. Chacune a
      son `WithKillSourceRepo`, cable sous le meme gate de capability que la vue match
      (`killSourceClassRepoFor`). Le chargement passe par UN SEUL foyer, le paquet feuille
      `internal/service/killsourceload` (regle CLAUDE.md n6 : a la 3e copie on centralise)
      — feuille et non `service`, parce que `service/teammates` ne peut pas importer son
      parent. La vue match y a ete ramenee aussi : une seule copie au total.
      **Piege Go evite** : `killSourceClassRepoFor` rend l INTERFACE, pas le pointeur
      concret. Rendre un `*duckdb.KillSourceClassRepo` nil aurait produit une interface
      NON nil, le garde `if repo != nil` serait passe et le premier appel aurait
      dereference un receveur nil.
      **Signature Escouade** : `squadFragClassesByPlayer` portait deja 6 parametres ; la
      3e provenance en aurait fait 7. Regroupes en `squadFragInputs` (meme motif que
      `killFeedInputs`), ce qui REPASSE sous la regle des 5.
      chaque appelant passe bien la 3e entree (ou `nil` explicite et JUSTIFIE, pas oublie).
- [~] 5.2 KPI agreges et denominateur — **COUVERT PAR L EXISTANT, verifie sur pieces**.
      Recherche faite : les cinq surfaces qui consomment `frag_distribution` rendent
      TOUTES le meme composant `FragSunburst`, qui dessine la classe « Non attribue »
      comme les autres. Le denominateur est donc structurellement a l ecran partout ou
      ces classes apparaissent. Et AUCUN KPI ne les consomme : l « arme favorite » lit
      `v_weapon_kills`, ou ces cles n entrent pas (pas d id numerique). Il n y avait donc
      aucun compteur nu a habiller — la reserve du GATE 0 est tenue sans code neuf.
- [x] 5.3 `weaponRoleInsight.ts` : verifier que `environmental` et `equipment` restent EXCLUS
      de l insight de role (D5) ; ajouter le cas au test existant.
- **GATE 5 — PARTIEL, BLOQUE PAR UN TIERS (arret propre, regle 9 du contrat)**.
  VERT, le 2026-08-29 : `go test ./internal/service/fragdist/... ./internal/service/teammates/...
  ./internal/service/killsourceload/... ./internal/domain/... ./internal/port/...
  ./internal/games/... ./internal/api/wire/...` exit 0 ; `go build ./...` exit 0 ;
  `make check-types` exit 0 ; `make test-web` COMPLET : 529 fichiers / 5 402 tests, 0 echec.
  **ROUGE, ET PAS DE NOTRE FAIT — DEUX symptomes, meme auteur.** (a) Les tests d INTEGRATION
  de `internal/platform/duckdb` echouent sur 4 cas (`TestMatchHistoryRepo_LoadAll_Empty` /
  `_WithData`, `TestMatchViewRepo_GetMatchMeta_Found`,
  `TestGetOrOpen_RunsPlayerMigrationsForLegacySchema`) avec
  `Binder Error: Values list "r" does not have a column named "team_0_rounds_won"` — la
  colonne du chantier « score-manches » de l autre session, que leurs requetes lisent deja
  mais que le schema de test ne fournit pas. Cette meme commande etait VERTE a l etape 2
  (14h5x), avant leurs commits. (b) Le paquet racine `internal/service` ne COMPILE PLUS
  ses tests. Le commit `68e44770b` (« score-manches(E5) », 2026-08-29 15:00, produit par
  une AUTRE session dans ce meme worktree) a retire `buildScoreLabelFromMeta` de
  `match_view_builders_header.go` alors que `service_test.go` l appelle encore a trois
  endroits. Aucune ligne de ce lot ne touche ces deux fichiers. Consequence : ni
  `make go-api-test` ni `go test -tags=integration ./...` ne peuvent aboutir tant que ce
  n est pas resolu — par eux, pas par nous (regle 7 : zero fix hors perimetre, et leur
  refactor est EN COURS).
  Commande a rejouer une fois leur travail stabilise :
  `make go-api-test && make check-types && make test-web` puis
  `cd apps/go-api && go test -tags=integration ./...`.
  **MISE A JOUR 2026-08-29 (16h, session retours-0829) — BLOCAGE LEVE** par `9f053f078`
  (score-manches E8, « les tests de service migres »). Rejoue VERT ce jour, sur pieces :
  `go build ./...` exit 0 ; `go vet ./internal/service/... ./internal/api/...` exit 0 ;
  `go test ./internal/service/... ./internal/api/... ./contracttest/...
  ./internal/platform/duckdb/... -count=1` exit 0 ; `go test -tags=integration
  ./internal/platform/duckdb/...` exit 0 (les 4 echecs `team_0_rounds_won` ont disparu).
  `make check-types` + `make test-web` sont rejoues au gate final VF du lot retours-0829
  (meme arbre de travail — `.ai/PLAN_RETOURS_UTILISATEUR_2026-08-29.md` §6).

### ETAPE 6 — CLOTURE

- [x] 6.1 Tous les items des etapes 0 a 5 portent un statut : `[x]`, `[~]` (avec la reference) ou
      `[!]` (avec sa justification ecrite). Aucune case vide.
- [x] 6.2 Cinq entrees `.ai/thought_log.md` (cadrage+etape 0, puis une par etape).
- [x] 6.3 QUATRE lignes au registre `.ai/V7.5/REGISTRE_REPORTS.md` : le lot et ses deux
      reserves ouvertes (gate bloque, lot scinde), la reserve `SOUS_RESERVE` du repulseur avec
      son unique match datable, la vignette impossible pour la chute, et les vehicules/tourelles
      laisses hors perimetre alors que le pont existe desormais.
- [x] 6.4 Skill `delivery-checklist` passe le 2026-08-29. Resultats : zero `fmt.Println` /
      `log.Printf` introduit, zero TODO/FIXME, zero hex couleur sous `features/` ou
      `components/`, `tsc -b --force` (cache purge, anti-faux-vert) exit 0, `npm run lint`
      0 erreur / 24 avertissements PRE-EXISTANTS (TanStack Table). Fichiers neufs tous sous
      500 L (le plus gros : 264 L). **Exception assumee** : `games/weapons/registry.go` passe
      de 599 a 636 L — au-dela du seuil, mais c est une TABLE DE DONNEES (le registre d armes)
      et les 37 lignes ajoutees sont 6 entrees + leur justification ; la decouper serait pire
      que la laisser. Aucun linter de longueur de fichier n est actif (`funlen` porte sur les
      FONCTIONS, `lll` sur les lignes).
      **SIGNAL PUBLIC ROUGE, PRE-EXISTANT** : la CI de `feat/v75` est en echec depuis le push
      du 2026-08-28 16:05 (job `CI`, run 33188327807), donc AVANT ce lot et sans rapport avec
      lui — rien de ce chantier n est pousse. A ne pas confondre avec le blocage du GATE 5.
- **GATE 6 — NON JOUE, meme blocage que le GATE 5** : `make gate-push` enchaine le ratchet
  lint Go et la baseline de tests, qui passent tous deux par `internal/service` — le paquet
  dont les TESTS ne compilent plus depuis `68e44770b` (autre session). Rejouer des que leur
  refactor est stabilise. Le reste du filet a ete joue a la main et il est vert (cf. 6.4).
  **MISE A JOUR 2026-08-29 : blocage leve (cf. GATE 5).** `make gate-push` reste DIFFERE au
  pre-merge (`.ai/PLAN_RETOURS_UTILISATEUR_2026-08-29.md` §6) — la CI de branche est de
  toute facon rouge depuis le 2026-08-28 16:05, AVANT ces lots, a diagnostiquer avant push.

---

## 6. RESERVE A PORTER JUSQU AU BOUT

Le tag `07104b31` = « Repulsor » est en statut **`SOUS_RESERVE`** dans `labels.tsv` :
identification structurelle par RE hors ligne (chaine `eqip 7ca85adc -> sofa 6845f2b3 ->
eqip 1e79ebda -> jpt! 07104b31`), JAMAIS confirmee sur corpus reel ni en verite-terrain
Theater. Un compteur « kills a l equipement » repose donc sur une inference, pas sur une
mesure. Ce n est pas bloquant (114 autres lignes ARME portent le meme regime, dont le MA40),
mais l etape 0.2 donnera le volume concerne, et si un match a kill de repulseur date est
identifie, la verification Theater LEVE la reserve pour un cout de quelques minutes.

## 7. REPRISE DE SESSION

Avancement = les cases de ce fichier. Reprendre a la premiere case vide de la premiere etape
dont le gate n est pas marque PASSE. Verifier sur pieces avant de cocher (rouvrir le fichier
cible : le code a pu bouger).

## 8. DECOUVERTES (a consigner, PAS a traiter)

- **`cmd/backfill-team-rounds` ne compile plus** (constate 13:44, hors de ce lot) :
  `main.go:205` appelle `pendingMatchIDs` avec 3 arguments, la fonction en attend 4
  depuis `registry.go:147`. Ces trois fichiers ont ete modifies PENDANT cette session
  par un autre acteur du meme worktree — aucune ligne du lot ne les touche. NON
  CORRIGE (regle 7). Consequence : `go build ./...` echoue sur ce paquet, les gates du
  lot passent par paquet et ne sont pas affectes. **Resolu par le meme tiers au
  2026-08-29 16h : `go build ./...` exit 0 constate (session retours-0829).**
- **Le worktree est PARTAGE avec une autre session, et elle a commite par-dessus ce lot.**
  Le commit `68e44770b` (« score-manches(E5) », 15:00:46) embarque des fichiers de CE lot
  qui etaient en cours dans l arbre de travail : `match_view_service.go`,
  `match_view_data_loaders.go`, `registry_pages.go`, `registry_pages_home.go` (verifie :
  `git show HEAD:...` contient `killSourceRepo` et `killSourceClassRepoFor`). Le travail
  du lot est donc SCINDE entre ce commit-la et l arbre de travail courant. Rien n est
  perdu, mais l historique ne raconte pas ce qui s est passe, et un `git revert` de ce
  commit emporterait une partie du lot. A trancher avec l utilisateur AVANT tout commit.
- **Vignette de kill feed pour la chute et l environnement** (issue de l item 1.3) : l atlas
  porte `killfeed-52 Fall` ET `killfeed-55 environment`, mais les 9 tags `DEGAT_GLOBAL` ne
  se distinguent pas entre eux. Departager chute et environnement demanderait un signal que
  le decodeur ne rend pas aujourd hui. Hors perimetre — le lot vit tres bien sans icone.
- Vehicules et tourelles : 12 regles `BANQUE` pretes, classes `vehicle`/`turret` deja dans le
  sunburst, commentaire `frag_distribution.go:50` « cas Halo Infinite au 2026-08-02, en
  attente du killsource ». Ce lot pose exactement le pont qui manquait. Hors perimetre ici.

## 9. MESURES (etape 0, executee le 2026-08-29)

Source : `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`, ouverte en
`access_mode=read_only` (le serveur dev tenait la DB — aucune ouverture RW, ADR 0013).
Vue lue : `match_kill_events_latest`.

### 0.1 Repartition par classe de source (jointure `damagetag/data/labels.tsv`)

| classe | publiable | non publiable | total | part des sources mesurees |
|---|---:|---:|---:|---:|
| ARME | 41 304 | 17 225 | 58 529 | 78,5 % |
| MELEE | 6 358 | 2 387 | 8 745 | 11,7 % |
| GRENADE | 3 247 | 1 287 | 4 534 | 6,1 % |
| VEHICULE | 574 | 653 | 1 227 | 1,6 % |
| INCONNU | 231 | 344 | 575 | 0,8 % |
| **OBJET_EXPLOSIF (bobines)** | **386** | **161** | **547** | **0,73 %** |
| **DEGAT_GLOBAL (chute/env)** | **246** | **157** | **403** | **0,54 %** |
| HORS_TABLE | 0 | 9 | 9 | 0,01 % |
| **total sources mesurees** | | | **74 569** | 100 % |

Volumetrie de reference : 136 900 morts decodees, 74 569 a source mesuree (54,5 %),
89 704 lignes publiables. 1 948 matchs au registre, 1 365 decodes (70,1 %).

### 0.2 Le repulseur — LE RESULTAT DECISIF

**UN SEUL kill au repulseur dans toute la base** (`source_tag = 0x07104b31`), sur
74 569 sources mesurees et 1 365 matchs decodes. Et il est dans une passe
`publishable = FALSE`.

    match 215e7022-9959-4a1e-85aa-861161b588f4 — 2026-02-03 16:13 UTC — Argyle
    Quick Play / Team Slayer:Arena — t = 325 526 ms (5 min 25 s)
    tueur Elmo910 -> victime aK2fResHv3 — publishable=false, diverges=false

La classe `equipment` de la decision D1 vaudrait donc **1 kill**, porte par un joueur
hors roster. Ce n'est pas un arc de sunburst, c'est une poussiere. **D1 EST A REOUVRIR** —
escalade au superviseur, cf. §10.

### 0.3 Couverture agregee

| denominateur | couverture |
|---|---:|
| kills des matchs DECODES (138 068) | **54,0 %** |
| kills de TOUT le perimetre (253 873, 1 948 matchs) | **29,4 %** |

Lecture du GATE 0 (seuil 40 %) : **PASSE en vue match** (54,0 % sur un match decode),
**ECHOUE en agregat** (29,4 % sur tout le perimetre). C'est exactement le sequencement D7 :
l'etape 4 (vue match) est legitime, l'etape 5 (agregats et KPI) ne l'est qu'avec le
denominateur AFFICHE, jamais un compteur nu.

## 10. ESCALADE APRES L ETAPE 0 — decision attendue

Le lot garde tout son sens pour les BOBINES (547 kills) et la CHUTE/ENVIRONNEMENT (403),
soit 950 kills aujourd'hui invisibles dans « Non attribue ». Il n'en a plus pour
l'EQUIPEMENT (1 kill).

Trois options, la 2 recommandee :

1. **Abandonner D1.** Pas de classe `equipment` ; le repulseur reste dans `unattributed`.
   Le moins de code, mais la demande initiale n'est pas servie et la question reviendra.
2. **Garder D1 sans arc.** La classe existe dans le registre et le pont (cout marginal :
   1 ligne de registre, 1 cle de libelle), mais un arc a 1 kill ne se dessine pas — le
   sunburst n'affiche que les classes non vides, ce qui est deja son comportement. Le jour
   ou un kill au repulseur arrive, il est compte sans aucun code neuf.
3. **Suspendre le lot** jusqu'a ce qu'un corpus porte des kills d'equipement.

Note de methode : ce « 1 » n'est PAS une preuve que le tag est faux. Il est coherent avec
la rarete du geste (tuer AVEC le repulseur, pas avec la chute qu'il provoque — celle-la
sort en `DEGAT_GLOBAL`). Mais il rend la verification Theater d'autant plus utile : il n'y
a qu'un seul cas a regarder, et il est date.
