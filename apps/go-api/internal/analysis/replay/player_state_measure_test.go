package replay

// player_state_measure_test.go — INSTRUMENT DE MESURE de la moitie de B.0.4 qui a besoin du
// FIL DES MORTS : le delai entre une mort et la reapparition suivante du meme joueur, et le
// temps mort cumule. Phase 0 du lot B.
//
// POURQUOI CET INSTRUMENT EST SCINDE EN DEUX PAQUETS. Il portait aussi P.0.3 (l'arme en main)
// et P.0.4 (la seconde source de visee). Ces deux-la consomment la CHAINE SEQUENTIELLE, et la
// regle « 0 code mort » a fait des trois scanners des fichiers `_test.go` de `filmdec` — donc
// invisibles depuis ici. Les mesures qui consomment la chaine ont suivi la chaine
// (`filmdec/player_bridge_measure_test.go`) ; celles qui consomment le fil des morts et le pont
// des vies sont restees. Le partage est fait PAR DEPENDANCE, et aucun decodage n'est en double.
//
// CE QUE CE QUI RESTE MESURE, ET POURQUOI ON LE GARDE MALGRE UN LOT CLOS. La fenetre ACTIVE du
// compte a rebours de ti=5 ne couvre que 0,85-4,30 % des morts : le canal du film ne dit pas le
// temps mort. Les TRAJECTOIRES, elles, le disent tres bien — fin d'une vie, debut de la vie
// suivante DU MEME JOUEUR (jamais du meme slot : le slot migre a la reapparition). C'est le
// seul acquis publiable du lot B, et il ne vient pas de l'entite moteur.
//
// LE SEUIL, ECRIT AVANT LA MESURE (D13) : fenetre `Active` du compte a rebours (ti=5 i1) contre
// le delai mort -> reapparition, a +/- 1 s, sur >= 90 % des morts.
//
// LECTURE SEULE, garde par PLAYER_FILM, saute partout ailleurs (CI comprise). UN SEUL film par
// processus (D17).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 PLAYER_FILM=C:/.../data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/replay/ -run '^TestPlayerBridgePhase0$' -timeout 30m -v

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const playerFilmEnv = "PLAYER_FILM"

func TestPlayerBridgePhase0(t *testing.T) {
	dir := os.Getenv(playerFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", playerFilmEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	in := psLoad(t, dir)
	psLogInputs(t, in)
	psRespawn(t, in)
}

// psInputs porte tout ce que le film a rendu, une seule lecture par flux.
type psInputs struct {
	dir, short string
	pos        []filmdec.BipedPosition
	shots      []filmdec.FireEvent
	deaths     []Death
	lives      []lifeSpan
	// own porte le pont slot de bipede -> joueur, construit par le chemin de PRODUCTION.
	own OwnerReport
}

// psLoad lit le film une fois pour toutes. Chaque flux manquant est DIT, jamais remplace.
func psLoad(t *testing.T, dir string) psInputs {
	t.Helper()
	in := psInputs{dir: dir, short: filepath.Base(strings.TrimRight(filepath.Clean(dir), string(filepath.Separator)))}
	scan := filmdec.DefaultScanFilmOptions()
	scan.CaptureDirs = true
	// AUCUNE BORNE DE CARTE N EST DEMANDEE, et c est un choix : ce qui est mesure ici ne compare
	// que des INSTANTS et des SLOTS. Exiger les bornes du BSP obligerait a resoudre le catalogue
	// de la carte pour chaque film du corpus, sans qu une seule coordonnee monde n entre dans le
	// calcul. Contrepartie ASSUMEE et dite : sans bornes, le filtre de vitesse est inoperant,
	// donc une position aberrante n est plus ecartee — elle peut allonger une vie, jamais en
	// fabriquer une.
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions de bipede illisibles : %v", err)
	}
	if in.shots, err = filmdec.ScanFilmFireEvents(dir); err != nil {
		t.Logf("tirs illisibles : %v", err)
	}
	if in.deaths, err = ScanFilmDeaths(dir); err != nil {
		t.Logf("fil des morts illisible : %v", err)
	}
	sorted := append([]filmdec.BipedPosition(nil), pos...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampUS < sorted[j].TimestampUS })
	in.pos = sorted
	tracks := indexBySlot(sorted)
	in.lives = buildLifeSpans(tracks)
	var idx PlayerIndexTable
	if len(in.deaths) > 0 {
		if raw, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(in.deaths)); err == nil {
			idx, _ = injectiveOrEmpty(raw)
		}
	}
	in.own = buildOwners(tracks, in.deaths, idx, fireRefs(in.shots))
	return in
}

func psLogInputs(t *testing.T, in psInputs) {
	t.Helper()
	t.Logf("FILM %s · positions %d · tirs %d · morts %d · vies %d · slots nommes %d "+
		"(vies nommees %d / %d)", in.short, len(in.pos), len(in.shots), len(in.deaths),
		len(in.lives), len(in.own.Owner), in.own.DeathsNamed, in.own.LivesTotal)
}

// psRespawn mesure B.0.4 : le delai reel entre la mort et la reapparition suivante du meme
// joueur, et le temps mort cumule. Le compte des fenetres ACTIVES du canal ti=5, qui est
// l'autre moitie de l'item, est rendu par `filmdec/player_bridge_measure_test.go`.
func psRespawn(t *testing.T, in psInputs) {
	t.Helper()
	gaps := psDeathToSpawnGaps(in)
	sort.Float64s(gaps)
	med := 0.0
	if len(gaps) > 0 {
		med = gaps[len(gaps)/2]
	}
	t.Logf("B.0.4 TEMPS MORT · morts du fil %d · intervalles mort -> reapparition mesurables %d "+
		"· mediane %.2f s", len(in.deaths), len(gaps), med)
	if len(gaps) == 0 {
		t.Log("B.0.4 VERDICT · NEGATIF : aucun intervalle mesurable sur ce film — le pont des " +
			"vies n'a nomme aucun joueur")
		return
	}
	psTeamDowntime(t, gaps)
}

// psDeathToSpawnGaps rend, en secondes, l'intervalle entre la fin d'une vie et le debut de la
// vie suivante DU MEME JOUEUR (jamais du meme slot : le slot migre a la reapparition).
func psDeathToSpawnGaps(in psInputs) []float64 {
	byXUID := map[uint64][]lifeSpan{}
	for _, l := range in.lives {
		if x, ok := in.own.SlotXUID[l.slot]; ok {
			byXUID[x] = append(byXUID[x], l)
		}
	}
	var out []float64
	for _, ls := range byXUID {
		sort.Slice(ls, func(i, j int) bool { return ls[i].from < ls[j].from })
		for i := 1; i < len(ls); i++ {
			if d := ls[i].from - ls[i-1].to; d > 0 {
				out = append(out, float64(d)/1e6)
			}
		}
	}
	return out
}

// psTeamDowntime rend le temps mort cumule, par joueur puis somme — le chiffre que la phase 2
// du lot B voulait afficher en en-tete. L'EQUIPE n'est pas resolue ici : le film ne la donne
// pas de facon fiable (decision de 2026-06 sur les evenements), et l'inventer serait pire que
// de rendre le total par joueur.
func psTeamDowntime(t *testing.T, gaps []float64) {
	t.Helper()
	total := 0.0
	for _, g := range gaps {
		total += g
	}
	t.Logf("B.0.4 TEMPS MORT CUMULE · %d intervalles · total %.1f s · moyenne %.2f s "+
		"(equipe NON resolue : le film ne la porte pas de facon fiable)",
		len(gaps), total, total/float64(len(gaps)))
}
