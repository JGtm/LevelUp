package killsource

// roster_remplacements_research_test.go — L INSTRUMENT DU LOT 5.2b.1 : QUI LE ROSTER DU
// KILLSOURCE CONNAIT-IL, ET QUI LUI MANQUE.
//
// Il repond a UNE question, sur UN film a la fois : quand `OutOfRoster > 0` eteint la
// publication ligne par ligne d un match entier, quel INDICE le dead-state porte-t-il, et une
// autre lecture du MEME film le nomme-t-elle ?
//
// TROIS LECTURES D IDENTITE SONT MISES COTE A COTE, et c est tout l objet :
//
//	table du film    `chunk_00`, 32 slots (xuid + gamertag + rang absolu) — la table ECRITE A
//	                 L OUVERTURE du film. Un joueur qui REMPLACE un partant apres cette ecriture
//	                 n y figure pas.
//	kill-feed        les gamertags et xuids que le chunk HIGHLIGHT porte — un joueur qui n a ni
//	                 tue ni ete tue n y figure pas.
//	motif du xuid    les 5 bits qui PRECEDENT le motif 64-bit du xuid dans un chunk de
//	                 replication (`weaponv3.ResolveXuidToPI`) — la meme lecture que
//	                 `replay.ScanPlayerIndices` publie sous le nom `PlayerIndexTable`, et la
//	                 seule des trois qui voie les remplacants.
//
// LECTURE SEULE, garde par deux variables — saute partout ailleurs, CI comprise :
//
//	KS_ROSTER_FILM=<repo>/data/cache/film_chunks/b1ad85eb KS_ROSTER_CARTE=Domicile \
//	  go test ./internal/games/halo_infinite/film/internal/facts/killsource/ \
//	  -run '^TestRosterEtRemplacements$' -v -count=1

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

const (
	ksRosterFilmEnv  = "KS_ROSTER_FILM"
	ksRosterCarteEnv = "KS_ROSTER_CARTE"
)

func TestRosterEtRemplacements(t *testing.T) {
	dir := os.Getenv(ksRosterFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : mesure sautee", ksRosterFilmEnv)
	}
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	f, err := loadFilm(src)
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	t.Logf("FILM %s", filepath.Base(dir))

	slots := journaliserTableDuFilm(t, f)
	feed := journaliserKillFeed(t, f)
	journaliserMotifDuXUID(t, f, slots, feed)
	journaliserRoster(t, f, feed)
	journaliserHorsRoster(t, f)
	journaliserSortiePubliee(t, src, filepath.Base(dir))
}

// journaliserSortiePubliee : LA SORTIE, decodee comme la PRODUCTION la decode — sous l entree de
// catalogue de la carte du match. C est la seule mesure comparable au bloc de sante du fichier de
// faits ; `cmd/killsource`, qui n a pas de base, decode aux largeurs par defaut.
func journaliserSortiePubliee(t *testing.T, src *source.Film, nom string) {
	t.Helper()
	opts := DefaultOptions()
	if carte := os.Getenv(ksRosterCarteEnv); carte != "" {
		e := carteDuCatalogue(t, carte)
		opts.Carte = &e
	}
	res, err := Decode(context.Background(), nom, src, &opts)
	if err != nil {
		t.Fatalf("decodage : %v", err)
	}
	h := res.Health
	t.Logf("SORTIE PUBLIEE (carte %q) — verdict %s", os.Getenv(ksRosterCarteEnv), h.Verdict())
	t.Logf("   candidats %d · publiees %d · horsRoster %d · inexpliques %d (couple %d, self %d, bot %d)",
		h.Candidates, h.Published, h.OutOfRoster, h.UnexplainedTotal(),
		h.UnexplainedPair, h.UnexplainedSelf, h.UnexplainedBotIdx)
	t.Logf("   alertes %v · degradations %v", h.Alerts(), h.Degradations())
	t.Logf("   LIGNE PAR LIGNE PUBLIABLE : %v (marge %d, determinee %v)",
		res.LineByLinePublishable(), res.BijectionMargin, res.BijectionDetermined)
	nommees, distinctes := 0, map[string]bool{}
	for i := range res.Kills {
		if res.Kills[i].Source.Named {
			nommees++
			distinctes[res.Kills[i].Source.Display] = true
		}
	}
	t.Logf("   sources NOMMEES : %d lignes sur %d, %d etiquettes distinctes",
		nommees, len(res.Kills), len(distinctes))
}

