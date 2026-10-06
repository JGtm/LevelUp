//go:build research

package replay

// rejeu_bouche_trou_research_test.go — INSTRUMENT de l'etape G4 du lot « toute entree du roster a
// l'equipe que le film ecrit » (`.ai/PLAN_REJEU_EQUIPES_SOURCE_2026-10-06.md`). Lecture seule : faits
// persistes et artefacts publies du parc, jamais un film.
//
// L'HYPOTHESE MESUREE (utilisateur, 2026-10-06) : quand une place est sans joueur humain — entre le
// depart d'un partant et l'arrivee du suivant, ou avant l'arrivee de son premier occupant — un bot
// la tient « en attendant ».
//
// LES PLACES sont les sieges que l'artefact publie pour ses entrees HUMAINES (les bots sans equipe y
// sont sur leur index, donc hors de toute place) : chaque intervalle de presence est une occupation.
// UN INTERVALLE VIDE va de la fin CERTAINE d'une occupation (`to`) a la veille du debut de la suivante
// sur la meme place (`relais`), ou de la frame 0 a la veille du premier occupant arrive apres elle
// (`debut`) ; une place dont le dernier occupant part avant la fin sans successeur se compte a part
// (`fin`). Il est COUVERT par les declarations BOT_METADATA (frames exactes) qui le recoupent, de
// n'importe quel bot : entierement, en partie, ou pas du tout.
//
// LE TROISIEME CONTROLE DU CHAMP : pour chaque bot qui recoupe un intervalle, l'equipe que lit le
// champ de BOT_METADATA (octet 0x783 apres le nom, releve par l'instrument de la couche des faits,
// fichier `RJE_CHAMP`) contre l'equipe de la place (celle de ses occupants humains).
//
// « PAS ENCORE APPARU » : pour chaque intervalle de presence publie d'une entree, l'ecart entre son
// debut et le debut de la premiere vie publiee qu'elle porte dans cet intervalle ; aucune vie = jamais
// apparu. Au coup d'envoi (presence des la frame 0) ou en cours de match.
//
//	RJE_FAITS=... RJE_ARTEFACTS=... RJE_CATALOGUE=... RJE_CHAMP=<g2c_botmeta.txt> RJE_SORTIE=<tsv> \
//	  go test -tags research -count=1 -run '^TestRJEHypotheseBoucheTrou$' -v ./internal/games/halo_infinite/film/replay/

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
)

// rjeIntervalle : une place sans occupant humain, en frames du document, bornes incluses.
type rjeIntervalle struct {
	film, genre      string
	place            int
	equipe           *int // equipe de la place ; nil quand ses occupants humains ne s'accordent pas
	de, a            int
	partant, suivant string
	couvert          int
	bots             []string
	// departMax : la derniere frame ou le partant PEUT etre encore la (sa fin affichee : la veille de
	// l'image-cle qui prouve son absence) ; iv.de - 1 sans partant (intervalle `debut`).
	departMax int
	// arriveeMin : la premiere frame ou le suivant PEUT etre deja la (le lendemain de la derniere
	// image-cle porteuse qui precede son debut publie) ; iv.a + 1 sans suivant (intervalle `fin`).
	arriveeMin int
	// filmAvecBots : le film declare au moins un bot (BOT_METADATA).
	filmAvecBots bool
	// premiere / derniere : les bornes de la couverture quand elle est d'un seul tenant ; contigue dit
	// qu'elle l'est.
	premiere, derniere int
	contigue           bool
}

// rjeDeclaration : une declaration BOT_METADATA d'un bot, en frames, bornes incluses.
type rjeDeclaration struct {
	bot   string
	de, a int
}

// rjeApparition : un intervalle de presence publie et le debut de sa premiere vie (-1 : aucune).
type rjeApparition struct {
	film, nom    string
	bot          bool
	de, a, corps int
}

// rjeOccupation : une occupation humaine d'une place (aMax : sa fin affichee).
type rjeOccupation struct {
	nom         string
	de, a, aMax int
	equipe      *int
}

