package version

import (
	"sort"

	"github.com/kezhuw/leveldb/internal/endian"
	"github.com/kezhuw/leveldb/internal/iterator"
	"github.com/kezhuw/leveldb/internal/keys"
	"github.com/kezhuw/leveldb/internal/options"
	"github.com/kezhuw/leveldb/internal/table"
)

type fileIterator struct {
	open    bool
	icmp    keys.Comparator
	index   int
	files   []FileMeta
	cache   *table.Cache
	opts    *options.ReadOptions
	scratch [16]byte
}

func (it *fileIterator) First() bool {
	it.open = true
	it.index = 0
	return true
}

func (it *fileIterator) Last() bool {
	it.open = true
	it.index = len(it.files) - 1
	return true
}

func (it *fileIterator) Next() bool {
	if !it.open {
		return it.First()
	}
	it.index++
	return it.Valid()
}

func (it *fileIterator) Prev() bool {
	if !it.open {
		return it.Last()
	}
	it.index--
	return it.Valid()
}

func (it *fileIterator) Seek(ikey []byte) bool {
	it.open = true
	it.index = sort.Search(len(it.files), func(i int) bool { return it.icmp.Compare(ikey, it.files[i].Largest) < 0 })
	return it.Valid()
}

func (it *fileIterator) Valid() bool {
	return it.index >= 0 && it.index < len(it.files)
}

func (it *fileIterator) Key() []byte {
	return it.files[it.index].Largest
}

func (it *fileIterator) Value() []byte {
	endian.PutUint64(it.scratch[:8], it.files[it.index].Number)
	endian.PutUint64(it.scratch[8:], it.files[it.index].Size)
	return it.scratch[:]
}

func (it *fileIterator) child(value []byte) iterator.Iterator {
	fileNumber := endian.Uint64(value[:8])
	fileSize := endian.Uint64(value[8:])
	return it.cache.NewIterator(fileNumber, fileSize, it.opts)
}

func (it *fileIterator) Err() error {
	return nil
}

func (it *fileIterator) Release() error {
	it.index = 0
	it.icmp = nil
	it.files = nil
	return nil
}

func newFileIterator(files []FileMeta, icmp keys.Comparator) iterator.Iterator {
	n := len(files)
	if n == 0 {
		return iterator.Empty()
	}
	return &fileIterator{icmp: icmp, files: files}
}

func newSortedFileIterator(icmp keys.Comparator, files []FileMeta, cache *table.Cache, opts *options.ReadOptions) iterator.Iterator {
	n := len(files)
	if n == 0 {
		return iterator.Empty()
	}
	index := &fileIterator{icmp: icmp, files: files, cache: cache, opts: opts}
	return iterator.NewIndexIterator(index, index.child)
}