//go:build research

package grammar

// mouvement_5_3_2d_trame_research_test.go — LA VENTILATION PAR LE VRAI DECODEUR DE TRAME.
//
// # POURQUOI CE TROISIEME INSTRUMENT
//
// 5.3.2 a ventile `ti=35` avec `walkDeltaBipedPayload`, qui est un CHERCHEUR D ANCRES : il
// balaie le payload bit a bit et retient ce qui RESSEMBLE a un en-tete de bipede. Il a conclu
// `i29 unit-crouch` a 0,0 % des records delta — ce qui contredit trois documents du depot
// (« per-tick biped state … crouch »). Quand une mesure contredit le depot, c est l instrument
// qu on change.
//
// CE FICHIER UTILISE `DecodeFrameRecords` — le decodeur de trame du depot : preambule de
// paquet, puis les records DANS L ORDRE, contre un `World` amorce par les images-cles du chunk.
// C est la recette de `biped_pickup_research_test.go`, reprise telle quelle.
//
// # LA LIMITE, ECRITE
//
// `DecodeFrameRecords` saute la liste d evenements : il ne cadre que les paquets dont la liste
// est VIDE (`pay[0]&0x40 == 0`), soit 83,1 % des paquets de `bfecd02b`. Les 16,9 % restants
// portent un evenement et ne sont PAS ventiles ici.
//
// # L ETALON, ET IL EST EXIGE AVANT TOUTE CONCLUSION
//
// `i21 unit-desired-aiming-vector` doit etre vu sur une large part des records (le depot le dit
// lu par image). Sinon l instrument est faux, et le rapport le DIT au lieu de publier un chiffre.
//
//	MOUV532D_FILM=<repertoire de chunks> \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	    -run '^TestMouvement532Trame$' -count=1 -v -timeout 60m

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m532dStat porte les denominateurs de la passe.
type m532dStat struct {
	paquets, trames, records int
	tmin, tmax               uint64
	// preambule : la distribution de l octet de TETE et de la TAILLE des paquets, separee par
	// sort du paquet. SANS HYPOTHESE : on ne cherche pas un drapeau precis, on regarde si les
	// deux populations sont distinguables. Si elles ne le sont pas, la piste du preambule se
	// ferme ; si une valeur est propre aux rejetes, elle se lit ensuite chez l ecrivain.
	teteSains, teteRejetes, teteDesync map[byte]int
	tailleSains, tailleRejetes         map[int]int
}

// m532dEchec est UN record sur lequel la marche a desynchronise : son masque et l index du
// composant fautif. C EST LA POPULATION INTERESSANTE — un delta ne porte `i29` QUE quand
// l accroupi CHANGE, donc ces records sont rares, et s ils sont precisement ceux qui echouent,
// l accroupi par instant est derriere une largeur fausse.
type m532dEchec struct {
	ti     uint32
	masque uint64
	fautif int
	// composants : le nombre de composants que la marche a franchis. ZERO avec `fautif == 0`
	// et `ti == 0` est la signature d un REJET DE GENERATION, pas d une desynchronisation de
	// grammaire : `DecodeFrameRecords` pose alors `EntityTrace{DesyncAt: 0}` SANS toucher
	// `TypeIndex`. Confondre les deux fait passer un defaut de monde pour un composant fautif.
	composants int
	slot       uint32
}

// TestMouvement532Trame — LA VENTILATION ETALONNEE.
func TestMouvement532Trame(t *testing.T) {
	dir := os.Getenv("MOUV532D_FILM")
	if dir == "" {
		t.Skip("MOUV532D_FILM absent : chemin du repertoire de chunks du film attendu")
	}
	ech, echecs, st, reg := m532dLire(t, dir)
	t.Logf("FILM %s", dir)
	t.Logf("  paquets delta a liste VIDE : %d cadres, %d trames decodees sans erreur (%.1f %%)",
		st.paquets, st.trames, m532Pct(st.trames, st.paquets))
	t.Logf("  records de trame : %d au total, dont %d de ti=35 (%.1f %%)",
		st.records, len(ech), m532Pct(len(ech), st.records))
	if len(ech) == 0 {
		t.Fatalf("aucun record ti=35 : l instrument ne mesure rien")
	}
	m532dPreambule(t, st)
	m532dHistoEchecs(t, ech, echecs, reg)
	if !m532dEtalon(t, ech) {
		return
	}
	m532dCadence(t, ech, st)
	m532dIntervalles(t, ech, st)
}

