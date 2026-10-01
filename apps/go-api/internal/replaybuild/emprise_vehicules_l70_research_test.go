package replaybuild

// emprise_vehicules_l70_research_test.go — LOT L7.0 DU PLAN
// `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md` : LES MESURES DE LA RESSOURCE « VEHICULES » sur les
// huit soirees temoins, sur un document construit EN MEMOIRE par le code actuel.
//
// # CE QUE CE FICHIER MESURE
//
//	prises (D2)        par camp : une prise = le premier episode d'un joueur d'un camp dans une vie
//	                   de vehicule apres que la vie a appartenu a l'autre camp (ou a personne) ;
//	                   conducteur d'abord si deux episodes commencent a la meme image ;
//	temps a bord (D4)  somme des episodes, par camp et par joueur ;
//	proximity          part des episodes `src = proximity` (seuil du plan : <= 50 %) ;
//	frags (D5)         les frags de classe vehicule ou tourelle lus par le volet base
//	                   (`platform/duckdb`, `TestEmpriseL70Frags`), contre les episodes publies de
//	                   leur tueur (seuil du plan : >= 90 % pendant un episode de l'escouade).
//
// # L'HORLOGE : CELLE DU LOT V0.1, ET ELLE EST RELUE, PAS REINVENTEE
//
// Frame f du document -> film µs = origine + f × pas (origine = premier paquet de position de la
// cuisson) ; horloge du MATCH = film ms − `coverage.bridge.deathOffsetMs` (le calage publie) —
// celle de `match_kill_events.time_ms`. Le seuil est lu en STRICT (bornes de l'episode incluses) ;
// les lectures a ±500 ms et ±1 s sont imprimees a cote, jamais substituees au verdict.
//
// CE QU'IL N'ECRIT PAS : rien hors du repertoire du lot ; les faits de film ne sont pas ranges
// (`sansFaitsPersistes`) ; aucune base n'est ouverte ici (les identites et les frags viennent des
// JSON poses par `TestEmpriseV0Identites` et `TestEmpriseL70Frags`).
//
// SANS SES VARIABLES, IL SE SAUTE :
//
//	EMPRISE_L70_FILMS=<match_id,...> EMPRISE_L70_DIR=<scratch> EMPRISE_L70_CACHE=<racine data/cache> \
//	  go test ./internal/replaybuild/ -run '^TestEmpriseL70$' -v -count=1 -timeout 60m

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/testutil"
)

// Les seuils du plan, ecrits avant la mesure.
const (
	l70SeuilFragsEnEpisode = 0.90
	l70SeuilProximiteMax   = 0.50
	l70SrcProximite        = "proximity" // valeur publiee de la constante `VehicleRideSrcProximity` de film/replay
)

// l70Squad : l'escouade temoin (xuid -> gamertag), `db_profiles.json`.
var l70Squad = map[string]string{
	"2533274823110022": "JGtm",
	"2533274858283686": "Madina97294",
	"2535469190789936": "Chocoboflor",
}

// l70Frag : un frag de classe d'engin (sortie de `TestEmpriseL70Frags`).
type l70Frag struct {
	TimeMS     int    `json:"timeMs"`
	KillerXUID string `json:"killerXuid"`
	KillerTeam *int   `json:"killerTeam"`
	Key        string `json:"key"`
	Class      string `json:"class"`
}

type l70Frags struct {
	Engins []l70Frag `json:"engins"`
}

// l70Fenetre : un episode, en ms du MATCH.
type l70Fenetre struct{ t0, t1 int64 }

// l70Prise : une prise (D2).
type l70Prise struct {
	camp   int
	xuid   string
	famile string
	src    string
}

