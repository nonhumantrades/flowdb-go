package proto

// Hand-written (non-generated) allocation-free decoder for StreamQueryChunk.
//
// The generated StreamQueryBatch.UnmarshalVT allocates a fresh *Row (plus its
// Data slice and Timestamp) for every row on the wire. For large streaming
// reads that is ~1 allocation per payload byte. ReusableStreamQueryChunk
// recycles the Row objects, their Data/Timestamp storage, the Rows slice and
// the chunk envelope across UnmarshalVT calls.
//
// Wire format handled here (must match flow.proto):
//
//	StreamQueryChunk: oneof { header = 1; batch = 2; footer = 3 }
//	StreamQueryBatch: uint32 index = 1; repeated Row rows = 2
//
// Row payloads are decoded with the generated Row.UnmarshalVT, so any change
// to Row is picked up automatically; only the two envelopes above are
// hand-rolled.

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// ReusableStreamQueryChunk decodes StreamQueryChunk messages while reusing
// row storage between calls. After a successful UnmarshalVT exactly one of
// Header, Batch or Footer is non-nil.
//
// Ownership: everything reachable from Header/Batch/Footer (including every
// *Row and its Data) is only valid until the next UnmarshalVT/RecvFrom call.
// Callers that need to retain a row must copy it (e.g. row.CloneVT()).
type ReusableStreamQueryChunk struct {
	Header *StreamQueryHeader
	Batch  *StreamQueryBatch
	Footer *StreamQueryFooter

	batch StreamQueryBatch
	pool  []*Row
}

// RecvFrom receives the next chunk from the stream into m, reusing storage
// from previous calls. Returns io.EOF at end of stream.
func (m *ReusableStreamQueryChunk) RecvFrom(s DRPCFlowDB_StreamQueryClient) error {
	return s.MsgRecv(m, drpcEncoding_File_flow_proto{})
}

// MarshalVT exists only to satisfy the vtprotobuf drpc codec interface;
// this type is receive-only.
func (m *ReusableStreamQueryChunk) MarshalVT() ([]byte, error) {
	return nil, fmt.Errorf("proto: ReusableStreamQueryChunk is receive-only")
}

func (m *ReusableStreamQueryChunk) UnmarshalVT(dAtA []byte) error {
	m.Header, m.Batch, m.Footer = nil, nil, nil

	for len(dAtA) > 0 {
		num, typ, n := protowire.ConsumeTag(dAtA)
		if n < 0 {
			return protowire.ParseError(n)
		}
		dAtA = dAtA[n:]

		if typ != protowire.BytesType {
			n = protowire.ConsumeFieldValue(num, typ, dAtA)
			if n < 0 {
				return protowire.ParseError(n)
			}
			dAtA = dAtA[n:]
			continue
		}

		v, n := protowire.ConsumeBytes(dAtA)
		if n < 0 {
			return protowire.ParseError(n)
		}
		dAtA = dAtA[n:]

		// Assumes at most one oneof member and at most one batch field per
		// wire message, which is what the flowdb server always emits. On
		// nonconforming input this diverges from the generated decoder:
		// generated oneof handling is last-field-wins and repeated batch
		// fields merge their rows, whereas here all members are surfaced
		// (caller dispatch prefers Header) and a repeated batch field
		// overwrites the previous one.
		switch num {
		case 1: // header: once per stream, a fresh alloc is fine
			h := &StreamQueryHeader{}
			if err := h.UnmarshalVT(v); err != nil {
				return err
			}
			m.Header = h
		case 2:
			if err := m.unmarshalBatch(v); err != nil {
				return err
			}
			m.Batch = &m.batch
		case 3: // footer: once per stream
			f := &StreamQueryFooter{}
			if err := f.UnmarshalVT(v); err != nil {
				return err
			}
			m.Footer = f
		}
	}
	return nil
}

func (m *ReusableStreamQueryChunk) unmarshalBatch(dAtA []byte) error {
	m.batch.Index = 0
	count := 0

	for len(dAtA) > 0 {
		num, typ, n := protowire.ConsumeTag(dAtA)
		if n < 0 {
			return protowire.ParseError(n)
		}
		dAtA = dAtA[n:]

		switch {
		case num == 1 && typ == protowire.VarintType:
			v, n := protowire.ConsumeVarint(dAtA)
			if n < 0 {
				return protowire.ParseError(n)
			}
			dAtA = dAtA[n:]
			m.batch.Index = uint32(v)
		case num == 2 && typ == protowire.BytesType:
			v, n := protowire.ConsumeBytes(dAtA)
			if n < 0 {
				return protowire.ParseError(n)
			}
			dAtA = dAtA[n:]

			if count == len(m.pool) {
				m.pool = append(m.pool, &Row{})
			}
			r := m.pool[count]
			r.resetForReuse()
			if err := r.UnmarshalVT(v); err != nil {
				return err
			}
			count++
		default:
			n = protowire.ConsumeFieldValue(num, typ, dAtA)
			if n < 0 {
				return protowire.ParseError(n)
			}
			dAtA = dAtA[n:]
		}
	}

	m.batch.Rows = m.pool[:count]
	return nil
}

// resetForReuse zeroes a Row's fields while keeping the Data slice capacity
// and the Timestamp allocation, so the generated UnmarshalVT (which appends
// to Data[:0] and reuses a non-nil Timestamp) does not allocate. Needed
// because UnmarshalVT leaves fields absent from the wire untouched.
func (m *Row) resetForReuse() {
	if m.Timestamp != nil {
		m.Timestamp.Seconds = 0
		m.Timestamp.Nanos = 0
	}
	m.Data = m.Data[:0]
	m.Prefix = ""
	m.unknownFields = m.unknownFields[:0]
}
