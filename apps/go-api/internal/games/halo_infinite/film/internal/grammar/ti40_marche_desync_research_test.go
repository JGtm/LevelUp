//go:build research

package grammar

// ti40_marche_desync_research_test.go — LOT 5.1.7 : OU LA MARCHE DE `ti=40` SE DESYNCHRONISE.
//
// # LA QUESTION
//
// Le lot 5.1.4 a mesure que le film ANNONCE par masque 52 / 21 / 11 dead-states de vehicule sur
// `a349fea8` / `a521164d` / `4f77afc1` et que la marche en PERD 47 / 20 / 11 : le compteur
// `MaskDeclaredDesync` dit COMBIEN, il ne dit pas OU. Cet instrument rend l histogramme du
// PREMIER composant non consomme (`EntityTrace.DesyncAt`) sur les records `ti=40`, separement
// pour ceux dont le masque declare le dead-state.
//
// Un desync a un index > celui du dead-state est une QUEUE inconnue (la tete est lue, la mort est
// acceptee, cf. `object_deaths.go`) ; un desync a un index <= celui du dead-state est une PERTE.
//
// DEUX CADRES SONT MESURES, et c est le point : la cuisson installe les LARGEURS MPP DU FILM
// (`gwInstallMPPWidths` dans `replay/build_vehicles.go`) pour toute la duree du calque des
// vehicules — la marche des morts comprise. `object-multiplayer-properties-component` est `i9`
// de `ti=40`, donc AVANT le dead-state `i11` : ces largeurs decident de l alignement du record.
//
// LECTURE SEULE, N ASSERTE RIEN, UN SEUL FILM PAR INVOCATION, aucune base ouverte :
//
//	TI40D_FILM=<repo>/data/cache/film_chunks/4f77afc1 TI40D_CARTE="Flood Gulch" \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run '^TestTi40MarcheDesync$' -v -timeout 60m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// ti40dReleve porte les comptes d un balayage.
type ti40dReleve struct {
	records, propres    int
	masque, masquePerdu int
	parIndex            map[int]int
	parIndexMasque      map[int]int
	nomParIndex         map[int]string
	mortsLues           int
}

func TestTi40MarcheDesync(t *testing.T) {
	dir, carte := os.Getenv("TI40D_FILM"), os.Getenv("TI40D_CARTE")
	if dir == "" || carte == "" {
		t.Skip("instrument de mesure : TI40D_FILM et TI40D_CARTE requis")
	}
	nom := filepath.Base(dir)
	fc := ti40dContexte(t, dir, carte)
	t.Logf("%s : MPP par defaut = %+v", nom, fc.ProfilDeBalayage().MPP)
	ti40dTemoinProduction(t, nom+" [param_4 NON impose]", fc)
	ti40dPasse(t, nom+" [param_4 NON impose]", fc)

	ti40dMatrice(t, nom, fc)
}

// ti40dPasse mesure un cadre : l histogramme du premier composant non consomme.
func ti40dPasse(t *testing.T, etiquette string, fc *FilmContext) {
	t.Helper()
	ti40dPublier(t, etiquette, ti40dBalayer(t, fc))
}

// ti40dContexte ouvre le contexte de film EXACTEMENT comme la cuisson : entree de catalogue de
// la carte pour le profil, largeurs d axe du catalogue sur le profil de balayage.
func ti40dContexte(t *testing.T, dir, carte string) *FilmContext {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	entree, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q : %v", carte, err)
	}
	fc := NewFilmContextForMap(film, &entree, nil)
	bal := fc.ProfilDeBalayage()
	bal.PoserLargeursObjetDuMondeDepuisDecoupage(entree.Layout())
	fc.PoserProfilDeBalayage(bal)
	return fc
}

// ti40dTemoinProduction rejoue `ScanObjectDeaths` sur LE MEME contexte et publie ses compteurs.
func ti40dTemoinProduction(t *testing.T, etiquette string, fc *FilmContext) {
	t.Helper()
	morts, st, err := ScanObjectDeaths(fc)
	if err != nil {
		t.Fatalf("ScanObjectDeaths : %v", err)
	}
	ti := uint32(VehicleTypeIndex)
	n := 0
	for _, m := range morts {
		if m.TypeIndex == ti {
			n++
		}
	}
	t.Logf("%s TEMOIN ScanObjectDeaths : mortsToutesEntites=%d mortsVehicules=%d "+
		"masqueDeclareLeDeadState=%d dontDesynchronises=%d recordsAtteints=%d "+
		"recordsEntierementPortes=%d paquetsAEvenements=%d paquetsLocalises=%d cadreParDefaut=%v",
		etiquette, len(morts), n, st.MaskDeclared[ti], st.MaskDeclaredDesync[ti], st.Records[ti],
		st.CleanRecords[ti], st.EventPackets, st.LocatedPackets, st.CadreParDefaut)
}

