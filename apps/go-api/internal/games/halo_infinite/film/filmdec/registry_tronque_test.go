package filmdec

// registry_tronque_test.go — UN CHUNK_00 TRONQUE NE FAIT PAS TOMBER LE PROCESSUS.
//
// # LE DEFAUT QUE CE FICHIER FERME (revue R1 du lot 1.2, constat C1, 2026-09-14)
//
// Le recadrage du registre a l'octet 8 avait relache la borne de la boucle de blocs : elle
// acceptait d'entrer dans un bloc INCOMPLET. Sur un tampon qui s'arrete au milieu d'un nom, la
// suite nommee du bloc se prolonge au-dela du tampon, et `registryBlockTail` demandait alors
// `zeroTail(data, from, to)` avec `from > to` — c'est-a-dire `data[from:to]`, une PANIQUE
// (`slice bounds out of range`). Aucun `recover` ne la rattrape ni dans `film/` ni dans
// `killcollector/` : les deux appelants de production (`killcollector/hits.go`,
// `filmdec/film_context.go`) auraient fait tomber le processus au lieu de rendre un registre.
// L'ancienne borne (blocs ENTIERS seulement) n'avait pas ce defaut, et le recadrage ne
// demandait pas de la changer.
//
// # LES DEUX REPRODUCTIONS, TELLES QUE LE RELECTEUR LES A JOUEES
//
//	(A) SYNTHETIQUE, 13 octets : l'en-tete de 8 octets, puis un nom de 4 caracteres et son NUL.
//	    La suite nommee vaut 1, la queue commence a 268 et le tampon s'arrete a 13 -> `[268:13]`.
//	(B) REGISTRE REEL COUPE : `chunk_00` du build de reference tronque a
//	    `registryEntryBase + 6*archetypeBlockSize + 41*registrySlotSize + 2`, soit 110 510 —
//	    deux octets DANS le nom de l'entree 41 du bloc 6 (`statborg-finalized-rounds-values-…`,
//	    dont il ne reste que « st »). La suite nommee vaut 42, la queue commence a 110 768
//	    -> `[110768:110510]`. Mesure du 2026-09-14 : le relecteur l'a jouee sur `53ce4390`, le
//	    registre etant bit-a-bit identique dans un build, la bobine VERSIONNEE de `killsource`
//	    porte les memes octets et rend le meme offset — le test ne depend donc d'aucun cache.
//
// # CE QUE LE TEST EXIGE
//
// Pas de panique, le registre des blocs ENTIERS rendu, et `Registry.TruncatedBytes` egal aux
// octets de queue qu'aucun bloc entier ne couvre. Une troncature n'est pas muette (D14).

import (
	"path/filepath"
	"testing"
)

// registreDeReference rend les octets INFLATES du `chunk_00` de la bobine versionnee du build
// de reference — celle sur laquelle `KnownRegistryFingerprint` est calculee.
func registreDeReference(t *testing.T) []byte {
	t.Helper()
	d, err := ReadFilmChunk(filepath.Join("..", "killsource", "testdata", "minibobine_000d5950"), 0)
	if err != nil {
		t.Fatalf("chunk_00 de la bobine de reference illisible : %v", err)
	}
	return d
}

// coupeAuMilieuDUnNom : l'offset de la reproduction (B), ecrit comme le relecteur l'a derive.
const coupeAuMilieuDUnNom = registryEntryBase + 6*archetypeBlockSize + 41*registrySlotSize + 2

// TestParseRegistreTronqueNePaniquePas tient les deux reproductions.
func TestParseRegistreTronqueNePaniquePas(t *testing.T) {
	reel := registreDeReference(t)
	if len(reel) <= coupeAuMilieuDUnNom {
		t.Fatalf("la bobine de reference ne fait que %d octets : la coupe a %d ne mesure rien",
			len(reel), coupeAuMilieuDUnNom)
	}
	synthetique := make([]byte, registryEntryBase)
	synthetique = append(synthetique, []byte("abcd")...)
	synthetique = append(synthetique, 0)

	for _, cas := range []struct {
		nom             string
		data            []byte
		blocs           int
		queue           int
		nomDeLaPremiere string
	}{
		{"(A) synthetique de 13 octets", synthetique, 0, 5, ""},
		{"(B) registre reel coupe au milieu du nom de l'entree 41 du bloc 6",
			reel[:coupeAuMilieuDUnNom], 6, 10662, "game-engine-team-mapping-component"},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			reg := parseRegistry(cas.data)
			if got := len(reg.Archetypes); got != cas.blocs {
				t.Errorf("%d bloc(s) entier(s) rendu(s), %d attendu(s)", got, cas.blocs)
			}
			if reg.TruncatedBytes != cas.queue {
				t.Errorf("TruncatedBytes = %d, %d attendu — une troncature ne se tait pas",
					reg.TruncatedBytes, cas.queue)
			}
			if cas.nomDeLaPremiere == "" {
				return
			}
			a, ok := reg.Archetype(0)
			if !ok || a.component(0) != cas.nomDeLaPremiere {
				t.Errorf("bloc 0 i0 = %q (present=%v), %q attendu — les blocs entiers doivent "+
					"rester lisibles", a.component(0), ok, cas.nomDeLaPremiere)
			}
		})
	}
}

// TestParseRegistreCompletNEstPasTronque : le cas nominal ne declare aucune queue. Sans lui,
// `TruncatedBytes` pourrait valoir n'importe quoi sur un registre sain sans que rien ne le dise.
func TestParseRegistreCompletNEstPasTronque(t *testing.T) {
	reg := parseRegistry(registreDeReference(t))
	if reg.TruncatedBytes != 0 {
		t.Errorf("TruncatedBytes = %d sur un registre complet, 0 attendu", reg.TruncatedBytes)
	}
	if len(reg.Archetypes) != registreBlocsAttendus {
		t.Errorf("%d archetypes, %d attendus", len(reg.Archetypes), registreBlocsAttendus)
	}
}