func TestEmpriseL70(t *testing.T) {
	dir, cache := os.Getenv("EMPRISE_L70_DIR"), os.Getenv("EMPRISE_L70_CACHE")
	var films []string
	for _, f := range strings.Split(os.Getenv("EMPRISE_L70_FILMS"), ",") {
		if f = strings.TrimSpace(f); f != "" {
			films = append(films, f)
		}
	}
	if len(films) == 0 || dir == "" || cache == "" {
		t.Skip("EMPRISE_L70_FILMS, EMPRISE_L70_DIR et EMPRISE_L70_CACHE requis : instrument saute")
	}
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	// LA SENTINELLE DU DEPOT (comme `TestEmpriseV0Reference`) : une cuisson BTB monte a plusieurs Gio.
	garde := filmproc.Arm("emprise-l70", 8, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "EMPRISE L70 : empreinte %.2f Gio > plafond — arret\n", float64(pic)/(1<<30))
		os.Exit(3)
	})
	defer garde.Disarm()
	var agg l70Agregat
	for _, id := range films {
		b, err := NewBuilder(repoRoot, title.DefaultSlug)
		if err != nil {
			t.Fatalf("builder : %v", err)
		}
		b.sansFaitsPersistes = true
		court := title.FilmShortMatchID(id)
		var ident v0Identite
		blob, err := os.ReadFile(filepath.Join(dir, "k1", court+".json"))
		if err != nil || json.Unmarshal(blob, &ident) != nil {
			t.Fatalf("%s : k1 illisible (%v)", court, err)
		}
		var fr l70Frags
		blob, err = os.ReadFile(filepath.Join(dir, "frags", court+".json"))
		if err != nil || json.Unmarshal(blob, &fr) != nil {
			t.Fatalf("%s : frags illisibles (%v) — jouer TestEmpriseL70Frags d'abord", court, err)
		}
		doc, faits, _, _ := v0Document(t, b, id, ident, filmcache.ChunkDir(cache, court))
		l70Mesurer(t, &agg, court, doc, faits, ident, fr)
	}
	agg.verdict(t)
}

// l70Agregat : les totaux des huit films, pour le verdict.
type l70Agregat struct {
	episodes, proximite                        int
	fragsSquad, dansEpisode, dans500, dans1000 int
}

