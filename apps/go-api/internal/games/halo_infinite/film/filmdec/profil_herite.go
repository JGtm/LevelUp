package filmdec

// profil_herite.go — CE QU UNE PASSE DE DECODAGE LAISSE A LA SUIVANTE, EN UN SEUL ENDROIT NOMME
// (lots 2.2.a, 2.2.b et 2.2.e du PLAN_DECODEUR_FILM ; le fichier s appelait `mouvement_herite.go`
// jusqu au 2.2.e, quand il a cesse de ne porter que le mouvement).
//
// # CE QUE CE FICHIER PORTE, ET POURQUOI IL EXISTE
//
// Le lot 2.2.a fait lire au chemin de position SON PROFIL, porte par le lecteur de bits
// ([BitReader.poserMouvement]) au lieu de cinq variables de paquet. Quatre de ces cinq valeurs
// n avaient AUCUN ecrivain : leur retrait est sec. La cinquieme et la sixieme — le descripteur
// de traversee et la largeur d axe absolue — en ont un, la CALIBRATION de `killsource`, et son
// resultat ne se contente pas de servir `killsource` :
//
//	`replaybuild.BuildBytes` decode `killsource` PUIS appelle `replay.BuildFromFilm`, dans le
//	MEME processus. `killsource.Decode` prend et rend `LockProcessDecode`, mais il ne RESTAURE
//	PAS les largeurs qu il a calibrees. Toute la cuisson du rejeu qui suit decode donc ses
//	composants aux largeurs que la calibration du kill-feed a retenues sur CE film — 14/1 par
//	defaut, mais 17/2, 16/2, 17/1 sur les quatre films de reference de `calibrate.go`.
//
// CET HERITAGE EST UN FAIT DE PRODUCTION, PAS UNE INTENTION ECRITE : aucun commentaire du depot
// ne le declare, et `replay.BuildFromFilm` ne demande rien. Le supprimer changerait la largeur
// de chaque i0 de la cuisson du rejeu — exactement ce que le critere D4 du jalon interdit
// (« un pas structurel n est clos qu a zero difference »). Il est donc CONSERVE, mais il cesse
// d etre invisible : il tient en UNE variable, elle porte son nom, et elle porte sa date de
// retrait.
//
// # KILL-SWITCH (regle 11 du CLAUDE.md)
//
//	BASCULE         2026-09-17 (lot 2.2.a) : cinq variables de paquet deviennent une ; le
//	                2.2.b y ajoute les largeurs de carte, le 2.2.e les largeurs MPP et le
//	                `param_4` force — soit ONZE variables regroupees en une.
//	CIBLE DE RETRAIT lot 2.3 (« plus de globale, plus de verrou »), et au plus tard le lot 2.5,
//	                qui fait descendre le profil du film jusqu aux balayages de `replay`.
//	CRITERE MESURABLE `replay.BuildFromFilm` recoit le profil de mouvement de son appelant —
//	                donc `replaybuild` lui passe explicitement ce que la calibration a retenu —
//	                et `filmdecVarsGeles` tombe a 0 variable mutable.
//
// Le site qui EXIGE ce relais est hors du perimetre du lot 2.2 : `replay/build_from_film.go`
// (tenu par le lot 2.7p) et `internal/replaybuild/replaybuild.go`. Il est consigne en §4 du
// plan (decouverte « l heritage de calibration entre killsource et la cuisson du rejeu »).
//
// # CE QU IL N EST PAS
//
// Ce n est PAS un reglage : rien ne l ecrit hors de la calibration de `killsource`, et sa
// valeur au repos est EXACTEMENT l invariant du profil ([mouvementDuProfil]). Un balayage qui
// tient son propre profil le pose sur son lecteur ([BitReader.poserMouvement]) et n en depend
// plus : c est deja le cas de toutes les portes a [FrameConfig].

// profilHerite : TOUT ce qu une passe laisse a la suivante dans le processus.
//
// UNE SEULE STRUCTURE, ET C EST LE POINT. Ces valeurs vivaient dans SIX variables de paquet
// distinctes (deux du chemin de position, deux du bloc `object-multiplayer-properties`, deux du
// `param_4` par composant) dont les durees de vie se geraient a la main, une par une — le genre
// d oubli qui laisse la calibration d un match fuir sur le suivant. Groupees, elles se posent,
// se lisent et se remettent a zero d un seul geste, et le jour du retrait il n y a qu une chose
// a retirer.
type profilHerite struct {
	// mouvement : descripteur de traversee, largeur d axe absolue, descripteur world-object,
	// range de dequantification, quantum et largeur du delta, drapeaux de contexte.
	mouvement MovementProfile
	// mpp : le decoupage des deux champs de largeur variable du bloc
	// `object-multiplayer-properties`, pose par la VERSION DE FORMAT du film.
	mpp MPPWidths
	// rsp : le `param_4` du moteur qu un harnais de balayage a force, et le drapeau qui dit
	// qu il l a force. Hors balayage, la table par composant decide seule.
	rsp       uint32
	rspImpose bool
}

// herite : l instance. Sa valeur AU REPOS est EXACTEMENT ce que le profil pose.
var herite = profilHerite{mouvement: mouvementDuProfil(), mpp: mppDuProfil()}

// PoserMouvementHerite installe le profil de mouvement que les balayages SUIVANTS du processus
// prendront par defaut. Le seul appelant de production est la calibration de `killsource`.
//
// L APPELANT DOIT DETENIR `LockProcessDecode` — meme contrat que l installateur des largeurs
// d axe de la carte, cote `replay`, et pour la meme raison : c est un etat de processus.
func PoserMouvementHerite(m MovementProfile) { herite.mouvement = m }

// ReinitialiserMouvementHerite remet l invariant du profil SUR LE SEUL MOUVEMENT. Appele en tete
// de `killsource.Decode` (`resetGlobals`) : sans cela, enchainer deux films dans le meme
// processus ferait demarrer la calibration du second depuis les valeurs du premier.
//
// IL NE TOUCHE NI AUX LARGEURS MPP NI AU `param_4`, ET C EST DELIBERE : `resetGlobals` ne les
// remettait pas non plus avant le lot 2.2.e. Les largeurs MPP sont POSEES ET RESTAUREES par
// leur installateur (`InstallFilmFormatMPP` rend sa restauration, l appelant la differe), donc
// equilibrees par construction ; le `param_4`, lui, est remis par `SetRecordStateParam(0)` juste
// apres, comme avant. Elargir cette remise a zero serait un changement de comportement, pas un
// nettoyage — exactement ce que le critere D4 du jalon interdit.
func ReinitialiserMouvementHerite() { herite.mouvement = mouvementDuProfil() }

// MouvementHerite rend le profil de mouvement herite. Lecture seule ; il sert aux instruments et
// aux garde-rails qui verifient que le repos vaut bien l invariant.
func MouvementHerite() MovementProfile { return herite.mouvement }
