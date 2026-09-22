package main

// cmd_backfill_killsource_etat.go — LA PASSE DIT OU ELLE EN EST, A CHAQUE INSTANT (lot 5.24.3).
//
// # CE QUE LE SILENCE A COUTE, ET QUI L A PAYE
//
// Passe du 2026-09-21 : 1 598 films en 4 h 14, puis une passe credit de plus de 22 heures. Les
// deux ont tourne SANS QU ON PUISSE REPONDRE A « il en est ou, il reste combien ». La passe des
// films journalisait chaque film (debut/fin/duree), ce qui est lisible quand on regarde le
// journal defiler et inutilisable quand on revient trois heures plus tard ; la passe credit ne
// disait rien du tout (corrige au lot 5.12). Impossible de distinguer « lente » de « bloquee »,
// impossible d estimer une fin, impossible de decider d interrompre.
//
// # TROIS PIECES, ET ELLES REPONDENT A TROIS QUESTIONS DIFFERENTES
//
//	LE FICHIER D ETAT   reecrit apres CHAQUE film. Il repond a « ou en est-elle ? » depuis un
//	                    AUTRE terminal, sans ouvrir aucune base — ce qui est la seule facon de
//	                    demander quoi que ce soit a un process qui tient le shared en ecriture
//	                    (un seul writer, ADR 0013).
//	LA LIGNE DE JOURNAL tous les 25 films OU toutes les 60 s, le premier des deux. Elle repond a
//	                    « avance-t-elle ? » pour qui regarde le terminal de la passe.
//	`--status`          lit le fichier et l affiche en clair, une fois. C est ce qu on tape.
//
// # L ETA EST PONDERE PAR LA TAILLE, ET CE N EST PAS UN RAFFINEMENT
//
// Les films partent du MOINS CHER au plus cher (la selection les trie). Un ETA lineaire en
// NOMBRE de films serait donc faux par construction, et faux dans le mauvais sens : il
// annoncerait une fin proche juste avant la queue de gros films qui coute le plus. L ETA d ici
// compte des CHUNKS — la somme des chunks restants multipliee par le cout par chunk MESURE
// depuis le debut de la passe, divisee par le nombre d ouvriers. Le cout par chunk se mesure en
// marchant : il n est pas pris dans une constante qui vieillirait.
//
// ⚠ LE FICHIER N EST PAS UNE SOURCE DE VERITE POUR LA REPRISE. Ce qui decide de ce qui reste a
// faire est `decoder_rev` EN BASE, relu au demarrage de chaque passe. Le fichier d etat ne sert
// qu a REGARDER : le supprimer ne perd rien d autre qu un affichage.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/sync/killcollector"
)

// Les deux cadences de la ligne de progression.
//
// LES DEUX, ET PAS L UNE OU L AUTRE : le pas par compteur (25 films) dit le debit quand la passe
// avance vite, le pas par horloge (60 s) prouve qu elle vit quand elle est sur un gros film qui
// coute quarante secondes. Une seule des deux laisserait un des deux regimes muet — et c est
// precisement dans le regime muet qu on se demande si le programme est bloque.
const (
	pasDeProgressionFilms = 25
	delaiDeProgressionMin = 60 * time.Second
	filmsDuDebitGlissant  = 20
	permissionFichierEtat = 0o644
	permissionDossierEtat = 0o755
	phaseFilms            = "films"
	phaseCredit           = "credit"
	phaseTerminee         = "terminee"
	resultatFilmErreur    = "erreur"
)

// filmEnCours : un film qu un ouvrier decode en ce moment.
type filmEnCours struct {
	MatchID string    `json:"match_id"`
	Chunks  int       `json:"chunks"`
	Depuis  time.Time `json:"depuis"`
	DepuisS float64   `json:"depuis_s"`
}

// filmFini : le dernier film termine.
type filmFini struct {
	MatchID  string  `json:"match_id"`
	Chunks   int     `json:"chunks"`
	DureeS   float64 `json:"duree_s"`
	Resultat string  `json:"resultat"`
}

