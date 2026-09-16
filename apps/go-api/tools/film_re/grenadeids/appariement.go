//go:build research

package grenadeids

// appariement.go — LA QUESTION (2) : CE QUI SUIT LE MARQUEUR S APPARIE-T-IL AUX DECREMENTS
// UNITAIRES DU COMPTEUR i22 ?
//
// C EST LA CHAINE QUI A ETABLI LES QUATRE IDENTIFIANTS ACTUELS (note M3 §2.2 : « le second
// signal, independant, est l appariement de 35 lancers aux decrements unitaires du compteur
// d inventaire porte »). Elle se rejoue ici sur les candidats des builds anciens.
//
// CE QUI EST MESURE, ET CE QUI NE L EST PAS. i22 (`unit-grenade-counts-component`) porte UN
// compteur PAR RANG : un decrement d exactement une unite sur un seul rang, entre deux lectures
// consecutives du MEME porteur, est un lancer de ce rang-la (ou un echange d arme qui vide le
// stock, d ou l exigence « une seule unite, un seul rang »). L instrument ne prononce donc pas
// « X est la grenade de rang r » : il compte combien d occurrences de X tombent dans
// l intervalle d un decrement du rang r, et laisse la lecture du verdict a la note.
//
// LE PONT INDEX JOUEUR / SLOT N EST PAS SUPPOSE. Le lancer porte un index de 5 bits
// (indexation « acurtis ») et i22 porte un SLOT de bipede (une vie) : les deux ne se deduisent
// pas l un de l autre sans la couche `replay`. L instrument OBSERVE les couples (index, slot)
// que l appariement produit et dit s ils forment une fonction — c est un controle de l
// appariement, pas une hypothese qui le fonde.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/grammar"
)

// rangsGrenade : le nombre de rangs de grenade du titre, c est-a-dire la longueur des compteurs
// d i22. Recopie de `grammar.invDeltaGrenadeSlots`, qui est prive.
const rangsGrenade = 4

// DecrementI22 est un decrement d exactement une unite, sur un seul rang, entre deux lectures
// consecutives du compteur du meme porteur.
type DecrementI22 struct {
	Slot           uint32
	Rang           int
	DebutUS, FinUS uint64
	// Avant et Apres sont les deux compteurs, conserves pour la lecture a la main.
	Avant, Apres [rangsGrenade]uint32
}

// StatsI22 dit ce que la lecture d i22 a donne, avant tout appariement.
type StatsI22 struct {
	Lectures    int
	LecturesI22 int
	Porteurs    int
	Decrements  int
	ParRang     [rangsGrenade]int
	Implausible int
	Records     int
	// I22Read / I22Unread : lectures d i22 abouties / marches arretees avant. Sur un build
	// ancien, un I22Read au plancher dirait que le canal d inventaire lui-meme ne marche pas,
	// et l absence d appariement ne prouverait alors rien sur les identifiants.
	I22Read, I22Unread int
}

// Appariement est le resultat pour UN candidat.
type Appariement struct {
	ID uint32
	// N est le nombre d occurrences du candidat derriere le marqueur.
	N int
	// Apparies : occurrences tombant dans l intervalle d exactement un rang de decrement.
	Apparies int
	// Ambigus : occurrences tombant dans des decrements de rangs DIFFERENTS — elles ne
	// designent aucun rang et ne sont comptees nulle part ailleurs.
	Ambigus int
	// ParRang : la repartition des occurrences appariees.
	ParRang [rangsGrenade]int
	// Couples : les couples (index joueur du lancer, slot du porteur) vus a l appariement.
	Couples map[int]map[uint32]int
}

// Fonctionnel dit si chaque index joueur apparie ne designe qu un seul slot porteur : c est le
// controle de coherence du pont, pas sa definition.
func (a Appariement) Fonctionnel() bool {
	for _, slots := range a.Couples {
		if len(slots) > 1 {
			return false
		}
	}
	return len(a.Couples) > 0
}

// LireDecrements charge l inventaire suivi dans les paquets delta du film et en extrait les
// decrements unitaires.
//
// IL RELIT LE FILM (enveloppe D2 `ScanFilmInventoryDeltas`, hors production) : la marche des
// composants d i22 demande un contexte de bobine que l instrument ne reconstruit pas a la main.
// C est un second passage disque sur le MEME film, jamais un second film en vol.
func LireDecrements(racine, id string) ([]DecrementI22, StatsI22, error) {
	lectures, st, err := grammar.ScanFilmInventoryDeltas(cheminFilm(racine, id))
	if err != nil {
		return nil, StatsI22{}, err
	}
	stats := StatsI22{
		Lectures:    len(lectures),
		Implausible: st.Implausible,
		Records:     st.Records,
		I22Read:     st.I22Read,
		I22Unread:   st.I22Unread,
	}
	parPorteur := map[uint32][]grammar.InventoryDelta{}
	for _, d := range lectures {
		if len(d.Grenades) != rangsGrenade {
			continue
		}
		stats.LecturesI22++
		parPorteur[d.Slot] = append(parPorteur[d.Slot], d)
	}
	stats.Porteurs = len(parPorteur)
	var out []DecrementI22
	for slot, suite := range parPorteur {
		sort.Slice(suite, func(i, j int) bool { return suite[i].TimestampUS < suite[j].TimestampUS })
		out = append(out, decrementsDUnPorteur(slot, suite)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FinUS < out[j].FinUS })
	stats.Decrements = len(out)
	for _, d := range out {
		stats.ParRang[d.Rang]++
	}
	return out, stats, nil
}

