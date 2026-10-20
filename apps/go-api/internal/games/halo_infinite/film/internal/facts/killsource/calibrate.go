package killsource

// calibrate.go — L INFERENCE DES LARGEURS, DEVENUE ORACLE, ET LA CALIBRATION DE `param_4`.
//
// # CE QUI A CHANGE AU LOT 3.4.1-b, ET POURQUOI
//
// Deux parametres du decodeur de replication sont installes AU CHARGEMENT DE LA CARTE et ne se
// lisent nulle part dans le film : les largeurs d axe de position (`axisW`) et la largeur
// d index de plage (`indexW`). Ce fichier les INFERAIT, par balayage, faute de les connaitre.
//
// ON LES CONNAIT. Elles sont dans le catalogue des cartes, calculees des bornes par la loi du
// moteur (`profile/loi_largeurs.go`, accord 79 cartes sur 79) et posees au profil par
// [profile.MapQuantEntry.PrecisionAbsolue]. L inference NE DECIDE DONC PLUS : la valeur LUE
// prime sur la valeur mesuree (arbitrage utilisateur V17, M3-Q8 — « 3.4 lira la valeur dans le
// film plutot que de la mesurer »). Le balayage RESTE, comme ORACLE : il est le seul juge
// INTERNE AU FILM dont on dispose pour dire qu une entree de catalogue ment, et un balayage
// qu on retire est une mesure qu on cesse de faire.
//
// CE QUE LE BALAYAGE PROUVAIT, ET CE QU IL PROUVE ENCORE : les quatre films de reference
// rendaient QUATRE couples DISTINCTS (14/1, 17/2, 16/2, 17/1) — c est-a-dire exactement ce
// qu une grandeur PAR CARTE produit. La lecture Ghidra du remplisseur (`FUN_140be9a14`, note
// 3.4 du 2026-09-16) explique enfin POURQUOI, et d ou la valeur vient.
//
// CRITERE, ET IL EST INTERNE : nombre de records d archetype biped decodes sans desynchronisation
// sur un echantillon FIGE de paquets sans event. Il ne regarde ni le kill-feed, ni une arme, ni
// aucun oracle externe. GARDE-FOU : la configuration designee doit dominer la MEDIANE d un
// facteur >= 2 ; un profil plat signifie que le parametre reel n est pas dans l espace balaye,
// et alors l oracle ne designe rien (champ `Flat`) — il ne contredit personne.
//
// LE MONDE EST CELUI DES PRELIMINAIRES DE LA MARCHE DES TRAMES, SOUS LE PROFIL DU CONTEXTE (son
// decoupage MPP compris) : chunk par chunk, la table anticipee puis la liaison des images-cles du
// chunk, chaque trame de l echantillon marchee sous le monde de son chunk
// ([grammar.FilmContext.ScoresDeCalibration]). Il est chronologique, et c est un correctif paye
// cher : un monde avance a la FIN du film pour des paquets du DEBUT tue 387 paquets sur 400 en BTB,
// ou les slots de biped sont RECYCLES avec une nouvelle generation.

