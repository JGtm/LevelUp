package grammar

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

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
// Ce n est pas [profile.Profile] — le profil du FILM, resolu une fois a la construction du contexte et
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
	Mouvement profile.MovementProfile
	// Cadre : l en-tete par entite et la largeur d un mot de taille des images-cles d etat
	// complet. Invariants du format, relus chez l ecrivain — rien ne les installe par film.
	Cadre profile.KeyframeProfile
	// MPP : le decoupage des deux champs de largeur variable du bloc
	// `object-multiplayer-properties`, pose par la VERSION DE FORMAT du film.
	MPP profile.MPPWidths
	// Grammaire : les bascules A/B de retro-ingenierie, chacune a son defaut de production.
	Grammaire GrammaireBalayage
}

// ProfilDeBalayageParDefaut rend l INVARIANT — la valeur qu un lecteur porte quand aucun
// balayage ne lui a rien pose. Ses trois composantes viennent des MEMES fonctions que
// [ResolveProfile] : il n existe pas de seconde table de valeurs.
func ProfilDeBalayageParDefaut() ProfilDeBalayage {
	return ProfilDeBalayage{Mouvement: profile.MouvementParDefaut(), Cadre: profile.CadreParDefaut(),
		MPP: profile.MPPParDefaut(), Grammaire: grammaireDuProfil()}
}

// LargeursObjetDuMonde rend les largeurs d axe du chemin world-object de ce profil.
func (p ProfilDeBalayage) LargeursObjetDuMonde() profile.PrecisionDescriptor {
	return p.Mouvement.WorldObject
}

// PoserLargeursObjetDuMonde installe des largeurs world-object brutes. Les instruments et les
// garde-rails qui sauvent puis restaurent un profil passent par la.
func (p *ProfilDeBalayage) PoserLargeursObjetDuMonde(d profile.PrecisionDescriptor) {
	p.Mouvement.WorldObject = d
}

// PoserLargeursObjetDuMondeDepuisDecoupage installe les largeurs d axe de la CARTE pour le
// chemin world-object. Les axes sont partages avec l absolu du bipede : c est le meme AABB de
// BSP qui les fixe — hypothese verifiee par ses consequences le 2026-08-15 (cf.
// [Lecteur.worldObjectPrecision]).
//
// SOURCE ATTENDUE : `profile.MapQuantEntry.AxisWidths`, deduit des bornes par la loi du moteur. Le
// decoupage lu dans le film (`DetectI0Layout`) sert de controle : s il contredit le catalogue,
// ce sont les BORNES qui sont fausses.
func (p *ProfilDeBalayage) PoserLargeursObjetDuMondeDepuisDecoupage(l profile.I0Layout) {
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
	if l.GateBits > profile.I0SpineBits+profile.I0UseDefaultBits {
		p.Mouvement.WorldObject.IndexW = uint(l.GateBits - profile.I0SpineBits - profile.I0UseDefaultBits)
	}
	// LA REGION ATTENDUE SUIT LES LARGEURS, par le meme chemin et dans le meme appel
	// (lot B-bis, 2026-09-12). Sans elle, le lecteur world-object exigeait un index de region
	// NUL — vrai partout sauf sur Live Fire, dont la region jouee est la 1 sur 2 bits. Il y
	// lisait donc ses trois axes un bit trop tot, et le bit de poids fort de chaque axe
	// devenait le bit de poids faible du champ precedent : un pas de la moitie de l etendue
	// de l axe a chaque bascule (31,89 m sur Y, mesure sur quatre films).
	p.Mouvement.WorldObject.Region = l.Region
	// LA CARTE EST LA, DONC `i60` EST DECLARE COMPLET : c est le critere de bascule que
	// `SimStateComplet` porte depuis le lot R7-b, et il est TENU par ce geste meme (cf.
	// [grammaireSousCarte]).
	p.Grammaire = grammaireSousCarte(p.Grammaire, true)
}

