//go:build research && campagne_overlay

package grammar

// r_veh_imagecle_research_test.go — CAMPAGNE DE GRAMMAIRE, RECHERCHE R-L4 (2026-10-02),
// ELARGISSEMENT : la lecture des images-cles etablie sur `ti=40` (portee DAT_144e61ea0 sur tout
// le record d etat complet, chemin absolu d i0 de l ecrivain, decoupage MPP 8/3 des formats
// <= 25) mesuree sur TOUS les archetypes. Mesure seulement ; exige la surcouche `r_veh_overlay/`
// (crochet des composants `ti=40` non portes, porte `+0x818` posee par chassis au moment ou le
// bloc MPP publie MPPWord32).
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 120m \
//	  -run '^TestRVehImagesClesTousArchetypes$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rviVariante : une lecture des images-cles. La portee `DAT_144e61ea0` sur le record d etat
// complet et le chemin absolu d i0 de l ecrivain sont ceux de la production (plan LK, 2026-10-08) :
// les variantes qui les posaient se confondent avec celles qui restent.
type rviVariante struct {
	nom          string
	crochet, m83 bool
}

// rviCompte : par archetype.
type rviCompte struct{ n, fermes, voisins, voisinsFermes int }

// TestRVehImagesClesTousArchetypes : fermeture des records d image-cle bornes, par archetype.
func TestRVehImagesClesTousArchetypes(t *testing.T) {
	racine, sortie, films, physique, _ := rvEnv(t)
	defer func() { bis2Intercepteur, b2vPorte = nil, false }()
	variantes := []rviVariante{
		{nom: "production"},
		{nom: "ti40-composants", crochet: true},
		{nom: "mpp8/3+ti40-composants", crochet: true, m83: true},
		{nom: "mpp8/3+production", m83: true},
	}
	lignes := []string{"film\tbuild\tvariante\tti\trecords\tfermes\tvoisins\tvoisins_fermes"}
	for _, id := range films {
		f, fin := rvOuvrir(t, racine, id, "campagne/r-veh-imagecle")
		if f == nil {
			continue
		}
		marche := f.fc.MarcheDImageCle()
		for _, v := range variantes {
			prec := f.fc.ProfilDeBalayage().MPP
			if v.m83 {
				f.fc.PoserMPP(profile.MPPWidths{Lead: 8, Index: 3})
			}
			ctx := f.fc.ContexteDeLecture()
			ctx.Obs = &Observation{MppHook: func(fl MPPField, val uint64, present bool) {
				if fl == MPPWord32 && present {
					b2vPorte = physique[uint32(val)] == 6 //nolint:gosec // mot de 32 bits
				}
			}}
			bis2Intercepteur = nil
			if v.crochet {
				bis2Intercepteur = b2vCrochet(true, false)
			}
			parTI := map[int]*rviCompte{}
			for _, num := range f.fc.ChunkNumbers() {
				data, pks, ok := f.fc.ChunkAt(num)
				if !ok {
					continue
				}
				for _, pk := range pks {
					if pk.Type != PacketTypeKeyframe {
						continue
					}
					pay := pk.Payload(data)
					for _, b := range seulesBornees(keyframeBornesDe(marche.Records(pay))) {
						b2vPorte = false
						tr := WalkKeyframeFullState(pay, b.Bit, f.reg, ctx)
						c := parTI[b.TI]
						if c == nil {
							c = &rviCompte{}
							parTI[b.TI] = c
						}
						c.n++
						ferme := tr.DesyncAt < 0 && tr.EndBit == b.Want
						if ferme {
							c.fermes++
						}
						if b.Voisin {
							c.voisins++
							if ferme {
								c.voisinsFermes++
							}
						}
					}
				}
			}
			bis2Intercepteur = nil
			f.fc.PoserMPP(prec)
			var tot rviCompte
			for ti, c := range parTI {
				lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%02d\t%d\t%d\t%d\t%d", f.id, f.build, v.nom, ti, c.n, c.fermes,
					c.voisins, c.voisinsFermes))
				tot.n += c.n
				tot.fermes += c.fermes
				tot.voisins += c.voisins
				tot.voisinsFermes += c.voisinsFermes
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\ttout\t%d\t%d\t%d\t%d", f.id, f.build, v.nom, tot.n, tot.fermes,
				tot.voisins, tot.voisinsFermes))
			t.Logf("%s %-44s fermes %6d / %6d ; voisins %6d / %6d", id, v.nom, tot.fermes, tot.n, tot.voisinsFermes,
				tot.voisins)
		}
		fin()
	}
	b2Ecrire(t, sortie, "r_veh_imagecle_archetypes.tsv", lignes)
}
