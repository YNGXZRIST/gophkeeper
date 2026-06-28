package filelist

import (
	"encoding/json"
	"fmt"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/repository"
	"gophkeeper/internal/client/vault"
)

func decodeFile(v *vault.Vault, row repository.FileRow) (clientmodel.File, error) {
	raw, err := v.Decrypt(row.Meta)
	if err != nil {
		return clientmodel.File{}, err
	}
	var meta clientmodel.FileMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return clientmodel.File{}, err
	}
	return clientmodel.File{
		ID:         row.ID,
		Meta:       meta,
		ChunkCount: row.ChunkCount,
		Version:    row.Version,
	}, nil
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
