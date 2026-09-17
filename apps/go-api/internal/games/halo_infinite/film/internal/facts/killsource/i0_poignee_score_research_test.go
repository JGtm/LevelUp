package killsource

// i0_poignee_score_research_test.go — LE BALAYAGE DU MOT DE POIGNEE EST-IL SCORE DANS LE MONDE
// QUE LA PRODUCTION DECODE ?
//
// # LA QUESTION
//
// `infererLargeurs` (calibrate.go) balaie DEUX grandeurs ensemble — une largeur d axe UNIFORME
// `aw` et la largeur du mot de poignee `iw` — et retient le COUPLE de meilleur score. Depuis le
// lot 3.4.1 la production ne lit plus jamais une largeur d axe uniforme : elle lit le TRIPLET de
// la carte. Le `iw` retenu est donc l argmax dans un monde ou l axe vaut `aw/aw/aw`, pas dans
// celui que le decodeur habite.
//
// Cet instrument mesure le MEME critere (`countBipedRecords`) des DEUX facons, sur un film :
//
//	UNIFORME   `WorldObject.AxisW = {aw, aw, aw}`, aw balaye — ce que le lot fait aujourd hui ;
//	TRIPLET    `WorldObject.AxisW` = les largeurs LUES de la carte — ce que la production lit.
//
// Si les deux ne designent pas le meme `iw`, le balayage decide sur un modele perime, et les
// lecteurs derriere i0 (`ScanAbilityImpulses`, `ScanGrappleReads`, les tapis) en portent la
// consequence.
//
// LECTURE SEULE, N ASSERTE RIEN, garde par `KS_POIGNEE_FILM` :
//
//	KS_POIGNEE_FILM=<repo>/data/cache/film_chunks/a521164d KS_POIGNEE_CARTE='Fragmentation Heavies' \
//	  go test ./internal/games/halo_infinite/film/internal/facts/killsource/ -run '^TestScoreDuMotDePoignee$' -v

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestScoreDuMotDePoignee(t *testing.T) {
	dir := os.Getenv("KS_POIGNEE_FILM")
	if dir == "" {
		t.Skip("instrument de mesure : KS_POIGNEE_FILM requis")
	}
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chargement %s : %v", dir, err)
	}
	o := DefaultOptions()
	if nom := os.Getenv("KS_POIGNEE_CARTE"); nom != "" {
		e := carteDuCatalogue(t, nom)
		o.Carte = &e
	}
	o.normalize()
	c := &decodeCtx{name: filepath.Base(dir), opts: o}
	f, err := loadFilm(src)
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	c.film = f
	tl, err := newTimeline(f)
	if err != nil {
		t.Fatalf("timeline : %v", err)
	}
	sample := calibSample(f, calibSampleSize)
	base := grammar.DefaultFrameConfig()
	base.Profil, _ = ProfilDeDepartPourCarte(o.Carte)
	lues := base.Profil.Mouvement.WorldObject
	saved := base.Profil.Mouvement.Traversal
	t.Logf("%s : largeurs LUES de la carte %v indexW_plage=%d ; echantillon %d paquets",
		c.name, lues.AxisW, lues.IndexW, len(sample))

	score := func(axes [3]uint, iw uint) int {
		cfg := base
		cfg.Profil.Mouvement.WorldObject = lues
		cfg.Profil.Mouvement.WorldObject.AxisW = axes
		cfg.Profil.Mouvement.Traversal = profile.PrecisionDescriptor{IndexW: iw, AxisW: saved.AxisW}
		return countBipedRecords(sample, tl, cfg, o.Views)
	}

	t.Logf("--- TRIPLET LU %v (ce que la production decode) ---", lues.AxisW)
	for iw := indexWMin; iw <= indexWMax; iw++ {
		t.Logf("  iw=%d  score=%d", iw, score(lues.AxisW, iw))
	}
	t.Logf("--- UNIFORME (ce que le balayage score aujourd hui) : meilleur aw par iw ---")
	for iw := indexWMin; iw <= indexWMax; iw++ {
		bestAW, best := uint(0), -1
		for aw := axisWMin; aw <= axisWMax; aw++ {
			if s := score([3]uint{aw, aw, aw}, iw); s > best {
				bestAW, best = aw, s
			}
		}
		t.Logf("  iw=%d  meilleur aw=%d  score=%d", iw, bestAW, best)
	}
}