// etatDesFilms : ce que la passe des films a fait et ce qu il lui reste.
type etatDesFilms struct {
	TotalRegistre  int           `json:"total_registre"`
	AFaire         int           `json:"a_faire"`
	DejaAJour      int           `json:"deja_a_jour"`
	SansFilmEnCach int           `json:"sans_film_en_cache"`
	ChunksAFaire   int           `json:"chunks_a_faire"`
	ChunksFaits    int           `json:"chunks_faits"`
	Traites        int           `json:"traites"`
	Ecrits         int           `json:"ecrits"`
	Morts          int           `json:"morts"`
	SansFilm       int           `json:"sans_film"`
	SansKillFeed   int           `json:"sans_kill_feed"`
	CleInconnue    int           `json:"cle_inconnue"`
	AbandonsDelai  int           `json:"abandons_delai"`
	Erreurs        int           `json:"erreurs"`
	Ouvriers       int           `json:"ouvriers"`
	EnCours        []filmEnCours `json:"en_cours"`
	DernierFini    *filmFini     `json:"dernier_fini,omitempty"`
	DebitParMin    float64       `json:"debit_films_par_min"`
	CoutParChunkS  float64       `json:"cout_par_chunk_s"`
	RestantS       float64       `json:"restant_s"`
	FinEstimeeA    *time.Time    `json:"fin_estimee_a,omitempty"`
}

// etatDuCredit : la progression de la passe credit.
type etatDuCredit struct {
	AExaminer int     `json:"a_examiner"`
	Examines  int     `json:"examines"`
	RestantS  float64 `json:"restant_s"`
}

// etatDeLaPasse : LE document. Sa forme EST le contrat de `--status`.
type etatDeLaPasse struct {
	Commande      string       `json:"commande"`
	Titre         string       `json:"titre"`
	Phase         string       `json:"phase"`
	DemarreeA     time.Time    `json:"demarree_a"`
	MiseAJourA    time.Time    `json:"mise_a_jour_a"`
	EcouleeS      float64      `json:"ecoulee_s"`
	PID           int          `json:"pid"`
	RevisionMorts string       `json:"revision_journal_des_morts"`
	RevisionIsol  string       `json:"revision_isolement"`
	Force         bool         `json:"force"`
	Films         etatDesFilms `json:"films"`
	Credit        etatDuCredit `json:"credit"`
	// RepriseDe : le nombre de films que la passe a SAUTES parce qu ils etaient deja a jour.
	// C est la trace de la reprise (lot 5.24.4) — et elle se lit en base, pas ici.
	RepriseDe int `json:"repris_de"`
	// Interrompue : renseignee quand la passe s est arretee sur un signal.
	Interrompue string `json:"interrompue,omitempty"`
}

// suiviDeLaPasse : l ECRIVAIN du fichier d etat et de la ligne de progression.
//
// IL PORTE SON VERROU, et il en a besoin : depuis le lot 5.24.2 les ouvriers appellent
// `FilmDemarre`/`FilmFini` depuis N goroutines. Toute lecture et toute ecriture de l etat passe
// par lui.
type suiviDeLaPasse struct {
	mu      sync.Mutex
	chemin  string
	etat    etatDeLaPasse
	chunks  map[string]int
	encours map[string]time.Time
	// finis : les instants de fin des derniers films, pour le debit glissant.
	finis []time.Time
	// derniereLigne : quand la ligne de progression a ete journalisee pour la derniere fois.
	derniereLigne time.Time
	// depuisLaLigne : films finis depuis cette ligne.
	depuisLaLigne int
	// secondesDecodees : la somme des durees de film, pour le cout par chunk MESURE.
	secondesDecodees float64
}

var _ killcollector.ObservateurDePasse = (*suiviDeLaPasse)(nil)

