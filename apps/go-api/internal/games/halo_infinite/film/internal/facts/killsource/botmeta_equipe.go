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
//	l equipe            sur HI_1_13_0, `FUN_1424d512c` recopie ces 44 octets en
//	                    `configuration + 0xCC0` ; leur octet 0x25 (`+ 0xCE5`) est pose sur le
//	                    joueur par `FUN_140ad37f8` -> `FUN_140ad389c`, dont le changement notifie le
//	                    moteur (`FUN_140adedd8`, masque `1 << equipe`) ; `FUN_140a20620`
//	                    l initialise a -1 (aucune)
//	le jumeau           l octet 0x26 (`+ 0xCE6`), -1 par defaut, n est lu que par des charges
//	                    d evenement (`FUN_1430e17bc`, `FUN_1430e1de0`) : il se lit et se COMPARE,
//	                    il ne decide rien
//
// LA POSITION DES DEUX OCTETS DANS LE BLOC CHANGE AVEC LE BUILD : 0x25 et 0x26 sur HI_1_13_0, 4 et 6
// sur HI_1_10_0 a HI_1_12_0, ou l octet 0x25 tombe dans la copie du nom et se lit 0. Elle est une
// donnee du profil ([profile.PositionEquipeBotPour]), RELUE sur l executable de HI_1_13_0, MESUREE
// ailleurs ; un build sans position etablie ne donne aucune equipe ([EquipesDesBots.PositionInconnue]).
//
// L ORACLE : l equipe de la feuille de match et celle que l entite `ti=9` du bot lui donne, liee par
// ses declarations (`botmeta_equipe_research_test.go`, `.ai/V7.5/PLAN_REJEU_EQUIPES_SOURCE_2026-10-06.md`).
// Elles CONTROLENT la mesure ; aucune ne remplace la lecture.
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
// les noms par balayage, et il lit dans la seconde copie d un nom un fragment (`43 KaleDucky`,
// slot 0, `bid` 0) que la marche du paquet ne lit pas. La grammaire l ecarte d un paquet qui ferme
// ([grammar.PaquetBotMetadata.FragmentsEcartes], compte en [EquipesDesBots.FragmentsEcartes]) ; un
// nom balaye qu aucune entree de paquet ferme ne lit et qui n est pas un tel fragment n est pas un
// bot que l ecrivain ecrit : il SORT DU ROSTER et se compte ([EquipesDesBots.HorsGrammaire]) — jamais
// une entree sans equipe. Une entree que la marche lit sans que le lecteur historique la trouve se
// compte aussi ([EquipesDesBots.EntreesHorsBalayage]).
//
// Un VRAI bot sans equipe lue se compte et l orchestrateur le journalise en erreur. AUCUN REPLI : ni
// le siege, ni la feuille de match.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
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
	// personnalisation, ou pas de position de l equipe, au profil).
	Illisibles int
	// Contradictoires : les bots auxquels deux paquets fermes donnent deux equipes.
	Contradictoires int
	// HorsGrammaire : les bots du lecteur historique qu aucune entree de paquet ferme ne lit, alors
	// que tous les paquets ou il les trouve ferment et qu aucune entree ne les refuse — sans etre un
	// fragment ecarte. La grammaire de l ecrivain ne l ecrit pas : il est RETIRE DU ROSTER, et compte
	// ici (avertissement `killsource.entrees_de_bots`) ; il n est jamais publie sans equipe.
	HorsGrammaire int
	// FragmentsEcartes : les noms que la grammaire ecarte du balayage des paquets fermes, fragments de
	// la seconde copie d un nom ecrit (cf. l en-tete), sommes sur les paquets du film.
	FragmentsEcartes int
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
	// PositionInconnue : le build du film n a pas de position de l equipe au profil
	// ([profile.PositionEquipeBotPour]) ; les entrees se lisent, sans leur equipe.
	PositionInconnue bool
}

// aLire dit si le bilan porte un VRAI bot sans equipe lue.
func (e EquipesDesBots) aLire() bool {
	return e.Illisibles > 0 || e.Contradictoires > 0
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
		if !e.EquipeLue { // aucune position de l equipe pour ce build : refusee, jamais 0
			r.refuses[k] = true
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

// poser donne a chaque bot l equipe de ses paquets fermes quand ils s accordent, compte les autres,
// et retire du roster les noms hors grammaire (cf. l en-tete).
func (r *releveDesEquipes) poser(m *botMeta) {
	gardes := m.Bots[:0]
	for _, b := range m.Bots {
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
			continue
		}
		gardes = append(gardes, b)
	}
	m.Bots = gardes
	r.bilan.EntreesHorsBalayage = len(r.horsBalayage)
	m.Equipes = r.bilan
}

// poserLesEquipesDesBots donne a chaque bot l equipe que les paquets BOT_METADATA du film lui donnent,
// lus sous la `lecture` du build. Build sans largeur de personnalisation au profil : aucune entree
// ne se lit ; sans position de l equipe : les entrees se lisent sans equipe. Dans les deux cas,
// chaque bot se compte illisible — jamais une equipe par defaut.
func poserLesEquipesDesBots(paquets []grammar.PaquetBotMetadata, m *botMeta, lecture grammar.LectureDesBots) {
	fragments := 0
	for _, p := range paquets {
		fragments += p.FragmentsEcartes
	}
	if !lecture.PersoConnue || !lecture.EquipeConnue {
		m.Equipes = EquipesDesBots{Illisibles: len(m.Bots), FragmentsEcartes: fragments,
			PersoInconnue:    !lecture.PersoConnue && len(m.Bots) > 0,
			PositionInconnue: lecture.PersoConnue && len(m.Bots) > 0}
		return
	}
	r := &releveDesEquipes{parBot: map[[2]int]map[int]bool{}, refuses: map[[2]int]bool{},
		horsBalayage: map[[2]int]bool{}}
	for _, p := range paquets {
		r.noter(p)
	}
	r.bilan.FragmentsEcartes = fragments
	r.poser(m)
}

// botsDuFilm agrege les bots de BOT_METADATA et y lit l equipe de chacun, sous la lecture que le
// profil donne au build que la table du film a lu (largeur du bloc de personnalisation, position de
// l equipe). Les bots sans equipe lue se disent a l orchestrateur
// ([decodeCtx.signalerLesEquipesDesBots]).
func (c *decodeCtx) botsDuFilm(build string) botMeta {
	lecture := grammar.LectureDesBotsPour(build)
	paquets := grammar.PaquetsBotMetadata(c.film.src, lecture)
	bots := loadBotMeta(paquets)
	poserLesEquipesDesBots(paquets, &bots, lecture)
	c.signalerLesEquipesDesBots(bots.Equipes, build)
	return bots
}
