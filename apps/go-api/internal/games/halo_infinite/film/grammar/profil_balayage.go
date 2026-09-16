package grammar

// profil_balayage.go — CE QU UN BALAYAGE POSE SUR LES LECTEURS DE BITS QU IL CONSTRUIT
// (lot 2.3 du PLAN_DECODEUR_FILM ; remplace `profil_herite.go`, l heritage PAR L ETAT DU
// PROCESSUS que les lots 2.2.a, 2.2.b et 2.2.e avaient regroupe en attendant ce lot).
//
// # CE QUE CE FICHIER FERME
//
// Jusqu au 2.3, les valeurs ci-dessous vivaient dans UNE variable de paquet (`herite`) que la
// calibration de `killsource` ecrivait et que `LecteurSur` relisait. La consequence etait
// ecrite noir sur blanc dans `profil_herite.go` : `replaybuild.BuildBytes` decode `killsource`
// PUIS appelle `replay.BuildFromFilm` dans le MEME processus, et la cuisson du rejeu decodait
// donc ses composants aux largeurs que la calibration du kill-feed avait retenues — sans que
// personne ne le demande, et sans qu aucun commentaire ne le declare.
//
// LES LARGEURS CALIBREES SUR LE FILM SONT VOULUES (la grammaire mesuree prime sur le defaut) ;
// L HERITAGE PAR L ETAT NE L EST PAS. Le lot 2.3 separe les deux : la calibration RETOURNE son
// resultat ([killsource.Result.ProfilCalibre]), `replaybuild` le PASSE a
// `replay.BuildFromFilm`, et `BuildFromFilm` le pose sur le contexte du film. Le chemin est
// explicite d un bout a l autre, et deux films peuvent se decoder en parallele : plus rien
// n est partage par le processus.
//
// # CE QUE LE PROFIL DE BALAYAGE N EST PAS
//
// Ce n est pas [Profile] — le profil du FILM, resolu une fois a la construction du contexte et
// IMMUABLE. Le profil de balayage est ce qu un balayage POSE : l invariant du profil par
// defaut, puis ce que la carte du match (largeurs d axe objets du monde), la version de format
// (decoupage MPP) ou une calibration (descripteur de traversee, largeur d axe absolue,
// `param_4`) y substituent pour la duree de CE decodage.

// ProfilDeBalayage porte tout ce qu un lecteur de bits doit savoir avant de lire son premier
// bit. Il voyage par VALEUR : un lecteur qui le recoit en tient sa propre copie, et une passe
// ne peut donc pas modifier celle d une autre.
type ProfilDeBalayage struct {
	// Mouvement : descripteur de traversee, largeur d axe absolue, descripteur world-object,
	// range de dequantification, quantum et largeur du delta, drapeaux de contexte.
	Mouvement MovementProfile
	// Cadre : l en-tete par entite et la largeur d un mot de taille des images-cles d etat
	// complet. Invariants du format, relus chez l ecrivain — rien ne les installe par film.
	Cadre KeyframeProfile
	// MPP : le decoupage des deux champs de largeur variable du bloc
	// `object-multiplayer-properties`, pose par la VERSION DE FORMAT du film.
	MPP MPPWidths
	// ParamEtat / ParamEtatImpose : le `param_4` du moteur qu un harnais de balayage a force,
	// et le drapeau qui dit qu il l a force. Hors balayage, la table par composant decide seule.
	ParamEtat       uint32
	ParamEtatImpose bool
	// Grammaire : les bascules A/B de retro-ingenierie, chacune a son defaut de production.
	Grammaire GrammaireBalayage
}

// ProfilDeBalayageParDefaut rend l INVARIANT — la valeur qu un lecteur porte quand aucun
// balayage ne lui a rien pose. Ses trois composantes viennent des MEMES fonctions que
// [ResolveProfile] : il n existe pas de seconde table de valeurs.
func ProfilDeBalayageParDefaut() ProfilDeBalayage {
	return ProfilDeBalayage{Mouvement: mouvementDuProfil(), Cadre: cadreDuProfil(),
		MPP: mppDuProfil(), Grammaire: grammaireDuProfil()}
}

