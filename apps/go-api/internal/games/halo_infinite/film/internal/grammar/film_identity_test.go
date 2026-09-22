package grammar

// film_identity_test.go — LA SECTION 2 SE LIT SUR LES SEPT BUILDS (lot 1.5.4).
//
// CE QUE CES TESTS GARDENT, ET POURQUOI CHACUN MORD :
//
//	I-BUILD   Les sept bobines rendent leur build EN CLAIR, et c'est celui que la provenance
//	          de la bobine annonce. Un offset faux ne rendrait pas sept chaines exactes.
//	I-FERME   Le cardinal de la table par type se DERIVE : `(versionOff - finRegistre) / 4`.
//	          Il vaut 123 sur le build de reference — la valeur que l'ecrivain ecrit
//	          (`MOV R9D,0xf60` = 492 octets) — et decroit avec le build. La fermeture est
//	          exacte parce que le registre s'arrete a sa fin structurelle (lot 1.2).
//	I-TEMPS   L'horodatage tombe dans une fenetre plausible ET les sept films sont ORDONNES
//	          par build. C'est le controle interne gratuit de la note : une lecture fausse ne
//	          produirait pas sept dates plausibles NI leur ordre correct par rapport a un champ
//	          independant (la chaine de build).
//	I-COUPE   Un `chunk_00` tronque — dans le registre, dans la section 2, sur une frontiere de
//	          bloc — rend une ERREUR TYPEE et ne panique jamais. C'est la lecon du lot 1.2 :
//	          `zeroTail` paniquait sur un bloc incomplet, sans aucun `recover` chez les
//	          appelants de production.

import (
	"errors"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"path/filepath"
	"testing"
	"time"
)

// bobineIdentite : ce que la PROVENANCE de chaque bobine annonce. La table est ecrite a la
// main depuis les fichiers `PROVENANCE.txt`, pas depuis une mesure : c'est l'oracle externe.
type bobineIdentite struct {
	film, build string
	blocs       int // blocs de registre attendus
	types       int // cardinal de la table par type attendu
}

// bobinesIdentite rend les sept bobines par build et leur identite attendue.
//
// Les cardinaux de table par type sont ceux MESURES le 2026-09-14 sur les 1 351 chunk_00 du
// cache, ou ils sont UNIQUES par build : 123 pour les 1 269 films de HI_1_12_0/HI_1_13_0,
// 122 pour les 39 de HI_1_11_0, 121 pour les 37 de HI_1_10_0/HI_1_9_0/HI_1_8_0, 116 pour
// HI_1_4_1. Les trois ecarts a 123 ferment avec les decalages d'en-tete de la note
// (`16 640 + 4 x 1`, `+ 4 x 2`, `+ 4 x 7`), sans ajustement.
func bobinesIdentite() []bobineIdentite {
	return []bobineIdentite{
		{"a521164d", "HI_1_4_1", 49, 116},
		{"60ae07c4", "HI_1_8_0", 49, 121},
		{"11de8353", "HI_1_9_0", 49, 121},
		{"111fa685", "HI_1_10_0", 49, 121},
		{"e5adf7b2", "HI_1_11_0", 49, 122},
		{"bcb6d393", "HI_1_12_0", 50, 123},
		{"fb1a1a72", "HI_1_13_0", 50, 123},
	}
}

// bobineChunk00 rend le `chunk_00` DECOMPRESSE d'une bobine par build.
func bobineChunk00(t *testing.T, film string) []byte {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+film)
	d, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("bobine %s : %v — regenerer les bobines du lot 0.A.2", film, err)
	}
	return d
}

// TestReadFilmIdentitySurLesBobines execute I-BUILD, I-FERME et I-TEMPS.
func TestReadFilmIdentitySurLesBobines(t *testing.T) {
	var horodatages []uint32
	for _, b := range bobinesIdentite() {
		id, err := ReadFilmIdentity(bobineChunk00(t, b.film))
		if err != nil {
			t.Fatalf("%s (%s) : %v", b.film, b.build, err)
		}
		verifierIdentite(t, b, id)
		horodatages = append(horodatages, id.MatchStartUnix)
		t.Logf("%-10s build %-10q version %-18q saveur %-8q idBuild 0x%08x changelist 0x%08x "+
			"blocs %d table %d horodatage %s", b.film, id.Build, id.Version, id.Flavor,
			id.BuildID, id.Changelist, id.RegistryBlocks, len(id.TypeVersions),
			time.Unix(int64(id.MatchStartUnix), 0).UTC().Format(time.RFC3339))
	}
	// I-TEMPS, second volet : les bobines sont listees dans l'ordre des builds, donc leurs
	// horodatages doivent croitre. C'est un controle GRATUIT contre un champ independant.
	for i := 1; i < len(horodatages); i++ {
		if horodatages[i] <= horodatages[i-1] {
			t.Errorf("I-TEMPS CASSE : l'horodatage du build %d (%d) n'est pas posterieur "+
				"a celui du build %d (%d)", i, horodatages[i], i-1, horodatages[i-1])
		}
	}
}

