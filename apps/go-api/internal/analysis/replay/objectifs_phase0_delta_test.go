package replay

// objectifs_phase0_delta_test.go — ITEM 0.2 : LE CANAL DELTA `World.HeldWeapon`.
//
// CE QUE CE CANAL EST, EXACTEMENT. `weapon-state-type-info` (i43..i46) porte en clair, dans
// les paquets DELTA, le nom de variante de l'arme tenue — un entier de 32 bits, LE MEME
// espace que les familles lues aux images-cles (`consumeWeaponStateTypeInfoVariant`,
// `components_object.go`). Le traverseur le capture dans `EntityTrace.HeldWeapon`
// (`traverse.go`), et `DecodeFrameRecords` le recopie dans le cache par slot du World
// (`frame_records.go` -> `World.SetHeldWeapon`). Ce cache n'a AUCUN appelant en production et
// n'a jamais ete mesure : c'est ce que cet item corrige.
//
// LE CANAL N'EST ATTEIGNABLE QUE PAR LA MARCHE STATEFUL, ET C'EST CE QUI EST MESURE ICI.
// `World.SetHeldWeapon` n'est appele que depuis `DecodeFrameRecords` (deux sites, sur un NEW
// propre et sur un DELTA propre). Mesurer le canal, c'est donc mesurer ce que cette
// marche-la delivre — pas ce qu'un autre chemin d'acces POURRAIT delivrer. Le monde est
// reamorce a chaque image-cle (`WorldFromKeyframe`), exactement comme le fait la chaine de
// production du `killsource`.
//
// SEUIL, ECRIT AVANT LA MESURE (plan, item 0.2) : >= 90 % des prises appariees a une lecture
// du motif du drapeau dans les 2 s. En dessous, le canal est REFUTE et l'item 0.1 (les
// images-cles) reste le seul canal retenu.
//
// AUCUN CHANGEMENT DE PRODUCTION : ce fichier est un test, il n'emploie que l'API exportee du
// decodeur et n'ecrit rien.

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// objNoVariant : la sentinelle « pas d'arme lue » du decodeur (`filmdec.noVariant`, non
// exportee — valeur recopiee ici avec sa source, `traverse.go`).
const objNoVariant uint32 = 0xFFFFFFFF

// objVuesParPaquet : nombre de vues de replication deroulees par paquet, meme reglage que la
// marche de production du `killsource`.
const objVuesParPaquet = 4

// objSeuilLatenceMS / objSeuilApparie — le seuil de l'item 0.2.
const (
	objSeuilLatenceMS = 2000
	objSeuilApparie   = 0.90
)

// objLectureArme est une lecture du canal delta : quand, quel slot, quelle famille.
type objLectureArme struct {
	TS   uint64 // horloge du FILM, microsecondes
	Slot uint32
	Fam  uint32
}

// objStatDelta porte de quoi juger la couverture du canal — sans elle, une latence mediane
// ne dit rien (on peut avoir une excellente mediane sur trois lectures).
type objStatDelta struct {
	Paquets, PaquetsAvecLecture int
	Records, RecordsPropres     int
	Lectures                    int
	Slots, Familles             int
}

// objScanCanalDelta deroule la marche stateful sur tout le film et collecte les lectures
// d'arme tenue.
func objScanCanalDelta(dir string) ([]objLectureArme, objStatDelta, error) {
	brut, err := filmdec.ReadFilmChunk(dir, 0)
	if err != nil {
		return nil, objStatDelta{}, err
	}
	reg, err := filmdec.ParseRegistryChunk(brut)
	if err != nil {
		return nil, objStatDelta{}, err
	}
	cfg := filmdec.DefaultFrameConfig()
	w := filmdec.NewWorld(reg)
	var out []objLectureArme
	st := objStatDelta{}
	slots, familles := map[uint32]bool{}, map[uint32]bool{}
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			pay := p.Payload(data)
			if p.Type == filmdec.PacketTypeKeyframe {
				w = filmdec.WorldFromKeyframe(reg, pay)
				continue
			}
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			st.Paquets++
			out = objPaquetDelta(pay, p.TimestampUS, w, cfg, out, &st, slots, familles)
		}
	}
	st.Slots, st.Familles = len(slots), len(familles)
	return out, st, nil
}

// objPaquetDelta deroule un paquet et accumule ses lectures d'arme tenue.
func objPaquetDelta(pay []byte, ts uint64, w *filmdec.World, cfg filmdec.FrameConfig,
	out []objLectureArme, st *objStatDelta, slots, familles map[uint32]bool) []objLectureArme {
	recs, _ := filmdec.DecodeFrameViews(pay, w, cfg, objVuesParPaquet, cfg.PacketPreambleBits)
	avant := len(out)
	for _, r := range recs {
		st.Records++
		if r.DesyncAt != -1 {
			continue
		}
		st.RecordsPropres++
		if r.Trace.HeldWeapon == objNoVariant {
			continue
		}
		st.Lectures++
		slots[r.Slot] = true
		familles[r.Trace.HeldWeapon] = true
		out = append(out, objLectureArme{TS: ts, Slot: r.Slot, Fam: r.Trace.HeldWeapon})
	}
	if len(out) > avant {
		st.PaquetsAvecLecture++
	}
	return out
}

