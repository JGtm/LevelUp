//go:build research

package replay

// rejeu_equipes_research_test.go — INSTRUMENT du lot « toute entree du roster a l'equipe que le film
// ecrit » (2026-10-06, `.ai/V7.5/PLAN_REJEU_EQUIPES_SOURCE_2026-10-06.md`, etape G1). Lecture seule : le
// fichier de faits persiste d'un film et son artefact publie, jamais le film.
//
// Pour chaque entree du roster publie sans place (`seatSource: index`), il rend ce que la liaison
// aux entites `ti=9` a vu : les declarations BOT_METADATA (en frames du document), les images-cles
// porteuses qui l'encadrent, les entites de son index (slot, designateur, fenetre, a qui elles sont
// liees), la table de controle par index, ses vies, puis les places de la table libres pendant sa
// presence. La liaison est REJOUEE par le code de production (`lierLesOccupants`) sur les faits et
// le roster de l'artefact, et l'instrument verifie d'abord qu'elle rend les equipes publiees.
//
//	RJE_FAITS=<data>/cache/film_facts/halo_infinite RJE_ARTEFACTS=<data>/cache/replays/halo_infinite \
//	RJE_CATALOGUE=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json RJE_FILMS=43716616,... \
//	  go test -tags research -count=1 -run '^TestRJEDiagnosticDesEntreesIndex$' -v ./internal/games/halo_infinite/film/replay/

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rjeFilm : ce que l'instrument relit d'un film.
type rjeFilm struct {
	id     string
	faits  *FilmFactsFile
	doc    ReplayDocument
	h      replayClock
	bots   []BotIdentity
	occ    occupants
	scan   grammar.PlayerEntityScan
	equipe map[int]int // table de controle par index
}

func rjeEnv(t *testing.T, cle string) string {
	t.Helper()
	v := os.Getenv(cle)
	if v == "" {
		t.Skipf("%s absent : instrument saute", cle)
	}
	return v
}

// rjeEntree rend l'entree de catalogue sous laquelle les faits se relisent (meme regle que la sonde P4).
func rjeEntree(t *testing.T, blob []byte, catalogue string) profile.MapQuantEntry {
	t.Helper()
	ent, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("en-tete : %v", err)
	}
	if ent.LayoutDetected {
		return profile.MapQuantEntry{Module: ent.MapModule}
	}
	cat, err := profile.LoadMapQuantCatalog(catalogue)
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	for _, e := range cat.Maps {
		if e.Module == ent.MapModule && e.AxisWidths == ent.AxisW {
			return e
		}
	}
	return profile.MapQuantEntry{Module: ent.MapModule, AxisWidths: ent.AxisW}
}

// rjeBots projette le roster de bots de killsource comme `replayidentity.BotIdentities`.
func rjeBots(res *killsource.Result) []BotIdentity {
	if res == nil {
		return nil
	}
	nonEpingles := map[int]bool{}
	for _, b := range res.Roster.UnpinnedBots {
		nonEpingles[b.BotID] = true
	}
	var out []BotIdentity
	for _, b := range res.Roster.Bots {
		if nonEpingles[b.BotID] || b.Name == "" {
			continue
		}
		id := BotIdentity{FilmIndex: b.Slot, Name: b.Name + killsource.BotSuffix, BotID: b.BotID}
		if b.Team != nil {
			equipe := *b.Team
			id.Team = &equipe
		}
		for _, d := range b.Declarations {
			id.Declarations = append(id.Declarations, [2]uint64{d.FromUS, d.ToUS})
		}
		out = append(out, id)
	}
	return out
}

