package SyncTest

import (
	"cse224/proj5/pkg/syncinator"
	"fmt"
	"testing"
	"time"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func TestRaftSetLeader(t *testing.T) {
	cfgPath := "./config_files/6nodes.json"
	test := InitTest(cfgPath)
	defer EndTest(test)

	leaderIdx := 0
	res, err := test.Clients[leaderIdx].SetLeader(test.Context, &emptypb.Empty{})
	t.Logf("%d ||| %v ||| %v\n", leaderIdx, res, err)
	res, err = test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})
	t.Logf("%d ||| %v ||| %v\n", leaderIdx, res, err)
	time.Sleep(100 * time.Millisecond)

	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}

	leaderIdx = 1
	res, err = test.Clients[leaderIdx].SetLeader(test.Context, &emptypb.Empty{})
	t.Logf("%d ||| %v ||| %v\n", leaderIdx, res, err)
	res, err = test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})
	t.Logf("%d ||| %v ||| %v\n", leaderIdx, res, err)
	time.Sleep(100 * time.Millisecond)

	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}

	leaderIdx = 2
	res, err = test.Clients[leaderIdx].SetLeader(test.Context, &emptypb.Empty{})
	t.Logf("%d ||| %v ||| %v\n", leaderIdx, res, err)
	res, err = test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})
	t.Logf("%d ||| %v ||| %v\n", leaderIdx, res, err)
	time.Sleep(100 * time.Millisecond)

	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}
}

func TestRaftFollowersGetUpdates(t *testing.T) {
	cfgPath := "./config_files/6nodes.json"
	test := InitTest(cfgPath)
	defer EndTest(test)

	leaderIdx := 0
	test.Clients[leaderIdx].SetLeader(test.Context, &emptypb.Empty{})
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})

	filemeta1 := &syncinator.FileMetaData{
		Filename:      "testFile1",
		Version:       1,
		BlockHashList: nil,
	}
	test.Clients[leaderIdx].UpdateFile(test.Context, filemeta1)
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})
	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}
	fmt.Println()

	leaderIdx = 1
	test.Clients[leaderIdx].SetLeader(test.Context, &emptypb.Empty{})
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})

	filemeta2 := &syncinator.FileMetaData{
		Filename:      "testFile2",
		Version:       1,
		BlockHashList: nil,
	}
	test.Clients[leaderIdx].UpdateFile(test.Context, filemeta2)
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})
	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}
	fmt.Println()
}

func TestRaftLogsConsistentLeaderCrashesBeforeHeartbeat(t *testing.T) {
	cfgPath := "./config_files/6nodes.json"
	test := InitTest(cfgPath)
	defer EndTest(test)

	leaderIdx := 0
	test.Clients[leaderIdx].SetLeader(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)
	test.Clients[leaderIdx].Crash(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)
	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}
	fmt.Println()
	time.Sleep(200 * time.Millisecond)

	test.Clients[2].Crash(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)

	leaderIdx = 1
	test.Clients[leaderIdx].SetLeader(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)
	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}
	fmt.Println()
	time.Sleep(200 * time.Millisecond)

	test.Clients[0].Restore(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)
	test.Clients[2].Restore(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)

	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}
	fmt.Println()
}

func TestReadOnly(t *testing.T) {
	cfgPath := "./config_files/5nodes.json"
	test := InitTest(cfgPath)
	defer EndTest(test)

	leaderIdx := 0
	test.Clients[leaderIdx].SetLeader(test.Context, &emptypb.Empty{})
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})

	leaderIdx = 1
	test.Clients[leaderIdx].SetLeader(test.Context, &emptypb.Empty{})
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})

	leaderIdx = 2
	test.Clients[leaderIdx].SetLeader(test.Context, &emptypb.Empty{})
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})

	time.Sleep(200 * time.Millisecond)

	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}
	fmt.Println()

	filemeta1 := &syncinator.FileMetaData{
		Filename:      "testFile1",
		Version:       1,
		BlockHashList: nil,
	}
	res, err := test.Clients[0].UpdateFile(test.Context, filemeta1)
	t.Logf("%d ||| %v ||| %v\n", 0, res, err)
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})

	res, err = test.Clients[2].UpdateFile(test.Context, filemeta1)
	t.Logf("%d ||| %v ||| %v\n", 2, res, err)
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})

	time.Sleep(200 * time.Millisecond)

	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}
	fmt.Println()

	test.Clients[0].Crash(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)

	res1, err := test.Clients[2].GetFileInfoMap(test.Context, &emptypb.Empty{})
	t.Logf("%d ||| %v ||| %v\n", 2, res1, err)

	test.Clients[1].Crash(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)

	res2, err := test.Clients[2].GetBlockStoreMap(test.Context, &syncinator.BlockHashes{Hashes: []string{}})
	t.Logf("%d ||| %v ||| %v\n", 2, res2, err)

	res3, err := test.Clients[2].GetBlockStoreAddrs(test.Context, &emptypb.Empty{})
	t.Logf("%d ||| %v ||| %v\n", 2, res3, err)

	filemeta1 = &syncinator.FileMetaData{
		Filename:      "testFile1",
		Version:       2,
		BlockHashList: nil,
	}
	res, err = test.Clients[2].UpdateFile(test.Context, filemeta1)
	t.Logf("%d ||| %v ||| %v\n", 2, res, err)
	test.Clients[leaderIdx].SendHeartbeat(test.Context, &emptypb.Empty{})

	time.Sleep(200 * time.Millisecond)
	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}
	fmt.Println()

	test.Clients[0].Restore(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)

	test.Clients[1].Restore(test.Context, &emptypb.Empty{})
	time.Sleep(200 * time.Millisecond)

	for id, server := range test.Clients {
		res, err := server.GetInternalState(test.Context, &emptypb.Empty{})
		t.Logf("%d ||| %v ||| %v\n", id, res, err)
	}
	fmt.Println()
}
