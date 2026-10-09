package grammar

// profile_test.go — LE PROFIL SUR LES SEPT BOBINES : les trois cles, l immuabilite, la table.
//
// Les gardes d EGALITE AVEC LES GLOBALES vivent dans `profile_globales_test.go` : ce sont deux
// questions distinctes — « le profil dit-il ce que le film ecrit » (ici) et « le profil dit-il
// ce que le decodeur applique » (la-bas).

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestProfilResoudLesTroisCles : sur chaque bobine par build, le profil resout la version de
// format, le build et la version majeure, et ses valeurs derivees suivent les oracles des lots
// 1.5.1 et 1.5.2 (`bobinesIdentite`, `bobinesTable`).
func TestProfilResoudLesTroisCles(t *testing.T) {
	tables := map[string]bobineTable{}
	for _, b := range bobinesTable() {
		tables[b.film] = b
	}
	for _, b := range bobinesIdentite() {
		film := bobineFilm(t, b.film)
		p := ResolveProfile(film, nil)
		if err := p.Err(); err != nil {
			t.Fatalf("%s (%s) : le profil refuse une cle que le film ecrit : %v", b.film, b.build, err)
		}
		if p.Build() != b.build {
			t.Errorf("%s : build du profil %q, attendu %q", b.film, p.Build(), b.build)
		}
		verifierFormatDuProfil(t, b, p)
		verifierSlotsDuProfil(t, tables[b.film], p)
		maj, lue := FilmMajorVersion(film)
		if !lue || p.Highlight().MajorVersion != maj || !p.Highlight().Lue {
			t.Errorf("%s : version majeure du profil %d (lue=%v), attendu %d (lue=%v)", b.film,
				p.Highlight().MajorVersion, p.Highlight().Lue, maj, lue)
		}
		t.Logf("%-10s build %-10q format %2d majeure %2d -> MPP %v perso %4d o (%+6d bits) "+
			"implantation %s (+%d o)", b.film, p.Build(), p.FormatVersion(),
			p.Highlight().MajorVersion, p.MPP(), p.Slots().PersoBytes, p.Slots().DeltaBits,
			p.Highlight().Implantation, p.Highlight().GamertagOffsetBytes)
	}
}

// verifierFormatDuProfil confronte la cle de FORMAT et le decoupage MPP qu elle porte.
func verifierFormatDuProfil(t *testing.T, b bobineIdentite, p profile.Profile) {
	t.Helper()
	attendu, connu := profile.MPPPourFormat(p.FormatVersion())
	if !connu {
		t.Fatalf("%s : format %d absent de la table alors que Err() est nul", b.film, p.FormatVersion())
	}
	if p.MPP() != attendu {
		t.Errorf("%s : MPP du profil %v, table %v", b.film, p.MPP(), attendu)
	}
	// Le cardinal de la table par type suit la version de format (oracle du lot 1.5.1) : c est
	// un CONTROLE independant de la cle, pas une seconde lecture de la meme valeur.
	if got := len(p.Identity().TypeVersions); got != b.types {
		t.Errorf("%s : %d entrees de table par type, attendu %d", b.film, got, b.types)
	}
}

// verifierSlotsDuProfil confronte la transposition de slot a l oracle du lot 1.5.2.
func verifierSlotsDuProfil(t *testing.T, b bobineTable, p profile.Profile) {
	t.Helper()
	if !p.Slots().Connu {
		t.Fatalf("%s : build inconnu du profil alors que Err() est nul", b.film)
	}
	if p.Slots().PersoBytes != b.persoOctets || p.Slots().DeltaBits != b.transpositionBits {
		t.Errorf("%s : slots du profil {%d o, %+d bits}, oracle {%d o, %+d bits}", b.film,
			p.Slots().PersoBytes, p.Slots().DeltaBits, b.persoOctets, b.transpositionBits)
	}
}

// TestProfilSansCleRendUneErreurTypeeEtPoseQuandMeme : un film sans `chunk_00` (bobine partielle,
// fixture) ne fait PAS mettre le film de cote — D-4 d ADR 0034 interdit de le lire au profil du
// voisin, pas de le lire du tout. Les deux erreurs sont typees, et le reste du profil est pose.
func TestProfilSansCleRendUneErreurTypeeEtPoseQuandMeme(t *testing.T) {
	p := ResolveProfile(nil, nil)
	if !errors.Is(p.Err(), profile.ErrUnknownFormat) || !errors.Is(p.Err(), profile.ErrUnknownBuild) {
		t.Fatalf("film nil : les deux cles doivent etre signalees typees, obtenu %v", p.Err())
	}
	if p.Keyframe().CadreBits() != 172 {
		t.Errorf("le cadre d image-cle ne depend d aucune cle : %d bits au lieu de 172",
			p.Keyframe().CadreBits())
	}
	if p.Movement().DeltaQuantum == 0 || p.Movement().WorldObject.AxisW[0] == 0 {
		t.Errorf("les invariants de mouvement ne dependent d aucune cle : %+v", p.Movement())
	}
	if p.Slots().Connu || p.MPP().Valid() {
		t.Errorf("aucune cle lue : ni slots ni MPP ne doivent etre poses (%+v / %v)",
			p.Slots(), p.MPP())
	}
	// L implantation du gamertag TOMBE EN TETE, et la NOMME : c est le comportement historique
	// que `grammar.ParseHighlightEvents` applique a la version 0, pas un silence.
	if h := p.Highlight(); h.Lue || h.Implantation != profile.ImplantationEnTete || h.GamertagOffsetBytes != 0 {
		t.Errorf("version majeure non lue : implantation attendue %q a l octet 0, obtenu %+v",
			profile.ImplantationEnTete, h)
	}
}

