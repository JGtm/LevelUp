package replay

// zone_bande_ti13_test.go — LE CONTROLE DE BANDE DU CALQUE DES ZONES.
//
// # POURQUOI CET INSTRUMENT EXISTE
//
// `filmdec.ScanFilmManagedProperties` ancre ses records delta sur une BANDE DE SLOTS relevee aux
// images-cles. Deux regles de bande se disputent cet ancrage :
//
//	COMBLEE    la plage [min, max] des slots vus, trous compris, moins les slots vus porter un
//	           autre archetype (`worldObjectSlotBand`). Elle existe pour les objets NOMBREUX ET
//	           EPHEMERES — un projectile vit moins d'une seconde, les images-cles sont espacees
//	           de 20 s, et sans comblement on decode 57 vies au lieu de 580.
//	OBSERVEE   les seuls slots REELLEMENT vus (`observedSlotBand`). Pour un objet RARE ET
//	           DURABLE, qui apparait a chaque image-cle, le comblement n'a rien a rattraper.
//
// Le chainage des records delta de `ti=13` passe de 6,3 % (bande comblee) a 43,7 % (bande
// observee) sur 13 films — mesure du 2026-09-01, `filmdec.TestObjectifTi11DeltaControleTi13`.
// MAIS UN TAUX DE CHAINAGE N'EST PAS UN LIVRABLE : le seul juge est ce que le CONSOMMATEUR
// publie. Une bande plus etroite peut faire perdre des lectures REELLES, donc des etats de zone.
//
// # CE QU'IL MESURE, ET COMMENT IL SE LIT
//
// Le chemin de PRODUCTION de bout en bout — `ScanFilmManagedProperties` puis
// `BuildFromPositions` avec le calque des zones — et il publie, en lignes `TI13BANDE` tabulees,
// ce que l'artefact porterait : slots, lectures, series, designateur de colline, etats de zone,
// intervalles, frames actives, couverture. Deux passages du MEME instrument — le depot avant le
// changement de bande, puis apres — se diffent ligne a ligne.
//
// UN FILM PAR PROCESSUS, AVANT-PLAN, sous garde d'environnement — meme regime que la phase 2a
// dont il reprend le corpus, les entrees et les assembleurs :
//
//	$env:ZONE_FILM="<cache>/film_chunks/7f1bbf06"
//	go test -count=1 -run TestZoneBandeTi13Consommateur -v -timeout 30m ./internal/analysis/replay/
//
// # LE PERIMETRE DE PRODUCTION EST PLUS ETROIT QUE LE CORPUS
//
// `replaybuild.heldZoneRoles` ne retient que les roles de zone TENUE (`strongholds_zone`,
// `hill`) : un CTF, une Extraction, un Assaut n'ont AUCUN catalogue de zone, donc
// `decodeFilmZoneReads` ne balaye pas `ti=13` du tout. Les films de ces modes sont mesures au
// BALAYAGE seul — leur consommateur n'existe pas.

