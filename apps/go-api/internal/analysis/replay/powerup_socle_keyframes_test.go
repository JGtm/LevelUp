package replay

// powerup_socle_keyframes_test.go — PHASE 3 : ce que les IMAGES-CLES recensent, et que les
// paquets delta ne peuvent pas montrer.
//
// POURQUOI CETTE PHASE EXISTE. La phase 2 balaye les paquets DELTA : elle ne voit que ce qui
// EMET une position. Un objet pose qui ne bouge jamais n'y figure pas — c'est l'acquis
// commun a `ti=37` et `ti=42`. L'image-cle, elle, est un ETAT COMPLET du monde : chaque
// entite vivante y a un record, qu'elle bouge ou non. C'est la seule fenetre sur H1 (« present
// des t=0, sans record NEW delta ») et sur H3 (« jamais replique »).
//
// CE QUE L'IMAGE-CLE DONNE, ET CE QU'ELLE NE DONNE PAS. `WalkKeyframeWorld` rend le couple
// (slot -> archetype) de chaque record, et RIEN D'AUTRE : aucune position n'en est lisible
// (le corps du record n'est decode de facon bit-exacte par personne — RE arretee apres R7-e).
// La phase ne peut donc pas dire « cet objet est au socle ». Elle peut dire quelque chose de
// plus fort et d'independant : QUAND un slot cesse d'etre recense, et quand il revient.
//
// L'HORLOGE : LES INTERVALLES, PAS LES INSTANTS. Les images-cles sont datees sur l'horloge du
// FILM ; les ramassages de la phase 1 sur la grille du DOCUMENT, dont l'origine (premier
// paquet de POSITION) n'est pas dans l'artefact. Les deux axes different d'une constante
// inconnue de quelques secondes. On ne compare donc PAS des instants : on compare des
// INTERVALLES, qui sont invariants par translation. Les quatre ramassages de `01e1f945` sont
// espaces de 123,2 / 124,6 / 252,9 s — une signature qu'aucune constante ne deplace.
//
// LECTURE SEULE. Garde `OBJ_FILM` + `OBJ_FILM_ART`.
//
//	CGO_ENABLED=0 OBJ_FILM=<depot>/data/cache \
//	  go test ./internal/analysis/replay/ -run '^TestPowerupSocleImagesCles$' -timeout 90m -v

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// psTIRecenses — les archetypes dont on suit le recensement. Memes que la phase 2 : un
// recensement qui ne regarde que `ti=37` ne peut rendre que `ti=37`.
var psTIRecenses = []int{36, 37, 38, 39, 40, 41, 42}

// psKF est UNE image-cle : son instant et ce qu'elle recense.
type psKF struct {
	US uint64
	// Slots liste, par archetype, les slots recenses a cette image-cle.
	Slots map[int]map[uint32]bool
}

// psRecenseKF marche toutes les images-cles du film et rend leur recensement, dans l'ordre.
//
// HORS LIGNE : une seule passe sur les chunks, aucun decodage de corps de record.
func psRecenseKF(dir string) []psKF {
	var out []psKF
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			kf := psKF{US: pk.TimestampUS, Slots: map[int]map[uint32]bool{}}
			for _, r := range filmdec.WalkKeyframeWorld(pk.Payload(data)) {
				if kf.Slots[r.TI] == nil {
					kf.Slots[r.TI] = map[uint32]bool{}
				}
				kf.Slots[r.TI][uint32(r.Slot)] = true
			}
			out = append(out, kf)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].US < out[j].US })
	return out
}

// psPresence est le profil de presence d'UN slot au fil des images-cles.
type psPresence struct {
	TI   int
	Slot uint32
	// Vu[i] : le slot est recense a l'image-cle i.
	Vu []bool
	// Premiere / Derniere : index des images-cles extremes ou il est vu ; Total le compte.
	Premiere, Derniere, Total int
	// Trous : nombre d'intervalles d'absence STRICTEMENT INTERIEURS (encadres par une
	// presence des deux cotes). Un trou interieur est une disparition SUIVIE d'un retour —
	// exactement la forme d'un socle ramasse puis reapparu.
	Trous int
	// RetoursUS : instants des images-cles ou le slot REVIENT apres un trou interieur.
	RetoursUS []uint64
}

