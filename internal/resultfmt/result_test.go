package resultfmt

import "testing"

func TestEncodeDecodeLine(t *testing.T) {
	input := Result{
		Kind:        KindFile,
		PreviewMode: "file",
		Display:     "foo/bar.go:12:hello",
		Path:        "/tmp/foo/bar.go",
		Line:        12,
	}

	line, err := EncodeLine(input)
	if err != nil {
		t.Fatalf("EncodeLine returned error: %v", err)
	}

	decoded, err := DecodeLine(line)
	if err != nil {
		t.Fatalf("DecodeLine returned error: %v", err)
	}

	if decoded.Path != input.Path || decoded.Line != input.Line || decoded.Display != input.Display {
		t.Fatalf("decoded = %#v, want %#v", decoded, input)
	}
}
