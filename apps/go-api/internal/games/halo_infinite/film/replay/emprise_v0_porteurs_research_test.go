//go:build research

package replay

// emprise_v0_porteurs_research_test.go — LOT V0.2 DU PLAN `.ai/PLAN_EMPRISE_VIES_2026-09-28.md` :
// LES PORTEURS D'OBJECTIF LUS AU SYNC, DEUX VOIES MESUREES SUR LE FILM QUE LE COLLECTEUR A OUVERT.
//
// # CE QUE CET INSTRUMENT REJOUE
//
// La BASE, d'abord : ce que la passe du collecteur lit deja du film, par les MEMES fonctions et
// dans le MEME ordre que `sync/killcollector/positions.go` (`buildPositionRows`) — decodage
// killsource, positions (bornes et decoupage du catalogue), horloge, fil des morts, index de
// joueur, creations de bipede, puis le registre (`BuildIdentityRegistry` sur la meme entree que
// `entreeDuRegistre`). Le registre obtenu est CONTROLE contre celui de la vraie passe (calage et
// vies nommees, poses par le volet collecteur) : une difference est publiee, jamais ignoree.
//
// PUIS LES DEUX VOIES, en supplement, sur ce meme film et ce meme contexte :
//
//	(a) LE CANAL DES ARMES TENUES : images-cles d'armes, dotations de naissance, changements
//	    d'arme tenue (`ScanHeldWeaponChanges`, la chaine de `balayerPortage`) ; les portages du
//	    drapeau (famille 0x2a392328), du crane (0x0017592c) et de la bombe (0x3fee4fcf) sont
//	    reconstruits par la regle de production `BuildHeldObjectCarry`, pontes par le registre
//	    du collecteur a l'instant. La couronne VIP n'est pas un objet tenu : aucune source.
//	(b) LES LECTURES DE LA CUISSON : les enregistrements d'entite (`StatRecordsCtx`), les bursts
//	    de capture, le pont par manche (morts, triplet, elimination, residu — la chaine de
//	    `replaybuild/matchfacts.go`), et ce que chaque calque exige en plus : pour le drapeau les
//	    equipes du film et les objets du monde (vies libres), pour la bombe le canal ci-dessus
//	    (`bombInput` ne lit « aucune donnee de plus »). Les quatre calques sont produits par
//	    l'assembleur de production (`BuildFromPositions`) nourri des SEULES entrees que la
//	    synchronisation aurait — positions et registre du collecteur, garde de mode par la
//	    variante comme la cuisson.
//
// LA REFERENCE est le document construit en memoire par la cuisson actuelle (volet
// `replaybuild`), comparee en MILLISECONDES DU MATCH, chaque cote avec son propre calage.
//
// COUTS : chaque lecture supplementaire est chronometree sur plusieurs tours (mediane) ; le
// denominateur est la mediane de la vraie passe (`CollectMatch`, volet collecteur). MEMOIRE :
// pic d'empreinte (`filmproc.Footprint`, echantillonne a 10 ms) de la base, puis des lectures
// supplementaires tant que la base est vivante.
//
// Aucune base, aucune ecriture hors du repertoire du lot. SANS LES VARIABLES, IL SE SAUTE :
//
//	EMPRISE_V0_FILMS=<match_id,...> EMPRISE_V0_DIR=<scratch> EMPRISE_V0_CACHE=<racine data/cache> \
//	[EMPRISE_V0_TOURS=3] go test ./internal/games/halo_infinite/film/replay/ \
//	  -run '^TestEmpriseV0Porteurs$' -v -count=1

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"levelup/go-api/internal/testutil"
)

// Les trois familles d'objet tenu (32 bits hauts de l'identifiant, `replay_labels.toml` et
// `bomb_carries.go`).
const (
	v0FamDrapeau = uint32(0x2a392328)
	v0FamCrane   = uint32(0x0017592c)
	v0FamBombe   = uint32(0x3fee4fcf)
)

