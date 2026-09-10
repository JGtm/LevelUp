package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/canonical"
)

// identity_registry_scoreboard_test.go — LES PROPRIETES DU NOMMAGE PAR LE TABLEAU DE L'API.
//
// Chaque test porte une regle du lot 4.3, et chacune a sa MUTATION. La plus importante est
// [TestTableauDepartageUnIndexPartageParUnBotEtUnArrivant] : elle rejoue la figure mesuree sur
// `4f77afc1` — l'index 26 declare a la fois par `BOT_METADATA` (343 Doomfruit) et par la table
// d'index (Narotlcs, arrive en cours) —, et elle exige que la FENETRE DE PARTICIPATION du
// tableau tranche, jamais l'ordre de lecture.

// tableauDe fabrique une ligne de tableau pour un participant arrive EN COURS.
func tableauDe(id string, arriveeMS int64) Participant {
	ms := arriveeMS
	return Participant{ID: id, JoinedInProgress: true, JoinMatchMS: &ms}
}

// filmSiegePartage : le slot 300 porte DEUX vies, et son index de participant (9) est declare a
// la fois par un bot (`BOT_METADATA`) et par la table d'index (le xuid 222, arrive a 10 s).
// Le slot 100 porte le joueur 111 et ses morts — c'est lui qui CALE l'horloge du fil sur celle
// du film, sans quoi aucune fenetre de participation n'est exprimable.
func filmSiegePartage() IdentityInput {
	var pos []filmdec.BipedPosition
	for t := uint64(1_000_000); t <= 4_000_000; t += 500_000 {
		pos = append(pos, posAt(100, t, 1, 1, 1))
	}
	for t := uint64(20_000_000); t <= 23_000_000; t += 500_000 {
		pos = append(pos, posAt(100, t, 1, 1, 1))
	}
	// Le siege partage : une vie AVANT l'arrivee de 222, une vie APRES.
	for t := uint64(1_000_000); t <= 4_000_000; t += 500_000 {
		pos = append(pos, posAt(300, t, 3, 3, 3))
	}
	for t := uint64(20_000_000); t <= 24_000_000; t += 500_000 {
		pos = append(pos, posAt(300, t, 3, 3, 3))
	}
	return IdentityInput{
		Positions: pos,
		BipedCreations: []filmdec.BipedCreation{
			creationDe(100, 1_000_000, 0), creationDe(300, 1_000_000, 9),
		},
		Deaths: []Death{
			{XUID: 111, Gamertag: "MORTEL", TimeMS: 4_000},
			{XUID: 111, Gamertag: "MORTEL", TimeMS: 23_000},
		},
		PlayerIndices: PlayerIndexTable{ByXUID: map[uint64]int{111: 0, 222: 9}, Readings: 26},
		Bots:          []BotIdentity{{FilmIndex: 9, Name: "343 Doomfruit [bot]", BotID: 7}},
		Clock:         IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 241},
		MatchID:       "test",
	}
}

// TestTableauNommeLeCorpsDunBotQueLaTableIgnore : l'index lu est celui d'un bot que le film
// DECLARE et que le tableau de l'API porte — la vie prend son `bid`, la voie est nommee, et la
// cause `index_hors_table` DISPARAIT pour cette vie.
//
// MUTATION : retirer l'appel a `resolveByScoreboard` -> les vies du slot 200 restent
// `non_resolu` / `index_hors_table`, rouge.
func TestTableauNommeLeCorpsDunBotQueLaTableIgnore(t *testing.T) {
	in := entreeHorsTable()
	in.Bots = []BotIdentity{{FilmIndex: 8, Name: "343 Donos [bot]", BotID: 3}}
	in.Participants = []Participant{{ID: "111"}, {ID: "222"}, {ID: "bid(3.0)"}}
	reg := BuildIdentityRegistry(in)
	var vues int
	for _, l := range reg.Vies() {
		if l.slot != 200 {
			continue
		}
		vues++
		if l.bid != "bid(3.0)" {
			t.Fatalf("vie du slot 200 : bid = %q, attendu \"bid(3.0)\"", l.bid)
		}
		if l.nomPar != NomParTableauAPI {
			t.Fatalf("vie du slot 200 : voie = %q, attendu %q", l.nomPar, NomParTableauAPI)
		}
		if l.xuid != 0 {
			t.Fatalf("un xuid a ete fabrique pour un bot : %d", l.xuid)
		}
	}
	if vues == 0 {
		t.Fatal("aucune vie sur le slot 200 : la figure a change")
	}
	if reg.ViesNommeesParLeTableau() != vues {
		t.Fatalf("vies nommees par le tableau = %d, attendu %d",
			reg.ViesNommeesParLeTableau(), vues)
	}
	c := reg.Section.Coverage.BipedSlot
	if c.UnresolvedByCause.IndexOutOfTable != 0 {
		t.Fatalf("index_hors_table = %d, attendu 0 — le compteur doit BAISSER, jamais etre masque",
			c.UnresolvedByCause.IndexOutOfTable)
	}
	if c.External != vues {
		t.Fatalf("couverture `externe` = %d, attendu %d", c.External, vues)
	}
}

