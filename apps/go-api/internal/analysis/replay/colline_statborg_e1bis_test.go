package replay

// colline_statborg_e1bis_test.go — ETAPE E1-bis : LE COMPTEUR DE GARDE EST-IL DANS LE STATBORG ?
//
// POURQUOI CETTE ETAPE EXISTE (objection utilisateur du 2026-08-30). E1 a mesure un SEUIL de 43 s
// par reconstruction depuis le canal de propriete, et j'ai propose de l'ajuster pour que la jauge
// tombe juste a l'ecran. C'est caler une mesure sur un rendu, et l'utilisateur l'a refuse :
// « soit une equipe marque soit elle marque pas ». Il a raison, et il a designe la bonne piste —
// le statborg.
//
// CE QUI EST VERIFIE AVANT DE COMMENCER : les emplacements statborg de KOTH n'ont JAMAIS ete
// nommes. Le code le dit lui-meme (`objectiveevents/named.go` : « Un mode sans table (KOTH,
// Oddball) rend nil [...] les emplacements de `hill` et `ball` n'ont pas encore ete nommes : le
// balayage est le meme, c'est le corpus qui manque »). Le corpus, lui, existe : l'API donne PAR
// JOUEUR `StrongholdScoringTicks` et `StrongholdOccupationTime`, persistes en base, renseignes
// sur 100 % des joueurs des quatre films ci-dessous.
//
// LA METHODE A DEJA ABOUTI DEUX FOIS, a l'identique : VIP (`comp 22 A` reproduit
// `TimesSelectedAsVip` exactement par joueur, 3 films sur 3) et Oddball (`comp 0 A` =
// `skull_scoring_ticks`). On balaie les composants, on compare le total par entite a l'oracle.
//
// LE GATE EST ECRIT AVANT LA MESURE, dans `.ai/V7.5/PLAN_KOTH_GARDE_VIVANTE_2026-08-30.md`
// (E1b.3), et les valeurs d'oracle sont GELEES ci-dessous : elles ne peuvent pas bouger apres
// coup pour arranger un resultat.
//
// REGIME : garde `ZONE_FILM`, un film par processus, lecture seule, AUCUNE base ouverte.
//
//	$env:ZONE_FILM="<cache>/film_chunks/01e1f945"; go test ./internal/analysis/replay/ -run CollineStatborgE1Bis -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// e1bJoueur est la ligne d'oracle d'un joueur : ce que l'API dit de sa garde, et le triplet qui
// permettra (phase 2) de le rattacher a un slot du statborg.
type e1bJoueur struct {
	xuid                   string
	tics                   int
	secondes               int
	kills, deaths, assists int
	team                   int
}

// e1bOracle — RELEVE DU 2026-08-30 sur `match_objective_stats_latest` joint a
// `match_participants`, gele ici (le paquet `replay` n'ouvre aucune DuckDB, meme convention que
// `p2aCorpus` et `e1Cartes`). `tics` = StrongholdScoringTicks, `secondes` = StrongholdOccupationTime
// arrondi.
var e1bOracle = map[string][]e1bJoueur{
	"01e1f945": {
		{"2533274824966873", 33, 54, 18, 13, 4, 1},
		{"2533274832680942", 22, 40, 11, 18, 7, 0},
		{"2533274969015015", 34, 57, 13, 15, 7, 0},
		{"2535451337001599", 50, 69, 14, 15, 8, 0},
		{"2535455052477469", 33, 41, 11, 10, 7, 0},
		{"2535459782888916", 35, 52, 9, 11, 8, 1},
		{"2535462158176179", 22, 43, 23, 9, 6, 1},
		{"2535469190789936", 40, 56, 7, 14, 3, 1},
		// BOT : xuid corrompu en base, aucune ligne de match — donc aucun pont possible.
		{"bid(42.0", 3, 4, -1, -1, -1, -1},
	},
	"21ece4d8": {
		{"2533274818644345", 44, 70, 8, 11, 6, 1},
		{"2533274823110022", 59, 96, 18, 12, 6, 1},
		{"2533274937251227", 26, 42, 17, 12, 5, 0},
		{"2533274969128194", 23, 60, 8, 13, 6, 0},
		{"2535447289950713", 38, 55, 7, 12, 8, 1},
		{"2535449038390597", 30, 52, 13, 12, 7, 0},
		{"2535457719922537", 50, 93, 7, 8, 9, 0},
		{"2535471057307458", 89, 136, 11, 10, 6, 1},
	},
	"7f1bbf06": {
		{"2533274811165209", 20, 30, 8, 5, 9, 0},
		{"2533274823110022", 2, 8, 6, 9, 5, 1},
		{"2533274853201323", 23, 37, 5, 9, 2, 1},
		{"2533274933911910", 6, 17, 4, 9, 5, 1},
		{"2533274952587369", 0, 6, 8, 10, 3, 1},
		{"2535454710220286", 34, 42, 8, 7, 6, 0},
		{"2535454909563916", 94, 117, 10, 4, 4, 0},
		{"2683394777983413", 13, 16, 11, 7, 3, 0},
	},
	"a36c8bed": {
		{"2533274823110022", 37, 47, 10, 11, 4, 0},
		{"2533274923802103", 56, 67, 4, 11, 7, 1},
		{"2535438872399421", 4, 10, 9, 10, 4, 0},
		{"2535449697867729", 3, 9, 10, 8, 3, 1},
		{"2535454710220286", 84, 97, 11, 6, 5, 1},
		{"2535468370244850", 0, 0, 0, 3, 0, 0},
		{"2535468773663308", 7, 11, 13, 7, 2, 1},
		{"2535472706803453", 24, 25, 12, 8, 2, 0},
		{"bid(2.0", 12, 12, -1, -1, -1, -1},
	},
}

