### Fixed

- Gloas initial sync now requests a parent's payload envelope by root when the range response doesn't include it. Previously sync retried earlier ranges until one included it.
