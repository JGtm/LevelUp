package grammar

// vue_a_lecture.go — LA LECTURE COMPLETE DE LA VUE A PAR SA GRAMMAIRE, LA SEULE DE LA COUCHE GRAMMAR
// (lots LN et VA de la campagne de grammaire).
//
// # CE QUE LE JEU ECRIT, ET POURQUOI LA FIN DE LA VUE A EST LE DEBUT DE LA VUE B
//
// L enregistreur par tick (`FUN_142f2c3b0`) ecrit la vue A par `FUN_142f2c050` : il recopie, sans
// prefixe de longueur, les messages deja serialises par `FUN_140bbd474` (`1`, genre `R(7)`, trois
// references gardees, charge du genre par `vtable + 0x60`), puis le terminateur `0`
// (`FUN_1406d49c4`). La vue B (`FUN_142f2cc78`) commence au bit suivant. Le lecteur est
// `FUN_14076a1c4` : `{ R(1) ; 0 -> fin ; FUN_14080a9d4 }`, et `FUN_14080a9d4` lit :
//
//	genre = R(7) ; genre >= 0x7b -> code 3 (le message est refuse)
//	pour i = 0, 1, 2 : R(1) garde ; si 1 : FUN_1406d3140(domaine vtable+0x58(i))
//	si taille (vtable + 0x10) > 0 :
//	    charge : vtable + 0x68(..., param_5 = 1) ; faux -> code 3
//	    si FUN_14076cea8() : R(1) ; si 1 : R(32)       (controle de corruption du film)
//
// Rien dans le flux ne donne la longueur d un message : la vue A ne se lit jusqu au bout qu en
// lisant la charge de CHAQUE message. Quand elle l est, le bit qui suit son terminateur EST le
// premier bit de la vue B — une lecture, pas une recherche.
//
// # UNE SEULE LECTURE COMPLETE
//
// [lireLaVueA] est, dans la couche grammar, la seule lecture de la vue A par sa GRAMMAIRE jusqu a
// son terminateur, message par message, charge comprise (garde-rail :
// `archlint/film_vue_a_lecteur_unique_test.go`). Deux autres lectures existent hors de ce contrat :
// la tete seule — le bit de configuration, la continuation et le genre du premier message — est
// relue par [readPacketHead], et par lui [teteDuPayload], [lireEnteteTir36] et [scanChunkDamages]
// (ces deux derniers lisent encore le corps du premier message pour leur canal), et, dans la couche
// facts, par `killsource` (`hasEvents`, `estAncreDeKillEvent`) ; et `killsource` garde sa propre
// lecture en CHAINE des evenements (`facts/killsource/eventchain.go`, portage de `FUN_14076a1c4` et
// `FUN_14080a9d4`), qui suit la vue A message par message depuis chaque evenement de mort candidat
// jusqu au terminateur ou a une longueur bornee — sa lecture pour son canal, pas celle-ci. La marche des trames joue [lireLaVueA] une fois par trame,
// en rangeant la tete ([rangerLaTete]), et passe ce qu elle a lu a la marche par rangs
// ([lireTrameParRangs]) ; les autres marches par rangs (essais de localisation, cartes) l appellent
// depuis la tete du paquet ; les deux marches qui lisent les morts sans ranger de structure la
// lisent par [DebutDeLaVueB]. Quand elle atteint le terminateur, sa fin decide du debut de la vue B
// selon la classe du film ([debutParLaVueA], `localisateur.go`). Elle LIT LA TETE A L IDENTIQUE dans tous les cas — la continuation, puis le genre
// du premier message — et ne lit la suite que si le film la rend lisible ; elle ne devine rien : un
// message qu elle ne sait pas lire arrete la lecture, apres son genre.
//
// LE BIT DE CONFIGURATION. `FUN_142987460` lit `DAT_144706104 = FUN_1406cf008(reader)` avant les
// trois vues. `FUN_1406d3140` (la reference de domaine, [readVarWidthInt]) ne prend la plage de sa
// categorie dans la table de `FUN_140d10bb0` que si ce bit vaut 1 ; a 0, la plage est
// `DAT_144706100` pour TOUTES les categories. Les lecteurs portes ici ne portent que la table par
// categorie : un bit a 0 arrete la lecture apres la tete, comme tout message que le lecteur ne sait
// pas lire.
//
// CE QUI REND LA SUITE LISIBLE : un film dont la table des genres est EGALE a la table native ou en
// est un PREFIXE STRICT ([classeDesGenres]), un bit de configuration a 1. Le lecteur porte le profil
// du cadre (controle de corruption, tables du profil), la grammaire de la vue A que le film declare
// ([grammaireDeLaVueA] : table des genres, Script, tables de la region jouee) et AUCUNE
// observation : la vue A ne publie rien.

