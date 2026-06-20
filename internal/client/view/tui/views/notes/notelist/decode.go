package notelist

import (
	"encoding/json"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	notev1 "gophkeeper/internal/shared/proto/note/v1"
	"strings"
)

func decodeNote(v *vault.Vault, pb *notev1.Note) (clientmodel.Note, error) {
	raw, err := v.Decrypt(pb.GetData())
	if err != nil {
		return clientmodel.Note{}, err
	}
	var data clientmodel.NoteData
	if err := json.Unmarshal(raw, &data); err != nil {
		return clientmodel.Note{}, err
	}
	return clientmodel.Note{
		ID:        pb.GetId(),
		Data:      data,
		Version:   pb.GetVersion(),
		CreatedAt: pb.GetCreatedAt().AsTime(),
		UpdatedAt: pb.GetUpdatedAt().AsTime(),
	}, nil
}

func snippet(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if s == "" {
		return "—"
	}
	r := []rune(s)
	if len(r) > colText-1 {
		return string(r[:colText-2]) + "…"
	}
	return s
}
