package syncinator

import (
	context "context"
	"fmt"
	"time"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type RPCClient struct {
	MetaStoreAddrs []string
	BaseDir        string
	BlockSize      int
}

func (syncClient *RPCClient) GetBlock(blockHash string, blockStoreAddr string, block *Block) error {
	conn, err := grpc.Dial(blockStoreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	c := NewBlockStoreClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	b, err := c.GetBlock(ctx, &BlockHash{Hash: blockHash})
	if err != nil {
		conn.Close()
		return err
	}
	block.BlockData = b.BlockData
	block.BlockSize = b.BlockSize

	return conn.Close()
}

func (syncClient *RPCClient) PutBlock(block *Block, blockStoreAddr string, succ *bool) error {
	conn, err := grpc.Dial(blockStoreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	c := NewBlockStoreClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s, err := c.PutBlock(ctx, block)
	if err != nil {
		conn.Close()
		return err
	}
	*succ = s.Flag

	return conn.Close()
}

func (syncClient *RPCClient) MissingBlocks(blockHashesIn []string, blockStoreAddr string, blockHashesOut *[]string) error {
	conn, err := grpc.Dial(blockStoreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	c := NewBlockStoreClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	h, err := c.MissingBlocks(ctx, &BlockHashes{Hashes: blockHashesIn})
	if err != nil {
		conn.Close()
		return err
	}
	*blockHashesOut = h.Hashes

	return conn.Close()
}

func (syncClient *RPCClient) GetBlockHashes(blockStoreAddr string, blockHashes *[]string) error {
	conn, err := grpc.Dial(blockStoreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	c := NewBlockStoreClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	h, err := c.GetBlockHashes(ctx, &emptypb.Empty{})
	if err != nil {
		conn.Close()
		return err
	}
	*blockHashes = h.Hashes

	return conn.Close()
}

func (syncClient *RPCClient) GetFileInfoMap(serverFileInfoMap *map[string]*FileMetaData) error {
	for _, server := range syncClient.MetaStoreAddrs {
		conn, err := grpc.Dial(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}

		c := NewRaftSyncinatorClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		f, err := c.GetFileInfoMap(ctx, &emptypb.Empty{})
		if err != nil {
			continue
		}
		*serverFileInfoMap = f.FileInfoMap

		return conn.Close()
	}
	return fmt.Errorf("could not find a leader")
}

func (syncClient *RPCClient) UpdateFile(fileMetaData *FileMetaData, latestVersion *int32) error {
	for _, server := range syncClient.MetaStoreAddrs {
		conn, err := grpc.Dial(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}

		c := NewRaftSyncinatorClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		l, err := c.UpdateFile(ctx, fileMetaData)
		if err != nil {
			continue
		}
		*latestVersion = l.Version

		return conn.Close()
	}
	return fmt.Errorf("could not find a leader")
}

func (syncClient *RPCClient) GetBlockStoreMap(blockHashesIn []string, blockStoreMap *map[string][]string) error {
	for _, server := range syncClient.MetaStoreAddrs {
		conn, err := grpc.Dial(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}

		c := NewRaftSyncinatorClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		b, err := c.GetBlockStoreMap(ctx, &BlockHashes{Hashes: blockHashesIn})
		if err != nil {
			continue
		}
		for k, v := range b.BlockStoreMap {
			(*blockStoreMap)[k] = v.Hashes
		}

		return conn.Close()
	}
	return fmt.Errorf("could not find a leader")
}

func (syncClient *RPCClient) GetBlockStoreAddrs(blockStoreAddrs *[]string) error {
	for _, server := range syncClient.MetaStoreAddrs {
		conn, err := grpc.Dial(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}

		c := NewRaftSyncinatorClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		b, err := c.GetBlockStoreAddrs(ctx, &emptypb.Empty{})
		if err != nil {
			continue
		}
		*blockStoreAddrs = b.BlockStoreAddrs

		return conn.Close()
	}
	return fmt.Errorf("could not find a leader")
}

// This line guarantees all method for RPCClient are implemented
var _ ClientInterface = new(RPCClient)

// Create an Syncinator RPC client
func NewSyncinatorRPCClient(addrs []string, baseDir string, blockSize int) RPCClient {
	return RPCClient{
		MetaStoreAddrs: addrs,
		BaseDir:        baseDir,
		BlockSize:      blockSize,
	}
}
