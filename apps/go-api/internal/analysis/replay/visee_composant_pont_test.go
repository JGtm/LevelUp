package replay

// visee_composant_pont_test.go — LOT F : LE PONT SLOT -> JOUEUR, ET LA CALIBRATION DE LA MARCHE.
//
// DEUX PREALABLES, ET AUCUN N'EST UNE HYPOTHESE.
//
// 1. QUELS SLOTS SONT CEUX DU JOUEUR. La marche de production rend des records indexes par SLOT
//    d'entite ; le releve Theater, lui, parle d'un GAMERTAG. Le pont est celui des instruments
//    des phases 6 et A, repris sans modification : les positions bipedes decoupees en vies
//    (`buildLifeSpans`), le decalage feed->film resolu par les fins de vie (`bestDeathOffset`),
//    puis le nommage par le fil des morts (`nameLivesByDeaths`) — QUI NE NOMME UNE VIE QUE PAR
//    LA MORT QUI LA TERMINE. Les fragments anterieurs de la meme vie physique restent donc
//    anonymes, et la plage etiquetee (debut de match) tombe justement dedans : d'ou la
//    PROPAGATION par meme-slot, deja ecrite dans `visee_chronologie_research_test.go` et reprise
//    ici a l'identique (un slot ne se recycle qu'apres mort puis reapparition, donc deux
//    fragments de meme slot separes de moins de 30 s portent le meme joueur).
//
// 2. LE CHEMIN SEQUENTIEL A ETE ESSAYE, MESURE ET ECARTE — le tableau est garde ici parce qu'un
//    negatif de methode se publie aussi. `DecodeFrameViews` (la marche non ancree) depend d'un
//    parametre de RUNTIME absent du film, `FrameConfig.IDLowBits`, dont le depot dit lui-meme
//    qu'il « differe d'un film a l'autre » et que le 13 par defaut « n'est qu'une hypothese de
//    depart, PAS une constante du format ». Balaye sur 10..14, il rend au mieux SEPT records
//    bipedes sur les trois premiers chunks de 00162144, et ZERO sur les slots cibles (execution 1
//    du lot F, 2026-08-30) — quand le chemin ANCRE en rend 141 045 sur le meme film. La collecte
//    passe donc par `ScanBipedRecords` + `ConsumeComponentAt`, et `vfCalibre` ne sert plus qu'a
//    republier ce constat sur demande.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// vfPropagationUS : ecart maximal entre deux fragments de meme slot consideres comme la meme
// vie physique. Meme valeur que la propagation de `visee_chronologie_research_test.go`.
const vfPropagationUS = 30_000_000

// vfIDLowBitsCandidats : les largeurs d'identifiant bas balayees par la calibration. La plage
// couvre les deux valeurs mesurees du dossier (11 et 14) et leurs voisines.
var vfIDLowBitsCandidats = []int{10, 11, 12, 13, 14}

// vfPont porte le resultat du pont : les slots du joueur mesure, le decalage feed->film mesure
// sur CE film, et les vies retenues.
type vfPont struct {
	xuid  uint64
	slots map[uint32]bool
	// offMS : decalage mesure par le pont des morts. Publie et COMPARE a la dependance figee
	// du dossier (sig114OffsetMS) — s'ils divergent, la chronologie ne s'applique pas.
	offMS     int64
	apparies  int
	positions int
	// vies : les vies NOMMEES du joueur. Le filtrage par slot seul ne suffit pas — un slot
	// MIGRE aux reapparitions, donc deux joueurs successifs peuvent porter le meme numero. Les
	// bornes de vie sont ce qui rattache un record a une personne.
	vies []lifeSpan
	// anonymes : les fragments non nommes qui recouvrent la fenetre d'analyse, et morts : les
	// instants de mort du joueur sur l'horloge du film. Publies ensemble : c'est leur
	// confrontation qui dit si un fragment anonyme est une vie du joueur.
	anonymes []lifeSpan
	morts    []int64
	// rattaches : le compte rendu du rattachement des fragments anonymes.
	rattaches []vfRattachement
}

