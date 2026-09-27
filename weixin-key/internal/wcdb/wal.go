package wcdb

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"weixin-key/internal/safefile"
)

const (
	walHeaderSize      = 32
	walFrameHeaderSize = 24
	walMagicBE0        = 0x377f0682
	walMagicBE1        = 0x377f0683
)

type WALReplayReport struct {
	MainPages        int
	WALFrames        int
	WALCommits       int
	WALFramesApplied int
	Note             string
}

// DecryptDBWithWAL accepts an already-consistent, frozen main/WAL pair.
// It authenticates pages AND SQLite's cumulative WAL header/frame checksums,
// applies only a validated committed prefix, and replaces output only after
// all work succeeds. It does not obtain a live-source snapshot itself.
func DecryptDBWithWAL(encPath, walPath, outPath string, key []byte) (*WALReplayReport, error) {
	return DecryptDBWithWALContext(context.Background(), encPath, walPath, outPath, key)
}

func DecryptDBWithWALContext(ctx context.Context, encPath, walPath, outPath string, key []byte) (*WALReplayReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(filepath.Dir(outPath), ".wal-snapshot-*")
	if err != nil {
		return nil, err
	}
	tmp := f.Name()
	if err = f.Close(); err != nil {
		return nil, err
	}
	defer os.Remove(tmp)
	pages, err := DecryptDBContext(ctx, encPath, tmp, key)
	if err != nil {
		return nil, err
	}
	rep := &WALReplayReport{MainPages: pages}
	salt, err := ReadSalt(encPath)
	if err != nil {
		return nil, err
	}
	macKey := deriveMACKey(key, salt)
	defer clear(macKey)
	if walPath != "" {
		wal, err := os.Open(walPath)
		if err != nil {
			return nil, err
		}
		defer wal.Close()
		info, err := wal.Stat()
		if err != nil {
			return nil, err
		}
		if info.Size() > 0 {
			if err := applyEncryptedWAL(ctx, tmp, wal, info.Size(), key, macKey, rep); err != nil {
				return nil, err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := finalizePlaintextHeader(tmp); err != nil {
		return nil, err
	}
	if err := safefile.Replace(tmp, outPath); err != nil {
		return nil, err
	}
	return rep, nil
}

func ReadSalt(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	salt := make([]byte, SaltLength)
	if _, err := io.ReadFull(f, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// SQLite checksums use pairs of 32-bit words. Magic's low bit selects input
// word endianness; stored checksum fields are always big endian.
func walChecksum(order binary.ByteOrder, data []byte, s0, s1 uint32) (uint32, uint32) {
	for i := 0; i+8 <= len(data); i += 8 {
		s0 += order.Uint32(data[i:i+4]) + s1
		s1 += order.Uint32(data[i+4:i+8]) + s0
	}
	return s0, s1
}

func applyEncryptedWAL(ctx context.Context, outPath string, wal io.ReaderAt, size int64, key, macKey []byte, rep *WALReplayReport) error {
	var header [walHeaderSize]byte
	if _, err := wal.ReadAt(header[:], 0); err != nil {
		return fmt.Errorf("incomplete WAL header: %w", err)
	}
	magic := binary.BigEndian.Uint32(header[:4])
	if magic != walMagicBE0 && magic != walMagicBE1 {
		return fmt.Errorf("invalid WAL magic")
	}
	if binary.BigEndian.Uint32(header[4:8]) != 3007000 {
		return fmt.Errorf("unsupported WAL format version")
	}
	pageSize := int(binary.BigEndian.Uint32(header[8:12]))
	if pageSize != SQLCipherPageSize {
		return fmt.Errorf("WAL page size %d: unknown crypto profile", pageSize)
	}
	var order binary.ByteOrder = binary.LittleEndian
	if magic&1 != 0 {
		order = binary.BigEndian
	}
	s0, s1 := walChecksum(order, header[:24], 0, 0)
	if s0 != binary.BigEndian.Uint32(header[24:28]) || s1 != binary.BigEndian.Uint32(header[28:32]) {
		return fmt.Errorf("WAL header checksum mismatch")
	}
	frameSize := int64(walFrameHeaderSize + pageSize)
	frameCount := (size - walHeaderSize) / frameSize
	if (size-walHeaderSize)%frameSize != 0 {
		rep.Note = "incomplete WAL tail; only preceding validated commits can be replayed"
	}
	frame := make([]byte, frameSize) // bounded independently of total WAL size
	lastCommit := int64(-1)
	maxPage := uint32(rep.MainPages)
	for i := int64(0); i < frameCount; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := wal.ReadAt(frame, walHeaderSize+i*frameSize); err != nil {
			return err
		}
		hdr, page := frame[:walFrameHeaderSize], frame[walFrameHeaderSize:]
		if binary.BigEndian.Uint64(hdr[8:16]) != binary.BigEndian.Uint64(header[16:24]) {
			rep.Note = fmt.Sprintf("WAL frame %d belongs to another generation; replay stopped", i)
			break
		}
		n0, n1 := walChecksum(order, hdr[:8], s0, s1)
		n0, n1 = walChecksum(order, page, n0, n1)
		if n0 != binary.BigEndian.Uint32(hdr[16:20]) || n1 != binary.BigEndian.Uint32(hdr[20:24]) {
			rep.Note = fmt.Sprintf("WAL frame %d cumulative checksum failed; replay stopped", i)
			break
		}
		pgno := binary.BigEndian.Uint32(hdr[:4])
		start := 0
		if pgno == 1 {
			start = SaltLength
		}
		if pgno == 0 || !pageMACValid(macKey, page, int(pgno), start) {
			rep.Note = fmt.Sprintf("WAL frame %d page HMAC failed; replay stopped", i)
			break
		}
		maxPage = max(maxPage, pgno)
		commitSize := binary.BigEndian.Uint32(hdr[4:8])
		if commitSize > maxPage {
			rep.Note = fmt.Sprintf("WAL frame %d claims an unexplained database size; replay stopped", i)
			break
		}
		s0, s1 = n0, n1
		rep.WALFrames++
		if commitSize != 0 {
			lastCommit = i
			rep.WALCommits++
		}
	}
	if lastCommit < 0 {
		return nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(outPath, os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	ivOff := SQLCipherPageSize - SQLCipherReserve
	plain := make([]byte, pageSize)
	defer clear(plain)
	var finalSize int64
	for i := int64(0); i <= lastCommit; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := wal.ReadAt(frame, walHeaderSize+i*frameSize); err != nil {
			return err
		}
		hdr, page := frame[:walFrameHeaderSize], frame[walFrameHeaderSize:]
		pgno := binary.BigEndian.Uint32(hdr[:4])
		clear(plain)
		start := 0
		if pgno == 1 {
			copy(plain, sqliteHeader)
			start = SaltLength
		}
		cipher.NewCBCDecrypter(block, page[ivOff:ivOff+SQLCipherIVSize]).CryptBlocks(plain[start:ivOff], page[start:ivOff])
		if _, err := out.WriteAt(plain, (int64(pgno)-1)*int64(pageSize)); err != nil {
			return err
		}
		rep.WALFramesApplied++
		if size := binary.BigEndian.Uint32(hdr[4:8]); size != 0 {
			finalSize = int64(size) * int64(pageSize)
		}
	}
	if err := out.Truncate(finalSize); err != nil {
		return err
	}
	return out.Sync()
}

func finalizePlaintextHeader(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteAt([]byte{1, 1}, 18); err != nil {
		return err
	}
	return f.Sync()
}
