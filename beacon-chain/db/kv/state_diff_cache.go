package kv

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/cmd/beacon-chain/flags"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/golang/snappy"
	pkgerrors "github.com/pkg/errors"
	"go.etcd.io/bbolt"
)

// anchor holds a cached state diff anchor, either as a live state (st) or as a snappy-compressed SSZ encoding (data).
// Only the most recently set anchor is live: every finer-level save diffs against it, so keeping it live spares a
// full state decode on each save. Older anchors are kept compressed, since they are only needed again at coarser
// boundaries.
type anchor struct {
	slot primitives.Slot
	st   state.BeaconState
	data []byte
}
type stateDiffCache struct {
	sync.RWMutex
	anchors          []anchor
	levelsWithData   []bool
	offset           uint64
	anchorGeneration uint64
}

func populateStateDiffCacheFromDB(s *Store, offset uint64) (*stateDiffCache, error) {
	cache := &stateDiffCache{
		anchors:        make([]anchor, len(flags.Get().StateDiffExponents)-1),
		levelsWithData: make([]bool, len(flags.Get().StateDiffExponents)),
		offset:         offset,
	}

	if err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(stateDiffBucket)
		if bucket == nil {
			return bbolt.ErrBucketNotFound
		}
		for level := range cache.levelsWithData {
			if level == 0 {
				if bucket.Get(makeKeyForStateDiffTree(0, offset)) != nil {
					cache.levelsWithData[level] = true
				}
				continue
			}
			cursor := bucket.Cursor()
			prefix := []byte{byte(level)}
			key, _ := cursor.Seek(prefix)
			if key != nil && key[0] == byte(level) {
				slot, ok := slotFromStateDiffKey(key)
				if !ok {
					return ErrStateDiffCorrupted
				}
				if slot < offset {
					return ErrStateDiffCorrupted
				}
				if computeLevel(offset, primitives.Slot(slot)) != level {
					return ErrStateDiffCorrupted
				}
				if !hasCompleteDiffAtLevelSlot(bucket, level, slot) {
					return ErrStateDiffCorrupted
				}
				cache.levelsWithData[level] = true
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	anchor0, err := s.getFullSnapshot(offset)
	if err != nil {
		if errors.Is(err, errSnapshotNotFound) {
			return nil, pkgerrors.Wrapf(ErrStateDiffMissingSnapshot, "offset snapshot at slot %d", offset)
		}
		return nil, pkgerrors.Wrapf(ErrStateDiffCorrupted, "failed to load offset snapshot at slot %d: %v", offset, err)
	}
	// Only cache anchor if there are higher levels that need it.
	// With a single exponent, len(anchors)==0 and no caching is needed.
	if len(cache.anchors) > 0 {
		err := cache.setAnchor(0, anchor0)
		if err != nil {
			return nil, err
		}
	}
	cache.levelsWithData[0] = true

	return cache, nil
}

func validateStateDiffCache(ctx context.Context, s *Store, cache *stateDiffCache) error {
	// Copy level flags under lock, then release before validation work.
	// stateByDiff may consult cache metadata and should never be called while holding cache locks.
	cache.RLock()
	levels := make([]bool, len(cache.levelsWithData))
	copy(levels, cache.levelsWithData)
	cache.RUnlock()

	for level, hasData := range levels {
		if !hasData || level == 0 {
			continue
		}
		maxSlot, err := latestSlotForLevel(s, level)
		if err != nil {
			return err
		}
		if _, err := s.stateByDiff(ctx, primitives.Slot(maxSlot)); err != nil {
			return pkgerrors.Wrapf(ErrStateDiffCorrupted, "state diff validation failed for level %d slot %d: %v", level, maxSlot, err)
		}
	}
	return nil
}

func latestSlotForLevel(s *Store, level int) (uint64, error) {
	var maxSlot uint64
	found := false
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(stateDiffBucket)
		if bucket == nil {
			return bbolt.ErrBucketNotFound
		}
		cursor := bucket.Cursor()
		prefix := []byte{byte(level)}
		for key, _ := cursor.Seek(prefix); key != nil && key[0] == byte(level); key, _ = cursor.Next() {
			slot, ok := slotFromStateDiffKey(key)
			if !ok {
				return ErrStateDiffCorrupted
			}
			if !found || slot > maxSlot {
				maxSlot = slot
				found = true
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, ErrStateDiffCorrupted
	}
	return maxSlot, nil
}

func slotFromStateDiffKey(key []byte) (uint64, bool) {
	if len(key) < 9 {
		return 0, false
	}
	return binary.LittleEndian.Uint64(key[1:9]), true
}

func hasCompleteDiffAtLevelSlot(bucket *bbolt.Bucket, level int, slot uint64) bool {
	key := makeKeyForStateDiffTree(level, slot)
	stateKey := append(append([]byte{}, key...), stateSuffix...)
	validatorKey := append(append([]byte{}, key...), validatorSuffix...)
	balancesKey := append(append([]byte{}, key...), balancesSuffix...)
	return bucket.Get(stateKey) != nil && bucket.Get(validatorKey) != nil && bucket.Get(balancesKey) != nil
}

func newStateDiffCache(s *Store) (*stateDiffCache, error) {
	var offset uint64

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(stateDiffBucket)
		if bucket == nil {
			return bbolt.ErrBucketNotFound
		}

		offsetBytes := bucket.Get(offsetKey)
		if offsetBytes == nil {
			return errors.New("state diff cache: offset not found")
		}
		offset = binary.LittleEndian.Uint64(offsetBytes)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &stateDiffCache{
		anchors:        make([]anchor, len(flags.Get().StateDiffExponents)-1), // -1 because last level doesn't need to be cached
		levelsWithData: make([]bool, len(flags.Get().StateDiffExponents)),
		offset:         offset,
	}, nil
}

type getAnchorOpts struct {
	exactSlot *primitives.Slot
}

type optFunc func(*getAnchorOpts)

func withExactSlot(slot primitives.Slot) optFunc {
	return func(opts *getAnchorOpts) {
		opts.exactSlot = &slot
	}
}

func (c *stateDiffCache) getAnchor(level int, opts ...optFunc) state.BeaconState {
	cfg := getAnchorOpts{}

	for _, opt := range opts {
		opt(&cfg)
	}

	c.RLock()
	if level < 0 || level >= len(c.anchors) {
		c.RUnlock()
		return nil
	}
	cachedAnchor := c.anchors[level]
	c.RUnlock()

	if cfg.exactSlot != nil && *cfg.exactSlot != cachedAnchor.slot {
		return nil
	}

	if cachedAnchor.st != nil {
		return cachedAnchor.st.Copy()
	}

	if len(cachedAnchor.data) == 0 {
		return nil
	}

	uncompressed, err := snappy.Decode(nil, cachedAnchor.data)
	if err != nil {
		return nil
	}

	st, err := decodeStateSnapshot(uncompressed)
	if err != nil {
		return nil
	}

	return st
}

// setAnchor caches anchorState as the live anchor of the given level.
// The previous live anchor is dropped if it is at the same or a finer level, since no later save diffs against it.
// Otherwise it is still needed at the next coarser boundary, so it is compressed.
func (c *stateDiffCache) setAnchor(level int, anchorState state.ReadOnlyBeaconState) error {
	c.RLock()
	if level < 0 || level >= len(c.anchors) {
		c.RUnlock()
		return errors.New("state diff cache: anchor level out of range")
	}
	generation := c.anchorGeneration
	prevLevel, prev := c.liveAnchorLocked()
	c.RUnlock()

	if anchorState == nil {
		return errors.New("state diff cache: anchor cannot be nil")
	}

	var demoted []byte
	if prev.st != nil && prevLevel < level {
		encoded, err := encodeStateWithKey(prev.st)
		if err != nil {
			return fmt.Errorf("encode state with key: %w", err)
		}
		// snappy.Encode over-allocates, trim the capacity since the anchor is kept around.
		demoted = make([]byte, len(encoded))
		copy(demoted, encoded)
	}

	live := anchorState.Copy()

	c.Lock()
	defer c.Unlock()

	if generation != c.anchorGeneration {
		return nil
	}
	if prev.st != nil && c.anchors[prevLevel].st == prev.st {
		c.anchors[prevLevel] = anchor{}
		stateDiffAnchorCacheBytes.WithLabelValues(strconv.Itoa(prevLevel)).Set(0)
		if demoted != nil {
			c.anchors[prevLevel] = anchor{slot: prev.slot, data: demoted}
			stateDiffAnchorCacheBytes.WithLabelValues(strconv.Itoa(prevLevel)).Set(float64(len(demoted)))
		}
	}
	c.anchors[level] = anchor{slot: anchorState.Slot(), st: live}
	stateDiffAnchorCacheBytes.WithLabelValues(strconv.Itoa(level)).Set(0)
	return nil
}

// liveAnchorLocked returns the live anchor and its level, or -1 and an empty anchor if there is none.
// The caller must hold the lock.
func (c *stateDiffCache) liveAnchorLocked() (int, anchor) {
	for level, a := range c.anchors {
		if a.st != nil {
			return level, a
		}
	}
	return -1, anchor{}
}

// reanchor points the cache at a new offset and drops the cached anchors
func (c *stateDiffCache) reanchor(offset uint64, levelsWithData []bool) {
	c.Lock()
	defer c.Unlock()

	c.offset = offset
	c.levelsWithData = levelsWithData

	c.clearAnchorsLocked()
}

func (c *stateDiffCache) levelHasData(level int) bool {
	c.RLock()
	defer c.RUnlock()
	if level < 0 || level >= len(c.levelsWithData) {
		return false
	}
	return c.levelsWithData[level]
}

func (c *stateDiffCache) setLevelHasData(level int) error {
	c.Lock()
	defer c.Unlock()
	if level < 0 || level >= len(c.levelsWithData) {
		return errors.New("state diff cache: level data index out of range")
	}
	c.levelsWithData[level] = true
	return nil
}

func (c *stateDiffCache) getOffset() uint64 {
	c.RLock()
	defer c.RUnlock()
	return c.offset
}

func (c *stateDiffCache) setOffset(offset uint64) {
	c.Lock()
	defer c.Unlock()
	c.offset = offset
}

func (c *stateDiffCache) clearAnchors() {
	c.Lock()
	defer c.Unlock()
	c.clearAnchorsLocked()
}

// clearAnchorsLocked is clearAnchors, for the callers that already hold the lock.
func (c *stateDiffCache) clearAnchorsLocked() {
	c.anchorGeneration++
	c.anchors = make([]anchor, len(flags.Get().StateDiffExponents)-1) // -1 because last level doesn't need to be cached
	for level := range len(c.anchors) {
		stateDiffAnchorCacheBytes.WithLabelValues(strconv.Itoa(level)).Set(0)
	}
}
