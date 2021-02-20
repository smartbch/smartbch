package testutils

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var TestKeys = []string{
	"0xe3d9be2e6430a9db8291ab1853f5ec2467822b33a1a08825a22fab1425d2bff9",
	"0x5a09e9d6be2cdc7de8f6beba300e52823493cd23357b1ca14a9c36764d600f5e",
	"0x7e01af236f9c9536d9d28b07cea24ccf21e21c9bc9f2b2c11471cd82dbb63162",
	"0x1f67c31733dc3fd02c1f9ce9cb9e05b1d2f1b7b5463fef8acf6cf17f3bd01467",
	"0x8aa75c97b22e743e2d14a0472406f03cc5b4a050e8d4300040002096f50c0c6f",
	"0x84a453fe127ae889de1cfc28590bf5168d2843b50853ab3c5080cd5cf9e18b4b",
	"0x40580320383dbedba7a5305a593ee2c46581a4fd56ff357204c3894e91fbaf48",
	"0x0e3e6ba041d8ad56b0825c549b610e447ec55a72bb90762d281956c56146c4b3",
	"0x867b73f28bea9a0c83dfc233b8c4e51e0d58197de7482ebf666e40dd7947e2b6",
	"0xa3ff378a8d766931575df674fbb1024f09f7072653e1aa91641f310b3e1c5275",
}

func GenKeyAndAddr() (string, common.Address) {
	key, _ := crypto.GenerateKey()
	keyHex := hex.EncodeToString(crypto.FromECDSA(key))
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return keyHex, addr
}

func LoadOrGenKeys(keysFile string) ([]string, error) {
	keys, err := LoadKeys(keysFile)
	if err != nil && os.IsNotExist(err) {
		keys, err = GenKeysAndSaveToFile(keysFile, 10)
	}
	return keys, err
}

func LoadKeys(keysFile string) ([]string, error) {
	accData, err := ioutil.ReadFile(keysFile)
	if err != nil {
		return nil, err
	}

	var keys []string
	err = json.Unmarshal(accData, &keys)
	return keys, err
}

func GenKeysAndSaveToFile(keysFile string, n int) ([]string, error) {
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateKey()
		keys[i] = hex.EncodeToString(crypto.FromECDSA(key))
	}
	bytes, err := json.Marshal(keys)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(keysFile, bytes, 0644)
	return keys, err
}
