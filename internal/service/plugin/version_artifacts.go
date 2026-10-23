package plugin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Mininglamp-OSS/octo-marketplace/internal/model"
	"github.com/Mininglamp-OSS/octo-marketplace/internal/storage"
)

// freezeVersionArtifacts detaches saved bytes from mutable caller upload keys.
// It completes before the DB transaction. Version objects are never reclaimed
// on failure: another concurrent save may already reference the same content.
// Unreferenced objects can be audited later; this PR introduces no physical GC.
func (s *Service) freezeVersionArtifacts(ctx context.Context, typ model.PluginType, docs *CanonicalDocuments, spaceID string) error {
	if typ == model.PluginTypeConnector {
		return nil
	}
	keys := attachmentKeyMap(docs.AttachmentKeys)
	if len(keys) == 0 {
		return nil
	}
	if s.storage == nil {
		return ErrInvalidRequest
	}
	writer, ok := s.storage.(storage.ImmutableWriter)
	if !ok {
		return fmt.Errorf("storage does not support immutable artifacts")
	}
	var total int64
	for _, a := range decodePackageAttachments(docs.Package) {
		if a.ContentType != "storage" {
			continue
		}
		key := keys[a.Path]
		if !safeObjectSegment.MatchString(spaceID) || !validReferencedObjectKey(key, spaceID) {
			return ErrInvalidRequest
		}
		data, err := s.readArtifactBytes(ctx, key, s.maxAttachmentBytes)
		if err != nil {
			return err
		}
		total += int64(len(data))
		if total > s.maxArchiveBytes {
			return ErrTooLarge
		}
		sum := sha256.Sum256(data)
		digest := "sha256:" + hex.EncodeToString(sum[:])
		if a.ContentSize != int64(len(data)) || a.ContentHash != digest {
			return ErrIntegrity
		}
		frozen := approvedAttachmentPrefix(spaceID) + "versions/" + hex.EncodeToString(sum[:])
		if key != frozen {
			if _, err = writer.PutImmutable(ctx, frozen, bytes.NewReader(data), int64(len(data)), "application/octet-stream"); err != nil {
				return err
			}
			// Verify even when the conditional upload lost a concurrent-create race.
			existing, err := s.readArtifactBytes(ctx, frozen, s.maxAttachmentBytes)
			if err != nil {
				return err
			}
			if !bytes.Equal(existing, data) {
				return ErrIntegrity
			}
		}
		keys[a.Path] = frozen
	}
	raw, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	docs.AttachmentKeys = raw
	return nil
}

func (s *Service) readArtifactBytes(ctx context.Context, key string, limit int64) ([]byte, error) {
	body, err := s.storage.GetObject(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read version artifact: %w", err)
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrTooLarge
	}
	return data, nil
}
