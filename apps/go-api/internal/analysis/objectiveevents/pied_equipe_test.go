package objectiveevents

// pied_equipe_test.go — L'ÉQUIPE D'UN ÉVÉNEMENT DU PIED EST À L'OCTET 37 (lot 1.1.3).
//
// # CE QUE CE TEST VERROUILLE, ET CONTRE QUOI EXACTEMENT
//
// Il fait tourner le décodeur de production sur un BLOC RÉEL du pied de `53ce4390`, recopié
// octet pour octet depuis le film (cf. le fichier de provenance à côté du binaire), et exige le
// TRIPLET attendu : slot 2, équipe 1, instant 133 033 ms.
//
// Le bloc est choisi pour SÉPARER LES CHAMPS, pas seulement pour valoir 1 quelque part. Ses
// quatre octets en jeu valent :
//
//	octet 36 (slot)   2
//	octet 37 (équipe) 1
//	octet 38          1
//	octet 55          0
//
// Un test qui n'exigerait que « l'équipe vaut 1 » resterait VERT si `footerByteTeam` glissait
// sur l'octet 36 (le SLOT, déclaré la ligne au-dessus dans le même bloc de constantes) d'un bloc
// où slot et équipe coïncident — c'est le défaut qu'a trouvé la revue R1 sur la première version
// de cette fixture, dont les octets 36, 37 et 38 valaient tous 1. Ici, les substitutions
// plausibles donnent des valeurs DIFFÉRENTES, sauf une, et cette exception est mesurée :
//
//	footerByteTeam = 36 -> 2, donc ROUGE (et symétriquement footerByteSlot = 37 -> 1, ROUGE)
//	footerByteTeam = 55 -> 0, donc ROUGE
//	footerByteTeam = 38 -> 1, donc VERT — ET AUCUN BLOC RÉEL NE PEUT FAIRE MIEUX
//
// L'octet 38 est un DOUBLON EXACT de l'octet 37 : mesure du 2026-09-14 sur tous les blocs de
// quatre films et deux builds (`bcb6d393`, `53ce4390`, `7344d24f`, `64e8adfa`), 190 blocs sur
// 190 avec `b37 == b38`. Aucune fixture ne peut donc les séparer, et c'est précisément ce qui
// fonde la règle du lot : on lit UN des deux (l'octet 37, celui que
// `.ai/archive/V7/RESEARCH_THEATER_RE.md:534` identifie) et on ne lit JAMAIS l'autre, parce que
// deux lectures d'un même fait divergent un jour sans que rien ne le dise. Le contre-test
// ci-dessous le vérifie dans les deux sens : 36 et 55 séparent, 38 ne sépare pas.
//
// # POURQUOI LA VALEUR ATTENDUE N'EST PAS CIRCULAIRE
//
// L'équipe 1 de `2535430195856593` sur ce match est ÉTABLIE AILLEURS QUE DANS CE BLOC, par la
// trame d'état du film (rang de la table des slots de `chunk_00` -> xuid, i-ème entité ti=9 ->
// désignateur d'équipe). C'est ce que publie l'oracle corpus
// `filmdec.TestResidusPiedOctetEquipe`, qui reste la mesure de référence — 665/665 sur quatorze
// films le 2026-09-13, rejoué le 2026-09-14 sur trois films (`53ce4390`, `64e8adfa`,
// `7344d24f`) : 173 événements sur 173 pour l'octet 37, 84 sur 173 pour l'octet 55, et QUATRE
// lectures en accord parfait sur cent quatre-vingts essayées.
//
// # CE QU'IL NE FAIT PAS
//
// Il ne rejoue pas le balayage aveugle des soixante octets : ça, c'est l'instrument corpus, qui
// exige le cache de films. Ce test-ci tient dans le dépôt, sans film et sans base, et il tient
// la seule chose qu'un test unitaire puisse tenir — que le décodeur lit BIEN les octets que la
// mesure a désignés.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// Le bloc de référence, et d'où il vient. Ces constantes sont la provenance MISE EN CODE : le
// fichier `testdata/pied_bloc_53ce4390.PROVENANCE.txt` dit la même chose en français.
const (
	piedFixture = "testdata/pied_bloc_53ce4390.bin"
	piedFilm    = "53ce4390"
	// piedChunk : le NUMÉRO du chunk de pied au manifeste du film (`chunk_40.bin`, type 3).
	piedChunk = 40
	// piedLo / piedHi : la tranche, EN OCTETS, du pied DÉCOMPRESSÉ.
	piedLo = 165000
	piedHi = 166930
	// Ce que le décodeur doit rendre sur cette tranche — le TRIPLET, pas un seul octet.
	piedAttenduTime = 133033
	piedAttenduSlot = 2
	piedAttenduTeam = 1
	piedAttenduXUID = uint64(2535430195856593)
	// Ce que rendraient les substitutions plausibles de `footerByteTeam` sur ce bloc.
	piedOctet38 = 1 // doublon exact de l'octet 37 : 190 blocs sur 190 (4 films, 2 builds)
	piedOctet55 = 0 // ce que la production lisait avant le 2026-09-14
)

