//go:build research

package grammar

// durees_effets_research_test.go — PLAN « DUREES ET POSITIONS DES EFFETS » (2026-10-28), ETAPE A :
// LE FILM PORTE-T-IL, PAR FAMILLE D EQUIPEMENT, UN OBJET D EFFET (ne a l usage ou a l impact, distinct
// de l objet que la pile i28 suit) DONT LA NAISSANCE, LA SUPPRESSION ET LA POSITION DONNENT LA DUREE ET
// LA POSITION DE L EFFET ?
//
// Pour chaque USAGE d equipement porte (lecture d activation du canal de production), l instrument
// liste les vies NEES dans [usage - 0,5 s, fin de l objet suivi + 1,5 s] (ou usage + 20 s quand aucun
// objet n est suivi ou que sa fin n est pas lue) que AUCUNE pile i28 ne nomme, et les memes comptes aux
// instants TEMOINS (usage + 37 s et usage - 37 s) pour separer le signal du bruit. La position a la
// creation se lit quand le premier composant de l archetype est la position d objet du monde, par
// [cfCanal.positionALaCreation]. Les canaux et aides sont ceux de champ_fin_research_test.go et
// equipporte_research_test.go : rien n est reecrit.
//
// Deux marches du meme film : la premiere (canal de production + [cfCanal] sans position) donne les
// usages, les objets suivis et TOUTES les vies ; la seconde lit la position a la creation des seules
// vies voulues. Rien n est ecrit hors de EQUIPPORTE_OUT. Un film par processus.
//
//	CGO_ENABLED=0 EQUIPPORTE_FILMS=<repo>/data/cache/film_chunks EQUIPPORTE_ID=f59c9332 \
//	  EQUIPPORTE_CARTE=Catalyst EQUIPPORTE_OUT=<scratchpad>/durees_effets go test -tags=research \
//	  ./internal/games/halo_infinite/film/internal/grammar/ -run '^TestDureesEffets$' -count=1 -v
//
// EQUIPPORTE_ARTEFACT (facultatif) : l artefact du rejeu, par defaut
// <EQUIPPORTE_FILMS>/../replays/halo_infinite/<id>.json (palette des familles, positions de repos).
// EQUIPPORTE_TIS_EXTRA (facultatif, defaut « 12 ») : archetypes dont TOUTES les vies sont listees dans
// <id>.ti_extra.tsv avec l usage le plus proche.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

const (
	deAvantUsageUS  = 500_000
	deApresFinUS    = 1_500_000
	deSansFinUS     = 20_000_000
	deTemoinUS      = 37_000_000
	deNaissanceNomm = 2_000_000 // une vie nommee par une pile naît a moins de 2 s de son nommage
	deUsageAvantUS  = 300_000   // l usage precede le nommage de l objet suivi (a 300 ms pres)
	deUsageApresUS  = 3_000_000
)

// deArtefact est ce que l instrument lit de l artefact du rejeu.
type deArtefact struct {
	FrameIntervalMs int `json:"frameIntervalMs"`
	AbilityLabels   map[string]struct {
		Family string `json:"family"`
	} `json:"abilityLabels"`
}

type deSuivi struct {
	cle, objet types.LifeKey
	nomme, fin uint64
	ti         int
	creation   [3]float32
	aCreation  bool
	repos      [3]float32
	reposEtat  string // piste (dernier point de la trajectoire), creation (ti hors projectile), absent
}

type deUsage struct {
	idx     int
	famille string
	rang    int
	objet   types.LifeKey
	porteur uint32
	ts      uint64
	suivis  []*deSuivi
}

type deCleNaissance struct {
	slot, gen uint32
	ts        uint64
}

type deCandidat struct {
	u      *deUsage
	role   string
	vie    *cfVie
	autres int
}

// deFen est une fenetre de recherche (reelle ou temoin) : autres = autres usages dans la fenetre,
// autresRang = ceux dont la famille est connue ; n = vies candidates.
type deFen struct {
	u                  *deUsage
	role               string
	autres, autresRang int
	n                  int
	dureeMs            float64
}

// dePositions est la seconde marche : la position a la creation des vies voulues.
type dePositions struct {
	aide    *cfCanal
	reg     *Registry
	voulus  map[deCleNaissance]bool
	trouves map[deCleNaissance][3]float32
}

func (c *dePositions) Interets() []Interet                      { return nil }
func (c *dePositions) Clore(BilanDeMarche)                      {}
func (c *dePositions) Brancher(*Observation, *MarcheDistribuee) {}

