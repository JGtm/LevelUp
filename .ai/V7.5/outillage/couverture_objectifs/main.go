package main

// couverture-objectifs — instrument d'AUDIT hors ligne des calques d'objectif du rejeu 2D.
//
// Il confronte les artefacts cuits (data/cache/replays/<slug>/<8hex>.json) a l'oracle officiel
// de l'API Halo exporte en TSV (vue `match_objective_stats_latest`). Il n'ouvre AUCUNE base et
// n'ecrit rien dans le parc : lecture seule, sorties en TSV sur disque.
//
// Usage :
//
//	go run . -parc <dir artefacts> -oracle <dir TSV> -out <dir sorties>
//
// Six sorties, une par axe de l'audit (cf. AUDIT_COUVERTURE_OBJECTIFS_2026-09-10.md).

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	parc := flag.String("parc", "", "repertoire des artefacts <8hex>.json")
	oracle := flag.String("oracle", "", "repertoire des TSV oracle_vague6_*.tsv")
	out := flag.String("out", ".", "repertoire des sorties TSV")
	flag.Parse()
	if *parc == "" || *oracle == "" {
		fmt.Fprintln(os.Stderr, "usage: -parc <dir> -oracle <dir> [-out <dir>]")
		os.Exit(2)
	}
	ctx, err := charger(*parc, *oracle)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "erreur:", err)
		os.Exit(1)
	}
	ecrire(*out, "vague6_calibrage.log", ctx.calibrage())
	ecrire(*out, "vague6_couverture_parc.tsv", ctx.censusParc())
	ecrire(*out, "vague6_stats_publiees.tsv", ctx.statsPubliees())
	ecrire(*out, "vague6_couverture_portage.tsv", ctx.axe1Portage())
	ecrire(*out, "vague6_couverture_crane.tsv", ctx.axe1Crane())
	ecrire(*out, "vague6_bilan_portages.tsv", ctx.bilanPortages())
	ecrire(*out, "vague6_couverture_zones.tsv", ctx.axe1Zones())
	ecrire(*out, "vague6_couverture_actions.tsv", ctx.axe2Actions())
	ecrire(*out, "vague6_identite.tsv", ctx.axe3Identite())
	ecrire(*out, "vague6_bornage.tsv", ctx.axe4Bornage())
	ecrire(*out, "vague6_objet_sans_position.tsv", ctx.axe5SansPosition())
	fmt.Println("sorties ecrites dans", *out)
}

func ecrire(dir, nom string, lignes []string) {
	p := filepath.Join(dir, nom)
	if err := os.WriteFile(p, []byte(strings.Join(lignes, "\n")+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "ecriture", p, ":", err)
		os.Exit(1)
	}
	fmt.Printf("  %-40s %d lignes\n", nom, len(lignes)-1)
}

// --- contexte ----------------------------------------------------------------

type film struct {
	court string // 8 hex
	match string // uuid complet
	mode  string
	carte string
	duree int
	art   *artefact
	cov   *couverture
}

type ctxAudit struct {
	films []film
	// oracle[matchID][xuid] = ligne d'objective_stats
	oracle map[string]map[string][]string
	oTSV   *tsv
	// equipe[matchID][xuid] = team
	equipe map[string]map[string]string
	// gamertag[matchID][xuid]
	gt map[string]map[string]string
	// registre[court] = ligne de registre
	regParMatch map[string][]string
	rTSV        *tsv
}

