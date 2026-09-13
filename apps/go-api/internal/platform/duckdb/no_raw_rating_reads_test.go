package duckdb

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestNoRawAppendOnlyReads — garde-rail Lot B (ADR 0026 / règle ART n°2).
//
// Les tables append-only (match_skill_rank, match_csrs, player_csr_snapshots,
// pve_match_stats) portent N versions par clé (id + written_at). Une lecture
// APPLICATIVE de la table BRUTE sert des lignes périmées de façon non
// déterministe. Toute lecture passe donc par la vue `<table>_latest`.
//
// Périmètre : couches de LECTURE uniquement (platform/duckdb, api, service,
// analysis). Les writers (internal/sync, internal/persist), les migrations
// (internal/games/*/migrations, internal/migration, schema.go) et les CLI de
// diagnostic (cmd/) lisent légitimement le brut → hors scan.
//
// Ajout d'une lecture brute = migrer vers `_latest`, ou (rare) l'allowlister
// ci-dessous avec une justification datée. L'allowlist est décroissante.
func TestNoRawAppendOnlyReads(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// thisFile = <goapi>/internal/platform/duckdb/no_raw_rating_reads_test.go
	goAPIRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	scanDirs := []string{
		filepath.Join(goAPIRoot, "internal", "platform", "duckdb"),
		filepath.Join(goAPIRoot, "internal", "api"),
		filepath.Join(goAPIRoot, "internal", "service"),
		filepath.Join(goAPIRoot, "internal", "analysis"),
	}

	// `FROM|JOIN <table>` : le groupe 2 capture un éventuel suffixe `_latest`.
	// Une occurrence dont le groupe 2 est vide = lecture BRUTE.
	//
	// `match_lives` / `match_death_context` AJOUTÉES LE 2026-09-07 (lot 7C.8) : append-only,
	// vues `_latest` PAR PASSE — une lecture brute y servirait un MÉLANGE de décodages, pas
	// seulement une ligne périmée. `match_kill_events` / `kill_positions` / `match_bomb_stats`
	// AJOUTÉES LE 2026-09-07 (clôture Q8) : même famille append-only, découverte consignée au
	// §7 du plan Tactique — tous leurs lecteurs actuels (platform/duckdb, api, analysis)
	// passaient déjà par `_latest`, l'ajout n'a fait rougir aucun lecteur existant.
	// `_latest_by_type` AJOUTÉ AU SUFFIXE LE 2026-09-13 (lot finitions LUSR, C.3 bis) :
	// match_skill_rank a DEUX vues de lecture légitimes, qui répondent à deux questions.
	// `_latest` arbitre par match_id (priorité CSR > LUSR) — « quel rang afficher pour ce
	// match ? ». `_latest_by_type` retient la dernière ligne par (match_id, rating_type)
	// sans arbitrer un type contre un autre — « quel checkpoint LUSR, et quel checkpoint
	// CSR ? », ce que veut le graphe d'évolution de la page Carrière. Les deux masquent
	// les lignes supersédées d'une table append-only ; ni l'une ni l'autre n'est une
	// lecture brute. Sans cet ajout le motif ne matchait PAS `_latest_by_type` du tout
	// (la frontière de mot échouait devant `_`) : la lecture passait, mais par accident
	// plutôt que par décision — et un renommage futur l'aurait rendue invisible au garde.
	rawRe := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+(match_skill_rank|match_csrs|player_csr_snapshots|pve_match_stats|match_lives|match_death_context|match_kill_events|kill_positions|match_bomb_stats|match_flag_grabs_net)(_latest(?:_by_type)?)?\b`)

	// Allowlist datée (2026-07-02) — lectures brutes VOLONTAIRES et documentées.
	// 2026-09-13 (lot finitions LUSR, C.3 bis) : `queries_career.go` RETIRÉ de l'allowlist.
	// Q8LUSRHistoryPlayer lit désormais match_skill_rank_latest_by_type — le motif « le
	// graphe veut TOUS les checkpoints » rendait le graphe d'évolution non réparable par
	// un replay append-only (résidu h5_arena du 2026-06-26 encore tracé après la
	// réparation d'août). L'allowlist est décroissante : 5 → 4 entrées.
	allow := map[string]string{
		"queries_home_citations.go":    "Q26g : filtre H5 placeholder CSR=0 appliqué AVANT le choix de ligne (la vue a déjà tranché CSR>LUSR) + tie-break written_at déterministe. (NB 2026-07-10 : l'ex-mention Q26f est obsolète — la classification effective_type vit en Go sur match_registry.is_ranked et lit déjà _latest, cf. leaderboard_repo/home_repo_skill_peak.)",
		"queries_career_encounters.go": "Q24LUSRHistory (pipeline LUSR, échelle mu) : raw VOLONTAIRE — _latest (CSR>LUSR) injecterait des valeurs CSR (échelle ~1500) sur les matchs ranked à double ligne = rupture d'échelle. DÉCISION B7 (2026-07-10) : statu quo, ne pas migrer.",
		"queries_squad.go":             "MAX(expected_win_prob) WHERE IS NOT NULL : stale-safe (colonne écrite seulement sur LUSR) ; _latest perdrait le winProb sur ranked",
		"career_repo_csr_seasons.go":   "SELECT DISTINCT season_id : stale-safe (season_id invariant entre versions)",
	}

	var offenders []string
	for _, dir := range scanDirs {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil
			}
			hasRaw := false
			for _, m := range rawRe.FindAllSubmatch(src, -1) {
				if len(m[2]) == 0 { // pas de suffixe `_latest` → lecture brute
					hasRaw = true
					break
				}
			}
			if !hasRaw {
				return nil
			}
			if _, allowed := allow[filepath.Base(path)]; allowed {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			offenders = append(offenders, rel)
			return nil
		})
	}

	if len(offenders) > 0 {
		t.Errorf("lecture(s) brute(s) d'une table append-only hors vue _latest (ADR 0026) — "+
			"migrer vers `<table>_latest` ou allowlister avec justification datée :\n  %s",
			strings.Join(offenders, "\n  "))
	}
}
