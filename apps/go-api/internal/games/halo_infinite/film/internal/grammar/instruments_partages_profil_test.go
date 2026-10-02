package grammar

// instruments_partages_profil_test.go — les lecteurs de profil, de table de joueurs et
// d'image-cle des instruments (`profil*`, `rs*`, `imc*`, `chunk00Films`), dont se servent les
// gardes `player_table_corpus_test.go` et `default_state_n2_constant_test.go`.
// Deplaces tels quels au J12.7 bis depuis les fichiers tagues `research` (decision DU-5 : le
// tag cache les instruments, jamais une garde) ; chaque declaration garde le corps et le
// commentaire de son fichier d'origine.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// chunk00Films rend les repertoires de film de la garde d'environnement.
func chunk00Films(t *testing.T, envName string) []string {
	t.Helper()
	v := os.Getenv(envName)
	if v == "" {
		t.Skipf("%s absent : instrument saute", envName)
	}
	var out []string
	for p := range strings.SplitSeq(v, ";") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// imcCharger lit le registre et TOUS les payloads d'image-cle d'un film, plus sa chaine de
// build (lue dans l'en-tete de `chunk_00`, le meme champ que la table de profil de la
// phase 4).
func imcCharger(t *testing.T, dir string) (imcFilm, bool) {
	t.Helper()
	f := imcFilm{Nom: filepath.Base(dir)}
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Logf("%s : ECARTE (aucun chunk)", f.Nom)
		return f, false
	}
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Logf("%s : ECARTE (chunk_00 illisible : %v)", f.Nom, err)
		return f, false
	}
	if f.Reg, err = ParseRegistryChunk(raw); err != nil {
		t.Logf("%s : ECARTE (registre illisible : %v)", f.Nom, err)
		return f, false
	}
	if _, d := readChunk00(t, dir); len(d) > 0 {
		f.Build, _ = s3bBuild(d)
	}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type == PacketTypeKeyframe {
				f.Pays = append(f.Pays, pk.Payload(data))
			}
		}
	}
	return f, len(f.Pays) > 0
}

