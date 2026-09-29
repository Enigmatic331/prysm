### Fixed

- Gloas initial sync now verifies a parent execution payload envelope and checks that its data columns are available before importing it, the same checks applied to envelopes received over gossip. Previously the envelope was sent to the execution client without these checks.
- Gloas initial sync now requests a parent's payload envelope by root when the range response doesn't include it. Previously the child was imported without its parent's payload. Envelope range requests are also capped at `MAX_REQUEST_PAYLOADS`; fork recovery could previously request more.
- Gloas initial sync now saves a verified parent payload envelope as soon as it is imported. Previously, if a later block in the same batch was rejected, the envelope was never saved even though forkchoice already treated the parent's payload as present.
