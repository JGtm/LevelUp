package replay

// oddball_portage_d4_test.go — PHASE D4, SEUIL (2) : LE PORTAGE DU CRANE.
//
// LA QUESTION, EN UNE PHRASE : quand le crane cesse de repliquer sa position, y a-t-il
// EXACTEMENT UN joueur dont le score personnel monte sans interruption pendant tout ce temps ?
//
// LE PROTOCOLE — trou, tranche de 5 s, « exactement un », temoin, diagnostic d'oracle — EST
// ECRIT ET COMMITE AVANT CE FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md`, section
// « D4.3 — LE SEUIL (2) : les definitions operatoires »). Ce qui suit l'applique.
//
// # LE CANAL EST UNE ABSENCE, ET C'EST CE QUI LE REND SUR
//
// Un objet PORTE cesse d'emettre sa position : il suit son porteur, et le monde n'a plus rien a
// repliquer pour lui (`flag_objects.go`, § « Le principe »). Le trou entre deux vies LIBRES est
// donc un portage — non pas un indice de portage, le portage lui-meme. Ce que le film ne dit
// pas, c'est PAR QUI ; c'est la seule chose que l'oracle a a trancher.
//
// # POURQUOI LE DIAGNOSTIC DE L'ORACLE PRECEDE LE VERDICT
//
// En D2-ter, le score personnel a coule comme oracle de garde de colline : il etait DOMINE PAR
// LES FRAGS (delta dominant median de l'ordre de la centaine, la ou un tic de colline vaut
// quelques points). En Oddball il est CENSE etre du portage — mais « cense » n'est pas une
// mesure. Le diagnostic (points par seconde de trou) est donc publie AVANT le verdict, et le
// protocole engage d'avance : s'il dit « frags », le seuil (2) n'est pas evalue.
//
// REGIME : gardes `ATT_FILM` + `ODDBALL_FILM`, UN FILM PAR PROCESSUS, lecture seule, AUCUNE base.
//
//	$env:ATT_FILM="<repo>/data/cache"; $env:ODDBALL_FILM="43716616"
//	go test ./internal/analysis/replay/ -run EtatVivantOddballPortage -v

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/filmproc"
)

const (
	// d4MotCrane est le mot MPP de 32 bits ELU par le seuil (1) le 2026-08-27 : le meme sur les
	// quatre films mesurables, toujours ne a 0,0 m du socle `oddball_spawn` unique de sa carte,
	// toujours a 3-6 ms d'un evenement `th=10` de crane. Il est ecrit ici parce que ce fichier
	// MESURE l'hypothese ; il n'entre au manifeste du titre que si le seuil (2) tient.
	d4MotCrane = 0x0017592C
	// d4TrancheMS : la tranche de decoupage d'un trou. Le score d'Oddball monte a ~1 Hz pendant
	// le portage : 5 s laissent attendre cinq increments — assez pour qu'une absence signifie
	// quelque chose, assez peu pour qu'un trou de 20 s porte quatre verdicts independants.
	d4TrancheMS = 5000
	// d4GraineTemoin fixe le tirage du temoin. Une graine fixe, donc deux executions rendent la
	// MEME sortie : un temoin qui bouge d'une execution a l'autre ne se confronte a rien.
	d4GraineTemoin = 20260827
	// d4PartMinimale / d4TemoinMax : les deux seuils du §2.4, recopies sans modification.
	d4PartMinimale = 0.90
	d4TemoinMax    = 0.05
)

// d4Intervalle est une fenetre sur l'horloge du MATCH, en millisecondes.
type d4Intervalle struct{ debutMS, finMS int64 }

func (i d4Intervalle) dureeMS() int64 { return i.finMS - i.debutMS }

// TestEtatVivantOddballPortage — LA MESURE DU SEUIL (2). Un film par processus.
func TestEtatVivantOddballPortage(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(d4FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide (identifiant court du film Oddball)", d4FilmEnv)
	}
	g := filmproc.Arm("d4-oddball-portage", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio — mesure interrompue, ce film ne compte "+
			"NI POUR NI CONTRE", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio (plafond souple %d Gio)",
			id, float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()

	dir := objChunkDir(root, id)
	vies, ok := d4ViesLibres(t, root, id)
	if !ok {
		return
	}
	clockUS, err := ScanFilmClockOrigin(dir)
	if err != nil {
		t.Fatalf("%s : origine d'horloge illisible : %v", id, err)
	}
	libres, trous := d4Decoupe(vies, clockUS)
	t.Logf("%s : %d vie(s) libre(s) du mot 0x%08X, %d intervalle(s) LIBRE(s) mesurable(s), "+
		"%d trou(s) fermes", id, len(vies), uint32(d4MotCrane), len(libres), len(trous))
	if len(trous) == 0 {
		t.Logf("NON EXPLOITABLE %s : aucun trou ferme — l'objet n'a jamais cesse puis repris de "+
			"repliquer. NI POUR NI CONTRE.", id)
		return
	}

	src, _ := objOpenFilm(t, root, id)
	recs := objectiveevents.StatRecords(src)
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("%s : fil des morts illisible : %v", id, err)
	}
	identity := objectiveevents.SlotIdentityByDeaths(recs, deathInstantsOf(deaths))
	perso := objectiveevents.SeriesTotal(recs, objectiveevents.PersonalScoreComponent, false)
	t.Logf("%s : pont %d slot(s) nomme(s) ; score personnel sur %d slot(s)",
		id, len(identity), len(perso))
	if len(perso) < 2 {
		t.Logf("NON EXPLOITABLE %s : %d slot(s) portent un score personnel — sans au moins deux "+
			"joueurs, « exactement un » ne discrimine rien. NI POUR NI CONTRE.", id, len(perso))
		return
	}

	d4Mesure(t, id, trous, libres, perso)
}

