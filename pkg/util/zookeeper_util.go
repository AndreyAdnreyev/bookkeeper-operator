/**
 * Copyright (c) 2018 Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package util

import (
	"container/list"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	re *regexp.Regexp
)

const (
	// Set in https://github.com/pravega/bookkeeper/blob/master/docker/bookkeeper/entrypoint.sh#L21
	PravegaPath        = "pravega"
	ZkFinalizer        = "cleanUpZookeeper"
	IPRegexp    string = `([1-9][0-9]*\.[0-9]+\.[0-9]+\.[0-9]+)`
)

func init() {
	re = regexp.MustCompile(IPRegexp)
}

func getHost(uri string, namespace string) []string {
	zkUri := strings.Split(uri, ":")
	zkSvcName := ""
	zkSvcPort := ""
	if len(zkUri) >= 1 {
		zkSvcName = zkUri[0]
		if len(zkUri) == 1 {
			zkSvcPort = "2181"
		} else {
			zkSvcPort = zkUri[1]
		}
	}
	match := re.MatchString(zkSvcName)
	hostname := ""
	if match {
		hostname = zkSvcName + ":" + zkSvcPort
	} else {
		hostname = zkSvcName + "." + namespace + ".svc.cluster.local:" + zkSvcPort
	}
	return []string{hostname}
}

func getRoot(name string) string {
	return fmt.Sprintf("/%s/%s", PravegaPath, name)
}

func getZnode(name string) string {
	return fmt.Sprintf("/%s/bookkeeper/conf", getRoot(name))
}

func CreateZnode(uri string, namespace string, name string, replicas int32) (err error) {
	host := getHost(uri, namespace)
	conn, _, err := zk.Connect(host, time.Second*5)
	if err != nil {
		return fmt.Errorf("failed to connect to zookeeper (%s): %v", host[0], err)
	}
	defer conn.Close()

	zNodePath := getZnode(name)
	exist, _, err := conn.Exists(zNodePath)
	if err != nil {
		return fmt.Errorf("failed to check if zookeeper path exists: %v", err)
	}
	if exist {
		return nil
	} else {
		data := "CLUSTER_SIZE=" + strconv.Itoa(int(replicas))
		if _, err := conn.Create(zNodePath, []byte(data), 0, zk.WorldACL(zk.PermAll)); err != nil {
			return fmt.Errorf("failed to create znode (%s) : %v", zNodePath, err)
		}
	}
	return nil
}

func UpdateZnode(uri string, namespace string, name string, replicas int32) (err error) {
	host := getHost(uri, namespace)
	conn, _, err := zk.Connect(host, time.Second*5)
	if err != nil {
		return fmt.Errorf("failed to connect to zookeeper (%s): %v", host[0], err)
	}
	defer conn.Close()

	zNodePath := getZnode(name)
	exist, zNodeStat, err := conn.Exists(zNodePath)
	if err != nil {
		return fmt.Errorf("failed to check if zookeeper path exists: %v", err)
	}
	if exist {
		data := "CLUSTER_SIZE=" + strconv.Itoa(int(replicas))
		if _, err := conn.Set(zNodePath, []byte(data), zNodeStat.Version); err != nil {
			return fmt.Errorf("failed to update znode (%s) : %v", zNodePath, err)
		}
	}
	return nil
}

// Delete all znodes related to a specific Bookkeeper cluster
func DeleteAllZnodes(uri string, namespace string, name string) (err error) {
	host := getHost(uri, namespace)
	conn, _, err := zk.Connect(host, time.Second*5)
	if err != nil {
		return fmt.Errorf("failed to connect to zookeeper (%s): %v", host[0], err)
	}
	defer conn.Close()

	root := getRoot(name)
	exist, _, err := conn.Exists(root)
	if err != nil {
		return fmt.Errorf("failed to check if zookeeper path exists: %v", err)
	}

	if exist {
		// Construct BFS tree to delete all znodes recursively
		tree, err := ListSubTreeBFS(conn, root)
		if err != nil {
			return fmt.Errorf("failed to construct BFS tree: %v", err)
		}

		for tree.Len() != 0 {
			err := conn.Delete(tree.Back().Value.(string), -1)
			if err != nil {
				return fmt.Errorf("failed to delete znode (%s): %v", tree.Back().Value.(string), err)
			}
			tree.Remove(tree.Back())
		}
		log.Println("zookeeper metadata deleted")
	} else {
		log.Println("zookeeper metadata not found")
	}
	return nil
}

// Construct a BFS tree
func ListSubTreeBFS(conn *zk.Conn, root string) (*list.List, error) {
	queue := list.New()
	tree := list.New()
	queue.PushBack(root)
	tree.PushBack(root)

	for {
		if queue.Len() == 0 {
			break
		}
		node := queue.Front()
		children, _, err := conn.Children(node.Value.(string))
		if err != nil {
			return tree, err
		}

		for _, child := range children {
			childPath := fmt.Sprintf("%s/%s", node.Value.(string), child)
			queue.PushBack(childPath)
			tree.PushBack(childPath)
		}
		queue.Remove(node)
	}
	return tree, nil
}
