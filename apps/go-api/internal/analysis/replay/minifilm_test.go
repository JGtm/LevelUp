package replay

// minifilm_test.go — LA MINI-BOBINE : L ETAGE 2 DU VERROUILLAGE.
//
// CE QUE L ETAGE 1 NE PEUT PAS FAIRE. `inputs_000d5950.bin.gz` fige les entrees DEJA DECODEES :
// il verrouille l assemblage, et il est par construction AVEUGLE a un changement du decodeur —
// si `ScanFilmFireEvents` cessait de lire l arme, le fixture continuerait de la porter. Il faut
// donc du BINAIRE REEL, et il faut qu il tienne dans le depot.
//
// CE QUE LA MINI-BOBINE EST. Des PAQUETS REELS DU FILM, concatenes sans etre modifies d un bit,
// choisis pour couvrir les quatre decodeurs d evenements et rien d autre :
//
//	TOUTES les images-cles du film       armes portees (150), inventaire (184), bande de slots
//	tous les paquets a record de tir     les 519 evenements de tir (record type 105 long)
//	tous les paquets a lancer de grenade les 70 lancers (marqueur + identifiant en liste blanche)
//	une FENETRE de paquets consecutifs   des vols de projectile COMPLETS — un projectile vit
//	                                     moins d une seconde et n existe pas hors de sa fenetre
//	le chunk highlight, tel quel         le fil des morts (93 morts, xuid + instant)
//
// POURQUOI TOUTES LES IMAGES-CLES ET NON UNE SEULE. Mesure faite en construisant la bobine :
// la PREMIERE image-cle du film ne rend NI loadout NI inventaire — c est une image d amorce, et
// elle ne porte aucun record de biped exploitable. Et la bande de slots d objets se lit sur
// l ENSEMBLE des images-cles : un projectile vit moins d une seconde quand elles sont espacees
// de vingt, donc une seule image n en voit presque jamais. Vingt-cinq images-cles pesent
// quelques dizaines de Ko compresses ; en garder une seule aurait verrouille deux decodeurs sur
// un resultat vide, ce qui est pire que ne rien verrouiller.
//
// CE QU ELLE N EST PAS. Ce n est PAS un film valide : les paquets y sont concatenes hors de leur
// continuite, donc les POSITIONS de biped — qui s accumulent par deltas — y sont sans
// signification. C est assume : les positions sont verrouillees par l etage 1, qui les porte
// deja decodees. Une mini-bobine qui aurait voulu les porter aussi aurait pese le film entier.
//
// POIDS : ~20,2 Mo pour le film, 23 Go pour le cache complet, quelques centaines de Ko ici.
//
// REGENERATION (jamais d edition a la main) :
//
//	REPLAY_FILM_DIR=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/replay/ -run MiniFilmRegenerate -update

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// MiniFilmDir est le repertoire de la mini-bobine, relatif au paquet.
//
// EXPORTE parce que le paquet `filmdec` la lit aussi (par un chemin relatif) : ses decodeurs
// d evenements sont ceux que cette bobine verrouille, et dupliquer la bobine pour les servir
// aurait cree deux verites binaires a maintenir.
const MiniFilmDir = "testdata/minifilm_" + goldenFilm

// miniFilmWindowUS est la duree de la fenetre de paquets consecutifs conservee autour du premier
// lancer de grenade. Un projectile vit moins d une seconde et est repliqué a ~60 Hz : deux
// secondes suffisent a porter des VOLS ENTIERS, ce qui est la seule facon de verrouiller le
// decoupage en vies (`splitLives`) autrement que sur des donnees fabriquees.
const miniFilmWindowUS = 2_000_000

// miniFilmManifest decrit la provenance de chaque octet. Il est versionne AVEC la bobine :
// une fixture binaire sans provenance ecrite est un fait sans source.
const miniFilmManifest = "PROVENANCE.txt"