// rjeChampLu lit, dans la sortie de l'instrument de la couche des faits, l'equipe que le champ donne a
// chaque bot : cle `film/nom [bot]`.
func rjeChampLu(t *testing.T, chemin string) map[string]int {
	t.Helper()
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("champ : %v", err)
	}
	re := regexp.MustCompile(`CHAMP (\S+) "([^"]+)".*-> equipe lue (\S+)\s*$`)
	out := map[string]int{}
	for _, ligne := range strings.Split(string(blob), "\n") {
		m := re.FindStringSubmatch(strings.TrimRight(ligne, "\r"))
		if m == nil {
			continue
		}
		if v, err := strconv.Atoi(m[3]); err == nil {
			out[m[1]+"/"+m[2]+killsource.BotSuffix] = v
		}
	}
	return out
}

// occupationsHumaines rend, par place publiee, les occupations humaines triees, et le nombre
// d'entrees humaines presentes que l'artefact laisse sans place.
func (r *rjeFilm) occupationsHumaines() (map[int][]rjeOccupation, int) {
	parPlace, sansPlace := map[int][]rjeOccupation{}, 0
	for _, e := range r.doc.Roster {
		if e.Bot || len(e.Presence) == 0 {
			continue
		}
		if e.SeatSource == SeatSourceIndex {
			sansPlace++
			continue
		}
		for _, p := range e.Presence {
			aMax := p.To
			if p.ToMax != nil {
				aMax = *p.ToMax
			}
			parPlace[e.Seat] = append(parPlace[e.Seat], rjeOccupation{e.Name, p.From, p.To, aMax, e.Team})
		}
	}
	for s := range parPlace {
		slices.SortFunc(parPlace[s], func(x, y rjeOccupation) int { return cmp.Compare(x.de, y.de) })
	}
	return parPlace, sansPlace
}

// equipeDesOccupants rend l'equipe commune des occupations, nil si elles se contredisent ou se taisent.
func equipeDesOccupants(occs []rjeOccupation) *int {
	var eq *int
	for _, o := range occs {
		switch {
		case o.equipe == nil:
		case eq == nil:
			eq = o.equipe
		case *eq != *o.equipe:
			return nil
		}
	}
	return eq
}

// intervallesVides rend les intervalles vides de chaque place (cf. l'en-tete).
func (r *rjeFilm) intervallesVides() ([]rjeIntervalle, int) {
	parPlace, sansPlace := r.occupationsHumaines()
	places := make([]int, 0, len(parPlace))
	for s := range parPlace {
		places = append(places, s)
	}
	slices.Sort(places)
	var out []rjeIntervalle
	for _, s := range places {
		occs := parPlace[s]
		base := rjeIntervalle{film: r.id, place: s, equipe: equipeDesOccupants(occs), filmAvecBots: len(r.bots) > 0}
		if occs[0].de > 0 {
			iv := base
			iv.genre, iv.de, iv.a, iv.suivant = "debut", 0, occs[0].de-1, occs[0].nom
			iv.departMax, iv.arriveeMin = -1, r.arriveeAuPlusTot(occs[0].de)
			out = append(out, iv)
		}
		partant := occs[0]
		for _, o := range occs[1:] {
			if o.de > partant.a+1 {
				iv := base
				iv.genre, iv.de, iv.a, iv.partant, iv.suivant = "relais", partant.a+1, o.de-1, partant.nom, o.nom
				iv.departMax, iv.arriveeMin = partant.aMax, r.arriveeAuPlusTot(o.de)
				out = append(out, iv)
			}
			if o.a > partant.a {
				partant = o
			}
		}
		if partant.a < r.h.frames-1 {
			iv := base
			iv.genre, iv.de, iv.a, iv.partant = "fin", partant.a+1, r.h.frames-1, partant.nom
			iv.departMax, iv.arriveeMin = partant.aMax, r.h.frames
			out = append(out, iv)
		}
	}
	return out, sansPlace
}

// arriveeAuPlusTot rend la premiere frame ou un occupant publie a partir de la frame `de` PEUT deja etre
// la : le lendemain de la derniere image-cle porteuse anterieure a `de` (0 sans elle). Une presence lue
// a une image-cle ne date l'arrivee qu'a l'image-cle pres.
func (r *rjeFilm) arriveeAuPlusTot(de int) int {
	out := 0
	for k := range r.scan.KeyframesUS {
		if f := r.rjeFrameKF(k); f < de && f+1 > out {
			out = f + 1
		}
	}
	return out
}

