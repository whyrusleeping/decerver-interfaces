package ipfs

import (    
    "bytes"
    "fmt"
    "io"
    "strings"
    "errors"
    "encoding/hex"
    "time"

    decore "github.com/eris-ltd/decerver-interfaces/core"
    api "github.com/eris-ltd/decerver-interfaces/api"
    events "github.com/eris-ltd/decerver-interfaces/events"
    modules "github.com/eris-ltd/decerver-interfaces/modules"

    core "github.com/jbenet/go-ipfs/core"
    cmds "github.com/jbenet/go-ipfs/core/commands"
    blocks "github.com/jbenet/go-ipfs/blocks"
    config "github.com/jbenet/go-ipfs/config"
    mdag "github.com/jbenet/go-ipfs/merkledag"
    util "github.com/jbenet/go-ipfs/util"
    ftpb "github.com/jbenet/go-ipfs/unixfs/pb"
    uio "github.com/jbenet/go-ipfs/unixfs/io"
    //b58 "github.com/jbenet/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-base58"
    mh "github.com/jbenet/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-multihash"
    proto "github.com/jbenet/go-ipfs/Godeps/_workspace/src/code.google.com/p/goprotobuf/proto"
    context "github.com/jbenet/go-ipfs/Godeps/_workspace/src/code.google.com/p/go.net/context"

)

var (
    StreamSize = 1024
)


// implements decerver-interface module
type IpfsModule struct{
    ipfs Ipfs
}

// implements file system
type Ipfs struct{
    node *core.IpfsNode
    confdir string
    cfg *config.Config
}

func (mod *IpfsModule) Register(fileIO decore.FileIO, registry api.ApiRegistry, runtime decore.Runtime, eReg events.EventRegistry) error {
    return nil
}

func NewIpfs() *IpfsModule{
    return &IpfsModule{Ipfs{}}
}

func (mod *IpfsModule) Init() error{

	filename, err := config.Filename(mod.ipfs.confdir)
	if err != nil {
		return err
	}

	mod.ipfs.cfg, err = config.Load(filename)
	if err != nil {
		return err
	}

    // TODO: check ipfs init

	/*if err := updates.CliCheckForUpdates(cfg, filename); err != nil {
		return nil, err
	}*/
    return nil
}

func (mod *IpfsModule) Start() error{
    n, err := core.NewIpfsNode(mod.ipfs.cfg, true) //config, online
    if err != nil{
        return err
    }

    mod.ipfs.node = n
    return nil
}

func (mod *IpfsModule) Shutdown() error{
    return nil
}


func (mod *IpfsModule) ReadConfig(config_file string){
}

func (mod *IpfsModule) WriteConfig(config_file string){
}

func (mod *IpfsModule) Name() string{
    return "ipfs"
}

func (mod *IpfsModule) Get(cmd string, params ...string) (interface{}, error){
    return mod.ipfs.Get(cmd, params...)
}

func (mod *IpfsModule) Push(cmd string, params ...string) (string, error){
    return mod.ipfs.Push(cmd, params ...)
}

func (mod *IpfsModule) GetBlock(hash string) ([]byte, error){
    return mod.ipfs.GetBlock(hash)
}

func (mod *IpfsModule) GetFile(hash string) ([]byte, error){
    return mod.ipfs.GetFile(hash)
}

func (mod *IpfsModule) GetStream(hash string) (chan []byte, error){
    return mod.ipfs.GetStream(hash)
}

func (mod *IpfsModule) GetTree(hash string, depth int) (*modules.FsNode, error){
    return mod.ipfs.GetTree(hash, depth)
}


func (mod *IpfsModule) PushBlock(block []byte) (string, error){
    return mod.ipfs.PushBlock(block)
}

func (mod *IpfsModule) PushBlockString(block string) (string, error){
    return mod.ipfs.PushBlockString(block)
}

func (mod *IpfsModule) PushFile(fpath string) (string, error){
    return mod.ipfs.PushFile(fpath)
}

func (mod *IpfsModule) PushTree(fpath string, depth int) (string, error){
    return mod.ipfs.PushTree(fpath, depth)
}

func (mod *IpfsModule) Subscribe(name string, event string, target string) chan events.Event{
    // not implemented yet
    return nil
}


