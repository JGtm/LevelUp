package replay

// lives_decoupe_test.go — LES PROPRIETES DE LA DECOUPE DES VIES, ET LEURS MUTATIONS (lot 1.9.13).
//
// Chaque test porte UNE regle de la grammaire — une vie finit a une mort ECRITE, a une
// apparition de corps, a une fin de manche — et chacun dit la mutation qui le fait rougir.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
)

// Les instants de la figure commune : un slot replique de 1 s a 4 s, se tait 6 s (au-dela de
// `lifeGapUS`), puis reprend de 10 s a 14 s.
const (
	trouDebutUS = 4_000_000
	trouFinUS   = 10_000_000
	finFilmUS   = 14_000_000
)

// deuxSejoursDUnCorps rend l'ECHAFAUDAGE que `buildLifeSpans` produit sur cette figure : deux
// sejours du meme slot, separes par le trou.
func deuxSejoursDUnCorps() []lifeSpan {
	return []lifeSpan{
		{slot: 100, from: 1_000_000, to: trouDebutUS, xuid: 111, cause: CauseVieCoupure,
			nomPar: NomParCreation},
		{slot: 100, from: trouFinUS, to: finFilmUS, xuid: 111, cause: CauseVieFinFilm,
			nomPar: NomParCreation},
	}
}

// faitsAvecUneMortALaFin : le joueur MEURT dans le film (a la toute fin), mais RIEN n'est ecrit
// dans le trou. C'est la figure de reference : le trou est une lacune, pas une fin de vie.
func faitsAvecUneMortALaFin(fb *fallback.Compteur) faitsQuiBornentUneVie {
	return faitsQuiBornentUneVie{
		mortsParJoueur: map[uint64][]int64{111: {finFilmUS}},
		fb:             fb,
	}
}

// TestUnTrouDeReplicationNEstPasUneFinDeVie — LA REGLE DU LOT.
//
// MUTATION JOUEE (2026-09-15) : ajouter une mort de 111 dans le trou (`{trouDebutUS + 500_000}`)
// -> la coupure redevient ECRITE, deux vies, ce test rouge. Restauree par nom.
func TestUnTrouDeReplicationNEstPasUneFinDeVie(t *testing.T) {
	fb := fallback.NouveauCompteur()
	got := decouperAuxFaitsEcrits(deuxSejoursDUnCorps(), faitsAvecUneMortALaFin(fb))
	if len(got) != 1 {
		t.Fatalf("%d vie(s), attendu 1 : le trou de 6 s n'est PAS une mort, c'est une lacune", len(got))
	}
	if got[0].from != 1_000_000 || got[0].to != finFilmUS {
		t.Errorf("vie = [%d, %d], attendu [1000000, %d] — la vie couvre le trou",
			got[0].from, got[0].to, finFilmUS)
	}
	if got[0].cause != CauseVieFinFilm {
		t.Errorf("cause = %q, attendu %q : c'est le DERNIER sejour qui dit comment la vie finit",
			got[0].cause, CauseVieFinFilm)
	}
	if n := fb.Compte(fallback.NomVieCoupeeAuTrouDeReplication); n != 0 {
		t.Errorf("repli declenche %d fois, attendu 0 : le joueur MEURT dans ce film, la lecture "+
			"est disponible", n)
	}
}

// TestUneMortEcriteDansLeTrouCoupeLaVie : l'inverse du precedent, sur la MEME figure.
//
// MUTATION JOUEE (2026-09-15) : retirer la mort du trou (`mortsParJoueur` ne garde que celle de
// la fin) -> une seule vie, ce test rouge. Restauree par nom.
func TestUneMortEcriteDansLeTrouCoupeLaVie(t *testing.T) {
	fb := fallback.NouveauCompteur()
	in := faitsAvecUneMortALaFin(fb)
	in.mortsParJoueur = map[uint64][]int64{111: {trouDebutUS + 500_000, finFilmUS}}
	got := decouperAuxFaitsEcrits(deuxSejoursDUnCorps(), in)
	if len(got) != 2 {
		t.Fatalf("%d vie(s), attendu 2 : une mort ECRITE du joueur ferme la premiere", len(got))
	}
	if got[0].cause != CauseVieMort {
		t.Errorf("cause de la premiere vie = %q, attendu %q — c'est la mort qui l'a fermee, et "+
			"l'appariement 1:1 du pont, glouton sur le seul TEMPS, n'a pas a le confirmer",
			got[0].cause, CauseVieMort)
	}
}

