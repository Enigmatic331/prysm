### Ignore
- enforce ssz limits that have migrated to the stf at unmarshal time.
- skip spectest vectors whose only defect is an over-limit list, since that limit now rejects them at unmarshal time.
