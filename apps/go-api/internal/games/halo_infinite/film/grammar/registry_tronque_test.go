package grammar

// registry_tronque_test.go — UN CHUNK_00 TRONQUE NE FAIT PAS TOMBER LE PROCESSUS.
//
// # LE DEFAUT QUE CE FICHIER FERME (revue R1 du lot 1.2, constat C1, 2026-09-14)
//
// Le recadrage du registre a l'octet 8 avait relache la borne de la boucle de blocs : elle
// acceptait d'entrer dans un bloc INCOMPLET. Sur un tampon qui s'arrete au milieu d'un nom, la
// suite nommee du bloc se prolonge au-dela du tampon, et `registryBlockTail` demandait alors
// `zeroTail(data, from, to)` avec `from > to` — c'est-a-dire `data[from:to]`, une PANIQUE
// (`slice bounds out of range`). Aucun `recover` ne la rattrape, et les appelants de production
// sont TROIS — releve du 2026-09-14,
// `grep -rn "ParseRegistryChunk(" --include=*.go internal/ cmd/ | grep -v _test.go` :
// `filmdec/film_context.go:254`, `killsource/world.go:58` (via `killsource/decode.go:123`,
// paquet importe par `killcollector` ET par `replaybuild`) et `killcollector/hits.go:113`.
// Les trois auraient fait tomber le processus au lieu de rendre un registre.
// L'ancienne borne (blocs ENTIERS seulement) n'avait pas ce defaut, et le recadrage ne
// demandait pas de la changer.
//
// # LES DEUX REPRODUCTIONS, TELLES QUE LE RELECTEUR LES A JOUEES
//
//	(A) SYNTHETIQUE, 13 octets : l'en-tete de 8 octets, puis un nom de 4 caracteres et son NUL.
//	    La suite nommee vaut 1, la queue commence a 268 et le tampon s'arrete a 13 -> `[268:13]`.
//	(C) COUPE ALIGNEE SUR UNE FRONTIERE DE BLOC : le meme `chunk_00` tronque a
//	    `registryEntryBase + 6*archetypeBlockSize`. Aucun octet de queue — et c'est le piege que
//	    la revue R2 a trouve : `TruncatedBytes` vaut ZERO, donc lui seul redevenait muet. C'est
//	    `Truncated` qui porte le fait, parce que le parse a EPUISE le tampon sans rencontrer la
//	    fin structurelle du registre.
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
// Pas de panique, le registre des blocs ENTIERS rendu, `Registry.Truncated` vrai, et
// `Registry.TruncatedBytes` egal aux octets de queue qu'aucun bloc entier ne couvre — ZERO
// compris. Une troncature n'est pas muette (D14).

import (
	"path/filepath"
	"testing"
)

// registreDeReference rend les octets INFLATES du `chunk_00` de la bobine versionnee du build
// de reference — celle sur laquelle `KnownRegistryFingerprint` est calculee.
func registreDeReference(t *testing.T) []byte {
	t.Helper()
	d, err := ReadFilmChunk(filepath.Join("..", "facts", "killsource", "testdata", "minibobine_000d5950"), 0)
	if err != nil {
		t.Fatalf("chunk_00 de la bobine de reference illisible : %v", err)
	}
	return d
}

// coupeAuMilieuDUnNom : l'offset de la reproduction (B), ecrit comme le relecteur l'a derive.
const coupeAuMilieuDUnNom = registryEntryBase + 6*archetypeBlockSize + 41*registrySlotSize + 2

// coupeSurFrontiereDeBloc : l'offset de la reproduction (C), une coupe SANS octet de queue.
const coupeSurFrontiereDeBloc = registryEntryBase + 6*archetypeBlockSize

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
		{"(C) registre reel coupe SUR une frontiere de bloc (zero octet de queue)",
			reel[:coupeSurFrontiereDeBloc], 6, 0, "game-engine-team-mapping-component"},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			reg := parseRegistry(cas.data)
			if got := len(reg.Archetypes); got != cas.blocs {
				t.Errorf("%d bloc(s) entier(s) rendu(s), %d attendu(s)", got, cas.blocs)
			}
			if !reg.Truncated {
				t.Errorf("Truncated = false sur un tampon coupe — c'est le drapeau qui porte le "+
					"fait, TruncatedBytes n'en est que la mesure (elle vaut %d ici)",
					reg.TruncatedBytes)
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
	if reg.Truncated {
		t.Error("Truncated = true sur un registre complet : la lecture s'y arrete sur la fin " +
			"structurelle, elle n'epuise pas le tampon")
	}
	if reg.TruncatedBytes != 0 {
		t.Errorf("TruncatedBytes = %d sur un registre complet, 0 attendu", reg.TruncatedBytes)
	}
	if len(reg.Archetypes) != registreBlocsAttendus {
		t.Errorf("%d archetypes, %d attendus", len(reg.Archetypes), registreBlocsAttendus)
	}
}
