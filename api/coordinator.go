package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

func sendTaskToNode(addr string, task Task, wg *sync.WaitGroup, ch chan<- Result) {
	defer wg.Done()

	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		log.Printf("error connecting to %s: %v\n", addr, err)
		ch <- Result{Error: fmt.Sprintf("dial %s: %v", addr, err)}
		return
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(task); err != nil {
		log.Printf("error sending task to %s: %v\n", addr, err)
		ch <- Result{Error: err.Error()}
		return
	}

	// Training can take several minutes with large datasets
	conn.SetDeadline(time.Now().Add(10 * time.Minute))

	var res Result
	if err := json.NewDecoder(conn).Decode(&res); err != nil {
		log.Printf("error receiving result from %s: %v\n", addr, err)
		ch <- Result{Error: err.Error()}
		return
	}

	log.Printf("result from %s: %d trees in %dms\n", addr, len(res.Trees), res.DurMS)
	ch <- res
}

func distributeTraining(nodes []string, req TrainRequest) ([]Tree, []NodeInfo, int64, error) {
	treesPerNode := req.TotalTrees / len(nodes)
	ch := make(chan Result, len(nodes))
	var wg sync.WaitGroup
	start := time.Now()

	for i, addr := range nodes {
		wg.Add(1)
		task := Task{
			TaskID:      i + 1,
			NumTrees:    treesPerNode,
			MaxDepth:    req.MaxDepth,
			MaxFeatures: req.MaxFeatures,
		}
		go sendTaskToNode(addr, task, &wg, ch)
	}

	wg.Wait()
	close(ch)

	dur := time.Since(start).Milliseconds()
	var forest []Tree
	var infos []NodeInfo

	for res := range ch {
		if res.Error != "" {
			log.Println("node error:", res.Error)
			continue
		}
		forest = append(forest, res.Trees...)
		infos = append(infos, NodeInfo{
			NodeID: res.NodeID,
			Trees:  len(res.Trees),
			DurMS:  res.DurMS,
		})
	}

	if len(forest) == 0 {
		return nil, nil, 0, fmt.Errorf("no nodes responded successfully")
	}
	return forest, infos, dur, nil
}
