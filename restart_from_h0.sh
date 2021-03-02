export EVMWRAP=libevmwrap.so

rm -rf ~/.moeingd/
go run github.com/moeing-chain/moeing-chain/cmd/moeingd init m1 --chain-id moeing-1


export NODIASM=1 
export NOSTACK=1
export NOINSTLOG=1
go run github.com/moeing-chain/moeing-chain/cmd/moeingd start
