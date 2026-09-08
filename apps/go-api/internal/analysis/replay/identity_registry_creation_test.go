package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/canonical"
)

// identity_registry_creation_test.go — LES PROPRIETES DU LIEN DIRECT CORPS -> JOUEUR.
//
// Chaque test porte une regle du lot E2, et chacune a sa MUTATION. La plus importante est
// [TestPontParMortsNEcrasePasLeLienDirect] : elle rejoue la figure exacte que le sondage a
// mesuree — deux vies qui se terminent a la MEME image, que l'appariement glouton du pont
// echange —, et elle exige que le film fasse foi.

// creationDe fabrique un record de creation lu, a la date et pour l'index donnes.
func creationDe(slot uint32, tUS uint64, index uint32) filmdec.BipedCreation {
	return filmdec.BipedCreation{
		Slot: slot, Generation: 1, ParticipantIndex: index, HasIndex: true,
		TimestampUS: tUS, Version: 13, Representation: filmdec.BipedRepresentationName,
	}
}

// filmDeuxCorps : deux slots, deux joueurs, et le slot 100 coupe en DEUX sejours par un trou de
// plus de `lifeGapUS` — la figure (a) de l'instruction du residu. Un seul record de creation par
// corps, comme les cinq films mesures.
func filmDeuxCorps() IdentityInput {
	var pos []filmdec.BipedPosition
	for t := uint64(1_000_000); t <= 4_000_000; t += 500_000 {
		pos = append(pos, posAt(100, t, 1, 1, 1))
	}
	// Le MEME corps, apres un trou de 10 s : la decoupe en fera une seconde vie.
	for t := uint64(14_000_000); t <= 18_000_000; t += 500_000 {
		pos = append(pos, posAt(100, t, 1, 1, 1))
	}
	for t := uint64(1_000_000); t <= 18_000_000; t += 500_000 {
		pos = append(pos, posAt(200, t, 2, 2, 2))
	}
	return IdentityInput{
		Positions: pos,
		BipedCreations: []filmdec.BipedCreation{
			creationDe(100, 1_000_000, 0),
			creationDe(200, 1_000_000, 1),
		},
		PlayerIndices: PlayerIndexTable{ByXUID: map[uint64]int{111: 0, 222: 1}, Readings: 26},
		Clock:         IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 171},
		MatchID:       "test",
	}
}

// TestCreationNommeLaVieQuiPorteSonRecord : le lien est DIRECT, la voie est nommee, et le xuid
// vient de la table d'index — jamais d'une mort.
//
// MUTATION : retirer l'appel a `nommerViesParCreations` -> les vies sont anonymes (aucune mort
// dans cette entree), rouge.
func TestCreationNommeLaVieQuiPorteSonRecord(t *testing.T) {
	reg := BuildIdentityRegistry(filmDeuxCorps())
	var vu bool
	for _, l := range reg.Vies() {
		if l.slot != 100 || l.from != 1_000_000 {
			continue
		}
		vu = true
		if l.xuid != 111 {
			t.Fatalf("vie 100[1000000..] : xuid = %d, attendu 111 (index 0 du record)", l.xuid)
		}
		if l.nomPar != NomParCreation {
			t.Fatalf("voie = %q, attendu %q", l.nomPar, NomParCreation)
		}
	}
	if !vu {
		t.Fatal("la premiere vie du slot 100 n'existe pas : la decoupe a change")
	}
	if reg.ViesParCreation() != 2 {
		t.Fatalf("vies nommees par un record date DANS leur intervalle = %d, attendu 2",
			reg.ViesParCreation())
	}
}