// TestTableauSeTaitSansLigneDeTableau : sans tableau, RIEN ne bouge — la degradation est
// complete et le compteur `index_hors_table` reste ce qu'il etait.
//
// MUTATION : poser le lien sans exiger la ligne du tableau -> le compteur tombe a 0, rouge.
func TestTableauSeTaitSansLigneDeTableau(t *testing.T) {
	in := entreeHorsTable()
	in.Bots = []BotIdentity{{FilmIndex: 8, Name: "343 Donos [bot]", BotID: 3}}
	reg := BuildIdentityRegistry(in)
	if reg.ViesNommeesParLeTableau() != 0 {
		t.Fatalf("vies nommees sans tableau = %d, attendu 0 — un lien que l'API ne donne pas "+
			"ne se devine pas", reg.ViesNommeesParLeTableau())
	}
	if reg.Section.Coverage.BipedSlot.UnresolvedByCause.IndexOutOfTable == 0 {
		t.Fatal("le compteur `index_hors_table` a disparu sans que rien ne le resolve")
	}
}

// TestTableauDepartageUnIndexPartageParUnBotEtUnArrivant : la figure de `4f77afc1`. L'index est
// declare par un bot ET par un humain arrive en cours ; la FENETRE du tableau tranche par vie.
//
// MUTATION : ignorer `JoinMatchMS` et rendre le bot pour tout le slot -> la vie tardive porte le
// bot au lieu du xuid 222, rouge. C'est exactement le lien APLATI que le defaut P0-2 interdit.
func TestTableauDepartageUnIndexPartageParUnBotEtUnArrivant(t *testing.T) {
	in := filmSiegePartage()
	in.Participants = []Participant{{ID: "111"}, {ID: "bid(7.0)"}, tableauDe("222", 10_000)}
	reg := BuildIdentityRegistry(in)
	var avant, apres int
	for _, l := range reg.Vies() {
		if l.slot != 300 {
			continue
		}
		if l.from < 10_000_000 {
			avant++
			if l.bid != "bid(7.0)" || l.xuid != 0 {
				t.Fatalf("vie [%d..%d] AVANT l'arrivee : bid = %q, xuid = %d — attendu le bot",
					l.from, l.to, l.bid, l.xuid)
			}
			continue
		}
		apres++
		if l.xuid != 222 || l.bid != "" {
			t.Fatalf("vie [%d..%d] APRES l'arrivee : xuid = %d, bid = %q — attendu 222",
				l.from, l.to, l.xuid, l.bid)
		}
		if l.nomPar != NomParTableauAPI {
			t.Fatalf("voie = %q, attendu %q", l.nomPar, NomParTableauAPI)
		}
	}
	if avant != 1 || apres != 1 {
		t.Fatalf("vies du siege partage : %d avant / %d apres, attendu 1 / 1", avant, apres)
	}
}

