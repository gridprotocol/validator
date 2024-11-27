package core

import (
	"context"
	"grid-prover/database"
	"grid-prover/logs"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// blockNumber = big.NewInt(0)
	logger = logs.Logger("dumper")
)

type Dumper struct {
	endpoint        string
	contractABI     []abi.ABI
	contractAddress []common.Address
	// store           MapStore

	blockNumber *big.Int

	eventNameMap map[common.Hash]string
	indexedMap   map[common.Hash]abi.Arguments
}

func getEndpointByChain(chain string) string {
	switch chain {
	case "dev":
		//return "https://devchain.metamemo.one:8501" // external ip for dev chain
		return "http://10.10.100.82:8201" // internal ip for dev chain, m21 use only
	case "test":
		return "https://testchain.metamemo.one:24180"
	case "product":
		return "https://chain.metamemo.one:8501"
	case "goerli":
		return "https://eth-goerli.g.alchemy.com/v2/Bn3AbuwyuTWanFLJiflS-dc23r1Re_Af"
	}
	return "https://devchain.metamemo.one:8501"
}

func NewGRIDDumper(chain string, registerAddress, marketAddress common.Address) (dumper *Dumper, err error) {
	dumper = &Dumper{
		// store:        store,
		endpoint:     getEndpointByChain(chain),
		eventNameMap: make(map[common.Hash]string),
		indexedMap:   make(map[common.Hash]abi.Arguments),
	}

	dumper.contractAddress = []common.Address{registerAddress, marketAddress}

	registerABI, err := abi.JSON(strings.NewReader(RegisterABI))
	if err != nil {
		return dumper, err
	}

	marketABI, err := abi.JSON(strings.NewReader(MarketABI))
	if err != nil {
		return dumper, err
	}

	dumper.contractABI = []abi.ABI{registerABI, marketABI}

	for _, ABI := range dumper.contractABI {
		for name, event := range ABI.Events {
			dumper.eventNameMap[event.ID] = name
			var indexed abi.Arguments
			for _, arg := range ABI.Events[name].Inputs {
				if arg.Indexed {
					indexed = append(indexed, arg)
				}
			}
			dumper.indexedMap[event.ID] = indexed
		}
	}

	blockNumber, err := database.GetBlockNumber()
	if err != nil {
		blockNumber = 0
	}
	dumper.blockNumber = big.NewInt(blockNumber)

	return dumper, nil
}

