package filmdec

// profil_roster_research_test.go — PHASE 4, QUESTION 3 : POURQUOI LE LECTEUR DE LA TABLE DES
// SLOTS PERD DES ENREGISTREMENTS SUR LES GROS ROSTERS.
//
// LE CONSTAT, DEJA CONSIGNE DEUX FOIS. Phase 2 section G.3 puis phase 3 section I.4 : le
// balayage de `section3_roster_research_test.go` rend 11 a 23 enregistrements la ou le match en
// compte 24 a 30, et la phase 3 a contourne par un repli au lieu de traiter la cause.
//
// L'ORACLE INTERNE, et il ne coute rien : le nombre d'entites **ti=9** d'une image-cle. La
// phase 3 a etabli que ce sont EXACTEMENT les joueurs (8 en arene, 24 en Grande bataille, slots
// consecutifs de pas 2), et il est lu dans la TRAME, pas dans `chunk_00` : les deux sources sont
// independantes. C'est lui qui dit combien d'enregistrements la table doit porter, sans ouvrir
// aucune base.
//
// LES QUATRE CAUSES CANDIDATES, ecrites avant la mesure (une seule sera retenue, les autres
// seront ecartees PAR UNE MESURE, pas par choix) :
//
//	D1 L'EN-TETE. `s3rBalayage` exige `booleens 1/0/0`, `u32 nul`, `champ de 2 bits nul`. Si un
//	   slot porte une autre valeur sur l'un de ces champs, il est INVISIBLE au balayage.
//	D2 LE FILTRE DE `s3rGrappe`. Il rejette `xuid == borne basse` et `token48 == 0`.
//	D3 LA GRAPPE TERMINALE. `s3rGrappe` remonte tant que l'ecart au precedent reste sous
//	   `s3rEcartMax = 40 000` bits. Un seul enregistrement invisible double l'ecart et peut
//	   depasser le seuil : la grappe perd alors TOUTE SA TETE.
//	D4 LA PLAGE DE XUID. `[0x0009000000000000, 0x000A000000000000)`. Un joueur hors plage
//	   (compte de bot, compte migre) serait invisible.
//
// LA MESURE EST DIFFERENTIELLE : on relache UN critere a la fois et on compte. Celui dont la
// levee rend le compte attendu est la cause ; les autres sont ecartes avec leur chiffre.
//
// Gardes CHUNK00_FILMS (+ CHUNK00_CORPUS pour le controle d'echelle). Aucun code de production
// touche.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// profilRosterCrit dit quels criteres du balayage sont EXIGES. Tous vrais = le balayage
// d'origine (`s3rBalayage` + `s3rGrappe`).
type profilRosterCrit struct {
	Booleens    bool // b0/b1/b2 == 1/0/0
	U32Nul      bool // le u32 de `slot+0x04` est nul
	Deux        bool // le champ de 2 bits de `slot+0x08` est nul
	TokenNonNul bool // le jeton de 48 bits n'est pas nul
	EcartMax    int  // seuil de la grappe terminale ; 0 = pas de regroupement
}

// profilRosterCritPlein rend les criteres d'origine.
func profilRosterCritPlein() profilRosterCrit {
	return profilRosterCrit{Booleens: true, U32Nul: true, Deux: true,
		TokenNonNul: true, EcartMax: s3rEcartMax}
}

// profilRosterBalaye rejoue le balayage avec les criteres donnes, puis, si `EcartMax > 0`,
// la grappe terminale. Aucune autre difference avec `s3rBalayage` / `s3rGrappe`.
func profilRosterBalaye(d []byte, c profilRosterCrit) []s3rTouche {
	fin := (dernierNonNul(d) + 1) * 8
	var out []s3rTouche
	for p := s3rCorpsBit; p+s3rEnteteBits+64 <= fin; p++ {
		if c.Booleens && s3rBit(d, p, 3) != 4 {
			continue
		}
		if c.U32Nul && s3rBit(d, p+3, 32) != 0 {
			continue
		}
		if c.Deux && s3rBit(d, p+35, 2) != 0 {
			continue
		}
		x := s3rBit(d, p+s3rEnteteBits, 64)
		if x < s3rXuidLo || x >= s3rXuidHi {
			continue
		}
		tok := s3rBit(d, p+37, 48)
		if c.TokenNonNul && tok == 0 {
			continue
		}
		if x == s3rXuidLo {
			continue
		}
		out = append(out, s3rTouche{bit: p + s3rEnteteBits, xuid: x, token: tok})
	}
	if c.EcartMax <= 0 {
		return out
	}
	return profilRosterGrappe(out, c.EcartMax)
}

