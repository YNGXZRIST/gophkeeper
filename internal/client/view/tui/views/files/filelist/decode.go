package filelist

import (
	"encoding/json"
	"fmt"
	clientmodel "gophkeeper/internal/client/model"
	"gophkeeper/internal/client/vault"
	filev1 "gophkeeper/internal/shared/proto/file/v1"
)

func decodeFile(v *vault.Vault, pb *filev1.File) (clientmodel.File, error) {
	raw, err := v.Decrypt(pb.GetMeta())
	if err != nil {
		return clientmodel.File{}, err
	}
	var meta clientmodel.FileMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return clientmodel.File{}, err
	}
	return clientmodel.File{
		ID:         pb.GetId(),
		Meta:       meta,
		ChunkCount: int(pb.GetChunkCount()),
		Version:    pb.GetVersion(),
		CreatedAt:  pb.GetCreatedAt().AsTime(),
		UpdatedAt:  pb.GetUpdatedAt().AsTime(),
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