// rjeCharger relit les faits et l'artefact d'un film et rejoue la liaison aux entites.
func rjeCharger(t *testing.T, id, faitsDir, artDir, catalogue string) *rjeFilm {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join(faitsDir, id+".filmfacts.bin"))
	if err != nil {
		t.Fatalf("%s faits : %v", id, err)
	}
	f, err := DecodeFilmFactsFile(blob, rjeEntree(t, blob, catalogue))
	if err != nil {
		t.Fatalf("%s faits : %v", id, err)
	}
	art, err := os.ReadFile(filepath.Join(artDir, id+".json"))
	if err != nil {
		t.Fatalf("%s artefact : %v", id, err)
	}
	r := &rjeFilm{id: id, faits: f}
	if err := json.Unmarshal(art, &r.doc); err != nil {
		t.Fatalf("%s artefact : %v", id, err)
	}
	in := f.Facts.FilmInputs
	var origine uint64
	for i, p := range in.Positions {
		if i == 0 || p.TimestampUS < origine {
			origine = p.TimestampUS
		}
	}
	r.h = replayClock{origin: origine, step: uint64(r.doc.FrameIntervalMS) * 1000, frames: r.doc.FrameCount}
	r.bots = rjeBots(f.Kills)
	r.scan = in.PlayerEntities
	r.equipe = in.PlayerTeams
	roster := append([]RosterEntry(nil), r.doc.Roster...)
	r.occ = lierLesOccupants(roster, r.doc.Tracks, entreesDesOccupants{scan: r.scan, bots: r.bots,
		horloge: r.h, parIndex: r.equipe})
	for i, e := range r.doc.Roster {
		if !memeEquipePubliee(e.Team, r.occ.parEntree[i].equipe) {
			t.Errorf("%s : la liaison rejouee ne rend pas l'equipe publiee de %q (%v / %v) — instrument faux",
				id, e.Name, e.Team, r.occ.parEntree[i].equipe)
		}
	}
	return r
}

func memeEquipePubliee(a, b *int) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

func rjeEquipe(t *int) string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprint(*t)
}

// rjeFrameKF rend la frame (non bornee) d'une image-cle porteuse.
func (r *rjeFilm) rjeFrameKF(rang int) int {
	us, _ := r.scan.KeyframeUS(rang)
	return frameBrute(r.h, us)
}

// declarationsEnFrames rend les declarations d'un bot en frames `[de, a]` (a = derniere frame declaree).
func (r *rjeFilm) declarationsEnFrames(e RosterEntry) [][2]int {
	var out [][2]int
	for _, d := range declarationsDe(e, r.bots) {
		de := frameBrute(r.h, d[0])
		a := r.h.frames - 1
		if d[1] != 0 {
			a = frameBrute(r.h, d[1]) - 1
		}
		out = append(out, [2]int{de, a})
	}
	return out
}

// imagesClesAutour rend la derniere image-cle porteuse <= f et la premiere >= g, en frames.
func (r *rjeFilm) imagesClesAutour(f, g int) (avant, apres int) {
	avant, apres = -1<<30, 1<<30
	for k := range r.scan.KeyframesUS {
		x := r.rjeFrameKF(k)
		if x <= f && x > avant {
			avant = x
		}
		if x >= g && x < apres {
			apres = x
		}
	}
	return avant, apres
}

// lieeA rend le nom de l'entree a laquelle l'entite k est liee, ou « - ».
func (r *rjeFilm) lieeA(k int) string {
	for i, o := range r.occ.parEntree {
		for _, x := range o.entites {
			if x == k {
				return r.doc.Roster[i].Name
			}
		}
	}
	return "-"
}

// placesLibres rend les sieges de la table qu'aucune presence CERTAINE publiee ne couvre a la frame f.
func (r *rjeFilm) placesLibres(f int) []string {
	tenues := map[int]string{}
	for _, e := range r.doc.Roster {
		if e.SeatSource == SeatSourceIndex {
			continue
		}
		for _, p := range e.Presence {
			if p.From <= f && f <= p.To {
				tenues[e.Seat] = e.Name
			}
		}
	}
	var out []string
	for _, s := range r.faits.Facts.FilmInputs.FilmTable.Seats {
		if _, ok := tenues[s.FilmIndex]; !ok {
			out = append(out, fmt.Sprintf("place %d (table : %s)", s.FilmIndex, s.Gamertag))
		}
	}
	sort.Strings(out)
	return out
}

