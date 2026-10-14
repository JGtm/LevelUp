package service

// tactical_service_sequence_test.go — LA SEQUENCE D'UNE MORT cote service : acteurs de la BASE (tueur
// du journal, coequipiers de la mort), pistes de l'artefact, horloge du match convertie par l'origine
// du film, rayon du registre ; et tous les cas qui rendent la sequence indisponible.

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// fakePistes double le lecteur de pistes.
type fakePistes struct {
	pistes *domain.TacticalPistes
	err    error
	appels int
}

func (f *fakePistes) Pistes(context.Context, string) (*domain.TacticalPistes, error) {
	f.appels++
	return f.pistes, f.err
}

// ligneDePiste : une piste image par image (100 ms), a abscisse fixe `x` ou croissante.
func ligneDePiste(xuid, bot string, de, a int, x func(t int) float32) domain.TacticalPiste {
	p := domain.TacticalPiste{XUID: xuid, Bot: bot}
	for t := de; t <= a; t++ {
		p.Points = append(p.Points, domain.TacticalPistePoint{T: t, X: x(t), Y: 0})
	}
	return p
}

func fixe(v float32) func(int) float32 { return func(int) float32 { return v } }

// corpusSequence : m2 (Oddball, rayon 24 m, coup d'envoi a 2 000 ms) ; je meurs a 3 000 ms de match,
// l'image zero du film tombant a 1 000 ms de match (originMs) : ma mort est a l'image 20.
//
//	moi     court le long de x (1 m par image) jusqu'a l'image 20 ;
//	Rival   (tueur, xuid) immobile a x = 40 ; Ami (coequipier, ligne visible du corpus) a x = 25.
func corpusSequence() (*mockTacticalRepo, *fakePistes) {
	repo := corpusEquipe()
	t0 := int64(2000)
	repo.univ = repo.pos.Univers
	for i := range repo.univ.Matchs {
		if repo.univ.Matchs[i].MatchID == "m2" {
			repo.univ.Matchs[i].T0Ms = &t0
		}
	}
	origine := int64(1000)
	return repo, &fakePistes{pistes: &domain.TacticalPistes{
		FrameIntervalMS: 100, OriginMs: &origine,
		Tracks: []domain.TacticalPiste{
			ligneDePiste(tsMoi, "", 0, 20, func(t int) float32 { return float32(t) }),
			ligneDePiste(tsAdv, "", 0, 80, fixe(40)),
			ligneDePiste(tsAmi, "", 0, 80, fixe(25)),
		},
	}}
}

func serviceSequence(repo *mockTacticalRepo, store port.TacticalSequenceStore) *TacticalService {
	_, matchs := corpusDetail()
	svc := sourcesCompletes(matchs).service(repo, capsCompletes())
	if store != nil {
		svc = svc.WithSequenceStore(store)
	}
	return svc
}

func demandeSequence() domain.TacticalSequenceRequest {
	return domain.TacticalSequenceRequest{MapID: tsCarte, MatchID: "m2", VictimXUID: tsMoi, TimeMs: 3000}
}

func TestSequence_Nominal(t *testing.T) {
	repo, store := corpusSequence()
	seq, err := serviceSequence(repo, store).Sequence(context.Background(), demandeSequence())
	if err != nil {
		t.Fatalf("Sequence: %v", err)
	}
	if seq.RayonM != 24 || seq.Mort.X != 20 || len(seq.Etapes) != 8 {
		t.Fatalf("sequence = rayon %v, mort %+v, %d etapes", seq.RayonM, seq.Mort, len(seq.Etapes))
	}
	roles := map[string]string{}
	for _, a := range seq.Acteurs {
		roles[a.XUID] = a.Role
	}
	if roles[tsMoi] != domain.TacticalRoleVictime || roles[tsAdv] != domain.TacticalRoleTueur || roles[tsAmi] != domain.TacticalRoleCoequipier {
		t.Errorf("roles = %v : l'identite vient de la base", roles)
	}
	// horloge affichee : mort a 3 000 ms de match, coup d'envoi a 2 000 -> 1 000 ; a -4 s plancher a 0.
	for _, e := range seq.Etapes {
		if e.OffsetMs == 0 && (e.InstantAfficheMs == nil || *e.InstantAfficheMs != 1000) {
			t.Errorf("etape 0 : %v, attendu 1 000", e.InstantAfficheMs)
		}
		if e.OffsetMs == -4000 && (e.InstantAfficheMs == nil || *e.InstantAfficheMs != 0) {
			t.Errorf("etape -4000 : %v, attendu 0 (plancher)", e.InstantAfficheMs)
		}
	}
	// a l'instant 0 : centre x = 20, Ami a x = 25 : 5 m, dans le rayon de 24 m.
	for _, e := range seq.Etapes {
		if e.OffsetMs == 0 && (e.DansLeRayon != 1 || e.Positionnes != 1 || len(e.DansLeRayonXUIDs) != 1 || e.DansLeRayonXUIDs[0] != tsAmi) {
			t.Errorf("etape 0 : %+v", e)
		}
	}
}

