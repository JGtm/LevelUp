package ep

import "fmt"

// Variable représente un nœud de variable dans le factor graph EP.
//
// Une variable stocke (a) son **marginal** courant — produit de tous les
// messages entrants — et (b) la **trace** du dernier message reçu de chaque
// facteur. Cette trace est nécessaire pour calculer le message sortant
// vers un facteur f (= marginal / dernier_message_de_f), opération qui
// garantit qu'EP ne compte pas deux fois l'info venant de f.
//
// Les noms sont volontairement courts (Name) car ils apparaissent surtout
// en debug/logs.
type Variable struct {
	Name     string
	Marginal Gaussian

	// messagesByFactor : dernier message reçu, identifié par un FactorID stable
	// (pointeur de Factor). map plutôt que slice : un facteur ne peut pas
	// apparaître deux fois dans le même graph mais un graph peut grossir
	// dynamiquement → indexer par identité.
	messagesByFactor map[FactorID]Gaussian
}

// FactorID est l'identifiant opaque d'un facteur. Implémenté typiquement par
// le pointeur du struct concret (Factor interface), permet d'indexer une map.
type FactorID interface{}

// NewVariable construit une variable initialement uniforme (aucune info).
func NewVariable(name string) *Variable {
	return &Variable{
		Name:             name,
		Marginal:         UniformGaussian(),
		messagesByFactor: map[FactorID]Gaussian{},
	}
}

// LastMessageFrom retourne le dernier message reçu d'un facteur. Si aucun
// message n'a encore été échangé, retourne Uniform (élément neutre).
func (v *Variable) LastMessageFrom(f FactorID) Gaussian {
	if msg, ok := v.messagesByFactor[f]; ok {
		return msg
	}
	return UniformGaussian()
}

// UpdateMessage met à jour le marginal après réception d'un nouveau message de
// `from`. Retourne l'AbsoluteDifference entre l'ancien et le nouveau marginal,
// utile pour le critère de convergence du runner.
//
// La nouvelle valeur du marginal est :
//
//	new_marginal = (marginal / oldMsgFrom) * newMsgFrom
//
// c'est-à-dire : retire l'ancien contributif du facteur, ajoute le nouveau.
func (v *Variable) UpdateMessage(from FactorID, newMsg Gaussian) float64 {
	prevMarginal := v.Marginal
	oldMsg := v.LastMessageFrom(from)
	v.Marginal = v.Marginal.Div(oldMsg).Mul(newMsg)
	v.messagesByFactor[from] = newMsg
	return v.Marginal.AbsoluteDifference(prevMarginal)
}

// MessageTo retourne le message que la variable envoie au facteur `to` :
//
//	messageTo = marginal / lastMessageFrom(to)
//
// Cancel-out élégant : le facteur destinataire ne reçoit que l'info provenant
// des AUTRES facteurs, jamais ce qu'il a lui-même contribué.
func (v *Variable) MessageTo(to FactorID) Gaussian {
	return v.Marginal.Div(v.LastMessageFrom(to))
}

// Reset remet la variable à uniforme (utile pour réutiliser un graph entre
// matchs sans en reconstruire un nouveau).
func (v *Variable) Reset() {
	v.Marginal = UniformGaussian()
	for k := range v.messagesByFactor {
		delete(v.messagesByFactor, k)
	}
}

// String pour debug/logs.
func (v *Variable) String() string {
	return fmt.Sprintf("Var(%s, %s)", v.Name, v.Marginal)
}
