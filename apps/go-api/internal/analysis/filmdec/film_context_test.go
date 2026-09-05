package filmdec

// film_context_test.go — LE CONTEXTE REND EXACTEMENT CE QUE LE RECALCUL DIRECT RENDAIT.
//
// C'est le test qui autorise la migration du lot 2 : si `FilmContext` differait d'un slot, d'un
// bit de decoupage ou d'un archetype de ce que chaque balayage calculait pour son compte, les
// huit balayages migres changeraient de sortie en silence — memes signatures, memes appels, et
// une divergence qui ne se verrait qu'au digest du corpus. Le meme role que
// `film_chunks_test.go` a joue pour le lot 1.
//
// # DEUX ETAGES, ET C'EST VOULU
//
//	LA MINI-BOBINE (toujours, CI comprise)  prouve l'egalite des VALEURS ET DES ECHECS. Elle n'a
//	    ni `chunk_00` ni slot bipede aux images-cles (cf. PROVENANCE.txt et
//	    `replay/zero_disque_test.go`) : c'est precisement le film ou les trois derivations
//	    ECHOUENT, et ou l'on verifie que le contexte rejoue l'echec au lieu de le lisser.
//	UN VRAI FILM (si `FILM_CONTEXT_FILM` designe un repertoire de chunks)  prouve l'egalite sur
//	    les valeurs NON VIDES — bande de plusieurs dizaines de slots, decoupage detecte, registre
//	    analyse, six archetypes. La CI n'a pas de film ; le cache local du depot en a 1 380.
//
//	CGO_ENABLED=0 FILM_CONTEXT_FILM=<depot>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run FilmContext -count=1 -v

import (
	"errors"
	"os"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// memeBande dit si deux bandes de slots sont identiques, et nomme le premier ecart.
func memeBande(t *testing.T, quoi string, got, want SlotBand) {
	t.Helper()
	if got.Count() != want.Count() {
		t.Fatalf("%s : %d slots, le recalcul direct en donne %d", quoi, got.Count(), want.Count())
	}
	for _, s := range want.Slots() {
		if !got.Has(s) {
			t.Fatalf("%s : slot %d absent, le recalcul direct le donne present", quoi, s)
		}
	}
}

// memeErreur dit si deux erreurs sont la meme — meme nil-ite, meme message.
func memeErreur(t *testing.T, quoi string, got, want error) {
	t.Helper()
	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		t.Fatalf("%s : erreur %v, le recalcul direct rend %v", quoi, got, want)
	case got.Error() != want.Error():
		t.Fatalf("%s : erreur %q, le recalcul direct rend %q", quoi, got, want)
	}
}

// comparerContexteAuRecalcul verifie les trois derivations d'un film, contexte contre recalcul
// direct. C'est le coeur des deux tests ci-dessous ; il ne suppose RIEN du film (bobine partielle
// ou film complet, il compare ce qui sort).
func comparerContexteAuRecalcul(t *testing.T, film *filmsource.Film) {
	t.Helper()
	fc := NewFilmContext(film)

	// (1) les numeros de chunks.
	nums, attendus := fc.ChunkNumbers(), FilmChunkNumbers(film)
	if len(nums) != len(attendus) {
		t.Fatalf("numeros de chunk : %v, le recalcul direct donne %v", nums, attendus)
	}
	for i := range attendus {
		if nums[i] != attendus[i] {
			t.Fatalf("numero %d : %d, le recalcul direct dit %d", i, nums[i], attendus[i])
		}
	}

	// (2) la bande de slots bipede — l'appel EXACT que faisaient les huit balayages.
	var bandeDirecte SlotBand
	if len(attendus) > 0 {
		bandeDirecte = bipedSlotBand(film, attendus)
	}
	memeBande(t, "bande bipede", fc.BipedSlots(), bandeDirecte)

	// (3) le decoupage d'i0 — valeur ET erreur.
	layDirect, _, errDirect := DetectI0LayoutOf(film)
	lay, err := fc.I0Layout()
	memeErreur(t, "decoupage i0", err, errDirect)
	if lay != layDirect {
		t.Fatalf("decoupage i0 : %s, le recalcul direct dit %s", lay, layDirect)
	}

	// (4) le registre chunk_00 — valeur ET erreur.
	reg, errReg := fc.Registry()
	raw, ok := FilmRegistryChunk(film)
	switch {
	case !ok:
		memeErreur(t, "registre", errReg, ErrNoRegistryChunk)
		if reg != nil {
			t.Fatal("registre : le contexte rend un registre alors que le film n'en porte pas")
		}
	default:
		regDirect, errRegDirect := ParseRegistryChunk(raw)
		memeErreur(t, "registre", errReg, errRegDirect)
		comparerRegistres(t, reg, regDirect)
	}

	// (5) LA MEMOISATION NE CHANGE RIEN AU SECOND APPEL — et rend la MEME bande, pas une copie :
	// c'est ce qui fait que le gain est reel plutot qu'un recalcul deguise.
	memeBande(t, "bande bipede (2e appel)", fc.BipedSlots(), bandeDirecte)
	lay2, err2 := fc.I0Layout()
	memeErreur(t, "decoupage i0 (2e appel)", err2, errDirect)
	if lay2 != layDirect {
		t.Fatalf("decoupage i0 (2e appel) : %s, attendu %s", lay2, layDirect)
	}
	reg2, errReg2 := fc.Registry()
	memeErreur(t, "registre (2e appel)", errReg2, errReg)
	if reg2 != reg {
		t.Fatal("registre : le 2e appel rend une AUTRE instance — le chunk_00 est re-analyse")
	}
}

