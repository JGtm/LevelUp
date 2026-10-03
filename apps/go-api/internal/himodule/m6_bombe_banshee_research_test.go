//go:build research

// m6_bombe_banshee_research_test.go — SONDE M6.2 (campagne « retours rejeu », 2026-09-24) :
// l'identité du tag `850902EF`, arme de véhicule OBSERVÉE dans un film (25 tirs sur un seul
// document, portés par une Banshee 11 fois et un Warthog 4 fois — mesure du lot M4a) et qu'aucune
// pièce ne nommait. Même méthode que la sonde CA9 (`ca9_unarmed_research_test.go`, dont ce fichier
// réutilise les lecteurs) : lecture SEULE des modules installés, aucun film ouvert, rien d'écrit.
//
// CE QUE L'INSTRUMENT MESURE :
//
//  1. `850902EF` est un tag `weap` du module `common`, et il partage avec `0000AA69` — la bombe
//     de la Banshee du lot V3F (`V3F_TIRS_COVENANT_2026-09-02.md` : « M2 tir lourd unique »,
//     `snd! dc3d707b`) — son SON DE TIR et la majorité de ses dépendances ; seuls le projectile
//     et deux accessoires diffèrent. Témoin négatif : les canons de la Banshee `0000AA68`.
//  2. Qui le porte : la chaîne des dépendances DÉCLARÉES remonte `weap <- uwfa <- sofa <- sofd <-
//     vcdd`, et le `vcdd` déclare aussi le `vehi` de la Banshee (`000026ED`). Le script Lua global
//     (`hsc*` `A35C6CE9`) référence ce `vcdd` : c est une valeur de sa table `MPVehicleConfigs`
//     (même encodage du pool que `WeaponTags`, mesuré par CA9 ; la clé ne se lit pas par adjacence).
//
// Commande (depuis `apps/go-api`, CGO actif ; racine `deploy` passée par l'environnement, jamais
// écrite dans le dépôt) :
//
//	M6_DEPLOY=<bibliotheque>/<jeux>/Halo Infinite/deploy go test -tags=research -count=1 -v \
//	  -run '^TestM6BombeBanshee$' ./internal/himodule/
package himodule_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/himodule"
)

const (
	m6Observe       uint32 = 0x850902EF // tag de `Shot.w` observé au parc
	m6BombeV3F      uint32 = 0x0000AA69 // bombe de la Banshee (V3F)
	m6CanonsV3F     uint32 = 0x0000AA68 // canons de la Banshee (V3F), témoin négatif
	m6SonBombe      uint32 = 0xDC3D707B // snd! de tir de la bombe (V3F)
	m6VehiBanshee   uint32 = 0x000026ED
	m6ScriptGlobal  uint32 = 0xA35C6CE9
	m6ProfondeurMax        = 6
	m6FenetreCles          = 400
)

// m6DepsParGid : les dépendances déclarées d'un tag, indexées par GlobalID.
func m6DepsParGid(tag []byte) map[uint32]string {
	out := map[uint32]string{}
	for _, d := range ca9Deps(tag) {
		out[d.gid] = d.groupe
	}
	return out
}

// m6TableAvant rend le nom de la DERNIÈRE table du pool du script Lua ouverte avant l'entier
// TAG(gid) — la dernière constante chaîne de la fenêtre dont le nom commence par une majuscule
// (les tables globales du script : `WeaponTags`, `MPVehicleConfigs`...). Mesuré le 2026-09-24 :
// la CLÉ d'une valeur ne se lit PAS par adjacence dans ce pool — une clé déjà internée plus haut
// (`banshee`, `ghost`...) n'est pas réécrite, sa valeur suit seule. Vide : aucune occurrence.
func m6TableAvant(lua []byte, gid uint32) string {
	motif := make([]byte, 9)
	motif[0] = 0x02
	binary.BigEndian.PutUint64(motif[1:], uint64(gid))
	idx := bytes.Index(lua, motif)
	if idx < 0 {
		return ""
	}
	for off := idx - 11; off >= 0 && off >= idx-m6FenetreCles; off-- {
		if s, _, ok := ca9Chaine(lua, off); ok && s[0] >= 'A' && s[0] <= 'Z' {
			return s
		}
	}
	return ""
}

// m6Index : en UNE passe sur le module, les dépendances déclarées de chaque tag et l'index
// inverse (qui déclare quoi). Seules les tables de dépendances sont gardées, jamais les corps.
type m6Index struct {
	deps      map[uint32]map[uint32]string
	referents map[uint32][]uint32
}

