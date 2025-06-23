package types

import (
	"strings"
)

// ExtractTLD extracts the Top-Level Domain (TLD) from a full domain name.
// It returns the TLD in lowercase and an empty string if no TLD is found
// (e.g., for "localhost" or a single label).
// Examples:
//
//	"example.com" -> "com"
//	"my.example.co.uk" -> "uk"
//	"mydomain" -> "" (o podrías decidir que esto es un error de formato de dominio)
//	".com" -> "com" (después de trim)
//	"example." -> "" (después de trim)
func ExtractTLD(domainName string) string {
	trimmedName := strings.Trim(domainName, ".")
	if trimmedName == "" {
		return ""
	}

	parts := strings.Split(trimmedName, ".")
	if len(parts) < 2 {
		// This means it's a single label (like "localhost") or an invalid format.
		// For our purpose, a single label doesn't have a TLD in the traditional sense
		// that would be in our PermittedTLDs list.
		return ""
	}
	// The TLD is the last part.
	return strings.ToLower(parts[len(parts)-1])
}