// chaines rend, pour chaque place de l'artefact (siege publie hors `index`) qu'aucune presence
// CERTAINE ne couvre sur tout [de, a], son equipe, son dernier occupant avant `de` (le partant :
// fin certaine / fin affichee) et ses occupants qui commencent apres `de` (les successeurs).
func (r *rjeFilm) chaines(de, a int) []string {
	type occ struct {
		nom          string
		from, to, mx int
		equipe       *int
	}
	parPlace := map[int][]occ{}
	for _, e := range r.doc.Roster {
		if e.SeatSource == SeatSourceIndex {
			continue
		}
		for _, p := range e.Presence {
			mx := p.To
			if p.ToMax != nil {
				mx = *p.ToMax
			}
			parPlace[e.Seat] = append(parPlace[e.Seat], occ{e.Name, p.From, p.To, mx, e.Team})
		}
	}
	var out []string
	places := make([]int, 0, len(parPlace))
	for s := range parPlace {
		places = append(places, s)
	}
	sort.Ints(places)
	for _, s := range places {
		libre, equipe := true, "?"
		var partant, suivants []string
		for _, o := range parPlace[s] {
			if o.from <= a && de <= o.to {
				libre = false
			}
			if o.equipe != nil {
				equipe = fmt.Sprint(*o.equipe)
			}
			switch {
			case o.to < de:
				partant = append(partant, fmt.Sprintf("%s(fin %d/%d)", o.nom, o.to, o.mx))
			case o.from > a:
				suivants = append(suivants, fmt.Sprintf("%s(des %d)", o.nom, o.from))
			}
		}
		if libre {
			out = append(out, fmt.Sprintf("place %d equipe %s partant %v suivants %v", s, equipe, partant, suivants))
		}
	}
	return out
}

func TestRJEDiagnosticDesEntreesIndex(t *testing.T) {
	faitsDir, artDir := rjeEnv(t, "RJE_FAITS"), rjeEnv(t, "RJE_ARTEFACTS")
	catalogue := rjeEnv(t, "RJE_CATALOGUE")
	for _, id := range strings.Split(rjeEnv(t, "RJE_FILMS"), ",") {
		r := rjeCharger(t, id, faitsDir, artDir, catalogue)
		t.Logf("=== %s : %d frames, %d images-cles porteuses, %d entites, table de controle %v, %d bots",
			id, r.h.frames, len(r.scan.KeyframesUS), len(r.scan.Entities), r.equipe, len(r.bots))
		t.Logf("    origine %d us, module %q, largeurs d'axe %v", r.h.origin,
			r.faits.Facts.MapModule, r.faits.Facts.AxisW)
		for _, ent := range r.scan.Entities {
			t.Logf("    entite slot %d index %d designateur %d images-cles [%d..%d] vue %d", ent.Slot, ent.Index,
				ent.Team, r.rjeFrameKF(ent.FirstKF), r.rjeFrameKF(ent.LastKF), ent.Seen)
		}
		for _, b := range r.bots {
			ents := []string{}
			for _, k := range entitesDuBot(r.scan, b.FilmIndex, b.Declarations) {
				ents = append(ents, fmt.Sprintf("slot %d des %d", r.scan.Entities[k].Slot, r.scan.Entities[k].Team))
			}
			t.Logf("    bot %q index %d declarations %v entites %v", b.Name, b.FilmIndex,
				r.declarationsEnFrames(RosterEntry{Name: b.Name, FilmIndex: b.FilmIndex, Bot: true}), ents)
		}
		for i, e := range r.doc.Roster {
			if e.SeatSource != SeatSourceIndex {
				continue
			}
			r.diagnostiquer(t, i, e)
		}
		r.botsHorsRoster(t)
	}
}

// TestRJEEquipesDesBotsParLeurEntite releve, sur TOUS les films dont l'artefact et les faits existent,
// chaque bot declare (BOT_METADATA) et l'equipe que ses entites `ti=9` lui donnent — celles que ses
// declarations croisent (fenetre stricte, `entitesDuBot`), a l'unanimite. C'est la moitie « equipe
// connue » de la correlation G2.c : le TSV (film, bot, slot, bid, equipe ou « ? ») est l'oracle
// qu'une correlation aux octets des paquets BOT_METADATA, instrument de la couche des faits, consomme.
//
//	RJE_FAITS=... RJE_ARTEFACTS=... RJE_CATALOGUE=... RJE_SORTIE=<fichier tsv> \
//	  go test -tags research -count=1 -run '^TestRJEEquipesDesBotsParLeurEntite$' -v ./internal/games/halo_infinite/film/replay/
func TestRJEEquipesDesBotsParLeurEntite(t *testing.T) {
	faitsDir, artDir := rjeEnv(t, "RJE_FAITS"), rjeEnv(t, "RJE_ARTEFACTS")
	catalogue, sortie := rjeEnv(t, "RJE_CATALOGUE"), rjeEnv(t, "RJE_SORTIE")
	ents, err := os.ReadDir(artDir)
	if err != nil {
		t.Fatalf("artefacts : %v", err)
	}
	var b strings.Builder
	b.WriteString("film\tbot\tslot\tbid\tequipe\tentites\tdeclarations\n")
	films, bots, connus := 0, 0, 0
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
		for _, bot := range r.bots {
			bots++
			equipes := map[int]bool{}
			n := 0
			for _, k := range entitesDuBot(r.scan, bot.FilmIndex, bot.Declarations) {
				equipes[r.scan.Entities[k].Team] = true
				n++
			}
			equipe := "?"
			if len(equipes) == 1 {
				for v := range equipes {
					equipe = fmt.Sprint(v)
				}
				connus++
			}
			fmt.Fprintf(&b, "%s\t%s\t%d\t%d\t%s\t%d\t%d\n", id, strings.TrimSuffix(bot.Name, killsource.BotSuffix),
				bot.FilmIndex, bot.BotID, equipe, n, len(bot.Declarations))
		}
	}
	if err := os.WriteFile(sortie, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("sortie : %v", err)
	}
	t.Logf("%d films, %d bots declares, %d a l'equipe connue par leur entite -> %s", films, bots, connus, sortie)
}

