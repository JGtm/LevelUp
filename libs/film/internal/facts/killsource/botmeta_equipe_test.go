package killsource

// botmeta_equipe_test.go — L EQUIPE D UN BOT, LUE DANS SON ENTREE BOT_METADATA (botmeta_equipe.go).
//
// Les paquets sont FABRIQUES PAR LA GRAMMAIRE DE L ECRIVAIN (`FUN_14299bda0`, corps
// `FUN_1407edea8`), avec des LARGEURS ET DES POSITIONS LITTERALES : un test qui relirait les
// constantes qu il verifie ne verifierait rien. Une entree au nom de 13 unites fait 16 622 bits sur
// HI_1_13_0 : la seconde entree d un paquet commence six bits apres une frontiere d octet, comme sur
// le parc. Deux dispositions du bloc de 44 octets : celle de HI_1_13_0 (le nom, puis l equipe a 0x25)
// et celle de HI_1_10_0 a HI_1_12_0 (l equipe a l octet 4, puis le nom a l octet 12).
//
//	EQ-UN          un bot, equipe 1 : lue, jumeau egal, rien d autre ne se compte.
//	EQ-DEUX        deux bots dans un paquet, equipes 0 et 1 : la seconde entree, decalee, se lit.
//	EQ-ANCIEN      disposition HI_1_12_0, equipes 1 et 0 : lues a l octet 4 ; les fragments `43 X`
//	               que la copie du nom fait lire au balayage sont ecartes, le paquet est complet.
//	EQ-ANCIEN-0x25 la meme disposition lue a l octet 0x25 donne 0 aux deux bots : c est le defaut que
//	               la position par build corrige.
//	EQ-POSITION    un build sans position de l equipe au profil : les entrees se lisent, aucune
//	               equipe, chaque bot se compte illisible — jamais 0 par defaut.
//	EQ-FERMETURE   une largeur de personnalisation fausse de 8 bits, dans les deux sens : le paquet
//	               ne ferme pas, aucune equipe, le bot se compte illisible.
//	EQ-PERSO       un build sans largeur au profil : aucun paquet ne se lit.
//	EQ-CONTRADICTION deux paquets fermes, deux equipes : aucune equipe, le bot se compte.
//	EQ-JUMEAU      jumeau different de l equipe : l equipe fait foi, l entree se compte.
//	EQ-AUCUNE      equipe -1 (« aucune », mode sans camps) : une LECTURE, publiee -1 — l octet est signe.
//	EQ-DOMAINE     equipe hors de -1..8 : refusee, le bot se compte illisible.
//	EQ-FANTOME     le dernier octet de personnalisation non nul fait lire au balayage historique un
//	               fragment de la copie petit-boutiste du nom (`43 KaleDucky`, slot 0, bid 0 — le cas
//	               de `8076f97f`) : le vrai bot est lu, le fragment est ecarte et compte.
//	EQ-HORS-BALAYAGE un nom de trois unites, que le balayage ne lit pas : l entree se compte.
//	EQ-HORS-GRAMMAIRE un nom que le balayage lit dans le bloc de personnalisation, qu aucune entree
//	               n ecrit et qui n est pas un fragment : retire du roster, compte.
//	EQ-FRAGMENT-NON-FERME une largeur fausse de 8 bits : le paquet ne ferme pas, le fragment `43 X`
//	               reste au balayage, aucun n est compte ecarte.
//	EQ-ENTREE-SANS-EQUIPE des entrees lues sans position de l equipe ne donnent jamais d equipe,
//	               meme a un releve qui croit la position connue : bots illisibles.
//	EQ-DIAGNOSTICS les comptes non nuls se disent : ERREUR pour un vrai bot sans equipe,
//	               AVERTISSEMENT pour un paquet qui ne ferme pas (meme quand chaque bot tient son
//	               equipe d un autre paquet), une entree refusee, hors balayage ou hors grammaire ;
//	               un fragment ecarte ne se dit pas.

import (
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/constat"
	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
)

// persoBitsHI113 : le bloc de personnalisation de HI_1_13_0 (et de HI_1_12_0), 1 852 octets.
const persoBitsHI113 = 1852 * 8

