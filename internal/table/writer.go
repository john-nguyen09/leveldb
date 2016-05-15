package table

import (
	"io"

	"github.com/kezhuw/leveldb/internal/compress"
	"github.com/kezhuw/leveldb/internal/crc"
	"github.com/kezhuw/leveldb/internal/endian"
	filterp "github.com/kezhuw/leveldb/internal/filter"
	"github.com/kezhuw/leveldb/internal/iovecs"
	"github.com/kezhuw/leveldb/internal/options"
	"github.com/kezhuw/leveldb/internal/table/block"
	"github.com/kezhuw/leveldb/internal/table/filter"
)

type Writer struct {
	w       iovecs.Writer
	options *options.Options

	err        error
	offset     int64
	footer     Footer
	lastKey    []byte
	indexKey   []byte
	numEntries int

	dataBlock        block.Writer
	dataIndexBlock   block.Writer
	pendingDataIndex block.Handle
	filterBlock      filter.Writer

	// Large enough to store encoded footer, block.Handle, etc.
	scratch       [footerLength]byte
	compressedBuf []byte
}

func (w *Writer) Reset(f io.Writer, opts *options.Options) {
	w.w = iovecs.NewWriter(f)
	w.err = nil
	w.offset = 0
	w.numEntries = 0
	w.lastKey = w.lastKey[:0]
	w.indexKey = w.indexKey[:0]
	w.dataBlock.Reset()
	w.filterBlock.Reset()
	w.dataIndexBlock.Reset()
	w.pendingDataIndex.Length = 0
	w.options = opts
	w.dataBlock.RestartInterval = opts.BlockRestartInterval
	w.dataIndexBlock.RestartInterval = 1
	if opts.Filter != nil && w.filterBlock.Generator == nil {
		w.filterBlock.Generator = filterp.NewGenerator(opts.Filter)
	}
}

func (w *Writer) LastKey() []byte {
	return w.lastKey
}

func (w *Writer) Add(key, value []byte) error {
	if w.err != nil {
		return w.err
	}

	w.flushPendingDataIndex(key)

	w.numEntries++
	w.lastKey = append(w.lastKey[:0], key...)
	w.dataBlock.Add(key, value)

	if w.dataBlock.ApproximateSize() >= w.options.BlockSize {
		w.flushDataBlock()
	}
	return w.err
}

func (w *Writer) Empty() bool {
	return w.numEntries == 0
}

func (w *Writer) flushPendingDataIndex(limit []byte) {
	if w.pendingDataIndex.Length != 0 {
		w.indexKey = w.options.Comparator.AppendSuccessor(w.indexKey[:0], w.lastKey, limit)
		n := block.EncodeHandle(w.scratch[:], w.pendingDataIndex)
		w.dataIndexBlock.Add(w.indexKey, w.scratch[:n])
		w.pendingDataIndex.Length = 0
	}
}

func (w *Writer) setError(errp *error) {
	if *errp != nil {
		w.err = *errp
	}
}

func (w *Writer) Finish() (err error) {
	if w.err != nil {
		return w.err
	}

	defer w.setError(&err)

	err = w.flushDataBlock()
	if err != nil {
		return nil
	}

	// We use dataBlock here, since it is empty and no longer used.
	w.footer.MetaIndexHandle, err = w.writeMetaIndexBlock(&w.dataBlock)
	if err != nil {
		return err
	}

	w.flushPendingDataIndex(nil)
	w.footer.DataIndexHandle, err = w.writeBlock(&w.dataIndexBlock)
	if err != nil {
		return err
	}

	i := w.footer.Encode(w.scratch[:])
	n, err := w.w.Write(w.scratch[:i])
	if err != nil {
		return err
	}
	w.offset += int64(n)

	return w.err
}

func (w *Writer) writeMetaIndexBlock(metaIndex *block.Writer) (block.Handle, error) {
	if data := w.filterBlock.Finish(); data != nil {
		handle, err := w.writeRawBlock(data, compress.NoCompression)
		if err != nil {
			return handle, err
		}
		name := []byte("filter." + w.options.Filter.Name())
		n := block.EncodeHandle(w.scratch[:], handle)
		metaIndex.Add(name, w.scratch[:n])
	}
	return w.writeBlock(metaIndex)
}

func (w *Writer) flushDataBlock() error {
	if w.dataBlock.Empty() {
		return w.err
	}
	w.pendingDataIndex, w.err = w.writeBlock(&w.dataBlock)
	return w.err
}

func (w *Writer) FileSize() int64 {
	return w.offset
}

func (w *Writer) writeBlock(block *block.Writer) (block.Handle, error) {
	data := block.Finish()
	compression := w.options.Compression
	if compression != compress.NoCompression {
		compressed, err := compress.Encode(compression, w.compressedBuf, data)
		switch err {
		case nil:
			w.compressedBuf = compressed
			data = compressed
		default:
			// TODO logging compression failure
			// Fallback to NoCompression
			compression = compress.NoCompression
		}
	}
	defer block.Reset()
	return w.writeRawBlock(data, compression)
}

func (w *Writer) writeRawBlock(contents []byte, compression compress.Type) (block.Handle, error) {
	trailer := w.scratch[:blockTrailerSize]
	trailer[0] = byte(compression)
	checksum := crc.Update(crc.New(contents), trailer[:1])
	endian.PutUint32(trailer[1:], checksum.Value())
	n, err := w.w.Writev(contents, trailer)
	if err != nil {
		return block.Handle{}, err
	}
	handle := block.Handle{Offset: uint64(w.offset), Length: uint64(len(contents))}
	w.offset += n
	return handle, nil
}