// TestPiedEquipeOctet37 : le décodeur de production rend le TRIPLET attendu sur le bloc réel.
func TestPiedEquipeOctet37(t *testing.T) {
	e := piedSeulEvenement(t, scanTh10Events(lirePiedFixture(t)))
	if e.XUID != piedAttenduXUID {
		t.Fatalf("bloc inattendu : xuid=%d (attendu %d) — la fixture n'est plus celle que la "+
			"provenance décrit", e.XUID, piedAttenduXUID)
	}
	// Les trois champs sont exigés ENSEMBLE : c'est ce qui empêche un offset de glisser sur le
	// voisin sans que rien ne bouge.
	if e.Slot != piedAttenduSlot {
		t.Errorf("SLOT lu au mauvais octet : %d (attendu %d, octet %d du bloc)",
			e.Slot, piedAttenduSlot, footerByteSlot)
	}
	if e.TimeMS != piedAttenduTime {
		t.Errorf("INSTANT lu au mauvais octet : %d (attendu %d, octets %d..%d, gros-boutiste)",
			e.TimeMS, piedAttenduTime, footerByteTime, footerByteTime+3)
	}
	if e.Team != piedAttenduTeam {
		t.Errorf("ÉQUIPE LUE AU MAUVAIS OCTET.\n"+
			"  équipe rendue  : %d\n  équipe prouvée : %d (trame d'état du film, oracle "+
			"filmdec.TestResidusPiedOctetEquipe)\n  octet lu       : %d\n"+
			"L'équipe d'un événement du pied est à l'octet 37 du bloc de %d (665/665 sur "+
			"quatorze films). Sur CE bloc, l'octet 36 vaut %d (c'est le slot) et l'octet 55 "+
			"vaut %d : si l'un des deux est revenu dans `footerByteTeam`, c'est la régression "+
			"que ce test ferme.",
			e.Team, piedAttenduTeam, footerByteTeam, footerBlockBytes,
			piedAttenduSlot, piedOctet55)
	}
}

// TestPiedBlocSepareLesChamps : LE CONTRE-TEST. Il ne relit pas le décodeur — il vérifie que la
// FIXTURE a bien le pouvoir de faire rougir, en confrontant les octets en jeu. Sans lui, le test
// ci-dessus pourrait passer au vert pour la mauvaise raison le jour où la fixture est régénérée
// sur un bloc où slot et équipe coïncident.
func TestPiedBlocSepareLesChamps(t *testing.T) {
	bloc := lirePiedFixture(t)
	ebs := piedDebutBlocBits(t, bloc)
	lire := func(octet int) int { return int(filmsource.OctetAuBit(bloc, ebs+octet*8)) }
	equipe := lire(footerByteTeam)
	if equipe != piedAttenduTeam {
		t.Fatalf("octet %d = %d, attendu %d", footerByteTeam, equipe, piedAttenduTeam)
	}
	// Les deux substitutions qui DOIVENT séparer.
	for _, cas := range []struct {
		octet, valeur int
		quoi          string
	}{
		{footerByteSlot, piedAttenduSlot, "le slot, déclaré juste au-dessus dans le même bloc"},
		{55, piedOctet55, "ce que la production lisait avant le 2026-09-14"},
	} {
		got := lire(cas.octet)
		if got != cas.valeur {
			t.Errorf("octet %d = %d, attendu %d", cas.octet, got, cas.valeur)
			continue
		}
		if got == equipe {
			t.Errorf("LA FIXTURE NE SÉPARE PLUS l'octet %d (%s) de l'octet %d : les deux valent "+
				"%d. Un bloc où les deux lectures coïncident ne peut pas faire rougir la "+
				"régression — en régénérer un où elles diffèrent.",
				cas.octet, cas.quoi, footerByteTeam, equipe)
		}
	}
	// La substitution qu'AUCUN bloc réel ne peut séparer, et c'est un fait MESURÉ, pas un aveu
	// d'échec : l'octet 38 est un doublon exact de l'octet 37 (190 blocs sur 190, 4 films,
	// 2 builds). La règle qui en découle — lire l'un, ne jamais lire l'autre — est portée par le
	// commentaire de `footerByteTeam` ; ici on ne fait que constater le doublon.
	if got := lire(38); got != piedOctet38 || got != equipe {
		t.Errorf("l'octet 38 vaut %d : il n'est plus le doublon exact de l'octet 37 (%d) que "+
			"190 blocs sur 190 montrent. Si le doublon cesse, la question « lequel des deux le "+
			"jeu écrit-il ? » se rouvre et ne se tranche PAS ici : la remonter au plan.",
			got, equipe)
	}
}

