package sop

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
)

// Regex to capture markdown image links like ![alt](assets/image.png)
var assetRefRegex = regexp.MustCompile(`\((assets\/[^)]+)\)`)

func (s *Service) ValidateSOPContentAssets(
	sopID string,
	content string,
) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.validateSOPContentAssets(sopID, content)
}

func (s *Service) validateSOPContentAssets(
	sopID string,
	content string,
) error {
	// 1. Find all asset references in the markdown
	// Regex matches ![Alt](path) -> m[1] is the path
	matches := assetRefRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	// 2. Fetch ALL assets for this SOP
	existingAssets, err := getSOPAssetsBySOPIDRecord(s.db, sopID)
	if err != nil {
		return fmt.Errorf("failed to validate assets: %w", err)
	}

	// 3. Create a lookup map of valid paths
	// CRITICAL: Normalize DB paths to forward slashes (sops/id/assets/img.png)
	validPaths := make(map[string]struct{})
	for _, asset := range existingAssets {
		// filepath.ToSlash converts 'sops\id\assets\img.png' -> 'sops/id/assets/img.png'
		normalizedPath := filepath.ToSlash(asset.ContentPath)
		validPaths[normalizedPath] = struct{}{}
	}

	// 4. Validate every reference found in content
	for _, m := range matches {
		rawRef := m[1] // e.g. "assets/Screen%20Shot.png"

		decodedRef, err := url.QueryUnescape(rawRef)
		if err != nil {
			return fmt.Errorf("invalid asset URL encoding: %s", rawRef)
		}

		fullRelPath := filepath.Join("sops", sopID, decodedRef)

		lookupPath := filepath.ToSlash(fullRelPath)

		if _, exists := validPaths[lookupPath]; !exists {
			return fmt.Errorf("referenced asset not found: %s", decodedRef)
		}
	}

	return nil
}