import (
	"cmp"
	"fmt"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// calibration : ce que le decodeur APPLIQUE, et ce que le balayage MESURE a cote.
type calibration struct {
	// LueAxisW / LueIndexW : LES VALEURS QUE LE DECODAGE APPLIQUE, lues au profil — les trois
	// largeurs d axe de la table PAR INDEX de la carte et la largeur d index de plage
	// ([profile.MapQuantEntry.PrecisionAbsolue]). Elles PRIMENT (V17, M3-Q8).
	LueAxisW  [3]uint
	LueIndexW uint
	// AxisW : la largeur d axe UNIFORME que le BALAYAGE designe. Depuis le lot 3.4.1 elle ne
	// DECIDE plus : elle sert d ORACLE, confrontee a [LueAxisW].
	AxisW uint
	// PoigneeIndexW : LA LARGEUR DU MOT DE POIGNEE, ET LE BALAYAGE LA DECIDE (lot 3.4.1, apres
	// la mesure d equivalence du 2026-09-17).
	//
	// ELLE N EST PAS LA LARGEUR D INDEX DE PLAGE, et les confondre a coute une lecture publiee.
	// `Traversal.IndexW` est la largeur du mot que `consumePositionHandleTail` lit dans la
	// queue de poignee (`FUN_1406d3140`, bitlen du compte de poignees) ; la largeur d INDEX DE
	// PLAGE (`DAT_144632be0`) est celle de la carte, et elle vit dans `WorldObject.IndexW`.
	// Aucune source lue ne donne la premiere : ni le catalogue, ni l executable relu a ce jour.
	// Le balayage reste donc son SEUL pourvoyeur — repli `repli_largeur_mot_de_poignee_inferee`
	// au registre — exactement comme `param_4`.
	//
	// CE QUE LA MESURE A MONTRE : en demotant l inference en oracle, le lot 3.4.1 avait retire
	// cette valeur sans rien mettre a sa place ; elle retombait a l invariant 1. Sur
	// `a521164d`, `ScanAbilityImpulses` rend 1 impulsion a la largeur 2 et ZERO a 1 comme a 3 —
	// une lecture publiee perdue, vue par `replay-equiv` (`abilityImpulses` 1 -> 0, artefact
	// -14 octets). Les largeurs d AXE, elles, restent celles de la carte : deux grandeurs, deux
	// noms, deux sources (D5 (3.4.1), fermee par ce commit).
	PoigneeIndexW uint
	// PoigneeScore / PoigneeMedian / PoigneeDiscriminee : CE QUE LA MESURE DU MOT DE POIGNEE A
	// VU, et s il y avait quelque chose a voir.
	//
	// Le balayage score les trois largeurs candidates AU TRIPLET LU de la carte (le monde que la
	// production decode) et n ecrit `PoigneeIndexW` au profil que si la meilleure DOMINE la
	// mediane d un facteur `flatRatio`. `PoigneeDiscriminee` faux = l invariant tient, sous le
	// repli `repli_largeur_mot_de_poignee_inferee`.
	//
	// POURQUOI CES TROIS CHAMPS EXISTENT. Avant le 2026-09-17, rien ne disait si la valeur posee
	// venait d une mesure ou d un ex aequo. Elle venait d un ex aequo sur les deux films
	// instruits (272/272/272 et 61/61/61), et elle voyageait jusqu au rejeu. Un lecteur du rendu
	// lisible doit pouvoir lire la difference.
	PoigneeScore, PoigneeMedian int
	PoigneeDiscriminee          bool
	// PoigneeDecidee : le balayage du mot de poignee a TOURNE — retenue ou invariant faute de
	// discrimination, la largeur ne vient d aucune lecture (`repli_largeur_mot_de_poignee_inferee`,
	// compte par decodage, lot J8.7).
	PoigneeDecidee bool
	// CarteLue : les largeurs viennent-elles de l entree de catalogue de la CARTE du match ?
	// FAUX n arrive qu en RECHERCHE : en production [Decode] refuse un film sans carte avant la
	// calibration ([ErrCarteAbsente], lot J7 ; le repli qui decodait aux largeurs d une autre
	// carte est retire depuis le 2026-09-27). En recherche, FAUX = les largeurs d UNE carte
	// (`cliffhanger`) appliquees a celle-ci ; le rendu lisible le dit et
	// [decodeCtx.avertirReplisDeCalibration] l avertit par film.
	CarteLue bool
	// ControleDeCorruptionLu : le film a-t-il DECLARE son controle de corruption par composant
	// (le bit de `chunk_00 + 0x0CB45C`, lot 5.18.2) ? FAUX = le film ne porte pas de section
	// d identification, et la grammaire garde son invariant : repli
	// `repli_controle_corruption_section_absente`. Le rendu lisible le dit, comme pour la carte.
	ControleDeCorruptionLu bool
	// Desaccords : nombre de grandeurs ou l inference CONTREDIT la valeur lue. UNE SEULE
	// grandeur y entre — la largeur d axe, hors du voisinage du triplet lu : c est la seule que
	// l oracle et la carte disent toutes les deux. La largeur du mot de poignee en est SORTIE
	// avec D5 (3.4.1) : comparer une grandeur decidee a une grandeur lue qui n est pas la meme
	// rendait un desaccord qui ne voulait rien dire.
	Desaccords    int
	Score, Median int
	Flat          bool // le profil est plat : l inference n a rien designe de net
	// Profil est le PROFIL DE BALAYAGE retenu, celui que toutes les passes qui suivent posent
	// sur leur lecteur de bits (lot 2.2.a pour le mouvement, elargi au `param_4` au lot 2.3).
	//
	// AVANT CES LOTS, LE RESULTAT DE LA CALIBRATION N ETAIT NULLE PART : il etait ECRIT dans
	// des variables de paquet de `grammar` (`SetAbsoluteAxisW`, `TraversalPrecision`,
	// `SetRecordStateParam`) que la passe suivante relisait sans le savoir — `runWalk`
	// reconstruisait un `DefaultFrameConfig` et heritait pourtant des largeurs calibrees, par
	// effet de bord du processus, et LA CUISSON DU REJEU aussi (decouverte D1 du lot 2.2.a).
	// Il est desormais RENDU, et passe explicitement par `FrameConfig.Profil` puis par
	// [Result.ProfilCalibre].
	Profil grammar.ProfilDeBalayage
}

func (c calibration) String() string {
	src := fmt.Sprintf("score %d, mediane %d", c.Score, c.Median)
	if c.Flat {
		src = fmt.Sprintf("PROFIL PLAT (score %d, mediane %d)", c.Score, c.Median)
	}
	source := "CARTE"
	if !c.CarteLue {
		source = "DEFAUT (carte absente)"
	}
	corr := fmt.Sprintf("controle_corruption=%v [FILM]", c.Profil.Grammaire.ControleDeCorruption)
	if !c.ControleDeCorruptionLu {
		corr = "controle_corruption=INVARIANT (section d identification absente)"
	}
	// LE RENDU DIT D OU VIENT LA LARGEUR DU MOT DE POIGNEE. « decidee » = la mesure a discrimine
	// au triplet lu ; « INVARIANT (non discriminee) » = elle ne l a pas fait, et le repli tient.
	// Sans cette mention, une valeur devinee sur un ex aequo se lisait comme une valeur mesuree.
	poignee := fmt.Sprintf("decidee [score %d, mediane %d]", c.PoigneeScore, c.PoigneeMedian)
	if !c.PoigneeDiscriminee {
		poignee = fmt.Sprintf("INVARIANT (non discriminee) [score %d, mediane %d]",
			c.PoigneeScore, c.PoigneeMedian)
	}
	return fmt.Sprintf("LU axisW=%v indexW_plage=%d [%s] %s | ORACLE axisW=%d [%s] desaccords=%d "+
		"| DECIDE indexW_poignee=%d %s",
		c.LueAxisW, c.LueIndexW, source, corr, c.AxisW, src, c.Desaccords,
		c.PoigneeIndexW, poignee)
}

// bornes des DEUX espaces balayes, separes depuis le 2026-09-17 : 21 largeurs d axe pour
// l oracle, PUIS 3 largeurs de mot de poignee pour la decision — 24 configurations,
// ET NON PLUS les 63 d un produit cartesien qui melangeait deux grandeurs.
const (
	axisWMin, axisWMax   = uint(6), uint(26)
	indexWMin, indexWMax = uint(1), uint(3)
	calibSampleSize      = 400
	// calibSampleOctets : la taille minimale d une trame de l echantillon, en octets.
	calibSampleOctets = 400
	flatRatio         = 2.0
)

// calibrate : pose le profil du film sur le contexte de la marche, MESURE l inference a cote, et
// decide la largeur du mot de poignee. Le critere se compte sous le monde des preliminaires de la
// marche des trames ([grammar.FilmContext.ScoresDeCalibration]).
//
// L INFERENCE DES LARGEURS NE DECIDE PLUS (lot 3.4.1-b, V17 M3-Q8 : « la valeur LUE prime sur
// la valeur mesuree »). Le profil porte les largeurs de la table PAR INDEX de la carte et la
// largeur d index de plage ; le balayage reste, il rend un VERDICT qu on confronte — c est le
// seul oracle INTERNE AU FILM dont on dispose pour dire qu une entree de catalogue ment.
func calibrate(fc *grammar.FilmContext, views int, carte *profile.MapQuantEntry) (calibration, error) {
	profil, carteLue := ProfilDeDepartPourCarte(carte)
	// LE CONTROLE DE CORRUPTION PAR COMPOSANT VIENT DU FILM (lot 5.18.2) : le profil part de
	// l invariant, et la MEME regle s applique ici, par la seule porte qui l exporte.
	profil, corrLu := grammar.GrammaireSousFilm(profil, fc.Film())
	abs := profil.LargeursObjetDuMonde()
	res := calibration{Profil: profil, CarteLue: carteLue, ControleDeCorruptionLu: corrLu,
		LueAxisW: abs.AxisW, LueIndexW: abs.IndexW}
	poserLeProfil(fc, profil)
	err := infererLargeurs(fc, views, &res)
	return res, err
}

// infererLargeurs : DEUX MESURES SEPAREES, DEUX SORTS, ET UN SEUL BALAYAGE QUI DECIDE ENCORE.
//
//	LARGEUR D AXE     la CARTE decide (catalogue, loi verifiee 79 fois sur 79). Le balayage est
//	                  un ORACLE : il compte ses desaccords avec elle, il n ecrit rien.
//	MOT DE POIGNEE    aucune source lue ne la donne. Le balayage la DECIDE — mais SEULEMENT s il
//	                  la DISCRIMINE, et il la score dans le monde que la production decode.
//
// POURQUOI LES DEUX MESURES SONT SEPAREES DEPUIS LE 2026-09-17 (voie (a1) du pilote).
// Le balayage unique balayait les deux grandeurs ENSEMBLE et retenait le COUPLE de meilleur
// score, sous une largeur d axe UNIFORME `aw/aw/aw`. Or la production ne lit plus jamais une
// largeur uniforme depuis le lot 3.4.1 : elle lit le TRIPLET de la carte. Le `iw` retenu etait
// donc l argmax dans un monde que le decodeur n habite plus, et le garde-fou ne le voyait pas :
// `flatRatio` testait la nettete de la largeur d AXE, puis le code prenait le `iw` du MEME
// gagnant sans jamais verifier qu il fut discrimine. UNE SEULE MESURE, DEUX GRANDEURS, UN SEUL
// GARDE — et la consequence voyageait jusqu au rejeu, `profilDeBalayageDeLaCuisson`
// (`replaybuild/kills.go`) transmettant `Result.ProfilCalibre` a tous les lecteurs derriere i0.
//
// CE QUE LA MESURE A DIT (§5 du plan, 2026-09-17, deux films en lecture seule) :
//
//	a521164d  au TRIPLET LU [17 17 15]   iw=1 272 · iw=2 272 · iw=3 272   AVEUGLE
//	          sous l UNIFORME            iw=1 226 · iw=2 226 · iw=3 230   4 records sur 226
//	64e8adfa  au TRIPLET LU [15 15 15]   61 · 61 · 61                     EGALITE PARFAITE
//
// Le critere est donc AVEUGLE a la largeur du mot de poignee sur ces films : la valeur publiee
// roulait sur un ex aequo tranche par un tri instable. Elle ne roule plus.
func infererLargeurs(fc *grammar.FilmContext, views int, res *calibration) error {
	cadres := cadresDeCalibration(fc.ProfilDeBalayage())
	scores, _, err := fc.ScoresDeCalibration(cadres,
		grammar.EchantillonDeCalibration{Taille: calibSampleSize, OctetsMin: calibSampleOctets}, views)
	if err != nil {
		return err
	}
	nAxe := int(axisWMax - axisWMin + 1)
	oracleLargeurAxe(scores[:nAxe], res)
	decideMotDePoignee(scores[nAxe:], res)
	return nil
}

// cadresDeCalibration rend les cadres que le critere score sous `profil` — le profil du contexte,
// son decoupage MPP compris —, dans cet ordre : les 21 largeurs d axe UNIFORMES a mot de poignee
// FIGE sur l invariant (l oracle), puis les trois largeurs du mot de poignee au TRIPLET LU de la
// carte (la decision).
func cadresDeCalibration(profil grammar.ProfilDeBalayage) []grammar.FrameConfig {
	lues, invariant := profil.Mouvement.WorldObject, profil.Mouvement.Traversal
	out := make([]grammar.FrameConfig, 0, axisWMax-axisWMin+1+indexWMax-indexWMin+1)
	for aw := axisWMin; aw <= axisWMax; aw++ {
		cfg := grammar.DefaultFrameConfig()
		cfg.Profil = profil
		cfg.Profil.Mouvement.WorldObject = lues
		cfg.Profil.Mouvement.WorldObject.AxisW = [3]uint{aw, aw, aw}
		cfg.Profil.Mouvement.Traversal = invariant
		out = append(out, cfg)
	}
	for iw := indexWMin; iw <= indexWMax; iw++ {
		cfg := grammar.DefaultFrameConfig()
		cfg.Profil = profil
		cfg.Profil.Mouvement.WorldObject = lues
		cfg.Profil.Mouvement.Traversal = profile.PrecisionDescriptor{IndexW: iw, AxisW: invariant.AxisW}
		out = append(out, cfg)
	}
	return out
}

// oracleLargeurAxe : L ORACLE, ET IL N ECRIT RIEN AU PROFIL.
//
// Il balaie les 21 largeurs d axe UNIFORMES a mot de poignee FIGE sur l invariant — figer la
// seconde grandeur est ce qui rend la premiere lisible. Il retient la meilleure, la confronte au
// triplet LU de la carte, et compte le desaccord.
//
// LE GARDE-FOU EST CELUI D ORIGINE : une configuration qui ne domine pas la MEDIANE d un facteur
// `flatRatio` signifie que le parametre reel n est pas dans l espace balaye. L oracle ne designe
// alors rien (`Flat`) et ne contredit personne — un oracle qui ne voit rien se tait.
//
// LE DESACCORD SE COMPTE A LA PORTEE DE L ORACLE. Il balaie un UNIFORME quand la verite est un
// TRIPLET (17/17/15 sur Fragmentation) : exiger l egalite ferait un desaccord sur presque toute
// carte, et le compteur ne dirait plus rien. Le critere est donc « la valeur lue est-elle dans
// le voisinage [min, max] du triplet ? ». Un oracle a 16 contre `[17 17 15]` ne contredit pas la
// carte ; un oracle a 16 contre `[13 13 14]` — l invariant applique a une carte qui n est pas la
// sienne — si. Limite ecrite au §4, D5 (3.4.1).
func oracleLargeurAxe(scores []int, res *calibration) {
	type cand struct {
		aw    uint
		score int
	}
	out := make([]cand, 0, len(scores))
	for k, s := range scores {
		out = append(out, cand{axisWMin + uint(k), s}) //nolint:gosec // rang d un des 21 candidats
	}
	// TRI DETERMINISTE : score decroissant, PUIS largeur croissante. `sort.Slice` n est pas
	// stable, et sur des ex aequo son `out[0]` est arbitraire — c est exactement le defaut que
	// ce commit retire ; il ne sera pas laisse ici.
	slices.SortFunc(out, func(a, b cand) int { // largeur : unique par candidat
		return cmp.Or(cmp.Compare(b.score, a.score), cmp.Compare(a.aw, b.aw))
	})
	med := out[len(out)/2].score
	res.AxisW, res.Score, res.Median = out[0].aw, out[0].score, med
	if float64(out[0].score) < flatRatio*float64(max1(med)) {
		res.Flat = true
		return
	}
	if res.AxisW < minLargeur(res.LueAxisW) || res.AxisW > maxLargeur(res.LueAxisW) {
		res.Desaccords++
	}
}

// decideMotDePoignee : LA SEULE VALEUR QUE CE BALAYAGE DECIDE ENCORE — quand il la voit.
//
// `scores` sont ceux des trois largeurs candidates AU TRIPLET LU de la carte, c est-a-dire dans le
// monde que la production decode, et non sous un uniforme qu elle a cesse de lire. La valeur n est
// ecrite au profil que si elle est DISCRIMINEE ; sinon l invariant reste, sous le repli
// `repli_largeur_mot_de_poignee_inferee` (registre, famille `killsource/calibration`).
func decideMotDePoignee(scores []int, res *calibration) {
	invariant := res.Profil.Mouvement.Traversal
	retenu, score, med, discriminee := motDePoigneeRetenu(scores, invariant.IndexW)
	res.PoigneeIndexW, res.PoigneeScore, res.PoigneeMedian = retenu, score, med
	res.PoigneeDiscriminee = discriminee
	res.Profil.Mouvement.Traversal.IndexW = retenu
	res.PoigneeDecidee = true
}

// motDePoigneeRetenu : LA DECISION, ISOLEE POUR ETRE TESTABLE SANS FILM.
//
// `scores[k]` est le nombre de records de bipede lus sans desynchronisation a la largeur
// `indexWMin + k`, mesure au triplet LU. Rend la largeur retenue, son score, la mediane des
// candidats, et SI la mesure a discrimine.
//
// LE SEUIL EST CELUI DE L ORACLE D AXE, `flatRatio`, ET C EST VOULU : meme critere interne, meme
// doctrine — « une configuration qui ne domine pas la MEDIANE d un facteur 2 signifie que le
// parametre reel n est pas dans l espace balaye ». Il est severe, et il doit l etre : la mesure
// du 2026-09-17 a montre que l ecart qui decidait autrefois valait QUATRE RECORDS SUR 226
// (1,8 %) — du bruit promu en donnee. Un seuil qui laisserait passer 1,8 % ne garderait rien.
//
// TROIS CANDIDATS, DONC LA MEDIANE EST LE DEUXIEME : une egalite parfaite (272/272/272) rend
// `out[0] == med` et le rapport vaut 1, donc jamais 2 — l invariant tient, et c est le cas
// mesure sur `a521164d` comme sur `64e8adfa`.
//
// DETERMINISME : a scores egaux la PLUS PETITE largeur gagne, et de toute facon l egalite ne
// passe pas le seuil. Rejoue deux fois, la fonction rend la meme valeur — c est ce que le test
// cible verifie sur les deux films.
func motDePoigneeRetenu(scores []int, invariant uint) (retenu uint, score, mediane int, discriminee bool) {
	if len(scores) == 0 {
		return invariant, 0, 0, false
	}
	type cand struct {
		iw    uint
		score int
	}
	out := make([]cand, 0, len(scores))
	for k, s := range scores {
		out = append(out, cand{indexWMin + uint(k), s})
	}
	slices.SortFunc(out, func(a, b cand) int { // largeur : unique par candidat
		return cmp.Or(cmp.Compare(b.score, a.score), cmp.Compare(a.iw, b.iw))
	})
	med := out[len(out)/2].score
	if float64(out[0].score) < flatRatio*float64(max1(med)) {
		// NON DISCRIMINEE : la mesure ne separe pas les candidats. Le balayage se tait et
		// l invariant tient — jamais une valeur devinee sur un ex aequo.
		return invariant, out[0].score, med, false
	}
	return out[0].iw, out[0].score, med, true
}

// minLargeur / maxLargeur : les bornes du triplet de largeurs lu.
func minLargeur(w [3]uint) uint {
	m := w[0]
	for _, v := range w[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxLargeur(w [3]uint) uint {
	m := w[0]
	for _, v := range w[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// max1 : denominateur jamais nul.
func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
