package replay

// assaut_a8_ancrage_test.go — POURQUOI `ti=13` NE SE LIT PAS EN ASSAUT. Le diagnostic, avant
// tout correctif.
//
// # Ce qui est etabli, et ce qui ne l'est pas
//
// La phase A7 a mesure que le canal des proprietes d'objet gere est ILLISIBLE en Assaut :
// chainage 1,9 a 16,4 % contre 87 a 99 % sur un KOTH de reference. Les 8 slots `ti=13` sont
// pourtant bien la, a chaque film. Ce qui n'est PAS etabli, c'est OU la lecture casse — et sans
// ca, tout correctif serait un coup dans le noir.
//
// # Les quatre compteurs disent OU ca casse, et ils sont deja publies
//
//	Records   records ANCRES dans la bande de slots — si ce nombre s'effondre, c'est la BANDE
//	          ou l'ancrage d'en-tete qui est faux ;
//	Walked    marches ABOUTIES — si Records est haut et Walked bas, la marche bute sur un
//	          composant (grammaire, largeur, archetype) ;
//	Broken    marches interrompues, le complement de Walked ;
//	Chained   marches dont la position de FIN porte un en-tete de record valide — si Walked est
//	          haut et Chained bas, la marche « aboutit » mais s'arrete au MAUVAIS ENDROIT : les
//	          largeurs sont fausses quelque part au milieu.
//
// Les trois diagnostics sont mutuellement exclusifs, et chacun designe un correctif different.
// Cet instrument les met cote a cote, Assaut contre temoins d'autres modes.
//
// # LE TEMOIN EST OBLIGATOIRE
//
// Un chainage bas ne veut rien dire dans l'absolu — il faut le meme balayage sur un mode ou la
// lecture est SAINE, sur cette machine, avec ce code. Les temoins sont pris hors du corpus
// d'Assaut : KOTH, Strongholds, CTF.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run AssautA8Ancrage -v -timeout 60m

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// a8Temoins : des films d'AUTRES modes, ou `ti=13` est le canal de production des zones et des
// collines. Ils donnent l'echelle a laquelle lire les chiffres d'Assaut.
var a8Temoins = []struct{ id, mode string }{
	{"7f1bbf06", "KOTH:Arena"},
	{"21ece4d8", "KOTH:Arena"},
	{"0a247154", "Ranked:King of the Hill"},
	{"2ce58582", "Ranked:Strongholds"},
	{"696a9d7c", "Strongholds:Arena"},
	{"cde26226", "CTF:Arena"},
}

// TestAssautA8Ancrage imprime les quatre compteurs, Assaut contre temoins.
func TestAssautA8Ancrage(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA8Ancrage")()

	ligne := func(id, mode string) {
		sc, err := filmdec.ScanFilmManagedProperties(filepath.Join(cache, "film_chunks", id))
		if err != nil {
			t.Logf("%-9s %-26s balayage impossible : %v", id, mode, err)
			return
		}
		pct := func(a, b int) float64 {
			if b == 0 {
				return 0
			}
			return 100 * float64(a) / float64(b)
		}
		t.Logf("%-9s %-26s slots=%-3d records=%-6d marches=%-6d (%.0f %%) brisees=%-6d "+
			"chainees=%-6d (%.1f %%) lectures=%d",
			id, mode, sc.Slots, sc.Records, sc.Walked, pct(sc.Walked, sc.Records), sc.Broken,
			sc.Chained, pct(sc.Chained, sc.Walked), len(sc.Reads))
	}

	t.Log("=== TEMOINS (modes ou ti=13 est le canal de production)")
	for _, w := range a8Temoins {
		ligne(w.id, w.mode)
	}

	t.Log("=== ASSAUT")
	films := make([]string, 0, len(a5Explosions))
	for id := range a5Explosions {
		films = append(films, id)
	}
	sort.Strings(films)
	for _, id := range films {
		ligne(id, "Assaut")
	}
}

