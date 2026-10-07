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
// jumeau egal a l equipe sur les 78 bots dont l equipe se lit (79 bots du lecteur historique sur 48
// films, dont un fantome, cf. [EquipesDesBots.HorsGrammaire] ;
// `.ai/PLAN_REJEU_EQUIPES_SOURCE_2026-10-06.md`, G2.c, F.2 et G3.0).
//
// # LA LECTURE VIT DANS LA GRAMMAIRE
//
// Depuis le lot 2.7.c1 de la representation intermediaire, la grammaire lit le paquet par le corps
// commun des fiches de joueur ([grammar.PaquetsBotMetadata], `grammar/bot_metadata.go`), qui expose
// le bloc de 44 octets ; ce fichier n en garde que le releve des equipes par bot.
//
// # LA LECTURE EST PROUVEE PAQUET PAR PAQUET
//
// Le paquet se lit EN ENTIER par cette grammaire, et une equipe n en est retenue que si la marche
// FERME : la derniere entree finit a moins d un octet de la fin du paquet (l ecrivain arrondit a
// l octet). Une largeur fausse — un build dont le bloc de personnalisation n est pas celui du profil
// — decale tout ce qui suit : la fermeture tombe, et le paquet ne donne rien. Une entree lue ne
// donne son equipe qu au bot du lecteur historique (le balayage des noms) qui a le MEME slot, le MEME
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
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// botEquipeMax : le domaine d une equipe va de [grammar.TeamNone] (aucune) a 8, celui du designateur
// d equipe de `ti=9` (`mp_team_designator`, neuf designateurs). Hors domaine : refuse, compte.
const botEquipeMax = 8

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

// releveDesEquipes : les equipes que les paquets FERMES donnent a chaque bot, et ce qui s est refuse.
type releveDesEquipes struct {
	parBot map[[2]int]map[int]bool // (slot, bid) -> equipes lues
	// refuses : les bots que le lecteur historique trouve dans un paquet qui ne ferme pas, ou dont
	// l entree lue porte une equipe hors domaine.
	refuses      map[[2]int]bool
	horsBalayage map[[2]int]bool
	bilan        EquipesDesBots
}

// noter releve UN paquet : les entrees que la grammaire de l ecrivain y lit, confrontees a celles que
// le lecteur historique (le balayage des noms) y a trouvees.
func (r *releveDesEquipes) noter(p grammar.PaquetBotMetadata) {
	if !p.Ferme {
		r.bilan.PaquetsNonFermes++
		for _, b := range p.Balayees {
			r.refuses[[2]int{b.Slot, b.BotID}] = true
		}
		return
	}
	noms := make(map[[2]int]string, len(p.Balayees))
	for _, b := range p.Balayees {
		noms[[2]int{b.Slot, b.BotID}] = b.Nom
	}
	for _, e := range p.Ecrites {
		k := [2]int{e.Slot, e.BotID}
		if nom, ok := noms[k]; !ok || nom != e.Nom {
			r.horsBalayage[k] = true
			continue
		}
		if e.Equipe < grammar.TeamNone || e.Equipe > botEquipeMax {
			r.bilan.HorsDomaine++
			r.refuses[k] = true
			continue
		}
		if e.Jumeau != e.Equipe {
			r.bilan.JumeauxDiscordants++
		}
		if r.parBot[k] == nil {
			r.parBot[k] = map[int]bool{}
		}
		r.parBot[k][e.Equipe] = true
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

// poserLesEquipesDesBots donne a chaque bot l equipe que les paquets BOT_METADATA du film lui donnent,
// lus sous la largeur de personnalisation du build ; `persoConnue` faux (build absent du profil, ou
// non lu) : aucune entree ne se lit, chaque bot se compte illisible.
func poserLesEquipesDesBots(paquets []grammar.PaquetBotMetadata, m *botMeta, persoConnue bool) {
	if !persoConnue {
		m.Equipes = EquipesDesBots{Illisibles: len(m.Bots), PersoInconnue: len(m.Bots) > 0}
		return
	}
	r := &releveDesEquipes{parBot: map[[2]int]map[int]bool{}, refuses: map[[2]int]bool{},
		horsBalayage: map[[2]int]bool{}}
	for _, p := range paquets {
		r.noter(p)
	}
	r.poser(m)
}

// botsDuFilm agrege les bots de BOT_METADATA et y lit l equipe de chacun : la largeur du bloc de
// personnalisation, qui la place, est celle du build que la table du film a lu. Les bots sans
// equipe lue se disent a l orchestrateur ([decodeCtx.signalerLesEquipesDesBots]).
func (c *decodeCtx) botsDuFilm(build string) botMeta {
	perso, persoConnue := profile.PersonnalisationOctets(build)
	paquets := grammar.PaquetsBotMetadata(c.film.src, perso*8, persoConnue)
	bots := loadBotMeta(paquets)
	poserLesEquipesDesBots(paquets, &bots, persoConnue)
	c.signalerLesEquipesDesBots(bots.Equipes, build)
	return bots
}
