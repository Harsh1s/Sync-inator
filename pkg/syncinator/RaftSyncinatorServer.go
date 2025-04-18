package syncinator

import (
	context "context"
	"log"
	"sync"

	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type RaftSyncinator struct {
	serverStatus      ServerStatus
	serverStatusMutex *sync.RWMutex

	id   int64
	term int64

	log            []*UpdateOperation
	metaStore      *MetaStore
	commitIndex    int64
	raftStateMutex *sync.RWMutex

	rpcConns   []*grpc.ClientConn
	grpcServer *grpc.Server

	/*--------------- Added --------------*/
	n int
	m int

	lastApplied      int64
	nextIndex        []int64
	matchIndex       []int64
	pendingResponses map[int64]*UpdateFileResponse

	peers []string

	/*--------------- Chaos Monkey --------------*/
	unreachableFrom map[int64]bool
	UnimplementedRaftSyncinatorServer
}

func (s *RaftSyncinator) GetFileInfoMap(ctx context.Context, empty *emptypb.Empty) (*FileInfoMap, error) {
	// Ensure that the majority of servers are up

	// Check status
	if _, err := s.checkStatus(false, -1); err != nil {
		return nil, err
	}

	// Wait for majority, ensure the meta store is up-to-date
	success := s.sendPersistentHeartbeats()
	if !success {
		// Reverted to follower
		return nil, ErrNotLeader
	}

	return s.metaStore.GetFileInfoMap(s.getNewContext(), empty)
}

func (s *RaftSyncinator) GetBlockStoreMap(ctx context.Context, hashes *BlockHashes) (*BlockStoreMap, error) {
	// Ensure that the majority of servers are up

	// Check status
	if _, err := s.checkStatus(false, -1); err != nil {
		return nil, err
	}

	// Wait for majority, ensure the meta store is up-to-date
	success := s.sendPersistentHeartbeats()
	if !success {
		// Reverted to follower
		return nil, ErrNotLeader
	}

	return s.metaStore.GetBlockStoreMap(s.getNewContext(), hashes)

}

func (s *RaftSyncinator) GetBlockStoreAddrs(ctx context.Context, empty *emptypb.Empty) (*BlockStoreAddrs, error) {
	// Ensure that the majority of servers are up

	// Check status
	if _, err := s.checkStatus(false, -1); err != nil {
		return nil, err
	}

	// Wait for majority, ensure the meta store is up-to-date
	success := s.sendPersistentHeartbeats()
	if !success {
		// Reverted to follower
		return nil, ErrNotLeader
	}

	return s.metaStore.GetBlockStoreAddrs(s.getNewContext(), empty)

}

func (s *RaftSyncinator) UpdateFile(ctx context.Context, filemeta *FileMetaData) (*Version, error) {
	// Ensure that the request gets replicated on majority of the servers.
	// Commit the entries and then apply to the state machine

	// Check status
	if _, err := s.checkStatus(false, -1); err != nil {
		return nil, err
	}

	// Append to log
	s.raftStateMutex.Lock()
	entry := &UpdateOperation{
		Term:         s.term,
		FileMetaData: filemeta,
	}
	s.log = append(s.log, entry)
	requestLogIndex := int64(len(s.log) - 1)
	s.raftStateMutex.Unlock()

	// Wait for majority, commit logs and apply to state machine
	success := s.sendPersistentHeartbeats()
	if !success {
		// Reverted to follower
		return nil, ErrNotLeader
	}

	s.raftStateMutex.Lock()
	// Get response
	response := s.pendingResponses[requestLogIndex]
	delete(s.pendingResponses, requestLogIndex)
	s.raftStateMutex.Unlock()

	return response.version, response.Err
}

// 1. Reply false if term < currentTerm (§5.1)
// 2. Reply false if log doesn’t contain an entry at prevLogIndex or whose term
// doesn't match prevLogTerm (§5.3)
// 3. If an existing entry conflicts with a new one (same index but different
// terms), delete the existing entry and all that follow it (§5.3)
// 4. Append any new entries not already in the log
// 5. If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index
// of last new entry)
func (s *RaftSyncinator) AppendEntries(ctx context.Context, input *AppendEntryInput) (*AppendEntryOutput, error) {
	// Check status
	myStatus, err := s.checkStatus(true, input.LeaderId)
	if err != nil {
		return nil, err
	}

	s.raftStateMutex.RLock()
	myTerm := s.term
	myId := s.id
	s.raftStateMutex.RUnlock()

	// Reject if peer is stale
	if input.Term < myTerm {
		return s.makeAppendEntryOutput(myTerm, myId, false, -1), nil
	}

	// Revert to follower if I am stale
	if myTerm < input.Term {
		if myStatus != ServerStatus_FOLLOWER {
			s.serverStatusMutex.Lock()
			s.serverStatus = ServerStatus_FOLLOWER
			s.serverStatusMutex.Unlock()
		}

		s.raftStateMutex.Lock()
		s.term = input.Term
		s.raftStateMutex.Unlock()
		myTerm = input.Term
	}

	s.raftStateMutex.Lock()

	// Reply false if no matched index
	if !s.isPrevLogMatched(input.PrevLogIndex, input.PrevLogTerm) {
		s.raftStateMutex.Unlock()
		return s.makeAppendEntryOutput(myTerm, myId, false, -1), nil
	}

	// Replicate log
	s.mergeLog(input.PrevLogIndex, input.Entries)
	matchedIndex := input.PrevLogIndex + int64(len(input.Entries))

	// Update commit index
	if input.LeaderCommit > s.commitIndex {
		s.commitIndex = min(input.LeaderCommit, int64(len(s.log)-1))
	}

	// Apply to state machine
	s.executeStateMachine(false)

	s.raftStateMutex.Unlock()

	return s.makeAppendEntryOutput(myTerm, myId, true, matchedIndex), nil
}

func (s *RaftSyncinator) SetLeader(ctx context.Context, _ *emptypb.Empty) (*Success, error) {
	// Check status
	s.serverStatusMutex.RLock()
	myStatus := s.serverStatus
	s.serverStatusMutex.RUnlock()
	if myStatus == ServerStatus_CRASHED {
		return &Success{Flag: false}, ErrServerCrashed
	}

	s.serverStatusMutex.Lock()
	s.serverStatus = ServerStatus_LEADER
	s.serverStatusMutex.Unlock()

	s.raftStateMutex.Lock()
	s.term++
	s.initLeaderStates()
	// Append no-op entry
	s.log = append(s.log, &UpdateOperation{Term: s.term, FileMetaData: nil})
	s.raftStateMutex.Unlock()

	// Wait for majority
	success := s.sendPersistentHeartbeats()
	if !success {
		// Reverted to follower
		return &Success{Flag: false}, ErrNotLeader
	}

	return &Success{Flag: true}, nil
}

func (s *RaftSyncinator) SendHeartbeat(ctx context.Context, _ *emptypb.Empty) (*Success, error) {
	// Check status
	if _, err := s.checkStatus(false, -1); err != nil {
		return &Success{Flag: false}, err
	}

	// Wait for majority
	success := s.sendPersistentHeartbeats()
	if !success {
		// Reverted to follower
		return &Success{Flag: false}, ErrNotLeader
	}

	return &Success{Flag: true}, nil
}

// ========== DO NOT MODIFY BELOW THIS LINE =====================================

func (s *RaftSyncinator) MakeServerUnreachableFrom(ctx context.Context, servers *UnreachableFromServers) (*Success, error) {
	s.raftStateMutex.Lock()
	if len(servers.ServerIds) == 0 {
		s.unreachableFrom = make(map[int64]bool)
		log.Printf("Server %d is reachable from all servers", s.id)
	} else {
		for _, serverId := range servers.ServerIds {
			s.unreachableFrom[serverId] = true
		}
		log.Printf("Server %d is unreachable from %v", s.id, s.unreachableFrom)
	}
	s.raftStateMutex.Unlock()

	return &Success{Flag: true}, nil
}

func (s *RaftSyncinator) Crash(ctx context.Context, _ *emptypb.Empty) (*Success, error) {
	s.serverStatusMutex.Lock()
	s.serverStatus = ServerStatus_CRASHED
	log.Printf("Server %d is crashed", s.id)
	s.serverStatusMutex.Unlock()

	return &Success{Flag: true}, nil
}

func (s *RaftSyncinator) Restore(ctx context.Context, _ *emptypb.Empty) (*Success, error) {
	s.serverStatusMutex.Lock()
	s.serverStatus = ServerStatus_FOLLOWER
	s.serverStatusMutex.Unlock()

	s.raftStateMutex.Lock()
	s.unreachableFrom = make(map[int64]bool)
	s.raftStateMutex.Unlock()

	log.Printf("Server %d is restored to follower and reachable from all servers", s.id)

	return &Success{Flag: true}, nil
}

func (s *RaftSyncinator) GetInternalState(ctx context.Context, empty *emptypb.Empty) (*RaftInternalState, error) {
	fileInfoMap, _ := s.metaStore.GetFileInfoMap(ctx, empty)
	s.serverStatusMutex.RLock()
	s.raftStateMutex.RLock()
	state := &RaftInternalState{
		Status:      s.serverStatus,
		Term:        s.term,
		CommitIndex: s.commitIndex,
		Log:         s.log,
		MetaMap:     fileInfoMap,
	}
	s.raftStateMutex.RUnlock()
	s.serverStatusMutex.RUnlock()

	return state, nil
}

var _ RaftSyncinatorInterface = new(RaftSyncinator)
