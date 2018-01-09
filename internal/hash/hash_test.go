package hash_test

import (
	"testing"

	"github.com/kezhuw/leveldb/internal/hash"
)

type hashTest struct {
	b []byte
	h uint32
}

var hashTests = []hashTest{
	{
		h: 0xbc9f1d34,
	},
	{
		b: []byte{0x8f},
		h: 0xda04bad8,
	},
	{
		b: []byte{0x39, 0xa5},
		h: 0x47cded56,
	},
	{
		b: []byte{0x8f, 0xe3, 0x23},
		h: 0x4aa4315e,
	},
	{
		b: []byte{0x49, 0x9a, 0xbf, 0xb7},
		h: 0x00d4fc07,
	},
	{
		b: []byte{
			0x99, 0xaa, 0xbb, 0xcc,
			0x88, 0x97, 0xa6, 0xb5,
			0xc9, 0xe7, 0x29, 0x8b,
			0xfb, 0x09, 0x00, 0x07,
		},
		h: 0xc61f9da3,
	},
	{
		b: []byte{
			0x99, 0xaa, 0xbb, 0xcc,
			0x88, 0x97, 0xa6, 0xb5,
			0xc9, 0xe7, 0x29, 0x8b,
			0xfb, 0x09, 0x00, 0x07,
			0xbf, 0x17, 0x24, 0x65,
			0x39, 0xba, 0xe3, 0xf9,
			0x72, 0x49, 0xd3, 0xc3,
			0x00, 0x77, 0x7b, 0xd7,
			0x34, 0x59, 0xb6, 0xc9,
		},
		h: 0x316e68f4,
	},
}

func TestHash(t *testing.T) {
	for i, test := range hashTests {
		got := hash.Hash(test.b)
		if got != test.h {
			t.Errorf("test=%d b=%x got=%#08x want=%#08x", i, test.b, got, test.h)
		}
	}
}
