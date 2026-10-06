package replay

// occupants_equipe_declaree_test.go — L'EQUIPE D'UN BOT QU'AUCUNE ENTITE NE PORTE EST CELLE QUE SON
// ENTREE BOT_METADATA ECRIT (lot « toute entree du roster a l'equipe que le film ecrit »,
// 2026-10-06, `.ai/PLAN_REJEU_EQUIPES_SOURCE_2026-10-06.md`).
//
//	E-DECLAREE    un bot declare entre deux images-cles porteuses n'a aucune entite : son equipe est
//	              celle de sa declaration, et la table par index (l'equipe d'un AUTRE occupant de
//	              son index, `c7f94693` : `343 Donos`) ne la remplace pas ;
//	E-CONTRE      une entite et une declaration qui se contredisent : l'entite est publiee, l'ecart
//	              se compte ;
//	E-MUET        un bot d'un film balaye que ni ses entites ni sa declaration ne nomment reste sans
//	              equipe : aucun emprunt a l'index ;
//	E-HUMAIN      un humain sans entite garde la table de controle (rien ne change pour lui) ;
//	E-NONBALAYE   sans balayage des entites, la declaration passe encore devant l'index.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// botDeclare fabrique l'entree de roster et l'identite d'un bot d'index 8, declare des frames
// `de` a `a` (document a 100 ms), d'equipe ecrite `equipe` (nil : non lue).
func botDeclare(nom string, de, a int, equipe *int) (RosterEntry, BotIdentity) {
	e := RosterEntry{FilmIndex: 8, Name: nom, Bot: true}
	b := BotIdentity{FilmIndex: 8, Name: nom, Team: equipe,
		Declarations: [][2]uint64{{uint64(de) * 100_000, uint64(a) * 100_000}}}
	return e, b
}

func equipePtr(t int) *int { return &t }

func TestEquipeDuBotParSaDeclaration(t *testing.T) { // E-DECLAREE
	// L'index 8 est porte, plus tard, par l'entite d'equipe 1 d'un AUTRE bot : la table par index
	// dit 1 ; la declaration de Donos dit 0.
	scan := scanDeTest([]int{10, 50, 90}, grammar.PlayerEntity{Slot: 7, Index: 8, Team: 1, FirstKF: 2,
		LastKF: 2, Seen: 1})
	donos, idDonos := botDeclare("343 Donos [bot]", 20, 40, equipePtr(0))
	autre, idAutre := botDeclare("343 Byrontron [bot]", 85, 99, equipePtr(1))
	in := entreesDeTest(scan)
	in.bots, in.parIndex = []BotIdentity{idDonos, idAutre}, map[int]int{8: 1}
	occ := lierLesOccupants([]RosterEntry{donos, autre}, nil, in)
	if eq := occ.parEntree[0].equipe; eq == nil || *eq != 0 {
		t.Fatalf("equipe de Donos %v : attendu 0, celle de sa declaration", deref(eq))
	}
	if occ.equipesParDeclaration != 1 || occ.equipesContreDeclaration != 0 {
		t.Fatalf("par declaration %d, contre %d : attendu 1 et 0 (Byrontron a son entite)",
			occ.equipesParDeclaration, occ.equipesContreDeclaration)
	}
}

func TestEntiteContreDeclarationSeCompte(t *testing.T) { // E-CONTRE
	scan := scanDeTest([]int{10, 50, 90}, grammar.PlayerEntity{Slot: 7, Index: 8, Team: 1, FirstKF: 1,
		LastKF: 1, Seen: 1})
	bot, id := botDeclare("343 PardonMy [bot]", 45, 60, equipePtr(0))
	in := entreesDeTest(scan)
	in.bots = []BotIdentity{id}
	occ := lierLesOccupants([]RosterEntry{bot}, nil, in)
	if eq := occ.parEntree[0].equipe; eq == nil || *eq != 1 {
		t.Fatalf("equipe %v : attendu 1, celle de l'entite liee", deref(eq))
	}
	if occ.equipesContreDeclaration != 1 || occ.equipesParDeclaration != 0 {
		t.Fatalf("contre declaration %d, par declaration %d : attendu 1 et 0", occ.equipesContreDeclaration,
			occ.equipesParDeclaration)
	}
}

func TestBotMuetNEmprunteRienALIndex(t *testing.T) { // E-MUET
	scan := scanDeTest([]int{10, 50, 90}, grammar.PlayerEntity{Slot: 3, Index: 2, Team: 0, LastKF: 2, Seen: 3})
	bot, id := botDeclare("343 Sandwolf [bot]", 20, 40, nil)
	in := entreesDeTest(scan)
	in.bots, in.parIndex = []BotIdentity{id}, map[int]int{8: 1}
	occ := lierLesOccupants([]RosterEntry{bot}, nil, in)
	if eq := occ.parEntree[0].equipe; eq != nil {
		t.Fatalf("equipe %d : un bot sans entite ni equipe declaree reste sans equipe, l'index ne la "+
			"prete pas", *eq)
	}
}

func TestHumainSansEntiteGardeLaTableDeControle(t *testing.T) { // E-HUMAIN
	scan := scanDeTest([]int{10, 50, 90}, grammar.PlayerEntity{Slot: 3, Index: 2, Team: 0, LastKF: 2, Seen: 3})
	in := entreesDeTest(scan)
	in.parIndex = map[int]int{9: 1}
	occ := lierLesOccupants([]RosterEntry{entree(2, "120"), entree(9, "190")},
		[]Track{vieDe("120", 5, 95), vieDe("190", 60, 70)}, in)
	if eq := occ.parEntree[1].equipe; eq == nil || *eq != 1 {
		t.Fatalf("equipe de l'humain sans entite %v : attendu 1, la table de controle", deref(eq))
	}
}

func TestSansBalayageLaDeclarationPasseDevantLIndex(t *testing.T) { // E-NONBALAYE
	in := entreesDeTest(grammar.PlayerEntityScan{})
	declare, idDeclare := botDeclare("343 Ritzy [bot]", 20, 40, equipePtr(1))
	in.bots, in.parIndex = []BotIdentity{idDeclare}, map[int]int{8: 0}
	occ := lierLesOccupants([]RosterEntry{declare}, nil, in)
	if eq := occ.parEntree[0].equipe; eq == nil || *eq != 1 {
		t.Fatalf("equipe %v : attendu 1, la declaration avant l'index", deref(eq))
	}
	muet, idMuet := botDeclare("343 Hollis [bot]", 20, 40, nil)
	in.bots = []BotIdentity{idMuet}
	occ = lierLesOccupants([]RosterEntry{muet}, nil, in)
	if eq := occ.parEntree[0].equipe; eq == nil || *eq != 0 {
		t.Fatalf("equipe %v : sans balayage ni declaration, l'index comme avant (0)", deref(eq))
	}
}