// lectureHI113 / lectureHI112 : les lectures des deux dispositions du bloc de 44 octets.
var (
	lectureHI113 = grammar.LectureDesBots{PersoBits: persoBitsHI113, PersoConnue: true,
		Equipe: profile.PositionEquipeBot{Equipe: 0x25, Jumeau: 0x26}, EquipeConnue: true}
	lectureHI112 = grammar.LectureDesBots{PersoBits: persoBitsHI113, PersoConnue: true,
		Equipe: profile.PositionEquipeBot{Equipe: 4, Jumeau: 6}, EquipeConnue: true}
)

// lectureHI113Perso : la lecture de HI_1_13_0 sous une autre largeur de personnalisation.
func lectureHI113Perso(persoBits int) grammar.LectureDesBots {
	l := lectureHI113
	l.PersoBits = persoBits
	return l
}

// ecrivainDeBits : un ecrivain de bits, poids fort d abord, comme celui du jeu.
type ecrivainDeBits struct {
	b []byte
	n int
}

func (w *ecrivainDeBits) ecrire(v uint64, bits int) {
	for i := bits - 1; i >= 0; i-- {
		if w.n/8 >= len(w.b) {
			w.b = append(w.b, 0)
		}
		if v>>uint(i)&1 == 1 {
			w.b[w.n/8] |= 0x80 >> uint(w.n%8)
		}
		w.n++
	}
}

func (w *ecrivainDeBits) zeros(bits int) {
	for bits > 0 {
		k := min(bits, 64)
		w.ecrire(0, k)
		bits -= k
	}
}

// botEcrit : un bot du paquet fabrique et ce que son entree porte.
type botEcrit struct {
	slot, bid      int
	nom            string
	equipe, jumeau int
	// octetAvantLeBloc : le dernier octet du bloc de personnalisation. Nul sur le parc HI_1_13_0 ;
	// non nul, il fait lire au balayage un fragment de la copie du nom (EQ-FANTOME).
	octetAvantLeBloc byte
	// nomCache : un nom UTF-16BE ferme par l unite nulle, ecrit en tete du bloc de personnalisation —
	// le balayage le lit (slot 0, `bid` 0), la marche le saute (EQ-HORS-GRAMMAIRE).
	nomCache string
}

// blocHI113 : le bloc de 44 octets de HI_1_13_0 — le nom en UTF-16 petit-boutiste, puis l equipe a
// 0x25 et le jumeau a 0x26.
func blocHI113(b botEcrit) []byte {
	bloc := make([]byte, 44)
	for i, c := range b.nom {
		bloc[2*i] = byte(c)
	}
	bloc[0x25], bloc[0x26] = byte(int8(b.equipe)), byte(int8(b.jumeau)) //nolint:gosec // octets signes
	return bloc
}

// blocHI112 : le bloc de 44 octets de HI_1_10_0 a HI_1_12_0, tel que le parc l ecrit —
// `02 00 00 00 | equipe 01 jumeau 07 | a0 bd 76 18 |` puis le nom en UTF-16 petit-boutiste.
func blocHI112(b botEcrit) []byte {
	bloc := make([]byte, 44)
	bloc[0], bloc[5], bloc[7] = 2, 1, 7
	bloc[4], bloc[6] = byte(int8(b.equipe)), byte(int8(b.jumeau)) //nolint:gosec // octets signes
	copy(bloc[8:], []byte{0xa0, 0xbd, 0x76, 0x18})
	for i, c := range b.nom {
		bloc[12+2*i] = byte(c)
	}
	return bloc
}

// payloadEcrit fabrique un payload de type 12 par la grammaire de l ecrivain, au bloc de
// personnalisation de `persoBits` bits, bloc de 44 octets de HI_1_13_0.
func payloadEcrit(persoBits int, bots ...botEcrit) []byte {
	return payloadDispose(persoBits, blocHI113, bots...)
}