// compatibleAvecUnePlaceJamaisVide dit qu'un intervalle couvert d'un seul tenant l'est peut-etre
// ENTIEREMENT, aux bornes humaines pres : le partant peut etre reste jusqu'a l'arrivee du bot (le bot
// arrive au plus tard le lendemain de sa fin affichee) et le suivant peut etre arrive au depart du bot
// (le bot est encore declare a la derniere image-cle ou le suivant n'est pas lu).
func compatibleAvecUnePlaceJamaisVide(iv rjeIntervalle) bool {
	if iv.couvert == 0 || !iv.contigue {
		return false
	}
	return iv.premiere <= max(iv.departMax+1, iv.de) && iv.derniere+1 >= min(iv.arriveeMin, iv.a+1)
}

// declarationsDuFilm rend les declarations de tous les bots du film, en frames.
func (r *rjeFilm) declarationsDuFilm() []rjeDeclaration {
	var out []rjeDeclaration
	for _, b := range r.bots {
		for _, d := range r.declarationsEnFrames(RosterEntry{Name: b.Name, FilmIndex: b.FilmIndex, Bot: true}) {
			out = append(out, rjeDeclaration{bot: b.Name, de: d[0], a: d[1]})
		}
	}
	return out
}

// couvrir pose sur l'intervalle les frames que les declarations recoupent (union) et les bots.
func couvrir(iv *rjeIntervalle, decls []rjeDeclaration) {
	var morceaux [][2]int
	vus := map[string]bool{}
	for _, d := range decls {
		de, a := max(d.de, iv.de), min(d.a, iv.a)
		if de > a {
			continue
		}
		morceaux = append(morceaux, [2]int{de, a})
		if !vus[d.bot] {
			vus[d.bot] = true
			iv.bots = append(iv.bots, d.bot)
		}
	}
	slices.SortFunc(morceaux, func(x, y [2]int) int { return cmp.Or(cmp.Compare(x[0], y[0]), cmp.Compare(x[1], y[1])) })
	fin, blocs := iv.de-1, 0
	for _, m := range morceaux {
		if m[1] <= fin {
			continue
		}
		if m[0] > fin+1 || blocs == 0 {
			blocs++
		}
		if blocs == 1 && iv.couvert == 0 {
			iv.premiere = m[0]
		}
		iv.couvert += m[1] - max(m[0], fin+1) + 1
		fin = m[1]
	}
	iv.derniere, iv.contigue = fin, blocs == 1
}

// apparitions rend, pour chaque intervalle de presence publie et chaque declaration d'un bot hors du
// roster, le debut de la premiere vie publiee de l'occupant dans l'intervalle.
func (r *rjeFilm) apparitions() []rjeApparition {
	vies := viesParIdentite(r.doc.Tracks)
	var out []rjeApparition
	presents := map[string]bool{}
	ajouter := func(nom, cle string, bot bool, de, a int) {
		corps := -1
		for _, v := range vies[cle] {
			if v[1] >= de && v[0] <= a {
				corps = max(v[0], de)
				break
			}
		}
		out = append(out, rjeApparition{film: r.id, nom: nom, bot: bot, de: de, a: a, corps: corps})
	}
	for _, e := range r.doc.Roster {
		presents[cleDeRoster(e)] = true
		for _, p := range e.Presence {
			ajouter(e.Name, cleDeRoster(e), e.Bot, p.From, p.To)
		}
	}
	for _, d := range r.declarationsDuFilm() {
		if cle := botIdentityKey(d.bot); !presents[cle] {
			ajouter(d.bot, cle, true, max(d.de, 0), d.a)
		}
	}
	return out
}

