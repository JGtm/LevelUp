package replay

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// identity_registry_bridge_test.go — LE PONT PAR MORTS EN TEMOIN, ET LA PAIRE ECHANGEE.
//
// Le sondage E2 a mesure SEPT PAIRES EXACTEMENT ECHANGEES entre deux vies qui se terminent a la
// meme image : l'appariement glouton du pont y departage par l'ordre des slots. La fixture
// ci-dessous porte la MEME propriete — le pont attribue a une vie le joueur que le film ecrit sur
// une AUTRE — dans une forme DETERMINEE (cf. la doc de `filmPaireEchangee`), et exige que le film
// fasse foi.

// filmPaireEchangee : trois corps, deux joueurs, et DEUX NOMS CROISES.
//
//	slot 300 (joueur 111 selon le film) se termine a 3 000 ms · mort de 111 a 3 000 ms  -> ACCORD
//	slot 100 (joueur 111 selon le film) se termine a 5 000 ms · mort de 222 a 5 000 ms  -> CROISE
//	slot 200 (joueur 222 selon le film) se termine a 9 000 ms · mort de 111 a 9 000 ms  -> CROISE
//
// LE CROISEMENT EST DETERMINE PAR LE CALAGE, ET C'EST VOULU : les trois fins de vie sont assez
// espacees pour qu'un seul calage apparie les trois morts (tout autre en apparie au plus une), si
// bien que la fixture ne depend pas d'un departage d'egalite. Elle porte la MEME propriete que la
// figure du sondage — le pont attribue a une vie le joueur que le film ecrit sur une AUTRE —
// sans en heriter l'indetermination : quand deux fins tombent au meme instant, l'ordre des
// candidats de meme ecart n'est pas garanti, et un test ne se batit pas sur cela.
func filmPaireEchangee() IdentityInput {
	var pos []filmdec.BipedPosition
	corps := []struct {
		slot  uint32
		finUS uint64
		index uint32
	}{
		{100, 5_000_000, 0}, {200, 9_000_000, 1}, {300, 3_000_000, 0},
	}
	var creations []filmdec.BipedCreation
	for _, c := range corps {
		for t := uint64(1_000_000); t < c.finUS; t += 500_000 {
			pos = append(pos, posAt(c.slot, t, 1, 1, 1))
		}
		pos = append(pos, posAt(c.slot, c.finUS, 1, 1, 1))
		creations = append(creations, creationDe(c.slot, 1_000_000, c.index))
	}
	return IdentityInput{
		Positions: pos, BipedCreations: creations,
		Deaths: []Death{
			{XUID: 111, Gamertag: "A", TimeMS: 3_000},
			{XUID: 222, Gamertag: "B", TimeMS: 5_000},
			{XUID: 111, Gamertag: "A", TimeMS: 9_000},
		},
		PlayerIndices: PlayerIndexTable{ByXUID: map[uint64]int{111: 0, 222: 1}, Readings: 26},
		Clock:         IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 91},
		MatchID:       "test",
	}
}

// TestPontParMortsNEcrasePasLeLienDirect — LA MUTATION DU LOT E2.
//
// MUTATION : remettre le pont AVANT le direct (appeler `nommerParLesMorts` sans condition, ou
// avant `nommerViesParCreations`) -> le slot 100 porte 222 et le slot 200 porte 111, ROUGE. C'est
// la paire echangee, et c'est ce que la production servait jusqu'a ce lot.
func TestPontParMortsNEcrasePasLeLienDirect(t *testing.T) {
	reg := BuildIdentityRegistry(filmPaireEchangee())
	attendu := map[uint32]uint64{100: 111, 200: 222}
	for _, l := range reg.Vies() {
		if want, ok := attendu[l.slot]; ok && l.xuid != want {
			t.Fatalf("slot %d : xuid = %d, attendu %d — le pont a echange les deux noms que le "+
				"film ecrit", l.slot, l.xuid, want)
		}
		if !nomParLecture(l.nomPar) {
			t.Fatalf("slot %d : voie = %q, attendu une LECTURE", l.slot, l.nomPar)
		}
	}
}

// TestPontParMortsCompteSesDiscordances : le desaccord n'ecrase pas, il s'inscrit. Sans ces deux
// compteurs, la correction du lot serait muette — et personne ne saurait que le pont se trompait.
func TestPontParMortsCompteSesDiscordances(t *testing.T) {
	reg := BuildIdentityRegistry(filmPaireEchangee())
	if reg.PontDiscordant() != 2 {
		t.Fatalf("discordances = %d, attendu 2 (la paire echangee compte pour ses deux vies) ; "+
			"concordances = %d", reg.PontDiscordant(), reg.PontConcordant())
	}
	if reg.PontConcordant() != 1 {
		t.Fatalf("concordances = %d, attendu 1 (la vie du slot 300, seule non croisee)",
			reg.PontConcordant())
	}
	if got := reg.SanteDuPont(); got.Discordant != 2 || got.Concordant != 1 {
		t.Fatalf("sante du pont = concordant %d, discordant %d ; attendu 1 et 2",
			got.Concordant, got.Discordant)
	}
}

// TestPontParMortsNeNommeAucuneVieQuandLeFilmNomme — L'OBJECTIF CHIFFRE DU LOT : zero vie nommee
// par le pont des lors que le film porte ses records de creation.
func TestPontParMortsNeNommeAucuneVieQuandLeFilmNomme(t *testing.T) {
	reg := BuildIdentityRegistry(filmPaireEchangee())
	if reg.ViesNommeesParLePont() != 0 {
		t.Fatalf("vies nommees par le pont = %d, attendu 0 : le film les nomme toutes",
			reg.ViesNommeesParLePont())
	}
	for _, l := range reg.Vies() {
		if l.nomPar == NomParMort {
			t.Fatalf("slot %d [%d..%d] porte la voie `death` — le pont ne nomme plus",
				l.slot, l.from, l.to)
		}
	}
	// LA CAUSE DE FIN, ELLE, RESTE CELLE DU FIL DES MORTS : le declassement porte sur le NOM, pas
	// sur la fin. Les confondre est ce qui a coute une lecture d'isolement entiere.
	var mortes int
	for _, l := range reg.Vies() {
		if l.cause == CauseVieMort {
			mortes++
		}
	}
	if mortes != 3 {
		t.Fatalf("vies terminees par une mort = %d, attendu 3 : le fil des morts reste la SEULE "+
			"source qui dise « ce joueur est mort »", mortes)
	}
}

// TestSectionPublieLaVoieDeCreationEtSaProvenance : la section porte `direct` + la voie exacte,
// et non un `deduit` qui ferait passer une lecture pour une supposition.
func TestSectionPublieLaVoieDeCreationEtSaProvenance(t *testing.T) {
	reg := BuildIdentityRegistry(filmPaireEchangee())
	for _, b := range reg.Section.BipedSlots {
		if b.Link.Source != canonical.LinkDirect {
			t.Fatalf("slot %d : source = %q, attendu %q",
				b.Slot, b.Link.Source, canonical.LinkDirect)
		}
		if b.Link.Method != canonical.MethodBipedCreation {
			t.Fatalf("slot %d : voie = %q, attendu %q",
				b.Slot, b.Link.Method, canonical.MethodBipedCreation)
		}
	}
}
