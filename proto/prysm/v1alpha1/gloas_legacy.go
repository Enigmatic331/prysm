package eth

import (
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	"github.com/pkg/errors"
)

// This file translates between the Gloas execution payload envelope message
// chain and its legacy twin in gloas_legacy.proto, whose payload carries
// transactions as `repeated bytes` under protobuf tag 14. The legacy chain
// exists only so the V1 envelope gRPC endpoints can keep serving validator
// clients that predate the ProgressiveTransactionList encoding; nothing else
// should construct these types. Sub-messages whose layout did not change
// (withdrawals, execution requests) are shared by pointer rather than copied,
// since a translated value is consumed immediately by the RPC layer.

// ExecutionPayloadGloasToLegacy converts a payload to its legacy form. The
// transaction byte slices are read-only views into the source list.
func ExecutionPayloadGloasToLegacy(p *enginev1.ExecutionPayloadGloas) *ExecutionPayloadGloasLegacy {
	if p == nil {
		return nil
	}
	return &ExecutionPayloadGloasLegacy{
		ParentHash:      p.ParentHash,
		FeeRecipient:    p.FeeRecipient,
		StateRoot:       p.StateRoot,
		ReceiptsRoot:    p.ReceiptsRoot,
		LogsBloom:       p.LogsBloom,
		PrevRandao:      p.PrevRandao,
		BlockNumber:     p.BlockNumber,
		GasLimit:        p.GasLimit,
		GasUsed:         p.GasUsed,
		Timestamp:       p.Timestamp,
		ExtraData:       p.ExtraData,
		BaseFeePerGas:   p.BaseFeePerGas,
		BlockHash:       p.BlockHash,
		Transactions:    p.Transactions.Slice(),
		Withdrawals:     p.Withdrawals,
		BlobGasUsed:     p.BlobGasUsed,
		ExcessBlobGas:   p.ExcessBlobGas,
		BlockAccessList: p.BlockAccessList,
		SlotNumber:      p.SlotNumber,
	}
}

// ExecutionPayloadGloasFromLegacy converts a legacy payload to the current
// form, building the serialized transaction list. It fails if the
// transactions exceed the payload limits.
func ExecutionPayloadGloasFromLegacy(l *ExecutionPayloadGloasLegacy) (*enginev1.ExecutionPayloadGloas, error) {
	if l == nil {
		return nil, nil
	}
	txs, err := enginev1.NewProgressiveTransactionList(l.Transactions)
	if err != nil {
		return nil, errors.Wrap(err, "legacy payload transactions")
	}
	return &enginev1.ExecutionPayloadGloas{
		ParentHash:      l.ParentHash,
		FeeRecipient:    l.FeeRecipient,
		StateRoot:       l.StateRoot,
		ReceiptsRoot:    l.ReceiptsRoot,
		LogsBloom:       l.LogsBloom,
		PrevRandao:      l.PrevRandao,
		BlockNumber:     l.BlockNumber,
		GasLimit:        l.GasLimit,
		GasUsed:         l.GasUsed,
		Timestamp:       l.Timestamp,
		ExtraData:       l.ExtraData,
		BaseFeePerGas:   l.BaseFeePerGas,
		BlockHash:       l.BlockHash,
		Transactions:    txs,
		Withdrawals:     l.Withdrawals,
		BlobGasUsed:     l.BlobGasUsed,
		ExcessBlobGas:   l.ExcessBlobGas,
		BlockAccessList: l.BlockAccessList,
		SlotNumber:      l.SlotNumber,
	}, nil
}

// ExecutionPayloadEnvelopeToLegacy converts an envelope to its legacy form.
func ExecutionPayloadEnvelopeToLegacy(e *ExecutionPayloadEnvelope) *ExecutionPayloadEnvelopeLegacy {
	if e == nil {
		return nil
	}
	return &ExecutionPayloadEnvelopeLegacy{
		Payload:               ExecutionPayloadGloasToLegacy(e.Payload),
		ExecutionRequests:     e.ExecutionRequests,
		BuilderIndex:          e.BuilderIndex,
		BeaconBlockRoot:       e.BeaconBlockRoot,
		ParentBeaconBlockRoot: e.ParentBeaconBlockRoot,
	}
}

// ExecutionPayloadEnvelopeFromLegacy converts a legacy envelope to the current form.
func ExecutionPayloadEnvelopeFromLegacy(l *ExecutionPayloadEnvelopeLegacy) (*ExecutionPayloadEnvelope, error) {
	if l == nil {
		return nil, nil
	}
	payload, err := ExecutionPayloadGloasFromLegacy(l.Payload)
	if err != nil {
		return nil, err
	}
	return &ExecutionPayloadEnvelope{
		Payload:               payload,
		ExecutionRequests:     l.ExecutionRequests,
		BuilderIndex:          l.BuilderIndex,
		BeaconBlockRoot:       l.BeaconBlockRoot,
		ParentBeaconBlockRoot: l.ParentBeaconBlockRoot,
	}, nil
}