// nouveauSuivi construit le suivi et ecrit le premier etat — celui d une passe qui demarre.
//
// LE PREMIER ETAT EST ECRIT AVANT LE PREMIER FILM, et c est voulu : `--status` tape dans la
// seconde qui suit le lancement doit repondre « elle demarre », pas « aucun fichier ».
func nouveauSuivi(
	chemin, titre string, candidats []filmCandidat, bilan bilanDeSelection, o killsourceOptions,
) *suiviDeLaPasse {
	s := &suiviDeLaPasse{
		chemin:  chemin,
		chunks:  make(map[string]int, len(candidats)),
		encours: map[string]time.Time{},
		etat: etatDeLaPasse{
			Commande:      "backfill-killsource",
			Titre:         titre,
			Phase:         phaseFilms,
			DemarreeA:     time.Now(),
			PID:           os.Getpid(),
			RevisionMorts: decfilm.Rev,
			RevisionIsol:  killcollector.IsolationDecoderRev,
			Force:         o.force,
			RepriseDe:     bilan.DejaAJour,
			Films: etatDesFilms{
				TotalRegistre:  bilan.TotalRegistre,
				AFaire:         len(candidats),
				DejaAJour:      bilan.DejaAJour,
				SansFilmEnCach: bilan.SansFilmEnCache,
				Ouvriers:       o.workers,
			},
		},
	}
	for _, c := range candidats {
		s.chunks[c.matchID] = c.chunks
		s.etat.Films.ChunksAFaire += c.chunks
	}
	s.derniereLigne = s.etat.DemarreeA
	s.ecrire()
	return s
}

// FilmDemarre note qu un ouvrier a pris ce film.
func (s *suiviDeLaPasse) FilmDemarre(matchID string, debut time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.encours[matchID] = debut
	s.ecrire()
}

// FilmFini comptabilise l issue d un film, reecrit l etat et journalise si c est un jalon.
func (s *suiviDeLaPasse) FilmFini(ev killcollector.EvenementDeFilm) {
	s.mu.Lock()
	delete(s.encours, ev.MatchID)
	s.comptabiliser(ev)
	s.ecrire()
	jalon := s.jalonAtteint()
	ligne := s.chiffresDeProgression()
	s.mu.Unlock()

	if jalon {
		slog.InfoContext(context.Background(), "killsource: progression de la passe des films", ligne...)
	}
}

// comptabiliser : l issue d UN film dans l etat. Appelee verrou tenu.
func (s *suiviDeLaPasse) comptabiliser(ev killcollector.EvenementDeFilm) {
	f := &s.etat.Films
	f.Traites++
	f.ChunksFaits += s.chunks[ev.MatchID]
	s.secondesDecodees += ev.Duree.Seconds()
	s.depuisLaLigne++
	s.finis = append(s.finis, time.Now())
	if len(s.finis) > filmsDuDebitGlissant {
		s.finis = s.finis[len(s.finis)-filmsDuDebitGlissant:]
	}
	resultat := string(ev.Outcome)
	switch {
	case ev.Err != nil:
		f.Erreurs++
		resultat = resultatFilmErreur
	case ev.Outcome == killcollector.OutcomeWritten:
		f.Ecrits++
		f.Morts += ev.Morts
	case ev.Outcome == killcollector.OutcomeNoFilm:
		f.SansFilm++
	case ev.Outcome == killcollector.OutcomeNoKillFeed:
		f.SansKillFeed++
	case ev.Outcome == killcollector.OutcomeUnknownKey:
		f.CleInconnue++
	case ev.Outcome == killcollector.OutcomeTimeout:
		f.AbandonsDelai++
	}
	f.DernierFini = &filmFini{MatchID: ev.MatchID, Chunks: s.chunks[ev.MatchID],
		DureeS: arrondi(ev.Duree.Seconds()), Resultat: resultat}
}

// jalonAtteint : ce film declenche-t-il une ligne de journal ? Appelee verrou tenu.
//
// Le compteur ET l horloge, le premier des deux — et le compteur REPART a chaque ligne, quel que
// soit ce qui l a declenchee : sans cela une passe lente produirait une ligne par minute PUIS une
// rafale au 25e film.
func (s *suiviDeLaPasse) jalonAtteint() bool {
	if s.depuisLaLigne >= pasDeProgressionFilms ||
		time.Since(s.derniereLigne) >= delaiDeProgressionMin {
		s.derniereLigne = time.Now()
		s.depuisLaLigne = 0
		return true
	}
	return false
}

