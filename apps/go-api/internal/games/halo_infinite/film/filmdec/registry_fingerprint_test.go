package filmdec

// registry_fingerprint_test.go — L'EMPREINTE, SON TEMOIN, ET LE COUT DU RE-PARSE.
//
// Deux tests inconditionnels (grammaire de l'empreinte, deduplication de l'alerte) et un test
// sous garde `DELTA_WITNESS_FILM` qui CHIFFRE l'empreinte d'un film reel et le cout de son
// re-parse — la mesure que l'item 0.3 du plan demande de publier.

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
	"time"
)

// registryFixture fabrique un chunk_00 INFLATE d'un seul bloc portant les entrees nommees,
// au cadrage du jeu : en-tete de `registryEntryBase` octets, puis des entrees de
// `registrySlotSize` octets `[nom @ +0][u32 niveau @ +0x100]`.
func registryFixture(levels []uint32, names []string) []byte {
	data := make([]byte, registryEntryBase+archetypeBlockSize)
	for i, n := range names {
		off := registryEntryBase + i*registrySlotSize
		copy(data[off:], n)
		binary.LittleEndian.PutUint32(data[off+registryEntryLevelOffset:], levels[i])
	}
	return data
}

// TestRegistryFingerprintDomain — CE QUE L'EMPREINTE COUVRE, ET CE QU'ELLE IGNORE.
//
// Elle doit bouger sur chacun des deux champs haches (niveau, nom) et sur l'ORDRE des
// entrees ; elle doit rester STABLE quand seul le BOURRAGE change. Sans le cas du bourrage,
// une empreinte qui hacherait toute l'entree passerait ce test tout en alertant a chaque film.
func TestRegistryFingerprintDomain(t *testing.T) {
	base := registryFixture([]uint32{7, 8}, []string{"alpha-component", "beta-component"})
	ref := RegistryFingerprint(parseRegistry(base))
	if ref == 0 {
		t.Fatal("empreinte nulle sur un registre non vide")
	}

	bouge := []struct {
		nom  string
		data []byte
	}{
		{"niveau different", registryFixture([]uint32{7, 9}, []string{"alpha-component", "beta-component"})},
		{"nom different", registryFixture([]uint32{7, 8}, []string{"alpha-component", "gamma-component"})},
		{"ordre echange", registryFixture([]uint32{8, 7}, []string{"beta-component", "alpha-component"})},
	}
	for _, c := range bouge {
		if got := RegistryFingerprint(parseRegistry(c.data)); got == ref {
			t.Errorf("%s : l'empreinte ne bouge pas (%#016x) — le champ n'entre pas dans le hachage",
				c.nom, got)
		}
	}

	// Le bourrage d'une entree NOMMEE — les octets entre le NUL de fin de nom et le u32 de
	// niveau — ne dit rien de la grammaire et n'entre pas dans l'empreinte.
	bourre := append([]byte(nil), base...)
	for i := registryEntryBase + len("alpha-component") + 1; i < registryEntryBase+registryEntryLevelOffset; i++ {
		bourre[i] = 0xa5
	}
	if got := RegistryFingerprint(parseRegistry(bourre)); got != ref {
		t.Errorf("le bourrage d'une entree nommee change l'empreinte (%#016x != %#016x)", got, ref)
	}

	// La section qui SUIT le registre (bloc entier de bruit) : elle n'entre ni dans les
	// archetypes ni dans l'empreinte — c'est la fin structurelle mesuree au lot 3 (2026-08-30).
	suite := append(append([]byte(nil), base...), bytes.Repeat([]byte{0xa5}, archetypeBlockSize)...)
	if reg := parseRegistry(suite); len(reg.Archetypes) != 1 || RegistryFingerprint(reg) != ref {
		t.Errorf("la section suivant le registre entre dans le parse (%d blocs, %#016x != %#016x)",
			len(reg.Archetypes), RegistryFingerprint(reg), ref)
	}

	// Du bruit DANS le bourrage d'un bloc, apres la suite nommee : ce n'est plus un bloc de
	// registre — le parse s'arrete AVANT lui (regle « suite nommee puis zeros », verifiee sur
	// les 1 367 films du corpus, lot3_registre_compte_research_test.go).
	bruite := append([]byte(nil), base...)
	for i := registryEntryBase + 2*registrySlotSize; i < len(bruite); i++ {
		bruite[i] = 0xa5
	}
	if reg := parseRegistry(bruite); len(reg.Archetypes) != 0 {
		t.Errorf("un bloc au bourrage bruite a ete accepte comme registre (%d blocs)",
			len(reg.Archetypes))
	}
	if RegistryFingerprint(nil) != 0 {
		t.Error("RegistryFingerprint(nil) doit rendre 0")
	}
}

