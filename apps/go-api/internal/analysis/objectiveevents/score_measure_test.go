package objectiveevents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// score_measure_test.go — INSTRUMENT de la phase 0 du lot A de
// `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md` (items A.0.1 a A.0.4).
//
// Il ne publie RIEN et ne change AUCUN code de production : il confronte ce que le statborg
// du film dit (score de mode par slot d'equipe, frags/morts/assistances par slot de joueur)
// a l'oracle de la base, exporte une fois en TSV.
//
// # Deux gardes, et UN SEUL FILM par processus
//
//	SCORE_FILM    chemin ABSOLU du dossier de chunks d'UN film (.../film_chunks/<short8>)
//	SCORE_ORACLE  chemin du TSV exporte de `match_registry` (participants : meme nom + _participants)
//
// Un film par processus est une REGLE DE MACHINE, pas une commodite : le balayage accumule
// tous les enregistrements d'entite du film en memoire, et une boucle sur le corpus dans un
// meme processus a rendu la machine de l'utilisateur inutilisable deux fois en aout 2026
// (memoire `reference_statrecords_corpus_sweep_ram_bomb`, D17 du plan).
//
// Sortie : `<dossier de SCORE_ORACLE>/lotA/<short8>.tsv`, une ligne par mesure, premiere
// colonne = nature de la ligne (meta, team, a01, a02, teamcore, player, a03, volume).

const (
	// scoreFilmEnv porte le dossier de chunks du film a mesurer.
	scoreFilmEnv = "SCORE_FILM"
	// scoreOracleEnv porte le TSV de `match_registry` exporte en lecture seule.
	scoreOracleEnv = "SCORE_ORACLE"
)

// TestLotAPhase0Mesure mesure UN film et ecrit sa table de resultats.
func TestLotAPhase0Mesure(t *testing.T) {
	filmDir, oraclePath := os.Getenv(scoreFilmEnv), os.Getenv(scoreOracleEnv)
	if filmDir == "" || oraclePath == "" {
		t.Skipf("instrument non arme (%s=%q, %s=%q)", scoreFilmEnv, filmDir, scoreOracleEnv, oraclePath)
	}
	short := filepath.Base(filepath.Clean(filmDir))
	root := filepath.Dir(filepath.Dir(filepath.Clean(filmDir)))
	or := loadOracle(t, oraclePath, short)

	t.Setenv(filmCacheEnv, root)
	src, ok := newDiskFilmSource(t, short)
	if !ok {
		t.Fatalf("manifeste du film %s absent sous %s", short, root)
	}

	// UN SEUL decodage : toutes les mesures derivent de `recs` par les fonctions pures du
	// paquet. Rappeler ScoreCurve/SlotIdentity re-decoderait le film a chaque appel.
	start := time.Now()
	recs := StatRecords(src)
	decodeMS := time.Since(start).Milliseconds()
	if len(recs) == 0 {
		t.Fatalf("aucun enregistrement d'entite decode dans %s", filmDir)
	}

	w := &measureRows{}
	writeMeta(w, short, or, src, recs, decodeMS)
	teams := writeTeams(w, recs, or)
	ident, found := writePlayers(w, recs, or)
	writeIdentity(w, recs, or, teams, ident)
	writeRounds(w, recs, or)
	writeVolume(w, recs, ident)
	// Aucun triplet retrouve dans l'oracle = le mode ne porte peut-etre pas ces compteurs
	// a ces emplacements. La sonde le tranche au lieu de laisser une ignorance (A.0.3).
	if found == 0 {
		writeProbe(w, recs, or)
	}

	// Cout machine mesure DEDANS : la surveillance externe (Start-Process, PeakWorkingSet64)
	// echantillonne et peut manquer le pic ; `Sys` est le total reserve a l'OS depuis le
	// demarrage, donc une borne du pic qui ne depend d'aucun echantillonnage.
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	w.row("cost", decodeMS, time.Since(start).Milliseconds(),
		ms.Sys/(1<<20), ms.HeapAlloc/(1<<20), ms.TotalAlloc/(1<<20), len(recs))

	out := filepath.Join(filepath.Dir(oraclePath), "lotA", short+".tsv")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatalf("creation du dossier de sortie : %v", err)
	}
	if err := os.WriteFile(out, []byte(w.b.String()), 0o644); err != nil {
		t.Fatalf("ecriture de %s : %v", out, err)
	}
	t.Logf("mesure ecrite : %s (%d enregistrements, decodage %d ms)", out, len(recs), decodeMS)
}

