package replay

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
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

// TestCreationPartageUnSlotRecycleEntreSesDeuxCorps : LE POOL DE HANDLES REBOUCLE, et le siege
// change d'occupant. Deux records du MEME slot, dates et d'index differents : chaque vie revient
// au corps que le slot portait AU DEBUT de cette vie.
//
// MESURE QUI L'IMPOSE (E2-bis) : `084a804d` — 379 records pour 256 slots, 123 slots a deux
// records (`gen=1` en tete de film, `gen=2` apres la 11e minute). Le lot E2 refusait ces
// 59 vies en bloc (`lectures_divergentes`), et les calques perdaient avec elles la moitie du
// bornage de leurs episodes.
//
// MUTATION : grouper les records par slot SANS leur date (rendre `indexAuDebutDe` le premier
// index) -> la seconde vie du slot 100 porte 111, rouge.
func TestCreationPartageUnSlotRecycleEntreSesDeuxCorps(t *testing.T) {
	in := filmDeuxCorps()
	// Le second corps du slot 100 nait pendant le trou de replication, avant sa seconde vie ;
	// un TROISIEME sejour suit, que ce record n'ouvre pas — c'est lui qui exerce le partage.
	in.BipedCreations = append(in.BipedCreations, creationDe(100, 12_000_000, 1))
	for t := uint64(24_000_000); t <= 28_000_000; t += 500_000 {
		in.Positions = append(in.Positions, posAt(100, t, 1, 1, 1))
	}
	in.Clock.FrameCount = 281
	reg := BuildIdentityRegistry(in)
	var vues, propagees int
	for _, l := range reg.Vies() {
		if l.slot != 100 {
			continue
		}
		vues++
		attendu := uint64(111)
		if l.from > 10_000_000 {
			attendu = 222
		}
		if l.xuid != attendu {
			t.Fatalf("vie 100[%d..%d] : xuid = %d, attendu %d — chaque vie revient au corps qui "+
				"tenait le slot a son debut", l.from, l.to, l.xuid, attendu)
		}
		if l.nomPar == NomParCreationPropagee {
			propagees++
		}
	}
	if vues != 3 {
		t.Fatalf("vies du slot 100 = %d, attendu 3", vues)
	}
	if propagees != 1 {
		t.Fatalf("vies propagees du slot 100 = %d, attendu 1 (le troisieme sejour, que le second "+
			"record n'ouvre pas)", propagees)
	}
	c := reg.Section.Coverage.BipedSlot
	if c.UnresolvedByCause.DivergentReadings != 0 {
		t.Fatalf("cause `lectures_divergentes` = %d, attendu 0 : un slot recycle n'est pas une "+
			"divergence (%+v)", c.UnresolvedByCause.DivergentReadings, c)
	}
}

