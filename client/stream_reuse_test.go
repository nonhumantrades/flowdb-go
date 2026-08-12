package client

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/nonhumantrades/flowdb-go/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
)

type fakeServer struct {
	proto.DRPCFlowDBUnimplementedServer
	batches int
	rows    int
}

func (s *fakeServer) StreamQuery(req *proto.QueryRequest, stream proto.DRPCFlowDB_StreamQueryStream) error {
	if err := stream.Send(&proto.StreamQueryChunk{Chunk: &proto.StreamQueryChunk_Header{
		Header: &proto.StreamQueryHeader{TableName: "tbl", Prefix: "pfx"},
	}}); err != nil {
		return err
	}
	count := uint64(0)
	for b := 0; b < s.batches; b++ {
		rows := make([]*proto.Row, s.rows)
		for i := range rows {
			count++
			rows[i] = &proto.Row{
				Timestamp: &timestamppb.Timestamp{Seconds: int64(count)},
				Data:      []byte(fmt.Sprintf("data-%d-%d", b, i)),
			}
		}
		if err := stream.Send(&proto.StreamQueryChunk{Chunk: &proto.StreamQueryChunk_Batch{
			Batch: &proto.StreamQueryBatch{Index: uint32(b), Rows: rows},
		}}); err != nil {
			return err
		}
	}
	return stream.Send(&proto.StreamQueryChunk{Chunk: &proto.StreamQueryChunk_Footer{
		Footer: &proto.StreamQueryFooter{Count: count},
	}})
}

func startTestServer(t *testing.T, srv *fakeServer) *Client {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mux := drpcmux.New()
	if err := proto.DRPCRegisterFlowDB(mux, srv); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = drpcserver.New(mux).Serve(ctx, lis) }()
	t.Cleanup(func() { cancel(); _ = lis.Close() })

	c, err := Dial(context.Background(), Config{Address: lis.Addr().String(), PoolSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func verifyStream(t *testing.T, c *Client, reuse bool) {
	t.Helper()
	var (
		gotRows    int
		gotBatches int
		lastIndex  = uint32(0)
	)
	params := NewStreamQueryParams().
		WithRequest(&proto.QueryRequest{TableName: "tbl"}).
		WithRowReuse(reuse).
		WithOnBatch(func(index uint32, rows []*proto.Row) error {
			gotBatches++
			lastIndex = index
			for i, r := range rows {
				want := fmt.Sprintf("data-%d-%d", index, i)
				if string(r.Data) != want || r.Timestamp == nil || r.Timestamp.Seconds == 0 {
					return fmt.Errorf("bad row %d in batch %d: %v", i, index, r)
				}
				gotRows++
			}
			return nil
		})

	resp, err := c.StreamQuery(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if resp.TableName != "tbl" || resp.Prefix != "pfx" {
		t.Fatalf("header not propagated: %+v", resp)
	}
	if gotBatches != 3 || gotRows != 12 || lastIndex != 2 {
		t.Fatalf("got %d batches / %d rows / last index %d", gotBatches, gotRows, lastIndex)
	}
	if resp.Count != 12 {
		t.Fatalf("footer count = %d", resp.Count)
	}
}

func TestStreamQueryDefaultPath(t *testing.T) {
	c := startTestServer(t, &fakeServer{batches: 3, rows: 4})

	// Default path: rows are owned by the callback and survive after StreamQuery.
	var retained []*proto.Row
	params := NewStreamQueryParams().
		WithRequest(&proto.QueryRequest{TableName: "tbl"}).
		WithOnBatch(func(index uint32, rows []*proto.Row) error {
			retained = append(retained, rows...)
			return nil
		})
	if _, err := c.StreamQuery(context.Background(), params); err != nil {
		t.Fatal(err)
	}
	if len(retained) != 12 {
		t.Fatalf("retained %d rows", len(retained))
	}
	for b := 0; b < 3; b++ {
		for i := 0; i < 4; i++ {
			r := retained[b*4+i]
			if want := fmt.Sprintf("data-%d-%d", b, i); string(r.Data) != want {
				t.Fatalf("retained row corrupted: got %q want %q", r.Data, want)
			}
		}
	}

	verifyStream(t, c, false)
}

func TestStreamQueryRowReuse(t *testing.T) {
	c := startTestServer(t, &fakeServer{batches: 3, rows: 4})
	verifyStream(t, c, true)

	// Row objects are recycled across batches (documented invalidation).
	seen := map[*proto.Row]int{}
	params := NewStreamQueryParams().
		WithRequest(&proto.QueryRequest{TableName: "tbl"}).
		WithRowReuse(true).
		WithOnBatch(func(index uint32, rows []*proto.Row) error {
			for _, r := range rows {
				seen[r]++
			}
			return nil
		})
	if _, err := c.StreamQuery(context.Background(), params); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 4 {
		t.Fatalf("expected 4 recycled row objects across 3 batches, got %d", len(seen))
	}
	for _, n := range seen {
		if n != 3 {
			t.Fatalf("row reused %d times, want 3", n)
		}
	}
}
