package killsource

// botmeta_equipe_test.go — L EQUIPE D UN BOT, LUE DANS SON ENTREE BOT_METADATA (botmeta_equipe.go).
//
// Les paquets sont FABRIQUES PAR LA GRAMMAIRE DE L ECRIVAIN (`FUN_14299bda0`, corps
// `FUN_1407edea8`), avec des LARGEURS LITTERALES : un test qui relirait les constantes qu il verifie
// ne verifierait rien. Une entree au nom de 13 unites fait 16 622 bits sur HI_1_13_0 : la seconde
// entree d un paquet commence six bits apres une frontiere d octet, comme sur le parc.
//
//	EQ-UN          un bot, equipe 1 : lue, jumeau egal, rien d autre ne se compte.
//	EQ-DEUX        deux bots dans un paquet, equipes 0 et 1 : la seconde entree, decalee, se lit.
//	EQ-FERMETURE   une largeur de personnalisation fausse de 8 bits, dans les deux sens : le paquet
//	               ne ferme pas, aucune equipe, le bot se compte illisible.
//	EQ-PERSO       un build sans largeur au profil : aucun paquet ne se lit.
//	EQ-CONTRADICTION deux paquets fermes, deux equipes : aucune equipe, le bot se compte.
//	EQ-JUMEAU      jumeau different de l equipe : l equipe fait foi, l entree se compte.
//	EQ-AUCUNE      equipe -1 (« aucune », mode sans camps) : une LECTURE, publiee -1 — l octet est signe.
//	EQ-DOMAINE     equipe hors de -1..8 : refusee, le bot se compte illisible.
//	EQ-FANTOME     le dernier octet de personnalisation non nul fait lire au balayage historique un
//	               fragment de la copie petit-boutiste du nom (`43 KaleDucky`, slot 0, bid 0 — le cas
//	               de `8076f97f`) : le vrai bot est lu, le fragment se compte hors grammaire.
//	EQ-HORS-BALAYAGE un nom de trois unites, que le balayage ne lit pas : l entree se compte.
//	EQ-DIAGNOSTICS les comptes non nuls se disent : ERREUR pour un bot sans equipe, AVERTISSEMENT
//	               pour un paquet qui ne ferme pas (meme quand chaque bot tient son equipe d un autre
//	               paquet), une entree refusee ou hors balayage.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
)

// persoBitsHI113 : le bloc de personnalisation de HI_1_13_0, 1 852 octets.
const persoBitsHI113 = 1852 * 8

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
}

// payloadEcrit fabrique un payload de type 12 par la grammaire de l ecrivain, au bloc de
// personnalisation de `persoBits` bits.
func payloadEcrit(persoBits int, bots ...botEcrit) []byte {
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
		w.zeros(128 + 32 + 64 + 46 + persoBits - 8)
		w.ecrire(uint64(b.octetAvantLeBloc), 8)
		bloc := make([]byte, 44) // le nom en UTF-16 petit-boutiste, puis l equipe a 0x25, le jumeau a 0x26
		for i, c := range b.nom {
			bloc[2*i] = byte(c)
		}
		bloc[0x25], bloc[0x26] = byte(int8(b.equipe)), byte(int8(b.jumeau)) //nolint:gosec // octets signes
		for _, o := range bloc {
			w.ecrire(uint64(o), 8)
		}
	}
	return w.b
}

// lireLesEquipes charge les paquets comme le decodeur et y lit l equipe des bots.
func lireLesEquipes(persoBits int, persoConnue bool, payloads ...[]byte) botMeta {
	paquets := make([]packet, len(payloads))
	for i, pl := range payloads {
		paquets[i] = packet{chunk: i, ts: uint64(1_000 * (i + 1)), payload: pl}
	}
	f := filmDePaquetsBotMeta(paquets...)
	m := loadBotMeta(f)
	poserLesEquipesDesBots(f, &m, persoBits/8, persoConnue)
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
	m := lireLesEquipes(persoBitsHI113, true, payloadEcrit(persoBitsHI113, sandwolf))
	exigerEquipe(t, m, "343 Sandwolf", 1)
	exigerBilan(t, m, EquipesDesBots{Lues: 1})
}