// TestFooterEventsSurUnFilm : LE CHEMIN EXPORTÉ, de bout en bout.
//
// Les deux tests ci-dessus appellent `scanTh10Events` en direct, ce qui laissait `FooterEvents`
// — le point d'entrée que le lot 1.7 lira — sans aucune couverture (constat de la revue R1).
// Celui-ci monte un `filmsource.Film` en mémoire à partir de la MÊME fixture : un répertoire
// temporaire, un `chunk_40.bin` qui porte les octets bruts, et le manifeste positionnel qui le
// déclare de type 3. `FooterEvents` doit alors choisir ce chunk (c'est `footerData` qui décide,
// sur le type du manifeste et non sur la position) et rendre l'événement avec son équipe.
func TestFooterEventsSurUnFilm(t *testing.T) {
	film := piedFilmEnMemoire(t)
	e := piedSeulEvenement(t, FooterEvents(film))
	if e.Team != piedAttenduTeam || e.Slot != piedAttenduSlot || e.TimeMS != piedAttenduTime {
		t.Fatalf("FooterEvents rend t=%d slot=%d équipe=%d (attendu %d / %d / %d)",
			e.TimeMS, e.Slot, e.Team, piedAttenduTime, piedAttenduSlot, piedAttenduTeam)
	}
	// Et le chemin qui CONSOMME ces événements voit le même bloc, par son point d'entrée public.
	// DEPUIS LE LOT 1.7.3, `Extract` PREND L'ÉQUIPE DU PIED : le roster passé ici est VIDE, et
	// `TeamID` vaut quand même celle du film. C'est exactement le basculement que ce test
	// verrouillait « pour plus tard » — il le verrouille maintenant dans l'autre sens.
	evs, ctl := Extract(piedFilm, "Strongholds:Arena", film, MapRoster{})
	if len(evs) != 1 {
		t.Fatalf("Extract rend %d événement(s), attendu 1", len(evs))
	}
	if evs[0].TimeMS == nil || *evs[0].TimeMS != piedAttenduTime {
		t.Fatalf("Extract : instant %v, attendu %d", evs[0].TimeMS, piedAttenduTime)
	}
	if evs[0].ObjectiveType != ObjectiveTypeZone || evs[0].Source != SourceTh10 {
		t.Fatalf("Extract : type=%q source=%q, attendu %q / %q",
			evs[0].ObjectiveType, evs[0].Source, ObjectiveTypeZone, SourceTh10)
	}
	if evs[0].TeamID == nil || *evs[0].TeamID != piedAttenduTeam {
		t.Fatalf("Extract : TeamID=%v, attendu %d — l'équipe vient du PIED (octet 37), pas du "+
			"roster, qui est vide ici", evs[0].TeamID, piedAttenduTeam)
	}
	// LE CONTRÔLE COMPTE, IL NE POSE RIEN : roster vide, donc un silence et rien d'autre.
	if ctl.Film != 1 || ctl.Silence != 1 || ctl.Accord != 0 || ctl.Contradiction != 0 {
		t.Fatalf("contrôle %+v : un roster VIDE doit rendre 1 lecture du film et 1 silence", ctl)
	}
}