// contient dit si le record (slot, tMS) tombe dans une vie du joueur.
func (p vfPont) contient(slot uint32, tMS int64) bool {
	us := tMS * 1000
	for _, l := range p.vies {
		if l.slot == slot && us >= l.from && us <= l.to {
			return true
		}
	}
	return false
}

// vfXUID resout un gamertag en xuid par le fil d'evenements du film.
func vfXUID(dir, gt string) (uint64, error) {
	n := filmdec.CountFilmChunks(dir)
	raw, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		return 0, fmt.Errorf("chunk d'evenements : %w", err)
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		return 0, fmt.Errorf("feed illisible : %w", err)
	}
	for _, e := range evs {
		if e.Gamertag == gt {
			return e.XUID, nil
		}
	}
	return 0, fmt.Errorf("gamertag %q absent du feed", gt)
}

// vfPropage etend le nommage aux fragments anonymes de meme slot. Repris a l'identique de
// l'instrument de chronologie : le nommage par les morts laisse anonymes les fragments qui ne
// se terminent pas par une mort, et la plage etiquetee est precisement de ceux-la.
func vfPropage(lives []lifeSpan) {
	for propage := true; propage; {
		propage = false
		for i := range lives {
			if lives[i].xuid != 0 {
				continue
			}
			for j := range lives {
				if lives[j].xuid == 0 || lives[j].slot != lives[i].slot {
					continue
				}
				ecart := lives[j].from - lives[i].to
				if ecart < 0 {
					ecart = lives[i].from - lives[j].to
				}
				if ecart >= 0 && ecart <= vfPropagationUS {
					lives[i].xuid = lives[j].xuid
					propage = true
					break
				}
			}
		}
	}
}

// LE RATTACHEMENT DES FRAGMENTS ANONYMES — LE GESTE QUI DEBLOQUE LE LOT F, ET SON CRITERE.
//
// Mesure du 2026-08-30 sur 00162144 : la premiere mort de Nilton410 tombe a 1 333 133 ms de
// film, et le fragment anonyme du slot 513 court sur [1 211,1 ; 1 337,4] s. Sa fin depasse la
// mort de 4,3 s — le corps reste replique quelques secondes apres le coup fatal — donc
// `nameLivesByDeaths`, qui exige un appariement serre, ne le nomme pas. Or c'est EXACTEMENT le
// fragment que le releve Theater etiquette : sans lui, la fenetre d'analyse ne contient aucun
// record du joueur, et le lot F ne mesurerait rien.
//
// LE CRITERE EST DOUBLE ET IL EXIGE L'UNICITE — un seul des deux volets serait un vote :
//
//	(a) une MORT du joueur tombe a moins de `vfRattacheEcartMS` de la fin du fragment ;
//	(b) une VIE NOMMEE de ce meme joueur commence dans les `vfRattacheRespawnMS` qui suivent
//	    cette fin — c'est la reapparition, et elle porte un AUTRE slot (le slot migre) ;
//	(c) AUCUNE mort du joueur ne tombe A L'INTERIEUR du fragment. Ce volet n'est pas un reglage,
//	    c'est une impossibilite : une vie ne contient pas la mort de son porteur. Il a ete
//	    ajoute apres mesure — (a) et (b) seuls laissaient 2 candidats sur le fragment du slot
//	    513 et 3 sur celui du slot 520, parce que plusieurs joueurs meurent dans la meme
//	    fenetre de 6 s. C'est un DURCISSEMENT du critere, jamais un elargissement : il ne peut
//	    que retirer des candidats.
//
// Un fragment n'est rattache que si UN SEUL joueur satisfait les trois. Le nombre de candidats
// est publie fragment par fragment : deux candidats, le fragment reste anonyme, et on le voit.
//
// LES DEUX FENETRES SONT BORNEES PAR LA DONNEE, PAS AJUSTEES SUR ELLE. 6 s couvre la
// replication post-mortem ; la reapparition suivante de Nilton arrive 8,1 s apres sa mort, donc
// une fenetre de 6 s ne peut pas confondre une mort avec le debut de la vie d'apres. Les ecarts
// REELLEMENT mesures sont publies a cote du rattachement, pour que le lecteur juge la marge.
const (
	vfRattacheEcartMS   = 6000
	vfRattacheRespawnMS = 15000
)

