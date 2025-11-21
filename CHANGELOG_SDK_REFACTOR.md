# LCC-SDK Refactoring Changelog

## Date: 2025-11-21

## Summary

Implemented Phase 1 of the authorization model refactoring to separate feature registration (YAML) from authorization control (License).

## Changes Made

### 1. Configuration Schema (`pkg/config/types.go`)

#### Deprecated Fields
- `FeatureConfig.Tier` - Marked as deprecated with clear documentation
- `FeatureConfig.Quota` - Marked as deprecated, quotas now defined in License

#### New Fields
- `FeatureConfig.Category` - Optional metadata for feature organization
- `FeatureConfig.Tags` - Optional tags for feature categorization

#### Documentation
- Added detailed comments explaining that YAML is for technical mapping
- Clarified that authorization control is in License file, not YAML
- Made tier and quota fields optional (`omitempty`)

**Impact**: Backward compatible - old YAML files with tier/quota still work

### 2. Client Documentation (`pkg/client/client.go`)

#### Updated Comments
- `Client` struct: Added explanation of old vs new authorization model
- `CheckFeature()`: Detailed documentation explaining License-based control
- Clarified that authorization is determined by License, not YAML

**Impact**: No code changes, only documentation improvements

### 3. README Updates (`README.md`)

#### Changes
- Updated example YAML to remove `tier` field
- Added note about not specifying tier/quota in YAML
- Added "Authorization Model" section explaining:
  - New model (License controls features)
  - Old model (tier-based, deprecated)
  - Benefits of new approach

**Impact**: Users understand the new recommended approach

### 4. New Documentation

#### `MIGRATION_GUIDE.md`
- Step-by-step migration instructions
- Before/after examples
- Backward compatibility explanation
- Troubleshooting section
- 317 lines of comprehensive guidance

#### `REFACTOR_PLAN.md` (already existed)
- Complete refactoring plan for all components
- Timeline estimates
- Testing strategy
- Phase-based implementation approach

## Backward Compatibility

✅ **Fully Backward Compatible**

1. Old YAML files with `tier` field → Field is ignored, no errors
2. Old License format → LCC server still handles tier-based checks
3. Mixed usage → Works with any combination of old/new formats

## Testing

- ✅ All existing config tests pass
- ✅ No breaking changes to API
- ✅ Deprecated fields still accepted

## What's Next (Future Phases)

### Phase 2: LCC Server Changes
- Update `models/sdk.go` to add `FeaturePermission` structure
- Modify `CheckFeatureAuthorized` → `CheckFeatureEnabled`
- Support new License format in storage layer

### Phase 3: LMF Updates
- Generate new License format with detailed feature permissions
- Include per-feature quotas and limits

### Phase 4: Deprecation
- Remove old tier-based logic (after transition period)
- Clean up deprecated fields

## Migration Path for Users

Users can migrate gradually:

1. **Now**: Update YAML files (remove tier/quota)
   - Works with old licenses
   - No risk

2. **Later**: Update LCC server
   - Generates new license format
   - Still supports old format

3. **Eventually**: Full migration
   - All licenses in new format
   - Old tier logic removed

## Key Benefits

1. **Clear Separation of Concerns**
   - YAML: Technical (feature → function mapping)
   - License: Business (enabled/disabled, limits)

2. **Flexible Authorization**
   - Enable/disable features per customer
   - Custom quotas per customer
   - No code changes needed

3. **Stable Interface**
   - Feature IDs as business interface
   - Function implementations can change
   - License independent of code

4. **Better Product Tiers**
   - Tiers are just license templates
   - Easy custom configurations
   - No hardcoded tier hierarchy

## Files Modified

```
lcc-sdk/
├── pkg/
│   ├── config/
│   │   └── types.go              # Deprecated tier/quota, added metadata
│   └── client/
│       └── client.go              # Updated documentation
├── README.md                      # Updated examples and added Authorization Model section
├── MIGRATION_GUIDE.md             # NEW: Complete migration guide
├── REFACTOR_PLAN.md               # Created earlier: Complete refactoring plan
└── CHANGELOG_SDK_REFACTOR.md      # NEW: This file
```

## Breaking Changes

**None** - This is a fully backward-compatible change.

## Notes

- The `tier` field in YAML is now optional and ignored by new license logic
- Old behavior is preserved for backward compatibility
- Users are encouraged to migrate but not required
- LCC server changes (Phase 2) will fully implement new authorization model

## Testing Checklist

- [x] Config tests pass
- [x] No compilation errors
- [x] Backward compatibility maintained
- [x] Documentation updated
- [ ] Integration tests with LCC server (requires Phase 2)
- [ ] End-to-end tests with demo app (requires Phase 2)

## Support

For questions or issues:
1. See MIGRATION_GUIDE.md for migration help
2. See REFACTOR_PLAN.md for technical details
3. Contact team for assistance