// profilRosterGrappe est `s3rGrappe` avec un seuil parametrable (le filtre de valeurs
// degenerees est deja applique par `profilRosterBalaye`).
func profilRosterGrappe(f []s3rTouche, seuil int) []s3rTouche {
	if len(f) == 0 {
		return nil
	}
	deb := len(f) - 1
	for deb > 0 && f[deb].bit-f[deb-1].bit <= seuil {
		deb--
	}
	return f[deb:]
}

// profilRosterAttendu rend le nombre d'entites ti=9 de la trame : l'ORACLE INTERNE.
func profilRosterAttendu(dir string) int {
	v, err := equipePrepare(dir)
	if err != nil || !v.Longueur {
		return 0
	}
	return v.Card
}

// profilRosterXuids rend les XUID d'une liste de touches, dans l'ordre des positions.
func profilRosterXuids(hs []s3rTouche) []uint64 {
	out := make([]uint64, 0, len(hs))
	for _, h := range hs {
		out = append(out, h.xuid)
	}
	return out
}

// TestProfilRosterCause est la mesure differentielle : un critere releve a la fois.
func TestProfilRosterCause(t *testing.T) {
	variantes := []struct {
		nom string
		c   profilRosterCrit
	}{
		{"origine", profilRosterCritPlein()},
		{"sans regroupement", func() profilRosterCrit {
			c := profilRosterCritPlein()
			c.EcartMax = 0
			return c
		}()},
		{"seuil x4", func() profilRosterCrit {
			c := profilRosterCritPlein()
			c.EcartMax = 4 * s3rEcartMax
			return c
		}()},
		{"sans booleens", func() profilRosterCrit {
			c := profilRosterCritPlein()
			c.Booleens = false
			return c
		}()},
		{"sans u32 nul", func() profilRosterCrit {
			c := profilRosterCritPlein()
			c.U32Nul = false
			return c
		}()},
		{"sans 2 bits nuls", func() profilRosterCrit {
			c := profilRosterCritPlein()
			c.Deux = false
			return c
		}()},
		{"sans jeton non nul", func() profilRosterCrit {
			c := profilRosterCritPlein()
			c.TokenNonNul = false
			return c
		}()},
	}
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		att := profilRosterAttendu(dir)
		build, _ := s3bBuild(d)
		var parties []string
		for _, v := range variantes {
			parties = append(parties, fmt.Sprintf("%s=%d", v.nom, len(profilRosterBalaye(d, v.c))))
		}
		t.Logf("%-10s build %-10q : attendu (entites ti=%d) %2d | %v",
			filepath.Base(dir), build, equipeTI, att, parties)
	}
}

// TestProfilRosterEcarts publie, pour chaque film, la SUITE DES ECARTS entre touches
// consecutives du balayage sans regroupement. C'est la mesure qui dit si D3 (la grappe
// terminale) est la cause : un ecart au-dela du seuil coupe la tete de la table.
func TestProfilRosterEcarts(t *testing.T) {
	c := profilRosterCritCorrigee()
	c.EcartMax = 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		hs := profilRosterBalaye(d, c)
		att := profilRosterAttendu(dir)
		var ecarts []int
		depasse := 0
		for i := 0; i+1 < len(hs); i++ {
			e := hs[i+1].bit - hs[i].bit
			ecarts = append(ecarts, e)
			if e > s3rEcartMax {
				depasse++
			}
		}
		t.Logf("%-10s : attendu %2d, balayage brut %2d ; %d ecart(s) au-dela de %d ; ecarts %v",
			filepath.Base(dir), att, len(hs), depasse, s3rEcartMax, ecarts)
	}
}

// profilRosterTable est le lecteur CORRIGE, complet : criteres d'en-tete rectifies, puis le
// filtre de parasite deja etabli par la phase 2 (une position parasite est un motif d'en-tete
// fortuit A L'INTERIEUR d'un vrai enregistrement, et elle se reconnait sans rien supposer de la
// grammaire : son champ de nom ne rend pas de texte imprimable).
func profilRosterTable(d []byte) []s3rTouche {
	fin := (dernierNonNul(d) + 1) * 8
	var out []s3rTouche
	for _, h := range profilRosterBalaye(d, profilRosterCritCorrigee()) {
		e := s3sDecode(d, h.bit-s3rEnteteBits, fin)
		if e == nil || !s3sImprimable(e.gamertag) {
			continue
		}
		out = append(out, h)
	}
	return out
}

