package killsource

// film_table_test.go — LES GARDES DU LOT 1.8 : la table du film est la SOURCE, l inference le
// REPLI, et chaque refus est NOMME.
//
// TOUS TOURNENT EN CI, sans fixture hors depot et sans variable d environnement : les deux
// mini-bobines versionnees portent leur `chunk_00`, donc leur table de joueurs.

import (
	"fmt"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// maxReplicationIndexKS : la borne du champ d indice du dead-state (5 bits). Nommee ici parce que
// c est la propriete que la table doit respecter pour pouvoir epingler quoi que ce soit.
const maxReplicationIndexKS = 31

// TestTableDuFilmLueSurLesDeuxBobines — LA LECTURE, sur deux builds. Elle ne fige pas un compte
// grave dans le marbre : elle verifie que la table est LUE, qu elle rend des noms imprimables, et
// que le rang tient dans l espace des 5 bits du dead-state — les trois proprietes dont
// l epinglage depend.
func TestTableDuFilmLueSurLesDeuxBobines(t *testing.T) {
	for _, cas := range []struct {
		nom, dir, build string
		sieges          int
	}{
		{"000d5950", miniBobineDir, "HI_1_13_0", 8},
		{"e5adf7b2", miniBobineV40Dir, "HI_1_11_0", 23},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			f := chargerBobine(t, cas.dir)
			tab := readFilmTable(f)
			if tab.Refusal != FilmTableRead {
				t.Fatalf("table refusee (%s) — les deux bobines portent leur chunk_00", tab.Refusal)
			}
			if tab.Build != cas.build {
				t.Errorf("build lu %q, attendu %q", tab.Build, cas.build)
			}
			if len(tab.Seats) != cas.sieges {
				t.Errorf("%d siege(s) nomme(s), attendu %d", len(tab.Seats), cas.sieges)
			}
			if tab.Occupied+tab.Vacant != 32 {
				t.Errorf("occupes %d + vacants %d != 32 : la table des joueurs en compte 32",
					tab.Occupied, tab.Vacant)
			}
			for idx, nom := range tab.Seats {
				if idx < 0 || idx > maxReplicationIndexKS {
					t.Errorf("rang %d hors de l espace du dead-state (0..%d)", idx, maxReplicationIndexKS)
				}
				if nom == "" || !imprimableKS(nom) {
					t.Errorf("rang %d : gamertag %q non imprimable", idx, nom)
				}
			}
		})
	}
}

func imprimableKS(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// TestTableDuFilmEpingleTousLesIndicesDeLaBobine — LE GAIN, mesure sur la bobine de reference :
// les huit indices sont LUS, aucun n est infere, et le kill-feed CONFIRME les huit.
func TestTableDuFilmEpingleTousLesIndicesDeLaBobine(t *testing.T) {
	src := chargerSourceKS(t, miniBobineDir)
	res, err := Decode(t.Context(), miniBobineFilm, src, nil)
	if err != nil {
		t.Fatalf("Decode : %v", err)
	}
	p := res.Roster.FilmTable
	if p.Refusal != FilmTableRead || p.Pinned != 8 || p.Inferred != 0 {
		t.Fatalf("provenance = %+v, attendu 8 indices LUS et 0 infere", p)
	}
	if p.Agree != 8 || p.Contradict != 0 {
		t.Errorf("controle par le kill-feed = accord %d / contradiction %d, attendu 8 / 0",
			p.Agree, p.Contradict)
	}
	for i, origine := range res.Roster.IndexSource {
		if origine != OriginFilmTable {
			t.Errorf("indice %d : provenance %q, attendu %q", i, origine, OriginFilmTable)
		}
	}
	if !res.BijectionDetermined {
		t.Error("bijection non DETERMINEE alors que rien n est laisse a l inference")
	}
}

// TestUnSiegeDuFilmNEstPasUnBot — LE GARDE DE NON-REGRESSION DU DEFAUT MESURE A L ECRITURE DU LOT.
//
// `pin` porte deux epinglages depuis ce lot (BOT_METADATA et la table du film) et `isBotIndex` ne
// doit reconnaitre que le premier. Les confondre reclassait les huit joueurs de la mini-bobine en
// BOTS : la sortie tombait de dix lignes publiees a DEUX, la population « mort de bot » n etant
// jamais publiee. La permutation, elle, etait IDENTIQUE — un test qui ne regarderait que la
// bijection n aurait rien vu.
func TestUnSiegeDuFilmNEstPasUnBot(t *testing.T) {
	kf := &killFeed{names: []string{"A", "B"}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 2, BotID: 39, Name: "343 Aloysius"}}}
	r := buildRoster(kf, bm, true, FilmTable{Build: "b", Seats: map[int]string{0: "A", 1: "B"}})
	if r.isBotIndex(0) || r.isBotIndex(1) {
		t.Errorf("un siege de la table du film est pris pour un bot (pin=%v seatPin=%v)",
			r.pin, r.seatPin)
	}
	if !r.isBotIndex(2) {
		t.Error("le slot declare par BOT_METADATA doit rester un bot")
	}
	if r.originOf(0) != OriginFilmTable || r.originOf(2) != OriginBotMeta {
		t.Errorf("provenances = %q / %q", r.originOf(0), r.originOf(2))
	}
}

