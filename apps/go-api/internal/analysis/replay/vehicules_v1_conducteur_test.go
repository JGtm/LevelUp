package replay

// vehicules_v1_conducteur_test.go — ITEM 1 du lot V1 (suite) : ATTRIBUTION DU CONDUCTEUR.
//
// CE QU'IL PRODUIT. La primitive de V1a.4 (rapport du 2026-08-31, § 4.5) productionisee : pour
// chaque VIE de vehicule (recensement images-cles (slot, gen)), le bipede dont un TROU du flux de
// position S'OUVRE le plus pres du vehicule pendant la vie = OCCUPANT candidat. Le DEBUT DE TROU
// est la bonne primitive : l'occupant attache cesse de repliquer sa position monde, donc un
// embarquement se voit comme le dernier point d'un flux bipede, pres d'un vehicule, suivi d'un
// silence. C'est ce candidat qui donne au rejeu 2D la COULEUR D'EQUIPE du vehicule (le rejeu sait
// deja bipede -> joueur -> equipe -> couleur).
//
// LES SEUILS SONT ECRITS ICI, AVANT TOUTE EXECUTION :
//
//	TEMOIN FANTOME   une bande de slots jamais vus porter le moindre archetype rend, par le MEME
//	                 chemin, un nombre de debuts-de-trou pres d'un "vehicule" fantome < 5 % du
//	                 signal reel. Un temoin fantome au-dessus arrete l'item.
//	AMBIGUITE        publiee, chiffree, jamais cachee : une vie avec >= 2 occupants candidats
//	                 distincts ne se departage pas par la geometrie (conducteur vs passager).
//	CHANCE           la presence de fond (part du temps ou un bipede est a moins de 1,5 m d'un
//	                 vehicule) est le denominateur : "X % des trous s'ouvrent pres d'un vehicule"
//	                 ne dit rien sans elle. Publiee comme rapport signal / hasard.
//
// LIMITE STRUCTURELLE, ecrite car elle borne le livrable : `BipedPosition` ne porte PAS de
// generation. Les vies d'un vehicule sont fusionnees PAR SLOT dans le flux de position ; le
// recensement (SeenUS, cle (slot, gen)) les separe. Toute attribution lue sur les positions
// designe donc un SLOT, borne a la fenetre de recensement de la vie — pas une vie au sens strict.
//
// CE QUI EST REUTILISE (et ne doit pas etre reecrit, sous peine de chiffres incomparables) :
// `v0Corpus`/`v0Bornes` (corpus + bornes), `v1aBandeVehicule`/`v1aOptions`/`v1aPistes` (nuage
// vehicule par la grammaire bipede), `attVehiculeLePlusProche`/`attTrouMS`/`attBordRayonM`
// (l'oracle geometrique du 18/08), `attBandeFantome`/`attBandesKeyframe` (temoin fantome),
// `v1aPresenceDeFond` (denominateur). Ce fichier vit et meurt avec les instruments V0/V1a.
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte.
//
//	CGO_ENABLED=0 ATT_FILM=<depot>/data/cache \
//	  V0_FILMS="0d76e8f1:behemoth,4898d586:behemoth" \
//	  go test ./internal/analysis/replay/ -run TestV1Conducteur -v -timeout 120m

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// Seuils et bornes de l'item 1, ecrits avant mesure.
const (
	// v1cGapMinMS : duree minimale d'un TROU du flux pour compter comme embarquement. = attTrouMS.
	v1cGapMinMS = attTrouMS
	// v1cVieTolMS : le recensement BORNE a ~1 image-cle pres (~20 s : mediane d'intervalle mesuree
	// 20,00 s). La fenetre d'une vie est donc [premier - tol, dernier + tol] pour rattacher un
	// debut de trou. SeenUS ne DATE pas, il BORNE.
	v1cVieTolMS = uint64(20_000)
	// v1cTemoinMaxPart : le temoin fantome doit rester sous 5 % du signal reel.
	v1cTemoinMaxPart = 0.05
)

// v1cVie est une vie de vehicule telle que le recensement la borne.
type v1cVie struct {
	Key         filmdec.EquipmentLifeKey
	T0, T1      uint64 // premiere et derniere image-cle qui la recense
	CensusCount int    // nombre d'images-cles qui la recensent
	Cand        map[uint32]uint64
}

// v1cEvent est un DEBUT DE TROU du flux d'un bipede dont le dernier point est pres d'un vehicule.
type v1cEvent struct {
	BipedeSlot, VehiculeSlot uint32
	TgapUS                   uint64
}

// TestV1ConducteurAttribution mesure l'attribution du conducteur sur le corpus.
func TestV1ConducteurAttribution(t *testing.T) {
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		v1cUnFilm(t, root, f)
	}
}

