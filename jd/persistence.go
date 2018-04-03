package jd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

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

	var err error

	getCurrentDirectory := func() (string, error) {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return "", err
		}
		return strings.Replace(dir, "\\", "/", -1), nil
	}

	path, err := getCurrentDirectory()
	if err != nil {
		return err
	}

	// file, _ := exec.LookPath(os.Args[0])
	// applicationPath, _ := filepath.Abs(file)
	// applicationDir, _ := filepath.Split(applicationPath)

	p.db, err = leveldb.OpenFile(path+"/persistence", nil)
	if err != nil {
		return err
	}
	return nil
}

// Delete 删除数据
func (p *Persistence) Delete(key []byte) error {
	if p.db == nil {
		return errors.New("db not open")
	}
	p.db.Delete([]byte(key), nil)
	return nil
}

// DeleteByPrefix 根据KEY前缀删除
func (p *Persistence) DeleteByPrefix(key string) error {
	if p.db == nil {
		return errors.New("db not open")
	}
	iter := p.db.NewIterator(util.BytesPrefix([]byte(key)), nil)
	for iter.Next() {
		p.Delete(iter.Key())
	}
	iter.Release()
	return iter.Error()
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
	return p.PutByte(key, []byte(value))
}

// PutByte 插入数据
func (p *Persistence) PutByte(key string, b []byte) error {
	if p.db == nil {
		return errors.New("db not open")
	}
	return p.db.Put([]byte(key), b, nil)
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

// Has 是否存在key
func (p *Persistence) Has(key string) (bool, error) {
	if p.db == nil {
		return false, errors.New("db not open")
	}
	return p.db.Has([]byte(key), nil)
}

// SizeOf 大小
func (p *Persistence) SizeOf(key string) (int64, error) {
	if p.db == nil {
		return 0, errors.New("db not open")
	}
	r := *util.BytesPrefix([]byte(key))
	size, err := p.db.SizeOf([]util.Range{r})
	if err != nil {
		return 0, err
	}
	return size.Sum(), nil
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
