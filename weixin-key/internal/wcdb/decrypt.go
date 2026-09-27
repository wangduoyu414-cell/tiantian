package wcdb

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"weixin-key/internal/safefile"
)

// sqliteHeader is the 16-byte magic that SQLCipher replaces with the KDF salt
// on page 1; decryption restores it.
var sqliteHeader = []byte("SQLite format 3\x00")

// DecryptDB decrypts a SQLCipher 4 database in the WeChat 4.1+ layout
// (page 4096, reserve 80 = IV16 + HMAC64) into a plaintext SQLite file,
// entirely in pure Go - no native WCDB/SQLCipher library required.
//
// Every page's HMAC is verified before its plaintext page is emitted, so a
// wrong key or a corrupted file aborts without publishing partial output.
// outPath is written atomically via safefile (temp file + rename).
//
// Caveat: only the main .db file is converted. If the database has hot pages
// in a -wal file, use a consistent snapshot that includes WAL instead. Never
// remove or checkpoint the user's source WAL as a workaround.
func DecryptDB(encPath, outPath string, encKey []byte) (int, error) {
	return DecryptDBContext(context.Background(), encPath, outPath, encKey)
}

func DecryptDBContext(ctx context.Context, encPath, outPath string, encKey []byte) (pages int, err error) {
	if len(encKey) != EncKeyLength {
		return 0, fmt.Errorf("enc_key must be %d bytes (got %d)", EncKeyLength, len(encKey))
	}
	info, err := os.Stat(encPath)
	if err != nil {
		return 0, err
	}
	if info.Size() == 0 || info.Size()%SQLCipherPageSize != 0 {
		return 0, fmt.Errorf("%s: size %d is not a multiple of the %d-byte page size; use a consistent snapshot including WAL, never alter the source to repair it", encPath, info.Size(), SQLCipherPageSize)
	}
	src, err := os.Open(encPath)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return 0, err
	}

	// Temp file lives in the destination directory so safefile.Replace stays
	// on the same filesystem.
	dir := filepath.Dir(outPath)
	tmp, err := os.CreateTemp(dir, ".weixin-key-decrypt-*")
	if err != nil {
		return 0, err
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		if err != nil {
			os.Remove(tmpName)
		}
	}()

	var macKey []byte
	page := make([]byte, SQLCipherPageSize)
	out := make([]byte, SQLCipherPageSize)
	pageCount := int(info.Size() / SQLCipherPageSize)
	ivOff := SQLCipherPageSize - SQLCipherReserve

	for pgno := 1; pgno <= pageCount; pgno++ {
		if err = ctx.Err(); err != nil {
			return 0, err
		}
		if _, err = io.ReadFull(src, page); err != nil {
			return 0, fmt.Errorf("read page %d: %w", pgno, err)
		}
		start := 0
		if pgno == 1 {
			macKey = deriveMACKey(encKey, page[:SaltLength])
			start = SaltLength
		}
		if !pageMACValid(macKey, page, pgno, start) {
			return 0, fmt.Errorf("page %d HMAC mismatch: wrong key or corrupted database", pgno)
		}
		clear(out)
		if pgno == 1 {
			copy(out, sqliteHeader)
		}
		cipher.NewCBCDecrypter(block, page[ivOff:ivOff+SQLCipherIVSize]).CryptBlocks(out[start:ivOff], page[start:ivOff])
		if _, err = tmp.Write(out); err != nil {
			return 0, err
		}
	}
	if err = tmp.Sync(); err != nil {
		return 0, err
	}
	if err = tmp.Close(); err != nil {
		return 0, err
	}
	if err = safefile.Replace(tmpName, outPath); err != nil {
		return 0, err
	}
	return pageCount, nil
}