// TestMiniFilmRegenerate : LA SEULE PORTE D ECRITURE DE LA MINI-BOBINE.
func TestMiniFilmRegenerate(t *testing.T) {
	dir := os.Getenv("REPLAY_FILM_DIR")
	switch {
	case !*updateGolden:
		t.Skip("regeneration de la mini-bobine : passer -update (et REPLAY_FILM_DIR)")
	case dir == "":
		t.Skip("regeneration de la mini-bobine : REPLAY_FILM_DIR non defini")
	}
	sel, err := selectMiniFilmPackets(dir)
	if err != nil {
		t.Fatalf("selection des paquets : %v", err)
	}
	if err := os.MkdirAll(MiniFilmDir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", MiniFilmDir, err)
	}
	// chunk_01 : les paquets retenus, concatenes dans l ordre du film, puis zlib — le meme
	// conditionnement que les chunks reels, donc le meme chemin de lecture (`inflateChunk`).
	blob, err := zlibBytes(sel.body)
	if err != nil {
		t.Fatalf("zlib : %v", err)
	}
	if err := os.WriteFile(filepath.Join(MiniFilmDir, "chunk_01.bin"), blob, 0o600); err != nil {
		t.Fatalf("ecriture chunk_01 : %v", err)
	}
	// chunk_02 : le chunk highlight du film, OCTET POUR OCTET. Le recomposer n aurait aucun
	// sens — c est un chunk entier, et c est deja le plus petit du film.
	hl, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", sel.highlightChunk)))
	if err != nil {
		t.Fatalf("lecture du chunk highlight : %v", err)
	}
	if err := os.WriteFile(filepath.Join(MiniFilmDir, "chunk_02.bin"), hl, 0o600); err != nil {
		t.Fatalf("ecriture chunk_02 : %v", err)
	}
	if err := os.WriteFile(filepath.Join(MiniFilmDir, miniFilmManifest),
		[]byte(sel.provenance(len(blob), len(hl))), 0o600); err != nil {
		t.Fatalf("ecriture de la provenance : %v", err)
	}
	t.Logf("mini-bobine reecrite : chunk_01 %d octets (%d paquets, %d octets bruts), chunk_02 %d octets",
		len(blob), sel.count, len(sel.body), len(hl))
}

// miniSelection porte les paquets retenus et de quoi ecrire leur provenance.
type miniSelection struct {
	body           []byte
	count          int
	keyframes      int
	fire           int
	grenade        int
	window         int
	highlightChunk int
	sourceChunks   []int
}

func (s miniSelection) provenance(chunk1, chunk2 int) string {
	return fmt.Sprintf(`MINI-BOBINE — film %s (Cliffhanger, Fiesta, 8 joueurs)

CE FICHIER DIT D OU VIENT CHAQUE OCTET. Une fixture binaire sans provenance ecrite est un fait
sans source ; celle-ci se REGENERE, elle ne s edite pas :

    REPLAY_FILM_DIR=<repo>/data/cache/film_chunks/%s \
      go test ./internal/analysis/replay/ -run MiniFilmRegenerate -update

chunk_01.bin  %d octets (zlib) — %d paquets REELS concatenes dans l ordre du film,
              %d octets une fois decompresses, extraits des chunks %v :
                %3d image(s)-cle          (type 2) — armes portees, inventaire, bande de slots
                %3d paquet(s) a tir       (type 0, record 105 long)
                %3d paquet(s) a grenade   (type 0, marqueur + identifiant en liste blanche)
                %3d paquet(s) de fenetre  (type 0, 2 s consecutives) — vols de projectile entiers
chunk_02.bin  %d octets — le chunk highlight du film, OCTET POUR OCTET (fil des morts)

CE QUE CETTE BOBINE N EST PAS : un film valide. Les paquets sont concatenes hors de leur
continuite, donc les POSITIONS de biped — qui s accumulent par deltas — y sont sans
signification. Elles sont verrouillees ailleurs, par le fixture d entrees decodees.
`, goldenFilm, goldenFilm, chunk1, s.count, len(s.body), s.sourceChunks,
		s.keyframes, s.fire, s.grenade, s.window, chunk2)
}

