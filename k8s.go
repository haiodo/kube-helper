// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

const (
	// DefaultNamespace - a default kubernetes namespace to use
	DefaultNamespace = "default"
)

var _client *kubernetes.Clientset
var _config *rest.Config
var once sync.Once

// client returns kubernetes client
func client() *kubernetes.Clientset {
	once.Do(func() {
		doInit()
	})
	return _client
}

func doInit() {
	path := os.Getenv("KUBECONFIG")
	if path == "" {
		path = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	var err error
	_config, err = clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		logrus.Fatalf("failed to connect kubernetes: %v", err)
	}
	_client, err = kubernetes.NewForConfig(_config)
	if err != nil {
		logrus.Fatalf("failed to connect kubernetes: %v", err)
	}
}

// ListPods -  List all pods by matching all labels passed.
// namespace - a namespace to check in
// nameExpr - name matching expression.
// labels - a set of labels pod to contain
func ListPods(namespace, nameExpr string, labels map[string]string) ([]*corev1.Pod, error) {
	var pods *corev1.PodList
	var err error
	var result []*corev1.Pod
	pods, err = client().CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var expr *regexp.Regexp
	if nameExpr != "" {
		expr, err = regexp.Compile(nameExpr)
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(pods.Items); i++ {
		pod := pods.Items[i]

		if expr != nil {
			if !expr.MatchString(pod.Name) {
				// Skip not matched pods.
				continue
			}
		}
		if labels != nil && pod.Labels != nil && pod.Status.Phase == corev1.PodRunning {
			// Check if all labels are in pod labels,
			matches := len(labels)
			for k, v := range labels {
				if pod.Labels[k] == v {
					matches--
				}
			}
			if matches == 0 {
				result = append(result, &pod)
			}
		}
	}

	return result, nil
}
