package killcollector

// duels_bouclier_mesures_test.go — les mesures A, O, B, temoin et discrimination de la sonde
// n°2 (scinde de duels_bouclier_research_test.go pour le seuil de 500 lignes du depot). Voir
// l'en-tete de ce fichier-la pour la question posee, le gate et les seuils ecrits AVANT la
// mesure.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
)

// duelsBChute : une baisse de bouclier, datee, rattachee a un JOUEUR (et plus a un slot).
//
// LE RATTACHEMENT SE FAIT ICI ET PAS PLUS TARD, et c'est ce qui distingue cette sonde de la
// n°1 : celle-la mesurait des chutes par SLOT et ne pouvait donc pas dire « le bouclier DU
// TUEUR », faute de savoir qui occupait le slot. Le pont `ResolveSlotXUID` le donne.
type duelsBChute struct {
	xuid uint64
	ts   uint64
}

// duelsBSonde porte les entrees d'une mesure. Une structure et pas six parametres : le depot
// plafonne a 5 parametres, et ces six quantites voyagent toujours ensemble.
type duelsBSonde struct {
	kills     []duelsBKill
	positions []filmdec.BipedPosition
	slotXUID  map[uint32]uint64
	equipes   map[uint64]int64
	originUS  int64
}

// duelsBBilan : les comptes bruts d'un match. TOUS les taux du plan se recalculent depuis ces
// entiers — c'est ce qui rend le CUMUL possible (les pourcentages ne s'additionnent pas).
type duelsBBilan struct {
	kills, pont, a, o, b  int
	eligibles, bElig, tem int
	discrimine            int
}

// duelsBMesurer produit les cinq mesures et le verdict du gate.
func duelsBMesurer(t *testing.T, s duelsBSonde) {
	t.Helper()
	chutes, premiere := duelsBChutesBouclier(s.positions, s.slotXUID)
	t.Logf("DENSITE : %d chutes de bouclier rattachees a un joueur pour %d kills du feed "+
		"(%.1f par kill) — premiere lecture de bouclier a %d us",
		len(chutes), len(s.kills), float64(len(chutes))/float64(max(1, len(s.kills))), premiere)

	bilan := duelsBComptes(s, chutes, premiere)
	duelsBPublier(t, bilan)
	duelsBVerdict(t, bilan)
}

// duelsBComptes parcourt les kills UNE fois et remplit tous les compteurs.
//
// `premiere` est l'instant de la PREMIERE lecture de bouclier du film. Il sert a l'ELIGIBILITE
// DU TEMOIN, et cette borne n'est pas cosmetique : reculer la fenetre de 37 s pour un kill
// survenu dans les 39 premieres secondes la place AVANT le debut des lectures, ou personne ne
// peut chuter. Compter ces kills au denominateur du temoin le ferait tomber vers zero par
// construction, et le rapport B/temoin serait un artefact de bord, pas une specificite mesuree.
func duelsBComptes(s duelsBSonde, chutes []duelsBChute, premiere uint64) duelsBBilan {
	bilan := duelsBBilan{kills: len(s.kills)}
	localise := duelsBTueursLocalises(s)
	slots := duelsBSlotsParXUID(s.slotXUID)
	decale := premiere + duelsBShiftUS + duelsBWindowUS

	for i, k := range s.kills {
		tUS := uint64(k.timeMS*1000 + s.originUS)
		lo := duelsBBorneBasse(tUS, duelsBWindowUS)
		if len(slots[k.killer]) > 0 {
			bilan.pont++
		}
		if localise[i] {
			bilan.a++
		}
		if duelsBChuteDe(chutes, k.victim, lo, tUS) {
			bilan.o++
		}
		tueurEnChute := duelsBChuteDe(chutes, k.killer, lo, tUS)
		if tueurEnChute {
			bilan.b++
			if duelsBVictimeSeule(chutes, s.equipes, k, [2]uint64{lo, tUS}) {
				bilan.discrimine++
			}
		}
		if tUS >= decale {
			bilan.eligibles++
			if tueurEnChute {
				bilan.bElig++
			}
			hi := tUS - duelsBShiftUS
			if duelsBChuteDe(chutes, k.killer, duelsBBorneBasse(hi, duelsBWindowUS), hi) {
				bilan.tem++
			}
		}
	}
	return bilan
}

