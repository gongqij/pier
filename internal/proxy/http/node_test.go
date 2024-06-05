package http

import (
	"sort"
	"testing"
)

func TestArray(t *testing.T) {
	nodes := []*node{
		&node{
			ip:    "127.0.0.1",
			alive: true,
		},
		&node{
			ip:    "172.22.67.71",
			alive: false,
		},
	}
	t.Log(nodes[0].alive)
	t.Log(nodes[1].alive)
	t.Log("================")
	//nodes[1].alive = true
	//t.Log(nodes[0].alive)
	//t.Log(nodes[1].alive)
	//t.Log("=================")
	changeValue(nodes, t)
}

func changeValue(nodes []*node, t *testing.T) {
	var tempNodes []*node
	for _, v := range nodes {
		tempNodes = append(tempNodes, v)
	}
	sort.SliceStable(tempNodes, func(i, j int) bool {
		if tempNodes[i].alive && !tempNodes[j].alive {
			return true
		} else if !tempNodes[i].alive && tempNodes[j].alive {
			return false
		}
		return false
	})
	t.Log(tempNodes[0].alive)
	t.Log(tempNodes[1].alive)
}
