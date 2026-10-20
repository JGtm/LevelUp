package replaybuild

// artifact_verdict_test.go — LE VERDICT A TROIS SORTIES, ET SA PREUVE PAR MUTATION (lot 4.4.1).
//
// # CE QUE CES TESTS FERMENT
//
// `UpToDate()` etait une egalite de `SchemaVersion`, donc toute montee de schema marquait le parc
// entier « a redecoder » — 15 a 100 s par match — alors que la montee la plus frequente ne change
// QUE la publication. Le verdict distingue desormais trois conduites, et ces tests prouvent que
// chacune s obtient par la SEULE cause qui doit la produire.
//
// LA PREUVE EST UNE MUTATION, DANS LES DEUX SENS : on part d un artefact « a jour », on mute UNE
// chose (la revision d une couche, la version de schema, la presence des faits), et on verifie
// que le verdict bascule — puis on remet la valeur et on verifie qu il revient. Un test qui ne
// verifierait que l aller passerait sur un verdict constant.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// couchesCourantes rend une table `layers` qui declare les revisions du binaire courant. Jamais
// de litteral : la table suit toute montee de revision (meme raison qu `artefactCourantAvecCouches`).
func couchesCourantes() map[string]string {
	c := replay.RevisionsCourantesDesCouches()
	return map[string]string{
		"tracks":     c["grammar"],
		"objectives": c["killsource"],
		"bounds":     c["grammar"],
		"matchId":    c["publication"],
	}
}

// digestDeTest rend le digest d un artefact au schema donne portant ces couches.
func digestDeTest(schema int, layers map[string]string) Digest {
	return Digest{MatchID: "m", SchemaVersion: schema, Tracks: 1, Layers: layers}
}

// TestVerdictAJourExigeLeSchemaETLesCouches : les deux conditions, chacune prouvee par sa mutation.
func TestVerdictAJourExigeLeSchemaETLesCouches(t *testing.T) {
	aJour := digestDeTest(replay.SchemaVersion, couchesCourantes())
	if v := aJour.Verdict(true); v != VerdictAJour {
		t.Fatalf("temoin : un artefact au schema courant et aux couches courantes rend %q", v)
	}
	// MUTATION 1 — la revision d une COUCHE DE DECODAGE bascule le verdict en `redecoder`, et la
	// presence des faits n y change rien : les faits eux-memes ont ete produits par la couche qui
	// a bouge.
	mute := digestDeTest(replay.SchemaVersion, couchesCourantes())
	mute.Layers["tracks"] = "grammar-1999-01-01"
	for _, faits := range []bool{true, false} {
		if v := mute.Verdict(faits); v != VerdictRedecoder {
			t.Errorf("grammaire perimee, faits=%v : verdict %q, attendu %q", faits, v, VerdictRedecoder)
		}
	}
	// RETOUR — on remet la revision courante, le verdict revient a `a-jour`. Sans ce sens, le
	// test passerait sur un verdict constamment `redecoder`.
	mute.Layers["tracks"] = replay.RevisionsCourantesDesCouches()["grammar"]
	if v := mute.Verdict(true); v != VerdictAJour {
		t.Errorf("revision remise : verdict %q, attendu %q", v, VerdictAJour)
	}
}

// TestVerdictRepublierQuandSeuleLaPublicationABouge : le cas que tout le lot sert.
func TestVerdictRepublierQuandSeuleLaPublicationABouge(t *testing.T) {
	// UN ARTEFACT DU SCHEMA PRECEDENT qui declare les couches de decodage COURANTES : c est
	// exactement ce que produit une montee de schema sans changement de decodage — celle du lot
	// 4.2.1-b, par exemple.
	d := digestDeTest(replay.SchemaVersion-1, couchesCourantes())
	if v := d.Verdict(true); v != VerdictRepublier {
		t.Fatalf("faits presents : verdict %q, attendu %q", v, VerdictRepublier)
	}
	// LE PIEGE QU IL N Y A PAS : sans les faits, `republier` retombe sur `redecoder`. JAMAIS un
	// quatrieme etat « je republierais si j avais les faits ».
	if v := d.Verdict(false); v != VerdictRedecoder {
		t.Errorf("faits absents : verdict %q, attendu %q — il n existe pas de quatrieme etat",
			v, VerdictRedecoder)
	}
	// RETOUR : les faits reviennent, le verdict redevient `republier`.
	if v := d.Verdict(true); v != VerdictRepublier {
		t.Errorf("faits revenus : verdict %q, attendu %q", v, VerdictRepublier)
	}
}

// TestVerdictSansCouchesDeclarees : tout le parc anterieur au schema 62 se redecode, et c est le
// defaut SUR — on ne peut pas prouver son decodage intact.
func TestVerdictSansCouchesDeclarees(t *testing.T) {
	for _, schema := range []int{replay.SchemaVersion, replay.SchemaVersion - 1} {
		d := digestDeTest(schema, nil)
		if v := d.Verdict(true); v != VerdictRedecoder {
			t.Errorf("schema %d sans `layers` : verdict %q, attendu %q", schema, v, VerdictRedecoder)
		}
	}
}

// TestVerdictIgnoreLaFamilleDePublicationDansLesCouches : si la comparaison incluait la
// publication, `republier` serait INATTEIGNABLE — la publication est justement ce qui bouge.
func TestVerdictIgnoreLaFamilleDePublicationDansLesCouches(t *testing.T) {
	layers := couchesCourantes()
	layers["matchId"] = "publication-1" // une publication franchement perimee
	d := digestDeTest(replay.SchemaVersion-1, layers)
	if v := d.Verdict(true); v != VerdictRepublier {
		t.Errorf("publication perimee dans `layers` : verdict %q, attendu %q — la famille de "+
			"publication ne doit pas entrer dans le test du decodage", v, VerdictRepublier)
	}
}

