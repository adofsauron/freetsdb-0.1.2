package haraft

import (
	"context"
	"fmt"
	"time"

	"github.com/Jille/raft-grpc-leader-rpc/rafterrors"
	"github.com/hashicorp/raft"
	pb "influxdb.cluster/services/haraft/internal"

)

type rpcInterface struct {
	WordTracker *WordTracker
	raft        *raft.Raft
}

func (r rpcInterface) AddWord(ctx context.Context, req *pb.AddWordRequest) (*pb.AddWordResponse, error) {
	fmt.Println("--> AddWord, raft::Apply ", req.GetWord())

	f := r.raft.Apply([]byte(req.GetWord()), time.Second)
	if err := f.Error(); err != nil {
		return nil, rafterrors.MarkRetriable(err)
	}

	return &pb.AddWordResponse{
		CommitIndex: f.Index(),
	}, nil
}

func (r rpcInterface) GetWords(ctx context.Context, req *pb.GetWordsRequest) (*pb.GetWordsResponse, error) {
	r.WordTracker.mtx.RLock()
	defer r.WordTracker.mtx.RUnlock()
	return &pb.GetWordsResponse{
		BestWords:   cloneWords(r.WordTracker.words),
		ReadAtIndex: r.raft.AppliedIndex(),
	}, nil
}