// botsHorsRoster rend, pour chaque bot declare qu'aucune entree du roster ne porte, ses declarations
// et les places libres sur chacune.
func (r *rjeFilm) botsHorsRoster(t *testing.T) {
	t.Helper()
	presents := map[string]bool{}
	for _, e := range r.doc.Roster {
		presents[cleDeRoster(e)] = true
	}
	for _, b := range r.bots {
		e := RosterEntry{Name: b.Name, FilmIndex: b.FilmIndex, Bot: true}
		if presents[cleDeRoster(e)] {
			continue
		}
		for _, d := range r.declarationsEnFrames(e) {
			t.Logf("  BOT HORS ROSTER %q index %d declare [%d..%d] : places libres %v", b.Name, b.FilmIndex,
				d[0], d[1], r.chaines(d[0], d[1]))
		}
	}
}

func (r *rjeFilm) diagnostiquer(t *testing.T, i int, e RosterEntry) {
	t.Helper()
	pres := make([]string, 0, len(e.Presence))
	for _, p := range e.Presence {
		pres = append(pres, fmt.Sprintf("[%d..%d]", p.From, p.To))
	}
	t.Logf("  ENTREE %q index %d bot=%v equipe=%s presence %s", e.Name, e.FilmIndex, e.Bot, rjeEquipe(e.Team),
		strings.Join(pres, ""))
	if e.Bot {
		t.Logf("    declarations (frames) : %v", r.declarationsEnFrames(e))
	}
	if len(e.Presence) > 0 {
		avant, apres := r.imagesClesAutour(e.Presence[0].From, e.Presence[len(e.Presence)-1].To)
		t.Logf("    images-cles porteuses autour : <= %d, >= %d", avant, apres)
		t.Logf("    places de la table sans presence certaine a la frame %d : %v", e.Presence[0].From,
			r.placesLibres(e.Presence[0].From))
		for _, p := range e.Presence {
			t.Logf("    places libres sur TOUT [%d..%d] : %v", p.From, p.To, r.chaines(p.From, p.To))
		}
	}
	for k, ent := range r.scan.Entities {
		if ent.Index != e.FilmIndex {
			continue
		}
		t.Logf("    entite slot %d index %d designateur %d images-cles [%d..%d] vue %d instable %v croise ses declarations %v liee a %s",
			ent.Slot, ent.Index, ent.Team, r.rjeFrameKF(ent.FirstKF), r.rjeFrameKF(ent.LastKF), ent.Seen,
			ent.Unstable, e.Bot && entiteDeclareeParLeBot(r.scan, ent, declarationsDe(e, r.bots)), r.lieeA(k))
	}
	if v, ok := r.equipe[e.FilmIndex]; ok {
		t.Logf("    table de controle : index %d -> %d ; porte par une entite liee a une autre entree : %v",
			e.FilmIndex, v, indexPorteParUneEntiteLiee(&r.occ, r.scan, e.FilmIndex))
	} else {
		t.Logf("    table de controle : index %d absent (aucune lecture, ou index divergent)", e.FilmIndex)
	}
	for _, v := range r.occ.parEntree[i].vies {
		t.Logf("    vie [%d..%d]", v[0], v[1])
	}
}
