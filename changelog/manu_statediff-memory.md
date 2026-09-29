### Changed

- State diff: keep the most recent anchor as a live state instead of a compressed encoding, so saving a finalized state diff no longer decodes a full beacon state every 32 slots.
