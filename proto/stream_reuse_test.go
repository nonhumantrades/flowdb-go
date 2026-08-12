package proto

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func mkChunkBatch(index uint32, rows ...*Row) []byte {
	c := &StreamQueryChunk{Chunk: &StreamQueryChunk_Batch{Batch: &StreamQueryBatch{Index: index, Rows: rows}}}
	b, err := c.MarshalVT()
	if err != nil {
		panic(err)
	}
	return b
}

func TestReusableChunkMatchesGenerated(t *testing.T) {
	wire := mkChunkBatch(7,
		&Row{Timestamp: timestamppb.Now(), Data: []byte("hello"), Prefix: "p1"},
		&Row{Timestamp: &timestamppb.Timestamp{Seconds: 42, Nanos: 9}, Data: []byte("world")},
	)

	var want StreamQueryChunk
	if err := want.UnmarshalVT(wire); err != nil {
		t.Fatal(err)
	}
	var got ReusableStreamQueryChunk
	if err := got.UnmarshalVT(wire); err != nil {
		t.Fatal(err)
	}

	wb := want.GetBatch()
	if got.Header != nil || got.Footer != nil || got.Batch == nil {
		t.Fatalf("expected batch chunk, got %+v", &got)
	}
	if got.Batch.Index != wb.Index || len(got.Batch.Rows) != len(wb.Rows) {
		t.Fatalf("batch mismatch: %v vs %v", got.Batch, wb)
	}
	for i := range wb.Rows {
		g, w := got.Batch.Rows[i], wb.Rows[i]
		if !bytes.Equal(g.Data, w.Data) || g.Prefix != w.Prefix ||
			g.Timestamp.Seconds != w.Timestamp.Seconds || g.Timestamp.Nanos != w.Timestamp.Nanos {
			t.Fatalf("row %d mismatch: %v vs %v", i, g, w)
		}
	}
}

func TestReusableChunkHeaderFooter(t *testing.T) {
	var m ReusableStreamQueryChunk

	hb, _ := (&StreamQueryChunk{Chunk: &StreamQueryChunk_Header{Header: &StreamQueryHeader{TableName: "t", Prefix: "p"}}}).MarshalVT()
	if err := m.UnmarshalVT(hb); err != nil {
		t.Fatal(err)
	}
	if m.Header == nil || m.Header.TableName != "t" || m.Header.Prefix != "p" || m.Batch != nil || m.Footer != nil {
		t.Fatalf("header mismatch: %+v", &m)
	}

	fb, _ := (&StreamQueryChunk{Chunk: &StreamQueryChunk_Footer{Footer: &StreamQueryFooter{Count: 5, TruncatedByLimit: true}}}).MarshalVT()
	if err := m.UnmarshalVT(fb); err != nil {
		t.Fatal(err)
	}
	if m.Footer == nil || m.Footer.Count != 5 || !m.Footer.TruncatedByLimit || m.Header != nil || m.Batch != nil {
		t.Fatalf("footer mismatch: %+v", &m)
	}
}

func TestReusableChunkRecyclesAndResets(t *testing.T) {
	var m ReusableStreamQueryChunk

	if err := m.UnmarshalVT(mkChunkBatch(1,
		&Row{Timestamp: &timestamppb.Timestamp{Seconds: 1, Nanos: 1}, Data: []byte("long-data-payload"), Prefix: "stale"},
		&Row{Timestamp: &timestamppb.Timestamp{Seconds: 2}, Data: []byte("bb")},
	)); err != nil {
		t.Fatal(err)
	}
	r0, r1 := m.Batch.Rows[0], m.Batch.Rows[1]

	// Second, smaller batch with fields omitted: rows must be reused and stale
	// values (prefix, timestamp nanos, data) must not leak through.
	if err := m.UnmarshalVT(mkChunkBatch(2, &Row{Data: []byte("x")})); err != nil {
		t.Fatal(err)
	}
	if len(m.Batch.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(m.Batch.Rows))
	}
	got := m.Batch.Rows[0]
	if got != r0 {
		t.Fatal("row object was not reused")
	}
	if string(got.Data) != "x" || got.Prefix != "" || got.Timestamp.Seconds != 0 || got.Timestamp.Nanos != 0 {
		t.Fatalf("stale state leaked: %v", got)
	}

	// Grow again: first row reused, second row reused from pool.
	if err := m.UnmarshalVT(mkChunkBatch(3, &Row{Data: []byte("a")}, &Row{Data: []byte("b"), Prefix: "p"})); err != nil {
		t.Fatal(err)
	}
	if m.Batch.Rows[0] != r0 || m.Batch.Rows[1] != r1 {
		t.Fatal("pooled rows not reused after shrink/grow")
	}
	if string(m.Batch.Rows[1].Data) != "b" || m.Batch.Rows[1].Prefix != "p" {
		t.Fatalf("bad row after reuse: %v", m.Batch.Rows[1])
	}
	if m.Batch.Index != 3 {
		t.Fatalf("index = %d", m.Batch.Index)
	}
}

// Documents the default-path ownership contract the reuse path deliberately
// changes: generated unmarshal produces fresh rows every time.
func TestGeneratedUnmarshalAllocatesFreshRows(t *testing.T) {
	wire := mkChunkBatch(1, &Row{Data: []byte("a")})
	var a, b StreamQueryChunk
	if err := a.UnmarshalVT(wire); err != nil {
		t.Fatal(err)
	}
	if err := b.UnmarshalVT(wire); err != nil {
		t.Fatal(err)
	}
	if a.GetBatch().Rows[0] == b.GetBatch().Rows[0] {
		t.Fatal("generated path unexpectedly shared rows")
	}
}

func BenchmarkChunkUnmarshal(b *testing.B) {
	rows := make([]*Row, 256)
	for i := range rows {
		rows[i] = &Row{Timestamp: &timestamppb.Timestamp{Seconds: int64(i)}, Data: bytes.Repeat([]byte("x"), 512)}
	}
	wire := mkChunkBatch(1, rows...)

	b.Run("generated", func(b *testing.B) {
		b.SetBytes(int64(len(wire)))
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var c StreamQueryChunk
			if err := c.UnmarshalVT(wire); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("reusable", func(b *testing.B) {
		b.SetBytes(int64(len(wire)))
		b.ReportAllocs()
		var c ReusableStreamQueryChunk
		for i := 0; i < b.N; i++ {
			if err := c.UnmarshalVT(wire); err != nil {
				b.Fatal(err)
			}
		}
	})
}
