### Fixed

- Derive the late-block attestation count time from `PROPOSER_REORG_CUTOFF_BPS` instead of a hardcoded 2s, so the late-slot `UpdateHead` tick and its clock disparity scale with the slot duration.