// measureRows accumule les lignes TSV de sortie.
type measureRows struct{ b strings.Builder }

func (m *measureRows) row(kind string, vals ...any) {
	parts := make([]string, 0, len(vals)+1)
	parts = append(parts, kind)
	for _, v := range vals {
		parts = append(parts, fmt.Sprint(v))
	}
	m.b.WriteString(strings.Join(parts, "\t") + "\n")
}

// writeMeta ecrit l'identite du film et le cout du decodage.
func writeMeta(m *measureRows, short string, or oracleMatch, src FilmSource, recs []StatRecord, decodeMS int64) {
	tMin, tMax := recs[0].TimeMS, recs[0].TimeMS
	nTeam, nPlayer := 0, 0
	for _, r := range recs {
		if r.TimeMS < tMin {
			tMin = r.TimeMS
		}
		if r.TimeMS > tMax {
			tMax = r.TimeMS
		}
		if IsTeamSlot(r.Slot) {
			nTeam++
		} else {
			nPlayer++
		}
	}
	m.row("meta", short, or.Variant, ObjectiveTypeOf(or.Variant), or.MapName,
		or.Team0, or.Team1, or.DurationS, len(or.Lines), len(recs), nTeam, nPlayer,
		tMin, tMax, framePacketCount(src), decodeMS)
}

// framePacketCount compte les paquets FRAME du film — denominateur commun avec la voie
// « chaine » du controle D1, qui balaie les memes paquets.
func framePacketCount(src FilmSource) int {
	n := 0
	for _, meta := range src.Chunks() {
		raw, ok := src.ChunkData(meta.Index)
		if !ok {
			continue
		}
		n += len(walkFrames(decompressChunk(raw)))
	}
	return n
}

// writeTeams ecrit la courbe de score de chaque slot d'equipe et le verdict A.0.1. Il rend
// la derniere valeur par slot d'equipe.
//
// # Un slot d'equipe qui n'emet RIEN vaut zero, et ce n'est pas une devinette
//
// Le protocole ne reemet un composant que lorsqu'il CHANGE (statborg.go, en-tete). Une equipe
// qui ne marque pas de toute la partie n'emet donc jamais le score de mode : son absence EST
// son score. Mesure du 2026-08-17 : en CTF `530820e5` (oracle 3-0), le slot 6 n'a aucune
// emission et le slot 8 en a trois, une par capture. Le compte de slots emetteurs est publie
// a cote du verdict pour que la convention reste verifiable ligne par ligne.
func writeTeams(m *measureRows, recs []StatRecord, or oracleMatch) map[int]int64 {
	raw := collectComponent(recs, modeScoreComp, false)
	kept := keepMonotoneBySlot(raw)
	last := map[int]int64{}
	emitters := 0
	for _, slot := range []int{6, 8} {
		rawN, keptPts := 0, []ScorePoint{}
		for _, p := range raw {
			if p.Slot == slot {
				rawN++
			}
		}
		for _, p := range kept {
			if p.Slot == slot {
				keptPts = append(keptPts, p)
			}
		}
		if len(keptPts) == 0 {
			last[slot] = 0
			m.row("team", slot, 0, 0, rawN, "", "", "absent")
			continue
		}
		emitters++
		last[slot] = keptPts[len(keptPts)-1].Value
		m.row("team", slot, last[slot], len(keptPts), rawN,
			keptPts[0].TimeMS, keptPts[len(keptPts)-1].TimeMS, monotoneOf(keptPts))
	}
	m.row("a01", agreementOf(last, or), fmt.Sprintf("%d/%d", last[6], last[8]),
		fmt.Sprintf("%d/%d", or.Team0, or.Team1), emitters)
	return last
}