func TestEquipesDeDeuxBotsDecales(t *testing.T) { // EQ-DEUX
	thumb := botEcrit{slot: 8, bid: 57, nom: "343 The Thumb", equipe: 0, jumeau: 0}
	donos := botEcrit{slot: 9, bid: 6, nom: "343 Donos", equipe: 1, jumeau: 1}
	pl := payloadEcrit(persoBitsHI113, thumb, donos)
	if entrees, ferme := marcherLePaquet(pl, persoBitsHI113); !ferme || len(entrees) != 2 {
		t.Fatalf("marche : %d entree(s), fermee %v", len(entrees), ferme)
	}
	m := lireLesEquipes(persoBitsHI113, true, pl)
	exigerEquipe(t, m, "343 The Thumb", 0)
	exigerEquipe(t, m, "343 Donos", 1)
	exigerBilan(t, m, EquipesDesBots{Lues: 2})
}

func TestUneLargeurFausseNeFermePas(t *testing.T) { // EQ-FERMETURE
	for _, ecart := range []int{-8, 8} {
		m := lireLesEquipes(persoBitsHI113+ecart, true, payloadEcrit(persoBitsHI113, sandwolf))
		exigerSansEquipe(t, m, "343 Sandwolf")
		exigerBilan(t, m, EquipesDesBots{Illisibles: 1, PaquetsNonFermes: 1})
	}
}

func TestBuildSansLargeurDePersonnalisation(t *testing.T) { // EQ-PERSO
	m := lireLesEquipes(0, false, payloadEcrit(persoBitsHI113, sandwolf))
	exigerSansEquipe(t, m, "343 Sandwolf")
	exigerBilan(t, m, EquipesDesBots{Illisibles: 1, PersoInconnue: true})
}

func TestDeuxPaquetsDeuxEquipes(t *testing.T) { // EQ-CONTRADICTION
	autre := sandwolf
	autre.equipe, autre.jumeau = 0, 0
	m := lireLesEquipes(persoBitsHI113, true, payloadEcrit(persoBitsHI113, sandwolf),
		payloadEcrit(persoBitsHI113, autre))
	exigerSansEquipe(t, m, "343 Sandwolf")
	exigerBilan(t, m, EquipesDesBots{Contradictoires: 1})
}

func TestJumeauDiscordant(t *testing.T) { // EQ-JUMEAU
	b := sandwolf
	b.jumeau = -1
	m := lireLesEquipes(persoBitsHI113, true, payloadEcrit(persoBitsHI113, b))
	exigerEquipe(t, m, "343 Sandwolf", 1)
	exigerBilan(t, m, EquipesDesBots{Lues: 1, JumeauxDiscordants: 1})
}

func TestAucuneEquipeSePublieAMoinsUn(t *testing.T) { // EQ-AUCUNE
	b := sandwolf
	b.equipe, b.jumeau = -1, -1
	m := lireLesEquipes(persoBitsHI113, true, payloadEcrit(persoBitsHI113, b))
	exigerEquipe(t, m, "343 Sandwolf", -1)
	exigerBilan(t, m, EquipesDesBots{Lues: 1})
}

func TestEquipeHorsDomaine(t *testing.T) { // EQ-DOMAINE
	b := sandwolf
	b.equipe, b.jumeau = 9, 9
	m := lireLesEquipes(persoBitsHI113, true, payloadEcrit(persoBitsHI113, b))
	exigerSansEquipe(t, m, "343 Sandwolf")
	exigerBilan(t, m, EquipesDesBots{Illisibles: 1, HorsDomaine: 1})
}

func TestFragmentDuBalayageHorsGrammaire(t *testing.T) { // EQ-FANTOME
	kale := botEcrit{slot: 8, bid: 48, nom: "343 KaleDucky", equipe: 0, jumeau: 0, octetAvantLeBloc: 0xFF}
	m := lireLesEquipes(persoBitsHI113, true, payloadEcrit(persoBitsHI113, kale))
	if len(m.Bots) != 2 || m.Incomplets != 1 {
		t.Fatalf("le balayage devait lire le fragment : %d bot(s), %d incomplet(s)", len(m.Bots), m.Incomplets)
	}
	exigerEquipe(t, m, "343 KaleDucky", 0)
	exigerSansEquipe(t, m, "43 KaleDucky")
	exigerBilan(t, m, EquipesDesBots{Lues: 1, HorsGrammaire: 1})
}

func TestEntreeQueLeBalayageNeLitPas(t *testing.T) { // EQ-HORS-BALAYAGE
	court := botEcrit{slot: 8, bid: 3, nom: "Bob", equipe: 1, jumeau: 1}
	m := lireLesEquipes(persoBitsHI113, true, payloadEcrit(persoBitsHI113, court))
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
