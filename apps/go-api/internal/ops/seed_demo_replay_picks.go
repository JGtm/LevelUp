// Package ops — seed_demo_replay_picks.go : le CHOIX des rejeux figés de la démo
// (`seed-demo --emit-replay-picks`), un par famille de mode de jeu, plus la famille « véhicules ».
//
// LA RÈGLE, déterministe pour un même état du dépôt :
//   - candidats : les matchs du registre dont le cache de films porte le film, et où le
//     joueur SOURCE de la démo a joué (le rejeu s'ouvre depuis sa vue match) ;
//   - exclus d'office : les matchs de mode non géré par le rejeu (Firefight, décision du
//     2026-10-15) — le drapeau du registre (`analysis.SQLModeNonGere`), pas le nom de variante ;
//   - famille : demoReplayFamilyOf(game_variant_name) ; un nom qu'aucune famille ne
//     reconnaît est journalisé et écarté ;
//   - par famille de mode : d'abord un match dont l'artefact du dépôt est déjà au dernier
//     schéma (pas de cuisson au premier seed), puis le plus RÉCENT, puis le plus petit match_id ;
//   - une famille de plus, « véhicules » (demoModeVehicles), choisie par les DONNÉES du match
//     et non par son nom de variante : seed_demo_replay_vehicles.go.
//
// LE RÉSULTAT EST FIGÉ dans le manifeste committé (`replay_matches`) : le seed ne choisit
// jamais, il lit la liste. Seules les listes de rejeux sont réécrites ; le reste du
// manifeste (corpus curé) est conservé tel quel. L'ÉMISSION EST ADDITIVE ([mergeDemoReplayPicks]) :
// une famille déjà figée garde son match.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/replaybuild"
)

// demoReplayFamily : une famille de mode et les jetons de `game_variant_name` qui la
// désignent (sous-chaîne, casse ignorée). L'ORDRE EST LA PRIORITÉ : « BTB:Fiesta CTF » est
// une Fiesta, « Arena:VIP » n'est pas un Assassin.
type demoReplayFamily struct {
	key    string
	tokens []string
}

// Clés des familles de mode (identifiants du manifeste, jamais des libellés). Chacune sert
// aussi de jeton quand le nom de variante la porte telle quelle.
const (
	demoModeFiesta       = "fiesta"
	demoModeEscalation   = "escalation"
	demoModeExtraction   = "extraction"
	demoModeVIP          = "vip"
	demoModeTotalControl = "total_control"
	demoModeStrongholds  = "strongholds"
	demoModeKOTH         = "koth"
	demoModeOddball      = "oddball"
	demoModeAssault      = "assault"
	demoModeCTF          = "ctf"
	demoModeSlayer       = "slayer"
)

// demoReplayFamilies : les familles de mode servies par la démo, dans l'ordre de priorité.
var demoReplayFamilies = []demoReplayFamily{
	{key: demoModeFiesta, tokens: []string{demoModeFiesta}},
	{key: demoModeEscalation, tokens: []string{demoModeEscalation}},
	{key: demoModeExtraction, tokens: []string{demoModeExtraction}},
	{key: demoModeVIP, tokens: []string{demoModeVIP}},
	{key: demoModeTotalControl, tokens: []string{"total control"}},
	{key: demoModeStrongholds, tokens: []string{demoModeStrongholds}},
	{key: demoModeKOTH, tokens: []string{"king of the hill", demoModeKOTH}},
	{key: demoModeOddball, tokens: []string{demoModeOddball}},
	{key: demoModeAssault, tokens: []string{"bomb", demoModeAssault}},
	{key: demoModeCTF, tokens: []string{demoModeCTF, "flag"}},
	{key: demoModeSlayer, tokens: []string{demoModeSlayer, "snipers"}},
}

// demoReplayFamilyOf rend la famille d'un `game_variant_name`, "" si aucune ne le reconnaît.
func demoReplayFamilyOf(variant string) string {
	v := strings.ToLower(variant)
	for _, f := range demoReplayFamilies {
		for _, tok := range f.tokens {
			if strings.Contains(v, tok) {
				return f.key
			}
		}
	}
	return ""
}