// psProfils construit, pour un archetype, le profil de presence de chaque slot recense.
func psProfils(kfs []psKF, ti int) []psPresence {
	slots := map[uint32]bool{}
	for _, kf := range kfs {
		for s := range kf.Slots[ti] {
			slots[s] = true
		}
	}
	out := make([]psPresence, 0, len(slots))
	for s := range slots {
		p := psPresence{TI: ti, Slot: s, Vu: make([]bool, len(kfs)), Premiere: -1, Derniere: -1}
		for i, kf := range kfs {
			if !kf.Slots[ti][s] {
				continue
			}
			p.Vu[i] = true
			p.Total++
			if p.Premiere < 0 {
				p.Premiere = i
			}
			p.Derniere = i
		}
		dedans := false
		for i := p.Premiere; i >= 0 && i <= p.Derniere; i++ {
			switch {
			case !p.Vu[i]:
				dedans = true
			case dedans:
				p.Trous++
				p.RetoursUS = append(p.RetoursUS, kfs[i].US)
				dedans = false
			}
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slot < out[j].Slot })
	return out
}

// psSlotsAvecDelta rend les slots d'un archetype qui ont AU MOINS une position dans les
// paquets delta. Le complementaire — recense aux images-cles, jamais vu en delta — est la
// population « invisible » : la forme meme de H1.
func psSlotsAvecDelta(dir string, wr *filmdec.Vec3Range, ti int) map[uint32]bool {
	out := map[uint32]bool{}
	tracks, err := filmdec.ScanFilmWorldObjects(dir, wr, ti)
	if err != nil {
		return out
	}
	for _, tr := range tracks {
		out[tr.Slot] = true
	}
	return out
}

// TestPowerupSocleImagesCles — 3.1 a 3.3 du plan.
func TestPowerupSocleImagesCles(t *testing.T) {
	root := objRequireRoot(t)
	entry := psEntreeCarte(t)

	for _, f := range psFilmsCatalyst {
		t.Run(f.ID+"_"+f.Mode, func(t *testing.T) {
			dir := filepath.Join(root, "film_chunks", f.ID)
			if filmdec.CountFilmChunks(dir) == 0 {
				t.Skipf("aucun chunk dans %s", dir)
			}
			release := filmdec.LockProcessDecode()
			defer release()
			defer installWorldObjectPrecision(entry, dir)()
			wr := entry.Range()

			kfs := psRecenseKF(dir)
			if len(kfs) == 0 {
				t.Skip("aucune image-cle lisible")
			}
			t0 := kfs[0].US
			t.Logf("=== 3.1 RECENSEMENT : %d images-cles, de %.1f a %.1f s ===",
				len(kfs), 0.0, psSecondes(kfs[len(kfs)-1].US, t0))
			for _, ti := range psTIRecenses {
				psRapportTI(t, kfs, dir, &wr, ti, t0)
			}
		})
	}
}

// psRapportTI ecrit le recensement d'UN archetype : ses denominateurs, sa population
// invisible, et les slots a trou interieur.
func psRapportTI(t *testing.T, kfs []psKF, dir string, wr *filmdec.Vec3Range, ti int, t0 uint64) {
	t.Helper()
	profils := psProfils(kfs, ti)
	if len(profils) == 0 {
		t.Logf("  ti=%2d %-20s : aucun slot recense", ti, psNomTI(ti))
		return
	}
	avecDelta := psSlotsAvecDelta(dir, wr, ti)
	var desLeDebut, partout, invisibles, aTrou int
	var candidats []psPresence
	for _, p := range profils {
		if p.Premiere == 0 {
			desLeDebut++
		}
		if p.Total == len(kfs) {
			partout++
		}
		if !avecDelta[p.Slot] {
			invisibles++
		}
		if p.Trous > 0 {
			aTrou++
			if !avecDelta[p.Slot] {
				candidats = append(candidats, p)
			}
		}
	}
	t.Logf("  ti=%2d %-20s : %4d slots recenses | des la 1re image %3d | a TOUTES %3d"+
		" | SANS position delta %3d | a trou interieur %3d",
		ti, psNomTI(ti), len(profils), desLeDebut, partout, invisibles, aTrou)
	psRapportCandidats(t, candidats, kfs, t0)
}

// psRapportCandidats detaille les slots SANS position delta qui disparaissent puis
// reviennent — les seuls profils qu'un socle ramasse puis reapparu puisse produire.
func psRapportCandidats(t *testing.T, candidats []psPresence, kfs []psKF, t0 uint64) {
	t.Helper()
	if len(candidats) == 0 {
		return
	}
	sort.Slice(candidats, func(i, j int) bool { return candidats[i].Trous > candidats[j].Trous })
	for i, p := range candidats {
		if i >= 12 {
			t.Logf("      (... %d slots a trou de plus)", len(candidats)-i)
			return
		}
		retours := make([]float64, 0, len(p.RetoursUS))
		for _, us := range p.RetoursUS {
			retours = append(retours, psSecondes(us, t0))
		}
		t.Logf("      slot %4d | vu %2d/%2d images-cles | %d trou(s) | retours a %.0f s",
			p.Slot, p.Total, len(kfs), p.Trous, retours)
	}
}