// comparerRegistres compare deux registres champ pour champ, empreinte comprise.
func comparerRegistres(t *testing.T, got, want *Registry) {
	t.Helper()
	if got == nil || want == nil {
		t.Fatalf("registre : got=%v want=%v", got == nil, want == nil)
	}
	if got.fingerprint != want.fingerprint {
		t.Fatalf("registre : empreinte %d, le recalcul direct dit %d", got.fingerprint, want.fingerprint)
	}
	if len(got.Archetypes) != len(want.Archetypes) {
		t.Fatalf("registre : %d archetypes, le recalcul direct en donne %d",
			len(got.Archetypes), len(want.Archetypes))
	}
	for i := range want.Archetypes {
		a, b := got.Archetypes[i], want.Archetypes[i]
		if len(a.Components) != len(b.Components) {
			t.Fatalf("archetype %d : %d composants, le recalcul direct en donne %d",
				i, len(a.Components), len(b.Components))
		}
		for k := range b.Components {
			if a.Components[k] != b.Components[k] || a.Flags[k] != b.Flags[k] {
				t.Fatalf("archetype %d, composant %d : %q/%d, le recalcul direct dit %q/%d",
					i, k, a.Components[k], a.Flags[k], b.Components[k], b.Flags[k])
			}
		}
	}
}

// TestFilmContextEgaleLeRecalculDirectMiniBobine — l'egalite des ECHECS, sur la fixture versionnee.
func TestFilmContextEgaleLeRecalculDirectMiniBobine(t *testing.T) {
	film := chargerMiniBobine(t)
	comparerContexteAuRecalcul(t, film)

	// La bobine n'a PAS de registre : les six accesseurs d'archetype doivent tous rendre
	// ErrNoRegistryChunk — c'est ce qui garantit qu'un film partiel ne se lit pas comme un film
	// dont l'archetype manquerait au build.
	fc := NewFilmContext(film)
	if _, err := fc.bipedArchetype(); !errors.Is(err, ErrNoRegistryChunk) {
		t.Fatalf("archetype biped : %v, attendu ErrNoRegistryChunk", err)
	}
	if _, err := fc.EquipmentArchetype(); !errors.Is(err, ErrNoRegistryChunk) {
		t.Fatalf("archetype equipement : %v, attendu ErrNoRegistryChunk", err)
	}
	if _, err := fc.groundWeaponArchetype(); !errors.Is(err, ErrNoRegistryChunk) {
		t.Fatalf("archetype arme au sol : %v, attendu ErrNoRegistryChunk", err)
	}
	if _, err := fc.managedPropertyArchetype(); !errors.Is(err, ErrNoRegistryChunk) {
		t.Fatalf("archetype ti=13 : %v, attendu ErrNoRegistryChunk", err)
	}
	if _, _, err := fc.filmArchetype(navpointRadialArchIndex); !errors.Is(err, ErrNoRegistryChunk) {
		t.Fatalf("archetype ti=12 : %v, attendu ErrNoRegistryChunk", err)
	}
	if _, _, err := fc.objectiveArchetype(); !errors.Is(err, ErrNoRegistryChunk) {
		t.Fatalf("archetype ti=11 : %v, attendu ErrNoRegistryChunk", err)
	}
}