// TestSiegeQueLeKillFeedIgnoreEntreAuRoster — LE GAIN LE PLUS DIRECT DU LOT, et il est mesure :
// le kill-feed ne nomme que les joueurs qui TUENT ou qui MEURENT, donc six joueurs sur les
// 30 films du corpus de mesure n y figuraient pas et l inference donnait leur indice a un autre
// (`FlukiestGolf` 111fa685 i10, `MarshallG6443` e5adf7b2 i13, `manistoff` a521164d i18,
// `Iskra 20252993` 11de8353 i23, `probablybxllets` 1c5c10cc i22, `Alpha122092` 23ffd885 i4).
func TestSiegeQueLeKillFeedIgnoreEntreAuRoster(t *testing.T) {
	kf := &killFeed{names: []string{"A", "B"}}
	r := buildRoster(kf, botMeta{}, true, FilmTable{
		Build: "b", Seats: map[int]string{0: "A", 1: "MuetAuFeed", 2: "B"},
	})
	if r.table.AddedNames != 1 || r.table.Pinned != 3 {
		t.Fatalf("provenance = %+v, attendu 1 ajout et 3 indices lus", r.table)
	}
	if r.nPlay != 3 {
		t.Errorf("nPlay = %d, attendu 3 : la table etend la borne des indices", r.nPlay)
	}
	r.perm, _ = solveBijection(r, nil, nil, 1)
	if got := r.nameOf(1); got != "MuetAuFeed" {
		t.Errorf("indice 1 porte %q, attendu le nom que SEUL le film donne", got)
	}
	// Le probleme d affectation n est plus carre : il y a plus de noms libres que d indices
	// libres des que la table ajoute quelqu un. Le hongrois doit l accepter, pas l ignorer.
	free, freeNames := r.freeSlots()
	if len(freeNames) < len(free) {
		t.Errorf("%d indices libres pour %d noms libres : le hongrois n est pas defini",
			len(free), len(freeNames))
	}
}

// TestSiegeRefuseQuandUnBotTientDejaLIndice — DEUX LECTURES DU FILM QUI SE CONTREDISENT. On garde
// la plus ancienne et la plus eprouvee (BOT_METADATA, RE_LOG 7ter.62) et on COMPTE ; on ne
// tranche pas en silence.
func TestSiegeRefuseQuandUnBotTientDejaLIndice(t *testing.T) {
	kf := &killFeed{names: []string{"A", "B"}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 2, BotID: 7, Name: "343 Bot"}}}
	r := buildRoster(kf, bm, true, FilmTable{Build: "b", Seats: map[int]string{2: "Humain"}})
	if r.table.BotConflict != 1 || r.table.Pinned != 0 {
		t.Fatalf("provenance = %+v, attendu 1 conflit et 0 indice lu", r.table)
	}
	if !r.isBotIndex(2) {
		t.Error("l indice conteste doit rester au bot")
	}
}

// TestSiegeRefuseQuandLeNomEstDejaEpingle — un meme nom ne peut porter deux indices. Le second
// siege est refuse et compte, jamais ecrase en silence.
func TestSiegeRefuseQuandLeNomEstDejaEpingle(t *testing.T) {
	kf := &killFeed{names: []string{"A"}}
	r := buildRoster(kf, botMeta{}, true, FilmTable{Build: "b", Seats: map[int]string{0: "A", 1: "A"}})
	if r.table.Pinned != 1 || r.table.DuplicateName != 1 {
		t.Fatalf("provenance = %+v, attendu 1 epingle et 1 doublon refuse", r.table)
	}
}

// TestControleParLeKillFeedNeCorrigeJamais — D14 (b) : les votes du kill-feed CONTROLENT la
// lecture, ils ne la corrigent pas. Une contradiction se compte, la valeur publiee ne bouge pas.
func TestControleParLeKillFeedNeCorrigeJamais(t *testing.T) {
	kf := &killFeed{names: []string{"A", "B"}}
	r := buildRoster(kf, botMeta{}, true, FilmTable{Build: "b", Seats: map[int]string{0: "A", 1: "B"}})
	// Votes fabriques : l indice 0 est massivement vote pour "B" (position 1), l indice 1 n a
	// aucun vote. La table, elle, dit "A" a l indice 0.
	r.controlerEpinglage([][]int{{1, 9}, {0, 0}})
	if r.table.Contradict != 1 || r.table.Silent != 1 || r.table.Agree != 0 {
		t.Fatalf("controle = %+v, attendu 1 contradiction et 1 silence", r.table)
	}
	if r.pin[0] != 0 {
		t.Error("la contradiction a DEPLACE l epinglage : le controle a corrige, " +
			"ce qu il ne doit jamais faire")
	}
}

