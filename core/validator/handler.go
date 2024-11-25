package validator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"grid-prover/core/types"
	"grid-prover/database"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (v *GRIDValidator) LoadValidatorModule(g *gin.RouterGroup) {
	g.GET("/rnd", v.GetRNDHandler)
	g.GET("/withdraw/signature", v.GetWithdrawSignatureHandler)
	g.POST("/proof", v.SubmitProofHandler)
	fmt.Println("load light node moudle success!")
}

func (v *GRIDValidator) GetRNDHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"rnd": hex.EncodeToString(RND[:]),
	})
}

// func GetSettingInfo(c *gin.Context) {

// }

func (v *GRIDValidator) SubmitProofHandler(c *gin.Context) {
	var proof types.Proof
	err := c.BindJSON(&proof)
	if err != nil {
		logger.Error(err)
		c.AbortWithStatusJSON(500, err.Error())
		return
	}

	if !v.IsProveTime() {
		logger.Error("Failure to submit proof within the proof time")
		c.AbortWithStatusJSON(400, "Failure to submit proof within the proof time")
		return
	}

	hash := sha256.New()
	hash.Write(RND[:])
	hash.Write(proof.ToBytes())
	result := hash.Sum(nil)

	diffcult, err := getDiffcultByProviderId(proof.NodeID)
	if err != nil {
		logger.Error(err)
		c.AbortWithStatusJSON(500, err.Error())
		return
	}

	if !checkPOWResult(result, diffcult) {
		logger.Error("Verify Proof Failed:", hex.EncodeToString(result))
		c.AbortWithStatusJSON(400, "Verify Proof Failed")
		return
	}

	resultChan <- types.Result{
		NodeID:  proof.NodeID,
		Success: true,
	}

	c.JSON(http.StatusOK, "Verify Proof Success")
}

func (v *GRIDValidator) GetProfitInfo(c *gin.Context) {
	address := c.Query("address")
	if len(address) == 0 {
		logger.Error("field address or amount is not set")
		c.AbortWithStatusJSON(400, "field address is not set")
		return
	}

	profit, err := database.GetProfitByAddress("address")
	if err != nil {
		logger.Error(err.Error())
		c.AbortWithStatusJSON(400, err.Error())
		return
	}

	c.JSON(200, profit)
}

func (v *GRIDValidator) GetWithdrawSignatureHandler(c *gin.Context) {
	address := c.Query("address")
	amount := c.Query("amount")
	if len(address) == 0 || len(amount) == 0 {
		logger.Error("field address or amount is not set")
		c.AbortWithStatusJSON(400, "field address or amount is not set")
		return
	}

	amountBig, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		logger.Error("field amount is not a decimal number")
		c.AbortWithStatusJSON(400, "field amount is not a decimal number")
		return
	}

	signature, err := v.GenerateWithdrawSignature(address, amountBig)
	if err != nil {
		logger.Error(err.Error())
		c.AbortWithStatusJSON(500, err.Error())
		return
	}

	c.JSON(200, hex.EncodeToString(signature))

}

func getDiffcultByProviderId(nodeID types.NodeID) (int, error) {
	return 8, nil
}

func checkPOWResult(hash []byte, diffcult int) bool {
	if diffcult >= 256 {
		return false
	}

	n := diffcult / 8
	var remain byte = 0xff ^ (0xff >> (diffcult % 8))

	for i := 0; i < n; i++ {
		if hash[i] != 0 {
			return false
		}
	}

	if hash[n]&remain != 0 {
		return false
	}

	return true
}