// ethereum stores hashes as 32 bytes, but ipfs expects base58 encoding
// thus our convention is that params can be a path, but it must have only a single leading hash (hex encoded)
//  and it must lead with it
func (ipfs *Ipfs) Get(cmd string, params ... string) (interface{}, error){
    // ipfs 
    n := ipfs.node
    if len(params) == 0{
        return ipfs.getCmd(cmd)
    }
    switch(cmd){
        case "block":
            h, err := hex.DecodeString(params[0])
            if err != nil{
                return nil, err
            }
            k := util.Key(h)
            ctx, _ := context.WithTimeout(context.TODO(), time.Second*5)
            b, err := n.Blocks.GetBlock(ctx, k)
            if err != nil {
                return nil, fmt.Errorf("block get: %v", err)
            }
            return b.Data, nil
        case "file":
            // should return a file's raw byte contents
            // don't use on large objects
            h, err := hexPath2B58(params[0]) 
            if err != nil{
                return nil, err
            }
            buf := bytes.NewBuffer(nil)
            err = cmds.Cat(n, []string{h}, nil, buf)
            if err != nil{
                return nil, err
            }
            return buf.Bytes(), nil
        case "stream":
            // should stream the bytes over a channel
            fpath, err := hexPath2B58(params[0])
            if err != nil{
                return nil, err
            }
            dagnode, err := n.Resolver.ResolvePath(fpath)
            if err != nil {
                return nil, fmt.Errorf("catFile error: %v", err)
            }
            read, err := uio.NewDagReader(dagnode, n.DAG)
            if err != nil {
                return nil, fmt.Errorf("cat error: %v", err)
            }
            ch := make(chan []byte)
            var n int
            go func(){
                for err != io.EOF{
                    b := make([]byte, 1024)
                    // read from reader 1024 bytes at a time
                    n, err = read.Read(b)
                    if err != nil && err != io.EOF{
                        //return nil, err
                        break
                        // how to handle these errors?!
                    }
                    // broadcast on channel
                    ch <- b[:n]
                }
                close(ch)
            }()
            return ch, nil
        case "tree": 
            // should return a directory tree structure
            fpath, err := hexPath2B58(params[0])
            if err != nil{
                return nil, err
            }
            nd, err := n.Resolver.ResolvePath(fpath)
            if err != nil {
                return nil, err
            }
            mhash, err := nd.Multihash()
            if err != nil{
                return nil, err
            }
            tree := modules.FsNode{[]*modules.FsNode{}, "", hex.EncodeToString(mhash)}
            grabRefs(n, nd, &tree)
            return tree, nil
        case "object":
            // return raw file bytes or a dir tree
            fpath, err := hexPath2B58(params[0])
            if err != nil{
                return nil, err
            }
            nd, err := n.Resolver.ResolvePath(fpath)
            if err != nil {
                return nil, err
            }

            pb := new(ftpb.Data)
            err = proto.Unmarshal(nd.Data, pb)
            if err != nil {
                return nil, err
            }

            if pb.GetType() == ftpb.Data_Directory{
                return ipfs.Get("tree", params[0])                
            } else{
               return ipfs.Get("file", params[0]) 
            }
    }
    return nil, errors.New("Invalid commmand")
}


func (ipfs *Ipfs) GetBlock(hash string) ([]byte, error){
    h, err := hex.DecodeString(hash)
    if err != nil{
        return nil, err
    }
    k := util.Key(h)
    ctx, _ := context.WithTimeout(context.TODO(), time.Second*5)
    b, err := ipfs.node.Blocks.GetBlock(ctx, k)
    if err != nil {
        return nil, fmt.Errorf("block get: %v", err)
    }
    return b.Data, nil
}

func (ipfs *Ipfs) GetFile(hash string) ([]byte, error){
    h, err := hexPath2B58(hash)
    if err != nil{
        return nil, err
    }
    buf := bytes.NewBuffer(nil)
    err = cmds.Cat(ipfs.node, []string{h}, nil, buf)
    if err != nil{
        return nil, err
    }
    return buf.Bytes(), nil
}

func (ipfs *Ipfs) GetStream(hash string) (chan []byte, error){
    fpath, err := hexPath2B58(hash)
    if err != nil{
        return nil, err
    }
    dagnode, err := ipfs.node.Resolver.ResolvePath(fpath)
    if err != nil {
        return nil, fmt.Errorf("catFile error: %v", err)
    }
    read, err := uio.NewDagReader(dagnode, ipfs.node.DAG)
    if err != nil {
        return nil, fmt.Errorf("cat error: %v", err)
    }
    ch := make(chan []byte)
    var n int
    go func(){
        for err != io.EOF{
            b := make([]byte, StreamSize)
            // read from reader 1024 bytes at a time
            n, err = read.Read(b)
            if err != nil && err != io.EOF{
                //return nil, err
                break
                // how to handle these errors?!
            }
            // broadcast on channel
            ch <- b[:n]
        }
        close(ch)
    }()
    return ch, nil
}