// TestArtifactVerdictLitLeDisqueEtLaPresenceDesFaits : la forme que les sites de decision
// emploient, prouvee SUR FICHIERS et dans les deux sens.
func TestArtifactVerdictLitLeDisqueEtLaPresenceDesFaits(t *testing.T) {
	repo := t.TempDir()
	res := title.NewPathResolver(repo)
	const matchID = "000d5950-83d9-423f-ab55-d068a7237b9f"
	artefact := res.ReplayArtifactPath(title.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(artefact), 0o750); err != nil {
		t.Fatalf("dossier des artefacts : %v", err)
	}
	blob, err := json.Marshal(map[string]any{
		"schemaVersion": replay.SchemaVersion - 1,
		"matchId":       matchID,
		"layers":        couchesCourantes(),
	})
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	if err = os.WriteFile(artefact, blob, 0o600); err != nil {
		t.Fatalf("pose de l artefact : %v", err)
	}
	// SANS le fichier de faits : `redecoder`.
	v, ok := ArtifactVerdict(res, title.DefaultSlug, matchID)
	if !ok || v != VerdictRedecoder {
		t.Fatalf("faits absents : verdict %q (ok=%v), attendu %q", v, ok, VerdictRedecoder)
	}
	// AVEC le fichier de faits : `republier`. Le verdict ne lit PAS son contenu — sa fraicheur se
	// juge au rejeu (`lireLesFaitsFrais`), qui redecode si l en-tete ne tient pas.
	faits := res.FilmFactsPath(title.DefaultSlug, matchID)
	if err = os.MkdirAll(filepath.Dir(faits), 0o750); err != nil {
		t.Fatalf("dossier des faits : %v", err)
	}
	if err = os.WriteFile(faits, []byte("des octets quelconques"), 0o600); err != nil {
		t.Fatalf("pose des faits : %v", err)
	}
	if v, ok = ArtifactVerdict(res, title.DefaultSlug, matchID); !ok || v != VerdictRepublier {
		t.Fatalf("faits presents : verdict %q (ok=%v), attendu %q", v, ok, VerdictRepublier)
	}
	// RETOUR : on retire les faits, le verdict rebascule.
	if err = os.Remove(faits); err != nil {
		t.Fatalf("retrait des faits : %v", err)
	}
	if v, ok = ArtifactVerdict(res, title.DefaultSlug, matchID); !ok || v != VerdictRedecoder {
		t.Fatalf("faits retires : verdict %q (ok=%v), attendu %q", v, ok, VerdictRedecoder)
	}
	// ARTEFACT ABSENT : ok=false, et le verdict par defaut est le plus sur.
	if v, ok = ArtifactVerdict(res, title.DefaultSlug, "ffffffff-0000-0000-0000-000000000000"); ok {
		t.Errorf("artefact absent : ok=true, attendu false (verdict %q)", v)
	}
}

// TestWriteArtifactRefuseUneRegressionDeDecodage : `wouldDowngrade` apprend les revisions.
//
// LE CAS QUE LE GARDE NE VOYAIT PAS : un binaire en retard republie un artefact, et sa montee de
// schema le faisait passer par le silence delibere du garde a schema different.
func TestWriteArtifactRefuseUneRegressionDeDecodage(t *testing.T) {
	repo := t.TempDir()
	out := filepath.Join(repo, "artefact.json")
	enPlace, err := json.Marshal(map[string]any{
		"schemaVersion": replay.SchemaVersion,
		"matchId":       "m",
		"tracks":        []map[string]any{{"xuid": "1"}},
		"layers":        couchesCourantes(),
	})
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	if err = os.WriteFile(out, enPlace, 0o600); err != nil {
		t.Fatalf("pose : %v", err)
	}
	perimees := couchesCourantes()
	perimees["tracks"] = "grammar-1999-01-01"
	candidat, err := json.Marshal(map[string]any{
		"schemaVersion": replay.SchemaVersion + 1, // une montee de schema, qui passait sans un mot
		"matchId":       "m",
		"tracks":        []map[string]any{{"xuid": "1"}},
		"layers":        perimees,
	})
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	if _, oui := wouldDowngrade(out, candidat); !oui {
		t.Error("un candidat cuit sous une grammaire perimee N EST PAS refuse : le garde ne lit " +
			"pas les revisions")
	}
	// RETOUR : le meme candidat, couches courantes, doit passer.
	bon, err := json.Marshal(map[string]any{
		"schemaVersion": replay.SchemaVersion + 1,
		"matchId":       "m",
		"tracks":        []map[string]any{{"xuid": "1"}},
		"layers":        couchesCourantes(),
	})
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	if _, oui := wouldDowngrade(out, bon); oui {
		t.Error("un candidat aux couches COURANTES est refuse : le garde mord trop large")
	}
}

// TestFamilleDeRevisionEstCelleDuProducteur : la lecture du prefixe vit chez le producteur, et ce
// test le verifie sur les cinq familles reelles.
func TestFamilleDeRevisionEstCelleDuProducteur(t *testing.T) {
	for famille, revision := range replay.RevisionsCourantesDesCouches() {
		if lue := replay.FamilleDeRevision(revision); lue != famille {
			t.Errorf("la revision %q se classe en famille %q, attendu %q", revision, lue, famille)
		}
	}
	for _, mauvaise := range []string{"", "sansTiret", "-commenceParUnTiret"} {
		if lue := replay.FamilleDeRevision(mauvaise); lue != "" {
			t.Errorf("%q rend la famille %q, attendu la chaine vide", mauvaise, lue)
		}
	}
}
