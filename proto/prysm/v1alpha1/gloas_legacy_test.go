package eth_test

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	eth "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func legacyTestEnvelope(t *testing.T) *eth.SignedExecutionPayloadEnvelope {
	t.Helper()
	txs, err := enginev1.NewProgressiveTransactionList([][]byte{{0x01, 0x02}, {}, {0x03}})
	require.NoError(t, err)
	return &eth.SignedExecutionPayloadEnvelope{
		Message: &eth.ExecutionPayloadEnvelope{
			Payload: &enginev1.ExecutionPayloadGloas{
				ParentHash:      make([]byte, 32),
				FeeRecipient:    make([]byte, 20),
				StateRoot:       make([]byte, 32),
				ReceiptsRoot:    make([]byte, 32),
				LogsBloom:       make([]byte, 256),
				PrevRandao:      make([]byte, 32),
				BlockNumber:     7,
				GasLimit:        30_000_000,
				GasUsed:         21_000,
				Timestamp:       1_000,
				ExtraData:       []byte("extra"),
				BaseFeePerGas:   make([]byte, 32),
				BlockHash:       make([]byte, 32),
				Transactions:    txs,
				Withdrawals:     []*enginev1.Withdrawal{{Index: 1, ValidatorIndex: 2, Address: make([]byte, 20), Amount: 3}},
				BlobGasUsed:     131_072,
				ExcessBlobGas:   262_144,
				BlockAccessList: []byte{0xaa, 0xbb},
				SlotNumber:      primitives.Slot(11),
			},
			ExecutionRequests:     &enginev1.ExecutionRequestsGloas{},
			BuilderIndex:          primitives.BuilderIndex(5),
			BeaconBlockRoot:       make([]byte, 32),
			ParentBeaconBlockRoot: make([]byte, 32),
		},
		Signature: make([]byte, 96),
	}
}

func TestExecutionPayloadGloasLegacy_RoundTrip(t *testing.T) {
	signed := legacyTestEnvelope(t)
	payload := signed.Message.Payload

	legacy := eth.ExecutionPayloadGloasToLegacy(payload)
	require.DeepEqual(t, [][]byte{{0x01, 0x02}, {}, {0x03}}, legacy.Transactions, "legacy form carries the raw transactions under the old field")
	require.Equal(t, payload.SlotNumber, legacy.SlotNumber)

	back, err := eth.ExecutionPayloadGloasFromLegacy(legacy)
	require.NoError(t, err)
	require.DeepEqual(t, payload, back)

	wantRoot, err := payload.HashTreeRoot()
	require.NoError(t, err)
	gotRoot, err := back.HashTreeRoot()
	require.NoError(t, err)
	require.Equal(t, wantRoot, gotRoot)

	require.IsNil(t, eth.ExecutionPayloadGloasToLegacy(nil))
	fromNil, err := eth.ExecutionPayloadGloasFromLegacy(nil)
	require.NoError(t, err)
	require.IsNil(t, fromNil)
}

func TestSignedExecutionPayloadEnvelopeLegacy_RoundTrip(t *testing.T) {
	signed := legacyTestEnvelope(t)

	legacy := eth.SignedExecutionPayloadEnvelopeToLegacy(signed)
	back, err := eth.SignedExecutionPayloadEnvelopeFromLegacy(legacy)
	require.NoError(t, err)
	require.DeepEqual(t, signed, back)

	wantRoot, err := signed.Message.HashTreeRoot()
	require.NoError(t, err)
	gotRoot, err := back.Message.HashTreeRoot()
	require.NoError(t, err)
	require.Equal(t, wantRoot, gotRoot, "the signing root must survive translation")
}

func TestGenericSignedExecutionPayloadEnvelopeLegacy_RoundTrip(t *testing.T) {
	signed := legacyTestEnvelope(t)

	t.Run("contents arm", func(t *testing.T) {
		generic := &eth.GenericSignedExecutionPayloadEnvelope{
			Envelope: &eth.GenericSignedExecutionPayloadEnvelope_Contents{
				Contents: &eth.SignedExecutionPayloadEnvelopeContents{
					SignedExecutionPayloadEnvelope: signed,
					KzgProofs:                      [][]byte{make([]byte, 48)},
					Blobs:                          [][]byte{{0x01}},
				},
			},
		}
		legacy := eth.GenericSignedExecutionPayloadEnvelopeToLegacy(generic)
		require.NotNil(t, legacy.GetContents())
		back, err := eth.GenericSignedExecutionPayloadEnvelopeFromLegacy(legacy)
		require.NoError(t, err)
		require.DeepEqual(t, generic, back)
	})

	t.Run("signed_envelope arm", func(t *testing.T) {
		generic := &eth.GenericSignedExecutionPayloadEnvelope{
			Envelope: &eth.GenericSignedExecutionPayloadEnvelope_SignedEnvelope{SignedEnvelope: signed},
		}
		legacy := eth.GenericSignedExecutionPayloadEnvelopeToLegacy(generic)
		require.NotNil(t, legacy.GetSignedEnvelope())
		back, err := eth.GenericSignedExecutionPayloadEnvelopeFromLegacy(legacy)
		require.NoError(t, err)
		require.DeepEqual(t, generic, back)
	})

	t.Run("unset arm stays unset", func(t *testing.T) {
		back, err := eth.GenericSignedExecutionPayloadEnvelopeFromLegacy(&eth.GenericSignedExecutionPayloadEnvelopeLegacy{})
		require.NoError(t, err)
		require.IsNil(t, back.Envelope)
	})

	t.Run("nil", func(t *testing.T) {
		require.IsNil(t, eth.GenericSignedExecutionPayloadEnvelopeToLegacy(nil))
		back, err := eth.GenericSignedExecutionPayloadEnvelopeFromLegacy(nil)
		require.NoError(t, err)
		require.IsNil(t, back)
	})
}
