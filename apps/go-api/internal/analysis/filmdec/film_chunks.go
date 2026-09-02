package filmdec

// film_chunks.go — LE PONT ENTRE UN FILM CHARGE UNE FOIS ET LES BALAYAGES.
//
// # CE QUE CE FICHIER REMPLACE
//
// Avant le lot 1 de PLAN_CUISSON_PERF, chaque balayage relisait et redecompressait le film
// entier : `CountFilmChunks(dir)` pour savoir jusqu'ou aller, `ReadFilmChunk(dir, c)` pour les
// octets, `WalkPackets(data)` pour les paquets — soit ~36-40 lectures completes du meme film pour
// un seul artefact. Les balayages prennent desormais un `*filmsource.Film` deja charge, et ce
// fichier leur rend les MEMES trois choses, sans disque et sans inflate :
//
//	CountFilmChunks(dir)     ->  FilmChunkNumbers(film)   les numeros des chunks de donnees
//	ReadFilmChunk(dir, c)    ->  FilmChunkAt(film, c)     les octets decompresses du chunk NUMERO c
//	WalkPackets(data)        ->  FilmChunkAt(film, c)     ses paquets, deja decoupes
//
// # LE NUMERO N'EST PAS LA POSITION, ET C'EST TOUT LE POINT
//
// Un balayage publie le NUMERO du chunk dans ses sorties (`Chunk: c`) et le lit dans l'horloge du
// manifeste (`chunkStartMS[c]`). La position dans le film, elle, depend de ce qui est descendu sur
// le disque : une bobine sans `chunk_00` (la fixture `replay/testdata/minifilm_000d5950`) a son
// premier chunk de DONNEES a la position 0. Les deux se confondent sur un cache complet et
// divergent partout ailleurs ; `filmsource.LoadDir` synthetise le numero dans `Meta()[i].Index`,
// et ce fichier ne travaille QUE par numero.
//
// # POURQUOI L'ARRET AU PREMIER TROU
//
// `CountFilmChunks` comptait `chunk_01, chunk_02, ...` et S'ARRETAIT au premier manquant. Le lot 1
// est un refacto PUR — les sorties doivent etre identiques a l'octet sur le corpus d'equivalence —
// donc [FilmChunkNumbers] s'arrete au meme endroit, sciemment. Ce n'est pas la bonne regle dans
// l'absolu (un film dont le chunk 7 manque perd ses chunks 8..N alors qu'ils sont lisibles) :
// c'est l'ANCIENNE regle, conservee le temps que les enveloppes de compatibilite disparaissent.
// Le jour ou elles tombent, cette fonction rend tous les numeros >= 1 et la ligne d'arret saute.

import (
	"errors"

	"levelup/go-api/internal/analysis/filmsource"
)

// Les trois refus d'un film charge. Ils sont NOMMES parce que quinze balayages les rendaient
// chacun sous forme de litteral (`fmt.Errorf("aucun chunk film dans %s", dir)`) : au-dela de deux
// copies, le depot exige une constante (CLAUDE.md n°6). Ils remplacent des messages qui citaient
// le REPERTOIRE — un film charge n'en a plus, et le chemin appartient desormais a l'appelant qui
// a fait le chargement.
var (
	// ErrNoFilmChunk : le film ne porte aucun chunk de DONNEES (numero >= 1).
	ErrNoFilmChunk = errors.New("aucun chunk de donnees dans le film")
	// ErrNoReadableFilmChunk : aucun chunk de donnees n'a pu etre lu (film vide ou tronque).
	ErrNoReadableFilmChunk = errors.New("aucun chunk film lisible")
	// ErrNoRegistryChunk : le film ne porte pas son registre (le chunk NUMERO 0) — bobine
	// partielle ou fixture. Les lecteurs d'archetype le disent plutot que de rendre un registre
	// vide, qui se lirait comme « archetype absent du build ».
	ErrNoRegistryChunk = errors.New("chunk_00 (registre) absent du film")
)

// FilmChunkNumbers rend les NUMEROS des chunks de DONNEES du film (le registre `chunk_00` exclu),
// dans l'ordre, en s'arretant au premier trou de numerotation — l'equivalent exact de
// `for c := 1; c <= CountFilmChunks(dir); c++`.
//
// Film nil ou sans chunk de donnees : tranche vide. Les balayages testent deja `len(...) == 0` et
// rendent leur erreur « aucun chunk film » — un film absent traverse donc les memes portes qu'un
// repertoire vide, sans garde supplementaire.
//
// FILM SANS METADONNEES (un [filmsource.Load] sur des chunks en memoire, sans manifeste) : la
// POSITION vaut le numero, convention d'une source qui commence au registre — c'est la forme d'un
// pipeline qui vient de telecharger `chunk_00..chunk_NN`.
func FilmChunkNumbers(f *filmsource.Film) []int {
	if f == nil || f.NumChunks() == 0 {
		return nil
	}
	meta := f.Meta()
	if len(meta) == 0 {
		out := make([]int, 0, f.NumChunks()-1)
		for c := 1; c < f.NumChunks(); c++ {
			out = append(out, c)
		}
		return out
	}
	out := make([]int, 0, len(meta))
	want := 1
	for i, m := range meta {
		if i >= f.NumChunks() {
			break
		}
		if m.Index < 1 {
			continue // le registre (0), ou un fichier dont le nom ne porte pas de numero
		}
		if m.Index != want {
			break // trou de numerotation : meme arret que l'ancien CountFilmChunks
		}
		out = append(out, m.Index)
		want++
	}
	return out
}