// demoReplayCandidate : un match candidat, tel que la sélection le lit.
type demoReplayCandidate struct {
	matchID  string
	variant  string
	mapName  string
	start    time.Time
	upToDate bool // artefact du dépôt au dernier schéma

	// Données de la famille « véhicules » (seed_demo_replay_vehicles.go) : 4 contre 4 au coup
	// d'envoi (établi pour les seuls matchs qui portent des prises), familles de véhicules
	// distinctes montées, et montées au total.
	fourVsFour   bool
	vehicleTypes int64
	vehicleTakes int64
}

// pickDemoReplays applique la règle : un match par famille de mode, plus le match de la
// famille « véhicules » (rendu en dernier). Rend les choix, et les variantes écartées des
// familles de mode (aucune famille).
func pickDemoReplays(cands []demoReplayCandidate) (picks []DemoReplayPick, unclassified []string) {
	best := map[string]demoReplayCandidate{}
	var bestVehicles demoReplayCandidate
	haveVehicles := false
	seenUnclassified := map[string]bool{}
	for _, c := range cands {
		if c.isVehicleDemoReplay() && (!haveVehicles || betterVehicleDemoReplay(c, bestVehicles)) {
			bestVehicles, haveVehicles = c, true
		}
		fam := demoReplayFamilyOf(c.variant)
		if fam == "" {
			if !seenUnclassified[c.variant] {
				seenUnclassified[c.variant] = true
				unclassified = append(unclassified, c.variant)
			}
			continue
		}
		if cur, ok := best[fam]; !ok || betterDemoReplay(c, cur) {
			best[fam] = c
		}
	}
	for _, f := range demoReplayFamilies {
		if c, ok := best[f.key]; ok {
			picks = append(picks, DemoReplayPick{MatchID: c.matchID, Mode: f.key, Map: c.mapName})
		}
	}
	if haveVehicles {
		picks = append(picks, DemoReplayPick{MatchID: bestVehicles.matchID, Mode: demoModeVehicles, Map: bestVehicles.mapName})
	}
	sort.Strings(unclassified)
	return picks, unclassified
}

// knownDemoReplayFamily dit si la clé désigne une famille que l'émission sait choisir.
func knownDemoReplayFamily(key string) bool {
	if key == demoModeVehicles {
		return true
	}
	for _, f := range demoReplayFamilies {
		if f.key == key {
			return true
		}
	}
	return false
}

// demoReplayMerge : le bilan d'une fusion additive, pour le journal.
type demoReplayMerge struct {
	picks   []DemoReplayPick
	kept    []string // familles dont l'entrée du manifeste est conservée
	filled  []string // familles absentes, remplies par le choix neuf
	unknown []string // entrées dont la famille n'est plus connue, conservées telles quelles
}

// mergeDemoReplayPicks fusionne le choix neuf `fresh` dans les entrées `existing` du manifeste,
// SANS RIEN RETIRER : une famille déjà présente garde son match (même quand un candidat plus
// récent existe), seules les familles absentes prennent le choix neuf, et une entrée dont la
// famille n'est plus connue est conservée telle quelle. Ordre rendu : les familles connues dans
// l'ordre de priorité, puis les entrées inconnues dans leur ordre d'origine.
func mergeDemoReplayPicks(existing, fresh []DemoReplayPick) demoReplayMerge {
	var out demoReplayMerge
	have := map[string]DemoReplayPick{}
	for _, p := range existing {
		have[p.Mode] = p
	}
	freshBy := map[string]DemoReplayPick{}
	for _, p := range fresh {
		freshBy[p.Mode] = p
	}
	keys := make([]string, 0, len(demoReplayFamilies)+1)
	for _, f := range demoReplayFamilies {
		keys = append(keys, f.key)
	}
	keys = append(keys, demoModeVehicles)
	for _, key := range keys {
		if p, ok := have[key]; ok {
			out.picks = append(out.picks, p)
			out.kept = append(out.kept, key)
		} else if p, ok := freshBy[key]; ok {
			out.picks = append(out.picks, p)
			out.filled = append(out.filled, key)
		}
	}
	for _, p := range existing {
		if !knownDemoReplayFamily(p.Mode) {
			out.picks = append(out.picks, p)
			out.unknown = append(out.unknown, p.Mode)
		}
	}
	return out
}

// betterDemoReplay : a passe-t-il devant b ? Artefact à jour, puis récence, puis match_id.
func betterDemoReplay(a, b demoReplayCandidate) bool {
	if a.upToDate != b.upToDate {
		return a.upToDate
	}
	if !a.start.Equal(b.start) {
		return a.start.After(b.start)
	}
	return a.matchID < b.matchID
}

