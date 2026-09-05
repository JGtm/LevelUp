package main

// armes_noms.go — NOMMER les candidats sans deviner : (1) chaine sonore weap -> snd!/lsnd ->
// sbnk, confrontee au FNV-1 des noms de packs de tourelle (`tur_un_machinegun`,
// `tur_un_gausscannon`, `tur_un_rocketturret`... — convention etablie par cmd/weapon-sounds) ;
// (2) brute-force murmur3 (mapvar.LabelHash) d'un vocabulaire genere sur les StringId de noeuds
// (noeud d'attache de l'arme du chassis warthog). Aucun module n'est necessaire pour (2).
import (
	"fmt"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himap"
)

// fnv1 : FNV-1 32 bits sur le nom en minuscules (meme convention que cmd/weapon-sounds/noms.go).
func fnv1(s string) uint32 {
	h := uint32(2166136261)
	for _, c := range []byte(strings.ToLower(s)) {
		h *= 16777619
		h ^= uint32(c)
	}
	return h
}

// packsTourelles : noms de packs `.pck` de tourelles/vehicules UNSC (scout sons 2026-08-31).
var packsTourelles = []string{
	"tur_un_machinegun", "tur_un_gausscannon", "tur_un_rocketturret", "tur_cv_plasmacannon",
	"tur_cv_shadeturret", "veh_un_rockethog", "veh_un_warthog", "veh_un_wargoose", "veh_un_wasp",
	"veh_un_scorpion", "veh_un_falconlmgturret", "veh_un_razorback", "veh_un_gausshog",
	"veh_un_warthog_gauss", "veh_un_warthog_rocket", "veh_un_mongoose",
}

// dicoBanques : FNV-1 -> nom, avec et sans prefixe `sb_010_` / `sb_009_`.
func dicoBanques() map[uint32]string {
	d := map[uint32]string{}
	for _, p := range packsTourelles {
		for _, pre := range []string{"", "sb_010_", "sb_009_"} {
			d[fnv1(pre+p)] = pre + p
		}
	}
	return d
}

// chaineSonore suit weap -> (snd!|lsnd) -> sbnk et imprime les GlobalID de banques atteints,
// nommes quand leur GlobalID est le FNV-1 d'un nom de pack connu.
func chaineSonore(idx *himap.ModuleIndex, weapID uint32) {
	tag, err := idx.Extract(weapID)
	if err != nil {
		fmt.Printf("SONS weap %#08x : extraction KO (%v)\n", weapID, err)
		return
	}
	dico := dicoBanques()
	sons := refsDuGroupe(idx, tag, weapID, map[string]bool{"snd!": true, "lsnd": true})
	var banques []string
	vus := map[uint32]bool{}
	for _, s := range sons {
		st, err := idx.Extract(s)
		if err != nil {
			continue
		}
		for _, b := range refsDuGroupe(idx, st, s, map[string]bool{"sbnk": true}) {
			if vus[b] {
				continue
			}
			vus[b] = true
			wwise := idWwise(idx, b)
			nom := dico[wwise]
			if nom == "" {
				nom = "?"
			}
			banques = append(banques, fmt.Sprintf("%#08x(bkhd %#08x)=%s", b, wwise, nom))
		}
	}
	sort.Strings(banques)
	fmt.Printf("SONS weap %#08x : %d sons, banques [%s]\n", weapID, len(sons), strings.Join(banques, " "))
}

// idWwise lit l'identifiant Wwise d'un tag `sbnk` (chunk BKHD : magic(4) taille(4) version(4)
// id(4)) — c'est lui qui vaut FNV-1 du nom de pack (calibrage cmd/weapon-sounds/mapping.go).
func idWwise(idx *himap.ModuleIndex, sbnk uint32) uint32 {
	b, err := idx.Extract(sbnk)
	if err != nil {
		return 0
	}
	i := strings.Index(string(b), "BKHD")
	if i < 0 || i+16 > len(b) {
		return 0
	}
	o := i + 12
	return uint32(b[o]) | uint32(b[o+1])<<8 | uint32(b[o+2])<<16 | uint32(b[o+3])<<24
}

// refsDuGroupe : balayage d'octets, refs vers des tags des groupes voulus (tout ID, dedupliques).
func refsDuGroupe(idx *himap.ModuleIndex, tag []byte, self uint32, groupes map[string]bool) []uint32 {
	var out []uint32
	vus := map[uint32]bool{}
	for o := 0; o+4 <= len(tag); o += 4 {
		h := uint32(tag[o]) | uint32(tag[o+1])<<8 | uint32(tag[o+2])<<16 | uint32(tag[o+3])<<24
		if h == 0 || h == 0xffffffff || h == self || vus[h] {
			continue
		}
		if g, _, ok := idx.Lookup(h); ok && groupes[g] {
			vus[h] = true
			out = append(out, h)
		}
	}
	return out
}

// Vocabulaire genere pour le brute-force murmur3 des noms de noeuds/marqueurs.
var (
	prefixesNoeud = []string{"", "b_", "w_", "m_", "n_", "j_", "bip01_", "warthog_", "hog_", "turret_", "gun_", "gunner_",
		"seat_", "mount_", "attach_", "marker_", "weapon_", "primary_", "secondary_", "vehicle_", "veh_", "rear_", "back_", "chaingun_", "laag_", "gauss_", "rocket_"}
	coeursNoeud = []string{"turret", "gun", "gunner", "mount", "attach", "attachment", "weapon", "pivot", "base", "yaw", "pitch",
		"hardpoint", "socket", "seat", "chaingun", "laag", "rocket", "rockets", "gauss", "cannon", "mg", "machinegun", "node", "marker",
		"point", "root", "body", "chassis", "frame", "hull", "bed", "cargo", "rear", "back", "aft", "top", "roof", "trigger", "fire",
		"muzzle", "barrel", "ring", "pedestal", "stand", "tripod", "cage", "rollcage", "roll_cage", "bumper", "winch", "hood", "hatch",
		"door", "wheel", "axle", "suspension", "steering", "spare", "tire", "light", "headlight", "exhaust", "engine", "grill",
		"passenger", "driver", "control", "child", "object", "arm", "swivel", "gimbal", "shield", "plate", "handle", "grip"}
	suffixesNoeud = []string{"", "_yaw", "_pitch", "_base", "_mount", "_attach", "_marker", "_node", "_point", "_00", "_01", "_02", "_1", "_2",
		"01", "1", "_l", "_r", "_left", "_right", "_front", "_rear", "_back", "_a", "_b", "_top", "_bottom", "_center", "_ctrl", "_control"}
)

// bruteForceNoeuds tente de nommer des StringId (murmur3) par un vocabulaire genere
// prefixe+coeur+suffixe (~60k noms). Imprime les resolutions trouvees.
func bruteForceNoeuds(hashes []uint32) {
	cible := map[uint32]bool{}
	for _, h := range hashes {
		cible[h] = true
	}
	n, trouves := 0, 0
	for _, p := range prefixesNoeud {
		for _, c := range coeursNoeud {
			for _, s := range suffixesNoeud {
				nom := p + c + s
				n++
				if h := uint32(mapvar.LabelHash(nom)); cible[h] {
					fmt.Printf("NOM %#08x = %q\n", h, nom)
					trouves++
				}
			}
		}
	}
	fmt.Printf("brute-force : %d noms testes, %d StringId resolus sur %d\n", n, trouves, len(hashes))
}