// TestUneApparitionEcriteCoupeLaVie : un record de CREATION dans le trou dit qu'un corps NEUF
// commence — c'est le signal d'un slot RECYCLE, et sans lui deux occupants fusionneraient.
//
// MUTATION JOUEE (2026-09-15) : dater le record AVANT le trou (`trouDebutUS - 1`) -> il n'ouvre
// plus rien, une seule vie, ce test rouge. Restauree par nom.
func TestUneApparitionEcriteCoupeLaVie(t *testing.T) {
	fb := fallback.NouveauCompteur()
	in := faitsAvecUneMortALaFin(fb)
	in.creations = []grammar.BipedCreation{
		{Slot: 100, Generation: 2, TimestampUS: trouFinUS - 200_000, HasIndex: true},
	}
	if got := decouperAuxFaitsEcrits(deuxSejoursDUnCorps(), in); len(got) != 2 {
		t.Fatalf("%d vie(s), attendu 2 : le film ECRIT une apparition de corps dans le trou",
			len(got))
	}
}

// TestUneFinDeMancheCoupeLaVie : une fin de manche fait REAPPARAITRE tout le monde sans qu'aucune
// mort soit ecrite — elle ferme donc une vie la ou le fil des morts se tait.
//
// MUTATION JOUEE (2026-09-15) : poser la frontiere hors du trou (`finFilmUS + 1`) -> une seule
// vie, ce test rouge. Restauree par nom.
func TestUneFinDeMancheCoupeLaVie(t *testing.T) {
	fb := fallback.NouveauCompteur()
	in := faitsAvecUneMortALaFin(fb)
	in.manches = []int64{trouDebutUS + 2_000_000}
	if got := decouperAuxFaitsEcrits(deuxSejoursDUnCorps(), in); len(got) != 2 {
		t.Fatalf("%d vie(s), attendu 2 : une frontiere de manche tombe dans le trou", len(got))
	}
}

// TestJoueurSansAucuneMortEcriteGardeLeSeuilEtLeCompte — LE SEUL REPLI QUI RESTE (D14).
//
// Quand le film n'ecrit AUCUNE mort du joueur, aucune des trois lectures ne peut borner ses
// vies : le seuil de trou reste la seule voie, et il se COMPTE.
//
// MUTATION JOUEE (2026-09-15) : retirer l'appel a `in.fb.Declenche` -> le compte tombe a 0 et ce
// test rouge, c'est-a-dire qu'un repli silencieux est impossible. Restauree par nom.
func TestJoueurSansAucuneMortEcriteGardeLeSeuilEtLeCompte(t *testing.T) {
	fb := fallback.NouveauCompteur()
	got := decouperAuxFaitsEcrits(deuxSejoursDUnCorps(), faitsQuiBornentUneVie{fb: fb})
	if len(got) != 2 {
		t.Fatalf("%d vie(s), attendu 2 : sans aucune mort ecrite, rien ne borne les vies de ce "+
			"joueur et le seuil reste la seule voie", len(got))
	}
	if n := fb.Compte(fallback.NomVieCoupeeAuTrouDeReplication); n != 1 {
		t.Fatalf("repli declenche %d fois, attendu 1 : un repli anonyme est interdit (D14 a)", n)
	}
}

// TestDecoupeSurEntreeTronquee : aucune borne de boucle ne deborde sur une entree amputee.
func TestDecoupeSurEntreeTronquee(t *testing.T) {
	fb := fallback.NouveauCompteur()
	if got := decouperAuxFaitsEcrits(nil, faitsAvecUneMortALaFin(fb)); len(got) != 0 {
		t.Errorf("aucune vie en entree -> %d en sortie, attendu 0", len(got))
	}
	un := deuxSejoursDUnCorps()[:1]
	if got := decouperAuxFaitsEcrits(un, faitsAvecUneMortALaFin(fb)); len(got) != 1 {
		t.Errorf("une seule vie en entree -> %d en sortie, attendu 1 (rien a fusionner)", len(got))
	}
	if i := vieDuPoint(nil, -1, 1_000_000); i != -1 {
		t.Errorf("vieDuPoint sur une liste VIDE = %d, attendu -1 (aucune vie a designer)", i)
	}
	if i := vieDuPoint(deuxSejoursDUnCorps(), 0, finFilmUS+1_000_000); i != 0 {
		t.Errorf("vieDuPoint au-dela de la derniere vie = %d, attendu 0 (la vie courante est "+
			"conservee, jamais une vie neuve ouverte sur une borne qu'on n'a pas su lire)", i)
	}
}

