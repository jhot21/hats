package passgen

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"ekyu.moe/base91"
	"github.com/aead/skein"
)

func (p *Passgen) Generate() string {
	switch p.Version {
	case 1:
		return p.V1()
	default:
		return p.V2()
	}
}

func (p *Passgen) V2() string {
	charset := p.createCharset()
	var out string

	for i := 0; true; i++ {
		if len(out) >= p.Length { // password is long enough
			if p.containsNecessaryCharacters(out[0:p.Length]) { // password contains all necessary character classes
				break
			} else { // password is missing character classes, keep second half and keep generating if necessary
				out = out[int(len(out)/2):]
			}
		}

		passphrasePart := strings.Repeat(p.Passphrase, i+1)
		hash := sha256.Sum256([]byte(passphrasePart + p.Salt))

		// convert hash array of bytes into bigint
		num := new(big.Int)
		num.SetString(hex.EncodeToString(hash[:]), 16)
		// encode bigint in custom character set
		encoded := encodeToCustomCharset(num, charset)

		out = fmt.Sprintf("%s%s", out, encoded)
	}

	return out[0:p.Length]
}

func (p *Passgen) createCharset() string {
	charset := ""
	if !p.NoUppers {
		charset += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}
	charset += "abcdefghijklmnopqrstuvwxyz"
	if !p.NoNumbers {
		charset += "0123456789"
	}
	if !p.NoSpecials {
		charset += p.CustomSpecials
	}

	return charset
}

func (p *Passgen) containsNecessaryCharacters(password string) bool {
	if !p.NoNumbers {
		if matches, _ := regexp.MatchString("[0-9]", password); !matches {
			return false
		}
	}
	if !p.NoUppers {
		if matches, _ := regexp.MatchString("[A-Z]", password); !matches {
			return false
		}
	}
	if !p.NoSpecials {
		customSpecials := p.CustomSpecials
		escapeChars := []string{"-", "[", "]", "^"}
		for _, e := range escapeChars {
			customSpecials = strings.ReplaceAll(customSpecials, e, "\\"+e)
		}
		if matches, _ := regexp.MatchString(fmt.Sprintf("[%s]", customSpecials), password); !matches {
			return false
		}
	}
	if matches, _ := regexp.MatchString("[a-z]", password); !matches {
		return false
	}
	return true
}

func encodeToCustomCharset(v *big.Int, charset string) string {
	charsetLength := big.NewInt(int64(len(charset)))
	var ret string
	for {
		if v.Cmp(big.NewInt(0)) == 0 {
			break
		}
		remainder := new(big.Int).Mod(v, charsetLength)
		v = new(big.Int).Div(v, charsetLength)

		ret = fmt.Sprintf("%c%s", charset[remainder.Uint64()], ret)
	}
	return ret
}

func (p *Passgen) V1() string {
	encoded := utf16.Encode([]rune(p.Passphrase + p.Salt))
	var hashOut [64]byte
	skein.Sum512(&hashOut, convertUTF16ToLittleEndianBytes(encoded), nil)
	hashHex := hex.EncodeToString(hashOut[:])

	start := hexToInt(string(hashHex[0])) + hexToInt(string(hashHex[1])) + hexToInt(string(hashHex[2]))
	end := start + p.Length

	encoded91 := base91.EncodeToString([]byte(hashHex))

	for {
		if len(encoded91) > end {
			break
		}
		encoded91 = encoded91 + encoded91
	}

	return encoded91[start:end]
}

func convertUTF16ToLittleEndianBytes(u []uint16) []byte {
	b := make([]byte, 2*len(u))
	for index, value := range u {
		binary.LittleEndian.PutUint16(b[index*2:], value)
	}
	return b
}

func hexToInt(hexString string) int {
	i, _ := strconv.ParseInt(hexString, 16, 0)
	return int(i)
}
