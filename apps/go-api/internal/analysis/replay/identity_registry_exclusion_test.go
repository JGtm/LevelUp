package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/canonical"
)

// identity_registry_exclusion_test.go — LES PROPRIETES DE L'EXCLUSION TEMPORELLE.
//
// Chaque test porte une regle, et chacune a sa MUTATION jouee au journal du lot P2-bis
// (`.ai/V7.5/v2/RESTES_P2_2026-09-08.md` §B). Aucune valeur figee : des proprietes.

// sejour ajoute a `pos` les positions d'un sejour continu [debutS, finS] sur un slot, une
// position toutes les 500 ms — bien en deca de `lifeGapUS`, donc un seul sejour.
func sejour(pos []filmdec.BipedPosition, slot uint32, debutS, finS uint64) []filmdec.BipedPosition {
	for t := debutS * 1_000_000; t <= finS*1_000_000; t += 500_000 {
		pos = append(pos, posAt(slot, t, float32(slot), float32(slot), 1))
	}
	return pos
}

// filmExclusion fabrique le film du cas d'ecole : 111 meurt trois fois sur le slot 100, 222 deux
// fois sur le slot 200, et le slot 400 porte une vie qu'AUCUNE mort ne termine, sur un intervalle
// que 111 passe ailleurs. Un seul joueur du roster y est donc libre : 222.
//
// LES DEUX JOUEURS ONT DES VIES NOMMEES : `resolveByRosterElimination` ne peut pas s'appliquer
// (aucun xuid libre au sens du MATCH). C'est bien l'exclusion temporelle qui est exercee.
func filmExclusion() IdentityInput {
	var pos []filmdec.BipedPosition
	pos = sejour(pos, 100, 1, 4)
	pos = sejour(pos, 100, 10, 20)
	pos = sejour(pos, 100, 26, 34)
	pos = sejour(pos, 200, 1, 8)
	pos = sejour(pos, 200, 30, 36)
	pos = sejour(pos, 400, 16, 24)
	deaths := []Death{
		{XUID: 111, Gamertag: "UN", TimeMS: 4_000},
		{XUID: 222, Gamertag: "DEUX", TimeMS: 8_000},
		{XUID: 111, Gamertag: "UN", TimeMS: 20_000},
		{XUID: 111, Gamertag: "UN", TimeMS: 34_000},
		{XUID: 222, Gamertag: "DEUX", TimeMS: 36_000},
	}
	return IdentityInput{
		Positions: pos, Deaths: deaths,
		PlayerIndices: PlayerIndexTable{ByXUID: map[uint64]int{111: 0, 222: 1}, Readings: 26},
		RosterXUIDs:   []uint64{111, 222},
		Clock:         IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 360},
		MatchID:       "test-exclusion",
	}
}

// vieDuSlot rend la premiere vie du slot demande, ou nil.
func vieDuSlot(reg IdentityRegistry, slot uint32) *lifeSpan {
	for i, l := range reg.Vies() {
		if l.slot == slot {
			return &reg.Vies()[i]
		}
	}
	return nil
}

// TestExclusionNommeLaVieDuSeulJoueurLibre : la vie du slot 400 revient a 222, seul joueur
// qu'aucune vie nommee ne place ailleurs pendant son intervalle.
//
// MUTATION : retirer l'appel a `resolveByTemporalExclusion` -> la vie reste anonyme, rouge.
func TestExclusionNommeLaVieDuSeulJoueurLibre(t *testing.T) {
	reg := BuildIdentityRegistry(filmExclusion())
	v := vieDuSlot(reg, 400)
	if v == nil {
		t.Fatal("le slot 400 n'a aucune vie : le film synthetique ne joue pas le cas")
	}
	if v.xuid != 222 {
		t.Fatalf("vie du slot 400 : xuid = %d, attendu 222", v.xuid)
	}
	if v.nomPar != NomParExclusionTemporelle {
		t.Fatalf("vie du slot 400 : nomPar = %q, attendu %q", v.nomPar, NomParExclusionTemporelle)
	}
	if v.cause == CauseVieMort {
		t.Fatal("l'exclusion a fabrique une MORT — elle ajoute une presence, jamais une fin")
	}
}

