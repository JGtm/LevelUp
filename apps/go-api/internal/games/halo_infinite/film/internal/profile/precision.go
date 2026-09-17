package profile

// precision.go — LE DESCRIPTEUR DE QUANTIFICATION D UN CHEMIN DE POSITION.
//
// EXTRAIT DE `grammar/components_movement.go` AU LOT 2.5.b. Le type n a aucune methode et
// aucune logique : c est une largeur d index, trois largeurs d axe et une valeur d index de
// region — de la donnee de carte. Il est le champ de [MovementProfile.Traversal] et de
// [MovementProfile.WorldObject], donc il ne pouvait pas rester au-dessus du profil sans faire
// remonter `profile` vers `grammar`. Les LECTEURS qui s en servent (le chemin de traversee, le
// chemin world-object) restent, eux, en `grammar`.

// PrecisionDescriptor carries the runtime-populated position quantization widths
// (DAT_144632be0 index width + the 3 per-axis widths from DAT_1445cc9e0..) that
// FUN_140be9a14 installs at map/replication-config load. The film's own precision
// block is the source of truth; statically the .exe tables read 0, so a real
// decode MUST supply these from the film header. AxisW are the FUN_140cc5128
// per-axis widths; IndexW is the FUN_14076e524 gate-path index width.
type PrecisionDescriptor struct {
	IndexW uint
	AxisW  [3]uint
	// Region est la VALEUR d'index de région attendue sur les records, sur `IndexW` bits.
	// N'a de sens que pour le descripteur WORLD-OBJECT (`WorldObjectPrecision`), dont le
	// lecteur compare l'index lu à cette valeur — un record d'une AUTRE région exprime ses
	// quanta dans une autre AABB. Le chemin de TRAVERSÉE (`Lecteur.traversal`) ne la lit
	// pas : il consomme l'index sans le juger, parce qu'il ne déquantifie pas.
	//
	// ELLE VIT DANS LE DESCRIPTEUR, et pas à côté, pour une raison précise :
	// `replay.installWorldObjectPrecision` sauve et restaure le descripteur PAR VALEUR. Un
	// second global aurait une durée de vie à gérer à la main, et c'est exactement le genre
	// d'oubli qui laisse la région d'un match fuir sur le suivant.
	Region uint32
}