func TestRJEHypotheseBoucheTrou(t *testing.T) {
	faitsDir, artDir := rjeEnv(t, "RJE_FAITS"), rjeEnv(t, "RJE_ARTEFACTS")
	catalogue, sortie := rjeEnv(t, "RJE_CATALOGUE"), rjeEnv(t, "RJE_SORTIE")
	champ := rjeChampLu(t, rjeEnv(t, "RJE_CHAMP"))
	ents, err := os.ReadDir(artDir)
	if err != nil {
		t.Fatalf("artefacts : %v", err)
	}
	var ivs []rjeIntervalle
	var apps []rjeApparition
	films, humainsSansPlace := 0, 0
	for _, de := range ents {
		nom := de.Name()
		if !strings.HasSuffix(nom, ".json") || strings.HasSuffix(nom, ".derived.json") {
			continue
		}
		id := strings.TrimSuffix(nom, ".json")
		if _, err := os.Stat(filepath.Join(faitsDir, id+".filmfacts.bin")); err != nil {
			continue
		}
		r := rjeCharger(t, id, faitsDir, artDir, catalogue)
		films++
		decls := r.declarationsDuFilm()
		vides, sansPlace := r.intervallesVides()
		humainsSansPlace += sansPlace
		for i := range vides {
			couvrir(&vides[i], decls)
		}
		ivs = append(ivs, vides...)
		apps = append(apps, r.apparitions()...)
	}
	t.Logf("%d films ; %d entrees humaines presentes sans place (exclues des places)", films, humainsSansPlace)
	rjeRapporterLesIntervalles(t, ivs)
	rjeTroisiemeControle(t, ivs, champ)
	rjeRapporterLesApparitions(t, apps)
	rjeEcrireLesIntervalles(t, sortie, ivs, champ)
}

// rjeMediane rend la mediane et le maximum d'une liste de durees en frames (0, 0 si vide).
func rjeMediane(v []int) (mediane, maxi int) {
	if len(v) == 0 {
		return 0, 0
	}
	s := slices.Clone(v)
	slices.Sort(s)
	return s[len(s)/2], s[len(s)-1]
}

// rjeRapporterLesIntervalles publie, par genre, le total et la couverture par les bots declares.
func rjeRapporterLesIntervalles(t *testing.T, ivs []rjeIntervalle) {
	t.Helper()
	for _, genre := range []string{"relais", "debut", "fin"} {
		for _, avecBots := range []bool{true, false} {
			rjeRapporterUnGenre(t, ivs, genre, avecBots)
		}
	}
}

// rjeRapporterUnGenre publie un genre d'intervalle, dans les films avec ou sans bot declare.
func rjeRapporterUnGenre(t *testing.T, ivs []rjeIntervalle, genre string, avecBots bool) {
	t.Helper()
	total, entier, partiel, aucun, compatibles := 0, 0, 0, 0, 0
	var nonCouverts, durees, restes []int
	for _, iv := range ivs {
		if iv.genre != genre || iv.filmAvecBots != avecBots {
			continue
		}
		n := iv.a - iv.de + 1
		total++
		durees = append(durees, n)
		switch {
		case iv.couvert == n:
			entier++
		case iv.couvert > 0:
			partiel++
			restes = append(restes, n-iv.couvert)
			if compatibleAvecUnePlaceJamaisVide(iv) {
				compatibles++
			}
		default:
			aucun++
			nonCouverts = append(nonCouverts, n)
		}
	}
	if total == 0 {
		return
	}
	md, mx := rjeMediane(durees)
	mnc, xnc := rjeMediane(nonCouverts)
	mr, xr := rjeMediane(restes)
	t.Logf("INTERVALLES %-6s films avec bots=%v : %4d (duree mediane %d f, max %d f) | entierement couverts %d | "+
		"en partie %d, dont %d compatibles avec une place jamais vide (part non couverte mediane %d f, max %d f) | "+
		"pas du tout %d (duree mediane %d f, max %d f)", genre, avecBots, total, md, mx, entier, partiel, compatibles,
		mr, xr, aucun, mnc, xnc)
}

