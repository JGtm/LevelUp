// Package archlint — no_french_label_literal_test.go : LE GARDE-RAIL FINAL du plan
// « libellés en dur » (.ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md, §4 « Garde-rail FINAL »).
//
// # CE QUI EST COMPTÉ
//
// Tout littéral de chaîne Go interprété ou brut (backticks) contenant un caractère accentué
// (`[éèêàùçÉÈÊÀÙÇ]`) dans les fichiers NON-TEST de `internal/{service,analysis,
// api/handlers,notify,games}` (récursif). Sont EXCLUS de la mesure — pas du périmètre du
// dépôt, seulement de CE ratchet :
//   - les fichiers `_test.go` ;
//   - tout fichier sous un répertoire `migrations/` (DDL + notes de migration — HORS
//     PÉRIMÈTRE du plan entier, CLAUDE.md « Multi-titre » / plan libellés §2.H : ce n'est
//     jamais lu à l'écran) ;
//   - les littéraux servis EN ARGUMENT DIRECT d'un appel `slog.<Xxx>(...)` (logs internes,
//     hors périmètre du plan, §2.H) ;
//   - les littéraux servis EN ARGUMENT DIRECT d'un appel `fmt.Errorf(...)` (erreurs — lot
//     L7 du plan, pas encore statué : D6 recommande `code` seul + table FR/EN côté web).
//
// Un littéral qui transite par une variable avant d'atteindre `slog`/`fmt.Errorf` N'EST
// PAS exclu (comme les autres ratchets du dossier, cette mesure est volontairement simple :
// elle attrape le nouveau code, elle n'audite pas rétroactivement chaque détour).
//
// # RATCHET PAR FICHIER, PAS UN TOTAL
//
// Contrairement à `filmdec_package_vars_test.go` (un total global), ce garde-rail liste
// chaque fichier PORTEUR avec son compte du jour (mesuré par ce test lui-même le
// 2026-09-07, cf. TOTAL en pied de fichier) : décroissant, jamais remonté. Le test échoue
// si :
//   - un fichier de l'allowlist dépasse SON compte (régression sur un site déjà connu) ;
//   - un fichier HORS allowlist porte au moins un littéral accentué (nouveau site, ou
//     fichier qu'une baisse à zéro aurait dû retirer de la liste).
//
// Baisser un compte (ou retirer une entrée tombée à zéro) est TOUJOURS accepté — c'est
// l'objectif des lots L2 à L8 du plan libellés. Le faire monter, ou en ajouter une
// nouvelle, ne l'est jamais : passer par `TitleSemanticAdapter` + TOML
// (`config/titles/{slug}/mappings/`), ou pour les erreurs un `code` machine (D6) —
// jamais un mot FR/EN de plus en dur.
//
// # PÉRIMÈTRE MESURÉ, PAS ENCORE TRIÉ FAMILLE PAR FAMILLE
//
// Le plan libellés (§2) n'avait audité en détail que les familles A (issue de match,
// clos par Q4), B/C/D (modes/armes/rangs) et un ÉCHANTILLON de F (erreurs handlers,
// 9 fichiers cités « … » = liste non exhaustive). La mesure PAR AST de ce garde-rail (Q4,
// 2026-09-07) est la première PASSE COMPLÈTE sur le périmètre déclaré : elle révèle 132
// fichiers (538 littéraux), très au-delà des ~15 fichiers cités dans l'inventaire §2 —
// l'essentiel étant des messages d'erreur `api/handlers/*` (famille F) non encore
// individuellement cités. Conforme à la doctrine RE-VÉRIFIER du plan : la carte du 07/09
// datait déjà. Reprise consignée : .ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md §9.
//
// Modèle : les autres `no_*_test.go` du dossier (ratchet nommé/daté) et `no_mojibake_test.go`
// (marche à suivre, allowlist vide visée).
package archlint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// accentedLiteralRE détecte un caractère accentué français dans un littéral de chaîne.
// Volontairement étroit (pas de \p{L} générique) : cible le français en dur, pas tout
// unicode — un UUID, une URL ou un mot anglais légitime ne le contiennent jamais.
var accentedLiteralRE = regexp.MustCompile(`[éèêàùçÉÈÊÀÙÇ]`)