// selectMiniFilmPackets choisit les paquets de la bobine. Un paquet peut satisfaire plusieurs
// criteres : il n est retenu QU UNE FOIS, et l ordre du film est conserve.
func selectMiniFilmPackets(dir string) (miniSelection, error) {
	var sel miniSelection
	n := filmdec.CountFilmChunks(dir)
	if n == 0 {
		return sel, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	sel.highlightChunk = n

	firstThrowUS, err := firstGrenadeThrowUS(dir)
	if err != nil {
		return sel, err
	}
	type keep struct {
		chunk, index int
		raw          []byte
		kind         string
	}
	var kept []keep
	srcChunks := map[int]bool{}
	for c := 1; c < n; c++ { // les chunks de REPLICATION ; le dernier est le highlight
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			kind := miniPacketKind(chunk, p, firstThrowUS)
			if kind == "" {
				continue
			}
			start := p.Start - packetHeaderSizeMini
			raw := make([]byte, p.Size+packetHeaderSizeMini)
			copy(raw, chunk[start:p.Start+p.Size])
			kept = append(kept, keep{chunk: c, index: p.Index, raw: raw, kind: kind})
			srcChunks[c] = true
		}
	}
	sort.SliceStable(kept, func(i, j int) bool {
		if kept[i].chunk != kept[j].chunk {
			return kept[i].chunk < kept[j].chunk
		}
		return kept[i].index < kept[j].index
	})
	for _, k := range kept {
		sel.body = append(sel.body, k.raw...)
		sel.count++
		switch k.kind {
		case "keyframe":
			sel.keyframes++
		case "fire":
			sel.fire++
		case "grenade":
			sel.grenade++
		default:
			sel.window++
		}
	}
	for c := range srcChunks {
		sel.sourceChunks = append(sel.sourceChunks, c)
	}
	sort.Ints(sel.sourceChunks)
	return sel, nil
}

// packetHeaderSizeMini est la taille de l en-tete d un paquet de chunk film. Elle est redeclaree
// ici parce que `filmdec` ne l exporte pas : la bobine recopie l en-tete AVEC son payload, sans
// quoi le paquet perdrait son type et son horodatage.
const packetHeaderSizeMini = 16

// miniPacketKind dit pourquoi un paquet est retenu, ou "" s il ne l est pas.
func miniPacketKind(chunk []byte, p filmdec.FilmPacket, throwUS uint64) string {
	if p.Type == filmdec.PacketTypeKeyframe {
		return "keyframe"
	}
	if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
		return ""
	}
	pay := p.Payload(chunk)
	if int(pay[0]>>1) == filmdec.FireEventType && int(pay[0])&1 == 0 {
		return "fire"
	}
	if throwUS > 0 && p.TimestampUS >= throwUS && p.TimestampUS < throwUS+miniFilmWindowUS {
		return "window"
	}
	if hasKnownGrenadeMarker(pay) {
		return "grenade"
	}
	return ""
}

// hasKnownGrenadeMarker dit si le payload porte au moins un lancer reconnu. C est le MEME
// critere que le decodeur (marqueur 24 bits + identifiant en liste blanche), reutilise par sa
// surface publique plutot que recopie.
func hasKnownGrenadeMarker(pay []byte) bool {
	limit := len(pay)*8 - (24 + 32)
	for bp := 0; bp <= limit; bp++ {
		if filmdec.PeekBits(pay, bp, 24) != 0x4C0C00 {
			continue
		}
		if _, ok := filmdec.KnownGrenadeIDs[uint32(filmdec.PeekBits(pay, bp+24, 32))]; ok {
			return true
		}
	}
	return false
}

// firstGrenadeThrowUS rend l instant du premier lancer du film : l origine de la fenetre de
// paquets consecutifs. On l ancre sur un lancer parce que c est le seul instant dont on SAIT
// qu un projectile y nait.
func firstGrenadeThrowUS(dir string) (uint64, error) {
	th, err := filmdec.ScanFilmGrenadeThrows(dir)
	if err != nil {
		return 0, err
	}
	if len(th) == 0 {
		return 0, fmt.Errorf("aucun lancer de grenade dans %s : la fenetre n a pas d ancre", dir)
	}
	best := th[0].TimestampUS
	for _, t := range th {
		if t.TimestampUS < best {
			best = t.TimestampUS
		}
	}
	return best, nil
}

func zlibBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(b); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ---------------------------------------------------------------------------
// Ce que la mini-bobine verrouille — les decodeurs, sur du binaire REEL.
// ---------------------------------------------------------------------------