// TestFilmContextNilNeParaniquePas — un film nil traverse les memes portes qu'un repertoire vide.
func TestFilmContextNilNePaniquePas(t *testing.T) {
	fc := NewFilmContext(nil)
	if n := fc.ChunkNumbers(); len(n) != 0 {
		t.Fatalf("film nil : %d numeros de chunk, attendu 0", len(n))
	}
	if b := fc.BipedSlots(); b.Count() != 0 {
		t.Fatalf("film nil : %d slots, attendu 0", b.Count())
	}
	if _, err := fc.I0Layout(); err == nil {
		t.Fatal("film nil : le decoupage d'i0 devrait refuser")
	}
	if _, err := fc.Registry(); !errors.Is(err, ErrNoRegistryChunk) {
		t.Fatalf("film nil : registre %v, attendu ErrNoRegistryChunk", err)
	}
	if _, _, ok := fc.ChunkAt(1); ok {
		t.Fatal("film nil : ChunkAt devrait rendre ok=false")
	}
}

// TestFilmContextEgaleLeRecalculDirectVraiFilm — l'egalite des VALEURS NON VIDES, sur un film du
// cache local. Ignore sans `FILM_CONTEXT_FILM` (la CI n'a pas de film).
func TestFilmContextEgaleLeRecalculDirectVraiFilm(t *testing.T) {
	dir := os.Getenv("FILM_CONTEXT_FILM")
	if dir == "" {
		t.Skip("FILM_CONTEXT_FILM absent : ce controle demande un repertoire de chunks reel")
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chunks du film %s illisibles : %v", dir, err)
	}
	comparerContexteAuRecalcul(t, film)

	// Le film doit etre ASSEZ RICHE pour que la comparaison ait un sens : sans bande, sans
	// decoupage et sans registre, le test ci-dessus ne comparerait que des vides.
	fc := NewFilmContext(film)
	if fc.BipedSlots().Count() == 0 {
		t.Fatal("bande bipede vide : ce film ne prouve rien de plus que la mini-bobine")
	}
	if _, err := fc.I0Layout(); err != nil {
		t.Fatalf("decoupage i0 non detecte (%v) : ce film ne prouve rien de plus", err)
	}
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre absent (%v) : ce film ne prouve rien de plus", err)
	}

	// Les six accesseurs d'archetype rendent ce que la lecture directe du registre rend.
	type cas struct {
		nom string
		ti  int
		got func() (Archetype, error)
	}
	for _, c := range []cas{
		{"biped", BipedTypeIndex, fc.bipedArchetype},
		{"equipement", EquipmentTypeIndex, fc.EquipmentArchetype},
		{"arme au sol", GroundWeaponTypeIndex, fc.groundWeaponArchetype},
		{"propriete reseau", ManagedPropertyTypeIndex, fc.managedPropertyArchetype},
		{"navpoint", navpointRadialArchIndex, func() (Archetype, error) {
			a, _, e := fc.filmArchetype(navpointRadialArchIndex)
			return a, e
		}},
		{"objectif", ObjectiveTypeIndex, func() (Archetype, error) {
			a, _, e := fc.objectiveArchetype()
			return a, e
		}},
	} {
		arch, err := c.got()
		attendu, ok := reg.Archetype(c.ti)
		if !ok {
			if err == nil {
				t.Fatalf("archetype %s : le contexte le rend alors que le registre ne le porte pas", c.nom)
			}
			continue
		}
		if err != nil {
			t.Fatalf("archetype %s : %v, le registre le porte pourtant", c.nom, err)
		}
		if len(arch.Components) != len(attendu.Components) {
			t.Fatalf("archetype %s : %d composants, le registre en donne %d",
				c.nom, len(arch.Components), len(attendu.Components))
		}
		for k := range attendu.Components {
			if arch.Components[k] != attendu.Components[k] {
				t.Fatalf("archetype %s, composant %d : %q, le registre dit %q",
					c.nom, k, arch.Components[k], attendu.Components[k])
			}
		}
	}
}