// d4ViesLibres balaye l'archetype `ti=42` et rend les vies LIBRES du mot elu.
func d4ViesLibres(t *testing.T, root, id string) ([]flagFreeLife, bool) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()

	wr, _, ok := attBornes(t, root, id)
	if !ok {
		t.Logf("NON EXPLOITABLE %s : bornes de quantification indisponibles. NI POUR NI CONTRE.", id)
		return nil, false
	}
	dir := objChunkDir(root, id)
	kf := filmdec.ScanFilmWorldObjectKeyframes(dir, filmdec.GroundWeaponTypeIndex)
	if len(kf.Band) == 0 {
		t.Logf("NON EXPLOITABLE %s : aucun slot `ti=42` aux images-cles. NI POUR NI CONTRE.", id)
		return nil, false
	}
	cre, st, err := filmdec.ScanFilmGroundWeaponCreationsForBand(dir, &wr, kf.Band)
	if err != nil {
		t.Logf("NON EXPLOITABLE %s : creations `ti=42` illisibles : %v. NI POUR NI CONTRE.", id, err)
		return nil, false
	}
	// LES PISTES DELTA SONT LA PIECE INDISPENSABLE, et leur absence est le cas TRAITRE : sans
	// elles toute vie paraitrait reduite a un point, donc le film n'aurait QUE des trous.
	tracks, err := filmdec.ScanFilmWorldObjectsForBand(dir, &wr, kf.Band)
	if err != nil {
		t.Logf("NON EXPLOITABLE %s : pistes delta illisibles : %v — sans elles toute vie paraitrait "+
			"reduite a un point et le film n'aurait que des trous. NI POUR NI CONTRE.", id, err)
		return nil, false
	}
	scan := WorldObjectScan{Scanned: true, Creations: cre, Stats: st, Keyframes: kf, Tracks: tracks}
	vies := flagFreeLives(scan, map[uint32]Label{uint32(d4MotCrane): {En: "Skull", Fr: "Crane"}})
	t.Logf("%s : %d ancres, %d creations acceptees, %d pistes delta ; %d vie(s) du mot elu",
		id, st.Anchors, st.Accepted, len(tracks), len(vies))
	return vies, true
}

// d4Decoupe rend les intervalles LIBRES (l'objet replique) et les TROUS (il a cesse), tous deux
// sur l'horloge du MATCH.
//
// L'INTERVALLE AVANT LA PREMIERE VIE ET CELUI APRES LA DERNIERE NE SONT PAS DES TROUS : rien ne
// les ferme, et un intervalle ouvert n'a pas de duree.
func d4Decoupe(vies []flagFreeLife, clockUS uint64) (libres, trous []d4Intervalle) {
	conv := func(us uint64) int64 {
		if us < clockUS {
			return 0
		}
		return int64(us-clockUS) / 1000
	}
	sort.Slice(vies, func(i, j int) bool { return vies[i].T0US < vies[j].T0US })
	for i, v := range vies {
		if v.T1US > v.T0US {
			libres = append(libres, d4Intervalle{conv(v.T0US), conv(v.T1US)})
		}
		if i+1 < len(vies) && vies[i+1].T0US > v.T1US {
			trous = append(trous, d4Intervalle{conv(v.T1US), conv(vies[i+1].T0US)})
		}
	}
	return libres, trous
}

// d4Verdict porte le compte d'une passe (mesure ou temoin).
type d4Verdict struct {
	exploitables, unSeul, aucun, plusieurs int
	// ptsParSec / deltas : le DIAGNOSTIC de l'oracle, collecte sur les trous a porteur unique.
	ptsParSec []float64
	deltas    []int64
}

func (v d4Verdict) taux() float64 {
	if v.exploitables == 0 {
		return 0
	}
	return float64(v.unSeul) / float64(v.exploitables)
}

