package replay

// filmfacts_fichier.go — LE FICHIER DE FAITS D UN FILM : CINQ SECTIONS ET UN EN-TETE
// (lot 4.1.1-b du PLAN_DECODEUR_FILM, 2026-09-17).
//
// # CE QUE CE FICHIER EST
//
// Tout ce qu une cuisson lit DANS un film, sous une forme qui permet de rejouer l artefact SANS
// redecoder. Il se range a `data/cache/film_facts/{slug}/{short8}.filmfacts.bin`
// (`PathResolver.FilmFactsPath`) et il ne remplace RIEN : le film reste la source, les faits sont
// une projection datee par les revisions sous lesquelles elle a ete prise.
//
// # L EN-TETE, ET POURQUOI IL EST A LONGUEUR PREFIXEE
//
//	prefixe fixe        magie + version de CODEC + numero de SCHEMA DE FAITS + longueur d en-tete
//	en-tete             [DecoderCoverage] VERBATIM, puis la CLE DE CUISSON
//	sections            [id][longueur][octets], cinq fois
//
// LA DECISION « DECODER OU RELIRE » SE PREND SUR LES PREMIERS OCTETS, jamais sur le mega-octet de
// positions : [DecodeFilmFactsEntete] lit le prefixe et l en-tete et s arrete la. Un fichier
// perime coute donc une lecture d en-tete, pas une decompression.
//
// LES REVISIONS NE SONT PAS RECOPIEES : l en-tete porte [DecoderCoverage] tel quel — le MEME type
// que `coverage.decoder` de l artefact, avec sa regle « chaine vide et bloc present sur un build
// inconnu » (V15 (15)). Ecrire un second bloc de revisions aurait ete la TROISIEME copie, et
// l invariant gratuit qui en sort est testable : l en-tete relu == le `coverage.decoder` de
// l artefact produit, champ pour champ.
//
// LA CLE DE CUISSON EST VERIFIEE DANS LES DEUX SENS, par des erreurs TYPEES
// ([ErrFilmFactsCarte], [ErrFilmFactsDecoupage]) — c est [verifierCleDeCuisson], la MEME fonction
// que le blob des entrees emploie. Sans elle, une entree de catalogue corrigee rendrait des
// coordonnees FAUSSES et pas approximatives : les positions sont des quanta.
//
// DEUX NUMEROS, ET ILS NE MESURENT PAS LA MEME CHOSE : [VersionCodecFaits] est le CONTENEUR (la
// forme du prefixe, de l en-tete et du cadre des sections) ; [SchemaDesFaits] est la CHARGE (quelles
// sections existent et ce qu elles portent). Un conteneur stable qui gagne une section ne monte
// que le second ; changer le cadre monte le premier.
//
// LES SECTIONS SONT A LONGUEUR PREFIXEE pour qu un lecteur SAUTE une section inconnue. La regle de
// fraicheur du lot 4.1 reste pourtant TOUT OU RIEN (note de preparation de M4, §2.4) : faits
// utilisables si et seulement si version de codec, schema de faits, LES QUATRE revisions et la cle
// de cuisson sont egaux a ce que le binaire courant resout. La finesse par couche est l objet du
// lot 4.4 — ne pas l anticiper.
//
// # LES CINQ SECTIONS, ET CE QUI MANQUAIT A CHACUNE
//
//	1  entrees       [FilmFacts] (le blob delta-code des entrees) PLUS les quatre canaux
//	                 GARDES PAR L APPELANT que ce blob ne porte pas — `FlagMarks`, `ZoneReads`,
//	                 `ZoneScanned`, `BombReads`. Cf. `encodeGardesDeMode`.
//	2  identite      [profile.FilmIdentity], posee par `BuildFromFilm`. SANS ELLE
//	                 `coverage.decoder.build` sort vide et le bloc `registry` est absent — c est
//	                 exactement l ambiguite que D-7 interdit.
//	3  replis        le rapport du compteur de replis AU SORTIR DU BALAYAGE, et de lui seul.
//	                 MESURE DU 2026-09-17 : sur les 18 sites de `Declenche`/`DeclencheN` de la
//	                 production, DEUX sont du balayage (`ScanKeyframeInventory`, et la pose
//	                 des largeurs d axe de la carte de `world_object_precision.go`) et SEIZE
//	                 de l assemblage, repartis sur 15 fichiers. Les seize se re-declenchent tout seuls quand l assemblage rejoue :
//	                 persister le rapport d APRES assemblage les compterait DEUX FOIS.
//	4  statborg      les enregistrements d entite, les instants de rafale de capture, le temoin
//	                 de troncature, et l horloge des chunks du manifeste.
//	5  killsource    le resultat du kill-feed, ou nil.
//
// # CE QUE L ALLER-RETOUR PERD, ET POURQUOI C EST SANS EFFET SUR LE DOCUMENT
//
// `killsource.Kill.paquet` est NON EXPORTE ET DELIBEREMENT (`killsource/kill.go:69-72`) : c est une
// coordonnee INTERNE au decodeur — ou l octet a ete lu — et aucun consommateur hors de
// `killsource` ne la lit. Elle se perd donc a l aller-retour. SANS EFFET SUR LE DOCUMENT, mais AVEC
// effet sur `digest.Of`, qui hache les champs exportes OU NON (`internal/analysis/digest/digest.go:20`) :
// c est precisement ce qui interdit a l etape `killsource` du TSV d equivalence de servir d oracle
// a un rejeu depuis les faits. S8 se juge sur la ligne `artifact`, et sur elle seule.
// [TestFilmFactsFichierNePerdQueLePaquetDeKillsource] est le ratchet de cette phrase : tout AUTRE
// champ non exporte apparaissant dans le graphe des cinq sections le fait rougir.
//
// # POURQUOI LES SECTIONS 2 A 5 SONT EN JSON, ET PAS AU CODEC MAISON
//
// DECISION MESUREE, pas un raccourci. Le codec maison existe pour la section 1 et il y gagne son
// prix : les positions sont des suites longues et redondantes, ou le delta-varint ramene 12 octets
// a un ou deux. Les sections 2 a 5 n ont AUCUNE redondance de ce genre — une identite, un rapport
// de quelques lignes, et deux structures profondes et heterogenes (`killsource.Result` porte a lui
// seul une douzaine de types imbriques).
//
// Ce qu un codec ecrit a la main y couterait : quelques centaines de lignes dont LE MODE DE PANNE
// EST LA PERTE SILENCIEUSE — un champ ajoute a `killsource.Kill` qu on oublie d ecrire rend des
// faits plausibles et faux. `encoding/json` derive du TYPE : le champ voyage tout seul. Les trois
// proprietes dont on a besoin sont tenues :
//
//	DETERMINISME  `encoding/json` TRIE les cles de map (depuis Go 1.12). C est ce qui repond a
//	              `StatRecord.Comps map[int]StatValue` — une map Go ne s itere pas deux fois
//	              pareil, et un ORDRE D ECRITURE EXPLICITE est exige. Il l est par construction
//	              ici, et [TestFilmFactsFichierEstUnPointFixe] le mesure.
//	FIDELITE      les entiers passent par leurs chiffres exacts (jamais par un float64), donc
//	              aucun xuid ne perd de bit ; les flottants par la forme courte qui rejoue.
//	COMPLETUDE    tenue par reflexion, pas par relecture humaine (les deux tests ci-dessus).
//
// `encoding/gob` a ete ESSAYE et REJETE le 2026-09-17, sur mesure : il ignore SILENCIEUSEMENT les
// champs non exportes des types imbriques, et un aller-retour des huit fixtures d entrees y a perdu
// 32 % du volume (11 049 200 octets contre 7 498 871 apres transcodage). Un format qui perd sans le
// dire est exactement ce que ce lot doit interdire.