// frenchLabelAllowlist : compte du jour (2026-09-07, mesuré par ce test), par fichier
// relatif à `internal/`. TOTAL au jour de la mesure : 132 fichiers, 538 littéraux (après retrait Q4 de 3 fichiers tombés à zéro).
// Mise à jour 2026-09-07 (lot M5, L5) : compare_service.go 3 → 2 (csrUnrankedLabel migré
// vers la clé canonique "unranked", D5) — 132 fichiers, 537 littéraux.
// Mise à jour 2026-09-08 (lot M5, L3) : rankedplaylists.go 15 → 0, retiré de la liste
// (NameEN/NameFR lisent désormais ranked_playlists_labels.toml) — 131 fichiers, 522
// littéraux.
// Familles connues (cf. plan libellés §2) annotées ; le reste (essentiel : messages
// d'erreur des handlers, famille F/D6) attend la décision utilisateur D6 avant tri fin.
var frenchLabelAllowlist = map[string]int{
	// L6 — narratif / prestige (D7 : ce sont des PHRASES/contenu, ADR 0028 — hors plan
	// libellés sauf mojibake, mais comptées ici tant qu'elles restent en dur).
	"analysis/filmdec/equipment_creation_width.go": 1,
	"analysis/match_impact.go":                     3,
	"analysis/patterns/behavioral.go":              2,
	"analysis/patterns/behavioral_engagement.go":   3,
	"analysis/prestigetuning/analyze.go":           6,
	"analysis/prestigetuning/render.go":            16,
	"analysis/replay/coverage_bridge.go":           5,
	"analysis/replay/neutral_deaths.go":            2,
	"analysis/skill_v2/display_smoothing.go":       1,
	"notify/coach.go":                              5,
	"service/synthesis_service_builders.go":        1,
	"service/synthesis_service_legacy.go":          5,

	// L3 — modes / playlists / catégories (cible : assets.toml). rankedplaylists.go
	// retiré le 2026-09-08 (lot M5 L3, branche feat/libelles-modes-playlists) :
	// NameEN/NameFR sont devenues des méthodes lisant ranked_playlists_labels.toml
	// (embarqué, mêmes loader/validation que assets.toml — cf. commentaire du TOML
	// pour la justification de l'emplacement et sa condition de reprise) : 15 → 0.
	// match_history_service.go (expTypePVPRanked/expTypePVPUnranked) NON traité :
	// la VALUE canonique FR est un CONTRAT testé avec la cascade de filtres du web
	// (GH5-2, ~80 fichiers `apps/web/src` matchent dessus, ex.
	// features/_shared/experienceCascade.ts, commentaire « NE PAS traduire ces
	// chaînes ici ») — le LABEL, lui, est déjà localisé FR/EN (expTypeLabelEN).
	// Migrer la VALUE vers une clé neutre exige un lot dédié coordonné back+front
	// (cf. .ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md §11, découverte M5 L3).
	"analysis/playlist_label.go":       2,
	"service/match_history_service.go": 2,

	// L4 — armes (cible : mappings/fields.toml ou assets.toml).
	"games/weapons/labels.go":   12,
	"games/weapons/registry.go": 22,

	// L5 — rangs / CSR. mappings/ranks.go (RankCatalog) est le rang de CARRIÈRE,
	// PAS le tier CSR (découverte lot M5, 2026-09-07 : la carte du plan datait —
	// doctrine RE-VÉRIFIER). csrUnrankedLabel = "Non classé" (D5) migré vers la
	// clé canonique "unranked" (lib/skillTiers.ts::localizeTierLabel côté web) :
	// 3 → 2 littéraux. Les 2 restants sont des messages `logBestEffortErr` (family
	// F, hors périmètre L5 — le wrapper n'est pas reconnu par l'exclusion slog.*/
	// fmt.Errorf de ce garde-rail).
	"service/compare_service.go": 2,

	// L8 — notifications Discord (D8 tranché : langue du compte propriétaire).
	"notify/discord.go": 34,

	// L7 — messages d'erreur / descriptions handlers (D6 recommandé, pas encore statué :
	// `code` seul côté Go, table FR/EN côté web ; descriptions OpenAPI gardées FR). Liste
	// complète mesurée le 2026-09-07 (dépasse largement l'échantillon du plan §2.F).
	"api/handlers/admin.go":                       9,
	"api/handlers/admin_actions.go":               4,
	"api/handlers/admin_actions_catalog_drain.go": 4,
	"api/handlers/admin_actions_convergence.go":   2,
	"api/handlers/admin_actions_initial_sync.go":  3,
	"api/handlers/admin_actions_replay_build.go":  8,
	"api/handlers/admin_appearance_diag.go":       1,
	"api/handlers/admin_auto_sync.go":             1,
	"api/handlers/admin_data_quality.go":          17,
	"api/handlers/admin_invariants.go":            2,
	"api/handlers/admin_logs.go":                  2,
	"api/handlers/admin_lusr_gaps.go":             4,
	"api/handlers/admin_monitoring.go":            17,
	"api/handlers/admin_title_diagnostic.go":      1,
	"api/handlers/admin_titles.go":                3,
	"api/handlers/admin_token_health.go":          2,
	"api/handlers/admin_weapon_coverage.go":       2,
	"api/handlers/assets.go":                      2,
	"api/handlers/assets_metadata.go":             5,
	"api/handlers/auth.go":                        7,
	"api/handlers/auth_xbox_oauth.go":             8,
	"api/handlers/backfill.go":                    2,
	"api/handlers/build_worker.go":                8,
	"api/handlers/build_worker_artifact.go":       4,
	"api/handlers/campaign.go":                    4,
	"api/handlers/capabilities.go":                2,
	"api/handlers/career.go":                      3,
	"api/handlers/citations.go":                   1,
	"api/handlers/coach_proposals.go":             3,
	"api/handlers/commendation_totals.go":         1,
	"api/handlers/diag_prestige_telemetry.go":     1,
	"api/handlers/engagement.go":                  1,
	"api/handlers/explorer.go":                    1,
	"api/handlers/feature_matrix.go":              1,
	"api/handlers/field_mappings.go":              1,
	"api/handlers/filters.go":                     2,
	"api/handlers/groups.go":                      18,
	"api/handlers/health_home.go":                 1,
	"api/handlers/help.go":                        1,
	"api/handlers/helpers.go":                     2,
	"api/handlers/home.go":                        3,
	"api/handlers/leaderboard.go":                 2,
	"api/handlers/match_events.go":                1,
	"api/handlers/match_exclusion.go":             2,
	"api/handlers/match_history.go":               3,
	"api/handlers/match_view.go":                  3,
	"api/handlers/medals.go":                      1,
	"api/handlers/media.go":                       9,
	"api/handlers/media_delete.go":                1,
	"api/handlers/media_paths.go":                 1,
	"api/handlers/media_upload.go":                3,
	"api/handlers/notifications.go":               3,
	"api/handlers/openspartan_import.go":          3,
	"api/handlers/patterns.go":                    1,
	"api/handlers/player_profile.go":              2,
	"api/handlers/presence.go":                    1,
	"api/handlers/prestige.go":                    24,
	"api/handlers/relations.go":                   3,
	"api/handlers/replay.go":                      3,
	"api/handlers/season_pass.go":                 1,
	"api/handlers/session_context.go":             2,
	"api/handlers/session_page.go":                1,
	"api/handlers/sessions.go":                    1,
	"api/handlers/settings.go":                    21,
	"api/handlers/settings_backup.go":             1,
	"api/handlers/settings_replay_sound.go":       1,
	"api/handlers/setup.go":                       12,
	"api/handlers/squad_v2.go":                    1,
	"api/handlers/stats.go":                       1,
	"api/handlers/sync_handler.go":                7,
	"api/handlers/sync_handler_align.go":          1,
	"api/handlers/synthesis.go":                   1,
	"api/handlers/teammates.go":                   1,
	"api/handlers/timeseries.go":                  1,
	"api/handlers/title_sync.go":                  1,
	"api/handlers/user_auth.go":                   13,
	"api/handlers/watcher_handler.go":             3,

	// Non triés — code Halo 5 / mappings / services divers, messages d'erreur internes
	// pour l'essentiel (même statut que la famille F ci-dessus : attend D6).
	"games/halo_5/adapter_data.go":                      1,
	"games/halo_5/adapter_data_loaders.go":              2,
	"games/halo_5/career_local.go":                      1,
	"games/halo_5/commendation_totals.go":               1,
	"games/halo_5/livesync/backfill.go":                 3,
	"games/halo_5/livesync/events_backfill_surfaces.go": 2,
	"games/halo_5/livesync/kill_kind_backfill.go":       3,
	"games/halo_5/livesync/resolver.go":                 1,
	"games/halo_5/livesync/runner.go":                   5,
	"games/halo_5/livesync/wire.go":                     1,
	"games/halo_infinite/citations_custom.go":           2,
	"games/halo_infinite/events.go":                     2,
	"games/halo_infinite/medal_category_gen.go":         9,
	"games/mappings/loader.go":                          1,
	"games/mappings/loader_objective_roles.go":          1,
	"games/mappings/loader_regulation.go":               2,
	"games/mappings/loader_replay_labels.go":            3,
	"games/mappings/loader_replay_labels_equipment.go":  1,
	"games/mappings/loader_replay_labels_flagzone.go":   2,
	"games/mappings/registry.go":                        1,
	"service/backfill_orchestrator.go":                  9,
	"service/career_live_target.go":                     2,
	"service/filters_service.go":                        3,
	"service/match_view_service.go":                     3,
	"service/media_index_service.go":                    4,
	"service/media_service_delete.go":                   4,
	"service/media_service_upload.go":                   7,
	"service/profile_service.go":                        1,
	"service/release_notes_service.go":                  2,
	"service/season_pass_service.go":                    1,
	"service/seasons_catalog.go":                        4,
	"service/session_compare_service.go":                2,
	"service/session_compare_table_helpers.go":          1,
	"service/session_page_service.go":                   7,
	"service/skill_v2_service.go":                       1,
	"service/squad_service_v2_intersect.go":             1,
	"service/squadagg/consts.go":                        2,
	// service/stats_service.go, service/synthesis_service.go et
	// service/teammates/teammates_service.go sont sortis de l'allowlist le 2026-09-07
	// (Q4) : leur seul littéral accentué était le repli FR de l'issue de match, supprimé
	// par la clé canonique (D5).
	"service/title_diagnostic_service.go": 2,
	"service/xbox_auth_service.go":        1,
}

