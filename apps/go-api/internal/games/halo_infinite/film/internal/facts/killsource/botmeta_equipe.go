package killsource

// botmeta_equipe.go — L EQUIPE D UN BOT, LUE DANS SON ENTREE BOT_METADATA.
//
// # D OU VIENT LA GRAMMAIRE : DE L ECRIVAIN (ADR 0034 D-3, regle 2)
//
// Ghidra, `HaloInfinite.exe` (base 0x140000000), lecture seule, 2026-10-06 :
//
//	ecrivain du paquet  `FUN_14299bda0` : W(32) le nombre d entrees, puis par bot W(32) son index
//	                    absolu, W(32) son slot, W(32) son `bid`, et le CORPS de sa fiche par
//	                    `FUN_1407edea8` ; type de paquet `0xc`, taille `(bits + 7) / 8` octets
//	lecteur miroir      `FUN_1429875e4` (branche `0xc` de `FUN_1428e22c0`), corps `FUN_1407eeba4`
//	le corps            celui des fiches de joueur que `grammar/player_table_record.go` lit pour la
//	                    table de `chunk_00` et le paquet de type 8 (`decodeSlotListes` puis
//	                    `decodeSlotCorps`) : masque R(11) + n, liste R(12) + N octets, liste R(8) +
//	                    M mots, 104 octets, le nom (<= 16 x R(16), arret APRES l unite nulle), puis
//	                    16 octets, R(32), R(64), six champs courts (10, 14, 6, 8, 7, 1), le bloc de
//	                    personnalisation (largeur PAR BUILD, [profile.PersonnalisationOctets]) et
//	                    44 octets bruts (`fiche + 0x1400`)
//	l equipe            `FUN_1424d512c` recopie ces 44 octets en `configuration + 0xCC0` ; leur
//	                    octet 0x25 (`+ 0xCE5`) est pose sur le joueur par `FUN_140ad37f8` ->
//	                    `FUN_140ad389c`, dont le changement notifie le moteur (`FUN_140adedd8`,
//	                    masque `1 << equipe`) ; `FUN_140a20620` l initialise a -1 (aucune)
//	le jumeau           l octet 0x26 (`+ 0xCE6`), -1 par defaut, n est lu que par des charges
//	                    d evenement (`FUN_1430e17bc`, `FUN_1430e1de0`) : il se lit et se COMPARE,
//	                    il ne decide rien
//
// L ORACLE : l equipe que l entite `ti=9` du bot lui donne, liee par ses declarations, sur 56 bots
// de 36 films du parc (55 HI_1_13_0, 1 HI_1_12_0) : 56/56 (`botmeta_equipe_research_test.go`) ; le
// jumeau egal a l equipe sur les 79 bots lus (`.ai/PLAN_REJEU_EQUIPES_SOURCE_2026-10-06.md`, G2.c,
// F.2 et G3.0).
//
// # UNE SECONDE COPIE DE LA GRAMMAIRE DU CORPS, ET POURQUOI ELLE VIT ICI
//
// La grammaire du corps existe une fois dans `grammar` (partagee par `chunk_00` et le type 8) ; elle
// n y expose pas le bloc de 44 octets. L ajouter la ferait monter `grammar.Rev`, donc toutes les
// revisions et tous les faits du parc, pendant que la campagne de grammaire et la representation
// intermediaire travaillent cette couche : la lecture vit dans `killsource`, qui lit deja ce paquet
// (exception datee de `archlint/film_faits_sans_octets_test.go`). Elle rejoint la grammaire quand
// `killsource` devient un canal de la marche unique (lot 2.7.c,
// `.ai/PLAN_REPRESENTATION_INTERMEDIAIRE_ETAPE2_2026-10-03.md`) ; c est la seconde copie, une
// troisieme est interdite (CLAUDE.md, regle 6).
//
// # LA LECTURE EST PROUVEE PAQUET PAR PAQUET
//
// Le paquet se lit EN ENTIER par cette grammaire, et une equipe n en est retenue que si la marche
// FERME : la derniere entree finit a moins d un octet de la fin du paquet (l ecrivain arrondit a
// l octet). Une largeur fausse — un build dont le bloc de personnalisation n est pas celui du profil
// — decale tout ce qui suit : la fermeture tombe, et le paquet ne donne rien. Une entree lue ne
// donne son equipe qu au bot du lecteur historique (`scanBotEntries`) qui a le MEME slot, le MEME
// `bid` et le MEME nom.
//
// LES DEUX LECTEURS PEUVENT DIFFERER, ET C EST COMPTE, PAS ARBITRE : le lecteur historique trouve
// les noms par balayage, et il lit sur `8076f97f` (HI_1_12_0) un `43 KaleDucky` slot 0 `bid` 0 que
// la marche du paquet ne lit pas (un fragment de la seconde copie du nom) ; un tel bot n a pas
// d equipe ([EquipesDesBots.HorsGrammaire]). Une entree que la marche lit sans que le lecteur
// historique la trouve se compte aussi ([EquipesDesBots.EntreesHorsBalayage]).
//
// Un bot sans equipe lue se compte et l orchestrateur le journalise. AUCUN REPLI : ni le siege, ni
// la feuille de match.

