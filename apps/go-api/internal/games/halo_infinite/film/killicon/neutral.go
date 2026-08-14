package killicon

// neutral.go — LE PICTOGRAMME DU **TYPE** D UNE MORT QUE PERSONNE NE REVENDIQUE.
//
// # POURQUOI CE N EST PAS [Lookup], ET POURQUOI CA NE DOIT PAS L ETRE
//
// [Lookup] repond a << QUELLE ARME >>. Sur une ligne de kill, c est la bonne question : il y a
// un tueur, et l icone de son arme est ce que le jeu affiche entre les deux noms.
//
// Sur une mort SANS TUEUR, la bonne question est << QUEL TYPE DE MORT >>. Poser la l icone de
// l arme donnerait a lire un kill par roquette la ou personne n a tue : le joueur s est tue tout
// seul. C est la distinction que le jeu fait lui-meme — son kill feed y affiche un pictogramme
// de suicide, pas une arme — et c est la raison pour laquelle cette table est SEPAREE de
// `data/rules.tsv` au lieu d y ajouter des lignes.
//
// # LA TABLE, ET CE QUI LA JUSTIFIE LIGNE PAR LIGNE
//
// Elle a DEUX entrees, pas sept. Les cinq autres pictogrammes de mort de l atlas (ecrasement,
// bidon a fusion, chute d eau, ricochet, joueur parti) existent bien dans les assets extraits,
// mais RIEN aujourd hui ne permet de les atteindre depuis une donnee mesuree : la nature du
// degat ne descend pas a ce grain. Les cabler au jugé poserait l icone d une autre mort — la
// meme faute que l icone d une autre arme, deja refusee ici.
//
//	DEGAT_GLOBAL  ->  environment  la classe EST la reponse : `labels.tsv` la qualifie
//	                               << chute, environnement >>, et c est le pictogramme du jeu
//	                               pour ces morts-la. 4 des 5 morts mesurees le 2026-08-14.
//	toute autre    ->  suicide     la mort n a aucun tueur et le dead-state designe la victime :
//	  classe                       le joueur s est tue avec sa propre source de degat (temoin
//	  ETABLIE                      mesure : un M41 SPNKr tire trop pres). Le jeu appelle ca un
//	                               suicide, et il a un pictogramme pour ca.
//	INCONNU        ->  RIEN        un `jpt!` valide dont aucune source ne remonte ne dit pas de
//	  ou hors table                quoi le joueur est mort : le fil garde son repere neutre.
//
// # LA POPULATION A LAQUELLE CETTE TABLE S APPLIQUE
//
// UNIQUEMENT `killsource.Result.UnclaimedDeaths` — les morts que le kill-feed porte sans aucun
// kill en face ET dont le dead-state designe la victime elle-meme. Appliquer cette table a une
// mort ordinaire rendrait << suicide >> sur un kill parfaitement attribue.

import "levelup/go-api/internal/games/halo_infinite/film/damagetag"

// Les TYPES de mort publies. Identifiants stables, jamais des libelles : la traduction FR/EN
// appartient a la couche d affichage (regle i18n du depot).
const (
	// NeutralKindEnvironment : chute, hors-limites, degat de monde.
	NeutralKindEnvironment = "environment"
	// NeutralKindSuicide : le joueur s est tue avec sa propre source de degat.
	NeutralKindSuicide = "suicide"
)

// neutralSprites : le stem de l atlas KILL FEED par type. Les deux vignettes sont celles que
// l extraction a nommees `environment` (killfeed-55) et `suicide` (killfeed-61) dans
// `static/weapons-assets/halo_infinite/jeu/index.json` — le garde-rail de ce paquet verifie que
// les PNG existent, comme pour toute regle.
var neutralSprites = map[string]string{
	NeutralKindEnvironment: "killfeed-55",
	NeutralKindSuicide:     "killfeed-61",
}

// NeutralDeath : le type d une mort sans tueur, et son pictogramme.
//
// Le second retour est FAUX quand la nature n est pas etablie — l appelant garde alors son
// repere neutre. Il ne rend JAMAIS un type par defaut : un type par defaut serait une
// affirmation sur une mort dont on ne sait rien.
func NeutralDeath(tag uint32) (kind string, icon Icon, ok bool) {
	lab, known := damagetag.Lookup(tag)
	if !known || lab.Class == damagetag.ClassInconnu {
		return "", Icon{}, false
	}
	kind = NeutralKindSuicide
	if lab.Class == damagetag.ClassGlobal {
		kind = NeutralKindEnvironment
	}
	return kind, Icon{Sprite: neutralSprites[kind], Genre: GenreClasse}, true
}

// NeutralSprites : la table type -> vignette, copiee. Sert aux garde-rails d assets.
func NeutralSprites() map[string]string {
	out := make(map[string]string, len(neutralSprites))
	for k, v := range neutralSprites {
		out[k] = v
	}
	return out
}