// monotoneOf dit si une suite est strictement croissante en valeur.
func monotoneOf(pts []ScorePoint) string {
	for i := 1; i < len(pts); i++ {
		if pts[i].Value <= pts[i-1].Value {
			return "non"
		}
	}
	return "oui"
}

// agreementOf compare l'ENSEMBLE des deux derniers scores de slot a l'ensemble
// {team_0_score, team_1_score} : l'accord ne suppose aucun etiquetage des equipes (D3).
func agreementOf(last map[int]int64, or oracleMatch) string {
	film := []int64{last[6], last[8]}
	oracle := []int64{int64(or.Team0), int64(or.Team1)}
	sort.Slice(film, func(i, j int) bool { return film[i] < film[j] })
	sort.Slice(oracle, func(i, j int) bool { return oracle[i] < oracle[j] })
	if film[0] == oracle[0] && film[1] == oracle[1] {
		return "exact"
	}
	return "ecart"
}

// writePlayers ecrit les compteurs par slot de joueur (A.0.3) et rend l'appariement retenu.
func writePlayers(m *measureRows, recs []StatRecord, or oracleMatch) (map[int]string, int) {
	kills := lastBySlot(recs, statSlotKey{coreKillsComp, sideA})
	deaths := lastBySlot(recs, statSlotKey{coreKillsComp, sideB})
	assists := lastBySlot(recs, statSlotKey{coreAssistsComp, sideA})

	triplet := slotIdentityFrom(recs, or.Lines)
	noncirc := identityByDeathsAssists(deaths, assists, or.Lines)

	byXUID := map[string]PlayerLine{}
	for _, l := range or.Lines {
		byXUID[l.XUID] = l
	}
	exactT, exactNC, found := 0, 0, 0
	for _, slot := range sortedSlots(kills) {
		xt, xn := triplet[slot], noncirc[slot]
		// Combien de lignes de match portent le triplet lu dans le film ? Zero = le film
		// compte FAUX. Deux ou plus = le film compte JUSTE mais l'identite est indecidable
		// (deux joueurs ont fini sur le meme triplet) : c'est un refus d'attribuer, pas une
		// erreur de compteur. Les deux cas sont distincts et se mesurent separement.
		nMatch := 0
		for _, l := range or.Lines {
			if kills[slot] == int64(l.Kills) && deaths[slot] == int64(l.Deaths) &&
				assists[slot] == int64(l.Assists) {
				nMatch++
			}
		}
		if nMatch >= 1 {
			found++
		}
		l, has := byXUID[xn]
		okNC := has && kills[slot] == int64(l.Kills) &&
			deaths[slot] == int64(l.Deaths) && assists[slot] == int64(l.Assists)
		if okNC {
			exactNC++
		}
		lt, hasT := byXUID[xt]
		okT := hasT && kills[slot] == int64(lt.Kills) &&
			deaths[slot] == int64(lt.Deaths) && assists[slot] == int64(lt.Assists)
		if okT {
			exactT++
		}
		m.row("player", slot, orNone(xt), orNone(xn),
			kills[slot], oracleOf(l, has, "k"), deaths[slot], oracleOf(l, has, "d"),
			assists[slot], oracleOf(l, has, "a"), boolFR(okNC), boolFR(okT), nMatch)
	}
	m.row("a03", exactNC, len(noncirc), exactT, len(triplet), len(kills), len(or.Lines), found)
	return triplet, found
}