// v0Films lit la liste des films, le repertoire du lot, le cache et le nombre de tours.
func v0Films(t *testing.T) (films []string, dir, cache string, tours int) {
	t.Helper()
	dir, cache, tours = os.Getenv("EMPRISE_V0_DIR"), os.Getenv("EMPRISE_V0_CACHE"), 3
	for _, f := range strings.Split(os.Getenv("EMPRISE_V0_FILMS"), ",") {
		if f = strings.TrimSpace(f); f != "" {
			films = append(films, f)
		}
	}
	if len(films) == 0 || dir == "" || cache == "" {
		t.Skip("EMPRISE_V0_FILMS, EMPRISE_V0_DIR et EMPRISE_V0_CACHE requis : instrument saute")
	}
	if v, err := strconv.Atoi(os.Getenv("EMPRISE_V0_TOURS")); err == nil && v > 0 {
		tours = v
	}
	return films, dir, cache, tours
}

// v0Lire relit un JSON du lot ; faux s'il manque.
func v0Lire(t *testing.T, dir, sous, court string, v any) bool {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join(dir, sous, court+".json"))
	if err != nil {
		return false
	}
	if err := json.Unmarshal(blob, v); err != nil {
		t.Fatalf("%s/%s : %v", sous, court, err)
	}
	return true
}

// v0Ref : la reference de la cuisson (sous-ensemble).
type v0Ref struct {
	Variante        string       `json:"variante"`
	FrameIntervalMS int          `json:"frameIntervalMs"`
	OrigineUS       uint64       `json:"origineUs"`
	CalageRefMs     *int64       `json:"calageRefMs"`
	Drapeau         []v0Periode  `json:"drapeau"`
	Crane           []v0Periode  `json:"crane"`
	Bombe           []v0Periode  `json:"bombe"`
	VIP             []v0Periode  `json:"vip"`
	Socles          []FlagSpawn  `json:"socles"`
	Libelles        LabelCatalog `json:"libelles"`
}

// v0Periode : un portage en frames du document.
type v0Periode struct {
	XUID string `json:"xuid"`
	T0   int    `json:"t0"`
	T1   int    `json:"t1"`
}

// v0Col : ce que le volet collecteur a pose (sous-ensemble).
type v0Col struct {
	PasseMedMS   float64       `json:"passeMedianeMs"`
	PicPasse     []uint64      `json:"picPasse"`
	RosterXUIDs  []string      `json:"rosterXuids"`
	Participants []Participant `json:"participants"`
	Bots         []BotIdentity `json:"bots"`
	Calage       int64         `json:"calageMs"`
	Vies         [][3]int64    `json:"vies"`
}

// v0K1 : les faits du match lus en base par le volet collecteur.
type v0K1 struct {
	Noms  []string          `json:"noms"`
	Faits domain.MatchFacts `json:"faits"`
}

// v0Base : ce que la passe du collecteur a deja en main quand les voies commencent.
type v0Base struct {
	id        string
	film      *source.Film
	fc        *grammar.FilmContext
	entry     decfilm.MapQuantEntry
	res       *decfilm.Result
	positions []grammar.BipedPosition
	creations []grammar.BipedCreation
	deaths    []types.Death
	idx       types.PlayerIndexTable
	roster    []uint64
	reg       IdentityRegistry
	clockUS   uint64
	col       v0Col
}

