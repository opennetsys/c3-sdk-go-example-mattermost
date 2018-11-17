package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/c3systems/c3-go/common/c3crypto"
	"github.com/c3systems/c3-go/common/txparamcoder"
	"github.com/c3systems/c3-go/core/chain/mainchain"
	"github.com/c3systems/c3-go/core/chain/statechain"
	"github.com/c3systems/c3-go/core/p2p/protobuff"
	methodTypes "github.com/c3systems/c3-go/core/types/methods"
	nodetypes "github.com/c3systems/c3-go/node/types"
	"github.com/c3systems/c3-sdk-go-example-mattermost/cmd/mattermost/commands"
	"github.com/c3systems/c3-sdk-go-example-mattermost/utils"
	"github.com/davecgh/go-spew/spew"
	ipfsaddr "github.com/ipfs/go-ipfs-addr"
	csms "github.com/libp2p/go-conn-security-multistream"
	lCrypt "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-peerstore/pstoremem"
	secio "github.com/libp2p/go-libp2p-secio"
	swarm "github.com/libp2p/go-libp2p-swarm"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	tcp "github.com/libp2p/go-tcp-transport"
	ma "github.com/multiformats/go-multiaddr"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

var (
	imageHash string
	pBuff     *protobuff.Node
	priv      *ecdsa.PrivateKey
	pub       *ecdsa.PublicKey
	pubAddr   string
	newNode   host.Host
	peerID    peer.ID
)

func getHeadblock() (mainchain.Block, error) {
	return mainchain.Block{}, nil

}