// TestTableauSeTaitSurUneVieQuiEnjambeLArrivee : une vie a cheval sur l'instant d'arrivee
// appartient aux DEUX candidats ; LE TABLEAU se tait et la COMPTE. Les voies suivantes du
// registre (elimination, exclusion) restent libres de la nommer — elles disent d'ou elles
// viennent, et ce n'est pas `tableau_api`.
//
// MUTATION : rendre un candidat sur le chevauchement -> la vie porte `tableau_api` alors que
// rien ne le prouve, et `ViesConflitAuTableau` tombe a 0, rouge.
func TestTableauSeTaitSurUneVieQuiEnjambeLArrivee(t *testing.T) {
	in := filmSiegePartage()
	// Arrivee A 21 s : la seconde vie du slot 300 court de 20 s a 24 s, elle l'enjambe.
	in.Participants = []Participant{{ID: "111"}, {ID: "bid(7.0)"}, tableauDe("222", 21_000)}
	reg := BuildIdentityRegistry(in)
	for _, l := range reg.Vies() {
		if l.slot != 300 || l.from < 10_000_000 {
			continue
		}
		if l.nomPar == NomParTableauAPI {
			t.Fatalf("vie a cheval sur l'arrivee nommee par le tableau (xuid=%d bid=%q) — "+
				"rien ne le prouve", l.xuid, l.bid)
		}
	}
	if reg.ViesConflitAuTableau() == 0 {
		t.Fatal("le conflit n'est pas compte : un refus muet est ce que le registre interdit")
	}
}

// TestUneDeductionNEcrasePasLeBotDuTableau : l'ELIMINATION sur le roster raisonne sur un SLOT
// entier ; sur un siege partage, elle reprenait les vies que le tableau venait d'attribuer au
// bot. Une deduction ne remplace jamais une source.
//
// MUTATION : retirer la garde `bid != ""` de `poserIdentiteDeVie` -> la vie du bot repasse au
// xuid 222, rouge.
func TestUneDeductionNEcrasePasLeBotDuTableau(t *testing.T) {
	in := filmSiegePartage()
	in.Participants = []Participant{{ID: "111"}, {ID: "bid(7.0)"}, tableauDe("222", 21_000)}
	in.RosterXUIDs = []uint64{111, 222}
	for _, l := range BuildIdentityRegistry(in).Vies() {
		if l.slot != 300 || l.from >= 10_000_000 {
			continue
		}
		if l.bid != "bid(7.0)" || l.xuid != 0 {
			t.Fatalf("la vie du bot a ete reprise par une deduction : bid = %q, xuid = %d",
				l.bid, l.xuid)
		}
	}
}

// TestCouvertureBipedeVentileEncoreChaqueNonResoluAvecTableau : l'invariant « somme des causes =
// non_resolu » survit a la nouvelle voie. Une vie que le tableau refuse garde sa cause.
func TestCouvertureBipedeVentileEncoreChaqueNonResoluAvecTableau(t *testing.T) {
	in := filmSiegePartage()
	in.Participants = []Participant{{ID: "111"}, {ID: "bid(7.0)"}, tableauDe("222", 21_000)}
	c := BuildIdentityRegistry(in).Section.Coverage.BipedSlot
	if c.UnresolvedByCause.Total() != c.Unresolved {
		t.Fatalf("causes = %d, non_resolu = %d (%+v)", c.UnresolvedByCause.Total(), c.Unresolved, c)
	}
}

// TestSectionPublieLeBidEtSaProvenance : le lien du tableau se PUBLIE — identifiant, voie
// canonique et provenance `externe`. Un lien pose et non publie serait invisible au gate corpus.
func TestSectionPublieLeBidEtSaProvenance(t *testing.T) {
	in := entreeHorsTable()
	in.Bots = []BotIdentity{{FilmIndex: 8, Name: "343 Donos [bot]", BotID: 3}}
	in.Participants = []Participant{{ID: "bid(3.0)"}}
	sec := BuildIdentityRegistry(in).Section
	var vu bool
	for _, b := range sec.BipedSlots {
		if b.Slot != 200 {
			continue
		}
		vu = true
		if b.Bid != "bid(3.0)" {
			t.Fatalf("ligne du slot 200 : bid = %q, attendu \"bid(3.0)\"", b.Bid)
		}
		if b.Link.Source != canonical.LinkExternal {
			t.Fatalf("provenance = %q, attendu %q", b.Link.Source, canonical.LinkExternal)
		}
		if b.Link.Method != canonical.MethodScoreboard {
			t.Fatalf("voie = %q, attendu %q", b.Link.Method, canonical.MethodScoreboard)
		}
	}
	if !vu {
		t.Fatal("le slot 200 n'est pas publie dans la section")
	}
}