// grammaireSousCarte rend la grammaire d un profil selon que les LARGEURS D AXE DE LA CARTE du
// match sont disponibles pour le chemin absolu d i0. C EST LA REGLE, ECRITE UNE FOIS.
//
// # POURQUOI UNE BASCULE DE GRAMMAIRE DEPEND D UNE CARTE
//
// [GrammaireBalayage.SimStateComplet] declare `i60 simulation-state` entierement decode, queue
// comprise. La grammaire de cette queue est etablie depuis le lot R7-b (2026-08-17) ; ce qui lui
// manquait sur le chemin de production etait la SOURCE DES LARGEURS D AXE, et son critere de
// bascule etait ecrit noir sur blanc : « que le chemin absolu d i0 tire ses trois largeurs de la
// CARTE du match ». Un profil qui porte la carte TIENT ce critere ; un profil qui ne la porte pas
// ne le tient pas, et `i60` y desynchronise PAR CONCEPTION — pas par lacune de grammaire.
//
// # CE QUI NE BASCULE PAS, ET C EST LE POINT
//
// Le DEFAUT GLOBAL ([grammaireDuProfil]) reste faux. `NewFilmContext` — auto-detecte, SANS
// carte — sert les enveloppes `ScanFilm*(dir)` et les instruments, ou les largeurs d axe ne
// viennent pas de la carte : un defaut global leve y ferait lire `i60` au-dela de ce que le
// critere autorise. La bascule est donc une LIGNE DE PROFIL, posee par les deux portes par
// lesquelles une carte entre dans un profil de balayage — ce geste-ci
// ([ProfilDeBalayage.PoserLargeursObjetDuMondeDepuisDecoupage], la porte de `killsource` et de
// `replay.installWorldObjectPrecision`) et la construction du contexte de film sous catalogue
// (`NewFilmContextForMap`).
//
// MESURE DE SA LEVEE (carte `snowbound`, film `bfecd02b`, lot 5.3.3-a) : fautif `i60` 38 -> 0 ·
// records `ti=35` 11 150 -> 11 228 · `i29` lu 7 -> 14 · `i62` lu 6 -> 14 · trames saines
// 40,5 % -> 40,6 %.
func grammaireSousCarte(g GrammaireBalayage, carte bool) GrammaireBalayage {
	g.SimStateComplet = carte
	return g
}

// grammaireSousFilm rend la grammaire d un profil selon CE QUE LE FILM DECLARE. C EST LA REGLE,
// ECRITE UNE FOIS, et son second rendu dit si le film a parle.
//
// # UNE SEULE BASCULE VIENT DU FILM, ET ELLE EST LUE CHEZ L ECRIVAIN
//
// [GrammaireBalayage.ControleDeCorruption] n est pas un reglage : c est le bit de
// `chunk_00 + 0x0CB45C`, que `FUN_14299b198` @14299b25b ecrit et que `FUN_14299ab50` @14299ac28
// relit, que `FUN_1428e219c` @1428e2239 recopie dans le singleton du film (`+0x1AE`) et que
// `FUN_14076cea8` rend a `FUN_14076cb60` en rejeu. Le film le porte, donc le decodeur le LIT —
// ADR 0034 : le film est autoportant, jamais un profil par build (cf.
// [profile.FilmIdentity.ControleDeCorruption]).
//
// # POURQUOI ELLE NE SE POSE PLUS A LA MAIN SUR UN CONTEXTE DE FILM
//
// Avant le lot 5.18.2 ce champ etait de defaut FAUX et n avait d ecrivain qu un instrument. Le
// defaut se trouvait JUSTE — le bit vaut zero sur les 1 605 films du cache, 8 builds, 5 formats
// — mais il l etait par hasard, et un film qui le leverait aurait desynchronise sans un mot.
// [FilmContext.ProfilDeBalayage] le DERIVE desormais a chaque rendu : rien d autre qu un film
// n a le droit de le poser, et un profil pose par-dessus ne peut donc pas l effacer.
//
// # LE REPLI, NOMME ET COMPTE
//
// `lue` faux = le film ne porte PAS de section d identification (5 films du cache, format 20,
// deja mis de cote par [profile.ErrUnknownBuild]). La grammaire garde alors son invariant : c est
// le repli `repli_controle_corruption_section_absente` du registre, et il est COMPTE par
// [FilmContext.ControleDeCorruptionRepli] et par la calibration de `killsource`.
func grammaireSousFilm(g GrammaireBalayage, p profile.Profile) (GrammaireBalayage, bool) {
	if !p.IdentityRead() {
		return g, false
	}
	g.ControleDeCorruption = p.Identity().ControleDeCorruption
	return g, true
}

