package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"io/ioutil"
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
	// 生成  SECTORS
	// GENERATE SECTORS
	fmt.Println("# ")
	fmt.Println("# HELP lotus_miner_sector_state sector state")
	fmt.Println("# TYPE lotus_miner_sector_state gauge")
	fmt.Println("# HELP lotus_miner_sector_event contains important event of the sector life")
	fmt.Println("# TYPE lotus_miner_sector_event gauge")
	fmt.Println("# HELP lotus_miner_sector_sealing_deals_info contains information related to deals that are not in Proving and Removed state.")
	fmt.Println("# TYPE lotus_miner_sector_sealing_deals_info gauge")
	sectorList, err := storageMiner.SectorsList(context.Background())
	if err != nil {
		fmt.Println("sectorList error", err)
		return
	}
	for _, sector := range sectorList {
		detail, err := storageMiner.SectorsStatus(context.Background(), sector, false)
		if err != nil {
			fmt.Println("sectorList error", err)
			return
		}
		// 计算 0 出现在数组中的个数
		a := 0
		for _, j := range detail.Deals {
			if j == 0 {
				a++
			}
		}
		deals := len(detail.Deals) - a
		creationDate := detail.Log[0].Timestamp
		verifiedWeight := detail.VerifiedDealWeight
		var pledged int
		if detail.Log[0].Kind == "event;sealing.SectorStartCC" {
			pledged = 1
		} else {
			pledged = 0
		}
		fmt.Print("lotus_miner_sector_state { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", sector_id=", `"`, sector, `"`, ", state=", `"`, detail.State, `"`, ", pledged=", `"`, pledged, `"`, ", deals=", `"`, deals, `"`, ", verified_weight=", `"`, verifiedWeight, `"`, "} 1", "\n")

		if string(creationDate) != "" {
			fmt.Print("lotus_miner_sector_event { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", sector_id=", `"`, sector, `"`, ", event_type=\"creation\" } ", creationDate, "\n")
		}
	}
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
	fmt.Println("MinerUrl:", MinerUrl, "MinerToken:", MinerToken)
	// FULLNODE_API_INFO
	nodeApiArr := strings.Split(env.FullApiInfo, ":")
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
	fmt.Println(str)
	var env Env
	err = json.Unmarshal([]byte(str), &env)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println(env.FullApiInfo)
	fmt.Println(env.MinerApiInfo)
	fmt.Println(env.LotusPath)
	fmt.Println(env.LotusMinerPath)
	return &env, nil
}

type Env struct {
	FullApiInfo    string `json:"FULLNODE_API_INFO"`
	MinerApiInfo   string `json:"MINER_API_INFO"`
	LotusPath      string `json:"LOTUS_PATH"`
	LotusMinerPath string `json:"LOTUS_MINER_PATH"`
}
