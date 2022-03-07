package haraft

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/hashicorp/raft"

)

// wordTracker keeps track of the three longest words it ever saw.
type wordTracker struct {
	mtx           sync.RWMutex
	words         [3]string
	HaRaftService *Service
}

var _ raft.FSM = &wordTracker{}

// compareWords returns true if a is longer (lexicography breaking ties).
func compareWords(a, b string) bool {
	if len(a) == len(b) {
		return a < b
	}
	return len(a) > len(b)
}

func cloneWords(words [3]string) []string {
	var ret [3]string
	copy(ret[:], words[:])
	return ret[:]
}

func (f *wordTracker) ApplyWritePoint(b []byte) error {
	database, retentionPolicy, consistencyLevel, points, err := f.HaRaftService.UnmarshalWrite(b)
	if nil != err {
		f.HaRaftService.Logger.Error(fmt.Sprintf("ERROR: wordTracker ApplyWritePoint fail, UnmarshalWrite err = %v", err))
		return err
	}

	err = f.HaRaftService.WritePointsPrivilegedApply(database, retentionPolicy, consistencyLevel, points)
	if nil != err {
		f.HaRaftService.Logger.Error(fmt.Sprintf("ERROR: wordTracker ApplyWritePoint fail, PointsWriter WriteToShardApply err = %v", err))
		return err
	}

	f.HaRaftService.Logger.Info(fmt.Sprintf("wordTracker ApplyWritePoint ok, database: %s, consistencyLevel: %d ", database, consistencyLevel))
	return nil
}

func (f *wordTracker) ApplyQuery(b []byte) error {

	if f.HaRaftService.IstLeader() {
		return nil
	}

	qry, uid, opts, err := f.HaRaftService.UnmarshalQuery(b)
	if nil != err {
		f.HaRaftService.Logger.Error(fmt.Sprint("ERROR: wordTracker ApplyQuery fail, UnmarshalQuery err = %s", err.Error()))
		return err
	}

	if "" == qry {
		f.HaRaftService.Logger.Info(fmt.Sprint("wordTracker ApplyQuery no execute, query is nil, uid: %s", uid))
		return nil
	}

	err = f.HaRaftService.ServeQueryApply(qry, uid, opts)
	if nil != err {
		f.HaRaftService.Logger.Error(fmt.Sprint("ERROR: wordTracker ApplyQuery fail, ServeQueryApply err = %v, uid:%s", err, uid))
		return err
	}

	f.HaRaftService.Logger.Info(fmt.Sprintf("wordTracker ApplyQuery ok, database: %s, uid: %s", opts.Database, uid))
	return nil
}

func (f *wordTracker) Apply(l *raft.Log) interface{} {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	b := []byte(l.Data)

	var err error
	b_n := b[0:8]
	rf_type := int(binary.BigEndian.Uint64(b_n))

	f.HaRaftService.Logger.Info(fmt.Sprintf("wordTracker Apply, rf_type: %d", rf_type))

	rf_type = (int)(rf_type)
	switch rf_type {
	case RFTYPE_WRITEPOINT:
		err = f.ApplyWritePoint(b)
	case RFTYPE_QUERY:
		err = f.ApplyQuery(b)
	}

	return err
}

func (f *wordTracker) Snapshot() (raft.FSMSnapshot, error) {
	// Make sure that any future calls to f.Apply() don't change the snapshot.
	return &snapshot{cloneWords(f.words)}, nil
}

func (f *wordTracker) Restore(r io.ReadCloser) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	words := strings.Split(string(b), "\n")
	copy(f.words[:], words)
	return nil
}
