# PLAN D'EXECUTION — Usages d'equipement, socles et objectifs par SESSION (voie BDD)

> Source : `.ai/HANDOFF_SESSION_USAGE_BDD_2026-09-04.md` (worktree principal). Le handoff
> fait foi sur le fond ; ce fichier ne porte que l'avancement et le journal d'execution.
> Pilote/verificateur : session du 2026-09-04. Executants : agents en worktree dedie.
> Doctrine confirmee par l'utilisateur le 2026-09-04 : la mecanique de sync existante
> fait foi — on se branche sur l'etape post-sync replaybuild existante et le pipeline
> persist existant, on ne reinvente RIEN.

## Etape 0 — Socle du worktree (pilote)

- [x] Verifier le handoff sur pieces (worktree, WIP b8dc38107, points de branchement,
      match_objective_stats_latest, document_pickups.go, ValueGrid)
- [x] Fusionner feat/v75 (da616828f) dans wt/session-usage — apporte ValueGrid pour S3
- [x] Purger les restes sidecar : ReplayUsageSummaryPath supprime, doc CapFilmUsageSummary
      reecrite voie BDD ; PadWeaponFamilyKey conserve (seule piece reutilisable du WIP)
- [x] Gate : go build ./... + go test games/title/analysis-replay — verts
- [x] Commit `35baa7863`

Constats de verification (ecarts mineurs au handoff, sans consequence) :
- Le commentaire de `engine_postsync.go` cite `engine_postsync_replay.go` qui n'existe
  pas : `runReplayArtifacts` vit dans `internal/sync/convergence.go:594`.
- Le WIP n'avait PAS declare `film.usage_summary` dans `capabilities.toml` — a faire en S1.
- L'artefact maquette (2ec1b8eb...) n'est pas relisible par cette session (lecteur public
  non membre) ; sans effet sur S1/S2, a re-verifier avant S3 aupres de l'utilisateur si
  le detail visuel des seize formes est necessaire.

## Lot S1 — Socle de donnee (table append-only + producteur post-sync + backfill)

- [x] Table(s) append-only dans shared_matches_v2 (recette ADR 0026 : id PK + written_at
      + vue `_latest`), migration dans internal/migration/order.go (modele kill_positions)
      — steps_shared_usage_summary.go : match_usage_players + match_usage_films, vues
      `_latest` PAR PASSE (summary_pass, modele match_weapon_hit_distance)
- [x] Persister INSERT-only dans internal/persist (modele kill_position_persister)
      — usage_summary_persister.go (+ test integration), aucune allowlist anti-ART
- [x] Projection pure dans internal/analysis/replay (artefact -> lignes), frontiere
      socle d'arme / socle de bonus par PadWeaponFamilyKey UNIQUEMENT
      — cles de ventilation NORMALISEES par PadWeaponFamilyKey (decision executant)
- [x] Branchement post-sync dans l'etape replaybuild existante, gate CapFilmUsageSummary
      — replayartifacts/usage.go, patron t0film (projection disque, burst apres cuisson)
- [x] Declaration capability halo_infinite (supported) ; halo_5 absente
- [x] CLI backfill (modele backfill-killsource) : un artefact a la fois, reprenable,
      --force, --match, --dry-run — cmd_backfill_usage_summary.go
- [x] Tests : fixture projection, assertion nominative socle-bonus, exclusion grenades,
      persister anti-ART sur DB migree — go test cible vert ; suite complete + integration
      lancees, resultat au rapport de l'executant
- [x] Controles croises (dry-run bac a sable, data/ intact) : 696a9d7c = 26 nommees /
      8 anonymes, 10 powerup_camo en bonus et ZERO en pad_pickups (executant) ;
      b8a44fe8 = 51 / 11 (executant + pilote). Session 2026-07-31, 8 films
      (4577fcc4, a4083bd2, b0fe12b1, 16ea3668, af3500aa, e85d7bad, 3923bede, b8a44fe8) :
      193 nommees EXACT ; anonymes = 82 et non 102 — VERDICT PILOTE : le chiffre 102 du
      handoff est FAUX, il comptait les 20 occupations powerup_overshield (The Pit 9,
      Recharge Forge 11) comme socles d'arme anonymes tout en excluant powerup_camo,
      asymetrie contraire a la regle §4 du handoff lui-meme (verifie sur artefacts bruts :
      overshield etiquete par NOM, comme camo). 82 + 20 = 102, ecart integralement
      explique. Bonus session : camo 45, overshield 20. Signale a l'utilisateur.