// TestCreationSePropageAuxAutresViesDuMemeCorps — LA REGLE (a) DE L'INSTRUCTION DU RESIDU.
//
// La seconde vie du slot 100 ne porte AUCUN record : elle est le meme corps, separe par un trou
// de replication. Le lien s'y propage, il reste DIRECT, et sa voie le dit.
//
// MUTATION : rendre `indexPour` sans repli (« un record dans l'intervalle, ou rien ») -> la
// seconde vie du slot 100 reste anonyme, rouge.
func TestCreationSePropageAuxAutresViesDuMemeCorps(t *testing.T) {
	reg := BuildIdentityRegistry(filmDeuxCorps())
	var propagees int
	for _, l := range reg.Vies() {
		if l.slot == 100 && l.from > 10_000_000 {
			if l.xuid != 111 {
				t.Fatalf("seconde vie du slot 100 : xuid = %d, attendu 111 (meme corps)", l.xuid)
			}
			if l.nomPar != NomParCreationPropagee {
				t.Fatalf("voie = %q, attendu %q", l.nomPar, NomParCreationPropagee)
			}
			propagees++
		}
	}
	if propagees != 1 {
		t.Fatalf("vies propagees trouvees = %d, attendu 1", propagees)
	}
	if reg.ViesParCreationPropagee() != 1 {
		t.Fatalf("compteur de propagation = %d, attendu 1", reg.ViesParCreationPropagee())
	}
	// LA PROPAGATION EST UNE LECTURE, PAS UNE DEDUCTION : la couverture doit la compter dans
	// `direct`, sinon un gate qui surveille `direct` verrait le lot faire baisser la lecture.
	c := reg.Section.Coverage.BipedSlot
	if c.Direct != 3 || c.DirectPropagated != 1 || c.Inferred != 0 {
		t.Fatalf("couverture = direct %d (dont propages %d), deduit %d ; attendu 3, 1, 0",
			c.Direct, c.DirectPropagated, c.Inferred)
	}
}

// TestCreationSeTaitSurDesLecturesDivergentes : deux records du MEME slot portant des index
// DIFFERENTS. Seule la vie qui CONTIENT la date d'un record est nommee ; les autres passent
// `non_resolu`, cause `lectures_divergentes`.
//
// LE REFUS EST MESURE A ZERO SUR LES CINQ FILMS, et c'est pour cela qu'il doit etre teste : rien
// dans le materiau d'aujourd'hui ne le declencherait, donc rien ne dirait qu'il a disparu.
//
// MUTATION : faire rendre a `indexPour` le premier index quand les lectures divergent -> la
// seconde vie du slot 100 est nommee, rouge.
func TestCreationSeTaitSurDesLecturesDivergentes(t *testing.T) {
	in := filmDeuxCorps()
	in.BipedCreations = append(in.BipedCreations, creationDe(100, 30_000_000, 1))
	reg := BuildIdentityRegistry(in)
	for _, l := range reg.Vies() {
		if l.slot == 100 && l.from > 10_000_000 && l.xuid != 0 {
			t.Fatalf("une vie a ete nommee (%d) malgre deux lectures divergentes sur son corps",
				l.xuid)
		}
	}
	c := reg.Section.Coverage.BipedSlot
	if c.UnresolvedByCause.DivergentReadings == 0 {
		t.Fatalf("cause `lectures_divergentes` = 0 : le refus n'est pas compte (%+v)", c)
	}
}

// TestCreationNeRattachePasUnIndexHorsTable — LE VERDICT I0.
//
// L'espace d'index est PARTAGE entre humains et bots : un index lu que la table publiee ne nomme
// pas designe un participant que l'artefact ne porte pas. La vie reste `non_resolu`, cause
// `index_hors_table`, JAMAIS rattachee au premier venu.
//
// MUTATION : rattacher l'index inconnu au xuid le plus proche -> rouge.
func TestCreationNeRattachePasUnIndexHorsTable(t *testing.T) {
	in := filmDeuxCorps()
	in.BipedCreations = []filmdec.BipedCreation{
		creationDe(100, 1_000_000, 0),
		creationDe(200, 1_000_000, 8), // 8 : l'index de `c75f33b8` que rien ne publie
	}
	reg := BuildIdentityRegistry(in)
	for _, l := range reg.Vies() {
		if l.slot == 200 && l.xuid != 0 {
			t.Fatalf("le slot 200 porte le xuid %d alors que son index (8) n'est pas publie", l.xuid)
		}
	}
	c := reg.Section.Coverage.BipedSlot
	if c.UnresolvedByCause.IndexOutOfTable != 1 {
		t.Fatalf("cause `index_hors_table` = %d, attendu 1 (%+v)",
			c.UnresolvedByCause.IndexOutOfTable, c)
	}
}