// TestExclusionMarqueLaVieDeduiteEtSaProvenance : une vie posee par exclusion est DEDUITE (les
// lecteurs qui prouvent une absence doivent s'en abstenir) et publie sa voie canonique.
//
// MUTATION : ne plus marquer `deducedLives` -> le gate de presence des portages eteint son
// abstention, rouge.
func TestExclusionMarqueLaVieDeduiteEtSaProvenance(t *testing.T) {
	reg := BuildIdentityRegistry(filmExclusion())
	deduites := 0
	for i, l := range reg.Vies() {
		if l.slot == 400 && reg.VieDeduite(i) {
			deduites++
		}
	}
	if deduites != 1 {
		t.Fatalf("vies deduites du slot 400 = %d, attendu 1", deduites)
	}
	trouve := false
	for _, b := range reg.Section.BipedSlots {
		if b.Slot != 400 {
			continue
		}
		trouve = true
		if b.Link.Source != canonical.LinkInferred {
			t.Fatalf("provenance = %q, attendu %q", b.Link.Source, canonical.LinkInferred)
		}
		if b.Link.Method != canonical.MethodTemporalExclusion {
			t.Fatalf("voie = %q, attendu %q", b.Link.Method, canonical.MethodTemporalExclusion)
		}
	}
	if !trouve {
		t.Fatal("le slot 400 n'est pas publie dans la section identity")
	}
}

// TestExclusionSeTaitADeuxCandidats : un troisieme joueur libre sur le meme intervalle, et
// l'unicite disparait — la vie reste anonyme.
//
// MUTATION : prendre « le premier » candidat -> une identite choisie par l'ordre, rouge.
func TestExclusionSeTaitADeuxCandidats(t *testing.T) {
	in := filmExclusion()
	// 333 joue sur le slot 300, en dehors de l'intervalle [16..24] : il est LIBRE lui aussi.
	in.Positions = sejour(in.Positions, 300, 1, 6)
	in.Positions = sejour(in.Positions, 300, 28, 38)
	in.Deaths = append(in.Deaths,
		Death{XUID: 333, Gamertag: "TROIS", TimeMS: 6_000},
		Death{XUID: 333, Gamertag: "TROIS", TimeMS: 38_000})
	in.PlayerIndices.ByXUID[333] = 2
	in.RosterXUIDs = []uint64{111, 222, 333}
	reg := BuildIdentityRegistry(in)
	if v := vieDuSlot(reg, 400); v != nil && v.xuid != 0 {
		t.Fatalf("un candidat a ete choisi malgre l'ambiguite : xuid %d", v.xuid)
	}
	if reg.Section.Coverage.BipedSlot.Unresolved == 0 {
		t.Fatal("aucun lien non resolu publie alors que le slot 400 reste anonyme")
	}
}

// TestExclusionSeTaitSansRosterDeLaFeuille : sans roster de la base, l'univers des candidats se
// reduit aux joueurs que le fil des morts nomme — un joueur absent de l'univers ferait passer un
// candidat FAUX pour unique. La regle ne tourne pas.
//
// MUTATION : retirer le garde-fou -> une identite apparait hors ligne, rouge.
func TestExclusionSeTaitSansRosterDeLaFeuille(t *testing.T) {
	in := filmExclusion()
	in.RosterXUIDs = nil
	reg := BuildIdentityRegistry(in)
	if v := vieDuSlot(reg, 400); v != nil && v.xuid != 0 {
		t.Fatalf("une identite est apparue sans roster de la feuille : xuid %d", v.xuid)
	}
}

// TestExclusionSeTaitQuandUnBotEstDeclare : un bot occupe un slot de bipede sans porter de xuid,
// donc sa vie est « anonyme » au sens du registre — l'exclusion lui attribuerait un humain.
//
// MUTATION : retirer le garde-fou -> la vie d'un bot prend le nom d'un humain, rouge.
func TestExclusionSeTaitQuandUnBotEstDeclare(t *testing.T) {
	in := filmExclusion()
	in.Bots = []BotIdentity{{FilmIndex: 5, BotID: 7, Name: "343 Bot"}}
	reg := BuildIdentityRegistry(in)
	if v := vieDuSlot(reg, 400); v != nil && v.xuid != 0 {
		t.Fatalf("une identite humaine est posee alors qu'un bot est declare : xuid %d", v.xuid)
	}
}