// TestExtractPrendLEquipeDuPiedEtCompteLeControle : le contrôle DISTINGUE l'accord de la
// contradiction, et la contradiction ne change PAS la valeur publiée.
//
// C'est la moitié que le test ci-dessus ne couvre pas : avec un roster vide, `accord` et
// `contradiction` restent structurellement à zéro et une implémentation qui les confondrait
// passerait. Ici la feuille de match DIT quelque chose — d'abord la même équipe, puis une autre.
func TestExtractPrendLEquipeDuPiedEtCompteLeControle(t *testing.T) {
	film := piedFilmEnMemoire(t)
	xuid := formatXUID(piedAttenduXUID)

	evs, ctl := Extract(piedFilm, "Strongholds:Arena", film, MapRoster{xuid: piedAttenduTeam})
	if ctl.Accord != 1 || ctl.Contradiction != 0 || ctl.Silence != 0 {
		t.Fatalf("contrôle %+v : la feuille dit la MÊME équipe, c'est un accord", ctl)
	}
	if evs[0].TeamID == nil || *evs[0].TeamID != piedAttenduTeam {
		t.Fatalf("Extract : TeamID=%v, attendu %d", evs[0].TeamID, piedAttenduTeam)
	}

	autre := piedAttenduTeam + 1
	evs, ctl = Extract(piedFilm, "Strongholds:Arena", film, MapRoster{xuid: autre})
	if ctl.Accord != 0 || ctl.Contradiction != 1 || ctl.Silence != 0 {
		t.Fatalf("contrôle %+v : la feuille dit une AUTRE équipe, c'est une contradiction", ctl)
	}
	if evs[0].TeamID == nil || *evs[0].TeamID != piedAttenduTeam {
		t.Fatalf("Extract : TeamID=%v alors que la feuille disait %d — une contradiction se "+
			"COMPTE, elle ne corrige rien : le film fait foi", evs[0].TeamID, autre)
	}
}

// TestCaptureScorerPrendLeDernierDuCluster : `captureScorer` est l'autre consommateur de
// [FooterEvent] (chemin CTF), et le lot 1.1 a renommé les champs qu'il lit. Il est PUR — une
// liste d'événements, un instant de burst — donc il se teste sans film : ce qui manquait pour
// couvrir le chemin CTF, ce n'est pas un fixture, c'est ce test.
//
// CE QUI RESTE A ZERO, ET POURQUOI : `extractCTF` lui-même. Il apparie des BURSTS DE CAPTURE —
// détectés dans les chunks de type 2, à six tiers distincts — avec les événements du pied. Une
// fixture de pied ne peut donc pas l atteindre : il faudrait un chunk de jeu, c est-à-dire un
// autre fixture, pour un chemin que ce lot n a fait que renommer. Dit, pas contourné.
func TestCaptureScorerPrendLeDernierDuCluster(t *testing.T) {
	evs := []FooterEvent{
		{TimeMS: 1000, Team: 0, XUID: 11},
		{TimeMS: 100000, Team: 1, XUID: 22}, // hors de la fenêtre de coïncidence
		{TimeMS: 1500, Team: 1, XUID: 33},   // le t MAX du cluster : c'est l'acteur
		{TimeMS: 1200, Team: 0, XUID: 44},
	}
	got, ok := captureScorer(evs, 1400)
	if !ok {
		t.Fatal("aucun événement coïncident alors que trois le sont")
	}
	if got.XUID != 33 || got.Team != 1 {
		t.Fatalf("acteur rendu : xuid=%d équipe=%d (attendu 33 / 1 — le t MAX du cluster)",
			got.XUID, got.Team)
	}
	// Et hors de toute fenêtre, il se tait au lieu de rendre le moins mauvais.
	if _, ok := captureScorer(evs, 500000); ok {
		t.Fatal("captureScorer a rendu un acteur pour un burst sans aucun événement coïncident")
	}
}

