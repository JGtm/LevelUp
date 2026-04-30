/**
 * Sérialise un objet JavaScript en code JS valide, en préservant les fonctions
 * (qui seraient perdues par JSON.stringify standard).
 *
 * Usage typique : injecter une option ECharts dans un fichier HTML, où les
 * formatters (fonctions) doivent rester exécutables côté navigateur.
 */
export function serializeJS(value: unknown, indent = 2, currentDepth = 0): string {
  const pad = ' '.repeat(currentDepth * indent);
  const padInner = ' '.repeat((currentDepth + 1) * indent);

  if (value === null) return 'null';
  if (value === undefined) return 'undefined';

  if (typeof value === 'function') {
    // toString() conserve le code source de la fonction
    return value.toString();
  }

  if (typeof value === 'string') {
    return JSON.stringify(value);
  }

  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }

  if (Array.isArray(value)) {
    if (value.length === 0) return '[]';
    // Tableaux courts d'éléments simples → inline
    const allSimple = value.every(
      (v) => v === null || ['string', 'number', 'boolean'].includes(typeof v),
    );
    if (allSimple && value.length <= 10) {
      return `[${value.map((v) => serializeJS(v, indent, 0)).join(', ')}]`;
    }
    const items = value.map((v) => padInner + serializeJS(v, indent, currentDepth + 1));
    return `[\n${items.join(',\n')}\n${pad}]`;
  }

  if (typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>);
    if (entries.length === 0) return '{}';
    // Filtre les clés __meta (privé, on garde, mais ok)
    const lines = entries.map(([key, val]) => {
      const safeKey = /^[A-Za-z_$][\w$]*$/.test(key) ? key : JSON.stringify(key);
      return `${padInner}${safeKey}: ${serializeJS(val, indent, currentDepth + 1)}`;
    });
    return `{\n${lines.join(',\n')}\n${pad}}`;
  }

  return 'null';
}
