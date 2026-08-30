package replay

// visee_chronologie_research_test.go — LA CHRONOLOGIE DE ZOOM RELEVEE A LA MAIN, CONFRONTEE A
// TOUS LES FLUX DU FILM.
//
// LA VERITE TERRAIN. L'utilisateur a relu le match 00162144 dans Theater EN PREMIERE PERSONNE
// (2026-08-30) et notee les periodes a la lunette, a la seconde pres (horloge de la video, qui
// est celle du feed — le frag Counter-snipe est bien a 0:46 = 46 338 ms) :
//
//	Nilton410    0:41 -> frag a 0:46 ; 0:49 -> ~0:52 ; bref a ~1:01 ; bref vers ~1:08 ;
//	             ~1:11 -> ~1:13 ; encore a ~1:25.
//	Madina97294  0:45 -> sa mort a 0:46.
//
// Et une observation NEGATIVE precieuse : en TROISIEME personne, rien ne change — la question de
// l'epaulement etait donc mal posee ; le signal est du cote de la vue, pas de la pose du bipede.
//
// Douze transitions etiquetees en ~50 s la ou l'oracle des medailles ne donnait qu'UN instant
// par kill : de quoi tester d'un seul coup toutes les hypotheses restantes. TROIS MESURES :
//
//	A. SILENCE CAMERA — le flux type 97 (camera POV, phase 4) se tait-il pendant les periodes
//	   zoomees de Nilton ? Taux de paquets pendant / hors periodes.
//	B. ALIGNEMENT DES TYPES — pour chaque type d'event, ses paquets tombent-ils sur les
//	   transitions (fenetre ±1,2 s, la precision de lecture video) plus souvent que son debit
//	   de fond ne l'explique ?
//	C. EMISSIONS DU BIPEDE DE NILTON — pour chaque composant du masque de ses records delta
//	   (MaskBits), ses emissions s'alignent-elles sur les transitions ?
//
// SEUILS ECRITS AVANT LA MESURE : un signal est CANDIDAT s'il couvre >= 8 des 12 transitions
// ET que sa densite dans les fenetres vaut >= 3 fois son debit de fond. Ces seuils s'appliquent
// aux trois mesures.
//
// SOUS GARDE (CHRONO_FILM, qui doit pointer 00162144 — la chronologie est celle de CE film).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 CHRONO_FILM=<repo>/data/cache/film_chunks/00162144 \
//	  go test ./internal/analysis/replay/ -run TestViseeChronologie -v -timeout 30m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const (
	chronoFilmEnv   = "CHRONO_FILM"
	chronoFenetreMS = 1200 // ± autour d'une transition : la precision de lecture video
	chronoCouvMin   = 8    // transitions couvertes minimales (sur 12)
	chronoEnrichMin = 3.0  // densite fenetre / debit de fond
	chronoGT        = "Nilton410"
	chronoGTVictime = "Madina97294"
)

// chronoEpisodes : les periodes zoomees de Nilton410, en SECONDES d'horloge du feed.
var chronoEpisodes = [][2]float64{
	{41, 46.3}, {49, 52}, {61, 61.8}, {68, 68.8}, {71, 73}, {85, 86},
}

// chronoEpisodeMadina : la periode unique de la victime.
var chronoEpisodeMadina = [2]float64{45, 46.3}

