package replay

// assaut_a1_identite_test.go — LOT A, PHASE A1 : L'IDENTITE DE L'OBJET BOMBE.
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER (`registre_film/A_PROTOCOLE.md`, §3).
// La recette est celle du drapeau (2026-08-18) et du crane (D4, 2026-08-27), TELLE QUELLE :
// parmi les creations `ti=42` que le catalogue d'armes ECARTE, un mot de 32 bits qui naisse
// a un site `assault_bomb` du catalogue de carte ET qui coincide avec un debut de manche ou
// une remise en jeu. Seuls changent le role de socle interroge et l'oracle temporel — sans
// evenement nomme d'Assaut au statborg, les classes temporelles sont les DEBUTS DE MANCHE
// (manches BRUTES, §2 du protocole) et les EXPLOSIONS datees par le score de mode.
//
// CE QUE LE §1 DU PROTOCOLE ETABLIT D'AVANCE : le catalogue d'objectifs ne porte AUCUN site
// `assault_bomb` pour les cartes du corpus. Le gate A1.3 ne peut donc pas etre TENU — cet
// instrument tourne pour CHIFFRER le `[!]` : denominateurs du balayage, et chaque jambe du
// critere publiee separement (aucun candidat ne s'elit sans la jambe du site).
//
// REGIME : gardes `ATT_FILM` + `ASSAUT_FILM`, UN FILM PAR PROCESSUS, lecture seule, AUCUNE
// base.
//
//	$env:ATT_FILM="<repo>/data/cache"; $env:ASSAUT_FILM="35b75a31"
//	go test ./internal/analysis/replay/ -run AssautA1Identite -v

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/filmproc"
)

const (
	// a1DebutMancheMS : une creation est « au debut de manche » si elle suit le premier
	// enregistrement d'une manche BRUTE d'au plus 5 000 ms (protocole §3, ecrit avant mesure).
	a1DebutMancheMS = 5000
	// a1RemiseMaxMS : une creation est « a la remise en jeu » si elle suit une explosion
	// datee de [0, 15 000] ms (protocole §3 : le delai moteur de reapparition n'est pas connu
	// d'avance, la fenetre est large mais bornee).
	a1RemiseMaxMS = 15000
)

// a1Candidat resume UN mot de 32 bits ecarte du catalogue d'armes.
type a1Candidat struct {
	mot uint32
	// creations : combien de creations portent ce mot ; auSite : nees a <= attDrapeauRayonM
	// d'un site `assault_bomb` ; coincidentes : dans une classe temporelle du protocole ;
	// lesDeux : creations qui reunissent LES DEUX conditions (la definition du candidat).
	creations, auSite, coincidentes, lesDeux int
	// dMinM : distance minimale a un site ; tMinMS : ecart minimal a une classe temporelle
	// (MaxInt64 si aucune classe ne date ce film — et cela se dit).
	dMinM  float64
	tMinMS int64
}

