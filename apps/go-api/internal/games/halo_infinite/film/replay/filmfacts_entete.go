package replay

// filmfacts_entete.go — L EN-TETE D UN FICHIER DE FAITS : ce que ses premiers octets disent, et
// la decision « decoder ou relire » qui se prend sur eux seuls.
//
// SORTI DE `filmfacts_fichier.go` le 2026-09-26 (lot J3.3 du PLAN_SUITE_AUDIT_DECODEUR_FILM) :
// ce fichier-la atteignait 477 lignes, et le lot et les deux suivants (J3.4, J3.5) inscrivent de
// nouveaux champs dans l en-tete. DEPLACEMENT PUR des fonctions d en-tete ; ce qui change dans le
// meme lot est ecrit a sa place (les revisions par consommateur de faits, le codec 2).

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// FilmFactsEntete est ce que les PREMIERS OCTETS d un fichier de faits disent, et qui suffit a
// decider « decoder ou relire ».
type FilmFactsEntete struct {
	// VersionCodec / Schema : les deux numeros lus dans le prefixe.
	VersionCodec int
	Schema       int
	// Coverage : les revisions sous lesquelles ces faits ont ete pris.
	Coverage DecoderCoverage
	// MapModule / AxisW / LayoutDetected : la CLE DE CUISSON.
	MapModule      string
	AxisW          [3]uint
	LayoutDetected bool
	// Gardes : les gardes de l appelant sous lesquelles ces faits ont ete cuits (lot J3.4).
	Gardes GardesDeCuisson
	// corps est l offset du premier octet de section.
	corps int
}

// encodeEnteteDuFichier ecrit l en-tete du FICHIER de faits : [DecoderCoverage] VERBATIM, puis la
// CLE DE CUISSON, puis les GARDES DE L APPELANT (codec 2). (`encodeEntete`, dans `filmfacts_encode.go`, est celui du BLOB des entrees.)
func encodeEnteteDuFichier(f *FilmFactsFile) *gwriter {
	entete := &gwriter{}
	encodeCouvertureDuDecodeur(entete, f.Coverage)
	entete.str(f.Facts.MapModule)
	for a := 0; a < 3; a++ {
		entete.u(uint64(f.Facts.AxisW[a]))
	}
	entete.bool8(f.Facts.LayoutDetected)
	encodeGardesDeCuisson(entete, f.Gardes)
	return entete
}

// DecodeFilmFactsEntete lit le prefixe et l en-tete, ET RIEN DE PLUS.
//
// C EST LA PORTE DE LA DECISION « decoder ou relire » : elle ne touche aucune section, donc aucun
// mega-octet de positions. Un fichier perime coute une lecture d en-tete.
func DecodeFilmFactsEntete(blob []byte) (FilmFactsEntete, error) {
	var e FilmFactsEntete
	if len(blob) < len(magieFaitsDeFilm) || string(blob[:len(magieFaitsDeFilm)]) != magieFaitsDeFilm {
		return e, fmt.Errorf("%w : magie absente", ErrFilmFactsVersion)
	}
	r := &greader{b: blob, off: len(magieFaitsDeFilm)}
	e.VersionCodec, e.Schema = int(r.u()), int(r.u())
	longueur := int(r.u())
	if r.err != nil {
		return e, fmt.Errorf("%w : prefixe illisible (%v)", ErrFilmFactsVersion, r.err)
	}
	if e.VersionCodec != VersionCodecFaits || e.Schema != SchemaDesFaits {
		return e, fmt.Errorf("%w : codec %d schema %d, ce binaire lit codec %d schema %d",
			ErrFilmFactsVersion, e.VersionCodec, e.Schema, VersionCodecFaits, SchemaDesFaits)
	}
	if longueur < 0 || longueur > len(r.b)-r.off {
		return e, fmt.Errorf("%w : en-tete annonce %d octets, %d disponibles",
			ErrFilmFactsVersion, longueur, len(r.b)-r.off)
	}
	e.corps = r.off + longueur
	e.Coverage = decodeCouvertureDuDecodeur(r)
	e.MapModule = r.str()
	for a := 0; a < 3; a++ {
		e.AxisW[a] = uint(r.u())
	}
	e.LayoutDetected = r.bool8()
	e.Gardes = decodeGardesDeCuisson(r)
	if r.err != nil {
		return e, fmt.Errorf("%w : en-tete illisible (%v)", ErrFilmFactsVersion, r.err)
	}
	return e, nil
}

