package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
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
	"strconv"
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
	fmt.Print("lotus_chain_height { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, " } ", chainHead.Height(), "\n")

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
		// 大整数     原值是:bigInt  -->  int  -->  bigFloat  -->   Float64
		fBalance := new(big.Float).SetInt(balance.Int)
		afterBalance, _ := fBalance.Float64()
		fmt.Print("lotus_wallet_balance { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", address=", `"`, addr, `"`, ", short=", `"`, short, `"`, " } ", afterBalance/1000000000000000000.0, "\n")

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
	fmt.Print("lotus_wallet_balance { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", address=", `"`, minerId, `"`, ", short=", `"`, minerId, `"`, " } ", afterBalance/1000000000000000000.0, "\n")

	// 生成矿工信息
	// GENERATE MINER INFO
	minerVersion, err := storageMiner.Version(context.Background())
	if err != nil {
		fmt.Println("minerVersion error", err)
		return
	}
	// 检索主要地址
	// RETRIEVE MAIN ADDRESSES
	daemonStats, err := fullNode.StateMinerInfo(context.Background(), minerId, emptyTipSetKey)
	if err != nil {
		fmt.Println("daemonStats error", err)
		return
	}
	minerOwner := daemonStats.Owner
	minerOwnerAddr, err := fullNode.StateAccountKey(context.Background(), minerOwner, emptyTipSetKey)
	if err != nil {
		fmt.Println("minerOwnerAddr error", err)
		return
	}

	minerWorker := daemonStats.Worker
	minerWorkerAddr, err := fullNode.StateAccountKey(context.Background(), minerWorker, emptyTipSetKey)
	if err != nil {
		fmt.Println("minerWorkerAddr error", err)
		return
	}
	var minerControl0 address.Address
	if daemonStats.ControlAddresses != nil {
		minerControl0 = daemonStats.ControlAddresses[0]
	} else {
		minerControl0 = minerWorker
	}
	minerControl0Addr, err := fullNode.StateAccountKey(context.Background(), minerControl0, emptyTipSetKey)
	if err != nil {
		fmt.Println("minerControl0Addr error", err)
		return
	}
	fmt.Println("# HELP lotus_miner_info lotus miner information like adress version etc")
	fmt.Println("# TYPE lotus_miner_info gauge")
	fmt.Println("# HELP lotus_miner_info_sector_size lotus miner sector size")
	fmt.Println("# TYPE lotus_miner_info_sector_size gauge")
	fmt.Print("lotus_miner_info { miner_id = ", `"`, minerId, `"`, ", miner_host = ", `"`, minerHost, `"`, ", version=", `"`, minerVersion.Version, `"`, ", owner=", `"`, minerOwner, `"`, ", owner_addr=", `"`, minerOwnerAddr, `"`, ", worker=", `"`, minerWorker, `"`, ", worker_addr=", `"`, minerWorkerAddr, `"`, ", control0=", `"`, minerControl0, `"`, ", control0_addr=", `"`, minerControl0Addr, `"`, " } 1", "\n")
	fmt.Print("lotus_miner_info_sector_size { miner_id = ", `"`, minerId, `"`, " } ", daemonStats.SectorSize, "\n")

	// 生成daemon信息
	// GENERATE DAEMON INFO
	daemonNetwork, err := fullNode.StateNetworkName(context.Background())
	if err != nil {
		fmt.Println("daemonNetwork error", err)
		return
	}
	daemonNetworkVersion, err := fullNode.StateNetworkVersion(context.Background(), emptyTipSetKey)
	if err != nil {
		fmt.Println("daemonNetworkVersion error", err)
		return
	}
	daemonVersion, err := fullNode.Version(context.Background())
	if err != nil {
		fmt.Println("daemonVersion error", err)
		return
	}
	fmt.Println("# HELP lotus_info lotus daemon information like adress version, value is set to network version number")
	fmt.Println("# TYPE lotus_info gauge")
	fmt.Print("lotus_info { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", version=", `"`, daemonVersion.Version, `"`, ", network=", `"`, daemonNetwork, `"`, " } ", daemonNetworkVersion, "\n")

	// 生成  SECTORS
	// GENERATE SECTORS
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
		packedDate := ""
		finalizedDate := ""
		verifiedWeight := detail.VerifiedDealWeight
		for i := 0; i < len(detail.Log); i++ {
			if detail.Log[i].Kind == "event;sealing.SectorPacked" {
				packedDate = strconv.Itoa(int(detail.Log[i].Timestamp))
			}
			if detail.Log[i].Kind == "event;sealing.SectorFinalized" {
				finalizedDate = strconv.Itoa(int(detail.Log[i].Timestamp))
			}
		}
		var pledged int
		if detail.Log[0].Kind == "event;sealing.SectorStartCC" {
			pledged = 1
		} else {
			pledged = 0
		}
		fmt.Print("lotus_miner_sector_state { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", sector_id=", `"`, sector, `"`, ", state=", `"`, detail.State, `"`, ", pledged=", `"`, pledged, `"`, ", deals=", `"`, deals, `"`, ", verified_weight=", `"`, verifiedWeight, `"`, " } 1\n")

		if packedDate != "" {
			fmt.Print("lotus_miner_sector_event { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", sector_id=", `"`, sector, `", event_type="packed" } `, packedDate, "\n")
		}
		if string(creationDate) != "" {
			fmt.Print("lotus_miner_sector_event { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", sector_id=", `"`, sector, `", event_type="creation" } `, creationDate, "\n")
		}
		if finalizedDate != "" {
			fmt.Print("lotus_miner_sector_event { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", sector_id=", `"`, sector, `", event_type="finalized" } `, finalizedDate, "\n")
		}

		// 这段for循环暂时无法测试到　TODO
		list1 := [2]string{"Proving", "Removed"}
		for _, j := range list1 {
			if string(detail.State) != j {
				for _, deal := range detail.Deals {
					if deal != 0 {
						var dealIsVerified, dealSize, dealSlashEpoch, dealPricePerEpoch, dealProviderCollateral, dealClientCollateral, dealStartEpoch, dealEndEpoch string
						dealInfo, err := fullNode.StateMarketStorageDeal(context.Background(), deal, emptyTipSetKey)
						if err != nil {
							dealIsVerified = "unknown"
							dealSize = "unknown"
							dealSlashEpoch = "unknown"
							dealPricePerEpoch = "unknown"
							dealProviderCollateral = "unknown"
							dealClientCollateral = "unknown"
							dealStartEpoch = "unknown"
							dealEndEpoch = "unknown"
						} else {
							dealIsVerified = strconv.FormatBool(dealInfo.Proposal.VerifiedDeal)
							dealSize = string(dealInfo.Proposal.PieceSize)
							dealSlashEpoch = string(dealInfo.State.SlashEpoch)
							dealPricePerEpoch = dealInfo.Proposal.StoragePricePerEpoch.String()
							dealProviderCollateral = dealInfo.Proposal.ProviderCollateral.String()
							dealClientCollateral = dealInfo.Proposal.ClientCollateral.String()
							dealStartEpoch = string(dealInfo.Proposal.StartEpoch)
							dealEndEpoch = string(dealInfo.Proposal.EndEpoch)
						}
						fmt.Print("lotus_miner_sector_sealing_deals_size { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", sector_id=", `"`, sector, `"`, ", deal_id=", `"`, deal, `"`, ", deal_is_verified=", `"`, dealIsVerified, `"`, ", deal_slash_epoch=", `"`, dealSlashEpoch, `"`, ", deal_price_per_epoch=", `"`, dealPricePerEpoch, `"`, ",deal_provider_collateral=", `"`, dealProviderCollateral, `"`, ", deal_client_collateral=", `"`, dealClientCollateral, `"`, ", deal_size=", `"`, dealSize, `"`, ", deal_start_epoch=", `"`, dealStartEpoch, `"`, ", deal_end_epoch=", `"`, dealEndEpoch, `"`, " } 1\n")
					}
				}
			}
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
