package killcollector

// positions_openings_test.go — L'ACCORD ENTRE LE DÉCALAGE DE L'ENTAME ET L'INSTANT PERSISTÉ,
// et ce que les compteurs de la passe disent.
//
// LE DÉFAUT QUE CE FICHIER FERME (revue adversariale du 2026-09-06, constat B1). Avant lui,
// AUCUN test ne pinçait l'accord entre les DEUX gestes de l'entame — le décalage des couples
// et le `time_ms` écrit en base. Inverser le signe du décalage laissait toute la suite VERTE :
// les lignes auraient porté des coordonnées prises 1,5 s APRÈS la mort et un `time_ms` décalé
// de +3 s, donc une jointure `kp.time_ms = e.time_ms` vide POUR TOUJOURS — sans une seule
// erreur, sans une seule ligne de journal. Le seul test qui existait
// (`TestToKillOpeningRows_LInstantRedevientCeluiDuKill`, positions_test.go) épinglait la
// PROJECTION sur une entrée fabriquée à la main : il ne pouvait rien dire du producteur qui la
// remplit.
//
// AUCUN FILM ICI : c'est précisément pour cela que `composerPassePositions` a été extraite de
// la lecture du film. Les trajectoires sont fabriquées, la réponse est connue d'avance.

import (
	"context"
	"math"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/observability"
)

// registreSynthetique monte un VRAI registre d'identite sur des trajectoires fabriquees, par le
// LIEN DIRECT — le record de creation du bipede, qui ECRIT l'index de participant du proprietaire
// du corps. C'est la voie que le film emploie, et la seule qui soit deterministe sans fil des
// morts (l'appariement des morts departage par l'ordre des slots quand deux vies finissent au
// meme instant, ce que ces trajectoires font toutes).
//
// POURQUOI IL REMPLACE UNE TABLE slot -> xuid (lot 6.1, 2026-09-10) : `composerPassePositions`
// prend desormais le registre, qui repond A L'INSTANT. Lui passer une table aplatie serait
// exactement ce que ce lot a retire du chemin de production.
func registreSynthetique(positions []filmdec.BipedPosition,
	slotXUID map[uint32]uint64) replay.IdentityRegistry {
	sieges := make([]uint32, 0, len(slotXUID))
	for s := range slotXUID {
		sieges = append(sieges, s)
	}
	sort.Slice(sieges, func(i, j int) bool { return sieges[i] < sieges[j] })
	idx := replay.PlayerIndexTable{ByXUID: map[uint64]int{}, Readings: 1}
	creations := make([]filmdec.BipedCreation, 0, len(sieges))
	for i, s := range sieges {
		idx.ByXUID[slotXUID[s]] = i
		creations = append(creations, filmdec.BipedCreation{
			Slot: s, Generation: 1, ParticipantIndex: uint32(i), HasIndex: true})
	}
	return replay.BuildIdentityRegistry(replay.IdentityInput{
		Positions: positions, BipedCreations: creations, PlayerIndices: idx})
}

// bipedAt fabrique un échantillon de trajectoire monde — le type que `ScanBipedPositions` rend.
func bipedAt(slot uint32, tMS int64, x, y, z float32) filmdec.BipedPosition {
	return filmdec.BipedPosition{
		Slot: slot, TimestampUS: uint64(tMS) * 1000,
		X: x, Y: y, Z: z, HasWorld: true,
	}
}

// trajectoiresSynthetiques : deux joueurs échantillonnés toutes les 100 ms (bien en deçà de la
// tolérance de 120 ms du placement) de 0 à finMS — donc UNE SEULE vie chacun, sans trou. Le
// tueur (slot 1) avance : son X vaut le temps en SECONDES, ce qui rend chaque instant
// identifiable par sa seule coordonnée. La victime (slot 2) ne bouge pas.
func trajectoiresSynthetiques(finMS int64) []filmdec.BipedPosition {
	var out []filmdec.BipedPosition
	for t := int64(0); t <= finMS; t += 100 {
		out = append(out, bipedAt(1, t, float32(t)/1000, 0, 0))
		out = append(out, bipedAt(2, t, 20, 0, 0))
	}
	return out
}