func TestNoNewFrenchLabelLiteral(t *testing.T) {
	internalRoot := filepath.Join(apiRootDepuisIci(t), "internal")

	compte := map[string]int{}
	for _, sub := range []string{"service", "analysis", "api/handlers", "notify", "games"} {
		dir := filepath.Join(internalRoot, filepath.FromSlash(sub))
		scanFrenchLabelLiterals(t, internalRoot, dir, compte)
	}

	var regressions []string
	var nouveaux []string
	var baisses []string
	for rel, allowed := range frenchLabelAllowlist {
		got := compte[rel]
		switch {
		case got > allowed:
			regressions = append(regressions, fmt.Sprintf("%s : %d littéraux, alloué %d", rel, got, allowed))
		case got < allowed:
			baisses = append(baisses, fmt.Sprintf("%s : %d littéraux (alloué %d) — resserrer frenchLabelAllowlist", rel, got, allowed))
		}
	}
	for rel, got := range compte {
		if _, known := frenchLabelAllowlist[rel]; !known && got > 0 {
			nouveaux = append(nouveaux, fmt.Sprintf("%s : %d littéraux, hors allowlist", rel, got))
		}
	}
	sort.Strings(regressions)
	sort.Strings(nouveaux)
	sort.Strings(baisses)

	for _, b := range baisses {
		t.Logf("baisse (à figer) : %s", b)
	}
	if len(regressions) > 0 || len(nouveaux) > 0 {
		t.Errorf("littéraux FR en dur détectés hors des sites déjà comptés (D5 : la clé "+
			"canonique côté Go, jamais un mot FR/EN en dur — passer par TitleSemanticAdapter + "+
			"TOML config/titles/{slug}/mappings/) :\n  régressions :\n    %s\n  nouveaux fichiers :\n    %s",
			strings.Join(regressions, "\n    "), strings.Join(nouveaux, "\n    "))
	}
}

