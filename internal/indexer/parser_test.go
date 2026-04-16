package indexer_test

import (
	"testing"

	"github.com/TruyLabs/rias/internal/indexer"
)

const goSrc = `package main

import "fmt"

type User struct {
	Name string
	Age  int
}

func NewUser(name string) *User {
	return &User{Name: name}
}

func (u *User) Greet() string {
	return fmt.Sprintf("Hello, %s", u.Name)
}

const Version = "1.0.0"
`

func TestExtractGoSymbols(t *testing.T) {
	symbols := indexer.ExtractSymbols("main.go", []byte(goSrc))

	if len(symbols) == 0 {
		t.Fatal("expected symbols, got none")
	}

	names := make(map[string]string)
	for _, s := range symbols {
		names[s.Name] = s.Kind
	}

	if names["User"] != "type" {
		t.Errorf("expected User to be type, got %q", names["User"])
	}
	if names["NewUser"] != "func" {
		t.Errorf("expected NewUser to be func, got %q", names["NewUser"])
	}
	if names["Greet"] != "method" {
		t.Errorf("expected Greet to be method, got %q", names["Greet"])
	}
}

func TestExtractGoSymbolsCodeNotEmpty(t *testing.T) {
	symbols := indexer.ExtractSymbols("main.go", []byte(goSrc))
	for _, s := range symbols {
		if s.Code == "" {
			t.Errorf("symbol %q has empty Code", s.Name)
		}
	}
}

func TestExtractSymbolsNonGoFile(t *testing.T) {
	tsSrc := `export function greet(name: string): string {
  return "Hello " + name;
}

export class UserService {
  create(name: string) { return name; }
}
`
	symbols := indexer.ExtractSymbols("service.ts", []byte(tsSrc))
	if len(symbols) == 0 {
		t.Fatal("expected at least one symbol from TypeScript file")
	}
	names := make(map[string]bool)
	for _, s := range symbols {
		names[s.Name] = true
	}
	if !names["greet"] && !names["UserService"] {
		t.Errorf("expected greet or UserService in symbols, got: %v", symbols)
	}
}

func TestExtractSymbolsEmptyFile(t *testing.T) {
	symbols := indexer.ExtractSymbols("empty.go", []byte("package main\n"))
	// Empty file is fine — just no symbols
	_ = symbols
}
