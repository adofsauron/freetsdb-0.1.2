package haraft

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Jille/raft-grpc-leader-rpc/leaderhealth"
	"github.com/Jille/raft-grpc-leader-rpc/rafterrors"
	transport "github.com/Jille/raft-grpc-transport"
	"github.com/Jille/raftadmin"
	"github.com/hashicorp/raft"
	boltdb "github.com/hashicorp/raft-boltdb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"influxdb.cluster/models"
	"influxdb.cluster/query"
	pb "influxdb.cluster/services/haraft/internal"
	"influxdb.cluster/tcp"
	"influxdb.cluster/tsdb"

)

var g_localaddr = ""

const NodeMuxHeader = 9
const RequestClusterJoin = 1
const RequestDataServerWrite = 2
const RequestDataServerQuery = 3

type Request struct {
	Type  uint64   `json:"Type"`
	Peers []string `json:"Peers"`
	Data  string   `json:"Data"`
}

type Reponse struct {
	Code int    `json:"Code"`
	Msg  string `json:"Msg"`
}

const (
	RFTYPE_WRITEPOINT = 1
	RFTYPE_QUERY      = 2
)

type WritePointsPrivilegedApply func(database, retentionPolicy string, consistencyLevel uint64, points []models.Point) error
type ServeQueryApply func(qry string, uid string, opts query.ExecutionOptions) error

type Service struct {
	wg  sync.WaitGroup
	err chan error

	TSDBStore interface {
		Shard(id uint64) *tsdb.Shard
	}

	Logger        *zap.Logger
	EngineOptions tsdb.EngineOptions

	raft *raft.Raft

	WritePointsPrivilegedApply WritePointsPrivilegedApply
	ServeQueryApply            ServeQueryApply
}

func NewService(opt tsdb.EngineOptions) *Service {
	return &Service{
		err:           make(chan error),
		Logger:        zap.NewNop(),
		EngineOptions: opt,
	}
}

func (s *Service) Close() error {
	s.wg.Wait()
	return nil
}

func (s *Service) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("service", "haraft"))
}

func (s *Service) Open() error {
	s.Logger.Info("Starting haraft service")

	s.wg.Add(1)
	go s.serve()
	return nil
}

