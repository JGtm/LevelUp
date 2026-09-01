package filmdec

// objectif_ti11_ancre_test.go — LA GRAMMAIRE DE L'EN-TETE DE RECORD, LUE DANS LE JEU PUIS MESUREE.
//
// # CE QUE GHIDRA A RENDU (2026-09-01), ET POURQUOI CA DEPLACE LE CHANTIER
//
// Le lecteur d'en-tete de record du jeu est `FUN_141f85fe0`, appele par les deux lecteurs de
// record `FUN_142e309b4` et `FUN_142e30b9c` (tous deux enchainent ensuite sur `FUN_14076cb60`, la
// boucle de composants). Sa grammaire :
//
//	R(6)                                  -> typeIndex de l'archetype
//	FUN_1406d3140(ti, lecteur, 7, &id)    -> l'identifiant d'entite
//	FUN_1406d676c(lecteur)                -> alignement
//	valide SSI typeIndex < 0x32 ET id != -1 ET position <= taille x 8
//
// Et `FUN_1406d3140` decompose l'identifiant ainsi :
//
//	largeur = FUN_1406d310c(capacite)     -> CEIL(LOG2(capacite)), lu dans une table par pool
//	R(largeur)                            -> l'index de slot, auquel s'ajoute une base
//	R(2)                                  -> la GENERATION
//	id = generation << 30 | (base + index)
//
// # LE POINT QUI COMPTE : LA LARGEUR DU SLOT N'EST PAS UNE CONSTANTE
//
// Elle vaut `ceil(log2(capacite du pool))`, et cette capacite est posee AU CHARGEMENT DE CARTE :
// la table `DAT_1451f98d0` est NULLE dans le binaire statique. C'est exactement la classe de
// probleme que le depot documente deja pour les composants a precision runtime — « des largeurs
// peuplees au chargement de carte et absentes du .exe statique ».
//
// Le depot, lui, ancre avec une largeur FIXE : `matchWorldObjectRecord` lit `R(1)` de prefixe,
// `R(13)` de slot, `R(2)` de generation. Si la vraie largeur n'est pas 13, chaque `rec.After` est
// decale — et tout ce qui suit avec lui. Or l'ancre est le levier dominant : la bande d'ancrage a
// deja multiplie le chainage par cinq a vingt.
//
// # L'EPREUVE
//
// La largeur ne se lit pas dans le binaire, donc elle se MESURE : balayer la largeur du slot et la
// largeur de la porte de masque, et lire le chainage. La combinaison qui domine EST la grammaire.
// Le PLANCHER de reference vaut 3 % (`TestObjectifTi11DeltaTemoin`) : une combinaison qui ne le
// depasse pas franchement ne dit rien.
//
// La configuration actuelle du depot — slot 13, porte 2 — est dans le balayage, a sa place, sans
// traitement de faveur.
//
// # LE VERDICT : L'ANCRE DU DEPOT EST CONFIRMEE, PAS CORRIGEE (mesure du 2026-09-01)
//
//	slot=12 porte=1  13,1 %      slot=14 porte=1  12,9 %      slot=16 porte=1  12,7 %
//	slot=12 porte=2   7,1 %      slot=14 porte=2  **20,9 %**  slot=16 porte=2   9,8 %
//	slot=13 porte=1  14,1 %      slot=15 porte=1  15,7 %
//	slot=13 porte=2   7,7 %      slot=15 porte=2  13,2 %
//
// `slot=14 porte=2` domine nettement, a nombre de marches EGAL (37 901 contre 37 902 pour
// `slot=13 porte=2`) — ce n'est donc pas un effet de selectivite.
//
// ET CE GAGNANT EST L'EN-TETE DU DEPOT. `matchWorldObjectRecord` lit `R(1)` de prefixe + `R(13)`
// de slot + `R(2)` de generation + `R(2)` de porte + `R(3)` de compte, soit **21 bits** — la meme
// largeur totale que `slot=14 porte=2` de ce balayage. Le candidat « slot=13 » est un en-tete de
// VINGT bits, et il perd.
//
// LES DEUX SOURCES CONCORDENT : Ghidra donne `ceil(log2(capacite))` pour la largeur du slot, et
// 14 = ceil(log2(16 384)). Le decoupage du depot en `1 + 13` est la meme lecture qu'un `R(14)`
// dont on exige le bit de tete a 1 — et cette exigence supplementaire est ce qui fait passer le
// chainage de 20,9 % (ici, sans elle) a 29,3 % (`TestObjectifTi11DeltaChainage`, avec elle).
//
// **L'ancre n'est donc pas le trou.** Ce balayage ferme une piste, et c'est son utilite : il
// interdit d'y revenir.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11Ancre -v -timeout 60m

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// ti11AncreCand est une grammaire d'en-tete candidate.
type ti11AncreCand struct {
	slotBits  int // largeur du champ de slot, prefixe compris
	porteBits int // largeur de la porte qui precede le compte du masque
}

// ti11AncreCands : les grammaires balayees. `ceil(log2(capacite))` pour une capacite de 4 096 a
// 65 536 donne 12 a 16 ; la porte est balayee de 1 a 3 autour des 2 bits du depot.
func ti11AncreCands() []ti11AncreCand {
	var out []ti11AncreCand
	for _, s := range []int{12, 13, 14, 15, 16} {
		for _, g := range []int{1, 2, 3} {
			out = append(out, ti11AncreCand{slotBits: s, porteBits: g})
		}
	}
	return out
}

