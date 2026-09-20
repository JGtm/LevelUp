//go:build research

package main

// verdict_lignees_research_test.go — LES QUATRE LIGNEES A REGLES EGALES, LE VERDICT STRICT
// ET LE VERDICT ELARGI, LA PLANCHE DE CONTROLE (item 2bis.D, seconde moitie, 2026-09-20).
//
// Le harnais juge UN fichier de positions (`MAPPOWER_POSITIONS`) ; ce fichier-ci lui donne
// de quoi comparer : chaque lignee du chantier dont le fichier existe (empirique v1,
// empirique v2, geometrie seule, fusion) est jugee avec les memes regles, et une table les
// met cote a cote. La lignee du fichier juge se deduit de son nom — pas d'une variable
// qu'on pourrait oublier.
//
// DEUX VERDICTS, NOMMES. Le critere du plan (>= 4 cartes a rappel >= 0,7 ET precision
// >= 0,6, zero piege pur) se lit sur les cartes de VALIDATION : c'est le verdict STRICT. Avec
// trois cartes de validation a geometrie cuite, quatre est hors d'atteinte par construction ;
// le verdict ELARGI compte aussi les cartes de calibrage — et il est nomme comme tel, parce
// qu'un GO elargi n'est pas un GO strict (le reglage de fusion a ete choisi sur ces cartes).
//
// LA PLANCHE. Pour chaque carte jugee : le fond publie, les zones nommees en filigrane, les
// zones FORTES de l'oracle en contour orange, les positions du fichier en contour vert. Le
// pilote doit VOIR l'ecart, pas le deduire d'une table.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// lignee est un fichier de positions connu du chantier.
type lignee struct {
	Nom, Fichier string
}

// ligneesConnues : dans l'ordre du chantier.
var ligneesConnues = []lignee{
	{"empirique v1", filepath.Join(verdictMesures, "positions.json")},
	{"empirique v2", filepath.Join(verdictV2Mesures, verdictV2PositionsNom)},
	{"geometrie seule", filepath.Join(verdictGeometrie, positionsGeoNom)},
	{"fusion", filepath.Join(verdictFusion, positionsFusionNom)},
}

// ligneeJugee est une lignee jugee aux regles v2.
type ligneeJugee struct {
	lignee
	Ordre   []string
	Reels   map[string]bilanV2
	Temoins map[string]bilanV2
}

// ligneeDuFichier nomme la lignee d'un fichier de positions par son nom de base.
func ligneeDuFichier(chemin string) string {
	switch filepath.Base(chemin) {
	case positionsFusionNom:
		return "FUSION geometrie x empirique"
	case positionsGeoNom:
		return "GEOMETRIE SEULE"
	case verdictV2PositionsNom:
		return "EMPIRIQUE V2 SEUL"
	case "positions.json":
		return "EMPIRIQUE V1 SEUL"
	default:
		return strings.ToUpper(strings.TrimSuffix(filepath.Base(chemin), ".json"))
	}
}

// jugeLignees juge chaque lignee dont le fichier existe.
func jugeLignees(t *testing.T, racine string, src sourcesV2) []ligneeJugee {
	t.Helper()
	var out []ligneeJugee
	for _, l := range ligneesConnues {
		chemin := filepath.Join(racine, verdictDossier, l.Fichier)
		if _, err := os.Stat(chemin); err != nil {
			t.Logf("lignee %s : fichier absent (%s), non comparee", l.Nom, chemin)
			continue
		}
		j := ligneeJugee{lignee: l}
		j.Ordre, j.Reels, j.Temoins = jugeDocument(t, src, litPositions(t, chemin))
		out = append(out, j)
	}
	return out
}

// reglageBrut rend l'objet `reglage` du fichier juge, compact, quelle que soit sa lignee.
func reglageBrut(chemin string) string {
	blob, err := os.ReadFile(chemin)
	if err != nil {
		return "(illisible)"
	}
	var doc struct {
		Reglage json.RawMessage `json:"reglage"`
	}
	if err := json.Unmarshal(blob, &doc); err != nil || len(doc.Reglage) == 0 {
		return "(absent)"
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, doc.Reglage); err != nil {
		return string(doc.Reglage)
	}
	return compact.String()
}

// trancheElargi rend les cartes qui tiennent parmi validation ET calibrage, et les pieges
// purs sur ces memes cartes. C'est le verdict ELARGI ; le strict est `trancheV2`.
func trancheElargi(ordre []string, reels map[string]bilanV2) ([]string, int) {
	var eligibles []string
	pieges := 0
	for _, nom := range ordre {
		b := reels[nom]
		if b.Role == roleHorsOracle || b.NbResolues == 0 {
			continue
		}
		pieges += len(b.ContreExemplesPurs)
		if tient(b) {
			eligibles = append(eligibles, nom)
		}
	}
	return eligibles, pieges
}