// e1bMaxComp borne le balayage. L'archetype de statistiques ne porte pas des centaines de
// composants ; 64 couvre tout ce que les lots precedents ont vu (le plus haut nomme est 22, VIP).
const e1bMaxComp = 64

// TestCollineStatborgE1Bis — LE BALAYAGE. Un film par processus.
func TestCollineStatborgE1Bis(t *testing.T) {
	dir := p2aRequireFilm(t)
	short := filepath.Base(dir)
	oracle, ok := e1bOracle[short]
	if !ok {
		t.Skipf("film %s hors corpus E1-bis (aucun oracle gele pour lui)", short)
	}

	recs := objectiveevents.StatRecords(p2aSource(t, dir))
	if len(recs) == 0 {
		t.Fatalf("%s : aucun enregistrement de statistiques — rien a balayer", short)
	}
	t.Logf("%s : %d enregistrements, %d manches retenues, oracle a %d joueurs",
		short, len(recs), len(objectiveevents.RealRounds(recs)), len(oracle))

	cibleTics := e1bMultiset(oracle, func(j e1bJoueur) int { return j.tics })
	cibleSec := e1bMultiset(oracle, func(j e1bJoueur) int { return j.secondes })
	t.Logf("%s : oracle TICS  %v", short, cibleTics)
	t.Logf("%s : oracle SECONDES %v", short, cibleSec)

	for comp := 0; comp <= e1bMaxComp; comp++ {
		for _, sideB := range []bool{false, true} {
			for _, strict := range []bool{false, true} {
				c := objectiveevents.StatComponent{Comp: comp, SideB: sideB, Strict: strict}
				totaux := e1bTotaux(recs, c)
				if len(totaux) == 0 {
					continue
				}
				nom := fmt.Sprintf("comp %d %s%s", comp, e1bSide(sideB), e1bStrict(strict))
				if e1bEgal(totaux, cibleTics) {
					t.Logf("E1BIS\t%s\tTICS\t%s\t%v", short, nom, totaux)
				}
				if e1bEgal(totaux, cibleSec) {
					t.Logf("E1BIS\t%s\tSECONDES\t%s\t%v", short, nom, totaux)
				}
			}
		}
	}
	// LE BALAYAGE PUBLIE AUSSI CE QU'IL A VU, meme quand rien ne colle : un negatif sans
	// denominateur ne se relit pas. On journalise les composants qui portent 8 valeurs (la
	// bonne cardinalite), c'est-a-dire les candidats plausibles ecartes par leur CONTENU.
	e1bCandidats(t, short, recs, len(oracle))
	e1bPhase2(t, short, recs, oracle)
}

