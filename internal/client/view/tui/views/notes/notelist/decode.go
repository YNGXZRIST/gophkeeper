package notelist

import (
	"encoding/json"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
	"strings"
)

func decodeNote(v *vault.Vault, row repository.NoteRow) (clientmodel.Note, error) {
	raw, err := v.Decrypt(row.Data)
	if err != nil {
		return clientmodel.Note{}, err
	}
	var data clientmodel.NoteData
	if err := json.Unmarshal(raw, &data); err != nil {
		return clientmodel.Note{}, err
	}
	return clientmodel.Note{ID: row.ID, Data: data, Version: row.Version}, nil
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
