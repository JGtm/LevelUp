package replay

// document_vip_crown.go — LA COURONNE VIP : la forme qu'elle prend dans l'artefact.
//
// CHRONIQUE — v22 (2026-08-27, lot VIP COURONNE, `.ai/V7.5/replay2d/registre_film/
// VIP_COURONNE_PROTOCOLE.md`). Le document publie `vipCrown` — LES PERIODES DE PORT DE LA
// COURONNE, en intervalles de frames, chacun nomme par le xuid du VIP. Un champ de document, un
// schema de plus (`VipPeriod`) et un bloc de couverture (`VipCrownCoverage`).
//
// SOURCE, TOUTE DANS LE FILM : les bornes viennent des SELECTIONS `vip_selected` (`comp 22 A` du
// statborg, resolu `TimesSelectedAsVip` au gate corrige — 100 % par joueur x3 films, temoin
// decale 0) et du fil des morts ; le VIP est nomme par le pont d'INSTANTS DE MORT (aucune ligne
// de match, donc aucune base). La couronne est a la position de son porteur — le client la
// dessine sur la piste PUBLIEE du joueur, rien de plus n'est decode.
//
// LE MODE VIENT DE L'APPELANT, ET C'EST DELIBERE. `comp 22 A` vaut `flag_grabs` en CTF : lu sur
// un film de CTF, il rendrait de fausses couronnes. La garde de mode est chez `replaybuild`, qui
// connait `game_variant_name` — comme la colline de KOTH. Un film non-VIP ne fournit pas de
// `VipInput.Scanned`, et `vipCrown` reste absent.
//
// POURQUOI LA VERSION MONTE ALORS QUE LE CHAMP EST OPTIONNEL : la reprise du backfill se fait par
// `SchemaVersion`, et un artefact 21 doit se lire « a re-cuire », pas « a jour » — sans quoi
// aucun rejeu VIP deja cuit ne montrerait jamais la couronne. Meme raison que les montees v14
// (drapeau), v16 (zones) et v21 (proprietaire de colline).

// VipPeriod est UNE periode de port de la couronne : un joueur, un intervalle de frames.
//
// PAS DE POSITION. La couronne est TOUJOURS sur le joueur qui la porte : le client joint par
// `xuid` et la pose sur la piste deja publiee, exactement comme le drapeau porte. Republier une
// trajectoire de couronne serait republier celle de son porteur.
type VipPeriod struct {
	// XUID est le porteur de la couronne, en decimal.
	XUID string `json:"xuid"`
	// T0 / T1 bornent l'intervalle en frames (meme axe que Point.T). T1 est INCLUS.
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// Closed dit qu'un FAIT DATE a mis fin au port (mort du VIP, ou selection suivante). Faux :
	// rien ne l'a ferme — l'intervalle court jusqu'a la fin de l'axe, une BORNE HAUTE.
	Closed bool `json:"closed"`
}

// VipCrownCoverage porte les denominateurs du calque. Sans eux, « 15 periodes » se lirait comme
// une exhaustivite, et un film VIP sans aucune periode publiee serait indistinguable d'un film
// non-VIP. ELLE EST PUBLIEE MEME QUAND AUCUNE PERIODE NE L'EST, pour la meme raison que les
// couvertures freres ; son ABSENCE dit encore autre chose : l'appelant n'a pas reconnu un film VIP.
type VipCrownCoverage struct {
	// VipFilm dit que l'appelant a reconnu un film VIP et fourni de quoi lire. Toujours vrai
	// quand ce bloc existe (l'absence du bloc EST le « pas un film VIP »).
	VipFilm bool `json:"vipFilm"`
	// Selections est le nombre de selections VIP de l'oracle (`vip_selected`) : le denominateur
	// de tout ce qui suit.
	Selections int `json:"selections"`
	// Periods est le nombre de periodes effectivement publiees.
	Periods int `json:"periods"`
	// Closed / Open partagent ces periodes : celles qu'un fait date a fermees, et celles que
	// rien ne ferme (borne haute a la fin de l'axe).
	Closed int `json:"closed"`
	Open   int `json:"open"`
	// ClosedByDeath / ClosedBySelection : la cause de fermeture des periodes fermees. La mort du
	// VIP est la cause dominante (il perd la couronne en tombant) ; la selection suivante ferme
	// une periode que le fil des morts aurait manquee.
	ClosedByDeath     int `json:"closedByDeath"`
	ClosedBySelection int `json:"closedBySelection"`
	// NoBridge : selections dont le slot statborg n'a pas ete resolu en xuid. Le pont se tait
	// plutot que de poser la couronne sur le mauvais joueur.
	NoBridge int `json:"noBridge"`
	// OutOfWindow : selections tombant hors de l'axe de frames publie (fins de partie que le
	// film prolonge au-dela de la derniere position rendue).
	OutOfWindow int `json:"outOfWindow"`
}

// tally impute une periode publiee a ses compteurs de population et de cause.
func (c *VipCrownCoverage) tally(r vipRawPeriod) {
	if r.closed {
		c.Closed++
		switch r.cause {
		case "mort":
			c.ClosedByDeath++
		case "selection":
			c.ClosedBySelection++
		}
		return
	}
	c.Open++
}

// Balanced verifie les deux invariants : toute selection est publiee ou rejetee sous une cause
// NOMMEE, et toute periode publiee est fermee ou ouverte.
func (c VipCrownCoverage) Balanced() bool {
	return c.Periods+c.NoBridge+c.OutOfWindow == c.Selections && c.Closed+c.Open == c.Periods
}
