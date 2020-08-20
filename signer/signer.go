package signer

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"golang.org/x/crypto/md4"
	"io"
	"math/big"
	"regexp"
)

const (
	keyLength = 164
)

var (
	ErrorKeyFileBroken         = errors.New("TestKey file is broken")
	ErrorWmIdNotConfigured     = errors.New("the WebMoney's WMID identifier not configured")
	ErrorWmIdIsIncorrect       = errors.New("the WebMoney's WMID identifier is incorrect")
	ErrorKeyNotConfigured      = errors.New("the WebMoney Pro *.kvm TestKey not configured")
	ErrorPasswordNotConfigured = errors.New("the TestPassword to WebMoney Pro *.kvm TestKey not configured")

	WmIdRegex = regexp.MustCompile("[0-9]{12}")
)

type KeyContainerInterface interface {
	Extract() (*big.Int, *big.Int, error)
	Encrypt(wmid, keyPassword string)
	Verify() bool
}

type ExternalInterface interface {
	RandRead(b []byte) (n int, err error)
	BinaryRead(r io.Reader, order binary.ByteOrder, data interface{}) error
	BinaryWrite(w io.Writer, order binary.ByteOrder, data interface{}) error
}

type WebMoneySignerInterface interface {
	Sign(data string) (string, error)
}

type word [2]byte

type Signer struct {
	power    *big.Int
	modulus  *big.Int
	external ExternalInterface
}

type crc [16]byte
type buffer [140]byte
type external struct{}

type keyContainer struct {
	key      *keyContainerKey
	external ExternalInterface
}

type keyContainerKey struct {
	Reserved uint16
	SignFlag uint16
	Crc      crc
	Length   uint32
	Buffer   buffer
}

type keyData struct {
	Reserved    uint32
	PowerBase   uint16
	Power       [66]byte
	ModulusBase uint16
	Modulus     [66]byte
}

func NewSigner(opts ...Option) (WebMoneySignerInterface, error) {
	options, err := executeOptions(opts...)

	if err != nil {
		return nil, err
	}

	decodedKey, err := base64.StdEncoding.DecodeString(options.key)

	if err != nil {
		return nil, err
	}

	if len(decodedKey) != keyLength {
		return nil, ErrorKeyFileBroken
	}

	signer := &Signer{
		external: newReader(),
	}
	keyContainer, err := options.newKeyContainerFn(decodedKey, options.wmId, options.password, signer.external)

	if err != nil {
		return nil, err
	}

	if !keyContainer.Verify() {
		return nil, ErrorKeyFileBroken
	}

	signer.power, signer.modulus, err = keyContainer.Extract()

	if err != nil {
		return nil, err
	}

	return signer, err
}

func executeOptions(opts ...Option) (*Options, error) {
	options := &Options{}

	for _, opt := range opts {
		opt(options)
	}

	if options.wmId == "" {
		return nil, ErrorWmIdNotConfigured
	}

	if !WmIdRegex.MatchString(options.wmId) {
		return nil, ErrorWmIdIsIncorrect
	}

	if options.key == "" {
		return nil, ErrorKeyNotConfigured
	}

	if options.password == "" {
		return nil, ErrorPasswordNotConfigured
	}

	if options.newKeyContainerFn == nil {
		options.newKeyContainerFn = newKeyContainer
	}

	return options, nil
}

func newReader() ExternalInterface {
	return &external{}
}

func (m *Signer) Sign(data string) (string, error) {
	hash := md4Hash([]byte(data))
	base := hash[:]

	b := make([]byte, 40)
	_, err := m.external.RandRead(b)

	if err != nil {
		return "", err
	}

	base = append(base, b...)

	baseLength := uint8(len(base))
	base = append([]byte{baseLength, 0}, base...)
	baseLength += 2
	baseReversed := reverseBytes(base)
	result := new(big.Int).Exp(new(big.Int).SetBytes(baseReversed), m.power, m.modulus)
	reversedResult, err := m.reverseBytesAsWords(result.Bytes())

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(reversedResult), nil
}

func newKeyContainer(key []byte, wmid, keyPassword string, reader ExternalInterface) (KeyContainerInterface, error) {
	keyContainer := &keyContainer{
		key:      &keyContainerKey{},
		external: reader,
	}
	err := reader.BinaryRead(bytes.NewReader(key), binary.LittleEndian, keyContainer.key)

	if err != nil {
		return nil, err
	}

	keyContainer.Encrypt(wmid, keyPassword)

	return keyContainer, nil
}

func reverseBytes(data []byte) []byte {
	length := len(data)
	reversed := make([]byte, length)

	for i := length; i > 0; i-- {
		reversed[i-1] = data[length-i]
	}

	return reversed
}

func (m *Signer) reverseWords(data []word) []word {
	length := len(data)
	reversed := make([]word, length)

	for i := length; i > 0; i-- {
		reversed[i-1] = data[length-i]
	}

	return reversed
}

func (m *Signer) reverseBytesAsWords(data []byte) ([]byte, error) {
	if len(data)%2 != 0 {
		data = append([]byte{0}, data...)
	}

	words := make([]word, len(data)/2)

	buffer := bytes.NewBuffer(data)
	err := m.external.BinaryRead(buffer, binary.LittleEndian, &words)

	if err != nil {
		return nil, err
	}

	words = m.reverseWords(words)

	var result []byte

	for _, v := range words {
		result = append(result, v[:]...)
	}

	return result, nil
}

func (m *keyContainer) Extract() (*big.Int, *big.Int, error) {
	data := new(keyData)
	err := m.external.BinaryRead(bytes.NewReader(m.key.Buffer[:]), binary.LittleEndian, data)

	if err != nil {
		return nil, nil, err
	}

	copy(data.Power[:], reverseBytes(data.Power[:]))
	copy(data.Modulus[:], reverseBytes(data.Modulus[:]))

	return new(big.Int).SetBytes(data.Power[:]), new(big.Int).SetBytes(data.Modulus[:]), nil
}

func (m *keyContainer) Encrypt(wmid, keyPassword string) {
	hash := md4Hash([]byte(wmid + keyPassword))
	m.key.Buffer = xor(m.key.Buffer, hash, 6)
}

func (m *keyContainer) Verify() bool {
	key := &keyContainerKey{
		Reserved: m.key.Reserved,
		Length:   m.key.Length,
		Buffer:   m.key.Buffer,
	}
	buffer := new(bytes.Buffer)
	err := m.external.BinaryWrite(buffer, binary.LittleEndian, key)

	if err != nil {
		return false
	}

	crc := md4Hash(buffer.Bytes())

	return crc == m.key.Crc
}

func md4Hash(data []byte) crc {
	length := len(data)

	hash := md4.New()
	hash.Write(data)

	var result crc
	copy(result[:], hash.Sum(data)[length:])

	return result
}

func xor(subject buffer, modifier crc, shift int) buffer {
	subjectLength := len(subject)
	modifierLength := len(modifier)

	var result buffer
	copy(result[:shift-1], subject[:shift-1])

	j := 0
	for i := shift; i < subjectLength; i++ {
		result[i] = subject[i] ^ modifier[j]
		j++
		if j == modifierLength {
			j = 0
		}

	}

	return result
}

func (m *external) RandRead(b []byte) (int, error) {
	return rand.Read(b)
}

func (m *external) BinaryRead(reader io.Reader, order binary.ByteOrder, data interface{}) error {
	return binary.Read(reader, order, data)
}

func (m *external) BinaryWrite(w io.Writer, order binary.ByteOrder, data interface{}) error {
	return binary.Write(w, order, data)
}
