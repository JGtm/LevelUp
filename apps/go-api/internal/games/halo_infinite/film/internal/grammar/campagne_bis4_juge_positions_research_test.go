//go:build research && campagne_overlay

package grammar

// campagne_bis4_juge_positions_research_test.go — MESURES BIS 4 DE LA CAMPAGNE DE GRAMMAIRE
// (2026-10-02), RESTE DU POINT 25 DE LA CRITIQUE N° 1 : LE JUGE DES INVARIANTS DE L ECRIVAIN
// ([cmJuge]) JOUE SUR LES A/B DE POSITION DE MESURES_BIS_2 §3.
//
// CE FICHIER EXIGE LA SURCOUCHE DE RECHERCHE (tag `campagne_overlay`), comme
// `campagne_bis2_positions_research_test.go` dont il reprend les variantes ([b2pVariantes]) et la
// marche ([b2pMesure]), contexte des instruments ([cmOuvrir]). Aucun fichier de production n est
// modifie sur disque ; `grammar.Rev` ne bouge pas.
//
// Par film : la reference est marchee sous son juge (`ref` nil : le juge note, paquet par paquet,
// si la fermeture de reference contredit un invariant) ; puis chaque variante sous un juge neuf qui
// herite de ces notes (meme regle que `campagne_bis1_research_test.go`). Un paquet GAGNE est sain
// s il ne contredit aucun invariant ; un paquet PERDU est sain si la fermeture de reference qu il
// perd n en contredisait aucun (une perte « ref contredit » retire une fermeture factice, D2).
//
// Rejouable (un film a la fois, plafond 4 Gio) :
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 180m \
//	  -run '^TestCampagneBis4JugePositions$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// b4jCompte : ce que le juge dit d une variante.
type b4jCompte struct {
	fermes, fermesContre                          int
	gagnes, gagnesSains, gagnesContredits         int
	perdus, perdusSains, perdusRefContredits      int
	utilesGagnesSains, utilesPerdusSains          int
	conserveSainVersContredit, conserveInverse    int
	controleGagnes, controlePerdus, controleEcart int
}

var b4jColonnes = []string{"fermes", "fermes_contredits", "gagnes", "gagnes_sains", "gagnes_contredits",
	"perdus", "perdus_sains", "perdus_ref_contredits", "utiles_gagnes_sains", "utiles_perdus_sains",
	"conserves_sain_vers_contredit", "conserves_contredit_vers_sain", "ecart_controle"}

func (c *b4jCompte) valeurs() []int {
	return []int{c.fermes, c.fermesContre, c.gagnes, c.gagnesSains, c.gagnesContredits, c.perdus,
		c.perdusSains, c.perdusRefContredits, c.utilesGagnesSains, c.utilesPerdusSains,
		c.conserveSainVersContredit, c.conserveInverse, c.controleEcart}
}

func (c *b4jCompte) ajouter(x *b4jCompte) {
	c.fermes += x.fermes
	c.fermesContre += x.fermesContre
	c.gagnes += x.gagnes
	c.gagnesSains += x.gagnesSains
	c.gagnesContredits += x.gagnesContredits
	c.perdus += x.perdus
	c.perdusSains += x.perdusSains
	c.perdusRefContredits += x.perdusRefContredits
	c.utilesGagnesSains += x.utilesGagnesSains
	c.utilesPerdusSains += x.utilesPerdusSains
	c.conserveSainVersContredit += x.conserveSainVersContredit
	c.conserveInverse += x.conserveInverse
	c.controleEcart += x.controleEcart
}

