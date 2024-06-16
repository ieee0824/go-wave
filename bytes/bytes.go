package bytes

import (
	"os"
)

func New(chunkSize int) Buffer {
	tmp, err := os.CreateTemp("", "")
	if err != nil {
		return nil
	}
	ro, err := os.Open(tmp.Name())
	if err != nil {
		return nil
	}
	return &buffer{
		chunkMaxSize: chunkSize,
		chunk:        make([]byte, 0, chunkSize*2),
		tmpFile:      tmp,
		roFile:       ro,
	}
}

type Buffer interface {
	Bytes() []byte
	Write(p []byte) (n int, err error)
	Close() error
	DataSize() uint32
	Read(p []byte) (int, error)
}

type buffer struct {
	chunkMaxSize int
	chunk        []byte
	tmpFile      *os.File
	roFile       *os.File
	dataSize     uint32
}

func (b *buffer) DataSize() uint32 {
	return b.dataSize
}

func (b *buffer) Read(p []byte) (int, error) {
	if len(b.chunk) > 0 {
		_, err := b.tmpFile.Write(b.chunk)
		if err != nil {
			return 0, err
		}
		b.chunk = make([]byte, 0, b.chunkMaxSize*2)
	}
	return b.roFile.Read(p)
}

func (b *buffer) Bytes() []byte {
	raw, _ := os.ReadFile(b.tmpFile.Name())
	return append(raw, b.chunk...)
}

func (b *buffer) Write(p []byte) (n int, err error) {
	b.dataSize += uint32(len(p))
	b.chunk = append(b.chunk, p...)

	if len(b.chunk) > b.chunkMaxSize {
		_, err := b.tmpFile.Write(b.chunk)
		if err != nil {
			return 0, err
		}
		b.chunk = make([]byte, 0, b.chunkMaxSize*2)
	}

	return len(p), nil
}

func (b *buffer) Close() error {
	if err := b.tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Remove(b.tmpFile.Name()); err != nil {
		return err
	}

	return nil
}