func (s *Service) serve() {
	haRaftAddr := s.EngineOptions.Config.HaRaftAddr
	haRaftDir := s.EngineOptions.Config.HaRaftDir
	haRaftId := s.EngineOptions.Config.HaRaftId

	s.Logger.Info(fmt.Sprintf("haraft serve haRaftAddr: %s", haRaftAddr))
	s.Logger.Info(fmt.Sprintf("haraft serve haRaftDir: %s", haRaftDir))
	s.Logger.Info(fmt.Sprintf("haraft serve haRaftId: %s", haRaftId))

	ctx := context.Background()
	_, port, err := net.SplitHostPort(haRaftAddr)
	if nil != err {
		s.Logger.Error(fmt.Sprintf("failed to parse local address (%q): %v", haRaftAddr, err))
		return
	}
	sock, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if nil != err {
		s.Logger.Error(fmt.Sprintf("failed to listen: %v", err))
	}

	wt := &wordTracker{
		HaRaftService: s,
	}
	r, tm, err := s.newRaft(ctx, haRaftDir, haRaftId, haRaftAddr, wt)
	if nil != err {
		s.Logger.Error(fmt.Sprintf("haraft failed to start raft: %v", err))
		return
	}

	s.raft = r
	grpcSrv := grpc.NewServer()
	rpcSrv := &rpcInterface{
		wordTracker: wt,
		raft:        r,
	}

	pb.RegisterExampleServer(grpcSrv, rpcSrv)
	tm.Register(grpcSrv)
	leaderhealth.Setup(r, grpcSrv, []string{"Example"})
	raftadmin.Register(grpcSrv, r)
	reflection.Register(grpcSrv)

	g_localaddr = haRaftAddr
	if err := grpcSrv.Serve(sock); nil != err {
		s.Logger.Error(fmt.Sprintf("haraft failed to start server: %v", err))
		return
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *Service) newRaft(ctx context.Context, haRaftDir string, myID string, myAddress string, fsm raft.FSM) (*raft.Raft, *transport.Manager, error) {

	raftBootstrap := false

	{
		raftDir := haRaftDir + "/" + myID

		isExist, err := PathExists(raftDir)
		if nil != err {
			s.Logger.Error(fmt.Sprintf(`newRaft fail, PathExists fail, dir = %s, err = %v`, raftDir, err))
			return nil, nil, fmt.Errorf(`newRaft fail, PathExists fail, dir = %s, err = %v`, raftDir, err)
		}

		if !isExist {
			raftBootstrap = true

			err := os.Mkdir(raftDir, os.ModePerm)
			if nil != err {
				s.Logger.Error(fmt.Sprintf(`newRaft fail, create haRaftDir fail, dir = %s, err = %v`, raftDir, err))
				return nil, nil, fmt.Errorf(`newRaft fail, create haRaftDir fail, dir = %s, err = %v`, raftDir, err)
			}
		}
	}

	c := raft.DefaultConfig()
	c.LocalID = raft.ServerID(myID)

	baseDir := filepath.Join(haRaftDir, myID)

	ldb, err := boltdb.NewBoltStore(filepath.Join(baseDir, "logs.dat"))
	if nil != err {
		return nil, nil, fmt.Errorf(`boltdb.NewBoltStore(%q): %v`, filepath.Join(baseDir, "logs.dat"), err)
	}

	sdb, err := boltdb.NewBoltStore(filepath.Join(baseDir, "stable.dat"))
	if nil != err {
		return nil, nil, fmt.Errorf(`boltdb.NewBoltStore(%q): %v`, filepath.Join(baseDir, "stable.dat"), err)
	}

	fss, err := raft.NewFileSnapshotStore(baseDir, 3, os.Stderr)
	if nil != err {
		return nil, nil, fmt.Errorf(`raft.NewFileSnapshotStore(%q, ...): %v`, baseDir, err)
	}

	tm := transport.New(raft.ServerAddress(myAddress), []grpc.DialOption{grpc.WithInsecure()})

	r, err := raft.NewRaft(c, fsm, ldb, sdb, fss, tm.Transport())
	if nil != err {
		return nil, nil, fmt.Errorf("raft.NewRaft: %v", err)
	}

	if raftBootstrap {
		cfg := raft.Configuration{
			Servers: []raft.Server{
				{
					Suffrage: raft.Voter,
					ID:       raft.ServerID(myID),
					Address:  raft.ServerAddress(myAddress),
				},
			},
		}
		f := r.BootstrapCluster(cfg)
		if err := f.Error(); nil != err {
			return nil, nil, fmt.Errorf("raft.Raft.BootstrapCluster: %v", err)
		}
	}

	return r, tm, nil
}

func (s *Service) Apply(cmd []byte, timeout time.Duration) error {
	f := s.raft.Apply(cmd, timeout)
	if err := f.Error(); nil != err {
		return rafterrors.MarkRetriable(err)
	}

	return nil
}

func (s *Service) MarshalWrite(database string, retentionPolicy string, consistencyLevel uint64, points []models.Point) []byte {
	b := make([]byte, 8)

	{
		binary.BigEndian.PutUint64(b, (uint64)(RFTYPE_WRITEPOINT))
	}

	{
		databaseLen := uint64(len(database))
		b_n := make([]byte, 8)
		binary.BigEndian.PutUint64(b_n, databaseLen)
		b = append(b, b_n...)

		b = append(b, []byte(database)...)
	}

	{
		retentionPolicyLen := uint64(len(retentionPolicy))

		b_n := make([]byte, 8)
		binary.BigEndian.PutUint64(b_n, retentionPolicyLen)
		b = append(b, b_n...)

		b = append(b, []byte(retentionPolicy)...)
	}

	{
		b_n := make([]byte, 8)
		binary.BigEndian.PutUint64(b_n, consistencyLevel)
		b = append(b, b_n...)
	}

	for _, p := range points {
		b = append(b, []byte(p.String())...)
		b = append(b, '\n')
	}

	return b
}

func (s *Service) UnmarshalWrite(b []byte) (string, string, uint64, []models.Point, error) {
	if len(b) < 8 {
		return "", "", 0, nil, fmt.Errorf("too short: len = %d", len(b))
	}

	index := (uint64)(0)
	over := index + 8

	index = over
	over = index + 8
	databaseLen := binary.BigEndian.Uint64(b[index:over])

	index = index + 8
	over = index + databaseLen
	database := (string)(b[index:over])

	index = index + databaseLen
	over = index + 8
	retentionPolicyLen := binary.BigEndian.Uint64(b[index:over])

	index += 8
	over = index + retentionPolicyLen
	retentionPolicy := (string)(b[index:over])

	index = index + retentionPolicyLen
	over = index + 8
	consistencyLevel := (uint64)(binary.BigEndian.Uint64(b[index:over]))

	index = over
	points, err := models.ParsePoints(b[index:])

	return database, retentionPolicy, consistencyLevel, points, err
}

func (s *Service) WritePointsPrivileged(database string, retentionPolicy string, consistencyLevel uint64, points []models.Point) error {

	// leader

	leaderAddr := s.GetLeader()
	if g_localaddr == leaderAddr {
		s.Logger.Info(fmt.Sprintf("haraft WritePointsPrivileged Apply, I'am leader: %s", leaderAddr))
		b := s.MarshalWrite(database, retentionPolicy, consistencyLevel, points)
		err := s.Apply(b, time.Second)
		if nil != err {
			s.Logger.Error(fmt.Sprintf("haraft WritePointsPrivileged Apply fail, err: %s", leaderAddr))
			return err
		}

		s.Logger.Info(fmt.Sprintf("haraft WritePointsPrivileged Apply ok, database: %s, retentionPolicy: %s", database, retentionPolicy))
		return nil
	}

	// follow proxy

	s.Logger.Info(fmt.Sprintf("haraft WritePointsPrivileged proxy, I'am follow, proxy to leader : %s", leaderAddr))

	r := Request{}
	r.Type = RequestDataServerWrite
	r.Peers = nil
	r.Data = string(s.MarshalWrite(database, retentionPolicy, consistencyLevel, points))

	conn, err := tcp.Dial("tcp", leaderAddr, NodeMuxHeader)
	if nil != err {
		s.Logger.Error(fmt.Sprintf("ERROR: addData fail, tcp.Dial err, leaderAddr = %s, err = %v \n", leaderAddr, err))
		return err
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(r); nil != err {
		s.Logger.Error(fmt.Sprintf("ERROR: addData fail, json.NewEncoder.Encode err, leaderAddr = %s, err = %v \n", leaderAddr, err))
		return fmt.Errorf(fmt.Sprintf("Encode snapshot request: %v", err))
	}

	res := Reponse{}
	if err := json.NewDecoder(conn).Decode(&res); nil != err {
		s.Logger.Error(fmt.Sprintf("ERROR: addData fail, json.NewEncoder.Decode err, leaderAddr = %s, err = %v \n", leaderAddr, err))
		return err
	}

	if 0 != res.Code {
		s.Logger.Error(fmt.Sprintf(`haraft WritePointsPrivileged response fail, code = %d, msg = %s`, res.Code, res.Msg))
		return fmt.Errorf(`haraft WritePointsPrivileged response fail, code = %d, msg = %s`, res.Code, res.Msg)
	}

	return nil
}

func (s *Service) GetLeader() string {
	return string(s.raft.Leader())
}

func (s *Service) IstLeader() bool {
	return g_localaddr == s.GetLeader()
}

func (s *Service) MarshalQuery(qry string, uid string, opts query.ExecutionOptions) ([]byte, error) {
	b := make([]byte, 8)

	index := 0
	{
		s.Logger.Debug(fmt.Sprintf("MarshalQuery RT index : %d", index))
		binary.BigEndian.PutUint64(b, (uint64)(RFTYPE_QUERY))

		index = index + 8
	}

	{
		queryLen := uint64(len(qry))

		s.Logger.Debug(fmt.Sprintf("MarshalQuery query len index : %d, queryLen: %d", index, queryLen))

		b_n := make([]byte, 8)
		binary.BigEndian.PutUint64(b_n, queryLen)
		b = append(b, b_n...)

		b = append(b, []byte(qry)...)

		index = index + 8 + int(queryLen)
	}

	{
		uidLen := uint64(len(uid))

		s.Logger.Debug(fmt.Sprintf("MarshalQuery uid len index : %d, uidLen: %d", index, uidLen))

		b_n := make([]byte, 8)
		binary.BigEndian.PutUint64(b_n, uidLen)
		b = append(b, b_n...)

		b = append(b, []byte(uid)...)

		index = index + 8 + int(uidLen)
	}

	{

		database := opts.Database
		retentionPolicy := opts.RetentionPolicy
		chunkSize := (uint64)(opts.ChunkSize)
		nodeID := opts.NodeID
		readOnly := "false"
		if opts.ReadOnly {
			readOnly = "true"
		}

		{

			databaseLen := uint64(len(database))

			s.Logger.Debug(fmt.Sprintf("MarshalQuery database len index : %d, databaseLen: %d", index, databaseLen))

			b_n := make([]byte, 8)
			binary.BigEndian.PutUint64(b_n, databaseLen)
			b = append(b, b_n...)

			if 0 < databaseLen {
				b = append(b, []byte(database)...)
			}

			index = index + 8 + int(databaseLen)
		}

		{
			retentionPolicyLen := uint64(len(retentionPolicy))

			s.Logger.Debug(fmt.Sprintf("MarshalQuery retentionPolicy len index : %d, retentionPolicyLen: %d", index, retentionPolicyLen))

			b_n := make([]byte, 8)
			binary.BigEndian.PutUint64(b_n, retentionPolicyLen)
			b = append(b, b_n...)

			if 0 < retentionPolicyLen {
				b = append(b, []byte(retentionPolicy)...)
			}

			index = index + 8 + int(retentionPolicyLen)
		}

		{
			s.Logger.Debug(fmt.Sprintf("MarshalQuery chunkSize len index : %d", index))

			b_n := make([]byte, 8)
			binary.BigEndian.PutUint64(b_n, chunkSize)
			b = append(b, b_n...)

			index = index + 8
		}

		{
			s.Logger.Debug(fmt.Sprintf("MarshalQuery nodeID len index : %d", index))

			b_n := make([]byte, 8)
			binary.BigEndian.PutUint64(b_n, nodeID)
			b = append(b, b_n...)

			index = index + 8
		}

		{
			readOnlyLen := uint64(len(readOnly))

			s.Logger.Debug(fmt.Sprintf("MarshalQuery readOnly len index : %d, readOnlyLen: %d", index, readOnlyLen))

			b_n := make([]byte, 8)
			binary.BigEndian.PutUint64(b_n, readOnlyLen)
			b = append(b, b_n...)

			b = append(b, []byte(readOnly)...)

			index = index + 8
		}
	}

	s.Logger.Debug(fmt.Sprintf("MarshalQuery over index : %d", index))

	return b, nil
}

func (s *Service) UnmarshalQuery(b []byte) (string, string, query.ExecutionOptions, error) {

	index := (uint64)(0)
	over := index + 8

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery rt len index : %d", index))

	uid := ""
	qry := ""
	var opts query.ExecutionOptions

	index = over
	over = index + 8

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery query len index : %d", index))
	queryLen := binary.BigEndian.Uint64(b[index:over])

	index = over
	over = over + queryLen

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery query value index : %d, queryLen: %d", index, queryLen))
	qry = (string)(b[index:over])

	index = over
	over = index + 8

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery uid len index : %d", index))
	uidLen := binary.BigEndian.Uint64(b[index:over])

	index = over
	over = over + uidLen

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery uid value index : %d, uidLen: %d", index, uidLen))
	uid = (string)(b[index:over])

	index = over
	over = index + 8

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery database len index : %d", index))
	databaseLen := binary.BigEndian.Uint64(b[index:over])

	if 0 < databaseLen {
		index = over
		over = over + databaseLen

		s.Logger.Debug(fmt.Sprintf("UnmarshalQuery database value index : %d, databaseLen: %d", index, databaseLen))
		database := (string)(b[index:over])
		opts.Database = database
	}

	index = over
	over = index + 8

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery retentionPolicy len index : %d", index))
	retentionPolicyLen := binary.BigEndian.Uint64(b[index:over])

	if 0 < retentionPolicyLen {
		index = over
		over = over + retentionPolicyLen

		s.Logger.Debug(fmt.Sprintf("UnmarshalQuery retentionPolicy value index : %d, databaseLen: %d", index, retentionPolicyLen))
		retentionPolicy := (string)(b[index:over])
		opts.RetentionPolicy = retentionPolicy
	}

	index = over
	over = index + 8

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery chunkSize len index : %d", index))
	chunkSize := binary.BigEndian.Uint64(b[index:over])
	opts.ChunkSize = int(chunkSize)

	index = over
	over = index + 8

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery nodeID len index : %d", index))
	nodeID := binary.BigEndian.Uint64(b[index:over])
	opts.NodeID = nodeID

	index = over
	over = index + 8

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery readOnly len index : %d", index))
	readOnlyLen := binary.BigEndian.Uint64(b[index:over])

	index = over
	over = over + readOnlyLen

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery readOnly value index : %d, readOnlyLen: %d", index, readOnlyLen))
	readonly := (string)(b[index:over])
	opts.ReadOnly = readonly == "true"

	index = over

	s.Logger.Debug(fmt.Sprintf("UnmarshalQuery  over index : %d", index))
	return qry, uid, opts, nil
}

func (s *Service) ServeQuery(qry string, uid string, opts query.ExecutionOptions) error {
	b, err := s.MarshalQuery(qry, uid, opts)
	if nil != err {
		s.Logger.Error(fmt.Sprintf(`haraft ServeQuery fail, MarshalQuery err = %v`, err))
		return err
	}

	s.Logger.Info(fmt.Sprintf("haraft ServeQuery, query: %s, uid: %s", qry, uid))

	// leader

	leaderAddr := s.GetLeader()
	if g_localaddr == leaderAddr {
		s.Logger.Info(fmt.Sprintf("haraft ServeQuery, I'am leader: %s, call raft.Apply on myself", leaderAddr))
		err := s.Apply(b, time.Second)
		if nil != err {
			s.Logger.Info(fmt.Sprintf("haraft ServeQuery fail, raft.Apply err = %v", err))
			return err
		}

		s.Logger.Info(fmt.Sprintf("haraft ServeQuery ok, raft.Apply over, query = %s", qry))
		return nil
	}

	// follow proxy

	s.Logger.Info(fmt.Sprintf("haraft ServeQuery, I'am follow, proxy to leader : %s", leaderAddr))

	r := Request{}
	r.Type = RequestDataServerQuery
	r.Peers = nil
	r.Data = string(b)

	conn, err := tcp.Dial("tcp", leaderAddr, NodeMuxHeader)
	if nil != err {
		s.Logger.Error(fmt.Sprintf("haraft ServeQuery, addData fail, tcp.Dial err, leaderAddr = %s, err = %v \n", leaderAddr, err))
		return err
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(r); nil != err {
		s.Logger.Error(fmt.Sprintf("haraft ServeQuery, addData fail, json.NewEncoder.Encode err, leaderAddr = %s, err = %v \n", leaderAddr, err))
		return fmt.Errorf(fmt.Sprintf("Encode snapshot request: %v", err))
	}

	res := Reponse{}
	if err := json.NewDecoder(conn).Decode(&res); nil != err {
		s.Logger.Error(fmt.Sprintf("haraft ServeQuery, addData fail, json.NewEncoder.Decode err, leaderAddr = %s, err = %v \n", leaderAddr, err))
		return err
	}

	if 0 != res.Code {
		s.Logger.Error(fmt.Sprintf("haraft ServeQuery fail, code = %d, msg = %s", res.Code, res.Msg))
		return fmt.Errorf("haraft ServeQuery fail, code = %d, msg = %s", res.Code, res.Msg)
	}

	s.Logger.Info(fmt.Sprintf("haraft ServeQuery proxy ok, to leader : %s", leaderAddr))
	return nil
}
