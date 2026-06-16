package cards

import (
	"encoding/json"
	"testing"
	"time"

	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	cardv1 "gophkeeper/internal/shared/proto/card/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// testVault returns an unlocked vault using a fixed 32-byte data key.
func testVault(t *testing.T) *vault.Vault {
	t.Helper()
	v := vault.New()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	if err := v.UseDEK(key); err != nil {
		t.Fatalf("UseDEK: %v", err)
	}
	return v
}

func TestDecodeCardRoundTrip(t *testing.T) {
	v := testVault(t)
	payload := clientmodel.CardData{
		Number: "4111111111111111",
		Holder: "JOHN DOE",
		Expiry: "12/30",
		CVV:    "123",
		Meta:   "personal",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	blob, err := v.Encrypt(raw)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	pb := &cardv1.Card{}
	pb.SetId("card-1")
	pb.SetVersion(7)
	pb.SetData(blob)
	pb.SetCreatedAt(timestamppb.New(time.Unix(1000, 0)))
	pb.SetUpdatedAt(timestamppb.New(time.Unix(2000, 0)))

	got, err := decodeCard(v, pb)
	if err != nil {
		t.Fatalf("decodeCard: %v", err)
	}
	if got.ID != "card-1" {
		t.Errorf("ID = %q, want card-1", got.ID)
	}
	if got.Version != 7 {
		t.Errorf("Version = %d, want 7", got.Version)
	}
	if got.Data != payload {
		t.Errorf("Data = %+v, want %+v", got.Data, payload)
	}
}

func TestDecodeCardCorrupt(t *testing.T) {
	v := testVault(t)
	pb := &cardv1.Card{}
	pb.SetId("card-2")
	pb.SetData([]byte("not a valid ciphertext"))

	if _, err := decodeCard(v, pb); err == nil {
		t.Fatal("decodeCard: expected error for corrupt data, got nil")
	}
}

func TestMaskNumber(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"4111111111111111", "•••• 1111"},
		{"1234", "•••• 1234"},
		{"99", "99"},
		{"", ""},
	}
	for _, c := range cases {
		if got := maskNumber(c.in); got != c.want {
			t.Errorf("maskNumber(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
