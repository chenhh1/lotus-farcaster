package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/chain/types"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"io/ioutil"
	"math/big"
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
	chainHead, err := fullNode.ChainHead(context.Background())
	if err != nil {
		fmt.Println("ChainHead error", err)
		return
	}

	var emptyTipSetKey types.TipSetKey

	fmt.Println("# HELP lotus_chain_height return current height")
	fmt.Println("# TYPE lotus_chain_height counter")
	fmt.Print("lotus_chain_height { miner_id=",`"`, minerId,`"`, ", miner_host=",`"`, minerHost,`"`," } ", chainHead.Height(),"\n")

	// 生成钱包+锁定资金余额
	// GENERATE WALLET + LOCKED FUNDS BALANCES
	walletList, err := fullNode.WalletList(context.Background())
	if err != nil {
		fmt.Println("walletList error", err)
		return
	}
	fmt.Println("# HELP lotus_wallet_balance return wallet balance")
	fmt.Println("# TYPE lotus_wallet_balance gauge")
	for _, addr := range walletList {
		balance, err := fullNode.WalletBalance(context.Background(), addr)
		if err != nil {
			fmt.Println("balance error", err)
			return
		}
		addr := addr.String()
		short := addr[0:5] + "..." + addr[len(addr)-5:]
		//大整数     原值是:bigInt  -->  int  -->  bigFloat  -->   Float64
		fBalance := new(big.Float).SetInt(balance.Int)
		afterBalance, _ := fBalance.Float64()
		fmt.Print("lotus_wallet_balance { miner_id=",`"`, minerId,`"`, ", miner_host=",`"`, minerHost,`"`,", address=",`"`, addr,`"`, ", short=",`"`,short,`"`, " } ", afterBalance/1000000000000000000.0,"\n")

	}

	// 增加矿工余额
	// Add miner balance :
	minerBalanceAvailable, err := fullNode.StateMinerAvailableBalance(context.Background(), minerId, emptyTipSetKey)
	if err != nil {
		fmt.Println("minerBalanceAvailable error", err)
		return
	}
	fBalance := new(big.Float).SetInt(minerBalanceAvailable.Int)
	afterBalance, _ := fBalance.Float64()
	fmt.Print("lotus_wallet_balance { miner_id=",`"`,minerId,`"`, ", miner_host=",`"`, minerHost,`"`, ", address=",`"`, minerId,`"`, ", short=",`"`, minerId,`"`, " } ", afterBalance/1000000000000000000.0,"\n")

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
	env, err := getEnvPath()
	if err != nil {
		fmt.Println("getEnvPath error", err)
		return
	}
	minerArr := strings.Split(env.MinerApiInfo, ":")
	minerUrl, err := getCredentials(minerArr[1])
	if err != nil {
		fmt.Println("getCredentials#minerUrl", minerArr)
		return
	}
	MinerUrl = apiURI(minerUrl)
	MinerToken = minerArr[0]
	// FULLNODE_API_INFO
	nodeApiArr := strings.Split(env.FullApiInfo, ":")
	nodeApiUrl, err := getCredentials(nodeApiArr[1])
	if err != nil {
		fmt.Println("getCredentials#nodeApiUrl", nodeApiArr)
		return
	}
	DaemonUrl = apiURI(nodeApiUrl)
	DaemonToken = nodeApiArr[0]
	fullNode, _, err = NewLotusFullNode(context.Background(), DaemonUrl, DaemonToken)
	if err != nil {
		fmt.Println("NewLotusFullNode error")
	}
	storageMiner, _, err = NewLotusStorageMiner(context.Background(), MinerUrl, MinerToken)
	if err != nil {
		fmt.Println("NewLotusStorageMiner error")
	}



}

func getEnvPath() (*Env, error) {
	data, err := ioutil.ReadFile("/usr/local/bin/lotus-exporter-farcaster.conf")
	if err != nil {
		fmt.Println("文件读取失败", err.Error())
		return nil, err
	}
	str := string(data)
	str = strings.ReplaceAll(str, "'", "\"")
	str = strings.ReplaceAll(str, "#BEGIN GET ENV PATH", "")
	str = strings.ReplaceAll(str, "#END GET ENV PATH", "")
	var env Env
	err = json.Unmarshal([]byte(str), &env)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &env, nil
}

type Env struct {
	FullApiInfo    string `json:"FULLNODE_API_INFO"`
	MinerApiInfo   string `json:"MINER_API_INFO"`
	LotusPath      string `json:"LOTUS_PATH"`
	LotusMinerPath string `json:"LOTUS_MINER_PATH"`
}
