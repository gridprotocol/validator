package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gridprotocol/validator/core/validator"

	"github.com/gridprotocol/dumper/database"
	"github.com/gridprotocol/dumper/dumper"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"

	"github.com/grid/contracts/eth"
)

//var provider1 = "0x867F691B053B61490F8eB74c2df63745CfC0A973"

var ValidatorCmd = &cli.Command{
	Name:  "validator",
	Usage: "grid validator node",
	Subcommands: []*cli.Command{
		// validatorNodeRunCmd,
		runCmd,
	},
}

// run validator with sk
var runCmd = &cli.Command{
	Name:  "run",
	Usage: "run meeda store node",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "endpoint",
			Aliases: []string{"e"},
			Usage:   "input your endpoint",
			Value:   ":8081",
		},
		&cli.StringFlag{
			Name:  "sk",
			Usage: "input your private key",
			Value: "5087077ba322c5bd02f95ed8b50ff9251b8f1d165455d0688c75f5d4740a19f4", // test validator sk
		},
		&cli.StringFlag{
			Name:  "chain",
			Usage: "input chain name, e.g.(dev)",
			Value: "dev",
		},
	},
	Action: func(ctx *cli.Context) error {
		endPoint := ctx.String("endpoint")
		sk := ctx.String("sk")
		chain := ctx.String("chain")

		privateKey, err := crypto.HexToECDSA(sk)
		if err != nil {
			privateKey, err = crypto.GenerateKey()
			if err != nil {
				return err
			}
		}

		err = database.InitDatabase("~/grid")
		if err != nil {
			return err
		}

		// contract address
		registryAddress := common.HexToAddress("0x10fd5Eb0A59398796aA6C368CF0562135C3e4c32")
		marketAddress := common.HexToAddress("0xd43241c35E49158B61aD5c061b2d050D276f9E94")

		fmt.Println("registry: ", registryAddress)
		fmt.Println("market: ", marketAddress)

		// new dumper
		dumper, err := dumper.NewGRIDDumper(getEndpointByChain(chain), registryAddress, marketAddress)
		if err != nil {
			return err
		}

		// generate db
		err = dumper.DumpGRID()
		if err != nil {
			return err
		}
		// sync db with chain
		go dumper.SubscribeGRID(context.TODO())

		// new validator
		validator, err := validator.NewGRIDValidator(chain, privateKey)
		if err != nil {
			return err
		}
		// validate all nodes every 2 hours
		go validator.Start(context.TODO())

		// new validator server
		server, err := NewValidatorServer(validator, endPoint)
		if err != nil {
			return err
		}

		// start server listen
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("Shutting down server...")

		if err := server.Shutdown(context.TODO()); err != nil {
			log.Fatal("Server forced to shutdown: ", err)
		}

		validator.Stop()
		log.Println("Server exiting")

		return nil
	},
}

// new gin server, register route
func NewValidatorServer(validator *validator.GRIDValidator, endpoint string) (*http.Server, error) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome GRID Validator Node")
	})

	// register all route
	validator.LoadValidatorModule(router.Group("/v1"))

	return &http.Server{
		Addr:    endpoint,
		Handler: router,
	}, nil
}

func getEndpointByChain(chain string) string {
	switch chain {
	case "local":
		return eth.Ganache
	case "dev":
		return "https://devchain.metamemo.one:8501"
		//return "http://10.10.100.82:8201"

		// case "test":
		// 	return "https://testchain.metamemo.one:24180"
		// case "product":
		// 	return "https://chain.metamemo.one:8501"
		// case "goerli":
		// 	return "https://eth-goerli.g.alchemy.com/v2/Bn3AbuwyuTWanFLJiflS-dc23r1Re_Af"
	}
	return "https://devchain.metamemo.one:8501"
}

// func InitTestDataBase(path string) error {
// 	err := database.RemoveDataBase(path)
// 	if err != nil {
// 		return err
// 	}

// 	err = database.InitDatabase(path)
// 	if err != nil {
// 		return err
// 	}

// 	provider := database.Provider{
// 		Address: provider1,
// 		Name:    "test",
// 		IP:      "127.0.0.1",
// 		Port:    "40",
// 	}
// 	err = provider.CreateProvider()
// 	if err != nil {
// 		return err
// 	}

// 	node := database.Node{
// 		Address:  provider1,
// 		Id:       1,
// 		CPUPrice: big.NewInt(10),
// 		CPUModel: "AMD 7309",

// 		GPUPrice: big.NewInt(20),
// 		GPUModel: "NIVIDA 3060",

// 		MemPrice:    big.NewInt(5),
// 		MemCapacity: 20,

// 		DiskPrice:    big.NewInt(1),
// 		DiskCapacity: 1000,
// 	}
// 	err = node.CreateNode()
// 	if err != nil {
// 		return err
// 	}

// 	order := database.Order{
// 		Address:      provider1,
// 		Id:           1,
// 		ActivateTime: time.Now(),
// 		StartTime:    time.Now().Add(30 * time.Second),
// 		EndTime:      time.Now().Add(2 * time.Hour),
// 		Probation:    30,
// 		Duration:     7170,
// 	}
// 	err = order.CreateOrder()
// 	if err != nil {
// 		return err
// 	}

// 	profit := database.Profit{
// 		Address:  provider1,
// 		Balance:  big.NewInt(0),
// 		Profit:   big.NewInt(1000000000),
// 		Penalty:  big.NewInt(0),
// 		LastTime: time.Now(),
// 		EndTime:  time.Now().Add(10 * time.Hour),
// 	}
// 	return profit.CreateProfit()
// }
