package gencommon

import (
	"os"
	"strings"
	"testing"
)

const fixtureGuardsCode = `package model

import (
	"net/netip"
	"time"
)

type Key struct {
	A int64
}

type Guarded struct {
	Keep      string
	Zero      [0]int32
	ArrayKey  map[[2]byte]int64
	PtrKey    map[*string]int64
	TextKey   map[netip.Addr]int64
	TimeKey   map[time.Time]int64
	StructKey map[Key]int64
	Keep2     int64
}
`

// TestParse_SkipsShapesTheOutputCannotCarry checks that a zero-length array
// and the map keys the generated code cannot handle are skipped with a
// warning, and that the fields around them survive.
func TestParse_SkipsShapesTheOutputCannotCarry(t *testing.T) {
	modDir := writeModelFixture(t, fixtureGuardsCode)
	var warnings strings.Builder
	SetWarnOutput(&warnings)
	t.Cleanup(func() { SetWarnOutput(os.Stderr) })

	_, _, structs, err := Parse([]string{modDir}, []string{"Guarded"}, false)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	var guarded *StructInfo
	for i := range structs {
		if structs[i].Name == "Guarded" {
			guarded = &structs[i]
		}
	}
	if guarded == nil {
		t.Fatal("Guarded not found in Parse() results")
	}
	var names []string
	for _, f := range guarded.Fields {
		names = append(names, f.Name)
	}
	if strings.Join(names, ",") != "Keep,Keep2" {
		t.Errorf("Guarded fields = %v, want Keep and Keep2", names)
	}
	for _, want := range []string{
		"Skipping field Zero in Guarded: zero-length arrays are not supported",
		"Skipping field ArrayKey in Guarded: map key type [2]uint8 is not supported",
		"Skipping field PtrKey in Guarded: map key type *string is not supported",
		"Skipping field TextKey in Guarded: map key type netip.Addr is not supported",
		"Skipping field TimeKey in Guarded: map key type time.Time is not supported",
		"Skipping field StructKey in Guarded: map key type Key is not supported",
	} {
		if !strings.Contains(warnings.String(), want) {
			t.Errorf("warnings lack %q:\n%s", want, warnings.String())
		}
	}
}

const fixtureRecursiveCode = `package model

type Node struct {
	Name     string
	Children []*Node
}

type Tree struct {
	Root *Node
}
`

// TestParse_RejectsRecursiveStructs checks that a struct referring to itself
// stops generation instead of producing a schema constructor that never
// returns.
func TestParse_RejectsRecursiveStructs(t *testing.T) {
	modDir := writeModelFixture(t, fixtureRecursiveCode)
	_, _, _, err := Parse([]string{modDir}, []string{"Tree"}, false)
	if err == nil {
		t.Fatal("Parse() accepted a self-referential struct")
	}
	if !strings.Contains(err.Error(), "struct Node refers to itself through Tree -> Node -> Node") {
		t.Errorf("error = %q, want the cycle spelled out", err)
	}
}

const fixtureEmptyCode = `package model

type Empty struct{}

type Holder struct {
	ID string
	E  Empty
}
`

// TestCheckEmpty_RejectsEmptyStructs checks that a struct with no fields is
// reported before generation, since Parquet cannot store it and the reader
// for it does not compile. Parse itself still returns it, so resolution can
// be inspected.
func TestCheckEmpty_RejectsEmptyStructs(t *testing.T) {
	modDir := writeModelFixture(t, fixtureEmptyCode)
	_, _, structs, err := Parse([]string{modDir}, []string{"Holder"}, false)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	err = CheckEmpty(structs)
	if err == nil {
		t.Fatal("CheckEmpty accepted a struct with no fields")
	}
	if !strings.Contains(err.Error(), "struct Empty has no field the generators can map") {
		t.Errorf("error = %q", err)
	}
}
