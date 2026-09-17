//go:build research

package grammar

// reapparition_37_bassin_film_research_test.go — LOT 3.7, VOIE LIBRE : LE BASSIN DE MINUTEURS LU
// SUR FILM ENTIER.
//
// # CE QUE CET INSTRUMENT PORTE, ET POURQUOI IL LE PORTE ICI
//
// La mesure du 2026-09-17 sur les mini-bobines a etabli que l entite du moteur de jeu est
// presente a CHAQUE image-cle et que 113 records sur 113 butent sur le MEME composant non porte :
// `i11 game-engine-soft-ceilings-component`. Lire `i15 managed-engine-timers-component` — le
// BASSIN — demande donc de franchir les trois composants non portes qui le precedent. Leurs
// grammaires ont ete relevees chez l ecrivain par `film/research/reapparition/` puis lues a
// `objdump` (note 3.7 § 2 bis) :
//
//	i11  game-engine-soft-ceilings-component               FUN_14116d1ac
//	     -> FUN_1406d676c(flux, _, dest, n = 0x80) = R(128) plat, inconditionnel.
//	i13  game-engine-disabled-kill-volume-flags-component  FUN_142f03498
//	     -> R(13) = n, puis n x R(1) (boucle 142f03579..142f035be, un bit par volume).
//	i14  GameEngineComposerLetterboxComponent              FUN_142f0328c, NIVEAU 2
//	     -> `cmp $0x2,%r9d ; jb` : le niveau du registre vaut 2 (ecs_table), donc la branche
//	        haute. R(1) ; R(16) quantifie ; 4 x FUN_142efd284 (porte INVERSEE R(1), si 0 -> R(7)) ;
//	        4 x [R(1) ; si 1 -> R(16)].
//	i15  managed-engine-timers-component                   FUN_1407ee7b8
//	     -> R(64) de masque ; par fente presente, R(2) d etiquette, puis :
//	        etiquette 0      : rien ;
//	        etiquette 1      : R(16)+R(16)+R(5)+R(16), bornes [0, 3600] s ;
//	        etiquette 2 ou 3 : R(16)+R(16)+R(5),      bornes [0, 36000] s.
//
// CES QUATRE LECTURES NE SONT PAS EN PRODUCTION, ET NE DOIVENT PAS L ETRE PAR CE LOT. 3.7 est un
// lot de RECHERCHE : elles vivent dans ce fichier de test, sous le tag `research`, le temps de
// MESURER. Le port est un lot post-M4 (§ 8 de la note) et il devra, lui, passer par le ratchet
// de fermeture 0.A.3.
//
// # L ORACLE QUI DIT SI LA LECTURE EST JUSTE — ET IL EST GRATUIT
//
// Un record d image-cle est BORNE : le balayeur rend le bit du record suivant. Si les quatre
// grammaires ci-dessus sont justes, la marche ATTERRIT EXACTEMENT sur cette frontiere. La
// fermeture est donc l oracle, et elle ne coute rien. Une grammaire fausse d un seul bit ne
// ferme pas.
//
// # REGIME
//
//	REAP_FILM_ROOT=<repo>/data/cache/film_chunks \
//	REAP_FILMS="bcb6d393,fb1a1a72" \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	    -run Reapparition37Bassin -v -timeout 60m
//
// UN SEUL DECODAGE A LA FOIS PAR PROCESS (hook global) ; un film a la fois, jamais de balayage
// du corpus. Sans `REAP_FILMS`, le test SKIP.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// Les largeurs relevees chez l ecrivain (cf. l en-tete). Nommees pour qu un port futur les
// reprenne telles quelles, et pour qu aucun litteral ne se promene dans la marche.
const (
	reap37SoftCeilingsBits   = 128 // i11 : FUN_1406d676c(n = 0x80)
	reap37KillVolumeCptBits  = 13  // i13 : le compte, puis un bit par volume
	reap37LetterboxQuantBits = 16  // i14 : le reel quantifie
	reap37LetterboxSlots     = 4   // i14 : deux boucles de quatre
	reap37LetterboxOptBits   = 7   // i14 : FUN_142efd284, porte INVERSEE
	reap37BassinFentes       = 64  // i15 : le masque
	reap37BassinTagBits      = 2   // i15 : l etiquette de fente
	reap37MinuteurBits       = 16  // i15 : les deux bornes du minuteur
	reap37MinuteurQueueBits  = 5   // i15 : la queue commune
	reap37IndexBassin        = 15  // i15 : l index du bassin dans la table des composants
)

// reap37BorneHeure est la borne haute de l echelle de l etiquette 1, lue en `.rdata`
// (`DAT_143cd86b4 = 0x45610000`). Celle de l etiquette 2/3 est `RoundTimerMax`, DEJA au depot
// (`quantize_endpoint.go`) : on la reutilise au lieu d en poser une copie.
const reap37BorneHeure float32 = 3600