// rjeTroisiemeControle confronte, pour chaque bot qui recoupe un intervalle, l'equipe que lit le champ
// a l'equipe de la place. Un bot qui recoupe plusieurs intervalles est AMBIGU : compte a part.
func rjeTroisiemeControle(t *testing.T, ivs []rjeIntervalle, champ map[string]int) {
	t.Helper()
	equipes, nombre := map[string]map[int]bool{}, map[string]int{}
	for _, iv := range ivs {
		for _, b := range iv.bots {
			cle := iv.film + "/" + b
			nombre[cle]++
			if equipes[cle] == nil {
				equipes[cle] = map[int]bool{}
			}
			if iv.equipe != nil {
				equipes[cle][*iv.equipe] = true
			}
		}
	}
	cles := make([]string, 0, len(nombre))
	for c := range nombre {
		cles = append(cles, c)
	}
	slices.Sort(cles)
	uAccord, uDesaccord, mTous, mUn, mAucun, inconnus, sansEquipe := 0, 0, 0, 0, 0, 0, 0
	for _, c := range cles {
		lu, ok := champ[c]
		eqs := equipes[c]
		switch {
		case !ok:
			inconnus++
		case len(eqs) == 0:
			sansEquipe++
		case nombre[c] == 1 && eqs[lu]:
			uAccord++
		case nombre[c] == 1:
			uDesaccord++
			t.Logf("  DESACCORD %s : champ %d, place d'equipe %v", c, lu, eqs)
		case eqs[lu] && len(eqs) == 1:
			mTous++
		case eqs[lu]:
			mUn++
		default:
			mAucun++
			t.Logf("  DESACCORD %s (plusieurs intervalles) : champ %d, places d'equipes %v", c, lu, eqs)
		}
	}
	t.Logf("TROISIEME CONTROLE (par bot) : %d bots recoupent au moins un intervalle ; un seul intervalle : %d accords, "+
		"%d desaccords ; plusieurs : %d d'une seule equipe egale au champ, %d dont une des equipes est le champ, %d "+
		"sans l'equipe du champ ; %d champ inconnu, %d places sans equipe", len(cles), uAccord, uDesaccord, mTous, mUn,
		mAucun, inconnus, sansEquipe)
}

// rjeRapporterLesApparitions publie les situations « pas encore apparu », humains et bots, au coup
// d'envoi et en cours de match.
func rjeRapporterLesApparitions(t *testing.T, apps []rjeApparition) {
	t.Helper()
	for _, bot := range []bool{false, true} {
		for _, debut := range []bool{true, false} {
			total, jamais, enRetard, plusDe5s := 0, 0, 0, 0
			var retards []int
			for _, a := range apps {
				if a.bot != bot || (a.de == 0) != debut {
					continue
				}
				total++
				switch {
				case a.corps < 0:
					jamais++
				case a.corps > a.de:
					enRetard++
					retards = append(retards, a.corps-a.de)
					if a.corps-a.de >= 50 {
						plusDe5s++
					}
				}
			}
			m, x := rjeMediane(retards)
			t.Logf("PAS ENCORE APPARU bot=%v coup d'envoi=%v : %d presences, %d sans corps au debut (mediane %d f, "+
				"max %d f, >= 5 s : %d), %d sans aucun corps", bot, debut, total, enRetard, m, x, plusDe5s, jamais)
		}
	}
}

// rjeEcrireLesIntervalles ecrit un intervalle par ligne, ses bots et l'equipe que le champ leur lit.
func rjeEcrireLesIntervalles(t *testing.T, chemin string, ivs []rjeIntervalle, champ map[string]int) {
	t.Helper()
	var b strings.Builder
	b.WriteString("film\tgenre\tplace\tequipe\tde\ta\tduree\tcouvert\tbots(equipe lue)\tpartant\tsuivant\n")
	for _, iv := range ivs {
		eq := "?"
		if iv.equipe != nil {
			eq = strconv.Itoa(*iv.equipe)
		}
		bots := make([]string, 0, len(iv.bots))
		for _, n := range iv.bots {
			lu := "?"
			if v, ok := champ[iv.film+"/"+n]; ok {
				lu = strconv.Itoa(v)
			}
			bots = append(bots, fmt.Sprintf("%s(%s)", n, lu))
		}
		fmt.Fprintf(&b, "%s\t%s\t%d\t%s\t%d\t%d\t%d\t%d\t%s\t%s\t%s\n", iv.film, iv.genre, iv.place, eq, iv.de, iv.a,
			iv.a-iv.de+1, iv.couvert, strings.Join(bots, ","), iv.partant, iv.suivant)
	}
	if err := os.WriteFile(chemin, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("sortie : %v", err)
	}
}
