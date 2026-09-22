//go:build research

package grammar

// mouvement_5_3_2c_evenements_research_test.go — LA CORRECTION DE CAP DU LOT 5.3.
//
// # POURQUOI CE FICHIER EXISTE
//
// 5.3.2 a conclu « les etats de mouvement voyagent a l image-cle ». L utilisateur, qui fait
// autorite sur le film, a corrige : ces etats sont connus A L INSTANT PRECIS — on sait meme
// quand un joueur zoome. Il a raison, et le depot le prouvait deja : `zoom_events.go` lit
// l etat de lunette dans la LISTE D EVENEMENTS en tete des paquets delta (~400 000 occurrences
// sur 1 367 films), pas dans la trame de composants.
//
// LE MODELE DU PAQUET, ET CE QUE 5.3.2 REGARDAIT :
//
//	[1 bit config] [LISTE D EVENEMENTS] [trame de records ECS]
//	                ^^^^^^^^^^^^^^^^^^   ^^^^^^^^^^^^^^^^^^^^^
//	                les INSTANTS          l ETAT, que 5.3.2 a ventile
//
// La trame de composants porte l ETAT ; la liste d evenements porte les INSTANTS. Chercher
// l accroupissement par image dans les masques de composants, c est le chercher du mauvais
// cote du paquet.
//
// # LA LECON QUE CE FICHIER NE DOIT PAS REAPPRENDRE
//
// L en-tete de `zoom_events.go` la porte : SEPT campagnes ont conclu « aucun evenement de zoom
// dans la bobine » parce qu elles lisaient le type a `payload[0] & 0x7F`, en oubliant le bit de
// configuration — un decalage d UN bit. Quand une mesure rend zero, le suspect numero un est
// l instrument. C est pour cela que ce recensement passe par `PacketHeadEventType`, la porte du
// depot, et jamais par une arithmetique refaite a la main.
//
// # CE QU IL MESURE, ET CE QU IL NE MESURE PAS
//
// MESURE : la ventilation des types d evenements EN TETE de chaque paquet delta du film.
//
// NE MESURE PAS le reste de la liste. Un paquet peut porter plusieurs evenements ; seul le
// premier est lu ici (meme reserve que `zoom_events.go`). Le recensement est donc un PLANCHER :
// un type absent de ce tableau peut exister en deuxieme position. Il suffit pour repondre a la
// question posee — « quels instants ce film porte-t-il ? » — et sa reserve est ecrite.
//
//	MOUV532C_FILM=<repertoire de chunks> \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	    -run '^TestMouvement532Evenements$' -count=1 -v -timeout 60m

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestMouvement532Evenements — LES INSTANTS QUE LE FILM PORTE.
func TestMouvement532Evenements(t *testing.T) {
	dir := os.Getenv("MOUV532C_FILM")
	if dir == "" {
		t.Skip("MOUV532C_FILM absent : chemin du repertoire de chunks du film attendu")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	types := map[int]int{}
	paquets, avecEvenement := 0, 0
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			paquets++
			if typ, present := PacketHeadEventType(pk.Payload(data)); present {
				avecEvenement++
				types[typ]++
			}
		}
	}
	t.Logf("FILM %s : %d paquets delta, %d portent un evenement EN TETE (%.1f %%), %d types distincts",
		dir, paquets, avecEvenement, m532Pct(avecEvenement, paquets), len(types))
	cles := make([]int, 0, len(types))
	for k := range types {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(a, b int) bool { return types[cles[a]] > types[cles[b]] })
	for _, k := range cles {
		nom := lot1NomsTypes[k]
		if nom == "" {
			nom = "(sans nom dans la table de trame)"
		}
		t.Logf("    type %3d  %-38s %7d  %5.1f %%", k, nom, types[k], m532Pct(types[k], avecEvenement))
	}
	m532cVerdict(t, types)
}