// GrammaireSousFilm applique [grammaireSousFilm] a un PROFIL DE BALAYAGE complet, pour les
// chemins de decodage qui n ouvrent pas de [FilmContext] — `killsource`, qui part de l invariant
// et calibre. Second rendu : faux = repli `repli_controle_corruption_section_absente`.
func GrammaireSousFilm(bal ProfilDeBalayage, f *source.Film) (ProfilDeBalayage, bool) {
	g, lue := grammaireSousFilm(bal.Grammaire, ResolveProfile(f, nil))
	bal.Grammaire = g
	return bal, lue
}

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
	// 0x0bcddcba « entity component corrupt »). Idem FUN_142e2c690 sur le chemin d etat complet.
	//
	// IL VIENT DU FILM DEPUIS LE LOT 5.18.2, ET DE NULLE PART AILLEURS : c est le bit de
	// `chunk_00 + 0x0CB45C` (cf. [grammaireSousFilm]). Le defaut de structure reste faux — c est
	// la valeur du singleton du jeu a la construction (`FUN_140eff23c`) et a chaque chargement
	// (`FUN_140eff3a8`) — mais il n est plus ce que la production LIT : elle derive le champ du
	// film a chaque rendu de profil. Les instruments qui le posent mesurent l A/B ; ils ne
	// decident plus de la production.
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
	// traversee continue vers i61-63. La grammaire de la queue est etablie
	// (`consumeSimStateHandleTail`, lot R7-b) ; ce qui manquait etait la SOURCE DES LARGEURS
	// D AXE de cette queue, et le critere ecrit etait « que le chemin absolu d i0 tire ses trois
	// largeurs de la CARTE du match ».
	//
	// IL NE SE POSE PLUS A LA MAIN DEPUIS LE LOT 5.3.3-a (2026-09-21) : il SUIT la carte, par
	// [grammaireSousCarte] — vrai sur un profil qui porte les largeurs de la carte, faux sinon.
	// Le defaut global reste faux, et c est le critere lui-meme qui l exige (cf. la doc de
	// [grammaireSousCarte]).
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
	// TablesParVue applique la GARDE DE TABLE DE VUE dans la boucle d inference : un delta dont
	// le slot appartient a une AUTRE vue n est pas un record, et la vue s arrete sur son en-tete
	// (`1 + idLow + 2` bits) sans lire un bit de corps — exactement `FUN_1406cd128` quand
	// `vue[0x38][slot].eid != eid` pose `uVar14 = 2` et sort de la boucle.
	//
	// C EST LE PIED DE TRAME, et sans elle la marche lit au-dela de la fin du paquet : mesure du
	// lot 5.11.6 sur `dad793c7`, 57 bits de trop sur 95,7 % des paquets, et un record FANTOME
	// (`ti=6` slot 26) fabrique a partir de zeros. Defaut : cf. le journal du lot 5.11.7.
	TablesParVue bool
	// ClassesDeVue applique A CHAQUE RANG DE VUE LA GRAMMAIRE DE SA CLASSE, la ou la marche
	// appliquait aux trois rangs celle du gestionnaire d entites (lot 5.14).
	//
	// Les trois vtables de vue ne portent pas la meme fonction a `+0x40`, et le registraire
	// `FUN_141f855b4` dit quel rang porte quelle classe : rang 0 = `FUN_14076a1c4` (vue A,
	// un flux de MESSAGES qui ne rend jamais un record), rang 1 = `FUN_1406cd128` (vue B, le
	// gestionnaire d entites), rang 2 = `FUN_1406cf548` (vue C, `replication_control_view.cpp`).
	// Voir `frame_vue_classes.go`.
	//
	// Sans elle, les flux des vues A et C sont decoupes en `[prefixe][idLow][tag]` et rendent
	// des records DEL FANTOMES (13 sur `dad793c7`, 304 sur `bfecd02b`) — et ces faux DEL
	// DELIENT des entites vivantes : les rebrancher a leur classe rend 15 114 records `ti=35`
	// sur `bfecd02b` (114 458 -> 129 572), a desyncs constants.
	//
	// DEFAUT LEVE DEPUIS LE LOT 5.14.3 (2026-09-22) : c est la grammaire de l ecrivain, et la
	// fermeture des paquets le prouve au bit (reste NUL sur 5 341 paquets sur 5 341 de
	// `dad793c7`). Ce n est PAS un kill-switch : il n y a pas de date de retrait, la bascule
	// existe pour que l A/B du lot reste rejouable.
	ClassesDeVue bool
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
		TablesParVue:          true,
		ClassesDeVue:          true,
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
