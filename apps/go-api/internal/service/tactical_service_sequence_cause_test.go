package service

// tactical_service_sequence_cause_test.go — « ANALYSE EN COURS » d'une sequence sans artefact : la
// decision est celle du resolveur partage (`analysis_pending_parts`, cause `replay`), pas une copie.

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// toutesLesCausesDeSequence : les causes « donnees » (hors refus) ; chaque cas de
// TestSequence_Indisponible verifie qu'une cause ne se lit pas comme une autre.
var toutesLesCausesDeSequence = []error{
	port.ErrSequenceAnalyseEnCours, port.ErrSequenceFilmNonAnalyse, port.ErrSequenceFilmAReanalyser,
	port.ErrSequenceCarteNonMesuree, port.ErrSequenceSansPosition, port.ErrReplayNotAvailable,
}

// sequenceSansArtefact sert une sequence dont l'artefact est absent, avec le resolveur donne.
func sequenceSansArtefact(t *testing.T, r *AnalysisPendingResolver) error {
	t.Helper()
	repo, store := corpusSequence()
	store.pistes = nil
	svc := serviceSequence(repo, store)
	if r != nil {
		svc = svc.WithAnalysisPending(r)
	}
	_, err := svc.Sequence(context.Background(), demandeSequence())
	return err
}

func TestSequence_SansArtefact_AnalyseEnCours(t *testing.T) {
	repo := &probeRepoFactice{probes: map[string]domain.AnalysisProbe{"m2": synchroIlYA(time.Hour, nil)}}
	err := sequenceSansArtefact(t, resolveurDeTest(repo, true))
	if !errors.Is(err, port.ErrSequenceAnalyseEnCours) {
		t.Fatalf("err = %v, want analyse en cours (match synchronise il y a 1 h, rejeu absent)", err)
	}
}

func TestSequence_SansArtefact_HorsDeLaFenetreOuFilmPerdu(t *testing.T) {
	cas := map[string]domain.AnalysisProbe{
		"synchro de plus de 24 h": synchroIlYA(25*time.Hour, nil),
		"film declare perdu":      synchroIlYA(time.Hour, func(p *domain.AnalysisProbe) { p.FilmAbsent = true }),
	}
	for nom, probe := range cas {
		t.Run(nom, func(t *testing.T) {
			repo := &probeRepoFactice{probes: map[string]domain.AnalysisProbe{"m2": probe}}
			if err := sequenceSansArtefact(t, resolveurDeTest(repo, true)); !errors.Is(err, port.ErrSequenceFilmNonAnalyse) {
				t.Fatalf("err = %v, want film non analyse", err)
			}
		})
	}
}

// Sans resolveur (titre sans film.kill_source) ou lecture de l'avancement en echec : jamais « en cours ».
func TestSequence_SansArtefact_SansResolveurOuLectureEnEchec(t *testing.T) {
	if err := sequenceSansArtefact(t, nil); !errors.Is(err, port.ErrSequenceFilmNonAnalyse) {
		t.Fatalf("sans resolveur : err = %v, want film non analyse", err)
	}
	repo := &probeRepoFactice{err: errors.New("boum")}
	if err := sequenceSansArtefact(t, resolveurDeTest(repo, true)); !errors.Is(err, port.ErrSequenceFilmNonAnalyse) {
		t.Fatalf("lecture en echec : err = %v, want film non analyse", err)
	}
}

// Un artefact present sans horloge est « a reanalyser », meme quand l'analyse du match est en cours :
// la cause `replay` ne s'applique qu'a un artefact ABSENT.
func TestSequence_ArtefactSansHorloge_PasEnCours(t *testing.T) {
	repo, store := corpusSequence()
	store.pistes.OriginMs = nil
	probe := &probeRepoFactice{probes: map[string]domain.AnalysisProbe{"m2": synchroIlYA(time.Hour, nil)}}
	svc := serviceSequence(repo, store).WithAnalysisPending(resolveurDeTest(probe, true))
	if _, err := svc.Sequence(context.Background(), demandeSequence()); !errors.Is(err, port.ErrSequenceFilmAReanalyser) {
		t.Fatalf("err = %v, want film a reanalyser", err)
	}
	if probe.appels != 0 {
		t.Errorf("%d lecture(s) de l'avancement pour un artefact present", probe.appels)
	}
}