func TestEmpriseV0Porteurs(t *testing.T) {
	films, dir, cache, tours := v0Films(t)
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	cat, err := decfilm.LoadMapQuantCatalog(title.NewPathResolver(repoRoot).MapQuantBoundsPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	for _, id := range films {
		court := title.FilmShortMatchID(id)
		var ref v0Ref
		var col v0Col
		var k1 v0K1
		if !v0Lire(t, dir, "ref", court, &ref) || !v0Lire(t, dir, "col", court, &col) ||
			!v0Lire(t, dir, "k1", court, &k1) {
			t.Fatalf("%s : ref/col/k1 absent — jouer les volets cuisson et collecteur d'abord", court)
		}
		r := v0MesurerFilm(t, v0Entree{id: id, cache: cache, cat: cat, ref: ref, col: col, k1: k1, tours: tours})
		v0Poser(t, dir, court, r)
	}
}

// v0Entree groupe ce qu'une mesure de film recoit (le depot borne a cinq parametres).
type v0Entree struct {
	id, cache string
	cat       *decfilm.MapQuantCatalog
	ref       v0Ref
	col       v0Col
	k1        v0K1
	tours     int
}

// v0FamilleDuMode rend la famille de porteur du mode, par la variante — la garde de la cuisson
// (`isVipVariant`, `isSkullVariant`, `isBombVariant`) et, pour le drapeau, le meme predicat
// canonique (`ObjectiveTypeOf`) que la famille d'objectif.
func v0FamilleDuMode(variante string) string {
	if strings.Contains(strings.ToLower(variante), "vip") {
		return "vip"
	}
	switch decfilm.ObjectiveTypeOf(variante) {
	case decfilm.ObjectiveTypeFlag:
		return "drapeau"
	case decfilm.ObjectiveTypeSkull:
		return "crane"
	case decfilm.ObjectiveTypeBomb:
		return "bombe"
	}
	return ""
}

// v0ChargerBase rejoue la passe du collecteur jusqu'au registre, SOUS ECHANTILLONNAGE MEMOIRE.
func v0ChargerBase(t *testing.T, e v0Entree) (v0Base, uint64) {
	t.Helper()
	court := title.FilmShortMatchID(e.id)
	film, ok, err := filmcache.LoadFilm(e.cache, court)
	if err != nil || !ok {
		t.Fatalf("%s : film : %v", court, err)
	}
	b := v0Base{id: e.id, film: film, col: e.col}
	for _, n := range e.k1.Noms {
		if entry, err := e.cat.Lookup(n); err == nil {
			b.entry = entry
			break
		}
	}
	ech := v0Echantillonner()
	opts := decfilm.DefaultOptions()
	opts.Carte = &b.entry
	if b.res, err = decfilm.Decode(context.Background(), e.id, film, &opts); err != nil {
		t.Fatalf("%s : decodage : %v", court, err)
	}
	b.fc = grammar.NewFilmContextForMap(film, &b.entry, nil)
	scan := grammar.DefaultScanFilmOptions()
	rng := b.entry.Range()
	scan.WorldRange, scan.Layout = &rng, b.fc.ImposedLayout()
	if b.positions, err = grammar.ScanBipedPositions(b.fc, scan); err != nil {
		t.Fatalf("%s : positions : %v", court, err)
	}
	b.clockUS, _ = grammar.ScanClockOrigin(film)
	b.deaths, _ = grammar.ScanDeaths(film)
	for _, s := range e.col.RosterXUIDs {
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			b.roster = append(b.roster, v)
		}
	}
	b.idx, _ = grammar.ScanPlayerIndices(film, b.roster)
	b.creations, _, _ = grammar.ScanBipedCreations(b.fc)
	b.reg = BuildIdentityRegistry(context.Background(), IdentityInput{
		Positions: b.positions, BipedCreations: b.creations, Deaths: b.deaths,
		PlayerIndices: b.idx, RosterXUIDs: b.roster, Bots: e.col.Bots,
		Participants: e.col.Participants, MatchID: e.id,
	})
	return b, ech.arreter()
}