func (c *dePositions) Trame(p *lecture.Paquet) {
	for _, r := range p.Records {
		if r.Genre != lecture.GenreNeuf {
			continue
		}
		k := deCleNaissance{r.Vie.Slot, r.Vie.Gen, p.TS}
		if !c.voulus[k] {
			continue
		}
		a, ok := c.reg.Archetype(int(r.TI))
		if !ok || len(a.Components) == 0 || a.Components[0] != compObjectPosition {
			continue
		}
		if pos, ok := c.aide.positionALaCreation(p, r); ok {
			c.trouves[k] = pos
		}
	}
}

// deLireArtefact lit la palette des familles de l artefact du rejeu.
func deLireArtefact(chemin string) (*deArtefact, error) {
	b, err := os.ReadFile(chemin)
	if err != nil {
		return nil, err
	}
	var a deArtefact
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// deLireLesUsages rend les usages (activations du canal de production) et les objets suivis, avec la
// famille de chaque usage par le rang i48 et la palette de l artefact.
func deLireLesUsages(lectures []types.CarriedEquipmentRead, rangs map[[2]uint64]int,
	famille func(int) string) ([]*deUsage, []*deSuivi) {
	rangObjet := map[types.LifeKey]int{}
	ouverts := map[types.LifeKey]*deSuivi{}
	var usages []*deUsage
	var suivis []*deSuivi
	for _, l := range lectures {
		switch l.Kind {
		case types.CarriedNew:
			r, ok := rangs[[2]uint64{l.TimestampUS, uint64(l.Carrier)}]
			if !ok {
				r = -1
			}
			rangObjet[l.Object] = r
		case types.CarriedTrackedEnd:
			if s := ouverts[l.Tracked[0]]; s != nil {
				s.fin = l.TimestampUS
				delete(ouverts, l.Tracked[0])
			}
			continue
		}
		if l.HasActivation {
			r, ok := rangObjet[l.Object]
			if !ok {
				r = -1
			}
			usages = append(usages, &deUsage{idx: len(usages), rang: r, famille: famille(r), objet: l.Object,
				porteur: l.Carrier, ts: l.TimestampUS})
		}
		for _, k := range l.Tracked {
			if ouverts[k] != nil {
				continue
			}
			s := &deSuivi{cle: k, objet: l.Object, nomme: l.TimestampUS, ti: -1, reposEtat: "absent"}
			for _, n := range l.Spawned {
				if n.Life == k {
					s.ti, s.creation, s.aCreation = int(n.TypeIndex), [3]float32{n.X, n.Y, n.Z}, n.HasPosition
				}
			}
			ouverts[k] = s
			suivis = append(suivis, s)
		}
	}
	return usages, suivis
}

// deRattacher rattache chaque objet suivi a l usage le plus recent du MEME objet porte qui le precede
// (a deUsageAvantUS pres) de moins de deUsageApresUS. Rend le nombre d objets suivis sans usage.
func deRattacher(usages []*deUsage, suivis []*deSuivi) int {
	sans := 0
	for _, s := range suivis {
		var best *deUsage
		for _, u := range usages {
			if u.objet != s.objet || int64(u.ts) > int64(s.nomme)+deUsageAvantUS ||
				int64(s.nomme)-int64(u.ts) > deUsageApresUS {
				continue
			}
			if best == nil || u.ts > best.ts {
				best = u
			}
		}
		if best == nil {
			sans++
			continue
		}
		best.suivis = append(best.suivis, s)
	}
	return sans
}

// deToleranceNaissanceUS borne l ecart entre le nommage d un objet suivi et la naissance de sa piste.
const deToleranceNaissanceUS = 400_000

// deSituerLesRepos joint a chaque objet suivi sa position de repos : le dernier point de la piste de
// SA vie (meme couple slot/generation, naissance la plus proche du nommage, a 400 ms pres) quand c est
// un projectile ti=41, sa position a la creation sinon. Le repos est lu sur le film, pas sur l artefact
// (les artefacts en cache ne le publient pas).
func deSituerLesRepos(suivis []*deSuivi, pistes []types.ProjectileTrack) {
	par := map[types.LifeKey][]types.ProjectileTrack{}
	for _, p := range pistes {
		if len(p.Pts) > 0 {
			k := types.LifeKey{Slot: p.Slot, Gen: p.Gen}
			par[k] = append(par[k], p)
		}
	}
	for _, s := range suivis {
		if s.ti != ProjectileTypeIndex {
			if s.aCreation {
				s.repos, s.reposEtat = s.creation, "creation"
			}
			continue
		}
		var best *types.ProjectileTrack
		ecart := int64(deToleranceNaissanceUS) + 1
		for i, p := range par[s.cle] {
			if e := deAbs64(int64(p.Pts[0].TimestampUS) - int64(s.nomme)); e < ecart {
				best, ecart = &par[s.cle][i], e
			}
		}
		if best != nil {
			d := best.Pts[len(best.Pts)-1]
			s.repos, s.reposEtat = [3]float32{d.X, d.Y, d.Z}, "piste"
		}
	}
}

func dePos(p [3]float32, a bool) string {
	if !a {
		return "\t\t"
	}
	return fmt.Sprintf("%.2f\t%.2f\t%.2f", p[0], p[1], p[2])
}

func deDist(a, b [3]float32) (d3, dh float64) {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz), math.Sqrt(dx*dx + dy*dy)
}