// m532dEtalon exige `i21` sur une large part des records AVANT de publier quoi que ce soit.
func m532dEtalon(t *testing.T, ech []m532Ech) bool {
	t.Helper()
	var i0, i1, i21, i25 int
	for _, e := range ech {
		if e.masque&1 != 0 {
			i0++
		}
		if e.masque&(1<<1) != 0 {
			i1++
		}
		if e.masque&(1<<21) != 0 {
			i21++
		}
		if e.masque&(1<<25) != 0 {
			i25++
		}
	}
	n := len(ech)
	t.Logf("ETALON : i0 %.1f %% · i1 %.1f %% · i21 %.1f %% · i25 %.1f %% (sur %d records ti=35)",
		m532Pct(i0, n), m532Pct(i1, n), m532Pct(i21, n), m532Pct(i25, n), n)
	if m532Pct(i21, n) < 50 {
		t.Logf("ETALON REFUSE : `i21` n est vu que sur %.1f %% des records alors que le depot le "+
			"dit lu PAR IMAGE. L instrument ne cadre donc pas la trame comme il faut, et AUCUNE "+
			"cadence n est publiee — publier un chiffre ici serait refaire l erreur de 5.3.2.",
			m532Pct(i21, n))
		return false
	}
	t.Logf("ETALON TENU : la ventilation qui suit porte sur une trame correctement cadree.")
	return true
}

// m532dCadence rend, pour chaque composant suivi, sa frequence et sa cadence en records par
// seconde et par slot — la reponse a « i29 est-il par tick ? ».
func m532dCadence(t *testing.T, ech []m532Ech, st m532dStat) {
	t.Helper()
	secondes := float64(st.tmax-st.tmin) / 1e6
	slots := map[uint32]bool{}
	for _, e := range ech {
		slots[e.slot] = true
	}
	var ctl, cr, mo, po, sl, vi int
	for _, e := range ech {
		if e.aControle {
			ctl++
		}
		if e.aCrouch {
			cr++
		}
		if e.aMobilite {
			mo++
		}
		if e.aPosture {
			po++
		}
		if e.aSlide {
			sl++
		}
		if e.aVitesse {
			vi++
		}
	}
	t.Logf("CADENCE (%.1f s de film, %d slots) — frequence, puis records par seconde et par slot",
		secondes, len(slots))
	type suivi struct {
		nom string
		n   int
	}
	for _, s := range []suivi{
		{"i1  object-translational-velocity", vi},
		{"i18 unit-control", ctl},
		{"i29 unit-crouch", cr},
		{"i54 biped-mobility-action", mo},
		{"i55 biped-posture-physics", po},
		{"i62 biped-slide", sl},
	} {
		par := 0.0
		if secondes > 0 && len(slots) > 0 {
			par = float64(s.n) / secondes / float64(len(slots))
		}
		t.Logf("    %-36s %7d  %5.1f %%  %6.2f rec/s/slot", s.nom, s.n,
			m532Pct(s.n, len(ech)), par)
	}
}

// m532dInter est UN intervalle d etat reconstruit par front du booleen.
type m532dInter struct {
	slot   uint32
	t0, t1 uint64
}

// m532dIntervalles reconstruit les intervalles d accroupi par slot et publie les tags d i55.
func m532dIntervalles(t *testing.T, ech []m532Ech, st m532dStat) {
	t.Helper()
	parSlot := map[uint32][]m532Ech{}
	for _, e := range ech {
		parSlot[e.slot] = append(parSlot[e.slot], e)
	}
	slots := make([]int, 0, len(parSlot))
	for s := range parSlot {
		slots = append(slots, int(s)) //nolint:gosec // slot d entite
	}
	sort.Ints(slots)
	var accroupis []m532dInter
	tags := map[uint32]int{}
	for _, s := range slots {
		serie := parSlot[uint32(s)] //nolint:gosec // clef issue d un uint32
		sort.Slice(serie, func(a, b int) bool { return serie[a].tUS < serie[b].tUS })
		var debut uint64
		dedans := false
		for _, e := range serie {
			if e.aPosture {
				tags[e.tag]++
			}
			if !e.aCrouch {
				continue
			}
			bas := e.crouch && e.crouchQ > 512
			switch {
			case bas && !dedans:
				debut, dedans = e.tUS, true
			case !bas && dedans:
				accroupis = append(accroupis, m532dInter{uint32(s), debut, e.tUS}) //nolint:gosec // clef
				dedans = false
			}
		}
	}
	m532dPublierIntervalles(t, accroupis, len(slots), st)
	if len(tags) > 0 {
		cles := make([]int, 0, len(tags))
		for k := range tags {
			cles = append(cles, int(k)) //nolint:gosec // tag sur 2 bits
		}
		sort.Ints(cles)
		var p []string
		for _, k := range cles {
			p = append(p, fmt.Sprintf("%d:%d", k, tags[uint32(k)])) //nolint:gosec // clef
		}
		t.Logf("I55 SUR LA TRAME : tags %s", strings.Join(p, " "))
	}
}