// piedFilmEnMemoire monte un film d'un seul chunk, de type 3, portant la fixture.
func piedFilmEnMemoire(t *testing.T) *filmsource.Film {
	t.Helper()
	bloc := lirePiedFixture(t)
	dir := t.TempDir()
	nom := filepath.Join(dir, "chunk_40.bin")
	if err := os.WriteFile(nom, bloc, 0o600); err != nil {
		t.Fatalf("écriture de %s : %v", nom, err)
	}
	film, err := filmsource.LoadDir(dir, []filmsource.ChunkMeta{
		{Index: piedChunk, ChunkType: chunkTypePied},
	})
	if err != nil {
		t.Fatalf("chargement du film de test : %v", err)
	}
	// Le chunk ne doit surtout pas avoir été « décompressé » : la fixture est déjà en clair, et
	// `filmsource.Inflate` ne touche que ce qui commence par 0x78. Si un jour le premier octet
	// de la fixture valait 0x78 par hasard, ce contrôle le dirait au lieu de laisser le test
	// échouer plus loin pour une raison illisible.
	if got := film.Chunk(0); len(got) != len(bloc) {
		t.Fatalf("le chunk chargé fait %d octets, la fixture %d — filmsource l'a transformé",
			len(got), len(bloc))
	}
	return film
}

// piedSeulEvenement exige qu'il n'y ait qu'UN événement et le rend.
func piedSeulEvenement(t *testing.T, evs []FooterEvent) FooterEvent {
	t.Helper()
	if len(evs) != 1 {
		t.Fatalf("la tranche porte %d événement(s) th=10, attendu exactement 1 — la fixture "+
			"n'est plus celle que la provenance décrit", len(evs))
	}
	return evs[0]
}

// piedDebutBlocBits rend l'offset BIT du début du bloc de 60 octets dans la tranche, en refaisant
// le seul chemin qui le connaisse : le marqueur de fin. Il n'y a qu'un bloc dans la fixture.
func piedDebutBlocBits(t *testing.T, bloc []byte) int {
	t.Helper()
	total := len(bloc) * 8
	for b := 0; b <= total-32; b++ {
		if filmsource.OctetAuBit(bloc, b) == 0 && filmsource.OctetAuBit(bloc, b+8) == 0 &&
			filmsource.OctetAuBit(bloc, b+16) == 0x2e && filmsource.OctetAuBit(bloc, b+24) == 0xe0 {
			ebs := b - footerBlockBytes*8
			if ebs >= 0 && int(filmsource.OctetAuBit(bloc, ebs+footerByteType*8)) == 10 {
				return ebs
			}
		}
	}
	t.Fatal("aucun bloc th=10 dans la fixture")
	return 0
}

// lirePiedFixture charge le bloc versionné.
func lirePiedFixture(t *testing.T) []byte {
	t.Helper()
	blob, err := os.ReadFile(piedFixture)
	if err != nil {
		t.Fatalf("fixture %s illisible : %v — elle est VERSIONNÉE, son absence est une erreur",
			piedFixture, err)
	}
	if len(blob) != piedHi-piedLo {
		t.Fatalf("fixture %s : %d octets, attendu %d (la tranche [%d, %d) du pied)",
			piedFixture, len(blob), piedHi-piedLo, piedLo, piedHi)
	}
	return blob
}

