# PLAN — STATISTIQUES D'OBJECTIF DU MODE ASSAUT, RECONSTRUITES DU FILM

> Ouvert le 2026-09-04. Branche : `wt/assaut-stats` (worktree dedie
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-assaut-stats`), depuis `feat/v75`.
> Contrat d'execution : skill `plan-execution` (ordre strict, aucun report d'une etape
> executable, statut obligatoire par item, verification sur pieces avant de cocher, zero fix
> hors perimetre).

## 1. LE PROBLEME, ET POURQUOI IL EST SOLUBLE

L'API 343 ne publie AUCUNE statistique d'objectif pour l'Assaut. Verifie sur pieces le
2026-08-31 sur le payload `GetMatchStats` des trois variantes : le bundle `Stats` ne porte que
`CoreStats` et `PvpStats`, la chaine « bomb » n'y apparait nulle part
(`objectiveevents/named.go`, commentaire de `ObjectiveTypeBomb`). La table
`match_objective_stats` couvre CTF, Zones, Oddball, Stockpile, Extraction et VIP — et rien pour
la bombe.

Le film Theater, lui, porte tout ce qu'il faut, et le chantier `wt/assaut-bombe` /
`wt/armement-portage` / `wt/onebomb` / `wt/bombe-portee` l'a deja decode et gate :

| Fait | Ou il est lu | Gate mesure |
|---|---|---|
| Explosion datee **et attribuee par joueur** | statborg `comp 0` canal A (`StatBombDetonations`) | 4/4 films, moities disjointes |
| Prises / lachers / periodes de portage par xuid | canal des armes tenues, bombe = famille `0x3fee4fcf` (`held_object_carry.go`) | temoin Oddball 46/46 ; bombe posee portee par personne 27/28 |
| Armement date | anneau `ti=12 i14` (`filmdec/navpoint_radial_scan.go`) | Neutral 3/3, Husky 4/4, One Bomb 9/9 portees (meche pausable) |
| Porteur = poseur | confrontation canal des armes tenues x detonateur statborg | **100 %** sur le denominateur « les deux cotes resolus » (`bombe_portage_gate_test.go`) |

**Ce qui manque n'est pas une mesure, c'est une JOINTURE et une PERSISTANCE.** L'armement est
date par un canal, l'acteur est nomme par un autre, et les deux n'ont jamais ete joints en
production ; rien de tout cela n'entre en base.

## 2. DECISIONS TRANCHEES — ne pas rouvrir en cours d'execution

1. **Table DEDIEE `match_bomb_stats`**, pas des colonnes ajoutees a `match_objective_stats`.
   Raison mesuree : la vue `match_objective_stats_latest` ne garde qu'UNE ligne par
   `(match_id, xuid)`, la plus recente (`migration/objective_stats_view.go`). Deux producteurs
   sur la meme table — le sync API et le film — et le dernier ecrit masque les colonnes de
   l'autre : un re-sync API effacerait les stats bombe de la vue.
2. **Append-only + vue `_latest`** (regle ART n2, ADR 0026), **ecriture INSERT-only via
   `persist.BatchBuilder`** (regle ART n1, ADR 0019/0030). Le
   `ObjectiveEventsRepo.WriteMatch` existant fait un DELETE-then-INSERT et est explicitement
   documente « hors chemin live » : il ne peut PAS etre appele depuis le sync tel quel.
3. **Cinq statistiques par joueur, pas plus** : `bomb_detonations`, `bomb_arms`, `bomb_grabs`,
   `time_as_bomb_carrier_seconds`, `bomb_carriers_killed`. Ecartees a la demande de
   l'utilisateur : `longest_time_as_bomb_carrier_seconds`, `kills_as_bomb_carrier`.
4. **Les evenements dates** (armement, explosion) entrent dans `match_objective_events` avec
   `objective_type = 'bomb'` — valeur DEJA prevue au schema
   (`migration/steps_shared_objective_events.go`). Aucune table neuve pour eux.
5. **Le DESAMORCAGE est HORS LOT.** Neutral/Husky rendent poses completes = explosions 17/17 et
   zero candidate, et les 32 candidates One Bomb se sont revelees etre des armements une fois
   la meche pausable de 16,2 s appliquee.

   **AMENDE LE 2026-09-04 par le handoff — la raison du report a CHANGE, la decision non.**
   Deux corrections sur pieces, l'une et l'autre en faveur du sujet :
   - **Il y a CINQ montees pleines sans explosion, pas une.** L'affirmation « un seul residu
     (`df8fcbef` a 527 s) » ecrite ci-dessus etait fausse : les journaux en donnent cinq
     (`9f57c612` @388 080, `c75f33b8` @196 605, `df8fcbef` @51 901 / @168 619 / @527 367),
     dont deux portent la signature d'armement (lacher a +0,25 s).
   - **LE DESAMORCAGE N'A JAMAIS ETE CHERCHE SOUS SA VRAIE FORME.** Le protocole D1-D3 ne
     cherche que des MONTEES ; la signature d'un desamorcage est une **DESCENTE complete**, et
     aucun instrument ne l'a jamais balayee. « Pas trouve » n'etait donc pas « pas cherche ».
     Mieux : les tenues de desamorcage INTERROMPUES sont DEJA mesurees sur One Bomb (descentes
     251 -> 213/238/218, pente 14-26 quanta/s contre 138 pour une chute d'explosion, l'anneau
     revenant a 251 = tenue relachee).

   Le verrou n'est donc pas seulement le corpus, c'est l'absence d'**ORACLE** : aucun juge
   exterieur ne dit ou est un desamorcage. Reprise : `.ai/HANDOFF_ASSAUT_DESAMORCAGE_2026-09-04.md`
   (6 marches chiffrees, la marche 0 est GRATUITE — les 5 candidats sont deja en cache et
   D1-D3 balaie 9 films en 170 s a 0,02 Gio). Entree au registre des reports.
6. **Branchement au sync dans le hook `replayartifacts`**, qui decode DEJA le film : le cout
   marginal de l'extraction est proche de zero. Aucun second decodage.
7. **Aucune cuisson d'artefacts en lot** pendant l'execution. Le backfill du parc est une
   tache de RELEASE, inscrite au carnet Notion, jamais lancee par l'agent.

## 3. ETAPES

Regle d'ordre : l'etape N+1 ne commence pas avant que N soit close. « Close » = tous ses items
statues (`[x]` fait / `[~]` couvert ailleurs avec reference / `[!]` non traite avec
justification ecrite) ET son gate passe avec sa sortie reelle collee au journal.

### E0 — Cadrage sur pieces (aucune ecriture de production)

- [x] Relire `bombe_portage_gate_test.go`, `bombe_b2_chronologie_test.go`,
      `bombe_b3_desaccords_test.go` : noter le denominateur exact du « 100 % » et la liste des
      4 desaccords (`1c01e34f`, `3d58eb37`, `69b16f5d`).
      **CONSTAT** : le « 100 % » vaut sur `35b75a31` (Neutral, 3 explosions) et `9f57c612`
      (One Bomb, 4 explosions), et son denominateur est « les explosions ou le statborg NOMME
      le detonateur ET le porteur est ponte » — les periodes au slot non ponte sont hors
      denominateur (`coverage.noBridge`), pas comptees comme desaccord. Les 4 desaccords du
      corpus vivent sur les trois AUTRES films, aucun sur les deux du gate : un desaccord neuf
      y serait une regression.
- [x] Verifier que `doc.BombArmings` et `doc.BombCarries` sont bien presents sur `feat/v75` et
      lire leur forme exacte.
      **CONSTAT** : presents (`replay/document.go:848` et `:860`). Le document est en
      `SchemaVersion = 38` (`document.go:549`) — PAS 33/34 comme la note de merge le disait :
      les schemas ont continue d'avancer depuis. Toute note qui cite 33/34 est perimee.
- [x] Verifier la capability d'objectif servie aujourd'hui et decider laquelle porte ces stats
      (`HasCapability`, jamais `slug ==`).
      **DECISION** : `match.objective.stats` gouverne le JOIN sur `match_objective_stats`,
      table alimentee par le SYNC API — elle ne peut pas porter des stats issues du film. La
      convention du depot prefixe `film.*` tout ce qui vient du decodeur (`film.kill_source`,
      `film.weapon_shots`, positions de kill). La nouvelle clef est donc **`film.bomb_stats`**,
      `supported` pour `halo_infinite`, ABSENTE pour `halo_5` (autre format de film).

**Gate E0** : PASSE — les trois constats sont ecrits ci-dessus.

### E1 — Noyau pur d'extraction

Nouveau fichier `apps/go-api/internal/analysis/replay/bomb_stats.go` (<= 500 L, fonctions
<= 80 L). Fonction PURE : entrees deja decodees (`[]StatRecord` / `NamedEvent`,
`HeldObjectCarry`, `[]BombArming`, fil des morts et des kills, pont slot->xuid), sortie
`BombMatchStats` (par xuid) + `[]BombEvent` (dates).

- [x] `bomb_detonations` — reprendre `NamedEventsFrom` + `StatBombDetonations`, pas de
      recalcul ad hoc.
      **FAIT** : `bombDetonationsByXUID` lit `[]objectiveevents.IdentifiedEvent` (la forme de
      `Options.Objectives`, deja nommee par `NamedEventsFrom` et identifiee PAR MANCHE chez
      l'appelant) et filtre sur `objectiveevents.StatBombDetonations`. Aucun decodage, aucun
      recomptage.
- [x] `bomb_grabs` — transitions VERS la famille bombe du canal des armes tenues.
      **FAIT** : compte les periodes de `HeldObjectCarry.Periods` par xuid. Une periode
      s'ouvre sur une prise et une seule (`heldObjectPeriods`), donc periodes = prises ; passer
      par les periodes plutot que par les evenements bruts donne en prime la ventilation
      (sans pont / ouvertes / par mort). La famille `0x3fee4fcf` reste definie une seule fois,
      dans `bomb_carries.go` — elle n'est pas recopiee.
- [x] `time_as_bomb_carrier_seconds` — somme des periodes (`CarryMSByXUID`), periodes fermees
      par mort incluses.
      **FAIT** : lecture DIRECTE de `HeldObjectCarry.CarryMSByXUID` (ms -> s). Ce champ est
      deja alimente par `BuildHeldObjectCarry`, qui inclut les periodes fermees par la MORT et
      exclut les periodes ouvertes ; le recalculer ici aurait cree une seconde definition du
      meme fait.
- [x] `bomb_carriers_killed` — kill dont la VICTIME est en periode de portage a l'instant du
      kill (patron valide 10/10 sur `skull_carriers_killed` en Oddball).
      **FAIT** : `bombCarriersKilledByXUID` croise `[]replay.KillRef` (le type existant de
      `killpos.go` : tueur + victime + instant sur l'horloge du MATCH, exactement celle des
      periodes) avec les periodes de la victime. Borne de fin elargie de
      `bombCarrierKillToleranceMS`, **derivee** de `deathMatchWindowMS` (lives.go) — pas une
      troisieme copie du seuil 150. Ecartes : suicide (tueur = victime) et tueur non nomme.
      RESERVE ECRITE dans l'en-tete : `KillRef` ne porte pas l'equipe, donc un tir ami
      compterait ; la corriger demanderait une entree que ce noyau n'a pas.
      **SON ENTREE EST BRANCHEE LE 2026-09-05 (lot G.6)** : le noyau savait calculer, il lui
      manquait des couples. `replaybuild.killRefs` resout desormais la VICTIME en plus du tueur,
      dans la MEME passe et par la MEME table, et les couples voyagent par
      `replay.Options.MatchKills` — SANS recalage, `killsource.Kill.TimeMS` etant deja l'horloge
      du fil des morts. Premiere mesure reelle : `9f57c612`, 3 porteurs tues sur 58 couples.
- [x] Aucune stat inventee : tout champ sans source mesuree est absent, pas a zero.
      **FAIT** : les quatre champs de `BombPlayerStats` sont des POINTEURS, pilotes par trois
      temoins de lecture (`DetonationsRead` / `CarryRead` / `KillsRead`, patron de
      `KillsInput.Read`). Sans temoin, `BuildBombStats` ne publie AUCUNE ligne — pas une ligne
      de zeros (`TestBombStatsAucuneSourceLue`).
- [x] Tests purs (table-driven, sans film, sans base) pour chacune des quatre.
      **FAIT** : `bomb_stats_test.go`, 5 tests table-driven / 12 sous-cas, aucun
      `ASSAUT_CACHE`, aucun decodage, 0,38 s.

**Gate E1** : `cd apps/go-api && go test ./internal/analysis/replay/ -run BombStats -v` vert,
et `go vet ./internal/analysis/replay/`.

**Gate E1 — PASSE le 2026-09-04** (codes de sortie verifies, pas seulement la sortie filtree) :

```
$ go test ./internal/analysis/replay/ -run BombStats -v   # TEST_EXIT=0
--- PASS: TestBombStatsDetonations (0.00s)
    --- PASS: .../source_non_lue_:_aucun_compte,_aucun_evenement
    --- PASS: .../lue_et_vide_:_zero_evenement,_mais_le_champ_existe
    --- PASS: .../deux_joueurs,_une_autre_stat_ignoree
--- PASS: TestBombStatsGrabs (0.00s)
    --- PASS: .../canal_non_balaye_:_champ_absent
    --- PASS: .../prises_fermees,_par_mort,_ouverte,_et_une_sans_pont
--- PASS: TestBombStatsCarrierSeconds (0.00s)
    --- PASS: .../canal_non_balaye_:_champ_absent
    --- PASS: .../lacher_plus_mort_cumules,_ouverte_exclue
--- PASS: TestBombStatsCarriersKilled (0.00s)
    --- PASS: .../kills_non_lus_:_champ_absent
    --- PASS: .../portage_non_lu_:_champ_absent_meme_si_les_kills_le_sont
    --- PASS: .../kill_pendant_le_portage,_kill_a_la_fermeture_par_mort
    --- PASS: .../hors_periode,_avant_la_prise,_suicide,_tueur_non_nomme
--- PASS: TestBombStatsAucuneSourceLue (0.00s)
PASS
ok      levelup/go-api/internal/analysis/replay 0.377s

$ go vet ./internal/analysis/replay/                      # VET_EXIT=0 (sortie vide)
$ gofmt -l internal/analysis/replay/                      # GOFMT_EXIT=0 (aucun fichier liste)
$ golangci-lint run ./internal/analysis/replay/           # LINT_EXIT=0 — 0 issues.
```

Seuils CLAUDE.md verifies : `bomb_stats.go` 345 L (<= 500), plus longue fonction 29 L
(<= 80), 2 parametres au maximum (<= 5).

### E2 — Attribution de l'armement (`bomb_arms`)

La jointure : pour chaque `BombArming` date, l'armeur est le porteur dont la periode de portage
se ferme par un LACHER dans la fenetre precedant l'armement (le lacher EST le geste de pose,
delai median mesure 4 804 ms contre une meche de 4 930).

- [x] Ecrire la regle de jointure et sa fenetre dans l'en-tete du fichier, AVANT de coder.
      **FAIT** : nouveau fichier `internal/analysis/replay/bomb_arms.go` (191 L). L'en-tete
      pose les QUATRE conditions (periode commencee avant l'armement / fermee par un LACHER /
      dans +/-2 500 ms / la plus proche, et non deja consommee), la derivation du RECALAGE
      d'horloge, et ce que la regle ne couvre pas. La fenetre n'est pas choisie : elle est
      bornee par trois grandeurs mesurees (ecart lacher-armement ~255 ms, residu d'horloge
      <= 114 ms, separation minimale de deux armements >= ~6 s).
- [x] Cas non resolus (slot non ponte, aucun lacher dans la fenetre) : l'armement reste PUBLIE
      SANS acteur. Jamais d'acteur devine. Un compteur de couverture dit combien.
      **FAIT** : `BombEvent{Type: bomb_armed, XUID: ""}` est publie dans les deux cas, et la
      couverture les distingue (`ArmingsNoBridge` / `ArmingsNoDrop`) au lieu de les confondre
      sous « non attribue ». `Arms` reste ABSENT (nil) tant que les DEUX canaux ne sont pas
      lus — jamais un zero. Le compteur est imprime par film au journal du gate ; le
      `slog.InfoContext` de production arrive avec le branchement (E4), le noyau restant PUR.
- [x] Gate de non-regression sur films reels, sous garde `ASSAUT_CACHE`, un seul decodage a la
      fois (`filmdec.LockProcessDecode`) : sur `35b75a31` et `9f57c612`, chaque explosion garde
      son poseur et aucun desaccord neuf n'apparait ; sur les trois films a desaccord, le
      resultat reste celui arbitre par B3.
      **FAIT** : `assaut_bomb_arms_gate_test.go`, criteres (a) a (d) ecrits avant le premier
      run. (a) juge par `bpJugeExplosion` — le juge du gate de portage, a l'identique, donc le
      meme denominateur ; (b) compte les desaccords B2 sur les trois films instruits et les
      compare a la reference figee 4. Sortie reelle collee ci-dessous.
- [x] La somme des `bomb_arms` attribues par equipe ne depasse jamais le nombre d'armements
      dates du film (controle de coherence, publie).
      **FAIT, EN PLUS FORT** : le noyau ne connait pas les equipes (elles ne sont pas une
      entree de `BombStatsInput`), et le controle porte donc sur la somme de TOUS les joueurs —
      ce qui majore toute sous-somme par equipe. Il est publie dans la couverture
      (`ArmingsAttributed`) et verifie a deux niveaux : par construction (une periode consommee
      ne peut plus servir), par le gate reel (`baVerifierCoherence`) et par les tests purs
      (`assertBombArmCoverage`). L'invariant complet est `attribues + sansLacher + sansPont ==
      armements dates`, plus fort que la simple majoration demandee.

**Gate E2** : le test de gate imprime, par film, `armements dates / armements attribues /
non attribues`, et le run est vert. Sortie collee au journal.

**Gate E2 — PASSE le 2026-09-04** (codes de sortie verifies, motifs d'echec ancres sur
`^--- FAIL:`) :

```
(a) TESTS PURS — sans film, sans base
$ go test ./internal/analysis/replay/ -run 'BombStats|BombArm' -v     # TEST_EXIT=0
--- SKIP: TestAssautBombArmsGate           (ASSAUT_CACHE absent : le gate reel est en (b))
--- PASS: TestBuildBombArmings{DedupliqueLaPaire,EcarteLesMonteesSousLePlein,
          GardeDeuxArmementsDistincts,ConfrontationRetientToutOuRien,HorsFenetreCompte,
          SansLectureResteVide}
--- PASS: TestBombArmsJointure          (9 sous-cas : lacher a +126 ms, deux poseurs, le plus
          proche gagne, fermeture par MORT, periode OUVERTE, lacher hors fenetre, prise APRES
          l'armement, slot non ponte, exclusivite d'une periode)
--- PASS: TestBombArmsRecalageHorloge   (offset 40 000 ms applique ; contre-epreuve a 0 : plus
          aucune attribution, et l'armement reste publie sans acteur)
--- PASS: TestBombArmsSourcesNonLues    (2 sous-cas)
--- PASS: TestBombStats{Detonations,Grabs,CarrierSeconds,CarriersKilled,AucuneSourceLue}
PASS — ok levelup/go-api/internal/analysis/replay 0.377s
$ go vet ./internal/analysis/replay/          # VET_EXIT=0 (sortie vide)
$ gofmt -l internal/analysis/replay/          # GOFMT_EXIT=0 (aucun fichier liste)
$ golangci-lint run ./internal/analysis/replay/   # LINT_EXIT=0 — 0 issues.

(b) FILMS REELS — ASSAUT_CACHE, LockProcessDecode, sentinelle armee
$ go test ./internal/analysis/replay/ -run AssautBombArmsGate -v -timeout 60m  # GATE_EXIT=0
35b75a31 : recalage film -> match = 114 ms (premier paquet 8892226, deathOffset 8892112)
35b75a31 : armements dates 3 / attribues 1 / non attribues 2 (sans lacher 1, non ponte 1)
35b75a31 : ecart lacher - armement : min +247, mediane +247, max +247 ms (n=2)
9f57c612 : recalage film -> match = 88 ms
9f57c612 : armements dates 0 / attribues 0 / non attribues 0     (calque retenu a la source)
1c01e34f : recalage film -> match = 50 ms
1c01e34f : armements dates 4 / attribues 3 / non attribues 1 (sans lacher 1, non ponte 0)
1c01e34f : ecart lacher - armement : min +252, mediane +253, max +254 ms (n=3)
3d58eb37 : recalage film -> match = 33 ms
3d58eb37 : armements dates 3 / attribues 3 / non attribues 0
3d58eb37 : ecart lacher - armement : min +256, mediane +257, max +259 ms (n=3)
69b16f5d : recalage film -> match = 62 ms
69b16f5d : armements dates 3 / attribues 2 / non attribues 1 (sans lacher 1, non ponte 0)
69b16f5d : ecart lacher - armement : min +253, mediane +258, max +258 ms (n=2)
critere (b) tenu : 4 desaccords B2 sur les trois films instruits, inchanges
pic memoire observe : 0.20 Gio (plafond souple 2 Gio)
--- PASS: TestAssautBombArmsGate (344.00s)   —  aucune ligne `^--- FAIL:`
```

**LES DENOMINATEURS, ecrits pour qu'on ne lise pas ces chiffres comme des taux bruts :**

- **Critere (a), le « 100 % » du portage** : le denominateur est « les explosions ou le
  statborg NOMME le detonateur ET le porteur est ponte ». Sur `35b75a31`, 2 explosions y
  entrent (accord 2/2) et la troisieme (787051) en sort par `coverage.noBridge` — ce n'est PAS
  un desaccord. Sur `9f57c612` les 4 explosions sont jugees. Zero desaccord neuf.
- **Critere (b)** : 4 desaccords B2, exactement les quatre connus — 1 sur `1c01e34f` (400853),
  1 sur `3d58eb37` (203065), 2 sur `69b16f5d` (278617 et 310215).
- **Couverture d'attribution** : 13 armements dates sur les 5 films, **9 attribues (69 %)**,
  3 sans lacher dans la fenetre, 1 au slot non ponte. Le denominateur des 69 % est « les
  armements DATES par l'anneau », pas les explosions : One Bomb n'en date aucun (calque retenu
  a la source), et ses 4 explosions sont donc hors de ce ratio.
- **Le recalage n'est pas un detail cosmetique** : mesure directe sur les 5 films, 33 a 114 ms
  — meme ordre de grandeur que les 16-81 ms des quatre films temoins d'`origin.go`. Le terme
  est bien plus petit que la fenetre, mais il est APPLIQUE et teste
  (`TestBombArmsRecalageHorloge` : la contre-epreuve a offset 0 tombe a 0 attribution).

### E2-bis — Le repli « porteur actif a l'instant arme » (ARBITRE PAR L'UTILISATEUR le 2026-09-04)

E2 attribue 9 armements sur 13. Sur 3 des 4 restants, le porteur **traverse la pose** : sa
periode de portage couvre l'instant arme et ne se ferme jamais par un lacher (sur `35b75a31`
@299 176 elle se ferme 4 245 ms APRES l'explosion ; meme figure sur `1c01e34f` @395 764 et
`69b16f5d` @273 746, qui tombent exactement sur des explosions a desaccord B2).

**Decision de l'utilisateur : PRENDRE le repli.** La regle existe deja et est validee ailleurs
(`b2PorteurA`), et le premier cas est corrobore par le statborg. Couverture attendue
69 % -> ~85 %.

- [x] Ajouter le repli APRES la regle du lacher, jamais a sa place : le lacher reste la source
      primaire, le porteur actif est un SECOND recours, et l'evenement dit LEQUEL l'a nomme.
      **FAIT, ET LA PRIORITE EST STRUCTURELLE, PAS ORDINALE.** `bombArmsByXUID` fait DEUX
      PASSES : la passe 1 sert la regle du lacher sur TOUS les armements, la passe 2 ne sert le
      repli que sur les restants. Un parcours chronologique unique melant les deux regles
      laisserait le repli d'un armement ANTERIEUR consommer une periode qu'un lacher ulterieur
      reclamait — le second recours volerait la source primaire
      (`TestBombArmsLacherPrimeSurRepli/le_repli_d_un_armement_ANTERIEUR_ne_vole_pas_la_periode`).
      Le repli ne s'applique QU'AU cas « aucune periode candidate » : un armement dont la passe
      1 a retenu une periode au slot NON PONTE n'est PAS re-nomme (le poseur est identifie, son
      nom manque — le renommer designerait quelqu'un d'autre). L'evenement porte
      `BombEvent.ActorSource` (`carry_drop` / `carry_active`), et la couverture ventile
      `ArmingsByDrop` / `ArmingsByActiveCarry` : deux forces de preuve, jamais un chiffre
      global. La REGLE EXACTE : a defaut de lacher, l'armeur est le porteur dont une periode
      FERMEE (A) couvre l'instant arme `DebutMS <= t_arm <= FinMS`, (B) est la SEULE dans ce
      cas, (C) n'a pas deja servi. Les periodes OUVERTES sont hors candidature — leur `FinMS`
      est la sentinelle `HeldObjectOpenEndMS`, pas une mesure, et l'admettre ferait degenerer le
      repli en « le dernier qui a ramasse la bombe ». Les periodes fermees par la MORT sont
      candidates : leur fin est datee par le fil des morts.
- [x] Le controle de coherence reste : la somme des `bomb_arms` ne depasse jamais les armements
      dates.
      **FAIT, RENFORCE.** Deux invariants au lieu d'un, verifies par les tests purs
      (`assertBombArmCoverage`) ET par le gate reel (`baVerifierCoherence`) :
      `ArmingsAttributed == ArmingsByDrop + ArmingsByActiveCarry` (aucune regle non comptee) et
      `ArmingsAttributed + ArmingsNoCarrier + ArmingsNoBridge + ArmingsAmbiguous == Armings`
      (aucun armement perdu ni compte deux fois). La somme des `bomb_arms` publies vaut
      `ArmingsAttributed`, donc <= `Armings` par construction : une periode consommee ne peut
      plus servir, dans les DEUX passes.
- [x] Un armement dont deux porteurs seraient candidats reste SANS acteur — le repli ne doit
      jamais trancher entre deux.
      **FAIT** : `bombPorteurActifOf` rend `(-1, true)` des la DEUXIEME periode couvrante, et
      l'armement sort en `ArmingsAmbiguous` — publie, sans acteur, sans `ActorSource`. Un
      porteur ANONYME (slot non ponte) compte comme candidat : sa presence suffit a rendre
      l'instant ambigu meme si l'autre candidat est parfaitement nomme
      (`TestBombArmsRepliPorteurActif`, sous-cas « un candidat NOMME et un candidat ANONYME »).
      Le lacher, lui, garde son departage (condition 4, le plus proche gagne) : il designe un
      GESTE, la presence ne designe rien.
- [x] Gate : la couverture par film est republiee ; aucun desaccord neuf sur les deux films du
      gate de portage ; les 4 desaccords B2 restent ceux arbitres par B3.
      **FAIT** — sortie reelle collee ci-dessous, codes de sortie verifies.

**Gate E2-bis** : le test de gate republie, par film, `armements dates / attribues par lacher /
attribues par repli / non attribues`, et le run est vert.

**Gate E2-bis — PASSE le 2026-09-04** (codes de sortie verifies, motifs d'echec ancres sur
`^--- FAIL:`) :

```
(a) TESTS PURS — sans film, sans base
$ go test ./internal/analysis/replay/ -run 'BombStats|BombArm' -v -count=1   # TEST_EXIT=0
--- SKIP: TestAssautBombArmsGate           (ASSAUT_CACHE absent : le gate reel est en (b))
--- PASS: TestBombArmsJointure             (8 sous-cas — la regle PRIMAIRE seule)
--- PASS: TestBombArmsRepliPorteurActif    (7 sous-cas : le porteur TRAVERSE la pose ; lacher
          hors fenetre mais periode couvrante ; DEUX porteurs -> aucun acteur ; un candidat
          NOMME + un candidat ANONYME -> toujours ambigu ; unique candidat non ponte ; periode
          OUVERTE -> repli muet ; exclusivite)
--- PASS: TestBombArmsLacherPrimeSurRepli  (2 sous-cas : deux joueurs DIFFERENTS designes, le
          lacher gagne ; le repli d'un armement ANTERIEUR ne vole pas la periode)
--- PASS: TestBombArmsRecalageHorloge      (offset 40 000 ms sur les DEUX regles ; contre-
          epreuve a 0 : plus aucune attribution, armement publie sans acteur)
--- PASS: TestBombArmsSourcesNonLues       (2 sous-cas)
--- PASS: TestBombStats{Detonations,Grabs,CarrierSeconds,CarriersKilled,AucuneSourceLue}
PASS — ok levelup/go-api/internal/analysis/replay 0.355s   ;  0 ligne `^--- FAIL:`
$ go vet ./internal/analysis/replay/            # VET_EXIT=0 (sortie vide)
$ gofmt -l internal/analysis/replay/            # GOFMT_EXIT=0 (aucun fichier liste)
$ golangci-lint run ./internal/analysis/replay/ # LINT_EXIT=0 — 0 issues.
$ go test ./internal/analysis/replay/ -count=1  # PKG_EXIT=0 — paquet entier, 17,9 s (filet)

(b) FILMS REELS — ASSAUT_CACHE, LockProcessDecode, sentinelle memoire armee
$ go test ./internal/analysis/replay/ -run AssautBombArmsGate -v -count=1 -timeout 60m
                                                                          # GATE_EXIT=0
35b75a31 : recalage film -> match = 114 ms (premier paquet 8892226, deathOffset 8892112)
35b75a31 : armements dates 3 / par lacher 1 / par repli 1 / non attribues 1
           (sans porteur 0, slot non ponte 1, ambigus 0)
  bomb_armed 299176 -> 2535460750735339 (carry_active)   [290194, 308258], fin par mort
  bomb_armed 536215 -> 2533274824432629 (carry_drop)
  bomb_armed 782064 -> SANS ACTEUR                       (slot 665 non ponte)
9f57c612 : recalage 88 ms — armements dates 0 (calque retenu a la source, cf. E2-ter)
1c01e34f : recalage 50 ms — armements dates 4 / par lacher 3 / par repli 1 / non attribues 0
  bomb_armed 395764 -> 2535406537212173 (carry_active)
3d58eb37 : recalage 33 ms — armements dates 3 / par lacher 3 / par repli 0 / non attribues 0
69b16f5d : recalage 62 ms — armements dates 3 / par lacher 2 / par repli 1 / non attribues 0
  bomb_armed 273746 -> 2812739231567214 (carry_active)
critere (b) tenu : 4 desaccords B2 sur les trois films instruits, inchanges
pic memoire observe : 0.20 Gio (plafond souple 2 Gio)
--- PASS: TestAssautBombArmsGate (286.80s)   —  aucune ligne `^--- FAIL:`
```

**COUVERTURE D'ATTRIBUTION, AVANT / APRES** (denominateur : les armements DATES par l'anneau,
pas les explosions — One Bomb n'en date aucun, ses 4 explosions sont hors de ce ratio) :

| Film | dates | E2 attribues | E2-bis par lacher | par repli | E2-bis total | non attribues |
|---|---|---|---|---|---|---|
| `35b75a31` Neutral | 3 | 1 | 1 | 1 | **2** | 1 (slot non ponte) |
| `9f57c612` One Bomb | 0 | 0 | 0 | 0 | 0 | 0 |
| `1c01e34f` Husky | 4 | 3 | 3 | 1 | **4** | 0 |
| `3d58eb37` | 3 | 3 | 3 | 0 | 3 | 0 |
| `69b16f5d` | 3 | 2 | 2 | 1 | **3** | 0 |
| **TOTAL** | **13** | **9 (69 %)** | **9** | **3** | **12 (92,3 %)** | **1** |

**CE QUE CES CHIFFRES DISENT, ET CE QU'ILS NE DISENT PAS :**

- **Le repli n'a rien retire a la regle primaire** : les 9 attributions par lacher sont
  EXACTEMENT les 9 d'E2, film par film. Le repli n'ajoute que sur des armements que la source
  primaire laissait anonymes — c'est la structure en deux passes qui le garantit.
- **Les 3 armements du repli sont les 3 temoins releves par E2** : `35b75a31` @299 176,
  `1c01e34f` @395 764, `69b16f5d` @273 746. Aucun quatrieme cas n'est apparu, et aucun
  armement n'est tombe en `ArmingsAmbiguous` sur le corpus : sur ces cinq films, jamais deux
  porteurs ne couvrent le meme instant arme.
- **LE PREMIER EST CORROBORE PAR UNE SOURCE INDEPENDANTE.** Le gate imprime, pour l'explosion
  de 304 013 ms : `poseur = detonateur 2535460750735339` — le STATBORG nomme le meme joueur
  que le repli a nomme sur l'armement de 299 176. Les deux autres tombent sur des explosions a
  desaccord B2 connu : le repli y nomme le PORTEUR, le statborg y nomme un autre detonateur,
  et B3 n'a pas tranche. Le desaccord n'est ni cree ni resolu par ce lot — il reste le meme,
  et c'est le critere (b) qui le verifie.
- **Le seul armement non attribue est au slot NON PONTE** (`35b75a31` @782 064) : le porteur
  est identifie, son nom manque. Ce n'est pas un manque de la jointure mais du pont
  slot->xuid ; le repli ne peut rien y faire, et inventer le nom serait la seule vraie faute.
- **92,3 % au lieu des ~85 % attendus** : l'ecart vient du choix d'admettre au repli les
  periodes fermees par la MORT du porteur (le temoin `35b75a31` @299 176 en est une). Leur fin
  est DATEE par le fil des morts, donc « il la tenait a cet instant » y est une mesure, pas
  une extrapolation — a la difference d'une periode restee ouverte, qui reste ecartee.
- **Criteres (a) et (b) tenus** : zero desaccord neuf sur `35b75a31` et `9f57c612` (les deux
  explosions du denominateur de `35b75a31` sont en accord, la troisieme sort par
  `coverage.noBridge`, les 4 de `9f57c612` sont jugees), et exactement 4 desaccords B2 sur les
  trois films instruits — 1c01e34f (400853), 3d58eb37 (203065), 69b16f5d (278617, 310215).

### E2-ter — Lever la garde One Bomb (ARBITRE PAR L'UTILISATEUR le 2026-09-04)

**Constat sur pieces** : `bomb_armings.go` porte une garde de mode DOUBLE, et sa garde 1
exclut nommement One Bomb (`replaybuild.isArmableBombVariant`) parce que la lecture SIMPLE n'y
tient pas (CV 0,725, 87/1000 tirages nuls font aussi bien). Consequence mesuree en E2 :
`9f57c612` rend **0 armement date**. Toute une variante d'Assaut reste donc sans `bomb_arms`.

Or `wt/onebomb` a RESOLU la meche One Bomb le 2026-09-01 — **16,2 s et PAUSABLE**, 9/9
explosions portees, CV 0,017, 0/1000 tirages nuls — et cette lecture n'a **jamais ete portee en
production** : c'est la reconciliation notee « A RECONCILIER » a la fusion et jamais faite.

> **ETAT AU 2026-09-04, apres execution** : les deux paragraphes ci-dessus decrivent le
> CONSTAT D'OUVERTURE, plus l'etat du code. La garde 1 n'existe plus, la lecture pausable est
> en production, et la reconciliation est faite. Details, chiffres et gates ci-dessous.

- [x] Porter la lecture pausable en production : armement = segment contigu finissant a son
      sommet (tol. 4 quanta) ; pause = descente < 60 quanta/s ; delai = (explosion − fin
      d'armement) − somme des pauses du meme slot.
      **FAIT, ET LA COPIE A DISPARU AU LIEU DE SE DUPLIQUER.** Nouveau fichier de production
      `internal/analysis/filmdec/navpoint_radial_segments.go` (145 L) : `NavpointSegment` +
      `NavpointSegments()` (suite contigue SANS exigence de monotonie — c'est ce decoupage-la
      qui fait sortir de lui-meme le cycle de RECHARGE 130 -> 254 -> 127, qui finit a son
      MINIMUM), predicat `EndsAtSummit()` (armement) et `IsDisarmHold()` (tenue de
      desarmement), seuils `NavpointSummitToleranceQ = 4` et `NavpointPauseMaxSlopeQS = 60`.
      L'INSTRUMENT APPELLE MAINTENANT LA PRODUCTION : `obSegmenter`, `mpEstArmement` et
      `mpEstPause` (navpoint_ti12_onebomb_test.go / navpoint_ti12_meche_test.go) delegent, et
      les deux constantes ont QUITTE le fichier de test — le gate juge le code livre, pas une
      copie. Cote interpretation, `buildBombArmings` enchaine segments -> armements pleins et
      pauses -> deduplication de paire -> confrontation. **LA MECHE N'EST PLUS UNE CONSTANTE
      UNIQUE** : elle est MESUREE par film (mediane des delais corriges) et publiee avec chaque
      evenement dans `BombArming.FuseMS` — le champ existait deja et disait qu'il servirait a
      cela. `BombFuseMS` (4 930) ne subsiste que comme valeur de REFERENCE, DEDUITE, pour le
      seul film qui ne porte AUCUNE explosion a mesurer (`coverage.detonations == 0` dit ce
      cas). Mesures du gate : 4 987 ms sur `35b75a31`, 5 089 ms sur `1c01e34f` (la meche Husky
      que `b3MecheMS` estimait a 5 100), 16 183 ms sur `9f57c612` — trois valeurs, UNE regle,
      zero branchement sur le nom de la variante.
- [x] La garde 2 (confrontation locale tout-ou-rien) reste : elle protege contre un film qui
      contredit la lecture. C'est la garde 1 (le NOM de la variante) qui tombe.
      **FAIT, ET LA GARDE 2 EST RENFORCEE PLUTOT QUE RELACHEE.** Garde 1 supprimee :
      `replaybuild.isArmableBombVariant` n'existe plus, `bombInput` ne prend qu'un booleen de
      FAMILLE (`isBombVariant`), et un RATCHET interdit son retour
      (`replaybuild.TestAucuneGardeParNomDeVariante` : aucun fichier de production du paquet ne
      doit contenir le litteral « one bomb »). Garde 2 : elle etait « chaque explosion a un
      armement a 4,93 s +/- 0,6 s » — une fenetre qui SUPPOSAIT la meche, c'est-a-dire la
      variante reconnue. Elle est desormais en DEUX branches, toujours TOUT-OU-RIEN par film :
      (1) COUVERTURE — chaque explosion a un armement dans la fenetre de sens du protocole
      (`BombFuseSenseWindowMS` = 120 s), delai corrige des pauses ; (2) DISPERSION — les delais
      corriges du film doivent s'accorder (`bombFuseMaxCV` = 0,20, le seuil du protocole ; la
      marge est d'un ordre de grandeur : 0,010 a 0,019 mesures, contre 0,725 pour la lecture
      SIMPLE que ce seuil a refutee). Les deux branches MORDENT sur le corpus, et c'est la
      preuve qu'elles ne sont pas decoratives (cf. le tableau One Bomb ci-dessous).
- [x] Ne PAS toucher aux temoins Neutral/Husky : ils doivent rester a 13/13 et 4/4.
      **TENU AU CHIFFRE PRES, ET VERIFIE DEUX FOIS.** (a) Cote `filmdec`,
      `TestNavpointTi12MecheTemoin` rend 13/13 sur les 13 explosions Neutral Bomb et 4/4 sur
      Husky Raid — et l'exigence Husky, qui n'etait qu'IMPRIMEE, est maintenant JUGEE (le test
      ne faisait echouer que Neutral). (b) Cote `replay`, le gate de portage republie des
      armements identiques instant par instant : `35b75a31` 3 dates (299 176 / 536 215 /
      782 064), `1c01e34f` 4 (145 306 / 268 731 / 330 599 / 395 764), `3d58eb37` 3, `69b16f5d`
      3, avec EXACTEMENT les memes acteurs et les memes regles (`carry_drop` / `carry_active`)
      qu'en E2-bis. Aucun armement n'a bouge.
- [x] Gate : `9f57c612` publie ses armements ; les temoins Neutral et Husky sont inchanges au
      chiffre pres.
      **FAIT** — sorties reelles collees ci-dessous, codes de sortie verifies.

**Dependance** : E2-bis et E2-ter modifient tous deux l'attribution ; les executer dans cet
ordre, et rejouer le gate d'E2 apres chacun. **RESPECTE** : E2-bis etait clos, et les deux
gates d'E2/E2-bis ont ete rejoues integralement apres E2-ter.

**Gate E2-ter — PASSE le 2026-09-04** (codes de sortie verifies, motifs d'echec ANCRES sur
`^--- FAIL:`) :

```
(a) TESTS PURS — sans film, sans base
$ go test ./internal/analysis/replay/ -run 'BombStats|BombArm' -v -count=1   # TEST_EXIT=0
--- SKIP: TestAssautBombArmsGate            (ASSAUT_CACHE absent : le gate reel est en (b))
--- PASS: TestBuildBombArmings{DedupliqueLaPaire,EcarteLesSegmentsSousLePlein,
          EcarteLeCycleDeRecharge,GardeDeuxArmementsDistincts,ConfrontationRetientToutOuRien,
          MecheMesureeEtPausable,MechesQuiSeContredisent,HorsFenetreCompte,SansLectureResteVide}
--- PASS: TestBombArms{Jointure,RepliPorteurActif,LacherPrimeSurRepli,RecalageHorloge,
          SourcesNonLues}
--- PASS: TestBombStats{Detonations,Grabs,CarrierSeconds,CarriersKilled,AucuneSourceLue}
PASS — ok levelup/go-api/internal/analysis/replay 0.429s ; 0 ligne `^--- FAIL:`
$ go test ./internal/analysis/filmdec/ -run 'Navpoint' -v -count=1           # TEST_EXIT=0
$ go vet ./internal/analysis/replay/ ./internal/analysis/filmdec/            # VET_EXIT=0
$ gofmt -l internal/analysis/replay/ internal/analysis/filmdec/              # GOFMT_EXIT=0
$ go test ./internal/analysis/replay/ ./internal/analysis/filmdec/ ./internal/replaybuild/
                                                          # PKG_EXIT=0 (filet, 3 paquets)
```

TROIS tests purs sont NEUFS et portent la lecture : `EcarteLeCycleDeRecharge` (le cycle
complet du marqueur finit a son minimum — c'est ce que le decoupage en montees ne voyait pas),
`MecheMesureeEtPausable` (une tenue de 4 000 ms retranchee d'un delai brut de 20 200 ms rend
une meche de 16 200 ms, avec CONTRE-EPREUVE sans la tenue) et `MechesQuiSeContredisent` (deux
explosions COUVERTES mais a 5 s et 20 s : la garde 2 retient par la DISPERSION).

```
(b1) FILMS REELS — LE TEMOIN DE LA LECTURE (filmdec), ASSAUT_CACHE, LockProcessDecode,
     sentinelle memoire armee
$ go test ./internal/analysis/filmdec/ -run 'NavpointTi12Meche' -v -count=1  # GATE_MECHE_EXIT=0
########## TEMOIN NEUTRAL BOMB — 13 explosions : couverture 13/13, delai median 4.94 s, CV 0.016
           VERDICT : RESOLU sous la regle (couverture pleine, CV <= 0,20, nulle < 1 %)
TEMOIN : EXIGENCE TENUE (13/13, CV 0.016 <= 0.02)
########## TEMOIN HUSKY RAID — 4 explosions : couverture 4/4, delai median 5.09 s, CV 0.016
           VERDICT : RESOLU sous la regle
TEMOIN HUSKY : EXIGENCE TENUE (4/4, CV 0.016 <= 0.02)
########## ONE BOMB — 11 explosions : couverture 10/11, delai median 16.18 s, CV 0.228
           VERDICT : NON RESOLU sous la regle
########## ONE BOMB — 9 explosions PORTEES (partition a5SansPorteur) : couverture 9/9,
           delai median 16.18 s, CV 0.017 — VERDICT : RESOLU (nulle 0/1000)
--- PASS: TestNavpointTi12Meche{SansPorteurFige,OneBomb,Temoin}   — 0 ligne `^--- FAIL:`
```

**LE PORTAGE EST FIDELE AU CHIFFRE PRES** : 13/13 CV 0,016, 4/4 CV 0,016, 9/9 median 16,18 s
CV 0,017 — les MEMES nombres que la mesure du 2026-09-01, obtenus par le code de PRODUCTION
(l'instrument l'appelle desormais). Et la ligne « 11 explosions : 10/11, CV 0,228, NON RESOLU »
est le fait qui commande tout le reste : les deux explosions d'`a5SansPorteur` ne sont pas
expliquees par l'anneau non plus.

```
(b2) FILMS REELS — LE GATE DE L'ARMEMENT (replay)
$ go test ./internal/analysis/replay/ -run AssautArmementGate -v -count=1  # GATE_ARM_EXIT=0
35b75a31 : 1900 lectures, 177 segments, 3 armements (3 fondus de paire), 3 publies,
           meche MESUREE 4987 ms (CV 0.019)
  explosions : delais CORRIGES 4837 / 5055 / 4987 ms   (critere (b) : tous dans 4930 +/- 600)
1c01e34f : 572 lectures, 76 segments, 4 armements (4 fondus), 4 publies,
           meche MESUREE 5089 ms (CV 0.016)
  explosions : delais CORRIGES 5240 / 5056 / 5038 / 5089 ms
9f57c612 (One Bomb) : 111 segments, 5 armements, 5 PUBLIES, 4/4 explosions couvertes,
           meche MESUREE 16183 ms (CV 0.010)
  explosions : delais CORRIGES 16183 / 16181 / 16165 / 16516 ms
  armements publies (horloge du film) : 65 137 · 279 103 · 335 193 · 388 080 · 445 839 ms
c75f33b8 (One Bomb) : 94 segments, 4 armements, 0 publie, 2/3 couvertes — RETENU
  l'explosion 395 724 (a5SansPorteur) n'a AUCUN armement dans la fenetre de sens ;
  les deux autres rendent 16 318 et 15 900 ms  -> garde 2 par la COUVERTURE
df8fcbef (One Bomb) : 126 segments, 7 armements, 0 publie, 4/4 couvertes — RETENU
  l'explosion 778 033 (a5SansPorteur) rend 27 845 ms contre 15 965 / 15 929 / 16 785 ;
  CV 0,331 > 0,20  -> garde 2 par la DISPERSION
--- PASS: TestAssautArmementGate (46.13s)   — aucune ligne `^--- FAIL:`

(b3) FILMS REELS — LE GATE D'ATTRIBUTION (E2/E2-bis rejoue apres E2-ter)
$ go test ./internal/analysis/replay/ -run AssautBombArmsGate -v -count=1
35b75a31 : armements dates 3 / par lacher 1 / par repli 1 / non attribues 1 (non ponte 1)
9f57c612 : armements dates 5 / par lacher 2 / par repli 1 / non attribues 2
           (sans porteur 1, slot non ponte 1, ambigus 0)
  bomb_armed  279103 -> 2535470823750470 (carry_drop)
  bomb_armed  335193 -> 2535419279251362 (carry_drop)
  bomb_armed  388080 -> 2535470823750470 (carry_active)
  bomb_armed   65137 -> SANS ACTEUR (slot 518 non ponte)
  bomb_armed  445839 -> SANS ACTEUR (aucune periode fermee ne le couvre)
  ecart lacher - armement : min +250, mediane +250, max +251 ms (n=3, fenetre +/-2500 ms)
1c01e34f : armements dates 4 / par lacher 3 / par repli 1 / non attribues 0
3d58eb37 : armements dates 3 / par lacher 3 / par repli 0 / non attribues 0
69b16f5d : armements dates 3 / par lacher 2 / par repli 1 / non attribues 0
critere (b) tenu : 4 desaccords B2 sur les trois films instruits, inchanges
pic memoire observe : 0.18 Gio (plafond souple 2 Gio)
--- PASS: TestAssautBombArmsGate (286.08s)   — aucune ligne `^--- FAIL:`
```

**COUVERTURE D'ATTRIBUTION, AVANT (E2-bis) / APRES (E2-ter)** — denominateur : les armements
DATES par l'anneau.

| Film | dates AVANT | attrib. AVANT | dates APRES | par lacher | par repli | attrib. APRES | non attribues |
|---|---|---|---|---|---|---|---|
| `35b75a31` Neutral | 3 | 2 | 3 | 1 | 1 | 2 | 1 (non ponte) |
| `9f57c612` One Bomb | **0** | **0** | **5** | **2** | **1** | **3** | **2** (sans porteur 1, non ponte 1) |
| `1c01e34f` Husky | 4 | 4 | 4 | 3 | 1 | 4 | 0 |
| `3d58eb37` | 3 | 3 | 3 | 3 | 0 | 3 | 0 |
| `69b16f5d` | 3 | 3 | 3 | 2 | 1 | 3 | 0 |
| **TOTAL** | **13** | **12 (92,3 %)** | **18** | **11** | **4** | **15 (83,3 %)** | **3** |

**LIRE CE TABLEAU DANS LE BON SENS.** Le TAUX baisse (92,3 -> 83,3 %) pendant que le COMPTE
monte (12 -> 15) : le denominateur a grandi de 5 armements qu'aucune mesure ne datait avant,
et 2 d'entre eux restent sans acteur. Un taux qui baisse en gagnant des faits n'est pas une
regression — c'est ce que dit un denominateur honnete. **Aucune ligne des quatre films temoins
n'a bouge d'une unite**, ni en compte, ni en instant, ni en acteur, ni en regle.

**CE QUE 9f57c612 PUBLIE, ET A QUELS INSTANTS** (horloge du film) : 5 armements a
**65 137 · 279 103 · 335 193 · 388 080 · 445 839 ms**, dont 3 nommes — 279 103 et 388 080 a
`2535470823750470`, 335 193 a `2535419279251362`. Les deux anonymes le sont pour des raisons
DIFFERENTES et nommees : 65 137 a bien un lacher a +250 ms, mais au slot 518 que le pont ne
nomme pas (`ArmingsNoBridge`) ; 445 839 n'est couvert par AUCUNE periode de portage fermee
(`ArmingsNoCarrier`). Dans les deux cas l'armement est PUBLIE sans acteur, jamais devine.

**CORROBORATION INDEPENDANTE** : le gate de portage imprime, pour l'explosion 298 489 de
`9f57c612`, `poseur = detonateur 2535470823750470` — le STATBORG nomme le meme joueur que la
regle du lacher a nomme sur l'armement de 279 103. Deux canaux, deux chaines, un seul nom.

**LA FENETRE DE RATTRAPAGE DE L'ATTRIBUTION N'A PAS BOUGE, ET LA MESURE DIT POURQUOI.** La
question etait ouverte au brief : une meche trois fois plus longue pouvait desaccorder la
jointure. Elle ne le fait pas, parce que **la fenetre ne joint pas l'explosion, elle joint le
LACHER a la FIN DE L'ARMEMENT** — un geste, pas un compte a rebours. Ecarts mesures sur
`9f57c612` : **+250, +250, +251 ms** (n=3), contre +247 a +259 ms sur les quatre films a meche
courte. La demi-fenetre de 2 500 ms garde donc le meme facteur 10 de marge. **AUCUN ajustement
n'est fait, et c'est ecrit ici plutot que suppose.**

**SCHEMA 38 -> 39, ET POURQUOI CE N'EST PAS UN FIX OPPORTUNISTE.** Aucun champ n'est ajoute ni
retire — la couverture publiee garde exactement ses clefs, et la meche mesuree passe par
`BombArming.FuseMS`, un champ qui existait deja et dont le commentaire annoncait ce role. Mais
le CONTENU d'un calque change pour toute une variante, et la doctrine du depot est explicite et
repetee (`document.go`, montees v14/v22/v25/v37) : la reprise du backfill se fait par
`SchemaVersion`, donc un artefact 38 d'un match One Bomb doit se lire « a re-cuire », pas « a
jour ». Sans la montee, aucun rejeu One Bomb deja cuit ne porterait jamais son compte a
rebours. La montee est donc DICTEE par le changement, pas ajoutee a cote. Elle ne cuit rien par
elle-meme : le backfill du parc reste la tache de RELEASE d'E6. Le ratchet
`TestStructureIsOptionalInDocument` porte la raison ecrite ; le golden d'assemblage est
regenere (`-update`, une ligne : `schema 38` -> `schema 39`). **Contrat OpenAPI : INCHANGE** —
`contracttest` compare les champs Go au schema, et aucun champ n'a bouge, donc ni `openapi-gen`
ni `generate-types` n'entrent dans ce lot.

### E3 — Persistance

- [x] Migration `shared_create_bomb_stats` : table `match_bomb_stats` append-only (id PK
      sequence, `match_id`, `xuid`, les 5 colonnes, `written_at`), index `match_id`, vue
      `match_bomb_stats_latest` (`QUALIFY ROW_NUMBER() ... PARTITION BY match_id, xuid ORDER BY
      written_at DESC, id DESC` — reprendre le patron de `objective_stats_view.go`).
      **FAIT** : `internal/migration/steps_shared_bomb_stats.go` (step `shared_create_bomb_stats`,
      `TargetShared`). Les CINQ colonnes de mesure sont NULLABLE — « absent n'est pas zero » est
      une propriete du SCHEMA, pas seulement une convention de code, et
      `TestColonnesDeMesureNullables` la verrouille. Entree ajoutee a `canonicalOrder` a la
      position dictee par l'ordre d'init (entre `shared_append_only_weapon_kills_v1` et
      `shared_h5_weapon_kill_kind_v1`) — `TestSortByCanonicalIsNoOpOnCurrentRegistry` vert.
      L'en-tete ecrit d'ou vient chaque colonne, et que tout elargissement futur passe par un
      step au NOM NEUF qui RECREE la vue (`SELECT *` fige a la creation).
- [x] Persister INSERT-only branche sur `persist.BatchBuilder.Submit()`. Aucun UPSERT, aucun
      `ON CONFLICT DO UPDATE`.
      **FAIT** : `internal/persist/bomb_stats_persister.go` — `BombStatsPersister` sur le patron
      de `WeaponShotsPersister` (le film arrive un cycle apres le sync primaire, donc
      `SharedPersister` ne peut pas le porter : il est no-op quand `Shared.Match == nil`).
      Chemin batch : `SharedBatch.BombStats` + `BatchBuilder.SetBombStats()` + appel cable dans
      `CombinedPersister` (meme fenetre de lease, transaction distincte) — un setter dont la
      charge serait jetee serait une perte MUETTE. Chemin direct : `PersistPass`. Uniquement des
      INSERT ; toutes les lignes d'une passe partagent le meme `written_at`.
- [x] Les evenements dates vont dans `match_objective_events` (`objective_type='bomb'`,
      `event_type` dans `bomb_armed` / `bomb_detonated`, `team_id`, `source`, `confidence`),
      acteurs dans `match_objective_event_players` (`role='scorer'`) — par un chemin
      INSERT-only, PAS par `ObjectiveEventsRepo.WriteMatch`.
      **FAIT, dans la MEME transaction que les statistiques.** `ObjectiveEventsRepo.WriteMatch`
      n'est ni appele ni modifie. Deux points TRANCHES faute d'alternative sans DELETE, et
      ecrits dans l'en-tete du persister : (a) le `seq` est alloue APRES le maximum deja present
      sur le match — la PK `(match_id, seq)` est partagee avec les autres producteurs, repartir
      de 0 entrerait en collision ; (b) les faits ne sont ecrits QUE si le match n'en porte pas
      deja de la famille `bomb` — cette table n'a ni vue `_latest` ni `decode_pass`, donc un
      INSERT-only repete ne pourrait que violer la PK ou empiler deux generations que tout
      lecteur compterait double. Consequence assumee et journalisee
      (`slog.InfoContext`) : apres un changement de decodeur les statistiques se rafraichissent,
      les faits dates NON. Un armement sans acteur resolu s'ecrit SANS ligne dans
      `match_objective_event_players` (`TestFaitDateSansActeurResteUnFait`).
- [x] Lecteurs : vue `_latest` UNIQUEMENT (regle ART n2).
      **FAIT** : aucun lecteur n'est ecrit en E3 (c'est E5). Ce qui est POSE ici, c'est le
      garde-rail qui l'imposera : `match_bomb_stats` entre dans `appendOnlyStateTables`
      (`sync/append_only_state_guard_test.go`), qui interdit statement-level tout DELETE /
      ON CONFLICT / INSERT OR REPLACE|IGNORE sur elle dans le hot path.
- [x] Ratchet : verifier que `internal/sync/no_art_patterns_test.go` passe sans nouvelle entree
      d'allowlist. Si une entree est necessaire, elle porte sa justification datee.
      **FAIT, ET DANS LE SENS DU DURCISSEMENT** : `match_bomb_stats` est AJOUTEE a
      `tablesProtegees`. `allowlistArtPatterns` et `allowlistRawDelete` restent VIDES — aucune
      entree ajoutee, aucune n'etait necessaire (le persister n'emet que des INSERT).

**Gate E3** : `cd apps/go-api && go test -tags=integration -p 1 ./internal/persist/...
./internal/migration/... ./internal/sync/...` — code de sortie 0 verifie, pas seulement la
sortie filtree. `-p 1` non negociable (driver DuckDB mono-process).

**Gate E3 — PASSE le 2026-09-04** (code de sortie verifie ; motif d'echec ANCRE sur
`^--- FAIL:`, jamais un `FAIL` nu qui attraperait les logs applicatifs) :

```
$ go test -tags=integration -p 1 -count=1 ./internal/persist/... ./internal/migration/... ./internal/sync/...
ok      levelup/go-api/internal/persist              52.884s
ok      levelup/go-api/internal/migration             5.396s
ok      levelup/go-api/internal/sync                123.147s
ok      levelup/go-api/internal/sync/haloclient       5.212s
ok      levelup/go-api/internal/sync/halotest         0.360s
ok      levelup/go-api/internal/sync/invariants       0.348s
ok      levelup/go-api/internal/sync/killcollector    9.521s
?       levelup/go-api/internal/sync/matchflags       [no test files]
ok      levelup/go-api/internal/sync/objective        0.064s
ok      levelup/go-api/internal/sync/replayartifacts  5.972s
?       levelup/go-api/internal/sync/schemadrift      [no test files]
ok      levelup/go-api/internal/sync/skill            1.231s
ok      levelup/go-api/internal/sync/snapshot         0.547s
?       levelup/go-api/internal/sync/testutil         [no test files]
ok      levelup/go-api/internal/sync/v2              13.389s
GATE_EXIT=0        # nb de lignes `^--- FAIL:` = 0

$ go vet ./internal/persist/... ./internal/migration/... ./internal/sync/...   # VET_EXIT=0
$ go vet -tags=integration ./internal/persist/... ./internal/migration/...     # VET_INTEG_EXIT=0
$ gofmt -l internal/persist internal/migration internal/sync                   # GOFMT_EXIT=0, aucun fichier liste
$ golangci-lint run ./internal/persist/... ./internal/migration/... ./internal/sync/...
  37 issues — TOUTES preexistantes (dette baseline du perimetre). ZERO sur les fichiers de ce
  lot : le filtre sur `bomb`, `combined_persister.go`, `builder.go`, `batch.go`, `order.go`,
  `no_art_patterns_test.go` et `append_only_state_guard_test.go` ne retourne rien.
```

**DDL REEL POSE** (`internal/migration/steps_shared_bomb_stats.go`) :

```sql
CREATE SEQUENCE IF NOT EXISTS match_bomb_stats_id_seq START 1;
CREATE TABLE IF NOT EXISTS match_bomb_stats (
    id                           BIGINT    PRIMARY KEY DEFAULT nextval('match_bomb_stats_id_seq'),
    match_id                     VARCHAR   NOT NULL,
    xuid                         VARCHAR   NOT NULL,
    bomb_detonations             INTEGER,     -- NULL = non mesure, JAMAIS 0
    bomb_arms                    INTEGER,
    bomb_grabs                   INTEGER,
    time_as_bomb_carrier_seconds DOUBLE,
    bomb_carriers_killed         INTEGER,
    written_at                   TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
);
CREATE INDEX IF NOT EXISTS idx_match_bomb_stats_match ON match_bomb_stats(match_id);

CREATE OR REPLACE VIEW match_bomb_stats_latest AS
SELECT *
FROM match_bomb_stats
QUALIFY ROW_NUMBER() OVER (
    PARTITION BY match_id, xuid
    ORDER BY written_at DESC, id DESC
) = 1;
```

**CHEMIN D'ECRITURE** : `BatchBuilder.SetBombStats(*BombStatsBatch)` ->
`SharedBatch.BombStats` -> `persist.BombStatsPersister` (appele par `CombinedPersister` dans la
meme fenetre de lease que `SharedPersister`, transaction distincte) ; chemin direct
`BombStatsPersister.PersistPass` pour une completion tardive de film. UNE transaction ecrit
`match_bomb_stats` + `match_objective_events` + `match_objective_event_players`, en INSERT purs.

### E4 — Branchement au sync

> **EXECUTE LE 2026-09-05, etape G.2 du plan d'integration** (`wt/integ-assaut`). La forme a
> CHANGE par rapport a la lettre de ces items, et le constat qui l'a imposee est mesure : le
> DOCUMENT ne porte pas les entrees de `BuildBombStats` sous la forme voulue (portage en FRAMES
> et non en ms, periodes non pontees ecartees, pas de recalage film->match, pas de paire
> tueur/victime). Les recalculer chez le consommateur en aurait fait un SECOND decodeur du meme
> fait. Les stats se calculent donc A LA CUISSON (`replay.attachBombStats`, appele par
> `BuildFromPositions`, ou les quatre sources vivent en pleine fidelite) et voyagent dans
> l'artefact (`bombStats` + `bombEvents`, schema 39) ; le crochet post-sync ne fait plus que les
> TRANSPORTER vers `persist`. L'esprit de l'item — « aucun second decodage, aucun film relu » —
> est tenu a la lettre.

- [x] Dans `internal/sync/replayartifacts/`, extraire les stats bombe PENDANT le decodage deja
      fait pour l'artefact, et les soumettre au batch. Aucun second decodage, aucun film relu.
      **FAIT, sous la forme ci-dessus** : `attachBombStats` calcule au moment du decodage (aucun
      balayage de plus, aucune etape observee de plus — `BuildFromFilmSteps` reste a 35), et
      `replayartifacts/bombstats.go` lit l'artefact RANGE (jamais le blob candidat) puis ecrit
      par `persist.BombStatsPersister.PersistPass`. Patron EXACT de `usage.go` : projections
      avant le writer, burst court apres TOUTE cuisson.
- [x] Degradation gracieuse : film absent, mode non-Assaut, capability absente -> rien d'ecrit,
      aucune erreur remontee, un `slog.InfoContext` qui dit pourquoi.
      **FAIT, et les trois silences sont DISTINGUES** : mode hors Assaut (le document ne porte
      aucun `bombStats` — DEBUG, et surtout PAS un echec : c'est le cas majoritaire de chaque
      cycle), capability absente (DEBUG), artefact illisible (WARN + compteur d'echecs). Un
      quatrieme cas a ete ajoute apres coup : un film d'Assaut dont AUCUNE source n'a rien rendu
      est ecarte lui aussi, sans quoi le persister aurait WARNe « passe vide » a chaque cycle.
- [x] `slog.ErrorContext(ctx, ..., "err", err, "match_id", ...)` sur toute erreur non triviale.
      Jamais d'erreur avalee.
      **FAIT** : l'echec d'ECRITURE d'un match est un `slog.ErrorContext` avec `match_id` et
      `err` ; les indisponibilites (writer absent, capabilities illisibles) sont des WARN
      comptes. Aucun `_ = f()`, aucun `continue` muet.
- [x] Verifier que le plafond `maxPerCycle` (5) et le verrou de decodage sont respectes : ce
      lot n'augmente PAS le nombre de films decodes par cycle.
      **VERIFIE SUR PIECES** : ni `maxPerCycle` ni `filmproc` ne sont touches par le lot, et le
      crochet ne decode rien — il lit un fichier JSON deja ecrit. La liste des artefacts cuits
      du cycle (`artefactCuit`, ex-`rapportUsage`) est PARTAGEE avec le resume d'usage : une
      seule collecte, deux projections.

**Gate E4** : test du hook avec un film factice ; et `go test ./internal/sync/... -run
ReplayArtifacts` vert.

**Gate E4 — PASSE le 2026-09-05** :

```
$ go test -tags=integration -p 1 ./internal/sync/replayartifacts/ -run StatsBombe -v   # EXIT=0
--- PASS: TestPersisterStatsBombe_UnMatchDAssautEcritSesLignesEtSesFaits (0.96s)
--- PASS: TestPersisterStatsBombe_ModeHorsAssautNEcritRien              (0.90s)
--- PASS: TestPersisterStatsBombe_TitreSansCapability                   (0.97s)
--- PASS: TestPersisterStatsBombe_ArtefactIllisibleNArretePasLeLot      (0.94s)
$ go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/migration/...
                                                          # EXIT_G2_INTEG=0, 12 paquets ok
$ go build ./... ; go vet ./... ; gofmt -l .              # 0, 0, vide
```

Le gate par capability est juge SUR LES VRAIS TOML du depot : `halo_infinite` declare
`film.bomb_stats`, `halo_5` ne le declare pas — un test qui fabriquerait ses TOML prouverait la
mecanique, pas la configuration livree.

### E5 — Lecture, API, UI

> **EXECUTE LE 2026-09-05, etapes G.3 (serveur) et G.4 (web).**

- [x] Repo de lecture (`platform/duckdb/`) sur la vue `_latest`, type de retour canonique.
      **FAIT** : `Q12cBombStats` sur `match_bomb_stats_latest` UNIQUEMENT (regle ART n2), lue par
      `loadMatchBombStats` — best-effort, comme sa soeur `loadMatchObjectiveStats` : vue absente
      -> WARN structure + colonnes absentes, scoreboard servi ENTIER. Type de retour :
      `domain.ObjectiveRaw`, le meme que les autres modes.
- [~] Service + handler : aucune logique metier dans le handler, aucun SQL inline dans le
      service.
      **COUVERT AUTREMENT, ET C'EST UN CHOIX** : aucun handler neuf. Les cinq colonnes entrent
      dans le bloc `objective` DEJA servi par la fiche de match — `buildScoreboardObjective` les
      recopie sous `HasBomb()`, exactement comme les six autres modes. Un endpoint dedie aurait
      demande au web une seconde requete pour la meme page, et aurait laisse la section
      « Objectifs » incapable de les afficher avec ses deux vues. Le contrat de l'item est tenu :
      zero SQL dans le service (la requete vit dans `platform/duckdb`), zero logique metier dans
      un handler (il n'y en a pas de neuf).
- [x] Multi-titre : branche sur capability, degradation `ErrCapabilityNotSupported` en reponse
      partielle propre. Halo 5 ne doit rien casser.
      **FAIT, et la degradation est plus propre qu'une erreur** : le gate est
      `WithBombStats(caps.Has(games.CapFilmBombStats))` au wiring (jamais `slug ==`). Un titre
      sans la cle ne paie meme pas la requete et n'expose AUCUNE des cinq cles ; le bloc
      `objective` reste celui des autres modes, et la section web se masque d'elle-meme
      (`detectObjectiveMode` rend `null`). Il n'y a pas d'erreur a lever :
      `ErrCapabilityNotSupported` sert a exprimer une reponse partielle quand un endpoint DEDIE
      existe — ici il n'y en a pas, et la reponse est partielle par construction.
- [x] Web : affichage dans la fiche de match sur le patron des autres modes a objectif ;
      libelles via `useFieldLabel()`, i18n **FR et EN**, query key dans `lib/query/keys.ts`,
      zero hex / zero classe Tailwind couleur.
      **FAIT sur le patron EXACT** : la section « Objectifs » a remplace son tableau par DEUX
      VUES le 2026-09-03 (grille `ValueGrid` par joueur, face-a-face par equipe), toutes deux
      pilotees par `objectiveColsFor(mode)`. L'Assaut y entre en trois points — un mode de plus a
      `ObjectiveMode`, `BOMB_COLS`, un discriminant dans `detectObjectiveMode`. DEUX NUANCES,
      dites plutot que passees sous silence : (a) les libelles ne passent pas par
      `useFieldLabel()` mais par la table `t.objectives.cols` de la section, typee
      `Record<MatchViewLocale, MatchViewText>` — donc parite FR/EN par TYPAGE, ce que l'item
      demande ; en devier aurait donne deux sources de libelles dans la meme grille ; (b) aucune
      query key n'est ajoutee, les colonnes voyageant avec la requete de la fiche de match deja
      en place. Zero hex, zero classe couleur : la section suit les jetons `team-ally` /
      `team-enemy` qu'elle emploie deja.
- [x] `make generate-types` apres modification de l'openapi.
      **FAIT** : `make openapi-gen` + `make generate-types` + `make openapi-check` (exit 0), et
      `node tools/check-generated-types-fresh.mjs` exit 0. `wantReplayDocumentFields` 54 -> 56.

**Gate E5** : `make check-types` (apres purge de `node_modules/.tmp`), `make test-web`,
`go test ./internal/api/... ./internal/service/...`.

**Gate E5 — PASSE le 2026-09-05** (D12 complet, pas seulement les trois commandes de l'item) :

```
$ go test ./internal/service/ ./internal/platform/duckdb/ ./internal/api/... ./internal/domain/...
  ./contracttest/...                                                          # EXIT=0
$ go test -tags=integration -p 1 ./internal/platform/duckdb/ -run BombStats -v # EXIT=0
--- PASS: TestGetMatchScoreboard_BombStatsSansCapability_AucuneColonne
--- PASS: TestGetMatchScoreboard_BombStatsVueAbsente_ServiEtAvertit
--- PASS: TestGetMatchScoreboard_BombStatsPresentes_JointesParXUID
$ npm run typecheck   # 0      $ npm run lint        # 0 (28 warnings, EXACTEMENT la baseline)
$ npm run lint:fields # 0      $ npm run test        # 0 (585 fichiers, 6 187 tests)
$ npm run build       # 0      $ node tools/knip-ratchet.mjs # 0 (files/exports/types 0/0)
```

### E6 — Backfill et cloture

> **EXECUTE LE 2026-09-05, etape G.5.**

- [x] CLI de backfill : reutiliser `levelup backfill-replay` si le chemin existant suffit,
      sinon sous-commande dediee. **Ne pas la lancer sur le parc.**
      **SOUS-COMMANDE DEDIEE, et la question a ete tranchee SUR PIECES.** `backfill-replay` ne
      passe PAS par le crochet `replayartifacts` (verifie : aucune reference dans
      `cmd_backfill_replay.go`) — il CUIT et range, il ne projette rien. Et le crochet du fil de
      l'eau ne voit que les artefacts cuits DANS SON CYCLE. Il fallait donc les deux, dans cet
      ordre : `backfill-replay` re-cuit le parc (le schema 39 perime tout artefact anterieur —
      c'est cette passe qui fait NAITRE `bombStats` dans les artefacts), puis
      `backfill-bomb-stats` PROJETTE. L'ordre est ecrit dans l'en-tete de la commande, avec ce
      qui se passe si on l'inverse (un no-op qui le dit, pas une erreur). Patron EXACT de
      `backfill-usage-summary` : aucun film decode, un artefact lu a la fois, reprenable,
      `--dry-run` / `--force` / `--match` / `--limit`, `OpenReadWrite` sous precondition
      « serveur arrete » documentee en tete comme ses freres. **JAMAIS LANCEE** : aucune base de
      prod ouverte de toute l'etape.
- [!] Entree au carnet Notion, section « Sequence a derouler a la release, dans l'ordre » :
      le backfill des stats d'Assaut, sa condition et sa position dans l'ordre.
      **NON TRAITE — le carnet Notion est le carnet de l'UTILISATEUR, pas un livrable d'agent**
      (regle memorisee). La sequence a y inscrire est ecrite ici et dans l'en-tete de la
      commande, prete a etre recopiee : (1) `levelup backfill-replay` serveur arrete, (2)
      `levelup backfill-bomb-stats --dry-run` pour controle, (3) `levelup backfill-bomb-stats`.
      A signaler a l'utilisateur en cloture.
- [x] Entree `.ai/thought_log.md` (date, titre, statut, decision, resultats, prochaine etape).
- [x] Entree au registre `.ai/V7.5/REGISTRE_REPORTS.md` pour le desamorcage, avec sa condition
      de reprise (« un corpus portant un desamorcage avere »).
      **FAIT** : la ligne existait deja (posee par la branche) ; elle portait une DEPENDANCE DE
      LIVRAISON devenue fausse (« la garde `isArmableBombVariant` exclut One Bomb — donc etape
      E2-ter d'abord »). Elle est amendee dans le commit qui la perime : la garde n'existe plus,
      One Bomb publie ses armements, et la condition de reprise du desamorcage — un corpus
      portant un desamorcage AVERE — est la seule qui reste. Un second report a ete AJOUTE :
      `bomb_carriers_killed`, absent partout faute d'une paire tueur/victime datee sur l'horloge
      du match — **il est CLOS le 2026-09-05, lot G.6 du plan d'integration**. Son motif etait
      FAUX sur l'horloge : `killsource.Kill.TimeMS` et `replay.Death.TimeMS` sont le MEME champ
      du MEME enregistrement du chunk highlight, donc les couples etaient deja sur l'horloge du
      match. Seule la VICTIME manquait a `replaybuild.killRefs`, et elle se resout par la meme
      table gamertag -> xuid dans la meme passe. Mesure sur `9f57c612` (le seul film d'Assaut du
      corpus) : `killsRead: true`, 58 couples, 0 ecarte, **3 porteurs tues** (2 + 1, quatre
      joueurs a zero MESURE) — recoupes par `periodsByDeath = 3`.
- [!] Verifier l'etat CI de la branche (`gh run list --branch wt/assaut-stats`) avant de
      declarer le lot clos.
      **SANS OBJET SOUS CETTE FORME** : le travail ne vit plus sur `wt/assaut-stats` mais sur
      `wt/integ-assaut`, une branche d'integration JETABLE qui n'est pas poussee (regle du plan
      d'integration : un seul merge final vers `feat/v75`, un seul push, etape F). L'etat CI se
      juge donc la, sur `feat/v75`, au niveau JOB — c'est la que la condition de ce gate se
      realise. Les gates locaux equivalents sont passes et colles ci-dessus.

**Gate E6** : CI verte au niveau JOB sur la branche.

## 4. CE QUI EST HORS PERIMETRE — noter, ne pas traiter

- Le desamorcage (decision 5).
- La re-cuisson du parc d'artefacts (regle absolue : demander a l'utilisateur).
- Toute reparation d'ancrage `ti=13` en Assaut (chantier A3, ouvert ailleurs).
- Les recompenses `+150/+200/+220` du pied non nommees (chantier medailles, sans urgence).

## 5. DECOUVERTES

> Toute anomalie rencontree et NON traitee s'ecrit ici, datee, avec son fichier:ligne.

- **2026-09-04 (E1) — `intPtr` est squatte par un fichier de TEST du paquet.**
  `internal/analysis/replay/zone_states_test.go:93` declare `func intPtr(v int) *int` dans le
  paquet `replay` ; un helper de PRODUCTION du meme nom ne compile donc pas. Contourne en
  nommant les miens `measuredInt` / `measuredSeconds` (noms qui portent d'ailleurs mieux le
  sens « valeur mesuree »). Non traite : deplacer le helper de test hors du paquet est un fix
  hors perimetre.
- **2026-09-04 (E1) — PIEGE POUR E2 : la jointure de l'armement traverse DEUX horloges.**
  `BombArming.TimeMS` est sur l'horloge du FILM (anneau date via `BombInput.ChunkStartMS`, le
  manifeste), tandis que les periodes de portage et le fil des kills sont sur l'horloge du
  MATCH (`bomb_carries.go` : `matchMS = TimestampUS/1000 - deathOffsetMS`). E1 n'a eu aucun
  pont a faire — chacune de ses quatre statistiques ne consulte qu'une seule horloge. E2, lui,
  DOIT recaler explicitement avant de chercher un lacher dans la fenetre precedant un
  armement ; l'oublier decalerait la jointure de `deathOffsetMS` sans qu'aucun test pur ne le
  voie.

- **2026-09-04 (E2) — LE PIEGE DES DEUX HORLOGES EST REEL, MAIS SON AMPLITUDE N'EST PAS
  `deathOffsetMS`.** La note E1 annoncait un decalage de l'ordre de `deathOffsetMS` (des
  MILLIONS de ms : l'horodatage moteur est un temps depuis le demarrage du jeu). C'est FAUX, et
  la verification sur pieces le montre : les deux horloges partagent presque le meme zero,
  parce que `deathOffsetMS` et le premier paquet du film se compensent. Le pont exact est
  `horlogeMatch = horlogeFilm + premierPaquetDuFilmUS/1000 - deathOffsetMS`, et cette
  difference est deja calculee en production dans `resolveOriginMs` (origin.go) sous le nom
  d'`ecart` — c'est le controle croise de l'origine publiee. MESURE sur les 5 films du gate :
  33 a 114 ms. Consequence pratique : oublier le recalage n'aurait PAS decale la jointure de
  `deathOffsetMS`, mais de ~30-115 ms — soit sous la fenetre de +/-2 500 ms, donc INVISIBLE au
  gate. Le recalage est applique et teste quand meme (`TestBombArmsRecalageHorloge` force un
  offset de 40 000 ms pour que sa disparition rougisse).
- **2026-09-04 (E2) — 3 armements sur 13 n'ont AUCUN lacher : le porteur traverse la pose.**
  Sur `35b75a31` a 299 176 ms, la periode du porteur ([290 194, 308 258], horloge du match)
  COUVRE l'instant arme et ne se ferme que 4 245 ms APRES l'explosion : sur cette pose-la, le
  canal des armes tenues n'emet pas de lacher. Meme figure sur `1c01e34f` (395 764) et
  `69b16f5d` (273 746) — et ces deux-la tombent exactement sur des explosions ou B2 a releve un
  DESACCORD. Une regle de repli « porteur ACTIF a l'instant arme » (celle de `b2PorteurA`)
  nommerait au moins le premier, avec corroboration du statborg. **NON TRAITE** : c'est une
  extension de la regle de jointure ecrite au plan, pas un correctif — elle se statue avec
  l'utilisateur. En l'etat la couverture d'attribution est de 9/13 (69 %) et les 4 restants
  sont publies SANS acteur, ce que le plan demande explicitement.
- **2026-09-04 (E2) — `absInt` est squatte par un fichier de TEST du paquet**, exactement comme
  `intPtr` en E1 : `visee_elevation_ajustement_test.go:99` declare `func absInt(v int) int` dans
  le paquet `replay`. La valeur absolue de `bombPoseurOf` est donc ecrite en ligne (3 lignes)
  plutot que factorisee. Non traite : deplacer ces helpers de test hors du paquet est un fix
  hors perimetre — mais c'est la DEUXIEME occurrence, la troisieme justifiera un lot dedie.

- **2026-09-04 (E3) — `match_objective_events` N'A AUCUNE SEMANTIQUE DE GENERATION, et ca borne
  ce qu'un chemin INSERT-only peut y faire.** PK naturelle `(match_id, seq)`
  (`migration/steps_shared_objective_events.go:30`), aucune vue `_latest`, aucun `decode_pass` :
  contrairement a `match_bomb_stats`, elle ne sait pas « remplacer une passe ». Un INSERT-only
  qui re-ecrirait les memes faits ne pourrait que violer la PK ou empiler deux generations que
  tout lecteur compterait double — et le seul mecanisme de remplacement existant est le
  DELETE-then-INSERT de `ObjectiveEventsRepo.WriteMatch`, explicitement « hors chemin live ».
  E3 ecrit donc les faits UNE FOIS par match (garde par lecture, `slog.InfoContext` quand il
  s'abstient) et alloue `seq` a la suite du maximum existant. **NON TRAITE** : donner a cette
  table une semantique de generation (colonne de passe + vue `_latest`) est un chantier de
  SCHEMA — il toucherait ses autres producteurs et ses lecteurs vivants
  (`api/handlers/match_view_objective_events.go`), ce que le perimetre d'E3 exclut. Consequence
  a garder en tete pour E6 : un re-decodage rafraichit les statistiques d'un match, PAS ses
  faits dates.
- **2026-09-04 (E3) — l'ordre canonique des migrations n'est PAS l'ordre alphabetique des
  fichiers, et l'erreur est muette.** `canonicalOrder` doit reproduire l'ordre d'enregistrement
  REEL de `All()` : les steps enregistres par `init()` (socle `internal/migration`) viennent
  TOUS avant ceux fournis par le provider de titre, quel que soit leur nom de fichier. Poser
  `shared_create_bomb_stats` a sa place « alphabetique » entre `shared_backfill_is_ranked_…` et
  `shared_create_player_squad_offset` a fait echouer `TestSortByCanonicalIsNoOpOnCurrentRegistry`
  au rang 19 ; la bonne place est entre `shared_append_only_weapon_kills_v1` et
  `shared_h5_weapon_kill_kind_v1`. Traite (le test est vert), note ici parce que le prochain qui
  ajoute une migration tombera exactement dessus.
- **2026-09-04 (E2-bis) — UNE PERIODE FERMEE PAR LA PRISE D'UN AUTRE SLOT EST INDISCERNABLE
  D'UN LACHER.** `heldObjectPeriods` (held_object_carry.go:118) ferme la periode ouverte quand
  un AUTRE slot ramasse l'objet, avec `FinParMort = false` et `Ouverte = false` — exactement la
  signature d'une fermeture par lacher. Ni la regle primaire ni le repli ne peuvent donc
  distinguer « il a lache la bombe » de « quelqu'un d'autre l'a ramassee alors que le canal
  n'avait rien emis ». La consequence est benigne ici (dans les deux cas la bombe a QUITTE ses
  mains a cet instant, ce qui est bien ce que la regle primaire lit), et le modele est
  ANTERIEUR a ce lot. **NON TRAITE** : ajouter un troisieme motif de fermeture a
  `HeldObjectPeriod` toucherait Oddball autant que l'Assaut et tous les lecteurs du canal —
  hors perimetre. A garder en tete si un jour la ventilation des `bomb_grabs` doit distinguer
  un vol d'un depot.
- **2026-09-04 (E2-bis) — AUCUN armement AMBIGU sur le corpus des 5 films.** La condition B du
  repli (deux porteurs candidats -> aucun acteur) est donc gatee UNIQUEMENT par les tests purs :
  le corpus ne porte pas un seul instant arme couvert par deux periodes de portage. Ce n'est pas
  une faiblesse du garde-fou, c'est une propriete de l'objet (la bombe n'a qu'un exemplaire, et
  `heldObjectPeriods` borne une periode ouverte des qu'un autre slot ramasse) — mais il faut le
  savoir : `ArmingsAmbiguous` est un compteur qui vaut 0 partout aujourd'hui, et un jour ou il
  monterait, c'est la chronologie du portage qu'il faudrait regarder d'abord, pas le repli.

- **2026-09-04 (E2-ter) — DEUX DES TROIS FILMS ONE BOMB RESTENT RETENUS, ET LE FAIT QUI LES
  RETIENT EST DEJA CONNU.** `c75f33b8` et `df8fcbef` ne publient rien : le premier a une
  explosion sans armement dans la fenetre de sens (395 724), le second a une explosion dont le
  delai corrige vaut 27 845 ms contre ~16 000 pour les trois autres (778 033). Ces DEUX
  instants sont EXACTEMENT les deux entrees d'`a5SansPorteur` (mesure du 2026-08-31, anterieure
  a ce lot : les explosions dont aucun slot de JOUEUR ne porte le point de mode). L'anneau ne
  les explique pas plus que le statborg. **NON TRAITE, ET C'EST UNE DECISION** : la garde 2 est
  tout-ou-rien par arbitrage utilisateur, et la relacher pour recuperer ces deux films
  reviendrait a publier un calque qu'on n'explique qu'aux trois quarts. Ce n'est PAS une
  regression : avant ce lot ces films n'avaient meme pas de couverture (garde de nom), ils en
  ont une desormais qui DIT pourquoi elle se tait. Condition de reprise : comprendre ce que
  sont les explosions « sans porteur » — meme verrou que le desamorcage, meme corpus a
  demander a l'utilisateur.
- **2026-09-04 (E2-ter) — LE DENOMINATEUR DU « 100 % » SUR `9f57c612` EST 2/4, PAS 4/4.** Le
  journal d'E2-bis ecrit « les 4 de `9f57c612` sont jugees » ; le gate imprime le contraire, et
  c'est lui qui a raison : sur les explosions 83 322 et 353 160, le statborg NE NOMME PAS le
  detonateur, elles sortent donc du denominateur du critere (a) exactement comme une periode
  non pontee. Le verdict ne change pas (zero desaccord neuf, le gate est vert) ; c'est la PROSE
  du plan qui etait fausse. Rien a corriger dans le code — le juge est `bpJugeExplosion`, que
  ce lot ne touche pas ; la ligne d'E2-bis est amendee ici plutot que reecrite la-bas.
- **2026-09-04 (E2-ter) — LA MECHE HUSKY MESUREE EN PRODUCTION (5 089 ms) TOMBE SUR LA VALEUR
  QUE LES TESTS CODAIENT EN DUR (`b3MecheMS` = 5 100 ms).** Ce n'est pas une coincidence, c'est
  la meme grandeur lue deux fois — mais l'une est desormais MESUREE par film et l'autre reste
  une constante de test, avec un `if id == "1c01e34f"` en clair. **NON TRAITE** (hors
  perimetre) : `b3MecheMS` sert les instruments B2/B3 qui jugent le PORTAGE, pas l'armement ;
  les faire lire la meche mesuree serait un lot a part. A garder en tete : c'est une troisieme
  copie de la meche dans le depot.
- **2026-09-04 (E2-ter) — UN SEGMENT NE MESURE AUCUNE DISPERSION QUAND LE FILM N'A QU'UNE
  EXPLOSION.** La branche DISPERSION de la garde 2 exige `len(delays) >= 2` : sur un film a une
  seule explosion (le corpus en a un, `34bb3bc8`), la garde se reduit a la COUVERTURE, c'est-a-
  dire « un armement existe dans les 120 s qui precedent » — nettement plus faible que la
  fenetre de +/-600 ms d'avant. La limite est ECRITE dans `measureBombFuse` plutot que masquee.
  **NON TRAITE** : durcir demanderait soit une meche supposee (ce que le lot supprime), soit un
  seuil absolu invente. A revoir si un film a une explosion publiait un jour un armement
  visiblement faux.

### 2026-09-04 — LES DEUX RECHERCHES PARALLELES, ET CE QU'ELLES CHANGENT

**(a) La famille `BombStats` est de la TELEMETRIE, pas de la replication — question fermee.**
Mesure Ghidra de premiere main sur `HaloInfinite.exe` : les 9 statistiques
(`BombDetonations`, `BombDefusals`, `BombPlants`, `BombCarriersKilled`, `BombDefusersKilled`,
`BombPickUps`, `BombReturns`, `KillsAsBombCarrier`, `TimeAsBombCarrier`, adresses
`14381e790`..`14381f1b8`) sont des champs Bond
`Microsoft.Halo.HaloStats.Bond.HaloInfinite.Match.Stats.BombStats` (`14373a2a0`), ecrits par un
serialiseur d'evenement de telemetrie (`FUN_1434771b8`). **Ce ne sont pas des composants
d'entite : le film ne peut pas les repliquer, par construction.** C'est la MEME cause unique
que le silence de l'API. La question « pourquoi l'Assaut n'a pas de stats » est donc close, et
la reponse est structurelle.

**CONSEQUENCE DE NOMMAGE, a tenir en E1/E5** : notre `bomb_detonations` **n'est pas**
`BombStats_BombDetonations`. C'est le compteur generique de points de mode (statborg `comp 0`
canal A), qui vaut numeriquement les explosions parce qu'en Assaut un point de mode EST une
explosion. La stat reste juste et gatee (4/4 en moities disjointes) ; l'en-tete du code doit
dire cette provenance exacte, et l'UI ne doit pas laisser croire qu'on lit un compteur du
moteur.

**(b) Statborg et pied de film : negatifs bornes, ne pas rouvrir.** 9 films,
6 904 enregistrements, 133 unites porteuses sur 928 (composants 0-57, canaux A/B/C/D, slots
joueur ET equipe, progression ET transition), 17 armements juges contre l'oracle `ti=12`,
plancher sur 4 500 instants aleatoires : **0 unite sur 133** tient couverture + selectivite
(meilleure 2,59x pour un seuil de 3x). Cote pied : l'octet qu'on prenait pour un indice de type
est le petit octet de la VALEUR de recompense (2 490 blocs sur 2 490), l'enumeration est close,
et +20 « couvre » 24 % des armements pour un plancher de 22 %. Les canaux C et D N'ETAIENT PAS
neufs : A6 = 28 composants x 4 canaux, entre dans le meme commit que leur decodage.

**(c) Le negatif `bombstate` du 01/09 portait sur la MAUVAISE CHAINE — la porte n'est pas
fermee.** Il cherchait `murmur3("bombobjectstate")`, le nom du TYPE, que le moteur ne hache
jamais comme cle. Verifie au passage : `FUN_140748a74` et `FUN_140748d64` ne sont pas deux
fabriques de hachage distinctes, c'est UNE fonction que Ghidra a scindee. Bonne fonction,
mauvaise chaine.

**(d) Le moteur NOMME ce qu'on cherche** : `prop_stats_mode_assault_bomb_arms` et
`prop_stats_mode_assault_bomb_disarms` (`14380f8f0`, `14380fbf0`), freres de `_detonations`
dans la meme famille. Ce qui manque n'est pas le vocabulaire mais l'adresse dans le film. Et
`BombObjectState` est replique JUSQU'AU HUD (`CenterScoreboard_BombState` `14382d300`,
`CenterScoreboard_BombNormalizedCaptureProgress` `14382d160`, `bomb_icon_reader_component`
`143803848`) : un client ne peut afficher que ce qu'on lui replique.

**(e) LE LUA NE PORTE PAS L'ACTEUR — ceci confirme E2 comme seule voie.**
`primitive_carriable_arming_base` (tag `25af9c45`) emet six evenements :
`OnInitializationStarted / Interrupted / Completed` et `OnDeactivationStarted / Interrupted /
Completed`. Le seul porteur d'identite est l'EQUIPE (`activatingTeam`, via
`Item_GetInventoryUnit` -> `Player_GetMultiplayerTeam`) ; il n'existe aucun `activatingPlayer`.
**Donc meme si un canal d'armement tombe, il donnera l'instant et l'equipe, jamais le joueur :
l'acteur restera a fermer par le canal des armes tenues, c'est-a-dire par E2.**

**(f) Le desamorcage : le moteur le distingue, le corpus ne le porte pas.**
`OnDeactivationCompleted` vs `OnDeactivationInterrupted`, etat `Disarming`(3), et deux
appareils declares separement (`ArmingDeviceTag` / `DefuseDeviceTag`). Cote film le negatif
tient : sur Neutral/Husky, 17/17 des poses completes ont explose — personne n'a desamorce. La
decision 5 est INCHANGEE, et sa condition de reprise se precise : il faut un corpus qui porte
un desamorcage avere, ce qui se DEMANDE a l'utilisateur avant de se faire.

**PISTE OUVERTE, NON TRAITEE DANS CE PLAN** (elle ne bloque aucune etape) : chercher par
EGALITE de nom dans le dictionnaire ECS — `bomb_icon_reader_component` d'abord (une lecture de
`chunk_00`, aucun corps de record decode), puis les trois `prop_stats_mode_assault_bomb_*`
comme aiguilles sur les composants 28-57. A NE PAS rouvrir en revanche : `ti=11` delta,
`ti=13`, le pied, les composants 0-27 — quatre negatifs mesures et bornes. Et `ti=11 i14 state`
est deja porte, publie et REFUTE (p = 0,61 / 0,63 / 0,09) : ne pas le reproposer.

## 6. REPRISE DE SESSION

Avancement = les cases de la section 3 de ce fichier. Reprendre a la premiere etape dont le
gate n'est pas colle. `git log --oneline -10` sur `wt/assaut-stats` pour situer le dernier lot.
