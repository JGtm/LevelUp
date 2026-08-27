package hinavmesh

// reflexion_test.go — LE TEMOIN QUI COMPTE : abimer la REFLEXION doit se voir.
//
// Muter le blob comprime ne prouve pas grand-chose : la somme de controle zlib attrape
// tout. Le vrai risque est ailleurs — un decalage d'un octet dans la table des types
// donnerait des offsets de membres decales, donc des indices lus au mauvais endroit, donc
// une geometrie plausible et FAUSSE que rien ne signalerait.
//
// Ce test mute donc la charge INFLATEE, la re-emballe, et exige de chaque mutation des
// tables TNA1 et TBDY l'un des deux verdicts acceptables : une erreur, ou un maillage
// IDENTIQUE au maillage de reference. Un maillage silencieusement different signifierait
// que le decodeur a avale un octet faux.

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"testing"
)

func TestDecodeRefuseUneReflexionAbimee(t *testing.T) {
	// Kiken'na : le plus petit temoin, donc le balayage le plus large a duree egale.
	const asset = "df7dbf08-b8de-4ade-9d7f-1947128c9ae4"
	charge, err := decompresse(chargeBlobTemoin(t, asset))
	if err != nil {
		t.Fatalf("Decompresse: %v", err)
	}
	reference := empreinte(decodeTemoin(t, asset))

	// Le re-emballage doit d'abord etre neutre, sinon le test ne mesure que lui-meme.
	rendu, err := Decode(emballe(t, charge))
	if err != nil {
		t.Fatalf("re-emballage non neutre: %v", err)
	}
	if empreinte(rendu) != reference {
		t.Fatalf("re-emballage non neutre : empreinte %q au lieu de %q", empreinte(rendu), reference)
	}

	cibles := sectionsDeReflexion(t, charge)
	var mutees, silencieuses int
	for _, plage := range cibles {
		// Un octet sur cinq : assez dense pour couvrir chaque champ, assez court pour
		// tenir dans la duree d'un test unitaire.
		for pos := plage.debut; pos < plage.fin; pos += 5 {
			abimee := append([]byte(nil), charge...)
			abimee[pos] ^= 0xFF
			mutees++
			m, err := Decode(emballe(t, abimee))
			if err != nil {
				continue
			}
			if empreinte(m) != reference {
				silencieuses++
				t.Errorf("mutation a l'offset %d (section %s) : maillage DIFFERENT sans erreur (%q au lieu de %q)",
					pos, plage.nom, empreinte(m), reference)
			}
		}
	}
	if mutees == 0 {
		t.Fatal("aucune mutation jouee : les sections de reflexion n'ont pas ete localisees")
	}
	t.Logf("%d mutations de TNA1/TBDY jouees, %d acceptees en silence", mutees, silencieuses)
}

type plageSection struct {
	nom        string
	debut, fin int
}

// sectionsDeReflexion localise TNA1 et TBDY du fichier-tag du navmesh, en offsets absolus
// dans la charge inflatee.
func sectionsDeReflexion(t *testing.T, charge []byte) []plageSection {
	t.Helper()
	decoupe, err := regions(charge)
	if err != nil {
		t.Fatalf("regions: %v", err)
	}
	// Les regions sont des sous-tranches de `charge` : leur decalage absolu se retrouve
	// par la capacite restante, ce qui evite de recalculer le preambule.
	base := 0
	for _, region := range decoupe {
		if len(region) >= 8 && string(region[4:8]) == "TAG0" {
			sections := map[string][2]int{}
			if err := parcoursSections(region, 0, len(region), sections, 0); err == nil {
				if _, ok := sections["TNA1"]; ok {
					var out []plageSection
					for _, nom := range []string{"TNA1", "TBDY"} {
						s := sections[nom]
						out = append(out, plageSection{nom: nom, debut: base + s[0], fin: base + s[0] + s[1]})
					}
					return out
				}
			}
		}
		base += len(region)
	}
	t.Fatal("aucune region ne porte une section TNA1")
	return nil
}

// empreinte resume un maillage : deux maillages de meme empreinte sont interchangeables
// pour un fond de carte.
func empreinte(m *Maillage) string {
	var somme float64
	for _, f := range m.Faces {
		for _, i := range f.Sommets {
			p := m.Sommets[i]
			somme += p.X + p.Y*3 + p.Z*7
		}
	}
	return fmt.Sprintf("s=%d f=%d aire=%.3f somme=%.3f haut=%v", len(m.Sommets), len(m.Faces),
		m.AireAuSol(), somme, m.Haut)
}

// emballe reconstruit un `navmesh.blob` autour d'une charge : en-tete de 12 octets,
// conteneur Bond CompactBinary v2, flux zlib. C'est l'inverse exact de Decompresse.
func emballe(t *testing.T, charge []byte) []byte {
	t.Helper()
	var comprime bytes.Buffer
	zw := zlib.NewWriter(&comprime)
	if _, err := zw.Write(charge); err != nil {
		t.Fatalf("deflate: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("deflate (fermeture): %v", err)
	}

	// Tags mesures sur les blobs reels : type | (identifiant << 5).
	const (
		tagChamp0 = 0x10 // int32, identifiant 0
		tagChamp1 = 0x2b // liste, identifiant 1
		tagChamp2 = 0x45 // uint32, identifiant 2
		elemInt8  = 0x0e // element de type int8, compte en varint
		stop      = 0x00
	)
	corps := []byte{tagChamp0, 0x02} // zigzag(2) = 1
	corps = append(corps, tagChamp1, elemInt8)
	corps = ajouteVarint(corps, uint64(comprime.Len()))
	corps = append(corps, comprime.Bytes()...)
	corps = append(corps, tagChamp2)
	corps = ajouteVarint(corps, uint64(len(charge)))
	corps = append(corps, stop)

	bond := ajouteVarint(nil, uint64(len(corps)))
	bond = append(bond, corps...)

	blob := make([]byte, tailleEntete)
	binary.BigEndian.PutUint32(blob[0:], versionAttendue)
	binary.BigEndian.PutUint32(blob[4:], uint32(len(bond)))
	binary.BigEndian.PutUint32(blob[8:], 0x001FFFFF)
	return append(blob, bond...)
}

func ajouteVarint(dst []byte, v uint64) []byte {
	for v >= 0x80 {
		dst = append(dst, byte(v)|0x80)
		v >>= 7
	}
	return append(dst, byte(v))
}