import (
	"unicode/utf16"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// Largeurs, en bits, de l entree d un bot, LUES CHEZ L ECRIVAIN (cf. l en-tete) : les valeurs des
// constantes `slot*` de `grammar/player_table.go`, pour la seconde copie de la grammaire du corps.
const (
	botMasquePrefixe    = 11 // rang du bit haut ; le masque compte la valeur + 1 bits
	botMasqueMax        = 2048
	botListeNPrefixe    = 12 // puis N octets
	botListeNMax        = 2048
	botListeMPrefixe    = 8 // puis M mots de 32 bits
	botListeMMax        = 192
	botBloc104          = 832
	botNomUnitesMax     = 16
	botApresLeNom       = 128 + 32 + 64 + 10 + 14 + 6 + 8 + 7 + 1 // jusqu au bloc de personnalisation
	botBloc44           = 352
	botEquipeDansBloc44 = 0x25 * 8 // `configuration + 0xCE5`
	botJumeauDansBloc44 = 0x26 * 8 // `configuration + 0xCE6`
	// botEquipeMax : le domaine d une equipe va de [grammar.TeamNone] (aucune) a 8, celui du
	// designateur d equipe de `ti=9` (`mp_team_designator`, neuf designateurs). Hors domaine :
	// refuse, compte.
	botEquipeMax = 8
)

// EquipesDesBots : le bilan de la lecture de l equipe des bots d un film. Un compte non nul autre
// que [EquipesDesBots.Lues] est a lire : l orchestrateur le journalise (diagnostics.go).
type EquipesDesBots struct {
	// Lues : les bots dont les paquets fermes donnent UNE equipe.
	Lues int
	// Illisibles : les bots sans equipe lue qu un paquet non ferme declare, ou dont une entree porte
	// une equipe hors domaine (ou tous les bots d un film dont le build n a pas de largeur de
	// personnalisation au profil).
	Illisibles int
	// Contradictoires : les bots auxquels deux paquets fermes donnent deux equipes.
	Contradictoires int
	// HorsGrammaire : les bots du lecteur historique qu aucune entree de paquet ferme ne lit, alors
	// que tous les paquets ou il les trouve ferment et qu aucune entree ne les refuse (cf. l en-tete :
	// un fragment de nom).
	HorsGrammaire int
	// EntreesHorsBalayage : les entrees (slot, `bid`) que la marche lit et que le lecteur historique
	// ne trouve pas sous le meme nom — un bot que le roster ne porte pas.
	EntreesHorsBalayage int
	// JumeauxDiscordants : les entrees lues dont le jumeau (`+ 0xCE6`) differe de l equipe.
	JumeauxDiscordants int
	// HorsDomaine : les entrees lues dont l equipe sort de -1..8 ; elles ne donnent rien.
	HorsDomaine int
	// PaquetsNonFermes : les paquets BOT_METADATA dont la marche ne ferme pas.
	PaquetsNonFermes int
	// PersoInconnue : le build du film n a pas de largeur de personnalisation au profil.
	PersoInconnue bool
}

// aLire dit si le bilan porte un bot sans equipe lue.
func (e EquipesDesBots) aLire() bool {
	return e.Illisibles > 0 || e.Contradictoires > 0 || e.HorsGrammaire > 0
}

// equipeDEntree : ce que la grammaire de l ecrivain lit d UNE entree d un paquet BOT_METADATA.
type equipeDEntree struct {
	slot, botID    int
	nom            string
	equipe, jumeau int
}

// lecteurBorne : un lecteur de bits qui refuse de lire au-dela du paquet.
type lecteurBorne struct {
	br *grammar.Lecteur
	ok bool
}

func (l *lecteurBorne) lire(n uint) uint64 {
	if !l.ok || l.br.Remaining() < int(n) {
		l.ok = false
		return 0
	}
	return l.br.ReadBits(n)
}

func (l *lecteurBorne) sauter(n int) {
	if !l.ok || n < 0 || l.br.Remaining() < n {
		l.ok = false
		return
	}
	l.br.Skip(n)
}

// marcherLePaquet lit un payload de type 12 par la grammaire de l ecrivain, `persoBits` etant la
// largeur du bloc de personnalisation du build. Faux quand la marche ne ferme pas (cf. l en-tete).
func marcherLePaquet(pl []byte, persoBits int) ([]equipeDEntree, bool) {
	l := &lecteurBorne{br: grammar.LecteurSur(pl), ok: true}
	n := int(l.lire(32))
	if !l.ok || n > botMaxSlot {
		return nil, false
	}
	out := make([]equipeDEntree, 0, n)
	for range n {
		e, ok := l.entree(persoBits)
		if !ok {
			return nil, false
		}
		out = append(out, e)
	}
	return out, l.br.Remaining() < 8
}

// entree lit UNE entree, de son index absolu a la fin de son bloc de 44 octets.
func (l *lecteurBorne) entree(persoBits int) (equipeDEntree, bool) {
	var e equipeDEntree
	l.sauter(32) // index absolu du bot
	e.slot, e.botID = int(l.lire(32)), int(l.lire(32))
	masque := int(l.lire(botMasquePrefixe)) + 1
	if !l.ok || masque > botMasqueMax {
		return e, false
	}
	l.sauter(masque)
	if nOctets := int(l.lire(botListeNPrefixe)); l.ok && nOctets <= botListeNMax {
		l.sauter(nOctets * 8)
	} else {
		return e, false
	}
	if mMots := int(l.lire(botListeMPrefixe)); l.ok && mMots <= botListeMMax {
		l.sauter(mMots * 32)
	} else {
		return e, false
	}
	l.sauter(botBloc104)
	e.nom = l.nom()
	l.sauter(botApresLeNom + persoBits)
	// LE BLOC DE 44 OCTETS : l equipe et son jumeau y sont des octets SIGNES (-1 = aucune).
	l.sauter(botEquipeDansBloc44)
	e.equipe = int(int8(l.lire(8))) //nolint:gosec // octet signe de l ecrivain, -1 = aucune
	e.jumeau = int(int8(l.lire(8))) //nolint:gosec // idem
	l.sauter(botBloc44 - botJumeauDansBloc44 - 8)
	return e, l.ok
}

// nom lit le nom : des unites de 16 bits, l ecriture s arretant APRES l unite nulle, ou a la 16e.
func (l *lecteurBorne) nom() string {
	u := make([]uint16, 0, botNomUnitesMax)
	for range botNomUnitesMax {
		v := uint16(l.lire(16)) //nolint:gosec // lecture de 16 bits
		if !l.ok || v == 0 {
			break
		}
		u = append(u, v)
	}
	return string(utf16.Decode(u))
}

// releveDesEquipes : les equipes que les paquets FERMES donnent a chaque bot, et ce qui s est refuse.
type releveDesEquipes struct {
	parBot map[[2]int]map[int]bool // (slot, bid) -> equipes lues
	// refuses : les bots que le lecteur historique trouve dans un paquet qui ne ferme pas, ou dont
	// l entree lue porte une equipe hors domaine.
	refuses      map[[2]int]bool
	horsBalayage map[[2]int]bool
	bilan        EquipesDesBots
}

// noter lit UN paquet ; `scan` sont les entrees que le lecteur historique y a trouvees.
func (r *releveDesEquipes) noter(pl []byte, scan []bot, persoBits int) {
	entrees, ferme := marcherLePaquet(pl, persoBits)
	if !ferme {
		r.bilan.PaquetsNonFermes++
		for _, b := range scan {
			r.refuses[[2]int{b.Slot, b.BotID}] = true
		}
		return
	}
	noms := make(map[[2]int]string, len(scan))
	for _, b := range scan {
		noms[[2]int{b.Slot, b.BotID}] = b.Name
	}
	for _, e := range entrees {
		k := [2]int{e.slot, e.botID}
		if nom, ok := noms[k]; !ok || nom != e.nom {
			r.horsBalayage[k] = true
			continue
		}
		if e.equipe < grammar.TeamNone || e.equipe > botEquipeMax {
			r.bilan.HorsDomaine++
			r.refuses[k] = true
			continue
		}
		if e.jumeau != e.equipe {
			r.bilan.JumeauxDiscordants++
		}
		if r.parBot[k] == nil {
			r.parBot[k] = map[int]bool{}
		}
		r.parBot[k][e.equipe] = true
	}
}

// poser donne a chaque bot l equipe de ses paquets fermes quand ils s accordent, et compte les
// autres.
func (r *releveDesEquipes) poser(m *botMeta) {
	for i := range m.Bots {
		b := &m.Bots[i]
		k := [2]int{b.Slot, b.BotID}
		eqs := r.parBot[k]
		switch {
		case len(eqs) == 1:
			for v := range eqs {
				b.equipe, b.equipeLue = v, true
			}
			r.bilan.Lues++
		case len(eqs) > 1:
			r.bilan.Contradictoires++
		case r.refuses[k]:
			r.bilan.Illisibles++
		default:
			r.bilan.HorsGrammaire++
		}
	}
	r.bilan.EntreesHorsBalayage = len(r.horsBalayage)
	m.Equipes = r.bilan
}

// poserLesEquipesDesBots lit l equipe de chaque bot dans les paquets BOT_METADATA du film. La largeur
// du bloc de personnalisation est celle du build ; `persoConnue` faux (build absent du profil, ou
// non lu) : aucune entree ne se lit, chaque bot se compte illisible.
func poserLesEquipesDesBots(f *film, m *botMeta, persoOctets int, persoConnue bool) {
	if !persoConnue {
		m.Equipes = EquipesDesBots{Illisibles: len(m.Bots), PersoInconnue: len(m.Bots) > 0}
		return
	}
	r := &releveDesEquipes{parBot: map[[2]int]map[int]bool{}, refuses: map[[2]int]bool{},
		horsBalayage: map[[2]int]bool{}}
	for _, pi := range paquetsBotMeta(f) {
		r.noter(pi.p.payload, scanBotEntries(pi.p.payload), persoOctets*8)
	}
	r.poser(m)
}

// botsDuFilm agrege les bots de BOT_METADATA et y lit l equipe de chacun : la largeur du bloc de
// personnalisation, qui la place, est celle du build que la table du film a lu. Les bots sans
// equipe lue se disent a l orchestrateur ([decodeCtx.signalerLesEquipesDesBots]).
func (c *decodeCtx) botsDuFilm(build string) botMeta {
	bots := loadBotMeta(c.film)
	perso, persoConnue := profile.PersonnalisationOctets(build)
	poserLesEquipesDesBots(c.film, &bots, perso, persoConnue)
	c.signalerLesEquipesDesBots(bots.Equipes, build)
	return bots
}
