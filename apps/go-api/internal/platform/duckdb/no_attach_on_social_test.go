// Package duckdb — test sentinel anti-régression : aucun ATTACH ne doit être
// exécuté sur la connexion shared_social.duckdb (socialDB) dans tout le projet.
//
// Contexte : ATTACH sur une conn RW écrit dans le WAL une entrée non-rejouable
// au reboot (bug DuckDB upstream #7659). Cf. ADR 0021 + thought_log 2026-05-27.
//
// Ce test parse tous les fichiers .go du projet (sauf vendor + tests) et
// vérifie qu'aucune occurrence ne :
//
//  1. Appelle `attachGlobalXuidAliases(*, socialDB, *)` ou `attachShared*(*, socialDB, *)`
//  2. Exécute un SQL contenant `ATTACH ` :
//     a. directement sur une variable nommée `socialDB`, `sharedSocialDB`, etc.
//     b. via un selector chain `*.SharedSocial.Exec(...)` ou `*.socialDB().Exec(...)`
//        (ADR 0021 Phase 2.4 — couvre les patterns repo r.pdb.SharedSocial.Exec et
//        r.socialDB().Exec qui contournaient la détection v1).
//
// Whitelist : aucune entrée — l'invariant ADR 0021 est strict (aucun ATTACH
// sur la conn shared_social, jamais, pour aucune raison). Si un cas légitime
// émerge, l'ajouter ici avec justification + lien commit + audit CHECKPOINT
// post-ATTACH garanti.
//
// Limite : ne détecte pas les ATTACH via abstractions (interfaces, méthodes
// nommées différemment, sql.Tx obtenu via shared_social puis tx.Exec). Pour
// ces cas, recourir à l'audit manuel (cf. .ai/audit_shared_social_writes_2026-05-27.md).

package duckdb

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoATTACHOnSocialDB scanne tout le projet pour les appels à
// attachGlobalXuidAliases ou méthodes ATTACH-like ciblant socialDB. Échoue si
// l'un est trouvé — empêche la régression du bug WAL corruption.
func TestNoATTACHOnSocialDB(t *testing.T) {
	// Remonter à la racine apps/go-api/ pour scanner tous les packages.
	root := findGoAPIRoot(t)

	var violations []string

	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "tmp" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil // skip tests (peuvent contenir ATTACH pour reproduire le bug en test)
		}

		f, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return nil // skip fichiers non-parsables (CGO, build tags, etc.)
		}

		ast.Inspect(f, func(n ast.Node) bool {
			// Cas 1 : appel à attachGlobalXuidAliases(*, socialDB, *) ou variations
			if call, ok := n.(*ast.CallExpr); ok {
				if ident, ok := call.Fun.(*ast.Ident); ok && isATTACHFuncName(ident.Name) {
					for _, arg := range call.Args {
						if argIdent, ok := arg.(*ast.Ident); ok && isSocialDBVarName(argIdent.Name) {
							pos := fset.Position(call.Pos())
							violations = append(violations, formatViolation(pos, ident.Name+"("+argIdent.Name+")"))
						}
					}
				}
			}
			// Cas 2 : string literal SQL "ATTACH ..." dans un Exec/Query ciblant socialDB.
			// Phase 2.4 : couvre deux patterns receiver :
			//   a) variable directe nommée socialDB-like (ex: socialDB.Exec, sharedSocialDB.Exec)
			//   b) selector chain via SharedSocial (ex: pdb.SharedSocial.Exec, r.pdb.SharedSocial.Exec)
			//      ou via une méthode socialDB() (ex: r.socialDB().Exec).
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					methodName := sel.Sel.Name
					if methodName != "Exec" && methodName != "ExecContext" && methodName != "Query" && methodName != "QueryContext" && methodName != "ExecRecovered" {
						return true
					}
					recvLabel := socialReceiverLabel(sel.X)
					if recvLabel == "" {
						return true
					}
					for _, arg := range call.Args {
						if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
							if containsATTACHKeyword(lit.Value) {
								pos := fset.Position(call.Pos())
								violations = append(violations, formatViolation(pos, recvLabel+"."+methodName+"(ATTACH ...)"))
							}
						}
					}
				}
			}
			return true
		})
		return nil
	})

	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("ATTACH/DETACH detecté sur socialDB — bug DuckDB #7659 ré-introduit :\n%s\n"+
			"Fix : faire la jointure cross-DB en Go (cf. ops/media_associate.go) au lieu de ATTACH sur shared_social RW.",
			strings.Join(violations, "\n"))
	}
}

