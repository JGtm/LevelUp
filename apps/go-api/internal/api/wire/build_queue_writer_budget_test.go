package wire

// build_queue_writer_budget_test.go — LE BUDGET D'ATTENTE DU DEPOT TIENT DANS LE BUDGET DE
// REPONSE DU SERVEUR (constat C7 de la revue A-R1).
//
// # Ce que ce garde-rail tient
//
// `StoreBuildArtifact` derive SYNCHRONEMENT l'artefact qu'il vient de ranger, et cette
// derivation acquiert un segment d'ecriture shared. L'attente est bornee par
// `acquireWriterDepot` — mais une borne ne vaut que RELATIVEMENT au `WriteTimeout` du serveur :
// au-dela, la connexion est coupee et l'ouvrier enregistre un ECHEC DE DEPOT pour un artefact
// pourtant range, qu'il rejouera (une passe de plus dans chaque table append-only).
//
// Les deux valeurs vivent dans deux paquets que rien ne relie : `internal/api/wire` ne peut pas
// importer `cmd/server`. Le lien est donc pose ICI, en lisant la source — meme parti pris que
// les ratchets d'`internal/archlint`. Baisser le `WriteTimeout` du serveur sans revoir ce
// budget fait desormais rougir.

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"
)

// motifWriteTimeout capture la valeur du `WriteTimeout` du serveur HTTP, en secondes.
var motifWriteTimeout = regexp.MustCompile(`WriteTimeout:\s*(\d+)\s*\*\s*time\.Second`)

// TestBudgetWriterDepotTientDansLeWriteTimeout : l'attente du writer, MAJOREE, doit laisser au
// serveur de quoi ecrire la reponse.
func TestBudgetWriterDepotTientDansLeWriteTimeout(t *testing.T) {
	chemin := filepath.Join("..", "..", "..", "cmd", "server", "main.go")
	src, err := os.ReadFile(chemin) //nolint:gosec // chemin fixe, relatif au paquet
	if err != nil {
		t.Fatalf("lecture de %s: %v — le garde-rail ne peut pas se taire quand il ne sait pas",
			chemin, err)
	}
	m := motifWriteTimeout.FindSubmatch(src)
	if m == nil {
		t.Fatalf("`WriteTimeout: N * time.Second` introuvable dans %s — la forme a change, "+
			"ce garde-rail doit etre mis a jour AVEC elle (ne pas le supprimer : c'est lui qui "+
			"empeche le depot d'ouvrier de depasser le budget de reponse du serveur)", chemin)
	}
	secondes, err := strconv.Atoi(string(m[1]))
	if err != nil {
		t.Fatalf("WriteTimeout illisible (%q): %v", m[1], err)
	}
	writeTimeout := time.Duration(secondes) * time.Second

	// LA MARGE EST LA MOITIE, ET C'EST DELIBERE : le reste du budget paie les ecritures des
	// quatre familles pour CE match, plus la serialisation de la reponse.
	if acquireWriterDepot*2 > writeTimeout {
		t.Errorf("acquireWriterDepot = %v pour un WriteTimeout serveur de %v : un depot qui "+
			"attend le writer peut faire couper la connexion, et l'ouvrier enregistre un echec "+
			"de depot pour un artefact pourtant range (constat C7)", acquireWriterDepot, writeTimeout)
	}
	// Et il reste PLUS COURT que le budget des actions admin : ce n'est pas le meme appelant.
	if acquireWriterDepot >= acquireWriterTimeout {
		t.Errorf("acquireWriterDepot (%v) >= acquireWriterTimeout (%v) : le depot est un handler "+
			"HTTP, l'action admin une commande d'operateur — ils n'ont pas le meme budget",
			acquireWriterDepot, acquireWriterTimeout)
	}
}
