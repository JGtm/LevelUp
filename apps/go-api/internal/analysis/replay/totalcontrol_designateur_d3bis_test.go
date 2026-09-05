package replay

// totalcontrol_designateur_d3bis_test.go — PHASE D3-bis : LE SEUIL (2), SEUL.
//
// Le seuil (1) a coule sur l'ATTRIBUTION — « dans quelle zone se tenait le capteur », de la
// geometrie posee sur des instants approximatifs (38,2 % sur le corpus, seuil 80 %). Celui-ci
// n'en depend pas : il ne demande AUCUNE position. Il compte des IDENTITES designees par le
// film, manche par manche.
//
// CE QUI EST TESTE, EN UNE PHRASE : par MANCHE, le film designe-t-il exactement TROIS zones ?
//
// LE PROTOCOLE — definition de « exactement 3 », tolerance de rotation, seuils, escalade — EST
// ECRIT ET COMMITE AVANT CE FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md`, section
// D3-bis). Ce qui suit l'applique, il ne le decide pas.
//
// # POURQUOI CET INSTRUMENT N'A BESOIN NI DE CARTE NI DE POSITIONS
//
// Il ne joint rien a la geometrie. Il lit deux choses et les croise sur UNE horloge :
//
//	les DESIGNATEURS  les slots `ti=13` a serie de tag 5 CHAINEE (le tag 5 non chaine est de la
//	                  contamination d'ancrage — meme predicat que le volet colline).
//	les MANCHES       `objectiveevents.RealRounds` sur les enregistrements d'entite.
//
// LES DEUX HORLOGES SONT RAMENEES A CELLE DU MOTEUR. Les lectures `ti=13` portent un horodatage
// MOTEUR ; les enregistrements d'entite sont dates depuis le PREMIER PAQUET du film.
// `ScanFilmClockOrigin` rend precisement l'horodatage moteur de ce premier paquet : la
// conversion est donc `moteur = origine + match x 1000`, sans passer par l'axe de frames du
// rejeu — donc sans bornes de carte, sans catalogue, sans document.
//
// REGIME : garde `ZONE_FILM`, un film par processus, lecture seule, AUCUNE base.
//
//	$env:ZONE_FILM="<cache>/film_chunks/0862dce4"; go test ./internal/analysis/replay/ -run TotalControlDesignateur -v

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

const (
	// tcRotationMarginMS : la fenetre EXCLUE de part et d'autre de chaque borne de manche.
	//
	// A la bascule, le jeu retire trois zones et en pose trois autres. Les emissions de cette
	// fenetre appartiennent a la ROTATION, pas a l'une des deux manches : les compter ferait
	// mecaniquement six zones par manche — un faux negatif garanti.
	tcRotationMarginMS = 2000
	// tcZonesAttendues : le cardinal exige. Total Control active TROIS zones par manche.
	tcZonesAttendues = 3
	// tcPartMinimale : part des manches exploitables qui doivent rendre exactement 3.
	tcPartMinimale = 0.80
)

// tcManche : une manche reelle, bornee sur l'horloge du MATCH.
type tcManche struct {
	num            int
	debutMS, finMS int
}

