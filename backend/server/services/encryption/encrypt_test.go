// cryptopasta - basic cryptography examples
// https://github.com/gtank/cryptopasta
//
// Written in 2015 by George Tankersley <george.tankersley@gmail.com>
//
// To the extent possible under law, the author(s) have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
//
// You should have received a copy of the CC0 Public Domain Dedication along
// with this software. If not, see // <http://creativecommons.org/publicdomain/zero/1.0/>.

package encryption

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	"testing"
)

func TestEncryptDecryptGCM(t *testing.T) {

	gcmTests := []struct {
		plaintext []byte
		key       *[32]byte
	}{
		{
			plaintext: []byte("Hello, world!"),
			key:       newEncryptionKey(),
		},
		{
			plaintext: []byte("435rt4qttttttttttttawsefsf234r2das"),
			key:       newEncryptionKey(),
		},
		{
			plaintext: []byte("$#R%QW$#%RFff4tr	445353QW5WFWEFd"),
			key:       newEncryptionKey(),
		},
		{
			plaintext: []byte("#@#$$$$DR#^kjfjis0094309390"),
			key:       newEncryptionKey(),
		},
	}

	for _, tt := range gcmTests {
		ciphertext, err := encrypt(tt.plaintext, tt.key)
		if err != nil {
			t.Fatal(err)
		}

		plaintext, err := decrypt(ciphertext, tt.key)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(plaintext, tt.plaintext) {
			t.Errorf("plaintexts don't match")
		}

		ciphertext[0] ^= 0xff
		_, err = decrypt(ciphertext, tt.key)
		if err == nil {
			t.Errorf("gcmOpen should not have worked, but did")
		}
	}
}

func BenchmarkAESGCM(b *testing.B) {
	randomKey := &[32]byte{}
	_, err := io.ReadFull(rand.Reader, randomKey[:])
	if err != nil {
		b.Fatal(err)
	}

	data, err := ioutil.ReadFile("testdata/big")
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		encrypt(data, randomKey)
	}
}