// journaliserTableDuFilm : la table de `chunk_00`, slots occupes, avec leur xuid.
func journaliserTableDuFilm(t *testing.T, f *film) []uint64 {
	t.Helper()
	registre := f.src.Chunk(0)
	ident, err := grammar.ReadFilmIdentity(registre)
	if err != nil {
		t.Logf("TABLE DU FILM : REFUSEE (identite) : %v", err)
		return nil
	}
	slots, rep, err := grammar.ReadPlayerTable(registre, ident)
	if err != nil {
		t.Logf("TABLE DU FILM : REFUSEE (table, build %s) : %v", ident.Build, err)
		return nil
	}
	t.Logf("TABLE DU FILM (chunk_00) — build %s, occupes %d, vacants %d, vacant intercale %v",
		ident.Build, rep.Occupied, rep.Vacant, rep.InterleavedVacant)
	xuids := make([]uint64, 0, len(slots))
	for _, s := range slots {
		t.Logf("   index %2d  xuid %-17d  %q", s.FilmIndex, s.XUID, s.Gamertag)
		if s.XUID != 0 {
			xuids = append(xuids, s.XUID)
		}
	}
	return xuids
}

// journaliserKillFeed : les noms du kill-feed et les xuids qu il porte.
func journaliserKillFeed(t *testing.T, f *film) *killFeed {
	t.Helper()
	kf, err := loadKillFeed(f)
	if err != nil {
		t.Fatalf("kill-feed : %v", err)
	}
	t.Logf("KILL-FEED — %d noms humains, %d kills, %d morts", len(kf.names), kf.nKills, kf.nDeaths)
	for _, n := range kf.names {
		t.Logf("   %-24q xuid %d", n, kf.xuidDe[n])
	}
	return kf
}

// journaliserMotifDuXUID : le meme lecteur que `replay.ScanPlayerIndices` — les 5 bits qui
// precedent le motif du xuid dans chaque chunk de replication, et la CONCORDANCE entre chunks.
func journaliserMotifDuXUID(t *testing.T, f *film, tableXUIDs []uint64, kf *killFeed) {
	t.Helper()
	xuids := unionDesXUID(tableXUIDs, kf)
	nums := grammar.FilmChunkNumbers(f.src)
	if len(nums) == 0 {
		t.Logf("MOTIF DU XUID : aucun chunk de replication")
		return
	}
	vus := map[uint64]map[int]int{}
	lectures := 0
	for _, c := range nums[:len(nums)-1] {
		raw, _, ok := grammar.FilmChunkAt(f.src, c)
		if !ok {
			continue
		}
		got := weaponv3.ResolveXuidToPI(xuids, raw)
		if len(got) == 0 {
			continue
		}
		lectures++
		for x, pi := range got {
			if vus[x] == nil {
				vus[x] = map[int]int{}
			}
			vus[x][pi]++
		}
	}
	t.Logf("MOTIF DU XUID (5 bits avant le motif 64-bit) — %d chunk(s) de replication lu(s), "+
		"%d xuid(s) cherche(s)", lectures, len(xuids))
	for _, x := range xuids {
		par := vus[x]
		if len(par) == 0 {
			t.Logf("   xuid %-17d  ABSENT des chunks de replication", x)
			continue
		}
		t.Logf("   xuid %-17d  index %s", x, indicesLisibles(par))
	}
}