// loadDemoReplayCandidates lit les matchs du joueur source dont le cache porte le film, avec
// les données de la famille « véhicules » (seed_demo_replay_vehicles.go).
func loadDemoReplayCandidates(ctx context.Context, opts SeedDemoOptions) ([]demoReplayCandidate, error) {
	pr := titlePkg.NewPathResolver(opts.cacheRepoRoot())
	films, err := filmcache.ListShortIDs(pr.CacheRootDir())
	if err != nil {
		return nil, err
	}
	hasFilm := make(map[string]bool, len(films))
	for _, s := range films {
		hasFilm[s] = true
	}
	shared, err := duckdb.OpenReadOnly(opts.SourceSharedDB)
	if err != nil {
		return nil, fmt.Errorf("base partagée source: %w", err)
	}
	defer func() { _ = shared.Close() }()
	rows, err := shared.SQLDb().QueryContext(ctx, vehicleDemoReplayClauses()+`
		SELECT r.match_id, COALESCE(r.game_variant_name, ''),
			COALESCE(r.map_name, ''), `+analysis.SQLStartTimeCanonical("r")+qDemoReplayVehicleSelect+`
		FROM match_registry r`+qDemoReplayVehicleJoins+`
		WHERE NOT `+analysis.SQLModeNonGere("r")+`
		  AND EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = r.match_id AND p.xuid = ?)`,
		opts.SourceXUID)
	if err != nil {
		return nil, fmt.Errorf("candidats aux rejeux démo: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []demoReplayCandidate
	for rows.Next() {
		var c demoReplayCandidate
		var start sql.NullTime
		if err := rows.Scan(&c.matchID, &c.variant, &c.mapName, &start,
			&c.fourVsFour, &c.vehicleTypes, &c.vehicleTakes); err != nil {
			return nil, fmt.Errorf("candidats aux rejeux démo (scan): %w", err)
		}
		if !hasFilm[titlePkg.FilmShortMatchID(c.matchID)] {
			continue
		}
		c.start = start.Time
		c.upToDate = replaybuild.ArtifactUpToDate(pr.ReplayArtifactPath(opts.TitleSlug, c.matchID))
		out = append(out, c)
	}
	return out, rows.Err()
}

// EmitDemoReplayPicks choisit les rejeux figés et les écrit dans le manifeste existant
// `manifestPath` (seule la liste `replay_matches` change). Rend la liste écrite.
//
// L'ÉMISSION EST ADDITIVE (décision D7 du plan de release v8.0.0 : les rejeux de la démo
// survivent aux releases suivantes) : une famille déjà présente au manifeste GARDE son match,
// même si un candidat plus récent existe ; seules les familles absentes sont remplies ; une
// entrée dont la famille n'est plus connue est conservée telle quelle et journalisée. UN RETRAIT
// OU UN CHANGEMENT DE MATCH SE FAIT À LA MAIN dans le manifeste (on retire l'entrée, puis on
// réémet pour la remplir par le choix du jour).
func EmitDemoReplayPicks(ctx context.Context, opts SeedDemoOptions, manifestPath string) ([]DemoReplayPick, error) {
	if err := validateSeedDemoOpts(&opts); err != nil {
		return nil, fmt.Errorf("emit-replay-picks: %w", err)
	}
	m, found, err := LoadDemoManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("emit-replay-picks: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("emit-replay-picks: manifeste absent (%s) — émettre d'abord le corpus", manifestPath)
	}
	cands, err := loadDemoReplayCandidates(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("emit-replay-picks: %w", err)
	}
	fresh, unclassified := pickDemoReplays(cands)
	if len(unclassified) > 0 {
		slog.WarnContext(ctx, "seed-demo: variantes sans famille de rejeu, écartées", "variantes", unclassified)
	}
	merged := mergeDemoReplayPicks(m.Corpus.ReplayMatches, fresh)
	if len(merged.unknown) > 0 {
		slog.WarnContext(ctx, "seed-demo: rejeux figés d'une famille inconnue, conservés tels quels",
			"familles", merged.unknown)
	}
	m.Corpus.ReplayMatches = merged.picks
	if err := writeDemoManifest(manifestPath, m); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "seed-demo: rejeux figés émis", "path", manifestPath,
		"candidats", len(cands), "rejeux", len(merged.picks),
		"conserves", merged.kept, "remplis", merged.filled)
	return merged.picks, nil
}