// duelsBTueursLocalises rend, pour chaque kill, si le TUEUR est localise a T PAR LE PONT DE
// PRODUCTION. La mesure A passe par `BuildKillPositions` elle-meme, et pas par une resolution
// locale : c'est la fonction que le lot 7 utiliserait, donc c'est elle qu'il faut mesurer.
//
// Le decalage d'horloge est `originUS` (`ScanClockOrigin`), exactement comme dans
// `buildPositionRows` (positions.go) — aucun calage reinvente ici.
func duelsBTueursLocalises(s duelsBSonde) []bool {
	refs := make([]replay.KillRef, len(s.kills))
	for i, k := range s.kills {
		refs[i] = replay.KillRef{KillerXUID: k.killer, VictimXUID: k.victim, TimeMS: k.timeMS}
	}
	// BuildKillPositions n'emet PAS les morts dont aucun des deux n'est localise : le resultat
	// se relit par couple (tueur, victime, instant), jamais par indice.
	posOut, _ := replay.BuildKillPositions(s.positions, s.slotXUID, refs, s.originUS)
	localise := make(map[replay.KillRef]bool, len(posOut))
	for i := range posOut {
		if posOut[i].Killer != nil {
			localise[posOut[i].KillRef] = true
		}
	}
	out := make([]bool, len(refs))
	for i, r := range refs {
		out[i] = localise[r]
	}
	return out
}

// duelsBPublier ecrit les cinq mesures, numerateur ET denominateur, plus la ligne CUMUL qui
// porte les entiers bruts — c'est elle qui permet d'additionner quatre matchs sans additionner
// des pourcentages.
func duelsBPublier(t *testing.T, b duelsBBilan) {
	t.Helper()
	t.Logf("A pont du tueur (BuildKillPositions localise le tueur a T) : %s", duelsBPct(b.a, b.kills))
	t.Logf("A' diagnostic — le tueur a au moins un slot au pont (sans exigence de position) : %s",
		duelsBPct(b.pont, b.kills))
	t.Logf("O oracle victime (chute du bouclier de la VICTIME dans [T-2 s, T]) : %s",
		duelsBPct(b.o, b.kills))
	t.Logf("B bouclier du TUEUR dans [T-2 s, T] : %s", duelsBPct(b.b, b.kills))
	t.Logf("T temoin (meme mesure, fenetre reculee de %d s) : %s — B sur la MEME population : %s",
		duelsBShiftUS/1_000_000, duelsBPct(b.tem, b.eligibles), duelsBPct(b.bElig, b.eligibles))
	t.Logf("D discrimination (la victime est le SEUL adversaire en chute dans la fenetre) : %s",
		duelsBPct(b.discrimine, b.b))
	t.Logf("CUMUL kills=%d pont=%d A=%d O=%d B=%d eligibles=%d Belig=%d T=%d D=%d",
		b.kills, b.pont, b.a, b.o, b.b, b.eligibles, b.bElig, b.tem, b.discrimine)
}

// duelsBVerdict confronte les mesures au gate du plan. IL NE MODIFIE AUCUN SEUIL : les quatre
// valeurs sont celles ecrites dans `.ai/PLAN_DUELS_PORTEE_2026-09-06.md` avant la mesure.
//
// Il n'echoue PAS le test sur un NO-GO : une sonde qui plante ne rend pas ses nombres, et ce
// sont les nombres qui decident. Le verdict est JOURNALISE, statue dans la note, et le plan
// tranche.
func duelsBVerdict(t *testing.T, b duelsBBilan) {
	t.Helper()
	tauxA := duelsBTaux(b.a, b.kills)
	ratioBO := duelsBTaux(b.b, b.o)
	ratioBT := duelsBTaux(b.bElig, b.tem)
	tauxD := duelsBTaux(b.discrimine, b.b)

	okA := b.kills > 0 && tauxA >= 0.80
	okBO := b.o > 0 && ratioBO >= 0.35 && ratioBO <= 0.90
	// Un temoin NUL est la specificite parfaite : le rapport est infini, donc au-dessus de 3.
	okBT := b.tem == 0 && b.bElig > 0 || b.tem > 0 && ratioBT >= 3
	okD := b.b > 0 && tauxD >= 0.60

	t.Logf("GATE A >= 80 %% : %.1f %% -> %s", 100*tauxA, duelsBOui(okA))
	t.Logf("GATE B/O dans [0,35 ; 0,90] : %s -> %s", duelsBRatio(b.b, b.o), duelsBOui(okBO))
	t.Logf("GATE B/temoin >= 3 : %s -> %s", duelsBRatio(b.bElig, b.tem), duelsBOui(okBT))
	t.Logf("GATE D >= 60 %% : %.1f %% -> %s", 100*tauxD, duelsBOui(okD))
	t.Logf("VERDICT DE CE MATCH : %s (le verdict qui DECIDE du lot 7 est celui du CUMUL des "+
		"quatre matchs, pas celui d'un match isole)", duelsBGoNoGo(okA && okBO && okBT && okD))
}

