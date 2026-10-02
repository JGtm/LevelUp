package grammar

// frame_closure_detail.go — LA CARTE DE FERMETURE DETAILLEE : LE DETAIL DE CHAQUE PAQUET. UN
// INSTRUMENT, comme [FrameClosure] : ni la cuisson ni le collecteur ne l appellent, et aucune
// sortie de production ne change.
//
// # CE QU ELLE AJOUTE A LA CARTE
//
// [FrameClosure] dit QUELLE cause arrete un paquet. La cause « vue C : terminateur hors cadre »
// ne dit pas OU la lecture
// s est faussee : la vue C a lu son terminateur, le paquet ne se ferme pas, donc une largeur est
// fausse QUELQUE PART devant. [FrameClosureDetaillee] rend, pour chaque paquet, ce qu il faut pour
// la ventiler : comment la vue B s est arretee (son terminateur, ou un REJET d en-tete — et
// lequel des deux), l eid rejete, ce que la vue C a lu (vide, nombre d entrees), le reste du
// payload derriere elle, le dernier composant lu, les lectures qui ont depasse la fin du payload,
// les regles de l ecrivain que la lecture contredit (`ecrivain_invariants.go`) et les departs de
// vue C decales d ou elle referme le paquet (`frame_closure_temoins.go`).
//
// # LA MARCHE EST CELLE DE PRODUCTION, ET LA CARTE DETAILLEE LA CONSOMME
//
// La marche des trames ([FilmContext.Trames], `marche_trames.go`) rend, pour chaque trame, ce que
// la marche par rangs a lu ([lectureDeTrame]) : les bornes et la sortie typee de la vue B, l eid
// rejete, le flux de la vue C, le curseur et le debordement. Le detail en est une fonction, sans
// relire un bit (sauf les temoins decales, qui rejouent la vue C a d autres departs). Le
// classement est celui de [FrameClosure] (meme `mesureDesTrames`) : sur les bobines du depot, la
// carte rendue par [FrameClosureDetaillee] est IDENTIQUE, champ a champ, a celle de
// [FrameClosure] (`frame_closure_detail_test.go`).

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// CauseHorsCadre est le nom de la cause « vue C : terminateur hors cadre » : la vue C a lu son
// terminateur, et le paquet ne se ferme pas.
const CauseHorsCadre = causeTerminateurHorsCadre

// CauseListeNonLocalisee est le nom de la cause d un paquet a liste d evenements non localisee.
const CauseListeNonLocalisee = causeListeNonLocalisee

// SortieDeVueB nomme la facon dont la boucle de records de la vue B ([decodeInferLoop]) s est
// arretee.
type SortieDeVueB int

// Les sorties de la vue B.
const (
	// SortieVueBNonAtteinte : la marche n est pas entree dans la vue B (liste d evenements non
	// localisee, vue A non portee).
	SortieVueBNonAtteinte SortieDeVueB = iota
	// SortieVueBOuverte : la boucle s est arretee sans terminateur (desynchronisation, fin de
	// payload, garde de boucle) ; la vue C n est pas lue.
	SortieVueBOuverte
	// SortieVueBTerminateur : la boucle a lu un record de type 0, la fin de liste.
	SortieVueBTerminateur
	// SortieVueBRejetHorsDatum : un DELTA dont le slot n est dans aucune table de datums connue
	// (la garde vive de `FUN_1406cbaa0`, cf. [Observation.RejetsHorsDatum]).
	SortieVueBRejetHorsDatum
	// SortieVueBRejetDeVue : un DELTA dont le slot est connu mais possede par une autre vue (le
	// repli de `FUN_1406cd128`, cf. [Observation.RejetsDeVue]).
	SortieVueBRejetDeVue
	// SortieVueBAutre : une sortie qu aucune branche de la boucle ne produit — la sortie est typee
	// par la boucle elle-meme ([Lecteur.sortirDeLaVueB]). Son compte vaut zero par construction ;
	// la valeur reste tant que la colonne de l instrument la porte.
	SortieVueBAutre
	// NombreDeSortiesDeVueB est le nombre de sorties.
	NombreDeSortiesDeVueB = 6
)

// String rend le nom de la sortie, tel que les rapports l ecrivent.
func (s SortieDeVueB) String() string {
	switch s {
	case SortieVueBNonAtteinte:
		return "non atteinte"
	case SortieVueBOuverte:
		return "ouverte"
	case SortieVueBTerminateur:
		return "terminateur"
	case SortieVueBRejetHorsDatum:
		return "rejet hors datum"
	case SortieVueBRejetDeVue:
		return "rejet de vue"
	case SortieVueBAutre:
		return "autre"
	}
	return fmt.Sprintf("sortie %d", int(s))
}

// EstUnRejet dit si la vue B s est arretee sur un en-tete REJETE.
func (s SortieDeVueB) EstUnRejet() bool {
	return s == SortieVueBRejetHorsDatum || s == SortieVueBRejetDeVue
}

