package filmdec

// map_quant_control_test.go — LE CONTRÔLE DU CATALOGUE DE BORNES, FILM PAR FILM.
//
// CE QU'IL CONTRÔLE. `MapQuantEntry.AxisWidths` est DÉDUIT des bornes par la loi du moteur
// (W = min(26, ceilLog2(ceil(60*extent)))) ; `DetectI0Layout` LIT le découpage réel dans le
// bitstream du film, sans aucun a priori de largeur. Les deux doivent coïncider. Un désaccord
// dit que les BORNES sont fausses — c'est écrit depuis toujours dans le commentaire
// d'`AxisWidths`, et c'est ce qui a démasqué, le 2026-08-16, les bornes de décor lointain
// servies pour six canevas Forge.
//
// POURQUOI IL EXISTE. Ce contrôle était joué À LA MAIN, un film à la fois, par
// `TestWorldObjectPrecisionLayout` (19 films au lot alertes). À la main, il ne couvre que ce
// qu'on pense à lui donner : Starboard et Dredge portaient les mêmes fausses bornes que les
// trois canevas réfutés et personne ne les avait contrôlées. Ici la liste des couples
// (film, carte) est une DONNÉE, produite depuis le registre, et le verdict est agrégé par
// carte — une carte sans film le dit, elle ne disparaît pas du compte.
//
// CE QU'IL NE PEUT PAS DIRE, dit d'avance : `DetectI0Layout` suppose `DefaultI0GateBits`
// (5 bits d'en-tête = 3 spine + 1 useDefault + 1 index de région). Une carte à plus de deux
// BSP valides porte un index plus large et décale la PREMIÈRE largeur d'autant ; les
// deux autres, lues comme des écarts entre frontières, restent justes. CE CAS EXISTE depuis
// le 2026-08-27 (lot C catalogues) : Live Fire déclare 4 régions (index 2 bits,
// `regionIndexBits` de l'entrée), et le contrôle compare donc le découpage lu à
// [W0 + (bits-1), W1, W2] — l'écart d'index est une donnée du catalogue, jamais un
// ajustement au film.
//
// LECTURE SEULE, gardé par MAPQUANT_CTRL_PAIRES, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process : le verrou est pris pour toute la durée du contrôle.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 \
//	  MAPQUANT_CTRL_ROOT=<repo>/data/cache/film_chunks \
//	  MAPQUANT_CTRL_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  MAPQUANT_CTRL_PAIRES=<csv film,carte> \
//	  go test ./internal/analysis/filmdec/ -run '^TestControleBornesFilms$' -timeout 60m -v

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	ctrlRootEnv   = "MAPQUANT_CTRL_ROOT"
	ctrlBoundsEnv = "MAPQUANT_CTRL_BOUNDS"
	ctrlPairesEnv = "MAPQUANT_CTRL_PAIRES"
)

// ctrlPaire est un couple (film, carte affichée) à contrôler.
type ctrlPaire struct{ film, carte string }

// ctrlBilan agrège le verdict d'une carte sur tous ses films.
type ctrlBilan struct {
	accord, desaccord, illisible int
	attendu                      [3]uint
	lus                          map[[3]uint]int
}

func TestControleBornesFilms(t *testing.T) {
	paires, root, cat := ctrlEntrees(t)
	release := LockProcessDecode()
	defer release()

	bilans := map[string]*ctrlBilan{}
	var sansEntree []string
	for _, p := range paires {
		entry, err := cat.Lookup(p.carte)
		if err != nil {
			sansEntree = append(sansEntree, p.carte+" ("+p.film+")")
			continue
		}
		// Le détecteur lit avec un gate fixe de DefaultI0GateBits : sur une carte dont
		// l'index de région est plus large (regionIndexBits > 1, donnée du catalogue),
		// l'excédent d'index est lu comme des bits de X. L'attendu du contrôle intègre cet
		// écart — il vient du catalogue, jamais du film.
		attendu := entry.AxisWidths
		attendu[0] += entry.EffectiveRegionIndexBits() - 1
		b := bilans[p.carte]
		if b == nil {
			b = &ctrlBilan{attendu: attendu, lus: map[[3]uint]int{}}
			bilans[p.carte] = b
		}
		lay, rep, err := DetectI0Layout(filepath.Join(root, p.film))
		if err != nil {
			b.illisible++
			t.Logf("  %-38s %-10s DÉCOUPAGE ILLISIBLE (%v · %d paires · frontières %v)",
				p.carte, p.film, err, rep.Pairs, rep.Boundaries)
			continue
		}
		b.lus[lay.AxisW]++
		if lay.AxisW == attendu {
			b.accord++
			continue
		}
		b.desaccord++
		t.Errorf("  %-38s %-10s DÉSACCORD : catalogue %v · film %v (module %s)",
			p.carte, p.film, attendu, lay.AxisW, entry.Module)
	}
	ctrlPublie(t, bilans, sansEntree, len(paires))
}