// payloadDispose fabrique un payload de type 12 dont le bloc de 44 octets suit la disposition `bloc`.
func payloadDispose(persoBits int, bloc func(botEcrit) []byte, bots ...botEcrit) []byte {
	w := &ecrivainDeBits{}
	w.ecrire(uint64(len(bots)), 32)
	for k, b := range bots {
		w.ecrire(uint64(k), 32) // index absolu
		w.ecrire(uint64(b.slot), 32)
		w.ecrire(uint64(b.bid), 32)
		w.ecrire(0, 11) // rang du bit haut du masque : un masque d un bit
		w.ecrire(0, 1)
		w.ecrire(0, 12) // N = 0
		w.ecrire(0, 8)  // M = 0
		w.zeros(832)    // 104 octets
		for _, c := range b.nom {
			w.ecrire(uint64(c), 16)
		}
		w.ecrire(0, 16) // l unite nulle (nom de moins de 16 unites)
		w.zeros(128 + 32 + 64 + 46)
		for _, c := range b.nomCache {
			w.ecrire(uint64(c), 16)
		}
		if b.nomCache != "" {
			w.ecrire(0, 16)
		}
		w.zeros(persoBits - 8 - 16*len(b.nomCache) - 16*min(len(b.nomCache), 1))
		w.ecrire(uint64(b.octetAvantLeBloc), 8)
		for _, o := range bloc(b) {
			w.ecrire(uint64(o), 8)
		}
	}
	return w.b
}

// paquetsEcrits rend les paquets BOT_METADATA d un film synthetique, un payload par chunk, tels que la
// grammaire les lit sous `lecture`.
func paquetsEcrits(t *testing.T, lecture grammar.LectureDesBots, payloads ...[]byte) []grammar.PaquetBotMetadata {
	t.Helper()
	paquets := make([]paquetDeTest, len(payloads))
	for i, pl := range payloads {
		paquets[i] = paquetDeTest{chunk: i, typ: grammar.PacketTypeBotMetadata, ts: uint64(1_000 * (i + 1)), payload: pl} //nolint:gosec // rang de test
	}
	return grammar.PaquetsBotMetadata(sourceDePaquets(t, paquets...), lecture)
}

// lireLesEquipes charge les paquets comme le decodeur et y lit l equipe des bots.
func lireLesEquipes(t *testing.T, lecture grammar.LectureDesBots, payloads ...[]byte) botMeta {
	t.Helper()
	paquets := paquetsEcrits(t, lecture, payloads...)
	m := loadBotMeta(paquets)
	poserLesEquipesDesBots(paquets, &m, lecture)
	return m
}

// equipeDe rend l equipe publiee du bot `nom`, et s il existe.
func equipeDe(t *testing.T, m botMeta, nom string) *int {
	t.Helper()
	for _, b := range m.Bots {
		if b.Name == nom {
			return b.entree().Team
		}
	}
	t.Fatalf("bot %q absent de %+v", nom, m.Bots)
	return nil
}

func exigerEquipe(t *testing.T, m botMeta, nom string, attendue int) {
	t.Helper()
	if eq := equipeDe(t, m, nom); eq == nil || *eq != attendue {
		t.Fatalf("%s : equipe %v, attendu %d (bilan %+v)", nom, eq, attendue, m.Equipes)
	}
}

func exigerSansEquipe(t *testing.T, m botMeta, nom string) {
	t.Helper()
	if eq := equipeDe(t, m, nom); eq != nil {
		t.Fatalf("%s : equipe %d publiee, attendu aucune (bilan %+v)", nom, *eq, m.Equipes)
	}
}

func exigerBilan(t *testing.T, m botMeta, attendu EquipesDesBots) {
	t.Helper()
	if m.Equipes != attendu {
		t.Fatalf("bilan %+v, attendu %+v", m.Equipes, attendu)
	}
}

var sandwolf = botEcrit{slot: 8, bid: 44, nom: "343 Sandwolf", equipe: 1, jumeau: 1}

func TestEquipeDUnBot(t *testing.T) { // EQ-UN
	m := lireLesEquipes(t, lectureHI113, payloadEcrit(persoBitsHI113, sandwolf))
	exigerEquipe(t, m, "343 Sandwolf", 1)
	exigerBilan(t, m, EquipesDesBots{Lues: 1})
}