// LargeursObjetDuMonde rend les largeurs d axe du chemin world-object de ce profil.
func (p ProfilDeBalayage) LargeursObjetDuMonde() PrecisionDescriptor { return p.Mouvement.WorldObject }

// PoserLargeursObjetDuMonde installe des largeurs world-object brutes. Les instruments et les
// garde-rails qui sauvent puis restaurent un profil passent par la.
func (p *ProfilDeBalayage) PoserLargeursObjetDuMonde(d PrecisionDescriptor) {
	p.Mouvement.WorldObject = d
}

// PoserLargeursObjetDuMondeDepuisDecoupage installe les largeurs d axe de la CARTE pour le
// chemin world-object. Les axes sont partages avec l absolu du bipede : c est le meme AABB de
// BSP qui les fixe — hypothese verifiee par ses consequences le 2026-08-15 (cf.
// [Lecteur.worldObjectPrecision]).
//
// SOURCE ATTENDUE : `MapQuantEntry.AxisWidths`, deduit des bornes par la loi du moteur. Le
// decoupage lu dans le film (`DetectI0Layout`) sert de controle : s il contredit le catalogue,
// ce sont les BORNES qui sont fausses.
func (p *ProfilDeBalayage) PoserLargeursObjetDuMondeDepuisDecoupage(l I0Layout) {
	if l.AxisW[0] == 0 || l.AxisW[1] == 0 || l.AxisW[2] == 0 {
		return // decoupage non detecte : garder le defaut plutot qu installer des zeros
	}
	p.Mouvement.WorldObject.AxisW = l.AxisW
	// La largeur de l INDEX DE REGION est elle aussi une constante par carte
	// (ceilLog2(nb de regions) — 2 bits sur Live Fire, lot C catalogues 2026-08-27). Un
	// decoupage sans gate (appels historiques qui ne posent que AxisW) laisse le defaut.
	// LIMITE ASSUMEE : le lecteur world-object dequantifie tous les records aux largeurs
	// de LA region cataloguee ; un record d une autre region (rarissime — l ordre des
	// 3/291 288 de Cliffhanger) consommerait des largeurs differentes et desalignerait
	// SON record.
	if l.GateBits > i0SpineBits+i0UseDefaultBits {
		p.Mouvement.WorldObject.IndexW = uint(l.GateBits - i0SpineBits - i0UseDefaultBits)
	}
	// LA REGION ATTENDUE SUIT LES LARGEURS, par le meme chemin et dans le meme appel
	// (lot B-bis, 2026-09-12). Sans elle, le lecteur world-object exigeait un index de region
	// NUL — vrai partout sauf sur Live Fire, dont la region jouee est la 1 sur 2 bits. Il y
	// lisait donc ses trois axes un bit trop tot, et le bit de poids fort de chaque axe
	// devenait le bit de poids faible du champ precedent : un pas de la moitie de l etendue
	// de l axe a chaque bascule (31,89 m sur Y, mesure sur quatre films).
	p.Mouvement.WorldObject.Region = l.Region
}

// PoserParamEtat force le `param_4` du moteur (l actor-tick / weapon-set count que le
// descripteur d un composant rend a FUN_14076cb60) pour les balayages qui prennent ce profil.
// Cf. [paramForComponent] : le repli n est consulte que HORS de la table par composant.
func (p *ProfilDeBalayage) PoserParamEtat(v uint32) { p.ParamEtat, p.ParamEtatImpose = v, true }