// vfRattachement est le compte rendu du rattachement d'UN fragment anonyme.
type vfRattachement struct {
	slot                    uint32
	from, to                int64
	xuid                    uint64
	ecartMort, ecartRespawn int64
	candidats               int
}

// vfRattacheAnonymes nomme les fragments anonymes qui satisfont le double critere, et rend le
// compte rendu de CHAQUE fragment — rattache ou non.
func vfRattacheAnonymes(lives []lifeSpan, deaths []Death, off int64) []vfRattachement {
	var out []vfRattachement
	for i := range lives {
		if lives[i].xuid != 0 {
			continue
		}
		r := vfCandidatsDe(lives, lives[i], deaths, off)
		if r.candidats == 1 {
			lives[i].xuid = r.xuid
		}
		out = append(out, r)
	}
	return out
}

// vfCandidatsDe compte les joueurs qui satisfont les deux volets du critere pour un fragment.
func vfCandidatsDe(lives []lifeSpan, frag lifeSpan, deaths []Death, off int64) vfRattachement {
	r := vfRattachement{slot: frag.slot, from: frag.from, to: frag.to}
	fin := frag.to / 1000
	vus := map[uint64]bool{}
	for _, d := range deaths {
		t := d.TimeMS + off
		if absI64(t-fin) > vfRattacheEcartMS || vus[d.XUID] {
			continue
		}
		respawn := vfRespawnApres(lives, d.XUID, fin)
		if respawn < 0 || vfMortDedans(deaths, d.XUID, off, frag.from/1000, fin-vfRattacheEcartMS) {
			continue
		}
		vus[d.XUID] = true
		r.candidats++
		r.xuid, r.ecartMort, r.ecartRespawn = d.XUID, t-fin, respawn
	}
	return r
}

// vfMortDedans dit qu'une mort du joueur tombe strictement a l'interieur de [debut ; fin] —
// le volet (c) du critere. Le fragment ne peut alors pas etre une vie de ce joueur.
func vfMortDedans(deaths []Death, xuid uint64, off, debut, fin int64) bool {
	for _, d := range deaths {
		if d.XUID != xuid {
			continue
		}
		if t := d.TimeMS + off; t > debut && t < fin {
			return true
		}
	}
	return false
}

// vfRespawnApres rend le delai jusqu'a la prochaine vie NOMMEE du joueur, ou -1 s'il n'y en a
// pas dans la fenetre de reapparition.
func vfRespawnApres(lives []lifeSpan, xuid uint64, finMS int64) int64 {
	best := int64(-1)
	for _, l := range lives {
		if l.xuid != xuid {
			continue
		}
		d := l.from/1000 - finMS
		if d < 0 || d > vfRattacheRespawnMS {
			continue
		}
		if best < 0 || d < best {
			best = d
		}
	}
	return best
}

// vfBatPont construit le pont slot -> joueur pour un gamertag. L'appelant detient
// `LockProcessDecode` : ce balayage est un decodage filmdec de plus.
func vfBatPont(dir, gt string) (vfPont, error) {
	var p vfPont
	xuid, err := vfXUID(dir, gt)
	if err != nil {
		return p, err
	}
	p.xuid = xuid
	scan := filmdec.DefaultScanFilmOptions()
	scan.CaptureDirs = true
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		return p, fmt.Errorf("balayage des positions : %w", err)
	}
	p.positions = len(pos)
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		return p, fmt.Errorf("fil des morts : %w", err)
	}
	lives := buildLifeSpans(indexBySlot(pos))
	off, apparies := bestDeathOffset(lives, deaths)
	if nameLivesByDeaths(lives, deaths, off) == 0 {
		return p, fmt.Errorf("pont slot->xuid vide")
	}
	vfPropage(lives)
	// Les fragments sont releves AVANT le rattachement : apres, ils ne seraient plus anonymes
	// et la publication ne montrerait plus le probleme qu'elle est censee documenter.
	p.anonymes = vfAnonymesDansFenetre(lives)
	p.rattaches = vfRattacheAnonymes(lives, deaths, off)
	vfPropage(lives) // un fragment fraichement nomme peut en nommer d'autres, par meme-slot
	p.offMS, p.apparies = off, apparies
	p.slots = map[uint32]bool{}
	for _, l := range lives {
		if l.xuid == xuid {
			p.slots[l.slot] = true
			p.vies = append(p.vies, l)
		}
	}
	for _, d := range deaths {
		if d.XUID == xuid {
			p.morts = append(p.morts, d.TimeMS+off)
		}
	}
	sort.Slice(p.morts, func(i, j int) bool { return p.morts[i] < p.morts[j] })
	return p, nil
}

