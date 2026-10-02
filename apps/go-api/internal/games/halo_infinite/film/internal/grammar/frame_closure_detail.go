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
// payload derriere elle, le dernier composant lu, et les lectures qui ont depasse la fin du payload.
//
// # LA MARCHE EST CELLE DE PRODUCTION, PILOTEE A LA MAIN
//
// [decodeFrameParRangs] ne rend ni la fin de la vue B ni le flux de la vue C : la marche de ce
// fichier ([marcheDetaillee.marcherParRangs]) en recopie le PILOTAGE — dix lignes : bit de
// configuration, [consumeVueA], [decodeInferLoop], [consumeVueC], [vueCFermee] — et appelle les
// MEMES lecteurs, sans lire un bit a cote. Le classement est celui de [FrameClosure] (meme
// `mesureDesTrames`). Le garde-fou de la recopie est un test : sur les bobines du depot, la carte
// rendue par [FrameClosureDetaillee] est IDENTIQUE, champ a champ, a celle de [FrameClosure]
// (`frame_closure_detail_test.go`). La sortie de vue B se lit aux compteurs que l observateur
// tient ([Observation.RejetsHorsDatum], [Observation.RejetsDeVue]), avant et apres la boucle.

import "fmt"

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
	// SortieVueBAutre : une combinaison des compteurs qu aucune branche de la boucle ne produit.
	// Son compte doit valoir zero ; il existe pour que ce zero soit VU.
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
	// ListeNonLocalisee : aucune vue n a ete lue (cf. [CauseListeNonLocalisee]).
	ListeNonLocalisee bool
	// Fermee : le paquet se ferme au bit pres. Cause : la premiere cause d arret, vide s il ferme.
	Fermee bool
	Cause  string
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
	// NEW traverses proprement (`DesyncAt == -1`).
	RecordsLus int
	NeufsLus   []uint32
	// DernierLu decrit le dernier record de la vue B et son dernier composant lu.
	DernierLu string
	// UtilesEnJeu : les records utiles lus et non fermes du paquet.
	UtilesEnJeu int
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
}

// FrameClosureDetaillee est [FrameClosure] qui rend en plus, a `voir` (qui peut etre nil), le
// detail de CHAQUE paquet delta marche, dans l ordre du film. La carte rendue est celle de
// [FrameClosure].
func FrameClosureDetaillee(fc *FilmContext, utiles UsagesProduit,
	voir func(PaquetDeCarte)) (FrameClosureReport, error) {
	if fc == nil {
		return FrameClosureReport{}, fmt.Errorf("filmdec: contexte de film nil — aucune fermeture a mesurer")
	}
	chunks := fc.ChunkNumbers()
	if len(chunks) == 0 {
		return FrameClosureReport{}, ErrNoFilmChunk
	}
	reg, err := fc.Registry()
	if err != nil {
		return FrameClosureReport{}, fmt.Errorf("filmdec: registre illisible, les trames n ont pas de grammaire: %w", err)
	}
	cfg := fc.CadreDeBalayage()
	if !cfg.Profil.Grammaire.ClassesDeVue {
		return FrameClosureReport{}, fmt.Errorf("filmdec: la carte de fermeture exige la marche " +
			"par classes de vue (GrammaireBalayage.ClassesDeVue)")
	}
	md := marcheDetaillee{mesureDesTrames: nouvelleMesureDesTrames(reg, utiles, cfg), voir: voir}
	monde := NewWorld(reg)
	monde.PoserTableAnticipee(ConstruireTableAnticipee(fc))
	marche := fc.MarcheDImageCle()
	for _, c := range chunks {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		monde.PoserChunkCourant(c)
		lierLeChunkAuMonde(monde, marche, data, pks, md.cfg.Obs)
		for _, pk := range pks {
			md.paquet(c, pk, data, monde)
		}
	}
	return md.rapport(), nil
}

// paquet marche UN paquet comme [mesureDesTrames.paquet], et en publie le detail.
func (md *marcheDetaillee) paquet(c int, pk FilmPacket, data []byte, w *World) {
	if pk.Type != PacketTypeDelta || pk.Size < 1 {
		return
	}
	pay := pk.Payload(data)
	d := PaquetDeCarte{Chunk: c, Index: pk.Index, TimestampUS: pk.TimestampUS, Bits: len(pay) * 8,
		DebutVueB: -1, FinVueB: -1}
	debut := movementStateSkipLeadBits
	if _, present := PacketHeadEventType(pay); present {
		debut, _ = debutDeLaListe(pay, w, md.cfg)
		if debut < 0 {
			md.listeNonLocalisee()
			d.ListeNonLocalisee, d.Cause = true, causeListeNonLocalisee
			md.publier(d)
			return
		}
	}
	md.marcherPaquetDetaille(pay, w, debut, &d)
	md.publier(d)
}

// publier rend le detail d un paquet a l appelant, s il en veut.
func (md *marcheDetaillee) publier(d PaquetDeCarte) {
	if md.voir != nil {
		md.voir(d)
	}
}