// identityByDeathsAssists apparie un slot a un joueur sur le SEUL couple (morts,
// assistances), donc SANS l'ancre des frags.
//
// La circularite est documentee dans named.go : le nommage de `comp 2 A` a ete etabli en
// s'en servant d'ancre d'identite. Verifier ensuite « comp 2 A == kills » sur un appariement
// qui exige deja cette egalite ne prouve rien. Cet appariement-ci laisse les frags LIBRES,
// et le controle de `comp 2 A` devient une mesure. Le prix est une resolution plus faible :
// deux joueurs au meme couple (morts, assistances) ne sont apparies ni l'un ni l'autre.
func identityByDeathsAssists(deaths, assists map[int]int64, lines []PlayerLine) map[int]string {
	claim := map[int]string{}
	for slot := range deaths {
		var found string
		n := 0
		for _, l := range lines {
			if deaths[slot] == int64(l.Deaths) && assists[slot] == int64(l.Assists) {
				found, n = l.XUID, n+1
			}
		}
		if n == 1 {
			claim[slot] = found
		}
	}
	byXUID := map[string][]int{}
	for slot, xuid := range claim {
		byXUID[xuid] = append(byXUID[xuid], slot)
	}
	out := map[int]string{}
	for xuid, slots := range byXUID {
		if len(slots) == 1 {
			out[slots[0]] = xuid
		}
	}
	return out
}

// writeIdentity applique la cascade D3 : (a) par les scores finaux, (b) par les sommes de
// frags des slots joueurs identifies, (c) non resolu.
func writeIdentity(m *measureRows, recs []StatRecord, or oracleMatch, last map[int]int64, ident map[int]string) {
	// (a) — les deux scores oracle doivent differer, sinon l'etiquetage est ambigu.
	if or.Team0 != or.Team1 && agreementOf(last, or) == "exact" {
		t6 := 0
		if last[6] == int64(or.Team1) {
			t6 = 1
		}
		m.row("a02", "a", t6, 1-t6, "scores finaux distincts")
		return
	}
	// (b) — le slot d'equipe porte-t-il la somme des frags de son camp ?
	kills := lastBySlot(recs, statSlotKey{coreKillsComp, sideA})
	sum := map[int]int64{}
	for slot, xuid := range ident {
		if tid, ok := or.Teams[xuid]; ok {
			sum[tid] += kills[slot]
		}
	}
	c6 := lastValueOfSlot(recs, 6, statSlotKey{coreKillsComp, sideA})
	c8 := lastValueOfSlot(recs, 8, statSlotKey{coreKillsComp, sideA})
	detail := fmt.Sprintf("comp2A equipes %d/%d vs sommes joueurs %d/%d", c6, c8, sum[0], sum[1])
	if sum[0] != sum[1] && c6 != c8 &&
		((c6 == sum[0] && c8 == sum[1]) || (c6 == sum[1] && c8 == sum[0])) {
		t6 := 0
		if c6 == sum[1] {
			t6 = 1
		}
		m.row("a02", "b", t6, 1-t6, detail)
		return
	}
	m.row("teamcore", c6, c8, sum[0], sum[1], len(ident))
	m.row("a02", "c", "", "", detail)
}

// writeVolume mesure ce que la publication couterait (A.0.4) : les points reellement
// publies (aux CHANGEMENTS) et la taille JSON de la charge utile.
func writeVolume(m *measureRows, recs []StatRecord, ident map[int]string) {
	var doc struct {
		Teams   []volTeam   `json:"teams"`
		Players []volPlayer `json:"players"`
	}
	teamPts := 0
	for i, slot := range []int{6, 8} {
		pts := changesOnly(serieOfSlot(recs, slot, statSlotKey{modeScoreComp, sideA}, true))
		teamPts += len(pts)
		doc.Teams = append(doc.Teams, volTeam{TeamID: i, Points: asPoints(pts)})
	}
	personal := collectComponent(recs, personalScoreComp, true)
	nS, nK, nD, nA := 0, 0, 0, 0
	for _, slot := range sortedSlots(ident) {
		sc := changesOnly(filterSlot(personal, slot))
		ki := changesOnly(serieOfSlot(recs, slot, statSlotKey{coreKillsComp, sideA}, false))
		de := changesOnly(serieOfSlot(recs, slot, statSlotKey{coreKillsComp, sideB}, false))
		as := changesOnly(serieOfSlot(recs, slot, statSlotKey{coreAssistsComp, sideA}, false))
		nS, nK, nD, nA = nS+len(sc), nK+len(ki), nD+len(de), nA+len(as)
		doc.Players = append(doc.Players, volPlayer{XUID: ident[slot], Score: asPoints(sc),
			Kills: asPoints(ki), Deaths: asPoints(de), Assists: asPoints(as)})
	}
	full, _ := json.Marshal(doc)
	teamsOnly, _ := json.Marshal(doc.Teams)
	m.row("volume", teamPts, nS, nK, nD, nA, len(teamsOnly), len(full)-len(teamsOnly), len(full))
}

