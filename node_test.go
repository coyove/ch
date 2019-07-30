package node

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNodesFuzzy(t *testing.T) {
	rand.Seed(time.Now().Unix())
	testNode = true

	nodes := []*Node{
		&Node{Name: "aa", Weight: 10},
		&Node{Name: "bb", Weight: 25},
		&Node{Name: "cc", Weight: 10},
		&Node{Name: "dd", Weight: 5},
	}

	mgr := &Nodes{}
	mgr.LoadNodes(nodes)

	m := sync.Map{}

	for i := 0; i < 1e3; i++ {
		wg := sync.WaitGroup{}
		for j := 0; j < 100; j++ {
			wg.Add(1)

			if rand.Intn(1000000) == 0 {
				nodes = append(nodes, &Node{
					Name:   strconv.Itoa(i*200000 + j),
					Weight: int64(rand.Intn(10) + 10),
				})
				mgr.LoadNodes(nodes)
			}

			go func() {
				k, v := fmt.Sprintf("%x", rand.Uint64()), fmt.Sprintf("%x", rand.Uint64())
				mgr.Put(k, v)
				m.Store(k, v)
				wg.Done()
			}()
		}
		wg.Wait()
		//log.Println(i)
	}

	for i := 0; i < 2; i++ {
		count := 0
		m.Range(func(k, v interface{}) bool {
			v2, err := mgr.Get(k.(string))
			if err != nil {
				panic(err)
			}
			if v.(string) != v2 {
				t.Fatal(v, v2)
			}
			count++
			log.Println(count)
			return true
		})
	}
}