// l70Mesurer imprime les mesures d'un film et cumule les totaux.
func l70Mesurer(t *testing.T, agg *l70Agregat, court string, doc replay.ReplayDocument,
	faits *replay.FilmFactsFile, ident v0Identite, fr l70Frags) {
	t.Helper()
	if doc.Coverage == nil || doc.Coverage.Bridge.DeathOffsetMs == nil || faits == nil {
		t.Fatalf("%s : calage ou faits absents — l'horloge du match n'est pas reconstruisible", court)
	}
	var origine uint64
	for i := range faits.Facts.Positions {
		if ts := faits.Facts.Positions[i].TimestampUS; origine == 0 || ts < origine {
			origine = ts
		}
	}
	calage, pas := *doc.Coverage.Bridge.DeathOffsetMs, int64(doc.FrameIntervalMS)
	matchMS := func(f int) int64 { return (int64(origine)+int64(f)*pas*1000)/1000 - calage }

	campDB := map[string]int{}
	for _, p := range ident.Faits.Players {
		campDB[p.XUID] = p.TeamID
	}
	campFilm := map[string]int{}
	for _, r := range doc.Roster {
		if r.Team != nil {
			campFilm[r.XUID] = *r.Team
		}
	}
	discord := 0
	for x, c := range campFilm {
		if d, ok := campDB[x]; ok && d != c {
			discord++
		}
	}

	byKey := map[[2]uint32]*replay.VehicleTrack{}
	for i := range doc.Vehicles {
		byKey[[2]uint32{doc.Vehicles[i].Slot, doc.Vehicles[i].Gen}] = &doc.Vehicles[i]
	}
	type ride struct {
		r      replay.VehicleRide
		famile string
	}
	parVie := map[[2]uint32][]ride{}
	var pieces, pieceOrpheline, sansRide int
	for i := range doc.Vehicles {
		v := &doc.Vehicles[i]
		owner := [2]uint32{v.Slot, v.Gen}
		fam := v.Family
		if v.Part == "turret" {
			pieces++
			if v.Carrier != nil {
				if c, ok := byKey[[2]uint32{v.Carrier.Slot, v.Carrier.Gen}]; ok {
					owner, fam = [2]uint32{c.Slot, c.Gen}, c.Family
				} else {
					pieceOrpheline++
				}
			} else {
				pieceOrpheline++
			}
		}
		if len(v.Rides) == 0 {
			sansRide++
		}
		for _, r := range v.Rides {
			parVie[owner] = append(parVie[owner], ride{r, fam})
		}
	}

	var prises []l70Prise
	tempsCamp, tempsJoueur := map[int]int64{}, map[string]int64{}
	var episodes, prox, sansXUID, sansCamp int
	var tProx int64
	fenetres := map[string][]l70Fenetre{}
	for _, rides := range parVie {
		sort.SliceStable(rides, func(i, j int) bool {
			a, b := rides[i].r, rides[j].r
			if a.T0 != b.T0 {
				return a.T0 < b.T0
			}
			return l70Siege(a) < l70Siege(b)
		})
		cur := -99
		for _, e := range rides {
			r := e.r
			episodes++
			duree := int64(r.T1-r.T0) * pas
			if r.Src == l70SrcProximite {
				prox++
				tProx += duree
			}
			if r.XUID == "" {
				sansXUID++
			}
			fenetres[r.XUID] = append(fenetres[r.XUID], l70Fenetre{matchMS(r.T0), matchMS(r.T1)})
			camp, ok := campFilm[r.XUID]
			if !ok {
				camp, ok = campDB[r.XUID]
			}
			if !ok {
				sansCamp++
				continue
			}
			tempsCamp[camp] += duree
			tempsJoueur[r.XUID] += duree
			if camp != cur {
				prises = append(prises, l70Prise{camp: camp, xuid: r.XUID, famile: e.famile, src: r.Src})
				cur = camp
			}
		}
	}
	agg.episodes += episodes
	agg.proximite += prox

	priseCamp, famCamp := map[int]int{}, map[string]map[int]int{}
	for _, p := range prises {
		priseCamp[p.camp]++
		f := p.famile
		if f == "" {
			f = "(inconnue)"
		}
		if famCamp[f] == nil {
			famCamp[f] = map[int]int{}
		}
		famCamp[f][p.camp]++
	}
	t.Logf("%s  schema=%d vies=%d (sans episode %d, pieces montees %d dont orphelines %d) ; "+
		"episodes=%d dont proximity=%d (%.1f %%, %.0f s) ; sans xuid=%d, sans camp=%d ; "+
		"discordance camp film/base=%d",
		court, doc.SchemaVersion, len(doc.Vehicles), sansRide, pieces, pieceOrpheline, episodes, prox,
		l70Pct(prox, episodes), float64(tProx)/1000, sansXUID, sansCamp, discord)
	t.Logf("%s  prises par camp %v ; par famille %v ; temps a bord par camp (s) %v",
		court, priseCamp, famCamp, l70Secondes(tempsCamp))

	byTeam := map[int]int{}
	for _, f := range fr.Engins {
		if f.KillerTeam != nil {
			byTeam[*f.KillerTeam]++
		}
	}
	t.Logf("%s  frags d'engin (D5) par camp du tueur %v", court, byTeam)
	l70Frags5(t, agg, court, fr, fenetres, tempsJoueur, ident)
}