import (
	"encoding/json"
	"errors"
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// magieFaitsDeFilm ouvre tout fichier de faits. INVARIANTE A JAMAIS : c est ce qui rend le refus
// d un fichier etranger explicite au lieu de le faire mourir sur un varint illisible.
const magieFaitsDeFilm = "LEVELUPFILMFACTS\n"

// VersionCodecFaits est la version du CONTENEUR : forme du prefixe, de l en-tete et du cadre des
// sections. Elle monte quand le CADRE change, jamais quand une section change de contenu.
const VersionCodecFaits = 1

// SchemaDesFaits est la version de la CHARGE : quelles sections existent et ce qu elles portent.
// Elle monte quand une section nait, meurt ou change de contenu.
//
// DISTINCTE DE [VersionCodecFaits] parce que les deux ne commandent pas la meme decision : un
// lecteur qui ne connait pas le CONTENEUR ne sait rien lire ; un lecteur qui connait le conteneur
// mais pas le SCHEMA sait lire l en-tete, donc sait dire « perime » proprement.
// SCHEMA 2 (2026-09-18, lot 4.1.3) : la CHARGE de la section 1 a change de forme — le blob des
// entrees passe en v23 (pistes d objets du monde en float32 exact, record de creation entier,
// denominateurs entiers). C EST CE NUMERO QUI DOIT PORTER LE CHANGEMENT, et pas la seule magie du
// blob : la magie vit DANS la section 1, donc un fichier d une version anterieure passerait le
// verdict de fraicheur (l en-tete est identique, les quatre revisions n ont pas bouge) et ne
// serait refuse qu au decodage de la section, par le chemin « illisible malgre un en-tete frais ».
// Un fichier PERIME doit se dire perime SUR SON EN-TETE, en 110 octets, pas apres avoir ete lu
// jusqu au mega-octet de positions. [TestFaitsDUnSchemaAnterieurSontRefusesSurLEnTete] le prouve.
// SCHEMA 3 (2026-09-19, post-chantier lot 5.1) : la CHARGE de la section 1 change encore — le blob
// des entrees passe en v24 (la JAUGE DE RETOUR du drapeau et son temoin). Meme raisonnement qu au
// schema 2 : le refus doit tomber sur l EN-TETE, pas au decodage de la section.
// SCHEMA 4 (2026-09-23, campagne « retours rejeu », lot M3) : la CHARGE de la section 1 change —
// le blob passe en v26 (la sante de la marche d image-cle, puis les dotations de naissance lues
// dans le record NEW du bipede). Meme raisonnement qu aux schemas 2 et 3 : le refus tombe sur
// l EN-TETE. La grammaire monte avec (`grammar.Rev`) : les faits d avant sont PERIMES, il faut
// redecoder.
const SchemaDesFaits = 4

// Identifiants de section. Ils ne se reutilisent JAMAIS : un identifiant retire reste retire, sinon
// un vieux fichier se relit comme une section qui n est pas la sienne.
const (
	sectionEntrees    = 1
	sectionIdentite   = 2
	sectionReplis     = 3
	sectionStatborg   = 4
	sectionKillsource = 5
)

// ErrFilmFactsVersion : le fichier n est pas de ce conteneur ou pas de ce schema. Les faits sont
// alors PERIMES, pas corrompus : la reponse est de redecoder le film, jamais de lire « au mieux ».
var ErrFilmFactsVersion = errors.New("faits de film : version de codec ou de schema inconnue")

// ErrFilmFactsRevisions : les faits ont ete pris sous d autres revisions de couche que celles du
// binaire courant. PERIMES, au sens de la regle de fraicheur TOUT OU RIEN du lot 4.1.
var ErrFilmFactsRevisions = errors.New("faits de film : revisions de couche differentes")

// FilmStatborg porte ce que la porte des ENREGISTREMENTS D ENTITE rend d un film.
//
// LES QUATRE VOYAGENT ENSEMBLE parce que les quatre sont consommes ensemble par les calques de
// score, d objectifs, de drapeau, de couronne, de crane et de bombe.
type FilmStatborg struct {
	// Records : les enregistrements d entite decodes des paquets FRAME.
	Records []types.StatRecord
	// BurstMS : les instants de rafale de capture, en millisecondes de match.
	BurstMS []int
	// Truncated : le temoin de TRONCATURE du balayage. Une liste courte et un balayage
	// interrompu ne disent pas la meme chose, et la couverture publie la difference.
	Truncated bool
	// ChunkStartMS : l horloge du MANIFESTE, `index de chunk -> start_ms`.
	//
	// ELLE EST ICI PARCE QU ELLE EST UN FAIT DU FILM, et qu un rejeu depuis les faits n ouvre
	// pas le manifeste. L anneau d armement de la bombe se date sur elle (`bombInput`) : sans
	// elle, un film d Assaut rejoue perdrait `chunkStartMS` et le calque d armement avec.
	ChunkStartMS map[int]int
}

// FilmFactsFile est le contenu COMPLET d un fichier de faits : l en-tete et les cinq sections.
type FilmFactsFile struct {
	// Coverage : les quatre revisions, le build et le registre — l en-tete, et le MEME type que
	// `coverage.decoder` de l artefact.
	Coverage DecoderCoverage
	// Facts : section 1 — les entrees de l assemblage et la cle de cuisson.
	Facts FilmFacts
	// Identity : section 2 — la section 2 de `chunk_00`, sans laquelle le build sort vide.
	//
	// EN POINTEUR, ET SON ABSENCE A UN SENS : `Options.FilmIdentity` est nil quand le film ne
	// porte AUCUNE section d identification (5 films du cache, versions majeures 31 et 33), et
	// `couvertureDuDecodeur` en tire « build vide, bloc registry absent ». Le porter par VALEUR
	// aurait aplati ce nil sur une identite a zero — c est-a-dire exactement l ambiguite que D-7
	// interdit, reintroduite par le fichier de faits.
	Identity *profile.FilmIdentity
	// Fallbacks : section 3 — le rapport des replis DU BALAYAGE, et de lui seul (cf. l en-tete).
	Fallbacks []fallback.Declenchement
	// Statborg : section 4.
	Statborg FilmStatborg
	// Kills : section 5 — le resultat du kill-feed, ou nil quand il n a pas ete decode.
	Kills *killsource.Result
}

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
	// corps est l offset du premier octet de section.
	corps int
}

