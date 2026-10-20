//go:build research

package killcollector

// porteurs_au_sync_parite_research_test.go — LE SYNC ET LA CUISSON NOMMENT LES MEMES VIES.
//
// # CE QU'IL COMPARE
//
// Le document interne que `replay.PortagesAuSync` assemble (`EntreePorteursAuSync.DocumentInterne`)
// sur l'entree que le collecteur lui donne en production, rebatie par les COUTURES du paquet
// (`decoderLeFilm`, `lireLePontDuCollecteur`, `entreeDuRegistre`, `depsDuPlacement.entreeDesPorteurs`),
// et le document de la cuisson (`replaybuild.Builder.BuildBytes`) du meme film sous les memes faits.
// Chaque vie du sync qui recouvre une piste de la cuisson sur le meme slot doit porter la MEME
// identite (xuid ou bot), et aucune vie publiee ne reste sans identite, d'un cote comme de l'autre
// (« les vies anonymes n'existent pas », hors mode non gere : Firefight).
//
// La DECOUPE des vies peut differer entre les deux chemins (le collecteur lit ses positions sous ses
// propres options, cf. `lireLePontDuCollecteur`) : seule l'identite des vies appariees est comparee.
//
// # SES TEMOINS ET SA MUTATION
//
// `69b16f5d`, `4db2574e`, `c75f33b8`, `9f57c612` : des bots qui se relaient sur un meme index, que la
// cuisson nomme par l'entite `ti=9` vivante a la creation de leur corps (sur `4db2574e`, le bot
// `343 Ritzy`, `bid(0.0)`). Retirer la table des sieges et les occupants des options du document
// interne (`optionsDuRegistre`) laisse une vie sans identite sur `69b16f5d`, `c75f33b8` et
// `9f57c612` (mesure du 2026-10-15).
//
// UN FILM PAR PROCESSUS, ET SANS LES VARIABLES IL SE SAUTE (films et faits ne sont pas versionnes ;
// la cuisson ecrit ses faits sous PARITE_RACINE, une COPIE de la racine — jamais le depot) :
//
//	levelup replay-facts-export --out <faits> <id>    (faits du match, base en lecture)
//	PARITE_FILM=<id court> PARITE_FAITS=<faits> PARITE_CACHE=<racine data/cache des films> \
//	  PARITE_RACINE=<copie de la racine> go test -tags=research \
//	  -run '^TestLeSyncEtLaCuissonNommentLesMemesVies$' ./internal/sync/killcollector/

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/replayidentity"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/testutil"
)

func TestLeSyncEtLaCuissonNommentLesMemesVies(t *testing.T) {
	court, faitsDir := os.Getenv("PARITE_FILM"), os.Getenv("PARITE_FAITS")
	cache, racine := os.Getenv("PARITE_CACHE"), os.Getenv("PARITE_RACINE")
	if court == "" || faitsDir == "" || cache == "" || racine == "" {
		t.Skip("PARITE_FILM, PARITE_FAITS, PARITE_CACHE et PARITE_RACINE requis : parite sautee")
	}
	faits, err := replaybuild.ReadFactsFile(filepath.Join(faitsDir, court+".facts.json"))
	if err != nil {
		t.Fatalf("%s : %v", court, err)
	}
	if strings.HasPrefix(faits.GameVariantName, "Firefight") {
		t.Skipf("%s : Firefight, mode non gere par l'application", court)
	}
	sync := viesDuDocumentInterne(t, court, cache, faits)
	cuisson := viesDeLaCuisson(t, court, filepath.Join(cache, "film_chunks", court), racine, faits)
	comparerLeNommage(t, court, sync, cuisson)
}

// vieComparee : une piste reduite a son slot, ses bornes et son identite.
type vieComparee struct {
	slot       uint32
	debut, fin int
	identite   string
}

// vies reduit les pistes d'un document a leur identite (xuid, sinon nom du bot).
func vies(tracks []replay.Track) []vieComparee {
	out := make([]vieComparee, 0, len(tracks))
	for _, tr := range tracks {
		id := tr.XUID
		if id == "" {
			id = tr.Bot
		}
		out = append(out, vieComparee{slot: tr.Slot, debut: tr.StartFrame, fin: tr.EndFrame, identite: id})
	}
	return out
}