// TestObjectifsPhase0CanalDelta — ITEM 0.2.
func TestObjectifsPhase0CanalDelta(t *testing.T) {
	root := objRequireRoot(t)
	motif, aMotif := objMotifRetenu(t, root)
	joues := 0
	var cumApparies, cumPrises int
	var latences []int64
	for _, id := range objCTFFilms {
		src, ok := objOpenFilm(t, root, id)
		if !ok {
			continue
		}
		joues++
		lectures, st, err := objScanCanalDelta(objChunkDir(root, id))
		if err != nil {
			t.Fatalf("%s : canal delta : %v", id, err)
		}
		t.Logf("%s : canal delta — %d paquets delta, %d avec au moins une lecture ; %d records "+
			"deroules dont %d propres ; %d lectures d'arme tenue, %d slots, %d familles distinctes",
			id, st.Paquets, st.PaquetsAvecLecture, st.Records, st.RecordsPropres,
			st.Lectures, st.Slots, st.Familles)
		if aMotif {
			t.Logf("%s : le motif du drapeau 0x%08X apparait %d fois dans le canal delta",
				id, motif, objCompteFamille(lectures, motif))
		}
		a, p, l := objLatencesPrise(t, root, id, src, lectures, motif)
		cumApparies, cumPrises = cumApparies+a, cumPrises+p
		latences = append(latences, l...)
	}
	if joues == 0 {
		t.Skipf("aucun film CTF dans le cache (%s=%q)", objFilmEnv, root)
	}
	part := objPart(cumApparies, cumPrises)
	t.Logf("ITEM 0.2 — VERDICT : %d/%d prises appariees a <= %d ms = %.1f %% (seuil %.0f %%) ; "+
		"latence mediane %d ms, p90 %d ms -> canal %s",
		cumApparies, cumPrises, objSeuilLatenceMS, 100*part, 100*objSeuilApparie,
		objMedianeI64(latences), objPercentileI64(latences, 90),
		objVerdictCanal(part >= objSeuilApparie))
}

// objVerdictCanal met en mots le verdict du canal.
func objVerdictCanal(ok bool) string {
	if ok {
		return "RETENU"
	}
	return "REFUTE"
}

// objCompteFamille compte les lectures d'une famille donnee.
func objCompteFamille(l []objLectureArme, fam uint32) int {
	n := 0
	for _, r := range l {
		if r.Fam == fam {
			n++
		}
	}
	return n
}

// objLatencesPrise apparie chaque prise de drapeau a la premiere lecture du motif sur le slot
// du porteur, et rend (appariees, prises, latences).
func objLatencesPrise(t *testing.T, root, id string, src *objDiskFilm, lectures []objLectureArme,
	motif uint32) (int, int, []int64) {
	t.Helper()
	b := objBridgeOf(t, root, id)
	identity, _, _ := objStatPont(objectiveevents.StatRecords(src), b.Deaths)
	evs := objectiveevents.IdentifyNamedEvents(
		objectiveevents.NamedEvents(src, objectiveevents.ObjectiveTypeFlag), objIdentityStrings(identity))
	wins, _ := objPortageWindows(evs, b.Deaths, objFinMatch(evs, b.Deaths))
	parXUID := map[uint64][]int64{}
	for _, r := range lectures {
		if r.Fam != motif {
			continue
		}
		if x, ok := b.SlotXUID[r.Slot]; ok {
			parXUID[x] = append(parXUID[x], int64(r.TS/1000)-b.OffsetMS)
		}
	}
	for x := range parXUID {
		sort.Slice(parXUID[x], func(i, j int) bool { return parXUID[x][i] < parXUID[x][j] })
	}
	apparies := 0
	var lat []int64
	for _, wnd := range wins {
		for _, at := range parXUID[wnd.XUID] {
			if at < wnd.T0 || at > wnd.T0+objSeuilLatenceMS {
				continue
			}
			apparies++
			lat = append(lat, at-wnd.T0)
			break
		}
	}
	t.Logf("%s : %d prises, %d appariees a une lecture du motif dans les %d ms",
		id, len(wins), apparies, objSeuilLatenceMS)
	return apparies, len(wins), lat
}

// objMedianeI64 rend la mediane d'une serie (0 si vide).
func objMedianeI64(v []int64) int64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]int64{}, v...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c[len(c)/2]
}

// objPercentileI64 rend le percentile p d'une serie (0 si vide).
func objPercentileI64(v []int64, p int) int64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]int64{}, v...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	i := len(c) * p / 100
	if i >= len(c) {
		i = len(c) - 1
	}
	return c[i]
}