func m532dPublierIntervalles(t *testing.T, inters []m532dInter, nbSlots int, st m532dStat) {
	t.Helper()
	t.Logf("INTERVALLES D ACCROUPI (progression > 0,5) : %d sur %d slots", len(inters), nbSlots)
	if len(inters) == 0 {
		return
	}
	durees := make([]float64, 0, len(inters))
	for _, i := range inters {
		durees = append(durees, float64(i.t1-i.t0)/1e6)
	}
	sort.Float64s(durees)
	t.Logf("    duree : mediane %.2f s · p90 %.2f s · max %.2f s",
		durees[len(durees)/2], durees[(len(durees)*9)/10], durees[len(durees)-1])
	var parts []string
	var dernier uint64
	for _, i := range inters {
		if len(parts) > 0 && i.t0 < dernier+10_000_000 {
			continue
		}
		dernier = i.t0
		parts = append(parts, fmt.Sprintf("%s->%s (slot %d)",
			m532Barre(i.t0, st.tmin), m532Barre(i.t1, st.tmin), i.slot))
		if len(parts) == 5 {
			break
		}
	}
	t.Logf("    CINQ INSTANTS ACCROUPI (temps de barre Theater) : %s", strings.Join(parts, " · "))
}

// m532dLire decode le film par `DecodeFrameRecords`, chunk par chunk.
//
//nolint:gocyclo // instrument de recherche : une seule marche, lisible de haut en bas
func m532dLire(t *testing.T, dir string) ([]m532Ech, []m532dEchec, m532dStat, *Registry) {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	cfg := fc.CadreDeBalayage()
	// LA LARGEUR D ID BAS EST UNE VALEUR DE RUNTIME, PAS UNE CONSTANTE DU FORMAT
	// (`FrameConfig.IDLowBits` le dit : « le defaut 13 n est pas une constante du format »).
	// `bpkCalibre` la mesure par film : sur `bfecd02b` elle vaut 9 (97,5 % de trames exactes)
	// contre 5,1 % au defaut. Sans elle, la trame se cadre faux et tout ce qui suit est du bruit.
	if v := os.Getenv("MOUV532D_IDLOW"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.IDLowBits = n
		}
	}
	var ech []m532Ech
	var echecs []m532dEchec
	st := m532dStat{
		teteSains: map[byte]int{}, teteRejetes: map[byte]int{}, teteDesync: map[byte]int{},
		tailleSains: map[int]int{}, tailleRejetes: map[int]int{},
	}
	// LE MONDE PERSISTE D UN CHUNK A L AUTRE, et c est une correction : un `World` remis a neuf
	// a chaque chunk perd les liaisons slot -> archetype posees par les chunks precedents, et
	// `DecodeFrameRecords` rejette alors le delta sur son test de generation SANS lire un seul
	// composant. Mesure avant correction : 13 463 rejets de generation contre 2 709
	// desynchronisations reelles — 83 % des echecs venaient du harnais, pas de la grammaire.
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m532dLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0]&0x40 != 0 {
				continue // la liste d evenements n est pas vide : hors du domaine de ce decodeur
			}
			st.paquets++
			if st.tmin == 0 || pk.TimestampUS < st.tmin {
				st.tmin = pk.TimestampUS
			}
			if pk.TimestampUS > st.tmax {
				st.tmax = pk.TimestampUS
			}
			br := LecteurSur(pay)
			recs, errD := DecodeFrameRecords(br, w, cfg)
			taille := len(pay) / 64 // classes de 64 octets
			if errD != nil {
				rejet := false
				if n := len(recs); n > 0 {
					l := recs[n-1]
					rejet = l.TypeIndex == 0 && l.DesyncAt == 0 && len(l.Trace.Comps) == 0
				}
				if rejet {
					st.teteRejetes[pay[0]]++
					st.tailleRejetes[taille]++
				} else {
					st.teteDesync[pay[0]]++
				}
				// LES RECORDS PARTIELS SONT RENDUS AVEC L ERREUR : le dernier porte le masque et
				// l index du composant sur lequel la marche s est arretee. C est la matiere de
				// l histogramme des echecs.
				if n := len(recs); n > 0 {
					last := recs[n-1]
					echecs = append(echecs, m532dEchec{ti: last.TypeIndex, masque: last.Trace.Mask,
						fautif: last.DesyncAt, composants: len(last.Trace.Comps), slot: last.Slot})
				}
				continue
			}
			st.teteSains[pay[0]]++
			st.tailleSains[taille]++
			st.trames++
			st.records += len(recs)
			for _, r := range recs {
				if r.TypeIndex != BipedTypeIndex {
					continue
				}
				ech = append(ech, m532dEch(pay, r, pk.TimestampUS))
			}
		}
	}
	return ech, echecs, st, reg
}

// m532dLierMonde ajoute au monde les liaisons portees par les images-cles d un chunk. Sans
// elles, le decodeur de trame ne sait pas quel archetype porte un slot, et chaque record delta
// est rejete avant toute lecture.
func m532dLierMonde(w *World, data []byte, pks []FilmPacket) {
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			// LIAISON PAR SLOT, GENERATION NEUTRALISEE (`world.go` prevoit `BindWildcard` pour
			// « une liaison dont la GENERATION est INCONNUE »). MESURE : ne deplace AUCUN
			// chiffre — la cause des rejets est ailleurs (note 5.3, § 2 octies).
			w.BindWildcard(uint32(r.Slot), uint32(r.TI)) //nolint:gosec // valeurs de registre
		}
	}
}
