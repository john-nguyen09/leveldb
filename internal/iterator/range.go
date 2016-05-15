package iterator

import "github.com/kezhuw/leveldb/internal/keys"

type rangeIterator struct {
	cmp   keys.Comparer
	soi   bool
	iter  Iterator
	valid bool
	start []byte
	limit []byte
}

func (it *rangeIterator) checkStart(valid bool) bool {
	if valid && it.cmp.Compare(it.iter.Key(), it.start) >= 0 {
		it.valid = true
		return true
	}
	it.valid = false
	return false
}

func (it *rangeIterator) checkLimit(valid bool) bool {
	if valid && it.cmp.Compare(it.iter.Key(), it.limit) < 0 {
		it.valid = true
		return true
	}
	it.valid = false
	return false
}

func (it *rangeIterator) First() bool {
	it.soi = true
	return it.checkLimit(it.iter.Seek(it.start))
}

func (it *rangeIterator) Last() bool {
	it.soi = true
	switch {
	case it.iter.Seek(it.limit):
		for it.iter.Prev() {
			if it.cmp.Compare(it.iter.Key(), it.limit) < 0 {
				return it.checkStart(true)
			}
		}
	case it.iter.Last():
		// There are two reasons to fall in this case statment:
		// * Iterator has no elements greater than or equal to limit.
		// * Error happens in seeking. In this case, possibility exists
		//   that the last element exceed limit. So we iterate backward
		//   if necessary.
		for it.cmp.Compare(it.iter.Key(), it.limit) >= 0 {
			if !it.iter.Prev() {
				it.valid = false
				return false
			}
		}
		return it.checkStart(true)
	}
	it.valid = false
	return false
}

func (it *rangeIterator) Next() bool {
	switch {
	case !it.soi:
		return it.First()
	case it.valid:
		return it.checkLimit(it.iter.Next())
	}
	return false
}

func (it *rangeIterator) Prev() bool {
	switch {
	case !it.soi:
		return it.Last()
	case it.valid:
		return it.checkStart(it.iter.Prev())
	}
	return false
}

func (it *rangeIterator) Seek(target []byte) bool {
	it.soi = true
	switch {
	case it.cmp.Compare(target, it.start) < 0:
		target = it.start
	case it.cmp.Compare(target, it.limit) >= 0:
		it.valid = false
		return false
	}
	return it.checkLimit(it.iter.Seek(target))
}

func (it *rangeIterator) Valid() bool {
	return it.valid
}

func (it *rangeIterator) Key() []byte {
	if it.valid {
		return it.iter.Key()
	}
	return nil
}

func (it *rangeIterator) Value() []byte {
	if it.valid {
		return it.iter.Value()
	}
	return nil
}

func (it *rangeIterator) Err() error {
	return it.iter.Err()
}

func (it *rangeIterator) Release() error {
	return it.iter.Release()
}

func NewRangeIterator(start, limit []byte, cmp keys.Comparer, it Iterator) Iterator {
	switch {
	case len(start) == 0 && len(limit) == 0:
		return it
	case len(limit) == 0:
		return newStartIterator(start, cmp, it)
	case len(start) == 0:
		return newLimitIterator(limit, cmp, it)
	case cmp.Compare(start, limit) >= 0:
		return empty
	}
	return &rangeIterator{cmp: cmp, iter: it, start: start, limit: limit}
}