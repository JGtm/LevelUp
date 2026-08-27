package replay

// oddball_crane_d4_test.go — PHASE D4, SEUIL (1) : L'IDENTITE DU CRANE.
//
// LA QUESTION, EN UNE PHRASE : parmi les creations `ti=42` que le catalogue d'armes ECARTE,
// y a-t-il UN SEUL mot de 32 bits qui naisse au socle d'Oddball ET qui coincide avec les
// evenements `th=10` de crane ?
//
// LE PROTOCOLE — corpus, seuils, temoins, controle d'alignement, escalade — EST ECRIT ET
// COMMITE AVANT CE FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md`, section « D4 —
// ODDBALL : le PROTOCOLE »). Ce qui suit l'applique ; il ne le decide pas.
//
// # LA RECETTE N'EST PAS NEUVE, ET C'EST TOUT SON INTERET
//
// Elle a etabli le DRAPEAU (`attachement_phase0_drapeau_test.go`, mot `0x2A392328`) : trois
// lectures — combien d'ecartees, ou naissent-elles, quand — et un temoin de selectivite. Le
// crane est le meme genre d'objet : porte, hors de tout catalogue d'armes, ne a un socle
// nomme au catalogue de carte. Le balayage est LITTERALEMENT le meme appel ; seul le role de
// socle interroge change (`oddball_spawn` au lieu de `flag_spawn`).
//
// # LE CONTROLE D'ALIGNEMENT PASSE AVANT LA MESURE, ET IL EST LE POINT LE PLUS FRAGILE
//
// Le mot d'identite se lit derriere DEUX champs de largeur VARIABLE par film (9/5 en Quick
// Play, 8/3 sur les films BTB mesures — decouverte 8 du lot des armes au sol). Ce balayage lit
// aux largeurs PAR DEFAUT. Un film calibre autrement ne resoudrait AUCUNE identite, rendrait
// « aucun mot candidat », et ferait passer une panne de lecture pour une refutation.
//
// LE TEMOIN EST GRATUIT ET IL EST DEJA LA : le compte de creations RESOLUES au catalogue
// d'armes. Un film ou il vaut zero est lu aux mauvaises largeurs — il sort NON EXPLOITABLE et
// ne compte NI POUR NI CONTRE.
//
// # POURQUOI L'HORLOGE NE PASSE PAS PAR LE FIL DES MORTS
//
// Les creations sont datees sur l'horloge du FILM, les evenements `th=10` sur celle du MATCH.
// L'instrument du drapeau convertit par l'ecart MESURE sur le fil des morts (`bestDeathOffset`).
// Ici on prend l'autre expression de la meme origine, celle que `resolveOriginMs` utilise comme
// LECTURE et confronte a la premiere comme TEMOIN : l'horodatage du premier paquet du film
// (`ScanFilmClockOrigin`). Elle ne depend d'aucun appariement de morts — donc elle ne se degrade
// pas sur un film TRONQUE, et `24dbb67d` en est un (29 chunks).
//
// REGIME : gardes `ATT_FILM` (racine du cache) + `ODDBALL_FILM` (l'identifiant court), UN FILM
// PAR PROCESSUS, lecture seule, AUCUNE base.
//
//	$env:ATT_FILM="<repo>/data/cache"; $env:ODDBALL_FILM="43716616"
//	go test ./internal/analysis/replay/ -run EtatVivantOddballIdentite -v

import (
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
	// d4FilmEnv designe LE film mesure par ce processus. Sans elle, rien ne tourne : une boucle
	// sur sept films dans un seul processus est exactement ce que la doctrine de l'executeur
	// borne interdit.
	d4FilmEnv = "ODDBALL_FILM"
	// d4RayonSocleM : au-dela, une creation n'est plus « au socle ». Meme tolerance que le
	// volet drapeau (`attDrapeauRayonM`), elle-meme celle de la chaine des poses.
	d4RayonSocleM = 3.0
	// d4EcartEvenementMS : la coincidence exigee avec un evenement `th=10` de crane. Une
	// seconde est la valeur du protocole du §2.4, ecrite avant toute mesure.
	d4EcartEvenementMS = 1000
	// d4VariantOddball : le libelle de mode donne a `objectiveevents.Extract`. Le film ne nomme
	// pas son mode (map_objectives.go) ; le corpus, lui, est Oddball par construction — c'est
	// le recensement D1 qui l'a classe, sur le `pair_name` du registre.
	d4VariantOddball = "Oddball:Arena"
)

// d4Candidat porte le resume d'UN mot de 32 bits ecarte du catalogue d'armes.
type d4Candidat struct {
	mot uint32
	// creations : combien de creations portent ce mot ; auSocle : combien naissent a moins de
	// d4RayonSocleM du socle d'Oddball.
	creations, auSocle int
	// dMinM : distance minimale au socle ; tMinMS : ecart temporel minimal a un evenement de
	// crane. MaxInt64 quand aucun evenement n'existe — et cela se DIT, cela ne se convertit pas
	// en zero.
	dMinM  float64
	tMinMS int64
}

