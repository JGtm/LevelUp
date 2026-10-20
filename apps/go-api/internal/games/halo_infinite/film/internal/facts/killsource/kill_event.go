package killsource

// kill_event.go — LES KILL-EVENTS DE KILLSOURCE : les messages de kill (genre 85) que la marche des
// trames lit dans la vue A de chaque trame, et ceux que son rattrapage retrouve dans les trames ou la
// lecture de la vue A n est pas etablie ([grammar.LireLaMarcheDeKillsource]). killsource ne lit aucun
// bit : il range les messages en kill-events, avec leur instant, leur trame et leur position, et les
// confronte au kill-feed (`assist.go`, `feed.go`).

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar"

// killEventFields : les champs du kill-event (genre 85), struct 0x28 de FUN_14104bd08.
//
// LA QUEUE N EST PAS LUE, ET LA MESURE LE CONFIRME : le champ arme/categorie existe dans
// l executable mais est ABSENT des films (0/133 au vrai curseur, sur les cinq films testes,
// RE_LOG 7ter.25 (4)) ; la lecture tient les reglages qui le gardent a leur defaut
// (`grammar/vue_a_charges_execution.go`). C est precisement pourquoi la source du degat se lit dans
// le dead-state.
//
// LES DEUX BLOCS R32 SONT DES PARTS DE DEGATS EN POURCENTAGE ENTIER — le premier pour le TUEUR,
// le second pour l ASSISTANT. QUATRE JAMBES CONVERGENTES, chacune mesuree :
//
//	(1) ADJACENCE ET ORDRE dans le modele de recap du jeu : `KillerPercentageDamageDone` a
//	    +0x228 est immediatement suivi de `AssistantPercentageDamageDone` a +0x22c — le meme
//	    couple, dans le meme ordre, que les deux blocs de la grammaire.
//	(2) TYPE ENTIER confirme au desassemblage : tag 1 pour ces deux champs, contre 2 pour les
//	    flottants voisins du meme modele.
//	(3) SOMME == 99 sur 22 367 des 31 204 kills ASSISTES de 892 films. La deuxieme valeur la plus
//	    frequente est 89 (1.78 %) et RIEN ne tombe a 100 : la signature du double arrondi vers le
//	    bas de deux parts complementaires.
//	(4) COLLISION avec une capture Cheat Engine du 2026-06-10 sur la constante ARBITRAIRE 149 :
//	    presente sur 3 films de 886, absente de 983 000 lectures a position aleatoire.
//
// RESERVE, ET ELLE SE PORTE PARTOUT OU CES DEUX CHAMPS SONT CITES : le CHEMIN DE DONNEES entre le
// kill-event du film et ces deux champs du modele de recap N EST PAS DEMONTRE — l ecrivain de
// +0x228 est un setter GENERIQUE partage par neuf modeles d UI. Quatre jambes convergentes, PAS
// une chaine d appels prouvee.
type killEventFields struct {
	killer, victim, assist int
	// killerPct : part de degats du TUEUR, pourcentage ENTIER. AUCUN PLAFOND A 100 : des valeurs
	// jusqu a 228 sont mesurees sur des kill-events attaches a de vraies morts nommees (1.7 % des
	// attaches). L hypothese naturelle est le degat EXCEDENTAIRE — une roquette qui retire plus
	// que la vie restante — et elle n est PAS etablie ; ce qui l est, c est qu une valeur > 100
	// est une donnee reelle et non une lecture ratee.
	killerPct uint32
	// assistPct : part de degats de l ASSISTANT. NE VEUT RIEN DIRE quand `assist` vaut -1 : le
	// bloc porte alors une CONSTANTE PAR FILM (149, 70, 20, 197 selon le film). Voir
	// [decodeCtx.fillAssist], qui est le seul endroit ou cette regle se decide.
	assistPct uint32
	flag      int // le R1 entre `killerPct` et `assist` — semantique NON etablie
}

// champsDuKill rend les champs d un message de kill lu par la marche.
func champsDuKill(k grammar.KillLu) killEventFields {
	m := k.Kill
	return killEventFields{killer: int(m.Tueur), victim: int(m.Victime), assist: int(m.Assistant),
		killerPct: m.PartDuTueur, assistPct: m.PartDeLAssistant, flag: int(m.Drapeau)}
}

// killEventsDeLaMarche range les messages de kill de la marche des trames en kill-events : l instant
// et l identite de leur trame, la position de leur genre (`bit`), leurs champs ; puis retire les
// exemplaires repetes ([assistScan.dedoublonner]) et les trie dans l ordre total.
func killEventsDeLaMarche(l grammar.LectureDeKillsource, f *film) *assistScan {
	s := &assistScan{gate15: l.Rattrapage.Gate15, rattrapes: l.Rattrapage.Kills,
		chainesArretees: l.Rattrapage.ChainesArretees}
	s.recs = make([]killEventRec, 0, len(l.Kills))
	for _, k := range l.Kills {
		s.recs = append(s.recs, killEventRec{ms: f.msDe(k.TS), chunk: k.PositionDuChunk, pidx: k.Index,
			bit: int(k.Kill.Debut) + 1, fields: champsDuKill(k)})
	}
	s.dedoublonner()
	trierKillEvents(s.recs)
	return s
}