// TestTotalControlDesignateurParManche — LA MESURE. Un film par processus.
func TestTotalControlDesignateurParManche(t *testing.T) {
	dir := p2aRequireFilm(t)
	// L identite courte est le NOM DU REPERTOIRE de chunks : ces films ne sont pas au corpus
	// fige (p2aCorpus), qui ne connait ni Total Control ni leurs rosters.
	short := filepath.Base(dir)

	clockUS, err := ScanFilmClockOrigin(dir)
	if err != nil {
		t.Fatalf("%s : origine d'horloge illisible : %v", short, err)
	}
	recs := objectiveevents.StatRecords(p2aBobine(t, dir))
	manches := tcManchesOf(recs)
	if len(manches) == 0 {
		t.Logf("NON EXPLOITABLE %s : aucune manche lisible — ce film ne compte ni pour ni contre", short)
		return
	}

	sc, err := filmdec.ScanFilmManagedProperties(dir)
	if err != nil {
		t.Fatalf("%s : proprietes ti=13 illisibles : %v", short, err)
	}
	desig := tcDesignateurs(sc.Reads)
	t.Logf("%s : %d manche(s) reelle(s), %d slot(s) designateur (tag 5 chaine), %d lecture(s) ti=13, "+
		"chainage %d/%d", short, len(manches), len(desig), len(sc.Reads), sc.Chained, sc.Walked)
	if len(desig) == 0 {
		t.Logf("NON EXPLOITABLE %s : AUCUN designateur elu — ce film ne compte ni pour ni contre", short)
		return
	}

	exploitables, exactes := 0, 0
	for _, m := range manches {
		t0 := uint64(int64(clockUS) + int64(m.debutMS+tcRotationMarginMS)*1000)
		t1 := uint64(int64(clockUS) + int64(m.finMS-tcRotationMarginMS)*1000)
		if t1 <= t0 {
			t.Logf("  manche %d : plus courte que la fenetre de rotation — ecartee", m.num)
			continue
		}
		set := tcEnsembleDesigne(desig, t0, t1)
		if len(set) == 0 {
			t.Logf("  manche %d : aucune emission de designateur hors rotation — ecartee", m.num)
			continue
		}
		exploitables++
		if len(set) == tcZonesAttendues {
			exactes++
		}
		t.Logf("  manche %d [%d ; %d] ms : %d zone(s) designee(s) %v",
			m.num, m.debutMS, m.finMS, len(set), set)
	}
	if exploitables == 0 {
		t.Logf("NON EXPLOITABLE %s : aucune manche confrontable", short)
		return
	}
	part := float64(exactes) / float64(exploitables)
	verdict := "SOUS LE SEUIL"
	if part >= tcPartMinimale {
		verdict = "TENU"
	}
	t.Logf("SIGNAL %s : %d/%d manche(s) a exactement %d zones = %.1f %% (seuil %.0f %%) — %s",
		short, exactes, exploitables, tcZonesAttendues, 100*part, 100*tcPartMinimale, verdict)
}

// tcManchesOf borne chaque manche REELLE sur l'horloge du match.
//
// LES MANCHES FANTOMES SONT ECARTEES par `RealRounds` — les cumuler ferait exploser les
// compteurs (mesure du lot A : un score d'equipe passait de 1 a 2 104).
func tcManchesOf(recs []objectiveevents.StatRecord) []tcManche {
	reelles := objectiveevents.RealRounds(recs)
	bornes := map[int][2]int{}
	for _, r := range recs {
		if !reelles[r.Round] {
			continue
		}
		b, vu := bornes[r.Round]
		if !vu {
			bornes[r.Round] = [2]int{r.TimeMS, r.TimeMS}
			continue
		}
		if r.TimeMS < b[0] {
			b[0] = r.TimeMS
		}
		if r.TimeMS > b[1] {
			b[1] = r.TimeMS
		}
		bornes[r.Round] = b
	}
	out := make([]tcManche, 0, len(bornes))
	for n, b := range bornes {
		out = append(out, tcManche{num: n, debutMS: b[0], finMS: b[1]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].debutMS < out[j].debutMS })
	return out
}

// tcDesignateurs rend, par slot, la serie des valeurs de tag 5 CHAINEES.
//
// LE CHAINAGE EST LA GARDE : sur un KOTH de reference, le canal par joueur chaine a 33 % contre
// 97 % pour le canal scalaire — le tag 5 non chaine des slots combles est de la contamination
// d'ancrage, pas une designation.
func tcDesignateurs(reads []filmdec.ManagedPropertyRead) map[uint32][]filmdec.ManagedPropertyRead {
	out := map[uint32][]filmdec.ManagedPropertyRead{}
	for _, r := range reads {
		if r.Field != filmdec.ManagedPropertyScalar || !r.HasValue || !r.Chained {
			continue
		}
		if r.Tag != filmdec.ManagedPropertyTagStringID {
			continue
		}
		out[r.Slot] = append(out[r.Slot], r)
	}
	return out
}

// tcEnsembleDesigne rend les valeurs DISTINCTES en vigueur dans la fenetre, tous slots
// confondus — c'est l'ensemble dont on compte le cardinal.
//
// LA VALEUR ZERO N'EST PAS UNE DESIGNATION : un slot qui emet zero ne nomme rien.
func tcEnsembleDesigne(desig map[uint32][]filmdec.ManagedPropertyRead, t0, t1 uint64) []uint64 {
	vu := map[uint64]bool{}
	for _, serie := range desig {
		for _, r := range serie {
			if r.TimestampUS < t0 || r.TimestampUS > t1 || r.Value == 0 {
				continue
			}
			vu[r.Value] = true
		}
	}
	out := make([]uint64, 0, len(vu))
	for v := range vu {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