func charger(parc, oracleDir string) (*ctxAudit, error) {
	reg, err := lireTSV(filepath.Join(oracleDir, "oracle_vague6_registry.tsv"))
	if err != nil {
		return nil, err
	}
	obj, err := lireTSV(filepath.Join(oracleDir, "oracle_vague6_objective_stats.tsv"))
	if err != nil {
		return nil, err
	}
	part, err := lireTSV(filepath.Join(oracleDir, "oracle_vague6_participants.tsv"))
	if err != nil {
		return nil, err
	}
	c := &ctxAudit{
		oracle:      map[string]map[string][]string{},
		oTSV:        obj,
		equipe:      map[string]map[string]string{},
		gt:          map[string]map[string]string{},
		regParMatch: map[string][]string{},
		rTSV:        reg,
	}
	for _, r := range obj.rows {
		m := obj.s(r, "match_id")
		if c.oracle[m] == nil {
			c.oracle[m] = map[string][]string{}
		}
		c.oracle[m][obj.s(r, "xuid")] = r
	}
	for _, r := range part.rows {
		m := part.s(r, "match_id")
		if c.equipe[m] == nil {
			c.equipe[m] = map[string]string{}
			c.gt[m] = map[string]string{}
		}
		c.equipe[m][part.s(r, "xuid")] = part.s(r, "team_id")
		c.gt[m][part.s(r, "xuid")] = part.s(r, "gamertag")
	}
	for _, r := range reg.rows {
		c.regParMatch[reg.s(r, "match_id")] = r
	}
	entrees, err := os.ReadDir(parc)
	if err != nil {
		return nil, err
	}
	for _, e := range entrees {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".json") || strings.Contains(n, ".derived.") {
			continue
		}
		a, cov, err := lireArtefact(filepath.Join(parc, n))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		f := film{court: strings.TrimSuffix(n, ".json"), match: a.MatchID, art: a, cov: cov}
		if r, ok := c.regParMatch[a.MatchID]; ok {
			f.mode = reg.s(r, "game_variant_name")
			f.carte = reg.s(r, "map_name")
			f.duree, _ = reg.i(r, "duration_seconds")
		}
		c.films = append(c.films, f)
	}
	sort.Slice(c.films, func(i, j int) bool {
		if c.films[i].mode != c.films[j].mode {
			return c.films[i].mode < c.films[j].mode
		}
		return c.films[i].court < c.films[j].court
	})
	return c, nil
}

// famille rend la famille d'objectif d'un mode (cle des tableaux de l'audit).
func famille(mode string) string {
	switch {
	case strings.Contains(mode, "CTF"):
		return "CTF"
	case strings.Contains(mode, "Oddball"):
		return "Oddball"
	case strings.Contains(mode, "Strongholds"):
		return "Strongholds"
	case strings.Contains(mode, "Total Control"):
		return "TotalControl"
	case strings.Contains(mode, "King of the Hill"), strings.Contains(mode, "KOTH"):
		return "KOTH"
	case strings.Contains(mode, "Assault"), strings.Contains(mode, "Assaut"):
		return "Assaut"
	case mode == "":
		return "INCONNU"
	default:
		return "SansObjectif"
	}
}

// --- census du parc (ce qui est publie, par film) ----------------------------

func (c *ctxAudit) censusParc() []string {
	l := []string{strings.Join([]string{"film", "famille", "mode", "carte", "duree_s", "schema",
		"frames", "manches", "flagCarries_calque", "flagFilm", "flag_openings", "flag_carries",
		"flag_noBridge", "zoneStates", "zones_captures", "zones_attributed", "skullCarries",
		"bombCarries", "obj_available", "obj_attached", "obj_noSlot", "obj_outOfWindow",
		"oracle_lignes"}, "\t")}
	for _, f := range c.films {
		fc, ff, fo, fca, fnb := "non", "", "", "", ""
		if f.cov.FlagCarries != nil {
			ff = fmt.Sprint(f.cov.FlagCarries.FlagFilm)
			fo = fmt.Sprint(f.cov.FlagCarries.Openings)
			fca = fmt.Sprint(f.cov.FlagCarries.Carries)
			fnb = fmt.Sprint(f.cov.FlagCarries.NoBridge)
		}
		if len(f.art.FlagCarries) > 0 {
			fc = "oui"
		}
		zc, za := "", ""
		if f.cov.Zones != nil {
			zc, za = fmt.Sprint(f.cov.Zones.Captures), fmt.Sprint(f.cov.Zones.Attributed)
		}
		l = append(l, strings.Join([]string{f.court, famille(f.mode), f.mode, f.carte,
			fmt.Sprint(f.duree), fmt.Sprint(f.art.SchemaVersion), fmt.Sprint(f.art.FrameCount),
			fmt.Sprint(f.cov.Score.Rounds), fc, ff, fo, fca, fnb,
			fmt.Sprint(len(f.art.ZoneStates)), zc, za,
			fmt.Sprint(len(f.art.SkullCarries)), fmt.Sprint(len(f.art.BombCarries)),
			fmt.Sprint(f.cov.Objectives.Available), fmt.Sprint(f.cov.Objectives.Attached),
			fmt.Sprint(f.cov.Objectives.NoSlot), fmt.Sprint(f.cov.Objectives.OutOfWindow),
			fmt.Sprint(len(c.oracle[f.match]))}, "\t"))
	}
	return l
}