// serieOfSlot rend la suite chronologique d'un emplacement pour UN slot, debarrassee des
// ancrages parasites par le meme critere de plus longue sous-suite que la production
// (strict pour le score de mode, non decroissant pour un compteur).
func serieOfSlot(recs []StatRecord, slot int, key statSlotKey, strict bool) []ScorePoint {
	var pts []ScorePoint
	for _, r := range recs {
		if r.Slot != slot {
			continue
		}
		v, ok := r.Comps[key.Comp]
		if !ok {
			continue
		}
		val := v.A
		if key.Side == sideB {
			val = v.B
		}
		if val < 0 {
			continue
		}
		pts = append(pts, ScorePoint{TimeMS: r.TimeMS, Slot: slot, Value: val})
	}
	return longestRun(pts, strict)
}

// lastValueOfSlot rend la derniere valeur retenue d'un emplacement pour un slot, ou -1.
func lastValueOfSlot(recs []StatRecord, slot int, key statSlotKey) int64 {
	pts := serieOfSlot(recs, slot, key, false)
	if len(pts) == 0 {
		return -1
	}
	return pts[len(pts)-1].Value
}

// lastBySlot rend, par slot de JOUEUR, la derniere valeur retenue d'un emplacement.
func lastBySlot(recs []StatRecord, key statSlotKey) map[int]int64 {
	out := map[int]int64{}
	for slot, pts := range seriesBySlot(recs, key) {
		if len(pts) > 0 {
			out[slot] = pts[len(pts)-1].Value
		}
	}
	return out
}

// changesOnly ne garde que les emissions ou la valeur CHANGE — la forme publiee.
func changesOnly(pts []ScorePoint) []ScorePoint {
	out := pts[:0:0]
	for i, p := range pts {
		if i == 0 || p.Value != pts[i-1].Value {
			out = append(out, p)
		}
	}
	return out
}

// filterSlot restreint une suite a un slot.
func filterSlot(pts []ScorePoint, slot int) []ScorePoint {
	out := pts[:0:0]
	for _, p := range pts {
		if p.Slot == slot {
			out = append(out, p)
		}
	}
	return out
}

// volPoint / volTeam / volPlayer reproduisent la charge utile decrite en A.1.1 du plan :
// c'est ELLE qu'on pese, pas une approximation. Les instants sont en MILLISECONDES ici, alors
// que la publication les porterait en frames du document (entiers plus courts) : la mesure de
// volume MAJORE donc la taille reelle.
type volPoint struct {
	T int   `json:"t"`
	V int64 `json:"v"`
}

type volTeam struct {
	TeamID int        `json:"teamId"`
	Points []volPoint `json:"points"`
}

type volPlayer struct {
	XUID    string     `json:"xuid"`
	Score   []volPoint `json:"score"`
	Kills   []volPoint `json:"kills"`
	Deaths  []volPoint `json:"deaths"`
	Assists []volPoint `json:"assists"`
}

// asPoints convertit une suite d'emissions en points publiables.
func asPoints(pts []ScorePoint) []volPoint {
	out := make([]volPoint, 0, len(pts))
	for _, p := range pts {
		out = append(out, volPoint{T: p.TimeMS, V: p.Value})
	}
	return out
}

// sortedSlots rend les cles d'une table indexee par slot, triees.
func sortedSlots[V any](m map[int]V) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func orNone(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func boolFR(b bool) string {
	if b {
		return "oui"
	}
	return "non"
}

// oracleOf rend le compteur oracle demande, ou "-" si le joueur n'est pas apparie.
func oracleOf(l PlayerLine, has bool, which string) string {
	if !has {
		return "-"
	}
	switch which {
	case "k":
		return strconv.Itoa(l.Kills)
	case "d":
		return strconv.Itoa(l.Deaths)
	default:
		return strconv.Itoa(l.Assists)
	}
}
