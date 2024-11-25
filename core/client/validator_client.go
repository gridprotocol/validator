package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"grid-prover/core/types"
	"io"
	"net/http"

	"golang.org/x/xerrors"
)

type GRIDClient struct {
	baseUrl string
}

func NewGRIDClient(url string) *GRIDClient {
	return &GRIDClient{
		baseUrl: url,
	}
}

type rndResult struct {
	Rnd string
}

func (c *GRIDClient) GetRND(ctx context.Context) ([32]byte, error) {
	var url = c.baseUrl + "/rnd"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return [32]byte{}, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return [32]byte{}, err
	}

	if res.StatusCode != http.StatusOK {
		return [32]byte{}, xerrors.Errorf("Failed to get rnd, status [%d]", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return [32]byte{}, err
	}

	var rndRes rndResult
	err = json.Unmarshal(body, &rndRes)
	if err != nil {
		return [32]byte{}, err
	}

	rndBytes, err := hex.DecodeString(rndRes.Rnd)
	if err != nil {
		return [32]byte{}, err
	}

	var rnd [32]byte
	copy(rnd[:], rndBytes)
	return rnd, nil
}

func (c *GRIDClient) SubmitProof(ctx context.Context, proof types.Proof) error {
	var url = c.baseUrl + "/proof"
	payload := make(map[string]interface{})

	payload["address"] = proof.Address
	payload["id"] = proof.ID
	payload["nonce"] = proof.Nonce
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return xerrors.Errorf("Failed to submit proof, status [%d]", res.StatusCode)
	}

	return nil
}
