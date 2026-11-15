package wire

// registry_replay_service_icons_test.go — le câblage de PRODUCTION des clés d'arme de cause d'une
// destruction de véhicule.
//
// Le service de rejeu ne pose la clé d'arme de la cause (`vehicles[].destruction.causeKey`) que s'il
// reçoit le résolveur de source de dégât du titre ; ce résolveur est branché dans `replayServiceFor`,
// l'endroit que le registre partage entre l'endpoint /replay et la Match View. Les tests du service
// lui passent leur propre résolveur : sans ce test, retirer l'option de `replayServiceFor` laisserait
// toute la suite verte et les lignes « a détruit » sans pictogramme en production.
//
// Le service est construit par le registre réel (`replayServiceFor`), avec l'adapter d'assets du titre
// que le registre résout lui-même ; l'artefact porte une cause dont la clé est celle du dictionnaire
// embarqué du décodeur.

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/duckdb"

	"github.com/JGtm/halo-infinite-film-decoder/killicon"
)

func TestReplayServiceDeProduction_PoseLaCleDArmeDeLaCause(t *testing.T) {
	var tag uint32
	var cle string
	for _, id := range killicon.ResolvedTags() {
		if ic, _ := killicon.Lookup(id); ic.WeaponKey != "" {
			tag, cle = id, ic.WeaponKey
			break
		}
	}
	if cle == "" {
		t.Fatal("aucune règle du dictionnaire embarqué ne porte de clé d'arme : le test ne peut rien prouver")
	}
	root := t.TempDir()
	mappingsReelsDuTitre(t, root)
	const matchID = "c0de0001"
	doc := `{"schemaVersion":115,"matchId":"` + matchID + `","titleSlug":"` + title.DefaultSlug + `",` +
		`"frameCount":1,"bounds":{"minX":0,"minY":0,"maxX":1,"maxY":1},"tracks":[],"vehicles":[` +
		`{"slot":1,"gen":0,"family":"ghost","t0":0,"t1":1,"t1max":1,"end":"destroyed","tEnd":1,` +
		`"destruction":{"cause":` + strconv.FormatUint(uint64(tag), 10) + `}}]}`
	ecrireFichier(t, title.NewPathResolver(root).ReplayArtifactPath(title.DefaultSlug, matchID), doc)

	resolver := games.NewStaticResolver(title.DefaultSlug)
	resolver.RegisterAssetURL(halo_infinite.NewAssetURLAdapter()) // comme buildTitleRuntime, au démarrage
	reg := (&ServiceRegistry{cfg: &config.AppConfig{RepoRoot: root}}).WithTitleResolver(resolver)
	svc := reg.replayServiceFor(&duckdb.PlayerDB{TitleSlug: title.DefaultSlug, SharedReader: lecteurPartageAbsent{}})

	got, err := svc.GetReplay(context.Background(), matchID)
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if len(got.Vehicles) != 1 || got.Vehicles[0].Destruction == nil {
		t.Fatalf("une vie détruite avec un bloc de destruction attendue : %+v", got.Vehicles)
	}
	if k := got.Vehicles[0].Destruction.CauseKey; k != cle {
		t.Errorf("causeKey = %q, attendu %q : le résolveur de source de dégât n'est pas branché par replayServiceFor", k, cle)
	}
}

// lecteurPartageAbsent : la base partagée n'est pas ouverte ; la carte du match se dégrade (pas de
// fond), le reste du rejeu se sert, ce qui suffit à lire les causes des destructions.
type lecteurPartageAbsent struct{}

func (lecteurPartageAbsent) Get(context.Context) (*sql.DB, func(), error) {
	return nil, func() {}, errors.New("base partagée non ouverte dans ce test")
}

// mappingsReelsDuTitre recopie sous `root` les deux tables du titre dont dépend la lecture du rejeu
// (noms d'armes, libellés) : un test qui écrirait les siennes vérifierait son fixture, pas le catalogue livré.
func mappingsReelsDuTitre(t *testing.T, root string) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("répertoire courant : %v", err)
	}
	for ; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "config", "titles")); err == nil {
			break
		}
		if filepath.Dir(dir) == dir {
			t.Fatal("racine du dépôt introuvable")
		}
	}
	src := title.NewPathResolver(dir).TitleMappingsDir(title.DefaultSlug)
	dst := title.NewPathResolver(root).TitleMappingsDir(title.DefaultSlug)
	for _, f := range []string{mappings.WeaponNamesPath(src), filepath.Join(src, "replay_labels.toml")} {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("lecture du mapping réel %s : %v", f, err)
		}
		ecrireFichier(t, filepath.Join(dst, filepath.Base(f)), string(body))
	}
}
