package replay

// ground_weapon_after_test.go — L'APRÈS de la réfutation des positions d'armes au sol, et
// l'IDENTITÉ portée par le record de création.
//
// CE QUE MESURE CET INSTRUMENT, et en quoi il diffère de `ground_weapon_research_test.go`
// (l'AVANT). L'AVANT prend la BANDE de slots `ti=42` lue aux images-clés, comblée puis privée
// des slots vus porter un autre archétype, et regarde la dispersion des positions delta de
// chaque slot. C'est ce qui a été RÉFUTÉ le 2026-08-12 : 3,3 % seulement des slots tiennent
// dans 0,5 u. La cause n'est pas le décodeur de position — c'est que la bande est une
// PRÉSOMPTION : un record delta ne dit pas son archétype.
//
// L'APRÈS remplace la présomption par une PREUVE. Le record de CRÉATION porte son
// `R(6) typeIndex` : un slot dont une création `ti=42` a été acceptée EST une arme au sol.
// Ce record n'était pas lisible avant que le default-state de l'archétype ne le soit
// (default_state_ti42.go, validé par l'oracle de position). On mesure donc la MÊME dispersion,
// sur la MÊME chaîne de décodage, mais sur la bande CONFIRMÉE.
//
// TROIS BANDES, UN SEUL CODE : la bande complète (l'AVANT), la bande confirmée (l'APRÈS) et le
// témoin FANTÔME de même cardinalité que la bande confirmée. Sans le fantôme recalibré, un gain
// de dispersion pourrait n'être qu'un effet de taille d'échantillon.
//
// LECTURE SEULE, aucune base. SOUS GARDE D'ENVIRONNEMENT (GW_FILM), comme l'AVANT.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 GW_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  GW_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  GW_MAP=Cliffhanger \
//	  go test ./internal/analysis/replay/ -run GroundWeaponAfter -timeout 60m -v