// TestAssautA8Densite — L'HYPOTHESE QUI RENVERSE LE DIAGNOSTIC DE LA PHASE A3.
//
// `Chained` est FAUX PAR CONSTRUCTION pour le DERNIER record d'un paquet : rien ne peut le
// suivre, donc sa position de fin ne porte aucun en-tete. Son en-tete de champ le dit
// (`zone_state_scan.go` : « Faux pour le dernier record d'un paquet »), et l'estime a ~3 % de
// perte — sur un canal DENSE.
//
// Or l'Assaut a la bande la plus MAIGRE du corpus : 8 slots, contre 26 a 52 en Strongholds et
// KOTH. Si ses paquets ne portent qu'un ou deux records `ti=13`, alors la PLUPART des records y
// sont le dernier de leur paquet, et le chainage s'effondre SANS QU'AUCUNE GRAMMAIRE NE SOIT
// FAUSSE. Le « 1,9 a 16,4 % » de la phase A3 mesurerait alors la DENSITE, pas la contamination.
//
// LE CRITERE, ecrit avant la mesure : si le chainage suit la densite de records par paquet sur
// TOUS les modes — temoins compris —, alors c'est une mesure de densite et elle ne dit rien de
// la justesse de la lecture. Si l'Assaut se detache de la droite, le diagnostic A3 tient.
func TestAssautA8Densite(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA8Densite")()

	ligne := func(id, mode string) {
		dir := filepath.Join(cache, "film_chunks", id)
		sc, err := filmdec.ScanFilmManagedProperties(dir)
		if err != nil {
			t.Logf("%-9s %-26s balayage impossible : %v", id, mode, err)
			return
		}
		paquets := 0
		for c := 1; c <= filmdec.CountFilmChunks(dir); c++ {
			data, err := filmdec.ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range filmdec.WalkPackets(data) {
				if pk.Type == filmdec.PacketTypeDelta {
					paquets++
				}
			}
		}
		densite, chainage := 0.0, 0.0
		if paquets > 0 {
			densite = float64(sc.Records) / float64(paquets)
		}
		if sc.Walked > 0 {
			chainage = 100 * float64(sc.Chained) / float64(sc.Walked)
		}
		t.Logf("%-9s %-26s slots=%-4d paquets=%-7d records=%-7d densite=%.3f rec/paquet  "+
			"chainage=%.1f %%", id, mode, sc.Slots, paquets, sc.Records, densite, chainage)
	}

	t.Log("=== TEMOINS")
	for _, w := range a8Temoins {
		ligne(w.id, w.mode)
	}
	t.Log("=== ASSAUT")
	films := make([]string, 0, len(a5Explosions))
	for id := range a5Explosions {
		films = append(films, id)
	}
	sort.Strings(films)
	for _, id := range films {
		ligne(id, "Assaut")
	}
	t.Log("LECTURE : si le chainage suit la densite sur TOUS les modes, il mesure la densite et " +
		"non la justesse — et le diagnostic « contamination d'ancrage » de la phase A3 tombe.")
}

// TestAssautA8Contenu — CE QUE PORTENT VRAIMENT LES 8 SLOTS `ti=13` D'UN FILM D'ASSAUT.
//
// La densite ayant montre que `Chained` ne mesure pas la justesse (cf. [TestAssautA8Densite]),
// le canal redevient lisible : ces lectures ne sont plus suspectes, elles sont RARES. Reste a
// voir ce qu'elles disent. Ce dump est le premier regard porte sur le contenu de `ti=13` en
// Assaut — par slot, par tag, avec l'etendue des valeurs et la fenetre de temps.
//
// LE POINT DE COMPARAISON EST DANS LE MEME LOG : un temoin de Strongholds, ou le tag 3 est la
// JAUGE de capture et le tag 4 le PROPRIETAIRE (semantique etablie ailleurs, cf. l'en-tete de
// `zone_state_scan.go`). Ce qui ressemble a une jauge en Assaut se reconnaitra a sa forme.
func TestAssautA8Contenu(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA8Contenu")()

	dump := func(id, mode string, exps []int) {
		sc, err := filmdec.ScanFilmManagedProperties(filepath.Join(cache, "film_chunks", id))
		if err != nil {
			t.Logf("%s : %v", id, err)
			return
		}
		type stat struct {
			n              int
			min, max       uint64
			tmin, tmax     int
			distinct       map[uint64]bool
			champScalaire  int
			champParJoueur int
		}
		st := map[a7Cle]*stat{}
		for _, r := range sc.Reads {
			if !r.HasValue {
				continue
			}
			k := a7Cle{slot: r.Slot, tag: r.Tag}
			s := st[k]
			if s == nil {
				s = &stat{min: r.Value, max: r.Value, tmin: int(r.TimestampUS / 1000),
					tmax: int(r.TimestampUS / 1000), distinct: map[uint64]bool{}}
				st[k] = s
			}
			s.n++
			s.distinct[r.Value] = true
			if r.Value < s.min {
				s.min = r.Value
			}
			if r.Value > s.max {
				s.max = r.Value
			}
			if ms := int(r.TimestampUS / 1000); ms < s.tmin {
				s.tmin = ms
			} else if ms > s.tmax {
				s.tmax = ms
			}
			if r.FilmIndex < 0 {
				s.champScalaire++
			} else {
				s.champParJoueur++
			}
		}
		cles := make([]a7Cle, 0, len(st))
		for k := range st {
			cles = append(cles, k)
		}
		sort.Slice(cles, func(i, j int) bool {
			if cles[i].slot != cles[j].slot {
				return cles[i].slot < cles[j].slot
			}
			return cles[i].tag < cles[j].tag
		})
		t.Logf("--- %s (%s) : %d couples porteurs de valeur ; explosions %v", id, mode, len(cles), exps)
		for _, k := range cles {
			s := st[k]
			t.Logf("    %-16s n=%-5d valeurs [%d..%d] %d distinctes  t=[%d..%d] ms  "+
				"scalaire=%d parJoueur=%d",
				k, s.n, s.min, s.max, len(s.distinct), s.tmin, s.tmax, s.champScalaire, s.champParJoueur)
		}
	}

	dump("2ce58582", "Ranked:Strongholds — TEMOIN, tag 3 = jauge, tag 4 = proprietaire", nil)
	for _, id := range []string{"9f57c612", "c75f33b8", "df8fcbef"} {
		dump(id, "Assaut One Bomb", a5Explosions[id])
	}
}
