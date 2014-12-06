package monkrpc

import (
	"fmt"
    "io"
	"os"
	"os/user"
    "log"
    "path"

	"github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/thelonious/monkdb"
	"github.com/eris-ltd/thelonious/monkrpc"
	"github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/thelonious/monklog"
)

var (
	GoPath = os.Getenv("GOPATH")
	usr, _ = user.Current() // error?!
)

// A tx to be signed by a local daemon
func newLocalTx(addr, value, gas, gasprice, body string) monkrpc.NewTxArgs {
	return monkrpc.NewTxArgs{
		Recipient: addr,
		Value:     value,
		Gas:       gas,
		GasPrice:  gasprice,
		Body:      body,
	}
}

// A full formed and signed rlp encoded tx to be broadcast by a remote server
func newRemoteTx(key []byte, addr, value, gas, gasprice, body string) monkrpc.PushTxArgs {
	addrB := monkutil.Hex2Bytes(addr)
	valB := monkutil.Big(value)
	gasB := monkutil.Big(gas)
	gaspriceB := monkutil.Big(gasprice)
	bodyB := monkutil.Hex2Bytes(body)
	tx := monkchain.NewTransactionMessage(addrB, valB, gasB, gaspriceB, bodyB)
	tx.Sign(key)
	txenc := tx.RlpEncode()
	return monkrpc.PushTxArgs{monkutil.Bytes2Hex(txenc)}
}

// Send a tx to the local server
func (mod *MonkRpcModule) rpcLocalTxCall(args monkrpc.NewTxArgs) (string, error) {
	res := new(monkrpc.TxResponse)
	err := mod.client.Call("TheloniousApi.Transact", args, res)
	if err != nil {
		return "", err
	}
	return res.Hash, nil
}

// Send a tx to the remote server
func (mod *MonkRpcModule) rpcRemoteTxCall(args monkrpc.PushTxArgs) (string, error) {
	res := new(string)
	err := mod.client.Call("TheloniousApi.Transact", args, res)
	if err != nil {
		return "", err
	}
	return *res, nil
}

func NewDatabase(dbName string) monkutil.Database {
	db, err := monkdb.NewLDBDatabase(dbName)
	if err != nil {
		exit(err)
	}
	return db
}

func NewKeyManager(KeyStore string, Datadir string, db monkutil.Database) *monkcrypto.KeyManager {
	var keyManager *monkcrypto.KeyManager
	switch {
	case KeyStore == "db":
		keyManager = monkcrypto.NewDBKeyManager(db)
	case KeyStore == "file":
		keyManager = monkcrypto.NewFileKeyManager(Datadir)
	default:
		exit(fmt.Errorf("unknown keystore type: %s", KeyStore))
	}
	return keyManager
}

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		status = 1
	}
	os.Exit(status)
}


func InitLogging(Datadir string, LogFile string, LogLevel int, DebugFile string) {
	var writer io.Writer
	if LogFile == "" {
		writer = os.Stdout
	} else {
		writer = openLogFile(Datadir, LogFile)
	}
	monklog.AddLogSystem(monklog.NewStdLogSystem(writer, log.LstdFlags, monklog.LogLevel(LogLevel)))
	if DebugFile != "" {
		writer = openLogFile(Datadir, DebugFile)
		monklog.AddLogSystem(monklog.NewStdLogSystem(writer, log.LstdFlags, monklog.DebugLevel))
	}
}

func AbsolutePath(Datadir string, filename string) string {
	if path.IsAbs(filename) {
		return filename
	}
	return path.Join(Datadir, filename)
}

func openLogFile(Datadir string, filename string) *os.File {
	path := AbsolutePath(Datadir, filename)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("error opening log file '%s': %v", filename, err))
	}
	return file
}