// TestObjectifTi11Ancre balaie les grammaires d'en-tete et rend le chainage de chacune.
func TestObjectifTi11Ancre(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11Ancre", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — balayage interrompu", float64(peak)/(1<<30))
	})
	defer func() { g.Disarm() }()

	type film struct {
		arch Archetype
		band map[uint32]bool
		pays [][]byte
	}
	var films []film
	for _, f := range ti11Corpus {
		dir := filepath.Join(cache, "film_chunks", f.id)
		n := CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		arch, _, err := objectiveArchetype(dir)
		if err != nil {
			continue
		}
		band := objectiveSlotSet(dir, n)
		if len(band) == 0 {
			continue
		}
		x := film{arch: arch, band: band}
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type == PacketTypeDelta {
					x.pays = append(x.pays, pk.Payload(data))
				}
			}
		}
		films = append(films, x)
	}
	t.Logf("corpus : %d film(s) ; PLANCHER de reference : 3 %% ; le depot lit slot=13 porte=2", len(films))

	for _, cand := range ti11AncreCands() {
		marches, chaines := 0, 0
		for _, x := range films {
			for _, pay := range x.pays {
				m, c := ti11AncrePayload(pay, x.band, x.arch, cand)
				marches += m
				chaines += c
			}
		}
		marque := ""
		if cand.slotBits == 13 && cand.porteBits == 2 {
			marque = "   <- lecture actuelle du depot"
		}
		t.Logf("slot=%2d porte=%d : %7d marche(s), %5.1f %% chainees%s",
			cand.slotBits, cand.porteBits, marches, ti11Part(chaines, marches), marque)
	}
}

// ti11AncrePayload ancre et marche les records d'UN payload sous la grammaire candidate.
func ti11AncrePayload(pay []byte, band map[uint32]bool, arch Archetype, cand ti11AncreCand) (int, int) {
	total := len(pay) * 8
	entete := cand.slotBits + 2 + cand.porteBits + 3 // slot + generation + porte + compte
	limit := total - (entete + worldObjectIndexBits)
	marches, chaines := 0, 0
	for p := 0; p <= limit; p++ {
		rec, ok := ti11AncreMatch(pay, p, band, cand, entete)
		if !ok || !ti11IdxDansDomaine(rec.Idx, len(arch.Components)) {
			continue
		}
		fin, done := ti11MarcheRecord(pay, rec, arch, total)
		p = rec.After
		if !done {
			continue
		}
		marches++
		if ti11AncreEnTeteA(pay, fin, cand, entete) {
			chaines++
		}
	}
	return marches, chaines
}

// ti11AncreMatch est `matchWorldObjectRecord` parametre par la grammaire candidate.
//
// LE SLOT EST LU SUR `slotBits` BITS EN PARTANT DE LA POSITION, sans prefixe separe : le depot
// decoupe `R(1) prefixe + R(13) slot`, ce qui est la MEME lecture qu'un `R(14)` de slot dont on
// exigerait le bit de tete a 1. Le balayage n'impose donc rien — il compare des decoupages.
func ti11AncreMatch(pay []byte, p int, band map[uint32]bool, cand ti11AncreCand, entete int) (WorldObjectRecord, bool) {
	var rec WorldObjectRecord
	slot := uint32(PeekBits(pay, p, cand.slotBits))
	// La bande est indexee sur les slots du depot (13 bits utiles) : on compare sur ces bits-la,
	// quel que soit le decoupage teste. Sinon le balayage comparerait des bandes differentes.
	if !band[slot&0x1FFF] {
		return rec, false
	}
	at := p + cand.slotBits
	rec.Gen = uint32(PeekBits(pay, at, 2))
	at += 2
	if PeekBits(pay, at, cand.porteBits) != 0 {
		return rec, false
	}
	at += cand.porteBits
	mc := int(PeekBits(pay, at, 3))
	if mc < 1 || mc > worldObjectMaxMaskCnt {
		return rec, false
	}
	idx, ok := ascendingComponents(pay, p+entete, mc)
	if !ok {
		return rec, false
	}
	rec.Slot = slot & 0x1FFF
	rec.Idx, rec.After = idx, p+entete+worldObjectIndexBits*mc
	return rec, true
}

// ti11AncreEnTeteA est `worldObjectHeaderAt` sous la grammaire candidate, SANS exiger la bande —
// le chainage mesure si la LARGEUR LUE tombe juste, pas qui parle ensuite.
func ti11AncreEnTeteA(pay []byte, p int, cand ti11AncreCand, entete int) bool {
	total := len(pay) * 8
	if p < 0 || p+entete+worldObjectIndexBits > total {
		return false
	}
	at := p + cand.slotBits + 2
	if PeekBits(pay, at, cand.porteBits) != 0 {
		return false
	}
	mc := int(PeekBits(pay, at+cand.porteBits, 3))
	if mc < 1 || mc > worldObjectMaxMaskCnt || p+entete+worldObjectIndexBits*mc > total {
		return false
	}
	_, ok := ascendingComponents(pay, p+entete, mc)
	return ok
}
