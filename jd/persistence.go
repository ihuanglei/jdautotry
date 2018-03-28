package jd

import (
	"errors"

	"github.com/syndtr/goleveldb/leveldb"
)

// Persistence 持久化
type Persistence struct {
	db *leveldb.DB
}

// Open 打开数据库
func (p *Persistence) Open() error {
	var err error
	p.db, err = leveldb.OpenFile("persistence", nil)
	if err != nil {
		return err
	}
	return nil
}

// Close 关闭数据库
func (p *Persistence) Close() error {
	if p.db == nil {
		return errors.New("db not open")
	}
	p.db.Close()
	return nil
}

// Put 插入数据
func (p *Persistence) Put(key string, value string) error {
	if p.db == nil {
		return errors.New("db not open")
	}
	return p.db.Put([]byte(key), []byte(value), nil)
}

// Get 获取数据
func (p *Persistence) Get(key string) (string, error) {
	if p.db == nil {
		return "", errors.New("db not open")
	}
	data, err := p.db.Get([]byte(key), nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// NewPersistence 创建实例
func NewPersistence() *Persistence {
	p := Persistence{}
	return &p
}
