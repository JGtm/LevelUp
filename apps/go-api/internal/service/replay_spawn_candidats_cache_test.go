package service

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
)

// avecCompteurDeLectures remplace le cache des sidecars par un cache dont le chargeur compte ses lectures
// de fichier ; le cache d'origine revient en fin de test.
func avecCompteurDeLectures(t *testing.T) *atomic.Int64 {
	t.Helper()
	var lectures atomic.Int64
	origine := sidecarsDeReapparitions
	sidecarsDeReapparitions = nouveauCacheAEmpreinte(func(chemin string) (*sidecarReapparitions, error) {
		lectures.Add(1)
		return decoderSidecarReapparitions(chemin)
	})
	t.Cleanup(func() { sidecarsDeReapparitions = origine })
	return &lectures
}

// autresMatchsDuDecor appelle la lecture des reapparitions de la famille « assassin » sur les matchs du decor.
func autresMatchsDuDecor(t *testing.T, d *decorPrevision) bilanCandidats {
	t.Helper()
	familles, err := mappings.LoadSpawnFamiliesFromFile(filepath.Join(title.NewPathResolver(d.root).TitleMappingsDir(title.DefaultSlug), fichierFamillesReapparition))
	if err != nil {
		t.Fatalf("familles de modes: %v", err)
	}
	svc := NewReplayService(title.DefaultSlug, d.root, d.reg, nil).(*replayService)
	_, bilan := svc.reapparitionsDesAutresMatchs(context.Background(), d.reg.matchs, familles, "assassin", prevuID)
	return bilan
}

// Le second passage sur les memes sidecars ne relit aucun fichier : le bilan et les reapparitions sont ceux
// du premier.
func TestReapparitionsDesAutresMatchs_SecondAppelSansLecture(t *testing.T) {
	lectures := avecCompteurDeLectures(t)
	d := nouveauDecor(t)
	d.autres(t, 3)

	premier := autresMatchsDuDecor(t, d)
	if premier.Lus != 3 || lectures.Load() != 3 {
		t.Fatalf("premier appel : bilan %+v, %d lectures, attendu 3 lus et 3 lectures", premier, lectures.Load())
	}
	second := autresMatchsDuDecor(t, d)
	if second != premier {
		t.Errorf("second appel : bilan %+v, attendu %+v", second, premier)
	}
	if lectures.Load() != 3 {
		t.Errorf("second appel : %d lectures au total, attendu 3 (aucune relecture)", lectures.Load())
	}
}

// Un sidecar reecrit (taille et date changees) est relu et son contenu neuf est servi.
func TestReapparitionsDesAutresMatchs_FichierReecritRelu(t *testing.T) {
	lectures := avecCompteurDeLectures(t)
	d := nouveauDecor(t)
	d.match(t, autreID(1), varianteAss, sidecarAvecReapparitions(domain.TacticalRasterSchemaVersion, 10))
	if b := autresMatchsDuDecor(t, d); b.Lus != 1 || lectures.Load() != 1 {
		t.Fatalf("premier appel : bilan %+v, %d lectures", b, lectures.Load())
	}

	chemin := title.NewPathResolver(d.root).TacticalRasterPath(title.DefaultSlug, autreID(1))
	ecrire(t, chemin, sidecarAvecReapparitions(domain.TacticalRasterSchemaVersion, 10, 110, 20))
	futur := time.Now().Add(time.Hour)
	if err := os.Chtimes(chemin, futur, futur); err != nil {
		t.Fatal(err)
	}
	sc, err := lireSidecarReapparitions(chemin)
	if err != nil {
		t.Fatalf("lecture apres reecriture: %v", err)
	}
	if lectures.Load() != 2 {
		t.Errorf("%d lectures au total, attendu 2 (le fichier reecrit est relu)", lectures.Load())
	}
	if n := len(sc.Joueurs[0].Spawns); n != 3 {
		t.Errorf("spawns servis = %d, attendu 3 (le contenu reecrit)", n)
	}
}

// Un sidecar lu puis disparu compte « absent » : le cache ne le sert plus.
func TestReapparitionsDesAutresMatchs_FichierDisparuAbsent(t *testing.T) {
	avecCompteurDeLectures(t)
	d := nouveauDecor(t)
	d.match(t, autreID(1), varianteAss, sidecarAvecReapparitions(domain.TacticalRasterSchemaVersion, 10))
	if b := autresMatchsDuDecor(t, d); b.Lus != 1 {
		t.Fatalf("premier appel : bilan %+v, attendu 1 lu", b)
	}

	if err := os.Remove(title.NewPathResolver(d.root).TacticalRasterPath(title.DefaultSlug, autreID(1))); err != nil {
		t.Fatal(err)
	}
	if b := autresMatchsDuDecor(t, d); b.Absents != 1 || b.Lus != 0 {
		t.Errorf("apres suppression : bilan %+v, attendu 1 absent et 0 lu", b)
	}
}

// Un echec de lecture n'est pas garde : la lecture suivante, le fichier repare, reussit.
func TestReapparitionsDesAutresMatchs_EchecNonGarde(t *testing.T) {
	lectures := avecCompteurDeLectures(t)
	d := nouveauDecor(t)
	d.match(t, autreID(1), varianteAss, "pas du json")
	if b := autresMatchsDuDecor(t, d); b.Illisibles != 1 {
		t.Fatalf("premier appel : bilan %+v, attendu 1 illisible", b)
	}
	chemin := title.NewPathResolver(d.root).TacticalRasterPath(title.DefaultSlug, autreID(1))
	ecrire(t, chemin, sidecarAvecReapparitions(domain.TacticalRasterSchemaVersion, 10, 110))
	if b := autresMatchsDuDecor(t, d); b.Lus != 1 {
		t.Errorf("apres reparation : bilan %+v, attendu 1 lu", b)
	}
	if lectures.Load() != 2 {
		t.Errorf("%d lectures, attendu 2", lectures.Load())
	}
}
