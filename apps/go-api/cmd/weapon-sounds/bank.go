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
	// typeSwitch : conteneur pilote par un ETAT DE JEU. Ses enfants ne sont pas des
	// variantes interchangeables — les confondre avec ceux d'un `RandomSequence` fausse
	// 31 coups reconstitues sur 107. Cf. `conteneurs.go`.
	typeSwitch = 6
	typeBlend  = 9
)

// objetHIRC est une entree brute de la hierarchie.
type objetHIRC struct {
	Type byte
	ID   uint32
	Data []byte
}

// bank est une bank Wwise decodee.
type bank struct {
	// Embarques : `.wem` stockes dans la bank (chunk DIDX) -> offset et taille dans DATA.
	Embarques map[uint32][2]uint32
	Objets    map[uint32]objetHIRC
	Sons      map[uint32]uint32 // idObjet Sound -> id .wem
	// Gains : id .wem -> volume en dB declare par son objet Sound. Mesure : 5 312 sons sur
	// 62 753 portent un volume non nul, jusqu'a -96 dB. Additionner les couches a gain
	// unitaire ecrasait donc des renforts censes rester en arriere-plan.
	Gains   map[uint32]float32
	Events  map[uint32][]uint32 // idEvent -> ids d'Action
	Actions map[uint32]uint32   // idAction -> id de la cible
	Enfants map[uint32][]uint32 // idConteneur -> ids enfants
	// Switchs : conteneurs pilotes par un etat de jeu, decodes. `Enfants` n'en retient que
	// l'etat par defaut ; cette table garde la vue complete pour l'inspection.
	Switchs map[uint32]conteneurSwitch
	// VolNoeud : volume propre de CHAQUE noeud (Sound comme conteneur). Mesure de l'etape
	// 18 : 5 063 ActorMixer, 5 180 RandomSequence, 181 Blend et 128 Switch portent un
	// volume non nul, jusqu'a -96 dB. Le gain d'un `.wem` est donc la SOMME du chemin
	// evenement -> ... -> Sound, pas le volume du Sound seul.
	VolNoeud map[uint32]float32
	// GainsFondu : gain additionnel (dB) impose par la courbe de fondu d'un `Blend` a l'un
	// de ses enfants, au point de reference. Mesure : 0 gain partiel sur les 1305 banks —
	// la table reste correcte si une future version du jeu en introduit.
	GainsFondu map[uint32]map[uint32]float64
	// Compteurs de resolution des `Switch`, pour que le rendu puisse dire ce qu'il a fait
	// plutot que de le taire.
	SwParDefaut, SwVides, SwNonLus int
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

// mediasEmbarques lit le chunk `DIDX` : les `.wem` stockes DANS la bank.
//
// Un `.wem` n'est pas forcement dans un `.pck`. Le chunk `DIDX` indexe des medias embarques
// directement dans la bank (chunk `DATA`), et 694 des 1305 banks en portent. Ignorer cet
// index revenait a rejeter les sons correspondants : c'est ce qui vidait certaines couches
// d'un tir, l'elargissement aux autres packs n'y changeant rien.
// Format : suite d'entrees de 12 octets {id u32, offset u32, taille u32}.
func mediasEmbarques(ch map[string][]byte) map[uint32][2]uint32 {
	out := map[uint32][2]uint32{}
	d, ok := ch["DIDX"]
	if !ok {
		return out
	}
	for o := 0; o+12 <= len(d); o += 12 {
		id := binary.LittleEndian.Uint32(d[o:])
		off := binary.LittleEndian.Uint32(d[o+4:])
		taille := binary.LittleEndian.Uint32(d[o+8:])
		out[id] = [2]uint32{off, taille}
	}
	return out
}

// parserBank decode une bank et resout sa hierarchie.
// estWem valide un candidat `sourceID` (typiquement : appartenance aux `.pck` connus).
// Les medias EMBARQUES de la bank sont acceptes en plus, sans condition.
func parserBank(brut []byte, estWem func(uint32) bool) (*bank, error) {
	ch := chunks(brut)
	embarques := mediasEmbarques(ch)
	if len(embarques) > 0 {
		base := estWem
		estWem = func(id uint32) bool {
			if _, ok := embarques[id]; ok {
				return true
			}
			return base(id)
		}
	}
	h, ok := ch["HIRC"]
	if !ok {
		return nil, fmt.Errorf("bank: chunk HIRC absent")
	}
	objs, err := objetsHIRC(h)
	if err != nil {
		return nil, err
	}
	b := &bank{
		Embarques:  embarques,
		Objets:     make(map[uint32]objetHIRC, len(objs)),
		Sons:       map[uint32]uint32{},
		Gains:      map[uint32]float32{},
		Events:     map[uint32][]uint32{},
		Actions:    map[uint32]uint32{},
		Enfants:    map[uint32][]uint32{},
		Switchs:    map[uint32]conteneurSwitch{},
		VolNoeud:   map[uint32]float32{},
		GainsFondu: map[uint32]map[uint32]float64{},
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
				if p := lireProprietes(o.Data); p.Lu && p.VolumeDB != 0 {
					b.Gains[wem] = p.VolumeDB
					b.VolNoeud[o.ID] = p.VolumeDB
				}
			}
		case typeAction:
			// Seules les actions « jouer » contribuent au son. Retenir les autres revenait
			// a empiler dans le mixage la cible d'un `Stop` ou d'un `Break`.
			if cible, estPlay, ok := lireCibleAction(o.Data, connu); ok && estPlay {
				b.Actions[o.ID] = cible
			}
		case typeEvent:
			if acts, ok := lireActionsEvent(o.Data, b.Objets); ok {
				b.Events[o.ID] = acts
			}
		case typeBlend:
			// Le volume propre du conteneur, puis ses courbes de fondu : les enfants
			// inaudibles au point de reference ne sont pas retenus (etape 18.2).
			if pr := lireProprietesConteneur(o.Data); pr.Lu && pr.VolumeDB != 0 {
				b.VolNoeud[o.ID] = pr.VolumeDB
			}
			b.resoudreBlend(o, connu)
		case typeSwitch:
			if pr := lireProprietesConteneur(o.Data); pr.Lu && pr.VolumeDB != 0 {
				b.VolNoeud[o.ID] = pr.VolumeDB
			}
			// UN `Switch` N'EST PAS UN LOT DE VARIANTES. Il choisit ses enfants selon un
			// etat de jeu (distance, materiau...). Retenir tous ses enfants revenait a
			// melanger des etats qui ne coexistent jamais : mesure a l'origine de ce
			// correctif, 31 coups reconstitues sur 107 en portaient un, jusqu'a 71 % du
			// melange. On ne retient donc que l'etat par defaut.
			b.resoudreSwitch(o, connu)
		default:
			if pr := lireProprietesConteneur(o.Data); pr.Lu && pr.VolumeDB != 0 {
				b.VolNoeud[o.ID] = pr.VolumeDB
			}
			if enf := lireEnfants(o.Data, connu); len(enf) > 0 {
				b.Enfants[o.ID] = enf
			}
		}
	}
	return b, nil
}