func TestSequence_TueurBot_ParSonNom(t *testing.T) {
	repo, store := corpusSequence()
	repo.journal = append(repo.journal, domain.KillDuJournal{MatchID: "m2", TimeMs: 3000, VictimXUID: tsMoi,
		VictimGamertag: "Moi", KillerGamertag: "Bot Elite"})
	for i, k := range repo.journal { // le tueur du journal de m2 / 3000 devient un bot
		if k.MatchID == "m2" && k.TimeMs == 3000 && k.KillerXUID == tsAdv {
			repo.journal = append(repo.journal[:i], repo.journal[i+1:]...)
			break
		}
	}
	store.pistes.Tracks = append(store.pistes.Tracks, ligneDePiste("", "bot elite", 0, 80, fixe(30)))
	seq, err := serviceSequence(repo, store).Sequence(context.Background(), demandeSequence())
	if err != nil {
		t.Fatal(err)
	}
	trouve := false
	for _, a := range seq.Acteurs {
		if a.Role == domain.TacticalRoleTueur && a.Gamertag == "Bot Elite" && a.XUID == "" {
			trouve = true
		}
	}
	if !trouve {
		t.Errorf("acteurs = %+v : le tueur bot se designe par son nom", seq.Acteurs)
	}
}

func TestSequence_Indisponible(t *testing.T) {
	cas := []struct {
		nom  string
		mod  func(*mockTacticalRepo, *fakePistes) port.TacticalSequenceStore
		want error
	}{
		{"artefact absent", func(_ *mockTacticalRepo, s *fakePistes) port.TacticalSequenceStore { s.pistes = nil; return s }, port.ErrReplayNotAvailable},
		{"origine non etablie", func(_ *mockTacticalRepo, s *fakePistes) port.TacticalSequenceStore { s.pistes.OriginMs = nil; return s }, port.ErrReplayNotAvailable},
		{"sans echelle de temps", func(_ *mockTacticalRepo, s *fakePistes) port.TacticalSequenceStore {
			s.pistes.FrameIntervalMS = 0
			return s
		}, port.ErrReplayNotAvailable},
		{"match d'un autre joueur", func(r *mockTacticalRepo, s *fakePistes) port.TacticalSequenceStore {
			r.ouvrables = map[string]time.Time{}
			return s
		}, port.ErrReplayNotAvailable},
		{"match sans rayon", func(r *mockTacticalRepo, s *fakePistes) port.TacticalSequenceStore {
			for i := range r.univ.Matchs {
				r.univ.Matchs[i].GameVariantName = "Inconnue"
			}
			return s
		}, port.ErrReplayNotAvailable},
		{"victime sans position a sa mort", func(_ *mockTacticalRepo, s *fakePistes) port.TacticalSequenceStore {
			s.pistes.Tracks[0] = ligneDePiste(tsMoi, "", 0, 5, fixe(1))
			return s
		}, port.ErrReplayNotAvailable},
		{"lecteur de pistes non cable", func(*mockTacticalRepo, *fakePistes) port.TacticalSequenceStore { return nil }, games.ErrCapabilityNotSupported},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			repo, store := corpusSequence()
			lecteur := c.mod(repo, store)
			_, err := serviceSequence(repo, lecteur).Sequence(context.Background(), demandeSequence())
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
}

func TestSequence_SansCapabilityDuTitre(t *testing.T) {
	repo, store := corpusSequence()
	_, matchs := corpusDetail()
	svc := sourcesCompletes(matchs).service(repo, capsPositionsSeules()).WithSequenceStore(store)
	if _, err := svc.Sequence(context.Background(), demandeSequence()); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("err = %v, want ErrCapabilityNotSupported", err)
	}
	if store.appels != 0 {
		t.Error("un titre sans artefact de rejeu ne doit rien lire")
	}
}

func TestSequence_SourcesDeLaBaseAbsentes_LaSequenceReste(t *testing.T) {
	repo, store := corpusSequence()
	repo.errMates, repo.errJournal = errors.New("boum"), errors.New("boum")
	seq, err := serviceSequence(repo, store).Sequence(context.Background(), demandeSequence())
	if err != nil {
		t.Fatalf("la base muette ne retire que les acteurs : %v", err)
	}
	if len(seq.Acteurs) != 1 || seq.Acteurs[0].Role != domain.TacticalRoleVictime {
		t.Errorf("acteurs = %+v, attendu la seule victime", seq.Acteurs)
	}
}
