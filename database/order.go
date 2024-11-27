package database

import "time"

type Order struct {
	Address      string
	Id           int
	ActivateTime time.Time `gorm:"column:activate"`
	StartTime    time.Time `gorm:"column:start"`
	EndTime      time.Time `gorm:"column:end"`
	Probation    int64
	Duration     int64
}

func InitOrder() error {
	return GlobalDataBase.AutoMigrate(&Order{})
}

func (o *Order) CreateOrder() error {
	o.StartTime = o.ActivateTime.Add(time.Duration(o.Probation) * time.Second)
	o.EndTime = o.StartTime.Add(time.Duration(o.Duration) * time.Second)
	return GlobalDataBase.Create(o).Error
}

func GetOrderByAddressAndId(address string, id int64) (Order, error) {
	var order Order
	err := GlobalDataBase.Model(&Order{}).Where("address = ? AND id = ?", address, id).Last(&order).Error
	if err != nil {
		return Order{}, err
	}

	return order, nil
}

// get orders count by provider address
func GetOrderCount(address string) (int64, error) {
	var cnt int64
	err := GlobalDataBase.Model(&Order{}).Where("address = ?", address).Count(&cnt).Error
	if err != nil {
		return -1, err
	}

	return cnt, nil
}

func ListAllActivedOrder() ([]Order, error) {
	var now = time.Now()
	var orders []Order
	err := GlobalDataBase.Model(&Order{}).Where("start < ? AND end > ?", now, now).Find(&orders).Error
	if err != nil {
		return nil, err
	}

	return orders, nil
}
