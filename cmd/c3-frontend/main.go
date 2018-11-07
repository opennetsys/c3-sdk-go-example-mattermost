package c3frontend

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/c3systems/c3-go/common/c3crypto"
	"github.com/c3systems/c3-go/common/txparamcoder"
	"github.com/c3systems/c3-go/core/chain/mainchain"
	"github.com/c3systems/c3-go/core/chain/statechain"
	"github.com/c3systems/c3-go/core/p2p/protobuff"
	methodTypes "github.com/c3systems/c3-go/core/types/methods"
	nodetypes "github.com/c3systems/c3-go/node/types"
	"github.com/davecgh/go-spew/spew"
	ipfsaddr "github.com/ipfs/go-ipfs-addr"
	csms "github.com/libp2p/go-conn-security-multistream"
	lCrypt "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	secio "github.com/libp2p/go-libp2p-secio"
	swarm "github.com/libp2p/go-libp2p-swarm"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	tcp "github.com/libp2p/go-tcp-transport"
	ma "github.com/multiformats/go-multiaddr"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

var imageHash string

var (
	pBuff   *protobuff.Node
	priv    *ecdsa.PrivateKey
	pub     *ecdsa.PublicKey
	pubAddr string
	newNode host.Host
	peerID  peer.ID
)

func getHeadblock() (mainchain.Block, error) {
	return mainchain.Block{}, nil

}

func broadcastTx(tx *statechain.Transaction) (*nodetypes.SendTxResponse, error) {
	return nil, nil

}