// TestMiniFilmDecodesTheFireEvents : les 519 records de tir, decodes du binaire.
//
// C EST LE TEST QUE L ETAGE 1 NE POUVAIT PAS ECRIRE. Le fixture d entrees porte 519 evenements
// parce qu on les y a mis ; ici ils sont RELUS du film, avec leur arme et leur tireur.
func TestMiniFilmDecodesTheFireEvents(t *testing.T) {
	ev, err := filmdec.ScanFilmFireEvents(MiniFilmDir)
	if err != nil {
		t.Fatalf("ScanFilmFireEvents : %v", err)
	}
	if len(ev) != wantShotsAvailable {
		t.Errorf("%d evenements de tir decodes, attendu %d — le decodeur du record type 105 a bouge",
			len(ev), wantShotsAvailable)
	}
	withWeapon, withAim := 0, 0
	idx := map[int]int{}
	for _, e := range ev {
		if e.WeaponID != 0 {
			withWeapon++
		}
		if e.HasAim {
			withAim++
		}
		idx[e.FilmIndex]++
		if e.Variant != 0 {
			t.Fatalf("un record COURT a ete emis : il ne porte pas d arme, il n a rien a faire ici")
		}
	}
	if withWeapon != len(ev) {
		t.Errorf("%d evenements sur %d portent une arme : le champ d arme s est deplace",
			withWeapon, len(ev))
	}
	if withAim == 0 {
		t.Error("aucun evenement ne porte de visee : le chemin sur du record vide a disparu")
	}
	if len(idx) != 8 {
		t.Errorf("%d index de tireur distincts, attendu 8 (les huit joueurs de l arene) : %v",
			len(idx), idx)
	}
}

// TestMiniFilmDecodesTheGrenadeThrows : les 70 lancers, avec leur type.
func TestMiniFilmDecodesTheGrenadeThrows(t *testing.T) {
	th, err := filmdec.ScanFilmGrenadeThrows(MiniFilmDir)
	if err != nil {
		t.Fatalf("ScanFilmGrenadeThrows : %v", err)
	}
	if len(th) != wantGrenades {
		t.Errorf("%d lancers decodes, attendu %d", len(th), wantGrenades)
	}
	byKind := map[string]int{}
	for _, g := range th {
		if g.Name() == "" {
			t.Fatalf("lancer de type %08x hors liste blanche : le filtre qui fait la selectivite "+
				"du marqueur a saute", g.TypeID)
		}
		if g.FilmIndex < 0 || g.FilmIndex > 7 {
			t.Errorf("index de lanceur %d hors 0..7 : le champ a +103 bits s est deplace "+
				"(a +102 les valeurs tombent entre 16 et 19)", g.FilmIndex)
		}
		byKind[g.Name()]++
	}
	if len(byKind) != 4 {
		t.Errorf("%d types de grenade lances, attendu 4 : %v", len(byKind), byKind)
	}
}

// TestMiniFilmDecodesProjectileFlights : des VOLS ENTIERS, pas des fragments.
//
// La fenetre de deux secondes est la seule partie de la bobine ou la continuite des paquets est
// preservee ; c est donc la seule ou le decoupage en vies (`splitLives`) se mesure.
func TestMiniFilmDecodesProjectileFlights(t *testing.T) {
	wr := filmdec.Vec3Range{{Min: -100, Max: 100}, {Min: -100, Max: 100}, {Min: -100, Max: 100}}
	tr, err := filmdec.ScanFilmProjectiles(MiniFilmDir, &wr)
	if err != nil {
		t.Fatalf("ScanFilmProjectiles : %v", err)
	}
	if len(tr) == 0 {
		t.Fatal("aucune trajectoire de projectile : la fenetre de paquets consecutifs ne porte " +
			"plus de vol, ou l archetype ti=41 n est plus reconnu")
	}
	for _, p := range tr {
		if len(p.Pts) < 3 {
			t.Fatalf("trajectoire de %d point(s) publiee : deux points ne dessinent pas un vol",
				len(p.Pts))
		}
		for i := 1; i < len(p.Pts); i++ {
			if p.Pts[i].TimestampUS < p.Pts[i-1].TimestampUS {
				t.Fatal("les points d une trajectoire ne sont pas tries par instant")
			}
			if p.Pts[i].TimestampUS-p.Pts[i-1].TimestampUS > 250_000 {
				t.Fatal("un trou de plus de 250 ms subsiste DANS une vie : le decoupage en vies " +
					"ne s applique plus, et deux vols distincts sont concatenes")
			}
		}
	}
}

