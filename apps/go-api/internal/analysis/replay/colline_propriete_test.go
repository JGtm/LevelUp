package replay

// colline_propriete_test.go — LOT C-ter VOLET 1, CT.1.1 : LA SERIE DES VALEURS DE CHAQUE TAG DE
// `ti=13`, PAR SLOT, SUR UN FILM KOTH — pas seulement la rampe.
//
// CE QUE CE FICHIER FAIT. Il balaye `ti=13` par le chemin de PRODUCTION
// (`filmdec.ScanFilmManagedProperties`, zero copie de grammaire), pose chaque lecture sur les
// deux axes de temps du rejeu (ms depuis le premier paquet du film ; frame du rejeu), et ecrit la
// SERIE COMPLETE des valeurs par (slot, mode, tag) — mode A : les tags du variant scalaire i1 ;
// mode B : les tags des 32 variants par joueur i2..i33, avec l'index de film du joueur — dans un
// TSV par film sous `.ai/V7.5/replay2d/registre_film/lotCter/`, plus un RESUME par (slot, mode,
// tag) : emissions, valeurs distinctes, changements, premiere et derniere emission.
//
// POURQUOI. La phase 2a du lot C-bis n'a exploite que la rampe (tag 3) et le proprietaire
// (tag 4) ; en KOTH la colline ACTIVE est DEDUITE du mouvement de la jauge (a-coups de 4-5 s,
// colline vide invisible, delai d'activation au depart inconnu). Le volet 1 du lot C-ter cherche
// si un AUTRE tag porte l'activation de la colline : il faut d'abord VOIR ce que chaque tag
// emet, quand, et combien. La mesure contre l'oracle est l'objet de CT.1.2 (second instrument).
//
// RIEN N'EST PUBLIE : aucun champ de document, aucun schema. Instrument de mesure, sous garde.
//
// USAGE (depuis apps/go-api, UN film par processus, avant-plan — D17) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/01e1f945"
//	go test -count=1 -run TestCollineProprieteTemoin -v -timeout 30m ./internal/analysis/replay/

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// ctOutDirName est le sous-dossier de sortie du lot C-ter sous `registre_film/`.
const ctOutDirName = "lotCter"

// ctLecture est UNE lecture de `ti=13`, posee sur les deux axes de temps du rejeu.
type ctLecture struct {
	slot  uint32
	modeA bool
	tag   int
	// film est l'index de film du joueur (mode B, 0..31) ; -1 en mode A. Un ORDRE interne au
	// film, jamais une identite (garde-rail archlint) : il ne sert qu'a separer les 32 series.
	film int
	// tMS : ms depuis le PREMIER PAQUET DU FILM (l'horloge des evenements nommes et du score).
	tMS int64
	// frame : index de frame du rejeu, -1 hors de l'axe publie.
	frame int
	value uint64
	has   bool
	// chained : le record porteur est CHAINE (temoin de fiabilite par lecture du balayage) ;
	// strict : le slot est vu porter ti=13, et lui seul, aux images-cles (bande STRICTE, celle
	// de la phase 2a) — la bande de production COMBLE les trous, ou seule la contamination parle.
	chained, strict bool
}

// ctEntree regroupe ce que les deux instruments du volet partagent (regle des 5 parametres).
type ctEntree struct {
	dir, short string
	film       p2aFilm
	sc         filmdec.ManagedPropertyScan
	doc        ReplayDocument
	zones      []Zone
	// clockUS : premier paquet du film (zero de l'horloge des evenements) ; posUS : premier
	// paquet de position (zero des frames). Leur difference est `doc.OriginMs`.
	clockUS, posUS uint64
	// bande est la bande STRICTE des slots ti=13 (vus aux images-cles, purges des ambigus).
	bande    map[uint32]bool
	lectures []ctLecture
}

// TestCollineProprieteTemoin ecrit la serie de chaque tag de ti=13 d'UN film KOTH (CT.1.1).
func TestCollineProprieteTemoin(t *testing.T) {
	e := ctCharge(t)
	out := ctOutDir(t)
	ctLogEntete(t, e)
	ctEcritSeries(t, out, e)
	res := ctResume(e.lectures)
	ctLogResume(t, res)
	ctEcritResume(t, out, e.short, res)
}

