package grammar

// distribuer_tetes.go — LA TETE DE CHAQUE TRAME DELTA, LUE UNE FOIS ET DONNEE AUX CANAUX DE TETE
// (ADR 0037 ; lot 2.3 du plan de l etape 2).
//
// La tete d une trame est le debut de sa vue A : apres le bit de configuration du frame-processeur,
// la continuation de la liste de messages et, quand elle annonce un message, son genre `R(7)`. La
// vue A est lue UNE fois, avec sa tete, par la lecture unique ([lireLaVueA]) : jusqu a son
// terminateur quand le film la rend lisible, sa tete seule sinon. La tete est lue a l identique
// dans les deux cas. Les canaux de tete (tirs, translocations, lunette, ramassages, apparitions,
// vehicules) decodent le corps de leur evenement a partir de cette tete, sans la relire.
//
// La marche des trames range la vue A de chaque paquet ([rangerLaTete]) avant de le marcher ; une
// distribution sans canal des trames ne marche rien et joue la seule PASSE DES TETES
// ([distribuerLesTetes]) : un parcours des paquets, sans record, sans monde, sans registre.

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"

// rangerLaTete lit la vue A du paquet delta `p` (payload non vide) sous le profil `bal` et la
// grammaire `g` que le film declare — debut au bit 1, apres le bit de configuration — la range dans
// `p.VueA` et rend ce qu elle a lu.
func rangerLaTete(p *lecture.Paquet, bal ProfilDeBalayage, g grammaireDeLaVueA) FluxVueA {
	a := lireLaVueA(p.Payload, 1, bal, g)
	rangerLaVueA(p, &a)
	return a
}

// rangerLaVueA range une vue A lue dans `p.VueA` : son etendue, son etat, ses genres et le rang ou
// leur numerotation devient presumee. Une liste vide termine la vue sur son terminateur ; une vue lue
// jusqu a son terminateur la termine apres lui ; un message que la lecture ne sait pas lire l arrete
// apres son genre, et une tete qui ne tient pas dans le payload l arrete sans genre.
func rangerLaVueA(p *lecture.Paquet, a *FluxVueA) {
	p.VueA.Debut, p.VueA.Bits, p.VueA.Etat = uint32(a.Debut), uint32(a.Fin-a.Debut), etatDeVue(a.Porte) //nolint:gosec // positions d un payload
	p.VueA.Genres = p.VueA.Genres[:0]
	for _, g := range a.Genres {
		p.VueA.Genres = append(p.VueA.Genres, uint8(g)) //nolint:gosec // genre R(7)
	}
	p.VueA.PremierPresume = uint16(a.PremierPresume) //nolint:gosec // rang dans la vue A d un payload
}

// listeAnnoncee dit que la continuation de la vue A annonce une liste de messages — un paquet a
// evenements : tout sauf une vue terminee sans message.
func listeAnnoncee(a *lecture.VueA) bool {
	return a.Etat != lecture.VueTerminee || len(a.Genres) > 0
}

// teteDeTrame est la tete d une trame delta : la continuation de sa vue A — une liste de messages
// suit — et, quand elle est posee, le genre du premier message.
type teteDeTrame struct {
	liste bool
	genre int
}

// teteDe rend la tete du paquet delta `p` telle que sa vue A la porte : le genre du PREMIER message
// quand elle en porte un, la liste vide quand elle s est terminee sans message. Une tete qui ne
// tenait pas dans le payload (vue arretee sans genre) garde la lecture TOLERANTE d avant la structure
// ([teteDuPayload], des zeros au-dela du payload) : c est la regle des paquets dont la tete n est
// pas lisible (lot 2.3.2).
func teteDe(p *lecture.Paquet) teteDeTrame {
	a := &p.VueA
	switch {
	case len(a.Genres) > 0:
		return teteDeTrame{liste: true, genre: int(a.Genres[0])}
	case a.Etat == lecture.VueTerminee:
		return teteDeTrame{}
	}
	return teteDuPayload(p.Payload)
}

// teteDuPayload lit la tete d un payload de trame delta hors de la structure ([readPacketHead]) :
// la forme des lecteurs qui balaient des payloads sans marche (precision par arme, visee modale).
func teteDuPayload(pay []byte) teteDeTrame {
	h := readPacketHead(LecteurSur(pay))
	return teteDeTrame{liste: h.More, genre: h.Type}
}

// PacketHeadEventType lit le type de l EVENEMENT DE TETE d un payload de paquet delta, hors de la
// structure ([teteDuPayload]). Si la continuation est nulle, la liste est vide (le paquet est une
// trame de records pure) et present vaut false. Aucune charge n est decodee.
//
// C est la forme des INSTRUMENTS, qui comptent les familles par type de tete ou trient des paquets
// sans marche (fabrique de mini-films du rejeu) ; la production lit la tete rangee dans la vue A de
// la trame ([teteDe]).
func PacketHeadEventType(pay []byte) (typ int, present bool) {
	if len(pay) < 1 {
		return 0, false
	}
	t := teteDuPayload(pay)
	if !t.liste {
		return 0, false
	}
	return t.genre, true
}

// distribuerLesTetes joue la passe des tetes : chaque trame delta du film (type 0, payload non vide),
// dans l ordre du flux, sa tete rangee, donnee a chaque canal de tete. Elle rend le nombre de chunks
// lus. Le paquet rendu est l arene de la passe : il n est valide que le temps de l appel.
func distribuerLesTetes(fc *FilmContext, canaux []CanalDesTetes) int {
	var p lecture.Paquet
	bal, g := fc.profilDeLaVueA(), fc.grammaireDeLaVueA()
	lus := 0
	for _, num := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		lus++
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			viderLePaquet(&p)
			p.Chunk, p.Index, p.Type, p.TS, p.Payload = num, pk.Index, pk.Type, pk.TimestampUS, pk.Payload(data)
			rangerLaTete(&p, bal, g)
			for _, c := range canaux {
				c.Tete(&p)
			}
		}
	}
	return lus
}

// distribuerLesTetesSeules joue la passe des tetes pour des canaux de tete, et les clot.
func distribuerLesTetesSeules(fc *FilmContext, canaux []Canal) {
	distribuerSansMarcherLesTrames(fc, canaux)
}