import (
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

func TestGroundWeaponAfter(t *testing.T) {
	filmDir := os.Getenv(gwFilmEnv)
	if filmDir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", gwFilmEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prevPrec := filmdec.WorldObjectPrecision
	t.Cleanup(func() { filmdec.WorldObjectPrecision = prevPrec })

	wr, ok := gwWorldRange(t)
	if !ok {
		t.Skipf("%s/%s absents : sans bornes, un quantum n'est pas une position", gwBoundsEnv, gwMapEnv)
	}
	census := gwKeyframeCensus(t, filmDir)
	band := filmdec.GroundWeaponSlotBand(filmDir)

	// CRÉATIONS : la bande réelle, puis le témoin fantôme de MÊME cardinalité par le MÊME code.
	cre, st, err := filmdec.ScanFilmGroundWeaponCreationsForBand(filmDir, &wr, band)
	if err != nil {
		t.Fatalf("créations ti=42 : %v", err)
	}
	phantomBand := gwPhantomBand(band, census.allSlots)
	pcre, pst, err := filmdec.ScanFilmGroundWeaponCreationsForBand(filmDir, &wr, phantomBand)
	if err != nil {
		t.Fatalf("créations ti=42 (fantôme) : %v", err)
	}
	t.Logf("CRÉATIONS ti=42 — bande %d slots · ancres %d · acceptées %d (masque douteux %d,"+
		" position rejetée %d, débordement %d) || FANTÔME %d slots · ancres %d · acceptées %d",
		st.Slots, st.Anchors, st.Accepted, st.MaskBad, st.PosBad, st.Overflow,
		pst.Slots, pst.Anchors, len(pcre))

	confirmed := gwConfirmedSlots(cre)
	t.Logf("BANDE CONFIRMÉE — %d slots prouvés ti=42 par un record de création, sur %d slots de"+
		" la bande présumée (%d slots vus ti=42 en image-clé)",
		len(confirmed), len(band), len(census.slotsTI[filmdec.GroundWeaponTypeIndex]))

	gwLogDispersions(t, filmDir, &wr, band, confirmed)
	gwLogLifeDispersion(t, filmDir, &wr, band, cre)
	gwLogIdentity(t, filmDir, cre)
}

// gwLogLifeDispersion mesure la dispersion À LA GRANULARITÉ D'UNE VIE — le couple
// (slot, génération) — et non du seul slot.
//
// POURQUOI C'EST LA BONNE CLÉ, et pourquoi la mesure de l'AVANT ne pouvait pas la prendre. Tout
// le décodeur d'objets du monde raisonne en couples (slot, génération) : le pool de slots
// reboucle et la génération ne fait que 2 bits, donc un même slot porte plusieurs objets
// successifs au cours d'un match. Regrouper les échantillons PAR SLOT mélange donc des objets
// distincts, posés à des endroits distincts — et un mélange s'étale, quelle que soit la qualité
// du décodage. L'AVANT ne pouvait pas faire autrement : sans record de création, rien ne disait
// quelles générations étaient des armes au sol. Le record de création les nomme.
func gwLogLifeDispersion(
	t *testing.T, dir string, wr *filmdec.Vec3Range, band map[uint32]bool,
	cre []filmdec.EquipmentCreation,
) {
	t.Helper()
	lives := map[[2]uint32]bool{}
	slots := map[uint32]bool{}
	for _, c := range cre {
		lives[[2]uint32{c.Slot, c.Gen}] = true
		slots[c.Slot] = true
	}
	samples := filmdec.WorldObjectPositionsForBand(dir, wr, slots)
	real := gwSplitByLife(samples, lives)
	// TÉMOIN : les MÊMES slots, mais les générations que la création n'a PAS confirmées. Même
	// code, même film, même bande — seule la confirmation change.
	ghost := gwSplitByLife(samples, nil)
	for k := range real {
		delete(ghost, k)
	}
	t.Logf("DISPERSION par VIE (slot,gen) CONFIRMÉE — %s", gwDispersion(real))
	t.Logf("DISPERSION par VIE non confirmée (témoin) — %s", gwDispersion(ghost))
}

// gwSplitByLife regroupe les échantillons par couple (slot, génération). `keep` nil garde tout.
// La clé de sortie encode le couple : `gwDispersion` ne s'intéresse qu'aux groupes, pas au nom.
func gwSplitByLife(
	m map[uint32][]filmdec.WorldObjectSample, keep map[[2]uint32]bool,
) map[uint32][]filmdec.WorldObjectSample {
	out := map[uint32][]filmdec.WorldObjectSample{}
	for slot, pts := range m {
		for _, p := range pts {
			if keep != nil && !keep[[2]uint32{slot, p.Gen}] {
				continue
			}
			k := slot<<2 | (p.Gen & 3)
			out[k] = append(out[k], p)
		}
	}
	return out
}

// gwConfirmedSlots rend les slots dont AU MOINS un record de création ti=42 a été accepté.
func gwConfirmedSlots(cre []filmdec.EquipmentCreation) map[uint32]bool {
	out := map[uint32]bool{}
	for _, c := range cre {
		out[c.Slot] = true
	}
	return out
}

// gwLogDispersions publie la dispersion des trois bandes : présumée (l'AVANT), confirmée
// (l'APRÈS) et fantôme recalibré à la cardinalité de la confirmée.
func gwLogDispersions(
	t *testing.T, dir string, wr *filmdec.Vec3Range, band, confirmed map[uint32]bool,
) {
	t.Helper()
	full := filmdec.WorldObjectPositionsForBand(dir, wr, band)
	conf := filmdec.WorldObjectPositionsForBand(dir, wr, confirmed)
	ghost := filmdec.WorldObjectPositionsForBand(dir, wr, gwGhostOfSize(band, len(confirmed)))
	t.Logf("DISPERSION bande PRÉSUMÉE (AVANT) — %s", gwDispersion(full))
	t.Logf("DISPERSION bande CONFIRMÉE (APRÈS) — %s", gwDispersion(conf))
	t.Logf("DISPERSION fantôme recalibré      — %s", gwDispersion(ghost))
}

// gwGhostOfSize construit un témoin de n slots jamais vus dans la bande. Il double le témoin de
// l'AVANT : celui-ci a la cardinalité de la bande CONFIRMÉE, la seule comparaison honnête.
func gwGhostOfSize(band map[uint32]bool, n int) map[uint32]bool {
	out := make(map[uint32]bool, n)
	for s := uint32(1); len(out) < n && s < 8192; s++ {
		if band[s] {
			continue
		}
		out[s] = true
	}
	return out
}

// gwLogIdentity croise le mot de 32 bits du bloc `object-multiplayer-properties` du record de
// création avec la FAMILLE high-32 lue aux images-clés pour la MÊME vie (slot, génération).
//
// DEUX CHAÎNES INDÉPENDANTES : le mot MPP vient du default-state d'un record delta, la famille
// d'un balayage de motifs dans le payload d'image-clé. Leur accord ne peut pas être un artefact
// commun — et leur DÉSACCORD dirait que le mot de 32 bits n'est pas ce qu'on croit.
func gwLogIdentity(t *testing.T, dir string, cre []filmdec.EquipmentCreation) {
	t.Helper()
	known := loadoutFamilies()
	kf, err := filmdec.ScanFilmKeyframeGroundWeapons(dir, known)
	if err != nil {
		t.Logf("IDENTITÉ — armes au sol d'image-clé illisibles : %v", err)
		return
	}
	fam := map[[2]uint32]uint32{}
	for _, g := range kf {
		if len(g.Families) > 0 {
			fam[[2]uint32{g.Slot, g.Gen}] = g.Families[0]
		}
	}
	pairs, agree, inCatalog, words := 0, 0, 0, map[uint32]int{}
	for _, c := range cre {
		if !c.MPPPresent[filmdec.MPPWord32] {
			continue
		}
		w := uint32(c.MPPVal[filmdec.MPPWord32])
		words[w]++
		if known[w] {
			inCatalog++
		}
		f, ok := fam[[2]uint32{c.Slot, c.Gen}]
		if !ok {
			continue
		}
		pairs++
		if f == w {
			agree++
		}
	}
	t.Logf("IDENTITÉ — créations avec mot MPP 32 bits %d · valeurs distinctes %d · dont dans le"+
		" catalogue d'armes %d · paires croisées avec une famille d'image-clé %d · ACCORD %d",
		len(cre), len(words), inCatalog, pairs, agree)
	for _, e := range gwTopWords(words, 12) {
		t.Logf("    0x%08x  x%-4d  %s", e.word, e.count, weaponv3.WeaponName(e.word))
	}
}

type gwWordCount struct {
	word  uint32
	count int
}

// gwTopWords rend les mots les plus fréquents, par compte décroissant puis par valeur (ordre
// TOTAL : une map d'itération aléatoire ferait varier le rapport d'une exécution à l'autre).
func gwTopWords(words map[uint32]int, max int) []gwWordCount {
	out := make([]gwWordCount, 0, len(words))
	for w, c := range words {
		out = append(out, gwWordCount{w, c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}
		return out[i].word < out[j].word
	})
	if len(out) > max {
		out = out[:max]
	}
	return out
}