// deFenetre rend la fenetre de recherche d un usage, en µs de l horloge des paquets, et si la fin de
// l objet suivi y est lue.
func deFenetre(u *deUsage) (w0, w1 int64, finLue bool) {
	tu := int64(u.ts)
	w0, w1 = tu-deAvantUsageUS, tu+deSansFinUS
	if len(u.suivis) > 0 && u.suivis[0].fin > 0 {
		w1, finLue = int64(u.suivis[0].fin)+deApresFinUS, true
	}
	return w0, w1, finLue
}

// deCandidats rend, pour chaque usage et chaque instant (reel, temoin+, temoin-), les vies nees dans la
// fenetre que aucune pile i28 ne nomme.
func deCandidats(usages []*deUsage, vies []*cfVie, nommes map[[2]uint32][]uint64, origine uint64,
	derniere uint64) ([]deCandidat, []deFen) {
	tsUsages := make([]int64, len(usages))
	for i, u := range usages {
		tsUsages[i] = int64(u.ts)
	}
	sort.Slice(tsUsages, func(i, j int) bool { return tsUsages[i] < tsUsages[j] })
	tsRang := make([]int64, 0, len(usages))
	for _, u := range usages {
		if u.famille != "?" {
			tsRang = append(tsRang, int64(u.ts))
		}
	}
	sort.Slice(tsRang, func(i, j int) bool { return tsRang[i] < tsRang[j] })
	compter := func(ts []int64, a, b int64) int {
		return sort.Search(len(ts), func(i int) bool { return ts[i] > b }) -
			sort.Search(len(ts), func(i int) bool { return ts[i] >= a })
	}
	var out []deCandidat
	var fens []deFen
	for _, u := range usages {
		w0, w1, _ := deFenetre(u)
		for _, d := range []struct {
			role  string
			decal int64
		}{{"reel", 0}, {"temoin+", deTemoinUS}, {"temoin-", -deTemoinUS}} {
			a, b := w0+d.decal, w1+d.decal
			if d.decal != 0 && (int64(u.ts)+d.decal < int64(origine) || int64(u.ts)+d.decal > int64(derniere)) {
				continue
			}
			autres, autresRang := compter(tsUsages, a, b), compter(tsRang, a, b)
			if d.decal == 0 {
				autres--
				if u.famille != "?" {
					autresRang--
				}
			}
			n := 0
			i0 := sort.Search(len(vies), func(i int) bool { return int64(vies[i].ne) >= a })
			for i := i0; i < len(vies) && int64(vies[i].ne) <= b; i++ {
				v := vies[i]
				if deNomme(nommes, v) {
					continue
				}
				out = append(out, deCandidat{u: u, role: d.role, vie: v, autres: autres})
				n++
			}
			fens = append(fens, deFen{u: u, role: d.role, autres: autres, autresRang: autresRang, n: n,
				dureeMs: float64(b-a) / 1000})
		}
	}
	return out, fens
}

// deNomme dit si la vie est l objet qu une pile i28 a designe (meme couple, naissance a moins de 2 s du
// nommage).
func deNomme(nommes map[[2]uint32][]uint64, v *cfVie) bool {
	for _, n := range nommes[[2]uint32{v.slot, v.gen}] {
		if math.Abs(float64(int64(v.ne)-int64(n))) <= deNaissanceNomm {
			return true
		}
	}
	return false
}

func deMs(ts, origine uint64) float64 { return float64(int64(ts)-int64(origine)) / 1000 }

func deEntier(s string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return v
	}
	return def
}