// TestPiedBlocProvenance : LA FIXTURE EST-ELLE BIEN CES OCTETS-LÀ DU FILM ?
//
// Sauté sans `PIED_FILM_DIR` (le cache de films ne vit pas dans le dépôt). Avec, il recoupe la
// tranche depuis le film et la compare octet pour octet ; avec `PIED_BLOC_UPDATE=1`, il la
// réécrit. C'est la seule porte d'écriture : une fixture binaire sans provenance vérifiable est
// un fait sans source. Une variable d'environnement plutôt qu'un `flag` : un `flag.Bool` dans un
// paquet qui n'en déclare aucun autre ajouterait un drapeau global au binaire de test pour un
// usage qui sert une fois par an.
//
// LE PIED SE CHOISIT PAR LE MANIFESTE, JAMAIS PAR LA POSITION (correctif de la revue R1) : une
// première version prenait `film.Chunk(NumChunks()-1)`, ce qui marche sur `53ce4390` et ment sur
// tout film dont le cache porte des chunks APRÈS le pied. Cette porte lit donc
// `film_manifests/<id>.json`, exige que `chunk_type` y vaille 3 pour le chunk nommé par la
// provenance, et passe ces métadonnées à `filmsource` — la sélection est ensuite celle de la
// production, [footerData].
func TestPiedBlocProvenance(t *testing.T) {
	dir := os.Getenv("PIED_FILM_DIR")
	if dir == "" {
		t.Skip("PIED_FILM_DIR vide : la tranche ne peut pas être recoupée sur le film")
	}
	// NORMALISER UNE FOIS, A L'ENTREE, AVANT TOUTE DERIVATION (constat C4 de la revue R2). La
	// complétion d'un shell rend `…/film_chunks/53ce4390/`, séparateur final compris :
	// `filepath.Base` l'absorbe, `filepath.Dir` non — la remontée vers le cache s'arrêtait alors
	// un cran trop bas et la porte échouait sur un manifeste introuvable, message trompeur pour
	// une entrée parfaitement valide.
	dir = filepath.Clean(dir)
	if filepath.Base(dir) != piedFilm {
		t.Fatalf("PIED_FILM_DIR désigne %s : la fixture vient de %s", filepath.Base(dir), piedFilm)
	}
	film, err := filmsource.LoadDir(dir, piedMetaDuManifeste(t, dir))
	if err != nil {
		t.Fatalf("chargement de %s : %v", dir, err)
	}
	pied, ok := footerData(film)
	if !ok {
		t.Fatalf("aucun chunk de type %d au manifeste de %s : la provenance ne tient plus",
			chunkTypePied, piedFilm)
	}
	if len(pied) < piedHi {
		t.Fatalf("pied de %d octets : la tranche [%d, %d) n'y tient pas", len(pied), piedLo, piedHi)
	}
	tranche := make([]byte, piedHi-piedLo)
	copy(tranche, pied[piedLo:piedHi])
	if os.Getenv("PIED_BLOC_UPDATE") == "1" {
		if err := os.WriteFile(piedFixture, tranche, 0o600); err != nil {
			t.Fatalf("écriture de %s : %v", piedFixture, err)
		}
		t.Fatalf("1 fixture réécrite : %s (%d octets) ; relancer sans PIED_BLOC_UPDATE pour vérifier",
			piedFixture, len(tranche))
	}
	fixture := lirePiedFixture(t)
	for i := range tranche {
		if tranche[i] != fixture[i] {
			t.Fatalf("divergence à l'octet %d de la tranche (%d du pied) : film %#02x, fixture %#02x",
				i, piedLo+i, tranche[i], fixture[i])
		}
	}
	t.Logf("tranche [%d, %d) du pied de %s : %d octets identiques à la fixture",
		piedLo, piedHi, piedFilm, len(tranche))
}

// piedMetaDuManifeste lit `<cache>/film_manifests/<id>.json` et rend les métadonnées de chunk.
// Il EXIGE que le chunk nommé par la provenance ([piedChunk]) y soit déclaré de type 3 : sans
// cette vérification, la porte de régénération devinerait le pied par sa position.
func piedMetaDuManifeste(t *testing.T, dir string) []filmsource.ChunkMeta {
	t.Helper()
	chemin := filepath.Join(filepath.Dir(filepath.Dir(dir)), "film_manifests", piedFilm+".json")
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin dérivé de PIED_FILM_DIR, poste de dev
	if err != nil {
		t.Fatalf("manifeste %s illisible : %v — le pied se choisit par le manifeste, jamais par "+
			"la position du chunk dans le film", chemin, err)
	}
	var doc struct {
		Chunks []struct {
			Index     int `json:"index"`
			ChunkType int `json:"chunk_type"`
			StartMS   int `json:"start_ms"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("manifeste %s illisible : %v", chemin, err)
	}
	out := make([]filmsource.ChunkMeta, 0, len(doc.Chunks))
	typeDuPied := -1
	for _, c := range doc.Chunks {
		out = append(out, filmsource.ChunkMeta{Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS})
		if c.Index == piedChunk {
			typeDuPied = c.ChunkType
		}
	}
	if typeDuPied != chunkTypePied {
		t.Fatalf("le manifeste de %s donne chunk_type=%d au chunk %d, attendu %d : la provenance "+
			"de la fixture nomme ce chunk comme LE pied", piedFilm, typeDuPied, piedChunk, chunkTypePied)
	}
	return out
}
