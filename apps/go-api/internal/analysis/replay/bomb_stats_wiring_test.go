package replay

// bomb_stats_wiring_test.go — LE CÂBLAGE DE « ABSENT N'EST PAS ZÉRO », VU DE `BuildFromPositions`
// (revue adversariale de branche, constat I-2).
//
// # LE DÉFAUT QUE CE TEST ATTRAPE, ET POURQUOI AUCUN AUTRE NE LE VOYAIT
//
// `bomb_stats_test.go` et `bomb_arms_test.go` appellent `BuildBombStats` DIRECTEMENT : ils
// prouvent que le noyau applique la règle « une source non lue laisse son champ à `nil` », mais
// ils fournissent eux-mêmes les témoins de lecture. Or ces témoins ne sont pas des entrées de
// l'appelant : ils sont DÉRIVÉS au site de câblage (`attachBombStats`, bomb_stats_document.go) —
//
//	DetonationsRead  = opt.Score != nil
//	CarryRead        = len(own.SlotXUID) > 0
//	ArmingsRead      = bombArmingsRead(doc)   (couverture Scanned ET NON Suppressed)
//
// Inverser l'un des trois COMPILE, et fait sortir des ZÉROS là où le champ devait rester absent —
// « ce joueur n'a rien armé » au lieu de « on n'a pas regardé ». Aucun test du noyau ne le verrait,
// et le gate sur films (`assaut_bomb_arms_gate_test.go`) est sous garde d'environnement : il ne
// tourne pas en CI. C'est la leçon de `build_score_test.go` et de `build_pickups_wiring_test.go` —
// prouver qu'un calque est JUSTE ne prouve pas qu'il est appelé avec les bons arguments.
//
// Ce test passe donc par `BuildFromPositions`, LA fonction de production, sur des entrées
// synthétiques. AUCUNE GARDE D'ENVIRONNEMENT : il tourne en CI.
//
// # DISCRIMINANCE PROUVÉE PAR MUTATION (2026-09-05)
//
//	`bombArmingsRead` rendu constant à `true`   -> cas « anneau NON balayé » et « anneau balayé
//	                                               mais RETENU » rouges (arms = 0 publié) ;
//	`bombArmingsRead` rendu constant à `false`  -> les deux cas « anneau PUBLIÉ » rouges (arms
//	                                               absent), et le test du recalage avec eux ;
//	`CarryRead` inversé (`len(...) == 0`)       -> les cinq cas rouges, des deux côtés de la règle ;
//	recalage remplacé par `0`                   -> `TestBombStatsCablageRecalageHorloge` rouge ;
//	recalage de SIGNE INVERSÉ                   -> le même, rouge.
//
// # LES TROIS ÉTATS DE `Coverage.BombArmings`, ET CELUI QUI N'EXISTE PAS SANS PONT
//
//	ABSENTE (nil)             `opt.Bomb.Scanned` faux : `attachBombArmings` sort avant de rien poser ;
//	SCANNED, NON SUPPRESSED   le calque est publié — la confrontation locale tient ;
//	SUPPRESSED                une explosion du film n'a aucun armement dans la fenêtre de sens :
//	                          le calque ENTIER est retenu à la source (bomb_armings.go).
//
// Le troisième état est INATTEIGNABLE sans pont slot -> xuid, et ce n'est pas un trou du test :
// la confrontation lit `doc.Objectives`, que `dropUnpublishedActions` vide quand aucune piste
// n'est nommée — or c'est le MÊME pont qui nomme les pistes et qui arme `CarryRead`. Les deux
// combinaisons « sans pont » couvertes ici sont donc les deux seules que la production peut
// produire, et la plus instructive y est : armements LUS, portage NON LU, `arms` ABSENT — la
// règle des DEUX canaux, celle qu'un `if in.ArmingsRead` seul casserait en silence.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// Le témoin : un joueur, un slot de biped, et son xuid en décimal — la clef que le noyau publie.
const (
	bwXUID    = uint64(2533274800000001)
	bwXUIDDec = "2533274800000001"
	bwSlot    = uint32(1)
)

// bwNavSlot est le point de navigation porteur de l'anneau d'armement. Il n'a AUCUN rapport avec
// le slot de biped : l'anneau est un marqueur d'écran (cf. document_bomb_armings.go).
const bwNavSlot = uint32(100)

