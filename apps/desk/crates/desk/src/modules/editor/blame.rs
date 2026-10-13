use std::time::{SystemTime, UNIX_EPOCH};

use desk_core::git;
use gpui::SharedString;

const MINUTE: i64 = 60;
const HOUR: i64 = 60 * MINUTE;
const DAY: i64 = 24 * HOUR;
const WEEK: i64 = 7 * DAY;
const MONTH: i64 = 30 * DAY;
const YEAR: i64 = 365 * DAY;
const YOU: &str = "You";
const NOT_COMMITTED: &str = "not committed";

pub struct Blamed {
    pub text: String,
    pub lines: Vec<git::BlameLine>,
    pub own_email: Option<String>,
}

#[derive(Clone, PartialEq)]
pub struct Inline {
    pub row: usize,
    pub user: Option<SharedString>,
    pub author: SharedString,
    pub email: SharedString,
    pub when: SharedString,
    pub mine: bool,
}

impl Inline {
    pub fn name(&self) -> SharedString {
        if self.mine {
            return YOU.into();
        }
        Some(self.author.clone())
            .filter(|author| !author.trim().is_empty())
            .unwrap_or_else(|| self.email.clone())
    }
}

pub struct Me<'a> {
    pub email: Option<&'a str>,
    pub login: Option<&'a str>,
}

impl Me<'_> {
    fn owns(&self, line: &git::BlameLine) -> bool {
        line.mine
            || self
                .email
                .is_some_and(|email| email.eq_ignore_ascii_case(&line.email))
    }

    fn user(&self, line: &git::BlameLine) -> Option<SharedString> {
        self.login
            .filter(|_| self.owns(line))
            .map(|login| login.to_owned().into())
    }
}

fn now() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .ok()
        .and_then(|since| i64::try_from(since.as_secs()).ok())
        .unwrap_or(0)
}

fn age(seconds: i64) -> String {
    let (count, unit) = match seconds.max(0) {
        seconds if seconds < MINUTE => return "just now".to_owned(),
        seconds if seconds < HOUR => (seconds / MINUTE, "minute"),
        seconds if seconds < DAY => (seconds / HOUR, "hour"),
        seconds if seconds < WEEK => (seconds / DAY, "day"),
        seconds if seconds < MONTH => (seconds / WEEK, "week"),
        seconds if seconds < YEAR => (seconds / MONTH, "month"),
        seconds => (seconds / YEAR, "year"),
    };
    let plural = if count == 1 { "" } else { "s" };
    format!("{count} {unit}{plural} ago")
}

pub fn inline(blamed: &Blamed, row: usize, me: &Me) -> Option<Inline> {
    let line = blamed.lines.get(row)?;
    let email = line.email.clone().into();
    Some(match line.commit {
        None => Inline {
            row,
            user: None,
            author: YOU.into(),
            email,
            when: NOT_COMMITTED.into(),
            mine: true,
        },
        Some(_) => Inline {
            row,
            user: me.user(line),
            author: line.author.clone().into(),
            mine: me.owns(line),
            email,
            when: age(now().saturating_sub(line.time)).into(),
        },
    })
}
