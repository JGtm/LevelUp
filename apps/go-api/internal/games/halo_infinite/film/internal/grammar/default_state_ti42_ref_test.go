package grammar

// default_state_ti42_ref_test.go — LE POINT 6 DE `ti=42` SORT, ET IL SORT SUR LA BONNE PORTE.
//
// Ce que ce fichier verrouille, sur les OCTETS de la mini-bobine versionnee (jamais sur un
// gabarit synthetique, qui ne prouverait que ce qu'on y aurait ecrit) :
//
//  1. la reference d'entite du point 6 est PUBLIEE (elle etait jetee avant le lot 6.6) ;
//  2. sa porte est INVERSEE (`consumeGate0R`) : la valeur voyage quand le bit est a ZERO. Une
//     polarite retournee change le compte de references transmises — c'est la mutation que ce
//     test attrape, et elle a ete jouee ;
//  3. la valeur tient sur 5 bits. Une largeur elargie decalerait tout ce qui suit et le
//     balayage n'accepterait plus 27 records.
//
// LE COMPTE EST FIGE PARCE QU'IL EST DISCRIMINANT : 21 references transmises sur 27 creations.
// MUTATION JOUEE LE 2026-09-10, REJOUEE LE 2026-10-08 : le `!` retire de la porte, le balayage
// n'accepte plus qu'un record — la porte inversee consomme 5 bits de trop et tout ce qui suit se
// decale.
// Le test rougit donc sur le COMPTE avant meme d'arriver aux references.
//
// Les creations sont celles que la marche des trames lit, puis celles que la passe rend derriere
// elle : la 28e creation de la mesure (slot 1529, generation 2, chunk 3 paquet 378, avec sa
// reference), un en-tete trouve dans une trame dont la fermeture prouve le paquet, ne se rend plus
// (plan de l etape 2 de la representation intermediaire, 2.7.d3).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// bobineTI42RefsTransmises / bobineTI42Creations : la mesure de la mini-bobine.
const (
	bobineTI42Creations      = 27
	bobineTI42RefsTransmises = 21
)

// entityRefIndex5Max borne la valeur du point 6 : `ECS_ReadEntityRefIndex5` lit R(5).
const entityRefIndex5Max = 31

func TestTI42_PointSixPublieSaReference(t *testing.T) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("mini-bobine illisible : %v", err)
	}
	wr := &profile.Vec3Range{{Min: -100, Max: 100}, {Min: -100, Max: 100}, {Min: -100, Max: 100}}
	cre, st, err := ScanGroundWeaponCreations(NewFilmContext(film), wr)
	if err != nil {
		t.Fatalf("balayage ti=42 : %v", err)
	}
	if len(cre) != bobineTI42Creations {
		t.Fatalf("%d creations acceptees, attendu %d — la grammaire a bouge",
			len(cre), bobineTI42Creations)
	}
	avecRef := 0
	for _, c := range cre {
		if !c.HasRef {
			continue
		}
		avecRef++
		if c.Ref > entityRefIndex5Max {
			t.Fatalf("reference %d hors de R(5) sur le slot %d : la largeur a bouge",
				c.Ref, c.Slot)
		}
	}
	if avecRef != bobineTI42RefsTransmises {
		t.Fatalf("%d references transmises, attendu %d — porte du point 6 changee ?",
			avecRef, bobineTI42RefsTransmises)
	}
	if st.WithRef != avecRef {
		t.Fatalf("le compteur de couverture dit %d references, la tranche en porte %d",
			st.WithRef, avecRef)
	}
}