// EncodeFilmFactsFile serialise un fichier de faits.
//
// Rend une erreur quand une section JSON ne se serialise pas : un fichier de faits INCOMPLET
// ecrirait des faits plausibles et faux, ce qui est pire que pas de fichier du tout.
func EncodeFilmFactsFile(f *FilmFactsFile) ([]byte, error) {
	entete := &gwriter{}
	encodeCouvertureDuDecodeur(entete, f.Coverage)
	entete.str(f.Facts.MapModule)
	for a := 0; a < 3; a++ {
		entete.u(uint64(f.Facts.AxisW[a]))
	}
	entete.bool8(f.Facts.LayoutDetected)

	w := &gwriter{b: []byte(magieFaitsDeFilm)}
	w.u(VersionCodecFaits)
	w.u(SchemaDesFaits)
	w.u(uint64(len(entete.b)))
	w.b = append(w.b, entete.b...)

	entrees := &gwriter{}
	blob, err := EncodeFilmFactsAvecErreur(&f.Facts)
	if err != nil {
		return nil, err
	}
	entrees.u(uint64(len(blob)))
	entrees.b = append(entrees.b, blob...)
	encodeGardesDeMode(entrees, f.Facts.FilmInputs)
	if entrees.echec != nil {
		return nil, entrees.echec
	}
	ecrireSection(w, sectionEntrees, entrees.b)

	for _, s := range []struct {
		id      int
		valeur  any
		libelle string
	}{
		{sectionIdentite, f.Identity, "identite du film"},
		{sectionReplis, f.Fallbacks, "rapport de replis"},
		{sectionStatborg, f.Statborg, "statborg"},
		{sectionKillsource, f.Kills, "killsource"},
	} {
		charge, err := json.Marshal(s.valeur)
		if err != nil {
			return nil, fmt.Errorf("faits de film : section %s : %w", s.libelle, err)
		}
		ecrireSection(w, s.id, charge)
	}
	return w.b, nil
}

