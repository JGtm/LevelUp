//! Le journal de la coquille, `<racine des journaux>\desktop.log` (ADR 0045 D-2, A14).
//!
//! Y arrivent la sortie d'erreur du serveur (relayee ligne a ligne) et les lignes de la coquille
//! elle-meme. Rotation simple : au-dela de la taille maximale, le fichier devient `desktop.log.1`
//! (le precedent `.1` est remplace) et un nouveau fichier commence. Deux fichiers au plus.

use std::ffi::OsString;
use std::fs::{self, File, OpenOptions};
use std::io::{self, Write};
use std::path::PathBuf;
use std::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH};

/// Taille a partir de laquelle le journal tourne (5 Mo).
pub const MAX_LOG_BYTES: u64 = 5 * 1024 * 1024;

/// Qui parle : la coquille ou le serveur.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Source {
    Shell,
    Sidecar,
}

impl Source {
    fn tag(self) -> &'static str {
        match self {
            Self::Shell => "shell",
            Self::Sidecar => "sidecar",
        }
    }
}

struct Inner {
    path: PathBuf,
    rotated: PathBuf,
    /// `None` le temps d'une rotation ratee : l'ecriture suivante rouvre le journal.
    file: Option<File>,
    size: u64,
    max: u64,
}

pub struct LogSink {
    inner: Mutex<Inner>,
}

impl LogSink {
    /// Ouvre (ou cree) le journal en ajout.
    pub fn open(path: PathBuf, max_bytes: u64) -> io::Result<Self> {
        if let Some(dir) = path.parent() {
            fs::create_dir_all(dir)?;
        }
        let file = OpenOptions::new().create(true).append(true).open(&path)?;
        let size = file.metadata()?.len();
        let mut rotated = OsString::from(path.as_os_str());
        rotated.push(".1");
        Ok(Self {
            inner: Mutex::new(Inner {
                path,
                rotated: PathBuf::from(rotated),
                file: Some(file),
                size,
                max: max_bytes,
            }),
        })
    }

    /// Ecrit une ligne horodatee (UTC). Un echec d'ecriture ne bloque jamais l'appelant : il est
    /// signale une fois par ligne perdue sur la sortie d'erreur de la coquille.
    pub fn write_line(&self, source: Source, text: &str) {
        let line = format!("{} [{}] {}\n", format_utc(now_secs()), source.tag(), text);
        let mut inner = match self.inner.lock() {
            Ok(guard) => guard,
            Err(poisoned) => poisoned.into_inner(),
        };
        if let Err(e) = inner.append(line.as_bytes()) {
            eprintln!("desktop.log : ecriture impossible : {e}");
        }
    }

    /// Le chemin du journal courant.
    pub fn path(&self) -> PathBuf {
        match self.inner.lock() {
            Ok(guard) => guard.path.clone(),
            Err(poisoned) => poisoned.into_inner().path.clone(),
        }
    }
}

impl Inner {
    fn append(&mut self, bytes: &[u8]) -> io::Result<()> {
        if self.size > 0 && self.size + bytes.len() as u64 > self.max {
            self.rotate()?;
        }
        if self.file.is_none() {
            self.file = Some(OpenOptions::new().create(true).append(true).open(&self.path)?);
        }
        if let Some(file) = self.file.as_mut() {
            file.write_all(bytes)?;
            self.size += bytes.len() as u64;
        }
        Ok(())
    }

    /// `desktop.log` -> `desktop.log.1`, puis un fichier neuf. Sous Windows on ne renomme pas par
    /// dessus un fichier existant : l'ancien `.1` est supprime d'abord.
    fn rotate(&mut self) -> io::Result<()> {
        self.file = None; // ferme le descripteur avant de renommer
        if self.rotated.exists() {
            fs::remove_file(&self.rotated)?;
        }
        fs::rename(&self.path, &self.rotated)?;
        self.file = Some(OpenOptions::new().create(true).append(true).open(&self.path)?);
        self.size = 0;
        Ok(())
    }
}

