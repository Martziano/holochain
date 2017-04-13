// Copyright (C) 2013-2017, The MetaCurrency Project (Eric Harris-Braun, Arthur Brock, et. al.)
// Use of this source code is governed by GPLv3 found in the LICENSE file
//----------------------------------------------------------------------------------------

// implements bootstrap server access

package holochain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"io/ioutil"
	"net/http"
	"strings"
)

type BSReq struct {
	Version  int
	NodeID   string
	NodeAddr string
}

type BSResp struct {
	Req    BSReq
	Remote string
}

func (h *Holochain) BSpost() (err error) {
	if h.node == nil {
		return errors.New("Node hasn't been initialized yet.")
	}
	nodeID := peer.IDB58Encode(h.id)
	req := BSReq{Version: 1, NodeID: nodeID, NodeAddr: h.node.NetAddr.String()}
	host := h.config.BootstrapServer
	id := h.DNAHash()
	url := fmt.Sprintf("http://%s/%s/%s", host, id.String(), nodeID)
	var b []byte
	b, err = json.Marshal(req)
	//var resp *http.Response
	if err == nil {
		_, err = http.Post(url, "application/json", bytes.NewBuffer(b))
	}
	return
}

func (h *Holochain) BSget() (err error) {
	if h.node == nil {
		return errors.New("Node hasn't been initialized yet.")
	}
	host := h.config.BootstrapServer
	if host == "" {
		return
	}
	id := h.DNAHash()
	url := fmt.Sprintf("http://%s/%s", host, id.String())
	var resp *http.Response
	resp, err = http.Get(url)
	if err == nil {
		defer resp.Body.Close()
		var b []byte
		b, err = ioutil.ReadAll(resp.Body)
		if err == nil {
			var nodes []BSResp
			err = json.Unmarshal(b, &nodes)
			if err == nil {
				myNodeID := peer.IDB58Encode(h.id)
				for _, r := range nodes {
					h.dht.dlog.Logf("checking returned node: %v", r)

					var id peer.ID
					var addr ma.Multiaddr
					id, err = peer.IDB58Decode(r.Req.NodeID)
					if err == nil {
						//@TODO figure when to use Remote or r.NodeAddr
						x := strings.Split(r.Remote, ":")
						y := strings.Split(r.Req.NodeAddr, "/")
						port := y[len(y)-1]

						addr, err = ma.NewMultiaddr("/ip4/" + x[0] + "/tcp/" + port)
						if err == nil {
							if myNodeID != r.Req.NodeID {
								h.dht.dlog.Logf("discovered peer: %s (%v)", r.Req.NodeID, addr)
								h.node.Host.Peerstore().AddAddr(id, addr, pstore.PermanentAddrTTL)
								err = h.dht.UpdateGossiper(id, 0)

							}

						}
					}

				}
			}
		}
	}
	return
}