func (d *Dumper) SubscribeGRID(ctx context.Context) {
	for {
		d.DumpGRID()

		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

func (d *Dumper) DumpGRID() error {
	client, err := ethclient.DialContext(context.TODO(), d.endpoint)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	defer client.Close()

	events, err := client.FilterLogs(context.TODO(), ethereum.FilterQuery{
		FromBlock: d.blockNumber,
		Addresses: d.contractAddress,
	})
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	lastBlockNumber := d.blockNumber

	for _, event := range events {
		eventName, ok1 := d.eventNameMap[event.Topics[0]]
		if !ok1 {
			continue
		}
		switch eventName {
		case "Register":
			logger.Info("Handle Register Event")
			err = d.HandleRegister(event)
		case "AddNode":
			logger.Info("Handle Add Node Event")
			err = d.HandleAddNode(event)
		case "CreateOrder":
			logger.Info("==== Handle Create Order Event")
			tx, _, err := client.TransactionByHash(context.TODO(), event.TxHash)
			if err != nil {
				logger.Debug("handle create order error: ", err.Error())
			}

			// get user address
			user, err := types.LatestSignerForChainID(tx.ChainId()).Sender(tx)
			if err != nil {
				logger.Debug(err.Error())
				return err
			}

			// store order info
			err = d.HandleCreateOrder(event, user)
			if err != nil {
				logger.Debug(err.Error())
			}
		// case "Withdraw":
		// 	logger.Info("Handle Withdraw Event")
		// 	err = d.HandleCreateOrder(event)
		default:
			continue
		}
		if err != nil {
			logger.Error(err.Error())
			break
		}

		logger.Info(event.BlockNumber, d.blockNumber.Uint64())
		if event.BlockNumber >= d.blockNumber.Uint64() {
			d.blockNumber = big.NewInt(int64(event.BlockNumber) + 1)
		}
	}

	if d.blockNumber.Cmp(lastBlockNumber) == 1 {
		database.SetBlockNumber(d.blockNumber.Int64())
	}

	return nil
}

// func (d *Dumper) unpack(log types.Log, ABI abi.ABI, out interface{}) error {
// 	eventName := d.eventNameMap[log.Topics[0]]
// 	indexed := d.indexedMap[log.Topics[0]]

// 	logger.Info(log.Topics)

// 	logger.Info(eventName)
// 	logger.Info(indexed)

// 	err := ABI.UnpackIntoInterface(out, eventName, log.Data)
// 	if err != nil {
// 		return err
// 	}
// 	logger.Info(out)

// 	return abi.ParseTopics(out, indexed, log.Topics[1:])
// }

// unpack a log
func (d *Dumper) unpack(log types.Log, ABI abi.ABI, out interface{}) error {
	// get event name from map with hash
	eventName := d.eventNameMap[log.Topics[0]]
	// get all topics
	indexed := d.indexedMap[log.Topics[0]]

	logger.Infof("log data: %x", log.Data)
	logger.Info("log topics: ", log.Topics)

	// logger.Info("event name: ", eventName)
	logger.Info("topics in map: ", indexed)

	// parse data
	logger.Info("parse data, event name: ", eventName)
	err := ABI.UnpackIntoInterface(out, eventName, log.Data)
	if err != nil {
		return err
	}
	logger.Info("unpack out(no topics):", out)

	// parse topic
	logger.Info("parse topic")
	err = abi.ParseTopics(out, indexed, log.Topics[1:])
	if err != nil {
		return err
	}
	logger.Info("unpack out(with topics):", out)

	return nil
}

type RegisterEvent struct {
	Cp     common.Address
	Name   string
	Ip     string
	Domain string
	Port   string
}

func (d *Dumper) HandleRegister(log types.Log) error {
	var out RegisterEvent
	err := d.unpack(log, d.contractABI[0], &out)
	if err != nil {
		return err
	}

	providerInfo := database.Provider{
		Address: out.Cp.Hex(),
		Name:    out.Name,
		IP:      out.Ip,
		Domain:  out.Domain,
		Port:    out.Port,
	}

	err = providerInfo.CreateProvider()
	if err != nil {
		return err
	}

	now := time.Now()
	profitInfo := database.Profit{
		Address:  out.Cp.Hex(),
		Balance:  big.NewInt(0),
		Profit:   big.NewInt(0),
		Penalty:  big.NewInt(0),
		LastTime: now,
		EndTime:  now,
	}
	return profitInfo.CreateProfit()
}

type AddNodeEvent struct {
	Cp common.Address
	Id uint64

	Cpu struct {
		CpuPriceMon *big.Int
		CpuPriceSec *big.Int
		Model       string
	}

	Gpu struct {
		GpuPriceMon *big.Int
		GpuPriceSec *big.Int
		Model       string
	}

	Mem struct {
		MemPriceMon *big.Int
		MemPriceSec *big.Int
		Num         uint64
	}

	Disk struct {
		DiskPriceMon *big.Int
		DiskPriceSec *big.Int
		Num          uint64
	}
}

func (d *Dumper) HandleAddNode(log types.Log) error {
	var out AddNodeEvent
	err := d.unpack(log, d.contractABI[0], &out)
	if err != nil {
		return err
	}

	nodeInfo := database.Node{
		Address:  out.Cp.Hex(),
		Id:       int(out.Id),
		CPUPrice: out.Cpu.CpuPriceSec,
		CPUModel: out.Cpu.Model,

		GPUPrice: out.Gpu.GpuPriceSec,
		GPUModel: out.Gpu.Model,

		MemPrice:    out.Mem.MemPriceSec,
		MemCapacity: int64(out.Mem.Num),

		DiskPrice:    out.Disk.DiskPriceSec,
		DiskCapacity: int64(out.Disk.Num),
	}

	return nodeInfo.CreateNode()
}

type CreateOrderEvent struct {
	Cp  common.Address
	Id  uint64
	Nid uint64
	Act *big.Int
	Pro *big.Int
	Dur *big.Int
}

func (d *Dumper) HandleCreateOrder(log types.Log, from common.Address) error {
	var out CreateOrderEvent

	// abi1 = market
	err := d.unpack(log, d.contractABI[1], &out)
	if err != nil {
		return err
	}

	startTime := out.Act.Add(out.Act, out.Pro)
	endTime := startTime.Add(startTime, out.Dur)
	orderInfo := database.Order{
		User:         from.Hex(),
		Provider:     out.Cp.Hex(),
		Id:           out.Id,
		Nid:          out.Nid,
		ActivateTime: time.Unix(out.Act.Int64(), 0),
		StartTime:    time.Unix(startTime.Int64(), 0),
		EndTime:      time.Unix(endTime.Int64(), 0),
		Probation:    out.Pro.Int64(),
		Duration:     out.Dur.Int64(),
	}

	logger.Info("store create order..")
	err = orderInfo.CreateOrder()
	if err != nil {
		logger.Debug("store create order error: ", err.Error())
		return err
	}

	return nil
}

type WithdrawEvent struct {
	Cp     common.Address
	Amount *big.Int
}

func (d *Dumper) HandleWithdraw(log types.Log) error {
	var out WithdrawEvent
	err := d.unpack(log, d.contractABI[1], &out)
	if err != nil {
		return err
	}

	profit, err := database.GetProfitByAddress(out.Cp.Hex())
	if err != nil {
		return err
	}

	profit.Balance.Sub(profit.Balance, out.Amount)
	profit.Nonce++
	return profit.UpdateProfit()
}
