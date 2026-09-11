package replay

// projectiles_pas_test.go — UN VOL S'ARRETE LA OU LE FILM CESSE D'ETRE LISIBLE.
//
// LE DEFAUT MESURE (parc du 2026-09-11, 76 artefacts de schema 51) : 947 trajectoires sur
// 15 735 (6,0 %) portent au moins un pas de 100 ms superieur a 10 m, soit 4 901 pas. La
// signature est nette, et elle n'est pas celle que le fork decrivait : le saut vaut l'etendue
// de la carte sur un axe DIVISEE PAR UNE PUISSANCE DE DEUX — c'est le poids d'UN bit du champ
// quantifie, pas un repli de toute la plage. Sur les quatre films Live Fire du parc
// (`sgh_interlock`, Y sur 12 bits) le saut vaut EXACTEMENT la moitie de l'etendue Y (31,89 m
// pour 63,775 m), l'autre axe ne bougeant pas d'un centimetre : le bit de poids fort de Y
// bascule. 3 907 pas sur 4 901 sont de cette forme.
//
// CE QUI EST CORRIGE ICI EST LA PUBLICATION, PAS LA CAUSE. La cause est en amont, dans la
// dequantification (`filmdec`) — elle est caracterisee dans le rapport du lot, pas traitee. Ce
// qui est traite : le client tracait une droite en travers de la carte, a 300 m/s et plus.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// projPas fabrique une piste de projectile a partir de positions X (Y et Z fixes), un point
// toutes les 100 ms — un point par frame de la grille du rejeu.
func projPas(origine uint64, repos bool, xs ...float32) filmdec.ProjectileTrack {
	pts := make([]filmdec.ProjectileSample, len(xs))
	for i, x := range xs {
		pts[i] = filmdec.ProjectileSample{
			TimestampUS: origine + uint64(i)*100_000, X: x, Y: 0, Z: 0,
		}
	}
	pts[len(pts)-1].AtRest = repos
	return filmdec.ProjectileTrack{Slot: 4096, Gen: 1, Pts: pts}
}

func TestVolCoupeAuPremierPasImpossible(t *testing.T) {
	// Trois pas lisibles, puis un saut de 40 m en 100 ms (400 m/s). Le vol s'arrete au dernier
	// point LISIBLE : rien ne dit ou le projectile est reellement passe apres, et le recoudre
	// reviendrait a inventer la suite.
	tracks := []filmdec.ProjectileTrack{projPas(1_000_000, true, 0, 1, 2, 42, 43, 44)}
	out, _, tronquees := buildProjectiles(tracks, 1_000_000, 100_000)
	if len(out) != 1 {
		t.Fatalf("une trajectoire attendue, obtenu %d", len(out))
	}
	if n := len(out[0].P); n != 3 {
		t.Fatalf("le vol doit s'arreter au 3e point, obtenu %d points : %+v", n, out[0].P)
	}
	if out[0].Rest {
		// `Rest` CERTIFIE une fin de vol. Un vol coupe n'a pas la sienne : le dire serait
		// affirmer qu'on a vu le projectile s'immobiliser la.
		t.Error("un vol coupe ne CERTIFIE aucune fin : Rest doit valoir false")
	}
	if tronquees != 1 {
		t.Errorf("la coupure doit etre COMPTEE : 1 attendue, obtenu %d", tronquees)
	}
}

func TestVolLisibleResteIntactEtGardeSonRepos(t *testing.T) {
	// LE TEMOIN : sans pas impossible, rien ne change — ni les points, ni `Rest`, ni le compte.
	tracks := []filmdec.ProjectileTrack{projPas(1_000_000, true, 0, 2, 4, 6, 8)}
	out, _, tronquees := buildProjectiles(tracks, 1_000_000, 100_000)
	if len(out) != 1 || len(out[0].P) != 5 {
		t.Fatalf("la trajectoire doit sortir entiere (5 points) : %+v", out)
	}
	if !out[0].Rest {
		t.Error("le dernier point porte at-rest : Rest doit valoir true")
	}
	if tronquees != 0 {
		t.Errorf("aucune coupure attendue, obtenu %d", tronquees)
	}
}

func TestVolCoupeTropTotNEstPasPublie(t *testing.T) {
	// Une trajectoire coupee des son deuxieme point n'a qu'UN point de grille : elle ne se
	// dessine pas. Elle n'est pas publiee — mais la coupure est comptee quand meme, sans quoi
	// le compteur mentirait par omission.
	tracks := []filmdec.ProjectileTrack{projPas(1_000_000, true, 0, 60, 61, 62)}
	out, _, tronquees := buildProjectiles(tracks, 1_000_000, 100_000)
	if len(out) != 0 {
		t.Fatalf("une trajectoire d'un seul point de grille ne se publie pas : %+v", out)
	}
	if tronquees != 1 {
		t.Errorf("la coupure doit etre comptee meme sans publication : 1 attendue, obtenu %d", tronquees)
	}
}

func TestCouvertureDesProjectilesRemonteLesTronquees(t *testing.T) {
	// LE COMPTE VOYAGE JUSQU'AU DOCUMENT. Un decodeur qui coupe sans le dire est un rejet
	// avale — l'anti-patron que `coverage.go` existe pour interdire.
	opt := Options{Projectiles: []filmdec.ProjectileTrack{
		projPas(2_000_000, true, 0, 1, 2, 42, 43, 44),
		projPas(2_000_000, true, 0, 2, 4, 6, 8),
	}}
	pos := []filmdec.BipedPosition{
		{Slot: 1024, TimestampUS: 2_000_000, X: 0, Y: 0, HasWorld: true},
		{Slot: 1024, TimestampUS: 2_100_000, X: 1, Y: 0, HasWorld: true},
		{Slot: 1024, TimestampUS: 2_200_000, X: 2, Y: 0, HasWorld: true},
	}
	doc := BuildFromPositions("m", "halo_infinite", pos, nil, opt)
	if doc.Coverage.Projectiles == nil {
		t.Fatal("la couverture des projectiles doit etre publiee des qu'une piste est fournie")
	}
	c := doc.Coverage.Projectiles
	if c.Tracks != 2 || c.Published != 2 || c.Truncated != 1 {
		t.Errorf("couverture attendue tracks=2 published=2 truncated=1, obtenu %+v", *c)
	}
}