// TestExclusionSeTaitQuandLOccupationDepasseLeRoster : plus de vies simultanees que de joueurs,
// c'est que la decoupe ou le roster est faux — la premisse « un joueur, un slot » ne tient plus.
//
// MUTATION : retirer le garde-fou -> la regle raisonne sur une lecture qu'elle sait fausse, rouge.
func TestExclusionSeTaitQuandLOccupationDepasseLeRoster(t *testing.T) {
	in := filmExclusion()
	// Deux corps de plus a [30..33], ou 111 et 222 sont deja tous les deux presents : quatre
	// vies simultanees pour deux joueurs au roster. Le film ne peut pas dire vrai.
	//
	// ILS SONT LOIN DE LA VIE DU SLOT 400 ([16..24]), a dessein : sans le garde-fou, cette
	// vie-la reste uniquement attribuable a 222 et serait nommee. C'est ce que le test pince.
	in.Positions = sejour(in.Positions, 500, 30, 33)
	in.Positions = sejour(in.Positions, 600, 30, 33)
	reg := BuildIdentityRegistry(in)
	for _, l := range reg.Vies() {
		if l.nomPar == NomParExclusionTemporelle {
			t.Fatalf("l'exclusion a tourne malgre 4 vies simultanees pour 2 joueurs (slot %d)",
				l.slot)
		}
	}
}

// TestExclusionSeTaitSansAucunCandidat : quand les huit — ici les deux — joueurs sont places
// ailleurs pendant l'intervalle, la lecture se CONTREDIT. On ne choisit pas, on compte.
// (Mesure du parc : `d9781168` slot 641, zero candidat.)
//
// MUTATION : traiter « zero candidat » comme « un candidat » -> un nom sorti de nulle part, rouge.
func TestExclusionSeTaitSansAucunCandidat(t *testing.T) {
	var pos []filmdec.BipedPosition
	pos = sejour(pos, 100, 10, 20)
	pos = sejour(pos, 200, 22, 30)
	pos = sejour(pos, 300, 1, 17)
	pos = sejour(pos, 400, 16, 24)
	in := IdentityInput{
		Positions: pos,
		Deaths: []Death{
			{XUID: 333, Gamertag: "TROIS", TimeMS: 17_000},
			{XUID: 111, Gamertag: "UN", TimeMS: 20_000},
			{XUID: 222, Gamertag: "DEUX", TimeMS: 30_000},
		},
		PlayerIndices: PlayerIndexTable{
			ByXUID: map[uint64]int{111: 0, 222: 1, 333: 2}, Readings: 26},
		RosterXUIDs: []uint64{111, 222, 333},
		Clock:       IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 300},
		MatchID:     "test-contradiction",
	}
	reg := BuildIdentityRegistry(in)
	if v := vieDuSlot(reg, 400); v != nil && v.xuid != 0 {
		t.Fatalf("une identite est posee sans aucun candidat libre : xuid %d", v.xuid)
	}
	if reg.excludedContradictions == 0 {
		t.Fatal("la contradiction n'est pas comptee : elle serait invisible au journal")
	}
}

// TestExclusionSeTaitSurDeuxViesQuiSeDisputentLeMemeJoueur : deux vies forcees sur le MEME joueur
// et qui se chevauchent ne peuvent pas etre toutes deux siennes, et rien ne dit laquelle l'est.
//
// MUTATION : retirer `conflitDExclusion` -> le meme joueur occupe deux corps au meme instant,
// rouge.
func TestExclusionSeTaitSurDeuxViesQuiSeDisputentLeMemeJoueur(t *testing.T) {
	var pos []filmdec.BipedPosition
	pos = sejour(pos, 100, 10, 12)
	pos = sejour(pos, 100, 24, 25)
	pos = sejour(pos, 300, 18, 20)
	pos = sejour(pos, 200, 1, 5)
	pos = sejour(pos, 200, 30, 32)
	pos = sejour(pos, 400, 10, 20)
	pos = sejour(pos, 500, 15, 25)
	in := IdentityInput{
		Positions: pos,
		Deaths: []Death{
			{XUID: 222, Gamertag: "DEUX", TimeMS: 5_000},
			{XUID: 111, Gamertag: "UN", TimeMS: 12_000},
			{XUID: 333, Gamertag: "TROIS", TimeMS: 20_000},
			{XUID: 111, Gamertag: "UN", TimeMS: 25_000},
			{XUID: 222, Gamertag: "DEUX", TimeMS: 32_000},
		},
		PlayerIndices: PlayerIndexTable{
			ByXUID: map[uint64]int{111: 0, 222: 1, 333: 2}, Readings: 26},
		RosterXUIDs: []uint64{111, 222, 333},
		Clock:       IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 320},
		MatchID:     "test-conflit",
	}
	reg := BuildIdentityRegistry(in)
	for _, l := range reg.Vies() {
		if (l.slot == 400 || l.slot == 500) && l.xuid != 0 {
			t.Fatalf("slot %d nomme %d malgre le conflit : le meme joueur occuperait deux corps",
				l.slot, l.xuid)
		}
	}
}

