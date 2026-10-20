package grammar

// neufs_prouves.go — UN NEW QUE LA FERMETURE DE SA TRAME PROUVE CREE SON ENTITE (lot des arrets de
// la vue B, suite, point (3), 2026-10-08).
//
// [corpsDeRecordNeuf] refuse un NEW traverse proprement dont le slot est lie en dur a un AUTRE
// archetype ([contreditUneEntiteVivante]) : chez le jeu une creation prend une entree libre, donc un
// tel NEW est soit une lecture fausse, soit la preuve que la suppression de l occupant n a pas ete
// lue. Pendant la trame, rien ne les distingue : le NEW reste non lie. A la fin de la trame, sa
// fermeture tranche. Une trame fermee dont le debut de vue B a ete LU prouve tous ses records
// ([preuveDeLaTrame] : 0) ; une trame fermee depuis un debut LOCALISE prouve ceux qui suivent ce
// debut. Un NEW refuse dans l etendue prouvee est une creation du jeu : il est lie
// ([World.BindFull]), et l entite vivante qu il contredisait — dont le DEL n a pas ete lu — cesse
// de l etre. Un NEW refuse hors de l etendue prouvee reste non lie, comme avant.
//
// MESURE (`bf15f7ab`, slot 553) : le NEW du bipede (`ti=35`, generation 1, chunk 14 paquet 1094)
// est lu dans une trame FERMEE et refuse contre le `ti=20` (generation 0) que l image-cle du chunk
// liait ; les records delta du joueur se lisaient sous `ti=20` pendant onze secondes.

// neufSuspendu est un NEW refuse dans la trame en cours : son identifiant, son archetype et le bit
// de son en-tete.
type neufSuspendu struct {
	id, ti uint32
	bit    int
}

// suspendreUnNeuf retient un NEW que [corpsDeRecordNeuf] vient de refuser.
func (w *World) suspendreUnNeuf(rec *FrameRecord) {
	w.neufsSuspendus = append(w.neufsSuspendus, neufSuspendu{id: rec.ID, ti: rec.TypeIndex, bit: rec.HeaderBit})
}

// oublierLesNeufsSuspendus vide la liste : la marche d une trame commence (les marches d essai qui
// la precedent ne lient rien).
func (w *World) oublierLesNeufsSuspendus() { w.neufsSuspendus = w.neufsSuspendus[:0] }

// lierLesNeufsProuves lie les NEW suspendus de la trame dont l en-tete est dans l etendue que sa
// fermeture prouve, a partir du bit `prouveeDes` ([preuveDeLaTrame]), puis vide la liste.
func (w *World) lierLesNeufsProuves(prouveeDes uint32) {
	for _, n := range w.neufsSuspendus {
		if n.bit < 0 || uint64(n.bit) < uint64(prouveeDes) {
			continue
		}
		w.BindFull(n.id, n.ti) // liaison lecture.LiaisonLueNeuf
	}
	w.oublierLesNeufsSuspendus()
}