fn now_secs() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

/// `YYYY-MM-DDTHH:MM:SSZ` pour un instant en secondes depuis l'epoque Unix.
pub fn format_utc(secs: u64) -> String {
    let days = (secs / 86_400) as i64;
    let rest = secs % 86_400;
    let (y, m, d) = civil_from_days(days);
    format!(
        "{y:04}-{m:02}-{d:02}T{:02}:{:02}:{:02}Z",
        rest / 3600,
        rest % 3600 / 60,
        rest % 60
    )
}

/// Jour du calendrier gregorien depuis le nombre de jours ecoules depuis 1970-01-01.
fn civil_from_days(days: i64) -> (i64, u32, u32) {
    let z = days + 719_468;
    let era = z.div_euclid(146_097);
    let doe = z.rem_euclid(146_097);
    let yoe = (doe - doe / 1460 + doe / 36_524 - doe / 146_096) / 365;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = (doy - (153 * mp + 2) / 5 + 1) as u32;
    let m = if mp < 10 { mp + 3 } else { mp - 9 } as u32;
    let y = yoe + era * 400 + i64::from(m <= 2);
    (y, m, d)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn temp_log(name: &str) -> PathBuf {
        let dir = std::env::temp_dir().join(format!("levelup-log-{}-{name}", std::process::id()));
        let _ = fs::remove_dir_all(&dir);
        dir.join("desktop.log")
    }

    #[test]
    fn l_horodatage_est_du_utc_exact() {
        assert_eq!(format_utc(0), "1970-01-01T00:00:00Z");
        assert_eq!(format_utc(951_782_400), "2000-02-29T00:00:00Z");
        assert_eq!(format_utc(1_700_000_000), "2023-11-14T22:13:20Z");
        assert_eq!(format_utc(1_782_950_399), "2026-07-01T23:59:59Z");
    }

    #[test]
    fn les_lignes_portent_l_origine_et_s_ajoutent() {
        let path = temp_log("ajout");
        let sink = LogSink::open(path.clone(), MAX_LOG_BYTES).unwrap();
        sink.write_line(Source::Shell, "event=a");
        sink.write_line(Source::Sidecar, "ligne du serveur");
        let text = fs::read_to_string(&path).unwrap();
        let lines: Vec<&str> = text.lines().collect();
        assert_eq!(lines.len(), 2);
        assert!(lines[0].contains(" [shell] event=a"));
        assert!(lines[1].ends_with(" [sidecar] ligne du serveur"));
        // Une reouverture continue le meme fichier.
        LogSink::open(path.clone(), MAX_LOG_BYTES)
            .unwrap()
            .write_line(Source::Shell, "event=b");
        assert_eq!(fs::read_to_string(&path).unwrap().lines().count(), 3);
        fs::remove_dir_all(path.parent().unwrap()).unwrap();
    }

    #[test]
    fn la_rotation_garde_un_seul_ancien_fichier() {
        let path = temp_log("rotation");
        let sink = LogSink::open(path.clone(), 200).unwrap();
        let rotated = PathBuf::from(format!("{}.1", path.display()));
        for i in 0..40 {
            sink.write_line(Source::Sidecar, &format!("ligne numero {i:02} {}", "x".repeat(20)));
        }
        let current = fs::metadata(&path).unwrap().len();
        let previous = fs::metadata(&rotated).unwrap().len();
        assert!(
            current <= 200 && previous <= 200,
            "courant={current} precedent={previous}"
        );
        assert!(!PathBuf::from(format!("{}.2", path.display())).exists());
        // La derniere ligne est dans le fichier courant, jamais perdue par la rotation.
        assert!(fs::read_to_string(&path).unwrap().contains("ligne numero 39"));
        fs::remove_dir_all(path.parent().unwrap()).unwrap();
    }
}