// TestViseeChronologie confronte les trois mesures a la chronologie relevee.
func TestViseeChronologie(t *testing.T) {
	dir := os.Getenv(chronoFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", chronoFilmEnv)
	}
	if filepath.Base(dir) != "00162144" {
		t.Fatalf("la chronologie relevee est celle de 00162144 ; film fourni : %s", filepath.Base(dir))
	}
	release := filmdec.LockProcessDecode()
	defer release()

	xuid := chronoXUID(t, dir, chronoGT)
	xuidV := chronoXUID(t, dir, chronoGTVictime)
	t.Logf("IDENTITES — %s xuid=%d · %s xuid=%d", chronoGT, xuid, chronoGTVictime, xuidV)

	scan := filmdec.DefaultScanFilmOptions()
	scan.CaptureDirs = true
	scan.QuantaOnly = true
	debut := time.Now()
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts : %v", err)
	}
	lives := buildLifeSpans(indexBySlot(pos))
	off, matched := bestDeathOffset(lives, deaths)
	if nameLivesByDeaths(lives, deaths, off) == 0 {
		t.Fatalf("pont slot->xuid vide")
	}
	t.Logf("COUT — %d positions en %s · decalage feed->film %d ms (%d fins de vie appariees)",
		len(pos), time.Since(debut).Round(time.Millisecond), off, matched)

	// CONTROLE CROISE DU RECALAGE : le vote de mode sur les kills (tirRecale, valide a 58 ms
	// pres sur 000d5950) doit tomber sur le meme ecart que le pont des morts. S'ils divergent,
	// AUCUNE des mesures n'est interpretable.
	feed, okFeed := tirLitFeed(dir)
	types, records105 := canalLitTypes(dir)
	if okFeed && len(feed.kills) > 0 {
		if off2, ok2 := tirRecale(feed.kills, records105); ok2 {
			t.Logf("RECALAGE — pont des morts %d ms · vote de mode des kills %d ms · ecart %d ms",
				off, off2, off-off2)
		} else {
			t.Logf("RECALAGE — vote de mode NON CONCLUANT sur ce film (pont des morts seul : %d ms)", off)
		}
	}
	// PROPAGATION AUX FRAGMENTS : nameLivesByDeaths ne nomme une vie que par la mort qui la
	// TERMINE ; les fragments anterieurs de la MEME vie physique (meme slot, coupures de
	// tracking) restent anonymes — et la plage etiquetee (debut de match, avant sa premiere
	// mort) tombait justement dedans. Un slot ne se recycle qu'apres mort + respawn : un
	// fragment de meme slot qui s'enchaine (trou <= 30 s) porte donc le meme joueur.
	for propage := true; propage; {
		propage = false
		for i := range lives {
			if lives[i].xuid != 0 {
				continue
			}
			for j := range lives {
				if lives[j].xuid == 0 || lives[j].slot != lives[i].slot {
					continue
				}
				ecart := lives[j].from - lives[i].to
				if ecart < 0 {
					ecart = lives[i].from - lives[j].to
				}
				if ecart >= 0 && ecart <= 30_000_000 {
					lives[i].xuid = lives[j].xuid
					propage = true
					break
				}
			}
		}
	}
	// Vies et fragments de Nilton (diagnostic de nommage).
	for _, l := range lives {
		if l.xuid == xuid {
			t.Logf("  vie %s : slot %d, film [%.1f ; %.1f] s", chronoGT, l.slot,
				float64(l.from)/1e6, float64(l.to)/1e6)
		}
	}
	t.Log("  fragments ANONYMES restants dans la plage etiquetee :")
	for _, l := range lives {
		if l.xuid == 0 && float64(l.to)/1e6 > 1190 && float64(l.from)/1e6 < 1280 {
			t.Logf("    slot %d, film [%.1f ; %.1f] s", l.slot, float64(l.from)/1e6, float64(l.to)/1e6)
		}
	}

	episodes := chronoVersFilm(chronoEpisodes, off)
	transitions := chronoTransitions(episodes)

	chronoMesureCamera(t, types, episodes)
	chronoMesureTypes(t, types, transitions)
	chronoMesureComposants(t, pos, lives, xuid, episodes, transitions)
	chronoMesure114(t, dir, episodes, transitions)
}