// TODO: depth
func (ipfs *Ipfs) GetTree(hash string, depth int) (*modules.FsNode, error){
    fpath, err := hexPath2B58(hash)
    if err != nil{
        return nil, err
    }
    nd, err := ipfs.node.Resolver.ResolvePath(fpath)
    if err != nil {
        return nil, err
    }
    mhash, err := nd.Multihash()
    if err != nil{
        return nil, err
    }
    tree := modules.FsNode{[]*modules.FsNode{}, "", hex.EncodeToString(mhash)}
    grabRefs(ipfs.node, nd, &tree)
    return &tree, nil
}

func (ipfs *Ipfs) getCmd(cmd string) (interface{}, error){
    return nil, nil
}

func grabRefs(n *core.IpfsNode, nd *mdag.Node, tree *modules.FsNode) error{
    for _, link := range nd.Links{
        h := link.Hash
        newNode := modules.FsNode{[]*modules.FsNode{}, link.Name, h.B58String()}
        nd, err := n.DAG.Get(util.Key(h))                
        if err != nil{
            //log.Errorf("error: cannot retrieve %s (%s)", h.B58String(), err)
            return err
        }
        err = grabRefs(n, nd, &newNode)
        if err != nil{
            return err
        }

        tree.Nodes = append(tree.Nodes, &newNode)
    }
    return nil
}

func (ipfs *Ipfs) Push(cmd string, params ... string) (string, error){
    if len(params) < 1{
       return "", errors.New("Invalid number of parameters") 
    }
    fpath := params[0]
    switch(cmd){
        case "block":
            data, err := hex.DecodeString(params[0])
            if err != nil{
                return "", err
            }
            b := blocks.NewBlock(data)
           
            k, err := ipfs.node.Blocks.AddBlock(b)
            if err != nil {
                return "", err
            }            
            return hex.EncodeToString([]byte(k)), nil
        case "file":
            b := bytes.NewBuffer(nil)
            nd, err := cmds.AddPath(ipfs.node, fpath, 1, b)
            h, err := nd.Multihash()
            if err != nil{
                return "", err
            }
            return hex.EncodeToString(h), nil
        case "tree": 
            b := bytes.NewBuffer(nil)
            nd, err := cmds.AddPath(ipfs.node, fpath, -1, b)
            h, err := nd.Multihash()
            if err != nil{
                return "", err
            }
            return hex.EncodeToString(h), nil
    }
    return "", errors.New("Invalid cmd")
}

func (ipfs *Ipfs) PushBlock(data []byte) (string, error){
    b := blocks.NewBlock(data)
   
    k, err := ipfs.node.Blocks.AddBlock(b)
    if err != nil {
        return "", err
    }            
    return hex.EncodeToString([]byte(k)), nil
}

func (ipfs *Ipfs) PushBlockString(data string) (string, error){
    return ipfs.PushBlock([]byte(data))
}

func (ipfs *Ipfs) PushFile(fpath string) (string, error){
    return ipfs.PushTree(fpath, 1)
}

func (ipfs *Ipfs) PushTree(fpath string, depth int) (string, error){
    b := bytes.NewBuffer(nil)
    nd, err := cmds.AddPath(ipfs.node, fpath, depth, b)
    if err != nil{
        return "", err
    }
    h, err := nd.Multihash()
    if err != nil{
        return "", err
    }
    return hex.EncodeToString(h), nil
}


func HexToB58(s string) (string, error){
    var b []byte
    if len(s) > 2{
        if s[:2] == "0x"{
            s = s[2:]
        }
    }
    b, err := hex.DecodeString(s)
    if err != nil{
        return "", err
    }
    bmh := mh.Multihash(b)
    return bmh.B58String(), nil //b58.Encode(b), nil
}


// should this return 0x prefixed?
func B58ToHex(s string) (string, error){
    r, err := mh.FromB58String(s) //b58.Decode(s) with panic recovery
    if err != nil{
        return "", err
    }
    h := hex.EncodeToString(r)
    return "0x"+h, nil
}

// convert path beginning with 32 byte hex string to path beginning with base58 encoded
func hexPath2B58(p string) (string, error){
    var err error
    p = strings.TrimLeft(p, "/") // trim leading slash
    spl := strings.Split(p, "/") // split path
    leadingHash := spl[0]
    spl[0], err = HexToB58(leadingHash) // convert leading hash to base58
    if err != nil{
        return "", err
    }

    if len(spl) > 1{
        return strings.Join(spl, "/"), nil
    }
    return spl[0], nil
}