// d4Mesure applique le predicat aux trous, publie le diagnostic, puis le verdict et son temoin.
func d4Mesure(t *testing.T, id string, trous, libres []d4Intervalle,
	perso map[int][]objectiveevents.ScorePoint) {
	t.Helper()
	mes := d4Passe(trous, perso)
	t.Logf("%s : %d trou(s) dont %d exploitable(s) (>= %d ms) — %d a porteur UNIQUE, %d sans "+
		"aucun marqueur continu, %d a plusieurs", id, len(trous), mes.exploitables,
		d4TrancheMS, mes.unSeul, mes.aucun, mes.plusieurs)

	// LE DIAGNOSTIC DE L'ORACLE, PUBLIE AVANT LE VERDICT.
	t.Logf("DIAGNOSTIC %s : sur les trous a porteur unique — %s pt/s median, delta brut median "+
		"%s (le portage d'Oddball vaut ~1 pt/s ; un frag vaut ~100 d'un coup)",
		id, d4MedianeF(mes.ptsParSec), d4MedianeI(mes.deltas))

	tem, joueur, essais := d4Temoin(trous, libres, perso)
	// DEUX TEMOINS, PARCE QUE LE PROTOCOLE EN A DECRIT DEUX. Le §2.4 disait « un joueur tire au
	// hasard hors trou », l'addendum operatoire « le MEME predicat sur un intervalle hors trou ».
	// Les deux sont publies plutot que l'un choisi en silence : le premier controle l'INTERVALLE
	// (le trou explique-t-il quelque chose ?), le second le JOUEUR (n'importe qui marque-t-il en
	// continu ?). Le seuil s'applique au plus DEFAVORABLE des deux.
	t.Logf("TEMOIN %s : %d intervalle(s) de meme duree places DANS une vie libre — INTERVALLE : "+
		"%s rendent un marqueur continu unique ; JOUEUR (un slot tire hors porteur) : %s qualifient",
		id, essais, d4Part(tem, essais), d4Part(joueur, essais))

	if mes.exploitables == 0 {
		t.Logf("NON EXPLOITABLE %s : aucun trou d'au moins %d ms. NI POUR NI CONTRE.", id, d4TrancheMS)
		return
	}
	pire := tem
	if joueur > pire {
		pire = joueur
	}
	t.Logf("SIGNAL %s : %d/%d = %.1f %% des trous a porteur UNIQUE (seuil %.0f %%) ; temoin le "+
		"plus DEFAVORABLE %s (seuil <= %.0f %%)", id, mes.unSeul, mes.exploitables,
		100*mes.taux(), 100*d4PartMinimale, d4Part(pire, essais), 100*d4TemoinMax)
	switch {
	case mes.taux() >= d4PartMinimale && d4TauxTemoin(pire, essais) <= d4TemoinMax:
		t.Logf("VERDICT %s : les DEUX bornes du seuil (2) sont tenues sur ce film.", id)
	case mes.taux() >= d4PartMinimale:
		t.Logf("VERDICT %s : la part est tenue mais le TEMOIN ne l'est pas — un intervalle "+
			"quelconque rend le meme signal, donc le trou n'explique rien.", id)
	default:
		t.Logf("VERDICT %s : la part de trous a porteur unique est SOUS le seuil.", id)
	}
}

// d4Passe applique le predicat a une liste d'intervalles.
func d4Passe(ivs []d4Intervalle, perso map[int][]objectiveevents.ScorePoint) d4Verdict {
	var v d4Verdict
	for _, iv := range ivs {
		if iv.dureeMS() < d4TrancheMS {
			continue
		}
		v.exploitables++
		qual := d4Qualifiants(iv, perso)
		switch len(qual) {
		case 0:
			v.aucun++
		case 1:
			v.unSeul++
			d := d4Delta(perso[qual[0]], iv)
			v.deltas = append(v.deltas, d)
			v.ptsParSec = append(v.ptsParSec, float64(d)*1000/float64(iv.dureeMS()))
		default:
			v.plusieurs++
		}
	}
	return v
}

// d4Qualifiants rend les slots dont le score personnel croit STRICTEMENT dans CHAQUE tranche.
func d4Qualifiants(iv d4Intervalle, perso map[int][]objectiveevents.ScorePoint) []int {
	var out []int
	for slot := range perso {
		if d4CroitPartout(perso[slot], iv) {
			out = append(out, slot)
		}
	}
	sort.Ints(out)
	return out
}

