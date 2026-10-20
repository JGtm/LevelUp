package main

// frequentation.go — LES HAUTEURS OU LES JOUEURS SE RENDENT.
//
// L'oracle est le meme que celui qui a juge la reconstruction des cartes (`himap`,
// carte_oracle_gamefiles_test.go) : les positions de joueur decodees des films. Un joueur qui
// a couru quelque part avait forcement du sol sous les pieds — c'est la seule mesure de ce
// chantier qui ne vienne pas de la geometrie elle-meme, donc la seule qui ne puisse pas etre
// tautologique.
//
// LE CHAMP EXACT : `tracks[].points[].z` de l'artefact de rejeu cuit
// (`replay.Point.Z`, document_aim.go:50 — float32, `json:"z,omitempty"`). Il est deja dans le
// REPERE MONDE de la geometrie : c'est ce meme champ que l'oracle des 29 221 positions
// confronte au volume de praticabilite reconstruit.
//
// FILM PAR FILM, JAMAIS LE CORPUS EN UN BLOC : chaque document est lu, resume en histogramme,
// puis relache. Le corpus entier charge en un processus est une bombe RAM vecue (registre).

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// filmMesure resume un document de rejeu.
type filmMesure struct {
	id     string
	module string
	// hist porte les altitudes de CE film — de quoi retirer un film du total sans relire le
	// corpus (le controle de stabilite « un film en moins », cf. rapport.go).
	hist *histogramme
	// ecart dit POURQUOI un film n'a ete rattache a aucune carte.
	ecart string
}

// frequentation est le resultat de la passe corpus.
type frequentation struct {
	// parCarte : histogramme d'altitude cumule, par module.
	parCarte map[string]*histogramme
	// films : un resume par document lu, dans l'ordre des noms de fichier.
	films []filmMesure
	// lus / rattaches : combien de documents ont ete lus, et combien ont trouve leur carte.
	lus       int
	rattaches int
}

// mesureFrequentation lit tous les documents de rejeu d'un repertoire et rend, par carte, la
// distribution des altitudes jouees.
func mesureFrequentation(ctx context.Context, dir string, cartes []carte) (*frequentation, error) {
	entrees, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("documents de rejeu illisibles (%s) : %w", dir, err)
	}
	noms := make([]string, 0, len(entrees))
	for _, e := range entrees {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			noms = append(noms, e.Name())
		}
	}
	sort.Strings(noms)
	f := &frequentation{parCarte: map[string]*histogramme{}}
	for _, nom := range noms {
		f.ajouteFilm(ctx, filepath.Join(dir, nom), cartes)
	}
	slog.InfoContext(ctx, "corpus de rejeux resume", "documents", f.lus,
		"rattaches", f.rattaches, "cartes", len(f.parCarte), "dir", dir)
	return f, nil
}

// ajouteFilm lit UN document, le rattache a une carte et verse ses altitudes. Les positions
// brutes ne survivent pas a l'appel.
func (f *frequentation) ajouteFilm(ctx context.Context, chemin string, cartes []carte) {
	pts, err := litPositions(chemin)
	if err != nil {
		// Journalise, jamais avale : un document illisible est une donnee manquante, et c'est
		// elle qu'on ira chercher si une carte manque au tableau.
		slog.WarnContext(ctx, "document de rejeu illisible", "err", err, "path", chemin)
		return
	}
	if len(pts) == 0 {
		slog.WarnContext(ctx, "document de rejeu sans position", "path", chemin)
		return
	}
	f.lus++
	id := strings.TrimSuffix(filepath.Base(chemin), ".json")
	module, ecart := reconnait(cartes, pts)
	m := filmMesure{id: id, module: module, ecart: ecart, hist: nouvelHistogramme()}
	if module == "" {
		slog.InfoContext(ctx, "film non rattache a une carte", "film", id, "ecart", ecart)
		f.films = append(f.films, m)
		return
	}
	for _, p := range pts {
		m.hist.ajoute(p[2])
	}
	h := f.parCarte[module]
	if h == nil {
		h = nouvelHistogramme()
		f.parCarte[module] = h
	}
	for _, p := range pts {
		h.ajoute(p[2])
	}
	f.rattaches++
	f.films = append(f.films, m)
}

// litPositions decode les positions 3D d'un document de rejeu. La structure anonyme ne lit que
// ce dont l'instrument a besoin : un artefact porte des dizaines de calques, les charger tous
// ne ferait que consommer de la memoire.
func litPositions(chemin string) ([][3]float64, error) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // corpus hors ligne, lecture seule
	if err != nil {
		return nil, err
	}
	var doc struct {
		Tracks []struct {
			Points []struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
				Z float64 `json:"z"`
			} `json:"points"`
		} `json:"tracks"`
	}
	if err := json.Unmarshal(blob, &doc); err != nil {
		return nil, err
	}
	var pts [][3]float64
	for _, tr := range doc.Tracks {
		for _, p := range tr.Points {
			pts = append(pts, [3]float64{p.X, p.Y, p.Z})
		}
	}
	return pts, nil
}

// hauteurMaxSansFilm rend la plus haute altitude jouee sur une carte en RETIRANT un film.
//
// C'est le controle de STABILITE de l'estimateur : si retirer un seul film fait descendre le
// maximum de plusieurs metres, alors le corpus ne borne pas la carte — il borne ce que ce
// corpus-la a visite, et un plafond pose dessus rognerait un etage praticable des qu'un match
// y monterait. NaN quand il ne reste plus rien.
func (f *frequentation) hauteurMaxSansFilm(module, filmID string) float64 {
	best := math.NaN()
	for i := range f.films {
		if f.films[i].module != module || f.films[i].id == filmID {
			continue
		}
		m := f.films[i].hist.maximum()
		if math.IsNaN(best) || m > best {
			best = m
		}
	}
	return best
}

// filmsDe rend les films rattaches a une carte.
func (f *frequentation) filmsDe(module string) []filmMesure {
	var out []filmMesure
	for i := range f.films {
		if f.films[i].module == module {
			out = append(out, f.films[i])
		}
	}
	return out
}
