package port

import "levelup/go-api/internal/domain"

// ErrNotFound : une lecture n'a trouvé aucune ligne. Les dépôts la rendent (jointe à la cause du moteur de
// base, dont le texte est conservé) et la couche métier la reconnaît par errors.Is ; aucun service ne
// compare plus à un sentinelle du moteur de base. Une panne de lecture n'est jamais ErrNotFound.
var ErrNotFound = domain.ErrRecordNotFound
