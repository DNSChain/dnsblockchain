package types

import (
	"strings"
)

// ReservedTLDs is a list of TLDs that are reserved by ICANN and cannot be registered.
// This list is not exhaustive and should be updated regularly.
// For a more comprehensive list, refer to ICANN's official TLD lists.
var ReservedTLDs = map[string]struct{}{
	// Common gTLDs
	"com":    {},
	"org":    {},
	"net":    {},
	"gov":    {},
	"edu":    {},
	"mil":    {},
	"int":    {},
	"info":   {},
	"biz":    {},
	"name":   {},
	"pro":    {},
	"aero":   {},
	"coop":   {},
	"mobi":   {},
	"cat":    {},
	"jobs":   {},
	"tel":    {},
	"asia":   {},
	"xyz":    {},
	"club":   {},
	"site":   {},
	"online": {},
	"tech":   {},
	"store":  {},
	"app":    {},
	"io":     {}, // Often used as gTLD, but is ccTLD for British Indian Ocean Territory
	"co":     {}, // Often used as gTLD, but is ccTLD for Colombia

	// Common ccTLDs (examples)
	"us": {}, // United States
	"uk": {}, // United Kingdom
	"de": {}, // Germany
	"jp": {}, // Japan
	"cn": {}, // China
	"in": {}, // India
	"br": {}, // Brazil
	"ca": {}, // Canada
	"au": {}, // Australia
	"fr": {}, // France
	"ru": {}, // Russia
	"it": {}, // Italy
	"es": {}, // Spain
	"mx": {}, // Mexico
	"ch": {}, // Switzerland
	"nl": {}, // Netherlands
	"se": {}, // Sweden
	"no": {}, // Norway
	"fi": {}, // Finland
	"dk": {}, // Denmark
	"pl": {}, // Poland
	"ar": {}, // Argentina
	"at": {}, // Austria
	"be": {}, // Belgium
	"kr": {}, // South Korea
	"sg": {}, // Singapore
	"za": {}, // South Africa

	// New gTLDs (examples)
	"academy":       {},
	"agency":        {},
	"accountant":    {},
	"ai":            {}, // ccTLD for Anguilla, often used for AI
	"bitcoin":       {},
	"blockchain":    {},
	"camera":        {},
	"coffee":        {},
	"company":       {},
	"computer":      {},
	"construction":  {},
	"contact":       {},
	"cool":          {},
	"credit":        {},
	"cruise":        {},
	"dance":         {},
	"data":          {},
	"design":        {},
	"diamond":       {},
	"digital":       {},
	"direct":        {},
	"directory":     {},
	"discount":      {},
	"doctor":        {},
	"dog":           {},
	"domains":       {},
	"email":         {},
	"energy":        {},
	"engineer":      {},
	"enterprises":   {},
	"equipment":     {},
	"estate":        {},
	"events":        {},
	"expert":        {},
	"express":       {},
	"farm":          {},
	"fashion":       {},
	"finance":       {},
	"fitness":       {},
	"flights":       {},
	"flowers":       {},
	"foundation":    {},
	"fund":          {},
	"furniture":     {},
	"futbol":        {},
	"gallery":       {},
	"gifts":         {},
	"glass":         {},
	"global":        {},
	"graphics":      {},
	"gratis":        {},
	"green":         {},
	"gripe":         {},
	"guide":         {},
	"guru":          {},
	"health":        {},
	"hockey":        {},
	"holdings":      {},
	"holiday":       {},
	"home":          {},
	"hospital":      {},
	"hotel":         {},
	"house":         {},
	"immobilien":    {},
	"industries":    {},
	"institute":     {},
	"insure":        {},
	"international": {},
	"investments":   {},
	"jewelry":       {},
	"kitchen":       {},
	"land":          {},
	"lawyer":        {},
	"lease":         {},
	"legal":         {},
	"life":          {},
	"lighting":      {},
	"limited":       {},
	"limo":          {},
	"link":          {},
	"living":        {},
	"loans":         {},
	"london":        {},
	"ltd":           {},
	"luxury":        {},
	"management":    {},
	"marketing":     {},
	"media":         {},
	"medical":       {},
	"men":           {},
	"money":         {},
	"movie":         {},
	"music":         {},
	"network":       {},
	"news":          {},
	"ninja":         {},
	"partners":      {},

	// Special-Use Domain Names (RFC 6761)
	"example":   {},
	"invalid":   {},
	"local":     {}, // Note: .local is used by mDNS, so definitely should be reserved.
	"localhost": {}, // Note: localhost is for loopback.
	"test":      {},

	// Infrastructure TLD
	"arpa": {},
}

// IsReservedTLD checks if a TLD is in the hardcoded deny list.
// The TLD should be normalized (lowercase, no leading/trailing spaces or dots) before calling this.
func IsReservedTLD(tld string) bool {
	normalizedTLD := strings.ToLower(strings.Trim(tld, "."))
	_, exists := ReservedTLDs[normalizedTLD]
	return exists
}