// retenu applique les DEUX conditions du protocole : naitre au socle ET coincider.
func (c d4Candidat) retenu() bool {
	return c.auSocle > 0 && c.tMinMS <= d4EcartEvenementMS
}

// TestEtatVivantOddballIdentite — LA MESURE DU SEUIL (1). Un film par processus.
func TestEtatVivantOddballIdentite(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(d4FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide (identifiant court du film Oddball)", d4FilmEnv)
	}
	// LA SENTINELLE MEMOIRE EST ARMEE DANS LE PROCESSUS QUI DECODE, jamais autour de lui. Le
	// lot RUNNER l'a etabli sur pieces : un seul film peut allouer plusieurs gibioctets en
	// quelques secondes (3,17 Gio en 3,6 s sur `a349fea8`), et « un film par processus » sans
	// plafond n'a pas suffi a preserver le poste de travail. `t.Fatal` depuis la goroutine de
	// la sentinelle serait illegal : on RAPPORTE, et le plafond souple fait le reste.
	g := filmproc.Arm("d4-oddball", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio — mesure interrompue, ce film ne compte "+
			"NI POUR NI CONTRE", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio (plafond souple %d Gio)",
			id, float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()

	if _, ok := objOpenFilm(t, root, id); !ok {
		t.Fatalf("%s : film absent du cache (%s=%q)", id, attFilmEnv, root)
	}

	cre, socles, ok := attCreationsEcartees(t, root, id, "oddball_spawn")
	if !ok {
		t.Logf("NON EXPLOITABLE %s : bornes de quantification de la carte indisponibles — "+
			"un objet du monde ne rend que des quanta, « a moins de %.0f m » n'a aucun sens. "+
			"Ce film ne compte NI POUR NI CONTRE.", id, d4RayonSocleM)
		return
	}
	if len(socles) == 0 {
		t.Logf("NON EXPLOITABLE %s : aucun socle `oddball_spawn` au catalogue d'objectifs pour "+
			"cette carte — il n'y a pas de lieu de naissance a mesurer. Ce film ne compte NI "+
			"POUR NI CONTRE.", id)
		return
	}

	// D4.0 — LE CONTROLE D'ALIGNEMENT, AVANT TOUT LE RESTE.
	t.Logf("%s : denominateurs du balayage — %d ancres, %d acceptees, %d RESOLUES au catalogue "+
		"d'armes, %d ecartees, %d mots distincts parmi les ecartees ; %d socle(s) `oddball_spawn`",
		id, cre.st.Anchors, cre.st.Accepted, len(cre.connues), len(cre.ecartees), len(cre.mots),
		len(socles))
	if len(cre.connues) == 0 {
		t.Logf("NON EXPLOITABLE %s : ZERO creation resolue au catalogue d'armes — le bloc MPP est "+
			"lu aux mauvaises largeurs sur ce film. « Aucun mot candidat » ne vaudrait rien ici : "+
			"ce serait une panne de lecture, pas une refutation. NI POUR NI CONTRE.", id)
		return
	}

	instants := d4EvenementsCrane(t, root, id)
	clockUS, err := ScanFilmClockOrigin(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : origine d'horloge illisible : %v", id, err)
	}
	t.Logf("%s : %d evenement(s) `th=10` de crane, origine d'horloge %d us", id, len(instants), clockUS)
	if len(instants) == 0 {
		t.Logf("NON EXPLOITABLE %s : aucun evenement `th=10` de crane — la seconde condition "+
			"d'identite n'a pas d'oracle sur ce film. NI POUR NI CONTRE.", id)
		return
	}

	cands := d4Resume(cre.ecartees, socles, instants, clockUS)
	d4Publie(t, id, cands)
}

// d4EvenementsCrane rend les instants (horloge du MATCH) des evenements `th=10` de crane.
//
// LE ROSTER EST VIDE, ET C'EST VOULU : le seuil (1) ne demande que des INSTANTS. Nommer les
// acteurs exigerait la base, que cette phase n'ouvre pas — et le camp d'un porteur ne dit rien
// de l'identite de l'objet qu'il porte.
func d4EvenementsCrane(t *testing.T, root, id string) []int64 {
	t.Helper()
	src, ok := objOpenFilm(t, root, id)
	if !ok {
		t.Fatalf("%s : film absent du cache", id)
	}
	var out []int64
	for _, ev := range objectiveevents.Extract(id, d4VariantOddball, src, objectiveevents.MapRoster{}) {
		if ev.EventType != objectiveevents.EventTypeSkullCarry || ev.TimeMS == nil {
			continue
		}
		out = append(out, int64(*ev.TimeMS))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// d4Resume regroupe les creations ecartees par mot et mesure les deux conditions.
func d4Resume(ecartees []filmdec.EquipmentCreation, socles []PointObjective,
	instants []int64, clockUS uint64) []d4Candidat {
	parMot := map[uint32][]filmdec.EquipmentCreation{}
	for _, c := range ecartees {
		parMot[uint32(c.MPPVal[filmdec.MPPWord32])] = append(
			parMot[uint32(c.MPPVal[filmdec.MPPWord32])], c)
	}
	out := make([]d4Candidat, 0, len(parMot))
	for mot, cs := range parMot {
		cand := d4Candidat{mot: mot, creations: len(cs), dMinM: math.MaxFloat64, tMinMS: math.MaxInt64}
		for _, c := range cs {
			d := attDistSocleMin(c, socles)
			if d < cand.dMinM {
				cand.dMinM = d
			}
			if d <= d4RayonSocleM {
				cand.auSocle++
			}
			if e := d4EcartMin(c, instants, clockUS); e < cand.tMinMS {
				cand.tMinMS = e
			}
		}
		out = append(out, cand)
	}
	// ORDRE TOTAL : les retenus d'abord, puis le nombre de creations, puis le mot. Sans lui, un
	// parcours de map rendrait une sortie differente a chaque execution.
	sort.Slice(out, func(i, j int) bool {
		if out[i].retenu() != out[j].retenu() {
			return out[i].retenu()
		}
		if out[i].creations != out[j].creations {
			return out[i].creations > out[j].creations
		}
		return out[i].mot < out[j].mot
	})
	return out
}

// d4EcartMin rend l'ecart temporel minimal (ms) entre une creation et un evenement de crane.
//
// LA CONVERSION EST CELLE DE L'ORIGINE LUE : matchMS = (creationUS - premierPaquetUS) / 1000.
func d4EcartMin(c filmdec.EquipmentCreation, instants []int64, clockUS uint64) int64 {
	if c.TimestampUS < clockUS {
		return math.MaxInt64
	}
	at := int64(c.TimestampUS-clockUS) / 1000
	best := int64(math.MaxInt64)
	for _, v := range instants {
		if d := attAbs(at - v); d < best {
			best = d
		}
	}
	return best
}

// d4Publie imprime le tableau des mots et le VERDICT du seuil (1) pour ce film.
func d4Publie(t *testing.T, id string, cands []d4Candidat) {
	t.Helper()
	retenus := 0
	for i, c := range cands {
		if c.retenu() {
			retenus++
		}
		if i >= 12 && !c.retenu() {
			t.Logf("%s :   ... %d autre(s) mot(s) ecarte(s), aucun retenu", id, len(cands)-i)
			break
		}
		t.Logf("%s :   mot 0x%08X — %d creation(s), %d au socle (<= %.0f m ; min %s), "+
			"ecart min a un evenement de crane %s%s",
			id, c.mot, c.creations, c.auSocle, d4RayonSocleM, d4Distance(c.dMinM),
			d4Ecart(c.tMinMS), d4Marque(c.retenu()))
	}
	t.Logf("SIGNAL %s : %d mot(s) reunissant les DEUX conditions (naissance au socle ET "+
		"coincidence <= %d ms) sur %d mot(s) ecarte(s)", id, retenus, d4EcartEvenementMS, len(cands))
	switch {
	case retenus == 1:
		t.Logf("VERDICT %s : UN SEUL candidat — mot 0x%08X. Temoin de selectivite TENU sur ce "+
			"film (0 autre). Le seuil (1) exige le MEME mot sur >= 2 films : verdict de corpus, "+
			"pas de film.", id, cands[0].mot)
	case retenus == 0:
		t.Logf("VERDICT %s : AUCUN candidat — ce film n'elit pas de crane.", id)
	default:
		t.Logf("VERDICT %s : %d candidats — le temoin de SELECTIVITE est REFUTE sur ce film "+
			"(le protocole en exige exactement un).", id, retenus)
	}
}

// d4Distance rend une distance lisible, ou « aucun socle » quand il n'y en avait pas.
func d4Distance(d float64) string {
	if d == math.MaxFloat64 {
		return "aucune"
	}
	return fmt.Sprintf("%.1f m", d)
}

// d4Ecart rend un ecart lisible. L'INFINI SE DIT : le convertir en zero ferait passer une
// absence d'evenement pour une coincidence parfaite.
func d4Ecart(ms int64) string {
	if ms == math.MaxInt64 {
		return "aucun evenement"
	}
	return fmt.Sprintf("%d ms", ms)
}

// d4Marque signale les mots qui reunissent les deux conditions.
func d4Marque(retenu bool) string {
	if retenu {
		return "  <== CANDIDAT"
	}
	return ""
}