// reap37Fente est UNE fente du bassin, telle que le film l ecrit.
type reap37Fente struct {
	Index    int
	Tag      uint64
	A, B     float32
	QA, QB   uint16
	Queue    uint8
	Trois    float32 // present a l etiquette 1 seulement
	HasTrois bool
}

// reap37Lecture est le bassin d UNE image-cle.
type reap37Lecture struct {
	TI uint32
	// Queue nomme le composant non porte rencontre APRES le bassin, ou est vide quand la
	// marche est allee au bout. Une queue inconnue n invalide pas le bassin (cf. la regle de
	// V13 dans `reap37LireMoteur`), elle interdit seulement la fermeture.
	Queue        string
	TimestampUS  uint64
	Masque       uint64
	Fentes       []reap37Fente
	Ferme        bool
	FinBit       int
	FrontiereBit int
}

func TestReapparition37BassinSurFilmEntier(t *testing.T) {
	racine, films := reap37Corpus(t)
	for _, court := range films {
		t.Run(court, func(t *testing.T) { reap37UnFilmBassin(t, racine, court) })
	}
}

// reap37Corpus lit la garde d environnement. SKIP propre si elle manque.
func reap37Corpus(t *testing.T) (string, []string) {
	t.Helper()
	racine := os.Getenv("REAP_FILM_ROOT")
	liste := os.Getenv("REAP_FILMS")
	if racine == "" || liste == "" {
		t.Skip("REAP_FILM_ROOT et REAP_FILMS requis — aucun film ouvert sans eux")
	}
	var out []string
	for _, s := range strings.Split(liste, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		t.Skip("REAP_FILMS vide")
	}
	return racine, out
}

func reap37UnFilmBassin(t *testing.T, racine, court string) {
	t.Helper()
	film, err := source.LoadDir(filepath.Join(racine, court), nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", court, err)
	}
	fc := contexteDeBobine(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre %s : %v", court, err)
	}
	tis := reap37ArchetypesMoteur(reg)
	if len(tis) == 0 {
		t.Fatalf("%s : aucun archetype ne declare `managed-engine-timers-component`", court)
	}
	for _, ti := range tis {
		a, _ := reg.Archetype(int(ti))
		t.Logf("FILM %s — archetype moteur CANDIDAT ti=%d, %d composants ; i15 = %q",
			court, ti, len(a.Components), a.Components[15])
	}

	var lectures []reap37Lecture
	bloquants := map[string]int{}
	for _, num := range fc.ChunkNumbers() {
		data, pks, okc := fc.ChunkAt(num)
		if !okc {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, b := range keyframeBornes(pay) {
				if !reap37Contient(tis, uint32(b.TI)) { //nolint:gosec // TI borne par le format (< 50)
					continue
				}
				l, bloc := reap37LireMoteur(pay, b.Bit, b.Want, reg, uint32(b.TI), contexteDInstrument()) //nolint:gosec // idem
				l.TI = uint32(b.TI)                                                                       //nolint:gosec // idem
				if bloc != "" {
					bloquants[bloc]++
					continue
				}
				l.TimestampUS = pk.TimestampUS
				lectures = append(lectures, l)
			}
		}
	}
	reap37PublierBassin(t, lectures, bloquants)
	reap37JoindreIndex(t, fc, reap37UnionDesMasques(lectures))
}

// reap37UnionDesMasques rend l UNION des masques de fente observes : l ensemble des fentes que
// le bassin a ALLOUEES pendant le match.
func reap37UnionDesMasques(l []reap37Lecture) uint64 {
	var u uint64
	for _, x := range l {
		u |= x.Masque
	}
	return u
}

