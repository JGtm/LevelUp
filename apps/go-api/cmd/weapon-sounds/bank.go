package main

// bank.go — parseur de bank Wwise (chunks + hierarchie HIRC).
//
// PRINCIPE : NE RIEN POSTULER SUR LE LAYOUT. Les offsets internes des objets HIRC
// dependent de la version de Wwise, et l'en-tete ne la donne pas de facon exploitable.
// Chaque lecture ambigue est donc ESSAYEE PUIS VALIDEE contre un ensemble connu :
//
//   - un `sourceID` de Sound n'est retenu que s'il appartient aux IDs du `.pck` de l'arme ;
//   - une liste d'enfants n'est retenue que si TOUS ses elements sont des IDs d'objets
//     reellement presents dans cette bank ;
//   - une liste d'actions d'Event n'est retenue que si tous ses elements sont des Actions.
//
// Une lecture au mauvais offset ne survit pas a ces filtres. C'est ce qui rend le parseur
// robuste sans connaitre la version exacte.

import (
	"encoding/binary"
	"fmt"
)

// Types d'objets HIRC utiles ici (nomenclature Wwise).
const (
	typeSound     = 2
	typeAction    = 3
	typeEvent     = 4
	typeRandomSeq = 5
)

// objetHIRC est une entree brute de la hierarchie.
type objetHIRC struct {
	Type byte
	ID   uint32
	Data []byte
}

// bank est une bank Wwise decodee.
type bank struct {
	Objets  map[uint32]objetHIRC
	Sons    map[uint32]uint32   // idObjet Sound -> id .wem
	Events  map[uint32][]uint32 // idEvent -> ids d'Action
	Actions map[uint32]uint32   // idAction -> id de la cible
	Enfants map[uint32][]uint32 // idConteneur -> ids enfants
}

// chunks decoupe une bank en chunks {magic, charge utile}.
func chunks(b []byte) map[string][]byte {
	out := map[string][]byte{}
	for off := 0; off+8 <= len(b); {
		magic := string(b[off : off+4])
		taille := int(binary.LittleEndian.Uint32(b[off+4:]))
		debut := off + 8
		if taille < 0 || debut+taille > len(b) {
			break
		}
		out[magic] = b[debut : debut+taille]
		off = debut + taille
	}
	return out
}

// objetsHIRC decode la liste d'objets du chunk HIRC.
func objetsHIRC(h []byte) ([]objetHIRC, error) {
	if len(h) < 4 {
		return nil, fmt.Errorf("bank: chunk HIRC trop court")
	}
	n := int(binary.LittleEndian.Uint32(h))
	out := make([]objetHIRC, 0, n)
	off := 4
	for i := 0; i < n && off+9 <= len(h); i++ {
		typ := h[off]
		taille := int(binary.LittleEndian.Uint32(h[off+1:]))
		id := binary.LittleEndian.Uint32(h[off+5:])
		fin := off + 5 + taille
		if taille < 4 || fin > len(h) {
			break
		}
		out = append(out, objetHIRC{Type: typ, ID: id, Data: h[off+9 : fin]})
		off = fin
	}
	return out, nil
}

// parserBank decode une bank et resout sa hierarchie.
// estWem valide un candidat `sourceID` (typiquement : appartenance au `.pck` de l'arme).
func parserBank(brut []byte, estWem func(uint32) bool) (*bank, error) {
	ch := chunks(brut)
	h, ok := ch["HIRC"]
	if !ok {
		return nil, fmt.Errorf("bank: chunk HIRC absent")
	}
	objs, err := objetsHIRC(h)
	if err != nil {
		return nil, err
	}
	b := &bank{
		Objets:  make(map[uint32]objetHIRC, len(objs)),
		Sons:    map[uint32]uint32{},
		Events:  map[uint32][]uint32{},
		Actions: map[uint32]uint32{},
		Enfants: map[uint32][]uint32{},
	}
	for _, o := range objs {
		b.Objets[o.ID] = o
	}
	connu := func(id uint32) bool { _, ok := b.Objets[id]; return ok }

	for _, o := range objs {
		switch o.Type {
		case typeSound:
			if wem, ok := lireSourceID(o.Data, estWem); ok {
				b.Sons[o.ID] = wem
			}
		case typeAction:
			if cible, ok := lireCibleAction(o.Data, connu); ok {
				b.Actions[o.ID] = cible
			}
		case typeEvent:
			if acts, ok := lireActionsEvent(o.Data, b.Objets); ok {
				b.Events[o.ID] = acts
			}
		default:
			if enf := lireEnfants(o.Data, connu); len(enf) > 0 {
				b.Enfants[o.ID] = enf
			}
		}
	}
	return b, nil
}

