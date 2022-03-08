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

// WordTracker keeps track of the three longest words it ever saw.
type WordTracker struct {
	mtx           sync.RWMutex
	words         [3]string
	HaRaftService *Service
}

var _ raft.FSM = &WordTracker{}

func cloneWords(words [3]string) []string {
	var ret [3]string
	copy(ret[:], words[:])
	return ret[:]
}

func (f *WordTracker) ApplyWritePoint(b []byte) error {
	database, retentionPolicy, consistencyLevel, points, err := f.HaRaftService.UnmarshalWrite(b)
	if nil != err {
		f.HaRaftService.Logger.Error(fmt.Sprintf("ERROR: WordTracker ApplyWritePoint fail, UnmarshalWrite err = %v", err))
		return err
	}

	err = f.HaRaftService.WritePointsPrivilegedApply(database, retentionPolicy, consistencyLevel, points)
	if nil != err {
		f.HaRaftService.Logger.Error(fmt.Sprintf("ERROR: WordTracker ApplyWritePoint fail, PointsWriter WriteToShardApply err = %v", err))
		return err
	}

	f.HaRaftService.Logger.Info(fmt.Sprintf("WordTracker ApplyWritePoint ok, database: %s, consistencyLevel: %d ", database, consistencyLevel))
	return nil
}

func (f *WordTracker) ApplyQuery(b []byte) error {

	if f.HaRaftService.IstLeader() {
		return nil
	}

	qry, uid, opts, err := f.HaRaftService.UnmarshalQuery(b)
	if nil != err {
		f.HaRaftService.Logger.Error(fmt.Sprint("ERROR: WordTracker ApplyQuery fail, UnmarshalQuery err = %s", err.Error()))
		return err
	}

	if "" == qry {
		f.HaRaftService.Logger.Info(fmt.Sprint("WordTracker ApplyQuery no execute, query is nil, uid: %s", uid))
		return nil
	}

	err = f.HaRaftService.ServeQueryApply(qry, uid, opts)
	if nil != err {
		f.HaRaftService.Logger.Error(fmt.Sprint("ERROR: WordTracker ApplyQuery fail, ServeQueryApply err = %v, uid:%s", err, uid))
		return err
	}

	f.HaRaftService.Logger.Info(fmt.Sprintf("WordTracker ApplyQuery ok, database: %s, uid: %s", opts.Database, uid))
	return nil
}

func (f *WordTracker) ApplyByte(b []byte) interface{} {
	if len(b) < 8 {
		return fmt.Errorf("too short: len = %d", len(b))
	}

	var err error
	b_n := b[0:8]
	rf_type := int(binary.BigEndian.Uint64(b_n))

	f.HaRaftService.Logger.Info(fmt.Sprintf("WordTracker Apply, rf_type: %d", rf_type))

	rf_type = (int)(rf_type)
	switch rf_type {
	case RFTYPE_WRITEPOINT:
		err = f.ApplyWritePoint(b)
	case RFTYPE_QUERY:
		err = f.ApplyQuery(b)
	}

	return err
}

func (f *WordTracker) Apply(l *raft.Log) interface{} {
	f.mtx.Lock() // 这个读写锁, 是否有必要?
	defer f.mtx.Unlock()

	b := []byte(l.Data)
	return f.ApplyByte(b)
}

func (f *WordTracker) Snapshot() (raft.FSMSnapshot, error) {
	// Make sure that any future calls to f.Apply() don't change the snapshot.
	return &snapshot{cloneWords(f.words)}, nil
}

func (f *WordTracker) Restore(r io.ReadCloser) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	words := strings.Split(string(b), "\n")
	copy(f.words[:], words)
	return nil
}