// TestNoUnauthorizedSharedSocialMention (ADR 0021 Phase 2.4) — scope file-level :
// vérifie qu'aucun fichier Go non-test ne mentionne "shared_social" (chaîne
// littérale) en dehors de la whitelist explicite `sharedSocialFilesWhitelist`.
//
// But : éviter qu'un nouveau site d'écriture émerge dans un fichier inattendu
// (ex: un nouveau handler qui ferait `db.Exec("INSERT INTO shared_social...")`
// directement au lieu de passer par MediaRepo / SocialPersister).
//
// Pour ajouter un fichier légitime : enrichir `sharedSocialFilesWhitelist` avec
// une description explicite. Si un fichier apparaît dans les violations sans
// raison valable, le bug est dans le code (router via Persister).
func TestNoUnauthorizedSharedSocialMention(t *testing.T) {
	root := findGoAPIRoot(t)
	violations, err := listForbiddenSharedSocialMentions(root)
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	if len(violations) == 0 {
		return
	}
	t.Fatalf("référence à 'shared_social' hors whitelist détectée — "+
		"si légitime, ajouter une entrée dans sharedSocialFilesWhitelist (no_attach_on_social_test.go) "+
		"avec description justifiée :\n  - %s",
		strings.Join(violations, "\n  - "))
}

// TestSharedSocialWhitelistEntriesPointToExistingFiles (V4d, VF-6) — self-check
// anti-pourrissement : chaque clé de sharedSocialFilesWhitelist est un CHEMIN de
// fichier (pas un motif) ; elle doit désigner un fichier EXISTANT. Une entrée
// dont le fichier a disparu (déplacé/supprimé par un refactor) est un trou
// latent — c'est le défaut révélé par VF-6 (social_persister_combined.go
// « si présent », jamais créé). Un fichier recréé plus tard à ce chemin
// mentionnerait shared_social sans déclencher le sentinel.
func TestSharedSocialWhitelistEntriesPointToExistingFiles(t *testing.T) {
	root := findGoAPIRoot(t)
	for rel := range sharedSocialFilesWhitelist {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			t.Errorf("sharedSocialFilesWhitelist : entrée %q pointe un fichier inexistant (%v) — "+
				"refactor de renommage/suppression ? Retirer l'entrée morte (trou latent : un fichier "+
				"recréé à ce chemin échapperait au sentinel).", rel, err)
		}
	}
}

func isATTACHFuncName(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "attach") &&
		(strings.Contains(lower, "global") || strings.Contains(lower, "shared") || strings.Contains(lower, "xuid"))
}

func isSocialDBVarName(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "social")
}

// socialReceiverLabel extrait un label "human-readable" du receiver d'un appel
// Exec/Query si ce receiver pointe (directement ou via selector chain) vers la
// connexion shared_social. Retourne "" sinon.
//
// Patterns reconnus (ADR 0021 Phase 2.4) :
//
//	socialDB              -> "socialDB"            (variable directe)
//	pdb.SharedSocial      -> "pdb.SharedSocial"    (selector simple)
//	r.pdb.SharedSocial    -> "r.pdb.SharedSocial"  (selector profond)
//	r.socialDB()          -> "r.socialDB()"        (call method, via expr.Fun)
func socialReceiverLabel(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if isSocialDBVarName(e.Name) {
			return e.Name
		}
	case *ast.SelectorExpr:
		if isSocialDBVarName(e.Sel.Name) {
			// Reconstruit le chemin "x.y.SharedSocial" en remontant l'arbre.
			return selectorChainString(e)
		}
	case *ast.CallExpr:
		// Cas "r.socialDB()" — le receiver est un call dont la fonction est un selector.
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if isSocialDBVarName(sel.Sel.Name) {
				return selectorChainString(sel) + "()"
			}
		}
	}
	return ""
}