import (
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

// bandeHorsCorpus — les films du controle de bande ABSENTS du corpus gele de la phase 2a, avec
// ce qu'il faut pour les assembler.
//
// `7f1bbf06` est le KOTH ou la bande comblee coute le plus cher (52 slots contre 20 observes).
// SA CARTE EST DEDUITE, ET LA DEDUCTION EST VERIFIABLE : le releve du 2026-08-30 de
// `colline_seuil_garde_e1_test.go` lui donne la carte `streets`, et les cles du catalogue de
// bornes sont les noms publics en minuscules — une carte classee y serait `streets - ranked`,
// qui n'existe pas dans le fichier. Le `map_id` est donc celui de la carte publique « Streets ».
//
// AUCUN ROSTER, ET IL N'EN FAUT PAS : en KOTH le calque ne lit aucune capture nommee, et un
// `TeamByXUID` vide fait accepter les deux seuls camps mesures, 0 et 1 (`zoneOwnerTeam`).
var bandeHorsCorpus = map[string]p2aFilm{
	"7f1bbf06": {Mode: "KOTH", ObjType: "none", Carte: "streets",
		MapID: "9c7b0b0f-e933-4c2d-9d4a-3e4500d0de99"},
}

// bandeRoleTenu nomme le role de zone TENUE du mode — celui, et celui-la seul, que
// `replaybuild.matchZones` sert en production.
func bandeRoleTenu(f p2aFilm) (mapvar.Role, bool) {
	switch f.Mode {
	case "Strongholds":
		return mapvar.RoleStrongholdZone, true
	case "KOTH":
		return mapvar.RoleHill, true
	default:
		return "", false
	}
}

// bandeFilmOf rend la fiche du film, du corpus gele de la phase 2a ou du complement ci-dessus.
func bandeFilmOf(short string) (p2aFilm, bool) {
	if f, ok := p2aCorpus[short]; ok {
		return f, true
	}
	f, ok := bandeHorsCorpus[short]
	return f, ok
}

// TestZoneBandeTi13Consommateur publie ce que le calque des zones porte pour UN film.
func TestZoneBandeTi13Consommateur(t *testing.T) {
	dir := p2aRequireFilm(t)
	short := filepath.Base(dir)
	sc := p2bScan(t, dir)
	t.Logf("FILM %s — BALAYAGE ti=13 : %d slots, %d records, %d marches, %d cassees, %d chainees,"+
		" %d lectures", short, sc.Slots, sc.Records, sc.Walked, sc.Broken, sc.Chained, len(sc.Reads))
	bandeInt(t, short, "scan.slots", sc.Slots)
	bandeInt(t, short, "scan.records", sc.Records)
	bandeInt(t, short, "scan.marches", sc.Walked)
	bandeInt(t, short, "scan.cassees", sc.Broken)
	bandeInt(t, short, "scan.chainees", sc.Chained)
	bandeInt(t, short, "scan.lectures", len(sc.Reads))

	film, ok := bandeFilmOf(short)
	if !ok {
		t.Logf("  film hors corpus : aucun assemblage possible — BALAYAGE SEUL")
		return
	}
	role, tenue := bandeRoleTenu(film)
	if !tenue {
		t.Logf("  mode %s : aucun role de zone TENUE (heldZoneRoles) — la production ne balaye"+
			" PAS ti=13 sur ce film, son consommateur n'existe pas", film.Mode)
		return
	}
	zones := p2aZones(t, film.MapID, role)
	doc, origin := bandeDoc(t, dir, short, film, bandeEntree{sc: sc, zones: zones, role: role})
	bandeSeries(t, short, sc, doc, origin)
	bandeEtats(t, short, doc)
}

// bandeEntree regroupe ce que l'assemblage ajoute au film (regle des 5 parametres).
type bandeEntree struct {
	sc    filmdec.ManagedPropertyScan
	zones []Zone
	role  mapvar.Role
}

// bandeDoc assemble le document par le chemin de PRODUCTION, avec le calque des zones.
func bandeDoc(t *testing.T, dir, short string, film p2aFilm,
	e bandeEntree,
) (ReplayDocument, uint64) {
	t.Helper()
	caps := p2aCaptures(p2aSource(t, dir), film)
	zone := ZoneInput{
		Scanned: true, Reads: e.sc.Reads, Zones: e.zones, Roles: string(e.role),
		TeamByXUID: film.p2aTeams(), Hill: film.Mode == "KOTH",
	}
	doc, origin := p2bBuild(t, dir, short, p2aQuant(t, film.Carte), zone, caps)
	t.Logf("  ASSEMBLAGE : mode %s, carte %s, role %s — %d zone(s) au catalogue, %d capture(s)"+
		" nommee(s), %d trajectoires, %d frames de %d ms",
		film.Mode, film.Carte, e.role, len(e.zones), len(caps), len(doc.Tracks), doc.FrameCount,
		doc.FrameIntervalMS)
	bandeInt(t, short, "doc.frames", doc.FrameCount)
	bandeInt(t, short, "doc.catalogue", len(e.zones))
	return doc, origin
}

// bandeSeries publie les series que `zoneSeriesOf` tire des lectures, et le designateur de
// colline que l'election en retire — les deux entrees dont depend tout ce qui se publie ensuite.
func bandeSeries(t *testing.T, short string, sc filmdec.ManagedPropertyScan, doc ReplayDocument,
	origin uint64,
) {
	t.Helper()
	c := zoneCtx{origin: origin, step: uint64(doc.FrameIntervalMS) * 1000,
		frames: doc.FrameCount, intervalMS: doc.FrameIntervalMS}
	ser := zoneSeriesOf(sc.Reads, c)
	t.Logf("  SERIES : %d slot(s) parlant(s) — jauge %d, proprietaire %d, designation chainee %d",
		ser.slots, len(ser.gauge), len(ser.owner), len(ser.desig))
	bandeInt(t, short, "serie.slots", ser.slots)
	bandeInt(t, short, "serie.jauge", len(ser.gauge))
	bandeInt(t, short, "serie.proprietaire", len(ser.owner))
	bandeInt(t, short, "serie.designation", len(ser.desig))
	bandeInt(t, short, "serie.rampes", len(zoneRampsOf(ser)))
	d, ok := hillDesignatorOf(ser)
	if !ok {
		t.Logf("  DESIGNATEUR : AUCUN — attendu hors KOTH ; un film KOTH retomberait ici sur la" +
			" methode par rampes")
		bandeInt(t, short, "designateur.slot", -1)
		return
	}
	t.Logf("  DESIGNATEUR : slot %d, %d bascule(s), premier contact frame %d — proprietaire"+
		" slot %d (%d emissions)",
		d.slot, len(d.changes), d.first, d.slot+1, len(ser.owner[d.slot+1]))
	bandeInt(t, short, "designateur.slot", int(d.slot))
	bandeInt(t, short, "designateur.bascules", len(d.changes))
	bandeInt(t, short, "designateur.premier", d.first)
	bandeInt(t, short, "designateur.proprietaire", len(ser.owner[d.slot+1]))
}

// bandeEtats publie CE QUE L'ARTEFACT PORTERAIT : les etats de zone et leur couverture.
func bandeEtats(t *testing.T, short string, doc ReplayDocument) {
	t.Helper()
	spans, actives, avecOwner := 0, 0, 0
	for _, st := range doc.ZoneStates {
		spans += len(st.Spans)
		for _, sp := range st.Spans {
			if sp.Active {
				actives += sp.T1 - sp.T0 + 1
			}
			if sp.Owner != nil {
				avecOwner++
			}
		}
		t.Logf("  zone %d : %d intervalle(s), %s", st.ZoneRef, len(st.Spans), bandeApercu(st))
	}
	t.Logf("  ETATS : %d zone(s) publiee(s), %d intervalle(s), %d avec proprietaire, %d frame(s)"+
		" active(s) sur %d", len(doc.ZoneStates), spans, avecOwner, actives, doc.FrameCount)
	bandeInt(t, short, "etat.zones", len(doc.ZoneStates))
	bandeInt(t, short, "etat.intervalles", spans)
	bandeInt(t, short, "etat.avecProprietaire", avecOwner)
	bandeInt(t, short, "etat.framesActives", actives)
	bandeCouverture(t, short, doc)
}

// bandeCouverture publie les denominateurs du calque, ceux-la memes que l'artefact porte.
func bandeCouverture(t *testing.T, short string, doc ReplayDocument) {
	t.Helper()
	if doc.Coverage == nil || doc.Coverage.Zones == nil {
		t.Logf("  COUVERTURE : ABSENTE — l'appelant n'a rien fourni a lire")
		return
	}
	cov := doc.Coverage.Zones
	t.Logf("  COUVERTURE : methode %s · slots %d · apparies %d · non apparies %d · captures %d"+
		" dont %d attribuees · intervalles %d · periodes colline %d · sans proprietaire %d"+
		" · valeurs inconnues %d · points de jauge %d",
		cov.Method, cov.Slots, cov.Paired, cov.Unpaired, cov.Captures, cov.Attributed, cov.Spans,
		cov.HillPeriods, cov.OwnerUnpaired, cov.UnknownOwner, cov.GaugePoints)
	bandeTexte(t, short, "cov.methode", cov.Method)
	bandeInt(t, short, "cov.slots", cov.Slots)
	bandeInt(t, short, "cov.apparies", cov.Paired)
	bandeInt(t, short, "cov.nonApparies", cov.Unpaired)
	bandeInt(t, short, "cov.captures", cov.Captures)
	bandeInt(t, short, "cov.attribuees", cov.Attributed)
	bandeInt(t, short, "cov.intervalles", cov.Spans)
	bandeInt(t, short, "cov.periodesColline", cov.HillPeriods)
	bandeInt(t, short, "cov.sansProprietaire", cov.OwnerUnpaired)
	bandeInt(t, short, "cov.valeursInconnues", cov.UnknownOwner)
	bandeInt(t, short, "cov.pointsJauge", cov.GaugePoints)
}

// bandeApercu resume un etat de zone en une ligne : bornes couvertes et camps rencontres.
func bandeApercu(st ZoneState) string {
	if len(st.Spans) == 0 {
		return "aucun intervalle"
	}
	camps := map[int]int{}
	neutres := 0
	for _, sp := range st.Spans {
		if sp.Owner == nil {
			neutres++
			continue
		}
		camps[*sp.Owner]++
	}
	return fmt.Sprintf("frames %d..%d, camps %v, sans camp %d, %d point(s) de jauge",
		st.Spans[0].T0, st.Spans[len(st.Spans)-1].T1, camps, neutres, len(st.Gauge))
}

// bandeInt et bandeTexte ecrivent une observation sous une forme diffable d'un passage a
// l'autre. Le format est le contrat entre l'instrument et la comparaison AVANT/APRES.
func bandeInt(t *testing.T, short, clef string, v int) {
	t.Helper()
	t.Logf("TI13BANDE\t%s\t%s\t%d", short, clef, v)
}

func bandeTexte(t *testing.T, short, clef, v string) {
	t.Helper()
	t.Logf("TI13BANDE\t%s\t%s\t%s", short, clef, v)
}