// Utilisable dit si ces faits sont relisables PAR LE BINAIRE COURANT, ou rend la raison typee.
//
// TOUT OU RIEN, ET C EST LA REGLE DU LOT 4.1 (note de preparation de M4, §2.4) : version de codec,
// schema de faits, LES REVISIONS DE COUCHE et la cle de cuisson. La finesse par couche est
// l objet du lot 4.4 (`coverage_decoder.go` le dit deja) — ne pas l anticiper ici.
//
// # `build` ET `registry` NE SONT PAS COMPARES, ET C EST UNE PROPRIETE, PAS UN OUBLI
//
// Les revisions de couche sont des CONSTANTES DE COMPILATION : le binaire courant les connait sans
// ouvrir un fichier. `build` et le bloc `registry`, eux, sont des FAITS DU FILM — la cle ecrite
// dans la section 2 de `chunk_00` et l empreinte de son registre ECS. Ils ne peuvent pas avoir
// change pour un film donne, et les RECALCULER exigerait precisement ce que cette porte evite :
// ouvrir le film. Les comparer serait donc soit impossible, soit une tautologie.
//
// # LES GARDES DE L APPELANT SONT COMPAREES DEPUIS LE LOT J3.4 (RA1-1)
//
// `gardes` sont celles que la cuisson COURANTE commande ([GardesDe] sur ses options). Des faits
// cuits sans l une d elles, ou sous un autre roster, rendent [ErrFilmFactsGardes] ; un sur-ensemble
// sert (cf. `gardes_de_cuisson.go`).
func (e FilmFactsEntete) Utilisable(entry profile.MapQuantEntry, gardes GardesDeCuisson) error {
	if err := e.Frais(entry); err != nil {
		return err
	}
	return e.Gardes.couvre(gardes)
}

// Frais dit si ces faits ont ete pris par CE binaire sur CETTE entree de catalogue — codec,
// schema, revisions de couche, cle de cuisson —, SANS juger les gardes de l appelant.
//
// C EST LA PREMIERE MOITIE DE [FilmFactsEntete.Utilisable], exposee parce que la cuisson la tranche
// AVANT de connaitre ses options : les gardes se derivent des options, et les options se
// construisent sur le statborg — que les faits frais fournissent. La seconde moitie se juge une
// fois les options posees (`replaybuild.documentDeLaCuisson`).
func (e FilmFactsEntete) Frais(entry profile.MapQuantEntry) error {
	if e.VersionCodec != VersionCodecFaits || e.Schema != SchemaDesFaits {
		return ErrFilmFactsVersion
	}
	courantes := couvertureDuDecodeur(nil)
	if !memesRevisionsDeCouche(e.Coverage, *courantes) {
		return fmt.Errorf("%w : faits %s contre binaire %s", ErrFilmFactsRevisions,
			listeDesRevisions(e.Coverage), listeDesRevisions(*courantes))
	}
	return verifierCleDeCuisson(e.MapModule, e.AxisW, e.LayoutDetected, entry)
}

// encodeCouvertureDuDecodeur / decodeCouvertureDuDecodeur : [DecoderCoverage] VERBATIM.
//
// Le bloc `registry` est un POINTEUR et son absence a un sens ecrit (le registre n a pas ete lu) :
// il voyage donc derriere son temoin, jamais aplati sur des zeros.
func encodeCouvertureDuDecodeur(w *gwriter, c DecoderCoverage) {
	w.str(c.SourceRev)
	w.str(c.ProfileRev)
	w.str(c.GrammarRev)
	w.str(c.KillsourceRev)
	w.str(c.ObjectivesRev)
	w.str(c.Build)
	w.bool8(c.Registry != nil)
	if c.Registry == nil {
		return
	}
	w.str(c.Registry.Fingerprint)
	w.str(c.Registry.Status)
	w.u(uint64(c.Registry.Blocks))
	w.u(uint64(c.Registry.NamedSlots))
}

func decodeCouvertureDuDecodeur(r *greader) DecoderCoverage {
	c := DecoderCoverage{
		SourceRev:     r.str(),
		ProfileRev:    r.str(),
		GrammarRev:    r.str(),
		KillsourceRev: r.str(),
		ObjectivesRev: r.str(),
		Build:         r.str(),
	}
	if !r.bool8() {
		return c
	}
	c.Registry = &RegistryCoverage{
		Fingerprint: r.str(),
		Status:      r.str(),
		Blocks:      int(r.u()),
		NamedSlots:  int(r.u()),
	}
	return c
}

// memesRevisionsDeCouche compare LES REVISIONS DE COUCHE — cinq depuis le lot J3.3 — et elles seules.
//
// PAS `a == b` SUR LE TYPE ENTIER, et ce n est pas qu une question de perimetre : `DecoderCoverage`
// porte un POINTEUR (`Registry`), donc `==` compare des ADRESSES — deux couvertures identiques
// sorties de deux appels seraient alors toujours differentes, et la porte de fraicheur refuserait
// TOUS les faits en silence (constate le 2026-09-17 a la pose de cette porte).
func memesRevisionsDeCouche(a, b DecoderCoverage) bool {
	return a.SourceRev == b.SourceRev && a.ProfileRev == b.ProfileRev &&
		a.GrammarRev == b.GrammarRev && a.KillsourceRev == b.KillsourceRev &&
		a.ObjectivesRev == b.ObjectivesRev
}

// listeDesRevisions rend les revisions de couche d une couverture, pour un message d erreur.
func listeDesRevisions(c DecoderCoverage) string {
	return "{" + strings.Join([]string{c.SourceRev, c.ProfileRev, c.GrammarRev, c.KillsourceRev,
		c.ObjectivesRev}, " ") + "}"
}
