use crate::error::ErrorCode;
use crate::verbs::{Call, Method};
use schemars::JsonSchema;
use serde::de::IntoDeserializer;
use serde::de::value::Error as NameError;
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
pub enum JsonRpc {
    #[serde(rename = "2.0")]
    V2,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize, JsonSchema)]
#[serde(untagged)]
pub enum Id {
    Number(i64),
    Text(String),
}

#[derive(Debug, Clone, Serialize, Deserialize, JsonSchema)]
#[serde(deny_unknown_fields)]
pub struct WireRequest {
    pub jsonrpc: JsonRpc,
    pub id: Id,
    pub method: String,
    #[serde(default)]
    pub params: Option<Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize, JsonSchema)]
pub struct Success<T> {
    pub jsonrpc: JsonRpc,
    pub id: Id,
    pub result: T,
}

#[derive(Debug, Clone, Serialize, Deserialize, JsonSchema)]
pub struct Failure {
    pub jsonrpc: JsonRpc,
    pub id: Option<Id>,
    pub error: RpcError,
}

#[derive(Debug, Clone, Serialize, Deserialize, JsonSchema)]
pub struct RpcError {
    pub code: i32,
    pub message: String,
    pub data: ErrorData,
}

#[derive(Debug, Clone, Serialize, Deserialize, JsonSchema)]
pub struct ErrorData {
    pub kind: ErrorCode,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub method: Option<String>,
}

#[derive(Debug, Clone)]
pub struct Request {
    pub id: Id,
    pub call: Call,
}

impl Failure {
    pub fn new(id: Option<Id>, kind: ErrorCode, message: String, method: Option<String>) -> Self {
        Failure {
            jsonrpc: JsonRpc::V2,
            id,
            error: RpcError {
                code: kind.rpc_code(),
                message,
                data: ErrorData { kind, method },
            },
        }
    }
}

pub fn parse_line(line: &str) -> Result<Request, Failure> {
    let value: Value = serde_json::from_str(line)
        .map_err(|e| Failure::new(None, ErrorCode::ParseError, e.to_string(), None))?;
    let wire: WireRequest = serde_json::from_value(value)
        .map_err(|e| Failure::new(None, ErrorCode::InvalidRequest, e.to_string(), None))?;
    let named: Result<Method, NameError> =
        Method::deserialize(wire.method.as_str().into_deserializer());
    let Ok(method) = named else {
        let message = format!("unknown method `{}`", wire.method);
        return Err(Failure::new(
            Some(wire.id),
            ErrorCode::UnknownMethod,
            message,
            Some(wire.method),
        ));
    };
    let params = wire.params.unwrap_or_else(|| Value::Object(Map::new()));
    match method.call(params) {
        Ok(call) => Ok(Request { id: wire.id, call }),
        Err(e) => Err(Failure::new(
            Some(wire.id),
            ErrorCode::InvalidParams,
            e.to_string(),
            Some(wire.method),
        )),
    }
}
