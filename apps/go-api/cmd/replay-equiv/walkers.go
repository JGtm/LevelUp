package main

// walkers.go — LES QUATRE GRAMMAIRES DE DECOUPAGE, COTE A COTE, POUR LES MESURER.
//
// # CE QUE CE FICHIER EST, ET COMBIEN DE TEMPS IL VIT
//
// Les trois marcheurs de paquets de la chaine de cuisson DIVERGENT (PLAN_CUISSON_PERF §3 D3).
// Le lot 1 les remplace par UNE grammaire ; le plan exige que la purete de ce remplacement soit
// MESUREE sur les 1 380 films du cache AVANT d'etre affirmee. Les fonctions ci-dessous sont
// donc des COPIES DE MESURE, pures et sans effet de bord, des trois marcheurs actuels — plus la
// grammaire unifiee retenue. Elles reproduisent le comportement du depot AU 2026-09-02 :
//
//	marcheFilmdec           <- filmdec.WalkPackets            (analysis/filmdec/film_packets.go:71-89)
//	marcheKillsource        <- killsource.splitPackets        (games/halo_infinite/film/killsource/chunks.go:156-171)
//	marcheObjectiveevents   <- objectiveevents.walkFrames     (analysis/objectiveevents/film.go:120-140)
//	marcheUnifiee           <- la grammaire retenue par D3, REVISEE le 2026-09-02
//
// ELLES SONT SUPPRIMEES AVEC LE MODE `-walkers`, au lot 1 : ce sont des temoins de mesure, pas
// une quatrieme implementation a maintenir. Toute divergence entre ces copies et leur original
// invaliderait la mesure — c'est pourquoi elles sont recopiees a l'identique, y compris les
// tests morts (`size < 0` ne se declenche jamais sur 64 bits, il est garde par fidelite).
//
// # LA GRAMMAIRE UNIFIEE EST CELLE DE D3 *REVISEE* (2026-09-02)
//
// La premiere ecriture de D3 s'arretait sur `size <= 0` sans rien emettre. La mesure 0.7
// (1 378 films) puis le diagnostic paquet par paquet ont montre que, sur les chunks de DONNEES,
// l'UNIQUE paquet de taille 0 est le marqueur CHUNK_END (type 7), en derniere position — cette
// regle-la jetait donc le terminateur que `filmdec` emet. La grammaire retenue est celle de la
// revision, et c'est elle qui est implementee ici :
//
//  1. en-tete de 16 octets ; arret si `off+16+size > len(data)` ;
//  2. le paquet est EMIS, taille 0 comprise ;
//  3. arret APRES avoir emis un paquet de type 7 (le terminateur fait partie de la sortie) ;
//  4. arret AVANT d'emettre un paquet de taille 0 qui n'est PAS de type 7 — l'en-tete degenere
//     qu'on ne trouve que dans `chunk_00`, le REGISTRE, qu'aucun consommateur legitime ne
//     marche comme un flux de paquets.
//
// # LES QUATRE AXES DE DIVERGENCE (D3)
//
//  1. `size == 0` — filmdec l'accepte et avance de 16 ; killsource s'arrete sans emettre ;
//     l'unifiee EMET le terminateur (type 7) puis s'arrete, et s'arrete SANS emettre devant un
//     en-tete degenere (taille 0, type autre que 7).
//  2. `CHUNK_END` (type 7) — objectiveevents avance dessus puis s'arrete, et ne l'emet jamais
//     (il ne rend que le type 0) ; l'unifiee l'EMET puis s'arrete ; filmdec et killsource ne
//     regardent pas le type (killsource s'arrete de fait quand le terminateur porte une taille
//     nulle, mais par la regle `size <= 0`, pas par le type).
//  3. borne haute — objectiveevents borne par `size > len(data)` SANS l'offset, et AVANCE meme
//     quand le payload deborde ; les autres bornent par `off+16+size > len(data)` et s'arretent.
//  4. flux zlib tronque — objectiveevents rend alors le BRUT COMPRESSE (qu'il marche ensuite
//     comme des paquets) ; filmdec et killsource rendent le partiel decompresse.

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// entetePaquet : en-tete de 16 octets little-endian `[u16 type][2][u32 size][u64 ts]`.
const entetePaquet = 16