// TestCreationSeTaitSurUneVieAnterieureAuxLecturesDivergentes : une vie qui commence AVANT le
// premier record de son slot, sur un slot dont les records portent des index DIFFERENTS. Aucun
// corps n'est etabli a cet instant et departager serait un choix : `non_resolu`, cause
// `lectures_divergentes`.
//
// LE REFUS EST MESURE A ZERO SUR LES DIX TEMOINS (la creation precede toujours la replication),
// et c'est pour cela qu'il doit etre teste : rien dans le materiau d'aujourd'hui ne le
// declencherait, donc rien ne dirait qu'il a disparu.
//
// MUTATION : faire rendre a `indexAuDebutDe` le premier index quand aucun record ne precede la
// vie -> la premiere vie du slot 100 est nommee, rouge.
func TestCreationSeTaitSurUneVieAnterieureAuxLecturesDivergentes(t *testing.T) {
	in := filmDeuxCorps()
	// Les DEUX records du slot 100 sont posterieurs a sa premiere vie ([1 s..4 s]).
	in.BipedCreations = []filmdec.BipedCreation{
		creationDe(100, 12_000_000, 0),
		creationDe(100, 30_000_000, 1),
		creationDe(200, 1_000_000, 1),
	}
	reg := BuildIdentityRegistry(in)
	for _, l := range reg.Vies() {
		if l.slot == 100 && l.from < 10_000_000 && l.xuid != 0 {
			t.Fatalf("la vie 100[%d..%d] a ete nommee (%d) alors qu'aucun record ne la precede "+
				"sur un slot aux lectures divergentes", l.from, l.to, l.xuid)
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

// TestAlarmerSurLesRefusNeSoustraitPasDeuxPopulationsNonComparables — lot 5.1, revue de vague 4,
// constat P2.
//
// # LA FIGURE : DEUX INDEX DE BOT, UN SEUL SURVIT AU TABLEAU
//
// Slot 300 (index 5) : un bot SEUL a cet index, et son `bid` est publie au tableau — la regle 1
// de `identity_registry_scoreboard.go` le resout, la vie SORT du residu.
//
// Slot 400 (index 6) : un bot est DECLARE a cet index (BOT_METADATA), mais son `bid` n'entre PAS
// dans le tableau (`Participants` ne le porte pas) — la vie reste `non_resolu`,
// `index_hors_table`, elle est le SEUL residu.
//
// A LA LECTURE DIRECTE (avant le tableau), les DEUX vies comptent dans `r.IndexBot` (2). APRES
// le tableau, une seule reste dans `causes.IndexOutOfTable` (1, slot 400). L'ANCIEN CALCUL
// (`causes.IndexOutOfTable - r.IndexBot` = 1 - 2 = -1, garde par `causes.IndexOutOfTable >
// r.IndexBot` qui vaut `1 > 2` = FAUX) ne declenche AUCUNE alarme : le residu reel (1 vie
// authentiquement perdue) se tait.
//
// MUTATION : remettre `causes.IndexOutOfTable > r.IndexBot` (et `causes.IndexOutOfTable -
// r.IndexBot` dans le champ `vies`) -> l'alarme disparait, rouge.
func TestAlarmerSurLesRefusNeSoustraitPasDeuxPopulationsNonComparables(t *testing.T) {
	in := filmDeuxCorps()
	for tUS := uint64(1_000_000); tUS <= 2_000_000; tUS += 500_000 {
		in.Positions = append(in.Positions, posAt(300, tUS, 3, 3, 3))
		in.Positions = append(in.Positions, posAt(400, tUS, 4, 4, 4))
	}
	in.BipedCreations = append(in.BipedCreations,
		creationDe(300, 1_000_000, 5), creationDe(400, 1_000_000, 6))
	in.Bots = []BotIdentity{
		{FilmIndex: 5, Name: "Bot Publie [bot]", BotID: 1},
		{FilmIndex: 6, Name: "Bot Sans Ligne [bot]", BotID: 2},
	}
	// SEUL bid(1.0) entre au tableau : bid(2.0) est declare par le film, mais la base n'en porte
	// aucune ligne (le cas qui doit ALARMER — un participant que le film sait nommer, mais dont
	// la base n'a AUCUNE trace, n'est pas une degradation muette).
	in.Participants = []Participant{{ID: "bid(1.0)"}}

	reg := BuildIdentityRegistry(in)
	if reg.creation.IndexBot != 2 {
		t.Fatalf("IndexBot (lecture directe) = %d, attendu 2 — la figure a change", reg.creation.IndexBot)
	}
	c := reg.Section.Coverage.BipedSlot
	if c.UnresolvedByCause.IndexOutOfTable != 1 {
		t.Fatalf("index_hors_table (residu apres tableau) = %d, attendu 1 (slot 400 seul) : %+v",
			c.UnresolvedByCause.IndexOutOfTable, c)
	}

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	reg.creation.alarmerSurLesRefus("test", c.UnresolvedByCause)

	vies := viesAlarmeesIndexHorsTable(t, buf.Bytes())
	if vies != 1 {
		t.Fatalf("alarme index_hors_table : vies = %d, attendu 1 — le residu REEL (1 vie, slot "+
			"400) doit alarmer meme si 2 lectures directes portaient un index de bot", vies)
	}
}

// viesAlarmeesIndexHorsTable extrait le champ `vies` du log JSON de l'alarme
// « index de participant LU mais absent de la table publiee ». Rend -1 si l'alarme n'est pas
// sortie DU TOUT — c'est exactement le silence que l'ancien calcul produisait.
func viesAlarmeesIndexHorsTable(t *testing.T, journal []byte) int {
	t.Helper()
	for _, ligne := range bytes.Split(journal, []byte("\n")) {
		if len(ligne) == 0 {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal(ligne, &rec); err != nil {
			t.Fatalf("ligne de journal illisible : %v (%s)", err, ligne)
		}
		msg, _ := rec["msg"].(string)
		if msg != "rejeu : index de participant LU mais absent de la table publiee — vies NON "+
			"rattachees (verdict I0 : participant que PlayerIndexTable ne nomme pas)" {
			continue
		}
		v, _ := rec["vies"].(float64)
		return int(v)
	}
	return -1
}