// PaquetDeCarte est ce que la marche a rendu d UN paquet delta, avec son classement.
type PaquetDeCarte struct {
	// Chunk, Index et TimestampUS situent le paquet ; Bits est la taille de son payload.
	Chunk, Index int
	TimestampUS  uint64
	Bits         int
	// ListeNonLocalisee : aucune vue n a ete lue (cf. [CauseListeNonLocalisee]). ListeLocalisee :
	// le paquet porte une liste d evenements dont le debut a ete trouve ([localiserLaListe]).
	ListeNonLocalisee, ListeLocalisee bool
	// Fermee : le paquet se ferme ([LectureVueC.Fermee]). Cause : la premiere cause d arret, vide
	// s il ferme.
	Fermee bool
	Cause  string
	// FermeeAuBit et Invariant : le verdict de cadrage et la premiere regle de l ecrivain
	// contredite ([LectureVueC]). Invariants : TOUTES les regles contredites par la lecture (un bit
	// par [InvariantEcrivain]), jugees sur tout paquet dont la vue B est atteinte.
	FermeeAuBit bool
	Invariant   InvariantEcrivain
	Invariants  uint32
	// Deborde : la lecture a depasse la fin du payload ([source.Bits.Deborde]) — le verdict
	// d echec du moteur (`FUN_14298816c`).
	Deborde bool
	// TemoinsDecales : pour un paquet ferme au bit pres, les departs de vue C DECALES de k bits
	// (k de -8 a -1 : bits 0 a 7 ; k de 1 a 8 : bits 8 a 15) d ou la vue C ferme AUSSI le
	// paquet au bit pres ([temoinsDecales]).
	TemoinsDecales uint16
	// Sortie : comment la vue B s est arretee. EIDRejete : l identifiant COMPLET (tete de
	// generation comprise) de l en-tete rejete, quand [SortieDeVueB.EstUnRejet].
	Sortie    SortieDeVueB
	EIDRejete uint32
	// DebutVueB / FinVueB : le curseur a l entree de la vue B et a sa sortie (-1 : non atteinte).
	DebutVueB, FinVueB int
	// VueCAtteinte et VueC : le flux de la vue C tel que [consumeVueC] l a lu, entrees comprises
	// MEME quand le paquet ne se ferme pas (elles ne valent alors rien : lues a une position non
	// prouvee).
	VueCAtteinte bool
	VueC         FluxVueC
	// Curseur : la position ou la marche s est arretee ; Bits - Curseur est le reste du payload.
	Curseur int
	// RecordsLus : les records de la vue B rendus par la marche ; NeufsLus les slots de ses records
	// NEW traverses proprement (`DesyncAt == -1`), NeufsDesynchronises ceux des NEW lus dont la
	// traversee a desynchronise.
	RecordsLus          int
	NeufsLus            []uint32
	NeufsDesynchronises []uint32
	// DernierLu decrit le dernier record de la vue B et son dernier composant lu.
	DernierLu string
	// UtilesEnJeu : les records utiles lus et non fermes du paquet ; UtilesLus : ses records
	// utiles lus.
	UtilesEnJeu, UtilesLus int
	// LE MODE BORNE : ce qu un lecteur qui refuserait de lire au-dela du payload changerait.
	// RecordsDebordants : records dont la lecture finit APRES le dernier bit du payload ;
	// ComposantsDebordants : composants dont la lecture finit apres lui ; NeufsPropresDebordants :
	// records NEW traverses proprement sur un corps qui deborde (donc lies au monde, sauf refus
	// contre une entite vivante).
	RecordsDebordants, ComposantsDebordants, NeufsPropresDebordants int
	// Anticipations : les liaisons posees par le repli d anticipation (table anticipee) dans ce
	// paquet.
	Anticipations int
}

// marcheDetaillee est la mesure de la carte, qui rend en plus le detail de chaque paquet.
type marcheDetaillee struct {
	*mesureDesTrames
	voir func(PaquetDeCarte)
	// anticipations : les liaisons par anticipation comptees jusqu ici par l observateur de la
	// mesure — le compte d un paquet est l ecart d une trame a la suivante.
	anticipations int
}

// FrameClosureDetaillee est [FrameClosure] qui rend en plus, a `voir` (qui peut etre nil), le
// detail de CHAQUE paquet delta marche, dans l ordre du film. La carte rendue est celle de
// [FrameClosure].
func FrameClosureDetaillee(fc *FilmContext, utiles UsagesProduit,
	voir func(PaquetDeCarte)) (FrameClosureReport, error) {
	m, mt, err := mesureSurLaMarche(fc, utiles)
	if err != nil {
		return FrameClosureReport{}, err
	}
	md := marcheDetaillee{mesureDesTrames: m, voir: voir}
	mt.parcourir(func(t *trameLue) bool {
		md.detaillerLaTrame(t)
		return true
	})
	return md.rapport(), nil
}

