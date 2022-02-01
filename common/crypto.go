package common

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"

	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/box"
)

type PrivateKey [32]byte

func NewPrivateKey(dat [32]byte) PrivateKey {
	return PrivateKey(dat)
}

func NewPrivateKeyFromString(strPrivate string) (res PrivateKey, err error) {
	privSlice, err := hex.DecodeString(strPrivate)
	if err != nil {
		return res, errors.Wrap(err, "unable to parse private key")
	}
	if len(privSlice) != 32 {
		return res, errors.New("invalid private key")
	}
	copy(res[:], privSlice)
	return res, nil
}

func (k PrivateKey) RawPtr() *[32]byte {
	return (*[32]byte)(&k)
}

type PublicKey [32]byte

func NewPublicKey(dat [32]byte) PublicKey {
	return PublicKey(dat)
}

func (k PublicKey) Raw() [32]byte {
	return [32]byte(k)
}

func (k PublicKey) RawPtr() *[32]byte {
	return (*[32]byte)(&k)
}

func (k PublicKey) Slice() []byte {
	return []byte(k[:])
}

func (k PublicKey) String() string {
	return hex.EncodeToString(k[:])
}

func NewPublicKeyFromString(strPublic string) (res PublicKey, err error) {
	publicSlice, err := hex.DecodeString(strPublic)
	if err != nil {
		return res, errors.Wrap(err, "unable to parse public key")
	}
	if len(publicSlice) != 32 {
		return res, fmt.Errorf("invalid public key: len: %d", len(publicSlice))
	}
	copy(res[:], publicSlice)
	return res, nil
}

type KeyPair struct {
	Public  PublicKey
	Private PrivateKey
}

func NewKeyPair(public PublicKey, private PrivateKey) *KeyPair {
	return &KeyPair{
		Public:  public,
		Private: private,
	}
}

func NewKeyPairFromString(strPublic, strPrivate string) (res *KeyPair, err error) {
	res = new(KeyPair)
	if res.Public, err = NewPublicKeyFromString(strPublic); err != nil {
		return nil, errors.Wrap(err, "unable to parse public key")
	}
	if res.Private, err = NewPrivateKeyFromString(strPrivate); err != nil {
		return nil, errors.Wrap(err, "unable to parse private key")
	}
	return res, nil
}

type SharedSecret [32]byte

func NewSharedSecret(dat [32]byte) SharedSecret {
	return SharedSecret(dat)
}

func (s SharedSecret) Raw() [32]byte {
	return [32]byte(s)
}

func (s SharedSecret) RawPtr() *[32]byte {
	return (*[32]byte)(&s)
}

func (s SharedSecret) String() string {
	return hex.EncodeToString(s[:])
}

func DialEncrypted(address string, remotePublicKey PublicKey) (net.Conn, *SharedSecret, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, nil, err
	}
	public, private, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	if _, err := io.Copy(conn, bytes.NewBuffer(public[:])); err != nil {
		return nil, nil, err
	}
	var out [32]byte
	box.Precompute(&out, remotePublicKey.RawPtr(), private)
	secret := NewSharedSecret(out)
	return conn, &secret, nil
}
