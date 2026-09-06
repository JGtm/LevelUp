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

// TestFermetureBNommeLaVieDesigneeQuandLeSlotEnPorteDeux — LA RÉGRESSION DU BALAYAGE DU PARC,
// FIGÉE AVEC SA PROVENANCE (instruction des régressions, candidate 4, 2026-09-06).
//
// LE CAS RÉEL. Match `145908d1` (BTB:CTF, Breaker), slot 562 : deux vies anonymes — [1439..1452]
// et [1578..2080] en images de 100 ms —, le pont attribue le slot par la fermeture B, et le
// document publiait 17 TIRS de ce slot sur une piste SANS NOM. Le nommage cherchait « l'unique
// vie anonyme du slot » et s'abstenait dès qu'il y en avait deux ; la fermeture, elle, avait
// désigné la première. Mesure de bout en bout sur ce film : 53 slots au pont, 53 pistes nommées
// au schéma 35, 51 aux schémas 36 à 40, 53 de nouveau avec ce correctif — et 24 identités
// distinctes sur les pistes, contre 23 pendant la régression.
//
// CE QUE LE TEST VERROUILLE : la vie DÉSIGNÉE est nommée, l'AUTRE reste anonyme (nommer les deux
// serait l'héritage de slot que le schéma 36 a précisément supprimé).
func TestFermetureBNommeLaVieDesigneeQuandLeSlotEnPorteDeux(t *testing.T) {
	// Une vie nommée (slot 1) calibre la fenêtre de réapparition à 8 000 ms. Le slot 2 porte
	// DEUX vies anonymes : la première commence 8 s après la mort de 222, la seconde 10 s plus
	// tard — aucune mort ne tombe dans SA fenêtre, elle ne revendique donc personne.
	tracks := tracksOf(at(1, 9_000_000), at(2, 20_000_000), at(2, 30_000_000))
	lives := []lifeSpan{
		{slot: 1, from: 9_000_000, to: 10_000_000, xuid: 111},
		{slot: 2, from: 20_000_000, to: 21_000_000},
		{slot: 2, from: 30_000_000, to: 31_000_000},
	}
	deaths := []Death{{XUID: 111, TimeMS: 1_000}, {XUID: 222, TimeMS: 12_000}}
	owner := map[uint32]int{1: 0}
	var rep closureReport
	closeByRespawn(tracks, owner, lives, deaths, 0, map[uint64]int{111: 0, 222: 1}, &rep)
	if owner[2] != 1 || rep.byRespawn != 1 {
		t.Fatalf("la fermeture B devait attribuer le slot 2 au joueur 1, pont = %v, rapport %+v",
			owner, rep)
	}
	if got, ok := rep.closedLife[2]; !ok || got != 1 {
		t.Fatalf("la fermeture devait DÉSIGNER la vie d'indice 1 (celle qui suit la mort), "+
			"obtenu %v (présent : %v)", got, ok)
	}
	nameClosedLives(lives, owner, rep.closedLife, map[uint64]int{111: 0, 222: 1})
	if lives[1].xuid != 222 {
		t.Fatalf("la vie désignée devait porter le xuid 222, obtenu %d", lives[1].xuid)
	}
	if lives[2].xuid != 0 {
		t.Fatalf("la SECONDE vie du même slot n'est pas désignée : elle doit rester anonyme, "+
			"obtenu %d", lives[2].xuid)
	}
	// Bout en bout : la piste de la vie désignée porte le nom, l'autre non.
	trs := []Track{{Slot: 2, StartFrame: 200, EndFrame: 210}, {Slot: 2, StartFrame: 300, EndFrame: 310}}
	nameTracksByLives(trs, lives, 0, 100_000)
	if trs[0].XUID != "222" || trs[1].XUID != "" {
		t.Fatalf("la piste de la vie désignée devait être nommée et l'autre rester anonyme, "+
			"obtenu %q puis %q", trs[0].XUID, trs[1].XUID)
	}
}