// e1bPhase2 — LE DISCRIMINANT : la valeur JUSTE POUR CHAQUE JOUEUR NOMME.
//
// La phase 1 compare des ensembles, ce qui laisse deux failles que celle-ci ferme : un joueur a
// zero n'emet rien (son slot est absent de la serie, l'ensemble a une valeur de moins), et un
// participant peut n'avoir aucune ligne de match (les BOTS, dont le xuid est corrompu en base).
// Ici on ne compare plus des ensembles : on demande au composant la valeur d'un joueur NOMME,
// apres le pont slot -> xuid par le triplet frags/morts/assistances.
//
// C'est le discriminant qui a tranche VIP, et il est plus dur que la phase 1 : une permutation
// des slots le fait echouer alors qu'elle laisse l'ensemble intact.
func e1bPhase2(t *testing.T, short string, recs []objectiveevents.StatRecord, oracle []e1bJoueur) {
	t.Helper()
	lines := make([]objectiveevents.PlayerLine, 0, len(oracle))
	attendu := map[string]int{}
	for _, j := range oracle {
		attendu[j.xuid] = j.tics
		if j.kills < 0 {
			continue // bot sans ligne de match : aucun pont possible, et c'est dit
		}
		lines = append(lines, objectiveevents.PlayerLine{
			XUID: j.xuid, Kills: j.kills, Deaths: j.deaths, Assists: j.assists,
		})
	}
	identity := objectiveevents.SlotIdentityFrom(recs, lines)
	series := objectiveevents.SeriesTotal(recs,
		objectiveevents.StatComponent{Comp: 23, SideB: false}, false)

	justes, faux, sansSerie := 0, 0, 0
	for slot, xuid := range identity {
		pts := series[slot]
		if len(pts) == 0 {
			// Aucune emission : c'est la lecture « zero » d'un compteur.
			if attendu[xuid] == 0 {
				justes++
			} else {
				faux++
				t.Logf("E1BIS-P2\t%s\tslot %d (%s) : AUCUNE emission, oracle %d",
					short, slot, xuid, attendu[xuid])
			}
			sansSerie++
			continue
		}
		got := int(pts[len(pts)-1].Value)
		if got == attendu[xuid] {
			justes++
			continue
		}
		faux++
		t.Logf("E1BIS-P2\t%s\tslot %d (%s) : comp 23 A = %d, oracle = %d",
			short, slot, xuid, got, attendu[xuid])
	}
	t.Logf("E1BIS-P2\t%s\tcomp 23 A : %d juste(s) / %d slot(s) apparie(s) (%d sans emission), "+
		"%d ligne(s) de match fournie(s) sur %d participants",
		short, justes, justes+faux, sansSerie, len(lines), len(oracle))
	e1bParPoint(t, short, recs, oracle, identity, series)
}