// verifierIdentite confronte une identite lue a ce que la provenance annonce.
func verifierIdentite(t *testing.T, b bobineIdentite, id profile.FilmIdentity) {
	t.Helper()
	if id.Build != b.build {
		t.Errorf("%s : build lu %q, attendu %q", b.film, id.Build, b.build)
	}
	if id.RegistryBlocks != b.blocs {
		t.Errorf("%s : %d blocs de registre, attendu %d", b.film, id.RegistryBlocks, b.blocs)
	}
	if len(id.TypeVersions) != b.types {
		t.Errorf("%s : table par type de %d entrees, attendu %d (I-FERME)",
			b.film, len(id.TypeVersions), b.types)
	}
	if id.Flavor != "release" {
		t.Errorf("%s : saveur %q, attendu \"release\"", b.film, id.Flavor)
	}
	if id.Version == "" || id.Version[0] != '6' {
		t.Errorf("%s : version %q, attendue de la forme 6.x.y.z", b.film, id.Version)
	}
	if id.BuildID == 0 || id.Changelist == 0 {
		t.Errorf("%s : identifiant de build %d et changelist %d, aucun ne doit etre nul",
			b.film, id.BuildID, id.Changelist)
	}
	// Fenetre ecrite avant la mesure (2020-09 .. 2030-01), celle de l'instrument d'origine.
	if id.MatchStartUnix <= 1600000000 || id.MatchStartUnix >= 1900000000 {
		t.Errorf("%s : horodatage %d hors de la fenetre plausible", b.film, id.MatchStartUnix)
	}
	if id.BodyBit <= 0 {
		t.Errorf("%s : BodyBit %d, le corps doit commencer dans le tampon", b.film, id.BodyBit)
	}
}

// TestReadFilmIdentiteTronquee execute I-COUPE : aucune troncature ne panique, toutes rendent
// une erreur TYPEE.
func TestReadFilmIdentiteTronquee(t *testing.T) {
	complet := bobineChunk00(t, "fb1a1a72")
	id, err := ReadFilmIdentity(complet)
	if err != nil {
		t.Fatalf("temoin positif : %v", err)
	}
	cas := []struct {
		nom     string
		taille  int
		attendu error
	}{
		{"tampon vide", 0, ErrChunk00Truncated},
		{"en-tete seul", 8, ErrChunk00Truncated},
		{"coupe sur une frontiere de bloc", registryEntryBase + 10*archetypeBlockSize,
			ErrChunk00Truncated},
		{"coupe au dernier bloc entier", registryEntryBase + 49*archetypeBlockSize + 1,
			ErrChunk00Truncated},
		// COUPER DANS LA SECTION 2 NE REND PAS `ErrNoFilmIdentity`, ET C'EST LE BON
		// DIAGNOSTIC : le registre s'arrete a sa fin STRUCTURELLE (lot 1.2), donc un tampon
		// coupe avant cette fin epuise la boucle de blocs et `Registry.Truncated` le dit. La
		// cause premiere est la troncature, pas l'absence de section — confondre les deux
		// ferait accuser le film d'un defaut du tampon.
		{"coupe juste apres le registre", id.BuildOffset - 100, ErrChunk00Truncated},
		{"coupe dans la chaine de build", id.BuildOffset + 2, ErrChunk00Truncated},
		{"coupe apres la chaine de build", id.BuildOffset + identBoolOff + 8,
			ErrChunk00Truncated},
		{"coupe juste avant le corps", id.BuildOffset + identBoolOff + identApresBoolBytes,
			ErrChunk00Truncated},
	}
	for _, c := range cas {
		coupe := append([]byte(nil), complet[:c.taille]...)
		got, err := ReadFilmIdentity(coupe)
		if !errors.Is(err, c.attendu) {
			t.Errorf("%s (%d octets) : erreur %v, attendue %v", c.nom, c.taille, err, c.attendu)
		}
		if got.Build != "" {
			t.Errorf("%s : identite partielle rendue (%q) alors que la lecture a echoue",
				c.nom, got.Build)
		}
	}
}

