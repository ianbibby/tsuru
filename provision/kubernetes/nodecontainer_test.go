// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kubernetes

import (
	"github.com/fsouza/go-dockerclient"
	"github.com/tsuru/tsuru/provision/nodecontainer"
	"github.com/tsuru/tsuru/provision/servicecommon"
	"gopkg.in/check.v1"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (s *S) TestManagerDeployNodeContainer(c *check.C) {
	s.mockfakeNodes(c)
	c1 := nodecontainer.NodeContainerConfig{
		Name: "bs",
		Config: docker.Config{
			Image:      "bsimg",
			Env:        []string{"a=b"},
			Entrypoint: []string{"cmd0"},
			Cmd:        []string{"cmd1"},
		},
		HostConfig: docker.HostConfig{
			RestartPolicy: docker.AlwaysRestart(),
			Privileged:    true,
			Binds:         []string{"/xyz:/abc:ro"},
		},
	}
	err := nodecontainer.AddNewContainer("", &c1)
	c.Assert(err, check.IsNil)
	m := nodeContainerManager{client: s.client}
	err = m.DeployNodeContainer(&c1, "", servicecommon.PoolFilter{}, false)
	c.Assert(err, check.IsNil)
	daemons, err := s.client.Extensions().DaemonSets(tsuruNamespace).List(v1.ListOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(daemons.Items, check.HasLen, 1)
	daemon, err := s.client.Extensions().DaemonSets(tsuruNamespace).Get("node-container-bs-all")
	c.Assert(err, check.IsNil)
	trueVar := true
	c.Assert(daemon, check.DeepEquals, &extensions.DaemonSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      "node-container-bs-all",
			Namespace: tsuruNamespace,
		},
		Spec: extensions.DaemonSetSpec{
			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{
					"tsuru.io/node-container-name": "bs",
					"tsuru.io/node-container-pool": "",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"tsuru.io/is-tsuru":            "true",
						"tsuru.io/is-node-container":   "true",
						"tsuru.io/provisioner":         "kubernetes",
						"tsuru.io/node-container-name": "bs",
						"tsuru.io/node-container-pool": "",
					},
					Annotations: map[string]string{},
				},
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						{
							Name: "volume-0",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/xyz",
								},
							},
						},
					},
					RestartPolicy: v1.RestartPolicyAlways,
					Containers: []v1.Container{
						{
							Name:    "bs",
							Image:   "bsimg",
							Command: []string{"cmd0"},
							Args:    []string{"cmd1"},
							Env: []v1.EnvVar{
								{Name: "a", Value: "b"},
							},
							VolumeMounts: []v1.VolumeMount{
								{Name: "volume-0", MountPath: "/abc", ReadOnly: true},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: &trueVar,
							},
						},
					},
				},
			},
		},
	})
}

func (s *S) TestManagerDeployNodeContainerWithFilter(c *check.C) {
	s.mockfakeNodes(c)
	c1 := nodecontainer.NodeContainerConfig{
		Name: "bs",
		Config: docker.Config{
			Image:      "bsimg",
			Env:        []string{"a=b"},
			Entrypoint: []string{"cmd0"},
			Cmd:        []string{"cmd1"},
		},
		HostConfig: docker.HostConfig{
			RestartPolicy: docker.AlwaysRestart(),
			Privileged:    true,
			Binds:         []string{"/xyz:/abc:ro"},
		},
	}
	err := nodecontainer.AddNewContainer("", &c1)
	c.Assert(err, check.IsNil)
	m := nodeContainerManager{client: s.client}
	err = m.DeployNodeContainer(&c1, "", servicecommon.PoolFilter{Exclude: []string{"p1", "p2"}}, false)
	c.Assert(err, check.IsNil)
	daemon, err := s.client.Extensions().DaemonSets(tsuruNamespace).Get("node-container-bs-all")
	c.Assert(err, check.IsNil)
	c.Assert(daemon.Spec.Template.ObjectMeta.Annotations, check.DeepEquals, map[string]string{
		"scheduler.alpha.kubernetes.io/affinity": "{\"nodeAffinity\":{\"requiredDuringSchedulingIgnoredDuringExecution\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"pool\",\"operator\":\"NotIn\",\"values\":[\"p1\",\"p2\"]}]}]}}}",
	})
	err = m.DeployNodeContainer(&c1, "", servicecommon.PoolFilter{Include: []string{"p1"}}, false)
	c.Assert(err, check.IsNil)
	daemon, err = s.client.Extensions().DaemonSets(tsuruNamespace).Get("node-container-bs-all")
	c.Assert(err, check.IsNil)
	c.Assert(daemon.Spec.Template.ObjectMeta.Annotations, check.DeepEquals, map[string]string{
		"scheduler.alpha.kubernetes.io/affinity": "{\"nodeAffinity\":{\"requiredDuringSchedulingIgnoredDuringExecution\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"pool\",\"operator\":\"In\",\"values\":[\"p1\"]}]}]}}}",
	})
}

func (s *S) TestManagerDeployNodeContainerBSSpecialMount(c *check.C) {
	s.mockfakeNodes(c)
	c1 := nodecontainer.NodeContainerConfig{
		Name: nodecontainer.BsDefaultName,
		Config: docker.Config{
			Image: "img1",
		},
		HostConfig: docker.HostConfig{},
	}
	err := nodecontainer.AddNewContainer("", &c1)
	c.Assert(err, check.IsNil)
	m := nodeContainerManager{client: s.client}
	err = m.DeployNodeContainer(&c1, "", servicecommon.PoolFilter{}, false)
	c.Assert(err, check.IsNil)
	daemons, err := s.client.Extensions().DaemonSets(tsuruNamespace).List(v1.ListOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(daemons.Items, check.HasLen, 1)
	daemon, err := s.client.Extensions().DaemonSets(tsuruNamespace).Get("node-container-big-sibling-all")
	c.Assert(err, check.IsNil)
	c.Assert(daemon.Spec.Template.Spec.Volumes, check.DeepEquals, []v1.Volume{
		{
			Name: "volume-0",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/var/log",
				},
			},
		},
		{
			Name: "volume-1",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/var/lib/docker/containers",
				},
			},
		},
	})
	c.Assert(daemon.Spec.Template.Spec.Containers[0].VolumeMounts, check.DeepEquals, []v1.VolumeMount{
		{Name: "volume-0", MountPath: "/var/log", ReadOnly: true},
		{Name: "volume-1", MountPath: "/var/lib/docker/containers", ReadOnly: true},
	})
}
