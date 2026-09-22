//go:build research

package grammar

// mouvement_5_3_2_research_test.go — LA PREUVE SUR FILM DU LOT 5.3 (point 5.3.2).
//
// # CE QU IL MESURE
//
// Pour CHAQUE record bipede DELTA d UN film, la valeur des champs de mouvement, lue a la
// position que la marche de production publie (`CompResult.StartBit`) :
//
//	i1  object-translational-velocity   direction (19 b, cubemap) + magnitude (10 b)
//	i18 unit-control                    les deux index bornes ET LE MOT DE 32 BITS `+0x544`
//	i29 unit-crouch                     booleen accroupi + progression (10 b, [0,1])
//	i54 biped-mobility-action           flag1/flag2, l identifiant optionnel, `+0x98` R(7), `+0x9c` R(2)
//	i55 biped-posture-physics           le tag de 2 bits
//	i62 biped-slide                     booleen glissade
//
// # POURQUOI UNE MARCHE COMPLETE, ET PAS LE BALAYAGE DE PRODUCTION
//
// DECOUVERTE DU LOT (D8) : le balayage bipede de production (`scanRecordDirs`) ne modelise
// QUE i1, i2, i3, i4, i5 et i21, et s ARRETE au premier composant hors de cette liste
// (« composant non modelise -> curseur non fiable »). Les composants de mouvement, tous situes
// au-dela d i21, ne sont donc JAMAIS lus sur le chemin delta en production. Cet instrument
// rejoue la VRAIE boucle de composants (`traverseComponentLoopFrom`, celle de l image-cle et du
// record NEW) sur les records delta : aucune grammaire n est recopiee, c est la production qui
// marche, seule la POSITION DE DEPART change.
//
// # CE QU IL N OUVRE PAS
//
//	Aucune base DuckDB. Aucun artefact. UN SEUL film par execution.
//
//	MOUV532_FILM=<chemin du repertoire de chunks> [MOUV532_TSV=<sortie>] \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	    -run '^TestMouvement532$' -count=1 -v -timeout 60m

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// Les noms des composants suivis. Le nom est la clef : l index varie d un archetype a l autre.
const (
	m532Velocite = "object-translational-velocity-dynamic-precision"
	m532Controle = "unit-control"
	m532Crouch   = "unit-crouch"
	m532Mobilite = "biped-mobility-action"
	m532Posture  = "biped-posture-physics"
	m532Slide    = "biped-slide"
)

// m532Nom normalise le nom d un composant en retirant le suffixe `-component`.
//
// POURQUOI C EST NECESSAIRE, ET COMMENT LE PIEGE A ETE VU : le registre d un film peut nommer
// un composant SANS le suffixe — le dispatch accepte les deux orthographes (`dispatch_biped.go`
// porte `case "biped-mobility-action", "biped-mobility-action-component"`), et `ecs_table.tsv`
// consigne ces alias en ligne `-1`. Une premiere passe qui ne comparait qu au nom long a vu
// `i54` 5 fois la ou le MASQUE le declarait 487 fois : c est l ecart entre les deux comptes qui
// a revele le piege, et c est pour cela que le masque est mesure a part (m532Masques).
func m532Nom(n string) string { return strings.TrimSuffix(n, "-component") }

// m532Ech est UN record bipede delta, reduit a ce que le lot mesure.
type m532Ech struct {
	slot           uint32
	tUS            uint64
	marcheComplete bool   // la boucle de composants est allee au bout (DesyncAt == -1)
	bloquant       string // le composant qui a arrete la marche, vide si elle est allee au bout
	masque         uint64 // le masque de composants du record, pour le DENOMINATEUR du negatif
	// i1
	aVitesse bool
	dirZ     float32 // composante verticale de la DIRECTION unitaire
	magQ     uint32  // magnitude quantifiee (10 b), monotone en vitesse
	// i18
	aControle bool
	aMot32    bool
	mot32     uint32
	// i29
	aCrouch bool
	crouch  bool
	crouchQ uint32 // progression 0..1023
	// i54
	aMobilite bool
	flag1     bool
	flag2     bool
	aIdent    bool
	ident     uint32 // l identifiant optionnel de 10 bits
	queue7    uint32 // `+0x98`
	queue2    uint32 // `+0x9c`
	// i55
	aPosture bool
	tag      uint32
	// i62
	aSlide bool
	slide  bool
}