// ecrireSection ecrit `[id][longueur][octets]` — le cadre qui permet a un lecteur de SAUTER ce
// qu il ne connait pas.
func ecrireSection(w *gwriter, id int, charge []byte) {
	w.u(uint64(id))
	w.u(uint64(len(charge)))
	w.b = append(w.b, charge...)
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
	if r.off+longueur > len(r.b) {
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
	if r.err != nil {
		return e, fmt.Errorf("%w : en-tete illisible (%v)", ErrFilmFactsVersion, r.err)
	}
	return e, nil
}

// Utilisable dit si ces faits sont relisables PAR LE BINAIRE COURANT, ou rend la raison typee.
//
// TOUT OU RIEN, ET C EST LA REGLE DU LOT 4.1 (note de preparation de M4, §2.4) : version de codec,
// schema de faits, LES QUATRE REVISIONS DE COUCHE et la cle de cuisson. La finesse par couche est
// l objet du lot 4.4 (`coverage_decoder.go` le dit deja) — ne pas l anticiper ici.
//
// # `build` ET `registry` NE SONT PAS COMPARES, ET C EST UNE PROPRIETE, PAS UN OUBLI
//
// Les quatre revisions sont des CONSTANTES DE COMPILATION : le binaire courant les connait sans
// ouvrir un fichier. `build` et le bloc `registry`, eux, sont des FAITS DU FILM — la cle ecrite
// dans la section 2 de `chunk_00` et l empreinte de son registre ECS. Ils ne peuvent pas avoir
// change pour un film donne, et les RECALCULER exigerait precisement ce que cette porte evite :
// ouvrir le film. Les comparer serait donc soit impossible, soit une tautologie.
func (e FilmFactsEntete) Utilisable(entry profile.MapQuantEntry) error {
	if e.VersionCodec != VersionCodecFaits || e.Schema != SchemaDesFaits {
		return ErrFilmFactsVersion
	}
	courantes := couvertureDuDecodeur(nil)
	if !memesRevisionsDeCouche(e.Coverage, *courantes) {
		return fmt.Errorf("%w : faits {%s %s %s %s} contre binaire {%s %s %s %s}",
			ErrFilmFactsRevisions,
			e.Coverage.SourceRev, e.Coverage.ProfileRev, e.Coverage.GrammarRev, e.Coverage.FactsRev,
			courantes.SourceRev, courantes.ProfileRev, courantes.GrammarRev, courantes.FactsRev)
	}
	return verifierCleDeCuisson(e.MapModule, e.AxisW, e.LayoutDetected, entry)
}

// DecodeFilmFactsFile relit un fichier de faits ENTIER. `entry` est l entree de catalogue de la
// carte du film : les positions sont des quanta, et les relire avec une autre entree rendrait des
// coordonnees FAUSSES (cf. [ErrFilmFactsCarte]).
func DecodeFilmFactsFile(blob []byte, entry profile.MapQuantEntry) (*FilmFactsFile, error) {
	entete, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		return nil, err
	}
	if err := verifierCleDeCuisson(entete.MapModule, entete.AxisW, entete.LayoutDetected, entry); err != nil {
		return nil, err
	}
	out := &FilmFactsFile{Coverage: entete.Coverage}
	r := &greader{b: blob, off: entete.corps}
	for r.off < len(r.b) && r.err == nil {
		id := int(r.u())
		charge := r.tranche(int(r.u()))
		if r.err != nil {
			break
		}
		if err := out.lireSection(id, charge, entry); err != nil {
			return nil, err
		}
	}
	if r.err != nil {
		return nil, fmt.Errorf("faits de film : cadre de section illisible : %w", r.err)
	}
	return out, nil
}

