use std::io::{BufRead, BufWriter, Write};
use std::time::{Duration, Instant};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut out = BufWriter::new(std::io::stdout().lock());
    let mut spent = Duration::ZERO;
    let mut lines: u128 = 0;
    let mut failed: u128 = 0;
    for line in std::io::stdin().lock().lines() {
        let line = line?;
        let started = Instant::now();
        let parsed = desk_control::parse_line(&line);
        spent = spent.saturating_add(started.elapsed());
        lines = lines.saturating_add(1);
        match parsed {
            Ok(request) => writeln!(out, "{request:?}")?,
            Err(failure) => {
                failed = failed.saturating_add(1);
                writeln!(out, "{}", serde_json::to_string(&failure)?)?;
            }
        }
    }
    out.flush()?;
    let per_line = spent.as_nanos().checked_div(lines).unwrap_or(0);
    eprintln!(
        "parsed {lines} lines, {failed} errors, {} us total, {per_line} ns per line",
        spent.as_micros()
    );
    Ok(())
}