// profilLireEtatComplet decompose un record a l'ancre `anchor` selon FUN_142e2bfd0.
// `ds` est joue par le deserialiseur d'etat par defaut PORTE du depot (default_state_arch.go),
// donc aucune largeur n'est inventee ici.
func profilLireEtatComplet(pay []byte, anchor, ti int) profilEtatComplet {
	e := profilEtatComplet{TI: ti}
	p := anchor + profile.KeyframeEnTeteBits
	e.N1 = source.BitsBourres(pay, p, profile.KeyframeMotDeTailleBits)
	p += profile.KeyframeMotDeTailleBits
	if e.N1 > 0 { // FUN_142e2bfd0 : `if (0 < (int)uVar7)` — sans taille, pas d'etat par defaut
		br := LecteurSur(pay)
		br.SetBitPos(p)
		consumeKeyframeDefaultState(br, uint32(ti))
		e.DSBits = br.BitPos() - p
		p = br.BitPos()
	}
	e.N2 = source.BitsBourres(pay, p, profile.KeyframeMotDeTailleBits)
	p += profile.KeyframeMotDeTailleBits
	e.CorpsRelatif = p - anchor
	return e
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

// rsDelta rend la transposition MODALE d'un film : la valeur de `mesure - predit` qui couvre le
// plus d'ecarts. C'est la calibration du lecteur corrige, et elle se fait SUR LE FILM, sans
// table de build ecrite a la main — donc elle vaut aussi pour un build inconnu.
func rsDelta(d []byte) (delta, couverts, total int) {
	es, ecarts := rsEcarts(d)
	comptes := map[int]int{}
	for i, e := range es {
		if ecarts[i] == 0 {
			continue
		}
		comptes[ecarts[i]-s3sPredite(e)]++
		total++
	}
	best := -1
	for k, n := range comptes {
		if n > best || (n == best && k < delta) {
			delta, best = k, n
		}
	}
	if best < 0 {
		return 0, 0, 0
	}
	return delta, best, total
}

// rsChaine est le LECTEUR CANONIQUE CORRIGE : `s3sChaine`, mais le pas predit est corrige de la
// constante du build, CALIBREE SUR LE FILM par `rsDelta`. Le depart est le premier
// enregistrement a nom imprimable du balayage CORRIGE (critere `slot+0x08` leve, phase 4).
func rsChaine(d []byte, delta int) (enrs []*s3sEnr, vacants int) {
	fin := (dernierNonNul(d) + 1) * 8
	hs := profilRosterBalaye(d, profilRosterCritCorrigee())
	deb := -1
	for _, h := range hs {
		e := s3sDecode(d, h.bit-s3rEnteteBits, fin)
		if e != nil && s3sImprimable(e.gamertag) {
			deb = e.debut
			break
		}
	}
	if deb < 0 {
		return nil, 0
	}
	for p, lus := deb, 0; lus < 32; lus++ {
		if rsVacant(d, p) {
			// Slot VACANT : le balayage ne peut pas le voir, mais la grammaire en connait la
			// longueur EXACTE. On l'enjambe sans le compter comme un joueur — c'est la cause
			// des deux ecarts aberrants de R1.
			vacants++
			p += rsVide(delta)
			continue
		}
		e := s3sDecode(d, p, fin)
		if e == nil || !s3sImprimable(e.gamertag) {
			break
		}
		enrs = append(enrs, e)
		p += s3sPredite(e) + delta
	}
	return enrs, vacants
}

// dernierNonNul rend l'offset du dernier octet non nul, ou -1.
// DELEGUE DEPUIS LE LOT 1.5 : la meme lecture est devenue une borne de PRODUCTION
// (`dernierOctetNonNul`, film_identity.go). Deux copies auraient pu diverger — celle des
// instruments est l'oracle de celle du lecteur, donc elles doivent etre la MEME.
func dernierNonNul(data []byte) int { return dernierOctetNonNul(data) }

// imcFilm porte ce qu'un film offre a la mesure.
type imcFilm struct {
	Nom, Build string
	Reg        *Registry
	Pays       [][]byte
}

// profilEtatComplet est la decomposition d'un record par le lecteur d'ETAT COMPLET.
type profilEtatComplet struct {
	TI           int
	N1, N2       uint64
	DSBits       int
	CorpsRelatif int // position du premier composant, relative a l'ancre
}

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

// les slots vacants (R1).
func rsEcarts(d []byte) ([]*s3sEnr, []int) {
	fin := (dernierNonNul(d) + 1) * 8
	var es []*s3sEnr
	for _, h := range profilRosterTable(d) {
		if e := s3sDecode(d, h.bit-s3rEnteteBits, fin); e != nil && s3sImprimable(e.gamertag) {
			es = append(es, e)
		}
	}
	ecarts := make([]int, len(es))
	for i := range es {
		if i+1 < len(es) {
			ecarts[i] = es[i+1].debut - es[i].debut
		}
	}
	return es, ecarts
}

func rsVide(delta int) int { return rsVideHorsBlocs + s3sPersoBits + delta + s3sBloc44 + 32 }

// rsVacant dit si un enregistrement de slot VACANT commence au bit `p`. Le predicat est
// GRAMMATICAL et sans seuil : chaque champ est lu a sa place et doit valoir zero. Un tel
// enregistrement est doublement INVISIBLE au balayage — son booleen de tete vaut 0 (le balayage
// exige `1/0/0`) et son XUID tombe hors de la plage Xbox — et c'est la raison mecanique pour
// laquelle il DOUBLE l'ecart entre ses deux voisins.
//
// Les champs laisses libres sont ceux dont la mesure montre qu'ils ne sont PAS nuls dans un
// enregistrement vacant : `sub+0xcb8` (64 bits) et le champ de 6 bits `sub+0xc35`.
func rsVacant(d []byte, p int) bool {
	if s3rBit(d, p, 3) != 0 || s3rBit(d, p+3, 32) != 0 || s3rBit(d, p+35, 2) != 0 {
		return false
	}
	if s3rBit(d, p+37, 48) != 0 || s3rBit(d, p+s3rEnteteBits, 64) != 0 {
		return false
	}
	q := p + s3rEnteteBits + 64
	if s3rBit(d, q, s3sPrefixeMasque+1) != 0 { // prefixe nul, donc un seul bit de masque, nul
		return false
	}
	q += s3sPrefixeMasque + 1
	if s3rBit(d, q, s3sLargeurN) != 0 || s3rBit(d, q+s3sLargeurN, s3sLargeurM) != 0 {
		return false
	}
	q += s3sLargeurN + s3sLargeurM
	for k := 0; k < s3sBloc104; k += 64 { // les 104 octets de sub+0xc48
		if s3rBit(d, q+k, 64) != 0 {
			return false
		}
	}
	q += s3sBloc104
	if s3rBit(d, q, 16) != 0 { // la chaine reduite a son terminateur
		return false
	}
	q += 16
	for k := 0; k < s3sBloc16+32; k += 32 { // sub+0xc38 puis `desired-representation`
		if s3rBit(d, q+k, 32) != 0 {
			return false
		}
	}
	return true
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

// rsVide rend la longueur, en bits, d'un enregistrement de slot ENTIEREMENT A ZERO sur un build
// dont la transposition vaut `delta`. Elle est CALCULEE terme a terme depuis la grammaire, pas
// ajustee sur une mesure :
//
//	  85  en-tete du slot (1+1+1+32+2+48)        FUN_1407ecb08
//	+ 64  le XUID, nul                           FUN_1406d6498
//	+ 11  prefixe du masque (rang du bit haut)    FUN_1424ccf94
//	+  1  le masque reduit a un bit               masque vide -> rang 0 -> 1 bit ecrit
//	+ 12  N, nul                                  FUN_1411b1a24
//	+  8  M, nul                                  FUN_1411b198c
//	+ 832 le bloc de 104 octets                   sub+0xc48
//	+ 16  la chaine reduite a son NUL             FUN_1407ece18 s'arrete APRES l'unite nulle
//	+ 128 le bloc de 16 octets                    sub+0xc38
//	+ 32  `desired-representation`                sub+0xcb0
//	+ 64  le champ de 64 bits                     sub+0xcb8
//	+ 46  les six champs courts (10+14+6+8+7+1)
//	= 1 299 bits, puis le bloc de personnalisation, le bloc de queue et le u32 final.
const rsVideHorsBlocs = 1299
