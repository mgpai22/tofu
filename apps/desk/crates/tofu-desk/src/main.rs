mod instance;

use desk_control::{ErrorCode, Method};
use instance::{CONNECT_TIMEOUT, Found};
use schemars::SchemaGenerator;
use serde_json::{Map, Value, json};
use std::io::Write;
use std::process::ExitCode;
use std::time::{Instant, SystemTime, UNIX_EPOCH};

enum Exit {
    Done,
    Failure,
    Usage,
    NotRunning,
}

impl Exit {
    fn code(self) -> ExitCode {
        ExitCode::from(match self {
            Exit::Done => 0,
            Exit::Failure => 1,
            Exit::Usage => 2,
            Exit::NotRunning => 3,
        })
    }
}

struct Outcome {
    exit: Exit,
    code: &'static str,
    message: String,
    hint: String,
}

fn main() -> ExitCode {
    let started = Instant::now();
    let args: Vec<String> = std::env::args().skip(1).collect();
    let json = args.iter().any(|a| a == "--json");
    let args: Vec<&str> = args
        .iter()
        .map(String::as_str)
        .filter(|a| *a != "--json")
        .collect();
    let Some((&verb, rest)) = args.split_first() else {
        return print_plain(&verb_groups(), Exit::Usage);
    };
    if verb == "schema" {
        return match serde_json::to_string_pretty(&desk_control::schema()) {
            Ok(text) => print_plain(&text, Exit::Done),
            Err(e) => print_plain(&e.to_string(), Exit::Failure),
        };
    }
    let outcome = run(verb, rest, started);
    let line = if json {
        envelope(verb, &outcome).to_string()
    } else {
        format!("✗ {}: {} → {}", outcome.code, outcome.message, outcome.hint)
    };
    print_plain(&line, outcome.exit)
}

fn run(verb: &str, rest: &[&str], started: Instant) -> Outcome {
    let Some(&method) = Method::ALL.iter().find(|m| m.name() == verb) else {
        return Outcome {
            exit: Exit::Usage,
            code: ErrorCode::UnknownMethod.name(),
            message: format!("unknown verb `{verb}`"),
            hint: "run tofu-desk with no arguments for the verbs".to_owned(),
        };
    };
    let (fields, required) = fields(method);
    let usage = usage(method, &fields, &required);
    let params = match params(&fields, rest) {
        Ok(params) => params,
        Err(message) => return usage_outcome(message, usage),
    };
    if let Err(e) = method.call(params) {
        return usage_outcome(e.to_string(), usage);
    }
    match instance::find(started + CONNECT_TIMEOUT) {
        Ok(Found::Running) => Outcome {
            exit: Exit::Failure,
            code: "transport_not_built",
            message: "transport not built".to_owned(),
            hint: "a desk is running but this tofu-desk cannot reach it yet".to_owned(),
        },
        Ok(Found::NotRunning) => Outcome {
            exit: Exit::NotRunning,
            code: ErrorCode::DeskNotRunning.name(),
            message: "no desk is running".to_owned(),
            hint: "start the desk, or use tofu for non-desk work".to_owned(),
        },
        Err(e) => Outcome {
            exit: Exit::Failure,
            code: "instance_unreadable",
            message: e.to_string(),
            hint: "check ~/.tofu/desk/runtime".to_owned(),
        },
    }
}

fn usage_outcome(message: String, usage: String) -> Outcome {
    Outcome {
        exit: Exit::Usage,
        code: ErrorCode::BadParams.name(),
        message,
        hint: usage,
    }
}

fn fields(method: Method) -> (Vec<String>, Vec<String>) {
    let mut generator = SchemaGenerator::default();
    let described = method.describe(&mut generator);
    let defs = generator.take_definitions(true);
    let mut schema = &described["params"];
    if let Some(name) = schema["$ref"].as_str().and_then(|r| r.rsplit('/').next()) {
        schema = defs.get(name).unwrap_or(&Value::Null);
    }
    let names = |key: &str| -> Vec<String> {
        match &schema[key] {
            Value::Object(map) => map.keys().cloned().collect(),
            Value::Array(list) => list
                .iter()
                .filter_map(|v| v.as_str().map(str::to_owned))
                .collect(),
            _ => Vec::new(),
        }
    };
    (names("properties"), names("required"))
}

fn params(fields: &[String], rest: &[&str]) -> Result<Value, String> {
    let mut params = Map::new();
    let mut positional = fields.iter();
    let mut rest = rest.iter();
    while let Some(&arg) = rest.next() {
        let (name, raw) = match arg.strip_prefix("--") {
            Some(name) => {
                let raw = rest.next().ok_or(format!("--{name} needs a value"))?;
                (name.to_owned(), *raw)
            }
            None => {
                let name = positional
                    .find(|f| !params.contains_key(*f))
                    .ok_or(format!("unexpected argument `{arg}`"))?;
                (name.clone(), arg)
            }
        };
        let value = serde_json::from_str(raw).unwrap_or_else(|_| Value::String(raw.to_owned()));
        params.insert(name, value);
    }
    Ok(Value::Object(params))
}

fn usage(method: Method, fields: &[String], required: &[String]) -> String {
    let mut line = format!("tofu-desk {}", method.name());
    for field in fields {
        if required.contains(field) {
            line.push_str(&format!(" <{field}>"));
        } else {
            line.push_str(&format!(" [--{field} v]"));
        }
    }
    line.push_str(" [--json]");
    line
}

fn verb_groups() -> String {
    let mut groups: Vec<(&str, Vec<&str>)> = Vec::new();
    for method in Method::ALL {
        let name = method.name();
        let group = name.split('.').next().unwrap_or(name);
        match groups.iter_mut().find(|(g, _)| *g == group) {
            Some((_, verbs)) => verbs.push(name),
            None => groups.push((group, vec![name])),
        }
    }
    let mut text = "usage: tofu-desk <verb> [args] [--json]\n  schema".to_owned();
    for (group, verbs) in groups {
        text.push_str(&format!("\n  {group}: {}", verbs.join(" ")));
    }
    text
}

fn envelope(verb: &str, outcome: &Outcome) -> Value {
    let at = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(0, |d| d.as_millis());
    json!({
        "tofu": "desk",
        "verb": verb,
        "ok": false,
        "at": at,
        "data": null,
        "problems": [{"code": outcome.code, "message": outcome.message, "hint": outcome.hint}],
    })
}

fn print_plain(text: &str, exit: Exit) -> ExitCode {
    let mut out = std::io::stdout().lock();
    match writeln!(out, "{text}").and_then(|()| out.flush()) {
        Ok(()) => exit.code(),
        Err(_) => Exit::Failure.code(),
    }
}