// ctLogEntete publie le film, le balayage et ses temoins d'ancrage.
func ctLogEntete(t *testing.T, e ctEntree) {
	t.Helper()
	chainees, strictes, strictesChainees := 0, 0, 0
	for _, l := range e.lectures {
		if l.chained {
			chainees++
		}
		if l.strict {
			strictes++
			if l.chained {
				strictesChainees++
			}
		}
	}
	t.Logf("FILM %s (%s, %s) — %d lectures ti=13 sur %d slots de bande comblee (%d en bande"+
		" stricte) · %d records ancres, %d marches, %d chainees (%.1f %%) · lectures chainees %d"+
		" (%.1f %%), en bande stricte %d dont %d chainees (%.1f %%) · rejeu %d frames de %d ms,"+
		" origine %s · film %d zones au catalogue",
		e.short, e.film.Mode, e.film.Carte, len(e.lectures), e.sc.Slots, len(e.bande), e.sc.Records,
		e.sc.Walked, e.sc.Chained, 100*p2aRate(e.sc.Chained, e.sc.Walked), chainees,
		100*p2aRate(chainees, len(e.lectures)), strictes, strictesChainees,
		100*p2aRate(strictesChainees, strictes), e.doc.FrameCount, e.doc.FrameIntervalMS,
		p2aOrigine(e.doc), len(e.zones))
}

// ctCharge balaye le film et assemble tout ce que les instruments du volet consomment.
func ctCharge(t *testing.T) ctEntree {
	t.Helper()
	dir := p2aRequireFilm(t)
	short, film := p2aFilmOf(t, dir)
	if film.Mode != "KOTH" {
		t.Skipf("film %s (%s) : le lot C-ter ne mesure que les KOTH", short, film.Mode)
	}
	e := ctEntree{dir: dir, short: short, film: film}
	clockUS, err := ScanFilmClockOrigin(dir)
	if err != nil {
		t.Fatalf("origine d'horloge illisible (%s) : %v", dir, err)
	}
	e.clockUS = clockUS
	e.sc = p2bScan(t, dir)
	e.bande = p2aBande(dir)
	e.zones = ctZones(t, film.MapID)
	quant := p2aQuant(t, film.Carte)
	e.doc, e.posUS = p2bBuild(t, dir, short, quant, ZoneInput{
		Scanned: true, Reads: e.sc.Reads, Zones: e.zones, Roles: p2bRoles(film),
		TeamByXUID: film.p2aTeams(), Hill: true,
	}, nil)
	e.lectures = ctLectures(e)
	return e
}

// ctZones rend les zones surfaciques de la carte, ou nil (et non un skip) quand la carte est
// absente du catalogue : le volet 1 mesure une PROPRIETE du film, la carte n'en est pas la
// condition — seul l'appariement a une forme en depend, et il est publie comme absent.
func ctZones(t *testing.T, mapID string) []Zone {
	t.Helper()
	cat, err := LoadMapObjectives(filepath.Join(p2aRefDir(t), "map_objectives.json"))
	if err != nil {
		t.Fatalf("catalogue d'objectifs illisible : %v", err)
	}
	entry, err := cat.Lookup(mapID)
	if err != nil {
		t.Logf("carte %s ABSENTE du catalogue de formes : aucune zone (appariement impossible)", mapID)
		return nil
	}
	var out []Zone
	for _, r := range p2aRolesZones {
		out = append(out, entry.ZonesOfRole(r).Zones...)
	}
	sortZonesSpatially(out)
	return out
}

