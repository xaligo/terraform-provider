package common

type BlockKind string

const (
	BlockResource BlockKind = "resource"
	BlockData     BlockKind = "data"
	BlockModule   BlockKind = "module"
)

type ValueKind uint8

const (
	ValueUnknown ValueKind = iota
	ValueString
	ValueBool
	ValueNumber
	ValueObject
	ValueList
)

// Value is the conservative constant subset needed by mapping rules.
// Expressions that require Terraform evaluation remain unknown.
type Value struct {
	Kind   ValueKind
	String string
	Bool   bool
	Object map[string]Value
	List   []Value
}

func (rcvr Value) AsString() (string, bool) {
	return rcvr.String, rcvr.Kind == ValueString
}

func (rcvr Value) AsBool() (bool, bool) {
	return rcvr.Bool, rcvr.Kind == ValueBool
}

func (rcvr Value) ObjectString(key string) (string, bool) {
	if rcvr.Kind != ValueObject {
		return "", false
	}
	value, ok := rcvr.Object[key]
	if !ok {
		return "", false
	}
	return value.AsString()
}

type Attribute struct {
	Name       string
	Value      Value
	References []string
	Range      SourceRange
}

type Block struct {
	Kind         BlockKind
	Type         string
	Name         string
	Address      string
	Attributes   map[string]Attribute
	References   []string
	Range        SourceRange
	NestedBlocks []string
}

type SourceFile struct {
	Path  string
	Bytes []byte
}

type TerraformConfig struct {
	Directory string
	Files     []SourceFile
	Blocks    []Block
}
