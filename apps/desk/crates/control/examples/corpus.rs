use std::io::{BufWriter, Write};

const LINES: [&str; 10] = [
    r#"{"jsonrpc":"2.0","id":1,"method":"tile.open","params":{"module":"editor","target":{"edge":"left"}}}"#,
    r#"{"jsonrpc":"2.0","id":2,"method":"desk.ping"}"#,
    r#"{"jsonrpc":"2.0","id":3,"method":"workspace.switch","params":{"to":"next"}}"#,
    r#"{"jsonrpc":"2.0","id":4,"method":"file.open","params":{"path":"src/main.rs","line":12,"col":4}}"#,
    r#"{"jsonrpc":"2.0","id":5,"method":"command.run","params":{"name":"notepad.append","args":{"text":"hi"},"dry_run":true}}"#,
    r#"{"jsonrpc":"2.0","id":6,"method":"editor.text.replace","params":{"path":"a.rs","range":{"start":{"l":1,"c":0},"end":{"l":1,"c":4}},"text":"fn"}}"#,
    r#"{"jsonrpc":"2.0","id":7,"method":"layout.apply","params":{"layout":{"preset":"work"},"dry_run":true}}"#,
    r#"{"jsonrpc":"2.0","id":8,"method":"state.get","params":{"uri":"desk://focus"}}"#,
    r#"{"jsonrpc":"2.0","id":9,"method":"nope"}"#,
    r#"{"jsonrpc":"2.0","id":10,"method":"tile.resize","params":{"id":3,"side":"left"}}"#,
];

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let count: usize = match std::env::args().nth(1) {
        Some(text) => text.parse()?,
        None => 10_000,
    };
    let mut out = BufWriter::new(std::io::stdout().lock());
    for line in LINES.iter().cycle().take(count) {
        writeln!(out, "{line}")?;
    }
    out.flush()?;
    Ok(())
}
