package main

// corpus.go — LE RECENSEMENT : quelles cartes le corpus couvre, avec combien de matchs
// porteurs de positions de kills, et combien d'artefacts de rejeu en cache.
//
// POURQUOI IL PASSE AVANT TOUT LE RESTE (item 0.1 du plan). La derivation n'a de sens que
// la ou il y a de la matiere ; « pas de calque sans preuve » (D8) exige un plancher de
// matchs par carte, et ce plancher se choisit sur la distribution reelle, pas d'avance.
//
// LA CLE D'AGREGATION EST `decfilm.NormalizeMapName` (cf. l'en-tete de main.go) : le nom
// affiche rabote de ses suffixes de variante de playlist. La CATEGORIE vient de
// `halo_infinite.InferModeCategoryFromPairName`, la meme fonction que les filtres de l'app —
// la colonne `mode_category` du registre n'est PAS utilisable ici : mesure du 2026-09-20,
// elle vaut « Other » sur 126 matchs de « Ranked:Strongholds on Live Fire ».

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/domain/title"
	halo "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/platform/duckdb"
)

// Axes du corpus : la dichotomie de format qui change la carte jouee (12 v 12 contre 4 v 4).
const (
	AxeArene = "arene"
	AxeBTB   = "btb"
	AxePvE   = "pve"
)

// MatchCorpus est un match du registre, reduit a ce que la derivation consomme.
type MatchCorpus struct {
	MatchID   string
	MapName   string // nom affiche, tel quel
	MapID     string
	Carte     string // decfilm.NormalizeMapName(MapName) — la cle d'agregation
	Categorie string // halo.InferModeCategoryFromPairName(pair_name)
	Axe       string // AxeArene / AxeBTB / AxePvE
	NbKills   int    // lignes de kill_positions_latest
	Artefact  string // chemin de l'artefact de rejeu, vide si absent du cache
}

// Corpus est le registre indexe, plus les agregats par carte.
type Corpus struct {
	TitleSlug string
	Matchs    map[string]MatchCorpus // par match_id
	Cartes    []CarteCorpus          // trie, le plus fourni d'abord

	SansNomDeCarte     int // matchs ecartes faute de nom de carte
	SansCorrespondance int // artefacts de rejeu sans match au registre
}

// CarteCorpus agrege une (carte, axe).
type CarteCorpus struct {
	Carte       string
	Axe         string
	Categories  []string
	MatchsKills int // matchs porteurs d'au moins une position de kill
	Kills       int
	MatchsTotal int // matchs joues sur la carte, positions ou non
	Artefacts   int
}

// ChargeCorpus lit le registre, les comptes de kills et l'inventaire des artefacts.
func ChargeCorpus(res *title.PathResolver, titleSlug string) (*Corpus, error) {
	chemin := res.SharedDBPath(titleSlug)
	db, release, err := duckdb.OpenReadForQuery(chemin)
	if err != nil {
		return nil, fmt.Errorf("base partagee illisible (%s) : %w", chemin, err)
	}
	defer release()

	c := &Corpus{TitleSlug: titleSlug, Matchs: map[string]MatchCorpus{}}
	if err := c.lisRegistre(db); err != nil {
		return nil, err
	}
	if err := c.lisComptesDeKills(db); err != nil {
		return nil, err
	}
	c.lisArtefacts(res.ReplayArtifactsDir(titleSlug))
	c.agrege()
	slog.Info("mappower: corpus charge", "titleSlug", titleSlug, "matchs", len(c.Matchs),
		"cartes", len(c.Cartes), "sans_nom_de_carte", c.SansNomDeCarte,
		"artefacts_sans_match", c.SansCorrespondance)
	return c, nil
}

