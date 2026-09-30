### Fixed

- Derive the late-block attestation count time and `ProcessAttestationsThreshold` from `PROPOSER_REORG_CUTOFF_BPS` instead of hardcoded 2s and 10s values, so the late-slot `UpdateHead` tick, its clock disparity, and the `ShouldOverrideFCU` parent-weight check all scale with the slot duration.
- Disable the late-slot head update tick instead of panicking at startup when `PROPOSER_REORG_CUTOFF_BPS` is configured outside `(0, 10000)`.
