package replayartifacts

// derivations_writer_test.go — UN SEUL SEGMENT D'ECRITURE PAR PASSE DE DERIVATIONS
// (constat C7 de la revue A-R1).
//
// # Le defaut que ces tests ferment
//
// Chacune des quatre familles acquerait le writer POUR SON COMPTE — `reporterT0Film`,
// `persisterResumesUsage`, `persisterStatsBombe`, `persisterPositions` — soit QUATRE
// acquisitions successives, chacune bornee a `acquireWriterTimeout` (60 s cote wire). Sur le
// chemin du depot d'ouvrier, ces quatre acquisitions vivent dans un HANDLER HTTP dont le
// serveur ferme l'ecriture a 30 s : un depot pendant qu'un cycle de sync tient le lease pouvait
// donc tourner jusqu'a 4 x 60 s, voir sa connexion coupee, et faire enregistrer a l'ouvrier un
// ECHEC DE DEPOT pour un artefact pourtant range. Rejoue, le depot rejouait les quatre
// derivations et ecrivait une passe de plus dans chaque table append-only.
//
// # La regle
//
// LE SEGMENT APPARTIENT A LA PASSE, PAS A LA FAMILLE. [Deriver] l'acquiert AU PLUS UNE FOIS,
// paresseusement (une passe qui n'a rien a ecrire n'en ouvre aucun), et le relache une fois.

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

// compteurWriter : une source d'acquisition qui compte les appels et les retraits.
type compteurWriter struct {
	acquis   int
	relaches int
	err      error
}

func (c *compteurWriter) acquerir(context.Context) (*sql.DB, func(), error) {
	c.acquis++
	if c.err != nil {
		return nil, nil, c.err
	}
	// db nil : aucune famille de ce test n'ecrit vraiment (les preuves d'ecriture sont dans
	// les tests d'integration). Ce qui est mesure ici, c'est le NOMBRE d'acquisitions.
	return nil, func() { c.relaches++ }, nil
}

// TestDeriver_UnSeulSegmentWriterPourLesQuatreFamilles — LE test du constat C7.
func TestDeriver_UnSeulSegmentWriterPourLesQuatreFamilles(t *testing.T) {
	dir := t.TempDir()
	chemin := artefactADeriver(t, dir, "writer1")
	// Une acquisition EN ERREUR : elle traverse les quatre familles sans qu'aucune n'ecrive,
	// donc elle mesure exactement le nombre d'acquisitions et rien d'autre.
	c := &compteurWriter{err: errors.New("lease shared tenu")}

	Deriver(context.Background(), DerivationsDeps{
		RepoRoot: racineDepot(t), TitleSlug: "halo_infinite", Gamertag: "testeur",
		AcquireWriter: c.acquerir,
	}, []ArtefactRange{{MatchID: "writer1", Path: chemin}})

	if c.acquis != 1 {
		t.Fatalf("writer acquis %d fois pour UNE passe de derivations, attendu 1 — quatre "+
			"acquisitions en file derriere un lease peuvent depasser le WriteTimeout du "+
			"serveur sur le chemin du depot d'ouvrier (constat C7)", c.acquis)
	}
}

// TestDeriver_RienAEcrireNAcquiertAucunWriter — la propriete que le regroupement ne doit PAS
// perdre : une passe sans rien a ecrire n'ouvre aucun segment. halo_5 ne declare aucune clef
// `film.*`, et le document ne porte ni coup d'envoi ni trajectoire.
func TestDeriver_RienAEcrireNAcquiertAucunWriter(t *testing.T) {
	dir := t.TempDir()
	chemin := artefactSansRienADeriver(t, dir, "writer2")
	c := &compteurWriter{}

	Deriver(context.Background(), DerivationsDeps{
		RepoRoot: racineDepot(t), TitleSlug: "halo_5", Gamertag: "testeur",
		AcquireWriter: c.acquerir,
	}, []ArtefactRange{{MatchID: "writer2", Path: chemin}})

	if c.acquis != 0 {
		t.Fatalf("writer acquis %d fois alors qu'il n'y avait rien a ecrire, attendu 0 — "+
			"l'acquisition doit rester PARESSEUSE", c.acquis)
	}
	if c.relaches != 0 {
		t.Errorf("%d retrait(s) sans acquisition", c.relaches)
	}
}
