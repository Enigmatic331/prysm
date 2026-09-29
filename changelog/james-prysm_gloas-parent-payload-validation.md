### Fixed

- Validate parent execution payload envelopes and their data availability during gloas initial sync.
- Fetch missing parent payload envelopes and keep payload requests within protocol limits.
- Persist newly imported parent payload envelopes even when a child block in the batch is rejected.