// selectorChainString reconstruit "a.b.c" depuis un *ast.SelectorExpr profond.
func selectorChainString(sel *ast.SelectorExpr) string {
	switch x := sel.X.(type) {
	case *ast.Ident:
		return x.Name + "." + sel.Sel.Name
	case *ast.SelectorExpr:
		return selectorChainString(x) + "." + sel.Sel.Name
	}
	return sel.Sel.Name
}

func containsATTACHKeyword(litValue string) bool {
	// litValue est inclus avec ses délimiteurs (" ou `).
	// Phase 2.4 ADR 0021 : étendu à ATTACH + DETACH + CREATE ... ATTACHED.
	// Tout DDL qui modifie l'arbre des bases attachées est interdit sur la conn social.
	upper := strings.ToUpper(litValue)
	b := []byte(upper)
	return bytes.Contains(b, []byte("ATTACH ")) ||
		bytes.Contains(b, []byte("DETACH ")) ||
		bytes.Contains(b, []byte("ATTACHED"))
}

// sharedSocialFilesWhitelist liste les fichiers Go autorisés à mentionner la
// chaîne "shared_social" dans un littéral string (typiquement un chemin de DB,
// un nom de target migration, ou une référence d'audit/log). Tout fichier qui
// référence "shared_social" hors de cette whitelist doit justifier explicitement
// son besoin via un commentaire et une entrée ajoutée ci-dessous.
//
// Ordre alphabétique. Chemins relatifs à apps/go-api/ (avec / unix-style).
var sharedSocialFilesWhitelist = map[string]string{
	"cmd/analyze_media_tz/main.go":                                                      "outil one-shot diag timezone media",
	"cmd/backfill-media-hls/main.go":                                                    "outil one-shot backfill HLS média : UPDATE media_files (commentaires + flag --db référencent le chemin shared_social.duckdb, serveur arrêté)",
	"internal/ops/media_hls_sweep.go":                                                   "balayage « assure le HLS » partagé scan/CLI (fix HEVC 2026-06-11) : candidats hls_path NULL + UPDATE via le handle injecté par le caller (MediaIndexService in-process ou CLI serveur arrêté) — la mention shared_social est le commentaire du champ DBPath",
	"cmd/cleanup_media_index/main.go":                                                   "outil one-shot cleanup index media",
	"cmd/diag_match_id_tables/main.go":                                                  "outil one-shot diag match_id",
	"cmd/migrate-media-paths/main.go":                                                   "outil one-shot migration paths media",
	"cmd/migrate-to-shared-social/main.go":                                              "outil one-shot CLI de migration legacy",
	"cmd/prestige-seed/main.go":                                                         "outil one-shot seed prestige",
	"cmd/purge_player_media/main.go":                                                    "outil one-shot purge media joueur",
	"cmd/rebuild_shared_social/main.go":                                                 "outil one-shot reconstruction (ADR 0021)",
	"cmd/regen-thumbnails/main.go":                                                      "outil one-shot regen thumbnails",
	"cmd/reindex-media-thumbs/main.go":                                                  "outil one-shot reindex thumbnails",
	"cmd/restore/main.go":                                                               "outil one-shot restore backup",
	"cmd/snapshot_shared_social/main.go":                                                "outil one-shot diag counts",
	"cmd/wal_forensic_compare/main.go":                                                  "outil one-shot forensique WAL (ADR 0021 Gap 2)",
	"cmd/duckdb_7659_repro/main.go":                                                     "outil one-shot repro bug DuckDB upstream (ADR 0021 Bonus 13)",
	"cmd/server/main.go":                                                                "boot serveur : pool init + CHECKPOINT scheduler",
	"internal/api/wire/registry_media.go":                                               "factory MediaService + acquéreur leased writer (wire, ex-internal/api, K3)",
	"internal/api/wire/registry_notifications.go":                                       "factory NotificationsService (shared_social path) (wire, ex-internal/api, K3)",
	"internal/api/wire/notifications_title_ready.go":                                    "notifier MT-19 title_ready : la mention 'shared_social' est un commentaire doc (per-title via PathResolver.SharedSocialDBPath) ; l'émission passe par l'Emitter (path Persister), pas d'ATTACH/INSERT direct (wire, ex-internal/api, K3)",
	"internal/api/wire/post_sync_progression.go":                                        "post-sync prestige/records (path Persister) (wire, ex-internal/api, K3)",
	"internal/api/wire/prestige_setup.go":                                               "init prestige (path Persister) (wire, ex-internal/api, K3)",
	"internal/api/wire/prestige_lazy_service.go":                                        "lazy init prestige (wire, ex-internal/api, K3)",
	"internal/api/server_apiv1.go":                                                      "montage routes /api/v1 : commentaires shared_social (match favoris, init coach/prestige) — ex-server.go, split K2a",
	"internal/games/halo_infinite/migrations/steps_shared_social.go":                    "migrations shared_social CONSOMMATRICES title-owned (Phase 1.5 b19, voie B)",
	"internal/migration/registry.go":                                                    "framework migration (target TargetSharedSocial)",
	"internal/migration/migration_test.go":                                              "tests framework migration",
	"internal/migration/steps_player.go":                                                "migrations player référencent shared_social path pour orchestration",
	"internal/migration/steps_player_notifications.go":                                  "migrations notifications player",
	"internal/migration/steps_player_progression.go":                                    "migrations progression player",
	"internal/migration/steps_shared.go":                                                "migrations shared (référence cross-target)",
	"internal/migration/steps_shared_social.go":                                         "migrations shared_social principales",
	"internal/migration/steps_shared_social_prestige.go":                                "migrations shared_social prestige",
	"internal/migration/steps_shared_social_purge_data_health.go":                       "migrations purge",
	"internal/migration/steps_shared_social_drop_pn_unread_art_index.go":                "migration drop idx_pn_xuid_unread (régression ART read_at réarmée par le rebuild purge, fix 2026-06-19)",
	"internal/migration/steps_shared_social_favorites_append_only.go":                   "migration match_favorites append-only (élimine le DELETE, surface ART shared_social)",
	"internal/migration/steps_shared_social_likes_append_only.go":                       "migration media_likes append-only (élimine DELETE + ON CONFLICT, surface ART shared_social)",
	"internal/migration/steps_shared_social_squad_member_append_only.go":                "migration squad_member append-only (élimine DELETE + ON CONFLICT, surface ART shared_social)",
	"internal/migration/steps_shared_social_squad_member_gamertag.go":                   "migration squad_member_history : ajoute la colonne gamertag (snapshot d'affichage du roster) + reconstruit la vue _latest",
	"internal/migration/steps_shared_social_notif_prefs_append_only.go":                 "migration notification_preferences append-only (élimine ON CONFLICT DO UPDATE, surface ART shared_social)",
	"internal/migration/steps_shared_social_media_assoc_append_only.go":                 "migration media_match_associations append-only (élimine DELETE manual-replace + reindex, surface ART shared_social)",
	"internal/migration/steps_shared_social_media_files_drop_filepath_unique.go":        "migration media_files rebuild sans contrainte UNIQUE sur file_path — élimine surface ART (UPDATE file_path conversion/HLS/reconcile, blast MAX shared_social)",
	"internal/migration/steps_shared_social_notifications_append_only.go":               "migration player_notifications append-only (élimine DELETE×2 + UPDATE read_at, surface ART shared_social)",
	"internal/migration/steps_shared_social_squad_challenge_participant_append_only.go": "migration squad_challenge_participant append-only (élimine ON CONFLICT + UPDATE, surface ART shared_social)",
	"internal/migration/steps_shared_social_user_prestige_append_only.go":               "migration user_prestige append-only (élimine ON CONFLICT DO UPDATE accumulation/overwrite, surface ART shared_social)",
	"internal/migration/steps_shared_social_records_append_only.go":                     "migrations records append-only",
	"internal/migration/steps_shared_social_records_previous_cols.go":                   "migration previous_* sur player_records_history (fix 2026-05-30)",
	"internal/migration/steps_shared_social_records_window.go":                          "migrations records window",
	"internal/migration/steps_shared_social_squad_xuid.go":                              "migration re-key squad_member par xuid (Phase C escouade)",
	"internal/migration/steps_shared_social_align_media_files_schema.go":                "ADR 0021 Bonus 12 — align media_files legacy schema",
	"internal/platform/duckdb/art_probe.go":                                             "ART probe sur shared_social",
	"internal/platform/dblease/writer.go":                                               "LeasedWriter (CommitWithCheckpoint, ADR 0021 Phase 3.2 bis)",
	"internal/notifications/service.go":                                                 "service notifications",
	"internal/notify/notifiers.go":                                                      "notifications outbound",
	"internal/ops/backup_service.go":                                                    "backup/restore ops",
	"internal/ops/media.go":                                                             "IndexMedia + CHECKPOINT (Phase 3.2)",
	"internal/ops/media_associate.go":                                                   "association média-match sans ATTACH (ADR 0021)",
	"internal/ops/media_store.go":                                                       "ensureMediaTables : crée le substrat append-only media_match_associations_history + vue (réconciliation migration shared_social)",
	"internal/ops/media_hls.go":                                                         "transcoding HLS média : commentaire sur DBPath = shared_social.duckdb (cible UPDATE media_files)",
	"internal/ops/seed_demo.go":                                                         "seed démo : construit le chemin + copie/recrée data/demo/warehouse/shared_social.duckdb (fichier + path, pas d'ATTACH sur RW)",
	"internal/ops/seed_demo_media.go":                                                   "seed démo média : recrée le shared_social démo et lit la SOURCE prod pour les media_files (pas d'ATTACH sur RW)",
	"internal/config/player_resolver.go":                                                "résolution du chemin data/demo/warehouse/shared_social.duckdb en mode démo (construction de path, pas d'accès DB)",
	"internal/persist/shared_social_persister.go":                                       "SocialPersister canonique (CHECKPOINT garanti)",
	"internal/persist/shared_social_rows.go":                                            "types batch SocialPersister",
	"internal/platform/dblease/kind.go":                                                 "type Kind=SharedSocial pour lease tracking",
	"internal/platform/dblease/metrics.go":                                              "expvar par kind",
	"internal/platform/duckdb/db.go":                                                    "API DB générique (commentaire sur policy)",
	"internal/platform/duckdb/db_query.go":                                              "API DB générique — commentaire policy sur l'invalidation process-level (ex-db.go, split K3f)",
	"internal/platform/duckdb/pool.go":                                                  "pool : SharedSocial config + ouverture",
	"internal/platform/duckdb/pool_shared_social_recovery.go":                           "recovery WAL (ADR 0021 Phase 2)",
	"internal/platform/duckdb/pool_writers.go":                                          "acquéreurs LeasedWriter par kind",
	"internal/platform/duckdb/social_repo.go":                                           "lectures social shared",
	"internal/platform/duckdb/social_persister_iface.go":                                "interface SocialPersister consommée",
	"internal/platform/duckdb/notifications_repo.go":                                    "notifications repo (lit shared_social)",
	"internal/platform/duckdb/records_repo.go":                                          "records repo (lit shared_social)",
	"internal/platform/duckdb/prestige/prestige_social_repo.go":                         "prestige social repo",
	"internal/platform/duckdb/progression_diag_repo.go":                                 "progression diag repo",
	"internal/platform/duckdb/home_repo_matches.go":                                     "home recent media (lit shared_social via SharedSocial)",
	"internal/platform/duckdb/match_view_repo_extras.go":                                "match view media (lit shared_social)",
	"internal/platform/duckdb/queries_match.go":                                         "queries shared utilities",
	"internal/platform/duckdb/queries_home_citations.go":                                "queries home utilities",
	"internal/platform/duckdb/media_repo.go":                                            "MediaRepo + socialDB() helper",
	"internal/platform/duckdb/media_repo_writes.go":                                     "MediaRepo writes (Phase 3.2 CHECKPOINT)",
	"internal/platform/duckdb/media_repo_filters.go":                                    "MediaRepo filters",
	"internal/platform/duckdb/media_repo_q37_pipeline.go":                               "pipeline Q37 (lit shared_social)",
	"internal/platform/duckdb/media_repo_registry.go":                                   "registry helpers",
	"internal/platform/duckdb/media_repo_translations.go":                               "translations helpers",
	"internal/progression/coach/types.go":                                               "coach types touchent shared_social",
	"internal/progression/records/repository.go":                                        "records repo wiring",
	"internal/progression/records/types.go":                                             "records types",
	"internal/prestige/repository.go":                                                   "prestige repo wiring",
	"internal/prestige/types.go":                                                        "prestige types",
	"internal/domain/title/registry.go":                                                 "PathResolver SharedSocialPath",
	"internal/domain/media.go":                                                          "types domain Media",
	"internal/domain/match_view.go":                                                     "types domain MatchView",
	"internal/domain/progression_diag.go":                                               "types domain diag",
	"internal/service/media_service.go":                                                 "MediaService (orchestration)",
	"internal/service/media_index_service.go":                                           "MediaIndexService (orchestration)",
	"internal/service/match_view_service.go":                                            "MatchViewService (lit shared_social)",
	"internal/service/match_view_data_loaders.go":                                       "data loaders match_view",
	"internal/service/social_service.go":                                                "SocialService",
	"internal/port/repository_data.go":                                                  "interfaces port DBExecutor/Provider",
	"pkg/duckdbbackup/target.go":                                                        "backup targets enum",
	"pkg/duckdbbackup/exporter.go":                                                      "backup exporter",
	// Ajoutés 2026-06-03
	"internal/api/handlers/media.go":                    "commentaire doc : liste auteurs depuis shared_social.media_files",
	"internal/platform/duckdb/player_record_repo.go":    "records perso (Load/UpsertPlayerRecord, path Persister shared_social) — ex-api/post_sync_deltas_records.go, déplacé K1a 2026-07-05",
	"internal/migration/order.go":                       "ordre de migration (cible TargetSharedSocial dans la liste)",
	"internal/persist/shared_social_persister_batch.go": "batch SocialPersister (accès shared_social via persister canonique)",
	"internal/platform/duckdb/queries_match_detail.go":  "Q24 matchs media shared_social (commentaire)",
	// Ajoutés 2026-06-30
	"cmd/levelup/cmd_data.go":            "CLI unifiée levelup data : commentaire doc (associations média du shared_social du joueur, best-effort)",
	"cmd/levelup/main.go":                "CLI unifiée levelup : commentaire doc sur le schéma shared_social (media_files/associations)",
	"internal/ops/seed_demo_media_h5.go": "seed démo média H5 : lit le shared_social SOURCE (prod) en read-only pour ancrer les clips démo (variante H5 de seed_demo_media.go) — pas d'ATTACH/INSERT sur RW",
}