func handler(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	// note: second field is header
	file, _, err := r.FormFile("file")
	if err != nil {
		fmt.Printf("err getting file %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)

	}
	defer file.Close()

	// name := strings.Split(header.Filename, ".")
	// fmt.Printf("File name %s\n", name[0])
	// Copy the file data to my buffer
	if _, err = io.Copy(&buf, file); err != nil {
		fmt.Printf("err copying %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)

	}

	payload := txparamcoder.ToJSONArray(
		txparamcoder.EncodeMethodName("processImage"),
		txparamcoder.EncodeParam(hex.EncodeToString(buf.Bytes())),
		txparamcoder.EncodeParam("jpg"), // TODO: read this from the filename...

	)

	tx := statechain.NewTransaction(&statechain.TransactionProps{
		ImageHash: imageHash,
		Method:    methodTypes.InvokeMethod,
		Payload:   payload,
		From:      pubAddr,
	})
	if err = tx.SetSig(priv); err != nil {
		fmt.Printf("error setting sig\n%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)

	}
	if err = tx.SetHash(); err != nil {
		fmt.Printf("error setting hash\n%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)

	}
	if tx.Props().TxHash == nil {
		fmt.Print("tx hash is nil!")
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)

	}

	txBytes, err := tx.Serialize()
	if err != nil {
		fmt.Printf("error getting tx bytes\n%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)

	}
	go func() {
		ch := make(chan interface{})
		if err := pBuff.ProcessTransaction.SendTransaction(peerID, txBytes, ch); err != nil {
			fmt.Printf("err processing tx\n%v", err)
			return

		}

		res := <-ch
		fmt.Printf("received response on channel %v", res)

	}()

	if _, err = fmt.Fprint(w, *tx.Props().TxHash); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)

	}

}

// note: https://github.com/libp2p/go-libp2p-swarm/blob/da01184afe4c67bec58c5e73f3350ad80b624c0d/testing/testing.go#L39
func genUpgrader(n *swarm.Swarm) *tptu.Upgrader {
	id := n.LocalPeer()
	pk := n.Peerstore().PrivKey(id)
	secMuxer := new(csms.SSMuxer)
	secMuxer.AddTransport(secio.ID, &secio.Transport{
		LocalID:    id,
		PrivateKey: pk,
	})

	stMuxer := msmux.NewBlankTransport()
	stMuxer.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)

	return &tptu.Upgrader{
		Secure:  secMuxer,
		Muxer:   stMuxer,
		Filters: n.Filters,
	}

}

func sendGenesisBlock() error {
	ch := make(chan interface{})
	tx := statechain.NewTransaction(&statechain.TransactionProps{
		ImageHash: imageHash,
		Method:    methodTypes.Deploy,
		Payload:   nil,
		From:      pubAddr,
	})

	if err := tx.SetHash(); err != nil {
		return err

	}

	if err := tx.SetSig(priv); err != nil {
		return err

	}

	txBytes, err := tx.Serialize()
	if err != nil {
		return err

	}

	if err := pBuff.ProcessTransaction.SendTransaction(peerID, txBytes, ch); err != nil {
		return err

	}

	v := <-ch

	switch v.(type) {
	case error:
		err, _ := v.(error)
		return err

	default:
		spew.Dump(v)

		return nil

	}

}

func buildNode(peerStr string) error {
	wPriv, wPub, err := lCrypt.GenerateKeyPairWithReader(lCrypt.RSA, 4096, rand.Reader)
	if err != nil {
		fmt.Printf("err generating keypairs %v", err)
		return err

	}

	pid, err := peer.IDFromPublicKey(wPub)
	if err != nil {
		fmt.Printf("err getting pid %v", err)
		return err

	}

	uri := "/ip4/0.0.0.0/tcp/9008"
	listen, err := ma.NewMultiaddr(uri)
	if err != nil {
		fmt.Printf("err listening %v", err)
		return err

	}

	ps := peerstore.NewPeerstore()
	if err = ps.AddPrivKey(pid, wPriv); err != nil {
		fmt.Printf("err adding priv key %v", err)
		return err

	}
	if err = ps.AddPubKey(pid, wPub); err != nil {
		fmt.Printf("err adding pub key %v", err)
		return err

	}

	swarmNet := swarm.NewSwarm(context.Background(), pid, ps, nil)
	tcpTransport := tcp.NewTCPTransport(genUpgrader(swarmNet))
	if err = swarmNet.AddTransport(tcpTransport); err != nil {
		fmt.Printf("err adding transport %v", err)
		return err

	}
	if err = swarmNet.AddListenAddr(listen); err != nil {
		fmt.Printf("err adding listenaddr %v", err)
		return err

	}
	newNode = bhost.New(swarmNet)

	addr, err := ipfsaddr.ParseString(peerStr)
	if err != nil {
		fmt.Printf("err parsing peer string %v", err)
		return err

	}

	pinfo, err := peerstore.InfoFromP2pAddr(addr.Multiaddr())
	if err != nil {
		fmt.Printf("err getting pinfo %v", err)
		return err

	}

	log.Println("[node] FULL", addr.String())
	log.Println("[node] PIN INFO", pinfo)

	if err = newNode.Connect(context.Background(), *pinfo); err != nil {
		fmt.Printf("err connecting to peer; %v\n", err)
		return err

	}

	peerID = pinfo.ID
	newNode.Peerstore().AddAddrs(pinfo.ID, pinfo.Addrs, peerstore.PermanentAddrTTL)

	pBuff, err = protobuff.NewNode(&protobuff.Props{
		Host:                   newNode,
		GetHeadBlockFN:         getHeadblock,
		BroadcastTransactionFN: broadcastTx,
	})
	if err != nil {
		fmt.Printf("error starting protobuff node\n%v", err)
		return err

	}

	priv, pub, err = c3crypto.NewKeyPair()
	if err != nil {
		fmt.Printf("error getting keypair\n%v", err)
		return err

	}

	pubAddr, err = c3crypto.EncodeAddress(pub)
	if err != nil {
		fmt.Printf("error getting addr\n%v", err)
		return err

	}

	fmt.Println("FOO\n", pubAddr)

	return nil

}

func main() {
	log.Println("building node")

	imageHashFlag := flag.String("image", "", "Image hash")
	shouldSendGenesisBlock := flag.Bool("genesis", false, "send genesis block")
	peer := flag.String("peer", "", "peer multiaddr")
	flag.Parse()
	imageHash = *imageHashFlag
	if err := buildNode(*peer); err != nil {
		log.Fatalf("err building node\n%v", err)

	}

	if *shouldSendGenesisBlock {
		log.Println("sending genesis block")
		if err := sendGenesisBlock(); err != nil {
			log.Fatalf("err sending genesis block\n%v", err)

		}

	}

	http.HandleFunc("/submit", handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./index.html")

	})

	log.Println("listening on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
