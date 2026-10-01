### Fixed

- Derive the late-block attestation count time and `ProcessAttestationsThreshold` from `PROPOSER_REORG_CUTOFF_BPS` instead of hardcoded 2s and 10s values, so the late-slot `UpdateHead` tick, its clock disparity, and the `ShouldOverrideFCU` parent-weight check all scale with the slot duration.
- Disable the late-slot head update and proposer reorg decisions when `PROPOSER_REORG_CUTOFF_BPS` cannot produce a cutoff within the slot.
