package validator

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"math/big"
	"math/rand"
	"time"

	"github.com/gridprotocol/dumper/database"
	"github.com/gridprotocol/validator/core/types"
	"github.com/gridprotocol/validator/logs"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var logger = logs.Logger("grid validator")

// var proofChan = make(chan Proof, 100)
var resultChan = make(chan types.Result, 100)
var RND [32]byte

type GRIDValidator struct {
	last            int64
	prepareInterval time.Duration
	proveInterval   time.Duration
	waitInterval    time.Duration

	sk *ecdsa.PrivateKey

	done  chan struct{}
	doned bool
}

func NewGRIDValidator(chain string, sk *ecdsa.PrivateKey) (*GRIDValidator, error) {
	// get time information from contract
	prepareInterval := 10 * time.Second
	proveInterval := 10 * time.Second
	waitInterval := 2*time.Minute - prepareInterval - proveInterval
	return &GRIDValidator{
		last:            0,
		prepareInterval: prepareInterval,
		proveInterval:   proveInterval,
		waitInterval:    waitInterval,

		sk: sk,

		done:  make(chan struct{}),
		doned: false,
	}, nil
}

func (v *GRIDValidator) Start(ctx context.Context) {
	for {
		// 等待下一个prepare时期
		wait, nextTime := v.CalculateWatingToPrepare()
		select {
		case <-ctx.Done():
			v.doned = true
			return
		case <-v.done:
			v.doned = true
			return
		case <-time.After(wait):
		}

		// generate a random value
		err := v.GenerateRND(ctx)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		// 等待下一个prove时期
		wait, _ = v.CalculateWatingToProve()
		select {
		case <-ctx.Done():
			return
		case <-v.done:
			v.doned = true
			return
		case <-time.After(wait):
		}

		// get nodes list with order
		resultMap, err := v.GetChallengeNode(ctx)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		// receive succeeded proof from chan and set resultMap
		res, err := v.HandleResult(ctx, resultMap)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		logger.Info("Start update profits")

		// add penalty for each failed proof
		err = v.AddPenalty(ctx, res)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		v.last = nextTime
	}
}

// receive proof from chan and set resultMap
func (v *GRIDValidator) HandleResult(ctx context.Context, resultMap map[types.NodeID]bool) (map[types.NodeID]bool, error) {
	var channel = make(chan struct{})

	logger.Info("start handle result")

	// send quit signal
	go func() {
		select {
		case <-ctx.Done():
			channel <- struct{}{}
			return
		case <-v.done:
			channel <- struct{}{}
			return
		case <-time.After(v.proveInterval):
			channel <- struct{}{}
			return
		}
	}()

	for {
		select {
		// quit
		case <-channel:
			logger.Info("end handle result")
			return resultMap, nil

		// receive succeeded proof result
		case result := <-resultChan:
			if _, ok := resultMap[result.NodeID]; ok {
				resultMap[result.NodeID] = true
			}
		}
	}

}

