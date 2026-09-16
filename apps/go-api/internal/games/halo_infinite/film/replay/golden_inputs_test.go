package replay

// golden_inputs_test.go — LES ENTREES DECODEES DU FILM DE REFERENCE, FIGEES.
//
// POURQUOI CE FICHIER EXISTE. `BuildFromPositions` est deja PUR — il ne lui manquait que ses
// entrees. Tant qu elles ne vivaient que dans un film de 20,2 Mo hors depot, les chiffres du
// chantier (475/519 tirs, 90/105 vies nommees, 70 lancers, 439 projectiles, 184 etats
// d inventaire) n etaient verrouilles par AUCUN test : ils vivaient dans des Markdown et dans
// un artefact genere. Un refactor de l assemblage pouvait les deplacer en silence.
//
// CE QUE CE FIXTURE EST, ET CE QU IL N EST PAS. C est un ETAGE 1 : les entrees DEJA DECODEES,
// serialisees, rejouees dans l assemblage pur. Il verrouille l ASSEMBLAGE et rien d autre — un
// changement du DECODAGE ne le fait pas bouger, c est le travail de l etage 2 (la mini-bobine,
// minifilm_test.go). Les deux etages sont complementaires et ne se remplacent pas.
//
// ZERO OCTET DE FILM. Les tests de golden_assembly_test.go ne lisent que ce fichier fige. Le
// film n intervient qu a la REGENERATION, ci-dessous, qui est la seule porte d ecriture.
//
// REGENERATION (la seule ; un golden ne s edite JAMAIS a la main) :
//
//	REPLAY_FILM_DIR=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/games/halo_infinite/film/replay/ -run GoldenInputs -update
//
// puis, la sortie figee elle-meme :
//
//	go test ./internal/games/halo_infinite/film/replay/ -run GoldenAssembly -update
//
// CE QUE LE FIXTURE PORTE, ET POURQUOI PAS PLUS. Il porte exactement les champs que
// l assemblage CONSOMME (cf. la liste par type ci-dessous). Il ne porte NI la geometrie NI la
// structure de carte : celles-ci ne viennent pas du film mais de catalogues versionnes a part,
// et les inclure ferait grossir le fixture d un ordre de grandeur pour verrouiller un chargement
// de fichier, pas un decodage.
//
// LE FORMAT EST DELTA-CODE, ET L ORDRE D ORIGINE EST PRESERVE. `BuildFromPositions` trie ses
// positions par `sort.SliceStable` et numerote les traces dans l ORDRE DE PREMIERE APPARITION
// des slots : reordonner les positions changerait la sortie. Le codec conserve donc la suite
// telle que le decodeur l a produite, et ne delta-code que les VALEURS (horodatage global,
// coordonnees par slot).