func TestDureesEffets(t *testing.T) {
	racine, id, out := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID"), os.Getenv("EQUIPPORTE_OUT")
	if racine == "" || id == "" || out == "" {
		t.Skip("EQUIPPORTE_FILMS, EQUIPPORTE_ID ou EQUIPPORTE_OUT absent : instrument saute")
	}
	debut, mem := time.Now(), deSurveillerLaMemoire()
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	chemin := os.Getenv("EQUIPPORTE_ARTEFACT")
	if chemin == "" {
		chemin = filepath.Join(racine, "..", "replays", "halo_infinite", id+".json")
	}
	art, err := deLireArtefact(chemin)
	if err != nil {
		t.Fatalf("%s : artefact : %v", id, err)
	}
	famille := func(r int) string {
		if l, ok := art.AbilityLabels[strconv.Itoa(r)]; ok && l.Family != "" {
			return l.Family
		}
		return "?"
	}

	// MARCHE 1 : toutes les vies (sans position), les lectures de l equipement porte.
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	origine, _ := ScanClockOrigin(fc.Film())
	wr := bornesDeLaCarte(t)
	cf := &cfCanal{actives: map[uint32]*cfVie{}, posTI: -1}
	if err := Distribuer(fc, cf, nouveauCanalDeLEquipementPorte(fc)); err != nil {
		t.Fatalf("%s : marche 1 : %v", id, err)
	}
	t1 := time.Now()
	lectures, st, err := ScanEquipementPorte(fc, &wr)
	if err != nil {
		t.Fatal(err)
	}
	fc2, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	ch, _, err := ScanEquipmentChanges(fc2, nil)
	if err != nil {
		t.Fatalf("%s : i48 : %v", id, err)
	}
	rangs := map[[2]uint64]int{}
	for _, e := range ch {
		if e.Kind == types.EquipmentTaken || e.Kind == types.EquipmentSpawned {
			rangs[[2]uint64{e.TimestampUS, uint64(e.Slot)}] = e.Rank
		}
	}
	fc2 = nil
	fc3, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}

	usages, suivis := deLireLesUsages(lectures, rangs, famille)
	sansUsage := deRattacher(usages, suivis)
	pistes, err := ScanProjectiles(fc3, &wr)
	if err != nil {
		t.Fatalf("%s : pistes : %v", id, err)
	}
	fc3 = nil
	deSituerLesRepos(suivis, pistes)
	vies := append([]*cfVie(nil), cf.vies...)
	sort.SliceStable(vies, func(i, j int) bool { return vies[i].ne < vies[j].ne })
	var derniere uint64
	if len(vies) > 0 {
		derniere = vies[len(vies)-1].ne
	}
	nommes := map[[2]uint32][]uint64{}
	for _, s := range suivis {
		k := [2]uint32{s.cle.Slot, s.cle.Gen}
		nommes[k] = append(nommes[k], s.nomme)
	}
	cands, fens := deCandidats(usages, vies, nommes, origine, derniere)

	// Les archetypes extra : toutes leurs vies.
	extra := map[int16]bool{}
	for _, s := range strings.Split(envOu("EQUIPPORTE_TIS_EXTRA", "12"), ",") {
		if s = strings.TrimSpace(s); s != "" {
			extra[int16(deEntier(s, -1))] = true
		}
	}
	voulus := map[deCleNaissance]bool{}
	tiVus := map[int16]int{}
	for _, c := range cands {
		voulus[deCleNaissance{c.vie.slot, c.vie.gen, c.vie.ne}] = true
		tiVus[c.vie.ti]++
	}
	for _, v := range vies {
		if extra[v.ti] {
			voulus[deCleNaissance{v.slot, v.gen, v.ne}] = true
			tiVus[v.ti] += 0
		}
	}

	// MARCHE 2 : la position a la creation des vies voulues.
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("%s : registre : %v", id, err)
	}
	pos := &dePositions{aide: &cfCanal{fc: fc, wr: wr}, reg: reg, voulus: voulus,
		trouves: map[deCleNaissance][3]float32{}}
	if err := Distribuer(fc, pos, nouveauCanalDeLEquipementPorte(fc)); err != nil {
		t.Fatalf("%s : marche 2 : %v", id, err)
	}
	t2 := time.Now()

	deEcrireUsages(t, out, id, origine, usages)
	deEcrireCandidats(t, out, id, origine, cands, pos.trouves)
	deEcrireFenetres(t, out, id, fens)
	deEcrireExtra(t, out, id, origine, vies, extra, usages, suivis, pos.trouves)
	deEcrireArchetypes(t, out, id, reg, tiVus)

	tas, sys := mem.arreter()
	run := fmt.Sprintf("film\t%s\nusages\t%d\nsuivis\t%d\nsuivis_sans_usage\t%d\nvies\t%d\ncandidats\t%d\n"+
		"marche1_s\t%.1f\nmarche2_s\t%.1f\ntotal_s\t%.1f\npic_tas_mo\t%d\npic_sys_mo\t%d\nstats\t%+v\n",
		id, len(usages), len(suivis), sansUsage, len(vies), len(cands), t1.Sub(debut).Seconds(),
		t2.Sub(t1).Seconds(), time.Since(debut).Seconds(), tas>>20, sys>>20, st)
	if err := os.WriteFile(filepath.Join(out, id+".run.txt"), []byte(run), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Log("\n" + run)
}