// resoudreSwitch retient, pour un conteneur `Switch`, les enfants de son etat par defaut.
//
// Trois issues, et chacune est COMPTEE plutot que silencieuse :
//
//   - etat par defaut porteur d'enfants : c'est lui qu'on retient ;
//   - etat par defaut vide : le conteneur ne joue RIEN tant que le jeu n'impose pas d'etat.
//     On n'invente aucun repli — 200 etats sur l'ensemble des banks sont declares sans
//     aucun enfant, c'est une situation normale du format, pas un echec de lecture ;
//   - table non decodee : on retombe sur l'heuristique generique, faute de mieux.
func (b *bank) resoudreSwitch(o objetHIRC, connu func(uint32) bool) {
	c := lireSwitch(o.Data, connu)
	if !c.Lu {
		b.SwNonLus++
		if enf := lireEnfants(o.Data, connu); len(enf) > 0 {
			b.Enfants[o.ID] = enf
		}
		return
	}
	b.Switchs[o.ID] = c
	enf := c.EnfantsParDefaut()
	if len(enf) == 0 {
		b.SwVides++
		return
	}
	b.SwParDefaut++
	b.Enfants[o.ID] = enf
}

// resoudreBlend retient les enfants d'un `Blend` AUDIBLES au point de reference — la
// courbe de fondu tranche, pas un choix a nous (etape 18.2 : 91 Blend a courbes, chacun
// garde ~1 enfant sur 2,4, zero gain partiel). Sans courbe, tous les enfants jouent.
func (b *bank) resoudreBlend(o objetHIRC, connu func(uint32) bool) {
	enf := lireEnfants(o.Data, connu)
	if len(enf) == 0 {
		return
	}
	c := lireBlend(o.Data, connu)
	if !c.Lu || !c.PiloteParRTPC() {
		b.Enfants[o.ID] = enf
		return
	}
	aud := c.Audibles(enf)
	garde := make([]uint32, 0, len(aud))
	for _, e := range enf {
		g, ok := aud[e]
		if !ok {
			continue
		}
		garde = append(garde, e)
		if g != 0 {
			if b.GainsFondu[o.ID] == nil {
				b.GainsFondu[o.ID] = map[uint32]float64{}
			}
			b.GainsFondu[o.ID][e] = g
		}
	}
	b.Enfants[o.ID] = garde
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

// actionPlay : octet de poids fort du type d'action correspondant a « jouer ».
//
// UNE ACTION N'EST PAS FORCEMENT UN SON A JOUER. L'audit du format le montre : sur 41 464
// actions, 38 089 sont des `Play` mais 1 976 sont des `Stop`, 912 des `Break`, et le reste
// des `Mute`, `SetLPF`, `SetState`... Le parseur ne lisait que la CIBLE, jamais le TYPE :
// la cible d'un `Stop` etait donc empilee comme une couche a jouer, ajoutant au mixage un
// son que le moteur, lui, arrete. C'est une cause directe de coups reconstitues qui
// sonnent moins juste qu'un `.wem` isole.
const actionPlay = 0x04

// lireCibleAction lit le TYPE et l'objet vise par une Event Action.
//
//	u16 typeAction | u32 idCible   (layout courant ; le repli u8 couvre les variantes)
//
// Rend aussi si l'action est de type « jouer » : seules celles-la contribuent au son.
func lireCibleAction(d []byte, connu func(uint32) bool) (uint32, bool, bool) {
	if len(d) < 2 {
		return 0, false, false
	}
	estPlay := byte(binary.LittleEndian.Uint16(d)>>8) == actionPlay
	for _, off := range []int{2, 1} {
		if off+4 > len(d) {
			continue
		}
		id := binary.LittleEndian.Uint32(d[off:])
		if id != 0 && connu(id) {
			return id, estPlay, true
		}
	}
	return 0, estPlay, false
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