// TestRefusDeTableNommeEtRepliComplet — les causes de refus, chacune NOMMEE, et aucune ne panique
// ni ne rend de lecture partielle.
//
// LES CAUSES SONT MESUREES, PAS SUPPOSEES — et la mesure a corrige une attente ecrite avant elle :
// couper le tampon a MOITIE ne rend pas `tronque` mais `table_introuvable`, parce que le registre
// s arrete a sa fin STRUCTURELLE et que la section d identification, elle, est encore lisible ;
// c est la TABLE qui ne ferme plus a 32 slots. Meme observation qu au lot 1.6.0 sur le chemin du
// rejeu. La cause premiere est bien la troncature ; le diagnostic, lui, nomme ce qui a echoue.
func TestRefusDeTableNommeEtRepliComplet(t *testing.T) {
	f := chargerBobine(t, miniBobineDir)
	sain := f.src.Chunk(0)
	cas := []struct {
		nom     string
		chunk0  []byte
		attendu FilmTableRefusal
	}{
		{"tampon vide", nil, FilmTableNoRegistry},
		{"tampon d un octet", append([]byte(nil), sain[:1]...), FilmTableTruncated},
		{"en-tete seul", append([]byte(nil), sain[:64]...), FilmTableTruncated},
		{"registre ampute de moitie", append([]byte(nil), sain[:len(sain)/2]...), FilmTableNotFound},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			// Le film mute passe par la SOURCE, comme le film reel : `filmsource.MemoryChunks`
			// est l implantation en memoire de la porte aux octets (lot 2.4.2).
			src, err := filmsource.Load(filmsource.MemoryChunks{c.chunk0}, nil)
			if err != nil {
				t.Fatalf("chargement du chunk mute : %v", err)
			}
			mute := &film{src: src}
			var tab FilmTable
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("PANIQUE sur une entree tronquee : %v", r)
					}
				}()
				tab = readFilmTable(mute)
			}()
			if tab.Refusal != c.attendu {
				t.Errorf("cause = %q, attendue %q", tab.Refusal, c.attendu)
			}
			if len(tab.Seats) != 0 {
				t.Errorf("%d siege(s) rendu(s) sur un refus : aucune lecture partielle n est "+
					"acceptable", len(tab.Seats))
			}
		})
	}
}

// TestRefusFaitTomberLaBijectionSurLInferenceEntiere — le repli, COMPTE. Sans table, les indices
// sont tous inferes et la porte de publication reprend son critere d avant le lot.
func TestRefusFaitTomberLaBijectionSurLInferenceEntiere(t *testing.T) {
	kf := &killFeed{names: []string{"A", "B", "C"}}
	r := buildRoster(kf, botMeta{}, true, FilmTable{Refusal: FilmTableNoSection})
	r.perm, _ = solveBijection(r, nil, nil, 1)
	if r.table.Pinned != 0 || r.table.Inferred != 3 {
		t.Fatalf("provenance = %+v, attendu 0 lu et 3 inferes", r.table)
	}
	if r.table.Refusal != FilmTableNoSection {
		t.Errorf("la cause du refus n est pas portee : %q", r.table.Refusal)
	}
}

