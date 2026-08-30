package main

// cmd_archive_films_test.go — ce qui se teste sans base ni reseau.
//
// La selection elle-meme (`filmsManquants`) ouvre la base partagee : elle releve d une passe
// d integration, et elle est couverte de fait par le `--dry-run` de la commande. Ce fichier
// verrouille les deux proprietes qui, cassees, feraient perdre des films en silence :
// la porte d entree (une passe lancee sans tokens echouerait apres avoir lu tout le registre)
// et la reconnaissance d un film DEJA archive (la re-telecharger a chaque passe annulerait
// l interet du cache).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

func TestArchiveFilms_GamertagObligatoireHorsDryRun(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	err := runArchiveFilms(cfg, []string{"--cache", t.TempDir()})
	if err == nil {
		t.Fatal("aucune erreur : la passe partirait sans tokens et n echouerait qu apres avoir lu " +
			"tout le registre")
	}
	if !strings.Contains(err.Error(), "--gamertag") {
		t.Errorf("erreur = %q, attendu un message sur --gamertag", err.Error())
	}
	// Le message doit dire que le joueur est INDIFFERENT : le film s obtient par identifiant de
	// match. Sans cette precision, un operateur croit devoir re-authentifier chaque joueur —
	// c est l erreur que la version 1 du registre a induite.
	if !strings.Contains(err.Error(), "n importe lequel") {
		t.Errorf("le message n explique pas que le joueur est indifferent : %q", err.Error())
	}
}

// TestArchiveFilms_DryRunNExigeAucunToken : un plan doit pouvoir se faire sans authentification.
// L echec attendu porte sur la BASE (absente sous ce repoRoot), pas sur les tokens.
func TestArchiveFilms_DryRunNExigeAucunToken(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	err := runArchiveFilms(cfg, []string{"--dry-run", "--cache", t.TempDir()})
	if err == nil {
		t.Fatal("erreur attendue : la base partagee n existe pas sous ce repoRoot")
	}
	if strings.Contains(err.Error(), "--gamertag") {
		t.Errorf("un --dry-run exige des tokens : %v", err)
	}
}

// TestFilmDejaEnCache : LE MANIFESTE fait foi, pas les chunks.
//
// `filmcache.Write` ecrit les chunks d abord et le manifeste EN DERNIER, precisement pour que
// le manifeste soit le marqueur de commit. Juger la presence sur un dossier de chunks ferait
// donc passer un film INTERROMPU pour archive, et il ne serait jamais recupere.
func TestFilmDejaEnCache(t *testing.T) {
	racine := t.TempDir()
	if err := filmcache.EnsureDirs(racine); err != nil {
		t.Fatal(err)
	}
	const match = "aabbccdd-1111-2222-3333-444455556666"
	court := titlePkg.FilmShortMatchID(match)

	if filmDejaEnCache(racine, match) {
		t.Error("cache vide : le film ne doit pas etre vu comme archive")
	}

	// Des chunks SANS manifeste = ecriture interrompue. Le film reste a recuperer.
	if err := os.MkdirAll(filmcache.ChunkDir(racine, court), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filmcache.ChunkDir(racine, court), "chunk_00.bin"),
		[]byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if filmDejaEnCache(racine, match) {
		t.Error("chunks orphelins SANS manifeste : le film n est pas lisible, il doit rester a " +
			"recuperer — sinon une ecriture interrompue le perd definitivement")
	}

	// Manifeste ecrit : le film est archive.
	if err := filmcache.Write(racine, court, []filmcache.WriteChunk{
		{Index: 0, ChunkType: 1, Data: []byte("entete")},
	}); err != nil {
		t.Fatal(err)
	}
	if !filmDejaEnCache(racine, match) {
		t.Error("manifeste present : le film doit etre reconnu comme archive, sinon chaque passe " +
			"le retelecharge")
	}
}