// l70Frags5 confronte les frags d'engin aux episodes de leur tueur.
func l70Frags5(t *testing.T, agg *l70Agregat, court string, fr l70Frags,
	fenetres map[string][]l70Fenetre, tempsJoueur map[string]int64, ident v0Identite) {
	t.Helper()
	type cpt struct{ n, strict, p500, p1000 int }
	parJoueur := map[string]*cpt{}
	var tout cpt
	var manques []string
	for _, f := range fr.Engins {
		dist := l70DistanceAuxEpisodes(fenetres[f.KillerXUID], int64(f.TimeMS))
		_, squad := l70Squad[f.KillerXUID]
		for _, c := range []*cpt{&tout} {
			c.n++
			if dist == 0 {
				c.strict++
			}
			if dist <= 500 {
				c.p500++
			}
			if dist <= 1000 {
				c.p1000++
			}
		}
		if !squad {
			continue
		}
		c := parJoueur[f.KillerXUID]
		if c == nil {
			c = &cpt{}
			parJoueur[f.KillerXUID] = c
		}
		c.n++
		agg.fragsSquad++
		if dist == 0 {
			c.strict++
			agg.dansEpisode++
		} else if len(manques) < 12 {
			manques = append(manques, fmt.Sprintf("%s t=%d %s (ecart %d ms, %d episode(s))",
				l70Squad[f.KillerXUID], f.TimeMS, f.Key, dist, len(fenetres[f.KillerXUID])))
		}
		if dist <= 500 {
			c.p500++
			agg.dans500++
		}
		if dist <= 1000 {
			c.p1000++
			agg.dans1000++
		}
	}
	t.Logf("%s  frags d'engin, tout le lobby : %d ; en episode strict %d (%.1f %%), ±500 ms %d, ±1 s %d",
		court, tout.n, tout.strict, l70Pct(tout.strict, tout.n), tout.p500, tout.p1000)
	presents := map[string]int{}
	for _, p := range ident.Faits.Players {
		if _, ok := l70Squad[p.XUID]; ok {
			presents[p.XUID] = p.TeamID
		}
	}
	for x, g := range l70Squad {
		camp, ici := presents[x]
		c := parJoueur[x]
		if !ici {
			continue
		}
		if c == nil {
			c = &cpt{}
		}
		t.Logf("%s  escouade %-12s camp %d : %d frags d'engin, en episode strict %d (%.1f %%), ±500 ms %d, "+
			"±1 s %d ; temps a bord %.0f s",
			court, g, camp, c.n, c.strict, l70Pct(c.strict, c.n), c.p500, c.p1000,
			float64(tempsJoueur[x])/1000)
	}
	for _, m := range manques {
		t.Logf("%s  hors episode : %s", court, m)
	}
}

// l70DistanceAuxEpisodes rend 0 quand l'instant tombe dans un episode (bornes incluses), sinon
// l'ecart en ms a l'episode le plus proche ; tres grand sans episode.
func l70DistanceAuxEpisodes(fs []l70Fenetre, tMS int64) int64 {
	best := int64(1 << 40)
	for _, f := range fs {
		var d int64
		switch {
		case tMS < f.t0:
			d = f.t0 - tMS
		case tMS > f.t1:
			d = tMS - f.t1
		}
		if d < best {
			best = d
		}
	}
	return best
}

// l70Siege : le conducteur (siege 0) d'abord, les sieges inconnus apres.
func l70Siege(r replay.VehicleRide) int {
	if r.Seat == nil {
		return 9
	}
	return *r.Seat
}

func l70Pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}

func l70Secondes(m map[int]int64) map[int]int {
	out := map[int]int{}
	for k, v := range m {
		out[k] = int(v / 1000)
	}
	return out
}

// verdict applique les deux seuils du plan sur les huit films.
func (a *l70Agregat) verdict(t *testing.T) {
	t.Helper()
	part := l70Pct(a.proximite, a.episodes) / 100
	frags := l70Pct(a.dansEpisode, a.fragsSquad) / 100
	t.Logf("VERDICT  proximity : %d / %d episodes = %.1f %% (seuil <= %.0f %%)",
		a.proximite, a.episodes, 100*part, 100*l70SeuilProximiteMax)
	t.Logf("VERDICT  frags d'engin de l'escouade en episode : strict %d / %d = %.1f %% ; ±500 ms %d ; "+
		"±1 s %d (seuil >= %.0f %%)", a.dansEpisode, a.fragsSquad, 100*frags, a.dans500, a.dans1000,
		100*l70SeuilFragsEnEpisode)
	if part > l70SeuilProximiteMax {
		t.Errorf("part proximity %.1f %% > %.0f %%", 100*part, 100*l70SeuilProximiteMax)
	}
	if a.fragsSquad > 0 && frags < l70SeuilFragsEnEpisode {
		t.Errorf("frags en episode %.1f %% < %.0f %%", 100*frags, 100*l70SeuilFragsEnEpisode)
	}
}
