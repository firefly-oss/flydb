package protocol

import (
	"bytes"
	"testing"
)

func TestWriteAndReadHeader(t *testing.T) {
	tests := []struct {
		name   string
		header Header
	}{
		{
			name: "Query message",
			header: Header{
				Magic:   MagicByte,
				Version: ProtocolVersion,
				Type:    MsgQuery,
				Flags:   FlagNone,
				Length:  100,
			},
		},
		{
			name: "Auth message",
			header: Header{
				Magic:   MagicByte,
				Version: ProtocolVersion,
				Type:    MsgAuth,
				Flags:   FlagNone,
				Length:  50,
			},
		},
		{
			name: "Compressed message",
			header: Header{
				Magic:   MagicByte,
				Version: ProtocolVersion,
				Type:    MsgQueryResult,
				Flags:   FlagCompressed,
				Length:  1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)

			// Write header
			err := WriteHeader(buf, tt.header)
			if err != nil {
				t.Fatalf("WriteHeader failed: %v", err)
			}

			// Read header back
			readHeader, err := ReadHeader(buf)
			if err != nil {
				t.Fatalf("ReadHeader failed: %v", err)
			}

			// Verify
			if readHeader.Magic != tt.header.Magic {
				t.Errorf("Magic mismatch: got %x, want %x", readHeader.Magic, tt.header.Magic)
			}
			if readHeader.Version != tt.header.Version {
				t.Errorf("Version mismatch: got %x, want %x", readHeader.Version, tt.header.Version)
			}
			if readHeader.Type != tt.header.Type {
				t.Errorf("Type mismatch: got %x, want %x", readHeader.Type, tt.header.Type)
			}
			if readHeader.Flags != tt.header.Flags {
				t.Errorf("Flags mismatch: got %x, want %x", readHeader.Flags, tt.header.Flags)
			}
			if readHeader.Length != tt.header.Length {
				t.Errorf("Length mismatch: got %d, want %d", readHeader.Length, tt.header.Length)
			}
		})
	}
}

func TestWriteAndReadMessage(t *testing.T) {
	payload := []byte(`{"query": "SELECT * FROM users"}`)

	buf := new(bytes.Buffer)

	// Write message
	err := WriteMessage(buf, MsgQuery, payload)
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Read message back
	msg, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	// Verify
	if msg.Header.Type != MsgQuery {
		t.Errorf("Type mismatch: got %x, want %x", msg.Header.Type, MsgQuery)
	}
	if !bytes.Equal(msg.Payload, payload) {
		t.Errorf("Payload mismatch: got %s, want %s", msg.Payload, payload)
	}
}

func TestInvalidMagicByte(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00})

	_, err := ReadHeader(buf)
	if err != ErrInvalidMagic {
		t.Errorf("Expected ErrInvalidMagic, got %v", err)
	}
}

func TestInvalidVersion(t *testing.T) {
	buf := bytes.NewBuffer([]byte{MagicByte, 0xFF, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00})

	_, err := ReadHeader(buf)
	if err != ErrInvalidVersion {
		t.Errorf("Expected ErrInvalidVersion, got %v", err)
	}
}

func TestMessageTooLarge(t *testing.T) {
	// Create a header with length > MaxMessageSize
	buf := new(bytes.Buffer)
	h := Header{
		Magic:   MagicByte,
		Version: ProtocolVersion,
		Type:    MsgQuery,
		Flags:   FlagNone,
		Length:  MaxMessageSize + 1,
	}
	WriteHeader(buf, h)

	_, err := ReadHeader(bytes.NewReader(buf.Bytes()))
	if err != ErrMessageTooLarge {
		t.Errorf("Expected ErrMessageTooLarge, got %v", err)
	}
}

func TestEmptyPayload(t *testing.T) {
	buf := new(bytes.Buffer)

	// Write message with empty payload
	err := WriteMessage(buf, MsgPing, nil)
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Read message back
	msg, err := ReadMessage(buf)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if msg.Header.Type != MsgPing {
		t.Errorf("Type mismatch: got %x, want %x", msg.Header.Type, MsgPing)
	}
	if len(msg.Payload) != 0 {
		t.Errorf("Expected empty payload, got %d bytes", len(msg.Payload))
	}
}

