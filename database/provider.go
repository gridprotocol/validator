package database

import (
	"math/big"

	"golang.org/x/xerrors"
)

type Provider struct {
	Address string `gorm:"primarykey"`
	Name    string
	IP      string
	Domain  string
	Port    string
}

func InitProvider() error {
	return GlobalDataBase.AutoMigrate(&Provider{})
}

func (p *Provider) CreateProvider() error {
	return GlobalDataBase.Create(p).Error
}

func GetProviderByAddress(address string) (Provider, error) {
	var provider Provider
	err := GlobalDataBase.Model(&Provider{}).Where("address = ?", address).First(&provider).Error
	if err != nil {
		return Provider{}, err
	}

	return provider, nil
}

type Node struct {
	Address string
	Id      int

	CPUPrice *big.Int
	CPUModel string

	GPUPrice *big.Int
	GPUModel string

	MemPrice    *big.Int
	MemCapacity int64

	DiskPrice    *big.Int
	DiskCapacity int64
}

type NodeStore struct {
	Address string `gorm:"primaryKey"`
	Id      int    `gorm:"primaryKey"`

	CPUPrice string
	CPUModel string

	GPUPrice string
	GPUModel string

	MemPrice    string
	MemCapacity int64

	DiskPrice    string
	DiskCapacity int64
}

func InitNode() error {
	return GlobalDataBase.AutoMigrate(&NodeStore{})
}

func (n *Node) CreateNode() error {
	nodeStore, err := NodeToNodeStore(*n)
	if err != nil {
		return err
	}
	return GlobalDataBase.Create(&nodeStore).Error
}

func GetNodeByAddressAndId(address string, id int) (Node, error) {
	var nodeStore NodeStore
	err := GlobalDataBase.Model(&NodeStore{}).Where("address = ? AND id = ?", address, id).First(&nodeStore).Error
	if err != nil {
		return Node{}, err
	}

	return NodeStoreToNode(nodeStore)
}

func NodeToNodeStore(node Node) (NodeStore, error) {
	return NodeStore{
		Address: node.Address,
		Id:      node.Id,

		CPUPrice: node.CPUPrice.String(),
		CPUModel: node.CPUModel,

		GPUPrice: node.GPUPrice.String(),
		GPUModel: node.GPUModel,

		MemPrice:    node.MemPrice.String(),
		MemCapacity: node.MemCapacity,

		DiskPrice:    node.DiskPrice.String(),
		DiskCapacity: node.DiskCapacity,
	}, nil
}

func NodeStoreToNode(node NodeStore) (Node, error) {
	cpuPrice, ok := new(big.Int).SetString(node.CPUPrice, 10)
	if !ok {
		return Node{}, xerrors.Errorf("Failed to convert %s to BigInt", node.CPUPrice)
	}

	gpuPrice, ok := new(big.Int).SetString(node.GPUPrice, 10)
	if !ok {
		return Node{}, xerrors.Errorf("Failed to convert %s to BigInt", node.GPUPrice)
	}

	memPrice, ok := new(big.Int).SetString(node.MemPrice, 10)
	if !ok {
		return Node{}, xerrors.Errorf("Failed to convert %s to BigInt", node.MemPrice)
	}

	diskPrice, ok := new(big.Int).SetString(node.DiskPrice, 10)
	if !ok {
		return Node{}, xerrors.Errorf("Failed to convert %s to BigInt", node.DiskPrice)
	}

	return Node{
		Address: node.Address,
		Id:      node.Id,

		CPUPrice: cpuPrice,
		CPUModel: node.CPUModel,

		GPUPrice: gpuPrice,
		GPUModel: node.GPUModel,

		MemPrice:    memPrice,
		MemCapacity: node.MemCapacity,

		DiskPrice:    diskPrice,
		DiskCapacity: node.DiskCapacity,
	}, nil
}
