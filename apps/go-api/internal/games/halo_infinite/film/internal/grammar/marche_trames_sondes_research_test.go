//go:build research

package grammar

// marche_trames_sondes_research_test.go — LES PORTES DES SONDES DE RECHERCHE SUR LA MARCHE DES
// TRAMES.
//
// La production marche les trames par [FilmContext.Trames] ; les sondes de la campagne de
// grammaire marchent, elles, UN paquet depuis un debut de leur choix (localisateur alternatif,
// essai depuis un candidat) et lisent le debut d une liste sous sa forme booleenne. Ces portes
// passent par les MEMES fonctions que la production : la marche par rangs ([lireTrameParRangs]),
// le detail de la carte ([detaillerLaLecture]), la localisation des listes ([localiserLaListe],
// [debutParFermetureRangee]).

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"

// largeurTagDeGeneration est la largeur du tag de generation de l en-tete d un record
// ([readRecordID] : `R(2)` en bits 30-31) ; les sondes l emploient pour relire un en-tete.
const largeurTagDeGeneration = 2

// marcherParRangs marche UN paquet depuis `debut` sur le monde `w`, en remplit le detail `d` et rend
// ses records, ses rangs lus et le verdict de sa vue C.
func (md *marcheDetaillee) marcherParRangs(pay []byte, w *World, debut int,
	d *PaquetDeCarte) ([]FrameRecord, int, LectureVueC) {
	avant := compteDesAnticipations(md.cfg.Obs)
	br := LecteurSur(pay)
	br.poserCadre(md.cfg)
	var l lectureDeTrame
	lireTrameParRangs(br, pay, w, md.cfg, departDeTrame{bit: debut}, &l)
	detaillerLaLecture(&l, pay, md.cfg, d)
	d.Anticipations = compteDesAnticipations(md.cfg.Obs) - avant
	return l.recs, l.rangs, l.verdict
}

// debutDeLaListe rend le debut de la marche d un paquet a evenements ([localiserLaListe]) et dit
// s il commence a un record NEW que le localisateur sautait ; -1 pour une liste non localisee.
func debutDeLaListe(pay []byte, w *World, cfg FrameConfig) (int, bool) {
	debut, comment := localiserLaListe(pay, w, cfg)
	return debut, comment != lecture.DebutParSignature && comment != lecture.DebutNonLocalise
}

// debutDeLaListeSous rend le debut de la marche de cuisson d un paquet a evenements sous la grammaire de
// vue A `g` ([debutDeLaVueBDeCuisson] : la fin de la vue A quand elle decide, sinon [localiserLaListe]),
// sous la forme de [debutDeLaListe].
func debutDeLaListeSous(g grammaireDeLaVueA) func([]byte, *World, FrameConfig) (int, bool) {
	return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
		a := lireLaVueA(pay, 1, cfg.Profil, g)
		debut, comment := debutDeLaVueBDeCuisson(pay, &a, g.classe, w, cfg)
		return debut, comment != lecture.DebutParSignature && comment != lecture.DebutNonLocalise
	}
}

// debutParFermeture rend le debut par fermeture d une liste ([debutParFermetureRangee]) et dit
// s il a ete trouve, a l un ou l autre rang.
func debutParFermeture(pay []byte, candidats []int, w *World, cfg FrameConfig) (int, bool) {
	debut, rang := debutParFermetureRangee(pay, candidats, w, cfg)
	return debut, rang != lecture.DebutNonLocalise
}
