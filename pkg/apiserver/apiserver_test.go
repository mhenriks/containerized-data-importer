package apiserver

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func Test_tokenEncrption(t *testing.T) {

	proxyKeys, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Errorf("unable to generate keys: %v", err)
	}

	apiServerKeys, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Errorf("unable to generate keys: %v", err)
	}

	encryptedToken, err := GenerateEncryptedToken("fakepvc", "fakenamespace", &proxyKeys.PublicKey, apiServerKeys)

	if err != nil {
		t.Errorf("unable to encrypt token: %v", err)
	}

	tokenData, err := DecryptToken(encryptedToken, proxyKeys, &apiServerKeys.PublicKey)

	if err != nil {
		t.Errorf("unable to decrypt token: %v", err)
	}

	if tokenData.PvcName != "fakepvc" && tokenData.Namespace != "fakenamespace" {
		t.Errorf("unexpected token generated")
	}
}