// TestMouvement532 — LA MESURE, UN FILM A LA FOIS.
func TestMouvement532(t *testing.T) {
	dir := os.Getenv("MOUV532_FILM")
	if dir == "" {
		t.Skip("MOUV532_FILM absent : chemin du repertoire de chunks du film attendu")
	}
	ech, kf, tzero := m532Lire(t, dir)
	if len(ech) == 0 {
		t.Fatalf("aucun record bipede delta lu dans %s", dir)
	}
	t.Logf("FILM %s : %d records bipedes DELTA (%d slots) · %d records bipedes d IMAGE-CLE",
		dir, len(ech), m532NbSlots(ech), len(kf))
	m532Couverture(t, ech)
	m532Masques(t, ech)
	m532Etats(t, ech)
	m532TableauMobilite(t, ech)
	m532Postures(t, ech)
	m532BitsDeControle(t, ech)
	if len(kf) > 0 {
		t.Logf("---- LES ETATS VOYAGENT A L IMAGE-CLE : meme tableau, sur les %d records d image-cle ----", len(kf))
		m532Couverture(t, kf)
		m532Etats(t, kf)
		m532Postures(t, kf)
		m532BitsDeControle(t, kf)
	}
	m532Instants(t, append(append([]m532Ech(nil), ech...), kf...), tzero)
	if out := os.Getenv("MOUV532_TSV"); out != "" {
		m532EcrireTSV(t, append(append([]m532Ech(nil), ech...), kf...), tzero, out)
	}
}

// ---------------------------------------------------------------------------
// LA LECTURE
// ---------------------------------------------------------------------------