// typeChunkEnd : le paquet CHUNK_END, sur lequel deux des quatre grammaires s'arretent.
const typeChunkEnd = 7

// paquet : ce qu'une grammaire voit d'un paquet. L'OFFSET EN FAIT PARTIE : deux grammaires qui
// rendent les memes types dans le meme ordre mais a des offsets differents ne voient pas le
// meme film.
type paquet struct {
	off  int
	typ  int
	size int
	ts   uint64
}

// marcheFilmdec — COPIE DE MESURE de filmdec.WalkPackets. Accepte `size == 0` et continue ; ne
// s'arrete que sur la borne haute avec offset.
func marcheFilmdec(chunk []byte) []paquet {
	var out []paquet
	off := 0
	for off+entetePaquet <= len(chunk) {
		size := int(binary.LittleEndian.Uint32(chunk[off+4:]))
		if size < 0 || off+entetePaquet+size > len(chunk) {
			break
		}
		out = append(out, paquet{
			off:  off,
			typ:  int(binary.LittleEndian.Uint16(chunk[off:])),
			size: size,
			ts:   binary.LittleEndian.Uint64(chunk[off+8:]),
		})
		off += entetePaquet + size
	}
	return out
}

// marcheKillsource — COPIE DE MESURE de killsource.splitPackets. S'arrete sur `size <= 0` et
// sur la borne haute avec offset ; ignore CHUNK_END.
func marcheKillsource(d []byte) []paquet {
	var out []paquet
	off := 0
	for off+entetePaquet <= len(d) {
		typ := int(binary.LittleEndian.Uint16(d[off:]))
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz <= 0 || off+entetePaquet+sz > len(d) {
			break
		}
		out = append(out, paquet{off: off, typ: typ, size: sz, ts: ts})
		off += entetePaquet + sz
	}
	return out
}

// marcheObjectiveevents — COPIE DE MESURE de objectiveevents.walkFrames. N'emet QUE le type 0,
// borne par `size > len(data)` SANS l'offset, avance meme quand le payload deborde, et s'arrete
// APRES avoir avance sur un CHUNK_END.
func marcheObjectiveevents(data []byte) []paquet {
	var out []paquet
	off := 0
	for off+entetePaquet <= len(data) {
		typ := int(binary.LittleEndian.Uint16(data[off:]))
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		us := binary.LittleEndian.Uint64(data[off+8:])
		if size < 0 || size > len(data) {
			break
		}
		if typ == 0 && off+entetePaquet+size <= len(data) {
			out = append(out, paquet{off: off, typ: typ, size: size, ts: us})
		}
		off += entetePaquet + size
		if typ == typeChunkEnd {
			break
		}
	}
	return out
}

// marcheUnifiee — LA GRAMMAIRE RETENUE (D3 REVISEE, cf. l'en-tete de ce fichier) : arret sur
// `off+16+size > len` ; le paquet est emis meme a taille 0 ; arret APRES un paquet de type 7 ;
// arret AVANT un paquet de taille 0 qui n'est pas de type 7.
func marcheUnifiee(data []byte) []paquet {
	var out []paquet
	off := 0
	for off+entetePaquet <= len(data) {
		typ := int(binary.LittleEndian.Uint16(data[off:]))
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if size < 0 || off+entetePaquet+size > len(data) {
			break
		}
		// EN-TETE DEGENERE : une taille nulle sur un type autre que le terminateur. On s'arrete
		// AVANT de l'emettre — il ne porte rien, et son seul lieu connu est `chunk_00`, qui est
		// le registre et pas un flux de paquets.
		if size == 0 && typ != typeChunkEnd {
			break
		}
		out = append(out, paquet{
			off: off, typ: typ, size: size, ts: binary.LittleEndian.Uint64(data[off+8:]),
		})
		off += entetePaquet + size
		// LE TERMINATEUR EST DANS LA SORTIE, et il ferme le chunk : rien apres lui n'est lu.
		if typ == typeChunkEnd {
			break
		}
	}
	return out
}

