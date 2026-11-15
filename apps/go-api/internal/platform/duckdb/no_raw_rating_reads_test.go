package duckdb

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// rawAppendOnlyReadAllowlist : lectures brutes VOLONTAIRES et documentées, par chemin COMPLET du
// fichier (relatif à apps/go-api) : une exemption par nom de base laissait passer tout fichier
// homonyme d'un autre dossier. Chaque raison commence par la date de l'entrée, puis « (revu
// AAAA-MM-JJ) » (TestAllowlistLecturesBrutesDatee). Décroissante.
//
// 2026-09-13 (lot finitions LUSR, C.3 bis) : `queries_career.go` RETIRÉ de l'allowlist.
// Q8LUSRHistoryPlayer lit désormais match_skill_rank_latest_by_type — le motif « le
// graphe veut TOUS les checkpoints » rendait le graphe d'évolution non réparable par
// un replay append-only (résidu h5_arena du 2026-06-26 encore tracé après la
// réparation d'août). L'allowlist est décroissante : 5 → 4 entrées.
// 2026-09-26 (lot A, décision DA.10 du plan perf « lectures par périmètre ») : 4 → 5, une
// entrée pour le SEUL site de « locating read » de l'ADR 0036 (AnnuaireKillFeedLocaliserSQL).
// 2026-11-10 (plan architecture, 1.6) : `queries_squad.go` RETIRÉ, sa seule lecture brute (la probabilité
// de victoire d'Escouade) ayant disparu avec son affichage : 5 → 4 entrées.
var rawAppendOnlyReadAllowlist = map[string]string{
	"internal/platform/duckdb/sqlfrag/annuaire.go": "2026-10-20 (premier commit de l'entrée, squash v8.0.0 ; lot A daté 2026-09-26 par le texte d'origine ; revu 2026-11-12) : DA.10, ADR 0036 « locating read ». " +
		"AnnuaireKillFeedLocaliserSQL lit match_kill_events BRUTE pour localiser des match_id — lecture de localisation, " +
		"sur-ensemble de matchs, aucune valeur lue ; valeurs par `_latest` (AnnuaireKillFeedSQL sur ces seuls matchs).",
	"internal/platform/duckdb/queries_home_citations.go": "2026-07-02 (revu 2026-11-12) : Q26g, lecture brute de " +
		"match_skill_rank, filtre H5 placeholder CSR=0 appliqué AVANT le choix de ligne (la vue a déjà tranché CSR>LUSR) + " +
		"tie-break written_at/id déterministe (la requête porte le commentaire « allowlist B8 »).",
	"internal/platform/duckdb/queries_career_encounters.go": "2026-07-02 (revu 2026-11-12) : Q24LUSRHistory " +
		"(pipeline LUSR, échelle mu), raw VOLONTAIRE — _latest (CSR>LUSR) injecterait des valeurs CSR (échelle ~1500) sur " +
		"les matchs ranked à double ligne = rupture d'échelle. DÉCISION B7 (2026-07-10) : statu quo, ne pas migrer. " +
		"Non revérifié : la requête n'a ni filtre rating_type ni tie-break written_at (voir les découvertes du plan architecture).",
	"internal/platform/duckdb/career_repo_csr_seasons.go": "2026-07-02 (revu 2026-11-12) : qDistinctMatchCSRSeasons, " +
		"SELECT DISTINCT season_id sur match_csrs : stale-safe (season_id invariant entre versions).",
}

// TestAllowlistLecturesBrutesDatee : toute raison de l'allowlist commence par une date AAAA-MM-JJ
// (règle ART 3 : jamais d'allowlist sans justification datée) et les clés sont des chemins
// complets, pas des noms de fichier.
func TestAllowlistLecturesBrutesDatee(t *testing.T) {
	forme := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\b`)
	for rel, raison := range rawAppendOnlyReadAllowlist {
		if !forme.MatchString(raison) {
			t.Errorf("%s : la raison ne commence pas par une date AAAA-MM-JJ : %q", rel, raison)
		}
		if !strings.Contains(rel, "/") || !strings.HasSuffix(rel, ".go") {
			t.Errorf("%s : la clé doit être un chemin complet relatif à apps/go-api, pas un nom de fichier", rel)
		}
	}
}

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
// dans rawAppendOnlyReadAllowlist avec une justification datée. L'allowlist est décroissante.
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
	// `match_death_mates` AJOUTÉE LE 2026-10-14 (plan Tactique coéquipiers, L1) : append-only, vue
	// `_latest` PAR PASSE ; enrôlée à sa création, avant tout lecteur.
	// `match_life_placement` AJOUTÉE LE 2026-09-29 (plan Emprise vies, lot V2) : append-only,
	// vue `_latest` PAR PASSE, comme `match_lives` ; enrôlée à sa création, avant tout lecteur.
	// `match_kill_heads` / `match_kill_pairs` / `match_assist_pairs` AJOUTÉES LE 2026-10-21 (ADR
	// 0039) : résumés par match du journal, vues `_latest` décidées par la tête ; enrôlées à leur
	// création, avant tout lecteur.
	rawRe := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+(match_skill_rank|match_csrs|player_csr_snapshots|pve_match_stats|match_lives|match_death_context|match_death_mates|match_life_placement|match_kill_events|kill_positions|match_bomb_stats|match_flag_grabs_net|match_pad_pickups_by_tier|match_vehicle_takes|match_lead_share|match_kill_heads|match_kill_pairs|match_assist_pairs)(_latest(?:_by_type)?)?\b`)

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
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if _, allowed := rawAppendOnlyReadAllowlist[rel]; allowed {
				return nil
			}
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