// decrementsDUnPorteur parcourt la suite chronologique d UN porteur et retient les transitions
// ou un seul rang perd exactement une unite.
func decrementsDUnPorteur(slot uint32, suite []grammar.InventoryDelta) []DecrementI22 {
	var out []DecrementI22
	for i := 1; i < len(suite); i++ {
		avant, apres := compteurs(suite[i-1]), compteurs(suite[i])
		rang, ok := rangDecremente(avant, apres)
		if !ok {
			continue
		}
		out = append(out, DecrementI22{
			Slot:    slot,
			Rang:    rang,
			DebutUS: suite[i-1].TimestampUS,
			FinUS:   suite[i].TimestampUS,
			Avant:   avant,
			Apres:   apres,
		})
	}
	return out
}

// compteurs recopie les quatre compteurs d une lecture.
func compteurs(d grammar.InventoryDelta) [rangsGrenade]uint32 {
	var out [rangsGrenade]uint32
	copy(out[:], d.Grenades)
	return out
}

// rangDecremente rend le rang qui a perdu exactement une unite, a condition qu il soit le SEUL
// a avoir bouge. Toute autre transition (ramassage, vidage, deux rangs) est refusee.
func rangDecremente(avant, apres [rangsGrenade]uint32) (int, bool) {
	rang, trouve := -1, false
	for r := 0; r < rangsGrenade; r++ {
		switch {
		case avant[r] == apres[r]:
		case avant[r] == apres[r]+1:
			if trouve {
				return 0, false
			}
			rang, trouve = r, true
		default:
			return 0, false
		}
	}
	return rang, trouve
}

// Apparier confronte les occurrences d un marqueur aux decrements, avec une tolerance
// temporelle appliquee aux deux bornes de chaque intervalle.
func Apparier(occ []Occurrence, dec []DecrementI22, toleranceUS uint64) []Appariement {
	parID := map[uint32]*Appariement{}
	for _, o := range occ {
		a := parID[o.ID]
		if a == nil {
			a = &Appariement{ID: o.ID, Couples: map[int]map[uint32]int{}}
			parID[o.ID] = a
		}
		a.N++
		appliquerDecrements(a, o, dec, toleranceUS)
	}
	out := make([]Appariement, 0, len(parID))
	for _, v := range parID {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// appliquerDecrements impute UNE occurrence : un seul rang la reclame, ou elle est ambigue.
func appliquerDecrements(a *Appariement, o Occurrence, dec []DecrementI22, tol uint64) {
	rang, slot, n := -1, uint32(0), 0
	for _, d := range dec {
		if !dansIntervalle(o.TimestampUS, d, tol) {
			continue
		}
		if n > 0 && d.Rang != rang {
			a.Ambigus++
			return
		}
		rang, slot, n = d.Rang, d.Slot, n+1
	}
	if n == 0 {
		return
	}
	a.Apparies++
	a.ParRang[rang]++
	if a.Couples[o.Index] == nil {
		a.Couples[o.Index] = map[uint32]int{}
	}
	a.Couples[o.Index][slot]++
}

// Decaler rend les memes occurrences, horodatage decale d une constante. C EST LE TEMOIN DE
// HASARD de la passe D : le meme appariement joue sur des instants deplaces mesure ce que la
// seule densite des decrements produit, sans aucun lien de cause. Un appariement qui ne bat pas
// son temoin ne prouve rien.
func Decaler(occ []Occurrence, deltaUS uint64) []Occurrence {
	out := make([]Occurrence, len(occ))
	copy(out, occ)
	for i := range out {
		out[i].TimestampUS += deltaUS
	}
	return out
}

// dansIntervalle dit si l instant tombe dans l intervalle du decrement, tolerance comprise.
func dansIntervalle(at uint64, d DecrementI22, tol uint64) bool {
	bas := uint64(0)
	if d.DebutUS > tol {
		bas = d.DebutUS - tol
	}
	return at >= bas && at <= d.FinUS+tol
}