func duelsBTaux(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func duelsBRatio(n, d int) string {
	if d == 0 {
		return fmt.Sprintf("%d/0 (temoin nul)", n)
	}
	return fmt.Sprintf("%d/%d = %.2f", n, d, float64(n)/float64(d))
}

func duelsBOui(ok bool) string {
	if ok {
		return "PASSE"
	}
	return "ECHOUE"
}

func duelsBGoNoGo(ok bool) string {
	if ok {
		return "GO"
	}
	return "NO-GO"
}

// duelsBChutesBouclier rend les baisses de bouclier RATTACHEES A UN JOUEUR.
//
// La detection est celle de la sonde n°1 (`duelsChutesBouclier`, analysis/replay) : par slot,
// lectures triees, une chute est une baisse superieure a `duelsBChuteEps` entre deux lectures
// CONSECUTIVES separees de moins de `duelsBLifeGapUS` (au-dela, c'est une nouvelle vie et le
// bouclier repart plein — la comparaison serait une fausse chute). La transposition ici est
// necessaire : ces fonctions ne sont pas exportees, et les mesures de la n°1 restent la
// reference a laquelle celles-ci doivent pouvoir se comparer.
//
// Un slot ABSENT du pont est ignore : sa chute est reelle mais anonyme, et une chute qu'on ne
// sait pas attribuer ne peut ni confirmer ni infirmer un duel. Cette perte est exactement ce
// que la mesure A quantifie.
// Rend AUSSI l'instant de la premiere lecture retenue : c'est le bord gauche du domaine ou une
// chute est OBSERVABLE, et le temoin decale doit tomber a l'interieur (cf. duelsBComptes).
func duelsBChutesBouclier(
	positions []filmdec.BipedPosition, slotXUID map[uint32]uint64,
) ([]duelsBChute, uint64) {
	parSlot := map[uint32][]filmdec.BipedPosition{}
	premiere := uint64(0)
	for _, p := range positions {
		if _, ok := slotXUID[p.Slot]; !ok {
			continue
		}
		if _, ok := p.ShieldAt(); ok {
			parSlot[p.Slot] = append(parSlot[p.Slot], p)
			if premiere == 0 || p.TimestampUS < premiere {
				premiere = p.TimestampUS
			}
		}
	}
	var out []duelsBChute
	for slot, ps := range parSlot {
		sort.Slice(ps, func(i, j int) bool { return ps[i].TimestampUS < ps[j].TimestampUS })
		prec, precTS, ouvert := float32(0), uint64(0), false
		for _, p := range ps {
			v, _ := p.ShieldAt()
			if ouvert && p.TimestampUS-precTS <= duelsBLifeGapUS && float64(prec-v) > duelsBChuteEps {
				out = append(out, duelsBChute{xuid: slotXUID[slot], ts: p.TimestampUS})
			}
			prec, precTS, ouvert = v, p.TimestampUS, true
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out, premiere
}

// duelsBChuteDe dit si le joueur a une chute de bouclier dans [lo, hi].
func duelsBChuteDe(chutes []duelsBChute, xuid, lo, hi uint64) bool {
	for _, c := range chutes {
		if c.ts > hi {
			break // le flux est trie
		}
		if c.ts >= lo && c.xuid == xuid {
			return true
		}
	}
	return false
}

// duelsBVictimeSeule dit si la VICTIME est le seul ADVERSAIRE du tueur a avoir perdu du
// bouclier dans la fenetre.
//
// ADVERSAIRE = equipe connue des deux cotes ET differente. Un joueur dont l'equipe est inconnue
// n'est ni compte comme candidat ni ignore en silence : il ne peut pas etre candidat, et c'est
// dit dans la note. « Vivant » n'a pas besoin d'etre teste separement : une chute de bouclier
// EST une lecture de position, donc le slot etait replique — un mort n'en produit pas.
//
// `fenetre` est le couple [lo, hi] : deux bornes qui ne voyagent jamais l'une sans l'autre, et
// le depot plafonne a 5 parametres.
func duelsBVictimeSeule(
	chutes []duelsBChute, equipes map[uint64]int64, k duelsBKill, fenetre [2]uint64,
) bool {
	campTueur, ok := equipes[k.killer]
	if !ok {
		return false // camp du tueur inconnu : aucun adversaire n'est definissable
	}
	candidats := map[uint64]bool{}
	for _, c := range chutes {
		if c.ts > fenetre[1] {
			break
		}
		if c.ts < fenetre[0] {
			continue
		}
		if camp, connu := equipes[c.xuid]; connu && camp != campTueur {
			candidats[c.xuid] = true
		}
	}
	return len(candidats) == 1 && candidats[k.victim]
}

// duelsBSlotsParXUID inverse le pont : un joueur occupe plusieurs slots au cours d'un match.
func duelsBSlotsParXUID(slotXUID map[uint32]uint64) map[uint64][]uint32 {
	out := map[uint64][]uint32{}
	for slot, x := range slotXUID {
		out[x] = append(out[x], slot)
	}
	return out
}

// duelsBBorneBasse borne la fenetre a zero — l'horloge du film ne descend pas sous son origine.
func duelsBBorneBasse(ts, w uint64) uint64 {
	if ts > w {
		return ts - w
	}
	return 0
}
