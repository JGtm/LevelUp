package replay

// objectifs_phase0_nommage_test.go — ITEM 0.3 : NOMMER LE MOTIF, ET LE GENERALISER AU CRANE.
//
// DEUX VOLETS, DEUX SEUILS ECRITS AVANT LA MESURE :
//
//	NOMMER. Le motif est-il un `weap` ? La reponse ne se devine pas : un identifiant d'arme
//	de ce jeu est un entier de 64 bits dont la moitie basse vaut 0x42C9679F, et sa moitie
//	haute est le global tag id du `weap` (chaine etablie, cf. l'etat de l'art des icones).
//	Le motif porte-t-il ce suffixe ? Sinon il n'a PAS de nom de tag, et c'est ce qu'on ecrit.
//
//	LE CRANE. Sur `24dbb67d` (Oddball), aucun oracle nomme n'existe — le statborg ne replique
//	aucun compteur de crane (mesure `objectiveevents`, mode sans table). La signature est
//	donc STRUCTURELLE : une famille portee par UN SEUL bipede a la fois, qui change de main.
//	Seuil : <= 1 porteur sur >= 90 % des images ou la famille est portee, et au moins deux
//	porteurs distincts sur le match. Controle grossier : les evenements `th=10` du footer.

import (
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// objSeuilMonoPorteur / objMinImagesPortees / objMinPorteursDistincts — les seuils de la
// signature structurelle du crane.
const (
	objSeuilMonoPorteur     = 0.90
	objMinPartImages        = 0.20
	objMinPorteursDistincts = 2
)

// objMotifMemo memorise le motif retenu par l'item 0.1 (il coute un balayage des trois films).
var objMotifMemo = map[string]uint32{}

// objMotifRetenu rend le motif COMMUN aux trois films CTF, celui que l'item 0.1 designe.
// Recalcule (et memorise) plutot que code en dur : un chiffre grave dans le code cesse d'etre
// une mesure des que le decodeur bouge.
func objMotifRetenu(t *testing.T, root string) (uint32, bool) {
	t.Helper()
	if v, ok := objMotifMemo["ctf"]; ok {
		return v, v != 0
	}
	compte := map[uint32]int{}
	films := 0
	for _, id := range objCTFFilms {
		src, ok := objOpenFilm(t, root, id)
		if !ok {
			continue
		}
		films++
		b := objBridgeOf(t, root, id)
		identity, _, _ := objStatPont(objectiveevents.StatRecords(src), b.Deaths)
		evs := objectiveevents.IdentifyNamedEvents(
			objectiveevents.NamedEvents(src, objectiveevents.ObjectiveTypeFlag), objIdentityStrings(identity))
		recs, _ := objRecordsOf(t, root, id)
		wins, _ := objPortageWindows(evs, b.Deaths, objFinMatch(evs, b.Deaths))
		for _, c := range objCandidats(objConfronte(recs, b, wins)) {
			compte[c.Val]++
		}
	}
	var communs []uint32
	for v, n := range compte {
		if films > 0 && n == films {
			communs = append(communs, v)
		}
	}
	sort.Slice(communs, func(i, j int) bool { return communs[i] < communs[j] })
	var retenu uint32
	if g := objGroupesDecalage(communs); len(g) == 1 {
		for r := range g {
			retenu = r
		}
	}
	objMotifMemo["ctf"] = retenu
	return retenu, retenu != 0
}

// TestObjectifsPhase0Nommage — ITEM 0.3, volet « nommer ».
func TestObjectifsPhase0Nommage(t *testing.T) {
	root := objRequireRoot(t)
	motif, ok := objMotifRetenu(t, root)
	if !ok {
		t.Skipf("aucun motif unique retenu par l'item 0.1 — rien a nommer")
	}
	for _, id := range objCTFFilms {
		if _, present := objOpenFilm(t, root, id); !present {
			continue
		}
		ctx, err := objMotifContexte(objChunkDir(root, id), motif)
		if err != nil {
			t.Fatalf("%s : contexte : %v", id, err)
		}
		t.Logf("%s : motif 0x%08X — %d occurrences, offset median %d bits depuis le debut du "+
			"record ; suffixe d'identifiant d'arme (0x%08X) trouve %d fois",
			id, motif, ctx.Occurrences, ctx.OffsetMedian, objSuffixeArme, ctx.SuffixeArme)
		t.Logf("%s : 32 bits AVANT les plus frequents : %s", id, objMotsString(ctx.Avant))
		t.Logf("%s : 32 bits APRES les plus frequents : %s", id, objMotsString(ctx.Apres))
		recs, _ := objRecordsOf(t, root, id)
		distrib, images := objSimultaneite(recs, motif)
		t.Logf("%s : simultaneite du motif sur %d images-cles — %s", id, images, objDistribString(distrib))
	}
}

// objMotsString met en forme une liste de mots frequents.
func objMotsString(ms []objMotFrequent) string {
	if len(ms) == 0 {
		return "(aucun)"
	}
	s := ""
	for i, m := range ms {
		if i > 0 {
			s += ", "
		}
		s += "0x" + strings.ToUpper(strconv.FormatUint(uint64(m.Mot), 16)) + " x" + itoa(m.Compte)
	}
	return s
}

// objSignature est le resultat de la recherche structurelle d'un objet porte.
type objSignature struct {
	Val               uint32
	Images            int
	ImagesMonoPorteur int
	Porteurs          int
}

// objSignaturesPortees cherche, SANS ORACLE, les valeurs qui se comportent comme un objet
// porte : presentes sur une part notable des images-cles, tenues par un seul bipede a la
// fois, et passees d'un joueur a un autre.
func objSignaturesPortees(recs []objRecord, b objBridge) []objSignature {
	instants := map[uint64]bool{}
	parVal := map[uint32]map[uint64]map[uint32]bool{}
	for _, r := range recs {
		instants[r.TS] = true
		if _, ok := b.SlotXUID[r.Slot]; !ok {
			continue
		}
		for _, v := range r.Vals {
			if parVal[v] == nil {
				parVal[v] = map[uint64]map[uint32]bool{}
			}
			if parVal[v][r.TS] == nil {
				parVal[v][r.TS] = map[uint32]bool{}
			}
			parVal[v][r.TS][r.Slot] = true
		}
	}
	total := len(instants)
	var out []objSignature
	for v, parTS := range parVal {
		if float64(len(parTS)) < objMinPartImages*float64(total) {
			continue
		}
		mono, porteurs := 0, map[uint64]bool{}
		for _, slots := range parTS {
			if len(slots) <= 1 {
				mono++
			}
			for s := range slots {
				porteurs[b.SlotXUID[s]] = true
			}
		}
		if float64(mono) < objSeuilMonoPorteur*float64(len(parTS)) || len(porteurs) < objMinPorteursDistincts {
			continue
		}
		out = append(out, objSignature{Val: v, Images: len(parTS), ImagesMonoPorteur: mono, Porteurs: len(porteurs)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Images != out[j].Images {
			return out[i].Images > out[j].Images
		}
		return out[i].Val < out[j].Val
	})
	return out
}

// TestObjectifsPhase0Crane — ITEM 0.3, volet « le crane d'Oddball ».
func TestObjectifsPhase0Crane(t *testing.T) {
	root := objRequireRoot(t)
	src, ok := objOpenFilm(t, root, objBallFilm)
	if !ok {
		t.Skipf("film Oddball %s absent du cache (%s=%q)", objBallFilm, objFilmEnv, root)
	}
	b := objBridgeOf(t, root, objBallFilm)
	recs, images := objRecordsOf(t, root, objBallFilm)
	sigs := objSignaturesPortees(recs, b)
	t.Logf("%s (Oddball) : %d images-cles, %d records bipede ; %d valeurs tiennent la signature "+
		"structurelle (presentes sur >= %.0f %% des images, <= 1 porteur sur >= %.0f %% d'entre "+
		"elles, >= %d porteurs distincts)",
		objBallFilm, images, len(recs), len(sigs), 100*objMinPartImages, 100*objSeuilMonoPorteur,
		objMinPorteursDistincts)
	groupes := objGroupesDecalage(objValsDe(sigs))
	t.Logf("%s : soit %d MOTIFS distincts apres repli des vues decalees", objBallFilm, len(groupes))
	for i, s := range sigs {
		if i >= 8 {
			t.Logf("%s : ... %d autres", objBallFilm, len(sigs)-8)
			break
		}
		t.Logf("%s : signature 0x%08X — %d/%d images, mono-porteur %d/%d, %d porteurs distincts",
			objBallFilm, s.Val, s.Images, images, s.ImagesMonoPorteur, s.Images, s.Porteurs)
	}
	if motif, okm := objMotifRetenu(t, root); okm {
		distrib, img := objSimultaneite(recs, motif)
		t.Logf("%s : LE MOTIF CTF 0x%08X sur le film Oddball — %d images-cles, %s",
			objBallFilm, motif, img, objDistribString(distrib))
	}
	objControleGrossierCrane(t, src, b, recs)
}

// objValsDe extrait les valeurs d'une liste de signatures.
func objValsDe(sigs []objSignature) []uint32 {
	out := make([]uint32, 0, len(sigs))
	for _, s := range sigs {
		out = append(out, s.Val)
	}
	return out
}

// objControleGrossierCrane confronte les signatures au CONTROLE GROSSIER : les evenements
// `th=10` du footer, approximes a 5-20 s, publies comme tels par la chaine de production.
func objControleGrossierCrane(t *testing.T, src *objDiskFilm, b objBridge, recs []objRecord) {
	t.Helper()
	roster := objectiveevents.MapRoster{}
	for _, p := range objCorpus[objBallFilm].Players {
		roster[p.XUID] = p.Team
	}
	evs := objectiveevents.Extract(objBallFilm, "Ranked:Oddball", src, roster)
	acteurs := 0
	for _, e := range evs {
		acteurs += len(e.Players)
	}
	t.Logf("%s : controle grossier th=10 — %d evenements de crane, %d acteurs nommes "+
		"(approximation 5-20 s, publiee comme telle par la chaine de production)",
		objBallFilm, len(evs), acteurs)
	if len(evs) == 0 {
		t.Logf("%s : aucun evenement th=10 — le controle grossier n'a rien a confronter", objBallFilm)
		return
	}
	porteurs := map[uint64]bool{}
	for _, r := range recs {
		if x, ok := b.SlotXUID[r.Slot]; ok {
			porteurs[x] = true
		}
	}
	nommes := 0
	for _, e := range evs {
		for _, p := range e.Players {
			if x, err := strconv.ParseUint(p.XUID, 10, 64); err == nil && porteurs[x] {
				nommes++
			}
		}
	}
	t.Logf("%s : %d/%d acteurs du controle grossier sont des joueurs que le pont bipede nomme",
		objBallFilm, nommes, acteurs)
}