// reap37JoindreIndex EST LA JOINTURE DU LOT : les index de minuteur de `ti=11 i0` designent-ils
// des fentes REELLEMENT ALLOUEES du bassin ?
//
// # POURQUOI CETTE JOINTURE TRANCHE UNE RESERVE OUVERTE DEPUIS LE 2026-09-01
//
// `objectif_ti11_minuteurs_verdict_test.go` a mesure que le quantum de `ti=11 i0` est TOUJOURS
// PAIR sur 1 149 lectures d image-cle, et a laisse DEUX lectures ouvertes sans pouvoir les
// separer :
//
//	(a) `q - 1`     — la lecture du depot (`ObjectiveTimerValue`) : index 15, 17, 19, 21, 23.
//	(b) `q/2 - 1`   — la boucle de composants demarrerait UN BIT TROP LOIN : index 7, 8, 9, 10, 11.
//
// La note du verdict ecrit que « la legalite ne tranchera pas ; la CONTIGUITE, si ». Le bassin
// donne mieux qu un argument de contiguite : il donne l ENSEMBLE DES FENTES QUI EXISTENT. Un
// index qui pointe une fente hors du masque ne designe rien. C est un oracle EXTERIEUR a ti=11.
func reap37JoindreIndex(t *testing.T, fc *FilmContext, union uint64) {
	t.Helper()
	vues := reap37LireIndexTi11(fc)
	t.Logf("  JOINTURE — union des masques de fente = %#016x (%d fente(s) allouee(s))",
		union, reap37Popcount(union))
	if len(vues) == 0 {
		t.Logf("  JOINTURE : aucune lecture d image-cle de ti=11 i0 sur ce film")
		return
	}
	dedansA, dedansB, total := 0, 0, 0
	for p, n := range vues {
		ia := reap37Lecture37A(p.a)
		ib := reap37Lecture37A(p.b)
		ja := reap37Lecture37B(p.a)
		jb := reap37Lecture37B(p.b)
		t.Logf("      quanta (%3d, %3d) x%-4d  lecture (a) q-1 = (%3d, %3d) %s  |  lecture (b) q/2-1 = (%3d, %3d) %s",
			p.a, p.b, n, ia, ib, reap37Dedans(union, ia, ib), ja, jb, reap37Dedans(union, ja, jb))
		total += n
		if reap37FenteAllouee(union, ia) || reap37FenteAllouee(union, ib) {
			dedansA += n
		}
		if reap37FenteAllouee(union, ja) || reap37FenteAllouee(union, jb) {
			dedansB += n
		}
	}
	t.Logf("  JOINTURE — lectures dont AU MOINS un index tombe sur une fente allouee : "+
		"lecture (a) %d/%d, lecture (b) %d/%d", dedansA, total, dedansB, total)
}

// reap37Lecture37A applique la lecture du depot : `ObjectiveTimerValue(q) = q - 1`.
func reap37Lecture37A(q int) int { return ObjectiveTimerValue(uint64(q)) } //nolint:gosec // quantum 7 bits

// reap37Lecture37B applique l hypothese (b) du verdict : la boucle demarre un bit trop loin,
// donc le quantum lu vaut `((vrai & 0x3F) << 1) | bit voisin` et le vrai index est `q/2 - 1`.
func reap37Lecture37B(q int) int { return q/2 - 1 }

// reap37FenteAllouee dit si l index designe une fente du bassin. Les valeurs hors [0, 63] ne
// sont pas des fentes : -1 = « aucun minuteur », 65/66/67 = les trois minuteurs reserves.
func reap37FenteAllouee(union uint64, idx int) bool {
	if idx < 0 || idx >= reap37BassinFentes {
		return false
	}
	return union&(uint64(1)<<uint(idx)) != 0
}

func reap37Dedans(union uint64, a, b int) string {
	switch {
	case reap37FenteAllouee(union, a) && reap37FenteAllouee(union, b):
		return "[les DEUX allouees]"
	case reap37FenteAllouee(union, a) || reap37FenteAllouee(union, b):
		return "[une allouee]"
	default:
		return "[AUCUNE allouee]"
	}
}

// reap37Paire est un couple de quanta de `ti=11 i0`, tel que le flux l ecrit.
type reap37Paire struct{ a, b int }

// reap37LireIndexTi11 lit `ti=11 i0` PAR LA MEME MARCHE que le bassin, et non par
// `ScanObjectives`.
//
// POURQUOI PAS `ScanObjectives`. Son balayage d image-cle ne rend AUCUNE lecture de `i0` sur
// `bcb6d393` (mesure du 2026-09-17), alors que la bobine de ce film porte 85 records `ti=11` :
// il ancre sur la bande de slots OBSERVEE et ne voit pas les records que le balayeur borne ici.
// Lire `i0` avec la marche de ce fichier garantit surtout une chose qui compte davantage : le
// MEME cadre d image-cle, le MEME etat par defaut et la MEME frontiere que la lecture du bassin
// a laquelle on le joint. Deux cadres differents ne se joignent pas.
//
// `i0` est le PREMIER composant apres l en-tete, et le bloquant de `ti=11` est `i4` : la lecture
// est donc acquise sans rien porter (regle de V13, deja appliquee au bassin).
func reap37LireIndexTi11(fc *FilmContext) map[reap37Paire]int {
	vues := map[reap37Paire]int{}
	reg, err := fc.Registry()
	if err != nil {
		return vues
	}
	ti, ok := reap37ArchetypeObjectif(reg)
	if !ok {
		return vues
	}
	for _, num := range fc.ChunkNumbers() {
		data, pks, okc := fc.ChunkAt(num)
		if !okc {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, b := range keyframeBornes(pay) {
				if uint32(b.TI) != ti { //nolint:gosec // TI borne par le format (< 50)
					continue
				}
				br := LecteurSur(pay)
				br.PoserContexte(contexteDInstrument())
				br.SetBitPos(b.Bit + br.cadre().EnTeteBits)
				if !consumeFullStateDefaultBlock(br, ti, false) {
					continue
				}
				v0 := int(br.ReadBits(objectiveTimerBits))
				v1 := int(br.ReadBits(objectiveTimerBits))
				vues[reap37Paire{v0, v1}]++
			}
		}
	}
	return vues
}