// vfAnonymesDansFenetre rend les fragments NON NOMMES qui recouvrent la fenetre d'analyse.
//
// C'EST LE PIEGE CONNU DU DOSSIER, et il doit se voir : `nameLivesByDeaths` ne nomme une vie
// que par la mort qui la TERMINE, donc la vie de DEBUT DE MATCH — celle que le releve Theater
// etiquette — reste anonyme tant qu'aucune mort ne lui est appariee. La phase 6 l'avait deja
// paye une fois (« 0 emission partout = signature d'un pont troue, pas d'un negatif »).
func vfAnonymesDansFenetre(lives []lifeSpan) []lifeSpan {
	var out []lifeSpan
	for _, l := range lives {
		if l.xuid != 0 {
			continue
		}
		if l.to < int64(ondeFeneDebutMS)*1000 || l.from > int64(ondeFeneFinMS)*1000 {
			continue
		}
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].from < out[j].from })
	return out
}

// vfCalib est UNE ligne du tableau du chemin sequentiel : combien de records bipedes la marche
// non ancree rend, pour une largeur d'identifiant bas donnee.
type vfCalib struct {
	idLowBits int
	// bipeds : records d'archetype bipede rendus par la marche ; cibles : ceux dont le slot est
	// celui du joueur ; entiers : ceux traverses sans desynchronisation.
	bipeds, cibles, entiers int
}

// vfSequentiel deroule la marche NON ANCREE sur les premiers chunks, pour chaque largeur
// d'identifiant bas candidate. Il ne sert plus a choisir quoi que ce soit : il PUBLIE le
// rendement de ce chemin, dont l'echec est ce qui justifie le chemin ancre.
func vfSequentiel(dir string, cibles map[uint32]bool, maxChunks int) []vfCalib {
	var table []vfCalib
	brut, err := filmdec.ReadFilmChunk(dir, 0)
	if err != nil {
		return nil
	}
	reg, err := filmdec.ParseRegistryChunk(brut)
	if err != nil {
		return nil
	}
	for _, n := range vfIDLowBitsCandidats {
		cfg := filmdec.DefaultFrameConfig()
		cfg.IDLowBits = n
		table = append(table, vfSequentielUn(dir, reg, cfg, cibles, maxChunks))
	}
	return table
}

// vfSequentielUn compte les records d'UNE largeur candidate.
func vfSequentielUn(dir string, reg *filmdec.Registry, cfg filmdec.FrameConfig,
	cibles map[uint32]bool, maxChunks int,
) vfCalib {
	l := vfCalib{idLowBits: cfg.IDLowBits}
	w := filmdec.NewWorld(reg)
	fin := filmdec.CountFilmChunks(dir)
	if maxChunks > 0 && maxChunks < fin {
		fin = maxChunks
	}
	for c := 1; c <= fin; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			pay := p.Payload(data)
			if p.Type == filmdec.PacketTypeKeyframe {
				w = filmdec.WorldFromKeyframe(reg, pay)
				continue
			}
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			recs, _ := filmdec.DecodeFrameViews(pay, w, cfg, vfVuesParPaquet, cfg.PacketPreambleBits)
			for _, r := range recs {
				if r.TypeIndex != filmdec.BipedTypeIndex {
					continue
				}
				l.bipeds++
				if !cibles[r.Slot] {
					continue
				}
				l.cibles++
				if r.DesyncAt == -1 {
					l.entiers++
				}
			}
		}
	}
	return l
}

// vfVuesParPaquet : meme reglage que la marche de production du killsource et que la phase 0
// de l'attachement — quatre vues de replication deroulees par paquet.
const vfVuesParPaquet = 4