// toleranceX : les coordonnées voyagent en float32 puis en float64 ; on compare à l'epsilon.
const toleranceX = 1e-6

// TestComposerPassePositions_LEntameEstPriseAvantLeKillEtPorteSonInstant — LE test de l'accord.
//
// Il vérifie les DEUX faits ENSEMBLE, parce que c'est leur accord qui est fragile :
// (i) la ligne d'entame porte le `time_ms` DU KILL — la seule clé de jointure ;
// (ii) ses coordonnées sont celles de l'échantillon situé `OpeningLeadMS` AVANT le kill,
// distinctes de celles du kill ET de celles du même écart APRÈS (le signe inversé).
func TestComposerPassePositions_LEntameEstPriseAvantLeKillEtPorteSonInstant(t *testing.T) {
	const mortMS = int64(5000)
	positions := trajectoiresSynthetiques(8000)
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []replay.KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: mortMS}}

	pass := composerPassePositions(positions, registreSynthetique(positions, slotXUID), kills, 0, "m1")

	if len(pass.rows) != 1 || len(pass.openRows) != 1 {
		t.Fatalf("attendu 1 position et 1 entame, obtenu %d et %d", len(pass.rows), len(pass.openRows))
	}
	fatal, entame := pass.rows[0], pass.openRows[0]

	// (i) LES DEUX LIGNES PORTENT LE MÊME INSTANT : celui du kill.
	if int64(fatal.TimeMS) != mortMS || int64(entame.TimeMS) != mortMS {
		t.Fatalf("time_ms = position %d / entame %d, attendu %d des deux côtés (l'instant DU KILL)",
			fatal.TimeMS, entame.TimeMS, mortMS)
	}

	// (ii) LES COORDONNÉES, ELLES, DIFFÈRENT — et dans le BON sens.
	xAuKill := float64(mortMS) / 1000                     // 5,0
	xAvant := float64(mortMS-replay.OpeningLeadMS) / 1000 // 3,5
	xApres := float64(mortMS+replay.OpeningLeadMS) / 1000 // 6,5
	if fatal.KillerX == nil || math.Abs(*fatal.KillerX-xAuKill) > toleranceX {
		t.Fatalf("position du coup fatal : killer_x = %v, attendu %v", fatal.KillerX, xAuKill)
	}
	if entame.KillerX == nil {
		t.Fatal("entame sans position de tueur : le placement n'a rien rendu")
	}
	if math.Abs(*entame.KillerX-xAvant) > toleranceX {
		t.Errorf("entame : killer_x = %v, attendu %v (l'échantillon %d ms AVANT le kill)",
			*entame.KillerX, xAvant, replay.OpeningLeadMS)
	}
	if math.Abs(*entame.KillerX-xAuKill) <= toleranceX {
		t.Errorf("entame : killer_x = %v — c'est la position DU KILL : le décalage n'a pas été appliqué",
			*entame.KillerX)
	}
	if math.Abs(*entame.KillerX-xApres) <= toleranceX {
		t.Errorf("entame : killer_x = %v — c'est la position %d ms APRÈS le kill : le SIGNE du "+
			"décalage est inversé", *entame.KillerX, replay.OpeningLeadMS)
	}
	// La victime n'a pas bougé : sa coordonnée ne distingue aucun instant, mais son ABSENCE
	// dirait que le placement a échoué de ce côté.
	if entame.VictimX == nil || math.Abs(*entame.VictimX-20) > toleranceX {
		t.Errorf("entame : victim_x = %v, attendu 20", entame.VictimX)
	}
}