// positionsAvecTrou : la figure commune, en positions decodees — un point toutes les 500 ms.
func positionsAvecTrou() []grammar.BipedPosition {
	var out []grammar.BipedPosition
	for t := uint64(1_000_000); t <= trouDebutUS; t += 500_000 {
		out = append(out, posAt(100, t, 1, 1, 0))
	}
	for t := uint64(trouFinUS); t <= finFilmUS; t += 500_000 {
		out = append(out, posAt(100, t, 2, 2, 0))
	}
	return out
}

// TestUneLacuneSePublieSurLaPISTE ET DANS LA COUVERTURE : la piste reste UNE, le silence est
// compte, et le point qui rouvre la piste porte sa duree — sans quoi le client tracerait un
// segment de 6 s a travers un trou ou le film ne dit RIEN.
//
// MUTATION JOUEE (2026-09-15) : passer les DEUX sejours en `vies` (c'est-a-dire remettre
// `lifeGapUS` en decideur) -> deux pistes, `gaps` a 0, ce test rouge sur les trois assertions.
// Restauree par nom.
func TestUneLacuneSePublieSurLaPisteEtDansLaCouverture(t *testing.T) {
	tracks, cov := decimateTracks(positionsAvecTrou(), decoupeDesTraces{
		origin: 1_000_000, step: 100_000, minPoints: 1,
		vies: []lifeSpan{{slot: 100, from: 1_000_000, to: finFilmUS, xuid: 111}},
		fb:   fallback.NouveauCompteur(),
	})
	if len(tracks) != 1 {
		t.Fatalf("%d piste(s), attendu 1 : le trou est une LACUNE de la meme vie", len(tracks))
	}
	if cov.Gaps != 1 || cov.GapMS != 6_000 {
		t.Errorf("couverture : gaps=%d gapMs=%d, attendu 1 et 6000", cov.Gaps, cov.GapMS)
	}
	var portes int
	for _, p := range tracks[0].Points {
		if p.G != 0 {
			portes++
			if p.G != 6_000 {
				t.Errorf("point t=%d : g=%d, attendu 6000 ms", p.T, p.G)
			}
			if p.T != 90 {
				t.Errorf("la lacune est portee par le point t=%d, attendu t=90 (celui qui ROUVRE "+
					"la piste, pas celui qui la ferme)", p.T)
			}
		}
	}
	if portes != 1 {
		t.Errorf("%d point(s) portent une lacune, attendu 1", portes)
	}
}

// TestSansVieLueLaPublicationSeRabatSurLeSeuilEtLeCompte : un film dont le registre ne nomme
// AUCUNE vie (table des joueurs vide) n'a rien a suivre. Le repli est alors le seuil de trou, et
// il se COMPTE — un artefact muet sur sa propre provenance est ce que D14 (c) interdit.
func TestSansVieLueLaPublicationSeRabatSurLeSeuilEtLeCompte(t *testing.T) {
	fb := fallback.NouveauCompteur()
	tracks, cov := decimateTracks(positionsAvecTrou(), decoupeDesTraces{
		origin: 1_000_000, step: 100_000, minPoints: 1, fb: fb,
	})
	if len(tracks) != 2 {
		t.Fatalf("%d piste(s), attendu 2 : sans vie lue, le seuil de trou reprend la main", len(tracks))
	}
	if cov.Gaps != 0 {
		t.Errorf("gaps=%d, attendu 0 : le trou a COUPE, il n'est pas une lacune", cov.Gaps)
	}
	if n := fb.Compte(fallback.NomVieCoupeeAuTrouDeReplication); n != 1 {
		t.Fatalf("repli declenche %d fois, attendu 1", n)
	}
}
