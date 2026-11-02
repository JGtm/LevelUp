//! Lecture de lignes bornee (A14).
//!
//! La coquille vide la sortie du serveur en continu : un tube plein bloque le serveur. Une ligne
//! sans fin (journal egare, binaire) ne doit ni epuiser la memoire ni arreter la lecture : au-dela
//! de `max` octets le reste de la ligne est lu et jete, et la ligne est marquee tronquee.

use std::io::{self, BufRead, ErrorKind};

#[derive(Debug, PartialEq, Eq)]
pub struct BoundedLine {
    pub text: String,
    /// Vrai si la ligne depassait la borne et a ete coupee.
    pub truncated: bool,
}

/// Lit la ligne suivante ; `None` a la fin du flux. `\n` et `\r\n` terminent une ligne ; un flux
/// qui finit sans saut de ligne rend quand meme sa derniere ligne.
pub fn read_line_bounded<R: BufRead>(reader: &mut R, max: usize) -> io::Result<Option<BoundedLine>> {
    let mut buf: Vec<u8> = Vec::new();
    let mut truncated = false;
    let mut seen_any = false;
    loop {
        let chunk = match reader.fill_buf() {
            Ok(chunk) => chunk,
            Err(e) if e.kind() == ErrorKind::Interrupted => continue,
            Err(e) => return Err(e),
        };
        if chunk.is_empty() {
            break;
        }
        seen_any = true;
        let (take, newline) = match chunk.iter().position(|b| *b == b'\n') {
            Some(i) => (i + 1, true),
            None => (chunk.len(), false),
        };
        // Le contenu de la ligne, sans son saut de ligne : c'est lui qui est borne.
        let content = if newline { take - 1 } else { take };
        let keep = content.min(max.saturating_sub(buf.len()));
        buf.extend_from_slice(&chunk[..keep]);
        truncated |= keep < content;
        reader.consume(take);
        if newline {
            break;
        }
    }
    if !seen_any {
        return Ok(None);
    }
    while buf.last() == Some(&b'\r') {
        buf.pop();
    }
    Ok(Some(BoundedLine {
        text: String::from_utf8_lossy(&buf).into_owned(),
        truncated,
    }))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Cursor;

    fn read_all(input: &[u8], max: usize) -> Vec<BoundedLine> {
        let mut r = Cursor::new(input.to_vec());
        let mut out = Vec::new();
        while let Some(line) = read_line_bounded(&mut r, max).unwrap() {
            out.push(line);
        }
        out
    }

    fn line(text: &str, truncated: bool) -> BoundedLine {
        BoundedLine {
            text: text.to_string(),
            truncated,
        }
    }

    #[test]
    fn lit_les_lignes_unix_et_windows_et_la_derniere_sans_saut() {
        assert_eq!(
            read_all(b"a\nb\r\nc", 64),
            vec![line("a", false), line("b", false), line("c", false)]
        );
        assert_eq!(read_all(b"", 64), vec![]);
        assert_eq!(read_all(b"\n", 64), vec![line("", false)]);
    }

    #[test]
    fn une_ligne_trop_longue_est_coupee_et_la_suivante_reste_intacte() {
        let mut input = vec![b'x'; 1000];
        input.extend_from_slice(b"\nsuite\n");
        assert_eq!(
            read_all(&input, 10),
            vec![line(&"x".repeat(10), true), line("suite", false)]
        );
    }

    #[test]
    fn une_ligne_exactement_a_la_borne_n_est_pas_tronquee() {
        assert_eq!(read_all(b"1234567890\n", 10), vec![line("1234567890", false)]);
        assert_eq!(read_all(b"12345678901\n", 10), vec![line("1234567890", true)]);
    }

    #[test]
    fn une_ligne_sans_fin_est_lue_jusqu_au_bout_du_flux_sans_stocker_le_reste() {
        let input = vec![b'z'; 5 * 1024 * 1024];
        assert_eq!(read_all(&input, 16), vec![line(&"z".repeat(16), true)]);
    }

    #[test]
    fn un_octet_invalide_ne_fait_pas_echouer_la_lecture() {
        let got = read_all(b"ok\xff\xfe\n", 64);
        assert_eq!(got.len(), 1);
        assert!(got[0].text.starts_with("ok"));
    }
}