// FilmChunkAt rend les octets DECOMPRESSES du chunk de NUMERO `num` et ses paquets deja decoupes.
// ok=false quand le film ne porte pas ce numero — le remplacant exact de l'erreur de
// `ReadFilmChunk`, que les balayages traitent par `continue` (un film peut etre partiel, et
// `bipedSlotBand` demande deliberement le chunk d'apres le dernier).
//
// LES PAQUETS SONT CONVERTIS A CHAQUE APPEL, et c'est voulu : la conversion est un decodage
// d'en-tetes de 16 octets deja fait par `filmsource`, sans allocation de payload (les payloads
// sont des sous-tranches du chunk). Elle coute ce que coutait `WalkPackets` ; ce qui disparait,
// c'est la lecture disque et l'inflate, qui pesaient trois ordres de grandeur de plus.
func FilmChunkAt(f *filmsource.Film, num int) ([]byte, []FilmPacket, bool) {
	pos := filmChunkPos(f, num)
	if pos < 0 {
		return nil, nil, false
	}
	return f.Chunk(pos), filmPacketsOf(f.Packets(pos)), true
}

// FilmRegistryChunk rend les octets DECOMPRESSES du registre (le chunk NUMERO 0). ok=false quand
// le film n'en porte pas — une bobine partielle, une fixture : les lecteurs de registre rendent
// alors leur erreur habituelle, pas un registre vide qui se lirait comme un archetype absent.
func FilmRegistryChunk(f *filmsource.Film) ([]byte, bool) {
	pos := filmChunkPos(f, 0)
	if pos < 0 {
		return nil, false
	}
	return f.Chunk(pos), true
}

// filmRegistry lit et analyse le registre du film (chunk NUMERO 0). LE SEUL lecteur de registre
// du paquet : les six accesseurs d'archetype (`bipedArchetype`, `EquipmentArchetypeOf`,
// `groundWeaponArchetype`, `filmArchetype`, `objectiveArchetype`, `managedPropertyArchetype`) en
// derivent tous, chacun ne gardant que son message d'archetype manquant.
//
// LE REGISTRE EST RE-ANALYSE A CHAQUE APPEL, comme avant le lot 1 : le decodage, lui, ne se fait
// plus qu'une fois (le chunk est deja decompresse). Le parse unique est le lot 2 du plan
// (`FilmContext.Registry`) — le poser ici melangerait deux lots.
func filmRegistry(f *filmsource.Film) (*Registry, error) {
	raw, ok := FilmRegistryChunk(f)
	if !ok {
		return nil, ErrNoRegistryChunk
	}
	return ParseRegistryChunk(raw)
}

// filmChunkPos traduit un NUMERO de chunk en position dans le film, ou -1.
func filmChunkPos(f *filmsource.Film, num int) int {
	if f == nil {
		return -1
	}
	meta := f.Meta()
	if len(meta) == 0 {
		// Sans metadonnees, la position vaut le numero (cf. FilmChunkNumbers).
		if num >= 0 && num < f.NumChunks() {
			return num
		}
		return -1
	}
	for i, m := range meta {
		if i >= f.NumChunks() {
			break
		}
		if m.Index == num {
			return i
		}
	}
	return -1
}

// filmPacketsOf traduit les paquets de `filmsource` dans la forme historique [FilmPacket], que
// toute la grammaire de records consomme (`pk.Payload(chunk)`, `pk.Size`, `pk.TimestampUS`).
//
// LES BORNES SE RECONSTRUISENT PAR CONTIGUITE, et c'est exact : la grammaire de `filmsource`
// avance de `16 + taille` a chaque paquet emis et ne SAUTE jamais d'octet — ses deux regles
// d'arret (en-tete qui deborde, en-tete degenere) arretent la marche, elles ne la font pas
// avancer. `TestFilmChunkAtEgaleWalkPackets` le verifie sur un vrai chunk, borne comprise.
func filmPacketsOf(pkts []filmsource.Packet) []FilmPacket {
	if len(pkts) == 0 {
		return nil
	}
	out := make([]FilmPacket, len(pkts))
	off := 0
	for i, p := range pkts {
		out[i] = FilmPacket{
			Index:       p.Index,
			Type:        uint16(p.Type),
			Start:       off + packetHeaderSize,
			Size:        len(p.Payload),
			TimestampUS: p.TS,
		}
		off += packetHeaderSize + len(p.Payload)
	}
	return out
}