// TestExclusionRendAmbiguUnSlotQueDeuxJoueursSePartagent : quand la vie deduite tombe sur un slot
// que le pont attribue DEJA a quelqu'un d'autre, le pont APLATI se tait — il ne retient qu'un
// occupant par slot, et le servir publierait un nom arbitraire. La VIE, elle, garde son identite :
// elle est bornee dans le temps, elle ne ment pas. (Cas reel : `d9781168` slot 637.)
//
// MUTATION : ecraser `SlotXUID` au lieu de marquer le slot ambigu -> `PontDeSlot` sert un nom
// arbitraire aux lecteurs du pont aplati, rouge.
func TestExclusionRendAmbiguUnSlotQueDeuxJoueursSePartagent(t *testing.T) {
	var pos []filmdec.BipedPosition
	pos = sejour(pos, 100, 1, 4)
	pos = sejour(pos, 100, 10, 20)
	pos = sejour(pos, 200, 1, 8)
	pos = sejour(pos, 200, 30, 36)
	pos = sejour(pos, 400, 16, 24) // anonyme : seul 222 est libre
	pos = sejour(pos, 400, 30, 38) // nommee par la mort de 111 — plus de 5 s apres, donc une AUTRE vie
	in := IdentityInput{
		Positions: pos,
		Deaths: []Death{
			{XUID: 111, Gamertag: "UN", TimeMS: 4_000},
			{XUID: 222, Gamertag: "DEUX", TimeMS: 8_000},
			{XUID: 111, Gamertag: "UN", TimeMS: 20_000},
			{XUID: 222, Gamertag: "DEUX", TimeMS: 36_000},
			{XUID: 111, Gamertag: "UN", TimeMS: 38_000},
		},
		PlayerIndices: PlayerIndexTable{ByXUID: map[uint64]int{111: 0, 222: 1}, Readings: 26},
		RosterXUIDs:   []uint64{111, 222},
		Clock:         IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 380},
		MatchID:       "test-slot-partage",
	}
	reg := BuildIdentityRegistry(in)
	var deduite bool
	for _, l := range reg.Vies() {
		if l.slot == 400 && l.nomPar == NomParExclusionTemporelle && l.xuid == 222 {
			deduite = true
		}
	}
	if !deduite {
		t.Fatal("la vie anonyme du slot 400 n'a pas ete nommee 222 : le cas n'est pas joue")
	}
	if !reg.SlotsAmbigus()[400] {
		t.Fatal("le slot 400 porte deux joueurs et n'est pas marque ambigu")
	}
	if got := reg.PontDeSlot(400); got != "" {
		t.Fatalf("le pont aplati sert %q sur un slot partage — un nom choisi par l'ordre", got)
	}
}

// TestOccupationMaximaleCompteLesViesSimultanees : la garde du roster repose sur ce balayage.
func TestOccupationMaximaleCompteLesViesSimultanees(t *testing.T) {
	// Trois vies se recouvrent en [80..100] ; la quatrieme est seule.
	lives := []lifeSpan{
		{slot: 1, from: 0, to: 100},
		{slot: 2, from: 50, to: 150},
		{slot: 3, from: 80, to: 200},
		{slot: 4, from: 300, to: 400},
	}
	if got := occupationMaximale(lives); got != 3 {
		t.Fatalf("occupation maximale = %d, attendu 3", got)
	}
	// LES BORNES SONT INCLUSIVES, ET UNE FIN A L'INSTANT D'UN DEBUT NE COMPTE PAS DEUX FOIS :
	// une vie [0..100] et une vie [101..200] ne sont jamais simultanees.
	if got := occupationMaximale(lives[:1]); got != 1 {
		t.Fatalf("occupation maximale d'une seule vie = %d, attendu 1", got)
	}
	if got := occupationMaximale([]lifeSpan{{from: 0, to: 100}, {from: 101, to: 200}}); got != 1 {
		t.Fatalf("occupation maximale de deux vies disjointes = %d, attendu 1", got)
	}
	if got := occupationMaximale(nil); got != 0 {
		t.Fatalf("occupation maximale d'un film vide = %d, attendu 0", got)
	}
}