// chronoMesure114 — le type 114 (« biped_board_vehicle ») couvre 9/12 transitions sur une carte
// SANS vehicule ; sa charge utile est un seul R(6) (siege), l'entite vit dans le preambule
// (bits 9..35, cle opaque ici). Si la lunette est un « siege » de l'arme, les paquets des mises
// en lunette et ceux des sorties porteront des valeurs systematiquement differentes, avec une
// cle d'entite commune. Dump brut de la plage etiquetee ; la lecture se fait a l'oeil AVANT
// d'ecrire un decodeur — c'est un instrument d'observation, pas encore de verdict.
func chronoMesure114(t *testing.T, dir string, eps [][2]int64, trans []int64) {
	t.Helper()
	n := filmdec.CountFilmChunks(dir)
	type rec114 struct {
		tMS          int64
		cle          uint32
		siege        uint32
		bitsEnvelope uint32
		taille       int
	}
	var dansPlage []rec114
	comptesSiege := map[uint32]int{}
	total := 0
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 8 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != 114 {
				continue
			}
			total++
			r := rec114{
				tMS:          int64(p.TimestampUS / 1000),
				bitsEnvelope: filmdec.ReadBitsAtForDiag(pay, 7, 2),
				cle:          filmdec.ReadBitsAtForDiag(pay, 9, 27),
				siege:        filmdec.ReadBitsAtForDiag(pay, 36, 6),
				taille:       len(pay),
			}
			comptesSiege[r.siege]++
			if r.tMS >= eps[0][0]-8000 && r.tMS <= eps[len(eps)-1][1]+8000 {
				dansPlage = append(dansPlage, r)
			}
		}
	}
	t.Logf("D. TYPE 114 — %d paquets au total ; valeurs du R(6) a bit 36 (film entier) :", total)
	var sieges []uint32
	for sg := range comptesSiege {
		sieges = append(sieges, sg)
	}
	sort.Slice(sieges, func(i, j int) bool { return comptesSiege[sieges[i]] > comptesSiege[sieges[j]] })
	for i, sg := range sieges {
		if i == 8 {
			break
		}
		t.Logf("    R(6)=%2d : %d paquets", sg, comptesSiege[sg])
	}
	sort.Slice(dansPlage, func(i, j int) bool { return dansPlage[i].tMS < dansPlage[j].tMS })
	t.Logf("  paquets 114 dans la plage etiquetee (Δ = ecart a la transition la plus proche,"+
		" in/out d'apres la chronologie) : %d", len(dansPlage))
	for _, r := range dansPlage {
		best, bd, sens := 0, int64(1<<62), "?"
		for i, tr := range trans {
			d := r.tMS - tr
			if d < 0 {
				d = -d
			}
			if d < bd {
				best, bd = i, d
			}
		}
		if best%2 == 0 {
			sens = "IN "
		} else {
			sens = "OUT"
		}
		t.Logf("    t=%d ms · Δ=%5d ms (%s) · env2=%d · cle=%07x · R6=%d · %d o",
			r.tMS, bd, sens, r.bitsEnvelope, r.cle, r.siege, r.taille)
	}
	t.Log("  bits 0..71 alignes (le motif de variance dessine les champs) :")
	for _, r := range dansPlage {
		_ = r
	}
	compte := 0
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 10 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != 114 {
				continue
			}
			tMS := int64(p.TimestampUS / 1000)
			if tMS < eps[0][0]-8000 || tMS > eps[len(eps)-1][1]+8000 {
				continue
			}
			var sb strings.Builder
			for b := 0; b < 72; b++ {
				if b == 7 || b == 8 || b == 24 || b == 40 || b == 56 {
					sb.WriteByte(' ')
				}
				sb.WriteByte('0' + byte(filmdec.ReadBitsAtForDiag(pay, b, 1)))
			}
			t.Logf("    t=%d : %s", tMS, sb.String())
			compte++
		}
	}
}

// chronoXUID resout un gamertag en xuid via le feed (kills, morts et medailles le portent).
func chronoXUID(t *testing.T, dir, gt string) uint64 {
	t.Helper()
	n := filmdec.CountFilmChunks(dir)
	raw, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		t.Fatalf("chunk d'evenements : %v", err)
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		t.Fatalf("feed illisible : %v", err)
	}
	for _, e := range evs {
		if e.Gamertag == gt {
			return e.XUID
		}
	}
	t.Fatalf("gamertag %q absent du feed", gt)
	return 0
}