// profilRosterFermeture est l'ORACLE INTERNE de la lecture, et il est celui de la phase 2 :
// LA GRAMMAIRE PREDIT L'ECART. Pour chaque enregistrement, `s3sPredite` donne sa longueur a
// partir de quatre nombres lus dans le flux ; l'ecart mesure jusqu'au suivant doit valoir cette
// longueur plus la CONSTANTE DU BUILD (0 sur `HI_1_13_0`, -2 880 / -4 320 / +1 600 ailleurs).
//
// Une seule constante doit donc couvrir TOUS les ecarts d'un film. Un enregistrement saute se
// voit immediatement : son ecart vaut la longueur predite PLUS celle du saute, donc pas la
// constante. Une position parasite aussi : son ecart est trop court. Ce critere ne depend
// d'AUCUNE source externe et il ne confond pas les deux classes de longueur, puisque la
// prediction est faite enregistrement par enregistrement.
//
// Rend le nombre d'ecarts conformes, le nombre d'ecarts NON conformes, et la constante modale.
func profilRosterFermeture(d []byte, hs []s3rTouche) (conformes, aberrants, constante int) {
	fin := (dernierNonNul(d) + 1) * 8
	var ecarts []int
	comptes := map[int]int{}
	for i := 0; i+1 < len(hs); i++ {
		e := s3sDecode(d, hs[i].bit-s3rEnteteBits, fin)
		if e == nil {
			continue
		}
		k := (hs[i+1].bit - hs[i].bit) - s3sPredite(e)
		ecarts = append(ecarts, k)
		comptes[k]++
	}
	best := -1
	for k, n := range comptes {
		if n > best || (n == best && k < constante) {
			constante, best = k, n
		}
	}
	for _, k := range ecarts {
		if k == constante {
			conformes++
		} else {
			aberrants++
		}
	}
	return conformes, aberrants, constante
}

// TestProfilRosterCorrige est la PREUVE. Deux criteres, tous deux ECRITS AVANT LA MESURE :
//
//	P1 FERMETURE DE LA GRAMMAIRE (oracle 100 % interne) : chaque ecart de la table lue doit
//	   valoir la longueur PREDITE de l'enregistrement plus UNE SEULE constante par film (celle
//	   du build). Un enregistrement saute ou une position parasite casse cette egalite. C'est
//	   le critere que le test fait echouer, parce qu'il ne depend d'AUCUNE source externe.
//	P2 CARDINAL : le compte lu, confronte au nombre d'entites ti=9 de la trame. Il est PUBLIE
//	   et non impose : la table de `chunk_00` est le roster a l'instant ou le chunk est ecrit,
//	   la trame compte les joueurs PRESENTS a l'image-cle — les deux peuvent legitimement
//	   differer (arrivee en cours de partie, depart). La ventilation est dans
//	   `TestProfilRosterBilan`.
func TestProfilRosterCorrige(t *testing.T) {
	bons, vus, gros, grosBons, defauts := 0, 0, 0, 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		att := profilRosterAttendu(dir)
		if att == 0 {
			continue
		}
		vus++
		hs := profilRosterTable(d)
		avant := len(s3rGrappe(s3rBalayage(d, (dernierNonNul(d)+1)*8)))
		conf, aberr, k := profilRosterFermeture(d, hs)
		if aberr > 0 {
			defauts++
			t.Errorf("%s : %d ecart(s) ne valent pas la longueur predite plus la constante %d",
				filepath.Base(dir), aberr, k)
		}
		if len(hs) == att {
			bons++
		}
		if att > 16 {
			gros++
			if len(hs) == att {
				grosBons++
			}
		}
		t.Logf("%-10s : entites ti=%d %2d | AVANT %2d | APRES %2d | P1 %d/%d ecart(s) fermes "+
			"a la constante %d | P2 %s",
			filepath.Base(dir), equipeTI, att, avant, len(hs), conf, conf+aberr, k,
			profilRosterVerdict(len(hs) == att))
	}
	t.Logf("=== CORRECTION === P1 : %d/%d films SANS trou ni parasite ; "+
		"P2 : %d/%d films au compte des entites ti=%d, dont %d/%d a plus de 16 joueurs",
		vus-defauts, vus, bons, vus, equipeTI, grosBons, gros)
}