// lisRegistre charge tous les matchs nommes du registre.
func (c *Corpus) lisRegistre(db *sql.DB) error {
	const q = `SELECT match_id, COALESCE(map_name, ''), COALESCE(map_id, ''),
		COALESCE(pair_name, ''), COALESCE(is_firefight, FALSE)
		FROM match_registry`
	rows, err := db.QueryContext(context.Background(), q)
	if err != nil {
		return fmt.Errorf("lecture de match_registry : %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var m MatchCorpus
		var pairName string
		var firefight bool
		if err := rows.Scan(&m.MatchID, &m.MapName, &m.MapID, &pairName, &firefight); err != nil {
			return fmt.Errorf("lecture d'une ligne de match_registry : %w", err)
		}
		if strings.TrimSpace(m.MapName) == "" {
			// Journalise en agregat, jamais avale : un match sans nom de carte n'est
			// rattachable a aucun calque.
			c.SansNomDeCarte++
			continue
		}
		m.Carte = decfilm.NormalizeMapName(m.MapName)
		m.Categorie = halo.InferModeCategoryFromPairName(pairName)
		m.Axe = axeDe(m.Categorie, firefight)
		c.Matchs[m.MatchID] = m
	}
	return rows.Err()
}

// axeDe range un match sur l'axe de format. Le PvE est a part : il se joue sur les memes
// cartes mais contre des vagues d'IA — ses kills ne disent rien d'un duel entre equipes.
func axeDe(categorie string, firefight bool) string {
	switch {
	case firefight || categorie == halo.ModeCategoryFirefight:
		return AxePvE
	case categorie == halo.ModeCategoryBTB:
		return AxeBTB
	default:
		return AxeArene
	}
}

// lisComptesDeKills compte les positions de kill par match, sur la VUE `_latest`.
func (c *Corpus) lisComptesDeKills(db *sql.DB) error {
	const q = `SELECT match_id, count(1) FROM kill_positions_latest
		WHERE killer_x IS NOT NULL AND victim_x IS NOT NULL GROUP BY 1`
	rows, err := db.QueryContext(context.Background(), q)
	if err != nil {
		return fmt.Errorf("lecture de kill_positions_latest : %w", err)
	}
	defer rows.Close()
	orphelins := 0
	for rows.Next() {
		var matchID string
		var n int
		if err := rows.Scan(&matchID, &n); err != nil {
			return fmt.Errorf("lecture d'un compte de kills : %w", err)
		}
		m, ok := c.Matchs[matchID]
		if !ok {
			orphelins++
			continue
		}
		m.NbKills = n
		c.Matchs[matchID] = m
	}
	if orphelins > 0 {
		slog.Warn("mappower: positions de kill sans match nomme au registre", "matchs", orphelins)
	}
	return rows.Err()
}

// lisArtefacts inventorie le cache des artefacts de rejeu. L'absence du repertoire n'est
// pas fatale : la presence par equipe est un signal OPTIONNEL du corpus.
func (c *Corpus) lisArtefacts(dir string) {
	entrees, err := os.ReadDir(dir)
	if err != nil {
		slog.Warn("mappower: cache d'artefacts de rejeu illisible — aucune mesure de presence",
			"err", err, "dir", dir)
		return
	}
	court := make(map[string]string, len(c.Matchs))
	for id := range c.Matchs {
		court[title.FilmShortMatchID(id)] = id
	}
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".json") || strings.HasSuffix(nom, ".derived.json") {
			continue
		}
		matchID, ok := court[strings.TrimSuffix(nom, ".json")]
		if !ok {
			c.SansCorrespondance++
			continue
		}
		m := c.Matchs[matchID]
		m.Artefact = filepath.Join(dir, nom)
		c.Matchs[matchID] = m
	}
}

// agrege construit la table par (carte, axe), triee par matchs porteurs de kills.
func (c *Corpus) agrege() {
	type cle struct{ carte, axe string }
	par := map[cle]*CarteCorpus{}
	categories := map[cle]map[string]bool{}
	for _, m := range c.Matchs {
		k := cle{m.Carte, m.Axe}
		entree := par[k]
		if entree == nil {
			entree = &CarteCorpus{Carte: m.Carte, Axe: m.Axe}
			par[k] = entree
			categories[k] = map[string]bool{}
		}
		entree.MatchsTotal++
		categories[k][m.Categorie] = true
		if m.NbKills > 0 {
			entree.MatchsKills++
			entree.Kills += m.NbKills
		}
		if m.Artefact != "" {
			entree.Artefacts++
		}
	}
	c.Cartes = make([]CarteCorpus, 0, len(par))
	for k, e := range par {
		for cat := range categories[k] {
			e.Categories = append(e.Categories, cat)
		}
		sort.Strings(e.Categories)
		c.Cartes = append(c.Cartes, *e)
	}
	sort.Slice(c.Cartes, func(i, j int) bool {
		if c.Cartes[i].MatchsKills != c.Cartes[j].MatchsKills {
			return c.Cartes[i].MatchsKills > c.Cartes[j].MatchsKills
		}
		if c.Cartes[i].Carte != c.Cartes[j].Carte {
			return c.Cartes[i].Carte < c.Cartes[j].Carte
		}
		return c.Cartes[i].Axe < c.Cartes[j].Axe
	})
}

// MatchsDe rend les matchs d'une (carte, axe), tries par identifiant pour que deux passes
// rendent le meme ordre.
func (c *Corpus) MatchsDe(carte, axe string) []MatchCorpus {
	var out []MatchCorpus
	for _, m := range c.Matchs {
		if m.Carte == carte && m.Axe == axe {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MatchID < out[j].MatchID })
	return out
}