- [x] Backfill reel (serveur arrete au prealable) : 106 ecrits, 0 echec, 4 s, sur les
      106 artefacts du corpus (1853 matchs du registre sans artefact = hors fenetre de
      retention, attendu). Reprise verifiee : second passage = 106 deja a jour, 0 ecrit.
      Racine composite scratchpad (config S1 + jonction data reelle) — les deux worktrees
      intacts. Serveur air relance depuis apps/go-api (piege : lance depuis la racine il
      boucle sur exit status 1, .air.toml vit dans apps/go-api) et port 8000 re-ouvert.
- [x] Commit S1 : `d448c3328` (+ etape 0 : `35baa7863`). Entree thought_log executant
      + cloture pilote du 2026-09-04.

  AVERTISSEMENT PILOTE : deux executants ont ecrit EN PARALLELE dans ce worktree le
  2026-09-04 apres-midi (la session interrompue a repris d'elle-meme). Les implementations
  ont converge, mais relire le diff entier avant commit.

## Lot S2 — Agregat de session et contrat

> EN COURS depuis le 2026-09-04 soir. ARBITRAGE UTILISATEUR (2026-09-04, ~21h30) :
> le pilotage S2/S3 revient a la session « levelup-go-migration-ad » (pilote de
> l'etape 0, du commit S1 d448c3328 et du backfill reel). La session
> « levelup-go-migration-8a » a ete notifiee par message direct : ne pas lancer
> S2/S3, ne pas committer, ne pas ecrire dans ce worktree. Une seule main au volant.

- [x] Service d'agregat sur analysis.ComputeSessions (gap 120 min) : somme joueur/camp/
      lobby, deux effectifs moyens (-> deux parites), etendue par match, matchs au-dessus
      de la parite, cadence par 10 min (duree des matchs MESURES)
      — sessionusage.ComputeUsage + attachSessionUsage (session_page_usage.go),
      bloc `usage` ATTACHE a POST .../pages/sessions/detail (design B arbitre,
      patron IntensityRows/FirstBlood — pas d'endpoint dedie)
- [x] Objectifs agreges depuis match_objective_stats_latest (rien a produire)
      — LoadObjectiveRoleRows (SUM generes depuis narrative.ObjectiveRoleColumns,
      SOURCE UNIQUE arbitree ; la table concurrente de sessionusage/objectives.go
      a ete SUPPRIMEE) ; roles par famille derives (intersection role x vocabulaire
      narrative), jamais de liste parallele
- [x] Contrat : QUE du normalise + champ « matchs mesures / matchs de la session »
      — domain/session_usage.go (design B) + lignes d'escouade (SquadPlayers au bloc,
      Squad par metrique et par role — contexte Filters.MatchContext=squad, resolution
      miroir sessionCoreTeammates, cap 3) ; openapi-gen regenere, garde vert
- [x] Verification pilote contre le §7 du handoff (sonde SQL pilote sur copie scratchpad) :
      CINQ grandeurs EXACTES au dixieme (socles 45,6/20,5/9,3 · camo 50,0/14,8/7,4 ·
      surbouclier 45,5/20,0/9,1 · grappin 44,4/50,0/22,2 · laches 49,1/24,6/12,1).
      DEUX references du §7 sont FAUSSES (meme famille d'erreur que le « 102 ») :
      (a) MURS 33,3/42,9/14,3 : le mock comptait les 21 lignes wall (appareils lachés
      compris — 0x2974c233, 86 % dropped) ; la regle validee vue-match (panneaux
      deployes seuls, 0x686b40c9) donne lobby 7, camp JGtm 0, JGtm 0 — JGtm n'a
      DEPLOYE aucun mur sur le temoin, ses « 3 murs » etaient des appareils laches en
      mourant. Regle S1 conservee, temoin corrige.
      (b) PARITES 11,9/24,2 : le mock comptait tous les inscrits (lobby moyen 8,4) ;
      le canon du depot (LobbySizesAtCompletion, presents a la fin) donne 8,0 -> parites
      12,5 % / 25,0 %. Canon conserve (regle 14 : reutiliser les KPI existants).
      ROLE « PRENDRE » — mesures executant (les trois classifications candidates,
      aucune ne reproduit le §7 56,5/11,5/6,5 calcule hors depot) : DECISION UTILISATEUR
      du 2026-09-05 : flag_grabs EXCLU (les porter-jeter tactiques en chaine gonfleraient
      la grandeur) — table narrative conservee telle quelle (48,3/14,0/6,7 au temoin).
- [x] Revue adversariale S2 : ronde 1 = 1 relecteur frais (lentille correction des
      donnees) — 3 P1 (scope camp connu/inconnu croise : parts > 100 % possibles ;
      TeamPer10Min et team_share_of_lobby a 0-pour-inconnu) + 3 P2 (copie ToAnySlice,
      champs FilmRow morts, cadences gonflees par match sans echelle de temps), 18
      conditions qui tiennent. Ronde de correction (executant) : regle de scope unique
      appliquee aux 4 sites, tests anti-mutants (600 % -> 100 %), DTO TeamTotal/
      MatchesAboveTeamParity passes en pointeurs, openapi regenere. Ronde 2 (contexte
      frais, corrections seules) : AUCUN defaut recevable, 6/6 tiennent.
- [x] Commit(s) S2 (PILOTE) + entree thought_log [x]

## Lot S3 — Front Sessions (Solo + Escouade)

- [ ] Seize formes de la maquette dans features/session-detail, ValueGrid reutilise,
      jetons team-ally/team-enemy et squad-player-1..3, i18n FR+EN par typage
- [ ] Gate : make check-types + make test-web + verification visuelle sur le temoin
      (session 2026-07-31, JGtm)
- [ ] Commit(s) S3 + entree thought_log

## Decouvertes hors perimetre (notees, non traitees)

- (utilisateur, 2026-09-05) Idee future pour le role « prendre » : un filtre anti-spam
  des prises de drapeau — distinguer une prise PORTEE vers le socle des porter-jeter
  tactiques en chaine. Exige la chronologie des portages, qui vit dans les artefacts de
  film (les episodes de portage drapeau/bombe existent deja cote rejeu 2D), pas dans
  match_objective_stats. Chantier separe si retenu.
- (revue adversariale 2026-09-04, P2 consignes) `usage_summary_persister.go:71-85` :
  l'atomicite films+joueurs (une seule transaction) n'est prouvee par aucun test — un
  sabotage en deux transactions passerait inapercu et la reprise (qui ne lit que
  films_latest) marquerait « a jour » un match aux joueurs manquants.
- (idem) `cmd_backfill_usage_summary.go` : le gate capability (l.122) et la voie
  `--match` (l.151-153, un id absent du registre ecrit des lignes orphelines en shared)
  n'ont pas de test ; seule la cle de reprise est couverte (test ajoute en ronde 1).
- (idem, dette pre-existante) `match_weapon_hit_distance` manque a `tablesProtegees`
  de no_art_patterns_test.go — meme trou que celui corrige ici pour les tables neuves.
- (idem) `killcollector.PostSyncHook` memorise ses capabilities pour la vie du hook ;
  le chemin replayartifacts les relit a chaque cycle — deux politiques a harmoniser un jour.
- (executant) le chemin « ouvrier » de placement ne cuit pas localement, donc ne produit
  pas de resume au fil de l'eau : le CLI backfill couvre ce cas — a garder en tete prod.
- (pilote) le corpus reel compte 106 artefacts au 2026-09-04 (le handoff en annoncait
  114) — quelques-uns purges/archives entre-temps, sans consequence.
- (pilote, gate complet -p 1 du 2026-09-04 soir) DEUX rouges PRE-EXISTANTS de la base
  feat/v75, etrangers au lot (reproduits a l'identique sur feat/v75 nu) :
  `TestNoRewrittenSlotBand` (archlint — faille_activation_research_test.go du commit
  d28a3aa6a, chantier translocateur, reecrit la regle de bande de slots ; tache separee
  creee) ; `internal/himap` : VERDICT TOMBE (session 8a, 2026-09-04 soir) —
  `TestBalayageCoquille/aquarius_map` PEND a l'identique sur feat/v75 nu et sur
  wt/session-usage (timeout a 2/10/15 min, blocage ~1m14 dans le sous-test), rouge
  PRE-EXISTANT etranger au chantier ; fiche de tache separee proposee a l'utilisateur.
  `TestCareerLive_NilAPIResponse_NotCached` etait un flaky de contention (vert isole
  des deux cotes). Les paquets DU LOT sont tous verts, integration -p 1 comprise.
  CONSEQUENCE GATES S2/S3 : `go test ./...` restera rouge sur ces deux paquets quoi
  qu'on fasse — depouiller les FAIL nominativement (motif ancre `^--- FAIL:` + code de
  sortie), ne jamais attendre un exit 0 global ni maquiller ces deux rouges en verts.

- (executant S2, 2026-09-05) Ordre de mise en place d'une DB shared de test : appliquer
  EnsureSharedSchema AVANT migration.RunForDB(TargetShared) fait echouer la migration
  `add_mv_player_matches_fr_cols` (match_registry preexiste sans les colonnes _fr que
  le step halo n'a pas encore ajoutees). Sur DB vierge, RunForDB d'abord puis
  EnsureSharedSchema passe (session_usage_aggregate_integration_test.go). Fragilite
  d'ordonnancement a regarder un jour — non traitee.
- (executant S2) `cmd/inspect_bp/main.go` pointe par defaut sur l'ancien chemin v6
  `data/warehouse/metadata.duckdb` (mort depuis l'ADR 0008) — non traite.
- (executant S2) L'entree thought_log S1 annoncee par la cloture pilote n'est pas dans
  `.ai/thought_log.md` du worktree (premiere entree au 2026-09-03) — probablement dans le
  worktree principal ; non traite.

## Journal

- [2026-09-04] Etape 0 close. Verification du handoff : conforme, deux ecarts mineurs
  notes ci-dessus. Base worktree = feat/v75 @ da616828f + purge sidecar. S1 confie a un
  agent executant (implementation seule, commits et verification par le pilote).
- [2026-09-05] S2 — reconciliation TERMINEE par l'executant de reprise (3e session,
  apres deux coupures) : design B retabli dans domain/session_usage.go (le fichier du
  worktree portait le contrat concurrent a endpoint dedie — remplace par la sauvegarde
  s2-executant-B + types escouade) ; table de roles concurrente SUPPRIMEE de
  sessionusage/objectives.go, source unique narrative.ObjectiveRoleColumns, sommes par
  role GENEREES au repo (LoadObjectiveRoleRows) ; lignes d'escouade (bloc + metriques +
  roles) via ResolveTrackedSquad (miroir sessionCoreTeammates, cap 3) ; service
  attachSessionUsage (patron IntensityRows), DI gated CapFilmUsageSummary dans
  registry_pages (~l.292), degradation unsupported/load_failed jamais 500 ;
  openapi-gen regenere + garde vert ; tests unitaires (14) + integration persister
  reel verts. MESURE flag_grabs consignee au lot S2 ci-dessus. RESTE PILOTE : verdict
  §7, relecture du diff entier, commit S2.
- [2026-09-04] S1 — revue adversariale (2 relecteurs frais, lentilles anti-ART et
  couverture de tests) : 2 P1 convergents corriges (vue players_latest desormais assise
  sur films_latest — l'autorite de passe est la ligne film, toujours ecrite ; tables
  neuves ajoutees a tablesProtegees) + 2 commentaires inverses corriges + 2 tests ajoutes
  (passe B vide de joueurs supplante A ; cle de reprise du CLI, 6 cas + artefact corrompu).
  Ronde 2 (contexte frais, corrections seules) : 0 P0/P1, 1 P2 de commentaire corrige dans
  la foulee (en-tete du garde des familles). 29 + 14 conditions verifiees qui tiennent.
  Gates pilote re-passes apres correction ; bac a sable re-valide de bout en bout
  (migration sur copie du schema prod, 106 ecrits 0 echec, reprise OK, recoupement
  pad_named/pad_pickups exact a 1099).
