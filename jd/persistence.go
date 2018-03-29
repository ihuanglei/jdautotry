package jd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Persistence 持久化
type Persistence struct {
	db    *leveldb.DB
	batch *leveldb.Batch
}

// Open 打开数据库
func (p *Persistence) Open() error {
	file, _ := exec.LookPath(os.Args[0])
	applicationPath, _ := filepath.Abs(file)
	applicationDir, _ := filepath.Split(applicationPath)
	var err error
	p.db, err = leveldb.OpenFile(applicationDir+"/persistence", nil)
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

// Batch 批量更新
func (p *Persistence) Batch() {
	p.batch = new(leveldb.Batch)
}

// BatchPutString 批量插入字符串
func (p *Persistence) BatchPutString(key string, value string) error {
	return p.BatchPutByte(key, []byte(value))
}

// BatchPutByte 批量插入
func (p *Persistence) BatchPutByte(key string, value []byte) error {
	if p.batch == nil {
		return errors.New("must Batch")
	}
	p.batch.Put([]byte(key), value)
	return nil
}

// BatchCommit 批量提交
func (p *Persistence) BatchCommit() error {
	if p.batch == nil {
		return errors.New("must Batch")
	}
	return p.db.Write(p.batch, nil)
}

// ForEach 迭代数据
func (p *Persistence) ForEach(key string, cb func(key string, val []byte)) error {
	iter := p.db.NewIterator(util.BytesPrefix([]byte(key)), nil)
	for iter.Next() {
		cb(string(iter.Key()), iter.Value())
	}
	iter.Release()
	return iter.Error()
}

// NewPersistence 创建实例
func NewPersistence() *Persistence {
	p := Persistence{}
	return &p
}
