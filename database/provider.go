package database

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