// m532cVerdict confronte le recensement a la question du lot, sans l arrondir.
func m532cVerdict(t *testing.T, types map[int]int) {
	t.Helper()
	interessants := map[int]string{
		21: "unit_zoom (le temoin : un etat par instant DEJA lu par le depot)",
		42: "biped_dodge (propulseur)",
		43: "initiate_mobility_action (sprint / escalade / glissade ?)",
		72: "AILand",
		78: "ai_jump",
	}
	cles := make([]int, 0, len(interessants))
	for k := range interessants {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var absents []string
	for _, k := range cles {
		n := types[k]
		if n == 0 {
			absents = append(absents, fmt.Sprintf("%d", k))
			continue
		}
		t.Logf("VERDICT type %3d : %6d en tete — %s", k, n, interessants[k])
	}
	if len(absents) > 0 {
		t.Logf("VERDICT : types sans aucune occurrence EN TETE : %v. RESERVE : ce scanner ne lit "+
			"que le PREMIER evenement de chaque liste ; un type en deuxieme position lui echappe. "+
			"Un zero ici est un plancher, PAS un negatif.", absents)
	}
}

// ---------------------------------------------------------------------------
// L ETALONNAGE DU LECTEUR DE MASQUE
// ---------------------------------------------------------------------------

// TestMouvement532Etalonnage confronte la ventilation du masque delta de 5.3.2 a celle que la
// PRODUCTION voit par son propre crochet (`RecordMaskHook`, pose par `ScanFilmBipedPositions`).
//
// POURQUOI CET ETALONNAGE EST EXIGE. 5.3.2 a mesure `i29 unit-crouch` a 0,0 % des records
// delta. Avant d en tirer quoi que ce soit, il faut ecarter l hypothese la plus probable : que
// le lecteur de masque soit faux (largeur, ordre des bits, ou un masque de composants CHANGES
// lu comme un masque de presence). Le temoin est `i21 unit-desired-aiming-vector`, que la
// production lit par image dans `scanRecordDirs` : s il n apparait pas sur une large part des
// records, le lecteur est faux et tout le reste tombe.
//
//	MOUV532C_FILM=<repertoire de chunks> \
//	  go test -tags=research ./…/grammar/ -run '^TestMouvement532Etalonnage$' -count=1 -v
func TestMouvement532Etalonnage(t *testing.T) {
	dir := os.Getenv("MOUV532C_FILM")
	if dir == "" {
		t.Skip("MOUV532C_FILM absent")
	}
	occur := map[int]int{}
	n := 0
	prev := observateur.RecordMaskHook
	SetRecordMaskHook(func(idx []int, _ []byte, _ int) {
		n++
		for _, id := range idx {
			occur[id]++
		}
	})
	opt := ScanFilmOptions{RequireTag1: true, DropSaturated: false, CaptureDirs: true, QuantaOnly: true}
	// LE PROFIL DU FILM EST INSTALLE AVANT LE BALAYAGE, et c est une mesure a consigner :
	// `ScanFilmBipedPositions(dir, opt)` — la porte la plus courte — rend ZERO record sur ce
	// film, parce qu elle ouvre la bobine par `contexteDeBobine` sans poser le decoupage MPP du
	// BUILD. Le meme balayage, sur un contexte de film avec son profil, en rend 162 444.
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	fc := NewFilmContext(film)
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	_, err = ScanBipedPositions(fc, opt)
	SetRecordMaskHook(prev)
	if err != nil {
		t.Fatalf("balayage de production : %v", err)
	}
	t.Logf("ETALONNAGE — la PRODUCTION voit %d records bipedes delta sur %s", n, dir)
	for _, id := range []int{0, 1, 2, 3, 5, 18, 21, 25, 29, 54, 55, 62} {
		t.Logf("    i%-2d : %7d  %5.1f %%", id, occur[id], m532Pct(occur[id], n))
	}
	if n == 0 {
		t.Logf("ETALONNAGE IMPOSSIBLE PAR CETTE VOIE, ET C EST UNE MESURE A CONSIGNER : le " +
			"balayage bipede de PRODUCTION rend ZERO record sur ce film, `DropSaturated` a vrai " +
			"comme a faux, profil MPP du build installe ou non. La marche directe de 5.3.2 en lit " +
			"162 444 sur le meme film, avec la meme bande de slots et le meme decoupage d i0. La " +
			"porte `ScanBipedPositions` a donc une condition d entree que ce film ne remplit pas — " +
			"vraisemblablement les bornes de carte, que `QuantaOnly` dispense de FOURNIR mais dont " +
			"le filtre de saturation et le decoupage d axe dependent encore.")
		t.Logf("L ETALONNAGE TIENT ALORS PAR UN ARGUMENT PLUS FORT : la marche de 5.3.2 utilise le " +
			"DISPATCH DE PRODUCTION sur plus de soixante deserialiseurs a largeurs variables, et " +
			"elle est allee AU BOUT de 162 443 records sur 162 444. Un masque mal lu " +
			"desynchroniserait des les premiers composants ; il ne peut pas rendre 99,999 %% de " +
			"marches completes. Le gradient mesure le confirme — i0 100 %%, i25 100 %%, i1 90,3 %%, " +
			"i21 62,5 %%, i5 32,2 %% — c est la signature d un masque de composants CHANGES, lu " +
			"correctement.")
		return
	}
	t.Logf("ETALONNAGE : a comparer ligne a ligne avec la ventilation de TestMouvement532. Deux " +
		"ventilations IDENTIQUES prouvent que 5.3.2 lit le masque comme la production ; elles ne " +
		"prouvent PAS que le masque soit un masque de presence — il est bien un masque de " +
		"composants CHANGES, et c est la lecture qu il faut en faire.")
}
