package store

import "testing"

func TestParseMcpServerConfigStdio(t *testing.T) {
	server, err := ParseMcpServerConfig("local", "stdio", `{"command":"node","args":["server.js"],"env":{"FOO":"bar"}}`)
	if err != nil {
		t.Fatalf("ParseMcpServerConfig: %v", err)
	}
	if server.Type != "stdio" || server.Command != "node" {
		t.Fatalf("unexpected stdio server: %#v", server)
	}
	if len(server.Args) != 1 || server.Args[0] != "server.js" {
		t.Fatalf("unexpected args: %#v", server.Args)
	}
	if server.Env["FOO"] != "bar" {
		t.Fatalf("unexpected env: %#v", server.Env)
	}
}

func TestParseMcpServerConfigRejectsInvalidSchema(t *testing.T) {
	if _, err := ParseMcpServerConfig("broken", "stdio", `{"args":["server.js"]}`); err == nil {
		t.Fatal("expected stdio config error")
	}
	if _, err := ParseMcpServerConfig("remote", "http", `{"url":"not-a-url"}`); err == nil {
		t.Fatal("expected remote url error")
	}
	if _, err := ParseMcpServerConfig("extra", "sse", `{"url":"https://example.com","extra":true}`); err == nil {
		t.Fatal("expected unknown field error")
	}
}
