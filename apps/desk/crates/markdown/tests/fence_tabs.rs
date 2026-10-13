use desk_markdown::{BlockKind, Options, parse};

#[test]
fn go_fence_body_keeps_leading_tabs() {
    let doc = parse(
        "```go\nfunc main() {\n\tfmt.Println(1)\n}\n```",
        Options::ALL,
    );
    let Some(BlockKind::Code { text, .. }) = doc.blocks.first().map(|b| &b.kind) else {
        panic!("no code block in {:?}", doc.blocks.len());
    };
    println!("fence body: {text:?}");
    assert_eq!(text, "func main() {\n\tfmt.Println(1)\n}\n");
}