// ctrlEntrees résout les trois entrées de l'instrument, ou déclare le test sauté.
func ctrlEntrees(t *testing.T) ([]ctrlPaire, string, *MapQuantCatalog) {
	t.Helper()
	pairesPath := os.Getenv(ctrlPairesEnv)
	if pairesPath == "" {
		t.Skipf("%s absent : contrôle sauté", ctrlPairesEnv)
	}
	root, boundsPath := os.Getenv(ctrlRootEnv), os.Getenv(ctrlBoundsEnv)
	if root == "" || boundsPath == "" {
		t.Fatalf("%s et %s sont requis avec %s", ctrlRootEnv, ctrlBoundsEnv, ctrlPairesEnv)
	}
	cat, err := LoadMapQuantCatalog(boundsPath)
	if err != nil {
		t.Fatalf("catalogue de bornes %s : %v", boundsPath, err)
	}
	paires, err := ctrlLisPaires(pairesPath)
	if err != nil {
		t.Fatalf("liste de couples %s : %v", pairesPath, err)
	}
	if len(paires) == 0 {
		t.Fatalf("aucun couple (film, carte) dans %s — le contrôle ne contrôlerait rien", pairesPath)
	}
	t.Logf("CONTRÔLE : %d couples (film, carte) · catalogue %s · %d cartes catalogué(es)",
		len(paires), boundsPath, len(cat.Maps))
	return paires, root, cat
}

// ctrlLisPaires lit un CSV `film,carte` (en-tête tolérée).
func ctrlLisPaires(path string) ([]ctrlPaire, error) {
	f, err := os.Open(path) //nolint:gosec // chemin d'instrument, lecture seule
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	lignes, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	var out []ctrlPaire
	for _, l := range lignes {
		if len(l) < 2 {
			continue
		}
		film, carte := strings.TrimSpace(l[0]), strings.TrimSpace(l[1])
		if film == "" || carte == "" || strings.EqualFold(film, "film") {
			continue
		}
		out = append(out, ctrlPaire{film: film, carte: carte})
	}
	return out, nil
}

// ctrlPublie publie le bilan par carte puis le total. Les dénominateurs sont toujours là :
// un taux d'accord sans le nombre de films contrôlés ne vaut rien.
func ctrlPublie(t *testing.T, bilans map[string]*ctrlBilan, sansEntree []string, nPaires int) {
	t.Helper()
	cartes := make([]string, 0, len(bilans))
	for c := range bilans {
		cartes = append(cartes, c)
	}
	sort.Strings(cartes)
	var totAccord, totDesaccord, totIllisible, cartesVertes int
	for _, c := range cartes {
		b := bilans[c]
		totAccord += b.accord
		totDesaccord += b.desaccord
		totIllisible += b.illisible
		verdict := "ACCORD"
		switch {
		case b.desaccord > 0:
			verdict = "DÉSACCORD"
		case b.accord == 0:
			verdict = "NON CONTRÔLÉE (aucun découpage lisible)"
		default:
			cartesVertes++
		}
		t.Logf("%-38s attendu %v · accord %d/%d · illisible %d · lectures %s -> %s",
			c, b.attendu, b.accord, b.accord+b.desaccord, b.illisible, ctrlLectures(b), verdict)
	}
	if len(sansEntree) > 0 {
		t.Logf("couples dont la carte est HORS CATALOGUE (non contrôlés) : %d — %s",
			len(sansEntree), strings.Join(sansEntree, ", "))
	}
	t.Logf("TOTAL : %d couples · %d cartes · %d vertes · accord %d · désaccord %d · illisible %d",
		nPaires, len(cartes), cartesVertes, totAccord, totDesaccord, totIllisible)
}

// ctrlLectures rend les découpages effectivement lus et leur effectif, du plus fréquent au
// moins fréquent : une carte dont les films ne s'accordent pas entre eux se voit ici.
func ctrlLectures(b *ctrlBilan) string {
	type kv struct {
		w [3]uint
		n int
	}
	l := make([]kv, 0, len(b.lus))
	for w, n := range b.lus {
		l = append(l, kv{w, n})
	}
	if len(l) == 0 {
		return "(aucune)"
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	parts := make([]string, 0, len(l))
	for _, e := range l {
		parts = append(parts, fmt.Sprintf("%vx%d", e.w, e.n))
	}
	return strings.Join(parts, " ")
}