// typeZero restreint un jeu de paquets au type 0 — le seul qu'objectiveevents emet.
func typeZero(in []paquet) []paquet {
	out := make([]paquet, 0, len(in))
	for _, p := range in {
		if p.typ == 0 {
			out = append(out, p)
		}
	}
	return out
}

// inflateMesure decompresse un chunk UNE fois et rend les DEUX vues qui existent aujourd'hui.
//
// C'est le quatrieme axe de D3, et il ne se voit que la : sur un flux tronque, filmdec et
// killsource gardent le PARTIEL decompresse, tandis qu'objectiveevents rend le BRUT COMPRESSE —
// qu'il marche ensuite comme s'il portait des paquets.
func inflateMesure(raw []byte) (dec, vueObjectifs []byte, tronque bool) {
	if len(raw) < 2 || raw[0] != 0x78 {
		return raw, raw, false
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return raw, raw, false
	}
	defer func() { _ = zr.Close() }()
	out, err := io.ReadAll(zr)
	switch {
	case err == nil:
		return out, out, false
	case len(out) == 0:
		// Rien d'exploitable : les trois inflates rendent le brut, aucune divergence possible.
		return raw, raw, false
	default:
		return out, raw, true
	}
}

// axeAuPointDArret nomme POURQUOI la grammaire unifiee s'est arretee la ou une autre a continue.
//
// DEUX LECTURES, DANS CET ORDRE, parce que depuis D3 REVISEE l'unifiee peut s'arreter APRES
// avoir emis (le terminateur de type 7) et non plus seulement AVANT :
//
//  1. le DERNIER paquet emis est de type 7 -> c'est lui qui a ferme le chunk ;
//  2. sinon l'arret s'est fait devant l'en-tete suivant, qu'on relit pour le classer.
func axeAuPointDArret(data []byte, uni []paquet) string {
	off := 0
	if n := len(uni); n > 0 {
		dernier := uni[n-1]
		if dernier.typ == typeChunkEnd {
			return "chunk_end"
		}
		off = dernier.off + entetePaquet + dernier.size
	}
	if off+entetePaquet > len(data) {
		return "fin_de_tampon"
	}
	size := int(binary.LittleEndian.Uint32(data[off+4:]))
	switch {
	case size < 0 || off+entetePaquet+size > len(data):
		return "borne_haute"
	case size == 0:
		// Taille nulle sur un type autre que 7 : l'en-tete degenere de `chunk_00`. Un type 7 a
		// taille nulle, lui, aurait ete EMIS et pris par la branche `chunk_end` ci-dessus.
		return "taille_nulle"
	default:
		return "indetermine"
	}
}

// ligneWalkers : la mesure d'UN film.
type ligneWalkers struct {
	film                      string
	chunks, tronques          int
	divFilmdec, divKS, divObj int
	axe                       string
}

// String rend la ligne TSV du fichier de mesure.
func (l ligneWalkers) String() string {
	axe := l.axe
	if axe == "" {
		axe = "-"
	}
	return fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%d\t%s",
		l.film, l.chunks, l.tronques, l.divFilmdec, l.divKS, l.divObj, axe)
}

// enteteWalkers : la ligne de titre du fichier de mesure (ignoree a la relecture).
const enteteWalkers = "# film\tchunks\ttronques\tdivFilmdec\tdivKillsource\tdivObj\tpremierAxe"

// errAucunChunk : le repertoire du film existe mais ne porte aucun `chunk_NN.bin`. Ce n'est PAS
// un echec de mesure — c'est un film que le cache n'a pas (telechargement interrompu, purge) et
// sur lequel il n'y a rien a comparer. Le parent le compte en ECARTE (cf. CodeSkipped) : le
// confondre avec un echec ferait rendre 1 a une passe qui n'a rien trouve d'anormal.
var errAucunChunk = errors.New("aucun chunk_NN.bin")

