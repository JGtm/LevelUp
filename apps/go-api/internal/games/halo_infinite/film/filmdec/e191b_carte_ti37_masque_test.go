//go:build research

package filmdec

// e191b_carte_ti37_masque_test.go — LOT 1.9.1 bis, PAS 1 : QUAND LE JEU ECRIT CHAQUE
// COMPOSANT DE ti=37.
//
// # POURQUOI CETTE MESURE EXISTE
//
// Le lot 1.9.1 a etabli qu AUCUN des composants d etat (i10 porteur, i11 mort, i18 au repos,
// i20 deploye, i21 active, i23 createur) n est au masque a l instant de la CREATION d un
// objet d equipement (13 films, 4 583 poses). La question restee ouverte est : OU et QUAND
// le jeu les ecrit-il ? Trois endroits, dont deux portent un masque :
//
//	RECORD NEW     — la creation : le masque dit ce qui accompagne la naissance de l objet ;
//	RECORD DELTA   — la mise a jour : le masque dit ce que le jeu REECRIT en cours de vie ;
//	IMAGE-CLE      — l etat complet : AUCUN masque, tous les composants sont presents (c est
//	                 la definition du cadre d etat complet, lot 1.4), donc rien a mesurer ici.
//
// Cet instrument compte la presence de chaque index de composant au masque des records ti=37,
// separement pour NEW et pour DELTA. C est la colonne « a quel moment » du tableau
// « famille x ce que le jeu ecrit ».
//
// # OU IL LIT, ET POURQUOI PAS SUR LES BOBINES
//
// LES BOBINES PAR BUILD NE PORTENT AUCUN PAQUET DELTA — mesure du 2026-09-15, colonne
// « paquets delta » de la sortie : elles sont faites de `chunk_00`, d un chunk d IMAGE-CLE et
// du pied (V7 du plan). La question du masque ne s y pose donc pas, et un « 0 » y serait un
// artefact du fixture, pas un fait du jeu. L instrument lit donc des FILMS ENTIERS du cache,
// designes par `E191B_FILMS` (repertoires absolus separes par `;`), et SAUTE sans elle.
//
// RESERVE ECRITE, ET ELLE EST IMPORTANTE : un masque ne se lit juste que si la marche du
// record precedent s est terminee au bon bit. La traversee ti=37 ne FERME pas (mesure [1] de
// `TestE191bCarteTI37`), donc ces comptes portent le bruit d une marche qui derive. Ils sont
// un ORDRE DE GRANDEUR, pas une preuve — et c est exactement pourquoi la fermeture doit etre
// traitee AVANT de publier un fait tire d un de ces composants.
//
//	CHUNK00_FILMS='C:/.../film_chunks/a521164d;C:/.../film_chunks/fb1a1a72' \
//	  go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191bMasqueTI37$' -v -count=1

import (
	"path/filepath"
	"testing"
)

// e191bMasqueChunks borne le nombre de chunks de replication balayes par film : la mesure est
// une PROPORTION par masque, pas un inventaire, et lire douze chunks suffit a la stabiliser
// (meme borne que le temoin de marche delta).
const e191bMasqueChunks = 12

// e191bComptesMasque porte les comptes d une population de records.
type e191bComptesMasque struct {
	Records int
	ParIdx  map[int]int
	// Paquets et Tous sont des COMPTEURS DE CONTROLE : sans eux, un « 0 record ti=37 » ne
	// distingue pas « le film n en porte pas » de « la lecture n a rien vu ».
	Paquets int
	Tous    int
}

func TestE191bMasqueTI37(t *testing.T) {
	sources := chunk00Films(t, "CHUNK00_FILMS")
	t.Logf("######## PAS 1 — PRESENCE AU MASQUE DES RECORDS ti=%d (NEW et DELTA) ########", e191bTI)
	neufs := &e191bComptesMasque{ParIdx: map[int]int{}}
	deltas := &e191bComptesMasque{ParIdx: map[int]int{}}
	for _, dir := range sources {
		n, d := e191bMasqueDUnFilm(t, dir)
		t.Logf("  %-12s : %5d paquets delta, %7d records tous archetypes, "+
			"%5d NEW ti=%d, %6d DELTA ti=%d",
			filepath.Base(dir), n.Paquets, n.Tous, n.Records, e191bTI, d.Records, e191bTI)
		e191bAjouterMasque(neufs, n)
		e191bAjouterMasque(deltas, d)
	}
	t.Logf("")
	t.Logf("  %-4s %-42s %16s %16s", "i", "composant", "NEW (n / %)", "DELTA (n / %)")
	for i := 0; i < len(e191bNomsTI37); i++ {
		t.Logf("  i%-3d %-42s %9d %5.1f %%  %9d %5.1f %%", i, e191bNomsTI37[i],
			neufs.ParIdx[i], e191bPct(neufs.ParIdx[i], neufs.Records),
			deltas.ParIdx[i], e191bPct(deltas.ParIdx[i], deltas.Records))
	}
	t.Logf("  TOTAL : %d records NEW, %d records DELTA", neufs.Records, deltas.Records)
}