// viesDuDocumentInterne rejoue la passe du collecteur et rend les vies du document des porteurs.
func viesDuDocumentInterne(t *testing.T, court, cache string, faits replaybuild.FactsFile) []vieComparee {
	t.Helper()
	ctx := context.Background()
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	constructeur, err := replaybuild.NewBuilder(ctx, repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	film, ok, err := filmcache.LoadFilm(cache, court)
	if err != nil || !ok {
		t.Fatalf("%s : film absent ou illisible du cache %s : %v", court, cache, err)
	}
	carte, err := constructeur.ResolveMapEntry(faits.MapNames)
	if err != nil {
		t.Fatalf("%s : %v", court, err)
	}
	opts := decfilm.DefaultOptions()
	opts.Carte = &carte
	res, err := decoderLeFilm(ctx, faits.MatchID, film, &opts)
	if err != nil {
		t.Fatalf("%s : decodage killsource : %v", court, err)
	}
	ids := identitesDesFaits(faits.MatchFacts)
	lectures, _, err := lireLePontDuCollecteur(ctx, film, carte, ids, faits.MatchID)
	if err != nil {
		t.Fatalf("%s : etage du pont : %v", court, err)
	}
	entree := entreeDuRegistre(lectures, ids, replayidentity.BotIdentities(res), faits.MatchID)
	libelles, objectifs := cataloguesDuPlacement(ctx, repoRoot, title.DefaultSlug)
	mat := materiauDIsolement{registre: replay.BuildIdentityRegistry(ctx, entree), positions: lectures.positions,
		film: film, contexte: lectures.contexte, carte: carte, profil: profilCalibre(res), identite: entree}
	entreeDesPorteurs := depsDuPlacement{libelles: libelles, objectifs: objectifs}.entreeDesPorteurs(ctx, faits.MatchID, mat, ids)
	doc, bilan := entreeDesPorteurs.DocumentInterne(ctx)
	if bilan.Gardes.Aucune() {
		t.Skipf("%s : variante %q sans porteur, le sync n'assemble aucun document", court, ids.Variante)
	}
	_, portes := replay.PortagesAuSync(ctx, entreeDesPorteurs)
	t.Logf("%s : %d portage(s) relu(s) au sync", court, portes.Intervalles)
	if bilan.ViesSansIdentite != 0 {
		t.Errorf("%s : %d vie(s) du document interne sans identite", court, bilan.ViesSansIdentite)
	}
	return vies(doc.Tracks)
}

// viesDeLaCuisson cuit le film en memoire, sous la racine COPIEE (la cuisson y range ses faits).
func viesDeLaCuisson(t *testing.T, court, filmDir, racine string, faits replaybuild.FactsFile) []vieComparee {
	t.Helper()
	ctx := context.Background()
	b, err := replaybuild.NewBuilder(ctx, racine, title.DefaultSlug)
	if err != nil {
		t.Fatalf("constructeur sous %s : %v", racine, err)
	}
	built, err := b.BuildBytes(ctx, faits.MatchID, faits.MapNames, filmDir, faits.MatchFacts)
	if err != nil {
		t.Fatalf("%s : cuisson : %v", court, err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(built.Blob, &doc); err != nil {
		t.Fatalf("%s : document de la cuisson illisible : %v", court, err)
	}
	if doc.Coverage != nil && doc.Coverage.Bridge.UnnamedLives != 0 {
		t.Errorf("%s : %d vie(s) PUBLIEE(S) sans identite par la cuisson", court, doc.Coverage.Bridge.UnnamedLives)
	}
	return vies(doc.Tracks)
}

// comparerLeNommage exige la meme identite sur chaque couple de vies appariees (meme slot, bornes qui
// se recouvrent).
func comparerLeNommage(t *testing.T, court string, sync, cuisson []vieComparee) {
	t.Helper()
	appariees := 0
	for _, s := range sync {
		for _, c := range cuisson {
			if c.slot != s.slot || c.fin < s.debut || s.fin < c.debut {
				continue
			}
			appariees++
			if c.identite != s.identite {
				t.Errorf("%s : slot %d [%d-%d] nomme %q au sync, %q a la cuisson", court, s.slot,
					s.debut, s.fin, s.identite, c.identite)
			}
		}
	}
	if appariees == 0 {
		t.Fatalf("%s : aucune vie appariee entre le sync (%d) et la cuisson (%d) : la comparaison ne "+
			"prouverait rien", court, len(sync), len(cuisson))
	}
	t.Logf("%s : %d vie(s) au sync, %d piste(s) a la cuisson, %d couple(s) apparie(s)", court,
		len(sync), len(cuisson), appariees)
}