// mesureFilm compare, chunk par chunk, ce que voient les quatre grammaires.
func mesureFilm(film, dir string) (ligneWalkers, error) {
	fichiers, err := filepath.Glob(filepath.Join(dir, "chunk_*.bin"))
	if err != nil {
		return ligneWalkers{}, fmt.Errorf("lecture de %s : %w", dir, err)
	}
	if len(fichiers) == 0 {
		return ligneWalkers{}, fmt.Errorf("%w dans %s", errAucunChunk, dir)
	}
	sort.Strings(fichiers)
	l := ligneWalkers{film: film, chunks: len(fichiers)}
	for _, f := range fichiers {
		raw, err := os.ReadFile(f) //nolint:gosec // chemin enumere sous le cache film
		if err != nil {
			return ligneWalkers{}, fmt.Errorf("%s : %w", f, err)
		}
		if compareChunk(raw, &l) {
			l.tronques++
		}
	}
	return l, nil
}

// compareChunk mesure UN chunk et cumule ses divergences dans l. Rend vrai si le flux etait
// tronque. Le chunk est relache des le retour : la mesure ne garde jamais le film entier.
func compareChunk(raw []byte, l *ligneWalkers) bool {
	dec, vueObj, tronque := inflateMesure(raw)
	uni := marcheUnifiee(dec)
	ecarts := []struct {
		diffe bool
		cible *int
	}{
		{!slices.Equal(marcheFilmdec(dec), uni), &l.divFilmdec},
		{!slices.Equal(marcheKillsource(dec), uni), &l.divKS},
		{!slices.Equal(marcheObjectiveevents(vueObj), typeZero(uni)), &l.divObj},
	}
	for i, e := range ecarts {
		if !e.diffe {
			continue
		}
		*e.cible++
		if l.axe != "" {
			continue
		}
		// Le troisieme axe est celui d'objectiveevents : sur flux tronque, il ne marche meme
		// pas les memes octets, et c'est LA la divergence — pas la grammaire.
		if i == 2 && tronque {
			l.axe = "flux_tronque"
			continue
		}
		l.axe = axeAuPointDArret(dec, uni)
	}
	return tronque
}

// enfantWalkers mesure UN film et ecrit sa ligne.
//
// PAS DE VERROU SOLO ICI, et c'est delibere : ce mode LIT et DECOMPRESSE, il ne decode aucune
// entite ECS. Le verrou protege le decodeur (un seul film decode a la fois sur la machine) ;
// l'imposer a une mesure de lecture ferait attendre 1 380 processus derriere une cuisson en
// cours sans rien proteger. La priorite basse et la sentinelle, elles, restent armees.
func enfantWalkers(o options) int {
	filmproc.LowerOwnPriority(outilNom)
	g := filmproc.Arm(outilNom, o.memGiB, func(peak uint64) {
		slog.Error("plafond memoire depasse — mesure abandonnee",
			"pic_octets", peak, "pic_gio", gio(peak), "film", o.film)
		filmproc.EmitPeak(peak)
		os.Exit(filmproc.CodeMemory)
	})
	defer func() {
		g.Disarm()
		filmproc.EmitPeak(g.Peak())
	}()
	cacheRoot := title.NewPathResolver(o.repoRoot).CacheRootDir()
	l, err := mesureFilm(o.film, filmcache.ChunkDir(cacheRoot, o.film))
	if errors.Is(err, errAucunChunk) {
		slog.Warn("film sans chunk — mesure ecartee", "err", err, "film", o.film)
		return filmproc.CodeSkipped
	}
	if err != nil {
		slog.Error("mesure des grammaires impossible", "err", err, "film", o.film)
		return filmproc.CodeFailed
	}
	if err := ecrireLignes(o.out, []string{l.String()}); err != nil {
		slog.Error("ecriture de la mesure", "err", err, "path", o.out, "film", o.film)
		return filmproc.CodeFailed
	}
	return filmproc.CodeOK
}