// TestFermetureANeTranchePasEntreDeuxViesDuMemeSlot — LA GARDE D'AMBIGUÏTÉ, ET ELLE EST TESTÉE.
//
// Le pont ne retient qu'UN propriétaire par slot ; quand deux vies distinctes du même slot ont
// été désignées (deux tirs, deux instants, un candidat unique à chaque fois), rien ne dit
// LAQUELLE est la sienne. `closedLife` vaut alors -1 et AUCUNE vie n'est nommée — l'attribution
// du slot au pont, elle, reste acquise.
//
// Sans cette garde, la dernière désignation écrite gagnerait, c'est-à-dire l'ORDRE DES TIRS :
// exactement le genre d'arbitrage par l'ordre que les fermetures refusent partout ailleurs.
func TestFermetureANeTranchePasEntreDeuxViesDuMemeSlot(t *testing.T) {
	// Le joueur 3 a un corps connu (slot 1) qui s'achève à 0,6 s — l'ancrage exigé par la
	// corroboration. Le slot 2 porte deux vies libres, séparées par plus de lifeGapUS ; le
	// joueur 3 tire une fois dans chacune, et il est seul candidat aux deux instants.
	tracks := tracksOf(at(1, 400_000), at(1, 600_000),
		at(2, 900_000), at(2, 1_100_000), at(2, 20_000_000), at(2, 20_200_000))
	owner := map[uint32]int{1: 3}
	lives := []lifeSpan{
		{slot: 1, from: 400_000, to: 600_000, xuid: 111},
		{slot: 2, from: 900_000, to: 1_100_000},
		{slot: 2, from: 20_000_000, to: 20_200_000},
	}
	var rep closureReport
	closeByAvailableBody(tracks, owner, lives, []FireEventRef{
		{FilmIndex: 3, TimestampUS: 1_000_000},
		{FilmIndex: 3, TimestampUS: 20_100_000},
	}, &rep)
	if owner[2] != 3 || rep.byShot != 1 {
		t.Fatalf("le slot 2 devait revenir au joueur 3, pont = %v, rapport %+v", owner, rep)
	}
	if got := rep.closedLife[2]; got != -1 {
		t.Fatalf("deux vies désignées pour un même slot : la désignation doit valoir -1, "+
			"obtenu %d", got)
	}
	nameClosedLives(lives, owner, rep.closedLife, map[uint64]int{111: 0, 222: 3})
	if lives[1].xuid != 0 || lives[2].xuid != 0 {
		t.Fatalf("aucune des deux vies ne doit être nommée, obtenu %d et %d",
			lives[1].xuid, lives[2].xuid)
	}
}

// TestNameClosedLivesNeReecritJamaisUneVieLue — la lecture prime sur la déduction.
//
// Une vie que le fil des morts a nommée garde SON identité même si une fermeture a désigné le
// même indice : le pont est fait de lectures d'abord, de déductions ensuite (cf. owners.go).
func TestNameClosedLivesNeReecritJamaisUneVieLue(t *testing.T) {
	lives := []lifeSpan{{slot: 2, from: 0, to: 1_000_000, xuid: 111}}
	nameClosedLives(lives, map[uint32]int{2: 1}, map[uint32]int{2: 0}, map[uint64]int{111: 0, 222: 1})
	if lives[0].xuid != 111 {
		t.Fatalf("la vie lue devait garder son xuid 111, obtenu %d", lives[0].xuid)
	}
}

// TestFermetureBNeTranchePasQuandDeuxMortsDesignentLeMemeSlot — la garde d'ambiguïté du
// RAPPORT (`closureReport.noteLife`), qui double celle du dépouillement des tirs sur l'autre
// chemin : deux vies du même slot, chacune seule à revendiquer SA victime.
//
// Le pont n'a qu'un propriétaire par slot ; ici deux joueurs y sont écrits l'un après l'autre.
// Rien ne permet de dire à laquelle des deux vies appartient le nom retenu : `closedLife` vaut
// -1, et aucune vie n'est nommée. (Que le pont lui-même se laisse écraser dans ce cas est une
// faiblesse ANTÉRIEURE, notée au registre des reports du 2026-09-06 et NON traitée ici : ce
// correctif ne touche à aucune décision du pont.)
func TestFermetureBNeTranchePasQuandDeuxMortsDesignentLeMemeSlot(t *testing.T) {
	tracks := tracksOf(at(1, 9_000_000), at(2, 20_000_000), at(2, 30_000_000))
	lives := []lifeSpan{
		{slot: 1, from: 9_000_000, to: 10_000_000, xuid: 111},
		{slot: 2, from: 20_000_000, to: 21_000_000},
		{slot: 2, from: 30_000_000, to: 31_000_000},
	}
	// Fenêtre calibrée à 8 000 ms par la vie nommée ; 222 puis 333 meurent 8 s avant chacune
	// des deux vies libres du slot 2.
	deaths := []Death{{XUID: 111, TimeMS: 1_000}, {XUID: 222, TimeMS: 12_000}, {XUID: 333, TimeMS: 22_000}}
	owner := map[uint32]int{1: 0}
	var rep closureReport
	closeByRespawn(tracks, owner, lives, deaths, 0, map[uint64]int{111: 0, 222: 1, 333: 2}, &rep)
	if rep.byRespawn != 2 {
		t.Fatalf("les deux vies devaient être attribuées au pont (pré-requis du test), rapport %+v", rep)
	}
	if got := rep.closedLife[2]; got != -1 {
		t.Fatalf("deux vies désignées pour un même slot : la désignation doit valoir -1, obtenu %d", got)
	}
	nameClosedLives(lives, owner, rep.closedLife, map[uint64]int{111: 0, 222: 1, 333: 2})
	if lives[1].xuid != 0 || lives[2].xuid != 0 {
		t.Fatalf("aucune des deux vies ne doit être nommée, obtenu %d et %d", lives[1].xuid, lives[2].xuid)
	}
}