// L'HORLOGE DU SCÉNARIO, posée une fois pour toutes — les trois grandeurs dont le recalage de
// production (`premierPaquetDuFilmUS/1000 − deathOffsetMS`) est fait :
//
//	premier paquet du FILM         1 000 000 µs  -> 1 000 ms
//	premier paquet de POSITION     2 000 000 µs  -> origine de la frame 0, originMs = 1 000 ms
//	fin de la vie du témoin       12 000 000 µs  -> 12 000 ms sur l'horloge du film
//	la mort qui la nomme              10 000 ms  -> deathOffsetMS = 12 000 − 10 000 = 2 000 ms
//
// d'où FilmToMatchOffsetMS = 1 000 − 2 000 = −1 000 ms : un instant du film se lit 1 000 ms
// PLUS TÔT sur l'horloge du match. Le décalage est volontairement d'un ordre de grandeur
// au-dessus du bruit (16-81 ms sur les films témoins) : un recalage oublié se VOIT.
const (
	bwFilmClockUS      = uint64(1_000_000)
	bwPremierePosUS    = uint64(2_000_000)
	bwDernierePosUS    = uint64(12_000_000)
	bwMortMS           = int64(10_000)
	bwDeathOffsetMS    = 2_000
	bwRecalageAttenduM = 1_000 - bwDeathOffsetMS // −1 000 ms
)

// bwPositions rend la trajectoire du témoin : un seul slot, un échantillon toutes les 500 ms
// (bien en deçà de `lifeGapUS`, donc UNE seule vie), de 2 s à 12 s de l'horloge moteur.
//
// `HasWorld` EST OBLIGATOIRE : sans les bornes de la carte, un quantum n'est pas une coordonnée
// et `decimateTracks` n'en publie AUCUNE piste — donc aucune identité, donc aucune action
// d'objectif publiable (`dropUnpublishedActions`). Tout le scénario en dépend.
func bwPositions() []filmdec.BipedPosition {
	var out []filmdec.BipedPosition
	for us := bwPremierePosUS; us <= bwDernierePosUS; us += 500_000 {
		n := float32(us / 500_000)
		out = append(out, filmdec.BipedPosition{
			Slot: bwSlot, TimestampUS: us, X: n, Y: n, Z: 1, HasWorld: true})
	}
	return out
}

// bwDeaths / bwIndices : les DEUX pièces du pont slot -> xuid. Retirer l'une des deux suffit à
// vider `own.SlotXUID`, et c'est exactement ce que fait le cas « sans pont ».
func bwDeaths() []Death {
	return []Death{{XUID: bwXUID, Gamertag: "Temoin", TimeMS: bwMortMS}}
}

func bwIndices() PlayerIndexTable {
	return PlayerIndexTable{ByXUID: map[uint64]int{bwXUID: 0}, Readings: 1}
}

// bwPortage rend les DEUX transitions du canal des armes tenues qui font UNE période fermée par
// LÂCHER : prise à 5 s, lâcher à 6 s (horloge du film) — soit [3 000, 4 000] ms sur l'horloge du
// match, une fois `deathOffsetMS` retranché (cf. bomb_carries.go). Une seconde de portage.
func bwPortage() []filmdec.HeldWeaponChange {
	return []filmdec.HeldWeaponChange{
		{TimestampUS: 5_000_000, Slot: bwSlot, Family: bombHeldFamily},
		{TimestampUS: 6_000_000, Slot: bwSlot, Family: 0x11112222, Previous: bombHeldFamily},
	}
}

// bwAnneauArme rend trois lectures de l'anneau qui forment UN segment d'armement : trois
// échantillons (`NavpointRiseMinSamples`), une montée de 154 quanta (`NavpointRiseMinQuanta` = 16)
// et une fin AU QUANTUM PLEIN (`bombArmedFullQuantum` = 254). `finMS` est l'instant ARMÉ, sur
// l'horloge du manifeste — la même que les explosions.
func bwAnneauArme(finMS int32) []filmdec.NavpointRadialRead {
	return []filmdec.NavpointRadialRead{
		{Slot: bwNavSlot, TMS: finMS - 400, Q: 100, Chained: true},
		{Slot: bwNavSlot, TMS: finMS - 200, Q: 200, Chained: true},
		{Slot: bwNavSlot, TMS: finMS, Q: 254, Chained: true},
	}
}