// PhaseCredit bascule l etat sur la passe credit.
func (s *suiviDeLaPasse) PhaseCredit(aExaminer int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.etat.Phase = phaseCredit
	s.etat.Credit.AExaminer = aExaminer
	s.ecrire()
}

// ProgressionCredit note l avancee de la passe credit (elle journalise deja d elle-meme).
func (s *suiviDeLaPasse) ProgressionCredit(examines int, restant time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.etat.Credit.Examines = examines
	s.etat.Credit.RestantS = arrondi(restant.Seconds())
	s.ecrire()
}

// Terminee ferme l etat. `cause` vide = fin normale.
func (s *suiviDeLaPasse) Terminee(cause string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.etat.Phase = phaseTerminee
	s.etat.Interrompue = cause
	s.etat.Films.EnCours = nil
	s.ecrire()
}

// chiffresDeProgression : les arguments `slog` de la ligne de progression. Verrou tenu.
//
// CE SONT LES MEMES CHIFFRES QUE LE FICHIER, et c est le point : deux sources qui diraient deux
// choses obligeraient a choisir laquelle croire.
func (s *suiviDeLaPasse) chiffresDeProgression() []any {
	f := s.etat.Films
	return []any{
		"traites", f.Traites, "a_faire", f.AFaire,
		"chunks_faits", f.ChunksFaits, "chunks_a_faire", f.ChunksAFaire,
		"ecrits", f.Ecrits, "morts", f.Morts, "sans_film", f.SansFilm,
		"sans_kill_feed", f.SansKillFeed, "cle_inconnue", f.CleInconnue,
		"erreurs", f.Erreurs, "abandons_delai", f.AbandonsDelai,
		"films_par_min", f.DebitParMin, "cout_par_chunk_s", f.CoutParChunkS,
		"restant", time.Duration(f.RestantS * float64(time.Second)).Round(time.Second),
		"etat", s.chemin,
	}
}

// ecrire recalcule les quantites derivees et reecrit le fichier. Appelee verrou tenu.
//
// ⚠ ECRITURE ATOMIQUE (fichier temporaire + `os.Rename`) : `--status` peut lire a l instant ou
// la passe ecrit, et un JSON tronque serait pris pour un etat corrompu alors que la passe va
// parfaitement bien.
//
// SON ECHEC NE FAIT PAS TOMBER LA PASSE, et il n est pas avale non plus (CLAUDE.md n 3) : un
// disque plein ne doit pas perdre quatre heures de decodage, mais il doit se voir.
func (s *suiviDeLaPasse) ecrire() {
	s.recalculer()
	if s.chemin == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.chemin), permissionDossierEtat); err != nil {
		slog.Warn("killsource: dossier du fichier d etat inaccessible — la passe continue SANS etat",
			"chemin", s.chemin, "err", err)
		return
	}
	raw, err := json.MarshalIndent(s.etat, "", "  ")
	if err != nil {
		slog.Warn("killsource: etat de passe non serialisable — la passe continue SANS etat",
			"err", err)
		return
	}
	tmp := s.chemin + ".tmp"
	if err := os.WriteFile(tmp, raw, permissionFichierEtat); err != nil {
		slog.Warn("killsource: fichier d etat non ecrit — la passe continue",
			"chemin", tmp, "err", err)
		return
	}
	if err := os.Rename(tmp, s.chemin); err != nil {
		slog.Warn("killsource: fichier d etat non publie — la passe continue",
			"chemin", s.chemin, "err", err)
	}
}

