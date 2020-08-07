package message

import "testing"

func TestMessageMarshalUnmarshal(t *testing.T) {
	m := New("default", []byte("Hello, World"))

	msgBytes, err := m.Marshal()
	if err != nil {
		t.Error(err)
	}

	m2, err := NewFromBytes(msgBytes)
	if err != nil {
		t.Error(err)
	}

	data := m2.Data()
	if string(data) != "Hello, World" {
		t.Errorf("expected Hello, World, got %s", string(data))
	}
}