// bwExplosion rend l'action d'objectif qui porte UNE explosion de bombe, nommée par le témoin.
// Sans armement pour la précéder, elle fait ÉCHOUER la confrontation locale et retenir le calque
// entier — c'est le seul levier de production pour l'état `Suppressed`.
func bwExplosion(timeMS int) []objectiveevents.IdentifiedEvent {
	return []objectiveevents.IdentifiedEvent{{
		NamedEvent: objectiveevents.NamedEvent{
			TimeMS: timeMS, Slot: 10, Stat: objectiveevents.StatBombDetonations},
		XUID: bwXUIDDec,
	}}
}

// bwEtatAnneau nomme les trois états de `Coverage.BombArmings` que le câblage peut produire.
type bwEtatAnneau int

const (
	bwAnneauNonBalaye bwEtatAnneau = iota // `opt.Bomb.Scanned` faux -> couverture ABSENTE
	bwAnneauPublie                        // balayé, confrontation tenue -> calque publié
	bwAnneauRetenu                        // balayé, explosion orpheline -> calque SUPPRIMÉ
)

// bwCas décrit UN point du plan d'expérience. Les quatre attendus sont des POINTEURS : `nil` dit
// « la colonne doit être ABSENTE », un pointeur dit « mesurée, et à cette valeur » — la
// distinction que ce test existe pour défendre.
type bwCas struct {
	nom   string
	etat  bwEtatAnneau
	pont  bool // le pont slot -> xuid est fourni : c'est lui qui arme `CarryRead`
	score bool // une entrée de score est fournie : c'est elle qui arme `DetonationsRead`

	veutDetonations *int
	veutArms        *int
	veutGrabs       *int
	veutSecondes    *float64
}

// bwOptions assemble les entrées de production du cas. L'instant armé est posé à 2 000 ms
// (horloge du film) : recalé, il tombe à 1 000 ms, LOIN de la période [3 000, 4 000] — ni la
// règle du lâcher (fenêtre ±2 500 ms) ni le repli (couverture stricte) ne le nomment. `arms` sort
// donc à ZÉRO MESURÉ, ce qui est précisément la valeur qu'il ne faut pas confondre avec `nil`.
func bwOptions(c bwCas) Options {
	opt := Options{
		FilmClockOriginUS: bwFilmClockUS,
		WeaponChanges:     bwPortage(),
		Bomb:              BombInput{CarryScanned: true},
	}
	if c.pont {
		opt.Deaths, opt.PlayerIndices = bwDeaths(), bwIndices()
	}
	// L'EXPLOSION SERT DEUX RÔLES DISTINCTS, et c'est le point de câblage que ce test isole :
	// elle est la LIGNE JOUEUR des cas sans pont (seule source qui nomme quelqu'un quand le
	// portage n'est pas lu) ET le levier de l'état `Suppressed`. Elle n'est comptée en
	// `bomb_detonations` que si `opt.Score` la légitime — c'est `DetonationsRead`, et rien
	// d'autre, qui décide de la colonne.
	if c.score {
		opt.Score = scoreEntreeSynthetique()
	}
	if c.score || c.etat == bwAnneauRetenu {
		opt.Objectives = bwExplosion(5_000)
	}
	switch c.etat {
	case bwAnneauPublie:
		opt.Bomb.Scanned = true
		opt.Bomb.Reads = bwAnneauArme(2_000)
	case bwAnneauRetenu:
		opt.Bomb.Scanned = true
	case bwAnneauNonBalaye:
	}
	return opt
}

func bwInt(v int) *int         { return &v }
func bwSec(v float64) *float64 { return &v }