// TestAssautA1Identite — la mesure du gate A1.3 sur UN film.
func TestAssautA1Identite(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(a0FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide (identifiant court du film Assaut)", a0FilmEnv)
	}
	g := filmproc.Arm("a1-assaut", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio — mesure interrompue, ce film ne compte "+
			"NI POUR NI CONTRE", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio (plafond souple %d Gio)",
			id, float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()

	src, ok := objOpenFilm(t, root, id)
	if !ok {
		t.Fatalf("%s : film absent du cache (%s=%q)", id, attFilmEnv, root)
	}

	cre, sites, ok := attCreationsEcartees(t, root, id, a0RoleSite)
	if !ok {
		t.Logf("NON EXPLOITABLE %s : bornes de quantification indisponibles. NI POUR NI CONTRE.", id)
		return
	}
	t.Logf("%s : denominateurs du balayage — %d ancres, %d acceptees, %d RESOLUES au catalogue "+
		"d'armes, %d ecartees, %d mots distincts parmi les ecartees ; %d site(s) `%s`",
		id, cre.st.Anchors, cre.st.Accepted, len(cre.connues), len(cre.ecartees), len(cre.mots),
		len(sites), a0RoleSite)
	if len(cre.connues) == 0 {
		t.Logf("NON EXPLOITABLE %s : ZERO creation resolue au catalogue d'armes — bloc MPP lu aux "+
			"mauvaises largeurs, « aucun candidat » serait une panne de lecture. NI POUR NI CONTRE.", id)
		return
	}

	debuts, explosions := a1ClassesTemporelles(t, id, src)
	clockUS, err := ScanFilmClockOrigin(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : origine d'horloge illisible : %v", id, err)
	}
	t.Logf("%s : %d debut(s) de manche brute, %d explosion(s) datee(s), origine d'horloge %d us",
		id, len(debuts), len(explosions), clockUS)

	cands := a1Resume(cre.ecartees, sites, debuts, explosions, clockUS)
	a1Publie(t, id, cands)
}

// a1ClassesTemporelles rend les deux classes du protocole §3, sur l'horloge du manifeste :
// le premier enregistrement de chaque manche BRUTE, et chaque montee du score de mode d'un
// slot d'equipe (une montee = une explosion — releve A0.3, corrobore par le score API 9/9).
func a1ClassesTemporelles(t *testing.T, id string, src *objDiskFilm) (debuts, explosions []int64) {
	t.Helper()
	recs, truncated := objectiveevents.StatRecordsCtx(context.Background(), src, id)
	if truncated {
		t.Logf("%s : enregistrements TRONQUES — classes temporelles partielles, et cela se dit", id)
	}
	premier := map[int]int{}
	type cle struct{ slot, round int }
	prev := map[cle]int64{}
	for _, r := range recs {
		if p, ok := premier[r.Round]; !ok || r.TimeMS < p {
			premier[r.Round] = r.TimeMS
		}
		if !objectiveevents.IsTeamSlot(r.Slot) {
			continue
		}
		v, ok := r.Comps[0]
		if !ok || v.A < 0 {
			continue
		}
		k := cle{r.Slot, r.Round}
		if v.A > prev[k] {
			// Une montee du score de mode = une explosion. La valeur parasite eventuelle
			// (releve A0.3 : 127 sur le film exclu) rendrait UNE fausse date : le releve brut
			// du log A0 est la reference, l'ecart se verrait.
			explosions = append(explosions, int64(r.TimeMS))
		}
		if v.A >= prev[k] {
			prev[k] = v.A
		}
	}
	for _, tMS := range premier {
		debuts = append(debuts, int64(tMS))
	}
	sort.Slice(debuts, func(i, j int) bool { return debuts[i] < debuts[j] })
	sort.Slice(explosions, func(i, j int) bool { return explosions[i] < explosions[j] })
	return debuts, explosions
}

// a1Resume mesure les deux jambes du critere pour chaque mot ecarte.
func a1Resume(ecartees []filmdec.EquipmentCreation, sites []PointObjective,
	debuts, explosions []int64, clockUS uint64) []a1Candidat {
	parMot := map[uint32][]filmdec.EquipmentCreation{}
	for _, c := range ecartees {
		parMot[uint32(c.MPPVal[filmdec.MPPWord32])] = append(
			parMot[uint32(c.MPPVal[filmdec.MPPWord32])], c)
	}
	out := make([]a1Candidat, 0, len(parMot))
	for mot, cs := range parMot {
		cand := a1Candidat{mot: mot, creations: len(cs), dMinM: math.MaxFloat64, tMinMS: math.MaxInt64}
		for _, c := range cs {
			d := attDistSocleMin(c, sites)
			if d < cand.dMinM {
				cand.dMinM = d
			}
			site := d <= attDrapeauRayonM
			if site {
				cand.auSite++
			}
			e := a1EcartClasse(c, debuts, explosions, clockUS)
			if e < cand.tMinMS {
				cand.tMinMS = e
			}
			coinc := e == 0
			if coinc {
				cand.coincidentes++
			}
			if site && coinc {
				cand.lesDeux++
			}
		}
		out = append(out, cand)
	}
	sort.Slice(out, func(i, j int) bool {
		if (out[i].lesDeux > 0) != (out[j].lesDeux > 0) {
			return out[i].lesDeux > 0
		}
		if out[i].creations != out[j].creations {
			return out[i].creations > out[j].creations
		}
		return out[i].mot < out[j].mot
	})
	return out
}

// a1EcartClasse rend 0 si la creation tombe dans une classe temporelle du protocole, sinon
// son ecart minimal (ms) au bord de la classe la plus proche — publie pour que le `[!]` se
// chiffre, jamais pour elargir la fenetre.
func a1EcartClasse(c filmdec.EquipmentCreation, debuts, explosions []int64, clockUS uint64) int64 {
	if c.TimestampUS < clockUS {
		return math.MaxInt64
	}
	at := int64(c.TimestampUS-clockUS) / 1000
	best := int64(math.MaxInt64)
	upd := func(e int64) {
		if e < best {
			best = e
		}
	}
	for _, d := range debuts {
		delta := at - d
		if delta >= 0 && delta <= a1DebutMancheMS {
			return 0
		}
		if delta < 0 {
			upd(-delta)
		} else {
			upd(delta - a1DebutMancheMS)
		}
	}
	for _, x := range explosions {
		delta := at - x
		if delta >= 0 && delta <= a1RemiseMaxMS {
			return 0
		}
		if delta < 0 {
			upd(-delta)
		} else {
			upd(delta - a1RemiseMaxMS)
		}
	}
	return best
}

// a1Publie ecrit le tableau (mots a jambe non nulle + les 10 plus frequents) et le constat
// du gate.
func a1Publie(t *testing.T, id string, cands []a1Candidat) {
	t.Helper()
	elus := 0
	for i, c := range cands {
		if c.lesDeux > 0 || c.auSite > 0 || i < 10 {
			d := "-"
			if c.dMinM < math.MaxFloat64 {
				d = formatM(c.dMinM)
			}
			e := "-"
			if c.tMinMS < math.MaxInt64 {
				e = formatMS(c.tMinMS)
			}
			t.Logf("%s : mot 0x%08X — %d creation(s), %d au site, %d coincidente(s), %d les deux ; "+
				"dMin %s, ecart classe min %s", id, c.mot, c.creations, c.auSite, c.coincidentes,
				c.lesDeux, d, e)
		}
		if c.lesDeux > 0 {
			elus++
		}
	}
	t.Logf("GATE A1.3 %s : %d candidat(s) (site ET coincidence). Rappel du protocole §1 : 0 site "+
		"`assault_bomb` au catalogue pour cette carte — un zero ici chiffre l'ancrage manquant, il "+
		"ne refute pas l'objet.", id, elus)
}

// formatM / formatMS : rendus « - » compatibles du tableau ci-dessus.
func formatM(v float64) string { return fmt.Sprintf("%.2f m", v) }
func formatMS(v int64) string  { return fmt.Sprintf("%d ms", v) }