// TestUnIndiceLibrePourDeuxNomsLibresNEstPasDETERMINE — LE CORRECTIF DE LA REVUE DE JALON M1
// (lentille L4), sur les DEUX cotes du probleme d affectation.
//
// LE DEFAUT QU IL GARDE. Le critere du lot 1.8 etait `Inferred <= 1` : « au plus un indice a
// inferer, donc une seule affectation possible ». La premisse est fausse depuis ce meme lot —
// la table du film AJOUTE au roster les joueurs que le kill-feed ne nomme pas, donc il peut
// rester plus de NOMS libres que d indices libres. Un indice pour deux noms se tranche par les
// votes ; un nom qui n a ni tue ni ete tue ne pese aucun vote, [refine] n a pas deux indices a
// echanger et [bijectionMargin] rend zero. `BijectionDetermined` decidait donc seul, et la
// source du degat, le credit, l assistant et les deux parts de degats partaient sur un occupant
// tire au sort.
//
// LES DEUX CAS SONT DANS LE MEME TEST PARCE QUE LEUR SEULE DIFFERENCE EST LE NOM DU SIEGE 1 :
// « B » est deja au kill-feed (un nom libre, porte forcee et donc OUVERTE — le gain du lot 1.8
// est preserve), « D » ne l est pas (il ENTRE au roster, laisse deux noms libres pour un indice,
// porte FERMEE).
func TestUnIndiceLibrePourDeuxNomsLibresNEstPasDETERMINE(t *testing.T) {
	for _, cas := range []struct {
		nom          string
		siege1       string
		nomsLibres   int
		determinee   bool
		ajoutsRoster int
	}{
		{"un nom libre pour un indice libre", "B", 1, true, 0},
		{"deux noms libres pour un indice libre", "D", 2, false, 1},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			kf := &killFeed{names: []string{"A", "B", "C"}}
			r := buildRoster(kf, botMeta{}, true,
				FilmTable{Build: "b", Seats: map[int]string{0: "A", 1: cas.siege1}})
			r.perm, _ = solveBijection(r, nil, nil, 1)
			if r.table.Pinned != 2 || r.table.Inferred != 1 {
				t.Fatalf("provenance = %+v, attendu 2 indices LUS et 1 infere", r.table)
			}
			if r.table.AddedNames != cas.ajoutsRoster {
				t.Fatalf("ajouts au roster = %d, attendu %d", r.table.AddedNames, cas.ajoutsRoster)
			}
			if r.table.FreeNames != cas.nomsLibres {
				t.Fatalf("noms libres = %d, attendu %d (names=%v nPlay=%d pin=%v)",
					r.table.FreeNames, cas.nomsLibres, r.names, r.nPlay, r.pin)
			}
			if got := r.table.AffectationUnique(); got != cas.determinee {
				t.Errorf("affectation unique = %v, attendu %v : %d indice(s) libre(s) pour %d"+
					" nom(s) libre(s)", got, cas.determinee, r.table.Inferred, r.table.FreeNames)
			}
			res := &Result{BijectionDetermined: r.table.AffectationUnique()}
			if got := res.LineByLinePublishable(); got != cas.determinee {
				t.Errorf("publication ligne par ligne = %v, attendu %v", got, cas.determinee)
			}
		})
	}
}

// TestAffectationUniqueNeRegardePasQueLesIndices — la table de verite du predicat, bornes
// comprises. Zero indice libre reste DETERMINE quels que soient les noms qui restent : ils ne
// portent aucun indice (cf. [hungarianStart]), donc rien n est choisi.
func TestAffectationUniqueNeRegardePasQueLesIndices(t *testing.T) {
	for _, cas := range []struct {
		inferes, nomsLibres int
		veut                bool
	}{
		{0, 0, true}, {0, 1, true}, {0, 7, true},
		{1, 0, true}, {1, 1, true}, {1, 2, false}, {1, 9, false},
		{2, 2, false}, {2, 0, false},
	} {
		t.Run(fmt.Sprintf("%d_indices_%d_noms", cas.inferes, cas.nomsLibres), func(t *testing.T) {
			p := FilmTablePinning{Inferred: cas.inferes, FreeNames: cas.nomsLibres}
			if got := p.AffectationUnique(); got != cas.veut {
				t.Errorf("AffectationUnique() = %v, attendu %v", got, cas.veut)
			}
		})
	}
}

// TestPorteDePublicationNeSOuvrePasSurUnResultatVide — LE ZERO-VALUE NE DOIT PAS MENTIR. Un
// [Result] construit a la main (un test, un appelant qui n a pas decode) ne doit pas devenir
// publiable ligne par ligne du seul fait qu il ne porte aucune provenance.
func TestPorteDePublicationNeSOuvrePasSurUnResultatVide(t *testing.T) {
	if (&Result{}).LineByLinePublishable() {
		t.Error("un Result a zero se declare publiable : le zero-value ment")
	}
	if !(&Result{BijectionDetermined: true}).LineByLinePublishable() {
		t.Error("une bijection DETERMINEE sans alerte doit etre publiable ligne par ligne")
	}
	if !(&Result{BijectionMargin: 3}).LineByLinePublishable() {
		t.Error("le critere d avant le lot (marge stricte) doit continuer d ouvrir la porte")
	}
}

func chargerSourceKS(t *testing.T, dir string) *filmsource.Film {
	t.Helper()
	src, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("bobine illisible sous %s : %v — elle est VERSIONNEE", dir, err)
	}
	return src
}

func chargerBobine(t *testing.T, dir string) *film {
	t.Helper()
	f, err := loadFilm(chargerSourceKS(t, dir))
	if err != nil {
		t.Fatalf("loadFilm(%s) : %v", dir, err)
	}
	return f
}
