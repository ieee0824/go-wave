package wave

import (
	"encoding/binary"
	"fmt"
	"io"

	std_buffer "bytes"

	"github.com/ieee0824/go-wave/bytes"
)

type WriterParam struct {
	Out            io.WriteCloser
	WaveFormatType int
	Channel        int
	SampleRate     int
	BitsPerSample  int
}

type Writer struct {
	out            io.WriteCloser // 実際に書きだすファイルや bytes など
	writtenSamples int            // 書き込んだサンプル数

	riffChunk *RiffChunk
	fmtChunk  *FmtChunk
	dataChunk *DataWriterChunk
}

func NewWriter(param WriterParam) (*Writer, error) {
	w := &Writer{}
	w.out = param.Out

	blockSize := uint16(param.BitsPerSample*param.Channel) / 8
	samplesPerSec := uint32(int(blockSize) * param.SampleRate)
	//	fmt.Println(blockSize, param.SampleRate, samplesPerSec)

	// riff chunk
	w.riffChunk = &RiffChunk{
		ID:         []byte(riffChunkToken),
		FormatType: []byte(waveFormatType),
	}
	// fmt chunk
	w.fmtChunk = &FmtChunk{
		ID:   []byte(fmtChunkToken),
		Size: uint32(fmtChunkSize),
	}
	w.fmtChunk.Data = &WavFmtChunkData{
		WaveFormatType: uint16(param.WaveFormatType),
		Channel:        uint16(param.Channel),
		SamplesPerSec:  uint32(param.SampleRate),
		BytesPerSec:    samplesPerSec,
		BlockSize:      uint16(blockSize),
		BitsPerSamples: uint16(param.BitsPerSample),
	}
	// data chunk
	w.dataChunk = &DataWriterChunk{
		ID:   []byte(dataChunkToken),
		Data: bytes.New(4096),
	}

	return w, nil
}

func (w *Writer) WriteSample8(samples []uint8) (int, error) {
	buf := new(std_buffer.Buffer)

	for i := 0; i < len(samples); i++ {
		err := binary.Write(buf, binary.LittleEndian, samples[i])
		if err != nil {
			return 0, err
		}
	}
	// n, err := w.Write(buf.Bytes())
	n, err := io.Copy(w, buf)
	return int(n), err
}

func (w *Writer) WriteSample16(samples []int16) (int, error) {
	buf := new(std_buffer.Buffer)

	for i := 0; i < len(samples); i++ {
		err := binary.Write(buf, binary.LittleEndian, samples[i])
		if err != nil {
			return 0, err
		}
	}
	// n, err := w.Write(buf.Bytes())
	n, err := io.Copy(w, buf)

	return int(n), err
}

func (w *Writer) WriteSample24(samples []byte) (int, error) {
	return 0, fmt.Errorf("WriteSample24 is not implemented")
}

func (w *Writer) Write(p []byte) (int, error) {
	blockSize := int(w.fmtChunk.Data.BlockSize)
	if len(p) < blockSize {
		return 0, fmt.Errorf("writing data need at least %d bytes", blockSize)
	}
	// 書き込みbyte数は BlockSize の倍数
	if len(p)%blockSize != 0 {
		return 0, fmt.Errorf("writing data must be a multiple of %d bytes", blockSize)
	}
	num := len(p) / blockSize

	n, err := w.dataChunk.Data.Write(p)

	if err == nil {
		w.writtenSamples += num
	}
	return n, err
}

type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) Write(order binary.ByteOrder, data interface{}) {
	if ew.err != nil {
		return
	}
	ew.err = binary.Write(ew.w, order, data)
}

func (w *Writer) Close() error {
	w.riffChunk.Size = uint32(len(w.riffChunk.ID)) + (8 + w.fmtChunk.Size) + (8 + w.dataChunk.Data.DataSize())
	w.dataChunk.Size = w.dataChunk.Data.DataSize()

	ew := &errWriter{w: w.out}
	// riff chunk
	ew.Write(binary.BigEndian, w.riffChunk.ID)
	ew.Write(binary.LittleEndian, w.riffChunk.Size)
	ew.Write(binary.BigEndian, w.riffChunk.FormatType)

	// fmt chunk
	ew.Write(binary.BigEndian, w.fmtChunk.ID)
	ew.Write(binary.LittleEndian, w.fmtChunk.Size)
	ew.Write(binary.LittleEndian, w.fmtChunk.Data)

	//data chunk
	ew.Write(binary.BigEndian, w.dataChunk.ID)
	ew.Write(binary.LittleEndian, w.dataChunk.Size)

	if ew.err != nil {
		return ew.err
	}

	// _, err := w.out.Write(w.dataChunk.Data.Bytes())
	_, err := io.Copy(w.out, w.dataChunk.Data)
	if err != nil {
		return err
	}

	if err := w.out.Close(); err != nil {
		return err
	}

	if err := w.dataChunk.Data.Close(); err != nil {
		return err
	}

	return nil
}