func TestEquipesDeDeuxBotsDecales(t *testing.T) { // EQ-DEUX
	thumb := botEcrit{slot: 8, bid: 57, nom: "343 The Thumb", equipe: 0, jumeau: 0}
	donos := botEcrit{slot: 9, bid: 6, nom: "343 Donos", equipe: 1, jumeau: 1}
	pl := payloadEcrit(persoBitsHI113, thumb, donos)
	if ps := paquetsEcrits(t, lectureHI113, pl); len(ps) != 1 || !ps[0].Ferme || len(ps[0].Ecrites) != 2 {
		t.Fatalf("marche : %+v", ps)
	}
	m := lireLesEquipes(t, lectureHI113, pl)
	exigerEquipe(t, m, "343 The Thumb", 0)
	exigerEquipe(t, m, "343 Donos", 1)
	exigerBilan(t, m, EquipesDesBots{Lues: 2})
}

func TestUneLargeurFausseNeFermePas(t *testing.T) { // EQ-FERMETURE
	for _, ecart := range []int{-8, 8} {
		m := lireLesEquipes(t, lectureHI113Perso(persoBitsHI113+ecart), payloadEcrit(persoBitsHI113, sandwolf))
		exigerSansEquipe(t, m, "343 Sandwolf")
		exigerBilan(t, m, EquipesDesBots{Illisibles: 1, PaquetsNonFermes: 1})
	}
}

func TestBuildSansLargeurDePersonnalisation(t *testing.T) { // EQ-PERSO
	m := lireLesEquipes(t, grammar.LectureDesBots{}, payloadEcrit(persoBitsHI113, sandwolf))
	exigerSansEquipe(t, m, "343 Sandwolf")
	exigerBilan(t, m, EquipesDesBots{Illisibles: 1, PersoInconnue: true})
}

func TestDeuxPaquetsDeuxEquipes(t *testing.T) { // EQ-CONTRADICTION
	autre := sandwolf
	autre.equipe, autre.jumeau = 0, 0
	m := lireLesEquipes(t, lectureHI113, payloadEcrit(persoBitsHI113, sandwolf),
		payloadEcrit(persoBitsHI113, autre))
	exigerSansEquipe(t, m, "343 Sandwolf")
	exigerBilan(t, m, EquipesDesBots{Contradictoires: 1})
}

func TestJumeauDiscordant(t *testing.T) { // EQ-JUMEAU
	b := sandwolf
	b.jumeau = -1
	m := lireLesEquipes(t, lectureHI113, payloadEcrit(persoBitsHI113, b))
	exigerEquipe(t, m, "343 Sandwolf", 1)
	exigerBilan(t, m, EquipesDesBots{Lues: 1, JumeauxDiscordants: 1})
}

func TestAucuneEquipeSePublieAMoinsUn(t *testing.T) { // EQ-AUCUNE
	b := sandwolf
	b.equipe, b.jumeau = -1, -1
	m := lireLesEquipes(t, lectureHI113, payloadEcrit(persoBitsHI113, b))
	exigerEquipe(t, m, "343 Sandwolf", -1)
	exigerBilan(t, m, EquipesDesBots{Lues: 1})
}

func TestEquipeHorsDomaine(t *testing.T) { // EQ-DOMAINE
	b := sandwolf
	b.equipe, b.jumeau = 9, 9
	m := lireLesEquipes(t, lectureHI113, payloadEcrit(persoBitsHI113, b))
	exigerSansEquipe(t, m, "343 Sandwolf")
	exigerBilan(t, m, EquipesDesBots{Illisibles: 1, HorsDomaine: 1})
}