// GrammaireBalayage porte les BASCULES DE GRAMMAIRE d un balayage : les choix de lecture qui
// changent la consommation de bits sans venir du flux.
//
// # CE QU ELLES SONT, ET POURQUOI ELLES VIVENT ICI
//
// Ce ne sont ni des largeurs mesurees (le profil du film) ni des observations (l observateur) :
// ce sont des A/B de RETRO-INGENIERIE — « le corps de ce composant est-il porte ? », « la
// portee baseline est-elle levee ? », « ce composant est-il saute a une largeur de bouchon ? ».
// Chacune a son defaut de PRODUCTION, et chacune n a d autre ecrivain qu un instrument de
// mesure ou un harnais de calibration.
//
// ELLES ETAIENT DOUZE VARIABLES DE PAQUET jusqu au lot 2.3, avec leurs douze reglages publics :
// tout instrument qui en posait une la posait pour le PROCESSUS, et c est l une des deux
// raisons pour lesquelles tout decodage passait sous un verrou. Elles voyagent desormais avec
// le lecteur de bits, comme les largeurs.
type GrammaireBalayage struct {
	// ControleDeCorruption : en mode FILM/replay (FUN_1404f2b4c()==2), FUN_14076cb60 lit APRES
	// chaque composant present un R(1) garde ; si le bit vaut 1, un R(32) sentinelle (marqueur
	// 0xbcddcba « entity component corrupt »). Defaut false — le decodeur sautait ces bits, et
	// la mesure live (record NEW bipede a 1924 bits contre 1863 lus) a etabli la grammaire.
	ControleDeCorruption bool
	// BitsDeQueueRecordNew : bits terminaux consommes APRES la boucle de composants d un record
	// NEW (candidat : la queue de FUN_1408f1aa4, non identifiee bit-exact). Defaut 0 =
	// comportement historique ; le chemin delta n appelle pas `TraverseEntity`, donc il n est
	// pas affecte.
	BitsDeQueueRecordNew int
	// DeserEtatParArchetype route les archetypes non-bipede vers leur deserialiseur
	// vtable[0x60] porte (`default_state_arch.go`) au lieu du Skip(0) historique. Defaut TRUE ;
	// le passer a false rejoue la ligne de base d avant le portage.
	DeserEtatParArchetype bool
	// SimStateComplet : si vrai, i60 (simulation-state) est declare ENTIEREMENT decode et la
	// traversee continue vers i61-63. Defaut false — la grammaire de la queue est etablie
	// (`consumeSimStateHandleTail`, lot R7-b), ce qui manque est la SOURCE DES LARGEURS D AXE
	// de cette queue sur le chemin de production. Critere de bascule du defaut : que le chemin
	// absolu d i0 tire ses trois largeurs de la carte du match.
	SimStateComplet bool
	// PorteeBaseline mirroite `DAT_144e61ea0` : une PORTEE, pas un reglage. Les lecteurs d etat
	// complet du groupe `142e2*`/`142e3*` la levent juste AVANT l appel vtable[0x60] et la
	// rabaissent juste apres ; pendant cette portee, tous les lecteurs de position passent du
	// quantifie au BRUT 96 bits. Defaut false depuis le 2026-08-17 ; critere de bascule :
	// l atterrissage bit-exact des 591 records `ti=35` bornes au-dessus de 50 %.
	PorteeBaseline bool
	// GrammaireEcrivainI0 route le chemin ABSOLU d i0 sur la grammaire que l ECRIVAIN d etat
	// complet du jeu pose (lot R7-d) : le 3e bit ne supprime pas la charge utile, il choisit la
	// table de plage et ouvre la queue de handle ; le champ de 2 bits vient EN DERNIER. Defaut
	// false depuis le 2026-08-17 (R7-e) : la correction n est pas prouvee bit-exacte sur
	// l oracle de frontiere, et ce chemin sert la trajectoire de PRODUCTION du rejeu 2D.
	// Retrait de la bascule vise a la cloture du chantier image-cle, au plus tard le 2026-10-31.
	GrammaireEcrivainI0 bool
	// CorpsActionMobilite : le corps de FUN_1408f02c8 (i54) est-il decode ? Defaut TRUE.
	CorpsActionMobilite bool
	// CorpsAncrageCapacite : le corps tag==3 d i59 est-il decode ? Defaut TRUE.
	CorpsAncrageCapacite bool
	// InferenceChaine route l inference de slot non lie par le resolveur RECURSIF
	// (`frame_chain_infer.go`), qui voit a travers des suites de transitoires que la
	// confirmation a un pas ne franchit pas. Defaut false (comportement historique).
	InferenceChaine bool
	// LargeursCalibrees remplace le deserialiseur d un composant par un SAUT de largeur fixe,
	// pour un harnais de calibration keye sur un film et une carte. Vide par defaut. Le
	// composant dead-state ne doit PAS y entrer (il est decode pour lire l index de participant
	// du tueur absolu).
	LargeursCalibrees map[string]int
	// GenerationStricte exige que l eid COMPLET d un delta (tag de generation inclus)
	// corresponde a celui pose au binding, comme le fait `FUN_1406caad8` (`entry[0] != eid` ->
	// return 3, corps NON lu, boucle abandonnee). Defaut false : les liaisons issues des
	// images-cles portent une generation qui peut avoir change depuis.
	//
	// `killsource` LE LEVE POUR TOUT SON DECODAGE ([killsource.ProfilDeDepart]) — et il le
	// levait jusqu au lot 2.3 pour tout le PROCESSUS, donc aussi pour la cuisson du rejeu qui
	// suivait. Ce fait est desormais porte par le profil, comme les largeurs calibrees.
	GenerationStricte bool
	// LargeursBouchon donne une largeur PROVISOIRE a un composant dont le deserialiseur n est
	// pas encore porte, pour que la traversee continue au-dela (recherche de la largeur d une
	// queue manquante par chainage de records). Vide par defaut : un composant non porte
	// desynchronise. N est PAS un chemin de decodage de production.
	LargeursBouchon map[string]int
}