// e191bPct : pourcentage a denominateur jamais nul.
func e191bPct(n, d int) float64 {
	if d <= 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}

// e191bNomsTI37 : la liste ORDONNEE des composants de ti=37, telle que le registre des 7
// bobines la rend (mesure [4] de `TestE191bCarteTI37`). Recopiee ici pour l AFFICHAGE
// seulement ; la mesure, elle, lit toujours le registre du film.
var e191bNomsTI37 = []string{
	"object-position", "object-translational-velocity", "object-forward-and-up",
	"object-angular-velocity", "object-body-vitality", "object-shield-vitality",
	"object-region-state", "object-damage-sections", "object-constraint",
	"object-multiplayer-properties", "object-parent-state", "object-dead-state",
	"object-scale", "object-maximum-vitalities", "object-dissolver",
	"object-low-frequency", "object-physics-flags", "object-frame-configuration",
	"item-at-rest", "item-ignore-player", "equipment-deployed", "equipment-activated",
	"equipment-control-signal", "equipment-creator", "equipment-energy",
	"equipment-being-hacked", "equipment-energy-delay-ticks-left",
	"equipment-charges-remaining", "equipment-tracked-object-handles-stack",
	"equipment-command-tick", "equipment-has-infinite-uses",
}

// e191bAjouterMasque cumule un film.
func e191bAjouterMasque(dst, src *e191bComptesMasque) {
	dst.Records += src.Records
	dst.Paquets += src.Paquets
	dst.Tous += src.Tous
	for i, n := range src.ParIdx {
		dst.ParIdx[i] += n
	}
}

// e191bMasqueDUnFilm balaye les paquets delta d un film et compte la presence de chaque
// composant au masque des records ti=37, separement pour NEW et pour DELTA.
func e191bMasqueDUnFilm(t *testing.T, dir string) (neufs, deltas *e191bComptesMasque) {
	t.Helper()
	neufs = &e191bComptesMasque{ParIdx: map[int]int{}}
	deltas = &e191bComptesMasque{ParIdx: map[int]int{}}
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Logf("  %s : chunk_00 illisible (%v) — hors mesure", filepath.Base(dir), err)
		return neufs, deltas
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Logf("  %s : registre illisible (%v) — hors mesure", filepath.Base(dir), err)
		return neufs, deltas
	}
	n := CountFilmChunks(dir)
	if n > e191bMasqueChunks {
		n = e191bMasqueChunks
	}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue // un film peut avoir moins de chunks que la borne : ce n est pas une erreur
		}
		e191bMasqueDUnChunk(data, reg, neufs, deltas)
	}
	return neufs, deltas
}

// e191bMasqueDUnChunk amorce le monde par les images-cles du chunk, puis lit ses paquets delta.
func e191bMasqueDUnChunk(data []byte, reg *Registry, neufs, deltas *e191bComptesMasque) {
	w := NewWorld(reg)
	pks := WalkPackets(data)
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI)) //nolint:gosec // slot et gen sont bornes
		}
	}
	cfg := DefaultFrameConfig()
	for _, pk := range pks {
		if pk.Type != PacketTypeDelta {
			continue
		}
		neufs.Paquets++
		br := LecteurSur(pk.Payload(data))
		recs, _ := DecodeFrameRecords(br, w, cfg)
		neufs.Tous += len(recs)
		for i := range recs {
			if recs[i].TypeIndex != e191bTI {
				continue
			}
			dst := deltas
			if recs[i].Type == recNew {
				dst = neufs
			}
			dst.Records++
			e191bCompterMasque(dst, recs[i].Trace.Mask)
		}
	}
}

// e191bCompterMasque incremente un compteur par bit pose du masque.
func e191bCompterMasque(dst *e191bComptesMasque, mask uint64) {
	for i := 0; i < len(e191bNomsTI37); i++ {
		if mask&(uint64(1)<<uint(i)) != 0 {
			dst.ParIdx[i]++
		}
	}
}