// detaillerLaTrame classe UNE trame de la marche, comme [mesureDesTrames.classerLaTrame], et en
// publie le detail.
func (md *marcheDetaillee) detaillerLaTrame(t *trameLue) {
	p := t.paquet
	d := PaquetDeCarte{Chunk: p.Chunk, Index: p.Index, TimestampUS: p.TS, Bits: len(p.Payload) * 8,
		DebutVueB: -1, FinVueB: -1}
	// Les liaisons par anticipation ne se posent que dans la boucle de records de la vue B :
	// l ecart du compteur d une trame a la suivante est celui de la marche de ce paquet.
	n := compteDesAnticipations(md.cfg.Obs)
	d.Anticipations, md.anticipations = n-md.anticipations, n
	if t.debut < 0 {
		md.listeNonLocalisee()
		d.ListeNonLocalisee, d.Cause = true, causeListeNonLocalisee
		md.publier(d)
		return
	}
	d.ListeLocalisee = p.Debut != lecture.DebutEnTete
	md.detaillerLaMarche(&t.lecture, t.debut, p.Payload, &d)
	md.publier(d)
}

// publier rend le detail d un paquet a l appelant, s il en veut.
func (md *marcheDetaillee) publier(d PaquetDeCarte) {
	if md.voir != nil {
		md.voir(d)
	}
}

// detaillerLaMarche classe UNE trame lue depuis `debut` et en remplit le detail.
func (md *marcheDetaillee) detaillerLaMarche(l *lectureDeTrame, debut int, pay []byte, d *PaquetDeCarte) {
	detaillerLaLecture(l, pay, md.cfg, d)
	p := paquetMarche{enTete: partDeLaTete(debut, md.cfg), recs: l.recs, rangs: l.rangs, vueC: l.verdict}
	avantLus, avantFermes := md.rep.Utiles.Records, md.rep.Utiles.RecordsFermes
	md.classer(p)
	d.UtilesLus = md.rep.Utiles.Records - avantLus
	d.UtilesEnJeu = d.UtilesLus - (md.rep.Utiles.RecordsFermes - avantFermes)
	d.Fermee = l.verdict.Fermee
	if !l.verdict.Fermee {
		d.Cause = md.bloquantDuPaquet(p).nom
	}
	decrireLesRecords(md.reg, l.recs, len(pay)*8, d)
}

// detaillerLaLecture remplit ce que la marche par rangs dit d une trame : le curseur, le
// debordement, les bornes et la sortie de la vue B, l eid rejete, le flux de la vue C, le verdict,
// TOUTES les regles de l ecrivain que la lecture contredit, et les temoins decales d un paquet
// ferme au bit pres.
func detaillerLaLecture(l *lectureDeTrame, pay []byte, cfg FrameConfig, d *PaquetDeCarte) {
	d.Curseur, d.Deborde = l.curseur, l.deborde
	if l.debutVueB < 0 {
		return // la vue A n est pas portee : la vue B n est pas atteinte
	}
	rejet := estUnRejet(l.sortieVueB)
	d.DebutVueB, d.FinVueB, d.Sortie = l.debutVueB, l.finVueB, sortieDeLaCarte(l.sortieVueB)
	if rejet {
		d.EIDRejete = l.eidRejete
	}
	if !l.vueCAtteinte {
		d.Invariants = jugerLePaquet(l.recs, false, FluxVueC{}, true).ensemble
		return
	}
	d.VueCAtteinte, d.VueC = true, l.fluxC
	d.FermeeAuBit, d.Invariant = l.verdict.FermeeAuBit, l.verdict.Invariant
	d.Invariants = jugerLePaquet(l.recs, rejet, l.fluxC, true).ensemble
	if l.verdict.FermeeAuBit {
		d.TemoinsDecales = temoinsDecales(pay, cfg, d.FinVueB)
	}
}

// sortieDeLaCarte rend la sortie de vue B de la carte : les trois sorties qui closent la liste
// sont les siennes, les trois qui la laissent ouverte se confondent en [SortieVueBOuverte].
func sortieDeLaCarte(s lecture.SortieVueB) SortieDeVueB {
	switch s {
	case lecture.SortieNonAtteinte:
		return SortieVueBNonAtteinte
	case lecture.SortieTerminateur:
		return SortieVueBTerminateur
	case lecture.SortieRejetHorsDatum:
		return SortieVueBRejetHorsDatum
	case lecture.SortieRejetDeVue:
		return SortieVueBRejetDeVue
	}
	return SortieVueBOuverte
}

// compteDesAnticipations rend le nombre de liaisons par anticipation que l observateur a
// comptees, tous archetypes confondus.
func compteDesAnticipations(o *Observation) int {
	if o == nil {
		return 0
	}
	n := 0
	for _, k := range o.LiaisonsParRepliDAnticipation {
		n += k
	}
	return n
}