// FluxVueA est ce que la lecture de la vue A (rang 0) d une trame delta a lu.
//
// `FUN_14076a1c4` ne rend JAMAIS un record (`*param_6 = 0` sans condition) : la vue A est un flux
// de MESSAGES, pas d entites.
type FluxVueA struct {
	// Debut : le premier bit de la vue A (la continuation du premier message).
	Debut int
	// Vide : la vue n a ecrit que son terminateur (UN bit).
	Vide bool
	// Genres : les selecteurs `R(7)` des messages lus, dans l ordre, puis celui du message qui a
	// arrete la lecture. Le premier est le genre de la TETE.
	Genres []int
	// Porte : la vue a ete lue jusqu a son terminateur.
	Porte bool
	// Fin : la fin de l etendue lue. Vue portee : le bit qui suit le terminateur — le debut de la
	// vue B chez l ecrivain. Vue arretee : le bit qui suit le genre du message qui l a arretee, ou
	// la position ou le payload s est epuise.
	Fin int
}

// finDeTete rend le bit qui suit la tete : la continuation et, quand elle annonce un message, son
// genre. C est la position ou la lecture d avant le lot VA s arretait.
func (a *FluxVueA) finDeTete() int {
	if len(a.Genres) > 0 {
		return a.Debut + 1 + LargeurGenreVueA
	}
	return a.Fin
}

// lireLaVueA lit la vue A d un payload de trame delta qui commence au bit `debut`, le bit de
// configuration du frame-processeur au bit `debut - 1` (`FUN_142987460`, puis `FUN_14076a1c4`), sous
// le profil `bal` et la grammaire `g` que le film declare.
func lireLaVueA(pay []byte, debut int, bal ProfilDeBalayage, g grammaireDeLaVueA) FluxVueA {
	br := LecteurSur(pay)
	br.PoserProfil(bal)
	br.vueA = g
	frameLen := len(pay) * 8
	lisible := false
	if debut >= 1 && debut <= frameLen {
		br.SetBitPos(debut - 1)
		lisible = br.ReadBit() && g.classe != vueAIllisible // DAT_144706104
	}
	out := FluxVueA{Debut: debut}
	br.SetBitPos(debut)
	for placeDisponible(br, frameLen, 1) {
		if !br.ReadBit() { // le terminateur
			out.Porte, out.Vide = true, len(out.Genres) == 0
			break
		}
		if !placeDisponible(br, frameLen, LargeurGenreVueA) {
			break
		}
		genre := int(br.ReadBits(LargeurGenreVueA))
		out.Genres = append(out.Genres, genre)
		finDuGenre := br.BitPos()
		if !lisible || !lireUnMessage(br, genre) || br.Deborde() {
			br.SetBitPos(finDuGenre)
			break
		}
	}
	out.Fin = br.BitPos()
	return out
}

// lireUnMessage lit le corps d un message de genre `genre` (`FUN_14080a9d4` apres `R(7)`). Un genre
// au-dela du cardinal du film n existe pas chez son ecrivain : il arrete la lecture.
func lireUnMessage(br *Lecteur, genre int) bool {
	if genre >= br.vueA.genres {
		return false
	}
	domaines, vide := descripteurDuGenre(genre)
	for _, dom := range domaines {
		if br.ReadBit() { // FUN_1406cf008 : garde de la reference
			readVarWidthInt(br, dom) // FUN_1406d3140
		}
	}
	if vide {
		return true
	}
	charge := chargeDuGenre(genre)
	if charge == nil || !charge(br) {
		return false
	}
	if br.p.Grammaire.ControleDeCorruption && br.ReadBit() { // FUN_14076cea8, R(1)
		br.Skip(32)
	}
	return true
}
