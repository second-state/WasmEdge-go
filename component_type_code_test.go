package wasmedge

import "testing"

func TestComponentTypeCodeValuesAndStrings(t *testing.T) {
	tests := []struct {
		name string
		code ComponentTypeCode
		want uint8
		text string
	}{
		{name: "type index", code: ComponentTypeCodeTypeIndex, want: 0x00, text: "type_index"},
		{name: "bool", code: ComponentTypeCodeBool, want: 0x7F, text: "bool"},
		{name: "s8", code: ComponentTypeCodeS8, want: 0x7E, text: "s8"},
		{name: "u8", code: ComponentTypeCodeU8, want: 0x7D, text: "u8"},
		{name: "s16", code: ComponentTypeCodeS16, want: 0x7C, text: "s16"},
		{name: "u16", code: ComponentTypeCodeU16, want: 0x7B, text: "u16"},
		{name: "s32", code: ComponentTypeCodeS32, want: 0x7A, text: "s32"},
		{name: "u32", code: ComponentTypeCodeU32, want: 0x79, text: "u32"},
		{name: "s64", code: ComponentTypeCodeS64, want: 0x78, text: "s64"},
		{name: "u64", code: ComponentTypeCodeU64, want: 0x77, text: "u64"},
		{name: "f32", code: ComponentTypeCodeF32, want: 0x76, text: "f32"},
		{name: "f64", code: ComponentTypeCodeF64, want: 0x75, text: "f64"},
		{name: "char", code: ComponentTypeCodeChar, want: 0x74, text: "char"},
		{name: "string", code: ComponentTypeCodeString, want: 0x73, text: "string"},
		{name: "record", code: ComponentTypeCodeRecord, want: 0x72, text: "record"},
		{name: "variant", code: ComponentTypeCodeVariant, want: 0x71, text: "variant"},
		{name: "list", code: ComponentTypeCodeList, want: 0x70, text: "list"},
		{name: "tuple", code: ComponentTypeCodeTuple, want: 0x6F, text: "tuple"},
		{name: "flags", code: ComponentTypeCodeFlags, want: 0x6E, text: "flags"},
		{name: "enum", code: ComponentTypeCodeEnum, want: 0x6D, text: "enum"},
		{name: "option", code: ComponentTypeCodeOption, want: 0x6B, text: "enum"},
		{name: "result", code: ComponentTypeCodeResult, want: 0x6A, text: "result"},
		{name: "own", code: ComponentTypeCodeOwn, want: 0x69, text: "own"},
		{name: "borrow", code: ComponentTypeCodeBorrow, want: 0x68, text: "own"},
		{name: "list length", code: ComponentTypeCodeListLen, want: 0x67, text: "list_len"},
		{name: "stream", code: ComponentTypeCodeStream, want: 0x66, text: "stream"},
		{name: "future", code: ComponentTypeCodeFuture, want: 0x65, text: "future"},
		{name: "error context", code: ComponentTypeCodeErrContext, want: 0x64, text: "error-context"},
	}

	seen := make(map[ComponentTypeCode]string, len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uint8(tt.code); got != tt.want {
				t.Fatalf("%s = 0x%02x, want 0x%02x", tt.name, got, tt.want)
			}
			if got := tt.code.String(); got != tt.text {
				t.Fatalf("%s.String() = %q, want %q", tt.name, got, tt.text)
			}
		})
		if previous, ok := seen[tt.code]; ok {
			t.Fatalf("%s and %s share code 0x%02x", previous, tt.name, uint8(tt.code))
		}
		seen[tt.code] = tt.name
	}

	if len(seen) != 28 {
		t.Fatalf("tested %d component type codes, want 28", len(seen))
	}
}

func TestComponentTypeCodeStringUnknown(t *testing.T) {
	const code = ComponentTypeCode(0xFF)
	if got, want := code.String(), "component type code 0xff"; got != want {
		t.Fatalf("ComponentTypeCode(0xff).String() = %q, want %q", got, want)
	}
}
