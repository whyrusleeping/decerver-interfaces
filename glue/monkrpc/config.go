package monkrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/thelonious/monkutil"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
)

var ErisLtd = path.Join(GoPath, "src", "github.com", "eris-ltd")

type ChainConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ConfigFile   string `json:"config_file"`
	DbName       string `json:"db_name"`
	RootDir      string `json:"root_dir"`
	LogFile      string `json:"log_file"`
	LLLPath      string `json:"lll_path"`
	ContractPath string `json:"contract_path"`
	Local        bool   `json:"local"`
	KeySession   string `json:"key_session"`
	KeyStore     string `json:"key_store"`
	KeyCursor    int    `json:"key_cursor"`
	KeyFile      string `json:"key_file"`
	LogLevel     int    `json:"log_level"`
}

// set default config object
var DefaultConfig = &ChainConfig{
	Host:       "localhost",
	Port:       30304,
	ConfigFile: "config",
	DbName:     "db",
	RootDir:    path.Join(usr.HomeDir, ".monkchain2"),
	Local:      false,
	KeySession: "generous",
	LogFile:    "",
	//LLLPath: path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc"),
	LLLPath:      "NETCALL",
	ContractPath: path.Join(ErisLtd, "eris-std-lib"),
	KeyStore:     "file",
	KeyCursor:    0,
	KeyFile:      path.Join(ErisLtd, "thelonious", "monk", "keys.txt"),
	LogLevel:     5,
}

// can these methods be functions in decerver that take the modules as argument?
func (mod *MonkRpcModule) WriteConfig(config_file string) {
	b, err := json.Marshal(mod.Config)
	if err != nil {
		fmt.Println("error marshalling config:", err)
		return
	}
	var out bytes.Buffer
	json.Indent(&out, b, "", "\t")
	ioutil.WriteFile(config_file, out.Bytes(), 0600)
}
func (mod *MonkRpcModule) ReadConfig(config_file string) {
	b, err := ioutil.ReadFile(config_file)
	if err != nil {
		fmt.Println("could not read config", err)
		fmt.Println("resorting to defaults")
		mod.WriteConfig(config_file)
		return
	}
	var config ChainConfig
	err = json.Unmarshal(b, &config)
	if err != nil {
		fmt.Println("error unmarshalling config from file:", err)
		fmt.Println("resorting to defaults")
		//mod.monk.config = DefaultConfig
		return
	}
	*(mod.Config) = config
}

func (mod *MonkRpcModule) SetConfig(field string, value interface{}) error {
	cv := reflect.ValueOf(mod.Config).Elem()
	f := cv.FieldByName(field)
	kind := f.Kind()

	k := reflect.ValueOf(value).Kind()
	if kind != k {
		return fmt.Errorf("Invalid kind. Expected %s, received %s", kind, k)
	}

	if kind == reflect.String {
		f.SetString(value.(string))
	} else if kind == reflect.Int {
		f.SetInt(int64(value.(int)))
	} else if kind == reflect.Bool {
		f.SetBool(value.(bool))
	}
	return nil
}

// this will probably never be used
func (mod *MonkRpcModule) SetConfigObj(config interface{}) error {
	if c, ok := config.(*ChainConfig); ok {
		mod.Config = c
	} else {
		return fmt.Errorf("Invalid config object")
	}
	return nil
}

// Set the package global variables, create the root data dir,
//  copy keys if they are available, and setup logging
func (mod *MonkRpcModule) gConfig() {
	cfg := mod.Config
	// set lll path
	if cfg.LLLPath != "" {
		monkutil.PathToLLL = cfg.LLLPath
	}

	// check on data dir
	// create keys
	_, err := os.Stat(cfg.RootDir)
	if err != nil {
		os.Mkdir(cfg.RootDir, 0777)
		_, err := os.Stat(path.Join(cfg.RootDir, cfg.KeySession) + ".prv")
		if err != nil {
			Copy(cfg.KeyFile, path.Join(cfg.RootDir, cfg.KeySession)+".prv")
		}
	}
}

// common golang, really?
func Copy(src, dst string) {
	r, err := os.Open(src)
	if err != nil {
		fmt.Println(src, err)
		logger.Errorln(err)
		return
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		fmt.Println(err)
		logger.Errorln(err)
		return
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		fmt.Println(err)
		logger.Errorln(err)
		return
	}
}