func broadcastTx(tx *statechain.Transaction) (*nodetypes.SendTxResponse, error) {
	return nil, nil

}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Method is: %s; Path is: %s", r.Method, r.URL.Path)

	dissalowed := make(map[string]bool)
	dissalowed["/api/v4/roles/names"] = true
	dissalowed["/api/v4/users/status/ids"] = true
	dissalowed["/api/v4/channels/members/me/view"] = true

	if strings.ToUpper(r.Method) == "POST" && !dissalowed[r.URL.Path] {
		log.Println("Sending request to C3")

		tReq, err := utils.TransformRequest(r)
		if err != nil {
			log.Printf("err transforming request\n%v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		}

		var b bytes.Buffer
		enc := gob.NewEncoder(&b)
		if err := enc.Encode(tReq); err != nil {
			log.Printf("err encoding transformed request\n%v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		}

		payload := txparamcoder.ToJSONArray(
			txparamcoder.EncodeMethodName("processReq"),
			txparamcoder.EncodeParam(hex.EncodeToString(b.Bytes())),
		)

		tx := statechain.NewTransaction(&statechain.TransactionProps{
			ImageHash: imageHash,
			Method:    methodTypes.InvokeMethod,
			Payload:   payload,
			From:      pubAddr,
		})
		if err = tx.SetSig(priv); err != nil {
			log.Printf("error setting sig\n%v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

		}
		if err = tx.SetHash(); err != nil {
			log.Printf("error setting hash\n%v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

		}
		if tx.Props().TxHash == nil {
			log.Print("tx hash is nil!")
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

		}

		txBytes, err := tx.Serialize()
		if err != nil {
			log.Printf("error getting tx bytes\n%v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

		}
		go func() {
			ch := make(chan interface{})
			if err := pBuff.ProcessTransaction.SendTransaction(peerID, txBytes, ch); err != nil {
				log.Printf("err processing tx\n%v", err)
				return

			}

			res := <-ch
			log.Printf("received response on channel %v", res)
		}()

		log.Printf("\n\ntx hash: %s\n\n", *tx.Props().TxHash)
	}

	commands.Serve(w, r)
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

func sendGenesisBlock(loc string) error {
	f, err := ioutil.ReadFile(loc)
	if err != nil {
		log.Printf("err reading file\n%v", err)
		return err
	}

	payload := txparamcoder.ToJSONArray(
		// txparamcoder.EncodeMethodName("processReq"),
		txparamcoder.EncodeParam(hex.EncodeToString(f)),
	)

	ch := make(chan interface{})
	tx := statechain.NewTransaction(&statechain.TransactionProps{
		ImageHash: imageHash,
		Method:    methodTypes.Deploy,
		Payload:   payload,
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
		log.Printf("err generating keypairs %v", err)
		return err

	}

	pid, err := peer.IDFromPublicKey(wPub)
	if err != nil {
		log.Printf("err getting pid %v", err)
		return err

	}

	uri := "/ip4/0.0.0.0/tcp/9008"
	listen, err := ma.NewMultiaddr(uri)
	if err != nil {
		log.Printf("err listening %v", err)
		return err

	}

	ps := pstoremem.NewPeerstore()
	if err = ps.AddPrivKey(pid, wPriv); err != nil {
		log.Printf("err adding priv key %v", err)
		return err

	}
	if err = ps.AddPubKey(pid, wPub); err != nil {
		log.Printf("err adding pub key %v", err)
		return err

	}

	swarmNet := swarm.NewSwarm(context.Background(), pid, ps, nil)
	tcpTransport := tcp.NewTCPTransport(genUpgrader(swarmNet))
	if err = swarmNet.AddTransport(tcpTransport); err != nil {
		log.Printf("err adding transport %v", err)
		return err

	}
	if err = swarmNet.AddListenAddr(listen); err != nil {
		log.Printf("err adding listenaddr %v", err)
		return err

	}
	newNode = bhost.New(swarmNet)

	addr, err := ipfsaddr.ParseString(peerStr)
	if err != nil {
		log.Printf("err parsing peer string %v", err)
		return err

	}

	pinfo, err := peerstore.InfoFromP2pAddr(addr.Multiaddr())
	if err != nil {
		log.Printf("err getting pinfo %v", err)
		return err

	}

	log.Println("[node] FULL", addr.String())
	log.Println("[node] PIN INFO", pinfo)

	if err = newNode.Connect(context.Background(), *pinfo); err != nil {
		log.Printf("err connecting to peer; %v\n", err)
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
		log.Printf("error starting protobuff node\n%v", err)
		return err

	}

	priv, pub, err = c3crypto.NewKeyPair()
	if err != nil {
		log.Printf("error getting keypair\n%v", err)
		return err

	}

	pubAddr, err = c3crypto.EncodeAddress(pub)
	if err != nil {
		log.Printf("error getting addr\n%v", err)
		return err

	}

	log.Println("Pub Addr\n", pubAddr)

	return nil

}

func main() {
	log.Println("building node")

	imageHashFlag := os.Getenv("image")
	shouldSendGenesisBlockStr := os.Getenv("genesis")
	shouldSendGenesisBlock, err := strconv.ParseBool(shouldSendGenesisBlockStr)
	if err != nil {
		log.Fatalf("err parsing genesis\n%v", err)
	}

	peer := os.Getenv("peer")
	genesisLoc := os.Getenv("genesisLoc")

	log.Printf("\n\nflags\nimage: %v\npeer: %v\ngenesisLoc: %v\nsendGenesis: %v\n\n", imageHashFlag, peer, genesisLoc, shouldSendGenesisBlock)
	if peer == "" {
		log.Fatal("peer command line flag is required")
	}
	if imageHashFlag == "" {
		log.Fatal("image command line flag is required")
	}
	if shouldSendGenesisBlock && genesisLoc == "" {
		log.Fatal("a genesisLoc is required when sending a genesis block")
	}

	imageHash = imageHashFlag
	if err := buildNode(peer); err != nil {
		log.Fatalf("err building node\n%v", err)

	}

	if shouldSendGenesisBlock {
		log.Println("sending genesis block")
		if err := sendGenesisBlock(genesisLoc); err != nil {
			log.Fatalf("err sending genesis block\n%v", err)
		}

		log.Println("genesis block set")
		os.Exit(0)
	}

	// TODO: get state from peer and update state

	http.HandleFunc("/", handler)

	go func() {
		log.Println("listening on :8065")
		log.Fatal(http.ListenAndServe(":8065", nil))
	}()

	if err := commands.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