import (
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

var updateGolden = flag.Bool("update", false, "reecrire les fichiers figes de testdata/")

// goldenFilm est le film de reference du chantier : Cliffhanger, Fiesta, 8 joueurs.
const goldenFilm = "000d5950"

// goldenDir est le repertoire des sorties figees.
const goldenDir = "testdata"

// goldenInputsPath est le fixture d entrees, versionne.
func goldenInputsPath() string {
	return filepath.Join(goldenDir, "inputs_"+goldenFilm+".bin.gz")
}

// goldenInputsMagic identifie le format et sa version. Un fixture d une autre version est une
// ERREUR, jamais une lecture « au mieux » : un decodage decale rendrait des chiffres plausibles.
//
// LA MAGIE S INCREMENTE DANS LE MEME COMMIT QUE LA SUITE DES SECTIONS — c est la seule chose
// qui rend la garde utile. Laisser la magie en arriere (constat de revue du 2026-08-25 : le
// commentaire annoncait v10, la constante disait encore v9) fait ACCEPTER un fixture d une
// autre version par la garde, et le decodage decale meurt plus loin sur un « uvarint illisible »
// a un offset arbitraire : bruyant par chance, pas par construction, et le message ne dit pas
// quoi faire. [TestGoldenInputsVersionGuard] verrouille le refus explicite.
//
// v21 (2026-09-14, lot 1.7) : le fixture porte L EQUIPE DE CHAQUE JOUEUR (`FilmInputs.PlayerTeams`,
// `index de joueur -> designateur`, lue par `filmdec.ScanPlayerTeams` dans le composant i0 de
// ti=9) ET le RAPPORT de cette lecture (`FilmInputs.TeamScan`). Les deux, parce qu une table vide
// et une lecture REFUSEE ne disent pas la meme chose : `coverage.teams` publie la difference, et
// un fixture qui ne porterait que la table figerait un refus comme un film sans equipes.
//
// v20 (2026-09-14, lot 1.6) : le fixture porte LA TABLE DES JOUEURS DU FILM (`FilmInputs.FilmTable`,
// lue par `ScanFilmPlayerTable` dans `chunk_00`) — le lien DIRECT `index <-> xuid <-> gamertag`
// dont le registre d identite fait sa source premiere. Elle porte son REFUS comme elle porte ses
// sieges : une table non lue n est pas une table vide, et le document publie la difference.
// Les neuf champs courts et le jeton de session que `filmdec.PlayerSlot` expose N ENTRENT PAS —
// aucun assemblage ne les lit, et la doctrine de ce fichier est que le fixture porte ce que
// l assemblage CONSOMME.
//
// v19 (2026-09-14, lot 1.0) : LE FIXTURE PORTE LE TYPE DE LA PRODUCTION. `goldenInputs` embarque
// desormais [FilmInputs] — ce que `scanFilmInputs` rend et ce que `BuildFromPositions` consomme —
// au lieu de redeclarer la liste a la main. SIX CANAUX y entrent du meme geste, parce que le
// chemin du fixture appelle enfin l etage de production au lieu de le recopier :
// `BipedCreations`, `WeaponChanges`, `Pickups` (+ stats), `EquipmentChanges` (+ stats),
// `ZoomEvents` (d ou sort `Options.Scoped`) et `Vehicles` (cf. golden_inputs_canaux_test.go).
// Les huit goldens d assemblage cessent donc d affirmer des calques VIDES que la production
// publie. La suite des sections change en deux points de plus : la table des slots descend de
// l en-tete dans la section des positions (un SEUL codec de positions, partage avec le nuage des
// vehicules — cf. encodePositionSection), et les six nouvelles sections s intercalent chacune
// pres de sa famille.
//
// v18 (2026-09-14, lot 0.D revue R1) : le fixture porte les VERDICTS DE BALAYAGE que
// l assemblage publie — `InventoryDeltaAmmoRefused` (« canal munitions refuse », publie en
// `coverage.grenadeReads.ammoRefused`) et `FilmMajorVersion` (publie en
// `coverage.filmMajorVersion`). Sans eux ces deux lignes etaient des CONSTANTES dans les huit
// goldens et les huit fixtures, et la fidelite ne pouvait pas les contredire — meme piege que
// le temoin `Scanned` de v13. Au meme geste, RETRAIT de deux champs que le codec portait sans
// qu aucun assemblage ne les lise : `InventoryDelta.Ammo` (entre en v15) et
// `KeyframeInventory.GrenadesByPosition` — la doctrine de ce fichier est que le fixture porte
// ce que l assemblage CONSOMME, ni plus ni moins.
//
// v17 (2026-09-14, lot 0.D.7) : le decoupage d i0 vient du CATALOGUE, par la fonction de la
// production (`NewFilmContextForMap(...).ImposedLayout()`), et le blob porte un drapeau
// `LayoutDetected` qui NOMME le repli d auto-detection — que la production n emploie que sur
// une entree de carte invalide. Une contradiction entre le blob et le catalogue, dans les DEUX
// sens, est une erreur typee (`errGoldenInputsDecoupage`). Sur `60ae07c4` la detection rendait
// `13/12/11` la ou le catalogue rend `12/12/11` : le golden y affirmait des coordonnees que la
// production ne produit pas.
//
// v16 (2026-09-14, lot 0.D.3 bis) : les positions portent les QUANTA du film (`BipedPosition.Q`,
// delta-varint par slot) et non plus les flottants derives ; la relecture re-dequantifie par
// `filmdec.DequantBipedAxis`, avec les bornes de l entree de catalogue passee en PARAMETRE. Le
// blob ouvre sur le MODULE de la carte et refuse une entree qui ne correspond pas
// (`errGoldenInputsCarte`), parce que les bornes d une autre carte rendent des coordonnees
// FAUSSES et non approximatives. Le blob porte aussi les trois largeurs d axe employees. Prix :
// les huit fixtures passent de 10,35 a 9,86 Mio — MOINS que les flottants ronds d avant.
//
// v15 (2026-09-14, lot 0.D.3) : le fixture a porte enfin ce que l assemblage lit —
// `KeyframeInventory.SelectedGrenadeRank` (il vaut -1 quand rien n est selectionne ; relu 0, il
// inventait une selection sur les huit builds) et des coordonnees EXACTES au lieu d un arrondi
// au centimetre (`equipmentOwner` choisit le poseur d une pose a la plus courte distance :
// l arrondi faisait basculer trois poses de `fb1a1a72`). `InventoryDelta.Ammo` y est entre AU
// MEME GESTE, mais A TORT : la revue de ronde 1 a montre le 2026-09-14 qu aucun assemblage ne
// le lit, et il est RETIRE en v18.
//
// v14 (2026-09-04, lot P5) : le fixture porte les CHARGES D'EQUIPEMENT RESTANTES (les
// emplacements ARMES du composant i56, quartet haut = charges entieres — rapport R11) et
// les statistiques de leur balayage, temoins `Absent` et `Scanned` compris (la lecon H1 de
// la seconde passe de revue P3, appliquee d'emblee : sans `Scanned`, le fixture rendrait
// une couverture de zeros indistinguable d'un balayage qui n'a jamais tourne). Que
// BuildFromFilm decode desormais. La magie monte parce que la SUITE DES SECTIONS change
// (un fixture v13 relu par ce codec deraillerait des la premiere lecture de charge). LE
// FILM DE REFERENCE EN PORTE : `000d5950` est de famille B, ou le grappin est le rang 20
// et le propulseur le rang 21 — les deux familles que le manifeste declare mesurees sur ce
// canal — donc le golden exerce le calque pour de vrai, jointure d'identite comprise.
//
// v13 (2026-09-03, lot P3) : le fixture porte les IMPULSIONS DE CAPACITE (corps tag==1 des
// composants i57/i59) et les statistiques de leur balayage — dont le temoin `Scanned`, ajoute
// a la revue de ronde 1 (constat H1) : sans lui le fixture rendrait une couverture de zeros
// indistinguable d un balayage qui n a jamais tourne, et l assemblage publierait ce zero comme
// une mesure. Que BuildFromFilm decode desormais. La magie monte parce que la SUITE DES SECTIONS change (un fixture v12 relu par
// ce codec derailerait des la premiere pose d equipement). LE FILM DE REFERENCE EN PORTE :
// `000d5950` est de famille B, ou le propulseur est le rang 21, et le lot R8 y a mesure
// 43 lectures `tag == 1` dont 38 sur ce rang — le golden exerce donc le calque pour de vrai,
// jointure d identite comprise, au lieu de figer une liste vide.
//
// v12 (2026-09-03, lot P1bis) : chaque teleportation porte desormais son VA-ET-VIENT — les
// deux positions monde lues dans la CHARGE de l evenement 117 (R6 par.1, valide 18/18) — plus le
// temoin qui dit si la charge a ete lue. La magie monte parce que la SUITE DES SECTIONS change
// (un fixture v11 relu par ce codec derailerait des la premiere teleportation).
//
// LES OCTETS FIGES ONT BOUGE, ET CE N EST PAS CE LOT : le film de reference ne porte AUCUNE
// teleportation, donc la section ci-dessus ecrit exactement les memes octets qu en v11 (un
// compte a zero, puis rien) ; 98 octets diffèrent pourtant hors magie. LA CAUSE EST UN ORDRE
// NON DETERMINISTE EN AMONT, mesuree ici : deux regenerations SUCCESSIVES du meme film
// rendent des fixtures differentes (verifie le 2026-09-03 — copie, `-update`, `cmp`). Ce n est
// donc PAS une regression de donnees de ce lot, mais un fixture non reproductible : la piste
// est `filmdec.lessTrack` (`projectiles.go`), qui ordonne des SEGMENTS sur (naissance, slot,
// gen) alors que `splitLives` en produit plusieurs par cle, avec un `sort.Slice` NON STABLE
// derriere. Consigne au plan (Decouvertes, G2), NON traitee dans ce lot — hors perimetre.
//
// v11 (2026-09-03, lot P1 lecture fiable de l equipement, schema 38) : le fixture porte les
// TELEPORTATIONS du translocateur (evenements type 117) que BuildFromFilm decode desormais —
// et que le decodage des positions CONSOMME AUSSI (exemption du filtre de vitesse a ±200 ms,
// decision D2) : sans elles le fixture ne porterait ni le calque `translocations` ni les
// positions telles que la production les decode. Le film de reference (Fiesta Cliffhanger)
// peut n en porter aucune : la liste vide se serialise, et le zero se fige avec le reste.
//
// v10 (2026-08-25, lot 4.4 du suivi delta de l inventaire) : le fixture porte les lectures
// d INVENTAIRE DELTA (compteurs de grenades i22 et jeu selectionne i47) que BuildFromFilm
// decode desormais. Sans elles le golden d assemblage n exercerait JAMAIS le second canal de
// l axe `grenadeReads` — il figerait un document que la production ne produit plus.
//
// v4 (2026-08-16, PLAN_EQUIPEMENT_TI37 phase 1) : la position serialise AUSSI le QUANTUM
// brut du bouclier (Shield.Q — la regle du surbouclier est `q > 64`, et le clamp de
// ShieldFraction efface l information), et le fixture porte les lectures CamoStates (i28
// queue[1]) que BuildFromFilm decode desormais.
//
// v5 (2026-08-16, PLAN_GRAPPIN_LIGNE phase 1) : le fixture porte les lectures GrappleReads
// (corps tag==3 d i59 — tir et accroche de grappin, quanta de position aux largeurs de la
// carte) que BuildFromFilm decode desormais.
//
// v6 (2026-08-18, PLAN_POSES_EQUIPEMENT_PUBLICATION phase 2) : le fixture porte les POSES
// d equipement (records de creation ti=37 confirmes par l oracle de position) ET la
// CALIBRATION du bloc de replication mesuree sur ce film. La calibration en fait partie parce
// que l assemblage la PUBLIE : sans elle, une liste vide de poses serait indistinguable d un
// film sans equipement, alors que ce peut etre un film dont la largeur n a pas ete tranchee.
//
// v7 (2026-08-17, PLAN_ARMES_AU_SOL_2E_LECTURE phase 3) : le fixture porte les ARMES AU SOL —
// records de creation ti=42 (position i0 et identite MPP), RECENSEMENT des images-cles qui
// borne les disparitions, et pistes de position qui disent si l objet a bouge. LES TROIS
// ENSEMBLE, parce qu il en manque une et le calque ment : sans les pistes, toute apparition
// passerait pour un objet apparu au repos, donc pour un socle.
//
// v8 (2026-08-18, PLAN_EXPLOITATION_REGISTRE_FILM lot E phase 1) : la position serialise AUSSI
// le SECOND scalaire d i21 (`PitchRaw`, l elevation de visee), que l assemblage publie
// desormais dans `Point.p`. Sans lui le fixture ne porterait qu un des deux angles du meme
// composant, et le golden verrouillerait un document dont toutes les visees sont a plat —
// c est-a-dire pas celui que la production sert.
//
// v9 (2026-08-19, PLAN_POWERUP_SOCLE_CATALYST phase 8) : le fixture porte la SECONDE voie de la
// chaine des socles — les creations `ti=37` avec leur recensement d images-cles et leurs pistes
// delta, d ou sortent les socles de POWER-UP. Elle est serialisee par le MEME codec que la voie
// des armes (une seule forme, `WorldObjectScan`), a la suite, et non a sa place : les deux
// entrent ensemble dans l assemblage.
// v22 (2026-09-15, lot 1.9.1) : le fixture porte les EVENEMENTS 103 `EquipmentSpawnedObject` et
// les denominateurs de leur balayage. Ils sont LE SIGNAL ECRIT de l origine d une pose de
// panneau (D13) : sans eux, un golden d assemblage figerait des poses dont l origine vient d un
// repli alors que la production la LIT.
const goldenInputsMagic = "REPLAYINPUTS22\n"

// goldenInputs porte les entrees de BuildFromPositions decodees du film de reference.
//
// LES CHAMPS SERIALISES, PAR TYPE — ce sont ceux que l assemblage consomme :
//
//	BipedPosition     Slot · TimestampUS · X/Y/Z · HasWorld · HasYaw+YawRaw+PitchRaw ·
//	                  HasBody+Body.Health · HasShield+Shield.Shield+Shield.Q
//	FireEvent         TimestampUS · FilmIndex · WeaponID · HasAim+Aim
//	KeyframeLoadout   TimestampUS · Slot · Families
//	GrenadeThrow      TimestampUS · FilmIndex · TypeID
//	ProjectileTrack   Slot · Gen · Pts(TimestampUS · X/Y/Z · AtRest)
//	KeyframeInventory tout, sauf Chunk/PacketIndex (traçabilite dans le film, pas une entree)
//	AbilityRank       Slot · TimestampUS · Rank (le compteur de rotation n entre pas dans
//	                  l assemblage : il borne une lecture, il ne se publie pas)
//	CamoRead          Slot · TimestampUS · Q (la voie queue[1] d i28 — l interrupteur)
//	GrappleRead       Slot · TimestampUS · Heavy · PosQ (les quanta de l ancre, aux
//	                  largeurs d axe de la carte)
//	Death             XUID · Gamertag · TimeMS
//	PlayerIndexTable  entier
//	ClockOriginUS     l horodatage du premier paquet du film (l origine publiee en depend)
//
// errGoldenInputsCarte : le fixture a ete cuit pour UNE carte, et on le relit avec une autre.
//
// ERREUR TYPEE parce que la confusion est SILENCIEUSE autrement : les positions du fixture sont
// des quanta, et `DequantBipedAxis` les rendrait avec les bornes de la mauvaise carte sans rien
// signaler — des coordonnees FAUSSES, pas approximatives (cf. son en-tete).
var errGoldenInputsCarte = errors.New("fixture d entrees : carte du catalogue differente")

// errGoldenInputsDecoupage : le fixture dit tenir son decoupage du catalogue, et le catalogue
// n en dit plus autant. Les quanta se dequantifieraient avec un AUTRE pas — silencieusement.
var errGoldenInputsDecoupage = errors.New("fixture d entrees : decoupage d i0 en contradiction avec le catalogue")

// imposeAxisW rend les largeurs d un decoupage impose, ou un marqueur quand il n y en a pas.
func imposeAxisW(impose *filmdec.I0Layout) any {
	if impose == nil {
		return "aucun (entree de carte invalide)"
	}
	return impose.AxisW
}

// goldenInputs porte les entrees de BuildFromPositions decodees du film de reference.
//
// IL EMBARQUE LE TYPE DE LA PRODUCTION (lot 1.0, 2026-09-14). Avant, il REDECLARAIT champ par
// champ ce que l assemblage consomme, et la liste devait etre maintenue en parallele de celle
// de `BuildFromFilm` — elle ne l a pas ete (decouverte D7 : cinq canaux absents). Le fixture
// porte desormais un [FilmInputs], c est-a-dire EXACTEMENT le type que l etage de balayage rend
// et que l assemblage consomme, plus les trois champs d en-tete qui n en font pas partie (le
// film, sa carte, son decoupage d i0). Un canal ajoute a `FilmInputs` apparait ici tout seul —
// et [TestCodecCouvreFilmInputs] exige qu il soit soit serialise, soit NOMME comme non
// transporte.
type goldenInputs struct {
	Film string
	// MapModule est le module de l entree de catalogue qui a dequantifie les positions
	// (`MapQuantEntry.Module`). Il ouvre le blob et se verifie a la relecture.
	MapModule string
	// AxisW est le decoupage d axe qui a produit les quanta — celui que le balayage a EMPLOYE,
	// pas celui du catalogue. Les deux different sur Live Fire (detecte [13 12 11], catalogue
	// [12 12 11]) : un bit d ecart sur X double le pas de quantification, donc l etendue des
	// coordonnees. Le porter est la seule facon de redequantifier a l identique.
	AxisW [3]uint
	// LayoutDetected dit que le decoupage ci-dessus vient de l AUTO-DETECTION et non du
	// catalogue. La production ne s y rabat que sur une entree de carte invalide
	// (`resolveI0Layout`) ; le drapeau existe pour que ce repli soit NOMME dans le fixture au
	// lieu de se confondre avec une lecture du catalogue.
	LayoutDetected bool
	// FilmInputs porte le reste : tout ce que l etage de balayage rend a l assemblage.
	FilmInputs
}

// options rend les Options d assemblage portees par le fixture, PAR LA FONCTION DE LA PRODUCTION
// (`FilmInputs.applyTo`). La geometrie et la structure sont volontairement absentes (cf.
// l en-tete) : elles ne viennent pas du film.
func (g *goldenInputs) options() Options {
	var opt Options
	g.FilmInputs.applyTo(&opt)
	return opt
}

// ---------------------------------------------------------------------------
// Encodeur / decodeur
// ---------------------------------------------------------------------------

// cmScale convertit une coordonnee en CENTIMETRES ENTIERS.
//
// CE N EST PAS UNE PERTE, ET C EST MESURABLE : toute coordonnee publiee par l assemblage passe
// par `round2` (arrondi au centieme), sans exception — traces, tirs, lancers, projectiles. Le
// centimetre entier est donc exactement la precision que la sortie porte. Coder un float32 brut
// couterait 12 octets par position pour une decimale que personne ne lit.

// loadGoldenInputs relit le fixture versionne. AUCUN OCTET DE FILM.
func loadGoldenInputs(t *testing.T) *goldenInputs {
	t.Helper()
	raw, err := os.ReadFile(goldenInputsPath()) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("fixture d entrees illisible : %v — regenerer avec "+
			"REPLAY_FILM_DIR=<film> go test -run GoldenInputs -update", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("fixture d entrees : gzip illisible : %v", err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("fixture d entrees : decompression : %v", err)
	}
	entry, err := goldenMapQuant()
	if err != nil {
		t.Fatalf("entree de catalogue du film de reference : %v", err)
	}
	g, err := decodeGoldenInputs(buf.Bytes(), entry)
	if err != nil {
		t.Fatalf("fixture d entrees : %v", err)
	}
	return g
}

// TestGoldenInputsRoundTrip : le codec ne perd rien de ce que l assemblage consomme.
//
// C EST LE FILET DU FIXTURE LUI-MEME. Un codec qui perdrait un champ produirait un golden
// parfaitement stable et parfaitement faux. On le verifie par la seule mesure qui compte :
// l assemblage rejoue sur les entrees RELUES doit rendre le MEME document que sur les entrees
// d origine — ici, les entrees d origine etant deja le fixture, on controle qu un second
// aller-retour est un point fixe, octet pour octet.
func TestGoldenInputsRoundTrip(t *testing.T) {
	g := loadGoldenInputs(t)
	blob := encodeGoldenInputs(g)
	again, err := decodeGoldenInputs(blob, goldenEntryPourTest(t))
	if err != nil {
		t.Fatalf("second decodage : %v", err)
	}
	if got := encodeGoldenInputs(again); !bytes.Equal(blob, got) {
		t.Fatalf("le codec n est pas un point fixe : %d octets contre %d — un champ se perd "+
			"ou se reconstruit differemment a chaque tour", len(got), len(blob))
	}
	docA := BuildFromPositions(goldenFilm, "halo_infinite", g.Positions, g.Fire, g.options())
	docB := BuildFromPositions(goldenFilm, "halo_infinite", again.Positions, again.Fire, again.options())
	if renderAssembly(docA) != renderAssembly(docB) {
		t.Error("l assemblage differe entre les entrees relues et leur re-serialisation : " +
			"le codec perd un champ que BuildFromPositions consomme")
	}
}

// TestGoldenInputsVersionGuard : un fixture d une AUTRE version est refuse par la MAGIE, et le
// message dit quoi faire.
//
// C EST LE VERROU DU CONSTAT DE REVUE DU 2026-08-25. La section `InventoryDeltas` a ete inseree
// au milieu du flux sans que la magie bouge : un fixture de la version precedente passait la
// garde, puis mourait plus loin sur « uvarint illisible a l offset N » — un message qui parle
// d octets alors que le probleme est une version. Le test relit le corps COURANT precede de la
// magie PRECEDENTE : la seule reponse acceptable est le refus de version.
func TestGoldenInputsVersionGuard(t *testing.T) {
	const previousMagic = "REPLAYINPUTS21\n"
	if previousMagic == goldenInputsMagic {
		t.Fatal("la magie precedente et la courante sont identiques : le test ne prouve plus rien")
	}
	body := encodeGoldenInputs(loadGoldenInputs(t))[len(goldenInputsMagic):]
	stale := append([]byte(previousMagic), body...)
	_, err := decodeGoldenInputs(stale, goldenEntryPourTest(t))
	if err == nil {
		t.Fatal("un fixture d une autre version a ete accepte : la garde de version ne sert a rien")
	}
	if !strings.Contains(err.Error(), "version inconnue") {
		t.Fatalf("la version doit etre refusee POUR CE QU ELLE EST ; message obtenu : %v", err)
	}
}

// TestGoldenInputsRegenerate : LA SEULE PORTE D ECRITURE DU FIXTURE.
//
// Elle exige DEUX conditions explicites — `-update` et `REPLAY_FILM_DIR` — parce qu une
// regeneration accidentelle transformerait n importe quelle regression en « nouvelle
// reference ». C est la meme discipline que le paquet killsource.
func TestGoldenInputsRegenerate(t *testing.T) {
	dir := os.Getenv("REPLAY_FILM_DIR")
	switch {
	case !*updateGolden:
		t.Skip("regeneration du fixture : passer -update (et REPLAY_FILM_DIR)")
	case dir == "":
		t.Skip("regeneration du fixture : REPLAY_FILM_DIR non defini")
	}
	g, err := decodeFilmInputs(goldenFilm, dir)
	if err != nil {
		t.Fatalf("decodage du film %s : %v", dir, err)
	}
	blob := encodeGoldenInputs(g)
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if _, err := zw.Write(blob); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := os.MkdirAll(goldenDir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", goldenDir, err)
	}
	if err := os.WriteFile(goldenInputsPath(), buf.Bytes(), 0o600); err != nil {
		t.Fatalf("ecriture du fixture : %v", err)
	}
	// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` (revue R1, constat R1-8).
	t.Fatalf("fixture reecrit : %s (%d octets brut, %d compresse) — %d positions, %d tirs, "+
		"%d loadouts, %d lancers, %d projectiles, %d inventaires, %d lectures grappin, "+
		"%d impulsions de capacite, %d lectures de charge, %d morts, %d index",
		goldenInputsPath(), len(blob), buf.Len(), len(g.Positions), len(g.Fire), len(g.Loadouts),
		len(g.Grenades), len(g.Projectiles), len(g.Inventory), len(g.GrappleReads),
		len(g.AbilityImpulses), len(g.AbilityCharges), len(g.Deaths), len(g.PlayerIndices.ByXUID))
}

// decodeFilmInputs rejoue EXACTEMENT la sequence de decodage de BuildFromFilm — c est ce qui
// garantit que le fixture porte les memes entrees que la production. Les bornes de carte sont
// celles de Cliffhanger, lues dans le catalogue versionne du titre.