// v1cUnFilm mesure UN film.
func v1cUnFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	dir := objChunkDir(root, f.ID)
	if filmdec.CountFilmChunks(dir) == 0 {
		t.Logf("%s : film absent du cache — saute", f.ID)
		return
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := v0Bornes(t, root, f.Carte)
	if !ok {
		return
	}
	bande := v1aBandeVehicule(dir)
	if len(bande) == 0 {
		t.Logf("V1c %s (%s) — bande ti=%d vide : rien a mesurer", f.ID, f.Carte, attVehiculeTI)
		return
	}
	vies := v1cVies(dir)
	vehTracks := v1aPistes(v1cScan(t, f, dir, bande, &wr))
	optBip := filmdec.DefaultScanFilmOptions()
	optBip.WorldRange = &wr
	bip, err := filmdec.ScanFilmBipedPositions(dir, optBip)
	if err != nil {
		t.Logf("V1c %s : balayage des bipedes : %v", f.ID, err)
		return
	}
	events := v1cGapStartsNearVehicles(bip, vehTracks)
	v1cAttribue(events, vies)
	v1cPublie(t, f, vies, vehTracks, bip, events)
	v1cTemoinFantome(t, f, dir, &wr, bip, len(events))
}

// v1cScan balaie le nuage vehicule par la grammaire bipede (filtres de production armes, comme
// V1a.4). Un echec est journalise et rend un nuage vide plutot que d'interrompre la mesure.
func v1cScan(t *testing.T, f v0Film, dir string, band map[uint32]bool, wr *filmdec.Vec3Range) []filmdec.BipedPosition {
	t.Helper()
	pos, err := filmdec.ScanFilmBipedPositionsForBand(dir, band, v1aOptions(wr, true))
	if err != nil {
		t.Logf("V1c %s : balayage du nuage vehicule : %v", f.ID, err)
		return nil
	}
	return pos
}

