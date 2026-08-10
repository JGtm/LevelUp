package replay

// closures_test.go — LES FERMETURES, ET SURTOUT CE QU'ELLES REFUSENT.
//
// L'ESSENTIEL DE CE FICHIER TESTE DES ABSTENTIONS. Une fermeture qui attribue est banale ; ce
// qui la distingue du vote retiré le 2026-07-28, c'est qu'elle se TAIT dès que deux candidats
// subsistent, ET qu'elle n'attribue rien que le film ne corrobore. Un test qui ne vérifierait que
// les succès laisserait passer un retour au vote sans que rien ne casse.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// at construit un échantillon de position au seul instant qui nous intéresse. Les helpers
// `posAt` (shots_test.go) et `tracksOf` (lives_test.go) sont RÉUTILISÉS tels quels : une
// troisième copie divergerait, et la règle du dépôt l'interdit.
func at(slot uint32, tUS uint64) filmdec.BipedPosition { return posAt(slot, tUS, 0, 0, 0) }

// TestFermetureAAttribueLeCorpsQuiProlongeLeTireur : le cas nominal de la fermeture A, ET SA
// SÉMANTIQUE EXACTE.
//
// CE TEST A ÉTÉ REPENSÉ LE 2026-08-11, parce que sa version d'origine ENCODAIT UN DÉFAUT. Elle
// faisait tirer un joueur QUE RIEN NE SITUAIT DANS LE FILM — aucune vie, aucun corps connu — et
// exigeait que le corps anonyme unique lui revienne. Elle verrouillait donc l'attribution d'un
// corps sans la moindre corroboration : « un seul corps libre est là » y valait « ce corps est le
// sien », ce qui ne s'ensuit pas.
//
// LA SÉMANTIQUE JUSTE, ET C'EST CE QUE LA VERSION CI-DESSOUS VÉRIFIE : le corps déduit doit
// PROLONGER le tireur. Le joueur 3 a un corps connu (slot 1) qui s'achève à 0,6 s ; le corps
// anonyme (slot 2) commence après, et il est seul à couvrir l'instant du tir. Il est donc la
// suite possible de sa vie — la vie qu'aucune mort ne termine, celle où les tirs se perdent.
func TestFermetureAAttribueLeCorpsQuiProlongeLeTireur(t *testing.T) {
	// Le tir tombe à 1,0 s, où le slot 2 n'a AUCUN échantillon : l'unicité se juge sur
	// l'intervalle de vie, jamais sur la position répliquée (cf. closures.go).
	tracks := tracksOf(at(1, 400_000), at(1, 600_000), at(2, 900_000), at(2, 1_100_000))
	owner := map[uint32]int{1: 3}
	lives := []lifeSpan{{slot: 1, from: 400_000, to: 600_000, xuid: 111},
		{slot: 2, from: 900_000, to: 1_100_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{{FilmIndex: 3, TimestampUS: 1_000_000}}, &rep)
	if owner[2] != 3 {
		t.Fatalf("le slot 2 prolonge le joueur 3 et devait lui revenir, pont = %v", owner)
	}
	if rep.byShot != 1 || rep.contested != 0 || rep.refused != 0 {
		t.Fatalf("compte rendu inattendu : %+v", rep)
	}
}

// TestFermetureARefuseUnCorpsQueLeTireurNePeutProlonger — LE DUAL DU DÉFAUT DU 2026-08-09, ET IL
// ÉTAIT RESTÉ OUVERT.
//
// La ronde précédente a fermé le cas « le corps du tireur couvre l'instant ET un autre aussi » :
// deux candidats, abstention. Le DUAL est celui-ci : le tireur n'a AUCUN corps qui couvre
// l'instant, un seul corps ÉTRANGER le couvre, et il lui était attribué — sans que rien ne
// corrobore qu'il soit le sien, et sans qu'aucun compteur ne s'en émeuve.
//
// ICI LE FILM RÉFUTE L'ATTRIBUTION, ET C'EST MESURABLE : le tireur REPREND un corps à 5-6 s,
// APRÈS le corps candidat. Le candidat serait donc une vie INTERMÉDIAIRE de sa vie, donc une vie
// terminée par sa mort — mort qui l'aurait NOMMÉE. Elle ne l'a pas été : ce corps est celui d'un
// autre. Le contrôle de recouvrement ne pouvait pas le voir, les deux traces étant disjointes.
func TestFermetureARefuseUnCorpsQueLeTireurNePeutProlonger(t *testing.T) {
	tracks := tracksOf(at(2, 900_000), at(2, 1_100_000), at(3, 5_000_000), at(3, 6_000_000))
	owner := map[uint32]int{3: 3}
	lives := []lifeSpan{{slot: 2, from: 900_000, to: 1_100_000},
		{slot: 3, from: 5_000_000, to: 6_000_000, xuid: 333}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{{FilmIndex: 3, TimestampUS: 1_000_000}}, &rep)
	if _, ok := owner[2]; ok {
		t.Fatalf("le tireur reprend un corps APRÈS celui-ci : l'attribution devait être rejetée, "+
			"pont = %v", owner)
	}
	if rep.byShot != 0 || rep.refused != 1 {
		t.Fatalf("un rejet faute de corroboration attendu, compte rendu %+v", rep)
	}
}

// TestFermetureARefuseUnCorpsQuandLeTireurNEstAncreNullePart — L'AUTRE MOITIÉ DU DUAL.
//
// Le tireur n'a AUCUN corps connu : rien ne le situe dans le film. L'unicité du candidat ne dit
// alors qu'une chose — un seul corps libre couvre cet instant — et rien du tout sur son
// appartenance. Un corps attribué là est un corps attribué au hasard de ce qui restait libre.
func TestFermetureARefuseUnCorpsQuandLeTireurNEstAncreNullePart(t *testing.T) {
	tracks := tracksOf(at(1, 400_000), at(1, 600_000), at(2, 900_000), at(2, 1_100_000))
	owner := map[uint32]int{1: 0} // le joueur 3 n'apparaît nulle part au pont
	lives := []lifeSpan{{slot: 1, from: 400_000, to: 600_000, xuid: 111},
		{slot: 2, from: 900_000, to: 1_100_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{{FilmIndex: 3, TimestampUS: 1_000_000}}, &rep)
	if _, ok := owner[2]; ok {
		t.Fatalf("le tireur n'est ancré nulle part : rien ne devait lui être attribué, pont = %v", owner)
	}
	if rep.byShot != 0 || rep.refused != 1 {
		t.Fatalf("un rejet faute d'ancrage attendu, compte rendu %+v", rep)
	}
}

// TestFermetureASAbstientQuandDeuxCorpsRevendiquentLeMemeTireur — UN JOUEUR N'A QU'UN DERNIER
// CORPS.
//
// C'est le symétrique du refus « deux joueurs revendiquent le même corps », et il devient
// nécessaire AVEC la corroboration : la première attribution fait du corps déduit un corps connu
// du tireur, ce qui fait tomber la seconde — ou pas, selon que son numéro de slot est plus petit
// ou plus grand. L'ORDRE DES SLOTS déciderait alors laquelle des deux passe, exactement le défaut
// corrigé à la fermeture B le 2026-08-09. Deux déductions qui s'excluent tombent toutes les deux.
func TestFermetureASAbstientQuandDeuxCorpsRevendiquentLeMemeTireur(t *testing.T) {
	tracks := tracksOf(at(1, 200_000), at(1, 300_000),
		at(2, 900_000), at(2, 1_100_000), at(4, 2_900_000), at(4, 3_100_000))
	owner := map[uint32]int{1: 3}
	lives := []lifeSpan{{slot: 1, from: 200_000, to: 300_000, xuid: 111},
		{slot: 2, from: 900_000, to: 1_100_000},
		{slot: 4, from: 2_900_000, to: 3_100_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{
		{FilmIndex: 3, TimestampUS: 1_000_000},
		{FilmIndex: 3, TimestampUS: 3_000_000},
	}, &rep)
	if len(owner) != 1 {
		t.Fatalf("un joueur n'a qu'un dernier corps : aucune des deux déductions ne devait "+
			"passer, pont = %v", owner)
	}
	if rep.byShot != 0 || rep.contested != 2 {
		t.Fatalf("les deux déductions exclusives devaient être comptées, compte rendu %+v", rep)
	}
}

// TestFermetureASAbstientQuandDeuxCorpsSontPossibles : L'ABSTENTION, ET C'EST LE TEST QUI COMPTE.
func TestFermetureASAbstientQuandDeuxCorpsSontPossibles(t *testing.T) {
	tracks := tracksOf(at(2, 1_000_000), at(3, 1_000_000))
	owner := map[uint32]int{}
	lives := []lifeSpan{{slot: 2, from: 900_000, to: 1_100_000}, {slot: 3, from: 900_000, to: 1_100_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{{FilmIndex: 3, TimestampUS: 1_000_000}}, &rep)
	if len(owner) != 0 {
		t.Fatalf("deux corps possibles : rien ne devait être attribué, pont = %v", owner)
	}
	if rep.byShot != 0 {
		t.Fatalf("aucune attribution attendue, compte rendu %+v", rep)
	}
}

// TestFermetureASAbstientQuandUnCandidatEstDansUnTrouDeReplication — LE SCÉNARIO QUI FAISAIT
// ATTRIBUER LE CORPS D'AUTRUI, ET IL N'AVAIT AUCUN TEST.
//
// Deux vies anonymes couvrent l'instant du tir. Celle du VRAI tireur traverse un trou de
// réplication à cet instant ; l'autre y est échantillonnée. Juger la candidature sur
// l'ÉCHANTILLON rendait la première invisible : la seconde passait pour l'unique candidate, et
// son corps — celui d'un autre joueur — était attribué au tireur sans qu'aucun garde-fou ne
// puisse s'en apercevoir, puisque du point de vue du code il n'y avait qu'un candidat.
//
// LES DEUX CONSTANTES QUI RENDENT LE SCÉNARIO POSSIBLE SONT DANS LE CODE, pas inventées ici :
// une vie survit à un trou de `lifeGapUS` (5 s), le rattachement exige un échantillon à
// `shotPosToleranceUS` (120 ms). Entre les deux, il y a 4,88 s de fenêtre pour se tromper.
func TestFermetureASAbstientQuandUnCandidatEstDansUnTrouDeReplication(t *testing.T) {
	// slot 2 : échantillonné À l'instant du tir. slot 3 : échantillons à 0,5 s et 3,0 s — un trou
	// de 2,5 s qui contient le tir, sous lifeGapUS, donc UNE SEULE vie qui couvre l'instant.
	tracks := tracksOf(at(2, 1_000_000), at(3, 500_000), at(3, 3_000_000))
	owner := map[uint32]int{}
	lives := []lifeSpan{{slot: 2, from: 900_000, to: 1_100_000},
		{slot: 3, from: 500_000, to: 3_000_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{{FilmIndex: 3, TimestampUS: 1_000_000}}, &rep)
	if len(owner) != 0 {
		t.Fatalf("un corps en trou de réplication reste un corps possible : rien ne devait être "+
			"attribué, pont = %v", owner)
	}
	if rep.byShot != 0 {
		t.Fatalf("aucune attribution attendue, compte rendu %+v", rep)
	}
	if rep.contested != 2 {
		t.Fatalf("les deux vies écartées faute d'unicité devaient être comptées, compte rendu %+v", rep)
	}
}

// TestFermetureARefuseQuandDeuxJoueursRevendiquentLeMemeCorps.
func TestFermetureARefuseQuandDeuxJoueursRevendiquentLeMemeCorps(t *testing.T) {
	tracks := tracksOf(at(2, 1_000_000), at(2, 1_200_000))
	owner := map[uint32]int{}
	lives := []lifeSpan{{slot: 2, from: 900_000, to: 1_300_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{
		{FilmIndex: 3, TimestampUS: 1_000_000},
		{FilmIndex: 5, TimestampUS: 1_200_000},
	}, &rep)
	if len(owner) != 0 {
		t.Fatalf("corps contesté : rien ne devait être attribué, pont = %v", owner)
	}
	if rep.contested != 1 {
		t.Fatalf("une contestation attendue, compte rendu %+v", rep)
	}
}

// TestLeControleDeRecouvrementRejette : un joueur n'a qu'un corps.
//
// C'EST LE GARDE-FOU QUI PEUT RÉFUTER LA MÉTHODE, et il doit donc mordre au moins une fois dans
// les tests — sinon rien ne prouve qu'il est branché.
//
// LE RECOUVREMENT EST DEPUIS ABSORBÉ PAR LA CORROBORATION (2026-08-11) : exiger que tous les corps
// connus du tireur s'achèvent AVANT le corps candidat interdit le chevauchement comme cas
// particulier. Le cas garde son test — c'est celui qui échouerait en premier si la corroboration
// était affaiblie en simple test de non-contradiction.
func TestLeControleDeRecouvrementRejette(t *testing.T) {
	// Le joueur 3 possède déjà le slot 1, dont la VIE ENCADRE l'instant du tir sans qu'aucun
	// échantillon n'y soit assez proche (500 ms, au-delà de la tolérance de 120 ms). Le tir est
	// donc orphelin — la fermeture A s'active — mais le corps déduit chevauche dans le temps un
	// corps déjà attribué au même joueur : c'est impossible, et le contrôle doit le dire.
	tracks := tracksOf(at(1, 500_000), at(1, 2_000_000), at(2, 1_000_000))
	owner := map[uint32]int{1: 3}
	lives := []lifeSpan{{slot: 1, from: 500_000, to: 2_000_000, xuid: 111},
		{slot: 2, from: 900_000, to: 1_100_000}}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{{FilmIndex: 3, TimestampUS: 1_000_000}}, &rep)
	if _, ok := owner[2]; ok {
		t.Fatalf("le joueur 3 a déjà un corps à cet instant : l'attribution devait être rejetée, pont = %v", owner)
	}
	if rep.refused != 1 {
		t.Fatalf("un rejet par recouvrement attendu, compte rendu %+v", rep)
	}
}

// TestFermetureBAttribueSurUneSeuleMortDansLaFenetre : le cas nominal de la fermeture B.
func TestFermetureBAttribueSurUneSeuleMortDansLaFenetre(t *testing.T) {
	// Une vie nommée calibre la fenêtre à 8 000 ms ; la vie anonyme commence 8 000 ms après une
	// mort unique, celle du joueur 222.
	tracks := tracksOf(at(1, 9_000_000), at(2, 20_000_000))
	lives := []lifeSpan{
		{slot: 1, from: 9_000_000, to: 10_000_000, xuid: 111},
		{slot: 2, from: 20_000_000, to: 21_000_000},
	}
	deaths := []Death{{XUID: 111, TimeMS: 1_000}, {XUID: 222, TimeMS: 12_000}}
	owner := map[uint32]int{1: 0}
	var rep closureReport
	closeByRespawn(tracks, owner, lives, deaths, 0, map[uint64]int{111: 0, 222: 1}, &rep)
	if owner[2] != 1 {
		t.Fatalf("le slot 2 devait revenir au joueur 1 (xuid 222), pont = %v", owner)
	}
	if rep.byRespawn != 1 {
		t.Fatalf("une fermeture par réapparition attendue, compte rendu %+v", rep)
	}
}

// TestFermetureBSAbstientQuandDeuxMortsSontDansLaFenetre : L'ABSTENTION DE B.
func TestFermetureBSAbstientQuandDeuxMortsSontDansLaFenetre(t *testing.T) {
	tracks := tracksOf(at(1, 9_000_000), at(2, 20_000_000))
	lives := []lifeSpan{
		{slot: 1, from: 9_000_000, to: 10_000_000, xuid: 111},
		{slot: 2, from: 20_000_000, to: 21_000_000},
	}
	// Deux morts distinctes tombent dans la même fenêtre de réapparition.
	deaths := []Death{{XUID: 111, TimeMS: 1_000}, {XUID: 222, TimeMS: 12_000}, {XUID: 333, TimeMS: 12_100}}
	owner := map[uint32]int{1: 0}
	var rep closureReport
	closeByRespawn(tracks, owner, lives, deaths, 0, map[uint64]int{111: 0, 222: 1, 333: 2}, &rep)
	if _, ok := owner[2]; ok {
		t.Fatalf("deux victimes possibles : rien ne devait être attribué, pont = %v", owner)
	}
	if rep.contested != 1 {
		t.Fatalf("une contestation attendue, compte rendu %+v", rep)
	}
}

// TestFermetureBSAbstientQuandDeuxViesRevendiquentLaMemeMort — UNE MORT NE REND QU'UN CORPS.
//
// L'exclusion mutuelle manquait DANS CE SENS-LÀ. Refuser la vie qui voit deux morts était déjà
// fait ; refuser la mort que deux vies revendiquent ne l'était pas. Chacune des deux vies voyait
// alors un candidat unique, et c'est l'ORDRE DE PARCOURS DES SLOTS qui décidait laquelle serait
// nommée en premier — la seconde héritant du même joueur un instant plus tard, faute de
// chevauchement qui la fasse rejeter.
func TestFermetureBSAbstientQuandDeuxViesRevendiquentLaMemeMort(t *testing.T) {
	// Une vie nommée calibre la fenêtre à 8 000 ms. Les deux vies anonymes commencent chacune ~8 s
	// après LA MÊME mort (celle de 222) ; aucune autre mort n'entre dans leur fenêtre. Leurs traces
	// ne se chevauchent pas : le contrôle de recouvrement ne peut pas rattraper le défaut.
	tracks := tracksOf(at(1, 9_000_000), at(2, 20_000_000), at(3, 20_100_000))
	lives := []lifeSpan{
		{slot: 1, from: 9_000_000, to: 10_000_000, xuid: 111},
		{slot: 2, from: 20_000_000, to: 20_000_000},
		{slot: 3, from: 20_100_000, to: 20_100_000},
	}
	deaths := []Death{{XUID: 111, TimeMS: 1_000}, {XUID: 222, TimeMS: 12_000}}
	owner := map[uint32]int{1: 0}
	var rep closureReport
	closeByRespawn(tracks, owner, lives, deaths, 0, map[uint64]int{111: 0, 222: 1}, &rep)
	if len(owner) != 1 {
		t.Fatalf("une mort ne rend qu'un corps : aucune des deux vies ne devait être attribuée, "+
			"pont = %v", owner)
	}
	if rep.byRespawn != 0 {
		t.Fatalf("aucune fermeture attendue, compte rendu %+v", rep)
	}
	if rep.contested != 2 {
		t.Fatalf("les deux déductions abandonnées devaient être comptées, compte rendu %+v", rep)
	}
}

// TestFermetureBCompteLeRejetDUneIdentiteHorsTable : un rejet muet ferait mentir la somme publiée.
//
// La victime est connue du fil des morts, mais la table d'index ne porte pas son identité : on
// sait QUI est mort, pas quel index de film lui répond. L'attribution est impossible — et ce
// refus doit se voir, comme tous les autres.
func TestFermetureBCompteLeRejetDUneIdentiteHorsTable(t *testing.T) {
	tracks := tracksOf(at(1, 9_000_000), at(2, 20_000_000))
	lives := []lifeSpan{
		{slot: 1, from: 9_000_000, to: 10_000_000, xuid: 111},
		{slot: 2, from: 20_000_000, to: 21_000_000},
	}
	deaths := []Death{{XUID: 111, TimeMS: 1_000}, {XUID: 222, TimeMS: 12_000}}
	owner := map[uint32]int{1: 0}
	var rep closureReport
	closeByRespawn(tracks, owner, lives, deaths, 0, map[uint64]int{111: 0}, &rep)
	if _, ok := owner[2]; ok {
		t.Fatalf("sans index de film, rien ne peut être attribué, pont = %v", owner)
	}
	if rep.refused != 1 {
		t.Fatalf("le rejet d'une identité hors table devait être compté, compte rendu %+v", rep)
	}
}

// TestVerdictDuPontRefuseUneSourceNonComptee : la règle de provenance, après les fermetures.
func TestVerdictDuPontRefuseUneSourceNonComptee(t *testing.T) {
	base := BridgeHealth{Slots: 10, FromReading: 8, ClosedByShot: 1, ClosedByRespawn: 1,
		IndexReadings: 26}
	if got := verdictOfBridge(base); got != VerdictNominal {
		t.Fatalf("lecture + fermetures = entrées : verdict attendu %q, obtenu %q", VerdictNominal, got)
	}
	orphelin := base
	orphelin.ClosedByRespawn = 0 // une entrée n'est plus justifiée par aucune source
	if got := verdictOfBridge(orphelin); got == VerdictNominal {
		t.Fatal("une entrée non justifiée doit rendre le pont NON PUBLIABLE")
	}
}
