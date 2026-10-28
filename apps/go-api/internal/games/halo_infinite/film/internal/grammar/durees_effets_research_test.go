//go:build research

package grammar

// durees_effets_research_test.go — PLAN « DUREES ET POSITIONS DES EFFETS » (2026-10-28), ETAPE A :
// LE FILM PORTE-T-IL, PAR FAMILLE D EQUIPEMENT, UN OBJET D EFFET (ne a l usage ou a l impact, distinct
// de l objet que la pile i28 suit) DONT LA NAISSANCE, LA SUPPRESSION ET LA POSITION DONNENT LA DUREE ET
// LA POSITION DE L EFFET ?
//
// TAILLE (701 lignes, au-dela du seuil de 500) : instrument de mesure hors build de production
// (`research`), une seule passe et ses sorties TSV ; le decouper disperserait une mesure qui se relit d un bloc.
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
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
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

// deMemoire echantillonne le tas et la memoire obtenue de l OS pendant l instrument.
type deMemoire struct {
	stop, done     chan struct{}
	maxTas, maxSys uint64
}

func deSurveillerLaMemoire() *deMemoire {
	m := &deMemoire{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(m.done)
		tk := time.NewTicker(200 * time.Millisecond)
		defer tk.Stop()
		for {
			var s runtime.MemStats
			runtime.ReadMemStats(&s)
			m.maxTas, m.maxSys = max(m.maxTas, s.HeapAlloc), max(m.maxSys, s.Sys)
			select {
			case <-m.stop:
				return
			case <-tk.C:
			}
		}
	}()
	return m
}