// TestMiniFilmDecodesTheDeathThread : le fil des morts, du chunk highlight tel quel.
func TestMiniFilmDecodesTheDeathThread(t *testing.T) {
	deaths, err := ScanFilmDeaths(MiniFilmDir)
	if err != nil {
		t.Fatalf("ScanFilmDeaths : %v", err)
	}
	if len(deaths) != wantDeaths {
		t.Errorf("%d morts lues, attendu %d", len(deaths), wantDeaths)
	}
	roster := rosterFromDeaths(deaths)
	if len(roster) != 8 {
		t.Errorf("%d joueurs au roster du fil des morts, attendu 8", len(roster))
	}
	for i := 1; i < len(deaths); i++ {
		if deaths[i].TimeMS < deaths[i-1].TimeMS {
			t.Fatal("le fil des morts n est pas trie par instant")
		}
	}
	named := 0
	for _, d := range deaths {
		if d.Gamertag != "" {
			named++
		}
		if d.XUID == 0 {
			t.Fatal("une mort sans xuid : l identite est le xuid, et sans elle la vie ne se nomme pas")
		}
	}
	if named != len(deaths) {
		t.Errorf("%d morts sur %d portent le gamertag ecrit PAR LE FILM", named, len(deaths))
	}
}

// TestMiniFilmDecodesTheKeyframes : les images-cles servent les armes portees ET l inventaire.
//
// LES DEUX COMPTES SONT CEUX DE L ETAGE 1, ET C EST LE POINT : le fixture d entrees porte 150
// loadouts et 184 inventaires parce qu on les y a mis ; ici ils sont RELUS du binaire. Si les
// deux etages divergeaient, l un des deux serait perime — et on saurait lequel.
func TestMiniFilmDecodesTheKeyframes(t *testing.T) {
	lo, err := filmdec.ScanFilmKeyframeLoadouts(MiniFilmDir, loadoutFamilies())
	if err != nil {
		t.Fatalf("ScanFilmKeyframeLoadouts : %v", err)
	}
	if len(lo) != wantLoadouts {
		t.Errorf("%d loadouts decodes des images-cles, attendu %d", len(lo), wantLoadouts)
	}
	inv, err := ScanFilmKeyframeInventory(MiniFilmDir, loadoutFamilies(), 0)
	if err != nil {
		t.Fatalf("ScanFilmKeyframeInventory : %v", err)
	}
	if len(inv) != wantInventoryRead {
		t.Errorf("%d inventaires decodes des images-cles, attendu %d", len(inv), wantInventoryRead)
	}
	perKeyframe := map[uint64]map[uint32]bool{}
	for _, i := range inv {
		if perKeyframe[i.TimestampUS] == nil {
			perKeyframe[i.TimestampUS] = map[uint32]bool{}
		}
		if perKeyframe[i.TimestampUS][i.Slot] {
			t.Fatalf("deux inventaires pour le slot %d a la meme image-cle : les bornes de record "+
				"se recouvrent", i.Slot)
		}
		perKeyframe[i.TimestampUS][i.Slot] = true
	}
	if len(perKeyframe) < 2 {
		t.Fatalf("%d image(s)-cle exploitable(s) : la bobine n en porte plus assez pour que le "+
			"decodage se mesure", len(perKeyframe))
	}
}

// wantDeaths : 93 morts au fil du film de reference. C est le denominateur du nommage des vies
// (90 vies nommees sur 105) ; les deux ne se confondent pas — une mort peut ne clore aucune vie
// decoupee, et une vie peut n etre close par aucune mort.
const wantDeaths = 93

// wantLoadouts / wantInventoryRead : ce que les images-cles rendent AVANT tout filtrage par
// trace publiee. `wantInventory` (184) est le compte APRES publication ; ici les deux coincident
// parce qu aucun etat n est ecarte sur ce film, mais ils repondent a deux questions distinctes.
const (
	wantLoadouts      = 150
	wantInventoryRead = 184
)
