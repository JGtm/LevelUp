// census.go — LE RECENSEMENT DU CORPUS PAR MODE : combien de films, et de quel schema.
//
// # Pourquoi ce fichier vit ICI et pas dans un binaire neuf
//
// La question — « combien de films de Total Control, d'Oddball, d'Extraction, de Stockpile
// avons-nous en cache ? » — est le prealable des phases D3 et D4 du plan des objectifs vivants.
// Y repondre demande exactement les trois pieces que cette commande reunit deja : la base en
// LECTURE SEULE (`OpenReadForQuery`, correct meme si le serveur la tient), le cache film sur
// disque, et la racine du depot. Un binaire de plus les rassemblerait a l'identique.
//
// # Ce que le recensement compte, et ce qu'il ne suppose pas
//
// Le mode vient du `pair_name` du registre, normalise par la MEME fonction que le service de
// rejeu (`analysis.NormalizeModeLabel`) : « Arena:Total Control on Streets » devient « Total
// Control ». Aucune liste de modes n'est ecrite ici — le recensement rend ce que le registre
// contient, y compris les modes qu'on n'attendait pas. Une liste en dur ne montrerait que ce
// qu'on sait deja.
//
// # La difference avec `-select-only`, et pourquoi les deux existent
//
// `-select-only` dimensionne le corpus MESURABLE d'un seul mode (les zones de Bastion) : il
// exige des bornes de carte et des formes au catalogue, parce que la mesure qui suit en depend.
// Le recensement, lui, ne filtre RIEN : il compte tous les matchs par mode, avec ou sans film.
// Le premier prepare une mesure, le second prepare une DECISION.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// censusMode agrege un mode : ses matchs, ceux qui ont un film, et les schemas des artefacts.
type censusMode struct {
	label string
	// matchs : lignes du registre portant ce mode.
	matchs int
	// avecFilm : matchs dont les chunks sont en cache — le corpus reellement exploitable.
	avecFilm int
	// avecArtefact : matchs dont un artefact de rejeu est cuit.
	avecArtefact int
	// schemas : combien d'artefacts par version de schema.
	schemas map[int]int
	// films : les identifiants COURTS des matchs dont le film est en cache. Ils ne sont
	// imprimes que pour les modes RARES (cf. censusMaxListe) : un mode a quatre films, c'est
	// un corpus qu'on va nommer film par film pour le mesurer ; un mode a deux cents, c'est un
	// agregat. La liste est gardee pour tous, le filtre est a l'impression.
	films []string
}

// censusMaxListe : au-dela de ce nombre de films, un mode reste agrege. Douze est le seuil ou
// une liste cesse d'etre lisible sur une ligne, et il couvre largement les modes que les phases
// D3 et D4 doivent nommer (Total Control 4, Oddball 7, Stockpile 2, Extraction 2).
const censusMaxListe = 12

// runCensus recense le corpus par mode et l'imprime. LECTURE SEULE de bout en bout.
func runCensus(ctx context.Context, db *sql.DB, slug, cacheDir, repoRoot string) error {
	cands, err := loadCandidates(ctx, db, "")
	if err != nil {
		return err
	}
	res := title.NewPathResolver(repoRoot)
	byMode := map[string]*censusMode{}
	sansMode := 0
	for _, c := range cands {
		label := analysis.NormalizeModeLabel(c.pairName)
		if label == "" {
			// Un match sans `pair_name` exploitable n'est PAS un mode : le compter sous une
			// etiquette vide melangerait un trou de registre a un mode reel.
			sansMode++
			continue
		}
		m := byMode[label]
		if m == nil {
			m = &censusMode{label: label, schemas: map[int]int{}}
			byMode[label] = m
		}
		m.matchs++
		if st, err := os.Stat(filmcache.ChunkDir(cacheDir, c.short)); err == nil && st.IsDir() {
			m.avecFilm++
			m.films = append(m.films, c.short)
		}
		if v, ok := artifactSchema(res.ReplayArtifactPath(slug, c.full)); ok {
			m.avecArtefact++
			m.schemas[v]++
		}
	}
	printCensus(byMode, len(cands), sansMode)
	return nil
}

// artifactSchema lit la version de schema d'un artefact de rejeu. Faux quand il n'existe pas
// ou ne se lit pas — un artefact illisible n'est PAS un artefact absent, mais il ne se compte
// pas davantage : les deux se voient au denominateur `avecArtefact`.
func artifactSchema(path string) (int, bool) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var doc struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(blob, &doc); err != nil {
		return 0, false
	}
	return doc.SchemaVersion, true
}

// printCensus imprime le tableau, modes tries par nombre de films DECROISSANT — c'est le
// corpus disponible qu'on vient lire, pas l'alphabet.
func printCensus(byMode map[string]*censusMode, total, sansMode int) {
	modes := make([]*censusMode, 0, len(byMode))
	for _, m := range byMode {
		modes = append(modes, m)
	}
	sort.Slice(modes, func(i, j int) bool {
		if modes[i].avecFilm != modes[j].avecFilm {
			return modes[i].avecFilm > modes[j].avecFilm
		}
		return modes[i].label < modes[j].label
	})
	fmt.Printf("RECENSEMENT — %d match(s) au registre, %d mode(s) distinct(s)", total, len(modes))
	if sansMode > 0 {
		fmt.Printf(", %d sans pair_name exploitable", sansMode)
	}
	fmt.Println()
	fmt.Printf("  %-34s %8s %8s %10s  %s\n", "MODE", "matchs", "films", "artefacts", "schemas")
	for _, m := range modes {
		fmt.Printf("  %-34s %8d %8d %10d  %s\n",
			m.label, m.matchs, m.avecFilm, m.avecArtefact, formatSchemas(m.schemas))
		// LES MODES RARES SONT NOMMES FILM PAR FILM : c'est leur corpus entier, et c'est ce
		// qu'une phase de mesure a besoin de citer. Trie pour que deux recensements se
		// comparent.
		if n := len(m.films); n > 0 && n <= censusMaxListe {
			sort.Strings(m.films)
			fmt.Printf("  %-34s %s\n", "", strings.Join(m.films, " "))
		}
	}
}

// formatSchemas rend « 18x3, 19x1 », ou un tiret quand aucun artefact n'est cuit.
func formatSchemas(s map[int]int) string {
	if len(s) == 0 {
		return "-"
	}
	vs := make([]int, 0, len(s))
	for v := range s {
		vs = append(vs, v)
	}
	sort.Ints(vs)
	out := ""
	for i, v := range vs {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%dx%d", v, s[v])
	}
	return out
}