// v0RegistreConforme compare le registre rejoue a celui de la vraie passe.
func v0RegistreConforme(b v0Base) string {
	if b.reg.DeathOffsetMS() != b.col.Calage {
		return fmt.Sprintf("calage %d contre %d", b.reg.DeathOffsetMS(), b.col.Calage)
	}
	var vies [][3]int64
	for _, v := range b.reg.ViesNommees() {
		vies = append(vies, [3]int64{int64(v.XUID), v.DebutMS, v.FinMS})
	}
	a, c := append([][3]int64(nil), vies...), append([][3]int64(nil), b.col.Vies...)
	less := func(s [][3]int64) func(i, j int) bool {
		return func(i, j int) bool {
			for k := 0; k < 3; k++ {
				if s[i][k] != s[j][k] {
					return s[i][k] < s[j][k]
				}
			}
			return false
		}
	}
	sort.Slice(a, less(a))
	sort.Slice(c, less(c))
	if len(a) != len(c) {
		return fmt.Sprintf("%d vies nommees contre %d", len(a), len(c))
	}
	for i := range a {
		if a[i] != c[i] {
			return fmt.Sprintf("vie %d differente : %v contre %v", i, a[i], c[i])
		}
	}
	return ""
}

// v0Chrono execute `f` `n` fois, et rend la mediane (ms) et le pic d'empreinte de tous les tours.
func v0Chrono(n int, f func()) (float64, uint64) {
	var durees []float64
	var pic uint64
	for i := 0; i < n; i++ {
		ech := v0Echantillonner()
		debut := time.Now()
		f()
		durees = append(durees, float64(time.Since(debut).Microseconds())/1000)
		if p := ech.arreter(); p > pic {
			pic = p
		}
	}
	sort.Float64s(durees)
	return durees[len(durees)/2], pic
}

// v0Lectures : ce que les voies ont lu en supplement, et ce que ca a coute.
type v0Lectures struct {
	held                 []types.HeldWeaponChange
	recs                 []types.StatRecord
	bursts               []int
	pont                 decfilm.RoundIdentity
	teams                map[int]int
	teamScan             grammar.TeamScanReport
	entities             grammar.PlayerEntityScan
	pads                 PadScans
	coutA, coutStatborg  float64
	coutPont, coutDrap   float64
	coutEquipes          float64
	coutMonde            float64
	picA, picStat, picDr uint64
}

// v0LireEnPlus execute les lectures supplementaires des deux voies (tours, mediane, pics).
func v0LireEnPlus(t *testing.T, e v0Entree, b v0Base, famille string) v0Lectures {
	t.Helper()
	var l v0Lectures
	// LE PROFIL CALIBRE PAR LE KILL-FEED, PUIS LA PRECISION DES OBJETS DU MONDE : l'ordre de la
	// cuisson (`poserProfilPuisCarte`), pose sur le contexte AVANT toute lecture supplementaire —
	// sans lui, les objets du monde se liraient aux largeurs d'une autre carte.
	prof := b.res.ProfilCalibre
	poserProfilPuisCarte(context.Background(), b.fc, e.id, Options{ProfilDeBalayage: &prof, Fallbacks: fallback.NouveauCompteur()})
	l.coutA, l.picA = v0Chrono(e.tours, func() {
		loadouts, _, _ := grammar.ScanKeyframeLoadoutsMarche(b.fc, loadoutFamilies())
		births, _, _ := grammar.ScanBirthLoadouts(b.fc, b.creations)
		l.held, _, _ = grammar.ScanHeldWeaponChanges(b.fc, spawnSetFrom(loadouts, births, b.creations))
	})
	l.coutStatborg, l.picStat = v0Chrono(e.tours, func() {
		l.recs, _, _ = decfilm.StatRecordsBornes(b.film, e.id)
		l.bursts = decfilm.CaptureBurstTimes(b.film)
	})
	lines := make([]types.PlayerLine, 0, len(e.k1.Faits.Players))
	for _, p := range e.k1.Faits.Players {
		lines = append(lines, types.PlayerLine{XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists})
	}
	var picPont uint64
	l.coutPont, picPont = v0Chrono(e.tours, func() {
		l.pont = decfilm.ResolveRoundIdentity(l.recs, deathInstantsOf(b.deaths), nil).
			CompletedByLines(l.recs, lines).
			CompletedByElimination(l.recs, lines).
			CompletedByRoundResidue(l.recs, lines)
	})
	l.picStat = max(l.picStat, picPont)
	if famille == "drapeau" {
		world := b.entry.Range()
		var picEq uint64
		l.coutEquipes, picEq = v0Chrono(e.tours, func() {
			l.teams, l.teamScan, l.entities = grammar.ScanPlayerTeams(b.fc)
		})
		// LES OBJETS DU MONDE : les poses calibrent les largeurs MPP dont les socles (et les vies
		// libres du drapeau) heritent — la chaine de `balayerMonde`, dans son ordre.
		l.coutMonde, l.picDr = v0Chrono(e.tours, func() {
			_, st := decodeFilmPlacements(context.Background(), b.fc, e.id, &world)
			l.pads = decodeFilmPadScans(context.Background(), b.fc, e.id, &world, st.Calibration.Widths)
		})
		l.coutDrap, l.picDr = l.coutEquipes+l.coutMonde, max(l.picDr, picEq)
	}
	return l
}