func TestFragmentDuBalayageEcarte(t *testing.T) { // EQ-FANTOME
	kale := botEcrit{slot: 8, bid: 48, nom: "343 KaleDucky", equipe: 0, jumeau: 0, octetAvantLeBloc: 0xFF}
	m := lireLesEquipes(t, lectureHI113, payloadEcrit(persoBitsHI113, kale))
	if len(m.Bots) != 1 || m.Incomplets != 0 {
		t.Fatalf("le fragment devait etre ecarte : %d bot(s) %+v, %d incomplet(s)", len(m.Bots), m.Bots, m.Incomplets)
	}
	exigerEquipe(t, m, "343 KaleDucky", 0)
	exigerBilan(t, m, EquipesDesBots{Lues: 1, FragmentsEcartes: 1})
}

// hollis / ritzy : deux bots de `4d78ae14` (HI_1_11_0), equipes de la feuille 1 et 0. `343 Ritzy`
// porte le `bid` 0 : un vrai bot de `bid` 0 n est pas un fragment.
var (
	hollis = botEcrit{slot: 24, bid: 59, nom: "343 Hollis", equipe: 1, jumeau: 1}
	ritzy  = botEcrit{slot: 25, bid: 0, nom: "343 Ritzy", equipe: 0, jumeau: 0}
)

func TestEquipeDeLaDispositionAncienne(t *testing.T) { // EQ-ANCIEN
	pl := payloadDispose(persoBitsHI113, blocHI112, hollis, ritzy)
	m := lireLesEquipes(t, lectureHI112, pl)
	if len(m.Bots) != 2 || m.Incomplets != 0 {
		t.Fatalf("les fragments `43 X` devaient etre ecartes : %d bot(s) %+v, %d incomplet(s)", len(m.Bots), m.Bots,
			m.Incomplets)
	}
	exigerEquipe(t, m, "343 Hollis", 1)
	exigerEquipe(t, m, "343 Ritzy", 0)
	exigerBilan(t, m, EquipesDesBots{Lues: 2, FragmentsEcartes: 2})
}

func TestDispositionAncienneLueA0x25(t *testing.T) { // EQ-ANCIEN-0x25
	m := lireLesEquipes(t, lectureHI113, payloadDispose(persoBitsHI113, blocHI112, hollis))
	exigerEquipe(t, m, "343 Hollis", 0) // l octet 0x25 tombe dans la copie du nom
}

func TestBuildSansPositionDeLEquipe(t *testing.T) { // EQ-POSITION
	sansPosition := grammar.LectureDesBots{PersoBits: persoBitsHI113, PersoConnue: true}
	m := lireLesEquipes(t, sansPosition, payloadDispose(persoBitsHI113, blocHI112, hollis, ritzy))
	exigerSansEquipe(t, m, "343 Hollis")
	exigerSansEquipe(t, m, "343 Ritzy")
	exigerBilan(t, m, EquipesDesBots{Illisibles: 2, FragmentsEcartes: 2, PositionInconnue: true})
}

func TestEntreeQueLeBalayageNeLitPas(t *testing.T) { // EQ-HORS-BALAYAGE
	court := botEcrit{slot: 8, bid: 3, nom: "Bob", equipe: 1, jumeau: 1}
	m := lireLesEquipes(t, lectureHI113, payloadEcrit(persoBitsHI113, court))
	if len(m.Bots) != 0 {
		t.Fatalf("le balayage ne lit pas un nom de trois unites : %+v", m.Bots)
	}
	exigerBilan(t, m, EquipesDesBots{EntreesHorsBalayage: 1})
}

