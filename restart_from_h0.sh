export EVMWRAP=libevmwrap.so

rm -rf ~/.smartbchd/
go run github.com/smartbch/smartbch/cmd/smartbchd init m1 --chain-id 0x1


export NODIASM=1 
export NOSTACK=1
export NOINSTLOG=1
go run github.com/smartbch/smartbch/cmd/smartbchd start
