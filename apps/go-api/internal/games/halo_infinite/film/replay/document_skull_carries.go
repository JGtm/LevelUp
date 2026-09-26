package replay

// document_skull_carries.go — LE PORTEUR DU CRANE D'ODDBALL : la forme qu'il prend dans
// l'artefact.
//
// CHRONIQUE — v23 (2026-08-28, lot PORTEUR ODDBALL,
// `.ai/V7.5/replay2d/registre_film/ODDBALL_PORTEUR_PROTOCOLE.md`). Le document publie
// `skullCarries` — LES PERIODES DE PORTAGE DU CRANE, en intervalles de frames, chacun nomme par
// le xuid du porteur. Un champ de document, un schema de plus (`SkullCarry`) et un bloc de
// couverture (`SkullCarriesCoverage`).
//
// SOURCE, TOUTE DANS LE FILM : le porteur est le joueur dont les TICS DE SCORE DE MODE montent
// (`comp 0 A` = `skull_scoring_ticks` en Oddball). Un train de tics d'un meme joueur EST une
// periode de portage ; le porteur est nomme par le pont d'INSTANTS DE MORT PAR MANCHE (le slot
// est reattribue d'une manche a l'autre), donc AUCUNE base. Les PRISES (`comp 21 B` =
// `skull_grabs`) sont publiees en couverture (denominateur), pas comme bornes.
//
// CE QUI A ETE MESURE, ECRIT ICI PARCE QUE C'EST CE QU'UN LECTEUR DOIT TROUVER SUR PLACE. Le
// portage a resiste a CINQ campagnes (D4-D10, proximite/traversee/score personnel : negatifs,
// biais des longs portages). Le canal des TICS de score de mode, lui, tient : gate oracle
// porteur PRINCIPAL correct 7/7 films (recouvrement du temps de portage), gate terrain manche 1
// de d9781168 prises 9/9 et porteurs d'intervalle 8/9 (seuil 8/9). Emplacements identifies par
// l'oracle films confondus, PAS ajustes au film terrain. Detail :
// `ODDBALL_VERITE_TERRAIN_d9781168.md` + `TERRAIN_*.log`.
//
// LE MODE VIENT DE L'APPELANT, comme la couronne VIP et la colline. `comp 0 A` est le score de
// mode de N'IMPORTE quel mode ; seul un film qu'`replaybuild` reconnait Oddball (par
// `game_variant_name`) fournit `SkullInput.Scanned`. Un film non-Oddball ne publie pas de
// `skullCarries`.
//
// LE CRANE LIBRE RESTE PUBLIE (`objectiveObjects`, schema 21) : il dit OU est le crane quand
// PERSONNE ne le porte. Le porteur est la couche VIVANTE par-dessus — le crane est a la position
// de son porteur (le client le pose sur la piste deja publiee), comme la couronne et le drapeau.
//
// POURQUOI LA VERSION MONTE ALORS QUE LE CHAMP EST OPTIONNEL : la reprise du backfill se fait par
// `SchemaVersion`, et un artefact 22 doit se lire « a re-cuire », pas « a jour » — sans quoi aucun
// rejeu Oddball deja cuit ne montrerait jamais le porteur. Meme raison que les montees v14
// (drapeau), v16 (zones), v21 (colline) et v22 (couronne VIP).

// SkullCarry est UNE periode de portage du crane : un joueur, un intervalle de frames.
//
// PAS DE POSITION. Le crane porte est TOUJOURS sur le joueur qui le porte : le client joint par
// `xuid` et le pose sur la piste deja publiee, exactement comme la couronne VIP et le drapeau
// porte. Republier une trajectoire de crane serait republier celle de son porteur.
type SkullCarry struct {
	// XUID est le porteur du crane, en decimal.
	XUID string `json:"xuid"`
	// T0 / T1 bornent l'intervalle en frames (meme axe que Point.T). T1 est INCLUS. Ils datent le
	// PREMIER et le DERNIER tic de score du train de portage — la borne basse suit donc la prise
	// reelle d'environ une seconde (le premier tic tombe apres le ramassage), un biais qui joue
	// CONTRE la duree affirmee, comme la surcote de la couronne VIP.
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// Closed dit qu'un FAIT a mis fin au portage AVANT la fin du rejeu (une chute suivie d'une
	// reprise, ou la fin d'une manche). Faux : le train de tics court jusqu'au bout de l'axe — le
	// film s'arrete pendant le portage, une BORNE HAUTE.
	Closed bool `json:"closed"`
}

// SkullCarriesCoverage porte les denominateurs du calque. Sans eux, « 8 portages » se lirait
// comme une exhaustivite, et un film Oddball sans aucun portage publie serait indistinguable d'un
// film non-Oddball. ELLE EST PUBLIEE MEME QUAND AUCUN PORTAGE NE L'EST, pour la meme raison que
// les couvertures freres ; son ABSENCE dit encore autre chose : l'appelant n'a pas reconnu un
// film Oddball.
type SkullCarriesCoverage struct {
	// SkullFilm dit que l'appelant a reconnu un film Oddball et fourni de quoi lire. Toujours vrai
	// quand ce bloc existe (l'absence du bloc EST le « pas un film Oddball »).
	SkullFilm bool `json:"skullFilm"`
	// Grabs est le nombre de PRISES du crane (`comp 21 B` = `skull_grabs`), toutes manches. Un
	// denominateur independant des trains de tics : il dit combien de fois le crane a change de
	// mains, la ou `Carries` dit combien de portages ont ete publiables.
	Grabs int `json:"grabs"`
	// Trains est le nombre de trains de tics detectes (periodes de portage candidates), avant
	// rejet — le denominateur de `Carries`, `NoBridge` et `OutOfWindow`.
	Trains int `json:"trains"`
	// Carries est le nombre de portages effectivement publies.
	Carries int `json:"carries"`
	// Closed / Open partagent ces portages : ceux qu'un fait a fermes avant la fin de l'axe, et
	// ceux que rien ne ferme (borne haute a la fin de l'axe).
	Closed int `json:"closed"`
	Open   int `json:"open"`
	// NoBridge : trains dont le slot statborg n'a pas ete resolu en xuid pour SA manche. Le pont
	// se tait plutot que de poser le crane sur le mauvais joueur.
	NoBridge int `json:"noBridge"`
	// OutOfWindow : trains dont le debut tombe hors de l'axe de frames publie.
	OutOfWindow int `json:"outOfWindow"`
	// CarrierAbsent : trains dont le porteur ponte n'est PAS present sur la carte (aucune vie
	// bipede ne couvre l'intervalle) — le canal de score a attribue le portage a un joueur absent
	// (mort ou pas encore apparu). Un tel portage n'a AUCUNE position ou poser le crane : on
	// l'ecarte plutot que de faire disparaitre l'icone. Un porteur jamais nomme dans les tracks
	// (presence inconnue) n'entre PAS ici — on ne verifie pas ce qu'on ne connait pas.
	CarrierAbsent int `json:"carrierAbsent"`
}

// Balanced verifie l'invariant : tout train est publie ou rejete sous une cause NOMMEE, et tout
// portage publie est ferme ou ouvert.
func (c SkullCarriesCoverage) Balanced() bool {
	return c.Carries+c.NoBridge+c.OutOfWindow+c.CarrierAbsent == c.Trains &&
		c.Closed+c.Open == c.Carries
}