// TestRegistryFingerprintWarnsOnce — LA DEDUPLICATION DE L'ALERTE.
//
// Le registre est re-parse par quatre chemins de production a chaque film : une alerte par
// parse serait du bruit. La garde est `registryWarned`, et le test la verifie EN OBSERVANT
// l'etat de la carte (pas le journal, qui n'est pas interrogeable).
func TestRegistryFingerprintWarnsOnce(t *testing.T) {
	data := registryFixture([]uint32{5}, []string{"temoin-de-deduplication"})
	fp := RegistryFingerprint(parseRegistry(data))
	if fp == KnownRegistryFingerprint {
		t.Fatalf("la fixture porte l'empreinte CONNUE (%#016x) : le test ne mesurerait rien", fp)
	}
	t.Cleanup(func() { registryWarned.Delete(fp) })

	registryWarned.Delete(fp)
	warnUnknownRegistry(fp, 1, 1)
	if _, ok := registryWarned.Load(fp); !ok {
		t.Fatal("la premiere alerte n'a pas marque l'empreinte comme signalee")
	}
	warnUnknownRegistry(fp, 1, 1) // doit etre un no-op

	// L'empreinte CONNUE ne doit jamais entrer dans la carte : elle n'alerte pas.
	warnUnknownRegistry(KnownRegistryFingerprint, 1, 1)
	if _, ok := registryWarned.Load(KnownRegistryFingerprint); ok {
		t.Error("l'empreinte connue a ete traitee comme inconnue")
	}
}

// TestRegistryFingerprintOnFilm — L'EMPREINTE D'UN FILM REEL ET LE COUT DE SON RE-PARSE.
//
// Sous garde `DELTA_WITNESS_FILM` (meme garde que le temoin de marche delta : un film par
// process, chemin absolu). Publie l'empreinte mesuree, la taille INFLATEE du registre, le
// nombre de blocs et de slots non vides, et le cout d'un parse — de quoi chiffrer les quatre
// re-parses par film que le plan demande de compter.
func TestRegistryFingerprintOnFilm(t *testing.T) {
	dir := os.Getenv(deltaWitnessFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : mesure d'empreinte sautee", deltaWitnessFilmEnv)
	}
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 de %s illisible : %v", dir, err)
	}
	brut, err := os.ReadFile(dir + "/chunk_00.bin") //nolint:gosec // chemin fourni par la garde
	if err != nil {
		t.Fatalf("taille compressee de %s : %v", dir, err)
	}

	deb := time.Now()
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre de %s : %v", dir, err)
	}
	duree := time.Since(deb)

	slots := 0
	for _, a := range reg.Archetypes {
		slots += len(a.Components)
	}
	fp := RegistryFingerprint(reg)
	t.Logf("== FILM %s ==", dir)
	t.Logf("  chunk_00 : %d octets compresses -> %d octets INFLATES (x%.1f)",
		len(brut), len(raw), float64(len(raw))/float64(max(1, len(brut))))
	t.Logf("  %d blocs · %d slots non vides · %d archetypes porteurs",
		len(reg.Archetypes), slots, registryCarriers(reg))
	t.Logf("  EMPREINTE mesuree : %#016x · connue : %#016x · concordance : %v",
		fp, KnownRegistryFingerprint, fp == KnownRegistryFingerprint)
	t.Logf("  cout d'UN parse (inflate exclu, deja fait par ReadFilmChunk) : %s", duree)

	// Le cout des N re-parses : mesure sur 4 passes, le compte des chemins de production.
	deb = time.Now()
	const reparses = 4
	for i := 0; i < reparses; i++ {
		if _, err := ParseRegistryChunk(raw); err != nil {
			t.Fatalf("re-parse %d : %v", i, err)
		}
	}
	t.Logf("  cout des %d re-parses de production : %s (soit %s par parse)",
		reparses, time.Since(deb), time.Since(deb)/reparses)

	if bytes.Equal(raw, brut) {
		t.Log("  NOTE : chunk_00 n'etait pas compresse (dump deja inflate)")
	}
}

// registryCarriers compte les archetypes portant au moins un composant.
func registryCarriers(reg *Registry) int {
	n := 0
	for _, a := range reg.Archetypes {
		if len(a.Components) > 0 {
			n++
		}
	}
	return n
}