// TestCouvertureBipedeVentileChaqueNonResolu — L'INVARIANT DE LA VENTILATION.
//
// La somme des causes EGALE `non_resolu`. Un non-resolu sans cause serait exactement le silence
// que le registre existe pour interdire : « non resolu » ne se corrige pas, « sans record » si.
func TestCouvertureBipedeVentileChaqueNonResolu(t *testing.T) {
	for nom, in := range map[string]IdentityInput{
		"lectures divergentes": entreeDivergente(),
		"index hors table":     entreeHorsTable(),
		"aucune creation":      entreeDeuxJoueurs([]uint64{111}),
	} {
		c := BuildIdentityRegistry(in).Section.Coverage.BipedSlot
		if c.UnresolvedByCause.Total() != c.Unresolved {
			t.Errorf("%s : causes = %d, non_resolu = %d (%+v) — un non-resolu sans cause ne se "+
				"corrige pas", nom, c.UnresolvedByCause.Total(), c.Unresolved, c)
		}
	}
}

func entreeDivergente() IdentityInput {
	in := filmDeuxCorps()
	in.BipedCreations = append(in.BipedCreations, creationDe(100, 30_000_000, 1))
	return in
}

func entreeHorsTable() IdentityInput {
	in := filmDeuxCorps()
	in.BipedCreations = []filmdec.BipedCreation{
		creationDe(100, 1_000_000, 0), creationDe(200, 1_000_000, 8),
	}
	return in
}

// TestCreationPubliePourLeSlotDUnBotSonIndexSansXuid : un index de bot DECLARE est lu, compte a
// part, et n'alarme pas — mais il ne pose aucun xuid (verdict I0). Le pont slot -> index, lui,
// le porte : c'est ce qui donne au nommage des pistes de bot une source DIRECTE.
func TestCreationPubliePourLeSlotDUnBotSonIndexSansXuid(t *testing.T) {
	in := filmDeuxCorps()
	in.BipedCreations = []filmdec.BipedCreation{
		creationDe(100, 1_000_000, 0), creationDe(200, 1_000_000, 9),
	}
	in.Bots = []BotIdentity{{FilmIndex: 9, Name: "343 Flippant [bot]", BotID: 7}}
	reg := BuildIdentityRegistry(in)
	if reg.LecturesDIndexDeBot() != 1 {
		t.Fatalf("lectures d'index de bot = %d, attendu 1", reg.LecturesDIndexDeBot())
	}
	if got := reg.IndexParSlot()[200]; got != 9 {
		t.Fatalf("pont slot 200 -> index = %d, attendu 9 : le record le porte, le xuid non", got)
	}
	if reg.PontDeSlot(200) != "" {
		t.Fatalf("un xuid a ete pose sur le corps d'un bot : %q", reg.PontDeSlot(200))
	}
}

// TestCauseSansRecordQuandLeFilmNePorteAucuneCreation : un producteur sans le canal des creations
// degrade EN ENTIER — le pont nomme, et le registre le DIT.
func TestCauseSansRecordQuandLeFilmNePorteAucuneCreation(t *testing.T) {
	reg := BuildIdentityRegistry(entreeDeuxJoueurs(nil))
	if reg.CorpsAvecCreation() != 0 {
		t.Fatalf("corps avec creation = %d, attendu 0", reg.CorpsAvecCreation())
	}
	if reg.ViesNommeesParLePont() == 0 {
		t.Fatal("sans lecture directe, le pont doit nommer — sinon le rejeu perd tout d'un coup")
	}
	if reg.CauseNonResolue(0) != canonical.MethodNoCreationRecord {
		t.Fatalf("cause de la vie 0 = %q, attendu %q",
			reg.CauseNonResolue(0), canonical.MethodNoCreationRecord)
	}
}