// add penalty for each failed proof, update profit info in db
func (v *GRIDValidator) AddPenalty(ctx context.Context, res map[types.NodeID]bool) error {
	for nodeID, result := range res {

		// get profit from db
		profitInfo, err := database.GetProfitByAddress(nodeID.Provider)
		if err != nil {
			return err
		}

		// calc reward
		var reward = new(big.Int)
		if v.last <= profitInfo.LastTime.Unix() {
			reward.SetInt64(0)
		} else if v.last >= profitInfo.EndTime.Unix() {
			reward.Set(profitInfo.Profit)
		} else if profitInfo.LastTime.Unix() >= profitInfo.EndTime.Unix() {
			reward.SetInt64(0)
		} else {
			reward.Mul(profitInfo.Profit, big.NewInt((v.last-profitInfo.LastTime.Unix())/(profitInfo.EndTime.Unix()-profitInfo.LastTime.Unix())))
		}

		// remain := profitInfo.Profit - reward
		remain := new(big.Int).Sub(profitInfo.Profit, reward)

		// calc penalty if proof failed, 1% of remain per failure proof
		var penalty = big.NewInt(0)
		if !result {
			// penalty = remain / 100
			penalty.Div(remain, big.NewInt(100))
		}

		profitInfo.LastTime = time.Unix(v.last, 0)
		profitInfo.Balance.Add(profitInfo.Balance, reward)
		profitInfo.Profit.Sub(remain, penalty)
		profitInfo.Penalty.Add(profitInfo.Penalty, penalty)

		logger.Debugf("Balance: %d, Profit: %d, penalty: %d", profitInfo.Balance, profitInfo.Profit, profitInfo.Penalty)

		// update profit into db
		err = profitInfo.UpdateProfit()
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *GRIDValidator) Stop() {
	close(v.done)

	for !v.doned {
		time.Sleep(200 * time.Millisecond)
	}
}

func (v *GRIDValidator) IsProveTime() bool {
	challengeCycleSeconds := int64((v.prepareInterval + v.proveInterval + v.waitInterval).Seconds())
	now := time.Now().Unix()
	duration := now - v.last
	over := duration % challengeCycleSeconds
	if over >= int64(v.prepareInterval.Seconds()) && over <= int64((v.prepareInterval+v.proveInterval).Seconds()) {
		return true
	}

	return false
}

// wait time
func (v *GRIDValidator) CalculateWatingToPrepare() (time.Duration, int64) {
	challengeCycleSeconds := int64((v.prepareInterval + v.proveInterval + v.waitInterval).Seconds())
	now := time.Now().Unix()
	duration := now - v.last
	over := duration % challengeCycleSeconds
	var waitingSeconds int64 = 0
	if over >= int64(v.prepareInterval.Seconds()) {
		waitingSeconds = challengeCycleSeconds - over
	}

	v.last = now - over
	next := v.last + challengeCycleSeconds

	return time.Duration(waitingSeconds) * time.Second, next
}

// wait time
func (v *GRIDValidator) CalculateWatingToProve() (time.Duration, int64) {
	challengeCycleSeconds := int64((v.prepareInterval + v.proveInterval + v.waitInterval).Seconds())
	now := time.Now().Unix()
	duration := now - v.last
	over := duration % challengeCycleSeconds
	var waitingSeconds int64 = 0
	if over < int64(v.prepareInterval.Seconds()) {
		waitingSeconds = int64(v.prepareInterval.Seconds()) - over
		v.last = now - over
	} else if over > int64((v.prepareInterval + v.proveInterval).Seconds()) {
		waitingSeconds = challengeCycleSeconds + int64(v.prepareInterval.Seconds()) - over
		v.last = now - over + challengeCycleSeconds
	}

	next := v.last + challengeCycleSeconds

	return time.Duration(waitingSeconds) * time.Second, next
}

// random value
func (v *GRIDValidator) GenerateRND(ctx context.Context) error {
	// TODO: call the contract
	for index := range RND {
		RND[index] = byte(rand.Int())
	}

	return nil
}

// get all nodes with order
func (v *GRIDValidator) GetChallengeNode(ctx context.Context) (map[types.NodeID]bool, error) {
	orders, err := database.ListAllActivedOrder()
	if err != nil {
		return nil, err
	}

	// record proof result
	resultMap := make(map[types.NodeID]bool)

	// init
	for _, order := range orders {
		resultMap[types.NodeID{
			Provider: order.Provider,
			ID:       order.Id,
		}] = false
	}

	return resultMap, nil
}

// generate signature
func (v *GRIDValidator) GenerateWithdrawSignature(address string, amount *big.Int) ([]byte, error) {
	profit, err := database.GetProfitByAddress(address)
	if err != nil {
		return nil, err
	}

	// get nonce
	var nonceBuf = make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBuf, profit.Nonce)

	// get hash
	hash := crypto.Keccak256(common.FromHex(address), amount.Bytes(), nonceBuf)

	// sign
	return crypto.Sign(hash, v.sk)
}