// b4jLire tire du juge les gains et pertes sains ; `ligne` est la ligne de [b2pMesure] (ses
// colonnes 8 et 9 : gagnes et perdus du comparateur, controle croise).
func b4jLire(j *cmJuge, ligne string) *b4jCompte {
	c := &b4jCompte{fermes: j.fermes, fermesContre: j.fermesContre, gagnes: j.gagnes,
		gagnesContredits: j.gagnesContredits, perdus: j.perdus}
	gainSain := cmJoindre("gagne", "aucun invariant contredit", "")
	for k, x := range j.t["juge"] {
		switch {
		case strings.HasPrefix(k, gainSain):
			c.gagnesSains += x.n
			c.utilesGagnesSains += x.enJeu
		case k == cmJoindre("perdu", "ref sain"):
			c.perdusSains += x.n
			c.utilesPerdusSains += x.enJeu
		case k == cmJoindre("perdu", "ref contredit"):
			c.perdusRefContredits += x.n
		case k == cmJoindre("conserve", "ref sain", "variante contredit"):
			c.conserveSainVersContredit += x.n
		case k == cmJoindre("conserve", "ref contredit", "variante sain"):
			c.conserveInverse += x.n
		}
	}
	col := strings.Split(ligne, "\t")
	if len(col) > 8 {
		c.controleGagnes, _ = strconv.Atoi(col[7])
		c.controlePerdus, _ = strconv.Atoi(col[8])
	}
	// Controles : le juge et le comparateur comptent les memes paquets ; les gains se partagent
	// en sains et contredits, les pertes en « ref sain » et « ref contredit ».
	if c.controleGagnes != c.gagnes || c.controlePerdus != c.perdus ||
		c.gagnesSains+c.gagnesContredits != c.gagnes || c.perdusSains+c.perdusRefContredits != c.perdus {
		c.controleEcart = 1
	}
	return c
}

// b4jFormat rend une ligne : prefixe puis les colonnes.
func b4jFormat(prefixe string, c *b4jCompte) string {
	var b strings.Builder
	b.WriteString(prefixe)
	for _, v := range c.valeurs() {
		fmt.Fprintf(&b, "\t%d", v)
	}
	return b.String()
}

// TestCampagneBis4JugePositions : le juge sur les 16 A/B de position (contexte des instruments).
func TestCampagneBis4JugePositions(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	entete := "film\tbuild\tvariante\t" + strings.Join(b4jColonnes, "\t")
	lignes := []string{entete}
	agg := map[[2]string]*b4jCompte{}
	sommer := func(variante, build string, c *b4jCompte) {
		for _, cle := range [][2]string{{variante, build}, {variante, "corpus"}} {
			if agg[cle] == nil {
				agg[cle] = &b4jCompte{}
			}
			agg[cle].ajouter(c)
		}
	}
	vs := b2pVariantes()
	for _, id := range films {
		garde := filmproc.Arm("campagne/bis4-juge-positions", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		debut := time.Now()
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		blocs := cmLireBlocs(f)
		jref := cmNouveauJuge(f, blocs, nil)
		lref, statut := b2pMesure(f, vs[0], nil, nil, jref)
		cref := b4jLire(jref, lref)
		lignes = append(lignes, b4jFormat(id+"\t"+f.build+"\t"+vs[0].nom, cref))
		sommer(vs[0].nom, f.build, cref)
		refB := make(map[[2]int]bool, len(statut))
		for k := range statut {
			refB[k] = true
		}
		for _, v := range vs[1:] {
			j := cmNouveauJuge(f, blocs, refB)
			j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
			l, _ := b2pMesure(f, v, statut, nil, j)
			c := b4jLire(j, l)
			if c.controleEcart != 0 {
				t.Errorf("%s %s : juge %d/%d, comparateur %d/%d", id, v.nom, c.gagnes, c.perdus,
					c.controleGagnes, c.controlePerdus)
			}
			lignes = append(lignes, b4jFormat(id+"\t"+f.build+"\t"+v.nom, c))
			sommer(v.nom, f.build, c)
		}
		t.Logf("%s %s : %d variantes jugees ; pic %d Mio, %s", id, f.build, len(vs), garde.Peak()>>20,
			time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
	parBuild := []string{"variante\tbuild\t" + strings.Join(b4jColonnes, "\t")}
	cles := make([][2]string, 0, len(agg))
	for k := range agg {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		if cles[i][0] != cles[j][0] {
			return cles[i][0] < cles[j][0]
		}
		return cles[i][1] < cles[j][1]
	})
	for _, k := range cles {
		parBuild = append(parBuild, b4jFormat(k[0]+"\t"+k[1], agg[k]))
	}
	b2Ecrire(t, sortie, "mb4_juge_positions.tsv", lignes)
	b2Ecrire(t, sortie, "mb4_juge_positions_par_build.tsv", parBuild)
}