// TestProfilChunk00TronqueNePaniquePas : un `chunk_00` coupe rend une erreur typee, jamais une
// panique ni une identite partielle. Les sept longueurs couvrent l en-tete (4 o), la frontiere
// du second u32 (8 o) et un registre amorce mais coupe (le quart du chunk).
func TestProfilChunk00TronqueNePaniquePas(t *testing.T) {
	complet := bobineChunk00(t, "fb1a1a72")
	// Les six longueurs coupent AVANT le second u32 (4, 8), dans l en-tete, et dans le REGISTRE
	// (le quart d un chunk_00 tombe bien avant sa fin structurelle, 0x0CB200 octets) : la section
	// d identification n est alors pas atteinte, donc le build manque.
	for _, n := range []int{0, 1, 4, 8, 64, 1024, len(complet) / 4} {
		p := ResolveProfile(filmDUnChunk00(t, complet[:n]), nil)
		if p.Err() == nil {
			t.Errorf("chunk_00 tronque a %d octets : une cle au moins doit manquer", n)
		}
		if p.Keyframe().CadreBits() != 172 {
			t.Errorf("chunk_00 tronque a %d octets : le cadre invariant a bouge", n)
		}
	}
}

// filmDUnChunk00 fabrique un film d UN SEUL chunk, celui du registre, depuis des octets deja
// decompresses. Il sert aux entrees TRONQUEES : un `chunk_00` coupe ne se trouve pas sur disque.
func filmDUnChunk00(t *testing.T, chunk0 []byte) *source.Film {
	t.Helper()
	f, err := source.Load(source.MemoryChunks{chunk0}, nil)
	if err != nil {
		// Zero chunk n est pas un film : c est le cas `film nil`, deja couvert par son test.
		t.Fatalf("film synthetique d un chunk de %d octets : %v", len(chunk0), err)
	}
	return f
}

// TestProfilEstImmuable : un lecteur qui modifie ce qu il a recu ne modifie pas le profil du
// film. Sans cela « immuable » serait un mot dans un commentaire.
func TestProfilEstImmuable(t *testing.T) {
	p := ResolveProfile(bobineFilm(t, "fb1a1a72"), &profile.MapQuantEntry{Module: "temoin"})
	id := p.Identity()
	if len(id.TypeVersions) == 0 {
		t.Fatalf("bobine sans table par type : le test ne mesure rien")
	}
	avant := id.TypeVersions[0]
	id.TypeVersions[0] = avant ^ 0xffff
	id.Build = "SABOTE"
	if p.Identity().TypeVersions[0] != avant {
		t.Errorf("la table par type du profil est partagee avec son lecteur : %d au lieu de %d",
			p.Identity().TypeVersions[0], avant)
	}
	if p.Build() == "SABOTE" {
		t.Errorf("le build du profil a suivi la copie de son lecteur")
	}
	m := p.Map()
	m.Module = "SABOTE"
	if p.Map().Module != "temoin" {
		t.Errorf("l entree de carte du profil a suivi la copie de son lecteur : %q", p.Map().Module)
	}
}

// dateProfil : la forme obligatoire de la colonne Date de la table.
var dateProfil = regexp.MustCompile(`^20\d\d-\d\d-\d\d$`)

// TestProfilTableComplete : chaque ligne de la table porte ses six colonnes, et sa provenance
// est l une des trois. Une valeur sans preuve n est pas une valeur de profil (D3).
func TestProfilTableComplete(t *testing.T) {
	lignes := profile.TableProfil()
	if len(lignes) == 0 {
		t.Fatalf("table de profil vide : le ratchet ne mesure plus rien")
	}
	vues := map[string]bool{}
	for _, l := range lignes {
		cle := l.Cle + "|" + l.Champ
		if vues[cle] {
			t.Errorf("deux lignes pour %s : la table doit avoir UNE ligne par cle et par champ", cle)
		}
		vues[cle] = true
		switch {
		case l.Cle == "" || l.Champ == "" || l.Valeur == "":
			t.Errorf("ligne incomplete : %+v", l)
		case l.Preuve == "":
			t.Errorf("%s / %s : une valeur sans preuve n est pas une valeur de profil", l.Cle, l.Champ)
		case !dateProfil.MatchString(l.Date):
			t.Errorf("%s / %s : date %q, forme attendue AAAA-MM-JJ", l.Cle, l.Champ, l.Date)
		case l.Source != profile.ProvenanceRelue && l.Source != profile.ProvenanceMesuree &&
			l.Source != profile.ProvenancePresumee:
			t.Errorf("%s / %s : provenance %q hors des trois", l.Cle, l.Champ, l.Source)
		}
	}
}