// listForbiddenSharedSocialMentions retourne la liste des fichiers Go non-test
// qui mentionnent "shared_social" en littéral ET ne sont pas dans la whitelist.
// Utilisé par TestNoUnauthorizedSharedSocialMention.
func listForbiddenSharedSocialMentions(root string) ([]string, error) {
	var violations []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "tmp" || name == "testdata" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if !bytes.Contains(content, []byte("shared_social")) {
			return nil
		}
		if _, ok := sharedSocialFilesWhitelist[rel]; ok {
			return nil
		}
		violations = append(violations, rel)
		return nil
	})
	return violations, err
}

func formatViolation(pos token.Position, what string) string {
	return "  " + pos.Filename + ":" + intToStr(pos.Line) + " — " + what
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// findGoAPIRoot remonte depuis le test courant jusqu'à apps/go-api/.
func findGoAPIRoot(t *testing.T) string {
	t.Helper()
	// Le test tourne depuis internal/platform/duckdb/ → remonter 3 niveaux.
	wd, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	// On veut le répertoire qui contient internal/ et cmd/.
	root := wd
	for i := 0; i < 5; i++ {
		if _, err := filepath.Glob(filepath.Join(root, "internal")); err == nil {
			if matches, _ := filepath.Glob(filepath.Join(root, "internal")); len(matches) > 0 {
				if matches2, _ := filepath.Glob(filepath.Join(root, "cmd")); len(matches2) > 0 {
					return root
				}
			}
		}
		root = filepath.Dir(root)
	}
	t.Fatalf("apps/go-api/ root introuvable depuis %s", wd)
	return ""
}