// SignedExecutionPayloadEnvelopeToLegacy converts a signed envelope to its legacy form.
func SignedExecutionPayloadEnvelopeToLegacy(s *SignedExecutionPayloadEnvelope) *SignedExecutionPayloadEnvelopeLegacy {
	if s == nil {
		return nil
	}
	return &SignedExecutionPayloadEnvelopeLegacy{
		Message:   ExecutionPayloadEnvelopeToLegacy(s.Message),
		Signature: s.Signature,
	}
}

// SignedExecutionPayloadEnvelopeFromLegacy converts a legacy signed envelope to the current form.
func SignedExecutionPayloadEnvelopeFromLegacy(l *SignedExecutionPayloadEnvelopeLegacy) (*SignedExecutionPayloadEnvelope, error) {
	if l == nil {
		return nil, nil
	}
	message, err := ExecutionPayloadEnvelopeFromLegacy(l.Message)
	if err != nil {
		return nil, err
	}
	return &SignedExecutionPayloadEnvelope{Message: message, Signature: l.Signature}, nil
}

// SignedExecutionPayloadEnvelopeContentsToLegacy converts envelope contents to the legacy form.
func SignedExecutionPayloadEnvelopeContentsToLegacy(c *SignedExecutionPayloadEnvelopeContents) *SignedExecutionPayloadEnvelopeContentsLegacy {
	if c == nil {
		return nil
	}
	return &SignedExecutionPayloadEnvelopeContentsLegacy{
		SignedExecutionPayloadEnvelope: SignedExecutionPayloadEnvelopeToLegacy(c.SignedExecutionPayloadEnvelope),
		KzgProofs:                      c.KzgProofs,
		Blobs:                          c.Blobs,
	}
}

// SignedExecutionPayloadEnvelopeContentsFromLegacy converts legacy envelope contents to the current form.
func SignedExecutionPayloadEnvelopeContentsFromLegacy(l *SignedExecutionPayloadEnvelopeContentsLegacy) (*SignedExecutionPayloadEnvelopeContents, error) {
	if l == nil {
		return nil, nil
	}
	signed, err := SignedExecutionPayloadEnvelopeFromLegacy(l.SignedExecutionPayloadEnvelope)
	if err != nil {
		return nil, err
	}
	return &SignedExecutionPayloadEnvelopeContents{
		SignedExecutionPayloadEnvelope: signed,
		KzgProofs:                      l.KzgProofs,
		Blobs:                          l.Blobs,
	}, nil
}

// GenericSignedExecutionPayloadEnvelopeToLegacy converts a generic signed
// envelope to its legacy form, preserving which oneof arm is set.
func GenericSignedExecutionPayloadEnvelopeToLegacy(g *GenericSignedExecutionPayloadEnvelope) *GenericSignedExecutionPayloadEnvelopeLegacy {
	if g == nil {
		return nil
	}
	out := &GenericSignedExecutionPayloadEnvelopeLegacy{}
	switch arm := g.Envelope.(type) {
	case *GenericSignedExecutionPayloadEnvelope_Contents:
		out.Envelope = &GenericSignedExecutionPayloadEnvelopeLegacy_Contents{
			Contents: SignedExecutionPayloadEnvelopeContentsToLegacy(arm.Contents),
		}
	case *GenericSignedExecutionPayloadEnvelope_SignedEnvelope:
		out.Envelope = &GenericSignedExecutionPayloadEnvelopeLegacy_SignedEnvelope{
			SignedEnvelope: SignedExecutionPayloadEnvelopeToLegacy(arm.SignedEnvelope),
		}
	}
	return out
}

// GenericSignedExecutionPayloadEnvelopeFromLegacy converts a legacy generic
// signed envelope to the current form, preserving which oneof arm is set. An
// unset arm is preserved as unset so the caller reports it as it would for a
// current request.
func GenericSignedExecutionPayloadEnvelopeFromLegacy(l *GenericSignedExecutionPayloadEnvelopeLegacy) (*GenericSignedExecutionPayloadEnvelope, error) {
	if l == nil {
		return nil, nil
	}
	out := &GenericSignedExecutionPayloadEnvelope{}
	switch arm := l.Envelope.(type) {
	case *GenericSignedExecutionPayloadEnvelopeLegacy_Contents:
		contents, err := SignedExecutionPayloadEnvelopeContentsFromLegacy(arm.Contents)
		if err != nil {
			return nil, err
		}
		out.Envelope = &GenericSignedExecutionPayloadEnvelope_Contents{Contents: contents}
	case *GenericSignedExecutionPayloadEnvelopeLegacy_SignedEnvelope:
		signed, err := SignedExecutionPayloadEnvelopeFromLegacy(arm.SignedEnvelope)
		if err != nil {
			return nil, err
		}
		out.Envelope = &GenericSignedExecutionPayloadEnvelope_SignedEnvelope{SignedEnvelope: signed}
	}
	return out, nil
}
