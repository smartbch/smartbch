module github.com/moeing-chain/moeing-chain

go 1.15

require (
	github.com/ethereum/go-ethereum v1.9.25
	github.com/holiman/uint256 v1.1.1
	github.com/moeing-chain/MoeingADS v0.0.0-20210208154249-d4465b91aa22
	github.com/moeing-chain/MoeingDB v0.0.0-20210205013121-7d93963898a1
	github.com/moeing-chain/MoeingEVM v0.0.0-20210219061522-5900fa719f32
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.7
)

replace github.com/moeing-chain/MoeingEVM v0.0.0-20210219061522-5900fa719f32 => ./../MoeingEVM
