# Decouvertes du chantier Tactique — registre a part (decision utilisateur 2026-09-07)

Regle : on CONSIGNE, on ne traite pas. Chaque entree : date ; fichier:ligne ; fait ; condition de reprise.
Le §7 du plan `.ai/PLAN_TACTIQUE_2026-09-06.md` reste la source des decouvertes des phases 1 a 7A ;
ce fichier prend le relais a partir de 7C et recoit toute nouvelle decouverte.

## 7C — faits d'isolement au sync (branche `feat/tactique`, HEAD `c894e0282`)

- 2026-09-07 ; `internal/sync/killcollector/roster.go:157` + `analysis/replay/death_context.go:194` ;
  **FAIT FAUX POSSIBLE** : `ArriveeMS`/`DepartMS` (horloge API, depuis `start_time`) sont compares a
  `time_ms` du journal (horloge FILM, frame 0) ; ecart T0 de 17 a 92 s selon les matchs ; pour les
  morts du debut de partie un coequipier present est sorti du total -> `teammates_visible = 0`,
  `nearest NULL` = « isolee ». NON CORRIGE, NON PUSHE EN PROD (branche de feature). Reprise : soit
  retirer l'usage de la participation en V1 (`teammates_left` = 0), soit caler avec
  `match_registry.real_start_time` / `t0_quality` (appareil existant, `cmd/backfill_t0_film`).
  **A traiter AVANT tout merge de `feat/tactique`.**
- 2026-09-07 ; `analysis/replay/lives_export_test.go:180-193` ; test d'orthogonalite tautologique
  (trois litteraux) ; l'orthogonalite `end_cause`/`named_by` n'est pas prouvee sur un vrai pont.
  Reprise : fixture `ResolveSlotXUID` avec fermeture par reapparition (`closeByRespawn`, `fire = nil`).
- 2026-09-07 ; `sync/killcollector/isolation_facts_integration_test.go` ; le test film reel ne peut
  jamais passer (roster `fakeRoster{}` vide -> `Equipes` vide -> skip). Reprise : roster construit
  depuis `replay.ScanDeaths(film)`, exiger vies > 0 ET contextes > 0, `t.Fatal` sinon ; commande
  `KILLSOURCE_FIXTURES=<racine>/data/cache/film_chunks go test -count=1 -tags=integration -p 1 -run FaitsDIsolementFilmReel ./internal/sync/killcollector/`.
- 2026-09-07 ; `sync/killcollector/isolation_facts.go:176` ; un pont non publiable
  (`IndexDisagreements > 0`) fait tomber toutes les morts dans `killsource_isolement_morts_sans_lieu`.
  Reprise : compteur dedie.
- 2026-09-07 ; `sync/killcollector/roster.go` ; `LEFT JOIN match_registry` sans test. Reprise : test
  `IdentitiesForMatch` sur base `:memory:` migree, participants sans registre.
- 2026-09-07 ; `platform/duckdb/no_raw_rating_reads_test.go:44` ; `match_kill_events`,
  `kill_positions`, `match_bomb_stats` absentes de la regex de lecture brute. Reprise : les ajouter.
- 2026-09-07 ; `sync/killcollector/positions.go:103` ; le refus des positions au sync etait
  journalise en Debug seulement — une table neuve vide ne se remarque pas. Reprise : WARN + compteur.
- 2026-09-07 ; `sync/killcollector/capture.go` ; la capture de positions est desormais branchee au
  sync de PROD (elle ne l'etait que dans le backfill) : scan complet des bipedes par match synchronise.
  Cout a observer au premier deploiement (duree de l'etape post-sync, `PostSyncBudget`).
- 2026-09-07 ; `api/wire/registry.go` + `cmd/server/main.go:1421` + `api/server_apiv1.go` ;
  `settingsStore.Load()` en erreur se degrade en silence chez deux appelants (dont le cron de purge :
  settings illisible = retention illimitee = purge desactivee sans un mot). Reprise : WARN.