// lireSourceID tente les deux layouts connus de `AkBankSourceData` et valide le resultat.
//
//	A : pluginID u32 | streamType u8  | sourceID u32  (Wwise recent)
//	B : pluginID u32 | streamType u32 | sourceID u32  (variantes anciennes)
func lireSourceID(d []byte, estWem func(uint32) bool) (uint32, bool) {
	for _, off := range []int{5, 8} {
		if off+4 > len(d) {
			continue
		}
		id := binary.LittleEndian.Uint32(d[off:])
		if id != 0 && estWem(id) {
			return id, true
		}
	}
	return 0, false
}

// lireCibleAction lit l'objet vise par une Event Action.
//
//	u16 typeAction | u32 idCible   (layout courant ; le repli u8 couvre les variantes)
func lireCibleAction(d []byte, connu func(uint32) bool) (uint32, bool) {
	for _, off := range []int{2, 1} {
		if off+4 > len(d) {
			continue
		}
		id := binary.LittleEndian.Uint32(d[off:])
		if id != 0 && connu(id) {
			return id, true
		}
	}
	return 0, false
}

// lireActionsEvent lit la liste d'Actions d'un Event (compteur u8 ou u32 selon la version).
func lireActionsEvent(d []byte, objets map[uint32]objetHIRC) ([]uint32, bool) {
	essais := []struct{ debut, n int }{}
	if len(d) >= 1 {
		essais = append(essais, struct{ debut, n int }{1, int(d[0])})
	}
	if len(d) >= 4 {
		essais = append(essais, struct{ debut, n int }{4, int(binary.LittleEndian.Uint32(d))})
	}
	for _, e := range essais {
		if e.n <= 0 || e.n > 256 || e.debut+4*e.n > len(d) {
			continue
		}
		ids := make([]uint32, 0, e.n)
		ok := true
		for i := 0; i < e.n; i++ {
			id := binary.LittleEndian.Uint32(d[e.debut+4*i:])
			o, present := objets[id]
			if !present || o.Type != typeAction {
				ok = false
				break
			}
			ids = append(ids, id)
		}
		if ok {
			return ids, true
		}
	}
	return nil, false
}

// lireEnfants cherche la liste d'enfants d'un conteneur sans connaitre l'offset exact :
// un u32 N suivi de N identifiants TOUS presents dans la bank. On retient la plus longue
// liste valide — un faux positif exigerait N identifiants consecutifs tous valides.
func lireEnfants(d []byte, connu func(uint32) bool) []uint32 {
	var meilleur []uint32
	for off := 0; off+4 <= len(d); off++ {
		n := int(binary.LittleEndian.Uint32(d[off:]))
		if n < 1 || n > 512 || off+4+4*n > len(d) {
			continue
		}
		if n <= len(meilleur) {
			continue
		}
		ids := make([]uint32, 0, n)
		ok := true
		for i := 0; i < n; i++ {
			id := binary.LittleEndian.Uint32(d[off+4+4*i:])
			if !connu(id) {
				ok = false
				break
			}
			ids = append(ids, id)
		}
		if ok {
			meilleur = ids
		}
	}
	return meilleur
}

// wemsDeEvent rend l'ensemble des `.wem` atteignables depuis un Event, en descendant
// Event -> Actions -> cible -> enfants (transitivement) -> Sound.
func (b *bank) wemsDeEvent(idEvent uint32) []uint32 {
	vus := map[uint32]bool{}
	set := map[uint32]bool{}
	var descendre func(id uint32)
	descendre = func(id uint32) {
		if vus[id] {
			return
		}
		vus[id] = true
		if wem, ok := b.Sons[id]; ok {
			set[wem] = true
		}
		for _, enfant := range b.Enfants[id] {
			descendre(enfant)
		}
	}
	for _, idAction := range b.Events[idEvent] {
		if cible, ok := b.Actions[idAction]; ok {
			descendre(cible)
		}
	}
	return trier(set)
}

func trier(set map[uint32]bool) []uint32 {
	out := make([]uint32, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
