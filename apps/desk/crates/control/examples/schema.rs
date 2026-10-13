use std::io::Write;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let text = serde_json::to_string_pretty(&desk_control::schema())?;
    let mut out = std::io::stdout().lock();
    writeln!(out, "{text}")?;
    out.flush()?;
    Ok(())
}