// lireSection pose UNE section. UN IDENTIFIANT INCONNU EST SAUTE, et c est tout le point du cadre
// a longueur prefixee : la fraicheur se decide sur l en-tete, pas en butant sur des octets.
func (f *FilmFactsFile) lireSection(id int, charge []byte, entry profile.MapQuantEntry) error {
	switch id {
	case sectionEntrees:
		r := &greader{b: charge}
		blob := r.tranche(int(r.u()))
		if r.err != nil {
			return fmt.Errorf("faits de film : section entrees : %w", r.err)
		}
		g, err := DecodeFilmFacts(blob, entry)
		if err != nil {
			return err
		}
		f.Facts = *g
		decodeGardesDeMode(r, &f.Facts.FilmInputs)
		if r.err != nil {
			return fmt.Errorf("faits de film : canaux gardes : %w", r.err)
		}
		return nil
	case sectionIdentite:
		return lireSectionJSON(charge, &f.Identity, "identite du film")
	case sectionReplis:
		return lireSectionJSON(charge, &f.Fallbacks, "rapport de replis")
	case sectionStatborg:
		return lireSectionJSON(charge, &f.Statborg, "statborg")
	case sectionKillsource:
		return lireSectionJSON(charge, &f.Kills, "killsource")
	default:
		return nil
	}
}

// lireSectionJSON relit une section, et NOMME la section dans l erreur : « json invalide » sans
// dire laquelle n aide personne a 3 h du matin.
func lireSectionJSON(charge []byte, cible any, libelle string) error {
	if err := json.Unmarshal(charge, cible); err != nil {
		return fmt.Errorf("faits de film : section %s : %w", libelle, err)
	}
	return nil
}

// encodeCouvertureDuDecodeur / decodeCouvertureDuDecodeur : [DecoderCoverage] VERBATIM.
//
// Le bloc `registry` est un POINTEUR et son absence a un sens ecrit (le registre n a pas ete lu) :
// il voyage donc derriere son temoin, jamais aplati sur des zeros.
func encodeCouvertureDuDecodeur(w *gwriter, c DecoderCoverage) {
	w.str(c.SourceRev)
	w.str(c.ProfileRev)
	w.str(c.GrammarRev)
	w.str(c.FactsRev)
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
		SourceRev:  r.str(),
		ProfileRev: r.str(),
		GrammarRev: r.str(),
		FactsRev:   r.str(),
		Build:      r.str(),
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

// memesRevisionsDeCouche compare LES QUATRE REVISIONS, et elles seules.
//
// PAS `a == b` SUR LE TYPE ENTIER, et ce n est pas qu une question de perimetre : `DecoderCoverage`
// porte un POINTEUR (`Registry`), donc `==` compare des ADRESSES — deux couvertures identiques
// sorties de deux appels seraient alors toujours differentes, et la porte de fraicheur refuserait
// TOUS les faits en silence (constate le 2026-09-17 a la pose de cette porte).
func memesRevisionsDeCouche(a, b DecoderCoverage) bool {
	return a.SourceRev == b.SourceRev && a.ProfileRev == b.ProfileRev &&
		a.GrammarRev == b.GrammarRev && a.FactsRev == b.FactsRev
}