// TestComposerPassePositions_ReapparitionEcarteLEntameEtLaCompte — le filtre « même vie » de
// `replay.BuildKillOpenings` remonte bien jusqu'ici : le tueur a réapparu entre l'entame et le
// coup fatal, donc sa position d'avant est un POINT D'APPARITION, jamais une entame. Le côté
// est écarté et COMPTÉ ; la position du coup fatal, elle, reste entière.
func TestComposerPassePositions_ReapparitionEcarteLEntameEtLaCompte(t *testing.T) {
	const mortMS = int64(20_000)
	// Victime : une vie continue. Tueur : une vie qui s'arrête à 10 s, puis une NOUVELLE vie
	// à partir de 18,56 s (trou de 8,5 s, bien au-delà de lifeGapUS). L'entame tombe à 18,5 s,
	// à portée de tolérance du PREMIER échantillon de la vie neuve — exactement le piège.
	var positions []filmdec.BipedPosition
	for tMS := int64(0); tMS <= mortMS+1000; tMS += 100 {
		positions = append(positions, bipedAt(2, tMS, 20, 0, 0))
	}
	for tMS := int64(0); tMS <= 10_000; tMS += 100 {
		positions = append(positions, bipedAt(1, tMS, 1, 0, 0))
	}
	for tMS := int64(18_560); tMS <= mortMS+1000; tMS += 100 {
		positions = append(positions, bipedAt(1, tMS, 2, 0, 0))
	}
	slotXUID := map[uint32]uint64{1: 111, 2: 222}
	kills := []replay.KillRef{{KillerXUID: 111, VictimXUID: 222, TimeMS: mortMS}}

	pass := composerPassePositions(positions, registreSynthetique(positions, slotXUID), kills, 0, "m1")

	if len(pass.rows) != 1 || pass.rows[0].KillerX == nil {
		t.Fatalf("la position du coup fatal doit rester entière : %+v", pass.rows)
	}
	if pass.openRep.OpeningOutOfLife != 1 {
		t.Errorf("OpeningOutOfLife = %d, attendu 1 (le côté tueur écarté)",
			pass.openRep.OpeningOutOfLife)
	}
	if len(pass.openRows) != 1 || pass.openRows[0].KillerX != nil {
		t.Fatalf("l'entame du tueur devait être ABSENTE (point d'apparition) : %+v", pass.openRows)
	}
}

// ─── publishOpeningsPass : ce que les compteurs disent, et ce qu'ils taisent ────────────────

// TestPublishOpeningsPass_ZeroLigneNeCompteAucunMatch — la garde `rowsWritten == 0` (constat
// C8, non testée jusqu'ici). Compter un match « couvert » sans aucune ligne rendrait le
// compteur muet sur la seule question qu'il sert à poser : sur combien de matchs ce proxy
// est-il RÉELLEMENT mesuré.
func TestPublishOpeningsPass_ZeroLigneNeCompteAucunMatch(t *testing.T) {
	avant := observability.LoadCounter(metricOpeningsMatches)
	publishOpeningsPass(context.Background(), "m1", replay.KillPosReport{Kills: 4, Dropped: 4}, 0)
	if got := observability.LoadCounter(metricOpeningsMatches) - avant; got != 0 {
		t.Errorf("%s a bougé de %d sur une passe à zéro ligne, attendu 0", metricOpeningsMatches, got)
	}
}

// TestPublishOpeningsPass_CotesHorsVieComptes — le compteur ajouté par la bascule sur
// `BuildKillOpenings` (item 3.10 bis). Sans lui, un film dont toutes les entames tombent se
// lirait comme une couverture basse, sans cause. Il compte MÊME quand rien n'est écrit : il
// mesure la LECTURE, pas l'écriture.
func TestPublishOpeningsPass_CotesHorsVieComptes(t *testing.T) {
	avant := observability.LoadCounter(metricOpeningsOutOfLife)
	publishOpeningsPass(context.Background(), "m1",
		replay.KillPosReport{Kills: 3, Dropped: 3, OpeningOutOfLife: 5}, 0)
	if got := observability.LoadCounter(metricOpeningsOutOfLife) - avant; got != 5 {
		t.Errorf("%s a bougé de %d, attendu 5", metricOpeningsOutOfLife, got)
	}
}