func (m *deMemoire) arreter() (tas, sys uint64) {
	close(m.stop)
	<-m.done
	return m.maxTas, m.maxSys
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

func envOu(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func deEcrire(t *testing.T, chemin string, entete string, ecrire func(w *bufio.Writer)) {
	f, err := os.Create(chemin)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintln(w, entete)
	ecrire(w)
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
}

// deEcrireUsages : une ligne par usage.
func deEcrireUsages(t *testing.T, out, id string, origine uint64, usages []*deUsage) {
	deEcrire(t, filepath.Join(out, id+".usages.tsv"),
		"film\tusage\tfamille\trang\tobjet\tporteur\tt_ms\tn_suivis\tsuivi_ti\tsuivi_nomme_ms\tsuivi_fin_ms\t"+
			"suivi_duree_ms\tcreation_x\tcreation_y\tcreation_z\trepos_etat\trepos_x\trepos_y\trepos_z",
		func(w *bufio.Writer) {
			for _, u := range usages {
				nomme, fin, duree, ti := "", "", "", ""
				cr, rp, etat := "\t\t", "\t\t", ""
				if len(u.suivis) > 0 {
					s := u.suivis[0]
					ti, nomme = strconv.Itoa(s.ti), fmt.Sprintf("%.1f", deMs(s.nomme, origine))
					if s.fin > 0 {
						fin = fmt.Sprintf("%.1f", deMs(s.fin, origine))
						duree = fmt.Sprintf("%.0f", float64(s.fin-s.nomme)/1000)
					}
					cr, etat = dePos(s.creation, s.aCreation), s.reposEtat
					rp = dePos(s.repos, s.reposEtat != "absent")
				}
				fmt.Fprintf(w, "%s\t%d\t%s\t%d\t%d\t%d\t%.1f\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", id, u.idx, u.famille,
					u.rang, u.objet.Slot, u.porteur, deMs(u.ts, origine), len(u.suivis), ti, nomme, fin, duree, cr,
					etat, rp)
			}
		})
}

// deEcrireCandidats : une ligne par (usage, instant, vie candidate).
func deEcrireCandidats(t *testing.T, out, id string, origine uint64, cands []deCandidat,
	pos map[deCleNaissance][3]float32) {
	deEcrire(t, filepath.Join(out, id+".candidats.tsv"),
		"film\tusage\tfamille\trole\tautres_usages\tti\tslot\tgen\toff_usage_ms\toff_fin_ms\tduree_ms\tx\ty\tz\t"+
			"dist_repos_3d\tdist_repos_h\tdist_creation_suivi_3d",
		func(w *bufio.Writer) {
			for _, c := range cands {
				v, u := c.vie, c.u
				decal := int64(0)
				switch c.role {
				case "temoin+":
					decal = deTemoinUS
				case "temoin-":
					decal = -deTemoinUS
				}
				offF, duree := "", "-1"
				var s *deSuivi
				if len(u.suivis) > 0 {
					s = u.suivis[0]
					if s.fin > 0 {
						offF = fmt.Sprintf("%.0f", float64(int64(v.ne)-int64(s.fin)-decal)/1000)
					}
				}
				if v.fin > 0 {
					duree = fmt.Sprintf("%.0f", float64(v.fin-v.ne)/1000)
				}
				p, aP := pos[deCleNaissance{v.slot, v.gen, v.ne}]
				dr3, drh, dc := "", "", ""
				if aP && s != nil && c.role == "reel" {
					if s.reposEtat != "absent" {
						d3, dh := deDist(p, s.repos)
						dr3, drh = fmt.Sprintf("%.2f", d3), fmt.Sprintf("%.2f", dh)
					}
					if s.aCreation {
						d3, _ := deDist(p, s.creation)
						dc = fmt.Sprintf("%.2f", d3)
					}
				}
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%d\t%d\t%d\t%d\t%.0f\t%s\t%s\t%s\t%s\t%s\n", id, u.idx, u.famille,
					c.role, c.autres, v.ti, v.slot, v.gen, float64(int64(v.ne)-int64(u.ts)-decal)/1000, offF, duree,
					dePos(p, aP), dr3+"\t"+drh, dc)
			}
		})
}

// deEcrireExtra : toutes les vies des archetypes extra, avec l usage et l objet suivi les plus proches.
func deEcrireExtra(t *testing.T, out, id string, origine uint64, vies []*cfVie, extra map[int16]bool,
	usages []*deUsage, suivis []*deSuivi, pos map[deCleNaissance][3]float32) {
	deEcrire(t, filepath.Join(out, id+".ti_extra.tsv"),
		"film\tti\tslot\tgen\tne_ms\tduree_ms\tx\ty\tz\tusage_proche\tfamille\toff_usage_ms\tsuivi_proche_nomme_off_ms\t"+
			"suivi_proche_fin_off_ms",
		func(w *bufio.Writer) {
			for _, v := range vies {
				if !extra[v.ti] {
					continue
				}
				duree := "-1"
				if v.fin > 0 {
					duree = fmt.Sprintf("%.0f", float64(v.fin-v.ne)/1000)
				}
				p, aP := pos[deCleNaissance{v.slot, v.gen, v.ne}]
				bu, bf, bn := -1, "?", int64(math.MaxInt64)
				for _, u := range usages {
					if d := int64(v.ne) - int64(u.ts); deAbs64(d) < deAbs64(bn) {
						bu, bf, bn = u.idx, u.famille, d
					}
				}
				offU := ""
				if bu >= 0 {
					offU = fmt.Sprintf("%.0f", float64(bn)/1000)
				}
				sn, sf := "", ""
				var mn, mf int64 = math.MaxInt64, math.MaxInt64
				for _, s := range suivis {
					if d := int64(v.ne) - int64(s.nomme); deAbs64(d) < deAbs64(mn) {
						mn = d
					}
					if s.fin > 0 {
						if d := int64(v.ne) - int64(s.fin); deAbs64(d) < deAbs64(mf) {
							mf = d
						}
					}
				}
				if mn != math.MaxInt64 {
					sn = fmt.Sprintf("%.0f", float64(mn)/1000)
				}
				if mf != math.MaxInt64 {
					sf = fmt.Sprintf("%.0f", float64(mf)/1000)
				}
				fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%.1f\t%s\t%s\t%d\t%s\t%s\t%s\t%s\n", id, v.ti, v.slot, v.gen,
					deMs(v.ne, origine), duree, dePos(p, aP), bu, bf, offU, sn, sf)
			}
		})
}

func deAbs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// deEcrireArchetypes : les composants des archetypes vus parmi les candidats et les extras.
func deEcrireArchetypes(t *testing.T, out, id string, reg *Registry, tiVus map[int16]int) {
	tis := make([]int, 0, len(tiVus))
	for ti := range tiVus {
		tis = append(tis, int(ti))
	}
	sort.Ints(tis)
	deEcrire(t, filepath.Join(out, id+".archetypes.tsv"), "film\tti\tcandidats\tcomposants", func(w *bufio.Writer) {
		for _, ti := range tis {
			a, ok := reg.Archetype(ti)
			if !ok {
				continue
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", id, ti, tiVus[int16(ti)], strings.Join(a.Components, ","))
		}
	})
}

// deEcrireFenetres : une ligne par fenetre de recherche, avec ou sans candidat (le denominateur des
// parts de la synthese).
func deEcrireFenetres(t *testing.T, out, id string, fens []deFen) {
	deEcrire(t, filepath.Join(out, id+".fenetres.tsv"),
		"film\tusage\tfamille\trole\tautres_usages\tautres_famille\tcandidats\tfenetre_ms",
		func(w *bufio.Writer) {
			for _, f := range fens {
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%d\t%d\t%d\t%.0f\n", id, f.u.idx, f.u.famille, f.role, f.autres,
					f.autresRang, f.n, f.dureeMs)
			}
		})
}