// reap37ArchetypeObjectif trouve `ti=11` par le NOM de son composant i0, jamais par un index.
func reap37ArchetypeObjectif(reg *Registry) (uint32, bool) {
	for ti := 0; ti < objectArchetypeCount; ti++ {
		a, ok := reg.Archetype(ti)
		if !ok || len(a.Components) == 0 {
			continue
		}
		if a.Components[0] == "managed-objective-timers-component" {
			return uint32(ti), true //nolint:gosec // ti < 50
		}
	}
	return 0, false
}

func reap37Popcount(v uint64) int {
	n := 0
	for v != 0 {
		v &= v - 1
		n++
	}
	return n
}

// reap37ArchetypesMoteur rend TOUS les archetypes qui declarent le bassin en i15 — jamais un
// index en dur, et jamais le PREMIER trouve.
//
// POURQUOI TOUS. La mesure du 2026-09-17 disait « ti=2 sur les builds du corpus, ti=0 sur
// d autres ». C est plus subtil, et la premiere passe sur film entier l a montre : `bcb6d393`
// declare le bassin a `ti=0` (27 composants) ET a `ti=2` (18), et l entite VIVANTE est celle de
// `ti=2` — `ti=0` n a qu UN record, dont le mot de taille `n2` vaut zero, donc sans aucun
// composant. Choisir le premier candidat aurait rendu « 0 record marche » et laisse croire que
// le bassin n est pas dans le film. On marche donc les deux, et le compte par archetype dit
// laquelle des deux entites le jeu emploie.
func reap37ArchetypesMoteur(reg *Registry) []uint32 {
	var out []uint32
	for ti := 0; ti < objectArchetypeCount; ti++ {
		a, ok := reg.Archetype(ti)
		if !ok || len(a.Components) <= 15 {
			continue
		}
		if a.Components[15] == "managed-engine-timers-component" {
			out = append(out, uint32(ti)) //nolint:gosec // ti < 50
		}
	}
	return out
}

func reap37Contient(l []uint32, v uint32) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

// reap37LireMoteur marche UN record d image-cle du moteur de jeu : les composants portes par le
// depot, les trois grammaires relevees au lot 3.7, puis le bassin. Rend le nom du composant
// bloquant quand la marche ne peut pas avancer.
func reap37LireMoteur(pay []byte, recBit, frontiere int, reg *Registry, ti uint32,
	ctx ContexteDeLecture) (reap37Lecture, string) {
	br := LecteurSur(pay)
	br.PoserContexte(ctx)
	br.SetBitPos(recBit + br.cadre().EnTeteBits)
	arch, ok := reg.Archetype(int(ti))
	if !ok {
		return reap37Lecture{}, "archetype absent du registre"
	}
	if !consumeFullStateDefaultBlock(br, ti, false) {
		return reap37Lecture{}, "n2 nul : aucun composant"
	}
	l := reap37Lecture{FrontiereBit: frontiere}
	for i, nom := range arch.Components {
		switch nom {
		case "game-engine-soft-ceilings-component":
			br.Skip(reap37SoftCeilingsBits)
		case "game-engine-disabled-kill-volume-flags-component":
			reap37LireVolumes(br)
		case "GameEngineComposerLetterboxComponent":
			reap37LireLetterbox(br)
		case "managed-engine-timers-component":
			l.Masque, l.Fentes = reap37LireBassin(br)
		default:
			if _, _, _, porte := consumeByNameCapturing(br, nom, ti, arch.Level(i)); !porte {
				// LA REGLE DE V13, APPLIQUEE ICI : un bloquant SITUE APRES le bassin ne
				// disqualifie pas la lecture du bassin. `DesyncAt` est l index du PREMIER
				// composant present non porte, donc tout ce qui precede a ete consomme dans
				// l ordre (`NOTE_V13_DEADSTATE_VEHICULE` § 3). La lecture sort, avec le nom
				// de sa QUEUE inconnue, et la fermeture ne peut evidemment pas etre atteinte.
				if i > reap37IndexBassin {
					l.Queue = fmt.Sprintf("i%d %s", i, nom)
					l.FinBit = br.BitPos()
					return l, ""
				}
				return reap37Lecture{}, fmt.Sprintf("i%d %s", i, nom)
			}
		}
	}
	l.FinBit = br.BitPos()
	l.Ferme = frontiere >= 0 && l.FinBit == frontiere
	return l, ""
}
