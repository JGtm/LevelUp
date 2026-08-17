package filmdec

// player_bridge_measure_test.go — LES ITEMS QUI ONT BESOIN DE LA CHAINE SEQUENTIELLE :
// P.0.3 (l'arme en main image par image), P.0.4 (la seconde source de visee) et le compte des
// fenetres actives de reapparition (moitie de B.0.4).
//
// POURQUOI ILS VIVENT ICI ET NON DANS `replay`. Ils etaient d'abord dans
// `replay/player_state_measure_test.go`, avec le reste du pont. La regle « 0 code mort » a fait
// des trois scanners des fichiers `_test.go` de CE paquet — et un fichier de test n'est pas
// visible depuis un autre paquet. Les mesures qui consomment la chaine ont donc suivi la
// chaine ; celles qui consomment le fil des morts et le pont des vies sont restees dans
// `replay`. Le partage est fait PAR DEPENDANCE, et il n'y a aucun decodage en double.
//
// LA SEULE CHOSE QUE CE DEPLACEMENT COUTE, ET COMMENT ELLE EST PAYEE. Le pont
// slot de bipede -> index de joueur du film vient de `replay` (fil des morts) et ne peut pas
// suivre. P.0.3 compare donc l'arme du tir a une lecture d'arme en main trouvee sur N'IMPORTE
// QUEL slot dans la fenetre, au lieu du seul slot du tireur : c'est une BORNE SUPERIEURE de la
// couverture reelle. Une borne superieure nulle prouve une couverture nulle, donc le negatif
// mesure n'est pas affaibli — il est rendu plus difficile a atteindre, et il tient quand meme.
//
// LES SEUILS, ECRITS AVANT LA MESURE (D13) :
//
//	P.0.3  au frame de chaque TIR, l'arme en main est de la meme FAMILLE que l'arme du tir, sur
//	       >= 90 % des tirs. Plus la cadence du canal et le recensement de ses annonces.
//	P.0.4  quand la visee du CORPS (i21) et celle du JOUEUR (ti=5 i17) tombent a <= 100 ms l'une
//	       de l'autre sur le meme joueur, |delta cap| <= 5° sur >= 90 % des paires.
//
// LECTURE SEULE, garde par BRIDGE_FILM, saute partout ailleurs (CI comprise). UN SEUL film par
// processus (D17).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 BRIDGE_FILM=C:/.../data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestPlayerChannelsPhase0$' -timeout 30m -v

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	bridgeFilmEnv = "BRIDGE_FILM"
	// pbPairWindowUS : ecart maximal entre une lecture de visee du corps et une lecture de
	// visee du joueur pour que les deux forment une paire (100 ms, seuil du plan P.0.4).
	pbPairWindowUS = 100_000
	// pbShotWindowUS : anteriorite maximale d'une lecture d'arme en main pour valoir « arme du
	// tireur au moment du tir ». Deux secondes : le rattachement des tirs du rejeu se joue a
	// ~120 ms, mais le canal i43-i46 n'est transmis qu'au CHANGEMENT d'arme — exiger une
	// lecture dans la meme frame mesurerait la frequence de transmission, pas l'accord.
	pbShotWindowUS = 2_000_000
	// pbLifeGapUS : au-dela de ce trou dans un meme slot, une nouvelle vie commence. MEME
	// valeur que `replay.lifeGapUS`, et pour la meme raison mesuree : tres au-dessus du pas de
	// replication (~16 ms), tres en deca du temps de reapparition median (8,0 s).
	pbLifeGapUS = 5_000_000
	// pbBridgeWindowUS : tolerance d'appariement entre une fenetre active de reapparition et
	// une fin de vie de bipede.
	pbBridgeWindowUS = 5_000_000
)

func TestPlayerChannelsPhase0(t *testing.T) {
	dir := os.Getenv(bridgeFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", bridgeFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	in := pbLoad(t, dir)
	pbLogInputs(t, in)
	pbHeldWeapon(t, in)
	pbAimPairs(t, in)
	pbRespawnWindows(t, in)
	pbDump(t, in)
}

// pbInputs porte tout ce que le film a rendu, une seule lecture par flux.
type pbInputs struct {
	dir, short string
	pos        []BipedPosition
	shots      []FireEvent
	held       []HeldWeaponSample
	player     []GameEntityRecord
	chain      GameChainStats
	// ends[slot] : fins de vie du slot de bipede, en microsecondes.
	ends map[uint32][]uint64
}

// pbLoad lit le film une fois pour toutes. Chaque flux manquant est DIT, jamais remplace.
func pbLoad(t *testing.T, dir string) pbInputs {
	t.Helper()
	in := pbInputs{dir: dir, short: filepath.Base(strings.TrimRight(filepath.Clean(dir), string(filepath.Separator)))}
	scan := DefaultScanFilmOptions()
	scan.CaptureDirs = true
	// AUCUNE BORNE DE CARTE N EST DEMANDEE, et c est un choix : les items mesures ici ne
	// comparent que des INSTANTS, des SLOTS et des CAPS. Contrepartie ASSUMEE et dite : sans
	// bornes, le filtre de vitesse est inoperant, donc une position aberrante n est plus
	// ecartee — elle peut allonger une vie, jamais en fabriquer une.
	scan.QuantaOnly = true
	pos, err := ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions de bipede illisibles : %v", err)
	}
	sort.SliceStable(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	in.pos = pos
	if in.shots, err = ScanFilmFireEvents(dir); err != nil {
		t.Logf("tirs illisibles : %v", err)
	}
	recs, held, st, err := ScanFilmGameEntitiesChain(dir)
	if err != nil {
		t.Fatalf("chaine sequentielle impossible : %v", err)
	}
	in.player, in.held, in.chain = recs, held, st
	in.ends = pbLifeEnds(pos)
	return in
}

// pbLifeEnds decoupe les trajectoires en vies et rend leurs fins, par slot. MEME regle que
// `replay.buildLifeSpans` : un slot qui disparait plus de `pbLifeGapUS` puis revient est une
// NOUVELLE vie. Ce n'est pas un decodage, c'est un groupement — le porter ici ne duplique
// aucun lecteur de bits.
func pbLifeEnds(pos []BipedPosition) map[uint32][]uint64 {
	bySlot := map[uint32][]uint64{}
	for _, p := range pos {
		bySlot[p.Slot] = append(bySlot[p.Slot], p.TimestampUS)
	}
	out := map[uint32][]uint64{}
	for slot, ts := range bySlot {
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		last := ts[0]
		for _, t := range ts[1:] {
			if t-last > pbLifeGapUS {
				out[slot] = append(out[slot], last)
			}
			last = t
		}
		out[slot] = append(out[slot], last)
	}
	return out
}

func pbLogInputs(t *testing.T, in pbInputs) {
	t.Helper()
	vies := 0
	for _, e := range in.ends {
		vies += len(e)
	}
	t.Logf("FILM %s · positions %d · tirs %d · vies decoupees %d sur %d slots",
		in.short, len(in.pos), len(in.shots), vies, len(in.ends))
	t.Logf("CHAINE · paquets %d dont propres %d · records de bipede confirmes %d dont porteurs "+
		"d une identite d arme %d · echantillons d arme en main %d · records ti=5 %d",
		in.chain.Packets, in.chain.PacketsClean, in.chain.BipedRecords, in.chain.HeldWeaponReads,
		len(in.held), in.chain.PlayerRecords)
}