// v1cVies construit les vies bornees depuis le recensement des images-cles.
func v1cVies(dir string) []v1cVie {
	k := filmdec.ScanFilmWorldObjectKeyframes(dir, int(attVehiculeTI))
	out := make([]v1cVie, 0, len(k.SeenUS))
	for key, vus := range k.SeenUS {
		out = append(out, v1cVie{
			Key: key, T0: vus[0], T1: vus[len(vus)-1], CensusCount: len(vus),
			Cand: map[uint32]uint64{},
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key.Slot != out[j].Key.Slot {
			return out[i].Key.Slot < out[j].Key.Slot
		}
		return out[i].Key.Gen < out[j].Key.Gen
	})
	return out
}

// v1cGapStartsNearVehicles releve les DEBUTS DE TROU (>= v1cGapMinMS) dont le dernier point est a
// moins de attBordRayonM d'un vehicule, avec le slot de ce vehicule et la distance.
func v1cGapStartsNearVehicles(bip []filmdec.BipedPosition, veh []filmdec.ProjectileTrack) []v1cEvent {
	parBipede := map[uint32][]filmdec.BipedPosition{}
	for _, b := range bip {
		if b.HasWorld {
			parBipede[b.Slot] = append(parBipede[b.Slot], b)
		}
	}
	slots := make([]uint32, 0, len(parBipede))
	for s := range parBipede {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []v1cEvent
	for _, s := range slots {
		ech := parBipede[s]
		sort.SliceStable(ech, func(i, j int) bool { return ech[i].TimestampUS < ech[j].TimestampUS })
		out = append(out, v1cGapStartsDuBipede(s, ech, veh)...)
	}
	return out
}

// v1cGapStartsDuBipede releve les debuts de trou d'UN bipede. Le predicat de proximite est
// `attVehiculeLePlusProche` — exactement celui de l'oracle geometrique du 18/08 (1,5 m, plus
// proche voisin temporel), pour que le signal reste comparable a V1a.4.
func v1cGapStartsDuBipede(slot uint32, ech []filmdec.BipedPosition, veh []filmdec.ProjectileTrack) []v1cEvent {
	var out []v1cEvent
	for i := 1; i < len(ech); i++ {
		if int64(ech[i].TimestampUS-ech[i-1].TimestampUS)/1000 < v1cGapMinMS {
			continue
		}
		vs, ok := attVehiculeLePlusProche(ech[i-1], veh)
		if !ok {
			continue
		}
		out = append(out, v1cEvent{BipedeSlot: slot, VehiculeSlot: vs, TgapUS: ech[i-1].TimestampUS})
	}
	return out
}

// v1cAttribue rattache chaque evenement a la vie (slot, gen) dont la fenetre bornee contient son
// instant. Plusieurs generations d'un meme slot : la plus proche du centre l'emporte.
func v1cAttribue(events []v1cEvent, vies []v1cVie) {
	parSlot := map[uint32][]int{}
	for i := range vies {
		parSlot[vies[i].Key.Slot] = append(parSlot[vies[i].Key.Slot], i)
	}
	for _, e := range events {
		best, bestGap := -1, uint64(1)<<63
		for _, vi := range parSlot[e.VehiculeSlot] {
			v := vies[vi]
			if e.TgapUS+v1cVieTolMS < v.T0 || e.TgapUS > v.T1+v1cVieTolMS {
				continue
			}
			centre := (v.T0 + v.T1) / 2
			if g := attEcartUS(e.TgapUS, centre); g < bestGap {
				best, bestGap = vi, g
			}
		}
		if best >= 0 {
			vies[best].Cand[e.BipedeSlot] = e.TgapUS
		}
	}
}

// v1cPublie ecrit la synthese et la table vie -> conducteur(s) candidat(s).
func v1cPublie(t *testing.T, f v0Film, vies []v1cVie, veh []filmdec.ProjectileTrack,
	bip []filmdec.BipedPosition, events []v1cEvent) {
	t.Helper()
	attrib, ambig, longues := 0, 0, 0
	for _, v := range vies {
		if v.CensusCount >= 2 {
			longues++
		}
		if len(v.Cand) >= 1 {
			attrib++
		}
		if len(v.Cand) >= 2 {
			ambig++
		}
	}
	pres, tot := v1aPresenceDeFond(veh, bip)
	trous := v1cCompteTrous(bip)
	fond := attPart(pres, tot)
	attendus := float64(trous) * fond
	ratio := 0.0
	if attendus > 0 {
		ratio = float64(len(events)) / attendus
	}
	t.Logf("V1c %s (%s) — %d vies recensees (%d vues >= 2 fois) · %d pistes vehicule · "+
		"%d trous >= %d ms · %d debuts de trou pres d'un vehicule (< %.1f m)",
		f.ID, f.Carte, len(vies), longues, len(veh), trous, v1cGapMinMS, len(events), attBordRayonM)
	t.Logf("V1c %s — ATTRIBUTION : %d/%d vies avec >= 1 conducteur candidat (%.1f %%) · "+
		"AMBIGUITE : %d vies avec >= 2 candidats (%.1f %% des attribuees) · "+
		"presence de fond %.1f %% -> les debuts de trou sont x%.1f le hasard",
		f.ID, attrib, len(vies), 100*attPart(attrib, len(vies)), ambig,
		100*attPart(ambig, attrib), 100*fond, ratio)
	v1cTable(t, f, vies)
}

// v1cCompteTrous compte les trous >= v1cGapMinMS, tous bipedes confondus — le denominateur du
// rapport a la chance.
func v1cCompteTrous(bip []filmdec.BipedPosition) int {
	parBipede := map[uint32][]uint64{}
	for _, b := range bip {
		if b.HasWorld {
			parBipede[b.Slot] = append(parBipede[b.Slot], b.TimestampUS)
		}
	}
	n := 0
	for _, ts := range parBipede {
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		for i := 1; i < len(ts); i++ {
			if int64(ts[i]-ts[i-1])/1000 >= v1cGapMinMS {
				n++
			}
		}
	}
	return n
}

// v1cTable journalise la table des vies attribuees (au plus 40 lignes pour ne pas noyer la sortie).
func v1cTable(t *testing.T, f v0Film, vies []v1cVie) {
	t.Helper()
	lignes := 0
	for _, v := range vies {
		if len(v.Cand) == 0 {
			continue
		}
		lignes++
		if lignes > 40 {
			t.Logf("V1c %s —   ... (table tronquee a 40 lignes)", f.ID)
			return
		}
		slots := make([]uint32, 0, len(v.Cand))
		for s := range v.Cand {
			slots = append(slots, s)
		}
		sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
		marque := ""
		if len(slots) >= 2 {
			marque = " [AMBIGU]"
		}
		t.Logf("V1c %s —   vie slot=%d gen=%d [%.0f..%.0f s] : conducteur(s) candidat(s) %v%s",
			f.ID, v.Key.Slot, v.Key.Gen, float64(v.T0)/1e6, float64(v.T1)/1e6, slots, marque)
	}
	if lignes == 0 {
		t.Logf("V1c %s —   aucune vie attribuee", f.ID)
	}
}

// v1cTemoinFantome rejoue le releve de debuts-de-trou contre une bande fantome (slots jamais vus
// porter le moindre archetype) et verdit le gate.
func v1cTemoinFantome(t *testing.T, f v0Film, dir string, wr *filmdec.Vec3Range,
	bip []filmdec.BipedPosition, signal int) {
	t.Helper()
	vus, autres := attBandesKeyframe(dir)
	fantome := attBandeFantome(vus, autres)
	if len(fantome) == 0 {
		t.Logf("V1c %s — aucun slot libre pour une bande fantome — temoin impossible", f.ID)
		return
	}
	fveh, err := filmdec.ScanFilmBipedPositionsForBand(dir, fantome, v1aOptions(wr, true))
	if err != nil {
		t.Logf("V1c %s — bande fantome : %v", f.ID, err)
		return
	}
	nf := len(v1cGapStartsNearVehicles(bip, v1aPistes(fveh)))
	verdict := "PASSE"
	if float64(nf) >= v1cTemoinMaxPart*float64(signal) {
		verdict = "ECHOUE"
	}
	t.Logf("V1c %s — TEMOIN FANTOME (%d slots jamais vus porter ti=%d) : %d debuts de trou pres "+
		"d'un vehicule fantome contre %d reels (seuil < %.0f %% du signal) %s",
		f.ID, len(fantome), attVehiculeTI, nf, signal, 100*v1cTemoinMaxPart, verdict)
}