// grammaireDuProfil rend l INVARIANT des bascules — les valeurs de PRODUCTION.
func grammaireDuProfil() GrammaireBalayage {
	return GrammaireBalayage{
		DeserEtatParArchetype: true,
		CorpsActionMobilite:   true,
		CorpsAncrageCapacite:  true,
	}
}

// largeurCalibree rend la largeur de saut calibree d un composant, si un harnais en a pose une.
func (g GrammaireBalayage) largeurCalibree(nom string) (int, bool) {
	w, ok := g.LargeursCalibrees[nom]
	return w, ok
}

// largeurBouchon rend la largeur provisoire d un composant non porte, si un harnais en a pose une.
func (g GrammaireBalayage) largeurBouchon(nom string) (int, bool) {
	w, ok := g.LargeursBouchon[nom]
	return w, ok
}

// ContexteDeLecture : CE QU UN LECTEUR DE BITS PORTE, en un seul objet (lot 2.3).
//
// DEUX CHAMPS, ET LEUR NATURE EST OPPOSEE — c est pour cela qu ils restent deux :
//
//	Profil  DECIDE des largeurs. Le fausser change les bits lus.
//	Obs     ne fait que RECEVOIR ce que le deserialiseur a deja lu. Il ne change AUCUNE
//	        consommation de bits ; un champ qui en changerait une serait une valeur de profil
//	        mal rangee (cf. l en-tete de `observateur.go`).
//
// ILS VOYAGENT ENSEMBLE PARCE QU ILS VIENNENT DU MEME BALAYAGE, et les faire voyager separement
// laissait un appelant en oublier un : c est exactement ce qui est arrive aux largeurs de carte
// avant le lot 2.3, quand un instrument oubliait de les installer.
type ContexteDeLecture struct {
	Profil ProfilDeBalayage
	Obs    *Observation
}

// ContexteParDefaut rend l INVARIANT : le profil par defaut, et personne qui observe.
func ContexteParDefaut() ContexteDeLecture {
	return ContexteDeLecture{Profil: ProfilDeBalayageParDefaut()}
}