// TestBombStatsCablageAbsentNestPasZero — les colonnes non lues sortent ABSENTES du chemin de
// production, jamais à zéro, dans les cinq combinaisons que le câblage peut produire.
func TestBombStatsCablageAbsentNestPasZero(t *testing.T) {
	cas := []bwCas{{
		nom: "portage lu · anneau NON balaye", etat: bwAnneauNonBalaye, pont: true,
		veutGrabs: bwInt(1), veutSecondes: bwSec(1),
	}, {
		nom: "portage lu · anneau balaye et PUBLIE", etat: bwAnneauPublie, pont: true,
		// arms = 0 MESURE : les deux canaux sont lus, aucun armement n'est attribuable.
		veutArms: bwInt(0), veutGrabs: bwInt(1), veutSecondes: bwSec(1),
	}, {
		nom: "portage lu · anneau balaye mais RETENU a la source", etat: bwAnneauRetenu, pont: true,
		veutGrabs: bwInt(1), veutSecondes: bwSec(1),
	}, {
		nom: "SANS pont · anneau NON balaye", etat: bwAnneauNonBalaye, score: true,
		veutDetonations: bwInt(1),
	}, {
		// LE CAS QUI PORTE LA REGLE DES DEUX CANAUX : l'anneau est LU et publié, le portage ne
		// l'est pas, et `arms` reste ABSENT — un `if in.ArmingsRead` seul publierait 0 ici.
		nom: "SANS pont · anneau balaye et PUBLIE", etat: bwAnneauPublie, score: true,
		veutDetonations: bwInt(1),
	}}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			doc := BuildFromPositions("m", "halo_infinite", bwPositions(), nil, bwOptions(c))
			bwVerifieCouverture(t, doc, c)
			st := bwStats(t, doc)
			bwVerifieColonnes(t, st, c)
		})
	}
}

// bwStats rend les statistiques posées sur le document, en échouant si le calque n'est pas câblé.
func bwStats(t *testing.T, doc ReplayDocument) BombMatchStats {
	t.Helper()
	if doc.BombStats == nil {
		t.Fatal("document SANS bombStats alors que opt.Bomb.CarryScanned est vraie : le calque " +
			"n'est pas cable dans BuildFromPositions")
	}
	if len(doc.BombStats.Players) != 1 {
		t.Fatalf("%d ligne(s) joueur publiee(s), attendu 1 : %+v",
			len(doc.BombStats.Players), doc.BombStats.Players)
	}
	return *doc.BombStats
}

// bwVerifieCouverture confronte l'état de `Coverage.BombArmings` à celui que le cas voulait, PUIS
// les trois témoins de lecture que le site de câblage a dérivés. Sans cette première vérification,
// un cas qui n'aurait pas produit l'état voulu (une explosion hors fenêtre, un segment refusé)
// passerait pour une preuve alors qu'il ne mesure rien.
func bwVerifieCouverture(t *testing.T, doc ReplayDocument, c bwCas) {
	t.Helper()
	if doc.Coverage == nil {
		t.Fatal("document sans couverture")
	}
	ann := doc.Coverage.BombArmings
	switch c.etat {
	case bwAnneauNonBalaye:
		if ann != nil {
			t.Fatalf("coverage.bombArmings publiee sans balayage : %+v", ann)
		}
	case bwAnneauPublie:
		if ann == nil || !ann.Scanned || ann.Suppressed {
			t.Fatalf("coverage.bombArmings = %+v, attendu balayee et NON supprimee", ann)
		}
	case bwAnneauRetenu:
		if ann == nil || !ann.Scanned || !ann.Suppressed {
			t.Fatalf("coverage.bombArmings = %+v, attendu balayee et SUPPRIMEE", ann)
		}
	}
	cov := doc.BombStats.Coverage
	veutArmings := c.etat == bwAnneauPublie
	if cov.ArmingsRead != veutArmings {
		t.Errorf("coverage.armingsRead = %v, attendu %v (etat de l'anneau : %+v)",
			cov.ArmingsRead, veutArmings, ann)
	}
	if cov.CarryRead != c.pont {
		t.Errorf("coverage.carryRead = %v, attendu %v : le temoin ne suit pas le pont slot -> xuid",
			cov.CarryRead, c.pont)
	}
	if cov.DetonationsRead != c.score {
		t.Errorf("coverage.detonationsRead = %v, attendu %v : le temoin ne suit pas opt.Score",
			cov.DetonationsRead, c.score)
	}
}