// unionDesXUID : les xuids que le FILM SEUL donne — table de `chunk_00` et kill-feed.
func unionDesXUID(tableXUIDs []uint64, kf *killFeed) []uint64 {
	vu := map[uint64]bool{}
	out := make([]uint64, 0, len(tableXUIDs)+len(kf.xuidDe))
	ajouter := func(x uint64) {
		if x == 0 || vu[x] {
			return
		}
		vu[x] = true
		out = append(out, x)
	}
	for _, x := range tableXUIDs {
		ajouter(x)
	}
	for _, x := range kf.xuidDe {
		ajouter(x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// indicesLisibles : « 3 (x27) » ou « 3 (x20) 4 (x7) DESACCORD ».
func indicesLisibles(par map[int]int) string {
	idx := make([]int, 0, len(par))
	for i := range par {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	s := ""
	for _, i := range idx {
		s += fmt.Sprintf("%d (x%d) ", i, par[i])
	}
	if len(idx) > 1 {
		s += "DESACCORD"
	}
	return s
}

// journaliserRoster : ce que le decodeur RETIENT aujourd hui.
func journaliserRoster(t *testing.T, f *film, kf *killFeed) {
	t.Helper()
	tbl := readFilmTable(f)
	r := buildRoster(kf, loadBotMeta(f), DefaultOptions().Bots, tbl, lireIndexParMotif(f, tbl.slots, kf))
	r.perm, _ = solveBijection(r, kf.pairs, nil, DefaultOptions().BijectionRestarts)
	pub := r.public()
	t.Logf("ROSTER RETENU — borne nPlay %d, humains %d, noms %d", r.nPlay, pub.Humans, len(pub.Names))
	t.Logf("   provenance %+v", pub.FilmTable)
	libres, nomsLibres := r.freeSlots()
	t.Logf("   indices libres %v · noms libres %v · affectation unique %v",
		libres, nomsLibres, pub.FilmTable.AffectationUnique())
	for i, n := range pub.IndexToName {
		t.Logf("   index %2d  %-24q  source %s", i, n, pub.IndexSource[i])
	}
}

// journaliserHorsRoster : les dead-states que le compteur `OutOfRoster` retient, avec l indice
// qui deborde. C est la ligne qui NOMME la cause de l alerte.
func journaliserHorsRoster(t *testing.T, f *film) {
	t.Helper()
	opts := DefaultOptions()
	if nom := os.Getenv(ksRosterCarteEnv); nom != "" {
		e := carteDuCatalogue(t, nom)
		opts.Carte = &e
	}
	kf, err := loadKillFeed(f)
	if err != nil {
		t.Fatalf("kill-feed : %v", err)
	}
	tbl := readFilmTable(f)
	r := buildRoster(kf, loadBotMeta(f), opts.Bots, tbl, lireIndexParMotif(f, tbl.slots, kf))
	tl, err := newTimeline(f)
	if err != nil {
		t.Fatalf("timeline : %v", err)
	}
	tl.rewind()
	cal := calibrate(f, tl, opts.Views, opts.Carte)
	w := runWalk(f, tl, r, opts.Views, cal.Profil)
	t.Logf("DEAD-STATES HORS ROSTER (carte %q, %s)", os.Getenv(ksRosterCarteEnv), cal.String())
	par := map[[2]int]int{}
	for _, d := range w.deads {
		if d.slot < w.bipLo || d.slot > w.bipHi {
			continue
		}
		if int(d.dead.EnumA) < r.nPlay && int(d.dead.EnumB) < r.nPlay {
			continue
		}
		if !isCatalogued(d.dead.SrcTag0) {
			continue
		}
		par[[2]int{int(d.dead.EnumA), int(d.dead.EnumB)}]++
	}
	if len(par) == 0 {
		t.Logf("   aucun")
		return
	}
	cles := make([][2]int, 0, len(par))
	for k := range par {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		return cles[i][0] < cles[j][0] || (cles[i][0] == cles[j][0] && cles[i][1] < cles[j][1])
	})
	for _, k := range cles {
		t.Logf("   victime %2d  tueur %2d  x%d", k[0], k[1], par[k])
	}
}
