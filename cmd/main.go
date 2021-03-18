package main

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"net/http"
	"os"
	"strings"
)

var (
	MinerUrl    = ""
	MinerToken  = ""
	DaemonUrl   = ""
	DaemonToken = ""
)

var storageMiner api.StorageMiner
var fullNode api.FullNode

func getCredentials(listenAddr string) (string, error) {
	parsedAddr, err := ma.NewMultiaddr(listenAddr)
	if err != nil {
		return "", err
	}

	_, addr, err := manet.DialArgs(parsedAddr)
	if err != nil {
		return "", err
	}
	return addr, nil
}

func main() {
	// 检索矿工ID
	// RETRIEVE MINER ID
	actorAddress, err := storageMiner.ActorAddress(context.Background())
	if err != nil {
		fmt.Println("actorAddress error", err)
		return
	}
	minerId := actorAddress

	// 获取本地主机名
	minerHost, err := os.Hostname()
	if err != nil {
		fmt.Println("minerHost error", err)
		return
	}

	// RETRIEVE TIPSET + CHAINHEAD
	// 链头
	chainHead, err := fullNode.ChainHead(context.Background())
	if err != nil {
		fmt.Println("ChainHead error", err)
		return
	}

	fmt.Println("# HELP lotus_chain_height return current height")
	fmt.Println("# TYPE lotus_chain_height counter")
	fmt.Println("lotus_chain_height {{ minerId=", minerId, ", minerHost=", minerHost, "chainHeadHeight=", chainHead.Height())
}

func apiURI(addr string) string {
	return "http://" + addr + "/rpc/v0"
}

func NewLotusFullNode(ctx context.Context, addr, token string) (api.FullNode, func(), error) {
	headers := http.Header{"Authorization": []string{"Bearer " + token}}
	var fullNode apistruct.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin", []interface{}{&fullNode.Internal, &fullNode.CommonStruct.Internal}, headers)
	if err != nil {
		return nil, func() {}, err
	}
	return &fullNode, closer, err
}

func NewLotusStorageMiner(ctx context.Context, addr, token string) (api.StorageMiner, func(), error) {
	headers := http.Header{"Authorization": []string{"Bearer " + token}}
	var storageMiner apistruct.StorageMinerStruct
	closer, err := jsonrpc.NewMergeClient(ctx, addr, "Filecoin", []interface{}{&storageMiner.Internal, &storageMiner.CommonStruct.Internal}, headers)
	if err != nil {
		return nil, func() {}, err
	}
	return &storageMiner, closer, err
}

func init() {
	miner := os.Getenv("MINER_API_INFO")
	minerArr := strings.Split(miner, ":")
	minerUrl, err := getCredentials(minerArr[1])
	if err != nil {
		fmt.Println("getCredentials#minerUrl", minerArr)
		return
	}
	MinerUrl = apiURI(minerUrl)
	MinerToken = minerArr[0]
	fmt.Println("MinerUrl:", MinerUrl, "MinerToken:", MinerToken)
	// FULLNODE_API_INFO
	nodeApi := os.Getenv("FULLNODE_API_INFO")
	nodeApiArr := strings.Split(nodeApi, ":")
	nodeApiUrl, err := getCredentials(nodeApiArr[1])
	if err != nil {
		fmt.Println("getCredentials#nodeApiUrl", nodeApiArr)
		return
	}
	DaemonUrl = apiURI(nodeApiUrl)
	DaemonToken = nodeApiArr[0]
	fmt.Println("DaemonUrl:", DaemonUrl, "DaemonToken:", DaemonToken)
	fullNode, _, err = NewLotusFullNode(context.Background(), DaemonUrl, DaemonToken)
	if err != nil {
		fmt.Println("NewLotusFullNode error")
	}
	storageMiner, _, err = NewLotusStorageMiner(context.Background(), MinerUrl, MinerToken)
	if err != nil {
		fmt.Println("NewLotusStorageMiner error")
	}
}