// presumesGeles : les entrees PRESUMEES de la table du profil, gelees au 2026-09-17 (lot 2.1).
//
// CE RATCHET NE MONTE PAS. Une valeur presumee est une valeur dont la variabilite par build n a
// jamais ete mesuree : en ajouter une est une regression de connaissance, en retirer une (parce
// qu elle a ete relue chez l ecrivain ou mesuree sur un temoin) est le travail du lot 3.
var presumesGeles = []string{
	"toutes / Movement.Traversal",
	// `toutes / Movement.AbsoluteAxisW` EST SORTIE LE 2026-09-17 (lot 3.4.1-a), et c est
	// exactement le travail que ce ratchet annonce : la largeur UNIFORME devinee n est pas
	// devenue RELUE, elle a DISPARU au profit de la table par index de plage de la carte, dont
	// la loi est relue (`toutes / Movement.LoiLargeursAxe`, ligne RELUE de la table).
	"toutes / Movement.DeltaAxisWidth",
	"toutes / Movement.Range",
	"toutes / Movement.CalibratedSkip",
	"toutes / Movement.MobilityActionExtraBits",
}

// TestProfilPresumes LISTE les entrees presumees, une par une, et gele leur liste.
//
// C est le registre de ce qui reste a etablir : le lot 2.1 pose le profil, il ne transforme pas
// une supposition en mesure. Le message de ce test est la liste que le lot 3 (« exploiter le
// profil ») consommera.
func TestProfilPresumes(t *testing.T) {
	var vus []string
	for _, l := range profile.TableProfil() {
		if l.Source != profile.ProvenancePresumee {
			continue
		}
		vus = append(vus, l.Cle+" / "+l.Champ)
		t.Logf("PRESUME  %-38s %-22s ce qui manque : %s", l.Cle+" / "+l.Champ, l.Valeur, l.Preuve)
	}
	if strings.Join(vus, "\n") != strings.Join(presumesGeles, "\n") {
		t.Errorf("la liste des valeurs PRESUMEES a bouge.\nobtenu :\n  %s\ngele :\n  %s\n"+
			"En ajouter une est une regression de connaissance ; en retirer une se fait avec la "+
			"ligne de table qui passe RELUE ou MESUREE, et sa preuve.",
			strings.Join(vus, "\n  "), strings.Join(presumesGeles, "\n  "))
	}
}

// TestFilmContextResoutLeProfilALaConstruction : le constructeur de la CUISSON
// ([NewFilmContextForMap]) pose le profil, carte comprise, et son erreur remonte TYPEE par
// [FilmContext.ProfileErr] — le film n est pas mis de cote (item 2.1.2, D1).
func TestFilmContextResoutLeProfilALaConstruction(t *testing.T) {
	entry := profile.MapQuantEntry{Module: "temoin", Min: [3]float32{-1, -1, -1}, Max: [3]float32{1, 1, 1}}
	film := bobineFilm(t, "fb1a1a72")
	fc := NewFilmContextForMap(film, &entry, nil)
	if err := fc.ProfileErr(); err != nil {
		t.Fatalf("bobine HI_1_13_0 : le contexte refuse une cle que le film ecrit : %v", err)
	}
	if fc.Profile().Map().Module != entry.Module {
		t.Errorf("la carte du match n est pas au profil : %q", fc.Profile().Map().Module)
	}
	// MEME VALEUR QUE LA RESOLUTION DIRECTE : c est ce qui autorise le lot 2.1 a resoudre le
	// profil a DEUX endroits (ici et `replay.BuildFromFilm`) sans que les deux divergent.
	direct := ResolveProfile(film, &entry)
	if fc.Profile().Build() != direct.Build() || fc.Profile().MPP() != direct.MPP() ||
		fc.Profile().Slots() != direct.Slots() || fc.Profile().Movement() != direct.Movement() {
		t.Errorf("le profil du contexte differe de la resolution directe :\n  contexte %+v\n  direct %+v",
			fc.Profile(), direct)
	}
	// LE CONSTRUCTEUR SANS CARTE le resout au premier acces, et rend la MEME chose moins la carte.
	sansCarte := NewFilmContext(film).Profile()
	if sansCarte.Build() != direct.Build() || sansCarte.Map().Module != "" {
		t.Errorf("profil sans carte : build %q, module %q", sansCarte.Build(), sansCarte.Map().Module)
	}
	// CONTEXTE NIL : les invariants, jamais une largeur inventee.
	var nul *FilmContext
	if nul.Profile().Keyframe().CadreBits() != 172 || nul.Profile().Slots().Connu {
		t.Errorf("contexte nil : %+v", nul.Profile())
	}
}