// marcherPaquetDetaille est [mesureDesTrames.marcherPaquet] sous la marche detaillee.
func (md *marcheDetaillee) marcherPaquetDetaille(pay []byte, w *World, debut int, d *PaquetDeCarte) {
	md.vueC = LectureVueC{}
	recs, rangs, l := md.marcherParRangs(pay, w, debut, d)
	enTete := debut == md.cfg.PacketPreambleBits && md.cfg.PacketPreambleBits >= 1
	p := paquetMarche{enTete: enTete, recs: recs, rangs: rangs, vueC: l}
	avantLus, avantFermes := md.rep.Utiles.Records, md.rep.Utiles.RecordsFermes
	md.classer(p)
	d.UtilesEnJeu = (md.rep.Utiles.Records - avantLus) - (md.rep.Utiles.RecordsFermes - avantFermes)
	d.Fermee = l.Fermee
	if !l.Fermee {
		d.Cause = md.bloquantDuPaquet(p).nom
	}
	decrireLesRecords(md.reg, recs, len(pay)*8, d)
}

// marcherParRangs est [decodeFrameParRangs] — meme pilotage, memes lecteurs — qui note en plus la
// fin de la vue B, sa sortie, l eid rejete et le flux de la vue C. Elle rend le verdict de la vue
// C que [decodeFrameParRangs] publierait au crochet.
func (md *marcheDetaillee) marcherParRangs(pay []byte, w *World, debut int,
	d *PaquetDeCarte) ([]FrameRecord, int, LectureVueC) {
	cfg := md.cfg
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	frameLen := len(pay) * 8
	rangs := 0
	if debut == cfg.PacketPreambleBits && cfg.PacketPreambleBits >= 1 {
		br.Skip(cfg.PacketPreambleBits - 1)
		if a := consumeVueA(br, frameLen); !a.Porte {
			d.Curseur = br.BitPos()
			return nil, rangs, LectureVueC{}
		}
		rangs++
	} else {
		br.Skip(debut)
	}
	w.PoserVueCourante(int(vueDeLImageCle))
	d.DebutVueB = br.BitPos()
	avant := lireCompteursDeSortie(cfg.Obs)
	recs, _, hitEnd := decodeInferLoop(br, pay, w, cfg)
	apres := lireCompteursDeSortie(cfg.Obs)
	d.Sortie, d.Anticipations = sortieDeVueB(avant, apres, hitEnd), apres.anticipations-avant.anticipations
	d.FinVueB, d.Curseur = br.BitPos(), br.BitPos()
	if d.Sortie.EstUnRejet() {
		d.EIDRejete = eidDeLEnTeteRejete(pay, cfg, d.FinVueB)
	}
	if !hitEnd {
		return recs, rangs, LectureVueC{}
	}
	rangs++
	c := consumeVueC(br, frameLen)
	if c.Porte {
		rangs++
	}
	l := LectureVueC{Atteinte: true, Arret: c.Arret, Fermee: c.Porte && vueCFermee(pay, br.BitPos())}
	if l.Fermee {
		l.Entrees = c.Entrees
	}
	d.VueCAtteinte, d.VueC, d.Curseur = true, c, br.BitPos()
	return recs, rangs, l
}

// compteursDeSortie : les compteurs de l observateur qui disent comment la vue B s arrete.
type compteursDeSortie struct {
	horsDatum, deVue, anticipations int
}

// lireCompteursDeSortie releve les compteurs de sortie de l observateur.
func lireCompteursDeSortie(o *Observation) compteursDeSortie {
	if o == nil {
		return compteursDeSortie{}
	}
	n := 0
	for _, k := range o.LiaisonsParRepliDAnticipation {
		n += k
	}
	return compteursDeSortie{horsDatum: o.RejetsHorsDatum, deVue: o.RejetsDeVue, anticipations: n}
}

// sortieDeVueB lit la sortie de la boucle de records : un rejet incremente UN des deux compteurs
// et rend `hitEnd` vrai ([decodeInferLoop]) ; un terminateur rend `hitEnd` vrai sans compteur.
func sortieDeVueB(avant, apres compteursDeSortie, hitEnd bool) SortieDeVueB {
	horsDatum, deVue := apres.horsDatum-avant.horsDatum, apres.deVue-avant.deVue
	switch {
	case !hitEnd && horsDatum == 0 && deVue == 0:
		return SortieVueBOuverte
	case !hitEnd:
		return SortieVueBAutre
	case horsDatum == 0 && deVue == 0:
		return SortieVueBTerminateur
	case horsDatum == 1 && deVue == 0:
		return SortieVueBRejetHorsDatum
	case horsDatum == 0 && deVue == 1:
		return SortieVueBRejetDeVue
	}
	return SortieVueBAutre
}

// largeurTagDeGeneration est la largeur du tag de generation de l en-tete d un record
// ([readRecordID] : `R(2)` en bits 30-31).
const largeurTagDeGeneration = 2

// eidDeLEnTeteRejete relit l identifiant de l en-tete rejete : [decodeInferLoop] remet le curseur
// a la FIN de cet en-tete, dont les `IDLowBits + 2` derniers bits sont l identifiant.
func eidDeLEnTeteRejete(pay []byte, cfg FrameConfig, finEntete int) uint32 {
	debut := finEntete - cfg.IDLowBits - largeurTagDeGeneration
	if debut < 0 {
		return 0
	}
	br := LecteurSur(pay)
	br.SetBitPos(debut)
	return readRecordID(br, cfg.IDLowBits, cfg.IDBase)
}