// tableLignees : chaque carte, chaque lignee, a regles egales.
func tableLignees(b *strings.Builder, v verdictV2Rendu) {
	fmt.Fprintf(b, "## 8. Les lignees a regles egales\n\n")
	fmt.Fprintf(b, "Chaque fichier de positions du chantier est juge ici avec les MEMES regles v2 (oracle v2,"+
		" precision sur les fortes, temoin geographique) : c'est la seule lecture comparee qui vaille."+
		" Live Fire v1 etait CONTAMINEE (variante classee incluse) : sa colonne v1 se lit avec cette"+
		" reserve. Une cellule « — » : la carte n'est pas dans le fichier de la lignee (la geometrie et"+
		" la fusion ne couvrent que les six cartes cuites ; Empyrean et Solitude, cartes Forge sans"+
		" geometrie, restent « empirique seul »).\n\n")
	fmt.Fprint(b, "| Carte | Role | Fortes |")
	for _, l := range v.Lignees {
		fmt.Fprintf(b, " %s : rappel / precision / positions / pieges purs |", l.Nom)
	}
	fmt.Fprintln(b)
	fmt.Fprint(b, "|---|---|---|")
	for range v.Lignees {
		fmt.Fprint(b, "---|")
	}
	fmt.Fprintln(b)
	for _, nom := range ordreDesLignees(v.Lignees) {
		role, fortes := roleEtFortes(v.Lignees, nom)
		if role == roleHorsOracle {
			continue
		}
		fmt.Fprintf(b, "| %s | %s | %d |", nom, role, fortes)
		for _, l := range v.Lignees {
			r, ok := l.Reels[nom]
			if !ok {
				fmt.Fprint(b, " — |")
				continue
			}
			fmt.Fprintf(b, " %.2f / %.2f / %d / %d (%s) |", r.Rappel, r.PrecisionForte, len(r.Positions),
				len(r.ContreExemplesPurs), lectureCourte(r))
		}
		fmt.Fprintln(b)
	}
	fmt.Fprintln(b)
	for _, l := range v.Lignees {
		strict, ps := trancheV2(l.Ordre, l.Reels)
		elargi, pe := trancheElargi(l.Ordre, l.Reels)
		fmt.Fprintf(b, "- **%s** : strict %d carte(s) de validation tiennent (%s), %d piege(s) pur(s) ;"+
			" elargi %d carte(s) validation + calibrage (%s), %d piege(s) pur(s).\n",
			l.Nom, len(strict), liste(strict), ps, len(elargi), liste(elargi), pe)
	}
	fmt.Fprintln(b)
}

// lectureCourte resume la lecture d'une carte en un mot.
func lectureCourte(b bilanV2) string {
	switch {
	case len(b.Positions) == 0:
		return "aucune position"
	case tient(b):
		return "TIENT"
	case b.NbResolues <= fortesIndeterminees && b.Rappel == 0.5:
		return "indetermine"
	default:
		return "echoue"
	}
}

// ordreDesLignees rend l'union triee des cartes de toutes les lignees.
func ordreDesLignees(lignees []ligneeJugee) []string {
	var tous []string
	for _, l := range lignees {
		tous = unionTriee(tous, l.Ordre)
	}
	return tous
}

// roleEtFortes rend le role et le nombre de fortes resolues d'une carte, lus dans la
// premiere lignee qui la porte.
func roleEtFortes(lignees []ligneeJugee, nom string) (string, int) {
	for _, l := range lignees {
		if r, ok := l.Reels[nom]; ok {
			return r.Role, r.NbResolues
		}
	}
	return roleHorsOracle, 0
}

// peinsVerdicts ecrit, a cote du fichier juge, une planche par carte : fond, zones
// nommees, fortes de l'oracle (orange), positions jugees (vert).
func peinsVerdicts(t *testing.T, racineDonnees string, doc SortiePositions, reels map[string]bilanV2) {
	t.Helper()
	res := title.NewPathResolver(racineDonnees)
	dossier := filepath.Dir(cheminPositionsJuge(racineDepot(t)))
	for _, c := range doc.Cartes {
		b, ok := reels[c.Carte]
		if !ok || b.Role == roleHorsOracle {
			continue
		}
		chemin := filepath.Join(dossier, nomDeFichier(c.Carte, c.Axe)+"_verdict.png")
		if err := peinsVerdict(res, c, b, chemin); err != nil {
			t.Logf("planche de verdict non produite (%s) : %v", c.Carte, err)
		}
	}
}

// peinsVerdict peint une carte.
func peinsVerdict(res *title.PathResolver, c SortieCartePos, b bilanV2, chemin string) error {
	fond, err := chargeFond(res, verdictTitleSlug, c.Carte)
	if err != nil {
		return err
	}
	zones := chargeZones(res, verdictTitleSlug, c.Carte, c.MapIDDominant)
	toile := peins(fond, zones, nil, nil)
	orange := color.NRGBA{R: 255, G: 150, B: 30, A: 240}
	for _, z := range b.Attendues {
		if z.Confiance != confianceForte || !z.Resolue {
			continue
		}
		for _, s := range z.surfaces {
			for _, anneau := range s.anneaux {
				tracePolygone(toile, fond.cal, anneau, orange)
			}
		}
	}
	vert := color.NRGBA{R: 60, G: 255, B: 140, A: 240}
	for _, p := range c.Positions {
		tracePolygone(toile, fond.cal, p.Polygone, vert)
	}
	return encode(chemin, toile)
}