// scanFrenchLabelLiterals parcourt dir (récursif), compte par fichier relatif à
// internalRoot les littéraux de chaîne accentués hors slog.*/fmt.Errorf/migrations/, dans
// les fichiers non-test.
func scanFrenchLabelLiterals(t *testing.T, internalRoot, dir string, compte map[string]int) {
	t.Helper()
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("répertoire %s introuvable (a-t-il déménagé ?) : %v", dir, err)
	}
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(internalRoot, path)
		rel = filepath.ToSlash(rel)
		if strings.Contains(rel, "/migrations/") {
			return nil // DDL + notes de migration — HORS PÉRIMÈTRE, cf. en-tête du fichier.
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			t.Fatalf("analyse de %s : %v", path, perr)
		}
		n := countAccentedLiteralsExcludingLogsAndErrors(f)
		if n > 0 {
			compte[rel] += n
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
}

// countAccentedLiteralsExcludingLogsAndErrors compte les littéraux de chaîne accentués de
// f, en excluant ceux servis en ARGUMENT DIRECT d'un appel slog.<Xxx>(...) ou
// fmt.Errorf(...).
func countAccentedLiteralsExcludingLogsAndErrors(f *ast.File) int {
	excluded := map[*ast.BasicLit]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isSlogOrErrorfCall(call.Fun) {
			return true
		}
		for _, arg := range call.Args {
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				excluded[lit] = true
			}
		}
		return true
	})

	count := 0
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING || excluded[lit] {
			return true
		}
		value, ok := unquoteGoLiteral(lit.Value)
		if !ok {
			return true
		}
		if accentedLiteralRE.MatchString(value) {
			count++
		}
		return true
	})
	return count
}

// isSlogOrErrorfCall dit si fun désigne `slog.<Xxx>` (n'importe quelle fonction du paquet
// slog, y compris les variantes Context) ou `fmt.Errorf`.
func isSlogOrErrorfCall(fun ast.Expr) bool {
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if pkg.Name == "slog" {
		return true
	}
	return pkg.Name == "fmt" && sel.Sel.Name == "Errorf"
}

// unquoteGoLiteral retire les délimiteurs d'un littéral Go interprété (`"..."`) ou brut
// (“ `...` “), sans traiter les séquences d'échappement — seule la présence d'un
// caractère accentué compte ici, pas un déquotage exact. ok=false sur un littéral mal
// formé (ne devrait pas arriver sur du code qui compile).
func unquoteGoLiteral(raw string) (string, bool) {
	if len(raw) < 2 {
		return "", false
	}
	if raw[0] == '`' && raw[len(raw)-1] == '`' {
		return raw[1 : len(raw)-1], true
	}
	if raw[0] == '"' && raw[len(raw)-1] == '"' {
		return raw[1 : len(raw)-1], true
	}
	return "", false
}