// ctLectures pose chaque lecture du balayage sur les deux axes de temps, dans l'ordre du flux.
func ctLectures(e ctEntree) []ctLecture {
	c := zoneCtx{origin: e.posUS, step: uint64(e.doc.FrameIntervalMS) * 1000, frames: e.doc.FrameCount}
	out := make([]ctLecture, 0, len(e.sc.Reads))
	for _, r := range e.sc.Reads {
		l := ctLecture{slot: r.Slot, modeA: r.Field == filmdec.ManagedPropertyScalar, tag: r.Tag,
			film: r.FilmIndex, value: r.Value, has: r.HasValue, frame: -1, chained: r.Chained,
			strict: e.bande[r.Slot]}
		l.tMS = (int64(r.TimestampUS) - int64(e.clockUS)) / 1000
		if f, ok := zoneFrameOf(r.TimestampUS, c); ok {
			l.frame = f
		}
		out = append(out, l)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	return out
}

// ctOutDir rend le dossier de sortie du lot (`registre_film/lotCter/`), cree au besoin.
func ctOutDir(t *testing.T) string {
	t.Helper()
	if v := os.Getenv(p2aOutEnv); v != "" {
		p2aMkdir(t, v)
		return v
	}
	out := filepath.Join(repoRootForTest(t), ".ai", "V7.5", "replay2d", "registre_film", ctOutDirName)
	p2aMkdir(t, out)
	return out
}

// ctBit rend 1 ou 0.
func ctBit(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ctMode rend l'etiquette du mode d'une lecture.
func ctMode(modeA bool) string {
	if modeA {
		return "A"
	}
	return "B"
}

// ctValeurLue rend la valeur DECODEE d'une lecture selon le type de son tag — la convention des
// convertisseurs exportes de `filmdec`, jamais une interpretation de plus.
func ctValeurLue(l ctLecture) string {
	if !l.has {
		return "-"
	}
	switch l.tag {
	case filmdec.ManagedPropertyTagQuant, filmdec.ManagedPropertyTagQuantJ:
		return fmt.Sprintf("%.5f", filmdec.ManagedPropertyQuantValue(l.value))
	case filmdec.ManagedPropertyTagBool, filmdec.ManagedPropertyTagBoolJ:
		return fmt.Sprintf("%d", l.value)
	case filmdec.ManagedPropertyTagEnum:
		return fmt.Sprintf("%d", filmdec.ManagedPropertyEnumValue(l.value))
	}
	if l.tag >= filmdec.ManagedPropertyTagEnumJ {
		return fmt.Sprintf("%d", filmdec.ManagedPropertyEnumValue(l.value))
	}
	return fmt.Sprintf("0x%08X", l.value)
}

// ctEcritSeries ecrit la SERIE complete : une ligne par lecture, dans l'ordre du temps.
func ctEcritSeries(t *testing.T, out string, e ctEntree) {
	t.Helper()
	var sb strings.Builder
	fmt.Fprintf(&sb, "# lot C-ter volet 1 — CT.1.1 — film %s (%s / %s) — %d lectures ti=13\n",
		e.short, e.film.Mode, e.film.Carte, len(e.lectures))
	fmt.Fprintf(&sb, "# t_ms = ms depuis le premier paquet du film ; t_frame = frame du rejeu"+
		" (origine %s, pas %d ms), -1 hors axe ; film_index = -1 en mode A ; chaine = le record"+
		" porteur chaine (1) ; bande = stricte (1) ou comblee (0)\n",
		p2aOrigine(e.doc), e.doc.FrameIntervalMS)
	sb.WriteString("slot\tmode\ttag\tfilm_index\tt_ms\tt_frame\tvaleur_brute\tvaleur_lue\tchaine\tbande\n")
	for _, l := range e.lectures {
		brute := "-"
		if l.has {
			brute = fmt.Sprintf("%d", l.value)
		}
		fmt.Fprintf(&sb, "%d\t%s\t%d\t%d\t%d\t%d\t%s\t%s\t%d\t%d\n", l.slot, ctMode(l.modeA), l.tag,
			l.film, l.tMS, l.frame, brute, ctValeurLue(l), ctBit(l.chained), ctBit(l.strict))
	}
	p2aWrite(t, out, e.short+"_ti13_series.tsv", sb.String())
	t.Logf("  ecrit : %s (%d lignes)", filepath.Join(out, e.short+"_ti13_series.tsv"), len(e.lectures))
}

// ctCle identifie une serie : (slot, mode, tag).
type ctCle struct {
	slot  uint32
	modeA bool
	tag   int
}

// ctResumeLigne est le resume d'une serie, calcule sur les lectures CHAINEES ; les lectures non
// chainees sont comptees a cote, jamais melangees.
type ctResumeLigne struct {
	cle ctCle
	// emissions / avecValeur : lectures chainees, dont celles ou la branche du tag a lu quelque
	// chose ; nonChainees : lectures ecartees (record non chaine) ; strict : le slot est dans
	// la bande stricte.
	emissions, avecValeur, nonChainees int
	strict                             bool
	// distinctes : valeurs distinctes ; changements : lectures dont la valeur differe de la
	// precedente DE LA MEME SERIE (mode B : de la meme serie du meme joueur) ; joueurs : nombre
	// d'index de film distincts (mode B).
	distinctes, changements, joueurs int
	tMSPremiere, tMSDerniere         int64
	framePremiere, frameDerniere     int
	valeurs                          map[uint64]int
}

// ctResume agrege la serie de chaque (slot, mode, tag).
func ctResume(ls []ctLecture) []ctResumeLigne {
	acc := map[ctCle]*ctResumeLigne{}
	// prev : derniere valeur vue par (cle, joueur) — le changement se juge dans SA serie.
	type serie struct {
		cle  ctCle
		film int
	}
	prev := map[serie]uint64{}
	joueurs := map[ctCle]map[int]bool{}
	for _, l := range ls {
		k := ctCle{slot: l.slot, modeA: l.modeA, tag: l.tag}
		r, ok := acc[k]
		if !ok {
			r = &ctResumeLigne{cle: k, valeurs: map[uint64]int{}, strict: l.strict,
				framePremiere: -1, frameDerniere: -1}
			acc[k] = r
			joueurs[k] = map[int]bool{}
		}
		if !l.chained {
			r.nonChainees++
			continue
		}
		if r.emissions == 0 {
			r.tMSPremiere, r.framePremiere = l.tMS, l.frame
		}
		r.emissions++
		r.tMSDerniere, r.frameDerniere = l.tMS, l.frame
		joueurs[k][l.film] = true
		if !l.has {
			continue
		}
		r.avecValeur++
		r.valeurs[l.value]++
		s := serie{cle: k, film: l.film}
		if p, seen := prev[s]; seen && p != l.value {
			r.changements++
		}
		prev[s] = l.value
	}
	out := make([]ctResumeLigne, 0, len(acc))
	for k, r := range acc {
		r.distinctes, r.joueurs = len(r.valeurs), len(joueurs[k])
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].cle.slot != out[j].cle.slot {
			return out[i].cle.slot < out[j].cle.slot
		}
		if out[i].cle.modeA != out[j].cle.modeA {
			return out[i].cle.modeA
		}
		return out[i].cle.tag < out[j].cle.tag
	})
	return out
}

