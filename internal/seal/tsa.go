package seal

import (
	"bytes"
	"crypto/rand"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"
)

const DefaultTSA = "https://freetsa.org/tsr"

var oidSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}

type algorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue
}

type messageImprint struct {
	HashAlgorithm algorithmIdentifier
	HashedMessage []byte
}

type timeStampReq struct {
	Version        int
	MessageImprint messageImprint
	Nonce          *big.Int `asn1:"optional"`
	CertReq        bool     `asn1:"optional"`
}

type pkiStatusInfo struct {
	Status int
	Rest   asn1.RawValue `asn1:"optional"`
}

type timeStampResp struct {
	Status pkiStatusInfo
	Token  asn1.RawValue `asn1:"optional"`
}

func Request(digest []byte) ([]byte, error) {
	if len(digest) != 32 {
		return nil, errors.New("digest must be SHA-256")
	}
	nonce, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 63))
	if err != nil {
		return nil, err
	}
	return asn1.Marshal(timeStampReq{
		Version: 1,
		MessageImprint: messageImprint{
			HashAlgorithm: algorithmIdentifier{Algorithm: oidSHA256, Parameters: asn1.NullRawValue},
			HashedMessage: digest,
		},
		Nonce:   nonce,
		CertReq: true,
	})
}

func CheckReply(reply []byte) error {
	var resp timeStampResp
	rest, err := asn1.Unmarshal(reply, &resp)
	if err != nil {
		return fmt.Errorf("not a timestamp reply: %w", err)
	}
	if len(rest) > 0 {
		return errors.New("trailing data after timestamp reply")
	}
	if resp.Status.Status > 1 {
		return fmt.Errorf("timestamp refused (status %d)", resp.Status.Status)
	}
	if len(resp.Token.FullBytes) == 0 {
		return errors.New("timestamp reply has no token")
	}
	return nil
}

func HTTPStamper(url string, client *http.Client) Stamper {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return func(digest []byte) ([]byte, error) {
		req, err := Request(digest)
		if err != nil {
			return nil, err
		}
		resp, err := client.Post(url, "application/timestamp-query", bytes.NewReader(req))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("timestamp server answered HTTP %d", resp.StatusCode)
		}
		reply, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		if err != nil {
			return nil, err
		}
		if err := CheckReply(reply); err != nil {
			return nil, err
		}
		return reply, nil
	}
}