// TestReadFilmIdentiteSansSection : un `chunk_00` COMPLET mais dont la section 2 ne porte aucune
// chaine de build rend [ErrNoFilmIdentity], pas une identite vide.
//
// LE CAS EST REEL, ET IL A UNE POPULATION NOMMEE : 5 films du cache (`03af54c3`, `13b00e35`,
// `47d20b5d`, `50247b26`, `a349fea8`) sont exactement dans cet etat — c'est le corpus qui les
// compte (player_table_corpus_test.go). Ici la meme situation est fabriquee a partir d'un film
// sain, pour que le cas soit garde SANS garde d'environnement, donc aussi en CI.
func TestReadFilmIdentiteSansSection(t *testing.T) {
	d := append([]byte(nil), bobineChunk00(t, "fb1a1a72")...)
	id, err := ReadFilmIdentity(d)
	if err != nil {
		t.Fatalf("temoin positif : %v", err)
	}
	// On efface les trois champs de chaine : version, build, saveur.
	for i := id.BuildOffset - identFieldBytes; i < id.BuildOffset+2*identFieldBytes; i++ {
		d[i] = 0
	}
	got, err := ReadFilmIdentity(d)
	if !errors.Is(err, ErrNoFilmIdentity) {
		t.Errorf("erreur %v, attendue %v", err, ErrNoFilmIdentity)
	}
	if got.Build != "" || got.BodyBit != 0 {
		t.Errorf("identite partielle rendue : build %q, BodyBit %d", got.Build, got.BodyBit)
	}
}

// TestReadFilmIdentiteEncoreCompressee : un tampon qui porte encore son en-tete zlib est REFUSE,
// pas lu en silence. Meme refus que `ParseRegistryChunk` (lot 1 de PLAN_CUISSON_PERF).
func TestReadFilmIdentiteEncoreCompressee(t *testing.T) {
	// Un en-tete zlib valide (CM=8, pas de dictionnaire, somme de controle % 31 == 0).
	brut := append([]byte{0x78, 0x9c}, make([]byte, 4096)...)
	if _, err := ReadFilmIdentity(brut); !errors.Is(err, ErrRegistryStillCompressed) {
		t.Errorf("erreur %v, attendue %v", err, ErrRegistryStillCompressed)
	}
}

// TestControleDeCorruptionEstLeBitDe0xCB45C : le drapeau rendu est EXACTEMENT le bit de poids
// fort de l octet `buildOff + identBoolOff`, et rien d autre du tampon ne le decide (lot 5.18.1).
//
// C EST UN TEST BIT-EXACT CONTRE L ECRIVAIN, et il fige les deux moities du maillon :
//
//	la POSITION   `FUN_14299b198` @14299b25b ecrit ce bit par `FUN_1406d49c4` juste apres la
//	              changelist (`film+0xCB458`), et `FUN_14299ab50` @14299ac28 le relit par
//	              `FUN_1406cf008`. Le bit bascule, le drapeau bascule — et AUCUN autre bit de
//	              l octet ne le fait (les sept suivants appartiennent au premier champ de nom).
//	le VOISINAGE  l horodatage, qui vit `identDecalageBit` plus loin, ne bouge pas quand le
//	              drapeau bascule : le decalage d un bit est INDEPENDANT de la valeur du bit.
func TestControleDeCorruptionEstLeBitDe0xCB45C(t *testing.T) {
	base := bobineChunk00(t, "fb1a1a72")
	temoin, err := ReadFilmIdentity(base)
	if err != nil {
		t.Fatalf("temoin positif : %v", err)
	}
	if temoin.ControleDeCorruption {
		t.Fatalf("la bobine de reference porte le drapeau LEVE — le cas de base du test tombe")
	}
	off := temoin.BuildOffset + identBoolOff
	for bit := 0; bit < 8; bit++ {
		d := append([]byte(nil), base...)
		d[off] |= byte(1) << (7 - uint(bit))
		got, errBit := ReadFilmIdentity(d)
		if errBit != nil {
			t.Fatalf("bit %d : %v", bit, errBit)
		}
		if attendu := bit == 0; got.ControleDeCorruption != attendu {
			t.Errorf("bit %d leve : ControleDeCorruption %v, attendu %v",
				bit, got.ControleDeCorruption, attendu)
		}
		if bit == 0 && got.MatchStartUnix != temoin.MatchStartUnix {
			t.Errorf("le drapeau leve a deplace l horodatage : %d au lieu de %d",
				got.MatchStartUnix, temoin.MatchStartUnix)
		}
	}
}