// v0OptionsSync assemble les entrees que la synchronisation aurait pour la voie (b).
func v0OptionsSync(e v0Entree, b v0Base, l v0Lectures, famille string) Options {
	opt := Options{
		MapQuant: &b.entry, Labels: e.ref.Libelles, RosterXUIDs: b.roster,
		Participants: e.col.Participants, Bots: e.col.Bots, Deaths: b.deaths,
		PlayerIndices: b.idx, BipedCreations: b.creations, FilmClockOriginUS: b.clockUS,
	}
	switch famille {
	case "drapeau":
		opt.Flag = FlagInput{Scanned: true, Records: l.recs, Bursts: l.bursts, Spawns: e.ref.Socles}
		if decfilm.FlagFilmSignalsFrom(l.bursts, decfilm.NamedEventsFrom(l.recs, decfilm.ObjectiveTypeFlag, nil)).IsFlagFilm() {
			opt.Flag.Identity = l.pont
		}
		opt.PlayerTeams, opt.TeamScan, opt.PlayerEntities = l.teams, l.teamScan, l.entities
		opt.Pads = l.pads
	case "crane":
		opt.Skull = SkullInput{Scanned: true, Records: l.recs, Identity: l.pont}
	case "vip":
		opt.Vip = VipInput{Scanned: true, Records: l.recs}
	case "bombe":
		opt.Bomb = BombInput{CarryScanned: true}
		opt.WeaponChanges = l.held
	}
	return opt
}

// v0Poser ecrit le resultat d'un film.
func v0Poser(t *testing.T, dir, court string, r v0Resultat) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "porteurs"), 0o750); err != nil {
		t.Fatalf("repertoire : %v", err)
	}
	blob, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("%s : %v", court, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "porteurs", court+".json"), blob, 0o600); err != nil {
		t.Fatalf("%s : %v", court, err)
	}
	t.Logf("%s  %-26s famille=%-7s registre=%s passe=%.0f ms | (a) %.0f ms %.1f %% | (b) %.0f ms %.1f %% "+
		"| pic base %.2f (a) %.2f (b) %.2f Gio", court, r.Variante, r.Famille, v0OuiNon(r.RegistreEcart),
		r.PasseMS, r.CoutA, r.SurcoutA, r.CoutB, r.SurcoutB, v0G(r.PicBase), v0G(r.PicA), v0G(r.PicB))
	for _, voie := range []string{"a", "b"} {
		for fam, f := range r.Fid[voie] {
			t.Logf("%s    voie (%s) %-7s reference %8d ms  identique a +-100 ms %8d ms  en trop %7d ms  -> %.2f %%",
				court, voie, fam, f.RefMS, f.OkMS, f.ExtraMS, 100*f.Taux())
		}
	}
}

func v0OuiNon(ecart string) string {
	if ecart == "" {
		return "conforme"
	}
	return "ECART(" + ecart + ")"
}

func v0G(v uint64) float64 { return float64(v) / (1 << 30) }