// bwVerifieColonnes applique la règle, colonne par colonne : un attendu `nil` exige l'ABSENCE
// (et le message dit explicitement quand un zéro a été publié à la place) ; un attendu pointé
// exige la valeur.
func bwVerifieColonnes(t *testing.T, st BombMatchStats, c bwCas) {
	t.Helper()
	p := bombRowOf(t, st, bwXUIDDec)
	bwColonneInt(t, "detonations", p.Detonations, c.veutDetonations)
	bwColonneInt(t, "arms", p.Arms, c.veutArms)
	bwColonneInt(t, "grabs", p.Grabs, c.veutGrabs)
	if c.veutSecondes == nil {
		if p.TimeAsCarrierSeconds != nil {
			t.Errorf("timeAsCarrierSeconds = %v alors que la source n'est pas lue — ABSENT n'est "+
				"pas ZERO", *p.TimeAsCarrierSeconds)
		}
		return
	}
	if p.TimeAsCarrierSeconds == nil {
		t.Errorf("timeAsCarrierSeconds absent, attendu %v", *c.veutSecondes)
		return
	}
	if *p.TimeAsCarrierSeconds != *c.veutSecondes {
		t.Errorf("timeAsCarrierSeconds = %v, attendu %v", *p.TimeAsCarrierSeconds, *c.veutSecondes)
	}
}

func bwColonneInt(t *testing.T, nom string, got, veut *int) {
	t.Helper()
	switch {
	case veut == nil && got != nil:
		t.Errorf("%s = %d alors que la source n'est pas lue — ABSENT n'est pas ZERO : un %d "+
			"publie se lit « mesure faite », ce qui est le contraire de ce qui s'est passe",
			nom, *got, *got)
	case veut != nil && got == nil:
		t.Errorf("%s absent, attendu %d", nom, *veut)
	case veut != nil && *got != *veut:
		t.Errorf("%s = %d, attendu %d", nom, *got, *veut)
	}
}

// TestBombStatsCablageRecalageHorloge — LE RECALAGE EST CELUI DE LA PRODUCTION, et il est
// appliqué DANS LE BON SENS.
//
// `TestBombArmsRecalageHorloge` prouve que le NOYAU consomme `FilmToMatchOffsetMS` ; il le reçoit
// tout fait. Ce test-ci prouve que le SITE DE CÂBLAGE le CALCULE — `premierPaquetDuFilmUS/1000 −
// deathOffsetMS` (bomb_stats_document.go, dérivation en tête de bomb_arms.go) — et il pin le
// diagnostic du gate sur films, qui affiche la même grandeur.
//
// LE SCÉNARIO EST DISCRIMINANT PAR CONSTRUCTION. L'armement est daté 7 000 ms sur l'horloge du
// film, la période de portage se ferme par lâcher à 4 000 ms sur celle du match, et la fenêtre de
// la règle primaire vaut ±2 500 ms (`bombArmDropWindowMS`) :
//
//	recalage de production (−1 000)  ->  armement à 6 000 ms match, écart 2 000 ms  -> ATTRIBUÉ
//	recalage oublié (0)              ->  armement à 7 000 ms match, écart 3 000 ms  -> anonyme
//	recalage de signe inversé (+1 000)-> armement à 8 000 ms match, écart 4 000 ms  -> anonyme
//
// et le repli ne rattrape rien : il exige une couverture STRICTE de l'instant armé par la
// période, que ni 7 000 ni 8 000 ne satisfont. Vérifié par mutation le 2026-09-05.
func TestBombStatsCablageRecalageHorloge(t *testing.T) {
	opt := bwOptions(bwCas{etat: bwAnneauPublie, pont: true})
	opt.Bomb.Reads = bwAnneauArme(7_000)

	doc := BuildFromPositions("m", "halo_infinite", bwPositions(), nil, opt)
	st := bwStats(t, doc)
	assertBombInt(t, st, bwXUIDDec, func(p BombPlayerStats) *int { return p.Arms }, 1, "arms")
	cov := st.Coverage
	if cov.Armings != 1 || cov.ArmingsAttributed != 1 || cov.ArmingsByDrop != 1 {
		t.Fatalf("armements %d / attribues %d / par lacher %d, attendu 1/1/1 — le recalage "+
			"applique par attachBombStats n'est pas %d ms",
			cov.Armings, cov.ArmingsAttributed, cov.ArmingsByDrop, bwRecalageAttenduM)
	}
	// Le fait daté reste publié sur l'horloge du FILM : c'est celle des explosions, et la
	// persistance les pose sur le même axe.
	var armes []BombEvent
	for _, e := range doc.BombEvents {
		if e.Type == BombEventArmed {
			armes = append(armes, e)
		}
	}
	assertBombEvents(t, armes, []BombEvent{{Type: BombEventArmed, TimeMS: 7_000,
		XUID: bwXUIDDec, ActorSource: BombActorSourceDrop}})
}