// ctLogResume publie le resume, une ligne par serie CHAINEE (au moins une lecture chainee) ; les
// series entierement non chainees sont comptees en une ligne.
func ctLogResume(t *testing.T, res []ctResumeLigne) {
	t.Helper()
	muettes, horsBande := 0, 0
	for _, r := range res {
		if !r.strict {
			horsBande++
		}
		if r.emissions == 0 {
			muettes++
		}
	}
	t.Logf("  RESUME par (slot, mode, tag) — %d series, dont %d sans aucune lecture chainee"+
		" (non listees) et %d sur des slots hors bande stricte", len(res), muettes, horsBande)
	for _, r := range res {
		if r.emissions == 0 {
			continue
		}
		t.Logf("    slot %-5d%s mode %s tag %-2d : %4d chainees (%4d avec valeur, %4d non chainees"+
			" ecartees) · %3d distinctes · %4d changements · %2d joueur(s) · t %7d -> %7d ms"+
			" (frames %5d -> %5d) · %v", r.cle.slot, ctBande(r.strict), ctMode(r.cle.modeA),
			r.cle.tag, r.emissions, r.avecValeur, r.nonChainees, r.distinctes, r.changements,
			r.joueurs, r.tMSPremiere, r.tMSDerniere, r.framePremiere, r.frameDerniere,
			p2bValeurs(r.valeurs))
	}
}

// ctBande marque un slot hors bande stricte.
func ctBande(strict bool) string {
	if strict {
		return " "
	}
	return "*"
}

// ctEcritResume ecrit le resume en TSV (toutes les series, y compris celles sans lecture chainee).
func ctEcritResume(t *testing.T, out, short string, res []ctResumeLigne) {
	t.Helper()
	var sb strings.Builder
	fmt.Fprintf(&sb, "# lot C-ter volet 1 — CT.1.1 — film %s — resume par (slot, mode, tag),"+
		" calcule sur les lectures CHAINEES ; non_chainees = lectures ecartees ; bande = 1 stricte\n", short)
	sb.WriteString("slot\tbande\tmode\ttag\temissions\tavec_valeur\tnon_chainees\tdistinctes\t" +
		"changements\tjoueurs\tt_ms_premiere\tt_ms_derniere\tframe_premiere\tframe_derniere\tvaleurs_top4\n")
	for _, r := range res {
		fmt.Fprintf(&sb, "%d\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n", r.cle.slot,
			ctBit(r.strict), ctMode(r.cle.modeA), r.cle.tag, r.emissions, r.avecValeur,
			r.nonChainees, r.distinctes, r.changements, r.joueurs, r.tMSPremiere, r.tMSDerniere,
			r.framePremiere, r.frameDerniere, strings.Join(p2bValeurs(r.valeurs), " "))
	}
	p2aWrite(t, out, short+"_ti13_resume.tsv", sb.String())
	t.Logf("  ecrit : %s (%d series)", filepath.Join(out, short+"_ti13_resume.tsv"), len(res))
}