// recalculer : debit glissant, cout par chunk mesure, ETA pondere. Appelee verrou tenu.
func (s *suiviDeLaPasse) recalculer() {
	maintenant := time.Now()
	s.etat.MiseAJourA = maintenant
	s.etat.EcouleeS = arrondi(maintenant.Sub(s.etat.DemarreeA).Seconds())

	f := &s.etat.Films
	f.EnCours = f.EnCours[:0]
	for id, debut := range s.encours {
		f.EnCours = append(f.EnCours, filmEnCours{MatchID: id, Chunks: s.chunks[id],
			Depuis: debut, DepuisS: arrondi(maintenant.Sub(debut).Seconds())})
	}
	// ORDRE STABLE : une map se parcourt au hasard, et un fichier dont les lignes dansent d une
	// ecriture a l autre se lit mal (et se `diff` encore plus mal).
	sort.Slice(f.EnCours, func(i, j int) bool { return f.EnCours[i].MatchID < f.EnCours[j].MatchID })

	f.DebitParMin = arrondi(debitGlissant(s.finis))
	if f.ChunksFaits > 0 && s.secondesDecodees > 0 {
		f.CoutParChunkS = arrondi(s.secondesDecodees / float64(f.ChunksFaits))
	}
	f.RestantS = arrondi(resteEstime(f))
	if f.RestantS > 0 {
		fin := maintenant.Add(time.Duration(f.RestantS * float64(time.Second)))
		f.FinEstimeeA = &fin
	} else {
		f.FinEstimeeA = nil
	}
}

// debitGlissant : films par minute sur les derniers films finis. 0 s il n y a rien a mesurer.
func debitGlissant(finis []time.Time) float64 {
	if len(finis) < 2 {
		return 0
	}
	fenetre := finis[len(finis)-1].Sub(finis[0]).Seconds()
	if fenetre <= 0 {
		return 0
	}
	return float64(len(finis)-1) / fenetre * 60
}

// resteEstime : le temps restant, EN CHUNKS et divise par les ouvriers.
//
// EN CHUNKS PARCE QUE LES GROS FILMS SONT A LA FIN. Un reste compte en NOMBRE de films
// annoncerait une fin proche juste avant la queue la plus chere de la passe — c est-a-dire qu il
// mentirait exactement au moment ou on le consulte le plus.
//
// DIVISE PAR LES OUVRIERS parce que le cout par chunk se mesure en temps DE FILM, pas en temps
// d horloge : N ouvriers consomment N chunks a la fois.
func resteEstime(f *etatDesFilms) float64 {
	restants := f.ChunksAFaire - f.ChunksFaits
	if restants <= 0 || f.CoutParChunkS <= 0 {
		return 0
	}
	ouvriers := f.Ouvriers
	if ouvriers < 1 {
		ouvriers = 1
	}
	return float64(restants) * f.CoutParChunkS / float64(ouvriers)
}

// arrondi : deux decimales. Un fichier d etat n a pas besoin de nanosecondes, et il se relit.
func arrondi(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// bilanInitial : le meme resume que `--dry-run` affiche, une seule fois.
//
// IL EST ICI ET PAS DANS `afficherPlan` parce qu il parle du TEMPS, que le plan ne connait pas :
// combien de chunks, a quel cout connu, donc pour combien de temps avec N ouvriers.
func bilanInitial(candidats []filmCandidat, totalRegistre, dejaAJour, ouvriers int) string {
	chunks := 0
	gros := 0
	for _, c := range candidats {
		chunks += c.chunks
		if c.chunks > 50 {
			gros++
		}
	}
	// coutParChunkConnu : la mesure 5.24.1 sur l echantillon (0,05 a 0,75 s/chunk selon la
	// taille ; 0,20 est la moyenne ponderee du parc). Elle ne sert QU AU bilan initial — des le
	// premier film fini, l ETA emploie le cout MESURE de la passe en cours.
	const coutParChunkConnu = 0.20
	reste := time.Duration(float64(chunks) * coutParChunkConnu / float64(max(ouvriers, 1)) * float64(time.Second))
	return fmt.Sprintf(
		"bilan : %d matchs au registre, %d deja a jour (sautes), %d films a decoder — "+
			"%d chunks au total dont %d film(s) au-dela de 50 chunks (passes en dernier) ; "+
			"a %.2f s/chunk et %d ouvrier(s), environ %s",
		totalRegistre, dejaAJour, len(candidats), chunks, gros, coutParChunkConnu, ouvriers,
		reste.Round(time.Second))
}
