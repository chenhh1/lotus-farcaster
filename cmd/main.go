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
	"time"
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
	// 起始时间时间戳
	StartTime := time.Now().Unix()
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

	// 生成 MPOOL
	// GENERATE MPOOL
	mPoolPending, err := fullNode.MpoolPending(context.Background(), emptyTipSetKey)
	if err != nil {
		fmt.Println("mpoolPending error", err)
		return
	}
	fmt.Println("# HELP lotus_mpool_total return number of message pending in mpool")
	fmt.Println("# TYPE lotus_mpool_total gauge")
	fmt.Println("# HELP lotus_mpool_local_total return total number in mpool comming from local adresses")
	fmt.Println("# TYPE lotus_power_local_total gauge")
	fmt.Println("# HELP lotus_mpool_local_message local message details")
	fmt.Println("# TYPE lotus_mpool_local_message gauge")
	mpoolTotal := 0
	mpoolLocalTotal := 0
	for _, message := range mPoolPending {
		mpoolTotal += 1
		frm := message.Message.From
		for _, value := range walletList {
			if value == frm {
				mpoolLocalTotal += 1
				var displayAddr string
				if frm == minerOwnerAddr {
					displayAddr = "owner"
				} else if frm == minerWorkerAddr {
					displayAddr = "worker"
				} else if frm == minerControl0Addr {
					displayAddr = "control0"
				} else if frm != minerId {
					// displayAddr = frm[0:5]
					displayAddr = frm.String()[0:5] + "..." + frm.String()[len(frm.String()):]
				}
				fmt.Print("lotus_mpool_local_message { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", from=", `"`, displayAddr, `"`, ", to=", `"`, message.Message.To, `"`, ", nonce=", `"`, message.Message.Nonce, `"`, ", value=", `"`, message.Message.Value, `"`, ", gaslimit=", `"`, message.Message.GasLimit, `"`, ", gasfeecap=", `"`, message.Message.GasFeeCap, `"`, ", gaspremium=", `"`, message.Message.GasPremium, `"`, ", method=", `"`, message.Message.Method, " } 1", "\n")
			}
		}
	}

	fmt.Print("lotus_mpool_total { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, " } ", mpoolTotal, "\n")
	fmt.Print("lotus_mpool_local_total { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, " } ", mpoolLocalTotal, "\n")

	// 生成 WORKER 信息
	// GENERATE WORKER INFOS
	workerStats, err := storageMiner.WorkerStats(context.Background())
	if err != nil {
		fmt.Println("workerStats error", err)
		return
	}
	fmt.Println("# HELP lotus_miner_worker_mem_physical_used worker minimal memory used")
	fmt.Println("# TYPE lotus_miner_worker_mem_physical_used gauge")
	fmt.Println("# HELP lotus_miner_worker_mem_vmem_used worker maximum memory used")
	fmt.Println("# TYPE lotus_miner_worker_mem_vmem_used gauge")
	fmt.Println("# HELP lotus_miner_worker_mem_reserved worker memory reserved by lotus")
	fmt.Println("# TYPE lotus_miner_worker_mem_reserved gauge")
	fmt.Println("# HELP lotus_miner_worker_gpu_used is the GPU used by lotus")
	fmt.Println("# TYPE lotus_miner_worker_gpu_used gauge")
	fmt.Println("# HELP lotus_miner_worker_cpu_used number of CPU used by lotus")
	fmt.Println("# TYPE lotus_miner_worker_cpu_used gauge")
	fmt.Println("# HELP lotus_miner_worker_cpu number of CPU")
	fmt.Println("# TYPE lotus_miner_worker_cpu gauge")
	fmt.Println("# HELP lotus_miner_worker_gpu number of GPU")
	fmt.Println("# TYPE lotus_miner_worker_gpu gauge")
	fmt.Println("# HELP lotus_miner_worker_mem_physical server RAM")
	fmt.Println("# TYPE lotus_miner_worker_mem_physical gauge")
	fmt.Println("# HELP lotus_miner_worker_mem_swap server SWAP")
	fmt.Println("# TYPE lotus_miner_worker_mem_swap gauge")
	for _, val := range workerStats {
		Info := val.Info
		workerHost := Info.Hostname
		memPhysical := Info.Resources.MemPhysical
		memSwap := Info.Resources.MemSwap
		memReserved := Info.Resources.MemReserved
		cpus := Info.Resources.CPUs
		gpus := len(Info.Resources.GPUs)
		memUsedMin := val.MemUsedMin
		memUsedMax := val.MemUsedMax
		var gpuUsed int
		if val.GpuUsed {
			gpuUsed = 1
		} else {
			gpuUsed = 0
		}
		cpuUsed := val.CpuUse
		fmt.Print("lotus_miner_worker_cpu { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", worker_host=", `"`, workerHost, `"`, " } ", cpus, "\n")
		fmt.Print("lotus_miner_worker_gpu { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", worker_host=", `"`, workerHost, `"`, " } ", gpus, "\n")
		fmt.Print("lotus_miner_worker_mem_physical { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", worker_host=", `"`, workerHost, `"`, " } ", memPhysical, "\n")
		fmt.Print("lotus_miner_worker_mem_swap { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", worker_host=", `"`, workerHost, `"`, " } ", memSwap, "\n")
		fmt.Print("lotus_miner_worker_mem_physical_used { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", worker_host=", `"`, workerHost, `"`, " } ", memUsedMin, "\n")
		fmt.Print("lotus_miner_worker_mem_vmem_used { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", worker_host=", `"`, workerHost, `"`, " } ", memUsedMax, "\n")
		fmt.Print("lotus_miner_worker_mem_reserved { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", worker_host=", `"`, workerHost, `"`, " } ", memReserved, "\n")
		fmt.Print("lotus_miner_worker_gpu_used { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", worker_host=", `"`, workerHost, `"`, " } ", gpuUsed, "\n")
		fmt.Print("lotus_miner_worker_cpu_used { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", worker_host=", `"`, workerHost, `"`, " } ", cpuUsed, "\n")

	}

	// 生成 JOB 信息
	// GENERATE JOB INFOS
	workerJobs, err := storageMiner.WorkerJobs(context.Background())
	if err != nil {
		fmt.Println("workerJobs error", err)
		return
	}
	fmt.Println("# HELP lotus_miner_worker_job status of each individual job running on the workers. Value is the duration")
	fmt.Println("# TYPE lotus_miner_worker_job gauge")

	for wrk, jobList := range workerJobs {
		for _, job := range jobList {
			jobId := job.ID.ID
			sector := job.Sector.Number
			workerHost := workerStats[wrk].Info.Hostname
			if workerStats[wrk].Info.Hostname == "" {
				workerHost = "unknown"
			}
			task := job.Task
			jobStartTime := job.Start
			runWait := job.RunWait
			jobStartEpoch := jobStartTime.Unix()
			fmt.Print("lotus_miner_worker_job { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", job_id=", `"`, jobId, `"`, ", worker_host=", `"`, workerHost, `"`, ", task=", `"`, task, `"`, ", sector_id=", `"`, sector, `"`, ", job_start_time=", `"`, jobStartTime, `"`, ", run_wait=", `"`, runWait, `" } `, StartTime-jobStartEpoch, "\n")
		}
	}

	// 此模块 scheddiag["result"]["SchedInfo"]["Requests"] 结果为none  todo
	// GENERATE JOB SCHEDDIAG
	// scheddiag = miner_get_json("SealingSchedDiag", [True])
	// if scheddiag["result"]["SchedInfo"]["Requests"]:
	// for req in scheddiag["result"]["SchedInfo"]["Requests"]:
	// sector = req["Sector"]["Number"]
	// task = req["TaskType"]
	// print(f'lotus_miner_worker_job {{ miner_id="{miner_id}", miner_host="{miner_host}", job_id="", worker="", task="{task}", sector_id="{sector}", start="", run_wait="99" }} 0')
	// checkpoint("SchedDiag")

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

		// 	// 这段for循环暂时无法测试到　TODO
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

		// GENERATE DEADLINES
		provenPartitions, err := fullNode.StateMinerDeadlines(context.Background(), minerId, emptyTipSetKey)
		if err != nil {
			fmt.Println("provenPartitions error", err)
			return
		}
		deadlines, err := fullNode.StateMinerProvingDeadline(context.Background(), minerId, emptyTipSetKey)
		if err != nil {
			fmt.Println("deadlines error", err)
			return
		}
		dlEpoch, err := strconv.Atoi(deadlines.CurrentEpoch.String())
		if err != nil {
			fmt.Println("dlEpoch error", err)
			return
		}
		dlIndex := deadlines.Index
		dlOpen := deadlines.Open
		dlNumbers := deadlines.WPoStPeriodDeadlines
		dlWindow := deadlines.WPoStChallengeWindow
		fmt.Println("# HELP lotus_miner_deadline_info deadlines and WPoSt informations")
		fmt.Println("# TYPE lotus_miner_deadline_info gauge")
		fmt.Print("lotus_miner_deadline_info { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", current_idx=", `"`, dlIndex, `"`, ", current_epoch=", `"`, dlEpoch, `"`, ",current_open_epoch=", `"`, dlOpen, `"`, ", wpost_period_deadlines=", `"`, dlNumbers, `"`, ", wpost_challenge_window=", `"`, dlWindow, `" } 1`, "\n")
		fmt.Println("# HELP lotus_miner_deadline_active_start remaining time before deadline start")
		fmt.Println("# TYPE lotus_miner_deadline_active_start gauge")
		fmt.Println("# HELP lotus_miner_deadline_active_sectors_all number of sectors in the deadline")
		fmt.Println("# TYPE lotus_miner_deadline_active_sectors_all gauge")
		fmt.Println("# HELP lotus_miner_deadline_active_sectors_recovering number of sectors in recovering state")
		fmt.Println("# TYPE lotus_miner_deadline_active_sectors_recovering gauge")
		fmt.Println("# HELP lotus_miner_deadline_active_sectors_faulty number of faulty sectors")
		fmt.Println("# TYPE lotus_miner_deadline_active_sectors_faulty gauge")
		fmt.Println("# HELP lotus_miner_deadline_active_sectors_live number of live sectors")
		fmt.Println("# TYPE lotus_miner_deadline_active_sectors_live gauge")
		fmt.Println("# HELP lotus_miner_deadline_active_sectors_active number of active sectors")
		fmt.Println("# TYPE lotus_miner_deadline_active_sectors_active gauge")
		fmt.Println("# HELP lotus_miner_deadline_active_partitions number of partitions in the deadline")
		fmt.Println("# TYPE lotus_miner_deadline_active_partitions gauge")
		fmt.Println("# HELP lotus_miner_deadline_active_partitions_proven number of partitions already proven for the deadline")
		fmt.Println("# TYPE lotus_miner_deadline_active_partitions_proven gauge")
		for i := 0; i < int(dlNumbers); i++ {
			idx := (int(dlIndex) + i) % int(dlNumbers)
			opened := int(dlOpen) + int(dlWindow)*i
			partitions, err := fullNode.StateMinerPartitions(context.Background(), minerId, uint64(idx), emptyTipSetKey)
			if err != nil {
				fmt.Println("partitions error", err)
				return
			}
			if partitions != nil {
				faulty := 0
				recovering := 0
				alls := 0
				active := 0
				live := 0
				count := len(partitions)
				proven, err := provenPartitions[idx].PostSubmissions.Count()
				if err != nil {
					fmt.Println("proven error ", err)
					return
				}

				for _, partition := range partitions {
					a, err := partition.FaultySectors.Count()
					if err != nil {
						fmt.Println("proven error ", err)
						return
					}
					faulty += int(a)

					b, err := partition.RecoveringSectors.Count()
					if err != nil {
						fmt.Println("proven error ", err)
						return
					}
					recovering += int(b)

					c, err := partition.ActiveSectors.Count()
					if err != nil {
						fmt.Println("proven error ", err)
						return
					}
					active += int(c)

					d, err := partition.LiveSectors.Count()
					if err != nil {
						fmt.Println("proven error ", err)
						return
					}
					live += int(d)

					e, err := partition.AllSectors.Count()
					if err != nil {
						fmt.Println("proven error ", err)
						return
					}
					alls = int(e)

				}
				fmt.Print("lotus_miner_deadline_active_start { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", index=", `"`, idx, `"`, " } ", (opened-dlEpoch)*30, "\n")
				fmt.Print("lotus_miner_deadline_active_partitions_proven { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", index=", `"`, idx, `"`, " } ", proven, "\n")
				fmt.Print("lotus_miner_deadline_active_partitions { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", index=", `"`, idx, `"`, " } ", count, "\n")
				fmt.Print("lotus_miner_deadline_active_sectors_all { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", index=", `"`, idx, `"`, " } ", alls, "\n")
				fmt.Print("lotus_miner_deadline_active_sectors_recovering { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", index=", `"`, idx, `"`, " } ", recovering, "\n")
				fmt.Print("lotus_miner_deadline_active_sectors_faulty { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", index=", `"`, idx, `"`, " } ", faulty, "\n")
				fmt.Print("lotus_miner_deadline_active_sectors_active { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", index=", `"`, idx, `"`, " } ", active, "\n")
				fmt.Print("lotus_miner_deadline_active_sectors_live { miner_id=", `"`, minerId, `"`, ", miner_host=", `"`, minerHost, `"`, ", index=", `"`, idx, `"`, " } ", live, "\n")
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
