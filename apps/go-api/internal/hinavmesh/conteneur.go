// Package hinavmesh lit le MAILLAGE DE NAVIGATION publie a cote du `.mvar` de chaque
// carte Forge Halo Infinite (`navmesh.blob` de l'asset UGC) : la surface sur laquelle un
// agent MARCHE. C'est ce qu'un fond de rejeu vu de dessus doit montrer, et cela ne demande
// de retirer aucune coque. Offline-pur, sans CGO, sans dependance externe.
//
// EMBALLAGE (mesure sur Isolation 01af558d et Kiken'na df7dbf08) :
//
//	[0..11]  en-tete GROS-BOUTISTE : u32 version (=2) | u32 taille_fichier-12 | u32 0x001FFFFF
//	[12..]   conteneur Bond CompactBinary v2 — LE MEME que les `.mvar`, lu tel quel par
//	         mapvar.DecodeRoot, sans la moindre retouche. Racine = struct a 3 champs :
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

	"levelup/go-api/internal/analysis/replay/mapvar"
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
	// tailleComprimeeMax borne la charge COMPRIMEE, et ce plafond-la est une contrainte de
	// MEMOIRE, pas de securite. On lit le conteneur avec le decodeur Bond generique de
	// mapvar (c'est la bonne dependance : le meme conteneur, un seul lecteur), mais il
	// materialise un mapvar.Value par element de liste, soit ~104 octets par octet de
	// charge. Mesure : 39,6 Mo alloues pour les 363 854 octets d'Isolation
	// (BenchmarkDecodeIsolation). A 8 Mo de charge on culmine donc vers 850 Mo — au-dela,
	// mieux vaut echouer avec ce message qu'epuiser la machine. Les plus gros navmesh
	// observes font 364 Ko (Isolation) et 99 Ko (Kiken'na) : la marge est de 20x. Si un
	// jour une carte depasse ce plafond, la correction n'est PAS de le relever mais
	// d'apprendre a mapvar a exposer une suite d'int8 comme une tranche d'octets.
	tailleComprimeeMax = 8 << 20
)

// Identifiants des champs de la racine Bond. Le conteneur n'est pas auto-descriptif :
// ils sont MESURES, pas devines (champ 0 = un i32 valant 1, non utilise ici).
const (
	champCharge         = 1 // liste d'int8 : le flux zlib
	champTailleInflatee = 2 // u32 : taille attendue apres inflate
)

// decompresse deroule l'emballage d'un `navmesh.blob` et rend la charge utile inflatee.
//
// La taille inflatee declaree par le conteneur est VERIFIEE contre la taille obtenue :
// un ecart signifie que le flux n'est pas celui que le conteneur annonce, et il vaut
// mieux echouer que rendre une charge tronquee dont le parcours produirait des
// coordonnees plausibles et fausses.
func decompresse(blob []byte) ([]byte, error) {
	if len(blob) < tailleEntete {
		return nil, fmt.Errorf("hinavmesh: blob de %d octets, en-tete de %d attendue", len(blob), tailleEntete)
	}
	version := binary.BigEndian.Uint32(blob[0:])
	if version != versionAttendue {
		return nil, fmt.Errorf("hinavmesh: version de blob %d inconnue (attendu %d)", version, versionAttendue)
	}
	if declaree := int(binary.BigEndian.Uint32(blob[4:])); declaree != len(blob)-tailleEntete {
		return nil, fmt.Errorf("hinavmesh: l'en-tete annonce %d octets de charge, le fichier en porte %d",
			declaree, len(blob)-tailleEntete)
	}
	if len(blob) > tailleComprimeeMax {
		return nil, fmt.Errorf("hinavmesh: blob de %d octets au-dela du plafond de %d "+
			"(cout memoire du decodage Bond, voir tailleComprimeeMax)", len(blob), tailleComprimeeMax)
	}
	// Le troisieme champ (0x001FFFFF) est CONSTANT sur les deux cartes mesurees, et sa
	// signification est inconnue : on ne s'en sert pas, et on ne le suppose pas non plus
	// invariant en echouant dessus.

	racine, err := mapvar.DecodeRoot(blob[tailleEntete:])
	if err != nil {
		return nil, fmt.Errorf("hinavmesh: conteneur Bond illisible: %w", err)
	}
	comprime, err := chargeComprimee(racine)
	if err != nil {
		return nil, err
	}
	tailleDeclaree, ok := racine.Field(champTailleInflatee)
	if !ok {
		return nil, fmt.Errorf("hinavmesh: le conteneur Bond ne porte pas le champ %d (taille inflatee)", champTailleInflatee)
	}
	if tailleDeclaree.Uint > tailleInflateeMax {
		return nil, fmt.Errorf("hinavmesh: taille inflatee declaree %d au-dela du plafond %d",
			tailleDeclaree.Uint, tailleInflateeMax)
	}
	charge, err := inflate(comprime, int(tailleDeclaree.Uint))
	if err != nil {
		return nil, err
	}
	if len(charge) != int(tailleDeclaree.Uint) {
		return nil, fmt.Errorf("hinavmesh: inflate rend %d octets, le conteneur en declare %d",
			len(charge), tailleDeclaree.Uint)
	}
	return charge, nil
}

// chargeComprimee extrait le flux zlib du champ 1 de la racine Bond.
func chargeComprimee(racine mapvar.Value) ([]byte, error) {
	liste, ok := racine.Field(champCharge)
	if !ok {
		return nil, fmt.Errorf("hinavmesh: le conteneur Bond ne porte pas le champ %d (charge)", champCharge)
	}
	if len(liste.Items) == 0 {
		return nil, fmt.Errorf("hinavmesh: charge vide")
	}
	comprime := make([]byte, len(liste.Items))
	for i, it := range liste.Items {
		comprime[i] = byte(int8(it.Int))
	}
	return comprime, nil
}

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
