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
- [ ] Backfill reel des 114 artefacts (serveur arrete au prealable)
- [ ] Commit(s) S1 (+ entree thought_log : ajoutee le 2026-09-04 par l'executant)

  AVERTISSEMENT PILOTE : deux executants ont ecrit EN PARALLELE dans ce worktree le
  2026-09-04 apres-midi (la session interrompue a repris d'elle-meme). Les implementations
  ont converge, mais relire le diff entier avant commit.

## Lot S2 — Agregat de session et contrat

- [ ] Service d'agregat sur analysis.ComputeSessions (gap 120 min) : somme joueur/camp/
      lobby, deux effectifs moyens (-> deux parites), etendue par match, matchs au-dessus
      de la parite, cadence par 10 min (duree des matchs MESURES)
- [ ] Objectifs agreges depuis match_objective_stats_latest (rien a produire)
- [ ] Contrat : QUE du normalise + champ « matchs mesures / matchs de la session »
- [ ] Verification pilote contre les chiffres de reference §7 du handoff (parite joueur
      11,9 %, equipe 24,2 %, tableau des 9 grandeurs)
- [ ] Commit(s) S2 + entree thought_log

## Lot S3 — Front Sessions (Solo + Escouade)

- [ ] Seize formes de la maquette dans features/session-detail, ValueGrid reutilise,
      jetons team-ally/team-enemy et squad-player-1..3, i18n FR+EN par typage
- [ ] Gate : make check-types + make test-web + verification visuelle sur le temoin
      (session 2026-07-31, JGtm)
- [ ] Commit(s) S3 + entree thought_log

## Decouvertes hors perimetre (notees, non traitees)

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

## Journal

- [2026-09-04] Etape 0 close. Verification du handoff : conforme, deux ecarts mineurs
  notes ci-dessus. Base worktree = feat/v75 @ da616828f + purge sidecar. S1 confie a un
  agent executant (implementation seule, commits et verification par le pilote).
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