// d4CroitPartout dit si la serie croit strictement dans chaque tranche de l'intervalle.
//
// LA DERNIERE TRANCHE INCOMPLETE EST FUSIONNEE A LA PRECEDENTE : une tranche d'une seconde
// n'attend qu'un increment, et l'exiger ferait echouer sur du bruit d'echantillonnage.
func d4CroitPartout(pts []objectiveevents.ScorePoint, iv d4Intervalle) bool {
	n := iv.dureeMS() / d4TrancheMS
	if n == 0 {
		return false
	}
	for i := int64(0); i < n; i++ {
		debut := iv.debutMS + i*d4TrancheMS
		fin := debut + d4TrancheMS
		if i == n-1 {
			fin = iv.finMS // la derniere tranche absorbe le reste
		}
		if d4Delta(pts, d4Intervalle{debut, fin}) <= 0 {
			return false
		}
	}
	return true
}

// d4Delta rend la croissance du score sur un intervalle : dernier point dedans moins dernier
// point AVANT le debut. Sans le point d'avant, un intervalle sans emission initiale rendrait une
// croissance imaginaire egale a la valeur absolue du score.
func d4Delta(pts []objectiveevents.ScorePoint, iv d4Intervalle) int64 {
	var avant, dedans int64
	vuAvant, vuDedans := false, false
	for _, p := range pts {
		switch {
		case int64(p.TimeMS) < iv.debutMS:
			avant, vuAvant = p.Value, true
		case int64(p.TimeMS) <= iv.finMS:
			dedans, vuDedans = p.Value, true
		}
	}
	if !vuDedans {
		return 0
	}
	if !vuAvant {
		return 0 // aucune reference : on ne compte pas une croissance qu'on ne sait pas mesurer
	}
	return dedans - avant
}

// d4Temoin applique le MEME predicat a des intervalles de MEME DUREE places DANS une vie libre.
//
// LE TIRAGE EST DETERMINISTE (graine fixe) : un temoin qui bouge d'une execution a l'autre ne se
// confronte a rien.
func d4Temoin(trous, libres []d4Intervalle,
	perso map[int][]objectiveevents.ScorePoint) (unSeul, joueur, essais int) {
	if len(libres) == 0 {
		return 0, 0, 0
	}
	slots := make([]int, 0, len(perso))
	for s := range perso {
		slots = append(slots, s)
	}
	sort.Ints(slots)                                // ordre total : sans lui le tirage dependrait du parcours de map
	rng := rand.New(rand.NewSource(d4GraineTemoin)) //nolint:gosec // temoin reproductible, pas de crypto
	for _, tr := range trous {
		d := tr.dureeMS()
		if d < d4TrancheMS {
			continue
		}
		var eligibles []d4Intervalle
		for _, l := range libres {
			if l.dureeMS() >= d {
				eligibles = append(eligibles, l)
			}
		}
		if len(eligibles) == 0 {
			continue // ce trou n'a pas de temoin, et le denominateur le dit
		}
		l := eligibles[rng.Intn(len(eligibles))]
		debut := l.debutMS + rng.Int63n(l.dureeMS()-d+1)
		iv := d4Intervalle{debut, debut + d}
		essais++
		qual := d4Qualifiants(iv, perso)
		if len(qual) == 1 {
			unSeul++
		}
		// TEMOIN « JOUEUR » : un slot tire au hasard, a l'exclusion du porteur du trou.
		porteur := -1
		if q := d4Qualifiants(tr, perso); len(q) == 1 {
			porteur = q[0]
		}
		if s, ok := d4TireHors(slots, porteur, rng); ok && d4CroitPartout(perso[s], iv) {
			joueur++
		}
	}
	return unSeul, joueur, essais
}

// d4TireHors tire un slot au hasard en excluant celui donne (-1 : aucune exclusion).
func d4TireHors(slots []int, exclu int, rng *rand.Rand) (int, bool) {
	cand := make([]int, 0, len(slots))
	for _, s := range slots {
		if s != exclu {
			cand = append(cand, s)
		}
	}
	if len(cand) == 0 {
		return 0, false
	}
	return cand[rng.Intn(len(cand))], true
}

// d4TauxTemoin rend le taux du temoin, zero quand il n'a pas d'essai.
func d4TauxTemoin(unSeul, essais int) float64 {
	if essais == 0 {
		return 0
	}
	return float64(unSeul) / float64(essais)
}

// d4Part rend un taux lisible, ou « pas de denominateur » — un taux sans essai n'est pas zero.
func d4Part(n, d int) string {
	if d == 0 {
		return "pas de denominateur"
	}
	return fmt.Sprintf("%d/%d = %.1f %%", n, d, 100*float64(n)/float64(d))
}

// d4MedianeF / d4MedianeI rendent une mediane lisible, ou « aucun » sur une serie vide.
func d4MedianeF(v []float64) string {
	if len(v) == 0 {
		return "aucun"
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return fmt.Sprintf("%.2f", s[len(s)/2])
}

func d4MedianeI(v []int64) string {
	if len(v) == 0 {
		return "aucun"
	}
	s := append([]int64(nil), v...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return fmt.Sprintf("%d", s[len(s)/2])
}