// profilRosterVerdict rend le verdict d'une ligne.
func profilRosterVerdict(ok bool) string {
	if ok {
		return "EGAL"
	}
	return "DIFFERENT"
}

// TestProfilRosterCorpus est le controle d'echelle de la correction : le balayage corrige sur
// tout un repertoire de films. Il verifie la borne de l'ecrivain (32 au plus) et publie la
// distribution des comptes. Garde CHUNK00_CORPUS (+ CHUNK00_CORPUS_MAX, defaut 250).
func TestProfilRosterCorpus(t *testing.T) {
	racine := os.Getenv("CHUNK00_CORPUS")
	if racine == "" {
		t.Skip("CHUNK00_CORPUS absent : controle d'echelle saute")
	}
	max := 250
	if v, err := strconv.Atoi(os.Getenv("CHUNK00_CORPUS_MAX")); err == nil && v > 0 {
		max = v
	}
	entrees, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture de %s : %v", racine, err)
	}
	avant, apres := map[int]int{}, map[int]int{}
	lus, hors := 0, 0
	for _, e := range entrees {
		if !e.IsDir() || lus >= max {
			continue
		}
		d, err := os.ReadFile(filepath.Join(racine, e.Name(), "chunk_00.bin"))
		if err != nil || len(d) < s3wBoolOff {
			continue
		}
		lus++
		avant[len(s3rGrappe(s3rBalayage(d, (dernierNonNul(d)+1)*8)))]++
		n := len(profilRosterTable(d))
		apres[n]++
		if n > 32 {
			hors++
			t.Errorf("%s rend %d enregistrements, or l'ecrivain en impose 32 au plus", e.Name(), n)
		}
	}
	t.Logf("%d films lus", lus)
	t.Logf("  AVANT : %s", equipeHistoTexte(avant))
	t.Logf("  APRES : %s", equipeHistoTexte(apres))
	t.Logf("  au-dela de la borne de 32 : %d film(s)", hors)
}

// TestProfilRosterSlots publie, pour les films a gros roster, les SLOTS des enregistrements
// trouves et ceux des entites ti=9 de la trame. C'est la mesure qui dit si la reattribution de
// slot en cours de match se voit dans `chunk_00`.
func TestProfilRosterSlots(t *testing.T) {
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur || v.Card <= 16 {
			continue
		}
		_, d := readChunk00(t, dir)
		hs := profilRosterTable(d)
		var slots []int
		for _, r := range v.Paquets[0].Recs {
			slots = append(slots, r.Slot)
		}
		t.Logf("%-10s : %d enregistrements de slot, %d entites ti=%d aux slots %v",
			filepath.Base(dir), len(hs), v.Card, equipeTI, slots)
		t.Logf("%-10s   xuids de la table : %v", filepath.Base(dir), profilRosterXuids(hs))
	}
}

// profilRosterCritCorrigee rend les criteres RETENUS apres le diagnostic.
//
// LA CORRECTION TIENT EN UNE LIGNE, ET ELLE EST JUSTIFIEE PAR LA SONDE : le champ de 2 bits de
// `slot+0x08` N'EST PAS constant. `FUN_1407ecb08` l'ecrit comme un octet SIGNE sur 2 bits — son
// domaine est donc -2..1, pas {0} — et la sonde `TestProfilRosterXuidIntrouvable` l'a mesure a
// **1** sur deux enregistrements bien reels (`1c4c63c2` xuid 2535450607961405 au bit 11 137 668,
// `111fa685` xuid 2535454874175468 au bit 10 812 405), tous deux avec `b=1/0/0`, `u32=0` et un
// jeton non nul. Exiger ce champ nul rendait ces slots INVISIBLES, et leur absence doublait
// l'ecart au voisin, ce qui faisait perdre a `s3rGrappe` TOUTE LA TETE de la table
// (`1c4c63c2` : 11 enregistrements lus au lieu de 24).
//
// Le seuil de regroupement est INCHANGE : la mesure des ecarts montre qu'une fois le critere
// corrige, plus aucun ecart intra-table ne depasse 40 000 bits.
func profilRosterCritCorrigee() profilRosterCrit {
	c := profilRosterCritPlein()
	c.Deux = false
	return c
}