// chronoVersFilm convertit les episodes feed (secondes) en bornes film (ms).
func chronoVersFilm(eps [][2]float64, off int64) [][2]int64 {
	out := make([][2]int64, 0, len(eps))
	for _, e := range eps {
		out = append(out, [2]int64{int64(e[0]*1000) + off, int64(e[1]*1000) + off})
	}
	return out
}

// chronoTransitions rend les 2N instants de bascule (entrees puis sorties confondues).
func chronoTransitions(eps [][2]int64) []int64 {
	var out []int64
	for _, e := range eps {
		out = append(out, e[0], e[1])
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func chronoDansEpisode(eps [][2]int64, tMS int64) bool {
	for _, e := range eps {
		if tMS >= e[0] && tMS <= e[1] {
			return true
		}
	}
	return false
}

// chronoMesureCamera — mesure A : le debit du type 97 pendant vs hors des periodes zoomees.
func chronoMesureCamera(t *testing.T, types [][2]int64, eps [][2]int64) {
	t.Helper()
	var dedans, total int
	var dureeMS int64
	for _, e := range eps {
		dureeMS += e[1] - e[0]
	}
	debutFen, finFen := eps[0][0]-10000, eps[len(eps)-1][1]+10000
	var voisinage int
	for _, tp := range types {
		if tp[0] != 97 {
			continue
		}
		total++
		if chronoDansEpisode(eps, tp[1]) {
			dedans++
		}
		if tp[1] >= debutFen && tp[1] <= finFen {
			voisinage++
		}
	}
	dureeVoisinageMS := finFen - debutFen - dureeMS
	tauxDedans := float64(dedans) / (float64(dureeMS) / 1000)
	tauxDehors := float64(voisinage-dedans) / (float64(dureeVoisinageMS) / 1000)
	t.Logf("A. CAMERA 97 — %d paquets dans les %0.1f s zoomees (%.2f/s) contre %.2f/s dans le"+
		" voisinage non zoome (%d paquets film entier)", dedans, float64(dureeMS)/1000,
		tauxDedans, tauxDehors, total)
	switch {
	case dedans == 0 && tauxDehors > 0.3:
		t.Log("   LECTURE : SILENCE TOTAL pendant les periodes zoomees, flux actif autour —" +
			" compatible avec « la camera se tait a la lunette ».")
	case tauxDehors > 0 && tauxDedans < tauxDehors/3:
		t.Log("   LECTURE : forte rarefaction pendant le zoom (facteur >= 3).")
	default:
		t.Log("   LECTURE : pas de contraste net sur CE film — le silence camera n'est pas" +
			" confirme a l'echelle d'un joueur.")
	}
}

// chronoMesureTypes — mesure B : alignement de chaque type d'event sur les 12 transitions.
func chronoMesureTypes(t *testing.T, types [][2]int64, trans []int64) {
	t.Helper()
	debutFilm, finFilm := types[0][1], types[len(types)-1][1]
	dureeS := float64(finFilm-debutFilm) / 1000
	if dureeS <= 0 {
		t.Fatalf("duree de film invalide")
	}
	var candidats int
	for ty := 0; ty < 128; ty++ {
		var totale int
		couverts := map[int]bool{}
		var dansFen int
		for _, tp := range types {
			if tp[0] != int64(ty) {
				continue
			}
			totale++
			for i, tr := range trans {
				if tp[1] >= tr-chronoFenetreMS && tp[1] <= tr+chronoFenetreMS {
					couverts[i] = true
					dansFen++
					break
				}
			}
		}
		if totale == 0 || len(couverts) < 6 {
			continue
		}
		fenS := float64(len(trans)) * 2 * float64(chronoFenetreMS) / 1000
		enrich := (float64(dansFen) / fenS) / (float64(totale) / dureeS)
		if len(couverts) >= 6 && enrich >= 1.5 && (len(couverts) < chronoCouvMin || enrich < chronoEnrichMin) {
			t.Logf("B. type %3d : sous les seuils mais notable — %d/%d transitions, x%0.1f (%d paquets)",
				ty, len(couverts), len(trans), enrich, totale)
		}
		if len(couverts) >= chronoCouvMin && enrich >= chronoEnrichMin {
			candidats++
			t.Logf("B. type %3d : CANDIDAT — %d/%d transitions couvertes, enrichissement x%0.1f"+
				" (%d paquets au total)", ty, len(couverts), len(trans), enrich, totale)
		}
	}
	if candidats == 0 {
		t.Logf("B. TYPES — aucun type d'event ne s'aligne sur les %d transitions aux seuils"+
			" declares (>= %d couvertes, enrichissement >= x%0.0f).",
			len(trans), chronoCouvMin, chronoEnrichMin)
	}
}

// chronoMesureComposants — mesure C : les emissions de chaque composant du bipede de Nilton.
func chronoMesureComposants(t *testing.T, pos []filmdec.BipedPosition, lives []lifeSpan,
	xuid uint64, eps [][2]int64, trans []int64) {
	t.Helper()
	dansVie := func(tMS int64) bool {
		us := tMS * 1000
		for _, l := range lives {
			if l.xuid == xuid && us >= l.from && us <= l.to {
				return true
			}
		}
		return false
	}
	slotOK := func(p filmdec.BipedPosition) bool {
		us := int64(p.TimestampUS)
		for _, l := range lives {
			if l.xuid == xuid && l.slot == p.Slot && us >= l.from && us <= l.to {
				return true
			}
		}
		return false
	}
	var dureeVieS float64
	for _, l := range lives {
		if l.xuid == xuid {
			dureeVieS += float64(l.to-l.from) / 1e6
		}
	}
	var candidats int
	var totalRecords int
	compTot := [64]int{}
	compFen := [64]int{}
	compCouv := [64]map[int]bool{}
	for _, p := range pos {
		if !slotOK(p) {
			continue
		}
		totalRecords++
		tMS := int64(p.TimestampUS / 1000)
		for b := 0; b < 64; b++ {
			if p.MaskBits>>uint(b)&1 == 0 {
				continue
			}
			compTot[b]++
			for i, tr := range trans {
				if tMS >= tr-chronoFenetreMS && tMS <= tr+chronoFenetreMS {
					compFen[b]++
					if compCouv[b] == nil {
						compCouv[b] = map[int]bool{}
					}
					compCouv[b][i] = true
					break
				}
			}
		}
	}
	_ = dansVie
	t.Logf("C. BIPEDE — %d records de %s (%.0f s de vies)", totalRecords, chronoGT, dureeVieS)
	fenS := float64(len(trans)) * 2 * float64(chronoFenetreMS) / 1000
	for b := 0; b < 64; b++ {
		if compTot[b] == 0 || len(compCouv[b]) < chronoCouvMin {
			continue
		}
		enrich := (float64(compFen[b]) / fenS) / (float64(compTot[b]) / dureeVieS)
		if enrich >= chronoEnrichMin {
			candidats++
			t.Logf("C. composant i%d : CANDIDAT — %d/%d transitions couvertes,"+
				" enrichissement x%0.1f (%d emissions)", b, len(compCouv[b]), len(trans),
				enrich, compTot[b])
		}
	}
	if candidats == 0 {
		t.Logf("C. BIPEDE — aucun composant ne s'aligne sur les transitions aux seuils declares.")
		var rares []string
		for b := 0; b < 64; b++ {
			if compTot[b] > 0 && compTot[b] < 60 {
				rares = append(rares, fmt.Sprintf("i%d=%d(fen %d, couv %d)",
					b, compTot[b], compFen[b], len(compCouv[b])))
			}
		}
		t.Logf("   composants RARES de ses records (diagnostic) : %s", strings.Join(rares, " "))
	}
}
