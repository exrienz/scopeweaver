package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// legacyStockSchemaHashes lists entire InputSchema SHA-256 digests (over
// encoding/json.Marshal of the schema map — compact, key-sorted) of prior-stock
// factory schemas whose persisted rows would otherwise pin a known structural
// defect forever: wireTools seeds tool rows first-insert-only, and ToolResolve
// overlays the stored schema onto the current factory tool, so fixing the factory
// alone never reaches existing installs.
//
// Recognition is exact: a row whose schema differs from prior stock by even one
// byte (page edits, custom defaults, added properties) never matches and is
// preserved untouched. Substitution happens only at runtime resolution; stored
// rows are never rewritten.
var legacyStockSchemaHashes = map[string][]string{
	"add_intent": {
		// Prior English stock: intents.items advertised as a bare {type:object}
		// with no per-item properties and no required summary.
		"eb9a45a0576028f7fde38c6b49f1a952ac6a37dfef8f2ed15d244759e938032d",
		// Prior Korean stock, same structural omission.
		"5c70e9aee8f4c0e48037dfcaf37d118eeafb1f90ef678070b9134ad70c32ecc8",
		// Prior zh-Hans stock, same structural omission.
		"4f1e846d98a2d864aaba60343c43646724370f4a4373ec126cce39315bbb00c0",
		// Prior zh-Hant, Japanese and Spanish stock from dev 7af06f1,
		// captured from the factory with the same structural omission.
		"862675cbf388df6d6dbd00afa3ebb1d7ccd446da18b605094c5d9b2bc9323c8a",
		"6375a464712bdb225d5b9c114047dcbb62ddfe5b34792b29fa113d27a1cad6ab",
		"77e347d710add637c0e3a5401ff3f5a1f0c987d5baee46918c13d67ab44273b9",
	},
}

// ResolveLegacyStockSchema returns a copy of current when persisted exactly
// matches a registered prior-stock schema for name; otherwise it returns persisted
// unchanged. Neither input is mutated. Callers pass the persisted DB schema and
// the current factory schema for the same tool in the requested language, so a
// recognized stock row resolves to the corrected stock schema in the right
// language — exactly what a fresh install would advertise.
func ResolveLegacyStockSchema(name string, persisted, current map[string]any) map[string]any {
	if persisted == nil || current == nil {
		return persisted
	}
	known := legacyStockSchemaHashes[name]
	if len(known) == 0 {
		return persisted
	}
	encoded, err := json.Marshal(persisted)
	if err != nil {
		return persisted
	}
	sum := sha256.Sum256(encoded)
	digest := hex.EncodeToString(sum[:])
	for _, want := range known {
		if digest == want {
			return cloneToolSchema(current)
		}
	}
	return persisted
}