//nolint:gocyclo // instrument de recherche : une seule marche, lisible de haut en bas
func m532Lire(t *testing.T, dir string) ([]m532Ech, []m532Ech, uint64) {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	arch, ok := reg.Archetype(BipedTypeIndex)
	if !ok {
		t.Fatalf("archetype ti=%d absent du registre", BipedTypeIndex)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	opt := ScanFilmOptions{RequireTag1: os.Getenv("MOUV532_TAG1") != "0", DropSaturated: true, QuantaOnly: true}
	chunks, err := bipedScanChunks(film, opt)
	if err != nil {
		t.Fatalf("chunks : %v", err)
	}
	band := bipedSlotBand(film, chunks)
	if band.Count() == 0 {
		t.Fatalf("aucun slot bipede dans les images-cles")
	}
	lay, err := bipedI0Layout(film, opt)
	if err != nil {
		t.Fatalf("layout i0 : %v", err)
	}
	m532Archetype(t, arch)
	ctx := fc.ContexteDeLecture()
	i0Bits := lay.TotalBits()

	var out, kf []m532Ech
	var tzero uint64
	for _, c := range chunks {
		data, pks, okc := FilmChunkAt(film, c)
		if !okc {
			continue
		}
		for _, pk := range pks {
			if tzero == 0 || pk.TimestampUS < tzero {
				tzero = pk.TimestampUS
			}
			pay := pk.Payload(data)
			ts := pk.TimestampUS
			switch pk.Type {
			case PacketTypeDelta:
				walkDeltaBipedPayload(pay, band, lay, opt.RequireTag1, func(r deltaBipedRecord) {
					out = append(out, m532Record(pay, r, arch, ctx, i0Bits, ts))
				})
			case PacketTypeKeyframe:
				kf = append(kf, m532Keyframe(pay, reg, ctx, ts)...)
			}
		}
	}
	return out, kf, tzero
}

// m532Keyframe lit les records BORNES de `ti=35` d un payload d image-cle, par la marche d etat
// complet de production. C EST LA QUE LES ETATS VOYAGENT : le chemin delta ne porte, de tout ce
// que ce lot suit, que l initiation d action de mobilite (mesure de bfecd02b).
func m532Keyframe(pay []byte, reg *Registry, ctx ContexteDeLecture, ts uint64) []m532Ech {
	var out []m532Ech
	total := len(pay) * 8
	for _, b := range keyframeBornes(pay) {
		if b.TI != BipedTypeIndex {
			continue
		}
		tr := WalkKeyframeFullState(pay, b.Bit, reg, ctx)
		e := m532Ech{slot: uint32(b.Slot), tUS: ts, masque: tr.Mask} //nolint:gosec // slot d un record
		e.marcheComplete = tr.DesyncAt == -1 && tr.EndBit <= total
		if tr.DesyncAt >= 0 {
			e.bloquant = nomComposantBloquant(reg, BipedTypeIndex, tr.DesyncAt)
		}
		for i, comp := range tr.Comps {
			fin := tr.EndBit
			if i+1 < len(tr.Comps) {
				fin = tr.Comps[i+1].StartBit
			}
			if comp.StartBit < 0 || fin > total || fin < comp.StartBit {
				continue
			}
			m532Champ(pay, comp, fin, &e)
		}
		out = append(out, e)
	}
	return out
}

// m532Archetype publie l INDEX DE CHAQUE COMPOSANT SUIVI DANS CE FILM.
//
// L INDEX N EST PAS UNE CONSTANTE DU JEU : il est celui du registre DU FILM. `ecs_table.tsv`
// donne ceux du build sur lequel elle a ete faite ; un autre film peut ranger le meme nom
// ailleurs. Tout ce lot compte par NOM, et cette ligne donne la correspondance pour que les
// chiffres du masque (qui, eux, sont par index) soient lisibles.
func m532Archetype(t *testing.T, arch Archetype) {
	t.Helper()
	suivis := map[string]bool{m532Velocite: true, m532Controle: true, m532Crouch: true,
		m532Mobilite: true, m532Posture: true, m532Slide: true}
	var parts []string
	for i, n := range arch.Components {
		if suivis[m532Nom(n)] {
			parts = append(parts, fmt.Sprintf("i%d=%s", i, m532Nom(n)))
		}
	}
	t.Logf("ARCHETYPE ti=%d DE CE FILM (%d composants) — index des composants suivis : %s",
		BipedTypeIndex, len(arch.Components), strings.Join(parts, " "))
}

// m532Record rejoue la boucle de composants de PRODUCTION sur un record delta, puis relit les
// champs suivis a la position que la marche publie.
func m532Record(pay []byte, r deltaBipedRecord, arch Archetype, ctx ContexteDeLecture,
	i0Bits int, ts uint64) m532Ech {
	e := m532Ech{slot: r.Slot, tUS: ts, masque: m532Masque(r.Mask)}
	br := LecteurSur(pay)
	br.PoserContexte(ctx)
	br.SetBitPos(r.I0 + i0Bits + i0TailBits)
	tr := EntityTrace{DesyncAt: -1, TypeIndex: BipedTypeIndex, Mask: e.masque}
	traverseComponentLoopFrom(br, arch, &tr, 1)
	// `traverseComponentLoopFrom` NE POSE PAS `EndBit` — c est `TraverseEntity` qui le fait
	// apres elle. Sans cette ligne, la fin du DERNIER composant vaut 0, et le dernier composant
	// du record n est jamais lu : `i54`, qui est le plus haut index de 459 records de ce film,
	// etait vu 5 fois au lieu de 487. Defaut de ce harnais, pas de la production.
	tr.EndBit = br.BitPos()
	total := len(pay) * 8
	// UNE MARCHE N EST EXPLOITABLE QUE SI ELLE EST ALLEE AU BOUT **ET** EST RESTEE DANS LE
	// PAYLOAD. Un en-tete accepte par erreur par le balayeur d ancres produit une marche qui
	// part hors des octets : ses positions ne designent rien, et les lire donnerait du bruit
	// presente comme une mesure. Ces records sont COMPTES (`m532Ech` vide) et jamais lus.
	e.marcheComplete = tr.DesyncAt == -1 && tr.EndBit <= total
	if !e.marcheComplete {
		return e
	}
	for i, comp := range tr.Comps {
		fin := tr.EndBit
		if i+1 < len(tr.Comps) {
			fin = tr.Comps[i+1].StartBit
		}
		if comp.StartBit < 0 || fin > total || fin < comp.StartBit {
			continue
		}
		m532Champ(pay, comp, fin, &e)
	}
	return e
}

// m532Champ lit UN composant suivi a sa position publiee. Les grammaires sont celles des
// deserialiseurs portes (`unit_weaponstate.go`, `unit_control.go`, `components_walk_batch9.go`,
// `components_biped_spartan.go`, `components_biped_ability.go`) — relues, pas redevinees.
func m532Champ(pay []byte, comp CompResult, fin int, e *m532Ech) {
	at := comp.StartBit
	switch m532Nom(comp.Name) {
	case m532Velocite:
		// R(1) brut ; sinon R(1) absent ; sinon R(19) direction + R(10) magnitude.
		if b, ok := m532Lit(pay, at, 1); !ok || b == 1 {
			return
		}
		if b, ok := m532Lit(pay, at+1, 1); !ok || b == 1 {
			return
		}
		raw, ok := m532Lit(pay, at+2, int(aimDirBits))
		if !ok {
			return
		}
		mag, okm := m532Lit(pay, at+2+int(aimDirBits), velScaleBits)
		if !okm {
			return
		}
		if v, okv := DecodeAimVectorChecked(raw, aimDirBits); okv {
			e.aVitesse, e.dirZ, e.magQ = true, v[2], mag
		}
	case m532Controle:
		e.aControle = true
		p := at + 1
		g, ok := m532Lit(pay, at, 1)
		if !ok {
			return
		}
		if g == 1 {
			p += 5
			f, okf := m532Lit(pay, p, 1)
			if !okf {
				return
			}
			if f == 1 {
				p += 7
			} else {
				p++
			}
		}
		g2, ok2 := m532Lit(pay, p, 1)
		if !ok2 || g2 != 1 {
			return
		}
		if w, okw := m532Lit(pay, p+1, 32); okw {
			e.aMot32, e.mot32 = true, w
		}
	case m532Crouch:
		b, ok := m532Lit(pay, at, 1)
		q, okq := m532Lit(pay, at+1, 10)
		if !ok || !okq {
			return
		}
		e.aCrouch, e.crouch, e.crouchQ = true, b == 1, q
	case m532Posture:
		if v, ok := m532Lit(pay, at, 2); ok {
			e.aPosture, e.tag = true, v
		}
	case m532Slide:
		if b, ok := m532Lit(pay, at, 1); ok {
			e.aSlide, e.slide = true, b == 1
		}
	case m532Mobilite:
		m532ChampMobilite(pay, at, fin, e)
	}
}

// m532Lit est la SEULE porte de lecture de cet instrument : elle refuse tout ce qui deborde du
// payload au lieu de paniquer. Un debordement n est pas un incident a rattraper, c est le signe
// qu une position ne designe rien — et une valeur lue hors des octets serait du bruit presente
// comme une mesure.
func m532Lit(pay []byte, at, n int) (uint32, bool) {
	if at < 0 || n <= 0 || n > 32 || at+n > len(pay)*8 {
		return 0, false
	}
	return readBitsAt(pay, at, n), true
}

// m532ChampMobilite lit i54. flag1 et flag2 sont en TETE ; l identifiant optionnel suit le
// handle, dont la largeur est CONSTANTE (categorie 0, donc aucun bit de sonde) ; les deux
// champs de queue se lisent depuis la FIN du composant, ou ils sont a position fixe :
// `... R(1) R(7) R(2) R(1)`.
func m532ChampMobilite(pay []byte, at, fin int, e *m532Ech) {
	f1, ok1 := m532Lit(pay, at, 1)
	f2, ok2 := m532Lit(pay, at+1, 1)
	if !ok1 || !ok2 {
		return
	}
	e.aMobilite, e.flag1, e.flag2 = true, f1 == 1, f2 == 1
	if !e.flag1 {
		return
	}
	// handle : R(1) porte + varWidthBits(0) bits + R(2) de queue, puis la tete du corps.
	p := at + 2
	g, okg := m532Lit(pay, p, 1)
	if !okg {
		return
	}
	if g == 1 {
		p += 1 + int(varWidthBits(0)) + 2
	} else {
		p++
	}
	if gi, oki := m532Lit(pay, p, 1); oki && gi == 1 {
		if id, okid := m532Lit(pay, p+1, 10); okid {
			e.aIdent, e.ident = true, id
		}
	}
	if fin-at >= 11 {
		if q7, ok7 := m532Lit(pay, fin-10, 7); ok7 {
			e.queue7 = q7
		}
		if q2, okq2 := m532Lit(pay, fin-3, 2); okq2 {
			e.queue2 = q2
		}
	}
}

func m532Masque(idx []int) uint64 {
	var m uint64
	for _, i := range idx {
		if i >= 0 && i < 64 {
			m |= uint64(1) << uint(i)
		}
	}
	return m
}