func TestLesComptesSeDisent(t *testing.T) { // EQ-DIAGNOSTICS
	cas := []struct {
		nom    string
		bilan  EquipesDesBots
		codes  []constat.Code
		niveau []constat.Niveau
	}{
		{"tout lu", EquipesDesBots{Lues: 3}, nil, nil},
		{"illisible", EquipesDesBots{Illisibles: 1, PaquetsNonFermes: 2},
			[]constat.Code{DiagEquipesDeBots, DiagEntreesDeBots}, []constat.Niveau{constat.NiveauError, constat.NiveauWarn}},
		{"paquet non ferme, equipes lues ailleurs", EquipesDesBots{Lues: 2, PaquetsNonFermes: 1},
			[]constat.Code{DiagEntreesDeBots}, []constat.Niveau{constat.NiveauWarn}},
		{"hors grammaire", EquipesDesBots{Lues: 1, HorsGrammaire: 1},
			[]constat.Code{DiagEntreesDeBots}, []constat.Niveau{constat.NiveauWarn}},
		{"fragment ecarte", EquipesDesBots{Lues: 1, FragmentsEcartes: 3}, nil, nil},
		{"position inconnue", EquipesDesBots{Illisibles: 2, PositionInconnue: true},
			[]constat.Code{DiagEquipesDeBots}, []constat.Niveau{constat.NiveauError}},
		{"contradiction et jumeau", EquipesDesBots{Contradictoires: 1, JumeauxDiscordants: 1},
			[]constat.Code{DiagEquipesDeBots, DiagEntreesDeBots}, []constat.Niveau{constat.NiveauError, constat.NiveauWarn}},
		{"hors balayage", EquipesDesBots{EntreesHorsBalayage: 1},
			[]constat.Code{DiagEntreesDeBots}, []constat.Niveau{constat.NiveauWarn}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			ctx := &decodeCtx{name: "film"}
			ctx.signalerLesEquipesDesBots(c.bilan, "HI_1_13_0")
			ds := ctx.diag.Relever()
			if len(ds) != len(c.codes) {
				t.Fatalf("%d diagnostic(s), attendu %d : %+v", len(ds), len(c.codes), ds)
			}
			for i, d := range ds {
				if d.Code != c.codes[i] || d.Niveau != c.niveau[i] {
					t.Fatalf("diagnostic %d : %s / %d, attendu %s / %d", i, d.Code, d.Niveau, c.codes[i], c.niveau[i])
				}
			}
		})
	}
}

func TestNomHorsGrammaireRetireDuRoster(t *testing.T) { // EQ-HORS-GRAMMAIRE
	h := hollis
	h.nomCache = "Zorglub"
	pl := payloadDispose(persoBitsHI113, blocHI112, h)
	m := lireLesEquipes(t, lectureHI112, pl)
	if len(m.Bots) != 1 || m.Bots[0].Name != "343 Hollis" {
		t.Fatalf("le nom hors grammaire devait sortir du roster : %+v", m.Bots)
	}
	exigerEquipe(t, m, "343 Hollis", 1)
	exigerBilan(t, m, EquipesDesBots{Lues: 1, HorsGrammaire: 1, FragmentsEcartes: 1})
}

func TestUneLargeurFausseNEcartePasLesFragments(t *testing.T) { // EQ-FRAGMENT-NON-FERME
	pl := payloadDispose(persoBitsHI113, blocHI112, hollis)
	for _, ecart := range []int{-8, 8} {
		l := lectureHI112
		l.PersoBits += ecart
		ps := paquetsEcrits(t, l, pl)
		if len(ps) != 1 || ps[0].Ferme || ps[0].FragmentsEcartes != 0 {
			t.Fatalf("ecart %d : %+v ; attendu un paquet non ferme, aucun fragment ecarte", ecart, ps)
		}
		present := false
		for _, b := range ps[0].Balayees {
			present = present || b.Nom == "43 Hollis"
		}
		if !present {
			t.Fatalf("ecart %d : le fragment `43 Hollis` devait rester au balayage : %+v", ecart, ps[0].Balayees)
		}
	}
}

func TestUneEntreeSansEquipeLueEstRefusee(t *testing.T) { // EQ-ENTREE-SANS-EQUIPE
	// Les paquets sont lus sans position de l equipe, le releve croit la position connue : aucune
	// entree ne doit donner l equipe 0 de son champ nul.
	sansPosition := grammar.LectureDesBots{PersoBits: persoBitsHI113, PersoConnue: true}
	paquets := paquetsEcrits(t, sansPosition, payloadDispose(persoBitsHI113, blocHI112, hollis, ritzy))
	m := loadBotMeta(paquets)
	poserLesEquipesDesBots(paquets, &m, lectureHI112)
	exigerSansEquipe(t, m, "343 Hollis")
	exigerSansEquipe(t, m, "343 Ritzy")
	exigerBilan(t, m, EquipesDesBots{Illisibles: 2, FragmentsEcartes: 2})
}