// ti40dBalayer deroule la MEME marche que `ScanObjectDeaths` et releve, record par record, le
// premier composant non consomme.
func ti40dBalayer(t *testing.T, fc *FilmContext) ti40dReleve {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	kfs, deltas := marchPacketsOf(fc)
	cfg, _, _, _ := calibrateFrameConfig(reg, kfs, deltas, fc.CadreDeBalayage())
	di := ti40dIndexDeadState(reg)
	rel := ti40dReleve{parIndex: map[int]int{}, parIndexMasque: map[int]int{},
		nomParIndex: map[int]string{}}
	tl := newMarchTimeline(reg, kfs)
	for _, d := range deltas {
		w := tl.advanceTo(d.timestampUS)
		start, _, ok := marchStartOf(d.payload, w, cfg)
		if !ok {
			continue
		}
		ti40dRanger(&rel, marchRecordsOf(d.payload, w, cfg, start), di)
	}
	return rel
}

// ti40dIndexDeadState resout l index du dead-state dans l archetype 40, par le NOM du registre.
func ti40dIndexDeadState(reg *Registry) int {
	a, ok := reg.Archetype(VehicleTypeIndex)
	if !ok {
		return -1
	}
	for i, c := range a.Components {
		if c == deadStateComponentName {
			return i
		}
	}
	return -1
}

// ti40dRanger classe les records `ti=40` d un paquet.
func ti40dRanger(rel *ti40dReleve, recs []FrameRecord, di int) {
	for i := range recs {
		r := &recs[i]
		if r.TypeIndex != uint32(VehicleTypeIndex) {
			continue
		}
		rel.records++
		if r.Trace.Dead != nil && r.Trace.Dead.Mort {
			rel.mortsLues++
		}
		declare := di >= 0 && di < 64 && r.Trace.Mask&(1<<uint(di)) != 0
		if declare {
			rel.masque++
		}
		if r.DesyncAt == -1 {
			rel.propres++
			continue
		}
		rel.parIndex[r.DesyncAt]++
		if declare {
			rel.parIndexMasque[r.DesyncAt]++
			rel.masquePerdu++
		}
		for _, c := range r.Trace.Comps {
			if c.Index == r.DesyncAt {
				rel.nomParIndex[r.DesyncAt] = c.Name
			}
		}
	}
}

// ti40dPublier ecrit le releve.
func ti40dPublier(t *testing.T, etiquette string, rel ti40dReleve) {
	t.Helper()
	t.Logf("%s : records ti=40 = %d (entierement portes %d) | masque declare le dead-state = %d "+
		"(dont desynchronises %d) | morts lues = %d",
		etiquette, rel.records, rel.propres, rel.masque, rel.masquePerdu, rel.mortsLues)
	t.Logf("%s : desync par index (TOUS les records ti=40) — %s", etiquette,
		ti40dHisto(rel.parIndex, rel.nomParIndex))
	t.Logf("%s : desync par index (records dont le MASQUE declare le dead-state) — %s", etiquette,
		ti40dHisto(rel.parIndexMasque, rel.nomParIndex))
}

// ti40dHisto rend un histogramme trie par index croissant.
func ti40dHisto(h map[int]int, noms map[int]string) string {
	if len(h) == 0 {
		return "(aucun desync)"
	}
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var b strings.Builder
	for _, k := range keys {
		if b.Len() > 0 {
			b.WriteString(" | ")
		}
		b.WriteString("i" + strconv.Itoa(k) + " " + noms[k] + " : " + strconv.Itoa(h[k]))
	}
	return b.String()
}

// ti40dMatrice balaye les TROIS grandeurs que `killsource` impose a la marche de la cuisson :
// la generation stricte (`ProfilDeDepart`), le mot de poignee `Traversal.IndexW` et le `param_4`
// global. Le but est de NOMMER celle qui deplace les records `ti=40`.
func ti40dMatrice(t *testing.T, nom string, fc *FilmContext) {
	t.Helper()
	base := fc.ProfilDeBalayage()
	for _, stricte := range []bool{false, true} {
		for iw := uint(1); iw <= 3; iw++ {
			for v := uint32(0); v <= 5; v++ {
				p := base
				p.Grammaire.GenerationStricte = stricte
				p.Mouvement.Traversal.IndexW = iw
				p.PoserParamEtat(v)
				fc.PoserProfilDeBalayage(p)
				ti40dPublierCourt(t, fmt.Sprintf("%s [stricte=%t iw=%d param_4=%d]", nom, stricte, iw, v),
					ti40dBalayer(t, fc))
			}
		}
	}
	fc.PoserProfilDeBalayage(base)
}

// ti40dPublierCourt rend UNE ligne par cadre : la matrice en compte trente-six.
func ti40dPublierCourt(t *testing.T, etiquette string, rel ti40dReleve) {
	t.Helper()
	avant := 0
	for k, n := range rel.parIndex {
		if k <= 11 {
			avant += n
		}
	}
	t.Logf("%s : records=%d portes=%d masque=%d perdus=%d morts=%d avant_ou_a_i11=%d | premier desync = %s",
		etiquette, rel.records, rel.propres, rel.masque, rel.masquePerdu, rel.mortsLues, avant,
		ti40dPremier(rel.parIndexMasque, rel.nomParIndex))
}

// ti40dPremier rend le PLUS PETIT index de desync d un histogramme, avec son nom.
func ti40dPremier(h map[int]int, noms map[int]string) string {
	if len(h) == 0 {
		return "(aucun)"
	}
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return "i" + strconv.Itoa(keys[0]) + " " + noms[keys[0]] + " x" + strconv.Itoa(h[keys[0]])
}