func m6Indexer(t *testing.T, fichiers []himodule.File, extraire func(himodule.File) ([]byte, error)) m6Index {
	t.Helper()
	ix := m6Index{deps: map[uint32]map[uint32]string{}, referents: map[uint32][]uint32{}}
	for _, f := range fichiers {
		data, err := extraire(f)
		if err != nil {
			continue
		}
		d := m6DepsParGid(data)
		ix.deps[f.GlobalID] = d
		for gid := range d {
			ix.referents[gid] = append(ix.referents[gid], f.GlobalID)
		}
	}
	return ix
}

func TestM6BombeBanshee(t *testing.T) {
	racine := strings.TrimSpace(os.Getenv("M6_DEPLOY"))
	if racine == "" {
		t.Skip("M6_DEPLOY non posé (racine deploy de l'installation)")
	}
	mc, ic := ca9Index(t, filepath.Join(racine, "any", "globals", "common-rtx-new.module"))

	// 1. Identité et parenté avec la bombe V3F.
	fo, obs := ca9Extraire(t, mc, ic, m6Observe)
	_, bombe := ca9Extraire(t, mc, ic, m6BombeV3F)
	_, canons := ca9Extraire(t, mc, ic, m6CanonsV3F)
	depsObs, depsBombe, depsCanons := m6DepsParGid(obs), m6DepsParGid(bombe), m6DepsParGid(canons)
	communsBombe, communsCanons := 0, 0
	for gid, g := range depsObs {
		if depsBombe[gid] == g {
			communsBombe++
		}
		if depsCanons[gid] == g {
			communsCanons++
		}
		t.Logf("  %#08x dépend de %s %#08x (bombe V3F : %v, canons : %v)", m6Observe, g, gid,
			depsBombe[gid] == g, depsCanons[gid] == g)
	}
	t.Logf("tag %#08x : groupe=%s, %d dépendances ; communes avec la bombe %#08x : %d/%d ; avec "+
		"les canons %#08x : %d/%d", m6Observe, fo.Group, len(depsObs), m6BombeV3F, communsBombe,
		len(depsBombe), m6CanonsV3F, communsCanons, len(depsCanons))
	if fo.Group != "weap" || depsObs[m6SonBombe] != "snd!" || depsBombe[m6SonBombe] != "snd!" {
		t.Errorf("%#08x : groupe %q, son de tir de la bombe partagé : %v", m6Observe, fo.Group,
			depsObs[m6SonBombe] == "snd!")
	}
	if communsBombe*2 <= len(depsObs) || communsCanons >= communsBombe {
		t.Errorf("parenté non établie : %d communes avec la bombe, %d avec les canons",
			communsBombe, communsCanons)
	}

	// 2. Le porteur, par la chaîne des dépendances déclarées (remontée bornée).
	ix := m6Indexer(t, mc.Files(""), mc.Extract)
	vehiTrouve, vcdd := false, uint32(0)
	front := []uint32{m6Observe}
	for prof := 0; prof < m6ProfondeurMax && len(front) > 0 && !vehiTrouve; prof++ {
		var suivant []uint32
		for _, id := range front {
			for _, r := range ix.referents[id] {
				g := ic[r].Group
				t.Logf("  remontée %d : %s %#08x <- %s %#08x", prof, ic[id].Group, id, g, r)
				if g == "hsc*" {
					continue
				}
				if _, ok := ix.deps[r][m6VehiBanshee]; ok {
					vehiTrouve, vcdd = true, r
				}
				suivant = append(suivant, r)
			}
		}
		front = suivant
	}
	if !vehiTrouve {
		t.Fatalf("aucun tag de la remontée ne déclare le vehi de la Banshee %#08x", m6VehiBanshee)
	}
	_, lua := ca9Extraire(t, mc, ic, m6ScriptGlobal)
	table := m6TableAvant(lua, vcdd)
	t.Logf("VERDICT : %#08x est l'arme de la configuration MULTIJOUEUR de la Banshee (%s %#08x "+
		"déclare vehi %#08x ; valeur de la table Lua %q) et partage le son de tir de la bombe %#08x",
		m6Observe, ic[vcdd].Group, vcdd, m6VehiBanshee, table, m6BombeV3F)
	if table != "MPVehicleConfigs" {
		t.Errorf("le vcdd %#08x n'est pas une valeur de MPVehicleConfigs (table lue : %q)", vcdd, table)
	}
}
