package grammar

// mpp_declare.go — LE DECOUPAGE DU BLOC MPP QU UN FILM DECLARE, ET SA RESOLUTION PAR FILM
// (ADR 0037 IR-7).
//
// La regle est dans `profile` ([profile.MPPPourTailleDeclaree]) : la taille d etat de creation
// qu un record d image-cle declare (`n1`) designe le decoupage du bloc
// `object-multiplayer-properties`. Ce fichier va chercher cette taille dans le film :
// - la premiere image-cle qui porte un record d un archetype de la cle ;
// - le mot `n1` de chacun de ces records, lu juste apres l en-tete par entite, avant l etat de
//   creation ;
// - l accord de tous.
//
// Les records viennent de la phase des images-cles ([FilmContext.ImagesCles]) : son ancre donne
// le debut et l archetype de chaque record, et `n1` se lit a une position fixe derriere eux, sans
// dependre de la traversee de son corps. Le contexte ne memorise rien de plus : la marche d ancres
// d un payload l est deja ([FilmContext.MarcheDImageCle]), et une image-cle suffit.
//
// LA RESOLUTION NE PASSE PAS PAR L EN-TETE DE LA MARCHE ([FilmContext.EnTete]). La preuve des
// ancres d image-cle lit l en-tete : y poser la declaration changerait ce que la preuve prouve, donc
// les ancres de toutes les lectures du film. La cuisson et killsource posent la resolution sur LEUR
// contexte, apres leur profil (`replay.poserLeDecoupageMPPDuFilm`, `killsource.poserLeProfil`).

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// imagesClesDeLaDeclaration : le nombre maximal d images-cles examinees pour trouver un record de
// la cle. La premiere image-cle d un film porte ses joueurs (archetype 35, de la cle).
const imagesClesDeLaDeclaration = 4

// DeclarationMPP est ce qu un film declare du decoupage de son bloc MPP.
type DeclarationMPP struct {
	// Widths et Provenance : le decoupage declare ([profile.MPPPourTailleDeclaree]).
	Widths     profile.MPPWidths
	Provenance profile.ProvenanceMPP
	// Records : les records de la cle lus dans l image-cle qui a decide. Discordants : ceux dont la
	// taille designe un autre decoupage que le premier, ou aucun.
	Records, Discordants int
}

// Declaree dit si le film declare un decoupage, sans discordance.
func (d DeclarationMPP) Declaree() bool {
	return d.Records > 0 && d.Discordants == 0 && d.Widths.Valid()
}

// DeclarationMPP rend le decoupage du bloc MPP que ce film declare : celui de la premiere de ses
// images-cles (parmi les [imagesClesDeLaDeclaration] premieres de la phase des images-cles,
// [FilmContext.ImagesCles]) qui porte un record de la cle. Zero quand aucune n en porte, ou que
// le film n a pas de registre.
func (c *FilmContext) DeclarationMPP() DeclarationMPP {
	cadre := c.ProfilDeBalayage().Cadre
	vues := 0
	for p, err := range c.ImagesCles() {
		if err != nil {
			return DeclarationMPP{}
		}
		if d := declarationDuPaquet(p, cadre); d.Records > 0 {
			return d
		}
		if vues++; vues >= imagesClesDeLaDeclaration {
			break
		}
	}
	return DeclarationMPP{}
}

// declarationDuPaquet lit la taille que declare chaque record de la cle d un paquet d image-cle,
// record a l ancre ELUE excepte (son debut n est pas atteint de proche en proche).
func declarationDuPaquet(p *lecture.Paquet, cadre profile.KeyframeProfile) DeclarationMPP {
	var d DeclarationMPP
	total := len(p.Payload) * 8
	for _, r := range p.Records {
		if r.Liaison == lecture.LiaisonImageCleElue || r.TI < 0 {
			continue
		}
		ti := uint32(r.TI)
		if !profile.CleDuDecoupageMPP(ti) {
			continue
		}
		pos := int(r.Debut) + cadre.EnTeteBits
		if pos+cadre.MotDeTailleBits > total {
			continue
		}
		n1 := int32(source.BitsBourres(p.Payload, pos, cadre.MotDeTailleBits)) //nolint:gosec // mot de 32 bits, compare signe comme le jeu
		w, prov, ok := profile.MPPPourTailleDeclaree(ti, n1)
		d.Records++
		switch {
		case !ok:
			d.Discordants++
		case d.Provenance == profile.MPPNonDeclare:
			d.Widths, d.Provenance = w, prov
		case w != d.Widths:
			d.Discordants++
		}
	}
	return d
}

// ResolutionMPP rend le decoupage du bloc MPP de ce film : celui de sa version de format quand
// elle le porte (format 27), sinon, pour un format connu sans largeur relue (20, 21, 24, 25),
// celui que le film declare sans discordance. Un format inconnu, ou une declaration absente ou
// discordante, rend une resolution qui ne decide pas : l appelant garde son chemin et le compte.
func (c *FilmContext) ResolutionMPP() ResolutionMPP {
	res := MPPWidthsForFilm(c.Film())
	if res.Decide() || res.FormatInconnu {
		return res
	}
	res.Declaration = c.DeclarationMPP()
	if res.Declaration.Declaree() {
		res.Widths, res.Provenance = res.Declaration.Widths, res.Declaration.Provenance
	}
	return res
}
