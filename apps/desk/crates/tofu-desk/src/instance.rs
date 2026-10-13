use serde::Deserialize;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::time::{Duration, Instant};

pub const CONNECT_TIMEOUT: Duration = Duration::from_secs(2);
const PROBE_POLL: Duration = Duration::from_millis(5);

#[derive(Deserialize)]
struct Instance {
    pid: u32,
}

pub enum Found {
    Running,
    NotRunning,
}

pub fn find(deadline: Instant) -> std::io::Result<Found> {
    let Some(dir) = runtime_dir() else {
        return Ok(Found::NotRunning);
    };
    let entries = match std::fs::read_dir(&dir) {
        Ok(entries) => entries,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(Found::NotRunning),
        Err(e) => return Err(e),
    };
    let mut found = Found::NotRunning;
    for entry in entries {
        let path = entry?.path();
        let name = path.file_name().and_then(|n| n.to_str()).unwrap_or("");
        if !(name.starts_with("instance") && name.ends_with(".json")) {
            continue;
        }
        let alive = match serde_json::from_slice::<Instance>(&std::fs::read(&path)?) {
            Ok(instance) => alive(instance.pid, deadline)?,
            Err(_) => Some(false),
        };
        match alive {
            Some(true) => found = Found::Running,
            Some(false) => remove_stale(&path)?,
            None => {}
        }
    }
    Ok(found)
}

fn runtime_dir() -> Option<PathBuf> {
    let home = if cfg!(windows) { "USERPROFILE" } else { "HOME" };
    let home = std::env::var_os(home).filter(|h| !h.is_empty())?;
    Some(
        PathBuf::from(home)
            .join(".tofu")
            .join("desk")
            .join("runtime"),
    )
}

fn remove_stale(path: &Path) -> std::io::Result<()> {
    match std::fs::remove_file(path) {
        Err(e) if e.kind() != std::io::ErrorKind::NotFound => Err(e),
        _ => Ok(()),
    }
}

fn alive(pid: u32, deadline: Instant) -> std::io::Result<Option<bool>> {
    let mut probe = if cfg!(windows) {
        let mut c = Command::new("tasklist");
        c.args(["/FI", &format!("PID eq {pid}"), "/NH", "/FO", "CSV"]);
        c
    } else {
        let mut c = Command::new("kill");
        c.args(["-0", &pid.to_string()]);
        c
    };
    let mut child = probe
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .spawn()?;
    let status = loop {
        if let Some(status) = child.try_wait()? {
            break status;
        }
        if Instant::now() >= deadline {
            child.kill()?;
            child.wait()?;
            return Ok(None);
        }
        std::thread::sleep(PROBE_POLL);
    };
    if !cfg!(windows) {
        return Ok(Some(status.success()));
    }
    let output = child.wait_with_output()?;
    let listed = String::from_utf8_lossy(&output.stdout);
    Ok(Some(listed.contains(&format!(",\"{pid}\","))))
}
