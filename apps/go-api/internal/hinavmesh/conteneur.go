// Package hinavmesh lit le MAILLAGE DE NAVIGATION publie a cote du `.mvar` de chaque
// carte Forge Halo Infinite (`navmesh.blob` de l'asset UGC) : la surface sur laquelle un
// agent MARCHE. C'est ce qu'un fond de rejeu vu de dessus doit montrer, et cela ne demande
// de retirer aucune coque. Offline-pur, sans CGO, sans dependance externe.
//
// EMBALLAGE (mesure sur Isolation 01af558d et Kiken'na df7dbf08) :
//
//	[0..11]  en-tete GROS-BOUTISTE : u32 version (=2) | u32 taille_fichier-12 | u32 0x001FFFFF
//	[12..]   conteneur Bond CompactBinary v2 — LE MEME que les `.mvar`, lu tel quel par
//	         le lecteur dedie de enveloppe.go (meme conteneur, meme emballage que les
//	         autres blobs de l asset). Racine = struct a 3 champs :
//	         champ0 i32 (=1) | champ1 liste d'int8 (la charge compressee) |
//	         champ2 u32 (taille inflatee, verifiee apres inflate).
//	champ1   flux ZLIB d'en-tete `58 09` : CM=8 (deflate), CINFO=5 donc fenetre de 8 Ko.
//	         Le premier octet n'est PAS 0x78 — un balayage des signatures usuelles
//	         78 9c / 78 da / 78 01 manque ce flux.
//
// CHARGE INFLATEE : preambule de 28 octets gros-boutistes [u32 2][u32 1][5 x u32 taille
// de region], puis les 5 regions bout a bout. Les quatre premieres sont des fichiers-tag
// Havok 2022.1.0 (`TAG0`, `SDKV` = "20220100") de classe racine respective hkaiNavMesh,
// hkaiClusterGraph, hkaiTraversalAnnotationLibrary et hkcdStaticAabbTree. La cinquieme
// n'est pas un fichier-tag et n'est pas necessaire au fond de carte.
//
// COUVERTURE : `navmesh.blob` n'existe QUE pour les cartes Forge (canevas). Les cartes
// natives du studio repondent 404 sur ce chemin — leur maillage est cuit dans les
// `.module`, pas publie avec l'asset UGC. Le parc se partage donc en deux chaines, et
// c'est la bonne complementarite : les cartes ou le rendu de geometrie donne une bouillie
// illisible sont exactement celles qui portent un navmesh.
package hinavmesh

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// tailleEntete est la longueur de l'en-tete proprietaire qui precede le conteneur Bond.
	tailleEntete = 12
	// versionAttendue est la seule valeur observee du champ de version (2 cartes mesurees).
	versionAttendue = 2
	// taillePreambule est la longueur du preambule de la charge inflatee.
	taillePreambule = 28
	// nbRegions est le nombre de regions du preambule. MESURE sur deux cartes : le
	// preambule ne porte pas de compte, voir le controle de queue dans regions().
	nbRegions = 5
	// tailleQueue est le reste nul apres les 5 regions : 4 octets sur les deux cartes.
	tailleQueue = 4
	// tailleInflateeMax borne l'allocation d'inflate : un blob corrompu ne doit pas
	// pouvoir demander une allocation arbitraire. Reference : Isolation inflate a 1,1 Mo.
	tailleInflateeMax = 128 << 20
)

// Identifiants des champs de la racine Bond. Le conteneur n'est pas auto-descriptif :
// ils sont MESURES, pas devines (champ 0 = un i32 valant 1, non utilise ici).
const (
	champCharge         = 1 // liste d'int8 : le flux zlib
	champTailleInflatee = 2 // u32 : taille attendue apres inflate
)

// inflate deroule le flux zlib. `attendu` ne sert qu'a pre-dimensionner le tampon.
func inflate(comprime []byte, attendu int) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(comprime))
	if err != nil {
		return nil, fmt.Errorf("hinavmesh: en-tete zlib illisible (%02x %02x): %w", comprime[0], comprime[1], err)
	}
	defer func() { _ = zr.Close() }()
	tampon := bytes.NewBuffer(make([]byte, 0, attendu))
	// io.CopyN plafonne l'ecriture : un flux zlib forge ne peut pas faire exploser la RAM.
	if _, err := io.CopyN(tampon, zr, int64(attendu)+1); err != nil && err != io.EOF {
		return nil, fmt.Errorf("hinavmesh: inflate a echoue: %w", err)
	}
	return tampon.Bytes(), nil
}

// regions decoupe la charge inflatee selon le preambule de 28 octets.
func regions(charge []byte) ([][]byte, error) {
	if len(charge) < taillePreambule {
		return nil, fmt.Errorf("hinavmesh: charge de %d octets, preambule de %d attendu", len(charge), taillePreambule)
	}
	decoupe := make([][]byte, 0, nbRegions)
	debut := taillePreambule
	for i := 0; i < nbRegions; i++ {
		taille := int(binary.BigEndian.Uint32(charge[8+4*i:]))
		fin := debut + taille
		if taille < 0 || fin > len(charge) {
			return nil, fmt.Errorf("hinavmesh: region %d de %d octets deborde la charge (%d..%d sur %d)",
				i+1, taille, debut, fin, len(charge))
		}
		decoupe = append(decoupe, charge[debut:fin])
		debut = fin
	}
	// Le nombre de regions (5) est MESURE sur deux cartes, pas declare par le fichier : le
	// preambule ne porte pas de compte. La queue le controle — les 5 tailles plus le
	// preambule laissent exactement 4 octets nuls sur les deux cartes. Si une carte
	// declarait un autre nombre de regions, ce reste ne tomberait pas juste, et il vaut
	// mieux le savoir ici que decouper de travers en silence.
	if reste := len(charge) - debut; reste != tailleQueue {
		return nil, fmt.Errorf("hinavmesh: %d octets apres les %d regions, %d attendus "+
			"(le preambule ne declare peut-etre pas %d regions sur cette carte)",
			reste, nbRegions, tailleQueue, nbRegions)
	}
	return decoupe, nil
}