// e1bParPoint — COMBIEN DE TICS VALENT UN POINT.
//
// Le compteur est PAR JOUEUR : quand un camp tient la colline, chacun de ses joueurs PRESENT
// prend un tic. La progression du camp n'est donc PAS la somme (elle dependrait du nombre de
// joueurs sur la colline) mais le MAXIMUM sur ses joueurs — le tic est le meme instant pour tous
// ceux qui y sont. Ce releve mesure ce maximum entre deux points, par camp qui marque.
//
// C'est un RELEVE, pas un gate : il dit si un denominateur EN TICS existe, et il ouvre (ou
// ferme) l'etape suivante.
func e1bParPoint(t *testing.T, short string, recs []objectiveevents.StatRecord, oracle []e1bJoueur,
	identity map[int]string, series map[int][]objectiveevents.ScorePoint,
) {
	t.Helper()
	team := map[string]int{}
	for _, j := range oracle {
		team[j.xuid] = j.team
	}
	score := objectiveevents.SeriesTotal(recs, objectiveevents.ModeScoreComponent, true)
	var pts []objectiveevents.ScorePoint
	slots := make([]int, 0, len(score))
	for s := range score {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	for _, s := range slots {
		var prev int64
		for _, p := range score[s] {
			if p.Value > prev {
				prev = p.Value
				pts = append(pts, objectiveevents.ScorePoint{TimeMS: p.TimeMS, Slot: s, Value: p.Value})
			}
		}
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
	// valeur d'un joueur a un instant : le dernier palier de sa serie de tics.
	valAt := func(slot, tMS int) int {
		var v int
		for _, p := range series[slot] {
			if p.TimeMS <= tMS {
				v = int(p.Value)
			}
		}
		return v
	}
	prevMS := 0
	for _, p := range pts {
		best := map[int]int{}
		for slot, xuid := range identity {
			d := valAt(slot, p.TimeMS) - valAt(slot, prevMS)
			if tm, ok := team[xuid]; ok && d > best[tm] {
				best[tm] = d
			}
		}
		union := e1cUnion(identity, team, series, prevMS, p.TimeMS)
		t.Logf("E1BIS-TICS\t%s\tpoint slot %d a %d ms : max par camp %v | UNION par camp %v",
			short, p.Slot, p.TimeMS, best, union)
		prevMS = p.TimeMS
	}
}

// e1cUnion — ETAPE E1-ter : LES TICS DU CAMP, ET NON CEUX D'UN JOUEUR.
//
// LE MAXIMUM SUR LA PERIODE SOUS-COMPTE, et la raison est dans le jeu : un camp peut tenir la
// colline EN SE RELAYANT, et alors aucun joueur n'accumule la totalite. Le releve E1-bis le
// montre (18 a 35 tics selon la periode, la ou la barre du jeu vaut toujours la meme chose).
//
// LA BONNE GRANDEUR EST L'UNION DES INSTANTS. Un tic est le meme instant pour tous les joueurs
// du camp presents sur la colline : a un instant donne, le camp avance de ce que son joueur le
// PLUS PRESENT a pris, pas de la somme (qui compterait deux fois le meme tic) ni du maximum sur
// toute la periode (qui perd les relais). On decoupe donc la periode aux instants d'emission, on
// prend le maximum PAR TRANCHE, et on somme — ce qui vaut exactement « le nombre de secondes ou
// au moins un joueur du camp a marque un tic », relais compris.
func e1cUnion(identity map[int]string, team map[string]int,
	series map[int][]objectiveevents.ScorePoint, deMS, aMS int,
) map[int]int {
	// Les instants d'emission de la fenetre, tous slots confondus, tries et dedoublonnes.
	vus := map[int]bool{}
	for slot := range identity {
		for _, p := range series[slot] {
			if p.TimeMS > deMS && p.TimeMS <= aMS {
				vus[p.TimeMS] = true
			}
		}
	}
	bornes := make([]int, 0, len(vus))
	for t := range vus {
		bornes = append(bornes, t)
	}
	sort.Ints(bornes)

	valAt := func(slot, tMS int) int {
		var v int
		for _, p := range series[slot] {
			if p.TimeMS <= tMS {
				v = int(p.Value)
			}
		}
		return v
	}
	out := map[int]int{}
	prev := deMS
	for _, b := range bornes {
		tranche := map[int]int{}
		for slot, xuid := range identity {
			tm, ok := team[xuid]
			if !ok {
				continue
			}
			if d := valAt(slot, b) - valAt(slot, prev); d > tranche[tm] {
				tranche[tm] = d
			}
		}
		for tm, d := range tranche {
			out[tm] += d
		}
		prev = b
	}
	return out
}

// e1bTotaux rend le total par slot de JOUEUR d'un composant : la derniere valeur cumulee de sa
// serie, manches sommees (`SeriesTotal` s'en charge). Les slots d'equipe sont exclus — l'oracle
// est par joueur.
func e1bTotaux(recs []objectiveevents.StatRecord, c objectiveevents.StatComponent) []int {
	series := objectiveevents.SeriesTotal(recs, c, false)
	out := make([]int, 0, len(series))
	for _, pts := range series {
		if len(pts) == 0 {
			continue
		}
		out = append(out, int(pts[len(pts)-1].Value))
	}
	sort.Ints(out)
	return out
}

// e1bMultiset rend les valeurs de l'oracle, triees — la forme comparable aux totaux.
func e1bMultiset(oracle []e1bJoueur, of func(e1bJoueur) int) []int {
	out := make([]int, 0, len(oracle))
	for _, j := range oracle {
		out = append(out, of(j))
	}
	sort.Ints(out)
	return out
}

func e1bEgal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// e1bCandidats journalise les composants de la BONNE CARDINALITE, ecartes par leur contenu. Sans
// eux, un negatif dirait « rien ne colle » sans dire ce qui a ete regarde.
func e1bCandidats(t *testing.T, short string, recs []objectiveevents.StatRecord, n int) {
	t.Helper()
	vus := 0
	for comp := 0; comp <= e1bMaxComp; comp++ {
		for _, sideB := range []bool{false, true} {
			c := objectiveevents.StatComponent{Comp: comp, SideB: sideB}
			totaux := e1bTotaux(recs, c)
			if len(totaux) != n {
				continue
			}
			vus++
			t.Logf("E1BIS-CAND\t%s\tcomp %d %s\t%v", short, comp, e1bSide(sideB), totaux)
		}
	}
	t.Logf("%s : %d composant(s) a %d valeurs balayes", short, vus, n)
}

func e1bSide(b bool) string {
	if b {
		return "B"
	}
	return "A"
}

func e1bStrict(b bool) string {
	if b {
		return " (strict)"
	}
	return ""
}
